package target

import (
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
			if err := validateCapabilityClosure([]listedPackage{pkg}); err == nil || !strings.Contains(err.Error(), "unsupported target capability") {
				t.Fatalf("validateCapabilityClosure() error = %v", err)
			}
		})
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

func TestValidateCapabilityClosureAllowsPinnedReflect2Linkname(t *testing.T) {
	directory := t.TempDir()
	contents := []byte("package reflect2\n\nimport _ \"unsafe\"\n\n//go:linkname mapiterinit reflect.mapiterinit\nfunc mapiterinit()\n")
	if err := os.WriteFile(filepath.Join(directory, "go_above_118.go"), contents, 0o600); err != nil {
		t.Fatal(err)
	}
	packages := []listedPackage{
		{ImportPath: "example.com/main", Name: "main", Standard: true},
		{
			ImportPath: "github.com/modern-go/reflect2", Name: "reflect2", Dir: directory, GoFiles: []string{"go_above_118.go"},
			Module: &listedModule{Path: "github.com/modern-go/reflect2", Version: "v1.0.3-0.20250322232337-35a7c28c31ee", Sum: "h1:W5t00kpgFdJifH4BDsTlE89Zl93FEloxaWZfGcifgq8="},
		},
	}
	if err := validateCapabilityClosure(packages); err != nil {
		t.Fatal(err)
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

func TestValidateCapabilityClosureAllowsStandardLibraryAndPinnedAdapter(t *testing.T) {
	packages := []listedPackage{
		{ImportPath: "example.com/main", Name: "main", Standard: true},
		{ImportPath: "os", Name: "os", Standard: true, Imports: []string{"syscall"}, SFiles: []string{"raw.s"}},
		{ImportPath: "modernc.org/libc", Name: "libc", Imports: []string{"golang.org/x/sys/unix"}, Module: &listedModule{
			Path: "modernc.org/libc", Version: "v1.72.3", Sum: "h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU=", Replace: &listedModule{Dir: "/private/gomad/libc"},
		}},
		{ImportPath: "golang.org/x/sys/unix", Name: "unix", Imports: []string{"syscall"}, SFiles: []string{"syscall.s"}, Module: &listedModule{Path: "golang.org/x/sys", Version: "v0.47.0", Sum: "h1:o7XGOvZQCADBQQ4Y7VNq2dRWQR7JmOUW8Kxx4ZsNgWs="}},
		{ImportPath: "github.com/mattn/go-isatty", Name: "isatty", Imports: []string{"golang.org/x/sys/unix"}, Module: &listedModule{Path: "github.com/mattn/go-isatty", Version: "v0.0.21", Sum: "h1:xYae+lCNBP7QuW4PUnNG61ffM4hVIfm+zUzDuSzYLGs="}},
		{ImportPath: "modernc.org/memory", Name: "memory", Imports: []string{"golang.org/x/sys/unix"}, Module: &listedModule{Path: "modernc.org/memory", Version: "v1.11.0", Sum: "h1:o4QC8aMQzmcwCK3t3Ux/ZHmwFPzE6hf2Y5LbkRs+hbI="}},
		{ImportPath: "modernc.org/sqlite", Name: "sqlite", Imports: []string{"golang.org/x/sys/unix"}, Module: &listedModule{Path: "modernc.org/sqlite", Version: "v1.51.0", Sum: "h1:aH/MMSoayAIhozZ7uJbVTT9QO/VhzBf0J9tymmmuC/U="}},
	}
	if err := validateCapabilityClosure(packages); err != nil {
		t.Fatal(err)
	}
}

func TestValidateCapabilityClosureDoesNotExtendAdapterAllowlist(t *testing.T) {
	packages := []listedPackage{
		{ImportPath: "modernc.org/libc", Module: &listedModule{Path: "modernc.org/libc", Version: "v1.72.3", Sum: "h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU=", Replace: &listedModule{Dir: "/private/gomad/libc"}}},
		{ImportPath: "example.com/dependency", Imports: []string{"golang.org/x/sys/unix"}, Module: &listedModule{Path: "example.com/dependency", Version: "v1.0.0"}},
	}
	if err := validateCapabilityClosure(packages); err == nil {
		t.Fatal("validateCapabilityClosure() succeeded")
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
