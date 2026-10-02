package toolchain

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

// A goroutine started from the scheduler's stack, or before the root identity
// exists, has no identified parent, so the parent-child derivation cannot name
// it. Every go statement of the pinned runtime package and of package time is
// pinned here with the schedule-independent derivation its goroutines take or
// the reason it stays a declared exception; a new upstream site fails the
// toolchain tier on either host.
type creationDisposition string

const (
	// creationDerived takes a schedule-independent identity through a
	// derivation of its own; the reason names it.
	creationDerived creationDisposition = "derived"
	// creationException keeps the identity it has today; the reason says why
	// that is acceptable or whom the change is referred to.
	creationException creationDisposition = "exception"
)

type goroutineCreation struct {
	file        string
	function    string
	count       int
	disposition creationDisposition
	reason      string
}

func (c goroutineCreation) key() string {
	return c.file + " " + c.function
}

var reviewedGoroutineCreations = []goroutineCreation{
	{"runtime/mcleanup.go", "cleanupQueue.createGs", 1, creationException, "cleanup goroutines start from the first AddCleanup caller that needs them; mcleanup.go is a prohibited collector file, so a derivation is referred to the patch-policy owner"},
	{"runtime/mfinal.go", "createfing", 1, creationException, "the finalizer goroutine starts from the first SetFinalizer caller; mfinal.go is a prohibited collector file, so a derivation is referred to the patch-policy owner"},
	{"runtime/mgc.go", "gcBgMarkStartWorkers", 1, creationException, "the mark worker starts from whichever goroutine triggers the first cycle; mgc.go is a prohibited collector file, so a derivation is referred to the patch-policy owner"},
	{"runtime/mgc.go", "gcenable", 2, creationException, "bgsweep and bgscavenge start from runtime.main before user code and before the root identity exists, so their runtime ordinals follow initialization order, not a schedule"},
	{"runtime/proc.go", "defaultGOMAXPROCSUpdateEnable", 1, creationException, "the GOMAXPROCS updater starts from runtime.main before user code and before the root identity exists, so its runtime ordinal follows initialization order, not a schedule"},
	{"runtime/proc.go", "init", 1, creationException, "forcegchelper starts from a runtime init task before user code and before the root identity exists, so its runtime ordinal follows initialization order, not a schedule"},
	{"runtime/signal_unix.go", "ensureSigM", 1, creationException, "the signal goroutine starts from the first os/signal Notify caller, which only an exact compatibility pack admits; signal_unix.go is a prohibited platform file"},
	{"runtime/trace.go", "traceAdvancerState.start", 1, creationException, "the execution tracer is outside the deterministic contract"},
	{"runtime/tracecpu.go", "traceStartReadCPU", 1, creationException, "the execution tracer is outside the deterministic contract"},
	{"time/sleep.go", "goFunc", 1, creationDerived, "an AfterFunc callback goroutine takes gomad3-choice-goroutine-timer/v1 from the timer's creator and firing ordinal; a timer created with no identified goroutine stays on the runtime ordinal"},
}

func TestPatchedRuntimeGoroutineCreationsAreReviewed(t *testing.T) {
	goRoot := builtGOROOT(t)
	got, err := goroutineCreationSites(goRoot, gomadversion.SupportedPlatforms[:])
	if err != nil {
		t.Fatal(err)
	}
	if problems := goroutineCreationProblems(got, reviewedGoroutineCreations); len(problems) > 0 {
		t.Fatalf("patched runtime goroutine creation inventory drifted; review each change and update reviewedGoroutineCreations:\n%s", strings.Join(problems, "\n"))
	}
}

