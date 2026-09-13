package protocolmigration

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	functionalFixture = "tests/testcore/testpilot/testdata/get-system-info-case.json"
	expectedFixture   = "common/testing/testpilot/testdata/case-runtime-conformance/satisfied/expected.json"
	correlatedFixture = "common/testing/testpilot/testdata/case-runtime-conformance/correlated.json"
)

// setJSON returns encoded with the value at path replaced; a string step is an object key and an
// int step a list index.
func setJSON(t *testing.T, encoded []byte, value any, path ...any) []byte {
	t.Helper()
	root, err := decodePlainJSON(encoded)
	require.NoError(t, err)
	parent := root
	for index, step := range path {
		last := index == len(path)-1
		switch key := step.(type) {
		case string:
			object, ok := parent.(map[string]any)
			require.True(t, ok, "step %v is not an object", step)
			if last {
				object[key] = value
			} else {
				parent = object[key]
			}
		case int:
			list, ok := parent.([]any)
			require.True(t, ok, "step %v is not a list", step)
			if last {
				list[key] = value
			} else {
				parent = list[key]
			}
		default:
			require.Failf(t, "unsupported path step", "%v", step)
		}
	}
	mutated, err := json.MarshalIndent(root, "", "  ")
	require.NoError(t, err)
	return mutated
}

func TestCheckNamesFixtureAndFirstDifferingField(t *testing.T) {
	t.Parallel()

	baseline := loadBaseline(t)
	for _, tc := range []struct {
		name            string
		fixture         string
		mutateBaseline  bool
		path            []any
		value           any
		wantErrorSubstr string
	}{
		{
			name: "Case field", fixture: functionalFixture,
			path: []any{"program", "programId"}, value: "mutated",
			wantErrorSubstr: "at program.program_id",
		},
		{
			name: "expected Verdict", fixture: expectedFixture,
			path: []any{"projection", "disposition"}, value: "STOPPED_BY_MONITOR",
			wantErrorSubstr: "at projection.disposition",
		},
		{
			name: "correlated expected", fixture: correlatedFixture,
			path: []any{0, "expected"}, value: 99,
			wantErrorSubstr: "at [0].expected",
		},
		{
			name: "correlated runnable Case", fixture: correlatedFixture,
			path: []any{1, "runnableCase", "caseId"}, value: "mutated",
			wantErrorSubstr: "at [1].runnableCase.case_id",
		},
		{
			name: "baseline outside the snapshot", fixture: functionalFixture, mutateBaseline: true,
			path: []any{"program", "undeclaredField"}, value: true,
			wantErrorSubstr: "does not decode through the snapshot",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			old, regenerated := readFixturePair(t, tc.fixture)
			if tc.mutateBaseline {
				old = setJSON(t, old, tc.value, tc.path...)
			} else {
				regenerated = setJSON(t, regenerated, tc.value, tc.path...)
			}
			err := baseline.Check(tc.fixture, old, regenerated, Declared)
			require.ErrorContains(t, err, tc.fixture)
			require.ErrorContains(t, err, tc.wantErrorSubstr)
		})
	}
}

func TestCheckNamesFailingStep(t *testing.T) {
	t.Parallel()

	baseline := loadBaseline(t)
	old, regenerated := readFixturePair(t, functionalFixture)
	failing := Step{
		Name: "deliberately failing step", Requirement: "R8",
		Apply: func(string, any) (any, error) { return nil, errors.New("assumption does not hold") },
	}

	err := baseline.Check(functionalFixture, old, regenerated, append(Mapping{failing}, Declared...))
	require.ErrorContains(t, err, functionalFixture)
	require.ErrorContains(t, err, `step "deliberately failing step" (R8): assumption does not hold`)
}

