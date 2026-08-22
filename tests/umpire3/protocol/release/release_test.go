package release

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReleaseManifestMatchesExportedExperiments(t *testing.T) {
	encoded, err := os.ReadFile("../../assurance/release/testdata/generated/umpire3-1.3.json")
	require.NoError(t, err)
	release, err := DecodeReleaseManifest(encoded)
	require.NoError(t, err)
	require.NoError(t, release.Validate())
	require.Equal(t, "umpire3/1.3", release.Release)
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
	checkerCoverage, err := DefaultCheckerCoverage()
	require.NoError(t, err)
	checkerCoverageJSON, err := checkerCoverage.CanonicalJSON()
	require.NoError(t, err)
	require.Equal(t, digestBytes(checkerCoverageJSON), release.CheckerCoverageHash)
	require.ElementsMatch(t, []string{
		"local-in-process", "ci-test-cluster", "remote-deployment", "grpc-only-black-box", "production-canary",
	}, release.Profiles)
	migrationBytes, err := os.ReadFile("../../assurance/migration/testdata/generated/ledger.json")
	require.NoError(t, err)
	migrationHash := sha256.Sum256(migrationBytes)
	require.Equal(t, "sha256:"+hex.EncodeToString(migrationHash[:]), release.Migration.LedgerHash)
	var migration struct {
		Entries []json.RawMessage `json:"entries"`
	}
	require.NoError(t, json.Unmarshal(migrationBytes, &migration))
	require.Len(t, migration.Entries, release.Migration.BehaviorCount)

	for _, path := range []string{"../../testdata/generated/nexus-cancellation.json", "../../testdata/generated/update-lifecycle.json"} {
		encoded, err := os.ReadFile(path)
		require.NoError(t, err)
		var experiment Experiment
		require.NoError(t, json.Unmarshal(encoded, &experiment))
		digest, digestErr := experiment.Digest()
		require.NoError(t, digestErr)
		require.Equal(t, ReleaseExperiment{
			SemanticHash: experiment.Model.SemanticHash,
			Digest:       digest,
		}, release.Experiments[experiment.ExperimentID])
	}
}

func TestQualificationReceiptRejectsPayloadMutation(t *testing.T) {
	seed := sha256.Sum256([]byte("umpire3 qualification receipt mutation test"))
	privateKey := ed25519.NewKeyFromSeed(seed[:])
	authority, err := NewQualificationAuthority("test-authority", privateKey.Public().(ed25519.PublicKey))
	require.NoError(t, err)
	receipt, err := signQualificationReceiptForTest(QualificationReceipt{
		FormatVersion:         QualificationReceiptFormatVersion,
		Release:               "umpire3/test",
		ReleaseDigest:         "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		Profile:               "remote-deployment",
		ExperimentID:          "experiment",
		ExperimentDigest:      "sha256:2222222222222222222222222222222222222222222222222222222222222222",
		ResultDigest:          "sha256:3333333333333333333333333333333333333333333333333333333333333333",
		BuildID:               "build",
		ConfigurationIdentity: "sha256:5555555555555555555555555555555555555555555555555555555555555555",
		EvidenceDigest:        "sha256:4444444444444444444444444444444444444444444444444444444444444444",
		Authority:             authority,
	}, privateKey)
	require.NoError(t, err)
	receipt.BuildID = "mutated-build"
	encoded, err := json.Marshal(receipt)
	require.NoError(t, err)

	_, err = DecodeQualificationReceipt(encoded)
	require.ErrorContains(t, err, "signature")
}

