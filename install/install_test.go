package install_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lemmego/mcp/install"
)

func project(t *testing.T, gomod string) string {
	t.Helper()
	dir := t.TempDir()
	if gomod != "" {
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const exampleGoMod = `module example.com/app

go 1.27

require (
	github.com/lemmego/api v0.7.0
	github.com/lemmego/apptest v0.2.5
	github.com/lemmego/inertia v0.2.4
	github.com/lemmego/migration v0.5.1
	github.com/spf13/cobra v1.8.1
)
`

func TestRunWritesAnMCPConfigAnEditorCanRead(t *testing.T) {
	dir := project(t, exampleGoMod)

	if _, err := install.Run(install.Options{Dir: dir, Agents: []install.Agent{install.ClaudeCode}}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	body, err := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if err != nil {
		t.Fatalf("reading .mcp.json: %v", err)
	}
	var config struct {
		MCPServers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(body, &config); err != nil {
		t.Fatalf("the written config is not valid JSON: %v\n%s", err, body)
	}
	server, ok := config.MCPServers["lemmego"]
	if !ok {
		t.Fatalf("no lemmego server in the config: %s", body)
	}
	if server.Command != "lemmego" || strings.Join(server.Args, " ") != "run mcp:serve" {
		t.Errorf("server definition = %s %v, want lemmego run mcp:serve", server.Command, server.Args)
	}
}

// Several agents share .mcp.json, so writing one must not erase another's
// servers — including servers this tool knows nothing about.
func TestRunMergesIntoAnExistingMCPConfig(t *testing.T) {
	dir := project(t, exampleGoMod)
	existing := `{"mcpServers":{"someone-elses":{"command":"other","args":["serve"]}}}`
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := install.Run(install.Options{Dir: dir, Agents: []install.Agent{install.ClaudeCode}}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	body, _ := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	var config map[string]map[string]any
	if err := json.Unmarshal(body, &config); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if _, kept := config["mcpServers"]["someone-elses"]; !kept {
		t.Errorf("the existing server was erased: %s", body)
	}
	if _, added := config["mcpServers"]["lemmego"]; !added {
		t.Errorf("the lemmego server was not added: %s", body)
	}
}

// A config file that exists and cannot be parsed is somebody's work. Replacing
// it to fix a syntax error is not this tool's call.
func TestRunRefusesToOverwriteUnparsableConfig(t *testing.T) {
	dir := project(t, exampleGoMod)
	broken := `{"mcpServers": {`
	path := filepath.Join(dir, ".mcp.json")
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := install.Run(install.Options{Dir: dir, Agents: []install.Agent{install.ClaudeCode}}); err == nil {
		t.Fatal("Run() overwrote an unparsable config instead of refusing")
	}
	after, _ := os.ReadFile(path)
	if string(after) != broken {
		t.Errorf("the unparsable file was modified: %s", after)
	}
}

// CLAUDE.md is a file people write in by hand. A tool that overwrote it would
// lose their work, so only the marked block is replaced.
func TestGuidelinesPreserveEverythingOutsideTheBlock(t *testing.T) {
	dir := project(t, exampleGoMod)
	path := filepath.Join(dir, "CLAUDE.md")
	handWritten := "# My project\n\nSomething I wrote myself.\n"
	if err := os.WriteFile(path, []byte(handWritten), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := install.Options{Dir: dir, Agents: []install.Agent{install.ClaudeCode}, Guidelines: true}
	if _, err := install.Run(opts); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	first, _ := os.ReadFile(path)
	if !strings.Contains(string(first), "Something I wrote myself.") {
		t.Fatalf("the hand-written content was lost: %s", first)
	}
	if !strings.Contains(string(first), install.BeginMarker) {
		t.Fatalf("no guideline block was written: %s", first)
	}

	// Reinstalling must replace the block rather than appending a second one.
	if _, err := install.Run(opts); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}
	second, _ := os.ReadFile(path)
	if n := strings.Count(string(second), install.BeginMarker); n != 1 {
		t.Errorf("reinstall produced %d guideline blocks, want 1", n)
	}
	if !strings.Contains(string(second), "Something I wrote myself.") {
		t.Errorf("reinstall lost the hand-written content: %s", second)
	}
}

// The guidelines are what an agent reads before it knows anything, so the two
// tools that prevent invented APIs have to be in them.
func TestGuidelinesLeadWithTheAPITools(t *testing.T) {
	body := install.Guidelines([]string{"api", "orm"})
	for _, want := range []string{"module-api", "search-docs", "list-routes", "record-rule"} {
		if !strings.Contains(body, want) {
			t.Errorf("the guidelines do not mention %q", want)
		}
	}
	if !strings.Contains(body, "api, orm") {
		t.Errorf("the guidelines do not list the project's modules: %s", body)
	}
}

// A skill is written only when the project uses its module, so a project
// without payments carries no payment skill.
func TestSkillsFollowTheProjectsModules(t *testing.T) {
	dir := project(t, exampleGoMod)

	if _, err := install.Run(install.Options{
		Dir:    dir,
		Agents: []install.Agent{install.ClaudeCode},
		Skills: true,
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// The example go.mod has migration, apptest and inertia.
	for _, name := range []string{"lemmego-migrations", "lemmego-testing", "lemmego-inertia"} {
		path := filepath.Join(dir, ".ai", "skills", name, "SKILL.md")
		body, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("expected skill %s: %v", name, err)
			continue
		}
		// The Agent Skills format requires name and description frontmatter.
		if !strings.HasPrefix(string(body), "---\nname: "+name+"\n") {
			t.Errorf("%s has no conforming frontmatter:\n%s", name, firstLines(string(body), 4))
		}
		if !strings.Contains(string(body), "description: ") {
			t.Errorf("%s has no description in its frontmatter", name)
		}
	}
}

func TestSkillsAreNotWrittenForAbsentModules(t *testing.T) {
	dir := project(t, "module example.com/app\n\ngo 1.27\n\nrequire github.com/lemmego/api v0.7.0\n")

	if _, err := install.Run(install.Options{
		Dir:    dir,
		Agents: []install.Agent{install.ClaudeCode},
		Skills: true,
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".ai", "skills", "lemmego-migrations")); !os.IsNotExist(err) {
		t.Error("a migrations skill was written for a project that does not use migration")
	}
}

// An agent is told to consult the index before planning, so a missing file
// reads as "there are no rules" only if something says so.
func TestRulesSeedAnIndex(t *testing.T) {
	dir := project(t, exampleGoMod)

	if _, err := install.Run(install.Options{
		Dir:    dir,
		Agents: []install.Agent{install.ClaudeCode},
		Rules:  true,
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	body, err := os.ReadFile(filepath.Join(dir, ".ai", "rules", "index.md"))
	if err != nil {
		t.Fatalf("no rules index: %v", err)
	}
	if !strings.Contains(string(body), "Applies to") {
		t.Errorf("the index has no table: %s", body)
	}
}

// Rules are recorded decisions. Reinstalling must never discard them.
func TestRulesAreNeverOverwritten(t *testing.T) {
	dir := project(t, exampleGoMod)
	rules := filepath.Join(dir, ".ai", "rules")
	if err := os.MkdirAll(rules, 0o755); err != nil {
		t.Fatal(err)
	}
	recorded := "# Project Rules Index\n\n| Applies to | Rule file |\n| --- | --- |\n| internal/** | .ai/rules/internal.md |\n"
	if err := os.WriteFile(filepath.Join(rules, "index.md"), []byte(recorded), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := install.Run(install.Options{
		Dir:    dir,
		Agents: []install.Agent{install.ClaudeCode},
		Rules:  true,
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	after, _ := os.ReadFile(filepath.Join(rules, "index.md"))
	if string(after) != recorded {
		t.Errorf("the recorded index was overwritten:\n%s", after)
	}
}

func TestRunRefusesAMissingDirectory(t *testing.T) {
	if _, err := install.Run(install.Options{Dir: filepath.Join(t.TempDir(), "nope")}); err == nil {
		t.Error("Run() accepted a directory that does not exist")
	}
}

// Every agent gets configured, each in the file it actually reads.
func TestRunConfiguresEveryAgentInItsOwnFile(t *testing.T) {
	dir := project(t, exampleGoMod)

	if _, err := install.Run(install.Options{Dir: dir, Guidelines: true}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, path := range []string{
		".mcp.json",
		"CLAUDE.md",
		"AGENTS.md",
		filepath.Join(".cursor", "mcp.json"),
		filepath.Join(".codex", "mcp.json"),
	} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Errorf("expected %s: %v", path, err)
		}
	}
}

func firstLines(s string, n int) string {
	lines := strings.SplitN(s, "\n", n+1)
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

// Several agents share a file — .mcp.json serves three of them and AGENTS.md
// serves three — so a path must be reported once, and never as both written
// and unchanged.
func TestResultReportsEachPathOnce(t *testing.T) {
	dir := project(t, exampleGoMod)

	result, err := install.Run(install.Options{Dir: dir, Guidelines: true})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	seen := map[string]int{}
	for _, path := range append(append([]string{}, result.Written...), result.Skipped...) {
		seen[path]++
	}
	for path, count := range seen {
		if count > 1 {
			t.Errorf("%s reported %d times, want once", path, count)
		}
	}
	for _, written := range result.Written {
		for _, skipped := range result.Skipped {
			if written == skipped {
				t.Errorf("%s reported as both written and unchanged", written)
			}
		}
	}
}
