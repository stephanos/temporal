package gomad3sim

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
)

var ErrFaultPlanMismatch = errors.New("simulation fault does not match the planned action")
var ErrFaultInapplicable = errors.New("simulation fault is inapplicable")

type FaultPlanError struct {
	Ordinal  uint64
	Expected *FaultAction
	Actual   *FaultAction
	Cause    error
}

func (err *FaultPlanError) Error() string {
	return fmt.Sprintf("%v: ordinal=%d expected=%+v actual=%+v", err.Cause, err.Ordinal, err.Expected, err.Actual)
}

func (err *FaultPlanError) Unwrap() error {
	return err.Cause
}

func (cluster *inProcessCluster) ApplyFault(ctx context.Context, action FaultAction) (FaultRealization, error) {
	if err := actionContextError(ctx); err != nil {
		return FaultRealization{}, err
	}
	if err := validateFaultAction(action); err != nil {
		return FaultRealization{}, err
	}
	if !faultMatchEmpty(action.Match) {
		return FaultRealization{}, errors.New("matched fault requires controller triggering")
	}
	return cluster.applyFault(ctx, action, FaultMatch{})
}

func (cluster *inProcessCluster) TriggerFault(ctx context.Context, event FaultMatch) (FaultRealization, bool, error) {
	if err := actionContextError(ctx); err != nil {
		return FaultRealization{}, false, err
	}
	if err := validateFaultEvent(event); err != nil {
		return FaultRealization{}, false, err
	}
	key, err := faultEventOccurrenceIdentity(event)
	if err != nil {
		return FaultRealization{}, false, err
	}
	cluster.mu.Lock()
	if cluster.closing || cluster.faultPending {
		cluster.mu.Unlock()
		return FaultRealization{}, false, ErrInvalidTransition
	}
	if cluster.replayFailure != nil {
		err := cluster.replayFailure
		cluster.mu.Unlock()
		return FaultRealization{}, false, err
	}
	cluster.faultOccurrences[key]++
	event.Occurrence = cluster.faultOccurrences[key]
	planned, plannedOK := cluster.plannedFaultLocked(uint64(len(cluster.faults)))
	if !plannedOK || faultMatchEmpty(planned.Match) || !faultMatches(planned.Match, event) {
		cluster.mu.Unlock()
		return FaultRealization{}, false, nil
	}
	cluster.mu.Unlock()
	realization, err := cluster.applyFault(ctx, planned, event)
	return realization, true, err
}

func (cluster *inProcessCluster) applyFault(ctx context.Context, action FaultAction, matched FaultMatch) (FaultRealization, error) {
	cluster.mu.Lock()
	if cluster.closing || cluster.faultPending {
		cluster.mu.Unlock()
		return FaultRealization{}, ErrInvalidTransition
	}
	if cluster.replayFailure != nil {
		err := cluster.replayFailure
		cluster.mu.Unlock()
		return FaultRealization{}, err
	}
	ordinal := uint64(len(cluster.faults))
	planned, plannedOK := cluster.plannedFaultLocked(ordinal)
	if !plannedOK || !equalFaultAction(planned, action) {
		err := cluster.faultMismatchLocked(ordinal, optionalFaultAction(planned, plannedOK), &action, ErrFaultPlanMismatch)
		cluster.mu.Unlock()
		return FaultRealization{}, err
	}
	realization, err := cluster.resolveFaultLocked(ordinal, action, matched)
	if err != nil {
		if cluster.replay != nil {
			err = cluster.faultMismatchLocked(ordinal, cluster.expectedFaultActionLocked(ordinal), &action, ErrFaultInapplicable)
		}
		cluster.mu.Unlock()
		return FaultRealization{}, err
	}
	if cluster.replay != nil {
		expected, ok := cluster.expectedFaultLocked(ordinal)
		if !ok || !equalFaultRealization(expected, realization) {
			err := cluster.faultRealizationMismatchLocked(ordinal, optionalFaultRealization(expected, ok), &realization)
			cluster.mu.Unlock()
			return FaultRealization{}, err
		}
	}
	if err := checkCapacity("fault_actions", ordinal+1, cluster.limits.FaultActions); err != nil {
		cluster.mu.Unlock()
		return FaultRealization{}, err
	}
	cluster.faultPending = true
	cluster.mu.Unlock()

	applyErr := cluster.applyResolvedFault(ctx, realization)

	cluster.mu.Lock()
	cluster.faultPending = false
	defer cluster.mu.Unlock()
	if applyErr != nil {
		if divergence := cluster.retainReplayFailureLocked(applyErr); divergence != nil {
			return FaultRealization{}, divergence
		}
		if cluster.replay != nil {
			return FaultRealization{}, cluster.faultMismatchLocked(ordinal, cluster.expectedFaultActionLocked(ordinal), &action, ErrFaultInapplicable)
		}
		return FaultRealization{}, &FaultPlanError{Ordinal: ordinal, Expected: faultActionPointer(planned), Actual: faultActionPointer(action), Cause: errors.Join(ErrFaultInapplicable, applyErr)}
	}
	cluster.faults = append(cluster.faults, realization)
	return cloneFaultRealization(realization), nil
}

