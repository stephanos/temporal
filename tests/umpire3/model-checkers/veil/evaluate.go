package veil

import (
	"fmt"
	"slices"
	"strings"

	"go.temporal.io/server/tests/umpire3/protocol"
)

type semanticState struct {
	values []string
}

func CompareReachableStates(view protocol.FirstOrderView) error {
	if err := view.Validate(); err != nil {
		return fmt.Errorf("validate first-order view: %w", err)
	}
	reachable, err := reachableStates(view)
	if err != nil {
		return err
	}
	actual := make(map[string]struct{}, len(reachable))
	for _, state := range reachable {
		actual[stateKey(view.StateFields, state.values)] = struct{}{}
	}
	expected := make(map[string]struct{}, len(view.Oracle.States))
	for _, state := range view.Oracle.States {
		values := make([]string, len(view.StateFields))
		for index, field := range view.StateFields {
			for _, binding := range state.Fields {
				if binding.Field == field.Identifier {
					values[index] = binding.Value
					break
				}
			}
		}
		expected[stateKey(view.StateFields, values)] = struct{}{}
	}
	missing := difference(expected, actual)
	if len(missing) != 0 {
		return fmt.Errorf("first-order translation is missing reachable state %s", missing[0])
	}
	unexpected := difference(actual, expected)
	if len(unexpected) != 0 {
		return fmt.Errorf("first-order translation has unexpected reachable state %s", unexpected[0])
	}
	return nil
}

func reachableStates(view protocol.FirstOrderView) ([]semanticState, error) {
	states, err := enumerateStates(view)
	if err != nil {
		return nil, err
	}
	queue := make([]semanticState, 0, len(states))
	seen := make(map[string]struct{}, len(states))
	for _, state := range states {
		if !evaluateFormula(view, state, view.Initial) {
			continue
		}
		key := stateKey(view.StateFields, state.values)
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		queue = append(queue, state)
	}
	for next := 0; next < len(queue); next++ {
		state := queue[next]
		for _, action := range view.Actions {
			if !evaluateFormula(view, state, action.Guard) {
				continue
			}
			successor := applyUpdates(view, state, action.Updates)
			key := stateKey(view.StateFields, successor.values)
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			if len(queue) >= view.Bounds.ConcreteStateLimit {
				return nil, fmt.Errorf("first-order differential exploration exceeds %d states", view.Bounds.ConcreteStateLimit)
			}
			seen[key] = struct{}{}
			queue = append(queue, successor)
		}
	}
	return queue, nil
}

func enumerateStates(view protocol.FirstOrderView) ([]semanticState, error) {
	states := []semanticState{{values: make([]string, 0, len(view.StateFields))}}
	for _, field := range view.StateFields {
		sort, ok := findSort(view.Sorts, field.Sort)
		if !ok {
			return nil, fmt.Errorf("state field %q has unknown sort %q", field.Identifier, field.Sort)
		}
		values := sort.Values
		if sort.Kind == protocol.FirstOrderSortUninterpreted {
			values = make([]string, sort.Cardinality)
			for index := range values {
				values[index] = fmt.Sprintf("member-%d", index)
			}
		}
		if len(states) > view.Bounds.ConcreteStateLimit/len(values) {
			return nil, fmt.Errorf("first-order state domain exceeds %d states", view.Bounds.ConcreteStateLimit)
		}
		expanded := make([]semanticState, 0, len(states)*len(values))
		for _, state := range states {
			for _, value := range values {
				next := append([]string(nil), state.values...)
				next = append(next, value)
				expanded = append(expanded, semanticState{values: next})
			}
		}
		states = expanded
	}
	return states, nil
}

func evaluateFormula(view protocol.FirstOrderView, state semanticState, formula protocol.FirstOrderFormula) bool {
	switch formula.Kind {
	case protocol.FirstOrderFormulaTrue:
		return true
	case protocol.FirstOrderFormulaEqual:
		return evaluateTerm(view, state, *formula.Left) == evaluateTerm(view, state, *formula.Right)
	case protocol.FirstOrderFormulaNot:
		return !evaluateFormula(view, state, *formula.Operand)
	case protocol.FirstOrderFormulaAll:
		for _, operand := range formula.Operands {
			if !evaluateFormula(view, state, operand) {
				return false
			}
		}
		return true
	case protocol.FirstOrderFormulaAny:
		for _, operand := range formula.Operands {
			if evaluateFormula(view, state, operand) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func evaluateTerm(view protocol.FirstOrderView, state semanticState, term protocol.FirstOrderTerm) string {
	if term.Kind == protocol.FirstOrderTermValue {
		return term.Value
	}
	for index, field := range view.StateFields {
		if field.Identifier == term.Field {
			return state.values[index]
		}
	}
	return ""
}

func applyUpdates(
	view protocol.FirstOrderView,
	state semanticState,
	updates []protocol.FirstOrderUpdate,
) semanticState {
	values := append([]string(nil), state.values...)
	for _, update := range updates {
		value := evaluateTerm(view, state, update.Value)
		for index, field := range view.StateFields {
			if field.Identifier == update.Field {
				values[index] = value
				break
			}
		}
	}
	return semanticState{values: values}
}

func stateKey(fields []protocol.FirstOrderField, values []string) string {
	parts := make([]string, len(fields))
	for index, field := range fields {
		parts[index] = field.Identifier + "=" + values[index]
	}
	return strings.Join(parts, ",")
}

func findSort(sorts []protocol.FirstOrderSort, identifier string) (protocol.FirstOrderSort, bool) {
	for _, sort := range sorts {
		if sort.Identifier == identifier {
			return sort, true
		}
	}
	return protocol.FirstOrderSort{}, false
}

func difference(left, right map[string]struct{}) []string {
	result := make([]string, 0)
	for value := range left {
		if _, found := right[value]; !found {
			result = append(result, value)
		}
	}
	slices.Sort(result)
	return result
}
