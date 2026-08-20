//nolint:revive // The package name is the public Umpire3 runtime.Run seam.
package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"time"

	"go.temporal.io/server/tests/umpire3/environment"
	evidencegraph "go.temporal.io/server/tests/umpire3/evidence"
	umpire3fault "go.temporal.io/server/tests/umpire3/fault"
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
)

type Limits struct {
	PrepareTimeout   time.Duration
	ActionTimeout    time.Duration
	ObserveTimeout   time.Duration
	FaultTimeout     time.Duration
	CleanupTimeout   time.Duration
	MaxActions       int
	MaxObservations  int
	MaxResources     int
	MaxEvidenceBytes int64
}

type Request struct {
	Experiment  protocol.Experiment
	Environment environment.Factory
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
	Identifier string                     `json:"identifier"`
	Kind       string                     `json:"kind"`
	Evidence   environment.ActionEvidence `json:"evidence"`
	Error      string                     `json:"error,omitempty"`
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
	FormatVersion    string                    `json:"formatVersion"`
	ExperimentDigest string                    `json:"experimentDigest"`
	Environment      EnvironmentProfile        `json:"environment"`
	Actions          []ActionResult            `json:"actions"`
	Bindings         environment.Bindings      `json:"bindings"`
	Observations     []environment.Observation `json:"observations"`
	Faults           []FaultResult             `json:"faults,omitempty"`
	Omissions        []string                  `json:"omissions"`
	Checkpoints      []CheckpointResult        `json:"checkpoints"`
	Evidence         evidencegraph.Graph       `json:"evidence"`
	Footprint        *umpire3fault.Report      `json:"footprint,omitempty"`
	Claim            Claim                     `json:"claim"`
	Outcome          protocol.Outcome          `json:"outcome"`
	Cleanup          environment.CleanupResult `json:"cleanup"`
}

type EnvironmentProfile struct {
	Name                  string   `json:"name,omitempty"`
	BuildID               string   `json:"buildID,omitempty"`
	ConfigurationIdentity string   `json:"configurationIdentity,omitempty"`
	EvidenceProfile       string   `json:"evidenceProfile,omitempty"`
	DrivingAuthority      string   `json:"drivingAuthority,omitempty"`
	ObservationAuthority  string   `json:"observationAuthority,omitempty"`
	FaultAuthority        string   `json:"faultAuthority,omitempty"`
	IsolationIdentity     string   `json:"isolationIdentity,omitempty"`
	RetentionClass        string   `json:"retentionClass,omitempty"`
	HardExecutionBudget   bool     `json:"hardExecutionBudget"`
	Capabilities          []string `json:"capabilities"`
}

