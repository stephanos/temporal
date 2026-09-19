package execution

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"unicode/utf8"

	enumspb "go.temporal.io/api/enums/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type admission struct {
	prepared         *PreparedProgram
	roles            map[string]testpilotspb.RoleKind
	allowed          map[string]contract.RolePolicy
	methods          map[string]map[string]bool
	carriers         map[string]map[string]contract.ReservationCarrierPolicy
	opcodes          map[contract.Opcode]bool
	commandTypes     map[enumspb.CommandType]bool
	bindingsRequired bool
	// environment holds the Profile's binding values, and bindings those of the Program's derived
	// binding graph.
	environment  map[string]string
	bindings     map[string]string
	observations map[string]ir.Type
	// outcomeTypes are the types of the outcome fields instructions produce.
	outcomeTypes struct{ status, text, any ir.Type }
	runID        ir.Type
	writers      map[string]slotWriter
	// Each declared evidence source, and the one instruction that may lift under it.
	evidenceSources map[string]contract.Coordinate
	graphIndex      map[string]*graph
	work            int64
}

func invalid(category ir.ErrorCategory, path, detail string) error {
	if len(path) > 256 {
		path = path[:256]
	}
	return &ir.Error{Category: category, Path: path, Detail: detail}
}
func validID(id string) bool {
	if len(id) == 0 || len(id) > 256 {
		return false
	}
	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '_', c == '-', c == '.':
		default:
			return false
		}
	}
	return true
}
func (a *admission) charge(count int64) error {
	if count < 0 || count > ir.DefaultLimits().Work-a.work {
		return invalid(ir.LimitExceeded, "program", "admission work ceiling exceeded")
	}
	a.work += count
	return nil
}

