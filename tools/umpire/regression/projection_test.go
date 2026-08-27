package regression

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequireProjectionIsIndependentOfWorkingDirectory(t *testing.T) {
	t.Chdir(t.TempDir())

	RequireProjection(t, Reference{
		FormatVersion: "umpire-experiment/v1",
		Identity:      "workflow-nexus.query.exact-action-caller-closure",
		FixturePath:   "model/Temporal/Feature/Nexus/Experimental/testdata/nexus-caller-closure-experiment-spec.json",
		Sources: []string{
			"Temporal/Feature/Nexus/Experimental/CallerClosure.lean",
		},
		Properties: []string{
			"workflow-nexus.property.caller-closure",
		},
		ObservationRequirements: []string{
			"nexus.observation.cancellation-delivered",
			"nexus.observation.pending-cancellation-count",
			"workflow-nexus.relation.owns-operation",
		},
		SemanticFingerprint: "sha256:8c2ba27730181616819a2bb4e0f083bc2ebfd6e3fc6df7717025539e95a3a46f",
	})
}

func TestLoadProjectionMatchesCompleteReference(t *testing.T) {
	repositoryRoot, reference, _ := newProjectionRepository(t)

	actual, err := loadProjection(repositoryRoot, reference)
	require.NoError(t, err)
	require.Equal(t, reference, actual)
}

func TestLoadProjectionDetectsEveryDisplayedFixtureField(t *testing.T) {
	tests := map[string]func(*testing.T, string, *fixtureEnvelope){
		"format": func(_ *testing.T, _ string, fixture *fixtureEnvelope) {
			fixture.FormatVersion = "umpire-experiment/unsupported"
		},
		"identity": func(_ *testing.T, _ string, fixture *fixtureEnvelope) {
			fixture.Plan.QueryIdentity = "query.changed"
		},
		"sources": func(t *testing.T, root string, fixture *fixtureEnvelope) {
			writeFile(t, filepath.Join(root, "model", "Temporal", "Changed.lean"), []byte("-- changed source\n"))
			fixture.Provenance.Sources[0].Path = "Temporal/Changed.lean"
		},
		"properties": func(_ *testing.T, _ string, fixture *fixtureEnvelope) {
			fixture.Properties[0].Identity = "property.changed"
		},
		"observation requirements": func(_ *testing.T, _ string, fixture *fixtureEnvelope) {
			fixture.ObservationRequirements[0] = "observation.changed"
		},
		"semantic fingerprint": func(_ *testing.T, _ string, fixture *fixtureEnvelope) {
			fixture.SemanticIdentity = "semantic-identity-changed"
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			repositoryRoot, reference, fixture := newProjectionRepository(t)
			mutate(t, repositoryRoot, &fixture)
			writeFixture(t, repositoryRoot, reference.FixturePath, fixture)

			actual, err := loadProjection(repositoryRoot, reference)
			if err == nil {
				require.NotEqual(t, reference, actual)
				return
			}
			require.Error(t, err)
		})
	}
}

func TestLoadProjectionRejectsInvalidFixtures(t *testing.T) {
	tests := map[string]func(*fixtureEnvelope){
		"missing format": func(fixture *fixtureEnvelope) {
			fixture.FormatVersion = ""
		},
		"missing identity": func(fixture *fixtureEnvelope) {
			fixture.Plan.QueryIdentity = ""
		},
		"missing sources": func(fixture *fixtureEnvelope) {
			fixture.Provenance.Sources = nil
		},
		"missing properties": func(fixture *fixtureEnvelope) {
			fixture.Properties = nil
		},
		"missing observation requirements": func(fixture *fixtureEnvelope) {
			fixture.ObservationRequirements = nil
		},
		"missing semantic identity": func(fixture *fixtureEnvelope) {
			fixture.SemanticIdentity = ""
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			repositoryRoot, reference, fixture := newProjectionRepository(t)
			mutate(&fixture)
			writeFixture(t, repositoryRoot, reference.FixturePath, fixture)

			_, err := loadProjection(repositoryRoot, reference)
			require.Error(t, err)
		})
	}

	malformed := map[string]string{
		"invalid JSON":            "{",
		"duplicate field":         `{"formatVersion":"first","formatVersion":"second"}`,
		"noncanonical field name": `{"FormatVersion":"umpire-experiment/v1"}`,
	}
	for name, encoded := range malformed {
		t.Run(name, func(t *testing.T) {
			repositoryRoot, reference, _ := newProjectionRepository(t)
			writeFile(
				t,
				filepath.Join(repositoryRoot, filepath.FromSlash(reference.FixturePath)),
				[]byte(encoded),
			)

			_, err := loadProjection(repositoryRoot, reference)
			require.ErrorContains(t, err, "decode fixture")
		})
	}
}

