package nexus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
	"go.temporal.io/server/tools/umpire/temporal/local"
)

const (
	runtimeCodeCanceled    = "umpire.runtime.code.canceled"
	runtimeCodeFailed      = "umpire.runtime.code.failed"
	runtimeCodeRejected    = "umpire.runtime.code.rejected"
	runtimeCodeTimedOut    = "umpire.runtime.code.timed-out"
	runtimeCodeUnsupported = "umpire.runtime.code.unsupported"
)

type commandAdapter interface {
	Prepare(context.Context, umpireruntime.Environment, umpireruntime.Command) umpireruntime.Receipt
	Realize(context.Context, umpireruntime.Environment, umpireruntime.Command) umpireruntime.Receipt
	Observe(context.Context, umpireruntime.Environment, umpireruntime.Command) umpireruntime.Receipt
	Cleanup(context.Context, umpireruntime.Environment, umpireruntime.Command) umpireruntime.Receipt
}

type participant struct {
	mu       sync.Mutex
	expected map[umpireruntime.CommandKind]umpireruntime.Command
	called   map[umpireruntime.CommandKind]bool
	adapter  commandAdapter
}

// NewParticipant creates the sole built-in participant only after the complete
// checked request matches the System-owned program and closed local authority.
func NewParticipant(request umpireruntime.CheckedRunRequest) (umpireruntime.Participant, error) {
	adapter, err := newSDKCommandAdapter(request)
	if err != nil {
		return nil, err
	}
	return newParticipant(request, adapter)
}

func newParticipant(request umpireruntime.CheckedRunRequest, adapter commandAdapter) (*participant, error) {
	if !exactCheckedRequest(request) {
		return nil, fmt.Errorf("unsupported checked caller-closure request")
	}
	expected := make(map[umpireruntime.CommandKind]umpireruntime.Command, 4)
	for _, kind := range []umpireruntime.CommandKind{
		umpireruntime.CommandPrepare,
		umpireruntime.CommandRealize,
		umpireruntime.CommandObserve,
		umpireruntime.CommandCleanup,
	} {
		command, ok := request.Command(kind)
		if !ok {
			return nil, fmt.Errorf("unsupported checked caller-closure command")
		}
		expected[kind] = command
	}
	return &participant{
		expected: expected,
		called:   make(map[umpireruntime.CommandKind]bool, len(expected)),
		adapter:  adapter,
	}, nil
}

func exactCheckedRequest(request umpireruntime.CheckedRunRequest) bool {
	program := request.Program()
	occurrences := program.Occurrences()
	authority := request.Authority()
	configuration := request.RuntimeConfiguration()
	return request.Seed() == 0 && request.Attempt() == 1 &&
		program.DefinitionID() == callerClosureProgramDefinitionID &&
		program.Version() == callerClosureProgramVersion &&
		program.BehaviorFingerprint() == callerClosureProgramBehaviorFingerprint &&
		slices.Equal(program.TargetDefinitionIDs(), []string{callerClosureTargetDefinitionID}) &&
		slices.Equal(program.ActionDefinitionIDs(), []string{forceCloseActionDefinitionID}) &&
		len(occurrences) == 1 &&
		occurrences[0].DefinitionID() == forceCloseOccurrenceDefinitionID &&
		occurrences[0].ActionDefinitionID() == forceCloseActionDefinitionID &&
		occurrences[0].Position() == 1 &&
		slices.Equal(program.CapabilityDefinitionIDs(), callerClosureCapabilities) &&
		authority.DefinitionID() == local.ProfileDefinitionID &&
		authority.Version() == local.ProfileVersion &&
		authority.BehaviorFingerprint() == local.ProfileBehaviorFingerprint &&
		authority.ConfigurationDefinitionID() == callerClosureConfigurationDefinitionID &&
		authority.ConfigurationBehaviorFingerprint() == callerClosureConfigurationBehaviorFingerprint &&
		authority.ParticipantDefinitionID() == callerClosureParticipantDefinitionID &&
		authority.ProtocolDefinitionID() == callerClosureProtocolDefinitionID &&
		authority.ProtocolVersion() == 2 &&
		configuration.ConfigurationDefinitionID == callerClosureConfigurationDefinitionID &&
		configuration.BehaviorFingerprint == callerClosureConfigurationBehaviorFingerprint
}

func (p *participant) Prepare(ctx context.Context, environment umpireruntime.Environment, command umpireruntime.Command) umpireruntime.Receipt {
	return p.invoke(ctx, environment, command, umpireruntime.CommandPrepare)
}