func TestQualificationReceiptRejectsOpaqueConfigurationIdentity(t *testing.T) {
	seed := sha256.Sum256([]byte("umpire3 qualification configuration identity test"))
	privateKey := ed25519.NewKeyFromSeed(seed[:])
	authority, err := NewQualificationAuthority("test-authority", privateKey.Public().(ed25519.PublicKey))
	require.NoError(t, err)
	_, err = signQualificationReceiptForTest(QualificationReceipt{
		FormatVersion:         QualificationReceiptFormatVersion,
		Release:               "umpire3/test",
		ReleaseDigest:         "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		Profile:               "remote-deployment",
		ExperimentID:          "experiment",
		ExperimentDigest:      "sha256:2222222222222222222222222222222222222222222222222222222222222222",
		ResultDigest:          "sha256:3333333333333333333333333333333333333333333333333333333333333333",
		BuildID:               "build",
		ConfigurationIdentity: "opaque-configuration",
		EvidenceDigest:        "sha256:4444444444444444444444444444444444444444444444444444444444444444",
		Authority:             authority,
	}, privateKey)
	require.ErrorContains(t, err, "configuration")
}

func signQualificationReceiptForTest(
	receipt QualificationReceipt,
	privateKey ed25519.PrivateKey,
) (QualificationReceipt, error) {
	if err := receipt.ValidateUnsigned(); err != nil {
		return QualificationReceipt{}, err
	}
	payload, err := receipt.SigningPayload()
	if err != nil {
		return QualificationReceipt{}, err
	}
	receipt.Signature = base64.RawStdEncoding.EncodeToString(ed25519.Sign(privateKey, payload))
	if err := receipt.Verify(receipt.Authority); err != nil {
		return QualificationReceipt{}, err
	}
	return receipt, nil
}

func TestReleaseManifestSummarizesMigrationFidelity(t *testing.T) {
	releaseBytes, err := os.ReadFile("../../assurance/release/testdata/generated/umpire3-1.3.json")
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
	migrationBytes, err := os.ReadFile("../../assurance/migration/testdata/generated/ledger.json")
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
	encoded, err := os.ReadFile("../../assurance/release/testdata/generated/umpire3-1.3.json")
	require.NoError(t, err)
	release, err := DecodeReleaseManifest(encoded)
	require.NoError(t, err)
	release.Migration.ExactCount--
	require.ErrorContains(t, release.Validate(), "migration fidelity counts")
}

