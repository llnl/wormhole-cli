package selfheal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"time"

	wormholepiko "github.com/llnl/wormhole-cli/internal/piko"
	"github.com/llnl/wormhole-cli/internal/requester"
	"github.com/llnl/wormhole-cli/internal/routeregistry"
)

type routeRegisterFunc func(context.Context, string, string) (*routeregistry.RegistrationResponse, error)
type jwtRefreshFunc func(context.Context, string) (string, error)
type listenAndForwardFunc func(context.Context, wormholepiko.Credentials, string, time.Duration, time.Duration) (wormholepiko.Forwarder, error)

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

func nextRetryDelay(previous, minimum, maximum time.Duration, jitter func() float64) (time.Duration, time.Duration) {
	backoff := minimum
	if previous > 0 {
		if previous >= maximum || previous > maximum-previous {
			backoff = maximum
		} else {
			backoff = previous * 2
		}
	}
	if backoff < minimum {
		backoff = minimum
	}
	if backoff > maximum {
		backoff = maximum
	}

	jitterValue := jitter()
	if jitterValue < 0 {
		jitterValue = 0
	} else if jitterValue > 1 {
		jitterValue = 1
	}
	delay := backoff - time.Duration(float64(backoff)*0.1*(1-jitterValue))
	if delay < minimum {
		delay = minimum
	}
	if delay > maximum {
		delay = maximum
	}
	return backoff, delay
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
	config Config,
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
		backoff, delay := nextRetryDelay(previous, config.MinRetryBackoff, config.MaxRetryBackoff, runtime.jitter)
		previous = backoff
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
	config Config,
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
	config Config,
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

func credentialsFromRegistration(registration *routeregistry.RegistrationResponse) (wormholepiko.Credentials, error) {
	if registration == nil || registration.Tunnel.URL == nil || registration.Tunnel.JWT == nil || registration.Tunnel.Endpoint == nil {
		return wormholepiko.Credentials{}, errors.New("route registration response is missing tunnel credentials")
	}
	if *registration.Tunnel.URL == "" || *registration.Tunnel.JWT == "" || *registration.Tunnel.Endpoint == "" {
		return wormholepiko.Credentials{}, errors.New("route registration response contains empty tunnel credentials")
	}
	return wormholepiko.Credentials{
		URL:        *registration.Tunnel.URL,
		JWT:        *registration.Tunnel.JWT,
		EndpointID: *registration.Tunnel.Endpoint,
	}, nil
}

// RegisterRoute performs route registration and retries transient failures.
func RegisterRoute(
	ctx context.Context,
	register routeRegisterFunc,
	community, name string,
	config Config,
	logger *slog.Logger,
) (*routeregistry.RegistrationResponse, error) {
	return registerRouteWithRetry(ctx, register, community, name, config, logger, defaultRetryRuntime())
}

// Run maintains the tunnel, refreshing or replacing rejected credentials.
func Run(
	ctx context.Context,
	initial *routeregistry.RegistrationResponse,
	register func(context.Context) (*routeregistry.RegistrationResponse, error),
	refresh jwtRefreshFunc,
	listen listenAndForwardFunc,
	targetAddr string,
	config Config,
	logger *slog.Logger,
	onFirstConnect func(string),
) error {
	return run(ctx, initial, register, refresh, listen, targetAddr, config, logger, onFirstConnect, defaultRetryRuntime())
}

func run(
	ctx context.Context,
	initial *routeregistry.RegistrationResponse,
	register func(context.Context) (*routeregistry.RegistrationResponse, error),
	refresh jwtRefreshFunc,
	listen listenAndForwardFunc,
	targetAddr string,
	config Config,
	logger *slog.Logger,
	onFirstConnect func(string),
	runtime retryRuntime,
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
		forwarder, err := listen(ctx, credentials, targetAddr, config.MinRetryBackoff, config.MaxRetryBackoff)
		if err == nil {
			if firstConnection {
				onFirstConnect(registration.URL)
				firstConnection = false
			}
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
		if !wormholepiko.AuthenticationFailure(err) {
			return fmt.Errorf("connect to Piko: %w", err)
		}
		if rotated {
			return fmt.Errorf("connect to Piko with refreshed JWT: %w", err)
		}
		logger.Warn("Piko credentials rejected; refreshing JWT")
		jwt, refreshErr := refreshJWTWithRetry(ctx, refresh, credentials.JWT, config, logger, runtime)
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
