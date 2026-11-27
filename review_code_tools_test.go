package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewCodeFindsLayerAndHandlerViolations(t *testing.T) {
	root := t.TempDir()
	write := func(rel, source string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("internal/handlers/update_client_report_status.go", `package handlers

func UpdateClientReportStatusHandler() error {
	return nil
}
`)
	write("internal/handlers/mixed_handlers.go", `package handlers

func PostIndexHandler() error { return nil }
func PostExportInvokeHandler() error { return nil }
func helper() {}
`)
	write("internal/models/post.go", "package models\n\nfunc (p Post) TooLong() {\n"+strings.Repeat("\t_ = 1\n", 20)+"}\n")
	write("internal/handlers/scratch.txt", "temporary")
	write("internal/custom/package.go", "package custom")

	_, out, err := (&Server{}).reviewCode(context.Background(), nil, reviewCodeInput{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	rules := map[string]bool{}
	for _, finding := range out.Findings {
		rules[finding.Rule] = true
	}
	for _, rule := range []string{"layer-file-name", "handler-kind", "handler-method", "function-length", "layer-file", "internal-directory"} {
		if !rules[rule] {
			t.Errorf("missing %s finding: %+v", rule, out.Findings)
		}
	}
}

func TestReviewCodeAcceptsResourcefulAndSingleInvokeFiles(t *testing.T) {
	root := t.TempDir()
	for rel, source := range map[string]string{
		"internal/handlers/post_handlers.go": `package handlers
func PostIndexHandler() error { return nil }
func PostStoreHandler() error { return nil }
`,
		"internal/handlers/post_export_handler.go": `package handlers
func PostExportInvokeHandler() error { return nil }
`,
		"internal/models/post.go": "package models\ntype Post struct{}\n",
	} {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, out, err := (&Server{}).reviewCode(context.Background(), nil, reviewCodeInput{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Findings) != 0 {
		t.Fatalf("valid conventions produced findings: %+v", out.Findings)
	}
}
