package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomadv3/internal/hostexec"
	"go.temporal.io/server/tools/gomadv3/record"
	"go.temporal.io/server/tools/gomadv3/target"
)

const maximumCoordinatorMessageBytes = 16 << 20

type coordinatorConfig struct {
	ResumeCampaign           string
	PlanSHA256               record.SHA256
	Shard                    CampaignShard
	Strategy                 Strategy
	Seeds                    string
	Parallel                 int
	ExecutionTimeout         time.Duration
	OverallTimeout           time.Duration
	TerminateGrace           time.Duration
	OnFailure                FailurePolicy
	FailureBudget            uint64
	OutputLimit              uint64
	WorldTransitionLimit     uint64
	ChoiceTraceLimit         uint64
	MaxExecutions            uint64
	MaxChoiceDepth           uint64
	MaxExplorationBytes      uint64
	Artifacts                string
	Environment              []string
	IOROMounts               []string
	IOROMountLimits          readonlymount.Limits
	Target                   target.Spec
	SupervisorCommand        []string
	RunnerBuild              string
	Coverage                 CoverageMode
	RequiredSemanticProbes   []string
	CollectExecutionEvidence bool
	KeepSuccesses            KeepSuccesses
	SuccessArtifactLimit     uint64
	SuccessBytesLimit        uint64
	Guide                    bool
	Corpus                   string
	GuideSnapshotSHA256      record.SHA256
	ProgressInterval         time.Duration
}

type coordinatorResponse struct {
	CampaignResult        CampaignResult
	ErrorReason           string
	ErrorDetail           string
	UnsupportedTarget     *target.UnsupportedCapabilityError
	MissingSemanticProbes []string
}

type coordinatorMessage struct {
	Type     string               `json:"type"`
	Progress *CampaignEvent       `json:"progress,omitempty"`
	Response *coordinatorResponse `json:"response,omitempty"`
}

type coordinatorDecodeResult struct {
	response coordinatorResponse
	err      error
}

type coordinatorProgressError struct {
	err error
}

func (err *coordinatorProgressError) Error() string {
	return err.err.Error()
}

func (err *coordinatorProgressError) Unwrap() error {
	return err.err
}

