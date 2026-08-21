package protocol

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReleaseManifestMatchesExportedExperiments(t *testing.T) {
	encoded, err := os.ReadFile("../testdata/umpire3-1.2.json")
	require.NoError(t, err)
	release, err := DecodeReleaseManifest(encoded)
	require.NoError(t, err)
	require.NoError(t, release.ValidateAgainstCurrent())
	require.Equal(t, "umpire3/1.2", release.Release)
	require.Equal(t, "candidate", release.Status)
	require.Equal(t, FormatVersion, release.ExperimentFormatVersion)
	require.Equal(t, "sha256:082b6b66bbd5faf7ccd88deb28e49df237e5fa9202fa8a233c6db4a608edf081", release.DescriptorHash)
	catalog, err := DefaultCatalog()
	require.NoError(t, err)
	catalogHash, err := catalog.Digest()
	require.NoError(t, err)
	require.Equal(t, catalogHash, release.CatalogHash)
	monitors, err := DefaultMonitorCatalog()
	require.NoError(t, err)
	require.Equal(t, monitors.SemanticHash, release.MonitorSemanticHash)
	composition, err := DefaultComposition()
	require.NoError(t, err)
	require.Equal(t, composition.SemanticHash, release.CompositionSemanticHash)
	parity, err := DefaultParityLedger()
	require.NoError(t, err)
	require.Equal(t, parity.SemanticHash, release.ParitySemanticHash)
	require.ElementsMatch(t, []string{
		"local-in-process", "ci-test-cluster", "remote-deployment", "grpc-only-black-box", "production-canary",
	}, release.Profiles)
	migrationBytes, err := os.ReadFile("../migration/ledger.json")
	require.NoError(t, err)
	migrationHash := sha256.Sum256(migrationBytes)
	require.Equal(t, "sha256:"+hex.EncodeToString(migrationHash[:]), release.Migration.LedgerHash)
	var migration struct {
		Entries []json.RawMessage `json:"entries"`
	}
	require.NoError(t, json.Unmarshal(migrationBytes, &migration))
	require.Len(t, migration.Entries, release.Migration.BehaviorCount)

	for _, path := range []string{"../testdata/nexus-cancellation.json", "../testdata/update-lifecycle.json"} {
		encoded, err := os.ReadFile(path)
		require.NoError(t, err)
		var experiment Experiment
		require.NoError(t, json.Unmarshal(encoded, &experiment))
		require.Equal(t, experiment.Model.SemanticHash, release.Experiments[experiment.ExperimentID])
	}
}

func TestReleaseManifestSummarizesMigrationFidelity(t *testing.T) {
	releaseBytes, err := os.ReadFile("../testdata/umpire3-1.2.json")
	require.NoError(t, err)
	var release struct {
		Migration struct {
			FormatVersion           string `json:"formatVersion"`
			ExactCount              int    `json:"exactCount"`
			SemanticEquivalentCount int    `json:"semanticEquivalentCount"`
			PartialCount            int    `json:"partialCount"`
			InventoryOnlyCount      int    `json:"inventoryOnlyCount"`
		} `json:"migration"`
	}
	require.NoError(t, json.Unmarshal(releaseBytes, &release))
	migrationBytes, err := os.ReadFile("../migration/ledger.json")
	require.NoError(t, err)
	var ledger struct {
		Entries []struct {
			Fidelity string `json:"fidelity"`
		} `json:"entries"`
	}
	require.NoError(t, json.Unmarshal(migrationBytes, &ledger))
	counts := make(map[string]int)
	for _, entry := range ledger.Entries {
		counts[entry.Fidelity]++
	}
	require.Equal(t, "umpire3/migration-ledger/v3", release.Migration.FormatVersion)
	require.Equal(t, counts["exact"], release.Migration.ExactCount)
	require.Equal(t, counts["semantic-equivalent"], release.Migration.SemanticEquivalentCount)
	require.Equal(t, counts["partial"], release.Migration.PartialCount)
	require.Equal(t, counts["inventory-only"], release.Migration.InventoryOnlyCount)
}

func TestReleaseManifestRejectsInconsistentMigrationFidelityCounts(t *testing.T) {
	encoded, err := os.ReadFile("../testdata/umpire3-1.2.json")
	require.NoError(t, err)
	release, err := DecodeReleaseManifest(encoded)
	require.NoError(t, err)
	release.Migration.ExactCount--
	require.ErrorContains(t, release.Validate(), "migration fidelity counts")
}

func TestReleaseManifestRejectsInvalidProofManifestDigest(t *testing.T) {
	releaseBytes, err := os.ReadFile("../testdata/umpire3-1.2.json")
	require.NoError(t, err)
	var release map[string]any
	require.NoError(t, json.Unmarshal(releaseBytes, &release))
	release["proofManifests"] = []any{
		map[string]any{
			"identifier": "nexus-tasks-refinement-v1",
			"digest":     "not-a-digest",
		},
	}
	releaseBytes, err = json.Marshal(release)
	require.NoError(t, err)

	_, err = DecodeReleaseManifest(releaseBytes)
	require.ErrorContains(t, err, "proof manifest digest")
}

