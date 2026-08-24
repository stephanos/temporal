package gomad3sim

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClusterRecordCanonicalRoundTripIncludesLimitsAndLimitations(t *testing.T) {
	limits := DefaultLimits()
	payload := []byte("node output")
	digest := sha256.Sum256(payload)
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Seed:     67,
		Limits:   limits,
		Nodes:    []NodeSpec{{ID: "node", Boot: "boot", Address: "10.0.0.1"}},
	}
	specSHA256, err := hashSpec(spec)
	require.NoError(t, err)
	record, err := buildClusterRecord(spec, specSHA256, Result{
		Outcome: OutcomeCompleted,
		Nodes:   []NodeResult{{Handle: NodeHandle{Node: "node", Incarnation: 1}, State: NodeStateExited}},
		Outputs: []OutputObservation{{
			Handle:        NodeHandle{Node: "node", Incarnation: 1},
			Stream:        OutputStdout,
			Bytes:         payload,
			FullSHA256:    fmt.Sprintf("sha256:%x", digest),
			TotalBytes:    uint64(len(payload)),
			RetainedBytes: uint64(len(payload)),
		}},
	})
	require.NoError(t, err)
	require.Equal(t, limits, record.Limits)
	require.Equal(t, []Limitation{
		LimitationSharedPackageGlobals,
		LimitationRevokedGoroutinesMayRemain,
		LimitationCPULoopsRequireWatchdog,
		LimitationHardIsolationRequiresProcess,
	}, record.Limitations)

	encoded, err := EncodeClusterRecord(record)
	require.NoError(t, err)
	decoded, err := DecodeClusterRecord(encoded)
	require.NoError(t, err)
	require.Equal(t, record, decoded)
}

func TestClusterRecordRejectsInconsistentOutputBeforeIdentity(t *testing.T) {
	payload := []byte("bounded")
	digest := sha256.Sum256(payload)
	spec := Spec{Schema: SpecSchema, Backend: BackendInProcess, Fidelity: FidelitySimulationModel, Limits: DefaultLimits(), Nodes: []NodeSpec{{ID: "node", Boot: "boot", Address: "10.0.0.1"}}}
	specSHA256, err := hashSpec(spec)
	require.NoError(t, err)
	record, err := buildClusterRecord(spec, specSHA256, Result{
		Outcome: OutcomeCompleted,
		Outputs: []OutputObservation{{
			Handle:         NodeHandle{Node: "node", Incarnation: 1},
			Stream:         OutputStderr,
			Bytes:          payload,
			FullSHA256:     fmt.Sprintf("sha256:%x", digest),
			TotalBytes:     uint64(len(payload)),
			RetainedBytes:  uint64(len(payload)),
			DiscardedBytes: 0,
			Truncated:      false,
		}},
	})
	require.NoError(t, err)
	record.Outputs[0].DiscardedBytes = 2
	identity, err := clusterRecordIdentity(record)
	require.NoError(t, err)
	record.Identity = identity
	_, err = EncodeClusterRecord(record)
	require.ErrorContains(t, err, "output metadata")
}

func TestProcessClusterEvidenceRejectsInProcessLeakDiagnostics(t *testing.T) {
	spec := Spec{
		Schema: SpecSchema, Backend: BackendProcess, Fidelity: FidelityHardIsolation, Limits: DefaultLimits(),
		Nodes: []NodeSpec{{ID: "node", Boot: "boot", Address: "10.0.0.1"}},
	}
	specSHA256, err := hashSpec(spec)
	require.NoError(t, err)
	record, err := buildClusterRecord(spec, specSHA256, Result{Outcome: OutcomeCompleted})
	require.NoError(t, err)
	record.Leaks = []LeakDiagnostic{{Handle: NodeHandle{Node: "node", Incarnation: 1}, Kind: LeakRevokedGoroutineMayRemain}}
	record.Identity, err = clusterRecordIdentity(record)
	require.NoError(t, err)
	_, err = EncodeClusterRecord(record)
	require.ErrorContains(t, err, "process backend")

	plan, err := ReplayPlanFor(func() ClusterRecord {
		clean := record
		clean.Leaks = nil
		clean.Identity, err = clusterRecordIdentity(clean)
		require.NoError(t, err)
		return clean
	}())
	require.NoError(t, err)
	plan.Leaks = append([]LeakDiagnostic(nil), record.Leaks...)
	plan.Identity, err = replayPlanIdentity(plan)
	require.NoError(t, err)
	spec.Replay = &plan
	require.ErrorContains(t, ValidateSpec(spec), "process backend")
}

