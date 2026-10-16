package cluster

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

func TestWASMInspectionRejectsIncompleteMetadata(t *testing.T) {
	for _, data := range []string{`null`, `{}`, `{"schema":"old","complete":true}`, `{"schema":"gomad3.wasm-inspection/v1","complete":true,"unknown":1}`} {
		_, err := DecodeWASMInspection([]byte(data))
		require.Error(t, err)
	}
}

func TestWASMInspectionTargetAndStopResolution(t *testing.T) {
	a := NodeHandle{Node: "a", Incarnation: 1}
	b := NodeHandle{Node: "a", Incarnation: 2}
	metadata := &WASMInspectionEvidence{Schema: WASMInspectionSchema, Complete: true,
		Incarnations: []WASMInspectionIncarnation{{Handle: a}, {Handle: b, FailureOrder: uint64Pointer(9)}},
		Boundaries:   []WASMInspectionBoundary{{Handle: a, Decision: 0, Trace: 1, Order: 2, TimeNanos: 4}, {Handle: b, Decision: 0, Trace: 3, Order: 7, TimeNanos: 4}, {Handle: b, Decision: 1, Trace: 5, Order: 8, TimeNanos: 4}},
	}
	for _, test := range []struct {
		name    string
		request WASMInspectionRequest
		order   uint64
		invalid bool
	}{
		{"ambiguous", WASMInspectionRequest{Stop: WASMInspectionStop{Kind: "choice"}}, 0, true},
		{"partial target", WASMInspectionRequest{Target: NodeHandle{Node: "a"}, Stop: WASMInspectionStop{Kind: "choice"}}, 0, true},
		{"unknown", WASMInspectionRequest{Target: NodeHandle{Node: "b", Incarnation: 1}, Stop: WASMInspectionStop{Kind: "choice"}}, 0, true},
		{"local ordinal", WASMInspectionRequest{Target: b, Stop: WASMInspectionStop{Kind: "choice", Value: 0}}, 7, false},
		{"equal time", WASMInspectionRequest{Target: b, Stop: WASMInspectionStop{Kind: "time", Value: 4}}, 7, false},
		{"failure", WASMInspectionRequest{Target: b, Stop: WASMInspectionStop{Kind: "failure"}}, 8, false},
		{"unreachable", WASMInspectionRequest{Target: b, Stop: WASMInspectionStop{Kind: "choice", Value: 2}}, 0, true},
		{"past time", WASMInspectionRequest{Target: b, Stop: WASMInspectionStop{Kind: "time", Value: 5}}, 0, true},
		{"missing failure", WASMInspectionRequest{Target: a, Stop: WASMInspectionStop{Kind: "failure"}}, 0, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			resolved, err := resolveWASMInspection(metadata, test.request, false)
			if test.invalid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.order, resolved.Boundary.Order)
		})
	}
}

