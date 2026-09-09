package worker

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sync/atomic"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
)

const (
	startWorkflowMethod = "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
	getHistoryMethod    = "/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory"
)

type Driver struct {
	options           hostOptions
	registry          *workerRegistry
	mu                contextMutex
	sessions          map[string]*Session
	tombstones        []*Session
	workflowRoutes    map[workflowRouteIndex][]*Session
	nexusRoutes       map[nexusRouteIndex][]*Session
	routeAssociations int
	nextSession       atomic.Uint64
}

type hostOptions struct {
	profile        testpilot.ProfileSpec
	workerRoleID   string
	client         client.Client
	workerOptions  worker.Options
	sessionOptions func(context.Context, string) (SessionOptions, error)
	maximum        int
	diagnostics    int
	requestBytes   int64
	now            func() time.Time
	completion     *completionTransport
}

func New(options Options) (*Driver, error) {
	if nilValue(options.Client) || options.WorkerRoleID == "" || !validWorkerProfile(options.Profile) {
		return nil, ErrInvalid
	}
	if _, err := options.Profile.BindingFingerprint(); err != nil {
		return nil, ErrInvalid
	}
	limits := options.Profile.ProgramLimits
	completion, err := newCompletionTransport(options.HTTPClient, options.SystemCallbackBaseURL, limits)
	if err != nil {
		return nil, err
	}
	maximum, diagnostics := boundedInt(limits.GetMaxActivations()), min(boundedInt(limits.GetMaxRunEvents()), 64)
	h := &Driver{
		mu:             newContextMutex(),
		sessions:       make(map[string]*Session),
		tombstones:     make([]*Session, 0, diagnostics),
		workflowRoutes: make(map[workflowRouteIndex][]*Session),
		nexusRoutes:    make(map[nexusRouteIndex][]*Session),
		options: hostOptions{
			profile: options.Profile.Snapshot(), workerRoleID: options.WorkerRoleID, client: options.Client,
			workerOptions:  worker.Options{WorkerStopTimeout: options.WorkerStopTimeout},
			sessionOptions: options.SessionOptions, maximum: maximum, diagnostics: diagnostics,
			requestBytes: limits.GetMaxRequestBytes(), now: time.Now, completion: completion,
		},
	}
	h.registry = newWorkerRegistry(maximum, h.newSDKWorker)
	return h, nil
}

func validWorkerProfile(profile testpilot.ProfileSpec) bool {
	limits := profile.ProgramLimits
	ceiling := &testpilotspb.ProgramLimits{
		MaxEntrypoints: 10000, MaxNodes: 10000, MaxEdges: 100000, MaxActivations: 100000,
		MaxAttempts: 100000, MaxRunEvents: 100000, MaxExpressionDepth: 64, MaxPathFanout: 10000,
		MaxRequestBytes: 16 << 20, MaxResponseBytes: 16 << 20,
		MaxTotalDurationMilliseconds: 86400000, MaxCleanupDurationMilliseconds: 86400000,
	}
	if profile.Identity == "" || len(profile.Identity) > 256 || profile.Catalog == nil || profile.Catalog.Identity() == "" || limits == nil || len(profile.Capabilities) > int(testpilot.MaxCapability) || len(profile.Roles) > 10000 {
		return false
	}
	fields := limits.ProtoReflect().Descriptor().Fields()
	for index := 0; index < fields.Len(); index++ {
		field := fields.Get(index)
		value := limits.ProtoReflect().Get(field).Int()
		if value <= 0 || value > ceiling.ProtoReflect().Get(field).Int() {
			return false
		}
	}
	methods, carriers, shapes := 0, 0, 0
	for _, role := range profile.Roles {
		if len(role.Methods) > 10000 || len(role.ReservationCarriers) > 10000 || methods > 100000-len(role.Methods) || carriers > 100000-len(role.ReservationCarriers) {
			return false
		}
		methods += len(role.Methods)
		carriers += len(role.ReservationCarriers)
		for _, carrier := range role.ReservationCarriers {
			if len(carrier.Shapes) > 2 || shapes > 100000-len(carrier.Shapes) {
				return false
			}
			shapes += len(carrier.Shapes)
		}
	}
	return true
}

