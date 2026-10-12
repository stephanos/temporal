package runner

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
)

type environmentRecordingProvider struct {
	*boundaryProvider
	environment []string
}

func (p *environmentRecordingProvider) ValidatePrepared(_ target.Spec, _ target.Prepared, environment []string) error {
	p.environment = environment
	return nil
}

func TestCampaignPlanPreparerValidatesWithRecordedEnvironment(t *testing.T) {
	source := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(source, []byte("prepared"), 0o500); err != nil {
		t.Fatal(err)
	}
	provider := &environmentRecordingProvider{boundaryProvider: &boundaryProvider{}}
	recorded := []string{"MODE=retained", "GOMAD3_IO_PROFILE=fixture", "GOMAD3_CHOICE_PROFILE=fixture"}
	preparer := &campaignPlanPreparer{source: target.Prepared{Path: source, Size: uint64(len("prepared"))}, provider: provider, environment: recorded}
	if _, err := preparer.Prepare(t.Context(), target.Spec{PreparationRoot: filepath.Join(t.TempDir(), "shard")}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(provider.environment, recorded) {
		t.Fatalf("shard target validated with environment %q; want %q", provider.environment, recorded)
	}
}
