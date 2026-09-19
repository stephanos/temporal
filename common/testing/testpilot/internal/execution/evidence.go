package execution

import (
	"context"
	"fmt"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// historyEventMessage is the recorded event a history declaration reads, whose attributes oneof
// names the kind.
const historyEventMessage = "temporal.api.history.v1.HistoryEvent"

// bindEvidence admits the Program's evidence declarations: each identity once, each source and key
// path once, every path typed against the recorded value its source supplies. A declaration a Run
// Event feeds is lifted by the scheduler as it records the event; a read declaration is polled by
// a ReadEvidence instruction; a history declaration is named by a history read's lift rule.
func (a *admission) bindEvidence(p *testpilotspb.Program) error {
	a.prepared.evidence = map[string]*evidenceDeclaration{}
	for _, observation := range a.prepared.view.observations {
		if observation.Type.Cardinality() == ir.Singular && ir.SameMessage(observation.Type.Message(), (&testpilotspb.CorrelatedEvidence{}).ProtoReflect().Descriptor()) {
			if a.prepared.correlatedObservationID != "" {
				a.prepared.correlatedObservationID = ""
				break
			}
			a.prepared.correlatedObservationID = observation.ID
		}
	}
	keys := map[string]bool{}
	for index, source := range p.Evidence {
		location := fmt.Sprintf("program.evidence[%d]", index)
		if err := a.charge(1); err != nil {
			return err
		}
		if source == nil || !validID(source.GetEvidenceId()) || !validID(source.GetEvidenceSource()) || source.GetOperation() == "" {
			return invalid(ir.Malformed, location, "evidence declaration requires an identity, a source and an operation key path")
		}
		if _, exists := a.prepared.evidence[source.EvidenceId]; exists {
			return invalid(ir.Malformed, location+".evidence_id", "duplicate evidence declaration")
		}
		bound, err := a.bindEvidenceDeclaration(location, source)
		if err != nil {
			return err
		}
		key := bound.source + "\x00" + bound.operation.Text()
		if keys[key] {
			return invalid(ir.Malformed, location, "evidence source and operation key path are declared twice")
		}
		keys[key] = true
		a.prepared.evidence[bound.id] = bound
		if bound.kind == RunEventSource {
			a.prepared.runEventLifts = append(a.prepared.runEventLifts, bound)
		}
		view := EvidenceDeclaration{ID: bound.id, Source: bound.source, Kind: bound.kind}
		for _, field := range bound.fields {
			view.Fields = append(view.Fields, field.fieldID)
		}
		a.prepared.view.evidence = append(a.prepared.view.evidence, view)
	}
	if len(a.prepared.runEventLifts) > 0 && a.prepared.correlatedObservationID == "" {
		return invalid(ir.TypeMismatch, "program.evidence", "a Run Event declaration requires exactly one declared CorrelatedEvidence Observation")
	}
	return nil
}

func (a *admission) bindEvidenceDeclaration(location string, source *testpilotspb.EvidenceDeclaration) (*evidenceDeclaration, error) {
	bound := &evidenceDeclaration{id: source.EvidenceId, source: source.EvidenceSource}
	switch recorded := source.Source.(type) {
	case *testpilotspb.EvidenceDeclaration_HistoryEvent:
		bound.kind = HistoryEventSource
		element, err := a.prepared.catalog.BindType(namedMessageType(historyEventMessage))
		if err != nil {
			return nil, err
		}
		arm := recorded.HistoryEvent.GetAttributesField()
		oneof := element.Message().Oneofs().ByName("attributes")
		if oneof == nil || oneof.Fields().ByName(protoreflect.Name(arm)) == nil {
			return nil, invalid(ir.Unknown, location+".history_event.attributes_field", "history evidence requires an attributes arm of the recorded event")
		}
		bound.element, bound.attributesField = element, arm
		bound.guard, err = a.liftGuard(location+".history_event", element, presentExpression(projectedPath("attributes<"+arm+">")))
		if err != nil {
			return nil, err
		}
	case *testpilotspb.EvidenceDeclaration_RunEvent:
		bound.kind = RunEventSource
		kind := recorded.RunEvent.GetKind()
		payload := ir.RunEventPayloadOf(kind)
		if kind <= 0 || kind > ir.MaxRunEventKind || payload.Arm == "" {
			return nil, invalid(ir.Unsupported, location+".run_event.kind", "Run Event evidence requires a kind that carries a payload")
		}
		element, ok := a.prepared.catalog.RunEventPayloadType(payload.Arm)
		if !ok {
			return nil, invalid(ir.Unknown, location+".run_event.kind", "Run Event payload arm is not in the catalog")
		}
		bound.element, bound.runEventKind, bound.payloadArm = element, kind, payload.Arm
		var err error
		bound.guard, err = a.liftGuard(location+".run_event", element, &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}}}})
		if err != nil {
			return nil, err
		}
	case *testpilotspb.EvidenceDeclaration_Read:
		bound.kind = ReadSource
		method, err := a.prepared.catalog.Method(recorded.Read.GetMethod())
		if err != nil {
			return nil, err
		}
		output, err := messageType(a.prepared.catalog, method.Output())
		if err != nil {
			return nil, err
		}
		path, err := a.prepared.catalog.BindPath(output, location+".read.path", recorded.Read.GetPath(), a.expressionLimits())
		if err != nil {
			return nil, err
		}
		element := path.Type()
		if element.Cardinality() != ir.Repeated {
			return nil, invalid(ir.TypeMismatch, location+".read.path", "read evidence requires a repeated field")
		}
		element = element.Element()
		if element.Message() == nil || element.Opaque() || element.Any() {
			return nil, invalid(ir.TypeMismatch, location+".read.path", "read evidence requires repeated messages")
		}
		bound.element, bound.method, bound.readPath = element, method, path
	default:
		return nil, invalid(ir.Malformed, location+".source", "evidence declaration requires a source")
	}
	var err error
	if bound.operation, err = a.bindEvidencePath(location, bound.element, location+".operation", source.Operation, evidenceKeyKinds...); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for index, scope := range source.Scope {
		text, isText := scope.GetValue().GetValue().(*testpilotspb.Value_TextValue)
		if scope == nil || !validID(scope.GetFieldId()) || seen[scope.GetFieldId()] || !isText || text.TextValue == "" {
			return nil, invalid(ir.Malformed, fmt.Sprintf("%s.scope[%d]", location, index), "evidence scope requires one unique declared field with a text value")
		}
		seen[scope.FieldId] = true
		bound.scope = append(bound.scope, evidenceBinding{fieldID: scope.FieldId, literal: text.TextValue})
	}
	fields := map[string]bool{}
	for index, field := range source.Fields {
		if field == nil || !validID(field.GetFieldId()) || fields[field.GetFieldId()] {
			return nil, invalid(ir.Malformed, fmt.Sprintf("%s.fields[%d]", location, index), "evidence field requires one unique declared field")
		}
		fields[field.FieldId] = true
		path, err := a.bindEvidencePath(location, bound.element, fmt.Sprintf("%s.fields[%d].path", location, index), field.GetPath(), evidenceFieldKinds...)
		if err != nil {
			return nil, err
		}
		bound.fields = append(bound.fields, evidenceBinding{fieldID: field.FieldId, path: path})
	}
	return bound, nil
}

