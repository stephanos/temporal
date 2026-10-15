package simulation

import (
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

func TestRetainedStorageSelectionProjectionIsDetached(t *testing.T) {
	boots := []retainedBoot{{ID: "selected", Request: wasi.Request{StorageCrashSelection: map[string][]uint64{"volume": {1, 2}, "empty": {}, "nil": nil}, StorageCrashLimits: &volume.CrashEnumerationLimits{}}}, {ID: "durable-default"}}
	boots[0].Request.StorageCrashContext = map[string]wasi.StorageCrashContext{"volume": {Operations: []volume.Operation{{ID: 2, Dependencies: []uint64{1}}}}}
	projected := detachedStorageCrashSelections(boots)
	if len(projected) != 2 || projected[0].Boot != "selected" || projected[0].Limits == nil || projected[1].Selection != nil || projected[1].Limits != nil {
		t.Fatal("projection omitted sealed default or selected boot")
	}
	projected[0].Selection["volume"][0] = 99
	projected[0].Context["volume"].Operations[0].Dependencies[0] = 99
	if boots[0].Request.StorageCrashContext["volume"].Operations[0].Dependencies[0] != 1 {
		t.Fatal("projection aliases sealed pending context")
	}
	delete(projected[0].Selection, "empty")
	if boots[0].Request.StorageCrashSelection["volume"][0] != 1 || boots[0].Request.StorageCrashSelection["empty"] == nil {
		t.Fatal("projection aliases sealed selections")
	}
	if projected[0].Limits == boots[0].Request.StorageCrashLimits {
		t.Fatal("projection aliases sealed enumeration limits")
	}
	if _, err := RetainedStorageCrashSelections(target.BackendPayload{}); err == nil {
		t.Fatal("projection bypassed shared complete preflight")
	}
}
