//nolint:revive // The package name is the public Umpire3 runtime.Run seam.
package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"

	"go.temporal.io/server/tests/umpire3/protocol"
)

type ExecuteCandidate func(context.Context, protocol.Experiment) (Result, error)

func MinimizeActions(ctx context.Context, experiment protocol.Experiment, execute ExecuteCandidate) (protocol.Experiment, error) {
	if execute == nil {
		return protocol.Experiment{}, errors.New("candidate executor is required")
	}
	if err := experiment.Validate(); err != nil {
		return protocol.Experiment{}, fmt.Errorf("validate original experiment: %w", err)
	}
	result, err := execute(ctx, experiment)
	if err != nil {
		return protocol.Experiment{}, fmt.Errorf("execute original experiment: %w", err)
	}
	if !isRequestedViolation(experiment, result) {
		return protocol.Experiment{}, errors.New("original experiment does not produce the requested property violation")
	}
	violationCheckpoint := result.Claim.Checkpoint
	violationIdentity := qualifiedViolationIdentity(result)

	minimized := experiment
	for index := 0; index < len(minimized.Actions) && len(minimized.Actions) > 1; {
		candidate := removeAction(minimized, index)
		if candidate.Scope.Bounds.MaxDepth > len(candidate.Actions) {
			candidate.Scope.Bounds.MaxDepth = len(candidate.Actions)
		}
		if err := candidate.Validate(); err != nil {
			index++
			continue
		}
		result, err := execute(ctx, candidate)
		if err != nil {
			return protocol.Experiment{}, fmt.Errorf("execute candidate without action %q: %w", minimized.Actions[index].Identifier, err)
		}
		if isRequestedViolation(experiment, result) && result.Claim.Checkpoint == violationCheckpoint &&
			qualifiedViolationIdentity(result) == violationIdentity {
			minimized = candidate
			index = 0
			continue
		}
		index++
	}
	return minimized, nil
}

