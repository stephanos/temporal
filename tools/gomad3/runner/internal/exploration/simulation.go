package exploration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"

	"go.temporal.io/server/tools/gomad3/record"
)

const (
	simulationCandidateDomain    = "gomad3-simulation-exploration-candidate/v1"
	decisionDomain               = "gomad3-simulation-exploration-decision/v1"
	forcedDomain                 = "gomad3-simulation-exploration-forced-decision/v1"
	simulationStateDomain        = "gomad3-simulation-exploration-state/v1"
	simulationRoundSegmentDomain = "gomad3-simulation-exploration-round-segment/v1"
	SimulationRoundSegmentSchema = "gomad3.simulation-exploration-round/v2"
	simulationControllerIdentity = "deterministic-rounds/breadth-first-combined-rank-overrides/v2"
)

func SimulationImplementationSHA256() record.SHA256 {
	return record.DomainHash("gomad3-simulation-exploration-controller/v1", []byte(simulationControllerIdentity))
}

type Dimension string

const (
	DimensionRuntime  Dimension = "runtime"
	DimensionScenario Dimension = "scenario"
	DimensionNetwork  Dimension = "network"
	DimensionStorage  Dimension = "storage"
	DimensionFault    Dimension = "fault"
	DimensionCrash    Dimension = "crash"
)

type SimulationStopReason string

const (
	SimulationStopExhausted           SimulationStopReason = "exploration_exhausted"
	SimulationStopDepthComplete       SimulationStopReason = "simulation_depth_complete"
	SimulationStopDimensionComplete   SimulationStopReason = "dimension_depth_complete"
	SimulationStopMaxExecutions       SimulationStopReason = "max_executions"
	SimulationStopExplorationCapacity SimulationStopReason = "exploration_capacity"
	SimulationStopFailureBudget       SimulationStopReason = "failure_budget"
)

type DimensionLimits struct {
	Runtime  uint64 `json:"runtime"`
	Scenario uint64 `json:"scenario"`
	Network  uint64 `json:"network"`
	Storage  uint64 `json:"storage"`
	Fault    uint64 `json:"fault"`
	Crash    uint64 `json:"crash"`
}

type SimulationConfig struct {
	ExecutionSHA256     record.SHA256   `json:"execution_sha256"`
	ControllerSHA256    record.SHA256   `json:"controller_sha256"`
	BaseSeed            uint64          `json:"base_seed"`
	Parallel            int             `json:"parallel"`
	MaxExecutions       uint64          `json:"max_executions"`
	MaxForcedDecisions  uint64          `json:"max_forced_decisions"`
	MaxExplorationBytes uint64          `json:"max_exploration_bytes"`
	MaxResultBytes      uint64          `json:"max_result_bytes"`
	FailureBudget       uint64          `json:"failure_budget"`
	Limits              DimensionLimits `json:"dimension_limits"`
}

type Decision struct {
	Dimension            Dimension       `json:"dimension"`
	Ordinal              uint64          `json:"ordinal"`
	SiteSHA256           record.SHA256   `json:"site_sha256"`
	Alternatives         []record.SHA256 `json:"alternatives"`
	AlternativeControls  [][]byte        `json:"alternative_controls,omitempty"`
	AlternativeSetSHA256 record.SHA256   `json:"alternative_set_sha256"`
	Selected             uint32          `json:"selected"`
	Identity             record.SHA256   `json:"identity"`
}

type ForcedDecision struct {
	Dimension            Dimension     `json:"dimension"`
	Ordinal              uint64        `json:"ordinal"`
	SiteSHA256           record.SHA256 `json:"site_sha256"`
	Alternatives         uint32        `json:"alternatives"`
	AlternativeSetSHA256 record.SHA256 `json:"alternative_set_sha256"`
	Selected             uint32        `json:"selected"`
	SelectedSHA256       record.SHA256 `json:"selected_sha256"`
	Control              []byte        `json:"control,omitempty"`
	Identity             record.SHA256 `json:"identity"`
}

type SimulationCandidate struct {
	SHA256       record.SHA256    `json:"sha256"`
	ParentSHA256 record.SHA256    `json:"parent_sha256,omitempty"`
	Overrides    []ForcedDecision `json:"overrides"`
}

type SimulationResult struct {
	CandidateSHA256 record.SHA256 `json:"candidate_sha256"`
	OutcomeSHA256   record.SHA256 `json:"outcome_sha256"`
	Failed          bool          `json:"failed"`
	FailureSHA256   record.SHA256 `json:"failure_sha256,omitempty"`
	Diverged        bool          `json:"diverged"`
	Decisions       []Decision    `json:"decisions"`
}

