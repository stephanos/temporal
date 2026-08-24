package temporal

import (
	"context"
	"errors"
	"fmt"
	"sync"

	environment "go.temporal.io/server/tools/umpire3/execution"
	umpire3fault "go.temporal.io/server/tools/umpire3/execution/fault"
	"go.temporal.io/server/tools/umpire3/execution/observation"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

type clusterInfo struct {
	BuildID           string
	ConfigurationID   string
	EvidenceProfile   string
	Namespace         string
	MintedOperationID string
	MintedWorkflowID  string
	MintedUpdateID    string
}

type clusterProbe func(context.Context) (clusterInfo, error)

type nexusOptions struct {
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

type nexusFactory struct {
	probe   clusterProbe
	options nexusOptions
}

func newNexusFactory(probe clusterProbe, options nexusOptions) *nexusFactory {
	return &nexusFactory{probe: probe, options: options}
}

func (f *nexusFactory) Capabilities() []protocolcatalog.CapabilityID {
	return []protocolcatalog.CapabilityID{
		protocolcatalog.CapabilityIDNexus, protocolcatalog.CapabilityIDNexusWorkerControl,
		protocolcatalog.CapabilityIDNexusObservation, protocolcatalog.CapabilityIDFailoverControl,
	}
}

func (f *nexusFactory) Prepare(ctx context.Context, experiment protocolexperiment.Experiment) (environment.PreparedEnvironment, error) {
	if f.probe == nil {
		return environment.PreparedEnvironment{}, errors.New("temporal cluster probe is required")
	}
	cluster, err := f.probe(ctx)
	if err != nil {
		return environment.PreparedEnvironment{}, fmt.Errorf("probe Temporal cluster: %w", err)
	}
	if cluster.BuildID == "" || cluster.Namespace == "" ||
		(cluster.MintedOperationID == "" && f.options.TaskTransport == nil) {
		return environment.PreparedEnvironment{}, errors.New("cluster probe returned incomplete identity evidence")
	}
	session := &nexusSession{
		cluster:      cluster,
		options:      f.options,
		transport:    f.options.TaskTransport,
		experimentID: experiment.ExperimentID,
		ownerEpoch:   0,
		workerEpoch:  -1,
		staleEpoch:   -1,
		returnEpoch:  -1,
	}
	return environment.PreparedEnvironment{Session: session, Identity: session.environmentIdentity(f.Capabilities())}, nil
}

type nexusSession struct {
	mu sync.Mutex

	cluster              clusterInfo
	options              nexusOptions
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
	faultInstalled       bool
	faultActive          bool
	faultFired           bool
	faultHandle          string
	facts                []observation.Fact
}

func (s *nexusSession) Realize(ctx context.Context, action protocolexperiment.Action, bindings environment.Bindings) (environment.ActionEvidence, error) {
	if err := ctx.Err(); err != nil {
		return environment.ActionEvidence{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	referenceID := s.cluster.MintedOperationID
	if referenceID == "" {
		referenceID = s.experimentID
	}
	if err := validateIdentityArgument(action, "operation", referenceID, bindings); err != nil {
		return environment.ActionEvidence{}, err
	}
	evidence := environment.ActionEvidence{
		Source:         "temporal-test-cluster",
		Outcome:        protocolexperiment.ActionOutcomeApplied,
		SourceIdentity: "temporal-test-cluster",
		ClockDomain:    "temporal-test-cluster-sequence",
		Reference:      s.cluster.Namespace + "/" + referenceID + "/" + action.Identifier,
		EntityIdentity: referenceID,
		Lineage:        []string{s.cluster.Namespace, referenceID},
	}
	switch action.Kind {
	case "schedule-operation":
		if s.scheduled {
			return environment.ActionEvidence{}, errors.New("operation already scheduled")
		}
		s.scheduled = true
		if s.cluster.MintedOperationID != "" {
			evidence.GroundedBindings = map[string]string{"operation": s.cluster.MintedOperationID}
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
		s.appendCancellationFact(observation.NexusCancellationAccepted)
	case "commit-cancellation":
		if !s.cancellationAccepted {
			return environment.ActionEvidence{}, errors.New("cancellation was not accepted")
		}
		s.cancelled = true
		s.appendCancellationFact(observation.NexusCancellationCommitted)
	case "acquire-ownership":
		if s.dispatched && s.faultActive {
			s.staleEpoch = s.workerEpoch
			s.faultFired = true
		}
		s.ownerEpoch++
		s.appendOwnershipFact()
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
		visible := (s.transport != nil && s.completionVisible) ||
			(s.transport == nil && s.options.AllowStaleSuccess)
		if stale && visible {
			s.staleVisible = true
			s.appendSuccessFact()
		} else if stale {
			evidence.Outcome = protocolexperiment.ActionOutcomeSuppressed
		}
		s.appendClosedWindow()
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
	grounded, err := groundActionBindings(action, map[string]string{
		"operation-id": s.cluster.MintedOperationID,
	})
	if err != nil {
		return environment.ActionEvidence{}, err
	}
	if len(grounded) != 0 && evidence.GroundedBindings == nil {
		evidence.GroundedBindings = make(map[string]string, len(grounded))
	}
	for symbol, concrete := range grounded {
		evidence.GroundedBindings[symbol] = concrete
	}
	evidence.SourceIdentity = evidence.Source
	evidence.ClockDomain = evidence.Source + "-sequence"
	return evidence, nil
}

type nexusFaultRealizer struct {
	session *nexusSession
}

func (s *nexusSession) FaultRealizer() umpire3fault.Realizer {
	return nexusFaultRealizer{session: s}
}

func (r nexusFaultRealizer) Install(_ context.Context, term umpire3fault.Term) (string, error) {
	s := r.session
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.faultInstalled {
		return "", errors.New("nexus fault is already installed")
	}
	if term.Kind != protocolcatalog.FaultKindStaleWorkerCompletion {
		return "", fmt.Errorf("nexus adapter cannot realize fault %q", term.Kind)
	}
	s.faultInstalled = true
	s.faultHandle = "nexus-stale-completion/" + s.experimentID
	return s.faultHandle, nil
}

func (r nexusFaultRealizer) Activate(_ context.Context, handle string) error {
	s := r.session
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.faultInstalled || handle != s.faultHandle || s.faultActive {
		return errors.New("nexus fault handle is not activatable")
	}
	s.faultActive = true
	return nil
}

func (r nexusFaultRealizer) Release(_ context.Context, handle string) error {
	s := r.session
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.faultInstalled || handle != s.faultHandle {
		return errors.New("unknown nexus fault handle")
	}
	s.faultActive = false
	return nil
}

func (r nexusFaultRealizer) Cleanup(_ context.Context, handle string) error {
	s := r.session
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.faultInstalled || handle != s.faultHandle {
		return errors.New("unknown nexus fault handle")
	}
	s.faultActive = false
	s.faultInstalled = false
	return nil
}

func (r nexusFaultRealizer) RealizationEvidence(
	_ context.Context,
	handle string,
) (umpire3fault.RealizationEvidence, error) {
	s := r.session
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.faultInstalled || handle != s.faultHandle || !s.faultFired {
		return umpire3fault.RealizationEvidence{}, errors.New("stale completion fault did not fire")
	}
	operationID := s.cluster.MintedOperationID
	if operationID == "" {
		operationID = s.experimentID
	}
	return umpire3fault.RealizationEvidence{
		SourceIdentity: "umpire3-controlled-nexus-state",
		Reference:      s.cluster.Namespace + "/" + operationID + "/fault/stale-completion",
		EntityIdentity: operationID,
	}, nil
}

func (s *nexusSession) ObserveFacts(
	ctx context.Context,
	checkpoint protocolexperiment.Checkpoint,
	_ environment.Bindings,
) ([]observation.Fact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	switch checkpoint.Observation {
	case "cancellation-accepted", "cancellation-won", "stale-success-absent":
	default:
		return nil, fmt.Errorf("unsupported observation %q", checkpoint.Observation)
	}
	return append([]observation.Fact(nil), s.facts...), nil
}

func (s *nexusSession) appendCancellationFact(eventType string) {
	sequence := s.nextFactSequence()
	operationID := s.operationID()
	s.facts = append(s.facts, observation.Fact{
		Identifier: fmt.Sprintf("history/%s/%d", eventType, sequence),
		Source: s.factSource(
			"umpire3-controlled-nexus-history", sequence,
			fmt.Sprintf("%s/%s/history/%d", s.cluster.Namespace, operationID, sequence),
		),
		History: &observation.HistoryEvent{
			EventType: eventType, EventID: sequence, OperationID: operationID,
		},
	})
}

func (s *nexusSession) appendOwnershipFact() {
	sequence := s.nextFactSequence()
	operationID := s.operationID()
	s.facts = append(s.facts, observation.Fact{
		Identifier: fmt.Sprintf("mechanism/%s/%d", observation.NexusOwnershipAcquired, sequence),
		Source: s.factSource(
			"umpire3-controlled-nexus-mechanism", sequence,
			fmt.Sprintf("%s/%s/mechanism/%d", s.cluster.Namespace, operationID, sequence),
		),
		Mechanism: &observation.MechanismReceipt{
			Action: observation.NexusOwnershipAcquired, Resource: operationID,
			Attempt: 1, OwnerEpoch: int64(s.ownerEpoch), Outcome: "acquired",
		},
	})
}

func (s *nexusSession) appendSuccessFact() {
	sequence := s.nextFactSequence()
	operationID := s.operationID()
	sourceIdentity := s.completionSource
	if sourceIdentity == "" {
		sourceIdentity = "umpire3-controlled-nexus-history"
	}
	reference := s.completionReference
	if reference == "" {
		reference = fmt.Sprintf("%s/%s/history/%d", s.cluster.Namespace, operationID, sequence)
	}
	cancellationCommitted := s.cancelled
	ownerEpoch := int64(s.returnEpoch)
	currentOwnerEpoch := int64(s.ownerEpoch)
	s.facts = append(s.facts, observation.Fact{
		Identifier: fmt.Sprintf("history/%s/%d", observation.NexusSuccessRecorded, sequence),
		Source:     s.factSource(sourceIdentity, sequence, reference),
		History: &observation.HistoryEvent{
			EventType: observation.NexusSuccessRecorded, EventID: sequence, OperationID: operationID,
			OwnerEpoch: &ownerEpoch, CurrentOwnerEpoch: &currentOwnerEpoch,
			CancellationCommitted: &cancellationCommitted,
		},
	})
}

func (s *nexusSession) appendClosedWindow() {
	sequence := s.nextFactSequence()
	operationID := s.operationID()
	sourceIdentity := s.completionSource
	if sourceIdentity == "" {
		sourceIdentity = "umpire3-controlled-nexus-history"
	}
	reference := s.completionReference
	if reference == "" {
		reference = fmt.Sprintf("%s/%s/window/%d", s.cluster.Namespace, operationID, sequence)
	} else {
		reference = fmt.Sprintf("%s/window/%d", reference, sequence)
	}
	s.facts = append(s.facts, observation.Fact{
		Identifier: fmt.Sprintf("window/%s/%d", observation.NexusCancellationWindow, sequence),
		Source:     s.factSource(sourceIdentity, sequence, reference),
		Window: &observation.EvidenceWindow{
			Purpose: observation.NexusCancellationWindow, Closed: true, ThroughSequence: sequence,
		},
	})
}

func (s *nexusSession) nextFactSequence() int64 {
	s.sequence++
	return s.sequence
}

func (s *nexusSession) operationID() string {
	if s.cluster.MintedOperationID != "" {
		return s.cluster.MintedOperationID
	}
	return s.experimentID
}

func (s *nexusSession) factSource(identity string, sequence int64, reference string) observation.Source {
	operationID := s.operationID()
	return observation.Source{
		Identity: identity, ClockDomain: identity + "-sequence", Sequence: sequence,
		Reference: reference, CausalReferences: []string{s.cluster.Namespace + "/" + operationID},
		EntityIdentity: operationID, Lineage: []string{s.cluster.Namespace, operationID},
	}
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

func (s *nexusSession) environmentIdentity(capabilities []protocolcatalog.CapabilityID) environment.EnvironmentIdentity {
	profile := environment.EvidenceProfileInProcessHooks
	observationAuthority := "controlled-state-hooks"
	if s.transport != nil {
		profile = environment.EvidenceProfilePublicGRPCHistory
		observationAuthority = "public-api-and-task-protocol"
	}
	name := s.options.ProfileName
	if name == "" {
		name = "controlled-local"
	}
	configurationIdentity := s.cluster.ConfigurationID
	if configurationIdentity == "" {
		configurationIdentity = s.cluster.BuildID + "/default"
	}
	return environment.EnvironmentIdentity{
		Name:                  name,
		BuildID:               s.cluster.BuildID,
		ConfigurationIdentity: configurationIdentity,
		EvidenceProfile:       profile,
		DrivingAuthority:      "temporal-api",
		ObservationAuthority:  observationAuthority,
		FaultAuthority:        "controlled-stale-completion",
		IsolationIdentity:     s.cluster.Namespace,
		RetentionClass:        "semantic-only",
		Capabilities:          append([]protocolcatalog.CapabilityID(nil), capabilities...),
	}
}
