package artifact_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

func TestArtifactSetEvaluationClosureCanonicalManifest(t *testing.T) {
	admitted, err := artifact.AdmitSet(artifactSetFixtureMembers(t))
	require.NoError(t, err)
	require.Equal(t,
		"umpire.artifact-set.1f4d21d39c33440ff37bee22db09680293459a69eb702f5496f8bfa6b1dab890",
		admitted.Identity())
	require.Equal(t,
		"sha256:12a0e9709e823da060eef54998e6cd36973779c725b177ce1eda5b9954e3499b",
		admitted.Checksum())
	require.Equal(t,
		"sha256:052fa0eff77536213db67f452c543df4bbda4a606ee6f504d3b6cb596b33c9db",
		admitted.ManifestSHA256())
	require.Equal(t, readExperimentV2Fixture(t,
		"model/Umpire/Artifact/Tests/Fixtures/ArtifactSetV2.json"), admitted.ManifestBytes())
	requireCanonicalJSONLine(t, admitted.ManifestBytes())
}

func TestArtifactSetManifestAdmissionRequiresExactCanonicalBytes(t *testing.T) {
	members := artifactSetFixtureMembers(t)
	manifest := readExperimentV2Fixture(t,
		"model/Umpire/Artifact/Tests/Fixtures/ArtifactSetV2.json")

	admitted, err := artifact.AdmitSetManifest(manifest, members)
	require.NoError(t, err)
	require.Equal(t, manifest, admitted.ManifestBytes())

	var compact bytes.Buffer
	require.NoError(t, json.Compact(&compact, manifest))
	admitted, err = artifact.AdmitSetManifest(compact.Bytes(), members)
	requireArtifactSetErrorCode(t, err, artifact.ErrorNoncanonical)
	require.Empty(t, admitted.Identity())

	extraLF := append(bytes.Clone(manifest), '\n')
	admitted, err = artifact.AdmitSetManifest(extraLF, members)
	requireArtifactSetErrorCode(t, err, artifact.ErrorNoncanonical)
	require.Empty(t, admitted.Identity())

	staleRow := bytes.Replace(manifest,
		[]byte("sha256:c7fc19d59b8b97922df475596bc45022e97c19d051149aa0c9aabe82dff18179"),
		[]byte("sha256:454acc851c5c1638166b1a334eaaedc97e4515b5ebe6614d5a57672ddbd9d1c2"), 1)
	admitted, err = artifact.AdmitSetManifest(staleRow, members)
	requireArtifactSetErrorCode(t, err, artifact.ErrorArtifactChecksum)
	require.Empty(t, admitted.Identity())
}

func TestArtifactSetAdmitsOnlyThreeExactClosures(t *testing.T) {
	members := artifactSetFixtureMembers(t)
	for _, test := range []struct {
		name     string
		count    int
		identity string
		checksum string
		sha256   string
	}{
		{
			name: "executable", count: 2,
			identity: "umpire.artifact-set.4b7c7fb8319e64bbab53abc7f0f73f3b22733b08c11caa9cbd508fe1f59c7775",
			checksum: "sha256:b616e4474e81c6409fb2476ea959782db1486091cbe71310188a9a6074f798b5",
			sha256:   "sha256:5c3c519826fa8867a9c453ce1305b6bccd1fb60f2cfd1cfa7f4c2a19e2478f91",
		},
		{
			name: "execution", count: 4,
			identity: "umpire.artifact-set.3dda4efe07ac24ef454f7dc4227440277cb59caf4a4d671ac09d5bc11555f2f0",
			checksum: "sha256:2aded1f8e2c52e3a775cb2a3ea009924d874e298cc733f1ba6f5de6559026c45",
			sha256:   "sha256:76a606f491e778a636fd6b6adb4d604af8f17d79b4d573ca41d421c49213c505",
		},
		{
			name: "evaluation", count: 6,
			identity: "umpire.artifact-set.1f4d21d39c33440ff37bee22db09680293459a69eb702f5496f8bfa6b1dab890",
			checksum: "sha256:12a0e9709e823da060eef54998e6cd36973779c725b177ce1eda5b9954e3499b",
			sha256:   "sha256:052fa0eff77536213db67f452c543df4bbda4a606ee6f504d3b6cb596b33c9db",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			admitted, err := artifact.AdmitSet(members[:test.count])
			require.NoError(t, err)
			require.Equal(t, test.identity, admitted.Identity())
			require.Equal(t, test.checksum, admitted.Checksum())
			require.Equal(t, test.sha256, admitted.ManifestSHA256())
		})
	}

	for _, count := range []int{0, 1, 3, 5, 7} {
		t.Run(fmt.Sprintf("reject %d members", count), func(t *testing.T) {
			candidate := append(artifactSetFixtureMembers(t), artifactSetFixtureMembers(t)[0])
			admitted, err := artifact.AdmitSet(candidate[:count])
			requireArtifactSetErrorCode(t, err, artifact.ErrorClosure)
			require.Empty(t, admitted.Identity())
			require.Nil(t, admitted.ManifestBytes())
		})
	}
}

