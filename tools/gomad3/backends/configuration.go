package backends

import (
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

type Configuration struct {
	Target                 target.Spec
	IOProfile              string
	ClockTick              string
	Environment            []string
	RequiredSemanticProbes []string
}

type Description struct {
	IOProfile                     record.IOProfile
	Toolchain                     target.ToolchainIdentity
	BoundaryVersion               string
	BoundarySHA256                record.SHA256
	SemanticInstrumentationSHA256 record.SHA256
	SemanticProbes                []string
}

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
