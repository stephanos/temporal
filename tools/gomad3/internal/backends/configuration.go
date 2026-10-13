package backends

import (
	"go.temporal.io/server/tools/gomad3/internal/backends/target"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
)

type Configuration struct {
	// ForExecution requests executor readiness checks, not standalone validation.
	ForExecution           bool
	Target                 target.Spec
	IOProfile              string
	ClockTick              string
	Environment            []string
	RequiredSemanticProbes []string
}

type Description struct {
	Isolated                      bool
	RunnerBuild                   string
	IOProfile                     record.IOProfile
	Toolchain                     target.ToolchainIdentity
	BoundaryVersion               string
	BoundarySHA256                record.SHA256
	SemanticInstrumentationSHA256 record.SHA256
	SemanticProbes                []string
}

type PreparationError struct {
	Stage string
	Err   error
}

func (failure *PreparationError) Error() string { return failure.Err.Error() }
func (failure *PreparationError) Unwrap() error { return failure.Err }

type EvidenceRequest struct {
	Transcript             []byte
	Probes                 []string
	RequiredSemanticProbes []string
}

type OperationOutcome struct {
	Name   string
	Result uint32
}

type Evidence struct {
	Coverage              record.SemanticCoverage
	MissingRequiredProbes []string
	Operations            []OperationOutcome
}
