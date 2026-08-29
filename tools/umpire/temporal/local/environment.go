package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"strconv"
	"sync"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/sdk/client"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/server/temporaltest"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

const (
	runtimeCodeCanceled      = "umpire.runtime.code.canceled"
	runtimeCodeCleanupFailed = "umpire.runtime.code.cleanup-failed"
	runtimeCodeFailed        = "umpire.runtime.code.failed"
	runtimeCodeTimedOut      = "umpire.runtime.code.timed-out"
	runtimeCodeUnsupported   = "umpire.runtime.code.unsupported"

	lifecycleFactAuthority = "authority-prepare"
	lifecycleFactWorker    = "worker-start"
	lifecycleFactIsolation = "environment-isolate"
	lifecycleFactCleanup   = "environment-cleanup"
)

// Identities contains only digest tokens safe for retained evidence. The raw
// namespace, endpoint, and task queue remain private to the live environment.
type Identities struct {
	Namespace string
	Endpoint  string
	TaskQueue string
}

// WorkerRegistration is the narrow SDK registration boundary used by the one
// built-in participant. It supplies no endpoint, namespace, credential, worker
// identity, task queue, executable name, or worker options.
type WorkerRegistration interface {
	Register(worker.Registry)
}

// WorkerEndpoint is one opaque, run-owned routing handle. Only its name is
// available to the built-in workflow; namespace and task queue remain private.
type WorkerEndpoint struct {
	name        string
	id          string
	version     int64
	runIdentity string
}

// Name returns the closed endpoint name required by the SDK workflow client.
func (e WorkerEndpoint) Name() string { return e.name }

// Environment is the Temporal-specific vertical adapter available to the one
// built-in participant. Its domain-neutral lifecycle remains runtime.Environment.
type Environment interface {
	umpireruntime.Environment
	Client() client.Client
	Identities() Identities
	WorkflowOptions(umpireruntime.Command) (client.StartWorkflowOptions, bool)
	CreateWorkerEndpoint(context.Context, umpireruntime.Command) (WorkerEndpoint, error)
	DeleteWorkerEndpoint(context.Context, umpireruntime.Command, WorkerEndpoint) error
	StartWorker(context.Context, umpireruntime.Command, WorkerRegistration) umpireruntime.Receipt
}

// NewFactory returns the sole closed local authority factory. It deliberately
// accepts no options so a caller cannot select or reuse a Temporal authority.
func NewFactory() umpireruntime.EnvironmentFactory {
	return newFactory(temporalStarter{})
}

// AsEnvironment narrows the generic runtime environment to this vertical adapter.
func AsEnvironment(value umpireruntime.Environment) (Environment, bool) {
	environment, ok := value.(*environment)
	return environment, ok
}

type factory struct {
	starter authorityStarter
}

func newFactory(starter authorityStarter) *factory {
	return &factory{starter: starter}
}

func (f *factory) Prepare(
	ctx context.Context,
	request umpireruntime.CheckedRunRequest,
	command umpireruntime.Command,
) (umpireruntime.Environment, umpireruntime.Receipt) {
	if !validPreparation(request, command) {
		return nil, lifecycleReceipt(command, lifecycleFactAuthority, umpireruntime.ReceiptUnsupported,
			runtimeCodeUnsupported, nil, nil, Identities{})
	}
	if ctx == nil || ctx.Err() != nil {
		return nil, lifecycleFailureReceipt(
			ctx, command, lifecycleFactAuthority, contextError(ctx), false, nil, Identities{},
		)
	}

	authority, startErr := f.starter.Start(ctx)
	if authority == nil {
		return nil, lifecycleFailureReceipt(
			ctx, command, lifecycleFactAuthority, startErr, false, nil, Identities{},
		)
	}
	environment := newEnvironment(request, authority)
	if startErr != nil {
		acquired := environment.recordOwnedResources()
		return environment, lifecycleFailureReceipt(
			ctx, command, lifecycleFactAuthority, startErr, false, acquired, environment.identities,
		)
	}
	if err := authority.Connect(ctx); err != nil {
		acquired := environment.recordOwnedResources()
		return environment, lifecycleFailureReceipt(
			ctx, command, lifecycleFactAuthority, err, false, acquired, environment.identities,
		)
	}
	environment.client = authority.SDKClient()
	acquired := environment.recordOwnedResources()
	return environment, lifecycleReceipt(
		command, lifecycleFactAuthority, umpireruntime.ReceiptAccepted,
		"", acquired, nil, environment.identities,
	)
}

