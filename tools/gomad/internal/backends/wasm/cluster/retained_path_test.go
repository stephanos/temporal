package simulation

import (
	"encoding/json"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
)

func TestRetainedDecoderAcceptsSharedCampaignPayloadPath(t *testing.T) {
	data, err := json.Marshal(retainedExecution{Schema: RetainedSchema, ModuleSHA256: strings.Repeat("a", 64)})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"backend/simulation.json", "backend/evidence.bin", "backend/unrecognized.json"} {
		payload := target.BackendPayload{Reference: record.BackendPayload{Schema: RetainedSchema, File: path, SHA256: record.HashBytes(data), Bytes: record.Uint64String(len(data))}, Data: data}
		_, err := decodeRetained(payload)
		if (err == nil) != (path != "backend/unrecognized.json") {
			t.Fatalf("path=%s error=%v", path, err)
		}
	}
}
