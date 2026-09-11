package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/common/artifactio"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestRunGenerationPublishesExactlySixCompleteClasses(t *testing.T) {
	configuration := generationConfig{RepositoryRoot: t.TempDir(), OutputRoot: t.TempDir()}
	entries := productionManifest()
	rendered := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		encoded, err := protojson.Marshal(&testpilotspb.Case{CaseId: entry.CaseID})
		require.NoError(t, err)
		rendered[entry.RendererArg] = encoded
	}
	var published bool
	dependencies := generationDependencies{
		Render: func(_ string, arguments ...string) (rendererOutput, error) {
			return rendererOutput{Stdout: slices.Clone(rendered[arguments[0]])}, nil
		},
		Publish: func(set artifactio.Set, root string, artifacts map[string][]byte, validate func(string) error) error {
			published = true
			require.Equal(t, []string{"common/testing/testpilot/testdata/case-runtime-conformance"}, set.Roots)
			require.Len(t, set.Paths, 12)
			require.Len(t, artifacts, 12)
			return set.Publish(root, artifacts, validate)
		},
	}

	require.NoError(t, runGeneration(configuration, entries, dependencies))
	require.True(t, published)
	for _, entry := range entries {
		caseBytes, err := os.ReadFile(filepath.Join(configuration.OutputRoot, filepath.FromSlash(casePath(entry.Class))))
		require.NoError(t, err)
		stored, err := persistedForm(rendered[entry.RendererArg])
		require.NoError(t, err)
		require.Equal(t, stored, caseBytes)
		expectedBytes, err := os.ReadFile(filepath.Join(configuration.OutputRoot, filepath.FromSlash(expectedPath(entry.Class))))
		require.NoError(t, err)
		canonical, err := marshalExpected(entry.Expected)
		require.NoError(t, err)
		require.Equal(t, expectedBytes, canonical)
	}
}

// registeredCases is a fake renderer's registry: the Case IDs it enumerates through `--list` and
// the fixture name each is stored under.
type registeredCases map[string]string

