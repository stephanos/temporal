package export

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
)

func checkOf(t *testing.T, s *Slice, machine string) *QuintCheck {
	t.Helper()
	c, err := exported(t, s).Check(machine)
	require.NoError(t, err)
	return c
}

// counterexampleOf writes a path of a machine as the ITF trace a check module's counterexample is:
// the first state's machine state, and the last state's steps.
func counterexampleOf(t *testing.T, s *Slice, c *QuintCheck, trace *umpiremodel.Trace) []byte {
	t.Helper()
	x, mm := c.from, s.machines[c.Machine]
	decl := mm.Decl
	start, _ := mm.State(trace.Initial.Value)
	state := trace.Initial.Value
	hist := []any{}
	for _, step := range trace.Steps {
		for i, tr := range mm.Transitions {
			if tr.Source.Key() != state || tr.Class.Key != step.Action.Value {
				continue
			}
			for n, res := range mm.Table.Rows[i].Results {
				if res.State != step.State.Value || res.Outcome != step.Outcome.Value {
					continue
				}
				st := tr.Steps[n]
				facts := []any{}
				for _, f := range st.Fields[2].Items {
					facts = append(facts, x.itf(f, named(decl.GetFactType())))
				}
				dump, err := s.dumpOf(x, c.index, mm)
				require.NoError(t, err)
				// The class is written as the dump writes it.
				var class any
				for _, cls := range dump["classes"].(map[string]any)["#set"].([]any) {
					if key, err := x.reader(c.index).class(cls); err == nil && key == tr.Class.Key {
						class = cls
					}
				}
				hist = append(hist, map[string]any{"cls": class, "step": map[string]any{
					"f_outcome": x.itf(st.Fields[0], named(decl.GetOutcomeType())), "f_state": x.itf(st.Fields[1], named(decl.GetStateType())),
					"f_facts": facts, "f_because": st.Fields[3].Text}})
			}
		}
		state = step.State.Value
	}
	require.Len(t, hist, len(trace.Steps))
	encoded, err := json.Marshal(map[string]any{"states": []any{
		map[string]any{"st": x.itf(start, named(decl.GetStateType())), "hist": []any{}},
		map[string]any{"hist": hist}}})
	require.NoError(t, err)
	return encoded
}

// What the model checker reports is held to Go with no tool installed: its verdict against Go's
// product, and its counterexample against a fresh interpretation and the reader's checker.
func TestVerifiedVerdictsAreHeldToGo(t *testing.T) {
	s := openNamed(t, "activity-system")
	stale, current := checkOf(t, s, "staleAdmission"), checkOf(t, s, "currentAdmission")
	require.Equal(t, []string{"atMostOneActiveAttempt", "terminalFinality"}, stale.Monitors)
	const finality = 1
	// Go's own counterexample of the stale design, as the model checker would report it.
	p, err := s.product(s.machines["staleAdmission"])
	require.NoError(t, err)
	view, err := s.view(s.machines["staleAdmission"])
	require.NoError(t, err)
	path := counterexample(s.machines["staleAdmission"].Table, view, finality)
	require.Len(t, path.Steps, p.violatedAt["terminalFinality"])

	agreed := s.QuintVerified(stale, finality, QuintVerdict{Violated: true, Trace: counterexampleOf(t, s, stale, path)})
	require.Equal(t, Agreed, agreed.Kind, agreed.Explanation)
	require.Equal(t, []string{"terminalFinality"}, agreed.Violated)

	require.Equal(t, Agreed, s.QuintVerified(current, finality, QuintVerdict{}).Kind)

	missed := s.QuintVerified(stale, finality, QuintVerdict{})
	require.Equal(t, Disagreed, missed.Kind)
	require.Contains(t, missed.Differences[0], "Go finds terminalFinality violated")

	// A path of the machine on which the monitor holds, reported as its violation.
	held := &umpiremodel.Trace{Initial: path.Initial, Steps: path.Steps[:len(path.Steps)-1]}
	rejected := s.QuintVerified(stale, finality, QuintVerdict{Violated: true, Trace: counterexampleOf(t, s, stale, held)})
	require.Equal(t, WitnessRejected, rejected.Kind)
	require.Contains(t, rejected.Explanation, "is not violated on the last step")

	// A counterexample of another machine's check is no path of this one's.
	crossed := s.QuintVerified(current, finality, QuintVerdict{Violated: true, Trace: counterexampleOf(t, s, stale, path)})
	require.Equal(t, WitnessRejected, crossed.Kind)

	unreadable := s.QuintVerified(stale, finality, QuintVerdict{Violated: true, Trace: []byte("{}")})
	require.Equal(t, WitnessRejected, unreadable.Kind)
}

// Apalache checks the monitors of both activity designs on every run within the product's depth, and
// agrees with Go; each counterexample it finds replays through Go.
func TestQuintVerifyAgreesWithGo(t *testing.T) {
	found := needs(t, VerifyTool)
	t.Cleanup(StopVerifier)
	cases := map[string]map[string][]string{
		"activity-system": {
			"currentAdmission": {"atMostOneActiveAttempt", "terminalFinality"},
			"staleAdmission":   {"atMostOneActiveAttempt", "terminalFinality"},
		},
	}
	for model, machines := range cases {
		s := openNamed(t, model)
		for machine, monitors := range machines {
			c := checkOf(t, s, machine)
			for _, monitor := range monitors {
				t.Run(machine+"."+monitor, func(t *testing.T) {
					k := indexOf(t, c.Monitors, monitor)
					verdict, err := RunQuintVerify(t.Context(), found, c, k, workDir(t))
					require.NoError(t, err)
					r := s.QuintVerified(c, k, verdict)
					report(t, r)
					require.Equal(t, Agreed, r.Kind, "%v", r.Differences)
				})
			}
		}
	}
}

func indexOf(t *testing.T, names []string, name string) int {
	t.Helper()
	for i, n := range names {
		if n == name {
			return i
		}
	}
	t.Fatalf("no %s among %v", name, names)
	return -1
}

// Apalache does not take the Nexus close check module: its inliner gives up on the module's step
// functions. The run is an error, and its receipt says the check was not run: the Nexus monitors are
// compared by the evaluator's product (TestQuintAgreesWithGo) and by no model checker.
func TestQuintVerifyDoesNotTakeTheNexusModule(t *testing.T) {
	found := needs(t, VerifyTool)
	t.Cleanup(StopVerifier)
	s := openNamed(t, "nexus-close")
	c := checkOf(t, s, "rejectAfterClose")
	k := indexOf(t, c.Monitors, "retainedOutcome")
	_, err := RunQuintVerify(t.Context(), found, c, k, workDir(t))
	require.ErrorContains(t, err, "Recursive substitution took more than 100000 iterations")
	r := c.NotVerified(k, err)
	require.Equal(t, NotRun, r.Kind)
	require.Equal(t, CheckerCoverage, r.Claim)
	report(t, r)
}
