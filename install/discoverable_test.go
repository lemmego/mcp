package install_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lemmego/mcp/install"
)

// agentSkillDirs is where each agent actually looks for on-demand guidance.
//
// Written out here rather than asked of the package, so that this is an
// independent statement of the requirement and not a restatement of the
// implementation. If somebody changes skillPath, this has to be changed too,
// deliberately.
var agentSkillDirs = map[install.Agent]string{
	install.ClaudeCode: ".claude/skills",
	install.Cursor:     ".cursor/rules",
	install.Codex:      ".ai/skills",
	install.Copilot:    ".ai/skills",
	install.Windsurf:   ".ai/skills",
}

// TestSkillsLandWhereTheAgentReadsThem is the guard for the bug this fixes.
//
// Every skill used to be written to .ai/skills/<name>/SKILL.md whatever the
// agent, and that path is read by none of them. The files were correct, were
// never loaded, and nothing failed — which is the worst shape a bug can have.
func TestSkillsLandWhereTheAgentReadsThem(t *testing.T) {
	for agent, dir := range agentSkillDirs {
		t.Run(string(agent), func(t *testing.T) {
			root := project(t, exampleGoMod)
			res, err := install.Run(install.Options{
				Dir:    root,
				Agents: []install.Agent{agent},
				Skills: true,
			})
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			var skills []string
			for _, path := range res.Written {
				if isSkill(path) {
					skills = append(skills, path)
				}
			}
			if len(skills) == 0 {
				t.Fatal("no skills written at all")
			}
			for _, path := range skills {
				if !strings.HasPrefix(filepath.ToSlash(path), dir+"/") {
					t.Errorf("skill written to %q, which %s does not read; want it under %q",
						path, agent, dir)
				}
				if _, err := os.Stat(filepath.Join(root, path)); err != nil {
					t.Errorf("reported %q as written but it is not there: %v", path, err)
				}
			}
		})
	}
}

// TestTheGuidelinesPointAtEverySkill is the other half.
//
// A skill in the right directory is still useless to an agent that has no
// skill mechanism, and the previous guideline block mentioned the skills
// nowhere — so even an agent that could read the file was never told to. The
// always-on block has to name each one and say when to open it.
func TestTheGuidelinesPointAtEverySkill(t *testing.T) {
	for agent := range agentSkillDirs {
		t.Run(string(agent), func(t *testing.T) {
			root := project(t, exampleGoMod)
			res, err := install.Run(install.Options{
				Dir:        root,
				Agents:     []install.Agent{agent},
				Skills:     true,
				Guidelines: true,
			})
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			guidelines := install.Guidelines(agent, []string{"migration", "apptest", "inertia"})

			for _, path := range res.Written {
				slash := filepath.ToSlash(path)
				if !isSkill(path) {
					continue
				}
				if !strings.Contains(guidelines, slash) {
					t.Errorf("the guidelines never mention %q, so nothing tells the "+
						"agent it exists", slash)
				}
			}
		})
	}
}

// TestTheGuidelinesPointAtTheRulesIndex covers the same failure for rules.
//
// ensureRules writes .ai/rules/index.md containing "find the row whose glob
// matches the file's path and read that rule file" — an instruction addressed
// to a reader that was never sent there.
func TestTheGuidelinesPointAtTheRulesIndex(t *testing.T) {
	guidelines := install.Guidelines(install.ClaudeCode, []string{"api"})
	if !strings.Contains(guidelines, ".ai/rules/index.md") {
		t.Error("the guidelines do not point at the rules index, so a recorded " +
			"rule is never read")
	}
}

// TestTheGuidelinesNameTheCatalogue keeps the ecosystem discoverable.
//
// Every other path to finding a module is gated on already depending on it:
// module-api runs `go doc` in the project, and the module list in the
// guidelines is read from go.mod. So a project that does not use tenancy has
// no way to learn tenancy exists unless the always-on block says to ask.
func TestTheGuidelinesNameTheCatalogue(t *testing.T) {
	guidelines := install.Guidelines(install.ClaudeCode, []string{"api"})
	if !strings.Contains(guidelines, "list-modules") {
		t.Error("the guidelines never mention list-modules, so the ecosystem " +
			"stays invisible to any project that does not already depend on it")
	}
}

// isSkill distinguishes a skill file from the guideline file.
//
// They share a directory under Cursor, whose rules are both: the always-on
// block is .cursor/rules/lemmego.mdc and a skill is
// .cursor/rules/lemmego-<topic>.mdc. The dash is the difference, so every
// skill name carries one.
func isSkill(path string) bool {
	slash := filepath.ToSlash(path)
	// Two layouts: a directory holding SKILL.md, or a single .mdc file. The
	// skill's name carries a dash in both, and the guideline file never does.
	for _, element := range strings.Split(slash, "/") {
		if strings.HasPrefix(element, "lemmego-") {
			return true
		}
	}
	return false
}

// TestEverySkillHasABody turns the panic in skillTemplate.body into a test
// failure. The bodies are embedded, so a missing one is a build-time mistake
// that would otherwise surface as a panic during somebody's install.
func TestEverySkillHasABody(t *testing.T) {
	// Every module that keys a skill, so every body is read. exampleGoMod
	// names only three, which would leave most of the library unchecked.
	var b strings.Builder
	b.WriteString("module example.com/app\n\ngo 1.27\n\nrequire (\n")
	for _, module := range install.SkillModules() {
		fmt.Fprintf(&b, "\tgithub.com/lemmego/%s v0.1.0\n", module)
	}
	b.WriteString(")\n")
	root := project(t, b.String())
	res, err := install.Run(install.Options{
		Dir:        root,
		Agents:     []install.Agent{install.ClaudeCode},
		Skills:     true,
		Guidelines: true,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, path := range res.Written {
		if !isSkill(path) {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Errorf("%s: %v", path, err)
			continue
		}
		text := string(body)
		if !strings.Contains(text, "## When to use this skill") {
			t.Errorf("%s has no \"When to use this skill\" section, so an agent "+
				"cannot tell whether it is relevant", path)
		}
		if len(text) < 500 {
			t.Errorf("%s is %d bytes, which is too short to be worth loading", path, len(text))
		}
	}
}

// TestNoOrphanedSkillBodies catches a markdown file nobody references, which
// would be a skill somebody wrote and no project ever receives.
func TestNoOrphanedSkillBodies(t *testing.T) {
	referenced := map[string]bool{}
	for _, name := range install.SkillNames() {
		referenced[name+".md"] = true
	}
	entries, err := os.ReadDir(filepath.Join("skills"))
	if err != nil {
		t.Skipf("no skills directory to check: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		if !referenced[entry.Name()] {
			t.Errorf("skills/%s is not referenced by any skill template, so no "+
				"project will ever receive it", entry.Name())
		}
	}
}
