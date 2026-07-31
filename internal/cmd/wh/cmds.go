package wh

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/llnl/wormhole-cli/internal/cmd/wh/args"
	"github.com/llnl/wormhole-cli/internal/logctx"
	"github.com/llnl/wormhole-cli/internal/ns"
	"github.com/llnl/wormhole-cli/internal/routeregistry"
	"github.com/llnl/wormhole-cli/internal/version"
)

var StopOnNthArg int = 1

type globalHandler func(ctx context.Context, cCmd *cli.Command, a *args.CLIArgs, registry routeregistry.RegistryService, logger *slog.Logger) error

func Tasks(a *args.CLIArgs, srcs ...cli.MapSource) *cli.Command {
	return &cli.Command{
		Name:    "wh",
		Usage:   "CLI app to create and forward connections through LC Wormhole",
		Version: version.GetVersion(),
		Commands: []*cli.Command{
			openCmd(a, srcs),
			communityCmd(a),
			routeCmd(a),
		},
	}
}

func openCmd(a *args.CLIArgs, srcs []cli.MapSource) *cli.Command {
	usage := "Open a new wormhole to proxy from a local app.\nIf a command is specified, launches the provided command in a new network namespace with wormhole forwarding."
	usageText := "wh open [options] [--] [command [options ...]]"

	_, err := ns.Initialize()
	if err != nil {
		usage = fmt.Sprintf("%s\nNamespacing is not available on this system", usage)
	} else {
		usage = fmt.Sprintf("%s\nNamespacing is available on this system", usage)
	}

	return &cli.Command{
		Name:         "open",
		Usage:        usage,
		UsageText:    usageText,
		Action:       globalWrap(a, handleOpen),
		Flags:        args.OpenFlags(a, srcs...),
		StopOnNthArg: &StopOnNthArg,
	}
}

func communityCmd(a *args.CLIArgs) *cli.Command {
	usage := "Manage wormhole communities"
	usageText := "wh community [command]"
	return &cli.Command{
		Name:      "community",
		Usage:     usage,
		UsageText: usageText,
		Commands: []*cli.Command{
			communityListCmd(a),
			communityAddCmd(a),
			communityRemoveCmd(a),
			communityAddRouteCmd(a),
			communityRemoveRouteCmd(a),
		},
	}
}

func communityListCmd(a *args.CLIArgs) *cli.Command {
	usage := "List communities or show details for a specific community"
	usageText := "wh community list [community]"
	return &cli.Command{
		Name:      "list",
		Usage:     usage,
		UsageText: usageText,
		Action:    globalWrap(a, listCommunities),
	}
}

func communityAddCmd(a *args.CLIArgs) *cli.Command {
	usage := "Add a new community with the specified name"
	usageText := "wh community add [name]"
	return &cli.Command{
		Name:      "add",
		Usage:     usage,
		UsageText: usageText,
		Action:    globalWrap(a, addCommunity),
	}
}

func communityRemoveCmd(a *args.CLIArgs) *cli.Command {
	usage := "Remove a community by its name or ID (min 8 chars for ID)"
	usageText := "wh community remove [name or id]"
	return &cli.Command{
		Name:      "remove",
		Usage:     usage,
		UsageText: usageText,
		Action:    globalWrap(a, removeCommunity),
	}
}

func communityAddRouteCmd(a *args.CLIArgs) *cli.Command {
	usage := fmt.Sprintf("Add a route to a community. Route can be identified by fqname (domain/name) or ID (min %d chars)", routeregistry.MinIDLength)
	usageText := "wh community add-route [community name or id] [route fqname or id]"
	return &cli.Command{
		Name:      "add-route",
		Usage:     usage,
		UsageText: usageText,
		Action:    globalWrap(a, addRouteToCommunity),
	}
}

func communityRemoveRouteCmd(a *args.CLIArgs) *cli.Command {
	usage := fmt.Sprintf("Remove a route from a community. Route can be identified by fqname (domain/name) or ID (min %d chars)", routeregistry.MinIDLength)
	usageText := "wh community remove-route [community name or id] [route fqname or id]"
	return &cli.Command{
		Name:      "remove-route",
		Usage:     usage,
		UsageText: usageText,
		Action:    globalWrap(a, removeRouteFromCommunity),
	}
}

func routeCmd(a *args.CLIArgs) *cli.Command {
	usage := "Manage wormhole routes"
	usageText := "wh route [command]"
	return &cli.Command{
		Name:      "route",
		Usage:     usage,
		UsageText: usageText,
		Commands: []*cli.Command{
			routeListCmd(a),
		},
	}
}

func routeListCmd(a *args.CLIArgs) *cli.Command {
	usage := "List routes or show details for a specific route"
	usageText := "wh route list [route]"
	return &cli.Command{
		Name:      "list",
		Usage:     usage,
		UsageText: usageText,
		Action:    globalWrap(a, listRoutes),
	}
}

// handle global flags before executing handler
func globalWrap(a *args.CLIArgs, handler globalHandler) cli.ActionFunc {
	return func(ctx context.Context, cCmd *cli.Command) error {
		verbose := a.Global.Verbose
		cl := logctx.GetLogger(ctx)
		if verbose {
			cl.SetLogLevel(slog.LevelInfo)
		}

		token := a.Global.Token
		endpoint := a.Global.Endpoint
		logger := logctx.Logger(ctx)
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		registry := routeregistry.NewRegistryClient(token, endpoint, client, logger)

		return handler(ctx, cCmd, a, registry, logger)
	}
}
