package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/compatibilitypack/authoring"
)

type refreshOutput struct {
	bytes.Buffer
	file      *os.File
	remaining int
	err       error
}

func (output *refreshOutput) Write(contents []byte) (int, error) {
	if output.remaining > 0 {
		output.remaining--
		return output.Buffer.Write(contents)
	}
	written, err := output.file.Write(contents)
	output.err = err
	return written, err
}

func newRefreshOutput(t *testing.T, successfulWrites int) *refreshOutput {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stdout")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	})
	return &refreshOutput{file: file, remaining: successfulWrites}
}

func TestRunCompatibilityPackRefreshOutputFailure(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	packRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(packRoot, "requests"), 0o700); err != nil {
		t.Fatal(err)
	}
	table := []byte("{\"requests\":[],\"schema\":\"gomad3.compatibility-pack-working-directories/v1\"}\n")
	if err := os.WriteFile(filepath.Join(packRoot, authoring.WorkingDirectoriesFile), table, 0o600); err != nil {
		t.Fatal(err)
	}
	arguments := []string{"compatibility-pack", "refresh", "--root=" + root, "--compatibility-root=" + packRoot, "--go=" + goCommand}
	var stdout, stderr bytes.Buffer
	if status := run(arguments, &stdout, &stderr); status != 0 {
		t.Fatalf("empty refresh status = %d, stderr = %s", status, &stderr)
	}
	want := fmt.Sprintf("gomad3 compatibility-pack refresh on %s/%s: 0 requests selected\n", runtime.GOOS, runtime.GOARCH)
	if stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("empty refresh stdout = %q, stderr = %q", &stdout, &stderr)
	}
	for _, test := range []struct {
		name      string
		arguments []string
		status    int
	}{
		{name: "invalid input", arguments: []string{"compatibility-pack", "refresh"}, status: 2},
		{name: "missing go", arguments: append(arguments, "--go="+filepath.Join(t.TempDir(), "missing-go")), status: 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			output := newRefreshOutput(t, 0)
			var stderr bytes.Buffer
			if status := run(test.arguments, output, &stderr); status != test.status || output.err != nil || stderr.Len() == 0 {
				t.Fatalf("status = %d, stdout write error = %v, stderr = %q", status, output.err, &stderr)
			}
		})
	}
	output := newRefreshOutput(t, 0)
	status := run(arguments, output, &stderr)
	if !errors.Is(output.err, syscall.EBADF) {
		t.Fatalf("stdout write error = %v, want EBADF", output.err)
	}
	t.Logf("stdout write observed EBADF after 0 successful writes")
	if status != 3 || stderr.Len() != 0 {
		t.Fatalf("stdout failure status = %d, stderr = %q", status, &stderr)
	}
}
