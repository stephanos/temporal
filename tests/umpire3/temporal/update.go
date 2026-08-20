package temporal

import (
	"context"
	"errors"
	"fmt"
	"sync"

	environment "go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/protocol"
)

type updateFactory struct {
	probe clusterProbe
}

func newUpdateFactory(probe clusterProbe) *updateFactory {
	return &updateFactory{probe: probe}
}

func (f *updateFactory) Capabilities() []protocol.CapabilityID {
	return []protocol.CapabilityID{
		protocol.CapabilityIDUpdate, protocol.CapabilityIDWorkflowTaskControl,
		protocol.CapabilityIDHistoryObservation,
	}
}

func (f *updateFactory) Prepare(ctx context.Context, experiment protocol.Experiment) (environment.PreparedEnvironment, error) {
	if f.probe == nil {
		return environment.PreparedEnvironment{}, errors.New("temporal cluster probe is required")
	}
	cluster, err := f.probe(ctx)
	if err != nil {
		return environment.PreparedEnvironment{}, fmt.Errorf("probe Temporal cluster: %w", err)
	}
	if cluster.BuildID == "" || cluster.Namespace == "" || cluster.MintedWorkflowID == "" || cluster.MintedUpdateID == "" {
		return environment.PreparedEnvironment{}, errors.New("cluster probe returned incomplete Update identity evidence")
	}
	session := &updateSession{cluster: cluster, experimentID: experiment.ExperimentID}
	return environment.PreparedEnvironment{Session: session, Identity: session.environmentIdentity(f.Capabilities())}, nil
}

type updateSession struct {
	mu sync.Mutex

	cluster         clusterInfo
	experimentID    string
	started         bool
	dispatched      bool
	accepted        bool
	historyRecorded bool
	taskCompleted   bool
	completed       bool
	sequence        int64
}

func (s *updateSession) Realize(ctx context.Context, action protocol.Action, bindings environment.Bindings) (environment.ActionEvidence, error) {
	if err := ctx.Err(); err != nil {
		return environment.ActionEvidence{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	evidence := environment.ActionEvidence{
		Source:         "temporal-test-cluster",
		SourceIdentity: "temporal-test-cluster",
		ClockDomain:    "temporal-test-cluster-sequence",
		Reference:      s.cluster.Namespace + "/" + action.Identifier,
		EntityIdentity: s.cluster.MintedUpdateID,
		Lineage:        []string{s.cluster.Namespace, s.cluster.MintedWorkflowID, s.cluster.MintedUpdateID},
	}
	if err := validateIdentityArgument(action, "update", s.cluster.MintedUpdateID, bindings); err != nil {
		return environment.ActionEvidence{}, err
	}
	switch action.Kind {
	case "start-update":
		s.started = true
		evidence.GroundedBindings = map[string]string{
			"workflow": s.cluster.MintedWorkflowID,
			"update":   s.cluster.MintedUpdateID,
		}
		grounded, err := groundActionBindings(action, map[string]string{"update-id": s.cluster.MintedUpdateID})
		if err != nil {
			return environment.ActionEvidence{}, err
		}
		for symbol, concrete := range grounded {
			evidence.GroundedBindings[symbol] = concrete
		}
	case "dispatch-workflow-task":
		if !s.started {
			return environment.ActionEvidence{}, errors.New("update is not started")
		}
		s.dispatched = true
	case "accept-update":
		if !s.dispatched {
			return environment.ActionEvidence{}, errors.New("workflow task is not dispatched")
		}
		s.accepted = true
	case "record-update-history":
		if !s.accepted {
			return environment.ActionEvidence{}, errors.New("update is not accepted")
		}
		s.historyRecorded = true
	case "complete-workflow-task":
		if !s.dispatched {
			return environment.ActionEvidence{}, errors.New("workflow task is not dispatched")
		}
		s.taskCompleted = true
	case "complete-update":
		if !s.accepted || !s.historyRecorded || !s.taskCompleted {
			return environment.ActionEvidence{}, errors.New("update completion prerequisites are missing")
		}
		s.completed = true
	default:
		return environment.ActionEvidence{}, fmt.Errorf("unsupported Update action %q", action.Kind)
	}
	return evidence, nil
}

func (s *updateSession) Observe(ctx context.Context, checkpoint protocol.Checkpoint, _ environment.Bindings) (environment.Observation, error) {
	if err := ctx.Err(); err != nil {
		return environment.Observation{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	observation := environment.Observation{
		CheckpointID:    checkpoint.Identifier,
		Kind:            checkpoint.Observation,
		Source:          "temporal-history",
		SourceIdentity:  "temporal-history",
		ClockDomain:     "temporal-history-sequence",
		SourceSequence:  s.sequence,
		CausalReference: s.cluster.Namespace + "/" + s.cluster.MintedWorkflowID + "/" + s.cluster.MintedUpdateID,
		Reference: s.cluster.Namespace + "/" + s.cluster.MintedWorkflowID + "/" +
			s.cluster.MintedUpdateID + "/" + checkpoint.Identifier,
		EntityIdentity: s.cluster.MintedUpdateID,
		Lineage:        []string{s.cluster.Namespace, s.cluster.MintedWorkflowID, s.cluster.MintedUpdateID},
	}
	switch checkpoint.Observation {
	case "update-accepted":
		observation.Satisfied = s.accepted
	case "update-completed":
		observation.Satisfied = s.completed
	default:
		return environment.Observation{}, fmt.Errorf("unsupported Update observation %q", checkpoint.Observation)
	}
	return observation, nil
}

func (s *updateSession) Cleanup(ctx context.Context) environment.CleanupResult {
	if err := ctx.Err(); err != nil {
		return environment.CleanupResult{Error: err.Error(), RecoverableResources: s.RecoveryMetadata()}
	}
	return environment.CleanupResult{Complete: true}
}

func (s *updateSession) RecoveryMetadata() map[string]string {
	return map[string]string{
		"experimentID": s.experimentID,
		"namespace":    s.cluster.Namespace,
		"workflow":     s.cluster.MintedWorkflowID,
		"update":       s.cluster.MintedUpdateID,
	}
}

func (s *updateSession) environmentIdentity(capabilities []protocol.CapabilityID) environment.EnvironmentIdentity {
	configurationIdentity := s.cluster.ConfigurationID
	if configurationIdentity == "" {
		configurationIdentity = s.cluster.BuildID + "/default"
	}
	return environment.EnvironmentIdentity{
		Name:                  "controlled-local",
		BuildID:               s.cluster.BuildID,
		ConfigurationIdentity: configurationIdentity,
		EvidenceProfile:       environment.EvidenceProfileInProcessHooks,
		DrivingAuthority:      "temporal-api",
		ObservationAuthority:  "controlled-state-hooks",
		FaultAuthority:        "none",
		IsolationIdentity:     s.cluster.Namespace,
		RetentionClass:        "semantic-only",
		Capabilities:          append([]protocol.CapabilityID(nil), capabilities...),
	}
}
