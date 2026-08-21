package command

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/evidence"
	"go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/profile"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/qualification"
)

func TestQualificationProducesDigestBoundReceipt(t *testing.T) {
	t.Parallel()

	configuration := qualificationFixture(t, true)
	var output bytes.Buffer
	require.NoError(t, qualifyCompatibility(configuration, &output, defaultBackend{}))
	var result receipt
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	require.Equal(t, "umpire3/1.2", result.Release)
	require.Equal(t, "remote-deployment", result.Profile)
	require.NotEmpty(t, result.ReleaseDigest)
	require.NotEmpty(t, result.ResultDigest)
	require.NotEmpty(t, result.EvidenceDigest)
	require.Equal(t, "test-authority/remote-deployment", result.Authority.Identity)
	require.NotEmpty(t, result.Signature)
}

func TestQualificationCompatibilityAcceptsSigningKeyFlag(t *testing.T) {
	_, err := parseQualificationCompatibilityFlags([]string{"-signing-key", "authority.pem"})
	require.NoError(t, err)
}

func TestUnifiedQualificationWritesPromotableReceipt(t *testing.T) {
	t.Parallel()

	configuration := qualificationFixture(t, true)
	output := filepath.Join(t.TempDir(), "receipt.json")
	_, err := executeQualify([]string{
		"-release", configuration.releasePath,
		"-experiment", configuration.experimentPath,
		"-result", configuration.resultPath,
		"-profile", configuration.profile,
		"-signing-key", configuration.signingKeyPath,
		"-output", output,
	}, defaultBackend{})
	require.NoError(t, err)
	encoded, err := os.ReadFile(output)
	require.NoError(t, err)
	receipt, err := qualification.DecodeReceipt(encoded)
	require.NoError(t, err)
	require.Equal(t, configuration.profile, receipt.Profile)
}

func TestQualificationRejectsConformanceWithoutEvidence(t *testing.T) {
	t.Parallel()

	configuration := qualificationFixture(t, false)
	require.ErrorContains(t, qualifyCompatibility(configuration, &bytes.Buffer{}, defaultBackend{}), "no supporting evidence")
}

func TestQualificationRejectsIncompleteCanaryEnvelope(t *testing.T) {
	t.Parallel()

	_, err := decodeResult([]byte(`{"formatVersion":"umpire3/canary/v3","runtime":{},"complete":false}`))
	require.ErrorContains(t, err, "canary result is incomplete")
}

func TestQualificationRejectsMissingFaultRealizationEvidence(t *testing.T) {
	t.Parallel()

	configuration := qualificationFixture(t, true)
	resultBytes, err := os.ReadFile(configuration.resultPath)
	require.NoError(t, err)
	var result execution.Result
	require.NoError(t, json.Unmarshal(resultBytes, &result))
	result.Faults = nil
	resultBytes, err = json.Marshal(result)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configuration.resultPath, resultBytes, 0o600))
	require.ErrorContains(t, qualifyCompatibility(configuration, &bytes.Buffer{}, defaultBackend{}), "every declared fault")
}

func qualificationFixture(t *testing.T, withEvidence bool) qualificationCompatibilityOptions {
	t.Helper()
	releaseBytes, err := os.ReadFile("../../testdata/umpire3-1.2.json")
	require.NoError(t, err)
	release, err := protocol.DecodeReleaseManifest(releaseBytes)
	require.NoError(t, err)
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	authority, err := protocol.NewQualificationAuthority(
		"test-authority/remote-deployment", privateKey.Public().(ed25519.PublicKey),
	)
	require.NoError(t, err)
	for index := range release.ExternalQualifications {
		if release.ExternalQualifications[index].Profile == "remote-deployment" {
			release.ExternalQualifications[index].Authority = &authority
		}
	}
	releaseBytes, err = release.CanonicalJSON()
	require.NoError(t, err)
	releasePath := filepath.Join(t.TempDir(), "release.json")
	require.NoError(t, os.WriteFile(releasePath, releaseBytes, 0o600))
	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)
	signingKeyPath := filepath.Join(t.TempDir(), "authority.pem")
	require.NoError(t, os.WriteFile(signingKeyPath, pem.EncodeToMemory(&pem.Block{
		Type: "PRIVATE KEY", Bytes: privateKeyDER,
	}), 0o600))

	experimentBytes, err := os.ReadFile("../../testdata/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(experimentBytes), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	digest, err := experiment.Digest()
	require.NoError(t, err)
	graph := evidence.Graph{FormatVersion: evidence.FormatVersion}
	if withEvidence {
		graph.Facts = []evidence.Fact{{
			Identifier: "qualified", Kind: experiment.Checkpoints[0].Observation, Value: true,
			SourceIdentity: "remote-history", ClockDomain: "remote-history-sequence", SourceSequence: 1,
			ObservedAtUnixNano: time.Now().UnixNano(), Reference: "history/1",
			EntityIdentity: "entity", Lineage: []string{"namespace", "entity"},
		}}
		graph.Claims = []evidence.Claim{{Property: experiment.Property.Identifier, Verdict: "conforming"}}
	}
	result := execution.Result{
		FormatVersion: execution.ResultFormatVersion, ExperimentDigest: digest,
		ResultClass: protocol.ResultClassImplementationConforming,
		TrustBadge:  protocol.TrustBadgeTestedInstance,
		Claim: execution.Claim{
			Kind: execution.ClaimConforming, Property: experiment.Property.Identifier,
		},
		Evidence: graph, Cleanup: execution.CleanupResult{Complete: true},
	}
	definition, err := profile.Define(profile.Remote(
		"https://temporal.example", "token", "build", "namespace", "task-queue",
	))
	require.NoError(t, err)
	result.Environment = definition.Environment
	for _, fault := range experiment.Faults {
		result.Faults = append(result.Faults, execution.FaultResult{
			Identifier: fault.Identifier, Kind: fault.Kind, SourceIdentity: "remote-fault-controller",
			Reference: "remote-fault/fired", EntityIdentity: "fault-scope",
			Installed: true, Activated: true, Realized: true, Released: true, CleanupComplete: true,
		})
	}
	require.NoError(t, result.BindEvidenceDigest())
	resultBytes, err := json.Marshal(result)
	require.NoError(t, err)
	resultPath := filepath.Join(t.TempDir(), "result.json")
	require.NoError(t, os.WriteFile(resultPath, resultBytes, 0o600))
	return qualificationCompatibilityOptions{
		releasePath: releasePath, experimentPath: "../../testdata/nexus-cancellation.json",
		resultPath: resultPath, profile: "remote-deployment", signingKeyPath: signingKeyPath,
	}
}
