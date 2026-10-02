package backends

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	"go.temporal.io/server/model/go/umpire"
	"google.golang.org/protobuf/proto"
)

// pDepth is the bound on the event traces P is given: past the four steps of the specimen's path on
// which the stale design starts a completed activity (specimens/activity.md, A4).
const pDepth = 5

func pExported(t *testing.T, s *Slice, machine string) *PExport {
	t.Helper()
	x, err := s.PMonitor(machine, "terminalFinality", pDepth)
	require.NoError(t, err)
	return x
}

// Go's monitor says of the bounded traces what the specimen says of the designs: the stale design
// starts an activity that is over, and the corrected one never does.
func TestPTracesCarryGoVerdicts(t *testing.T) {
	s := openNamed(t, "activity-system")
	stale, current := pExported(t, s, "staleAdmission"), pExported(t, s, "currentAdmission")
	require.Positive(t, stale.Rejected)
	require.Positive(t, stale.Accepted)
	require.Equal(t, stale.Traces, stale.Accepted+stale.Rejected)
	require.Zero(t, current.Rejected)
	require.Positive(t, current.Accepted)
	t.Logf("staleAdmission: %d traces, %d accepted, %d rejected; currentAdmission: %d traces", stale.Traces, stale.Accepted, stale.Rejected, current.Traces)
	// The specimen's path is one of the rejected traces, at its fourth step.
	mm := s.machines["staleAdmission"]
	traces, err := s.traces(mm, slices.IndexFunc(mm.Monitors, func(mo *modelirspb.Monitor) bool { return mo.GetName() == "terminalFinality" }), pDepth)
	require.NoError(t, err)
	a4 := []string{"dispatch", "attemptStart", "attemptResult-completed", "attemptStart"}
	i := slices.IndexFunc(traces, func(tr ptrace) bool {
		return tr.violated == 4 && slices.EqualFunc(tr.steps, a4, func(st pstep, class string) bool { return st.class == class })
	})
	require.GreaterOrEqual(t, i, 0)
	// What the traces say of it is what goir's checker says: the path replays through a fresh
	// interpretation, and the checker finds a monitor violated over its classes.
	table := mm.Table
	witness := &umpire.Trace{Initial: table.StateAtom(traces[i].steps[0].before.Key())}
	for _, st := range traces[i].steps {
		step := umpire.TraceStep{Action: table.ActionAtom(st.class), Outcome: table.OutcomeAtom(st.after.Fields[0].Key()),
			State: table.StateAtom(st.after.Fields[1].Key())}
		for _, f := range st.after.Fields[2].Items {
			step.Facts = append(step.Facts, table.FactAtom(f.Key()))
		}
		witness.Steps = append(witness.Steps, step)
	}
	require.NoError(t, s.Replay("staleAdmission", "terminalFinality", witness))
	// The same steps from where the attempt started, which is no start: the monitor is violated at their
	// end too, and the path is still no witness.
	suffix := &umpire.Trace{Initial: witness.Steps[1].State, Steps: witness.Steps[2:]}
	require.ErrorContains(t, s.Replay("staleAdmission", "terminalFinality", suffix), "which is no start")
	asked := []Receipt{{Claim: MonitorAgreement, Subject: "staleAdmission", Kind: Agreed, Witnesses: []Witness{{Monitor: "terminalFinality", Trace: witness}}}}
	require.NoError(t, s.confirm(asked, false))
	require.Equal(t, Agreed, asked[0].Kind, asked[0].Explanation)
}

// What P is not given is refused by name: a monitor that reads a step's facts, one over a state whose
// cases carry values, and one read at the machine's ends.
func TestPExportRejectsWhatItDoesNotTranslate(t *testing.T) {
	cases := map[string]struct {
		model, machine, monitor string
		edit                    func(m *modelirspb.Model)
		says                    string
	}{
		"a monitor that reads facts whose cases carry values": {"activity-system", "staleAdmission", "atMostOneActiveAttempt", nil, "carries values"},
		"a monitor that reads a step's facts": {"activity-system", "staleAdmission", "terminalFinality", func(m *modelirspb.Model) {
			next := function(m, "temporal.standaloneactivity.System$package$.terminalFinality.next")
			facts := &modelirspb.Expr{Kind: &modelirspb.Expr_Field{Field: &modelirspb.FieldAccess{
				Base: &modelirspb.Expr{Kind: &modelirspb.Expr_Var{Var: "after"}}, Field: "facts"}}}
			next.Body = &modelirspb.Expr{Kind: &modelirspb.Expr_If{If: &modelirspb.If{
				Condition: &modelirspb.Expr{Kind: &modelirspb.Expr_Binary{Binary: &modelirspb.Binary{Op: modelirspb.Binary_OP_EQ, Left: facts, Right: facts}}},
				Then:      next.GetBody(), Else: next.GetBody()}}}
		}, "a step's facts"},
		"a state whose cases carry values": {"nexus-close", "retainAndRoute", "retainedOutcome", nil, "carries values"},
		"a monitor read at the ends": {"activity-system", "staleAdmission", "terminalFinality", func(m *modelirspb.Model) {
			for _, mo := range m.GetMonitors() {
				if mo.GetName() == "terminalFinality" {
					mo.Evaluate = &modelirspb.Monitor_AtEnds{AtEnds: &modelirspb.Empty{}}
				}
			}
		}, "read at the machine's ends"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			m := proto.Clone(loadModel(t, c.model)).(*modelirspb.Model)
			if c.edit != nil {
				c.edit(m)
			}
			_, err := openSlice(t, m).PMonitor(c.machine, c.monitor, 3)
			var unsupported *UnsupportedError
			require.ErrorAs(t, err, &unsupported)
			require.Equal(t, pBackend, unsupported.Backend)
			require.Contains(t, err.Error(), c.says)
		})
	}
}