func MinimizeExperiment(ctx context.Context, experiment protocol.Experiment, execute ExecuteCandidate) (protocol.Experiment, error) {
	minimized, err := MinimizeActions(ctx, experiment, execute)
	if err != nil {
		return protocol.Experiment{}, err
	}
	baseline, err := execute(ctx, minimized)
	if err != nil {
		return protocol.Experiment{}, err
	}
	checkpoint := baseline.Claim.Checkpoint
	identity := qualifiedViolationIdentity(baseline)
	preserves := func(candidate protocol.Experiment) (bool, error) {
		result, err := execute(ctx, candidate)
		if err != nil {
			return false, err
		}
		return isRequestedViolation(experiment, result) && result.Claim.Checkpoint == checkpoint &&
			qualifiedViolationIdentity(result) == identity, nil
	}

	for index := 0; index < len(minimized.Faults); {
		candidate := minimized
		candidate.Faults = append([]protocol.Fault(nil), minimized.Faults[:index]...)
		candidate.Faults = append(candidate.Faults, minimized.Faults[index+1:]...)
		preserved, err := preservesValid(candidate, preserves)
		if err != nil {
			return protocol.Experiment{}, err
		}
		if preserved {
			minimized = candidate
			index = 0
			continue
		}
		index++
	}
	for index := 0; index < len(minimized.Policies); {
		candidate := minimized
		candidate.Policies = append([]protocol.Policy(nil), minimized.Policies[:index]...)
		candidate.Policies = append(candidate.Policies, minimized.Policies[index+1:]...)
		preserved, err := preservesValid(candidate, preserves)
		if err != nil {
			return protocol.Experiment{}, err
		}
		if preserved {
			minimized = candidate
			index = 0
			continue
		}
		index++
	}
	for index := 0; index < len(minimized.Order); {
		candidate := minimized
		candidate.Order = append([]protocol.OrderConstraint(nil), minimized.Order[:index]...)
		candidate.Order = append(candidate.Order, minimized.Order[index+1:]...)
		preserved, err := preservesValid(candidate, preserves)
		if err != nil {
			return protocol.Experiment{}, err
		}
		if preserved {
			minimized = candidate
			index = 0
			continue
		}
		index++
	}

	for index := 0; index < len(minimized.Resources) && len(minimized.Resources) > 1; {
		candidate := minimized
		candidate.Resources = append([]protocol.Resource(nil), minimized.Resources[:index]...)
		candidate.Resources = append(candidate.Resources, minimized.Resources[index+1:]...)
		preserved, err := preservesValid(candidate, preserves)
		if err != nil {
			return protocol.Experiment{}, err
		}
		if preserved {
			minimized = candidate
			index = 0
			continue
		}
		index++
	}
	for actionIndex := range minimized.Actions {
		argumentNames := make([]string, 0, len(minimized.Actions[actionIndex].Arguments))
		for _, argument := range minimized.Actions[actionIndex].Arguments {
			argumentNames = append(argumentNames, argument.Name)
		}
		slices.Sort(argumentNames)
		for _, name := range argumentNames {
			candidate := minimized
			candidate.Actions = append([]protocol.Action(nil), minimized.Actions...)
			candidate.Actions[actionIndex].Arguments = removeNamedValue(
				minimized.Actions[actionIndex].Arguments,
				name,
			)
			preserved, err := preservesValid(candidate, preserves)
			if err != nil {
				return protocol.Experiment{}, err
			}
			if preserved {
				minimized = candidate
			}
		}
		for argumentIndex := range minimized.Actions[actionIndex].Arguments {
			for {
				accepted := false
				for _, value := range simplerValues(minimized.Actions[actionIndex].Arguments[argumentIndex].Value) {
					candidate := minimized
					candidate.Actions = append([]protocol.Action(nil), minimized.Actions...)
					candidate.Actions[actionIndex].Arguments = append([]protocol.NamedValue(nil),
						minimized.Actions[actionIndex].Arguments...)
					candidate.Actions[actionIndex].Arguments[argumentIndex].Value = value
					preserved, err := preservesValid(candidate, preserves)
					if err != nil {
						return protocol.Experiment{}, err
					}
					if preserved {
						minimized = candidate
						accepted = true
						break
					}
				}
				if !accepted {
					break
				}
			}
		}

		bindingSymbols := make([]string, 0, len(minimized.Actions[actionIndex].Bindings))
		for _, binding := range minimized.Actions[actionIndex].Bindings {
			bindingSymbols = append(bindingSymbols, binding.Symbol)
		}
		slices.Sort(bindingSymbols)
		for _, symbol := range bindingSymbols {
			candidate := minimized
			candidate.Actions = append([]protocol.Action(nil), minimized.Actions...)
			candidate.Actions[actionIndex].Bindings = removeBinding(
				minimized.Actions[actionIndex].Bindings,
				symbol,
			)
			preserved, err := preservesValid(candidate, preserves)
			if err != nil {
				return protocol.Experiment{}, err
			}
			if preserved {
				minimized = candidate
			}
		}
	}
	for faultIndex := range minimized.Faults {
		for {
			accepted := false
			for _, candidate := range simplifyFault(minimized, faultIndex) {
				preserved, err := preservesValid(candidate, preserves)
				if err != nil {
					return protocol.Experiment{}, err
				}
				if preserved {
					minimized = candidate
					accepted = true
					break
				}
			}
			if !accepted {
				break
			}
		}
	}
	for policyIndex := range minimized.Policies {
		for argumentIndex := range minimized.Policies[policyIndex].Arguments {
			for {
				accepted := false
				for _, value := range simplerValues(minimized.Policies[policyIndex].Arguments[argumentIndex].Value) {
					candidate := minimized
					candidate.Policies = append([]protocol.Policy(nil), minimized.Policies...)
					candidate.Policies[policyIndex].Arguments = append([]protocol.NamedValue(nil),
						minimized.Policies[policyIndex].Arguments...)
					candidate.Policies[policyIndex].Arguments[argumentIndex].Value = value
					preserved, err := preservesValid(candidate, preserves)
					if err != nil {
						return protocol.Experiment{}, err
					}
					if preserved {
						minimized = candidate
						accepted = true
						break
					}
				}
				if !accepted {
					break
				}
			}
		}
	}
	return minimized, nil
}

