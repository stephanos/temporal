package verification

import (
	"context"
	"fmt"
	"strconv"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/execution"
	"go.temporal.io/server/tools/umpire/internal/ir"
	"google.golang.org/protobuf/proto"
)

// Evaluator owns one Run's state. Callbacks are synchronous and must not overlap.
// PreparedContract can create independent Evaluators concurrently.
type Evaluator struct {
	result                                                   *umpirespb.Verdict
	satisfied                                                int
	prepared                                                 *PreparedContract
	rules                                                    []ruleState
	trace                                                    []transitionTrace
	sequence, elapsed, totalWork, captureCount, captureBytes int64
	incomplete, violated, closed, sawClosure                 bool
	failure                                                  error
	failureSequence                                          int64
}
type capturedValue struct {
	value    *umpirespb.Value
	sequence int64
}
type ruleState struct {
	state    int
	captures map[string]capturedValue
	support  []int64
}
type transitionTrace struct {
	Sequence                   int64
	Rule, Transition, From, To string
}
type ruleChange struct {
	rule, state int
	captures    map[string]capturedValue
	support     bool
	trace       transitionTrace
}

// New implements the private execution factory contract over the exact admitted Program view.
func (p *PreparedContract) New(ctx context.Context, view execution.ProgramView) (execution.Monitor, error) {
	return p.newEvaluator(ctx, view)
}
func (p *PreparedContract) newEvaluator(ctx context.Context, view execution.ProgramView) (*Evaluator, error) {
	if p == nil || ctx == nil || view.ProgramID() != p.program.ProgramID() || view.CatalogIdentity() != p.program.CatalogIdentity() || !proto.Equal(view.Limits(), p.program.Limits()) {
		return nil, invalid(ir.Malformed, "matching prepared Program view and context required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	observations := view.Observations()
	if len(observations) != len(p.observations) {
		return nil, invalid(ir.TypeMismatch, "Program observations differ")
	}
	for _, o := range observations {
		if !o.Type.Equal(p.observations[o.ID]) {
			return nil, invalid(ir.TypeMismatch, "Program observations differ")
		}
	}
	e := &Evaluator{prepared: p, rules: make([]ruleState, len(p.rules)), result: &umpirespb.Verdict{Rules: make([]*umpirespb.RuleVerdict, len(p.rules))}}
	for i, m := range p.rules {
		e.rules[i] = ruleState{state: m.initial, captures: map[string]capturedValue{}}
		e.result.Rules[i] = &umpirespb.RuleVerdict{RuleId: m.source.RuleId, Kind: umpirespb.RULE_VERDICT_KIND_INCONCLUSIVE}
	}
	return e, nil
}
func (e *Evaluator) fail(sequence int64, err error) (execution.Decision, error) {
	e.incomplete = true
	if e.failure == nil {
		e.failure = err
		e.failureSequence = sequence
	}
	return e.decision(), err
}
func (e *Evaluator) decision() execution.Decision {
	if e.violated {
		return execution.Stop
	}
	return execution.Continue
}
func (e *Evaluator) Observe(ctx context.Context, event *umpirespb.RunEvent) (execution.Decision, error) {
	if e.closed || e.sawClosure {
		return e.decision(), invalid(ir.Malformed, "event after evaluator closure")
	}
	if e.failure != nil {
		return e.decision(), e.failure
	}
	if ctx == nil {
		return e.fail(event.GetSequence(), invalid(ir.Malformed, "context required"))
	}
	if err := ctx.Err(); err != nil {
		return e.fail(event.GetSequence(), err)
	}
	observations, err := e.checkEvent(event)
	if err != nil {
		return e.fail(event.GetSequence(), err)
	}
	incomplete := e.incomplete || event.ExecutionIncomplete
	if e.violated || incomplete {
		e.sawClosure = event.Kind == umpirespb.RUN_EVENT_KIND_RUN_CLOSED
		e.sequence = event.Sequence
		e.elapsed = event.ElapsedMilliseconds
		e.incomplete = incomplete
		return e.decision(), nil
	}
	work := e.prepared.workPerEvent
	if work > e.prepared.source.Limits.MaxTotalWork-e.totalWork {
		return e.fail(event.Sequence, invalid(ir.LimitExceeded, "total evaluation work exceeded"))
	}
	changes, count, bytes, err := e.changes(ctx, event, observations, incomplete)
	if err != nil {
		return e.fail(event.Sequence, err)
	}
	if err := ctx.Err(); err != nil {
		return e.fail(event.Sequence, err)
	}
	support := false
	for _, change := range changes {
		state := &e.rules[change.rule]
		state.state = change.state
		for id, value := range change.captures {
			state.captures[id] = value
		}
		if change.support {
			state.support = append(state.support, event.Sequence)
			support = true
			e.result.Rules[change.rule].SupportingEventSequences = state.support
		}
		e.trace = append(e.trace, change.trace)
		e.recordTerminal(change)
	}
	if support {
		e.result.SupportingEventSequences = append(e.result.SupportingEventSequences, event.Sequence)
	}
	e.captureCount += count
	e.captureBytes += bytes
	e.totalWork += work
	e.sawClosure = event.Kind == umpirespb.RUN_EVENT_KIND_RUN_CLOSED
	e.sequence = event.Sequence
	e.elapsed = event.ElapsedMilliseconds
	e.incomplete = incomplete
	return e.decision(), nil
}
func (e *Evaluator) checkEvent(event *umpirespb.RunEvent) (map[string]*umpirespb.Value, error) {
	limits := e.prepared.program.Limits()
	if event == nil || event.Sequence != e.sequence+1 || event.Sequence > limits.MaxRunEvents || event.ElapsedMilliseconds < e.elapsed || event.ElapsedMilliseconds < 0 {
		return nil, invalid(ir.Malformed, "invalid event sequence or elapsed coordinate")
	}
	if e.sequence == 0 && (event.Kind != umpirespb.RUN_EVENT_KIND_RUN_OPENED || event.ElapsedMilliseconds != 0) {
		return nil, invalid(ir.Malformed, "Run must open at elapsed zero")
	}
	if event.Kind < umpirespb.RUN_EVENT_KIND_RUN_OPENED || event.Kind > umpirespb.RUN_EVENT_KIND_DIAGNOSTIC || (e.sequence > 0 && event.Kind == umpirespb.RUN_EVENT_KIND_RUN_OPENED) {
		return nil, invalid(ir.Malformed, "invalid lifecycle event")
	}
	if err := ir.CheckSurface(event, ir.DefaultLimits()); err != nil {
		return nil, err
	}
	values := make(map[string]*umpirespb.Value, len(event.Observations))
	if len(event.Observations) > len(e.prepared.observations) {
		return nil, invalid(ir.LimitExceeded, "observation count exceeded")
	}
	for _, o := range event.Observations {
		typ, ok := e.prepared.observations[o.ObservationId]
		if !ok || values[o.ObservationId] != nil {
			return nil, invalid(ir.Malformed, "unknown or repeated Observation")
		}
		bounds := ir.DefaultLimits()
		bounds.Bytes = limits.MaxResponseBytes
		bounds.Fanout = limits.MaxPathFanout
		if err := e.prepared.catalog.CheckLiteral(o.Value, typ, bounds); err != nil {
			return nil, err
		}
		values[o.ObservationId] = o.Value
	}
	return values, nil
}

type eventEvaluation struct{ count, bytes, work int64 }

func (e *Evaluator) changes(ctx context.Context, event *umpirespb.RunEvent, observations map[string]*umpirespb.Value, incomplete bool) (staged []ruleChange, captureCount int64, captureBytes int64, resultErr error) {
	var changes []ruleChange
	cost := &eventEvaluation{}
	for i := range e.prepared.rules {
		change, err := e.nextChange(ctx, i, event, observations, incomplete, cost)
		if err != nil {
			return nil, 0, 0, err
		}
		if change != nil {
			changes = append(changes, *change)
		}
	}
	if cost.work > e.prepared.workPerEvent {
		return nil, 0, 0, invalid(ir.LimitExceeded, "prepared per-event work exceeded")
	}
	return changes, cost.count, cost.bytes, nil
}
func (e *Evaluator) nextChange(ctx context.Context, i int, event *umpirespb.RunEvent, observations map[string]*umpirespb.Value, incomplete bool, cost *eventEvaluation) (*ruleChange, error) {
	m, state := e.prepared.rules[i], e.rules[i]
	if m.source.States[state.state].Terminal != umpirespb.CONTRACT_TERMINAL_STATE_NONTERMINAL {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cost.work++
	if m.source.Horizon != nil && event.ElapsedMilliseconds >= m.source.Horizon.ElapsedMilliseconds {
		if incomplete {
			return nil, nil
		}
		return &ruleChange{rule: i, state: m.states[m.source.Horizon.ViolationStateId], support: true, trace: transitionTrace{event.Sequence, m.source.RuleId, "", m.source.States[state.state].StateId, m.source.Horizon.ViolationStateId}}, nil
	}
	resolve := func(ref ir.Reference) *umpirespb.Value {
		switch ref.Kind {
		case ir.ObservationReference:
			return observations[ref.ID]
		case ir.CaptureReference:
			return state.captures[ref.ID].value
		case ir.EventReference:
			return eventValue(event, umpirespb.RunEventField(ref.Field))
		default:
			return nil
		}
	}
	for _, index := range m.outgoing[state.state][event.Kind] {
		matched, used, err := m.transitions[index].Evaluate(ctx, resolve, e.prepared.source.Limits.MaxWorkPerEvent-cost.work)
		cost.work += used
		if err != nil {
			return nil, err
		}
		if !matched.GetBoolValue() {
			continue
		}
		tr := m.source.Transitions[index]
		captures, err := e.stageCaptures(state, tr, event, observations, cost)
		if err != nil {
			return nil, err
		}
		return &ruleChange{rule: i, state: m.states[tr.TargetState], captures: captures, support: tr.Support == umpirespb.CONTRACT_SUPPORT_MATCHING_EVENT, trace: transitionTrace{event.Sequence, m.source.RuleId, tr.TransitionId, tr.SourceState, tr.TargetState}}, nil
	}
	return nil, nil
}
func (e *Evaluator) stageCaptures(state ruleState, tr *umpirespb.ContractTransition, event *umpirespb.RunEvent, observations map[string]*umpirespb.Value, cost *eventEvaluation) (map[string]capturedValue, error) {
	captures := map[string]capturedValue{}
	for _, assignment := range tr.CaptureAssignments {
		value := observations[assignment.Observation.ObservationId]
		if value == nil || state.captures[assignment.CaptureId].value != nil {
			return nil, invalid(ir.Malformed, "missing or repeated capture assignment")
		}
		size := int64(proto.Size(value)) + 8
		if err := add(&cost.count, 1, e.prepared.source.Limits.MaxCaptures-e.captureCount); err != nil {
			return nil, err
		}
		if err := add(&cost.bytes, size, e.prepared.source.Limits.MaxCaptureBytes-e.captureBytes); err != nil {
			return nil, err
		}
		if err := add(&cost.work, size, e.prepared.source.Limits.MaxWorkPerEvent); err != nil {
			return nil, err
		}
		captures[assignment.CaptureId] = capturedValue{proto.CloneOf(value), event.Sequence}
	}
	return captures, nil
}
func eventValue(event *umpirespb.RunEvent, field umpirespb.RunEventField) *umpirespb.Value {
	var text string
	var number int64
	switch field {
	case umpirespb.RUN_EVENT_FIELD_SEQUENCE:
		number = event.Sequence
	case umpirespb.RUN_EVENT_FIELD_ELAPSED_MILLISECONDS:
		number = event.ElapsedMilliseconds
	case umpirespb.RUN_EVENT_FIELD_ATTEMPT:
		number = event.Coordinates.GetAttempt()
	case umpirespb.RUN_EVENT_FIELD_KIND:
		return &umpirespb.Value{Value: &umpirespb.Value_EnumValue{EnumValue: &umpirespb.EnumValue{Number: int32(event.Kind)}}}
	case umpirespb.RUN_EVENT_FIELD_ENTRYPOINT_ID:
		text = event.Coordinates.GetEntrypointId()
	case umpirespb.RUN_EVENT_FIELD_ACTIVATION_ID:
		text = event.Coordinates.GetActivationId()
	case umpirespb.RUN_EVENT_FIELD_INSTRUCTION_ID:
		text = event.Coordinates.GetInstructionId()
	case umpirespb.RUN_EVENT_FIELD_SOURCE_ID:
		text = event.SourceId
	default:
		return nil
	}
	if field == umpirespb.RUN_EVENT_FIELD_SEQUENCE || field == umpirespb.RUN_EVENT_FIELD_ELAPSED_MILLISECONDS || field == umpirespb.RUN_EVENT_FIELD_ATTEMPT {
		return &umpirespb.Value{Value: &umpirespb.Value_SignedInteger{SignedInteger: strconv.FormatInt(number, 10)}}
	}
	return &umpirespb.Value{Value: &umpirespb.Value_Text{Text: text}}
}

// Close transfers the frozen result once, including on failure; subsequent callbacks are rejected.
func (e *Evaluator) Close(ctx context.Context, run *umpirespb.Run) (*umpirespb.Verdict, error) {
	if e.closed {
		return nil, invalid(ir.Malformed, "evaluator already closed")
	}
	var err error
	if ctx == nil {
		err = invalid(ir.Malformed, "context required")
	} else {
		err = ctx.Err()
	}
	if err == nil {
		err = e.checkClosure(ctx, run)
	}
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		e.incomplete = true
	}
	result := e.verdict(run.GetDisposition())
	e.closed = true
	e.result = nil
	e.rules = nil
	return result, err
}
func (e *Evaluator) checkClosure(ctx context.Context, run *umpirespb.Run) error {
	if run == nil || run.RunId == "" || run.ProgramId != e.prepared.program.ProgramID() || len(run.Events) == 0 {
		return invalid(ir.Malformed, "closed Run identity and events required")
	}
	if run.Disposition < umpirespb.RUN_DISPOSITION_COMPLETED || run.Disposition > umpirespb.RUN_DISPOSITION_INCOMPLETE {
		return invalid(ir.Malformed, "closed Run disposition required")
	}
	if int64(len(run.Events)) > e.prepared.program.Limits().MaxRunEvents {
		return invalid(ir.LimitExceeded, "Run event count exceeded")
	}
	if run.Events[len(run.Events)-1].GetKind() != umpirespb.RUN_EVENT_KIND_RUN_CLOSED {
		return invalid(ir.Malformed, "Run closure event required")
	}
	if err := checkRunOrder(ctx, run.Events); err != nil {
		return err
	}
	return e.checkDisposition(run)
}
func (e *Evaluator) checkDisposition(run *umpirespb.Run) error {
	failure := run.EvaluationFailureSequence
	if failure != nil && (failure.Value <= 0 || failure.Value > int64(len(run.Events)) || run.Events[failure.Value-1].GetSequence() != failure.Value) {
		return invalid(ir.Malformed, "invalid evaluation failure sequence")
	}
	if failure == nil && e.sequence != int64(len(run.Events)) || failure != nil && e.sequence != failure.Value-1 {
		return invalid(ir.Malformed, "closed Run differs from evaluated prefix")
	}
	if e.failureSequence > 0 && (failure == nil || failure.Value != e.failureSequence) {
		return invalid(ir.Malformed, "Monitor failure coordinate missing or inconsistent")
	}
	if run.Disposition == umpirespb.RUN_DISPOSITION_COMPLETED && (e.incomplete || failure != nil || e.violated) {
		return invalid(ir.Malformed, "completed disposition conflicts with incompleteness")
	}
	if run.Disposition == umpirespb.RUN_DISPOSITION_STOPPED_BY_MONITOR && !e.violated {
		return invalid(ir.Malformed, "monitor stop without proved violation")
	}
	return nil
}
func (e *Evaluator) recordTerminal(change ruleChange) {
	terminal := e.prepared.rules[change.rule].source.States[change.state]
	result := e.result.Rules[change.rule]
	switch terminal.Terminal {
	case umpirespb.CONTRACT_TERMINAL_STATE_VIOLATED:
		result.Kind = umpirespb.RULE_VERDICT_KIND_VIOLATED
		result.TerminalStateId = terminal.StateId
		e.violated = true
	case umpirespb.CONTRACT_TERMINAL_STATE_SATISFIED:
		result.Kind = umpirespb.RULE_VERDICT_KIND_SATISFIED
		result.TerminalStateId = terminal.StateId
		e.satisfied++
	default:
	}
}
func (e *Evaluator) verdict(disposition umpirespb.RunDisposition) *umpirespb.Verdict {
	e.result.Kind = umpirespb.VERDICT_KIND_INCONCLUSIVE
	if !e.incomplete && disposition == umpirespb.RUN_DISPOSITION_COMPLETED && e.satisfied == len(e.prepared.rules) {
		e.result.Kind = umpirespb.VERDICT_KIND_SATISFIED
	}
	if e.violated {
		e.result.Kind = umpirespb.VERDICT_KIND_VIOLATED
	}
	return e.result
}

// Evaluate replays the recorded prefix through the same per-Run machine used by live callbacks.
func (p *PreparedContract) Evaluate(ctx context.Context, run *umpirespb.Run) (*umpirespb.Verdict, error) {
	_, verdict, err := p.evaluate(ctx, run)
	return verdict, err
}
func (p *PreparedContract) evaluate(ctx context.Context, run *umpirespb.Run) (*Evaluator, *umpirespb.Verdict, error) {
	if p == nil {
		return nil, nil, invalid(ir.Malformed, "prepared Contract required")
	}
	e, err := p.newEvaluator(ctx, p.program)
	if err != nil {
		return nil, nil, err
	}
	if run == nil {
		return e, e.verdict(umpirespb.RUN_DISPOSITION_INCOMPLETE), invalid(ir.Malformed, "Run required")
	}
	failure := run.EvaluationFailureSequence
	if failure != nil && (failure.Value <= 0 || failure.Value > int64(len(run.Events))) {
		return e, e.verdict(umpirespb.RUN_DISPOSITION_INCOMPLETE), invalid(ir.Malformed, "invalid evaluation failure sequence")
	}
	for _, event := range run.Events {
		if failure != nil && event.GetSequence() >= failure.Value {
			e.incomplete = true
			e.failureSequence = failure.Value
			break
		}
		if _, err := e.Observe(ctx, event); err != nil {
			verdict, _ := e.Close(ctx, run)
			return e, verdict, fmt.Errorf("event %d: %w", event.GetSequence(), err)
		}
	}
	verdict, err := e.Close(ctx, run)
	return e, verdict, err
}

func checkRunOrder(ctx context.Context, events []*umpirespb.RunEvent) error {
	var elapsed int64
	for i, event := range events {
		if err := ctx.Err(); err != nil {
			return err
		}
		if event == nil || event.Sequence != int64(i+1) || event.ElapsedMilliseconds < elapsed {
			return invalid(ir.Malformed, "invalid Run event ordering")
		}
		if i == 0 && (event.Kind != umpirespb.RUN_EVENT_KIND_RUN_OPENED || event.ElapsedMilliseconds != 0) {
			return invalid(ir.Malformed, "invalid Run opening")
		}
		if i > 0 && event.Kind == umpirespb.RUN_EVENT_KIND_RUN_OPENED || i < len(events)-1 && event.Kind == umpirespb.RUN_EVENT_KIND_RUN_CLOSED {
			return invalid(ir.Malformed, "invalid Run lifecycle")
		}
		elapsed = event.ElapsedMilliseconds
	}
	return nil
}
