package hostexec

import (
	"context"
	"io"
	"time"

	internal "go.temporal.io/server/tools/gomad3/internal/hostexec"
)

type Request struct {
	CollectResourceUsage bool
	Command              []string
	Dir                  string
	Env                  []string
	Stdin                io.Reader
	StdoutSink           io.Writer
	StdoutDone           func()
	Timeout              time.Duration
	TerminateGrace       time.Duration
	OutputLimit          uint64
}
type Result struct {
	ResourceUsage                         *ResourceUsage `json:",omitempty"`
	Termination                           string
	ExitCode                              int
	WatchdogTimeout, Cancelled, GroupGone bool
	Stdout, Stderr                        []byte
}

type ResourceUsage struct {
	UserNanos, SystemNanos, PeakRSSBytes uint64
}

const TerminationExit = "exit"

// CurrentProcessResourceUsage is an opt-in RUSAGE_SELF snapshot. Peak RSS is
// this process's lifetime high-water, including setup/models/capture/artifacts;
// it excludes child helpers and compilers and is not model-only memory.
func CurrentProcessResourceUsage() *ResourceUsage {
	return publicResourceUsage(internal.CurrentProcessResourceUsage())
}

func publicResourceUsage(usage *internal.ResourceUsage) *ResourceUsage {
	if usage == nil {
		return nil
	}
	return &ResourceUsage{UserNanos: usage.UserNanos, SystemNanos: usage.SystemNanos, PeakRSSBytes: usage.PeakRSSBytes}
}

func Run(ctx context.Context, request Request) (Result, error) {
	result, err := internal.Run(ctx, internal.Request{Command: request.Command, Dir: request.Dir, Env: request.Env, Stdin: request.Stdin, StdoutSink: request.StdoutSink, StdoutDone: request.StdoutDone, Timeout: request.Timeout, TerminateGrace: request.TerminateGrace, OutputLimit: request.OutputLimit, CollectResourceUsage: request.CollectResourceUsage})
	usage := publicResourceUsage(result.ResourceUsage)
	return Result{Termination: string(result.Termination), ExitCode: result.ExitCode, WatchdogTimeout: result.WatchdogTimeout, Cancelled: result.Cancelled, GroupGone: result.GroupGone, Stdout: result.Stdout.RawBytes, Stderr: result.Stderr.RawBytes, ResourceUsage: usage}, err
}
