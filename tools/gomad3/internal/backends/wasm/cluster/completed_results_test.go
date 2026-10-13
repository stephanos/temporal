package simulation

import (
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/wasi"
	hostexec "go.temporal.io/server/tools/gomad3/internal/host/exec"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
	gomad3sim "go.temporal.io/server/tools/gomad3/internal/model/cluster"
	"go.temporal.io/server/tools/gomad3/internal/model/volume"
)

func TestCompletedSummariesDetachCountersWithoutEvidencePayloads(t *testing.T) {
	code := uint32(17)
	handle := gomad3sim.NodeHandle{Node: "a", Incarnation: 2}
	r := wasi.Result{ExitCode: &code, Termination: "cancelled", Reaped: true, FuelRemaining: 11, PeakMemoryBytes: 12, VirtualTimeElapsedNanos: 13, AuthorizationToTerminalNanos: 14,
		Transcript: []byte{1, 2}, NetworkTranscript: []byte{1, 2, 3}, StorageTranscript: []byte{1, 2, 3, 4}, ChoiceTrace: choice.Trace{Bytes: []byte{1}}, InterruptedChoiceBytes: []byte{2}, DiagnosticTrace: choice.DiagnosticTrace{Bytes: []byte{1, 2}}, InterruptedDiagnosticBytes: []byte{3},
		Telemetry: &wasi.Telemetry{PrepareNanos: 23}, HelperResourceUsage: &hostexec.ResourceUsage{UserNanos: 29}}
	p := &Provider{latest: &execution{results: map[gomad3sim.NodeHandle]wasi.Result{handle: r}, elapsed: map[gomad3sim.NodeHandle]uint64{handle: 31}}}
	got := p.CompletedSummaries()
	if len(got) != 1 || got[0].Handle != handle || got[0].WallElapsedNanos != 31 || got[0].Termination != "cancelled" || !got[0].Reaped || *got[0].ExitCode != 17 || got[0].FuelRemaining != 11 || got[0].PeakMemoryBytes != 12 || got[0].VirtualTimeElapsedNanos != 13 || got[0].AuthorizationToTerminalNanos != 14 || got[0].WASITraceBytes != 2 || got[0].NetworkTraceBytes != 3 || got[0].StorageTraceBytes != 4 || got[0].ChoiceTraceBytes != 2 || got[0].DiagnosticTraceBytes != 3 {
		t.Fatalf("lost actual terminal/cost counters: %+v", got)
	}
	*got[0].ExitCode, got[0].Telemetry.PrepareNanos, got[0].HelperResourceUsage.UserNanos = 99, 99, 99
	again := p.CompletedSummaries()[0]
	if *again.ExitCode != 17 || again.Telemetry.PrepareNanos != 23 || again.HelperResourceUsage.UserNanos != 29 {
		t.Fatal("summary aliases actual terminal/cost observations")
	}
	p.latest.results[handle] = wasi.Result{Termination: "capacity", Reaped: false}
	again = p.CompletedSummaries()[0]
	if again.Telemetry != nil || again.HelperResourceUsage != nil || again.ExitCode != nil || again.Reaped || again.Termination != "capacity" {
		t.Fatal("summary fabricated terminal evidence")
	}
	for _, field := range reflect.VisibleFields(reflect.TypeFor[IncarnationSummary]()) {
		if field.Type.Kind() == reflect.Slice || field.Type.Kind() == reflect.Map || field.Type == reflect.TypeFor[wasi.Result]() {
			t.Fatalf("summary carries guest evidence payload: %s", field.Name)
		}
	}
}

func TestCompletedResultsDetachMutableObservationAndCostSurfaces(t *testing.T) {
	code := uint32(1)
	handle := gomad3sim.NodeHandle{Node: "a", Incarnation: 1}
	p := &Provider{latest: &execution{results: map[gomad3sim.NodeHandle]wasi.Result{handle: {
		ExitCode: &code, Started: &wasi.Started{Imports: []wasi.Import{{Name: "original"}}, Exports: []wasi.Export{{Name: "original"}}},
		HelperResourceUsage: &hostexec.ResourceUsage{UserNanos: 1}, Telemetry: &wasi.Telemetry{PrepareNanos: 1},
		DiagnosticTrace:         choice.DiagnosticTrace{Bytes: []byte{1}, Records: []choice.DiagnosticRecord{{Ordinal: 1}}},
		ChoiceDivergence:        &choice.Divergence{Expected: &choice.Decision{Ordinal: 1}, Observed: &choice.Decision{Ordinal: 1}},
		StorageSnapshots:        []volume.VolumeSnapshot{{Persisted: []volume.SnapshotEntry{{Path: "original"}}, Volatile: []volume.SnapshotEntry{{Path: "original"}}}},
		StorageOperations:       map[string][]volume.Operation{"volume": {{Dependencies: []uint64{1}}}},
		StorageCrashEnumeration: map[string]volume.CrashEnumeration{"volume": {States: []volume.CrashState{{SelectedOperations: []uint64{1}, Entries: []volume.CrashEntry{{Data: []byte{1}}}}}, ChoiceExploration: &volume.CrashExploration{Cursor: []byte{1}, Seen: []string{"original"}}}},
	}}}}
	actual := p.CompletedResults()[0].Result
	*actual.ExitCode = 9
	actual.Started.Imports[0].Name, actual.Started.Exports[0].Name = "changed", "changed"
	actual.HelperResourceUsage.UserNanos, actual.Telemetry.PrepareNanos = 9, 9
	actual.DiagnosticTrace.Bytes[0], actual.DiagnosticTrace.Records[0].Ordinal = 9, 9
	actual.ChoiceDivergence.Expected.Ordinal, actual.ChoiceDivergence.Observed.Ordinal = 9, 9
	actual.StorageSnapshots[0].Persisted[0].Path, actual.StorageSnapshots[0].Volatile[0].Path = "changed", "changed"
	actual.StorageOperations["volume"][0].Dependencies[0] = 9
	enumeration := actual.StorageCrashEnumeration["volume"]
	enumeration.States[0].SelectedOperations[0], enumeration.States[0].Entries[0].Data[0] = 9, 9
	enumeration.ChoiceExploration.Cursor[0], enumeration.ChoiceExploration.Seen[0] = 9, "changed"
	delete(actual.StorageCrashEnumeration, "volume")
	again := p.CompletedResults()[0].Result
	if *again.ExitCode != 1 || again.Started.Imports[0].Name != "original" || again.Started.Exports[0].Name != "original" || again.HelperResourceUsage.UserNanos != 1 || again.Telemetry.PrepareNanos != 1 || again.DiagnosticTrace.Bytes[0] != 1 || again.DiagnosticTrace.Records[0].Ordinal != 1 || again.ChoiceDivergence.Expected.Ordinal != 1 || again.ChoiceDivergence.Observed.Ordinal != 1 {
		t.Fatal("caller mutated retained terminal/cost/diagnostic observation")
	}
	if again.StorageSnapshots[0].Persisted[0].Path != "original" || again.StorageSnapshots[0].Volatile[0].Path != "original" || again.StorageOperations["volume"][0].Dependencies[0] != 1 {
		t.Fatal("caller mutated retained durable operation/snapshot observation")
	}
	enumeration = again.StorageCrashEnumeration["volume"]
	if len(enumeration.States) != 1 || enumeration.States[0].SelectedOperations[0] != 1 || enumeration.States[0].Entries[0].Data[0] != 1 || enumeration.ChoiceExploration.Cursor[0] != 1 || enumeration.ChoiceExploration.Seen[0] != "original" {
		t.Fatal("caller mutated retained native crash enumeration")
	}
}
