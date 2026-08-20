package temporal

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.temporal.io/server/tests/umpire3/environment"
	"go.temporal.io/server/tests/umpire3/protocol"
)

type WorkflowTaskIdentity struct {
	WorkflowID string
	RunID      string
	Source     string
	Reference  string
}

type WorkflowTaskDelivery struct {
	WorkflowTaskIdentity
	TaskToken []byte
}

type WorkflowTaskAcknowledgement struct {
	BacklogAbsent bool
	Source        string
	Reference     string
}

type WorkflowTaskTransport interface {
	Enqueue(context.Context) (WorkflowTaskIdentity, error)
	Deliver(context.Context, WorkflowTaskIdentity) (WorkflowTaskDelivery, error)
	Acknowledge(context.Context, WorkflowTaskDelivery) (WorkflowTaskAcknowledgement, error)
	Cleanup(context.Context, WorkflowTaskIdentity) error
}

type TaskAckFactory struct {
	probe     ClusterProbe
	transport WorkflowTaskTransport
}

func NewTaskAckFactory(probe ClusterProbe, transport WorkflowTaskTransport) *TaskAckFactory {
	return &TaskAckFactory{probe: probe, transport: transport}
}

func (f *TaskAckFactory) Capabilities() []string {
	return []string{"workflow-task-control"}
}

func (f *TaskAckFactory) Prepare(ctx context.Context, experiment protocol.Experiment) (environment.Session, error) {
	if f.probe == nil || f.transport == nil {
		return nil, errors.New("temporal cluster probe and Workflow Task transport are required")
	}
	cluster, err := f.probe(ctx)
	if err != nil {
		return nil, fmt.Errorf("probe Temporal cluster: %w", err)
	}
	if cluster.BuildID == "" || cluster.Namespace == "" {
		return nil, errors.New("cluster probe returned incomplete Workflow Task identity evidence")
	}
	return &taskAckSession{cluster: cluster, experimentID: experiment.ExperimentID, transport: f.transport}, nil
}

type taskAckSession struct {
	mu sync.Mutex

	cluster         ClusterInfo
	experimentID    string
	transport       WorkflowTaskTransport
	identity        WorkflowTaskIdentity
	delivery        WorkflowTaskDelivery
	acknowledgement WorkflowTaskAcknowledgement
	enqueued        bool
	delivered       bool
	acknowledged    bool
	sequence        int64
}

