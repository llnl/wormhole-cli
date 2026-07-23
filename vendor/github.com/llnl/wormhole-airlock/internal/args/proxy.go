package args

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/urfave/cli/v3"
)

const (
	categoryProxy = "Proxy"

	addPrefixName        = "add-prefix"
	authFileName         = "auth-file"
	authBearerHeaderName = "auth-bearer-header" //nolint: gosec
	headerGroupsName     = "header-group"
	headerUserName       = "header-user"
	jwtLeewayName        = "jwt-leeway"
	jwksName             = "jwks-url"
	proxyTargetName      = "proxy-target"
	responseHeadersName  = "response-headers"
	requestHeadersName   = "request-headers"
	stripPrefixName      = "strip-prefix"
	headerDebugName      = "header-debug"
)

type Proxy struct {
	AuthFile         string
	AuthBearerHeader bool
	HeaderDebug      bool
	HeaderGroups     string
	HeaderUser       string
	JWKSURL          string
	JWTLeeway        time.Duration
	MaxHeaderBytes   int
	PathPrefix       string
	RequestHeader    map[string]string
	ResponseHeaders  map[string]string
	StripPrefix      string
	Target           string
}

func (f *FlagBuilder) ProxyFlags(p *Proxy) *FlagBuilder {
	f.Flags = append(f.Flags, []cli.Flag{
		&cli.StringFlag{
			Aliases:     []string{"auth-json"},
			Destination: &p.AuthFile,
			Category:    categoryProxy,
			Name:        authFileName,
			Sources:     envWrapper([]string{"AUTH_FILE", "AUTH_JSON"}),
			Usage:       "Path to the application's authorization.json file or the full JSON contents",
			Value:       DefaultAuthJSON,
		},
		&cli.BoolFlag{
			Category:    categoryProxy,
			Destination: &p.AuthBearerHeader,
			Name:        authBearerHeaderName,
			Sources:     envWrapper("AUTH_BEARER_HEADER"),
			Usage:       "Indicates that the header 'Authorization: Bearer <x-token value>' should be set",
		},
		&cli.BoolFlag{
			Category:    categoryProxy,
			Destination: &p.HeaderDebug,
			Name:        headerDebugName,
			Sources:     envWrapper("HEADER_DEBUG"),
			Usage:       "Log all inbound requests to headers at debug level",
		},
		&cli.StringFlag{
			Category:    categoryProxy,
			Destination: &p.HeaderGroups,
			Name:        headerGroupsName,
			Sources:     envWrapper("HEADER_GROUPS"),
			Usage:       "Request header used for validated groups",
			Value:       DefaultHeaderGroups,
		},
		&cli.StringFlag{
			Category:    categoryProxy,
			Destination: &p.HeaderUser,
			Name:        headerUserName,
			Sources:     envWrapper("HEADER_USER"),
			Usage:       "Request header used for validated username",
			Value:       DefaultHeaderUser,
		},
		&cli.StringFlag{
			Category:    categoryProxy,
			Destination: &p.JWKSURL,
			Name:        jwksName,
			Sources:     envWrapper("JWKS_URL"),
			Usage:       "The URL to retrieve JSON Web Keys for JWT validation",
			Required:    true,
		},
		&cli.DurationFlag{
			Category:    categoryProxy,
			Destination: &p.JWTLeeway,
			Name:        jwtLeewayName,
			Sources:     envWrapper("JWT_LEEWAY"),
			Usage:       "The leeway period for JWT expiration",
			Value:       DefaultJWTLeeway,
		},
		&cli.StringFlag{
			Category:    categoryProxy,
			Destination: &p.PathPrefix,
			Name:        addPrefixName,
			Sources:     envWrapper("ADD_PREFIX"),
			Usage:       "Introduce the provided prefix (e.g., /wormhole-test) to all managed routes",
		},
		&cli.StringFlag{
			Action: func(_ context.Context, cCmd *cli.Command, s string) error {
				if cCmd.String(addPrefixName) != "" {
					// Defining both can easily lead to unexpected behavior. Its safe to prevent
					// that for the time being.
					return fmt.Errorf("--%s conflicts with --%s", stripPrefixName, addPrefixName)
				}

				_, err := url.Parse(s)
				if err != nil {
					return fmt.Errorf("invalid --%s defined: %w", stripPrefixName, err)
				}

				return nil
			},
			Category:    categoryProxy,
			Destination: &p.StripPrefix,
			Name:        stripPrefixName,
			Sources:     envWrapper("STRIP_PREFIX"),
			Usage:       "Remove the defined route prefix from all inbound requests (conflicts with --add-prefix)",
		},
		&cli.StringMapFlag{
			Category:    categoryProxy,
			Destination: &p.RequestHeader,
			Name:        requestHeadersName,
			Sources:     envWrapper("REQUEST_HEADERS"),
			Usage:       "List of headers to add to requests in the format <key>=<value>",
		},
		&cli.StringMapFlag{
			Category:    categoryProxy,
			Destination: &p.ResponseHeaders,
			Name:        responseHeadersName,
			Sources:     envWrapper("RESPONSE_HEADERS"),
			Usage:       "List of headers to add to responses in the format <key>=<value>",
		},
		&cli.StringFlag{
			Category:    categoryProxy,
			Destination: &p.Target,
			Name:        proxyTargetName,
			Sources:     envWrapper("PROXY_TARGET"),
			Usage:       "The target address the proxy will forward requests to",
			Required:    true,
		},
	}...)

	return f
}
