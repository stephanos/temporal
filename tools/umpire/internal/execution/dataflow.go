package execution

import (
	"maps"
	"slices"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/ir"
	"google.golang.org/protobuf/proto"
)

func instructionOpcode(instruction *umpirespb.Instruction) Opcode {
	if instruction == nil || isNil(instruction.Instruction) {
		return 0
	}
	switch instruction.Instruction.(type) {
	case *umpirespb.Instruction_InvokeRpc:
		return InvokeRPC
	case *umpirespb.Instruction_AwaitSlot:
		return AwaitSlot
	case *umpirespb.Instruction_CompleteNexusOperation:
		return CompleteNexusOperation
	case *umpirespb.Instruction_StartNexusOperation:
		return StartNexusOperation
	case *umpirespb.Instruction_AwaitOutcome:
		return Await
	case *umpirespb.Instruction_Finish:
		return Finish
	case *umpirespb.Instruction_RespondNexus:
		return RespondNexus
	default:
		return 0
	}
}
func opcodeContext(opcode Opcode) umpirespb.EntrypointContext {
	switch opcode {
	case InvokeRPC, AwaitSlot, CompleteNexusOperation:
		return umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER
	case StartNexusOperation, Await, Finish:
		return umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW
	case RespondNexus:
		return umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER
	default:
		return umpirespb.ENTRYPOINT_CONTEXT_UNSPECIFIED
	}
}
func scalarSchema(kind umpirespb.ScalarKind) *umpirespb.ValueType {
	return &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Scalar{Scalar: &umpirespb.ScalarType{Kind: kind}}}}}
}
func (a *admission) bindInstructions() error {
	for _, g := range a.prepared.graphs {
		for i, n := range g.nodes {
			if err := a.bindInstruction(g, i, n); err != nil {
				return err
			}
		}
	}
	return nil
}
func (a *admission) bindInstruction(g *graph, i int, n *node) error {
	n.opcode = instructionOpcode(n.source.Instruction)
	if n.opcode == 0 || opcodeContext(n.opcode) != g.context || !a.capabilities[n.opcode] {
		return invalid(ir.Unsupported, nodePath(g, n), "unsupported instruction context or Host capability")
	}
	if err := a.bindNodeBounds(g, n); err != nil {
		return err
	}
	if err := a.bindOutcomes(g, n); err != nil {
		return err
	}
	switch n.opcode {
	case InvokeRPC:
		return a.bindRPC(g, i, n)
	case AwaitSlot:
		if _, exists := a.prepared.slots[n.source.Instruction.GetAwaitSlot().GetSlotId()]; !exists {
			return invalid(ir.Unknown, nodePath(g, n), "AwaitSlot requires a declared Slot")
		}
	case CompleteNexusOperation:
		typ, exists := a.prepared.slots[n.source.Instruction.GetCompleteNexusOperation().GetCapabilitySlotId()]
		if !exists || !typ.Opaque() {
			return invalid(ir.TypeMismatch, nodePath(g, n), "completion requires a capability Slot")
		}
	case StartNexusOperation:
		start := n.source.Instruction.GetStartNexusOperation()
		if start == nil || !validID(start.Service) || !validID(start.Operation) {
			return invalid(ir.Malformed, nodePath(g, n), "invalid Nexus start")
		}
		if err := a.role(start.EndpointRoleId, umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT); err != nil {
			return err
		}
	case Await:
		reference := n.source.Instruction.GetAwaitOutcome().GetInstruction()
		dependency, exists := g.index[reference.GetInstructionId()]
		if !exists || reference.GetEntrypointId() != g.id || !n.ancestors[dependency] {
			return invalid(ir.Unavailable, nodePath(g, n), "Await requires an earlier local instruction")
		}
	case Finish:
		if n.source.Instruction.GetFinish() == nil {
			return invalid(ir.Malformed, nodePath(g, n), "nil Finish")
		}
	case RespondNexus:
		return a.bindNexusResponse(g, i, n)
	default:
		return invalid(ir.Unsupported, nodePath(g, n), "unknown opcode")
	}
	return nil
}
func (a *admission) bindNodeBounds(g *graph, n *node) error {
	bounds := n.source.Bounds
	limits := a.prepared.source.Limits
	duration := limits.MaxTotalDurationMilliseconds
	if g.cleanup {
		duration = limits.MaxCleanupDurationMilliseconds
	}
	if bounds == nil || bounds.TimeoutMilliseconds <= 0 || bounds.TimeoutMilliseconds > duration || bounds.MaxAttempts <= 0 || bounds.MaxAttempts > a.prepared.policy.Limits.MaxAttempts || bounds.MaxEmittedEvents < 0 || bounds.MaxEmittedEvents > limits.MaxRunEvents || bounds.MaxResponseBytes < 0 || bounds.MaxResponseBytes > limits.MaxResponseBytes || n.opcode == InvokeRPC && bounds.MaxResponseBytes == 0 {
		return invalid(ir.LimitExceeded, nodePath(g, n), "instruction bounds exceed Program limits")
	}

	return nil
}
func (a *admission) bindRPC(g *graph, i int, n *node) error {
	rpc := n.source.Instruction.GetInvokeRpc()
	if rpc == nil {
		return invalid(ir.Malformed, nodePath(g, n), "nil RPC")
	}
	if err := a.role(rpc.EndpointRoleId, umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT); err != nil {
		return err
	}
	if !slices.Contains(a.allowed[rpc.EndpointRoleId].Methods, rpc.Method) {
		return invalid(ir.Unsupported, nodePath(g, n), "unauthorized RPC method")
	}
	method, err := a.prepared.catalog.Method(rpc.Method)
	if err != nil {
		return err
	}
	n.method = method
	return a.bindProjections(g, i, n)
}
func (a *admission) bindNexusResponse(g *graph, i int, n *node) error {
	response := n.source.Instruction.GetRespondNexus()
	if response == nil || response.Kind < umpirespb.NEXUS_RESPONSE_KIND_SYNCHRONOUS || response.Kind > umpirespb.NEXUS_RESPONSE_KIND_ERROR {
		return invalid(ir.Malformed, nodePath(g, n), "invalid Nexus response")
	}
	if response.Kind == umpirespb.NEXUS_RESPONSE_KIND_ASYNCHRONOUS {
		typ, exists := a.prepared.slots[response.CapabilitySlotId]
		if !exists || !typ.Opaque() {
			return invalid(ir.TypeMismatch, nodePath(g, n), "async response requires a capability Slot")
		}
		if err := a.addWriter(response.CapabilitySlotId, slotWriter{graph: g, node: i}); err != nil {
			return err
		}
	} else if response.CapabilitySlotId != "" {
		return invalid(ir.Unsupported, nodePath(g, n), "only async responses publish capabilities")
	}

	return nil
}
func (a *admission) bindOutcomes(g *graph, n *node) error {
	if n.source.Outcome == nil {
		return invalid(ir.Malformed, nodePath(g, n), "outcome schema is required")
	}
	for _, field := range n.source.Outcome.Fields {
		if field == nil {
			return invalid(ir.Malformed, nodePath(g, n), "nil outcome field")
		}
		if _, exists := n.outcomes[field.Field]; exists {
			return invalid(ir.Malformed, nodePath(g, n), "duplicate outcome field")
		}
		typ, err := a.prepared.catalog.BindType(field.Type)
		if err != nil {
			return err
		}
		var expected *umpirespb.ValueType
		switch field.Field {
		case umpirespb.INSTRUCTION_OUTCOME_FIELD_STATUS:
			expected = &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Enumeration{Enumeration: &umpirespb.NamedType{ProtobufType: "temporal.server.api.umpire.v1.InstructionOutcomeStatus"}}}}}
		case umpirespb.INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE:
			if n.opcode != InvokeRPC && n.opcode != CompleteNexusOperation {
				return invalid(ir.Unsupported, nodePath(g, n), "protocol code requires a controller protocol effect")
			}
			expected = scalarSchema(umpirespb.SCALAR_KIND_TEXT)
		case umpirespb.INSTRUCTION_OUTCOME_FIELD_SDK_FAILURE_CODE:
			if g.context == umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER {
				return invalid(ir.Unsupported, nodePath(g, n), "SDK failure code requires an SDK instruction")
			}
			expected = scalarSchema(umpirespb.SCALAR_KIND_TEXT)
		case umpirespb.INSTRUCTION_OUTCOME_FIELD_DETAIL:
			expected = scalarSchema(umpirespb.SCALAR_KIND_TEXT)
		case umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE:
			// RPC payloads are available only through declared response projections.
			if g.context == umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER || typ.Opaque() {
				return invalid(ir.Unsupported, nodePath(g, n), "only SDK outcomes may declare a nonopaque value")
			}
		default:
			return invalid(ir.Unknown, nodePath(g, n), "unknown outcome field")
		}
		if expected != nil && !proto.Equal(expected, field.Type) {
			return invalid(ir.TypeMismatch, nodePath(g, n), "outcome field has the wrong type")
		}
		n.outcomes[field.Field] = typ
	}
	return nil
}
func (a *admission) addWriter(id string, writer slotWriter) error {
	if _, exists := a.writers[id]; exists {
		return invalid(ir.Malformed, "slots", "Slot has multiple writers")
	}
	a.writers[id] = writer
	return nil
}
func (a *admission) bindProjections(g *graph, index int, n *node) error {
	output, err := messageType(a.prepared.catalog, n.method.Output())
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	var events int64
	for _, source := range n.source.Instruction.GetInvokeRpc().ResponseProjections {
		if source == nil || len(source.Sinks) == 0 {
			return invalid(ir.Malformed, nodePath(g, n), "projection requires a path and sinks")
		}
		path, err := a.prepared.catalog.BindPath(output, source.Source, a.expressionLimits())
		if err != nil {
			return err
		}
		typ := path.Type()
		count := int64(1)
		switch source.Cardinality {
		case umpirespb.PROJECTION_CARDINALITY_ONE:
		case umpirespb.PROJECTION_CARDINALITY_EMIT_EACH:
			if typ.Cardinality() != ir.Repeated {
				return invalid(ir.TypeMismatch, nodePath(g, n), "EmitEach requires repeated values")
			}
			typ = typ.Element()
			count = a.prepared.source.Limits.MaxPathFanout
		default:
			return invalid(ir.Unknown, nodePath(g, n), "unknown projection cardinality")
		}
		emits, err := a.bindProjectionSinks(g, index, n, source, path, typ, seen)
		if err != nil {
			return err
		}
		if emits {
			if count > n.source.Bounds.MaxEmittedEvents-events {
				return invalid(ir.LimitExceeded, nodePath(g, n), "projection emission exceeds instruction bound")
			}
			events += count
		}
		n.projections = append(n.projections, projection{path: path, cardinality: source.Cardinality, sinks: source.Sinks})
	}
	return nil
}
func (a *admission) bindProjectionSinks(g *graph, index int, n *node, source *umpirespb.ResponseProjection, path *ir.Path, typ ir.Type, seen map[string]bool) (bool, error) {
	emits := false
	for _, sink := range source.Sinks {
		if sink == nil || isNil(sink.Sink) {
			return false, invalid(ir.Malformed, nodePath(g, n), "missing projection sink")
		}
		var target ir.Type
		var exists bool
		var key string
		switch destination := sink.Sink.(type) {
		case *umpirespb.ProjectionSink_SlotId:
			key = "slot:" + destination.SlotId
			target, exists = a.prepared.slots[destination.SlotId]
			if source.Cardinality == umpirespb.PROJECTION_CARDINALITY_EMIT_EACH {
				return false, invalid(ir.Unsupported, nodePath(g, n), "EmitEach cannot repeatedly assign an immutable Slot")
			}
			if err := a.addWriter(destination.SlotId, slotWriter{graph: g, node: index, optional: path.MayBeAbsent()}); err != nil {
				return false, err
			}
		case *umpirespb.ProjectionSink_ObservationId:
			key = "observation:" + destination.ObservationId
			target, exists = a.observations[destination.ObservationId]
			emits = true
		default:
			return false, invalid(ir.Unsupported, nodePath(g, n), "unknown projection sink")
		}
		if !exists || target.Opaque() || !typ.Equal(target) {
			return false, invalid(ir.TypeMismatch, nodePath(g, n), "projection type differs from declared sink")
		}
		if seen[key] {
			return false, invalid(ir.Malformed, nodePath(g, n), "conflicting projection sinks")
		}
		seen[key] = true
	}

	return emits, nil
}
func (a *admission) scope(g *graph, n *node) map[ir.Reference]ir.Binding {
	scope := map[ir.Reference]ir.Binding{}
	for id, typ := range a.prepared.slots {
		if writer, exists := a.writers[id]; exists && !typ.Opaque() && writer.graph != g &&
			(writer.graph.context != umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER || g.context != umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER) {
			continue
		}
		scope[ir.Reference{Kind: ir.SlotReference, ID: id}] = ir.Binding{Type: typ}
	}
	for index := range n.ancestors {
		previous := g.nodes[index]
		for field, typ := range previous.outcomes {
			scope[ir.Reference{Kind: ir.OutcomeReference, Entrypoint: g.id, ID: previous.source.InstructionId, Field: int32(field)}] = ir.Binding{Type: typ, Available: previous.source.Guard == nil && field != umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE}
		}
	}
	return scope
}

