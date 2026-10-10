package toolchain

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomad3/hostfs"
)

//go:embed runtime_wasm.go.txt
var runtimeGlue []byte

//go:embed overlay.go
var overlayImplementation []byte

func ImplementationSHA256() string {
	input := []byte("gomad-wasm-runtime-implementation/v1\x00")
	for _, source := range [][]byte{overlayImplementation, runtimeGlue} {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(source)))
		input = append(input, size[:]...)
		input = append(input, source...)
	}
	digest := sha256.Sum256(input)
	return "sha256:" + hex.EncodeToString(digest[:])
}

type rewrite struct {
	from, to string
	count    int
}

var nativeFunctions = []string{
	"gomadDiagnosticAllocations", "gomadChoiceRecordSelectResult", "gomadChoiceSelectOrigin",
	"gomadChoiceSelectReadiness", "gomadChoiceWaiterPresent", "gomadChoiceSeedRandom", "gomadTimerRand",
	"gomadRuntimeRand", "gomadRuntimeCheapRand", "gomadSeededDrawCheck", "gomadChoiceRunqSeeded",
	"gomadChoiceRunnextSeeded", "gomadChoiceRandom", "gomadChoiceSelectSeeded", "gomadChoiceEncodeRecord",
	"gomadChoiceZero", "gomadChoiceEqual", "gomadChoiceSite", "gomadChoiceRootIdentity",
	"gomadChoiceAssignGoroutineIdentity", "gomadChoiceTimerCreated", "gomadChoiceTimerFire",
	"gomadChoiceTimerFired", "gomadChoiceRunqIndex", "gomadChoiceSelectPollIndex", "gomadChoiceSelectIdentity",
}

