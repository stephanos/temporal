package temporal

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	sdkclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/server/tests/umpire3/deployment"
	environment "go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/execution/observation"
	"go.temporal.io/server/tests/umpire3/execution/participant"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

type SDKFactoryOptions struct {
	Client               sdkclient.Client
	Registry             worker.Registry
	Deployment           deployment.Profile
	Namespace            string
	TaskQueue            string
	WorkflowID           func(protocolexperiment.Experiment) string
	CleanupTimeout       time.Duration
	NegativeControl      bool
	NexusEndpoint        string
	NexusService         string
	NexusOperation       string
	CorroboratingHistory []CorroboratingHistorySource
	WorkflowTaskFencer   WorkflowTaskFencer
	CallbackDriver       CallbackDriver
	NexusDriver          NexusDriver
}

type HistoryRequest struct {
	Namespace  string
	WorkflowID string
	RunID      string
}

type CorroboratingHistoryEvent struct {
	Type                       enumspb.EventType
	ID                         int64
	TimeUnixNano               int64
	Reference                  string
	ContinuedExecutionRunID    string
	OriginalExecutionRunID     string
	FirstExecutionRunID        string
	TaskQueue                  string
	CallbackRegistered         bool
	NexusActivityForwardLinked bool
	NexusActivityReverseLinked bool
	NexusTimeoutType           enumspb.TimeoutType
	NexusTimeoutMessage        string
}

type CorroboratingHistory struct {
	Source         string
	SourceIdentity string
	ClockDomain    string
	Events         []CorroboratingHistoryEvent
}

type CorroboratingHistorySource interface {
	ReadHistory(context.Context, HistoryRequest) (CorroboratingHistory, error)
}

type SDKFactory struct {
	options  SDKFactoryOptions
	identity environment.EnvironmentIdentity
}

func NewSDKFactory(options SDKFactoryOptions) (*SDKFactory, error) {
	if options.Client == nil || options.Registry == nil || options.Namespace == "" ||
		options.TaskQueue == "" || options.WorkflowID == nil {
		return nil, errors.New("SDK environment requires client, registry, namespace, task queue, build, and workflow identity")
	}
	if options.CleanupTimeout <= 0 {
		return nil, errors.New("SDK environment requires a positive cleanup timeout")
	}
	identity := options.Deployment.Environment
	if err := identity.Validate(); err != nil {
		return nil, fmt.Errorf("validate SDK deployment profile: %w", err)
	}
	if options.Deployment.Namespace != options.Namespace || options.Deployment.TaskQueue != options.TaskQueue {
		return nil, errors.New("SDK namespace and task queue must match the deployment profile")
	}
	identity.FaultAuthority = "none"
	for _, source := range options.CorroboratingHistory {
		if source == nil {
			return nil, errors.New("SDK environment has a nil corroborating history source")
		}
	}
	if len(options.CorroboratingHistory) != 0 {
		identity.EvidenceProfile = environment.EvidenceProfileDualHistory
		identity.ObservationAuthority = "temporal-public-history+temporal-history-service"
	}
	capabilities, err := normalizeSDKCapabilities(options)
	if err != nil {
		return nil, err
	}
	identity.Capabilities = capabilities
	if err := identity.Validate(); err != nil {
		return nil, fmt.Errorf("validate SDK environment profile: %w", err)
	}
	return &SDKFactory{options: options, identity: identity}, nil
}

func (f *SDKFactory) Capabilities() []protocolcatalog.CapabilityID {
	return slices.Clone(f.identity.Capabilities)
}

