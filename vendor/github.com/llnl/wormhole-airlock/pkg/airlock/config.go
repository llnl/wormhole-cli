package airlock

import (
	"reflect"
	"time"

	"github.com/llnl/wormhole-airlock/internal/args"
)

// Options defines all potential configurations available to you that will include
// the deployment of Airlock. All variable are optional and have related defaults
// where appropriate.
type Options struct {
	// Web relates directly to the initialization of the web service.
	Web Web
	// Proxy manage logic relating to the core proxy requirements in Airlock.
	Proxy Proxy
	// Logging controls when and where any log message are written.
	Logging Logging
}

type Web struct {
	// Address (host:port) the server should listen on (default: :3128).
	Address string
	// MaxHeaderBytes controls the maximum number of bytes the server will read parsing the
	// request headers
	MaxHeaderBytes int
	// ReadTimeout is the maximum duration for reading the entire request
	// (default: 0). A zero or negative value means there will be no timeout.
	ReadTimeout time.Duration
	// WriteTimeout is the maximum duration before timing out write of the
	// response (default: 0). A zero or negative value means there will be no timeout.
	WriteTimeout time.Duration
	// ReadHeaderTimeout indicates the amount of time allotted to read request
	// headers (default: 0). A zero or negative value means there will be no timeout.
	ReadHeaderTimeout time.Duration
}

type Proxy struct {
	// AddPrefix introduces the provided prefix to all managed routes.
	AddPrefix string
	// AuthJSON path or contents of the application's authorization.json file
	// (default: /etc/airlock/authorization.json)
	AuthJSON string
	// AuthBearerHeader indicates that `Authorization: Bearer <x-token value>` should be set.
	AuthBearerHeader bool
	// JWTLeeway time allowed after JWT expiration (default: 30s).
	JWTLeeway time.Duration
	// JWKSURL to retrieve public keys used in JWT validation.
	JWKSURL string
	// ResponseHeaders key/value pairs to be added to every response.
	ResponseHeaders map[string]string
	// RequestHeaders key/value pairs to be added to every request.
	RequestHeaders map[string]string
	// StripPrefix Remove the defined route prefix from all inbound requests
	// (conflicts with --add-prefix).
	StripPrefix string
	// HeaderDebug Log all inbound requests to headers at debug level (do not
	// use with production date/tokens).
	HeaderDebug bool
	// HeaderGroups used for validated groups in request (default: X-LC-Groups).
	HeaderGroups string
	// HeaderUser used for validated username in request (default: X-LC-User).
	HeaderUser string
}

type Logging struct {
	// Disable indicated all logging should be turned off.
	Disable bool
	// Level (e.g., debug, info, warn, or error) for all logs (default: info).
	Level string
	// Location identifies where logs will be saved, this can be a distinct file or
	// syslog (default stdout).
	Location string
	// Network optionally specified (e.g., tcp) used for remote log daemon
	// connections
	Network string
	// Address optionally specified (e.g., localhost:1234) to be used for remote
	// log daemon connections
	Address string
}

//

func defaults() Options {
	return Options{
		Web: Web{
			Address:           args.DefaultAddress,
			MaxHeaderBytes:    args.DefaultMaxHeaderBytes,
			ReadTimeout:       args.DefaultReadTimeout,
			WriteTimeout:      args.DefaultWriteTimeout,
			ReadHeaderTimeout: args.DefaultReadHeaderTimeout,
		},
		Proxy: Proxy{
			AuthJSON:     args.DefaultAuthJSON,
			JWTLeeway:    args.DefaultJWTLeeway,
			HeaderGroups: args.DefaultHeaderGroups,
			HeaderUser:   args.DefaultHeaderUser,
		},
		Logging: Logging{
			Level:    args.DefaultLoggingLevel,
			Location: args.DefaultLoggingLocation,
		},
	}
}

// mergeStructs combines two structs of the same type. Non-zero values in src
// overwrite the values in dest.
func mergeStructs(dest, src any) {
	dVal := reflect.ValueOf(dest).Elem()
	sVal := reflect.ValueOf(src).Elem()

	for i := range dVal.NumField() {
		destField := dVal.Field(i)
		srcField := sVal.Field(i)

		// Check if the field is a struct
		if srcField.Kind() == reflect.Struct && destField.Kind() == reflect.Struct {
			// Recursive call to merge structs
			mergeStructs(destField.Addr().Interface(), srcField.Addr().Interface())
		} else { //nolint: gocritic
			// Merge non-zero fields from src into dest
			if !reflect.DeepEqual(
				srcField.Interface(),
				reflect.Zero(srcField.Type()).Interface(),
			) {
				if destField.CanSet() {
					destField.Set(srcField)
				}
			}
		}
	}
}
