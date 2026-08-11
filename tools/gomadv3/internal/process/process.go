package process

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
	SupervisorCommand    []string
	BootstrapCommand     []string
	Command              string
	Args                 []string
	Argv0                string
	Dir                  string
	Env                  []string
	RunTimeout           time.Duration
	TerminateGrace       time.Duration
	OutputLimit          uint64
	WorldRecordLimit     uint64
	WorldTransitionLimit uint64
	WorldSeed            uint64
	ExpectedWorldInitial []byte
	StdoutHead           io.Writer
	StderrHead           io.Writer
}

type Result struct {
	Captured        bool
	Termination     Termination
	ExitCode        int
	Signal          string
	WatchdogTimeout bool
	Cancelled       bool
	Stdout          Output
	Stderr          Output
	PID             int
	PGID            int
	GroupGone       bool
	WorldRecord     []byte
}

func validateRequest(request Request) error {
	if len(request.SupervisorCommand) == 0 || request.SupervisorCommand[0] == "" {
		return fmt.Errorf("supervisor command is required")
	}
	if len(request.BootstrapCommand) == 0 || request.BootstrapCommand[0] == "" {
		return fmt.Errorf("target bootstrap command is required")
	}
	if request.Command == "" {
		return fmt.Errorf("target command is required")
	}
	if request.Argv0 == "" {
		return fmt.Errorf("target argv[0] is required")
	}
	if request.Dir == "" {
		return fmt.Errorf("target working directory is required")
	}
	if request.RunTimeout <= 0 {
		return fmt.Errorf("run timeout must be positive")
	}
	if request.TerminateGrace < 0 {
		return fmt.Errorf("termination grace must not be negative")
	}
	if request.OutputLimit == 0 {
		return fmt.Errorf("output limit must be positive")
	}
	if request.WorldRecordLimit == 0 || request.WorldTransitionLimit == 0 {
		return fmt.Errorf("World record and transition limits must be positive")
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
	if remaining < configured {
		return remaining, nil
	}
	return configured, nil
}
