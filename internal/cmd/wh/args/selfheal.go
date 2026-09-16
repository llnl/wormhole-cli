package args

import (
	"fmt"
	"time"

	"github.com/BurntSushi/toml"
)

type selfHealFile struct {
	SelfHeal struct {
		MinRetryBackoff string `toml:"min-retry-backoff"`
		MaxRetryBackoff string `toml:"max-retry-backoff"`
	} `toml:"self-heal"`
}

// ParseSelfHeal parses only the root-controlled self-heal table. Callers must
// pass the system configuration contents, never a user configuration file.
func ParseSelfHeal(data []byte, name string) (SelfHealArgs, error) {
	config := DefaultSelfHealArgs()
	var file selfHealFile
	if err := toml.Unmarshal(data, &file); err != nil {
		return SelfHealArgs{}, fmt.Errorf("parsing %s: %w", name, err)
	}

	values := []struct {
		name string
		raw  string
		dst  *time.Duration
	}{
		{"min-retry-backoff", file.SelfHeal.MinRetryBackoff, &config.MinRetryBackoff},
		{"max-retry-backoff", file.SelfHeal.MaxRetryBackoff, &config.MaxRetryBackoff},
	}
	for _, value := range values {
		if value.raw == "" {
			continue
		}
		duration, err := time.ParseDuration(value.raw)
		if err != nil {
			return SelfHealArgs{}, fmt.Errorf("invalid self-heal.%s: %w", value.name, err)
		}
		*value.dst = duration
	}

	if err := config.Validate(); err != nil {
		return SelfHealArgs{}, err
	}
	return config, nil
}
