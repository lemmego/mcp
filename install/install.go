// Package install writes the files an editor needs to find a project's MCP
// server, plus the guideline and skill files that give an agent context.
//
// It is a separate package from the server for one reason, and it is not
// tidiness: installing must work in a project that cannot boot. The server
// needs app.App — a booted application with a database connection — while this
// needs only a directory. A project that has just been scaffolded, or whose
// credentials are not filled in yet, is exactly when somebody wants to wire up
// their editor.
//
// This repository has already paid for getting that backwards. `lemmego new`
// used to generate an app key by running the project's own `appkey` command
// through `lemmego run`, which boots the application; the application could not
// boot without database credentials, so the step printed "Generating app key"
// and then panicked, and every new project started life with no key at all.
//
// So: standard library only, no api/app, and a dependency test that keeps it
// that way.
package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Agent is an editor or coding tool this can configure.
type Agent string

const (
	ClaudeCode Agent = "claude-code"
	Cursor     Agent = "cursor"
	Codex      Agent = "codex"
	Copilot    Agent = "copilot"
	Windsurf   Agent = "windsurf"
)

// AllAgents is every agent this knows how to configure, in a stable order.
func AllAgents() []Agent {
	return []Agent{ClaudeCode, Cursor, Codex, Copilot, Windsurf}
}

// guidelineFile is the file an agent reads for always-on context.
func (a Agent) guidelineFile() string {
	switch a {
	case ClaudeCode:
		return "CLAUDE.md"
	case Cursor:
		return ".cursor/rules/lemmego.mdc"
	case Codex, Copilot, Windsurf:
		return "AGENTS.md"
	}
	return "AGENTS.md"
}

// mcpConfigFile is where an agent looks for MCP server definitions.
//
// Several agents read the same .mcp.json, which is why the writer merges
// rather than overwrites: two agents configured in turn must not each erase
// the other's servers.
func (a Agent) mcpConfigFile() string {
	switch a {
	case Cursor:
		return ".cursor/mcp.json"
	case Codex:
		return ".codex/mcp.json"
	default:
		return ".mcp.json"
	}
}

// Options configures an installation.
type Options struct {
	// Dir is the project root. Required.
	Dir string

	// Agents to configure. Empty means all of them.
	Agents []Agent

	// Skills, when true, writes one SKILL.md per lemmego module the project
	// depends on.
	Skills bool

	// Guidelines, when true, writes the always-on guideline block.
	Guidelines bool

	// Rules, when true, creates the .ai/rules directory and its index so an
	// agent has somewhere to record a decision from the first session.
	Rules bool
}

// Result reports what an installation did, so a command can print it and a
// test can assert on it.
type Result struct {
	Written []string
	Skipped []string

	// Wiring reports whether the project can actually serve MCP. Writing the
	// configuration and having a server are different things, and only the
	// first is this package's doing.
	Wiring ServerWiring
}

// The markers that bound this tool's section of a shared guideline file.
//
// Everything between them is replaced on reinstall and everything outside is
// preserved — which matters because CLAUDE.md is a file people write in by
// hand, and a tool that overwrote it would lose their work. Copied from
// Laravel Boost, which does the same thing for the same reason.
const (
	BeginMarker = "<!-- BEGIN lemmego-guidelines -->"
	EndMarker   = "<!-- END lemmego-guidelines -->"
)

// Run writes the configuration, and reports what it touched.
func Run(opts Options) (Result, error) {
	var result Result
	if strings.TrimSpace(opts.Dir) == "" {
		return result, fmt.Errorf("install: no project directory given")
	}
	if _, err := os.Stat(opts.Dir); err != nil {
		return result, fmt.Errorf("install: %s: %w", opts.Dir, err)
	}

	agents := opts.Agents
	if len(agents) == 0 {
		agents = AllAgents()
	}

	modules, err := ProjectModules(opts.Dir)
	if err != nil {
		return result, err
	}

	for _, agent := range agents {
		written, err := writeMCPConfig(opts.Dir, agent)
		if err != nil {
			return result, err
		}
		result.Written = append(result.Written, written)

		if opts.Guidelines {
			written, changed, err := writeGuidelines(opts.Dir, agent, modules)
			if err != nil {
				return result, err
			}
			if changed {
				result.Written = append(result.Written, written)
			} else {
				result.Skipped = append(result.Skipped, written)
			}
		}
	}

	if opts.Skills {
		paths, err := writeSkills(opts.Dir, modules)
		if err != nil {
			return result, err
		}
		result.Written = append(result.Written, paths...)
	}

	if opts.Rules {
		paths, err := ensureRules(opts.Dir)
		if err != nil {
			return result, err
		}
		result.Written = append(result.Written, paths...)
	}

	// Several agents share a file — .mcp.json serves Claude Code, Copilot and
	// Windsurf, and AGENTS.md serves three too — so the same path is touched
	// more than once and would otherwise be reported three times, once as
	// written and twice as current. Deduplicated with written winning,
	// because "this changed" is the truth a reader needs.
	result.Wiring = DetectServerWiring(opts.Dir)
	result.Written = unique(result.Written)
	result.Skipped = without(unique(result.Skipped), result.Written)
	sort.Strings(result.Written)
	sort.Strings(result.Skipped)
	return result, nil
}

