package args

import "github.com/urfave/cli/v3"

const (
	categoryLogging = "Logging"

	loggingDisableName  = "logging-disable"
	loggingLevelName    = "logging-level"
	loggingLocationName = "logging-location"
	loggingAddressName  = "logging-address"
	loggingNetworkName  = "logging-network"
)

type Logging struct {
	Disable  bool
	Level    string
	Location string
	Address  string
	Network  string
}

func (f *FlagBuilder) LoggingFlags(lo *Logging) *FlagBuilder {
	f.Flags = append(f.Flags, []cli.Flag{
		&cli.BoolFlag{
			Category:    categoryLogging,
			Destination: &lo.Disable,
			Name:        loggingDisableName,
			Sources:     envWrapper("LOGGING_DISABLE"),
			Usage:       "Disable all logging except fatal errors",
			Value:       false,
		},
		&cli.StringFlag{
			Category:    categoryLogging,
			Destination: &lo.Level,
			Name:        loggingLevelName,
			Sources:     envWrapper("LOGGING_LEVEL"),
			Usage:       "Set the logging level (e.g., debug, info, warn, error)",
			Value:       "info",
		},
		&cli.StringFlag{
			Category:    categoryLogging,
			Destination: &lo.Location,
			Name:        loggingLocationName,
			Sources:     envWrapper("LOGGING_LOCATION"),
			Usage:       "Location identifies where logs will be saved, this can be a distinct file or syslog",
			Value:       "stdout",
		},
		&cli.StringFlag{
			Category:    categoryLogging,
			Destination: &lo.Network,
			Name:        loggingNetworkName,
			Sources:     envWrapper("LOGGING_NETWORK"),
			Usage:       "Network specified (e.g., tcp) used for remote log daemon connections",
		},
		&cli.StringFlag{
			Category:    categoryLogging,
			Destination: &lo.Address,
			Name:        loggingAddressName,
			Sources:     envWrapper("LOGGING_ADDRESS"),
			Usage:       "Address specified (e.g., localhost:1234) used for remote log daemon connections",
		},
	}...)

	return f
}