type SimulationState struct {
	Config                  SimulationConfig      `json:"config"`
	Queue                   []SimulationCandidate `json:"queue"`
	Seen                    []record.SHA256       `json:"seen"`
	Outcomes                []record.SHA256       `json:"outcomes"`
	FailureSignatures       []record.SHA256       `json:"failure_signatures"`
	LogicalExecutions       uint64                `json:"logical_executions"`
	CommittedRounds         uint64                `json:"committed_rounds"`
	PendingBytes            uint64                `json:"pending_bytes"`
	DeepestOverride         uint64                `json:"deepest_override"`
	OmittedByExecutionBound uint64                `json:"omitted_by_execution_bound"`
	OmittedByDepth          uint64                `json:"omitted_by_depth"`
	OmittedByDimension      uint64                `json:"omitted_by_dimension"`
	OmittedByCapacity       uint64                `json:"omitted_by_capacity"`
	StopReason              SimulationStopReason  `json:"stop_reason,omitempty"`
}

type SimulationRound struct {
	Index      uint64                `json:"index"`
	Candidates []SimulationCandidate `json:"candidates"`
}

type SimulationRoundSegment struct {
	Schema       string             `json:"schema"`
	Index        uint64             `json:"index"`
	BeforeSHA256 record.SHA256      `json:"before_sha256"`
	AfterSHA256  record.SHA256      `json:"after_sha256"`
	Results      []SimulationResult `json:"results"`
	SHA256       record.SHA256      `json:"sha256,omitempty"`
}

type SimulationSummary struct {
	Parallel                int                  `json:"parallel"`
	MaxExecutions           uint64               `json:"max_executions"`
	MaxForcedDecisions      uint64               `json:"max_forced_decisions"`
	MaxExplorationBytes     uint64               `json:"max_exploration_bytes"`
	MaxResultBytes          uint64               `json:"max_result_bytes"`
	FailureBudget           uint64               `json:"failure_budget"`
	Limits                  DimensionLimits      `json:"dimension_limits"`
	LogicalExecutions       uint64               `json:"logical_executions"`
	CommittedRounds         uint64               `json:"committed_rounds"`
	Pending                 uint64               `json:"pending"`
	PendingBytes            uint64               `json:"pending_bytes"`
	SeenCandidates          uint64               `json:"seen_candidates"`
	DeduplicatedOutcomes    uint64               `json:"deduplicated_outcomes"`
	DistinctFailures        uint64               `json:"distinct_failures"`
	DeepestOverride         uint64               `json:"deepest_override"`
	OmittedByExecutionBound uint64               `json:"omitted_by_execution_bound"`
	OmittedByDepth          uint64               `json:"omitted_by_depth"`
	OmittedByDimension      uint64               `json:"omitted_by_dimension"`
	OmittedByCapacity       uint64               `json:"omitted_by_capacity"`
	StopReason              SimulationStopReason `json:"stop_reason,omitempty"`
	BoundedComplete         bool                 `json:"bounded_complete"`
}

func CanonicalDecision(dimension Dimension, ordinal uint64, site record.SHA256, alternatives []record.SHA256, selected uint32) (Decision, error) {
	return canonicalDecision(dimension, ordinal, site, alternatives, nil, selected)
}

func CanonicalControlledDecision(dimension Dimension, ordinal uint64, site record.SHA256, alternatives []record.SHA256, controls [][]byte, selected uint32) (Decision, error) {
	return canonicalDecision(dimension, ordinal, site, alternatives, controls, selected)
}

func canonicalDecision(dimension Dimension, ordinal uint64, site record.SHA256, alternatives []record.SHA256, controls [][]byte, selected uint32) (Decision, error) {
	decision := Decision{
		Dimension: dimension, Ordinal: ordinal, SiteSHA256: site,
		Alternatives: append([]record.SHA256(nil), alternatives...), AlternativeControls: cloneControls(controls), Selected: selected,
	}
	var err error
	decision.AlternativeSetSHA256, err = alternativeSetIdentity(decision.Dimension, decision.Ordinal, decision.SiteSHA256, decision.Alternatives)
	if err != nil {
		return Decision{}, err
	}
	decision.Identity, err = decisionIdentity(decision)
	if err != nil {
		return Decision{}, err
	}
	if err := validateDecision(decision); err != nil {
		return Decision{}, err
	}
	return decision, nil
}

func NewSimulation(config SimulationConfig) (SimulationState, error) {
	if err := validateConfig(config); err != nil {
		return SimulationState{}, err
	}
	root, err := simulationNewCandidate(config, nil, "")
	if err != nil {
		return SimulationState{}, err
	}
	pendingBytes, err := simulationQueueBytes([]SimulationCandidate{root})
	if err != nil {
		return SimulationState{}, err
	}
	if pendingBytes > config.MaxExplorationBytes {
		return SimulationState{}, fmt.Errorf("simulation exploration root requires %d bytes, exceeding the %d-byte exploration bound", pendingBytes, config.MaxExplorationBytes)
	}
	return SimulationState{
		Config: config, Queue: []SimulationCandidate{root}, Seen: []record.SHA256{root.SHA256},
		Outcomes: []record.SHA256{}, FailureSignatures: []record.SHA256{}, PendingBytes: pendingBytes,
	}, nil
}

