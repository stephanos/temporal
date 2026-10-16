package cluster

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

func TestWASMInspectionRejectsIncompleteMetadata(t *testing.T) {
	for _, data := range []string{`null`, `{}`, `{"schema":"old","complete":true}`, `{"schema":"gomad.wasm-inspection/v1","complete":true,"unknown":1}`} {
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

func commitInspectionFixture(t *testing.T, metadata *WASMInspectionEvidence) {
	t.Helper()
	metadata.WorldEncoding = WASMInspectionWorldEncoding
	metadata.World = []byte(WASMInspectionWorldHeader)
	for i := range metadata.Boundaries {
		metadata.World = binary.LittleEndian.AppendUint64(metadata.World, sha256.Size)
		digest := sha256.Sum256([]byte(fmt.Sprintf("World boundary %d", i)))
		metadata.World = append(metadata.World, digest[:]...)
		metadata.Boundaries[i].WorldBytes = uint64(len(metadata.World))
		for j := range metadata.Boundaries[i].Cuts {
			cut := &metadata.Boundaries[i].Cuts[j]
			if cut.Witness != nil {
				packInspectionFixtureWitness(cut.Witness)
			}
			if cut.StateEncoding != "" {
				continue
			}
			cut.StateEncoding = WASMInspectionStateEncoding
			for _, state := range []struct {
				kind  string
				bytes *[]byte
			}{{"storage", &cut.StorageState}, {"recovery", &cut.RecoveryState}, {"network", &cut.NetworkState}} {
				digest, err := WASMInspectionStateCommitment(state.kind, *state.bytes)
				require.NoError(t, err)
				*state.bytes = append([]byte{}, digest[:]...)
			}
		}
	}
	metadata.Lifecycle = []byte(WASMInspectionLifecycleHeader)
	var last uint64
	for _, n := range metadata.Incarnations {
		last = max(last, n.TerminalOrder)
	}
	for _, b := range metadata.Boundaries {
		last = max(last, b.Order)
	}
	for order := uint64(1); order <= last; order++ {
		handle, action := metadata.Incarnations[0].Handle, byte(6)
		for _, b := range metadata.Boundaries {
			if b.Order == order {
				handle, action = b.Handle, 2
			}
		}
		metadata.Lifecycle = append(metadata.Lifecycle, action)
		metadata.Lifecycle = binary.LittleEndian.AppendUint32(metadata.Lifecycle, uint32(len(handle.Node)))
		metadata.Lifecycle = append(metadata.Lifecycle, string(handle.Node)...)
		metadata.Lifecycle = binary.LittleEndian.AppendUint64(metadata.Lifecycle, handle.Incarnation)
		for i := range metadata.Boundaries {
			if metadata.Boundaries[i].Order == order {
				metadata.Boundaries[i].LifecycleBytes = uint64(len(metadata.Lifecycle))
			}
		}
	}
}

func packInspectionFixtureWitness(w *WASMRuntimeWitness) {
	w.RuntimeState = make([]byte, 128)
	put := func(at int, value uint64) { binary.BigEndian.PutUint64(w.RuntimeState[at:], value) }
	put(0, w.Choices+w.Observations)
	put(16, w.Allocations)
	binary.BigEndian.PutUint32(w.RuntimeState[24:], uint32(w.GCCycles))
	w.RuntimeState[28] = byte(w.GCPhase)
	binary.BigEndian.PutUint32(w.RuntimeState[32:], uint32(w.RunQueue))
	for i, value := range w.SeededDrawCounters {
		put(40+i*8, value)
	}
	put(96, w.AllocatedBytes)
	put(104, w.Choices)
	put(112, w.Goroutines)
	put(120, w.GlobalRunQueue)
}

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
		choiceBytes, err := choice.TracePayloadBytes(trace)
		require.NoError(t, err)
		metadata.Boundaries = append(metadata.Boundaries, WASMInspectionBoundary{Handle: handle, Decision: uint64(i), Trace: trace, Order: uint64(3 + i), TimeNanos: uint64(i), WorldBytes: 2, LifecycleBytes: 3, Cuts: []WASMInspectionCut{{Handle: handle, ChoiceRecords: trace, WASIBytes: 2, NetworkBytes: 2, StorageBytes: 2, StdoutBytes: 2, StderrBytes: 2, ClockRun: []byte("clock"), StorageState: []byte("state"), RecoveryState: []byte("recovery"), Witness: &WASMRuntimeWitness{Goroutines: 2, RuntimeState: make([]byte, 128)}}}})
		metadata.Boundaries[i].Cuts[0].ChoiceBytes = choiceBytes
		metadata.Boundaries[i].Cuts[0].NetworkState = []byte("network-state")
	}
	digest := func(data []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(data)) }
	execution := &WASMExecutionEvidence{Inspection: metadata, Incarnations: []WASMIncarnationEvidence{{Handle: handle, ChoiceTrace: trace.Bytes, WASISHA256: digest(streams.WASI), NetworkSHA256: digest(streams.Network), StorageSHA256: digest(streams.Storage)}}}
	commitInspectionFixture(t, metadata)
	return metadata, execution
}