func TestDecodeClusterRecordRejectsUnknownAndTrailingData(t *testing.T) {
	_, err := DecodeClusterRecord([]byte(`{"schema":"gomad3.cluster-record/v2","unknown":true}`))
	require.Error(t, err)
	_, err = DecodeClusterRecord([]byte(`{} {}`))
	require.Error(t, err)
}

func TestClusterRecordRejectsInvalidLifecycleAndTerminalEnums(t *testing.T) {
	spec := Spec{Schema: SpecSchema, Backend: BackendInProcess, Fidelity: FidelitySimulationModel, Limits: DefaultLimits(), Nodes: []NodeSpec{{ID: "node", Boot: "boot", Address: "10.0.0.1"}}}
	specSHA256, err := hashSpec(spec)
	require.NoError(t, err)
	valid, err := buildClusterRecord(spec, specSHA256, Result{Outcome: OutcomeCompleted, Record: ClusterRecord{
		Nodes: []NodeResult{{Handle: NodeHandle{Node: "node", Incarnation: 1}, State: NodeStateExited}},
		Transitions: []LifecycleTransition{
			{Ordinal: 0, Action: LifecycleStart, Handle: NodeHandle{Node: "node", Incarnation: 1}, From: NodeStateDefined, To: NodeStateRunning},
			{Ordinal: 1, Action: LifecycleWait, Handle: NodeHandle{Node: "node", Incarnation: 1}, From: NodeStateRunning, To: NodeStateExited},
		},
	}})
	require.NoError(t, err)
	mutations := map[string]func(*ClusterRecord){
		"outcome":           func(record *ClusterRecord) { record.Outcome = "unknown" },
		"node state":        func(record *ClusterRecord) { record.Nodes[0].State = "unknown" },
		"action":            func(record *ClusterRecord) { record.Transitions[0].Action = "unknown" },
		"transition source": func(record *ClusterRecord) { record.Transitions[0].From = "unknown" },
		"transition target": func(record *ClusterRecord) { record.Transitions[0].To = "unknown" },
		"empty handle":      func(record *ClusterRecord) { record.Transitions[0].Handle = NodeHandle{} },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			record := valid
			record.Nodes = append([]NodeResult(nil), valid.Nodes...)
			record.Transitions = append([]LifecycleTransition(nil), valid.Transitions...)
			mutate(&record)
			identity, err := clusterRecordIdentity(record)
			require.NoError(t, err)
			record.Identity = identity
			_, err = EncodeClusterRecord(record)
			require.Error(t, err)
		})
	}
}

func TestValidateSpecRejectsReplayPlanContentWithStaleIdentity(t *testing.T) {
	spec := validSpec()
	specSHA256, err := hashSpec(spec)
	require.NoError(t, err)
	record, err := buildClusterRecord(spec, specSHA256, Result{Outcome: OutcomeCompleted})
	require.NoError(t, err)
	plan, err := ReplayPlanFor(record)
	require.NoError(t, err)
	plan.Nodes = []NodeResult{{Handle: NodeHandle{Node: "client", Incarnation: 1}, State: NodeStateExited}}
	spec.Replay = &plan
	require.ErrorContains(t, ValidateSpec(spec), "identity")
}

