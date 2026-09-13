package protocolmigration

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
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
	added := []Addition{{Fixture: conformanceFixtureRoot + "/added/case.json", Requirement: "R3"}}
	for _, tc := range []struct {
		name            string
		baseline        []string
		regenerated     []string
		added           []Addition
		wantErrorSubstr string
	}{
		{
			name:        "declared added fixture",
			baseline:    []string{conformanceCase},
			regenerated: []string{conformanceCase, conformanceFixtureRoot + "/added/case.json"},
			added:       added,
		},
		{
			name:            "declared added fixture not regenerated",
			baseline:        []string{conformanceCase},
			regenerated:     []string{conformanceCase},
			added:           added,
			wantErrorSubstr: "added fixture " + conformanceFixtureRoot + "/added/case.json (R3) is not regenerated",
		},
		{
			name:            "declared added fixture with a baseline",
			baseline:        []string{conformanceCase, conformanceFixtureRoot + "/added/case.json"},
			regenerated:     []string{conformanceCase, conformanceFixtureRoot + "/added/case.json"},
			added:           added,
			wantErrorSubstr: "added fixture " + conformanceFixtureRoot + "/added/case.json (R3) has a baseline",
		},
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
			paired, err := PairedFixtures(baselineFixtures, repository, tc.added)
			if tc.wantErrorSubstr == "" {
				require.NoError(t, err)
				require.Equal(t, []string{conformanceCase}, paired)
				return
			}
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

func TestRenameMessageRewritesAnyTypeURL(t *testing.T) {
	t.Parallel()

	tree := []any{
		&Object{Message: anyMessage, Fields: map[string]any{"@type": "type.googleapis.com/test.v1.Old", "field": "kept"}},
		&Object{Message: anyMessage, Fields: map[string]any{"@type": "type.googleapis.com/test.v1.OldSibling"}},
		&Object{Message: "test.v1.Old", Fields: map[string]any{"@type": "type.googleapis.com/test.v1.Old"}},
	}
	mapped, err := RenameMessage("test.v1.Old", "test.v1.New")("fixture.json", tree)
	require.NoError(t, err)
	encoded, err := json.Marshal(mapped)
	require.NoError(t, err)
	require.JSONEq(t, `[
		{"@type": "type.googleapis.com/test.v1.New", "field": "kept"},
		{"@type": "type.googleapis.com/test.v1.OldSibling"},
		{"@type": "type.googleapis.com/test.v1.Old"}
	]`, string(encoded))
}

func TestRenameCorrelatedRuleKey(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name            string
		payload         string
		want            string
		wantErrorSubstr string
	}{
		{
			name:    "renames the key of every rule in place",
			payload: "{\n  \"correlatedRules\": [\n    {\n      \"old\": \"a\",\n      \"other\": \"b\"\n    }\n  ]\n}",
			want:    "{\n  \"correlatedRules\": [\n    {\n      \"new\": \"a\",\n      \"other\": \"b\"\n    }\n  ]\n}",
		},
		{
			name:    "leaves a payload without the key untouched",
			payload: "\x00\xff\x80",
			want:    "\x00\xff\x80",
		},
		{
			name:            "rejects the key outside the rules",
			payload:         `{"old": "a", "correlatedRules": [{"old": "b"}]}`,
			wantErrorSubstr: `producerData spells "old" outside its correlatedRules entries`,
		},
		{
			name:            "rejects a rule without the key",
			payload:         `{"correlatedRules": [{"old": "a"}, {"other": "b"}]}`,
			wantErrorSubstr: `producerData correlatedRules[1] does not carry "old" alone`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			object := &Object{Fields: map[string]any{"producerData": base64.StdEncoding.EncodeToString([]byte(tc.payload))}}
			mapped, err := renameCorrelatedRuleKey("old", "new")("fixture.json", object)
			if tc.wantErrorSubstr != "" {
				require.ErrorContains(t, err, tc.wantErrorSubstr)
				return
			}
			require.NoError(t, err)
			renamed, ok := mapped.(*Object)
			require.True(t, ok)
			require.Equal(t, base64.StdEncoding.EncodeToString([]byte(tc.want)), renamed.Fields["producerData"])
		})
	}
}

