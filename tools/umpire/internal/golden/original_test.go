package golden

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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

var captureOriginal = flag.String("capture-original", "", "archive the original baseline's inputs into this new directory")

func TestCaptureOriginalBaseline(t *testing.T) {
	if *captureOriginal == "" {
		t.Skip("explicit -capture-original=<new directory> required")
	}
	root, err := Root()
	require.NoError(t, err)
	require.NoError(t, CaptureOriginal(root, *captureOriginal))
	archived, err := Read(filepath.Join(*captureOriginal, "archive"))
	require.NoError(t, err)
	encoded, err := ownersOf(t, archived)
	require.NoError(t, err)
	file, err := os.OpenFile(filepath.Join(*captureOriginal, "owners.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	require.NoError(t, err)
	_, err = file.Write(encoded)
	require.NoError(t, err)
	require.NoError(t, file.Close())
}

// TestOriginalBaselineInputs holds every current IR Model, lifter fixture and refusal to the
// archive: the same files, and each Model equal to its baseline but for positions and the delta.
func TestOriginalBaselineInputs(t *testing.T) {
	if *captureOriginal != "" {
		t.Skip("capture is separate from verification")
	}
	root, err := Root()
	require.NoError(t, err)
	delta, err := OriginalDelta()
	require.NoError(t, err)
	archived, err := OriginalArchive(root)
	require.NoError(t, err)
	current, err := OriginalCurrent(root)
	require.NoError(t, err)
	require.NoError(t, delta.Inventory(archived, current))
	baselines, err := OriginalModels(archived)
	require.NoError(t, err)
	models, err := OriginalModels(current)
	require.NoError(t, err)
	require.Len(t, baselines, 12, "six IR Models and six positive lifter fixtures")
	applied := Applied{}
	for _, key := range slices.Sorted(maps.Keys(baselines)) {
		expected, err := delta.Expected(key, baselines[key], applied)
		require.NoError(t, err, key)
		ungenerated, err := delta.Ungenerated(key, models[key])
		require.NoError(t, err, key)
		require.NoError(t, delta.MatchOriginal(expected, ungenerated), key)
	}
	require.NoError(t, delta.Unapplied(applied))
}

// owner splits a symbol-based Definition ID into the compiler owner the lifter took it from, which
// ends at the last segment that names an object or a file's package object, and the captured name.
func owner(id string) (at, name string, ok bool) {
	segments := strings.Split(id, ".")
	for i := len(segments) - 1; i > 0; i-- {
		if strings.HasSuffix(segments[i-1], "$") {
			return strings.Join(segments[:i], "."), strings.Join(segments[i:], "."), true
		}
	}
	return "", "", false
}

// ownersOf is the map of every symbol-based Definition ID of the archived Models: its former compiler
// owner, its kind and its captured names. A DefinitionScope pins one owner and reproduces each ID as
// the owner followed by the captured name.
func ownersOf(t *testing.T, archived map[string][]byte) ([]byte, error) {
	t.Helper()
	models, err := OriginalModels(archived)
	if err != nil {
		return nil, err
	}
	owners := map[string]map[string][]string{}
	add := func(kind, id string) {
		at, name, ok := owner(id)
		require.True(t, ok, "%s %s names no compiler owner", kind, id)
		if owners[at] == nil {
			owners[at] = map[string][]string{}
		}
		if !slices.Contains(owners[at][kind], name) {
			owners[at][kind] = append(owners[at][kind], name)
		}
	}
	for _, m := range models {
		for _, x := range m.GetActions() {
			add("action", x.GetId())
		}
		for _, x := range m.GetMonitors() {
			add("monitor", x.GetId())
		}
		for _, x := range m.GetAssumptions() {
			add("assumption", x.GetId())
		}
		for _, x := range m.GetHoles() {
			add("hole", x.GetId())
		}
		for _, x := range m.GetChannels() {
			add("channel", x.GetId())
		}
		for _, x := range m.GetRealizations() {
			add("realization", x.GetId())
		}
	}
	for _, kinds := range owners {
		for _, names := range kinds {
			slices.Sort(names)
		}
	}
	encoded, err := json.MarshalIndent(owners, "", "  ")
	return append(encoded, '\n'), err
}

func TestOriginalBaselineOwnerMapIsTheArchives(t *testing.T) {
	root, err := Root()
	require.NoError(t, err)
	archived, err := OriginalArchive(root)
	require.NoError(t, err)
	want, err := os.ReadFile(filepath.Join(root, OriginalDir, "owners.json"))
	require.NoError(t, err)
	got, err := ownersOf(t, archived)
	require.NoError(t, err)
	require.Equal(t, string(want), string(got))
	for _, id := range []string{"a.B$package$", "a.b.c", "x"} {
		_, _, ok := owner(id)
		require.False(t, ok, id)
	}
	at, name, ok := owner("fixture.channels.Channels$package$.radio.deliver")
	require.True(t, ok)
	require.Equal(t, []string{"fixture.channels.Channels$package$", "radio.deliver"}, []string{at, name})
}

// originalJob is a baseline, with a refinement, a monitor and an `ends` that calls a Function, and
// its current spelling: moved to another file and line, lifted from other roots, with one Function
// renamed.
func originalJob(t *testing.T) (baseline, current *umpirespb.Model) {
	t.Helper()
	baseline = JobModel("once", "submit", "take", "finish")
	baseline.Actions[0].Examples = []*umpirespb.Example{
		{Value: &umpirespb.Value{Kind: &umpirespb.Value_Text{Text: "a"}}},
		{Value: &umpirespb.Value{Kind: &umpirespb.Value_Text{Text: "b"}}},
	}
	baseline.Machines[0].Ends.GetLambda().Body = &umpirespb.Expr{Kind: &umpirespb.Expr_Call{Call: &umpirespb.Call{Function: "job.ends",
		Args: []*umpirespb.Expr{{Kind: &umpirespb.Expr_Var{Var: "state"}}}}}}
	baseline.Machines[0].Refines = &umpirespb.Refinement{Product: "queue", Map: "job.queued", Visible: "job.listed"}
	baseline.Monitors = []*umpirespb.Monitor{{Id: "job.watch", Name: "watch", Next: "job.watch.next", Violated: "job.watch.violated",
		Evaluate: &umpirespb.Monitor_After{After: "job.watch.after"}}}
	current = proto.CloneOf(baseline)
	current.Source = "model: moved roots"
	require.NoError(t, positions(current.ProtoReflect(), func(at protoreflect.Message) error {
		at.Set(at.Descriptor().Fields().ByName("file"), protoreflect.ValueOfString("moved.scala"))
		at.Set(at.Descriptor().Fields().ByName("line"), protoreflect.ValueOfInt32(40))
		return nil
	}))
	renameFunction(current, "job.finishes", "job.Finishing$.finishes")
	return baseline, current
}

func TestOriginalMatchAdmitsOnlyTheRecordedDelta(t *testing.T) {
	baseline, current := originalJob(t)
	require.NoError(t, Delta{}.MatchOriginal(baseline, current))

	const label = "temporal.server.api.umpire.v1.Example.example"
	labelled := proto.CloneOf(current)
	for i, e := range labelled.Actions[0].Examples {
		e.Example = []string{"first", "second"}[i]
	}
	inert := Delta{InertFields: []string{label}}
	require.NoError(t, inert.check())
	require.NoError(t, inert.MatchOriginal(baseline, labelled), "an inert name is admitted")
	require.Error(t, Delta{}.MatchOriginal(baseline, labelled), "a name no inert field lists")
	_, err := inert.ProjectBaseline(labelled)
	require.ErrorContains(t, err, "is set in the baseline")

	// Functions are not compared here: what they mean is compared on the outputs derived from the
	// Model, which even a changed table row changes.
	admitted := map[string]func(*umpirespb.Model){
		"function rename": func(m *umpirespb.Model) { renameFunction(m, "job.evidence", "job.evidenceOf") },
		"step function rename": func(m *umpirespb.Model) {
			functionNamed(m, "job.take.step").Name = "job.Take$.step"
			m.Machines[0].Steps[1].Function = "job.Take$.step"
		},
		"table row": func(m *umpirespb.Model) {
			step := functionNamed(m, "job.take.step").Body.GetIf().Then.GetList().Items[0].GetConstruct()
			step.Args[1].GetLiteral().GetEnum().Case = "waiting"
		},
		"restructured body": func(m *umpirespb.Model) {
			guard := functionNamed(m, "job.take.step").Body.GetIf()
			guard.Then, guard.Else = guard.Else, guard.Then
			guard.Condition = &umpirespb.Expr{Kind: &umpirespb.Expr_Unary{Unary: &umpirespb.Unary{Op: umpirespb.Unary_OP_NOT, Operand: guard.Condition}}}
		},
		"inventory": func(m *umpirespb.Model) {
			m.Functions = append(m.Functions[1:], &umpirespb.Function{Name: "job.helper", Body: m.Functions[0].Body})
		},
		"called function rename": func(m *umpirespb.Model) { m.Machines[0].Ends.GetLambda().Body.GetCall().Function = "job.Ends$.ends" },
	}
	for name, change := range admitted {
		t.Run(name, func(t *testing.T) {
			changed := proto.CloneOf(labelled)
			change(changed)
			require.NoError(t, inert.MatchOriginal(baseline, changed))
		})
	}

	rejected := map[string]func(*umpirespb.Model){
		"state key": func(m *umpirespb.Model) { m.Types[0].GetEnum().Cases[1].Name = "pending" },
		"definition ID": func(m *umpirespb.Model) {
			m.Actions[0].Id = "job.Moved$.submit"
			m.Machines[0].Steps[0].Action = "job.Moved$.submit"
		},
		"branch order":                     func(m *umpirespb.Model) { slices.Reverse(m.Actions[0].Examples) },
		"branch count":                     func(m *umpirespb.Model) { m.Actions[0].Examples = m.Actions[0].Examples[:1] },
		"query limits":                     func(m *umpirespb.Model) { m.Queries[0].Limits.Steps++ },
		"entity":                           func(m *umpirespb.Model) { m.Machines[0].Entity = "queue" },
		"no visibility projection":         func(m *umpirespb.Model) { m.Machines[0].Refines.Visible = "" },
		"an outcome visibility projection": func(m *umpirespb.Model) { m.Machines[0].Refines.VisibleOutcomes = "job.outcomes" },
		"no evidence":                      func(m *umpirespb.Model) { m.Machines[0].Evidence = "" },
		"evaluation point": func(m *umpirespb.Model) {
			m.Monitors[0].Evaluate = &umpirespb.Monitor_EveryStep{EveryStep: &umpirespb.Empty{}}
		},
		"step of another action": func(m *umpirespb.Model) { m.Machines[0].Steps[1].Action = "job.drop" },
		"step added": func(m *umpirespb.Model) {
			m.Machines[0].Steps = append(m.Machines[0].Steps, &umpirespb.StepBinding{Action: "job.take", Function: "job.take.step"})
		},
		"step removed": func(m *umpirespb.Model) { m.Machines[0].Steps = m.Machines[0].Steps[1:] },
		"call argument": func(m *umpirespb.Model) {
			m.Machines[0].Ends.GetLambda().Body.GetCall().Args[0].Kind = &umpirespb.Expr_Var{Var: "before"}
		},
	}
	for name, change := range rejected {
		t.Run(name, func(t *testing.T) {
			changed := proto.CloneOf(labelled)
			change(changed)
			require.Error(t, inert.MatchOriginal(baseline, changed))
		})
	}
}

// choiceField is the one inert field this delta's named choices add.
const choiceField = "temporal.server.api.umpire.v1.Construct.choice"

// nameSteps names every step record a Model constructs, anywhere in it, and counts them.
func nameSteps(t *testing.T, m *umpirespb.Model, name func(i int) string) int {
	t.Helper()
	n := 0
	require.NoError(t, messages(m.ProtoReflect(), func(child protoreflect.Message) (bool, error) {
		if c, ok := child.Interface().(*umpirespb.Construct); ok && c.GetType() == "umpire.Step" {
			c.Choice = name(n)
			n++
		}
		return true, nil
	}))
	return n
}

// TestOriginalBaselineAdmitsChoiceNames holds the delta's one choice-name allowance to exactly the
// names. The IR comparison drops the Functions, where every step record a step function returns is
// built, so it never reads a name there, with or without the allowance: what such a name could change
// is compared on the derived outputs instead (tables, answers, receipts, fingerprints and Cases), which
// the reader's and the lowering's named-choice tests show unchanged by names. A step record outside
// the Functions, here an argument the job's `ends` passes, is compared here, and only its name is
// excused: anything else of it that differs, another label, or a baseline that sets a name, is not.
func TestOriginalBaselineAdmitsChoiceNames(t *testing.T) {
	delta, err := OriginalDelta()
	require.NoError(t, err)
	require.Contains(t, delta.InertFields, choiceField)
	baseline, current := originalJob(t)
	step := proto.CloneOf(functionNamed(baseline, "job.close.step").Body.GetIf().Then.GetList().Items[0])
	for _, m := range []*umpirespb.Model{baseline, current} {
		ends := m.Machines[0].Ends.GetLambda().Body.GetCall()
		ends.Args = append(ends.Args, proto.CloneOf(step))
	}
	require.NoError(t, delta.MatchOriginal(baseline, current))

	inFunctions := proto.CloneOf(current)
	closing := functionNamed(inFunctions, "job.close.step").Body.GetIf().Then.GetList()
	require.Len(t, closing.Items, 2, "closing a running job finishes it or drops it")
	closing.Items[0].GetConstruct().Choice, closing.Items[1].GetConstruct().Choice = "finished", "dropped"
	require.NoError(t, delta.MatchOriginal(baseline, inFunctions))
	require.NoError(t, Delta{}.MatchOriginal(baseline, inFunctions), "the IR comparison reads no Function")

	named := proto.CloneOf(inFunctions)
	require.Equal(t, 10, nameSteps(t, named, func(i int) string { return fmt.Sprintf("alternative-%d", i) }))
	require.NoError(t, delta.MatchOriginal(baseline, named))
	require.Error(t, Delta{}.MatchOriginal(baseline, named), "a name outside the Functions is compared")
	require.ErrorContains(t, delta.MatchOriginal(named, named), "inert field "+choiceField+" is set in the baseline")
	require.ErrorContains(t, delta.MatchOriginal(inFunctions, named), "inert field "+choiceField+" is set in the baseline",
		"a name in a Function of the baseline too")

	outside := func(m *umpirespb.Model) *umpirespb.Construct {
		args := m.Machines[0].Ends.GetLambda().Body.GetCall().Args
		return args[len(args)-1].GetConstruct()
	}
	for name, change := range map[string]func(*umpirespb.Model){
		"type":     func(m *umpirespb.Model) { outside(m).Type = "JobState" },
		"case":     func(m *umpirespb.Model) { outside(m).Case = "accepted" },
		"argument": func(m *umpirespb.Model) { outside(m).Args[1].GetLiteral().GetEnum().Case = "waiting" },
		"result":   func(m *umpirespb.Model) { outside(m).Args = outside(m).Args[:3] },
		"label": func(m *umpirespb.Model) {
			m.Actions[0].Examples[0].Example = "first"
		},
	} {
		t.Run(name, func(t *testing.T) {
			changed := proto.CloneOf(named)
			change(changed)
			require.Error(t, delta.MatchOriginal(baseline, changed))
		})
	}
}

// TestOriginalBaselineAdmitsChoiceNamesOnEveryModel names every step record of every current IR Model
// and lifter fixture: each still matches its baseline under the delta, and a baseline so named is
// refused.
func TestOriginalBaselineAdmitsChoiceNamesOnEveryModel(t *testing.T) {
	root, err := Root()
	require.NoError(t, err)
	delta, err := OriginalDelta()
	require.NoError(t, err)
	archived, err := OriginalArchive(root)
	require.NoError(t, err)
	current, err := OriginalCurrent(root)
	require.NoError(t, err)
	baselines, err := OriginalModels(archived)
	require.NoError(t, err)
	models, err := OriginalModels(current)
	require.NoError(t, err)
	require.Len(t, baselines, 12)
	applied := Applied{}
	for _, key := range slices.Sorted(maps.Keys(baselines)) {
		expected, err := delta.Expected(key, baselines[key], applied)
		require.NoError(t, err, key)
		named, err := delta.Ungenerated(key, models[key])
		require.NoError(t, err, key)
		named = proto.CloneOf(named)
		require.Positive(t, nameSteps(t, named, func(i int) string { return fmt.Sprintf("alternative-%d", i) }), key)
		require.NoError(t, delta.MatchOriginal(expected, named), key)
		require.ErrorContains(t, delta.MatchOriginal(named, named), "is set in the baseline", key)
	}
}

// TestOriginalFunctionReferencesAreTheIRs checks that each projected reference is a string field of
// the IR, so a renamed field cannot silently stop being projected.
func TestOriginalFunctionReferencesAreTheIRs(t *testing.T) {
	require.Len(t, functionReferences, 12)
	for name := range functionReferences {
		field, err := inertField(string(name))
		require.NoError(t, err)
		require.Equal(t, protoreflect.StringKind, field.Kind(), name)
		require.False(t, field.IsList(), name)
	}
}

func TestOriginalEntityAttachmentsAreExact(t *testing.T) {
	baseline, current := originalJob(t)
	delta := Delta{
		Attachments: []Attachment{{Machine: "job", Field: "entity", Entity: "queue"}, {Action: "job.submit", Field: "on", Entity: "queue"}},
	}
	require.NoError(t, delta.check())
	applied := Applied{}
	expected, err := delta.Expected("", baseline, applied)
	require.NoError(t, err)
	require.NoError(t, delta.Unapplied(applied))
	require.Empty(t, baseline.Machines[0].Entity, "the baseline is not changed")
	attached := proto.CloneOf(current)
	attached.Machines[0].Entity = "queue"
	attached.Actions[0].On = "queue"
	require.NoError(t, delta.MatchOriginal(expected, attached))
	for name, change := range map[string]func(*umpirespb.Model){
		"not attached":      func(m *umpirespb.Model) { m.Machines[0].Entity = "" },
		"another entity":    func(m *umpirespb.Model) { m.Actions[0].On = "worker" },
		"unlisted action":   func(m *umpirespb.Model) { m.Actions[1].On = "queue" },
		"created, not kept": func(m *umpirespb.Model) { m.Actions[0].On, m.Actions[0].Creates = "", "queue" },
	} {
		t.Run(name, func(t *testing.T) {
			changed := proto.CloneOf(attached)
			change(changed)
			require.Error(t, delta.MatchOriginal(expected, changed))
		})
	}
	_, err = delta.Expected("", expected, Applied{})
	require.ErrorContains(t, err, "already has entity")
	missing := Delta{Attachments: []Attachment{{Machine: "nothing", Field: "entity", Entity: "queue"}}}
	applied = Applied{}
	_, err = missing.Expected("", baseline, applied)
	require.NoError(t, err)
	require.ErrorContains(t, missing.Unapplied(applied), "names no declaration")
}

func TestOriginalDeltaIsClosed(t *testing.T) {
	d, err := OriginalDelta()
	require.NoError(t, err)
	require.NoError(t, d.check())
	for name, invalid := range map[string]Delta{
		"unknown field":   {InertFields: []string{"temporal.server.api.umpire.v1.Query.nothing"}},
		"unknown message": {InertFields: []string{"temporal.server.api.umpire.v1.Nothing.name"}},
		"not of the IR":   {InertFields: []string{"google.protobuf.Duration.seconds"}},
		"not a full name": {InertFields: []string{"total"}},
		"machine field":   {Attachments: []Attachment{{Machine: "job", Field: "on", Entity: "queue"}}},
		"action field":    {Attachments: []Attachment{{Action: "job.submit", Field: "entity", Entity: "queue"}}},
		"both":            {Attachments: []Attachment{{Machine: "job", Action: "job.submit", Field: "entity", Entity: "queue"}}},
		"no entity":       {Attachments: []Attachment{{Machine: "job", Field: "entity"}}},
	} {
		require.Error(t, invalid.check(), name)
	}
}

func TestOriginalLocatedDropsOnlyPositions(t *testing.T) {
	require.Equal(t,
		"path <source>; note <source>: no Query x; at <source>; rows model/ir/a.json",
		string(Located([]byte("path model/temporal/a/Claims.scala; note model: roots a, b: no Query x; at model/lifter/testdata/lifts/Rejects.scala:12:4; rows model/ir/a.json"), "model: roots a, b")))
}

func TestOriginalInventoryIsClosed(t *testing.T) {
	archived := map[string][]byte{
		"ir/a.json": nil, "lifts/b.json": nil, "cases/c.json": nil,
		OriginalRejects: []byte("lift: model/lifter/testdata/lifts/Rejects.scala:3: one\nlift: root x: none\n"),
	}
	require.NoError(t, Delta{}.Inventory(archived, archived))
	moved := maps.Clone(archived)
	moved[OriginalRejects] = []byte("lift: root x: none\nlift: model/lifter/testdata/lifts/Rejects.scala:9: one\nlift: model/lifter/testdata/lifts/Rejects.scala:20: a later refusal\n")
	moved["lifts/later.json"] = nil
	require.NoError(t, Delta{}.Inventory(archived, moved), "moved refusals and later fixtures")
	for name, change := range map[string]func(map[string][]byte){
		"refusal gone": func(m map[string][]byte) { m[OriginalRejects] = []byte("lift: root x: none\n") },
		"refusal reworded": func(m map[string][]byte) {
			m[OriginalRejects] = []byte("lift: model/lifter/testdata/lifts/Rejects.scala:3: two\nlift: root x: none\n")
		},
		"IR gone":      func(m map[string][]byte) { delete(m, "ir/a.json") },
		"IR added":     func(m map[string][]byte) { m["ir/other.json"] = nil },
		"Case added":   func(m map[string][]byte) { m["cases/other.json"] = nil },
		"fixture gone": func(m map[string][]byte) { delete(m, "lifts/b.json") },
	} {
		changed := maps.Clone(archived)
		change(changed)
		require.Error(t, Delta{}.Inventory(archived, changed), name)
	}
}

func TestOriginalDerivedOutputsAreClosed(t *testing.T) {
	parts := map[string][]any{"a": {"one", []byte(`{"at":"model/a/B.scala:3"}`)}, "b": {map[string]int{"two": 2}}}
	derive := func(parts map[string][]any, key string, s *Stream) error {
		for i, v := range parts[key] {
			if err := s.Add(strings.Repeat("p", i+1), v, "model: roots"); err != nil {
				return err
			}
		}
		return nil
	}
	digests := func(parts map[string][]any) Derived {
		d := Derived{}
		for key := range parts {
			s := &Stream{}
			require.NoError(t, derive(parts, key, s))
			d[key] = s.Digest()
		}
		return d
	}
	dir := t.TempDir()
	require.NoError(t, WriteDerived(dir, "derived.json", digests(parts)))
	require.Error(t, WriteDerived(dir, "derived.json", digests(parts)), "never replaced")
	encoded, err := os.ReadFile(filepath.Join(dir, "derived.json"))
	require.NoError(t, err)
	var archived Derived
	require.NoError(t, json.Unmarshal(encoded, &archived))
	require.NoError(t, CompareDerived(archived, digests(parts), nil))

	moved := maps.Clone(parts)
	moved["a"] = []any{"one", []byte(`{"at":"model/c/D.scala:9"}`)}
	require.NoError(t, CompareDerived(archived, digests(moved), nil), "positions are projected")
	for name, change := range map[string]func(map[string][]any){
		"changed":   func(m map[string][]any) { m["a"] = []any{"one, changed", m["a"][1]} },
		"reordered": func(m map[string][]any) { m["a"] = []any{m["a"][1], m["a"][0]} },
		"split":     func(m map[string][]any) { m["a"] = []any{"on", "e" + string(m["a"][1].([]byte))} },
		"missing":   func(m map[string][]any) { delete(m, "b") },
		"unknown":   func(m map[string][]any) { m["c"] = nil },
	} {
		changed := maps.Clone(parts)
		change(changed)
		require.Error(t, CompareDerived(archived, digests(changed), nil), name)
	}
	changed := maps.Clone(parts)
	changed["a"] = []any{"one, changed", parts["a"][1]}
	explain := func(key string) string {
		return Explain(archived[key], func(expected bool, s *Stream) error {
			if expected {
				return derive(parts, key, s)
			}
			return derive(changed, key, s)
		})
	}
	err = CompareDerived(archived, digests(changed), explain)
	require.ErrorContains(t, err, `part p at byte 4`)
	stale := Explain("other", func(_ bool, s *Stream) error { return derive(parts, "a", s) })
	require.Contains(t, stale, "the Go tooling changed")
	require.Contains(t, stale, "every part is the same")
}

// lawJob is a baseline whose Property `finishes` a law's generated claim `job.finishes` renames, with
// the verify Query `finishHolds` it retires and the Query `once` it re-points, and its current
// spelling: the renamed Property, its generated Scenario and Query, and the pure addition
// `job.settles` with its Property, the Functions it holds by, its Scenario and its Query.
func lawJob(t *testing.T) (key string, delta Delta, baseline, current *umpirespb.Model) {
	t.Helper()
	key = "ir/job.json"
	baseline = JobModel("once", "submit", "take", "finish")
	verify := proto.CloneOf(baseline.Queries[0])
	verify.Name, verify.Form = "finishHolds", umpirespb.Query_FORM_VERIFY
	baseline.Queries = append(baseline.Queries, verify)
	delta = Delta{Replacements: []Replacement{
		{Model: key, Machine: "job", Law: "finishes", Verdict: "found", Renames: "finishes", Retires: []string{"finishHolds"}},
		{Model: key, Machine: "job", Law: "settles", Verdict: "verified-within-limits"},
	}}
	require.NoError(t, delta.check())
	current = proto.CloneOf(baseline)
	current.Queries = current.Queries[:1]
	current.Properties[0].Name = "job.finishes"
	current.Queries[0].Property.Name = "job.finishes"
	at := baseline.Scenarios[0].Position
	for _, law := range []string{"finishes", "settles"} {
		name := "job." + law
		if law == "settles" {
			holds := "job.property." + name
			current.Properties = append(current.Properties, &umpirespb.Property{Machine: "job", Name: name, Position: at, Holds: holds, Transition: true})
			current.Functions = append(current.Functions,
				&umpirespb.Function{Name: holds, Body: functionNamed(baseline, "job.finishes").Body},
				&umpirespb.Function{Name: holds + ".never", Body: functionNamed(baseline, "job.finishes").Body})
		}
		current.Scenarios = append(current.Scenarios, &umpirespb.Scenario{Machine: "job", Name: name, Position: at, Start: baseline.Scenarios[0].Start, Free: true})
		current.Queries = append(current.Queries, &umpirespb.Query{Name: name, Position: at, Form: umpirespb.Query_FORM_VERIFY,
			Property: &umpirespb.ClaimRef{Machine: "job", Name: name}, Scenario: &umpirespb.ClaimRef{Machine: "job", Name: name},
			Limits: &umpirespb.Limits{Name: "three", Steps: 3, Actions: 3, Search: 4096}})
	}
	return key, delta, baseline, current
}

func TestOriginalLawReplacementsAreExact(t *testing.T) {
	key, delta, baseline, current := lawJob(t)
	compare := func(d Delta, baseline, current *umpirespb.Model) error {
		applied := Applied{}
		expected, err := d.Expected(key, baseline, applied)
		if err != nil {
			return err
		}
		ungenerated, err := d.Ungenerated(key, current)
		if err != nil {
			return err
		}
		return errors.Join(d.Unapplied(applied), d.MatchOriginal(expected, ungenerated))
	}
	require.NoError(t, compare(delta, baseline, current))
	expected, err := delta.Expected(key, baseline, Applied{})
	require.NoError(t, err)
	require.Equal(t, "finishes", baseline.Properties[0].GetName(), "the baseline is not changed")
	require.Equal(t, []string{"once"}, queryNames(expected))
	require.Equal(t, "job.finishes", expected.Queries[0].GetProperty().GetName(), "a Query that read the renamed Property reads it so")
	ungenerated, err := delta.Ungenerated(key, current)
	require.NoError(t, err)
	require.Equal(t, []string{"once"}, queryNames(ungenerated))
	require.Len(t, ungenerated.Properties, 1, "a renamed Property stays, a generated one goes")
	require.Nil(t, functionNamed(ungenerated, "job.property.job.settles.never"), "with the Functions it holds by")
	require.Len(t, current.Queries, 3, "the current Model is not changed")
	_, err = delta.Expected("ir/other.json", baseline, Applied{})
	require.NoError(t, err, "another file's replacements do not apply")

	rejected := map[string]struct {
		delta   func(*Delta)
		current func(*umpirespb.Model)
	}{
		"unlisted retirement":           {current: func(m *umpirespb.Model) { m.Queries = m.Queries[1:] }},
		"unlisted generated claim":      {delta: func(d *Delta) { d.Replacements = d.Replacements[:1] }},
		"twin without its Scenario":     {current: func(m *umpirespb.Model) { m.Scenarios = m.Scenarios[:2] }},
		"twin reading another Scenario": {current: func(m *umpirespb.Model) { m.Queries[2].Scenario.Name = "once" }},
		"twin reading another Property": {current: func(m *umpirespb.Model) { m.Queries[1].Property.Name = "job.settles" }},
		"generated Property missing":    {current: func(m *umpirespb.Model) { m.Properties = m.Properties[:1] }},
		"retirement matching nothing":   {delta: func(d *Delta) { d.Replacements[0].Retires = append(d.Replacements[0].Retires, "nothing") }},
		"rename matching nothing":       {delta: func(d *Delta) { d.Replacements[0].Renames = "nothing" }},
		"no rename":                     {delta: func(d *Delta) { d.Replacements[0].Renames = "" }},
		"no retirement":                 {delta: func(d *Delta) { d.Replacements[0].Retires = nil }},
		"Query left reading the old name": {current: func(m *umpirespb.Model) {
			m.Queries[0].Property.Name = "finishes"
		}},
		"re-pointed without a rename": {current: func(m *umpirespb.Model) {
			m.Properties = append(m.Properties, &umpirespb.Property{Machine: "job", Name: "other", Holds: "job.finishes"})
			m.Queries[0].Property.Name = "other"
		}},
		"changed limits of a remaining Query": {current: func(m *umpirespb.Model) { m.Queries[0].Limits.Steps++ }},
		"replacement of no baseline Model":    {delta: func(d *Delta) { d.Replacements[1].Model = "ir/other.json" }},
	}
	for name, c := range rejected {
		t.Run(name, func(t *testing.T) {
			d := Delta{Replacements: slices.Clone(delta.Replacements)}
			d.Replacements[0].Retires = slices.Clone(d.Replacements[0].Retires)
			if c.delta != nil {
				c.delta(&d)
			}
			changed := proto.CloneOf(current)
			if c.current != nil {
				c.current(changed)
			}
			require.Error(t, compare(d, baseline, changed))
		})
	}

	orphan := proto.CloneOf(baseline)
	orphan.Queries[0].Property.Name = "nothing"
	_, err = delta.Expected(key, orphan, Applied{})
	require.ErrorContains(t, err, "does not declare", "a Query of the expected Model reads a Property it lacks")
	taken := proto.CloneOf(baseline)
	taken.Properties = append(taken.Properties, &umpirespb.Property{Machine: "job", Name: "job.finishes"})
	_, err = delta.Expected(key, taken, Applied{})
	require.ErrorContains(t, err, "the name the baseline gives another")
}

func queryNames(m *umpirespb.Model) []string {
	var out []string
	for _, q := range m.GetQueries() {
		out = append(out, q.GetName())
	}
	return out
}

func TestOriginalLawReplacementsAreClosed(t *testing.T) {
	valid := Replacement{Model: "ir/a.json", Machine: "m", Law: "law", Verdict: "verified-within-limits", Renames: "old", Retires: []string{"q"}}
	require.Equal(t, "m.law", valid.Generated())
	require.Equal(t, "cases/a-m.law-case.json", NewCase{Model: "ir/a.json", Query: "m.law"}.File())
	with := func(change func(*Replacement)) []Replacement {
		r := valid
		r.Retires = slices.Clone(valid.Retires)
		change(&r)
		return []Replacement{r}
	}
	require.NoError(t, Delta{Replacements: []Replacement{valid}, NewCases: []NewCase{{Model: "ir/a.json", Query: "m.law"}}}.check())
	require.NoError(t, Delta{NewIRFiles: []string{"ir/new.json"}, NewCases: []NewCase{{Model: "ir/new.json", Query: "any"}}}.check(),
		"a new IR file's Cases are all new")
	for name, invalid := range map[string]Delta{
		"dotted law":                 {Replacements: with(func(r *Replacement) { r.Law = "a.law" })},
		"no law":                     {Replacements: with(func(r *Replacement) { r.Law = "" })},
		"dotted machine":             {Replacements: with(func(r *Replacement) { r.Machine = "a.m" })},
		"unknown verdict":            {Replacements: with(func(r *Replacement) { r.Verdict = "holds" })},
		"no verdict":                 {Replacements: with(func(r *Replacement) { r.Verdict = "" })},
		"not an IR file":             {Replacements: with(func(r *Replacement) { r.Model = "lifts/a.json" })},
		"a sidecar":                  {Replacements: with(func(r *Replacement) { r.Model = "ir/a.laws.json" })},
		"renamed to itself":          {Replacements: with(func(r *Replacement) { r.Renames = "m.law" })},
		"retired twice":              {Replacements: with(func(r *Replacement) { r.Retires = []string{"q", "q"} })},
		"retired unnamed":            {Replacements: with(func(r *Replacement) { r.Retires = []string{""} })},
		"retires a twin":             {Replacements: with(func(r *Replacement) { r.Retires = []string{"m.law"} })},
		"listed twice":               {Replacements: []Replacement{valid, {Model: "ir/a.json", Machine: "m", Law: "law", Verdict: "found"}}},
		"of a new IR file":           {Replacements: []Replacement{valid}, NewIRFiles: []string{"ir/a.json"}},
		"Case of no generated Query": {Replacements: []Replacement{valid}, NewCases: []NewCase{{Model: "ir/a.json", Query: "m.other"}}},
		"Case of another file":       {Replacements: []Replacement{valid}, NewCases: []NewCase{{Model: "ir/b.json", Query: "m.law"}}},
		"Case listed twice": {Replacements: []Replacement{valid},
			NewCases: []NewCase{{Model: "ir/a.json", Query: "m.law"}, {Model: "ir/a.json", Query: "m.law"}}},
		"new IR file twice":        {NewIRFiles: []string{"ir/new.json", "ir/new.json"}},
		"new IR file sidecar":      {NewIRFiles: []string{"ir/new.laws.json"}},
		"new IR file not under ir": {NewIRFiles: []string{"cases/new.json"}},
	} {
		require.Error(t, invalid.check(), name)
	}
}

func TestOriginalInventoryAdmitsOnlyTheListedGeneratedFiles(t *testing.T) {
	archived := map[string][]byte{"ir/a.json": nil, "ir/b.json": nil, "cases/a-q-case.json": nil, OriginalRejects: nil}
	delta := Delta{
		Replacements: []Replacement{{Model: "ir/a.json", Machine: "m", Law: "law", Verdict: "found"}, {Model: "ir/a.json", Machine: "m", Law: "other", Verdict: "verified-within-limits"}},
		NewCases:     []NewCase{{Model: "ir/a.json", Query: "m.law"}},
		NewIRFiles:   []string{"ir/new.json"},
	}
	require.NoError(t, delta.check())
	current := maps.Clone(archived)
	current["ir/a.laws.json"] = []byte(`{"claims": [{"name": "m.other", "law": "other"}, {"name": "m.law", "law": "law"}], "waivers": []}`)
	current["cases/a-m.law-case.json"] = nil
	current["ir/new.json"] = nil
	require.NoError(t, delta.Inventory(archived, current))
	newSidecar := maps.Clone(current)
	newSidecar["ir/new.laws.json"] = []byte(`{"claims": [{"name": "x.y"}]}`)
	require.NoError(t, delta.Inventory(archived, newSidecar), "a new IR file's sidecar is new with it")
	for name, change := range map[string]func(map[string][]byte){
		"sidecar missing": func(m map[string][]byte) { delete(m, "ir/a.laws.json") },
		"sidecar lists another": func(m map[string][]byte) {
			m["ir/a.laws.json"] = []byte(`{"claims": [{"name": "m.law"}, {"name": "m.third"}]}`)
		},
		"sidecar lists fewer": func(m map[string][]byte) { m["ir/a.laws.json"] = []byte(`{"claims": [{"name": "m.law"}]}`) },
		"sidecar unreadable":  func(m map[string][]byte) { m["ir/a.laws.json"] = []byte(`[]`) },
		"unlisted sidecar":    func(m map[string][]byte) { m["ir/b.laws.json"] = []byte(`{"claims": []}`) },
		"unlisted Case":       func(m map[string][]byte) { m["cases/a-m.other-case.json"] = nil },
		"listed Case missing": func(m map[string][]byte) { delete(m, "cases/a-m.law-case.json") },
		"new IR file missing": func(m map[string][]byte) { delete(m, "ir/new.json") },
		"unlisted IR file":    func(m map[string][]byte) { m["ir/other.json"] = nil },
		"archived file gone":  func(m map[string][]byte) { delete(m, "cases/a-q-case.json") },
	} {
		changed := maps.Clone(current)
		change(changed)
		require.Error(t, delta.Inventory(archived, changed), name)
	}
	archivedNew := maps.Clone(archived)
	archivedNew["ir/new.json"] = nil
	require.ErrorContains(t, delta.Inventory(archivedNew, current), "listed as new and archived")
}
