package release

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolrelease "go.temporal.io/server/tests/umpire3/protocol/release"
)

func TestReleaseManifestRejectsMismatchedCurrentProofManifestDigest(t *testing.T) {
	release := loadReleaseManifest(t)
	release.ProofManifests[0].Digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	require.ErrorContains(t, ValidateArtifactBindingsAgainstCurrent(release), "proof manifest")
}

func TestReleaseManifestRejectsMismatchedCheckerCoverage(t *testing.T) {
	release := loadReleaseManifest(t)
	release.CheckerCoverageHash =
		"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	require.ErrorContains(t, ValidateArtifactBindingsAgainstCurrent(release), "checker coverage")
}

func TestQualifiedReleaseRejectsUnresolvedParity(t *testing.T) {
	parity, err := protocolcatalog.DefaultParityLedger()
	require.NoError(t, err)
	parity.ResultClass = protocolcatalog.ResultClassMetadataValidated
	require.ErrorContains(t, validateQualifiedParity(parity), "not declaration-resolved")

	parity, err = protocolcatalog.DefaultParityLedger()
	require.NoError(t, err)
	parity.Entries[0].EvidenceLevel = protocolcatalog.EvidenceModelProof
	require.ErrorContains(t, validateQualifiedParity(parity), "incomplete")
}

func TestQualifiedReleaseAcceptsProofBackedComposition(t *testing.T) {
	composition, err := protocolcatalog.DefaultComposition()
	require.NoError(t, err)
	require.NoError(t, validateQualifiedComposition(composition))
	composition.ResultClass = protocolcatalog.ResultClassMetadataValidated
	require.ErrorContains(t, validateQualifiedComposition(composition), "not proof-backed")
}

func loadReleaseManifest(t *testing.T) protocolrelease.ReleaseManifest {
	t.Helper()
	encoded, err := os.ReadFile("testdata/generated/umpire3-1.3.json")
	require.NoError(t, err)
	manifest, err := protocolrelease.DecodeReleaseManifest(encoded)
	require.NoError(t, err)
	return manifest
}
