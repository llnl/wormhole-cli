package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/llnl/wormhole-airlock/internal/args"
	"github.com/llnl/wormhole-airlock/internal/ctls/logs"
	"github.com/llnl/wormhole-airlock/internal/ctls/rules"
	"github.com/llnl/wormhole-airlock/internal/ctls/storage"
	"github.com/llnl/wormhole-airlock/internal/tokens"
)

const (
	defaultPrefix       = "/-/airlock/"
	headerAuthorization = "Authorization"
	// headerToken is the expected source of the JWT token in the request header
	// that will be passed along to the upstream service.
	headerToken = "X-Token"
)

type data struct {
	ll                      logs.Logger
	authJson                []byte
	expandedResponseHeaders map[string]string
	expandedRequestHeader   map[string]string
	internalPathPrefix      string
	proxyArgs               args.Proxy
	stripPrefix             string
	tokenCache              *storage.TypedCache[tokens.ValidationResult]
	tokenValidator          tokens.JWTValidator
	webArgs                 args.WebServer
	vv                      *validator.Validate
}

func Serve(
	ctx context.Context,
	ll logs.Logger,
	proxyArgs args.Proxy,
	tokenValidator tokens.JWTValidator,
	webArgs args.WebServer,
) error {
	d := initializeData(ll, proxyArgs, tokenValidator, webArgs)

	proxy, err := d.initializeProxy()
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc(d.internalPathPrefix+"authz.json", d.authJSONHandler)
	mux.HandleFunc(d.internalPathPrefix+"version", versionHandler)
	mux.HandleFunc(d.internalPathPrefix, generic404Handler)
	mux.Handle("/", proxy)

	srv := d.initializeServer(mux)

	// Channel to receive server errors
	serverErrors := make(chan error, 1)

	go func() {
		ll.Info("starting HTTP server on " + srv.Addr)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		ll.Info("shutdown signal received")
	case err := <-serverErrors:
		ll.Error("server error: " + err.Error())
		return err
	}

	return d.graceful(ctx, srv)
}

func initializeData(
	ll logs.Logger,
	proxyArgs args.Proxy,
	tokenValidator tokens.JWTValidator,
	webArgs args.WebServer,
) *data {
	vv := rules.NewValidator()

	return &data{
		ll:                      ll,
		authJson:                readJSON(ll, args.GetValueOrFile(proxyArgs.AuthFile)),
		expandedResponseHeaders: buildStaticHeaders(proxyArgs.ResponseHeaders, ll, vv),
		expandedRequestHeader:   buildStaticHeaders(proxyArgs.RequestHeader, ll, vv),
		internalPathPrefix:      buildInternalPathPrefix(proxyArgs.PathPrefix),
		proxyArgs:               proxyArgs,
		stripPrefix:             cleanStripPrefix(proxyArgs.StripPrefix),
		tokenCache:              initTokenCache(),
		tokenValidator:          tokenValidator,
		webArgs:                 webArgs,
		vv:                      vv,
	}
}

func initTokenCache() *storage.TypedCache[tokens.ValidationResult] {
	return storage.NewTypedCache[tokens.ValidationResult]()
}

func buildInternalPathPrefix(pathPrefix string) string {
	return strings.TrimRight(pathPrefix, "/") + defaultPrefix
}

func cleanStripPrefix(stripPrefix string) string {
	return strings.TrimRight(stripPrefix, "/")
}

//

func (d *data) initializeProxy() (*httputil.ReverseProxy, error) {
	targetURL, err := url.Parse(d.proxyArgs.Target)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy target URL %q: %w", d.proxyArgs.Target, err)
	}

	d.ll.Info("proxy target: " + targetURL.String())
	d.ll.Info("internal request path prefix: " + d.internalPathPrefix)

	proxy := &httputil.ReverseProxy{
		Rewrite:        d.proxyRewrite(targetURL),
		ModifyResponse: d.proxyResponse(),
	}

	return proxy, nil
}

