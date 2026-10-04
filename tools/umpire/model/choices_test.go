package model

// Named choices (model/SEMANTICS.md): each result of a step may carry the name of the alternative it
// is, written on its step record's construct. The names are inert: a row's results report them, and
// nothing else a Model derives changes with them. The fixture is the lifted admission specimen, whose
// committed admission `admitted` either consumes the message or, besides that, retains it for another
// delivery: two results of one row.

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/golden"
	"google.golang.org/protobuf/proto"
)

const admissionPackage = "fixture.specimens.admission.Admission$package$."

// stepAt is the step record construct of an item of a list expression.
func stepAt(t *testing.T, list *umpirespb.Expr, item int) *umpirespb.Construct {
	t.Helper()
	c := list.GetList().GetItems()[item].GetConstruct()
	require.Equal(t, StepType, c.GetType())
	return c
}

// namedAdmission is the admission specimen with its committed admission's alternatives named: a
// redelivered message is consumed, a first delivery consumed or retained. A dispatch, one result of
// its own, is named too; the stale-message rejection of admitCurrent is left unnamed.
func namedAdmission(t *testing.T) (plain, named *umpirespb.Model) {
	t.Helper()
	plain = lifted(t, "admission")
	named = proto.CloneOf(plain)
	admitted := functionNamed(named, admissionPackage+"admitted").GetBody().GetLet().GetBody().GetIf()
	stepAt(t, admitted.GetThen(), 0).Choice = "consumed"
	stepAt(t, admitted.GetElse(), 0).Choice = "consumed"
	stepAt(t, admitted.GetElse(), 1).Choice = "retained"
	stepAt(t, functionNamed(named, admissionPackage+"dispatchStep").GetBody().GetIf().GetElse(), 0).Choice = "enqueued"
	return plain, named
}

func choicesOf(row Row) []string {
	out := []string{}
	for _, res := range row.Results {
		out = append(out, res.Choice)
	}
	return out
}

// TestNamedChoicesAreReportedAndInert names the alternatives of the admission specimen's steps. Each
// result reports the name of the step record it was read from, in the order of the results, and an
// unnamed one reports none. Every table, as encoded, every Definition ID and Behavior Fingerprint, and
// every output the original baseline derives, receipts and Query answers included, is the unnamed
// Model's.
func TestNamedChoicesAreReportedAndInert(t *testing.T) {
	plain, named := namedAdmission(t)
	require.NoError(t, Validate(named))
	before, err := Build(plain)
	require.NoError(t, err)
	after, err := Build(named)
	require.NoError(t, err)
	require.Len(t, after, len(before))
	seen := map[string]int{}
	for name, mm := range after {
		was := before[name].Table
		is := mm.Table
		wasJSON, err := json.Marshal(was)
		require.NoError(t, err)
		isJSON, err := json.Marshal(is)
		require.NoError(t, err)
		require.Equal(t, golden.Digest(wasJSON), golden.Digest(isJSON), "%s: the table encodes to other bytes", name)
		require.Equal(t, was.IDs(), is.IDs(), name)
		require.Equal(t, was.TargetFingerprint(), is.TargetFingerprint(), name)
		require.Len(t, is.Rows, len(was.Rows), name)
		for i, row := range is.Rows {
			unnamed := choicesOf(was.Rows[i])
			require.Equal(t, make([]string, len(unnamed)), unnamed, "the unnamed Model names no result")
			want := make([]string, len(row.Results))
			switch {
			case name == "activityProduct":
			case row.Action == "dispatch":
				want = []string{"enqueued"}
			case row.Action == "attemptStart" && len(row.Results) == 2:
				want = []string{"consumed", "retained"}
			case row.Action == "attemptStart" && slices.Contains(row.Results[0].Facts, "admissionRejected"):
				require.Equal(t, "currentAdmission", name, "only admitCurrent rejects a stale message")
				want = []string{""}
			case row.Action == "attemptStart":
				want = []string{"consumed"}
			default:
			}
			require.Equal(t, want, choicesOf(row), "%s %s", name, row.Key)
			seen[fmt.Sprintf("%s %s %d %s", name, row.Action, len(row.Results), choicesOf(row)[0])]++
		}
	}
	for _, kind := range []string{
		"currentAdmission dispatch 1 enqueued", "staleAdmission dispatch 1 enqueued",
		"currentAdmission attemptStart 2 consumed", "staleAdmission attemptStart 2 consumed",
		"currentAdmission attemptStart 1 consumed", "staleAdmission attemptStart 1 consumed",
		"currentAdmission attemptStart 1 ",
	} {
		require.Positive(t, seen[kind], "no row of the kind %q", kind)
	}
	// What the original baseline compares of a Model: its tables, refinements, Property rows and
	// receipts, Query answers among them; its Definition IDs, canonical forms and fingerprints; and its
	// Properties read through each refinement.
	wasMeaning, err := json.Marshal(migrationMeaning(t, plain))
	require.NoError(t, err)
	isMeaning, err := json.Marshal(migrationMeaning(t, named))
	require.NoError(t, err)
	require.Equal(t, string(wasMeaning), string(isMeaning))
	require.Equal(t, originalDigests(t, map[string]*umpirespb.Model{"admission": plain}, originalOutputs...),
		originalDigests(t, map[string]*umpirespb.Model{"admission": named}, originalOutputs...))
}

