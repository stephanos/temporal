package worker

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire"
)

type Host struct {
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
	profile        umpire.ProfileSpec
	namespace      string
	workerRoleID   string
	taskQueues     map[string]string
	endpoints      map[string]string
	client         client.Client
	workerOptions  worker.Options
	sessionOptions func(context.Context, string) (SessionOptions, error)
	maximum        int
	diagnostics    int
	requestBytes   int64
	now            func() time.Time
}

func New(options Options) (*Host, error) {
	if nilValue(options.Client) || options.Namespace == "" || options.WorkerRoleID == "" || !validWorkerProfile(options.Profile) {
		return nil, ErrInvalid
	}
	limits := options.Profile.ProgramLimits
	maximum, diagnostics := boundedInt(limits.GetMaxActivations()), min(boundedInt(limits.GetMaxRunEvents()), 64)
	taskQueues, err := bindingMap(options.TaskQueues)
	if err != nil || len(taskQueues) == 0 {
		return nil, ErrInvalid
	}
	endpoints, err := bindingMap(options.Endpoints)
	if err != nil {
		return nil, ErrInvalid
	}
	h := &Host{
		mu:             newContextMutex(),
		sessions:       make(map[string]*Session),
		tombstones:     make([]*Session, 0, diagnostics),
		workflowRoutes: make(map[workflowRouteIndex][]*Session),
		nexusRoutes:    make(map[nexusRouteIndex][]*Session),
		options: hostOptions{
			profile: options.Profile.Snapshot(), namespace: options.Namespace, workerRoleID: options.WorkerRoleID,
			taskQueues: taskQueues, endpoints: endpoints, client: options.Client,
			workerOptions:  worker.Options{WorkerStopTimeout: options.WorkerStopTimeout},
			sessionOptions: options.SessionOptions, maximum: maximum, diagnostics: diagnostics,
			requestBytes: limits.GetMaxRequestBytes(), now: time.Now,
		},
	}
	h.registry = newWorkerRegistry(maximum, h.newSDKWorker)
	return h, nil
}

