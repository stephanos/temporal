package gomadv3sim

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const ClusterInspectionSchema = "gomadv3.cluster-inspection/v1"

type ClusterInspectionCounts struct {
	Nodes              uint64 `json:"nodes"`
	Incarnations       uint64 `json:"incarnations"`
	Lifecycle          uint64 `json:"lifecycle_transitions"`
	Faults             uint64 `json:"faults"`
	ScenarioDecisions  uint64 `json:"scenario_decisions"`
	HistoryOperations  uint64 `json:"history_operations"`
	Observations       uint64 `json:"observations"`
	OracleResults      uint64 `json:"oracle_results"`
	NetworkTransitions uint64 `json:"network_transitions"`
	VolumeTransitions  uint64 `json:"volume_transitions"`
}

type ClusterInspectionTapes struct {
	FaultPlanSHA256 string `json:"fault_plan_sha256"`
	NetworkSHA256   string `json:"network_sha256"`
	VolumeSHA256    string `json:"volume_sha256"`
}

type ClusterInspectionTerminal struct {
	NetworkSHA256 string `json:"network_sha256"`
	VolumeSHA256  string `json:"volume_sha256"`
}

type ClusterOracleFailure struct {
	Name     string `json:"name"`
	Identity string `json:"identity"`
}

type ClusterInspection struct {
	Schema          string                    `json:"schema"`
	RecordIdentity  string                    `json:"record_identity"`
	Backend         Backend                   `json:"backend"`
	Fidelity        Fidelity                  `json:"fidelity"`
	Seed            uint64                    `json:"seed"`
	Outcome         Outcome                   `json:"outcome"`
	Reason          string                    `json:"reason,omitempty"`
	FailureIdentity string                    `json:"failure_identity,omitempty"`
	Static          ClusterStaticIdentities   `json:"static_identities"`
	Models          ClusterModelIdentities    `json:"model_identities"`
	Limits          Limits                    `json:"limits"`
	Counts          ClusterInspectionCounts   `json:"counts"`
	Tapes           ClusterInspectionTapes    `json:"tapes"`
	Terminal        ClusterInspectionTerminal `json:"terminal"`
	OracleFailures  []ClusterOracleFailure    `json:"oracle_failures"`
	Identity        string                    `json:"identity"`
}

func InspectClusterRecord(record ClusterRecord) (ClusterInspection, error) {
	if err := validateClusterRecord(record); err != nil {
		return ClusterInspection{}, err
	}
	failures := make([]ClusterOracleFailure, 0)
	for _, oracle := range record.Oracles {
		if !oracle.Passed {
			failures = append(failures, ClusterOracleFailure{Name: oracle.Name, Identity: oracle.FailureIdentity})
		}
	}
	sort.Slice(failures, func(left, right int) bool { return failures[left].Name < failures[right].Name })
	inspection := ClusterInspection{
		Schema: ClusterInspectionSchema, RecordIdentity: record.Identity, Backend: record.Backend, Fidelity: record.Fidelity,
		Seed: record.Seed, Outcome: record.Outcome, Reason: record.Reason, FailureIdentity: record.FailureIdentity,
		Static: record.Static, Models: record.Models, Limits: record.Limits,
		Counts: ClusterInspectionCounts{
			Nodes: uint64(len(record.NodeSpecs)), Incarnations: uint64(len(record.Nodes)), Lifecycle: uint64(len(record.Transitions)),
			Faults: uint64(len(record.Faults)), ScenarioDecisions: uint64(len(record.Scenarios)), HistoryOperations: uint64(len(record.History)),
			Observations: uint64(len(record.Observations)), OracleResults: uint64(len(record.Oracles)),
			NetworkTransitions: uint64(len(record.Network.Transitions)), VolumeTransitions: uint64(len(record.Volumes.Transitions)),
		},
		Tapes: ClusterInspectionTapes{
			FaultPlanSHA256: record.FaultPlan.Identity,
			NetworkSHA256:   record.Network.Snapshot.TransitionSHA256,
			VolumeSHA256:    record.Volumes.Snapshot.TransitionSHA256,
		},
		Terminal:       ClusterInspectionTerminal{NetworkSHA256: record.Network.Snapshot.Identity, VolumeSHA256: record.Volumes.Snapshot.Identity},
		OracleFailures: failures,
	}
	identity, err := clusterInspectionIdentity(inspection)
	if err != nil {
		return ClusterInspection{}, err
	}
	inspection.Identity = identity
	return inspection, nil
}

func EncodeClusterInspection(inspection ClusterInspection) ([]byte, error) {
	if err := validateClusterInspection(inspection); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(inspection)
	if err != nil {
		return nil, fmt.Errorf("encode cluster inspection: %w", err)
	}
	if len(encoded) > MaximumClusterRecordBytes {
		return nil, &CapacityError{Resource: "cluster_inspection_bytes", Required: uint64(len(encoded)), Maximum: MaximumClusterRecordBytes}
	}
	return encoded, nil
}