// TestAChoiceNamesOnlyAStep refuses a name on a construct of any type but a step record, at the
// construct's position, and admits one on a step record wherever it is built.
func TestAChoiceNamesOnlyAStep(t *testing.T) {
	_, named := namedAdmission(t)
	require.NoError(t, Validate(named))
	start := named.GetMachines()[1].GetStarts()[0]
	c := start.GetConstruct()
	require.Equal(t, "fixture.specimens.admission.AdmissionState", c.GetType())
	c.Choice = "initial"
	require.EqualError(t, Validate(named), where(start.GetPosition())+
		": fixture.specimens.admission.AdmissionState names the choice initial, which only a step record can")
}

// TestTwoResultsOfARowCannotShareAName refuses a row two of whose results have the same name, at the
// position of the class's step binding, naming the row and the name. Unnamed results may be many.
func TestTwoResultsOfARowCannotShareAName(t *testing.T) {
	plain, named := namedAdmission(t)
	admitted := functionNamed(named, admissionPackage+"admitted").GetBody().GetLet().GetBody().GetIf()
	stepAt(t, admitted.GetElse(), 1).Choice = "consumed"
	require.NoError(t, Validate(named), "a row's results are only known once it is evaluated")
	built, err := Build(plain)
	require.NoError(t, err)
	current := built["currentAdmission"].Table
	first := slices.IndexFunc(current.Rows, func(r Row) bool { return r.Action == "attemptStart" && len(r.Results) == 2 })
	require.GreaterOrEqual(t, first, 0)
	binding := named.GetMachines()[1].GetSteps()[2]
	require.Equal(t, admissionPackage+"admitCurrent", binding.GetFunction())
	_, err = Build(named)
	require.EqualError(t, err, where(binding.GetPosition())+": currentAdmission: row "+current.Rows[first].Key+" has two results named consumed")

	stepAt(t, admitted.GetElse(), 0).Choice = ""
	stepAt(t, admitted.GetElse(), 1).Choice = ""
	_, err = Build(named)
	require.NoError(t, err, "two unnamed results")
}

