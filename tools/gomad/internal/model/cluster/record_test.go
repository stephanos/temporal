package cluster

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/model/cluster/internal/wire"
)

func TestClusterRecordCanonicalRoundTripIncludesLimitsAndLimitations(t *testing.T) {
	limits := DefaultLimits()
	limits.NetworkConnectionBytes = 123
	limits.NetworkStallNanos = 456
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

func TestClusterRecordRejectsInvalidNetworkLimitsBeforeIdentity(t *testing.T) {
	spec := validSpec()
	specSHA256, err := hashSpec(spec)
	require.NoError(t, err)
	record, err := buildClusterRecord(spec, specSHA256, Result{Outcome: OutcomeCompleted})
	require.NoError(t, err)
	tests := []struct {
		name   string
		mutate func(*Limits)
	}{
		{name: "network_connection_bytes", mutate: func(limits *Limits) { limits.NetworkConnectionBytes = 0 }},
		{name: "network_connection_bytes", mutate: func(limits *Limits) { limits.NetworkConnectionBytes = (1 << 30) + 1 }},
		{name: "network_stall_nanos", mutate: func(limits *Limits) { limits.NetworkStallNanos = 0 }},
		{name: "network_stall_nanos", mutate: func(limits *Limits) { limits.NetworkStallNanos = math.MaxInt64 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invalid := record
			test.mutate(&invalid.Limits)
			_, err := EncodeClusterRecord(invalid)
			require.ErrorContains(t, err, test.name)
			_, err = ReplayPlanFor(invalid)
			require.ErrorContains(t, err, test.name)
			encoded, err := json.Marshal(invalid)
			require.NoError(t, err)
			_, err = DecodeClusterRecord(encoded)
			require.ErrorContains(t, err, test.name)
		})
	}
}

func TestClusterRecordRejectsOmittedNetworkLimits(t *testing.T) {
	spec := validSpec()
	specSHA256, err := hashSpec(spec)
	require.NoError(t, err)
	record, err := buildClusterRecord(spec, specSHA256, Result{Outcome: OutcomeCompleted})
	require.NoError(t, err)
	encoded, err := EncodeClusterRecord(record)
	require.NoError(t, err)
	for _, field := range []struct {
		name   string
		member string
	}{
		{name: "network_connection_bytes", member: `"network_connection_bytes":4194304,`},
		{name: "network_stall_nanos", member: `"network_stall_nanos":10000000000,`},
	} {
		t.Run(field.name, func(t *testing.T) {
			omitted := strings.Replace(string(encoded), field.member, "", 1)
			_, err := DecodeClusterRecord([]byte(omitted))
			require.ErrorContains(t, err, field.name)
		})
	}
}

func TestClusterRecordReplayBindsNetworkLimits(t *testing.T) {
	spec := validSpec()
	spec.Limits.NetworkConnectionBytes = 123
	spec.Limits.NetworkStallNanos = 456
	specSHA256, err := hashSpec(spec)
	require.NoError(t, err)
	record, err := buildClusterRecord(spec, specSHA256, Result{Outcome: OutcomeCompleted})
	require.NoError(t, err)
	plan, err := ReplayPlanFor(record)
	require.NoError(t, err)
	spec.Replay = &plan
	require.NoError(t, ValidateSpec(spec))
	for _, mutate := range []func(*Limits){
		func(limits *Limits) { limits.NetworkConnectionBytes++ },
		func(limits *Limits) { limits.NetworkStallNanos++ },
	} {
		changed := cloneSpec(spec)
		mutate(&changed.Limits)
		_, err := newInProcessCluster(changed)
		require.ErrorContains(t, err, "specification")
	}
}

func TestClusterRecordNetworkEndpointOccupancy(t *testing.T) {
	for _, test := range []struct {
		name           string
		client, server uint64
		reset          bool
		want           string
	}{
		{"both-endpoints-at-bound", 3, 2, false, ""},
		{"client-direction-full", 4, 0, false, "network_connection_bytes"},
		{"server-direction-full", 0, 4, false, "network_connection_bytes"},
		{"combined-global-full", 3, 3, false, "network_bytes"},
		{"unsigned-overflow", math.MaxUint64, 0, false, "network_connection_bytes"},
		{"reset-retains-client-bytes", 1, 0, true, "retained bytes"},
		{"reset-retains-server-bytes", 0, 1, true, "retained bytes"},
		{"reset-releases-both-endpoints", 0, 0, true, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			spec := validSpec()
			spec.Limits.NetworkConnectionBytes = 3
			spec.Limits.NetworkBytes = 5
			network := emptyNetworkRecord()
			fixture := fmt.Sprintf(`{"identity":1,"client":{"node":"client","incarnation":1,"address":"10.0.0.2","port":40000},"server":{"node":"server","incarnation":1,"address":"10.0.0.1","port":7233},"closed":%t,"reset":%t,"client_bytes":%d,"server_bytes":%d}`, test.reset, test.reset, test.client, test.server)
			var connection NetworkConnectionSnapshot
			require.NoError(t, json.Unmarshal([]byte(fixture), &connection))
			network.Snapshot.Connections = []NetworkConnectionSnapshot{connection}
			network.Snapshot.NextConnection = 1
			var err error
			network.Snapshot.Identity, err = networkSnapshotIdentity(network.Snapshot)
			require.NoError(t, err)
			specIdentity, err := hashSpec(spec)
			require.NoError(t, err)
			record, err := buildClusterRecord(spec, specIdentity, Result{Outcome: OutcomeCompleted, Network: network})
			require.NoError(t, err)
			_, err = EncodeClusterRecord(record)
			if test.want == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.want)
			}
		})
	}
}

