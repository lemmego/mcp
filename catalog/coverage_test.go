package catalog_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/lemmego/mcp/catalog"
	"github.com/lemmego/mcp/install"
)

// exemptFromASkill are the catalogued modules that deliberately have no skill
// of their own, each with the reason.
//
// A map rather than a list, because "this one is covered elsewhere" is a claim
// worth writing down. An exemption with no reason is how a gap becomes
// permanent.
var exemptFromASkill = map[string]string{
	"api": "the framework core itself — covered by lemmego-conventions and by " +
		"most of the documentation site",
	"cli": "covered by lemmego-conventions, which is where the generators and " +
		"the project layout belong",
	"mcp": "this server. An agent reads from it rather than writing against it",
	"tasker": "covered by lemmego-queue. An application uses queue; tasker is " +
		"the engine underneath and is not what you call",

	// The GPA family. One skill covers the registry, the repositories and the
	// error contract; a skill per adapter would repeat it five times and the
	// differences that matter are already named in it.
	"gpaorm":   "covered by lemmego-gpa",
	"gpagorm":  "covered by lemmego-gpa",
	"gpabun":   "covered by lemmego-gpa",
	"gpamongo": "covered by lemmego-gpa",
	"gparedis": "covered by lemmego-gpa",

	// The connectors differ only in which ORM they open.
	"ormconnector":  "covered by lemmego-gpa, which explains the connector seam",
	"gormconnector": "covered by lemmego-gpa, which explains the connector seam",
	"bunconnector":  "covered by lemmego-gpa, which explains the connector seam",
}

// TestEveryModuleHasASkillOrAReason is the answer to "do we have a skill for
// that?", kept true rather than asked again.
//
// The skill library started at three for thirty-eight modules — migrations,
// testing and Inertia — so an agent writing a notification, a rate limiter or
// an image upload had the documentation and nothing else. Adding a module
// without a skill is a quiet return to that, because nothing fails.
func TestEveryModuleHasASkillOrAReason(t *testing.T) {
	covered := map[string]bool{}
	for _, module := range install.SkillModules() {
		covered[module] = true
	}

	var uncovered []string
	for _, module := range catalog.All {
		if covered[module.Name] {
			continue
		}
		if _, exempt := exemptFromASkill[module.Name]; exempt {
			continue
		}
		uncovered = append(uncovered, module.Name)
	}
	sort.Strings(uncovered)

	if len(uncovered) > 0 {
		t.Errorf("%d published module(s) have no skill and no stated reason: %s\n\n"+
			"Either add install/skills/lemmego-<name>.md and register it in "+
			"skillTemplates, or add the module to exemptFromASkill with the reason "+
			"it does not need one.",
			len(uncovered), strings.Join(uncovered, ", "))
	}
}

// TestNoStaleExemptions catches an exemption for a module that has since
// gained a skill, or that no longer exists.
func TestNoStaleExemptions(t *testing.T) {
	covered := map[string]bool{}
	for _, module := range install.SkillModules() {
		covered[module] = true
	}

	for name, reason := range exemptFromASkill {
		if strings.TrimSpace(reason) == "" {
			t.Errorf("%s is exempt with no reason", name)
		}
		if _, ok := catalog.ByName(name); !ok {
			t.Errorf("%s is exempt from needing a skill but is not in the catalogue", name)
		}
		if covered[name] {
			t.Errorf("%s is listed as exempt but now has a skill; drop the exemption", name)
		}
	}
}

// TestSkillsNameRealModules catches a skill keyed on a module that does not
// exist — a typo in a key, which would mean the skill is never written.
func TestSkillsNameRealModules(t *testing.T) {
	for _, module := range install.SkillModules() {
		if _, ok := catalog.ByName(module); !ok {
			t.Errorf("a skill is keyed on %q, which is not a published module, "+
				"so no project will ever receive it", module)
		}
	}
}