func runIsolated(ctx context.Context, config CampaignSpec) (CampaignResult, error) {
	if _, _, err := validateConfig(config); err != nil {
		return CampaignResult{}, err
	}
	if config.CoordinatorCommand[0] == "" {
		return CampaignResult{}, fmt.Errorf("coordinator command is required")
	}
	overallCtx, cancel := context.WithTimeout(ctx, config.OverallTimeout)
	defer cancel()
	deadline, _ := overallCtx.Deadline()
	reserve := min(250*time.Millisecond, max(time.Until(deadline)/5, time.Nanosecond))
	childTimeout := max(time.Until(deadline)-2*reserve, time.Nanosecond)
	wire := coordinatorConfig{
		ResumeCampaign: config.ResumeCampaign, PlanSHA256: config.PlanSHA256, Shard: config.Shard,
		Strategy: config.Strategy, Seeds: config.Seeds, Parallel: config.Parallel, ExecutionTimeout: config.ExecutionTimeout, OverallTimeout: childTimeout,
		TerminateGrace: config.TerminateGrace, OnFailure: config.OnFailure, FailureBudget: config.FailureBudget,
		OutputLimit: config.OutputLimit, WorldTransitionLimit: config.WorldTransitionLimit, ChoiceTraceLimit: config.ChoiceTraceLimit,
		MaxExecutions: config.MaxExecutions, MaxChoiceDepth: config.MaxChoiceDepth, MaxExplorationBytes: config.MaxExplorationBytes, Artifacts: config.Artifacts,
		Environment: append([]string(nil), config.Environment...), Target: config.Target,
		IOROMounts: append([]string(nil), config.IOROMounts...), IOROMountLimits: config.IOROMountLimits,
		SupervisorCommand: append([]string(nil), config.SupervisorCommand...), RunnerBuild: config.RunnerBuild,
		Coverage: config.Coverage, RequiredSemanticProbes: append([]string(nil), config.RequiredSemanticProbes...),
		CollectExecutionEvidence: config.CollectExecutionEvidence,
		KeepSuccesses:            config.KeepSuccesses, SuccessArtifactLimit: config.SuccessArtifactLimit, SuccessBytesLimit: config.SuccessBytesLimit,
		Guide: config.Guide, Corpus: config.Corpus, GuideSnapshotSHA256: config.GuideSnapshotSHA256,
		ProgressInterval: config.ProgressInterval,
	}
	request, err := json.Marshal(wire)
	if err != nil {
		return CampaignResult{}, &HostError{Reason: "coordinator_encode", Err: err}
	}
	command := exec.Command(config.CoordinatorCommand[0], config.CoordinatorCommand[1:]...)
	command.Env = append(os.Environ(), "GOMADV3_RUNNER_COORDINATOR=1")
	command.Stdin = bytes.NewReader(request)
	stderr, err := hostexec.New(4096)
	if err != nil {
		return CampaignResult{}, &HostError{Reason: "coordinator_capture", Err: err}
	}
	stdout, stdoutWriter, err := os.Pipe()
	if err != nil {
		return CampaignResult{}, &HostError{Reason: "coordinator_pipe", Err: err}
	}
	command.Stdout = stdoutWriter
	command.Stderr = stderr
	configureCoordinatorCommand(command)
	if err := command.Start(); err != nil {
		return CampaignResult{}, &HostError{Reason: "coordinator_start", Err: errors.Join(err, stdout.Close(), stdoutWriter.Close())}
	}
	waited := make(chan error, 1)
	go func() { waited <- command.Wait() }()
	if err := stdoutWriter.Close(); err != nil {
		return CampaignResult{}, &HostError{Reason: "coordinator_pipe", Err: errors.Join(err, terminateCoordinator(command, waited, deadline))}
	}
	decoded := make(chan coordinatorDecodeResult, 1)
	go func() {
		response, decodeErr := decodeCoordinatorMessages(stdout, config.Progress)
		decodeErr = errors.Join(decodeErr, stdout.Close())
		decoded <- coordinatorDecodeResult{response: response, err: decodeErr}
	}()
	timer := time.NewTimer(max(time.Until(deadline)-reserve, 0))
	defer timer.Stop()
	var waitErr error
	var decodeResult coordinatorDecodeResult
	waitComplete := false
	decodeComplete := false
	for !waitComplete || !decodeComplete {
		select {
		case waitErr = <-waited:
			waitComplete = true
			waited = nil
		case decodeResult = <-decoded:
			decodeComplete = true
			decoded = nil
			if decodeResult.err != nil && !waitComplete {
				cleanupErr := terminateCoordinator(command, waited, deadline)
				var progressErr *coordinatorProgressError
				reason := "coordinator_decode"
				if errors.As(decodeResult.err, &progressErr) {
					reason = "progress_output"
				}
				return CampaignResult{}, &HostError{Reason: reason, Err: errors.Join(decodeResult.err, cleanupErr)}
			}
		case <-overallCtx.Done():
			if waitComplete {
				waited = completedCoordinatorWait(waitErr)
			}
			return killCoordinator(command, waited, deadline, overallCtx.Err())
		case <-timer.C:
			if waitComplete {
				waited = completedCoordinatorWait(waitErr)
			}
			return killCoordinator(command, waited, deadline, context.DeadlineExceeded)
		}
	}
	stderrResult := stderr.Result()
	if waitErr != nil {
		return CampaignResult{}, &HostError{Reason: "coordinator_exit", Err: errors.Join(waitErr, errorFromOutput(stderrResult.Bytes))}
	}
	if decodeResult.err != nil {
		var progressErr *coordinatorProgressError
		reason := "coordinator_decode"
		if errors.As(decodeResult.err, &progressErr) {
			reason = "progress_output"
		}
		return CampaignResult{}, &HostError{Reason: reason, Err: errors.Join(decodeResult.err, errorFromOutput(stderrResult.Bytes))}
	}
	response := decodeResult.response
	if response.ErrorReason != "" {
		if response.UnsupportedTarget != nil {
			return response.CampaignResult, &HostError{Reason: response.ErrorReason, Err: response.UnsupportedTarget}
		}
		if len(response.MissingSemanticProbes) != 0 {
			return response.CampaignResult, &deterministicio.MissingSemanticProbesError{Probes: append([]string(nil), response.MissingSemanticProbes...)}
		}
		return response.CampaignResult, &HostError{Reason: response.ErrorReason, Err: errors.New(response.ErrorDetail)}
	}
	return response.CampaignResult, nil
}

func completedCoordinatorWait(waitErr error) chan error {
	waited := make(chan error, 1)
	waited <- waitErr
	return waited
}

