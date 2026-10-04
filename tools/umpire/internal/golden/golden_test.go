package golden

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestRenameSourcesRenamesOnlyWholePaths(t *testing.T) {
	cfg := Config{Renames: []Substitution{{Old: "a/B.scala.fixture", New: "a/B.scala"}}}
	require.JSONEq(t, `{"path":"a/B.scala","note":"a/B.scala.fixture:3"}`,
		string(cfg.RenameSources([]byte(`{"path":"a/B.scala.fixture","note":"a/B.scala.fixture:3"}`))))
}

func TestCaptureNeverReplacesAnExistingDestination(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "capture")
	files := map[string][]byte{"one.json": []byte("original\n")}
	require.NoError(t, Capture(dir, files))
	require.Error(t, Capture(dir, map[string][]byte{"one.json": []byte("changed\n")}))
	got, err := Read(dir)
	require.NoError(t, err)
	require.Equal(t, files["one.json"], got["one.json"])
}

func TestClosedMigrationRejectsUnlistedSourceChanges(t *testing.T) {
	cfg := Config{Paths: []Substitution{{Old: "old.scala", New: "new.scala"}}, Labels: []Substitution{{Old: "old model", New: "new model"}}}
	original := &umpirespb.Model{Source: "old model", Machines: []*umpirespb.Machine{{Position: &umpirespb.Position{File: "old.scala", Line: 12}}}}
	mapped, err := cfg.Migrate(original)
	require.NoError(t, err)
	require.Equal(t, "old.scala", original.Machines[0].Position.File)
	require.True(t, proto.Equal(&umpirespb.Position{File: "new.scala", Line: 12}, mapped.Machines[0].Position))
	moved, err := cfg.Match(original, mapped)
	require.NoError(t, err)
	require.True(t, moved)
	for _, change := range []func(*umpirespb.Model){
		func(m *umpirespb.Model) { m.Machines[0].Position.File = "other.scala" },
		func(m *umpirespb.Model) { m.Machines[0].Position.Line++ },
		func(m *umpirespb.Model) { m.Source += " unexpected" },
	} {
		changed := proto.CloneOf(mapped)
		change(changed)
		_, err := cfg.Match(original, changed)
		require.Error(t, err)
	}
	original.Machines[0].Position.File = "unlisted.scala"
	_, err = cfg.Migrate(original)
	require.ErrorContains(t, err, "unlisted.scala")
}

func TestRenamesFollowTheCapturedMigration(t *testing.T) {
	cfg := Config{
		Paths:   []Substitution{{Old: "old.scala", New: "new.scala.fixture"}, {Old: "kept.scala", New: "moved.scala"}},
		Labels:  []Substitution{{Old: "old model", New: "new model"}},
		Renames: []Substitution{{Old: "new.scala.fixture", New: "new.scala"}},
	}
	original := &umpirespb.Model{Source: "old model", Machines: []*umpirespb.Machine{
		{Position: &umpirespb.Position{File: "old.scala", Line: 12}},
		{Position: &umpirespb.Position{File: "kept.scala", Line: 3}},
	}}
	mapped, err := cfg.Migrate(original)
	require.NoError(t, err)
	require.Equal(t, "new.scala.fixture", mapped.Machines[0].Position.File)
	current, err := cfg.Rename(mapped)
	require.NoError(t, err)
	require.Equal(t, "new.scala.fixture", mapped.Machines[0].Position.File)
	require.True(t, proto.Equal(&umpirespb.Model{Source: "new model", Machines: []*umpirespb.Machine{
		{Position: &umpirespb.Position{File: "new.scala", Line: 12}},
		{Position: &umpirespb.Position{File: "moved.scala", Line: 3}},
	}}, current))
	moved, err := cfg.Match(original, current)
	require.NoError(t, err)
	require.True(t, moved)
	_, err = cfg.Match(original, mapped)
	require.Error(t, err, "the current IR has the renamed path, not the captured one")
	cfg.Renames = []Substitution{{Old: "old.scala", New: "other.scala"}}
	_, err = cfg.Rename(mapped)
	require.ErrorContains(t, err, "old.scala")
}

func TestCompareRequiresTheWholeInventory(t *testing.T) {
	original := map[string][]byte{"a": []byte("one"), "b": []byte("two")}
	require.NoError(t, Compare(original, original))
	for _, changed := range []map[string][]byte{
		{"a": []byte("one")},
		{"a": []byte("one"), "b": []byte("two"), "c": nil},
		{"a": []byte("changed"), "b": []byte("two")},
	} {
		require.Error(t, Compare(original, changed))
	}
}

