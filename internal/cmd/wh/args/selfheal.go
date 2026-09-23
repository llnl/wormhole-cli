package args

import (
	"fmt"
	"time"

	"github.com/urfave/cli/v3"
)

const (
	MinimumRetryBackoff    = 100 * time.Millisecond
	DefaultMinRetryBackoff = MinimumRetryBackoff
	DefaultMaxRetryBackoff = 15 * time.Second

	minRetryBackoffName = "min-retry-backoff"
	maxRetryBackoffName = "max-retry-backoff"
)

// SelfHealArgs controls retry behavior for `wh open`.
type SelfHealArgs struct {
	MinRetryBackoff time.Duration
	MaxRetryBackoff time.Duration
}

func DefaultSelfHealArgs() SelfHealArgs {
	return SelfHealArgs{
		MinRetryBackoff: DefaultMinRetryBackoff,
		MaxRetryBackoff: DefaultMaxRetryBackoff,
	}
}

func (a SelfHealArgs) Validate() error {
	if a.MinRetryBackoff < MinimumRetryBackoff {
		return fmt.Errorf("min-retry-backoff must be at least %s", MinimumRetryBackoff)
	}
	if a.MaxRetryBackoff <= 0 {
		return fmt.Errorf("max-retry-backoff must be positive")
	}
	if a.MinRetryBackoff > a.MaxRetryBackoff {
		return fmt.Errorf("min-retry-backoff must not exceed max-retry-backoff")
	}
	return nil
}

func selfHealFlags(a *CLIArgs, srcs ...cli.MapSource) []cli.Flag {
	return []cli.Flag{
		&cli.DurationFlag{
			Category:    categoryTunnel,
			Destination: &a.SelfHeal.MinRetryBackoff,
			Name:        minRetryBackoffName,
			Usage:       "Minimum delay between tunnel recovery attempts (must be at least 100ms)",
			Sources:     flagSources(minRetryBackoffName, srcs...),
			Value:       DefaultMinRetryBackoff,
		},
		&cli.DurationFlag{
			Category:    categoryTunnel,
			Destination: &a.SelfHeal.MaxRetryBackoff,
			Name:        maxRetryBackoffName,
			Usage:       "Maximum delay between tunnel recovery attempts",
			Sources:     flagSources(maxRetryBackoffName, srcs...),
			Value:       DefaultMaxRetryBackoff,
		},
	}
}