func (state SimulationState) NextRound() (SimulationRound, bool) {
	candidates, ok := NextRound(state.Queue, state.Config.Parallel, state.StopReason != "", simulationCloneCandidate)
	return SimulationRound{Index: state.CommittedRounds, Candidates: candidates}, ok
}

func (state SimulationState) Summary() SimulationSummary {
	return SimulationSummary{
		Parallel: state.Config.Parallel, MaxExecutions: state.Config.MaxExecutions, MaxForcedDecisions: state.Config.MaxForcedDecisions,
		MaxExplorationBytes: state.Config.MaxExplorationBytes, MaxResultBytes: state.Config.MaxResultBytes,
		FailureBudget: state.Config.FailureBudget, Limits: state.Config.Limits,
		LogicalExecutions: state.LogicalExecutions, CommittedRounds: state.CommittedRounds,
		Pending: uint64(len(state.Queue)), PendingBytes: state.PendingBytes, SeenCandidates: uint64(len(state.Seen)),
		DeduplicatedOutcomes: uint64(len(state.Outcomes)), DistinctFailures: uint64(len(state.FailureSignatures)),
		DeepestOverride: state.DeepestOverride, OmittedByExecutionBound: state.OmittedByExecutionBound,
		OmittedByDepth: state.OmittedByDepth, OmittedByDimension: state.OmittedByDimension, OmittedByCapacity: state.OmittedByCapacity,
		StopReason:      state.StopReason,
		BoundedComplete: state.StopReason == SimulationStopExhausted || state.StopReason == SimulationStopDepthComplete || state.StopReason == SimulationStopDimensionComplete,
	}
}

func SimulationStateSHA256(state SimulationState) (record.SHA256, error) {
	return simulationStateIdentity(state)
}

func ValidateCandidate(config SimulationConfig, candidate SimulationCandidate) error {
	expected, err := simulationNewCandidate(config, candidate.Overrides, candidate.ParentSHA256)
	if err != nil {
		return err
	}
	if !simulationSameCandidate(candidate, expected) {
		return errors.New("simulation exploration candidate identity does not match its contents")
	}
	return nil
}

func CanonicalCandidate(config SimulationConfig, overrides []ForcedDecision, parent record.SHA256) (SimulationCandidate, error) {
	if err := validateConfig(config); err != nil {
		return SimulationCandidate{}, err
	}
	return simulationNewCandidate(config, overrides, parent)
}

func CanonicalForcedDecision(forced ForcedDecision) (ForcedDecision, error) {
	forced.Identity = ""
	identity, err := forcedDecisionIdentity(forced)
	if err != nil {
		return ForcedDecision{}, err
	}
	forced.Identity = identity
	return forced, validateForcedDecision(forced)
}

func ForceDecision(decision Decision, selected uint32) (ForcedDecision, error) {
	return forcedDecisionFor(decision, selected)
}

func CommitSimulationRound(state SimulationState, round SimulationRound, results []SimulationResult) (SimulationState, SimulationRoundSegment, error) {
	before, err := simulationStateIdentity(state)
	if err != nil {
		return SimulationState{}, SimulationRoundSegment{}, err
	}
	expected, ok := state.NextRound()
	if !ok || round.Index != expected.Index || !simulationSameCandidates(round.Candidates, expected.Candidates) {
		return SimulationState{}, SimulationRoundSegment{}, errors.New("simulation exploration round does not match the current state")
	}
	if len(results) != len(round.Candidates) {
		return SimulationState{}, SimulationRoundSegment{}, errors.New("simulation exploration result count does not match its round")
	}
	next := simulationCloneState(state)
	next.Queue = next.Queue[len(round.Candidates):]
	next.LogicalExecutions += uint64(len(results))
	next.CommittedRounds++
	canonicalResults := make([]SimulationResult, len(results))
	children := make(map[record.SHA256]SimulationCandidate)
	for index, result := range results {
		candidate := round.Candidates[index]
		canonical, canonicalErr := canonicalResult(result, candidate, next.Config)
		if canonicalErr != nil {
			//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
			return SimulationState{}, SimulationRoundSegment{}, fmt.Errorf("simulation exploration result %d: %w", index, canonicalErr)
		}
		canonicalResults[index] = canonical
		next.Outcomes = simulationInsertIdentity(next.Outcomes, canonical.OutcomeSHA256)
		if canonical.Failed {
			next.FailureSignatures = simulationInsertIdentity(next.FailureSignatures, canonical.FailureSHA256)
			if uint64(len(next.FailureSignatures)) >= next.Config.FailureBudget {
				next.StopReason = SimulationStopFailureBudget
			}
		}
		if next.StopReason == "" && !canonical.Diverged {
			if err := simulationExpandCandidate(&next, candidate, canonical.Decisions, children); err != nil {
				return SimulationState{}, SimulationRoundSegment{}, err
			}
		}
	}
	admitted := make([]SimulationCandidate, 0, len(children))
	for _, candidate := range children {
		admitted = append(admitted, candidate)
	}
	simulationSortCandidates(admitted)
	if err := simulationAdmitChildren(&next, admitted); err != nil {
		return SimulationState{}, SimulationRoundSegment{}, err
	}
	if next.StopReason == "" && len(next.Queue) == 0 {
		switch {
		case next.OmittedByExecutionBound != 0 || next.LogicalExecutions >= next.Config.MaxExecutions:
			next.StopReason = SimulationStopMaxExecutions
		case next.OmittedByDepth != 0:
			next.StopReason = SimulationStopDepthComplete
		case next.OmittedByDimension != 0:
			next.StopReason = SimulationStopDimensionComplete
		default:
			next.StopReason = SimulationStopExhausted
		}
	}
	next.PendingBytes, err = simulationQueueBytes(next.Queue)
	if err != nil {
		return SimulationState{}, SimulationRoundSegment{}, err
	}
	after, err := simulationStateIdentity(next)
	if err != nil {
		return SimulationState{}, SimulationRoundSegment{}, err
	}
	segment := SimulationRoundSegment{
		Schema: SimulationRoundSegmentSchema, Index: round.Index, BeforeSHA256: before, AfterSHA256: after, Results: canonicalResults,
	}
	segment.SHA256, err = simulationSegmentIdentity(segment)
	if err != nil {
		return SimulationState{}, SimulationRoundSegment{}, err
	}
	return next, segment, nil
}

