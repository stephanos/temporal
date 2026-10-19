package determinism

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/toolchain"
)

// The wasip1 runtime rows reuse the reviewed row grammar of the native
// toolchain inventories (draw_inventory_test.go and clock_inventory_test.go
// in internal/backends/native/toolchain), reimplemented here so the WASM
// backend keeps its own gates; the native tables are not read or edited.

const wasip1 = "wasip1/wasm"

// Draw rows: platform|file|function|symbol|kind|count|class|stream|reachability|reason.
// Every overlay patch that introduces or replaces a seeded draw or a clock
// path has a row; kind is "patch" and count is the patch's occurrence count.
// Clock patches use class "reference" and stream "none" because they take no
// draw; their clock reads are pinned by the host-clock rows below.
const reviewedWasip1DrawPatches = `
wasip1/wasm|runtime/lock_wasip1.go|beforeIdle|beforeIdle|patch|1|reference|none|active|idle hook hands the next virtual timer deadline to the helper's idle import instead of sleeping on the host
wasip1/wasm|runtime/lock_wasip1.go|notetsleepg|notetsleepg|patch|1|reference|none|active|note sleeps park on the virtual clock through gomadWasmNoteSleep; no host sleep
wasip1/wasm|runtime/os_wasip1.go|nanotime1|gomadWasmLastTime|patch|2|reference|none|active|records the last transcribed clock_time_get answer, and inspection returns it instead of issuing a clock import
wasip1/wasm|runtime/os_wasip1.go|usleep|usleep|patch|1|reference|none|active|inspection does not sleep; outside inspection usleep is the stock poll_oneoff on the virtual clock
wasip1/wasm|runtime/os_wasip1.go|walltime1|gomadWasmLastTime|patch|1|reference|none|active|inspection returns the last virtual time instead of issuing a clock import
wasip1/wasm|runtime/proc.go|runqget|gomadChoiceRunqIndex|patch|1|target-ordered|process-purpose|active|run queue pick is a seeded choice recorded on the choice tape
wasip1/wasm|runtime/proc.go|runqputslow|gomadChoiceRunnextSeeded|patch|2|target-ordered|process-purpose|active|batch shuffle draws come from the seeded scheduler stream
wasip1/wasm|runtime/proc.go|runqputslow|randomizeScheduler|patch|2|target-ordered|process-purpose|active|batch shuffling is forced on under Gomad so every batch order is a seeded draw
wasip1/wasm|runtime/proc.go|schedinit|gomadWasmInit|patch|1|target-ordered|process-purpose|active|seed and diagnostic flag come from the config import before randinit
wasip1/wasm|runtime/rand.go|cheaprand|gomadRuntimeCheapRand|patch|1|target-ordered|process-purpose|active|cheaprand on a P draws from the seeded runtime cheaprand stream
wasip1/wasm|runtime/rand.go|mrandinit|gomadSeed|patch|1|target-ordered|M-local|active|the single M's per-M streams are seeded from the declared seed
wasip1/wasm|runtime/rand.go|rand|gomadRuntimeRand|patch|1|target-ordered|process-purpose|active|rand on a P draws from the seeded runtime rand stream
wasip1/wasm|runtime/rand.go|randinit|gomadSeed|patch|1|target-ordered|process-purpose|active|the global seed is the declared seed instead of startup entropy
wasip1/wasm|runtime/select.go|selectgo|gomadChoiceSelectSeeded|patch|1|target-ordered|process-purpose|active|select poll order is a seeded draw recorded on the choice tape
wasip1/wasm|runtime/time.go|method.adjust|gomadTimerRand|patch|1|target-ordered|process-purpose|active|modified timers redraw their heap tie-break from the seeded timer stream
wasip1/wasm|runtime/time.go|method.maybeAdd|gomadTimerRand|patch|1|target-ordered|process-purpose|active|timer heap tie-break draws from the seeded timer stream
wasip1/wasm|runtime/time.go|method.updateHeap|gomadTimerRand|patch|1|target-ordered|process-purpose|active|re-queued timers redraw their heap tie-break from the seeded timer stream
`

