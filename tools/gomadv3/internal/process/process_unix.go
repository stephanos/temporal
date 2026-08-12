//go:build unix

package process

import (
	"bytes"
	"context"
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

	"go.temporal.io/server/tools/gomadv3/internal/romount"
	"go.temporal.io/server/tools/gomadv3/internal/worldpipe"
	"go.temporal.io/server/tools/gomadv3/world"
)

const (
	controlFD = 3 + iota
	reportFD
	stdoutFD
	stderrFD
	requestFD
	worldRecordFD
	targetIdentityFD
	ioTranscriptFD
	ioTerminalFD
	ioExpectedFD
	ioROMountRequestFD
	ioROMountResponseFD
)

const (
	bootstrapRequestFD = 3 + iota
	bootstrapActivationFD
	bootstrapReadinessFD
	bootstrapWorldConfigFD
	bootstrapWorldRecordFD
	bootstrapIdentityFD
	bootstrapIOTranscriptFD
	bootstrapIOTerminalFD
	bootstrapIOExpectedFD
	bootstrapIOROMountRequestFD
	bootstrapIOROMountResponseFD
)

type targetIdentity struct {
	pid  int
	pgid int
	err  error
}

type supervisorRequest struct {
	BootstrapCommand     []string      `json:"bootstrap_command"`
	Command              string        `json:"command"`
	Args                 []string      `json:"args"`
	Argv0                string        `json:"argv0"`
	Dir                  string        `json:"dir"`
	Env                  []string      `json:"env"`
	RunTimeout           time.Duration `json:"run_timeout"`
	TerminateGrace       time.Duration `json:"terminate_grace"`
	WorldTransitionLimit uint64        `json:"world_transition_limit"`
	WorldSeed            uint64        `json:"world_seed"`
	ExpectedWorldInitial []byte        `json:"expected_world_initial"`
	IOConfig             []byte        `json:"io_config"`
	IOTranscriptLimit    uint64        `json:"io_transcript_limit"`
	IOReplay             bool          `json:"io_replay"`
	IOROMounts           bool          `json:"io_ro_mounts"`
}

type targetBootstrapRequest struct {
	Command           string   `json:"command"`
	Args              []string `json:"args"`
	Argv0             string   `json:"argv0"`
	Dir               string   `json:"dir"`
	Env               []string `json:"env"`
	IOConfig          []byte   `json:"io_config"`
	IOTranscriptLimit uint64   `json:"io_transcript_limit"`
	IOReplay          bool     `json:"io_replay"`
	IOROMounts        bool     `json:"io_ro_mounts"`
}

type supervisorReport struct {
	Kind            string      `json:"kind"`
	PID             int         `json:"pid,omitempty"`
	PGID            int         `json:"pgid,omitempty"`
	Termination     Termination `json:"termination,omitempty"`
	ExitCode        int         `json:"exit_code,omitempty"`
	Signal          string      `json:"signal,omitempty"`
	WatchdogTimeout bool        `json:"watchdog_timeout,omitempty"`
	Cancelled       bool        `json:"cancelled,omitempty"`
	GroupGone       bool        `json:"group_gone,omitempty"`
	Error           string      `json:"error,omitempty"`
}

