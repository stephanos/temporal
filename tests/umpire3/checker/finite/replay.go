package finite

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tests/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

type FirstOrderReplay struct {
	Accepted        bool
	RejectedAction  protocolcatalog.ActionKind
	LiveOnlyActions []protocolcatalog.ActionKind
}

func ReplayFirstOrder(
	view protocolchecker.FirstOrderView,
	actions []protocolcatalog.ActionKind,
) (FirstOrderReplay, error) {
	if err := view.Validate(); err != nil {
		return FirstOrderReplay{}, err
	}
	if len(actions) == 0 {
		return FirstOrderReplay{}, errors.New("first-order replay requires actions")
	}
	states, err := initialRuntimeStates(view)
	if err != nil {
		return FirstOrderReplay{}, err
	}
	liveOnlySet := make(map[protocolcatalog.ActionKind]struct{}, len(view.LiveOnlyActions))
	for _, action := range view.LiveOnlyActions {
		liveOnlySet[action] = struct{}{}
	}
	var liveOnly []protocolcatalog.ActionKind
	for _, identifier := range actions {
		if _, live := liveOnlySet[identifier]; live {
			liveOnly = append(liveOnly, identifier)
			continue
		}
		action, found := findFirstOrderAction(view.Actions, string(identifier))
		if !found {
			return FirstOrderReplay{}, fmt.Errorf("action %q is neither modeled nor declared live-only", identifier)
		}
		nextStates := make([]runtimeState, 0, len(states))
		seen := make(map[string]struct{}, len(states))
		for _, state := range states {
			if !evaluateFormula(state, action.Guard) {
				continue
			}
			next := applyUpdates(state, action.Updates)
			key := next.key(view.StateFields)
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			seen[key] = struct{}{}
			nextStates = append(nextStates, next)
		}
		if len(nextStates) == 0 {
			return FirstOrderReplay{
				RejectedAction: identifier, LiveOnlyActions: sortedCompactActions(liveOnly),
			}, nil
		}
		states = nextStates
	}
	return FirstOrderReplay{Accepted: true, LiveOnlyActions: sortedCompactActions(liveOnly)}, nil
}

func replayAttempts(
	view protocolchecker.AttemptView,
	firstOrder protocolchecker.FirstOrderView,
	requests []AttemptRequest,
) (AttemptReplay, error) {
	if err := view.ValidateAgainst(firstOrder); err != nil {
		return AttemptReplay{}, err
	}
	if len(requests) == 0 {
		return AttemptReplay{}, errors.New("attempt replay requires actions")
	}
	states, err := initialRuntimeStates(firstOrder)
	if err != nil {
		return AttemptReplay{}, err
	}
	liveOnlySet := make(map[protocolcatalog.ActionKind]struct{}, len(firstOrder.LiveOnlyActions))
	for _, action := range firstOrder.LiveOnlyActions {
		liveOnlySet[action] = struct{}{}
	}
	var liveOnly []protocolcatalog.ActionKind
	for _, request := range requests {
		attempt, found := findAttemptMapping(view.Attempts, request.Action)
		if !found {
			return AttemptReplay{}, fmt.Errorf("action %q has no Lean-derived attempt mapping", request.Action)
		}
		outcomes := request.Outcomes
		if len(outcomes) == 0 {
			outcomes = make([]protocolexperiment.ActionOutcome, len(attempt.Outcomes))
			for index, mapping := range attempt.Outcomes {
				outcomes[index] = mapping.Outcome
			}
		}
		if err := validateRequestedOutcomes(attempt, outcomes); err != nil {
			return AttemptReplay{}, err
		}
		nextStates := make([]runtimeState, 0, len(states)*len(outcomes))
		seen := make(map[string]struct{}, len(states)*len(outcomes))
		for _, state := range states {
			for _, outcome := range outcomes {
				mapping, _ := findAttemptOutcome(attempt.Outcomes, outcome)
				next, accepted := applyAttemptOutcome(firstOrder, state, mapping)
				if !accepted {
					continue
				}
				key := next.key(firstOrder.StateFields)
				if _, duplicate := seen[key]; duplicate {
					continue
				}
				seen[key] = struct{}{}
				nextStates = append(nextStates, next)
			}
		}
		if len(nextStates) == 0 {
			rejectedOutcome := protocolexperiment.ActionOutcome("")
			if len(outcomes) == 1 {
				rejectedOutcome = outcomes[0]
			}
			return AttemptReplay{
				RejectedAction: request.Action, RejectedOutcome: rejectedOutcome,
				LiveOnlyActions: sortedCompactActions(liveOnly),
			}, nil
		}
		if _, live := liveOnlySet[request.Action]; live {
			liveOnly = append(liveOnly, request.Action)
		}
		states = nextStates
	}
	return AttemptReplay{Accepted: true, LiveOnlyActions: sortedCompactActions(liveOnly)}, nil
}

