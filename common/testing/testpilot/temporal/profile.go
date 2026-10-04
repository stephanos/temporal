package temporal

import (
	"cmp"
	"slices"
	"strings"

	enumspb "go.temporal.io/api/enums/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/primitive"
	workerhost "go.temporal.io/server/common/testing/testpilot/temporal/worker"
)

// Environment names the physical resources one Case's symbolic bindings resolve to, plus the
// identity the derived Profile carries. A Case that binds no Nexus endpoint leaves it empty.
// DynamicConfig is the dynamic configuration the environment runs under -- a machine's setup
// parameters bound through the realization's keys, and the value of the switch a functional set
// repeats over -- keyed by the setting's key and valued by its text; the derived Profile records it.
type Environment struct {
	Identity  string
	Namespace string
	TaskQueue string
	// HandlerTaskQueue is the queue a Nexus handler entrypoint polls when the Program binds it apart
	// from the workflow's, so a fault on the handler's worker leaves the caller's running; a Program
	// that binds one queue leaves it empty.
	HandlerTaskQueue string
	NexusEndpoint    string
	DynamicConfig    map[string]string
	// DeliveryControl says the environment runs the server and can hold a delivery inside it.
	DeliveryControl bool
	// BoundScale is how much slower this environment is than the one the Case's wait bounds were
	// declared for; zero runs them as declared.
	BoundScale testpilot.BoundScale
}

// HandlerTaskQueueBindingID names the resource binding of the task-queue role a Nexus handler
// entrypoint polls when that role differs from every workflow entrypoint's, or "" when the Program
// binds one queue for both. A caller provisioning the Case's Nexus endpoint routes it to that queue.
func HandlerTaskQueueBindingID(program *testpilotspb.Program) string {
	workflowQueues := map[string]bool{}
	handlerQueues := map[string]bool{}
	for _, entrypoint := range program.GetEntrypoints() {
		if workflow := entrypoint.GetWorkflow(); workflow != nil {
			workflowQueues[workflow.GetTaskQueueRoleId()] = true
		}
		if handler := entrypoint.GetNexusHandler(); handler != nil {
			handlerQueues[handler.GetTaskQueueRoleId()] = true
		}
	}
	for _, role := range program.GetRoles() {
		if role.GetKind() == testpilotspb.ROLE_KIND_TASK_QUEUE && handlerQueues[role.GetRoleId()] && !workflowQueues[role.GetRoleId()] {
			return role.GetResourceBindingId()
		}
	}
	return ""
}

// configurationOf records the environment's dynamic configuration in the canonical spelling the
// catalog uses -- lower-case keys, sorted -- so two environments that spell one key differently
// derive one Profile. An empty key or value is an error rather than a silently dropped setting.
func configurationOf(environment Environment) ([]testpilot.ConfigurationValue, error) {
	if len(environment.DynamicConfig) == 0 {
		return nil, nil
	}
	values := make([]testpilot.ConfigurationValue, 0, len(environment.DynamicConfig))
	for key, value := range environment.DynamicConfig {
		if key == "" || value == "" {
			return nil, ErrInvalid
		}
		values = append(values, testpilot.ConfigurationValue{Key: strings.ToLower(key), Value: value})
	}
	slices.SortFunc(values, func(a, b testpilot.ConfigurationValue) int { return cmp.Compare(a.Key, b.Key) })
	for i := 1; i < len(values); i++ {
		if values[i-1].Key == values[i].Key {
			return nil, ErrInvalid
		}
	}
	return values, nil
}

// DefaultInstructionLimits returns the limits a Temporal Profile gives an instruction that writes
// none: the most common timeout and attempts across the checked-in Temporal Cases when the defaults
// were introduced. An instruction with any other value writes it.
func DefaultInstructionLimits() testpilot.InstructionDefaults {
	return testpilot.InstructionDefaults{TimeoutMilliseconds: 10000, MaxAttempts: 1}
}

