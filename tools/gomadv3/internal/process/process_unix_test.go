//go:build unix

package process

import (
	"errors"
	"os"
	"os/exec"
	"reflect"
	"syscall"
	"testing"
	"time"
)

func TestStageFilesAreBuiltOnlyFromTheDescriptorPlan(t *testing.T) {
	capabilities := launchCapabilities{ioTranscript: true}
	layout := descriptorLayout(supervisorStage, capabilities)
	resources := make(resourceFiles, len(layout))
	for _, binding := range layout {
		file, err := os.CreateTemp(t.TempDir(), string(binding.resource))
		if err != nil {
			t.Fatal(err)
		}
		resource := file
		resources[binding.resource] = &resource
		defer func() {
			if resource != nil {
				_ = resource.Close()
			}
		}()
	}
	files, err := filesForStage(supervisorStage, capabilities, resources)
	if err != nil {
		t.Fatal(err)
	}
	for index, binding := range layout {
		if files[index] != *resources[binding.resource] {
			t.Fatalf("file %d = %p, want resource %q (%p)", index, files[index], binding.resource, *resources[binding.resource])
		}
	}
	delete(resources, ioTerminalResource)
	if _, err := filesForStage(supervisorStage, capabilities, resources); err == nil {
		t.Fatal("filesForStage() accepted a missing resource")
	}
	terminal := files[8]
	resources[ioTerminalResource] = &terminal
	if err := closeInheritedStage(supervisorStage, capabilities, resources); err != nil {
		t.Fatal(err)
	}
	if *resources[controlResource] != nil || *resources[ioTerminalResource] != nil {
		t.Fatal("inherited writers remained open after supervisor start")
	}
	if *resources[ioTranscriptResource] == nil || *resources[ioExpectedResource] == nil {
		t.Fatal("coordinator-owned I/O backings were closed after supervisor start")
	}
}

func TestLaunchResourcesOwnPipeCreationAndInheritance(t *testing.T) {
	resources := newLaunchResources(launchCapabilities{})
	controlWrite, err := resources.createPipe(controlResource, inheritRead, "supervisor control")
	if err != nil {
		t.Fatal(err)
	}
	defer resources.close()
	if controlWrite == nil || resources.files[controlResource] == nil || *resources.files[controlResource] == nil {
		t.Fatal("launch resource did not retain both pipe ends")
	}
	if controlWrite == *resources.files[controlResource] {
		t.Fatal("launch resource returned its inherited pipe end")
	}
	if err := resources.closeInherited(supervisorStage); err != nil {
		t.Fatal(err)
	}
	if *resources.files[controlResource] != nil {
		t.Fatal("launch resource retained an inherited end after process start")
	}
}

func TestDescriptorPlanOwnsEveryStageLayout(t *testing.T) {
	tests := map[string]struct {
		stage launchStage
		caps  launchCapabilities
		want  []descriptorBinding
	}{
		"supervisor baseline": {
			stage: supervisorStage,
			want:  []descriptorBinding{{controlResource, 3}, {reportResource, 4}, {stdoutResource, 5}, {stderrResource, 6}, {supervisorRequestResource, 7}, {worldRecordResource, 8}, {identityResource, 9}},
		},
		"bootstrap I/O and mounts": {
			stage: bootstrapStage, caps: launchCapabilities{ioTranscript: true, readOnlyMount: true},
			want: []descriptorBinding{{bootstrapRequestResource, 3}, {activationResource, 4}, {readinessResource, 5}, {worldConfigResource, 6}, {worldRecordResource, 7}, {identityResource, 8}, {ioTranscriptResource, 9}, {ioTerminalResource, 10}, {ioExpectedResource, 11}, {ioROMountRequestResource, 12}, {ioROMountResponseResource, 13}},
		},
		"target I/O and mounts": {
			stage: targetStage, caps: launchCapabilities{ioTranscript: true, readOnlyMount: true},
			want: []descriptorBinding{{worldConfigResource, 3}, {worldRecordResource, 4}, {ioConfigResource, 5}, {ioTranscriptResource, 6}, {ioTerminalResource, 7}, {ioExpectedResource, 8}, {ioROMountRequestResource, 9}, {ioROMountResponseResource, 10}},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := descriptorLayout(test.stage, test.caps); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("descriptorLayout() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestLaunchDescriptorNumbersRemainStable(t *testing.T) {
	if controlFD != 3 || reportFD != 4 || stdoutFD != 5 || stderrFD != 6 || requestFD != 7 || worldRecordFD != 8 || targetIdentityFD != 9 || ioTranscriptFD != 10 || ioTerminalFD != 11 || ioExpectedFD != 12 || ioROMountRequestFD != 13 || ioROMountResponseFD != 14 {
		t.Fatalf("coordinator descriptors changed: %d %d %d %d %d %d %d %d %d %d %d %d", controlFD, reportFD, stdoutFD, stderrFD, requestFD, worldRecordFD, targetIdentityFD, ioTranscriptFD, ioTerminalFD, ioExpectedFD, ioROMountRequestFD, ioROMountResponseFD)
	}
	if bootstrapRequestFD != 3 || bootstrapActivationFD != 4 || bootstrapReadinessFD != 5 || bootstrapWorldConfigFD != 6 || bootstrapWorldRecordFD != 7 || bootstrapIdentityFD != 8 || bootstrapIOTranscriptFD != 9 || bootstrapIOTerminalFD != 10 || bootstrapIOExpectedFD != 11 || bootstrapIOROMountRequestFD != 12 || bootstrapIOROMountResponseFD != 13 {
		t.Fatalf("supervisor descriptors changed: %d %d %d %d %d %d %d %d %d %d %d", bootstrapRequestFD, bootstrapActivationFD, bootstrapReadinessFD, bootstrapWorldConfigFD, bootstrapWorldRecordFD, bootstrapIdentityFD, bootstrapIOTranscriptFD, bootstrapIOTerminalFD, bootstrapIOExpectedFD, bootstrapIOROMountRequestFD, bootstrapIOROMountResponseFD)
	}
}

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
