//go:build unix

package process

import (
	"errors"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestProtocolErrorCleanupRetainsTrustedTargetIdentity(t *testing.T) {
	identity := targetIdentity{pid: 100, pgid: 100}
	groups := targetCleanupPGIDs(identity, nil)
	if len(groups) != 1 || groups[0] != identity.pgid {
		t.Fatalf("cleanup groups = %v, want [%d]", groups, identity.pgid)
	}
	groups = targetCleanupPGIDs(identity, []supervisorReport{{PID: 100, PGID: 100}, {PID: 200, PGID: 200}, {PID: 300, PGID: 300}, {PID: 200, PGID: 200}})
	if len(groups) != 3 || groups[0] != 100 || groups[1] != 200 || groups[2] != 300 {
		t.Fatalf("mismatched cleanup groups = %v, want [100 200 300]", groups)
	}
}

func TestGroupProbeRequiresExplicitESRCHForDisappearance(t *testing.T) {
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
			exists, err := classifyGroupProbe(probe.err)
			if exists != probe.exists || (err != nil) != probe.fails {
				t.Fatalf("classifyGroupProbe() = (%v, %v)", exists, err)
			}
		})
	}
}

func TestProbeFailureDoesNotBypassTargetReap(t *testing.T) {
	target := exec.Command("sh", "-c", "while :; do :; done")
	target.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := target.Start(); err != nil {
		t.Fatal(err)
	}
	probeErr := errors.New("probe failed")
	probes := 0
	err := killReapTargetWithProbe(target, target.Process.Pid, time.Now().Add(time.Second), func(pgid int) (bool, error) {
		probes++
		if probes == 1 {
			return false, probeErr
		}
		return groupExists(pgid)
	})
	if !errors.Is(err, probeErr) {
		t.Fatalf("killReapTargetWithProbe() error = %v, want %v", err, probeErr)
	}
	if target.ProcessState == nil {
		t.Fatal("target was not reaped")
	}
}

func TestSupervisorReportStateMachineRejectsMalformedSequences(t *testing.T) {
	identity := targetIdentity{pid: 100, pgid: 100}
	started := supervisorReport{Kind: "started", PID: 100, PGID: 100}
	final := supervisorReport{Kind: "final", PID: 100, PGID: 100, Termination: TerminationExit, GroupGone: true}
	if _, _, err := validateSupervisorReports([]supervisorReport{started, final}, identity); err != nil {
		t.Fatal(err)
	}
	for name, reports := range map[string][]supervisorReport{
		"unknown":          {{Kind: "unknown"}, final},
		"duplicate-start":  {started, started},
		"out-of-order":     {final, started},
		"identity-change":  {started, {Kind: "final", PID: 101, PGID: 101, Termination: TerminationExit, GroupGone: true}},
		"group-remains":    {started, {Kind: "final", PID: 100, PGID: 100, Termination: TerminationExit}},
		"truncated-json":   {started, {Kind: "protocol_error", Error: "unexpected EOF"}},
		"unexpected-third": {started, final, final},
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := validateSupervisorReports(reports, identity); err == nil {
				t.Fatal("validateSupervisorReports() succeeded")
			}
		})
	}
	if _, _, err := validateSupervisorReports([]supervisorReport{started, final}, targetIdentity{err: errors.New("missing")}); err == nil {
		t.Fatal("validateSupervisorReports() accepted missing trusted identity")
	}
}
