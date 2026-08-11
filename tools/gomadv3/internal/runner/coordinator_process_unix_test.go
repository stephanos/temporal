//go:build unix

package runner

import (
	"syscall"
	"testing"
)

func TestCoordinatorGroupProbeRequiresExplicitESRCHForDisappearance(t *testing.T) {
	for name, probe := range map[string]struct {
		err    error
		exists bool
		fails  bool
	}{
		"present": {exists: true},
		"denied":  {err: syscall.EPERM, exists: true},
		"gone":    {err: syscall.ESRCH},
		"unknown": {err: syscall.EINTR, fails: true},
	} {
		t.Run(name, func(t *testing.T) {
			exists, err := classifyCoordinatorGroupProbe(probe.err)
			if exists != probe.exists || (err != nil) != probe.fails {
				t.Fatalf("classifyCoordinatorGroupProbe() = (%v, %v)", exists, err)
			}
		})
	}
}