func decodeCoordinatorMessages(input io.Reader, progress CampaignEventFunc) (coordinatorResponse, error) {
	limited := &io.LimitedReader{R: input, N: maximumCoordinatorMessageBytes + 1}
	decoder := json.NewDecoder(limited)
	decoder.DisallowUnknownFields()
	var response coordinatorResponse
	foundResult := false
	for {
		var message coordinatorMessage
		err := decoder.Decode(&message)
		if err == io.EOF {
			break
		}
		if err != nil {
			return coordinatorResponse{}, drainCoordinatorInput(input, fmt.Errorf("decode coordinator message: %w", err))
		}
		if foundResult {
			return coordinatorResponse{}, drainCoordinatorInput(input, fmt.Errorf("coordinator emitted a message after its result"))
		}
		switch message.Type {
		case "progress":
			if message.Progress == nil || message.Response != nil {
				return coordinatorResponse{}, fmt.Errorf("coordinator progress message is malformed")
			}
			if progress != nil {
				if err := progress(*message.Progress); err != nil {
					return coordinatorResponse{}, drainCoordinatorInput(input, &coordinatorProgressError{err: err})
				}
			}
		case "result":
			if message.Response == nil || message.Progress != nil {
				return coordinatorResponse{}, fmt.Errorf("coordinator result message is malformed")
			}
			response = *message.Response
			foundResult = true
		default:
			return coordinatorResponse{}, fmt.Errorf("unknown coordinator message type %q", message.Type)
		}
	}
	if limited.N == 0 {
		return coordinatorResponse{}, fmt.Errorf("coordinator response exceeds its bound")
	}
	if !foundResult {
		return coordinatorResponse{}, fmt.Errorf("coordinator omitted its result")
	}
	return response, nil
}

func drainCoordinatorInput(input io.Reader, cause error) error {
	_, err := io.Copy(io.Discard, input)
	return errors.Join(cause, err)
}

func killCoordinator(command *exec.Cmd, waited <-chan error, deadline time.Time, cause error) (CampaignResult, error) {
	cleanupErr := terminateCoordinator(command, waited, deadline)
	return CampaignResult{}, &HostError{Reason: contextFailureReason(cause), Err: errors.Join(cause, cleanupErr)}
}

func errorFromOutput(output []byte) error {
	if len(output) == 0 {
		return nil
	}
	if len(output) > 4096 {
		output = output[:4096]
	}
	return errors.New(string(output))
}

func CoordinatorMain(input io.Reader, output io.Writer) error {
	decoder := json.NewDecoder(io.LimitReader(input, maximumCoordinatorMessageBytes+1))
	decoder.DisallowUnknownFields()
	var wire coordinatorConfig
	if err := decoder.Decode(&wire); err != nil {
		return fmt.Errorf("decode coordinator request: %w", err)
	}
	if token, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("trailing coordinator request %v: %w", token, err)
	}
	config := CampaignSpec{
		ResumeCampaign: wire.ResumeCampaign, PlanSHA256: wire.PlanSHA256, Shard: wire.Shard,
		Strategy: wire.Strategy, Seeds: wire.Seeds, Parallel: wire.Parallel, ExecutionTimeout: wire.ExecutionTimeout, OverallTimeout: wire.OverallTimeout,
		TerminateGrace: wire.TerminateGrace, OnFailure: wire.OnFailure, FailureBudget: wire.FailureBudget,
		OutputLimit: wire.OutputLimit, WorldTransitionLimit: wire.WorldTransitionLimit, ChoiceTraceLimit: wire.ChoiceTraceLimit,
		MaxExecutions: wire.MaxExecutions, MaxChoiceDepth: wire.MaxChoiceDepth, MaxExplorationBytes: wire.MaxExplorationBytes, Artifacts: wire.Artifacts,
		Environment: wire.Environment, Target: wire.Target, SupervisorCommand: wire.SupervisorCommand, RunnerBuild: wire.RunnerBuild,
		IOROMounts: wire.IOROMounts, IOROMountLimits: wire.IOROMountLimits,
		ProgressInterval: wire.ProgressInterval,
		Coverage:         wire.Coverage, RequiredSemanticProbes: wire.RequiredSemanticProbes,
		CollectExecutionEvidence: wire.CollectExecutionEvidence,
		KeepSuccesses:            wire.KeepSuccesses, SuccessArtifactLimit: wire.SuccessArtifactLimit, SuccessBytesLimit: wire.SuccessBytesLimit,
		Guide: wire.Guide, Corpus: wire.Corpus, GuideSnapshotSHA256: wire.GuideSnapshotSHA256,
	}
	encoder := json.NewEncoder(output)
	config.Progress = func(progress CampaignEvent) error {
		return encoder.Encode(coordinatorMessage{Type: "progress", Progress: &progress})
	}
	summary, runErr := runLocal(context.Background(), config)
	response := coordinatorResponse{CampaignResult: summary}
	if runErr != nil {
		var hostError *HostError
		if errors.As(runErr, &hostError) {
			response.ErrorReason = hostError.Reason
		} else {
			response.ErrorReason = "coordinator_run"
		}
		response.ErrorDetail = runErr.Error()
		var unsupported *target.UnsupportedCapabilityError
		if errors.As(runErr, &unsupported) {
			copied := *unsupported
			response.UnsupportedTarget = &copied
		}
		var missing *deterministicio.MissingSemanticProbesError
		if errors.As(runErr, &missing) {
			response.MissingSemanticProbes = append([]string(nil), missing.Probes...)
		}
	}
	return encoder.Encode(coordinatorMessage{Type: "result", Response: &response})
}
