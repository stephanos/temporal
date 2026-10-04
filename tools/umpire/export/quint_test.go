package export

import (
	"cmp"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// slices of the lifted IR the backends are gated on, by file.
var irFiles = []string{"activity", "activity-system", "nexus-caller", "nexus-close"}

func loadModel(t *testing.T, name string) *umpirespb.Model {
	t.Helper()
	m, err := umpiremodel.Load(filepath.Join("..", "..", "..", "model", "ir", name+".json"))
	require.NoError(t, err)
	return m
}

func openSlice(t *testing.T, m *umpirespb.Model) *Slice {
	t.Helper()
	s, err := Open(m)
	require.NoError(t, err)
	return s
}

// openNamed is the slice of a lifted IR file, named after it.
func openNamed(t *testing.T, name string) *Slice {
	t.Helper()
	s := openSlice(t, loadModel(t, name))
	s.Name = name
	return s
}

func exported(t *testing.T, s *Slice) *QuintExport {
	t.Helper()
	x, err := s.Quint()
	require.NoError(t, err)
	return x
}

// kinds is each receipt's kind by claim and subject.
func kinds(receipts []Receipt) map[string]Kind {
	out := map[string]Kind{}
	for _, r := range receipts {
		out[string(r.Claim)+" "+r.Subject] = r.Kind
	}
	return out
}

func only(t *testing.T, receipts []Receipt, claim Claim, subject string) Receipt {
	t.Helper()
	i := slices.IndexFunc(receipts, func(r Receipt) bool { return r.Claim == claim && r.Subject == subject })
	require.GreaterOrEqual(t, i, 0, "no receipt %s %s", claim, subject)
	return receipts[i]
}

// The dump Go's own interpretation gives, in the export's encoding, is what a Quint evaluator that
// agrees with Go writes. The comparison is tested against it and against tampered copies of it with
// no tool installed.
func TestAgreementReadsAFaithfulDump(t *testing.T) {
	for _, name := range irFiles {
		t.Run(name, func(t *testing.T) {
			s := openNamed(t, name)
			x := exported(t, s)
			receipts, err := s.QuintAgreement(x, encodeDump(t, s, x, nil))
			require.NoError(t, err)
			for _, r := range receipts {
				if r.Kind != Covered && r.Kind != Unsupported {
					require.Equal(t, Agreed, r.Kind, "%s %s: %s %v", r.Claim, r.Subject, r.Explanation, r.Differences)
				}
			}
			for _, mm := range s.Model.GetMachines() {
				r := only(t, receipts, TransitionAgreement, mm.GetName())
				table := s.machines[mm.GetName()].Table
				require.Equal(t, len(table.Reachable), r.States)
				require.Equal(t, len(table.Reachable)*len(table.Actions), r.Pairs)
				require.Equal(t, r.Pairs, r.Enabled+r.Disabled)
			}
			for _, name := range x.Compositions {
				c, err := s.bound.Composition(name)
				require.NoError(t, err)
				r := only(t, receipts, TransitionAgreement, name)
				require.Equal(t, len(c.Table.States), r.States)
				require.Equal(t, len(c.Table.States)*len(c.Table.Actions), r.Pairs)
				require.Equal(t, len(c.Table.Rows), r.Enabled)
				require.Equal(t, len(c.Properties), only(t, receipts, PropertyAgreement, name).Properties)
			}
		})
	}
}

func TestAgreementRejectsATamperedDump(t *testing.T) {
	s := openNamed(t, "activity-system")
	x := exported(t, s)
	parts := dumpPartsOf(t, s, x)
	const machine = "staleAdmission"
	cases := map[string]struct {
		tamper func(d map[string]any)
		claim  Claim
		says   string
	}{
		"a disabled pair enabled": {func(d map[string]any) {
			row, pair := firstPair(d, false)
			other, _ := firstPair(d, true)
			pair["steps"] = other["by"].(map[string]any)["#set"].([]any)[indexOfEnabled(other)].(map[string]any)["steps"]
			_ = row
		}, TransitionAgreement, "disabled in Go"},
		"an enabled pair disabled": {func(d map[string]any) {
			row, _ := firstPair(d, true)
			row["by"].(map[string]any)["#set"].([]any)[indexOfEnabled(row)].(map[string]any)["steps"] = []any{}
		}, TransitionAgreement, "disabled in Quint"},
		"a disabled pair left out": {func(d map[string]any) {
			row, pair := firstPair(d, false)
			by := row["by"].(map[string]any)
			by["#set"] = slices.DeleteFunc(by["#set"].([]any), func(p any) bool {
				return fmt.Sprint(p.(map[string]any)["cls"]) == fmt.Sprint(pair["cls"])
			})
		}, TransitionAgreement, "no such pair"},
		"a reachable state missing": {func(d map[string]any) {
			reach := d["reach"].(map[string]any)
			reach["#set"] = reach["#set"].([]any)[1:]
		}, TransitionAgreement, "reachable"},
		"an open frontier": {func(d map[string]any) { d["closed"] = false }, TransitionAgreement, "closed"},
		"no start":         {func(d map[string]any) { d["starts"] = []any{} }, TransitionAgreement, "starts"},
		"a result's facts dropped": {func(d map[string]any) {
			row, _ := firstPair(d, true)
			steps := row["by"].(map[string]any)["#set"].([]any)[indexOfEnabled(row)].(map[string]any)["steps"].([]any)
			steps[0].(map[string]any)["f_facts"] = []any{}
		}, TransitionAgreement, "results"},
		"a result's explanation changed": {func(d map[string]any) {
			row, _ := firstPair(d, true)
			steps := row["by"].(map[string]any)["#set"].([]any)[indexOfEnabled(row)].(map[string]any)["steps"].([]any)
			steps[0].(map[string]any)["f_because"] = "another reason"
		}, TransitionAgreement, "results"},
		"a result's name changed": {func(d map[string]any) {
			row, _ := firstPair(d, true)
			steps := row["by"].(map[string]any)["#set"].([]any)[indexOfEnabled(row)].(map[string]any)["steps"].([]any)
			steps[0].(map[string]any)["f_choice"] = "anotherName"
		}, TransitionAgreement, "results"},
		"an end that is none": {func(d map[string]any) {
			d["ends"] = d["reach"]
		}, TransitionAgreement, "is in Quint's and not in Go's"},
		"a Property's reading flipped": {func(d map[string]any) {
			for _, row := range d["claims"].(map[string]any)["#set"].([]any) {
				for _, by := range row.(map[string]any)["by"].(map[string]any)["#set"].([]any) {
					for _, st := range by.(map[string]any)["steps"].([]any) {
						read := st.(map[string]any)["p0"].(map[string]any)
						read["holds"] = !read["holds"].(bool)
					}
				}
			}
		}, PropertyAgreement, "Go reads"},
		"a product pair left out": {func(d map[string]any) {
			edge := d["product"].(map[string]any)["edges"].(map[string]any)["#set"].([]any)[0].(map[string]any)
			by := edge["by"].(map[string]any)
			by["#set"] = by["#set"].([]any)[1:]
		}, MonitorAgreement, "no such pair"},
		"a Property's pair left out": {func(d map[string]any) {
			row := d["claims"].(map[string]any)["#set"].([]any)[0].(map[string]any)
			by := row["by"].(map[string]any)
			by["#set"] = by["#set"].([]any)[1:]
		}, PropertyAgreement, "no such pair"},
		"a monitor's violations missed": {func(d map[string]any) {
			for _, p := range d["product"].(map[string]any)["edges"].(map[string]any)["#set"].([]any) {
				for _, by := range p.(map[string]any)["by"].(map[string]any)["#set"].([]any) {
					for _, st := range by.(map[string]any)["steps"].([]any) {
						st.(map[string]any)["viol"].(map[string]any)["m1"] = false
					}
				}
			}
		}, MonitorAgreement, "terminalFinality"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			dump := parts.encode(t, func(m string, d map[string]any) {
				if m == machine {
					c.tamper(d)
				}
			})
			receipts, err := s.QuintAgreement(x, dump)
			require.NoError(t, err)
			r := only(t, receipts, c.claim, machine)
			require.Equal(t, Disagreed, r.Kind, r.Explanation)
			require.Contains(t, strings.Join(r.Differences, "\n"), c.says)
			// Another machine's agreement is its own.
			require.Equal(t, Agreed, only(t, receipts, TransitionAgreement, "currentAdmission").Kind)
		})
	}
}