// BuildOverlay derives a WASI runtime from the pinned stock sources and the
// native owners' pure choice semantics without changing either input tree.
func BuildOverlay(stockRoot, nativeRuntimeRoot, outputRoot string) (overlayPath, identity string, err error) {
	stockRoot, err = filepath.EvalSymlinks(stockRoot)
	if err != nil {
		return "", "", fmt.Errorf("stock Go root: %w", err)
	}
	stockRoot, err = filepath.Abs(stockRoot)
	if err != nil {
		return "", "", err
	}
	nativeRuntimeRoot, err = filepath.EvalSymlinks(nativeRuntimeRoot)
	if err != nil {
		return "", "", fmt.Errorf("native runtime root: %w", err)
	}
	nativeRuntimeRoot, err = filepath.Abs(nativeRuntimeRoot)
	if err != nil {
		return "", "", err
	}
	version, err := os.ReadFile(filepath.Join(stockRoot, "VERSION"))
	if err != nil {
		return "", "", fmt.Errorf("stock Go version: %w", err)
	}
	if strings.SplitN(string(version), "\n", 2)[0] != "go1.27.1" {
		return "", "", fmt.Errorf("WASM runtime requires stock Go 1.27.1")
	}
	native, err := os.ReadFile(filepath.Join(nativeRuntimeRoot, "gomad.go"))
	if err != nil {
		return "", "", fmt.Errorf("native choice semantics: %w", err)
	}
	extracted, err := extractNative(native)
	if err != nil {
		return "", "", err
	}
	wire, err := os.ReadFile(filepath.Join(nativeRuntimeRoot, "gomad_choicewire_generated.go"))
	if err != nil {
		return "", "", fmt.Errorf("native choice codec: %w", err)
	}
	inputs := map[string][]byte{"VERSION": version, "native-semantic-declarations": extracted, "native-choicewire": wire, "wasm-glue": runtimeGlue}
	outputs := map[string][]byte{}
	for name, rules := range stockRewrites() {
		original, err := os.ReadFile(filepath.Join(stockRoot, "src/runtime", name))
		if err != nil {
			return "", "", fmt.Errorf("stock runtime %s: %w", name, err)
		}
		inputs["stock/"+name] = original
		patched := string(original)
		for _, rule := range rules {
			if count := strings.Count(patched, rule.from); count != rule.count {
				return "", "", fmt.Errorf("stock runtime %s anchor drift: expected %d occurrence(s), got %d: %q", name, rule.count, count, rule.from)
			}
			patched = strings.ReplaceAll(patched, rule.from, rule.to)
		}
		formatted, err := format.Source([]byte(patched))
		if err != nil {
			return "", "", fmt.Errorf("format patched runtime %s: %w", name, err)
		}
		outputs[name] = formatted
	}
	outputs["gomad_wasm.go"], err = format.Source(bytes.Join([][]byte{runtimeGlue, extracted}, []byte("\n")))
	if err != nil {
		return "", "", fmt.Errorf("format WASM runtime: %w", err)
	}
	outputs["gomad_choicewire_generated.go"] = wire
	for _, name := range []string{"gomad_wasm.go", "gomad_choicewire_generated.go"} {
		if _, err := os.Lstat(filepath.Join(stockRoot, "src/runtime", name)); err == nil || !os.IsNotExist(err) {
			return "", "", fmt.Errorf("WASM runtime overlay collides with stock file %s", name)
		}
	}
	for name, data := range outputs {
		inputs["output/"+name] = data
	}
	names := make([]string, 0, len(inputs))
	for name := range inputs {
		names = append(names, name)
	}
	slices.Sort(names)
	identityBytes := []byte("gomad-wasm-runtime-overlay/v1\x00")
	var length [8]byte
	for _, name := range names {
		for _, part := range [][]byte{[]byte(name), inputs[name]} {
			binary.BigEndian.PutUint64(length[:], uint64(len(part)))
			identityBytes = append(identityBytes, length[:]...)
			identityBytes = append(identityBytes, part...)
		}
	}
	digest := sha256.Sum256(identityBytes)
	identity = hex.EncodeToString(digest[:])
	outputRoot, err = filepath.Abs(outputRoot)
	if err != nil {
		return "", "", err
	}
	ancestor := outputRoot
	var suffix []string
	for {
		resolved, resolveErr := filepath.EvalSymlinks(ancestor)
		if resolveErr == nil {
			outputRoot = resolved
			for index := len(suffix) - 1; index >= 0; index-- {
				outputRoot = filepath.Join(outputRoot, suffix[index])
			}
			break
		}
		if !os.IsNotExist(resolveErr) || filepath.Dir(ancestor) == ancestor {
			return "", "", fmt.Errorf("resolve overlay output: %w", resolveErr)
		}
		suffix = append(suffix, filepath.Base(ancestor))
		ancestor = filepath.Dir(ancestor)
	}
	for _, source := range []string{stockRoot, nativeRuntimeRoot} {
		if relative, err := filepath.Rel(source, outputRoot); err != nil || relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return "", "", fmt.Errorf("overlay output must be outside input source trees")
		}
	}
	if err := os.MkdirAll(outputRoot, 0700); err != nil {
		return "", "", fmt.Errorf("overlay output: %w", err)
	}
	directory := filepath.Join(outputRoot, identity)
	manifest := struct{ Replace map[string]string }{Replace: make(map[string]string, len(outputs))}
	for name := range outputs {
		manifest.Replace[filepath.Join(stockRoot, "src/runtime", name)] = filepath.Join(directory, name)
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return "", "", err
	}
	outputs["overlay.json"] = data
	overlayPath = filepath.Join(directory, "overlay.json")
	if _, err := os.Lstat(directory); err == nil {
		if err := verifyPublished(directory, outputs); err != nil {
			return "", "", err
		}
		return overlayPath, identity, nil
	} else if !os.IsNotExist(err) {
		return "", "", err
	}
	staging, err := os.MkdirTemp(outputRoot, ".runtime-overlay-")
	if err != nil {
		return "", "", err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(staging)) }()
	for name, data := range outputs {
		if err := os.WriteFile(filepath.Join(staging, name), data, 0600); err != nil {
			return "", "", fmt.Errorf("write runtime overlay %s: %w", name, err)
		}
	}
	if publishErr := os.Rename(staging, directory); publishErr != nil {
		if err := verifyPublished(directory, outputs); err != nil {
			return "", "", errors.Join(fmt.Errorf("publish runtime overlay: %w", publishErr), err)
		}
	}
	return overlayPath, identity, nil
}

