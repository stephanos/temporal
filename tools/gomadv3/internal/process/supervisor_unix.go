//go:build unix

package process

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/worldpipe"
	"go.temporal.io/server/tools/gomadv3/world"
)

func SupervisorMain() (retErr error) {
	signal.Ignore(syscall.SIGTERM)
	startedAt := time.Now()
	control := os.NewFile(controlFD, "supervisor-control")
	report := os.NewFile(reportFD, "supervisor-report")
	stdout := os.NewFile(stdoutFD, "target-stdout")
	stderr := os.NewFile(stderrFD, "target-stderr")
	requestFile := os.NewFile(requestFD, "supervisor-request")
	worldRecord := os.NewFile(worldRecordFD, "target-world-record")
	identity := os.NewFile(targetIdentityFD, "target-identity")
	var ioTranscript, ioTerminal, ioExpected, ioROMountRequest, ioROMountResponse *os.File
	if control == nil || report == nil || stdout == nil || stderr == nil || requestFile == nil || worldRecord == nil || identity == nil {
		return fmt.Errorf("supervisor file descriptors are unavailable")
	}
	defer func() {
		retErr = errors.Join(retErr, closeOpenFile(&control), closeOpenFile(&report), closeOpenFile(&stdout), closeOpenFile(&stderr), closeOpenFile(&requestFile), closeOpenFile(&worldRecord), closeOpenFile(&identity), closeOpenFile(&ioTranscript), closeOpenFile(&ioTerminal), closeOpenFile(&ioExpected), closeOpenFile(&ioROMountRequest), closeOpenFile(&ioROMountResponse))
	}()

	var request supervisorRequest
	if err := json.NewDecoder(requestFile).Decode(&request); err != nil {
		return fmt.Errorf("decode supervisor request: %w", err)
	}
	if request.RunTimeout <= 0 || request.TerminateGrace < 0 || request.TerminateGrace > request.RunTimeout {
		return fmt.Errorf("invalid supervisor deadline")
	}
	if request.IOTranscriptLimit != 0 {
		ioTranscript = os.NewFile(ioTranscriptFD, "target-io-transcript")
		ioTerminal = os.NewFile(ioTerminalFD, "target-io-terminal")
		ioExpected = os.NewFile(ioExpectedFD, "target-io-expected")
		if ioTranscript == nil || ioTerminal == nil || ioExpected == nil {
			return errors.New("I/O transcript file descriptors are unavailable")
		}
	}
	if request.IOROMounts {
		ioROMountRequest = os.NewFile(ioROMountRequestFD, "target-io-ro-mount-request")
		ioROMountResponse = os.NewFile(ioROMountResponseFD, "target-io-ro-mount-response")
		if ioROMountRequest == nil || ioROMountResponse == nil {
			return errors.New("read-only mount file descriptors are unavailable")
		}
	}
	deadline := startedAt.Add(request.RunTimeout)
	if len(request.ExpectedWorldInitial) != 0 {
		initial, err := world.DecodeSnapshot(request.ExpectedWorldInitial)
		if err != nil {
			return fmt.Errorf("validate expected initial World snapshot before target activation: %w", err)
		}
		if uint64(initial.Config.Seed) != request.WorldSeed {
			return fmt.Errorf("expected initial World snapshot seed does not match target seed")
		}
	}

	if len(request.BootstrapCommand) == 0 || request.BootstrapCommand[0] == "" {
		return fmt.Errorf("target bootstrap command is required")
	}
	target := exec.Command(request.BootstrapCommand[0], request.BootstrapCommand[1:]...)
	target.Dir = request.Dir
	target.Env = append(os.Environ(), "GOMADV3_TARGET_BOOTSTRAP=1")
	target.Stdout = stdout
	target.Stderr = stderr
	capabilities := launchCapabilities{ioTranscript: ioTranscript != nil, readOnlyMount: ioROMountRequest != nil}
	resources := newLaunchResources(capabilities)
	defer func() { retErr = errors.Join(retErr, resources.close()) }()
	bootstrapWrite, err := resources.createPipe(bootstrapRequestResource, inheritRead, "target bootstrap request")
	if err != nil {
		return err
	}
	activationWrite, err := resources.createPipe(activationResource, inheritRead, "target activation")
	if err != nil {
		return err
	}
	readinessRead, err := resources.createPipe(readinessResource, inheritWrite, "target readiness")
	if err != nil {
		return err
	}
	configWrite, err := resources.createPipe(worldConfigResource, inheritRead, "target World configuration")
	if err != nil {
		return err
	}
	encodedConfig, err := worldpipe.Encode(worldpipe.Config{
		TransitionLimit: request.WorldTransitionLimit,
		Seed:            request.WorldSeed,
		ExpectedInitial: request.ExpectedWorldInitial,
	})
	if err != nil {
		return err
	}
	resources.bind(worldRecordResource, &worldRecord)
	resources.bind(identityResource, &identity)
	if ioTranscript != nil {
		resources.bind(ioTranscriptResource, &ioTranscript)
		resources.bind(ioTerminalResource, &ioTerminal)
		resources.bind(ioExpectedResource, &ioExpected)
	}
	if ioROMountRequest != nil {
		resources.bind(ioROMountRequestResource, &ioROMountRequest)
		resources.bind(ioROMountResponseResource, &ioROMountResponse)
	}
	target.ExtraFiles, err = resources.extraFiles(bootstrapStage)
	if err != nil {
		return err
	}
	target.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := target.Start(); err != nil {
		return fmt.Errorf("start target bootstrap: %w", err)
	}
	targetPGID := target.Process.Pid
	pid := target.Process.Pid
	if err := closeFile(resources.files[identityResource]); err != nil {
		return errors.Join(fmt.Errorf("close inherited target identity pipe: %w", err), bootstrapWrite.Close(), activationWrite.Close(), readinessRead.Close(), configWrite.Close(), killReapTarget(target, targetPGID, deadline))
	}
	encoder := json.NewEncoder(report)
	if err := encoder.Encode(supervisorReport{Kind: "started", PID: pid, PGID: targetPGID}); err != nil {
		return errors.Join(fmt.Errorf("report target start: %w", err), bootstrapWrite.Close(), activationWrite.Close(), readinessRead.Close(), configWrite.Close(), killReapTarget(target, targetPGID, deadline))
	}
	if closeErr := resources.closeInherited(bootstrapStage); closeErr != nil {
		return errors.Join(fmt.Errorf("close inherited target bootstrap pipe ends: %w", closeErr), bootstrapWrite.Close(), activationWrite.Close(), readinessRead.Close(), configWrite.Close(), killReapTarget(target, targetPGID, deadline))
	}
	bootstrapRequest := targetBootstrapRequest{Command: request.Command, Args: request.Args, Argv0: request.Argv0, Dir: request.Dir, Env: request.Env, IOConfig: request.IOConfig, IOTranscriptLimit: request.IOTranscriptLimit, IOReplay: request.IOReplay, IOROMounts: request.IOROMounts}
	if err := json.NewEncoder(bootstrapWrite).Encode(bootstrapRequest); err != nil {
		return errors.Join(fmt.Errorf("write target bootstrap request: %w", err), bootstrapWrite.Close(), activationWrite.Close(), readinessRead.Close(), configWrite.Close(), killReapTarget(target, targetPGID, deadline))
	}
	if err := bootstrapWrite.Close(); err != nil {
		return errors.Join(fmt.Errorf("close target bootstrap request: %w", err), activationWrite.Close(), readinessRead.Close(), configWrite.Close(), killReapTarget(target, targetPGID, deadline))
	}
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		cleanupErr := killReapTarget(target, targetPGID, deadline)
		return errors.Join(fmt.Errorf("read target process group: %w", err), activationWrite.Close(), readinessRead.Close(), configWrite.Close(), cleanupErr)
	}
	if pgid != pid {
		cleanupErr := killReapTarget(target, targetPGID, deadline)
		return errors.Join(fmt.Errorf("target process group %d does not match leader %d", pgid, pid), activationWrite.Close(), readinessRead.Close(), configWrite.Close(), cleanupErr)
	}
	if err := readinessRead.SetReadDeadline(deadline); err != nil {
		return errors.Join(fmt.Errorf("set target bootstrap readiness deadline: %w", err), activationWrite.Close(), readinessRead.Close(), configWrite.Close(), killReapTarget(target, pgid, deadline))
	}
	var ready [1]byte
	if _, err := io.ReadFull(readinessRead, ready[:]); err != nil {
		return errors.Join(fmt.Errorf("read target bootstrap readiness: %w", err), activationWrite.Close(), readinessRead.Close(), configWrite.Close(), killReapTarget(target, pgid, deadline))
	}
	if ready[0] != 1 {
		return errors.Join(fmt.Errorf("invalid target bootstrap readiness"), activationWrite.Close(), readinessRead.Close(), configWrite.Close(), killReapTarget(target, pgid, deadline))
	}
	if err := readinessRead.Close(); err != nil {
		return errors.Join(fmt.Errorf("close target bootstrap readiness: %w", err), activationWrite.Close(), configWrite.Close(), killReapTarget(target, pgid, deadline))
	}
	if _, err := activationWrite.Write([]byte{1}); err != nil {
		return errors.Join(fmt.Errorf("activate target bootstrap: %w", err), activationWrite.Close(), configWrite.Close(), killReapTarget(target, pgid, deadline))
	}
	if err := activationWrite.Close(); err != nil {
		return errors.Join(fmt.Errorf("close target activation: %w", err), configWrite.Close(), killReapTarget(target, pgid, deadline))
	}
	if _, err := io.Copy(configWrite, bytes.NewReader(encodedConfig)); err != nil {
		return errors.Join(fmt.Errorf("write target World configuration: %w", err), configWrite.Close(), killReapTarget(target, pgid, deadline))
	}
	if err := configWrite.Close(); err != nil {
		return errors.Join(fmt.Errorf("close target World configuration: %w", err), killReapTarget(target, pgid, deadline))
	}
	if closeErr := errors.Join(closeOpenFile(&stdout), closeOpenFile(&stderr)); closeErr != nil {
		return errors.Join(fmt.Errorf("close inherited target output pipes: %w", closeErr), killReapTarget(target, pgid, deadline))
	}

	waited := make(chan error, 1)
	go func() {
		waited <- target.Wait()
	}()
	controlLost := make(chan struct{}, 1)
	go func() {
		_, _ = io.Copy(io.Discard, control)
		controlLost <- struct{}{}
	}()

	cleanupReserve := min(50*time.Millisecond, request.RunTimeout/4)
	killAt := deadline.Add(-cleanupReserve)
	termAfter := max(time.Until(killAt)-request.TerminateGrace, 0)
	termTimer := time.NewTimer(termAfter)
	defer termTimer.Stop()

	var waitErr error
	watchdogTimeout := false
	cancelled := false
	terminationStarted := false
	select {
	case waitErr = <-waited:
	case <-termTimer.C:
		watchdogTimeout = true
		terminationStarted = true
	case <-controlLost:
		cancelled = true
		terminationStarted = true
	}

	if waitErr == nil && !terminationStarted && target.ProcessState == nil {
		return fmt.Errorf("target wait completed without process state")
	}
	groupPresent, err := groupExists(pgid)
	if err != nil {
		return errors.Join(fmt.Errorf("probe target process group: %w", err), cleanupTargetAfterProbeError(target, pgid, waited, deadline))
	}
	if !terminationStarted && groupPresent {
		terminationStarted = true
	}
	if terminationStarted {
		if err := signalGroup(pgid, syscall.SIGTERM); err != nil {
			return fmt.Errorf("terminate target process group: %w", err)
		}
		graceTimer := time.NewTimer(max(min(request.TerminateGrace, time.Until(killAt)), 0))
		poll := time.NewTicker(5 * time.Millisecond)
		graceExpired := false
		for !graceExpired {
			groupPresent, err = groupExists(pgid)
			if err != nil {
				return errors.Join(fmt.Errorf("probe target process group during termination grace: %w", err), cleanupTargetAfterProbeError(target, pgid, waited, deadline))
			}
			if !groupPresent {
				break
			}
			select {
			case waitErr = <-waited:
			case <-poll.C:
			case <-graceTimer.C:
				graceExpired = true
			}
		}
		poll.Stop()
		if !graceTimer.Stop() {
			select {
			case <-graceTimer.C:
			default:
			}
		}
		groupPresent, err = groupExists(pgid)
		if err != nil {
			return errors.Join(fmt.Errorf("probe target process group before kill: %w", err), cleanupTargetAfterProbeError(target, pgid, waited, deadline))
		}
		if groupPresent {
			if err := signalGroup(pgid, syscall.SIGKILL); err != nil {
				return fmt.Errorf("kill target process group: %w", err)
			}
		}
		if target.ProcessState == nil {
			select {
			case waitErr = <-waited:
			case <-time.After(max(time.Until(deadline), 0)):
				return fmt.Errorf("target could not be reaped before the process deadline")
			}
		}
		poll = time.NewTicker(5 * time.Millisecond)
		for time.Now().Before(deadline) {
			groupPresent, err = groupExists(pgid)
			if err != nil {
				return errors.Join(fmt.Errorf("probe target process group after kill: %w", err), cleanupTargetAfterProbeError(target, pgid, waited, deadline))
			}
			if !groupPresent {
				break
			}
			<-poll.C
		}
		poll.Stop()
	}

	if target.ProcessState == nil {
		return fmt.Errorf("target was not reaped")
	}
	if waitErr != nil {
		var exitError *exec.ExitError
		if !errors.As(waitErr, &exitError) {
			return fmt.Errorf("wait for target: %w", waitErr)
		}
	}
	groupPresent, err = groupExists(pgid)
	if err != nil {
		return errors.Join(fmt.Errorf("verify target process group disappearance: %w", err), cleanupTargetAfterProbeError(target, pgid, waited, deadline))
	}
	groupGone := !groupPresent
	if !groupGone {
		return fmt.Errorf("target process group %d remains after cleanup", pgid)
	}

	final := supervisorReport{
		Kind:            "final",
		PID:             pid,
		PGID:            pgid,
		WatchdogTimeout: watchdogTimeout,
		Cancelled:       cancelled,
		GroupGone:       groupGone,
	}
	status, ok := target.ProcessState.Sys().(syscall.WaitStatus)
	if !ok {
		return fmt.Errorf("target wait status has type %T", target.ProcessState.Sys())
	}
	if status.Signaled() {
		final.Termination = TerminationSignal
		final.Signal = status.Signal().String()
	} else {
		final.Termination = TerminationExit
		final.ExitCode = status.ExitStatus()
	}
	if err := encoder.Encode(final); err != nil {
		return fmt.Errorf("report target result: %w", err)
	}
	return nil
}