func TestClusterRecordRejectsCorruptVolumeTransitionAndSnapshotIdentities(t *testing.T) {
	volume := VolumeStateSnapshot{
		Volume: "data", Node: "node", Mount: "/data", CapacityBytes: 1024,
		Persisted:     []VolumeEntrySnapshot{{Path: "/", Mode: 0o755, Kind: "directory"}},
		Volatile:      []VolumeEntrySnapshot{{Path: "/", Mode: 0o755, Kind: "directory"}},
		PendingSHA256: volumeTransitionsIdentity(nil),
		NextOperation: 1,
	}
	volume.Identity = volumeStateSnapshotIdentity(volume)
	volumes := VolumeRecord{Snapshot: VolumeSnapshot{
		Volumes: []VolumeStateSnapshot{volume}, TransitionSHA256: volumeTransitionsIdentity(nil),
	}}
	volumes.Snapshot.Identity = volumeRunSnapshotIdentity(volumes.Snapshot)
	spec := Spec{
		Schema: SpecSchema, Backend: BackendInProcess, Fidelity: FidelitySimulationModel, Limits: DefaultLimits(),
		Nodes:   []NodeSpec{{ID: "node", Boot: "boot", Address: "10.0.0.1", Volumes: []VolumeMount{{Volume: "data", Path: "/data"}}}},
		Volumes: []VolumeSpec{{ID: "data", CapacityBytes: 1024}},
	}
	specSHA256, err := hashSpec(spec)
	require.NoError(t, err)
	record, err := buildClusterRecord(spec, specSHA256, Result{Outcome: OutcomeCompleted, Volumes: volumes})
	require.NoError(t, err)
	_, err = EncodeClusterRecord(record)
	require.NoError(t, err)

	corruptTransition := record
	corruptTransition.Volumes.Snapshot.TransitionSHA256 = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	corruptTransition.Volumes.Snapshot.Identity = volumeRunSnapshotIdentity(corruptTransition.Volumes.Snapshot)
	identity, err := clusterRecordIdentity(corruptTransition)
	require.NoError(t, err)
	corruptTransition.Identity = identity
	_, err = EncodeClusterRecord(corruptTransition)
	require.ErrorContains(t, err, "volume transition digest")

	corruptSnapshot := record
	corruptSnapshot.Volumes.Snapshot.Volumes = append([]VolumeStateSnapshot(nil), record.Volumes.Snapshot.Volumes...)
	corruptSnapshot.Volumes.Snapshot.Volumes[0].CapacityBytes++
	corruptSnapshot.Volumes.Snapshot.Identity = volumeRunSnapshotIdentity(corruptSnapshot.Volumes.Snapshot)
	identity, err = clusterRecordIdentity(corruptSnapshot)
	require.NoError(t, err)
	corruptSnapshot.Identity = identity
	_, err = EncodeClusterRecord(corruptSnapshot)
	require.ErrorContains(t, err, "volume state snapshot identity")
}

