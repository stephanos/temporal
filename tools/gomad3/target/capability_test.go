package target

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCapabilityClosureRejectsEscapeCapabilities(t *testing.T) {
	tests := map[string]listedPackage{
		"syscall import":   {ImportPath: "example.com/target", Name: "main", Imports: []string{"syscall"}, Module: &listedModule{Path: "example.com/target", Main: true}},
		"x/sys import":     {ImportPath: "example.com/target", Name: "main", Imports: []string{"golang.org/x/sys/unix"}, Module: &listedModule{Path: "example.com/target", Main: true}},
		"process import":   {ImportPath: "example.com/target", Name: "main", Imports: []string{"os/exec"}, Module: &listedModule{Path: "example.com/target", Main: true}},
		"signal import":    {ImportPath: "example.com/target", Name: "main", Imports: []string{"os/signal"}, Module: &listedModule{Path: "example.com/target", Main: true}},
		"host user import": {ImportPath: "example.com/target", Name: "main", Imports: []string{"os/user"}, Module: &listedModule{Path: "example.com/target", Main: true}},
		"plugin import":    {ImportPath: "example.com/target", Name: "main", Imports: []string{"plugin"}, Module: &listedModule{Path: "example.com/target", Main: true}},
		"cgo source":       {ImportPath: "example.com/target", Name: "main", CgoFiles: []string{"target.go"}, Module: &listedModule{Path: "example.com/target", Main: true}},
		"assembly source":  {ImportPath: "example.com/target", Name: "main", SFiles: []string{"target.s"}, Module: &listedModule{Path: "example.com/target", Main: true}},
		"external object":  {ImportPath: "example.com/target", Name: "main", SysoFiles: []string{"target.syso"}, Module: &listedModule{Path: "example.com/target", Main: true}},
	}
	for name, pkg := range tests {
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			pkg.Dir = directory
			for _, source := range append(append(append([]string{}, pkg.CgoFiles...), pkg.SFiles...), pkg.SysoFiles...) {
				if err := os.WriteFile(filepath.Join(directory, source), []byte("foreign source\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := validateCapabilityClosure([]listedPackage{pkg}); err == nil || !strings.Contains(err.Error(), "unsupported target capability") {
				t.Fatalf("validateCapabilityClosure() error = %v", err)
			}
		})
	}
}

func TestBuiltInSimulationLinknamesRequireExactFirstPartySource(t *testing.T) {
	want := builtInSimulationLinknames["runtime_domain.go"]
	pkg := CapabilityPackage{ImportPath: "go.temporal.io/server/tools/gomad3sim", Module: &CapabilityModule{Path: "go.temporal.io/server", Main: true}}
	if !builtInSimulationLinknameAllowed(pkg, want) {
		t.Fatal("exact built-in simulation linkname source was rejected")
	}
	changed := want
	changed.SHA256 = "sha256:" + strings.Repeat("0", 64)
	if builtInSimulationLinknameAllowed(pkg, changed) {
		t.Fatal("changed built-in simulation linkname source was accepted")
	}
	pkg.Module.Path = "example.com/lookalike"
	if builtInSimulationLinknameAllowed(pkg, want) {
		t.Fatal("lookalike simulation package was accepted")
	}
}

func TestValidateCapabilityClosureRejectsLinkname(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "target.go"), []byte("package target\n\n//go:linkname escape syscall.Syscall\nfunc escape()\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := validateCapabilityClosure([]listedPackage{{
		ImportPath: "example.com/target", Name: "main", Dir: directory, GoFiles: []string{"target.go"},
		Module: &listedModule{Path: "example.com/target", Main: true},
	}})
	if err == nil || !strings.Contains(err.Error(), "go:linkname") {
		t.Fatalf("validateCapabilityClosure() error = %v", err)
	}
	var unsupported *UnsupportedCapabilityError
	if !errors.As(err, &unsupported) || unsupported.ImportPath != "example.com/target" {
		t.Fatalf("validateCapabilityClosure() error type = %T, value = %v", err, err)
	}
}

func TestValidateCapabilityClosureRejectsUnapprovedReflect2ForeignSources(t *testing.T) {
	reflect2Package := pinnedReflect2ListedPackage(t)
	reflect2Package.SFiles = append(reflect2Package.SFiles, "unexpected_arm64.s")
	packages := []listedPackage{
		{ImportPath: "example.com/main", Name: "main", Standard: true},
		reflect2Package,
	}
	if err := validateCapabilityClosure(packages); err == nil || !strings.Contains(err.Error(), "unexpected_arm64.s") {
		t.Fatalf("validateCapabilityClosure() error = %v", err)
	}
}