func Run(ctx context.Context, request Request) (result Result, retErr error) {
	if err := request.Experiment.Validate(); err != nil {
		return Result{}, fmt.Errorf("validate experiment: %w", err)
	}
	monitorCatalog, err := protocol.DefaultMonitorCatalog()
	if err != nil {
		return Result{}, fmt.Errorf("load monitor programs: %w", err)
	}
	monitor, ok := monitorCatalog.Program(protocol.PropertyID(request.Experiment.Property.Identifier))
	if !ok {
		return Result{}, fmt.Errorf("property %q has no generated monitor program", request.Experiment.Property.Identifier)
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

	capabilities := uniqueSorted(request.Environment.Capabilities())
	result = Result{
		FormatVersion:    protocol.FormatVersion,
		ExperimentDigest: digest,
		Environment:      EnvironmentProfile{Capabilities: capabilities},
		Bindings:         make(environment.Bindings),
		Claim: Claim{
			Kind:     ClaimInconclusive,
			Property: request.Experiment.Property.Identifier,
		},
	}
	defer finalizeOutcome(&result)
	defer func() { finalizeEvidenceGraph(&result, limits.MaxEvidenceBytes) }()
	if missing := missingCapabilities(request.Experiment, capabilities); len(missing) != 0 {
		result.Claim.Kind = ClaimUnsupported
		result.Claim.Reason = "missing capabilities: " + fmt.Sprint(missing)
		return result, nil
	}

	prepareCtx, cancelPrepare := context.WithTimeout(ctx, limits.PrepareTimeout)
	session, err := request.Environment.Prepare(prepareCtx, request.Experiment)
	cancelPrepare()
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
	defer finalizeFootprint(&result, request.Environment, session)
	if provider, ok := session.(environment.ProfileProvider); ok {
		profile := provider.Profile()
		if err := profile.Validate(); err != nil {
			result.Claim.Kind = ClaimUnsupported
			result.Claim.Reason = "invalid environment profile: " + err.Error()
			return result, nil
		}
		result.Environment.Name = profile.Name
		result.Environment.BuildID = profile.BuildID
		result.Environment.ConfigurationIdentity = profile.ConfigurationIdentity
		result.Environment.EvidenceProfile = profile.EvidenceProfile
		result.Environment.DrivingAuthority = profile.DrivingAuthority
		result.Environment.ObservationAuthority = profile.ObservationAuthority
		result.Environment.FaultAuthority = profile.FaultAuthority
		result.Environment.IsolationIdentity = profile.IsolationIdentity
		result.Environment.RetentionClass = profile.RetentionClass
		result.Environment.HardExecutionBudget = profile.HardExecutionBudget
	}

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
	observe := func(identifier string) bool {
		if identifier == "" {
			return true
		}
		if _, exists := observed[identifier]; exists {
			return true
		}
		checkpoint := checkpointByID[identifier]
		observeCtx, cancelObserve := context.WithTimeout(ctx, limits.ObserveTimeout)
		observation, observeErr := session.Observe(observeCtx, checkpoint, result.Bindings)
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
			result.Checkpoints = append(result.Checkpoints, CheckpointResult{
				Identifier: identifier,
				Reason:     observeErr.Error(),
			})
			return false
		}
		if observation.ObservedAtUnixNano == 0 {
			observation.ObservedAtUnixNano = time.Now().UnixNano()
		}
		result.Observations = append(result.Observations, observation)
		qualified, reason := qualifyObservation(checkpoint, observation, monitor.Evidence)
		if qualified {
			qualified, reason = appendCorroboratingObservations(
				ctx, session, checkpoint, result.Bindings, limits.ObserveTimeout, &result, observation, monitor.Evidence,
			)
		}
		result.Checkpoints = append(result.Checkpoints, CheckpointResult{
			Identifier: identifier,
			Satisfied:  observation.Satisfied,
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
	declaredSymbols := make(map[string]struct{})
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

func finalizeFootprint(result *Result, factory environment.Factory, session environment.Session) {
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

func appendCorroboratingObservations(
	ctx context.Context,
	session environment.Session,
	checkpoint protocol.Checkpoint,
	bindings environment.Bindings,
	timeout time.Duration,
	result *Result,
	primary environment.Observation,
	requiredEvidence []protocol.EvidenceID,
) (bool, string) {
	corroborating, ok := session.(environment.CorroboratingSession)
	if !ok {
		return true, ""
	}
	corroborateCtx, cancelCorroborate := context.WithTimeout(ctx, timeout)
	observations, err := corroborating.Corroborate(corroborateCtx, checkpoint, bindings)
	cancelCorroborate()
	if err != nil {
		reason := "corroborate observation: " + err.Error()
		result.Omissions = append(result.Omissions, checkpoint.Identifier+": "+reason)
		result.Claim.Kind = ClaimEvidenceFailure
		result.Claim.Reason = reason
		return false, reason
	}
	if len(observations) == 0 {
		reason := "corroborating observation is unavailable"
		result.Omissions = append(result.Omissions, checkpoint.Identifier+": "+reason)
		result.Claim.Kind = ClaimEvidenceFailure
		result.Claim.Reason = reason
		return false, reason
	}
	sourceIdentities := map[string]struct{}{primary.SourceIdentity: {}}
	for _, observation := range observations {
		if observation.ObservedAtUnixNano == 0 {
			observation.ObservedAtUnixNano = time.Now().UnixNano()
		}
		result.Observations = append(result.Observations, observation)
		if qualified, reason := qualifyObservation(checkpoint, observation, requiredEvidence); !qualified {
			result.Omissions = append(result.Omissions, checkpoint.Identifier+": corroborating "+reason)
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = "corroborating " + reason
			return false, result.Claim.Reason
		}
		if _, duplicate := sourceIdentities[observation.SourceIdentity]; duplicate {
			reason := "corroborating observation is not independently sourced"
			result.Omissions = append(result.Omissions, checkpoint.Identifier+": "+reason)
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = reason
			return false, reason
		}
		sourceIdentities[observation.SourceIdentity] = struct{}{}
		if observation.EntityIdentity != primary.EntityIdentity || !slices.Equal(observation.Lineage, primary.Lineage) {
			reason := "corroborating observation identifies a different entity lineage"
			result.Omissions = append(result.Omissions, checkpoint.Identifier+": "+reason)
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = reason
			return false, reason
		}
		if observation.Satisfied != primary.Satisfied {
			reason := "corroborating observation contradicts the primary source"
			result.Claim.Kind = ClaimEvidenceFailure
			result.Claim.Reason = reason
			return false, reason
		}
	}
	return true, ""
}

func contradictionCheckpoint(
	observations []environment.Observation,
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

func finalizeCleanup(result *Result, session environment.Session, timeout time.Duration) {
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
	session environment.Session,
	capabilities []string,
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
	capabilityIDs := make([]protocol.CapabilityID, len(capabilities))
	for index, capability := range capabilities {
		capabilityIDs[index] = protocol.CapabilityID(capability)
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
		if err := umpire3fault.Preflight(term, capabilityIDs, request.AllowRestrictedFaults); err != nil {
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

func actionEvidenceSize(evidence environment.ActionEvidence) int64 {
	size := len(evidence.Source) + len(evidence.Reference)
	for key, value := range evidence.GroundedBindings {
		size += len(key) + len(value)
	}
	return int64(size)
}

func missingRuntimeBindings(arguments []protocol.NamedValue, bindings environment.Bindings) []string {
	missing := make(map[string]struct{})
	for _, argument := range arguments {
		collectMissingRuntimeBindings(argument.Value, bindings, missing)
	}
	return uniqueSortedMap(missing)
}

func collectMissingRuntimeBindings(value protocol.Value, bindings environment.Bindings, missing map[string]struct{}) {
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

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	slices.Sort(result)
	return result
}

func missingCapabilities(experiment protocol.Experiment, available []string) []string {
	have := make(map[string]struct{}, len(available))
	for _, capability := range available {
		have[capability] = struct{}{}
	}
	missing := make(map[string]struct{})
	for _, action := range experiment.Actions {
		for _, capability := range action.RequiredCapabilities {
			if _, exists := have[capability]; !exists {
				missing[capability] = struct{}{}
			}
		}
	}
	for _, fault := range experiment.Faults {
		for _, capability := range fault.RequiredCapabilities {
			if _, exists := have[capability]; !exists {
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
	observation environment.Observation,
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
			Identifier: action.Identifier, Kind: action.Kind, SourceIdentity: sourceIdentity,
			Reference: action.Evidence.Reference, EntityIdentity: action.Evidence.EntityIdentity,
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
