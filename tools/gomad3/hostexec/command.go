package hostexec

import (
	"context"
	"io"
	"time"

	internal "go.temporal.io/server/tools/gomad3/internal/hostexec"
)

type Request struct {
	Command        []string
	Dir            string
	Env            []string
	Stdin          io.Reader
	StdoutSink     io.Writer
	StdoutDone     func()
	Timeout        time.Duration
	TerminateGrace time.Duration
	OutputLimit    uint64
}
type Result struct {
	Termination                           string
	ExitCode                              int
	WatchdogTimeout, Cancelled, GroupGone bool
	Stdout, Stderr                        []byte
}

const TerminationExit = "exit"

func Run(ctx context.Context, request Request) (Result, error) {
	result, err := internal.Run(ctx, internal.Request{Command: request.Command, Dir: request.Dir, Env: request.Env, Stdin: request.Stdin, StdoutSink: request.StdoutSink, StdoutDone: request.StdoutDone, Timeout: request.Timeout, TerminateGrace: request.TerminateGrace, OutputLimit: request.OutputLimit})
	return Result{Termination: string(result.Termination), ExitCode: result.ExitCode, WatchdogTimeout: result.WatchdogTimeout, Cancelled: result.Cancelled, GroupGone: result.GroupGone, Stdout: result.Stdout.RawBytes, Stderr: result.Stderr.RawBytes}, err
}
