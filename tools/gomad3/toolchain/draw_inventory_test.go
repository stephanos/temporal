package toolchain

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
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
both|runtime/gomad.go|gomadChoiceShuffleSeeded|gomadChoiceShuffleSeeded|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/gomad.go|gomadClockTickDraw|gomadClockTickDraw|declaration|1|reference|wrapper|active|reviewed draw helper declaration
both|runtime/gomad.go|gomadDiagnosticAppend|gomadRuntimeCheapRand|call|1|host-timed|process-purpose|contract-excluded|numeric diagnostic fault injection; host prefix guard exits before mutation
both|runtime/gomad.go|gomadHostCheapRand|cheaprand|value|3|reference|none|active|matching field or local value; no draw
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
