package wh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"

	"github.com/llnl/wormhole-airlock/pkg/airlock"
	"github.com/llnl/wormhole-cli/internal/cmd/wh/args"
	"github.com/llnl/wormhole-cli/internal/ns"
	wormholepiko "github.com/llnl/wormhole-cli/internal/piko"
	"github.com/llnl/wormhole-cli/internal/routeregistry"
	"github.com/llnl/wormhole-cli/internal/selfheal"
	"github.com/llnl/wormhole-cli/internal/version"
)

type accessUserGroups struct {
	Users  []string `json:"users"`
	Groups []string `json:"groups"`
}

type airlockAuthV1 struct {
	Version string `json:"version"`
	Allowed struct {
		Users  []string `json:"users"`
		Groups []string `json:"groups"`
	} `json:"allowed"`
	Disallowed struct {
		Users  []string `json:"users"`
		Groups []string `json:"groups"`
	} `json:"disallowed"`
	Token string `json:"token"`
}

type airlockConfig struct {
	Addr                string
	JwksEndpoint        string
	TargetPort          int
	AllowedUsers        []string
	AllowedGroups       []string
	ForbiddenUsers      []string
	ForbiddenGroups     []string
	ForwardUserHeader   string
	ForwardGroupsHeader string
	AuthBearerHeader    bool
}

const (
	jwksWellKnownSuffix = ".well-known/jwks.json"
)

func intersect[T comparable](allowed, forbidden []T) []T {
	intersectionSet := make([]T, 0)

	for _, searchValue := range allowed {
		if slices.Contains(forbidden, searchValue) {
			intersectionSet = append(intersectionSet, searchValue)
		}
	}

	return intersectionSet
}

func launchAirlock(g *errgroup.Group, ctx context.Context, token string, config airlockConfig, verbose bool) error {
	authData := airlockAuthV1{
		Version: "1.0",
		Allowed: accessUserGroups{
			Users:  config.AllowedUsers,
			Groups: config.AllowedGroups,
		},
		Disallowed: accessUserGroups{
			Users:  config.ForbiddenUsers,
			Groups: config.ForbiddenGroups,
		},
		Token: token,
	}

	authDataJson, err := json.Marshal(authData)
	if err != nil {
		return err
	}

	// join config.JwksEndpoint and jwksWellKnownSuffix
	jwksEndpoint, err := url.Parse(config.JwksEndpoint)
	if err != nil {
		return err
	}

	jwksEndpoint.Path = path.Join(jwksEndpoint.Path, jwksWellKnownSuffix)

	// Set logging level based on verbose flag
	logLevel := "warn"
	if verbose {
		logLevel = "debug"
	}

	// create Airlock Proxy
	opts := airlock.Options{
		Web: airlock.Web{
			Address: config.Addr,
		},
		Proxy: airlock.Proxy{
			AuthJSON:         string(authDataJson),
			JWKSURL:          jwksEndpoint.String(),
			HeaderGroups:     config.ForwardGroupsHeader,
			HeaderUser:       config.ForwardUserHeader,
			AuthBearerHeader: config.AuthBearerHeader,
		},
		Logging: airlock.Logging{
			Level: logLevel,
		},
	}

	g.Go(func() error {
		return opts.StartProxy(ctx, fmt.Sprintf("http://localhost:%d", config.TargetPort))
	})

	return nil
}

func handleOpen(ctx context.Context, cCmd *cli.Command, a *args.CLIArgs, service routeregistry.RegistryService, logger *slog.Logger) error {
	verbose := a.Global.Verbose

	sidecar := func(cCtx context.Context) error {
		return openWormhole(cCtx, a, service, logger, verbose)
	}

	if cCmd.NArg() > 0 {
		nsConfig, err := ns.Initialize()
		if err != nil {
			logger.Error("Namespaces are not available on this system")
			log.Fatal(err)
		}

		if os.Getenv("_CONTAINERS_USERNS_CONFIGURED") == "init" {
			// Namespace: second stage launch
			podman := a.Open.PodmanCompat

			return nsConfig.Exec(ctx, cCmd.Args().Slice(), podman, sidecar)
		} else {
			// Namespace: first stage launch
			// preserve the original option ordering for the second stage
			secondStageCmd := append([]string{"/proc/self/exe"}, os.Args[1:]...)

			return nsConfig.LaunchNS(ctx, "slirp4netns", secondStageCmd)
		}
	} else {
		// normal execution flow
		return sidecar(ctx)
	}
}

