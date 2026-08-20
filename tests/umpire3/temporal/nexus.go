package temporal

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.temporal.io/server/tests/umpire3/environment"
	"go.temporal.io/server/tests/umpire3/protocol"
)

type ClusterInfo struct {
	BuildID           string
	ConfigurationID   string
	EvidenceProfile   string
	Namespace         string
	MintedOperationID string
	MintedWorkflowID  string
	MintedUpdateID    string
}

type ClusterProbe func(context.Context) (ClusterInfo, error)

type NexusOptions struct {
	AllowStaleSuccess bool
	ProfileName       string
	TaskTransport     NexusTaskTransport
}

type NexusTask struct {
	OperationID string
	Source      string
	Reference   string
}

type NexusTaskCompletion struct {
	ReportSuccess bool
}

type NexusTaskOutcome struct {
	SuccessVisible bool
	Source         string
	Reference      string
}

type NexusTaskTransport interface {
	Dispatch(context.Context) (NexusTask, error)
	Complete(context.Context, NexusTaskCompletion) (NexusTaskOutcome, error)
	Cleanup(context.Context) error
}

type NexusFactory struct {
	probe   ClusterProbe
	options NexusOptions
}

func NewNexusFactory(probe ClusterProbe, options NexusOptions) *NexusFactory {
	return &NexusFactory{probe: probe, options: options}
}

func (f *NexusFactory) Capabilities() []string {
	return []string{"nexus", "nexus-worker-control", "nexus-observation", "failover-control"}
}

func (f *NexusFactory) Prepare(ctx context.Context, experiment protocol.Experiment) (environment.Session, error) {
	if f.probe == nil {
		return nil, errors.New("temporal cluster probe is required")
	}
	cluster, err := f.probe(ctx)
	if err != nil {
		return nil, fmt.Errorf("probe Temporal cluster: %w", err)
	}
	if cluster.BuildID == "" || cluster.Namespace == "" ||
		(cluster.MintedOperationID == "" && f.options.TaskTransport == nil) {
		return nil, errors.New("cluster probe returned incomplete identity evidence")
	}
	return &nexusSession{
		cluster:      cluster,
		options:      f.options,
		transport:    f.options.TaskTransport,
		experimentID: experiment.ExperimentID,
		ownerEpoch:   0,
		workerEpoch:  -1,
		staleEpoch:   -1,
		returnEpoch:  -1,
	}, nil
}

type nexusSession struct {
	mu sync.Mutex

	cluster              ClusterInfo
	options              NexusOptions
	transport            NexusTaskTransport
	experimentID         string
	scheduled            bool
	dispatched           bool
	cancellationAccepted bool
	cancelled            bool
	ownerEpoch           int
	workerEpoch          int
	staleEpoch           int
	returnEpoch          int
	completionVisible    bool
	completionSource     string
	completionReference  string
	staleVisible         bool
	sequence             int64
	cleaned              bool
}

