// Command syncdocs refreshes the vendored documentation snapshot from a
// checkout of the documentation site.
//
// The snapshot exists because the site is a separate repository, so the server
// cannot embed it directly — and without a snapshot `search-docs` answered
// nothing at all for anybody whose project did not happen to sit beside a
// checkout, which was every real user.
//
//	go run ./corpus/syncdocs ../lemmego.github.io
//
// Run it after changing the documentation. TestCorpusMatchesTheSite fails in a
// tree that holds both repositories when it has not been run, because a
// snapshot nobody is forced to refresh is a snapshot that silently goes stale.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lemmego/mcp/corpus/internal/docsync"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: syncdocs <path to lemmego.github.io checkout>")
		os.Exit(2)
	}
	src, err := docsync.ResolveDocsDir(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	dst := filepath.Join("corpus", "docs")
	n, err := docsync.Sync(src, dst)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := docsync.WriteProvenance(filepath.Join("corpus", "snapshot.txt"), src, n); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("synced %d pages from %s into %s\n", n, src, dst)
}