func ValidateResult(config SimulationConfig, candidate SimulationCandidate, result SimulationResult) error {
	_, err := canonicalResult(result, candidate, config)
	return err
}

func SimulationReplaySegment(state SimulationState, segment SimulationRoundSegment) (SimulationState, error) {
	identity, err := simulationSegmentIdentity(segment)
	if err != nil || identity != segment.SHA256 {
		return SimulationState{}, errors.Join(errors.New("simulation exploration segment identity does not match"), err)
	}
	before, err := simulationStateIdentity(state)
	if err != nil || before != segment.BeforeSHA256 || segment.Schema != SimulationRoundSegmentSchema {
		return SimulationState{}, errors.Join(errors.New("simulation exploration segment does not link to its state"), err)
	}
	round, ok := state.NextRound()
	if !ok || round.Index != segment.Index || len(round.Candidates) != len(segment.Results) {
		return SimulationState{}, errors.New("simulation exploration segment round is unavailable")
	}
	results := cloneResults(segment.Results)
	next, regenerated, err := CommitSimulationRound(state, round, results)
	if err != nil {
		return SimulationState{}, err
	}
	left, err := json.Marshal(segment)
	if err != nil {
		return SimulationState{}, err
	}
	right, err := json.Marshal(regenerated)
	if err != nil {
		return SimulationState{}, err
	}
	after, err := simulationStateIdentity(next)
	if err != nil || !bytes.Equal(left, right) || after != segment.AfterSHA256 {
		return SimulationState{}, errors.Join(errors.New("simulation exploration segment replay changed its canonical result"), err)
	}
	return next, nil
}

func canonicalResult(result SimulationResult, candidate SimulationCandidate, config SimulationConfig) (SimulationResult, error) {
	if result.CandidateSHA256 != candidate.SHA256 {
		return SimulationResult{}, errors.New("candidate identity does not match")
	}
	if _, err := result.OutcomeSHA256.Bytes(); err != nil {
		//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
		return SimulationResult{}, fmt.Errorf("outcome identity: %w", err)
	}
	if result.Failed {
		if _, err := result.FailureSHA256.Bytes(); err != nil {
			//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
			return SimulationResult{}, fmt.Errorf("failure identity: %w", err)
		}
	} else if result.FailureSHA256 != "" {
		return SimulationResult{}, errors.New("successful result contains a failure identity")
	}
	result.Decisions = cloneDecisions(result.Decisions)
	sortDecisions(result.Decisions)
	for index, decision := range result.Decisions {
		if err := validateDecision(decision); err != nil {
			//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
			return SimulationResult{}, fmt.Errorf("decision %d: %w", index, err)
		}
		if index != 0 && sameDecisionKey(result.Decisions[index-1], decision) {
			return SimulationResult{}, errors.New("decisions contain a duplicate dimension ordinal")
		}
	}
	if !result.Diverged {
		for _, override := range candidate.Overrides {
			index, found := findDecision(result.Decisions, override.Dimension, override.Ordinal)
			if !found || !forcedDecisionMatches(override, result.Decisions[index]) {
				return SimulationResult{}, errors.New("result does not prove a forced decision")
			}
		}
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return SimulationResult{}, err
	}
	if uint64(len(encoded)) > config.MaxResultBytes {
		return SimulationResult{}, fmt.Errorf("result requires %d bytes, exceeding the %d-byte result bound", len(encoded), config.MaxResultBytes)
	}
	return result, nil
}