func validWorkerProfile(profile umpire.ProfileSpec) bool {
	limits := profile.ProgramLimits
	ceiling := &umpirespb.ProgramLimits{
		MaxEntrypoints: 10000, MaxNodes: 10000, MaxEdges: 100000, MaxActivations: 100000,
		MaxAttempts: 100000, MaxRunEvents: 100000, MaxExpressionDepth: 64, MaxPathFanout: 10000,
		MaxRequestBytes: 16 << 20, MaxResponseBytes: 16 << 20,
		MaxTotalDurationMilliseconds: 86400000, MaxCleanupDurationMilliseconds: 86400000,
	}
	if profile.Identity == "" || len(profile.Identity) > 256 || profile.Catalog == nil || profile.Catalog.Identity() == "" || limits == nil || len(profile.Capabilities) > 7 || len(profile.Roles) > 10000 {
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

func (h *Host) Identity(ctx context.Context) (umpire.HostIdentity, error) {
	if h == nil || ctx == nil {
		return umpire.HostIdentity{}, ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return umpire.HostIdentity{}, err
	}
	return umpire.HostIdentity{Profile: h.options.profile.Identity, Catalog: h.options.profile.Catalog.Identity()}, nil
}

func (h *Host) Open(ctx context.Context, runID string, program umpire.PreparedProgram) (umpire.Session, error) {
	if h == nil || ctx == nil || h.options.sessionOptions == nil {
		return nil, ErrInvalid
	}
	options, err := h.options.sessionOptions(ctx, runID)
	if err != nil {
		return nil, err
	}
	return h.OpenSession(ctx, runID, program, options)
}

func (h *Host) OpenSession(ctx context.Context, runID string, program umpire.PreparedProgram, options SessionOptions) (*Session, error) {
	if h == nil || ctx == nil || runID == "" || nilValue(options.Bridge) {
		return nil, ErrInvalid
	}
	definition, err := h.prepareDefinition(program)
	if err != nil {
		return nil, err
	}
	if definition.hasAsync && options.NewCompletionCapability == nil {
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

	release, err := h.registry.acquire(ctx, runID, definition.registrations, func(queue string, failure error) {
		session.workerFailed(queue, failure)
	})
	if err != nil {
		cleanupCtx, cancel := h.cleanupContext()
		cleanupErr := h.removeSession(cleanupCtx, session, false)
		cancel()
		return nil, errors.Join(err, cleanupErr)
	}
	session.releaseWorkers = release
	return session, nil
}

func (h *Host) cleanupContext() (context.Context, context.CancelFunc) {
	timeout := h.options.workerOptions.WorkerStopTimeout
	if timeout <= 0 {
		timeout = defaultCleanupTimeout
	}
	return context.WithTimeout(context.Background(), timeout)
}

func (h *Host) prepareDefinition(program umpire.PreparedProgram) (programDefinition, error) {
	return h.prepareDefinitionPlans(program.Snapshot(), program.Entrypoints())
}

func (h *Host) prepareDefinitionPlans(snapshot *umpirespb.Program, plans []umpire.EntrypointPlan) (programDefinition, error) {
	if snapshot == nil || snapshot.GetLimits() == nil {
		return programDefinition{}, ErrInvalid
	}
	definition := programDefinition{snapshot: snapshot, entries: make(map[string]entryDefinition), endpoints: make(map[string]string), queueWorkflows: make(map[string]map[string]struct{})}
	queueNexus := make(map[string]map[nexusRegistration]struct{})
	for _, plan := range plans {
		entry, relevant, err := h.boundEntry(plan)
		if err != nil {
			return programDefinition{}, err
		}
		if !relevant {
			continue
		}
		if err := definition.addEntry(entry, queueNexus); err != nil {
			return programDefinition{}, err
		}
		if err := h.addInstructionBindings(&definition, plan); err != nil {
			return programDefinition{}, err
		}
	}
	if err := definition.addRegistrations(queueNexus); err != nil {
		return programDefinition{}, err
	}
	if len(definition.entries) == 0 || len(definition.registrations) == 0 {
		return programDefinition{}, ErrInvalid
	}
	return definition, nil
}

func (h *Host) boundEntry(plan umpire.EntrypointPlan) (entryDefinition, bool, error) {
	activation := plan.Activation()
	entry := entryDefinition{plan: plan}
	var workerRole, queueRole string
	switch plan.Context() {
	case umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW:
		binding := activation.GetWorkflow()
		if binding == nil {
			return entryDefinition{}, false, ErrInvalid
		}
		entry.workflowType = binding.GetWorkflowType()
		workerRole, queueRole = binding.GetWorkerRoleId(), binding.GetTaskQueueRoleId()
	case umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER:
		binding := activation.GetNexusHandler()
		if binding == nil {
			return entryDefinition{}, false, ErrInvalid
		}
		entry.service, entry.operation = binding.GetService(), binding.GetOperation()
		workerRole, queueRole = binding.GetWorkerRoleId(), binding.GetTaskQueueRoleId()
	default:
		return entryDefinition{}, false, nil
	}
	entry.queue = h.options.taskQueues[queueRole]
	if workerRole != h.options.workerRoleID || entry.queue == "" {
		return entryDefinition{}, false, ErrInvalid
	}
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

func (h *Host) addInstructionBindings(definition *programDefinition, plan umpire.EntrypointPlan) error {
	for _, instruction := range plan.Instructions() {
		source := instruction.Source().GetInstruction()
		if start := source.GetStartNexusOperation(); start != nil {
			endpoint := h.options.endpoints[start.GetEndpointRoleId()]
			if endpoint == "" {
				return ErrInvalid
			}
			definition.endpoints[start.GetEndpointRoleId()] = endpoint
		}
		if response := source.GetRespondNexus(); response != nil && response.GetKind() == umpirespb.NEXUS_RESPONSE_KIND_ASYNCHRONOUS {
			definition.hasAsync = true
		}
	}
	return nil
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

func bindingMap(bindings []RoleBinding) (map[string]string, error) {
	result := make(map[string]string, len(bindings))
	for _, binding := range bindings {
		if strings.TrimSpace(binding.RoleID) != binding.RoleID || strings.TrimSpace(binding.Value) != binding.Value || binding.RoleID == "" || binding.Value == "" || result[binding.RoleID] != "" {
			return nil, ErrInvalid
		}
		result[binding.RoleID] = binding.Value
	}
	return result, nil
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
