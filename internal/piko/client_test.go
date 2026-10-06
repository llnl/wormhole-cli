package piko

import (
	"bytes"
	"errors"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewLoggerHonorsVerbose(t *testing.T) {
	quiet, err := NewLogger(false)
	assert.NoError(t, err)

	var output bytes.Buffer
	originalOutput := log.Writer()
	originalFlags := log.Flags()
	originalPrefix := log.Prefix()
	log.SetOutput(&output)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
		log.SetPrefix(originalPrefix)
	})

	quiet.Debug("disconnected; reconnecting", zap.String("error", "hidden"))
	quiet.Info("connect failed; retrying", zap.Int("attempt", 2))
	quiet.Warn("connected", zap.String("address", "hidden"))
	quiet.Debug("unrelated debug")
	quiet.Warn("unrelated warning")
	quiet.Error("unrelated error")
	assert.Equal(t, "disconnected; reconnecting\nconnect failed; retrying\nconnected\n", output.String())

	verbose, err := NewLogger(true)
	assert.NoError(t, err)
	defer func() { _ = verbose.Sync() }()
	zapLogger, ok := verbose.(*zap.Logger)
	assert.True(t, ok)
	assert.NotNil(t, zapLogger.Check(zap.DebugLevel, "debug"))
}

func TestAuthenticationFailure(t *testing.T) {
	assert.True(t, AuthenticationFailure(errors.New("401: unauthorized")))
	assert.False(t, AuthenticationFailure(assert.AnError))
	assert.False(t, AuthenticationFailure(nil))
}
