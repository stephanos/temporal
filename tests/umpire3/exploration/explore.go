package exploration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
	"go.temporal.io/server/tests/umpire3/scenario"
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
	StatusAssignmentsEnumerated  Status = "assignments-enumerated"
	StatusAssignmentLimitReached Status = "assignment-limit-reached"
)

type CoverageStatus string

const (
	CoverageNotRequested CoverageStatus = "coverage-not-requested"
	CoverageUndefined    CoverageStatus = "coverage-undefined"
	CoverageUncovered    CoverageStatus = "coverage-uncovered"
	CoverageCovered      CoverageStatus = "coverage-covered"
)

type Value struct {
	Key      string                     `json:"key"`
	Action   protocolcatalog.ActionKind `json:"action,omitempty"`
	Integer  int                        `json:"integer,omitempty"`
	Text     string                     `json:"text,omitempty"`
	Coverage []string                   `json:"coverage,omitempty"`
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
	Kind     GoalKind                   `json:"kind"`
	Property protocolcatalog.PropertyID `json:"property,omitempty"`
	Target   protocolcatalog.TargetID   `json:"target,omitempty"`
}

type Assignment map[string]Value

type Template struct {
	Identifier     string
	Goal           Goal
	Holes          []Hole
	Required       []Fragment
	Forbidden      []Fragment
	SymmetryGroups [][]string
	Build          func(Assignment) (scenario.Scenario, error)
	Observe        func(context.Context, Candidate) ([]string, error)
}

type Bounds struct {
	MaxAssignments int
	Compiler       scenario.Limits
}

type Candidate struct {
	Assignment Assignment                    `json:"assignment"`
	Experiment protocolexperiment.Experiment `json:"experiment"`
	Digest     string                        `json:"digest"`
	Coverage   []string                      `json:"coverage,omitempty"`
}

type Pruned struct {
	Required  int `json:"required"`
	Forbidden int `json:"forbidden"`
	Symmetry  int `json:"symmetry"`
	Invalid   int `json:"invalid"`
	Duplicate int `json:"duplicate"`
}

type Report struct {
	Template   string              `json:"template"`
	Status     Status              `json:"status"`
	Explored   int                 `json:"explored"`
	Pruned     Pruned              `json:"pruned"`
	Omissions  []string            `json:"omissions"`
	Candidates []Candidate         `json:"candidates"`
	Coverage   Coverage            `json:"coverage"`
	Reductions []ReductionEvidence `json:"reductions"`
}

type ReductionKind string

const ReductionSymmetry ReductionKind = "symmetry"

type ReductionStatus string

const ReductionCheckedCertificate ReductionStatus = "checked-certificate"

type ReductionEvidence struct {
	Kind               ReductionKind   `json:"kind"`
	Status             ReductionStatus `json:"status"`
	CertificateDigest  string          `json:"certificateDigest"`
	CheckedAssignments int             `json:"checkedAssignments"`
}

type Coverage struct {
	Target    protocolcatalog.TargetID   `json:"target,omitempty"`
	Property  protocolcatalog.PropertyID `json:"property,omitempty"`
	Status    CoverageStatus             `json:"status"`
	Reason    string                     `json:"reason,omitempty"`
	Total     int                        `json:"total"`
	Covered   []string                   `json:"covered"`
	Uncovered []string                   `json:"uncovered"`
}