func TestPairedFixturesRejectUnpairedFixtures(t *testing.T) {
	t.Parallel()

	const conformanceCase = conformanceFixtureRoot + "/satisfied/case.json"
	for _, tc := range []struct {
		name            string
		baseline        []string
		regenerated     []string
		wantErrorSubstr string
	}{
		{
			name:            "added regenerated fixture",
			baseline:        []string{conformanceCase},
			regenerated:     []string{conformanceCase, conformanceFixtureRoot + "/added/case.json"},
			wantErrorSubstr: "regenerated fixture " + conformanceFixtureRoot + "/added/case.json has no baseline",
		},
		{
			name:            "deleted regenerated fixture",
			baseline:        []string{conformanceCase, functionalFixtureRoot + "/deleted-case.json"},
			regenerated:     []string{conformanceCase},
			wantErrorSubstr: "baseline fixture " + functionalFixtureRoot + "/deleted-case.json has no regenerated counterpart",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			baselineFixtures, repository := t.TempDir(), t.TempDir()
			for _, fixture := range tc.baseline {
				writeFile(t, filepath.Join(baselineFixtures, filepath.FromSlash(fixture)))
			}
			for _, fixture := range tc.regenerated {
				writeFile(t, filepath.Join(repository, filepath.FromSlash(fixture)))
			}
			_, err := PairedFixtures(baselineFixtures, repository)
			require.ErrorContains(t, err, tc.wantErrorSubstr)
		})
	}
}

func writeFile(t *testing.T, file string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(file), 0o755))
	require.NoError(t, os.WriteFile(file, []byte("{}"), 0o644))
}

func TestMappingHelpers(t *testing.T) {
	t.Parallel()

	const message, other = "test.v1.Message", "test.v1.Other"
	tree := func() any {
		return &Object{Message: message, Fields: map[string]any{
			"old":    json.Number("1"),
			"kind":   json.Number("2"),
			"kinds":  []any{json.Number("2"), "KIND_THREE"},
			"nested": &Object{Message: other, Fields: map[string]any{"old": "kept"}},
		}}
	}
	for _, tc := range []struct {
		name            string
		apply           ApplyFunc
		want            string
		wantErrorSubstr string
	}{
		{
			name:  "rename field in the named message only",
			apply: RenameField(message, "old", "new"),
			want:  `{"kind":2,"kinds":[2,"KIND_THREE"],"nested":{"old":"kept"},"new":1}`,
		},
		{
			name:            "rename onto an existing field",
			apply:           RenameField(message, "old", "kind"),
			wantErrorSubstr: `test.v1.Message carries both "old" and "kind"`,
		},
		{
			name:  "rename enum literal in a list",
			apply: RenameEnumLiteral(message, "kinds", "KIND_THREE", "3"),
			want:  `{"kind":2,"kinds":[2,3],"nested":{"old":"kept"},"old":1}`,
		},
		{
			name:  "rename scalar enum literal to a name",
			apply: RenameEnumLiteral(message, "kind", "2", "KIND_TWO"),
			want:  `{"kind":"KIND_TWO","kinds":[2,"KIND_THREE"],"nested":{"old":"kept"},"old":1}`,
		},
		{
			name:  "drop a checked field",
			apply: DropField(other, "old", func(string, *Object) error { return nil }),
			want:  `{"kind":2,"kinds":[2,"KIND_THREE"],"nested":{},"old":1}`,
		},
		{
			name:            "drop whose check fails",
			apply:           DropField(other, "old", func(string, *Object) error { return errors.New("not derived") }),
			wantErrorSubstr: "test.v1.Other.old: not derived",
		},
		{
			name:            "drop without a check",
			apply:           DropField(other, "old", nil),
			wantErrorSubstr: "dropping test.v1.Other.old declares no check",
		},
		{
			name: "rewrite a subtree",
			apply: RewriteMessages(other, func(string, *Object) (any, error) {
				return "rewritten", nil
			}),
			want: `{"kind":2,"kinds":[2,"KIND_THREE"],"nested":"rewritten","old":1}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mapped, err := tc.apply("fixture.json", tree())
			if tc.wantErrorSubstr != "" {
				require.ErrorContains(t, err, tc.wantErrorSubstr)
				return
			}
			require.NoError(t, err)
			encoded, err := json.Marshal(mapped)
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(encoded))
		})
	}
}