func (cluster *inProcessCluster) resolveFaultLocked(ordinal uint64, action FaultAction, matched FaultMatch) (FaultRealization, error) {
	realization := FaultRealization{Ordinal: ordinal, Action: cloneFaultAction(action), Matched: matched}
	switch action.Kind {
	case FaultGracefulStop, FaultHarshCrash, FaultRestart:
		nodeID := action.Node
		if action.TargetFrom != "" {
			for _, prior := range cluster.faults {
				if prior.Action.ID == action.TargetFrom {
					nodeID = prior.Target.Node
					break
				}
			}
			if nodeID == "" {
				return FaultRealization{}, ErrFaultInapplicable
			}
		} else if nodeID == "" {
			selected := selectFaultTarget(cluster.seed, ordinal, action.ID, uint64(len(action.Candidates)))
			nodeID = action.Candidates[selected]
		}
		node := cluster.nodes[nodeID]
		if node == nil || action.Match.Node != "" && action.Match.Node != nodeID {
			return FaultRealization{}, ErrFaultInapplicable
		}
		switch action.Kind {
		case FaultGracefulStop, FaultHarshCrash:
			if node.state != NodeStateRunning || node.operation != nil {
				return FaultRealization{}, ErrFaultInapplicable
			}
			realization.Target = node.handle
		case FaultRestart:
			if node.operation != nil || node.state != NodeStateStopped && node.state != NodeStateCrashed && node.state != NodeStateExited && node.state != NodeStateFailed {
				return FaultRealization{}, ErrFaultInapplicable
			}
			realization.Target = NodeHandle{Node: nodeID, Incarnation: node.handle.Incarnation + 1}
		}
		if action.Match.Incarnation != 0 && action.Match.Incarnation != realization.Target.Incarnation {
			return FaultRealization{}, ErrFaultInapplicable
		}
	case FaultDisconnect, FaultReconnect, FaultDelay:
		if cluster.nodes[action.From] == nil || cluster.nodes[action.To] == nil {
			return FaultRealization{}, ErrFaultInapplicable
		}
	case FaultPartition, FaultHeal:
		for _, node := range append(append([]NodeID(nil), action.Left...), action.Right...) {
			if cluster.nodes[node] == nil {
				return FaultRealization{}, ErrFaultInapplicable
			}
		}
	default:
		return FaultRealization{}, ErrFaultInapplicable
	}
	identity, err := faultRealizationIdentity(realization)
	if err != nil {
		return FaultRealization{}, err
	}
	realization.Identity = identity
	return realization, nil
}

func (cluster *inProcessCluster) applyResolvedFault(ctx context.Context, realization FaultRealization) error {
	action := realization.Action
	switch action.Kind {
	case FaultGracefulStop:
		return cluster.Stop(ctx, realization.Target)
	case FaultHarshCrash:
		return cluster.crash(ctx, realization.Target, action.Persistence == FaultPersistencePersisted)
	case FaultRestart:
		handle, err := cluster.Restart(ctx, realization.Target.Node)
		if err != nil {
			return err
		}
		if handle != realization.Target {
			return ErrInvalidTransition
		}
		return nil
	case FaultDisconnect:
		return cluster.applyDirectionalNetworkFault(ctx, action.From, action.To, NetworkDisconnect, 0)
	case FaultReconnect:
		return cluster.applyDirectionalNetworkFault(ctx, action.From, action.To, NetworkReconnect, 0)
	case FaultDelay:
		return cluster.applyDirectionalNetworkFault(ctx, action.From, action.To, NetworkDirectionalDelay, action.DelayNanos)
	case FaultPartition, FaultHeal:
		if err := cluster.beginCall(ctx); err != nil {
			return err
		}
		defer cluster.endCall()
		return runtimeNetworkGroup(cluster.runtimeRun, action.Left, action.Right, action.Kind == FaultHeal)
	default:
		return ErrFaultInapplicable
	}
}

func (cluster *inProcessCluster) applyDirectionalNetworkFault(ctx context.Context, from, to NodeID, kind NetworkTransitionKind, delayNanos uint64) error {
	if err := cluster.beginCall(ctx); err != nil {
		return err
	}
	defer cluster.endCall()
	switch kind {
	case NetworkDisconnect:
		return runtimeNetworkPartition(cluster.runtimeRun, from, to, false)
	case NetworkReconnect:
		return runtimeNetworkHeal(cluster.runtimeRun, from, to, false)
	case NetworkDirectionalDelay:
		return runtimeNetworkDelay(cluster.runtimeRun, from, to, delayNanos, false)
	default:
		return ErrFaultInapplicable
	}
}