func (d *data) initializeServer(mux http.Handler) *http.Server {
	return &http.Server{
		Addr: d.webArgs.Address,
		Handler: d.loggingMiddleware(
			recoveryMiddleware(
				d.stripPrefixHandler(
					d.inspectHeadersMiddleware(mux),
				),
			),
		),
		ReadTimeout:       d.webArgs.ReadTimeout,
		WriteTimeout:      d.webArgs.WriteTimeout,
		ReadHeaderTimeout: d.webArgs.ReadHeaderTimeout,
		MaxHeaderBytes:    d.webArgs.MaxHeaderBytes,
	}
}

// graceful ensures that the server attempts to shutdown when the provided context
// indicates. It is the responsibly of the caller to provide context that sends
// the proper notification when SIGTERM/SIGINT are encountered.
func (d *data) graceful(ctx context.Context, srv *http.Server) error {
	<-ctx.Done()

	d.ll.Info("shutdown server...")

	// Use a bounded context to realize a grace period for the shutdown
	// regardless of the parent.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//nolint: contextcheck
	if err := srv.Shutdown(shutdownCtx); err != nil {
		d.ll.Warn("shutdown failure: " + err.Error())
		return err
	}

	d.ll.Info("shutdown completed")

	return nil
}

func (d *data) proxyRewrite(targetURL *url.URL) func(*httputil.ProxyRequest) {
	return func(pr *httputil.ProxyRequest) {
		// Replicates the core behavior you were getting from
		// httputil.NewSingleHostReverseProxy(targetURL)'s Director:
		pr.SetURL(targetURL)

		// Also replicate the usual X-Forwarded-* behavior.
		// (NewSingleHostReverseProxy historically did this via Director.)
		pr.SetXForwarded()

		pr.Out.Header.Set("X-Forwarded-By", "airlock")

		for k, v := range d.proxyArgs.RequestHeader {
			pr.Out.Header.Set(k, v)
		}
	}
}

func (d *data) proxyResponse() func(*http.Response) error {
	return func(r *http.Response) error {
		for k, v := range d.proxyArgs.ResponseHeaders {
			r.Header.Set(k, v)
		}

		return nil
	}
}

// safeHeader retrieves a header (given the specified key) from the request, validating
// it against a specified struct tag. These tags must either be defined in our rules
// packages or supported by the validation library defaults.
func (d *data) safeHeader(req *http.Request, tag, key string) (string, error) {
	tar := req.Header.Get(key)

	if err := d.vv.Var(tar, tag); err != nil {
		return "", err
	}

	return tar, nil
}

// logHeader optionally logs all headers found in the inbound request.
func (d *data) logHeader(req *http.Request, ll logs.Logger) {
	if d.proxyArgs.HeaderDebug {
		for k, v := range req.Header {
			ll.Debug(fmt.Sprintf("request header %s: %s", k, strings.Join(v, ", ")))
		}
	}
}

//

// buildStaticHeaders generates an expanded list of headers based upon the
// proposed values from an administrators configuration or cli. The proposed
// values/headers are validated; however, any error is simply logged.
func buildStaticHeaders(
	headers map[string]string,
	ll logs.Logger,
	vv *validator.Validate,
) map[string]string {
	sh := make(map[string]string)

	for k, v := range headers {
		// Validate header name
		if err := vv.Var(k, rules.TagHeaderName); err != nil {
			ll.Error("invalid header name: " + k)
			continue
		}

		expanded := os.ExpandEnv(v)

		// Validate header value
		if err := vv.Var(expanded, rules.TagHeaderValue); err != nil {
			ll.Error("invalid header value: " + expanded)
			continue
		}

		sh[k] = expanded
	}

	return sh
}

func readJSON(ll logs.Logger, filePathOrContents string) []byte {
	var test map[string]any

	// Check if the input is already valid JSON.
	if err := json.Unmarshal([]byte(filePathOrContents), &test); err == nil {
		ll.Debug("input is valid JSON string, using it directly")
		return []byte(filePathOrContents)
	}

	ll.Debug("input is not valid JSON string, treating as file path")

	b, err := os.ReadFile(filepath.Clean(filePathOrContents))
	if err != nil {
		ll.Error("failed to read json file: " + err.Error())
		return []byte{}
	}

	if err := json.Unmarshal(b, &test); err != nil {
		ll.Error("failed to parse json file: " + err.Error())
		return []byte{}
	}

	ll.Debug("loaded json file from " + filePathOrContents)

	return b
}
