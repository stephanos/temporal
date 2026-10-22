package wasi

import (
	"errors"

	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

// ProjectSQLiteCrashCatalogue applies the original reference owner's evidence
// decoder and context admission to actual retained SQLite storage streams.
// It neither opens a guest nor constructs a new enumeration algorithm.
func ProjectSQLiteCrashCatalogue(state, log []byte) (volume.CrashEnumeration, StorageCrashContext, error) {
	refuse := func() (volume.CrashEnumeration, StorageCrashContext, error) {
		return volume.CrashEnumeration{}, StorageCrashContext{}, errors.New("complete original four-state SQLite catalogue required")
	}
	if len(state) > 8<<20 || len(log) > 32<<20 || len(state) < 16 || string(state[:8]) != storageStateMagic || state[8] != 3 {
		return refuse()
	}
	model := &StorageModel{Volumes: []volume.VolumeConfig{{ID: "sqlite", Path: "/volume", CapacityBytes: 4 << 20}},
		Limits: volume.VolumeLimits{PendingOperations: 10000, Transitions: 100000}, Faults: []StorageFault{{Name: "acknowledged-crash", Resource: "/volume/crash-trigger", Operation: "open", Occurrence: 1, Action: "crash-before"}}, SQLiteLockDatabases: []string{"/volume/db"}}
	request := Request{Config: Config{Storage: model, Limits: Limits{OutputBytes: 1 << 20, TranscriptBytes: 32 << 20, Calls: 100000, Descriptors: 128, Files: 1024, FilesystemBytes: 8 << 20, PendingEvents: 4096}},
		StorageCrashLimits: &volume.CrashEnumerationLimits{States: 32, Operations: 8, Depth: 8, Bytes: 4 << 20, Work: 1 << 16}}
	sink := frameSink{request: request, terminal: &resultFrame{Termination: "storage_crash"}, helperEvidence: helperEvidence{storageState: state, storage: log}}
	var result Result
	if err := sink.collectHelperStorage(&result); err != nil {
		return volume.CrashEnumeration{}, StorageCrashContext{}, err
	}
	enumeration, found := result.StorageCrashEnumeration["sqlite"]
	context, hasContext := result.StorageCrashContext["sqlite"]
	if !found || !hasContext || !enumeration.Complete || enumeration.Capacity != "" || enumeration.ChoiceExploration != nil || len(enumeration.States) != 4 || len(context.Operations) > 8 {
		return refuse()
	}
	seen := map[string]bool{}
	var bytes uint64
	for _, state := range enumeration.States {
		if state.Volume != "sqlite" || state.Identity == "" || seen[state.Identity] || state.PendingSHA256 != context.PendingSHA256 || len(state.SelectedOperations) > 8 {
			return refuse()
		}
		seen[state.Identity] = true
		selected := request
		selected.StorageCrashContext = map[string]StorageCrashContext{"sqlite": context}
		selected.StorageCrashSelection = map[string][]uint64{"sqlite": state.SelectedOperations}
		if err := ValidateStorageCrashSelection(selected); err != nil {
			return volume.CrashEnumeration{}, StorageCrashContext{}, err
		}
		bytes += uint64(len(state.Volume) + len(state.PendingSHA256) + len(state.Identity) + 8*len(state.SelectedOperations))
		for _, entry := range state.Entries {
			bytes += uint64(len(entry.Path) + 32 + len(entry.Data))
		}
		if bytes > 4<<20 {
			return refuse()
		}
	}
	return enumeration, context, nil
}
