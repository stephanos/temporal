package execution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"time"

	evidencegraph "go.temporal.io/server/tests/umpire3/evidence"
	umpire3fault "go.temporal.io/server/tests/umpire3/fault"
	"go.temporal.io/server/tests/umpire3/observation"
	"go.temporal.io/server/tests/umpire3/protocol"
)

type ClaimKind = protocol.ClaimKind
type OutcomeKind = protocol.OutcomeKind

const (
	ClaimConforming      = protocol.ClaimConforming
	ClaimViolating       = protocol.ClaimViolating
	ClaimUnsupported     = protocol.ClaimUnsupported
	ClaimInconclusive    = protocol.ClaimInconclusive
	ClaimEvidenceFailure = protocol.ClaimEvidenceFailure
	OutcomeRecovered     = protocol.OutcomeRecovered
	OutcomeDegraded      = protocol.OutcomeDegraded
	OutcomeFlagged       = protocol.OutcomeFlagged
	OutcomeUnreached     = protocol.OutcomeUnreached
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
	Experiment  protocol.Experiment
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
	FormatVersion    string                  `json:"formatVersion"`
	ExperimentDigest string                  `json:"experimentDigest"`
	ResultClass      protocol.ResultClass    `json:"resultClass"`
	TrustBadge       protocol.TrustBadge     `json:"trustBadge"`
	Environment      EnvironmentIdentity     `json:"environment"`
	Actions          []ActionResult          `json:"actions"`
	Bindings         Bindings                `json:"bindings"`
	Observations     []Observation           `json:"observations"`
	Facts            []observation.Fact      `json:"facts,omitempty"`
	Faults           []FaultResult           `json:"faults,omitempty"`
	Omissions        []string                `json:"omissions"`
	Checkpoints      []CheckpointResult      `json:"checkpoints"`
	Evidence         evidencegraph.Graph     `json:"evidence"`
	EvidenceDigest   string                  `json:"evidenceDigest,omitempty"`
	Trace            *protocol.SemanticTrace `json:"trace,omitempty"`
	Footprint        *umpire3fault.Report    `json:"footprint,omitempty"`
	Claim            Claim                   `json:"claim"`
	Outcome          protocol.Outcome        `json:"outcome"`
	Cleanup          CleanupResult           `json:"cleanup"`
}

