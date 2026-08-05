package args

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"

	"github.com/llnl/wormhole-cli/test/testutil"
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
			t.Run(tt.flagName, func(t *testing.T) {
				t.Setenv(tt.envName, "from-env")
				chain := flagSources(tt.flagName)
				v, ok := chain.Lookup()
				assert.True(t, ok)
				assert.Equal(t, "from-env", v)
			})
		}
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
		{"flat keys", testutil.TOMLTable(testutil.DefaultTable, testutil.KV{Key: "endpoint", Value: "http://x"}), testutil.DefaultTable + ".endpoint", "http://x", true},
		{"nested defaults", testutil.TOMLTable(testutil.DefaultTable, testutil.KV{Key: "app-port", Value: 9090}, testutil.KV{Key: "name", Value: "myapp"}), testutil.DefaultTable + ".app-port", int64(9090), true},
		{"empty file", "", testutil.DefaultTable + ".anything", nil, false},
		{"boolean", testutil.TOMLTable(testutil.DefaultTable, testutil.KV{Key: "verbose", Value: true}), testutil.DefaultTable + ".verbose", true, true},
		{"integer", testutil.TOMLTable(testutil.DefaultTable, testutil.KV{Key: "app-port", Value: 9090}), testutil.DefaultTable + ".app-port", int64(9090), true},
		{"string", testutil.TOMLTable(testutil.DefaultTable, testutil.KV{Key: "name", Value: "myapp"}), testutil.DefaultTable + ".name", "myapp", true},
	}
	for _, tt := range tests {
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
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				t.Setenv("WORMHOLE_TEST_FLAG", tt.envVal)
			} else {
				os.Unsetenv("WORMHOLE_TEST_FLAG")
			}

			var srcs []cli.MapSource
			if tt.tomlVal != "" {
				ms, _ := ParseTOML([]byte(testutil.TOMLTable("defaults", testutil.KV{Key: "test-flag", Value: tt.tomlVal})), "test")
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
