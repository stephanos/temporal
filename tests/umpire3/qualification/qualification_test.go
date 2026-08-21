package qualification

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/canary"
	"go.temporal.io/server/tests/umpire3/evidence"
	umpire3runtime "go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestQualifyRejectsIncompleteInput(t *testing.T) {
	_, err := Qualify(Request{})
	require.EqualError(t, err, "release, experiment, result, profile, and signing key are required")
}

func TestQualificationRejectsSelfAttestedRuntimeAssurance(t *testing.T) {
	encoded, err := os.ReadFile("../testdata/update-lifecycle.json")
	require.NoError(t, err)
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(encoded), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	digest, err := experiment.Digest()
	require.NoError(t, err)
	result := umpire3runtime.Result{
		FormatVersion: umpire3runtime.ResultFormatVersion, ExperimentDigest: digest,
		ResultClass: protocol.ResultClassFiniteExhaustive, TrustBadge: protocol.TrustBadgeKernel,
		Claim: umpire3runtime.Claim{
			Kind: umpire3runtime.ClaimConforming, Property: experiment.Property.Identifier,
		},
	}
	require.ErrorContains(t, validateResult("remote-deployment", experiment, result), "assurance")
}

func TestQualifyRejectsIncompleteCanaryEnvelope(t *testing.T) {
	_, err := DecodeResult([]byte(`{"formatVersion":"umpire3/canary/v3","runtime":{},"complete":false}`))
	require.ErrorContains(t, err, "canary result is incomplete")
}

func TestQualificationRejectsIncompleteEnvironmentAuthority(t *testing.T) {
	experiment, result := qualificationResultFixture(t, umpire3runtime.EnvironmentIdentity{
		Name: "remote-deployment", BuildID: "build", ConfigurationIdentity: "configuration",
		EvidenceProfile: umpire3runtime.EvidenceProfilePublicGRPCHistory,
	})

	require.ErrorContains(t, validateResult("remote-deployment", experiment, result), "environment identity")
}

func TestQualificationAcceptsLocalInProcessEvidenceContract(t *testing.T) {
	experiment, result := qualificationResultFixture(t, umpire3runtime.EnvironmentIdentity{
		Name: "local-in-process", BuildID: "build",
		ConfigurationIdentity: digestFixture("local-configuration"),
		EvidenceProfile:       umpire3runtime.EvidenceProfileInProcessHooks,
		DrivingAuthority:      "local-test-authority",
		ObservationAuthority:  "local-server-hooks",
		FaultAuthority:        "isolated-local-faults",
		IsolationIdentity:     "namespace/task-queue",
		RetentionClass:        "semantic-redacted",
	})

	require.NoError(t, validateResult("local-in-process", experiment, result))
}

func TestQualificationRejectsCanaryWithoutHardExecutionBoundary(t *testing.T) {
	experiment, result := qualificationResultFixture(t, umpire3runtime.EnvironmentIdentity{
		Name: "production-canary", BuildID: "build", ConfigurationIdentity: "configuration",
		EvidenceProfile:      umpire3runtime.EvidenceProfilePublicGRPCHistory,
		DrivingAuthority:     "approved-production-worker",
		ObservationAuthority: "production-public-history",
		FaultAuthority:       "approved-production-fault-controller",
		IsolationIdentity:    "namespace/task-queue",
		RetentionClass:       "semantic-redacted",
	})

	require.ErrorContains(t, validateResult("production-canary", experiment, result), "hard execution")
}

func TestQualifyRejectsRawRuntimeResultForCanary(t *testing.T) {
	request, result := canaryQualificationFixture(t)
	resultBytes, err := json.Marshal(result)
	require.NoError(t, err)
	request.ResultBytes = resultBytes

	_, err = Qualify(request)
	require.ErrorContains(t, err, "canary envelope")
}

func TestQualifyRejectsCanaryWithoutRecoveryEvidence(t *testing.T) {
	request, result := canaryQualificationFixture(t)
	resultBytes, err := json.Marshal(canary.Result{
		FormatVersion: canary.FormatVersion,
		Runtime:       result,
		Complete:      true,
	})
	require.NoError(t, err)
	request.ResultBytes = resultBytes

	_, err = Qualify(request)
	require.ErrorContains(t, err, "recovery")
}

