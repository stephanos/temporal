package scenario

import (
	"fmt"
	"slices"

	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

type normalizedAction struct {
	identifier      string
	kind            protocolcatalog.ActionKind
	allowedOutcomes []protocolexperiment.ActionOutcome
	arguments       []protocolexperiment.NamedValue
	bindings        []protocolexperiment.Binding
	source          Source
	generated       bool
	responseMode    protocolexperiment.ResponseMode
	maxBlockNanos   int64
}

type normalizedFault struct {
	identifier string
	kind       protocolcatalog.FaultKind
	policy     string
	arguments  []protocolexperiment.NamedValue
	scope      protocolexperiment.FaultScope
	occurrence protocolexperiment.FaultOccurrence
	configured *protocolexperiment.Fault
}

type normalizedPolicy struct {
	identifier string
	scope      []string
}

type normalizedEdge struct {
	before   string
	after    string
	relation protocolexperiment.OrderRelation
}

type normalizedPlan struct {
	actions  []*normalizedAction
	edges    []normalizedEdge
	bindings []bindIntentWithSource
	policies []normalizedPolicy
	faults   []normalizedFault
	property protocolcatalog.PropertyID
	allPaths bool
}

type bindIntentWithSource struct {
	intent bindIntent
	source Source
}

type normalizedGroup struct {
	entries []string
	exits   []string
	actions []string
}

func normalize(root Term) (*normalizedPlan, error) {
	plan := &normalizedPlan{}
	if _, err := normalizeNode(root, plan, ""); err != nil {
		return nil, err
	}
	if len(plan.actions) == 0 {
		return nil, compileError(ErrorInvalidIntent, root.source, "scenario requires at least one action")
	}
	if plan.property == "" {
		return nil, compileError(ErrorInvalidIntent, root.source, "scenario requires one property")
	}
	return plan, nil
}

func normalizeNode(node Term, plan *normalizedPlan, suffix string) (normalizedGroup, error) {
	switch node.kind {
	case nodeAction:
		identifier := node.action.identifier + suffix
		action := &normalizedAction{
			identifier:      identifier,
			kind:            node.action.kind,
			allowedOutcomes: append([]protocolexperiment.ActionOutcome(nil), node.action.allowedOutcomes...),
			arguments:       append([]protocolexperiment.NamedValue(nil), node.action.arguments...),
			source:          node.source,
			responseMode:    node.action.responseMode,
			maxBlockNanos:   node.action.maxBlockNanos,
		}
		plan.actions = append(plan.actions, action)
		return normalizedGroup{entries: []string{identifier}, exits: []string{identifier}, actions: []string{identifier}}, nil
	case nodeBind:
		intent := node.bind
		intent.projection.ProducerAction += suffix
		plan.bindings = append(plan.bindings, bindIntentWithSource{intent: intent, source: node.source})
		return normalizedGroup{}, nil
	case nodeRequire:
		if plan.property != "" && plan.property != node.property {
			return normalizedGroup{}, compileError(ErrorInvalidIntent, node.source,
				fmt.Sprintf("scenario requires conflicting properties %q and %q", plan.property, node.property))
		}
		plan.property = node.property
		return normalizedGroup{}, nil
	case nodeOnePath, nodeAllPaths, nodeBefore:
		if node.kind == nodeAllPaths {
			plan.allPaths = true
		}
		if node.kind == nodeBefore && len(node.children) != 2 {
			return normalizedGroup{}, compileError(ErrorInvalidIntent, node.source, "Before requires two terms")
		}
		return normalizeSequence(node.children, plan, suffix)
	case nodeAnyOrder:
		var result normalizedGroup
		for _, child := range node.children {
			group, err := normalizeNode(child, plan, suffix)
			if err != nil {
				return normalizedGroup{}, err
			}
			result.entries = append(result.entries, group.entries...)
			result.exits = append(result.exits, group.exits...)
			result.actions = append(result.actions, group.actions...)
		}
		return result, nil
	case nodeDuring:
		if len(node.children) != 1 || node.fault.identifier == "" || node.fault.kind == "" {
			return normalizedGroup{}, compileError(ErrorInvalidIntent, node.source, "During requires one fault and one body")
		}
		group, err := normalizeNode(node.children[0], plan, suffix)
		if err != nil {
			return normalizedGroup{}, err
		}
		if len(group.actions) == 0 {
			return normalizedGroup{}, compileError(ErrorInvalidIntent, node.source, "During body requires at least one action")
		}
		policyID := "during-" + node.fault.identifier + suffix
		plan.policies = append(plan.policies, normalizedPolicy{identifier: policyID, scope: append([]string(nil), group.actions...)})
		plan.faults = append(plan.faults, normalizedFault{
			identifier: node.fault.identifier + suffix,
			kind:       node.fault.kind,
			policy:     policyID,
			arguments:  append([]protocolexperiment.NamedValue(nil), node.fault.arguments...),
			scope:      cloneFaultScope(node.fault.scope),
			occurrence: node.fault.occurrence,
			configured: node.fault.configured,
		})
		return group, nil
	case nodeRepeat:
		if len(node.children) != 1 || node.repeatCount <= 0 {
			return normalizedGroup{}, compileError(ErrorInvalidIntent, node.source, "Repeat requires a positive count and one body")
		}
		children := make([]Term, node.repeatCount)
		for index := range children {
			children[index] = node.children[0]
		}
		var result normalizedGroup
		for index, child := range children {
			group, err := normalizeNode(child, plan, fmt.Sprintf("%s#%d", suffix, index+1))
			if err != nil {
				return normalizedGroup{}, err
			}
			if len(result.exits) != 0 && len(group.entries) != 0 {
				addCrossEdges(plan, result.exits, group.entries, protocolexperiment.OrderUser)
			}
			if len(result.entries) == 0 {
				result.entries = append(result.entries, group.entries...)
			}
			if len(group.exits) != 0 {
				result.exits = append([]string(nil), group.exits...)
			}
			result.actions = append(result.actions, group.actions...)
		}
		return result, nil
	default:
		return normalizedGroup{}, compileError(ErrorInvalidIntent, node.source, "unknown scenario term")
	}
}

func cloneFaultScope(scope protocolexperiment.FaultScope) protocolexperiment.FaultScope {
	scope.Resources = append([]string(nil), scope.Resources...)
	scope.Endpoints = append([]string(nil), scope.Endpoints...)
	scope.TaskQueues = append([]string(nil), scope.TaskQueues...)
	scope.Services = append([]string(nil), scope.Services...)
	scope.Routes = append([]string(nil), scope.Routes...)
	scope.Participants = append([]string(nil), scope.Participants...)
	scope.Attempts = append([]int(nil), scope.Attempts...)
	return scope
}

func normalizeSequence(children []Term, plan *normalizedPlan, suffix string) (normalizedGroup, error) {
	var result normalizedGroup
	var previousExits []string
	for _, child := range children {
		group, err := normalizeNode(child, plan, suffix)
		if err != nil {
			return normalizedGroup{}, err
		}
		if len(previousExits) != 0 && len(group.entries) != 0 {
			addCrossEdges(plan, previousExits, group.entries, protocolexperiment.OrderUser)
		}
		if len(result.entries) == 0 && len(group.entries) != 0 {
			result.entries = append(result.entries, group.entries...)
		}
		if len(group.exits) != 0 {
			previousExits = append([]string(nil), group.exits...)
			result.exits = append([]string(nil), group.exits...)
		}
		result.actions = append(result.actions, group.actions...)
	}
	return result, nil
}

func addCrossEdges(plan *normalizedPlan, before, after []string, relation protocolexperiment.OrderRelation) {
	for _, left := range before {
		for _, right := range after {
			plan.edges = append(plan.edges, normalizedEdge{before: left, after: right, relation: relation})
		}
	}
}

func sortAndCompactEdges(edges []normalizedEdge) []normalizedEdge {
	slices.SortFunc(edges, func(left, right normalizedEdge) int {
		if result := stringCompare(left.before, right.before); result != 0 {
			return result
		}
		if result := stringCompare(left.after, right.after); result != 0 {
			return result
		}
		return stringCompare(string(left.relation), string(right.relation))
	})
	return slices.Compact(edges)
}

func stringCompare(left, right string) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func compileError(category ErrorCategory, source Source, detail string) *Error {
	return &Error{Category: category, Source: source, Detail: detail}
}
