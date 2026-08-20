package temporal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	sdkclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/server/tests/umpire3/environment"
	"go.temporal.io/server/tests/umpire3/participant"
	"go.temporal.io/server/tests/umpire3/protocol"
)

type SDKFactoryOptions struct {
	Client                sdkclient.Client
	Registry              worker.Registry
	Namespace             string
	TaskQueue             string
	BuildID               string
	ConfigurationIdentity string
	ProfileName           string
	EvidenceProfile       string
	DrivingAuthority      string
	ObservationAuthority  string
	FaultAuthority        string
	HardExecutionBudget   bool
	WorkflowID            func(protocol.Experiment) string
	CleanupTimeout        time.Duration
	NegativeControl       bool
	NexusEndpoint         string
	NexusService          string
	NexusOperation        string
	Capabilities          []string
	CorroboratingHistory  []CorroboratingHistorySource
	WorkflowTaskFencer    participant.WorkflowTaskFencer
	CallbackDriver        participant.CallbackDriver
	NexusDriver           participant.NexusDriver
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
	options SDKFactoryOptions
}

func NewSDKFactory(options SDKFactoryOptions) (*SDKFactory, error) {
	if options.Client == nil || options.Registry == nil || options.Namespace == "" ||
		options.TaskQueue == "" || options.BuildID == "" || options.WorkflowID == nil {
		return nil, errors.New("SDK environment requires client, registry, namespace, task queue, build, and workflow identity")
	}
	if options.CleanupTimeout <= 0 {
		return nil, errors.New("SDK environment requires a positive cleanup timeout")
	}
	if options.ConfigurationIdentity == "" {
		digest := sha256.Sum256([]byte(options.Namespace + "\x00" + options.TaskQueue + "\x00" + options.BuildID))
		options.ConfigurationIdentity = "sha256:" + hex.EncodeToString(digest[:])
	}
	if options.ProfileName == "" {
		options.ProfileName = "sdk-public-history"
	}
	if options.EvidenceProfile == "" {
		options.EvidenceProfile = environment.EvidenceProfilePublicGRPCHistory
		if len(options.CorroboratingHistory) != 0 {
			options.EvidenceProfile = environment.EvidenceProfileDualHistory
		}
	}
	if options.DrivingAuthority == "" {
		options.DrivingAuthority = "temporal-sdk"
	}
	if options.ObservationAuthority == "" {
		options.ObservationAuthority = "temporal-public-history"
		if len(options.CorroboratingHistory) != 0 {
			options.ObservationAuthority = "temporal-public-history+temporal-history-service"
		}
	}
	if options.FaultAuthority == "" {
		options.FaultAuthority = "none"
	}
	for _, source := range options.CorroboratingHistory {
		if source == nil {
			return nil, errors.New("SDK environment has a nil corroborating history source")
		}
	}
	if options.EvidenceProfile == environment.EvidenceProfileDualHistory && len(options.CorroboratingHistory) == 0 {
		return nil, errors.New("dual-history evidence profile requires a corroborating history source")
	}
	capabilities, err := normalizeSDKCapabilities(options)
	if err != nil {
		return nil, err
	}
	options.Capabilities = capabilities
	profile := environment.Profile{
		Name: options.ProfileName, BuildID: options.BuildID,
		ConfigurationIdentity: options.ConfigurationIdentity, EvidenceProfile: options.EvidenceProfile,
		DrivingAuthority: options.DrivingAuthority, ObservationAuthority: options.ObservationAuthority,
		FaultAuthority: options.FaultAuthority, IsolationIdentity: options.Namespace + "/" + options.TaskQueue,
		RetentionClass: "semantic-redacted", HardExecutionBudget: options.HardExecutionBudget,
	}
	if err := profile.Validate(); err != nil {
		return nil, fmt.Errorf("validate SDK environment profile: %w", err)
	}
	return &SDKFactory{options: options}, nil
}

func (f *SDKFactory) Capabilities() []string {
	return slices.Clone(f.options.Capabilities)
}

