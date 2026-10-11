package hostexec

import (
	"context"
	"fmt"
	"io"
	"time"
)

type Termination string

const (
	TerminationExit   Termination = "exit"
	TerminationSignal Termination = "signal"
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
	PreserveCommandError bool
	CombinedOutput       bool
}

type Result struct {
	ResourceUsage   *ResourceUsage `json:",omitempty"`
	Termination     Termination
	ExitCode        int
	Signal          string
	SignalNumber    int
	WatchdogTimeout bool
	Cancelled       bool
	PID             int
	PGID            int
	GroupGone       bool
	Stdout          Output
	Stderr          Output
	CommandError    error
}

// ResourceUsage is wait's accounting for the reaped leader, including only
// descendants whose usage the OS credits to that leader. It is not process-group
// accounting for arbitrary detached/unwaited descendants. PeakRSSBytes follows
// the platform wait-rusage high-water semantics, not a sum of concurrent RSS.
type ResourceUsage struct {
	UserNanos, SystemNanos, PeakRSSBytes uint64
}

func validateRequest(request Request) error {
	if !request.PreserveCommandError && (len(request.Command) == 0 || request.Command[0] == "") {
		return fmt.Errorf("command is required")
	}
	if !request.PreserveCommandError && request.Dir == "" {
		return fmt.Errorf("working directory is required")
	}
	if request.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if request.TerminateGrace < 0 {
		return fmt.Errorf("termination grace must not be negative")
	}
	if request.OutputLimit == 0 {
		return fmt.Errorf("output limit must be positive")
	}
	return nil
}

func effectiveTimeout(ctx context.Context, configured time.Duration) (time.Duration, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return configured, nil
	}
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return 0, context.DeadlineExceeded
	}
	return min(configured, remaining), nil
}