func TestArtifactSetExecutableProjectionIsExactAndImmutable(t *testing.T) {
	members := artifactSetFixtureMembers(t)
	admitted, err := artifact.AdmitSet(members[:2])
	require.NoError(t, err)

	executable, ok := admitted.Executable()
	require.True(t, ok)
	require.Equal(t, admitted.Identity(), executable.AdmittedSet().Identity())

	experiment := executable.Experiment()
	runtimeConfiguration := executable.RuntimeConfiguration()
	experiment.Plan.CapabilityRequirementDefinitionIDs[0] = "changed.capability"
	runtimeConfiguration.AuthorityProfile.RequiredCapabilityDefinitionIDs = append(
		runtimeConfiguration.AuthorityProfile.RequiredCapabilityDefinitionIDs,
		"changed.capability",
	)
	runtimeConfiguration.ParticipantBindings[0].CapabilityDefinitionIDs[0] = "changed.capability"

	again, ok := admitted.Executable()
	require.True(t, ok)
	require.NotEqual(t, experiment, again.Experiment())
	require.NotEqual(t, runtimeConfiguration, again.RuntimeConfiguration())
	require.Equal(t, admitted.Identity(), again.AdmittedSet().Identity())

	executionSet, err := artifact.AdmitSet(members[:4])
	require.NoError(t, err)
	_, ok = executionSet.Executable()
	require.False(t, ok)
}

func TestArtifactSetRejectsMemberPathAndOrderMutations(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func([]artifact.SetMember)
		code   artifact.ErrorCode
	}{
		{
			name: "duplicate path",
			mutate: func(members []artifact.SetMember) {
				members[1].Path = members[0].Path
			},
			code: artifact.ErrorClosure,
		},
		{
			name: "unsafe path",
			mutate: func(members []artifact.SetMember) {
				members[0].Path = "../experiment.json"
			},
			code: artifact.ErrorClosure,
		},
		{
			name: "path order",
			mutate: func(members []artifact.SetMember) {
				members[0].Path, members[1].Path = members[1].Path, members[0].Path
			},
			code: artifact.ErrorClosure,
		},
		{
			name: "family order",
			mutate: func(members []artifact.SetMember) {
				members[0].Encoded, members[1].Encoded = members[1].Encoded, members[0].Encoded
			},
			code: artifact.ErrorWrongFamily,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			members := artifactSetFixtureMembers(t)
			test.mutate(members)
			admitted, err := artifact.AdmitSet(members)
			requireArtifactSetErrorCode(t, err, test.code)
			require.Empty(t, admitted.Identity())
		})
	}
}

func TestArtifactSetRejectsNoncanonicalAndStaleMembersAtomically(t *testing.T) {
	t.Run("mixed version", func(t *testing.T) {
		members := artifactSetFixtureMembers(t)
		members[0].Encoded = bytes.Replace(members[0].Encoded,
			[]byte("umpire-experiment/v2"), []byte("umpire-experiment/"+"v1"), 1)
		admitted, err := artifact.AdmitSet(members)
		requireArtifactSetErrorCode(t, err, artifact.ErrorUnsupportedFormat)
		require.Empty(t, admitted.Identity())
	})

	t.Run("compact member", func(t *testing.T) {
		members := artifactSetFixtureMembers(t)
		var compact bytes.Buffer
		require.NoError(t, json.Compact(&compact, members[4].Encoded))
		members[4].Encoded = compact.Bytes()
		admitted, err := artifact.AdmitSet(members)
		requireArtifactSetErrorCode(t, err, artifact.ErrorNoncanonical)
		require.Empty(t, admitted.Identity())
	})

	t.Run("stale binding", func(t *testing.T) {
		members := artifactSetFixtureMembers(t)
		runtimeConfiguration, err := artifact.DecodeRuntimeConfigurationV2(members[1].Encoded)
		require.NoError(t, err)
		runtimeConfiguration.Experiment.ArtifactChecksum = runtimeConfiguration.ArtifactChecksum
		runtimeConfiguration, err = artifactv2.SealRuntimeConfiguration(runtimeConfiguration)
		require.NoError(t, err)
		members[1].Encoded, err = artifactv2.CanonicalRuntimeConfigurationBytes(runtimeConfiguration)
		require.NoError(t, err)
		admitted, err := artifact.AdmitSet(members)
		requireArtifactSetErrorCode(t, err, artifact.ErrorClosure)
		require.Empty(t, admitted.Identity())
	})

	t.Run("unresolved implementation source target", func(t *testing.T) {
		members := artifactSetFixtureMembers(t)
		documents := loadCrossLanguageGoldenDocuments(t)
		documents.result.ImplementationLink.SourceTarget =
			documents.result.ImplementationLink.DestinationTarget
		documents.result = sealGoldenResultWithOutcome(
			t, documents.result, documents.evidence, documents.experiment,
		)
		var err error
		members[5].Encoded, err = artifactv2.CanonicalResultBytes(documents.result)
		require.NoError(t, err)
		admitted, err := artifact.AdmitSet(members)
		requireArtifactSetErrorCode(t, err, artifact.ErrorClosure)
		require.Empty(t, admitted.Identity())
	})
}