// DefaultCeilings returns fresh copies of the Temporal Profile's resource ceilings: the Program,
// Contract and correlated limits every Temporal Profile admits a Case under. A Case declares none of
// them. Each ceiling is the largest value any checked-in Temporal Case declared when the bounds moved
// out of the Case, so no such Case is refused.
func DefaultCeilings() (*testpilotspb.ProgramLimits, *testpilotspb.ContractLimits, *testpilotspb.CorrelatedLimits) {
	return &testpilotspb.ProgramLimits{
		MaxEntrypoints: 4, MaxNodes: 16, MaxEdges: 24, MaxActivations: 8, MaxAttempts: 16,
		MaxRunEvents: 512, MaxExpressionDepth: 12, MaxPathFanout: 32,
		MaxRequestBytes: 32768, MaxResponseBytes: 8192,
		MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 20000,
		MaxInstructionEmittedEvents: 128, MaxInstructionResponseBytes: 8192,
	}, &testpilotspb.ContractLimits{
		MaxRules: 4, MaxStates: 16, MaxTransitions: 64, MaxExpressionDepth: 12,
		MaxWorkPerEvent: 4000000, MaxTotalWork: 1000000000, MaxCaptures: 64, MaxCaptureBytes: 65536,
	}, &testpilotspb.CorrelatedLimits{
		MaxEvents: 64, MaxBuffered: 32, MaxKeys: 8, MaxSupport: 256, MaxProjectionWork: 1000000000,
		MaxEventBytes: 512, MaxSemanticTransitions: 32, MaxObligations: 16, MaxObligationWork: 100000000,
		MaxCaptures: 16, MaxCorrelationDepth: 2,
	}
}

// DeriveProfile returns the minimal authorization the Case implies: the roles it declares, the
// methods it invokes, a reservation carrier for each StartWorkflowExecution an ordinary controller
// invokes when the Program has workflow, Nexus-handler or workflow-scheduled activity entrypoints
// to reserve and for each StartActivityExecution it invokes when the Program has other activity
// entrypoints to reserve, the
// opcodes its instructions require, the environment values its referenced bindings resolve to, and
// the dynamic configuration and bound scale the environment runs under. Nothing is widened beyond
// what the Case references, and anything the Case names that the catalog does not know is an error
// rather than a silently authorized surface. Its resource ceilings are DefaultCeilings and its instruction
// defaults DefaultInstructionLimits.
//
// The Profile stays an authorization snapshot, so the derived value is returned for the caller to
// review and tighten before Prepare rather than applied on its behalf.
func DeriveProfile(source *testpilotspb.Case, catalog *testpilot.Catalog, environment Environment) (testpilot.ProfileSpec, error) {
	program := source.GetProgram()
	if program == nil || catalog == nil || environment.Identity == "" {
		return testpilot.ProfileSpec{}, ErrInvalid
	}
	contexts, err := entrypointKinds(program)
	if err != nil {
		return testpilot.ProfileSpec{}, err
	}
	usage, err := deriveUsage(program, contexts, catalog)
	if err != nil {
		return testpilot.ProfileSpec{}, err
	}
	roles, err := deriveRoles(program, usage)
	if err != nil {
		return testpilot.ProfileSpec{}, err
	}
	bindings, err := deriveBindings(program, environment)
	if err != nil {
		return testpilot.ProfileSpec{}, err
	}
	configuration, err := configurationOf(environment)
	if err != nil {
		return testpilot.ProfileSpec{}, err
	}
	programLimits, contractLimits, correlatedLimits := DefaultCeilings()
	return testpilot.ProfileSpec{
		Identity:            environment.Identity,
		Catalog:             catalog,
		Roles:               roles,
		Opcodes:             usage.authorizedOpcodes(),
		CommandTypes:        usage.authorizedCommandTypes(),
		EnvironmentBindings: bindings,
		Configuration:       configuration,
		ProgramLimits:       programLimits,
		ContractLimits:      contractLimits,
		CorrelatedLimits:    correlatedLimits,
		InstructionDefaults: DefaultInstructionLimits(),
		BoundScale:          environment.BoundScale,
		DeliveryControl:     environment.DeliveryControl,
	}, nil
}

