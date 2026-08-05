package args

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"

	"github.com/llnl/wormhole-cli/test/testutil"
)

// buildTestCommand creates a minimal cli.Command with a single flag
// that uses flagSources for value resolution. The resolved value is
// captured in the destination pointer.
func buildTestCommand(t *testing.T, flagName string, dest *string, srcs ...cli.MapSource) *cli.Command {
	t.Helper()
	flag := &cli.StringFlag{
		Name:        flagName,
		Destination: dest,
		Value:       "default-value",
		Sources:     flagSources(flagName, srcs...),
	}
	return &cli.Command{
		Name:   "test",
		Flags:  []cli.Flag{flag},
		Action: func(ctx context.Context, c *cli.Command) error { return nil },
	}
}

var (
	tomlEndpointSys  = testutil.TOMLConfig("defaults", testutil.KV{Key: "endpoint", Value: "http://sys"})
	tomlEndpointUser = testutil.TOMLConfig("defaults", testutil.KV{Key: "endpoint", Value: "http://user"})
	tomlEndpointTest = testutil.TOMLConfig("defaults", testutil.KV{Key: "endpoint", Value: "http://test"})
	tomlTest         = testutil.TOMLConfig("defaults", testutil.KV{Key: "app-port", Value: 9090})
)

// Tests value source chain resolution through the full cli.Command.Run()
// pipeline. Complements TestFlagSources_Resolution in sources_test.go
// by covering the same env > TOML logic plus CLI arg override and
// multi-source precedence via the actual flag resolution path.
func TestIntegration_ValueSourceChain(t *testing.T) {
	const (
		testURL = "http://test"
		envURL  = "http://env"
		cliURL  = "http://cli"
		sysURL  = "http://sys"
		userURL = "http://user"
	)

	tests := map[string]struct {
		tomlContent []string
		envVal      string
		cliArgs     []string
		expected    string
	}{
		"env overrides TOML": {
			tomlContent: []string{tomlEndpointTest},
			envVal:      envURL,
			cliArgs:     nil,
			expected:    envURL,
		},
		"TOML default used": {
			tomlContent: []string{tomlEndpointTest},
			envVal:      "",
			cliArgs:     nil,
			expected:    testURL,
		},
		"CLI overrides TOML": {
			tomlContent: []string{tomlEndpointTest},
			envVal:      "",
			cliArgs:     []string{"--endpoint", cliURL},
			expected:    cliURL,
		},
		"CLI overrides env": {
			tomlContent: []string{tomlEndpointTest},
			envVal:      envURL,
			cliArgs:     []string{"--endpoint", cliURL},
			expected:    cliURL,
		},
		"multiple TOML user wins": {
			tomlContent: []string{tomlEndpointUser, tomlEndpointSys},
			envVal:      "",
			cliArgs:     nil,
			expected:    userURL,
		},
		"second TOML provides missing key": {
			tomlContent: []string{tomlTest, tomlEndpointSys},
			envVal:      "",
			cliArgs:     nil,
			expected:    sysURL,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var srcs []cli.MapSource
			for _, content := range tt.tomlContent {
				ms, _ := ParseTOML([]byte(content), "test")
				srcs = append(srcs, ms)
			}

			if tt.envVal != "" {
				t.Setenv("WORMHOLE_ENDPOINT", tt.envVal)
			} else {
				os.Unsetenv("WORMHOLE_ENDPOINT")
			}

			var value string
			cmd := buildTestCommand(t, "endpoint", &value, srcs...)
			allArgs := append([]string{"test"}, tt.cliArgs...)
			err := cmd.Run(context.Background(), allArgs)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, value)
		})
	}
}

func TestIntegration_RequiredFlagMissing(t *testing.T) {
	cmd := &cli.Command{
		Name: "test",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "endpoint",
				Required: true,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error { return nil },
	}
	err := cmd.Run(context.Background(), []string{"test"})
	assert.Error(t, err)
}

// buildRequiredFlagCommand creates a command with a single required flag
// that uses flagSources for value resolution. Used to test that the
// required-flag enforcement works even when sources are present.
func buildRequiredFlagCommand(t *testing.T, srcs ...cli.MapSource) *cli.Command {
	t.Helper()
	flag := &cli.StringFlag{
		Name:     "endpoint",
		Required: true,
		Sources:  flagSources("endpoint", srcs...),
	}
	return &cli.Command{
		Name:   "test",
		Flags:  []cli.Flag{flag},
		Action: func(ctx context.Context, c *cli.Command) error { return nil },
	}
}

// Tests that a required flag with sources but no value provided through
// any source (env, TOML, CLI) still returns an error. Complements
// TestIntegration_RequiredFlagMissing which tests a required flag with
// no sources at all.
func TestIntegration_RequiredFlagMissing_WithSources(t *testing.T) {
	ms, _ := ParseTOML([]byte(testutil.TOMLConfig("defaults")), "empty")
	cmd := buildRequiredFlagCommand(t, ms)
	err := cmd.Run(context.Background(), []string{"test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint")
}