func closeOpenFile(file **os.File) error {
	if file == nil || *file == nil {
		return nil
	}
	err := (*file).Close()
	*file = nil
	return err
}

func signalGroup(pgid int, signal syscall.Signal) error {
	if pgid <= 0 {
		return fmt.Errorf("invalid process group %d", pgid)
	}
	if err := syscall.Kill(-pgid, signal); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}

func groupExists(pgid int) (bool, error) {
	if pgid <= 0 {
		return false, fmt.Errorf("invalid process group %d", pgid)
	}
	return classifyGroupProbe(syscall.Kill(-pgid, 0))
}

func classifyGroupProbe(err error) (bool, error) {
	switch {
	case err == nil, errors.Is(err, syscall.EPERM):
		return true, nil
	case errors.Is(err, syscall.ESRCH):
		return false, nil
	default:
		return false, err
	}
}

func cleanupTargetAfterProbeError(target *exec.Cmd, pgid int, waited <-chan error, deadline time.Time) error {
	signalErr := signalGroup(pgid, syscall.SIGKILL)
	killErr := target.Process.Kill()
	if errors.Is(killErr, os.ErrProcessDone) {
		killErr = nil
	}
	if target.ProcessState != nil {
		return errors.Join(signalErr, killErr)
	}
	timer := time.NewTimer(max(time.Until(deadline), 0))
	defer timer.Stop()
	select {
	case waitErr := <-waited:
		var exitError *exec.ExitError
		if errors.As(waitErr, &exitError) {
			waitErr = nil
		}
		return errors.Join(signalErr, killErr, waitErr)
	case <-timer.C:
		return errors.Join(signalErr, killErr, fmt.Errorf("target could not be reaped after process-group probe failure"))
	}
}