func verifyPublished(directory string, outputs map[string][]byte) error {
	info, err := os.Lstat(directory)
	if err != nil {
		return fmt.Errorf("runtime overlay directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("runtime overlay directory is not a regular directory")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("runtime overlay inventory: %w", err)
	}
	if len(entries) != len(outputs) {
		return fmt.Errorf("runtime overlay inventory changed")
	}
	for name, expected := range outputs {
		data, err := hostfs.ReadBounded(filepath.Join(directory, name), uint64(len(expected)))
		if err != nil {
			return fmt.Errorf("runtime overlay %s: %w", name, err)
		}
		if !bytes.Equal(data, expected) {
			return fmt.Errorf("runtime overlay %s bytes changed", name)
		}
	}
	return nil
}

func extractNative(source []byte) ([]byte, error) {
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, "gomad.go", source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse native runtime: %w", err)
	}
	wanted := make(map[string]bool, len(nativeFunctions)+2)
	for _, name := range nativeFunctions {
		wanted[name] = true
	}
	wanted["gomadChoiceTimerIdentity"] = true
	wanted["gomadChoiceRecordValue"] = true
	seen := make(map[string]bool, len(wanted))
	var extracted bytes.Buffer
	for _, declaration := range file.Decls {
		name := ""
		switch decl := declaration.(type) {
		case *ast.FuncDecl:
			name = decl.Name.Name
		case *ast.GenDecl:
			if len(decl.Specs) == 1 {
				if spec, ok := decl.Specs[0].(*ast.TypeSpec); ok {
					name = spec.Name.Name
				}
			}
		}
		if seen[name] {
			return nil, fmt.Errorf("duplicate native declaration %s", name)
		}
		if !wanted[name] {
			continue
		}
		start := declaration.Pos()
		if decl, ok := declaration.(*ast.FuncDecl); ok && decl.Doc != nil {
			start = decl.Doc.Pos()
		}
		extracted.Write(source[set.Position(start).Offset:set.Position(declaration.End()).Offset])
		extracted.WriteString("\n\n")
		seen[name] = true
		delete(wanted, name)
	}
	if len(wanted) != 0 {
		names := make([]string, 0, len(wanted))
		for name := range wanted {
			names = append(names, name)
		}
		slices.Sort(names)
		return nil, fmt.Errorf("missing native semantic declarations: %s", strings.Join(names, ", "))
	}
	return extracted.Bytes(), nil
}

