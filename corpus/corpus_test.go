package corpus

import (
	"io/fs"
	"strings"
	"testing"
)

// TestEmbedKeepsSectionIndexPages guards the `all:` prefix on the go:embed
// directive.
//
// Without it, embed omits every path element beginning with "_" — and a Hugo
// section's own page is named _index.md, so a plain `//go:embed docs` loses
// the landing page of every section while still compiling and still answering
// about ordinary pages. The failure is invisible except as "search-docs never
// finds the observe overview".
func TestEmbedKeepsSectionIndexPages(t *testing.T) {
	var indexes []string
	err := fs.WalkDir(Docs(), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && entry.Name() == "_index.md" {
			indexes = append(indexes, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the snapshot: %v", err)
	}
	if len(indexes) == 0 {
		t.Fatal("the snapshot holds no _index.md pages, so go:embed dropped them: " +
			"the directive needs the all: prefix")
	}
	// A spot check that the content came through, not just the name.
	body, err := fs.ReadFile(Docs(), "observe/_index.md")
	if err != nil {
		t.Fatalf("reading a section index page: %v", err)
	}
	if !strings.Contains(string(body), "---") {
		t.Error("a section index page came through without its front matter")
	}
}

// TestSnapshotHoldsTheWholeSite refuses an empty or truncated snapshot, since
// an empty one would turn search-docs back into the tool that answers nothing
// and would do it without any build failure.
func TestSnapshotHoldsTheWholeSite(t *testing.T) {
	if n := Pages(); n < 50 {
		t.Fatalf("snapshot holds %d pages, which is too few to be the site; "+
			"run: go run ./corpus/syncdocs ../lemmego.github.io", n)
	}
}

// TestSnapshotKeepsCodeFences is why the snapshot is markdown rather than the
// site's own search index.
//
// Hextra publishes en.search-data.json, which is one request for the whole
// site and looks like a tempting corpus — but it strips code fences and
// collapses the newlines inside a code block, so
//
//	page.Items       // []Post
//	page.Total       // rows in the whole result set
//
// arrives as "page.Items // []Post page.Total // …" and everything after the
// first // is commented out. Handing that to an agent produces code that does
// not compile, from the one tool whose purpose is preventing exactly that.
func TestSnapshotKeepsCodeFences(t *testing.T) {
	body, err := fs.ReadFile(Docs(), "orm/queries.md")
	if err != nil {
		t.Fatalf("reading a page with code examples: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "```go") {
		t.Error("code fences are missing, so examples arrive as prose")
	}
	if !strings.Contains(text, "page.Items") || !strings.Contains(text, "\npage.Total") {
		t.Error("a multi-line code example lost its line structure; the whole " +
			"reason this snapshot is markdown is that the site's search index does that")
	}
}