func TestIRInventoryRejectsMissingAndUnknownFiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "model", "scalav2", "ir")
	require.NoError(t, os.MkdirAll(dir, 0755))
	cfg := Config{Inventory: []string{"model/scalav2/ir/one.json"}}
	_, err := cfg.Inputs(root)
	require.ErrorContains(t, err, "missing IR inventory entry")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "one.json"), []byte(`{}`), 0644))
	inputs, err := cfg.Inputs(root)
	require.NoError(t, err)
	require.Len(t, inputs, 1)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "other.json"), []byte(`{}`), 0644))
	_, err = cfg.Inputs(root)
	require.ErrorContains(t, err, "unknown IR inventory entry")
	// A file added after the goldens were captured is admitted where listed as later, and not compared.
	cfg.Later = []string{"model/scalav2/ir/other.json"}
	inputs, err = cfg.Inputs(root)
	require.NoError(t, err)
	require.Len(t, inputs, 1)
	require.NoError(t, os.Remove(filepath.Join(dir, "other.json")))
	_, err = cfg.Inputs(root)
	require.ErrorContains(t, err, "missing later IR inventory entry")
}

// The law sidecar and the accepted lint findings beside an IR file are no Model: neither is listed as
// an IR file, and the accepted findings, which the lifter does not produce, are no current file the
// archive freezes.
func TestIRFilesLeaveOutLawSidecarsAndAcceptedFindings(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "model", "ir")
	require.NoError(t, os.MkdirAll(dir, 0755))
	for _, name := range []string{"activity.json", "activity.laws.json", "activity.lint.json", "nexus.json", "notes.txt"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(`{}`), 0644))
	}
	paths, err := IRFiles(dir)
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(dir, "activity.json"), filepath.Join(dir, "nexus.json")}, paths)
	current, err := OriginalCurrent(root)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{OriginalIR + "activity.json", OriginalIR + "activity.laws.json", OriginalIR + "nexus.json"},
		slices.Collect(maps.Keys(current)))
}

func projectedJob(t *testing.T) (cfg Config, original, current *umpirespb.Model) {
	t.Helper()
	cfg = Config{
		Paths:  []Substitution{{Old: "job.go", New: "moved.go"}},
		Labels: []Substitution{{Old: "job.go", New: "moved"}},
		Projection: Projection{PositionsByFile: true, AlphaParameters: true,
			Functions: []Substitution{{Old: "job.finishes", New: "job.completes"}}},
	}
	original = JobModel("once", "submit", "take", "finish")
	current, err := cfg.Migrate(original)
	require.NoError(t, err)
	renameFunction(current, "job.finishes", "job.completes")
	return cfg, original, current
}

func renameFunction(m *umpirespb.Model, old, name string) {
	for _, f := range m.Functions {
		if f.Name == old {
			f.Name = name
		}
	}
	for _, p := range m.Properties {
		if p.Holds == old {
			p.Holds = name
		}
	}
	for _, machine := range m.Machines {
		if machine.Evidence == old {
			machine.Evidence = name
		}
	}
}

func functionNamed(m *umpirespb.Model, name string) *umpirespb.Function {
	for _, f := range m.Functions {
		if f.Name == name {
			return f
		}
	}
	return nil
}

