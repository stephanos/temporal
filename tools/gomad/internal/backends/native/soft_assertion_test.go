package native

import (
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
)

func TestSoftAssertionReportOnlyRefusedBeforeNativePreparation(t *testing.T) {
	provider := New(Config{})
	spec := target.Spec{Kind: target.KindGoTest, SoftAssertionReportOnly: true}
	if _, err := provider.Describe(t.Context(), backends.Configuration{Target: spec}); err == nil || !strings.Contains(err.Error(), "does not support soft assertion report-only") {
		t.Fatalf("unsupported native policy admitted: %v", err)
	}
	if _, err := provider.Prepare(t.Context(), spec); err == nil || !strings.Contains(err.Error(), "does not support soft assertion report-only") {
		t.Fatalf("unsupported native preparation admitted: %v", err)
	}
}
