package selfheal

import (
	"errors"
	"fmt"
	"time"
)

const (
	MinimumRetryBackoff    = 100 * time.Millisecond
	DefaultMinRetryBackoff = MinimumRetryBackoff
	DefaultMaxRetryBackoff = 15 * time.Second
)

// Config controls retry behavior during tunnel recovery.
type Config struct {
	MinRetryBackoff time.Duration
	MaxRetryBackoff time.Duration
}

// DefaultConfig returns the default tunnel recovery configuration.
func DefaultConfig() Config {
	return Config{
		MinRetryBackoff: DefaultMinRetryBackoff,
		MaxRetryBackoff: DefaultMaxRetryBackoff,
	}
}

// Validate checks that the retry backoff bounds are valid.
func (c Config) Validate() error {
	if c.MinRetryBackoff < MinimumRetryBackoff {
		return fmt.Errorf("min-retry-backoff must be at least %s", MinimumRetryBackoff)
	}

	if c.MaxRetryBackoff <= 0 {
		return errors.New("max-retry-backoff must be positive")
	}

	if c.MinRetryBackoff > c.MaxRetryBackoff {
		return errors.New("min-retry-backoff must not exceed max-retry-backoff")
	}

	return nil
}
