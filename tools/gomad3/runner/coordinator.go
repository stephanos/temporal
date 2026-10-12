package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

const maximumCoordinatorMessageBytes = 16 << 20

type coordinatorConfig struct {
	campaignOptions
}

func (config campaignRequest) coordinatorConfig(childTimeout time.Duration) coordinatorConfig {
	options := config.campaignOptions
	options.OverallTimeout = childTimeout
	return coordinatorConfig{campaignOptions: options}
}

func (wire coordinatorConfig) campaignRequest(provider backends.Provider, runnerBuild string) campaignRequest {
	request := campaignRequest{
		campaignOptions: wire.campaignOptions,
		campaignRuntime: campaignRuntime{Backend: provider, RunnerBuild: runnerBuild},
	}
	request.normalize()
	return request
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

type coordinatorExecutionResult struct {
	result backends.ExecutionResult
	err    error
}

func runIsolated(ctx context.Context, config campaignRequest) (CampaignResult, error) {
	if _, _, err := validateCampaignRequest(config); err != nil {
		return CampaignResult{}, err
	}
	if config.Backend == nil {
		return CampaignResult{}, errors.New("coordinator requires a backend provider")
	}
	overallCtx, cancel := context.WithTimeout(ctx, config.OverallTimeout)
	defer cancel()
	deadline, _ := overallCtx.Deadline()
	reserve := min(250*time.Millisecond, max(time.Until(deadline)/5, time.Nanosecond))
	childTimeout := max(time.Until(deadline)-2*reserve, time.Nanosecond)
	request, err := json.Marshal(config.coordinatorConfig(childTimeout))
	if err != nil {
		return CampaignResult{}, &HostError{Reason: "coordinator_encode", Err: err}
	}
	stdout, stdoutWriter, err := os.Pipe()
	if err != nil {
		return CampaignResult{}, &HostError{Reason: "coordinator_pipe", Err: err}
	}
	executionCtx, cancelExecution := context.WithCancel(overallCtx)
	defer cancelExecution()
	executed := make(chan coordinatorExecutionResult, 1)
	go func() {
		result, executeErr := config.Backend.Execute(executionCtx, backends.ExecutionRequest{
			Coordinator: &backends.CoordinatorRequest{Input: request, Stdout: stdoutWriter, Deadline: deadline},
		})
		executed <- coordinatorExecutionResult{result: result, err: executeErr}
	}()
	decoded := make(chan coordinatorDecodeResult, 1)
	go func() {
		response, decodeErr := decodeCoordinatorMessages(stdout, config.Progress)
		decodeErr = errors.Join(decodeErr, stdout.Close())
		decoded <- coordinatorDecodeResult{response: response, err: decodeErr}
	}()
	timer := time.NewTimer(max(time.Until(deadline)-reserve, 0))
	defer timer.Stop()
	var transport coordinatorExecutionResult
	var decodeResult coordinatorDecodeResult
	waitComplete, decodeComplete := false, false
	stopTransport := func() error {
		cancelExecution()
		if !waitComplete {
			transport = <-executed
			waitComplete = true
			executed = nil
		}
		var closeErr error
		if !decodeComplete {
			closeErr = stdout.Close()
			if errors.Is(closeErr, os.ErrClosed) {
				closeErr = nil
			}
		}
		cleanupErr := transport.err
		if transport.result.GroupGone && (transport.result.Cancelled || transport.result.WatchdogTimeout) && (errors.Is(cleanupErr, context.Canceled) || errors.Is(cleanupErr, context.DeadlineExceeded)) {
			cleanupErr = nil
		}
		return errors.Join(cleanupErr, closeErr)
	}
	for !waitComplete || !decodeComplete {
		select {
		case transport = <-executed:
			waitComplete = true
			executed = nil
		case decodeResult = <-decoded:
			decodeComplete = true
			decoded = nil
			if decodeResult.err != nil && !waitComplete {
				cleanupErr := stopTransport()
				switch transport.result.CoordinatorErrorReason {
				case "coordinator_start", "coordinator_pipe", "coordinator_capture", "coordinator_encode":
					return CampaignResult{}, &HostError{Reason: transport.result.CoordinatorErrorReason, Err: errors.Join(cleanupErr, errorFromOutput(transport.result.Stderr.Bytes))}
				}
				var progressErr *coordinatorProgressError
				reason := "coordinator_decode"
				if errors.As(decodeResult.err, &progressErr) {
					reason = "progress_output"
				}
				return CampaignResult{}, &HostError{Reason: reason, Err: errors.Join(decodeResult.err, cleanupErr)}
			}
		case <-overallCtx.Done():
			return CampaignResult{}, &HostError{Reason: contextFailureReason(overallCtx.Err()), Err: errors.Join(overallCtx.Err(), stopTransport())}
		case <-timer.C:
			return CampaignResult{}, &HostError{Reason: contextFailureReason(context.DeadlineExceeded), Err: errors.Join(context.DeadlineExceeded, stopTransport())}
		}
	}
	stderr := transport.result.Stderr.Bytes
	if transport.err != nil {
		reason := transport.result.CoordinatorErrorReason
		if reason == "" {
			reason = "coordinator_exit"
		}
		return CampaignResult{}, &HostError{Reason: reason, Err: errors.Join(transport.err, errorFromOutput(stderr))}
	}
	if decodeResult.err != nil {
		var progressErr *coordinatorProgressError
		reason := "coordinator_decode"
		if errors.As(decodeResult.err, &progressErr) {
			reason = "progress_output"
		}
		return CampaignResult{}, &HostError{Reason: reason, Err: errors.Join(decodeResult.err, errorFromOutput(stderr))}
	}
	response := decodeResult.response
	if response.ErrorReason != "" {
		if response.UnsupportedTarget != nil {
			return response.CampaignResult, &HostError{Reason: response.ErrorReason, Err: response.UnsupportedTarget}
		}
		if len(response.MissingSemanticProbes) != 0 {
			return response.CampaignResult, &record.MissingSemanticProbesError{Probes: append([]string(nil), response.MissingSemanticProbes...)}
		}
		return response.CampaignResult, &HostError{Reason: response.ErrorReason, Err: errors.New(response.ErrorDetail)}
	}
	return response.CampaignResult, nil
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

func errorFromOutput(output []byte) error {
	if len(output) == 0 {
		return nil
	}
	if len(output) > 4096 {
		output = output[:4096]
	}
	return errors.New(string(output))
}

func CoordinatorMain(input io.Reader, output io.Writer, provider backends.Provider) error {
	decoder := json.NewDecoder(io.LimitReader(input, maximumCoordinatorMessageBytes+1))
	decoder.DisallowUnknownFields()
	var wire coordinatorConfig
	if err := decoder.Decode(&wire); err != nil {
		return fmt.Errorf("decode coordinator request: %w", err)
	}
	if token, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("trailing coordinator request %v: %w", token, err)
	}
	if provider == nil {
		return errors.New("coordinator requires a backend provider")
	}
	description, err := provider.Describe(context.Background(), backends.Configuration{})
	if err != nil {
		return err
	}
	config := wire.campaignRequest(provider, description.RunnerBuild)
	config.profile = description
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
		var missing *record.MissingSemanticProbesError
		if errors.As(runErr, &missing) {
			response.MissingSemanticProbes = append([]string(nil), missing.Probes...)
		}
	}
	return encoder.Encode(coordinatorMessage{Type: "result", Response: &response})
}