func validPreparation(
	request umpireruntime.CheckedRunRequest,
	command umpireruntime.Command,
) bool {
	expected, ok := request.Command(umpireruntime.CommandPrepare)
	return ok && expected == command && command.Attempt() == 1 &&
		command.RunIdentity() == request.RunIdentity() && exactAuthority(request.Authority())
}

type environment struct {
	authority temporalAuthority
	client    client.Client

	runIdentity          string
	workflowCorrelation  string
	operationCorrelation string
	taskQueue            string
	workerIdentity       string
	identities           Identities

	mu            sync.Mutex
	live          map[umpireruntime.ResourceKind]umpireruntime.Resource
	workerStarted bool
}

func newEnvironment(
	request umpireruntime.CheckedRunRequest,
	authority temporalAuthority,
) *environment {
	environment := &environment{
		authority:   authority,
		runIdentity: request.RunIdentity(),
		live:        make(map[umpireruntime.ResourceKind]umpireruntime.Resource),
	}
	for _, correlation := range request.Correlations() {
		switch correlation.Kind() {
		case umpireruntime.CorrelationWorkflow:
			environment.workflowCorrelation = correlation.Identity()
		case umpireruntime.CorrelationOperation:
			environment.operationCorrelation = correlation.Identity()
		case umpireruntime.CorrelationTaskQueue:
			environment.taskQueue = correlation.Identity()
		case umpireruntime.CorrelationWorker:
			environment.workerIdentity = correlation.Identity()
		}
	}
	environment.identities = Identities{
		Namespace: digestIdentity("namespace", authority.Namespace()),
		Endpoint:  digestIdentity("endpoint", authority.Endpoint()),
		TaskQueue: digestIdentity("task-queue", environment.taskQueue),
	}
	return environment
}

func (e *environment) Client() client.Client { return e.client }

func (e *environment) Identities() Identities { return e.identities }

func (e *environment) WorkflowOptions(command umpireruntime.Command) (client.StartWorkflowOptions, bool) {
	if command.RunIdentity() != e.runIdentity || command.Attempt() != 1 ||
		(command.Kind() != umpireruntime.CommandPrepare && command.Kind() != umpireruntime.CommandRealize) {
		return client.StartWorkflowOptions{}, false
	}
	return client.StartWorkflowOptions{
		ID:                                       e.workflowCorrelation,
		TaskQueue:                                e.taskQueue,
		WorkflowExecutionTimeout:                 totalExecutionLimit(),
		WorkflowRunTimeout:                       totalExecutionLimit(),
		WorkflowTaskTimeout:                      10 * time.Second,
		WorkflowIDReusePolicy:                    enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		WorkflowIDConflictPolicy:                 enumspb.WORKFLOW_ID_CONFLICT_POLICY_FAIL,
		WorkflowExecutionErrorWhenAlreadyStarted: true,
		RetryPolicy:                              &sdktemporal.RetryPolicy{MaximumAttempts: 1},
	}, true
}

func (e *environment) CreateWorkerEndpoint(
	ctx context.Context,
	command umpireruntime.Command,
) (WorkerEndpoint, error) {
	if ctx == nil || ctx.Err() != nil || command.RunIdentity() != e.runIdentity ||
		command.Attempt() != 1 || command.Kind() != umpireruntime.CommandPrepare {
		if ctx == nil || ctx.Err() != nil {
			return WorkerEndpoint{}, contextError(ctx)
		}
		return WorkerEndpoint{}, errors.New("unsupported worker endpoint creation")
	}
	digest := sha256.Sum256([]byte("umpire.temporal.local.worker-endpoint/v1\n" + e.runIdentity))
	name := "umpire-" + hex.EncodeToString(digest[:16])
	response, err := e.client.OperatorService().CreateNexusEndpoint(
		ctx,
		&operatorservice.CreateNexusEndpointRequest{Spec: &nexuspb.EndpointSpec{
			Name: name,
			Target: &nexuspb.EndpointTarget{Variant: &nexuspb.EndpointTarget_Worker_{
				Worker: &nexuspb.EndpointTarget_Worker{
					Namespace: e.authority.Namespace(),
					TaskQueue: e.taskQueue,
				},
			}},
		}},
	)
	if err != nil {
		return WorkerEndpoint{}, err
	}
	if response.GetEndpoint() == nil {
		return WorkerEndpoint{}, errors.New("worker endpoint creation returned no handle")
	}
	return WorkerEndpoint{
		name: name, id: response.Endpoint.Id, version: response.Endpoint.Version,
		runIdentity: e.runIdentity,
	}, nil
}

