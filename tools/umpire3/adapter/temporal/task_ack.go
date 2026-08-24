package temporal

import (
	"context"
	"errors"
	"fmt"
	"sync"

	environment "go.temporal.io/server/tools/umpire3/execution"
	"go.temporal.io/server/tools/umpire3/execution/observation"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
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

type workflowTaskTransport interface {
	Enqueue(context.Context) (WorkflowTaskIdentity, error)
	Deliver(context.Context, WorkflowTaskIdentity) (WorkflowTaskDelivery, error)
	Acknowledge(context.Context, WorkflowTaskDelivery) (WorkflowTaskAcknowledgement, error)
	Cleanup(context.Context, WorkflowTaskIdentity) error
}

type taskAckFactory struct {
	probe     clusterProbe
	transport workflowTaskTransport
}

func newTaskAckFactory(probe clusterProbe, transport workflowTaskTransport) *taskAckFactory {
	return &taskAckFactory{probe: probe, transport: transport}
}

func (f *taskAckFactory) Capabilities() []protocolcatalog.CapabilityID {
	return []protocolcatalog.CapabilityID{protocolcatalog.CapabilityIDWorkflowTaskControl}
}

func (f *taskAckFactory) Prepare(ctx context.Context, experiment protocolexperiment.Experiment) (environment.PreparedEnvironment, error) {
	if f.probe == nil || f.transport == nil {
		return environment.PreparedEnvironment{}, errors.New("temporal cluster probe and Workflow Task transport are required")
	}
	cluster, err := f.probe(ctx)
	if err != nil {
		return environment.PreparedEnvironment{}, fmt.Errorf("probe Temporal cluster: %w", err)
	}
	if cluster.BuildID == "" || cluster.Namespace == "" {
		return environment.PreparedEnvironment{}, errors.New("cluster probe returned incomplete Workflow Task identity evidence")
	}
	session := &taskAckSession{cluster: cluster, experimentID: experiment.ExperimentID, transport: f.transport}
	return environment.PreparedEnvironment{Session: session, Identity: session.environmentIdentity(f.Capabilities())}, nil
}

type taskAckSession struct {
	mu sync.Mutex

	cluster         clusterInfo
	experimentID    string
	transport       workflowTaskTransport
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
	action protocolexperiment.Action,
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
		Source: source, Outcome: protocolexperiment.ActionOutcomeApplied,
		SourceIdentity: source, ClockDomain: source + "-sequence", SourceSequence: s.sequence,
		Reference: reference, EntityIdentity: s.identity.WorkflowID,
		Lineage: []string{s.cluster.Namespace, s.identity.WorkflowID, s.identity.RunID},
	}, nil
}

func (s *taskAckSession) ObserveFacts(
	ctx context.Context,
	checkpoint protocolexperiment.Checkpoint,
	_ environment.Bindings,
) ([]observation.Fact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if checkpoint.Observation != "workflow-task-acknowledged" {
		return nil, fmt.Errorf("unsupported Workflow Task observation %q", checkpoint.Observation)
	}
	if !s.acknowledged {
		return nil, environment.ErrObservationUnavailable
	}
	s.sequence++
	receiptSequence := s.sequence
	outcome := "backlog-present"
	if s.acknowledgement.BacklogAbsent {
		outcome = "backlog-absent"
	}
	s.sequence++
	windowSequence := s.sequence
	factSource := func(sequence int64, reference string) observation.Source {
		return observation.Source{
			Identity: s.acknowledgement.Source, ClockDomain: s.acknowledgement.Source + "-sequence",
			Sequence: sequence, Reference: reference,
			CausalReferences: []string{s.delivery.Reference, s.acknowledgement.Reference},
			EntityIdentity:   s.identity.WorkflowID,
			Lineage:          []string{s.cluster.Namespace, s.identity.WorkflowID, s.identity.RunID},
		}
	}
	return []observation.Fact{
		{
			Identifier: "mechanism/" + observation.WorkflowTaskAcknowledged,
			Source:     factSource(receiptSequence, s.acknowledgement.Reference),
			Mechanism: &observation.MechanismReceipt{
				Action: observation.WorkflowTaskAcknowledged, Resource: s.identity.WorkflowID,
				Attempt: receiptSequence, OwnerEpoch: 0, Outcome: outcome,
			},
		},
		{
			Identifier: "window/" + observation.WorkflowTaskAcknowledged,
			Source: factSource(windowSequence,
				s.acknowledgement.Reference+"/window/"+checkpoint.Identifier),
			Window: &observation.EvidenceWindow{
				Purpose: observation.WorkflowTaskAcknowledged, Closed: true,
				ThroughSequence: windowSequence,
			},
		},
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

func (s *taskAckSession) environmentIdentity(capabilities []protocolcatalog.CapabilityID) environment.EnvironmentIdentity {
	configurationIdentity := s.cluster.ConfigurationID
	if configurationIdentity == "" {
		configurationIdentity = s.cluster.BuildID + "/default"
	}
	return environment.EnvironmentIdentity{
		Name: "ci-test-cluster", BuildID: s.cluster.BuildID,
		ConfigurationIdentity: configurationIdentity,
		EvidenceProfile:       environment.EvidenceProfilePublicGRPCHistory,
		DrivingAuthority:      "public-workflow-task-api",
		ObservationAuthority:  "public-workflow-task-response",
		FaultAuthority:        "none", IsolationIdentity: s.cluster.Namespace,
		RetentionClass: "semantic-redacted", Capabilities: append([]protocolcatalog.CapabilityID(nil), capabilities...),
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
