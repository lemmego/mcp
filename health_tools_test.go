package mcp

import (
	"context"
	"testing"

	"github.com/lemmego/api/app"
)

func TestHealthMaterializesRoutesBeforeChecking(t *testing.T) {
	application := app.Configure(app.WithRoutes([]app.RouteCallback{
		func(a app.App) {
			a.GetHealthChecker().RegisterCheck("cache", app.HealthCheck{
				Name:        "cache",
				Description: "cache registered by a route callback",
				Critical:    true,
				CheckFunc:   func(context.Context) error { return nil },
			})
		},
	}))

	_, result, err := New(application).health(context.Background(), nil, emptyInput{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Checks["cache"]; !ok {
		t.Fatalf("health checks = %#v, want the cache check registered by the application routes", result.Checks)
	}
}