func normalizeSDKCapabilities(options SDKFactoryOptions) ([]protocolcatalog.CapabilityID, error) {
	catalog, err := protocolcatalog.DefaultCatalog()
	if err != nil {
		return nil, err
	}
	known := make(map[protocolcatalog.CapabilityID]struct{}, len(catalog.Capabilities))
	for _, capability := range catalog.Capabilities {
		known[capability.Identifier] = struct{}{}
	}
	capabilities := []protocolcatalog.CapabilityID{
		protocolcatalog.CapabilityIDHistoryObservation, protocolcatalog.CapabilityIDUpdate,
		protocolcatalog.CapabilityIDWorkflowTaskControl,
	}
	if options.NexusEndpoint != "" && options.NexusService != "" && options.NexusOperation != "" {
		capabilities = append(capabilities, protocolcatalog.CapabilityIDNexus,
			protocolcatalog.CapabilityIDNexusObservation, protocolcatalog.CapabilityIDNexusWorkerControl)
	}
	slices.Sort(capabilities)
	for index, capability := range capabilities {
		if _, exists := known[capability]; !exists {
			return nil, fmt.Errorf("SDK environment has unknown capability %q", capability)
		}
		if index > 0 && capabilities[index-1] == capability {
			return nil, fmt.Errorf("SDK environment has duplicate capability %q", capability)
		}
	}
	return capabilities, nil
}

func (f *SDKFactory) Prepare(ctx context.Context, experiment protocolexperiment.Experiment) (environment.PreparedEnvironment, error) {
	program, _, err := participant.CompileExperiment(experiment)
	if err != nil {
		return environment.PreparedEnvironment{}, fmt.Errorf("compile SDK participant experiment: %w", err)
	}
	if f.options.NegativeControl {
		for index := range program.Commands {
			program.Commands[index].Response = participant.ResponseFailure
		}
	}
	workflowID := f.options.WorkflowID(experiment)
	if workflowID == "" {
		return environment.PreparedEnvironment{}, errors.New("SDK environment returned an empty workflow identity")
	}
	runner, err := NewSDKParticipantAdapter(SDKParticipantOptions{
		Client: f.options.Client, Registry: f.options.Registry, TaskQueue: f.options.TaskQueue,
		Namespace: f.options.Namespace, WorkflowID: workflowID, CleanupTimeout: f.options.CleanupTimeout,
		NexusEndpoint: f.options.NexusEndpoint, NexusService: f.options.NexusService,
		NexusOperation:     f.options.NexusOperation,
		WorkflowTaskFencer: f.options.WorkflowTaskFencer,
		CallbackDriver:     f.options.CallbackDriver,
		NexusDriver:        f.options.NexusDriver,
	})
	if err != nil {
		return environment.PreparedEnvironment{}, err
	}
	session, err := participant.Start(ctx, program, runner)
	if err != nil {
		return environment.PreparedEnvironment{}, err
	}
	sdk := &sdkSession{
		experiment: experiment, participant: session, client: f.options.Client,
		namespace: f.options.Namespace, taskQueue: f.options.TaskQueue,
		identityValue: f.identity,
		results:       make(map[string]participant.Result),
	}
	if len(f.options.CorroboratingHistory) != 0 {
		session := &corroboratingSDKSession{
			sdkSession: sdk, sources: slices.Clone(f.options.CorroboratingHistory),
		}
		return environment.PreparedEnvironment{Session: session, Identity: sdk.identity(f.Capabilities())}, nil
	}
	return environment.PreparedEnvironment{Session: sdk, Identity: sdk.identity(f.Capabilities())}, nil
}

type sdkSession struct {
	experiment    protocolexperiment.Experiment
	participant   *participant.Session
	client        sdkclient.Client
	namespace     string
	taskQueue     string
	identityValue environment.EnvironmentIdentity
	results       map[string]participant.Result
}

type corroboratingSDKSession struct {
	*sdkSession
	sources []CorroboratingHistorySource
}

