package args

import "github.com/urfave/cli/v3"

const (
	configName     = "config"
	noDefaultsName = "nodefaults"
)

// BootstrapFlags returns the hidden root-level flags used during the
// bootstrap pass and retained on the real command tree so the second
// parse accepts the same inputs.
func BootstrapFlags(configs *[]string, noDefaults *bool) []cli.Flag {
	return []cli.Flag{
		&cli.StringSliceFlag{
			Name:        configName,
			Usage:       "Path to a wh.toml configuration file (repeatable, layered after system config)",
			Destination: configs,
			Sources:     cli.EnvVars(envPrefix + "CONFIG"),
		},
		&cli.BoolFlag{
			Name:        noDefaultsName,
			Usage:       "Skip system config at /etc/wormhole/cli.toml",
			Destination: noDefaults,
			Sources:     cli.EnvVars(envPrefix + "NODEFAULTS"),
		},
	}
}