func TestProjectionAdmitsOnlyLiftedTextChanges(t *testing.T) {
	cfg, original, current := projectedJob(t)
	shift := func(m *umpirespb.Model) {
		require.NoError(t, positions(m.ProtoReflect(), func(at protoreflect.Message) error {
			field := at.Descriptor().Fields().ByName("line")
			at.Set(field, protoreflect.ValueOfInt32(int32(at.Get(field).Int())+7))
			return nil
		}))
	}
	renameParameters := func(m *umpirespb.Model) {
		evidence := functionNamed(m, "job.evidence")
		evidence.Params[0].Name = "it"
		evidence.Body.GetMatch().Scrutinee.Kind = &umpirespb.Expr_Var{Var: "it"}
		ends := m.Machines[0].Ends.GetLambda()
		ends.Params[0].Name = "s"
		for _, side := range []*umpirespb.Expr{ends.Body.GetBinary().Left, ends.Body.GetBinary().Right} {
			side.GetBinary().Left.Kind = &umpirespb.Expr_Var{Var: "s"}
		}
	}
	admitted := map[string]func(*umpirespb.Model){
		"listed function rename": func(*umpirespb.Model) {},
		"lines":                  shift,
		"parameters":             renameParameters,
		"together": func(m *umpirespb.Model) {
			shift(m)
			renameParameters(m)
		},
	}
	for name, change := range admitted {
		t.Run(name, func(t *testing.T) {
			changed := proto.CloneOf(current)
			change(changed)
			moved, err := cfg.Match(original, changed)
			require.NoError(t, err)
			require.True(t, moved)
		})
	}
	rejected := map[string]func(*umpirespb.Model){
		"table row": func(m *umpirespb.Model) {
			step := functionNamed(m, "job.take.step").Body.GetIf().Then.GetList().Items[0].GetConstruct()
			step.Args[1].GetLiteral().GetEnum().Case = "waiting"
		},
		"unlisted function rename": func(m *umpirespb.Model) { renameFunction(m, "job.evidence", "job.evidenceOf") },
		"listed function rename not made": func(m *umpirespb.Model) {
			shift(m)
			renameFunction(m, "job.completes", "job.finishes")
		},
		"position in an unlisted file": func(m *umpirespb.Model) {
			shift(m)
			m.Machines[0].Position.File = "other.go"
		},
		"renamed parameter with a different body": func(m *umpirespb.Model) {
			functionNamed(m, "job.evidence").Params[0].Name = "it"
		},
		"renamed parameter with a changed operator": func(m *umpirespb.Model) {
			renameParameters(m)
			ends := m.Machines[0].Ends.GetLambda()
			ends.Body.GetBinary().Op = umpirespb.Binary_OP_AND
		},
		"function renamed to a listed old name": func(m *umpirespb.Model) { renameFunction(m, "job.evidence", "job.finishes") },
	}
	for name, change := range rejected {
		t.Run(name, func(t *testing.T) {
			changed := proto.CloneOf(current)
			change(changed)
			_, err := cfg.Match(original, changed)
			require.Error(t, err)
		})
	}
	require.Equal(t, "moved.go", current.Machines[0].Position.File, "the projection compares copies")
	require.Equal(t, int32(1), current.Machines[0].Position.Line)
}

func TestProjectionIsClosed(t *testing.T) {
	cfg, original, current := projectedJob(t)
	require.NoError(t, cfg.FunctionsRenamed(map[string]*umpirespb.Model{"job": original}))
	cfg.Projection.Functions = append(cfg.Projection.Functions, Substitution{Old: "job.unknown", New: "job.other"})
	require.ErrorContains(t, cfg.FunctionsRenamed(map[string]*umpirespb.Model{"job": original}), "job.unknown")

	cfg.Projection.Functions = []Substitution{{Old: "job.finishes", New: "job.evidence"}}
	_, err := cfg.Match(original, current)
	require.ErrorContains(t, err, "two Functions")

	cfg.Projection = Projection{}
	renameFunction(current, "job.completes", "job.finishes")
	moved, err := cfg.Match(original, current)
	require.NoError(t, err)
	require.True(t, moved)
	for _, change := range []func(*umpirespb.Model){
		func(m *umpirespb.Model) { m.Machines[0].Position.Line++ },
		func(m *umpirespb.Model) { functionNamed(m, "job.evidence").Params[0].Name = "it" },
	} {
		changed := proto.CloneOf(current)
		change(changed)
		_, err := cfg.Match(original, changed)
		require.Error(t, err, "a projection the configuration does not declare is not applied")
	}
}

