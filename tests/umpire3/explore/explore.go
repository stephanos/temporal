package explore

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"go.temporal.io/server/tests/umpire3/compiler"
	"go.temporal.io/server/tests/umpire3/protocol"
)

type HoleKind string

const (
	HoleAction      HoleKind = "action"
	HoleEntityCount HoleKind = "entity-count"
	HoleTopology    HoleKind = "topology"
	HoleParameter   HoleKind = "parameter"
	HoleSchedule    HoleKind = "schedule"
	HoleFault       HoleKind = "fault"
	HoleCheckpoint  HoleKind = "checkpoint"
)

type GoalKind string

const (
	GoalReachability       GoalKind = "reachability"
	GoalChallengeSafety    GoalKind = "challenge-safety"
	GoalTransitionCoverage GoalKind = "transition-coverage"
	GoalRelationCoverage   GoalKind = "relation-coverage"
)

type Status string

const (
	StatusExhaustive      Status = "exhaustive"
	StatusResourceLimited Status = "resource-limited"
)

type Value struct {
	Key      string              `json:"key"`
	Action   protocol.ActionKind `json:"action,omitempty"`
	Integer  int                 `json:"integer,omitempty"`
	Text     string              `json:"text,omitempty"`
	Coverage []string            `json:"coverage,omitempty"`
}

type Hole struct {
	Identifier string   `json:"identifier"`
	Kind       HoleKind `json:"kind"`
	Values     []Value  `json:"values"`
}

type Fragment struct {
	Hole  string `json:"hole"`
	Value string `json:"value"`
}

type Goal struct {
	Kind     GoalKind            `json:"kind"`
	Property protocol.PropertyID `json:"property,omitempty"`
	Target   protocol.TargetID   `json:"target,omitempty"`
}

type Assignment map[string]Value

type Template struct {
	Identifier                      string
	Goal                            Goal
	Holes                           []Hole
	Required                        []Fragment
	Forbidden                       []Fragment
	SymmetryGroups                  [][]string
	SymmetryPreservationChecked     bool
	PartialOrderReduction           bool
	PartialOrderPreservationChecked bool
	Build                           func(Assignment) (compiler.Scenario, error)
	Observe                         func(context.Context, Candidate) ([]string, error)
}

type Bounds struct {
	MaxAssignments int
	Compiler       compiler.Limits
}

type Candidate struct {
	Assignment Assignment          `json:"assignment"`
	Experiment protocol.Experiment `json:"experiment"`
	Digest     string              `json:"digest"`
	Coverage   []string            `json:"coverage,omitempty"`
}

type Pruned struct {
	Required  int `json:"required"`
	Forbidden int `json:"forbidden"`
	Symmetry  int `json:"symmetry"`
	Invalid   int `json:"invalid"`
	Duplicate int `json:"duplicate"`
}

type Report struct {
	Template   string      `json:"template"`
	Status     Status      `json:"status"`
	Complete   bool        `json:"complete"`
	Explored   int         `json:"explored"`
	Pruned     Pruned      `json:"pruned"`
	Omissions  []string    `json:"omissions"`
	Candidates []Candidate `json:"candidates"`
	Coverage   Coverage    `json:"coverage"`
}

type Coverage struct {
	Target    protocol.TargetID   `json:"target,omitempty"`
	Property  protocol.PropertyID `json:"property,omitempty"`
	Total     int                 `json:"total"`
	Covered   []string            `json:"covered"`
	Uncovered []string            `json:"uncovered"`
	Complete  bool                `json:"complete"`
}

func NexusLifecycleValues() ([]Value, error) {
	denominator, err := protocol.DefaultCoverageDenominator()
	if err != nil {
		return nil, err
	}
	for _, target := range denominator.Targets {
		if target.Identifier != protocol.TargetIDFeatureNexus ||
			target.Property != protocol.PropertyIDNexusOperationClosure {
			continue
		}
		values := make([]Value, len(target.Edges))
		for index, edge := range target.Edges {
			values[index] = Value{Key: edge.Identifier, Text: edge.Action, Coverage: []string{edge.Identifier}}
		}
		slices.SortFunc(values, func(left, right Value) int { return compare(left.Key, right.Key) })
		return values, nil
	}
	return nil, errors.New("generated Nexus lifecycle coverage denominator is unavailable")
}