func (e *environment) DeleteWorkerEndpoint(
	ctx context.Context,
	command umpireruntime.Command,
	endpoint WorkerEndpoint,
) error {
	if ctx == nil || ctx.Err() != nil || command.RunIdentity() != e.runIdentity ||
		command.Attempt() != 1 || command.Kind() != umpireruntime.CommandCleanup ||
		endpoint.runIdentity != e.runIdentity || endpoint.id == "" || endpoint.version <= 0 {
		if ctx == nil || ctx.Err() != nil {
			return contextError(ctx)
		}
		return errors.New("unsupported worker endpoint deletion")
	}
	_, err := e.client.OperatorService().DeleteNexusEndpoint(
		ctx,
		&operatorservice.DeleteNexusEndpointRequest{
			Id: endpoint.id, Version: endpoint.version,
		},
	)
	return err
}

func (e *environment) StartWorker(
	ctx context.Context,
	command umpireruntime.Command,
	registration WorkerRegistration,
) umpireruntime.Receipt {
	e.mu.Lock()
	defer e.mu.Unlock()
	if ctx == nil || ctx.Err() != nil {
		return lifecycleFailureReceipt(
			ctx, command, lifecycleFactWorker, contextError(ctx), false, nil, e.identities,
		)
	}
	if registration == nil || command.RunIdentity() != e.runIdentity || command.Attempt() != 1 ||
		command.Kind() != umpireruntime.CommandPrepare || e.workerStarted {
		return lifecycleReceipt(command, lifecycleFactWorker, umpireruntime.ReceiptUnsupported,
			runtimeCodeUnsupported, nil, nil, e.identities)
	}
	e.workerStarted = true
	before := ownedKinds(e.authority.OwnedResources())
	err := e.authority.StartWorker(ctx, e.taskQueue, e.workerIdentity, registration)
	acquired := e.recordOwnedResourcesAfter(before)
	if err != nil {
		return lifecycleFailureReceipt(
			ctx, command, lifecycleFactWorker, err, false, acquired, e.identities,
		)
	}
	return lifecycleReceipt(
		command, lifecycleFactWorker, umpireruntime.ReceiptAccepted,
		"", acquired, nil, e.identities,
	)
}

func (e *environment) Isolate(
	ctx context.Context,
	command umpireruntime.Command,
) umpireruntime.Receipt {
	if ctx == nil || ctx.Err() != nil {
		return lifecycleFailureReceipt(
			ctx, command, lifecycleFactIsolation, contextError(ctx), false, nil, e.identities,
		)
	}
	if command.RunIdentity() != e.runIdentity || command.Attempt() != 1 ||
		command.Phase() != umpireruntime.PhaseIsolation {
		return lifecycleReceipt(command, lifecycleFactIsolation, umpireruntime.ReceiptUnsupported,
			runtimeCodeUnsupported, nil, nil, e.identities)
	}
	return lifecycleReceipt(
		command, lifecycleFactIsolation, umpireruntime.ReceiptAccepted,
		"", nil, nil, e.identities,
	)
}

func (e *environment) Cleanup(
	ctx context.Context,
	command umpireruntime.Command,
) umpireruntime.Receipt {
	e.mu.Lock()
	defer e.mu.Unlock()
	if command.RunIdentity() != e.runIdentity || command.Attempt() != 1 ||
		command.Kind() != umpireruntime.CommandCleanup {
		return lifecycleReceipt(command, lifecycleFactCleanup, umpireruntime.ReceiptUnsupported,
			runtimeCodeUnsupported, nil, nil, e.identities)
	}
	if ctx == nil || ctx.Err() != nil {
		status, code := closedFailure(ctx, contextError(ctx), true)
		return cleanupReceipt(command, status, code, nil, len(e.live))
	}
	before := cloneLive(e.live)
	err := e.authority.Stop(ctx)
	remainingKinds := ownedKinds(e.authority.OwnedResources())
	released := make([]umpireruntime.Resource, 0, len(before))
	for kind, resource := range before {
		if _, remains := remainingKinds[ownedKindForResource(kind)]; remains {
			continue
		}
		released = append(released, resource)
		delete(e.live, kind)
	}
	sortResources(released)
	if err != nil {
		status, code := closedFailure(ctx, err, true)
		return cleanupReceipt(command, status, code, released, len(e.live))
	}
	return cleanupReceipt(command, umpireruntime.ReceiptAccepted, "", released, len(e.live))
}