func TestValidateCapabilityClosureRejectsModifiedReflect2Linkname(t *testing.T) {
	directory := t.TempDir()
	contents := []byte("package reflect2\n\nimport _ \"unsafe\"\n\n//go:linkname mapiterinit syscall.Syscall\nfunc mapiterinit()\n")
	if err := os.WriteFile(filepath.Join(directory, "go_above_118.go"), contents, 0o600); err != nil {
		t.Fatal(err)
	}
	err := validateCapabilityClosure([]listedPackage{
		{ImportPath: "example.com/main", Name: "main", Standard: true},
		{
			ImportPath: "github.com/modern-go/reflect2", Name: "reflect2", Dir: directory, GoFiles: []string{"go_above_118.go"},
			Module: &listedModule{Path: "github.com/modern-go/reflect2", Version: "v1.0.3-0.20250322232337-35a7c28c31ee", Sum: "h1:W5t00kpgFdJifH4BDsTlE89Zl93FEloxaWZfGcifgq8="},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "go:linkname") {
		t.Fatalf("validateCapabilityClosure() error = %v", err)
	}
}

func TestValidateCapabilityClosureRejectsLocallyReplacedReflect2Linkname(t *testing.T) {
	directory := t.TempDir()
	contents := []byte("package reflect2\n\nimport _ \"unsafe\"\n\n//go:linkname mapiterinit reflect.mapiterinit\nfunc mapiterinit()\n")
	if err := os.WriteFile(filepath.Join(directory, "go_above_118.go"), contents, 0o600); err != nil {
		t.Fatal(err)
	}
	err := validateCapabilityClosure([]listedPackage{
		{ImportPath: "example.com/main", Name: "main", Standard: true},
		{
			ImportPath: "github.com/modern-go/reflect2", Name: "reflect2", Dir: directory, GoFiles: []string{"go_above_118.go"},
			Module: &listedModule{
				Path: "github.com/modern-go/reflect2", Version: "v1.0.3-0.20250322232337-35a7c28c31ee", Sum: "h1:W5t00kpgFdJifH4BDsTlE89Zl93FEloxaWZfGcifgq8=",
				Replace: &listedModule{Dir: directory},
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "go:linkname") {
		t.Fatalf("validateCapabilityClosure() error = %v", err)
	}
}

func TestValidateCapabilityClosureIgnoresUnlinkedDependencyTests(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "dependency.go"), []byte("package dependency\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "dependency_test.go"), []byte("package dependency\n\nimport \"os/exec\"\n\nvar _ = exec.Command\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := validateCapabilityClosure([]listedPackage{
		{ImportPath: "example.com/main", Name: "main", Standard: true},
		{
			ImportPath: "example.com/dependency", Name: "dependency", Dir: directory,
			GoFiles: []string{"dependency.go"}, TestGoFiles: []string{"dependency_test.go"}, TestImports: []string{"os/exec"},
			Module: &listedModule{Path: "example.com/dependency", Version: "v1.0.0", Sum: "h1:dependency"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateCapabilityClosureRejectsIncompletePinnedAdapterEvidence(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "adapter.go"), []byte("package adapter\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{"asm_bsd_arm64.s", "zsyscall_darwin_arm64.s"} {
		if err := os.WriteFile(filepath.Join(directory, source), []byte("foreign source\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	packages := []listedPackage{
		{ImportPath: "example.com/main", Name: "main", Standard: true},
		{ImportPath: "os", Name: "os", Standard: true, Imports: []string{"syscall"}, SFiles: []string{"raw.s"}},
		{ImportPath: "modernc.org/libc", Name: "libc", Dir: directory, GoFiles: []string{"adapter.go"}, Module: &listedModule{
			Path: "modernc.org/libc", Version: "v1.72.3", Replace: &listedModule{Dir: "/private/gomad/libc"},
		}},
		{ImportPath: "golang.org/x/sys/unix", Name: "unix", Dir: directory, GoFiles: []string{"adapter.go"}, Imports: []string{"syscall"}, SFiles: []string{"asm_bsd_arm64.s", "zsyscall_darwin_arm64.s"}, Module: &listedModule{Path: "golang.org/x/sys", Version: "v0.47.0", Sum: "h1:o7XGOvZQCADBQQ4Y7VNq2dRWQR7JmOUW8Kxx4ZsNgWs="}},
		{ImportPath: "github.com/mattn/go-isatty", Name: "isatty", Dir: directory, GoFiles: []string{"adapter.go"}, Imports: []string{"golang.org/x/sys/unix"}, Module: &listedModule{Path: "github.com/mattn/go-isatty", Version: "v0.0.21", Sum: "h1:xYae+lCNBP7QuW4PUnNG61ffM4hVIfm+zUzDuSzYLGs="}},
		{ImportPath: "modernc.org/memory", Name: "memory", Dir: directory, GoFiles: []string{"adapter.go"}, Imports: []string{"golang.org/x/sys/unix"}, Module: &listedModule{Path: "modernc.org/memory", Version: "v1.11.0", Sum: "h1:o4QC8aMQzmcwCK3t3Ux/ZHmwFPzE6hf2Y5LbkRs+hbI="}},
		{ImportPath: "modernc.org/sqlite", Name: "sqlite", Dir: directory, GoFiles: []string{"adapter.go"}, Imports: []string{"golang.org/x/sys/unix"}, Module: &listedModule{Path: "modernc.org/sqlite", Version: "v1.51.0", Sum: "h1:aH/MMSoayAIhozZ7uJbVTT9QO/VhzBf0J9tymmmuC/U="}},
	}
	if err := validateCapabilityClosure(packages); err == nil {
		t.Fatal("validateCapabilityClosure() accepted incomplete adapter and source evidence")
	}
}

func TestProjectCapabilityClosureRecordsSelectedCompatibilityPack(t *testing.T) {
	closure, err := projectCapabilityClosure([]listedPackage{
		{ImportPath: "example.com/main", Name: "main", Standard: true},
		pinnedReflect2ListedPackage(t),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(closure.Compatibility) != 1 || closure.Compatibility[0].ID != "reflect2-go126" || closure.Compatibility[0].SHA256 == "" {
		t.Fatalf("compatibility = %#v", closure.Compatibility)
	}
}

func pinnedReflect2ListedPackage(t *testing.T) listedPackage {
	t.Helper()
	moduleCache, err := ReadModuleCache(context.Background(), toolchainRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	return listedPackage{
		ImportPath: "github.com/modern-go/reflect2", Name: "reflect2",
		Dir: filepath.Join(moduleCache, "github.com/modern-go/reflect2@v1.0.3-0.20250322232337-35a7c28c31ee"),
		GoFiles: []string{
			"go_above_118.go", "go_above_19.go", "reflect2.go", "reflect2_kind.go", "safe_field.go", "safe_map.go", "safe_slice.go", "safe_struct.go", "safe_type.go", "type_map.go",
			"unsafe_array.go", "unsafe_eface.go", "unsafe_field.go", "unsafe_iface.go", "unsafe_link.go", "unsafe_map.go", "unsafe_ptr.go", "unsafe_slice.go", "unsafe_struct.go", "unsafe_type.go",
		},
		SFiles: []string{"relfect2_arm64.s", "relfect2_mips64x.s", "relfect2_mipsx.s", "relfect2_ppc64x.s"},
		Module: &listedModule{
			Path: "github.com/modern-go/reflect2", Version: "v1.0.3-0.20250322232337-35a7c28c31ee", Sum: "h1:W5t00kpgFdJifH4BDsTlE89Zl93FEloxaWZfGcifgq8=",
		},
	}
}

func TestValidateCapabilityClosureDoesNotExtendAdapterAllowlist(t *testing.T) {
	packages := []listedPackage{
		{ImportPath: "modernc.org/libc", Module: &listedModule{Path: "modernc.org/libc", Version: "v1.72.3", Replace: &listedModule{Dir: "/private/gomad/libc"}}},
		{ImportPath: "example.com/dependency", Imports: []string{"golang.org/x/sys/unix"}, Module: &listedModule{Path: "example.com/dependency", Version: "v1.0.0"}},
	}
	if err := validateCapabilityClosure(packages); err == nil {
		t.Fatal("validateCapabilityClosure() succeeded")
	}
}

func TestValidateCapabilityClosureDoesNotAuthorizeAnotherPackageInPinnedModule(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "dependency.go"), []byte("package dependency\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	packages := []listedPackage{
		{ImportPath: "example.com/main", Name: "main", Standard: true},
		{ImportPath: "modernc.org/libc", Name: "libc", Dir: directory, GoFiles: []string{"dependency.go"}, Module: &listedModule{
			Path: "modernc.org/libc", Version: "v1.72.3", Replace: &listedModule{Dir: "/private/gomad/libc"},
		}},
		{ImportPath: "golang.org/x/sys/unix", Name: "unix", Dir: directory, GoFiles: []string{"dependency.go"}, Module: &listedModule{
			Path: "golang.org/x/sys", Version: "v0.47.0", Sum: "h1:o7XGOvZQCADBQQ4Y7VNq2dRWQR7JmOUW8Kxx4ZsNgWs=",
		}},
		{ImportPath: "golang.org/x/sys/unreviewed", Name: "unreviewed", Dir: directory, GoFiles: []string{"dependency.go"}, Imports: []string{"syscall"}, Module: &listedModule{
			Path: "golang.org/x/sys", Version: "v0.47.0", Sum: "h1:o7XGOvZQCADBQQ4Y7VNq2dRWQR7JmOUW8Kxx4ZsNgWs=",
		}},
	}
	if err := validateCapabilityClosure(packages); err == nil || !strings.Contains(err.Error(), "unreviewed imports syscall") {
		t.Fatalf("validateCapabilityClosure() error = %v", err)
	}
}

func TestValidateCapabilityReviewRejectsForgedCompatibilityIdentity(t *testing.T) {
	closure := validCapabilityClosure()
	closure.Compatibility = []CompatibilityIdentity{{ID: "unknown-pack", SHA256: "sha256:" + strings.Repeat("0", 64)}}
	if err := validateCapabilityReview(closure); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("validateCapabilityReview() error = %v", err)
	}
}

func TestValidateCapabilityReviewRejectsForgedGeneratedMain(t *testing.T) {
	closure := validCapabilityClosure()
	closure.Packages[0].GeneratedTestMain = true
	closure.Packages[0].Imports = []string{"os/exec"}
	if err := validateCapabilityReview(closure); err == nil {
		t.Fatal("validateCapabilityReview() succeeded")
	}
}