// Host-clock rows: platform|file|identifier|count|disposition|finding, over
// every standard-library and overlay file that builds for wasip1/wasm. On
// wasip1 the only host clock entry points are the clock_time_get and
// poll_oneoff WASI imports, which the helper answers from its virtual clock
// model and transcribes; disposition "virtual" marks a read that lands there.
const reviewedWasip1HostClockReferences = `
wasip1/wasm|internal/trace/internal/testgen/trace.go|walltime|4|unrelated|
wasip1/wasm|runtime/badlinkname.go|cputicks|1|implementation|
wasip1/wasm|runtime/chan.go|cputicks|6|virtual|block profiling samples waits with cputicks, which is nanotime on wasm and so the transcribed virtual clock
wasip1/wasm|runtime/debuglog.go|cputicks|1|inactive|debuglog_off selects the no-op logger for the qualified target configuration
wasip1/wasm|runtime/gomad_wasm.go|nanotime|1|virtual|note sleeps compute their virtual deadline from the transcribed clock
wasip1/wasm|runtime/mgc.go|time_now|1|virtual|GC LastGC and PauseEnd stamps read the virtual wall clock; GC timing is allocation-driven on the single seeded thread
wasip1/wasm|runtime/os_wasip1.go|clock_time_get|6|implementation|
wasip1/wasm|runtime/os_wasip1.go|nanotime1|1|implementation|
wasip1/wasm|runtime/os_wasip1.go|walltime|1|implementation|
wasip1/wasm|runtime/os_wasip1.go|walltime1|2|implementation|
wasip1/wasm|runtime/os_wasm.go|cputicks|1|implementation|
wasip1/wasm|runtime/os_wasm.go|time_now|1|virtual|syscall.now reads the virtual wall clock
wasip1/wasm|runtime/runtime.go|cputicks|2|virtual|profile ticks-per-second conversion reads the virtual clock through nanotime
wasip1/wasm|runtime/select.go|cputicks|1|virtual|block profiling samples waits with the virtual clock
wasip1/wasm|runtime/sema.go|cputicks|5|virtual|block and mutex profiling sample waits with the virtual clock
wasip1/wasm|runtime/time.go|time_now|2|virtual|time_runtimeNow and the FIPS monoTime read the virtual clock; no unguarded host read exists on wasip1
wasip1/wasm|runtime/time_nofake.go|nanotime1|1|implementation|
wasip1/wasm|runtime/timestub.go|time_now|2|implementation|
wasip1/wasm|runtime/timestub.go|walltime|1|implementation|
wasip1/wasm|runtime/tracetime.go|cputicks|1|inactive|the branch is compile-time unreachable because osHasLowResClock is false on wasip1
wasip1/wasm|runtime/tracetime.go|time_now|1|virtual|the execution tracer's clock snapshot reads the virtual clock
wasip1/wasm|syscall/syscall_wasip1.go|clock_time_get|3|virtual|syscall.Gettimeofday and friends call the clock_time_get import, answered by the helper's virtual clock
`

var (
	wasip1ClockIdentifier = regexp.MustCompile(`\b(cputicks|libc_gettimeofday|libc_gettimeofday_trampoline|nanotime1|walltime|walltime1|time_now|vdsoClockgettimeSym|vdsoGettimeofdaySym|clock_time_get)\b`)
	// The glue file also reads the runtime clock directly through nanotime.
	wasip1GlueClockIdentifier  = regexp.MustCompile(`\b(nanotime)\b`)
	wasip1DrawPatchIdentifier  = regexp.MustCompile(`\b(cheaprand|cheaprandn|cheaprand64|rand|randn|mrandinit|randinit|startupRand|randomizeScheduler|gomadSeed|gomadWasmInit|gomadChoiceRunnextSeeded|gomadChoiceRunqIndex|gomadChoiceSelectSeeded|gomadRuntimeRand|gomadRuntimeCheapRand|gomadTimerRand)\b`)
	wasip1ClockPatchIdentifier = regexp.MustCompile(`\b(nanotime|nanotime1|walltime|walltime1|usleep|notetsleepg|beforeIdle|time_now|cputicks|gomadWasmLastTime)\b`)
)

var (
	drawClasses       = []string{"target-ordered", "host-timed", "reference"}
	drawStreams       = []string{"process-purpose", "M-local", "wrapper", "none"}
	drawReachability  = []string{"active", "disabled", "contract-excluded", "blocked"}
	clockDispositions = []string{"implementation", "virtual", "inactive", "host-by-design", "escape", "unrelated"}
)

type drawRow struct {
	key                                 string
	count                               int
	class, stream, reachability, reason string
}

