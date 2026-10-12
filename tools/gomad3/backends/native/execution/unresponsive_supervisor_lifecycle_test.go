//go:build unix

package execution

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
)

func TestUnresponsiveSupervisorLifecycle(t *testing.T) {
	result, err := hostexec.Run(t.Context(), hostexec.Request{
		Command:              []string{os.Args[0], "-test.run=^TestUnresponsiveSupervisorHelper$", "-test.count=1"},
		Dir:                  t.TempDir(),
		Env:                  []string{"GOMAD3_PROCESS_SUPERVISOR=1"},
		Timeout:              300 * time.Millisecond,
		TerminateGrace:       100 * time.Millisecond,
		OutputLimit:          1024,
		PreserveCommandError: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.WatchdogTimeout || result.Cancelled || result.Termination != hostexec.TerminationSignal || result.SignalNumber != int(syscall.SIGKILL) || !result.GroupGone {
		t.Fatalf("unresponsive supervisor lifecycle = %#v", result)
	}
	var exitError *exec.ExitError
	if !errors.As(result.CommandError, &exitError) || exitError.Pid() != result.PID {
		t.Fatalf("supervisor wait error = %v, want reaped child %d", result.CommandError, result.PID)
	}
	if result.Stdout.TotalBytes != 0 || result.Stderr.TotalBytes != 0 {
		t.Fatalf("unresponsive supervisor output = %q/%q", result.Stdout.Bytes, result.Stderr.Bytes)
	}
}
