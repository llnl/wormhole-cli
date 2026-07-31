package args

import (
	"github.com/urfave/cli/v3"
)

const (
	DefaultAllowedGroups         = ""
	DefaultAllowedUsers          = ""
	DefaultAppPort               = "8080"
	DefaultAuthBearerHeader      = false
	DefaultCommunity             = ""
	DefaultEndpoint              = ""
	DefaultForbiddenGroups       = ""
	DefaultForbiddenUsers        = ""
	DefaultForwardedHeaderUser   = "X-Forwarded-User"
	DefaultForwardedHeaderGroups = "X-Forwarded-Groups"
	DefaultName                  = ""
	DefaultPodmanCompat          = false
	DefaultToken                 = ""
)

const (
	envPrefix = "WORMHOLE_"

	categoryGeneral = "General"
	categoryTunnel  = "Tunnel"
)

const (
	endpointName = "endpoint"
	tokenName    = "token"
	verboseName  = "verbose"
)

// CLIArgs is the superset of all CLI configuration. It composes
// GlobalArgs (root-level flags) and OpenArgs (open subcommand flags)
type CLIArgs struct {
	Global GlobalArgs
	Open   OpenArgs
}

// GlobalArgs holds values for flags defined on the root command.
// It does NOT include Config or NoDefaults — those are bootstrap-only
// local variables, consumed early and never exposed as flags to the
// actual application.
type GlobalArgs struct {
	Endpoint string
	Token    string
	Verbose  bool
}

// GlobalFlags returns the []cli.Flag for the root command (pass 2).
// The srcs parameter provides TOML MapSources for the Sources chain.
// --config and --nodefaults are NOT included here; they are handled
// by the bootstrap command in pass 1.
func GlobalFlags(a *CLIArgs, srcs ...cli.MapSource) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Category:    categoryGeneral,
			Destination: &a.Global.Endpoint,
			Name:        endpointName,
			Usage:       "Route registry endpoint URL",
			Required:    true,
			Sources:     flagSources(endpointName, srcs...),
			Value:       DefaultEndpoint,
			Config:      cli.StringConfig{TrimSpace: true},
		},
		&cli.StringFlag{
			Category:    categoryGeneral,
			Destination: &a.Global.Token,
			Name:        tokenName,
			Usage:       "Authentication token",
			Required:    true,
			Sources:     flagSources(tokenName, srcs...),
			Value:       DefaultToken,
			Config:      cli.StringConfig{TrimSpace: true},
		},
		&cli.BoolFlag{
			Category:    categoryGeneral,
			Destination: &a.Global.Verbose,
			Name:        verboseName,
			Usage:       "Enable verbose logging",
			Sources:     flagSources(verboseName, srcs...),
			Value:       false,
		},
	}
}