// TestFunctionProjectionReadsNoFunction compares Models without their Functions and with each set
// Function reference as one token, so a function-body-only change passes the IR comparison; what a
// body means is compared on what the current IR reads and lowers to (see FunctionsByReference).
// Everything else is still compared exactly.
func TestFunctionProjectionReadsNoFunction(t *testing.T) {
	cfg, original, current := projectedJob(t)
	cfg.Projection.FunctionsByReference = true
	not := func(e *umpirespb.Expr) *umpirespb.Expr {
		return &umpirespb.Expr{Position: e.GetPosition(), Kind: &umpirespb.Expr_Unary{Unary: &umpirespb.Unary{Op: umpirespb.Unary_OP_NOT, Operand: e}}}
	}
	admitted := map[string]func(*umpirespb.Model){
		"renamed step with a guard negated twice": func(m *umpirespb.Model) {
			guard := functionNamed(m, "job.take.step").Body.GetIf()
			guard.Condition = not(not(guard.Condition))
			functionNamed(m, "job.take.step").Name = "job.Take$.step"
			m.Machines[0].Steps[1].Function = "job.Take$.step"
		},
		"unlisted function rename": func(m *umpirespb.Model) { renameFunction(m, "job.evidence", "job.evidenceOf") },
		"flipped guard": func(m *umpirespb.Model) {
			guard := functionNamed(m, "job.take.step").Body.GetIf()
			guard.Then, guard.Else = guard.Else, guard.Then
		},
		"inventory": func(m *umpirespb.Model) {
			m.Functions = append(m.Functions[1:], &umpirespb.Function{Name: "job.helper", Body: m.Functions[0].Body})
		},
		"renamed parameter with a different body": func(m *umpirespb.Model) {
			functionNamed(m, "job.evidence").Params[0].Name = "it"
		},
	}
	for name, change := range admitted {
		t.Run(name, func(t *testing.T) {
			changed := proto.CloneOf(current)
			change(changed)
			moved, err := cfg.Match(original, changed)
			require.NoError(t, err)
			require.True(t, moved)
			unprojected := cfg
			unprojected.Projection.FunctionsByReference = false
			_, err = unprojected.Match(original, changed)
			require.Error(t, err, "a projection the configuration does not declare is not applied")
		})
	}
	rejected := map[string]func(*umpirespb.Model){
		"state key":       func(m *umpirespb.Model) { m.Types[0].GetEnum().Cases[1].Name = "pending" },
		"definition ID":   func(m *umpirespb.Model) { m.Actions[0].Id = "job.Moved$.submit" },
		"step removed":    func(m *umpirespb.Model) { m.Machines[0].Steps = m.Machines[0].Steps[1:] },
		"step's action":   func(m *umpirespb.Model) { m.Machines[0].Steps[1].Action = "job.drop" },
		"no evidence":     func(m *umpirespb.Model) { m.Machines[0].Evidence = "" },
		"query limits":    func(m *umpirespb.Model) { m.Queries[0].Limits.Steps++ },
		"ends operator":   func(m *umpirespb.Model) { m.Machines[0].Ends.GetLambda().Body.GetBinary().Op = umpirespb.Binary_OP_AND },
		"position's file": func(m *umpirespb.Model) { m.Machines[0].Position.File = "other.go" },
		"source":          func(m *umpirespb.Model) { m.Source += " unexpected" },
	}
	for name, change := range rejected {
		t.Run(name, func(t *testing.T) {
			changed := proto.CloneOf(current)
			change(changed)
			_, err := cfg.Match(original, changed)
			require.Error(t, err)
		})
	}
}

func TestTypeNameProjectionIsClosed(t *testing.T) {
	original := &umpirespb.Model{
		Source: "old source",
		Types: []*umpirespb.Type{
			{Name: "old.Outcome", Shape: &umpirespb.Type_Enum{Enum: &umpirespb.Enum{Cases: []*umpirespb.Case{{Name: "accepted"}, {Name: "rejected"}}}}},
			{Name: "old.State", Shape: &umpirespb.Type_Record{Record: &umpirespb.Record{Fields: []*umpirespb.Field{{Name: "phase"}, {Name: "attempts"}}}}},
		},
		Machines: []*umpirespb.Machine{{StateType: "old.State", OutcomeType: "old.Outcome"}},
	}
	cfg := Config{Labels: []Substitution{{Old: "old source", New: "new source"}}, Projection: Projection{Types: []Substitution{
		{Old: "old.State", New: "new.State"},
		{Old: "old.Outcome", New: "new.Outcome"},
	}}}
	reference := map[string]*umpirespb.Model{"model": original}
	require.NoError(t, cfg.TypesRenamed(reference))
	current := proto.CloneOf(original)
	current.Source = "new source"
	current.Types[0].Name = "new.Outcome"
	current.Types[1].Name = "new.State"
	current.Machines[0].StateType = "new.State"
	current.Machines[0].OutcomeType = "new.Outcome"
	moved, err := cfg.Match(original, current)
	require.NoError(t, err)
	require.True(t, moved)

	for name, change := range map[string]func(*umpirespb.Model){
		"unlisted declaration": func(m *umpirespb.Model) { m.Types[0].Name = "other.State" },
		"unlisted reference":   func(m *umpirespb.Model) { m.Machines[0].StateType = "other.State" },
		"declaration order": func(m *umpirespb.Model) {
			m.Types[0], m.Types[1] = m.Types[1], m.Types[0]
		},
		"enum case order": func(m *umpirespb.Model) {
			cases := m.Types[0].GetEnum().Cases
			cases[0], cases[1] = cases[1], cases[0]
		},
		"record field order": func(m *umpirespb.Model) {
			fields := m.Types[1].GetRecord().Fields
			fields[0], fields[1] = fields[1], fields[0]
		},
	} {
		t.Run(name, func(t *testing.T) {
			changed := proto.CloneOf(current)
			change(changed)
			_, err := cfg.Match(original, changed)
			require.Error(t, err)
		})
	}
	for name, substitutions := range map[string][]Substitution{
		"unknown source":   {{Old: "old.Unknown", New: "new.Unknown"}},
		"duplicate source": {{Old: "old.State", New: "new.State"}, {Old: "old.State", New: "new.Other"}},
		"duplicate target": {{Old: "old.State", New: "new.State"}, {Old: "old.Outcome", New: "new.State"}},
		"existing target":  {{Old: "old.State", New: "old.Outcome"}},
	} {
		t.Run(name, func(t *testing.T) {
			cfg.Projection.Types = substitutions
			require.Error(t, cfg.TypesRenamed(reference))
		})
	}
}

