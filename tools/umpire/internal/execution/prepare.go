package execution

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type admission struct {
	prepared     *PreparedProgram
	roles        map[string]umpirespb.SymbolicRoleKind
	allowed      map[string]RolePolicy
	capabilities map[Opcode]bool
	observations map[string]ir.Type
	writers      map[string]slotWriter
	graphIndex   map[string]*graph
	work         int64
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
func Prepare(source *umpirespb.Case, catalog *ir.Catalog, policy Policy) (*PreparedProgram, error) {
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
	if err := validateMetadata(source.Metadata); err != nil {
		return nil, err
	}
	prepared := &PreparedProgram{source: proto.CloneOf(source.Program), catalog: catalog, slots: map[string]ir.Type{}}
	a := &admission{prepared: prepared, roles: map[string]umpirespb.SymbolicRoleKind{}, allowed: map[string]RolePolicy{}, capabilities: map[Opcode]bool{}, observations: map[string]ir.Type{}, writers: map[string]slotWriter{}, graphIndex: map[string]*graph{}}
	for _, check := range []func() error{func() error { return a.bindPolicy(policy) }, a.bindSchemas, a.bindGraphs, a.bindInstructions, a.bindDataflow, a.bindReservations} {
		if err := check(); err != nil {
			return nil, err
		}
	}
	return prepared, nil
}
func validateMetadata(metadata *umpirespb.CaseMetadata) error {
	if metadata == nil {
		return nil
	}
	if !validID(metadata.ProducerId) {
		return invalid(ir.Malformed, "metadata", "invalid Producer identity")
	}
	seen := map[string]bool{}
	for _, definition := range metadata.Definitions {
		if !validID(definition.GetDefinitionId()) || seen[definition.GetDefinitionId()] || definition.GetBehaviorFingerprint() == "" || definition.GetKind() == umpirespb.CASE_DEFINITION_KIND_UNSPECIFIED {
			return invalid(ir.Malformed, "metadata", "invalid or duplicate definition")
		}
		seen[definition.DefinitionId] = true
	}
	for _, source := range metadata.Sources {
		if strings.TrimSpace(source.GetPath()) == "" || source.GetLine() <= 0 || source.GetColumn() <= 0 {
			return invalid(ir.Malformed, "metadata", "invalid source location")
		}
	}
	for _, gap := range metadata.KnownGaps {
		if gap.GetKind() == umpirespb.CASE_KNOWN_GAP_KIND_UNSPECIFIED || !validID(gap.GetCode()) {
			return invalid(ir.Malformed, "metadata", "invalid known gap")
		}
	}
	return nil
}
func checkLimits(limits, ceiling *umpirespb.ProgramLimits) error {
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
			return invalid(ir.LimitExceeded, string(field.Name()), "limit is outside the positive Host ceiling")
		}
	}
	return nil
}
func (a *admission) bindPolicy(policy Policy) error {
	if !validID(policy.Identity) || policy.CatalogIdentity != a.prepared.catalog.Identity() {
		return invalid(ir.Malformed, "policy", "Host or catalog identity mismatch")
	}
	if err := checkLimits(policy.Limits, hardLimits()); err != nil {
		return err
	}
	if err := checkLimits(a.prepared.source.Limits, policy.Limits); err != nil {
		return err
	}
	if len(policy.Roles) > 10000 || len(policy.Capabilities) > 7 {
		return invalid(ir.LimitExceeded, "policy", "policy collection ceiling exceeded")
	}
	snapshot := policy
	snapshot.Limits = proto.CloneOf(policy.Limits)
	snapshot.Roles = slices.Clone(policy.Roles)
	snapshot.Capabilities = slices.Clone(policy.Capabilities)
	for i, role := range snapshot.Roles {
		bound, err := a.bindRolePolicy(role)
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
	return nil
}
func (a *admission) bindRolePolicy(role RolePolicy) (RolePolicy, error) {
	if !validID(role.ID) || role.Kind < umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT || role.Kind > umpirespb.SYMBOLIC_ROLE_KIND_PARTICIPANT {
		return RolePolicy{}, invalid(ir.Malformed, "policy.roles", "invalid role")
	}
	if _, exists := a.allowed[role.ID]; exists {
		return RolePolicy{}, invalid(ir.Malformed, "policy.roles", "duplicate role")
	}
	if len(role.Methods) > 10000 || role.Kind != umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT && len(role.Methods) > 0 {
		return RolePolicy{}, invalid(ir.Malformed, "policy.roles", "invalid endpoint methods")
	}
	role.Methods = slices.Clone(role.Methods)
	for _, method := range role.Methods {
		if len(method) > 256 {
			return RolePolicy{}, invalid(ir.LimitExceeded, "policy.methods", "method identity ceiling exceeded")
		}
		if err := a.charge(1); err != nil {
			return RolePolicy{}, err
		}
		if _, err := a.prepared.catalog.Method(method); err != nil {
			return RolePolicy{}, err
		}
	}
	a.allowed[role.ID] = role
	return role, nil
}
func (a *admission) bindSchemas() error {
	p := a.prepared.source
	if !validID(p.ProgramId) {
		return invalid(ir.Malformed, "program", "invalid Program identity")
	}
	for _, role := range p.Roles {
		if !validID(role.GetRoleId()) || a.roles[role.GetRoleId()] != 0 || role.GetKind() == 0 || a.allowed[role.GetRoleId()].Kind != role.GetKind() {
			return invalid(ir.Malformed, "roles", "invalid, duplicate or unauthorized role")
		}
		a.roles[role.RoleId] = role.Kind
	}
	for _, slot := range p.Slots {
		if !validID(slot.GetSlotId()) {
			return invalid(ir.Malformed, "slots", "invalid Slot identity")
		}
		if _, exists := a.prepared.slots[slot.SlotId]; exists {
			return invalid(ir.Malformed, "slots", "duplicate Slot")
		}
		typ, err := a.prepared.catalog.BindType(slot.Type)
		if err != nil {
			return err
		}
		if slot.Kind != umpirespb.SLOT_KIND_VALUE && slot.Kind != umpirespb.SLOT_KIND_OPAQUE_CAPABILITY || typ.Opaque() != (slot.Kind == umpirespb.SLOT_KIND_OPAQUE_CAPABILITY) {
			return invalid(ir.TypeMismatch, "slots", "Slot kind and type disagree")
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
func (a *admission) role(id string, kind umpirespb.SymbolicRoleKind) error {
	if a.roles[id] != kind {
		return invalid(ir.Unknown, "role", "role is missing or has the wrong kind")
	}
	return nil
}
func (a *admission) bindActivation(g *graph) error {
	b := g.activation
	if b == nil || isNil(b.Binding) {
		return invalid(ir.Malformed, g.id, "activation binding is required")
	}
	var worker, queue, name, operation string
	var expected umpirespb.EntrypointContext
	switch binding := b.Binding.(type) {
	case *umpirespb.ActivationBinding_Controller:
		expected = umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER
		if binding.Controller == nil {
			return invalid(ir.Malformed, g.id, "nil activation")
		}
	case *umpirespb.ActivationBinding_Workflow:
		expected = umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW
		worker = binding.Workflow.GetWorkerRoleId()
		queue = binding.Workflow.GetTaskQueueRoleId()
		name = binding.Workflow.GetWorkflowType()
	case *umpirespb.ActivationBinding_Activity:
		expected = umpirespb.ENTRYPOINT_CONTEXT_ACTIVITY
		worker = binding.Activity.GetWorkerRoleId()
		queue = binding.Activity.GetTaskQueueRoleId()
		name = binding.Activity.GetActivityType()
	case *umpirespb.ActivationBinding_NexusHandler:
		expected = umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER
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
	if g.context != expected {
		return invalid(ir.TypeMismatch, g.id, "activation context mismatch")
	}
	if expected != umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER {
		if !validID(name) {
			return invalid(ir.Malformed, g.id, "invalid activation name")
		}
		if err := a.role(worker, umpirespb.SYMBOLIC_ROLE_KIND_WORKER); err != nil {
			return err
		}
		return a.role(queue, umpirespb.SYMBOLIC_ROLE_KIND_TASK_QUEUE)
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
		g := &graph{id: entry.EntrypointId, context: entry.Context, activation: entry.Activation}
		if err := a.bindActivation(g); err != nil {
			return err
		}
		if err := a.addGraph(g, entry.Nodes); err != nil {
			return err
		}
	}
	cleanup := p.Cleanup
	if cleanup.Context != umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER {
		return invalid(ir.Unsupported, "cleanup", "cleanup requires controller context")
	}
	if err := a.addGraph(&graph{id: cleanup.EntrypointId, context: cleanup.Context, cleanup: true}, cleanup.Nodes); err != nil {
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
func (a *admission) addGraph(g *graph, sources []*umpirespb.InstructionNode) error {
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
		g.nodes = append(g.nodes, &node{source: source, outcomes: map[umpirespb.InstructionOutcomeField]ir.Type{}, ancestors: map[int]bool{}})
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
		if !g.cleanup && g.context == umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER {
			controllers++
		}
		for _, n := range g.nodes {
			count, err := a.reservationCount(g, n)
			if err != nil {
				return err
			}
			if count > 0 {
				weights = append(weights, weighted{count: count, attempts: n.source.Bounds.MaxAttempts})
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
		if g.cleanup || g.context != umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER {
			return 0, invalid(ir.Unsupported, g.id, "only ordinary controller nodes may reserve activations")
		}
		target := a.graphIndex[reservation.GetEntrypointId()]
		if target == nil || target.cleanup || target.context != umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW && target.context != umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER {
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
		return catalog.BindType(&umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Any{Any: &umpirespb.AnyType{}}}}})
	}
	return catalog.BindType(&umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Message{Message: &umpirespb.NamedType{ProtobufType: string(descriptor.FullName())}}}}})
}
func nodePath(g *graph, n *node) string { return fmt.Sprintf("%s.%s", g.id, n.source.InstructionId) }
