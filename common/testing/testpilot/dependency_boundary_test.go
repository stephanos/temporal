package testpilot

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	serverModule = "go.temporal.io/server/"
	actPackage   = serverModule + "common/testing/testpilot/temporal/act"
)

// actImports are the Model packages the Runner's Temporal Driver may import: the Runner, the Run
// and protocol schemas, the shared envelopes, and Temporal's read instruction. Never the Host,
// the guest or the Realization.
var actImports = []string{
	"umpire.local/model/runner", "umpire.local/model/gen/umpire/run/v1", "umpire.local/model/gen/umpire/model/v1",
	"umpire.local/model/gen/umpire/realization/v1", "umpire.local/temporal/umpire/temporal/v1",
}

// forbiddenPrefixes are what no Testpilot package imports: repository tests, Umpire tooling and
// the canary, which all import Testpilot.
var forbiddenPrefixes = []string{serverModule + "tests", serverModule + "tools/umpire", serverModule + "tools/canary"}

type testpilotPackage struct {
	ImportPath   string
	Imports      []string
	TestImports  []string
	XTestImports []string
}

// boundaryProblems names each import that crosses the Testpilot boundary.
func boundaryProblems(packages []testpilotPackage) []string {
	var problems []string
	for _, pkg := range packages {
		check := func(imported string, test bool) {
			for _, prefix := range forbiddenPrefixes {
				if imported == prefix || strings.HasPrefix(imported, prefix+"/") {
					problems = append(problems, pkg.ImportPath+" imports "+imported)
				}
			}
			if !strings.HasPrefix(imported, "umpire.local/") {
				return
			}
			allowed := pkg.ImportPath == actPackage && (slices.Contains(actImports, imported) || test && imported == "umpire.local/temporal")
			if !allowed {
				problems = append(problems, pkg.ImportPath+" imports the Model host "+imported)
			}
		}
		for _, imported := range pkg.Imports {
			check(imported, false)
		}
		for _, imported := range append(slices.Clone(pkg.TestImports), pkg.XTestImports...) {
			check(imported, true)
		}
	}
	slices.Sort(problems)
	return problems
}

// Only the Runner's Temporal Driver imports the Model's packages, and only the ones it acts
// through; no Testpilot package imports what imports Testpilot.
func TestTestpilotDependencyBoundary(t *testing.T) {
	root, err := filepath.Abs("../../..")
	require.NoError(t, err)
	command := exec.Command("go", "list", "-tags", "test_dep", "-json", "./common/testing/testpilot/...")
	command.Dir = root
	output, err := command.Output()
	require.NoError(t, err)
	var packages []testpilotPackage
	var paths []string
	decoder := json.NewDecoder(bytes.NewReader(output))
	for {
		var pkg testpilotPackage
		err := decoder.Decode(&pkg)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		packages, paths = append(packages, pkg), append(paths, pkg.ImportPath)
	}
	require.Contains(t, paths, actPackage)
	require.Empty(t, boundaryProblems(packages))
}

func TestTestpilotDependencyBoundaryRejectsForbiddenEdges(t *testing.T) {
	other := serverModule + "common/testing/testpilot/temporal/provision"
	for name, test := range map[string]struct {
		pkg  testpilotPackage
		want string
	}{
		"a package imports repository tests": {testpilotPackage{ImportPath: other, Imports: []string{serverModule + "tests/testcore"}},
			other + " imports " + serverModule + "tests/testcore"},
		"a test imports Umpire tooling": {testpilotPackage{ImportPath: other, TestImports: []string{serverModule + "tools/umpire/ir"}},
			other + " imports " + serverModule + "tools/umpire/ir"},
		"a package imports the canary": {testpilotPackage{ImportPath: other, Imports: []string{serverModule + "tools/canary/policy"}},
			other + " imports " + serverModule + "tools/canary/policy"},
		"another package imports the Runner": {testpilotPackage{ImportPath: other, Imports: []string{"umpire.local/model/runner"}},
			other + " imports the Model host umpire.local/model/runner"},
		"the Driver imports the Host": {testpilotPackage{ImportPath: actPackage, Imports: []string{"umpire.local/model/host"}},
			actPackage + " imports the Model host umpire.local/model/host"},
		"the Driver imports the guest outside its tests": {testpilotPackage{ImportPath: actPackage, Imports: []string{"umpire.local/temporal"}},
			actPackage + " imports the Model host umpire.local/temporal"},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, []string{test.want}, boundaryProblems([]testpilotPackage{test.pkg}))
		})
	}
	require.Empty(t, boundaryProblems([]testpilotPackage{{
		ImportPath: actPackage, Imports: actImports, XTestImports: []string{"umpire.local/temporal"},
	}}), "the Driver's own imports and its tests' guest are allowed")
}