func TestClusterRecordNetworkOccupancyAcrossConnections(t *testing.T) {
	spec := validSpec()
	spec.Limits.NetworkConnectionBytes = 3
	spec.Limits.NetworkBytes = 9
	network := emptyNetworkRecord()
	network.Snapshot.Connections = []NetworkConnectionSnapshot{
		{Identity: 1, Client: NetworkEndpoint{Node: "client", Incarnation: 1, Address: "10.0.0.2", Port: 40000}, Server: NetworkEndpoint{Node: "server", Incarnation: 1, Address: "10.0.0.1", Port: 7233}, ClientBytes: 3, ServerBytes: 2},
		{Identity: 2, Client: NetworkEndpoint{Node: "client", Incarnation: 1, Address: "10.0.0.2", Port: 40001}, Server: NetworkEndpoint{Node: "server", Incarnation: 1, Address: "10.0.0.1", Port: 7233}, ClientBytes: 3, ServerBytes: 2},
	}
	network.Snapshot.NextConnection = 2
	var err error
	network.Snapshot.Identity, err = networkSnapshotIdentity(network.Snapshot)
	require.NoError(t, err)
	specIdentity, err := hashSpec(spec)
	require.NoError(t, err)
	record, err := buildClusterRecord(spec, specIdentity, Result{Outcome: OutcomeCompleted, Network: network})
	require.NoError(t, err)
	_, err = EncodeClusterRecord(record)
	require.ErrorContains(t, err, "network_bytes")
}

func TestClusterRecordNetworkSnapshotRejectsInvalidEndpointPorts(t *testing.T) {
	for _, port := range []uint64{0, 65536, math.MaxUint64} {
		for _, listener := range []bool{false, true} {
			network := emptyNetworkRecord()
			endpoint := NetworkEndpoint{Node: "client", Incarnation: 1, Address: "10.0.0.2", Port: port}
			if listener {
				network.Snapshot.Listeners = []NetworkListenerSnapshot{{Endpoint: endpoint}}
			} else {
				network.Snapshot.Connections = []NetworkConnectionSnapshot{{Identity: 1, Client: endpoint, Server: NetworkEndpoint{Node: "server", Incarnation: 1, Address: "10.0.0.1", Port: 7233}}}
			}
			var err error
			network.Snapshot.Identity, err = networkSnapshotIdentity(network.Snapshot)
			require.NoError(t, err)
			require.ErrorContains(t, validateNetworkRecord(network, DefaultLimits()), "snapshot is invalid")
		}
	}
}

func TestDecodeClusterRecordRejectsRetiredNetworkFields(t *testing.T) {
	spec := validSpec()
	specIdentity, err := hashSpec(spec)
	require.NoError(t, err)
	network := networkRecordForTransition(t, NetworkTransition{Kind: NetworkWrite, Source: NetworkEndpoint{Node: "client", Incarnation: 1, Address: "10.0.0.2", Port: 40000}, Destination: NetworkEndpoint{Node: "server", Incarnation: 1, Address: "10.0.0.1", Port: 7233}, Connection: 7, Bytes: 1, Outcome: NetworkOutcomeHeld})
	record, err := buildClusterRecord(spec, specIdentity, Result{Outcome: OutcomeCompleted, Network: network})
	require.NoError(t, err)
	encoded, err := EncodeClusterRecord(record)
	require.NoError(t, err)
	for _, test := range []struct {
		field, target, replacement string
	}{
		{"delivery", `"kind":"write",`, `"kind":"write","delivery":1,`},
		{"payload_sha256", `"kind":"write",`, `"kind":"write","payload_sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",`},
		{"deliveries", `"connections":null,`, `"connections":null,"deliveries":null,`},
		{"next_delivery", `"next_connection":0,`, `"next_connection":0,"next_delivery":0,`},
		{"network_deliveries", `"network_bytes":`, `"network_deliveries":65536,"network_bytes":`},
	} {
		t.Run(test.field, func(t *testing.T) {
			text := string(encoded)
			for _, retired := range []string{`"deliveries":null,`, `"next_delivery":0,`, `"network_deliveries":65536,`} {
				text = strings.Replace(text, retired, "", 1)
			}
			_, err := DecodeClusterRecord([]byte(strings.Replace(text, test.target, test.replacement, 1)))
			require.ErrorContains(t, err, `unknown field "`+test.field+`"`)
		})
	}
}

