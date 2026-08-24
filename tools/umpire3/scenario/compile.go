package scenario

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"go.temporal.io/server/tools/umpire3/checker/finite"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
	protocolmonitor "go.temporal.io/server/tools/umpire3/protocol/monitor"
)

func Compile(ctx context.Context, scenario Scenario, limits Limits) (Suite, error) {
	if err := validateLimits(limits); err != nil {
		return Suite{}, err
	}
	compileCtx, cancel := context.WithTimeout(ctx, limits.MaxTime)
	defer cancel()

	catalog, err := protocolcatalog.DefaultCatalog()
	if err != nil {
		return Suite{}, fmt.Errorf("load semantic catalog: %w", err)
	}
	composition, err := protocolcatalog.DefaultComposition()
	if err != nil {
		return Suite{}, fmt.Errorf("load model composition: %w", err)
	}
	monitors, err := protocolmonitor.DefaultMonitorCatalog()
	if err != nil {
		return Suite{}, fmt.Errorf("load monitor programs: %w", err)
	}
	if scenario.Identifier == "" || len(scenario.Resources) == 0 {
		return Suite{}, compileError(ErrorInvalidIntent, scenario.Root.source, "scenario identifier and resources are required")
	}
	target, ok := compositionTarget(composition, scenario.Target)
	if !ok {
		return Suite{}, compileError(ErrorInvalidIntent, scenario.Root.source, fmt.Sprintf("unknown target %q", scenario.Target))
	}

	plan, err := normalize(scenario.Root)
	if err != nil {
		return Suite{}, err
	}
	if !slices.Contains(target.Properties, plan.property) {
		return Suite{}, compileError(ErrorInvalidIntent, scenario.Root.source,
			fmt.Sprintf("target %q does not prove property %q", target.Identifier, plan.property))
	}
	if err := validateResources(scenario.Resources, catalog); err != nil {
		return Suite{}, err
	}
	if err := completeDependencies(plan, catalog, target); err != nil {
		return Suite{}, err
	}
	if len(plan.actions) > limits.MaxActions {
		return Suite{}, compileError(ErrorLimitExceeded, scenario.Root.source,
			fmt.Sprintf("completed action count %d exceeds limit %d", len(plan.actions), limits.MaxActions))
	}
	identities, err := compileBindings(plan, catalog)
	if err != nil {
		return Suite{}, err
	}
	plan.edges = sortAndCompactEdges(plan.edges)

	enumerated, err := enumerate(compileCtx, plan.actions, plan.edges, plan.allPaths, limits)
	if err != nil {
		return Suite{}, err
	}
	monitor, ok := monitors.Program(plan.property)
	if !ok {
		return Suite{}, compileError(ErrorInvalidIntent, scenario.Root.source,
			fmt.Sprintf("property %q has no generated monitor", plan.property))
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return Suite{}, fmt.Errorf("digest semantic catalog: %w", err)
	}
	scenarioDigest, err := digestScenario(scenario, plan)
	if err != nil {
		return Suite{}, err
	}

	experiments := make([]protocolexperiment.Experiment, len(enumerated.paths))
	digests := make([]string, len(enumerated.paths))
	modelReplay := ModelReplay{Status: ModelReplayNotSupported, LiveOnlyActions: []protocolcatalog.ActionKind{}}
	for index, path := range enumerated.paths {
		experiment, buildErr := buildExperiment(scenario, plan, target, monitor, catalog, composition, catalogHash, path, index, len(enumerated.paths))
		if buildErr != nil {
			return Suite{}, buildErr
		}
		checked, replayErr := replayExperimentPath(&experiment, scenario.Root.source)
		if replayErr != nil {
			return Suite{}, replayErr
		}
		if err := experiment.Validate(); err != nil {
			return Suite{}, compileError(ErrorInvalidIntent, Source{},
				"compiled experiment is invalid: "+err.Error())
		}
		modelReplay = mergeModelReplay(modelReplay, checked)
		experiments[index] = experiment
		digests[index], err = experiment.Digest()
		if err != nil {
			return Suite{}, fmt.Errorf("digest compiled experiment: %w", err)
		}
	}

	constraints := make([]protocolexperiment.OrderConstraint, len(plan.edges))
	for index, edge := range plan.edges {
		constraints[index] = protocolexperiment.OrderConstraint{Before: edge.before, After: edge.after, Relation: edge.relation}
	}
	addedKinds := addedActionKinds(plan, enumerated.paths[0])
	explain := Explain{
		FormatVersion:    ExplainFormatVersion,
		Scenario:         scenario.Identifier,
		ScenarioDigest:   scenarioDigest,
		CatalogHash:      catalogHash,
		Target:           scenario.Target,
		Property:         plan.property,
		AddedActionKinds: addedKinds,
		Constraints:      constraints,
		Identities:       identities,
		Paths:            enumerated.paths,
		Omissions:        append([]protocolcatalog.ProjectionOmission(nil), target.Omissions...),
		ModelReplay:      modelReplay,
		Enumeration: Enumeration{
			Mode:        map[bool]string{false: "one-path", true: "all-paths"}[plan.allPaths],
			States:      enumerated.states,
			Paths:       len(enumerated.paths),
			MaxPaths:    limits.MaxPaths,
			MaxActions:  limits.MaxActions,
			MaxStates:   limits.MaxStates,
			MemoryBytes: enumerated.memoryBytes,
		},
	}
	return Suite{
		FormatVersion: SuiteFormatVersion, ScenarioDigest: scenarioDigest,
		Experiments: experiments, Digests: digests, Explain: explain,
	}, nil
}