func entrypointKinds(program *testpilotspb.Program) (map[string]testpilot.EntrypointKind, error) {
	contexts := make(map[string]testpilot.EntrypointKind, len(program.GetEntrypoints()))
	for _, entrypoint := range program.GetEntrypoints() {
		kind := testpilot.EntrypointKindOf(entrypoint)
		if kind == 0 {
			return nil, ErrInvalid
		}
		if _, duplicate := contexts[entrypoint.GetEntrypointId()]; duplicate {
			return nil, ErrInvalid
		}
		contexts[entrypoint.GetEntrypointId()] = kind
	}
	return contexts, nil
}

// carrierKey names one reservation carrier: the method, on the endpoint role, whose instructions
// reserve activations.
type carrierKey struct{ role, method string }

// programUsage is what the Case actually references, in the order it references it. Order is kept
// so a derived Profile compares equal to a hand-written one rather than only equivalent.
type programUsage struct {
	methods      map[string][]string
	methodSeen   map[carrierKey]bool
	carrierOrder map[string][]string
	shapes       map[carrierKey]map[testpilot.EntrypointKind]int64
	opcodes      map[testpilot.Opcode]bool
	commandTypes map[enumspb.CommandType]bool
	// reservable counts the activations a carrier reserves: one of each workflow and Nexus-handler
	// entrypoint, and one per instruction of each activity entrypoint no workflow schedules, whose
	// script declares its attempts. scheduledAttempts counts the instructions of the activity
	// entrypoints a workflow schedules, which the workflow's start carries.
	reservable        map[testpilot.EntrypointKind]int64
	scheduledAttempts int64
	// evidence is the Program's declarations, which give a ReadEvidence poll its method.
	evidence map[string]*testpilotspb.EvidenceDeclaration
}

func (u *programUsage) authorizedOpcodes() []testpilot.Opcode {
	result := make([]testpilot.Opcode, 0, len(u.opcodes))
	for opcode := testpilot.InvokeRPC; opcode <= testpilot.MaxOpcode; opcode++ {
		if u.opcodes[opcode] {
			result = append(result, opcode)
		}
	}
	return result
}

// authorizedCommandTypes are the command types the Case's workflow commands carry that the worker
// Driver realizes, in enum order. A command type the Driver does not realize is left out, so the
// Case rejects at preparation as one the Profile does not admit, rather than widened.
func (u *programUsage) authorizedCommandTypes() []enumspb.CommandType {
	var result []enumspb.CommandType
	for _, commandType := range workerhost.CommandTypes() {
		if u.commandTypes[commandType] {
			result = append(result, commandType)
		}
	}
	slices.Sort(result)
	return result
}

func deriveUsage(program *testpilotspb.Program, contexts map[string]testpilot.EntrypointKind, catalog *testpilot.Catalog) (*programUsage, error) {
	usage := &programUsage{
		methods:      map[string][]string{},
		methodSeen:   map[carrierKey]bool{},
		carrierOrder: map[string][]string{},
		shapes:       map[carrierKey]map[testpilot.EntrypointKind]int64{},
		opcodes:      map[testpilot.Opcode]bool{},
		commandTypes: map[enumspb.CommandType]bool{},
		reservable:   map[testpilot.EntrypointKind]int64{},
		evidence:     map[string]*testpilotspb.EvidenceDeclaration{},
	}
	scheduled := scheduledActivities(program)
	for _, entrypoint := range program.GetEntrypoints() {
		switch kind := contexts[entrypoint.GetEntrypointId()]; kind {
		case testpilot.WorkflowEntrypoint, testpilot.NexusHandlerEntrypoint:
			usage.reservable[kind]++
		case testpilot.ActivityEntrypoint:
			activity := entrypoint.GetActivity()
			if scheduled[scheduledActivity{activityType: activity.GetActivityType(), queueRole: activity.GetTaskQueueRoleId()}] {
				usage.scheduledAttempts += int64(len(entrypoint.GetInstructions()))
			} else {
				usage.reservable[kind] += int64(len(entrypoint.GetInstructions()))
			}
		default:
		}
	}
	for _, declaration := range program.GetEvidence() {
		usage.evidence[declaration.GetEvidenceId()] = declaration
	}
	for _, entrypoint := range program.GetEntrypoints() {
		controller := contexts[entrypoint.GetEntrypointId()] == testpilot.ControllerEntrypoint
		for _, instruction := range entrypoint.GetInstructions() {
			if err := usage.add(instruction, controller, catalog); err != nil {
				return nil, err
			}
		}
	}
	for _, instruction := range program.GetCleanup().GetInstructions() {
		if err := usage.add(instruction, false, catalog); err != nil {
			return nil, err
		}
	}
	return usage, nil
}

