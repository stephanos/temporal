package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckRepositoryAcceptsCompleteConfinedRegistryWithoutWrites(t *testing.T) {
	root, index := validRepositoryFixture(t)
	before := repositorySnapshot(t, root)

	require.Equal(t, []string{}, checkRepository(root, index))
	require.Equal(t, before, repositorySnapshot(t, root))
}

func TestCheckRepositoryReportsSortedStableFindings(t *testing.T) {
	root, index := validRepositoryFixture(t)
	index.Documents[2].AuthorityParents = []string{".plans/Missing.md", ".plans/B.md"}
	index.FlowSpecs[0].Ready = false
	index.FlowSpecs[1].SpecDependencies = []string{"fn-9-missing", "fn-1-example"}
	index.FlowSpecs[2].Phase = "support"

	first := index
	first.Documents = []documentEntry{index.Documents[2], index.Documents[0], index.Documents[1]}
	first.FlowSpecs = []flowSpecEntry{index.FlowSpecs[2], index.FlowSpecs[0], index.FlowSpecs[1]}
	second := index
	second.Documents = []documentEntry{index.Documents[1], index.Documents[2], index.Documents[0]}
	second.FlowSpecs = []flowSpecEntry{index.FlowSpecs[1], index.FlowSpecs[2], index.FlowSpecs[0]}

	want := []string{
		`document .plans/C.md: authority parent ".plans/Missing.md" is not registered`,
		`document .plans/C.md: authorityParents must be sorted and unique`,
		`documents: entries must be sorted by path`,
		`flow spec fn-1-example: ready is false; Flow records true`,
		`flow spec fn-2-support: dependency "fn-9-missing" is not registered`,
		`flow spec fn-2-support: specDependencies [fn-9-missing fn-1-example]; Flow records [fn-1-example]`,
		`flow spec fn-2-support: specDependencies must be sorted and unique`,
		`flow spec fn-3-other: scope other requires disposition out-of-scope and phase none`,
		`flowSpecs: entries must be sorted by id`,
	}
	require.Equal(t, want, checkRepository(root, first))
	require.Equal(t, want, checkRepository(root, second))
}

func TestCheckRepositoryValidatesCoverageGraphAndLifecycle(t *testing.T) {
	root, index := validRepositoryFixture(t)
	writeFixtureFile(t, root, ".plans/Unregistered.md", "# Unregistered\n")
	index.Documents = append(index.Documents, documentEntry{
		Path: ".plans/Missing.md", Lifecycle: "active", Authority: "descriptive",
		AuthorityParents: []string{".plans/Missing.md"}, AllowedMissingLinks: []allowedMissingLink{},
	})
	index.Documents[0].AuthorityParents = []string{".plans/B.md"}
	index.Documents[1].SupersededBy = stringPointer(".plans/Missing.md")
	index.Documents[2].Lifecycle = "unclassified"
	index.Documents[2].Authority = "unclassified"
	index.Documents = append(index.Documents, index.Documents[0])

	want := []string{
		`authority graph: expected exactly one normative-rules document; found 0`,
		`document .plans/A.md: not registered`,
		`document .plans/A.md: registered more than once`,
		`document .plans/B.md: authority parent ".plans/A.md" is not registered`,
		`document .plans/B.md: supersededBy must be null unless lifecycle is superseded`,
		`document .plans/C.md: authority unclassified is not permitted in a checked registry`,
		`document .plans/C.md: lifecycle unclassified is not permitted in a checked registry`,
		`document .plans/Missing.md: authority parent must not reference itself`,
		`document .plans/Missing.md: registered file does not exist`,
		`document .plans/Unregistered.md: not registered`,
		`documents: entries must be sorted by path`,
	}
	require.Equal(t, want, checkRepository(root, index))
}

func TestCheckRepositoryRejectsSupersessionCycles(t *testing.T) {
	tests := []struct {
		name string
		edit func(*planIndex)
		want []string
	}{
		{
			name: "self reference",
			edit: func(index *planIndex) {
				index.Documents[1].Lifecycle = "superseded"
				index.Documents[1].SupersededBy = stringPointer(".plans/B.md")
			},
			want: []string{`document .plans/B.md: supersededBy must not reference itself`},
		},
		{
			name: "cycle",
			edit: func(index *planIndex) {
				index.Documents[1].Lifecycle = "superseded"
				index.Documents[1].SupersededBy = stringPointer(".plans/C.md")
				index.Documents[2].Lifecycle = "superseded"
				index.Documents[2].SupersededBy = stringPointer(".plans/B.md")
			},
			want: []string{`supersession graph: cycle .plans/B.md -> .plans/C.md -> .plans/B.md`},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, index := validRepositoryFixture(t)
			test.edit(&index)
			require.Equal(t, test.want, checkRepository(root, index))
		})
	}
}