func (s *taskAckSession) Realize(
	ctx context.Context,
	action protocol.Action,
	_ environment.Bindings,
) (environment.ActionEvidence, error) {
	if err := ctx.Err(); err != nil {
		return environment.ActionEvidence{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sequence++
	var source, reference string
	switch action.Kind {
	case "enqueue-workflow-task":
		if s.enqueued {
			return environment.ActionEvidence{}, errors.New("workflow task is already enqueued")
		}
		identity, err := s.transport.Enqueue(ctx)
		if err != nil {
			return environment.ActionEvidence{}, fmt.Errorf("enqueue Workflow Task: %w", err)
		}
		if err := validateWorkflowTaskIdentity(identity); err != nil {
			return environment.ActionEvidence{}, err
		}
		s.identity = identity
		s.enqueued = true
		source, reference = identity.Source, identity.Reference
	case "deliver-workflow-task":
		if !s.enqueued || s.delivered {
			return environment.ActionEvidence{}, errors.New("workflow task is not deliverable")
		}
		delivery, err := s.transport.Deliver(ctx, s.identity)
		if err != nil {
			return environment.ActionEvidence{}, fmt.Errorf("deliver Workflow Task: %w", err)
		}
		if err := validateWorkflowTaskDelivery(s.identity, delivery); err != nil {
			return environment.ActionEvidence{}, err
		}
		s.delivery = delivery
		s.delivered = true
		source, reference = delivery.Source, delivery.Reference
	case "acknowledge-workflow-task":
		if !s.delivered || s.acknowledged {
			return environment.ActionEvidence{}, errors.New("workflow task is not acknowledgeable")
		}
		acknowledgement, err := s.transport.Acknowledge(ctx, s.delivery)
		if err != nil {
			return environment.ActionEvidence{}, fmt.Errorf("acknowledge Workflow Task: %w", err)
		}
		if acknowledgement.Source == "" || acknowledgement.Reference == "" {
			return environment.ActionEvidence{}, errors.New("workflow task acknowledgement lacks source evidence")
		}
		s.acknowledgement = acknowledgement
		s.acknowledged = true
		source, reference = acknowledgement.Source, acknowledgement.Reference
	default:
		return environment.ActionEvidence{}, fmt.Errorf("unsupported Workflow Task action %q", action.Kind)
	}
	return environment.ActionEvidence{
		Source: source, SourceIdentity: source, ClockDomain: source + "-sequence", SourceSequence: s.sequence,
		Reference: reference, EntityIdentity: s.identity.WorkflowID,
		Lineage: []string{s.cluster.Namespace, s.identity.WorkflowID, s.identity.RunID},
	}, nil
}

func (s *taskAckSession) Observe(
	ctx context.Context,
	checkpoint protocol.Checkpoint,
	_ environment.Bindings,
) (environment.Observation, error) {
	if err := ctx.Err(); err != nil {
		return environment.Observation{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if checkpoint.Observation != "workflow-task-acknowledged" {
		return environment.Observation{}, fmt.Errorf("unsupported Workflow Task observation %q", checkpoint.Observation)
	}
	if !s.acknowledged {
		return environment.Observation{}, environment.ErrObservationUnavailable
	}
	s.sequence++
	return environment.Observation{
		CheckpointID: checkpoint.Identifier, Kind: checkpoint.Observation,
		Satisfied: s.acknowledgement.BacklogAbsent,
		Source:    s.acknowledgement.Source, SourceIdentity: s.acknowledgement.Source,
		ClockDomain: s.acknowledgement.Source + "-sequence", SourceSequence: s.sequence,
		Reference:        s.acknowledgement.Reference,
		CausalReferences: []string{s.delivery.Reference, s.acknowledgement.Reference},
		EntityIdentity:   s.identity.WorkflowID,
		Lineage:          []string{s.cluster.Namespace, s.identity.WorkflowID, s.identity.RunID},
	}, nil
}

func (s *taskAckSession) Cleanup(ctx context.Context) environment.CleanupResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.transport.Cleanup(ctx, s.identity); err != nil {
		return environment.CleanupResult{Error: err.Error(), RecoverableResources: s.recoveryMetadata()}
	}
	return environment.CleanupResult{Complete: true}
}

func (s *taskAckSession) RecoveryMetadata() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.recoveryMetadata()
}

func (s *taskAckSession) recoveryMetadata() map[string]string {
	return map[string]string{
		"experimentID": s.experimentID,
		"namespace":    s.cluster.Namespace,
		"workflow":     s.identity.WorkflowID,
		"run":          s.identity.RunID,
	}
}

func (s *taskAckSession) Profile() environment.Profile {
	configurationIdentity := s.cluster.ConfigurationID
	if configurationIdentity == "" {
		configurationIdentity = s.cluster.BuildID + "/default"
	}
	return environment.Profile{
		Name: "ci-test-cluster", BuildID: s.cluster.BuildID,
		ConfigurationIdentity: configurationIdentity,
		EvidenceProfile:       environment.EvidenceProfilePublicGRPCHistory,
		DrivingAuthority:      "public-workflow-task-api",
		ObservationAuthority:  "public-workflow-task-response",
		FaultAuthority:        "none", IsolationIdentity: s.cluster.Namespace,
		RetentionClass: "semantic-redacted",
	}
}

func validateWorkflowTaskIdentity(identity WorkflowTaskIdentity) error {
	if identity.WorkflowID == "" || identity.RunID == "" || identity.Source == "" || identity.Reference == "" {
		return errors.New("workflow task enqueue returned incomplete identity evidence")
	}
	return nil
}

func validateWorkflowTaskDelivery(identity WorkflowTaskIdentity, delivery WorkflowTaskDelivery) error {
	if err := validateWorkflowTaskIdentity(delivery.WorkflowTaskIdentity); err != nil {
		return err
	}
	if delivery.WorkflowID != identity.WorkflowID || delivery.RunID != identity.RunID {
		return errors.New("workflow task delivery identity does not match enqueued lineage")
	}
	if len(delivery.TaskToken) == 0 {
		return errors.New("workflow task delivery returned an empty task token")
	}
	return nil
}
