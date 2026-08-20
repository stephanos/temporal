package gomadv3sim

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime"
	"slices"
	"unicode/utf8"
)

func EncodeClusterRecord(record ClusterRecord) ([]byte, error) {
	if err := validateClusterRecord(record); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("encode cluster record: %w", err)
	}
	if len(encoded) > MaximumClusterRecordBytes {
		return nil, fmt.Errorf("cluster record exceeds %d bytes", MaximumClusterRecordBytes)
	}
	return encoded, nil
}

func DecodeClusterRecord(data []byte) (ClusterRecord, error) {
	if len(data) == 0 || len(data) > MaximumClusterRecordBytes {
		return ClusterRecord{}, fmt.Errorf("cluster record must be between 1 and %d bytes", MaximumClusterRecordBytes)
	}
	if !utf8.Valid(data) {
		return ClusterRecord{}, errors.New("cluster record is not valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var record ClusterRecord
	if err := decoder.Decode(&record); err != nil {
		return ClusterRecord{}, fmt.Errorf("decode cluster record: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ClusterRecord{}, errors.New("cluster record contains trailing JSON")
	}
	if err := validateClusterRecord(record); err != nil {
		return ClusterRecord{}, err
	}
	canonical, err := json.Marshal(record)
	if err != nil {
		return ClusterRecord{}, fmt.Errorf("canonicalize cluster record: %w", err)
	}
	if !bytes.Equal(data, canonical) {
		return ClusterRecord{}, errors.New("cluster record is not canonical JSON")
	}
	return record, nil
}

func ReplayPlanFor(record ClusterRecord) (ReplayPlan, error) {
	if err := validateClusterRecord(record); err != nil {
		return ReplayPlan{}, err
	}
	plan := ReplayPlan{
		Schema:          ClusterReplaySchema,
		SpecSHA256:      record.SpecSHA256,
		Static:          record.Static,
		Models:          record.Models,
		Outcome:         record.Outcome,
		Reason:          record.Reason,
		FailureIdentity: record.FailureIdentity,
		Nodes:           append([]NodeResult(nil), record.Nodes...),
		Transitions:     append([]LifecycleTransition(nil), record.Transitions...),
		FaultPlan:       cloneFaultPlan(record.FaultPlan),
		Faults:          cloneFaultRealizations(record.Faults),
		ScenarioChoices: cloneScenarioChoicePlan(record.ScenarioChoices),
		Scenarios:       cloneScenarioDecisions(record.Scenarios),
		History:         cloneHistoryOperations(record.History),
		Observations:    cloneObservations(record.Observations),
		Oracles:         cloneOracleResults(record.Oracles),
		Network:         cloneNetworkRecord(record.Network),
		Volumes:         cloneVolumeRecord(record.Volumes),
		Outputs:         cloneOutputs(record.Outputs),
		Leaks:           append([]LeakDiagnostic(nil), record.Leaks...),
	}
	identity, err := replayPlanIdentity(plan)
	if err != nil {
		return ReplayPlan{}, err
	}
	plan.Identity = identity
	return plan, nil
}

func buildClusterRecord(spec Spec, specSHA256 string, result Result) (ClusterRecord, error) {
	models, err := currentClusterModelIdentities()
	if err != nil {
		return ClusterRecord{}, err
	}
	static, err := clusterStaticIdentities(spec, models)
	if err != nil {
		return ClusterRecord{}, err
	}
	faultPlan := FaultPlan{}
	if spec.Faults == nil {
		faultPlan, err = NewFaultPlan(nil)
		if err != nil {
			return ClusterRecord{}, err
		}
	} else {
		faultPlan = cloneFaultPlan(*spec.Faults)
	}
	scenarioChoices, err := scenarioChoicePlanForSpec(spec)
	if err != nil {
		return ClusterRecord{}, err
	}
	network := cloneNetworkRecord(result.Network)
	if network.Snapshot.TransitionSHA256 == "" {
		network = emptyNetworkRecord()
	}
	volumes := cloneVolumeRecord(result.Volumes)
	if volumes.Snapshot.TransitionSHA256 == "" {
		volumes = emptyVolumeRecord()
	}
	failureIdentity := result.FailureIdentity
	if result.Outcome != OutcomeCompleted && failureIdentity == "" {
		failureIdentity, err = normalizedFailureIdentity(result.Outcome, result.Reason, divergenceIdentityValue(result.Divergence))
		if err != nil {
			return ClusterRecord{}, err
		}
	}
	record := ClusterRecord{
		Schema:          ClusterRecordSchema,
		Backend:         spec.Backend,
		Fidelity:        spec.Fidelity,
		Seed:            spec.Seed,
		Limits:          spec.Limits,
		Static:          static,
		Models:          models,
		NodeSpecs:       cloneNodeSpecs(spec.Nodes),
		LinkSpecs:       append([]LinkSpec(nil), spec.Links...),
		VolumeSpecs:     append([]VolumeSpec(nil), spec.Volumes...),
		SpecSHA256:      specSHA256,
		Outcome:         result.Outcome,
		Reason:          result.Reason,
		FailureIdentity: failureIdentity,
		Nodes:           append([]NodeResult(nil), result.Record.Nodes...),
		Transitions:     append([]LifecycleTransition(nil), result.Record.Transitions...),
		FaultPlan:       faultPlan,
		Faults:          cloneFaultRealizations(result.Faults),
		ScenarioChoices: scenarioChoices,
		Scenarios:       cloneScenarioDecisions(result.Scenarios),
		History:         cloneHistoryOperations(result.History),
		Observations:    cloneObservations(result.Observations),
		Oracles:         cloneOracleResults(result.Oracles),
		Network:         network,
		Volumes:         volumes,
		Outputs:         cloneOutputs(result.Outputs),
		Limitations:     backendLimitations(spec.Backend),
		Leaks:           append([]LeakDiagnostic(nil), result.Leaks...),
		Divergence:      cloneReplayDivergence(result.Divergence),
	}
	identity, err := clusterRecordIdentity(record)
	if err != nil {
		return ClusterRecord{}, err
	}
	record.Identity = identity
	return record, nil
}

func emptyNetworkRecord() NetworkRecord {
	encoded, err := encodeRuntimeNetworkTransitions(nil)
	if err != nil {
		return NetworkRecord{}
	}
	digest := sha256.Sum256(append([]byte("gomadv3-simulation-network-transitions/v2\x00"), encoded...))
	snapshot := NetworkSnapshot{TransitionSHA256: fmt.Sprintf("sha256:%x", digest)}
	identity, err := networkSnapshotIdentity(snapshot)
	if err != nil {
		return NetworkRecord{}
	}
	snapshot.Identity = identity
	return NetworkRecord{Snapshot: snapshot}
}

func emptyVolumeRecord() VolumeRecord {
	snapshot := VolumeSnapshot{TransitionSHA256: volumeTransitionsIdentity(nil)}
	snapshot.Identity = volumeRunSnapshotIdentity(snapshot)
	return VolumeRecord{Snapshot: snapshot}
}

func validateClusterRecord(record ClusterRecord) error {
	if record.Schema != ClusterRecordSchema {
		return fmt.Errorf("cluster record schema = %q, want %q", record.Schema, ClusterRecordSchema)
	}
	if !validSHA256(record.SpecSHA256) || !validSHA256(record.Identity) {
		return errors.New("cluster record contains an invalid SHA-256 identity")
	}
	if err := validateLimits(record.Limits); err != nil {
		return err
	}
	if record.Backend != BackendInProcess && record.Backend != BackendProcess || record.Backend == BackendInProcess && record.Fidelity != FidelitySimulationModel || record.Backend == BackendProcess && record.Fidelity != FidelitySimulationModel && record.Fidelity != FidelityHardIsolation {
		return errors.New("cluster record contains an unsupported backend or fidelity")
	}
	if err := validateClusterStaticEvidence(record); err != nil {
		return err
	}
	if !slices.Equal(record.Limitations, backendLimitations(record.Backend)) {
		return errors.New("cluster record limitations do not match its backend")
	}
	if err := validateOutcome(record.Outcome, record.Reason, record.FailureIdentity, record.Divergence, record.Oracles); err != nil {
		return err
	}
	if err := validateNodeResults(record.Nodes, record.Limits.ScenarioActions); err != nil {
		return err
	}
	if err := validateTransitions(record.Transitions, record.Limits.ScenarioActions); err != nil {
		return err
	}
	if err := validateControllerEvidence(record.FaultPlan, record.Faults, record.Scenarios, record.History, record.Observations, record.Oracles, record.Limits); err != nil {
		return err
	}
	if err := validateScenarioChoicePlan(record.ScenarioChoices); err != nil {
		return err
	}
	if err := checkCapacity("scenario_choice_overrides", uint64(len(record.ScenarioChoices.Overrides)), record.Limits.ScenarioDecisions); err != nil {
		return err
	}
	if err := validateNetworkRecord(record.Network, record.Limits); err != nil {
		return err
	}
	if err := validateVolumeRecord(record.Volumes, record.Limits); err != nil {
		return err
	}
	if err := validateOutputs(record.Outputs, record.Limits.ObservationBytes); err != nil {
		return err
	}
	if err := validateBackendLeaks(record.Backend, record.Leaks); err != nil {
		return err
	}
	want, err := clusterRecordIdentity(record)
	if err != nil {
		return err
	}
	if record.Identity != want {
		return errors.New("cluster record identity does not match its contents")
	}
	return nil
}

func validateClusterStaticEvidence(record ClusterRecord) error {
	models, err := currentClusterModelIdentities()
	if err != nil {
		return err
	}
	if record.Models != models {
		return errors.New("cluster record model identities do not match this implementation")
	}
	volumes, err := validateVolumes(record.VolumeSpecs, record.Limits.VolumeCapacityBytes)
	if err != nil {
		return err
	}
	nodes, err := validateNodes(record.NodeSpecs, volumes, record.Limits.BootConfigBytes)
	if err != nil {
		return err
	}
	if err := validateLinks(record.LinkSpecs, nodes); err != nil {
		return err
	}
	faultPlan := cloneFaultPlan(record.FaultPlan)
	scenarioChoices := cloneScenarioChoicePlan(record.ScenarioChoices)
	spec := Spec{
		Schema: SpecSchema, Backend: record.Backend, Fidelity: record.Fidelity, Seed: record.Seed, Limits: record.Limits,
		Nodes: cloneNodeSpecs(record.NodeSpecs), Links: append([]LinkSpec(nil), record.LinkSpecs...), Volumes: append([]VolumeSpec(nil), record.VolumeSpecs...),
		Faults: &faultPlan, ScenarioChoices: &scenarioChoices,
	}
	wantSpec, err := hashSpec(spec)
	if err != nil {
		return err
	}
	if record.SpecSHA256 != wantSpec {
		return errors.New("cluster record specification identity does not match its contents")
	}
	static, err := clusterStaticIdentities(spec, models)
	if err != nil {
		return err
	}
	if record.Static != static {
		return errors.New("cluster record static identities do not match its specification")
	}
	return nil
}

func validateControllerEvidence(plan FaultPlan, faults []FaultRealization, scenarios []ScenarioDecision, history []HistoryOperation, observations []Observation, oracles []OracleResult, limits Limits) error {
	if err := validateFaultPlan(plan); err != nil {
		return err
	}
	if err := checkCapacity("fault_actions", uint64(len(plan.Actions)), limits.FaultActions); err != nil {
		return err
	}
	if err := checkCapacity("fault_actions", uint64(len(faults)), limits.FaultActions); err != nil {
		return err
	}
	if len(faults) > len(plan.Actions) {
		return errors.New("cluster fault tape exceeds its planned actions")
	}
	for index, fault := range faults {
		if fault.Ordinal != uint64(index) || !equalFaultAction(fault.Action, plan.Actions[index]) {
			return errors.New("cluster fault realization is unordered or does not match its plan")
		}
		if err := validateFaultRealization(fault); err != nil {
			return err
		}
	}
	if err := checkCapacity("scenario_decisions", uint64(len(scenarios)), limits.ScenarioDecisions); err != nil {
		return err
	}
	for index, decision := range scenarios {
		if decision.Ordinal != uint64(index) {
			return errors.New("scenario decisions are unordered")
		}
		if err := validateScenarioDecision(decision); err != nil {
			return err
		}
	}
	if err := checkCapacity("history_operations", uint64(len(history)), limits.HistoryOperations); err != nil {
		return err
	}
	if err := ValidateHistory(history, limits.ScenarioEvidenceBytes); err != nil {
		return err
	}
	if err := checkCapacity("observations", uint64(len(observations)), limits.Observations); err != nil {
		return err
	}
	var evidenceBytes uint64
	seenObservations := make(map[string]struct{}, len(observations))
	for index, observation := range observations {
		if observation.Ordinal != uint64(index) {
			return errors.New("scenario observations are unordered")
		}
		if _, ok := seenObservations[observation.ID]; ok {
			return errors.New("scenario observation ID is duplicated")
		}
		seenObservations[observation.ID] = struct{}{}
		if err := validateObservation(observation); err != nil {
			return err
		}
		evidenceBytes = saturatingAdd(evidenceBytes, uint64(len(observation.Value)))
	}
	for _, operation := range history {
		evidenceBytes = saturatingAdd(evidenceBytes, historyOperationBytes(operation))
	}
	if err := checkCapacity("oracle_results", uint64(len(oracles)), limits.OracleResults); err != nil {
		return err
	}
	seenOracles := make(map[string]struct{}, len(oracles))
	for _, oracle := range oracles {
		if _, ok := seenOracles[oracle.Name]; ok {
			return errors.New("oracle result name is duplicated")
		}
		seenOracles[oracle.Name] = struct{}{}
		if err := validateOracleResult(oracle, limits.ScenarioEvidenceBytes); err != nil {
			return err
		}
		evidenceBytes = saturatingAdd(evidenceBytes, oracleEvidenceBytes(oracle))
	}
	return checkCapacity("scenario_evidence_bytes", evidenceBytes, limits.ScenarioEvidenceBytes)
}

func currentClusterModelIdentities() (ClusterModelIdentities, error) {
	definitions := []string{
		"gomadv3-runtime-domain-model/v1",
		"gomadv3-process-broker/v2-activation-barrier-bounded-private-descriptors-hard-reap",
		"gomadv3-network-model/v4-directional-links-atomic-groups",
		"gomadv3-volume-model/v2-explicit-crash-selection",
		"gomadv3-fault-controller/v2-stable-match-occurrence",
		"gomadv3-scenario-controller/v2-rank-bound-choice-plan",
		"gomadv3-oracle-model/v1",
	}
	identities := make([]string, len(definitions))
	for index, definition := range definitions {
		identity, err := hashCanonical("gomadv3-cluster-model/v1", definition)
		if err != nil {
			return ClusterModelIdentities{}, err
		}
		identities[index] = identity
	}
	return ClusterModelIdentities{
		RuntimeDomainSHA256: identities[0], ProcessSHA256: identities[1], NetworkSHA256: identities[2], VolumeSHA256: identities[3],
		FaultSHA256: identities[4], ScenarioSHA256: identities[5], OracleSHA256: identities[6],
	}, nil
}

func clusterStaticIdentities(spec Spec, models ClusterModelIdentities) (ClusterStaticIdentities, error) {
	target, err := hashCanonical("gomadv3-cluster-target/v1", struct {
		Nodes   []NodeSpec   `json:"nodes"`
		Links   []LinkSpec   `json:"links"`
		Volumes []VolumeSpec `json:"volumes"`
	}{Nodes: cloneNodeSpecs(spec.Nodes), Links: append([]LinkSpec(nil), spec.Links...), Volumes: append([]VolumeSpec(nil), spec.Volumes...)})
	if err != nil {
		return ClusterStaticIdentities{}, err
	}
	platform, err := hashCanonical("gomadv3-cluster-platform-bundle/v1", struct {
		GOOS     string                 `json:"goos"`
		GOARCH   string                 `json:"goarch"`
		Backend  Backend                `json:"backend"`
		Fidelity Fidelity               `json:"fidelity"`
		Models   ClusterModelIdentities `json:"models"`
	}{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, Backend: spec.Backend, Fidelity: spec.Fidelity, Models: models})
	if err != nil {
		return ClusterStaticIdentities{}, err
	}
	return ClusterStaticIdentities{TargetSHA256: target, PlatformBundleSHA256: platform}, nil
}

func validateReplayPlan(plan ReplayPlan, spec Spec) error {
	if plan.Schema != ClusterReplaySchema {
		return fmt.Errorf("cluster replay schema = %q, want %q", plan.Schema, ClusterReplaySchema)
	}
	if !validSHA256(plan.SpecSHA256) || !validSHA256(plan.Identity) {
		return errors.New("cluster replay contains an invalid SHA-256 identity")
	}
	models, err := currentClusterModelIdentities()
	if err != nil {
		return err
	}
	static, err := clusterStaticIdentities(spec, models)
	if err != nil {
		return err
	}
	if plan.Models != models || plan.Static != static {
		return errors.New("cluster replay static or model identities do not match the specification")
	}
	if err := validateReplayOutcome(plan.Outcome, plan.Reason, plan.FailureIdentity, plan.Oracles); err != nil {
		return err
	}
	if err := validateNodeResults(plan.Nodes, spec.Limits.ScenarioActions); err != nil {
		return err
	}
	if err := validateTransitions(plan.Transitions, spec.Limits.ScenarioActions); err != nil {
		return err
	}
	if err := validateControllerEvidence(plan.FaultPlan, plan.Faults, plan.Scenarios, plan.History, plan.Observations, plan.Oracles, spec.Limits); err != nil {
		return err
	}
	scenarioChoices, err := scenarioChoicePlanForSpec(spec)
	if err != nil {
		return err
	}
	if err := validateScenarioChoicePlan(plan.ScenarioChoices); err != nil {
		return err
	}
	if !equalScenarioChoicePlan(plan.ScenarioChoices, scenarioChoices) {
		return errors.New("cluster replay scenario choice plan does not match the specification")
	}
	if err := validateNetworkRecord(plan.Network, spec.Limits); err != nil {
		return err
	}
	if err := validateVolumeRecord(plan.Volumes, spec.Limits); err != nil {
		return err
	}
	if err := validateOutputs(plan.Outputs, spec.Limits.ObservationBytes); err != nil {
		return err
	}
	if err := validateBackendLeaks(spec.Backend, plan.Leaks); err != nil {
		return err
	}
	want, err := replayPlanIdentity(plan)
	if err != nil {
		return err
	}
	if plan.Identity != want {
		return errors.New("cluster replay identity does not match its contents")
	}
	return nil
}

func validateReplayOutcome(outcome Outcome, reason, failureIdentity string, oracles []OracleResult) error {
	if len(reason) > MaximumTerminalReasonBytes {
		return errors.New("cluster replay reason exceeds the terminal text limit")
	}
	switch outcome {
	case OutcomeCompleted:
		if reason != "" || failureIdentity != "" {
			return errors.New("completed cluster replay contains a reason")
		}
	case OutcomeScenarioFailed, OutcomeReplayDiverged:
		if reason == "" || !validSHA256(failureIdentity) {
			return errors.New("failed cluster replay has no reason")
		}
	case OutcomeOracleFailed:
		failed := firstFailedOracle(oracles)
		if reason == "" || failed == nil || failureIdentity != failed.FailureIdentity {
			return errors.New("oracle-failed cluster replay is inconsistent")
		}
	default:
		return fmt.Errorf("cluster replay outcome %q is invalid", outcome)
	}
	return nil
}

func validateOutcome(outcome Outcome, reason, failureIdentity string, divergence *ReplayDivergence, oracles []OracleResult) error {
	if len(reason) > MaximumTerminalReasonBytes {
		return errors.New("cluster outcome reason exceeds the terminal text limit")
	}
	switch outcome {
	case OutcomeCompleted:
		if reason != "" || failureIdentity != "" || divergence != nil {
			return errors.New("completed cluster outcome contains failure evidence")
		}
	case OutcomeScenarioFailed:
		if reason == "" || !validSHA256(failureIdentity) || divergence != nil {
			return errors.New("scenario failure outcome is inconsistent")
		}
		want, err := normalizedFailureIdentity(outcome, reason, "")
		if err != nil || failureIdentity != want {
			return errors.New("scenario failure identity does not match its outcome")
		}
	case OutcomeOracleFailed:
		failed := firstFailedOracle(oracles)
		if reason == "" || failed == nil || failureIdentity != failed.FailureIdentity || divergence != nil {
			return errors.New("oracle failure outcome is inconsistent")
		}
	case OutcomeReplayDiverged:
		if reason == "" || !validSHA256(failureIdentity) || divergence == nil {
			return errors.New("replay divergence outcome is inconsistent")
		}
		if err := validateDivergence(*divergence); err != nil {
			return err
		}
		want, err := normalizedFailureIdentity(outcome, reason, divergenceIdentity(*divergence))
		if err != nil || failureIdentity != want {
			return errors.New("replay failure identity does not match its divergence")
		}
	default:
		return fmt.Errorf("cluster outcome %q is invalid", outcome)
	}
	return nil
}

func validateNodeResults(nodes []NodeResult, limit uint64) error {
	if err := checkCapacity("scenario_actions", uint64(len(nodes)), limit); err != nil {
		return err
	}
	var previous NodeHandle
	for _, node := range nodes {
		if node.Handle.Node == "" || node.Handle.Incarnation == 0 || !terminalNodeState(node.State) || node.State == NodeStateFailed && node.Reason == "" || node.State != NodeStateFailed && node.Reason != "" || len(node.Reason) > MaximumTerminalReasonBytes {
			return errors.New("cluster node result is invalid or unordered")
		}
		if previous.Node != "" && !nodeHandleBefore(previous, node.Handle) {
			return errors.New("cluster node result is invalid or unordered")
		}
		previous = node.Handle
	}
	return nil
}

func terminalNodeState(state NodeState) bool {
	switch state {
	case NodeStateExited, NodeStateStopped, NodeStateCrashed, NodeStateFailed:
		return true
	default:
		return false
	}
}

func nodeHandleBefore(left, right NodeHandle) bool {
	if left.Node != right.Node {
		return left.Node < right.Node
	}
	return left.Incarnation < right.Incarnation
}

func validateTransitions(transitions []LifecycleTransition, limit uint64) error {
	if err := checkCapacity("scenario_actions", uint64(len(transitions)), limit); err != nil {
		return err
	}
	for index, transition := range transitions {
		if transition.Ordinal != uint64(index) || transition.Handle.Node == "" || transition.Handle.Incarnation == 0 || !validLifecycleTransition(transition) {
			return errors.New("cluster lifecycle transition is invalid")
		}
	}
	return nil
}

func validLifecycleTransition(transition LifecycleTransition) bool {
	switch transition.Action {
	case LifecycleStart:
		return transition.From == NodeStateDefined && transition.To == NodeStateRunning
	case LifecycleWait:
		return transition.From == NodeStateRunning && (transition.To == NodeStateExited || transition.To == NodeStateFailed)
	case LifecycleStop:
		return transition.From == NodeStateRunning && (transition.To == NodeStateStopped || transition.To == NodeStateFailed)
	case LifecycleCrash:
		return transition.From == NodeStateRunning && transition.To == NodeStateCrashed
	case LifecycleRestart:
		return (transition.From == NodeStateStopped || transition.From == NodeStateCrashed || transition.From == NodeStateExited || transition.From == NodeStateFailed) && transition.To == NodeStateRunning
	default:
		return false
	}
}

func validNodeState(state NodeState) bool {
	switch state {
	case NodeStateDefined, NodeStateRunning, NodeStateExited, NodeStateStopped, NodeStateCrashed, NodeStateFailed:
		return true
	default:
		return false
	}
}

func validateDivergence(divergence ReplayDivergence) error {
	switch divergence.Dimension {
	case ReplayDimensionTransition:
		if divergence.ExpectedSHA256 != "" || divergence.ActualSHA256 != "" || divergence.ExpectedNetwork != nil || divergence.ActualNetwork != nil || divergence.ExpectedVolume != nil || divergence.ActualVolume != nil || divergence.ExpectedFault != nil || divergence.ActualFault != nil || divergence.ExpectedScenario != nil || divergence.ActualScenario != nil {
			return errors.New("transition replay divergence contains terminal hashes")
		}
		if divergence.Expected.Action == "" && divergence.Actual.Action == "" {
			return errors.New("transition replay divergence contains no transition")
		}
		if divergence.Expected.Action != "" && !validLifecycleTransition(divergence.Expected) || divergence.Actual.Action != "" && !validLifecycleTransition(divergence.Actual) {
			return errors.New("transition replay divergence is invalid")
		}
	case ReplayDimensionNetwork:
		if !validSHA256(divergence.ExpectedSHA256) || !validSHA256(divergence.ActualSHA256) || divergence.ExpectedSHA256 == divergence.ActualSHA256 {
			return errors.New("network replay divergence is invalid")
		}
		if divergence.ExpectedNetwork != nil && !validNetworkTransition(*divergence.ExpectedNetwork) || divergence.ActualNetwork != nil && !validNetworkTransition(*divergence.ActualNetwork) {
			return errors.New("network replay divergence transitions are invalid")
		}
		if divergence.Expected.Action != "" || divergence.Actual.Action != "" || divergence.ExpectedVolume != nil || divergence.ActualVolume != nil || divergence.ExpectedFault != nil || divergence.ActualFault != nil || divergence.ExpectedScenario != nil || divergence.ActualScenario != nil {
			return errors.New("network replay divergence contains lifecycle transitions")
		}
	case ReplayDimensionVolume:
		if !validSHA256(divergence.ExpectedSHA256) || !validSHA256(divergence.ActualSHA256) || divergence.ExpectedSHA256 == divergence.ActualSHA256 {
			return errors.New("volume replay divergence is invalid")
		}
		if divergence.ExpectedVolume != nil && !validVolumeTransition(*divergence.ExpectedVolume) || divergence.ActualVolume != nil && !validVolumeTransition(*divergence.ActualVolume) {
			return errors.New("volume replay divergence transitions are invalid")
		}
		if divergence.Expected.Action != "" || divergence.Actual.Action != "" || divergence.ExpectedNetwork != nil || divergence.ActualNetwork != nil || divergence.ExpectedFault != nil || divergence.ActualFault != nil || divergence.ExpectedScenario != nil || divergence.ActualScenario != nil {
			return errors.New("volume replay divergence contains another transition dimension")
		}
	case ReplayDimensionFault:
		if !validSHA256(divergence.ExpectedSHA256) || !validSHA256(divergence.ActualSHA256) || divergence.ExpectedSHA256 == divergence.ActualSHA256 || divergence.ExpectedFault == nil && divergence.ActualFault == nil {
			return errors.New("fault replay divergence is invalid")
		}
		if divergence.ExpectedFault != nil {
			if err := validateFaultAction(*divergence.ExpectedFault); err != nil {
				return err
			}
		}
		if divergence.ActualFault != nil {
			if err := validateFaultAction(*divergence.ActualFault); err != nil {
				return err
			}
		}
		if divergence.Expected.Action != "" || divergence.Actual.Action != "" || divergence.ExpectedNetwork != nil || divergence.ActualNetwork != nil || divergence.ExpectedVolume != nil || divergence.ActualVolume != nil || divergence.ExpectedScenario != nil || divergence.ActualScenario != nil {
			return errors.New("fault replay divergence contains another transition dimension")
		}
	case ReplayDimensionScenario:
		if !validSHA256(divergence.ExpectedSHA256) || !validSHA256(divergence.ActualSHA256) || divergence.ExpectedSHA256 == divergence.ActualSHA256 || divergence.ExpectedScenario == nil && divergence.ActualScenario == nil {
			return errors.New("scenario replay divergence is invalid")
		}
		if divergence.ExpectedScenario != nil {
			if err := validateScenarioDecision(*divergence.ExpectedScenario); err != nil {
				return err
			}
		}
		if divergence.ActualScenario != nil {
			if err := validateScenarioDecision(*divergence.ActualScenario); err != nil {
				return err
			}
		}
		if divergence.Expected.Action != "" || divergence.Actual.Action != "" || divergence.ExpectedNetwork != nil || divergence.ActualNetwork != nil || divergence.ExpectedVolume != nil || divergence.ActualVolume != nil || divergence.ExpectedFault != nil || divergence.ActualFault != nil {
			return errors.New("scenario replay divergence contains another transition dimension")
		}
	case ReplayDimensionEvidence:
		if !validSHA256(divergence.ExpectedSHA256) || !validSHA256(divergence.ActualSHA256) || divergence.ExpectedSHA256 == divergence.ActualSHA256 || divergence.Expected.Action != "" || divergence.Actual.Action != "" || divergence.ExpectedNetwork != nil || divergence.ActualNetwork != nil || divergence.ExpectedVolume != nil || divergence.ActualVolume != nil || divergence.ExpectedFault != nil || divergence.ActualFault != nil || divergence.ExpectedScenario != nil || divergence.ActualScenario != nil {
			return errors.New("evidence replay divergence is invalid")
		}
	case ReplayDimensionTerminal:
		if !validSHA256(divergence.ExpectedSHA256) || !validSHA256(divergence.ActualSHA256) || divergence.ExpectedSHA256 == divergence.ActualSHA256 || divergence.Expected.Action != "" || divergence.Actual.Action != "" {
			return errors.New("terminal replay divergence is invalid")
		}
		if divergence.ExpectedNetwork != nil || divergence.ActualNetwork != nil || divergence.ExpectedVolume != nil || divergence.ActualVolume != nil || divergence.ExpectedFault != nil || divergence.ActualFault != nil || divergence.ExpectedScenario != nil || divergence.ActualScenario != nil {
			return errors.New("terminal replay divergence contains network transitions")
		}
	default:
		return errors.New("replay divergence dimension is invalid")
	}
	return nil
}

func validateLeaks(leaks []LeakDiagnostic) error {
	seen := make(map[NodeHandle]struct{}, len(leaks))
	for _, leak := range leaks {
		if leak.Handle.Node == "" || leak.Handle.Incarnation == 0 || leak.Kind != LeakRevokedGoroutineMayRemain {
			return errors.New("cluster leak diagnostic is invalid")
		}
		if _, ok := seen[leak.Handle]; ok {
			return errors.New("cluster leak diagnostic is duplicated")
		}
		seen[leak.Handle] = struct{}{}
	}
	return nil
}

func validateBackendLeaks(backend Backend, leaks []LeakDiagnostic) error {
	if err := validateLeaks(leaks); err != nil {
		return err
	}
	if backend == BackendProcess && len(leaks) != 0 {
		return errors.New("process backend cannot contain in-process leak diagnostics")
	}
	return nil
}

func validateOutputs(outputs []OutputObservation, limit uint64) error {
	var retained uint64
	for index, output := range outputs {
		if output.Handle.Node == "" || output.Handle.Incarnation == 0 || output.Stream != OutputStdout && output.Stream != OutputStderr || !validSHA256(output.FullSHA256) || output.RetainedBytes != uint64(len(output.Bytes)) || output.TotalBytes < output.RetainedBytes || output.DiscardedBytes != output.TotalBytes-output.RetainedBytes || output.Truncated != (output.DiscardedBytes != 0) {
			return errors.New("cluster output metadata is inconsistent")
		}
		if !output.Truncated {
			digest := sha256.Sum256(output.Bytes)
			if output.FullSHA256 != fmt.Sprintf("sha256:%x", digest) {
				return errors.New("cluster output full hash does not match retained bytes")
			}
		}
		if index != 0 && !outputBefore(outputs[index-1], output) {
			return errors.New("cluster outputs are not strictly ordered")
		}
		retained = saturatingAdd(retained, output.RetainedBytes)
		if err := checkCapacity("observation_bytes", retained, limit); err != nil {
			return err
		}
	}
	return nil
}

func validateNetworkRecord(record NetworkRecord, limits Limits) error {
	if err := checkCapacity("network_transitions", uint64(len(record.Transitions)), limits.NetworkTransitions); err != nil {
		return err
	}
	for index, transition := range record.Transitions {
		if transition.Ordinal != uint64(index) || !validNetworkTransition(transition) {
			return errors.New("cluster network transition is invalid")
		}
		if index != 0 && networkTransitionLane(record.Transitions[index-1]) > networkTransitionLane(transition) {
			return errors.New("cluster network transitions are not canonically ordered")
		}
	}
	encoded, err := encodeRuntimeNetworkTransitions(record.Transitions)
	if err != nil {
		return fmt.Errorf("encode cluster network transitions: %w", err)
	}
	digest := sha256.Sum256(append([]byte("gomadv3-simulation-network-transitions/v2\x00"), encoded...))
	if record.Snapshot.TransitionSHA256 != fmt.Sprintf("sha256:%x", digest) {
		return errors.New("cluster network transition digest does not match its transitions")
	}
	if !validSHA256(record.Snapshot.Identity) {
		return errors.New("cluster network snapshot identity is invalid")
	}
	wantSnapshotIdentity, err := networkSnapshotIdentity(record.Snapshot)
	if err != nil {
		return err
	}
	if record.Snapshot.Identity != wantSnapshotIdentity {
		return errors.New("cluster network snapshot identity does not match its contents")
	}
	if err := checkCapacity("network_listeners", uint64(len(record.Snapshot.Listeners)), limits.NetworkListeners); err != nil {
		return err
	}
	if err := checkCapacity("network_connections", uint64(len(record.Snapshot.Connections)), limits.NetworkConnections); err != nil {
		return err
	}
	if err := checkCapacity("network_deliveries", uint64(len(record.Snapshot.Deliveries)), limits.NetworkDeliveries); err != nil {
		return err
	}
	var pendingBytes uint64
	for index, node := range record.Snapshot.Nodes {
		if node.Node == "" || node.Address == "" || node.NextListenerPort == 0 || node.NextClientPort == 0 || index != 0 && record.Snapshot.Nodes[index-1].Node >= node.Node {
			return errors.New("cluster network node snapshot is invalid or unordered")
		}
	}
	for index, link := range record.Snapshot.Links {
		if link.From == "" || link.To == "" || link.From == link.To || index != 0 && networkLinkKey(record.Snapshot.Links[index-1].From, record.Snapshot.Links[index-1].To) >= networkLinkKey(link.From, link.To) {
			return errors.New("cluster network link snapshot is invalid or unordered")
		}
	}
	for index, listener := range record.Snapshot.Listeners {
		if !validNetworkEndpoint(listener.Endpoint, true) || index != 0 && !networkEndpointBefore(record.Snapshot.Listeners[index-1].Endpoint, listener.Endpoint) {
			return errors.New("cluster network listener snapshot is invalid or unordered")
		}
	}
	for index, connection := range record.Snapshot.Connections {
		if connection.Identity == 0 || !validNetworkEndpoint(connection.Client, true) || !validNetworkEndpoint(connection.Server, true) || connection.Reset && !connection.Closed || index != 0 && record.Snapshot.Connections[index-1].Identity >= connection.Identity {
			return errors.New("cluster network connection snapshot is invalid or unordered")
		}
	}
	for index, delivery := range record.Snapshot.Deliveries {
		if delivery.Identity == 0 || delivery.Connection == 0 || delivery.Bytes == 0 || !validNetworkEndpoint(delivery.Source, true) || !validNetworkEndpoint(delivery.Destination, true) || index != 0 && record.Snapshot.Deliveries[index-1].Identity >= delivery.Identity {
			return errors.New("cluster network delivery snapshot is invalid or unordered")
		}
		pendingBytes = saturatingAdd(pendingBytes, delivery.Bytes)
	}
	return checkCapacity("network_bytes", pendingBytes, limits.NetworkBytes)
}

func validateVolumeRecord(record VolumeRecord, limits Limits) error {
	if err := checkCapacity("volume_transitions", uint64(len(record.Transitions)), limits.VolumeTransitions); err != nil {
		return err
	}
	for index, transition := range record.Transitions {
		if transition.Ordinal != uint64(index) || !validVolumeTransition(transition) {
			return errors.New("cluster volume transition is invalid")
		}
		if index != 0 && volumeTransitionLane(record.Transitions[index-1]) > volumeTransitionLane(transition) {
			return errors.New("cluster volume transitions are not canonically ordered")
		}
	}
	if record.Snapshot.TransitionSHA256 != volumeTransitionsIdentity(record.Transitions) {
		return errors.New("cluster volume transition digest does not match its transitions")
	}
	if !validSHA256(record.Snapshot.Identity) || record.Snapshot.Identity != volumeRunSnapshotIdentity(record.Snapshot) {
		return errors.New("cluster volume snapshot identity does not match its contents")
	}
	if err := checkCapacity("volumes", uint64(len(record.Snapshot.Volumes)), limits.Volumes); err != nil {
		return err
	}
	var previous string
	for _, volume := range record.Snapshot.Volumes {
		key := string(volume.Node) + "\x00" + string(volume.Volume)
		if volume.Node == "" || volume.Volume == "" || volume.Mount == "" || volume.CapacityBytes == 0 || key <= previous || !validSHA256(volume.PendingSHA256) || !validSHA256(volume.Identity) || volume.PendingOperations > limits.VolumeOperations {
			return errors.New("cluster volume state snapshot is invalid or unordered")
		}
		if err := validateVolumeEntries(volume.Persisted); err != nil {
			return err
		}
		if err := validateVolumeEntries(volume.Volatile); err != nil {
			return err
		}
		if volume.Identity != volumeStateSnapshotIdentity(volume) {
			return errors.New("cluster volume state snapshot identity does not match its contents")
		}
		previous = key
	}
	return nil
}

func validateVolumeEntries(entries []VolumeEntrySnapshot) error {
	var previous string
	for _, entry := range entries {
		if entry.Path == "" || entry.Path <= previous || entry.Kind != "file" && entry.Kind != "directory" || entry.Kind == "file" && !validSHA256(entry.DataSHA256) || entry.Kind == "directory" && (entry.Size != 0 || entry.DataSHA256 != "") {
			return errors.New("cluster volume entry snapshot is invalid or unordered")
		}
		previous = entry.Path
	}
	return nil
}

func validVolumeTransition(transition VolumeTransition) bool {
	if transition.Handle.Node == "" || transition.Handle.Incarnation == 0 || transition.Volume == "" || transition.Outcome != VolumeOutcomeOK && transition.Outcome != VolumeOutcomeCapacity && transition.Outcome != VolumeOutcomeStale {
		return false
	}
	switch transition.Kind {
	case VolumeOperationAllocate, VolumeOperationResize, VolumeOperationWrite, VolumeOperationMetadata, VolumeOperationNamespace:
		if transition.Operation == 0 || !validSHA256(transition.EffectSHA256) {
			return false
		}
	case VolumeOperationFileSync, VolumeOperationDirectorySync, VolumeOperationFlush, VolumeOperationCrash:
		if transition.Operation != 0 || len(transition.Dependencies) != 0 || transition.EffectSHA256 != "" {
			return false
		}
	default:
		return false
	}
	if transition.PayloadSHA256 != "" && !validSHA256(transition.PayloadSHA256) {
		return false
	}
	return strictlyIncreasingUint64s(transition.Dependencies) && strictlyIncreasingUint64s(transition.SelectedOperations)
}

func strictlyIncreasingUint64s(values []uint64) bool {
	for index, value := range values {
		if value == 0 || index != 0 && values[index-1] >= value {
			return false
		}
	}
	return true
}

func volumeTransitionLane(transition VolumeTransition) string {
	return string(transition.Handle.Node) + "\x00" + string(transition.Volume)
}

type volumeIdentityHasher struct {
	data []byte
}

func newVolumeIdentityHasher() *volumeIdentityHasher {
	return &volumeIdentityHasher{}
}

func (hasher *volumeIdentityHasher) uint64(value uint64) {
	var encoded [8]byte
	binary.LittleEndian.PutUint64(encoded[:], value)
	hasher.data = append(hasher.data, encoded[:]...)
}

func (hasher *volumeIdentityHasher) string(value string) {
	hasher.bytes([]byte(value))
}

func (hasher *volumeIdentityHasher) bytes(value []byte) {
	hasher.uint64(uint64(len(value)))
	hasher.data = append(hasher.data, value...)
}

func (hasher *volumeIdentityHasher) digest() string {
	digest := sha256.Sum256(hasher.data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func volumeTransitionsIdentity(transitions []VolumeTransition) string {
	hasher := newVolumeIdentityHasher()
	hasher.string("gomadv3-volume-transitions/v1")
	hasher.uint64(uint64(len(transitions)))
	for _, transition := range transitions {
		writeVolumeTransitionIdentity(hasher, transition)
	}
	return hasher.digest()
}

func writeVolumeTransitionIdentity(hasher *volumeIdentityHasher, transition VolumeTransition) {
	hasher.uint64(transition.Ordinal)
	hasher.string(string(transition.Kind))
	hasher.string(string(transition.Handle.Node))
	hasher.uint64(transition.Handle.Incarnation)
	hasher.string(string(transition.Volume))
	hasher.uint64(transition.Operation)
	hasher.uint64(uint64(len(transition.Dependencies)))
	for _, dependency := range transition.Dependencies {
		hasher.uint64(dependency)
	}
	hasher.uint64(uint64(len(transition.SelectedOperations)))
	for _, selected := range transition.SelectedOperations {
		hasher.uint64(selected)
	}
	hasher.string(transition.Path)
	hasher.string(transition.Destination)
	hasher.uint64(transition.Inode)
	hasher.uint64(transition.Offset)
	hasher.uint64(transition.Bytes)
	hasher.string(transition.PayloadSHA256)
	hasher.string(transition.EffectSHA256)
	hasher.string(string(transition.Outcome))
}

func volumeRunSnapshotIdentity(snapshot VolumeSnapshot) string {
	hasher := newVolumeIdentityHasher()
	hasher.string("gomadv3-volume-run-snapshot/v1")
	hasher.string(snapshot.TransitionSHA256)
	hasher.uint64(uint64(len(snapshot.Volumes)))
	for _, volume := range snapshot.Volumes {
		hasher.string(string(volume.Node))
		hasher.string(string(volume.Volume))
		hasher.string(volume.Identity)
	}
	return hasher.digest()
}

func volumeStateSnapshotIdentity(snapshot VolumeStateSnapshot) string {
	hasher := newVolumeIdentityHasher()
	hasher.string("gomadv3-volume-snapshot/v1")
	hasher.string(string(snapshot.Volume))
	hasher.string(snapshot.Mount)
	hasher.uint64(snapshot.CapacityBytes)
	writeVolumeEntriesIdentity(hasher, snapshot.Persisted)
	writeVolumeEntriesIdentity(hasher, snapshot.Volatile)
	hasher.uint64(snapshot.PendingOperations)
	hasher.string(snapshot.PendingSHA256)
	hasher.uint64(snapshot.NextOperation)
	return hasher.digest()
}

func writeVolumeEntriesIdentity(hasher *volumeIdentityHasher, entries []VolumeEntrySnapshot) {
	hasher.uint64(uint64(len(entries)))
	for _, entry := range entries {
		hasher.string(entry.Path)
		hasher.uint64(uint64(entry.Mode))
		hasher.string(entry.Kind)
		hasher.uint64(uint64(entry.ModTime))
		hasher.uint64(entry.Size)
		hasher.string(entry.DataSHA256)
	}
}

func networkSnapshotIdentity(snapshot NetworkSnapshot) (string, error) {
	snapshot.Identity = ""
	encoded, err := encodeRuntimeNetworkSnapshotIdentity(snapshot)
	if err != nil {
		return "", fmt.Errorf("encode cluster network snapshot identity: %w", err)
	}
	digest := sha256.Sum256(append([]byte("gomadv3-simulation-network-snapshot/v2\x00"), encoded...))
	return fmt.Sprintf("sha256:%x", digest), nil
}

func validNetworkTransition(transition NetworkTransition) bool {
	switch transition.Kind {
	case NetworkListen, NetworkDial, NetworkAccept, NetworkWrite, NetworkDeliver, NetworkClose, NetworkListenerClose, NetworkStop, NetworkCrash:
		if !validNetworkEndpoint(transition.Source, true) {
			return false
		}
	case NetworkPartition, NetworkHeal, NetworkDelay, NetworkDisconnect, NetworkReconnect, NetworkDirectionalDelay:
		if !validNetworkEndpoint(transition.Source, false) || !validNetworkEndpoint(transition.Destination, false) || transition.Source.Node == transition.Destination.Node {
			return false
		}
	default:
		return false
	}
	switch transition.Outcome {
	case NetworkOutcomeOK, NetworkOutcomeClosed, NetworkOutcomeRefused, NetworkOutcomeDeadline, NetworkOutcomePartitionDrop, NetworkOutcomeStaleDrop, NetworkOutcomeReset, NetworkOutcomeCapacity, NetworkOutcomeUnsupported:
	default:
		return false
	}
	if transition.PayloadSHA256 != "" && !validSHA256(transition.PayloadSHA256) {
		return false
	}
	return true
}

func validNetworkEndpoint(endpoint NetworkEndpoint, requireIncarnation bool) bool {
	if endpoint.Node == "" || endpoint.Address == "" {
		return false
	}
	return !requireIncarnation || endpoint.Incarnation != 0
}

func networkTransitionLane(transition NetworkTransition) string {
	switch transition.Kind {
	case NetworkPartition, NetworkHeal, NetworkDelay, NetworkDisconnect, NetworkReconnect, NetworkDirectionalDelay:
		return "topology"
	case NetworkListen, NetworkListenerClose:
		return "listener\x00" + networkEndpointKey(transition.Source)
	case NetworkDial, NetworkAccept, NetworkWrite, NetworkDeliver, NetworkClose:
		if transition.Connection != 0 {
			return fmt.Sprintf("connection\x00%020d", transition.Connection)
		}
		return "operation\x00" + string(transition.Kind) + "\x00" + networkEndpointKey(transition.Source) + "\x00" + networkEndpointKey(transition.Destination)
	case NetworkStop, NetworkCrash:
		return "lifecycle\x00" + networkEndpointKey(transition.Source)
	default:
		return "unknown\x00" + string(transition.Kind)
	}
}

func networkEndpointKey(endpoint NetworkEndpoint) string {
	return fmt.Sprintf("%s\x00%020d\x00%s\x00%05d", endpoint.Node, endpoint.Incarnation, endpoint.Address, endpoint.Port)
}

func networkEndpointBefore(left, right NetworkEndpoint) bool {
	if left.Node != right.Node {
		return left.Node < right.Node
	}
	if left.Incarnation != right.Incarnation {
		return left.Incarnation < right.Incarnation
	}
	return left.Port < right.Port
}

func networkLinkKey(from, to NodeID) string {
	return string(from) + "\x00" + string(to)
}

func outputBefore(left, right OutputObservation) bool {
	if left.Handle.Node != right.Handle.Node {
		return left.Handle.Node < right.Handle.Node
	}
	if left.Handle.Incarnation != right.Handle.Incarnation {
		return left.Handle.Incarnation < right.Handle.Incarnation
	}
	if left.Stream != right.Stream {
		return left.Stream == OutputStdout
	}
	return false
}

func inProcessLimitations() []Limitation {
	return []Limitation{
		LimitationSharedPackageGlobals,
		LimitationRevokedGoroutinesMayRemain,
		LimitationCPULoopsRequireWatchdog,
		LimitationHardIsolationRequiresProcess,
	}
}

func backendLimitations(backend Backend) []Limitation {
	if backend == BackendProcess {
		return nil
	}
	return inProcessLimitations()
}

func hashSpec(spec Spec) (string, error) {
	spec.Replay = nil
	if spec.Faults == nil {
		faults, err := NewFaultPlan(nil)
		if err != nil {
			return "", err
		}
		spec.Faults = &faults
	} else {
		faults := cloneFaultPlan(*spec.Faults)
		spec.Faults = &faults
	}
	scenarioChoices, err := scenarioChoicePlanForSpec(spec)
	if err != nil {
		return "", err
	}
	spec.ScenarioChoices = &scenarioChoices
	return hashCanonical("gomadv3-simulation-spec/v6", spec)
}

func clusterRecordIdentity(record ClusterRecord) (string, error) {
	record.Identity = ""
	return hashCanonical("gomadv3-cluster-record/v6", record)
}

func replayPlanIdentity(plan ReplayPlan) (string, error) {
	plan.Identity = ""
	return hashCanonical("gomadv3-cluster-replay/v6", plan)
}

func hashCanonical(domain string, value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode %s identity: %w", domain, err)
	}
	payload := make([]byte, 0, len(domain)+1+len(encoded))
	payload = append(payload, domain...)
	payload = append(payload, 0)
	payload = append(payload, encoded...)
	digest := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func validSHA256(value string) bool {
	if len(value) != len("sha256:")+sha256.Size*2 || value[:len("sha256:")] != "sha256:" {
		return false
	}
	decoded, err := hex.DecodeString(value[len("sha256:"):])
	return err == nil && len(decoded) == sha256.Size
}

func cloneReplayPlan(plan ReplayPlan) ReplayPlan {
	plan.Nodes = append([]NodeResult(nil), plan.Nodes...)
	plan.Transitions = append([]LifecycleTransition(nil), plan.Transitions...)
	plan.FaultPlan = cloneFaultPlan(plan.FaultPlan)
	plan.Faults = cloneFaultRealizations(plan.Faults)
	plan.ScenarioChoices = cloneScenarioChoicePlan(plan.ScenarioChoices)
	plan.Scenarios = cloneScenarioDecisions(plan.Scenarios)
	plan.History = cloneHistoryOperations(plan.History)
	plan.Observations = cloneObservations(plan.Observations)
	plan.Oracles = cloneOracleResults(plan.Oracles)
	plan.Network = cloneNetworkRecord(plan.Network)
	plan.Volumes = cloneVolumeRecord(plan.Volumes)
	plan.Outputs = cloneOutputs(plan.Outputs)
	plan.Leaks = append([]LeakDiagnostic(nil), plan.Leaks...)
	return plan
}

func cloneNetworkRecord(record NetworkRecord) NetworkRecord {
	record.Transitions = append([]NetworkTransition(nil), record.Transitions...)
	record.Snapshot.Nodes = append([]NetworkNodeSnapshot(nil), record.Snapshot.Nodes...)
	record.Snapshot.Links = append([]NetworkLinkSnapshot(nil), record.Snapshot.Links...)
	record.Snapshot.Listeners = append([]NetworkListenerSnapshot(nil), record.Snapshot.Listeners...)
	record.Snapshot.Connections = append([]NetworkConnectionSnapshot(nil), record.Snapshot.Connections...)
	record.Snapshot.Deliveries = append([]NetworkDeliverySnapshot(nil), record.Snapshot.Deliveries...)
	return record
}

func cloneVolumeRecord(record VolumeRecord) VolumeRecord {
	record.Transitions = append([]VolumeTransition(nil), record.Transitions...)
	for index := range record.Transitions {
		record.Transitions[index].Dependencies = append([]uint64(nil), record.Transitions[index].Dependencies...)
		record.Transitions[index].SelectedOperations = append([]uint64(nil), record.Transitions[index].SelectedOperations...)
	}
	record.Snapshot.Volumes = append([]VolumeStateSnapshot(nil), record.Snapshot.Volumes...)
	for index := range record.Snapshot.Volumes {
		record.Snapshot.Volumes[index].Persisted = append([]VolumeEntrySnapshot(nil), record.Snapshot.Volumes[index].Persisted...)
		record.Snapshot.Volumes[index].Volatile = append([]VolumeEntrySnapshot(nil), record.Snapshot.Volumes[index].Volatile...)
	}
	return record
}

func cloneOutputs(outputs []OutputObservation) []OutputObservation {
	cloned := append([]OutputObservation(nil), outputs...)
	for index := range cloned {
		cloned[index].Bytes = append([]byte(nil), cloned[index].Bytes...)
	}
	return cloned
}

func cloneReplayDivergence(divergence *ReplayDivergence) *ReplayDivergence {
	if divergence == nil {
		return nil
	}
	cloned := *divergence
	if divergence.ExpectedNetwork != nil {
		expected := *divergence.ExpectedNetwork
		cloned.ExpectedNetwork = &expected
	}
	if divergence.ActualNetwork != nil {
		actual := *divergence.ActualNetwork
		cloned.ActualNetwork = &actual
	}
	if divergence.ExpectedVolume != nil {
		expected := *divergence.ExpectedVolume
		expected.Dependencies = append([]uint64(nil), expected.Dependencies...)
		expected.SelectedOperations = append([]uint64(nil), expected.SelectedOperations...)
		cloned.ExpectedVolume = &expected
	}
	if divergence.ActualVolume != nil {
		actual := *divergence.ActualVolume
		actual.Dependencies = append([]uint64(nil), actual.Dependencies...)
		actual.SelectedOperations = append([]uint64(nil), actual.SelectedOperations...)
		cloned.ActualVolume = &actual
	}
	cloned.ExpectedFault = cloneFaultActionPointer(divergence.ExpectedFault)
	cloned.ActualFault = cloneFaultActionPointer(divergence.ActualFault)
	if divergence.ExpectedScenario != nil {
		expected := *divergence.ExpectedScenario
		expected.Alternatives = append([]string(nil), expected.Alternatives...)
		cloned.ExpectedScenario = &expected
	}
	if divergence.ActualScenario != nil {
		actual := *divergence.ActualScenario
		actual.Alternatives = append([]string(nil), actual.Alternatives...)
		cloned.ActualScenario = &actual
	}
	return &cloned
}

func cloneFaultPlan(plan FaultPlan) FaultPlan {
	plan.Actions = cloneFaultActions(plan.Actions)
	return plan
}

func cloneNodeSpecs(nodes []NodeSpec) []NodeSpec {
	cloned := make([]NodeSpec, len(nodes))
	for index, node := range nodes {
		cloned[index] = node
		cloned[index].Config = append([]byte(nil), node.Config...)
		cloned[index].Volumes = append([]VolumeMount(nil), node.Volumes...)
	}
	return cloned
}