func replayExperimentPath(experiment *protocolexperiment.Experiment, source Source) (ModelReplay, error) {
	executionView, found, err := finite.DefaultAttemptExecutionView(*experiment)
	if err != nil {
		return ModelReplay{}, compileError(ErrorInvalidIntent, source,
			"load executable model view: "+err.Error())
	}
	if !found {
		for index := range experiment.Actions {
			if len(experiment.Actions[index].AllowedOutcomes) == 0 {
				experiment.Actions[index].AllowedOutcomes = []protocolexperiment.ActionOutcome{
					protocolexperiment.ActionOutcomeApplied,
				}
			}
		}
		return ModelReplay{
			Status: ModelReplayNotSupported, LiveOnlyActions: []protocolcatalog.ActionKind{},
			Reason: "No exact executable compiler view is registered for this target.",
		}, nil
	}
	attempts := make([]finite.AttemptRequest, len(experiment.Actions))
	for index, action := range experiment.Actions {
		identifier := protocolcatalog.ActionKind(action.Kind)
		outcomes := action.AllowedOutcomes
		if len(outcomes) == 0 {
			var mapped bool
			outcomes, mapped = executionView.Outcomes(identifier)
			if !mapped {
				return ModelReplay{}, compileError(ErrorInvalidIntent, source,
					fmt.Sprintf("action %q has no Lean-derived attempt outcomes", identifier))
			}
			experiment.Actions[index].AllowedOutcomes = outcomes
		}
		attempts[index] = finite.AttemptRequest{Action: identifier, Outcomes: outcomes}
	}
	replay, err := executionView.Replay(attempts)
	if err != nil {
		return ModelReplay{}, compileError(ErrorInvalidIntent, source,
			"replay compiled path through executable model: "+err.Error())
	}
	if !replay.Accepted {
		return ModelReplay{}, compileError(ErrorSemanticallyImpossible, source,
			fmt.Sprintf("executable model %q variant %q rejects action %q",
				executionView.CanonicalModel(), executionView.Variant(),
				replay.RejectedAction))
	}
	return ModelReplay{
		Status: ModelReplayChecked, CanonicalModel: executionView.CanonicalModel(),
		Variant: executionView.Variant(), LiveOnlyActions: replay.LiveOnlyActions,
	}, nil
}

func mergeModelReplay(left, right ModelReplay) ModelReplay {
	if right.Status == ModelReplayChecked || left.Status != ModelReplayChecked {
		left.Status = right.Status
		left.CanonicalModel = right.CanonicalModel
		left.Variant = right.Variant
		left.Reason = right.Reason
	}
	left.LiveOnlyActions = append(left.LiveOnlyActions, right.LiveOnlyActions...)
	slices.Sort(left.LiveOnlyActions)
	left.LiveOnlyActions = slices.Compact(left.LiveOnlyActions)
	return left
}

