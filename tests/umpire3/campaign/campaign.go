package campaign

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"go/format"
	"slices"
	"strings"
	"sync"

	"go.temporal.io/server/tests/umpire3/compiler"
	"go.temporal.io/server/tests/umpire3/protocol"
	umpire3runtime "go.temporal.io/server/tests/umpire3/runtime"
)

type CoverageKind string

const (
	CoverageTransition CoverageKind = "transition"
	CoverageProperty   CoverageKind = "property"
	CoverageRelation   CoverageKind = "relation"
	CoverageRefinement CoverageKind = "refinement"
	CoverageEvidence   CoverageKind = "evidence"
	CoverageProtobuf   CoverageKind = "protobuf-field-class"
	CoverageFault      CoverageKind = "fault"
	CoverageSchedule   CoverageKind = "schedule"
	CoverageTopology   CoverageKind = "topology"
	CoverageProfile    CoverageKind = "profile"
	CoverageAction     CoverageKind = "action"
)

type DropReason string

const (
	DropBudget      DropReason = "budget"
	DropDuplicate   DropReason = "duplicate"
	DropUnsupported DropReason = "unsupported"
)

type CoveragePoint struct {
	Kind       CoverageKind `json:"kind"`
	Identifier string       `json:"identifier"`
}

type Candidate struct {
	Identifier string
	Scenario   compiler.Scenario
	Coverage   []CoveragePoint
	Risk       []CoveragePoint
}

type Executor func(context.Context, protocol.Experiment) (umpire3runtime.Result, []CoveragePoint, error)

type Request struct {
	Candidates       []Candidate
	Seed             int64
	Workers          int
	MaxExecutions    int
	MinimizeAttempts int
	CompilerLimits   compiler.Limits
	RiskFocus        []CoveragePoint
	CorpusCoverage   []CoveragePoint
	Executor         Executor
}

type Execution struct {
	CandidateID string                `json:"candidateID"`
	Digest      string                `json:"digest"`
	Result      umpire3runtime.Result `json:"result"`
	Coverage    []CoveragePoint       `json:"coverage"`
}

type Dropped struct {
	CandidateID string     `json:"candidateID"`
	Digest      string     `json:"digest,omitempty"`
	Reason      DropReason `json:"reason"`
	Detail      string     `json:"detail"`
}

type Minimization struct {
	Attempts int  `json:"attempts"`
	Complete bool `json:"complete"`
}

type Promotion struct {
	Source string `json:"source"`
}

type Discovery struct {
	CandidateID    string              `json:"candidateID"`
	Original       protocol.Experiment `json:"original"`
	Minimized      protocol.Experiment `json:"minimized"`
	Minimization   Minimization        `json:"minimization"`
	Promotion      Promotion           `json:"promotion"`
	PromotionBlock string              `json:"promotionBlock,omitempty"`
}

type Report struct {
	CoverageBefore []CoveragePoint `json:"coverageBefore"`
	CoverageAfter  []CoveragePoint `json:"coverageAfter"`
	CoverageDelta  []CoveragePoint `json:"coverageDelta"`
	Executions     []Execution     `json:"executions"`
	Dropped        []Dropped       `json:"dropped"`
	Discoveries    []Discovery     `json:"discoveries"`
}

type rankedExperiment struct {
	candidateID string
	experiment  protocol.Experiment
	digest      string
	score       int
	seedOrder   uint64
}

type executionResult struct {
	ranked   rankedExperiment
	result   umpire3runtime.Result
	coverage []CoveragePoint
	err      error
}

