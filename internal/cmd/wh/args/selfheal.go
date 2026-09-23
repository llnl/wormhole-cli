package args

import "github.com/urfave/cli/v3"

const (
	minRetryBackoffName = "min-retry-backoff"
	maxRetryBackoffName = "max-retry-backoff"
)

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