// carried names the reservation carriers the Temporal Driver realizes and the entrypoint kinds each
// delivers: a workflow start carries the reservations of the workflow it starts and of the Nexus
// handlers and activities that workflow schedules, and an activity start the reservations of the
// activity it starts; an activity's are one per attempt.
var carried = map[string][]testpilot.EntrypointKind{
	primitive.StartWorkflowPath: {testpilot.WorkflowEntrypoint, testpilot.NexusHandlerEntrypoint, testpilot.ActivityEntrypoint},
	delivery.StartActivityPath:  {testpilot.ActivityEntrypoint},
}

// scheduledActivity names an activity a workflow's schedule command reaches: its type on its
// task-queue role.
type scheduledActivity struct{ activityType, queueRole string }

// scheduledActivities are the activities the Program's workflows schedule.
func scheduledActivities(program *testpilotspb.Program) map[scheduledActivity]bool {
	scheduled := map[scheduledActivity]bool{}
	for _, entrypoint := range program.GetEntrypoints() {
		if entrypoint.GetWorkflow() == nil {
			continue
		}
		for _, instruction := range entrypoint.GetInstructions() {
			if attributes := instruction.GetInstruction().GetWorkflowCommand().GetCommand().GetScheduleActivityTaskCommandAttributes(); attributes != nil {
				scheduled[scheduledActivity{activityType: attributes.GetActivityType().GetName(), queueRole: attributes.GetTaskQueue().GetName()}] = true
			}
		}
	}
	return scheduled
}

// reservableBy is how many activations of the kind a carrier of the method reserves: a workflow
// start reserves the attempts of the activities its workflow schedules, and an activity start those
// of the others.
func (u *programUsage) reservableBy(method string, kind testpilot.EntrypointKind) int64 {
	if kind == testpilot.ActivityEntrypoint && method == primitive.StartWorkflowPath {
		return u.scheduledAttempts
	}
	return u.reservable[kind]
}

// add records one instruction. An ordinary controller's StartWorkflowExecution or
// StartActivityExecution is a reservation carrier, and preparation derives its reservations from
// the carrier's shapes: one activation of each entrypoint of an admitted kind.
func (u *programUsage) add(instruction *testpilotspb.InstructionNode, controller bool, catalog *testpilot.Catalog) error {
	opcode := testpilot.InstructionOpcode(instruction.GetInstruction())
	if opcode == 0 {
		return ErrInvalid
	}
	u.opcodes[opcode] = true
	if command := instruction.GetInstruction().GetWorkflowCommand(); command != nil {
		u.commandTypes[command.GetCommand().GetCommandType()] = true
	}
	rpc := instruction.GetInstruction().GetInvokeRpc()
	if read := instruction.GetInstruction().GetReadEvidence(); read != nil {
		declaration := u.evidence[read.GetEvidenceId()]
		if declaration.GetRead() == nil {
			return ErrInvalid
		}
		rpc = &testpilotspb.InvokeRpc{EndpointRoleId: read.GetEndpointRoleId(), Method: declaration.GetRead().GetMethod()}
	}
	if rpc == nil {
		return nil
	}
	if err := catalog.CheckMethod(rpc.GetMethod()); err != nil {
		return err
	}
	key := carrierKey{role: rpc.GetEndpointRoleId(), method: rpc.GetMethod()}
	if !u.methodSeen[key] {
		u.methodSeen[key] = true
		u.methods[key.role] = append(u.methods[key.role], key.method)
	}
	if !controller || u.shapes[key] != nil {
		return nil
	}
	shapes := map[testpilot.EntrypointKind]int64{}
	for _, kind := range carried[key.method] {
		if count := u.reservableBy(key.method, kind); count > 0 {
			shapes[kind] = count
		}
	}
	if len(shapes) == 0 {
		return nil
	}
	u.shapes[key] = shapes
	u.carrierOrder[key.role] = append(u.carrierOrder[key.role], key.method)
	return nil
}

