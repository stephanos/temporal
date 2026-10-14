package ir

// Named choices (model/SEMANTICS.md): each result of a step may carry the name of the alternative it
// is, written on its step record's construct. The names are inert: a row's results report them, and
// nothing else a Model derives changes with them. The fixture is the lifted admission specimen, whose
// committed admission `admitted` either consumes the message or, besides that, retains it for another
// delivery: two results of one row.

import (
	"crypto/sha256"
	"encoding/json"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
	"google.golang.org/protobuf/proto"
)

const admissionPackage = "fixture.specimens.admission.Admission$package$."

// stepAt is the step record construct of an item of a list expression.
func stepAt(t *testing.T, list *umpirespb.Expr, item int) *umpirespb.Construct {
	t.Helper()
	c := list.GetList().GetItems()[item].GetConstruct()
	require.Equal(t, interp.StepType, c.GetType())
	return c
}

// namedAdmission is the admission specimen with its committed admission's alternatives named: a
// redelivered message is consumed, a first delivery consumed or retained. A dispatch, one result of
// its own, is named too; the stale-message rejection of admitCurrent is left unnamed. plain is the
// specimen with the names its source gives with `choose` cleared.
func namedAdmission(t *testing.T) (plain, named *umpirespb.Model) {
	t.Helper()
	plain = lifted(t, "admission")
	chosen := functionNamed(plain, admissionPackage+"admitted").GetBody().GetLet().GetBody().GetIf()
	stepAt(t, chosen.GetElse(), 0).Choice = ""
	stepAt(t, chosen.GetElse(), 1).Choice = ""
	named = proto.CloneOf(plain)
	admitted := functionNamed(named, admissionPackage+"admitted").GetBody().GetLet().GetBody().GetIf()
	stepAt(t, admitted.GetThen(), 0).Choice = "consumed"
	stepAt(t, admitted.GetElse(), 0).Choice = "consumed"
	stepAt(t, admitted.GetElse(), 1).Choice = "retained"
	stepAt(t, functionNamed(named, admissionPackage+"dispatchStep").GetBody().GetIf().GetElse(), 0).Choice = "enqueued"
	return plain, named
}

func choicesOf(row interp.Row) []string {
	out := []string{}
	for _, res := range row.Results {
		out = append(out, res.Choice)
	}
	return out
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
	require.EqualError(t, Validate(named), interp.Where(start.GetPosition())+
		": fixture.specimens.admission.AdmissionState names the choice initial, which only a step record can")
}

// TestTwoResultsOfARowCannotShareAName refuses a row two of whose results have the same name, at the
// position of the class's step binding, naming the row and the name. Unnamed results may be many.
func TestTwoResultsOfARowCannotShareAName(t *testing.T) {
	plain, named := namedAdmission(t)
	admitted := functionNamed(named, admissionPackage+"admitted").GetBody().GetLet().GetBody().GetIf()
	stepAt(t, admitted.GetElse(), 1).Choice = "consumed"
	require.NoError(t, Validate(named), "a row's results are only known once it is evaluated")
	built, err := interp.Build(plain)
	require.NoError(t, err)
	current := built["activityRecord"].Table
	first := slices.IndexFunc(current.Rows, func(r interp.Row) bool { return r.Action == "poll" && len(r.Results) == 2 })
	require.GreaterOrEqual(t, first, 0)
	binding := named.GetMachines()[1].GetSteps()[2]
	require.Equal(t, admissionPackage+"admitCurrent", binding.GetFunction())
	_, err = interp.Build(named)
	require.EqualError(t, err, interp.Where(binding.GetPosition())+": activityRecord: row "+current.Rows[first].Key+" has two results named consumed")

	stepAt(t, admitted.GetElse(), 0).Choice = ""
	stepAt(t, admitted.GetElse(), 1).Choice = ""
	_, err = interp.Build(named)
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
	before, err := interp.Build(plain)
	require.NoError(t, err)
	after, err := interp.Build(named)
	require.NoError(t, err)
	was, is := before["relay"].Table, after["relay"].Table
	wasJSON, err := json.Marshal(was)
	require.NoError(t, err)
	isJSON, err := json.Marshal(is)
	require.NoError(t, err)
	require.Equal(t, sha256.Sum256(wasJSON), sha256.Sum256(isJSON), "the table encodes to other bytes")
	redelivered := 0
	for _, row := range is.Rows {
		if !slices.ContainsFunc(row.Results, func(r interp.Result) bool { return r.Choice != "" }) {
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

// TestAlternativesAreNoClassesOfTheirOwn: a named choice's alternatives are results of one class, so
// they are no factor of a Query's combination total (model/SEMANTICS.md, Query totals). With one
// alternative of the committed admission instead of two, every Query's total and its factors, the
// state catalog, the classes, the counted evaluations and the rows are the same; only the results of
// the rows the second alternative was in are fewer.
func TestAlternativesAreNoClassesOfTheirOwn(t *testing.T) {
	_, two := namedAdmission(t)
	one := proto.CloneOf(two)
	admitted := functionNamed(one, admissionPackage+"admitted").GetBody().GetLet().GetBody().GetIf()
	admitted.GetElse().GetList().Items = admitted.GetElse().GetList().GetItems()[:1]
	actions := map[string]*umpirespb.Action{}
	for _, a := range two.GetActions() {
		actions[a.GetId()] = a
	}
	counts := func(m *umpirespb.Model, decl *umpirespb.Machine) (states, classes interp.Count, needed int64) {
		in := interp.NewInterpreterWithin(m, interp.Ceilings{Members: interp.DefaultCeilings.Members})
		states, err := in.Size(interp.Named(decl.GetStateType()))
		require.NoError(t, err)
		classes, err = in.ClassCount(decl, actions)
		require.NoError(t, err)
		var limit *interp.LimitError
		require.ErrorAs(t, in.Preflight(decl, actions), &limit)
		return states, classes, limit.Needed
	}
	withTwo, err := interp.Build(two)
	require.NoError(t, err)
	withOne, err := interp.Build(one)
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
	require.NotEmpty(t, two.GetQueries())
	for i, q := range two.GetQueries() {
		named, err := queryTotal(two, q)
		require.NoError(t, err, q.GetName())
		single, err := queryTotal(one, one.GetQueries()[i])
		require.NoError(t, err, q.GetName())
		require.Equal(t, single, named, q.GetName())
		n, ok := named.N()
		require.True(t, ok, q.GetName())
		require.Positive(t, n, q.GetName())
		require.Equal(t, q.GetTotal().GetValue(), n, "%s declares %d, counted %s", q.GetName(), q.GetTotal().GetValue(), named)
	}
	require.NoError(t, Validate(two))
	require.NoError(t, Validate(one))
}
