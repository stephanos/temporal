package model

import (
	"flag"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/golden"
	"google.golang.org/protobuf/proto"
)

var captureOriginal = flag.String("capture-original", "", "write the original baseline's derived Model outputs into this archive directory")

// The outputs the reader derives from each Model of the original baseline: tables with their finite
// catalogs, refinements, Property rows and Query receipts; Definition IDs, canonical forms and
// fingerprints; and every Property read through each refinement.
var originalOutputs = []string{"semantics", "declarations", "refined-properties"}

const originalModelOutputs = "model.json"

// originalDerive streams one output of a bound Model part by part, so even nexus-close's hundreds of
// megabytes of Property rows are digested without being held.
func originalDerive(t *testing.T, b *binding, output string, s *golden.Stream) error {
	t.Helper()
	var err error
	part := 0
	add := func(name string, v any) {
		if err == nil {
			err = s.Add(fmt.Sprintf("%d %s", part, name), v, b.model.GetSource())
			part++
		}
	}
	switch output {
	case "semantics":
		receipts := migrationMeaningEach(b,
			func(x migrationSubject) { add("subject "+x.Name, x) },
			func(x migrationProperty) { add("property "+x.Owner+"."+x.Name, x) })
		for _, r := range receipts {
			add("receipt "+r.Key.Owner+"."+r.Key.Name, r)
		}
	case "declarations":
		for _, d := range migrationDefinitionsOf(t, b) {
			add(d.Kind+" "+d.Owner+"."+d.Name, d)
		}
	case "refined-properties":
		for _, r := range migrationRefinedPropertiesOf(b) {
			add(r.Machine+" "+r.Owner+"."+r.Name+" "+r.Row, r)
		}
	default:
		return fmt.Errorf("unknown output %s", output)
	}
	return err
}

func originalDigests(t *testing.T, models map[string]*umpirespb.Model, outputs ...string) golden.Derived {
	t.Helper()
	d := golden.Derived{}
	for _, key := range slices.Sorted(maps.Keys(models)) {
		b := migrationBinding(t, models[key])
		for _, output := range outputs {
			s := &golden.Stream{}
			require.NoError(t, originalDerive(t, b, output, s), key)
			d[output+"/"+key] = s.Digest()
		}
	}
	return d
}

type originalInputs struct {
	root               string
	delta              golden.Delta
	baselines, current map[string]*umpirespb.Model
	expected           map[string]*umpirespb.Model
}

func readOriginal(t *testing.T) originalInputs {
	t.Helper()
	in := originalInputs{expected: map[string]*umpirespb.Model{}, current: map[string]*umpirespb.Model{}}
	var err error
	in.root, err = golden.Root()
	require.NoError(t, err)
	in.delta, err = golden.OriginalDelta()
	require.NoError(t, err)
	archived, err := golden.OriginalArchive(in.root)
	require.NoError(t, err)
	in.baselines, err = golden.OriginalModels(archived)
	require.NoError(t, err)
	files, err := golden.OriginalCurrent(in.root)
	require.NoError(t, err)
	current, err := golden.OriginalModels(files)
	require.NoError(t, err)
	applied := map[int]bool{}
	for key, baseline := range in.baselines {
		in.expected[key], err = in.delta.Expected(baseline, applied)
		require.NoError(t, err, key)
		require.Contains(t, current, key)
		in.current[key] = current[key]
	}
	require.NoError(t, in.delta.Unapplied(applied))
	return in
}

func TestCaptureOriginalBaseline(t *testing.T) {
	if *captureOriginal == "" {
		t.Skip("explicit -capture-original=<archive directory> required")
	}
	in := readOriginal(t)
	require.NoError(t, golden.WriteDerived(*captureOriginal, originalModelOutputs, originalDigests(t, in.baselines, originalOutputs...)))
}

// TestOriginalBaselineModel holds what the reader derives from every current IR Model and lifter
// fixture to what it derived from the original baseline. When the delta attaches entities, the
// expected outputs are derived again from the baseline with exactly those attachments, so every
// fingerprint that reads an entity is still compared.
func TestOriginalBaselineModel(t *testing.T) {
	if *captureOriginal != "" {
		t.Skip("capture is separate from verification")
	}
	in := readOriginal(t)
	for _, key := range slices.Sorted(maps.Keys(in.expected)) {
		require.NoError(t, in.delta.MatchOriginal(in.expected[key], in.current[key]), key)
	}
	expected, err := golden.ReadDerived(in.root, originalModelOutputs)
	require.NoError(t, err)
	if len(in.delta.Attachments) > 0 {
		require.NoError(t, golden.CompareDerived(expected, originalDigests(t, in.baselines, originalOutputs...), nil),
			"the archive derives as it was frozen")
		expected = originalDigests(t, in.expected, originalOutputs...)
	}
	explain := func(key string) string {
		output, model, _ := strings.Cut(key, "/")
		return golden.Explain(expected[key], func(original bool, s *golden.Stream) error {
			m := in.current[model]
			if original {
				m = in.expected[model]
			}
			return originalDerive(t, migrationBinding(t, m), output, s)
		})
	}
	require.NoError(t, golden.CompareDerived(expected, originalDigests(t, in.current, originalOutputs...), explain))
}