func simulationExpandCandidate(state *SimulationState, parent SimulationCandidate, decisions []Decision, children map[record.SHA256]SimulationCandidate) error {
	for _, decision := range decisions {
		if _, found := findForcedDecision(parent.Overrides, decision.Dimension, decision.Ordinal); found {
			continue
		}
		alternatives := uint64(len(decision.Alternatives) - 1)
		if decision.Ordinal >= dimensionLimit(state.Config.Limits, decision.Dimension) {
			state.OmittedByDimension += alternatives
			continue
		}
		if uint64(len(parent.Overrides)) >= state.Config.MaxForcedDecisions {
			state.OmittedByDepth += alternatives
			continue
		}
		for rank := range decision.Alternatives {
			if uint32(rank) == decision.Selected {
				continue
			}
			override, err := forcedDecisionFor(decision, uint32(rank))
			if err != nil {
				return err
			}
			overrides := append(cloneForcedDecisions(parent.Overrides), override)
			sortForcedDecisions(overrides)
			child, err := simulationNewCandidate(state.Config, overrides, parent.SHA256)
			if err != nil {
				return err
			}
			if simulationContainsIdentity(state.Seen, child.SHA256) {
				continue
			}
			if existing, found := children[child.SHA256]; found {
				if !slices.EqualFunc(existing.Overrides, child.Overrides, sameForcedDecision) {
					return errors.New("simulation exploration candidate identity collision")
				}
				if child.ParentSHA256 < existing.ParentSHA256 {
					children[child.SHA256] = child
				}
				continue
			}
			children[child.SHA256] = child
		}
	}
	return nil
}

func simulationAdmitChildren(state *SimulationState, children []SimulationCandidate) error {
	pendingBytes, err := simulationQueueBytes(state.Queue)
	if err != nil {
		return err
	}
	for index, candidate := range children {
		if uint64(len(state.Seen)) >= state.Config.MaxExecutions {
			state.OmittedByExecutionBound += uint64(len(children) - index)
			break
		}
		entryBytes, err := simulationCandidateBytes(candidate)
		if err != nil {
			return err
		}
		if entryBytes > ^uint64(0)-pendingBytes {
			return errors.New("simulation exploration byte accounting overflow")
		}
		if pendingBytes+entryBytes > state.Config.MaxExplorationBytes {
			state.OmittedByCapacity += uint64(len(children) - index)
			if state.StopReason == "" {
				state.StopReason = SimulationStopExplorationCapacity
			}
			break
		}
		pendingBytes += entryBytes
		state.Queue = append(state.Queue, simulationCloneCandidate(candidate))
		state.Seen = simulationInsertIdentity(state.Seen, candidate.SHA256)
		state.DeepestOverride = max(state.DeepestOverride, uint64(len(candidate.Overrides)))
	}
	simulationSortCandidates(state.Queue)
	state.PendingBytes = pendingBytes
	return nil
}

func simulationNewCandidate(config SimulationConfig, overrides []ForcedDecision, parent record.SHA256) (SimulationCandidate, error) {
	overrides = cloneForcedDecisions(overrides)
	sortForcedDecisions(overrides)
	for index, override := range overrides {
		if err := validateForcedDecision(override); err != nil {
			//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
			return SimulationCandidate{}, fmt.Errorf("forced decision %d: %w", index, err)
		}
		if index != 0 && sameForcedKey(overrides[index-1], override) {
			return SimulationCandidate{}, errors.New("candidate contains duplicate forced decisions")
		}
	}
	var identityOverrides []any
	if overrides != nil {
		identityOverrides = make([]any, len(overrides))
		for index, override := range overrides {
			identityOverrides[index] = forcedDecisionIdentityProjection(override, true)
		}
	}
	//gomad:dynamic reviewed 2026-10-13 stdlib JSON marshals only domain-owned simulation identity values
	encoded, err := json.Marshal(map[string]any{
		"base_seed": config.BaseSeed, "controller_sha256": config.ControllerSHA256,
		"execution_sha256": config.ExecutionSHA256, "overrides": identityOverrides,
	})
	if err != nil {
		return SimulationCandidate{}, err
	}
	candidate := SimulationCandidate{
		SHA256: record.DomainHash(simulationCandidateDomain, encoded), ParentSHA256: parent, Overrides: overrides,
	}
	if parent != "" {
		if _, err := parent.Bytes(); err != nil {
			//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
			return SimulationCandidate{}, fmt.Errorf("candidate parent identity: %w", err)
		}
	}
	return candidate, nil
}

func forcedDecisionFor(decision Decision, selected uint32) (ForcedDecision, error) {
	if err := validateDecision(decision); err != nil {
		return ForcedDecision{}, err
	}
	if selected >= uint32(len(decision.Alternatives)) {
		return ForcedDecision{}, errors.New("forced decision selected rank is invalid")
	}
	forced := ForcedDecision{
		Dimension: decision.Dimension, Ordinal: decision.Ordinal, SiteSHA256: decision.SiteSHA256,
		Alternatives: uint32(len(decision.Alternatives)), AlternativeSetSHA256: decision.AlternativeSetSHA256,
		Selected: selected, SelectedSHA256: decision.Alternatives[selected],
	}
	if decision.Dimension == DimensionRuntime {
		forced.Control = append([]byte(nil), decision.AlternativeControls[selected]...)
	}
	var err error
	forced.Identity, err = forcedDecisionIdentity(forced)
	if err != nil {
		return ForcedDecision{}, err
	}
	return forced, validateForcedDecision(forced)
}

