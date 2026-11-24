// Package docsync copies the documentation site's pages into the snapshot the
// server embeds, and reads either side of that copy so a test can compare
// them.
//
// Internal because it exists for one dev-time command and one gate; nothing a
// consumer of the server should see.
package docsync

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ResolveDocsDir accepts the site root, its content directory or the docs
// directory itself, because all three are things somebody will reasonably
// point at.
func ResolveDocsDir(root string) (string, error) {
	for _, suffix := range []string{filepath.Join("content", "docs"), "docs", ""} {
		dir := filepath.Clean(filepath.Join(root, suffix))
		if entries, err := os.ReadDir(dir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
					return dir, nil
				}
			}
		}
	}
	return "", fmt.Errorf("%s holds no documentation pages", root)
}

// Sync replaces dst with every markdown page under src, and reports how many
// it wrote.
//
// Markdown only. The site also carries images, layouts and a Hugo
// configuration, none of which the server reads, and embedding them would put
// the whole theme into every application's binary.
func Sync(src, dst string) (int, error) {
	pages, err := Pages(src)
	if err != nil {
		return 0, err
	}
	if len(pages) == 0 {
		return 0, fmt.Errorf("%s holds no documentation pages", src)
	}

	// Removed rather than merged, so a page deleted from the site is deleted
	// here too. Merging would leave the agent reading a page a reader cannot.
	if err := os.RemoveAll(dst); err != nil {
		return 0, err
	}

	for rel, body := range pages {
		path := filepath.Join(dst, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return 0, err
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			return 0, err
		}
	}
	return len(pages), nil
}

// Pages reads every markdown page under dir, keyed by its slash-separated
// path relative to dir.
func Pages(dir string) (map[string][]byte, error) {
	pages := map[string][]byte{}
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			return err
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		pages[filepath.ToSlash(rel)] = body
		return nil
	})
	if err != nil {
		return nil, err
	}
	return pages, nil
}

// WriteProvenance records where the snapshot came from and when.
//
// Generated rather than hand-maintained, which is the whole point. A version
// string somebody has to remember to bump is a string that does not get
// bumped: the scaffold shipped a fix to nobody that way, because the only
// thing gating delivery was a number a human had to edit. This file cannot go
// stale independently of the pages beside it, because the same command writes
// both.
func WriteProvenance(path, src string, pages int) error {
	commit := "unknown"
	cmd := exec.Command("git", "-C", src, "rev-parse", "--short", "HEAD")
	if out, err := cmd.Output(); err == nil {
		commit = strings.TrimSpace(string(out))
	}
	body := fmt.Sprintf("pages: %d\ncommit: %s\ntaken: %s\n",
		pages, commit, time.Now().UTC().Format(time.RFC3339))
	return os.WriteFile(path, []byte(body), 0o644)
}
