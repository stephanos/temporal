package observation

import (
	"fmt"
	"slices"

	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolmonitor "go.temporal.io/server/tools/umpire3/protocol/monitor"
)

func EvaluateMonitor(
	program protocolmonitor.MonitorProgram,
	facts []protocolmonitor.ObservedFact,
) (protocolmonitor.MonitorEvaluation, error) {
	values := make(map[protocolcatalog.ObservationID]bool, len(facts))
	for _, fact := range facts {
		if existing, exists := values[fact.Observation]; exists && existing != fact.Value {
			return protocolmonitor.MonitorEvaluation{}, fmt.Errorf("conflicting observation %q", fact.Observation)
		}
		values[fact.Observation] = fact.Value
	}
	satisfied, missing := evaluateMonitorExpression(program.Expression, values)
	missingValues := make([]protocolcatalog.ObservationID, 0, len(missing))
	for identifier := range missing {
		missingValues = append(missingValues, identifier)
	}
	slices.Sort(missingValues)
	evaluation := protocolmonitor.MonitorEvaluation{
		Complete: len(missingValues) == 0, Satisfied: satisfied, Missing: missingValues,
	}
	if evaluation.Complete && !evaluation.Satisfied {
		evaluation.Contradictions = monitorContradictions(program.Expression, values)
		slices.Sort(evaluation.Contradictions)
		evaluation.Contradictions = slices.Compact(evaluation.Contradictions)
	}
	return evaluation, nil
}

func evaluateMonitorExpression(
	expression protocolmonitor.MonitorExpression,
	values map[protocolcatalog.ObservationID]bool,
) (bool, map[protocolcatalog.ObservationID]struct{}) {
	missing := make(map[protocolcatalog.ObservationID]struct{})
	switch expression.Operation {
	case protocolmonitor.MonitorObservation:
		value, exists := values[expression.Observation]
		if !exists {
			missing[expression.Observation] = struct{}{}
			return false, missing
		}
		return value == *expression.Expected, missing
	case protocolmonitor.MonitorAll:
		result := true
		for _, child := range expression.Children {
			value, childMissing := evaluateMonitorExpression(child, values)
			result = result && value
			mergeMonitorMissing(missing, childMissing)
		}
		return result, missing
	case protocolmonitor.MonitorAny:
		result := false
		for _, child := range expression.Children {
			value, childMissing := evaluateMonitorExpression(child, values)
			result = result || value
			mergeMonitorMissing(missing, childMissing)
		}
		return result, missing
	case protocolmonitor.MonitorNot:
		value, childMissing := evaluateMonitorExpression(expression.Children[0], values)
		return !value, childMissing
	case protocolmonitor.MonitorImplies:
		premise, premiseMissing := evaluateMonitorExpression(expression.Children[0], values)
		conclusion, conclusionMissing := evaluateMonitorExpression(expression.Children[1], values)
		mergeMonitorMissing(premiseMissing, conclusionMissing)
		return !premise || conclusion, premiseMissing
	default:
		return false, missing
	}
}

func mergeMonitorMissing(
	target map[protocolcatalog.ObservationID]struct{},
	source map[protocolcatalog.ObservationID]struct{},
) {
	for identifier := range source {
		target[identifier] = struct{}{}
	}
}

func monitorContradictions(
	expression protocolmonitor.MonitorExpression,
	values map[protocolcatalog.ObservationID]bool,
) []protocolcatalog.ObservationID {
	switch expression.Operation {
	case protocolmonitor.MonitorObservation:
		if value, exists := values[expression.Observation]; exists && value != *expression.Expected {
			return []protocolcatalog.ObservationID{expression.Observation}
		}
	case protocolmonitor.MonitorAll:
		var result []protocolcatalog.ObservationID
		for _, child := range expression.Children {
			value, missing := evaluateMonitorExpression(child, values)
			if !value && len(missing) == 0 {
				result = append(result, monitorContradictions(child, values)...)
			}
		}
		return result
	case protocolmonitor.MonitorAny:
		var result []protocolcatalog.ObservationID
		for _, child := range expression.Children {
			result = append(result, monitorContradictions(child, values)...)
		}
		return result
	case protocolmonitor.MonitorNot:
		return monitorObservations(expression.Children[0])
	case protocolmonitor.MonitorImplies:
		premise, _ := evaluateMonitorExpression(expression.Children[0], values)
		conclusion, _ := evaluateMonitorExpression(expression.Children[1], values)
		if premise && !conclusion {
			return monitorContradictions(expression.Children[1], values)
		}
	default:
	}
	return nil
}

func monitorObservations(expression protocolmonitor.MonitorExpression) []protocolcatalog.ObservationID {
	if expression.Operation == protocolmonitor.MonitorObservation {
		return []protocolcatalog.ObservationID{expression.Observation}
	}
	var result []protocolcatalog.ObservationID
	for _, child := range expression.Children {
		result = append(result, monitorObservations(child)...)
	}
	return result
}
