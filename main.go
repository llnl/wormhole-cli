package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/llnl/wormhole-cli/internal/cmd/bootstrap"
	"github.com/llnl/wormhole-cli/internal/cmd/wh"
	"github.com/llnl/wormhole-cli/internal/cmd/wh/args"
	"github.com/llnl/wormhole-cli/internal/logctx"
	"github.com/llnl/wormhole-cli/internal/version"
	"github.com/urfave/cli/v3"
)

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

	// Pass 1: parse --config/--nodefaults, load TOML files into MapSources.
	srcs, err := bootstrap.Run(lCtx, os.Args)
	if err != nil {
		log.Fatalln(err)
	}

	// Pass 2: build the CLI command tree with TOML-sourced flags and run.
	cliArgs := &args.CLIArgs{}
	app := wh.Tasks(cliArgs, srcs...)
	app.Flags = args.GlobalFlags(cliArgs, srcs...)

	if err := app.Run(lCtx, os.Args); err != nil {
		log.Fatalln(err)
	}
}
