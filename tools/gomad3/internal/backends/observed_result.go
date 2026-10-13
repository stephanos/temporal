package backends

import (
	"errors"
	"os"

	hostexec "go.temporal.io/server/tools/gomad3/internal/host/exec"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
)

func ProjectObservedResult(request Request, observed Result, stdout, stderr *os.File) (ExecutionResult, error) {
	if observed.Termination != Exit && !observed.Cancelled && !observed.WatchdogTimeout {
		message := observed.Message
		if message == "" {
			message = "execution stopped without a guest exit"
		}
		return ExecutionResult{}, &FailureError{Termination: observed.Termination, Message: message, ChoiceDivergence: observed.ChoiceDivergence}
	}
	result := ExecutionResult{Captured: observed.Reaped, Termination: TerminationExit, ExitCode: observed.ExitCode, GroupGone: observed.Reaped, Cancelled: observed.Cancelled, WatchdogTimeout: observed.WatchdogTimeout, BackendEvidence: append([]byte(nil), observed.Evidence...)}
	result.VirtualTimeElapsedNanos = observed.VirtualTimeElapsedNanos
	if request.Choice != nil && !observed.Cancelled && !observed.WatchdogTimeout {
		if observed.ImplementationSHA256 != request.Choice.ExecutionIdentity.ImplementationSHA256 {
			return ExecutionResult{}, ErrChoiceTraceMalformed
		}
		trace, err := choice.DecodeStoredTrace(choice.Profile, observed.ChoiceTrace.Bytes, choice.TerminalMetadata{
			State: observed.ChoiceTrace.Summary.Terminal, Limit: request.Choice.Limit,
			Records: observed.ChoiceTrace.Summary.Records, SHA256: observed.ChoiceTrace.SHA256,
		})
		if err != nil {
			return ExecutionResult{}, err
		}
		if trace.Summary.Terminal != choice.TerminalComplete {
			return ExecutionResult{}, ErrChoiceTraceMalformed
		}
		trace.Summary.PeakGoroutines = observed.ChoiceTrace.Summary.PeakGoroutines
		result.ChoiceTrace = ChoiceTrace{Profile: choice.Profile, ImplementationSHA256: observed.ImplementationSHA256, Limit: request.Choice.Limit, Trace: trace}
		if request.Diagnostics {
			limit, err := choice.DiagnosticLimit(request.Choice.Limit)
			if err != nil {
				return ExecutionResult{}, err
			}
			diagnostic, err := choice.DecodeDiagnosticTrace(observed.DiagnosticTrace.Bytes)
			if err != nil {
				return ExecutionResult{}, err
			}
			if diagnostic.Capacity != limit || uint64(len(diagnostic.Records)) != trace.Summary.Records {
				return ExecutionResult{}, errors.New("diagnostic capacity or choice record count disagrees")
			}
			result.DiagnosticTrace = diagnostic
		}
	}
	for _, stream := range []struct {
		data   []byte
		sink   *os.File
		output *Output
	}{{observed.Stdout, stdout, &result.Stdout}, {observed.Stderr, stderr, &result.Stderr}} {
		capture, captureErr := hostexec.New(request.OutputBytes)
		if captureErr != nil {
			return ExecutionResult{}, captureErr
		}
		if _, err := capture.Write(stream.data); err != nil {
			return ExecutionResult{}, err
		}
		*stream.output = Output(capture.Result())
		if stream.sink != nil {
			if _, err := stream.sink.Write(stream.output.Bytes); err != nil {
				return ExecutionResult{}, err
			}
		}
	}
	if !observed.Reaped || (!observed.Cancelled && !observed.WatchdogTimeout && len(observed.Evidence) == 0) {
		return ExecutionResult{}, errors.New("backend execution did not produce complete reaped evidence")
	}
	return result, nil
}