func TestUnsupportedFormatMixedArtifactSetsPrecedeChecksumsAndClosure(t *testing.T) {
	fixtures := []string{
		"experiment",
		"runtime-configuration",
		"experiment-run",
		"raw-evidence",
		"evidence",
		"result",
	}
	for index, fixture := range fixtures {
		for _, major := range []string{"v1", "v3"} {
			t.Run(fixture+"/"+major, func(t *testing.T) {
				members := artifactSetFixtureMembers(t)
				invalidIndex := 0
				if index == invalidIndex {
					invalidIndex = 1
				}
				corruptFirstArtifactChecksum(t, members[invalidIndex].Encoded)
				members[index].Encoded = readExperimentV2Fixture(t,
					unsupportedFormatFixturePath(fixture, major))
				before := cloneArtifactSetMembers(members)

				admitted, err := artifact.AdmitSet(members)
				requireArtifactSetErrorCode(t, err, artifact.ErrorUnsupportedFormat)
				require.Empty(t, admitted.Identity())
				require.Nil(t, admitted.ManifestBytes())
				require.Equal(t, before, members)
			})
		}
	}
}

func TestUnsupportedFormatMixedArtifactSetsPreserveStructuralPrecedence(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func([]artifact.SetMember)
		code   artifact.ErrorCode
	}{
		{
			name: "syntax before unsupported format",
			mutate: func(members []artifact.SetMember) {
				members[0].Encoded = readExperimentV2Fixture(t,
					unsupportedFormatFixturePath("experiment", "v1"))
				members[5].Encoded = []byte("{\n")
			},
			code: artifact.ErrorSyntax,
		},
		{
			name: "unsupported format before wrong family",
			mutate: func(members []artifact.SetMember) {
				members[0].Encoded = []byte("{\n  \"formatVersion\": \"umpire-result/v2\"\n}\n")
				members[5].Encoded = readExperimentV2Fixture(t,
					"tools/umpire/artifact/testdata/unsupported/result-v3.json")
			},
			code: artifact.ErrorUnsupportedFormat,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			members := artifactSetFixtureMembers(t)
			test.mutate(members)
			admitted, err := artifact.AdmitSet(members)
			requireArtifactSetErrorCode(t, err, test.code)
			require.Empty(t, admitted.Identity())
			require.Nil(t, admitted.ManifestBytes())
		})
	}
}

func TestArtifactSetAdmittedValueOwnsManifestBytes(t *testing.T) {
	members := artifactSetFixtureMembers(t)
	admitted, err := artifact.AdmitSet(members)
	require.NoError(t, err)
	wantManifest := admitted.ManifestBytes()

	members[0].Encoded[0] = '['
	returned := admitted.ManifestBytes()
	returned[0] = '['

	require.Equal(t, wantManifest, admitted.ManifestBytes())
}

func artifactSetFixtureMembers(t *testing.T) []artifact.SetMember {
	t.Helper()
	return []artifact.SetMember{
		{Path: "artifacts/experiment.json", Encoded: readExperimentV2Fixture(t,
			"tools/umpire/artifact/testdata/switch-experiment-v2.json")},
		{Path: "artifacts/runtime-configuration.json", Encoded: readExperimentV2Fixture(t,
			"tools/umpire/artifact/testdata/runtime-configuration-v2.json")},
		{Path: "artifacts/experiment-run.json", Encoded: readExperimentV2Fixture(t,
			"tools/umpire/artifact/testdata/experiment-run-v2.json")},
		{Path: "artifacts/raw-evidence.json", Encoded: readExperimentV2Fixture(t,
			"tools/umpire/artifact/testdata/raw-evidence-v2.json")},
		{Path: "artifacts/evidence.json", Encoded: readExperimentV2Fixture(t,
			"tools/umpire/artifact/testdata/evidence-v2.json")},
		{Path: "artifacts/result.json", Encoded: readExperimentV2Fixture(t,
			"tools/umpire/artifact/testdata/result-v2.json")},
	}
}

func cloneArtifactSetMembers(members []artifact.SetMember) []artifact.SetMember {
	cloned := make([]artifact.SetMember, len(members))
	for index, member := range members {
		cloned[index] = artifact.SetMember{Path: member.Path, Encoded: bytes.Clone(member.Encoded)}
	}
	return cloned
}

func corruptFirstArtifactChecksum(t *testing.T, encoded []byte) {
	t.Helper()
	prefix := []byte(`"artifactChecksum": "sha256:`)
	start := bytes.Index(encoded, prefix)
	require.NotEqual(t, -1, start)
	position := start + len(prefix)
	if encoded[position] == '0' {
		encoded[position] = '1'
	} else {
		encoded[position] = '0'
	}
}

func requireArtifactSetErrorCode(t *testing.T, err error, expected artifact.ErrorCode) {
	t.Helper()
	require.Error(t, err)
	code, ok := artifact.CodeOf(err)
	require.True(t, ok, err)
	require.Equal(t, expected, code)
}
