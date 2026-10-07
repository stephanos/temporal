package worker

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/primitive"
)

const getHistoryMethod = "/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory"

type Driver struct {
	options        hostOptions
	registry       *workerRegistry
	mu             primitive.Mutex
	sessions       map[string]*Session
	tombstones     []*Session
	workflowRoutes map[delivery.WorkflowBinding][]*Session
	activityRoutes map[delivery.ActivityBinding][]*Session
	nexusRoutes    map[nexusRouteIndex][]*Session
	// scheduledRoutes index the Sessions whose workflows scheduled an activity, by the route its
	// schedule command carries.
	scheduledRoutes   map[scheduledRouteIndex][]*Session
	routeAssociations int
	nextSession       atomic.Uint64
}

type hostOptions struct {
	profile       testpilot.ProfileSpec
	workerRoleID  string
	client        client.Client
	workerOptions worker.Options
	maximum       int
	diagnostics   int
	requestBytes  int64
	now           func() time.Time
	completion    *completionTransport
	// activityClosed asks the server for the outcome of the standalone activity run and returns the
	// run the server reports closed. It is the only evidence that no further attempt of the activity
	// will be issued; nil means there is none.
	activityClosed func(ctx context.Context, namespace, activityID, activityRunID string) (string, error)
	// heartbeat records a heartbeat of the activity attempt whose context it is given, through
	// which alone the server tells the attempt that its cancellation is requested.
	heartbeat func(ctx context.Context)
}

func New(options Options) (*Driver, error) {
	if primitive.NilValue(options.Client) || options.WorkerRoleID == "" || !validWorkerProfile(options.Profile) {
		return nil, ErrInvalid
	}
	if _, err := options.Profile.BindingFingerprint(); err != nil {
		return nil, ErrInvalid
	}
	// The callback deadline is a duration ceiling, so it takes the scale preparation applied.
	limits := options.Profile.BoundScale.Ceilings(options.Profile.ProgramLimits)
	completion, err := newCompletionTransport(options.HTTPClient, options.SystemCallbackBaseURL, limits)
	if err != nil {
		return nil, err
	}
	maximum, diagnostics := boundedInt(limits.GetMaxActivations()), min(boundedInt(limits.GetMaxRunEvents()), 64)
	h := &Driver{
		mu:              primitive.NewMutex(),
		sessions:        make(map[string]*Session),
		tombstones:      make([]*Session, 0, diagnostics),
		workflowRoutes:  make(map[delivery.WorkflowBinding][]*Session),
		activityRoutes:  make(map[delivery.ActivityBinding][]*Session),
		nexusRoutes:     make(map[nexusRouteIndex][]*Session),
		scheduledRoutes: make(map[scheduledRouteIndex][]*Session),
		options: hostOptions{
			profile: options.Profile.Snapshot(), workerRoleID: options.WorkerRoleID, client: options.Client,
			// The SDK sends a heartbeat at most once per throttle interval, by default a share of
			// the activity's heartbeat timeout and half a minute without one. An attempt heartbeats
			// only to learn of a requested cancellation, so the interval is capped at that poll.
			workerOptions: worker.Options{WorkerStopTimeout: options.WorkerStopTimeout, MaxHeartbeatThrottleInterval: cancellationPoll},
			maximum:       maximum, diagnostics: diagnostics, requestBytes: limits.GetMaxRequestBytes(),
			now: time.Now, completion: completion, activityClosed: pollActivityClosed(options.Client),
			heartbeat: func(ctx context.Context) { activity.RecordHeartbeat(ctx) },
		},
	}
	h.registry = newWorkerRegistry(maximum, h.newSDKWorker)
	return h, nil
}

