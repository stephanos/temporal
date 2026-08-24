package execution

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"go.temporal.io/server/tools/umpire3/checker/finite"
	evidencegraph "go.temporal.io/server/tools/umpire3/execution/evidence"
	umpire3fault "go.temporal.io/server/tools/umpire3/execution/fault"
	"go.temporal.io/server/tools/umpire3/execution/observation"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
	protocolexecution "go.temporal.io/server/tools/umpire3/protocol/execution"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
	protocolmonitor "go.temporal.io/server/tools/umpire3/protocol/monitor"
)

type ClaimKind = protocolexecution.ClaimKind
type OutcomeKind = protocolexecution.OutcomeKind

const (
	ClaimConforming      = protocolexecution.ClaimConforming
	ClaimViolating       = protocolexecution.ClaimViolating
	ClaimUnsupported     = protocolexecution.ClaimUnsupported
	ClaimInconclusive    = protocolexecution.ClaimInconclusive
	ClaimEvidenceFailure = protocolexecution.ClaimEvidenceFailure
	OutcomeRecovered     = protocolexecution.OutcomeRecovered
	OutcomeDegraded      = protocolexecution.OutcomeDegraded
	OutcomeFlagged       = protocolexecution.OutcomeFlagged
	OutcomeUnreached     = protocolexecution.OutcomeUnreached
	ResultFormatVersion  = "umpire3/runtime-result/v3"
)

type Limits struct {
	PrepareTimeout      time.Duration
	ActionTimeout       time.Duration
	ObserveTimeout      time.Duration
	FaultTimeout        time.Duration
	CleanupTimeout      time.Duration
	MaxActions          int
	MaxObservations     int
	MaxResources        int
	MaxEvidenceBytes    int64
	MaxActionsPerSecond int
}

type Request struct {
	Experiment  protocolexperiment.Experiment
	Environment Factory
	Limits      Limits
	// AllowRestrictedFaults is an explicit opt-in in addition to profile capability and authority checks.
	AllowRestrictedFaults bool
}

type Claim struct {
	Kind       ClaimKind `json:"kind"`
	Property   string    `json:"property"`
	Checkpoint string    `json:"checkpoint,omitempty"`
	Reason     string    `json:"reason,omitempty"`
}

type ActionResult struct {
	Identifier string         `json:"identifier"`
	Kind       string         `json:"kind"`
	Evidence   ActionEvidence `json:"evidence"`
	Error      string         `json:"error,omitempty"`
}

type CheckpointResult struct {
	Identifier string `json:"identifier"`
	Satisfied  bool   `json:"satisfied"`
	Qualified  bool   `json:"qualified"`
	Reason     string `json:"reason,omitempty"`
}

type FaultResult struct {
	Identifier      string `json:"identifier"`
	Kind            string `json:"kind"`
	SourceIdentity  string `json:"sourceIdentity,omitempty"`
	Reference       string `json:"reference,omitempty"`
	EntityIdentity  string `json:"entityIdentity,omitempty"`
	Installed       bool   `json:"installed"`
	Activated       bool   `json:"activated"`
	Realized        bool   `json:"realized"`
	Released        bool   `json:"released"`
	CleanupComplete bool   `json:"cleanupComplete"`
	Error           string `json:"error,omitempty"`
}

type Result struct {
	FormatVersion    string                         `json:"formatVersion"`
	ExperimentDigest string                         `json:"experimentDigest"`
	ResultClass      protocolcatalog.ResultClass    `json:"resultClass"`
	TrustBadge       protocolcatalog.TrustBadge     `json:"trustBadge"`
	Environment      EnvironmentIdentity            `json:"environment"`
	Actions          []ActionResult                 `json:"actions"`
	Bindings         Bindings                       `json:"bindings"`
	Observations     []Observation                  `json:"observations"`
	Facts            []observation.Fact             `json:"facts,omitempty"`
	Faults           []FaultResult                  `json:"faults,omitempty"`
	Omissions        []string                       `json:"omissions"`
	Checkpoints      []CheckpointResult             `json:"checkpoints"`
	Evidence         evidencegraph.Graph            `json:"evidence"`
	EvidenceDigest   string                         `json:"evidenceDigest,omitempty"`
	Trace            *protocolchecker.SemanticTrace `json:"trace,omitempty"`
	Footprint        *umpire3fault.Report           `json:"footprint,omitempty"`
	Claim            Claim                          `json:"claim"`
	Outcome          protocolexecution.Outcome      `json:"outcome"`
	Cleanup          CleanupResult                  `json:"cleanup"`
}

