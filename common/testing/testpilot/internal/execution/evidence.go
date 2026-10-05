package execution

import (
	"context"
	"errors"
	"fmt"
	"slices"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// bindEvidence admits the Program's evidence declarations: each identity once, each recorded kind
// once under a source and key path, every path typed against the recorded value its source
// supplies. A declaration a Run Event feeds is lifted by the scheduler as it records the event; a
// read declaration is polled by a ReadEvidence instruction; a history declaration is named by a
// history read's lift rule. Kinds that share a source are numbered by its one emitter, the Run for
// its own events or the instruction that lifts the others, so no source holds both.
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
	a.historyEvent, a.historyRead = historyRead(a.prepared.catalog, p, a.expressionLimits())
	sources := map[string][]*evidenceDeclaration{}
	for index, source := range p.Evidence {
		location := fmt.Sprintf("program.evidence[%d]", index)
		if err := a.charge(1); err != nil {
			return err
		}
		if source == nil || !ir.ValidID(source.GetEvidenceId()) || !ir.ValidID(source.GetEvidenceSource()) || source.GetOperation() == "" && !source.GetRunEvent().GetRunKeyed() {
			return ir.Invalid(ir.Malformed, location, "evidence declaration requires an identity, a source and an operation key path")
		}
		if _, exists := a.prepared.evidence[source.EvidenceId]; exists {
			return ir.Invalid(ir.Malformed, location+".evidence_id", "duplicate evidence declaration")
		}
		bound, err := a.bindEvidenceDeclaration(location, source)
		if err != nil {
			return err
		}
		for _, other := range sources[bound.source] {
			if err := a.charge(int64(proto.Size(bound.guardSource)) + 1); err != nil {
				return err
			}
			if (bound.kind == RunEventSource) != (other.kind == RunEventSource) {
				return ir.Invalid(ir.Malformed, location, "evidence source is counted by the Run and by an instruction")
			}
			if bound.keyPath() == other.keyPath() && bound.sameRecord(other) {
				return ir.Invalid(ir.Malformed, location, "evidence source and operation key path are declared twice")
			}
		}
		sources[bound.source] = append(sources[bound.source], bound)
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
		return ir.Invalid(ir.TypeMismatch, "program.evidence", "a Run Event declaration requires exactly one declared CorrelatedEvidence Observation")
	}
	return nil
}