func Run(ctx context.Context, template Template, bounds Bounds) (Report, error) {
	if err := validateTemplate(template, bounds); err != nil {
		return Report{}, err
	}
	report := Report{Template: template.Identifier, Status: StatusExhaustive, Complete: true}
	denominator, err := coverageDenominator(template.Goal)
	if err != nil {
		return Report{}, err
	}
	report.Coverage = Coverage{
		Target: template.Goal.Target, Property: template.Goal.Property, Total: len(denominator),
	}
	covered := make(map[string]struct{}, len(denominator))
	holes := append([]Hole(nil), template.Holes...)
	slices.SortFunc(holes, func(left, right Hole) int { return compare(left.Identifier, right.Identifier) })
	seenDigests := make(map[string]struct{})
	limited := false

	var enumerate func(int, Assignment) error
	enumerate = func(index int, assignment Assignment) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if index != len(holes) {
			hole := holes[index]
			values := append([]Value(nil), hole.Values...)
			slices.SortFunc(values, func(left, right Value) int { return compare(left.Key, right.Key) })
			for _, value := range values {
				next := cloneAssignment(assignment)
				next[hole.Identifier] = value
				if err := enumerate(index+1, next); err != nil {
					return err
				}
				if limited {
					return nil
				}
			}
			return nil
		}
		if !includesAll(assignment, template.Required) {
			report.Pruned.Required++
			return nil
		}
		if includesAny(assignment, template.Forbidden) {
			report.Pruned.Forbidden++
			return nil
		}
		if !canonicalSymmetry(assignment, template.SymmetryGroups) {
			report.Pruned.Symmetry++
			return nil
		}
		if report.Explored == bounds.MaxAssignments {
			limited = true
			return nil
		}
		report.Explored++
		scenario, err := template.Build(cloneAssignment(assignment))
		if err != nil {
			report.Pruned.Invalid++
			return nil
		}
		suite, err := compiler.Compile(ctx, scenario, bounds.Compiler)
		if err != nil {
			var compileErr *compiler.Error
			if errors.As(err, &compileErr) {
				report.Pruned.Invalid++
				return nil
			}
			return err
		}
		candidateCoverage := assignmentCoverage(assignment)
		for experimentIndex, experiment := range suite.Experiments {
			if template.Goal.Property != "" && protocol.PropertyID(experiment.Property.Identifier) != template.Goal.Property {
				return fmt.Errorf("template goal property %q does not match built property %q",
					template.Goal.Property, experiment.Property.Identifier)
			}
			digest := suite.Digests[experimentIndex]
			if _, duplicate := seenDigests[digest]; duplicate {
				report.Pruned.Duplicate++
				continue
			}
			seenDigests[digest] = struct{}{}
			candidate := Candidate{
				Assignment: cloneAssignment(assignment), Experiment: experiment, Digest: digest,
				Coverage: append([]string(nil), candidateCoverage...),
			}
			report.Candidates = append(report.Candidates, candidate)
			observedCoverage := candidateCoverage
			if template.Goal.Kind == GoalTransitionCoverage {
				observedCoverage, err = template.Observe(ctx, candidate)
				if err != nil {
					return fmt.Errorf("observe candidate %q: %w", digest, err)
				}
				if err := validateObservedCoverage(observedCoverage, candidateCoverage); err != nil {
					return fmt.Errorf("observe candidate %q: %w", digest, err)
				}
			}
			for _, identifier := range observedCoverage {
				covered[identifier] = struct{}{}
			}
		}
		return nil
	}
	if err := enumerate(0, make(Assignment)); err != nil {
		return Report{}, fmt.Errorf("explore template: %w", err)
	}
	if limited {
		report.Status = StatusResourceLimited
		report.Complete = false
		report.Omissions = append(report.Omissions,
			fmt.Sprintf("assignment limit %d exhausted before enumeration completed", bounds.MaxAssignments))
	}
	slices.SortFunc(report.Candidates, func(left, right Candidate) int { return compare(left.Digest, right.Digest) })
	for _, identifier := range denominator {
		if _, exists := covered[identifier]; exists {
			report.Coverage.Covered = append(report.Coverage.Covered, identifier)
		} else {
			report.Coverage.Uncovered = append(report.Coverage.Uncovered, identifier)
		}
	}
	report.Coverage.Complete = len(denominator) != 0 && len(report.Coverage.Uncovered) == 0
	return report, nil
}

func coverageDenominator(goal Goal) ([]string, error) {
	if goal.Kind != GoalTransitionCoverage {
		return nil, nil
	}
	if goal.Target == "" || goal.Property == "" {
		return nil, errors.New("transition coverage requires a model target and property")
	}
	denominator, err := protocol.DefaultCoverageDenominator()
	if err != nil {
		return nil, fmt.Errorf("load model coverage denominator: %w", err)
	}
	for _, target := range denominator.Targets {
		if target.Identifier != goal.Target || target.Property != goal.Property {
			continue
		}
		identifiers := make([]string, len(target.Edges))
		for index, edge := range target.Edges {
			identifiers[index] = edge.Identifier
		}
		slices.Sort(identifiers)
		return identifiers, nil
	}
	return nil, fmt.Errorf("model coverage denominator has no target %q property %q", goal.Target, goal.Property)
}

func assignmentCoverage(assignment Assignment) []string {
	var result []string
	for _, value := range assignment {
		result = append(result, value.Coverage...)
	}
	slices.Sort(result)
	return slices.Compact(result)
}