// liftGuard binds a boolean over one recorded value in the evidence-lift context.
func (a *admission) liftGuard(location string, element ir.Type, source *testpilotspb.Expression) (*ir.Expression, error) {
	boolean, err := a.prepared.catalog.BindType(scalarSchema(testpilotspb.SCALAR_KIND_BOOLEAN))
	if err != nil {
		return nil, err
	}
	projected := map[ir.Reference]ir.Binding{{Kind: ir.ProjectedValueReference}: {Type: element, Available: true}}
	return a.prepared.catalog.BindExpression(ir.Site{Context: ir.EvidenceLiftContext, Path: location}, source, &boolean, projected, a.expressionLimits())
}

func namedMessageType(name string) *testpilotspb.ValueType {
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: name}}}}}
}
func presentExpression(operand *testpilotspb.Expression) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Present{Present: &testpilotspb.PresentExpression{Operand: operand}}}
}
func projectedPath(path string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Path{Path: &testpilotspb.PathExpression{
		Operand: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_ProjectedValue{ProjectedValue: &testpilotspb.ProjectedValueReference{}}}}},
		Path:    path,
	}}}
}

// bindDeclaredRule admits a lift rule that names a declaration instead of spelling itself: the
// declaration is a history event kind, the projected value is the recorded event, and the rule's
// guard, scope, key and fields are the declaration's.
func (a *admission) bindDeclaredRule(g *graph, n *node, location string, source *testpilotspb.CorrelatedEvidenceRule, typ ir.Type) (*evidenceRule, error) {
	if source.GetGuard() != nil || len(source.GetScope()) > 0 || len(source.GetFields()) > 0 || source.GetOperation() != "" || source.GetKind() != "" || source.GetEvidenceSource() != "" {
		return nil, invalid(ir.Malformed, location, "a rule naming a declaration spells nothing else")
	}
	declaration, exists := a.prepared.evidence[source.GetEvidenceId()]
	if !exists {
		return nil, invalid(ir.Unknown, location+".evidence_id", "evidence rule names an undeclared evidence kind")
	}
	if declaration.kind != HistoryEventSource {
		return nil, invalid(ir.Unsupported, location+".evidence_id", "only a history event declaration is lifted by a read")
	}
	if !typ.Equal(declaration.element) {
		return nil, invalid(ir.TypeMismatch, nodePath(g, n), "declared history evidence requires a history event read")
	}
	rule := declaration.lift("", declaration.guard).rules[0]
	return &rule, nil
}

