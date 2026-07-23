package args

import (
	"time"

	"github.com/urfave/cli/v3"
)

const (
	categoryServer = "Web Server"

	serverAddressName           = "address"
	serverMaxHeaderName         = "max-header-bytes"
	serverReadTimeoutName       = "read-timeout"
	serverWriteTimeoutName      = "write-timeout"
	serverReadHeaderTimeoutName = "read-header-timeout"
)

type WebServer struct {
	Address           string
	MaxHeaderBytes    int
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	ReadHeaderTimeout time.Duration
}

func (f *FlagBuilder) WebFlags(ws *WebServer) *FlagBuilder {
	f.Flags = append(f.Flags, []cli.Flag{
		&cli.StringFlag{
			Category:    categoryServer,
			Destination: &ws.Address,
			Name:        serverAddressName,
			Sources:     envWrapper("ADDRESS"),
			Usage:       "The address (host:port) or socket the server should listen on",
			Value:       DefaultAddress,
		},
		&cli.IntFlag{
			Category:    categoryServer,
			Destination: &ws.MaxHeaderBytes,
			Name:        serverMaxHeaderName,
			Sources:     envWrapper("MAX_HEADER_BYTES"),
			Usage:       "Controls the maximum number of bytes the server will read parsing the request headers",
			Value:       DefaultMaxHeaderBytes,
		},
		&cli.DurationFlag{
			Category:    categoryServer,
			Destination: &ws.ReadTimeout,
			Name:        serverReadTimeoutName,
			Sources:     envWrapper("READ_TIMEOUT"),
			Usage:       "Maximum duration for reading the entire request",
			Value:       DefaultReadTimeout,
		},
		&cli.DurationFlag{
			Name:        serverWriteTimeoutName,
			Destination: &ws.WriteTimeout,
			Sources:     envWrapper("WRITE_TIMEOUT"),
			Usage:       "Maximum duration before timing out writes of the response",
			Category:    categoryServer,
			Value:       DefaultWriteTimeout,
		},
		&cli.DurationFlag{
			Name:        serverReadHeaderTimeoutName,
			Destination: &ws.ReadHeaderTimeout,
			Sources:     envWrapper("READ_HEADER_TIMEOUT"),
			Usage:       "Amount of time allotted to read request headers",
			Category:    categoryServer,
			Value:       DefaultReadHeaderTimeout,
		},
	}...)

	return f
}