func Run(ctx context.Context, request Request) (result Result, retErr error) {
	if err := request.Experiment.Validate(); err != nil {
		return Result{}, fmt.Errorf("validate experiment: %w", err)
	}
	attemptExecutionView, hasAttemptExecutionView, err :=
		finite.DefaultAttemptExecutionView(request.Experiment)
	if err != nil {
		return Result{}, fmt.Errorf("load Lean-derived attempt execution view: %w", err)
	}
	monitorCatalog, err := protocolmonitor.DefaultMonitorCatalog()
	if err != nil {
		return Result{}, fmt.Errorf("load monitor programs: %w", err)
	}
	monitor, ok := monitorCatalog.Program(protocolcatalog.PropertyID(request.Experiment.Property.Identifier))
	if !ok {
		return Result{}, fmt.Errorf("property %q has no generated monitor program", request.Experiment.Property.Identifier)
	}
	observationCatalog, err := observation.DefaultCatalog()
	if err != nil {
		return Result{}, fmt.Errorf("load observation programs: %w", err)
	}
	if request.Environment == nil {
		return Result{}, errors.New("environment is required")
	}
	digest, err := request.Experiment.Digest()
	if err != nil {
		return Result{}, err
	}
	limits := request.Limits.withDefaults(request.Experiment)
	if len(request.Experiment.Actions) > limits.MaxActions ||
		len(request.Experiment.Checkpoints) > limits.MaxObservations ||
		len(request.Experiment.Resources) > limits.MaxResources {
		return Result{}, errors.New("experiment exceeds runtime count budget")
	}

	capabilities := uniqueSortedCapabilities(request.Environment.Capabilities())
	result = Result{
		FormatVersion:    ResultFormatVersion,
		ExperimentDigest: digest,
		Environment:      EnvironmentIdentity{Capabilities: capabilities},
		Bindings:         make(Bindings),
		Claim: Claim{
			Kind:     ClaimInconclusive,
			Property: request.Experiment.Property.Identifier,
		},
	}
	defer finalizeAssurance(&result)
	defer finalizeOutcome(&result)
	defer func() { finalizeEvidenceGraph(&result, limits.MaxEvidenceBytes) }()
	if missing := missingCapabilities(request.Experiment, capabilities); len(missing) != 0 {
		result.Claim.Kind = ClaimUnsupported
		result.Claim.Reason = "missing capabilities: " + fmt.Sprint(missing)
		return result, nil
	}

	prepareCtx, cancelPrepare := context.WithTimeout(ctx, limits.PrepareTimeout)
	prepared, err := request.Environment.Prepare(prepareCtx, request.Experiment)
	cancelPrepare()
	session := prepared.Session
	if session != nil {
		defer func() {
			finalizeCleanup(&result, session, limits.CleanupTimeout)
		}()
	}
	if err != nil {
		result.Claim.Reason = "prepare environment: " + err.Error()
		return result, nil
	}
	if session == nil {
		result.Claim.Reason = "prepare environment returned no session"
		return result, nil
	}
	factSession, emitsFacts := session.(FactSession)
	if !emitsFacts {
		result.Claim.Reason = "prepared session has no fact observation interface"
		return result, nil
	}
	defer finalizeFootprint(&result, request.Environment, session)
	if err := prepared.Identity.Validate(); err != nil {
		result.Claim.Kind = ClaimUnsupported
		result.Claim.Reason = "invalid environment identity: " + err.Error()
		return result, nil
	}
	prepared.Identity.Capabilities = capabilities
	result.Environment = prepared.Identity

	faults, faultErr := prepareFaults(ctx, request, session, capabilities, limits, &result)
	if faults != nil {
		defer faults.cleanup(&result, limits.CleanupTimeout)
	}
	if faultErr != nil {
		if errors.Is(faultErr, errFaultRealizerUnavailable) {
			result.Claim.Kind = ClaimUnsupported
		}
		result.Claim.Reason = faultErr.Error()
		return result, nil
	}

	checkpointByID := make(map[string]protocolexperiment.Checkpoint, len(request.Experiment.Checkpoints))
	for _, checkpoint := range request.Experiment.Checkpoints {
		checkpointByID[checkpoint.Identifier] = checkpoint
	}
	observed := make(map[string]struct{}, len(checkpointByID))
	primaryFacts := make([]observation.Fact, 0, len(request.Experiment.Checkpoints))
	observe := func(identifier string) bool {
		if identifier == "" {
			return true
		}
		if _, exists := observed[identifier]; exists {
			return true
		}
		checkpoint := checkpointByID[identifier]
		observeCtx, cancelObserve := context.WithTimeout(ctx, limits.ObserveTimeout)
		var observedValue Observation
		var observeErr error
		var emittedFacts []observation.Fact
		emittedFacts, observeErr = factSession.ObserveFacts(observeCtx, checkpoint, result.Bindings)
		if observeErr == nil {
			primaryFacts = appendDistinctFacts(primaryFacts, emittedFacts)
			result.Facts = appendDistinctFacts(result.Facts, emittedFacts)
			program, exists := observationCatalog.Program(protocolcatalog.ObservationID(checkpoint.Observation))
			if !exists {
				observeErr = fmt.Errorf("observation %q has no generated interpreter program", checkpoint.Observation)
			} else {
				evaluation := program.Evaluate(primaryFacts)
				switch evaluation.Value {
				case observation.True, observation.False:
					observedValue, observeErr = interpretedObservation(checkpoint, evaluation, primaryFacts)
					if observeErr != nil {
						result.Claim.Kind = ClaimEvidenceFailure
						result.Claim.Reason = "interpret typed observation: " + observeErr.Error()
						observeErr = errors.New(result.Claim.Reason)
					} else if !factsShareEntityIdentity(primaryFacts, observedValue) {
						result.Claim.Kind = ClaimEvidenceFailure
						result.Claim.Reason = "primary fact set combines multiple source or entity identities"
						observeErr = errors.New(result.Claim.Reason)
					}
				case observation.Unknown:
					observeErr = errors.New("typed observation remains unknown; required evidence or window closure is missing")
				case observation.Conflict:
					result.Claim.Kind = ClaimEvidenceFailure
					observeErr = fmt.Errorf("typed observation evidence conflicts: %v", evaluation.Support)
				default:
					result.Claim.Kind = ClaimEvidenceFailure
					observeErr = fmt.Errorf("typed observation returned invalid value %q", evaluation.Value)
				}
			}
		}
		cancelObserve()
		if observeErr != nil {
			result.Omissions = append(result.Omissions, identifier+": "+observeErr.Error())
			if checkpoint.OmissionPolicy == "optional" {
				result.Checkpoints = append(result.Checkpoints, CheckpointResult{
					Identifier: identifier,
					Qualified:  true,
					Reason:     "optional observation omitted",
				})
				observed[identifier] = struct{}{}
				return true
			}
			if result.Claim.Reason == "" {
				result.Claim.Reason = observeErr.Error()
			}
			result.Checkpoints = append(result.Checkpoints, CheckpointResult{
				Identifier: identifier,
				Reason:     observeErr.Error(),
			})
			return false
		}
		if observedValue.ObservedAtUnixNano == 0 {
			observedValue.ObservedAtUnixNano = time.Now().UnixNano()
		}
		result.Observations = append(result.Observations, observedValue)
		qualified, reason := qualifyObservation(checkpoint, observedValue, monitor.Evidence)
		if qualified {
			qualified, reason = appendCorroboratingFactObservations(
				ctx, session, checkpoint, result.Bindings, limits.ObserveTimeout, &result,
				observedValue, monitor.Evidence, observationCatalog,
			)
		}
		result.Checkpoints = append(result.Checkpoints, CheckpointResult{
			Identifier: identifier,
			Satisfied:  observedValue.Satisfied,
			Qualified:  qualified,
			Reason:     reason,
		})
		observed[identifier] = struct{}{}
		if !qualified {
			result.Omissions = append(result.Omissions, identifier+": "+reason)
			return false
		}
		facts := make([]protocolmonitor.ObservedFact, 0, len(result.Observations))
		for _, observed := range result.Observations {
			facts = append(facts, protocolmonitor.ObservedFact{
				Observation: protocolcatalog.ObservationID(observed.Kind), Value: observed.Satisfied,
			})
		}
		evaluation, evaluateErr := observation.EvaluateMonitor(monitor, facts)
		if evaluateErr == nil && evaluation.Complete && !evaluation.Satisfied {
			result.Claim.Kind = ClaimViolating
			result.Claim.Reason = "generated property monitor rejected qualified evidence"
			result.Claim.Checkpoint = contradictionCheckpoint(result.Observations, evaluation.Contradictions)
			return false
		}
		return true
	}

	completeEvidence := true
	var evidenceBytes int64
	observedAttempts := make([]finite.ObservedAttempt, 0, len(request.Experiment.Actions))
	defer func() {
		finalizeSemanticTrace(
			&result, request.Experiment, attemptExecutionView, hasAttemptExecutionView, observedAttempts)
	}()
	declaredSymbols := make(map[string]struct{})
	var previousAction time.Time
	for _, action := range request.Experiment.Actions {
		for _, binding := range action.Bindings {
			declaredSymbols[binding.Symbol] = struct{}{}
		}
	}
	for _, action := range request.Experiment.Actions {
		if faults != nil {
			faultCtx, cancelFault := context.WithTimeout(ctx, limits.FaultTimeout)
			faultErr := faults.beforeAction(faultCtx, action.Identifier, &result)
			cancelFault()
			if faultErr != nil {
				result.Claim.Reason = faultErr.Error()
				completeEvidence = false
				break
			}
		}
		if !observe(action.PreCheckpoint) {
			completeEvidence = false
			break
		}
		if missing := missingRuntimeBindings(action.Arguments, result.Bindings); len(missing) != 0 {
			reason := "action references ungrounded bindings: " + fmt.Sprint(missing)
			result.Actions = append(result.Actions, ActionResult{
				Identifier: action.Identifier, Kind: action.Kind, Error: reason,
			})
			result.Claim.Reason = reason
			completeEvidence = false
			break
		}
		if err := waitForActionRate(ctx, &previousAction, limits.MaxActionsPerSecond); err != nil {
			reason := "wait for action rate budget: " + err.Error()
			result.Actions = append(result.Actions, ActionResult{
				Identifier: action.Identifier, Kind: action.Kind, Error: reason,
			})
			result.Claim.Reason = reason
			completeEvidence = false
			break
		}
		actionCtx, cancelAction := context.WithTimeout(ctx, limits.ActionTimeout)
		evidence, actionErr := session.Realize(actionCtx, action, result.Bindings)
		cancelAction()
		actionResult := ActionResult{Identifier: action.Identifier, Kind: action.Kind, Evidence: evidence}
		if actionErr != nil {
			actionResult.Error = actionErr.Error()
			result.Actions = append(result.Actions, actionResult)
			result.Claim.Reason = "realize action " + action.Identifier + ": " + actionErr.Error()
			completeEvidence = false
			break
		}
		if evidence.Outcome == "" {
			actionResult.Error = "observed action outcome is missing"
			result.Actions = append(result.Actions, actionResult)
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = actionResult.Error
			completeEvidence = false
			break
		}
		if !slices.Contains(action.AllowedOutcomes, evidence.Outcome) {
			actionResult.Error = fmt.Sprintf("observed action outcome %q is not allowed", evidence.Outcome)
			result.Actions = append(result.Actions, actionResult)
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = actionResult.Error
			completeEvidence = false
			break
		}
		observedAttempts = append(observedAttempts, finite.ObservedAttempt{
			Action: protocolcatalog.ActionKind(action.Kind), Outcome: evidence.Outcome,
		})
		if hasAttemptExecutionView {
			replay, replayErr := attemptExecutionView.ReplayObserved(observedAttempts)
			if replayErr != nil {
				actionResult.Error = "replay observed action outcomes: " + replayErr.Error()
			} else if !replay.Accepted {
				actionResult.Error = fmt.Sprintf(
					"canonical attempt replay rejects action %q outcome %q",
					replay.RejectedAction, replay.RejectedOutcome)
			}
			if actionResult.Error != "" {
				result.Actions = append(result.Actions, actionResult)
				result.Claim.Kind = ClaimEvidenceFailure
				result.Claim.Reason = actionResult.Error
				completeEvidence = false
				break
			}
		}
		evidenceBytes += actionEvidenceSize(evidence)
		if evidenceBytes > limits.MaxEvidenceBytes {
			actionResult.Error = "evidence budget exhausted"
			result.Actions = append(result.Actions, actionResult)
			result.Omissions = append(result.Omissions, "evidence budget exhausted")
			result.Claim.Reason = "evidence budget exhausted"
			completeEvidence = false
			break
		}
		bindingsComplete := true
		for _, binding := range action.Bindings {
			concrete, grounded := evidence.GroundedBindings[binding.Symbol]
			if !grounded || concrete == "" {
				actionResult.Error = "action did not ground declared projection " + binding.Projection
				result.Omissions = append(result.Omissions, "missing binding: "+binding.Symbol)
				bindingsComplete = false
				break
			}
			symbolic := binding.Symbol
			if existing, exists := result.Bindings[symbolic]; exists && existing != concrete {
				result.Omissions = append(result.Omissions, "conflicting binding: "+symbolic)
				actionResult.Error = "action returned a conflicting binding for " + symbolic
				bindingsComplete = false
				break
			}
			result.Bindings[symbolic] = concrete
		}
		for symbolic, concrete := range evidence.GroundedBindings {
			if _, declared := declaredSymbols[symbolic]; declared || concrete == "" {
				continue
			}
			result.Bindings[symbolic] = concrete
		}
		result.Actions = append(result.Actions, actionResult)
		if !bindingsComplete {
			result.Claim.Reason = actionResult.Error
			completeEvidence = false
			break
		}
		if !observe(action.PostCheckpoint) {
			completeEvidence = false
			break
		}
		if faults != nil {
			faultCtx, cancelFault := context.WithTimeout(ctx, limits.FaultTimeout)
			faultErr := faults.afterAction(faultCtx, action.Identifier, &result)
			cancelFault()
			if faultErr != nil {
				result.Claim.Reason = faultErr.Error()
				completeEvidence = false
				break
			}
		}
	}
	if completeEvidence {
		for _, checkpoint := range request.Experiment.Checkpoints {
			if !observe(checkpoint.Identifier) {
				completeEvidence = false
				break
			}
		}
	}
	if completeEvidence && len(observed) == len(request.Experiment.Checkpoints) {
		facts := make([]protocolmonitor.ObservedFact, 0, len(result.Observations))
		for _, observed := range result.Observations {
			facts = append(facts, protocolmonitor.ObservedFact{
				Observation: protocolcatalog.ObservationID(observed.Kind),
				Value:       observed.Satisfied,
			})
		}
		evaluation, evaluateErr := observation.EvaluateMonitor(monitor, facts)
		switch {
		case evaluateErr != nil:
			result.Claim.Kind = ClaimInconclusive
			result.Claim.Reason = "evaluate property monitor: " + evaluateErr.Error()
		case !evaluation.Complete:
			result.Claim.Kind = ClaimInconclusive
			result.Claim.Reason = "property monitor is missing observations: " + fmt.Sprint(evaluation.Missing)
		case evaluation.Satisfied:
			result.Claim.Kind = ClaimConforming
			result.Claim.Reason = "generated property monitor accepted qualified evidence"
		default:
			result.Claim.Kind = ClaimViolating
			result.Claim.Reason = "generated property monitor rejected qualified evidence"
			result.Claim.Checkpoint = contradictionCheckpoint(result.Observations, evaluation.Contradictions)
		}
	} else if result.Claim.Kind != ClaimViolating && result.Claim.Kind != ClaimEvidenceFailure {
		result.Claim.Kind = ClaimInconclusive
		if result.Claim.Reason == "" {
			result.Claim.Reason = "required evidence is incomplete"
		}
	}
	return result, nil
}