func validateConfig(config SimulationConfig) error {
	if config.Parallel <= 0 || config.MaxExecutions == 0 || config.MaxExplorationBytes == 0 || config.MaxResultBytes == 0 || config.FailureBudget == 0 {
		return errors.New("simulation exploration requires positive parallel, execution, exploration, result, and failure bounds")
	}
	for name, limit := range map[string]uint64{
		"runtime": config.Limits.Runtime, "scenario": config.Limits.Scenario, "network": config.Limits.Network,
		"storage": config.Limits.Storage, "fault": config.Limits.Fault, "crash": config.Limits.Crash,
	} {
		if limit == 0 {
			return fmt.Errorf("simulation exploration %s dimension bound is zero", name)
		}
	}
	if _, err := config.ExecutionSHA256.Bytes(); err != nil {
		//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
		return fmt.Errorf("simulation exploration execution identity: %w", err)
	}
	if config.ControllerSHA256 != SimulationImplementationSHA256() {
		return errors.New("simulation exploration controller identity does not match this implementation")
	}
	return nil
}

func validateDecision(decision Decision) error {
	if dimensionOrder(decision.Dimension) < 0 {
		return fmt.Errorf("unknown decision dimension %q", decision.Dimension)
	}
	if _, err := decision.SiteSHA256.Bytes(); err != nil {
		//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
		return fmt.Errorf("decision site identity: %w", err)
	}
	if len(decision.Alternatives) < 2 || uint64(len(decision.Alternatives)) > uint64(^uint32(0)) {
		return errors.New("decision alternative count is invalid")
	}
	seen := make(map[record.SHA256]struct{}, len(decision.Alternatives))
	for _, alternative := range decision.Alternatives {
		if _, err := alternative.Bytes(); err != nil {
			//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
			return fmt.Errorf("decision alternative identity: %w", err)
		}
		if _, ok := seen[alternative]; ok {
			return errors.New("decision alternatives are duplicated")
		}
		seen[alternative] = struct{}{}
	}
	if decision.Selected >= uint32(len(decision.Alternatives)) {
		return errors.New("decision selected rank is invalid")
	}
	if decision.Dimension == DimensionRuntime {
		if len(decision.AlternativeControls) != len(decision.Alternatives) {
			return errors.New("runtime decision controls do not match its alternatives")
		}
		for rank, control := range decision.AlternativeControls {
			if uint32(rank) == decision.Selected {
				if len(control) != 0 {
					return errors.New("runtime decision selected alternative contains a control prefix")
				}
			} else if len(control) == 0 {
				return errors.New("runtime decision alternative control is missing")
			}
		}
	} else if decision.AlternativeControls != nil {
		return errors.New("non-runtime decision contains runtime controls")
	}
	set, err := alternativeSetIdentity(decision.Dimension, decision.Ordinal, decision.SiteSHA256, decision.Alternatives)
	if err != nil || set != decision.AlternativeSetSHA256 {
		return errors.Join(errors.New("decision alternative-set identity does not match"), err)
	}
	identity, err := decisionIdentity(decision)
	if err != nil || identity != decision.Identity {
		return errors.Join(errors.New("decision identity does not match"), err)
	}
	return nil
}

func validateForcedDecision(forced ForcedDecision) error {
	if dimensionOrder(forced.Dimension) < 0 || forced.Alternatives < 2 || forced.Selected >= forced.Alternatives {
		return errors.New("forced decision shape is invalid")
	}
	for _, identity := range []record.SHA256{forced.SiteSHA256, forced.AlternativeSetSHA256, forced.SelectedSHA256, forced.Identity} {
		if _, err := identity.Bytes(); err != nil {
			return err
		}
	}
	if forced.Dimension == DimensionRuntime && len(forced.Control) == 0 {
		return errors.New("runtime forced decision control is missing")
	}
	if forced.Dimension != DimensionRuntime && len(forced.Control) != 0 {
		return errors.New("non-runtime forced decision contains runtime control")
	}
	identity, err := forcedDecisionIdentity(forced)
	if err != nil || identity != forced.Identity {
		return errors.Join(errors.New("forced decision identity does not match"), err)
	}
	return nil
}

func alternativeSetIdentity(dimension Dimension, ordinal uint64, site record.SHA256, alternatives []record.SHA256) (record.SHA256, error) {
	if dimensionOrder(dimension) < 0 {
		return "", fmt.Errorf("unknown decision dimension %q", dimension)
	}
	if _, err := site.Bytes(); err != nil {
		return "", err
	}
	for _, alternative := range alternatives {
		if _, err := alternative.Bytes(); err != nil {
			return "", err
		}
	}
	//gomad:dynamic reviewed 2026-10-13 stdlib JSON marshals only domain-owned simulation identity values
	encoded, err := json.Marshal(map[string]any{
		"alternatives": append([]record.SHA256(nil), alternatives...), "dimension": dimension,
		"ordinal": ordinal, "site_sha256": site,
	})
	if err != nil {
		return "", err
	}
	return record.DomainHash("gomad3-simulation-exploration-alternative-set/v1", encoded), nil
}

