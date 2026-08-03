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
		assert.NotEmpty(t, openCmd.Flags, "open should have flags")
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

// setupWrap returns a pre-configured CLIArgs, context with logger, and a
// pointer to a captured value slot. The logger starts at Warn level; set
// a.Global.Verbose to true to enable Info level.
func setupWrap(t *testing.T) (context.Context, *args.CLIArgs, *any, *logctx.ContextLogger) {
	t.Helper()
	a := newTestCLIArgs()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	levelVar := &slog.LevelVar{}
	levelVar.Set(slog.LevelWarn)
	cl := logctx.New(logger, levelVar)
	ctx := logctx.WithLogger(context.Background(), cl)

	var captured any
	return ctx, a, &captured, cl
}

func TestGlobalWrap_BasicBehavior(t *testing.T) {
	tests := []struct {
		name     string
		verbose  bool
		validate func(a *args.CLIArgs, cl *logctx.ContextLogger) any
		expected any
	}{
		{
			"reads endpoint", false,
			func(a *args.CLIArgs, cl *logctx.ContextLogger) any { return a.Global.Endpoint },
			"http://test-endpoint",
		},
		{
			"reads token", false,
			func(a *args.CLIArgs, cl *logctx.ContextLogger) any { return a.Global.Token },
			"test-token",
		},
		{
			"verbose sets info level", true,
			func(a *args.CLIArgs, cl *logctx.ContextLogger) any {
				level, _ := cl.GetLogLevel()
				return *level
			},
			slog.LevelInfo,
		},
		{
			"non-verbose keeps warn", false,
			func(a *args.CLIArgs, cl *logctx.ContextLogger) any {
				level, _ := cl.GetLogLevel()
				return *level
			},
			slog.LevelWarn,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx, a, captured, cl := setupWrap(t)
			a.Global.Verbose = tt.verbose

			wrapped := globalWrap(a, func(ctx context.Context, cCmd *cli.Command, a *args.CLIArgs, registry routeregistry.RegistryService, logger *slog.Logger) error {
				*captured = tt.validate(a, cl)
				return nil
			})
			cli.ActionFunc(wrapped)(ctx, &cli.Command{})
			assert.Equal(t, tt.expected, *captured)
		})
	}
}

func TestGlobalWrap_RegistryClientCreated(t *testing.T) {
	ctx, a, captured, _ := setupWrap(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a.Global.Endpoint = server.URL

	wrapped := globalWrap(a, func(ctx context.Context, cCmd *cli.Command, a *args.CLIArgs, registry routeregistry.RegistryService, logger *slog.Logger) error {
		*captured = registry != nil
		return nil
	})
	cli.ActionFunc(wrapped)(ctx, &cli.Command{})
	assert.True(t, (*captured).(bool))
}