func killGroupBounded(pgid int, deadline time.Time) error {
	if err := signalGroup(pgid, syscall.SIGKILL); err != nil {
		return err
	}
	poll := time.NewTicker(5 * time.Millisecond)
	defer poll.Stop()
	for {
		exists, err := groupExists(pgid)
		if err != nil {
			return fmt.Errorf("probe target process group during cleanup: %w", err)
		}
		if !exists {
			return nil
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return fmt.Errorf("target process group %d remains after cleanup", pgid)
		}
		select {
		case <-poll.C:
		case <-time.After(remaining):
		}
	}
}

func killReapTarget(target *exec.Cmd, pgid int, deadline time.Time) error {
	return killReapTargetWithProbe(target, pgid, deadline, groupExists)
}

func killReapTargetWithProbe(target *exec.Cmd, pgid int, deadline time.Time, probe func(int) (bool, error)) error {
	signalErr := signalGroup(pgid, syscall.SIGKILL)
	killErr := target.Process.Kill()
	if errors.Is(killErr, os.ErrProcessDone) {
		killErr = nil
	}
	waited := make(chan error, 1)
	go func() {
		waited <- waitForTerminated(target)
	}()
	poll := time.NewTicker(5 * time.Millisecond)
	defer poll.Stop()
	reaped := false
	var probeFailure error
	var waitFailure error
	groupPresent, probeErr := probe(pgid)
	if probeErr != nil {
		probeFailure = errors.Join(probeFailure, fmt.Errorf("probe target process group during reap: %w", probeErr))
	}
	groupGone := probeErr == nil && !groupPresent
	for !reaped || !groupGone {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			var reapErr, groupErr error
			if !reaped {
				reapErr = fmt.Errorf("target could not be reaped before the process deadline")
			}
			if !groupGone {
				groupErr = fmt.Errorf("target process group %d remains after cleanup", pgid)
			}
			return errors.Join(signalErr, killErr, waitFailure, probeFailure, reapErr, groupErr)
		}
		select {
		case waitErr := <-waited:
			reaped = true
			waited = nil
			if waitErr != nil {
				waitFailure = errors.Join(waitFailure, waitErr)
			}
		case <-poll.C:
			groupPresent, probeErr = probe(pgid)
			if probeErr != nil {
				probeFailure = errors.Join(probeFailure, fmt.Errorf("probe target process group during reap: %w", probeErr))
				continue
			}
			groupGone = !groupPresent
		case <-time.After(remaining):
		}
	}
	return errors.Join(signalErr, killErr, waitFailure, probeFailure)
}