func (e *environment) recordOwnedResources() []umpireruntime.Resource {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.recordOwnedResourcesAfter(nil)
}

func (e *environment) recordOwnedResourcesAfter(
	before map[ownedResourceKind]struct{},
) []umpireruntime.Resource {
	resources := []umpireruntime.Resource{}
	for _, owned := range e.authority.OwnedResources() {
		if before != nil {
			if _, existed := before[owned.kind]; existed {
				continue
			}
		}
		kind := runtimeResourceKind(owned.kind)
		if _, exists := e.live[kind]; exists {
			continue
		}
		resource, err := umpireruntime.NewResource(kind, e.resourceIdentity(kind))
		if err != nil {
			continue
		}
		e.live[kind] = resource
		resources = append(resources, resource)
	}
	// The returned environment itself is owned even when temporaltest has already
	// unwound every partial server resource.
	if _, exists := e.live[umpireruntime.ResourceEnvironment]; !exists {
		resource, err := umpireruntime.NewResource(
			umpireruntime.ResourceEnvironment,
			e.resourceIdentity(umpireruntime.ResourceEnvironment),
		)
		if err == nil {
			e.live[umpireruntime.ResourceEnvironment] = resource
			resources = append(resources, resource)
		}
	}
	sortResources(resources)
	return resources
}

func (e *environment) resourceIdentity(kind umpireruntime.ResourceKind) string {
	digest := sha256.Sum256([]byte("umpire.temporal.local.resource/v1\n" + string(kind) + "\n" + e.runIdentity))
	return "runtime.resource." + string(kind) + "." + hex.EncodeToString(digest[:])
}

func cloneLive(
	resources map[umpireruntime.ResourceKind]umpireruntime.Resource,
) map[umpireruntime.ResourceKind]umpireruntime.Resource {
	cloned := make(map[umpireruntime.ResourceKind]umpireruntime.Resource, len(resources))
	for kind, resource := range resources {
		cloned[kind] = resource
	}
	return cloned
}

func totalExecutionLimit() time.Duration {
	var total time.Duration
	for _, limit := range umpireruntime.CanonicalPhaseLimits() {
		total += limit.Duration()
	}
	return total
}

func digestIdentity(kind string, raw string) string {
	digest := sha256.Sum256([]byte("umpire.temporal.local.identity/v1\n" + kind + "\n" + raw))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func lifecycleFailureReceipt(
	ctx context.Context,
	command umpireruntime.Command,
	factIdentityKind string,
	err error,
	cleanup bool,
	acquired []umpireruntime.Resource,
	identities Identities,
) umpireruntime.Receipt {
	status, code := closedFailure(ctx, err, cleanup)
	return lifecycleReceipt(command, factIdentityKind, status, code, acquired, nil, identities)
}

func closedFailure(
	ctx context.Context,
	err error,
	cleanup bool,
) (umpireruntime.ReceiptStatus, string) {
	if hasConcreteFailure(err) {
		if cleanup {
			return umpireruntime.ReceiptFailed, runtimeCodeCleanupFailed
		}
		return umpireruntime.ReceiptFailed, runtimeCodeFailed
	}
	if ctx != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return umpireruntime.ReceiptCanceled, runtimeCodeTimedOut
	}
	if ctx != nil && errors.Is(ctx.Err(), context.Canceled) {
		return umpireruntime.ReceiptCanceled, runtimeCodeCanceled
	}
	if errors.Is(err, context.Canceled) {
		return umpireruntime.ReceiptCanceled, runtimeCodeCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return umpireruntime.ReceiptFailed, runtimeCodeTimedOut
	}
	if cleanup {
		return umpireruntime.ReceiptFailed, runtimeCodeCleanupFailed
	}
	return umpireruntime.ReceiptFailed, runtimeCodeFailed
}

func hasConcreteFailure(err error) bool {
	if err == nil {
		return false
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, cause := range joined.Unwrap() {
			if hasConcreteFailure(cause) {
				return true
			}
		}
		return false
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok && wrapped.Unwrap() != nil {
		return hasConcreteFailure(wrapped.Unwrap())
	}
	return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return context.Canceled
	}
	return ctx.Err()
}