func canaryQualificationFixture(t *testing.T) (Request, umpire3runtime.Result) {
	t.Helper()
	releaseBytes, err := os.ReadFile("../testdata/umpire3-1.2.json")
	require.NoError(t, err)
	release, err := protocol.DecodeReleaseManifest(releaseBytes)
	require.NoError(t, err)
	privateKey := qualificationTestKey("production-canary")
	authority, err := protocol.NewQualificationAuthority(
		"test-authority/production-canary", privateKey.Public().(ed25519.PublicKey),
	)
	require.NoError(t, err)
	for index := range release.ExternalQualifications {
		if release.ExternalQualifications[index].Profile == "production-canary" {
			release.ExternalQualifications[index].Authority = &authority
		}
	}
	releaseBytes, err = release.CanonicalJSON()
	require.NoError(t, err)
	experimentBytes, err := os.ReadFile("../testdata/update-lifecycle.json")
	require.NoError(t, err)
	_, result := qualificationResultFixture(t, umpire3runtime.EnvironmentIdentity{
		Name: "production-canary", BuildID: "build", ConfigurationIdentity: "configuration",
		EvidenceProfile:      umpire3runtime.EvidenceProfilePublicGRPCHistory,
		DrivingAuthority:     "approved-production-worker",
		ObservationAuthority: "production-public-history",
		FaultAuthority:       "approved-production-fault-controller",
		IsolationIdentity:    "namespace/task-queue",
		RetentionClass:       "semantic-redacted",
		HardExecutionBudget:  true,
	})
	resultBytes, err := json.Marshal(result)
	require.NoError(t, err)
	return Request{
		ReleaseBytes: releaseBytes, ExperimentBytes: experimentBytes, ResultBytes: resultBytes,
		Profile: "production-canary", SigningKey: privateKey,
	}, result
}

func qualificationResultFixture(
	t *testing.T,
	environment umpire3runtime.EnvironmentIdentity,
) (protocol.Experiment, umpire3runtime.Result) {
	t.Helper()
	experimentBytes, err := os.ReadFile("../testdata/update-lifecycle.json")
	require.NoError(t, err)
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(experimentBytes), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	digest, err := experiment.Digest()
	require.NoError(t, err)
	result := umpire3runtime.Result{
		FormatVersion:    umpire3runtime.ResultFormatVersion,
		ExperimentDigest: digest,
		ResultClass:      protocol.ResultClassImplementationConforming,
		TrustBadge:       protocol.TrustBadgeTestedInstance,
		Environment:      environment,
		Claim: umpire3runtime.Claim{
			Kind: umpire3runtime.ClaimConforming, Property: experiment.Property.Identifier,
		},
		Evidence: evidence.Graph{
			FormatVersion: evidence.FormatVersion,
			Facts: []evidence.Fact{{
				Identifier: "fact", Kind: "observation", Value: true,
				SourceIdentity: "history", ClockDomain: "history", SourceSequence: 1,
				ObservedAtUnixNano: time.Now().UnixNano(), Reference: "history/1",
				EntityIdentity: "entity", Lineage: []string{"namespace", "entity"},
			}},
			Claims: []evidence.Claim{{Property: experiment.Property.Identifier, Verdict: "conforming"}},
		},
		Cleanup: umpire3runtime.CleanupResult{Complete: true},
	}
	result.DeriveAssurance()
	require.NoError(t, result.BindEvidenceDigest())
	return experiment, result
}

func TestDecodeResultRejectsEvidenceChangedAfterExecution(t *testing.T) {
	result := umpire3runtime.Result{
		FormatVersion: umpire3runtime.ResultFormatVersion,
		Claim:         umpire3runtime.Claim{Kind: umpire3runtime.ClaimInconclusive},
		Evidence: evidence.Graph{
			FormatVersion: evidence.FormatVersion,
			Claims:        []evidence.Claim{{Property: "property", Verdict: "inconclusive"}},
		},
	}
	result.DeriveAssurance()
	require.NoError(t, result.BindEvidenceDigest())
	require.NoError(t, validateDecodedResult(result))

	result.Evidence.Claims[0].Reason = "mutated"
	require.ErrorContains(t, validateDecodedResult(result), "evidence digest")
}

