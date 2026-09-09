package execution

import (
	"context"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func (a *activationValues) stage(ctx context.Context, c Coordinate, result EffectResult, limit int64) (*valueBatch, int64, error) {
	n, err := a.instruction(c)
	if err != nil {
		return nil, 0, err
	}
	w, err := a.newWork(ctx, limit)
	if err != nil {
		return nil, 0, err
	}
	batch := &valueBatch{owner: a, coordinate: c, writes: map[string]*testpilotspb.Value{}, fields: map[testpilotspb.InstructionOutcomeField]*testpilotspb.Value{}}
	snapshot, err := validateOutcome(w, a.graph.context, n, result.Outcome)
	if err != nil {
		return nil, w.work, err
	}
	batch.outcome, batch.fields = snapshot.Outcome, snapshot.Fields
	if n.opcode != InvokeRPC {
		if !isNil(result.Response) {
			return nil, w.work, invalid(ir.Unsupported, "projection", "only RPCs return raw responses")
		}
		return finishBatch(w, batch)
	}
	if isNil(result.Response) {
		if batch.outcome.Status == testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED {
			return nil, w.work, invalid(ir.Unavailable, "projection", "successful RPC has no response")
		}
		return finishBatch(w, batch)
	}
	response, work, err := ir.SnapshotMessage(ctx, result.Response, n.method.Output(), w.remaining(n.source.Limits.MaxResponseBytes))
	w.work += work
	if err != nil {
		return nil, w.work, err
	}
	if batch.outcome.Status == testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED {
		for i, p := range n.projections {
			value, work, err := p.path.Read(ctx, response, w.remaining(n.source.Limits.MaxResponseBytes))
			w.work += work
			if err != nil {
				return nil, w.work, err
			}
			if value == nil {
				continue
			}
			if err = a.stageProjection(w, n, batch, p, int64(i), value); err != nil {
				return nil, w.work, err
			}
		}
	}
	return finishBatch(w, batch)
}
func validateOutcome(w *valueWork, entryContext testpilotspb.EntrypointKind, n *node, outcome *testpilotspb.InstructionOutcome) (*OutcomeSnapshot, error) {
	if outcome == nil || outcome.Status == testpilotspb.INSTRUCTION_OUTCOME_STATUS_UNSPECIFIED {
		return nil, invalid(ir.Malformed, "outcome", "typed outcome status required")
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
	if entryContext == testpilotspb.ENTRYPOINT_KIND_CONTROLLER {
		if frozen.SdkFailureCode != "" || frozen.Status == testpilotspb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE {
			return nil, invalid(ir.TypeMismatch, "outcome", "SDK outcome in controller")
		}
	} else if frozen.ProtocolCode != "" || frozen.Status == testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS {
		return nil, invalid(ir.TypeMismatch, "outcome", "protocol outcome in worker")
	}
	valueType, hasValue := n.outcomes[testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE]
	if frozen.Value != nil && !hasValue {
		return nil, invalid(ir.Unsupported, "outcome", "undeclared payload")
	}
	if hasValue && frozen.Status == testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED && frozen.Value == nil {
		return nil, invalid(ir.Unavailable, "outcome", "successful outcome lacks required value")
	}
	if frozen.Value != nil {
		frozen.Value, err = w.copy(frozen.Value, valueType)
		if err != nil {
			return nil, err
		}
	}
	result := &OutcomeSnapshot{Outcome: frozen, Fields: make(map[testpilotspb.InstructionOutcomeField]*testpilotspb.Value, len(n.outcomes))}
	for _, declaration := range n.source.Outcome.Fields {
		field := declaration.Field
		typ := n.outcomes[field]
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
func textValue(text string) *testpilotspb.Value {
	return &testpilotspb.Value{Value: &testpilotspb.Value_Text{Text: text}}
}
func (a *activationValues) stageProjection(w *valueWork, n *node, batch *valueBatch, p projection, index int64, value *testpilotspb.Value) error {
	values := []*testpilotspb.Value{value}
	typ := p.path.Type()
	if p.cardinality == testpilotspb.PROJECTION_KIND_EMIT_EACH {
		values = value.GetListValue().GetValues()
		typ = typ.Element()
	}
	// A source ordinal is the position of the evidence in this lift's own dense stream, so it counts
	// the values already emitted from this projected list rather than reading anything out of them.
	emitted := make([]int64, len(p.sinks))
	for i, value := range values {
		if err := w.charge(1); err != nil {
			return err
		}
		fact := projectionFact{projection: index, index: int64(i)}
		for j, sink := range p.sinks {
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
			switch target := sink.Target.(type) {
			case *testpilotspb.ProjectionTarget_SlotId:
				if _, exists := batch.writes[target.SlotId]; exists {
					return invalid(ir.Malformed, "projection", "duplicate staged Slot")
				}
				batch.writes[target.SlotId] = copied
			case *testpilotspb.ProjectionTarget_ObservationId:
				fact.observations = append(fact.observations, &testpilotspb.ObservationResult{ObservationId: target.ObservationId, Value: copied})
			default:
				return invalid(ir.Unsupported, "projection", "unknown sink")
			}
		}
		if len(fact.observations) > 0 {
			if int64(len(batch.facts)) >= n.source.Limits.MaxEmittedEvents {
				return invalid(ir.LimitExceeded, "projection", "emitted event ceiling exceeded")
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

func outcomeField(outcome *testpilotspb.InstructionOutcome, field testpilotspb.InstructionOutcomeField) (*testpilotspb.Value, error) {
	var value *testpilotspb.Value
	switch field {
	case testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS:
		value = &testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Number: int32(outcome.Status)}}}
	case testpilotspb.INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE:
		value = textValue(outcome.ProtocolCode)
	case testpilotspb.INSTRUCTION_OUTCOME_FIELD_SDK_FAILURE_CODE:
		value = textValue(outcome.SdkFailureCode)
	case testpilotspb.INSTRUCTION_OUTCOME_FIELD_DETAIL:
		value = textValue(outcome.Detail)
	case testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE:
		value = outcome.Value
	default:
		return nil, invalid(ir.Unknown, "outcome", "unknown field")
	}
	return value, nil
}

func (p projection) liftAt(index int) *evidenceLift {
	if index >= len(p.lifts) {
		return nil
	}
	return p.lifts[index]
}

// liftEvidence builds the declared ScopedEvidence value from one projected value. The first rule
// whose guard resolves owns the value; a value no rule claims emits nothing, and a rule that fired
// but cannot read one of its own declared coordinates fails rather than recording partial evidence.
func (a *activationValues) liftEvidence(w *valueWork, lift *evidenceLift, value *testpilotspb.Value, ordinal int64) (*testpilotspb.Value, error) {
	for _, rule := range lift.rules {
		guard, err := a.readLift(w, lift, rule.guard, value)
		if err != nil {
			return nil, err
		}
		if guard == nil {
			continue
		}
		if rule.guardEquals != "" {
			text, ok := guard.Value.(*testpilotspb.Value_Text)
			if !ok || text.Text != rule.guardEquals {
				continue
			}
		}
		evidence := &testpilotspb.ScopedEvidence{Kind: rule.kind, Identity: &testpilotspb.ScopedIdentity{Source: rule.source, Ordinal: ordinal}}
		for _, binding := range rule.scope {
			text := binding.literal
			if binding.path != nil {
				if text, err = a.readLiftText(w, lift, binding.path, value); err != nil {
					return nil, err
				}
			}
			evidence.Identity.Scope = append(evidence.Identity.Scope, &testpilotspb.ScopedBinding{FieldId: binding.fieldID, Value: text})
		}
		if evidence.Operation, err = a.readLiftKey(w, lift, rule.operation, value); err != nil {
			return nil, err
		}
		for _, binding := range rule.fields {
			scalar := textValue(binding.literal)
			if binding.path != nil {
				if scalar, err = a.readLiftScalar(w, lift, binding.path, value); err != nil {
					return nil, err
				}
			}
			evidence.Fields = append(evidence.Fields, &testpilotspb.ScopedEvidenceField{FieldId: binding.fieldID, Value: scalar})
		}
		encoded, err := proto.Marshal(evidence)
		if err != nil {
			return nil, invalid(ir.Malformed, "projection", "evidence lift produced an unencodable value")
		}
		if err := w.charge(int64(len(encoded)) + 1); err != nil {
			return nil, err
		}
		return &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: &anypb.Any{
			TypeUrl: "type.googleapis.com/" + string(evidence.ProtoReflect().Descriptor().FullName()), Value: encoded}}}, nil
	}
	return nil, nil
}
func (a *activationValues) readLift(w *valueWork, lift *evidenceLift, path *ir.Path, value *testpilotspb.Value) (*testpilotspb.Value, error) {
	read, work, err := ir.ReadValue(w.ctx, value, lift.element, path, w.remaining(w.limits.Bytes))
	w.work += work
	return read, err
}
func (a *activationValues) readLiftScalar(w *valueWork, lift *evidenceLift, path *ir.Path, value *testpilotspb.Value) (*testpilotspb.Value, error) {
	read, err := a.readLift(w, lift, path, value)
	if err != nil {
		return nil, err
	}
	if read == nil {
		return nil, invalid(ir.Unavailable, "projection", "evidence lift read an absent declared coordinate")
	}
	// The portable evidence domain is text, natural and boolean; every admitted integer kind
	// narrows into a natural and a negative one has no evidence scalar to narrow to.
	switch item := read.Value.(type) {
	case *testpilotspb.Value_Text, *testpilotspb.Value_BoolValue, *testpilotspb.Value_Natural:
		return read, nil
	case *testpilotspb.Value_SignedInteger:
		if strings.HasPrefix(item.SignedInteger, "-") {
			return nil, invalid(ir.TypeMismatch, "projection", "evidence lift read a negative integer")
		}
		return &testpilotspb.Value{Value: &testpilotspb.Value_Natural{Natural: item.SignedInteger}}, nil
	case *testpilotspb.Value_UnsignedInteger:
		return &testpilotspb.Value{Value: &testpilotspb.Value_Natural{Natural: item.UnsignedInteger}}, nil
	default:
		return nil, invalid(ir.TypeMismatch, "projection", "evidence lift read an unsupported scalar")
	}
}
func (a *activationValues) readLiftText(w *valueWork, lift *evidenceLift, path *ir.Path, value *testpilotspb.Value) (string, error) {
	read, err := a.readLiftScalar(w, lift, path, value)
	if err != nil {
		return "", err
	}
	item, ok := read.Value.(*testpilotspb.Value_Text)
	if !ok {
		return "", invalid(ir.TypeMismatch, "projection", "evidence lift expected text")
	}
	return item.Text, nil
}
func (a *activationValues) readLiftKey(w *valueWork, lift *evidenceLift, path *ir.Path, value *testpilotspb.Value) (string, error) {
	read, err := a.readLiftScalar(w, lift, path, value)
	if err != nil {
		return "", err
	}
	switch item := read.Value.(type) {
	case *testpilotspb.Value_Text:
		return item.Text, nil
	case *testpilotspb.Value_Natural:
		return item.Natural, nil
	default:
		return "", invalid(ir.TypeMismatch, "projection", "evidence lift expected an operation key")
	}
}
