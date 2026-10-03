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

type drawReference struct {
	platform, file, function, symbol, kind string
	count                                  int
	class, stream, reachability, reason    string
}

func (r drawReference) key() string {
	return strings.Join([]string{r.platform, r.file, r.function, r.symbol, r.kind}, " ")
}

// Each row is platform|file|function|symbol|reference kind|count|ordering|stream|reachability|reason.
// "both" expands to both qualified platforms; a platform-specific row uses its exact name.
const reviewedDrawReferences = `
both|hash/maphash/maphash.go|package|runtime_rand|linkname:runtime.rand|1|reference|wrapper|active|reviewed linkname binding and target
both|hash/maphash/maphash.go|randUint64|runtime_rand|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|hash/maphash/maphash.go|runtime_rand|runtime_rand|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|internal/runtime/maps/map.go|NewEmptyMap|rand|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|internal/runtime/maps/map.go|NewMap|rand|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|internal/runtime/maps/map.go|method.Clear|rand|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|internal/runtime/maps/map.go|method.Delete|rand|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|internal/runtime/maps/runtime.go|bootstrapRand|bootstrapRand|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|internal/runtime/maps/runtime.go|package|rand|linkname|1|reference|wrapper|active|reviewed linkname binding and target
both|internal/runtime/maps/runtime.go|rand|rand|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|internal/runtime/maps/runtime_alg.go|AlgInit|bootstrapRand|call|1|host-timed|wrapper|disabled|bootstrap entropy is independent of GOMADSEED
both|internal/runtime/maps/runtime_alg.go|initAlgAES|bootstrapRand|call|1|host-timed|wrapper|disabled|bootstrap entropy is independent of GOMADSEED
both|internal/runtime/maps/table.go|method.Init|rand|call|2|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|internal/sync/hashtriemap.go|method.initSlow|runtime_rand|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|internal/sync/hashtriemap.go|package|runtime_rand|linkname:runtime.rand|1|reference|wrapper|active|reviewed linkname binding and target
both|internal/sync/hashtriemap.go|runtime_rand|runtime_rand|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|math/rand/rand.go|method.Int63|runtime_rand|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|math/rand/rand.go|method.Uint64|runtime_rand|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|math/rand/rand.go|package|runtime_rand|linkname:runtime.rand|1|reference|wrapper|active|reviewed linkname binding and target
both|math/rand/rand.go|runtime_rand|runtime_rand|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|math/rand/v2/rand.go|method.Uint64|runtime_rand|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|math/rand/v2/rand.go|package|runtime_rand|linkname:runtime.rand|1|reference|wrapper|active|reviewed linkname binding and target
both|math/rand/v2/rand.go|runtime_rand|runtime_rand|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|net/dnsclient.go|package|runtime_rand|linkname:runtime.rand|1|reference|wrapper|active|reviewed linkname binding and target
both|net/dnsclient.go|randInt|runtime_rand|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|net/dnsclient.go|runtime_rand|runtime_rand|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|os/tempfile.go|nextRandom|runtime_rand|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|os/tempfile.go|package|runtime_rand|linkname:runtime.rand|1|reference|wrapper|active|reviewed linkname binding and target
both|os/tempfile.go|runtime_rand|runtime_rand|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/alg.go|f32hash|rand|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|runtime/alg.go|f64hash|rand|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|runtime/gomad.go|gomadChoiceRandom|gomadChoiceRandom|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/gomad.go|gomadChoiceRunnextSeeded|gomadChoiceRandom|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|runtime/gomad.go|gomadChoiceRunnextSeeded|gomadChoiceRunnextSeeded|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/gomad.go|gomadChoiceRunnextSeeded|randn|call|1|target-ordered|wrapper|disabled|upstream fallback executes only when Gomad is off
both|runtime/gomad.go|gomadChoiceRunqIndex|gomadChoiceRunqSeeded|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|runtime/gomad.go|gomadChoiceRunqSeeded|gomadChoiceRandom|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|runtime/gomad.go|gomadChoiceRunqSeeded|gomadChoiceRunqSeeded|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/gomad.go|gomadChoiceRunqSeeded|randn|call|1|target-ordered|wrapper|disabled|upstream fallback executes only when Gomad is off
both|runtime/gomad.go|gomadChoiceSelectSeeded|cheaprandn|call|1|target-ordered|wrapper|disabled|upstream fallback executes only when Gomad is off
both|runtime/gomad.go|gomadChoiceSelectSeeded|gomadChoiceSelectSeeded|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/gomad.go|gomadChoiceShuffleSeeded|cheaprandn|call|1|target-ordered|wrapper|disabled|upstream fallback executes only when Gomad is off
both|runtime/gomad.go|gomadChoiceShuffleSeeded|gomadChoiceRandom|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|runtime/gomad.go|gomadChoiceShuffleSeeded|gomadHostCheapRandN|call|1|host-timed|M-local|active|netpoll batch shuffle uses the M-local stream
both|runtime/gomad.go|gomadChoiceShuffleSeeded|gomadChoiceShuffleSeeded|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/gomad.go|gomadClockTickDraw|gomadClockTickDraw|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/gomad.go|gomadDiagnosticAppend|gomadRuntimeCheapRand|call|1|host-timed|process-purpose|contract-excluded|numeric diagnostic fault injection; host prefix guard exits before mutation
both|runtime/gomad.go|gomadHostCheapRand|cheaprand|value|3|reference|none|active|matching field or local value; no draw
both|runtime/gomad.go|gomadHostCheapRand|gomadRuntimeCheapRand|call|1|host-timed|process-purpose|contract-excluded|diagnostic host-timed fault intentionally attempts a seeded draw and the bracketed path rejects it
both|runtime/gomad.go|gomadHostCheapRand|gomadHostCheapRand|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/gomad.go|gomadHostCheapRandN|gomadHostCheapRand|call|1|host-timed|M-local|active|host scheduling or profiling draw uses current M
both|runtime/gomad.go|gomadHostCheapRandN|gomadHostCheapRandN|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/gomad.go|gomadLockProfileStart|gomadHostCheapRandN|call|1|host-timed|M-local|active|host scheduling or profiling draw uses current M
both|runtime/gomad.go|gomadLockProfileStart|gomadLockProfileStart|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/gomad.go|gomadRuntimeCheapRand|gomadRuntimeCheapRand|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/gomad.go|gomadRuntimeRand|gomadRuntimeRand|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/gomad.go|gomadTimeNow|gomadClockTickDraw|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|runtime/gomad.go|gomadTimerRand|gomadTimerRand|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/iface.go|interfaceSwitch|cheaprand|call|2|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|runtime/iface.go|typeAssert|cheaprand|call|2|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|runtime/lock_spinbit.go|lock2|gomadLockProfileStart|call|1|host-timed|M-local|active|host scheduling or profiling draw uses current M
both|runtime/lock_spinbit.go|mutexSampleContention|cheaprandu64|call|1|host-timed|M-local|active|mutex sampling uses per-M 64-bit stream on qualified architectures
both|runtime/lock_spinbit.go|unlock2Wake|gomadHostCheapRandN|call|1|host-timed|M-local|active|host scheduling or profiling draw uses current M
both|runtime/malloc.go|fastexprand|cheaprandn|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|runtime/malloc.go|mallocinit|bootstrapRand|call|1|host-timed|wrapper|disabled|bootstrap entropy read precedes Gomad activation
both|runtime/mbitmap.go|doubleCheckHeapType|cheaprand|call|2|target-ordered|wrapper|disabled|doubleCheckMalloc is false in qualified builds
both|runtime/mgcmark_greenteagc.go|method.tryStealSpan|cheaprand|call|1|host-timed|process-purpose|contract-excluded|GreenTea collector is refused for seeded activation; collector edit prohibited
both|runtime/mgcpacer.go|method.enlistWorker|cheaprandn|call|1|host-timed|process-purpose|blocked|classic GC worker enlistment is host timed; collector edit prohibited
both|runtime/mprof.go|blocksampled|cheaprand64|call|1|host-timed|M-local|active|profile sampling uses per-M 64-bit stream on qualified architectures
both|runtime/mprof.go|method.recordUnlock|cheaprandu64|call|2|host-timed|M-local|active|profile sampling uses per-M 64-bit stream on qualified architectures
both|runtime/mprof.go|method.start|cheaprandn|call|1|host-timed|wrapper|disabled|only called from Gomad lock profile shim when Gomad is off
both|runtime/mprof.go|mutexevent|cheaprand64|call|1|host-timed|M-local|active|profile sampling uses per-M 64-bit stream on qualified architectures
linux/amd64|runtime/os_linux.go|setThreadCPUProfiler|gomadHostCheapRandN|call|1|host-timed|M-local|active|host scheduling or profiling draw uses current M
both|runtime/proc.go|newproc1|cheaprand|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|runtime/proc.go|runqput|gomadChoiceRunnextSeeded|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|runtime/proc.go|runqputbatch|gomadChoiceShuffleSeeded|call|1|target-ordered|process-purpose|active|target batch origin retains seeded scheduler stream
both|runtime/proc.go|runqputbatch|gomadHostCheapRandN|call|1|host-timed|M-local|active|host scheduling or profiling draw uses current M
both|runtime/proc.go|runqputslow|gomadChoiceShuffleSeeded|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|runtime/proc.go|stealWork|gomadHostCheapRand|call|1|host-timed|M-local|active|host scheduling or profiling draw uses current M
both|runtime/rand.go|bootstrapRand|bootstrapRand|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/rand.go|cheaprand|cheaprand|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/rand.go|cheaprand|cheaprand|value|3|reference|none|active|matching field or local value; no draw
both|runtime/rand.go|cheaprand|cheaprand64|value|1|reference|none|active|matching field or local value; no draw
both|runtime/rand.go|cheaprand|gomadRuntimeCheapRand|call|1|target-ordered|process-purpose|active|P-owned runtime cheap draw
both|runtime/rand.go|cheaprand64|cheaprand64|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/rand.go|cheaprand64|cheaprandu64|call|1|host-timed|M-local|active|64-bit cheap stream is per-M on qualified architectures
both|runtime/rand.go|cheaprandn|cheaprand|call|1|target-ordered|wrapper|active|runtime helper routes P draws to process stream and no-P draws to M
both|runtime/rand.go|cheaprandn|cheaprandn|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/rand.go|cheaprandu64|cheaprand|call|2|host-timed|wrapper|disabled|Mul64 per-M branch always chosen on both qualified architectures
both|runtime/rand.go|cheaprandu64|cheaprand64|value|3|reference|none|active|matching field or local value; no draw
both|runtime/rand.go|cheaprandu64|cheaprandu64|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/rand.go|legacy_fastrand|legacy_fastrand|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/rand.go|legacy_fastrand|rand|call|1|target-ordered|wrapper|active|runtime helper routes P draws to process stream and no-P draws to M
both|runtime/rand.go|legacy_fastrand64|legacy_fastrand64|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/rand.go|legacy_fastrand64|rand|call|1|target-ordered|wrapper|active|runtime helper routes P draws to process stream and no-P draws to M
both|runtime/rand.go|legacy_fastrandn|legacy_fastrandn|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/rand.go|legacy_fastrandn|randn|call|1|target-ordered|wrapper|active|runtime helper routes P draws to process stream and no-P draws to M
both|runtime/rand.go|maps_rand|maps_rand|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/rand.go|maps_rand|rand|call|1|target-ordered|wrapper|active|runtime helper routes P draws to process stream and no-P draws to M
both|runtime/rand.go|mrandinit|bootstrapRand|call|1|host-timed|wrapper|disabled|gomadEnabled branch returns before bootstrap draws
both|runtime/rand.go|mrandinit|cheaprand|value|2|reference|none|active|matching field or local value; no draw
both|runtime/rand.go|mrandinit|cheaprand64|value|2|reference|none|active|matching field or local value; no draw
both|runtime/rand.go|mrandinit|rand|call|2|host-timed|wrapper|disabled|gomadEnabled branch returns before bootstrap draws
both|runtime/rand.go|package|cheaprand|linkname|1|reference|wrapper|active|reviewed linkname binding and target
both|runtime/rand.go|package|cheaprand64|linkname|1|reference|wrapper|active|reviewed linkname binding and target
both|runtime/rand.go|package|cheaprandn|linkname|1|reference|wrapper|active|reviewed linkname binding and target
both|runtime/rand.go|package|legacy_fastrand|linkname:runtime.fastrand|1|reference|wrapper|active|reviewed linkname binding and target
both|runtime/rand.go|package|legacy_fastrand64|linkname:runtime.fastrand64|1|reference|wrapper|active|reviewed linkname binding and target
both|runtime/rand.go|package|legacy_fastrandn|linkname:runtime.fastrandn|1|reference|wrapper|active|reviewed linkname binding and target
both|runtime/rand.go|package|maps_rand|linkname:internal/runtime/maps.rand|1|reference|wrapper|active|reviewed linkname binding and target
both|runtime/rand.go|package|rand|linkname|1|reference|wrapper|active|reviewed linkname binding and target
both|runtime/rand.go|package|randn|linkname|1|reference|wrapper|active|reviewed linkname binding and target
both|runtime/rand.go|rand|gomadRuntimeRand|call|1|target-ordered|process-purpose|active|P-owned runtime rand draw
both|runtime/rand.go|rand|rand|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/rand.go|rand32|rand|call|1|target-ordered|wrapper|active|runtime helper routes P draws to process stream and no-P draws to M
both|runtime/rand.go|rand32|rand32|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/rand.go|randn|rand|call|1|target-ordered|wrapper|active|runtime helper routes P draws to process stream and no-P draws to M
both|runtime/rand.go|randn|randn|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/runtime2.go|package|cheaprand|value|1|reference|none|active|matching field or local value; no draw
both|runtime/runtime2.go|package|cheaprand64|value|1|reference|none|active|matching field or local value; no draw
both|runtime/select.go|selectgo|gomadChoiceSelectSeeded|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|runtime/sema.go|method.queue|cheaprand|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|runtime/symtab.go|pcvalue|gomadHostCheapRandN|call|1|host-timed|M-local|active|host scheduling or profiling draw uses current M
both|runtime/time.go|method.adjust|gomadTimerRand|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|runtime/time.go|method.adjust|rand|value|1|reference|none|active|matching field or local value; no draw
both|runtime/time.go|method.less|rand|value|2|reference|none|active|matching field or local value; no draw
both|runtime/time.go|method.maybeAdd|gomadTimerRand|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|runtime/time.go|method.maybeAdd|rand|value|1|reference|none|active|matching field or local value; no draw
both|runtime/time.go|method.updateHeap|gomadTimerRand|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|runtime/time.go|method.updateHeap|rand|value|1|reference|none|active|matching field or local value; no draw
both|runtime/time.go|package|rand|value|1|reference|none|active|matching field or local value; no draw
both|sync/pool.go|method.Put|runtime_randn|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|sync/pool.go|package|runtime_randn|linkname:runtime.randn|1|reference|wrapper|active|reviewed linkname binding and target
both|sync/pool.go|runtime_randn|runtime_randn|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|unique/canonmap.go|newCanonMap|runtime_rand|call|1|target-ordered|process-purpose|active|draw follows target operation or deterministic scheduler decision
both|unique/canonmap.go|package|runtime_rand|linkname:runtime.rand|1|reference|wrapper|active|reviewed linkname binding and target
both|unique/canonmap.go|runtime_rand|runtime_rand|declaration|1|reference|wrapper|active|reviewed draw helper declaration
`