func lifecycleReceipt(
	command umpireruntime.Command,
	factIdentityKind string,
	status umpireruntime.ReceiptStatus,
	code string,
	acquired []umpireruntime.Resource,
	released []umpireruntime.Resource,
	identities Identities,
) umpireruntime.Receipt {
	if acquired == nil {
		acquired = []umpireruntime.Resource{}
	}
	if released == nil {
		released = []umpireruntime.Resource{}
	}
	fields := environmentFields(command, status, code, identities)
	fact := checkedFact(command, umpireruntime.EvidenceSourceParticipantOutput,
		factIdentityKind, "umpire.evidence.kind.environment-lifecycle", fields)
	receipt, err := umpireruntime.NewReceipt(
		command, status, []umpireruntime.Fact{fact}, acquired, released,
	)
	if err != nil {
		return umpireruntime.Receipt{}
	}
	return receipt
}

func cleanupReceipt(
	command umpireruntime.Command,
	status umpireruntime.ReceiptStatus,
	code string,
	released []umpireruntime.Resource,
	openHandles int,
) umpireruntime.Receipt {
	if released == nil {
		released = []umpireruntime.Resource{}
	}
	fields := []umpireruntime.FactField{
		checkedField(umpireruntime.EvidenceFieldOpenHandleCount, strconv.Itoa(openHandles)),
		checkedField(umpireruntime.EvidenceFieldStatus, receiptStatusValue(status)),
	}
	if code != "" {
		fields = append(fields, checkedField(umpireruntime.EvidenceFieldErrorCode, code))
	}
	slices.SortFunc(fields, compareFactField)
	fact := checkedFact(command, umpireruntime.EvidenceSourceCleanup,
		"cleanup", "umpire.evidence.kind.environment-cleanup", fields)
	receipt, err := umpireruntime.NewReceipt(
		command, status, []umpireruntime.Fact{fact}, []umpireruntime.Resource{}, released,
	)
	if err != nil {
		return umpireruntime.Receipt{}
	}
	return receipt
}

func environmentFields(
	command umpireruntime.Command,
	status umpireruntime.ReceiptStatus,
	code string,
	identities Identities,
) []umpireruntime.FactField {
	values := map[string]string{
		umpireruntime.EvidenceFieldCommandKind:      string(command.Kind()),
		umpireruntime.EvidenceFieldRunCorrelationID: command.RunIdentity(),
		umpireruntime.EvidenceFieldStatus:           receiptStatusValue(status),
	}
	if code != "" {
		values[umpireruntime.EvidenceFieldErrorCode] = code
	}
	if identities.Endpoint != "" {
		values[umpireruntime.EvidenceFieldEndpointIdentity] = identities.Endpoint
	}
	if identities.Namespace != "" {
		values[umpireruntime.EvidenceFieldNamespaceIdentity] = identities.Namespace
	}
	if identities.TaskQueue != "" {
		values[umpireruntime.EvidenceFieldTaskQueueIdentity] = identities.TaskQueue
	}
	fields := make([]umpireruntime.FactField, 0, len(values))
	for definitionID, value := range values {
		fields = append(fields, checkedField(definitionID, value))
	}
	slices.SortFunc(fields, compareFactField)
	return fields
}

func checkedField(definitionID string, value string) umpireruntime.FactField {
	field, _ := umpireruntime.NewFactField(definitionID, value)
	return field
}

func compareFactField(left, right umpireruntime.FactField) int {
	if left.DefinitionID() < right.DefinitionID() {
		return -1
	}
	if left.DefinitionID() > right.DefinitionID() {
		return 1
	}
	return 0
}

func checkedFact(
	command umpireruntime.Command,
	source string,
	suffix string,
	kind string,
	fields []umpireruntime.FactField,
) umpireruntime.Fact {
	digest := sha256.Sum256([]byte("umpire.temporal.local.fact/v1\n" + suffix + "\n" + command.RunIdentity()))
	fact, _ := umpireruntime.NewFact(
		"umpire.runtime.fact."+suffix+"."+hex.EncodeToString(digest[:]),
		source,
		kind,
		[]string{},
		fields,
	)
	return fact
}

func receiptStatusValue(status umpireruntime.ReceiptStatus) string {
	return string(status)
}

