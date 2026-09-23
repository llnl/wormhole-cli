package selfheal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"syscall"
	"testing"
	"time"

	"github.com/llnl/wormhole-cli/internal/cmd/wh/args"
	wormholepiko "github.com/llnl/wormhole-cli/internal/piko"
	"github.com/llnl/wormhole-cli/internal/requester"
	"github.com/llnl/wormhole-cli/internal/routeregistry"
	"github.com/stretchr/testify/assert"
)

type fakeForwarder struct {
	closed  bool
	done    chan struct{}
	waitErr error
}

func (f *fakeForwarder) Close() error {
	f.closed = true
	select {
	case <-f.done:
	default:
		close(f.done)
	}
	return nil
}
func (f *fakeForwarder) Wait() error { <-f.done; return f.waitErr }

func newFakeForwarder() *fakeForwarder { return &fakeForwarder{done: make(chan struct{})} }
func newFailedForwarder(err error) *fakeForwarder {
	forwarder := newFakeForwarder()
	forwarder.waitErr = err
	close(forwarder.done)
	return forwarder
}

func testRegistration(routeURL, jwt, endpoint string) *routeregistry.RegistrationResponse {
	pikoURL := "https://piko.example"
	return &routeregistry.RegistrationResponse{
		URL: routeURL,
		Tunnel: routeregistry.Tunnel{
			URL:      &pikoURL,
			JWT:      &jwt,
			Endpoint: &endpoint,
		},
	}
}
func testConfig() args.SelfHealArgs {
	return args.SelfHealArgs{MinRetryBackoff: time.Second, MaxRetryBackoff: 4 * time.Second}
}
func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
func testRuntime(delays *[]time.Duration) retryRuntime {
	return retryRuntime{
		jitter: func() float64 { return 0 },
		sleep: func(_ context.Context, delay time.Duration) error {
			*delays = append(*delays, delay)
			return nil
		},
	}
}

func TestRouteRegistryRetries(t *testing.T) {
	for name, failures := range map[string][]error{
		"network":        {&url.Error{Op: "Post", URL: "https://registry", Err: syscall.ECONNRESET}, nil},
		"unexpected EOF": {fmt.Errorf("decode response: %w", io.ErrUnexpectedEOF), nil},
		"408":            {requester.HttpResponseError{Code: http.StatusRequestTimeout}, nil},
		"429":            {requester.HttpResponseError{Code: http.StatusTooManyRequests}, nil},
		"500":            {requester.HttpResponseError{Code: http.StatusInternalServerError}, nil},
		"599":            {requester.HttpResponseError{Code: 599}, nil},
	} {
		t.Run(name, func(t *testing.T) {
			var delays []time.Duration
			calls := 0
			_, err := retryRouteRegistry(context.Background(), func(context.Context) (string, error) {
				err := failures[calls]
				calls++
				return "ok", err
			}, testConfig(), testLogger(), testRuntime(&delays))
			assert.NoError(t, err)
			assert.Equal(t, 2, calls)
			assert.Equal(t, []time.Duration{time.Second}, delays)
		})
	}
}

func TestRouteRegistryDoesNotRetryPermanentErrors(t *testing.T) {
	for name, failure := range map[string]error{
		"ordinary 4xx": requester.HttpResponseError{Code: http.StatusUnauthorized},
		"600":          requester.HttpResponseError{Code: 600},
	} {
		t.Run(name, func(t *testing.T) {
			var delays []time.Duration
			calls := 0
			_, err := retryRouteRegistry(context.Background(), func(context.Context) (string, error) {
				calls++
				return "", failure
			}, testConfig(), testLogger(), testRuntime(&delays))
			assert.ErrorIs(t, err, failure)
			assert.Equal(t, 1, calls)
			assert.Empty(t, delays)
		})
	}
}

func TestRouteRegistryRetryStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	_, err := retryRouteRegistry(ctx, func(context.Context) (string, error) {
		calls++
		return "", requester.HttpResponseError{Code: http.StatusServiceUnavailable}
	}, testConfig(), testLogger(), retryRuntime{
		jitter: func() float64 { return 0 },
		sleep: func(ctx context.Context, _ time.Duration) error {
			cancel()
			return ctx.Err()
		},
	})
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, calls)
}

