package wasi

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

// storageCrashOrderCorpus is shared with the Rust helper: the same
// order-varied cases must produce the same first refusal in both lineages.
type storageCrashOrderCorpus struct {
	Schema            string   `json:"schema"`
	Volumes           []string `json:"volumes"`
	PendingOperations uint64   `json:"pending_operations"`
	Cases             []struct {
		Name      string            `json:"name"`
		Selection []json.RawMessage `json:"selection"`
		Context   []json.RawMessage `json:"context"`
		Want      string            `json:"want"`
	} `json:"cases"`
}

func decodeOrderEntry[T any](t *testing.T, raw json.RawMessage) (string, T) {
	t.Helper()
	var pair []json.RawMessage
	var id string
	var value T
	if err := json.Unmarshal(raw, &pair); err != nil || len(pair) != 2 {
		t.Fatalf("corpus entry %s: %v", raw, err)
	}
	if err := json.Unmarshal(pair[0], &id); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(pair[1], &value); err != nil {
		t.Fatal(err)
	}
	return id, value
}

func TestStorageCrashValidationOrderCorpus(t *testing.T) {
	raw, err := os.ReadFile("../../../testdata/parity/storage-crash-validation-order.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus storageCrashOrderCorpus
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.Schema != "gomad.storage-crash-validation-order/v1" || len(corpus.Cases) == 0 {
		t.Fatalf("unexpected corpus %q with %d cases", corpus.Schema, len(corpus.Cases))
	}
	for _, testCase := range corpus.Cases {
		for _, reversed := range []bool{false, true} {
			request := Request{Config: storageConfig()}
			request.Config.Storage.Volumes = nil
			for _, id := range corpus.Volumes {
				request.Config.Storage.Volumes = append(request.Config.Storage.Volumes, volume.VolumeConfig{ID: id, Path: "/" + id, CapacityBytes: 512})
			}
			request.Config.Storage.Limits.PendingOperations = corpus.PendingOperations
			selection, context := slices.Clone(testCase.Selection), slices.Clone(testCase.Context)
			if reversed {
				slices.Reverse(selection)
				slices.Reverse(context)
			}
			if testCase.Selection != nil {
				request.StorageCrashSelection = map[string][]uint64{}
				for _, entry := range selection {
					id, operations := decodeOrderEntry[[]uint64](t, entry)
					request.StorageCrashSelection[id] = operations
				}
			}
			if testCase.Context != nil {
				request.StorageCrashContext = map[string]StorageCrashContext{}
				for _, entry := range context {
					id, prior := decodeOrderEntry[StorageCrashContext](t, entry)
					request.StorageCrashContext[id] = prior
				}
			}
			if err := ValidateStorageCrashSelection(request); err == nil || err.Error() != testCase.Want {
				t.Fatalf("%s reversed=%v: got %v, want %q", testCase.Name, reversed, err, testCase.Want)
			}
		}
	}
}