func sortResources(resources []umpireruntime.Resource) {
	slices.SortFunc(resources, func(left, right umpireruntime.Resource) int {
		leftKey := string(left.Kind()) + "\n" + left.Identity()
		rightKey := string(right.Kind()) + "\n" + right.Identity()
		if leftKey < rightKey {
			return -1
		}
		if leftKey > rightKey {
			return 1
		}
		return 0
	})
}

type authorityStarter interface {
	Start(context.Context) (temporalAuthority, error)
}

type temporalAuthority interface {
	Connect(context.Context) error
	SDKClient() client.Client
	StartWorker(context.Context, string, string, WorkerRegistration) error
	Stop(context.Context) error
	OwnedResources() []ownedResource
	Namespace() string
	Endpoint() string
}

type ownedResourceKind string

const (
	ownedWorker ownedResourceKind = "worker"
	ownedClient ownedResourceKind = "client"
	ownedServer ownedResourceKind = "server"
)

type ownedResource struct {
	kind ownedResourceKind
}

func ownedKinds(resources []ownedResource) map[ownedResourceKind]struct{} {
	kinds := make(map[ownedResourceKind]struct{}, len(resources))
	for _, resource := range resources {
		kinds[resource.kind] = struct{}{}
	}
	return kinds
}

func runtimeResourceKind(kind ownedResourceKind) umpireruntime.ResourceKind {
	switch kind {
	case ownedWorker:
		return umpireruntime.ResourceWorker
	case ownedClient:
		return umpireruntime.ResourceConnection
	default:
		return umpireruntime.ResourceEnvironment
	}
}

func ownedKindForResource(kind umpireruntime.ResourceKind) ownedResourceKind {
	switch kind {
	case umpireruntime.ResourceWorker:
		return ownedWorker
	case umpireruntime.ResourceConnection:
		return ownedClient
	default:
		return ownedServer
	}
}

type temporalStarter struct{}

func (temporalStarter) Start(ctx context.Context) (temporalAuthority, error) {
	server, err := temporaltest.NewServerWithContext(ctx, temporaltest.WithFrontendHTTP())
	if server == nil {
		return nil, err
	}
	authority := &temporalTestAuthority{server: server, namespace: server.GetDefaultNamespace()}
	for _, resource := range server.OwnedResources() {
		if resource.Kind == temporaltest.LifecycleResourceServer {
			authority.endpoint = server.GetFrontendHostPort()
			break
		}
	}
	return authority, err
}

type temporalTestAuthority struct {
	server    *temporaltest.TestServer
	client    client.Client
	namespace string
	endpoint  string
}

func (a *temporalTestAuthority) Connect(ctx context.Context) error {
	connected, err := a.server.GetDefaultClientWithContext(ctx)
	a.client = connected
	return err
}

func (a *temporalTestAuthority) SDKClient() client.Client { return a.client }

func (a *temporalTestAuthority) StartWorker(
	ctx context.Context,
	taskQueue string,
	identity string,
	registration WorkerRegistration,
) error {
	_, err := a.server.NewWorkerWithOptionsContext(
		ctx,
		taskQueue,
		registration.Register,
		worker.Options{
			Identity:          identity,
			WorkerStopTimeout: umpireruntime.CanonicalPhaseLimits()[4].Duration(),
		},
	)
	return err
}

func (a *temporalTestAuthority) Stop(ctx context.Context) error {
	return a.server.StopContext(ctx)
}

func (a *temporalTestAuthority) OwnedResources() []ownedResource {
	owned := a.server.OwnedResources()
	resources := make([]ownedResource, 0, len(owned))
	for _, resource := range owned {
		switch resource.Kind {
		case temporaltest.LifecycleResourceWorker:
			resources = append(resources, ownedResource{kind: ownedWorker})
		case temporaltest.LifecycleResourceClient:
			resources = append(resources, ownedResource{kind: ownedClient})
		case temporaltest.LifecycleResourceServer:
			resources = append(resources, ownedResource{kind: ownedServer})
		}
	}
	return resources
}

func (a *temporalTestAuthority) Namespace() string { return a.namespace }
func (a *temporalTestAuthority) Endpoint() string  { return a.endpoint }

var _ umpireruntime.EnvironmentFactory = (*factory)(nil)
var _ Environment = (*environment)(nil)
var _ temporalAuthority = (*temporalTestAuthority)(nil)