func validateLimits(limits Limits) error {
	if limits.MaxPaths <= 0 || limits.MaxActions <= 0 || limits.MaxStates <= 0 ||
		limits.MaxMemoryBytes <= 0 || limits.MaxTime <= 0 {
		return compileError(ErrorInvalidIntent, Source{}, "positive path, action, state, memory, and time limits are required")
	}
	return nil
}

func validateResources(resources []Resource, catalog protocolcatalog.Catalog) error {
	knownKinds := make(map[protocolcatalog.EntityKind]struct{}, len(catalog.Entities))
	for _, entity := range catalog.Entities {
		knownKinds[protocolcatalog.EntityKind(entity.Identifier)] = struct{}{}
	}
	seen := make(map[string]struct{}, len(resources))
	for _, resource := range resources {
		if resource.Identifier == "" {
			return compileError(ErrorInvalidIntent, Source{}, "resource identifier is required")
		}
		if _, duplicate := seen[resource.Identifier]; duplicate {
			return compileError(ErrorInvalidIntent, Source{}, fmt.Sprintf("duplicate resource %q", resource.Identifier))
		}
		seen[resource.Identifier] = struct{}{}
		if _, known := knownKinds[resource.Kind]; !known {
			return compileError(ErrorInvalidIntent, Source{}, fmt.Sprintf("unknown resource kind %q", resource.Kind))
		}
	}
	return nil
}

func compositionTarget(composition protocolcatalog.Composition, identifier protocolcatalog.TargetID) (protocolcatalog.TargetProjection, bool) {
	for _, target := range composition.Targets {
		if target.Identifier == identifier {
			return target, true
		}
	}
	return protocolcatalog.TargetProjection{}, false
}

func completeDependencies(plan *normalizedPlan, catalog protocolcatalog.Catalog, target protocolcatalog.TargetProjection) error {
	retained := make(map[string]struct{}, len(target.RetainedActions))
	for _, action := range target.RetainedActions {
		retained[action] = struct{}{}
	}
	byKind := make(map[protocolcatalog.ActionKind][]*normalizedAction)
	byID := make(map[string]*normalizedAction)
	for _, action := range plan.actions {
		if action.identifier == "" || action.kind == "" {
			return compileError(ErrorInvalidIntent, action.source, "action identifier and kind are required")
		}
		if _, duplicate := byID[action.identifier]; duplicate {
			return compileError(ErrorAmbiguousProducer, action.source, fmt.Sprintf("duplicate action identifier %q", action.identifier))
		}
		declaration, known := catalog.Action(string(action.kind))
		if !known {
			return compileError(ErrorInvalidIntent, action.source, fmt.Sprintf("unknown action kind %q", action.kind))
		}
		if _, allowed := retained[declaration.Identifier]; !allowed {
			return compileError(ErrorInvalidIntent, action.source,
				fmt.Sprintf("target %q does not retain action %q", target.Identifier, action.kind))
		}
		byID[action.identifier] = action
		byKind[action.kind] = append(byKind[action.kind], action)
	}

	var ensure func(*normalizedAction) error
	ensure = func(action *normalizedAction) error {
		declaration, _ := catalog.Action(string(action.kind))
		for _, dependencyID := range declaration.Dependencies {
			dependencyKind := protocolcatalog.ActionKind(dependencyID)
			candidates := byKind[dependencyKind]
			var dependency *normalizedAction
			switch len(candidates) {
			case 0:
				if _, allowed := retained[dependencyID]; !allowed {
					return compileError(ErrorInvalidIntent, action.source,
						fmt.Sprintf("target %q cannot complete dependency %q", target.Identifier, dependencyID))
				}
				identifier := "generated-" + dependencyID
				if _, collision := byID[identifier]; collision {
					return compileError(ErrorAmbiguousProducer, action.source, fmt.Sprintf("generated action %q collides", identifier))
				}
				dependency = &normalizedAction{
					identifier: identifier, kind: dependencyKind, generated: true,
					responseMode: protocolexperiment.ResponseSynchronous,
				}
				plan.actions = append(plan.actions, dependency)
				byID[identifier] = dependency
				byKind[dependencyKind] = []*normalizedAction{dependency}
				if err := ensure(dependency); err != nil {
					return err
				}
			case 1:
				dependency = candidates[0]
				if err := ensure(dependency); err != nil {
					return err
				}
			default:
				return compileError(ErrorAmbiguousProducer, action.source,
					fmt.Sprintf("action %q has %d candidate dependencies of kind %q", action.identifier, len(candidates), dependencyKind))
			}
			plan.edges = append(plan.edges, normalizedEdge{before: dependency.identifier, after: action.identifier, relation: protocolexperiment.OrderSemantic})
		}
		return nil
	}
	initial := append([]*normalizedAction(nil), plan.actions...)
	for _, action := range initial {
		if err := ensure(action); err != nil {
			return err
		}
	}
	return nil
}

