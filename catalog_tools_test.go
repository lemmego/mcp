package mcp

import (
	"context"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func listModules(t *testing.T, in listModulesInput) (string, listModulesOutput) {
	t.Helper()
	s := &Server{}
	result, out, err := s.listModules(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("listModules: %v", err)
	}
	if result == nil {
		t.Fatal("no result")
	}
	var body strings.Builder
	for _, content := range result.Content {
		if text, ok := content.(*sdk.TextContent); ok {
			body.WriteString(text.Text)
		}
	}
	return body.String(), out
}

// TestListModulesFindsWhatTheProjectDoesNotHave is the point of the tool.
//
// module-api cannot answer this: it runs `go doc` inside the project, so a
// module that is not in go.mod comes back as "no required module provides
// package …", which an agent reads as "that does not exist" and then builds
// from scratch.
func TestListModulesFindsWhatTheProjectDoesNotHave(t *testing.T) {
	_, out := listModules(t, listModulesInput{Query: "tenancy"})
	if len(out.Modules) == 0 {
		t.Fatal("searching for tenancy found nothing")
	}
	var found bool
	for _, row := range out.Modules {
		if row.Name == "tenancy" {
			found = true
			if row.Path != "github.com/lemmego/tenancy" {
				t.Errorf("path = %q", row.Path)
			}
			if row.Summary == "" {
				t.Error("no summary, which is the only reason to read the row")
			}
		}
	}
	if !found {
		t.Error("tenancy is not in the answer")
	}
}

// Every module is reachable, not just the ones with a documentation page.
func TestListModulesCoversModulesWithNoDocsPage(t *testing.T) {
	body, out := listModules(t, listModulesInput{})
	if len(out.Modules) < 30 {
		t.Fatalf("only %d modules listed", len(out.Modules))
	}
	for _, name := range []string{"tenancy", "rbac", "modules"} {
		if !strings.Contains(body, "**"+name+"**") {
			t.Errorf("%s is missing from the rendered catalogue", name)
		}
	}
	// A module with no page must say so rather than send the agent to a
	// section that does not exist.
	if !strings.Contains(body, "no page on the site yet") {
		t.Error("a module without a documentation page does not say so")
	}
}

// The answer has to say what to do next, because knowing a module exists is
// not the same as being able to use it.
func TestListModulesExplainsHowToAddOne(t *testing.T) {
	body, _ := listModules(t, listModulesInput{Query: "rbac"})
	if !strings.Contains(body, "go get github.com/lemmego/rbac") {
		t.Error("no install line")
	}
	if !strings.Contains(body, "LoadProviders") {
		t.Error("nothing about registering the provider, which is the step that " +
			"makes an installed module actually run")
	}
}

func TestListModulesRejectsAnUnknownCategory(t *testing.T) {
	s := &Server{}
	result, _, err := s.listModules(context.Background(), nil, listModulesInput{Category: "nonsense"})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || !result.IsError {
		t.Error("want an error result naming the real categories")
	}
}

// Shows the rendered catalogue once, so a reviewer can judge what the agent
// is actually handed rather than trusting the assertions above.
func TestListModulesRendering(t *testing.T) {
	body, _ := listModules(t, listModulesInput{Query: "multi-tenancy"})
	t.Log("\n" + body)
}