func TestFunctionOnlyProjectionKeepsTypeOrder(t *testing.T) {
	cfg, original, current := projectedJob(t)
	cfg.Projection.Types = []Substitution{{Old: "other.State", New: "other.MovedState"}}
	reference := map[string]*umpirespb.Model{
		"job":   original,
		"other": {Types: []*umpirespb.Type{{Name: "other.State"}}},
	}
	require.NoError(t, cfg.TypesRenamed(reference))
	moved, err := cfg.Match(original, current)
	require.NoError(t, err)
	require.True(t, moved)
	changed := proto.CloneOf(current)
	changed.Types[0], changed.Types[1] = changed.Types[1], changed.Types[0]
	_, err = cfg.Match(original, changed)
	require.Error(t, err)
}

func TestAlphaNormalizationRespectsShadowing(t *testing.T) {
	variable := func(name string) *umpirespb.Expr { return &umpirespb.Expr{Kind: &umpirespb.Expr_Var{Var: name}} }
	lambda := func(param string, body *umpirespb.Expr) *umpirespb.Expr {
		return &umpirespb.Expr{Kind: &umpirespb.Expr_Lambda{Lambda: &umpirespb.Lambda{Params: []*umpirespb.Param{{Name: param}}, Body: body}}}
	}
	let := func(name string, value, body *umpirespb.Expr) *umpirespb.Expr {
		return &umpirespb.Expr{Kind: &umpirespb.Expr_Let{Let: &umpirespb.Let{Name: name, Value: value, Body: body}}}
	}
	bind := func(name string, body *umpirespb.Expr) *umpirespb.Expr {
		return &umpirespb.Expr{Kind: &umpirespb.Expr_Match{Match: &umpirespb.Match{Scrutinee: variable("outer"), Cases: []*umpirespb.MatchCase{{
			Pattern: &umpirespb.Pattern{Kind: &umpirespb.Pattern_Bind{Bind: &umpirespb.Bind{Name: name, Pattern: &umpirespb.Pattern{Kind: &umpirespb.Pattern_Wildcard{Wildcard: &umpirespb.Empty{}}}}}},
			Body:    body,
		}}}}}
	}
	model := func(e *umpirespb.Expr) *umpirespb.Model {
		return &umpirespb.Model{Machines: []*umpirespb.Machine{{Name: "m", Ends: e}}}
	}
	equal := func(x, y *umpirespb.Expr) bool {
		p := Projection{AlphaParameters: true}
		a, err := p.project(model(x), false)
		require.NoError(t, err)
		b, err := p.project(model(y), false)
		require.NoError(t, err)
		return proto.Equal(a, b)
	}
	require.True(t, equal(lambda("x", let("x", variable("x"), variable("x"))), lambda("y", let("x", variable("y"), variable("x")))))
	require.False(t, equal(lambda("x", let("x", variable("x"), variable("x"))), lambda("y", let("x", variable("y"), variable("y")))))
	require.True(t, equal(lambda("x", bind("x", variable("x"))), lambda("y", bind("x", variable("x")))))
	require.False(t, equal(lambda("x", bind("x", variable("x"))), lambda("y", bind("x", variable("y")))))
	require.True(t, equal(lambda("x", lambda("y", variable("x"))), lambda("a", lambda("b", variable("a")))))
	require.False(t, equal(lambda("x", lambda("y", variable("x"))), lambda("a", lambda("b", variable("b")))))
	require.False(t, equal(lambda("x", variable("free")), lambda("x", variable("other"))), "a free variable keeps its name")
	pair := func(first, second string, body *umpirespb.Expr) *umpirespb.Expr {
		return &umpirespb.Expr{Kind: &umpirespb.Expr_Lambda{Lambda: &umpirespb.Lambda{Params: []*umpirespb.Param{{Name: first}, {Name: second}}, Body: body}}}
	}
	minus := func(left, right string) *umpirespb.Expr {
		return &umpirespb.Expr{Kind: &umpirespb.Expr_Binary{Binary: &umpirespb.Binary{Op: umpirespb.Binary_OP_SUB, Left: variable(left), Right: variable(right)}}}
	}
	require.True(t, equal(pair("x", "y", minus("x", "y")), pair("a", "b", minus("a", "b"))))
	require.False(t, equal(pair("x", "y", minus("x", "y")), pair("y", "x", minus("x", "y"))), "parameters compare by their place")
	require.True(t, equal(lambda("x", lambda("x", variable("x"))), lambda("a", lambda("b", variable("b")))), "an inner parameter hides an outer one of its name")
	require.False(t, equal(lambda("x", lambda("x", variable("x"))), lambda("a", lambda("b", variable("a")))))
}