func (s *sdkSession) Realize(
	ctx context.Context,
	action protocolexperiment.Action,
	_ environment.Bindings,
) (environment.ActionEvidence, error) {
	result, err := s.participant.Execute(ctx, action.Identifier)
	if err != nil {
		return environment.ActionEvidence{}, err
	}
	s.results[action.Identifier] = result
	history, err := s.latestHistory(ctx, result.WorkflowID, result.RunID)
	if err != nil {
		return environment.ActionEvidence{}, err
	}
	grounded := make(map[string]string, len(action.Bindings))
	for _, binding := range action.Bindings {
		grounded[binding.Symbol] = result.WorkflowID + "/" + result.RunID + "/" + binding.Symbol
	}
	return environment.ActionEvidence{
		Source: "temporal-public-history", Outcome: participantActionOutcome(result.Status),
		SourceIdentity: s.namespace,
		ClockDomain:    "temporal-history-event-id", SourceSequence: history.sequence,
		Reference: history.reference, CausalReferences: []string{result.Reference},
		EntityIdentity: result.WorkflowID + "/" + result.RunID,
		Lineage:        append([]string{s.experiment.ExperimentID}, result.Lineage...),
		PayloadDigest:  result.PayloadDigest, GroundedBindings: grounded,
		TerminalState: result.TerminalState, TerminalDisposition: result.TerminalDisposition,
	}, nil
}

func participantActionOutcome(status string) protocolexperiment.ActionOutcome {
	switch status {
	case "completed", "accepted", "deferred":
		return protocolexperiment.ActionOutcomeApplied
	case "suppressed":
		return protocolexperiment.ActionOutcomeSuppressed
	case "failed":
		return protocolexperiment.ActionOutcomeRejected
	default:
		return ""
	}
}

func (s *sdkSession) ObserveFacts(
	ctx context.Context,
	checkpoint protocolexperiment.Checkpoint,
	_ environment.Bindings,
) ([]observation.Fact, error) {
	if len(s.results) == 0 {
		return nil, errors.New("no SDK participant result is available")
	}
	latest := s.latestResult()
	if latest.WorkflowID == "" || latest.RunID == "" {
		return nil, errors.New("SDK participant identity evidence is incomplete")
	}
	history, err := s.latestHistory(ctx, latest.WorkflowID, latest.RunID)
	if err != nil {
		return nil, err
	}
	return (sdkFactNormalizer{
		experiment: s.experiment,
		namespace:  s.namespace,
		taskQueue:  s.taskQueue,
		results:    s.results,
	}).Normalize(checkpoint, history)
}

func (s *corroboratingSDKSession) CorroborateFacts(
	ctx context.Context,
	checkpoint protocolexperiment.Checkpoint,
	_ environment.Bindings,
) ([][]observation.Fact, error) {
	latest := s.latestResult()
	if latest.WorkflowID == "" || latest.RunID == "" {
		return nil, errors.New("SDK participant identity evidence is incomplete")
	}
	factSets := make([][]observation.Fact, 0, len(s.sources))
	for _, source := range s.sources {
		history, err := source.ReadHistory(ctx, HistoryRequest{
			Namespace: s.namespace, WorkflowID: latest.WorkflowID, RunID: latest.RunID,
		})
		if err != nil {
			return nil, fmt.Errorf("read corroborating history: %w", err)
		}
		position, err := history.position()
		if err != nil {
			return nil, err
		}
		facts, err := (sdkFactNormalizer{
			experiment: s.experiment,
			namespace:  s.namespace,
			taskQueue:  s.taskQueue,
			results:    s.results,
		}).Normalize(checkpoint, position)
		if err != nil {
			return nil, fmt.Errorf("normalize corroborating history: %w", err)
		}
		factSets = append(factSets, facts)
	}
	return factSets, nil
}

func (s *sdkSession) latestResult() participant.Result {
	var latest participant.Result
	for _, action := range s.experiment.Actions {
		if result, exists := s.results[action.Identifier]; exists {
			latest = result
		}
	}
	return latest
}

