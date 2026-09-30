package toolchain

import (
	"errors"
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
	"strings"
	"testing"

	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

// The host-clock inventory replaces a dynamic Linux audit: on linux/amd64 the
// runtime reads the clock through the vDSO, which seccomp and ptrace cannot
// observe. The interception itself is platform-neutral Go, so what can differ
// per platform is who reaches the host clock without passing through it. Every
// reference is pinned here; a new upstream caller fails the toolchain tier on
// either host.
var hostClockIdentifier = regexp.MustCompile(`\b(nanotime1|walltime|time_now|vdsoClockgettimeSym|vdsoGettimeofdaySym)\b`)

type clockDisposition string

const (
	// clockImplementation declares or implements the host clock itself.
	clockImplementation clockDisposition = "implementation"
	// clockGuarded is reached only after the gomadEnabled check returned.
	clockGuarded clockDisposition = "guarded"
	// clockHostByDesign is an intentional host read that never feeds target state.
	clockHostByDesign clockDisposition = "host-by-design"
	// clockEscape reaches the host clock after activation; its finding names the
	// consequence and why it is recorded rather than fixed.
	clockEscape clockDisposition = "escape"
	// clockUnrelated shares a name with a host-clock identifier.
	clockUnrelated clockDisposition = "unrelated"
)

type clockReference struct {
	platform    string
	file        string
	identifier  string
	count       int
	disposition clockDisposition
	finding     string
}

func (r clockReference) key() string {
	return fmt.Sprintf("%s %s %s", r.platform, r.file, r.identifier)
}

var reviewedHostClockReferences = []clockReference{
	{"darwin/arm64", "internal/trace/internal/testgen/trace.go", "walltime", 4, clockUnrelated, ""},
	{"darwin/arm64", "runtime/gomad.go", "nanotime1", 1, clockHostByDesign, "gomadWallNanotime serves the deterministic I/O packages' wall bounds"},
	{"darwin/arm64", "runtime/mgc.go", "time_now", 1, clockEscape, "gcMarkTermination stores host wall time in MemStats.LastGC and debug.GCStats"},
	{"darwin/arm64", "runtime/sys_darwin.go", "nanotime1", 1, clockImplementation, ""},
	{"darwin/arm64", "runtime/sys_darwin.go", "walltime", 2, clockImplementation, ""},
	{"darwin/arm64", "runtime/time.go", "time_now", 2, clockEscape, "time_runtimeNow is guarded; crypto/internal/fips140deps/time.monoTime is not and seeds the FIPS CPU-jitter entropy source, reached only in FIPS mode"},
	{"darwin/arm64", "runtime/time_nofake.go", "nanotime1", 1, clockGuarded, ""},
	{"darwin/arm64", "runtime/timestub.go", "time_now", 2, clockImplementation, ""},
	{"darwin/arm64", "runtime/timestub.go", "walltime", 1, clockImplementation, ""},
	{"darwin/arm64", "runtime/tracetime.go", "time_now", 1, clockEscape, "the execution tracer's clock snapshot; tracing is outside the deterministic contract"},
	{"linux/amd64", "internal/trace/internal/testgen/trace.go", "walltime", 4, clockUnrelated, ""},
	{"linux/amd64", "runtime/badlinkname_linux.go", "vdsoClockgettimeSym", 1, clockImplementation, ""},
	{"linux/amd64", "runtime/gomad.go", "nanotime1", 1, clockHostByDesign, "gomadWallNanotime serves the deterministic I/O packages' wall bounds"},
	{"linux/amd64", "runtime/mgc.go", "time_now", 1, clockEscape, "gcMarkTermination stores host wall time in MemStats.LastGC and debug.GCStats"},
	{"linux/amd64", "runtime/stubs3.go", "nanotime1", 2, clockImplementation, ""},
	{"linux/amd64", "runtime/sys_linux_amd64.s", "nanotime1", 1, clockImplementation, ""},
	{"linux/amd64", "runtime/sys_linux_amd64.s", "vdsoClockgettimeSym", 1, clockImplementation, ""},
	{"linux/amd64", "runtime/time.go", "time_now", 2, clockEscape, "time_runtimeNow is guarded; crypto/internal/fips140deps/time.monoTime is not and seeds the FIPS CPU-jitter entropy source, reached only in FIPS mode"},
	{"linux/amd64", "runtime/time_linux_amd64.s", "vdsoClockgettimeSym", 2, clockImplementation, ""},
	{"linux/amd64", "runtime/time_nofake.go", "nanotime1", 1, clockGuarded, ""},
	{"linux/amd64", "runtime/timeasm.go", "time_now", 2, clockImplementation, ""},
	{"linux/amd64", "runtime/tracetime.go", "time_now", 1, clockEscape, "the execution tracer's clock snapshot; tracing is outside the deterministic contract"},
	{"linux/amd64", "runtime/vdso_linux_amd64.go", "vdsoClockgettimeSym", 2, clockImplementation, ""},
	{"linux/amd64", "runtime/vdso_linux_amd64.go", "vdsoGettimeofdaySym", 3, clockImplementation, ""},
	{"linux/amd64", "syscall/asm_linux_amd64.s", "vdsoGettimeofdaySym", 1, clockEscape, "syscall.Gettimeofday; the syscall package is admitted only by an exact compatibility pack"},
}

func TestPatchedRuntimeHostClockReferencesAreReviewed(t *testing.T) {
	goRoot := builtGOROOT(t)
	want := map[string]clockReference{}
	for _, reference := range reviewedHostClockReferences {
		if reference.disposition == clockEscape && reference.finding == "" {
			t.Fatalf("escape %s has no finding", reference.key())
		}
		want[reference.key()] = reference
	}
	for _, platform := range gomadversion.SupportedPlatforms {
		if !slices.ContainsFunc(reviewedHostClockReferences, func(r clockReference) bool { return r.platform == platform }) {
			t.Fatalf("supported platform %s has no reviewed host-clock inventory", platform)
		}
	}

	got, err := hostClockReferences(goRoot, gomadversion.SupportedPlatforms[:])
	if err != nil {
		t.Fatal(err)
	}
	var problems []string
	for key, count := range got {
		reference, reviewed := want[key]
		switch {
		case !reviewed:
			problems = append(problems, fmt.Sprintf("unreviewed host-clock reference: %s (%d)", key, count))
		case reference.count != count:
			problems = append(problems, fmt.Sprintf("host-clock reference count changed: %s = %d, reviewed %d", key, count, reference.count))
		default:
		}
	}
	for key := range want {
		if _, found := got[key]; !found {
			problems = append(problems, fmt.Sprintf("reviewed host-clock reference disappeared: %s", key))
		}
	}
	slices.Sort(problems)
	if len(problems) > 0 {
		t.Fatalf("patched runtime host-clock inventory drifted; review each change and update reviewedHostClockReferences:\n%s", strings.Join(problems, "\n"))
	}
}

func TestPatchedRuntimeClockEntryPointsCheckActivationFirst(t *testing.T) {
	goRoot := builtGOROOT(t)
	for _, entry := range []struct {
		file, function, hostCall string
	}{
		{"runtime/time_nofake.go", "nanotime", "nanotime1"},
		{"runtime/time.go", "time_runtimeNow", "time_now"},
	} {
		guard, host, err := guardAndHostCallPositions(filepath.Join(goRoot, "src", filepath.FromSlash(entry.file)), entry.function, entry.hostCall)
		if err != nil {
			t.Fatal(err)
		}
		if !guard.IsValid() || !host.IsValid() || guard.Offset > host.Offset {
			t.Fatalf("%s: %s must return on gomadEnabled before calling %s (guard=%v host=%v)", entry.file, entry.function, entry.hostCall, guard, host)
		}
	}
}

func builtGOROOT(t *testing.T) string {
	t.Helper()
	toolchainRoot, err := filepath.Abs(filepath.Join("..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	buildKey, err := os.ReadFile(filepath.Join(toolchainRoot, "build-key"))
	if errors.Is(err, fs.ErrNotExist) {
		t.Skip("the patched toolchain is not built; make test-toolchain builds it first")
	}
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(toolchainRoot, "builds", strings.TrimSpace(string(buildKey)))
}

// hostClockReferences counts host-clock identifiers in every standard-library
// Go and assembly file that builds for each platform, ignoring comments but not
// compiler directives, because a //go:linkname exposes the symbol.
func hostClockReferences(goRoot string, platforms []string) (map[string]int, error) {
	references := map[string]int{}
	sourceRoot := filepath.Join(goRoot, "src")
	for _, platform := range platforms {
		goos, goarch, ok := strings.Cut(platform, "/")
		if !ok {
			return nil, fmt.Errorf("malformed platform %q", platform)
		}
		context := build.Default
		context.GOROOT, context.GOOS, context.GOARCH, context.CgoEnabled = goRoot, goos, goarch, false
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
			if strings.HasSuffix(name, "_test.go") || (!strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, ".s")) {
				return nil
			}
			match, err := context.MatchFile(filepath.Dir(path), name)
			if err != nil || !match {
				return err
			}
			counts, err := countHostClockIdentifiers(path)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(sourceRoot, path)
			if err != nil {
				return err
			}
			for identifier, count := range counts {
				references[fmt.Sprintf("%s %s %s", platform, filepath.ToSlash(relative), identifier)] = count
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return references, nil
}

func countHostClockIdentifiers(path string) (map[string]int, error) {
	// Some generated standard-library sources carry multi-megabyte lines, so the
	// file is split in memory rather than scanned with a bounded line buffer.
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for line := range strings.Lines(string(contents)) {
		if comment := strings.Index(line, "//"); comment >= 0 && !strings.HasPrefix(strings.TrimSpace(line), "//go:") {
			line = line[:comment]
		}
		for _, identifier := range hostClockIdentifier.FindAllString(line, -1) {
			counts[identifier]++
		}
	}
	return counts, nil
}

// guardAndHostCallPositions returns the first `if gomadEnabled` statement and
// the first call to hostCall inside the named top-level function.
func guardAndHostCallPositions(path, function, hostCall string) (guard, host token.Position, err error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return guard, host, err
	}
	for _, declaration := range file.Decls {
		definition, ok := declaration.(*ast.FuncDecl)
		if !ok || definition.Recv != nil || definition.Name.Name != function || definition.Body == nil {
			continue
		}
		ast.Inspect(definition.Body, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.IfStmt:
				if identifier, ok := node.Cond.(*ast.Ident); ok && identifier.Name == "gomadEnabled" && !guard.IsValid() {
					if returnsImmediately(node.Body) {
						guard = fileSet.Position(node.Pos())
					}
				}
			case *ast.CallExpr:
				if identifier, ok := node.Fun.(*ast.Ident); ok && identifier.Name == hostCall && !host.IsValid() {
					host = fileSet.Position(node.Pos())
				}
			default:
			}
			return true
		})
		return guard, host, nil
	}
	return guard, host, fmt.Errorf("%s: function %s not found", path, function)
}

func returnsImmediately(body *ast.BlockStmt) bool {
	if len(body.List) == 0 {
		return false
	}
	_, ok := body.List[len(body.List)-1].(*ast.ReturnStmt)
	return ok
}