// Prepare performs static admission only; Contract semantics are admitted by verification.
func Prepare(source *testpilotspb.Case, catalog *ir.Catalog, policy Profile) (*PreparedProgram, error) {
	if catalog == nil {
		return nil, invalid(ir.Malformed, "catalog", "catalog is required")
	}
	if err := ir.CheckSurface(source, ir.DefaultLimits()); err != nil {
		return nil, err
	}
	if source.Version == nil || source.Version.Major != 1 || source.Version.Minor != 0 {
		return nil, invalid(ir.Unsupported, "version", "unsupported Case version")
	}
	if !validID(source.CaseId) || source.Program == nil || source.Contract == nil || !validID(source.Contract.ContractId) {
		return nil, invalid(ir.Malformed, "case", "Case identity, Program and Contract are required")
	}
	if int64(proto.Size(source)) > ir.DefaultLimits().Bytes {
		return nil, invalid(ir.LimitExceeded, "case", "Case byte ceiling exceeded")
	}
	if err := validateProvenance(source.Provenance); err != nil {
		return nil, err
	}
	prepared := &PreparedProgram{source: proto.CloneOf(source.Program), catalog: catalog, slots: map[string]ir.Type{}, carriers: map[carrierCoordinate]contract.ReservationCarrierPlan{}, roles: map[string]resolvedRole{}}
	a := &admission{prepared: prepared, roles: map[string]testpilotspb.RoleKind{}, allowed: map[string]contract.RolePolicy{}, methods: map[string]map[string]bool{}, carriers: map[string]map[string]contract.ReservationCarrierPolicy{}, opcodes: map[contract.Opcode]bool{}, commandTypes: map[enumspb.CommandType]bool{}, bindingsRequired: true, environment: map[string]string{}, bindings: map[string]string{}, observations: map[string]ir.Type{}, writers: map[string]slotWriter{}, evidenceSources: map[string]contract.Coordinate{}, graphIndex: map[string]*graph{}}
	for _, check := range []func() error{func() error { return a.bindPolicy(policy) }, a.bindSchemas, a.bindGraphs, a.bindInstructions, a.bindDataflow, a.deriveReservations, a.bindReservations, a.bindReservationCarriers} {
		if err := check(); err != nil {
			return nil, err
		}
	}
	return prepared, nil
}
func validateProvenance(provenance *testpilotspb.CaseProvenance) error {
	if provenance == nil {
		return nil
	}
	if !validID(provenance.ProducerId) {
		return invalid(ir.Malformed, "provenance", "invalid Producer identity")
	}
	return nil
}
func checkLimits(limits, ceiling *testpilotspb.ProgramLimits) error {
	if limits == nil || ceiling == nil {
		return invalid(ir.Malformed, "limits", "limits are required")
	}
	if err := ir.CheckSurface(limits, ir.DefaultLimits()); err != nil {
		return err
	}
	fields := limits.ProtoReflect().Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		value := limits.ProtoReflect().Get(field).Int()
		if value <= 0 || value > ceiling.ProtoReflect().Get(field).Int() {
			return invalid(ir.LimitExceeded, string(field.Name()), "limit is outside the positive Driver ceiling")
		}
	}
	return nil
}
func (a *admission) bindPolicy(policy Profile) error {
	if !validID(policy.Identity) || policy.CatalogIdentity != a.prepared.catalog.Identity() {
		return invalid(ir.Malformed, "policy", "Driver or catalog identity mismatch")
	}
	if err := checkLimits(policy.Limits, hardLimits()); err != nil {
		return err
	}
	if policy.Limits.MaxInstructionEmittedEvents > policy.Limits.MaxRunEvents {
		return invalid(ir.LimitExceeded, "max_instruction_emitted_events", "instruction ceiling exceeds the Program ceiling")
	}
	if policy.Limits.MaxInstructionResponseBytes > policy.Limits.MaxResponseBytes {
		return invalid(ir.LimitExceeded, "max_instruction_response_bytes", "instruction ceiling exceeds the Program ceiling")
	}
	if err := checkInstructionDefaults(policy.InstructionDefaults, policy.Limits); err != nil {
		return err
	}
	if len(policy.Roles) > 10000 || len(policy.Opcodes) > int(contract.MaxOpcode) || len(policy.CommandTypes) > len(enumspb.CommandType_name) || len(policy.EnvironmentBindings) > 10000 {
		return invalid(ir.LimitExceeded, "policy", "policy collection ceiling exceeded")
	}
	var environmentBytes int64
	for _, binding := range policy.EnvironmentBindings {
		if !validID(binding.ID) {
			return invalid(ir.Malformed, "policy.environment_bindings", "invalid environment binding identity")
		}
		if binding.Value == "" || !utf8.ValidString(binding.Value) {
			return invalid(ir.Malformed, "policy.environment_bindings", "invalid environment binding value")
		}
		if _, exists := a.environment[binding.ID]; exists {
			return invalid(ir.Malformed, "policy.environment_bindings", "duplicate environment binding")
		}
		bindingBytes := int64(len(binding.ID) + len(binding.Value))
		if bindingBytes > policy.Limits.MaxRequestBytes-environmentBytes {
			return invalid(ir.LimitExceeded, "policy.environment_bindings", "environment binding byte ceiling exceeded")
		}
		if err := a.charge(1); err != nil {
			return err
		}
		environmentBytes += bindingBytes
		a.environment[binding.ID] = binding.Value
	}
	snapshot := policy
	snapshot.Limits = proto.CloneOf(policy.Limits)
	snapshot.Roles = slices.Clone(policy.Roles)
	snapshot.Opcodes = slices.Clone(policy.Opcodes)
	snapshot.CommandTypes = slices.Clone(policy.CommandTypes)
	snapshot.EnvironmentBindings = slices.Clone(policy.EnvironmentBindings)
	for i, role := range snapshot.Roles {
		bound, err := a.bindRolePolicy(role, policy.Limits)
		if err != nil {
			return err
		}
		snapshot.Roles[i] = bound
	}
	for _, opcode := range snapshot.Opcodes {
		if opcode < contract.InvokeRPC || opcode > contract.MaxOpcode || a.opcodes[opcode] {
			return invalid(ir.Malformed, "policy.opcodes", "invalid or duplicate opcode")
		}
		a.opcodes[opcode] = true
	}
	for _, commandType := range snapshot.CommandTypes {
		if _, declared := enumspb.CommandType_name[int32(commandType)]; !declared || commandType == enumspb.COMMAND_TYPE_UNSPECIFIED || a.commandTypes[commandType] {
			return invalid(ir.Malformed, "policy.command_types", "invalid or duplicate command type")
		}
		a.commandTypes[commandType] = true
	}
	a.prepared.policy = snapshot
	a.prepared.limits = snapshot.Limits
	a.prepared.environmentFingerprint = policy.EnvironmentFingerprint
	return nil
}

