package finite

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
)

type Step struct {
	Action protocolcatalog.ActionKind      `json:"action"`
	State  protocolchecker.FirstOrderState `json:"state"`
}

type Machine struct {
	view protocolchecker.FirstOrderView
}

func NewMachine(view protocolchecker.FirstOrderView) (Machine, error) {
	if err := view.Validate(); err != nil {
		return Machine{}, err
	}
	return Machine{view: view}, nil
}

func (m Machine) InitialStates() ([]protocolchecker.FirstOrderState, error) {
	domain, err := m.stateDomain()
	if err != nil {
		return nil, err
	}
	initials := make([]protocolchecker.FirstOrderState, 0)
	for _, state := range domain {
		runtimeState, err := newRuntimeState(m.view.StateFields, state)
		if err != nil {
			return nil, err
		}
		if evaluateFormula(runtimeState, m.view.Initial) {
			initials = append(initials, state)
		}
	}
	if len(initials) == 0 {
		return nil, errors.New("first-order machine has no initial state")
	}
	return initials, nil
}

func (m Machine) Successors(state protocolchecker.FirstOrderState) ([]Step, error) {
	runtimeState, err := newRuntimeState(m.view.StateFields, state)
	if err != nil {
		return nil, err
	}
	steps := make([]Step, 0, len(m.view.Actions))
	for _, action := range m.view.Actions {
		if !evaluateFormula(runtimeState, action.Guard) {
			continue
		}
		next := applyUpdates(runtimeState, action.Updates)
		steps = append(steps, Step{
			Action: protocolcatalog.ActionKind(action.Identifier),
			State:  next.firstOrderState(m.view.StateFields),
		})
	}
	slices.SortFunc(steps, func(left, right Step) int {
		if order := strings.Compare(string(left.Action), string(right.Action)); order != 0 {
			return order
		}
		leftKey, _ := m.StateKey(left.State)
		rightKey, _ := m.StateKey(right.State)
		return strings.Compare(leftKey, rightKey)
	})
	return steps, nil
}

func (m Machine) Invariant(state protocolchecker.FirstOrderState) (bool, error) {
	runtimeState, err := newRuntimeState(m.view.StateFields, state)
	if err != nil {
		return false, err
	}
	return evaluateFormula(runtimeState, m.view.Invariant), nil
}

func (m Machine) StateKey(state protocolchecker.FirstOrderState) (string, error) {
	runtimeState, err := newRuntimeState(m.view.StateFields, state)
	if err != nil {
		return "", err
	}
	return runtimeState.key(m.view.StateFields), nil
}

func (m Machine) stateDomain() ([]protocolchecker.FirstOrderState, error) {
	states := []protocolchecker.FirstOrderState{{Fields: []protocolchecker.FirstOrderBinding{}}}
	for _, field := range m.view.StateFields {
		sort, found := findFirstOrderSort(m.view.Sorts, field.Sort)
		if !found {
			return nil, fmt.Errorf("state field %q has unknown sort %q", field.Identifier, field.Sort)
		}
		values := append([]string(nil), sort.Values...)
		if sort.Kind == protocolchecker.FirstOrderSortUninterpreted {
			values = make([]string, sort.Cardinality)
			for index := range values {
				values[index] = fmt.Sprintf("member-%d", index)
			}
		}
		if len(states) > m.view.Bounds.ConcreteStateLimit/len(values) {
			return nil, fmt.Errorf("first-order state domain exceeds %d states",
				m.view.Bounds.ConcreteStateLimit)
		}
		expanded := make([]protocolchecker.FirstOrderState, 0, len(states)*len(values))
		for _, state := range states {
			for _, value := range values {
				fields := append([]protocolchecker.FirstOrderBinding(nil), state.Fields...)
				fields = append(fields, protocolchecker.FirstOrderBinding{Field: field.Identifier, Value: value})
				expanded = append(expanded, protocolchecker.FirstOrderState{Fields: fields})
			}
		}
		states = expanded
	}
	return states, nil
}

func findFirstOrderSort(
	sorts []protocolchecker.FirstOrderSort,
	identifier string,
) (protocolchecker.FirstOrderSort, bool) {
	for _, sort := range sorts {
		if sort.Identifier == identifier {
			return sort, true
		}
	}
	return protocolchecker.FirstOrderSort{}, false
}