func compileBindings(plan *normalizedPlan, catalog protocolcatalog.Catalog) ([]IdentityRecord, error) {
	byID := make(map[string]*normalizedAction, len(plan.actions))
	for _, action := range plan.actions {
		byID[action.identifier] = action
	}
	bound := make(map[string]IdentityRecord, len(plan.bindings))
	for _, binding := range plan.bindings {
		symbol := binding.intent.symbol
		projection := binding.intent.projection
		if symbol.Name == "" || symbol.Type == "" {
			return nil, compileError(ErrorInvalidIntent, binding.source, "binding symbol and type are required")
		}
		if _, duplicate := bound[symbol.Name]; duplicate {
			return nil, compileError(ErrorRebind, binding.source, fmt.Sprintf("symbol %q is assigned more than once", symbol.Name))
		}
		producer, exists := byID[projection.ProducerAction]
		if !exists {
			return nil, compileError(ErrorAmbiguousProducer, binding.source,
				fmt.Sprintf("binding producer %q does not identify one action", projection.ProducerAction))
		}
		declaration, _ := catalog.Action(string(producer.kind))
		var declared *protocolcatalog.ProjectionDeclaration
		for index := range declaration.Projections {
			if declaration.Projections[index].Name == projection.Name {
				declared = &declaration.Projections[index]
				break
			}
		}
		if declared == nil {
			return nil, compileError(ErrorMissingProjection, binding.source,
				fmt.Sprintf("action %q does not project %q", producer.identifier, projection.Name))
		}
		if string(symbol.Type) != declared.Type || projection.Type != symbol.Type {
			return nil, compileError(ErrorTypeMismatch, binding.source,
				fmt.Sprintf("symbol %q type %q does not match projection type %q", symbol.Name, symbol.Type, declared.Type))
		}
		producer.bindings = append(producer.bindings, protocolexperiment.Binding{
			Symbol: symbol.Name, Type: string(symbol.Type), Projection: projection.Name,
		})
		bound[symbol.Name] = IdentityRecord{
			Symbol: symbol.Name, Type: string(symbol.Type), ProducerAction: producer.identifier, Projection: projection.Name,
		}
	}

	for _, action := range plan.actions {
		for _, argument := range action.arguments {
			for _, symbol := range valueSymbols(argument.Value) {
				record, exists := bound[symbol]
				if !exists {
					return nil, compileError(ErrorInvalidIntent, action.source, fmt.Sprintf("action %q references unbound symbol %q", action.identifier, symbol))
				}
				if record.ProducerAction != action.identifier {
					plan.edges = append(plan.edges, normalizedEdge{
						before: record.ProducerAction, after: action.identifier, relation: protocolexperiment.OrderRuntimeCausal,
					})
					record.ConsumerActions = append(record.ConsumerActions, action.identifier)
					bound[symbol] = record
				}
			}
		}
	}
	records := make([]IdentityRecord, 0, len(bound))
	for _, record := range bound {
		slices.Sort(record.ConsumerActions)
		record.ConsumerActions = slices.Compact(record.ConsumerActions)
		records = append(records, record)
	}
	slices.SortFunc(records, func(left, right IdentityRecord) int { return stringCompare(left.Symbol, right.Symbol) })
	return records, nil
}

func valueSymbols(value protocolexperiment.Value) []string {
	if value.Type == protocolexperiment.ValueSymbol && value.Text != nil {
		return []string{*value.Text}
	}
	var symbols []string
	for _, element := range value.Elements {
		symbols = append(symbols, valueSymbols(element)...)
	}
	for _, field := range value.Fields {
		symbols = append(symbols, valueSymbols(field.Value)...)
	}
	return symbols
}