func TestCaseProjectionDropsOnlyAnExplorationID(t *testing.T) {
	cfg, err := Configuration()
	require.NoError(t, err)
	require.Equal(t, []string{ExplorationCase}, cfg.Projection.CaseIDs)
	p := cfg.Projection
	compare := func(kind string, want []byte, wantID string, got []byte, gotID string) error {
		x, err := p.Case(kind, want, wantID)
		if err != nil {
			return err
		}
		y, err := p.Case(kind, got, gotID)
		if err != nil {
			return err
		}
		return Compare(map[string][]byte{"case": x}, map[string][]byte{"case": y})
	}
	explored := func(id, step string) []byte {
		return []byte(`{"caseId":"temporal.case.scala.explore.plan.` + id + `","program":{"programId":"temporal.case.scala.explore.plan.` + id + `.program","step":"` + step + `"}}`)
	}
	a, b, c := strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64)
	require.NoError(t, compare(ExplorationCase, explored(a, "take"), a, explored(b, "take"), b))
	require.Error(t, compare(ExplorationCase, explored(a, "take"), a, explored(b, "drop"), b), "a changed non-ID byte")
	require.Error(t, compare(ExplorationCase, explored(a, "take"), a, explored(b, "take"), c), "an ID the Case does not carry")
	require.Error(t, compare(ExplorationCase, explored(a, "take"), "plan", explored(b, "take"), "plan"), "an ID shorter than a digest")
	require.Error(t, compare(ExplorationCase, explored(a, "take"), strings.ToUpper(a), explored(b, "take"), b), "an ID not in lowercase hex")
	queried := func(id string) []byte { return []byte(`{"caseId":"temporal.case.scala.nexus-caller.` + id + `"}`) }
	require.NoError(t, compare(QueryCase, queried("retry"), "retry", queried("retry"), "retry"))
	require.Error(t, compare(QueryCase, queried("retry"), "retry", queried("retried"), "retried"), "a Query Case's ID stays strict")
	_, err = p.Case("other", queried("retry"), "retry")
	require.ErrorContains(t, err, "unknown lowered Case kind")
	p.CaseIDs = append(p.CaseIDs, "other")
	_, err = p.Case(ExplorationCase, explored(a, "take"), a)
	require.ErrorContains(t, err, "unknown lowered Case kind")
}

// A declaration split out of a file compares as one of the file it left, and a moved root by its
// frozen name in the frozen order; positions in other files and other roots are kept.
func TestUnsplitNamesTheFileAndRootADeclarationLeft(t *testing.T) {
	cfg := Config{
		Splits:    []Substitution{{Old: "model/b/Queue.scala", New: "model/a/System.scala"}},
		RootMoves: []Substitution{{Old: "a.System$package$.queue", New: "b.Queue$package$.queue"}},
	}
	current := &umpirespb.Model{
		Source: "model: a.System$package$.record, b.Queue$package$.queue",
		Machines: []*umpirespb.Machine{
			{Name: "queue", Position: &umpirespb.Position{File: "model/b/Queue.scala", Line: 3}},
			{Name: "record", Position: &umpirespb.Position{File: "model/a/Other.scala", Line: 4}},
		},
	}
	unsplit, err := cfg.Unsplit(current)
	require.NoError(t, err)
	require.Equal(t, "model: a.System$package$.queue, a.System$package$.record", unsplit.GetSource())
	require.Equal(t, "model/a/System.scala", unsplit.GetMachines()[0].GetPosition().GetFile())
	require.Equal(t, "model/a/Other.scala", unsplit.GetMachines()[1].GetPosition().GetFile())
	require.Equal(t, "model/b/Queue.scala", current.GetMachines()[0].GetPosition().GetFile(), "the current Model is kept")
}

