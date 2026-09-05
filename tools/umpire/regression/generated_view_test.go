package regression

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

func TestRequireGeneratedViewIsIndependentOfWorkingDirectory(t *testing.T) {
	t.Chdir(t.TempDir())

	RequireGeneratedView(t, Reference{
		FormatVersion: "umpire-experiment/v2",
		Identity:      "switch.query.exact-action",
		FixturePath:   "model/Umpire/Examples/testdata/switch-experiment-spec.json",
		Sources: []string{
			"Umpire/Examples/Switch.lean",
		},
		Properties: []string{
			"switch.property.flip-turns-on",
		},
		ObservationRequirements: []string{
			"switch.observation.power",
		},
		ArtifactChecksum: "sha256:ac3fde668a79ff0433106e28f8ec9579a36f9f7d0ab09845d01b563289b560fd",
	})
}

func TestLoadGeneratedViewMatchesCompleteReference(t *testing.T) {
	repositoryRoot, reference, _ := newGeneratedViewRepository(t)

	actual, err := loadGeneratedView(repositoryRoot, reference)
	require.NoError(t, err)
	require.Equal(t, reference, actual)
}

func TestLoadGeneratedViewDetectsEveryDisplayedFixtureField(t *testing.T) {
	tests := map[string]func(*testing.T, string, *fixtureEnvelope){
		"format": func(_ *testing.T, _ string, fixture *fixtureEnvelope) {
			fixture.FormatVersion = "umpire-experiment/unsupported"
		},
		"identity": func(_ *testing.T, _ string, fixture *fixtureEnvelope) {
			fixture.Plan.QueryDefinitionID = "query.changed"
		},
		"sources": func(t *testing.T, root string, fixture *fixtureEnvelope) {
			writeFile(t, filepath.Join(root, "model", "Temporal", "Changed.lean"), []byte("-- changed source\n"))
			fixture.Provenance.SourceLocations[0].Path = "Temporal/Changed.lean"
		},
		"properties": func(_ *testing.T, _ string, fixture *fixtureEnvelope) {
			fixture.Properties[0].DefinitionID = "property.changed"
		},
		"observation requirements": func(_ *testing.T, _ string, fixture *fixtureEnvelope) {
			fixture.ObservationRequirementDefinitionIDs[0] = "observation.changed"
		},
		"artifact checksum": func(_ *testing.T, _ string, fixture *fixtureEnvelope) {
			fixture.ArtifactChecksum = "semantic-identity-changed"
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			repositoryRoot, reference, fixture := newGeneratedViewRepository(t)
			mutate(t, repositoryRoot, &fixture)
			writeFixture(t, repositoryRoot, reference.FixturePath, fixture)

			actual, err := loadGeneratedView(repositoryRoot, reference)
			if err == nil {
				require.NotEqual(t, reference, actual)
				return
			}
			require.Error(t, err)
		})
	}
}

func TestLoadGeneratedViewRejectsInvalidFixtures(t *testing.T) {
	tests := map[string]func(*fixtureEnvelope){
		"missing format": func(fixture *fixtureEnvelope) {
			fixture.FormatVersion = ""
		},
		"missing identity": func(fixture *fixtureEnvelope) {
			fixture.Plan.QueryDefinitionID = ""
		},
		"missing sources": func(fixture *fixtureEnvelope) {
			fixture.Provenance.SourceLocations = nil
		},
		"missing properties": func(fixture *fixtureEnvelope) {
			fixture.Properties = nil
		},
		"missing observation requirements": func(fixture *fixtureEnvelope) {
			fixture.ObservationRequirementDefinitionIDs = nil
		},
		"missing artifact checksum": func(fixture *fixtureEnvelope) {
			fixture.ArtifactChecksum = ""
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			repositoryRoot, reference, fixture := newGeneratedViewRepository(t)
			mutate(&fixture)
			writeFixture(t, repositoryRoot, reference.FixturePath, fixture)

			_, err := loadGeneratedView(repositoryRoot, reference)
			require.Error(t, err)
		})
	}

	malformed := map[string]string{
		"invalid JSON":            "{",
		"duplicate field":         `{"formatVersion":"first","formatVersion":"second"}`,
		"noncanonical field name": `{"FormatVersion":"umpire-experiment/v2"}`,
	}
	for name, encoded := range malformed {
		t.Run(name, func(t *testing.T) {
			repositoryRoot, reference, _ := newGeneratedViewRepository(t)
			writeFile(
				t,
				filepath.Join(repositoryRoot, filepath.FromSlash(reference.FixturePath)),
				[]byte(encoded),
			)

			_, err := loadGeneratedView(repositoryRoot, reference)
			require.ErrorContains(t, err, "decode fixture")
		})
	}
}

