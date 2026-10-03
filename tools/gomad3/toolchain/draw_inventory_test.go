package toolchain

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

// With GOMADSEED set, every draw the runtime rand helpers serve to an M holding
// the P comes from process-wide state derived from the seed, so a draw taken at
// a moment chosen by host timing moves every later seeded decision. Every
// reference to a helper that reaches that state, or to the M-local helpers
// that host-timed sites use instead, is pinned here with its classification; a
// new upstream or Gomad reference fails the toolchain tier on either host.
type drawClass string

const (
	// drawImplementation defines a rand helper or forwards one helper to another.
	drawImplementation drawClass = "implementation"
	// drawTargetOrdered draws at a point the target's own execution orders.
	drawTargetOrdered drawClass = "target-ordered"
	// drawHostTimed draws at a moment host timing chooses; its helper must be an
	// M-local one.
	drawHostTimed drawClass = "host-timed"
	// drawHostTimedBlocked is host-timed but stays on the seeded stream because
	// rerouting it needs a prohibited runtime file; its reason is the finding.
	drawHostTimedBlocked drawClass = "host-timed-blocked"
	// drawInactive is reached only while Gomad is disabled, when no
	// seed-derived state exists.
	drawInactive drawClass = "inactive"
	// drawDiagnostic is the diagnostics-only fault switch.
	drawDiagnostic drawClass = "diagnostic"
)

// drawMLocalHelpers draw from the calling M's own stream, never from the
// seed-derived process-wide state. cheaprand64 qualifies on both supported
// platforms because 64-bit multiplication keeps it on m.cheaprand64.
var drawMLocalHelpers = []string{"gomadHostCheapRand", "gomadHostCheapRandN", "cheaprand64", "cheaprandu64"}

// drawRuntimeHelpers are the package runtime names inventoried by reference.
var drawRuntimeHelpers = []string{
	"bootstrapRand", "cheaprand", "cheaprand64", "cheaprandn", "cheaprandu64", "gomadChoiceRandom",
	"gomadChoiceRunnextSeeded", "gomadChoiceRunqSeeded", "gomadChoiceSelectSeeded", "gomadChoiceShuffleSeeded",
	"gomadClockTickDraw", "gomadHostCheapRand", "gomadHostCheapRandN", "gomadRuntimeCheapRand", "gomadRuntimeRand",
	"gomadTimerRand", "legacy_fastrand", "legacy_fastrand64", "legacy_fastrandn", "maps_rand", "rand", "rand32", "randn",
}

// drawCompilerHelpers are the runtime helpers the compiler calls by name.
var drawCompilerHelpers = []string{"rand", "rand32"}

type drawReference struct {
	file     string
	function string
	helper   string
	count    int
	class    drawClass
	reason   string
}

func (r drawReference) key() string {
	return r.file + " " + r.function + " " + r.helper
}