func (h CorroboratingHistory) position() (historyPosition, error) {
	if h.Source == "" || h.SourceIdentity == "" || h.ClockDomain == "" || len(h.Events) == 0 {
		return historyPosition{}, errors.New("corroborating history identity or events are incomplete")
	}
	position := historyPosition{
		source: h.Source, sourceIdentity: h.SourceIdentity, clockDomain: h.ClockDomain,
		events:     make(map[enumspb.EventType]historyEventPosition, len(h.Events)),
		taskQueues: make(map[string]bool),
	}
	for _, event := range h.Events {
		if event.ID <= 0 || event.TimeUnixNano <= 0 || event.Reference == "" || event.ID <= position.sequence {
			return historyPosition{}, errors.New("corroborating history event sequence, time, or reference is invalid")
		}
		position.sequence = event.ID
		position.timestamp = event.TimeUnixNano
		position.reference = event.Reference
		position.events[event.Type] = historyEventPosition{
			sequence: event.ID, timestamp: event.TimeUnixNano, reference: event.Reference,
		}
		if event.ContinuedExecutionRunID != "" {
			position.continuedExecutionRunID = event.ContinuedExecutionRunID
		}
		if event.OriginalExecutionRunID != "" {
			position.originalExecutionRunID = event.OriginalExecutionRunID
		}
		if event.FirstExecutionRunID != "" {
			position.firstExecutionRunID = event.FirstExecutionRunID
		}
		if event.TaskQueue != "" {
			position.taskQueues[event.TaskQueue] = true
		}
		position.callbackRegistered = position.callbackRegistered || event.CallbackRegistered
		position.nexusActivityForwardLinked =
			position.nexusActivityForwardLinked || event.NexusActivityForwardLinked
		position.nexusActivityReverseLinked =
			position.nexusActivityReverseLinked || event.NexusActivityReverseLinked
		if event.NexusTimeoutType != enumspb.TIMEOUT_TYPE_UNSPECIFIED {
			position.nexusTimeoutType = event.NexusTimeoutType
			position.nexusTimeoutMessage = event.NexusTimeoutMessage
		}
	}
	return position, nil
}

func (s *sdkSession) Cleanup(ctx context.Context) environment.CleanupResult {
	if err := s.participant.Cleanup(ctx); err != nil {
		return environment.CleanupResult{Error: err.Error(), RecoverableResources: s.RecoveryMetadata()}
	}
	return environment.CleanupResult{Complete: true}
}

func (s *sdkSession) RecoveryMetadata() map[string]string {
	result := map[string]string{
		"experiment": s.experiment.ExperimentID,
		"namespace":  s.namespace,
		"taskQueue":  s.taskQueue,
	}
	for _, receipt := range s.results {
		result["workflow"] = receipt.WorkflowID
		result["run"] = receipt.RunID
		break
	}
	return result
}

func (s *sdkSession) identity(capabilities []protocolcatalog.CapabilityID) environment.EnvironmentIdentity {
	identity := s.identityValue
	identity.Capabilities = append([]protocolcatalog.CapabilityID(nil), capabilities...)
	return identity
}

type historyPosition struct {
	source                     string
	sourceIdentity             string
	clockDomain                string
	sequence                   int64
	timestamp                  int64
	reference                  string
	events                     map[enumspb.EventType]historyEventPosition
	continuedExecutionRunID    string
	originalExecutionRunID     string
	firstExecutionRunID        string
	taskQueues                 map[string]bool
	callbackRegistered         bool
	nexusActivityForwardLinked bool
	nexusActivityReverseLinked bool
	nexusTimeoutType           enumspb.TimeoutType
	nexusTimeoutMessage        string
}

type historyEventPosition struct {
	sequence  int64
	timestamp int64
	reference string
}