func Run(ctx context.Context, request Request) (result Result, retErr error) {
	if err := request.Experiment.Validate(); err != nil {
		return Result{}, fmt.Errorf("validate experiment: %w", err)
	}
	attemptExecutionView, hasAttemptExecutionView, err :=
		protocol.DefaultAttemptExecutionView(request.Experiment)
	if err != nil {
		return Result{}, fmt.Errorf("load Lean-derived attempt execution view: %w", err)
	}
	monitorCatalog, err := protocol.DefaultMonitorCatalog()
	if err != nil {
		return Result{}, fmt.Errorf("load monitor programs: %w", err)
	}
	monitor, ok := monitorCatalog.Program(protocol.PropertyID(request.Experiment.Property.Identifier))
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

	checkpointByID := make(map[string]protocol.Checkpoint, len(request.Experiment.Checkpoints))
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
			program, exists := observationCatalog.Program(protocol.ObservationID(checkpoint.Observation))
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
		facts := make([]protocol.ObservedFact, 0, len(result.Observations))
		for _, observed := range result.Observations {
			facts = append(facts, protocol.ObservedFact{
				Observation: protocol.ObservationID(observed.Kind), Value: observed.Satisfied,
			})
		}
		evaluation, evaluateErr := monitor.Evaluate(facts)
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
	observedAttempts := make([]protocol.ObservedAttempt, 0, len(request.Experiment.Actions))
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
		observedAttempts = append(observedAttempts, protocol.ObservedAttempt{
			Action: protocol.ActionKind(action.Kind), Outcome: evidence.Outcome,
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
		facts := make([]protocol.ObservedFact, 0, len(result.Observations))
		for _, observation := range result.Observations {
			facts = append(facts, protocol.ObservedFact{
				Observation: protocol.ObservationID(observation.Kind),
				Value:       observation.Satisfied,
			})
		}
		evaluation, evaluateErr := monitor.Evaluate(facts)
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

func finalizeAssurance(result *Result) {
	result.DeriveAssurance()
}

func (r *Result) DeriveAssurance() {
	r.TrustBadge = protocol.TrustBadgeTestedInstance
	switch r.Claim.Kind {
	case ClaimConforming:
		r.ResultClass = protocol.ResultClassImplementationConforming
	case ClaimViolating:
		r.ResultClass = protocol.ResultClassTraceWitness
	default:
		r.ResultClass = protocol.ResultClassUnknown
	}
}

func (r Result) ValidateAssurance() error {
	expected := Result{Claim: r.Claim}
	finalizeAssurance(&expected)
	if r.ResultClass != expected.ResultClass || r.TrustBadge != expected.TrustBadge {
		return fmt.Errorf("runtime assurance %q/%q does not match final claim %q",
			r.ResultClass, r.TrustBadge, r.Claim.Kind)
	}
	if r.Claim.Kind == ClaimViolating {
		if r.Trace == nil {
			return errors.New("violating runtime result requires a canonical semantic trace")
		}
		if err := r.Trace.Validate(); err != nil {
			return fmt.Errorf("validate violating runtime semantic trace: %w", err)
		}
		if r.Trace.Kind != protocol.SemanticTraceLive ||
			r.Trace.Producer != protocol.SemanticTraceProducerLive ||
			r.Trace.ExperimentDigest != r.ExperimentDigest ||
			string(r.Trace.Property) != r.Claim.Property {
			return errors.New("violating runtime semantic trace does not match its result")
		}
	} else if r.Trace != nil {
		return errors.New("non-violating runtime result cannot carry a semantic trace")
	}
	return nil
}

func (r Result) ValidateEvidenceDigest() error {
	encoded, err := r.canonicalEvidence()
	if err != nil {
		return fmt.Errorf("encode runtime evidence: %w", err)
	}
	digest := sha256.Sum256(encoded)
	expected := "sha256:" + hex.EncodeToString(digest[:])
	if r.EvidenceDigest != expected {
		return fmt.Errorf("runtime evidence digest %q does not match %q", r.EvidenceDigest, expected)
	}
	return nil
}

func (r *Result) BindEvidenceDigest() error {
	encoded, err := r.canonicalEvidence()
	if err != nil {
		return fmt.Errorf("encode runtime evidence: %w", err)
	}
	digest := sha256.Sum256(encoded)
	r.EvidenceDigest = "sha256:" + hex.EncodeToString(digest[:])
	return nil
}

func (r *Result) NormalizeEvidence(maxBytes int64) error {
	if maxBytes <= 0 {
		return errors.New("positive evidence byte limit is required")
	}
	claim := r.Claim.Kind
	finalizeEvidenceGraph(r, maxBytes)
	if r.Claim.Kind != claim {
		return errors.New(r.Claim.Reason)
	}
	if err := r.Evidence.Validate(); err != nil {
		return err
	}
	return r.ValidateEvidenceDigest()
}

func (r Result) canonicalEvidence() ([]byte, error) {
	if _, err := r.Evidence.CanonicalJSON(); err != nil {
		return nil, err
	}
	factIdentifiers := make(map[string]struct{}, len(r.Facts))
	for _, fact := range r.Facts {
		if err := fact.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := factIdentifiers[fact.Identifier]; duplicate {
			return nil, fmt.Errorf("duplicate runtime fact %q", fact.Identifier)
		}
		factIdentifiers[fact.Identifier] = struct{}{}
	}
	if len(r.Facts) != 0 {
		for _, interpreted := range r.Observations {
			if len(interpreted.SupportingFacts) == 0 {
				return nil, fmt.Errorf("observation %q has no supporting facts", interpreted.CheckpointID)
			}
			for _, identifier := range interpreted.SupportingFacts {
				if _, exists := factIdentifiers[identifier]; !exists {
					return nil, fmt.Errorf("observation %q references missing supporting fact %q",
						interpreted.CheckpointID, identifier)
				}
			}
		}
	}
	encoded, err := json.Marshal(struct {
		Facts        []observation.Fact      `json:"facts"`
		Actions      []ActionResult          `json:"actions"`
		Observations []Observation           `json:"observations"`
		Graph        evidencegraph.Graph     `json:"graph"`
		Trace        *protocol.SemanticTrace `json:"trace,omitempty"`
	}{
		Facts: r.Facts, Actions: r.Actions, Observations: r.Observations,
		Graph: r.Evidence, Trace: r.Trace,
	})
	if err != nil {
		return nil, fmt.Errorf("encode runtime evidence: %w", err)
	}
	return encoded, nil
}

func finalizeSemanticTrace(
	result *Result,
	experiment protocol.Experiment,
	view protocol.AttemptExecutionView,
	hasView bool,
	attempts []protocol.ObservedAttempt,
) {
	if result.Claim.Kind != ClaimViolating {
		result.Trace = nil
		return
	}
	if !hasView || len(attempts) == 0 {
		result.Claim.Kind = ClaimEvidenceFailure
		result.Claim.Reason = "violating evidence has no canonical attempt trace"
		return
	}
	trace, err := protocol.NewLiveSemanticTrace(experiment, view, attempts)
	if err != nil {
		result.Claim.Kind = ClaimEvidenceFailure
		result.Claim.Reason = "replay violating evidence: " + err.Error()
		return
	}
	result.Trace = &trace
}

func finalizeFootprint(result *Result, factory Factory, session Session) {
	provider, ok := session.(umpire3fault.FootprintProvider)
	if !ok {
		provider, ok = factory.(umpire3fault.FootprintProvider)
	}
	if !ok {
		return
	}
	report, err := provider.FootprintReport()
	if err == nil {
		err = report.RequireComplete()
	}
	if err != nil {
		result.Claim.Kind = ClaimEvidenceFailure
		result.Claim.Reason = "qualify learned footprint: " + err.Error()
		result.Omissions = append(result.Omissions, result.Claim.Reason)
		return
	}
	result.Footprint = &report
}

func appendCorroboratingFactObservations(
	ctx context.Context,
	session Session,
	checkpoint protocol.Checkpoint,
	bindings Bindings,
	timeout time.Duration,
	result *Result,
	primary Observation,
	requiredEvidence []protocol.EvidenceID,
	catalog observation.Catalog,
) (bool, string) {
	corroborating, ok := session.(CorroboratingFactSession)
	if !ok {
		return true, ""
	}
	corroborateCtx, cancelCorroborate := context.WithTimeout(ctx, timeout)
	factSets, err := corroborating.CorroborateFacts(corroborateCtx, checkpoint, bindings)
	cancelCorroborate()
	if err != nil {
		reason := "corroborate facts: " + err.Error()
		result.Omissions = append(result.Omissions, checkpoint.Identifier+": "+reason)
		result.Claim.Kind = ClaimEvidenceFailure
		result.Claim.Reason = reason
		return false, reason
	}
	if len(factSets) == 0 {
		reason := "corroborating facts are unavailable"
		result.Omissions = append(result.Omissions, checkpoint.Identifier+": "+reason)
		result.Claim.Kind = ClaimEvidenceFailure
		result.Claim.Reason = reason
		return false, reason
	}
	program, exists := catalog.Program(protocol.ObservationID(checkpoint.Observation))
	if !exists {
		reason := fmt.Sprintf("observation %q has no generated interpreter program", checkpoint.Observation)
		result.Claim.Kind = ClaimEvidenceFailure
		result.Claim.Reason = reason
		return false, reason
	}
	sourceIdentities := map[string]struct{}{primary.SourceIdentity: {}}
	for _, facts := range factSets {
		evaluation := program.Evaluate(facts)
		if evaluation.Value != observation.True && evaluation.Value != observation.False {
			reason := fmt.Sprintf("corroborating typed observation is %s: %v", evaluation.Value, evaluation.Support)
			result.Omissions = append(result.Omissions, checkpoint.Identifier+": "+reason)
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = reason
			return false, reason
		}
		interpreted, interpretErr := interpretedObservation(checkpoint, evaluation, facts)
		if interpretErr != nil {
			reason := "interpret corroborating facts: " + interpretErr.Error()
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = reason
			return false, reason
		}
		if !factsShareObservationIdentity(facts, interpreted) {
			reason := "corroborating fact set combines multiple source or entity identities"
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = reason
			return false, reason
		}
		if _, duplicate := sourceIdentities[interpreted.SourceIdentity]; duplicate {
			reason := "corroborating facts are not independently sourced"
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = reason
			return false, reason
		}
		sourceIdentities[interpreted.SourceIdentity] = struct{}{}
		if interpreted.EntityIdentity != primary.EntityIdentity || !slices.Equal(interpreted.Lineage, primary.Lineage) {
			reason := "corroborating facts identify a different entity lineage"
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = reason
			return false, reason
		}
		if interpreted.Satisfied != primary.Satisfied {
			reason := "corroborating facts contradict the primary source"
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = reason
			return false, reason
		}
		if qualified, reason := qualifyObservation(checkpoint, interpreted, requiredEvidence); !qualified {
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = "corroborating " + reason
			return false, result.Claim.Reason
		}
		if interpreted.ObservedAtUnixNano == 0 {
			interpreted.ObservedAtUnixNano = time.Now().UnixNano()
		}
		result.Facts = appendDistinctFacts(result.Facts, facts)
		result.Observations = append(result.Observations, interpreted)
	}
	return true, ""
}

func factsShareObservationIdentity(facts []observation.Fact, interpreted Observation) bool {
	if interpreted.SourceIdentity == "" || len(facts) == 0 {
		return false
	}
	for _, fact := range facts {
		if fact.Source.Identity != interpreted.SourceIdentity ||
			fact.Source.ClockDomain != interpreted.ClockDomain ||
			fact.Source.EntityIdentity != interpreted.EntityIdentity ||
			!slices.Equal(fact.Source.Lineage, interpreted.Lineage) {
			return false
		}
	}
	return true
}

func factsShareEntityIdentity(facts []observation.Fact, interpreted Observation) bool {
	if interpreted.EntityIdentity == "" || len(facts) == 0 {
		return false
	}
	for _, fact := range facts {
		if fact.Source.EntityIdentity != interpreted.EntityIdentity ||
			!slices.Equal(fact.Source.Lineage, interpreted.Lineage) {
			return false
		}
	}
	return true
}

func contradictionCheckpoint(
	observations []Observation,
	contradictions []protocol.ObservationID,
) string {
	for _, contradiction := range contradictions {
		for _, observation := range observations {
			if observation.Kind == string(contradiction) {
				return observation.CheckpointID
			}
		}
	}
	return ""
}

func appendDistinctFacts(existing []observation.Fact, additions []observation.Fact) []observation.Fact {
	for _, addition := range additions {
		if slices.ContainsFunc(existing, func(current observation.Fact) bool {
			return reflect.DeepEqual(current, addition)
		}) {
			continue
		}
		existing = append(existing, addition)
	}
	return existing
}

func interpretedObservation(
	checkpoint protocol.Checkpoint,
	evaluation observation.Evaluation,
	facts []observation.Fact,
) (Observation, error) {
	if len(evaluation.Support) == 0 {
		return Observation{}, errors.New("typed observation returned no supporting fact")
	}
	byIdentifier := make(map[string]observation.Fact, len(facts))
	for _, fact := range facts {
		byIdentifier[fact.Identifier] = fact
	}
	supporting := make([]observation.Fact, len(evaluation.Support))
	for index, identifier := range evaluation.Support {
		fact, exists := byIdentifier[identifier]
		if !exists {
			return Observation{}, fmt.Errorf("typed observation supporting fact %q is missing", identifier)
		}
		supporting[index] = fact
	}
	identity := supporting[0].Source
	latest := supporting[0]
	var causalReferences []string
	for _, fact := range supporting {
		if fact.Source.Identity != identity.Identity || fact.Source.ClockDomain != identity.ClockDomain ||
			fact.Source.EntityIdentity != identity.EntityIdentity ||
			!slices.Equal(fact.Source.Lineage, identity.Lineage) ||
			fact.Source.PayloadDigest != identity.PayloadDigest {
			return Observation{}, errors.New("typed observation supporting facts have inconsistent identity")
		}
		if fact.Source.Sequence > latest.Source.Sequence {
			latest = fact
		}
		for _, reference := range fact.Source.CausalReferences {
			if !slices.Contains(causalReferences, reference) {
				causalReferences = append(causalReferences, reference)
			}
		}
	}
	causalReference := ""
	if len(latest.Source.CausalReferences) != 0 {
		causalReference = latest.Source.CausalReferences[0]
	} else if len(causalReferences) != 0 {
		causalReference = causalReferences[0]
	}
	return Observation{
		CheckpointID:     checkpoint.Identifier,
		Kind:             checkpoint.Observation,
		Satisfied:        evaluation.Value == observation.True,
		Source:           identity.Identity,
		SourceIdentity:   identity.Identity,
		ClockDomain:      identity.ClockDomain,
		SourceSequence:   latest.Source.Sequence,
		Reference:        latest.Source.Reference,
		CausalReference:  causalReference,
		CausalReferences: causalReferences,
		EntityIdentity:   identity.EntityIdentity,
		Lineage:          append([]string(nil), identity.Lineage...),
		PayloadDigest:    identity.PayloadDigest,
		SupportingFacts:  append([]string(nil), evaluation.Support...),
	}, nil
}

func finalizeOutcome(result *Result) {
	var terminal *protocol.TerminalEvidence
	for _, action := range result.Actions {
		evidence := action.Evidence
		if evidence.TerminalState == "" && evidence.TerminalDisposition == "" {
			continue
		}
		candidate := protocol.TerminalEvidence{
			State: evidence.TerminalState, Disposition: evidence.TerminalDisposition,
			Reference: evidence.Reference, EntityIdentity: evidence.EntityIdentity,
		}
		terminal = &candidate
	}
	outcome, err := protocol.ClassifyOutcome(result.Claim.Kind, terminal)
	if err != nil {
		result.Claim.Kind = ClaimEvidenceFailure
		result.Claim.Reason = "qualify lifecycle outcome: " + err.Error()
		result.Omissions = append(result.Omissions, result.Claim.Reason)
		outcome, _ = protocol.ClassifyOutcome(result.Claim.Kind, nil)
	}
	result.Outcome = outcome
}

func finalizeCleanup(result *Result, session Session, timeout time.Duration) {
	cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), timeout)
	defer cancelCleanup()
	result.Cleanup = session.Cleanup(cleanupCtx)
	if result.Cleanup.Complete {
		return
	}
	if len(result.Cleanup.RecoverableResources) == 0 {
		result.Cleanup.RecoverableResources = session.RecoveryMetadata()
	}
	result.Omissions = append(result.Omissions, "cleanup incomplete")
	if result.Claim.Kind == ClaimConforming {
		result.Claim.Kind = ClaimInconclusive
		result.Claim.Reason = "cleanup incomplete"
	}
}

func (limits Limits) withDefaults(experiment protocol.Experiment) Limits {
	if limits.PrepareTimeout <= 0 {
		limits.PrepareTimeout = 10 * time.Second
	}
	if limits.ActionTimeout <= 0 {
		limits.ActionTimeout = 5 * time.Second
	}
	if limits.ObserveTimeout <= 0 {
		limits.ObserveTimeout = 5 * time.Second
	}
	if limits.FaultTimeout <= 0 {
		limits.FaultTimeout = limits.ActionTimeout
	}
	if limits.CleanupTimeout <= 0 {
		limits.CleanupTimeout = 5 * time.Second
	}
	if limits.MaxActions <= 0 {
		limits.MaxActions = experiment.Scope.Bounds.MaxDepth
	}
	if limits.MaxObservations <= 0 {
		limits.MaxObservations = experiment.Scope.Bounds.MaxResults
	}
	if limits.MaxResources <= 0 {
		limits.MaxResources = len(experiment.Resources)
	}
	if limits.MaxEvidenceBytes <= 0 {
		limits.MaxEvidenceBytes = experiment.Retention.MaxArtifactBytes
	}
	return limits
}

func waitForActionRate(ctx context.Context, previous *time.Time, perSecond int) error {
	if perSecond <= 0 {
		return nil
	}
	interval := time.Second / time.Duration(perSecond)
	if previous.IsZero() || interval <= 0 {
		*previous = time.Now()
		return nil
	}
	wait := time.Until(previous.Add(interval))
	if wait <= 0 {
		*previous = time.Now()
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		*previous = time.Now()
		return nil
	}
}

var errFaultRealizerUnavailable = errors.New("environment does not provide a fault realizer")

type installedFault struct {
	definition protocol.Fault
	term       umpire3fault.Term
	handle     string
	result     int
	active     bool
	cleaned    bool
}

type faultSet struct {
	realizer         umpire3fault.Realizer
	evidenceProvider umpire3fault.EvidenceProvider
	values           []installedFault
}

func prepareFaults(
	ctx context.Context,
	request Request,
	session Session,
	capabilities []protocol.CapabilityID,
	limits Limits,
	result *Result,
) (*faultSet, error) {
	if len(request.Experiment.Faults) == 0 {
		return nil, nil
	}
	provider, ok := request.Environment.(umpire3fault.Provider)
	if !ok {
		provider, ok = session.(umpire3fault.Provider)
	}
	if !ok || provider.FaultRealizer() == nil {
		return nil, errFaultRealizerUnavailable
	}
	realizer := provider.FaultRealizer()
	evidenceProvider, ok := realizer.(umpire3fault.EvidenceProvider)
	if !ok {
		return nil, errors.New("fault realizer does not provide realization evidence")
	}
	set := &faultSet{
		realizer: realizer, evidenceProvider: evidenceProvider,
		values: make([]installedFault, 0, len(request.Experiment.Faults)),
	}
	actionIndexes := make(map[string]int64, len(request.Experiment.Actions))
	for index, action := range request.Experiment.Actions {
		actionIndexes[action.Identifier] = int64(index + 1)
	}
	isolationIdentity := result.Environment.IsolationIdentity
	if isolationIdentity == "" {
		isolationIdentity = request.Experiment.ExperimentID
	}
	for _, definition := range request.Experiment.Faults {
		term := umpire3fault.Term{
			Kind: protocol.FaultKind(definition.Kind),
			Scope: umpire3fault.Scope{
				Namespaces: []string{isolationIdentity}, Endpoints: slices.Clone(definition.Scope.Endpoints),
				TaskQueues: slices.Clone(definition.Scope.TaskQueues), Services: slices.Clone(definition.Scope.Services),
				Routes: slices.Clone(definition.Scope.Routes), Participants: slices.Clone(definition.Scope.Participants),
				Attempts: slices.Clone(definition.Scope.Attempts),
			},
			Occurrence: umpire3fault.Occurrence{First: definition.Occurrence.First, Count: definition.Occurrence.Count},
			Interval: umpire3fault.Interval{
				Start: actionIndexes[definition.Interval.StartAction],
				Stop:  actionIndexes[definition.Interval.StopAction] + 1,
			},
		}
		result.Faults = append(result.Faults, FaultResult{Identifier: definition.Identifier, Kind: definition.Kind})
		resultIndex := len(result.Faults) - 1
		if err := umpire3fault.Preflight(term, capabilities, request.AllowRestrictedFaults); err != nil {
			result.Faults[resultIndex].Error = err.Error()
			return set, fmt.Errorf("preflight fault %q: %w", definition.Identifier, err)
		}
		faultCtx, cancelFault := context.WithTimeout(ctx, limits.FaultTimeout)
		handle, err := realizer.Install(faultCtx, term)
		cancelFault()
		if err != nil {
			result.Faults[resultIndex].Error = err.Error()
			return set, fmt.Errorf("install fault %q: %w", definition.Identifier, err)
		}
		if handle == "" {
			result.Faults[resultIndex].Error = "fault realizer returned an empty handle"
			return set, fmt.Errorf("install fault %q: fault realizer returned an empty handle", definition.Identifier)
		}
		digest := sha256.Sum256([]byte(handle))
		result.Faults[resultIndex].Reference = "fault-installation/" + definition.Identifier + "/sha256:" + hex.EncodeToString(digest[:])
		result.Faults[resultIndex].Installed = true
		set.values = append(set.values, installedFault{
			definition: definition, term: term, handle: handle, result: resultIndex,
		})
	}
	return set, nil
}

func (s *faultSet) beforeAction(ctx context.Context, action string, result *Result) error {
	for index := range s.values {
		value := &s.values[index]
		if value.definition.Interval.StartAction != action || value.active {
			continue
		}
		if err := s.realizer.Activate(ctx, value.handle); err != nil {
			appendFaultError(&result.Faults[value.result], fmt.Errorf("activate fault: %w", err))
			return fmt.Errorf("activate fault %q: %w", value.definition.Identifier, err)
		}
		value.active = true
		result.Faults[value.result].Activated = true
	}
	return nil
}

func (s *faultSet) afterAction(ctx context.Context, action string, result *Result) error {
	for index := len(s.values) - 1; index >= 0; index-- {
		value := &s.values[index]
		if value.definition.Interval.StopAction != action || !value.active {
			continue
		}
		evidence, err := s.evidenceProvider.RealizationEvidence(ctx, value.handle)
		if err != nil {
			appendFaultError(&result.Faults[value.result], fmt.Errorf("observe fault realization: %w", err))
			return fmt.Errorf("observe fault %q realization: %w", value.definition.Identifier, err)
		}
		if evidence.SourceIdentity == "" || evidence.Reference == "" || evidence.EntityIdentity == "" {
			err := errors.New("fault realization evidence is incomplete")
			appendFaultError(&result.Faults[value.result], err)
			return fmt.Errorf("observe fault %q realization: %w", value.definition.Identifier, err)
		}
		faultResult := &result.Faults[value.result]
		faultResult.SourceIdentity = evidence.SourceIdentity
		faultResult.Reference = evidence.Reference
		faultResult.EntityIdentity = evidence.EntityIdentity
		faultResult.Realized = true
		if err := s.realizer.Release(ctx, value.handle); err != nil {
			appendFaultError(&result.Faults[value.result], fmt.Errorf("release fault: %w", err))
			return fmt.Errorf("release fault %q: %w", value.definition.Identifier, err)
		}
		value.active = false
		result.Faults[value.result].Released = true
	}
	return nil
}

func (s *faultSet) cleanup(result *Result, timeout time.Duration) {
	cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), timeout)
	defer cancelCleanup()
	cleanupFailed := false
	for index := len(s.values) - 1; index >= 0; index-- {
		value := &s.values[index]
		faultResult := &result.Faults[value.result]
		if value.active {
			if err := s.realizer.Release(cleanupCtx, value.handle); err != nil {
				appendFaultError(faultResult, fmt.Errorf("release fault during cleanup: %w", err))
				cleanupFailed = true
			} else {
				value.active = false
				faultResult.Released = true
			}
		}
		if err := s.realizer.Cleanup(cleanupCtx, value.handle); err != nil {
			appendFaultError(faultResult, fmt.Errorf("cleanup fault: %w", err))
			cleanupFailed = true
			continue
		}
		value.cleaned = true
		faultResult.CleanupComplete = true
	}
	if cleanupFailed {
		result.Omissions = append(result.Omissions, "fault cleanup incomplete")
		if result.Claim.Kind == ClaimConforming || result.Claim.Kind == ClaimViolating {
			result.Claim.Kind = ClaimInconclusive
			result.Claim.Reason = "fault cleanup incomplete"
		}
	}
}

