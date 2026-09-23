package selfheal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigValidation(t *testing.T) {
	tests := map[string]struct {
		config  Config
		wantErr string
	}{
		"minimum at floor": {
			config: Config{MinRetryBackoff: MinimumRetryBackoff, MaxRetryBackoff: time.Second},
		},
		"minimum below floor": {
			config:  Config{MinRetryBackoff: MinimumRetryBackoff - time.Nanosecond, MaxRetryBackoff: time.Second},
			wantErr: "min-retry-backoff must be at least 100ms",
		},
		"maximum is not positive": {
			config:  Config{MinRetryBackoff: MinimumRetryBackoff},
			wantErr: "max-retry-backoff must be positive",
		},
		"maximum below minimum": {
			config:  Config{MinRetryBackoff: time.Second, MaxRetryBackoff: 500 * time.Millisecond},
			wantErr: "min-retry-backoff must not exceed max-retry-backoff",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	assert.Equal(t, Config{
		MinRetryBackoff: DefaultMinRetryBackoff,
		MaxRetryBackoff: DefaultMaxRetryBackoff,
	}, DefaultConfig())
}