func (r registeredCases) listing() []byte {
	ids := make([]string, 0, len(r))
	for id := range r {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	lines := make([]string, 0, len(ids))
	for _, id := range ids {
		lines = append(lines, id+" "+r[id])
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func (r registeredCases) render(_ string, arguments ...string) (rendererOutput, error) {
	switch {
	case len(arguments) == 1 && arguments[0] == "--list":
		return rendererOutput{Stdout: r.listing()}, nil
	case len(arguments) == 2 && arguments[0] == "--render":
		if _, registered := r[arguments[1]]; !registered {
			return rendererOutput{Stderr: []byte("unknown Case")}, errors.New("exit status 1")
		}
		encoded, err := protojson.Marshal(&testpilotspb.Case{CaseId: arguments[1]})
		return rendererOutput{Stdout: encoded}, err
	case len(arguments) == 1 && arguments[0] == syntheticEntry().RendererArgs[0]:
		encoded, err := protojson.Marshal(&testpilotspb.Case{CaseId: syntheticEntry().CaseID})
		return rendererOutput{Stdout: encoded}, err
	}
	return rendererOutput{Stderr: []byte("unexpected renderer arguments")}, errors.New("exit status 1")
}

func fakeRegistry() registeredCases {
	return registeredCases{
		"temporal.case.async-nexus":     "async-nexus",
		"temporal.case.get-system-info": "get-system-info",
		"temporal.case.typed-nexus":     "typed-nexus",
		"temporal.case.typed-unary":     "typed-unary",
		"temporal.case.worker-outage":   "worker-outage",
	}
}

// The generator asks the renderer what Cases exist. A Case added to a Model file therefore reaches
// the fixture set without any edit here, and none is dropped for being absent from a table.
func TestRunFunctionalGenerationPublishesEveryRegisteredCaseWithoutATable(t *testing.T) {
	configuration := generationConfig{RepositoryRoot: t.TempDir(), OutputRoot: t.TempDir()}
	registry := fakeRegistry()
	registry["temporal.case.newly-authored"] = "newly-authored"
	stale := filepath.Join(configuration.OutputRoot, filepath.FromSlash(functionalFixtureRoot), "stale.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(stale), 0o700))
	require.NoError(t, os.WriteFile(stale, []byte("stale"), 0o600))

	require.NoError(t, runFunctionalGeneration(configuration, generationDependencies{
		Render:  registry.render,
		Publish: defaultGenerationDependencies().Publish,
	}))

	_, err := os.Stat(stale)
	require.ErrorIs(t, err, os.ErrNotExist)
	files, err := os.ReadDir(filepath.Join(configuration.OutputRoot, filepath.FromSlash(functionalFixtureRoot)))
	require.NoError(t, err)
	// Every registered Case, plus the synthetic fixture the registry does not model.
	require.Len(t, files, len(registry)+1)
	for caseID, fixture := range registry {
		encoded, err := os.ReadFile(filepath.Join(configuration.OutputRoot,
			filepath.FromSlash(functionalFixtureRoot), fixture+"-case.json"))
		require.NoError(t, err)
		decoded, err := testpilot.DecodeCaseProtoJSON(encoded)
		require.NoError(t, err)
		require.Equal(t, caseID, decoded.GetCaseId())
	}
	_, err = os.Stat(filepath.Join(configuration.OutputRoot,
		filepath.FromSlash(functionalCasePath(syntheticEntry()))))
	require.NoError(t, err)
}

func TestFunctionalEntriesRejectAnUnreadableListing(t *testing.T) {
	for _, probe := range []struct {
		name    string
		render  func(string, ...string) (rendererOutput, error)
		message string
	}{
		{
			name: "malformed line",
			render: func(string, ...string) (rendererOutput, error) {
				return rendererOutput{Stdout: []byte("temporal.case.async-nexus\n")}, nil
			},
			message: `is not "<case-id> <fixture-name>"`,
		},
		{
			name: "empty listing",
			render: func(string, ...string) (rendererOutput, error) {
				return rendererOutput{Stdout: []byte("\n")}, nil
			},
			message: "renderer produced an empty artifact",
		},
		{
			name: "unstable listing",
			render: func() func(string, ...string) (rendererOutput, error) {
				calls := 0
				return func(string, ...string) (rendererOutput, error) {
					calls++
					if calls%2 == 0 {
						return rendererOutput{Stdout: []byte("b b\na a\n")}, nil
					}
					return rendererOutput{Stdout: []byte("a a\nb b\n")}, nil
				}
			}(),
			message: "non-deterministic bytes",
		},
	} {
		t.Run(probe.name, func(t *testing.T) {
			_, err := functionalEntries(t.TempDir(), generationDependencies{Render: probe.render})

			require.ErrorContains(t, err, probe.message)
		})
	}
}

func TestParseGenerationConfigSelectsExplicitFunctionalMode(t *testing.T) {
	configuration, err := parseGenerationConfig([]string{"--mode", "functional"})
	require.NoError(t, err)
	require.Equal(t, generationModeFunctional, configuration.Mode)

	_, err = parseGenerationConfig([]string{"--mode", "unknown"})
	require.ErrorContains(t, err, "unknown generation mode")
}

func TestRunFunctionalGenerationPreservesPublishedSetWhenPublicationFails(t *testing.T) {
	configuration := generationConfig{RepositoryRoot: t.TempDir(), OutputRoot: t.TempDir()}
	registry := fakeRegistry()
	fixtures := append(registry.fixtureNames(), syntheticEntry().Filename)
	for _, fixture := range fixtures {
		path := filepath.Join(configuration.OutputRoot, filepath.FromSlash(functionalFixtureRoot), fixture)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte("original"), 0o600))
	}

	err := runFunctionalGeneration(configuration, generationDependencies{
		Render: registry.render,
		Publish: func(artifactio.Set, string, map[string][]byte, func(string) error) error {
			return errors.New("injected publication failure")
		},
	})

	require.ErrorContains(t, err, "injected publication failure")
	for _, fixture := range fixtures {
		encoded, readErr := os.ReadFile(filepath.Join(configuration.OutputRoot,
			filepath.FromSlash(functionalFixtureRoot), fixture))
		require.NoError(t, readErr)
		require.Equal(t, []byte("original"), encoded)
	}
}

func (r registeredCases) fixtureNames() []string {
	names := make([]string, 0, len(r))
	for _, fixture := range r {
		names = append(names, fixture+"-case.json")
	}
	slices.Sort(names)
	return names
}

func TestRunFunctionalGenerationRejectsRendererFailureOrNondeterminismBeforePublication(t *testing.T) {
	registry := fakeRegistry()
	for _, probe := range []struct {
		name   string
		render func(string, ...string) (rendererOutput, error)
	}{
		{
			name: "renderer failure",
			render: func(_ string, arguments ...string) (rendererOutput, error) {
				if len(arguments) == 1 && arguments[0] == "--list" {
					return registry.render("", arguments...)
				}
				return rendererOutput{Stderr: []byte("failed")}, errors.New("exit status 1")
			},
		},
		{
			name: "non-deterministic bytes",
			render: func() func(string, ...string) (rendererOutput, error) {
				calls := make(map[string]int)
				return func(root string, arguments ...string) (rendererOutput, error) {
					if len(arguments) == 1 && arguments[0] == "--list" {
						return registry.render(root, arguments...)
					}
					key := strings.Join(arguments, " ")
					calls[key]++
					output, err := registry.render(root, arguments...)
					if err == nil && calls[key]%2 == 0 {
						output.Stdout = append(slices.Clone(output.Stdout), '\n')
					}
					return output, err
				}
			}(),
		},
	} {
		t.Run(probe.name, func(t *testing.T) {
			published := false

			err := runFunctionalGeneration(
				generationConfig{RepositoryRoot: t.TempDir(), OutputRoot: t.TempDir()},
				generationDependencies{
					Render: probe.render,
					Publish: func(artifactio.Set, string, map[string][]byte, func(string) error) error {
						published = true
						return nil
					},
				})

			require.Error(t, err)
			require.False(t, published)
		})
	}
}

func TestValidateFunctionalArtifactsRejectsStaleFile(t *testing.T) {
	entries := fakeFunctionalEntries()
	artifacts := make(map[string][]byte, len(entries)+1)
	for _, entry := range entries {
		encoded, err := protojson.Marshal(&testpilotspb.Case{CaseId: entry.CaseID})
		require.NoError(t, err)
		artifacts[functionalCasePath(entry)] = encoded
	}
	artifacts[filepath.ToSlash(filepath.Join(functionalFixtureRoot, "stale.json"))] = []byte("stale")

	require.ErrorContains(t, validateFunctionalArtifacts(entries, artifacts),
		fmt.Sprintf("has %d files, want %d", len(entries)+1, len(entries)))
}

func TestRunGenerationRejectsIncompleteManifestAndRendererFailureBeforePublication(t *testing.T) {
	entries := productionManifest()
	for _, test := range []struct {
		name    string
		entries []manifestEntry
		render  func(string, ...string) (rendererOutput, error)
	}{
		{name: "incomplete manifest", entries: entries[:5], render: func(string, ...string) (rendererOutput, error) { return rendererOutput{}, nil }},
		{name: "renderer failure", entries: entries, render: func(string, ...string) (rendererOutput, error) {
			return rendererOutput{Stderr: []byte("failed")}, errors.New("exit status 1")
		}},
		{name: "renderer contradiction", entries: entries, render: func(string, ...string) (rendererOutput, error) {
			return rendererOutput{Stdout: []byte("partial")}, errors.New("exit status 1")
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			published := false
			err := runGeneration(generationConfig{RepositoryRoot: t.TempDir(), OutputRoot: t.TempDir()}, test.entries, generationDependencies{
				Render: test.render,
				Publish: func(artifactio.Set, string, map[string][]byte, func(string) error) error {
					published = true
					return nil
				},
			})
			require.Error(t, err)
			require.False(t, published)
		})
	}
}

func TestRunGenerationRejectsNondeterministicRenderingBeforePublication(t *testing.T) {
	entries := productionManifest()
	encodedByArgument := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		encoded, err := protojson.Marshal(&testpilotspb.Case{CaseId: entry.CaseID})
		require.NoError(t, err)
		encodedByArgument[entry.RendererArg] = encoded
	}
	calls := make(map[string]int)
	published := false
	err := runGeneration(
		generationConfig{RepositoryRoot: t.TempDir(), OutputRoot: t.TempDir()},
		entries,
		generationDependencies{
			Render: func(_ string, arguments ...string) (rendererOutput, error) {
				calls[arguments[0]]++
				encoded := slices.Clone(encodedByArgument[arguments[0]])
				if calls[arguments[0]]%2 == 0 {
					encoded = append(encoded, '\n')
				}
				return rendererOutput{Stdout: encoded}, nil
			},
			Publish: func(artifactio.Set, string, map[string][]byte, func(string) error) error {
				published = true
				return nil
			},
		},
	)
	require.ErrorContains(t, err, "non-deterministic bytes")
	require.False(t, published)
}

// The persisted form is what every generated Testpilot JSON file is stored as: two-space
// indentation and exactly one trailing newline, with key order and string escapes untouched.
func TestPersistedFormIndentsWithoutReorderingOrReescaping(t *testing.T) {
	compact := []byte(`{"zeta":"a\u0041b\n","alpha":[1,{"inner":"x/y"}],"empty":{}}`)

	stored, err := persistedForm(compact)
	require.NoError(t, err)

	require.Equal(t, []string{
		"{",
		`  "zeta": "a\u0041b\n",`,
		`  "alpha": [`,
		"    1,",
		"    {",
		`      "inner": "x/y"`,
		"    }",
		"  ],",
		`  "empty": {}`,
		"}",
	}, strings.Split(strings.TrimSuffix(string(stored), "\n"), "\n"))

	repeated, err := persistedForm(stored)
	require.NoError(t, err)
	require.Equal(t, stored, repeated, "the persisted form is idempotent")

	require.Equal(t, byte('\n'), stored[len(stored)-1])
	require.NotEqual(t, byte('\n'), stored[len(stored)-2])
}

func TestRequirePersistedFormRejectsCompactButValidJSONNamingTheFile(t *testing.T) {
	err := requirePersistedForm("tests/testcore/testpilot/testdata/async-nexus-case.json",
		[]byte(`{"caseId":"temporal.case.async-nexus"}`))

	require.ErrorContains(t, err, "tests/testcore/testpilot/testdata/async-nexus-case.json")
	require.ErrorContains(t, err, "not in persisted form")
}

// A staged fixture that is valid JSON but stored compact fails validation by name, rather than
// surfacing only as an unexplained `diff -ru` hunk.
func TestValidateFunctionalArtifactsRejectsCompactStagedFixture(t *testing.T) {
	entries := fakeFunctionalEntries()
	artifacts := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		encoded, err := protojson.Marshal(&testpilotspb.Case{CaseId: entry.CaseID})
		require.NoError(t, err)
		stored, err := persistedForm(encoded)
		require.NoError(t, err)
		artifacts[functionalCasePath(entry)] = stored
	}
	compact, err := protojson.Marshal(&testpilotspb.Case{CaseId: entries[0].CaseID})
	require.NoError(t, err)
	artifacts[functionalCasePath(entries[0])] = compact

	err = validateFunctionalArtifacts(entries, artifacts)

	require.ErrorContains(t, err, entries[0].Filename)
	require.ErrorContains(t, err, "not in persisted form")
}

// fakeFunctionalEntries is what functionalEntries would return for the fake registry, without
// running a renderer.
func fakeFunctionalEntries() []functionalEntry {
	registry := fakeRegistry()
	ids := make([]string, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	entries := make([]functionalEntry, 0, len(registry)+1)
	for _, id := range ids {
		entries = append(entries, functionalEntry{
			RendererArgs: []string{"--render", id},
			CaseID:       id,
			Filename:     registry[id] + "-case.json",
		})
	}
	return append(entries, syntheticEntry())
}