func (cluster *inProcessCluster) Observe(ctx context.Context, observation Observation) error {
	if err := cluster.beginCall(ctx); err != nil {
		return err
	}
	defer cluster.endCall()
	if observation.Ordinal != 0 || observation.FullSHA256 != "" || observation.Identity != "" {
		return errors.New("scenario observation contains record-owned fields")
	}
	if err := validateID("observation ID", observation.ID); err != nil {
		return err
	}
	if err := validateID("observation kind", observation.Kind); err != nil {
		return err
	}
	if observation.Handle.Node == "" != (observation.Handle.Incarnation == 0) {
		return errors.New("scenario observation contains a partial node handle")
	}
	observation.Value = append([]byte(nil), observation.Value...)
	digest := sha256.Sum256(observation.Value)
	observation.FullSHA256 = fmt.Sprintf("sha256:%x", digest)
	cluster.mu.Lock()
	defer cluster.mu.Unlock()
	observation.Ordinal = uint64(len(cluster.observations))
	identity, err := observationIdentity(observation)
	if err != nil {
		return err
	}
	observation.Identity = identity
	if err := checkCapacity("observations", observation.Ordinal+1, cluster.limits.Observations); err != nil {
		return err
	}
	if err := cluster.reserveScenarioEvidenceLocked(uint64(len(observation.Value))); err != nil {
		return err
	}
	if cluster.replay != nil {
		var expected Observation
		if observation.Ordinal < uint64(len(cluster.replay.Observations)) {
			expected = cluster.replay.Observations[observation.Ordinal]
		}
		if !equalObservation(expected, observation) {
			cluster.scenarioEvidenceBytes -= uint64(len(observation.Value))
			return cluster.evidenceDivergenceLocked(ReplayDimensionEvidence, observation.Ordinal, expected.Identity, observation.Identity)
		}
	}
	cluster.observations = append(cluster.observations, observation)
	return nil
}

func (cluster *inProcessCluster) RecordOperation(ctx context.Context, operation HistoryOperation) error {
	if err := cluster.beginCall(ctx); err != nil {
		return err
	}
	defer cluster.endCall()
	operation = cloneHistoryOperation(operation)
	bytesRequired := historyOperationBytes(operation)
	if err := ValidateHistory([]HistoryOperation{operation}, cluster.limits.ScenarioEvidenceBytes); err != nil {
		return err
	}
	cluster.mu.Lock()
	defer cluster.mu.Unlock()
	ordinal := uint64(len(cluster.history))
	if err := checkCapacity("history_operations", ordinal+1, cluster.limits.HistoryOperations); err != nil {
		return err
	}
	if _, ok := historyOperationByID(cluster.history, operation.ID); ok {
		return fmt.Errorf("history operation ID %q is duplicated", operation.ID)
	}
	if err := cluster.reserveScenarioEvidenceLocked(bytesRequired); err != nil {
		return err
	}
	if cluster.replay != nil {
		var expected HistoryOperation
		if ordinal < uint64(len(cluster.replay.History)) {
			expected = cluster.replay.History[ordinal]
		}
		if !equalHistoryOperation(expected, operation) {
			cluster.scenarioEvidenceBytes -= bytesRequired
			expectedIdentity, _ := hashCanonical("gomad3-history-operation/v1", expected)
			actualIdentity, _ := hashCanonical("gomad3-history-operation/v1", operation)
			return cluster.evidenceDivergenceLocked(ReplayDimensionEvidence, ordinal, expectedIdentity, actualIdentity)
		}
	}
	cluster.history = append(cluster.history, operation)
	return nil
}

func (cluster *inProcessCluster) RecordOracle(ctx context.Context, result OracleResult) error {
	if err := cluster.beginCall(ctx); err != nil {
		return err
	}
	defer cluster.endCall()
	result = cloneOracleResult(result)
	if err := validateOracleResult(result, cluster.limits.ScenarioEvidenceBytes); err != nil {
		return err
	}
	bytesRequired := oracleEvidenceBytes(result)
	cluster.mu.Lock()
	defer cluster.mu.Unlock()
	ordinal := uint64(len(cluster.oracles))
	if err := checkCapacity("oracle_results", ordinal+1, cluster.limits.OracleResults); err != nil {
		return err
	}
	if err := cluster.reserveScenarioEvidenceLocked(bytesRequired); err != nil {
		return err
	}
	if cluster.replay != nil {
		var expected OracleResult
		if ordinal < uint64(len(cluster.replay.Oracles)) {
			expected = cluster.replay.Oracles[ordinal]
		}
		if !equalOracleResult(expected, result) {
			cluster.scenarioEvidenceBytes -= bytesRequired
			return cluster.evidenceDivergenceLocked(ReplayDimensionEvidence, ordinal, expected.Identity, result.Identity)
		}
	}
	cluster.oracles = append(cluster.oracles, result)
	return nil
}

func (cluster *inProcessCluster) recordScenarioDecision(ctx context.Context, decision ScenarioDecision) error {
	_, err := cluster.commitScenarioDecision(ctx, decision, false)
	return err
}

func (cluster *inProcessCluster) chooseScenarioAlternative(ctx context.Context, decision ScenarioDecision) (uint64, error) {
	return cluster.commitScenarioDecision(ctx, decision, true)
}

