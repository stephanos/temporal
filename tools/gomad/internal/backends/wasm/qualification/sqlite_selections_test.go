package qualification

import (
	"testing"

	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	gomad3sim "go.temporal.io/server/tools/gomad/internal/model/cluster"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

func TestSQLiteSelectionCoverageRejectsMissingExtraOrAlteredNativeStates(t *testing.T) {
	states := []volume.CrashState{{Volume: "sqlite", PendingSHA256: "pending", Identity: "absent"}, {Volume: "sqlite", PendingSHA256: "pending", Identity: "created", SelectedOperations: []uint64{1}, Entries: []volume.CrashEntry{{Path: "/marker", Kind: volume.KindFile}}}, {Volume: "sqlite", PendingSHA256: "pending", Identity: "written", SelectedOperations: []uint64{1, 2}, Entries: []volume.CrashEntry{{Path: "/marker", Kind: volume.KindFile, Data: []byte("marker")}}}}
	enumeration := volume.CrashEnumeration{Complete: true, States: states}
	if err := ValidateSQLiteSelectionCoverage(enumeration, states); err != nil {
		t.Fatal(err)
	}
	for _, selected := range [][]volume.CrashState{states[:2], append(append([]volume.CrashState{}, states...), states[0]), {states[0], states[1], states[1]}, {states[0], states[1], {Volume: "sqlite", PendingSHA256: "pending", Identity: "written", SelectedOperations: []uint64{2}, Entries: states[2].Entries}}} {
		if err := ValidateSQLiteSelectionCoverage(enumeration, selected); err == nil {
			t.Fatal("missing, extra, repeated or mutated selected native state accepted")
		}
	}
	enumeration.Complete = false
	if err := ValidateSQLiteSelectionCoverage(enumeration, states); err == nil {
		t.Fatal("partial native enumeration accepted")
	}
	enumeration.Complete = true
	enumeration.Capacity = volume.CrashCapacityStates
	if err := ValidateSQLiteSelectionCoverage(enumeration, states); err == nil {
		t.Fatal("capacity-limited native enumeration accepted")
	}
}

func TestSQLiteSelectedMarkerMatchesVerifiedRecoveredSnapshot(t *testing.T) {
	state := volume.CrashState{Volume: "sqlite", Entries: []volume.CrashEntry{{Path: "/marker", Kind: volume.KindFile, Mode: 0600, ModTime: 7, Data: []byte("marker")}}}
	observed := gomad3sim.ClusterRecord{Volumes: gomad3sim.VolumeRecord{Snapshot: gomad3sim.VolumeSnapshot{Volumes: []gomad3sim.VolumeStateSnapshot{{Volume: "sqlite", Node: "sqlite", Persisted: []gomad3sim.VolumeEntrySnapshot{{Path: "/marker", Kind: "file", Mode: 0600, ModTime: 7, Size: 6, DataSHA256: string(record.HashBytes([]byte("marker")))}}}}}}}
	if err := validateSQLiteRecoveredMarker(observed, state); err != nil {
		t.Fatal(err)
	}
	observed.Volumes.Snapshot.Volumes[0].Persisted[0].DataSHA256 = string(record.HashBytes([]byte("changed")))
	if err := validateSQLiteRecoveredMarker(observed, state); err == nil {
		t.Fatal("changed selected marker survived acceptance")
	}
	state.Entries = nil
	if err := validateSQLiteRecoveredMarker(observed, state); err == nil {
		t.Fatal("unselected marker accepted")
	}
	observed.Volumes.Snapshot.Volumes[0].Persisted = nil
	if err := validateSQLiteRecoveredMarker(observed, state); err != nil {
		t.Fatal(err)
	}
}

func TestSQLiteSelectedMarkerKindAndMetadata(t *testing.T) {
	for _, test := range []struct {
		name  string
		entry volume.CrashEntry
		kind  string
		hash  string
	}{
		{"file", volume.CrashEntry{Path: "/marker", Kind: volume.KindFile, Mode: 0600, ModTime: 7, Data: []byte("marker")}, "file", string(record.HashBytes([]byte("marker")))},
		{"directory", volume.CrashEntry{Path: "/marker", Kind: volume.KindDirectory, Mode: 0700, ModTime: 9}, "directory", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := volume.CrashState{Volume: "sqlite", Entries: []volume.CrashEntry{test.entry}}
			marker := gomad3sim.VolumeEntrySnapshot{Path: "/marker", Kind: test.kind, Mode: test.entry.Mode, ModTime: test.entry.ModTime, Size: uint64(len(test.entry.Data)), DataSHA256: test.hash}
			recordFor := func(marker gomad3sim.VolumeEntrySnapshot) gomad3sim.ClusterRecord {
				return gomad3sim.ClusterRecord{Volumes: gomad3sim.VolumeRecord{Snapshot: gomad3sim.VolumeSnapshot{Volumes: []gomad3sim.VolumeStateSnapshot{{Volume: "sqlite", Node: "sqlite", Persisted: []gomad3sim.VolumeEntrySnapshot{marker}}}}}}
			}
			if err := validateSQLiteRecoveredMarker(recordFor(marker), state); err != nil {
				t.Fatal(err)
			}
			for _, mutation := range []struct {
				name   string
				mutate func(*gomad3sim.VolumeEntrySnapshot)
			}{
				{"kind", func(m *gomad3sim.VolumeEntrySnapshot) {
					m.Kind = "file"
					if test.kind == "file" {
						m.Kind = "directory"
					}
				}},
				{"numeric-kind", func(m *gomad3sim.VolumeEntrySnapshot) { m.Kind = "\x01" }},
				{"mode", func(m *gomad3sim.VolumeEntrySnapshot) { m.Mode++ }},
				{"modtime", func(m *gomad3sim.VolumeEntrySnapshot) { m.ModTime++ }},
				{"size", func(m *gomad3sim.VolumeEntrySnapshot) { m.Size++ }},
				{"hash", func(m *gomad3sim.VolumeEntrySnapshot) { m.DataSHA256 += "changed" }},
				{"absent", func(m *gomad3sim.VolumeEntrySnapshot) { m.Path = "/other" }},
			} {
				t.Run(mutation.name, func(t *testing.T) {
					changed := marker
					mutation.mutate(&changed)
					if err := validateSQLiteRecoveredMarker(recordFor(changed), state); err == nil {
						t.Fatal("changed selected marker accepted")
					}
				})
			}
			state.Entries[0].Kind = volume.Kind(255)
			if err := validateSQLiteRecoveredMarker(recordFor(marker), state); err == nil {
				t.Fatal("unknown selected marker kind accepted")
			}
		})
	}
}