// bindReadEvidence admits a controller poll of a read declaration: the endpoint role authorizes the
// declaration's method, the poll interval fits the instruction's timeout, and the instruction's one
// response read lifts every element `until` selects into the Program's CorrelatedEvidence
// Observation, under the declaration's coordinates.
func (a *admission) bindReadEvidence(g *graph, n *node) error {
	read := n.source.Instruction.GetReadEvidence()
	if read == nil {
		return invalid(ir.Malformed, nodePath(g, n), "nil ReadEvidence")
	}
	declaration, exists := a.prepared.evidence[read.GetEvidenceId()]
	if !exists {
		return invalid(ir.Unknown, expressionPath(g, n, "instruction.read_evidence.evidence_id"), "ReadEvidence names an undeclared evidence kind")
	}
	if declaration.kind != ReadSource {
		return invalid(ir.TypeMismatch, expressionPath(g, n, "instruction.read_evidence.evidence_id"), "ReadEvidence requires a read declaration")
	}
	if err := a.role(read.EndpointRoleId, testpilotspb.ROLE_KIND_ENDPOINT); err != nil {
		return err
	}
	if !a.methods[read.EndpointRoleId][methodName(declaration.method)] {
		return invalid(ir.Unsupported, nodePath(g, n), "unauthorized RPC method")
	}
	if a.prepared.correlatedObservationID == "" {
		return invalid(ir.TypeMismatch, nodePath(g, n), "ReadEvidence requires exactly one declared CorrelatedEvidence Observation")
	}
	if read.PollIntervalMilliseconds <= 0 {
		return invalid(ir.Malformed, expressionPath(g, n, "instruction.read_evidence.poll_interval_milliseconds"), "ReadEvidence requires a positive poll interval")
	}
	if read.PollIntervalMilliseconds > n.timeoutMilliseconds {
		return invalid(ir.LimitExceeded, expressionPath(g, n, "instruction.read_evidence.poll_interval_milliseconds"), "poll interval exceeds the instruction timeout")
	}
	if a.prepared.limits.MaxPathFanout > a.prepared.limits.MaxInstructionEmittedEvents {
		return invalid(ir.LimitExceeded, nodePath(g, n), "read evidence emission exceeds instruction bound")
	}
	until, err := a.liftGuard(expressionPath(g, n, "instruction.read_evidence.until"), declaration.element, read.Until)
	if err != nil {
		return err
	}
	owner := contract.Coordinate{EntrypointID: g.id, InstructionID: n.source.InstructionId}
	if claimed, exists := a.evidenceSources[declaration.source]; exists && claimed != owner {
		return invalid(ir.Malformed, nodePath(g, n), "evidence source is already lifted by another instruction")
	}
	a.evidenceSources[declaration.source] = owner
	lift := declaration.lift(a.prepared.correlatedObservationID, until)
	n.method, n.until, n.pollIntervalMilliseconds = declaration.method, until, read.PollIntervalMilliseconds
	n.responseReads = []responseRead{{
		path: declaration.readPath, cardinality: testpilotspb.READ_CARDINALITY_EMIT_EACH,
		targets: []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_CorrelatedEvidence{CorrelatedEvidence: &testpilotspb.CorrelatedEvidenceProjection{ObservationId: lift.observationID}}}},
		lifts:   []*evidenceLift{lift},
	}}
	return nil
}