func (cluster *inProcessCluster) commitScenarioDecision(ctx context.Context, decision ScenarioDecision, choose bool) (uint64, error) {
	if err := cluster.beginCall(ctx); err != nil {
		return 0, err
	}
	defer cluster.endCall()
	if decision.Ordinal != 0 || decision.Occurrence != 0 || decision.Selected != 0 || decision.Identity != "" {
		return 0, errors.New("scenario decision contains record-owned fields")
	}
	decision.Alternatives = append([]string(nil), decision.Alternatives...)
	cluster.mu.Lock()
	defer cluster.mu.Unlock()
	decision.Ordinal = uint64(len(cluster.scenarios))
	cluster.scenarioOccurrences[decision.ID]++
	decision.Occurrence = cluster.scenarioOccurrences[decision.ID]
	if choose {
		decision.Selected = selectScenarioAlternative(cluster.seed, decision.Ordinal, decision.ID, uint64(len(decision.Alternatives)))
	}
	var err error
	if cluster.scenarioChoiceCursor < uint64(len(cluster.scenarioChoicePlan.Overrides)) {
		override := cluster.scenarioChoicePlan.Overrides[cluster.scenarioChoiceCursor]
		if override.Ordinal <= decision.Ordinal {
			expected := scenarioDecisionFromChoiceOverride(override)
			if override.Ordinal != decision.Ordinal || !scenarioChoiceOverrideMatchesDecision(override, decision) {
				return 0, cluster.scenarioDivergenceLocked(decision.Ordinal, expected, decision)
			}
			decision.Selected = override.Selected
			decision.Identity, err = scenarioDecisionIdentity(decision)
			if err != nil {
				return 0, err
			}
			if err := validateScenarioDecision(decision); err != nil {
				return 0, err
			}
			cluster.scenarioChoiceCursor++
		}
	}
	if choose && cluster.explorationPlan != nil {
		site, alternatives, identityErr := scenarioExplorationIdentities(decision.ID, decision.Occurrence, decision.Alternatives)
		if identityErr != nil {
			return 0, identityErr
		}
		selected, explorationErr := cluster.commitExplorationDecisionLocked(ExplorationScenario, decision.Ordinal, site, alternatives, uint32(decision.Selected))
		if explorationErr != nil {
			return 0, explorationErr
		}
		decision.Selected = uint64(selected)
	}
	identity, err := scenarioDecisionIdentity(decision)
	if err != nil {
		return 0, err
	}
	decision.Identity = identity
	if err := validateScenarioDecision(decision); err != nil {
		return 0, err
	}
	if err := checkCapacity("scenario_decisions", decision.Ordinal+1, cluster.limits.ScenarioDecisions); err != nil {
		return 0, err
	}
	if cluster.replay != nil {
		var expected ScenarioDecision
		if decision.Ordinal < uint64(len(cluster.replay.Scenarios)) {
			expected = cluster.replay.Scenarios[decision.Ordinal]
		}
		if !equalScenarioDecision(expected, decision) {
			return 0, cluster.scenarioDivergenceLocked(decision.Ordinal, expected, decision)
		}
	}
	cluster.scenarios = append(cluster.scenarios, decision)
	return decision.Selected, nil
}

func (cluster *inProcessCluster) finishControllers() error {
	cluster.mu.Lock()
	defer cluster.mu.Unlock()
	if uint64(len(cluster.faults)) != uint64(len(cluster.faultPlan.Actions)) {
		ordinal := uint64(len(cluster.faults))
		planned, ok := cluster.plannedFaultLocked(ordinal)
		if cluster.replay != nil {
			return cluster.faultMismatchLocked(ordinal, optionalFaultAction(planned, ok), nil, ErrReplayDiverged)
		}
		return &FaultPlanError{Ordinal: ordinal, Expected: optionalFaultAction(planned, ok), Cause: ErrFaultPlanMismatch}
	}
	if cluster.scenarioChoiceCursor != uint64(len(cluster.scenarioChoicePlan.Overrides)) {
		override := cluster.scenarioChoicePlan.Overrides[cluster.scenarioChoiceCursor]
		return cluster.scenarioDivergenceLocked(override.Ordinal, scenarioDecisionFromChoiceOverride(override), ScenarioDecision{})
	}
	if err := cluster.finishExplorationLocked(); err != nil {
		return err
	}
	if cluster.replay == nil {
		return nil
	}
	if len(cluster.faults) != len(cluster.replay.Faults) {
		ordinal := uint64(len(cluster.faults))
		return cluster.faultMismatchLocked(ordinal, cluster.expectedFaultActionLocked(ordinal), nil, ErrReplayDiverged)
	}
	if len(cluster.scenarios) != len(cluster.replay.Scenarios) {
		ordinal := uint64(len(cluster.scenarios))
		var expected ScenarioDecision
		if ordinal < uint64(len(cluster.replay.Scenarios)) {
			expected = cluster.replay.Scenarios[ordinal]
		}
		return cluster.scenarioDivergenceLocked(ordinal, expected, ScenarioDecision{})
	}
	if len(cluster.history) != len(cluster.replay.History) || len(cluster.observations) != len(cluster.replay.Observations) || len(cluster.oracles) != len(cluster.replay.Oracles) {
		expected, _ := hashCanonical("gomad3-controller-evidence/v1", struct {
			History      []HistoryOperation `json:"history"`
			Observations []Observation      `json:"observations"`
			Oracles      []OracleResult     `json:"oracles"`
		}{cluster.replay.History, cluster.replay.Observations, cluster.replay.Oracles})
		actual, _ := hashCanonical("gomad3-controller-evidence/v1", struct {
			History      []HistoryOperation `json:"history"`
			Observations []Observation      `json:"observations"`
			Oracles      []OracleResult     `json:"oracles"`
		}{cluster.history, cluster.observations, cluster.oracles})
		return cluster.evidenceDivergenceLocked(ReplayDimensionEvidence, uint64(len(cluster.scenarios)), expected, actual)
	}
	return nil
}

