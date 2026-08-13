package testtier

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
		{mode: "test", tiers: []string{"test-builder", "test-runtime", "test-upstream"}, success: "gomadv3 all black-box tiers passed"},
		{mode: "test-builder", tiers: []string{"test-builder"}, success: "gomadv3 builder tier passed"},
		{mode: "test-runtime", tiers: []string{"test-runtime"}, success: "gomadv3 runtime tier passed"},
		{mode: "test-upstream", tiers: []string{"test-upstream"}, success: "gomadv3 upstream-compatibility tier passed"},
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