// checkInstructionDefaults admits the Profile's instruction defaults: each is absent (zero) or a
// positive value within the ceiling an instruction's own limit must fit.
func checkInstructionDefaults(defaults contract.InstructionDefaults, limits *testpilotspb.ProgramLimits) error {
	if defaults.TimeoutMilliseconds < 0 || defaults.MaxAttempts < 0 {
		return invalid(ir.Malformed, "policy.instruction_defaults", "negative instruction default")
	}
	if defaults.TimeoutMilliseconds > max(limits.MaxTotalDurationMilliseconds, limits.MaxCleanupDurationMilliseconds) || defaults.MaxAttempts > limits.MaxAttempts {
		return invalid(ir.LimitExceeded, "policy.instruction_defaults", "instruction default exceeds the Profile ceiling")
	}
	return nil
}
func (a *admission) bindRolePolicy(role contract.RolePolicy, limits *testpilotspb.ProgramLimits) (contract.RolePolicy, error) {
	if !validID(role.ID) || role.Kind < testpilotspb.ROLE_KIND_ENDPOINT || role.Kind > testpilotspb.ROLE_KIND_PARTICIPANT {
		return contract.RolePolicy{}, invalid(ir.Malformed, "policy.roles", "invalid role")
	}
	if _, exists := a.allowed[role.ID]; exists {
		return contract.RolePolicy{}, invalid(ir.Malformed, "policy.roles", "duplicate role")
	}
	if len(role.Methods) > 10000 || len(role.ReservationCarriers) > 10000 || role.Kind != testpilotspb.ROLE_KIND_ENDPOINT && (len(role.Methods) > 0 || len(role.ReservationCarriers) > 0) {
		return contract.RolePolicy{}, invalid(ir.Malformed, "policy.roles", "invalid endpoint methods")
	}
	methods := make(map[string]bool, len(role.Methods))
	for _, method := range role.Methods {
		if len(method) > 256 {
			return contract.RolePolicy{}, invalid(ir.LimitExceeded, "policy.methods", "method identity ceiling exceeded")
		}
		if err := a.charge(1); err != nil {
			return contract.RolePolicy{}, err
		}
		if methods[method] {
			return contract.RolePolicy{}, invalid(ir.Malformed, "policy.methods", "duplicate method")
		}
		methods[method] = true
		if _, err := a.prepared.catalog.Method(method); err != nil {
			return contract.RolePolicy{}, err
		}
	}
	carriers := make(map[string]contract.ReservationCarrierPolicy, len(role.ReservationCarriers))
	for _, carrier := range role.ReservationCarriers {
		if err := a.bindCarrierPolicy(carrier, methods, limits, carriers); err != nil {
			return contract.RolePolicy{}, err
		}
	}
	bound := role
	bound.Methods = slices.Clone(role.Methods)
	bound.ReservationCarriers = make([]contract.ReservationCarrierPolicy, len(role.ReservationCarriers))
	for i, carrier := range role.ReservationCarriers {
		bound.ReservationCarriers[i] = carrier
		bound.ReservationCarriers[i].Shapes = slices.Clone(carrier.Shapes)
	}
	a.allowed[role.ID] = bound
	a.methods[role.ID] = methods
	a.carriers[role.ID] = carriers
	return bound, nil
}