func Run(ctx context.Context, request Request) (Report, error) {
	if request.Executor == nil || request.Workers <= 0 || request.MaxExecutions <= 0 || len(request.Candidates) == 0 {
		return Report{}, errors.New("campaign candidates, executor, workers, and execution budget are required")
	}
	report := Report{CoverageBefore: normalizeCoverage(request.CorpusCoverage)}
	covered := coverageSet(report.CoverageBefore)
	risk := coverageSet(request.RiskFocus)
	seenDigests := make(map[string]struct{})
	var ranked []rankedExperiment
	for _, candidate := range request.Candidates {
		if candidate.Identifier == "" {
			report.Dropped = append(report.Dropped, Dropped{Reason: DropUnsupported, Detail: "candidate identifier is required"})
			continue
		}
		suite, err := compiler.Compile(ctx, candidate.Scenario, request.CompilerLimits)
		if err != nil {
			report.Dropped = append(report.Dropped, Dropped{
				CandidateID: candidate.Identifier, Reason: DropUnsupported, Detail: err.Error(),
			})
			continue
		}
		for index, experiment := range suite.Experiments {
			digest := suite.Digests[index]
			if _, duplicate := seenDigests[digest]; duplicate {
				report.Dropped = append(report.Dropped, Dropped{
					CandidateID: candidate.Identifier, Digest: digest, Reason: DropDuplicate, Detail: "compiled experiment digest already ranked",
				})
				continue
			}
			seenDigests[digest] = struct{}{}
			ranked = append(ranked, rankedExperiment{
				candidateID: candidate.Identifier, experiment: experiment, digest: digest,
				score: campaignScore(candidate, covered, risk), seedOrder: seededOrder(request.Seed, digest),
			})
		}
	}
	slices.SortFunc(ranked, func(left, right rankedExperiment) int {
		if left.score != right.score {
			return right.score - left.score
		}
		if left.seedOrder < right.seedOrder {
			return -1
		}
		if left.seedOrder > right.seedOrder {
			return 1
		}
		return compare(left.digest, right.digest)
	})
	if len(ranked) > request.MaxExecutions {
		for _, candidate := range ranked[request.MaxExecutions:] {
			report.Dropped = append(report.Dropped, Dropped{
				CandidateID: candidate.candidateID, Digest: candidate.digest,
				Reason: DropBudget, Detail: "execution budget exhausted",
			})
		}
		ranked = ranked[:request.MaxExecutions]
	}

	results := executeParallel(ctx, ranked, request.Workers, request.Executor)
	for _, execution := range results {
		if execution.err != nil {
			return Report{}, fmt.Errorf("execute candidate %q: %w", execution.ranked.candidateID, execution.err)
		}
		coverage := normalizeCoverage(execution.coverage)
		report.Executions = append(report.Executions, Execution{
			CandidateID: execution.ranked.candidateID, Digest: execution.ranked.digest,
			Result: execution.result, Coverage: coverage,
		})
		for _, point := range coverage {
			covered[point] = struct{}{}
		}
		if execution.result.Claim.Kind == umpire3runtime.ClaimViolating {
			report.Discoveries = append(report.Discoveries,
				minimizeDiscovery(ctx, execution, request.MinimizeAttempts, request.Executor))
		}
	}
	report.CoverageAfter = coverageSlice(covered)
	report.CoverageDelta = coverageDifference(report.CoverageBefore, report.CoverageAfter)
	sortReport(&report)
	return report, nil
}

func executeParallel(ctx context.Context, ranked []rankedExperiment, workers int, executor Executor) []executionResult {
	jobs := make(chan rankedExperiment)
	results := make(chan executionResult, len(ranked))
	var group sync.WaitGroup
	for range min(workers, len(ranked)) {
		group.Add(1)
		go func() {
			defer group.Done()
			for candidate := range jobs {
				result, coverage, err := executor(ctx, candidate.experiment)
				results <- executionResult{ranked: candidate, result: result, coverage: coverage, err: err}
			}
		}()
	}
	go func() {
		for _, candidate := range ranked {
			jobs <- candidate
		}
		close(jobs)
		group.Wait()
		close(results)
	}()
	merged := make([]executionResult, 0, len(ranked))
	for result := range results {
		merged = append(merged, result)
	}
	slices.SortFunc(merged, func(left, right executionResult) int { return compare(left.ranked.digest, right.ranked.digest) })
	return merged
}

