package main

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

func TestProductionManifestIsClosedAndMechanical(t *testing.T) {
	expected := []manifestEntry{
		{
			Identity:           switchIdentity,
			FixturePath:        "model/Umpire/Examples/testdata/switch-experiment-spec.json",
			GoOutputPath:       "tools/umpire/regression/switch_generated_view_test.go",
			MarkdownOutputPath: "model/Umpire/Examples/Generated/Switch.md",
		},
		{
			Identity:           callerClosureIdentity,
			FixturePath:        "model/Temporal/Feature/Nexus/Experimental/testdata/nexus-caller-closure-experiment-spec.json",
			GoOutputPath:       "tools/umpire/regression/catalog_generated_test.go",
			MarkdownOutputPath: "model/Temporal/Tool/Generated/Regressions.md",
		},
	}
	require.Equal(t, expected, productionManifest())
	require.Equal(t, []string{"Identity", "FixturePath", "GoOutputPath", "MarkdownOutputPath"}, structFieldNames(manifestEntry{}))
	require.NoError(t, validateManifest(productionManifest()))
}

func TestProductionFixtureCarriesCanonicalMetadata(t *testing.T) {
	repositoryRoot := testRepositoryRoot(t)
	entry := productionManifest()[1]
	encoded, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(entry.FixturePath)))
	require.NoError(t, err)

	view, err := extractGeneratedView(entry, encoded, filepath.Join(repositoryRoot, "model"))
	require.NoError(t, err)
	require.Equal(t, generatedViewRecord{
		Identity:           callerClosureIdentity,
		Format:             supportedExperimentFormat,
		FixturePath:        entry.FixturePath,
		GoOutputPath:       entry.GoOutputPath,
		MarkdownOutputPath: entry.MarkdownOutputPath,
		TestName:           "TestWorkflowNexusQueryExactActionCallerClosure",
		Sources: []sourceView{{
			CanonicalPath:  "Temporal/Feature/Nexus/Experimental/CallerClosure.lean",
			RepositoryPath: "model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean",
		}},
		Properties: []string{
			"workflow-nexus.property.caller-closure",
		},
		ObservationRequirements: []string{
			"nexus.observation.cancellation-delivered",
			"nexus.observation.pending-cancellation-count",
			"workflow-nexus.relation.owns-operation",
		},
		ArtifactChecksum: "sha256:dde2fb35891dcc0020dbedf301805feda1b5136ec8622dd67fdc47a3d00fb1a8",
	}, view)
}

func TestProductionGeneratedViewSetOwnsExactlyFourCompleteOutputs(t *testing.T) {
	repositoryRoot := testRepositoryRoot(t)
	records := make([]generatedViewRecord, 0, len(productionManifest()))
	for _, entry := range productionManifest() {
		encoded, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(entry.FixturePath)))
		require.NoError(t, err)
		record, err := extractGeneratedView(entry, encoded, filepath.Join(repositoryRoot, "model"))
		require.NoError(t, err)
		records = append(records, record)
	}

	artifacts, err := renderGeneratedViews(records)
	require.NoError(t, err)
	require.Equal(t, []string{
		"model/Temporal/Tool/Generated/Regressions.md",
		"model/Umpire/Examples/Generated/Switch.md",
		"tools/umpire/regression/catalog_generated_test.go",
		"tools/umpire/regression/switch_generated_view_test.go",
	}, managedArtifactPaths(productionManifest()))
	require.Len(t, artifacts, 4)
	require.NoError(t, validateGeneratedArtifacts(productionManifest(), records, artifacts))

	missing := cloneArtifacts(artifacts)
	delete(missing, records[0].MarkdownOutputPath)
	require.Error(t, validateGeneratedArtifacts(productionManifest(), records, missing))
	extra := cloneArtifacts(artifacts)
	extra["model/Unexpected.md"] = []byte("unexpected\n")
	require.Error(t, validateGeneratedArtifacts(productionManifest(), records, extra))
	stale := cloneArtifacts(artifacts)
	stale[records[0].MarkdownOutputPath] = []byte("stale\n")
	require.Error(t, validateGeneratedArtifacts(productionManifest(), records, stale))
	partial := map[string][]byte{
		records[0].GoOutputPath:       artifacts[records[0].GoOutputPath],
		records[0].MarkdownOutputPath: artifacts[records[0].MarkdownOutputPath],
	}
	require.Error(t, validateGeneratedArtifacts(productionManifest(), records, partial))
}