func (p *participant) Realize(ctx context.Context, environment umpireruntime.Environment, command umpireruntime.Command) umpireruntime.Receipt {
	return p.invoke(ctx, environment, command, umpireruntime.CommandRealize)
}

func (p *participant) Observe(ctx context.Context, environment umpireruntime.Environment, command umpireruntime.Command) umpireruntime.Receipt {
	return p.invoke(ctx, environment, command, umpireruntime.CommandObserve)
}

func (p *participant) Cleanup(ctx context.Context, environment umpireruntime.Environment, command umpireruntime.Command) umpireruntime.Receipt {
	return p.invoke(ctx, environment, command, umpireruntime.CommandCleanup)
}

func (p *participant) invoke(
	ctx context.Context,
	environment umpireruntime.Environment,
	command umpireruntime.Command,
	kind umpireruntime.CommandKind,
) umpireruntime.Receipt {
	p.mu.Lock()
	defer p.mu.Unlock()
	if command != p.expected[kind] || p.called[kind] || !p.predecessorCalled(kind) {
		return emptyReceipt(command, umpireruntime.ReceiptUnsupported)
	}
	p.called[kind] = true
	if ctx == nil || ctx.Err() != nil {
		return emptyReceipt(command, umpireruntime.ReceiptCanceled)
	}
	if p.adapter == nil {
		return emptyReceipt(command, umpireruntime.ReceiptUnsupported)
	}
	switch kind {
	case umpireruntime.CommandPrepare:
		return p.adapter.Prepare(ctx, environment, command)
	case umpireruntime.CommandRealize:
		return p.adapter.Realize(ctx, environment, command)
	case umpireruntime.CommandObserve:
		return p.adapter.Observe(ctx, environment, command)
	case umpireruntime.CommandCleanup:
		return p.adapter.Cleanup(ctx, environment, command)
	default:
		return emptyReceipt(command, umpireruntime.ReceiptUnsupported)
	}
}

func (p *participant) predecessorCalled(kind umpireruntime.CommandKind) bool {
	switch kind {
	case umpireruntime.CommandPrepare:
		return true
	case umpireruntime.CommandRealize:
		return p.called[umpireruntime.CommandPrepare]
	case umpireruntime.CommandObserve:
		return p.called[umpireruntime.CommandRealize]
	case umpireruntime.CommandCleanup:
		return p.called[umpireruntime.CommandPrepare]
	default:
		return false
	}
}

func emptyReceipt(command umpireruntime.Command, status umpireruntime.ReceiptStatus) umpireruntime.Receipt {
	receipt, _ := umpireruntime.NewReceipt(
		command,
		status,
		[]umpireruntime.Fact{},
		[]umpireruntime.Resource{},
		[]umpireruntime.Resource{},
	)
	return receipt
}

var _ umpireruntime.Participant = (*participant)(nil)

type sdkCommandAdapter struct {
	workflowCorrelation  string
	operationCorrelation string
	participantResource  umpireruntime.Resource

	environment local.Environment
	endpoint    local.WorkerEndpoint
	operation   *callerClosureOperation
	run         client.WorkflowRun

	endpointAcquired       bool
	forceCloseAttempted    bool
	forceCloseAcknowledged bool
}

func newSDKCommandAdapter(request umpireruntime.CheckedRunRequest) (*sdkCommandAdapter, error) {
	adapter := &sdkCommandAdapter{}
	participantCorrelation := ""
	for _, correlation := range request.Correlations() {
		switch correlation.Kind() {
		case umpireruntime.CorrelationWorkflow:
			adapter.workflowCorrelation = correlation.Identity()
		case umpireruntime.CorrelationOperation:
			adapter.operationCorrelation = correlation.Identity()
		case umpireruntime.CorrelationParticipant:
			participantCorrelation = correlation.Identity()
		}
	}
	if adapter.workflowCorrelation == "" || adapter.operationCorrelation == "" || participantCorrelation == "" {
		return nil, fmt.Errorf("unsupported checked caller-closure correlations")
	}
	resource, err := umpireruntime.NewResource(umpireruntime.ResourceParticipant, participantCorrelation)
	if err != nil {
		return nil, err
	}
	adapter.participantResource = resource
	return adapter, nil
}