func (a *admission) bindCarrierPolicy(carrier contract.ReservationCarrierPolicy, methods map[string]bool, limits *testpilotspb.ProgramLimits, carriers map[string]contract.ReservationCarrierPolicy) error {
	if !methods[carrier.Method] {
		return invalid(ir.Unsupported, "policy.reservation_carriers", "carrier method requires ordinary authorization on the same endpoint")
	}
	if _, exists := carriers[carrier.Method]; exists {
		return invalid(ir.Malformed, "policy.reservation_carriers", "duplicate carrier method")
	}
	method, err := a.prepared.catalog.Method(carrier.Method)
	if err != nil {
		return err
	}
	if method.IsStreamingClient() || method.IsStreamingServer() {
		return invalid(ir.Unsupported, "policy.reservation_carriers", "carrier method must be unary")
	}
	if len(carrier.Shapes) == 0 || len(carrier.Shapes) > 2 {
		return invalid(ir.Malformed, "policy.reservation_carriers", "carrier shape is empty or oversized")
	}
	seen := map[contract.EntrypointKind]bool{}
	var total int64
	for _, shape := range carrier.Shapes {
		if shape.Kind != contract.WorkflowEntrypoint && shape.Kind != contract.NexusHandlerEntrypoint {
			return invalid(ir.Unsupported, "policy.reservation_carriers", "carrier shape has an unsupported activation context")
		}
		if seen[shape.Kind] {
			return invalid(ir.Malformed, "policy.reservation_carriers", "duplicate carrier activation context")
		}
		seen[shape.Kind] = true
		if shape.MaximumCount <= 0 || shape.MaximumCount > limits.MaxActivations-total {
			return invalid(ir.LimitExceeded, "policy.reservation_carriers", "carrier cardinality exceeds the activation ceiling")
		}
		total += shape.MaximumCount
		if err := a.charge(1); err != nil {
			return err
		}
	}
	if err := a.charge(1); err != nil {
		return err
	}
	bound := carrier
	bound.Shapes = slices.Clone(carrier.Shapes)
	carriers[carrier.Method] = bound
	return nil
}
func (a *admission) bindSchemas() error {
	p := a.prepared.source
	if !validID(p.ProgramId) {
		return invalid(ir.Malformed, "program", "invalid Program identity")
	}
	if err := a.bindEnvironment(p); err != nil {
		return err
	}
	for _, role := range p.Roles {
		if !validID(role.GetRoleId()) || a.roles[role.GetRoleId()] != 0 || role.GetKind() == 0 || a.allowed[role.GetRoleId()].Kind != role.GetKind() {
			return invalid(ir.Malformed, "roles", "invalid, duplicate or unauthorized role")
		}
		a.roles[role.RoleId] = role.Kind
		resolved, err := a.bindRole(role)
		if err != nil {
			return err
		}
		a.prepared.roles[role.RoleId] = resolved
	}
	for _, slot := range p.Slots {
		if !validID(slot.GetSlotId()) {
			return invalid(ir.Malformed, "slots", "invalid Slot identity")
		}
		if _, exists := a.prepared.slots[slot.SlotId]; exists {
			return invalid(ir.Malformed, "slots", "duplicate Slot")
		}
		var typ ir.Type
		switch slot.Content.(type) {
		case *testpilotspb.Slot_Value:
			bound, err := a.prepared.catalog.BindType(slot.GetValue())
			if err != nil {
				return err
			}
			typ = bound
		case *testpilotspb.Slot_OpaqueHandle:
			typ = a.prepared.catalog.OpaqueHandleType()
		default:
			return invalid(ir.Malformed, "slots", "Slot content is required")
		}
		a.prepared.slots[slot.SlotId] = typ
	}
	a.prepared.view = ProgramView{programID: p.ProgramId, catalogIdentity: a.prepared.catalog.Identity(), limits: a.prepared.limits}
	for _, observation := range p.Observations {
		if !validID(observation.GetObservationId()) {
			return invalid(ir.Malformed, "observations", "invalid Observation identity")
		}
		if _, exists := a.observations[observation.ObservationId]; exists {
			return invalid(ir.Malformed, "observations", "duplicate Observation")
		}
		typ, err := a.prepared.catalog.BindType(observation.Type)
		if err != nil {
			return err
		}
		a.observations[observation.ObservationId] = typ
		a.prepared.view.observations = append(a.prepared.view.observations, Observation{ID: observation.ObservationId, Type: typ})
	}
	return a.bindEvidence(p)
}

