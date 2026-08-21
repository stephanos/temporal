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
	"unicode"

	umpire3runtime "go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/replay"
	"go.temporal.io/server/tests/umpire3/scenario"
)

type CoverageKind string

const (
	CoverageTransition  CoverageKind = "transition"
	CoverageProperty    CoverageKind = "property"
	CoverageRelation    CoverageKind = "relation"
	CoverageRefinement  CoverageKind = "refinement"
	CoverageObservation CoverageKind = "observation"
	CoverageEvidence    CoverageKind = "evidence"
	CoverageProtobuf    CoverageKind = "protobuf-field-class"
	CoverageParameter   CoverageKind = "parameter"
	CoverageFault       CoverageKind = "fault"
	CoverageSchedule    CoverageKind = "schedule"
	CoverageTopology    CoverageKind = "topology"
	CoverageProfile     CoverageKind = "profile"
	CoverageAction      CoverageKind = "action"
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
	Scenario   scenario.Scenario
	Coverage   []CoveragePoint
	Risk       []CoveragePoint
}

type Executor func(context.Context, protocol.Experiment) (umpire3runtime.Result, []CoveragePoint, error)

type Request struct {
	Candidates       []Candidate
	Traces           []protocol.SemanticTrace
	Mutation         *MutationRequest
	Seed             int64
	Workers          int
	MaxExecutions    int
	MinimizeAttempts int
	CompilerLimits   scenario.Limits
	RiskFocus        []CoveragePoint
	CorpusCoverage   []CoveragePoint
	Executor         Executor
}

type Execution struct {
	CandidateID string                `json:"candidateID"`
	Mutation    MutationKind          `json:"mutation,omitempty"`
	Path        string                `json:"path,omitempty"`
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
	Mutation       MutationKind        `json:"mutation,omitempty"`
	Path           string              `json:"path,omitempty"`
	Original       protocol.Experiment `json:"original"`
	Minimized      protocol.Experiment `json:"minimized"`
	Minimization   Minimization        `json:"minimization"`
	BundleDigest   string              `json:"bundleDigest,omitempty"`
	Replay         replay.Report       `json:"replay,omitempty"`
	Promotion      Promotion           `json:"promotion"`
	PromotionBlock string              `json:"promotionBlock,omitempty"`
}

type Report struct {
	Mutation       *MutationReport `json:"mutation,omitempty"`
	CoverageBefore []CoveragePoint `json:"coverageBefore"`
	CoverageAfter  []CoveragePoint `json:"coverageAfter"`
	CoverageDelta  []CoveragePoint `json:"coverageDelta"`
	Executions     []Execution     `json:"executions"`
	Dropped        []Dropped       `json:"dropped"`
	Discoveries    []Discovery     `json:"discoveries"`
}

type rankedExperiment struct {
	candidateID string
	mutation    MutationKind
	path        string
	experiment  protocol.Experiment
	digest      string
	score       int
	seedOrder   uint64
	coverage    []CoveragePoint
}

type executionResult struct {
	ranked   rankedExperiment
	result   umpire3runtime.Result
	coverage []CoveragePoint
	err      error
}