func deriveRoles(program *testpilotspb.Program, usage *programUsage) ([]testpilot.RolePolicy, error) {
	roles := make([]testpilot.RolePolicy, 0, len(program.GetRoles()))
	seen := make(map[string]struct{}, len(program.GetRoles()))
	for _, role := range program.GetRoles() {
		if role.GetKind() < testpilotspb.ROLE_KIND_ENDPOINT || role.GetKind() > testpilotspb.ROLE_KIND_PARTICIPANT {
			return nil, ErrInvalid
		}
		if _, duplicate := seen[role.GetRoleId()]; duplicate {
			return nil, ErrInvalid
		}
		seen[role.GetRoleId()] = struct{}{}
		policy := testpilot.RolePolicy{ID: role.GetRoleId(), Kind: role.GetKind(), Methods: usage.methods[role.GetRoleId()]}
		for _, method := range usage.carrierOrder[role.GetRoleId()] {
			counts := usage.shapes[carrierKey{role: role.GetRoleId(), method: method}]
			shapes := make([]testpilot.ReservationCarrierShape, 0, len(counts))
			for kind := testpilot.ControllerEntrypoint; kind <= testpilot.MaxEntrypointKind; kind++ {
				if count := counts[kind]; count > 0 {
					shapes = append(shapes, testpilot.ReservationCarrierShape{Kind: kind, MaximumCount: count})
				}
			}
			policy.ReservationCarriers = append(policy.ReservationCarriers, testpilot.ReservationCarrierPolicy{Method: method, Shapes: shapes})
		}
		roles = append(roles, policy)
	}
	// A method attributed to a role the Program never declared would authorize a surface the Case
	// cannot reach, so it is an error rather than a dropped entry.
	for role := range usage.methods {
		if _, declared := seen[role]; !declared {
			return nil, ErrInvalid
		}
	}
	return roles, nil
}

// deriveBindings resolves each binding the Program references through the role that references it,
// in the order preparation derives them. A binding no role claims has no derivable value, so it rejects
// rather than resolving to empty.
func deriveBindings(program *testpilotspb.Program, environment Environment) ([]testpilot.EnvironmentBinding, error) {
	values := map[string]string{}
	handlerQueue := HandlerTaskQueueBindingID(program)
	for _, role := range program.GetRoles() {
		switch role.GetKind() {
		case testpilotspb.ROLE_KIND_WORKER, testpilotspb.ROLE_KIND_TASK_QUEUE:
			if id := role.GetNamespaceBindingId(); id != "" {
				values[id] = environment.Namespace
			}
			if id := role.GetResourceBindingId(); id != "" && role.GetKind() == testpilotspb.ROLE_KIND_TASK_QUEUE {
				if id == handlerQueue {
					values[id] = environment.HandlerTaskQueue
				} else {
					values[id] = environment.TaskQueue
				}
			}
		case testpilotspb.ROLE_KIND_ENDPOINT:
			if id := role.GetResourceBindingId(); id != "" {
				values[id] = environment.NexusEndpoint
			}
		default:
		}
	}
	ids := testpilot.EnvironmentBindingIDs(program)
	bindings := make([]testpilot.EnvironmentBinding, 0, len(ids))
	for _, id := range ids {
		value, resolved := values[id]
		if !resolved || value == "" {
			return nil, ErrInvalid
		}
		bindings = append(bindings, testpilot.EnvironmentBinding{ID: id, Value: value})
	}
	return bindings, nil
}