func (cluster *inProcessCluster) commitExplorationDecisionLocked(dimension ExplorationDimension, ordinal uint64, site string, alternatives []string, selected uint32) (uint32, error) {
	decision, err := newExplorationDecision(dimension, ordinal, site, alternatives, selected)
	if err != nil {
		return 0, err
	}
	override, forced := cluster.nextExplorationOverrideLocked(dimension)
	if forced && override.Ordinal < ordinal {
		return 0, cluster.explorationDivergenceLocked(override, decision)
	}
	if forced && override.Ordinal == ordinal {
		if override.SiteSHA256 != decision.SiteSHA256 || override.Alternatives != uint32(len(decision.Alternatives)) || override.AlternativeSetSHA256 != decision.AlternativeSetSHA256 || override.SelectedSHA256 != decision.Alternatives[override.Selected] {
			return 0, cluster.explorationDivergenceLocked(override, decision)
		}
		decision.Selected = override.Selected
		decision.Identity, err = explorationDecisionIdentity(decision)
		if err != nil {
			return 0, err
		}
		cluster.explorationConsumed[dimension]++
	}
	if cluster.replay != nil {
		expected, ok := findExplorationDecision(cluster.replay.ExplorationDecisions, dimension, ordinal)
		if !ok || !equalExplorationDecision(expected, decision) {
			return 0, cluster.explorationDecisionDivergenceLocked(expected, decision)
		}
	}
	cluster.explorationDecisions = append(cluster.explorationDecisions, decision)
	return decision.Selected, nil
}

func (cluster *inProcessCluster) finishExplorationLocked() error {
	if cluster.explorationPlan != nil {
		for _, dimension := range []ExplorationDimension{ExplorationScenario, ExplorationNetwork, ExplorationStorage, ExplorationFault, ExplorationCrash} {
			if override, ok := cluster.nextExplorationOverrideLocked(dimension); ok {
				return cluster.explorationDivergenceLocked(override, ExplorationDecision{})
			}
		}
	}
	if cluster.replay != nil && len(cluster.explorationDecisions) != len(cluster.replay.ExplorationDecisions) {
		var expected ExplorationDecision
		if len(cluster.explorationDecisions) < len(cluster.replay.ExplorationDecisions) {
			expected = cluster.replay.ExplorationDecisions[len(cluster.explorationDecisions)]
		}
		return cluster.explorationDecisionDivergenceLocked(expected, ExplorationDecision{})
	}
	return nil
}

func (cluster *inProcessCluster) nextExplorationOverrideLocked(dimension ExplorationDimension) (ExplorationOverride, bool) {
	remaining := cluster.explorationConsumed[dimension]
	for _, override := range cluster.explorationPlan.Overrides {
		if override.Dimension != dimension {
			continue
		}
		if remaining == 0 {
			return override, true
		}
		remaining--
	}
	return ExplorationOverride{}, false
}

func (cluster *inProcessCluster) explorationDivergenceLocked(expected ExplorationOverride, actual ExplorationDecision) error {
	actualIdentity := actual.Identity
	if !validSHA256(actualIdentity) {
		actualIdentity, _ = hashCanonical("gomad3-missing-exploration-decision/v1", struct {
			Dimension ExplorationDimension `json:"dimension"`
			Ordinal   uint64               `json:"ordinal"`
		}{expected.Dimension, expected.Ordinal})
	}
	divergence := ReplayDivergence{
		Dimension: ReplayDimensionExploration, Ordinal: expected.Ordinal,
		ExpectedSHA256: expected.Identity, ActualSHA256: actualIdentity,
		ExpectedExplorationOverride: cloneExplorationOverridePointer(&expected),
		ActualExploration:           cloneExplorationDecisionPointerIfValid(actual),
	}
	err := &ReplayDivergenceError{Divergence: divergence}
	cluster.replayFailure = err
	return err
}

func (cluster *inProcessCluster) explorationDecisionDivergenceLocked(expected, actual ExplorationDecision) error {
	dimension, ordinal := expected.Dimension, expected.Ordinal
	if explorationDimensionOrder(dimension) < 0 {
		dimension, ordinal = actual.Dimension, actual.Ordinal
	}
	expectedIdentity := explorationDecisionOrMissingIdentity(expected, dimension, ordinal)
	actualIdentity := explorationDecisionOrMissingIdentity(actual, dimension, ordinal)
	divergence := ReplayDivergence{
		Dimension: ReplayDimensionExploration, Ordinal: ordinal,
		ExpectedSHA256: expectedIdentity, ActualSHA256: actualIdentity,
		ExpectedExploration: cloneExplorationDecisionPointerIfValid(expected),
		ActualExploration:   cloneExplorationDecisionPointerIfValid(actual),
	}
	err := &ReplayDivergenceError{Divergence: divergence}
	cluster.replayFailure = err
	return err
}

