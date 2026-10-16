package conformance

import (
	"slices"
	"testing"
)

func TestResolveReturnsOrderedTiersAndSpecificSuccessMessage(t *testing.T) {
	for _, test := range []struct {
		mode    string
		tiers   []string
		success string
	}{
		{mode: "test", tiers: []string{"test-builder", "test-live-capability", "test-runtime", "test-upstream"}, success: "gomad all black-box tiers passed"},
		{mode: "test-builder", tiers: []string{"test-builder"}, success: "gomad builder tier passed"},
		{mode: "test-runtime", tiers: []string{"test-runtime"}, success: "gomad runtime tier passed"},
		{mode: "test-upstream", tiers: []string{"test-upstream"}, success: "gomad upstream-compatibility tier passed"},
		{mode: "test-interception", tiers: []string{"test-interception"}, success: "gomad interception tier passed"},
		{mode: "test-live-capability", tiers: []string{"test-live-capability"}, success: "gomad live-capability tier passed"},
	} {
		t.Run(test.mode, func(t *testing.T) {
			mode, err := Resolve(test.mode)
			if err != nil || !slices.Equal(mode.Tiers, test.tiers) || mode.Success != test.success {
				t.Fatalf("Resolve(%q) = %#v, %v", test.mode, mode, err)
			}
		})
	}
}

func TestResolveRejectsUnknownMode(t *testing.T) {
	if _, err := Resolve("unknown"); err == nil {
		t.Fatal("Resolve(unknown) succeeded")
	}
}
