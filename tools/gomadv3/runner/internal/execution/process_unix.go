//go:build unix

package execution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"go.temporal.io/server/tools/gomadv3/choice"
	romount "go.temporal.io/server/tools/gomadv3/deterministicio"
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
	WorldReplayPlan      []byte        `json:"world_replay_plan"`
	IOConfig             []byte        `json:"io_config"`
	IOTranscriptLimit    uint64        `json:"io_transcript_limit"`
	IOReplay             bool          `json:"io_replay"`
	IOROMounts           bool          `json:"io_ro_mounts"`
	ChoiceTrace          bool          `json:"choice_trace"`
	ChoiceTraceLimit     uint64        `json:"choice_trace_limit"`
	ChoiceMode           choice.Mode   `json:"choice_mode"`
	ChoiceTapeBytes      uint64        `json:"choice_tape_bytes"`
	Simulation           bool          `json:"simulation"`
	SimulationBootstrap  bool          `json:"simulation_bootstrap"`
}

type targetBootstrapRequest struct {
	Command             string      `json:"command"`
	Args                []string    `json:"args"`
	Argv0               string      `json:"argv0"`
	Dir                 string      `json:"dir"`
	Env                 []string    `json:"env"`
	IOConfig            []byte      `json:"io_config"`
	IOTranscriptLimit   uint64      `json:"io_transcript_limit"`
	IOReplay            bool        `json:"io_replay"`
	IOROMounts          bool        `json:"io_ro_mounts"`
	ChoiceTrace         bool        `json:"choice_trace"`
	ChoiceTraceLimit    uint64      `json:"choice_trace_limit"`
	ChoiceMode          choice.Mode `json:"choice_mode"`
	ChoiceTapeBytes     uint64      `json:"choice_tape_bytes"`
	Simulation          bool        `json:"simulation"`
	SimulationBootstrap bool        `json:"simulation_bootstrap"`
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

func Run(ctx context.Context, request Spec) (result Result, retErr error) {
	if err := validateSpec(request); err != nil {
		return Result{}, err
	}
	var simulationCoordinator *simulationCoordinator
	if request.Simulation != nil && request.Simulation.Role == SimulationRoleCoordinator && request.Simulation.handler == nil {
		var coordinatorErr error
		simulationCoordinator, coordinatorErr = newSimulationCoordinator(request)
		if coordinatorErr != nil {
			return Result{}, coordinatorErr
		}
		request.Simulation.handler = simulationCoordinator.handle
		request.Simulation.time = simulationCoordinator.handleCoordinatorTime
		request.Simulation.accepting = simulationCoordinator.handleWaitAcceptance
		request.Simulation.delivering = simulationCoordinator.handleCoordinatorDelivery
		request.Simulation.responded = simulationCoordinator.handleCoordinatorResponse
		request.Simulation.arrived = func(arrivals uint32) error {
			return simulationCoordinator.time.acknowledgeExternal(simulationCoordinator.coordinator, arrivals)
		}
		defer func() { retErr = errors.Join(retErr, simulationCoordinator.close()) }()
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
	var ioSession *romount.Session
	var ioTranscriptFile, ioTerminalFile, ioExpectedFile *os.File
	if ioCapability != nil && ioCapability.Transcript != nil {
		transcript := ioCapability.Transcript
		ioSession, err = romount.NewSession(romount.SessionSpec{Limit: transcript.Limit, Replay: transcript.Replay, Expected: transcript.Expected})
		if err != nil {
			return Result{}, err
		}
		defer func() {
			retErr = errors.Join(retErr, ioSession.Close())
		}()
		files := ioSession.Files()
		ioTranscriptFile, ioTerminalFile, ioExpectedFile = files.Transcript, files.Terminal, files.Expected
	}
	var choiceSession *choice.Session
	var choiceTraceFile, choiceTerminalFile, choiceReplayPlanFile *os.File
	if request.Choice != nil {
		choiceSession, err = choice.NewSession(choice.SessionSpec{Limit: request.Choice.Limit, Mode: request.Choice.Mode, ReplayPlan: request.Choice.ReplayPlan})
		if err != nil {
			return Result{}, projectChoiceSessionError(err)
		}
		defer func() { retErr = errors.Join(retErr, choiceSession.Close()) }()
		files := choiceSession.Files()
		choiceTraceFile, choiceTerminalFile, choiceReplayPlanFile = files.Trace, files.Terminal, files.ReplayPlan
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
	capabilities := launchCapabilities{ioTranscript: ioSession != nil, readOnlyMount: readOnlyMountBroker != nil, choiceTrace: choiceSession != nil, choiceReplayPlan: choiceReplayPlanFile != nil, simulation: request.Simulation != nil, simulationBootstrap: request.Simulation != nil && request.Simulation.Role == SimulationRoleNode, simulationCoordinator: request.Simulation != nil && request.Simulation.Role == SimulationRoleCoordinator}
	resources := newLaunchResources(capabilities)
	defer func() { retErr = errors.Join(retErr, resources.close()) }()
	var mountRequestRead, mountResponseWrite *os.File
	var simulationRequestRead, simulationResponseWrite, simulationBootstrapWrite, simulationControlWrite, simulationModelRequestHost, simulationModelResponseHost, simulationTimeRequestRead, simulationTimeResponseWrite *os.File
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
	if request.Simulation != nil {
		simulationRequestRead, err = resources.createPipe(simulationRequestResource, inheritWrite, "simulation request")
		if err != nil {
			return Result{}, err
		}
		simulationResponseWrite, err = resources.createPipe(simulationResponseResource, inheritRead, "simulation response")
		if err != nil {
			return Result{}, err
		}
		if request.Simulation.Role == SimulationRoleNode {
			simulationBootstrapWrite, err = resources.createPipe(simulationBootstrapResource, inheritRead, "simulation node bootstrap")
			if err != nil {
				return Result{}, err
			}
			simulationControlWrite, err = resources.createPipe(simulationControlResource, inheritRead, "simulation node control")
			if err != nil {
				return Result{}, err
			}
			simulationModelRequestHost, err = resources.createPipe(simulationModelRequestResource, inheritWrite, "simulation node model request")
			if err != nil {
				return Result{}, err
			}
			simulationModelResponseHost, err = resources.createPipe(simulationModelResponseResource, inheritRead, "simulation node model response")
			if err != nil {
				return Result{}, err
			}
		} else {
			simulationModelRequestHost, err = resources.createPipe(simulationModelRequestResource, inheritRead, "simulation model request")
			if err != nil {
				return Result{}, err
			}
			simulationModelResponseHost, err = resources.createPipe(simulationModelResponseResource, inheritWrite, "simulation model response")
			if err != nil {
				return Result{}, err
			}
			if simulationCoordinator == nil {
				return Result{}, errors.New("simulation coordinator transport is unavailable")
			}
			simulationCoordinator.model = newSimulationModelTransport(simulationModelRequestHost, simulationModelResponseHost, func() {
				simulationCoordinator.time.deliverExternal(simulationCoordinator.coordinator)
			}, func(frame simulationFrame) error {
				return simulationCoordinator.handleModelArrival(frame)
			}, func(frame simulationFrame) error {
				return simulationCoordinator.time.acknowledgeExternal(simulationCoordinator.coordinator, frame.Arrivals)
			})
		}
		simulationTimeRequestRead, err = resources.createPipe(simulationTimeRequestResource, inheritWrite, "simulation time request")
		if err != nil {
			return Result{}, err
		}
		simulationTimeResponseWrite, err = resources.createPipe(simulationTimeResponseResource, inheritRead, "simulation time response")
		if err != nil {
			return Result{}, err
		}
		if request.Simulation.time == nil {
			return Result{}, errors.New("simulation time arbitration is unavailable")
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
	if ioSession != nil {
		resources.bind(ioTranscriptResource, &ioTranscriptFile)
		resources.bind(ioTerminalResource, &ioTerminalFile)
		resources.bind(ioExpectedResource, &ioExpectedFile)
	}
	if choiceSession != nil {
		resources.bind(choiceTraceResource, &choiceTraceFile)
		resources.bind(choiceTerminalResource, &choiceTerminalFile)
		if choiceReplayPlanFile != nil {
			resources.bind(choiceTapeResource, &choiceReplayPlanFile)
		}
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
	type ioCollection struct {
		transcript romount.Transcript
		err        error
	}
	var collectedIO <-chan ioCollection
	if ioSession != nil {
		collected := make(chan ioCollection, 1)
		go func() {
			transcript, collectErr := ioSession.Collect()
			collected <- ioCollection{transcript: transcript, err: collectErr}
		}()
		collectedIO = collected
	}
	type choiceCollection struct {
		trace choice.Trace
		err   error
	}
	var collectedChoice <-chan choiceCollection
	if choiceSession != nil {
		collected := make(chan choiceCollection, 1)
		go func() {
			trace, collectErr := choiceSession.Collect()
			collected <- choiceCollection{trace: trace, err: collectErr}
		}()
		collectedChoice = collected
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
		WorldReplayPlan:      append([]byte(nil), worldCapability.ReplayPlan...),
		IOROMounts:           readOnlyMountBroker != nil,
		ChoiceTrace:          choiceSession != nil,
		Simulation:           request.Simulation != nil,
		SimulationBootstrap:  request.Simulation != nil && request.Simulation.Role == SimulationRoleNode,
	}
	if request.Choice != nil {
		wireRequest.ChoiceTraceLimit = request.Choice.Limit
		wireRequest.ChoiceMode = request.Choice.Mode
		if request.Choice.ReplayPlan != nil {
			wireRequest.ChoiceTapeBytes = uint64(len(request.Choice.ReplayPlan.Bytes))
		}
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
	var simulationServed <-chan error
	var simulationModelsServed <-chan error
	var simulationTimeServed <-chan error
	var simulationBootstrapWritten <-chan error
	simulationTimeCtx := ctx
	cancelSimulationTime := func() {}
	if request.Simulation != nil {
		simulationTimeCtx, cancelSimulationTime = context.WithCancel(ctx)
		defer cancelSimulationTime()
		served := make(chan error, 1)
		go func() {
			served <- serveSimulation(ctx, simulationRequestRead, simulationResponseWrite, request.Simulation.handler, request.Simulation.accepting, request.Simulation.delivering, request.Simulation.responded, request.Simulation.arrived)
		}()
		simulationServed = served
		timeServed := make(chan error, 1)
		go func() {
			timeServed <- serveSimulationTime(simulationTimeCtx, simulationTimeRequestRead, simulationTimeResponseWrite, request.Simulation.time)
		}()
		simulationTimeServed = timeServed
		if simulationBootstrapWrite != nil {
			bootstrap := append([]byte(nil), request.Simulation.Bootstrap...)
			written := make(chan error, 1)
			go func() {
				count, writeErr := simulationBootstrapWrite.Write(bootstrap)
				if writeErr == nil && count != len(bootstrap) {
					writeErr = io.ErrShortWrite
				}
				written <- errors.Join(writeErr, simulationBootstrapWrite.Close())
			}()
			simulationBootstrapWritten = written
			modelsServed := make(chan error, 1)
			go func() {
				modelsServed <- serveSimulationModels(ctx, simulationModelRequestHost, simulationModelResponseHost, request.Simulation.handler, request.Simulation.delivering, request.Simulation.responded)
			}()
			simulationModelsServed = modelsServed
		}
	}
	var controlOnce sync.Once
	signalSupervisor := func(mode byte) {
		controlOnce.Do(func() {
			if mode != 0 {
				_, _ = controlWrite.Write([]byte{mode})
			}
			_ = controlWrite.Close()
		})
	}

	type captureResult struct {
		name string
		err  error
	}
	captures := make(chan captureResult, 3)
	go func() {
		writer := newCaptureWriter(stdoutCapture, request.StdoutHead, func() { signalSupervisor(0) })
		_, copyErr := io.Copy(writer, stdoutRead)
		captures <- captureResult{name: "stdout", err: errors.Join(copyErr, writer.err)}
	}()
	go func() {
		writer := newCaptureWriter(stderrCapture, request.StderrHead, func() { signalSupervisor(0) })
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
		if request.Simulation != nil && request.Simulation.hardCrash != nil {
			select {
			case <-request.Simulation.hardCrash:
				signalSupervisor(2)
			case <-ctx.Done():
				if simulationControlWrite != nil {
					_, _ = simulationControlWrite.Write([]byte{1})
					_ = simulationControlWrite.Close()
				}
				timer := time.NewTimer(request.TerminateGrace)
				select {
				case <-timer.C:
					signalSupervisor(1)
				case <-contextDone:
					if !timer.Stop() {
						<-timer.C
					}
				}
			case <-contextDone:
			}
			return
		}
		select {
		case <-ctx.Done():
			signalSupervisor(1)
		case <-contextDone:
		}
	}()

	waits := make(chan error, 1)
	go func() {
		waits <- command.Wait()
	}()
	remaining = time.Until(deadline)
	parentTimeout := max(supervisorTimeout(remaining), 0)
	cleanupReserve := remaining - parentTimeout
	parentTimer := time.NewTimer(parentTimeout)
	defer parentTimer.Stop()
	supervisorTimedOut := false
	var waitErr error
	select {
	case waitErr = <-waits:
	case <-parentTimer.C:
		supervisorTimedOut = true
		signalSupervisor(0)
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
	cancelSimulationTime()
	signalSupervisor(0)
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
	if request.Simulation != nil && request.Simulation.reaped != nil {
		close(request.Simulation.reaped)
	}
	var choiceErr error
	if choiceSession != nil {
		collected := <-collectedChoice
		trace, err := collected.trace, projectChoiceSessionError(collected.err)
		if err == nil || errors.Is(err, ErrChoiceTraceOverflow) || errors.Is(err, ErrChoiceReplayDivergence) {
			result.ChoiceTrace = ChoiceTrace{Profile: choice.Profile, ImplementationSHA256: request.Choice.ImplementationSHA256, Limit: request.Choice.Limit, Trace: trace}
		}
		if errors.Is(err, ErrChoiceReplayDivergence) {
			return result, err
		}
		choiceErr = err
	}
	if ioSession != nil {
		collected := <-collectedIO
		if collected.err != nil {
			return result, collected.err
		}
		result.IOTranscript = collected.transcript
	}
	if choiceErr != nil {
		return result, choiceErr
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
	if request.Simulation != nil {
		if simulationBootstrapWritten != nil {
			if writeErr := <-simulationBootstrapWritten; writeErr != nil {
				return result, fmt.Errorf("write simulation node bootstrap: %w", writeErr)
			}
		}
		if err := simulationRequestRead.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			return result, fmt.Errorf("close simulation request reader: %w", err)
		}
		if err := simulationResponseWrite.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			return result, fmt.Errorf("close simulation response writer: %w", err)
		}
		if serveErr := <-simulationServed; serveErr != nil && !errors.Is(serveErr, os.ErrClosed) {
			return result, fmt.Errorf("serve simulation transport: %w", serveErr)
		}
		if simulationModelsServed != nil {
			if err := simulationModelRequestHost.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
				return result, fmt.Errorf("close simulation model request reader: %w", err)
			}
			if err := simulationModelResponseHost.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
				return result, fmt.Errorf("close simulation model response writer: %w", err)
			}
			if serveErr := <-simulationModelsServed; serveErr != nil && !errors.Is(serveErr, os.ErrClosed) && !errors.Is(serveErr, context.Canceled) {
				return result, fmt.Errorf("serve simulation model transport: %w", serveErr)
			}
		}
		if err := simulationTimeRequestRead.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			return result, fmt.Errorf("close simulation time request reader: %w", err)
		}
		if err := simulationTimeResponseWrite.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			return result, fmt.Errorf("close simulation time response writer: %w", err)
		}
		if serveErr := <-simulationTimeServed; serveErr != nil && !errors.Is(serveErr, os.ErrClosed) && !errors.Is(serveErr, context.Canceled) {
			return result, fmt.Errorf("serve simulation time transport: %w", serveErr)
		}
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
