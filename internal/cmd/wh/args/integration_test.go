package args

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"

	"github.com/llnl/wormhole-cli/test/testhelpers"
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

// Tests value source chain resolution through the full cli.Command.Run()
// pipeline. Complements TestFlagSources_Resolution in sources_test.go
// by covering the same env > TOML logic plus CLI arg override and
// multi-source precedence via the actual flag resolution path.
func TestIntegration_ValueSourceChain(t *testing.T) {
	const (
		tomlEndpoint = "http://toml"
		envURL       = "http://env"
		cliURL       = "http://cli"
		sysURL       = "http://sys"
		userURL      = "http://user"
	)

	tests := []struct {
		name         string
		tomlContent  string
		envVal       string
		cliArgs      []string
		expected     string
		tomlContent2 string // second TOML for multi-source test
	}{
		{"env overrides TOML", testhelpers.TOMLEntry("endpoint", tomlEndpoint), envURL, nil, envURL, ""},
		{"TOML default used", testhelpers.TOMLEntry("endpoint", tomlEndpoint), "", nil, tomlEndpoint, ""},
		{"CLI overrides TOML", testhelpers.TOMLEntry("endpoint", tomlEndpoint), "", []string{"--endpoint", cliURL}, cliURL, ""},
		{"CLI overrides env", testhelpers.TOMLEntry("endpoint", tomlEndpoint), envURL, []string{"--endpoint", cliURL}, cliURL, ""},
		{"env overrides TOML no CLI", testhelpers.TOMLEntry("endpoint", tomlEndpoint), envURL, nil, envURL, ""},
		{"multiple TOML user wins", testhelpers.TOMLEntry("endpoint", userURL), "", nil, userURL, testhelpers.TOMLEntry("endpoint", sysURL)},
		{"second TOML provides missing key", testhelpers.TOMLEntry("app-port", 9090), "", nil, sysURL, testhelpers.TOMLEntry("endpoint", sysURL)},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var srcs []cli.MapSource
			if tt.tomlContent2 != "" {
				ms1, _ := ParseTOML([]byte(tt.tomlContent), "test1")
				ms2, _ := ParseTOML([]byte(tt.tomlContent2), "test2")
				srcs = []cli.MapSource{ms1, ms2}
			} else {
				ms, _ := ParseTOML([]byte(tt.tomlContent), "test")
				srcs = []cli.MapSource{ms}
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