func TestGoroutineCreationInventoryRejectsSeededSite(t *testing.T) {
	goRoot := t.TempDir()
	for relative, contents := range map[string]string{
		"src/runtime/gomad_seeded.go": "package runtime\n\nfunc seededWorker() {}\n\nfunc seededStart() {\n\tgo seededWorker()\n\tgo func() { seededWorker() }()\n}\n",
		"src/runtime/mgc.go":          "package runtime\n\nfunc bgsweep(c chan int) {}\n\nfunc bgscavenge(c chan int) {}\n\nfunc gcenable() {\n\tc := make(chan int, 2)\n\tgo bgsweep(c)\n\tgo bgscavenge(c)\n}\n",
		"src/time/sleep.go":           "package time\n\nfunc goFunc(arg any, seq uintptr, delta int64) {\n\tgo arg.(func())()\n}\n",
	} {
		path := filepath.Join(goRoot, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := goroutineCreationSites(goRoot, gomadversion.SupportedPlatforms[:])
	if err != nil {
		t.Fatal(err)
	}
	problems := goroutineCreationProblems(got, reviewedGoroutineCreations)
	if !slices.Contains(problems, "unreviewed goroutine creation site: runtime/gomad_seeded.go seededStart (2)") {
		t.Fatalf("seeded creation site was not reported: %q", problems)
	}
	for _, creation := range reviewedGoroutineCreations {
		if _, seeded := got[creation.key()]; seeded {
			continue
		}
		if !slices.Contains(problems, "reviewed goroutine creation site disappeared: "+creation.key()) {
			t.Fatalf("missing reviewed site %s was not reported: %q", creation.key(), problems)
		}
	}
	if len(problems) != 1+len(reviewedGoroutineCreations)-2 {
		t.Fatalf("problems = %q", problems)
	}
	unreasoned := append([]goroutineCreation{{"runtime/gomad_seeded.go", "seededStart", 2, creationException, ""}}, reviewedGoroutineCreations...)
	if problems := goroutineCreationProblems(got, unreasoned); !slices.Contains(problems, "reviewed goroutine creation site has no reason: runtime/gomad_seeded.go seededStart") {
		t.Fatalf("exception without a reason was accepted: %q", problems)
	}
}

// goroutineCreationSites counts the go statements of each top-level function
// in every non-test Go file of src/runtime and src/time that builds for one of
// the platforms. A site is keyed without its platform because its disposition
// does not depend on one; a count that differs between platforms is an error.
func goroutineCreationSites(goRoot string, platforms []string) (map[string]int, error) {
	sites := map[string]int{}
	for _, platform := range platforms {
		goos, goarch, ok := strings.Cut(platform, "/")
		if !ok {
			return nil, fmt.Errorf("malformed platform %q", platform)
		}
		context := build.Default
		context.GOROOT, context.GOOS, context.GOARCH, context.CgoEnabled = goRoot, goos, goarch, false
		counts := map[string]int{}
		for _, packagePath := range []string{"runtime", "time"} {
			directory := filepath.Join(goRoot, "src", packagePath)
			entries, err := os.ReadDir(directory)
			if err != nil {
				return nil, err
			}
			for _, entry := range entries {
				name := entry.Name()
				if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
					continue
				}
				match, err := context.MatchFile(directory, name)
				if err != nil {
					return nil, err
				}
				if !match {
					continue
				}
				if err := countGoStatements(filepath.Join(directory, name), packagePath+"/"+name, counts); err != nil {
					return nil, err
				}
			}
		}
		for key, count := range counts {
			if previous, found := sites[key]; found && previous != count {
				return nil, fmt.Errorf("goroutine creation site %s has %d statements on one platform and %d on another", key, previous, count)
			}
			sites[key] = count
		}
	}
	return sites, nil
}

func countGoStatements(path, relative string, counts map[string]int) error {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return err
	}
	for _, declaration := range file.Decls {
		definition, ok := declaration.(*ast.FuncDecl)
		if !ok || definition.Body == nil {
			continue
		}
		function := definition.Name.Name
		if definition.Recv != nil && len(definition.Recv.List) == 1 {
			function = receiverTypeName(definition.Recv.List[0].Type) + "." + function
		}
		ast.Inspect(definition.Body, func(node ast.Node) bool {
			if _, ok := node.(*ast.GoStmt); ok {
				counts[relative+" "+function]++
			}
			return true
		})
	}
	return nil
}

func receiverTypeName(expression ast.Expr) string {
	switch typed := expression.(type) {
	case *ast.StarExpr:
		return receiverTypeName(typed.X)
	case *ast.Ident:
		return typed.Name
	case *ast.IndexExpr:
		return receiverTypeName(typed.X)
	case *ast.IndexListExpr:
		return receiverTypeName(typed.X)
	default:
		return "?"
	}
}

// goroutineCreationProblems compares the scanned sites with the reviewed
// inventory; an empty result means every site is accounted for with a reason.
func goroutineCreationProblems(got map[string]int, reviewed []goroutineCreation) []string {
	var problems []string
	want := map[string]goroutineCreation{}
	for _, creation := range reviewed {
		if creation.reason == "" {
			problems = append(problems, "reviewed goroutine creation site has no reason: "+creation.key())
		}
		if _, duplicate := want[creation.key()]; duplicate {
			problems = append(problems, "reviewed goroutine creation site is listed twice: "+creation.key())
		}
		want[creation.key()] = creation
	}
	for key, count := range got {
		creation, reviewed := want[key]
		switch {
		case !reviewed:
			problems = append(problems, fmt.Sprintf("unreviewed goroutine creation site: %s (%d)", key, count))
		case creation.count != count:
			problems = append(problems, fmt.Sprintf("goroutine creation site count changed: %s = %d, reviewed %d", key, count, creation.count))
		default:
		}
	}
	for key := range want {
		if _, found := got[key]; !found {
			problems = append(problems, "reviewed goroutine creation site disappeared: "+key)
		}
	}
	slices.Sort(problems)
	return problems
}