func buildExperiment(
	scenario Scenario,
	plan *normalizedPlan,
	target protocolcatalog.TargetProjection,
	monitor protocolmonitor.MonitorProgram,
	catalog protocolcatalog.Catalog,
	composition protocolcatalog.Composition,
	catalogHash string,
	path []string,
	pathIndex int,
	pathCount int,
) (protocolexperiment.Experiment, error) {
	actionsByID := make(map[string]*normalizedAction, len(plan.actions))
	for _, action := range plan.actions {
		actionsByID[action.identifier] = action
	}
	actions := make([]protocolexperiment.Action, len(path))
	for index, identifier := range path {
		action := actionsByID[identifier]
		declaration, _ := catalog.Action(string(action.kind))
		capabilities := make([]string, len(declaration.RequiredCapabilities))
		for capabilityIndex, capability := range declaration.RequiredCapabilities {
			capabilities[capabilityIndex] = string(capability)
		}
		actions[index] = protocolexperiment.Action{
			Identifier:           identifier,
			Kind:                 string(action.kind),
			AllowedOutcomes:      append([]protocolexperiment.ActionOutcome(nil), action.allowedOutcomes...),
			Arguments:            append([]protocolexperiment.NamedValue(nil), action.arguments...),
			Bindings:             append([]protocolexperiment.Binding(nil), action.bindings...),
			RequiredCapabilities: capabilities,
			ResponseMode:         action.responseMode,
			MaxBlockNanos:        action.maxBlockNanos,
		}
	}
	properties := make(map[string]protocolcatalog.PropertyDeclaration, len(catalog.Properties))
	for _, property := range catalog.Properties {
		properties[property.Identifier] = property
	}
	property := properties[string(plan.property)]
	resources := make([]protocolexperiment.Resource, len(scenario.Resources))
	for index, resource := range scenario.Resources {
		resources[index] = protocolexperiment.Resource{Identifier: resource.Identifier, Kind: string(resource.Kind)}
	}
	policies := make([]protocolexperiment.Policy, len(plan.policies))
	for index, policy := range plan.policies {
		policies[index] = protocolexperiment.Policy{Identifier: policy.identifier, Kind: string(protocolcatalog.PolicyKindDuring), Scope: append([]string(nil), policy.scope...)}
	}
	faultDeclarations := make(map[protocolcatalog.FaultKind]protocolcatalog.FaultDeclaration, len(catalog.Faults))
	for _, declaration := range catalog.Faults {
		faultDeclarations[protocolcatalog.FaultKind(declaration.Identifier)] = declaration
	}
	faults := make([]protocolexperiment.Fault, len(plan.faults))
	policyScopes := make(map[string][]string, len(plan.policies))
	for _, policy := range plan.policies {
		policyScopes[policy.identifier] = policy.scope
	}
	for index, fault := range plan.faults {
		declaration, known := faultDeclarations[fault.kind]
		if !known {
			return protocolexperiment.Experiment{}, compileError(ErrorInvalidIntent, Source{}, fmt.Sprintf("unknown fault kind %q", fault.kind))
		}
		capabilities := make([]string, len(declaration.RequiredCapabilities))
		for capabilityIndex, capability := range declaration.RequiredCapabilities {
			capabilities[capabilityIndex] = string(capability)
		}
		resourceScope := make([]string, len(scenario.Resources))
		for resourceIndex, resource := range scenario.Resources {
			resourceScope[resourceIndex] = resource.Identifier
		}
		intervalScope := policyScopes[fault.policy]
		if len(intervalScope) == 0 {
			intervalScope = path
		}
		scope := cloneFaultScope(fault.scope)
		if len(scope.Resources) == 0 {
			scope.Resources = append([]string(nil), resourceScope...)
		}
		if len(scope.Participants) == 0 {
			scope.Participants = append([]string(nil), resourceScope...)
		}
		if len(scope.Attempts) == 0 {
			scope.Attempts = []int{1}
		}
		occurrence := fault.occurrence
		if occurrence.First == 0 && occurrence.Count == 0 {
			occurrence = protocolexperiment.FaultOccurrence{First: 1, Count: 1}
		}
		compiledFault := protocolexperiment.Fault{
			Identifier: fault.identifier, Kind: string(fault.kind), Policy: fault.policy,
			SafetyClass: declaration.SafetyClass,
			Scope:       scope,
			Occurrence:  occurrence,
			Interval: protocolexperiment.FaultInterval{
				StartAction: intervalScope[0], StopAction: intervalScope[len(intervalScope)-1],
			},
			Arguments: append([]protocolexperiment.NamedValue(nil), fault.arguments...), RequiredCapabilities: capabilities,
		}
		if fault.configured != nil {
			compiledFault = *fault.configured
			compiledFault.Identifier = fault.identifier
			compiledFault.Kind = string(fault.kind)
			compiledFault.Policy = fault.policy
			compiledFault.Interval = protocolexperiment.FaultInterval{
				StartAction: intervalScope[0], StopAction: intervalScope[len(intervalScope)-1],
			}
		}
		faults[index] = compiledFault
	}
	checkpoints := monitorCheckpoints(monitor)
	order := make([]protocolexperiment.OrderConstraint, len(plan.edges))
	for index, edge := range plan.edges {
		order[index] = protocolexperiment.OrderConstraint{Before: edge.before, After: edge.after, Relation: edge.relation}
	}
	modules := make([]string, len(target.Modules))
	for index, module := range target.Modules {
		modules[index] = string(module)
	}
	experiment := protocolexperiment.Experiment{
		FormatVersion: protocolcatalog.FormatVersion,
		ExperimentID:  fmt.Sprintf("%s-path-%03d", scenario.Identifier, pathIndex+1),
		Model: protocolexperiment.Model{
			Modules: modules, SourceRevision: "umpire3/compiler/v1", SemanticHash: composition.SemanticHash,
			CatalogHash: catalogHash, LeanVersion: protocolcatalog.LeanVersion,
		},
		Property: protocolexperiment.Property{
			Identifier: string(plan.property), StatementHash: property.StatementHash, Claim: "implementation-conformance",
		},
		Scope: protocolexperiment.Scope{
			Bounds:      protocolexperiment.Bounds{MaxDepth: len(actions), MaxResults: max(pathCount, len(checkpoints))},
			Assumptions: []protocolexperiment.Assumption{}, Strategy: map[bool]string{false: "deterministic-one-path", true: "complete-linearizations"}[plan.allPaths],
		},
		Resources: resources, Actions: actions, Policies: policies, Faults: faults, Order: order,
		Checkpoints: checkpoints,
		Provenance:  protocolexperiment.Provenance{Kind: "bounded-exploration", ProofManifest: "composition:" + string(target.Identifier)},
		Retention:   protocolexperiment.Retention{RedactionClass: "semantic-only", MaxArtifactBytes: 1 << 20},
	}
	return experiment, nil
}