func TestGeneratedViewRejectsInvalidExperimentMetadata(t *testing.T) {
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
		"unsupported version":     {encoded: syntheticExperiment(t, syntheticOptions{format: "umpire-experiment/v1"}), want: "unsupported format"},
		"identity mismatch":       {encoded: syntheticExperiment(t, syntheticOptions{identity: "another.query"}), want: "query definition ID mismatch"},
		"empty artifact checksum": {encoded: syntheticExperiment(t, syntheticOptions{emptyArtifactChecksum: true}), want: "artifact checksum"},
		"missing provenance":      {encoded: syntheticExperiment(t, syntheticOptions{sources: []string{}}), want: "at least one source location"},
		"empty provenance":        {encoded: syntheticExperiment(t, syntheticOptions{sources: []string{""}}), want: "source location is malformed"},
		"duplicate provenance":    {encoded: syntheticExperiment(t, syntheticOptions{sources: []string{"One.lean", "One.lean"}}), want: "duplicate source location"},
		"absolute provenance":     {encoded: syntheticExperiment(t, syntheticOptions{sources: []string{"/One.lean"}}), want: "unsafe"},
		"traversing provenance":   {encoded: syntheticExperiment(t, syntheticOptions{sources: []string{"../One.lean"}}), want: "unsafe"},
		"noncanonical provenance": {encoded: syntheticExperiment(t, syntheticOptions{sources: []string{"Dir/../One.lean"}}), want: "unsafe"},
		"non-Lean provenance":     {encoded: syntheticExperiment(t, syntheticOptions{sources: []string{"One.txt"}}), want: "not a Lean source"},
		"nonexistent provenance":  {encoded: syntheticExperiment(t, syntheticOptions{sources: []string{"Missing.lean"}}), want: "resolve provenance source"},
		"missing properties":      {encoded: syntheticExperiment(t, syntheticOptions{properties: []string{}}), want: "at least one property identity"},
		"empty property":          {encoded: syntheticExperiment(t, syntheticOptions{properties: []string{""}}), want: "malformed definition ID"},
		"duplicate property":      {encoded: syntheticExperiment(t, syntheticOptions{properties: []string{"property.one", "property.one"}}), want: "duplicate property identity"},
		"missing observations":    {encoded: syntheticExperiment(t, syntheticOptions{requirements: []string{}}), want: "at least one observation requirement identity"},
		"empty observation":       {encoded: syntheticExperiment(t, syntheticOptions{requirements: []string{""}}), want: "observation requirement definition ID is empty"},
		"duplicate observation":   {encoded: syntheticExperiment(t, syntheticOptions{requirements: []string{"observation.one", "observation.one"}}), want: "duplicate observation requirement definition ID"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := extractGeneratedView(syntheticEntry(callerClosureIdentity), test.encoded, modelRoot)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestGeneratedViewRejectsContradictoryJSONObjectKeys(t *testing.T) {
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
			want:    `duplicate or case-colliding top-level key`,
		},
		{
			name:    "case-variant top-level field",
			encoded: strings.Replace(valid, `"formatVersion"`, `"FormatVersion"`, 1),
			want:    `JSON object key "FormatVersion" must be spelled "formatVersion"`,
		},
		{
			name:    "duplicate nested field",
			encoded: strings.Replace(valid, `"queryDefinitionId":`, `"queryDefinitionId":"ignored","queryDefinitionId":`, 1),
			want:    `duplicate or case-colliding JSON object key`,
		},
		{
			name:    "case-variant nested field",
			encoded: strings.Replace(valid, `"queryDefinitionId"`, `"QueryDefinitionId"`, 1),
			want:    `JSON object key "QueryDefinitionId" must be spelled "queryDefinitionId"`,
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := extractGeneratedView(
				syntheticEntry(callerClosureIdentity),
				[]byte(test.encoded),
				modelRoot,
			)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestGeneratedViewRejectsProvenanceSymlinkEscapeAndWrongKind(t *testing.T) {
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
			_, err := extractGeneratedView(
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

	records := []generatedViewRecord{
		{Identity: "query.one-value", TestName: "TestQueryOneValue"},
		{Identity: "query.one.value", TestName: "TestQueryOneValue"},
	}
	require.ErrorContains(t, validateGeneratedViewTestNames(records), "collide as Go test name")

	records[1].Identity = "query.two"
	require.ErrorContains(t, validateGeneratedViewTestNames(records), "invalid Go test name")
}

func TestGeneratedViewCarriesMatchingMetadata(t *testing.T) {
	record := syntheticGeneratedView()
	artifacts, err := renderGeneratedViews([]generatedViewRecord{record})
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
	require.Contains(t, string(goSource), "RequireGeneratedView(t, Reference{")
	require.Contains(t, string(goSource), record.Sources[0].CanonicalPath)
	require.Contains(t, string(goSource), record.Sources[0].RepositoryPath)
	require.Contains(t, string(goSource), record.Properties[0])
	require.Contains(t, string(goSource), record.ObservationRequirements[0])
	require.Contains(t, string(goSource), record.ArtifactChecksum)
	require.NotContains(t, string(goSource), "full-semantic-identity")

	require.Contains(t, string(markdown), "model generated view only")
	require.Contains(t, string(markdown), "does not represent Temporal runtime execution, execution evidence, or conformance")
	for _, metadata := range []string{
		record.Identity,
		record.Format,
		record.FixturePath,
		record.Sources[0].RepositoryPath,
		record.Properties[0],
		record.ObservationRequirements[0],
		record.ArtifactChecksum,
	} {
		require.Contains(t, string(markdown), metadata)
	}
	require.NotContains(t, string(markdown), "full-semantic-identity")
}

func TestRenderingIsDeterministic(t *testing.T) {
	modelRoot := t.TempDir()
	writeLeanSource(t, modelRoot, "One.lean")
	writeLeanSource(t, modelRoot, "Two.lean")
	firstJSON := syntheticExperiment(t, syntheticOptions{
		sources:      []string{"Two.lean", "One.lean"},
		properties:   []string{"property.two", "property.one"},
		requirements: []string{"observation.two", "observation.one"},
	})
	entry := syntheticEntry(callerClosureIdentity)
	first, err := extractGeneratedView(entry, firstJSON, modelRoot)
	require.NoError(t, err)

	firstArtifacts, err := renderGeneratedViews([]generatedViewRecord{first})
	require.NoError(t, err)
	repeated, err := renderGeneratedViews([]generatedViewRecord{first})
	require.NoError(t, err)
	require.Equal(t, firstArtifacts, repeated)
	for _, encoded := range firstArtifacts {
		require.NotContains(t, string(encoded), modelRoot)
		require.NotContains(t, string(encoded), "full-semantic-identity")
		require.NotContains(t, encoded, []byte{'\r'})
	}
}

func TestRenderedPairRejectsGoOrMarkdownMetadataDivergence(t *testing.T) {
	record := syntheticGeneratedView()
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
	emptyArtifactChecksum bool
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
	slices.Sort(options.sources)
	slices.Sort(options.properties)
	slices.Sort(options.requirements)
	const fingerprint = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	properties := make([]artifactv2.Property, 0, len(options.properties))
	for _, identity := range options.properties {
		properties = append(properties, artifactv2.Property{
			DefinitionID: identity, BehaviorFingerprint: fingerprint, RequirementDefinitionIDs: []string{},
		})
	}
	sources := make([]artifactv2.SourceLocation, 0, len(options.sources))
	for _, source := range options.sources {
		sources = append(sources, artifactv2.SourceLocation{
			Path: source, Line: artifactv2.Natural("1"), Column: artifactv2.Natural("1"), Provenance: "lean-model",
		})
	}
	provenance := artifactv2.Provenance{SourceDefinitionIDs: []string{"synthetic.source"}, SourceLocations: sources}
	document, err := artifactv2.SealExperiment(artifactv2.Experiment{
		FormatVersion:            options.format,
		QueryBehaviorFingerprint: fingerprint,
		Plan: artifactv2.DrivePlan{
			FormatVersion: artifactv2.DrivePlanFormat, QueryDefinitionID: options.identity,
			QueryBehaviorFingerprint: fingerprint, BehaviorDefinitionID: "synthetic.behavior",
			BehaviorFingerprint: fingerprint, TargetDefinitionID: "synthetic.target",
			TargetBehaviorFingerprint: fingerprint, KernelDefinitionID: "synthetic.kernel",
			KernelBehaviorFingerprint: fingerprint, Bindings: []artifactv2.Binding{},
			SymbolicRoles: []artifactv2.Role{}, ModelPreconditions: []artifactv2.Precondition{},
			InitialState:     artifactv2.ModelValue{DefinitionID: "synthetic.state", Value: "initial"},
			RequestedActions: []artifactv2.ModelValue{}, ModelOutcomes: []artifactv2.ModelValue{},
			ResultingStates: []artifactv2.ModelValue{}, LinearExtension: []artifactv2.Occurrence{},
			SelectedChoices: []artifactv2.ModelValue{}, SelectedVariants: []artifactv2.ModelValue{},
			RequestedFaults: []artifactv2.ModelValue{}, CapabilityRequirementDefinitionIDs: []string{},
			ExpandedLimits: artifactv2.Limits{
				Behavior: artifactv2.BehaviorLimits{
					Transitions:     artifactv2.Limit{Value: artifactv2.Natural("1"), Unit: "semantic-transitions"},
					SelectedActions: artifactv2.Limit{Value: artifactv2.Natural("1"), Unit: "selected-actions"},
				},
				Search: artifactv2.Limit{Value: artifactv2.Natural("1"), Unit: "candidate-evaluations"},
			},
			Checkpoints: []artifactv2.Checkpoint{}, SelectionReason: "satisfying-witness",
			Explored: artifactv2.ExploredCounts{
				Setups: artifactv2.Natural("0"), Traces: artifactv2.Natural("0"),
				Transitions: artifactv2.Natural("0"), PropertyEvaluations: artifactv2.Natural("0"),
			},
			KnownGaps: []artifactv2.KnownGap{}, Provenance: provenance,
		},
		Properties: properties, ObservationRequirementDefinitionIDs: options.requirements,
		Provenance: provenance,
	})
	require.NoError(t, err)
	if options.emptyArtifactChecksum {
		document.ArtifactChecksum = ""
	}
	encoded, err := artifactv2.CanonicalExperimentBytes(document)
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

func syntheticGeneratedView() generatedViewRecord {
	return generatedViewRecord{
		Identity:           callerClosureIdentity,
		Format:             supportedExperimentFormat,
		FixturePath:        "model/fixture.json",
		GoOutputPath:       "tools/umpire/regression/catalog_generated_test.go",
		MarkdownOutputPath: "model/Generated.md",
		TestName:           "TestWorkflowNexusQueryExactActionCallerClosure",
		Sources: []sourceView{{
			CanonicalPath:  "Temporal/Feature/Nexus/Experimental/CallerClosure.lean",
			RepositoryPath: "model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean",
		}},
		Properties:              []string{"property.one"},
		ObservationRequirements: []string{"observation.one"},
		ArtifactChecksum:        "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
}

func testRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", ".."))
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

func cloneArtifacts(source map[string][]byte) map[string][]byte {
	result := make(map[string][]byte, len(source))
	for path, encoded := range source {
		result[path] = append([]byte(nil), encoded...)
	}
	return result
}