func parseReviewedDrawReferences(t *testing.T) map[string]drawReference {
	t.Helper()
	reviewed := make(map[string]drawReference)
	for _, line := range strings.Split(strings.TrimSpace(reviewedDrawReferences), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "|")
		if len(fields) != 10 {
			t.Fatalf("invalid draw inventory row %q", line)
		}
		count, err := strconv.Atoi(fields[5])
		if err != nil || count <= 0 {
			t.Fatalf("invalid draw inventory count %q", line)
		}
		platforms := []string{fields[0]}
		if fields[0] == "both" {
			platforms = gomadversion.SupportedPlatforms[:]
		}
		for _, platform := range platforms {
			ref := drawReference{platform, fields[1], fields[2], fields[3], fields[4], count, fields[6], fields[7], fields[8], fields[9]}
			if _, duplicate := reviewed[ref.key()]; duplicate {
				t.Fatalf("duplicate draw inventory row %s", ref.key())
			}
			if ref.class == "" || ref.stream == "" || ref.reachability == "" || ref.reason == "" {
				t.Fatalf("incomplete draw inventory row %s", ref.key())
			}
			if !slices.Contains([]string{"target-ordered", "host-timed", "reference"}, ref.class) ||
				!slices.Contains([]string{"process-purpose", "M-local", "wrapper", "none"}, ref.stream) ||
				!slices.Contains([]string{"active", "disabled", "contract-excluded", "blocked"}, ref.reachability) {
				t.Fatalf("invalid draw inventory classification %s", ref.key())
			}
			if ref.class == "host-timed" && ref.reachability == "active" && ref.stream != "M-local" {
				t.Fatalf("active host-timed draw must use an M-local stream: %s", ref.key())
			}
			if ref.reachability == "blocked" && !strings.Contains(ref.reason, "collector edit prohibited") {
				t.Fatalf("blocked draw needs its policy blocker: %s", ref.key())
			}
			reviewed[ref.key()] = ref
		}
	}
	return reviewed
}

