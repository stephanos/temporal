package pinimpact_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/native/upgrade/pinimpact"
)

func TestEncodePreservesReportSemanticsAndRejectsInvalidStrings(t *testing.T) {
	report := pinimpact.Report{
		Schema: pinimpact.Schema, Invalidated: true, PinnedGoVersion: "go1.27.1",
		Baseline:  pinimpact.ModuleEvidence{GoModSHA256: "sha256:baseline-mod", GoSumSHA256: "sha256:baseline-sum", GoDirective: "1.27.1"},
		Candidate: pinimpact.ModuleEvidence{GoModSHA256: "sha256:candidate-mod", GoSumSHA256: "sha256:candidate-sum", GoDirective: "1.27.1"},
		Platforms: []string{"darwin/arm64"},
		Classes:   []pinimpact.ClassSummary{{Class: pinimpact.ClassAdapter, Total: 1, Invalidated: 1}},
		Pins: []pinimpact.Pin{{
			Class: pinimpact.ClassAdapter, Status: pinimpact.StatusInvalidated, ID: "example.com/adapter", Module: "example.com/adapter",
			PinnedVersion: "v1.0.0", CandidateVersion: "v2.0.0", Platforms: []string{"darwin/arm64"}, Reason: "version changed",
		}},
	}
	encoded, err := pinimpact.Encode(report)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) == 0 || encoded[len(encoded)-1] != '\n' {
		t.Fatalf("Encode() = %q, want one JSON line", encoded)
	}
	var decoded pinimpact.Report
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, report) {
		t.Fatalf("decoded report = %#v, want %#v", decoded, report)
	}

	report.Pins[0].Reason = "invalid\xff"
	if _, err := pinimpact.Encode(report); err == nil || !strings.Contains(err.Error(), "valid UTF-8") {
		t.Fatalf("Encode() error = %v, want invalid UTF-8", err)
	}
}