func validWorkerProfile(profile testpilot.ProfileSpec) bool {
	limits := profile.ProgramLimits
	if profile.Identity == "" || len(profile.Identity) > 256 || profile.Catalog == nil || profile.Catalog.Identity() == "" || limits == nil || len(profile.Opcodes) > int(testpilot.MaxOpcode) || len(profile.Roles) > 10000 {
		return false
	}
	if !testpilot.WithinProgramCeiling(profile.BoundScale.Ceilings(limits)) {
		return false
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
	plans := ProgramPlans(program)
	// A fault needs a worker to stop, so a Program that requests one is only realizable when it
	// also brings a worker. Validate applies the same rule Open does rather than admitting a
	// Program that could only fail at dispatch.
	requireWorker := primitive.HasWorkerEntrypoint(plans) || DeclaresFault(plans)
	_, err := h.prepareDefinitionResources(program.Snapshot(), program.Limits(), plans, program.Roles(), requireWorker)
	return err
}

// ProgramPlans is every plan the worker Driver binds. The cleanup graph runs in the controller
// context and may carry instructions of its own, so Validate, Open and the composite Driver all
// read the same list rather than each assembling it.
func ProgramPlans(program testpilot.PreparedProgram) []testpilot.EntrypointPlan {
	plans := program.Entrypoints()
	if cleanup, ok := program.Cleanup(); ok {
		plans = append(plans, cleanup)
	}
	return plans
}

// DeclaresFault reports whether any plan requests a deliberate worker outage. The composite Driver uses
// it to decide that a Program needs a worker Session at all, so both Drivers read one predicate.
func DeclaresFault(plans []testpilot.EntrypointPlan) bool {
	for _, plan := range plans {
		for _, instruction := range plan.Instructions() {
			if fault := instruction.Source().GetInstruction().GetInjectFault(); fault != nil && WorkerFault(fault.GetKind()) {
				return true
			}
		}
	}
	return false
}

func (h *Driver) OpenSession(ctx context.Context, runID string, program testpilot.PreparedProgram, options SessionOptions) (*Session, error) {
	if h == nil || ctx == nil || runID == "" || primitive.NilValue(options.Bridge) {
		return nil, ErrInvalid
	}
	definition, err := h.prepareDefinition(program)
	if err != nil {
		return nil, err
	}
	if definition.hasAsync && options.NewHandle == nil {
		return nil, ErrInvalid
	}
	if err := h.mu.LockContext(ctx, ErrInvalid); err != nil {
		return nil, err
	}
	if len(h.sessions) >= h.options.maximum || h.sessions[runID] != nil {
		h.mu.Unlock()
		return nil, ErrCapacity
	}
	sessionID := fmt.Sprintf("session-%d", h.nextSession.Add(1))
	session, err := newSession(h, runID, sessionID, definition, options)
	if err != nil {
		h.mu.Unlock()
		return nil, err
	}
	h.sessions[runID] = session
	h.mu.Unlock()

	outage, err := h.registry.acquireOutage(ctx, runID, definition.registrations, definition.outages, func(queue string, failure error) {
		session.workerFailed(queue, failure)
	})
	if err != nil {
		cleanupCtx, cancel := h.cleanupContext()
		cleanupErr := h.removeSession(cleanupCtx, session, false)
		cancel()
		return nil, errors.Join(err, cleanupErr)
	}
	session.outage = outage
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

func (h *Driver) prepareDefinition(program testpilot.PreparedProgram) (programDefinition, error) {
	return h.prepareDefinitionResources(program.Snapshot(), program.Limits(), ProgramPlans(program), program.Roles(), true)
}

func (h *Driver) prepareDefinitionResources(snapshot *testpilotspb.Program, limits *testpilotspb.ProgramLimits, plans []testpilot.EntrypointPlan, preparedRoles []testpilot.PreparedRole, requireWorker bool) (programDefinition, error) {
	if snapshot == nil || limits == nil {
		return programDefinition{}, ErrInvalid
	}
	roles := preparedRolesByID(preparedRoles)
	definition := programDefinition{snapshot: snapshot, limits: limits, entries: make(map[string]entryDefinition), endpoints: make(map[string]string), queues: make(map[string]string), queueWorkflows: make(map[string]map[string]struct{}), queueActivities: make(map[string]map[string]struct{})}
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
	if err := definition.addRegistrations(roles[h.options.workerRoleID].Namespace, queueNexus); err != nil {
		return programDefinition{}, err
	}
	registered := make(map[string]bool, len(definition.registrations))
	for _, registration := range definition.registrations {
		registered[registration.queue] = true
	}
	outages, err := PlanOutages(plans, roles, registered)
	if err != nil {
		return programDefinition{}, err
	}
	definition.outages = outages
	if requireWorker && (len(definition.entries) == 0 || len(definition.registrations) == 0) {
		return programDefinition{}, ErrInvalid
	}
	return definition, nil
}

func (h *Driver) boundEntry(plan testpilot.EntrypointPlan, roles map[string]testpilot.PreparedRole) (entryDefinition, bool, error) {
	activation := plan.Activation()
	entry := entryDefinition{plan: plan}
	var workerRole, queueRole string
	switch plan.Kind() {
	case testpilot.WorkflowEntrypoint:
		binding := activation.GetWorkflow()
		if binding == nil {
			return entryDefinition{}, false, ErrInvalid
		}
		entry.workflowType = binding.GetWorkflowType()
		workerRole, queueRole = binding.GetWorkerRoleId(), binding.GetTaskQueueRoleId()
	case testpilot.ActivityEntrypoint:
		binding := activation.GetActivity()
		if binding == nil {
			return entryDefinition{}, false, ErrInvalid
		}
		entry.activityType = binding.GetActivityType()
		workerRole, queueRole = binding.GetWorkerRoleId(), binding.GetTaskQueueRoleId()
	case testpilot.NexusHandlerEntrypoint:
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
	if entry.activityType != "" {
		if d.queueActivities[entry.queue] == nil {
			d.queueActivities[entry.queue] = make(map[string]struct{})
		}
		if _, duplicate := d.queueActivities[entry.queue][entry.activityType]; duplicate {
			return ErrRegistrationConflict
		}
		d.queueActivities[entry.queue][entry.activityType] = struct{}{}
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
		var endpointRole string
		if schedule := scheduleNexusOperation(source); schedule != nil {
			endpointRole = schedule.GetEndpoint()
		}
		if endpointRole != "" {
			role, ok := roles[endpointRole]
			if !ok || role.Kind != testpilotspb.ROLE_KIND_ENDPOINT || role.ResourceBindingID == "" || role.Resource == "" || h.profileRoleHasMethods(role.ID) {
				return ErrInvalid
			}
			definition.endpoints[endpointRole] = role.Resource
		}
		if schedule := scheduleActivity(source); schedule != nil {
			// A workflow schedules its activities in its own namespace.
			queueRole := schedule.GetTaskQueue().GetName()
			role, ok := roles[queueRole]
			if !ok || role.Kind != testpilotspb.ROLE_KIND_TASK_QUEUE || role.ResourceBindingID == "" || role.Resource == "" || role.Namespace != roles[h.options.workerRoleID].Namespace {
				return ErrInvalid
			}
			definition.queues[queueRole] = role.Resource
		}
		if err := h.validateRPCBindings(instruction, roles, program); err != nil {
			return err
		}
		if source.GetNexusHandlerReply().GetResponse().GetAsyncSuccess() != nil {
			definition.hasAsync = true
		}
	}
	return nil
}

func (h *Driver) validateSymbolicRoles(roles map[string]testpilot.PreparedRole, requireWorker bool) error {
	if requireWorker && roles[h.options.workerRoleID].Kind != testpilotspb.ROLE_KIND_WORKER {
		return ErrInvalid
	}
	for _, role := range roles {
		if role.Kind == testpilotspb.ROLE_KIND_ENDPOINT && role.ResourceBindingID != "" && h.profileRoleHasMethods(role.ID) {
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
	// A reservation rides only on a start the Driver carries it on (delivery.Carried): an activity
	// on the request that started it or the workflow that scheduled it, and an activity start
	// carries no other activation.
	reservations := instruction.Reservations()
	for _, reservation := range reservations {
		if !slices.Contains(delivery.Carried[invoke.GetMethod()], reservation.Kind) {
			return ErrInvalid
		}
	}
	switch invoke.GetMethod() {
	case primitive.StartWorkflowPath:
		queueRole, ok := reservedWorkflowQueueRole(instruction, program)
		if !ok {
			return ErrInvalid
		}
		return validateStartBindings(invoke, roles, workerRole, queueRole)
	case delivery.StartActivityPath:
		// A start that reserves no activation runs no script here and stays an ordinary call.
		if len(reservations) == 0 {
			return nil
		}
		queueRole, ok := reservedActivityQueueRole(instruction, program)
		if !ok {
			return ErrInvalid
		}
		return validateStartBindings(invoke, roles, workerRole, queueRole)
	case getHistoryMethod:
		if !assignmentUsesBinding(invoke.GetRequestAssignments(), []string{"namespace"}, workerRole.NamespaceBindingID) {
			return ErrInvalid
		}
	default:
		return nil
	}
	return nil
}

// validateStartBindings checks that a carried start names the worker's namespace and the reserved
// entrypoint's task queue by the binding identities the worker itself is bound through.
func validateStartBindings(invoke *testpilotspb.InvokeRpc, roles map[string]testpilot.PreparedRole, workerRole testpilot.PreparedRole, queueRole string) error {
	queue, ok := roles[queueRole]
	if !ok || queue.Kind != testpilotspb.ROLE_KIND_TASK_QUEUE || queue.NamespaceBindingID != workerRole.NamespaceBindingID {
		return ErrInvalid
	}
	if !assignmentUsesBinding(invoke.GetRequestAssignments(), []string{"namespace"}, workerRole.NamespaceBindingID) ||
		!assignmentUsesBinding(invoke.GetRequestAssignments(), []string{"task_queue", "name"}, queue.ResourceBindingID) {
		return ErrInvalid
	}
	return nil
}

// A workflow start activates its workflow once; an activity start reserves one activation per
// attempt of the activity it starts.
func reservedWorkflowQueueRole(instruction testpilot.InstructionPlan, program *testpilotspb.Program) (string, bool) {
	return reservedQueueRole(instruction, program, func(reservation testpilot.ReservationTopology, entrypoint *testpilotspb.Entrypoint) (string, bool) {
		return entrypoint.GetWorkflow().GetTaskQueueRoleId(), reservation.Count == 1 && entrypoint.GetWorkflow() != nil
	})
}

func reservedActivityQueueRole(instruction testpilot.InstructionPlan, program *testpilotspb.Program) (string, bool) {
	return reservedQueueRole(instruction, program, func(_ testpilot.ReservationTopology, entrypoint *testpilotspb.Entrypoint) (string, bool) {
		return entrypoint.GetActivity().GetTaskQueueRoleId(), entrypoint.GetActivity() != nil
	})
}

// reservedQueueRole is the task-queue role of the one entrypoint queueRole selects among those the
// instruction reserves; none or more than one is no answer.
func reservedQueueRole(instruction testpilot.InstructionPlan, program *testpilotspb.Program, queueRole func(testpilot.ReservationTopology, *testpilotspb.Entrypoint) (string, bool)) (string, bool) {
	result, found := "", false
	for _, reservation := range instruction.Reservations() {
		for _, entrypoint := range program.GetEntrypoints() {
			if entrypoint.GetEntrypointId() != reservation.EntrypointID {
				continue
			}
			if role, selected := queueRole(reservation, entrypoint); selected {
				if found {
					return "", false
				}
				result, found = role, true
			}
		}
	}
	return result, found
}

func assignmentUsesBinding(assignments []*testpilotspb.RequestAssignment, fields []string, bindingID string) bool {
	if bindingID == "" {
		return false
	}
	for _, assignment := range assignments {
		// A path of plain field names has exactly one spelling.
		if assignment.GetTarget() == strings.Join(fields, ".") && assignment.GetValue().GetReference().GetEnvironmentBindingId() == bindingID {
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

func (d *programDefinition) addRegistrations(namespace string, queueNexus map[string]map[nexusRegistration]struct{}) error {
	queues := make(map[string]struct{})
	for queue := range d.queueWorkflows {
		queues[queue] = struct{}{}
	}
	for queue := range d.queueActivities {
		queues[queue] = struct{}{}
	}
	for queue := range queueNexus {
		queues[queue] = struct{}{}
	}
	for queue := range queues {
		registration, err := (queueRegistration{namespace: namespace, queue: queue, workflows: slices.AppendSeq(make([]string, 0, len(d.queueWorkflows[queue])), maps.Keys(d.queueWorkflows[queue])), activities: slices.Collect(maps.Keys(d.queueActivities[queue])), nexus: slices.AppendSeq(make([]nexusRegistration, 0, len(queueNexus[queue])), maps.Keys(queueNexus[queue]))}).canonical()
		if err != nil {
			return err
		}
		d.registrations = append(d.registrations, registration)
	}
	slices.SortFunc(d.registrations, func(left, right queueRegistration) int { return cmp.Compare(left.queue, right.queue) })
	return nil
}

func boundedInt(value int64) int {
	maximum := int64(^uint(0) >> 1)
	if value <= 0 || value > maximum {
		return 0
	}
	return int(value)
}
