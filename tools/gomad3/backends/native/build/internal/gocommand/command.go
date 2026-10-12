package gocommand

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
)

const defaultTimeout = 15 * time.Minute
const terminationGrace = 100 * time.Millisecond

type Request struct {
	Command     []string
	Dir         string
	Env         []string
	Timeout     time.Duration
	OutputLimit uint64
}

type Runner struct {
	run func(context.Context, hostexec.Request) (hostexec.Result, error)
}

func Default() Runner {
	return New(hostexec.Run)
}

func New(run func(context.Context, hostexec.Request) (hostexec.Result, error)) Runner {
	return Runner{run: run}
}

type StructuredResult struct {
	Stdout []byte
	Stderr []byte
}

type StructuredCommandResult struct {
	Stdout       []byte
	Stderr       []byte
	CommandError error
}

func (result StructuredCommandResult) OutputError() error {
	if exit, ok := result.CommandError.(*exec.ExitError); ok {
		const half = 32 << 10
		if len(result.Stderr) <= 2*half {
			exit.Stderr = append([]byte(nil), result.Stderr...)
		} else {
			exit.Stderr = append([]byte(nil), result.Stderr[:half]...)
			exit.Stderr = fmt.Appendf(exit.Stderr, "\n... omitting %d bytes ...\n", len(result.Stderr)-2*half)
			exit.Stderr = append(exit.Stderr, result.Stderr[len(result.Stderr)-half:]...)
		}
	}
	return result.CommandError
}

func (runner Runner) StructuredCommand(ctx context.Context, request Request) (StructuredCommandResult, error) {
	timeout := request.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	result, err := runner.run(ctx, hostexec.Request{
		Command: request.Command, Dir: request.Dir, Env: request.Env,
		Timeout: timeout, TerminateGrace: terminationGrace, OutputLimit: request.OutputLimit,
		PreserveCommandError: true,
	})
	structured := StructuredCommandResult{}
	if !result.Stderr.Truncated {
		structured.Stderr = result.Stderr.RawBytes
	}
	if err != nil {
		return structured, err
	}
	if result.Stdout.Truncated {
		return structured, &OverflowError{Stream: "stdout", Limit: request.OutputLimit}
	}
	if result.Stderr.Truncated {
		return structured, &OverflowError{Stream: "stderr", Limit: request.OutputLimit}
	}
	if result.WatchdogTimeout {
		return structured, &WatchdogError{Timeout: timeout, Cause: result.CommandError}
	}
	structured.Stdout = result.Stdout.RawBytes
	structured.CommandError = result.CommandError
	return structured, nil
}

type DiagnosticResult struct {
	Stdout hostexec.Output
	Stderr hostexec.Output
}

type DiagnosticCommandResult struct {
	Output       hostexec.Output
	CommandError error
}

func (runner Runner) DiagnosticCommand(ctx context.Context, request Request) (DiagnosticCommandResult, error) {
	timeout := request.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	result, err := runner.run(ctx, hostexec.Request{
		Command: request.Command, Dir: request.Dir, Env: request.Env,
		Timeout: timeout, TerminateGrace: terminationGrace, OutputLimit: request.OutputLimit,
		PreserveCommandError: true, CombinedOutput: true,
	})
	diagnostic := DiagnosticCommandResult{Output: result.Stdout, CommandError: result.CommandError}
	if err != nil {
		return diagnostic, err
	}
	if diagnostic.CommandError == nil {
		// Existing injected runners may supply only the original result fields.
		if len(result.Stderr.Bytes) != 0 {
			diagnostic.Output = hostexec.Output{Bytes: append(append([]byte(nil), result.Stdout.Bytes...), result.Stderr.Bytes...), Truncated: result.Stdout.Truncated || result.Stderr.Truncated}
		}
		switch {
		case result.Cancelled:
			diagnostic.CommandError = context.Canceled
		case result.WatchdogTimeout:
			diagnostic.CommandError = context.DeadlineExceeded
		default:
			diagnostic.CommandError = exitError(result)
		}
	}
	if result.WatchdogTimeout {
		return diagnostic, &WatchdogError{Timeout: timeout, Cause: diagnostic.CommandError}
	}
	return diagnostic, nil
}

