package archcheck_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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
