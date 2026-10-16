package inspect

import (
	"encoding/json"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/runner/internal/campaign"
)

func TestInspectionRetainsDetachedBackendPayloadIdentities(t *testing.T) {
	metadata := &record.BackendMetadata{Name: "wasm", ReplayMode: record.ReplayExact,
		Provenance: record.BackendPayload{Schema: "gomad-wasm.provenance/v1", File: "backend/provenance.json", SHA256: record.HashBytes([]byte("scenario and prepared inputs")), Bytes: 28},
		Evidence:   &record.BackendPayload{Schema: "gomad3.wasm-simulation-retained/v2", File: "backend/evidence.bin", SHA256: record.HashBytes([]byte("per-incarnation evidence")), Bytes: 24},
	}
	target := record.Target{Backend: metadata}
	plan := projectCampaignPlan(campaign.OpenedPortablePlan{Plan: campaign.CampaignPlan{Prepared: campaign.PreparedTargetPlan{Target: target}}})
	artifact := projectArtifact(record.ExecutionRecord{Target: target}, "")
	for _, value := range []TargetReport{plan.Target, artifact.Target} {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		var report struct {
			Backend *record.BackendMetadata `json:"backend"`
		}
		if err := json.Unmarshal(encoded, &report); err != nil {
			t.Fatal(err)
		}
		if report.Backend == nil || report.Backend.Name != "wasm" || report.Backend.Provenance != metadata.Provenance || report.Backend.Evidence == nil || *report.Backend.Evidence != *metadata.Evidence {
			t.Fatalf("inspection lost retained backend identities: %s", encoded)
		}
	}
}
