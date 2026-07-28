package wh

import (
	"errors"
	"strings"

	"github.com/urfave/cli/v3"
)

const (
	envPrefix = "WORMHOLE_"

	setNameName                  = "name"
	setConfigName                = "config"
	setNodefaultsName            = "nodefaults"
	setCommunityName             = "community"
	setEndpointName              = "endpoint"
	setTokenName                 = "token"
	setPortName                  = "app-port"
	setPodmanCompatName          = "podman-compat"
	setAllowedUsersName          = "allowed-users"
	setAllowedGroupsName         = "allowed-groups"
	setForbiddenUsersName        = "forbidden-users"
	setForbiddenGroupsName       = "forbidden-groups"
	setForwardedHeaderUserName   = "forwarded-header-user"
	setForwardedHeaderGroupsName = "forwarded-header-groups"
	setVerboseName               = "verbose"
	setAuthBearerHeaderName      = "auth-bearer-header"
)

var StopOnNthArg int = 1

func setName() cli.Flag {
	return &cli.StringFlag{
		Name:     setNameName,
		Sources:  confWrapper("NAME"),
		Usage:    "Wormhole name used for URL (e.g. galactic-vis-app)",
		Config:   cli.StringConfig{TrimSpace: true},
		Required: true,
		Validator: func(n string) error {
			if strings.ContainsAny(n, "_") {
				return errors.New("name may not contain any underscores _")
			}
			return nil
		},
	}
}

func setConfig() cli.Flag {
	return &cli.StringSliceFlag{
		// Config source not valid for this option
		Name:        setConfigName,
		Sources:     cli.EnvVars(envPrefix + "CONFIG"),
		Usage:       "Path to a wh.toml configuration file (repeatable, layered after system config)",
		Config:      cli.StringConfig{TrimSpace: true},
		Required:    false,
		Destination: &confFiles,
	}
}

func setNoDefaults() cli.Flag {
	return &cli.BoolFlag{
		// Config source not valid for this option
		Name:        setNodefaultsName,
		Sources:     cli.EnvVars(envPrefix + "NODEFAULTS"),
		Usage:       "Skip system config at " + systemConfigPath,
		Destination: &nodefaults,
	}
}

func setCommunity() cli.Flag {
	return &cli.StringFlag{
		Name:     setCommunityName,
		Sources:  confWrapper("COMMUNITY"),
		Usage:    "Name of app community group for shared authentication",
		Config:   cli.StringConfig{TrimSpace: true},
		Required: false,
	}
}

func setEndpoint() cli.Flag {
	return &cli.StringFlag{
		Name:     setEndpointName,
		Sources:  confWrapper("ENDPOINT"),
		Usage:    "Use a specified route registry endpoint",
		Config:   cli.StringConfig{TrimSpace: true},
		Required: true,
	}
}

func setToken() cli.Flag {
	return &cli.StringFlag{
		Name:     setTokenName,
		Sources:  confWrapper("TOKEN"),
		Usage:    "Required authentication token",
		Config:   cli.StringConfig{TrimSpace: true},
		Required: true,
	}
}

func setPort() cli.Flag {
	return &cli.StringFlag{
		Name:    setPortName,
		Value:   "8080",
		Sources: confWrapper("APP_PORT"),
		Usage:   "Port of your application to forward",
		Config:  cli.StringConfig{TrimSpace: true},
	}
}

func setPodmanCompat() cli.Flag {
	return &cli.BoolFlag{
		Name:    setPodmanCompatName,
		Sources: confWrapper("PODMAN_COMPAT"),
		Usage:   "Enable podman compatibility mode for running containers (namespaced only)",
		Config:  cli.BoolConfig{},
		Hidden:  true,
	}
}

func setAllowedUsers() cli.Flag {
	return &cli.StringFlag{
		Name:    setAllowedUsersName,
		Sources: confWrapper("ALLOWED_USERS"),
		Usage:   "Comma separated list of usernames (e.g. alice,bob1,joe2)",
		Config:  cli.StringConfig{TrimSpace: true},
	}
}

func setAllowedGroups() cli.Flag {
	return &cli.StringFlag{
		Name:    setAllowedGroupsName,
		Sources: confWrapper("ALLOWED_GROUPS"),
		Usage:   "Comma separated list of groups (e.g. a-team,skunkworks,users)",
		Config:  cli.StringConfig{TrimSpace: true},
	}
}

func setForbiddenUsers() cli.Flag {
	return &cli.StringFlag{
		Name:    setForbiddenUsersName,
		Sources: confWrapper("FORBIDDEN_USERS"),
		Usage:   "Comma separated list of usernames (e.g. steve,joe1)",
		Config:  cli.StringConfig{TrimSpace: true},
	}
}

func setForbiddenGroups() cli.Flag {
	return &cli.StringFlag{
		Name:    setForbiddenGroupsName,
		Sources: confWrapper("FORBIDDEN_GROUPS"),
		Usage:   "Comma separated list of groups (e.g. b-team,devs)",
		Config:  cli.StringConfig{TrimSpace: true},
	}
}

func setForwardedHeaderUser() cli.Flag {
	return &cli.StringFlag{
		Name:    setForwardedHeaderUserName,
		Value:   "X-Forwarded-User",
		Sources: confWrapper("FORWARDED_HEADER_USER"),
		Usage:   "Header to populate with request username",
		Config:  cli.StringConfig{TrimSpace: true},
	}
}

func setForwardedHeaderGroups() cli.Flag {
	return &cli.StringFlag{
		Name:    setForwardedHeaderGroupsName,
		Value:   "X-Forwarded-Groups",
		Sources: confWrapper("FORWARDED_HEADER_GROUPS"),
		Usage:   "Header to populate with request user's groups",
		Config:  cli.StringConfig{TrimSpace: true},
	}
}

func setVerbose() cli.Flag {
	return &cli.BoolFlag{
		Name:    setVerboseName,
		Sources: confWrapper("VERBOSE"),
		Usage:   "Enable verbose logging",
	}
}

func setAuthBearerHeader() cli.Flag {
	return &cli.BoolFlag{
		Name:    setAuthBearerHeaderName,
		Sources: confWrapper("AUTH_BEARER_HEADER"),
		Usage:   "Indicates that the header 'Authorization: Bearer <x-token value>' should be set",
		Config:  cli.BoolConfig{},
		Hidden:  true,
	}
}
