package catalog_test

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/lemmego/mcp/catalog"
)

// workspace is the monorepo's go.work relative to this package.
const workspace = "../../go.work"

// TestCatalogueCoversTheWorkspace compares the catalogue against go.work.
//
// This is the gate. A module that is published and not listed here is a
// module an agent cannot discover, which is the entire problem this package
// exists to solve — so the catalogue going stale reintroduces the bug
// silently. tenancy and rbac were both in that state: real modules with real
// READMEs, absent from the documentation site, absent from the scaffold's
// picker, and therefore invisible.
//
// Skips where go.work is absent, which is this module's own repository.
func TestCatalogueCoversTheWorkspace(t *testing.T) {
	body, err := os.ReadFile(workspace)
	if err != nil {
		t.Skipf("no %s: nothing to compare against", workspace)
	}

	// `use (\n\t./api\n\t./apptest\n…)`
	var inWorkspace []string
	for _, line := range strings.Split(string(body), "\n") {
		if match := regexp.MustCompile(`^\s*\./([a-z0-9]+)\s*$`).FindStringSubmatch(line); match != nil {
			inWorkspace = append(inWorkspace, match[1])
		}
	}
	if len(inWorkspace) == 0 {
		t.Fatalf("parsed no modules out of %s", workspace)
	}

	listed := map[string]bool{}
	for _, module := range catalog.All {
		listed[module.Name] = true
	}

	var missing []string
	for _, name := range inWorkspace {
		if !listed[name] {
			missing = append(missing, name)
		}
	}
	present := map[string]bool{}
	for _, name := range inWorkspace {
		present[name] = true
	}
	var extra []string
	for name := range listed {
		if !present[name] {
			extra = append(extra, name)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)

	if len(missing) > 0 {
		t.Errorf("published but not in the catalogue, so an agent cannot find them: %s",
			strings.Join(missing, ", "))
	}
	if len(extra) > 0 {
		t.Errorf("in the catalogue but not in the workspace, so the catalogue "+
			"advertises something that may not exist: %s", strings.Join(extra, ", "))
	}
}

// TestEveryModuleHasASummary refuses a half-filled entry, since the summary is
// the whole value of the row.
func TestEveryModuleHasASummary(t *testing.T) {
	for _, module := range catalog.All {
		if strings.TrimSpace(module.Summary) == "" {
			t.Errorf("%s has no summary", module.Name)
		}
		if module.Category == "" {
			t.Errorf("%s has no category", module.Name)
		}
	}
}

// TestTheInvisibleModulesAreFindable is the regression test for the complaint
// that started this: a project could not learn that tenancy, rbac, oauth2 or
// i18n existed.
func TestTheInvisibleModulesAreFindable(t *testing.T) {
	for _, name := range []string{"tenancy", "rbac", "oauth2", "i18n"} {
		module, ok := catalog.ByName(name)
		if !ok {
			t.Errorf("%s is not in the catalogue", name)
			continue
		}
		if module.Path() != "github.com/lemmego/"+name {
			t.Errorf("%s has path %q", name, module.Path())
		}
	}

	// And findable by what somebody would actually search for.
	for query, want := range map[string]string{
		"multi-tenancy": "tenancy",
		"roles":         "rbac",
		"permissions":   "rbac",
		"authorization": "authz",
		"translation":   "i18n",
	} {
		var found bool
		for _, module := range catalog.Match(query) {
			if module.Name == want {
				found = true
			}
		}
		if !found {
			t.Errorf("searching %q does not find %q", query, want)
		}
	}
}

// TestMatchNarrowsOnEveryTerm keeps a two-word query from widening.
func TestMatchNarrowsOnEveryTerm(t *testing.T) {
	all := len(catalog.Match(""))
	some := len(catalog.Match("database roles"))
	if some == 0 {
		t.Fatal("a reasonable two-word query matched nothing")
	}
	if some >= all {
		t.Errorf("two terms matched %d of %d modules, so it widened rather than narrowed", some, all)
	}
}