func FormatClusterInspection(inspection ClusterInspection) (string, error) {
	if err := validateClusterInspection(inspection); err != nil {
		return "", err
	}
	lines := []string{
		"record: " + inspection.RecordIdentity,
		"backend: " + string(inspection.Backend),
		"fidelity: " + string(inspection.Fidelity),
		"outcome: " + string(inspection.Outcome),
		fmt.Sprintf("nodes: %d", inspection.Counts.Nodes),
		fmt.Sprintf("faults: %d", inspection.Counts.Faults),
		fmt.Sprintf("scenario decisions: %d", inspection.Counts.ScenarioDecisions),
		fmt.Sprintf("history operations: %d", inspection.Counts.HistoryOperations),
		fmt.Sprintf("oracle results: %d", inspection.Counts.OracleResults),
		"network terminal: " + inspection.Terminal.NetworkSHA256,
		"volume terminal: " + inspection.Terminal.VolumeSHA256,
	}
	if inspection.FailureIdentity != "" {
		lines = append(lines, "failure: "+inspection.FailureIdentity)
	}
	for _, failure := range inspection.OracleFailures {
		lines = append(lines, "oracle failure: "+failure.Name+" "+failure.Identity)
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func validateClusterInspection(inspection ClusterInspection) error {
	if inspection.Schema != ClusterInspectionSchema {
		return fmt.Errorf("cluster inspection schema = %q, want %q", inspection.Schema, ClusterInspectionSchema)
	}
	if !validSHA256(inspection.RecordIdentity) || !validSHA256(inspection.Identity) || !validSHA256(inspection.Static.TargetSHA256) || !validSHA256(inspection.Static.PlatformBundleSHA256) {
		return errors.New("cluster inspection contains an invalid identity")
	}
	models, err := currentClusterModelIdentities()
	if err != nil {
		return err
	}
	if inspection.Models != models {
		return errors.New("cluster inspection model identities do not match this implementation")
	}
	if err := validateLimits(inspection.Limits); err != nil {
		return err
	}
	if inspection.Backend != BackendInProcess && inspection.Backend != BackendProcess || inspection.Backend == BackendInProcess && inspection.Fidelity != FidelitySimulationModel || inspection.Backend == BackendProcess && inspection.Fidelity != FidelitySimulationModel && inspection.Fidelity != FidelityHardIsolation {
		return errors.New("cluster inspection backend or fidelity is unsupported")
	}
	if err := validateInspectionCounts(inspection.Counts, inspection.Limits); err != nil {
		return err
	}
	for _, identity := range []string{
		inspection.Tapes.FaultPlanSHA256, inspection.Tapes.NetworkSHA256, inspection.Tapes.VolumeSHA256,
		inspection.Terminal.NetworkSHA256, inspection.Terminal.VolumeSHA256,
	} {
		if !validSHA256(identity) {
			return errors.New("cluster inspection tape or terminal identity is invalid")
		}
	}
	for index, failure := range inspection.OracleFailures {
		if !oracleNamePattern.MatchString(failure.Name) || !validSHA256(failure.Identity) || index != 0 && inspection.OracleFailures[index-1].Name >= failure.Name {
			return errors.New("cluster inspection oracle failures are invalid or unordered")
		}
	}
	want, err := clusterInspectionIdentity(inspection)
	if err != nil {
		return err
	}
	if inspection.Identity != want {
		return errors.New("cluster inspection identity does not match its contents")
	}
	return nil
}

func validateInspectionCounts(counts ClusterInspectionCounts, limits Limits) error {
	checks := []struct {
		name     string
		required uint64
		maximum  uint64
	}{
		{"nodes", counts.Nodes, limits.Nodes},
		{"scenario_actions", counts.Incarnations, limits.ScenarioActions},
		{"scenario_actions", counts.Lifecycle, limits.ScenarioActions},
		{"fault_actions", counts.Faults, limits.FaultActions},
		{"scenario_decisions", counts.ScenarioDecisions, limits.ScenarioDecisions},
		{"history_operations", counts.HistoryOperations, limits.HistoryOperations},
		{"observations", counts.Observations, limits.Observations},
		{"oracle_results", counts.OracleResults, limits.OracleResults},
		{"network_transitions", counts.NetworkTransitions, limits.NetworkTransitions},
		{"volume_transitions", counts.VolumeTransitions, limits.VolumeTransitions},
	}
	for _, check := range checks {
		if err := checkCapacity(check.name, check.required, check.maximum); err != nil {
			return err
		}
	}
	return nil
}

func clusterInspectionIdentity(inspection ClusterInspection) (string, error) {
	inspection.Identity = ""
	inspection.OracleFailures = append([]ClusterOracleFailure(nil), inspection.OracleFailures...)
	return hashCanonical("gomadv3-cluster-inspection/v1", inspection)
}