type clockRow struct {
	key, disposition, finding string
	count                     int
}

func parseDrawRows(table string) (map[string]drawRow, error) {
	rows := map[string]drawRow{}
	for line := range strings.Lines(strings.TrimSpace(table)) {
		fields := strings.Split(strings.TrimSpace(line), "|")
		if len(fields) != 10 || fields[0] != wasip1 {
			return nil, fmt.Errorf("invalid draw row %q", line)
		}
		count, err := strconv.Atoi(fields[5])
		if err != nil || count <= 0 {
			return nil, fmt.Errorf("invalid draw row count %q", line)
		}
		row := drawRow{strings.Join(fields[:5], " "), count, fields[6], fields[7], fields[8], fields[9]}
		if _, duplicate := rows[row.key]; duplicate {
			return nil, fmt.Errorf("duplicate draw row %s", row.key)
		}
		if !slices.Contains(drawClasses, row.class) || !slices.Contains(drawStreams, row.stream) || !slices.Contains(drawReachability, row.reachability) || row.reason == "" {
			return nil, fmt.Errorf("invalid draw row classification %s", row.key)
		}
		if row.class == "host-timed" && row.reachability == "active" && row.stream != "M-local" {
			return nil, fmt.Errorf("active host-timed draw must use an M-local stream: %s", row.key)
		}
		rows[row.key] = row
	}
	return rows, nil
}

func parseClockRows(table string) (map[string]clockRow, error) {
	rows := map[string]clockRow{}
	for line := range strings.Lines(strings.TrimSpace(table)) {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(strings.TrimSpace(line), "|")
		if len(fields) != 6 || fields[0] != wasip1 {
			return nil, fmt.Errorf("invalid host-clock row %q", line)
		}
		count, err := strconv.Atoi(fields[3])
		if err != nil || count <= 0 {
			return nil, fmt.Errorf("invalid host-clock row count %q", line)
		}
		row := clockRow{strings.Join(fields[:3], " "), fields[4], fields[5], count}
		if _, duplicate := rows[row.key]; duplicate {
			return nil, fmt.Errorf("duplicate host-clock row %s", row.key)
		}
		if !slices.Contains(clockDispositions, row.disposition) || (row.disposition == "escape" || row.disposition == "inactive" || row.disposition == "host-by-design") && row.finding == "" {
			return nil, fmt.Errorf("invalid host-clock row %s", row.key)
		}
		rows[row.key] = row
	}
	return rows, nil
}

// referenceProblems compares derived counts with reviewed rows.
func referenceProblems(kind string, got map[string]int, reviewed map[string]int) []string {
	var problems []string
	for key, count := range got {
		want, ok := reviewed[key]
		switch {
		case !ok:
			problems = append(problems, fmt.Sprintf("unreviewed %s: %s (%d)", kind, key, count))
		case want != count:
			problems = append(problems, fmt.Sprintf("%s count changed: %s = %d, reviewed %d", kind, key, count, want))
		}
	}
	for key := range reviewed {
		if _, ok := got[key]; !ok {
			problems = append(problems, fmt.Sprintf("reviewed %s disappeared: %s", kind, key))
		}
	}
	slices.Sort(problems)
	return problems
}

type overlayPatch struct {
	file, from, to string
	count          int
}