func Run(ctx context.Context, request Request) (result Result, retErr error) {
	if err := validateRequest(request); err != nil {
		return Result{}, err
	}
	timeout, err := effectiveTimeout(ctx, request.RunTimeout)
	if err != nil {
		return Result{}, err
	}
	deadline := time.Now().Add(timeout)

	stdoutCapture, err := NewOutputCapture(request.OutputLimit)
	if err != nil {
		return Result{}, fmt.Errorf("create stdout capture: %w", err)
	}
	stderrCapture, err := NewOutputCapture(request.OutputLimit)
	if err != nil {
		return Result{}, fmt.Errorf("create stderr capture: %w", err)
	}
	worldCapture, err := NewOutputCapture(request.WorldRecordLimit)
	if err != nil {
		return Result{}, fmt.Errorf("create World record capture: %w", err)
	}
	var ioBacking *ioTranscriptBacking
	if request.IOTranscriptLimit != 0 {
		ioBacking, err = newIOTranscriptBacking(request.IOTranscriptLimit, request.IOReplay, request.ExpectedIOTranscript)
		if err != nil {
			return Result{}, err
		}
		defer func() {
			retErr = errors.Join(retErr, ioBacking.close())
		}()
	}
	var readOnlyMountBroker *romount.Broker
	var mountRequestRead, mountRequestWrite, mountResponseRead, mountResponseWrite *os.File
	if len(request.IOROMounts) != 0 {
		if request.IOROMountReplay == nil {
			readOnlyMountBroker, err = romount.Prepare(request.IOROMounts, request.IOROMountLimits)
		} else {
			readOnlyMountBroker, err = romount.PrepareReplay(request.IOROMounts, request.IOROMountLimits, *request.IOROMountReplay)
		}
		if err != nil {
			return Result{}, fmt.Errorf("prepare read-only mounts: %w", err)
		}
		defer func() { retErr = errors.Join(retErr, readOnlyMountBroker.Close()) }()
		mountRequestRead, mountRequestWrite, err = os.Pipe()
		if err != nil {
			return Result{}, fmt.Errorf("create read-only mount request pipe: %w", err)
		}
		defer mountRequestRead.Close()
		defer mountRequestWrite.Close()
		mountResponseRead, mountResponseWrite, err = os.Pipe()
		if err != nil {
			return Result{}, fmt.Errorf("create read-only mount response pipe: %w", err)
		}
		defer mountResponseRead.Close()
		defer mountResponseWrite.Close()
	}

	controlRead, controlWrite, err := os.Pipe()
	if err != nil {
		return Result{}, fmt.Errorf("create supervisor control pipe: %w", err)
	}
	defer controlRead.Close()
	defer controlWrite.Close()
	reportRead, reportWrite, err := os.Pipe()
	if err != nil {
		return Result{}, fmt.Errorf("create supervisor report pipe: %w", err)
	}
	defer reportRead.Close()
	defer reportWrite.Close()
	stdoutRead, stdoutWrite, err := os.Pipe()
	if err != nil {
		return Result{}, fmt.Errorf("create target stdout pipe: %w", err)
	}
	defer stdoutRead.Close()
	defer stdoutWrite.Close()
	stderrRead, stderrWrite, err := os.Pipe()
	if err != nil {
		return Result{}, fmt.Errorf("create target stderr pipe: %w", err)
	}
	defer stderrRead.Close()
	defer stderrWrite.Close()
	requestRead, requestWrite, err := os.Pipe()
	if err != nil {
		return Result{}, fmt.Errorf("create supervisor request pipe: %w", err)
	}
	defer requestRead.Close()
	defer requestWrite.Close()
	worldRead, worldWrite, err := os.Pipe()
	if err != nil {
		return Result{}, fmt.Errorf("create World record pipe: %w", err)
	}
	defer worldRead.Close()
	defer worldWrite.Close()
	identityRead, identityWrite, err := os.Pipe()
	if err != nil {
		return Result{}, fmt.Errorf("create target identity pipe: %w", err)
	}
	defer identityRead.Close()
	defer identityWrite.Close()

	command := exec.Command(request.SupervisorCommand[0], request.SupervisorCommand[1:]...)
	command.Env = append(os.Environ(), "GOMADV3_PROCESS_SUPERVISOR=1")
	command.ExtraFiles = []*os.File{controlRead, reportWrite, stdoutWrite, stderrWrite, requestRead, worldWrite, identityWrite}
	if ioBacking != nil {
		command.ExtraFiles = append(command.ExtraFiles, ioBacking.file, ioBacking.terminalWrite, ioBacking.expected)
	}
	if readOnlyMountBroker != nil {
		command.ExtraFiles = append(command.ExtraFiles, mountRequestWrite, mountResponseRead)
	}
	if err := command.Start(); err != nil {
		return Result{}, fmt.Errorf("start supervisor: %w", err)
	}
	if closeErr := errors.Join(
		controlRead.Close(), reportWrite.Close(), stdoutWrite.Close(), stderrWrite.Close(), requestRead.Close(), worldWrite.Close(), identityWrite.Close(),
		closeOpenFile(&mountRequestWrite), closeOpenFile(&mountResponseRead),
	); closeErr != nil {
		return Result{}, errors.Join(fmt.Errorf("close inherited supervisor pipe ends: %w", closeErr), cleanupEarlySupervisor(command, controlWrite, reportRead, nil, deadline))
	}
	var ioTerminal <-chan []byte
	if ioBacking != nil {
		if err := closeFile(&ioBacking.terminalWrite); err != nil {
			return Result{}, errors.Join(fmt.Errorf("close inherited I/O terminal writer: %w", err), cleanupEarlySupervisor(command, controlWrite, reportRead, nil, deadline))
		}
		terminal := make(chan []byte, 1)
		go func() {
			bytes, _ := io.ReadAll(io.LimitReader(ioBacking.terminalRead, ioTerminalBytes+1))
			terminal <- bytes
		}()
		ioTerminal = terminal
	}
	identities := make(chan targetIdentity, 1)
	go func() { identities <- readTargetIdentity(identityRead) }()
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return Result{}, errors.Join(context.DeadlineExceeded, cleanupEarlySupervisor(command, controlWrite, reportRead, identities, deadline))
	}
	wireTimeout := targetTimeout(remaining)

	wireRequest := supervisorRequest{
		BootstrapCommand:     append([]string(nil), request.BootstrapCommand...),
		Command:              request.Command,
		Args:                 request.Args,
		Argv0:                request.Argv0,
		Dir:                  request.Dir,
		Env:                  request.Env,
		RunTimeout:           wireTimeout,
		TerminateGrace:       min(request.TerminateGrace, wireTimeout),
		WorldTransitionLimit: request.WorldTransitionLimit,
		WorldSeed:            request.WorldSeed,
		ExpectedWorldInitial: append([]byte(nil), request.ExpectedWorldInitial...),
		IOConfig:             append([]byte(nil), request.IOConfig...),
		IOTranscriptLimit:    request.IOTranscriptLimit,
		IOReplay:             request.IOReplay,
		IOROMounts:           readOnlyMountBroker != nil,
	}
	if err := json.NewEncoder(requestWrite).Encode(wireRequest); err != nil {
		cleanupErr := cleanupEarlySupervisor(command, controlWrite, reportRead, identities, deadline)
		return Result{}, errors.Join(fmt.Errorf("send supervisor request: %w", err), cleanupErr)
	}
	if err := requestWrite.Close(); err != nil {
		cleanupErr := cleanupEarlySupervisor(command, controlWrite, reportRead, identities, deadline)
		return Result{}, errors.Join(fmt.Errorf("close supervisor request: %w", err), cleanupErr)
	}
	var mountServed <-chan error
	if readOnlyMountBroker != nil {
		served := make(chan error, 1)
		go func() { served <- readOnlyMountBroker.Serve(mountRequestRead, mountResponseWrite) }()
		mountServed = served
	}

	type captureResult struct {
		name string
		err  error
	}
	captures := make(chan captureResult, 3)
	go func() {
		writer := newCaptureWriter(stdoutCapture, request.StdoutHead, func() { _ = controlWrite.Close() })
		_, copyErr := io.Copy(writer, stdoutRead)
		captures <- captureResult{name: "stdout", err: errors.Join(copyErr, writer.err)}
	}()
	go func() {
		writer := newCaptureWriter(stderrCapture, request.StderrHead, func() { _ = controlWrite.Close() })
		_, copyErr := io.Copy(writer, stderrRead)
		captures <- captureResult{name: "stderr", err: errors.Join(copyErr, writer.err)}
	}()
	go func() {
		_, copyErr := io.Copy(worldCapture, worldRead)
		captures <- captureResult{name: "World record", err: copyErr}
	}()

	reports := make(chan []supervisorReport, 1)
	go func() {
		var decoded []supervisorReport
		decoder := json.NewDecoder(reportRead)
		for {
			var report supervisorReport
			if decodeErr := decoder.Decode(&report); decodeErr != nil {
				if !errors.Is(decodeErr, io.EOF) {
					decoded = append(decoded, supervisorReport{Kind: "protocol_error", Error: decodeErr.Error()})
				}
				break
			}
			decoded = append(decoded, report)
		}
		reports <- decoded
	}()

	contextDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = controlWrite.Close()
		case <-contextDone:
		}
	}()

	waits := make(chan error, 1)
	go func() {
		waits <- command.Wait()
	}()
	remaining = time.Until(deadline)
	cleanupReserve := min(150*time.Millisecond, remaining/3)
	parentTimeout := max(remaining-cleanupReserve, 0)
	parentTimer := time.NewTimer(parentTimeout)
	defer parentTimer.Stop()
	supervisorTimedOut := false
	var waitErr error
	select {
	case waitErr = <-waits:
	case <-parentTimer.C:
		supervisorTimedOut = true
		_ = controlWrite.Close()
		killErr := command.Process.Kill()
		reapTimer := time.NewTimer(cleanupReserve / 2)
		select {
		case waitErr = <-waits:
		case <-reapTimer.C:
			var groupCleanupErr error
			startedWait := time.NewTimer(max(time.Until(deadline)/2, 0))
			select {
			case identity := <-identities:
				if identity.err == nil {
					groupCleanupErr = killGroupBounded(identity.pgid, deadline)
				}
			case <-startedWait.C:
			}
			if !startedWait.Stop() {
				select {
				case <-startedWait.C:
				default:
				}
			}
			return Result{}, errors.Join(fmt.Errorf("supervisor could not be reaped after deadline"), groupCleanupErr)
		}
		if !reapTimer.Stop() {
			select {
			case <-reapTimer.C:
			default:
			}
		}
		if killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
			return Result{}, fmt.Errorf("kill unresponsive supervisor: %w", killErr)
		}
	}
	close(contextDone)
	_ = controlWrite.Close()
	identity := <-identities
	decodedReports := <-reports

	started, final, protocolErr := validateSupervisorReports(decodedReports, identity)
	if protocolErr != nil {
		return Result{}, errors.Join(protocolErr, cleanupTargetGroups(identity, decodedReports, deadline))
	}
	groupPresent, probeErr := groupExists(started.PGID)
	if probeErr != nil {
		return Result{}, errors.Join(probeErr, cleanupTargetGroups(identity, decodedReports, deadline))
	}
	if groupPresent {
		return Result{}, errors.Join(fmt.Errorf("target process group %d remains after final supervisor report", started.PGID), cleanupTargetGroups(identity, decodedReports, deadline))
	}
	var groupCleanupErr error
	if started != nil && final == nil {
		groupCleanupErr = killGroupBounded(started.PGID, deadline)
	}
	captureResults := make([]captureResult, 0, 3)
	captureRemaining := time.Until(deadline)
	if captureRemaining <= 0 {
		if started != nil {
			groupCleanupErr = errors.Join(groupCleanupErr, killGroupBounded(started.PGID, deadline))
		}
		return Result{}, errors.Join(fmt.Errorf("target output pipes remained open at the process deadline"), groupCleanupErr)
	}
	captureTimer := time.NewTimer(captureRemaining)
	for len(captureResults) < 3 {
		select {
		case captured := <-captures:
			captureResults = append(captureResults, captured)
		case <-captureTimer.C:
			if started != nil {
				groupCleanupErr = errors.Join(groupCleanupErr, killGroupBounded(started.PGID, deadline))
			}
			return Result{}, errors.Join(fmt.Errorf("target output pipes remained open after supervisor exit"), groupCleanupErr)
		}
	}
	if !captureTimer.Stop() {
		select {
		case <-captureTimer.C:
		default:
		}
	}
	for _, captured := range captureResults {
		if captured.err != nil {
			if started != nil {
				groupCleanupErr = errors.Join(groupCleanupErr, killGroupBounded(started.PGID, deadline))
			}
			return Result{}, errors.Join(fmt.Errorf("capture target %s: %w", captured.name, captured.err), groupCleanupErr)
		}
	}
	result = Result{
		Captured:    true,
		Stdout:      stdoutCapture.Result(),
		Stderr:      stderrCapture.Result(),
		WorldRecord: worldCapture.Result().Bytes,
	}
	if ioBacking != nil {
		terminal := <-ioTerminal
		transcript, transcriptErr := ioBacking.result(terminal)
		if transcriptErr != nil {
			return result, transcriptErr
		}
		result.IOTranscript = transcript
	}
	if readOnlyMountBroker != nil {
		if err := mountRequestRead.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			return result, fmt.Errorf("close read-only mount request reader: %w", err)
		}
		if err := mountResponseWrite.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			return result, fmt.Errorf("close read-only mount response writer: %w", err)
		}
		if serveErr := <-mountServed; serveErr != nil && !errors.Is(serveErr, os.ErrClosed) {
			return result, fmt.Errorf("serve read-only mounts: %w", serveErr)
		}
		result.IOROMounts = readOnlyMountBroker.Captured()
	}
	if started != nil {
		result.PID = started.PID
		result.PGID = started.PGID
	}
	if final != nil {
		result.Termination = final.Termination
		result.ExitCode = final.ExitCode
		result.Signal = final.Signal
		result.WatchdogTimeout = final.WatchdogTimeout
		result.Cancelled = final.Cancelled
		result.PID = final.PID
		result.PGID = final.PGID
		result.GroupGone = final.GroupGone
	}
	if worldCapture.Result().Truncated {
		return result, fmt.Errorf("World child record exceeded its configured bound")
	}
	if waitErr != nil {
		if started != nil {
			groupCleanupErr = errors.Join(groupCleanupErr, killGroupBounded(started.PGID, deadline))
		}
		if supervisorTimedOut {
			return result, errors.Join(fmt.Errorf("supervisor exceeded the process deadline"), groupCleanupErr)
		}
		return result, errors.Join(fmt.Errorf("wait for supervisor: %w", waitErr), groupCleanupErr)
	}
	return result, nil
}