func TestLoadProjectionRejectsUnsafeFixturePaths(t *testing.T) {
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
			repositoryRoot, reference, _ := newProjectionRepository(t)
			mutate(t, repositoryRoot, &reference)

			_, err := loadProjection(repositoryRoot, reference)
			require.Error(t, err)
		})
	}
}

func TestLoadProjectionRejectsUnsafeLeanSourcePaths(t *testing.T) {
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
			fixture.Provenance.Sources = []fixtureSource{{Path: "escaped.lean"}}
		},
		"wrong kind": func(t *testing.T, root string, reference *Reference, fixture *fixtureEnvelope) {
			directory := filepath.Join(root, "model", "directory.lean")
			require.NoError(t, os.MkdirAll(directory, 0o755))
			reference.Sources = []string{"directory.lean"}
			fixture.Provenance.Sources = []fixtureSource{{Path: "directory.lean"}}
		},
		"nonexistent": func(_ *testing.T, _ string, reference *Reference, fixture *fixtureEnvelope) {
			reference.Sources = []string{"missing.lean"}
			fixture.Provenance.Sources = []fixtureSource{{Path: "missing.lean"}}
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			repositoryRoot, reference, fixture := newProjectionRepository(t)
			mutate(t, repositoryRoot, &reference, &fixture)
			writeFixture(t, repositoryRoot, reference.FixturePath, fixture)

			_, err := loadProjection(repositoryRoot, reference)
			require.Error(t, err)
		})
	}
}

func newProjectionRepository(t *testing.T) (string, Reference, fixtureEnvelope) {
	t.Helper()
	repositoryRoot := t.TempDir()
	source := "Temporal/Feature/Nexus/Experimental/CallerClosure.lean"
	writeFile(t, filepath.Join(repositoryRoot, "model", filepath.FromSlash(source)), []byte("-- canonical source\n"))
	fixture := fixtureEnvelope{
		FormatVersion: supportedFormatVersion,
		Plan: fixturePlan{
			QueryIdentity: "query.identity",
		},
		Properties: []fixtureProperty{
			{Identity: "property.identity"},
		},
		ObservationRequirements: []string{
			"observation.first",
			"observation.second",
		},
		SemanticIdentity: "semantic-identity",
		Provenance: fixtureProvenance{
			Sources: []fixtureSource{{Path: source}},
		},
	}
	reference := Reference{
		FormatVersion:           supportedFormatVersion,
		Identity:                fixture.Plan.QueryIdentity,
		FixturePath:             "fixtures/experiment-spec.json",
		Sources:                 []string{source},
		Properties:              []string{fixture.Properties[0].Identity},
		ObservationRequirements: append([]string(nil), fixture.ObservationRequirements...),
		SemanticFingerprint:     fingerprint(fixture.SemanticIdentity),
	}
	writeFixture(t, repositoryRoot, reference.FixturePath, fixture)
	return repositoryRoot, reference, fixture
}

func writeFixture(t *testing.T, repositoryRoot, relative string, fixture fixtureEnvelope) {
	t.Helper()
	encoded, err := json.Marshal(fixture)
	require.NoError(t, err)
	writeFile(t, filepath.Join(repositoryRoot, filepath.FromSlash(relative)), encoded)
}

func writeFile(t *testing.T, target string, contents []byte) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
	require.NoError(t, os.WriteFile(target, contents, 0o644))
}

func fingerprint(identity string) string {
	digest := sha256.Sum256([]byte(identity))
	return "sha256:" + hex.EncodeToString(digest[:])
}