func explorationDecisionOrMissingIdentity(decision ExplorationDecision, dimension ExplorationDimension, ordinal uint64) string {
	if validSHA256(decision.Identity) {
		return decision.Identity
	}
	identity, _ := hashCanonical("gomad3-missing-exploration-decision/v1", struct {
		Dimension ExplorationDimension `json:"dimension"`
		Ordinal   uint64               `json:"ordinal"`
	}{dimension, ordinal})
	return identity
}

func findExplorationDecision(decisions []ExplorationDecision, dimension ExplorationDimension, ordinal uint64) (ExplorationDecision, bool) {
	for _, decision := range decisions {
		if decision.Dimension == dimension && decision.Ordinal == ordinal {
			return decision, true
		}
	}
	return ExplorationDecision{}, false
}

func equalExplorationDecision(left, right ExplorationDecision) bool {
	return left.Dimension == right.Dimension && left.Ordinal == right.Ordinal && left.SiteSHA256 == right.SiteSHA256 && slices.Equal(left.Alternatives, right.Alternatives) && left.AlternativeSetSHA256 == right.AlternativeSetSHA256 && left.Selected == right.Selected && left.Identity == right.Identity
}

func (cluster *inProcessCluster) plannedFaultLocked(ordinal uint64) (FaultAction, bool) {
	if ordinal >= uint64(len(cluster.faultPlan.Actions)) {
		return FaultAction{}, false
	}
	return cluster.faultPlan.Actions[ordinal], true
}

func (cluster *inProcessCluster) expectedFaultLocked(ordinal uint64) (FaultRealization, bool) {
	if cluster.replay == nil || ordinal >= uint64(len(cluster.replay.Faults)) {
		return FaultRealization{}, false
	}
	return cluster.replay.Faults[ordinal], true
}

func (cluster *inProcessCluster) expectedFaultActionLocked(ordinal uint64) *FaultAction {
	expected, ok := cluster.expectedFaultLocked(ordinal)
	return optionalFaultAction(expected.Action, ok)
}

func (cluster *inProcessCluster) faultMismatchLocked(ordinal uint64, expected, actual *FaultAction, cause error) error {
	if cluster.replay == nil && !errors.Is(cause, ErrReplayDiverged) {
		return &FaultPlanError{Ordinal: ordinal, Expected: cloneFaultActionPointer(expected), Actual: cloneFaultActionPointer(actual), Cause: cause}
	}
	divergence := faultDivergence(ordinal, expected, actual)
	err := &ReplayDivergenceError{Divergence: divergence}
	cluster.replayFailure = err
	return err
}

func (cluster *inProcessCluster) faultRealizationMismatchLocked(ordinal uint64, expected, actual *FaultRealization) error {
	expectedIdentity, actualIdentity := "", ""
	var expectedAction, actualAction *FaultAction
	if expected != nil {
		expectedIdentity = expected.Identity
		if !validSHA256(expectedIdentity) {
			expectedIdentity, _ = faultRealizationIdentity(*expected)
		}
		expectedAction = faultActionPointer(expected.Action)
	}
	if actual != nil {
		actualIdentity = actual.Identity
		if !validSHA256(actualIdentity) {
			actualIdentity, _ = faultRealizationIdentity(*actual)
		}
		actualAction = faultActionPointer(actual.Action)
	}
	if !validSHA256(expectedIdentity) {
		expectedIdentity, _ = hashCanonical("gomad3-missing-fault-realization/v1", expectedIdentity)
	}
	if !validSHA256(actualIdentity) {
		actualIdentity, _ = hashCanonical("gomad3-missing-fault-realization/v1", actualIdentity)
	}
	err := &ReplayDivergenceError{Divergence: ReplayDivergence{
		Dimension: ReplayDimensionFault, Ordinal: ordinal,
		ExpectedSHA256: expectedIdentity, ActualSHA256: actualIdentity,
		ExpectedFault: expectedAction, ActualFault: actualAction,
	}}
	cluster.replayFailure = err
	return err
}

func (cluster *inProcessCluster) scenarioDivergenceLocked(ordinal uint64, expected, actual ScenarioDecision) error {
	expectedIdentity := expected.Identity
	if expectedIdentity == "" {
		expectedIdentity, _ = scenarioDecisionIdentity(expected)
	}
	actualIdentity := actual.Identity
	if actualIdentity == "" {
		actualIdentity, _ = scenarioDecisionIdentity(actual)
	}
	divergence := ReplayDivergence{Dimension: ReplayDimensionScenario, Ordinal: ordinal, ExpectedSHA256: expectedIdentity, ActualSHA256: actualIdentity}
	if expected.ID != "" {
		value := expected
		divergence.ExpectedScenario = &value
	}
	if actual.ID != "" {
		value := actual
		divergence.ActualScenario = &value
	}
	err := &ReplayDivergenceError{Divergence: divergence}
	cluster.replayFailure = err
	return err
}