func Run(ctx context.Context, request Request) (Report, error) {
	sources := 0
	if len(request.Candidates) != 0 {
		sources++
	}
	if len(request.Traces) != 0 {
		sources++
	}
	if request.Mutation != nil {
		sources++
	}
	if request.Executor == nil || request.Workers <= 0 || request.MaxExecutions <= 0 || sources != 1 {
		return Report{}, errors.New("campaign requires exactly one candidate source plus an executor, workers, and execution budget")
	}
	report := Report{CoverageBefore: normalizeCoverage(request.CorpusCoverage)}
	covered := coverageSet(report.CoverageBefore)
	risk := coverageSet(request.RiskFocus)
	seenDigests := make(map[string]struct{})
	var ranked []rankedExperiment
	if request.Mutation != nil {
		mutations, err := Mutate(*request.Mutation)
		if err != nil {
			return Report{}, err
		}
		report.Mutation = &mutations
		for _, mutation := range mutations.Selected {
			coverage, err := modelCoverage(mutation.Experiment)
			if err != nil {
				return Report{}, err
			}
			mutationPoint, err := mutationCoverage(mutation.Kind, mutation.Path)
			if err != nil {
				return Report{}, err
			}
			coverage = normalizeCoverage(append(coverage, mutationPoint))
			ranked = append(ranked, rankedExperiment{
				candidateID: string(mutation.Kind) + ":" + mutation.Path,
				mutation:    mutation.Kind, path: mutation.Path,
				experiment: mutation.Experiment, digest: mutation.Digest, coverage: coverage,
				score: campaignScore(coverage, nil, covered, risk), seedOrder: seededOrder(request.Seed, mutation.Digest),
			})
		}
	} else {
		candidates := append([]Candidate(nil), request.Candidates...)
		for _, trace := range request.Traces {
			identifier := scenario.SemanticTraceIdentifier(trace)
			if err := trace.Validate(); err != nil {
				return Report{}, fmt.Errorf("validate semantic trace %q: %w", identifier, err)
			}
			if trace.Kind == protocol.SemanticTraceLive {
				experiment := *trace.Experiment
				coverage, err := modelCoverage(experiment)
				if err != nil {
					return Report{}, fmt.Errorf("derive semantic trace %q coverage: %w", identifier, err)
				}
				digest, err := experiment.Digest()
				if err != nil {
					return Report{}, fmt.Errorf("digest semantic trace %q experiment: %w", identifier, err)
				}
				if _, duplicate := seenDigests[digest]; duplicate {
					report.Dropped = append(report.Dropped, Dropped{
						CandidateID: identifier, Digest: digest, Reason: DropDuplicate,
						Detail: "compiled experiment digest already ranked",
					})
					continue
				}
				seenDigests[digest] = struct{}{}
				ranked = append(ranked, rankedExperiment{
					candidateID: identifier, experiment: experiment, digest: digest,
					coverage: coverage, score: campaignScore(coverage, nil, covered, risk),
					seedOrder: seededOrder(request.Seed, digest),
				})
				continue
			}
			authored, err := scenario.FromSemanticTrace(identifier, trace)
			if err != nil {
				return Report{}, fmt.Errorf("compile semantic trace %q: %w", identifier, err)
			}
			candidates = append(candidates, Candidate{Identifier: identifier, Scenario: authored})
		}
		for _, candidate := range candidates {
			if candidate.Identifier == "" {
				report.Dropped = append(report.Dropped, Dropped{Reason: DropUnsupported, Detail: "candidate identifier is required"})
				continue
			}
			suite, err := scenario.Compile(ctx, candidate.Scenario, request.CompilerLimits)
			if err != nil {
				report.Dropped = append(report.Dropped, Dropped{
					CandidateID: candidate.Identifier, Reason: DropUnsupported, Detail: err.Error(),
				})
				continue
			}
			for index, experiment := range suite.Experiments {
				derived, err := modelCoverage(experiment)
				if err != nil {
					return Report{}, fmt.Errorf("derive candidate %q coverage: %w", candidate.Identifier, err)
				}
				coverage := normalizeCoverage(append(append([]CoveragePoint(nil), candidate.Coverage...), derived...))
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
					coverage: coverage,
					score:    campaignScore(coverage, candidate.Risk, covered, risk), seedOrder: seededOrder(request.Seed, digest),
				})
			}
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
		coverage := normalizeCoverage(append(append([]CoveragePoint(nil), execution.ranked.coverage...), execution.coverage...))
		report.Executions = append(report.Executions, Execution{
			CandidateID: execution.ranked.candidateID, Mutation: execution.ranked.mutation,
			Path: execution.ranked.path, Digest: execution.ranked.digest,
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
	discovery := Discovery{
		CandidateID: execution.ranked.candidateID, Mutation: execution.ranked.mutation,
		Path: execution.ranked.path, Original: execution.ranked.experiment,
	}
	if maxAttempts <= 0 {
		discovery.Minimized = execution.ranked.experiment
		discovery.PromotionBlock = "minimization budget is required"
		return discovery
	}
	attempts := 0
	budgetErr := errors.New("minimization attempt budget exhausted")
	minimized, err := MinimizeExperiment(ctx, execution.ranked.experiment,
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
	minimizedResult, _, executeErr := executor(ctx, minimized)
	if executeErr != nil {
		discovery.PromotionBlock = executeErr.Error()
		return discovery
	}
	encoded, encodeErr := replay.EncodeBundle(minimized, minimizedResult, minimized.Retention.MaxArtifactBytes)
	if encodeErr != nil {
		discovery.PromotionBlock = encodeErr.Error()
		return discovery
	}
	bundle, decodeErr := replay.DecodeBundle(encoded, minimized.Retention.MaxArtifactBytes)
	if decodeErr != nil {
		discovery.PromotionBlock = decodeErr.Error()
		return discovery
	}
	replayed, replayErr := replay.Reproduce(ctx, bundle, func(
		ctx context.Context,
		candidate protocol.Experiment,
	) (umpire3runtime.Result, error) {
		result, _, err := executor(ctx, candidate)
		return result, err
	})
	if replayErr != nil {
		discovery.PromotionBlock = replayErr.Error()
		return discovery
	}
	if !replayed.Reproduced {
		discovery.PromotionBlock = "minimized replay did not reproduce the qualified violation"
		return discovery
	}
	bundleHash := sha256.Sum256(encoded)
	discovery.BundleDigest = fmt.Sprintf("sha256:%x", bundleHash)
	discovery.Replay = replayed
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
	source.WriteString("\t\"go.temporal.io/server/tests/umpire3/execution\"\n")
	source.WriteString("\t\"go.temporal.io/server/tests/umpire3/scenario\"\n")
	source.WriteString("\t\"go.temporal.io/server/tests/umpire3/umpire3test\"\n")
	source.WriteString(")\n\n")
	source.WriteString("func RequirePromoted(t umpire3test.TestingT, factory execution.Factory) {\n")
	fmt.Fprintf(&source, "\tauthored := scenario.%sRegression(%q, []scenario.Resource{\n",
		facadeIdentifier(string(target)), experiment.ExperimentID+"-promoted")
	for _, resource := range experiment.Resources {
		fmt.Fprintf(&source, "\t\tscenario.%s(%q),\n", facadeIdentifier(string(resource.Kind)), resource.Identifier)
	}
	fmt.Fprintf(&source, "\t}, scenario.OnePath(%s, scenario.Require%s()))\n",
		root, facadeIdentifier(experiment.Property.Identifier))
	source.WriteString("\tumpire3test.RequireRegression(t, authored, umpire3test.WithEnvironment(factory))\n")
	source.WriteString("}\n")
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
		action, err := renderPromotionAction(experiment.Actions[index])
		if err != nil {
			return "", err
		}
		terms = append(terms, action)
		index++
	}
	body := "scenario.OnePath(" + strings.Join(terms, ", ") + ")"
	if synthetic {
		return body, nil
	}
	fault, err := renderPromotionFault(interval.fault, experiment)
	if err != nil {
		return "", err
	}
	return "scenario.During(" + fault + ", " + body + ")", nil
}

func renderPromotionAction(action protocol.Action) (string, error) {
	options := make([]string, 0, len(action.Arguments)+2)
	for _, argument := range action.Arguments {
		option, err := renderPromotionActionArgument(action, argument)
		if err != nil {
			return "", err
		}
		options = append(options, option)
	}
	if len(action.AllowedOutcomes) != 0 {
		outcomes := make([]string, len(action.AllowedOutcomes))
		for index, outcome := range action.AllowedOutcomes {
			name, err := promotionOutcomeName(outcome)
			if err != nil {
				return "", fmt.Errorf("action %q: %w", action.Identifier, err)
			}
			outcomes[index] = "scenario." + name
		}
		options = append(options, "scenario.Outcomes("+strings.Join(outcomes, ", ")+")")
	}
	switch action.EffectiveResponseMode() {
	case protocol.ResponseSynchronous:
	case protocol.ResponseAsynchronous:
		options = append(options, "scenario.Asynchronously()")
	case protocol.ResponseDeferred:
		options = append(options, "scenario.Deferred()")
	case protocol.ResponseBlocking:
		options = append(options, fmt.Sprintf("scenario.BlockingFor(%d)", action.MaxBlockNanos))
	case protocol.ResponseFailure:
		options = append(options, "scenario.FailingResponse()")
	default:
		return "", fmt.Errorf("action %q has unsupported response mode %q", action.Identifier, action.ResponseMode)
	}
	arguments := []string{fmt.Sprintf("%q", action.Identifier)}
	arguments = append(arguments, options...)
	actionTerm := "scenario." + facadeIdentifier(action.Kind) + "(" + strings.Join(arguments, ", ") + ")"
	if len(action.Bindings) == 0 {
		return actionTerm, nil
	}
	terms := []string{actionTerm}
	for _, binding := range action.Bindings {
		if binding.Type != string(protocol.SemanticTypeIDIdentity) {
			return "", fmt.Errorf("action %q binding %q has unsupported promoted type %q",
				action.Identifier, binding.Symbol, binding.Type)
		}
		terms = append(terms, fmt.Sprintf(
			"scenario.BindIdentity(scenario.Identity(%q), %q, %q)",
			binding.Symbol, action.Identifier, binding.Projection))
	}
	return "scenario.OnePath(" + strings.Join(terms, ", ") + ")", nil
}

func renderPromotionActionArgument(action protocol.Action, argument protocol.NamedValue) (string, error) {
	catalog, err := protocol.DefaultCatalog()
	if err != nil {
		return "", err
	}
	declaration, found := catalog.Action(action.Kind)
	if !found {
		return "", fmt.Errorf("action %q has unknown kind %q", action.Identifier, action.Kind)
	}
	for _, parameter := range declaration.Parameters {
		if parameter.Name != argument.Name {
			continue
		}
		name := "scenario.With" + facadeIdentifier(parameter.Name)
		switch parameter.Type {
		case "string":
			if argument.Value.Type != protocol.ValueString || argument.Value.Text == nil {
				return "", fmt.Errorf("action %q parameter %q requires a string", action.Identifier, argument.Name)
			}
			return fmt.Sprintf("%s(%q)", name, *argument.Value.Text), nil
		case "identity":
			if argument.Value.Type != protocol.ValueSymbol || argument.Value.Text == nil {
				return "", fmt.Errorf("action %q parameter %q requires an identity symbol", action.Identifier, argument.Name)
			}
			return fmt.Sprintf("%s(scenario.Identity(%q))", name, *argument.Value.Text), nil
		default:
			return "", fmt.Errorf("action %q parameter %q has unsupported facade type %q",
				action.Identifier, argument.Name, parameter.Type)
		}
	}
	return "", fmt.Errorf("action %q has undeclared argument %q", action.Identifier, argument.Name)
}

func renderPromotionValue(value protocol.Value) (string, error) {
	switch value.Type {
	case protocol.ValueString:
		return fmt.Sprintf("scenario.String(%q)", textOrEmpty(value.Text)), nil
	case protocol.ValueInteger:
		return fmt.Sprintf("scenario.Integer(%d)", valueOrZero(value.Integer)), nil
	case protocol.ValueBoolean:
		return fmt.Sprintf("scenario.Boolean(%t)", boolOrFalse(value.Boolean)), nil
	case protocol.ValueDuration:
		return fmt.Sprintf("scenario.Duration(%d)", valueOrZero(value.Integer)), nil
	case protocol.ValueEnum:
		return fmt.Sprintf("scenario.Enum(%q, %d)", textOrEmpty(value.Text), valueOrZero(value.Integer)), nil
	case protocol.ValueBytesDigest:
		return fmt.Sprintf("scenario.BytesDigest(%q)", textOrEmpty(value.Text)), nil
	case protocol.ValueSymbol:
		return fmt.Sprintf("scenario.SymbolValue(scenario.Identity(%q))", textOrEmpty(value.Text)), nil
	case protocol.ValueList:
		elements := make([]string, len(value.Elements))
		for index, element := range value.Elements {
			rendered, err := renderPromotionValue(element)
			if err != nil {
				return "", err
			}
			elements[index] = rendered
		}
		return "scenario.List(" + strings.Join(elements, ", ") + ")", nil
	case protocol.ValueRecord:
		fields := make([]string, len(value.Fields))
		for index, field := range value.Fields {
			rendered, err := renderPromotionValue(field.Value)
			if err != nil {
				return "", err
			}
			fields[index] = fmt.Sprintf("scenario.Named(%q, %s)", field.Name, rendered)
		}
		return "scenario.Record(" + strings.Join(fields, ", ") + ")", nil
	default:
		return "", fmt.Errorf("unsupported promoted value type %q", value.Type)
	}
}

func renderPromotionFault(fault protocol.Fault, experiment protocol.Experiment) (string, error) {
	catalog, err := protocol.DefaultCatalog()
	if err != nil {
		return "", err
	}
	if _, found := catalog.Fault(fault.Kind); !found {
		return "", fmt.Errorf("fault %q has unknown kind %q", fault.Identifier, fault.Kind)
	}
	options := make([]string, 0, 8+len(fault.Arguments))
	if len(fault.Scope.Resources) != 0 {
		resources, err := renderPromotionResources(fault.Scope.Resources, experiment.Resources)
		if err != nil {
			return "", fmt.Errorf("fault %q: %w", fault.Identifier, err)
		}
		options = append(options, "scenario.OnResources("+strings.Join(resources, ", ")+")")
	}
	appendStrings := func(name string, values []string) {
		if len(values) == 0 {
			return
		}
		options = append(options, "scenario."+name+"("+joinQuoted(values)+")")
	}
	appendStrings("OnEndpoints", fault.Scope.Endpoints)
	appendStrings("OnTaskQueues", fault.Scope.TaskQueues)
	appendStrings("OnServices", fault.Scope.Services)
	appendStrings("OnRoutes", fault.Scope.Routes)
	appendStrings("OnParticipants", fault.Scope.Participants)
	if len(fault.Scope.Attempts) != 0 {
		attempts := make([]string, len(fault.Scope.Attempts))
		for index, attempt := range fault.Scope.Attempts {
			attempts[index] = fmt.Sprint(attempt)
		}
		options = append(options, "scenario.OnAttempts("+strings.Join(attempts, ", ")+")")
	}
	options = append(options, fmt.Sprintf("scenario.AtOccurrence(%d, %d)",
		fault.Occurrence.First, fault.Occurrence.Count))
	for _, argument := range fault.Arguments {
		value, err := renderPromotionValue(argument.Value)
		if err != nil {
			return "", fmt.Errorf("fault %q argument %q: %w", fault.Identifier, argument.Name, err)
		}
		options = append(options, fmt.Sprintf("scenario.WithFaultValue(%q, %s)", argument.Name, value))
	}
	arguments := append([]string{fmt.Sprintf("%q", fault.Identifier)}, options...)
	return "scenario." + facadeIdentifier(fault.Kind) + "(" + strings.Join(arguments, ", ") + ")", nil
}

func renderPromotionResources(identifiers []string, resources []protocol.Resource) ([]string, error) {
	byIdentifier := make(map[string]protocol.Resource, len(resources))
	for _, resource := range resources {
		byIdentifier[resource.Identifier] = resource
	}
	result := make([]string, len(identifiers))
	for index, identifier := range identifiers {
		resource, found := byIdentifier[identifier]
		if !found {
			return nil, fmt.Errorf("scope references unknown resource %q", identifier)
		}
		result[index] = fmt.Sprintf("scenario.%s(%q)", facadeIdentifier(string(resource.Kind)), identifier)
	}
	return result, nil
}

func joinQuoted(values []string) string {
	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = fmt.Sprintf("%q", value)
	}
	return strings.Join(quoted, ", ")
}

func promotionOutcomeName(outcome protocol.ActionOutcome) (string, error) {
	switch outcome {
	case protocol.ActionOutcomeApplied:
		return "Applied", nil
	case protocol.ActionOutcomeSuppressed:
		return "Suppressed", nil
	case protocol.ActionOutcomeRejected:
		return "Rejected", nil
	case protocol.ActionOutcomeRetried:
		return "Retried", nil
	case protocol.ActionOutcomeFaultIntercepted:
		return "FaultIntercepted", nil
	default:
		return "", fmt.Errorf("unsupported action outcome %q", outcome)
	}
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
	boundTarget := protocol.TargetID("")
	if strings.HasPrefix(experiment.Provenance.ProofManifest, "composition:") {
		boundTarget = protocol.TargetID(strings.TrimPrefix(experiment.Provenance.ProofManifest, "composition:"))
	}
	var candidates []protocol.TargetID
	for _, target := range composition.Targets {
		if boundTarget != "" && target.Identifier != boundTarget {
			continue
		}
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

func facadeIdentifier(value string) string {
	parts := strings.FieldsFunc(value, func(character rune) bool {
		return !unicode.IsLetter(character) && !unicode.IsNumber(character)
	})
	var result strings.Builder
	for _, part := range parts {
		characters := []rune(part)
		if len(characters) == 0 {
			continue
		}
		result.WriteRune(unicode.ToUpper(characters[0]))
		result.WriteString(string(characters[1:]))
	}
	return result.String()
}

func campaignScore(candidateCoverage, candidateRisk []CoveragePoint, covered, risk map[CoveragePoint]struct{}) int {
	score := 0
	for _, point := range normalizeCoverage(candidateCoverage) {
		if _, exists := covered[point]; !exists {
			score += 10
		}
	}
	for _, point := range normalizeCoverage(candidateRisk) {
		if _, focused := risk[point]; focused {
			score += 100
		}
	}
	return score
}

func modelCoverage(experiment protocol.Experiment) ([]CoveragePoint, error) {
	denominator, err := protocol.DefaultCoverageDenominator()
	if err != nil {
		return nil, err
	}
	points, err := denominator.PointsForExperiment(experiment)
	if err != nil {
		return nil, err
	}
	result := make([]CoveragePoint, len(points))
	for index, point := range points {
		result[index] = CoveragePoint{Kind: CoverageKind(point.Dimension), Identifier: point.Identifier}
	}
	return result, nil
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
