package environment

import (
	"context"
	"errors"

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

type Profile struct {
	Name                  string `json:"name"`
	BuildID               string `json:"buildID"`
	ConfigurationIdentity string `json:"configurationIdentity"`
	EvidenceProfile       string `json:"evidenceProfile"`
	DrivingAuthority      string `json:"drivingAuthority"`
	ObservationAuthority  string `json:"observationAuthority"`
	FaultAuthority        string `json:"faultAuthority"`
	IsolationIdentity     string `json:"isolationIdentity"`
	RetentionClass        string `json:"retentionClass"`
	HardExecutionBudget   bool   `json:"hardExecutionBudget"`
}

type ProfileProvider interface {
	Profile() Profile
}

type Factory interface {
	Capabilities() []string
	Prepare(context.Context, protocol.Experiment) (Session, error)
}

type Session interface {
	Realize(context.Context, protocol.Action, Bindings) (ActionEvidence, error)
	Observe(context.Context, protocol.Checkpoint, Bindings) (Observation, error)
	Cleanup(context.Context) CleanupResult
	RecoveryMetadata() map[string]string
}

// CorroboratingSession advertises that every primary observation must be corroborated by an independent source.
type CorroboratingSession interface {
	Corroborate(context.Context, protocol.Checkpoint, Bindings) ([]Observation, error)
}