func TestLoadGeneratedViewRejectsUnsafeFixturePaths(t *testing.T) {
	tests := map[string]func(*testing.T, string, *Reference){
		"empty": func(_ *testing.T, _ string, reference *Reference) {
			reference.FixturePath = ""
		},
		"absolute": func(_ *testing.T, root string, reference *Reference) {
			reference.FixturePath = filepath.Join(root, "fixture.json")
		},
		"traversal": func(_ *testing.T, _ string, reference *Reference) {
			reference.FixturePath = "../fixture.json"
		},
		"symlink escape": func(t *testing.T, root string, reference *Reference) {
			external := filepath.Join(t.TempDir(), "fixture.json")
			writeFile(t, external, []byte("{}"))
			link := filepath.Join(root, "fixtures", "escaped.json")
			require.NoError(t, os.Symlink(external, link))
			reference.FixturePath = "fixtures/escaped.json"
		},
		"wrong kind": func(t *testing.T, root string, reference *Reference) {
			directory := filepath.Join(root, "fixtures", "directory.json")
			require.NoError(t, os.MkdirAll(directory, 0o755))
			reference.FixturePath = "fixtures/directory.json"
		},
		"nonexistent": func(_ *testing.T, _ string, reference *Reference) {
			reference.FixturePath = "fixtures/missing.json"
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			repositoryRoot, reference, _ := newGeneratedViewRepository(t)
			mutate(t, repositoryRoot, &reference)

			_, err := loadGeneratedView(repositoryRoot, reference)
			require.Error(t, err)
		})
	}
}

func TestLoadGeneratedViewRejectsUnsafeLeanSourcePaths(t *testing.T) {
	tests := map[string]func(*testing.T, string, *Reference, *fixtureEnvelope){
		"empty": func(_ *testing.T, _ string, reference *Reference, _ *fixtureEnvelope) {
			reference.Sources = []string{""}
		},
		"absolute": func(_ *testing.T, root string, reference *Reference, _ *fixtureEnvelope) {
			reference.Sources = []string{filepath.Join(root, "source.lean")}
		},
		"traversal": func(_ *testing.T, _ string, reference *Reference, _ *fixtureEnvelope) {
			reference.Sources = []string{"../source.lean"}
		},
		"symlink escape": func(t *testing.T, root string, reference *Reference, fixture *fixtureEnvelope) {
			external := filepath.Join(t.TempDir(), "external.lean")
			writeFile(t, external, []byte("-- external source\n"))
			link := filepath.Join(root, "model", "escaped.lean")
			require.NoError(t, os.Symlink(external, link))
			reference.Sources = []string{"escaped.lean"}
			fixture.Provenance.SourceLocations[0].Path = "escaped.lean"
		},
		"wrong kind": func(t *testing.T, root string, reference *Reference, fixture *fixtureEnvelope) {
			directory := filepath.Join(root, "model", "directory.lean")
			require.NoError(t, os.MkdirAll(directory, 0o755))
			reference.Sources = []string{"directory.lean"}
			fixture.Provenance.SourceLocations[0].Path = "directory.lean"
		},
		"nonexistent": func(_ *testing.T, _ string, reference *Reference, fixture *fixtureEnvelope) {
			reference.Sources = []string{"missing.lean"}
			fixture.Provenance.SourceLocations[0].Path = "missing.lean"
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			repositoryRoot, reference, fixture := newGeneratedViewRepository(t)
			mutate(t, repositoryRoot, &reference, &fixture)
			writeFixture(t, repositoryRoot, reference.FixturePath, fixture)

			_, err := loadGeneratedView(repositoryRoot, reference)
			require.Error(t, err)
		})
	}
}

func newGeneratedViewRepository(t *testing.T) (string, Reference, fixtureEnvelope) {
	t.Helper()
	repositoryRoot := t.TempDir()
	source := "Umpire/Examples/Switch.lean"
	writeFile(t, filepath.Join(repositoryRoot, "model", filepath.FromSlash(source)), []byte("-- canonical source\n"))
	realRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	encoded, err := os.ReadFile(filepath.Join(realRoot, "model/Umpire/Examples/testdata/switch-experiment-spec.json"))
	require.NoError(t, err)
	fixture, err := artifactv2.DecodeExperiment(encoded)
	require.NoError(t, err)
	reference := Reference{
		FormatVersion:           supportedFormatVersion,
		Identity:                fixture.Plan.QueryDefinitionID,
		FixturePath:             "fixtures/experiment-spec.json",
		Sources:                 []string{source},
		Properties:              []string{fixture.Properties[0].DefinitionID},
		ObservationRequirements: append([]string(nil), fixture.ObservationRequirementDefinitionIDs...),
		ArtifactChecksum:        fixture.ArtifactChecksum,
	}
	writeFixture(t, repositoryRoot, reference.FixturePath, fixture)
	return repositoryRoot, reference, fixture
}

func writeFixture(t *testing.T, repositoryRoot, relative string, fixture fixtureEnvelope) {
	t.Helper()
	encoded, err := artifactv2.CanonicalExperimentBytes(fixture)
	require.NoError(t, err)
	writeFile(t, filepath.Join(repositoryRoot, filepath.FromSlash(relative)), encoded)
}

func writeFile(t *testing.T, target string, contents []byte) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
	require.NoError(t, os.WriteFile(target, contents, 0o644))
}
