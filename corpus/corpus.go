// Package corpus holds the documentation site's pages, compiled into the
// binary.
//
// It exists because the documentation is a separate repository, so a project
// that does not happen to sit beside a checkout of it had no documentation at
// all — `search-docs` answered "found no documentation pages to search" for
// every user who was not working inside the monorepo, which is every user.
//
// This snapshot is the fallback, searched after any checkout on disk, so
// editing a page in a working tree still takes effect immediately. What a
// released server carries is the documentation as it stood at that version,
// which is the same principle that makes module-api read the version a
// project pins rather than whatever is newest.
//
// Refresh it with:
//
//	go run ./corpus/syncdocs ../lemmego.github.io
package corpus

import (
	"embed"
	"io/fs"
	"strings"
)

// The `all:` prefix is load-bearing. Without it `embed` omits every file whose
// name begins with "_" or "." — and twenty-nine of these pages are named
// _index.md, because that is how Hugo names a section's own page. A plain
// `//go:embed docs` compiles, passes a smoke test against any other page, and
// silently loses every section landing page. TestEmbedKeepsSectionIndexPages
// is what keeps somebody from "tidying" it away.
//
//go:embed all:docs
var embedded embed.FS

// provenance records which commit of the documentation this snapshot was
// taken from, written by the same command that writes the pages — so it
// cannot drift from them the way a hand-edited version string can.
//
//go:embed snapshot.txt
var provenance string

// Provenance is a one-line description of the snapshot, for a tool that wants
// to say where its answer came from.
func Provenance() string {
	fields := map[string]string{}
	for _, line := range strings.Split(provenance, "\n") {
		if key, value, ok := strings.Cut(line, ":"); ok {
			fields[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	taken := fields["taken"]
	if len(taken) > 10 {
		taken = taken[:10]
	}
	return strings.TrimSpace(strings.Join([]string{
		fields["pages"] + " pages",
		"docs@" + fields["commit"],
		"taken " + taken,
	}, ", "))
}

// Docs is the snapshot rooted so that a page's path is relative to it —
// "observe/_index.md", "logging.md" — matching what a checkout's
// content/docs directory yields.
func Docs() fs.FS {
	sub, err := fs.Sub(embedded, "docs")
	if err != nil {
		// Unreachable: the directory is embedded above, so fs.Sub can only
		// fail on a path that does not parse.
		panic("corpus: " + err.Error())
	}
	return sub
}

// Pages reports how many markdown pages the snapshot holds, so a caller can
// say so and a test can refuse an empty one.
func Pages() int {
	count := 0
	_ = fs.WalkDir(Docs(), ".", func(_ string, entry fs.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			count++
		}
		return nil
	})
	return count
}
