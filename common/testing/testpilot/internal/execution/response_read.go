package execution

import (
	"context"
	"strconv"

	celpb "cel.dev/expr"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func (a *activationValues) stage(ctx context.Context, c contract.Coordinate, result contract.EffectResult, limit int64) (*valueBatch, int64, error) {
	n, err := a.instruction(c)
	if err != nil {
		return nil, 0, err
	}
	w, err := a.newWork(ctx, limit)
	if err != nil {
		return nil, 0, err
	}
	batch := &valueBatch{owner: a, coordinate: c, writes: map[string]*celpb.Value{}, fields: map[testpilotspb.InstructionOutcomeField]*celpb.Value{}}
	snapshot, err := validateOutcome(w, a.graph.context, n, result.Outcome)
	if err != nil {
		return nil, w.work, err
	}
	batch.outcome, batch.fields = snapshot.Outcome, snapshot.Fields
	if n.opcode != contract.InvokeRPC && n.opcode != contract.ReadEvidence {
		if !ir.IsNil(result.Response) {
			return nil, w.work, ir.Invalid(ir.Unsupported, "response_read", "only RPCs return raw responses")
		}
		return finishBatch(w, batch)
	}
	if ir.IsNil(result.Response) {
		if batch.outcome.Status == testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED {
			return nil, w.work, ir.Invalid(ir.Unavailable, "response_read", "successful RPC has no response")
		}
		return finishBatch(w, batch)
	}
	response, work, err := ir.SnapshotMessage(ctx, result.Response, n.method.Output(), w.remaining(a.store.program.limits.MaxInstructionResponseBytes))
	w.work += work
	if err != nil {
		return nil, w.work, err
	}
	if batch.outcome.Status == testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED {
		for i, p := range n.responseReads {
			value, work, err := p.path.Read(ctx, response, w.remaining(a.store.program.limits.MaxInstructionResponseBytes))
			w.work += work
			if err != nil {
				return nil, w.work, err
			}
			if value == nil {
				continue
			}
			if err = a.stageResponseRead(w, n, batch, p, int64(i), value); err != nil {
				return nil, w.work, err
			}
		}
	}
	return finishBatch(w, batch)
}
func validateOutcome(w *valueWork, entryContext contract.EntrypointKind, n *node, outcome *testpilotspb.InstructionOutcome) (*contract.OutcomeSnapshot, error) {
	if outcome == nil || outcome.Status < testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED || outcome.Status > testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED {
		return nil, ir.Invalid(ir.Malformed, "outcome", "typed outcome status required")
	}
	if len(outcome.ProtoReflect().GetUnknown()) != 0 {
		return nil, ir.Invalid(ir.Malformed, "outcome", "unknown outcome fields")
	}
	snapshot, work, err := ir.SnapshotMessage(w.ctx, outcome, outcome.ProtoReflect().Descriptor(), w.remaining(w.limits.Bytes))
	w.work += work
	if err != nil {
		return nil, err
	}
	if err = w.charge(int64(proto.Size(snapshot)) + 1); err != nil {
		return nil, err
	}
	frozen := &testpilotspb.InstructionOutcome{}
	proto.Merge(frozen, snapshot)
	if entryContext == contract.ControllerEntrypoint {
		if frozen.SdkFailureCode != "" || frozen.Status == testpilotspb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE {
			return nil, ir.Invalid(ir.TypeMismatch, "outcome", "SDK outcome in controller")
		}
	} else if frozen.ProtocolCode != "" || frozen.Status == testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE {
		return nil, ir.Invalid(ir.TypeMismatch, "outcome", "protocol outcome in worker")
	}
	valueType, hasValue := n.outcomes[testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE]
	if frozen.Value != nil && !hasValue {
		return nil, ir.Invalid(ir.Unsupported, "outcome", "undeclared payload")
	}
	if hasValue && frozen.Status == testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED && frozen.Value == nil {
		return nil, ir.Invalid(ir.Unavailable, "outcome", "successful outcome lacks required value")
	}
	if frozen.Value != nil {
		frozen.Value, err = w.copy(frozen.Value, valueType)
		if err != nil {
			return nil, err
		}
	}
	if err := validateDeliveryAdmission(n, frozen); err != nil {
		return nil, err
	}
	result := &contract.OutcomeSnapshot{Outcome: frozen, Fields: make(map[testpilotspb.InstructionOutcomeField]*celpb.Value, len(n.outcomes))}
	for _, field := range outcomeFieldOrder {
		typ, produced := n.outcomes[field]
		if !produced {
			continue
		}
		value, err := outcomeField(frozen, field)
		if err != nil {
			return nil, err
		}
		if value != nil {
			result.Fields[field], err = w.copy(value, typ)
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

// validateDeliveryAdmission admits a delivery admission on the successful outcome of a delivery
// release alone, and requires it there: the release succeeds only once the decision is observed.
func validateDeliveryAdmission(n *node, outcome *testpilotspb.InstructionOutcome) error {
	kind := n.source.GetInstruction().GetInjectFault().GetKind()
	lost := kind == testpilotspb.FAULT_KIND_ADMISSION_RESPONSE_LOSS
	release := n.opcode == contract.InjectFault && (kind == testpilotspb.FAULT_KIND_DELIVERY_RELEASE || lost)
	succeeded := outcome.GetStatus() == testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED
	admission := outcome.GetDeliveryAdmission()
	if admission == nil && (!release || !succeeded) {
		return nil
	}
	if !release || !succeeded {
		return ir.Invalid(ir.Malformed, "outcome", "a delivery admission is the outcome of a successful delivery release alone")
	}
	decision := admission.GetDecision()
	if lost && (decision != testpilotspb.DELIVERY_ADMISSION_DECISION_ADMITTED || admission.GetAttempt() <= 0 || admission.GetActivityRunId() == "") {
		return ir.Invalid(ir.Malformed, "outcome", "a lost admission response requires its committed attempt and activity run")
	}
	if admission.GetActivityId() == "" || admission.GetDeliveryId() == "" ||
		decision != testpilotspb.DELIVERY_ADMISSION_DECISION_ADMITTED && decision != testpilotspb.DELIVERY_ADMISSION_DECISION_REJECTED {
		return ir.Invalid(ir.Malformed, "outcome", "a successful delivery release requires the admission decision it observed")
	}
	return nil
}

func textValue(text string) *celpb.Value {
	return &celpb.Value{Kind: &celpb.Value_StringValue{StringValue: text}}
}
func (a *activationValues) stageResponseRead(w *valueWork, n *node, batch *valueBatch, p responseRead, index int64, value *celpb.Value) error {
	values := []*celpb.Value{value}
	typ := p.path.Type()
	if p.emitEach {
		values = value.GetListValue().GetValues()
		typ = typ.Element()
	}
	// A source ordinal is the position of the evidence in this lift's own dense stream, so it counts
	// the values already emitted from this projected list rather than reading anything out of them.
	emitted := make([]int64, len(p.targets))
	for i, value := range values {
		if err := w.charge(1); err != nil {
			return err
		}
		fact := readFact{read: index, index: int64(i)}
		for j, readTarget := range p.targets {
			if lift := p.liftAt(j); lift != nil {
				evidence, err := a.liftEvidence(w, lift, value, emitted[j])
				if err != nil {
					return err
				}
				if evidence != nil {
					emitted[j]++
					fact.observations = append(fact.observations, &testpilotspb.ObservationResult{ObservationId: lift.observationID, Value: evidence})
				}
				continue
			}
			copied, err := w.copy(value, typ)
			if err != nil {
				return err
			}
			switch target := readTarget.Target.(type) {
			case *testpilotspb.ReadTarget_SlotId:
				if _, exists := batch.writes[target.SlotId]; exists {
					return ir.Invalid(ir.Malformed, "response_read", "duplicate staged Slot")
				}
				batch.writes[target.SlotId] = copied
			case *testpilotspb.ReadTarget_ObservationId:
				fact.observations = append(fact.observations, &testpilotspb.ObservationResult{ObservationId: target.ObservationId, Value: copied})
			default:
				return ir.Invalid(ir.Unsupported, "response_read", "unknown response read target")
			}
		}
		if len(fact.observations) > 0 {
			if int64(len(batch.facts)) >= a.store.program.limits.MaxInstructionEmittedEvents {
				return ir.Invalid(ir.LimitExceeded, "response_read", "emitted event ceiling exceeded")
			}
			batch.facts = append(batch.facts, fact)
		}
	}
	return nil
}

func finishBatch(w *valueWork, batch *valueBatch) (*valueBatch, int64, error) {
	if err := w.charge(1); err != nil {
		return nil, w.work, err
	}
	return batch, w.work, nil
}

// outcomeFieldOrder is the order an outcome snapshot copies the fields a node produces, so its work
// charge is deterministic.
var outcomeFieldOrder = []testpilotspb.InstructionOutcomeField{
	testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS,
	testpilotspb.INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE,
	testpilotspb.INSTRUCTION_OUTCOME_FIELD_SDK_FAILURE_CODE,
	testpilotspb.INSTRUCTION_OUTCOME_FIELD_DETAIL,
	testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE,
}

func outcomeField(outcome *testpilotspb.InstructionOutcome, field testpilotspb.InstructionOutcomeField) (*celpb.Value, error) {
	var value *celpb.Value
	switch field {
	case testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS:
		value = ir.EnumValue(outcome.Status.Descriptor(), outcome.Status.Number())
	case testpilotspb.INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE:
		value = textValue(outcome.ProtocolCode)
	case testpilotspb.INSTRUCTION_OUTCOME_FIELD_SDK_FAILURE_CODE:
		value = textValue(outcome.SdkFailureCode)
	case testpilotspb.INSTRUCTION_OUTCOME_FIELD_DETAIL:
		value = textValue(outcome.Detail)
	case testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE:
		value = outcome.Value
	default:
		return nil, ir.Invalid(ir.Unknown, "outcome", "unknown field")
	}
	return value, nil
}

func (p responseRead) liftAt(index int) *evidenceLift {
	if index >= len(p.lifts) {
		return nil
	}
	return p.lifts[index]
}

// liftEvidence builds the declared CorrelatedEvidence value from one projected value. The first rule
// whose guard is true owns the value; a value no rule claims emits nothing, and a rule that fired
// but cannot read one of its own declared coordinates fails rather than recording partial evidence.
func (a *activationValues) liftEvidence(w *valueWork, lift *evidenceLift, value *celpb.Value, ordinal int64) (*celpb.Value, error) {
	for _, rule := range lift.rules {
		selected, err := selectsEvidence(w, rule.guard, value)
		if err != nil {
			return nil, err
		}
		if !selected {
			continue
		}
		return a.buildEvidence(w, lift, rule, value, ordinal)
	}
	return nil, nil
}

// selectsEvidence evaluates a lift guard over one projected value. A guard that has no value is an
// error, never a rejection.
func selectsEvidence(w *valueWork, guard *ir.Expression, value *celpb.Value) (bool, error) {
	selected, work, err := guard.EvaluateExecution(w.ctx, func(reference ir.Reference) *celpb.Value {
		if reference.Kind == ir.ProjectedValueReference {
			return value
		}
		return nil
	}, w.limits.Work-w.work)
	w.work += work
	if err != nil {
		return false, err
	}
	return selected.GetBoolValue(), nil
}

// buildEvidence builds the evidence the rule declares from the projected value its guard selected,
// under the ordinal.
func (a *activationValues) buildEvidence(w *valueWork, lift *evidenceLift, rule evidenceRule, value *celpb.Value, ordinal int64) (*celpb.Value, error) {
	var err error
	evidence := &testpilotspb.CorrelatedEvidence{Kind: rule.kind, Identity: &testpilotspb.CorrelatedIdentity{EvidenceSource: rule.source, Ordinal: ordinal}}
	for _, binding := range rule.scope {
		text, err := a.readLiftText(w, lift, binding.value, value)
		if err != nil {
			return nil, err
		}
		if text == "" {
			return nil, ir.Invalid(ir.Unavailable, "response_read", "evidence scope read an empty declared coordinate")
		}
		evidence.Identity.Scope = append(evidence.Identity.Scope, &testpilotspb.NamedValue{FieldId: binding.fieldID, Value: textValue(text)})
	}
	if rule.operation == nil {
		evidence.Operation = a.store.runID
	} else if evidence.Operation, err = a.readLiftKey(w, lift, rule.operation, value); err != nil {
		return nil, err
	}
	for _, binding := range rule.fields {
		scalar, err := a.readLiftScalar(w, lift, binding.value, value)
		if err != nil {
			return nil, err
		}
		evidence.Fields = append(evidence.Fields, &testpilotspb.NamedValue{FieldId: binding.fieldID, Value: scalar})
	}
	a.store.chainEvidence(evidence)
	encoded, err := proto.Marshal(evidence)
	if err != nil {
		return nil, ir.Invalid(ir.Malformed, "response_read", "evidence lift produced an unencodable value")
	}
	if err := w.charge(int64(len(encoded)) + 1); err != nil {
		return nil, err
	}
	return &celpb.Value{Kind: &celpb.Value_ObjectValue{ObjectValue: &anypb.Any{TypeUrl: "type.googleapis.com/" + string(evidence.ProtoReflect().Descriptor().FullName()), Value: encoded}}}, nil
}
func (a *activationValues) readLift(w *valueWork, lift *evidenceLift, expression *ir.Expression, value *celpb.Value) (*celpb.Value, error) {
	read, work, err := expression.EvaluateExecution(w.ctx, func(reference ir.Reference) *celpb.Value {
		if reference.Kind == ir.ProjectedValueReference {
			return value
		}
		return nil
	}, w.limits.Work-w.work)
	w.work += work
	return read, err
}
func (a *activationValues) readLiftScalar(w *valueWork, lift *evidenceLift, expression *ir.Expression, value *celpb.Value) (*celpb.Value, error) {
	read, err := a.readLift(w, lift, expression, value)
	if err != nil {
		return nil, err
	}
	if read == nil {
		return nil, ir.Invalid(ir.Unavailable, "response_read", "evidence lift read an absent declared coordinate")
	}
	// The portable evidence domain is text, unsigned integer and boolean; every admitted integer kind
	// narrows into an unsigned integer and a negative one has no evidence scalar to narrow to.
	switch item := read.Kind.(type) {
	case *celpb.Value_StringValue, *celpb.Value_BoolValue, *celpb.Value_Uint64Value:
		return read, nil
	case *celpb.Value_Int64Value:
		if item.Int64Value < 0 {
			return nil, ir.Invalid(ir.TypeMismatch, "response_read", "evidence lift read a negative integer")
		}
		return &celpb.Value{Kind: &celpb.Value_Uint64Value{Uint64Value: uint64(item.Int64Value)}}, nil
	default:
		return nil, ir.Invalid(ir.TypeMismatch, "response_read", "evidence lift read an unsupported scalar")
	}
}
func (a *activationValues) readLiftText(w *valueWork, lift *evidenceLift, expression *ir.Expression, value *celpb.Value) (string, error) {
	read, err := a.readLiftScalar(w, lift, expression, value)
	if err != nil {
		return "", err
	}
	item, ok := read.Kind.(*celpb.Value_StringValue)
	if !ok {
		return "", ir.Invalid(ir.TypeMismatch, "response_read", "evidence lift expected text")
	}
	return item.StringValue, nil
}
func (a *activationValues) readLiftKey(w *valueWork, lift *evidenceLift, expression *ir.Expression, value *celpb.Value) (string, error) {
	read, err := a.readLiftScalar(w, lift, expression, value)
	if err != nil {
		return "", err
	}
	switch item := read.Kind.(type) {
	case *celpb.Value_StringValue:
		if item.StringValue == "" {
			return "", ir.Invalid(ir.Unavailable, "response_read", "evidence lift read an empty operation key")
		}
		return item.StringValue, nil
	case *celpb.Value_Uint64Value:
		return strconv.FormatUint(item.Uint64Value, 10), nil
	default:
		return "", ir.Invalid(ir.TypeMismatch, "response_read", "evidence lift expected an operation key")
	}
}