func validateSupervisorReports(reports []supervisorReport, identity targetIdentity) (*supervisorReport, *supervisorReport, error) {
	var started, final *supervisorReport
	for index := range reports {
		report := &reports[index]
		if report.Kind == "protocol_error" {
			return started, final, fmt.Errorf("decode supervisor report: %s", report.Error)
		}
		switch {
		case index == 0 && report.Kind == "started":
			started = report
		case index == 1 && report.Kind == "final":
			final = report
		default:
			return started, final, fmt.Errorf("invalid supervisor report %d kind %q", index, report.Kind)
		}
	}
	if started == nil || final == nil || len(reports) != 2 {
		return started, final, fmt.Errorf("supervisor protocol requires exactly started then final reports")
	}
	if identity.err != nil {
		return started, final, fmt.Errorf("read trusted target identity: %w", identity.err)
	}
	if started.PID <= 0 || started.PGID <= 0 || started.PID != identity.pid || started.PGID != identity.pgid {
		return started, final, fmt.Errorf("supervisor start report does not match trusted target identity")
	}
	if started.Termination != "" || started.ExitCode != 0 || started.Signal != "" || started.WatchdogTimeout || started.Cancelled || started.GroupGone || started.Error != "" {
		return started, final, fmt.Errorf("supervisor start report contains terminal state")
	}
	if final.PID != started.PID || final.PGID != started.PGID {
		return started, final, fmt.Errorf("supervisor final report identity changed")
	}
	if !final.GroupGone || final.Error != "" || final.WatchdogTimeout && final.Cancelled {
		return started, final, fmt.Errorf("supervisor final report has invalid containment state")
	}
	switch final.Termination {
	case TerminationExit:
		if final.ExitCode < 0 || final.Signal != "" {
			return started, final, fmt.Errorf("supervisor exit report has invalid status")
		}
	case TerminationSignal:
		if final.Signal == "" || final.ExitCode != 0 {
			return started, final, fmt.Errorf("supervisor signal report has invalid status")
		}
	default:
		return started, final, fmt.Errorf("supervisor final report has invalid termination %q", final.Termination)
	}
	return started, final, nil
}

