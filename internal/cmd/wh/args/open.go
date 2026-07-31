package args

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"
)

const (
	nameName                  = "name"
	communityName             = "community"
	appPortName               = "app-port"
	podmanCompatName          = "podman-compat"
	allowedUsersName          = "allowed-users"
	allowedGroupsName         = "allowed-groups"
	forbiddenUsersName        = "forbidden-users"
	forbiddenGroupsName       = "forbidden-groups"
	forwardedHeaderUserName   = "forwarded-header-user"
	forwardedHeaderGroupsName = "forwarded-header-groups"
	authBearerHeaderName      = "auth-bearer-header"
)

// OpenArgs holds values for flags defined on the "open" subcommand.
type OpenArgs struct {
	Name                  string
	Community             string
	AppPort               string
	PodmanCompat          bool
	AllowedUsers          string
	AllowedGroups         string
	ForbiddenUsers        string
	ForbiddenGroups       string
	ForwardedHeaderUser   string
	ForwardedHeaderGroups string
	AuthBearerHeader      bool
}

// OpenFlags returns the []cli.Flag for the "open" subcommand (pass 2).
// The srcs parameter provides TOML MapSources for the Sources chain.
func OpenFlags(a *CLIArgs, srcs ...cli.MapSource) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Category:    categoryTunnel,
			Destination: &a.Open.Name,
			Name:        nameName,
			Usage:       "Wormhole name used for URL (e.g. galactic-vis-app)",
			Sources:     flagSources(nameName, srcs...),
			Value:       DefaultName,
			Required:    true,
			Validator: func(n string) error {
				if strings.ContainsAny(n, "_") {
					return fmt.Errorf("name may not contain any underscores _")
				}
				return nil
			},
			Config: cli.StringConfig{TrimSpace: true},
		},
		&cli.StringFlag{
			Category:    categoryTunnel,
			Destination: &a.Open.Community,
			Name:        communityName,
			Usage:       "Name of app community group for shared authentication",
			Sources:     flagSources(communityName, srcs...),
			Value:       DefaultCommunity,
			Config:      cli.StringConfig{TrimSpace: true},
		},
		&cli.StringFlag{
			Category:    categoryTunnel,
			Destination: &a.Open.AppPort,
			Name:        appPortName,
			Usage:       "Port of your application to forward",
			Sources:     flagSources(appPortName, srcs...),
			Value:       DefaultAppPort,
			Config:      cli.StringConfig{TrimSpace: true},
		},
		&cli.BoolFlag{
			Category:    categoryTunnel,
			Destination: &a.Open.PodmanCompat,
			Name:        podmanCompatName,
			Usage:       "Enable podman compatibility mode for running containers (namespaced only)",
			Sources:     flagSources(podmanCompatName, srcs...),
			Value:       DefaultPodmanCompat,
			Hidden:      true,
		},
		&cli.StringFlag{
			Category:    categoryTunnel,
			Destination: &a.Open.AllowedUsers,
			Name:        allowedUsersName,
			Usage:       "Comma separated list of usernames (e.g. alice,bob1,joe2)",
			Sources:     flagSources(allowedUsersName, srcs...),
			Value:       DefaultAllowedUsers,
			Config:      cli.StringConfig{TrimSpace: true},
		},
		&cli.StringFlag{
			Category:    categoryTunnel,
			Destination: &a.Open.AllowedGroups,
			Name:        allowedGroupsName,
			Usage:       "Comma separated list of groups (e.g. a-team,skunkworks,users)",
			Sources:     flagSources(allowedGroupsName, srcs...),
			Value:       DefaultAllowedGroups,
			Config:      cli.StringConfig{TrimSpace: true},
		},
		&cli.StringFlag{
			Category:    categoryTunnel,
			Destination: &a.Open.ForbiddenUsers,
			Name:        forbiddenUsersName,
			Usage:       "Comma separated list of usernames (e.g. steve,joe1)",
			Sources:     flagSources(forbiddenUsersName, srcs...),
			Value:       DefaultForbiddenUsers,
			Config:      cli.StringConfig{TrimSpace: true},
		},
		&cli.StringFlag{
			Category:    categoryTunnel,
			Destination: &a.Open.ForbiddenGroups,
			Name:        forbiddenGroupsName,
			Usage:       "Comma separated list of groups (e.g. b-team,devs)",
			Sources:     flagSources(forbiddenGroupsName, srcs...),
			Value:       DefaultForbiddenGroups,
			Config:      cli.StringConfig{TrimSpace: true},
		},
		&cli.StringFlag{
			Category:    categoryTunnel,
			Destination: &a.Open.ForwardedHeaderUser,
			Name:        forwardedHeaderUserName,
			Usage:       "Header to populate with request username",
			Sources:     flagSources(forwardedHeaderUserName, srcs...),
			Value:       DefaultForwardedHeaderUser,
			Config:      cli.StringConfig{TrimSpace: true},
		},
		&cli.StringFlag{
			Category:    categoryTunnel,
			Destination: &a.Open.ForwardedHeaderGroups,
			Name:        forwardedHeaderGroupsName,
			Usage:       "Header to populate with request user's groups",
			Sources:     flagSources(forwardedHeaderGroupsName, srcs...),
			Value:       DefaultForwardedHeaderGroups,
			Config:      cli.StringConfig{TrimSpace: true},
		},
		&cli.BoolFlag{
			Category:    categoryTunnel,
			Destination: &a.Open.AuthBearerHeader,
			Name:        authBearerHeaderName,
			Usage:       "Indicates that the header 'Authorization: Bearer <x-token value>' should be set",
			Sources:     flagSources(authBearerHeaderName, srcs...),
			Value:       DefaultAuthBearerHeader,
			Hidden:      true,
		},
	}
}