func (a *sdkCommandAdapter) Prepare(
	ctx context.Context,
	runtimeEnvironment umpireruntime.Environment,
	command umpireruntime.Command,
) umpireruntime.Receipt {
	environment, ok := local.AsEnvironment(runtimeEnvironment)
	if !ok {
		return adapterReceipt(command, umpireruntime.ReceiptUnsupported, runtimeCodeUnsupported,
			nil, nil, nil, adapterCorrelations{})
	}
	a.environment = environment
	correlations := a.correlations()
	endpoint, err := environment.CreateWorkerEndpoint(ctx, command)
	if err != nil {
		return adapterFailureReceipt(ctx, command, err, nil, nil, correlations)
	}
	a.endpoint = endpoint
	a.endpointAcquired = true
	a.operation = newCallerClosureOperation(a.operationCorrelation)

	workerReceipt := environment.StartWorker(ctx, command, callerClosureRegistration{operation: a.operation})
	facts := workerReceipt.Facts()
	acquired := append(workerReceipt.AcquiredResources(), a.participantResource)
	if workerReceipt.Status() != umpireruntime.ReceiptAccepted {
		return adapterReceipt(command, workerReceipt.Status(), receiptErrorCode(workerReceipt),
			facts, acquired, workerReceipt.ReleasedResources(), correlations)
	}
	options, ok := environment.WorkflowOptions(command)
	if !ok || options.RetryPolicy == nil || options.RetryPolicy.MaximumAttempts != 1 {
		return adapterReceipt(command, umpireruntime.ReceiptUnsupported, runtimeCodeUnsupported,
			facts, acquired, nil, correlations)
	}
	if err := waitForWorkerReadiness(ctx, environment.Client(), options.TaskQueue); err != nil {
		return adapterFailureReceipt(ctx, command, err, facts, acquired, correlations)
	}
	run, err := environment.Client().ExecuteWorkflow(
		ctx,
		options,
		callerWorkflowName,
		callerWorkflowInput{EndpointName: endpoint.Name(), OperationIdentity: a.operationCorrelation},
	)
	if err != nil {
		return adapterFailureReceipt(ctx, command, err, facts, acquired, correlations)
	}
	a.run = run
	runDone := make(chan error, 1)
	go func() { runDone <- run.Get(ctx, nil) }()
	select {
	case <-a.operation.started:
		starts, _ := a.operation.counts()
		if starts != 1 {
			return adapterReceipt(command, umpireruntime.ReceiptRejected, runtimeCodeRejected,
				facts, acquired, nil, correlations)
		}
		return adapterReceipt(command, umpireruntime.ReceiptAccepted, "", facts, acquired, nil, correlations)
	case err := <-runDone:
		return adapterFailureReceipt(ctx, command, err, facts, acquired, correlations)
	case <-ctx.Done():
		return adapterFailureReceipt(ctx, command, ctx.Err(), facts, acquired, correlations)
	}
}

