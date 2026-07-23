package wh

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/llnl/wormhole-cli/internal/logctx"
	"github.com/llnl/wormhole-cli/internal/ns"
	"github.com/llnl/wormhole-cli/internal/routeregistry"
	"github.com/llnl/wormhole-cli/internal/version"
)

type whHandler func(ctx context.Context, cCmd *cli.Command, registry routeregistry.RegistryService, logger *slog.Logger) error

func Tasks() *cli.Command {
	return &cli.Command{
		Name:    "wh",
		Usage:   "CLI app to create and forward connections through LC Wormhole",
		Version: version.GetVersion(),
		Commands: []*cli.Command{
			openCmd(),
			communityCmd(),
			routeCmd(),
		},
		Flags: []cli.Flag{
			setConfig(),
			setEndpoint(),
			setToken(),
			setVerbose(),
		},
	}
}

func openCmd() *cli.Command {
	usage := "Open a new wormhole to proxy from a local app.\nIf a command is specified, launches the provided command in a new network namespace with wormhole forwarding."
	usageText := "wh open [options] [--] [command [options ...]]"
	flags := []cli.Flag{
		setName(),
		setCommunity(),
		setPort(),
		setPodmanCompat(),
		setAllowedGroups(),
		setAllowedUsers(),
		setForbiddenUsers(),
		setForbiddenGroups(),
		setForwardedHeaderUser(),
		setForwardedHeaderGroups(),
		setAuthBearerHeader(),
	}

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
		Action:       globalWrap(handleOpen),
		Flags:        flags,
		StopOnNthArg: &StopOnNthArg,
	}
}

func communityCmd() *cli.Command {
	usage := "Manage wormhole communities"
	usageText := "wh community [command]"
	return &cli.Command{
		Name:      "community",
		Usage:     usage,
		UsageText: usageText,
		Commands: []*cli.Command{
			communityListCmd(),
			communityAddCmd(),
			communityRemoveCmd(),
			communityAddRouteCmd(),
			communityRemoveRouteCmd(),
		},
	}
}

func communityListCmd() *cli.Command {
	usage := "List communities or show details for a specific community"
	usageText := "wh community list [community]"
	return &cli.Command{
		Name:      "list",
		Usage:     usage,
		UsageText: usageText,
		Action:    globalWrap(listCommunities),
	}
}

func communityAddCmd() *cli.Command {
	usage := "Add a new community with the specified name"
	usageText := "wh community add [name]"
	return &cli.Command{
		Name:      "add",
		Usage:     usage,
		UsageText: usageText,
		Action:    globalWrap(addCommunity),
	}
}

func communityRemoveCmd() *cli.Command {
	usage := "Remove a community by its name or ID (min 8 chars for ID)"
	usageText := "wh community remove [name or id]"
	return &cli.Command{
		Name:      "remove",
		Usage:     usage,
		UsageText: usageText,
		Action:    globalWrap(removeCommunity),
	}
}

func communityAddRouteCmd() *cli.Command {
	usage := fmt.Sprintf("Add a route to a community. Route can be identified by fqname (domain/name) or ID (min %d chars)", routeregistry.MinIDLength)
	usageText := "wh community add-route [community name or id] [route fqname or id]"
	return &cli.Command{
		Name:      "add-route",
		Usage:     usage,
		UsageText: usageText,
		Action:    globalWrap(addRouteToCommunity),
	}
}

func communityRemoveRouteCmd() *cli.Command {
	usage := fmt.Sprintf("Remove a route from a community. Route can be identified by fqname (domain/name) or ID (min %d chars)", routeregistry.MinIDLength)
	usageText := "wh community remove-route [community name or id] [route fqname or id]"
	return &cli.Command{
		Name:      "remove-route",
		Usage:     usage,
		UsageText: usageText,
		Action:    globalWrap(removeRouteFromCommunity),
	}
}

func routeCmd() *cli.Command {
	usage := "Manage wormhole routes"
	usageText := "wh route [command]"
	return &cli.Command{
		Name:      "route",
		Usage:     usage,
		UsageText: usageText,
		Commands: []*cli.Command{
			routeListCmd(),
		},
	}
}

func routeListCmd() *cli.Command {
	usage := "List routes or show details for a specific route"
	usageText := "wh route list [route]"
	return &cli.Command{
		Name:      "list",
		Usage:     usage,
		UsageText: usageText,
		Action:    globalWrap(listRoutes),
	}
}

// handle global flags before executing handler
func globalWrap(handler whHandler) cli.ActionFunc {
	return func(ctx context.Context, cCmd *cli.Command) error {
		verbose := cCmd.Bool(setVerboseName)
		cl := logctx.GetLogger(ctx)
		if verbose {
			cl.SetLogLevel(slog.LevelInfo)
		}

		token := cCmd.String(setTokenName)
		endpoint := cCmd.String(setEndpointName)
		logger := logctx.Logger(ctx)
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		registry := routeregistry.NewRegistryClient(token, endpoint, client, logger)

		return handler(ctx, cCmd, registry, logger)
	}
}