func TestCheckRepositoryRejectsAuthorityCycles(t *testing.T) {
	root, index := validRepositoryFixture(t)
	index.Documents[0].AuthorityParents = []string{".plans/B.md"}

	require.Equal(t, []string{
		`authority graph: cycle .plans/A.md -> .plans/B.md -> .plans/A.md`,
	}, checkRepository(root, index))
}

func TestCheckRepositoryValidatesMarkdownLinksAndAllowlist(t *testing.T) {
	root, index := validRepositoryFixture(t)
	index.Documents[0].AllowedMissingLinks = []allowedMissingLink{}
	writeFixtureFile(t, root, ".plans/C.md", "# Notes\n[missing anchor](B.md#absent)\n[missing reference][missing]\n[missing]: Missing.md\n[external](https://example.com/no-check)\n")

	want := []string{
		`document .plans/A.md: local link ".plans/Future.md#future" is missing and not allowlisted`,
		`document .plans/C.md: anchor "absent" is missing from .plans/B.md`,
		`document .plans/C.md: local link ".plans/Missing.md" is missing and not allowlisted`,
	}
	require.Equal(t, want, checkRepository(root, index))
}

func TestCheckRepositoryAllowsMissingAnchorOnExistingTarget(t *testing.T) {
	root, index := validRepositoryFixture(t)
	writeFixtureFile(t, root, ".plans/A.md", "# Authority\n[future anchor](B.md#future)\n")
	index.Documents[0].AllowedMissingLinks = []allowedMissingLink{{
		Target: ".plans/B.md", Reason: "historical heading", Anchor: stringPointer("future"),
	}}

	require.Equal(t, []string{}, checkRepository(root, index))
}

func TestCheckRepositoryParsesRenderedMarkdownLinksAndAnchors(t *testing.T) {
	root, index := validRepositoryFixture(t)
	writeFixtureFile(t, root, ".plans/A.md", "# Authority\n`[inline code](MissingInline.md)`\n```markdown\n# Fake\n```not a closing fence\n[fenced code](MissingFence.md)\n```\n[balanced](Target_(one).md#target-one)\n")
	writeFixtureFile(t, root, ".plans/B.md", "# Delivery\n[not a rendered heading](A.md#fake)\n")
	writeFixtureFile(t, root, ".plans/Target_(one).md", "# Target one\n")
	index.Documents[0].AllowedMissingLinks = []allowedMissingLink{}
	index.Documents = append(index.Documents, documentEntry{
		Path: ".plans/Target_(one).md", Lifecycle: "reference", Authority: "descriptive",
		AuthorityParents: []string{".plans/B.md"}, AllowedMissingLinks: []allowedMissingLink{},
	})

	require.Equal(t, []string{
		`document .plans/B.md: anchor "fake" is missing from .plans/A.md`,
	}, checkRepository(root, index))
}

func TestCheckRepositoryIsStableAcrossConflictingDuplicateRows(t *testing.T) {
	root, index := validRepositoryFixture(t)
	conflictingDocument := index.Documents[0]
	conflictingDocument.Lifecycle = "historical"
	conflictingDocument.Authority = "historical"
	conflictingFlowSpec := index.FlowSpecs[0]
	conflictingFlowSpec.Ready = false

	first := index
	first.Documents = append(first.Documents, conflictingDocument)
	first.FlowSpecs = append(first.FlowSpecs, conflictingFlowSpec)
	second := index
	second.Documents = append([]documentEntry{conflictingDocument}, second.Documents...)
	second.FlowSpecs = append([]flowSpecEntry{conflictingFlowSpec}, second.FlowSpecs...)

	want := []string{
		`authority graph: expected exactly one normative-rules document; found 0`,
		`document .plans/A.md: not registered`,
		`document .plans/A.md: registered more than once`,
		`document .plans/B.md: authority parent ".plans/A.md" is not registered`,
		`documents: entries must be sorted by path`,
		`flow spec fn-1-example: not registered`,
		`flow spec fn-1-example: registered more than once`,
		`flow spec fn-2-support: dependency "fn-1-example" is not registered`,
		`flowSpecs: entries must be sorted by id`,
	}
	require.Equal(t, want, checkRepository(root, first))
	require.Equal(t, want, checkRepository(root, second))
}

