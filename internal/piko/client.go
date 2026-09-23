package piko

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/andydunstall/piko/client"
	"go.uber.org/zap"
)

// Credentials identifies a Piko upstream endpoint.
type Credentials struct {
	URL        string
	JWT        string
	EndpointID string
}

// Forwarder is an active Piko tunnel.
type Forwarder interface {
	Close() error
	Wait() error
}

type quietLogger struct{}

func (quietLogger) log(msg string) {
	switch msg {
	case "disconnected; reconnecting", "connect failed; retrying", "connected":
		log.Printf("%s", msg)
	}
}

func (l quietLogger) Debug(msg string, _ ...zap.Field) { l.log(msg) }
func (l quietLogger) Info(msg string, _ ...zap.Field)  { l.log(msg) }
func (l quietLogger) Warn(msg string, _ ...zap.Field)  { l.log(msg) }
func (l quietLogger) Error(msg string, _ ...zap.Field) { l.log(msg) }
func (quietLogger) Sync() error                        { return nil }

// NewLogger returns Piko's verbose logger or a quiet connection-status logger.
func NewLogger(verbose bool) (client.Logger, error) {
	if !verbose {
		return quietLogger{}, nil
	}
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	return config.Build()
}

// ListenAndForward makes one Piko connection attempt. Reconnection internal to
// Piko is controlled by the supplied backoff settings.
func ListenAndForward(
	ctx context.Context,
	credentials Credentials,
	targetAddr string,
	minReconnectBackoff time.Duration,
	maxReconnectBackoff time.Duration,
	logger client.Logger,
) (Forwarder, error) {
	endpoint, err := url.Parse(credentials.URL)
	if err != nil {
		return nil, fmt.Errorf("parse Piko URL: %w", err)
	}
	return (&client.Upstream{
		URL:                 endpoint,
		Token:               credentials.JWT,
		MinReconnectBackoff: minReconnectBackoff,
		MaxReconnectBackoff: maxReconnectBackoff,
		Logger:              logger,
	}).ListenAndForward(ctx, credentials.EndpointID, targetAddr)
}

// AuthenticationFailure reports whether Piko rejected tunnel credentials.
// Piko v0.8.1 exposes this status only through formatted error text.
func AuthenticationFailure(err error) bool {
	return err != nil && strings.Contains(err.Error(), "401:")
}
