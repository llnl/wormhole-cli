package routeregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/llnl/wormhole-cli/internal/requester"
)

type RegistryService interface {
	RegisterRoute(ctx context.Context, communityName, routeName string) (*RegistrationResponse, error)
	RefreshJWT(ctx context.Context, jwt string) (string, error)
	AddCommunity(ctx context.Context, name string) (*Community, error)
	RemoveCommunity(ctx context.Context, identifier string) error
	ResolveCommunity(ctx context.Context, identifier string) (*Community, error)
	ListCommunities(ctx context.Context, identifier string) ([]Community, error)
	AddCommunityRoute(ctx context.Context, communityID, routeID string) error
	RemoveCommunityRoute(ctx context.Context, communityID, routeID string) error

	ResolveRoute(ctx context.Context, identifier string) (*Route, error)
	ListRoutes(ctx context.Context, identifier string) ([]Route, error)
}

type RegistryClient struct {
	client   *http.Client
	token    string
	endpoint string
	logger   *slog.Logger
}

func NewRegistryClient(token, endpoint string, client *http.Client, logger *slog.Logger) *RegistryClient {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &RegistryClient{
		client:   client,
		token:    token,
		endpoint: endpoint,
		logger:   logger,
	}
}

// community

func (c *RegistryClient) AddCommunity(ctx context.Context, name string) (*Community, error) {
	return do[Community, *Community](c, ctx, http.MethodPost, "api/v1/community", &Community{Name: name})
}

func (c *RegistryClient) RemoveCommunity(ctx context.Context, identifier string) error {
	path := "api/v1/community/" + url.PathEscape(identifier)
	_, err := do[None, None](c, ctx, http.MethodDelete, path, nil)

	return err
}

func (c *RegistryClient) ResolveCommunity(ctx context.Context, identifier string) (*Community, error) {
	cl, err := c.ListCommunities(ctx, identifier)
	if err != nil {
		return nil, err
	}

	if len(cl) > 1 {
		// = 0 case covered by error check
		return nil, fmt.Errorf("community identifier not unique, found %d matches: %s", len(cl), identifier)
	}

	return &cl[0], nil
}

func (c *RegistryClient) ListCommunities(ctx context.Context, identifier string) ([]Community, error) {
	cl, err := do[None, []Community](c, ctx, http.MethodGet, "api/v1/community", nil)
	if err != nil {
		return nil, err
	}

	if identifier == "" {
		return cl, nil
	}

	var filtered []Community

	for i := range cl {
		if cl[i].Matches(identifier) {
			filtered = append(filtered, cl[i])
		}
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("community not found: %s", identifier)
	}

	return filtered, nil
}

func (c *RegistryClient) AddCommunityRoute(ctx context.Context, communityID, routeID string) error {
	path := fmt.Sprintf("api/v1/community/%s/route/%s", url.PathEscape(communityID), url.PathEscape(routeID))
	_, err := do[None, MembershipResponse](c, ctx, http.MethodPut, path, nil)

	return err
}

func (c *RegistryClient) RemoveCommunityRoute(ctx context.Context, communityID, routeID string) error {
	path := fmt.Sprintf("api/v1/community/%s/route/%s", url.PathEscape(communityID), url.PathEscape(routeID))
	_, err := do[None, None](c, ctx, http.MethodDelete, path, nil)

	return err
}

// routes

func (c *RegistryClient) RegisterRoute(ctx context.Context, communityName, routeName string) (*RegistrationResponse, error) {
	r, err := do[RegistrationRequest, *RegistrationResponse](c, ctx, http.MethodPost, "api/v2/route", &RegistrationRequest{
		Name:          routeName,
		CommunityName: &communityName,
	})

	return r, err
}

// RefreshJWT exchanges an existing Piko JWT for a new one. It deliberately
// uses X-JWT rather than the user's X-Token credential.
func (c *RegistryClient) RefreshJWT(ctx context.Context, jwt string) (string, error) {
	endpoint, err := url.JoinPath(c.endpoint, "api/latest/jwt")
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create JWT refresh request: %w", err)
	}
	req.Header.Set("X-JWT", jwt)
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute JWT refresh request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", requester.HttpResponseError{Code: resp.StatusCode}
	}
	var result struct {
		JWT string `json:"jwt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode JWT refresh response: %w", err)
	}
	if result.JWT == "" {
		return "", fmt.Errorf("JWT refresh response contains an empty JWT")
	}
	return result.JWT, nil
}

// ResolveRoute resolves a route by partial ID or fully qualified name.
// TODO refactor: list and resolve methods are nearly identical between community and route.
func (c *RegistryClient) ResolveRoute(ctx context.Context, identifier string) (*Route, error) {
	rl, err := c.ListRoutes(ctx, identifier)
	if err != nil {
		return nil, err
	}

	if len(rl) > 1 {
		return nil, fmt.Errorf("route identifier not unique, found %d matches: %s", len(rl), identifier)
	}

	return &rl[0], nil
}

func (c *RegistryClient) ListRoutes(ctx context.Context, identifier string) ([]Route, error) {
	rl, err := do[None, []Route](c, ctx, http.MethodGet, "api/v1/route", nil)
	if err != nil {
		return nil, err
	}

	if identifier == "" {
		return rl, nil
	}

	var filtered []Route

	for i := range rl {
		if rl[i].Matches(identifier) {
			filtered = append(filtered, rl[i])
		}
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("route not found: %s", identifier)
	}

	return filtered, nil
}

func do[Req any, Resp any](c *RegistryClient, ctx context.Context, method string, path string, req *Req) (Resp, error) {
	r := requester.NewTokenAuthRequester[Req, Resp](c.token, c.endpoint, c.client, c.logger)

	switch method {
	case http.MethodGet:
		return r.Get(ctx, path)
	case http.MethodPut:
		return r.Put(ctx, path, req)
	case http.MethodPost:
		return r.Post(ctx, path, req)
	case http.MethodDelete:
		return r.Delete(ctx, path)
	default:
		var zero Resp

		return zero, fmt.Errorf("unsupported method: %s", method)
	}
}
