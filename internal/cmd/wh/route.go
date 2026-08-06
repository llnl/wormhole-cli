package wh

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/urfave/cli/v3"

	"github.com/llnl/wormhole-cli/internal/cmd/wh/args"
	"github.com/llnl/wormhole-cli/internal/routeregistry"
)

func listRoutes(ctx context.Context, cCmd *cli.Command, a *args.CLIArgs, registry routeregistry.RegistryService, logger *slog.Logger) error {
	identifier := cCmd.Args().First()

	rl, err := registry.ListRoutes(ctx, identifier)
	if err != nil {
		return err
	}

	if len(rl) == 1 {
		r := &rl[0]
		fmt.Print(formatRouteSingle(r))
	} else {
		fmt.Print(formatRouteList(rl))
	}

	return nil
}

// Find a route from the list by full name match or partial ID match.
func findRoute(rl []routeregistry.Route, identifier string) *routeregistry.Route {
	for i := range rl {
		if rl[i].Matches(identifier) {
			return &rl[i]
		}
	}

	return nil
}
