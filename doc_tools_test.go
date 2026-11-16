package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSite lays out a documentation checkout the way the real one is shaped:
// a section directory with an _index.md, a nested page, and front matter and
// shortcodes on both.
func writeSite(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	pages := map[string]string{
		"observe/_index.md": `---
title: Request Debugging
weight: 87
---

Records what your application did while answering one request.

## Record your queries

Queries are collected below your ORM, at the driver layer.

{{< callout type="warning" >}}
Don't skip the rebuild.
{{< /callout >}}
`,
		"frontend/inertia.md": `---
title: Inertia
---

## Reading a flash message in the browser

A flash message is not read from the page.
`,
		"_index.md": "---\ntitle: Docs\n---\n\nLanding page, deliberately not searched.\n",
	}
	for rel, body := range pages {
		path := filepath.Join(root, "content", "docs", rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func search(t *testing.T, s *Server, in searchDocsInput) searchDocsOutput {
	t.Helper()
	_, out, err := s.searchDocs(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("searchDocs: %v", err)
	}
	return out
}

func TestSearchDocsReadsTheDocumentationSite(t *testing.T) {
	s := &Server{docsRoots: []string{writeSite(t)}}

	out := search(t, s, searchDocsInput{Query: "driver layer"})
	if len(out.Hits) != 1 {
		t.Fatalf("want 1 hit, got %d: %+v", len(out.Hits), out.Hits)
	}
	hit := out.Hits[0]
	if hit.Page != "observe" {
		t.Errorf("page = %q, want %q — a section's _index.md is named for the section", hit.Page, "observe")
	}
	if hit.Title != "Request Debugging" {
		t.Errorf("title = %q, want it read from front matter", hit.Title)
	}
	if hit.Heading != "Record your queries" {
		t.Errorf("heading = %q", hit.Heading)
	}
	if strings.Contains(hit.Excerpt, "---") || strings.Contains(hit.Excerpt, "weight:") {
		t.Errorf("front matter leaked into the excerpt: %q", hit.Excerpt)
	}
}

// A nested page is reported by the path a reader would see.
func TestSearchDocsNamesANestedPageByItsPath(t *testing.T) {
	s := &Server{docsRoots: []string{writeSite(t)}}

	out := search(t, s, searchDocsInput{Query: "flash message"})
	if len(out.Hits) != 1 {
		t.Fatalf("want 1 hit, got %d", len(out.Hits))
	}
	if got := out.Hits[0].Page; got != "frontend/inertia" {
		t.Errorf("page = %q, want %q", got, "frontend/inertia")
	}
}

// The shortcode delimiters are not content. Left in, a search for "callout"
// matches every warning on the site, which is the opposite of useful.
func TestSearchDocsDoesNotMatchShortcodeDelimiters(t *testing.T) {
	s := &Server{docsRoots: []string{writeSite(t)}}

	if out := search(t, s, searchDocsInput{Query: "callout"}); len(out.Hits) != 0 {
		t.Errorf("want no hits for a shortcode name, got %d: %+v", len(out.Hits), out.Hits)
	}
	// but the text it wrapped is still searchable
	if out := search(t, s, searchDocsInput{Query: "skip the rebuild"}); len(out.Hits) != 1 {
		t.Errorf("want the callout's text to remain searchable, got %d hits", len(out.Hits))
	}
}

func TestSearchDocsRestrictsToASection(t *testing.T) {
	s := &Server{docsRoots: []string{writeSite(t)}}

	out := search(t, s, searchDocsInput{Query: "a", Section: "frontend"})
	for _, hit := range out.Hits {
		if !strings.HasPrefix(hit.Page, "frontend") {
			t.Errorf("section filter leaked %q", hit.Page)
		}
	}
	if len(out.Hits) == 0 {
		t.Error("the section filter matched nothing at all, so it asserts nothing")
	}
}

// Every term has to appear, so a second word narrows the search.
func TestSearchDocsRequiresEveryTerm(t *testing.T) {
	s := &Server{docsRoots: []string{writeSite(t)}}

	if out := search(t, s, searchDocsInput{Query: "queries driver"}); len(out.Hits) != 1 {
		t.Errorf("both terms are in one section: want 1 hit, got %d", len(out.Hits))
	}
	if out := search(t, s, searchDocsInput{Query: "queries inertia"}); len(out.Hits) != 0 {
		t.Errorf("terms from different sections should not match: got %d", len(out.Hits))
	}
}

// With nothing to search the tool says so, rather than reporting an empty
// result that reads as "the documentation does not mention it".
func TestSearchDocsSaysWhenThereIsNothingToSearch(t *testing.T) {
	s := &Server{docsRoots: []string{t.TempDir()}}

	result, out, err := s.searchDocs(context.Background(), nil, searchDocsInput{Query: "anything"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Hits) != 0 {
		t.Fatalf("want no hits, got %d", len(out.Hits))
	}
	if result == nil || !result.IsError {
		t.Error("want an error result naming the missing documentation")
	}
}
