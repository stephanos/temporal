package protocol

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

type FirstOrderReplay struct {
	Accepted        bool
	RejectedAction  ActionKind
	LiveOnlyActions []ActionKind
}

func (v FirstOrderView) Replay(actions []ActionKind) (FirstOrderReplay, error) {
	if err := v.Validate(); err != nil {
		return FirstOrderReplay{}, err
	}
	if len(actions) == 0 {
		return FirstOrderReplay{}, errors.New("first-order replay requires actions")
	}
	states := make([]firstOrderRuntimeState, 0, len(v.Oracle.States))
	for _, state := range v.Oracle.States {
		runtimeState, err := newFirstOrderRuntimeState(v.StateFields, state)
		if err != nil {
			return FirstOrderReplay{}, err
		}
		if evaluateFirstOrderFormula(runtimeState, v.Initial) {
			states = append(states, runtimeState)
		}
	}
	if len(states) == 0 {
		return FirstOrderReplay{}, errors.New("first-order view has no oracle initial state")
	}
	liveOnlySet := make(map[ActionKind]struct{}, len(v.LiveOnlyActions))
	for _, action := range v.LiveOnlyActions {
		liveOnlySet[action] = struct{}{}
	}
	var liveOnly []ActionKind
	for _, identifier := range actions {
		if _, live := liveOnlySet[identifier]; live {
			liveOnly = append(liveOnly, identifier)
			continue
		}
		action, found := findFirstOrderAction(v.Actions, string(identifier))
		if !found {
			return FirstOrderReplay{}, fmt.Errorf("action %q is neither modeled nor declared live-only", identifier)
		}
		nextStates := make([]firstOrderRuntimeState, 0, len(states))
		seen := make(map[string]struct{}, len(states))
		for _, state := range states {
			if !evaluateFirstOrderFormula(state, action.Guard) {
				continue
			}
			next := applyFirstOrderUpdates(state, action.Updates)
			key := next.key(v.StateFields)
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			seen[key] = struct{}{}
			nextStates = append(nextStates, next)
		}
		if len(nextStates) == 0 {
			return FirstOrderReplay{RejectedAction: identifier,
				LiveOnlyActions: sortedCompactActions(liveOnly)}, nil
		}
		states = nextStates
	}
	return FirstOrderReplay{Accepted: true, LiveOnlyActions: sortedCompactActions(liveOnly)}, nil
}

type firstOrderRuntimeState map[string]string

func newFirstOrderRuntimeState(fields []FirstOrderField, state FirstOrderState) (firstOrderRuntimeState, error) {
	result := make(firstOrderRuntimeState, len(state.Fields))
	for _, binding := range state.Fields {
		result[binding.Field] = binding.Value
	}
	if len(result) != len(fields) {
		return nil, errors.New("first-order runtime state is incomplete")
	}
	return result, nil
}

func (s firstOrderRuntimeState) key(fields []FirstOrderField) string {
	parts := make([]string, len(fields))
	for index, field := range fields {
		parts[index] = field.Identifier + "=" + s[field.Identifier]
	}
	return strings.Join(parts, "\x00")
}

func evaluateFirstOrderFormula(state firstOrderRuntimeState, formula FirstOrderFormula) bool {
	switch formula.Kind {
	case FirstOrderFormulaTrue:
		return true
	case FirstOrderFormulaEqual:
		return evaluateFirstOrderTerm(state, *formula.Left) == evaluateFirstOrderTerm(state, *formula.Right)
	case FirstOrderFormulaNot:
		return !evaluateFirstOrderFormula(state, *formula.Operand)
	case FirstOrderFormulaAll:
		for _, operand := range formula.Operands {
			if !evaluateFirstOrderFormula(state, operand) {
				return false
			}
		}
		return true
	case FirstOrderFormulaAny:
		for _, operand := range formula.Operands {
			if evaluateFirstOrderFormula(state, operand) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func evaluateFirstOrderTerm(state firstOrderRuntimeState, term FirstOrderTerm) string {
	if term.Kind == FirstOrderTermValue {
		return term.Value
	}
	return state[term.Field]
}

func applyFirstOrderUpdates(state firstOrderRuntimeState, updates []FirstOrderUpdate) firstOrderRuntimeState {
	next := make(firstOrderRuntimeState, len(state))
	for field, value := range state {
		next[field] = value
	}
	for _, update := range updates {
		next[update.Field] = evaluateFirstOrderTerm(state, update.Value)
	}
	return next
}

func findFirstOrderAction(actions []FirstOrderAction, identifier string) (FirstOrderAction, bool) {
	for _, action := range actions {
		if action.Identifier == identifier {
			return action, true
		}
	}
	return FirstOrderAction{}, false
}

func sortedCompactActions(actions []ActionKind) []ActionKind {
	result := append([]ActionKind(nil), actions...)
	slices.Sort(result)
	return slices.Compact(result)
}