func appendFaultError(result *FaultResult, err error) {
	if result.Error == "" {
		result.Error = err.Error()
		return
	}
	result.Error += "; " + err.Error()
}

func actionEvidenceSize(evidence ActionEvidence) int64 {
	size := len(evidence.Source) + len(evidence.Reference)
	for key, value := range evidence.GroundedBindings {
		size += len(key) + len(value)
	}
	return int64(size)
}

func missingRuntimeBindings(arguments []protocol.NamedValue, bindings Bindings) []string {
	missing := make(map[string]struct{})
	for _, argument := range arguments {
		collectMissingRuntimeBindings(argument.Value, bindings, missing)
	}
	return uniqueSortedMap(missing)
}

func collectMissingRuntimeBindings(value protocol.Value, bindings Bindings, missing map[string]struct{}) {
	if value.Type == protocol.ValueSymbol && value.Text != nil {
		if _, grounded := bindings[*value.Text]; !grounded {
			missing[*value.Text] = struct{}{}
		}
		return
	}
	for _, element := range value.Elements {
		collectMissingRuntimeBindings(element, bindings, missing)
	}
	for _, field := range value.Fields {
		collectMissingRuntimeBindings(field.Value, bindings, missing)
	}
}

func uniqueSortedCapabilities(values []protocol.CapabilityID) []protocol.CapabilityID {
	seen := make(map[protocol.CapabilityID]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	result := make([]protocol.CapabilityID, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	slices.Sort(result)
	return result
}

func missingCapabilities(experiment protocol.Experiment, available []protocol.CapabilityID) []string {
	have := make(map[protocol.CapabilityID]struct{}, len(available))
	for _, capability := range available {
		have[capability] = struct{}{}
	}
	missing := make(map[string]struct{})
	for _, action := range experiment.Actions {
		for _, capability := range action.RequiredCapabilities {
			if _, exists := have[protocol.CapabilityID(capability)]; !exists {
				missing[capability] = struct{}{}
			}
		}
	}
	for _, fault := range experiment.Faults {
		for _, capability := range fault.RequiredCapabilities {
			if _, exists := have[protocol.CapabilityID(capability)]; !exists {
				missing[capability] = struct{}{}
			}
		}
	}
	return uniqueSortedMap(missing)
}

func uniqueSortedMap(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	slices.Sort(result)
	return result
}

func qualifyObservation(
	checkpoint protocol.Checkpoint,
	observation Observation,
	requiredEvidence []protocol.EvidenceID,
) (bool, string) {
	if observation.CheckpointID != checkpoint.Identifier || observation.Kind != checkpoint.Observation {
		return false, "observation identity does not match checkpoint"
	}
	if observation.Source == "" || observation.SourceIdentity == "" {
		return false, "observation source identity is missing"
	}
	if observation.ClockDomain == "" || observation.SourceSequence <= 0 || observation.Reference == "" {
		return false, "observation clock, sequence, or reference is missing"
	}
	switch checkpoint.Ordering {
	case "causal":
		if observation.CausalReference == "" && len(observation.CausalReferences) == 0 {
			return false, "causal reference is missing"
		}
	case "source-sequence":
		if observation.SourceSequence <= 0 {
			return false, "source sequence is missing"
		}
	default:
		if checkpoint.Ordering != "none" {
			return false, "unknown ordering requirement"
		}
	}
	if slices.Contains(requiredEvidence, protocol.EvidenceIDIdentityLineage) &&
		(observation.EntityIdentity == "" || len(observation.Lineage) == 0) {
		return false, "entity identity or lineage is missing"
	}
	return true, ""
}

func finalizeEvidenceGraph(result *Result, maxBytes int64) {
	builder := evidencegraph.NewBuilder(evidencegraph.Limits{
		MaxFacts: max(1, len(result.Observations)), MaxBytes: max(maxBytes, int64(1)),
	})
	var graphErr error
	for _, action := range result.Actions {
		sourceIdentity := action.Evidence.SourceIdentity
		if sourceIdentity == "" {
			sourceIdentity = action.Evidence.Source
		}
		if action.Evidence.Source == "" && action.Evidence.Reference == "" {
			continue
		}
		if err := builder.AddAction(evidencegraph.Action{
			Identifier: action.Identifier, Kind: action.Kind, Outcome: string(action.Evidence.Outcome),
			SourceIdentity: sourceIdentity,
			Reference:      action.Evidence.Reference, EntityIdentity: action.Evidence.EntityIdentity,
			Lineage: action.Evidence.Lineage, PayloadDigest: action.Evidence.PayloadDigest,
		}); err != nil && graphErr == nil {
			graphErr = err
		}
	}
	for _, faultResult := range result.Faults {
		if !faultResult.Realized || faultResult.Reference == "" {
			continue
		}
		sourceIdentity := faultResult.SourceIdentity
		if sourceIdentity == "" {
			sourceIdentity = result.Environment.FaultAuthority
		}
		entityIdentity := faultResult.EntityIdentity
		if entityIdentity == "" {
			entityIdentity = result.Environment.IsolationIdentity
		}
		if entityIdentity == "" {
			entityIdentity = result.ExperimentDigest
		}
		if err := builder.AddAction(evidencegraph.Action{
			Identifier: faultResult.Identifier, Kind: "fault:" + faultResult.Kind,
			Outcome:        "realized",
			SourceIdentity: sourceIdentity, Reference: faultResult.Reference,
			EntityIdentity: entityIdentity, Lineage: []string{result.ExperimentDigest, entityIdentity},
		}); err != nil && graphErr == nil {
			graphErr = err
		}
	}
	checkpointCounts := make(map[string]int, len(result.Observations))
	for _, observation := range result.Observations {
		checkpointCounts[observation.CheckpointID]++
	}
	for _, observation := range result.Observations {
		causalReferences := append([]string(nil), observation.CausalReferences...)
		if observation.CausalReference != "" && !slices.Contains(causalReferences, observation.CausalReference) {
			causalReferences = append(causalReferences, observation.CausalReference)
		}
		identifier := observation.CheckpointID
		if checkpointCounts[observation.CheckpointID] > 1 {
			identifier += "@" + observation.SourceIdentity
		}
		if err := builder.AddFact(evidencegraph.Fact{
			Identifier: identifier, Kind: observation.Kind, Value: observation.Satisfied,
			SourceIdentity: observation.SourceIdentity, ClockDomain: observation.ClockDomain,
			SourceSequence:            observation.SourceSequence,
			AuthoritativeTimeUnixNano: observation.AuthoritativeTimeUnixNano,
			ObservedAtUnixNano:        observation.ObservedAtUnixNano, Reference: observation.Reference,
			CausalReferences: causalReferences, EntityIdentity: observation.EntityIdentity,
			Lineage: observation.Lineage, PayloadDigest: observation.PayloadDigest,
		}); err != nil && graphErr == nil {
			graphErr = err
		}
	}
	for _, omission := range result.Omissions {
		builder.AddOmission(omission)
	}
	if err := builder.AddClaim(evidencegraph.Claim{
		Property: result.Claim.Property, Verdict: string(result.Claim.Kind), Reason: result.Claim.Reason,
	}); err != nil && graphErr == nil {
		graphErr = err
	}
	graph, err := builder.Build()
	result.Evidence = graph
	_ = result.BindEvidenceDigest()
	if graphErr == nil {
		graphErr = err
	}
	if graphErr != nil {
		var contradiction *evidencegraph.ContradictionError
		if errors.As(graphErr, &contradiction) {
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = "normalize evidence graph: " + graphErr.Error()
		} else if result.Claim.Kind == ClaimConforming || result.Claim.Kind == ClaimViolating {
			result.Claim.Kind = ClaimInconclusive
			result.Claim.Reason = "normalize evidence graph: " + graphErr.Error()
		}
	}
}
