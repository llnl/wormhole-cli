package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/llnl/wormhole-cli/test/testhelpers"
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

func TestRun_EmptySrcs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantOK  bool
		wantLen int
	}{
		{"no args", nil, true, 0},
		{"nodefaults skips system", []string{"--nodefaults"}, true, 0},
		// System config at /etc/wormhole/cli.toml is optional:
		// missing file is not an error (see bootstrap.go:56).
		{"missing system config", nil, true, 0},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
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
	tests := []struct {
		name    string
		content string
		wantVal string
		wantOK  bool
	}{
		{
			"reads endpoint", testhelpers.TOMLEntry("endpoint", "http://user"), "http://user", true,
		},
		{
			"empty file has no keys", ``, "", false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
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

func TestRun_MultipleUserConfigs(t *testing.T) {
	t.Run("count", func(t *testing.T) {
		dir := t.TempDir()
		cfg1 := writeTOML(t, dir, "first.toml", testhelpers.TOMLEntry("endpoint", "http://first"))
		cfg2 := writeTOML(t, dir, "second.toml", testhelpers.TOMLEntry("endpoint", "http://second"))

		srcs, err := Run(context.Background(), bootstrapArgs("--config", cfg1, "--config", cfg2))
		assert.NoError(t, err)
		assert.Len(t, srcs, 2)
	})

	t.Run("second wins", func(t *testing.T) {
		dir := t.TempDir()
		cfg1 := writeTOML(t, dir, "first.toml", testhelpers.TOMLEntry("endpoint", "http://first"))
		cfg2 := writeTOML(t, dir, "second.toml", testhelpers.TOMLEntry("endpoint", "http://second"))

		srcs, err := Run(context.Background(), bootstrapArgs("--config", cfg1, "--config", cfg2))
		assert.NoError(t, err)
		assert.Len(t, srcs, 2)

		// After Reverse, cfg2 (second) is first in the slice, so it wins.
		v, ok := srcs[0].Lookup("defaults.endpoint")
		assert.True(t, ok)
		assert.Equal(t, "http://second", v)
	})
}

func TestRun_UnknownFlag_Ignored(t *testing.T) {
	dir := t.TempDir()
	cfg := writeTOML(t, dir, "user.toml", testhelpers.TOMLEntry("endpoint", "http://user"))

	// --bogus is not a recognized bootstrap flag; OnUsageError returns nil
	// so it's silently ignored and processing continues to file loading.
	srcs, err := Run(context.Background(), bootstrapArgs("--config", cfg, "--bogus"))
	assert.NoError(t, err)
	assert.Len(t, srcs, 1)
}

func TestRun_InvalidTOML_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	cfg := writeTOML(t, dir, "bad.toml", `endpoint = {{{invalid`)

	_, err := Run(context.Background(), bootstrapArgs("--config", cfg))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config")
}

func TestRun_EmptyTOMLFile(t *testing.T) {
	dir := t.TempDir()
	cfg := writeTOML(t, dir, "empty.toml", ``)

	srcs, err := Run(context.Background(), bootstrapArgs("--config", cfg))
	assert.NoError(t, err)
	assert.Len(t, srcs, 1)

	_, ok := srcs[0].Lookup("defaults.endpoint")
	assert.False(t, ok)
}