func TestPromoteAcceptsAllRequiredSignedProfileReceipts(t *testing.T) {
	releaseBytes, receipts := promotionFixture(t)
	promoted, err := Promote(PromotionRequest{ReleaseBytes: releaseBytes, Receipts: receipts})
	require.NoError(t, err)
	require.Equal(t, "qualified", promoted.Status)
	require.Len(t, promoted.Qualifications, 5)
	require.True(t, promoted.Assurance.Complete())
}

func TestPromoteRejectsUnsignedQualificationReceipts(t *testing.T) {
	releaseBytes, receipts := promotionFixture(t)
	receipt, err := DecodeReceipt(receipts[0])
	require.NoError(t, err)
	receipt.Signature = ""
	receipts[0], err = json.Marshal(receipt)
	require.NoError(t, err)

	_, err = Promote(PromotionRequest{ReleaseBytes: releaseBytes, Receipts: receipts})
	require.ErrorContains(t, err, "signature")
}

func TestPromoteRejectsMissingProfileAndReceiptMutation(t *testing.T) {
	releaseBytes, receipts := promotionFixture(t)
	_, err := Promote(PromotionRequest{ReleaseBytes: releaseBytes, Receipts: receipts[:2]})
	require.ErrorContains(t, err, "missing external qualification")

	drifted, err := DecodeReceipt(receipts[2])
	require.NoError(t, err)
	drifted.ExperimentDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	receipts[2], err = drifted.CanonicalJSON()
	require.NoError(t, err)
	_, err = Promote(PromotionRequest{ReleaseBytes: releaseBytes, Receipts: receipts})
	require.ErrorContains(t, err, "signature")
}

func promotionFixture(t *testing.T) ([]byte, [][]byte) {
	t.Helper()
	releaseBytes, err := os.ReadFile("../testdata/umpire3-1.2.json")
	require.NoError(t, err)
	release, err := protocol.DecodeReleaseManifest(releaseBytes)
	require.NoError(t, err)
	privateKeys := make(map[string]ed25519.PrivateKey, len(release.ExternalQualifications))
	for index := range release.ExternalQualifications {
		profileName := release.ExternalQualifications[index].Profile
		privateKey := qualificationTestKey(profileName)
		authority, authorityErr := protocol.NewQualificationAuthority(
			"test-authority/"+profileName, privateKey.Public().(ed25519.PublicKey),
		)
		require.NoError(t, authorityErr)
		release.ExternalQualifications[index].Authority = &authority
		privateKeys[profileName] = privateKey
	}
	releaseBytes, err = release.CanonicalJSON()
	require.NoError(t, err)
	releaseHash := sha256.Sum256(releaseBytes)

	experimentBytes, err := os.ReadFile("../testdata/update-lifecycle.json")
	require.NoError(t, err)
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(experimentBytes), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	experimentDigest, err := experiment.Digest()
	require.NoError(t, err)

	profiles := make([]string, len(release.ExternalQualifications))
	for index, gate := range release.ExternalQualifications {
		profiles[index] = gate.Profile
	}
	receipts := make([][]byte, len(profiles))
	for index, profileName := range profiles {
		receipt := Receipt{
			FormatVersion: FormatVersion, Release: release.Release,
			ReleaseDigest: "sha256:" + hex.EncodeToString(releaseHash[:]), Profile: profileName,
			ExperimentID: experiment.ExperimentID, ExperimentDigest: experimentDigest,
			ResultDigest: digestFixture("result/" + profileName), BuildID: "build/" + profileName,
			ConfigurationIdentity: digestFixture("configuration/" + profileName),
			EvidenceDigest:        digestFixture("evidence/" + profileName),
		}
		gate, found := requiredQualification(release, profileName)
		require.True(t, found)
		require.NotNil(t, gate.Authority)
		receipt.Authority = *gate.Authority
		receipt, err = protocol.SignQualificationReceipt(receipt, privateKeys[profileName])
		require.NoError(t, err)
		receipts[index], err = receipt.CanonicalJSON()
		require.NoError(t, err)
	}
	return releaseBytes, receipts
}

func qualificationTestKey(profile string) ed25519.PrivateKey {
	seed := sha256.Sum256([]byte("umpire3 qualification test authority/" + profile))
	return ed25519.NewKeyFromSeed(seed[:])
}

func digestFixture(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}