func TestPatchedRuntimeDrawReferencesAreReviewed(t *testing.T) {
	got, err := runtimeDrawReferences(builtGOROOT(t))
	if err != nil {
		t.Fatal(err)
	}
	problems := drawInventoryProblems(got, parseReviewedDrawReferences(t))
	if len(problems) != 0 {
		t.Fatalf("patched runtime draw inventory drifted; review every reference:\n%s", strings.Join(problems, "\n"))
	}
}

func TestDrawInventoryRejectsUnclassifiedReference(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "src", "runtime")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "inventory_scratch.go"), []byte("package runtime\nfunc unreviewedDraw() { cheaprand() }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	aliasDirectory := filepath.Join(root, "src", "drawalias")
	if err := os.MkdirAll(aliasDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(aliasDirectory, "alias.go"), []byte("package drawalias\n//go:linkname renamedDraw runtime.rand\nfunc renamedDraw() uint64\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(aliasDirectory, "use.go"), []byte("package drawalias\nvar borrowedDraw = renamedDraw\nfunc useDraw() { renamedDraw() }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := runtimeDrawReferences(root)
	if err != nil {
		t.Fatal(err)
	}
	problems := drawInventoryProblems(got, map[string]drawReference{})
	if !slices.ContainsFunc(problems, func(problem string) bool {
		return strings.Contains(problem, "unreviewed draw reference: darwin/arm64 runtime/inventory_scratch.go unreviewedDraw cheaprand call")
	}) {
		t.Fatalf("scratch draw did not fail closed: %v", problems)
	}
	if !slices.ContainsFunc(problems, func(problem string) bool {
		return strings.Contains(problem, "unreviewed draw reference: darwin/arm64 drawalias/use.go useDraw renamedDraw call")
	}) {
		t.Fatalf("aliased linkname draw did not fail closed: %v", problems)
	}
	if !slices.ContainsFunc(problems, func(problem string) bool {
		return strings.Contains(problem, "unreviewed draw reference: darwin/arm64 drawalias/use.go package renamedDraw value")
	}) {
		t.Fatalf("aliased function value did not fail closed: %v", problems)
	}
}

func drawInventoryProblems(got map[string]int, reviewed map[string]drawReference) []string {
	var problems []string
	for key, count := range got {
		reference, found := reviewed[key]
		if !found {
			problems = append(problems, fmt.Sprintf("unreviewed draw reference: %s (%d)", key, count))
		} else if reference.count != count {
			problems = append(problems, fmt.Sprintf("draw reference count changed: %s = %d, reviewed %d", key, count, reference.count))
		}
	}
	for key := range reviewed {
		if _, found := got[key]; !found {
			problems = append(problems, "reviewed draw reference disappeared: "+key)
		}
	}
	slices.Sort(problems)
	return problems
}

var drawSymbols = strings.Fields("bootstrapRand rand rand32 randn cheaprand cheaprand64 cheaprandu64 cheaprandn legacy_fastrand legacy_fastrand64 legacy_fastrandn gomadRuntimeRand gomadRuntimeCheapRand gomadHostCheapRand gomadHostCheapRandN gomadLockProfileStart gomadTimerRand gomadChoiceRunqSeeded gomadChoiceRunnextSeeded gomadChoiceShuffleSeeded gomadChoiceSelectSeeded gomadChoiceRandom gomadClockTickDraw runtime_rand runtime_randn")

func runtimeDrawReferences(goRoot string) (map[string]int, error) {
	references := map[string]int{}
	sourceRoot := filepath.Join(goRoot, "src")
	for _, platform := range gomadversion.SupportedPlatforms {
		goos, goarch, _ := strings.Cut(platform, "/")
		context := build.Default
		context.GOROOT, context.GOOS, context.GOARCH, context.CgoEnabled = goRoot, goos, goarch, false
		var tags []string
		for _, tag := range context.ToolTags {
			if tag != "goexperiment.greenteagc" {
				tags = append(tags, tag)
			}
		}
		context.ToolTags = tags
		type sourceFile struct {
			file       *ast.File
			relative   string
			packageKey string
		}
		var sources []sourceFile
		aliasesByPackage := map[string]map[string]string{}
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
			if err != nil {
				return err
			}
			if !match && !strings.Contains(name, "greenteagc") {
				return nil
			}
			fileSet := token.NewFileSet()
			file, err := parser.ParseFile(fileSet, path, nil, parser.ParseComments)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(sourceRoot, path)
			if err != nil {
				return err
			}
			rel := filepath.ToSlash(relative)
			packageKey := filepath.Dir(path) + "\x00" + file.Name.Name
			sources = append(sources, sourceFile{file: file, relative: rel, packageKey: packageKey})
			aliases := aliasesByPackage[packageKey]
			if aliases == nil {
				aliases = map[string]string{}
				aliasesByPackage[packageKey] = aliases
			}
			for _, group := range file.Comments {
				for _, comment := range group.List {
					fields := strings.Fields(comment.Text)
					if len(fields) >= 3 && fields[0] == "//go:linkname" && seededLinknameTarget(fields[2]) {
						aliases[fields[1]] = fields[2]
					}
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		for _, source := range sources {
			file, rel := source.file, source.relative
			aliases := aliasesByPackage[source.packageKey]
			isDrawSymbol := func(symbol string) bool {
				if _, linked := aliases[symbol]; linked {
					return true
				}
				return slices.Contains(drawSymbols, symbol) && (strings.HasPrefix(rel, "runtime/") || strings.HasPrefix(rel, "internal/runtime/maps/") || symbol == "runtime_rand" || symbol == "runtime_randn")
			}
			add := func(function, symbol, kind string) {
				if isDrawSymbol(symbol) {
					key := drawReference{platform: platform, file: rel, function: function, symbol: symbol, kind: kind}.key()
					references[key]++
				}
			}
			for _, group := range file.Comments {
				for _, comment := range group.List {
					fields := strings.Fields(comment.Text)
					if len(fields) >= 2 && fields[0] == "//go:linkname" {
						kind := "linkname"
						if len(fields) >= 3 {
							kind += ":" + fields[2]
						}
						add("package", fields[1], kind)
					}
				}
			}
			for _, declaration := range file.Decls {
				function := "package"
				if fn, ok := declaration.(*ast.FuncDecl); ok {
					function = fn.Name.Name
					if fn.Recv != nil {
						function = "method." + function
					}
					add(function, fn.Name.Name, "declaration")
				}
				calls := map[token.Pos]bool{}
				ast.Inspect(declaration, func(node ast.Node) bool {
					if call, ok := node.(*ast.CallExpr); ok {
						if id, ok := call.Fun.(*ast.Ident); ok {
							calls[id.Pos()] = true
						}
					}
					return true
				})
				ast.Inspect(declaration, func(node ast.Node) bool {
					id, ok := node.(*ast.Ident)
					if !ok || !isDrawSymbol(id.Name) {
						return true
					}
					if fn, ok := declaration.(*ast.FuncDecl); ok && id.Pos() == fn.Name.Pos() {
						return true
					}
					kind := "value"
					if calls[id.Pos()] {
						kind = "call"
					}
					add(function, id.Name, kind)
					return true
				})
			}
		}
	}
	return references, nil
}

func seededLinknameTarget(target string) bool {
	if target == "internal/runtime/maps.rand" {
		return true
	}
	symbol, ok := strings.CutPrefix(target, "runtime.")
	return ok && (slices.Contains(drawSymbols, symbol) || slices.Contains([]string{"fastrand", "fastrandn", "fastrand64"}, symbol))
}

// With GOMADSEED set, every draw the runtime rand helpers serve to an M holding
// the P comes from process-wide state derived from the seed, so a draw taken at
// a moment chosen by host timing moves every later seeded decision. Every
// reference to a helper that reaches that state, or to the M-local helpers
// that host-timed sites use instead, is pinned here with its classification; a
// new upstream or Gomad reference fails the toolchain tier on either host.
//
// A reference is classified by the function that contains it, not by that
// function's callers. A helper reached from both target-ordered and host-timed
// callers, such as runqputbatch, keeps its target-ordered entry only because
// its host-timed callers switch the draw to an M-local helper themselves; the
// entry's reason names that caller-side switch, and this scan cannot see it.
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

// drawSeededStates are the process-wide seeded states. Only their accessors and
// the functions that seed them may name them, as drawImplementation entries; a
// direct use elsewhere would draw without the accessor's check.
var drawSeededStates = []string{
	"gomadChoiceRunqRandom", "gomadChoiceSchedulerRandom", "gomadChoiceSelectRandom", "gomadClockTickState",
	"gomadRuntimeCheapRandom", "gomadRuntimeRandom", "gomadTimerRandom",
}

// drawCompilerHelpers are the runtime helpers the compiler calls by name.
var drawCompilerHelpers = []string{"rand", "rand32"}

type seededDrawReference struct {
	file     string
	function string
	helper   string
	count    int
	class    drawClass
	reason   string
}

func (r seededDrawReference) key() string {
	return r.file + " " + r.function + " " + r.helper
}

var reviewedSeededDrawReferences = []seededDrawReference{
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
	{"runtime/gomad.go", "gomadChoiceRunnextSeeded", "gomadChoiceSchedulerRandom", 1, drawImplementation, "the runnext demotion stream's accessor"},
	{"runtime/gomad.go", "gomadChoiceRunnextSeeded", "randn", 1, drawInactive, "upstream draw when Gomad is disabled"},
	{"runtime/gomad.go", "gomadChoiceRunqIndex", "gomadChoiceRunqSeeded", 1, drawTargetOrdered, "the run-queue pick among user goroutines; runtime-owned goroutines take no draw"},
	{"runtime/gomad.go", "gomadChoiceRunqSeeded", "gomadChoiceRandom", 1, drawImplementation, "the run-queue pick draw"},
	{"runtime/gomad.go", "gomadChoiceRunqSeeded", "gomadChoiceRunqRandom", 1, drawImplementation, "the run-queue pick stream's accessor"},
	{"runtime/gomad.go", "gomadChoiceRunqSeeded", "randn", 1, drawInactive, "upstream draw when Gomad is disabled"},
	{"runtime/gomad.go", "gomadChoiceSeedRandom", "gomadChoiceRunqRandom", 1, drawImplementation, "seeds the stream from GOMADSEED before user code"},
	{"runtime/gomad.go", "gomadChoiceSeedRandom", "gomadChoiceSchedulerRandom", 1, drawImplementation, "seeds the stream from GOMADSEED before user code"},
	{"runtime/gomad.go", "gomadChoiceSeedRandom", "gomadChoiceSelectRandom", 1, drawImplementation, "seeds the stream from GOMADSEED before user code"},
	{"runtime/gomad.go", "gomadChoiceSeedRandom", "gomadRuntimeCheapRandom", 1, drawImplementation, "seeds the stream from GOMADSEED before user code"},
	{"runtime/gomad.go", "gomadChoiceSeedRandom", "gomadRuntimeRandom", 1, drawImplementation, "seeds the stream from GOMADSEED before user code"},
	{"runtime/gomad.go", "gomadChoiceSeedRandom", "gomadTimerRandom", 1, drawImplementation, "seeds the stream from GOMADSEED before user code"},
	{"runtime/gomad.go", "gomadChoiceSelectSeeded", "cheaprandn", 1, drawInactive, "upstream draw when Gomad is disabled"},
	{"runtime/gomad.go", "gomadChoiceSelectSeeded", "gomadChoiceSelectRandom", 4, drawImplementation, "the select poll-order stream's accessor"},
	{"runtime/gomad.go", "gomadChoiceShuffleSeeded", "cheaprandn", 1, drawInactive, "upstream draw when Gomad is disabled"},
	{"runtime/gomad.go", "gomadChoiceShuffleSeeded", "gomadChoiceRandom", 1, drawImplementation, "the run-queue batch shuffle draw"},
	{"runtime/gomad.go", "gomadChoiceShuffleSeeded", "gomadChoiceSchedulerRandom", 1, drawImplementation, "the batch shuffle stream's accessor"},
	{"runtime/gomad.go", "gomadChoiceShuffleSeeded", "gomadHostCheapRandN", 1, drawHostTimed, "the shuffle of a netpoll batch injected through gomadInjectHostList, whose contents and moment are host-chosen; the injection is bracketed for the diagnostic check"},
	{"runtime/gomad.go", "gomadClockTickDraw", "gomadClockTickState", 2, drawImplementation, "the forward clock tick stream's accessor"},
	{"runtime/gomad.go", "gomadClockTickInit", "gomadClockTickState", 1, drawImplementation, "seeds the stream from GOMADSEED before user code"},
	{"runtime/gomad.go", "gomadDiagnosticAppend", "gomadRuntimeCheapRand", 1, drawDiagnostic, "GOMAD3_DIAGNOSTIC_PERTURB_DRAW=N takes one extra seeded draw at choice record N"},
	{"runtime/gomad.go", "gomadHostCheapRand", "gomadRuntimeCheapRand", 1, drawDiagnostic, "GOMAD3_DIAGNOSTIC_PERTURB_DRAW=host-timed:N puts the next host-timed draw on the seeded stream, which the diagnostic check refuses"},
	{"runtime/gomad.go", "gomadHostCheapRandN", "gomadHostCheapRand", 1, drawImplementation, "bounded form of the M-local draw"},
	{"runtime/gomad.go", "gomadLockProfileStart", "gomadHostCheapRandN", 1, drawHostTimed, "lock-profile wait sampling where lock2 is about to sleep on a contended runtime lock"},
	{"runtime/gomad.go", "gomadRuntimeCheapRand", "gomadRuntimeCheapRandom", 3, drawImplementation, "the process-wide cheaprand stream's accessor"},
	{"runtime/gomad.go", "gomadRuntimeRand", "gomadRuntimeRandom", 2, drawImplementation, "the process-wide rand stream's accessor"},
	{"runtime/gomad.go", "gomadTimeNow", "gomadClockTickDraw", 1, drawTargetOrdered, "the forward clock tick of a time.Now the target calls"},
	{"runtime/gomad.go", "gomadTimerRand", "gomadTimerRandom", 2, drawImplementation, "the timer tie-break stream's accessor"},
	{"runtime/iface.go", "interfaceSwitch", "cheaprand", 2, drawTargetOrdered, "type-switch cache growth on a switch the target executes"},
	{"runtime/iface.go", "typeAssert", "cheaprand", 2, drawTargetOrdered, "type-assertion cache growth on an assertion the target executes"},
	{"runtime/lock_spinbit.go", "mutexSampleContention", "cheaprandu64", 1, drawHostTimed, "runtime-lock contention sampling for the mutex profile; m.cheaprand64 is M-local"},
	{"runtime/lock_spinbit.go", "unlock2Wake", "gomadHostCheapRandN", 1, drawHostTimed, "anti-starvation wake of the M at the bottom of a contended lock's wait stack"},
	{"runtime/malloc.go", "fastexprand", "cheaprandn", 1, drawTargetOrdered, "heap-profile sampling of target allocations; Gomad sets MemProfileRate to zero when user code starts"},
	{"runtime/malloc.go", "mallocinit", "bootstrapRand", 1, drawTargetOrdered, "heap base randomization during runtime initialization, before user code"},
	{"runtime/mbitmap.go", "doubleCheckHeapType", "cheaprand", 2, drawTargetOrdered, "compiled only under the false doubleCheckHeapSetType debug constant; the allocation reaching it is the target's"},
	{"runtime/mgcpacer.go", "gcControllerState.enlistWorker", "cheaprandn", 1, drawHostTimedBlocked, "picks a P to preempt for collector work at a moment the collector chooses; mgcpacer.go is a prohibited collector file, so a reroute is referred to the patch-policy owner (fn-112 Open Question 3); unreachable while GOMAXPROCS stays one (Gomad starts with one, and raising it is unsupported), because enlistWorker returns before the draw"},
	{"runtime/mprof.go", "blocksampled", "cheaprand64", 1, drawHostTimed, "block-profile sampling of a blocking event's host-measured cycles; m.cheaprand64 is M-local"},
	{"runtime/mprof.go", "mLockProfile.recordUnlock", "cheaprandu64", 2, drawHostTimed, "chooses which contended runtime-lock stack the mutex profile keeps; m.cheaprand64 is M-local"},
	{"runtime/mprof.go", "mLockProfile.start", "cheaprandn", 1, drawInactive, "gomadLockProfileStart calls it only when Gomad is disabled"},
	{"runtime/mprof.go", "mutexevent", "cheaprand64", 1, drawHostTimed, "mutex-profile sampling of a contention event's host-measured cycles; m.cheaprand64 is M-local"},
	{"runtime/os_linux.go", "setThreadCPUProfiler", "gomadHostCheapRandN", 1, drawHostTimed, "spreads per-thread CPU-profile timers from the M-local stream when an M first runs after SetCPUProfileRate"},
	{"runtime/proc.go", "newproc1", "cheaprand", 1, drawTargetOrdered, "the scheduler-tracking sequence of a goroutine the target or a runtime goroutine creates"},
	{"runtime/proc.go", "runqput", "gomadChoiceRunnextSeeded", 1, drawTargetOrdered, "the runnext demotion of a goroutine readied on the P"},
	{"runtime/proc.go", "runqputbatch", "gomadChoiceShuffleSeeded", 1, drawTargetOrdered, "shuffles a target batch put on the local run queue; a host batch takes the separate M-local branch"},
	{"runtime/proc.go", "runqputbatch", "gomadHostCheapRandN", 1, drawHostTimed, "a host batch from netpoll shuffles on the M-local stream"},
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
	if problems := seededDrawProblems(got, reviewedSeededDrawReferences); len(problems) > 0 {
		t.Fatalf("patched runtime seeded-stream draw inventory drifted; classify each change and update reviewedSeededDrawReferences:\n%s", strings.Join(problems, "\n"))
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
	reviewed := []seededDrawReference{
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
		change  func(*seededDrawReference)
		problem string
	}{
		{"host-timed on the seeded stream", func(r *seededDrawReference) { r.helper, r.class = "cheaprand", drawHostTimed }, "host-timed reference uses a seeded helper: runtime/gomad_unclassified.go unclassified cheaprand"},
		{"blocked without a finding", func(r *seededDrawReference) { r.helper, r.class, r.reason = "cheaprand", drawHostTimedBlocked, "" }, "classified seeded-stream reference has no reason: runtime/gomad_unclassified.go unclassified cheaprand"},
		{"seeded state outside its accessor", func(r *seededDrawReference) { r.helper, r.class = "gomadRuntimeRandom", drawTargetOrdered }, "seeded state is used outside its accessor or seeding function: runtime/gomad_unclassified.go unclassified gomadRuntimeRandom"},
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
	helpers := map[string][]string{filepath.Join(sourceRoot, "runtime"): slices.Concat(drawRuntimeHelpers, drawSeededStates)}
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
// and the declared names of the helper and of package variables.
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
			case *ast.ValueSpec:
				// A declared name is not a use; its type and initial values are.
				if node.Type != nil {
					ast.Inspect(node.Type, func(inner ast.Node) bool { return countDrawIdentifier(inner, relative, function, helpers, counts) })
				}
				for _, value := range node.Values {
					ast.Inspect(value, func(inner ast.Node) bool { return countDrawIdentifier(inner, relative, function, helpers, counts) })
				}
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

// countCompilerDrawCalls counts the runtime rand helpers the compiler calls by
// name anywhere in cmd/compile, such as the seed of a non-escaping small map
// in its walk phase. The typecheck builtin table only declares the runtime
// functions the compiler may call, so it is not a call site.
func countCompilerDrawCalls(sourceRoot string, counts map[string]int) error {
	directory := filepath.Join(sourceRoot, "cmd", "compile")
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		return nil
	}
	builtinTable := filepath.Join(directory, "internal", "typecheck", "builtin.go")
	return filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := entry.Name()
		if entry.IsDir() {
			if name == "testdata" || name == "_builtin" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || path == builtinTable {
			return nil
		}
		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
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
					counts[relative+" "+definition.Name.Name+" "+helper]++
				}
				return true
			})
		}
		return nil
	})
}

// seededDrawProblems compares the scanned references with the reviewed
// inventory; an empty result means every reference is classified, every
// classification carries its reason, and every host-timed reference that is
// not blocked uses an M-local helper.
func seededDrawProblems(got map[string]int, reviewed []seededDrawReference) []string {
	var problems []string
	want := map[string]seededDrawReference{}
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
		if slices.Contains(drawSeededStates, reference.helper) && reference.class != drawImplementation {
			problems = append(problems, "seeded state is used outside its accessor or seeding function: "+reference.key())
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