func (result DiagnosticResult) Combined() []byte {
	combined := append([]byte(nil), result.Stdout.Bytes...)
	return append(combined, result.Stderr.Bytes...)
}

type OverflowError struct {
	Stream string
	Limit  uint64
}

func (err *OverflowError) Error() string {
	return fmt.Sprintf("%s output exceeds %d bytes", err.Stream, err.Limit)
}

type ExitError struct {
	Code   int
	Signal string
}

func (err *ExitError) Error() string {
	if err.Signal != "" {
		return "signal: " + err.Signal
	}
	return fmt.Sprintf("exit status %d", err.Code)
}

func (runner Runner) Structured(ctx context.Context, request Request) (StructuredResult, error) {
	result, err := runner.execute(ctx, request)
	if err != nil {
		return StructuredResult{}, err
	}
	if result.Stdout.Truncated {
		return StructuredResult{}, &OverflowError{Stream: "stdout", Limit: request.OutputLimit}
	}
	if result.Stderr.Truncated {
		return StructuredResult{}, &OverflowError{Stream: "stderr", Limit: request.OutputLimit}
	}
	structured := StructuredResult{Stdout: result.Stdout.RawBytes, Stderr: result.Stderr.RawBytes}
	return structured, exitError(result)
}

type WatchdogError struct {
	Timeout time.Duration
	Cause   error
}

func (err *WatchdogError) Error() string {
	return fmt.Sprintf("command watchdog exceeded %s: %v", err.Timeout, err.Cause)
}

func (err *WatchdogError) Unwrap() error { return err.Cause }

func (runner Runner) Compatibility(ctx context.Context, request Request) (StructuredResult, error) {
	timeout := request.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	result, err := runner.run(ctx, hostexec.Request{
		Command: request.Command, Dir: request.Dir, Env: request.Env,
		Timeout: timeout, TerminateGrace: terminationGrace, OutputLimit: request.OutputLimit,
		PreserveCommandError: true,
	})
	structured := StructuredResult{}
	if !result.Stderr.Truncated {
		structured.Stderr = result.Stderr.RawBytes
	}
	if err != nil {
		return structured, err
	}
	if result.Stdout.Truncated {
		return structured, &OverflowError{Stream: "stdout", Limit: request.OutputLimit}
	}
	if result.Stderr.Truncated {
		return structured, &OverflowError{Stream: "stderr", Limit: request.OutputLimit}
	}
	if result.WatchdogTimeout {
		return structured, &WatchdogError{Timeout: timeout, Cause: result.CommandError}
	}
	if result.CommandError != nil {
		return structured, result.CommandError
	}
	structured.Stdout = result.Stdout.RawBytes
	return structured, nil
}

func (runner Runner) Diagnostic(ctx context.Context, request Request) (DiagnosticResult, error) {
	result, err := runner.execute(ctx, request)
	diagnostic := DiagnosticResult{Stdout: result.Stdout, Stderr: result.Stderr}
	if err != nil {
		return diagnostic, err
	}
	return diagnostic, exitError(result)
}

func (runner Runner) execute(ctx context.Context, request Request) (hostexec.Result, error) {
	timeout := request.Timeout
	if timeout == 0 {
		// Go commands without a caller deadline still need a finite watchdog.
		timeout = defaultTimeout
	}
	result, err := runner.run(ctx, hostexec.Request{
		Command: request.Command, Dir: request.Dir, Env: request.Env,
		Timeout: timeout, TerminateGrace: terminationGrace, OutputLimit: request.OutputLimit,
	})
	if err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if result.Cancelled {
		return result, context.Canceled
	}
	if result.WatchdogTimeout {
		return result, context.DeadlineExceeded
	}
	return result, nil
}

func exitError(result hostexec.Result) error {
	if result.Termination == hostexec.TerminationExit {
		if result.ExitCode == 0 {
			return nil
		}
		return &ExitError{Code: result.ExitCode}
	}
	return &ExitError{Signal: result.Signal}
}
