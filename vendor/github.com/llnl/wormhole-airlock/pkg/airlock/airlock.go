package airlock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/llnl/wormhole-airlock/internal/args"
	"github.com/llnl/wormhole-airlock/internal/ctls/logs"
	"github.com/llnl/wormhole-airlock/internal/ctls/web"
	"github.com/llnl/wormhole-airlock/internal/jwks"
	"github.com/llnl/wormhole-airlock/internal/proxy"
	"github.com/llnl/wormhole-airlock/internal/tokens"
)

func (o Options) StartProxy(ctx context.Context, proxyTarget string) error {
	cmn, err := o.loadCommon(proxyTarget)
	if err != nil {
		return err
	}

	cmn.ll.Info("starting up proxy service wrapper")

	go func() {
		// Background process that updates the local public keys based
		// upon the target JWKS endpoint on a schedule.
		time.Sleep(5 * time.Minute)
		cmn.keys.UpdateKeys()
	}()

	return proxy.Serve(ctx, cmn.ll, cmn.proxyArgs, cmn.valid, cmn.webArgs)
}

//

type common struct {
	keys      jwks.WebKeys
	ll        logs.Logger
	proxyArgs args.Proxy
	webArgs   args.WebServer
	valid     tokens.JWTValidator
}

func (o Options) loadCommon(proxyTarget string) (common, error) {
	opts, err := initializeOpts(o, proxyTarget)
	if err != nil {
		return common{}, fmt.Errorf("configuration error: %w", err)
	}

	logArgs := args.Logging{
		Address:  opts.Logging.Address,
		Disable:  opts.Logging.Disable,
		Level:    opts.Logging.Level,
		Location: opts.Logging.Location,
		Network:  opts.Logging.Network,
	}

	ll, err := logs.Initialize(logArgs)
	if err != nil {
		return common{}, fmt.Errorf("failed to initialize logging: %w", err)
	}

	keys, err := jwks.Initialize(web.DefaultClient(), ll, opts.Proxy.JWKSURL)
	if err != nil {
		return common{}, fmt.Errorf("failed to initialize JWKS: %w", err)
	}

	return common{
		keys:  keys,
		ll:    ll,
		valid: tokens.Initialize(opts.Proxy.JWTLeeway, keys),
		proxyArgs: args.Proxy{
			AuthFile:         opts.Proxy.AuthJSON,
			AuthBearerHeader: opts.Proxy.AuthBearerHeader,
			PathPrefix:       opts.Proxy.AddPrefix,
			Target:           proxyTarget,
			JWKSURL:          opts.Proxy.JWKSURL,
			JWTLeeway:        opts.Proxy.JWTLeeway,
			ResponseHeaders:  opts.Proxy.ResponseHeaders,
			RequestHeader:    opts.Proxy.RequestHeaders,
			StripPrefix:      opts.Proxy.StripPrefix,
			HeaderDebug:      opts.Proxy.HeaderDebug,
			HeaderGroups:     opts.Proxy.HeaderGroups,
			HeaderUser:       opts.Proxy.HeaderUser,
		},
		webArgs: args.WebServer{
			Address:           opts.Web.Address,
			ReadTimeout:       opts.Web.ReadTimeout,
			WriteTimeout:      opts.Web.WriteTimeout,
			ReadHeaderTimeout: opts.Web.ReadHeaderTimeout,
		},
	}, nil
}

// initializeOpts generates the valid internal options structure but combing
// the callers configuration with the required defaults. Configuration rules
// are also enforced at this time.
func initializeOpts(src Options, proxyTarget string) (Options, error) {
	dst := defaults()
	mergeStructs(&dst, &src)

	if src.Proxy.JWKSURL == "" {
		return Options{}, errors.New("missing required Proxy.JWKSURL")
	}

	if proxyTarget == "" {
		return Options{}, errors.New("missing required proxyTarget")
	}

	if dst.Proxy.StripPrefix != "" && dst.Proxy.AddPrefix != "" {
		return Options{}, errors.New("Proxy.StripPrefix conflicts with Proxy.AddPrefix")
	}

	return dst, nil
}
