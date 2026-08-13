//go:build unix

package process

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/romount"
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
	worldCapability, ioCapability := request.World, request.IO
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
	worldCapture, err := NewOutputCapture(worldCapability.RecordLimit)
	if err != nil {
		return Result{}, fmt.Errorf("create World record capture: %w", err)
	}
	var ioBacking *ioTranscriptBacking
	if ioCapability != nil && ioCapability.Transcript != nil {
		transcript := ioCapability.Transcript
		ioBacking, err = newIOTranscriptBacking(transcript.Limit, transcript.Replay, transcript.Expected)
		if err != nil {
			return Result{}, err
		}
		defer func() {
			retErr = errors.Join(retErr, ioBacking.close())
		}()
	}
	var readOnlyMountBroker *romount.Broker
	if ioCapability != nil && ioCapability.ReadOnlyMount != nil {
		mounts := ioCapability.ReadOnlyMount
		if mounts.Replay == nil {
			readOnlyMountBroker, err = romount.Prepare(mounts.Mappings, mounts.Limits)
		} else {
			readOnlyMountBroker, err = romount.PrepareReplay(mounts.Mappings, mounts.Limits, *mounts.Replay)
		}
		if err != nil {
			return Result{}, fmt.Errorf("prepare read-only mounts: %w", err)
		}
		defer func() { retErr = errors.Join(retErr, readOnlyMountBroker.Close()) }()
	}
	capabilities := launchCapabilities{ioTranscript: ioBacking != nil, readOnlyMount: readOnlyMountBroker != nil}
	resources := newLaunchResources(capabilities)
	defer func() { retErr = errors.Join(retErr, resources.close()) }()
	var mountRequestRead, mountResponseWrite *os.File
	if readOnlyMountBroker != nil {
		mountRequestRead, err = resources.createPipe(ioROMountRequestResource, inheritWrite, "read-only mount request")
		if err != nil {
			return Result{}, err
		}
		mountResponseWrite, err = resources.createPipe(ioROMountResponseResource, inheritRead, "read-only mount response")
		if err != nil {
			return Result{}, err
		}
	}
	controlWrite, err := resources.createPipe(controlResource, inheritRead, "supervisor control")
	if err != nil {
		return Result{}, err
	}
	reportRead, err := resources.createPipe(reportResource, inheritWrite, "supervisor report")
	if err != nil {
		return Result{}, err
	}
	stdoutRead, err := resources.createPipe(stdoutResource, inheritWrite, "target stdout")
	if err != nil {
		return Result{}, err
	}
	stderrRead, err := resources.createPipe(stderrResource, inheritWrite, "target stderr")
	if err != nil {
		return Result{}, err
	}
	requestWrite, err := resources.createPipe(supervisorRequestResource, inheritRead, "supervisor request")
	if err != nil {
		return Result{}, err
	}
	worldRead, err := resources.createPipe(worldRecordResource, inheritWrite, "World record")
	if err != nil {
		return Result{}, err
	}
	identityRead, err := resources.createPipe(identityResource, inheritWrite, "target identity")
	if err != nil {
		return Result{}, err
	}
	if ioBacking != nil {
		resources.bind(ioTranscriptResource, &ioBacking.file)
		resources.bind(ioTerminalResource, &ioBacking.terminalWrite)
		resources.bind(ioExpectedResource, &ioBacking.expected)
	}

	command := exec.Command(request.SupervisorCommand[0], request.SupervisorCommand[1:]...)
	command.Env = append(os.Environ(), "GOMADV3_PROCESS_SUPERVISOR=1")
	command.ExtraFiles, err = resources.extraFiles(supervisorStage)
	if err != nil {
		return Result{}, err
	}
	if err := command.Start(); err != nil {
		return Result{}, fmt.Errorf("start supervisor: %w", err)
	}
	if closeErr := resources.closeInherited(supervisorStage); closeErr != nil {
		return Result{}, errors.Join(fmt.Errorf("close inherited supervisor pipe ends: %w", closeErr), cleanupEarlySupervisor(command, controlWrite, reportRead, nil, deadline))
	}
	var ioTerminal <-chan []byte
	if ioBacking != nil {
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
		WorldTransitionLimit: worldCapability.TransitionLimit,
		WorldSeed:            worldCapability.Seed,
		ExpectedWorldInitial: append([]byte(nil), worldCapability.ExpectedInitial...),
		IOROMounts:           readOnlyMountBroker != nil,
	}
	if ioCapability != nil {
		wireRequest.IOConfig = append([]byte(nil), ioCapability.Config...)
		if ioCapability.Transcript != nil {
			wireRequest.IOTranscriptLimit = ioCapability.Transcript.Limit
			wireRequest.IOReplay = ioCapability.Transcript.Replay
		}
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
	return &captureWriter{capture: capture, head: head, remaining: capture.HeadLimit(), cancel: cancel}
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