func unique(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, item := range in {
		if seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func without(in, exclude []string) []string {
	excluded := make(map[string]bool, len(exclude))
	for _, item := range exclude {
		excluded[item] = true
	}
	out := make([]string, 0, len(in))
	for _, item := range in {
		if !excluded[item] {
			out = append(out, item)
		}
	}
	return out
}

// serverDefinition is how an editor is told to start the server.
//
// Through `lemmego run` rather than a built binary, because the server has to
// be the application: it reports on the routes, schema and configuration this
// project actually has, which means compiling this project's code.
func serverDefinition() map[string]any {
	return map[string]any{
		"command": "lemmego",
		"args":    []string{"run", "mcp:serve"},
	}
}

// writeMCPConfig adds this project's server to an agent's MCP configuration,
// merging rather than replacing.
func writeMCPConfig(dir string, agent Agent) (string, error) {
	rel := agent.mcpConfigFile()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return rel, err
	}

	config := map[string]any{}
	if existing, err := os.ReadFile(path); err == nil {
		// A file that exists and cannot be parsed is left alone rather than
		// replaced: it is somebody's configuration, and silently discarding
		// it to fix a syntax error is not this tool's call.
		if err := json.Unmarshal(existing, &config); err != nil {
			return rel, fmt.Errorf("install: %s is not valid JSON, so it was not changed: %w", rel, err)
		}
	}

	servers, _ := config["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	servers["lemmego"] = serverDefinition()
	config["mcpServers"] = servers

	body, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return rel, err
	}
	return rel, os.WriteFile(path, append(body, '\n'), 0o644)
}

// writeGuidelines replaces this tool's block in an agent's guideline file,
// leaving everything else intact. It reports whether the file changed.
func writeGuidelines(dir string, agent Agent, modules []string) (string, bool, error) {
	rel := agent.guidelineFile()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return rel, false, err
	}

	block := BeginMarker + "\n" + Guidelines(modules) + "\n" + EndMarker

	existing, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return rel, false, err
		}
		return rel, true, os.WriteFile(path, []byte(block+"\n"), 0o644)
	}

	updated, changed := replaceBlock(string(existing), block)
	if !changed {
		return rel, false, nil
	}
	return rel, true, os.WriteFile(path, []byte(updated), 0o644)
}

// replaceBlock swaps the marked section for a new one, or appends it when
// there is none. It reports whether anything changed, so a reinstall that
// changes nothing can say so instead of touching every file.
func replaceBlock(body, block string) (string, bool) {
	start := strings.Index(body, BeginMarker)
	end := strings.Index(body, EndMarker)

	if start >= 0 && end > start {
		replaced := body[:start] + block + body[end+len(EndMarker):]
		return replaced, replaced != body
	}
	// An unterminated begin marker means a half-written file; appending a
	// second block would nest them. Treated as absent and appended, which at
	// worst leaves a stale fragment a person can see and delete.
	trimmed := strings.TrimRight(body, "\n")
	if trimmed == "" {
		return block + "\n", true
	}
	return trimmed + "\n\n" + block + "\n", true
}

// ensureRules creates the rules directory and a seed index.
//
// The index is created empty rather than left absent, because an agent is
// told to consult it before planning and a missing file reads as "there are
// no rules" only if something says so.
func ensureRules(dir string) ([]string, error) {
	rulesDir := filepath.Join(dir, ".ai", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		return nil, err
	}
	indexPath := filepath.Join(rulesDir, "index.md")
	if _, err := os.Stat(indexPath); err == nil {
		return nil, nil // never overwrite recorded rules
	}
	body := "# Project Rules Index\n\n" +
		"Before planning or editing, find the row whose glob matches the file's path\n" +
		"and read that rule file.\n\n" +
		"| Applies to | Rule file |\n| --- | --- |\n"
	if err := os.WriteFile(indexPath, []byte(body), 0o644); err != nil {
		return nil, err
	}
	return []string{filepath.Join(".ai", "rules", "index.md")}, nil
}
