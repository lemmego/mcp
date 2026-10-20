package mcp

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/lemmego/api/app"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// handlerBuilder is api's Bootstrapper.Handler, narrowed to the one method
// this needs. Asserted rather than required, so a host that does not provide
// it degrades to an explanation instead of a panic.
type handlerBuilder interface {
	Handler() http.Handler
}

// materialiseRouteTable makes the routing table readable from a CLI command.
//
// Route callbacks are invoked by serveHandler, which runs when the application
// serves HTTP. `mcp:serve` is a console command, so nothing had registered a
// route and list-routes reported an empty table on an application serving
// dozens — which is worse than no tool, because an agent would conclude the
// routes do not exist.
//
// Handler() is the sanctioned way to get there: its own doc says it boots the
// application and returns the handler Run would serve, without binding a port.
// The handler is discarded; the side effect on the router is the point.
//
// Once, because registerRoutes appends to the mux and calling it twice would
// register everything again.
func (s *Server) materialiseRouteTable() {
	s.routesOnce.Do(func() {
		if builder, ok := s.app.(handlerBuilder); ok {
			_ = builder.Handler()
		}
	})
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