func waitForWorkerReadiness(ctx context.Context, sdkClient client.Client, taskQueue string) error {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		workflowDescription, workflowErr := sdkClient.DescribeTaskQueue(ctx, taskQueue, enumspb.TASK_QUEUE_TYPE_WORKFLOW)
		nexusDescription, nexusErr := sdkClient.DescribeTaskQueue(ctx, taskQueue, enumspb.TASK_QUEUE_TYPE_NEXUS)
		if workflowErr == nil && nexusErr == nil &&
			len(workflowDescription.GetPollers()) > 0 && len(nexusDescription.GetPollers()) > 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (a *sdkCommandAdapter) Realize(
	ctx context.Context,
	_ umpireruntime.Environment,
	command umpireruntime.Command,
) umpireruntime.Receipt {
	correlations := a.correlations()
	if a.environment == nil || a.run == nil || a.operation == nil || a.forceCloseAttempted {
		return adapterReceipt(command, umpireruntime.ReceiptUnsupported, runtimeCodeUnsupported,
			nil, nil, nil, correlations)
	}
	a.forceCloseAttempted = true
	if err := a.environment.Client().CancelWorkflow(ctx, a.run.GetID(), a.run.GetRunID()); err != nil {
		return adapterFailureReceipt(ctx, command, err, nil, nil, correlations)
	}
	select {
	case <-a.operation.canceled:
		starts, cancellations := a.operation.counts()
		if starts != 1 || cancellations != 1 {
			return adapterReceipt(command, umpireruntime.ReceiptRejected, runtimeCodeRejected,
				nil, nil, nil, correlations)
		}
		a.forceCloseAcknowledged = true
		correlations = a.correlations()
		return adapterReceipt(command, umpireruntime.ReceiptAccepted, "", nil, nil, nil, correlations)
	case <-ctx.Done():
		return adapterFailureReceipt(ctx, command, ctx.Err(), nil, nil, correlations)
	}
}

func (a *sdkCommandAdapter) Observe(
	ctx context.Context,
	_ umpireruntime.Environment,
	command umpireruntime.Command,
) umpireruntime.Receipt {
	correlations := a.correlations()
	if a.environment == nil || a.run == nil || a.operation == nil || !a.forceCloseAcknowledged {
		return adapterReceipt(command, umpireruntime.ReceiptUnsupported, runtimeCodeUnsupported,
			nil, nil, nil, correlations)
	}
	iterator := a.environment.Client().GetWorkflowHistory(
		ctx, a.run.GetID(), a.run.GetRunID(), true, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	)
	_, cancellations := a.operation.counts()
	facts, err := projectTerminalHistory(command, iterator, correlations, cancellations)
	if err != nil {
		return adapterFailureReceipt(ctx, command, err, nil, nil, correlations)
	}
	return adapterReceipt(command, umpireruntime.ReceiptAccepted, "", facts, nil, nil, correlations)
}

func (a *sdkCommandAdapter) Cleanup(
	ctx context.Context,
	_ umpireruntime.Environment,
	command umpireruntime.Command,
) umpireruntime.Receipt {
	correlations := a.correlations()
	failed := false
	if a.environment == nil {
		return adapterReceipt(command, umpireruntime.ReceiptAccepted, "", nil, nil, nil, correlations)
	}
	if a.run != nil && !a.forceCloseAcknowledged {
		if err := a.environment.Client().TerminateWorkflow(
			ctx, a.run.GetID(), a.run.GetRunID(), "umpire participant cleanup",
		); err != nil {
			failed = true
		} else {
			a.run = nil
		}
	}
	released := []umpireruntime.Resource{}
	if a.endpointAcquired {
		if err := a.environment.DeleteWorkerEndpoint(ctx, command, a.endpoint); err != nil {
			failed = true
		} else {
			a.endpointAcquired = false
			released = append(released, a.participantResource)
		}
	}
	if failed {
		return adapterFailureReceipt(ctx, command, fmt.Errorf("participant cleanup failed"), nil, released, correlations)
	}
	return adapterReceipt(command, umpireruntime.ReceiptAccepted, "", nil, nil, released, correlations)
}

type adapterCorrelations struct {
	workflow      string
	operation     string
	cancellations uint64
	identities    local.Identities
}

func (a *sdkCommandAdapter) correlations() adapterCorrelations {
	correlations := adapterCorrelations{workflow: a.workflowCorrelation, operation: a.operationCorrelation}
	if a.environment != nil {
		correlations.identities = a.environment.Identities()
	}
	if a.operation != nil {
		_, correlations.cancellations = a.operation.counts()
	}
	return correlations
}

func adapterFailureReceipt(
	ctx context.Context,
	command umpireruntime.Command,
	_ error,
	facts []umpireruntime.Fact,
	resources []umpireruntime.Resource,
	correlations adapterCorrelations,
) umpireruntime.Receipt {
	status := umpireruntime.ReceiptFailed
	code := runtimeCodeFailed
	if ctx == nil || ctx.Err() != nil {
		status = umpireruntime.ReceiptCanceled
		code = runtimeCodeCanceled
		if ctx != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
			code = runtimeCodeTimedOut
		}
	}
	return adapterReceipt(command, status, code, facts, resources, nil, correlations)
}

func adapterReceipt(
	command umpireruntime.Command,
	status umpireruntime.ReceiptStatus,
	code string,
	facts []umpireruntime.Fact,
	acquired []umpireruntime.Resource,
	released []umpireruntime.Resource,
	correlations adapterCorrelations,
) umpireruntime.Receipt {
	if facts == nil {
		facts = []umpireruntime.Fact{}
	}
	if acquired == nil {
		acquired = []umpireruntime.Resource{}
	}
	if released == nil {
		released = []umpireruntime.Resource{}
	}
	fact, err := operationalFact(command, status, code, correlations)
	if err == nil {
		facts = append(facts, fact)
	}
	slices.SortFunc(facts, func(left, right umpireruntime.Fact) int {
		return compareStrings(left.DefinitionID(), right.DefinitionID())
	})
	slices.SortFunc(acquired, compareResources)
	slices.SortFunc(released, compareResources)
	receipt, err := umpireruntime.NewReceipt(command, status, facts, acquired, released)
	if err != nil {
		return emptyReceipt(command, umpireruntime.ReceiptFailed)
	}
	return receipt
}

