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

	"go.temporal.io/server/tools/gomadv3/internal/process"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

const maximumCoordinatorMessageBytes = 16 << 20

type coordinatorConfig struct {
	Seeds                string
	Parallel             int
	RunTimeout           time.Duration
	OverallTimeout       time.Duration
	TerminateGrace       time.Duration
	OnFailure            FailurePolicy
	FailureBudget        uint64
	OutputLimit          uint64
	WorldTransitionLimit uint64
	Artifacts            string
	Environment          []string
	IOProfile            string
	Target               target.Spec
	SupervisorCommand    []string
	RunnerBuild          string
}

type coordinatorResponse struct {
	Summary     Summary
	ErrorReason string
	ErrorDetail string
}

func runIsolated(ctx context.Context, config Config) (Summary, error) {
	if _, _, err := validateConfig(config); err != nil {
		return Summary{}, err
	}
	if config.CoordinatorCommand[0] == "" {
		return Summary{}, fmt.Errorf("coordinator command is required")
	}
	overallCtx, cancel := context.WithTimeout(ctx, config.OverallTimeout)
	defer cancel()
	deadline, _ := overallCtx.Deadline()
	reserve := min(250*time.Millisecond, max(time.Until(deadline)/5, time.Nanosecond))
	childTimeout := max(time.Until(deadline)-2*reserve, time.Nanosecond)
	wire := coordinatorConfig{
		Seeds: config.Seeds, Parallel: config.Parallel, RunTimeout: config.RunTimeout, OverallTimeout: childTimeout,
		TerminateGrace: config.TerminateGrace, OnFailure: config.OnFailure, FailureBudget: config.FailureBudget,
		OutputLimit: config.OutputLimit, WorldTransitionLimit: config.WorldTransitionLimit, Artifacts: config.Artifacts,
		Environment: append([]string(nil), config.Environment...), IOProfile: config.IOProfile, Target: config.Target,
		SupervisorCommand: append([]string(nil), config.SupervisorCommand...), RunnerBuild: config.RunnerBuild,
	}
	request, err := json.Marshal(wire)
	if err != nil {
		return Summary{}, &HostError{Reason: "coordinator_encode", Err: err}
	}
	command := exec.Command(config.CoordinatorCommand[0], config.CoordinatorCommand[1:]...)
	command.Env = append(os.Environ(), "GOMADV3_RUNNER_COORDINATOR=1")
	command.Stdin = bytes.NewReader(request)
	stdout, err := process.NewOutputCapture(maximumCoordinatorMessageBytes)
	if err != nil {
		return Summary{}, &HostError{Reason: "coordinator_capture", Err: err}
	}
	stderr, err := process.NewOutputCapture(4096)
	if err != nil {
		return Summary{}, &HostError{Reason: "coordinator_capture", Err: err}
	}
	command.Stdout = stdout
	command.Stderr = stderr
	configureCoordinatorCommand(command)
	if err := command.Start(); err != nil {
		return Summary{}, &HostError{Reason: "coordinator_start", Err: err}
	}
	waited := make(chan error, 1)
	go func() { waited <- command.Wait() }()
	timer := time.NewTimer(max(time.Until(deadline)-reserve, 0))
	defer timer.Stop()
	var waitErr error
	select {
	case waitErr = <-waited:
	case <-overallCtx.Done():
		return killCoordinator(command, waited, deadline, overallCtx.Err())
	case <-timer.C:
		return killCoordinator(command, waited, deadline, context.DeadlineExceeded)
	}
	stdoutResult := stdout.Result()
	stderrResult := stderr.Result()
	if waitErr != nil {
		return Summary{}, &HostError{Reason: "coordinator_exit", Err: errors.Join(waitErr, errorFromOutput(stderrResult.Bytes))}
	}
	if stdoutResult.TotalBytes > maximumCoordinatorMessageBytes {
		return Summary{}, &HostError{Reason: "coordinator_decode", Err: fmt.Errorf("coordinator response exceeds its bound")}
	}
	decoder := json.NewDecoder(bytes.NewReader(stdoutResult.Bytes))
	decoder.DisallowUnknownFields()
	var response coordinatorResponse
	if err := decoder.Decode(&response); err != nil {
		return Summary{}, &HostError{Reason: "coordinator_decode", Err: errors.Join(err, errorFromOutput(stderrResult.Bytes))}
	}
	if token, err := decoder.Token(); err != io.EOF {
		return Summary{}, &HostError{Reason: "coordinator_decode", Err: fmt.Errorf("trailing coordinator response %v: %w", token, err)}
	}
	if response.ErrorReason != "" {
		return response.Summary, &HostError{Reason: response.ErrorReason, Err: errors.New(response.ErrorDetail)}
	}
	return response.Summary, nil
}

func killCoordinator(command *exec.Cmd, waited <-chan error, deadline time.Time, cause error) (Summary, error) {
	cleanupErr := terminateCoordinator(command, waited, deadline)
	return Summary{}, &HostError{Reason: "overall_timeout", Err: errors.Join(cause, cleanupErr)}
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
	config := Config{
		Seeds: wire.Seeds, Parallel: wire.Parallel, RunTimeout: wire.RunTimeout, OverallTimeout: wire.OverallTimeout,
		TerminateGrace: wire.TerminateGrace, OnFailure: wire.OnFailure, FailureBudget: wire.FailureBudget,
		OutputLimit: wire.OutputLimit, WorldTransitionLimit: wire.WorldTransitionLimit, Artifacts: wire.Artifacts,
		Environment: wire.Environment, IOProfile: wire.IOProfile, Target: wire.Target, SupervisorCommand: wire.SupervisorCommand, RunnerBuild: wire.RunnerBuild,
	}
	summary, runErr := runLocal(context.Background(), config)
	response := coordinatorResponse{Summary: summary}
	if runErr != nil {
		var hostError *HostError
		if errors.As(runErr, &hostError) {
			response.ErrorReason = hostError.Reason
		} else {
			response.ErrorReason = "coordinator_run"
		}
		response.ErrorDetail = runErr.Error()
	}
	return json.NewEncoder(output).Encode(response)
}