func cleanupEarlySupervisor(command *exec.Cmd, control, report *os.File, identities <-chan targetIdentity, deadline time.Time) error {
	closeErr := control.Close()
	waited := make(chan error, 1)
	go func() {
		waited <- waitForTerminated(command)
	}()
	remaining := max(time.Until(deadline), 0)
	reserve := min(100*time.Millisecond, remaining/2)
	grace := time.NewTimer(max(remaining-reserve, 0))
	var waitErr error
	reaped := false
	select {
	case waitErr = <-waited:
		reaped = true
		if !grace.Stop() {
			select {
			case <-grace.C:
			default:
			}
		}
	case <-grace.C:
		killErr := command.Process.Kill()
		if errors.Is(killErr, os.ErrProcessDone) {
			killErr = nil
		}
		closeErr = errors.Join(closeErr, killErr)
		reap := time.NewTimer(max(time.Until(deadline)-reserve/2, 0))
		select {
		case waitErr = <-waited:
			reaped = true
		case <-reap.C:
			waitErr = fmt.Errorf("supervisor could not be reaped before the process deadline")
		}
		if !reap.Stop() {
			select {
			case <-reap.C:
			default:
			}
		}
	}
	var startedPGID int
	if identities != nil {
		identityTimer := time.NewTimer(max(time.Until(deadline)/2, 0))
		select {
		case identity := <-identities:
			if identity.err == nil {
				startedPGID = identity.pgid
			}
		case <-identityTimer.C:
		}
		if !identityTimer.Stop() {
			select {
			case <-identityTimer.C:
			default:
			}
		}
	}
	if reaped {
		decoder := json.NewDecoder(report)
		for {
			var decoded supervisorReport
			if err := decoder.Decode(&decoded); err != nil {
				if !errors.Is(err, io.EOF) {
					closeErr = errors.Join(closeErr, fmt.Errorf("decode supervisor report during cleanup: %w", err))
				}
				break
			}
			if decoded.Kind == "started" && startedPGID == 0 {
				startedPGID = decoded.PGID
			}
		}
	} else if startedPGID == 0 {
		readBudget := min(5*time.Millisecond, max(time.Until(deadline)/2, 0))
		if readBudget > 0 {
			if err := report.SetReadDeadline(time.Now().Add(readBudget)); err != nil {
				closeErr = errors.Join(closeErr, err)
			} else {
				var decoded supervisorReport
				if err := json.NewDecoder(report).Decode(&decoded); err == nil && decoded.Kind == "started" {
					startedPGID = decoded.PGID
				}
			}
		}
	}
	if startedPGID != 0 {
		closeErr = errors.Join(closeErr, killGroupBounded(startedPGID, deadline))
	}
	return errors.Join(closeErr, waitErr)
}