func simplerValues(value protocol.Value) []protocol.Value {
	var result []protocol.Value
	switch value.Type {
	case protocol.ValueString:
		empty := ""
		if value.Text != nil && *value.Text != empty {
			result = append(result, protocol.Value{Type: value.Type, Text: &empty})
		}
	case protocol.ValueInteger, protocol.ValueDuration:
		zero := int64(0)
		if value.Integer != nil && *value.Integer != zero {
			result = append(result, protocol.Value{Type: value.Type, Integer: &zero})
		}
	case protocol.ValueBoolean:
		falseValue := false
		if value.Boolean != nil && *value.Boolean {
			result = append(result, protocol.Value{Type: value.Type, Boolean: &falseValue})
		}
	case protocol.ValueEnum:
		name := ""
		number := int64(0)
		if value.Text != nil && value.Integer != nil && (*value.Text != "" || *value.Integer != 0) {
			result = append(result, protocol.Value{Type: value.Type, Text: &name, Integer: &number})
		}
	case protocol.ValueBytesDigest:
		sum := sha256.Sum256(nil)
		emptyDigest := "sha256:" + hex.EncodeToString(sum[:])
		if value.Text != nil && *value.Text != emptyDigest {
			result = append(result, protocol.Value{Type: value.Type, Text: &emptyDigest})
		}
	case protocol.ValueList:
		for index := range value.Elements {
			candidate := value
			candidate.Elements = append([]protocol.Value(nil), value.Elements[:index]...)
			candidate.Elements = append(candidate.Elements, value.Elements[index+1:]...)
			result = append(result, candidate)
		}
		for index, element := range value.Elements {
			for _, simpler := range simplerValues(element) {
				candidate := value
				candidate.Elements = append([]protocol.Value(nil), value.Elements...)
				candidate.Elements[index] = simpler
				result = append(result, candidate)
			}
		}
	case protocol.ValueRecord:
		for index := range value.Fields {
			candidate := value
			candidate.Fields = append([]protocol.NamedValue(nil), value.Fields[:index]...)
			candidate.Fields = append(candidate.Fields, value.Fields[index+1:]...)
			result = append(result, candidate)
		}
		for index, field := range value.Fields {
			for _, simpler := range simplerValues(field.Value) {
				candidate := value
				candidate.Fields = append([]protocol.NamedValue(nil), value.Fields...)
				candidate.Fields[index].Value = simpler
				result = append(result, candidate)
			}
		}
	case protocol.ValueSymbol:
	}
	return result
}

func simplifyFault(experiment protocol.Experiment, index int) []protocol.Experiment {
	var result []protocol.Experiment
	appendCandidate := func(update func(*protocol.Fault)) {
		candidate := experiment
		candidate.Faults = append([]protocol.Fault(nil), experiment.Faults...)
		update(&candidate.Faults[index])
		result = append(result, candidate)
	}
	fault := experiment.Faults[index]
	if fault.Occurrence.First != 1 || fault.Occurrence.Count != 1 {
		appendCandidate(func(value *protocol.Fault) {
			value.Occurrence = protocol.FaultOccurrence{First: 1, Count: 1}
		})
	}
	for _, item := range []struct {
		length   int
		simplify func(*protocol.Fault)
	}{
		{len(fault.Scope.Resources), func(value *protocol.Fault) { value.Scope.Resources = firstString(value.Scope.Resources) }},
		{len(fault.Scope.Endpoints), func(value *protocol.Fault) { value.Scope.Endpoints = firstString(value.Scope.Endpoints) }},
		{len(fault.Scope.TaskQueues), func(value *protocol.Fault) { value.Scope.TaskQueues = firstString(value.Scope.TaskQueues) }},
		{len(fault.Scope.Services), func(value *protocol.Fault) { value.Scope.Services = firstString(value.Scope.Services) }},
		{len(fault.Scope.Routes), func(value *protocol.Fault) { value.Scope.Routes = firstString(value.Scope.Routes) }},
		{len(fault.Scope.Participants), func(value *protocol.Fault) { value.Scope.Participants = firstString(value.Scope.Participants) }},
		{len(fault.Scope.Attempts), func(value *protocol.Fault) { value.Scope.Attempts = firstInt(value.Scope.Attempts) }},
	} {
		if item.length > 1 {
			appendCandidate(item.simplify)
		}
	}
	for argumentIndex, argument := range fault.Arguments {
		for _, simpler := range simplerValues(argument.Value) {
			argumentIndex := argumentIndex
			simpler := simpler
			appendCandidate(func(value *protocol.Fault) {
				value.Arguments = append([]protocol.NamedValue(nil), value.Arguments...)
				value.Arguments[argumentIndex].Value = simpler
			})
		}
	}
	return result
}

func firstString(values []string) []string {
	if len(values) <= 1 {
		return values
	}
	return append([]string(nil), values[0])
}

func firstInt(values []int) []int {
	if len(values) <= 1 {
		return values
	}
	return append([]int(nil), values[0])
}