func minimizeDiscovery(ctx context.Context, execution executionResult, maxAttempts int, executor Executor) Discovery {
	discovery := Discovery{CandidateID: execution.ranked.candidateID, Original: execution.ranked.experiment}
	if maxAttempts <= 0 {
		discovery.Minimized = execution.ranked.experiment
		discovery.PromotionBlock = "minimization budget is required"
		return discovery
	}
	attempts := 0
	budgetErr := errors.New("minimization attempt budget exhausted")
	minimized, err := umpire3runtime.MinimizeExperiment(ctx, execution.ranked.experiment,
		func(ctx context.Context, experiment protocol.Experiment) (umpire3runtime.Result, error) {
			if attempts == maxAttempts {
				return umpire3runtime.Result{}, budgetErr
			}
			attempts++
			result, _, err := executor(ctx, experiment)
			return result, err
		})
	discovery.Minimization = Minimization{
		Attempts: attempts, Complete: minimizationComplete(attempts, maxAttempts, err == nil),
	}
	if err != nil {
		discovery.Minimized = execution.ranked.experiment
		discovery.PromotionBlock = err.Error()
		return discovery
	}
	discovery.Minimized = minimized
	source, sourceErr := promotionSource(minimized)
	if sourceErr != nil {
		discovery.PromotionBlock = sourceErr.Error()
		return discovery
	}
	discovery.Promotion = Promotion{Source: source}
	return discovery
}

func minimizationComplete(attempts, maxAttempts int, finished bool) bool {
	return finished && attempts <= maxAttempts
}

func promotionSource(experiment protocol.Experiment) (string, error) {
	target := targetForExperiment(experiment)
	if target == "" {
		return "", errors.New("minimized experiment has no checked composition target")
	}
	root, err := promotionRoot(experiment)
	if err != nil {
		return "", err
	}
	var source strings.Builder
	source.WriteString("package umpire3promotion\n\n")
	source.WriteString("import (\n")
	source.WriteString("\t\"go.temporal.io/server/tests/umpire3/compiler\"\n")
	source.WriteString("\t\"go.temporal.io/server/tests/umpire3/environment\"\n")
	source.WriteString("\t\"go.temporal.io/server/tests/umpire3/protocol\"\n")
	source.WriteString("\t\"go.temporal.io/server/tests/umpire3/umpire3test\"\n")
	source.WriteString(")\n\n")
	source.WriteString("func RequirePromoted(t umpire3test.TestingT, factory environment.Factory) {\n")
	source.WriteString("\tscenario := compiler.Scenario{\n")
	fmt.Fprintf(&source, "\t\tIdentifier: %q,\n", experiment.ExperimentID+"-promoted")
	fmt.Fprintf(&source, "\t\tTarget: protocol.TargetID(%q),\n", target)
	source.WriteString("\t\tResources: []compiler.Resource{\n")
	for _, resource := range experiment.Resources {
		fmt.Fprintf(&source, "\t\t\t{Identifier: %q, Kind: protocol.EntityKind(%q)},\n", resource.Identifier, resource.Kind)
	}
	source.WriteString("\t\t},\n")
	fmt.Fprintf(&source, "\t\tRoot: compiler.OnePath(%s, compiler.Require(protocol.PropertyID(%q))),\n",
		root, experiment.Property.Identifier)
	source.WriteString("\t}\n")
	source.WriteString("\tumpire3test.RequireRegression(t, scenario, umpire3test.WithEnvironment(factory))\n")
	source.WriteString("}\n")
	source.WriteString("\nfunc textValue(kind protocol.ValueType, value string) protocol.Value {\n")
	source.WriteString("\treturn protocol.Value{Type: kind, Text: &value}\n}\n")
	source.WriteString("\nfunc integerValue(kind protocol.ValueType, value int64) protocol.Value {\n")
	source.WriteString("\treturn protocol.Value{Type: kind, Integer: &value}\n}\n")
	source.WriteString("\nfunc booleanValue(value bool) protocol.Value {\n")
	source.WriteString("\treturn protocol.Value{Type: protocol.ValueBoolean, Boolean: &value}\n}\n")
	source.WriteString("\nfunc enumValue(name string, number int64) protocol.Value {\n")
	source.WriteString("\treturn protocol.Value{Type: protocol.ValueEnum, Text: &name, Integer: &number}\n}\n")
	formatted, err := format.Source([]byte(source.String()))
	if err != nil {
		return "", fmt.Errorf("format promoted regression: %w", err)
	}
	return string(formatted), nil
}

