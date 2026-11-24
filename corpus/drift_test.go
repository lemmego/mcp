package corpus

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/lemmego/mcp/corpus/internal/docsync"
)

// siteCheckout is where the documentation repository sits relative to this
// package when both are checked out together.
const siteCheckout = "../../lemmego.github.io"

// TestCorpusMatchesTheSite fails when the snapshot and the documentation
// checkout disagree.
//
// This is the gate, and where it runs is the point. Every module here is its
// own repository, so the mcp repository's own CI has no checkout of the
// documentation and this test skips there — correctly, because nothing in
// that repository can change the documentation. The drift is created in the
// combined working tree, where somebody edits a page; that is the one place
// both trees exist, and it is where this fires.
//
// Without it the snapshot is exactly the trap the scaffold's VERSION file
// was: a copy that silently stops matching, where the fix is already written
// and simply reaches nobody.
func TestCorpusMatchesTheSite(t *testing.T) {
	src, err := docsync.ResolveDocsDir(siteCheckout)
	if err != nil {
		t.Skipf("no documentation checkout beside this module (%s): nothing to compare", siteCheckout)
	}

	live, err := docsync.Pages(src)
	if err != nil {
		t.Fatalf("reading the checkout: %v", err)
	}
	snapshot, err := docsync.Pages(filepath.Join("docs"))
	if err != nil {
		t.Fatalf("reading the snapshot: %v", err)
	}

	var added, removed, changed []string
	for page, body := range live {
		have, ok := snapshot[page]
		switch {
		case !ok:
			added = append(added, page)
		case string(have) != string(body):
			changed = append(changed, page)
		}
	}
	for page := range snapshot {
		if _, ok := live[page]; !ok {
			removed = append(removed, page)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	sort.Strings(changed)

	if len(added)+len(removed)+len(changed) == 0 {
		return
	}

	var b strings.Builder
	b.WriteString("the embedded documentation snapshot no longer matches the site.\n")
	for label, pages := range map[string][]string{
		"only on the site":     added,
		"only in the snapshot": removed,
		"different in each":    changed,
	} {
		if len(pages) == 0 {
			continue
		}
		b.WriteString("\n  " + label + ":\n")
		for _, page := range pages {
			b.WriteString("    " + page + "\n")
		}
	}
	b.WriteString("\nrefresh it with:\n    go run ./corpus/syncdocs " + siteCheckout + "\n")
	t.Fatal(b.String())
}

// TestProvenanceIsReadable keeps the generated note parseable, since a tool
// reports it to the agent.
func TestProvenanceIsReadable(t *testing.T) {
	if _, err := os.Stat("snapshot.txt"); err != nil {
		t.Fatalf("no snapshot.txt: run go run ./corpus/syncdocs %s", siteCheckout)
	}
	got := Provenance()
	for _, want := range []string{"pages", "docs@", "taken"} {
		if !strings.Contains(got, want) {
			t.Errorf("provenance %q is missing %q", got, want)
		}
	}
}