// TestOriginalBaselineDerivesEntityFingerprintsFromTheBaseline attaches the matching queue to an
// entity, as fn-112's task-queue extraction will: the attachment changes Scenario fingerprints, so
// they are compared with the baseline's derived again with the attachment, never waived.
func TestOriginalBaselineDerivesEntityFingerprintsFromTheBaseline(t *testing.T) {
	const key = "ir/activity-system.json"
	in := readOriginal(t)
	archived, err := golden.ReadDerived(in.root, originalModelOutputs)
	require.NoError(t, err)
	delta := golden.Delta{Attachments: []golden.Attachment{{Machine: "matchingQueue", Field: "entity", Entity: "taskQueue"}}}
	expected, err := delta.Expected(in.baselines[key], map[int]bool{})
	require.NoError(t, err)
	attached := proto.CloneOf(expected)
	attached.Source = "model: moved roots"
	attached.Machines[0].Position.Line += 40
	require.NoError(t, delta.MatchOriginal(expected, attached))
	require.Error(t, golden.Delta{}.MatchOriginal(in.baselines[key], attached), "an attachment the delta does not record")
	other := proto.CloneOf(attached)
	for _, m := range other.GetMachines() {
		if m.GetName() == "matchingQueue" {
			m.Entity = "worker"
		}
	}
	require.Error(t, delta.MatchOriginal(expected, other))

	declarations := func(m *umpirespb.Model) string {
		return originalDigests(t, map[string]*umpirespb.Model{key: m}, "declarations")["declarations/"+key]
	}
	frozen := archived["declarations/"+key]
	require.Equal(t, frozen, declarations(in.baselines[key]))
	rederived := declarations(expected)
	require.NotEqual(t, frozen, rederived, "the attachment enters fingerprints")
	require.Equal(t, rederived, declarations(attached))
	require.NotEqual(t, rederived, declarations(other))
}

// TestOriginalBaselineRejectsAChangedAnswer changes one Query's limits: the IR comparison and the
// derived receipts each refuse it.
func TestOriginalBaselineRejectsAChangedAnswer(t *testing.T) {
	const key = "ir/activity.json"
	in := readOriginal(t)
	archived, err := golden.ReadDerived(in.root, originalModelOutputs)
	require.NoError(t, err)
	changed := proto.CloneOf(in.baselines[key])
	changed.Queries[0].Limits.Steps--
	require.Error(t, in.delta.MatchOriginal(in.baselines[key], changed))
	got := originalDigests(t, map[string]*umpirespb.Model{key: changed}, "semantics")
	want := golden.Derived{"semantics/" + key: archived["semantics/"+key]}
	err = golden.CompareDerived(want, got, func(string) string {
		return golden.Explain(want["semantics/"+key], func(original bool, s *golden.Stream) error {
			m := changed
			if original {
				m = in.baselines[key]
			}
			return originalDerive(t, migrationBinding(t, m), "semantics", s)
		})
	})
	require.ErrorContains(t, err, "receipt")
}

// TestOriginalBaselineRejectsAFlippedGuard flips the guard of one step Function. The IR comparison
// no longer reads Function bodies, so it admits the flip; the derived semantics refuse it. A guard
// negated twice, in a renamed Function, means the same and derives the same.
func TestOriginalBaselineRejectsAFlippedGuard(t *testing.T) {
	const key = "ir/activity.json"
	in := readOriginal(t)
	archived, err := golden.ReadDerived(in.root, originalModelOutputs)
	require.NoError(t, err)
	want := golden.Derived{"semantics/" + key: archived["semantics/"+key]}
	compare := func(m *umpirespb.Model) error {
		got := originalDigests(t, map[string]*umpirespb.Model{key: m}, "semantics")
		return golden.CompareDerived(want, got, func(string) string {
			return golden.Explain(want["semantics/"+key], func(original bool, s *golden.Stream) error {
				derived := m
				if original {
					derived = in.baselines[key]
				}
				return originalDerive(t, migrationBinding(t, derived), "semantics", s)
			})
		})
	}
	// guarded is the first step Function of the Model whose body is a guard.
	guarded := func(m *umpirespb.Model) (*umpirespb.StepBinding, *umpirespb.Function) {
		for _, machine := range m.GetMachines() {
			for _, step := range machine.GetSteps() {
				for _, f := range m.GetFunctions() {
					if f.GetName() == step.GetFunction() && f.GetBody().GetIf() != nil {
						return step, f
					}
				}
			}
		}
		require.FailNow(t, "no step Function is a guard")
		return nil, nil
	}

	flipped := proto.CloneOf(in.baselines[key])
	_, f := guarded(flipped)
	guard := f.GetBody().GetIf()
	guard.Then, guard.Else = guard.Else, guard.Then
	require.NoError(t, in.delta.MatchOriginal(in.baselines[key], flipped))
	require.ErrorContains(t, compare(flipped), "derived output semantics/"+key+" differs from the original baseline: part 0 subject")

	restructured := proto.CloneOf(in.baselines[key])
	step, f := guarded(restructured)
	not := func(e *umpirespb.Expr) *umpirespb.Expr {
		return &umpirespb.Expr{Position: e.GetPosition(), Kind: &umpirespb.Expr_Unary{Unary: &umpirespb.Unary{Op: umpirespb.Unary_OP_NOT, Operand: e}}}
	}
	f.GetBody().GetIf().Condition = not(not(f.GetBody().GetIf().GetCondition()))
	f.Name += "Restructured"
	step.Function = f.GetName()
	require.NoError(t, in.delta.MatchOriginal(in.baselines[key], restructured))
	require.NoError(t, compare(restructured))
}
