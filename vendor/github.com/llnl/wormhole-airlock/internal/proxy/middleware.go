package proxy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/llnl/wormhole-airlock/internal/ctls/logs"
	"github.com/llnl/wormhole-airlock/internal/tokens"
)

type contextKey string

const (
	defaultExpiration            = 5 * time.Minute
	loggerKey         contextKey = "logger"
)

func (d *data) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := newResponseWriter(w)

		ll := d.ll.With(
			slog.String("x-request-id", r.Header.Get("X-Request-ID")),
			slog.String("method", r.Method),
			slog.String("url", r.URL.String()),
			slog.String("remote-address", r.RemoteAddr),
		)

		ll.Debug("inbound request")

		ctx := context.WithValue(r.Context(), loggerKey, ll)
		r = r.WithContext(ctx)

		next.ServeHTTP(rw, r)

		ll.Info(
			"request completed",
			ll.IntArg("status", rw.status),
			ll.StringArg("duration", time.Since(start).String()),
		)
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//nolint:contextcheck
		defer func() {
			if err := recover(); err != nil {
				ll := getLogger(r.Context())

				// Capture stack trace
				stack := debug.Stack()

				// Log with full context
				ll.Error(
					"recovered from panic",
					slog.String("error", fmt.Sprint(err)),
					slog.String("stack_trace", string(stack)),
				)

				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (d *data) inspectHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ll := getLogger(r.Context())

		// Optionally handle debug logging.
		d.logHeader(r, ll)

		if strings.HasPrefix(r.URL.Path, d.internalPathPrefix) {
			ll.Debug("internal path detected")

			// We don't require auth for internal paths at this time.
			next.ServeHTTP(w, r)

			return
		}

		claims, err := d.retrieveClaims(ll, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		r.Header.Set(d.proxyArgs.HeaderUser, claims.Subject)
		r.Header.Set(d.proxyArgs.HeaderGroups, strings.Join(claims.Groups, ","))

		if d.proxyArgs.AuthBearerHeader {
			// The x-token header has been validated before getting to this step.
			r.Header.Set(headerAuthorization, "Bearer "+r.Header.Get(headerToken))
		}

		next.ServeHTTP(w, r)
	})
}

func (d *data) stripPrefixHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, d.stripPrefix)
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}

		next.ServeHTTP(w, r)
	})
}

//

// getLogger retrieves the logger from the request context.
func getLogger(ctx context.Context) logs.Logger {
	//nolint:forcetypeassert
	return ctx.Value(loggerKey).(logs.Logger)
}

func (d *data) retrieveClaims(ll logs.Logger, r *http.Request) (tokens.Claims, error) {
	encoded, err := d.safeHeader(r, "jwt", headerToken)
	if err != nil {
		ll.Warn("failed to get token from header: " + err.Error())
		return tokens.Claims{}, err
	}

	key := d.tokenValidator.CacheKey(encoded)

	// Check cache for both successes and failures
	cachedResult, found := d.tokenCache.Get(key)
	if found {
		if !cachedResult.Valid {
			// Cached failure - return immediately without re-validating
			ll.Debug("token validation failed (cached)")
			return tokens.Claims{}, errors.New("authentication failed")
		}

		ll.Debug("jwt cache hit " + cachedResult.Claims.ID)

		return cachedResult.Claims, nil
	}

	// Cache miss - perform full validation
	header, err := d.tokenValidator.CheckHeader(encoded)
	if err != nil {
		// Invalid header - cache failure for 1 minute
		result := tokens.ValidationResult{
			Valid:       false,
			ErrorString: "invalid header",
			Timestamp:   time.Now(),
		}

		d.tokenCache.Set(key, result, 1*time.Minute)

		return tokens.Claims{}, err
	}

	claims, err := d.tokenValidator.Parse(header, encoded)
	if err != nil {
		result := tokens.ValidationResult{
			Valid:       false,
			ErrorString: "invalid signature",
			Timestamp:   time.Now(),
		}

		// Invalid signature - cache failure for 5 minutes
		d.tokenCache.Set(key, result, defaultExpiration)

		ll.Debug("token validation failed (signature)")

		return tokens.Claims{}, err
	}

	// Successful validation - cache for default expiration (5 minutes)
	result := tokens.ValidationResult{
		Valid:     true,
		Claims:    claims,
		Timestamp: time.Now(),
	}

	d.tokenCache.Set(key, result, d.tokenValidator.CacheExp(claims))

	ll.Debug("token validation successful")

	return claims, nil
}
