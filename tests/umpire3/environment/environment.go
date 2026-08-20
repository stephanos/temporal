package environment

import (
	"context"
	"errors"

	"go.temporal.io/server/tests/umpire3/protocol"
)

var ErrObservationUnavailable = errors.New("observation unavailable")

type Bindings map[string]string

type ActionEvidence struct {
	Source           string            `json:"source"`
	Reference        string            `json:"reference"`
	GroundedBindings map[string]string `json:"groundedBindings,omitempty"`
}

type Observation struct {
	CheckpointID    string `json:"checkpointID"`
	Kind            string `json:"kind"`
	Satisfied       bool   `json:"satisfied"`
	Source          string `json:"source"`
	SourceSequence  int64  `json:"sourceSequence,omitempty"`
	CausalReference string `json:"causalReference,omitempty"`
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
