package simulation

import (
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
)

// Only crash-before is a storage crash boundary; validateStorageModel refuses
// the former crash-after action before execution, so it never binds a crash.
func TestStorageCrashBoundOnlyByCrashBefore(t *testing.T) {
	for action, want := range map[string]bool{"crash-before": true, "crash-after": false, "io-error": false} {
		request := wasi.Request{Config: wasi.Config{Storage: &wasi.StorageModel{Faults: []wasi.StorageFault{{Action: action}}}}}
		if got := storageCrashBound(request); got != want {
			t.Fatalf("%s bound = %v, want %v", action, got, want)
		}
	}
	if storageCrashBound(wasi.Request{}) {
		t.Fatal("storage-free request bound a storage crash")
	}
}
