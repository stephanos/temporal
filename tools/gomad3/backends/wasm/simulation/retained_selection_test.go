package simulation

import (
	"testing"

	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad3/volume"
)

func TestRetainedStorageSelectionProjectionIsDetached(t *testing.T) {
	boots := []retainedBoot{{ID: "selected", Request: wasi.Request{StorageCrashSelection: map[string][]uint64{"volume": {1, 2}, "empty": {}, "nil": nil}, StorageCrashLimits: &volume.CrashEnumerationLimits{}}}, {ID: "durable-default"}}
	projected := detachedStorageCrashSelections(boots)
	if len(projected) != 2 || projected[0].Boot != "selected" || projected[0].Limits == nil || projected[1].Selection != nil || projected[1].Limits != nil {
		t.Fatal("projection omitted sealed default or selected boot")
	}
	projected[0].Selection["volume"][0] = 99
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
