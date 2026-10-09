package capabilityreview

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
	"go.temporal.io/server/tools/gomad3/target/internal/gocommand"
)

func TestListSourcePreservesActualCommandError(t *testing.T) {
	command := listFixtureCommand(t, "printf diagnostic >&2; exit 7")
	_, err := List(t.Context(), Request{GoCommand: command, Directory: t.TempDir(), Package: ".", OutputLimit: 1024, PackageLimit: 2})
	var commandErr *CommandError
	var exit *exec.ExitError
	if !errors.As(err, &commandErr) || !errors.As(err, &exit) || exit.ProcessState == nil || exit.ExitCode() != 7 || exit.Stderr != nil || string(commandErr.Stderr) != "diagnostic" {
		t.Fatalf("actual command outcome = %T %v", err, err)
	}
	_, err = List(t.Context(), Request{GoCommand: filepath.Join(t.TempDir(), "missing"), Directory: t.TempDir(), Package: ".", OutputLimit: 1024, PackageLimit: 2})
	if _, ok := err.(*os.PathError); !ok {
		t.Fatalf("startup outcome = %T %v, want original PathError", err, err)
	}
}

func TestListSourcePreservesContextOverrideForEveryFailure(t *testing.T) {
	infrastructure := errors.New("command infrastructure")
	for _, test := range []struct {
		name      string
		result    hostexec.Result
		operation error
	}{
		{name: "command", result: hostexec.Result{CommandError: errors.New("actual command")}},
		{name: "infrastructure", operation: infrastructure},
		{name: "overflow", result: hostexec.Result{Stdout: hostexec.Output{Truncated: true}}},
		{name: "watchdog", result: hostexec.Result{WatchdogTimeout: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			runner := gocommand.New(func(actual context.Context, request hostexec.Request) (hostexec.Result, error) {
				if actual != ctx || !request.PreserveCommandError {
					t.Fatalf("listing request = %#v", request)
				}
				return test.result, test.operation
			})
			packages, err := ListWith(ctx, Request{GoCommand: "go", Directory: "module", Package: ".", OutputLimit: 1024, PackageLimit: 2}, runner)
			if !errors.Is(err, context.Canceled) || packages != nil {
				t.Fatalf("original override = %#v, %T %v", packages, err, err)
			}
		})
	}
}
