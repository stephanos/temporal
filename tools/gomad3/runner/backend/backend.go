package backend

import (
	"context"
	"time"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/target"
)

type Provider interface {
	Prepare(context.Context, target.Spec) (target.Prepared, error)
	ValidatePrepared(target.Spec, target.Prepared, []string) error
	ValidateReplay(context.Context, *artifact.Opened) (target.Prepared, error)
	Run(context.Context, Request) (Result, error)
}

type Request struct {
	Target                       target.Prepared
	Seed                         uint64
	Environment                  []string
	Timeout                      time.Duration
	OutputBytes, TranscriptBytes uint64
	Choice                       *ChoiceRequest
	Diagnostics                  bool
}

type ChoiceRequest struct {
	Mode              choice.Mode
	ExecutionIdentity choice.ExecutionIdentity
	Limit             uint64
	Tape              *choice.ReplayPlan
}

type Result struct {
	Termination                        Termination
	ExitCode                           int
	Stdout, Stderr, Evidence           []byte
	Reaped, Cancelled, WatchdogTimeout bool
	ChoiceTrace                        choice.Trace
	DiagnosticTrace                    choice.DiagnosticTrace
	ImplementationSHA256               [32]byte
	ChoiceDivergence                   *choice.Divergence
}

type Termination string

type FailureError struct {
	Termination      Termination
	Message          string
	ChoiceDivergence *choice.Divergence
}

func (failure *FailureError) Error() string {
	return "backend " + string(failure.Termination) + ": " + failure.Message
}

const (
	Exit           Termination = "exit"
	Unsupported    Termination = "unsupported"
	Capacity       Termination = "capacity"
	Divergence     Termination = "divergence"
	Trap           Termination = "trap"
	Infrastructure Termination = "infrastructure"
)