func TestReleaseManifestRejectsInvalidProofManifestDigest(t *testing.T) {
	releaseBytes, err := os.ReadFile("../../assurance/release/testdata/generated/umpire3-1.3.json")
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

func TestReleaseManifestRejectsLegacyProofManifestReferenceFormat(t *testing.T) {
	releaseBytes, err := os.ReadFile("../../assurance/release/testdata/generated/umpire3-1.3.json")
	require.NoError(t, err)
	release, err := DecodeReleaseManifest(releaseBytes)
	require.NoError(t, err)
	release.FormatVersion = "umpire3/release/v1"

	require.Error(t, release.Validate())
}

func TestQualifiedReleaseRejectsPartialMigrationFidelity(t *testing.T) {
	encoded, err := os.ReadFile("../../assurance/release/testdata/generated/umpire3-1.3.json")
	require.NoError(t, err)
	release, err := DecodeReleaseManifest(encoded)
	require.NoError(t, err)
	release.Status = "qualified"
	release.Assurance = completeReleaseAssurance(t, release.Assurance)
	release.Migration.ExactCount--
	release.Migration.PartialCount++
	release.ExternalQualifications = nil
	require.ErrorContains(t, release.Validate(), "migration behaviors remain partial")
}

func TestReleaseManifestRejectsMissingVisionEvidence(t *testing.T) {
	encoded, err := os.ReadFile("../../assurance/release/testdata/generated/umpire3-1.3.json")
	require.NoError(t, err)
	release, err := DecodeReleaseManifest(encoded)
	require.NoError(t, err)
	release.Assurance.Goals = release.Assurance.Goals[:len(release.Assurance.Goals)-1]
	require.ErrorContains(t, release.Validate(), "every Umpire vision goal")
}

func TestReleaseManifestRejectsAuthoredVisionStatus(t *testing.T) {
	encoded, err := os.ReadFile("../../assurance/release/testdata/generated/umpire3-1.3.json")
	require.NoError(t, err)
	var release map[string]any
	require.NoError(t, json.Unmarshal(encoded, &release))
	assurance, ok := release["assurance"].(map[string]any)
	require.True(t, ok)
	goals, ok := assurance["goals"].([]any)
	require.True(t, ok)
	first, ok := goals[0].(map[string]any)
	require.True(t, ok)
	first["status"] = "passed"
	encoded, err = json.Marshal(release)
	require.NoError(t, err)

	_, err = DecodeReleaseManifest(encoded)
	require.ErrorContains(t, err, "unknown field")
}

func TestCandidateReleaseEvidenceMatchesAuditedVisionState(t *testing.T) {
	encoded, err := os.ReadFile("../../assurance/release/testdata/generated/umpire3-1.3.json")
	require.NoError(t, err)
	var retained struct {
		Assurance ReleaseAssurance `json:"assurance"`
	}
	require.NoError(t, json.Unmarshal(encoded, &retained))
	sealed, err := SealReleaseAssurance(retained.Assurance)
	require.NoError(t, err)
	require.Equal(t, sealed.Digest, retained.Assurance.Digest)
	release, err := DecodeReleaseManifest(encoded)
	require.NoError(t, err)
	require.False(t, release.Assurance.Complete())
	omitted := map[string]struct{}{
		"portable-profiles": {}, "white-box-black-box": {},
	}
	for _, goal := range release.Assurance.Goals {
		_, incomplete := omitted[goal.Identifier]
		require.Equal(t, incomplete, len(goal.Omissions) != 0, goal.Identifier)
	}
}

func TestQualifiedReleaseRejectsPartialVisionEvidenceAndExternalGates(t *testing.T) {
	encoded, err := os.ReadFile("../../assurance/release/testdata/generated/umpire3-1.3.json")
	require.NoError(t, err)
	release, err := DecodeReleaseManifest(encoded)
	require.NoError(t, err)
	release.Status = "qualified"
	require.ErrorContains(t, release.Validate(), "unresolved omissions")
	release.Assurance = completeReleaseAssurance(t, release.Assurance)
	require.ErrorContains(t, release.Validate(), "external qualification gates")
}

func TestQualifiedReleaseRequiresExternalQualificationEvidence(t *testing.T) {
	encoded, err := os.ReadFile("../../assurance/release/testdata/generated/umpire3-1.3.json")
	require.NoError(t, err)
	release, err := DecodeReleaseManifest(encoded)
	require.NoError(t, err)
	release.Status = "qualified"
	release.Assurance = completeReleaseAssurance(t, release.Assurance)
	release.ExternalQualifications = nil
	release.Qualifications = nil
	require.ErrorContains(t, release.Validate(), "qualification evidence")
}

func TestCandidateReleaseRequiresQualificationGateForEveryProfile(t *testing.T) {
	encoded, err := os.ReadFile("../../assurance/release/testdata/generated/umpire3-1.3.json")
	require.NoError(t, err)
	release, err := DecodeReleaseManifest(encoded)
	require.NoError(t, err)
	release.ExternalQualifications = release.ExternalQualifications[1:]

	require.ErrorContains(t, release.Validate(), "qualification gates cover")
}

func TestReleaseDocumentsResolve(t *testing.T) {
	encoded, err := os.ReadFile("../../assurance/release/testdata/generated/umpire3-1.3.json")
	require.NoError(t, err)
	release, err := DecodeReleaseManifest(encoded)
	require.NoError(t, err)
	for name, path := range release.Documents {
		_, statErr := os.Stat("../../" + path)
		require.NoError(t, statErr, "%s: %s", name, path)
	}
}

func completeReleaseAssurance(t *testing.T, assurance ReleaseAssurance) ReleaseAssurance {
	t.Helper()
	for index := range assurance.Goals {
		assurance.Goals[index].Omissions = nil
	}
	assurance, err := SealReleaseAssurance(assurance)
	require.NoError(t, err)
	return assurance
}
