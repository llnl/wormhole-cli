package wh

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/urfave/cli/v3"

	"github.com/llnl/wormhole-cli/internal/routeregistry"
)

func listCommunities(ctx context.Context, cCmd *cli.Command, registry routeregistry.RegistryService, logger *slog.Logger) error {
	identifier := cCmd.Args().First()

	cl, err := registry.ListCommunities(ctx, identifier)
	if err != nil {
		return err
	}

	if identifier != "" {
		if len(cl) == 0 {
			return fmt.Errorf("community not found: %s", identifier)
		}
		c := &cl[0]

		// route allow and disallow list are not populated from community api
		// instead query each route and populate it so we can print the entitysets
		if len(c.Routes) > 0 {
			rl, err := registry.ListRoutes(ctx, "")
			if err != nil {
				return err
			}

			for i, r := range c.Routes {
				if r.ID == nil {
					continue
				}
				nr := findRoute(rl, *r.ID)
				if nr != nil {
					c.Routes[i] = *nr
				}
			}
		}

		fmt.Print(formatCommunitySingle(c))
	} else {
		fmt.Print(formatCommunityList(cl))
	}

	return nil
}

func addCommunity(ctx context.Context, cCmd *cli.Command, registry routeregistry.RegistryService, logger *slog.Logger) error {
	name := cCmd.Args().First()
	if name == "" {
		return fmt.Errorf("community name is required")
	}

	c, err := registry.AddCommunity(ctx, name)
	if err != nil {
		return err
	}

	fmt.Print(formatCommunitySingle(c))
	return nil
}

func removeCommunity(ctx context.Context, cCmd *cli.Command, registry routeregistry.RegistryService, logger *slog.Logger) error {
	identifier := cCmd.Args().First()
	if identifier == "" {
		return fmt.Errorf("community name or id is required")
	}

	if err := registry.RemoveCommunity(ctx, identifier); err != nil {
		return err
	}

	return nil
}

func addRouteToCommunity(ctx context.Context, cCmd *cli.Command, registry routeregistry.RegistryService, logger *slog.Logger) error {
	if cCmd.Args().Len() != 2 {
		return fmt.Errorf("expected community name or id and route fully qualified name or id")
	}

	var (
		communityIdent = cCmd.Args().Get(0)
		routeIdent     = cCmd.Args().Get(1)
	)

	addOp := func(cID, rID string) error {
		if err := registry.AddCommunityRoute(ctx, cID, rID); err != nil {
			return err
		}

		return nil
	}

	return modRouteToCommunity(ctx, registry, logger, communityIdent, routeIdent, addOp)
}

func removeRouteFromCommunity(ctx context.Context, cCmd *cli.Command, registry routeregistry.RegistryService, logger *slog.Logger) error {
	if cCmd.Args().Len() != 2 {
		return fmt.Errorf("expected community name or id and route fully qualified name or id")
	}

	var (
		communityIdent = cCmd.Args().Get(0)
		routeIdent     = cCmd.Args().Get(1)
	)

	remOp := func(cID, rID string) error {
		if err := registry.RemoveCommunityRoute(ctx, cID, rID); err != nil {
			return err
		}

		return nil
	}

	return modRouteToCommunity(ctx, registry, logger, communityIdent, routeIdent, remOp)
}

// modify association between route and community
func modRouteToCommunity(
	ctx context.Context,
	registry routeregistry.RegistryService,
	logger *slog.Logger,
	communityIdent string,
	routeIdent string,
	op func(cID, rID string) error,
) error {
	c, err := registry.ResolveCommunity(ctx, communityIdent)
	if err != nil {
		return err
	}
	if c.ID == nil {
		return fmt.Errorf("community found but has no ID: %s", communityIdent)
	}

	r, err := registry.ResolveRoute(ctx, routeIdent)
	if err != nil {
		return err
	}
	if r.ID == nil {
		return fmt.Errorf("route found but has no ID: %s", routeIdent)
	}

	return op(*c.ID, *r.ID)
}