func (cluster *inProcessCluster) evidenceDivergenceLocked(dimension ReplayDimension, ordinal uint64, expected, actual string) error {
	if !validSHA256(expected) {
		expected, _ = hashCanonical("gomad3-missing-evidence/v1", expected)
	}
	if !validSHA256(actual) {
		actual, _ = hashCanonical("gomad3-missing-evidence/v1", actual)
	}
	err := &ReplayDivergenceError{Divergence: ReplayDivergence{Dimension: dimension, Ordinal: ordinal, ExpectedSHA256: expected, ActualSHA256: actual}}
	cluster.replayFailure = err
	return err
}

func (cluster *inProcessCluster) reserveScenarioEvidenceLocked(required uint64) error {
	next := saturatingAdd(cluster.scenarioEvidenceBytes, required)
	if err := checkCapacity("scenario_evidence_bytes", next, cluster.limits.ScenarioEvidenceBytes); err != nil {
		return err
	}
	cluster.scenarioEvidenceBytes = next
	return nil
}

func selectFaultTarget(seed, ordinal uint64, id FaultID, alternatives uint64) uint64 {
	input := make([]byte, 0, len(id)+40)
	input = append(input, "gomad3-fault-target/v1"...)
	input = append(input, 0)
	var encoded [8]byte
	binary.LittleEndian.PutUint64(encoded[:], seed)
	input = append(input, encoded[:]...)
	binary.LittleEndian.PutUint64(encoded[:], ordinal)
	input = append(input, encoded[:]...)
	input = append(input, id...)
	digest := sha256.Sum256(input)
	return binary.LittleEndian.Uint64(digest[:8]) % alternatives
}

func faultEventOccurrenceIdentity(event FaultMatch) (string, error) {
	event.Occurrence = 0
	return hashCanonical("gomad3-fault-event-occurrence/v1", event)
}

func faultRealizationIdentity(realization FaultRealization) (string, error) {
	realization.Identity = ""
	realization.Action = cloneFaultAction(realization.Action)
	return hashCanonical("gomad3-fault-realization/v1", realization)
}

func faultDivergence(ordinal uint64, expected, actual *FaultAction) ReplayDivergence {
	expectedValue := FaultAction{}
	actualValue := FaultAction{}
	if expected != nil {
		expectedValue = cloneFaultAction(*expected)
	}
	if actual != nil {
		actualValue = cloneFaultAction(*actual)
	}
	expectedIdentity, _ := hashCanonical("gomad3-fault-action/v1", expectedValue)
	actualIdentity, _ := hashCanonical("gomad3-fault-action/v1", actualValue)
	return ReplayDivergence{
		Dimension: ReplayDimensionFault, Ordinal: ordinal,
		ExpectedSHA256: expectedIdentity, ActualSHA256: actualIdentity,
		ExpectedFault: cloneFaultActionPointer(expected), ActualFault: cloneFaultActionPointer(actual),
	}
}

func observationIdentity(observation Observation) (string, error) {
	observation.Identity = ""
	observation.Value = append([]byte(nil), observation.Value...)
	return hashCanonical("gomad3-scenario-observation/v1", observation)
}

func validateObservation(observation Observation) error {
	if err := validateID("observation ID", observation.ID); err != nil {
		return err
	}
	if err := validateID("observation kind", observation.Kind); err != nil {
		return err
	}
	if observation.Handle.Node == "" != (observation.Handle.Incarnation == 0) || !validSHA256(observation.FullSHA256) || !validSHA256(observation.Identity) {
		return errors.New("scenario observation metadata is invalid")
	}
	digest := sha256.Sum256(observation.Value)
	if observation.FullSHA256 != fmt.Sprintf("sha256:%x", digest) {
		return errors.New("scenario observation hash does not match its value")
	}
	want, err := observationIdentity(observation)
	if err != nil {
		return err
	}
	if observation.Identity != want {
		return errors.New("scenario observation identity does not match its contents")
	}
	return nil
}

func validateFaultRealization(realization FaultRealization) error {
	if err := validateFaultAction(realization.Action); err != nil {
		return err
	}
	if err := validateFaultMatch(realization.Matched); err != nil {
		return err
	}
	if faultMatchEmpty(realization.Action.Match) {
		if !faultMatchEmpty(realization.Matched) {
			return errors.New("unmatched fault realization contains matched event fields")
		}
	} else if realization.Matched.Occurrence == 0 || !faultMatches(realization.Action.Match, realization.Matched) {
		return errors.New("fault realization does not match its planned event")
	}
	switch realization.Action.Kind {
	case FaultGracefulStop, FaultHarshCrash, FaultRestart:
		if realization.Target.Node == "" || realization.Target.Incarnation == 0 {
			return errors.New("lifecycle fault realization has no target")
		}
	default:
		if realization.Target != (NodeHandle{}) {
			return errors.New("topology fault realization contains a lifecycle target")
		}
	}
	want, err := faultRealizationIdentity(realization)
	if err != nil {
		return err
	}
	if !validSHA256(realization.Identity) || realization.Identity != want {
		return errors.New("fault realization identity does not match its contents")
	}
	return nil
}

