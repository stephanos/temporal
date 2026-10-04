package model

import (
	"errors"
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
	root      string
	delta     golden.Delta
	baselines map[string]*umpirespb.Model
	// generated are the current Models as produced; current are them without the generated claims
	// the delta lists (Ungenerated), which must equal expected, the baselines with the delta applied.
	generated, current map[string]*umpirespb.Model
	expected           map[string]*umpirespb.Model
}

func readOriginal(t *testing.T) originalInputs {
	t.Helper()
	in := originalInputs{expected: map[string]*umpirespb.Model{}, current: map[string]*umpirespb.Model{}, generated: map[string]*umpirespb.Model{}}
	var err error
	in.root, err = golden.Root()
	require.NoError(t, err)
	in.delta, err = golden.OriginalDelta()
	require.NoError(t, err)
	archived, err := golden.OriginalArchive(in.root)
	require.NoError(t, err)
	// A reduced fixture is a new Model, or retired, compared with no baseline (Delta.Reduced).
	in.baselines, err = golden.OriginalModels(in.delta.Compared(archived))
	require.NoError(t, err)
	files, err := golden.OriginalCurrent(in.root)
	require.NoError(t, err)
	current, err := golden.OriginalModels(files)
	require.NoError(t, err)
	applied := golden.Applied{}
	for key, baseline := range in.baselines {
		in.expected[key], err = in.delta.Expected(key, baseline, applied)
		require.NoError(t, err, key)
		require.Contains(t, current, key)
		in.generated[key] = current[key]
		in.current[key], err = in.delta.Ungenerated(key, current[key])
		require.NoError(t, err, key)
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
// fixture to what it derived from the original baseline. When the delta attaches entities or
// replaces claims by the ones laws generate, the expected outputs are derived again from the
// baseline with exactly that delta, so every fingerprint that reads an entity, and every row, ID,
// canonical form, fingerprint and receipt of a renamed Property and of the Queries that read it, is
// still compared. The current Model is read without the generated claims the delta lists; what those
// answer is held to the verdicts it records (TestOriginalLawVerdicts).
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
	expected = in.delta.ComparedOutputs(expected)
	if in.delta.Rederives() {
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
	expected, err := delta.Expected(key, in.baselines[key], golden.Applied{})
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

// originalVerdicts checks that each law replacement's generated Query answers, on the current Model
// as produced, the verdict the delta records, and that each Query it retires answered the same on
// the baseline: a generated claim replaces a Query only with the same answer.
func originalVerdicts(delta golden.Delta, generated, baselines map[string]*umpirespb.Model) error {
	verdicts := func(m *umpirespb.Model) map[ClaimKey]ReceiptKind {
		out := map[ClaimKey]ReceiptKind{}
		for _, r := range Check(m, DefaultScope).Receipts {
			if r.Subject == QuerySubject {
				out[ClaimKey{Owner: r.Key.Owner, Name: r.Key.Name}] = r.Kind
			}
		}
		return out
	}
	current, baseline := map[string]map[ClaimKey]ReceiptKind{}, map[string]map[ClaimKey]ReceiptKind{}
	var errs []error
	for _, r := range delta.Replacements {
		if current[r.Model] == nil {
			current[r.Model], baseline[r.Model] = verdicts(generated[r.Model]), verdicts(baselines[r.Model])
		}
		// A generated Query reads the Scenario of its machine, which owns its receipt.
		if got := current[r.Model][ClaimKey{Owner: r.Machine, Name: r.Generated()}]; got != ReceiptKind(r.Verdict) {
			errs = append(errs, fmt.Errorf("%s: generated Query %s answers %q, and the delta records %q", r.Model, r.Generated(), got, r.Verdict))
		}
		for _, retired := range r.Retires {
			var answered []ReceiptKind
			for key, kind := range baseline[r.Model] {
				if key.Name == retired {
					answered = append(answered, kind)
				}
			}
			if len(answered) != 1 || answered[0] != ReceiptKind(r.Verdict) {
				errs = append(errs, fmt.Errorf("%s: retired Query %s answered %v, and its replacement %s %q", r.Model, retired, answered, r.Generated(), r.Verdict))
			}
		}
	}
	return errors.Join(errs...)
}

// TestOriginalLawVerdicts holds every generated claim the delta lists to the verdict it records, and
// to the verdict of each Query it retires. The comparison with the baseline reads the current Model
// without the generated claims, so this is what holds their answers.
func TestOriginalLawVerdicts(t *testing.T) {
	in := readOriginal(t)
	require.NotEmpty(t, in.delta.Replacements)
	require.NoError(t, originalVerdicts(in.delta, in.generated, in.baselines))
	for name, change := range map[string]func(*golden.Replacement){
		"wrong verdict":        func(r *golden.Replacement) { r.Verdict = string(Counterexample) },
		"retired another":      func(r *golden.Replacement) { r.Retires = []string{"cancelRequest"} },
		"retired of no answer": func(r *golden.Replacement) { r.Retires = []string{"nothing"} },
		"another machine":      func(r *golden.Replacement) { r.Machine = "activityProtocol" },
	} {
		t.Run(name, func(t *testing.T) {
			delta := in.delta
			i := slices.IndexFunc(delta.Replacements, func(r golden.Replacement) bool {
				return r.Model == "ir/activity.json" && len(r.Retires) > 0
			})
			require.GreaterOrEqual(t, i, 0)
			r := delta.Replacements[i]
			r.Retires = slices.Clone(r.Retires)
			change(&r)
			delta.Replacements = []golden.Replacement{r}
			require.Error(t, originalVerdicts(delta, in.generated, in.baselines))
		})
	}
}

// TestOriginalVerdictsAreReceiptKinds holds the verdicts the delta may record to the receipt kinds.
func TestOriginalVerdictsAreReceiptKinds(t *testing.T) {
	kinds := []ReceiptKind{AdmissionError, DeclarationError, ResourceLimit, LimitReached, Unresolved, RefinementRejected,
		Counterexample, Verified, Found, NotFound, Incomplete, Unsupported, ReplayFailed}
	var want []string
	for _, kind := range kinds {
		want = append(want, string(kind))
	}
	require.ElementsMatch(t, want, golden.ReceiptKinds)
}

// TestOriginalBaselineDerivesRenamedClaimsFromTheBaseline renames a Property as a law replacement
// does: the rename enters Definition IDs and the receipts of the Queries that read it, so they are
// compared with the baseline's derived again with the rename, never waived, and a rename the delta
// does not record is refused by the IR comparison and the derived outputs alike.
func TestOriginalBaselineDerivesRenamedClaimsFromTheBaseline(t *testing.T) {
	const key = "ir/activity.json"
	in := readOriginal(t)
	archived, err := golden.ReadDerived(in.root, originalModelOutputs)
	require.NoError(t, err)
	digest := func(output string, m *umpirespb.Model) string {
		return originalDigests(t, map[string]*umpirespb.Model{key: m}, output)[output+"/"+key]
	}
	for _, output := range []string{"semantics", "declarations"} {
		frozen := archived[output+"/"+key]
		require.Equal(t, frozen, digest(output, in.baselines[key]), output)
		rederived := digest(output, in.expected[key])
		require.NotEqual(t, frozen, rederived, "the rename enters %s", output)
		require.Equal(t, rederived, digest(output, in.current[key]), output)
	}
	unrecorded := proto.CloneOf(in.current[key])
	for _, p := range unrecorded.GetProperties() {
		if p.GetName() == "completes" {
			p.Name = "activityProtocol.completes"
		}
	}
	for _, q := range unrecorded.GetQueries() {
		if q.GetProperty().GetName() == "completes" {
			q.Property.Name = "activityProtocol.completes"
		}
	}
	require.Error(t, in.delta.MatchOriginal(in.expected[key], unrecorded))
	require.NotEqual(t, digest("declarations", in.expected[key]), digest("declarations", unrecorded))
	repointed := proto.CloneOf(in.current[key])
	for _, q := range repointed.GetQueries() {
		if q.GetName() == "completion" {
			q.Property.Name = "retryCompletes"
		}
	}
	require.Error(t, in.delta.MatchOriginal(in.expected[key], repointed))
	require.NotEqual(t, digest("semantics", in.expected[key]), digest("semantics", repointed))
}