// bindEnvironment resolves the Program's derived binding graph against the Profile. Every binding a
// role or expression references must be supplied, and the resolved values together must fit one
// request.
func (a *admission) bindEnvironment(p *testpilotspb.Program) error {
	var environmentBytes int64
	for _, id := range EnvironmentBindingIDs(p) {
		if err := a.charge(1); err != nil {
			return err
		}
		if !validID(id) {
			return invalid(ir.Malformed, "environment", "invalid environment reference")
		}
		value, ok := a.environment[id]
		if !ok {
			return invalid(ir.Unknown, "environment", fmt.Sprintf("environment binding %q is not supplied by the Profile", id))
		}
		bytes := int64(len(id) + len(value))
		if bytes > a.prepared.limits.MaxRequestBytes-environmentBytes {
			return invalid(ir.LimitExceeded, "environment", "resolved environment byte ceiling exceeded")
		}
		environmentBytes += bytes
		a.bindings[id] = value
	}
	return nil
}

func (a *admission) bindRole(role *testpilotspb.Role) (resolvedRole, error) {
	result := resolvedRole{ID: role.RoleId, Kind: role.Kind, NamespaceBindingID: role.NamespaceBindingId, ResourceBindingID: role.ResourceBindingId}
	switch role.Kind {
	case testpilotspb.ROLE_KIND_ENDPOINT:
		if role.NamespaceBindingId != "" {
			return resolvedRole{}, invalid(ir.Unsupported, "roles", "endpoint role cannot bind a namespace")
		}
	case testpilotspb.ROLE_KIND_WORKER:
		if role.ResourceBindingId != "" {
			return resolvedRole{}, invalid(ir.Unsupported, "roles", "worker role cannot bind a resource")
		}
		if a.bindingsRequired && role.NamespaceBindingId == "" {
			return resolvedRole{}, invalid(ir.Unsupported, "roles", "worker role requires a namespace binding")
		}
	case testpilotspb.ROLE_KIND_TASK_QUEUE:
		if a.bindingsRequired && (role.NamespaceBindingId == "" || role.ResourceBindingId == "") {
			return resolvedRole{}, invalid(ir.Unsupported, "roles", "task-queue role requires namespace and resource bindings")
		}
	case testpilotspb.ROLE_KIND_PARTICIPANT:
		if role.NamespaceBindingId != "" || role.ResourceBindingId != "" {
			return resolvedRole{}, invalid(ir.Unsupported, "roles", "participant role cannot bind resources")
		}
	default:
		return resolvedRole{}, invalid(ir.Malformed, "roles", "unknown role kind")
	}
	var err error
	if role.NamespaceBindingId != "" {
		result.Namespace, err = a.resolveEnvironment(role.NamespaceBindingId)
		if err != nil {
			return resolvedRole{}, err
		}
	}
	if role.ResourceBindingId != "" {
		result.Resource, err = a.resolveEnvironment(role.ResourceBindingId)
		if err != nil {
			return resolvedRole{}, err
		}
	}
	return result, nil
}