var reviewedDrawReferences = []drawReference{
	{"cmd/compile/internal/walk/builtin.go", "walkMakeMap", "rand", 1, drawTargetOrdered, "seeds a non-escaping small map at its make statement"},
	{"hash/maphash/maphash.go", "randUint64", "runtime_rand", 1, drawTargetOrdered, "maphash.MakeSeed and the package seed, drawn by the caller"},
	{"internal/runtime/maps/map.go", "Map.Clear", "rand", 1, drawTargetOrdered, "reseeds a map the target clears"},
	{"internal/runtime/maps/map.go", "Map.Delete", "rand", 1, drawTargetOrdered, "reseeds a map the target empties by deletion"},
	{"internal/runtime/maps/map.go", "NewEmptyMap", "rand", 1, drawTargetOrdered, "seeds a map at its creation"},
	{"internal/runtime/maps/map.go", "NewMap", "rand", 1, drawTargetOrdered, "seeds a map at its creation"},
	{"internal/runtime/maps/runtime_alg.go", "AlgInit", "bootstrapRand", 1, drawTargetOrdered, "hash keys drawn once during runtime initialization, before user code"},
	{"internal/runtime/maps/runtime_alg.go", "initAlgAES", "bootstrapRand", 1, drawTargetOrdered, "AES hash keys drawn once during runtime initialization, before user code"},
	{"internal/runtime/maps/table.go", "Iter.Init", "rand", 2, drawTargetOrdered, "randomizes the start of a map iteration the target begins"},
	{"internal/sync/hashtriemap.go", "HashTrieMap.initSlow", "runtime_rand", 1, drawTargetOrdered, "seeds a sync.Map or unique map on its first use"},
	{"math/rand/rand.go", "runtimeSource.Int63", "runtime_rand", 1, drawTargetOrdered, "unseeded math/rand top-level functions, called by the target"},
	{"math/rand/rand.go", "runtimeSource.Uint64", "runtime_rand", 1, drawTargetOrdered, "unseeded math/rand top-level functions, called by the target"},
	{"math/rand/v2/rand.go", "runtimeSource.Uint64", "runtime_rand", 1, drawTargetOrdered, "math/rand/v2 top-level functions, called by the target"},
	{"net/dnsclient.go", "randInt", "runtime_rand", 1, drawTargetOrdered, "DNS query IDs and record shuffles of a lookup the target makes"},
	{"os/tempfile.go", "nextRandom", "runtime_rand", 1, drawTargetOrdered, "temporary names the target asks for"},
	{"runtime/alg.go", "f32hash", "rand", 1, drawTargetOrdered, "hashes a NaN key the target stores in a map"},
	{"runtime/alg.go", "f64hash", "rand", 1, drawTargetOrdered, "hashes a NaN key the target stores in a map"},
	{"runtime/gomad.go", "gomadChoiceRunnextSeeded", "gomadChoiceRandom", 1, drawImplementation, "the runnext demotion draw"},
	{"runtime/gomad.go", "gomadChoiceRunnextSeeded", "randn", 1, drawInactive, "upstream draw when Gomad is disabled"},
	{"runtime/gomad.go", "gomadChoiceRunqIndex", "gomadChoiceRunqSeeded", 1, drawTargetOrdered, "the run-queue pick among user goroutines; runtime-owned goroutines take no draw"},
	{"runtime/gomad.go", "gomadChoiceRunqSeeded", "gomadChoiceRandom", 1, drawImplementation, "the run-queue pick draw"},
	{"runtime/gomad.go", "gomadChoiceRunqSeeded", "randn", 1, drawInactive, "upstream draw when Gomad is disabled"},
	{"runtime/gomad.go", "gomadChoiceSelectSeeded", "cheaprandn", 1, drawInactive, "upstream draw when Gomad is disabled"},
	{"runtime/gomad.go", "gomadChoiceShuffleSeeded", "cheaprandn", 1, drawInactive, "upstream draw when Gomad is disabled"},
	{"runtime/gomad.go", "gomadChoiceShuffleSeeded", "gomadChoiceRandom", 1, drawImplementation, "the run-queue batch shuffle draw"},
	{"runtime/gomad.go", "gomadDiagnosticAppend", "gomadRuntimeCheapRand", 1, drawDiagnostic, "GOMAD3_DIAGNOSTIC_PERTURB_DRAW=N takes one extra seeded draw at choice record N"},
	{"runtime/gomad.go", "gomadHostCheapRand", "gomadRuntimeCheapRand", 1, drawDiagnostic, "GOMAD3_DIAGNOSTIC_PERTURB_DRAW=host-timed:N puts the next host-timed draw on the seeded stream, which the diagnostic check refuses"},
	{"runtime/gomad.go", "gomadHostCheapRandN", "gomadHostCheapRand", 1, drawImplementation, "bounded form of the M-local draw"},
	{"runtime/gomad.go", "gomadLockProfileStart", "gomadHostCheapRandN", 1, drawHostTimed, "lock-profile wait sampling where lock2 is about to sleep on a contended runtime lock"},
	{"runtime/gomad.go", "gomadTimeNow", "gomadClockTickDraw", 1, drawTargetOrdered, "the forward clock tick of a time.Now the target calls"},
	{"runtime/iface.go", "interfaceSwitch", "cheaprand", 2, drawTargetOrdered, "type-switch cache growth on a switch the target executes"},
	{"runtime/iface.go", "typeAssert", "cheaprand", 2, drawTargetOrdered, "type-assertion cache growth on an assertion the target executes"},
	{"runtime/lock_spinbit.go", "mutexSampleContention", "cheaprandu64", 1, drawHostTimed, "runtime-lock contention sampling for the mutex profile; m.cheaprand64 is M-local"},
	{"runtime/lock_spinbit.go", "unlock2Wake", "gomadHostCheapRandN", 1, drawHostTimed, "anti-starvation wake of the M at the bottom of a contended lock's wait stack"},
	{"runtime/malloc.go", "fastexprand", "cheaprandn", 1, drawTargetOrdered, "heap-profile sampling of target allocations; Gomad sets MemProfileRate to zero when user code starts"},
	{"runtime/malloc.go", "mallocinit", "bootstrapRand", 1, drawTargetOrdered, "heap base randomization during runtime initialization, before user code"},
	{"runtime/mbitmap.go", "doubleCheckHeapType", "cheaprand", 2, drawTargetOrdered, "compiled only under the false doubleCheckHeapSetType debug constant; the allocation reaching it is the target's"},
	{"runtime/mgcpacer.go", "gcControllerState.enlistWorker", "cheaprandn", 1, drawHostTimedBlocked, "picks a P to preempt for collector work at a moment the collector chooses; mgcpacer.go is a prohibited collector file, so a reroute is referred to the patch-policy owner (fn-112 Open Question 3); unreachable while Gomad pins GOMAXPROCS to one, because enlistWorker returns before the draw"},
	{"runtime/mprof.go", "blocksampled", "cheaprand64", 1, drawHostTimed, "block-profile sampling of a blocking event's host-measured cycles; m.cheaprand64 is M-local"},
	{"runtime/mprof.go", "mLockProfile.recordUnlock", "cheaprandu64", 2, drawHostTimed, "chooses which contended runtime-lock stack the mutex profile keeps; m.cheaprand64 is M-local"},
	{"runtime/mprof.go", "mLockProfile.start", "cheaprandn", 1, drawInactive, "gomadLockProfileStart calls it only when Gomad is disabled"},
	{"runtime/mprof.go", "mutexevent", "cheaprand64", 1, drawHostTimed, "mutex-profile sampling of a contention event's host-measured cycles; m.cheaprand64 is M-local"},
	{"runtime/os_linux.go", "setThreadCPUProfiler", "cheaprandn", 1, drawHostTimedBlocked, "spreads per-thread CPU-profile timers when an M first runs after SetCPUProfileRate; os_linux.go is a prohibited platform file, and CPU profiling (SIGPROF) is outside the deterministic contract"},
	{"runtime/proc.go", "newproc1", "cheaprand", 1, drawTargetOrdered, "the scheduler-tracking sequence of a goroutine the target or a runtime goroutine creates"},
	{"runtime/proc.go", "runqput", "gomadChoiceRunnextSeeded", 1, drawTargetOrdered, "the runnext demotion of a goroutine readied on the P"},
	{"runtime/proc.go", "runqputbatch", "gomadChoiceShuffleSeeded", 1, drawTargetOrdered, "shuffles a batch put on the local run queue; host netpoll batches are outside the contract"},
	{"runtime/proc.go", "runqputslow", "gomadChoiceShuffleSeeded", 1, drawTargetOrdered, "shuffles the half of a full local run queue moved to the global queue"},
	{"runtime/proc.go", "stealWork", "gomadHostCheapRand", 1, drawHostTimed, "work-steal order in the idle steal pass, whose repetitions follow idle windows the Runner decides; the pass is bracketed for the diagnostic check"},
	{"runtime/rand.go", "cheaprand", "gomadRuntimeCheapRand", 1, drawImplementation, "serves an M holding the P from the process-wide stream"},
	{"runtime/rand.go", "cheaprand64", "cheaprandu64", 1, drawImplementation, "forwards"},
	{"runtime/rand.go", "cheaprandn", "cheaprand", 1, drawImplementation, "forwards"},
	{"runtime/rand.go", "cheaprandu64", "cheaprand", 2, drawImplementation, "fallback for platforms without 64-bit multiplication, neither of the supported ones"},
	{"runtime/rand.go", "legacy_fastrand", "rand", 1, drawImplementation, "linkname alias for packages outside the standard library"},
	{"runtime/rand.go", "legacy_fastrand64", "rand", 1, drawImplementation, "linkname alias for packages outside the standard library"},
	{"runtime/rand.go", "legacy_fastrandn", "randn", 1, drawImplementation, "linkname alias for packages outside the standard library"},
	{"runtime/rand.go", "maps_rand", "rand", 1, drawImplementation, "pushes rand into internal/runtime/maps"},
	{"runtime/rand.go", "mrandinit", "bootstrapRand", 1, drawInactive, "Gomad seeds each M from GOMADSEED and returns before this draw"},
	{"runtime/rand.go", "mrandinit", "rand", 2, drawInactive, "Gomad seeds each M from GOMADSEED and returns before these draws"},
	{"runtime/rand.go", "rand", "gomadRuntimeRand", 1, drawImplementation, "serves an M holding the P from the process-wide stream"},
	{"runtime/rand.go", "rand32", "rand", 1, drawImplementation, "forwards"},
	{"runtime/rand.go", "randn", "rand", 1, drawImplementation, "forwards"},
	{"runtime/select.go", "selectgo", "gomadChoiceSelectSeeded", 1, drawTargetOrdered, "the poll order of a select the target executes"},
	{"runtime/sema.go", "semaRoot.queue", "cheaprand", 1, drawTargetOrdered, "treap ticket of a goroutine that blocks on a semaphore"},
	{"runtime/symtab.go", "pcvalue", "gomadHostCheapRandN", 1, drawHostTimed, "pcvalue-cache eviction, whose contents follow stack walks on whichever M holds the P"},
	{"runtime/time.go", "timer.maybeAdd", "gomadTimerRand", 1, drawTargetOrdered, "tie-break of a timer the target arms"},
	{"runtime/time.go", "timer.updateHeap", "gomadTimerRand", 1, drawTargetOrdered, "tie-break of a timer the target modifies, read under ts.mu"},
	{"runtime/time.go", "timers.adjust", "gomadTimerRand", 1, drawTargetOrdered, "tie-break of a modified timer at the P's next timer check"},
	{"sync/pool.go", "Pool.Put", "runtime_randn", 1, drawTargetOrdered, "drops a Put under the race detector only"},
	{"unique/canonmap.go", "newCanonMap", "runtime_rand", 1, drawTargetOrdered, "seeds a unique map on its first use"},
}