func decisionIdentity(decision Decision) (record.SHA256, error) {
	decision.Identity = ""
	decision.Alternatives = append([]record.SHA256(nil), decision.Alternatives...)
	decision.AlternativeControls = cloneControls(decision.AlternativeControls)
	//gomad:dynamic reviewed 2026-10-13 stdlib JSON marshals only domain-owned simulation identity values
	encoded, err := json.Marshal(simulationDecisionIdentityProjection(decision))
	if err != nil {
		return "", err
	}
	return record.DomainHash(decisionDomain, encoded), nil
}

func forcedDecisionIdentity(forced ForcedDecision) (record.SHA256, error) {
	forced.Identity = ""
	//gomad:dynamic reviewed 2026-10-13 stdlib JSON marshals only domain-owned simulation identity values
	encoded, err := json.Marshal(forcedDecisionIdentityProjection(forced, true))
	if err != nil {
		return "", err
	}
	return record.DomainHash(forcedDomain, encoded), nil
}

func forcedDecisionMatches(forced ForcedDecision, decision Decision) bool {
	return forced.Dimension == decision.Dimension && forced.Ordinal == decision.Ordinal &&
		forced.SiteSHA256 == decision.SiteSHA256 && forced.Alternatives == uint32(len(decision.Alternatives)) &&
		forced.AlternativeSetSHA256 == decision.AlternativeSetSHA256 && forced.Selected == decision.Selected &&
		forced.SelectedSHA256 == decision.Alternatives[decision.Selected]
}

func dimensionLimit(limits DimensionLimits, dimension Dimension) uint64 {
	switch dimension {
	case DimensionRuntime:
		return limits.Runtime
	case DimensionScenario:
		return limits.Scenario
	case DimensionNetwork:
		return limits.Network
	case DimensionStorage:
		return limits.Storage
	case DimensionFault:
		return limits.Fault
	case DimensionCrash:
		return limits.Crash
	default:
		return 0
	}
}

func dimensionOrder(dimension Dimension) int {
	switch dimension {
	case DimensionRuntime:
		return 0
	case DimensionScenario:
		return 1
	case DimensionNetwork:
		return 2
	case DimensionStorage:
		return 3
	case DimensionFault:
		return 4
	case DimensionCrash:
		return 5
	default:
		return -1
	}
}

func sortDecisions(decisions []Decision) {
	sort.Slice(decisions, func(left, right int) bool {
		if dimensionOrder(decisions[left].Dimension) != dimensionOrder(decisions[right].Dimension) {
			return dimensionOrder(decisions[left].Dimension) < dimensionOrder(decisions[right].Dimension)
		}
		return decisions[left].Ordinal < decisions[right].Ordinal
	})
}

func sortForcedDecisions(decisions []ForcedDecision) {
	sort.Slice(decisions, func(left, right int) bool {
		if dimensionOrder(decisions[left].Dimension) != dimensionOrder(decisions[right].Dimension) {
			return dimensionOrder(decisions[left].Dimension) < dimensionOrder(decisions[right].Dimension)
		}
		return decisions[left].Ordinal < decisions[right].Ordinal
	})
}

func simulationSortCandidates(candidates []SimulationCandidate) {
	sort.Slice(candidates, func(left, right int) bool {
		if len(candidates[left].Overrides) != len(candidates[right].Overrides) {
			return len(candidates[left].Overrides) < len(candidates[right].Overrides)
		}
		return candidates[left].SHA256 < candidates[right].SHA256
	})
}

func findDecision(decisions []Decision, dimension Dimension, ordinal uint64) (int, bool) {
	for index, decision := range decisions {
		if decision.Dimension == dimension && decision.Ordinal == ordinal {
			return index, true
		}
	}
	return 0, false
}

func findForcedDecision(decisions []ForcedDecision, dimension Dimension, ordinal uint64) (int, bool) {
	for index, decision := range decisions {
		if decision.Dimension == dimension && decision.Ordinal == ordinal {
			return index, true
		}
	}
	return 0, false
}

func sameDecisionKey(left, right Decision) bool {
	return left.Dimension == right.Dimension && left.Ordinal == right.Ordinal
}

func sameForcedKey(left, right ForcedDecision) bool {
	return left.Dimension == right.Dimension && left.Ordinal == right.Ordinal
}

func simulationSameCandidates(left, right []SimulationCandidate) bool {
	return slices.EqualFunc(left, right, simulationSameCandidate)
}

func simulationSameCandidate(left, right SimulationCandidate) bool {
	return left.SHA256 == right.SHA256 && left.ParentSHA256 == right.ParentSHA256 && slices.EqualFunc(left.Overrides, right.Overrides, sameForcedDecision)
}

