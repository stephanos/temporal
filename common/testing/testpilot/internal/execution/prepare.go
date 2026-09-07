package execution

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"unicode/utf8"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type admission struct {
	prepared               *PreparedProgram
	roles                  map[string]testpilotspb.RoleKind
	allowed                map[string]RolePolicy
	methods                map[string]map[string]bool
	carriers               map[string]map[string]ReservationCarrierPolicy
	capabilities           map[Opcode]bool
	bindingsRequired       bool
	environment            map[string]string
	environmentDefinitions map[string]bool
	environmentUsed        map[string]bool
	observations           map[string]ir.Type
	runID                  ir.Type
	writers                map[string]slotWriter
	graphIndex             map[string]*graph
	work                   int64
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
func Prepare(source *testpilotspb.Case, catalog *ir.Catalog, policy Policy) (*PreparedProgram, error) {
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
	prepared := &PreparedProgram{source: proto.CloneOf(source.Program), catalog: catalog, slots: map[string]ir.Type{}, carriers: map[carrierCoordinate]ReservationCarrierPlan{}, roles: map[string]resolvedRole{}}
	a := &admission{prepared: prepared, roles: map[string]testpilotspb.RoleKind{}, allowed: map[string]RolePolicy{}, methods: map[string]map[string]bool{}, carriers: map[string]map[string]ReservationCarrierPolicy{}, capabilities: map[Opcode]bool{}, bindingsRequired: true, environment: map[string]string{}, environmentDefinitions: map[string]bool{}, environmentUsed: map[string]bool{}, observations: map[string]ir.Type{}, writers: map[string]slotWriter{}, graphIndex: map[string]*graph{}}
	for _, check := range []func() error{func() error { return a.bindPolicy(policy) }, a.bindSchemas, a.bindGraphs, a.bindInstructions, a.bindDataflow, a.bindReservations, a.bindReservationCarriers} {
		if err := check(); err != nil {
			return nil, err
		}
	}
	for id := range a.environmentDefinitions {
		if !a.environmentUsed[id] {
			return nil, invalid(ir.Malformed, "environment", "environment definition is unused")
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
func (a *admission) bindPolicy(policy Policy) error {
	if !validID(policy.Identity) || policy.CatalogIdentity != a.prepared.catalog.Identity() {
		return invalid(ir.Malformed, "policy", "Driver or catalog identity mismatch")
	}
	if err := checkLimits(policy.Limits, hardLimits()); err != nil {
		return err
	}
	if err := checkLimits(a.prepared.source.Limits, policy.Limits); err != nil {
		return err
	}
	if len(policy.Roles) > 10000 || len(policy.Capabilities) > 7 || len(policy.EnvironmentBindings) > 10000 {
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
	snapshot.Capabilities = slices.Clone(policy.Capabilities)
	snapshot.EnvironmentBindings = slices.Clone(policy.EnvironmentBindings)
	for i, role := range snapshot.Roles {
		bound, err := a.bindRolePolicy(role, policy.Limits)
		if err != nil {
			return err
		}
		snapshot.Roles[i] = bound
	}
	for _, capability := range snapshot.Capabilities {
		if capability < InvokeRPC || capability > RespondNexus || a.capabilities[capability] {
			return invalid(ir.Malformed, "policy.capabilities", "invalid or duplicate capability")
		}
		a.capabilities[capability] = true
	}
	a.prepared.policy = snapshot
	a.prepared.environmentFingerprint = policy.EnvironmentFingerprint
	return nil
}
func (a *admission) bindRolePolicy(role RolePolicy, limits *testpilotspb.ProgramLimits) (RolePolicy, error) {
	if !validID(role.ID) || role.Kind < testpilotspb.ROLE_KIND_ENDPOINT || role.Kind > testpilotspb.ROLE_KIND_PARTICIPANT {
		return RolePolicy{}, invalid(ir.Malformed, "policy.roles", "invalid role")
	}
	if _, exists := a.allowed[role.ID]; exists {
		return RolePolicy{}, invalid(ir.Malformed, "policy.roles", "duplicate role")
	}
	if len(role.Methods) > 10000 || len(role.ReservationCarriers) > 10000 || role.Kind != testpilotspb.ROLE_KIND_ENDPOINT && (len(role.Methods) > 0 || len(role.ReservationCarriers) > 0) {
		return RolePolicy{}, invalid(ir.Malformed, "policy.roles", "invalid endpoint methods")
	}
	methods := make(map[string]bool, len(role.Methods))
	for _, method := range role.Methods {
		if len(method) > 256 {
			return RolePolicy{}, invalid(ir.LimitExceeded, "policy.methods", "method identity ceiling exceeded")
		}
		if err := a.charge(1); err != nil {
			return RolePolicy{}, err
		}
		if methods[method] {
			return RolePolicy{}, invalid(ir.Malformed, "policy.methods", "duplicate method")
		}
		methods[method] = true
		if _, err := a.prepared.catalog.Method(method); err != nil {
			return RolePolicy{}, err
		}
	}
	carriers := make(map[string]ReservationCarrierPolicy, len(role.ReservationCarriers))
	for _, carrier := range role.ReservationCarriers {
		if err := a.bindCarrierPolicy(carrier, methods, limits, carriers); err != nil {
			return RolePolicy{}, err
		}
	}
	bound := role
	bound.Methods = slices.Clone(role.Methods)
	bound.ReservationCarriers = make([]ReservationCarrierPolicy, len(role.ReservationCarriers))
	for i, carrier := range role.ReservationCarriers {
		bound.ReservationCarriers[i] = carrier
		bound.ReservationCarriers[i].Shapes = slices.Clone(carrier.Shapes)
	}
	a.allowed[role.ID] = bound
	a.methods[role.ID] = methods
	a.carriers[role.ID] = carriers
	return bound, nil
}

func (a *admission) bindCarrierPolicy(carrier ReservationCarrierPolicy, methods map[string]bool, limits *testpilotspb.ProgramLimits, carriers map[string]ReservationCarrierPolicy) error {
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
	seen := map[testpilotspb.EntrypointKind]bool{}
	var total int64
	for _, shape := range carrier.Shapes {
		if shape.Context != testpilotspb.ENTRYPOINT_KIND_WORKFLOW && shape.Context != testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER {
			return invalid(ir.Unsupported, "policy.reservation_carriers", "carrier shape has an unsupported activation context")
		}
		if seen[shape.Context] {
			return invalid(ir.Malformed, "policy.reservation_carriers", "duplicate carrier activation context")
		}
		seen[shape.Context] = true
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
	if len(p.Environment) > 10000 {
		return invalid(ir.LimitExceeded, "environment", "environment definition collection ceiling exceeded")
	}
	var environmentBytes int64
	for _, definition := range p.Environment {
		if definition == nil || !validID(definition.BindingId) || a.environmentDefinitions[definition.BindingId] {
			return invalid(ir.Malformed, "environment", "invalid or duplicate environment definition")
		}
		value, ok := a.environment[definition.BindingId]
		if !ok {
			return invalid(ir.Unknown, "environment", "environment binding is not supplied by the Profile")
		}
		bytes := int64(len(definition.BindingId) + len(value))
		if bytes > p.Limits.MaxRequestBytes-environmentBytes {
			return invalid(ir.LimitExceeded, "environment", "resolved environment byte ceiling exceeded")
		}
		environmentBytes += bytes
		a.environmentDefinitions[definition.BindingId] = true
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
		var schema *testpilotspb.ValueType
		switch slot.Content.(type) {
		case *testpilotspb.SlotDefinition_Value:
			schema = slot.GetValue()
		case *testpilotspb.SlotDefinition_OpaqueCapability:
			schema = &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_OpaqueCapability{OpaqueCapability: &testpilotspb.OpaqueCapabilityType{}}}}}
		default:
			return invalid(ir.Malformed, "slots", "Slot content is required")
		}
		typ, err := a.prepared.catalog.BindType(schema)
		if err != nil {
			return err
		}
		a.prepared.slots[slot.SlotId] = typ
	}
	a.prepared.view = ProgramView{programID: p.ProgramId, catalogIdentity: a.prepared.catalog.Identity(), limits: p.Limits}
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
		if typ.Opaque() {
			return invalid(ir.Unsupported, "observations", "capability Observation is forbidden")
		}
		a.observations[observation.ObservationId] = typ
		a.prepared.view.observations = append(a.prepared.view.observations, Observation{ID: observation.ObservationId, Type: typ})
	}
	return nil
}

func (a *admission) bindRole(role *testpilotspb.RoleDefinition) (resolvedRole, error) {
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

func (a *admission) resolveEnvironment(id string) (string, error) {
	if !validID(id) {
		return "", invalid(ir.Malformed, "environment", "invalid environment reference")
	}
	if !a.environmentDefinitions[id] {
		return "", invalid(ir.Unknown, "environment", "environment reference is not declared")
	}
	value, ok := a.environment[id]
	if !ok {
		return "", invalid(ir.Unknown, "environment", "environment binding is not supplied by the Profile")
	}
	a.environmentUsed[id] = true
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
	var expected testpilotspb.EntrypointKind
	switch binding := b.Activation.(type) {
	case *testpilotspb.EntrypointDefinition_Controller:
		expected = testpilotspb.ENTRYPOINT_KIND_CONTROLLER
		if binding.Controller == nil {
			return invalid(ir.Malformed, g.id, "nil activation")
		}
	case *testpilotspb.EntrypointDefinition_Workflow:
		expected = testpilotspb.ENTRYPOINT_KIND_WORKFLOW
		worker = binding.Workflow.GetWorkerRoleId()
		queue = binding.Workflow.GetTaskQueueRoleId()
		name = binding.Workflow.GetWorkflowType()
	case *testpilotspb.EntrypointDefinition_Activity:
		expected = testpilotspb.ENTRYPOINT_KIND_ACTIVITY
		worker = binding.Activity.GetWorkerRoleId()
		queue = binding.Activity.GetTaskQueueRoleId()
		name = binding.Activity.GetActivityType()
	case *testpilotspb.EntrypointDefinition_NexusHandler:
		expected = testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER
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
	if expected != testpilotspb.ENTRYPOINT_KIND_CONTROLLER {
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
	if len(p.Entrypoints) == 0 || int64(len(p.Entrypoints)) > p.Limits.MaxEntrypoints || p.Cleanup == nil {
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
	if err := a.addGraph(&graph{id: cleanup.EntrypointId, context: testpilotspb.ENTRYPOINT_KIND_CONTROLLER, cleanup: true}, cleanup.Instructions); err != nil {
		return err
	}
	var nodes, edges int64
	for _, g := range a.prepared.graphs {
		nodes += int64(len(g.nodes))
		for _, n := range g.nodes {
			edges += int64(len(n.dependencies))
		}
	}
	if nodes > p.Limits.MaxNodes || edges > p.Limits.MaxEdges {
		return invalid(ir.LimitExceeded, "program", "node or edge ceiling exceeded")
	}
	return nil
}
func (a *admission) addGraph(g *graph, sources []*testpilotspb.InstructionDefinition) error {
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
		seen := map[int]bool{}
		for _, dependency := range n.source.Dependencies {
			j, exists := g.index[dependency.GetInstructionId()]
			if !exists || dependency.GetEntrypointId() != g.id || seen[j] {
				return invalid(ir.Malformed, g.id, "missing, duplicate or cross-entrypoint dependency")
			}
			seen[j] = true
			n.dependencies = append(n.dependencies, j)
			g.nodes[j].successors = append(g.nodes[j].successors, i)
			indegree[i]++
		}
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
	if len(g.order) != len(g.nodes) {
		return invalid(ir.Malformed, g.id, "dependency cycle")
	}
	return nil
}
func (a *admission) expressionLimits() ir.Limits {
	limits := ir.DefaultLimits()
	limits.Depth = a.prepared.source.Limits.MaxExpressionDepth
	limits.Bytes = a.prepared.source.Limits.MaxRequestBytes
	limits.Fanout = a.prepared.source.Limits.MaxPathFanout
	return limits
}
func (a *admission) bindReservations() error {
	type weighted struct{ count, attempts int64 }
	var weights []weighted
	var controllers int64
	limit := a.prepared.source.Limits.MaxActivations
	for _, g := range a.prepared.graphs {
		if !g.cleanup && g.context == testpilotspb.ENTRYPOINT_KIND_CONTROLLER {
			controllers++
		}
		for _, n := range g.nodes {
			count, err := a.reservationCount(g, n)
			if err != nil {
				return err
			}
			if count > 0 {
				weights = append(weights, weighted{count: count, attempts: n.source.Limits.MaxAttempts})
			}
		}
	}
	if controllers > limit {
		return invalid(ir.LimitExceeded, "activations", "controller activations exceed ceiling")
	}
	// Taking the largest reservation weight first maximizes the sum under both attempt caps.
	slices.SortFunc(weights, func(a, b weighted) int { return cmp.Compare(b.count, a.count) })
	remaining := a.prepared.source.Limits.MaxAttempts
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
	limit := a.prepared.source.Limits.MaxActivations
	var count int64
	seen := map[string]bool{}
	for _, reservation := range n.source.ActivationReservations {
		if g.cleanup || g.context != testpilotspb.ENTRYPOINT_KIND_CONTROLLER {
			return 0, invalid(ir.Unsupported, g.id, "only ordinary controller nodes may reserve activations")
		}
		target := a.graphIndex[reservation.GetEntrypointId()]
		if target == nil || target.cleanup || target.context != testpilotspb.ENTRYPOINT_KIND_WORKFLOW && target.context != testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER {
			return 0, invalid(ir.TypeMismatch, g.id, "reservation requires a bound workflow or Nexus-handler entrypoint")
		}
		if seen[target.id] || reservation.GetCount() <= 0 {
			return 0, invalid(ir.Malformed, g.id, "reservation targets must be unique with positive counts")
		}
		seen[target.id] = true
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
