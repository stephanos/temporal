package regression

import (
	"encoding/json"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProductionManifestIsClosedAndMechanical(t *testing.T) {
	expected := []manifestEntry{{
		Identity:           callerClosureIdentity,
		FixturePath:        "model/Temporal/Feature/Nexus/testdata/nexus-caller-closure-experiment-spec.json",
		GoOutputPath:       "tools/umpire/regression/catalog_generated_test.go",
		MarkdownOutputPath: "model/Temporal/Tool/Generated/Regressions.md",
	}}
	require.Equal(t, expected, productionManifest())
	require.Equal(t, []string{"Identity", "FixturePath", "GoOutputPath", "MarkdownOutputPath"}, structFieldNames(manifestEntry{}))
	require.NoError(t, validateManifest(productionManifest()))
}

func TestProductionFixtureProjectsCanonicalMetadata(t *testing.T) {
	repositoryRoot := testRepositoryRoot(t)
	entry := productionManifest()[0]
	encoded, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(entry.FixturePath)))
	require.NoError(t, err)

	projection, err := extractProjection(entry, encoded, filepath.Join(repositoryRoot, "model"))
	require.NoError(t, err)
	require.Equal(t, projectionRecord{
		Identity:           callerClosureIdentity,
		Format:             supportedExperimentFormat,
		FixturePath:        entry.FixturePath,
		GoOutputPath:       entry.GoOutputPath,
		MarkdownOutputPath: entry.MarkdownOutputPath,
		TestName:           "TestWorkflowNexusQueryExactActionCallerClosure",
		Sources: []sourceProjection{{
			CanonicalPath:  "Temporal/Feature/Nexus/CallerClosure.lean",
			RepositoryPath: "model/Temporal/Feature/Nexus/CallerClosure.lean",
		}},
		Properties: []string{
			"workflow-nexus.property.caller-closure",
		},
		ObservationRequirements: []string{
			"nexus.observation.cancellation-delivered",
			"nexus.observation.pending-cancellation-count",
			"workflow-nexus.relation.owns-operation",
		},
		SemanticFingerprint: "sha256:315266c53b2c9d94fc1ab3c2772a8424b2aafd9857801c8e24fac111253c53f1",
	}, projection)
}