func TestNextRetryDelay(t *testing.T) {
	maxDuration := time.Duration(1<<63 - 1)
	tests := []struct {
		name                   string
		previous, min, max     time.Duration
		jitter                 float64
		wantBackoff, wantDelay time.Duration
	}{
		{name: "minimum", min: time.Second, max: 4 * time.Second, jitter: 0, wantBackoff: time.Second, wantDelay: time.Second},
		{name: "doubled", previous: time.Second, min: time.Second, max: 4 * time.Second, jitter: 0, wantBackoff: 2 * time.Second, wantDelay: 1800 * time.Millisecond},
		{name: "maximum", previous: 2 * time.Second, min: time.Second, max: 4 * time.Second, jitter: 1, wantBackoff: 4 * time.Second, wantDelay: 4 * time.Second},
		{name: "overflow", previous: maxDuration/2 + 1, min: time.Second, max: maxDuration, jitter: 1, wantBackoff: maxDuration, wantDelay: maxDuration},
		{name: "close bounds", previous: time.Second, min: time.Second, max: 1050 * time.Millisecond, jitter: 0, wantBackoff: 1050 * time.Millisecond, wantDelay: time.Second},
		{name: "jitter below lower boundary", previous: time.Second, min: time.Second, max: 4 * time.Second, jitter: -1, wantBackoff: 2 * time.Second, wantDelay: 1800 * time.Millisecond},
		{name: "jitter above upper boundary", previous: time.Second, min: time.Second, max: 4 * time.Second, jitter: 2, wantBackoff: 2 * time.Second, wantDelay: 2 * time.Second},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			backoff, delay := nextRetryDelay(test.previous, test.min, test.max, func() float64 { return test.jitter })
			assert.Equal(t, test.wantBackoff, backoff)
			assert.Equal(t, test.wantDelay, delay)
		})
	}
}

func TestRetryBackoffDoesNotCompoundJitter(t *testing.T) {
	var delays []time.Duration
	attempts := 0
	_, err := retryRouteRegistry(context.Background(), func(context.Context) (string, error) {
		attempts++
		if attempts <= 3 {
			return "", requester.HttpResponseError{Code: http.StatusServiceUnavailable}
		}
		return "ok", nil
	}, testConfig(), testLogger(), testRuntime(&delays))
	assert.NoError(t, err)
	assert.Equal(t, []time.Duration{time.Second, 1800 * time.Millisecond, 3600 * time.Millisecond}, delays)
}

func TestPikoAuthenticationRefreshesJWT(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	forwarder := newFakeForwarder()
	var got []string
	registrationCalls := 0
	refreshedJWT := ""
	listen := func(ctx context.Context, credentials wormholepiko.Credentials, _ string, _ args.SelfHealArgs) (wormholepiko.Forwarder, error) {
		got = append(got, credentials.JWT)
		if len(got) == 1 {
			return nil, errors.New("401: unauthorized")
		}
		cancel()
		return forwarder, nil
	}
	err := run(ctx, testRegistration("https://route", "old", "one"), func(context.Context) (*routeregistry.RegistrationResponse, error) {
		registrationCalls++
		return nil, errors.New("unexpected registration")
	}, func(_ context.Context, jwt string) (string, error) {
		refreshedJWT = jwt
		return "new", nil
	}, listen, "target", testConfig(), testLogger(), func(string) {}, defaultRetryRuntime())
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []string{"old", "new"}, got)
	assert.Equal(t, "old", refreshedJWT)
	assert.Zero(t, registrationCalls)
	assert.True(t, forwarder.closed)
}

func TestRefreshUnauthorizedRegistersOnce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registrations := 0
	attempts := 0
	listen := func(ctx context.Context, credentials wormholepiko.Credentials, _ string, _ args.SelfHealArgs) (wormholepiko.Forwarder, error) {
		attempts++
		if attempts == 1 {
			return nil, errors.New("401: unauthorized")
		}
		assert.Equal(t, "registered", credentials.JWT)
		cancel()
		return newFakeForwarder(), nil
	}
	err := run(ctx, testRegistration("https://route", "old", "one"), func(context.Context) (*routeregistry.RegistrationResponse, error) {
		registrations++
		return testRegistration("https://route", "registered", "two"), nil
	}, func(context.Context, string) (string, error) {
		return "", requester.HttpResponseError{Code: http.StatusUnauthorized}
	}, listen, "target", testConfig(), testLogger(), func(string) {}, defaultRetryRuntime())
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, registrations)
}