func NexusLifecycleValues() ([]Value, error) {
	denominator, err := protocolcatalog.DefaultCoverageDenominator()
	if err != nil {
		return nil, err
	}
	for _, target := range denominator.Targets {
		if target.Identifier != protocolcatalog.TargetIDFeatureNexus ||
			target.Property != protocolcatalog.PropertyIDNexusOperationClosure {
			continue
		}
		if target.Status != protocolcatalog.CoverageDenominatorDefined {
			return nil, errors.New("generated Nexus lifecycle coverage denominator is undefined")
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
	reductions, err := checkReductions(ctx, template, bounds)
	if err != nil {
		return Report{}, err
	}
	report := Report{Template: template.Identifier, Status: StatusAssignmentsEnumerated,
		Reductions: reductions}
	denominator, err := coverageDenominator(template.Goal)
	if err != nil {
		return Report{}, err
	}
	coverageStatus := CoverageNotRequested
	if denominator.requested {
		coverageStatus = CoverageUndefined
		if denominator.defined {
			coverageStatus = CoverageUncovered
		}
	}
	report.Coverage = Coverage{
		Target: template.Goal.Target, Property: template.Goal.Property, Status: coverageStatus,
		Reason: denominator.reason, Total: len(denominator.identifiers),
	}
	covered := make(map[string]struct{}, len(denominator.identifiers))
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
		authored, err := template.Build(cloneAssignment(assignment))
		if err != nil {
			report.Pruned.Invalid++
			return nil
		}
		suite, err := scenario.Compile(ctx, authored, bounds.Compiler)
		if err != nil {
			var compileErr *scenario.Error
			if errors.As(err, &compileErr) {
				report.Pruned.Invalid++
				return nil
			}
			return err
		}
		candidateCoverage := assignmentCoverage(assignment)
		for experimentIndex, experiment := range suite.Experiments {
			if template.Goal.Property != "" && protocolcatalog.PropertyID(experiment.Property.Identifier) != template.Goal.Property {
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
		report.Status = StatusAssignmentLimitReached
		report.Omissions = append(report.Omissions,
			fmt.Sprintf("assignment limit %d exhausted before enumeration completed", bounds.MaxAssignments))
	}
	slices.SortFunc(report.Candidates, func(left, right Candidate) int { return compare(left.Digest, right.Digest) })
	for _, identifier := range denominator.identifiers {
		if _, exists := covered[identifier]; exists {
			report.Coverage.Covered = append(report.Coverage.Covered, identifier)
		} else {
			report.Coverage.Uncovered = append(report.Coverage.Uncovered, identifier)
		}
	}
	if denominator.defined && len(report.Coverage.Uncovered) == 0 {
		report.Coverage.Status = CoverageCovered
	}
	return report, nil
}

type coverageSelection struct {
	requested   bool
	defined     bool
	reason      string
	identifiers []string
}

func coverageDenominator(goal Goal) (coverageSelection, error) {
	if goal.Kind != GoalTransitionCoverage {
		return coverageSelection{}, nil
	}
	if goal.Target == "" || goal.Property == "" {
		return coverageSelection{}, errors.New("transition coverage requires a model target and property")
	}
	denominator, err := protocolcatalog.DefaultCoverageDenominator()
	if err != nil {
		return coverageSelection{}, fmt.Errorf("load model coverage denominator: %w", err)
	}
	for _, target := range denominator.Targets {
		if target.Identifier != goal.Target || target.Property != goal.Property {
			continue
		}
		if target.Status == protocolcatalog.CoverageDenominatorUndefined {
			return coverageSelection{requested: true, reason: target.Reason}, nil
		}
		identifiers := make([]string, 0, len(target.Points))
		if len(target.Edges) != 0 {
			for _, edge := range target.Edges {
				identifiers = append(identifiers, edge.Identifier)
			}
		} else {
			for _, point := range target.Points {
				if point.Dimension == protocolcatalog.CoverageTransition {
					identifiers = append(identifiers, point.Identifier)
				}
			}
		}
		slices.Sort(identifiers)
		return coverageSelection{requested: true, defined: true, identifiers: identifiers}, nil
	}
	return coverageSelection{}, fmt.Errorf("model coverage denominator has no target %q property %q", goal.Target, goal.Property)
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
	knownCoverage := make(map[string]struct{}, len(denominator.identifiers))
	for _, identifier := range denominator.identifiers {
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

func checkReductions(ctx context.Context, template Template, bounds Bounds) ([]ReductionEvidence, error) {
	if len(template.SymmetryGroups) == 0 {
		return []ReductionEvidence{}, nil
	}
	holes := make(map[string]Hole, len(template.Holes))
	for _, hole := range template.Holes {
		holes[hole.Identifier] = hole
	}
	for _, group := range template.SymmetryGroups {
		expected := valueKeys(holes[group[0]].Values)
		for _, identifier := range group[1:] {
			if !slices.Equal(expected, valueKeys(holes[identifier].Values)) {
				return nil, fmt.Errorf("symmetry group %v requires identical finite domains", group)
			}
		}
	}
	ordered := append([]Hole(nil), template.Holes...)
	slices.SortFunc(ordered, func(left, right Hole) int { return compare(left.Identifier, right.Identifier) })
	certificate := make([]string, 0)
	checked := 0
	visited := 0
	var enumerate func(int, Assignment) error
	enumerate = func(index int, assignment Assignment) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if index < len(ordered) {
			for _, value := range ordered[index].Values {
				next := cloneAssignment(assignment)
				next[ordered[index].Identifier] = value
				if err := enumerate(index+1, next); err != nil {
					return err
				}
			}
			return nil
		}
		visited++
		if visited > bounds.MaxAssignments {
			return fmt.Errorf("symmetry certificate exceeds assignment limit %d", bounds.MaxAssignments)
		}
		if !includesAll(assignment, template.Required) || includesAny(assignment, template.Forbidden) ||
			canonicalSymmetry(assignment, template.SymmetryGroups) {
			return nil
		}
		canonical := canonicalizeSymmetry(assignment, template.SymmetryGroups)
		if !includesAll(canonical, template.Required) || includesAny(canonical, template.Forbidden) {
			return errors.New("symmetry canonicalization changes template constraints")
		}
		left, leftErr := compileAssignment(ctx, template, bounds, assignment)
		right, rightErr := compileAssignment(ctx, template, bounds, canonical)
		if (leftErr == nil) != (rightErr == nil) {
			return fmt.Errorf("symmetry assignment changes compiler validity: %v / %v", leftErr, rightErr)
		}
		if leftErr != nil {
			return nil
		}
		if !slices.Equal(left, right) {
			return errors.New("symmetry assignment changes compiled semantics")
		}
		checked++
		certificate = append(certificate, string(left))
		return nil
	}
	if err := enumerate(0, make(Assignment)); err != nil {
		return nil, fmt.Errorf("check symmetry preservation: %w", err)
	}
	encoded, err := json.Marshal(struct {
		Identifier string
		Groups     [][]string
		Witnesses  []string
	}{Identifier: template.Identifier, Groups: template.SymmetryGroups, Witnesses: certificate})
	if err != nil {
		return nil, fmt.Errorf("encode symmetry certificate: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return []ReductionEvidence{{
		Kind: ReductionSymmetry, Status: ReductionCheckedCertificate,
		CertificateDigest: "sha256:" + hex.EncodeToString(digest[:]), CheckedAssignments: checked,
	}}, nil
}

func valueKeys(values []Value) []string {
	keys := make([]string, len(values))
	for index, value := range values {
		keys[index] = value.Key
	}
	slices.Sort(keys)
	return keys
}

func canonicalizeSymmetry(assignment Assignment, groups [][]string) Assignment {
	result := cloneAssignment(assignment)
	for _, group := range groups {
		values := make([]Value, len(group))
		for index, identifier := range group {
			values[index] = result[identifier]
		}
		slices.SortFunc(values, func(left, right Value) int { return compare(left.Key, right.Key) })
		for index, identifier := range group {
			result[identifier] = values[index]
		}
	}
	return result
}

func compileAssignment(
	ctx context.Context,
	template Template,
	bounds Bounds,
	assignment Assignment,
) ([]byte, error) {
	authored, err := template.Build(cloneAssignment(assignment))
	if err != nil {
		return nil, err
	}
	suite, err := scenario.Compile(ctx, authored, bounds.Compiler)
	if err != nil {
		return nil, err
	}
	experiments := append([]protocolexperiment.Experiment(nil), suite.Experiments...)
	for index := range experiments {
		experiments[index].ExperimentID = "symmetry-canonical"
	}
	return json.Marshal(experiments)
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
