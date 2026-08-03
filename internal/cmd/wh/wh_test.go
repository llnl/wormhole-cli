package wh

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"

	"github.com/llnl/wormhole-cli/internal/cmd/wh/args"
	"github.com/llnl/wormhole-cli/internal/logctx"
	"github.com/llnl/wormhole-cli/internal/routeregistry"
)

func TestTasks_Structure(t *testing.T) {
	a := &args.CLIArgs{}
	app := Tasks(a)

	t.Run("app name", func(t *testing.T) {
		assert.Equal(t, "wh", app.Name)
	})

	t.Run("subcommands", func(t *testing.T) {
		names := make(map[string]bool)
		for _, cmd := range app.Commands {
			names[cmd.Name] = true
		}
		assert.True(t, names["open"])
		assert.True(t, names["community"])
		assert.True(t, names["route"])
	})

	t.Run("open has flags", func(t *testing.T) {
		var openCmd *cli.Command
		for _, cmd := range app.Commands {
			if cmd.Name == "open" {
				openCmd = cmd
				break
			}
		}
		assert.NotNil(t, openCmd)

		expectedFlags := args.OpenFlags(a)
		assert.Len(t, openCmd.Flags, len(expectedFlags), "open subcommand flags should match OpenFlags output")
	})

	t.Run("community subcommands", func(t *testing.T) {
		var communityCmd *cli.Command
		for _, cmd := range app.Commands {
			if cmd.Name == "community" {
				communityCmd = cmd
				break
			}
		}
		assert.NotNil(t, communityCmd)

		names := make(map[string]bool)
		for _, cmd := range communityCmd.Commands {
			names[cmd.Name] = true
		}
		assert.True(t, names["list"])
		assert.True(t, names["add"])
		assert.True(t, names["remove"])
		assert.True(t, names["add-route"])
		assert.True(t, names["remove-route"])
	})

	t.Run("route subcommands", func(t *testing.T) {
		var routeCmd *cli.Command
		for _, cmd := range app.Commands {
			if cmd.Name == "route" {
				routeCmd = cmd
				break
			}
		}
		assert.NotNil(t, routeCmd)

		names := make(map[string]bool)
		for _, cmd := range routeCmd.Commands {
			names[cmd.Name] = true
		}
		assert.True(t, names["list"])
	})
}

// newTestCLIArgs returns a CLIArgs pre-populated with test defaults.
func newTestCLIArgs() *args.CLIArgs {
	return &args.CLIArgs{
		Global: args.GlobalArgs{
			Endpoint: "http://test-endpoint",
			Token:    "test-token",
		},
	}
}

// setupWrap returns a pre-configured CLIArgs and context with logger.
// The logger starts at Warn level; set a.Global.Verbose to true to enable Info level.
func setupWrap(t *testing.T) (context.Context, *args.CLIArgs, *logctx.ContextLogger) {
	t.Helper()
	a := newTestCLIArgs()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	levelVar := &slog.LevelVar{}
	levelVar.Set(slog.LevelWarn)
	cl := logctx.New(logger, levelVar)
	ctx := logctx.WithLogger(context.Background(), cl)

	return ctx, a, cl
}

func TestGlobalWrap_BasicBehavior(t *testing.T) {
	t.Run("reads endpoint and token", func(t *testing.T) {
		ctx, a, _ := setupWrap(t)

		wrapped := globalWrap(a, func(ctx context.Context, cCmd *cli.Command, a *args.CLIArgs, registry routeregistry.RegistryService, logger *slog.Logger) error {
			return nil
		})
		cli.ActionFunc(wrapped)(ctx, &cli.Command{})

		assert.Equal(t, "http://test-endpoint", a.Global.Endpoint)
		assert.Equal(t, "test-token", a.Global.Token)
	})

	t.Run("verbose sets info level", func(t *testing.T) {
		ctx, a, cl := setupWrap(t)
		a.Global.Verbose = true

		wrapped := globalWrap(a, func(ctx context.Context, cCmd *cli.Command, a *args.CLIArgs, registry routeregistry.RegistryService, logger *slog.Logger) error {
			return nil
		})
		cli.ActionFunc(wrapped)(ctx, &cli.Command{})

		level, _ := cl.GetLogLevel()
		assert.Equal(t, slog.LevelInfo, *level)
	})

	t.Run("non-verbose keeps warn", func(t *testing.T) {
		ctx, a, cl := setupWrap(t)
		a.Global.Verbose = false

		wrapped := globalWrap(a, func(ctx context.Context, cCmd *cli.Command, a *args.CLIArgs, registry routeregistry.RegistryService, logger *slog.Logger) error {
			return nil
		})
		cli.ActionFunc(wrapped)(ctx, &cli.Command{})

		level, _ := cl.GetLogLevel()
		assert.Equal(t, slog.LevelWarn, *level)
	})
}

func TestGlobalWrap_RegistryClientCreated(t *testing.T) {
	ctx, a, _ := setupWrap(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a.Global.Endpoint = server.URL

	var registryOK bool
	wrapped := globalWrap(a, func(ctx context.Context, cCmd *cli.Command, a *args.CLIArgs, registry routeregistry.RegistryService, logger *slog.Logger) error {
		registryOK = registry != nil
		return nil
	})
	cli.ActionFunc(wrapped)(ctx, &cli.Command{})
	assert.True(t, registryOK)
}
