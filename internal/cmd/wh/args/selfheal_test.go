package args

import (
	"context"
	"testing"
	"time"

	"github.com/llnl/wormhole-cli/internal/selfheal"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

func TestOpenSelfHealFlags(t *testing.T) {
	a := &CLIArgs{}
	flags := OpenFlags(a)

	minimum := findFlag(t, flags, minRetryBackoffName).(*cli.DurationFlag)
	assert.Equal(t, selfheal.DefaultMinRetryBackoff, minimum.Value)
	assert.Equal(t, &a.SelfHeal.MinRetryBackoff, minimum.Destination)

	maximum := findFlag(t, flags, maxRetryBackoffName).(*cli.DurationFlag)
	assert.Equal(t, selfheal.DefaultMaxRetryBackoff, maximum.Value)
	assert.Equal(t, &a.SelfHeal.MaxRetryBackoff, maximum.Destination)
}

func TestOpenSelfHealFlagSources(t *testing.T) {
	t.Setenv("WORMHOLE_MIN_RETRY_BACKOFF", "250ms")
	source, err := ParseTOML([]byte(`[defaults]
max-retry-backoff = "9s"
`), "test")
	assert.NoError(t, err)

	a := &CLIArgs{}
	cmd := &cli.Command{
		Name:  "test",
		Flags: OpenFlags(a, source),
		Action: func(context.Context, *cli.Command) error {
			return nil
		},
	}
	assert.NoError(t, cmd.Run(context.Background(), []string{"test", "--name", "test"}))
	assert.Equal(t, 250*time.Millisecond, a.SelfHeal.MinRetryBackoff)
	assert.Equal(t, 9*time.Second, a.SelfHeal.MaxRetryBackoff)
}

func TestOpenSelfHealCLIOverridesSources(t *testing.T) {
	t.Setenv("WORMHOLE_MIN_RETRY_BACKOFF", "250ms")
	a := &CLIArgs{}
	cmd := &cli.Command{
		Name:  "test",
		Flags: OpenFlags(a),
		Action: func(context.Context, *cli.Command) error {
			return nil
		},
	}
	assert.NoError(t, cmd.Run(context.Background(), []string{"test", "--name", "test", "--min-retry-backoff", "500ms"}))
	assert.Equal(t, 500*time.Millisecond, a.SelfHeal.MinRetryBackoff)
	assert.Equal(t, selfheal.DefaultMaxRetryBackoff, a.SelfHeal.MaxRetryBackoff)
}