func operationalFact(
	command umpireruntime.Command,
	status umpireruntime.ReceiptStatus,
	code string,
	correlations adapterCorrelations,
) (umpireruntime.Fact, error) {
	values := map[string]string{
		umpireruntime.EvidenceFieldCommandKind:      string(command.Kind()),
		umpireruntime.EvidenceFieldRunCorrelationID: command.RunIdentity(),
		umpireruntime.EvidenceFieldStatus:           string(status),
	}
	if code != "" {
		values[umpireruntime.EvidenceFieldErrorCode] = code
	}
	if correlations.workflow != "" {
		values[umpireruntime.EvidenceFieldWorkflowCorrelationID] = correlations.workflow
	}
	if correlations.operation != "" {
		values[umpireruntime.EvidenceFieldOperationCorrelationID] = correlations.operation
	}
	if command.Kind() == umpireruntime.CommandRealize && correlations.cancellations != 0 {
		values[umpireruntime.EvidenceFieldCancellationCallbackCount] = strconv.FormatUint(correlations.cancellations, 10)
	}
	if correlations.identities.Endpoint != "" {
		values[umpireruntime.EvidenceFieldEndpointIdentity] = correlations.identities.Endpoint
	}
	if correlations.identities.Namespace != "" {
		values[umpireruntime.EvidenceFieldNamespaceIdentity] = correlations.identities.Namespace
	}
	if correlations.identities.TaskQueue != "" {
		values[umpireruntime.EvidenceFieldTaskQueueIdentity] = correlations.identities.TaskQueue
	}
	fields, err := checkedFields(values)
	if err != nil {
		return umpireruntime.Fact{}, err
	}
	return umpireruntime.NewFact(
		factIdentity("participant-"+string(command.Kind()), command.RunIdentity()),
		participantFactSource(command.Kind()),
		"umpire.evidence.kind.participant-command",
		[]string{},
		fields,
	)
}

func historyFact(
	command umpireruntime.Command,
	eventID int64,
	eventType enumspb.EventType,
	previous string,
	correlations adapterCorrelations,
) (umpireruntime.Fact, error) {
	values := map[string]string{
		umpireruntime.EvidenceFieldEventID:                strconv.FormatInt(eventID, 10),
		umpireruntime.EvidenceFieldEventType:              "temporal.history." + eventType.String(),
		umpireruntime.EvidenceFieldOperationCorrelationID: correlations.operation,
		umpireruntime.EvidenceFieldRunCorrelationID:       command.RunIdentity(),
		umpireruntime.EvidenceFieldWorkflowCorrelationID:  correlations.workflow,
	}
	fields, err := checkedFields(values)
	if err != nil {
		return umpireruntime.Fact{}, err
	}
	causes := []string{}
	if previous != "" {
		causes = append(causes, previous)
	}
	digest := sha256.Sum256([]byte("umpire.temporal.nexus.history/v1\n" + command.RunIdentity()))
	definitionID := fmt.Sprintf(
		"umpire.runtime.fact.history.%020d.%s",
		eventID,
		hex.EncodeToString(digest[:8]),
	)
	return umpireruntime.NewFact(
		definitionID,
		umpireruntime.EvidenceSourceHistory,
		"umpire.evidence.kind.workflow-history-event",
		causes,
		fields,
	)
}

func checkedFields(values map[string]string) ([]umpireruntime.FactField, error) {
	definitionIDs := make([]string, 0, len(values))
	for definitionID := range values {
		definitionIDs = append(definitionIDs, definitionID)
	}
	slices.Sort(definitionIDs)
	fields := make([]umpireruntime.FactField, 0, len(definitionIDs))
	for _, definitionID := range definitionIDs {
		field, err := umpireruntime.NewFactField(definitionID, values[definitionID])
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}
	return fields, nil
}

func factIdentity(kind string, runIdentity string) string {
	digest := sha256.Sum256([]byte("umpire.temporal.nexus.fact/v1\n" + kind + "\n" + runIdentity))
	return "umpire.runtime.fact." + kind + "." + hex.EncodeToString(digest[:])
}

func participantFactSource(kind umpireruntime.CommandKind) string {
	if kind == umpireruntime.CommandCleanup {
		return umpireruntime.EvidenceSourceCleanup
	}
	return umpireruntime.EvidenceSourceParticipantOutput
}

func receiptErrorCode(receipt umpireruntime.Receipt) string {
	for _, fact := range receipt.Facts() {
		for _, field := range fact.Fields() {
			if field.DefinitionID() == umpireruntime.EvidenceFieldErrorCode {
				return field.Value()
			}
		}
	}
	return runtimeCodeFailed
}

func compareStrings(left, right string) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func compareResources(left, right umpireruntime.Resource) int {
	leftKey := string(left.Kind()) + "\n" + left.Identity()
	rightKey := string(right.Kind()) + "\n" + right.Identity()
	return compareStrings(leftKey, rightKey)
}