func removeAction(experiment protocol.Experiment, index int) protocol.Experiment {
	removed := experiment.Actions[index].Identifier
	candidate := experiment
	candidate.Actions = append([]protocol.Action(nil), experiment.Actions[:index]...)
	candidate.Actions = append(candidate.Actions, experiment.Actions[index+1:]...)

	predecessors := make([]string, 0)
	successors := make([]string, 0)
	candidate.Order = make([]protocol.OrderConstraint, 0, len(experiment.Order))
	seenOrder := make(map[protocol.OrderConstraint]struct{}, len(experiment.Order))
	for _, constraint := range experiment.Order {
		switch {
		case constraint.After == removed:
			predecessors = append(predecessors, constraint.Before)
		case constraint.Before == removed:
			successors = append(successors, constraint.After)
		default:
			candidate.Order = append(candidate.Order, constraint)
			seenOrder[constraint] = struct{}{}
		}
	}
	for _, before := range predecessors {
		for _, after := range successors {
			constraint := protocol.OrderConstraint{
				Before: before, After: after, Relation: protocol.OrderSemantic,
			}
			if before == after {
				continue
			}
			if _, exists := seenOrder[constraint]; exists {
				continue
			}
			candidate.Order = append(candidate.Order, constraint)
			seenOrder[constraint] = struct{}{}
		}
	}

	removedPolicies := make(map[string]struct{})
	candidate.Policies = make([]protocol.Policy, 0, len(experiment.Policies))
	for _, policy := range experiment.Policies {
		policy.Scope = removeString(policy.Scope, removed)
		if len(policy.Scope) == 0 {
			removedPolicies[policy.Identifier] = struct{}{}
			continue
		}
		candidate.Policies = append(candidate.Policies, policy)
	}
	if len(removedPolicies) != 0 {
		candidate.Faults = make([]protocol.Fault, 0, len(experiment.Faults))
		for _, fault := range experiment.Faults {
			if _, removedPolicy := removedPolicies[fault.Policy]; !removedPolicy {
				candidate.Faults = append(candidate.Faults, fault)
			}
		}
	}
	policyScopes := make(map[string][]string, len(candidate.Policies))
	for _, policy := range candidate.Policies {
		policyScopes[policy.Identifier] = policy.Scope
	}
	candidate.Faults = append([]protocol.Fault(nil), candidate.Faults...)
	for index := range candidate.Faults {
		scope := policyScopes[candidate.Faults[index].Policy]
		if len(scope) != 0 {
			candidate.Faults[index].Interval.StartAction = scope[0]
			candidate.Faults[index].Interval.StopAction = scope[len(scope)-1]
		}
	}
	return candidate
}

func removeNamedValue(values []protocol.NamedValue, name string) []protocol.NamedValue {
	result := make([]protocol.NamedValue, 0, len(values)-1)
	for _, value := range values {
		if value.Name != name {
			result = append(result, value)
		}
	}
	return result
}

func removeBinding(bindings []protocol.Binding, symbol string) []protocol.Binding {
	result := make([]protocol.Binding, 0, len(bindings)-1)
	for _, binding := range bindings {
		if binding.Symbol != symbol {
			result = append(result, binding)
		}
	}
	return result
}

func removeString(values []string, removed string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != removed {
			result = append(result, value)
		}
	}
	return result
}

func preservesValid(
	candidate protocol.Experiment,
	preserves func(protocol.Experiment) (bool, error),
) (bool, error) {
	if err := candidate.Validate(); err != nil {
		return false, nil
	}
	return preserves(candidate)
}

func isRequestedViolation(experiment protocol.Experiment, result Result) bool {
	return result.Claim.Kind == ClaimViolating && result.Claim.Property == experiment.Property.Identifier
}

func qualifiedViolationIdentity(result Result) string {
	type factIdentity struct {
		Kind           string
		Value          bool
		EntityIdentity string
		Lineage        string
	}
	bindings := make([]string, 0, len(result.Bindings))
	for symbol, concrete := range result.Bindings {
		bindings = append(bindings, symbol+"\x00"+concrete)
	}
	slices.Sort(bindings)
	facts := make([]factIdentity, len(result.Evidence.Facts))
	for index, fact := range result.Evidence.Facts {
		facts[index] = factIdentity{
			Kind: fact.Kind, Value: fact.Value, EntityIdentity: fact.EntityIdentity,
			Lineage: fmt.Sprint(fact.Lineage),
		}
	}
	slices.SortFunc(facts, func(left, right factIdentity) int {
		return compareViolationIdentity(fmt.Sprint(left), fmt.Sprint(right))
	})
	return fmt.Sprintf("%s\x00%s\x00%v\x00%v", result.Claim.Property, result.Claim.Checkpoint, bindings, facts)
}

func compareViolationIdentity(left, right string) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