func (h *Driver) Identity(ctx context.Context) (testpilot.DriverIdentity, error) {
	if h == nil || ctx == nil {
		return testpilot.DriverIdentity{}, ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return testpilot.DriverIdentity{}, err
	}
	fingerprint, err := h.options.profile.BindingFingerprint()
	if err != nil {
		return testpilot.DriverIdentity{}, err
	}
	return testpilot.DriverIdentity{Profile: h.options.profile.Identity, Catalog: h.options.profile.Catalog.Identity(), Bindings: fingerprint}, nil
}

func (h *Driver) Validate(ctx context.Context, program testpilot.PreparedProgram) error {
	if h == nil || ctx == nil || program.Snapshot() == nil {
		return ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	plans := program.Entrypoints()
	if cleanup, ok := program.Cleanup(); ok {
		plans = append(plans, cleanup)
	}
	// A fault needs a worker to stop, so a Program that requests one is only realizable when it
	// also brings a worker. Validate applies the same rule Open does rather than admitting a
	// Program that could only fail at dispatch.
	requireWorker := hasWorkerEntrypoint(plans) || plansDeclareFault(plans)
	_, err := h.prepareDefinitionResources(program.Snapshot(), plans, program.Roles(), requireWorker)
	return err
}

func plansDeclareFault(plans []testpilot.EntrypointPlan) bool {
	for _, plan := range plans {
		for _, instruction := range plan.Instructions() {
			if instruction.Source().GetInstruction().GetInjectFault() != nil {
				return true
			}
		}
	}
	return false
}

func hasWorkerEntrypoint(plans []testpilot.EntrypointPlan) bool {
	for _, plan := range plans {
		if plan.Context() == testpilotspb.ENTRYPOINT_KIND_WORKFLOW || plan.Context() == testpilotspb.ENTRYPOINT_KIND_ACTIVITY || plan.Context() == testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER {
			return true
		}
	}
	return false
}

func (h *Driver) Open(ctx context.Context, runID string, program testpilot.PreparedProgram) (testpilot.Session, error) {
	if h == nil || ctx == nil || h.options.sessionOptions == nil {
		return nil, ErrInvalid
	}
	options, err := h.options.sessionOptions(ctx, runID)
	if err != nil {
		return nil, err
	}
	return h.OpenSession(ctx, runID, program, options)
}

func (h *Driver) OpenSession(ctx context.Context, runID string, program testpilot.PreparedProgram, options SessionOptions) (*Session, error) {
	if h == nil || ctx == nil || runID == "" || nilValue(options.Bridge) {
		return nil, ErrInvalid
	}
	definition, err := h.prepareDefinition(program)
	if err != nil {
		return nil, err
	}
	if definition.hasAsync && options.NewCapability == nil {
		return nil, ErrInvalid
	}
	if err := h.mu.lock(ctx); err != nil {
		return nil, err
	}
	if len(h.sessions) >= h.options.maximum || h.sessions[runID] != nil {
		h.mu.unlock()
		return nil, ErrCapacity
	}
	sessionID := fmt.Sprintf("session-%d", h.nextSession.Add(1))
	session, err := newSession(h, runID, sessionID, definition, options)
	if err != nil {
		h.mu.unlock()
		return nil, err
	}
	h.sessions[runID] = session
	h.mu.unlock()

	lease, err := h.registry.acquire(ctx, runID, definition.registrations, definition.hasFault, func(queue string, failure error) {
		session.workerFailed(queue, failure)
	})
	if err != nil {
		cleanupCtx, cancel := h.cleanupContext()
		cleanupErr := h.removeSession(cleanupCtx, session, false)
		cancel()
		return nil, errors.Join(err, cleanupErr)
	}
	session.workers = lease
	return session, nil
}

func (h *Driver) cleanupContext() (context.Context, context.CancelFunc) {
	timeout := h.options.workerOptions.WorkerStopTimeout
	if timeout <= 0 {
		timeout = defaultCleanupTimeout
	}
	return context.WithTimeout(context.Background(), timeout)
}

func (h *Driver) Close(ctx context.Context) error {
	if h == nil || ctx == nil {
		return ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	h.options.completion.close()
	return nil
}

// The cleanup graph runs in the controller context and may carry a fault of its own, so it is
// bound here too; Validate and Open would otherwise disagree about which queues a fault names.
func (h *Driver) prepareDefinition(program testpilot.PreparedProgram) (programDefinition, error) {
	plans := program.Entrypoints()
	if cleanup, ok := program.Cleanup(); ok {
		plans = append(plans, cleanup)
	}
	return h.prepareDefinitionResources(program.Snapshot(), plans, program.Roles(), true)
}

func (h *Driver) prepareDefinitionPlans(snapshot *testpilotspb.Program, plans []testpilot.EntrypointPlan) (programDefinition, error) {
	return h.prepareDefinitionResources(snapshot, plans, nil, true)
}

func (h *Driver) prepareDefinitionResources(snapshot *testpilotspb.Program, plans []testpilot.EntrypointPlan, preparedRoles []testpilot.PreparedRole, requireWorker bool) (programDefinition, error) {
	if snapshot == nil || snapshot.GetLimits() == nil {
		return programDefinition{}, ErrInvalid
	}
	roles := preparedRolesByID(preparedRoles)
	definition := programDefinition{snapshot: snapshot, entries: make(map[string]entryDefinition), endpoints: make(map[string]string), queueWorkflows: make(map[string]map[string]struct{}), faultQueues: make(map[string]string)}
	if err := h.validateSymbolicRoles(roles, requireWorker); err != nil {
		return programDefinition{}, err
	}
	queueNexus := make(map[string]map[nexusRegistration]struct{})
	for _, plan := range plans {
		entry, relevant, err := h.boundEntry(plan, roles)
		if err != nil {
			return programDefinition{}, err
		}
		if err := h.addInstructionBindings(&definition, plan, roles, snapshot); err != nil {
			return programDefinition{}, err
		}
		if !relevant {
			continue
		}
		if err := definition.addEntry(entry, queueNexus); err != nil {
			return programDefinition{}, err
		}
	}
	if err := definition.addRegistrations(queueNexus); err != nil {
		return programDefinition{}, err
	}
	if requireWorker && (len(definition.entries) == 0 || len(definition.registrations) == 0) {
		return programDefinition{}, ErrInvalid
	}
	return definition, nil
}

func (h *Driver) boundEntry(plan testpilot.EntrypointPlan, roles map[string]testpilot.PreparedRole) (entryDefinition, bool, error) {
	activation := plan.Activation()
	entry := entryDefinition{plan: plan}
	var workerRole, queueRole string
	switch plan.Context() {
	case testpilotspb.ENTRYPOINT_KIND_WORKFLOW:
		binding := activation.GetWorkflow()
		if binding == nil {
			return entryDefinition{}, false, ErrInvalid
		}
		entry.workflowType = binding.GetWorkflowType()
		workerRole, queueRole = binding.GetWorkerRoleId(), binding.GetTaskQueueRoleId()
	case testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER:
		binding := activation.GetNexusHandler()
		if binding == nil {
			return entryDefinition{}, false, ErrInvalid
		}
		entry.service, entry.operation = binding.GetService(), binding.GetOperation()
		workerRole, queueRole = binding.GetWorkerRoleId(), binding.GetTaskQueueRoleId()
	default:
		return entryDefinition{}, false, nil
	}
	if workerRole != h.options.workerRoleID {
		return entryDefinition{}, false, ErrInvalid
	}
	workerBinding, workerOK := roles[workerRole]
	queueBinding, queueOK := roles[queueRole]
	if !workerOK || !queueOK || workerBinding.Kind != testpilotspb.ROLE_KIND_WORKER || queueBinding.Kind != testpilotspb.ROLE_KIND_TASK_QUEUE ||
		workerBinding.NamespaceBindingID == "" || queueBinding.NamespaceBindingID != workerBinding.NamespaceBindingID || workerBinding.Namespace == "" || queueBinding.Namespace != workerBinding.Namespace ||
		queueBinding.ResourceBindingID == "" || queueBinding.Resource == "" {
		return entryDefinition{}, false, ErrInvalid
	}
	entry.queue = queueBinding.Resource
	entry.namespace = workerBinding.Namespace
	return entry, true, nil
}

func (d *programDefinition) addEntry(entry entryDefinition, queueNexus map[string]map[nexusRegistration]struct{}) error {
	if _, duplicate := d.entries[entry.plan.ID()]; duplicate {
		return ErrRegistrationConflict
	}
	d.entries[entry.plan.ID()] = entry
	if entry.workflowType != "" {
		if d.queueWorkflows[entry.queue] == nil {
			d.queueWorkflows[entry.queue] = make(map[string]struct{})
		}
		if _, duplicate := d.queueWorkflows[entry.queue][entry.workflowType]; duplicate {
			return ErrRegistrationConflict
		}
		d.queueWorkflows[entry.queue][entry.workflowType] = struct{}{}
		return nil
	}
	if queueNexus[entry.queue] == nil {
		queueNexus[entry.queue] = make(map[nexusRegistration]struct{})
	}
	key := nexusRegistration{service: entry.service, operation: entry.operation}
	if _, duplicate := queueNexus[entry.queue][key]; duplicate {
		return ErrRegistrationConflict
	}
	queueNexus[entry.queue][key] = struct{}{}
	return nil
}

func (h *Driver) addInstructionBindings(definition *programDefinition, plan testpilot.EntrypointPlan, roles map[string]testpilot.PreparedRole, program *testpilotspb.Program) error {
	for _, instruction := range plan.Instructions() {
		source := instruction.Source().GetInstruction()
		if start := source.GetStartNexusOperation(); start != nil {
			role, ok := roles[start.GetEndpointRoleId()]
			if !ok || role.Kind != testpilotspb.ROLE_KIND_ENDPOINT || role.ResourceBindingID == "" || role.Resource == "" || h.profileRoleHasMethods(role.ID) {
				return ErrInvalid
			}
			endpoint := role.Resource
			if endpoint == "" {
				return ErrInvalid
			}
			definition.endpoints[start.GetEndpointRoleId()] = endpoint
		}
		if err := h.validateRPCBindings(instruction, roles, program); err != nil {
			return err
		}
		if response := source.GetRespondNexus(); response != nil && response.GetKind() == testpilotspb.NEXUS_RESPONSE_KIND_ASYNCHRONOUS {
			definition.hasAsync = true
		}
		if fault := source.GetInjectFault(); fault != nil {
			role, ok := roles[fault.GetRoleId()]
			if !ok || role.Kind != testpilotspb.ROLE_KIND_TASK_QUEUE || role.Resource == "" {
				return ErrInvalid
			}
			definition.faultQueues[fault.GetRoleId()] = role.Resource
			definition.hasFault = true
		}
	}
	return nil
}

func (h *Driver) validateSymbolicRoles(roles map[string]testpilot.PreparedRole, requireWorker bool) error {
	if requireWorker {
		workerRole, ok := roles[h.options.workerRoleID]
		if !ok || workerRole.Kind != testpilotspb.ROLE_KIND_WORKER || workerRole.NamespaceBindingID == "" || workerRole.Namespace == "" || workerRole.ResourceBindingID != "" || workerRole.Resource != "" {
			return ErrInvalid
		}
	}
	for _, role := range roles {
		switch role.Kind {
		case testpilotspb.ROLE_KIND_ENDPOINT:
			if role.NamespaceBindingID != "" || role.Namespace != "" || (role.ResourceBindingID != "" && (role.Resource == "" || h.profileRoleHasMethods(role.ID))) {
				return ErrInvalid
			}
		case testpilotspb.ROLE_KIND_WORKER:
			if role.NamespaceBindingID == "" || role.Namespace == "" || role.ResourceBindingID != "" || role.Resource != "" {
				return ErrInvalid
			}
		case testpilotspb.ROLE_KIND_TASK_QUEUE:
			if role.NamespaceBindingID == "" || role.Namespace == "" || role.ResourceBindingID == "" || role.Resource == "" {
				return ErrInvalid
			}
		case testpilotspb.ROLE_KIND_PARTICIPANT:
			if role.NamespaceBindingID != "" || role.Namespace != "" || role.ResourceBindingID != "" || role.Resource != "" {
				return ErrInvalid
			}
		default:
			return ErrInvalid
		}
	}
	return nil
}

func (h *Driver) profileRoleHasMethods(roleID string) bool {
	for _, role := range h.options.profile.Roles {
		if role.ID == roleID {
			return len(role.Methods) != 0
		}
	}
	return false
}

func (h *Driver) validateRPCBindings(instruction testpilot.InstructionPlan, roles map[string]testpilot.PreparedRole, program *testpilotspb.Program) error {
	invoke := instruction.Source().GetInstruction().GetInvokeRpc()
	if invoke == nil {
		return nil
	}
	endpoint, ok := roles[invoke.GetEndpointRoleId()]
	if !ok || endpoint.Kind != testpilotspb.ROLE_KIND_ENDPOINT || endpoint.ResourceBindingID != "" || endpoint.Resource != "" {
		return ErrInvalid
	}
	workerRole := roles[h.options.workerRoleID]
	switch invoke.GetMethod() {
	case startWorkflowMethod:
		queueRole, ok := reservedWorkflowQueueRole(instruction.Source(), program)
		if !ok {
			return ErrInvalid
		}
		queue, ok := roles[queueRole]
		if !ok || queue.Kind != testpilotspb.ROLE_KIND_TASK_QUEUE || queue.NamespaceBindingID != workerRole.NamespaceBindingID {
			return ErrInvalid
		}
		if !assignmentUsesBinding(invoke.GetRequestAssignments(), []string{"namespace"}, workerRole.NamespaceBindingID) ||
			!assignmentUsesBinding(invoke.GetRequestAssignments(), []string{"task_queue", "name"}, queue.ResourceBindingID) {
			return ErrInvalid
		}
	case getHistoryMethod:
		if !assignmentUsesBinding(invoke.GetRequestAssignments(), []string{"namespace"}, workerRole.NamespaceBindingID) {
			return ErrInvalid
		}
	default:
		return nil
	}
	return nil
}

func reservedWorkflowQueueRole(instruction *testpilotspb.InstructionDefinition, program *testpilotspb.Program) (string, bool) {
	entrypointID := ""
	for _, reservation := range instruction.GetActivationReservations() {
		if reservation.GetCount() != 1 {
			continue
		}
		for _, entrypoint := range program.GetEntrypoints() {
			if entrypoint.GetEntrypointId() == reservation.GetEntrypointId() && entrypoint.GetWorkflow() != nil {
				if entrypointID != "" {
					return "", false
				}
				entrypointID = entrypoint.GetEntrypointId()
			}
		}
	}
	if entrypointID == "" {
		return "", false
	}
	for _, entrypoint := range program.GetEntrypoints() {
		if entrypoint.GetEntrypointId() == entrypointID {
			return entrypoint.GetWorkflow().GetTaskQueueRoleId(), true
		}
	}
	return "", false
}

func assignmentUsesBinding(assignments []*testpilotspb.RequestAssignment, fields []string, bindingID string) bool {
	if bindingID == "" {
		return false
	}
	for _, assignment := range assignments {
		if assignment == nil || assignment.GetTarget() == nil || len(assignment.GetTarget().GetSegments()) != len(fields) {
			continue
		}
		matches := true
		for i, segment := range assignment.GetTarget().GetSegments() {
			if segment.GetField() != fields[i] || segment.GetSelector() != nil {
				matches = false
				break
			}
		}
		if matches && assignment.GetValue().GetEnvironment().GetBindingId() == bindingID {
			return true
		}
	}
	return false
}

func preparedRolesByID(roles []testpilot.PreparedRole) map[string]testpilot.PreparedRole {
	result := make(map[string]testpilot.PreparedRole, len(roles))
	for _, role := range roles {
		result[role.ID] = role
	}
	return result
}

func (d *programDefinition) addRegistrations(queueNexus map[string]map[nexusRegistration]struct{}) error {
	queues := make(map[string]struct{})
	for queue := range d.queueWorkflows {
		queues[queue] = struct{}{}
	}
	for queue := range queueNexus {
		queues[queue] = struct{}{}
	}
	for queue := range queues {
		registration, err := (queueRegistration{queue: queue, workflows: setKeys(d.queueWorkflows[queue]), nexus: nexusSetKeys(queueNexus[queue])}).canonical()
		if err != nil {
			return err
		}
		d.registrations = append(d.registrations, registration)
	}
	slices.SortFunc(d.registrations, func(left, right queueRegistration) int { return cmp.Compare(left.queue, right.queue) })
	return nil
}

func setKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	return result
}

func nexusSetKeys(values map[nexusRegistration]struct{}) []nexusRegistration {
	result := make([]nexusRegistration, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	return result
}

func boundedInt(value int64) int {
	maximum := int64(^uint(0) >> 1)
	if value <= 0 || value > maximum {
		return 0
	}
	return int(value)
}

func nilValue(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