type promotionInterval struct {
	start    int
	end      int
	fault    protocol.Fault
	children []*promotionInterval
}

func promotionRoot(experiment protocol.Experiment) (string, error) {
	actionIndexes := make(map[string]int, len(experiment.Actions))
	for index, action := range experiment.Actions {
		actionIndexes[action.Identifier] = index
	}
	policyScopes := make(map[string][]string, len(experiment.Policies))
	for _, policy := range experiment.Policies {
		policyScopes[policy.Identifier] = policy.Scope
	}
	intervals := make([]*promotionInterval, 0, len(experiment.Faults))
	for _, fault := range experiment.Faults {
		scope, exists := policyScopes[fault.Policy]
		if !exists || len(scope) == 0 {
			return "", fmt.Errorf("fault %q has no bounded policy scope", fault.Identifier)
		}
		start, startExists := actionIndexes[scope[0]]
		end, endExists := actionIndexes[scope[len(scope)-1]]
		if !startExists || !endExists || start > end || len(scope) != end-start+1 {
			return "", fmt.Errorf("fault %q policy is not a contiguous sparse action interval", fault.Identifier)
		}
		for offset, identifier := range scope {
			if experiment.Actions[start+offset].Identifier != identifier {
				return "", fmt.Errorf("fault %q policy order differs from promoted action order", fault.Identifier)
			}
		}
		intervals = append(intervals, &promotionInterval{start: start, end: end, fault: fault})
	}
	slices.SortFunc(intervals, func(left, right *promotionInterval) int {
		if left.start != right.start {
			return left.start - right.start
		}
		if left.end != right.end {
			return right.end - left.end
		}
		return compare(left.fault.Identifier, right.fault.Identifier)
	})
	root := &promotionInterval{start: 0, end: len(experiment.Actions) - 1}
	stack := []*promotionInterval{root}
	for _, interval := range intervals {
		for len(stack) > 1 && interval.start > stack[len(stack)-1].end {
			stack = stack[:len(stack)-1]
		}
		parent := stack[len(stack)-1]
		if interval.start < parent.start || interval.end > parent.end {
			return "", fmt.Errorf("fault %q has a crossing policy interval", interval.fault.Identifier)
		}
		parent.children = append(parent.children, interval)
		stack = append(stack, interval)
	}
	return renderPromotionInterval(root, experiment, true)
}

func renderPromotionInterval(
	interval *promotionInterval,
	experiment protocol.Experiment,
	synthetic bool,
) (string, error) {
	var terms []string
	childIndex := 0
	for index := interval.start; index <= interval.end; {
		if childIndex < len(interval.children) && interval.children[childIndex].start == index {
			child := interval.children[childIndex]
			rendered, err := renderPromotionInterval(child, experiment, false)
			if err != nil {
				return "", err
			}
			terms = append(terms, rendered)
			index = child.end + 1
			childIndex++
			continue
		}
		terms = append(terms, renderPromotionAction(experiment.Actions[index]))
		index++
	}
	body := "compiler.OnePath(" + strings.Join(terms, ", ") + ")"
	if synthetic {
		return body, nil
	}
	return "compiler.During(compiler.ConfiguredFault(" + renderPromotionFault(interval.fault) + "), " + body + ")", nil
}