func TestClusterRecordV5IncludesControllerEvidenceAndStaticIdentities(t *testing.T) {
	plan, err := NewFaultPlan([]FaultAction{{
		ID: "disconnect-client", Kind: FaultDisconnect, From: "client", To: "server",
	}})
	require.NoError(t, err)
	spec := validSpec()
	spec.Faults = &plan
	specSHA256, err := hashSpec(spec)
	require.NoError(t, err)
	fault := FaultRealization{Ordinal: 0, Action: cloneFaultAction(plan.Actions[0])}
	fault.Identity, err = faultRealizationIdentity(fault)
	require.NoError(t, err)
	decision := ScenarioDecision{Ordinal: 0, ID: "send-request", Kind: ScenarioDecisionAction, Occurrence: 1}
	decision.Identity, err = scenarioDecisionIdentity(decision)
	require.NoError(t, err)
	observation := Observation{ID: "response", Kind: "payload", Value: []byte("ok")}
	observation.FullSHA256 = rawSHA256(observation.Value)
	observation.Identity, err = observationIdentity(observation)
	require.NoError(t, err)
	oracle, err := StateInvariant("response.valid", true, []OracleEvidence{{Label: "response", Value: []byte("ok")}}, 1024)
	require.NoError(t, err)
	history := []HistoryOperation{{ID: "request-1", Actor: "client", Kind: "request", Invocation: 1, Completion: 2, Input: []byte("request"), Output: []byte("ok")}}

	record, err := buildClusterRecord(spec, specSHA256, Result{
		Outcome: OutcomeCompleted,
		Faults:  []FaultRealization{fault}, Scenarios: []ScenarioDecision{decision},
		History: history, Observations: []Observation{observation}, Oracles: []OracleResult{oracle},
	})
	require.NoError(t, err)
	require.NotEmpty(t, record.Static.TargetSHA256)
	require.NotEmpty(t, record.Static.PlatformBundleSHA256)
	require.NotEmpty(t, record.Models.RuntimeDomainSHA256)
	require.NotEmpty(t, record.Models.ProcessSHA256)
	require.NotEmpty(t, record.Models.NetworkSHA256)
	require.NotEmpty(t, record.Models.VolumeSHA256)
	require.NotEmpty(t, record.Models.FaultSHA256)
	require.NotEmpty(t, record.Models.ScenarioSHA256)
	require.NotEmpty(t, record.Models.OracleSHA256)
	require.Equal(t, plan, record.FaultPlan)
	require.Equal(t, []FaultRealization{fault}, record.Faults)
	require.Equal(t, []ScenarioDecision{decision}, record.Scenarios)
	require.Equal(t, history, record.History)
	require.Equal(t, []Observation{observation}, record.Observations)
	require.Equal(t, []OracleResult{oracle}, record.Oracles)

	encoded, err := EncodeClusterRecord(record)
	require.NoError(t, err)
	decoded, err := DecodeClusterRecord(encoded)
	require.NoError(t, err)
	require.Equal(t, record, decoded)
	replay, err := ReplayPlanFor(record)
	require.NoError(t, err)
	require.Equal(t, record.FaultPlan, replay.FaultPlan)
	require.Equal(t, record.Faults, replay.Faults)
	require.Equal(t, record.Scenarios, replay.Scenarios)
	require.Equal(t, record.History, replay.History)
	require.Equal(t, record.Observations, replay.Observations)
	require.Equal(t, record.Oracles, replay.Oracles)
}

func TestClusterRecordRejectsCorruptControllerEvidenceBeforeOuterIdentity(t *testing.T) {
	plan, err := NewFaultPlan([]FaultAction{{ID: "disconnect", Kind: FaultDisconnect, From: "client", To: "server"}})
	require.NoError(t, err)
	spec := validSpec()
	spec.Faults = &plan
	specSHA256, err := hashSpec(spec)
	require.NoError(t, err)
	fault := FaultRealization{Action: cloneFaultAction(plan.Actions[0])}
	fault.Identity, err = faultRealizationIdentity(fault)
	require.NoError(t, err)
	record, err := buildClusterRecord(spec, specSHA256, Result{Outcome: OutcomeCompleted, Faults: []FaultRealization{fault}})
	require.NoError(t, err)

	tests := map[string]func(*ClusterRecord){
		"static identity": func(record *ClusterRecord) { record.Static.TargetSHA256 = rawSHA256([]byte("changed")) },
		"model identity":  func(record *ClusterRecord) { record.Models.FaultSHA256 = rawSHA256([]byte("changed")) },
		"fault plan":      func(record *ClusterRecord) { record.FaultPlan.Actions[0].From = "server" },
		"fault":           func(record *ClusterRecord) { record.Faults[0].Action.To = "client" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			corrupt := record
			corrupt.FaultPlan.Actions = cloneFaultActions(record.FaultPlan.Actions)
			corrupt.Faults = cloneFaultRealizations(record.Faults)
			mutate(&corrupt)
			corrupt.Identity, err = clusterRecordIdentity(corrupt)
			require.NoError(t, err)
			_, err = EncodeClusterRecord(corrupt)
			require.Error(t, err)
		})
	}
}

func rawSHA256(value []byte) string {
	digest := sha256.Sum256(value)
	return fmt.Sprintf("sha256:%x", digest)
}
