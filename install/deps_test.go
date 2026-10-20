package install_test

import (
	"os/exec"
	"strings"
	"testing"
)

// The installer must work in a project that cannot boot, so it must not depend
// on the framework.
//
// Measured rather than intended. The split is the whole design of this
// package — an installer that needed a booted application would be useless in
// a freshly scaffolded project, which is when somebody actually runs it — and
// an import added in the wrong file is exactly how that would be lost. This
// repository has the same guard in notify and media for the same reason.
//
// It also keeps the package importable by lemmego/cli, which deliberately
// keeps api/app out of its graph: that package alone reaches several hundred
// dependencies.
func TestInstallerDoesNotDependOnTheFramework(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}

	forbidden := []string{
		"github.com/lemmego/api/app",
		"github.com/lemmego/api/config",
		"github.com/modelcontextprotocol/go-sdk",
		"github.com/spf13/cobra",
	}
	for _, line := range strings.Split(string(out), "\n") {
		pkg := strings.TrimSpace(line)
		for _, bad := range forbidden {
			if pkg == bad || strings.HasPrefix(pkg, bad+"/") {
				t.Errorf("install depends on %s, which it must not: "+
					"this package has to work without booting the application", pkg)
			}
		}
	}
}

// Standard library only, stated as a number so a new dependency is a visible
// decision rather than something that slips in.
func TestInstallerUsesOnlyTheStandardLibrary(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		pkg := strings.TrimSpace(line)
		if pkg == "" || pkg == "github.com/lemmego/mcp/install" {
			continue
		}
		// A standard library package has no dot in its first path segment.
		first, _, _ := strings.Cut(pkg, "/")
		if strings.Contains(first, ".") {
			t.Errorf("install depends on the non-standard package %s", pkg)
		}
	}
}