func TestReleaseManifestRejectsMismatchedCurrentProofManifestDigest(t *testing.T) {
	releaseBytes, err := os.ReadFile("../testdata/umpire3-1.2.json")
	require.NoError(t, err)
	release, err := DecodeReleaseManifest(releaseBytes)
	require.NoError(t, err)
	release.ProofManifests[0].Digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	require.ErrorContains(t, release.ValidateAgainstCurrent(), "proof manifest")
}

func TestReleaseManifestRejectsLegacyProofManifestReferenceFormat(t *testing.T) {
	releaseBytes, err := os.ReadFile("../testdata/umpire3-1.2.json")
	require.NoError(t, err)
	release, err := DecodeReleaseManifest(releaseBytes)
	require.NoError(t, err)
	release.FormatVersion = "umpire3/release/v1"

	require.Error(t, release.Validate())
}

func TestQualifiedReleaseRejectsPartialMigrationFidelity(t *testing.T) {
	encoded, err := os.ReadFile("../testdata/umpire3-1.2.json")
	require.NoError(t, err)
	release, err := DecodeReleaseManifest(encoded)
	require.NoError(t, err)
	release.Status = "qualified"
	for index := range release.Evidence {
		release.Evidence[index].Status = "passed"
	}
	release.Migration.ExactCount--
	release.Migration.PartialCount++
	release.ExternalQualifications = nil
	require.ErrorContains(t, release.Validate(), "migration behaviors remain partial")
}

func TestReleaseManifestRejectsMissingVisionEvidence(t *testing.T) {
	encoded, err := os.ReadFile("../testdata/umpire3-1.2.json")
	require.NoError(t, err)
	release, err := DecodeReleaseManifest(encoded)
	require.NoError(t, err)
	release.Evidence = release.Evidence[:len(release.Evidence)-1]
	require.ErrorContains(t, release.Validate(), "every Umpire vision goal")
}

func TestCandidateReleaseEvidenceMatchesAuditedVisionState(t *testing.T) {
	encoded, err := os.ReadFile("../testdata/umpire3-1.2.json")
	require.NoError(t, err)
	release, err := DecodeReleaseManifest(encoded)
	require.NoError(t, err)
	passed := map[string]struct{}{
		"deterministic-plans": {},
		"non-linear-identity": {},
	}
	for _, evidence := range release.Evidence {
		if _, complete := passed[evidence.Goal]; complete {
			require.Equal(t, "passed", evidence.Status, evidence.Goal)
		} else {
			require.Equal(t, "partial", evidence.Status, evidence.Goal)
		}
	}
}

func TestQualifiedReleaseRejectsPartialVisionEvidenceAndExternalGates(t *testing.T) {
	encoded, err := os.ReadFile("../testdata/umpire3-1.2.json")
	require.NoError(t, err)
	release, err := DecodeReleaseManifest(encoded)
	require.NoError(t, err)
	release.Status = "qualified"
	require.ErrorContains(t, release.Validate(), "lacks passing evidence")
	for index := range release.Evidence {
		release.Evidence[index].Status = "passed"
	}
	require.ErrorContains(t, release.Validate(), "external qualification gates")
}

func TestQualifiedReleaseRejectsUnresolvedParity(t *testing.T) {
	parity, err := DefaultParityLedger()
	require.NoError(t, err)
	parity.ResultClass = ResultClassMetadataValidated
	require.ErrorContains(t, validateQualifiedParity(parity), "not declaration-resolved")
}

func TestQualifiedReleaseAcceptsProofBackedComposition(t *testing.T) {
	composition, err := DefaultComposition()
	require.NoError(t, err)
	require.NoError(t, validateQualifiedComposition(composition))
	composition.ResultClass = ResultClassMetadataValidated
	require.ErrorContains(t, validateQualifiedComposition(composition), "not proof-backed")
}

func TestReleaseEvidenceAnchorsResolve(t *testing.T) {
	encoded, err := os.ReadFile("../testdata/umpire3-1.2.json")
	require.NoError(t, err)
	release, err := DecodeReleaseManifest(encoded)
	require.NoError(t, err)
	for _, evidence := range release.Evidence {
		for _, anchor := range evidence.Anchors {
			_, statErr := os.Stat("../" + anchor)
			require.NoError(t, statErr, "%s: %s", evidence.Goal, anchor)
		}
	}
	for name, path := range release.Documents {
		_, statErr := os.Stat("../" + path)
		require.NoError(t, statErr, "%s: %s", name, path)
	}
}
