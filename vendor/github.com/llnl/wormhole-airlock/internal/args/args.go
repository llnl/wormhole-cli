package args

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
)

const (
	DefaultAddress           = ":3128"
	DefaultAuthJSON          = "/etc/airlock/authorization.json"
	DefaultHeaderGroups      = "X-Lc-Groups"
	DefaultHeaderUser        = "X-Lc-User"
	DefaultJWTLeeway         = 30 * time.Second
	DefaultLoggingLevel      = "info"
	DefaultLoggingLocation   = "stdout"
	DefaultMaxHeaderBytes    = 1 << 20 // 1 MB
	DefaultReadHeaderTimeout = 0 * time.Second
	DefaultReadTimeout       = 0 * time.Second
	DefaultWriteTimeout      = 0 * time.Second

	envPrefix = "AIRLOCK_"
)

type FlagBuilder struct {
	Flags []cli.Flag
}

func NewBuilder() *FlagBuilder {
	return &FlagBuilder{
		Flags: make([]cli.Flag, 0),
	}
}

func (f *FlagBuilder) Generic(add []cli.Flag) *FlagBuilder {
	f.Flags = append(f.Flags, add...)
	return f
}

// GetValueOrFile returns the value for a flag/arg name from ctx.
// If the value can be read as a file path, it returns the file contents.
// Otherwise it returns the raw value.
func GetValueOrFile(proposedValue string) string {
	v := strings.TrimSpace(proposedValue)
	if v == "" {
		return ""
	}

	if b, err := os.ReadFile(filepath.Clean(v)); err == nil {
		return strings.TrimSpace(string(b))
	}

	return v
}

//

func envWrapper(varName any) cli.ValueSourceChain {
	switch v := varName.(type) {
	case string:
		return cli.EnvVars(envPrefix + v)
	case []string:
		var prefixedStrings []string
		for _, str := range v {
			prefixedStrings = append(prefixedStrings, envPrefix+str)
		}

		return cli.EnvVars(prefixedStrings...)
	default:
		return cli.ValueSourceChain{}
	}
}