func (s *sdkSession) latestHistory(
	ctx context.Context,
	workflowID string,
	runID string,
) (historyPosition, error) {
	iterator := s.client.GetWorkflowHistory(
		ctx, workflowID, runID, false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	latest := historyPosition{
		source: "temporal-public-history", sourceIdentity: s.namespace,
		clockDomain: "temporal-history-event-id",
		events:      make(map[enumspb.EventType]historyEventPosition), taskQueues: make(map[string]bool),
	}
	for iterator.HasNext() {
		event, err := iterator.Next()
		if err != nil {
			return historyPosition{}, fmt.Errorf("read SDK participant history: %w", err)
		}
		latest.sequence = event.GetEventId()
		if event.GetEventTime() != nil {
			latest.timestamp = event.GetEventTime().AsTime().UnixNano()
		}
		latest.reference = fmt.Sprintf("%s/%s/history/%d", workflowID, runID, latest.sequence)
		latest.events[event.GetEventType()] = historyEventPosition{
			sequence: latest.sequence, timestamp: latest.timestamp, reference: latest.reference,
		}
		if started := event.GetWorkflowExecutionStartedEventAttributes(); started != nil {
			latest.continuedExecutionRunID = started.GetContinuedExecutionRunId()
			latest.originalExecutionRunID = started.GetOriginalExecutionRunId()
			latest.firstExecutionRunID = started.GetFirstExecutionRunId()
			latest.callbackRegistered = len(started.GetCompletionCallbacks()) != 0
			if taskQueue := started.GetTaskQueue().GetName(); taskQueue != "" {
				latest.taskQueues[taskQueue] = true
			}
		}
		if scheduled := event.GetWorkflowTaskScheduledEventAttributes(); scheduled != nil {
			if taskQueue := scheduled.GetTaskQueue().GetName(); taskQueue != "" {
				latest.taskQueues[taskQueue] = true
			}
		}
		if nexusStarted := event.GetNexusOperationStartedEventAttributes(); nexusStarted != nil &&
			nexusStarted.GetOperationToken() != "" {
			latest.callbackRegistered = true
		}
		for _, link := range event.GetLinks() {
			if activity := link.GetActivity(); activity != nil && activity.GetNamespace() != "" &&
				activity.GetActivityId() != "" && activity.GetRunId() != "" {
				latest.nexusActivityForwardLinked = true
			}
			if operation := link.GetNexusOperation(); operation != nil && operation.GetNamespace() != "" &&
				operation.GetOperationId() != "" && operation.GetRunId() != "" {
				latest.nexusActivityReverseLinked = true
			}
		}
		if timedOut := event.GetNexusOperationTimedOutEventAttributes(); timedOut != nil {
			cause := timedOut.GetFailure().GetCause()
			latest.nexusTimeoutType = cause.GetTimeoutFailureInfo().GetTimeoutType()
			latest.nexusTimeoutMessage = cause.GetMessage()
		}
	}
	if latest.sequence == 0 || latest.timestamp == 0 || latest.reference == "" {
		return historyPosition{}, errors.New("SDK participant history evidence is incomplete")
	}
	return latest, nil
}

func parseWorkflowOwnerFencingReference(reference string) (staleStarted int64, currentStarted int64, valid bool) {
	const marker = "/workflow-task/"
	index := strings.LastIndex(reference, marker)
	if index < 0 {
		return 0, 0, false
	}
	parts := strings.Split(reference[index+len(marker):], "/")
	if len(parts) != 3 || parts[1] != "fenced-before" {
		return 0, 0, false
	}
	staleStarted, staleErr := strconv.ParseInt(parts[0], 10, 64)
	currentStarted, currentErr := strconv.ParseInt(parts[2], 10, 64)
	return staleStarted, currentStarted,
		staleErr == nil && currentErr == nil && staleStarted > 0 && staleStarted < currentStarted
}

func (h historyPosition) latest(eventTypes ...enumspb.EventType) (historyEventPosition, bool) {
	var latest historyEventPosition
	found := false
	for _, eventType := range eventTypes {
		position, exists := h.events[eventType]
		if exists && (!found || position.sequence > latest.sequence) {
			latest = position
			found = true
		}
	}
	return latest, found
}
