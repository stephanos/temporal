package deterministicio

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomadv3/target"
)

func TestToolchainLeavesFD5ForProcessesWithoutIOProfile(t *testing.T) {
	toolchainRoot, err := filepath.Abs(filepath.Join("..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := target.Prepare(context.Background(), target.Spec{
		Kind: target.KindGoRun, Source: "./io_fd5", WorkingDir: filepath.Join("..", "toolchain", "internal", "conformance", "testdata"),
		PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := read.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	first, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := first.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	second, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := second.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	command := exec.Command(prepared.Path)
	command.ExtraFiles = []*os.File{first, second, write}
	if output, runErr := command.CombinedOutput(); runErr != nil {
		t.Fatalf("run fixture: %v: %s", runErr, output)
	}
	if err := write.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(read)
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != "preserved" {
		t.Fatalf("fd 5 output = %q, want preserved", output)
	}
}
