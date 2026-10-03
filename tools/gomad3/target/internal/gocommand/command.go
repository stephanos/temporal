package gocommand

import (
	"context"
	"fmt"
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

type DiagnosticResult struct {
	Stdout hostexec.Output
	Stderr hostexec.Output
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
