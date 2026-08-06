package wh

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/llnl/wormhole-cli/internal/routeregistry"
)

/* Community */

func getCommunitiesTable(communities []routeregistry.Community) ([]string, [][]string) {
	headers := []string{"ID", "Name", "Routes"}
	data := make([][]string, len(communities))

	for i, c := range communities {
		data[i] = []string{
			truncateID(ptrToString(c.ID)),
			c.Name,
			strconv.Itoa(len(c.Routes)),
		}
	}

	return headers, data
}

func formatCommunityList(communities []routeregistry.Community) string {
	if len(communities) == 0 {
		return ""
	}

	headers, data := getCommunitiesTable(communities)

	return formatTable(headers, data)
}

func formatCommunitySingle(c *routeregistry.Community) string {
	if c == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Community:\n")
	fmt.Fprintf(&sb, "ID:        %s\n", ptrToString(c.ID))
	fmt.Fprintf(&sb, "name:      %s\n", c.Name)

	if len(c.Routes) > 0 {
		sb.WriteString("Routes:\n")
		sb.WriteString(formatRouteList(c.Routes))
	}

	return sb.String()
}

/* Route */

func getRoutesTable(routes []routeregistry.Route) ([]string, [][]string) {
	headers := []string{"Route ID", "Route Name", "Community Name", "URL", "Allowed", "Disallowed"}
	data := make([][]string, len(routes))

	for i, r := range routes {
		routeName := r.Name
		if routeName != "" {
			if r.DomainName != nil {
				routeName = fmt.Sprintf("%s/%s", *r.DomainName, routeName)
			}
		}

		var allowedList, disallowedList string

		if r.Rules != nil {
			if r.Rules.Allowed != nil {
				allowedList = strings.Join(append(r.Rules.Allowed.Users, r.Rules.Allowed.Groups...), ", ")
			}

			if r.Rules.Disallowed != nil {
				disallowedList = strings.Join(append(r.Rules.Disallowed.Users, r.Rules.Disallowed.Groups...), ", ")
			}
		}

		data[i] = []string{
			truncateID(ptrToString(r.ID)),
			routeName,
			ptrToString(r.CommunityName),
			ptrToString(r.URL),
			allowedList,
			disallowedList,
		}
	}

	return headers, data
}

func formatRouteList(routes []routeregistry.Route) string {
	if len(routes) == 0 {
		return ""
	}

	headers, data := getRoutesTable(routes)

	return formatTable(headers, data)
}

func formatRouteSingle(r *routeregistry.Route) string {
	if r == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Route:\n")
	fmt.Fprintf(&sb, "ID:        %s\n", ptrToString(r.ID))
	fmt.Fprintf(&sb, "Name:      %s\n", r.Name)
	fmt.Fprintf(&sb, "Community: %s\n", ptrToString(r.CommunityName))
	fmt.Fprintf(&sb, "Domain:    %s\n", ptrToString(r.DomainName))
	fmt.Fprintf(&sb, "URL:       %s\n", ptrToString(r.URL))

	if r.Rules != nil && r.Rules.Allowed != nil {
		sb.WriteString("Allowed:\n")
		sb.WriteString(formatEntitySetTable(r.Rules.Allowed))
	}

	if r.Rules != nil && r.Rules.Disallowed != nil {
		sb.WriteString("Disallowed:\n")
		sb.WriteString(formatEntitySetTable(r.Rules.Disallowed))
	}

	return sb.String()
}

/* EntitySet */

func formatEntitySetTable(es *routeregistry.EntitySet) string {
	if es == nil {
		return ""
	}

	var (
		nUsers  = len(es.Users)
		nGroups = len(es.Groups)
		nTotal  = max(nUsers, nGroups)
		headers = []string{"Users", "Groups"}
		data    = make([][]string, nTotal)
	)

	for i := range nTotal {
		user := ""
		group := ""

		if i < nUsers {
			user = es.Users[i]
		}

		if i < nGroups {
			group = es.Groups[i]
		}

		data[i] = []string{
			user,
			group,
		}
	}

	return formatTable(headers, data)
}

/* Common */

func ptrToString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

func truncateID(id string) string {
	if len(id) > routeregistry.MinIDLength {
		return id[:routeregistry.MinIDLength]
	}

	return id
}

func formatTable(headers []string, data [][]string) string {
	if len(data) == 0 {
		return ""
	}

	maxW := make([]int, len(headers))
	for i, h := range headers {
		maxW[i] = len(h)
	}

	for _, row := range data {
		for j, val := range row {
			if j < len(maxW) {
				maxW[j] = max(maxW[j], len(val))
			}
		}
	}

	var sb strings.Builder
	formatRow(&sb, headers, maxW)

	for _, row := range data {
		formatRow(&sb, row, maxW)
	}

	return sb.String()
}

func formatRow(sb *strings.Builder, row []string, maxW []int) {
	for i, val := range row {
		if i < len(maxW) {
			fmt.Fprintf(sb, "%-*s", maxW[i], val)
		} else {
			sb.WriteString(val)
		}

		if i < len(row)-1 {
			sb.WriteString("  ")
		}
	}

	sb.WriteString("\n")
}