func TestMergeNamesTheDirectoryOfAPosition(t *testing.T) {
	cfg := Config{
		Paths:  []Substitution{{Old: "old/a/Claims.scala", New: "model/a/Claims.scala"}, {Old: "old/b/Model.scala", New: "model/b/Model.scala"}},
		Labels: []Substitution{{Old: "old model", New: "model"}},
		Merges: []Substitution{{Old: "model/a/", New: "model/a"}},
	}
	original := &umpirespb.Model{Source: "old model", Machines: []*umpirespb.Machine{
		{Name: "claim", Position: &umpirespb.Position{File: "old/a/Claims.scala", Line: 3}},
		{Name: "other", Position: &umpirespb.Position{File: "old/b/Model.scala", Line: 4}},
	}}
	current := &umpirespb.Model{Source: "model", Machines: []*umpirespb.Machine{
		{Name: "claim", Position: &umpirespb.Position{File: "model/a/admission/Properties.scala", Line: 3}},
		{Name: "other", Position: &umpirespb.Position{File: "model/b/Model.scala", Line: 4}},
	}}
	moved, err := cfg.Match(original, current)
	require.NoError(t, err)
	require.True(t, moved)
	require.Equal(t, "model/a/admission/Properties.scala", current.GetMachines()[0].GetPosition().GetFile(), "the current Model is kept")

	left := proto.CloneOf(current)
	left.Machines[0].Position.File = "model/b/Model.scala"
	_, err = cfg.Match(original, left)
	require.Error(t, err, "a declaration leaving the merged directory still fails")
	entered := proto.CloneOf(current)
	entered.Machines[1].Position.File = "model/a/Model.scala"
	_, err = cfg.Match(original, entered)
	require.Error(t, err, "a declaration entering the merged directory still fails")

	cfg.Merges = []Substitution{{Old: "model/a", New: "model/a"}}
	_, err = cfg.Match(original, current)
	require.ErrorContains(t, err, "not a directory or a Scala file")
}

func TestMergeOfAFileNamesOnlyThatFile(t *testing.T) {
	cfg := Config{
		Paths: []Substitution{
			{Old: "old/a/Realization.scala", New: "model/a/Realization.scala"},
			{Old: "old/a/Model.scala", New: "model/a/Model.scala"},
		},
		Labels: []Substitution{{Old: "old model", New: "model"}},
		Merges: []Substitution{
			{Old: "model/kit/", New: "model/shared"},
			{Old: "model/a/Realization.scala", New: "model/shared"},
			{Old: "model/a/", New: "model/a"},
		},
	}
	original := &umpirespb.Model{Source: "old model", Machines: []*umpirespb.Machine{
		{Name: "role", Position: &umpirespb.Position{File: "old/a/Realization.scala", Line: 3}},
		{Name: "machine", Position: &umpirespb.Position{File: "old/a/Model.scala", Line: 4}},
	}}
	current := &umpirespb.Model{Source: "model", Machines: []*umpirespb.Machine{
		{Name: "role", Position: &umpirespb.Position{File: "model/kit/Kit.scala", Line: 3}},
		{Name: "machine", Position: &umpirespb.Position{File: "model/a/Model.scala", Line: 4}},
	}}
	moved, err := cfg.Match(original, current)
	require.NoError(t, err)
	require.True(t, moved, "a declaration moving from the merged file into the kit compares alike")
	leaving := proto.CloneOf(current)
	leaving.Machines[1].Position.File = "model/kit/Kit.scala"
	_, err = cfg.Match(original, leaving)
	require.Error(t, err, "a declaration of another file of the directory moving into the kit still fails")
}

