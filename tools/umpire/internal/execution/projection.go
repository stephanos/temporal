package execution

import (
	"context"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/ir"
	"google.golang.org/protobuf/proto"
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
	batch := &valueBatch{owner: a, coordinate: c, writes: map[string]*umpirespb.Value{}, fields: map[umpirespb.InstructionOutcomeField]*umpirespb.Value{}}
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
		if batch.outcome.Status == umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED {
			return nil, w.work, invalid(ir.Unavailable, "projection", "successful RPC has no response")
		}
		return finishBatch(w, batch)
	}
	response, work, err := ir.SnapshotMessage(ctx, result.Response, n.method.Output(), w.remaining(n.source.Bounds.MaxResponseBytes))
	w.work += work
	if err != nil {
		return nil, w.work, err
	}
	if batch.outcome.Status == umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED {
		for i, p := range n.projections {
			value, work, err := p.path.Read(ctx, response, w.remaining(n.source.Bounds.MaxResponseBytes))
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
func validateOutcome(w *valueWork, entryContext umpirespb.EntrypointContext, n *node, outcome *umpirespb.InstructionOutcome) (*OutcomeSnapshot, error) {
	if outcome == nil || outcome.Status == umpirespb.INSTRUCTION_OUTCOME_STATUS_UNSPECIFIED {
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
	frozen := &umpirespb.InstructionOutcome{}
	proto.Merge(frozen, snapshot)
	if entryContext == umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER {
		if frozen.SdkFailureCode != "" || frozen.Status == umpirespb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE {
			return nil, invalid(ir.TypeMismatch, "outcome", "SDK outcome in controller")
		}
	} else if frozen.ProtocolCode != "" || frozen.Status == umpirespb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS {
		return nil, invalid(ir.TypeMismatch, "outcome", "protocol outcome in worker")
	}
	valueType, hasValue := n.outcomes[umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE]
	if frozen.Value != nil && !hasValue {
		return nil, invalid(ir.Unsupported, "outcome", "undeclared payload")
	}
	if hasValue && frozen.Status == umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED && frozen.Value == nil {
		return nil, invalid(ir.Unavailable, "outcome", "successful outcome lacks required value")
	}
	if frozen.Value != nil {
		frozen.Value, err = w.copy(frozen.Value, valueType)
		if err != nil {
			return nil, err
		}
	}
	result := &OutcomeSnapshot{Outcome: frozen, Fields: make(map[umpirespb.InstructionOutcomeField]*umpirespb.Value, len(n.outcomes))}
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
func textValue(text string) *umpirespb.Value {
	return &umpirespb.Value{Value: &umpirespb.Value_Text{Text: text}}
}
func (a *activationValues) stageProjection(w *valueWork, n *node, batch *valueBatch, p projection, index int64, value *umpirespb.Value) error {
	values := []*umpirespb.Value{value}
	typ := p.path.Type()
	if p.cardinality == umpirespb.PROJECTION_CARDINALITY_EMIT_EACH {
		values = value.GetListValue().GetValues()
		typ = typ.Element()
	}
	for i, value := range values {
		if err := w.charge(1); err != nil {
			return err
		}
		fact := projectionFact{projection: index, index: int64(i)}
		for _, sink := range p.sinks {
			copied, err := w.copy(value, typ)
			if err != nil {
				return err
			}
			switch target := sink.Sink.(type) {
			case *umpirespb.ProjectionSink_SlotId:
				if _, exists := batch.writes[target.SlotId]; exists {
					return invalid(ir.Malformed, "projection", "duplicate staged Slot")
				}
				batch.writes[target.SlotId] = copied
			case *umpirespb.ProjectionSink_ObservationId:
				fact.observations = append(fact.observations, &umpirespb.ObservationValue{ObservationId: target.ObservationId, Value: copied})
			default:
				return invalid(ir.Unsupported, "projection", "unknown sink")
			}
		}
		if len(fact.observations) > 0 {
			if int64(len(batch.facts)) >= n.source.Bounds.MaxEmittedEvents {
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

func outcomeField(outcome *umpirespb.InstructionOutcome, field umpirespb.InstructionOutcomeField) (*umpirespb.Value, error) {
	var value *umpirespb.Value
	switch field {
	case umpirespb.INSTRUCTION_OUTCOME_FIELD_STATUS:
		value = &umpirespb.Value{Value: &umpirespb.Value_EnumValue{EnumValue: &umpirespb.EnumValue{Number: int32(outcome.Status)}}}
	case umpirespb.INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE:
		value = textValue(outcome.ProtocolCode)
	case umpirespb.INSTRUCTION_OUTCOME_FIELD_SDK_FAILURE_CODE:
		value = textValue(outcome.SdkFailureCode)
	case umpirespb.INSTRUCTION_OUTCOME_FIELD_DETAIL:
		value = textValue(outcome.Detail)
	case umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE:
		value = outcome.Value
	default:
		return nil, invalid(ir.Unknown, "outcome", "unknown field")
	}
	return value, nil
}