func methodName(method protoreflect.MethodDescriptor) string {
	return "/" + string(method.Parent().FullName()) + "/" + string(method.Name())
}

// readSatisfied reports whether an element of the poll's declared path satisfies the instruction's
// `until`, which ends a ReadEvidence poll.
func (a *activationValues) readSatisfied(ctx context.Context, c contract.Coordinate, response proto.Message, limit int64) (bool, int64, error) {
	n, err := a.instruction(c)
	if err != nil {
		return false, 0, err
	}
	w, err := a.newWork(ctx, limit)
	if err != nil {
		return false, 0, err
	}
	if n.opcode != contract.ReadEvidence || len(n.responseReads) != 1 || n.until == nil {
		return false, w.work, invalid(ir.TypeMismatch, "read_evidence", "poll requires a ReadEvidence instruction")
	}
	if isNil(response) {
		return false, w.work, invalid(ir.Unavailable, "read_evidence", "poll returned no response")
	}
	snapshot, work, err := ir.SnapshotMessage(ctx, response, n.method.Output(), w.remaining(a.store.program.limits.MaxInstructionResponseBytes))
	w.work += work
	if err != nil {
		return false, w.work, err
	}
	read := n.responseReads[0]
	value, work, err := read.path.Read(ctx, snapshot, w.remaining(a.store.program.limits.MaxInstructionResponseBytes))
	w.work += work
	if err != nil {
		return false, w.work, err
	}
	for _, element := range value.GetListValue().GetValues() {
		if err := w.charge(1); err != nil {
			return false, w.work, err
		}
		accepted, work, err := n.until.EvaluateExecution(w.ctx, func(reference ir.Reference) *testpilotspb.Value {
			if reference.Kind == ir.ProjectedValueReference {
				return element
			}
			return nil
		}, w.limits.Work-w.work)
		w.work += work
		if err != nil {
			return false, w.work, err
		}
		if accepted.GetBoolValue() {
			return true, w.work, nil
		}
	}
	return false, w.work, nil
}

// liftRunEvents attaches, to each recorded fact whose kind a Run Event declaration names, the
// evidence lifted from the fact's payload. Ordinals are dense per source across the Run, in
// recording order.
func (s *scheduler) liftRunEvents(ctx context.Context, a *activationValues, facts []*testpilotspb.RunEvent) error {
	program := s.values.program
	if len(program.runEventLifts) == 0 {
		return nil
	}
	w, err := a.newWork(ctx, a.workLimit())
	if err != nil {
		return err
	}
	for _, fact := range facts {
		for _, declaration := range program.runEventLifts {
			if fact.Kind != declaration.runEventKind {
				continue
			}
			value := ir.RunEventPayloadValue(fact, declaration.payloadArm)
			if value == nil {
				continue
			}
			s.evidenceMu.Lock()
			ordinal := s.runEventOrdinals[declaration.source]
			s.runEventOrdinals[declaration.source]++
			s.evidenceMu.Unlock()
			evidence, err := a.liftEvidence(w, declaration.lift(program.correlatedObservationID, declaration.guard), value, ordinal)
			if err != nil {
				return err
			}
			fact.Observations = append(fact.Observations, &testpilotspb.ObservationResult{ObservationId: program.correlatedObservationID, Value: evidence})
		}
	}
	return nil
}
