package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/llnl/wormhole-cli/internal/cmd/wh"
	"github.com/llnl/wormhole-cli/internal/cmd/wh/args"
	"github.com/llnl/wormhole-cli/internal/logctx"
	"github.com/llnl/wormhole-cli/internal/version"
	"github.com/urfave/cli/v3"
)

const systemConfigPath = "/etc/wormhole/cli/wh.toml"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// create logger with appropriate log level and attach to context
	var logLevel slog.LevelVar
	logLevel.Set(slog.LevelWarn)

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: &logLevel,
	}))

	lCtx := logctx.WithLogger(ctx, logctx.New(logger, &logLevel))

	cli.VersionPrinter = version.Printer

	// Pass 1: Bootstrap — parse --config and --nodefaults only.
	// OnUsageError returns nil to silently ignore unrecognized flags/args
	// (e.g. subcommand name, open subcommand flags).
	cliArgs := &args.CLIArgs{}
	bootstrap := &cli.Command{
		HideHelp: true,
		OnUsageError: func(ctx context.Context, c *cli.Command, err error, isSubcommand bool) error {
			return nil
		},
		Action: func(ctx context.Context, c *cli.Command) error { return nil },
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:        "config",
				Usage:       "Path to a wh.toml configuration file (repeatable, layered after system config)",
				Destination: &cliArgs.Configs,
			},
			&cli.BoolFlag{
				Name:        "nodefaults",
				Usage:       "Skip system config at /etc/wormhole/cli/wh.toml",
				Destination: &cliArgs.NoDefaults,
			},
		},
	}
	if err := bootstrap.Run(lCtx, os.Args); err != nil {
		log.Fatalln(err)
	}

	// Between passes: load TOML files into MapSources.
	// Order: user configs first (highest priority), system config last.
	var mapSrcs []cli.MapSource
	if !cliArgs.NoDefaults {
		if ms, err := args.TOMLMapSource(systemConfigPath); err != nil {
			if !os.IsNotExist(err) {
				log.Fatalf("system config %s: %v\n", systemConfigPath, err)
			}
		} else {
			mapSrcs = append(mapSrcs, ms)
		}
	}
	for _, path := range cliArgs.Configs {
		ms, err := args.TOMLMapSource(path)
		if err != nil {
			log.Fatalf("config %s: %v\n", path, err)
		}
		mapSrcs = append(mapSrcs, ms)
	}

	// Pass 2: Full parse with Sources chains for env + TOML resolution.
	app := wh.Tasks(cliArgs, mapSrcs...)
	app.Flags = args.GlobalFlags(cliArgs, mapSrcs...)

	if err := app.Run(lCtx, os.Args); err != nil {
		log.Fatalln(err)
	}
}