func TestClusterRecordRejectsPreviousNetworkFormat(t *testing.T) {
	spec := validSpec()
	specIdentity, err := hashSpec(spec)
	require.NoError(t, err)
	record, err := buildClusterRecord(spec, specIdentity, Result{Outcome: OutcomeCompleted})
	require.NoError(t, err)
	record.Schema = "gomad3.cluster-record/v7"
	_, err = EncodeClusterRecord(record)
	require.ErrorContains(t, err, "schema")
}

func TestClusterRecordNetworkHeldTimeoutVocabulary(t *testing.T) {
	source := NetworkEndpoint{Node: "client", Incarnation: 1, Address: "10.0.0.2", Port: 40000}
	destination := NetworkEndpoint{Node: "server", Incarnation: 1, Address: "10.0.0.1", Port: 7233}
	for _, transition := range []NetworkTransition{
		{Kind: "write", Source: source, Destination: destination, Connection: 7, Bytes: 3, Outcome: "held"},
		{Kind: "write", Source: source, Destination: destination, Connection: 7, Bytes: 1, Outcome: "timeout"},
		{Kind: "reset", Source: source, Destination: destination, Connection: 7, Bytes: 3, Count: 2, Outcome: "timeout"},
		{Kind: "reset", Source: source, Destination: destination, Connection: 7, Outcome: "reset"},
		{Kind: "dial", Source: NetworkEndpoint{Node: source.Node, Incarnation: 1, Address: source.Address}, Destination: NetworkEndpoint{Node: destination.Node, Address: destination.Address, Port: destination.Port}, Outcome: "timeout"},
	} {
		t.Run(string(transition.Kind)+"/"+string(transition.Outcome), func(t *testing.T) {
			record := networkRecordForTransition(t, transition)
			require.NoError(t, validateNetworkRecord(record, DefaultLimits()))
			if transition.Connection != 0 {
				require.Equal(t, "connection\x0000000000000000000007", wire.NetworkTransitionLane(transition))
			}
		})
	}
}

func TestClusterRecordRejectsMalformedHeldTimeoutTransitions(t *testing.T) {
	valid := NetworkTransition{Kind: "reset", Source: NetworkEndpoint{Node: "client", Incarnation: 1, Address: "10.0.0.2", Port: 40000}, Destination: NetworkEndpoint{Node: "server", Incarnation: 1, Address: "10.0.0.1", Port: 7233}, Connection: 7, Outcome: "timeout"}
	for name, mutate := range map[string]func(*NetworkTransition){
		"zero connection":                 func(value *NetworkTransition) { value.Connection = 0 },
		"missing destination incarnation": func(value *NetworkTransition) { value.Destination.Incarnation = 0 },
		"zero destination port":           func(value *NetworkTransition) { value.Destination.Port = 0 },
		"port overflow":                   func(value *NetworkTransition) { value.Source.Port = 65536 },
		"same endpoint node":              func(value *NetworkTransition) { value.Destination.Node = value.Source.Node },
		"held reset":                      func(value *NetworkTransition) { value.Outcome = "held" },
		"unknown outcome":                 func(value *NetworkTransition) { value.Outcome = "timed_out" },
		"unknown kind":                    func(value *NetworkTransition) { value.Kind = "timeout_reset" },
		"retired deliver kind":            func(value *NetworkTransition) { value.Kind = "deliver" },
		"retired partition drop":          func(value *NetworkTransition) { value.Outcome = "partition_drop" },
		"retired stale drop":              func(value *NetworkTransition) { value.Outcome = "stale_drop" },
	} {
		t.Run(name, func(t *testing.T) {
			transition := valid
			mutate(&transition)
			require.Error(t, validateNetworkRecord(networkRecordForTransition(t, transition), DefaultLimits()))
		})
	}
}

func networkRecordForTransition(t *testing.T, transition NetworkTransition) NetworkRecord {
	t.Helper()
	record := emptyNetworkRecord()
	record.Transitions = []NetworkTransition{transition}
	encoded, err := wire.EncodeNetworkTransitions(record.Transitions)
	require.NoError(t, err)
	digest := sha256.Sum256(append([]byte("gomad3-simulation-network-transitions/v3\x00"), encoded...))
	record.Snapshot.TransitionSHA256 = fmt.Sprintf("sha256:%x", digest)
	record.Snapshot.Identity, err = networkSnapshotIdentity(record.Snapshot)
	require.NoError(t, err)
	return record
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