func equalFaultAction(left, right FaultAction) bool {
	return left.ID == right.ID && left.Kind == right.Kind && left.Match == right.Match && left.Node == right.Node && slices.Equal(left.Candidates, right.Candidates) && left.TargetFrom == right.TargetFrom && left.From == right.From && left.To == right.To && slices.Equal(left.Left, right.Left) && slices.Equal(left.Right, right.Right) && left.DelayNanos == right.DelayNanos && left.Persistence == right.Persistence
}

func equalFaultRealization(left, right FaultRealization) bool {
	return left.Ordinal == right.Ordinal && equalFaultAction(left.Action, right.Action) && left.Matched == right.Matched && left.Target == right.Target && left.Identity == right.Identity
}

func equalScenarioDecision(left, right ScenarioDecision) bool {
	return left.Ordinal == right.Ordinal && left.ID == right.ID && left.Kind == right.Kind && left.Occurrence == right.Occurrence && slices.Equal(left.Alternatives, right.Alternatives) && left.Selected == right.Selected && left.Identity == right.Identity
}

func equalObservation(left, right Observation) bool {
	return left.Ordinal == right.Ordinal && left.ID == right.ID && left.Kind == right.Kind && left.Handle == right.Handle && bytes.Equal(left.Value, right.Value) && left.FullSHA256 == right.FullSHA256 && left.Identity == right.Identity
}

func equalOracleResult(left, right OracleResult) bool {
	return left.Name == right.Name && left.Passed == right.Passed && left.FailureIdentity == right.FailureIdentity && left.Identity == right.Identity && equalOracleEvidence(left.Evidence, right.Evidence)
}

func historyOperationByID(operations []HistoryOperation, id string) (HistoryOperation, bool) {
	for _, operation := range operations {
		if operation.ID == id {
			return operation, true
		}
	}
	return HistoryOperation{}, false
}

func historyOperationBytes(operation HistoryOperation) uint64 {
	return saturatingAdd(saturatingAdd(uint64(len(operation.Input)), uint64(len(operation.Output))), uint64(len(operation.Error)))
}

func oracleEvidenceBytes(result OracleResult) uint64 {
	var total uint64
	for _, evidence := range result.Evidence {
		total = saturatingAdd(total, uint64(len(evidence.Value)))
	}
	return total
}

func cloneFaultRealization(realization FaultRealization) FaultRealization {
	realization.Action = cloneFaultAction(realization.Action)
	return realization
}

func cloneFaultRealizations(realizations []FaultRealization) []FaultRealization {
	cloned := make([]FaultRealization, len(realizations))
	for index, realization := range realizations {
		cloned[index] = cloneFaultRealization(realization)
	}
	return cloned
}

func cloneObservations(observations []Observation) []Observation {
	cloned := make([]Observation, len(observations))
	for index, observation := range observations {
		cloned[index] = observation
		cloned[index].Value = append([]byte(nil), observation.Value...)
	}
	return cloned
}

func cloneOracleResult(result OracleResult) OracleResult {
	evidenceValues := result.Evidence
	result.Evidence = make([]OracleEvidence, len(result.Evidence))
	for index, evidence := range evidenceValues {
		result.Evidence[index] = evidence
		result.Evidence[index].Value = append([]byte(nil), evidence.Value...)
	}
	return result
}

func cloneOracleResults(results []OracleResult) []OracleResult {
	cloned := make([]OracleResult, len(results))
	for index, result := range results {
		cloned[index] = cloneOracleResult(result)
	}
	return cloned
}

func optionalFaultAction(action FaultAction, ok bool) *FaultAction {
	if !ok {
		return nil
	}
	return faultActionPointer(action)
}

func optionalFaultRealization(realization FaultRealization, ok bool) *FaultRealization {
	if !ok {
		return nil
	}
	cloned := cloneFaultRealization(realization)
	return &cloned
}

func faultActionPointer(action FaultAction) *FaultAction {
	cloned := cloneFaultAction(action)
	return &cloned
}

func cloneFaultActionPointer(action *FaultAction) *FaultAction {
	if action == nil {
		return nil
	}
	return faultActionPointer(*action)
}

func firstFailedOracle(results []OracleResult) *OracleResult {
	for index := range results {
		if !results[index].Passed {
			return &results[index]
		}
	}
	return nil
}

func normalizedFailureIdentity(outcome Outcome, reason, sourceIdentity string) (string, error) {
	if outcome == OutcomeCompleted {
		return "", nil
	}
	return hashCanonical("gomad3-cluster-failure/v1", struct {
		Outcome        Outcome `json:"outcome"`
		Reason         string  `json:"reason"`
		SourceIdentity string  `json:"source_identity,omitempty"`
	}{Outcome: outcome, Reason: reason, SourceIdentity: sourceIdentity})
}

func divergenceIdentity(divergence ReplayDivergence) string {
	identity, _ := hashCanonical("gomad3-replay-divergence/v1", divergence)
	return identity
}

func divergenceIdentityValue(divergence *ReplayDivergence) string {
	if divergence == nil {
		return ""
	}
	return divergenceIdentity(*divergence)
}
