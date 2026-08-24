package generated

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistryOwnsEveryDeclaredArtifact(t *testing.T) {
	for _, name := range []Name{
		Catalog, CheckerCoverage, Composition, CoverageDenominator, DescriptorManifest,
		FamilyDependencies, MonitorPrograms, NexusExactMutationProofManifest,
		NexusMutationProofManifest, NexusProofManifest, ParityLedger,
		TaskDeliveryMutatedTemporal, TaskDeliveryTemporal, UpdateProofManifest,
	} {
		require.NotEmpty(t, Read(name), name)
	}
}