// TestARedeliveryIsUnnamed names what hearing a note on the wire, a channel that delivers a message
// once more, does. The receiver's result is named; the channel's redelivery of it is the channel's,
// unnamed, so the row does not hold the name twice.
func TestARedeliveryIsUnnamed(t *testing.T) {
	plain := lifted(t, "channels")
	named := proto.CloneOf(plain)
	stepAt(t, functionNamed(named, "fixture.channels.Channels$package$.hear").GetBody().GetIf().GetThen(), 0).Choice = "heard"
	require.NoError(t, Validate(named))
	before, err := Build(plain)
	require.NoError(t, err)
	after, err := Build(named)
	require.NoError(t, err)
	was, is := before["relay"].Table, after["relay"].Table
	wasJSON, err := json.Marshal(was)
	require.NoError(t, err)
	isJSON, err := json.Marshal(is)
	require.NoError(t, err)
	require.Equal(t, golden.Digest(wasJSON), golden.Digest(isJSON), "the table encodes to other bytes")
	redelivered := 0
	for _, row := range is.Rows {
		if !slices.ContainsFunc(row.Results, func(r Result) bool { return r.Choice != "" }) {
			continue
		}
		for _, res := range row.Results {
			if res.Because == "the channel delivers the message again" {
				require.Empty(t, res.Choice, row.Key)
				redelivered++
			} else {
				require.Equal(t, "heard", res.Choice, row.Key)
			}
		}
	}
	require.Positive(t, redelivered)
}

// TestAlternativesAreNoClassesOfTheirOwn is the static side of a Query's combination total: the work
// counted before a machine is listed is its states times its classes, and a named choice's
// alternatives are results of one class. With one alternative of the committed admission instead of
// two, the state catalog, the classes, the counted evaluations and the rows are the same; only the
// results of the rows the second alternative was in are fewer.
//
// fn-112.11 adds Query.total, the author's static combination total; when it merges, this is where
// the assertion that the alternatives do not enter it belongs.
func TestAlternativesAreNoClassesOfTheirOwn(t *testing.T) {
	_, two := namedAdmission(t)
	one := proto.CloneOf(two)
	admitted := functionNamed(one, admissionPackage+"admitted").GetBody().GetLet().GetBody().GetIf()
	admitted.GetElse().GetList().Items = admitted.GetElse().GetList().GetItems()[:1]
	actions := map[string]*umpirespb.Action{}
	for _, a := range two.GetActions() {
		actions[a.GetId()] = a
	}
	counts := func(m *umpirespb.Model, decl *umpirespb.Machine) (states, classes count, needed int64) {
		in := NewInterpreter(m)
		states, err := in.size(named(decl.GetStateType()))
		require.NoError(t, err)
		classes, err = in.classCount(decl, actions)
		require.NoError(t, err)
		in.ceilings.Evaluations = 0
		var limit *LimitError
		require.ErrorAs(t, in.preflight(decl, actions), &limit)
		return states, classes, limit.Needed
	}
	withTwo, err := Build(two)
	require.NoError(t, err)
	withOne, err := Build(one)
	require.NoError(t, err)
	fewer := 0
	for i, decl := range two.GetMachines() {
		s2, c2, n2 := counts(two, decl)
		s1, c1, n1 := counts(one, one.GetMachines()[i])
		require.Equal(t, []any{s2, c2, n2}, []any{s1, c1, n1}, decl.GetName())
		require.Positive(t, n2)
		t2, t1 := withTwo[decl.GetName()].Table, withOne[decl.GetName()].Table
		require.Equal(t, t2.States, t1.States)
		require.Equal(t, t2.Actions, t1.Actions)
		require.Len(t, t1.Rows, len(t2.Rows))
		for j, row := range t2.Rows {
			require.Equal(t, row.Key, t1.Rows[j].Key)
			if choicesOf(row)[0] == "consumed" && len(row.Results) == 2 {
				require.Equal(t, []string{"consumed"}, choicesOf(t1.Rows[j]))
				require.Equal(t, row.Results[0].State, t1.Rows[j].Results[0].State)
				fewer++
			} else {
				require.Len(t, t1.Rows[j].Results, len(row.Results))
			}
		}
	}
	require.Positive(t, fewer)
}