func TestCheckRepositoryRejectsNoncanonicalAndEscapingPaths(t *testing.T) {
	root, index := validRepositoryFixture(t)
	outside := filepath.Join(t.TempDir(), "outside.md")
	require.NoError(t, os.WriteFile(outside, []byte("# Outside\n"), 0o600))
	require.NoError(t, os.Symlink(outside, filepath.Join(root, ".plans", "Alias.md")))
	index.Documents = append(index.Documents,
		documentEntry{Path: "/absolute.md", Lifecycle: "reference", Authority: "descriptive", AuthorityParents: []string{}, AllowedMissingLinks: []allowedMissingLink{}},
		documentEntry{Path: ".plans/../escape.md", Lifecycle: "reference", Authority: "descriptive", AuthorityParents: []string{}, AllowedMissingLinks: []allowedMissingLink{}},
		documentEntry{Path: ".plans/Alias.md", Lifecycle: "reference", Authority: "descriptive", AuthorityParents: []string{}, AllowedMissingLinks: []allowedMissingLink{}},
	)

	findings := checkRepository(root, index)
	require.Equal(t, []string{
		`document .plans/../escape.md: path must be a normalized repository-relative .plans/*.md path`,
		`document .plans/Alias.md: path resolves outside repository root`,
		`document /absolute.md: path must be a normalized repository-relative .plans/*.md path`,
		`documents: entries must be sorted by path`,
	}, findings)
}

func TestCheckRepositoryRejectsEscapingMarkdownAndAllowlistTargets(t *testing.T) {
	root, index := validRepositoryFixture(t)
	outside := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(root, ".plans", "escape")))
	writeFixtureFile(t, root, ".plans/A.md", "# Authority\n[absolute](/outside.md)\n[escaped](escape/Future.md#future)\n")
	index.Documents[0].AllowedMissingLinks = []allowedMissingLink{{
		Target: ".plans/escape/Future.md", Reason: "must not escape", Anchor: stringPointer("future"),
	}}

	want := []string{
		`document .plans/A.md: allowed missing target ".plans/escape/Future.md" is not repository-confined`,
		`document .plans/A.md: local link ".plans/escape/Future.md#future" is not repository-confined`,
		`document .plans/A.md: local link "/outside.md" is not repository-confined`,
	}
	require.Equal(t, want, checkRepository(root, index))
}

func TestCheckRepositoryRejectsNoncanonicalReferences(t *testing.T) {
	root, index := validRepositoryFixture(t)
	index.Documents[1].AuthorityParents = []string{"../A.md"}
	index.Documents[2].Lifecycle = "superseded"
	index.Documents[2].SupersededBy = stringPointer("/A.md")
	index.FlowSpecs[1].SpecDependencies = []string{"not-a-flow-id"}

	findings := checkRepository(root, index)
	require.Equal(t, []string{
		`document .plans/B.md: authority parent "../A.md" is not a normalized registered document path`,
		`document .plans/C.md: supersededBy target "/A.md" is not a normalized registered document path`,
		`flow spec fn-2-support: dependency "not-a-flow-id" is not a canonical Flow spec ID`,
		`flow spec fn-2-support: specDependencies [not-a-flow-id]; Flow records [fn-1-example]`,
	}, findings)
}

func TestCheckRepositoryValidatesFlowStateAndCrossFields(t *testing.T) {
	root, index := validRepositoryFixture(t)
	index.FlowSpecs[0].Status = "done"
	index.FlowSpecs[1].Disposition = "deferred"
	index.FlowSpecs[1].Ready = true
	index.FlowSpecs[1].CompletionReview = "ship"
	index.FlowSpecs[2].Disposition = "unclassified"

	want := []string{
		`flow spec fn-1-example: retained disposition requires status open`,
		`flow spec fn-1-example: status done; Flow records open`,
		`flow spec fn-2-support: deferred disposition requires status open, ready false, and completionReview other than ship`,
		`flow spec fn-2-support: ready is true; Flow records false`,
		`flow spec fn-3-other: disposition unclassified is not permitted in a checked registry`,
		`flow spec fn-3-other: scope other requires disposition out-of-scope and phase none`,
	}
	require.Equal(t, want, checkRepository(root, index))
}