func renderPromotionAction(action protocol.Action) string {
	options := make([]string, len(action.Arguments))
	for index, argument := range action.Arguments {
		options[index] = fmt.Sprintf("compiler.WithArgument(%q, %s)", argument.Name, renderPromotionValue(argument.Value))
	}
	arguments := []string{fmt.Sprintf("%q", action.Identifier), fmt.Sprintf("protocol.ActionKind(%q)", action.Kind)}
	arguments = append(arguments, options...)
	actionTerm := "compiler.Action(" + strings.Join(arguments, ", ") + ")"
	if len(action.Bindings) == 0 {
		return actionTerm
	}
	terms := []string{actionTerm}
	for _, binding := range action.Bindings {
		terms = append(terms, fmt.Sprintf(
			"compiler.Bind(compiler.Symbol{Name: %q, Type: protocol.SemanticTypeID(%q)}, compiler.Project(%q, %q, protocol.SemanticTypeID(%q)))",
			binding.Symbol, binding.Type, action.Identifier, binding.Projection, binding.Type))
	}
	return "compiler.OnePath(" + strings.Join(terms, ", ") + ")"
}

func renderPromotionValue(value protocol.Value) string {
	switch value.Type {
	case protocol.ValueString, protocol.ValueDuration, protocol.ValueBytesDigest, protocol.ValueSymbol:
		if value.Type == protocol.ValueDuration {
			return fmt.Sprintf("integerValue(protocol.ValueType(%q), %d)", value.Type, valueOrZero(value.Integer))
		}
		return fmt.Sprintf("textValue(protocol.ValueType(%q), %q)", value.Type, textOrEmpty(value.Text))
	case protocol.ValueInteger:
		return fmt.Sprintf("integerValue(protocol.ValueInteger, %d)", valueOrZero(value.Integer))
	case protocol.ValueBoolean:
		return fmt.Sprintf("booleanValue(%t)", boolOrFalse(value.Boolean))
	case protocol.ValueEnum:
		return fmt.Sprintf("enumValue(%q, %d)", textOrEmpty(value.Text), valueOrZero(value.Integer))
	case protocol.ValueList:
		elements := make([]string, len(value.Elements))
		for index, element := range value.Elements {
			elements[index] = renderPromotionValue(element)
		}
		return "protocol.Value{Type: protocol.ValueList, Elements: []protocol.Value{" + strings.Join(elements, ", ") + "}}"
	case protocol.ValueRecord:
		fields := make([]string, len(value.Fields))
		for index, field := range value.Fields {
			fields[index] = fmt.Sprintf("{Name: %q, Value: %s}", field.Name, renderPromotionValue(field.Value))
		}
		return "protocol.Value{Type: protocol.ValueRecord, Fields: []protocol.NamedValue{" + strings.Join(fields, ", ") + "}}"
	default:
		return "protocol.Value{}"
	}
}

func renderPromotionFault(fault protocol.Fault) string {
	return fmt.Sprintf(`protocol.Fault{
		Identifier: %q, Kind: %q, Policy: %q, SafetyClass: %q,
		Scope: protocol.FaultScope{Resources: %#v, Endpoints: %#v, TaskQueues: %#v, Services: %#v, Routes: %#v, Participants: %#v, Attempts: %#v},
		Occurrence: protocol.FaultOccurrence{First: %d, Count: %d},
		Interval: protocol.FaultInterval{StartAction: %q, StopAction: %q},
		Arguments: %s, RequiredCapabilities: %#v,
	}`,
		fault.Identifier, fault.Kind, fault.Policy, fault.SafetyClass,
		fault.Scope.Resources, fault.Scope.Endpoints, fault.Scope.TaskQueues, fault.Scope.Services,
		fault.Scope.Routes, fault.Scope.Participants, fault.Scope.Attempts,
		fault.Occurrence.First, fault.Occurrence.Count, fault.Interval.StartAction, fault.Interval.StopAction,
		renderPromotionNamedValues(fault.Arguments), fault.RequiredCapabilities)
}

