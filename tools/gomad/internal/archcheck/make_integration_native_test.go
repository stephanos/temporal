//go:build gomad_native

package archcheck_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelectedNativeGatesRetainConformanceAndToolchainOrdering(t *testing.T) {
	for _, target := range []string{"test-host", "test-harness", "test"} {
		command := exec.CommandContext(t.Context(), "make", "--no-print-directory", "-n", target, "GOFLAGS=-tags=gomad_native")
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("expand selected %s: %v: %s", target, err, output)
		}
		expansion := string(output)
		if target == "test-host" && !strings.Contains(expansion, "go.temporal.io/server/tools/gomad/internal/backends/native/conformance") {
			t.Errorf("selected host gate omits conformance: %s", expansion)
		}
		if target != "test-host" {
			toolchain := strings.Index(expansion, " toolchain-build ")
			harness := strings.Index(expansion, "./cmd/gomad/... ./internal/host/...")
			if toolchain < 0 || harness < 0 || toolchain > harness {
				t.Errorf("selected %s must build toolchain before harness: %s", target, expansion)
			}
		}
	}
	directory := t.TempDir()
	shim := filepath.Join(directory, "go")
	script := "#!/bin/sh\ncase \" $* \" in\n  *' toolchain-build '*) touch \"$TOOLCHAIN_READY\";;\n  *' test '*) test -f \"$TOOLCHAIN_READY\" || exit 43; printf 'selected harness after toolchain\\n';;\nesac\n"
	if err := os.WriteFile(shim, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(t.Context(), "make", "--no-print-directory", "test-harness", "GOFLAGS=-tags=gomad_native")
	command.Env = append(os.Environ(), "PATH="+directory+string(os.PathListSeparator)+os.Getenv("PATH"), "TOOLCHAIN_READY="+filepath.Join(directory, "ready"))
	output, err := command.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "selected harness after toolchain") {
		t.Fatalf("clean selected harness prerequisite ordering: %v: %s", err, output)
	}
}

// clock-audit runs the native conformance script.
func TestClockAuditSelectsFilesystemScript(t *testing.T) {
	directory := t.TempDir()
	shim := filepath.Join(directory, "bash")
	if err := os.WriteFile(shim, []byte("#!/bin/sh\ntest -f \"$1\" || exit 42\nprintf 'clock script selected\\n'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(t.Context(), "make", "--no-print-directory", "-o", "toolchain", "clock-audit")
	command.Env = append(os.Environ(), "PATH="+directory+string(os.PathListSeparator)+os.Getenv("PATH"))
	output, err := command.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "clock script selected") {
		t.Fatalf("clock-audit script dispatch: %v: %s", err, output)
	}
}