func sameForcedDecision(left, right ForcedDecision) bool {
	return left.Dimension == right.Dimension && left.Ordinal == right.Ordinal && left.SiteSHA256 == right.SiteSHA256 &&
		left.Alternatives == right.Alternatives && left.AlternativeSetSHA256 == right.AlternativeSetSHA256 &&
		left.Selected == right.Selected && left.SelectedSHA256 == right.SelectedSHA256 && left.Identity == right.Identity &&
		bytes.Equal(left.Control, right.Control)
}

func simulationStateIdentity(state SimulationState) (record.SHA256, error) {
	cloned := simulationCloneState(state)
	encoded, err := json.Marshal(cloned)
	if err != nil {
		return "", err
	}
	return record.DomainHash(simulationStateDomain, encoded), nil
}

func simulationSegmentIdentity(segment SimulationRoundSegment) (record.SHA256, error) {
	segment.SHA256 = ""
	segment.Results = cloneResults(segment.Results)
	encoded, err := json.Marshal(segment)
	if err != nil {
		return "", err
	}
	return record.DomainHash(simulationRoundSegmentDomain, encoded), nil
}

func simulationCandidateBytes(candidate SimulationCandidate) (uint64, error) {
	encoded, err := json.Marshal(candidate)
	if err != nil {
		return 0, err
	}
	return uint64(len(encoded)), nil
}

func simulationQueueBytes(queue []SimulationCandidate) (uint64, error) {
	return SumBytes(queue, simulationCandidateBytes)
}

func simulationInsertIdentity(values []record.SHA256, value record.SHA256) []record.SHA256 {
	return InsertIdentity(values, value)
}

func simulationContainsIdentity(values []record.SHA256, value record.SHA256) bool {
	return ContainsIdentity(values, value)
}

func simulationCloneState(state SimulationState) SimulationState {
	state.Queue = cloneCandidates(state.Queue)
	state.Seen = append([]record.SHA256(nil), state.Seen...)
	state.Outcomes = append([]record.SHA256(nil), state.Outcomes...)
	state.FailureSignatures = append([]record.SHA256(nil), state.FailureSignatures...)
	return state
}

func cloneCandidates(candidates []SimulationCandidate) []SimulationCandidate {
	cloned := make([]SimulationCandidate, len(candidates))
	for index, candidate := range candidates {
		cloned[index] = simulationCloneCandidate(candidate)
	}
	return cloned
}

func simulationCloneCandidate(candidate SimulationCandidate) SimulationCandidate {
	candidate.Overrides = cloneForcedDecisions(candidate.Overrides)
	return candidate
}

func cloneForcedDecisions(decisions []ForcedDecision) []ForcedDecision {
	if decisions == nil {
		return nil
	}
	cloned := make([]ForcedDecision, len(decisions))
	for index, decision := range decisions {
		cloned[index] = decision
		cloned[index].Control = append([]byte(nil), decision.Control...)
	}
	return cloned
}

func cloneDecisions(decisions []Decision) []Decision {
	cloned := make([]Decision, len(decisions))
	for index, decision := range decisions {
		cloned[index] = decision
		cloned[index].Alternatives = append([]record.SHA256(nil), decision.Alternatives...)
		cloned[index].AlternativeControls = cloneControls(decision.AlternativeControls)
	}
	return cloned
}

func cloneControls(controls [][]byte) [][]byte {
	if controls == nil {
		return nil
	}
	cloned := make([][]byte, len(controls))
	for index, control := range controls {
		cloned[index] = append([]byte(nil), control...)
	}
	return cloned
}

func cloneResults(results []SimulationResult) []SimulationResult {
	cloned := make([]SimulationResult, len(results))
	for index, result := range results {
		cloned[index] = result
		cloned[index].Decisions = cloneDecisions(result.Decisions)
	}
	return cloned
}

func simulationDecisionIdentityProjection(decision Decision) map[string]any {
	projection := map[string]any{
		"alternative_set_sha256": decision.AlternativeSetSHA256,
		"alternatives":           append([]record.SHA256(nil), decision.Alternatives...),
		"dimension":              decision.Dimension,
		"identity":               decision.Identity,
		"ordinal":                decision.Ordinal,
		"selected":               decision.Selected,
		"site_sha256":            decision.SiteSHA256,
	}
	if len(decision.AlternativeControls) != 0 {
		projection["alternative_controls"] = cloneControls(decision.AlternativeControls)
	}
	return projection
}

func forcedDecisionIdentityProjection(forced ForcedDecision, includeControl bool) map[string]any {
	projection := map[string]any{
		"alternative_set_sha256": forced.AlternativeSetSHA256,
		"alternatives":           forced.Alternatives,
		"dimension":              forced.Dimension,
		"identity":               forced.Identity,
		"ordinal":                forced.Ordinal,
		"selected":               forced.Selected,
		"selected_sha256":        forced.SelectedSHA256,
		"site_sha256":            forced.SiteSHA256,
	}
	if includeControl && len(forced.Control) != 0 {
		projection["control"] = append([]byte(nil), forced.Control...)
	}
	return projection
}