func TestSQLiteSelectedStateRequiresExactSealedAppendBoot(t *testing.T) {
	state := volume.CrashState{Volume: "sqlite", Identity: "native", PendingSHA256: "pending", SelectedOperations: []uint64{1, 2}}
	context := wasi.StorageCrashContext{PendingSHA256: state.PendingSHA256, Operations: []volume.Operation{{ID: 1}, {ID: 2, Dependencies: []uint64{1}}}}
	limits := &volume.CrashEnumerationLimits{States: 32, Operations: 8, Depth: 8, Bytes: 4 << 20, Work: 1 << 16}
	bindings := []simulation.RetainedStorageCrashSelection{{Boot: "setup-live", Limits: limits}, {Boot: "append-crash", Limits: limits, Selection: map[string][]uint64{"sqlite": {1, 2}}, Context: map[string]wasi.StorageCrashContext{"sqlite": context}}, {Boot: "recover", Limits: limits}}
	if err := validateSQLiteBootSelections(bindings, state, context); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func([]simulation.RetainedStorageCrashSelection){
		func(b []simulation.RetainedStorageCrashSelection) { b[1].Selection = nil },
		func(b []simulation.RetainedStorageCrashSelection) {
			b[1].Selection = map[string][]uint64{"sqlite": {2}}
		},
		func(b []simulation.RetainedStorageCrashSelection) {
			b[1].Selection = map[string][]uint64{"foreign": {1, 2}}
		},
		func(b []simulation.RetainedStorageCrashSelection) {
			b[0].Selection = map[string][]uint64{"sqlite": {1, 2}}
		},
		func(b []simulation.RetainedStorageCrashSelection) { b[1].Limits = nil },
		func(b []simulation.RetainedStorageCrashSelection) { b[2].Boot = "foreign" },
		func(b []simulation.RetainedStorageCrashSelection) { b[1].Context = nil },
		func(b []simulation.RetainedStorageCrashSelection) {
			b[1].Context = map[string]wasi.StorageCrashContext{"foreign": context}
		},
		func(b []simulation.RetainedStorageCrashSelection) {
			changed := context
			changed.PendingSHA256 = "foreign"
			b[1].Context = map[string]wasi.StorageCrashContext{"sqlite": changed}
		},
		func(b []simulation.RetainedStorageCrashSelection) {
			changed := context
			changed.Operations = []volume.Operation{{ID: 1}, {ID: 2}}
			b[1].Context = map[string]wasi.StorageCrashContext{"sqlite": changed}
		},
		func(b []simulation.RetainedStorageCrashSelection) { b[0].Context = b[1].Context },
	} {
		copy := append([]simulation.RetainedStorageCrashSelection{}, bindings...)
		mutate(copy)
		if err := validateSQLiteBootSelections(copy, state, context); err == nil {
			t.Fatal("unsupported, missing or changed sealed selection accepted")
		}
	}
	if err := ValidateSQLiteSelectionResults(Result{}, PublicationReceipt{}, nil); err == nil {
		t.Fatal("missing selected-state measured cells accepted")
	}
}
