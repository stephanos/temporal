package upgrade_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/backends/native/upgrade"
)

func TestPinImpactCanonicalJSONFacadeRemainsSemanticAndRejectsInvalidStrings(t *testing.T) {
	report := upgrade.PinImpact{
		Schema: "gomad3.pin-impact/v1", CandidateSHA256: "sha256:candidate", BaselineSHA256: "sha256:baseline",
		Pins: []upgrade.PinResult{{Class: "adapter", ID: "example.com/adapter", Platforms: []string{"darwin/arm64"}, Status: "invalidated", Reason: "version changed"}},
	}
	encoded, err := report.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var decoded upgrade.PinImpact
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, report) {
		t.Fatalf("decoded report = %#v, want %#v", decoded, report)
	}

	report.Pins[0].Reason = "invalid\xff"
	if _, err := report.CanonicalJSON(); err == nil || !strings.Contains(err.Error(), "valid UTF-8") {
		t.Fatalf("CanonicalJSON() error = %v, want invalid UTF-8", err)
	}
}
