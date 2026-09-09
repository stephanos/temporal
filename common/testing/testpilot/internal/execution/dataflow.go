package execution

import (
	"maps"
	"slices"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

// InstructionOpcode is the single mapping from a declared instruction to the opcode a Profile must
// authorize. Profile derivation reads it through the facade so a Case's instructions and the
// capabilities that authorize them cannot drift apart.
func InstructionOpcode(instruction *testpilotspb.Instruction) Opcode {
	if instruction == nil || isNil(instruction.Instruction) {
		return 0
	}
	switch instruction.Instruction.(type) {
	case *testpilotspb.Instruction_InvokeRpc:
		return InvokeRPC
	case *testpilotspb.Instruction_AwaitSlot:
		return AwaitSlot
	case *testpilotspb.Instruction_CompleteNexusOperation:
		return CompleteNexusOperation
	case *testpilotspb.Instruction_StartNexusOperation:
		return StartNexusOperation
	case *testpilotspb.Instruction_AwaitOutcome:
		return Await
	case *testpilotspb.Instruction_Finish:
		return Finish
	case *testpilotspb.Instruction_RespondNexus:
		return RespondNexus
	case *testpilotspb.Instruction_InjectFault:
		return InjectFault
	default:
		return 0
	}
}
func opcodeContext(opcode Opcode) testpilotspb.EntrypointKind {
	switch opcode {
	case InvokeRPC, AwaitSlot, CompleteNexusOperation, InjectFault:
		return testpilotspb.ENTRYPOINT_KIND_CONTROLLER
	case StartNexusOperation, Await, Finish:
		return testpilotspb.ENTRYPOINT_KIND_WORKFLOW
	case RespondNexus:
		return testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER
	default:
		return testpilotspb.ENTRYPOINT_KIND_UNSPECIFIED
	}
}
func scalarSchema(kind testpilotspb.ScalarKind) *testpilotspb.ValueType {
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: kind}}}}}
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
	n.opcode = InstructionOpcode(n.source.Instruction)
	if n.opcode == 0 || opcodeContext(n.opcode) != g.context || !a.capabilities[n.opcode] {
		return invalid(ir.Unsupported, nodePath(g, n), "unsupported instruction context or Driver capability")
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
		if err := a.role(start.EndpointRoleId, testpilotspb.ROLE_KIND_ENDPOINT); err != nil {
			return err
		}
	case Await:
		return bindAwait(g, n)
	case Finish:
		if n.source.Instruction.GetFinish() == nil {
			return invalid(ir.Malformed, nodePath(g, n), "nil Finish")
		}
	case RespondNexus:
		return a.bindNexusResponse(g, i, n)
	case InjectFault:
		return a.bindFault(g, n)
	default:
		return invalid(ir.Unsupported, nodePath(g, n), "unknown opcode")
	}
	return nil
}
func bindAwait(g *graph, n *node) error {
	reference := n.source.Instruction.GetAwaitOutcome().GetInstruction()
	dependency, exists := g.index[reference.GetInstructionId()]
	if !exists || reference.GetEntrypointId() != g.id || !n.ancestors[dependency] {
		return invalid(ir.Unavailable, nodePath(g, n), "Await requires an earlier local instruction")
	}
	if InstructionOpcode(g.nodes[dependency].source.Instruction) != StartNexusOperation {
		return invalid(ir.TypeMismatch, nodePath(g, n), "Await requires StartNexusOperation")
	}
	return nil
}
func (a *admission) bindNodeBounds(g *graph, n *node) error {
	bounds := n.source.Limits
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
	if err := a.role(rpc.EndpointRoleId, testpilotspb.ROLE_KIND_ENDPOINT); err != nil {
		return err
	}
	if !a.methods[rpc.EndpointRoleId][rpc.Method] {
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
	if response == nil || response.Kind < testpilotspb.NEXUS_RESPONSE_KIND_SYNCHRONOUS || response.Kind > testpilotspb.NEXUS_RESPONSE_KIND_ERROR {
		return invalid(ir.Malformed, nodePath(g, n), "invalid Nexus response")
	}
	if response.Kind == testpilotspb.NEXUS_RESPONSE_KIND_ASYNCHRONOUS {
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

// A fault names the task-queue role whose worker the Driver stops or resumes; the role's own
// resource binding identifies the queue, so the instruction carries no queue of its own.
func (a *admission) bindFault(g *graph, n *node) error {
	fault := n.source.Instruction.GetInjectFault()
	if fault == nil || fault.Kind < testpilotspb.FAULT_KIND_WORKER_STOP || fault.Kind > testpilotspb.FAULT_KIND_WORKER_RESUME {
		return invalid(ir.Malformed, nodePath(g, n), "fault injection requires a known fault kind")
	}
	// The role check does not go through a.role: a fault aimed at the wrong role kind is a
	// malformed instruction, not an unknown role reference.
	if a.roles[fault.RoleId] != testpilotspb.ROLE_KIND_TASK_QUEUE {
		return invalid(ir.Malformed, nodePath(g, n), "fault injection requires a declared task-queue role")
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
		var expected *testpilotspb.ValueType
		switch field.Field {
		case testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS:
			expected = &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Enumeration{Enumeration: &testpilotspb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.InstructionOutcomeStatus"}}}}}
		case testpilotspb.INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE:
			if n.opcode != InvokeRPC && n.opcode != CompleteNexusOperation {
				return invalid(ir.Unsupported, nodePath(g, n), "protocol code requires a controller protocol effect")
			}
			expected = scalarSchema(testpilotspb.SCALAR_KIND_TEXT)
		case testpilotspb.INSTRUCTION_OUTCOME_FIELD_SDK_FAILURE_CODE:
			if g.context == testpilotspb.ENTRYPOINT_KIND_CONTROLLER {
				return invalid(ir.Unsupported, nodePath(g, n), "SDK failure code requires an SDK instruction")
			}
			expected = scalarSchema(testpilotspb.SCALAR_KIND_TEXT)
		case testpilotspb.INSTRUCTION_OUTCOME_FIELD_DETAIL:
			expected = scalarSchema(testpilotspb.SCALAR_KIND_TEXT)
		case testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE:
			// RPC payloads are available only through declared response projections.
			if g.context == testpilotspb.ENTRYPOINT_KIND_CONTROLLER || typ.Opaque() || n.opcode == StartNexusOperation {
				return invalid(ir.Unsupported, nodePath(g, n), "VALUE requires an SDK result, not a controller outcome, opaque capability or StartNexusOperation handle")
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
		if source == nil || len(source.Targets) == 0 {
			return invalid(ir.Malformed, nodePath(g, n), "projection requires a path and sinks")
		}
		path, err := a.prepared.catalog.BindPath(output, source.Source, a.expressionLimits())
		if err != nil {
			return err
		}
		typ := path.Type()
		count := int64(1)
		switch source.Kind {
		case testpilotspb.PROJECTION_KIND_ONE:
		case testpilotspb.PROJECTION_KIND_EMIT_EACH:
			if typ.Cardinality() != ir.Repeated {
				return invalid(ir.TypeMismatch, nodePath(g, n), "EmitEach requires repeated values")
			}
			typ = typ.Element()
			count = a.prepared.source.Limits.MaxPathFanout
		default:
			return invalid(ir.Unknown, nodePath(g, n), "unknown projection cardinality")
		}
		lifts, emits, err := a.bindProjectionSinks(g, index, n, source, path, typ, seen)
		if err != nil {
			return err
		}
		if emits {
			if count > n.source.Limits.MaxEmittedEvents-events {
				return invalid(ir.LimitExceeded, nodePath(g, n), "projection emission exceeds instruction bound")
			}
			events += count
		}
		n.projections = append(n.projections, projection{path: path, cardinality: source.Kind, sinks: source.Targets, lifts: lifts})
	}
	return nil
}
func (a *admission) bindProjectionSinks(g *graph, index int, n *node, source *testpilotspb.ResponseProjection, path *ir.Path, typ ir.Type, seen map[string]bool) ([]*evidenceLift, bool, error) {
	emits := false
	lifts := make([]*evidenceLift, len(source.Targets))
	for i, sink := range source.Targets {
		if sink == nil || isNil(sink.Target) {
			return nil, false, invalid(ir.Malformed, nodePath(g, n), "missing projection sink")
		}
		var target ir.Type
		var exists bool
		var key string
		switch destination := sink.Target.(type) {
		case *testpilotspb.ProjectionTarget_SlotId:
			key = "slot:" + destination.SlotId
			target, exists = a.prepared.slots[destination.SlotId]
			if source.Kind == testpilotspb.PROJECTION_KIND_EMIT_EACH {
				return nil, false, invalid(ir.Unsupported, nodePath(g, n), "EmitEach cannot repeatedly assign an immutable Slot")
			}
			if err := a.addWriter(destination.SlotId, slotWriter{graph: g, node: index, optional: path.MayBeAbsent()}); err != nil {
				return nil, false, err
			}
		case *testpilotspb.ProjectionTarget_ObservationId:
			key = "observation:" + destination.ObservationId
			target, exists = a.observations[destination.ObservationId]
			emits = true
		case *testpilotspb.ProjectionTarget_ScopedEvidence:
			lift, err := a.bindEvidenceLift(g, n, destination.ScopedEvidence, typ)
			if err != nil {
				return nil, false, err
			}
			lifts[i], emits = lift, true
			key = "observation:" + lift.observationID
			if seen[key] {
				return nil, false, invalid(ir.Malformed, nodePath(g, n), "conflicting projection sinks")
			}
			seen[key] = true
			continue
		default:
			return nil, false, invalid(ir.Unsupported, nodePath(g, n), "unknown projection sink")
		}
		if !exists || target.Opaque() || !typ.Equal(target) {
			return nil, false, invalid(ir.TypeMismatch, nodePath(g, n), "projection type differs from declared sink")
		}
		if seen[key] {
			return nil, false, invalid(ir.Malformed, nodePath(g, n), "conflicting projection sinks")
		}
		seen[key] = true
	}

	return lifts, emits, nil
}

// bindEvidenceLift type-checks one declared ScopedEvidence lift against the value being projected.
// The sink Observation must be the exact ScopedEvidence message the scoped capability decodes, and
// every bound path must read a scalar the portable evidence domain admits, so a lift that cannot
// produce decodable evidence rejects at Prepare rather than at the first recorded event.
func (a *admission) bindEvidenceLift(g *graph, n *node, source *testpilotspb.ScopedEvidenceProjection, typ ir.Type) (*evidenceLift, error) {
	target, exists := a.observations[source.GetObservationId()]
	if !exists || target.Cardinality() != ir.Singular || !ir.SameMessage(target.Message(), (&testpilotspb.ScopedEvidence{}).ProtoReflect().Descriptor()) {
		return nil, invalid(ir.TypeMismatch, nodePath(g, n), "evidence lift requires an exact declared ScopedEvidence Observation")
	}
	if typ.Cardinality() != ir.Singular || typ.Message() == nil || typ.Opaque() || typ.Any() {
		return nil, invalid(ir.TypeMismatch, nodePath(g, n), "evidence lift requires a singular message projection")
	}
	if len(source.GetRules()) == 0 {
		return nil, invalid(ir.Malformed, nodePath(g, n), "evidence lift requires at least one rule")
	}
	// A source ordinal is dense per Run and only the emitting instruction counts it, so one source
	// belongs to one instruction on an entrypoint that activates exactly once. A worker entrypoint
	// activates per task and a second instruction would restart the count, and either would be
	// rejected by the verifier's ordering rather than here.
	if g.context != testpilotspb.ENTRYPOINT_KIND_CONTROLLER {
		return nil, invalid(ir.Unsupported, nodePath(g, n), "evidence lift requires a controller entrypoint")
	}
	lift := &evidenceLift{observationID: source.GetObservationId(), element: typ}
	owner := Coordinate{EntrypointID: g.id, InstructionID: n.source.InstructionId}
	for _, rule := range source.GetRules() {
		bound, err := a.bindEvidenceRule(g, n, rule, typ)
		if err != nil {
			return nil, err
		}
		if claimed, exists := a.evidenceSources[bound.source]; exists && claimed != owner {
			return nil, invalid(ir.Malformed, nodePath(g, n), "evidence source is already lifted by another instruction")
		}
		a.evidenceSources[bound.source] = owner
		lift.rules = append(lift.rules, *bound)
	}
	return lift, nil
}
func (a *admission) bindEvidenceRule(g *graph, n *node, source *testpilotspb.ScopedEvidenceRule, typ ir.Type) (*evidenceRule, error) {
	if !validID(source.GetSource()) || !validID(source.GetKind()) {
		return nil, invalid(ir.Malformed, nodePath(g, n), "evidence rule requires a source and a kind")
	}
	guard, err := a.prepared.catalog.BindPath(typ, source.GetGuard(), a.expressionLimits())
	if err != nil {
		return nil, err
	}
	// A presence read answers false where the field is absent, so it resolves either way and could
	// never select a rule.
	steps := guard.Steps()
	if guard.Fanout() || len(steps) == 0 || steps[len(steps)-1].Selector == ir.Presence {
		return nil, invalid(ir.Unsupported, nodePath(g, n), "evidence guard must select a value that can be absent")
	}
	if source.GetGuardEqualsText() != "" && (guard.Type().Cardinality() != ir.Singular || guard.Type().Scalar() != testpilotspb.SCALAR_KIND_TEXT) {
		return nil, invalid(ir.TypeMismatch, nodePath(g, n), "evidence guard equality requires a text guard")
	}
	operation, err := a.bindEvidencePath(g, n, typ, source.GetOperation(), evidenceKeyKinds...)
	if err != nil {
		return nil, err
	}
	bound := &evidenceRule{guard: guard, guardEquals: source.GetGuardEqualsText(), source: source.GetSource(), kind: source.GetKind(), operation: operation}
	// A scope binding is a Run coordinate and carries plain text on the wire; an evidence field is
	// a typed scalar the portable decoder reads as text, natural or boolean.
	scope, err := a.bindEvidenceBindings(g, n, typ, source.GetScope(), testpilotspb.SCALAR_KIND_TEXT)
	if err != nil {
		return nil, err
	}
	fields, err := a.bindEvidenceBindings(g, n, typ, source.GetFields(), evidenceFieldKinds...)
	if err != nil {
		return nil, err
	}
	bound.scope, bound.fields = scope, fields
	return bound, nil
}

// evidenceKeyKinds are the scalars an operation key may read: text, or an integer in its canonical
// decimal spelling.
var evidenceKeyKinds = append([]testpilotspb.ScalarKind{testpilotspb.SCALAR_KIND_TEXT}, evidenceIntegerKinds...)

// evidenceFieldKinds are the scalars a lifted evidence field may read; the portable evidence domain
// admits text, natural and boolean, and every integer kind narrows into a natural.
var evidenceFieldKinds = append([]testpilotspb.ScalarKind{
	testpilotspb.SCALAR_KIND_TEXT, testpilotspb.SCALAR_KIND_BOOLEAN,
}, evidenceIntegerKinds...)

// evidenceIntegerKinds narrow into the portable evidence domain's natural.
var evidenceIntegerKinds = []testpilotspb.ScalarKind{
	testpilotspb.SCALAR_KIND_NATURAL,
	testpilotspb.SCALAR_KIND_INT32, testpilotspb.SCALAR_KIND_INT64, testpilotspb.SCALAR_KIND_UINT32,
	testpilotspb.SCALAR_KIND_UINT64, testpilotspb.SCALAR_KIND_SINT32, testpilotspb.SCALAR_KIND_SINT64,
	testpilotspb.SCALAR_KIND_FIXED32, testpilotspb.SCALAR_KIND_FIXED64, testpilotspb.SCALAR_KIND_SFIXED32,
	testpilotspb.SCALAR_KIND_SFIXED64,
}

func (a *admission) bindEvidenceBindings(g *graph, n *node, typ ir.Type, sources []*testpilotspb.ScopedEvidenceBinding, kinds ...testpilotspb.ScalarKind) ([]evidenceBinding, error) {
	bound := make([]evidenceBinding, 0, len(sources))
	seen := map[string]bool{}
	for _, source := range sources {
		if source == nil || !validID(source.GetFieldId()) || seen[source.GetFieldId()] {
			return nil, invalid(ir.Malformed, nodePath(g, n), "evidence binding requires one unique declared field")
		}
		seen[source.GetFieldId()] = true
		switch supply := source.GetValue().(type) {
		case *testpilotspb.ScopedEvidenceBinding_Literal:
			if supply.Literal == "" {
				return nil, invalid(ir.Malformed, nodePath(g, n), "evidence literal binding requires a value")
			}
			bound = append(bound, evidenceBinding{fieldID: source.GetFieldId(), literal: supply.Literal})
		case *testpilotspb.ScopedEvidenceBinding_Path:
			path, err := a.bindEvidencePath(g, n, typ, supply.Path, kinds...)
			if err != nil {
				return nil, err
			}
			bound = append(bound, evidenceBinding{fieldID: source.GetFieldId(), path: path})
		default:
			return nil, invalid(ir.Malformed, nodePath(g, n), "evidence binding requires a path or a literal")
		}
	}
	return bound, nil
}
func (a *admission) bindEvidencePath(g *graph, n *node, typ ir.Type, source *testpilotspb.FieldPath, kinds ...testpilotspb.ScalarKind) (*ir.Path, error) {
	path, err := a.prepared.catalog.BindPath(typ, source, a.expressionLimits())
	if err != nil {
		return nil, err
	}
	read := path.Type()
	if read.Cardinality() != ir.Singular || read.Message() != nil || read.Enum() != nil || !slices.Contains(kinds, read.Scalar()) {
		return nil, invalid(ir.TypeMismatch, nodePath(g, n), "evidence binding reads an unsupported scalar")
	}
	return path, nil
}
func (a *admission) scope(g *graph, n *node) map[ir.Reference]ir.Binding {
	scope := map[ir.Reference]ir.Binding{}
	for id, typ := range a.prepared.slots {
		if writer, exists := a.writers[id]; exists && !typ.Opaque() && writer.graph != g &&
			(writer.graph.context != testpilotspb.ENTRYPOINT_KIND_CONTROLLER || g.context != testpilotspb.ENTRYPOINT_KIND_CONTROLLER) {
			continue
		}
		scope[ir.Reference{Kind: ir.SlotReference, ID: id}] = ir.Binding{Type: typ}
	}
	for index := range n.ancestors {
		previous := g.nodes[index]
		for field, typ := range previous.outcomes {
			scope[ir.Reference{Kind: ir.OutcomeReference, Entrypoint: g.id, ID: previous.source.InstructionId, Field: int32(field)}] = ir.Binding{Type: typ, Available: previous.source.Guard == nil && field != testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE}
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
		reference := ir.Reference{Kind: ir.OutcomeReference, Entrypoint: g.id, ID: id, Field: int32(testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE)}
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
			if reference.Kind == ir.OutcomeReference && reference.Field == int32(testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS) && literal.GetEnumValue() != nil && literal.GetEnumValue().Number == int32(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED) {
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
	boolean, err := a.prepared.catalog.BindType(scalarSchema(testpilotspb.SCALAR_KIND_BOOLEAN))
	if err != nil {
		return err
	}
	a.runID, err = a.prepared.catalog.BindType(scalarSchema(testpilotspb.SCALAR_KIND_TEXT))
	if err != nil {
		return err
	}
	for _, g := range a.prepared.graphs {
		for _, index := range g.order {
			if err := a.bindNodeDataflow(g, g.nodes[index], boolean); err != nil {
				return err
			}
		}
		g.runtimeWork = runtimeWorkLimit(g, a.prepared.source.Limits)
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
	bindIn := func(expressionScope map[ir.Reference]ir.Binding, value *testpilotspb.ProgramExpression, expected *ir.Type) (*ir.Expression, error) {
		if err := a.charge(int64(proto.Size(n.source.Guard)) + int64(proto.Size(value)) + 1); err != nil {
			return nil, err
		}
		_, expression, err := a.prepared.catalog.BindGuardedExpression(n.source.Guard, value, expected, expressionScope, a.expressionLimits())
		return expression, err
	}
	bind := func(value *testpilotspb.ProgramExpression, expected *ir.Type) (*ir.Expression, error) {
		return bindIn(scope, value, expected)
	}
	switch n.opcode {
	case InvokeRPC:
		inputScope := maps.Clone(scope)
		inputScope[ir.Reference{Kind: ir.EventReference, Field: int32(testpilotspb.RUN_EVENT_FIELD_RUN_ID)}] = ir.Binding{Type: a.runID, Available: true}
		err = a.bindAssignments(g, n, func(value *testpilotspb.ProgramExpression, expected *ir.Type) (*ir.Expression, error) {
			return bindIn(inputScope, value, expected)
		})
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
	case Await, InjectFault:
		// A fault names its target role statically; it binds no Program expression.
	default:
		return invalid(ir.Unsupported, nodePath(g, n), "unknown opcode")
	}
	if err != nil {
		return err
	}
	return nil
}
func (a *admission) bindAssignments(g *graph, n *node, bind func(*testpilotspb.ProgramExpression, *ir.Type) (*ir.Expression, error)) error {
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
			if previous.target.Conflicts(target) {
				return invalid(ir.Malformed, nodePath(g, n), "request assignments overlap")
			}
		}
		typ := target.Type()
		var environmentBindingID string
		valueSource := source.Value
		if reference, ok := source.Value.GetExpression().(*testpilotspb.ProgramExpression_Environment); ok {
			if reference.Environment == nil || typ.Cardinality() != ir.Singular || typ.Scalar() != testpilotspb.SCALAR_KIND_TEXT {
				return invalid(ir.TypeMismatch, nodePath(g, n), "environment reference requires a singular text destination")
			}
			environmentBindingID = reference.Environment.BindingId
			resolved, err := a.resolveEnvironment(environmentBindingID)
			if err != nil {
				return err
			}
			valueSource = &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_Text{Text: resolved}}}}
		}
		value, err := bind(valueSource, &typ)
		if err != nil {
			return err
		}
		n.assignments = append(n.assignments, assignment{target: target, value: value, environmentBindingID: environmentBindingID})
	}

	return nil
}