func normalizeSDKCapabilities(options SDKFactoryOptions) ([]string, error) {
	catalog, err := protocol.DefaultCatalog()
	if err != nil {
		return nil, err
	}
	known := make(map[string]struct{}, len(catalog.Capabilities))
	for _, capability := range catalog.Capabilities {
		known[string(capability.Identifier)] = struct{}{}
	}
	capabilities := slices.Clone(options.Capabilities)
	if len(capabilities) == 0 {
		capabilities = []string{"history-observation", "update", "workflow-task-control"}
		if options.NexusEndpoint != "" && options.NexusService != "" && options.NexusOperation != "" {
			capabilities = append(capabilities, "nexus", "nexus-observation", "nexus-worker-control")
		}
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

func (f *SDKFactory) Prepare(ctx context.Context, experiment protocol.Experiment) (environment.Session, error) {
	program, _, err := participant.CompileExperiment(experiment)
	if err != nil {
		return nil, fmt.Errorf("compile SDK participant experiment: %w", err)
	}
	if f.options.NegativeControl {
		for index := range program.Commands {
			program.Commands[index].Response = participant.ResponseFailure
		}
	}
	workflowID := f.options.WorkflowID(experiment)
	if workflowID == "" {
		return nil, errors.New("SDK environment returned an empty workflow identity")
	}
	runner, err := participant.NewSDKRunner(participant.SDKOptions{
		Client: f.options.Client, Registry: f.options.Registry, TaskQueue: f.options.TaskQueue,
		Namespace: f.options.Namespace, WorkflowID: workflowID, CleanupTimeout: f.options.CleanupTimeout,
		NexusEndpoint: f.options.NexusEndpoint, NexusService: f.options.NexusService,
		NexusOperation:     f.options.NexusOperation,
		WorkflowTaskFencer: f.options.WorkflowTaskFencer,
		CallbackDriver:     f.options.CallbackDriver,
		NexusDriver:        f.options.NexusDriver,
	})
	if err != nil {
		return nil, err
	}
	session, err := participant.Start(ctx, program, runner)
	if err != nil {
		return nil, err
	}
	sdk := &sdkSession{
		experiment: experiment, participant: session, client: f.options.Client,
		namespace: f.options.Namespace, taskQueue: f.options.TaskQueue,
		buildID: f.options.BuildID, configurationIdentity: f.options.ConfigurationIdentity,
		profileName: f.options.ProfileName, evidenceProfile: f.options.EvidenceProfile,
		drivingAuthority: f.options.DrivingAuthority, observationAuthority: f.options.ObservationAuthority,
		faultAuthority: f.options.FaultAuthority, hardExecutionBudget: f.options.HardExecutionBudget,
		results: make(map[string]participant.Result),
	}
	if len(f.options.CorroboratingHistory) != 0 {
		return &corroboratingSDKSession{
			sdkSession: sdk, sources: slices.Clone(f.options.CorroboratingHistory),
		}, nil
	}
	return sdk, nil
}

type sdkSession struct {
	experiment            protocol.Experiment
	participant           *participant.Session
	client                sdkclient.Client
	namespace             string
	taskQueue             string
	buildID               string
	configurationIdentity string
	profileName           string
	evidenceProfile       string
	drivingAuthority      string
	observationAuthority  string
	faultAuthority        string
	hardExecutionBudget   bool
	results               map[string]participant.Result
}

type corroboratingSDKSession struct {
	*sdkSession
	sources []CorroboratingHistorySource
}

func (s *sdkSession) Realize(
	ctx context.Context,
	action protocol.Action,
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
		Source: "temporal-public-history", SourceIdentity: s.namespace,
		ClockDomain: "temporal-history-event-id", SourceSequence: history.sequence,
		Reference: history.reference, CausalReferences: []string{result.Reference},
		EntityIdentity: result.WorkflowID + "/" + result.RunID,
		Lineage:        append([]string{s.experiment.ExperimentID}, result.Lineage...),
		PayloadDigest:  result.PayloadDigest, GroundedBindings: grounded,
		TerminalState: result.TerminalState, TerminalDisposition: result.TerminalDisposition,
	}, nil
}

func (s *sdkSession) Observe(
	ctx context.Context,
	checkpoint protocol.Checkpoint,
	_ environment.Bindings,
) (environment.Observation, error) {
	if len(s.results) == 0 {
		return environment.Observation{}, errors.New("no SDK participant result is available")
	}
	latest := s.latestResult()
	if latest.WorkflowID == "" || latest.RunID == "" {
		return environment.Observation{}, errors.New("SDK participant identity evidence is incomplete")
	}
	history, err := s.latestHistory(ctx, latest.WorkflowID, latest.RunID)
	if err != nil {
		return environment.Observation{}, err
	}
	satisfied := s.observationSatisfied(checkpoint.Observation, history)
	position := history.supportingPosition(checkpoint.Observation)
	identity := latest.WorkflowID + "/" + latest.RunID
	return environment.Observation{
		CheckpointID: checkpoint.Identifier, Kind: checkpoint.Observation, Satisfied: satisfied,
		Source: "temporal-public-history", SourceIdentity: s.namespace,
		ClockDomain: "temporal-history-event-id", SourceSequence: position.sequence,
		AuthoritativeTimeUnixNano: position.timestamp, ObservedAtUnixNano: time.Now().UnixNano(),
		Reference:       position.reference + "/" + checkpoint.Identifier,
		CausalReference: position.reference, CausalReferences: []string{latest.Reference},
		EntityIdentity: identity,
		Lineage:        append([]string{s.experiment.ExperimentID}, latest.Lineage...),
		PayloadDigest:  latest.PayloadDigest,
	}, nil
}

func (s *corroboratingSDKSession) Corroborate(
	ctx context.Context,
	checkpoint protocol.Checkpoint,
	_ environment.Bindings,
) ([]environment.Observation, error) {
	latest := s.latestResult()
	if latest.WorkflowID == "" || latest.RunID == "" {
		return nil, errors.New("SDK participant identity evidence is incomplete")
	}
	observations := make([]environment.Observation, 0, len(s.sources))
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
		supporting := position.supportingPosition(checkpoint.Observation)
		observations = append(observations, environment.Observation{
			CheckpointID: checkpoint.Identifier, Kind: checkpoint.Observation,
			Satisfied: s.observationSatisfied(checkpoint.Observation, position),
			Source:    history.Source, SourceIdentity: history.SourceIdentity, ClockDomain: history.ClockDomain,
			SourceSequence: supporting.sequence, AuthoritativeTimeUnixNano: supporting.timestamp,
			ObservedAtUnixNano: time.Now().UnixNano(), Reference: supporting.reference + "/" + checkpoint.Identifier,
			CausalReference: supporting.reference, CausalReferences: []string{latest.Reference},
			EntityIdentity: latest.WorkflowID + "/" + latest.RunID,
			Lineage:        append([]string{s.experiment.ExperimentID}, latest.Lineage...),
			PayloadDigest:  latest.PayloadDigest,
		})
	}
	return observations, nil
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

func (s *sdkSession) Profile() environment.Profile {
	return environment.Profile{
		Name: s.profileName, BuildID: s.buildID,
		ConfigurationIdentity: s.configurationIdentity,
		EvidenceProfile:       s.evidenceProfile, DrivingAuthority: s.drivingAuthority,
		ObservationAuthority: s.observationAuthority, FaultAuthority: s.faultAuthority,
		IsolationIdentity: s.namespace + "/" + s.taskQueue, RetentionClass: "semantic-redacted",
		HardExecutionBudget: s.hardExecutionBudget,
	}
}

type historyPosition struct {
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
		events: make(map[enumspb.EventType]historyEventPosition), taskQueues: make(map[string]bool),
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

func (s *sdkSession) observationSatisfied(observation string, history historyPosition) bool {
	if s.hasFailedResult() && observation != "nexus-operation-closed" {
		return false
	}
	switch observation {
	case "cancellation-accepted":
		return s.hasActionStatus("request-cancellation", "completed") && history.has(
			enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_CANCELED)
	case "cancellation-won":
		return s.hasActionStatus("commit-cancellation", "completed") && history.has(
			enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_CANCELED)
	case "stale-success-absent":
		return s.hasActionStatus("commit-cancellation", "completed") &&
			s.allActionStatuses("worker-returns-success", "suppressed") &&
			s.allActionStatuses("persist-success", "suppressed")
	case "update-accepted":
		return history.has(enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED)
	case "update-completed":
		return history.has(enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED)
	case "workflow-task-acknowledged":
		return history.has(enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED)
	case "speculative-task-valid":
		created, createdQualified := s.qualifiedActionReceipt(
			"create-speculative-workflow-task", "temporal-sdk-speculative-update")
		committed, committedQualified := s.qualifiedActionReceipt(
			"commit-speculative-workflow-task", "temporal-sdk-speculative-update")
		return createdQualified && committedQualified &&
			strings.Contains(created.Reference, "/speculative-update/") &&
			strings.Contains(committed.Reference, "/speculative-update/") &&
			history.has(enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED,
				enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED,
				enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED)
	case "workflow-task-not-starved":
		receipt, qualified := s.qualifiedActionReceipt(
			"dispatch-assurance-workflow-task", "temporal-sdk-workflow-progress")
		scheduled, scheduledObserved := history.latest(enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED)
		started, startedObserved := history.latest(enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED)
		completed, completedObserved := history.latest(enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED)
		return qualified && strings.Contains(receipt.Reference, "/workflow-progress/") &&
			scheduledObserved && startedObserved && completedObserved &&
			scheduled.sequence < started.sequence && started.sequence < completed.sequence
	case "nexus-operation-closed":
		return history.nexusOperationClosedBeforeWorkflow()
	case "nexus-activity-links-consistent":
		return s.hasActionStatus("link-nexus-activity", "completed") &&
			history.has(enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED) &&
			history.nexusActivityForwardLinked && history.nexusActivityReverseLinked
	case "nexus-timeout-valid":
		return history.has(enumspb.EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT) &&
			history.nexusTimeoutType == enumspb.TIMEOUT_TYPE_START_TO_CLOSE &&
			strings.Contains(history.nexusTimeoutMessage, "operation timed out")
	case "callback-reference-valid":
		return history.callbackRegistered && s.hasQualifiedActionReceipt("register-callback",
			"temporal-completion-callback-registration", "temporal-shared-handler-registration",
			"temporal-nexus-callback-registration")
	case "callback-response-consistent":
		return history.callbackRegistered && history.has(enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED) &&
			s.hasQualifiedActionReceipt("register-callback",
				"temporal-completion-callback-registration", "temporal-shared-handler-registration",
				"temporal-nexus-callback-registration") &&
			s.hasQualifiedActionReceipt("record-callback-response",
				"temporal-nexus-completion-callback-receiver", "temporal-shared-handler-completion",
				"temporal-nexus-callback-rejection")
	case "workflow-continuation-lineage-valid":
		latest := s.latestResult()
		return s.hasQualifiedActionReceipt("continue-workflow", "temporal-sdk-continuation") &&
			len(latest.Lineage) >= 4 && history.continuedExecutionRunID == latest.Lineage[len(latest.Lineage)-2] &&
			history.originalExecutionRunID == latest.RunID && history.firstExecutionRunID == latest.Lineage[len(latest.Lineage)-2]
	case "workflow-reset-lineage-valid":
		latest := s.latestResult()
		return s.hasQualifiedActionReceipt("reset-workflow", "temporal-sdk-reset") &&
			len(latest.Lineage) >= 4 && history.originalExecutionRunID == latest.Lineage[len(latest.Lineage)-2] &&
			history.originalExecutionRunID != latest.RunID && history.firstExecutionRunID == latest.Lineage[len(latest.Lineage)-2]
	case "workflow-routing-isolated":
		route, qualified := s.qualifiedActionReceipt("route-workflow-task", "temporal-sdk-routing")
		return history.taskQueues[s.taskQueue] && history.has(enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED) &&
			qualified && route.SourceIdentity == s.taskQueue &&
			strings.Contains(route.Reference, "/task-queue/"+s.taskQueue)
	case "workflow-ownership-fenced":
		receipt, qualified := s.qualifiedActionReceipt(
			"fence-workflow-owner", "umpire3-workflow-task-fencer")
		failed, failedObserved := history.latest(enumspb.EVENT_TYPE_WORKFLOW_TASK_FAILED)
		completed, completedObserved := history.latest(enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED)
		_, _, referenceValid := parseWorkflowOwnerFencingReference(receipt.Reference)
		return qualified && receipt.SourceIdentity == "umpire3-workflow-task-fencer" &&
			referenceValid && failedObserved && completedObserved && failed.sequence < completed.sequence
	case "entity-progressed":
		receipt, qualified := s.qualifiedActionReceipt(
			"progress-entity", "temporal-sdk-workflow-progress")
		return qualified && strings.Contains(receipt.Reference, "/workflow-progress/") && history.hasAny(
			enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_COMPLETED,
			enumspb.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
			enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED,
			enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED,
		)
	default:
		return false
	}
}

func parseWorkflowOwnerFencingReference(reference string) (int64, int64, bool) {
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

func (s *sdkSession) hasFailedResult() bool {
	for _, result := range s.results {
		if result.Status == "failed" {
			return true
		}
	}
	return false
}

func (s *sdkSession) hasActionStatus(actionKind string, status string) bool {
	for _, action := range s.experiment.Actions {
		if action.Kind != actionKind {
			continue
		}
		if result, exists := s.results[action.Identifier]; exists && result.Status == status {
			return true
		}
	}
	return false
}

func (s *sdkSession) allActionStatuses(actionKind string, status string) bool {
	found := false
	for _, action := range s.experiment.Actions {
		if action.Kind != actionKind {
			continue
		}
		result, exists := s.results[action.Identifier]
		if !exists {
			continue
		}
		found = true
		if result.Status != status {
			return false
		}
	}
	return found
}

func (s *sdkSession) hasQualifiedActionReceipt(actionKind string, allowedSources ...string) bool {
	_, qualified := s.qualifiedActionReceipt(actionKind, allowedSources...)
	return qualified
}

func (s *sdkSession) qualifiedActionReceipt(
	actionKind string,
	allowedSources ...string,
) (participant.Result, bool) {
	for _, action := range s.experiment.Actions {
		if action.Kind != actionKind {
			continue
		}
		result, exists := s.results[action.Identifier]
		if !exists || result.Status != "completed" || result.Reference == "" ||
			result.WorkflowID == "" || result.RunID == "" || len(result.Lineage) == 0 {
			continue
		}
		if slices.Contains(allowedSources, result.Source) {
			return result, true
		}
	}
	return participant.Result{}, false
}

func (h historyPosition) has(eventTypes ...enumspb.EventType) bool {
	for _, eventType := range eventTypes {
		if _, exists := h.events[eventType]; !exists {
			return false
		}
	}
	return true
}

func (h historyPosition) hasAny(eventTypes ...enumspb.EventType) bool {
	for _, eventType := range eventTypes {
		if _, exists := h.events[eventType]; exists {
			return true
		}
	}
	return false
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

func (h historyPosition) nexusOperationClosedBeforeWorkflow() bool {
	operation, operationObserved := h.latest(
		enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED,
		enumspb.EVENT_TYPE_NEXUS_OPERATION_FAILED,
		enumspb.EVENT_TYPE_NEXUS_OPERATION_CANCELED,
		enumspb.EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT,
	)
	workflow, workflowObserved := h.latest(
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED,
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED,
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED,
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_TERMINATED,
		enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_TIMED_OUT,
	)
	return operationObserved && workflowObserved && operation.sequence > 0 && operation.sequence <= workflow.sequence
}

func (h historyPosition) supportingPosition(observation string) historyEventPosition {
	preferences := map[string][]enumspb.EventType{
		"cancellation-accepted":      {enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_CANCELED},
		"cancellation-won":           {enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_CANCELED},
		"stale-success-absent":       {enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED},
		"update-accepted":            {enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED},
		"update-completed":           {enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED},
		"workflow-task-acknowledged": {enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED},
		"speculative-task-valid":     {enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED},
		"workflow-task-not-starved":  {enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED},
		"nexus-operation-closed": {
			enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED, enumspb.EVENT_TYPE_NEXUS_OPERATION_FAILED,
			enumspb.EVENT_TYPE_NEXUS_OPERATION_CANCELED, enumspb.EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT,
		},
		"nexus-activity-links-consistent":     {enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED},
		"nexus-timeout-valid":                 {enumspb.EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT},
		"callback-reference-valid":            {enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED},
		"callback-response-consistent":        {enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED},
		"workflow-continuation-lineage-valid": {enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED},
		"workflow-reset-lineage-valid":        {enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED},
		"workflow-routing-isolated":           {enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED},
		"workflow-ownership-fenced":           {enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED},
		"entity-progressed": {
			enumspb.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_COMPLETED,
			enumspb.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
			enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED,
			enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED,
		},
	}
	for _, eventType := range preferences[observation] {
		if position, exists := h.events[eventType]; exists {
			return position
		}
	}
	return historyEventPosition{sequence: h.sequence, timestamp: h.timestamp, reference: h.reference}
}