// overlayPatches reads stockRewrites from the WASM overlay source.
func overlayPatches(t *testing.T) []overlayPatch {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "../toolchain/overlay.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	constants := map[string]string{}
	var rewrites *ast.CompositeLit
	for _, declaration := range file.Decls {
		switch declaration := declaration.(type) {
		case *ast.GenDecl:
			for _, spec := range declaration.Specs {
				if value, ok := spec.(*ast.ValueSpec); ok && declaration.Tok == token.CONST && len(value.Values) == len(value.Names) {
					for index, name := range value.Names {
						if literal, ok := value.Values[index].(*ast.BasicLit); ok && literal.Kind == token.STRING {
							constants[name.Name], _ = strconv.Unquote(literal.Value)
						}
					}
				}
			}
		case *ast.FuncDecl:
			if declaration.Name.Name == "stockRewrites" {
				ast.Inspect(declaration.Body, func(node ast.Node) bool {
					if composite, ok := node.(*ast.CompositeLit); ok && rewrites == nil {
						rewrites = composite
						return false
					}
					return true
				})
			}
		}
	}
	if rewrites == nil {
		t.Fatal("overlay.go has no stockRewrites map")
	}
	text := func(expression ast.Expr) string {
		switch expression := expression.(type) {
		case *ast.BasicLit:
			value, err := strconv.Unquote(expression.Value)
			if err != nil {
				t.Fatal(err)
			}
			return value
		case *ast.Ident:
			value, ok := constants[expression.Name]
			if !ok {
				t.Fatalf("overlay rewrite uses unknown constant %s", expression.Name)
			}
			return value
		}
		t.Fatalf("overlay rewrite operand %T not understood", expression)
		return ""
	}
	var patches []overlayPatch
	for _, element := range rewrites.Elts {
		pair := element.(*ast.KeyValueExpr)
		name := text(pair.Key)
		for _, rule := range pair.Value.(*ast.CompositeLit).Elts {
			fields := rule.(*ast.CompositeLit).Elts
			if len(fields) != 3 {
				t.Fatalf("overlay rewrite for %s has %d fields", name, len(fields))
			}
			count, err := strconv.Atoi(fields[2].(*ast.BasicLit).Value)
			if err != nil {
				t.Fatal(err)
			}
			patches = append(patches, overlayPatch{name, text(fields[0]), text(fields[1]), count})
		}
	}
	return patches
}

// stockPath maps an overlay output name to its path beneath GOROOT/src, as
// overlay.go's sourcePath does.
func stockPath(name string) string {
	if strings.Contains(name, "/") {
		return name
	}
	return "runtime/" + name
}

func enclosingFunction(source []byte, offset int) (string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), "", source, parser.SkipObjectResolution)
	if err != nil {
		return "", err
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || int(function.Pos())-1 > offset || offset >= int(function.End())-1 {
			continue
		}
		if function.Recv != nil {
			return "method." + function.Name.Name, nil
		}
		return function.Name.Name, nil
	}
	return "package", nil
}

// drawPatchReferences derives one draw-grammar key per overlay patch that
// touches a seeded draw or a clock path.
func drawPatchReferences(t *testing.T, stock string, patches []overlayPatch) map[string]int {
	t.Helper()
	got := map[string]int{}
	for _, patch := range patches {
		var symbol string
		for _, pattern := range []*regexp.Regexp{wasip1DrawPatchIdentifier, wasip1ClockPatchIdentifier} {
			if symbol != "" {
				break
			}
			// Prefer the Gomad identifier the patch introduces.
			for _, candidate := range pattern.FindAllString(patch.to, -1) {
				if !regexp.MustCompile(`\b`+candidate+`\b`).MatchString(patch.from) && (symbol == "" || !strings.HasPrefix(symbol, "gomad") && strings.HasPrefix(candidate, "gomad")) {
					symbol = candidate
				}
			}
			if symbol == "" && pattern.MatchString(patch.from) {
				symbol = pattern.FindString(patch.from)
			}
		}
		if symbol == "" {
			continue
		}
		source, err := os.ReadFile(filepath.Join(stock, "src", filepath.FromSlash(stockPath(patch.file))))
		if err != nil {
			t.Fatal(err)
		}
		offset := bytes.Index(source, []byte(patch.from))
		if offset < 0 {
			t.Fatalf("overlay anchor for %s not in stock source: %q", patch.file, patch.from)
		}
		function, err := enclosingFunction(source, offset)
		if err != nil {
			t.Fatal(err)
		}
		got[strings.Join([]string{wasip1, stockPath(patch.file), function, symbol, "patch"}, " ")] += patch.count
	}
	return got
}