func monitorCheckpoints(monitor protocolmonitor.MonitorProgram) []protocolexperiment.Checkpoint {
	seen := make(map[protocolcatalog.ObservationID]struct{})
	var observations []protocolcatalog.ObservationID
	var collect func(protocolmonitor.MonitorExpression)
	collect = func(expression protocolmonitor.MonitorExpression) {
		if expression.Operation == protocolmonitor.MonitorObservation {
			if _, exists := seen[expression.Observation]; !exists {
				seen[expression.Observation] = struct{}{}
				observations = append(observations, expression.Observation)
			}
		}
		for _, child := range expression.Children {
			collect(child)
		}
	}
	collect(monitor.Expression)
	slices.Sort(observations)
	ordering := "none"
	if slices.Contains(monitor.Evidence, protocolcatalog.EvidenceIDCausal) {
		ordering = "causal"
	} else if slices.Contains(monitor.Evidence, protocolcatalog.EvidenceIDSourceSequence) {
		ordering = "source-sequence"
	}
	checkpoints := make([]protocolexperiment.Checkpoint, len(observations))
	for index, observation := range observations {
		checkpoints[index] = protocolexperiment.Checkpoint{
			Identifier: "observe-" + string(observation), Observation: string(observation),
			Ordering: ordering, OmissionPolicy: "required",
		}
	}
	return checkpoints
}

func addedActionKinds(plan *normalizedPlan, firstPath []string) []string {
	byID := make(map[string]*normalizedAction, len(plan.actions))
	for _, action := range plan.actions {
		byID[action.identifier] = action
	}
	var kinds []string
	for _, identifier := range firstPath {
		if action := byID[identifier]; action.generated {
			kinds = append(kinds, string(action.kind))
		}
	}
	return kinds
}