func validateTemplate(template Template, bounds Bounds) error {
	if template.Identifier == "" || template.Build == nil || len(template.Holes) == 0 {
		return errors.New("template identifier, typed holes, and builder are required")
	}
	switch template.Goal.Kind {
	case GoalReachability, GoalChallengeSafety, GoalTransitionCoverage, GoalRelationCoverage:
	default:
		return fmt.Errorf("unknown exploration goal %q", template.Goal.Kind)
	}
	if bounds.MaxAssignments <= 0 {
		return errors.New("positive assignment bound is required")
	}
	denominator, err := coverageDenominator(template.Goal)
	if err != nil {
		return err
	}
	if template.Goal.Kind == GoalTransitionCoverage && template.Observe == nil {
		return errors.New("transition coverage requires positive runtime observation")
	}
	knownCoverage := make(map[string]struct{}, len(denominator))
	for _, identifier := range denominator {
		knownCoverage[identifier] = struct{}{}
	}
	holes := make(map[string]map[string]struct{}, len(template.Holes))
	for _, hole := range template.Holes {
		if hole.Identifier == "" || len(hole.Values) == 0 {
			return errors.New("every typed hole requires an identifier and finite domain")
		}
		switch hole.Kind {
		case HoleAction, HoleEntityCount, HoleTopology, HoleParameter, HoleSchedule, HoleFault, HoleCheckpoint:
		default:
			return fmt.Errorf("hole %q has unknown kind %q", hole.Identifier, hole.Kind)
		}
		if _, duplicate := holes[hole.Identifier]; duplicate {
			return fmt.Errorf("duplicate hole %q", hole.Identifier)
		}
		values := make(map[string]struct{}, len(hole.Values))
		for _, value := range hole.Values {
			if value.Key == "" {
				return fmt.Errorf("hole %q has an empty domain key", hole.Identifier)
			}
			if _, duplicate := values[value.Key]; duplicate {
				return fmt.Errorf("hole %q has duplicate domain key %q", hole.Identifier, value.Key)
			}
			values[value.Key] = struct{}{}
			for _, identifier := range value.Coverage {
				if _, known := knownCoverage[identifier]; !known {
					return fmt.Errorf("hole %q value %q references unknown coverage edge %q",
						hole.Identifier, value.Key, identifier)
				}
			}
		}
		holes[hole.Identifier] = values
	}
	for _, fragment := range append(append([]Fragment(nil), template.Required...), template.Forbidden...) {
		values, known := holes[fragment.Hole]
		if !known {
			return fmt.Errorf("constraint references unknown hole %q", fragment.Hole)
		}
		if _, known := values[fragment.Value]; !known {
			return fmt.Errorf("constraint references unknown value %q for hole %q", fragment.Value, fragment.Hole)
		}
	}
	if len(template.SymmetryGroups) != 0 && !template.SymmetryPreservationChecked {
		return errors.New("symmetry reduction requires a checked preservation condition")
	}
	if template.PartialOrderReduction && !template.PartialOrderPreservationChecked {
		return errors.New("partial-order reduction requires a checked preservation condition")
	}
	for _, group := range template.SymmetryGroups {
		if len(group) < 2 {
			return errors.New("symmetry group requires at least two holes")
		}
		for _, hole := range group {
			if _, known := holes[hole]; !known {
				return fmt.Errorf("symmetry group references unknown hole %q", hole)
			}
		}
	}
	return nil
}

func validateObservedCoverage(observed []string, declared []string) error {
	declaredSet := make(map[string]struct{}, len(declared))
	for _, identifier := range declared {
		declaredSet[identifier] = struct{}{}
	}
	seen := make(map[string]struct{}, len(observed))
	for _, identifier := range observed {
		if _, ok := declaredSet[identifier]; !ok {
			return fmt.Errorf("runtime reported undeclared coverage %q", identifier)
		}
		if _, duplicate := seen[identifier]; duplicate {
			return fmt.Errorf("runtime reported duplicate coverage %q", identifier)
		}
		seen[identifier] = struct{}{}
	}
	return nil
}

func includesAll(assignment Assignment, fragments []Fragment) bool {
	for _, fragment := range fragments {
		if assignment[fragment.Hole].Key != fragment.Value {
			return false
		}
	}
	return true
}

func includesAny(assignment Assignment, fragments []Fragment) bool {
	for _, fragment := range fragments {
		if assignment[fragment.Hole].Key == fragment.Value {
			return true
		}
	}
	return false
}

func canonicalSymmetry(assignment Assignment, groups [][]string) bool {
	for _, group := range groups {
		for index := 1; index < len(group); index++ {
			if assignment[group[index-1]].Key > assignment[group[index]].Key {
				return false
			}
		}
	}
	return true
}

func cloneAssignment(source Assignment) Assignment {
	result := make(Assignment, len(source))
	for identifier, value := range source {
		result[identifier] = value
	}
	return result
}

func compare(left, right string) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