func replayObservedAttempts(
	view protocolchecker.AttemptView,
	firstOrder protocolchecker.FirstOrderView,
	attempts []ObservedAttempt,
) (AttemptReplay, error) {
	requests := make([]AttemptRequest, len(attempts))
	for index, attempt := range attempts {
		requests[index] = AttemptRequest{Action: attempt.Action, Outcomes: []protocolexperiment.ActionOutcome{attempt.Outcome}}
	}
	return replayAttempts(view, firstOrder, requests)
}

func replayFinite(target protocolchecker.FiniteReplayTarget, requests []AttemptRequest) (AttemptReplay, error) {
	if len(requests) == 0 {
		return AttemptReplay{}, errors.New("finite replay requires actions")
	}
	states := append([]int(nil), target.InitialStates...)
	var liveOnly []protocolcatalog.ActionKind
	for _, request := range requests {
		attempt, found := findFiniteAttempt(target.Attempts, request.Action)
		if !found {
			return AttemptReplay{}, fmt.Errorf("action %q has no Lean-derived finite attempt mapping", request.Action)
		}
		outcomes := request.Outcomes
		if len(outcomes) == 0 {
			outcomes = attempt.Outcomes
		}
		if err := validateFiniteReplayOutcomes(attempt, outcomes); err != nil {
			return AttemptReplay{}, err
		}
		next := make(map[int]struct{})
		for _, state := range states {
			for _, outcome := range outcomes {
				if outcome != protocolexperiment.ActionOutcomeApplied {
					next[state] = struct{}{}
					continue
				}
				for _, path := range attempt.AppliedPaths {
					for _, nextState := range followFinite(target.Transitions, []int{state}, path) {
						next[nextState] = struct{}{}
					}
					if len(path) == 0 {
						liveOnly = append(liveOnly, request.Action)
					}
				}
			}
		}
		if len(next) == 0 {
			rejectedOutcome := protocolexperiment.ActionOutcome("")
			if len(outcomes) == 1 {
				rejectedOutcome = outcomes[0]
			}
			return AttemptReplay{
				RejectedAction: request.Action, RejectedOutcome: rejectedOutcome,
				LiveOnlyActions: sortedCompactActions(liveOnly),
			}, nil
		}
		states = make([]int, 0, len(next))
		for state := range next {
			states = append(states, state)
		}
		slices.Sort(states)
	}
	return AttemptReplay{Accepted: true, LiveOnlyActions: sortedCompactActions(liveOnly)}, nil
}

func attemptOutcomes(view protocolchecker.AttemptView, action protocolcatalog.ActionKind) ([]protocolexperiment.ActionOutcome, bool) {
	mapping, found := findAttemptMapping(view.Attempts, action)
	if !found {
		return nil, false
	}
	outcomes := make([]protocolexperiment.ActionOutcome, len(mapping.Outcomes))
	for index, outcome := range mapping.Outcomes {
		outcomes[index] = outcome.Outcome
	}
	return outcomes, true
}