func digestScenario(scenario Scenario, plan *normalizedPlan) (string, error) {
	type actionDigest struct {
		Identifier    string                          `json:"identifier"`
		Kind          protocolcatalog.ActionKind      `json:"kind"`
		Arguments     []protocolexperiment.NamedValue `json:"arguments"`
		Bindings      []protocolexperiment.Binding    `json:"bindings"`
		ResponseMode  protocolexperiment.ResponseMode `json:"responseMode"`
		MaxBlockNanos int64                           `json:"maxBlockNanos,omitempty"`
	}
	type edgeDigest struct {
		Before   string                           `json:"before"`
		After    string                           `json:"after"`
		Relation protocolexperiment.OrderRelation `json:"relation"`
	}
	type policyDigest struct {
		Identifier string   `json:"identifier"`
		Scope      []string `json:"scope"`
	}
	type faultDigest struct {
		Identifier string                             `json:"identifier"`
		Kind       protocolcatalog.FaultKind          `json:"kind"`
		Policy     string                             `json:"policy"`
		Arguments  []protocolexperiment.NamedValue    `json:"arguments"`
		Scope      protocolexperiment.FaultScope      `json:"scope"`
		Occurrence protocolexperiment.FaultOccurrence `json:"occurrence"`
		Configured *protocolexperiment.Fault          `json:"configured,omitempty"`
	}
	type scenarioDigestInput struct {
		Identifier string                     `json:"identifier"`
		Target     protocolcatalog.TargetID   `json:"target"`
		Resources  []Resource                 `json:"resources"`
		Property   protocolcatalog.PropertyID `json:"property"`
		Actions    []actionDigest             `json:"actions"`
		Edges      []edgeDigest               `json:"edges"`
		Policies   []policyDigest             `json:"policies"`
		Faults     []faultDigest              `json:"faults"`
		AllPaths   bool                       `json:"allPaths"`
	}
	actions := make([]actionDigest, len(plan.actions))
	for index, action := range plan.actions {
		actions[index] = actionDigest{
			Identifier: action.identifier, Kind: action.kind, Arguments: action.arguments, Bindings: action.bindings,
			ResponseMode: action.responseMode, MaxBlockNanos: action.maxBlockNanos,
		}
	}
	edges := make([]edgeDigest, len(plan.edges))
	for index, edge := range plan.edges {
		edges[index] = edgeDigest{Before: edge.before, After: edge.after, Relation: edge.relation}
	}
	policies := make([]policyDigest, len(plan.policies))
	for index, policy := range plan.policies {
		policies[index] = policyDigest{Identifier: policy.identifier, Scope: policy.scope}
	}
	faults := make([]faultDigest, len(plan.faults))
	for index, fault := range plan.faults {
		faults[index] = faultDigest{
			Identifier: fault.identifier, Kind: fault.kind, Policy: fault.policy, Arguments: fault.arguments,
			Scope: fault.scope, Occurrence: fault.occurrence,
			Configured: fault.configured,
		}
	}
	input := scenarioDigestInput{
		Identifier: scenario.Identifier, Target: scenario.Target, Resources: scenario.Resources,
		Property: plan.property, Actions: actions, Edges: edges, Policies: policies, Faults: faults,
		AllPaths: plan.allPaths,
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("encode sparse scenario: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func (s Suite) CanonicalJSON() ([]byte, error) {
	if s.FormatVersion != SuiteFormatVersion || !strings.HasPrefix(s.ScenarioDigest, "sha256:") ||
		len(s.Experiments) == 0 || len(s.Experiments) != len(s.Digests) {
		return nil, errors.New("complete compiler suite is required")
	}
	for index, experiment := range s.Experiments {
		if err := experiment.Validate(); err != nil {
			return nil, fmt.Errorf("experiment %d: %w", index, err)
		}
		digest, err := experiment.Digest()
		if err != nil {
			return nil, err
		}
		if digest != s.Digests[index] {
			return nil, fmt.Errorf("experiment %d digest mismatch", index)
		}
	}
	encoded, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("encode compiler suite: %w", err)
	}
	return encoded, nil
}