func (s *nexusSession) Realize(ctx context.Context, action protocol.Action, _ environment.Bindings) (environment.ActionEvidence, error) {
	if err := ctx.Err(); err != nil {
		return environment.ActionEvidence{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	referenceID := s.cluster.MintedOperationID
	if referenceID == "" {
		referenceID = s.experimentID
	}
	evidence := environment.ActionEvidence{
		Source:    "temporal-test-cluster",
		Reference: s.cluster.Namespace + "/" + referenceID + "/" + action.Identifier,
	}
	switch action.Kind {
	case "schedule-operation":
		if s.scheduled {
			return environment.ActionEvidence{}, errors.New("operation already scheduled")
		}
		s.scheduled = true
		if s.cluster.MintedOperationID != "" {
			evidence.GroundedBindings = map[string]string{
				"operation": s.cluster.MintedOperationID,
			}
		}
	case "dispatch-task":
		if !s.scheduled || s.dispatched {
			return environment.ActionEvidence{}, errors.New("operation is not dispatchable")
		}
		s.dispatched = true
		s.workerEpoch = s.ownerEpoch
		if s.transport != nil {
			task, err := s.transport.Dispatch(ctx)
			if err != nil {
				return environment.ActionEvidence{}, fmt.Errorf("dispatch Nexus task: %w", err)
			}
			if task.OperationID == "" || task.Source == "" || task.Reference == "" {
				return environment.ActionEvidence{}, errors.New("nexus task transport returned incomplete identity evidence")
			}
			s.cluster.MintedOperationID = task.OperationID
			evidence.Source = task.Source
			evidence.Reference = task.Reference
			evidence.GroundedBindings = map[string]string{"operation": task.OperationID}
		}
	case "request-cancellation":
		if !s.dispatched || s.cancelled {
			return environment.ActionEvidence{}, errors.New("cancellation cannot be requested")
		}
		s.cancellationAccepted = true
	case "commit-cancellation":
		if !s.cancellationAccepted {
			return environment.ActionEvidence{}, errors.New("cancellation was not accepted")
		}
		s.cancelled = true
	case "acquire-ownership":
		if s.dispatched {
			s.staleEpoch = s.workerEpoch
		}
		s.ownerEpoch++
	case "retry-task":
		if !s.scheduled {
			return environment.ActionEvidence{}, errors.New("operation is not retryable")
		}
		s.dispatched = true
		s.workerEpoch = s.ownerEpoch
	case "worker-returns-success":
		if !s.dispatched {
			return environment.ActionEvidence{}, errors.New("no dispatched worker")
		}
		if s.staleEpoch >= 0 {
			s.returnEpoch = s.staleEpoch
			s.staleEpoch = -1
		} else {
			s.returnEpoch = s.workerEpoch
		}
		if s.transport != nil {
			stale := s.returnEpoch != s.ownerEpoch || s.cancelled
			outcome, err := s.transport.Complete(ctx, NexusTaskCompletion{
				ReportSuccess: !stale || s.options.AllowStaleSuccess,
			})
			if err != nil {
				return environment.ActionEvidence{}, fmt.Errorf("complete Nexus task: %w", err)
			}
			if outcome.Source == "" || outcome.Reference == "" {
				return environment.ActionEvidence{}, errors.New("nexus task transport returned incomplete completion evidence")
			}
			s.completionVisible = outcome.SuccessVisible
			s.completionSource = outcome.Source
			s.completionReference = outcome.Reference
			evidence.Source = outcome.Source
			evidence.Reference = outcome.Reference
		}
	case "persist-success":
		if s.returnEpoch < 0 {
			return environment.ActionEvidence{}, errors.New("no worker completion to persist")
		}
		stale := s.returnEpoch != s.ownerEpoch || s.cancelled
		if stale && ((s.transport != nil && s.completionVisible) ||
			(s.transport == nil && s.options.AllowStaleSuccess)) {
			s.staleVisible = true
		}
		if s.transport != nil {
			evidence.Source = s.completionSource
			evidence.Reference = s.completionReference
		}
	case "crash-owner", "recover-owner", "ack-task":
	case "start-update", "accept-update", "complete-update", "record-update-history",
		"dispatch-workflow-task", "complete-workflow-task":
		return environment.ActionEvidence{}, fmt.Errorf("action %q belongs to the Update adapter", action.Kind)
	default:
		return environment.ActionEvidence{}, fmt.Errorf("unsupported action %q", action.Kind)
	}
	return evidence, nil
}

func (s *nexusSession) Observe(ctx context.Context, checkpoint protocol.Checkpoint, _ environment.Bindings) (environment.Observation, error) {
	if err := ctx.Err(); err != nil {
		return environment.Observation{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sequence++
	observation := environment.Observation{
		CheckpointID:    checkpoint.Identifier,
		Kind:            checkpoint.Observation,
		Source:          "umpire3-controlled-nexus-state",
		SourceSequence:  s.sequence,
		CausalReference: s.cluster.Namespace + "/" + s.cluster.MintedOperationID,
	}
	switch checkpoint.Observation {
	case "cancellation-accepted":
		observation.Satisfied = s.cancellationAccepted
	case "cancellation-won":
		observation.Satisfied = s.cancelled
	case "stale-success-absent":
		observation.Satisfied = !s.staleVisible
		if s.completionSource != "" {
			observation.Source = s.completionSource
			observation.CausalReference = s.completionReference
		}
	default:
		return environment.Observation{}, fmt.Errorf("unsupported observation %q", checkpoint.Observation)
	}
	return observation, nil
}

func (s *nexusSession) Cleanup(ctx context.Context) environment.CleanupResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return environment.CleanupResult{
			Error:                err.Error(),
			RecoverableResources: s.recoveryMetadata(),
		}
	}
	if s.transport != nil {
		if err := s.transport.Cleanup(ctx); err != nil {
			return environment.CleanupResult{
				Error:                err.Error(),
				RecoverableResources: s.recoveryMetadata(),
			}
		}
	}
	s.cleaned = true
	return environment.CleanupResult{Complete: true}
}

func (s *nexusSession) RecoveryMetadata() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.recoveryMetadata()
}

func (s *nexusSession) recoveryMetadata() map[string]string {
	return map[string]string{
		"experimentID": s.experimentID,
		"namespace":    s.cluster.Namespace,
		"operation":    s.cluster.MintedOperationID,
	}
}

func (s *nexusSession) Profile() environment.Profile {
	profile := s.cluster.EvidenceProfile
	if profile == "" {
		profile = "controlled"
	}
	name := s.options.ProfileName
	if name == "" {
		name = "controlled-local"
	}
	return environment.Profile{
		Name:                  name,
		BuildID:               s.cluster.BuildID,
		ConfigurationIdentity: s.cluster.ConfigurationID,
		EvidenceProfile:       profile,
	}
}