func finiteOutcomes(
	target protocolchecker.FiniteReplayTarget,
	action protocolcatalog.ActionKind,
) ([]protocolexperiment.ActionOutcome, bool) {
	attempt, found := findFiniteAttempt(target.Attempts, action)
	if !found {
		return nil, false
	}
	return append([]protocolexperiment.ActionOutcome(nil), attempt.Outcomes...), true
}

func initialRuntimeStates(view protocolchecker.FirstOrderView) ([]runtimeState, error) {
	states := make([]runtimeState, 0, len(view.Oracle.States))
	for _, state := range view.Oracle.States {
		runtimeState, err := newRuntimeState(view.StateFields, state)
		if err != nil {
			return nil, err
		}
		if evaluateFormula(runtimeState, view.Initial) {
			states = append(states, runtimeState)
		}
	}
	if len(states) == 0 {
		return nil, errors.New("first-order view has no oracle initial state")
	}
	return states, nil
}

func applyAttemptOutcome(
	view protocolchecker.FirstOrderView,
	state runtimeState,
	mapping protocolchecker.AttemptOutcomeMapping,
) (runtimeState, bool) {
	if !evaluateFormula(state, mapping.Guard) {
		return nil, false
	}
	next := state
	for _, identifier := range mapping.Transitions {
		action, found := findFirstOrderAction(view.Actions, string(identifier))
		if !found || !evaluateFormula(next, action.Guard) {
			return nil, false
		}
		next = applyUpdates(next, action.Updates)
	}
	return next, true
}

func validateRequestedOutcomes(
	mapping protocolchecker.AttemptMapping,
	outcomes []protocolexperiment.ActionOutcome,
) error {
	seen := make(map[protocolexperiment.ActionOutcome]struct{}, len(outcomes))
	for _, outcome := range outcomes {
		if _, duplicate := seen[outcome]; duplicate {
			return fmt.Errorf("action %q repeats requested outcome %q", mapping.Action, outcome)
		}
		seen[outcome] = struct{}{}
		if _, found := findAttemptOutcome(mapping.Outcomes, outcome); !found {
			return fmt.Errorf("action %q outcome %q is not declared by its Lean-derived attempt mapping",
				mapping.Action, outcome)
		}
	}
	return nil
}

func validateFiniteReplayOutcomes(
	attempt protocolchecker.FiniteReplayAttempt,
	outcomes []protocolexperiment.ActionOutcome,
) error {
	seen := make(map[protocolexperiment.ActionOutcome]struct{}, len(outcomes))
	for _, outcome := range outcomes {
		if _, duplicate := seen[outcome]; duplicate {
			return fmt.Errorf("action %q repeats requested outcome %q", attempt.Action, outcome)
		}
		seen[outcome] = struct{}{}
		if !slices.Contains(attempt.Outcomes, outcome) {
			return fmt.Errorf("action %q outcome %q is not declared by its Lean-derived finite mapping",
				attempt.Action, outcome)
		}
	}
	return nil
}

func findAttemptMapping(
	mappings []protocolchecker.AttemptMapping,
	action protocolcatalog.ActionKind,
) (protocolchecker.AttemptMapping, bool) {
	for _, mapping := range mappings {
		if mapping.Action == action {
			return mapping, true
		}
	}
	return protocolchecker.AttemptMapping{}, false
}

func findAttemptOutcome(
	mappings []protocolchecker.AttemptOutcomeMapping,
	outcome protocolexperiment.ActionOutcome,
) (protocolchecker.AttemptOutcomeMapping, bool) {
	for _, mapping := range mappings {
		if mapping.Outcome == outcome {
			return mapping, true
		}
	}
	return protocolchecker.AttemptOutcomeMapping{}, false
}

