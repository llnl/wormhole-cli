package bootstrap

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/llnl/wormhole-cli/internal/cmd/wh/args"
	"github.com/urfave/cli/v3"
)

const systemConfigPath = "/etc/wormhole/cli.toml"

// Run performs the two-pass bootstrap:
//  1. Parse --config and --nodefaults (silently ignoring other args).
//  2. Load TOML files into MapSources, returning them for the flag Sources chain.
//
// The logger is owned by main.go and passed in via ctx.
// The cmdArgs parameter should be os.Args (or a test-supplied slice).
func Run(ctx context.Context, cmdArgs []string) ([]cli.MapSource, error) {
	var (
		configs    []string
		noDefaults bool
	)

	bootstrap := &cli.Command{
		HideHelp: true,
		// Silently ignore usage errors (unknown flags, typos, etc.) so that
		// the bootstrap command doesn't interfere with the main app's parsing.
		// The main app will report its own errors on the second pass.
		OnUsageError: func(ctx context.Context, c *cli.Command, err error, isSubcommand bool) error {
			return nil
		},
		Action: func(ctx context.Context, c *cli.Command) error { return nil },
		Flags:  args.BootstrapFlags(&configs, &noDefaults),
	}
	if err := bootstrap.Run(ctx, cmdArgs); err != nil {
		return nil, err
	}

	var mapSrcs []cli.MapSource

	if !noDefaults {
		if ms, err := args.TOMLMapSource(systemConfigPath); err != nil {
			if os.IsNotExist(err) {
				// System config is optional: missing file is not an error.
			} else {
				return nil, fmt.Errorf("system config %s: %w", systemConfigPath, err)
			}
		} else {
			mapSrcs = append(mapSrcs, ms)
		}
	}

	for _, path := range configs {
		ms, err := args.TOMLMapSource(path)
		if err != nil {
			return nil, fmt.Errorf("config %s: %w", path, err)
		}

		mapSrcs = append(mapSrcs, ms)
	}

	// Reverse so user configs take precedence over system config.
	// ValueSourceChain.Lookup() returns the first source with a value,
	// so earlier entries win. After reversal, user configs come first
	// and system config comes last; user configs win when both define
	// the same key.
	slices.Reverse(mapSrcs)

	return mapSrcs, nil
}
