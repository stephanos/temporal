package backends

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/backends/target"
	"go.temporal.io/server/tools/gomad3/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
)

type Provider interface {
	Capabilities() Capabilities
	Prepare(context.Context, target.Spec) (target.Prepared, error)
	ValidatePrepared(target.Spec, target.Prepared, []string) error
	ValidateReplay(context.Context, *artifact.Opened) (target.Prepared, error)
	Run(context.Context, Request) (Result, error)
	Describe(context.Context, Configuration) (Description, error)
	Execute(context.Context, ExecutionRequest) (ExecutionResult, error)
	Evidence(context.Context, EvidenceRequest) (Evidence, error)
}

type Capabilities struct {
	Name           string
	ReplayModes    []string
	ChoiceProfiles []string
	Simulation     bool
	ReadOnlyMounts bool
	IOTranscript   bool
	Commands       []string
}

type Registry map[string]Capabilities

func (registry Registry) Resolve(name string) (Capabilities, error) {
	capabilities, found := registry[name]
	if !found || capabilities.Name != name {
		return Capabilities{}, fmt.Errorf("unregistered backend %q", name)
	}
	return capabilities, nil
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
	VirtualTimeElapsedNanos            uint64
	Termination                        Termination
	Message                            string
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
	SoftAssertions   *record.SoftAssertions
	Termination      Termination
	Message          string
	ChoiceDivergence *choice.Divergence
}

func (failure *FailureError) Error() string {
	message := "backend " + string(failure.Termination) + ": " + failure.Message
	if failure.SoftAssertions != nil && failure.SoftAssertions.Inconclusive {
		message += "; soft assertion oracle inconclusive"
	}
	return message
}

const (
	Exit           Termination = "exit"
	Unsupported    Termination = "unsupported"
	Capacity       Termination = "capacity"
	Divergence     Termination = "divergence"
	Trap           Termination = "trap"
	Infrastructure Termination = "infrastructure"
)
