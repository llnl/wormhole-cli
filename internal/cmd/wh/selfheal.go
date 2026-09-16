package wh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andydunstall/piko/client"
	"github.com/llnl/wormhole-cli/internal/cmd/wh/args"
	"github.com/llnl/wormhole-cli/internal/requester"
	"github.com/llnl/wormhole-cli/internal/routeregistry"
	"go.uber.org/zap"
)

type tunnelCredentials struct {
	URL        string
	JWT        string
	EndpointID string
}

type routeRegisterFunc func(context.Context, string, string) (*routeregistry.RegistrationResponse, error)
type jwtRefreshFunc func(context.Context, string) (string, error)
type listenAndForwardFunc func(context.Context, tunnelCredentials, string, args.SelfHealArgs) (tunnelForwarder, error)

type tunnelForwarder interface {
	Close() error
	Wait() error
}

type retryRuntime struct {
	sleep  func(context.Context, time.Duration) error
	jitter func() float64
}

func defaultRetryRuntime() retryRuntime {
	return retryRuntime{
		sleep: func(ctx context.Context, delay time.Duration) error {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-timer.C:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
		jitter: rand.Float64,
	}
}

func nextRetryDelay(previous, minimum, maximum time.Duration, jitter func() float64) time.Duration {
	delay := minimum
	if previous > 0 {
		delay = previous * 2
		if delay < previous || delay > maximum {
			delay = maximum
		}
	}
	if delay > maximum {
		delay = maximum
	}
	// Jitter is applied before capping, so retries remain randomized at max.
	return time.Duration(float64(delay) * (0.9 + 0.1*jitter()))
}

func registrationRetryable(err error) bool {
	var status requester.HttpResponseError
	if errors.As(err, &status) {
		return status.Code == http.StatusRequestTimeout || status.Code == http.StatusTooManyRequests ||
			status.Code >= http.StatusInternalServerError && status.Code <= 599
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var networkError net.Error
	return errors.As(err, &networkError)
}

func retryRouteRegistry[T any](
	ctx context.Context,
	operation func(context.Context) (T, error),
	config args.SelfHealArgs,
	logger *slog.Logger,
	runtime retryRuntime,
) (T, error) {
	var zero T
	var previous time.Duration
	for {
		result, err := operation(ctx)
		if err == nil {
			return result, nil
		}
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		if !registrationRetryable(err) {
			return zero, err
		}
		delay := nextRetryDelay(previous, config.MinRetryBackoff, config.MaxRetryBackoff, runtime.jitter)
		previous = delay
		logger.Warn("Route Registry request failed; retrying", slog.Duration("delay", delay), slog.Any("error", err))
		if err := runtime.sleep(ctx, delay); err != nil {
			return zero, err
		}
	}
}

func registerRouteWithRetry(
	ctx context.Context,
	register routeRegisterFunc,
	community, name string,
	config args.SelfHealArgs,
	logger *slog.Logger,
	runtime retryRuntime,
) (*routeregistry.RegistrationResponse, error) {
	result, err := retryRouteRegistry(ctx, func(ctx context.Context) (*routeregistry.RegistrationResponse, error) {
		return register(ctx, community, name)
	}, config, logger, runtime)
	if err != nil {
		return nil, fmt.Errorf("register route: %w", err)
	}
	return result, nil
}

func refreshJWTWithRetry(
	ctx context.Context,
	refresh jwtRefreshFunc,
	jwt string,
	config args.SelfHealArgs,
	logger *slog.Logger,
	runtime retryRuntime,
) (string, error) {
	result, err := retryRouteRegistry(ctx, func(ctx context.Context) (string, error) {
		return refresh(ctx, jwt)
	}, config, logger, runtime)
	if err != nil {
		return "", fmt.Errorf("refresh Piko JWT: %w", err)
	}
	return result, nil
}

func credentialsFromRegistration(registration *routeregistry.RegistrationResponse) (tunnelCredentials, error) {
	if registration == nil || registration.Tunnel.URL == nil || registration.Tunnel.JWT == nil || registration.Tunnel.Endpoint == nil {
		return tunnelCredentials{}, errors.New("route registration response is missing tunnel credentials")
	}
	if *registration.Tunnel.URL == "" || *registration.Tunnel.JWT == "" || *registration.Tunnel.Endpoint == "" {
		return tunnelCredentials{}, errors.New("route registration response contains empty tunnel credentials")
	}
	return tunnelCredentials{
		URL:        *registration.Tunnel.URL,
		JWT:        *registration.Tunnel.JWT,
		EndpointID: *registration.Tunnel.Endpoint,
	}, nil
}

// Piko v0.8.1 exposes terminal authentication status only through formatted
// error text. Replace this with typed matching if upstream adds such an API.
// Retryable connection failures remain internal to Upstream.
func pikoAuthenticationFailure(err error) bool { return strings.Contains(err.Error(), "401:") }

func newPikoLogger(verbose bool) (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	if verbose {
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	}
	return config.Build()
}

func pikoListenAndForward(
	ctx context.Context,
	credentials tunnelCredentials,
	targetAddr string,
	config args.SelfHealArgs,
	logger client.Logger,
) (tunnelForwarder, error) {
	endpoint, err := url.Parse(credentials.URL)
	if err != nil {
		return nil, fmt.Errorf("parse Piko URL: %w", err)
	}
	return (&client.Upstream{
		URL:                 endpoint,
		Token:               credentials.JWT,
		MinReconnectBackoff: config.MinRetryBackoff,
		MaxReconnectBackoff: config.MaxRetryBackoff,
		Logger:              logger,
	}).ListenAndForward(ctx, credentials.EndpointID, targetAddr)
}

func runSelfHealingTunnel(
	ctx context.Context,
	initial *routeregistry.RegistrationResponse,
	register func(context.Context) (*routeregistry.RegistrationResponse, error),
	refresh jwtRefreshFunc,
	listen listenAndForwardFunc,
	targetAddr string,
	config args.SelfHealArgs,
	logger *slog.Logger,
	onFirstConnect func(string),
) error {
	registration := initial
	rotated := false
	reRegistered := false
	firstConnection := true
	for {
		credentials, err := credentialsFromRegistration(registration)
		if err != nil {
			return err
		}
		forwarder, err := listen(ctx, credentials, targetAddr, config)
		if err == nil {
			if firstConnection {
				onFirstConnect(registration.URL)
				firstConnection = false
			}
			// A refreshed JWT has proved usable once it establishes a tunnel. A
			// later 401 is a new credential-expiry event, so allow one more
			// refresh and re-registration. Keep both guards set only while their
			// replacement credentials have not established a tunnel.
			rotated = false
			reRegistered = false
			wait := make(chan error, 1)
			go func() { wait <- forwarder.Wait() }()
			select {
			case err = <-wait:
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if err == nil {
					return errors.New("Piko forwarder stopped unexpectedly")
				}
			case <-ctx.Done():
				_ = forwarder.Close()
				return ctx.Err()
			}
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !pikoAuthenticationFailure(err) {
			return fmt.Errorf("connect to Piko: %w", err)
		}
		if rotated {
			return fmt.Errorf("connect to Piko with refreshed JWT: %w", err)
		}
		logger.Warn("Piko credentials rejected; refreshing JWT")
		jwt, refreshErr := refresh(ctx, credentials.JWT)
		if refreshErr == nil {
			registration.Tunnel.JWT = &jwt
			rotated = true
			continue
		}
		var status requester.HttpResponseError
		if !errors.As(refreshErr, &status) || status.Code != http.StatusUnauthorized {
			return refreshErr
		}
		if reRegistered {
			return fmt.Errorf("refresh Piko JWT after route re-registration: %w", refreshErr)
		}
		logger.Warn("Piko JWT refresh was unauthorized; registering route again")
		registration, err = register(ctx)
		if err != nil {
			return err
		}
		reRegistered = true
	}
}