// firstPair is the first row of a dump with a pair that is enabled, or one that is disabled, and that
// pair.
func firstPair(d map[string]any, enabled bool) (row, pair map[string]any) {
	for _, row := range d["rows"].(map[string]any)["#set"].([]any) {
		for _, pair := range row.(map[string]any)["by"].(map[string]any)["#set"].([]any) {
			if (len(pair.(map[string]any)["steps"].([]any)) > 0) == enabled {
				return row.(map[string]any), pair.(map[string]any)
			}
		}
	}
	panic("no such pair")
}

func indexOfEnabled(row map[string]any) int {
	return slices.IndexFunc(row["by"].(map[string]any)["#set"].([]any), func(p any) bool {
		return len(p.(map[string]any)["steps"].([]any)) > 0
	})
}

func TestADumpThatIsNoDumpIsAnError(t *testing.T) {
	s := openNamed(t, "activity")
	x := exported(t, s)
	for name, dump := range map[string]string{
		"no JSON":     "quint: command not found",
		"no states":   `{"vars":["out"],"states":[]}`,
		"no machine":  `{"vars":["out"],"states":[{"out":{}}]}`,
		"another tag": `{"vars":["out"],"states":[{"out":{"m0":{"starts":[{"tag":"T99_x","value":{"#tup":[]}}]}}}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := s.QuintAgreement(x, []byte(dump))
			require.Error(t, err)
		})
	}
}

// What the exporter does not translate it refuses by name: nothing is exported around it, and a hole
// is never written as a disabled action.
func TestQuintExportRejectsWhatItDoesNotTranslate(t *testing.T) {
	hole := &umpirespb.Expr{Kind: &umpirespb.Expr_Hole{Hole: "h"}}
	cases := map[string]struct {
		model string
		edit  func(m *umpirespb.Model)
		says  string
	}{
		"a declared hole in a step": {"", func(m *umpirespb.Model) {
			m.Holes = append(m.Holes, &umpirespb.Hole{Id: "h", Name: "unknownPolicy"})
			stepFunction(m, "activityWorker").Body = hole
		}, "hole unknownPolicy"},
		// A match with a case removed leaves a value no case matches: an undeclared hole, which Go reads
		// as hole rows.
		"a value no case matches": {"activity-system", func(m *umpirespb.Model) {
			match := function(m, "temporal.standaloneactivity.admission.Admission$.control").GetBody().GetMatch()
			match.Cases = match.GetCases()[:1]
		}, "an undeclared hole at the row"},
		"a hole in a machine's ends": {"", func(m *umpirespb.Model) {
			m.Holes = append(m.Holes, &umpirespb.Hole{Id: "h", Name: "unknownPolicy"})
			m.GetMachines()[0].GetEnds().GetLambda().Body = hole
		}, "the hole unknownPolicy"},
		"a channel": {"", func(m *umpirespb.Model) {
			m.Channels = append(m.Channels, &umpirespb.Channel{Id: "c", Name: "tasks", Capacity: 1, Order: umpirespb.Channel_ORDER_FIFO,
				Message: &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Bool{Bool: &umpirespb.Empty{}}}})
		}, "channel tasks"},
		"an anonymous function as a value": {"", func(m *umpirespb.Model) {
			f := stepFunction(m, "activityWorker")
			f.Body = &umpirespb.Expr{Kind: &umpirespb.Expr_Let{Let: &umpirespb.Let{Name: "f",
				Value: &umpirespb.Expr{Kind: &umpirespb.Expr_Lambda{Lambda: &umpirespb.Lambda{Body: f.GetBody()}}},
				Body:  f.GetBody()}}}
		}, "anonymous function"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			m := proto.Clone(loadModel(t, cmp.Or(c.model, "activity"))).(*umpirespb.Model)
			c.edit(m)
			s, err := Open(m)
			if err == nil {
				_, err = s.Quint()
			}
			var unsupported *UnsupportedError
			require.ErrorAs(t, err, &unsupported)
			require.Contains(t, err.Error(), c.says)
		})
	}
}

func function(m *umpirespb.Model, name string) *umpirespb.Function {
	i := slices.IndexFunc(m.GetFunctions(), func(f *umpirespb.Function) bool { return f.GetName() == name })
	if i < 0 {
		panic("no function " + name)
	}
	return m.GetFunctions()[i]
}

// overAtOnce narrows the admission record's notion of over, `s.phase.in(completed, timedOut)`, to
// both memberships at once: an activity is over only when it is completed and timed out, which no
// state is, so terminal finality is never closed and never reopened.
func overAtOnce(m *umpirespb.Model) {
	body := function(m, "temporal.standaloneactivity.admission.Admission$.terminal").GetBody()
	in := body.GetBinary()
	is := func(item *umpirespb.Expr) *umpirespb.Expr {
		return &umpirespb.Expr{Position: item.GetPosition(), Kind: &umpirespb.Expr_Binary{Binary: &umpirespb.Binary{
			Op: umpirespb.Binary_OP_EQ, Left: proto.Clone(in.GetLeft()).(*umpirespb.Expr), Right: item}}}
	}
	items := in.GetRight().GetList().GetItems()
	body.Kind = &umpirespb.Expr_Binary{Binary: &umpirespb.Binary{Op: umpirespb.Binary_OP_AND, Left: is(items[0]), Right: is(items[1])}}
}

// stepFunction is the function of a machine's first step binding.
func stepFunction(m *umpirespb.Model, machine string) *umpirespb.Function {
	for _, mm := range m.GetMachines() {
		if mm.GetName() != machine {
			continue
		}
		for _, f := range m.GetFunctions() {
			if f.GetName() == mm.GetSteps()[0].GetFunction() {
				return f
			}
		}
	}
	panic("no step function of " + machine)
}

// What is in the slice and not exported is listed, and is no agreement: compositions, refinements and
// progress claims.
func TestQuintExportListsWhatItLeavesOut(t *testing.T) {
	s := openNamed(t, "activity-system")
	x := exported(t, s)
	got := kinds(x.Unsupported)
	require.Equal(t, Unsupported, got["module-refinement matchingQueue"])
	require.Equal(t, Unsupported, got["query-agreement 84 Queries"])
	// Five of the seven compositions are exported. The two over a provider that does not refine the
	// queue it replaces have no composed table in Go either: the reader rejects the replacement.
	require.Equal(t, []string{"currentOverLossyMatching", "currentOverMatching", "currentOverQueue", "staleOverMatching", "staleOverQueue"}, x.Compositions)
	for _, rejected := range []string{"currentOverForgetful", "currentOverVolatile"} {
		r := only(t, x.Unsupported, TransitionAgreement, rejected)
		require.Equal(t, Unsupported, r.Kind)
		require.Contains(t, r.Explanation, "goir rejects the replacement")
	}
	// A replacement that holds is Go's verdict, and no refinement is exported for it.
	for _, replacing := range []string{"currentOverLossyMatching", "currentOverMatching", "staleOverMatching"} {
		require.Contains(t, only(t, x.Unsupported, ModuleRefinement, replacing).Explanation, "goir's verdict")
	}
	// Seven machine refinements, three replacements, two rejected compositions and the Queries; the
	// activity system declares no progress claim.
	require.Len(t, x.Unsupported, 7+3+2+1)
	progress := kinds(exported(t, openNamed(t, "nexus-close")).Unsupported)
	require.Equal(t, Unsupported, progress["progress-agreement retainAndRoute.outcomeReachesOwner"])
	require.Len(t, progress, 10+1)
	for _, r := range x.Unsupported {
		require.NotEmpty(t, r.Explanation)
	}
}

// The monitors a design violates are the ones the specimens say it violates, read off Go's own
// product of the machine and its monitors; Quint's must equal them (TestQuintAgreement).
func TestGoMonitorVerdictsAreTheSpecimens(t *testing.T) {
	want := map[string]map[string][]string{
		"activity-system": {
			"currentAdmission": nil,
			"staleAdmission":   {"atMostOneActiveAttempt", "terminalFinality"},
		},
		"nexus-close": {
			"retainAndRoute":             nil,
			"retainAndRouteBoundedRetry": nil,
			"retainAndRouteWithDeadline": nil,
			"rejectAfterClose":           {"retainedOutcome"},
			"ackByOriginal":              {"ownerAcknowledgment"},
			"forgetsCancelOnReset":       {"cancelPrincipal"},
		},
	}
	for name, machines := range want {
		s := openNamed(t, name)
		for machine, violated := range machines {
			t.Run(machine, func(t *testing.T) {
				p, err := s.product(s.machines[machine])
				require.NoError(t, err)
				got := keysOf(p.violatedAt)
				for _, m := range violated {
					require.Contains(t, got, m)
				}
				if violated == nil {
					require.Empty(t, got)
				}
				require.NotContains(t, got, "singleOutcome")
			})
		}
	}
}

// A counterexample read off the backend's product is replayed through a fresh interpretation, and one
// that is no path of the Model, or on which the monitor holds, is an error, never a result.
func TestExternalWitnessesReplayOrAreErrors(t *testing.T) {
	s := openNamed(t, "activity-system")
	x := exported(t, s)
	parts := dumpPartsOf(t, s, x)
	receipts, err := s.QuintAgreement(x, parts.encode(t, nil))
	require.NoError(t, err)
	r := only(t, receipts, MonitorAgreement, "staleAdmission")
	require.Len(t, r.Witnesses, 2)
	for _, w := range r.Witnesses {
		require.NoError(t, s.Replay("staleAdmission", w.Monitor, w.Trace))
	}
	// The same paths against the corrected design: the first is no path of it, or the monitor holds.
	for _, w := range r.Witnesses {
		require.Error(t, s.Replay("currentAdmission", w.Monitor, w.Trace))
	}
	// A dump whose product takes a step the Model does not: the witness through it is rejected, and the
	// receipt is the error.
	tampered := parts.encode(t, func(m string, d map[string]any) {
		if m != "currentAdmission" {
			return
		}
		for _, p := range d["product"].(map[string]any)["edges"].(map[string]any)["#set"].([]any) {
			for _, by := range p.(map[string]any)["by"].(map[string]any)["#set"].([]any) {
				for _, st := range by.(map[string]any)["steps"].([]any) {
					st.(map[string]any)["viol"].(map[string]any)["m1"] = true
				}
			}
		}
	})
	receipts, err = s.QuintAgreement(x, tampered)
	require.NoError(t, err)
	r = only(t, receipts, MonitorAgreement, "currentAdmission")
	require.Equal(t, WitnessRejected, r.Kind)
	require.Contains(t, r.Explanation, "did not replay")
}

// encodeDump writes the dump of Go's own interpretation in the export's encoding, with each machine's
// part tampered with.
func encodeDump(t *testing.T, s *Slice, x *QuintExport, tamper func(machine string, d map[string]any)) []byte {
	t.Helper()
	return dumpPartsOf(t, s, x).encode(t, tamper)
}

// dumpParts is the dump of Go's own interpretation, each machine's and each composition's part
// encoded apart. The parts are interpreted once and never changed: every encoding decodes them
// afresh, so what one tampering changes no other encoding sees.
type dumpParts struct {
	keys, names []string
	encoded     [][]byte
}

func dumpPartsOf(t *testing.T, s *Slice, x *QuintExport) *dumpParts {
	t.Helper()
	parts := &dumpParts{}
	add := func(key, name string, d any, err error) {
		require.NoError(t, err)
		encoded, err := json.Marshal(d)
		require.NoError(t, err)
		parts.keys, parts.names, parts.encoded = append(parts.keys, key), append(parts.names, name), append(parts.encoded, encoded)
	}
	for i, name := range x.Machines {
		d, err := s.dumpOf(x, i, s.machines[name])
		add(fmt.Sprintf("m%d", i), name, d, err)
	}
	for j, name := range x.Compositions {
		d, err := s.dumpOfComposition(x, j)
		add(fmt.Sprintf("c%d", j), name, d, err)
	}
	return parts
}

// encode writes the dump with each part tampered with.
func (p *dumpParts) encode(t *testing.T, tamper func(machine string, d map[string]any)) []byte {
	t.Helper()
	out := map[string]any{}
	for i, key := range p.keys {
		// A round trip gives the tamperers plain JSON values.
		var plain map[string]any
		require.NoError(t, json.Unmarshal(p.encoded[i], &plain))
		if tamper != nil {
			tamper(p.names[i], plain)
		}
		out[key] = plain
	}
	encoded, err := json.Marshal(map[string]any{"vars": []string{"out"}, "states": []any{map[string]any{"out": out}}})
	require.NoError(t, err)
	return encoded
}

// quintDump is what Quint's evaluator computes of an export.
func quintDump(t *testing.T, x *QuintExport) []byte {
	t.Helper()
	dump, err := RunQuint(t.Context(), needs(t, QuintTool), x, workDir(t))
	require.NoError(t, err)
	return dump
}

// Quint evaluates the export of every slice, and agrees with Go on all of it.
func TestQuintAgreesWithGo(t *testing.T) {
	needs(t, QuintTool)
	for _, name := range irFiles {
		t.Run(name, func(t *testing.T) {
			s := openNamed(t, name)
			x := exported(t, s)
			receipts, err := s.QuintAgreement(x, quintDump(t, x))
			require.NoError(t, err)
			for _, r := range receipts {
				report(t, r)
				if r.Kind != Covered && r.Kind != Unsupported {
					require.Equal(t, Agreed, r.Kind, "%s %s: %v", r.Claim, r.Subject, r.Differences)
				}
			}
			for _, mm := range s.Model.GetMachines() {
				require.Positive(t, only(t, receipts, TransitionAgreement, mm.GetName()).Enabled)
			}
			for _, name := range x.Compositions {
				require.Positive(t, only(t, receipts, TransitionAgreement, name).Enabled)
				require.Positive(t, only(t, receipts, PropertyAgreement, name).Reads)
			}
		})
	}
}

// A Model whose export says something else than the Model Go reads is found out by Quint's own
// evaluation of it: the agreement is no comparison of Go with itself.
func TestQuintDisagreesOnAnotherModel(t *testing.T) {
	needs(t, QuintTool)
	cases := map[string]struct {
		model   string
		mutate  func(m *umpirespb.Model)
		claim   Claim
		subject string
	}{
		"a step's condition inverted": {"activity-system", func(m *umpirespb.Model) {
			branch := function(m, "temporal.standaloneactivity.admission.Admission$.dispatch").GetBody().GetIf()
			branch.Then, branch.Else = branch.GetElse(), branch.GetThen()
		}, TransitionAgreement, "currentAdmission"},
		"a Property negated": {"activity-system", func(m *umpirespb.Model) {
			holds := function(m, "staleAdmission.property.atMostOneActive")
			holds.Body = &umpirespb.Expr{Kind: &umpirespb.Expr_Unary{Unary: &umpirespb.Unary{Op: umpirespb.Unary_OP_NOT, Operand: holds.GetBody()}}}
		}, PropertyAgreement, "staleAdmission"},
		"a step's condition inverted, in a composition": {"activity-system", func(m *umpirespb.Model) {
			branch := function(m, "temporal.standaloneactivity.admission.Admission$.dispatch").GetBody().GetIf()
			branch.Then, branch.Else = branch.GetElse(), branch.GetThen()
		}, TransitionAgreement, "currentOverMatching"},
		"a composition's Property negated": {"activity-system", func(m *umpirespb.Model) {
			holds := function(m, "staleOverQueue.property.failedCommitKeepsTheMessage")
			holds.Body = &umpirespb.Expr{Kind: &umpirespb.Expr_Unary{Unary: &umpirespb.Unary{Op: umpirespb.Unary_OP_NOT, Operand: holds.GetBody()}}}
		}, PropertyAgreement, "staleOverQueue"},
		"a monitor's notion of over narrowed": {"activity-system", func(m *umpirespb.Model) {
			overAtOnce(m)
		}, MonitorAgreement, "staleAdmission"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			reference := openNamed(t, c.model)
			mutant := proto.Clone(reference.Model).(*umpirespb.Model)
			c.mutate(mutant)
			x := exportedAsWritten(t, reference, mutant)
			receipts, err := reference.QuintAgreement(x, quintDump(t, x))
			require.NoError(t, err)
			r := only(t, receipts, c.claim, c.subject)
			require.Equal(t, Disagreed, r.Kind, r.Explanation)
			t.Log(r.Differences[0])
		})
	}
}

// exportedAsWritten is the export of a Model that differs from a slice's in its functions alone. Go's
// reading of the slice enters the module only in how many rounds of successors it takes.
func exportedAsWritten(t *testing.T, reference *Slice, written *umpirespb.Model) *QuintExport {
	t.Helper()
	x, err := (&Slice{Model: written, machines: reference.machines, in: reference.in, bound: reference.bound, types: reference.types, actions: reference.actions}).Quint()
	require.NoError(t, err)
	return x
}

// What the module has no value for stops Quint's evaluator: a call outside its precondition, like a
// value no case matches, is an error of the run and never a result to compare.
func TestQuintStopsWhereTheModelHasNoValue(t *testing.T) {
	found := needs(t, QuintTool)
	reference := openNamed(t, "nexus-caller")
	mutant := proto.Clone(reference.Model).(*umpirespb.Model)
	function(mutant, "temporal.nexuscaller.Protocol$.saturatingSucc").GetBody().GetIf().GetCondition().GetBinary().Op = umpirespb.Binary_OP_LE
	_, err := RunQuint(t.Context(), found, exportedAsWritten(t, reference, mutant), workDir(t))
	require.ErrorContains(t, err, "Runtime error")
}

// A counterexample the backend names for one monitor, on a path that violates another: the path is a
// path of the machine and the reader's checker finds a violation over its classes, and the replay still
// rejects it, because the named monitor holds on it.
func TestAWitnessOfAnotherMonitorIsRejected(t *testing.T) {
	s := openNamed(t, "activity-system")
	x := exported(t, s)
	tampered := encodeDump(t, s, x, func(m string, d map[string]any) {
		if m != "staleAdmission" {
			return
		}
		for _, p := range d["product"].(map[string]any)["edges"].(map[string]any)["#set"].([]any) {
			for _, by := range p.(map[string]any)["by"].(map[string]any)["#set"].([]any) {
				for _, st := range by.(map[string]any)["steps"].([]any) {
					read, viol := st.(map[string]any)["read"].(map[string]any), st.(map[string]any)["viol"].(map[string]any)
					viol["m1"] = read["m0"].(bool) && viol["m0"].(bool)
				}
			}
		}
	})
	receipts, err := s.QuintAgreement(x, tampered)
	require.NoError(t, err)
	r := only(t, receipts, MonitorAgreement, "staleAdmission")
	require.Equal(t, WitnessRejected, r.Kind)
	require.Contains(t, r.Explanation, "the monitor terminalFinality is not violated on the last step")
}

// The reader's checker's answers are folded into a monitor agreement: a counterexample it finds no
// violation on is rejected, and a verdict that is not the backend's is a difference.
func TestCheckerAnswersAreFoldedIn(t *testing.T) {
	witness := Witness{Monitor: "terminalFinality"}
	base := Receipt{Claim: MonitorAgreement, Subject: "m", Kind: Agreed, Violated: []string{"terminalFinality"}, Witnesses: []Witness{witness}}
	free, replay := "backends.monitors.m.0", "backends.witness.m.terminalFinality"
	found := umpiremodel.Receipt{Kind: umpiremodel.Counterexample, Monitor: "terminalFinality"}
	cases := map[string]struct {
		receipt Receipt
		answers map[string]umpiremodel.Receipt
		want    Kind
		says    string
	}{
		"the same verdict":               {base, map[string]umpiremodel.Receipt{free: found, replay: found}, Agreed, ""},
		"a counterexample verified":      {base, map[string]umpiremodel.Receipt{free: found, replay: {Kind: umpiremodel.Verified}}, WitnessRejected, ""},
		"a counterexample of no monitor": {base, map[string]umpiremodel.Receipt{free: found, replay: {Kind: umpiremodel.Counterexample}}, WitnessRejected, ""},
		"another monitor named": {base, map[string]umpiremodel.Receipt{free: {Kind: umpiremodel.Counterexample, Monitor: "atMostOneActiveAttempt"}, replay: found},
			Disagreed, "finds the monitor atMostOneActiveAttempt violated"},
		"verified where the backend finds a violation": {base, map[string]umpiremodel.Receipt{free: {Kind: umpiremodel.Verified}, replay: found},
			Disagreed, "verifies the monitors on every path"},
		"a violation the backend does not find": {Receipt{Claim: MonitorAgreement, Subject: "m", Kind: Agreed},
			map[string]umpiremodel.Receipt{free: found}, Disagreed, "finds the monitor terminalFinality violated"},
		"a limit": {Receipt{Claim: MonitorAgreement, Subject: "m", Kind: Agreed},
			map[string]umpiremodel.Receipt{free: {Kind: umpiremodel.LimitReached}}, Disagreed, "limit-reached"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got := confirmed(c.receipt, 1, c.answers)
			require.Equal(t, c.want, got.Kind, got.Explanation)
			require.Contains(t, strings.Join(got.Differences, "\n"), c.says)
		})
	}
}

// A Property about an action is about the steps of that action's classes and no others, counted here
// off the table's own keys.
func TestPropertiesAboutAnActionAreReadOnItsSteps(t *testing.T) {
	s := aboutActions(t)
	checked := 0
	for _, mm := range s.machines {
		view, err := s.view(mm)
		require.NoError(t, err)
		for k, p := range s.properties(mm.Decl.GetName()) {
			if p.GetWhenAction() == "" {
				continue
			}
			want, total := 0, 0
			for _, state := range mm.Table.Reachable {
				for _, row := range mm.Table.RowsFrom(state) {
					total += len(row.Results)
					if action, _, _ := strings.Cut(row.Action, "-"); action == p.GetWhenAction() {
						want += len(row.Results)
					}
				}
			}
			got := 0
			for _, by := range view.Claims {
				for _, steps := range by {
					for _, reads := range steps {
						if reads[k].About {
							got++
						}
					}
				}
			}
			require.Equal(t, want, got, p.GetName())
			require.Less(t, want, total, p.GetName())
			require.Positive(t, want, p.GetName())
			checked++
		}
	}
	require.Positive(t, checked)
}

// aboutActions is the activity slice with each Property about one class rewritten as one about that
// class's action: no exported machine of the lifted slices declares a Property about an action.
func aboutActions(t *testing.T) *Slice {
	t.Helper()
	m := proto.Clone(loadModel(t, "activity")).(*umpirespb.Model)
	names := map[string]string{}
	for _, a := range m.GetActions() {
		names[a.GetId()] = a.GetName()
	}
	for _, p := range m.GetProperties() {
		// The composition's Properties are not exported, and admission reads theirs by composed keys.
		if class := p.GetWhenClass(); class != nil && !slices.ContainsFunc(m.GetCompositions(), func(c *umpirespb.Composition) bool {
			return c.GetName() == p.GetMachine()
		}) {
			p.When = &umpirespb.Property_WhenAction{WhenAction: names[class.GetAction()]}
		}
	}
	s := openSlice(t, m)
	s.Name = "activity, Properties by action"
	return s
}

// Quint reads a Property about an action on the steps Go reads it on.
func TestQuintReadsPropertiesAboutAnAction(t *testing.T) {
	needs(t, QuintTool)
	s := aboutActions(t)
	x := exported(t, s)
	receipts, err := s.QuintAgreement(x, quintDump(t, x))
	require.NoError(t, err)
	r := only(t, receipts, PropertyAgreement, "activityProtocol")
	require.Equal(t, Agreed, r.Kind, "%v", r.Differences)
	require.Positive(t, r.About)
	require.Less(t, r.About, r.Reads)
}

// Every composition the reader builds is exported: the nine of the lifted slices but the two whose
// replacement the reader rejects, with their 19 Properties but those two's none.
func TestCompositionsAreExported(t *testing.T) {
	exportedOf := map[string][]string{
		"activity":        {"standaloneActivity"},
		"activity-system": {"currentOverLossyMatching", "currentOverMatching", "currentOverQueue", "staleOverMatching", "staleOverQueue"},
		"nexus-caller":    {"nexusCaller"},
		"nexus-close":     nil,
	}
	properties := 0
	for name, want := range exportedOf {
		s := openNamed(t, name)
		x := exported(t, s)
		require.Equal(t, want, x.Compositions, name)
		for _, c := range x.Compositions {
			properties += len(s.properties(c))
		}
	}
	require.Equal(t, 19, properties)
}

// A composition past the scope's ceiling is a resource-limit receipt: Go builds no table of it, and
// no part of it is exported.
func TestACompositionPastTheCeilingIsAResourceLimit(t *testing.T) {
	scope := umpiremodel.DefaultScope
	scope.Compose.States = 3
	s, err := OpenWithin(loadModel(t, "activity"), scope)
	require.NoError(t, err)
	x := exported(t, s)
	require.Empty(t, x.Compositions)
	r := only(t, x.Unsupported, TransitionAgreement, "standaloneActivity")
	require.Equal(t, ResourceLimit, r.Kind)
	require.Contains(t, r.Explanation, "states")
	require.NotContains(t, x.Text, "// The composition standaloneActivity.")
}

// A composition's part of a dump is compared as a machine's is: a result, a disabled pair and a
// Property's reading that differ are differences of the composition's own receipts.
func TestAgreementRejectsATamperedComposition(t *testing.T) {
	s := openNamed(t, "activity-system")
	x := exported(t, s)
	parts := dumpPartsOf(t, s, x)
	const composition = "staleOverQueue"
	cases := map[string]struct {
		tamper func(d map[string]any)
		claim  Claim
		says   string
	}{
		"a composed outcome changed": {func(d map[string]any) {
			row, _ := firstPair(d, true)
			steps := row["by"].(map[string]any)["#set"].([]any)[indexOfEnabled(row)].(map[string]any)["steps"].([]any)
			steps[0].(map[string]any)["f_outcome"] = "queue_accepted"
		}, TransitionAgreement, "results"},
		"a composed fact dropped": {func(d map[string]any) {
			for _, row := range d["rows"].(map[string]any)["#set"].([]any) {
				for _, pair := range row.(map[string]any)["by"].(map[string]any)["#set"].([]any) {
					for _, st := range pair.(map[string]any)["steps"].([]any) {
						st.(map[string]any)["f_facts"] = []any{}
					}
				}
			}
		}, TransitionAgreement, "results"},
		"a composed pair left out": {func(d map[string]any) {
			row, pair := firstPair(d, false)
			by := row["by"].(map[string]any)
			by["#set"] = slices.DeleteFunc(by["#set"].([]any), func(p any) bool {
				return fmt.Sprint(p.(map[string]any)["cls"]) == fmt.Sprint(pair["cls"])
			})
		}, TransitionAgreement, "no such pair"},
		"a composed start missing": {func(d map[string]any) { d["starts"] = []any{} }, TransitionAgreement, "starts"},
		"a composed result named": {func(d map[string]any) {
			row, _ := firstPair(d, true)
			steps := row["by"].(map[string]any)["#set"].([]any)[indexOfEnabled(row)].(map[string]any)["steps"].([]any)
			steps[0].(map[string]any)["f_choice"] = "aName"
		}, TransitionAgreement, "results"},
		"a composition's Property flipped": {func(d map[string]any) {
			for _, row := range d["claims"].(map[string]any)["#set"].([]any) {
				for _, by := range row.(map[string]any)["by"].(map[string]any)["#set"].([]any) {
					for _, st := range by.(map[string]any)["steps"].([]any) {
						read := st.(map[string]any)["p1"].(map[string]any)
						read["holds"] = !read["holds"].(bool)
					}
				}
			}
		}, PropertyAgreement, "failedCommitKeepsTheMessage"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			dump := parts.encode(t, func(m string, d map[string]any) {
				if m == composition {
					c.tamper(d)
				}
			})
			receipts, err := s.QuintAgreement(x, dump)
			require.NoError(t, err)
			r := only(t, receipts, c.claim, composition)
			require.Equal(t, Disagreed, r.Kind, r.Explanation)
			require.Contains(t, strings.Join(r.Differences, "\n"), c.says)
			require.Equal(t, Agreed, only(t, receipts, TransitionAgreement, "currentOverQueue").Kind)
		})
	}
}

// admittedSteps are the step records of the two alternatives of the activity system's `admitted`, in
// the order its result list holds them.
func admittedSteps(m *umpirespb.Model) []*umpirespb.Construct {
	var out []*umpirespb.Construct
	for _, item := range function(m, "temporal.standaloneactivity.admission.Admission$.admitted").GetBody().GetList().GetItems() {
		out = append(out, item.GetConstruct())
	}
	return out
}

// namedChoices is the activity system with the alternatives of `admitted` named, as the lifter names
// each alternative of a Scala `choose`.
func namedChoices(t *testing.T, names ...string) *Slice {
	t.Helper()
	m := proto.Clone(loadModel(t, "activity-system")).(*umpirespb.Model)
	steps := admittedSteps(m)
	require.Len(t, steps, len(names))
	for i, c := range steps {
		c.Choice = names[i]
	}
	s := openSlice(t, m)
	s.Name = "activity-system, named"
	return s
}

// A named choice is written whole: the step function keeps every alternative, in order, each with its
// name on its record, and the check module's step action, not the step function, picks one by index.
// The dump reports the names, and Quint's agrees with Go's only where every name is Go's.
func TestQuintKeepsEveryNamedAlternative(t *testing.T) {
	s := namedChoices(t, "accepts", "rejects")
	x := exported(t, s)
	accepts, rejects := strings.Index(x.Text, `f_choice: "accepts"}`), strings.Index(x.Text, `f_choice: "rejects"}`)
	require.Equal(t, 1, strings.Count(x.Text, `f_choice: "accepts"}`))
	require.Equal(t, 1, strings.Count(x.Text, `f_choice: "rejects"}`))
	require.Positive(t, accepts)
	require.Less(t, accepts, rejects)
	// One list holds both records: nothing is selected or dropped between them.
	list := x.Text[strings.LastIndex(x.Text[:accepts], "= [{f_outcome: "):]
	list = list[:strings.Index(list, "}]")+2]
	require.Contains(t, list, `f_choice: "accepts"}, {f_outcome: `)
	require.True(t, strings.HasSuffix(list, `f_choice: "rejects"}]`), list)
	require.NotContains(t, list, "oneOf")

	// Go's table reports the names, in the order of the list, on the rows that read `admitted`.
	named := map[string]bool{}
	for _, name := range x.Machines {
		for _, row := range s.machines[name].Table.Rows {
			var choices []string
			for _, r := range row.Results {
				if r.Choice != "" {
					choices = append(choices, r.Choice)
				}
			}
			// The queue's enqueue names its own alternatives in the Model.
			if len(choices) > 0 && !slices.Equal(choices, []string{"enqueueCommits", "enqueueFails"}) {
				require.Equal(t, []string{"accepts", "rejects"}, choices, name)
				named[name] = true
			}
		}
	}
	require.Contains(t, named, "staleAdmission")

	receipts, err := s.QuintAgreement(x, encodeDump(t, s, x, nil))
	require.NoError(t, err)
	for _, r := range receipts {
		if r.Kind != Covered && r.Kind != Unsupported {
			require.Equal(t, Agreed, r.Kind, "%s %s: %s %v", r.Claim, r.Subject, r.Explanation, r.Differences)
		}
	}
	// Each tampering rewrites the names of every step of staleAdmission's part of the dump.
	for name, tamper := range map[string]func(st map[string]any){
		"a wrong name": func(st map[string]any) {
			if st["f_choice"] == "accepts" {
				st["f_choice"] = "rejects"
			}
		},
		"a missing name": func(st map[string]any) {
			if st["f_choice"] == "accepts" {
				delete(st, "f_choice")
			}
		},
		"an empty name": func(st map[string]any) {
			if st["f_choice"] == "rejects" {
				st["f_choice"] = ""
			}
		},
		"the names swapped": func(st map[string]any) {
			switch st["f_choice"] {
			case "accepts":
				st["f_choice"] = "rejects"
			case "rejects":
				st["f_choice"] = "accepts"
			default:
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			dump := encodeDump(t, s, x, func(m string, d map[string]any) {
				if m != "staleAdmission" {
					return
				}
				for _, row := range d["rows"].(map[string]any)["#set"].([]any) {
					for _, pair := range row.(map[string]any)["by"].(map[string]any)["#set"].([]any) {
						for _, st := range pair.(map[string]any)["steps"].([]any) {
							tamper(st.(map[string]any))
						}
					}
				}
			})
			receipts, err := s.QuintAgreement(x, dump)
			require.NoError(t, err)
			r := only(t, receipts, TransitionAgreement, "staleAdmission")
			require.Equal(t, Disagreed, r.Kind, r.Explanation)
			require.Contains(t, strings.Join(r.Differences, "\n"), "the results differ")
			require.Equal(t, Agreed, only(t, receipts, TransitionAgreement, "currentAdmission").Kind)
		})
	}

	c, err := x.Check("staleAdmission")
	require.NoError(t, err)
	require.Regexp(t, `val rs = m\d+_step\(st, c\)\n    nondet n = oneOf\(rs\.indices\(\)\)\n    val r = rs\[n\]`, c.Text)
	require.Regexp(t, `var hist: List\[\{cls: K\d+, step: \{f_outcome: [^}]*, f_because: str, f_choice: str\}\}\]`, c.Text)
}

// Quint evaluates the export of named alternatives and agrees with Go on every result and its name.
func TestQuintAgreesWithGoOnNamedChoices(t *testing.T) {
	needs(t, QuintTool)
	s := namedChoices(t, "accepts", "rejects")
	x := exported(t, s)
	receipts, err := s.QuintAgreement(x, quintDump(t, x))
	require.NoError(t, err)
	for _, r := range receipts {
		report(t, r)
		if r.Kind != Covered && r.Kind != Unsupported {
			require.Equal(t, Agreed, r.Kind, "%s %s: %v", r.Claim, r.Subject, r.Differences)
		}
	}
}

// A name Quint cannot write as it is refused at the step record that carries it: a Quint string is
// the text between two double quotes, with no escapes.
func TestQuintRefusesANameItCannotWrite(t *testing.T) {
	for _, name := range []string{`say "hi"`, `back\slash`, "two\nlines", "tab\there", "caf\u00e9"} {
		t.Run(name, func(t *testing.T) {
			m := proto.Clone(loadModel(t, "activity-system")).(*umpirespb.Model)
			steps := admittedSteps(m)
			steps[0].Choice, steps[1].Choice = "accepts", name
			at := function(m, "temporal.standaloneactivity.admission.Admission$.admitted").GetBody().GetList().GetItems()[1].GetPosition()
			s, err := Open(m)
			require.NoError(t, err)
			_, err = s.Quint()
			var unsupported *UnsupportedError
			require.ErrorAs(t, err, &unsupported)
			require.Equal(t, fmt.Sprintf("%s:%d", at.GetFile(), at.GetLine()), unsupported.Position)
			require.Contains(t, unsupported.Construct, fmt.Sprintf("the choice %q", name))
		})
	}
	// Printable ASCII but for those two is written as it is.
	x := exported(t, namedChoices(t, "accepts", "it's-a $name_1 (ok)"))
	require.Contains(t, x.Text, `f_choice: "it's-a $name_1 (ok)"}]`)
}

// A step record carries the empty name unless the Model names it, and the step type its field.
func TestUnnamedStepRecordsCarryNoName(t *testing.T) {
	for _, name := range irFiles {
		allowed := []string{`""`, "str"}
		source, err := protojson.Marshal(loadModel(t, name))
		require.NoError(t, err)
		for _, named := range regexp.MustCompile(`"choice":\s*"([^"]+)"`).FindAllStringSubmatch(string(source), -1) {
			allowed = append(allowed, strconv.Quote(named[1]))
		}
		x := exported(t, openNamed(t, name))
		written := regexp.MustCompile(`f_choice: ([^,}]*)`).FindAllStringSubmatch(x.Text, -1)
		require.NotEmpty(t, written, name)
		for _, w := range written {
			require.Contains(t, allowed, w[1], name)
		}
	}
}