func (a *admission) bindEvidenceDeclaration(location string, source *testpilotspb.EvidenceDeclaration) (*evidenceDeclaration, error) {
	bound := &evidenceDeclaration{id: source.EvidenceId, source: source.EvidenceSource}
	switch recorded := source.Source.(type) {
	case *testpilotspb.EvidenceDeclaration_HistoryEvent:
		bound.kind = HistoryEventSource
		// The recorded event is what the Program's history read reads: the Case names no event
		// message of its own, so none is assumed.
		if !a.historyRead {
			return nil, ir.Invalid(ir.Unknown, location+".history_event", "history evidence requires a read that lifts it")
		}
		element := a.historyEvent
		arm := recorded.HistoryEvent.GetAttributesField()
		member := element.Message().Fields().ByName(protoreflect.Name(arm))
		if member == nil || member.ContainingOneof() == nil {
			return nil, ir.Invalid(ir.Unknown, location+".history_event.attributes_field", "history evidence requires an attributes arm of the recorded event")
		}
		bound.element, bound.attributesField = element, arm
		var err error
		bound.guard, err = a.liftGuard(location+".history_event", element, presentExpression(projectedPath(string(member.ContainingOneof().Name())+"<"+arm+">")))
		if err != nil {
			return nil, err
		}
	case *testpilotspb.EvidenceDeclaration_RunEvent:
		bound.kind = RunEventSource
		kind := recorded.RunEvent.GetKind()
		payload := ir.RunEventPayloadOf(kind)
		if kind <= 0 || kind > ir.MaxRunEventKind || payload.Arm == "" {
			return nil, ir.Invalid(ir.Unsupported, location+".run_event.kind", "Run Event evidence requires a kind that carries a payload")
		}
		element, ok := a.prepared.catalog.RunEventPayloadType(payload.Arm)
		if !ok {
			return nil, ir.Invalid(ir.Unknown, location+".run_event.kind", "Run Event payload arm is not in the catalog")
		}
		bound.element, bound.runEventKind, bound.payloadArm = element, kind, payload.Arm
		if bound.instruction = recorded.RunEvent.GetInstruction(); bound.instruction != nil && !recordsRunEvents(a.prepared.source, bound.instruction) {
			return nil, ir.Invalid(ir.Unknown, location+".run_event.instruction", "Run Event evidence names an instruction no controller entrypoint declares")
		}
		var err error
		if bound.guardSource = recorded.RunEvent.GetGuard(); bound.guardSource != nil {
			// The binder locates a reference it rejects, but not a guard of another type or one
			// that may have no value, which are faults of the guard as a whole.
			var unlocated *ir.Error
			if bound.guard, err = a.liftGuard(location+".run_event.guard", element, bound.guardSource); errors.As(err, &unlocated) && unlocated.Path == "expression" {
				err = ir.Invalid(unlocated.Category, location+".run_event.guard", unlocated.Detail)
			}
		} else {
			bound.guard, err = a.liftGuard(location+".run_event", element, &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}}}})
		}
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
		element, requirement := path.Type(), "read evidence requires repeated messages"
		if bound.single = recorded.Read.GetSingle(); bound.single {
			requirement = "a single read requires one message"
			if element.Cardinality() != ir.Singular {
				return nil, ir.Invalid(ir.TypeMismatch, location+".read.path", requirement)
			}
		} else {
			if element.Cardinality() != ir.Repeated {
				return nil, ir.Invalid(ir.TypeMismatch, location+".read.path", "read evidence requires a repeated field")
			}
			element = element.Element()
		}
		if element.Message() == nil || element.Opaque() || element.Any() {
			return nil, ir.Invalid(ir.TypeMismatch, location+".read.path", requirement)
		}
		bound.element, bound.method, bound.readPath = element, method, path
	default:
		return nil, ir.Invalid(ir.Malformed, location+".source", "evidence declaration requires a source")
	}
	var err error
	if source.GetRunEvent().GetRunKeyed() {
		if source.Operation != "" {
			return nil, ir.Invalid(ir.Malformed, location+".operation", "evidence keyed by the Run reads no operation key path")
		}
	} else if bound.operation, err = a.bindEvidencePath(location, bound.element, location+".operation", source.Operation, evidenceKeyKinds...); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for index, scope := range source.Scope {
		text, isText := scope.GetValue().GetValue().(*testpilotspb.Value_TextValue)
		if scope == nil || !ir.ValidID(scope.GetFieldId()) || seen[scope.GetFieldId()] || !isText || text.TextValue == "" {
			return nil, ir.Invalid(ir.Malformed, fmt.Sprintf("%s.scope[%d]", location, index), "evidence scope requires one unique declared field with a text value")
		}
		seen[scope.FieldId] = true
		bound.scope = append(bound.scope, evidenceBinding{fieldID: scope.FieldId, literal: text.TextValue})
	}
	fields := map[string]bool{}
	for index, field := range source.Fields {
		if field == nil || !ir.ValidID(field.GetFieldId()) || fields[field.GetFieldId()] {
			return nil, ir.Invalid(ir.Malformed, fmt.Sprintf("%s.fields[%d]", location, index), "evidence field requires one unique declared field")
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

// recordsRunEvents reports whether the reference names an instruction whose events the Run records
// under the instruction's own coordinates: one of a controller entrypoint.
func recordsRunEvents(p *testpilotspb.Program, reference *testpilotspb.InstructionReference) bool {
	for _, entrypoint := range p.GetEntrypoints() {
		if entrypoint.GetEntrypointId() == reference.GetEntrypointId() && entrypoint.GetController() != nil {
			return slices.ContainsFunc(entrypoint.GetInstructions(), func(n *testpilotspb.InstructionNode) bool {
				return n.GetInstructionId() == reference.GetInstructionId()
			})
		}
	}
	return false
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

// historyRead is the history event of a Program: the message its response reads that lift a
// declaration by name yield, each the element its path reads in its method's response. A read
// preparation refuses is left out here and refused where its instruction is bound, and so is one that
// yields another message than the first (bindDeclaredRule).
func historyRead(catalog *ir.Catalog, p *testpilotspb.Program, limits ir.Limits) (ir.Type, bool) {
	nodes := slices.Clone(p.GetCleanup().GetInstructions())
	for _, entrypoint := range p.GetEntrypoints() {
		nodes = append(nodes, entrypoint.GetInstructions()...)
	}
	for _, n := range nodes {
		rpc := n.GetInstruction().GetInvokeRpc()
		if rpc == nil {
			continue
		}
		method, err := catalog.Method(rpc.GetMethod())
		if err != nil {
			continue
		}
		output, err := messageType(catalog, method.Output())
		if err != nil {
			continue
		}
		for _, read := range rpc.GetResponseReads() {
			names := slices.ContainsFunc(read.GetTargets(), func(t *testpilotspb.ReadTarget) bool {
				return slices.ContainsFunc(t.GetCorrelatedEvidence().GetRules(), func(r *testpilotspb.CorrelatedEvidenceRule) bool { return r.GetEvidenceId() != "" })
			})
			if !names {
				continue
			}
			path, err := catalog.BindPath(output, "", read.GetPath(), limits)
			if err != nil {
				continue
			}
			typ := path.Type()
			if read.GetCardinality() == testpilotspb.READ_CARDINALITY_EMIT_EACH && typ.Cardinality() == ir.Repeated {
				typ = typ.Element()
			}
			if typ.Cardinality() == ir.Singular && typ.Message() != nil {
				return typ, true
			}
		}
	}
	return ir.Type{}, false
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
		return nil, ir.Invalid(ir.Malformed, location, "a rule naming a declaration spells nothing else")
	}
	declaration, exists := a.prepared.evidence[source.GetEvidenceId()]
	if !exists {
		return nil, ir.Invalid(ir.Unknown, location+".evidence_id", "evidence rule names an undeclared evidence kind")
	}
	if declaration.kind != HistoryEventSource {
		return nil, ir.Invalid(ir.Unsupported, location+".evidence_id", "only a history event declaration is lifted by a read")
	}
	if !typ.Equal(declaration.element) {
		return nil, ir.Invalid(ir.TypeMismatch, nodePath(g, n), "declared history evidence requires a history event read")
	}
	rule := declaration.lift("", declaration.guard).rules[0]
	return &rule, nil
}

// bindReadEvidence admits a controller poll of a read declaration: the endpoint role authorizes the
// declaration's method, the poll interval fits the instruction's timeout, or a read once has none,
// and the instruction's one response read lifts every value `until` selects into the Program's
// CorrelatedEvidence Observation, under the declaration's coordinates. The values are the elements
// of the declared repeated field, or the one message of a single read, which emits one event at most.
func (a *admission) bindReadEvidence(g *graph, n *node) error {
	read := n.source.Instruction.GetReadEvidence()
	if read == nil {
		return ir.Invalid(ir.Malformed, nodePath(g, n), "nil ReadEvidence")
	}
	declaration, exists := a.prepared.evidence[read.GetEvidenceId()]
	if !exists {
		return ir.Invalid(ir.Unknown, expressionPath(g, n, "instruction.read_evidence.evidence_id"), "ReadEvidence names an undeclared evidence kind")
	}
	if declaration.kind != ReadSource {
		return ir.Invalid(ir.TypeMismatch, expressionPath(g, n, "instruction.read_evidence.evidence_id"), "ReadEvidence requires a read declaration")
	}
	if err := a.role(read.EndpointRoleId, testpilotspb.ROLE_KIND_ENDPOINT); err != nil {
		return err
	}
	if !a.methods[read.EndpointRoleId][methodName(declaration.method)] {
		return ir.Invalid(ir.Unsupported, nodePath(g, n), "unauthorized RPC method")
	}
	if a.prepared.correlatedObservationID == "" {
		return ir.Invalid(ir.TypeMismatch, nodePath(g, n), "ReadEvidence requires exactly one declared CorrelatedEvidence Observation")
	}
	intervalPath := expressionPath(g, n, "instruction.read_evidence.poll_interval_milliseconds")
	// A hinted poll's interval is declared against its declared bound, not the scaled one it runs
	// under, so a Case admits under every scale or none.
	timeout := n.timeoutMilliseconds
	if len(n.source.GetWaitHints()) > 0 {
		timeout = n.source.GetLimits().GetTimeoutMilliseconds()
	}
	switch {
	case read.Once && read.PollIntervalMilliseconds != 0:
		return ir.Invalid(ir.Malformed, intervalPath, "a read once has no poll interval")
	case read.Once:
	case read.PollIntervalMilliseconds <= 0:
		return ir.Invalid(ir.Malformed, intervalPath, "ReadEvidence requires a positive poll interval")
	case read.PollIntervalMilliseconds > timeout:
		return ir.Invalid(ir.LimitExceeded, intervalPath, "poll interval exceeds the instruction timeout")
	default:
	}
	cardinality := testpilotspb.READ_CARDINALITY_EMIT_EACH
	if declaration.single {
		cardinality = testpilotspb.READ_CARDINALITY_ONE
	} else if a.prepared.limits.MaxPathFanout > a.prepared.limits.MaxInstructionEmittedEvents {
		return ir.Invalid(ir.LimitExceeded, nodePath(g, n), "read evidence emission exceeds instruction bound")
	}
	until, err := a.liftGuard(expressionPath(g, n, "instruction.read_evidence.until"), declaration.element, read.Until)
	if err != nil {
		return err
	}
	owner := contract.Coordinate{EntrypointID: g.id, InstructionID: n.source.InstructionId}
	if claimed, exists := a.evidenceSources[declaration.source]; exists && claimed != owner {
		return ir.Invalid(ir.Malformed, nodePath(g, n), "evidence source is already lifted by another instruction")
	}
	a.evidenceSources[declaration.source] = owner
	lift := declaration.lift(a.prepared.correlatedObservationID, until)
	n.method, n.until, n.pollIntervalMilliseconds, n.once = declaration.method, until, read.PollIntervalMilliseconds, read.Once
	n.responseReads = []responseRead{{
		path: declaration.readPath, cardinality: cardinality,
		targets: []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_CorrelatedEvidence{CorrelatedEvidence: &testpilotspb.CorrelatedEvidenceProjection{ObservationId: lift.observationID}}}},
		lifts:   []*evidenceLift{lift},
	}}
	return nil
}

