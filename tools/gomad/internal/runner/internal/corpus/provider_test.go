package corpus

import (
	"context"
	"errors"
	"slices"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/evidence/record/testfixture"
)

type corpusEvidenceProvider struct {
	operations []backends.OperationOutcome
	err        error
}

func (provider corpusEvidenceProvider) Evidence(_ context.Context, request backends.EvidenceRequest) (backends.Evidence, error) {
	if provider.err != nil {
		return backends.Evidence{}, provider.err
	}
	probes := request.Probes
	operations := provider.operations
	if request.Transcript != nil {
		decoded, err := testfixture.DecodeTranscript(request.Transcript)
		if err != nil {
			return backends.Evidence{}, err
		}
		if operations == nil {
			operations = make([]backends.OperationOutcome, 0, len(decoded))
			for _, operation := range decoded {
				operations = append(operations, backends.OperationOutcome{Name: operation.Name, Result: operation.Result})
			}
		}
		coverage, err := testfixture.DecodeSemanticCoverage(request.Transcript)
		if err != nil {
			return backends.Evidence{}, err
		}
		probes = coverage.Probes
	}
	coverage, err := testfixture.SummarizeSemanticProbes(probes)
	return backends.Evidence{Coverage: coverage, Operations: operations}, err
}

func corpusDescription() backends.Description {
	profile := testfixture.Default()
	version, boundary := testfixture.BoundaryManifestIdentity()
	return backends.Description{
		IOProfile:       record.IOProfile{Name: profile.Name(), ImplementationSHA256: record.SHA256(profile.ImplementationSHA256()), Inventory: string(profile.Inventory()), InventorySHA256: record.SHA256(profile.InventorySHA256())},
		BoundaryVersion: version, BoundarySHA256: record.SHA256(boundary),
		SemanticInstrumentationSHA256: record.SHA256(testfixture.SemanticInstrumentationIdentity()),
	}
}

func TestCorpusUsesProviderEvidenceAndInstrumentation(t *testing.T) {
	input, coverage, _ := guideArtifactInput(t, 7)
	description := corpusDescription()
	description.SemanticInstrumentationSHA256 = record.HashBytes([]byte("different backend instrumentation"))
	identity, err := IdentityFor(input.Manifest.Target, input.Manifest.Toolchain, description, input.Manifest.Environment)
	if err != nil {
		t.Fatal(err)
	}
	if identity.InstrumentationSHA256 == guideIdentity(t, input.Manifest).InstrumentationSHA256 {
		t.Fatal("provider instrumentation did not change corpus identity")
	}
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "typed outcomes", true: "evidence failure"}[fail], func(t *testing.T) {
			provider := corpusEvidenceProvider{operations: []backends.OperationOutcome{{Name: "provider.read", Result: 3}}}
			if fail {
				provider.err = errors.New("provider evidence rejected")
			}
			corpus, err := Open(context.Background(), t.TempDir(), identity, description, provider)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := corpus.Close(); err != nil {
					t.Error(err)
				}
			})
			added, err := corpus.Admit(context.Background(), Candidate{Artifact: input, Coverage: coverage}, func(context.Context, string) (ReplayResult, error) {
				return ReplayResult{Verified: true, Match: true}, nil
			})
			if fail {
				if added || !errors.Is(err, provider.err) || len(corpus.Snapshot().Entries) != 0 {
					t.Fatalf("Admit() = %t, %v, snapshot = %#v", added, err, corpus.Snapshot())
				}
				return
			}
			if err != nil || !added {
				t.Fatalf("Admit() = %t, %v", added, err)
			}
			want := []Feature{{Kind: FeatureOutcome, Value: "success/success/exit"}, {Kind: FeatureIOOutcome, Value: "provider.read/3"}}
			if got := corpus.Snapshot().Entries[0].Features; !slices.Equal(got, want) {
				t.Fatalf("features = %#v, want %#v", got, want)
			}
		})
	}
}