func openWormhole(ctx context.Context, a *args.CLIArgs, registry routeregistry.RegistryService, logger *slog.Logger, verbose bool) error {
	fmt.Printf("Using wormhole-cli v%s\n", version.GetVersion())

	token := a.Global.Token

	port, err := strconv.Atoi(a.Open.AppPort)
	if err != nil {
		log.Fatal(err)
	}

	splitFn := func(c rune) bool {
		return c == ','
	}

	// split allowed and excluded user flags into slices
	allowedUsers := strings.FieldsFunc(a.Open.AllowedUsers, splitFn)
	allowedGroups := strings.FieldsFunc(a.Open.AllowedGroups, splitFn)

	// ensure that either a set of allowed users or groups is set
	if len(allowedUsers)+len(allowedGroups) <= 0 {
		log.Fatal(errors.New("cannot create a wormhole with no allowed users and groups"))
	}

	forbiddenUsers := strings.FieldsFunc(a.Open.ForbiddenUsers, splitFn)
	forbiddenGroups := strings.FieldsFunc(a.Open.ForbiddenGroups, splitFn)

	// compute slice intersection and error if users are both allowed and excluded
	// as of 02/03/2026 this causes an error with duplicate routes in the registry
	allowedForbiddenUsers := intersect(allowedUsers, forbiddenUsers)
	if len(allowedForbiddenUsers) != 0 {
		log.Fatal(fmt.Errorf("cannot both allow and forbid access for %q", allowedForbiddenUsers))
	}

	allowedForbiddenGroups := intersect(allowedGroups, forbiddenGroups)
	if len(allowedForbiddenGroups) != 0 {
		log.Fatal(fmt.Errorf("cannot both allow and forbid access for %q", allowedForbiddenGroups))
	}

	// get airlock configuration for user and group headers
	forwardUserHeader := a.Open.ForwardedHeaderUser
	forwardGroupsHeader := a.Open.ForwardedHeaderGroups

	selfHeal := a.SelfHeal
	if selfHeal.MinRetryBackoff == 0 && selfHeal.MaxRetryBackoff == 0 {
		selfHeal = selfheal.DefaultConfig()
	}

	if err := selfHeal.Validate(); err != nil {
		return err
	}

	register := func(registerCtx context.Context) (*routeregistry.RegistrationResponse, error) {
		return selfheal.RegisterRoute(registerCtx, registry.RegisterRoute, a.Open.Community, a.Open.Name, selfHeal, logger)
	}

	pikoLogger, err := wormholepiko.NewLogger(verbose)
	if err != nil {
		return fmt.Errorf("create Piko logger: %w", err)
	}

	defer func() { _ = pikoLogger.Sync() }()

	routeData, err := register(ctx)
	if err != nil {
		return err
	}

	if routeData == nil {
		return errors.New("route registration returned no response")
	}

	if routeData.Airlock.JwtIssuerURL == nil || *routeData.Airlock.JwtIssuerURL == "" {
		return errors.New("route registration response is missing Airlock JWT issuer URL")
	}

	// briefly open a tcp socket to get a random open port then use that port for airlock
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatal(err)
	}

	addr := listener.Addr().String()
	_ = listener.Close()

	// construct config for Airlock
	airlockConfig := airlockConfig{
		Addr:                addr,
		TargetPort:          port,
		JwksEndpoint:        *routeData.Airlock.JwtIssuerURL,
		AllowedUsers:        allowedUsers,
		AllowedGroups:       allowedGroups,
		ForbiddenUsers:      forbiddenUsers,
		ForbiddenGroups:     forbiddenGroups,
		ForwardUserHeader:   forwardUserHeader,
		ForwardGroupsHeader: forwardGroupsHeader,
		AuthBearerHeader:    a.Open.AuthBearerHeader,
	}

	// create errgroup for managing goroutines
	g, gCtx := errgroup.WithContext(ctx)

	// launch internal airlock sub component
	// TODO: investigate if we can do this without opening an external port for airlock
	err = launchAirlock(g, gCtx, token, airlockConfig, verbose)
	if err != nil {
		return fmt.Errorf("airlock error: %w", err)
	}

	g.Go(func() error {
		listen := func(
			listenCtx context.Context,
			credentials wormholepiko.Credentials,
			targetAddr string,
			minBackoff, maxBackoff time.Duration,
		) (wormholepiko.Forwarder, error) {
			return wormholepiko.ListenAndForward(listenCtx, credentials, targetAddr, minBackoff, maxBackoff, pikoLogger)
		}

		err := selfheal.Run(gCtx, routeData, register, registry.RefreshJWT, listen, airlockConfig.Addr, selfHeal, logger, func(publicURL string) {
			log.Printf("Successfully Opened a Wormhole!\n")
			log.Printf("URL: %s\n", publicURL)
		})

		if errors.Is(err, context.Canceled) && ctx.Err() != nil {
			return ctx.Err()
		}

		return err
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("wormhole error: %w", err)
	}

	return nil
}
