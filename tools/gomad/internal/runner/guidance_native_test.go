//go:build gomad_native

package runner

import (
	"path/filepath"
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	campaignoptions "go.temporal.io/server/tools/gomad/internal/runner/options"
)

func TestGuidedCorpusSeedReproducesRetainedExecution(t *testing.T) {
	config := isolatedCampaign(t, conformanceTarget(t, "./environment"))
	config.Guide = true
	config.Corpus = filepath.Join(t.TempDir(), "corpus")
	config.Coverage = campaignoptions.CoverageSemantic
	config.KeepSuccesses = KeepSuccessesAll
	config.SuccessArtifactLimit = 1
	config.SuccessBytesLimit = 32 << 20
	first := exploreIsolated(t, config)
	config.Artifacts = t.TempDir()
	config.GuideRegression = true
	second := exploreIsolated(t, config)
	if first.CorpusAdded != 1 || second.CorpusAdded != 0 || second.Attempted != 1 {
		t.Fatalf("same-identity guided campaigns: first=%#v second=%#v", first, second)
	}
	original, err := artifact.OpenArtifact(first.SuccessArtifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := artifact.OpenArtifact(second.SuccessArtifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original.Manifest().Outcome, repeated.Manifest().Outcome) || !reflect.DeepEqual(original.Manifest().Streams, repeated.Manifest().Streams) || original.Manifest().IOProfile.Transcript.SHA256 != repeated.Manifest().IOProfile.Transcript.SHA256 {
		t.Fatalf("corpus seed did not reproduce its retained execution")
	}
}
