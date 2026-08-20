package execution

import (
	"context"
	"errors"

	"go.temporal.io/server/tests/umpire3/observation"
	"go.temporal.io/server/tests/umpire3/protocol"
)

var ErrObservationUnavailable = errors.New("observation unavailable")

type Bindings map[string]string

type ActionEvidence struct {
	Source              string                       `json:"source"`
	SourceIdentity      string                       `json:"sourceIdentity,omitempty"`
	ClockDomain         string                       `json:"clockDomain,omitempty"`
	SourceSequence      int64                        `json:"sourceSequence,omitempty"`
	Reference           string                       `json:"reference"`
	CausalReferences    []string                     `json:"causalReferences,omitempty"`
	EntityIdentity      string                       `json:"entityIdentity,omitempty"`
	Lineage             []string                     `json:"lineage,omitempty"`
	PayloadDigest       string                       `json:"payloadDigest,omitempty"`
	GroundedBindings    map[string]string            `json:"groundedBindings,omitempty"`
	TerminalState       string                       `json:"terminalState,omitempty"`
	TerminalDisposition protocol.TerminalDisposition `json:"terminalDisposition,omitempty"`
}

type Observation struct {
	CheckpointID              string   `json:"checkpointID"`
	Kind                      string   `json:"kind"`
	Satisfied                 bool     `json:"satisfied"`
	Source                    string   `json:"source"`
	SourceIdentity            string   `json:"sourceIdentity"`
	ClockDomain               string   `json:"clockDomain"`
	SourceSequence            int64    `json:"sourceSequence"`
	AuthoritativeTimeUnixNano int64    `json:"authoritativeTimeUnixNano,omitempty"`
	ObservedAtUnixNano        int64    `json:"observedAtUnixNano"`
	Reference                 string   `json:"reference"`
	CausalReference           string   `json:"causalReference,omitempty"`
	CausalReferences          []string `json:"causalReferences"`
	EntityIdentity            string   `json:"entityIdentity"`
	Lineage                   []string `json:"lineage"`
	PayloadDigest             string   `json:"payloadDigest,omitempty"`
}

type CleanupResult struct {
	Complete             bool              `json:"complete"`
	Error                string            `json:"error,omitempty"`
	RecoverableResources map[string]string `json:"recoverableResources,omitempty"`
}

type EnvironmentIdentity struct {
	Name                  string                  `json:"name,omitempty"`
	BuildID               string                  `json:"buildID,omitempty"`
	ConfigurationIdentity string                  `json:"configurationIdentity,omitempty"`
	EvidenceProfile       string                  `json:"evidenceProfile,omitempty"`
	DrivingAuthority      string                  `json:"drivingAuthority,omitempty"`
	ObservationAuthority  string                  `json:"observationAuthority,omitempty"`
	FaultAuthority        string                  `json:"faultAuthority,omitempty"`
	IsolationIdentity     string                  `json:"isolationIdentity,omitempty"`
	RetentionClass        string                  `json:"retentionClass,omitempty"`
	HardExecutionBudget   bool                    `json:"hardExecutionBudget"`
	Capabilities          []protocol.CapabilityID `json:"capabilities"`
}

type Factory interface {
	Capabilities() []protocol.CapabilityID
	Prepare(context.Context, protocol.Experiment) (PreparedEnvironment, error)
}

type PreparedEnvironment struct {
	Session  Session
	Identity EnvironmentIdentity
}

type Session interface {
	Realize(context.Context, protocol.Action, Bindings) (ActionEvidence, error)
	Cleanup(context.Context) CleanupResult
	RecoveryMetadata() map[string]string
}

type ObservationSession interface {
	Observe(context.Context, protocol.Checkpoint, Bindings) (Observation, error)
}

type FactSession interface {
	ObserveFacts(context.Context, protocol.Checkpoint, Bindings) ([]observation.Fact, error)
}

// CorroboratingSession advertises that every primary observation must be corroborated by an independent source.
type CorroboratingSession interface {
	Corroborate(context.Context, protocol.Checkpoint, Bindings) ([]Observation, error)
}
