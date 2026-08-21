package protocol

import (
	"errors"
	"fmt"
	"slices"
)

type FirstOrderStep struct {
	Action ActionKind      `json:"action"`
	State  FirstOrderState `json:"state"`
}

type FirstOrderMachine struct {
	view FirstOrderView
}

func NewFirstOrderMachine(view FirstOrderView) (FirstOrderMachine, error) {
	if err := view.Validate(); err != nil {
		return FirstOrderMachine{}, err
	}
	return FirstOrderMachine{view: view}, nil
}

func (m FirstOrderMachine) InitialStates() ([]FirstOrderState, error) {
	domain, err := m.stateDomain()
	if err != nil {
		return nil, err
	}
	initials := make([]FirstOrderState, 0)
	for _, state := range domain {
		runtimeState, err := newFirstOrderRuntimeState(m.view.StateFields, state)
		if err != nil {
			return nil, err
		}
		if evaluateFirstOrderFormula(runtimeState, m.view.Initial) {
			initials = append(initials, state)
		}
	}
	if len(initials) == 0 {
		return nil, errors.New("first-order machine has no initial state")
	}
	return initials, nil
}

func (m FirstOrderMachine) Successors(state FirstOrderState) ([]FirstOrderStep, error) {
	runtimeState, err := newFirstOrderRuntimeState(m.view.StateFields, state)
	if err != nil {
		return nil, err
	}
	steps := make([]FirstOrderStep, 0, len(m.view.Actions))
	for _, action := range m.view.Actions {
		if !evaluateFirstOrderFormula(runtimeState, action.Guard) {
			continue
		}
		next := applyFirstOrderUpdates(runtimeState, action.Updates)
		steps = append(steps, FirstOrderStep{
			Action: ActionKind(action.Identifier),
			State:  next.firstOrderState(m.view.StateFields),
		})
	}
	slices.SortFunc(steps, func(left, right FirstOrderStep) int {
		if order := compareStrings(string(left.Action), string(right.Action)); order != 0 {
			return order
		}
		leftKey, _ := m.StateKey(left.State)
		rightKey, _ := m.StateKey(right.State)
		return compareStrings(leftKey, rightKey)
	})
	return steps, nil
}

func (m FirstOrderMachine) Invariant(state FirstOrderState) (bool, error) {
	runtimeState, err := newFirstOrderRuntimeState(m.view.StateFields, state)
	if err != nil {
		return false, err
	}
	return evaluateFirstOrderFormula(runtimeState, m.view.Invariant), nil
}

func (m FirstOrderMachine) StateKey(state FirstOrderState) (string, error) {
	runtimeState, err := newFirstOrderRuntimeState(m.view.StateFields, state)
	if err != nil {
		return "", err
	}
	return runtimeState.key(m.view.StateFields), nil
}

func (m FirstOrderMachine) stateDomain() ([]FirstOrderState, error) {
	states := []FirstOrderState{{Fields: []FirstOrderBinding{}}}
	for _, field := range m.view.StateFields {
		sort, found := findFirstOrderSort(m.view.Sorts, field.Sort)
		if !found {
			return nil, fmt.Errorf("state field %q has unknown sort %q", field.Identifier, field.Sort)
		}
		values := append([]string(nil), sort.Values...)
		if sort.Kind == FirstOrderSortUninterpreted {
			values = make([]string, sort.Cardinality)
			for index := range values {
				values[index] = fmt.Sprintf("member-%d", index)
			}
		}
		if len(states) > m.view.Bounds.ConcreteStateLimit/len(values) {
			return nil, fmt.Errorf("first-order state domain exceeds %d states",
				m.view.Bounds.ConcreteStateLimit)
		}
		expanded := make([]FirstOrderState, 0, len(states)*len(values))
		for _, state := range states {
			for _, value := range values {
				fields := append([]FirstOrderBinding(nil), state.Fields...)
				fields = append(fields, FirstOrderBinding{Field: field.Identifier, Value: value})
				expanded = append(expanded, FirstOrderState{Fields: fields})
			}
		}
		states = expanded
	}
	return states, nil
}

func (s firstOrderRuntimeState) firstOrderState(fields []FirstOrderField) FirstOrderState {
	bindings := make([]FirstOrderBinding, len(fields))
	for index, field := range fields {
		bindings[index] = FirstOrderBinding{Field: field.Identifier, Value: s[field.Identifier]}
	}
	return FirstOrderState{Fields: bindings}
}

func findFirstOrderSort(sorts []FirstOrderSort, identifier string) (FirstOrderSort, bool) {
	for _, sort := range sorts {
		if sort.Identifier == identifier {
			return sort, true
		}
	}
	return FirstOrderSort{}, false
}
