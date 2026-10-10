//go:build unix

package execution

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/deterministicio"
)

func TestWatchdogFixtureReadinessWriteFailure(t *testing.T) {
	var terminal bytes.Buffer
	if err := deterministicio.WriteCompletion(&terminal, deterministicio.Transcript{Complete: true, SHA256: sha256.Sum256(nil)}); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"watchdog", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "read-only-stdout")
			if err := os.WriteFile(path, []byte("unchanged"), 0o600); err != nil {
				t.Fatal(err)
			}
			readOnly, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := readOnly.Close(); err != nil {
					t.Error(err)
				}
			}()
			if _, err := readOnly.Write([]byte("forbidden")); !errors.Is(err, syscall.EBADF) {
				t.Fatalf("read-only stdout error = %v, want EBADF", err)
			}
			terminalPath := filepath.Join(t.TempDir(), "terminal")
			terminalFile, err := os.Create(terminalPath)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := terminalFile.Close(); err != nil {
					t.Error(err)
				}
			}()
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestIOTerminalTargetHelper$", "--", mode, hex.EncodeToString(terminal.Bytes()))
			command.Env = []string{}
			command.Stdout = readOnly
			var stderr bytes.Buffer
			command.Stderr = &stderr
			descriptor := descriptorFor(targetStage, launchCapabilities{ioTranscript: true}, ioTerminalResource)
			for descriptor > 3+len(command.ExtraFiles) {
				command.ExtraFiles = append(command.ExtraFiles, readOnly)
			}
			command.ExtraFiles = append(command.ExtraFiles, terminalFile)
			err = command.Run()
			data, readErr := os.ReadFile(terminalPath)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !bytes.Equal(data, terminal.Bytes()) {
				t.Fatalf("terminal frame = %x, want %x", data, terminal.Bytes())
			}
			data, readErr = os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(data) != "unchanged" || stderr.Len() != 0 {
				t.Fatalf("input/stderr = %q/%q, want unchanged/no supplemental output", data, stderr.String())
			}
			if ctx.Err() != nil {
				t.Fatalf("readiness failure did not exit before deadline: %v", ctx.Err())
			}
			var exitError *exec.ExitError
			if !errors.As(err, &exitError) {
				t.Fatalf("child error = %v, want exit status 3", err)
			}
			if status := exitError.ExitCode(); status != 3 {
				t.Fatalf("child status = %d, want 3", status)
			}
		})
	}
}
