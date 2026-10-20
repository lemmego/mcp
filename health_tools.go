package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/lemmego/api/app"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type healthOutput struct {
	Status string                     `json:"status"`
	Checks map[string]app.CheckResult `json:"checks"`
}

func (s *Server) addHealth(server *sdk.Server) {
	sdk.AddTool(server, &sdk.Tool{
		Name: "health",
		Description: "Run this application's registered health checks and report each result. " +
			"Critical checks are what readiness depends on; non-critical ones are where a " +
			"survivable degradation shows up, such as a server-side renderer that has fallen " +
			"back. An empty check list means nothing was registered, so a 'healthy' answer " +
			"would have verified nothing.",
	}, s.health)
}

func (s *Server) health(ctx context.Context, _ *sdk.CallToolRequest, _ emptyInput) (*sdk.CallToolResult, healthOutput, error) {
	checker := s.app.GetHealthChecker()
	if checker == nil {
		return failure("this application has no health checker"), healthOutput{}, nil
	}

	status := checker.Check(ctx)
	if status == nil {
		return failure("the health checker returned nothing"), healthOutput{}, nil
	}
	out := healthOutput{Status: status.Status, Checks: status.Checks}

	var b strings.Builder
	fmt.Fprintf(&b, "overall: %s\n", status.Status)
	if len(status.Checks) == 0 {
		b.WriteString("\nNo checks are registered, so this result verified nothing. " +
			"Register them with checker.RegisterDefaultChecks() and a database check.\n")
		return text(b.String()), out, nil
	}
	b.WriteByte('\n')
	for _, name := range sortedKeys(status.Checks) {
		check := status.Checks[name]
		critical := ""
		if check.Critical {
			critical = " (critical)"
		}
		fmt.Fprintf(&b, "  %-12s %-5s %4dms%s", name, check.Status, check.DurationMs, critical)
		if check.Error != "" {
			fmt.Fprintf(&b, "  %s", check.Error)
		}
		b.WriteByte('\n')
	}
	return text(b.String()), out, nil
}
