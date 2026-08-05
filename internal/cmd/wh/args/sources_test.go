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

var testTableAll = testutil.TOMLConfig(testutil.DefaultTable,
	testutil.KV{Key: "endpoint", Value: "http://x"},
	testutil.KV{Key: "app-port", Value: 9090},
	testutil.KV{Key: "name", Value: "myapp"},
	testutil.KV{Key: "verbose", Value: true},
)

func TestFlagSources_BasicProperties(t *testing.T) {
	t.Run("env var name conversion", func(t *testing.T) {
		tests := map[string]struct {
			envName string
		}{
			"endpoint":           {envName: "WORMHOLE_ENDPOINT"},
			"app-port":           {envName: "WORMHOLE_APP_PORT"},
			"allowed-users":      {envName: "WORMHOLE_ALLOWED_USERS"},
			"auth-bearer-header": {envName: "WORMHOLE_AUTH_BEARER_HEADER"},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				t.Setenv(tt.envName, "from-env")
				chain := flagSources(name)
				v, ok := chain.Lookup()
				assert.True(t, ok)
				assert.Equal(t, "from-env", v)
			})
		}
	})
}

func TestTOMLMapSource_ValueTypes(t *testing.T) {
	tests := map[string]struct {
		content string
		key     string
		wantVal any
		wantOK  bool
	}{
		"flat keys": {
			content: testTableAll,
			key:     testutil.DefaultTable + ".endpoint",
			wantVal: "http://x",
			wantOK:  true,
		},
		"nested defaults": {
			content: testTableAll,
			key:     testutil.DefaultTable + ".app-port",
			wantVal: int64(9090),
			wantOK:  true,
		},
		"empty file": {
			content: "",
			key:     testutil.DefaultTable + ".anything",
			wantVal: nil,
			wantOK:  false,
		},
		"boolean": {
			content: testTableAll,
			key:     testutil.DefaultTable + ".verbose",
			wantVal: true,
			wantOK:  true,
		},
		"integer": {
			content: testTableAll,
			key:     testutil.DefaultTable + ".app-port",
			wantVal: int64(9090),
			wantOK:  true,
		},
		"string": {
			content: testTableAll,
			key:     testutil.DefaultTable + ".name",
			wantVal: "myapp",
			wantOK:  true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
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
	tests := map[string]struct {
		envVal   string
		tomlVal  string
		expected string
	}{
		"env overrides map source": {
			envVal:   "from-env",
			tomlVal:  "from-toml",
			expected: "from-env",
		},
		"map source wins when no env": {
			envVal:   "",
			tomlVal:  "from-toml",
			expected: "from-toml",
		},
		"env resolves without map source": {
			envVal:   "from-env",
			tomlVal:  "",
			expected: "from-env",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if tt.envVal != "" {
				t.Setenv("WORMHOLE_TEST_FLAG", tt.envVal)
			} else {
				os.Unsetenv("WORMHOLE_TEST_FLAG")
			}

			var srcs []cli.MapSource
			if tt.tomlVal != "" {
				ms, _ := ParseTOML([]byte(testutil.TOMLConfig("defaults", testutil.KV{Key: "test-flag", Value: tt.tomlVal})), "test")
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
