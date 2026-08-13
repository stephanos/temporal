package commandrun

import (
	"context"
	"fmt"
	"io"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/outputcapture"
)

type Termination string

const (
	TerminationExit   Termination = "exit"
	TerminationSignal Termination = "signal"
)

type Request struct {
	Command        []string
	Dir            string
	Env            []string
	Stdin          io.Reader
	Timeout        time.Duration
	TerminateGrace time.Duration
	OutputLimit    uint64
}

type Result struct {
	Termination     Termination
	ExitCode        int
	Signal          string
	SignalNumber    int
	WatchdogTimeout bool
	Cancelled       bool
	PID             int
	PGID            int
	GroupGone       bool
	Stdout          outputcapture.Output
	Stderr          outputcapture.Output
}

func validateRequest(request Request) error {
	if len(request.Command) == 0 || request.Command[0] == "" {
		return fmt.Errorf("command is required")
	}
	if request.Dir == "" {
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
