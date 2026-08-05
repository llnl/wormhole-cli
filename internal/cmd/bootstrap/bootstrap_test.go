package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"

	"github.com/llnl/wormhole-cli/test/testutil"
)

func writeTOML(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	assert.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

// bootstrapArgs returns ["wh", ...] with optional extra args appended.
func bootstrapArgs(extra ...string) []string {
	result := []string{"wh"}
	return append(result, extra...)
}

var (
	defaultsEndpointUser   = testutil.TOMLConfig("defaults", testutil.KV{Key: "endpoint", Value: "http://user"})
	defaultsEndpointFirst  = testutil.TOMLConfig("defaults", testutil.KV{Key: "endpoint", Value: "http://first"})
	defaultsEndpointSecond = testutil.TOMLConfig("defaults", testutil.KV{Key: "endpoint", Value: "http://second"})
	defaultsEndpointEnv    = testutil.TOMLConfig("defaults", testutil.KV{Key: "endpoint", Value: "http://env"})
)

func TestRun_EmptySrcs(t *testing.T) {
	tests := map[string]struct {
		args    []string
		wantLen int
	}{
		"no args": {
			args:    nil,
			wantLen: 0,
		},
		"nodefaults skips system": {
			args:    []string{"--nodefaults"},
			wantLen: 0,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			args := bootstrapArgs(tt.args...)
			srcs, err := Run(context.Background(), args)
			assert.NoError(t, err)
			assert.Len(t, srcs, tt.wantLen)
		})
	}
}

// Tests bootstrap.Run()'s file-loading behavior: reading TOML files from
// disk and returning MapSources. Distinct from value resolution tests
// (sources_test.go, integration_test.go) — this validates what files are
// loaded and that empty files produce empty MapSources.
func TestRun_SingleUserConfig(t *testing.T) {
	tests := map[string]struct {
		content string
		wantVal string
		wantOK  bool
	}{
		"reads endpoint": {
			content: defaultsEndpointUser,
			wantVal: "http://user",
			wantOK:  true,
		},
		"empty file has no keys": {
			content: "",
			wantVal: "",
			wantOK:  false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			cfg := writeTOML(t, dir, "user.toml", tt.content)

			srcs, err := Run(context.Background(), bootstrapArgs("--config", cfg))
			assert.NoError(t, err)
			assert.Len(t, srcs, 1)

			v, ok := srcs[0].Lookup("defaults.endpoint")
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantVal, v)
			}
		})
	}
}

func TestRun_NodefaultsStillLoadsUserConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := writeTOML(t, dir, "user.toml", defaultsEndpointUser)
	t.Setenv("WORMHOLE_NODEFAULTS", "true")
	t.Setenv("WORMHOLE_CONFIG", cfg)

	srcs, err := Run(context.Background(), bootstrapArgs())
	assert.NoError(t, err)
	assert.Len(t, srcs, 1)

	v, ok := srcs[0].Lookup("defaults.endpoint")
	assert.True(t, ok)
	assert.Equal(t, "http://user", v)
}

func TestRun_MultipleUserConfigs(t *testing.T) {
	t.Run("second wins", func(t *testing.T) {
		dir := t.TempDir()
		cfg1 := writeTOML(t, dir, "first.toml", defaultsEndpointFirst)
		cfg2 := writeTOML(t, dir, "second.toml", defaultsEndpointSecond)

		srcs, err := Run(context.Background(), bootstrapArgs("--config", cfg1, "--config", cfg2))
		assert.NoError(t, err)
		assert.Len(t, srcs, 2)

		chain := cli.NewValueSourceChain(cli.NewMapValueSource("defaults.endpoint", srcs[0]), cli.NewMapValueSource("defaults.endpoint", srcs[1]))
		v, ok := chain.Lookup()
		assert.True(t, ok)
		assert.Equal(t, "http://second", v)
	})
}

func TestRun_UnknownFlag_Ignored(t *testing.T) {
	dir := t.TempDir()
	cfg := writeTOML(t, dir, "user.toml", defaultsEndpointUser)

	// --bogus is not a recognized bootstrap flag; OnUsageError returns nil
	// so it's silently ignored and processing continues to file loading.
	srcs, err := Run(context.Background(), bootstrapArgs("--config", cfg, "--bogus"))
	assert.NoError(t, err)
	assert.Len(t, srcs, 1)
}

func TestRun_ConfigFromEnv(t *testing.T) {
	dir := t.TempDir()
	cfg := writeTOML(t, dir, "env.toml", defaultsEndpointEnv)
	t.Setenv("WORMHOLE_CONFIG", cfg)

	srcs, err := Run(context.Background(), bootstrapArgs())
	assert.NoError(t, err)
	assert.Len(t, srcs, 1)

	v, ok := srcs[0].Lookup("defaults.endpoint")
	assert.True(t, ok)
	assert.Equal(t, "http://env", v)
}

func TestRun_NoDefaultsFromEnv(t *testing.T) {
	t.Setenv("WORMHOLE_NODEFAULTS", "true")

	srcs, err := Run(context.Background(), bootstrapArgs())
	assert.NoError(t, err)
	assert.Len(t, srcs, 0)
}

func TestRun_InvalidTOML_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	cfg := writeTOML(t, dir, "bad.toml", `endpoint = {{{invalid`)

	_, err := Run(context.Background(), bootstrapArgs("--config", cfg))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config")
}
