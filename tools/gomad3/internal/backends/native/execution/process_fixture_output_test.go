package execution

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestTargetFixtureOutputPreservesProcessOutcome(t *testing.T) {
	for _, test := range []struct {
		name   string
		args   []string
		stdout string
		stderr string
		status int
	}{
		{name: "output", stdout: "target stdout\n", stderr: "target stderr\n", status: 7},
		{name: "choice-marker", stdout: "post-choice-marker\n"},
		{name: "choice-tape-readonly", stdout: "choice tape read-only\n"},
		{name: "choice-reorder", args: []string{"ab"}},
		{name: "choice-select"},
		{name: "choice-prefix-rng"},
	} {
		for _, failed := range []bool{false, true} {
			name := "healthy"
			if failed {
				name = "read-only-stdout"
			}
			t.Run(test.name+"/"+name, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "read-only")
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
				ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
				defer cancel()
				command := exec.CommandContext(ctx, os.Args[0], targetHelperArgs(test.name, test.args...)...)
				command.Env = []string{}
				var stdout, stderr bytes.Buffer
				command.Stdout = &stdout
				command.Stderr = &stderr
				if failed {
					command.Stdout = readOnly
				}
				if test.name == "choice-tape-readonly" {
					descriptor := descriptorFor(targetStage, launchCapabilities{choiceTrace: true, choiceReplayPlan: true}, choiceTapeResource)
					for descriptor >= 3+len(command.ExtraFiles) {
						command.ExtraFiles = append(command.ExtraFiles, readOnly)
					}
				}
				err = command.Run()
				if ctx.Err() != nil {
					t.Fatal(ctx.Err())
				}
				status := 0
				if err != nil {
					var exitError *exec.ExitError
					if !errors.As(err, &exitError) {
						t.Fatal(err)
					}
					status = exitError.ExitCode()
				}
				if status != test.status || stderr.String() != test.stderr {
					t.Fatalf("status/stderr = %d/%q, want %d/%q", status, stderr.String(), test.status, test.stderr)
				}
				if failed {
					if stdout.Len() != 0 {
						t.Fatalf("failed stdout = %q", stdout.String())
					}
				} else {
					output := stdout.String()
					switch test.name {
					case "choice-reorder":
						if output != "ab\n" && output != "ba\n" {
							t.Fatalf("reordered output = %q, want ab or ba followed by newline", output)
						}
					case "choice-select", "choice-prefix-rng":
						if len(output) != 9 || output[8] != '\n' || strings.Trim(output[:8], "ab") != "" {
							t.Fatalf("select output = %q, want eight a/b choices followed by newline", output)
						}
					default:
						if output != test.stdout {
							t.Fatalf("stdout = %q, want %q", output, test.stdout)
						}
					}
				}
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if string(data) != "unchanged" {
					t.Fatalf("read-only input = %q, want unchanged", data)
				}
			})
		}
	}
}