// EnvironmentBindingIDs is the Program's symbolic binding graph: each binding its roles reference, a
// role's namespace before its resource in role order, then each binding an expression of its
// instructions references, in declaration order, every binding once. A Program declares no bindings;
// preparation resolves exactly this set against the Profile, and Profile derivation reads it.
func EnvironmentBindingIDs(program *testpilotspb.Program) []string {
	var ids []string
	seen := map[string]bool{}
	add := func(id string) {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, role := range program.GetRoles() {
		// An empty role binding binds nothing; an empty reference is kept, so preparation rejects it.
		for _, id := range []string{role.GetNamespaceBindingId(), role.GetResourceBindingId()} {
			if id != "" {
				add(id)
			}
		}
	}
	var visit func(protoreflect.Message)
	visit = func(message protoreflect.Message) {
		if reference, ok := message.Interface().(*testpilotspb.Reference); ok {
			if id, isEnvironment := reference.GetReference().(*testpilotspb.Reference_EnvironmentBindingId); isEnvironment {
				add(id.EnvironmentBindingId)
			}
		}
		fields := message.Descriptor().Fields()
		for i := range fields.Len() {
			field := fields.Get(i)
			if field.Message() == nil || field.IsMap() || !message.Has(field) {
				continue
			}
			if field.IsList() {
				list := message.Get(field).List()
				for j := range list.Len() {
					visit(list.Get(j).Message())
				}
				continue
			}
			visit(message.Get(field).Message())
		}
	}
	for _, entrypoint := range program.GetEntrypoints() {
		for _, instruction := range entrypoint.GetInstructions() {
			visit(instruction.ProtoReflect())
		}
	}
	for _, instruction := range program.GetCleanup().GetInstructions() {
		visit(instruction.ProtoReflect())
	}
	return ids
}

// resolveEnvironment reads one reference's value from the derived binding graph, which holds every
// binding the Program references.
func (a *admission) resolveEnvironment(id string) (string, error) {
	value, ok := a.bindings[id]
	if !ok {
		return "", invalid(ir.Unknown, "environment", "environment reference is outside the derived binding graph")
	}
	return value, nil
}
func (a *admission) role(id string, kind testpilotspb.RoleKind) error {
	if a.roles[id] != kind {
		return invalid(ir.Unknown, "role", "role is missing or has the wrong kind")
	}
	return nil
}
func (a *admission) bindActivation(g *graph) error {
	b := g.activation
	if b == nil || isNil(b.Activation) {
		return invalid(ir.Malformed, g.id, "activation binding is required")
	}
	var worker, queue, name, operation string
	expected := contract.EntrypointKindOf(b)
	switch binding := b.Activation.(type) {
	case *testpilotspb.Entrypoint_Controller:
		if binding.Controller == nil {
			return invalid(ir.Malformed, g.id, "nil activation")
		}
	case *testpilotspb.Entrypoint_Workflow:
		worker = binding.Workflow.GetWorkerRoleId()
		queue = binding.Workflow.GetTaskQueueRoleId()
		name = binding.Workflow.GetWorkflowType()
	case *testpilotspb.Entrypoint_Activity:
		worker = binding.Activity.GetWorkerRoleId()
		queue = binding.Activity.GetTaskQueueRoleId()
		name = binding.Activity.GetActivityType()
	case *testpilotspb.Entrypoint_NexusHandler:
		worker = binding.NexusHandler.GetWorkerRoleId()
		queue = binding.NexusHandler.GetTaskQueueRoleId()
		name = binding.NexusHandler.GetService()
		operation = binding.NexusHandler.GetOperation()
		if !validID(operation) {
			return invalid(ir.Malformed, g.id, "invalid Nexus operation")
		}
	default:
		return invalid(ir.Unsupported, g.id, "unknown activation")
	}
	g.context = expected
	if expected != contract.ControllerEntrypoint {
		if !validID(name) {
			return invalid(ir.Malformed, g.id, "invalid activation name")
		}
		if err := a.role(worker, testpilotspb.ROLE_KIND_WORKER); err != nil {
			return err
		}
		return a.role(queue, testpilotspb.ROLE_KIND_TASK_QUEUE)
	}
	return nil
}
func (a *admission) bindGraphs() error {
	p := a.prepared.source
	if len(p.Entrypoints) == 0 || int64(len(p.Entrypoints)) > a.prepared.limits.MaxEntrypoints || p.Cleanup == nil {
		return invalid(ir.LimitExceeded, "entrypoints", "entrypoints and cleanup must fit the declared bound")
	}
	for _, entry := range p.Entrypoints {
		if entry == nil {
			return invalid(ir.Malformed, "entrypoints", "nil entrypoint")
		}
		g := &graph{id: entry.EntrypointId, activation: entry}
		if err := a.bindActivation(g); err != nil {
			return err
		}
		if err := a.addGraph(g, entry.Instructions); err != nil {
			return err
		}
	}
	cleanup := p.Cleanup
	if err := a.addGraph(&graph{id: cleanup.EntrypointId, context: contract.ControllerEntrypoint, cleanup: true}, cleanup.Instructions); err != nil {
		return err
	}
	var nodes, edges int64
	for _, g := range a.prepared.graphs {
		nodes += int64(len(g.nodes))
		for _, n := range g.nodes {
			edges += int64(len(n.dependencies))
		}
	}
	if nodes > a.prepared.limits.MaxNodes || edges > a.prepared.limits.MaxEdges {
		return invalid(ir.LimitExceeded, "program", "node or edge ceiling exceeded")
	}
	return nil
}
func (a *admission) addGraph(g *graph, sources []*testpilotspb.InstructionNode) error {
	if !validID(g.id) || a.graphIndex[g.id] != nil {
		return invalid(ir.Malformed, "entrypoints", "invalid or duplicate entrypoint identity")
	}
	a.graphIndex[g.id] = g
	g.index = map[string]int{}
	a.prepared.graphs = append(a.prepared.graphs, g)
	for i, source := range sources {
		if !validID(source.GetInstructionId()) {
			return invalid(ir.Malformed, g.id, "invalid instruction identity")
		}
		if _, exists := g.index[source.InstructionId]; exists {
			return invalid(ir.Malformed, g.id, "duplicate instruction identity")
		}
		g.index[source.InstructionId] = i
		g.nodes = append(g.nodes, &node{source: source, outcomes: map[testpilotspb.InstructionOutcomeField]ir.Type{}, ancestors: map[int]bool{}})
	}
	return a.orderGraph(g)
}
func (a *admission) orderGraph(g *graph) error {
	indegree := make([]int, len(g.nodes))
	for i, n := range g.nodes {
		dependencies, err := resolveAfter(g, i)
		if err != nil {
			return err
		}
		for _, j := range dependencies {
			n.dependencies = append(n.dependencies, j)
			g.nodes[j].successors = append(g.nodes[j].successors, i)
			indegree[i]++
		}
		n.guardSource = effectiveGuard(g, n)
	}
	var ready []int
	for i, degree := range indegree {
		if degree == 0 {
			ready = append(ready, i)
		}
	}
	for len(ready) > 0 {
		i := ready[0]
		ready = ready[1:]
		g.order = append(g.order, i)
		n := g.nodes[i]
		for _, dependency := range n.dependencies {
			if err := a.charge(int64(len(g.nodes[dependency].ancestors)) + 1); err != nil {
				return err
			}
			maps.Copy(n.ancestors, g.nodes[dependency].ancestors)
			n.ancestors[dependency] = true
		}
		for _, successor := range n.successors {
			indegree[successor]--
			if indegree[successor] == 0 {
				ready = append(ready, successor)
			}
		}
	}
	// A node left unordered lies on or behind a cycle. The first one always declares after: a
	// defaulted node waits only on its predecessor, which would be unordered and earlier.
	for i, degree := range indegree {
		if degree > 0 {
			return invalid(ir.Malformed, expressionPath(g, g.nodes[i], "after"), "dependency cycle")
		}
	}
	return nil
}

// resolveAfter returns the nodes node i runs after: its declared after set, or its entrypoint
// predecessor when it declares none. An after set names instructions of its own entrypoint only;
// entrypoints coordinate through the target, not through the scheduler.
func resolveAfter(g *graph, i int) ([]int, error) {
	n := g.nodes[i]
	after := n.source.GetAfter()
	if after == nil {
		if i == 0 {
			return nil, nil
		}
		return []int{i - 1}, nil
	}
	seen := map[int]bool{}
	dependencies := make([]int, 0, len(after.GetInstructions()))
	for k, reference := range after.GetInstructions() {
		path := expressionPath(g, n, fmt.Sprintf("after.instructions[%d]", k))
		if !validID(reference.GetEntrypointId()) || !validID(reference.GetInstructionId()) {
			return nil, invalid(ir.Malformed, path, "invalid instruction reference")
		}
		if reference.GetEntrypointId() != g.id {
			return nil, invalid(ir.Unsupported, path, "after names an instruction of another entrypoint")
		}
		j, exists := g.index[reference.GetInstructionId()]
		switch {
		case !exists:
			return nil, invalid(ir.Unknown, path, "after names an unknown instruction")
		case j == i:
			return nil, invalid(ir.Malformed, path, "after names the instruction itself")
		case seen[j]:
			return nil, invalid(ir.Malformed, path, "after names an instruction twice")
		}
		seen[j] = true
		dependencies = append(dependencies, j)
	}
	return dependencies, nil
}

// effectiveGuard is the guard a node runs under. Without an explicit guard it runs only when every
// dependency succeeded; an explicit guard replaces that condition, and a literal true one runs
// regardless, exactly like a node with no dependencies and no guard.
func effectiveGuard(g *graph, n *node) *testpilotspb.Expression {
	if guard := n.source.GetGuard(); guard != nil {
		if literal, ok := guard.GetLiteral().GetValue().(*testpilotspb.Value_BoolValue); ok && literal.BoolValue {
			return nil
		}
		return guard
	}
	switch len(n.dependencies) {
	case 0:
		return nil
	case 1:
		return dependencySucceeded(g.id, g.nodes[n.dependencies[0]].source.GetInstructionId())
	}
	operands := make([]*testpilotspb.Expression, len(n.dependencies))
	for k, j := range n.dependencies {
		operands[k] = dependencySucceeded(g.id, g.nodes[j].source.GetInstructionId())
	}
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_All{All: &testpilotspb.AllExpression{Operands: operands}}}
}