func findFiniteAttempt(
	attempts []protocolchecker.FiniteReplayAttempt,
	action protocolcatalog.ActionKind,
) (protocolchecker.FiniteReplayAttempt, bool) {
	for _, attempt := range attempts {
		if attempt.Action == action {
			return attempt, true
		}
	}
	return protocolchecker.FiniteReplayAttempt{}, false
}

func followFinite(
	transitions []protocolchecker.FiniteReplayTransition,
	states []int,
	path []string,
) []int {
	for _, action := range path {
		next := make(map[int]struct{})
		for _, state := range states {
			for _, transition := range transitions {
				if transition.From == state && transition.Action == action {
					next[transition.To] = struct{}{}
				}
			}
		}
		states = states[:0]
		for state := range next {
			states = append(states, state)
		}
		if len(states) == 0 {
			return nil
		}
	}
	slices.Sort(states)
	return slices.Compact(states)
}

type runtimeState map[string]string

func newRuntimeState(
	fields []protocolchecker.FirstOrderField,
	state protocolchecker.FirstOrderState,
) (runtimeState, error) {
	result := make(runtimeState, len(state.Fields))
	for _, binding := range state.Fields {
		result[binding.Field] = binding.Value
	}
	if len(result) != len(fields) {
		return nil, errors.New("first-order runtime state is incomplete")
	}
	return result, nil
}

func (state runtimeState) key(fields []protocolchecker.FirstOrderField) string {
	parts := make([]string, len(fields))
	for index, field := range fields {
		parts[index] = field.Identifier + "=" + state[field.Identifier]
	}
	return strings.Join(parts, "\x00")
}

func (state runtimeState) firstOrderState(fields []protocolchecker.FirstOrderField) protocolchecker.FirstOrderState {
	bindings := make([]protocolchecker.FirstOrderBinding, len(fields))
	for index, field := range fields {
		bindings[index] = protocolchecker.FirstOrderBinding{Field: field.Identifier, Value: state[field.Identifier]}
	}
	return protocolchecker.FirstOrderState{Fields: bindings}
}

func evaluateFormula(state runtimeState, formula protocolchecker.FirstOrderFormula) bool {
	switch formula.Kind {
	case protocolchecker.FirstOrderFormulaTrue:
		return true
	case protocolchecker.FirstOrderFormulaEqual:
		return evaluateTerm(state, *formula.Left) == evaluateTerm(state, *formula.Right)
	case protocolchecker.FirstOrderFormulaNot:
		return !evaluateFormula(state, *formula.Operand)
	case protocolchecker.FirstOrderFormulaAll:
		for _, operand := range formula.Operands {
			if !evaluateFormula(state, operand) {
				return false
			}
		}
		return true
	case protocolchecker.FirstOrderFormulaAny:
		for _, operand := range formula.Operands {
			if evaluateFormula(state, operand) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func evaluateTerm(state runtimeState, term protocolchecker.FirstOrderTerm) string {
	if term.Kind == protocolchecker.FirstOrderTermValue {
		return term.Value
	}
	return state[term.Field]
}

func applyUpdates(state runtimeState, updates []protocolchecker.FirstOrderUpdate) runtimeState {
	next := make(runtimeState, len(state))
	for field, value := range state {
		next[field] = value
	}
	for _, update := range updates {
		next[update.Field] = evaluateTerm(state, update.Value)
	}
	return next
}

func findFirstOrderAction(
	actions []protocolchecker.FirstOrderAction,
	identifier string,
) (protocolchecker.FirstOrderAction, bool) {
	for _, action := range actions {
		if action.Identifier == identifier {
			return action, true
		}
	}
	return protocolchecker.FirstOrderAction{}, false
}

func sortedCompactActions(actions []protocolcatalog.ActionKind) []protocolcatalog.ActionKind {
	result := append([]protocolcatalog.ActionKind(nil), actions...)
	slices.Sort(result)
	return slices.Compact(result)
}