func methodName(method protoreflect.MethodDescriptor) string {
	return "/" + string(method.Parent().FullName()) + "/" + string(method.Name())
}

// readSatisfied reports whether a value of the poll's declared path satisfies the instruction's
// `until`, which ends a ReadEvidence poll: an element of its repeated field, or its one message.
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
		return false, w.work, ir.Invalid(ir.TypeMismatch, "read_evidence", "poll requires a ReadEvidence instruction")
	}
	if ir.IsNil(response) {
		return false, w.work, ir.Invalid(ir.Unavailable, "read_evidence", "poll returned no response")
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
	elements := value.GetListValue().GetValues()
	if read.cardinality == testpilotspb.READ_CARDINALITY_ONE && value != nil {
		elements = []*testpilotspb.Value{value}
	}
	for _, element := range elements {
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

// liftRunEvents attaches, to each recorded fact a Run Event declaration names by kind and, when it
// names one, by instruction, and selects by its guard, the evidence lifted from the fact's payload. A fact carries the Program's one
// CorrelatedEvidence Observation once, so a fact two declarations select is an error rather than
// evidence of either. Ordinals are dense per source across the Run, in recording order: a fact no
// declaration selects, and one that fails to lift, takes none.
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
		var selected *evidenceDeclaration
		var value *testpilotspb.Value
		for _, declaration := range program.runEventLifts {
			if fact.Kind != declaration.runEventKind {
				continue
			}
			if at := declaration.instruction; at != nil && (fact.GetCoordinates().GetEntrypointId() != at.GetEntrypointId() || fact.GetCoordinates().GetInstructionId() != at.GetInstructionId()) {
				continue
			}
			payload := ir.RunEventPayloadValue(fact, declaration.payloadArm)
			if payload == nil {
				continue
			}
			accepted, err := selectsEvidence(w, declaration.guard, payload)
			if err != nil {
				return err
			}
			if !accepted {
				continue
			}
			if selected != nil {
				return ir.Invalid(ir.Malformed, "run_event", fmt.Sprintf("a Run Event is evidence of both %s and %s", selected.id, declaration.id))
			}
			selected, value = declaration, payload
		}
		if selected == nil {
			continue
		}
		lift := selected.lift(program.correlatedObservationID, selected.guard)
		s.evidenceMu.Lock()
		evidence, err := a.buildEvidence(w, lift, lift.rules[0], value, s.runEventOrdinals[selected.source])
		if err == nil {
			s.runEventOrdinals[selected.source]++
		}
		s.evidenceMu.Unlock()
		if err != nil {
			return err
		}
		fact.Observations = append(fact.Observations, &testpilotspb.ObservationResult{ObservationId: program.correlatedObservationID, Value: evidence})
	}
	return nil
}