// Successful dependencies establish nonoptional projections and successful AwaitSlot readiness.
func (a *admission) successScope(g *graph, n *node, guard *ir.Expression, scope map[ir.Reference]ir.Binding) {
	for id := range successFacts(guard) {
		index, exists := g.index[id]
		if !exists || !n.ancestors[index] {
			continue
		}
		for slotID, writer := range a.writers {
			if writer.graph == g && writer.node == index && !writer.optional {
				reference := ir.Reference{Kind: ir.SlotReference, ID: slotID}
				binding := scope[reference]
				binding.Available = true
				scope[reference] = binding
			}
		}
		previous := g.nodes[index]
		if previous.opcode == AwaitSlot {
			reference := ir.Reference{Kind: ir.SlotReference, ID: previous.source.Instruction.GetAwaitSlot().SlotId}
			binding := scope[reference]
			binding.Available = true
			scope[reference] = binding
		}
		reference := ir.Reference{Kind: ir.OutcomeReference, Entrypoint: g.id, ID: id, Field: int32(umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE)}
		if binding, exists := scope[reference]; exists {
			binding.Available = true
			scope[reference] = binding
		}
	}
}
func successFacts(expression *ir.Expression) map[string]bool {
	result := map[string]bool{}
	if expression == nil {
		return result
	}
	children := expression.Children()
	switch expression.Operator() {
	case ir.Equals:
		for i := range 2 {
			reference := children[i].Reference()
			literal := children[1-i].Literal()
			if reference.Kind == ir.OutcomeReference && reference.Field == int32(umpirespb.INSTRUCTION_OUTCOME_FIELD_STATUS) && literal.GetEnumValue() != nil && literal.GetEnumValue().Number == int32(umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED) {
				result[reference.ID] = true
			}
		}
	case ir.All, ir.Any:
		for i, child := range children {
			facts := successFacts(child)
			if expression.Operator() == ir.All || i == 0 {
				maps.Copy(result, facts)
			} else {
				for id := range result {
					if !facts[id] {
						delete(result, id)
					}
				}
			}
		}
	default:
	}
	return result
}
func (a *admission) bindDataflow() error {
	boolean, err := a.prepared.catalog.BindType(scalarSchema(umpirespb.SCALAR_KIND_BOOLEAN))
	if err != nil {
		return err
	}
	for _, g := range a.prepared.graphs {
		for _, index := range g.order {
			if err := a.bindNodeDataflow(g, g.nodes[index], boolean); err != nil {
				return err
			}
		}
	}
	return nil
}
func (a *admission) bindNodeDataflow(g *graph, n *node, boolean ir.Type) error {
	var err error
	if err := a.charge(int64(len(a.prepared.slots)) + int64(len(n.ancestors))*6); err != nil {
		return err
	}
	scope := a.scope(g, n)
	if err := a.charge(int64(proto.Size(n.source.Guard)) + 1); err != nil {
		return err
	}
	if n.source.Guard != nil {
		n.guard, err = a.prepared.catalog.BindExpression(n.source.Guard, &boolean, scope, a.expressionLimits())
		if err != nil {
			return err
		}
	}
	if err := a.charge(int64(len(successFacts(n.guard))) * (int64(len(a.writers)) + 1)); err != nil {
		return err
	}
	a.successScope(g, n, n.guard, scope)
	bind := func(value *umpirespb.ValueExpression, expected *ir.Type) (*ir.Expression, error) {
		if err := a.charge(int64(proto.Size(n.source.Guard)) + int64(proto.Size(value)) + 1); err != nil {
			return nil, err
		}
		_, expression, err := a.prepared.catalog.BindGuardedExpression(n.source.Guard, value, expected, scope, a.expressionLimits())
		return expression, err
	}
	switch n.opcode {
	case InvokeRPC:
		err = a.bindAssignments(g, n, bind)
	case AwaitSlot:
		if _, exists := a.writers[n.source.Instruction.GetAwaitSlot().SlotId]; !exists {
			return invalid(ir.Unavailable, nodePath(g, n), "awaited Slot has no writer")
		}
	case CompleteNexusOperation:
		instruction := n.source.Instruction.GetCompleteNexusOperation()
		if !scope[ir.Reference{Kind: ir.SlotReference, ID: instruction.CapabilitySlotId}].Available {
			return invalid(ir.Unavailable, nodePath(g, n), "completion requires successful AwaitSlot dependency")
		}
		n.input, err = bind(instruction.Result, nil)
	case StartNexusOperation:
		n.input, err = bind(n.source.Instruction.GetStartNexusOperation().Input, nil)
	case Finish:
		n.input, err = bind(n.source.Instruction.GetFinish().Result, nil)
	case RespondNexus:
		n.input, err = bind(n.source.Instruction.GetRespondNexus().Result, nil)
	case Await:
	default:
		return invalid(ir.Unsupported, nodePath(g, n), "unknown opcode")
	}
	if err != nil {
		return err
	}
	return nil
}
func (a *admission) bindAssignments(g *graph, n *node, bind func(*umpirespb.ValueExpression, *ir.Type) (*ir.Expression, error)) error {
	input, err := messageType(a.prepared.catalog, n.method.Input())
	if err != nil {
		return err
	}
	for _, source := range n.source.Instruction.GetInvokeRpc().RequestAssignments {
		if source == nil {
			return invalid(ir.Malformed, nodePath(g, n), "nil request assignment")
		}
		target, err := a.prepared.catalog.BindPath(input, source.Target, a.expressionLimits())
		if err != nil {
			return err
		}
		if target.Fanout() {
			return invalid(ir.Unsupported, nodePath(g, n), "assignment cannot fan out across destination elements")
		}
		for _, step := range target.Steps() {
			if step.Selector == ir.Presence {
				return invalid(ir.Unsupported, nodePath(g, n), "presence is not an assignment destination")
			}
		}
		for _, previous := range n.assignments {
			if err := a.charge(int64(len(previous.target.Steps())+len(target.Steps())) + 1); err != nil {
				return err
			}
			if pathsConflict(previous.target, target) {
				return invalid(ir.Malformed, nodePath(g, n), "request assignments overlap")
			}
		}
		typ := target.Type()
		value, err := bind(source.Value, &typ)
		if err != nil {
			return err
		}
		n.assignments = append(n.assignments, assignment{target: target, value: value})
	}

	return nil
}
func pathsConflict(left, right *ir.Path) bool {
	a, b := left.Steps(), right.Steps()
	for i := 0; i < min(len(a), len(b)); i++ {
		if a[i].Field != b[i].Field {
			group := a[i].Field.ContainingOneof()
			return group != nil && group == b[i].Field.ContainingOneof()
		}
		if a[i].Selector == ir.MapKey && b[i].Selector == ir.MapKey && !proto.Equal(a[i].Key, b[i].Key) {
			return false
		}
		if a[i].Selector != b[i].Selector && a[i].Selector != ir.Oneof && b[i].Selector != ir.Oneof {
			return true
		}
	}
	return true
}