func supervisorTimeout(timeout time.Duration) time.Duration {
	reserve := min(100*time.Millisecond, timeout/4)
	return timeout - reserve
}

func targetTimeout(timeout time.Duration) time.Duration {
	supervision := supervisorTimeout(timeout)
	reportReserve := min(50*time.Millisecond, supervision/4)
	return supervision - reportReserve
}

type captureWriter struct {
	capture   *OutputCapture
	head      io.Writer
	remaining uint64
	cancel    func()
	err       error
}

func newCaptureWriter(capture *OutputCapture, head io.Writer, cancel func()) *captureWriter {
	return &captureWriter{capture: capture, head: head, remaining: uint64(capture.headLimit), cancel: cancel}
}

func (writer *captureWriter) Write(data []byte) (int, error) {
	written, err := writer.capture.Write(data)
	if err != nil {
		return written, err
	}
	if writer.head != nil && writer.remaining != 0 && writer.err == nil {
		length := min(uint64(len(data)), writer.remaining)
		headWritten, headErr := writer.head.Write(data[:length])
		writer.remaining -= uint64(headWritten)
		if headErr == nil && uint64(headWritten) != length {
			headErr = io.ErrShortWrite
		}
		if headErr != nil {
			writer.err = headErr
			writer.cancel()
		}
	}
	return written, nil
}

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
	bootstrapRead, bootstrapWrite, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("create target bootstrap request pipe: %w", err)
	}
	activationRead, activationWrite, err := os.Pipe()
	if err != nil {
		return errors.Join(fmt.Errorf("create target activation pipe: %w", err), bootstrapRead.Close(), bootstrapWrite.Close())
	}
	readinessRead, readinessWrite, err := os.Pipe()
	if err != nil {
		return errors.Join(fmt.Errorf("create target readiness pipe: %w", err), bootstrapRead.Close(), bootstrapWrite.Close(), activationRead.Close(), activationWrite.Close())
	}
	configRead, configWrite, err := os.Pipe()
	if err != nil {
		return errors.Join(fmt.Errorf("create target World configuration pipe: %w", err), bootstrapRead.Close(), bootstrapWrite.Close(), activationRead.Close(), activationWrite.Close(), readinessRead.Close(), readinessWrite.Close())
	}
	encodedConfig, err := worldpipe.Encode(worldpipe.Config{
		TransitionLimit: request.WorldTransitionLimit,
		Seed:            request.WorldSeed,
		ExpectedInitial: request.ExpectedWorldInitial,
	})
	if err != nil {
		return errors.Join(err, bootstrapRead.Close(), bootstrapWrite.Close(), activationRead.Close(), activationWrite.Close(), readinessRead.Close(), readinessWrite.Close(), configRead.Close(), configWrite.Close())
	}
	target.ExtraFiles = []*os.File{bootstrapRead, activationRead, readinessWrite, configRead, worldRecord, identity}
	if ioTranscript != nil {
		target.ExtraFiles = append(target.ExtraFiles, ioTranscript, ioTerminal, ioExpected)
	}
	if ioROMountRequest != nil {
		target.ExtraFiles = append(target.ExtraFiles, ioROMountRequest, ioROMountResponse)
	}
	target.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := target.Start(); err != nil {
		return errors.Join(fmt.Errorf("start target bootstrap: %w", err), bootstrapRead.Close(), bootstrapWrite.Close(), activationRead.Close(), activationWrite.Close(), readinessRead.Close(), readinessWrite.Close(), configRead.Close(), configWrite.Close())
	}
	targetPGID := target.Process.Pid
	pid := target.Process.Pid
	if err := closeOpenFile(&identity); err != nil {
		return errors.Join(fmt.Errorf("close inherited target identity pipe: %w", err), bootstrapRead.Close(), bootstrapWrite.Close(), activationRead.Close(), activationWrite.Close(), readinessRead.Close(), readinessWrite.Close(), configRead.Close(), configWrite.Close(), killReapTarget(target, targetPGID, deadline))
	}
	encoder := json.NewEncoder(report)
	if err := encoder.Encode(supervisorReport{Kind: "started", PID: pid, PGID: targetPGID}); err != nil {
		return errors.Join(fmt.Errorf("report target start: %w", err), bootstrapRead.Close(), bootstrapWrite.Close(), activationRead.Close(), activationWrite.Close(), readinessRead.Close(), readinessWrite.Close(), configRead.Close(), configWrite.Close(), killReapTarget(target, targetPGID, deadline))
	}
	if closeErr := errors.Join(bootstrapRead.Close(), activationRead.Close(), readinessWrite.Close(), configRead.Close(), closeOpenFile(&worldRecord), closeOpenFile(&ioTranscript), closeOpenFile(&ioTerminal), closeOpenFile(&ioExpected), closeOpenFile(&ioROMountRequest), closeOpenFile(&ioROMountResponse)); closeErr != nil {
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