// asGoSays is what P's checker reports of every test case when its monitor is Go's.
func asGoSays(x *PExport) map[string]PResult {
	out := map[string]PResult{}
	for _, test := range x.Tests {
		switch {
		case test.Control:
			out[test.Name] = PResult{Ran: true, Bugs: 1, Message: "P's monitor is first violated at step 0, and Go's at step 1, 0 being never"}
		case test.Violated > 0:
			out[test.Name] = PResult{Ran: true, Bugs: 1, Message: "Assertion Failed: the monitor terminalFinality is violated at step " + string(rune('0'+test.Violated))}
		default:
			out[test.Name] = PResult{Ran: true}
		}
	}
	return out
}

// The comparison of P's reports with Go's verdicts, with no tool installed: it agrees on what Go
// says, and every other report is a difference.
func TestPAgreementReadsTheCheckersReports(t *testing.T) {
	x := pExported(t, openNamed(t, "activity-system"), "staleAdmission")
	rejected := x.Tests[0]
	require.Positive(t, rejected.Violated)
	receipts := x.Agreement(asGoSays(x))
	require.Equal(t, map[string]Kind{
		"monitor-agreement staleAdmission.terminalFinality": Agreed,
		"checker-coverage staleAdmission.terminalFinality":  Covered,
		"module-refinement staleAdmission.terminalFinality": Unsupported,
	}, kinds(receipts))
	cases := map[string]struct {
		test   string
		report PResult
		says   string
	}{
		"a rejected trace P accepts":         {rejected.Name, PResult{Ran: true}, "P finds no violation"},
		"a rejection at another step":        {rejected.Name, PResult{Ran: true, Bugs: 1, Message: "the monitor terminalFinality is violated at step 1"}, "Go's monitor is violated at step"},
		"an accepted trace P rejects":        {"tcAccepted", PResult{Ran: true, Bugs: 1, Message: "the monitor terminalFinality is violated at step 2"}, "no trace of it fails"},
		"a first violation elsewhere":        {"tcAgreement", PResult{Ran: true, Bugs: 1, Message: "P's monitor is first violated at step 3"}, "no trace of it fails"},
		"a test case that did not run":       {"tcAgreement", PResult{}, "did not run"},
		"an agreement that compares nothing": {"tcControlWrongExpectation", PResult{Ran: true}, "did not fail"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			reports := asGoSays(x)
			reports[c.test] = c.report
			r := only(t, x.Agreement(reports), MonitorAgreement, "staleAdmission.terminalFinality")
			require.Equal(t, Disagreed, r.Kind)
			require.Len(t, r.Differences, 1)
			require.Contains(t, r.Differences[0], c.says)
		})
	}
}

func pReceipts(t *testing.T, x *PExport) []Receipt {
	t.Helper()
	receipts, err := RunP(t.Context(), needs(t, PTool), x, workDir(t))
	require.NoError(t, err)
	return receipts
}

// P's checker runs the exported monitor over every bounded trace of both designs and agrees with Go's
// monitor on each: the stale design's rejections, at their steps, and the corrected design's none.
func TestPMonitorAgreesWithGo(t *testing.T) {
	needs(t, PTool)
	s := openNamed(t, "activity-system")
	for _, machine := range []string{"staleAdmission", "currentAdmission"} {
		t.Run(machine, func(t *testing.T) {
			subject := machine + ".terminalFinality"
			receipts := pReceipts(t, pExported(t, s, machine))
			for _, r := range receipts {
				report(t, r)
			}
			require.Equal(t, map[string]Kind{"monitor-agreement " + subject: Agreed, "checker-coverage " + subject: Covered,
				"module-refinement " + subject: Unsupported}, kinds(receipts))
		})
	}
}

// A monitor exported from a Model that says something else is found out by P's own run of it, against
// the traces and verdicts of the Model Go reads.
func TestPDisagreesOnAnotherMonitor(t *testing.T) {
	needs(t, PTool)
	reference := openNamed(t, "activity-system")
	cases := map[string]func(m *modelirspb.Model){
		"over means completed and timed out at once": func(m *modelirspb.Model) {
			function(m, "temporal.standaloneactivity.System$package$.admissionOver").GetBody().GetBinary().Op = modelirspb.Binary_OP_AND
		},
		"violated when closed": func(m *modelirspb.Model) {
			function(m, "temporal.standaloneactivity.System$package$.terminalFinality.violated").GetBody().GetBinary().GetRight().GetLiteral().GetEnum().Case = "closed"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			mutant := proto.Clone(reference.Model).(*modelirspb.Model)
			mutate(mutant)
			written := &Slice{Model: mutant, machines: reference.machines, in: reference.in, bound: reference.bound, types: reference.types, actions: reference.actions}
			r := only(t, pReceipts(t, pExported(t, written, "staleAdmission")), MonitorAgreement, "staleAdmission.terminalFinality")
			require.Equal(t, Disagreed, r.Kind)
			t.Log(r.Differences[0])
		})
	}
}