func TestCheckRepositoryRejectsRetainedDependenciesOnDeferredScope(t *testing.T) {
	tests := []struct {
		name         string
		dependencies map[string][]string
		wantPath     string
	}{
		{
			name:         "direct",
			dependencies: map[string][]string{"fn-1-example": {"fn-4-deferred"}},
			wantPath:     "fn-1-example -> fn-4-deferred",
		},
		{
			name: "transitive",
			dependencies: map[string][]string{
				"fn-1-example": {"fn-2-support"},
				"fn-2-support": {"fn-4-deferred"},
			},
			wantPath: "fn-1-example -> fn-2-support -> fn-4-deferred",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, index := validRepositoryFixture(t)
			index.FlowSpecs = append(index.FlowSpecs, flowSpecEntry{
				ID: "fn-4-deferred", Scope: "umpire-roadmap", Disposition: "deferred", Phase: "p3",
				Status: "open", Ready: false, CompletionReview: "unknown", SpecDependencies: []string{},
			})
			writeFlowFixture(t, root, "fn-4-deferred", "open", false, "unknown", nil)
			for id, dependencies := range test.dependencies {
				for position := range index.FlowSpecs {
					if index.FlowSpecs[position].ID != id {
						continue
					}
					index.FlowSpecs[position].SpecDependencies = dependencies
					entry := index.FlowSpecs[position]
					writeFlowFixture(t, root, id, entry.Status, entry.Ready, entry.CompletionReview, dependencies)
				}
			}

			require.Equal(t, []string{
				fmt.Sprintf("flow spec fn-1-example: retained dependency path reaches deferred-only scope: %s", test.wantPath),
			}, checkRepository(root, index))
		})
	}
}

func TestCheckRepositoryReportsInvalidFlowJSONAndCoverage(t *testing.T) {
	root, index := validRepositoryFixture(t)
	writeFixtureFile(t, root, ".flow/specs/fn-1-example.json", `{"id":`)
	writeFixtureFile(t, root, ".flow/specs/fn-4-unregistered.json", `{}`)
	index.FlowSpecs = append(index.FlowSpecs, flowSpecEntry{
		ID: "fn-5-missing", Scope: "other", Disposition: "out-of-scope", Phase: "none",
		Status: "done", CompletionReview: "unknown", SpecDependencies: []string{},
	})

	want := []string{
		`flow spec fn-1-example: decode .flow/specs/fn-1-example.json: unexpected end of JSON input`,
		`flow spec fn-4-unregistered: not registered`,
		`flow spec fn-5-missing: registered file does not exist`,
	}
	require.Equal(t, want, checkRepository(root, index))
}

func TestRunUsesStableStreamsAndDoesNotAcceptArguments(t *testing.T) {
	root, index := validRepositoryFixture(t)
	encoded, err := json.Marshal(indexRegistryJSON(index))
	require.NoError(t, err)
	writeFixtureFile(t, root, ".plans/index.json", string(encoded))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	require.Equal(t, 0, run([]string{"-repository-root", root}, &stdout, &stderr))
	require.Equal(t, "Umpire plan index is valid.\n", stdout.String())
	require.Empty(t, stderr.String())

	stdout.Reset()
	stderr.Reset()
	require.Equal(t, 2, run([]string{"unexpected"}, &stdout, &stderr))
	require.Empty(t, stdout.String())
	require.Equal(t, "planindex accepts no positional arguments\n", stderr.String())
}

func TestRepositoryPlanIndexCoversProductionFiles(t *testing.T) {
	root := filepath.Clean("../..")
	encoded, err := os.ReadFile(filepath.Join(root, ".plans", "index.json"))
	require.NoError(t, err)
	index, err := parseIndex(encoded)
	require.NoError(t, err)

	documents, err := discoverFiles(root, ".plans", ".md")
	require.NoError(t, err)
	registeredDocuments := make([]string, 0, len(index.Documents))
	for _, document := range index.Documents {
		registeredDocuments = append(registeredDocuments, document.Path)
	}
	require.Equal(t, documents, registeredDocuments)

	flowSpecs, err := discoverFiles(root, ".flow/specs", ".json")
	require.NoError(t, err)
	registeredFlowSpecs := make([]string, 0, len(index.FlowSpecs))
	for _, spec := range index.FlowSpecs {
		registeredFlowSpecs = append(registeredFlowSpecs, filepath.ToSlash(filepath.Join(".flow/specs", spec.ID+".json")))
	}
	require.Equal(t, flowSpecs, registeredFlowSpecs)
}

