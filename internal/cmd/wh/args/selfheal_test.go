package args

import (
	"context"
	"testing"
	"time"

	"github.com/llnl/wormhole-cli/internal/selfheal"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

func TestSelfHealFlags(t *testing.T) {
	a := &CLIArgs{}
	flags := selfHealFlags(a)
	assert.Len(t, flags, 2)

	minimum := findFlag(t, flags, minRetryBackoffName).(*cli.DurationFlag)
	assert.Equal(t, selfheal.DefaultMinRetryBackoff, minimum.Value)
	assert.Equal(t, &a.SelfHeal.MinRetryBackoff, minimum.Destination)

	maximum := findFlag(t, flags, maxRetryBackoffName).(*cli.DurationFlag)
	assert.Equal(t, selfheal.DefaultMaxRetryBackoff, maximum.Value)
	assert.Equal(t, &a.SelfHeal.MaxRetryBackoff, maximum.Destination)
}

func TestSelfHealFlagSources(t *testing.T) {
	t.Setenv("WORMHOLE_MIN_RETRY_BACKOFF", "250ms")
	source, err := ParseTOML([]byte(`[defaults]
max-retry-backoff = "9s"
`), "test")
	assert.NoError(t, err)

	a := &CLIArgs{}
	cmd := &cli.Command{
		Name:  "test",
		Flags: selfHealFlags(a, source),
		Action: func(context.Context, *cli.Command) error {
			return nil
		},
	}
	assert.NoError(t, cmd.Run(context.Background(), []string{"test"}))
	assert.Equal(t, 250*time.Millisecond, a.SelfHeal.MinRetryBackoff)
	assert.Equal(t, 9*time.Second, a.SelfHeal.MaxRetryBackoff)
}

func TestSelfHealCLIOverridesSources(t *testing.T) {
	t.Setenv("WORMHOLE_MIN_RETRY_BACKOFF", "250ms")
	a := &CLIArgs{}
	cmd := &cli.Command{
		Name:  "test",
		Flags: selfHealFlags(a),
		Action: func(context.Context, *cli.Command) error {
			return nil
		},
	}
	assert.NoError(t, cmd.Run(context.Background(), []string{"test", "--min-retry-backoff", "500ms"}))
	assert.Equal(t, 500*time.Millisecond, a.SelfHeal.MinRetryBackoff)
	assert.Equal(t, selfheal.DefaultMaxRetryBackoff, a.SelfHeal.MaxRetryBackoff)
}
