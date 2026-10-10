package runner

import (
	"context"
	"errors"
	"io"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/internal/hostexec"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/backend"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	"go.temporal.io/server/tools/gomad3/target"
)

func campaignPreparer(config campaignRequest) Preparer {
	if config.Target.Backend != "" {
		return config.Backend
	}
	return config.Preparer
}

func backendValidator(config campaignRequest) func(target.Spec, target.Prepared, []string) error {
	if config.Target.Backend == "" || config.Backend == nil {
		return nil
	}
	return config.Backend.ValidatePrepared
}

func recordedBackend(prepared target.Prepared, result execution.Result) *record.BackendMetadata {
	metadata := record.CloneBackendMetadata(prepared.Backend)
	if metadata != nil && len(result.BackendEvidence) != 0 {
		metadata.Evidence = &record.BackendPayload{Schema: "backend-evidence/v1", File: "backend/evidence.bin", SHA256: record.HashBytes(result.BackendEvidence), Bytes: record.Uint64String(len(result.BackendEvidence))}
	}
	return metadata
}

func backendChoiceRequest(capability *execution.ChoiceCapability) *backend.ChoiceRequest {
	if capability == nil {
		return nil
	}
	return &backend.ChoiceRequest{Mode: capability.Mode, ExecutionIdentity: capability.ExecutionIdentity, Limit: capability.Limit, Tape: capability.ReplayPlan}
}

func runBackend(ctx context.Context, provider backend.Provider, request backend.Request, stdout, stderr io.Writer) (execution.Result, error) {
	if provider == nil {
		return execution.Result{}, errors.New("external backend provider is required")
	}
	observed, err := provider.Run(ctx, request)
	if err != nil {
		return execution.Result{}, err
	}
	if observed.Termination != backend.Exit && !observed.Cancelled && !observed.WatchdogTimeout {
		return execution.Result{}, &backend.FailureError{Termination: observed.Termination, Message: "execution stopped without a guest exit", ChoiceDivergence: observed.ChoiceDivergence}
	}
	result := execution.Result{Captured: observed.Reaped, Termination: execution.TerminationExit, ExitCode: observed.ExitCode, GroupGone: observed.Reaped, Cancelled: observed.Cancelled, WatchdogTimeout: observed.WatchdogTimeout, BackendEvidence: append([]byte(nil), observed.Evidence...)}
	if request.Choice != nil && !observed.Cancelled && !observed.WatchdogTimeout {
		if observed.ImplementationSHA256 != request.Choice.ExecutionIdentity.ImplementationSHA256 {
			return execution.Result{}, execution.ErrChoiceTraceMalformed
		}
		trace, err := choice.DecodeStoredTrace(choice.Profile, observed.ChoiceTrace.Bytes, choice.TerminalMetadata{
			State: observed.ChoiceTrace.Summary.Terminal, Limit: request.Choice.Limit,
			Records: observed.ChoiceTrace.Summary.Records, SHA256: observed.ChoiceTrace.SHA256,
		})
		if err != nil {
			return execution.Result{}, err
		}
		if trace.Summary.Terminal != choice.TerminalComplete {
			return execution.Result{}, execution.ErrChoiceTraceMalformed
		}
		trace.Summary.PeakGoroutines = observed.ChoiceTrace.Summary.PeakGoroutines
		result.ChoiceTrace = execution.ChoiceTrace{Profile: choice.Profile, ImplementationSHA256: observed.ImplementationSHA256, Limit: request.Choice.Limit, Trace: trace}
		if request.Diagnostics {
			limit, err := choice.DiagnosticLimit(request.Choice.Limit)
			if err != nil {
				return execution.Result{}, err
			}
			diagnostic, err := choice.DecodeDiagnosticTrace(observed.DiagnosticTrace.Bytes)
			if err != nil {
				return execution.Result{}, err
			}
			if diagnostic.Capacity != limit || uint64(len(diagnostic.Records)) != trace.Summary.Records {
				return execution.Result{}, errors.New("diagnostic capacity or choice record count disagrees")
			}
			result.DiagnosticTrace = diagnostic
		}
	}
	for _, stream := range []struct {
		data   []byte
		sink   io.Writer
		output *hostexec.Output
	}{{observed.Stdout, stdout, &result.Stdout}, {observed.Stderr, stderr, &result.Stderr}} {
		capture, captureErr := hostexec.New(request.OutputBytes)
		if captureErr != nil {
			return execution.Result{}, captureErr
		}
		if _, err := capture.Write(stream.data); err != nil {
			return execution.Result{}, err
		}
		*stream.output = capture.Result()
		if stream.sink != nil {
			if _, err := stream.sink.Write(stream.output.Bytes); err != nil {
				return execution.Result{}, err
			}
		}
	}
	if !observed.Reaped || (!observed.Cancelled && !observed.WatchdogTimeout && len(observed.Evidence) == 0) {
		return execution.Result{}, errors.New("backend execution did not produce complete reaped evidence")
	}
	return result, nil
}