func TestPatchedRuntimeSeededDrawReferencesAreReviewed(t *testing.T) {
	goRoot := builtGOROOT(t)
	got, err := seededDrawReferences(goRoot, gomadversion.SupportedPlatforms[:])
	if err != nil {
		t.Fatal(err)
	}
	if problems := seededDrawProblems(got, reviewedDrawReferences); len(problems) > 0 {
		t.Fatalf("patched runtime seeded-stream draw inventory drifted; classify each change and update reviewedDrawReferences:\n%s", strings.Join(problems, "\n"))
	}
}

func TestSeededDrawInventoryRejectsUnclassifiedReference(t *testing.T) {
	goRoot := t.TempDir()
	for relative, contents := range map[string]string{
		"src/runtime/rand.go":               "package runtime\n\nfunc rand() uint64 { return 0 }\n\nfunc cheaprand() uint32 { return uint32(rand()) }\n",
		"src/runtime/gomad_unclassified.go": "package runtime\n\nfunc unclassified() uint32 {\n\t// cheaprand() in a comment is not a reference.\n\treturn cheaprand() + gomadHostCheapRand()\n}\n\nfunc gomadHostCheapRand() uint32 { return 0 }\n",
		"src/math/rand/rand.go":             "package rand\n\nimport _ \"unsafe\"\n\n//go:linkname runtime_rand runtime.rand\nfunc runtime_rand() uint64\n\nfunc seed() int64 { return int64(runtime_rand()) }\n",
	} {
		path := filepath.Join(goRoot, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := seededDrawReferences(goRoot, gomadversion.SupportedPlatforms[:])
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{
		"runtime/rand.go cheaprand rand":                                1,
		"runtime/gomad_unclassified.go unclassified cheaprand":          1,
		"runtime/gomad_unclassified.go unclassified gomadHostCheapRand": 1,
		"math/rand/rand.go seed runtime_rand":                           1,
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("references = %v, want %v", got, want)
	}
	reviewed := []drawReference{
		{"runtime/rand.go", "cheaprand", "rand", 1, drawImplementation, "forwards"},
		{"runtime/gomad_unclassified.go", "unclassified", "gomadHostCheapRand", 1, drawHostTimed, "M-local"},
		{"math/rand/rand.go", "seed", "runtime_rand", 1, drawTargetOrdered, "seeds the source"},
	}
	problems := seededDrawProblems(got, reviewed)
	if !slices.Equal(problems, []string{"unclassified seeded-stream reference: runtime/gomad_unclassified.go unclassified cheaprand (1)"}) {
		t.Fatalf("problems = %q", problems)
	}
	for _, mutate := range []struct {
		name    string
		change  func(*drawReference)
		problem string
	}{
		{"host-timed on the seeded stream", func(r *drawReference) { r.helper, r.class = "cheaprand", drawHostTimed }, "host-timed reference uses a seeded helper: runtime/gomad_unclassified.go unclassified cheaprand"},
		{"blocked without a finding", func(r *drawReference) { r.helper, r.class, r.reason = "cheaprand", drawHostTimedBlocked, "" }, "classified seeded-stream reference has no reason: runtime/gomad_unclassified.go unclassified cheaprand"},
	} {
		changed := slices.Clone(reviewed)
		entry := reviewed[1]
		mutate.change(&entry)
		changed = append(changed, entry)
		if problems := seededDrawProblems(got, changed); !slices.Contains(problems, mutate.problem) {
			t.Fatalf("%s: problems = %q", mutate.name, problems)
		}
	}
}

// seededDrawReferences counts, per file, enclosing function, and helper, the
// references to the runtime rand helpers in every non-test Go file that builds
// for a platform under the target build configuration, together with the
// references other packages make through a linkname to one of those helpers
// and the calls the compiler emits by name. A key is platform-independent; a
// count that differs between platforms is an error.
func seededDrawReferences(goRoot string, platforms []string) (map[string]int, error) {
	references := map[string]int{}
	sourceRoot := filepath.Join(goRoot, "src")
	for _, platform := range platforms {
		goos, goarch, ok := strings.Cut(platform, "/")
		if !ok {
			return nil, fmt.Errorf("malformed platform %q", platform)
		}
		context := build.Default
		context.GOROOT, context.GOOS, context.GOARCH, context.CgoEnabled = goRoot, goos, goarch, false
		// Targets build with GOEXPERIMENT=nogreenteagc (target/internal/build).
		context.ToolTags = slices.DeleteFunc(slices.Clone(context.ToolTags), func(tag string) bool { return tag == "goexperiment.greenteagc" })
		files := map[string][]string{}
		err := filepath.WalkDir(sourceRoot, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			name := entry.Name()
			if entry.IsDir() {
				if name == "testdata" || name == "vendor" || path == filepath.Join(sourceRoot, "cmd") {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				return nil
			}
			match, err := context.MatchFile(filepath.Dir(path), name)
			if err != nil || !match {
				return err
			}
			files[filepath.Dir(path)] = append(files[filepath.Dir(path)], path)
			return nil
		})
		if err != nil {
			return nil, err
		}
		helpers, err := drawHelpersByDirectory(sourceRoot, files)
		if err != nil {
			return nil, err
		}
		counts := map[string]int{}
		for directory, names := range helpers {
			for _, path := range files[directory] {
				if err := countDrawReferences(sourceRoot, path, names, counts); err != nil {
					return nil, err
				}
			}
		}
		if err := countCompilerDrawCalls(sourceRoot, counts); err != nil {
			return nil, err
		}
		for key, count := range counts {
			if previous, found := references[key]; found && previous != count {
				return nil, fmt.Errorf("seeded-stream reference %s counts %d on one platform and %d on another", key, previous, count)
			}
			references[key] = count
		}
	}
	return references, nil
}

var drawLinkname = regexp.MustCompile(`(?m)^//go:linkname(?:std)?\s+(\w+)\s+runtime\.(\w+)\s*$`)

// drawHelpersByDirectory names the helpers inventoried in each package
// directory: the runtime's own, and in another package each local name a
// linkname directive binds to one of them.
func drawHelpersByDirectory(sourceRoot string, files map[string][]string) (map[string][]string, error) {
	helpers := map[string][]string{filepath.Join(sourceRoot, "runtime"): drawRuntimeHelpers}
	for directory, paths := range files {
		if directory == filepath.Join(sourceRoot, "runtime") {
			continue
		}
		for _, path := range paths {
			contents, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			if !bytes.Contains(contents, []byte("linkname")) {
				continue
			}
			for _, match := range drawLinkname.FindAllStringSubmatch(string(contents), -1) {
				if slices.Contains(drawRuntimeHelpers, match[2]) && !slices.Contains(helpers[directory], match[1]) {
					helpers[directory] = append(helpers[directory], match[1])
				}
			}
		}
	}
	// internal/runtime/maps declares rand without a body; the runtime pushes
	// maps_rand into it.
	maps := filepath.Join(sourceRoot, "internal", "runtime", "maps")
	if _, found := files[maps]; found && !slices.Contains(helpers[maps], "rand") {
		helpers[maps] = append(helpers[maps], "rand")
	}
	return helpers, nil
}

// countDrawReferences counts the identifiers naming a helper inside each
// declaration of the file, ignoring comments, selector fields, struct fields,
// and the helper's own declared name.
func countDrawReferences(sourceRoot, path string, helpers []string, counts map[string]int) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(helpers, func(helper string) bool { return bytes.Contains(contents, []byte(helper)) }) {
		return nil
	}
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, contents, parser.SkipObjectResolution)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(sourceRoot, path)
	if err != nil {
		return err
	}
	relative = filepath.ToSlash(relative)
	for _, declaration := range file.Decls {
		function := "(package)"
		var body ast.Node = declaration
		if definition, ok := declaration.(*ast.FuncDecl); ok {
			if definition.Body == nil {
				continue
			}
			function = definition.Name.Name
			if definition.Recv != nil && len(definition.Recv.List) == 1 {
				function = receiverTypeName(definition.Recv.List[0].Type) + "." + function
			}
			body = definition.Body
		}
		ast.Inspect(body, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.SelectorExpr:
				ast.Inspect(node.X, func(inner ast.Node) bool { return countDrawIdentifier(inner, relative, function, helpers, counts) })
				return false
			case *ast.Field:
				if node.Type != nil {
					ast.Inspect(node.Type, func(inner ast.Node) bool { return countDrawIdentifier(inner, relative, function, helpers, counts) })
				}
				return false
			case *ast.KeyValueExpr:
				ast.Inspect(node.Value, func(inner ast.Node) bool { return countDrawIdentifier(inner, relative, function, helpers, counts) })
				return false
			default:
				return countDrawIdentifier(node, relative, function, helpers, counts)
			}
		})
	}
	return nil
}