func readTargetIdentity(reader io.Reader) targetIdentity {
	var encoded [16]byte
	if _, err := io.ReadFull(reader, encoded[:]); err != nil {
		return targetIdentity{err: err}
	}
	pid := binary.BigEndian.Uint64(encoded[:8])
	pgid := binary.BigEndian.Uint64(encoded[8:])
	if pid == 0 || pgid == 0 || pid > uint64(^uint(0)>>1) || pgid > uint64(^uint(0)>>1) {
		return targetIdentity{err: fmt.Errorf("invalid target identity")}
	}
	return targetIdentity{pid: int(pid), pgid: int(pgid)}
}

func cleanupTargetGroups(identity targetIdentity, reports []supervisorReport, deadline time.Time) error {
	var result error
	for _, pgid := range targetCleanupPGIDs(identity, reports) {
		result = errors.Join(result, killGroupBounded(pgid, deadline))
	}
	return result
}

func targetCleanupPGIDs(identity targetIdentity, reports []supervisorReport) []int {
	groups := make([]int, 0, len(reports)+1)
	seen := make(map[int]struct{}, len(reports)+1)
	if identity.err == nil && identity.pgid > 0 {
		groups = append(groups, identity.pgid)
		seen[identity.pgid] = struct{}{}
	}
	for _, report := range reports {
		if report.PGID <= 0 {
			continue
		}
		if _, found := seen[report.PGID]; found {
			continue
		}
		groups = append(groups, report.PGID)
		seen[report.PGID] = struct{}{}
	}
	return groups
}

func waitForTerminated(command *exec.Cmd) error {
	err := command.Wait()
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return nil
	}
	return err
}