// dependencySucceeded holds when the instruction recorded a SUCCEEDED outcome. A skipped dependency
// records no outcome, which the comparison alone already makes false; the presence conjunct gives the
// dependent the status's presence fact, which a comparison does not.
func dependencySucceeded(entrypointID, instructionID string) *testpilotspb.Expression {
	status := func() *testpilotspb.Expression {
		return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{
			Instruction: &testpilotspb.InstructionReference{EntrypointId: entrypointID, InstructionId: instructionID},
			Field:       testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS,
		}}}}}
	}
	succeeded := &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Name: ir.EnumName(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED)}}}}}
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_All{All: &testpilotspb.AllExpression{Operands: []*testpilotspb.Expression{
		{Expression: &testpilotspb.Expression_Present{Present: &testpilotspb.PresentExpression{Operand: status()}}},
		{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{Operator: testpilotspb.COMPARISON_OPERATOR_EQUAL, Left: status(), Right: succeeded}}},
	}}}}
}
func (a *admission) expressionLimits() ir.Limits {
	limits := ir.DefaultLimits()
	limits.Depth = a.prepared.limits.MaxExpressionDepth
	limits.Bytes = a.prepared.limits.MaxRequestBytes
	limits.Fanout = a.prepared.limits.MaxPathFanout
	return limits
}
func (a *admission) bindReservations() error {
	type weighted struct{ count, attempts int64 }
	var weights []weighted
	var controllers int64
	limit := a.prepared.limits.MaxActivations
	for _, g := range a.prepared.graphs {
		if !g.cleanup && g.context == contract.ControllerEntrypoint {
			controllers++
		}
		for _, n := range g.nodes {
			count, err := a.reservationCount(g, n)
			if err != nil {
				return err
			}
			if count > 0 {
				weights = append(weights, weighted{count: count, attempts: n.maxAttempts})
			}
		}
	}
	if controllers > limit {
		return invalid(ir.LimitExceeded, "activations", "controller activations exceed ceiling")
	}
	// Taking the largest reservation weight first maximizes the sum under both attempt caps.
	slices.SortFunc(weights, func(a, b weighted) int { return cmp.Compare(b.count, a.count) })
	remaining := a.prepared.limits.MaxAttempts
	total := controllers
	for _, weight := range weights {
		attempts := min(remaining, weight.attempts)
		if attempts > 0 && weight.count > (limit-total)/attempts {
			return invalid(ir.LimitExceeded, "activations", "attempt-scaled reservations exceed ceiling")
		}
		total += weight.count * attempts
		remaining -= attempts
	}
	a.prepared.view.maximumActivations = total
	return nil
}
func (a *admission) reservationCount(g *graph, n *node) (int64, error) {
	limit := a.prepared.limits.MaxActivations
	var count int64
	for _, reservation := range n.reservations {
		if reservation.Count > limit-count {
			return 0, invalid(ir.LimitExceeded, g.id, "reservation sum exceeds activation ceiling")
		}
		count += reservation.Count
	}
	return count, nil
}
func messageType(catalog *ir.Catalog, descriptor protoreflect.MessageDescriptor) (ir.Type, error) {
	if descriptor.FullName() == "google.protobuf.Any" {
		return catalog.BindType(&testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Any{Any: &testpilotspb.AnyType{}}}}})
	}
	return catalog.BindType(&testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: string(descriptor.FullName())}}}}})
}
func nodePath(g *graph, n *node) string { return fmt.Sprintf("%s.%s", g.id, n.source.InstructionId) }