func TestRepeatedRefreshUnauthorizedAfterReregistrationStops(t *testing.T) {
	registrations, refreshes := 0, 0
	err := run(context.Background(), testRegistration("https://route", "old", "one"), func(context.Context) (*routeregistry.RegistrationResponse, error) {
		registrations++
		return testRegistration("https://route", "registered", "two"), nil
	}, func(context.Context, string) (string, error) {
		refreshes++
		return "", requester.HttpResponseError{Code: http.StatusUnauthorized}
	}, func(context.Context, wormholepiko.Credentials, string, args.SelfHealArgs) (wormholepiko.Forwarder, error) {
		return nil, errors.New("401: unauthorized")
	}, "target", testConfig(), testLogger(), func(string) {}, defaultRetryRuntime())

	assert.ErrorIs(t, err, requester.HttpResponseError{Code: http.StatusUnauthorized})
	assert.Equal(t, 1, registrations)
	assert.Equal(t, 2, refreshes)
}

func TestFreshJWTAuthenticationFailureDoesNotLoop(t *testing.T) {
	attempts, refreshes := 0, 0
	err := run(context.Background(), testRegistration("https://route", "old", "one"), func(context.Context) (*routeregistry.RegistrationResponse, error) {
		return nil, errors.New("unexpected")
	}, func(context.Context, string) (string, error) { refreshes++; return "new", nil }, func(context.Context, wormholepiko.Credentials, string, args.SelfHealArgs) (wormholepiko.Forwarder, error) {
		attempts++
		return nil, errors.New("401: unauthorized")
	}, "target", testConfig(), testLogger(), func(string) {}, defaultRetryRuntime())
	assert.Error(t, err)
	assert.Equal(t, 2, attempts)
	assert.Equal(t, 1, refreshes)
}

func TestConnectedRefreshedJWTCanRefreshAgainAfterLaterAuthenticationFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var got []string
	refreshes := 0
	listen := func(ctx context.Context, credentials wormholepiko.Credentials, _ string, _ args.SelfHealArgs) (wormholepiko.Forwarder, error) {
		got = append(got, credentials.JWT)
		switch credentials.JWT {
		case "old":
			return nil, errors.New("401: unauthorized")
		case "first-refresh":
			return newFailedForwarder(errors.New("connect: 401: unauthorized")), nil
		case "second-refresh":
			cancel()
			return newFakeForwarder(), nil
		default:
			return nil, errors.New("unexpected JWT")
		}
	}

	err := run(ctx, testRegistration("https://route", "old", "one"), func(context.Context) (*routeregistry.RegistrationResponse, error) {
		return nil, errors.New("unexpected registration")
	}, func(_ context.Context, jwt string) (string, error) {
		refreshes++
		if jwt == "old" {
			return "first-refresh", nil
		}
		return "second-refresh", nil
	}, listen, "target", testConfig(), testLogger(), func(string) {}, defaultRetryRuntime())

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []string{"old", "first-refresh", "second-refresh"}, got)
	assert.Equal(t, 2, refreshes)
}

func TestCredentialsFromRegistrationRejectsIncompleteTunnel(t *testing.T) {
	valid := testRegistration("https://route", "jwt", "endpoint")
	pikoURL := "https://piko.example"
	jwt := "jwt"
	endpoint := "endpoint"
	empty := ""
	for name, registration := range map[string]*routeregistry.RegistrationResponse{
		"nil response":     nil,
		"missing URL":      {Tunnel: routeregistry.Tunnel{JWT: &jwt, Endpoint: &endpoint}},
		"missing JWT":      {Tunnel: routeregistry.Tunnel{URL: &pikoURL, Endpoint: &endpoint}},
		"missing endpoint": {Tunnel: routeregistry.Tunnel{URL: &pikoURL, JWT: &jwt}},
		"empty URL":        {Tunnel: routeregistry.Tunnel{URL: &empty, JWT: &jwt, Endpoint: &endpoint}},
		"empty JWT":        {Tunnel: routeregistry.Tunnel{URL: &pikoURL, JWT: &empty, Endpoint: &endpoint}},
		"empty endpoint":   {Tunnel: routeregistry.Tunnel{URL: &pikoURL, JWT: &jwt, Endpoint: &empty}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := credentialsFromRegistration(registration)
			assert.Error(t, err)
		})
	}

	got, err := credentialsFromRegistration(valid)
	assert.NoError(t, err)
	assert.Equal(t, wormholepiko.Credentials{URL: "https://piko.example", JWT: "jwt", EndpointID: "endpoint"}, got)
}