func stockRewrites() map[string][]rewrite {
	return map[string][]rewrite{
		"proc.go": {
			{"\trandinit() // must run before mallocinit, AlgInit, mcommoninit", "\tgomadWasmInit()\n\trandinit() // must run before mallocinit, AlgInit, mcommoninit", 1},
			{"\t// Run the initializing tasks. Depending on build mode this", "\tgomadWasmStartUserCode(mp)\n\n\t// Run the initializing tasks. Depending on build mode this", 1},
			{"\tif n, err := strconv.ParseInt(gogetenv(\"GOMAXPROCS\"), 10, 32); err == nil && n > 0 {", "\tif gomadEnabled {\n\t\tprocs = 1\n\t\tsched.customGOMAXPROCS = true\n\t} else if n, err := strconv.ParseInt(gogetenv(\"GOMAXPROCS\"), 10, 32); err == nil && n > 0 {", 1},
			{"\tnewg.gopc = callerpc", "\tnewg.gopc = callerpc\n\tgomadChoiceAssignGoroutineIdentity(newg, callergp, callerpc)", 1},
			{"\tif randomizeScheduler {\n", "\tif randomizeScheduler || gomadEnabled {\n", 2},
			{"\t\t\tj := cheaprandn(i + 1)", "\t\t\tj := gomadChoiceRunnextSeeded(i + 1)", 2},
			{"func runqget(pp *p) (gp *g, inheritTime bool) {", "func runqget(pp *p) (gp *g, inheritTime bool) {\n\tif gomadEnabled {\n\t\th, t := pp.runqhead, pp.runqtail\n\t\tif t-h > 1 {\n\t\t\toffset := gomadChoiceRunqIndex(pp, h, t)\n\t\t\tindex := (h + offset) % uint32(len(pp.runq))\n\t\t\tgp := pp.runq[index].ptr()\n\t\t\tpp.runq[index] = pp.runq[h%uint32(len(pp.runq))]\n\t\t\tatomic.StoreRel(&pp.runqhead, h+1)\n\t\t\treturn gp, false\n\t\t}\n\t}", 1},
		},
		"runtime2.go": {{"\tgoid         uint64\n", "\tgoid         uint64\n\tgomadID      [32]byte\n\tgomadChild   uint64\n\tgomadTimer   uint64\n", 1}},
		"rand.go": {
			{"\tif len(startupRand) >= 16 &&", "\tif gomadEnabled {\n\t\tbyteorder.BEPutUint64(seed[:], gomadSeed)\n\t} else if len(startupRand) >= 16 &&", 1},
			{"\tmp := getg().m\n\tc := &mp.chacha8", "\tmp := getg().m\n\tif gomadEnabled && mp.p != 0 {\n\t\treturn gomadRuntimeRand(mp)\n\t}\n\tc := &mp.chacha8", 1},
			{"func mrandinit(mp *m) {", "func mrandinit(mp *m) {\n\tif gomadEnabled {\n\t\tmp.chacha8.Init64([4]uint64{gomadSeed})\n\t\tmp.cheaprand = uint32(gomadSeed)\n\t\tmp.cheaprand64 = gomadSeed\n\t\treturn\n\t}", 1},
			{"func cheaprand() uint32 {\n\tmp := getg().m", "func cheaprand() uint32 {\n\tmp := getg().m\n\tif gomadEnabled && mp.p != 0 {\n\t\treturn gomadRuntimeCheapRand()\n\t}", 1},
		},
		"select.go": {
			{"\tgp := getg()\n\tif debugSelect", "\tgp := getg()\n\tchoiceSite, choiceSiteFlags := gomadChoiceSite(sys.GetCallerPC())\n\tchoiceOrigin := gomadChoiceSelectOrigin()\n\tif debugSelect", 1},
			{"\t\tj := cheaprandn(uint32(norder + 1))", "\t\tj := gomadChoiceSelectSeeded(uint32(norder + 1))\n\t\tj = gomadChoiceSelectPollIndex(pollorder, norder, i, nsends, choiceSite, choiceSiteFlags, j)", 1},
			{"\tvar recvOK bool\n", "\tvar recvOK bool\n\tvar selected, alternatives uint32\n\tchoiceReadiness := gomadChoiceSelectReadiness(scases, lockorder, nsends, block)\n", 1},
			{"\treturn casi, recvOK\n", "\tselected, alternatives = uint32(casi), uint32(ncases)\n\tif !block {\n\t\talternatives++\n\t\tif casi < 0 { selected = uint32(ncases) }\n\t}\n\tgomadChoiceRecordSelectResult(choiceSiteFlags, choiceSite, alternatives, selected, uint32(norder), choiceReadiness, choiceOrigin)\n\treturn casi, recvOK\n", 1},
		},
		"time.go": {
			{"\tblocked uint32 // number of goroutines blocked on timer's channel", "\tgomadCreator [32]byte\n\tgomadFirings uint64\n\tblocked uint32 // number of goroutines blocked on timer's channel", 1},
			{"\tt.timer.init(nil, nil)", "\tt.timer.init(nil, nil)\n\tgomadChoiceTimerCreated(&t.timer, sys.GetCallerPC())", 1},
			{"\t\tif t.isFake {\n\t\t\t// Re-randomize timer order.", "\t\tif t.isFake || gomadEnabled {\n\t\t\t// Re-randomize timer order.", 1},
			{"\t\t\tt.rand = cheaprand()", "\t\t\tt.rand = gomadTimerRand()", 1},
			{"\t\t// Update ts.heap[0].when and move within heap.\n", "\t\t// Update ts.heap[0].when and move within heap.\n\t\tif gomadEnabled && !t.isFake { t.rand = gomadTimerRand() }\n", 1},
			{"\t\tcase t.state&timerModified != 0:\n", "\t\tcase t.state&timerModified != 0:\n\t\t\tif gomadEnabled && !t.isFake { t.rand = gomadTimerRand() }\n", 1},
			{"\tf(arg, seq, delay)\n", "\tf(arg, seq, delay)\n\tgomadChoiceTimerFired(callback)\n", 1},
			{"\n\tt.unlock()\n\n\tif raceenabled {\n\t\t// Temporarily use the current P's racectx for g0.", "\n\tcallback := gomadChoiceTimerFire(t)\n\tt.unlock()\n\n\tif raceenabled {\n\t\t// Temporarily use the current P's racectx for g0.", 1},
		},
		"os_wasip1.go": {
			{"func exit(code int32)", "func gomadWasmExit(code int32)", 1},
			{"\treturn int64(time)\n", "\tgomadWasmLastTime = int64(time)\n\treturn int64(time)\n", 1},
		},
		"note_other.go": {{"\tkey uintptr\n", "\tkey uintptr\n\twaiter guintptr\n\tdeadline int64\n\tnext *note\n\tstatus uint8\n", 1}},
		"lock_wasip1.go": {
			{"\tn.key = 0\n", "\tif n.waiter != 0 { throw(\"noteclear with active waiter\") }\n\tn.key = 0\n", 1},
			{"\tn.key = 1\n", "\tn.key = 1\n\tif n.status == 1 && n.waiter != 0 {\n\t\tn.status = 2\n\t\tgoready(n.waiter.ptr(), 1)\n\t}\n", 1},
			{stockNoteSleep, "func notetsleepg(n *note, ns int64) bool {\n\treturn gomadWasmNoteSleep(n, ns)\n}", 1},
			{"func beforeIdle(int64, int64) (*g, bool) {\n\treturn nil, false\n}", "func beforeIdle(now, pollUntil int64) (*g, bool) {\n\treturn gomadWasmBeforeIdle(now, pollUntil)\n}", 1},
		},
	}
}

const stockNoteSleep = `func notetsleepg(n *note, ns int64) bool {
	gp := getg()
	if gp == gp.m.g0 {
		throw("notetsleepg on g0")
	}

	deadline := nanotime() + ns
	for {
		if n.key != 0 {
			return true
		}
		if sched_yield() != 0 {
			throw("sched_yield failed")
		}
		Gosched()
		if ns >= 0 && nanotime() >= deadline {
			return false
		}
	}
}`
