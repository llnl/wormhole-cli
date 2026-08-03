package args

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

// This file contains unit tests for flagSources() and TOMLMapSource().
// It tests value resolution (env > TOML) and TOML parsing in isolation.
// For integration coverage of the same value source chain through
// cli.Command.Run(), see integration_test.go.

func TestFlagSources_BasicProperties(t *testing.T) {
	t.Run("env var name conversion", func(t *testing.T) {
		tests := []struct {
			flagName string
			envName  string
		}{
			{"endpoint", "WORMHOLE_ENDPOINT"},
			{"app-port", "WORMHOLE_APP_PORT"},
			{"allowed-users", "WORMHOLE_ALLOWED_USERS"},
			{"auth-bearer-header", "WORMHOLE_AUTH_BEARER_HEADER"},
		}
		for _, tt := range tests {
			tt := tt
			t.Run(tt.flagName, func(t *testing.T) {
				chain := flagSources(tt.flagName)
				envSrc, ok := chain.Chain[0].(cli.EnvValueSource)
				assert.True(t, ok, "first element should be EnvValueSource")
				assert.Equal(t, tt.envName, envSrc.Key())
			})
		}
	})

	t.Run("chain structure", func(t *testing.T) {
		// env only
		assert.Len(t, flagSources("endpoint").Chain, 1)

		// env + 1 TOML
		ms, _ := ParseTOML([]byte(`endpoint = "http://test"`), "test")
		chain := flagSources("endpoint", ms)
		assert.Len(t, chain.Chain, 2)

		// env + 2 TOML
		ms2, _ := ParseTOML([]byte(`y = 2`), "test2")
		chain = flagSources("flag", ms, ms2)
		assert.Len(t, chain.Chain, 3)
	})

	t.Run("map source prefix", func(t *testing.T) {
		ms, _ := ParseTOML([]byte(`endpoint = "http://test"`), "test")
		chain := flagSources("endpoint", ms)

		msSrc, ok := chain.Chain[1].(interface{ String() string })
		assert.True(t, ok)
		assert.Contains(t, msSrc.String(), `"defaults.endpoint"`)
	})

	// Verifies EnvValueSource is always first even with no TOML sources.
	// (The "env only" case in "chain structure" checks Len==1; this
	// confirms the element at index 0 is specifically the env source.)
	t.Run("chain starts with env", func(t *testing.T) {
		chain := flagSources("endpoint")
		_, ok := chain.Chain[0].(cli.EnvValueSource)
		assert.True(t, ok)
	})
}

func TestTOMLMapSource_ValueTypes(t *testing.T) {
	tests := []struct {
		name    string
		content string
		key     string
		wantVal any
		wantOK  bool
	}{
		{"flat keys", "[defaults]\nendpoint = \"http://x\"", "defaults.endpoint", "http://x", true},
		{"nested defaults", "[defaults]\napp-port = 9090\nname = \"myapp\"", "defaults.app-port", int64(9090), true},
		{"empty file", "", "defaults.anything", nil, false},
		{"boolean", "[defaults]\nverbose = true", "defaults.verbose", true, true},
		{"integer", "[defaults]\napp-port = 9090", "defaults.app-port", int64(9090), true},
		{"string", "[defaults]\nname = \"myapp\"", "defaults.name", "myapp", true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ms, err := ParseTOML([]byte(tt.content), "test")
			assert.NoError(t, err)
			v, ok := ms.Lookup(tt.key)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantVal, v)
			}
		})
	}
}

func TestTOMLMapSource_MissingFile(t *testing.T) {
	_, err := TOMLMapSource("/nonexistent/path.toml")
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestTOMLMapSource_PermissionDenied(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "noread.toml")
	os.WriteFile(cfg, []byte(`x = 1`), 0000)

	_, err := TOMLMapSource(cfg)
	assert.Error(t, err)
	assert.False(t, os.IsNotExist(err))
}

func TestTOMLMapSource_InvalidTOML(t *testing.T) {
	_, err := ParseTOML([]byte(`endpoint = {{{invalid`), "bad.toml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing")
}

// Tests value resolution (env > TOML) at the unit level via flagSources()
// and chain.Lookup(). For integration coverage through cli.Command.Run(),
// see TestIntegration_ValueSourceChain in integration_test.go.
func TestFlagSources_Resolution(t *testing.T) {
	tests := []struct {
		name     string
		envVal   string
		tomlVal  string
		expected string
	}{
		{"env overrides map source", "from-env", "from-toml", "from-env"},
		{"map source wins when no env", "", "from-toml", "from-toml"},
		{"env resolves without map source", "from-env", "", "from-env"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				t.Setenv("WORMHOLE_TEST_FLAG", tt.envVal)
			} else {
				os.Unsetenv("WORMHOLE_TEST_FLAG")
			}

			var srcs []cli.MapSource
			if tt.tomlVal != "" {
				ms, _ := ParseTOML([]byte("[defaults]\ntest-flag = \""+tt.tomlVal+"\""), "test")
				srcs = []cli.MapSource{ms}
			}

			chain := flagSources("test-flag", srcs...)
			v, ok := chain.Lookup()
			assert.True(t, ok)
			assert.Equal(t, tt.expected, v)
		})
	}
}

func TestFlagSources_FlagNamesPresent(t *testing.T) {
	a := &CLIArgs{}
	flags := GlobalFlags(a)

	var names []string
	for _, f := range flags {
		names = append(names, f.Names()...)
	}

	expected := []string{"endpoint", "token", "verbose"}
	for _, n := range expected {
		assert.True(t, slices.Contains(names, n), "missing flag: %s", n)
	}
}
