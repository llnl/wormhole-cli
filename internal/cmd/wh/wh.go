package wh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	piko "github.com/andydunstall/piko/client"
	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/llnl/wormhole-airlock/pkg/airlock"
	"github.com/llnl/wormhole-cli/internal/cmd/wh/args"
	"github.com/llnl/wormhole-cli/internal/ns"
	"github.com/llnl/wormhole-cli/internal/routeregistry"
	"github.com/llnl/wormhole-cli/internal/version"
)

type accessUserGroups struct {
	Users  []string `json:"users"`
	Groups []string `json:"groups"`
}

type createRouteOptionsV2 struct {
	Name      string `json:"name"`
	Community string `json:"community_name"`
}

type createRouteResponseV2 struct {
	Url     string `json:"url"`
	Airlock struct {
		JwtIssuerUrl string `json:"jwt_issuer_url"`
	} `json:"airlock"`
	Tunnel struct {
		URL        string `json:"url"`
		JWT        string `json:"jwt"`
		EndpointID string `json:"endpoint"`
	} `json:"tunnel"`
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
	jwksWellKnownSuffix  = ".well-known/jwks.json"
	routeRegistryAPIPath = "/api/v2/route"
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

func registerRoute(token, endpoint string, options createRouteOptionsV2, logger *slog.Logger) (data createRouteResponseV2, err error) {
	payload, err := json.Marshal(options)
	if err != nil {
		return data, err
	}

	logger.Info("Registering route",
		slog.String("name", options.Name),
		slog.String("endpoint", endpoint))

	url, err := url.JoinPath(endpoint, routeRegistryAPIPath)
	if err != nil {
		return data, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return data, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Token", token)

	// set a default timeout of 30s when creating route to prevent indefinite hangs
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	logger.Info("Sending route registration request")

	resp, err := client.Do(req)
	if err != nil {
		return data, err
	}
	defer resp.Body.Close()

	logger.Info("Received response", slog.Int("status_code", resp.StatusCode))

	switch resp.StatusCode {
	case 404:
		return data, fmt.Errorf("Unable to create route: Invalid URL [404]")
	case 401:
		return data, fmt.Errorf("Unable to create route: Invalid Token [401]")
	case 301:
		return data, fmt.Errorf("Unable to create route: URL has moved [301]")
	case 200:
		break
	default:
		return data, fmt.Errorf("Unable to create route [%d]", resp.StatusCode)
	}
	decoder := json.NewDecoder(resp.Body)

	err = decoder.Decode(&data)
	if err != nil {
		return data, err
	}

	logger.Info("Route registered successfully",
		slog.String("url", data.Url),
		slog.String("tunnel_endpoint", data.Tunnel.EndpointID))

	return data, nil
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

	//create Airlock Proxy
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

func launchPiko(g *errgroup.Group, ctx context.Context, jwt, endpointURL, endpointID, targetAddr string, verbose bool) error {
	// create logger for piko with appropriate log level
	config := zap.NewProductionConfig()
	if verbose {
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	}

	logger, err := config.Build()
	if err != nil {
		return fmt.Errorf("failed to create logger for piko: %w", err)
	}
	defer logger.Sync()

	// parse relay endpoint URL
	pikoURL, err := url.Parse(endpointURL)
	if err != nil {
		return err
	}

	// construct piko config
	upstream := &piko.Upstream{
		Logger: logger,
		Token:  jwt,
		URL:    pikoURL,
	}

	g.Go(func() error {
		forwarder, err := upstream.ListenAndForward(
			ctx, endpointID, targetAddr,
		)
		if err != nil {
			return fmt.Errorf("piko listen: %w", err)
		}
		defer forwarder.Close()

		if err := forwarder.Wait(); err != nil {
			return fmt.Errorf("piko forwarder: %w", err)
		}
		return nil
	})
	return nil
}

func doAsync(f func() error) chan error {
	done := make(chan error, 1)
	go func() {
		done <- f()
		close(done)
	}()
	return done
}

// TODO convert route registration to use service layer
func handleOpen(ctx context.Context, cCmd *cli.Command, a *args.CLIArgs, service routeregistry.RegistryService, logger *slog.Logger) error {
	verbose := a.Global.Verbose

	sidecar := func(cCtx context.Context) error {
		return openWormhole(cCtx, a, logger, verbose)
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

func openWormhole(ctx context.Context, a *args.CLIArgs, logger *slog.Logger, verbose bool) error {
	fmt.Printf("Using wormhole-cli v%s\n", version.GetVersion())

	token := a.Global.Token
	endpoint := a.Global.Endpoint
	port, err := strconv.Atoi(a.Open.AppPort)
	if err != nil {
		log.Fatal(err)
	}

	routeOptions := createRouteOptionsV2{
		Name:      a.Open.Name,
		Community: a.Open.Community,
	}

	splitFn := func(c rune) bool {
		return c == ','
	}

	// split allowed and excluded user flags into slices
	allowedUsers := strings.FieldsFunc(a.Open.AllowedUsers, splitFn)
	allowedGroups := strings.FieldsFunc(a.Open.AllowedGroups, splitFn)

	// ensure that either a set of allowed users or groups is set
	if len(allowedUsers)+len(allowedGroups) <= 0 {
		log.Fatal(fmt.Errorf("Error: cannot create a wormhole with no allowed users and groups\n"))
	}

	forbiddenUsers := strings.FieldsFunc(a.Open.ForbiddenUsers, splitFn)
	forbiddenGroups := strings.FieldsFunc(a.Open.ForbiddenGroups, splitFn)

	// compute slice intersection and error if users are both allowed and excluded
	// as of 02/03/2026 this causes an error with duplicate routes in the registry
	allowedForbiddenUsers := intersect(allowedUsers, forbiddenUsers)
	if len(allowedForbiddenUsers) != 0 {
		log.Fatal(fmt.Errorf("Error: cannot both allow and forbid access for %q\n", allowedForbiddenUsers))
	}

	allowedForbiddenGroups := intersect(allowedGroups, forbiddenGroups)
	if len(allowedForbiddenGroups) != 0 {
		log.Fatal(fmt.Errorf("Error: cannot both allow and forbid access for %q\n", allowedForbiddenGroups))
	}

	// get airlock configuration for user and group headers
	forwardUserHeader := a.Open.ForwardedHeaderUser
	forwardGroupsHeader := a.Open.ForwardedHeaderGroups

	// briefly open a tcp socket to get a random open port then use that port for airlock
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatal(err)
	}
	addr := listener.Addr().String()
	listener.Close()

	routeData, err := registerRoute(token, endpoint, routeOptions, logger)
	if err != nil {
		log.Fatal(err)
	}

	// construct config for Airlock
	airlockConfig := airlockConfig{
		Addr:                addr,
		TargetPort:          port,
		JwksEndpoint:        routeData.Airlock.JwtIssuerUrl,
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

	// launch internal piko relay and point it at airlock
	err = launchPiko(g, gCtx, routeData.Tunnel.JWT, routeData.Tunnel.URL, routeData.Tunnel.EndpointID, airlockConfig.Addr, verbose)
	if err != nil {
		return fmt.Errorf("piko error: %w", err)
	}

	log.Printf("Successfully Opened a Wormhole!\n")
	log.Printf("URL: %s\n", routeData.Url)

	done := doAsync(func() error {
		if err := g.Wait(); err != nil {
			return fmt.Errorf("wormhole error: %w", err)
		}
		return nil
	})

	select {
	case <-gCtx.Done():
		// remove SIGINT if airlock switches to accepting context cancellation
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		for {
			// send SIGINT until terminate
			// workaround for child process exiting before airlock is listening for SIGINT
			timeout := time.After(100 * time.Millisecond)
			select {
			case <-done:
				return nil
			case <-timeout:
				syscall.Kill(syscall.Getpid(), syscall.SIGINT)
			}
		}
	case err := <-done:
		return err
	}
}