// effectiveSources builds the WASM overlay and returns a build context that
// reads the overlay replacement wherever one exists, plus the replaced paths
// that are new files.
func effectiveSources(t *testing.T) (string, build.Context, map[string]string) {
	t.Helper()
	stock, err := filepath.EvalSymlinks(stockRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	overlayPath, _, err := toolchain.BuildOverlay(stock, "../../../model/runtimesource", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(overlayPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct{ Replace map[string]string }
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	context := build.Default
	context.GOROOT, context.GOOS, context.GOARCH, context.CgoEnabled = stock, "wasip1", "wasm", false
	// The WASM guest builds with GOEXPERIMENT=nogreenteagc.
	context.ToolTags = slices.DeleteFunc(slices.Clone(build.Default.ToolTags), func(tag string) bool { return tag == "goexperiment.greenteagc" })
	context.OpenFile = func(path string) (io.ReadCloser, error) {
		if replacement, ok := manifest.Replace[path]; ok {
			path = replacement
		}
		return os.Open(path)
	}
	return stock, context, manifest.Replace
}

// wasip1HostClockReferences counts host-clock identifiers in every Go and
// assembly file that builds for wasip1/wasm after the overlay, ignoring
// comments but not compiler directives, as the native inventory does.
func wasip1HostClockReferences(t *testing.T, stock string, context build.Context, replace map[string]string) map[string]int {
	t.Helper()
	sourceRoot := filepath.Join(stock, "src")
	var paths []string
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
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for path := range replace {
		if !slices.Contains(paths, path) {
			paths = append(paths, path)
		}
	}
	got := map[string]int{}
	for _, path := range paths {
		name := filepath.Base(path)
		if strings.HasSuffix(name, "_test.go") || (!strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, ".s")) {
			continue
		}
		match, err := context.MatchFile(filepath.Dir(path), name)
		if err != nil {
			t.Fatal(err)
		}
		if !match {
			continue
		}
		reader, err := context.OpenFile(path)
		if err != nil {
			t.Fatal(err)
		}
		contents, err := io.ReadAll(reader)
		if closeErr := reader.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			t.Fatal(err)
		}
		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			t.Fatal(err)
		}
		relative = filepath.ToSlash(relative)
		patterns := []*regexp.Regexp{wasip1ClockIdentifier}
		if relative == "runtime/gomad_wasm.go" {
			patterns = append(patterns, wasip1GlueClockIdentifier)
		}
		for line := range strings.Lines(string(contents)) {
			if comment := strings.Index(line, "//"); comment >= 0 && !strings.HasPrefix(strings.TrimSpace(line), "//go:") {
				line = line[:comment]
			}
			for _, pattern := range patterns {
				for _, identifier := range pattern.FindAllString(line, -1) {
					got[strings.Join([]string{wasip1, relative, identifier}, " ")]++
				}
			}
		}
	}
	return got
}

func TestWasip1RuntimeDrawAndClockReferencesAreReviewed(t *testing.T) {
	stock, context, replace := effectiveSources(t)
	drawRows, err := parseDrawRows(reviewedWasip1DrawPatches)
	if err != nil {
		t.Fatal(err)
	}
	clockRows, err := parseClockRows(reviewedWasip1HostClockReferences)
	if err != nil {
		t.Fatal(err)
	}
	reviewedDraws := map[string]int{}
	for key, row := range drawRows {
		reviewedDraws[key] = row.count
	}
	reviewedClocks := map[string]int{}
	for key, row := range clockRows {
		reviewedClocks[key] = row.count
	}
	patches := overlayPatches(t)
	problems := referenceProblems("overlay draw/clock patch", drawPatchReferences(t, stock, patches), reviewedDraws)
	problems = append(problems, referenceProblems("wasip1 host-clock reference", wasip1HostClockReferences(t, stock, context, replace), reviewedClocks)...)
	if len(problems) != 0 {
		t.Fatalf("wasip1 runtime inventory drifted; review each change:\n%s", strings.Join(problems, "\n"))
	}
	// A new patch without a row fails by name.
	added := append(slices.Clone(patches), overlayPatch{"proc.go", "func runqget(pp *p) (gp *g, inheritTime bool) {", "func runqget(pp *p) (gp *g, inheritTime bool) {\n\t_ = cheaprand()", 1})
	if problems := referenceProblems("overlay draw/clock patch", drawPatchReferences(t, stock, added), reviewedDraws); !slices.ContainsFunc(problems, func(problem string) bool {
		return strings.Contains(problem, "unreviewed overlay draw/clock patch: wasip1/wasm runtime/proc.go runqget cheaprand patch")
	}) {
		t.Fatalf("new draw patch without a row not reported: %v", problems)
	}
	// Every escape row is an inventory escape row.
	inventory := loadInventory(t)
	for key, row := range clockRows {
		if row.disposition != "escape" {
			continue
		}
		if entry, ok := inventory.Sites("runtime-clock")[key]; !ok || entry.Disposition != "escape" {
			t.Errorf("wasip1 host-clock escape %s has no inventory escape row", key)
		}
	}
}