func TestWASMInspectionAuthenticatesCoordinatesAndMaterial(t *testing.T) {
	metadata, execution := inspectionFixture(t)
	require.NoError(t, ValidateWASMInspection(metadata, execution))
	for _, test := range []struct {
		name   string
		change func(*WASMInspectionEvidence)
	}{
		{"missing witness", func(m *WASMInspectionEvidence) { m.Boundaries[0].Cuts[0].Witness = nil }},
		{"truncated stream", func(m *WASMInspectionEvidence) { m.Incarnations[0].Streams.WASI = nil }},
		{"changed canonical stream", func(m *WASMInspectionEvidence) { m.Incarnations[0].Streams.Network[0] = 'X' }},
		{"world cut out of bounds", func(m *WASMInspectionEvidence) { m.Boundaries[0].WorldBytes = 999 }},
		{"missing World material", func(m *WASMInspectionEvidence) {
			m.World = nil
			for i := range m.Boundaries {
				m.Boundaries[i].WorldBytes = 0
			}
		}},
		{"missing shared cut", func(m *WASMInspectionEvidence) { m.Boundaries[0].WorldBytes = 0 }},
		{"missing network prefix state", func(m *WASMInspectionEvidence) { m.Boundaries[0].Cuts[0].NetworkState = nil }},
		{"missing storage prefix state", func(m *WASMInspectionEvidence) { m.Boundaries[0].Cuts[0].StorageState = nil }},
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
		{"network state", func(m *WASMInspectionEvidence) { m.Boundaries[1].Cuts[0].NetworkState[0] = 'X' }},
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

func TestWASMInspectionIncludesExecutedNeighborsAndExcludesFutureGuests(t *testing.T) {
	metadata, _ := inspectionFixture(t)
	neighbor := NodeHandle{Node: "b", Incarnation: 1}
	future := NodeHandle{Node: "c", Incarnation: 1}
	metadata.Incarnations = append(metadata.Incarnations, WASMInspectionIncarnation{Handle: neighbor, StartedOrder: 4, TerminalOrder: 9, Streams: WASMInspectionStreams{Stdout: []byte("neighbor")}}, WASMInspectionIncarnation{Handle: future, StartedOrder: 8, TerminalOrder: 10})
	metadata.Boundaries[1].Cuts = append(metadata.Boundaries[1].Cuts, WASMInspectionCut{Handle: neighbor, StdoutBytes: 8, Witness: &WASMRuntimeWitness{Goroutines: 2, RuntimeState: make([]byte, 128)}})
	commitInspectionFixture(t, metadata)
	require.NoError(t, ValidateWASMInspection(metadata, nil))
	for _, test := range []struct {
		name   string
		change func(*WASMInspectionEvidence)
	}{
		{"omit executed neighbor", func(m *WASMInspectionEvidence) { m.Boundaries[1].Cuts = m.Boundaries[1].Cuts[:1] }},
		{"change neighbor output", func(m *WASMInspectionEvidence) { m.Incarnations[1].Streams.Stdout[0] = 'X' }},
		{"fabricate future execution", func(m *WASMInspectionEvidence) {
			m.Boundaries[1].Cuts = append(m.Boundaries[1].Cuts, WASMInspectionCut{Handle: future, Witness: &WASMRuntimeWitness{Goroutines: 1, RuntimeState: make([]byte, 128)}})
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			observed := CloneWASMInspection(metadata)
			test.change(observed)
			require.Error(t, VerifyWASMInspectionPrefix(metadata, observed, metadata.Boundaries[1]))
		})
	}
}

func TestWASMInspectionRetainsCompleteTerminalCuts(t *testing.T) {
	metadata, _ := inspectionFixture(t)
	prior := NodeHandle{Node: "a", Incarnation: 0}
	prior.Incarnation = 2
	metadata.Incarnations = append(metadata.Incarnations, WASMInspectionIncarnation{Handle: prior, StartedOrder: 0, TerminalOrder: 2, Streams: WASMInspectionStreams{Stdout: []byte("finished")}})
	for i := range metadata.Boundaries {
		metadata.Boundaries[i].Cuts = append(metadata.Boundaries[i].Cuts, WASMInspectionCut{Handle: prior, StdoutBytes: 8, Witness: &WASMRuntimeWitness{Goroutines: 1, RuntimeState: make([]byte, 128)}})
	}
	commitInspectionFixture(t, metadata)
	require.NoError(t, ValidateWASMInspection(metadata, nil))
	metadata.Boundaries[0].Cuts[1].StdoutBytes = 7
	metadata.Boundaries[1].Cuts[1].StdoutBytes = 7
	require.Error(t, ValidateWASMInspection(metadata, nil), "a completed incarnation must contribute its full terminal stream")
}

func TestWASMInspectionSealedPrefixHasLiveIncarnationsAndExactRegistry(t *testing.T) {
	metadata, _ := inspectionFixture(t)
	observed := CloneWASMInspection(metadata)
	observed.Complete = false
	observed.Incarnations[0].TerminalOrder = 0
	require.NoError(t, VerifyWASMInspectionPrefix(metadata, observed, metadata.Boundaries[1]))
	observed.Incarnations[0].Registry.Entries = []WASMInspectionRegistration{{Kind: "probe", Name: "unexpected", AvailableOrder: 1}}
	require.Error(t, VerifyWASMInspectionPrefix(metadata, observed, metadata.Boundaries[1]))
}

func TestWASMInspectionCapacityRefusalPreservesOrdinaryEvidence(t *testing.T) {
	metadata, execution := inspectionFixture(t)
	metadata.Complete = false
	metadata.Boundaries = nil
	metadata.World = nil
	metadata.Incarnations[0].Streams = WASMInspectionStreams{}
	require.NoError(t, ValidateWASMInspection(metadata, execution))
	_, err := ResolveWASMInspection(execution, WASMInspectionRequest{Stop: WASMInspectionStop{Kind: "choice"}}, false)
	require.ErrorIs(t, err, ErrWASMInspectionUnsupported)
}

func TestWASMInspectionStrictWitnessDecodeAndClone(t *testing.T) {
	metadata, _ := inspectionFixture(t)
	encoded, err := json.Marshal(metadata)
	require.NoError(t, err)
	_, err = DecodeWASMInspection([]byte(strings.Replace(string(encoded), `"boundary_frames":`, `"unknown":1,"boundary_frames":`, 1)))
	require.Error(t, err)
	detached := CloneWASMInspection(metadata)
	expectedStorage := append([]byte{}, metadata.Boundaries[0].Cuts[0].StorageState...)
	detached.Boundaries[0].Cuts[0].Witness.RuntimeState[0] = 99
	detached.Boundaries[0].Cuts[0].StorageState[0] = 'X'
	require.Equal(t, byte(0), metadata.Boundaries[0].Cuts[0].Witness.RuntimeState[0])
	require.Equal(t, expectedStorage, metadata.Boundaries[0].Cuts[0].StorageState)
}

func TestWASMInspectionOutputMaterialMatchesRetainedOutput(t *testing.T) {
	metadata, _ := inspectionFixture(t)
	output := []OutputObservation{{Handle: metadata.Incarnations[0].Handle, Stream: OutputStdout, Bytes: []byte("out"), TotalBytes: 3, RetainedBytes: 3}, {Handle: metadata.Incarnations[0].Handle, Stream: OutputStderr, Bytes: []byte("err"), TotalBytes: 3, RetainedBytes: 3}}
	require.NoError(t, validateInspectionOutputs(metadata, output))
	metadata.Incarnations[0].Streams.Stdout[0] = 'X'
	require.Error(t, validateInspectionOutputs(metadata, output))
	metadata.Incarnations[0].Streams.Stdout[0] = 'o'
	require.Error(t, validateInspectionOutputs(metadata, output[:1]))
}