// A Query moved from one file of its package to a new one: the Case it lowers to names the file the
// Query left, as Match compares its position, and a Case of another file is kept.
func TestUnsplitSourcesNamesTheFileAQueryLeft(t *testing.T) {
	cfg := Config{Splits: []Substitution{
		{Old: "model/a/Properties.scala", New: "model/a/Claims.scala"},
		{Old: "model/a/Queries.scala", New: "model/a/Claims.scala"},
	}}
	require.JSONEq(t, `{"path":"model/a/Claims.scala","sub":"model/a/Claims.scala","other":"model/a/Model.scala","note":"at model/a/Queries.scala:3"}`,
		string(cfg.UnsplitSources([]byte(`{"path":"model/a/Queries.scala","sub":"model/a/Properties.scala","other":"model/a/Model.scala","note":"at model/a/Queries.scala:3"}`))))

	original := &umpirespb.Model{Source: "model: a.Claims$package$.queries", Queries: []*umpirespb.Query{
		{Name: "retry", Position: &umpirespb.Position{File: "model/a/Claims.scala", Line: 30}},
	}}
	cfg.Paths = []Substitution{{Old: "model/a/Claims.scala", New: "model/a/Claims.scala"}}
	cfg.Labels = []Substitution{{Old: original.GetSource(), New: original.GetSource()}}
	cfg.RootMoves = []Substitution{{Old: "a.Claims$package$.queries", New: "a.Queries$package$.queries"}}
	cfg.Projection.PositionsByFile = true
	current := &umpirespb.Model{Source: "model: a.Queries$package$.queries", Queries: []*umpirespb.Query{
		{Name: "retry", Position: &umpirespb.Position{File: "model/a/Queries.scala", Line: 4}},
	}}
	moved, err := cfg.Match(original, current)
	require.NoError(t, err)
	require.True(t, moved, "the IR of the moved Query compares alike")
}

func TestMergeSourcesRewritesOnlyQuotedPathsUnderTheDirectory(t *testing.T) {
	cfg := Config{Merges: []Substitution{{Old: "model/a/", New: "model/a"}}}
	require.JSONEq(t, `{"path":"model/a","sub":"model/a","other":"model/b/Model.scala","note":"at model/a/Model.scala:3"}`,
		string(cfg.MergeSources([]byte(`{"path":"model/a/Model.scala","sub":"model/a/admission/Queries.scala","other":"model/b/Model.scala","note":"at model/a/Model.scala:3"}`))))
}

// A retired root leaves the frozen input's source and an added root the current Model's, so the two
// name the same roots; a retirement, addition or move that names no root fails.
func TestRootRetirementsAndAdditionsAreClosed(t *testing.T) {
	cfg := Config{
		Labels:          []Substitution{{Old: "frozen: a.Claims$package$.holds, a.Claims$package$.queries", New: "model: a.Claims$package$.holds, a.Claims$package$.queries"}},
		RootMoves:       []Substitution{{Old: "a.Claims$package$.queries", New: "a.Queries$package$.queries"}},
		RootRetirements: []string{"a.Claims$package$.holds"},
		RootAdditions:   []string{"a.Properties$package$.capabilities"},
	}
	original := &umpirespb.Model{Source: "frozen: a.Claims$package$.holds, a.Claims$package$.queries"}
	current := &umpirespb.Model{Source: "model: a.Properties$package$.capabilities, a.Queries$package$.queries"}
	mapped, err := cfg.Migrate(original)
	require.NoError(t, err)
	retired, err := cfg.Retired(mapped)
	require.NoError(t, err)
	unsplit, err := cfg.Unsplit(current)
	require.NoError(t, err)
	require.Equal(t, "model: a.Claims$package$.queries", retired.GetSource())
	require.Equal(t, retired.GetSource(), unsplit.GetSource())
	require.Equal(t, "model: a.Claims$package$.holds, a.Claims$package$.queries", mapped.GetSource(), "the mapped original is kept")
	originals, currents := map[string]*umpirespb.Model{"a": original}, map[string]*umpirespb.Model{"a": current}
	require.NoError(t, cfg.RootsApply(originals, currents))
	for name, change := range map[string]func(*Config){
		"retirement": func(c *Config) { c.RootRetirements = append(c.RootRetirements, "a.Claims$package$.other") },
		"addition":   func(c *Config) { c.RootAdditions = append(c.RootAdditions, "a.Properties$package$.other") },
		"dead move": func(c *Config) {
			c.RootMoves = append(c.RootMoves, Substitution{Old: "a.Claims$package$.x", New: "a.Queries$package$.x"})
		},
		"current root": func(c *Config) { c.RootRetirements = []string{"a.Queries$package$.queries"} },
		"frozen root":  func(c *Config) { c.RootAdditions = []string{"a.Claims$package$.holds"} },
	} {
		changed := cfg
		changed.RootRetirements, changed.RootAdditions, changed.RootMoves = slices.Clone(cfg.RootRetirements), slices.Clone(cfg.RootAdditions), slices.Clone(cfg.RootMoves)
		change(&changed)
		require.Error(t, changed.RootsApply(originals, currents), name)
	}
}

func TestOriginalKeyNamesTheArchivedFile(t *testing.T) {
	require.Equal(t, "ir/activity.json", OriginalKey("model/scalav2/ir/activity.json"))
	require.Equal(t, "lifts/admission.json", OriginalKey("model/scalav2/lifter/testdata/lifts/expected/admission.json"))
}