func TestProjectionRejectsInvalidExperimentMetadata(t *testing.T) {
	modelRoot := t.TempDir()
	writeLeanSource(t, modelRoot, "One.lean")
	valid := syntheticExperiment(t, syntheticOptions{})
	cases := map[string]struct {
		encoded []byte
		want    string
	}{
		"empty JSON":              {encoded: nil, want: "JSON is empty"},
		"malformed JSON":          {encoded: []byte("{"), want: "decode canonical ExperimentSpec JSON"},
		"trailing JSON":           {encoded: append(append([]byte(nil), valid...), []byte("{}")...), want: "trailing JSON value"},
		"unsupported version":     {encoded: syntheticExperiment(t, syntheticOptions{format: "umpire-experiment/v2"}), want: "unsupported format"},
		"identity mismatch":       {encoded: syntheticExperiment(t, syntheticOptions{identity: "another.query"}), want: "query identity mismatch"},
		"empty semantic identity": {encoded: syntheticExperiment(t, syntheticOptions{emptySemanticIdentity: true}), want: "semantic identity is empty"},
		"missing provenance":      {encoded: syntheticExperiment(t, syntheticOptions{sources: []string{}}), want: "at least one provenance source"},
		"empty provenance":        {encoded: syntheticExperiment(t, syntheticOptions{sources: []string{""}}), want: "unsafe"},
		"duplicate provenance":    {encoded: syntheticExperiment(t, syntheticOptions{sources: []string{"One.lean", "One.lean"}}), want: "duplicate provenance source"},
		"absolute provenance":     {encoded: syntheticExperiment(t, syntheticOptions{sources: []string{"/One.lean"}}), want: "unsafe"},
		"traversing provenance":   {encoded: syntheticExperiment(t, syntheticOptions{sources: []string{"../One.lean"}}), want: "unsafe"},
		"noncanonical provenance": {encoded: syntheticExperiment(t, syntheticOptions{sources: []string{"Dir/../One.lean"}}), want: "unsafe"},
		"non-Lean provenance":     {encoded: syntheticExperiment(t, syntheticOptions{sources: []string{"One.txt"}}), want: "not a Lean source"},
		"nonexistent provenance":  {encoded: syntheticExperiment(t, syntheticOptions{sources: []string{"Missing.lean"}}), want: "resolve provenance source"},
		"missing properties":      {encoded: syntheticExperiment(t, syntheticOptions{properties: []string{}}), want: "at least one property identity"},
		"empty property":          {encoded: syntheticExperiment(t, syntheticOptions{properties: []string{""}}), want: "property identity is empty"},
		"duplicate property":      {encoded: syntheticExperiment(t, syntheticOptions{properties: []string{"property.one", "property.one"}}), want: "duplicate property identity"},
		"missing observations":    {encoded: syntheticExperiment(t, syntheticOptions{requirements: []string{}}), want: "at least one observation requirement identity"},
		"empty observation":       {encoded: syntheticExperiment(t, syntheticOptions{requirements: []string{""}}), want: "observation requirement identity is empty"},
		"duplicate observation":   {encoded: syntheticExperiment(t, syntheticOptions{requirements: []string{"observation.one", "observation.one"}}), want: "duplicate observation requirement identity"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := extractProjection(syntheticEntry(callerClosureIdentity), test.encoded, modelRoot)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestProjectionRejectsContradictoryJSONObjectKeys(t *testing.T) {
	modelRoot := t.TempDir()
	writeLeanSource(t, modelRoot, "One.lean")
	valid := string(syntheticExperiment(t, syntheticOptions{}))
	cases := []struct {
		name    string
		encoded string
		want    string
	}{
		{
			name:    "duplicate top-level field",
			encoded: strings.Replace(valid, `"formatVersion":`, `"formatVersion":"ignored","formatVersion":`, 1),
			want:    `duplicate JSON object key "formatVersion"`,
		},
		{
			name:    "case-variant top-level field",
			encoded: strings.Replace(valid, `"formatVersion"`, `"FormatVersion"`, 1),
			want:    `JSON object key "FormatVersion" must be spelled "formatVersion"`,
		},
		{
			name:    "duplicate nested field",
			encoded: strings.Replace(valid, `"queryIdentity":`, `"queryIdentity":"ignored","queryIdentity":`, 1),
			want:    `duplicate JSON object key "queryIdentity"`,
		},
		{
			name:    "case-variant nested field",
			encoded: strings.Replace(valid, `"queryIdentity"`, `"QueryIdentity"`, 1),
			want:    `JSON object key "QueryIdentity" must be spelled "queryIdentity"`,
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := extractProjection(
				syntheticEntry(callerClosureIdentity),
				[]byte(test.encoded),
				modelRoot,
			)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestProjectionRejectsProvenanceSymlinkEscapeAndWrongKind(t *testing.T) {
	modelRoot := t.TempDir()
	outsideRoot := t.TempDir()
	writeLeanSource(t, outsideRoot, "Outside.lean")
	require.NoError(t, os.Symlink(filepath.Join(outsideRoot, "Outside.lean"), filepath.Join(modelRoot, "Escape.lean")))
	require.NoError(t, os.Mkdir(filepath.Join(modelRoot, "Directory.lean"), 0o755))

	for name, test := range map[string]struct {
		source string
		want   string
	}{
		"symlink escape": {source: "Escape.lean", want: "resolves outside model root"},
		"directory":      {source: "Directory.lean", want: "not a regular file"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := extractProjection(
				syntheticEntry(callerClosureIdentity),
				syntheticExperiment(t, syntheticOptions{sources: []string{test.source}}),
				modelRoot,
			)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestManifestRejectsUnsafePathsAndCollisions(t *testing.T) {
	valid := syntheticEntry("query.one")
	cases := map[string]struct {
		entries []manifestEntry
		want    string
	}{
		"missing identity": {
			entries: []manifestEntry{withEntry(valid, func(entry *manifestEntry) { entry.Identity = "" })},
			want:    "identity is required",
		},
		"absolute fixture": {
			entries: []manifestEntry{withEntry(valid, func(entry *manifestEntry) { entry.FixturePath = "/fixture.json" })},
			want:    "unsafe",
		},
		"Windows fixture": {
			entries: []manifestEntry{withEntry(valid, func(entry *manifestEntry) { entry.FixturePath = `C:\fixture.json` })},
			want:    "unsafe",
		},
		"traversing Go output": {
			entries: []manifestEntry{withEntry(valid, func(entry *manifestEntry) { entry.GoOutputPath = "../catalog_test.go" })},
			want:    "unsafe",
		},
		"noncanonical Markdown output": {
			entries: []manifestEntry{withEntry(valid, func(entry *manifestEntry) { entry.MarkdownOutputPath = "docs/../index.md" })},
			want:    "unsafe",
		},
		"wrong fixture kind": {
			entries: []manifestEntry{withEntry(valid, func(entry *manifestEntry) { entry.FixturePath = "fixture.txt" })},
			want:    "must be JSON",
		},
		"duplicate identity": {
			entries: []manifestEntry{valid, withEntry(valid, func(entry *manifestEntry) {
				entry.GoOutputPath = "other_test.go"
				entry.MarkdownOutputPath = "other.md"
			})},
			want: "duplicate identity",
		},
		"colliding output": {
			entries: []manifestEntry{valid, withEntry(syntheticEntry("query.two"), func(entry *manifestEntry) {
				entry.GoOutputPath = valid.GoOutputPath
			})},
			want: "collides",
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, validateManifest(test.entries), test.want)
		})
	}
}

func TestGoTestNamesRejectInvalidNamesAndCollisions(t *testing.T) {
	_, err := deriveTestName("---")
	require.ErrorContains(t, err, "does not produce a valid Go test name")

	records := []projectionRecord{
		{Identity: "query.one-value", TestName: "TestQueryOneValue"},
		{Identity: "query.one.value", TestName: "TestQueryOneValue"},
	}
	require.ErrorContains(t, validateTestNames(records), "collide as Go test name")

	records[1].Identity = "query.two"
	require.ErrorContains(t, validateTestNames(records), "invalid Go test name")
}

func TestGeneratedProjectionCarriesMatchingMetadata(t *testing.T) {
	record := syntheticProjection()
	artifacts, err := renderProjections([]projectionRecord{record})
	require.NoError(t, err)
	goSource := artifacts[record.GoOutputPath]
	markdown := artifacts[record.MarkdownOutputPath]

	require.True(t, strings.HasPrefix(string(goSource), generatedMarker+"\n"))
	formatted, err := format.Source(goSource)
	require.NoError(t, err)
	require.Equal(t, goSource, formatted)
	parsed, err := parser.ParseFile(token.NewFileSet(), record.GoOutputPath, goSource, parser.ParseComments)
	require.NoError(t, err)
	require.Equal(t, "regression", parsed.Name.Name)
	require.Equal(t, 1, countGeneratedTests(parsed))
	require.Equal(t, 1, generatedTestStatementCount(t, parsed, record.TestName))
	require.Contains(t, string(goSource), "RequireProjection(t, Reference{")
	require.Contains(t, string(goSource), record.Sources[0].CanonicalPath)
	require.Contains(t, string(goSource), record.Sources[0].RepositoryPath)
	require.Contains(t, string(goSource), record.Properties[0])
	require.Contains(t, string(goSource), record.ObservationRequirements[0])
	require.Contains(t, string(goSource), record.SemanticFingerprint)
	require.NotContains(t, string(goSource), "full-semantic-identity")

	require.Contains(t, string(markdown), "model projection only")
	require.Contains(t, string(markdown), "does not represent Temporal runtime execution, execution evidence, or conformance")
	for _, metadata := range []string{
		record.Identity,
		record.Format,
		record.FixturePath,
		record.Sources[0].RepositoryPath,
		record.Properties[0],
		record.ObservationRequirements[0],
		record.SemanticFingerprint,
	} {
		require.Contains(t, string(markdown), metadata)
	}
	require.NotContains(t, string(markdown), "full-semantic-identity")
}

func TestRenderingIsIndependentOfInputAndJSONObjectOrder(t *testing.T) {
	modelRoot := t.TempDir()
	writeLeanSource(t, modelRoot, "One.lean")
	writeLeanSource(t, modelRoot, "Two.lean")
	firstJSON := syntheticExperiment(t, syntheticOptions{
		sources:      []string{"Two.lean", "One.lean"},
		properties:   []string{"property.two", "property.one"},
		requirements: []string{"observation.two", "observation.one"},
	})
	secondJSON := []byte(`{
		"semanticIdentity":"synthetic-semantic-identity",
		"observationRequirements":["observation.one","observation.two"],
		"properties":[{"identity":"property.one"},{"identity":"property.two"}],
		"provenance":{"sources":[{"path":"One.lean"},{"path":"Two.lean"}]},
		"plan":{"queryIdentity":"workflow-nexus.query.exact-action-caller-closure"},
		"formatVersion":"umpire-experiment/v1"
	}`)
	entry := syntheticEntry(callerClosureIdentity)
	first, err := extractProjection(entry, firstJSON, modelRoot)
	require.NoError(t, err)
	second, err := extractProjection(entry, secondJSON, modelRoot)
	require.NoError(t, err)
	require.Equal(t, first, second)

	firstArtifacts, err := renderProjections([]projectionRecord{first})
	require.NoError(t, err)
	secondArtifacts, err := renderProjections([]projectionRecord{second})
	require.NoError(t, err)
	require.Equal(t, firstArtifacts, secondArtifacts)
	repeated, err := renderProjections([]projectionRecord{first})
	require.NoError(t, err)
	require.Equal(t, firstArtifacts, repeated)
	for _, encoded := range firstArtifacts {
		require.NotContains(t, string(encoded), modelRoot)
		require.NotContains(t, string(encoded), "full-semantic-identity")
		require.NotContains(t, encoded, []byte{'\r'})
	}
}

func TestRenderedPairRejectsGoOrMarkdownMetadataDivergence(t *testing.T) {
	record := syntheticProjection()
	goSource, err := renderGo(record)
	require.NoError(t, err)
	markdown := renderMarkdown(record)

	badGo := bytesReplaceOnce(t, goSource, record.Properties[0], "property.changed")
	require.ErrorContains(t, validateRenderedPair(record, badGo, markdown), "Go metadata diverges")
	badMarkdown := bytesReplaceOnce(t, markdown, record.ObservationRequirements[0], "observation.changed")
	require.ErrorContains(t, validateRenderedPair(record, goSource, badMarkdown), "Markdown metadata diverges")
}

type syntheticOptions struct {
	format                string
	identity              string
	emptySemanticIdentity bool
	sources               []string
	properties            []string
	requirements          []string
}

func syntheticExperiment(t *testing.T, options syntheticOptions) []byte {
	t.Helper()
	if options.format == "" {
		options.format = supportedExperimentFormat
	}
	if options.identity == "" {
		options.identity = callerClosureIdentity
	}
	if options.sources == nil {
		options.sources = []string{"One.lean"}
	}
	if options.properties == nil {
		options.properties = []string{"property.one"}
	}
	if options.requirements == nil {
		options.requirements = []string{"observation.one"}
	}
	semanticIdentity := "synthetic-semantic-identity"
	if options.emptySemanticIdentity {
		semanticIdentity = ""
	}
	properties := make([]map[string]string, 0, len(options.properties))
	for _, identity := range options.properties {
		properties = append(properties, map[string]string{"identity": identity})
	}
	sources := make([]map[string]string, 0, len(options.sources))
	for _, source := range options.sources {
		sources = append(sources, map[string]string{"path": source})
	}
	encoded, err := json.Marshal(map[string]any{
		"formatVersion":           options.format,
		"plan":                    map[string]string{"queryIdentity": options.identity},
		"properties":              properties,
		"observationRequirements": options.requirements,
		"semanticIdentity":        semanticIdentity,
		"provenance":              map[string]any{"sources": sources},
	})
	require.NoError(t, err)
	return encoded
}

func syntheticEntry(identity string) manifestEntry {
	return manifestEntry{
		Identity:           identity,
		FixturePath:        "model/fixture.json",
		GoOutputPath:       "tools/umpire/regression/catalog_generated_test.go",
		MarkdownOutputPath: "model/Generated.md",
	}
}

func syntheticProjection() projectionRecord {
	return projectionRecord{
		Identity:           callerClosureIdentity,
		Format:             supportedExperimentFormat,
		FixturePath:        "model/fixture.json",
		GoOutputPath:       "tools/umpire/regression/catalog_generated_test.go",
		MarkdownOutputPath: "model/Generated.md",
		TestName:           "TestWorkflowNexusQueryExactActionCallerClosure",
		Sources: []sourceProjection{{
			CanonicalPath:  "Temporal/Feature/Nexus/CallerClosure.lean",
			RepositoryPath: "model/Temporal/Feature/Nexus/CallerClosure.lean",
		}},
		Properties:              []string{"property.one"},
		ObservationRequirements: []string{"observation.one"},
		SemanticFingerprint:     "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
}

func testRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", ".."))
}

func writeLeanSource(t *testing.T, root, relative string) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(relative))
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
	require.NoError(t, os.WriteFile(target, []byte("-- test source\n"), 0o644))
}

func structFieldNames(value any) []string {
	typeOf := reflect.TypeOf(value)
	result := make([]string, 0, typeOf.NumField())
	for index := range typeOf.NumField() {
		result = append(result, typeOf.Field(index).Name)
	}
	return result
}

func withEntry(entry manifestEntry, mutate func(*manifestEntry)) manifestEntry {
	mutate(&entry)
	return entry
}

func countGeneratedTests(file *ast.File) int {
	count := 0
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && strings.HasPrefix(function.Name.Name, "Test") {
			count++
		}
	}
	return count
}

func generatedTestStatementCount(t *testing.T, file *ast.File, name string) int {
	t.Helper()
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == name {
			return len(function.Body.List)
		}
	}
	require.FailNow(t, "generated test function not found", name)
	return 0
}

func bytesReplaceOnce(t *testing.T, source []byte, old, replacement string) []byte {
	t.Helper()
	require.Equal(t, 1, strings.Count(string(source), old))
	return []byte(strings.Replace(string(source), old, replacement, 1))
}