func validRepositoryFixture(t *testing.T) (string, planIndex) {
	t.Helper()
	root := t.TempDir()
	writeFixtureFile(t, root, ".plans/A.md", "# Authority\n[delivery](B.md#delivery)\n[future](Future.md#future)\n")
	writeFixtureFile(t, root, ".plans/B.md", "# Delivery ##\n")
	writeFixtureFile(t, root, ".plans/C.md", "# Notes\n")
	writeFlowFixture(t, root, "fn-1-example", "open", true, "unknown", nil)
	writeFlowFixture(t, root, "fn-2-support", "open", false, "ship", []string{"fn-1-example"})
	writeFixtureFile(t, root, ".flow/specs/fn-3-other.json", `{"id":"fn-3-other","status":"done","depends_on_epics":[]}`)
	return root, planIndex{
		Format: supportedIndexFormat,
		Documents: []documentEntry{
			{
				Path: ".plans/A.md", Lifecycle: "active", Authority: "normative-rules",
				AuthorityParents: []string{}, AllowedMissingLinks: []allowedMissingLink{{
					Target: ".plans/Future.md", Reason: "future contract", Anchor: stringPointer("future"),
				}},
			},
			{
				Path: ".plans/B.md", Lifecycle: "active", Authority: "delivery-order",
				AuthorityParents: []string{".plans/A.md"}, AllowedMissingLinks: []allowedMissingLink{},
			},
			{
				Path: ".plans/C.md", Lifecycle: "reference", Authority: "descriptive",
				AuthorityParents: []string{".plans/B.md"}, AllowedMissingLinks: []allowedMissingLink{},
			},
		},
		FlowSpecs: []flowSpecEntry{
			{ID: "fn-1-example", Scope: "umpire-roadmap", Disposition: "retained", Phase: "p0", Status: "open", Ready: true, CompletionReview: "unknown", SpecDependencies: []string{}},
			{ID: "fn-2-support", Scope: "umpire-support", Disposition: "completed-prerequisite", Phase: "support", Status: "open", Ready: false, CompletionReview: "ship", SpecDependencies: []string{"fn-1-example"}},
			{ID: "fn-3-other", Scope: "other", Disposition: "out-of-scope", Phase: "none", Status: "done", Ready: false, CompletionReview: "unknown", SpecDependencies: []string{}},
		},
	}
}

func writeFlowFixture(t *testing.T, root, id, status string, ready bool, review string, dependencies []string) {
	t.Helper()
	if dependencies == nil {
		dependencies = []string{}
	}
	encoded, err := json.Marshal(map[string]any{
		"id": id, "status": status, "ready": ready,
		"completion_review_status": review, "depends_on_epics": dependencies,
	})
	require.NoError(t, err)
	writeFixtureFile(t, root, filepath.ToSlash(filepath.Join(".flow/specs", id+".json")), string(encoded))
}

func writeFixtureFile(t *testing.T, root, relativePath, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func repositorySnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := make(map[string]string)
	require.NoError(t, filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		snapshot[filepath.ToSlash(relative)] = string(content)
		return nil
	}))
	return snapshot
}

func stringPointer(value string) *string {
	return &value
}

func indexRegistryJSON(index planIndex) map[string]any {
	documents := make([]map[string]any, 0, len(index.Documents))
	for _, document := range index.Documents {
		missing := make([]map[string]any, 0, len(document.AllowedMissingLinks))
		for _, link := range document.AllowedMissingLinks {
			missing = append(missing, map[string]any{"target": link.Target, "reason": link.Reason, "anchor": link.Anchor})
		}
		documents = append(documents, map[string]any{
			"path": document.Path, "lifecycle": document.Lifecycle, "authority": document.Authority,
			"authorityParents": document.AuthorityParents, "supersededBy": document.SupersededBy,
			"allowedMissingLinks": missing,
		})
	}
	flowSpecs := make([]map[string]any, 0, len(index.FlowSpecs))
	for _, spec := range index.FlowSpecs {
		flowSpecs = append(flowSpecs, map[string]any{
			"id": spec.ID, "scope": spec.Scope, "disposition": spec.Disposition, "phase": spec.Phase,
			"status": spec.Status, "ready": spec.Ready, "completionReview": spec.CompletionReview,
			"specDependencies": spec.SpecDependencies,
		})
	}
	return map[string]any{"format": index.Format, "documents": documents, "flowSpecs": flowSpecs}
}
