package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/lemmego/api/app"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// materialiseRouteTable makes the routing table readable from a CLI command.
//
// Route callbacks are invoked on the way to serving HTTP, and `mcp:serve` is a
// console command — so nothing had registered a route and list-routes reported
// an empty table on an application serving dozens, which is worse than no tool
// because an agent reads it as "the routes do not exist".
//
// api v0.7.1 added PopulateRoutes for exactly this. The first attempt here
// called Handler() instead, which re-runs the providers on an application that
// has already booted and panics with "service already registered" — a real
// crash, found by calling the tool rather than by reasoning about it.
//
// PopulateRoutes is itself idempotent; the Once only avoids the repeat call.
func (s *Server) materialiseRouteTable() {
	s.routesOnce.Do(s.app.PopulateRoutes)
}

type listRoutesInput struct {
	Method string `json:"method,omitempty" jsonschema:"filter to one HTTP method such as GET"`
	Path   string `json:"path,omitempty" jsonschema:"filter to paths containing this substring"`
}

type listRoutesOutput struct {
	Routes []app.RouteInfo `json:"routes"`
	Total  int             `json:"total"`
}

func (s *Server) addListRoutes(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name: "list-routes",
		Description: "Every route this application registers, with its method, resolved path, " +
			"and how many handlers and middlewares sit on it. Use it before adding a route, " +
			"to find the one serving a URL, or to check whether a path is behind a guard. " +
			"Routes mounted directly on the ServeMux with Handle or HandleFunc do not appear, " +
			"so an empty result does not mean nothing is served.",
	}, s.listRoutes)
}

func (s *Server) listRoutes(_ context.Context, _ *sdk.CallToolRequest, in listRoutesInput) (*sdk.CallToolResult, listRoutesOutput, error) {
	router := s.app.Router()
	if router == nil {
		return failure("this application has no router"), listRoutesOutput{}, nil
	}
	s.materialiseRouteTable()

	all := router.Routes()
	method := strings.ToUpper(strings.TrimSpace(in.Method))
	needle := strings.TrimSpace(in.Path)

	matched := make([]app.RouteInfo, 0, len(all))
	for _, route := range all {
		if method != "" && !strings.EqualFold(route.Method, method) {
			continue
		}
		if needle != "" && !strings.Contains(route.Path, needle) {
			continue
		}
		matched = append(matched, route)
	}

	out := listRoutesOutput{Routes: matched, Total: len(all)}

	var b strings.Builder
	if len(matched) == 0 {
		fmt.Fprintf(&b, "no route matched (%d registered in total)\n", len(all))
		return text(b.String()), out, nil
	}
	fmt.Fprintf(&b, "%d of %d routes:\n\n", len(matched), len(all))
	for _, route := range matched {
		fmt.Fprintf(&b, "  %-7s %-44s handlers=%d", route.Method, route.Path, route.Handlers)
		if route.Middlewares > 0 {
			fmt.Fprintf(&b, " middlewares=%d", route.Middlewares)
		}
		b.WriteByte('\n')
	}
	return text(b.String()), out, nil
}