func renderPromotionNamedValues(values []protocol.NamedValue) string {
	if len(values) == 0 {
		return "nil"
	}
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = fmt.Sprintf("{Name: %q, Value: %s}", value.Name, renderPromotionValue(value.Value))
	}
	return "[]protocol.NamedValue{" + strings.Join(result, ", ") + "}"
}

func textOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func valueOrZero(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func boolOrFalse(value *bool) bool {
	return value != nil && *value
}

func targetForExperiment(experiment protocol.Experiment) protocol.TargetID {
	composition, err := protocol.DefaultComposition()
	if err != nil {
		return ""
	}
	var candidates []protocol.TargetID
	for _, target := range composition.Targets {
		if slices.Contains(target.Properties, protocol.PropertyID(experiment.Property.Identifier)) {
			modules := make([]string, len(target.Modules))
			for index, module := range target.Modules {
				modules[index] = string(module)
			}
			if slices.Equal(modules, experiment.Model.Modules) {
				return target.Identifier
			}
			containsAll := true
			for _, module := range experiment.Model.Modules {
				if !slices.Contains(modules, module) {
					containsAll = false
					break
				}
			}
			if containsAll {
				candidates = append(candidates, target.Identifier)
			}
		}
	}
	if len(candidates) == 1 {
		return candidates[0]
	}
	return ""
}

func campaignScore(candidate Candidate, covered, risk map[CoveragePoint]struct{}) int {
	score := 0
	for _, point := range normalizeCoverage(candidate.Coverage) {
		if _, exists := covered[point]; !exists {
			score += 10
		}
	}
	for _, point := range normalizeCoverage(candidate.Risk) {
		if _, focused := risk[point]; focused {
			score += 100
		}
	}
	return score
}

func seededOrder(seed int64, digest string) uint64 {
	value := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", seed, digest)))
	return binary.BigEndian.Uint64(value[:8])
}

func normalizeCoverage(points []CoveragePoint) []CoveragePoint {
	result := append([]CoveragePoint(nil), points...)
	slices.SortFunc(result, func(left, right CoveragePoint) int {
		if compared := compare(string(left.Kind), string(right.Kind)); compared != 0 {
			return compared
		}
		return compare(left.Identifier, right.Identifier)
	})
	return slices.Compact(result)
}

func coverageSet(points []CoveragePoint) map[CoveragePoint]struct{} {
	result := make(map[CoveragePoint]struct{}, len(points))
	for _, point := range points {
		result[point] = struct{}{}
	}
	return result
}

func coverageSlice(points map[CoveragePoint]struct{}) []CoveragePoint {
	result := make([]CoveragePoint, 0, len(points))
	for point := range points {
		result = append(result, point)
	}
	return normalizeCoverage(result)
}

func coverageDifference(before, after []CoveragePoint) []CoveragePoint {
	known := coverageSet(before)
	var delta []CoveragePoint
	for _, point := range after {
		if _, exists := known[point]; !exists {
			delta = append(delta, point)
		}
	}
	return normalizeCoverage(delta)
}

func sortReport(report *Report) {
	slices.SortFunc(report.Executions, func(left, right Execution) int { return compare(left.Digest, right.Digest) })
	slices.SortFunc(report.Dropped, func(left, right Dropped) int {
		if result := compare(left.CandidateID, right.CandidateID); result != 0 {
			return result
		}
		return compare(left.Digest, right.Digest)
	})
	slices.SortFunc(report.Discoveries, func(left, right Discovery) int {
		return compare(left.CandidateID, right.CandidateID)
	})
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
