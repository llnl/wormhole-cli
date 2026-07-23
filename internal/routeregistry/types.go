package routeregistry

import (
	"fmt"
	"strings"
)

const MinIDLength = 8

// Expanded types generated from route-registry.json schemas.

type Identifiable interface {
	Matches(string) bool
}

type Airlock struct {
	JwtIssuerURL *string `json:"jwt_issuer_url,omitempty"`
}

type Tunnel struct {
	URL      *string `json:"url,omitempty"`
	JWT      *string `json:"jwt,omitempty"`
	Endpoint *string `json:"endpoint,omitempty"`
}

type EntitySet struct {
	Users  []string `json:"users,omitempty"`
	Groups []string `json:"groups,omitempty"`
}

type Rules struct {
	Version    *string    `json:"version,omitempty"`
	Allowed    *EntitySet `json:"allowed,omitempty"`
	Disallowed *EntitySet `json:"disallowed,omitempty"`
}

type Route struct {
	Name          string  `json:"name"`
	DomainName    *string `json:"domain_name,omitempty"`
	CommunityName *string `json:"community_name,omitempty"`
	ID            *string `json:"id,omitempty"`
	Dst           *string `json:"dst,omitempty"`
	Src           *string `json:"src,omitempty"`
	URL           *string `json:"url,omitempty"`
	Token         *string `json:"token,omitempty"`
	Endpoint      *string `json:"endpoint,omitempty"`
	JWT           *string `json:"jwt,omitempty"`
	CommunityID   *string `json:"community_id,omitempty"`
	Rules         *Rules  `json:"rules,omitempty"`
}

type Community struct {
	Name   string  `json:"name"`
	ID     *string `json:"id"`
	Routes []Route `json:"routes,omitempty"`
}

type None struct{}

type RegistrationRequest struct {
	Name          string  `json:"name"`
	DomainName    *string `json:"domain_name,omitempty"`
	CommunityName *string `json:"community_name,omitempty"`
}

type RegistrationResponse struct {
	URL      string  `json:"url"`
	Airlock  Airlock `json:"airlock"`
	Tunnel   Tunnel  `json:"tunnel"`
	Endpoint *string `json:"endpoint,omitempty"`
	JWT      *string `json:"jwt,omitempty"`
}

type MembershipResponse struct {
	EntityID    *string `json:"entity_id,omitempty"`
	CommunityID *string `json:"community_id,omitempty"`
	RouteID     *string `json:"route_id,omitempty"`
	Status      *string `json:"status,omitempty"`
}

type JWKSet struct {
	Keys []map[string]any `json:"keys,omitempty"`
}

type ValidationError struct {
	Loc   []any          `json:"loc"`
	Msg   string         `json:"msg"`
	Type  string         `json:"type"`
	Input any            `json:"input,omitempty"`
	Ctx   map[string]any `json:"ctx,omitempty"`
}

type HTTPValidationError struct {
	Detail []ValidationError `json:"detail,omitempty"`
}

// match partial ID or full name
func (c *Community) Matches(identifier string) bool {
	if c.Name == identifier || c.ID != nil && len(identifier) >= MinIDLength && strings.HasPrefix(*c.ID, identifier) {
		return true
	}

	return false
}

// match partial ID or fully qualified name
func (r *Route) Matches(identifier string) bool {
	if r.ID != nil && len(identifier) >= MinIDLength && strings.HasPrefix(*r.ID, identifier) {
		return true
	}

	if r.DomainName != nil {
		name := fmt.Sprintf("%s/%s", *r.DomainName, r.Name)
		if name == identifier {
			return true
		}
	}

	return false
}