func TestWASMInspectionRegistryAvailabilityAndNames(t *testing.T) {
	handle := NodeHandle{Node: "a", Incarnation: 1}
	metadata := &WASMInspectionEvidence{Incarnations: []WASMInspectionIncarnation{{Handle: handle, Registry: WASMInspectionRegistry{Closed: true, ClosedOrder: 8, Entries: []WASMInspectionRegistration{{ID: 0, Kind: "dump", Name: "state", AvailableOrder: 5}, {ID: 1, Kind: "probe", Name: "queue.push", AvailableOrder: 1}}}}}, Boundaries: []WASMInspectionBoundary{{Handle: handle, Order: 3}}}
	for _, test := range []struct {
		dumps, probes []string
		invalid       bool
	}{
		{[]string{"stacks", "heap"}, nil, false}, {[]string{"state"}, nil, true}, {[]string{"unknown"}, nil, true}, {nil, []string{"queue.*"}, false}, {nil, []string{"absent*"}, true}, {nil, []string{"["}, true},
	} {
		_, err := resolveWASMInspection(metadata, WASMInspectionRequest{Stop: WASMInspectionStop{Kind: "choice"}, Dumps: test.dumps, Probes: test.probes}, false)
		if test.invalid {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
	encoded, err := json.Marshal(metadata)
	require.NoError(t, err)
	var detached WASMInspectionEvidence
	require.NoError(t, json.Unmarshal(encoded, &detached))
	metadata.Incarnations[0].Registry.Entries[0].Name = "changed"
	require.Equal(t, "state", detached.Incarnations[0].Registry.Entries[0].Name)
}

func uint64Pointer(n uint64) *uint64 { return &n }

func inspectionFixture(t *testing.T) (*WASMInspectionEvidence, *WASMExecutionEvidence) {
	t.Helper()
	handle := NodeHandle{Node: "a", Incarnation: 1}
	a, b := sha256.Sum256([]byte("a")), sha256.Sum256([]byte("b"))
	first, err := choice.CanonicalDecision(0, choice.KindRunnable, 10, false, [][32]byte{a, b}, a, 0)
	require.NoError(t, err)
	second, err := choice.CanonicalDecision(2, choice.KindRunnable, 20, false, [][32]byte{a, b}, b, 0)
	require.NoError(t, err)
	ready, err := choice.NewReadiness(1, 0)
	require.NoError(t, err)
	trace, err := choice.BuildTrace([]choice.Record{first.Record(), {Ordinal: 1, Kind: choice.KindSelectResult, Flags: choice.FlagObservation, Alternatives: 1, Data: 1, SiteOffset: 15, Origin: 1, Readiness: ready}, second.Record()}, choice.TerminalComplete)
	require.NoError(t, err)
	streams := WASMInspectionStreams{WASI: []byte("wasi"), Network: []byte("network"), Storage: []byte("storage"), Stdout: []byte("out"), Stderr: []byte("err")}
	metadata := &WASMInspectionEvidence{Schema: WASMInspectionSchema, Complete: true, World: []byte("world"), Lifecycle: []byte("lifecycle"), Incarnations: []WASMInspectionIncarnation{{Handle: handle, StartedOrder: 1, TerminalOrder: 10, Registry: WASMInspectionRegistry{Closed: true, ClosedOrder: 2}, Streams: streams}}}
	metadata.Incarnations[0].Streams.Choice = trace.Bytes
	for i, trace := range []uint64{0, 2} {
		metadata.Boundaries = append(metadata.Boundaries, WASMInspectionBoundary{Handle: handle, Decision: uint64(i), Trace: trace, Order: uint64(3 + i), TimeNanos: uint64(i), WorldBytes: 2, LifecycleBytes: 3, Cuts: []WASMInspectionCut{{Handle: handle, ChoiceRecords: trace, WASIBytes: 2, NetworkBytes: 2, StorageBytes: 2, StdoutBytes: 2, StderrBytes: 2, ClockRun: []byte("clock"), StorageState: []byte("state"), RecoveryState: []byte("recovery"), Witness: &WASMRuntimeWitness{Goroutines: 2, RuntimeState: make([]byte, 128)}}}})
	}
	digest := func(data []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(data)) }
	execution := &WASMExecutionEvidence{Inspection: metadata, Incarnations: []WASMIncarnationEvidence{{Handle: handle, ChoiceTrace: trace.Bytes, WASISHA256: digest(streams.WASI), NetworkSHA256: digest(streams.Network), StorageSHA256: digest(streams.Storage)}}}
	return metadata, execution
}

func TestWASMInspectionAuthenticatesCoordinatesAndMaterial(t *testing.T) {
	metadata, execution := inspectionFixture(t)
	require.NoError(t, ValidateWASMInspection(metadata, execution))
	for _, test := range []struct {
		name   string
		change func(*WASMInspectionEvidence)
	}{
		{"trace observation is not decision", func(m *WASMInspectionEvidence) { m.Boundaries[1].Trace = 1; m.Boundaries[1].Cuts[0].ChoiceRecords = 1 }},
		{"missing boundary", func(m *WASMInspectionEvidence) { m.Boundaries = m.Boundaries[:1] }},
		{"missing witness", func(m *WASMInspectionEvidence) { m.Boundaries[0].Cuts[0].Witness = nil }},
		{"truncated stream", func(m *WASMInspectionEvidence) { m.Incarnations[0].Streams.WASI = nil }},
		{"changed canonical stream", func(m *WASMInspectionEvidence) { m.Incarnations[0].Streams.Network[0] = 'X' }},
		{"world cut out of bounds", func(m *WASMInspectionEvidence) { m.Boundaries[0].WorldBytes = 999 }},
		{"duplicate registration", func(m *WASMInspectionEvidence) {
			m.Incarnations[0].Registry.Entries = []WASMInspectionRegistration{{Kind: "probe", Name: "same", AvailableOrder: 1}, {ID: 1, Kind: "dump", Name: "same", AvailableOrder: 1}}
		}},
		{"late registration", func(m *WASMInspectionEvidence) {
			m.Incarnations[0].Registry.Entries = []WASMInspectionRegistration{{Kind: "probe", Name: "late", AvailableOrder: 5}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			m := CloneWASMInspection(metadata)
			test.change(m)
			require.Error(t, ValidateWASMInspection(m, execution))
		})
	}
}

func TestWASMInspectionPrefixVerificationRejectsPerturbations(t *testing.T) {
	metadata, _ := inspectionFixture(t)
	for _, test := range []struct {
		name   string
		change func(*WASMInspectionEvidence)
	}{
		{"none", func(*WASMInspectionEvidence) {}},
		{"allocation without GC", func(m *WASMInspectionEvidence) { m.Boundaries[1].Cuts[0].Witness.Allocations++ }},
		{"output", func(m *WASMInspectionEvidence) { m.Incarnations[0].Streams.Stdout[0] = 'X' }},
		{"choice", func(m *WASMInspectionEvidence) { m.Incarnations[0].Streams.Choice[10] ^= 1 }},
		{"world order", func(m *WASMInspectionEvidence) { m.World[0] = 'X' }},
		{"lifecycle", func(m *WASMInspectionEvidence) { m.Lifecycle[0] = 'X' }},
		{"folded clock", func(m *WASMInspectionEvidence) { m.Boundaries[1].Cuts[0].ClockRun[0] = 'X' }},
		{"storage", func(m *WASMInspectionEvidence) { m.Boundaries[1].Cuts[0].StorageState[0] = 'X' }},
		{"recovery", func(m *WASMInspectionEvidence) { m.Boundaries[1].Cuts[0].RecoveryState[0] = 'X' }},
		{"not reached", func(m *WASMInspectionEvidence) { m.Boundaries = m.Boundaries[:1] }},
	} {
		t.Run(test.name, func(t *testing.T) {
			observed := CloneWASMInspection(metadata)
			test.change(observed)
			err := VerifyWASMInspectionPrefix(metadata, observed, metadata.Boundaries[1])
			if test.name == "none" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
