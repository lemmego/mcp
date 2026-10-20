package mcp

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/lemmego/api/db"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type emptyInput struct{}

// --- app-info ---------------------------------------------------------------

type appInfoOutput struct {
	GoVersion   string            `json:"go_version"`
	Environment string            `json:"environment"`
	Production  bool              `json:"production"`
	Platform    string            `json:"platform"`
	Database    string            `json:"database,omitempty"`
	Drivers     map[string]string `json:"drivers"`
	Modules     map[string]string `json:"lemmego_modules"`
}

func (s *Server) addAppInfo(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name: "app-info",
		Description: "Versions and wiring of this Lemmego application: Go version, environment, " +
			"the database dialect in use, the driver chosen for each subsystem, and the version " +
			"of every lemmego module it depends on. Read this before assuming an API exists — " +
			"the pinned version decides that.",
	}, s.appInfo)
}

func (s *Server) appInfo(_ context.Context, _ *sdk.CallToolRequest, _ emptyInput) (*sdk.CallToolResult, appInfoOutput, error) {
	cfg := s.app.Config()

	out := appInfoOutput{
		GoVersion:   runtime.Version(),
		Environment: asString(cfg.Get("app.env")),
		Production:  s.app.InProduction(),
		Platform:    runtime.GOOS + "/" + runtime.GOARCH,
		Drivers: map[string]string{
			"session": asString(cfg.Get("session.driver")),
			"cache":   asString(cfg.Get("cache.driver")),
			"tasker":  asString(cfg.Get("tasker.driver")),
		},
		Modules: lemmegoModules(),
	}
	// The dialect from the live connection rather than from configuration:
	// configuration says what was asked for, the connection says what answered.
	if conn, ok := db.Resolve(s.app); ok && conn != nil {
		out.Database = fmt.Sprintf("%s (connection %q)", conn.Dialect(), conn.Name())
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Go %s on %s\n", out.GoVersion, out.Platform)
	fmt.Fprintf(&b, "environment: %s (production=%t)\n", out.Environment, out.Production)
	if out.Database != "" {
		fmt.Fprintf(&b, "database: %s\n", out.Database)
	}
	b.WriteString("\ndrivers:\n")
	for _, name := range sortedKeys(out.Drivers) {
		if out.Drivers[name] != "" {
			fmt.Fprintf(&b, "  %-8s %s\n", name, out.Drivers[name])
		}
	}
	if len(out.Modules) > 0 {
		b.WriteString("\nlemmego modules:\n")
		for _, name := range sortedKeys(out.Modules) {
			fmt.Fprintf(&b, "  %-32s %s\n", name, out.Modules[name])
		}
	}
	return text(b.String()), out, nil
}

// lemmegoModules reads the versions the running binary was built against.
//
// From the build info rather than by parsing go.mod, because go.mod records
// what was requested and the build records what minimal version selection
// actually chose. An agent asking "which version am I coding against" wants
// the second one.
func lemmegoModules() map[string]string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return nil
	}
	modules := make(map[string]string)
	for _, dep := range info.Deps {
		if strings.HasPrefix(dep.Path, "github.com/lemmego/") {
			version := dep.Version
			if dep.Replace != nil {
				// A replace directive means the code is not the tagged code,
				// and silently reporting the tag would be a lie an agent acts
				// on.
				version = fmt.Sprintf("%s (replaced by %s)", version, dep.Replace.Path)
			}
			modules[dep.Path] = version
		}
	}
	return modules
}

// --- config -----------------------------------------------------------------

type configInput struct {
	Key string `json:"key,omitempty" jsonschema:"a dot-separated key such as session.driver; omit for the whole tree"`
}

type configOutput struct {
	Key   string `json:"key,omitempty"`
	Value any    `json:"value"`
}

func (s *Server) addConfig(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name: "config",
		Description: "Read this application's resolved configuration, with secrets withheld. " +
			"Pass a dot-separated key for one value, or omit it for the whole tree. " +
			"Values are post-resolution, so this is what the application actually believes, " +
			"not what .env says.",
	}, s.config)
}

func (s *Server) config(_ context.Context, _ *sdk.CallToolRequest, in configInput) (*sdk.CallToolResult, configOutput, error) {
	cfg := s.app.Config()

	if key := strings.TrimSpace(in.Key); key != "" {
		value := RedactValue(key, cfg.Get(key))
		if nested, ok := value.(map[string]any); ok {
			value = RedactMap(nested)
		}
		out := configOutput{Key: key, Value: value}
		return text(fmt.Sprintf("%s = %v", key, value)), out, nil
	}

	all := RedactMap(cfg.GetAll())
	out := configOutput{Value: all}

	var b strings.Builder
	b.WriteString("configuration (secrets withheld):\n")
	writeTree(&b, all, 1)
	return text(b.String()), out, nil
}

func writeTree(b *strings.Builder, tree map[string]any, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, key := range sortedKeys(tree) {
		switch typed := tree[key].(type) {
		case map[string]any:
			fmt.Fprintf(b, "%s%s:\n", indent, key)
			writeTree(b, typed, depth+1)
		default:
			fmt.Fprintf(b, "%s%s: %v\n", indent, key, typed)
		}
	}
}

// --- absolute-url -----------------------------------------------------------

type absoluteURLInput struct {
	Path string `json:"path" jsonschema:"a path such as /login or @handle"`
}

type absoluteURLOutput struct {
	URL string `json:"url"`
}

func (s *Server) addAbsoluteURL(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name: "absolute-url",
		Description: "Turn a path into an absolute URL using this application's configured " +
			"APP_URL, so a generated link or a test fixture points somewhere real. " +
			"APP_URL is what the application itself builds redirects and OAuth callbacks from, " +
			"which is not necessarily the host serving the request.",
	}, s.absoluteURL)
}

func (s *Server) absoluteURL(_ context.Context, _ *sdk.CallToolRequest, in absoluteURLInput) (*sdk.CallToolResult, absoluteURLOutput, error) {
	base := strings.TrimRight(asString(s.app.Config().Get("app.url")), "/")
	if base == "" {
		return failure("app.url is not configured, so there is no base to build on"), absoluteURLOutput{}, nil
	}
	path := in.Path
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	out := absoluteURLOutput{URL: base + path}
	return text(out.URL), out, nil
}

// --- helpers ----------------------------------------------------------------

func asString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