func countDrawIdentifier(node ast.Node, file, function string, helpers []string, counts map[string]int) bool {
	if identifier, ok := node.(*ast.Ident); ok && slices.Contains(helpers, identifier.Name) {
		counts[file+" "+function+" "+identifier.Name]++
	}
	return true
}

// countCompilerDrawCalls counts the runtime rand helpers the compiler's walk
// phase calls by name, such as the seed of a non-escaping small map.
func countCompilerDrawCalls(sourceRoot string, counts map[string]int) error {
	directory := filepath.Join(sourceRoot, "cmd", "compile", "internal", "walk")
	entries, err := os.ReadDir(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, filepath.Join(directory, name), nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		for _, declaration := range file.Decls {
			definition, ok := declaration.(*ast.FuncDecl)
			if !ok || definition.Body == nil {
				continue
			}
			ast.Inspect(definition.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok || len(call.Args) == 0 {
					return true
				}
				literal, ok := call.Args[0].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					return true
				}
				helper, err := strconv.Unquote(literal.Value)
				if err == nil && slices.Contains(drawCompilerHelpers, helper) {
					counts["cmd/compile/internal/walk/"+name+" "+definition.Name.Name+" "+helper]++
				}
				return true
			})
		}
	}
	return nil
}

// seededDrawProblems compares the scanned references with the reviewed
// inventory; an empty result means every reference is classified, every
// classification carries its reason, and every host-timed reference that is
// not blocked uses an M-local helper.
func seededDrawProblems(got map[string]int, reviewed []drawReference) []string {
	var problems []string
	want := map[string]drawReference{}
	for _, reference := range reviewed {
		if reference.reason == "" {
			problems = append(problems, "classified seeded-stream reference has no reason: "+reference.key())
		}
		if _, duplicate := want[reference.key()]; duplicate {
			problems = append(problems, "classified seeded-stream reference is listed twice: "+reference.key())
		}
		if reference.class == drawHostTimed && !slices.Contains(drawMLocalHelpers, reference.helper) {
			problems = append(problems, "host-timed reference uses a seeded helper: "+reference.key())
		}
		want[reference.key()] = reference
	}
	for key, count := range got {
		reference, reviewed := want[key]
		switch {
		case !reviewed:
			problems = append(problems, fmt.Sprintf("unclassified seeded-stream reference: %s (%d)", key, count))
		case reference.count != count:
			problems = append(problems, fmt.Sprintf("seeded-stream reference count changed: %s = %d, reviewed %d", key, count, reference.count))
		default:
		}
	}
	for key := range want {
		if _, found := got[key]; !found {
			problems = append(problems, "classified seeded-stream reference disappeared: "+key)
		}
	}
	slices.Sort(problems)
	return problems
}