func TestDeclaredValueArmRenamesOnlyValueObjects(t *testing.T) {
	t.Parallel()

	tree := &Object{Message: protocol + "CorrelatedEvidenceBinding", Fields: map[string]any{
		"text":          "kept",
		"signedInteger": "kept",
		"literal":       &Object{Message: protocol + "Value", Fields: map[string]any{"text": "renamed"}},
	}}
	mapped, err := Declared.apply("fixture.json", tree)
	require.NoError(t, err)
	encoded, err := json.Marshal(mapped)
	require.NoError(t, err)
	require.JSONEq(t, `{"text": "kept", "signedInteger": "kept", "literal": {"textValue": "renamed"}}`, string(encoded))
}

// The expression step moves every baseline reference under one Reference and folds equals into an
// EQUAL comparison, and it refuses a shape it would otherwise have to drop.
func TestDeclaredExpressionStepRewritesReferencesAndEquality(t *testing.T) {
	t.Parallel()

	expression := func(message string, fields map[string]any) *Object {
		return &Object{Message: protocol + protoreflect.FullName(message), Fields: fields}
	}
	identifier := func(key, value string) *Object { return &Object{Fields: map[string]any{key: value}} }
	for _, tc := range []struct {
		name, message   string
		fields          map[string]any
		want            string
		wantErrorSubstr string
	}{
		{name: "slot", message: "Program" + "Expression", fields: map[string]any{"slot": identifier("slotId", "s")}, want: `{"reference": {"slotId": "s"}}`},
		{name: "empty slot", message: "Program" + "Expression", fields: map[string]any{"slot": &Object{Fields: map[string]any{}}}, want: `{"reference": {"slotId": ""}}`},
		{name: "environment", message: "Program" + "Expression", fields: map[string]any{"environment": identifier("bindingId", "namespace")}, want: `{"reference": {"environmentBindingId": "namespace"}}`},
		{name: "run", message: "Program" + "Expression", fields: map[string]any{"run": &Object{Fields: map[string]any{}}}, want: `{"reference": {"run": {}}}`},
		{name: "outcome", message: "Program" + "Expression", fields: map[string]any{"outcome": &Object{Fields: map[string]any{"field": "INSTRUCTION_OUTCOME_FIELD_STATUS"}}}, want: `{"reference": {"outcome": {"field": "INSTRUCTION_OUTCOME_FIELD_STATUS"}}}`},
		{name: "observation", message: "Contract" + "Expression", fields: map[string]any{"observation": identifier("observationId", "o")}, want: `{"reference": {"observationId": "o"}}`},
		{name: "capture", message: "Contract" + "Expression", fields: map[string]any{"capture": identifier("captureId", "c")}, want: `{"reference": {"captureId": "c"}}`},
		{name: "run event", message: "Contract" + "Expression", fields: map[string]any{"runEvent": &Object{Fields: map[string]any{"field": "RUN_EVENT_FIELD_KIND"}}}, want: `{"reference": {"runEvent": {"field": "RUN_EVENT_FIELD_KIND"}}}`},
		{name: "equals", message: "Contract" + "Expression", fields: map[string]any{"equals": &Object{Fields: map[string]any{"left": "l", "right": "r"}}}, want: `{"compare": {"operator": "COMPARISON_OPERATOR_EQUAL", "left": "l", "right": "r"}}`},
		{name: "negation", message: "Program" + "Expression", fields: map[string]any{"negation": &Object{Fields: map[string]any{"operand": "o"}}}, want: `{"not": {"operand": "o"}}`},
		{name: "reference with a second key", message: "Program" + "Expression", fields: map[string]any{"slot": &Object{Fields: map[string]any{"slotId": "s", "extra": "x"}}}, wantErrorSubstr: `slot carries "extra" beside "slotId"`},
		{name: "two arms", message: "Contract" + "Expression", fields: map[string]any{"literal": "l", "capture": identifier("captureId", "c")}, wantErrorSubstr: "expression carries 2 arms, want 1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mapped, err := Declared.apply("fixture.json", expression(tc.message, tc.fields))
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

// The correlated steps rewrite predicates, operands, correlations and lift guards into Expression,
// and refuse a shape the baseline never admitted rather than dropping part of it.
func TestDeclaredCorrelatedStepsRewriteConditionsAndGuards(t *testing.T) {
	t.Parallel()

	message := func(name string, fields map[string]any) *Object {
		return &Object{Message: protocol + protoreflect.FullName(name), Fields: fields}
	}
	path := func() *Object { return &Object{Fields: map[string]any{"segments": []any{}}} }
	const step = `{"reference": {"correlatedStep": {"definitionId": "d", "field": "CORRELATED_STEP_FIELD_FACT"}}}`
	const projected = `{"path": {"operand": {"reference": {"projectedValue": {}}}, "path": {"segments": []}}}`
	for _, tc := range []struct {
		name            string
		tree            *Object
		want            string
		wantErrorSubstr string
	}{
		{
			name: "present predicate",
			tree: message("Correlated"+"Predicate", map[string]any{"definitionId": "d", "field": "CORRELATED_" + "PREDICATE_FIELD_FACT", "present": true}),
			want: `{"present": {"operand": ` + step + `}}`,
		},
		{
			name: "text predicate",
			tree: message("Correlated"+"Predicate", map[string]any{"definitionId": "d", "field": json.Number("4"), "equalsText": "t"}),
			want: `{"compare": {"operator": "COMPARISON_OPERATOR_EQUAL", "left": ` + step + `, "right": {"literal": {"textValue": "t"}}}}`,
		},
		{
			name:            "false presence",
			tree:            message("Correlated"+"Predicate", map[string]any{"definitionId": "d", "present": false}),
			wantErrorSubstr: "predicate presence is false, want true",
		},
		{
			name:            "two constraints",
			tree:            message("Correlated"+"Predicate", map[string]any{"present": true, "equalsText": "t"}),
			wantErrorSubstr: "predicate carries no single constraint",
		},
		{
			name: "correlation comparison of a field and a capture",
			tree: message("Correlated"+"Correlation", map[string]any{"comparison": message("Correlated"+"Comparison", map[string]any{
				"operator": "CORRELATED_" + "COMPARISON_OPERATOR_NOT_EQUAL",
				"left":     message("Correlated"+"Operand", map[string]any{"fieldId": "f"}),
				"right":    message("Correlated"+"Operand", map[string]any{"capture": &Object{Fields: map[string]any{"captureId": "c"}}}),
			})}),
			want: `{"compare": {"operator": "COMPARISON_OPERATOR_NOT_EQUAL", "left": {"reference": {"evidenceFieldId": "f"}}, "right": {"reference": {"correlatedCapture": {"captureId": "c"}}}}}`,
		},
		{
			name:            "operand with two arms",
			tree:            message("Correlated"+"Operand", map[string]any{"fieldId": "f", "literal": "l"}),
			wantErrorSubstr: "operand carries 2 arms, want 1",
		},
		{
			name: "guard with text",
			tree: message("CorrelatedEvidenceRule", map[string]any{"guard": path(), "guard" + "EqualsText": "t", "kind": "k"}),
			want: `{"kind": "k", "guard": {"all": {"operands": [{"present": {"operand": ` + projected + `}}, {"compare": {"operator": "COMPARISON_OPERATOR_EQUAL", "left": ` + projected + `, "right": {"literal": {"textValue": "t"}}}}]}}}`,
		},
		{
			name:            "rule without a guard",
			tree:            message("CorrelatedEvidenceRule", map[string]any{"kind": "k"}),
			wantErrorSubstr: "evidence rule carries no guard path",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mapped, err := Declared.apply("fixture.json", tc.tree)
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

// The fault step rewrites only the two fault coordinates, by name or number, into payload paths.
func TestDeclaredFaultCoordinatesBecomePayloadPaths(t *testing.T) {
	t.Parallel()

	expression := func(field any, extra map[string]any) *Object {
		fields := map[string]any{"field": field}
		maps.Copy(fields, extra)
		return &Object{Message: protocol + "Contract" + "Expression", Fields: map[string]any{"runEvent": &Object{Fields: fields}}}
	}
	const payload = `{"reference": {"runEvent": {"payload": {}}}}`
	for _, tc := range []struct {
		name            string
		tree            *Object
		want            string
		wantErrorSubstr string
	}{
		{name: "role", tree: expression("RUN_EVENT_FIELD_"+"FAULT_ROLE_ID", nil), want: `{"path": {"operand": ` + payload + `, "path": {"segments": [{"field": "fault_injected"}, {"field": "role_id"}]}}}`},
		{name: "kind by number", tree: expression(json.Number("11"), nil), want: `{"path": {"operand": ` + payload + `, "path": {"segments": [{"field": "fault_injected"}, {"field": "kind"}]}}}`},
		{name: "common coordinate", tree: expression("RUN_EVENT_FIELD_KIND", nil), want: `{"reference": {"runEvent": {"field": "RUN_EVENT_FIELD_KIND"}}}`},
		{name: "a second key", tree: expression("RUN_EVENT_FIELD_"+"FAULT_KIND", map[string]any{"extra": "x"}), wantErrorSubstr: "fault coordinate reference carries 2 keys, want 1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mapped, err := Declared.apply("fixture.json", tc.tree)
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
