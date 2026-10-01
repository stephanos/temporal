package goir

// The standalone activity Model, lifted from model/scalav2/scala into ir/activity.json and interpreted
// here, against model/go/standaloneactivity, the Go Model it was ported beside. Nothing of the Scala
// code and nothing of the Go step functions is shared: one side is the IR's rows, the other Go's.
//
// The domain compared is what both sides declare: the product, protocol and worker machines, the
// composition of the activity with its worker, the protocol's refinement, every action those bind,
// and every Property, Scenario and Query Go declares on the product and protocol machines and on the
// composition. TestActivityClaimDomainIsWholeOrExcluded reads Go's declarations from its source and
// fails on one that is neither compared nor excluded by name. Outside the domain, and compared nowhere
// here, are what the IR does not carry: the functional, canary and exploratory sets, the product
// machine without controls, the attemptCount observation, which timers are unobservable, and the
// Definition IDs of Properties and Scenarios.

import (
	"cmp"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	"go.temporal.io/server/model/go/standaloneactivity"
	"go.temporal.io/server/model/go/umpire"
	"go.temporal.io/server/model/go/worker"
)

const activityIR = "../ir/activity.json"

var activityBaseline = sync.OnceValues(func() (*modelirspb.Model, error) { return Load(activityIR) })

func activityModel(t *testing.T) *modelirspb.Model {
	t.Helper()
	m, err := activityBaseline()
	require.NoError(t, err)
	return m
}

// tableSide is everything a table says, with a result's typed step left out: Go's is a Go value and
// the IR's an IR value, and each is already spelled by the result's keys. Evidence is its lines as a
// set; TestActivityEvidenceIsInCatalogOrder is about their order.
type tableSide struct {
	Machine, Owner, Entity, Stuck               string
	Family                                      umpire.Family
	States, Actions, Outcomes, Facts            []string
	Starts, Ends, Reachable, StateFields        []string
	Rows                                        []umpire.Row
	Disabled                                    []string
	Unknown                                     int
	Evidence                                    [][2]string
	IDs                                         umpire.IDs
	Fingerprint                                 string
	Assumptions                                 []umpire.Assumption
	RowResults, DisabledPairs, StatesTimesClass int
}

// sideOf reads a table as a tableSide. A pair is disabled when the table has no row for it; the rows
// of a table are states-major, so the pairs are listed in that order too.
func sideOf(t *umpire.Table) tableSide {
	side := tableSide{Machine: t.Machine, Owner: t.OwnerName(), Entity: t.Entity, Stuck: t.Stuck, Family: t.Family,
		States: t.States, Actions: t.Actions, Outcomes: t.Outcomes, Facts: t.Facts, Starts: t.Starts, Ends: t.Ends,
		Reachable: t.Reachable, StateFields: t.StateFields, Unknown: len(t.Unknown), Evidence: [][2]string{},
		IDs: t.IDs(), Fingerprint: t.TargetFingerprint(), Assumptions: t.Assumptions,
		StatesTimesClass: len(t.States) * len(t.Actions)}
	side.Evidence = append(side.Evidence, t.Evidence...)
	slices.SortFunc(side.Evidence, func(a, b [2]string) int { return cmp.Compare(a[0], b[0]) })
	enabled := map[string]bool{}
	for _, row := range t.Rows {
		enabled[row.Key] = true
		plain := umpire.Row{Key: row.Key, Source: row.Source, Action: row.Action}
		for _, res := range row.Results {
			res.Step = nil
			plain.Results = append(plain.Results, res)
			side.RowResults++
		}
		side.Rows = append(side.Rows, plain)
	}
	for _, s := range t.States {
		for _, a := range t.Actions {
			if key := s + "-" + a; !enabled[key] {
				side.Disabled = append(side.Disabled, key)
			}
		}
	}
	side.DisabledPairs = len(side.Disabled)
	return side
}

func goTable(t *testing.T, m umpire.Model) *umpire.Table {
	t.Helper()
	table, err := m.Table()
	require.NoError(t, err)
	return table
}

// Every state, class, outcome, fact, start, end, row, result and explanation of the lifted machines
// is Go's, and so is every pair with no row: the IR disables exactly what Go disables, and leaves no
// pair unknown.
func TestActivityTablesEqualTheGoModel(t *testing.T) {
	machines := built(t, activityModel(t))
	for name, want := range map[string]umpire.Model{
		"activityProduct":  standaloneactivity.ActivityProduct,
		"activityProtocol": standaloneactivity.ActivityProtocol,
		"activityWorker":   standaloneactivity.ActivityWorker,
		"polling":          worker.PollingMachine,
	} {
		t.Run(name, func(t *testing.T) {
			mm := machines[name]
			require.NotNil(t, mm)
			require.Empty(t, mm.Holes)
			require.NoError(t, mm.Rejected)
			got := sideOf(mm.Table)
			require.Equal(t, sideOf(goTable(t, want)), got)
			// The interpreter's own account of a disabled pair agrees with the rows.
			for _, s := range mm.Table.States {
				for _, c := range mm.Classes {
					require.Equal(t, !hasRow(mm, s+"-"+c.Key), mm.Disabled(s, c.Key), "%s-%s", s, c.Key)
				}
			}
			require.Equal(t, got.StatesTimesClass, got.DisabledPairs+len(got.Rows))
			t.Logf("%s: %d states, %d classes, %d rows with %d results, %d disabled pairs, %d reachable states",
				name, len(got.States), len(got.Actions), len(got.Rows), got.RowResults, got.DisabledPairs, len(got.Reachable))
		})
	}
}

// What the product machine disables is what the baseline pins by hand: a canceled answer without a
// cancel request, a worker stop, and a start of a paused activity.
func TestActivityDisabledBehaviorIsTheBaselines(t *testing.T) {
	product := built(t, activityModel(t))["activityProduct"]
	for _, pair := range [][2]string{
		{"started", "attemptResult-canceled"},
		{"paused", "attemptStart"},
		{"scheduled", "workerStop"},
		{"completed", "timeout"},
	} {
		require.True(t, product.Disabled(pair[0], pair[1]), pair)
	}
	require.False(t, product.Disabled("completed", "control-terminate"), "a control of an activity that is over is answered notFound, not disabled")
	require.Equal(t, []umpire.Result{{Outcome: "notFound", State: "completed", Facts: []string{}}},
		sideOf(product.Table).Rows[rowIndex(t, product.Table, "completed-control-terminate")].Results)
}

// The protocol machine's evidence lines are Go's, in another order: the IR lists a machine's evidence
// in the catalog order of its facts (SEMANTICS.md, Machines 5), where Go lists it as declared, which
// puts attemptCount first. No Definition ID and no fingerprint reads the order: both are compared
// above and are equal.
func TestActivityEvidenceIsInCatalogOrder(t *testing.T) {
	declared := goTable(t, standaloneactivity.ActivityProtocol).Evidence
	require.Equal(t, [2]string{"attemptCount", "attemptCount"}, declared[0])
	require.Equal(t, append(slices.Clone(declared[1:]), declared[0]), built(t, activityModel(t))["activityProtocol"].Table.Evidence)
	product := goTable(t, standaloneactivity.ActivityProduct).Evidence
	require.Equal(t, product, built(t, activityModel(t))["activityProduct"].Table.Evidence)
}

func rowIndex(t *testing.T, table *umpire.Table, key string) int {
	t.Helper()
	for i, row := range table.Rows {
		if row.Key == key {
			return i
		}
	}
	require.Failf(t, "no row", "no row %s in %s", key, table.Machine)
	return -1
}

func TestActivityCompositionEqualsTheGoModel(t *testing.T) {
	got := composedTable(t, activityModel(t), "standaloneActivity")
	require.Equal(t, sideOf(goTable(t, standaloneactivity.StandaloneActivity)), sideOf(got))
}

func TestActivityRefinementEqualsTheGoModel(t *testing.T) {
	want, err := standaloneactivity.ActivityProtocol.Refinement()
	require.NoError(t, err)
	protocol := built(t, activityModel(t))["activityProtocol"]
	require.NoError(t, protocol.Rejected)
	require.Equal(t, want.Rows, protocol.Refinement)
}

// actionSide is an action's declaration as both sides carry it.
type actionSide struct {
	Name, Party, On, Creates, Results string
	Timer                             bool
	Schemas, Inputs, Examples         []string
}

func TestActivityActionsEqualTheGoModel(t *testing.T) {
	declared := map[string]actionSide{}
	for _, a := range activityModel(t).GetActions() {
		side := actionSide{Name: a.GetName(), Party: a.GetParty(), On: a.GetOn(), Creates: a.GetCreates(),
			Results: a.GetResults(), Timer: a.GetTimer(), Schemas: a.GetSchemas()}
		for _, in := range a.GetInputs() {
			side.Inputs = append(side.Inputs, in.GetName())
		}
		for _, e := range a.GetExamples() {
			side.Examples = append(side.Examples, e.GetExample())
		}
		declared[a.GetName()] = side
	}
	// workerResume is declared by the worker Model and bound by `polling` alone.
	names := []string{"start", "attemptStart", "attemptResult", "control", "workerStop", "timeout", "backoff",
		"scheduleToClose", "scheduleToStart", "startToClose", "serve", "workerResume"}
	require.Len(t, declared, len(names))
	tables := []*umpire.Table{goTable(t, standaloneactivity.ActivityProduct), goTable(t, standaloneactivity.ActivityProtocol),
		goTable(t, worker.PollingMachine)}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			var decl *umpire.ActionDecl
			for _, table := range tables {
				if d, ok := table.Decl(name); ok {
					decl = d
				}
			}
			require.NotNil(t, decl)
			// Every action of the baseline that the system performs is a timer.
			want := actionSide{Name: decl.Name, Party: string(decl.Party), Results: decl.Results, Schemas: decl.Schemas,
				Inputs: decl.Inputs, Timer: decl.Party == umpire.System}
			if decl.On != nil {
				want.On = decl.On.Name
			}
			if decl.Creates != nil {
				want.Creates = decl.Creates.Name
			}
			for _, e := range decl.Examples {
				want.Examples = append(want.Examples, e.Example)
			}
			require.Equal(t, want, declared[name])
		})
	}
}

// cancelRequest asks the one Property of the protocol machine that no Query of the baseline asks,
// cancelRequestedWhileStarted, over the baseline's own path through a cancel request. Go declares the
// Property and the path and no Query that pairs them, so the pairing is made here from Go's
// declarations, and Claims.scala declares the same Query for the IR to carry.
var cancelRequest = sync.OnceValue(func() *umpire.Query {
	return standaloneactivity.CancelRequestedThenCanceled.Find("cancelRequest",
		standaloneactivity.CancelRequestedWhileStarted, standaloneactivity.Four)
})

// comparedQueries is every Query answered on both sides: the ten Go declares over the protocol
// machine, cancelRequest, and the cross-entity Query over the composition.
func comparedQueries() []*umpire.Query {
	return append(append([]*umpire.Query{}, standaloneactivity.FunctionalQueries...),
		standaloneactivity.TerminalHolds, standaloneactivity.PauseHolds, cancelRequest(),
		standaloneactivity.StoppedWorkerStartsNothing)
}

// queryReceipt is the key of the receipt that answers a Query of the baseline: a Query belongs to the
// machine or composition its Scenario runs on.
func queryReceipt(q *umpire.Query) string {
	return "query " + q.Scenario.Machine.Name() + " " + q.Name
}

// querySide is what a Query pairs and what its answer says, on both sides.
type querySide struct {
	Property, Scenario string
	Outcome            string
	Explored, Expanded int
	Exercised          bool
	Rows               []string
	Witness            *umpire.Trace
}

// Every Query of the baseline is answered through the lifted IR as Go answers it: the same outcome,
// the same witness with the same Definition IDs, over the same number of product states.
func TestActivityClaimsEqualTheGoModel(t *testing.T) {
	report := checked(t, activityModel(t))
	require.Empty(t, report.Unsupported())
	want := map[string]ReceiptKind{"refinement activityProtocol activityProduct": Verified}
	for _, q := range comparedQueries() {
		key := queryReceipt(q)
		answer, err := q.Answer()
		require.NoError(t, err)
		want[key] = ReceiptKind(answer.Outcome)
		got := receiptOf(t, report, key)
		require.Equal(t,
			querySide{Property: q.Property.Name, Scenario: q.Scenario.Name, Outcome: string(answer.Outcome),
				Explored: answer.Explored, Expanded: answer.Expanded, Exercised: answer.Exercised, Rows: answer.Rows,
				Witness: answer.Witness},
			querySide{Property: got.Property.Name, Scenario: got.Scenario.Name, Outcome: string(got.Kind),
				Explored: got.Explored, Expanded: got.Expanded, Exercised: got.Exercised, Rows: got.Rows,
				Witness: got.Witness}, key)
		require.Empty(t, got.Holes, key)
	}
	// The report holds these and nothing else, and they are the baseline's pinned answers, eight
	// found and three verified, beside the refinement and the Query that asks cancelRequestedWhileStarted.
	require.Equal(t, want, kinds(report))
	found, verified := 0, 0
	for _, kind := range want {
		switch kind {
		case Found:
			found++
		case Verified:
			verified++
		default:
		}
	}
	require.Equal(t, [2]int{9, 4}, [2]int{found, verified})
}

// The baseline's cross-entity Query, stoppedWorkerStartsNothing, asks a Property of the composition
// that is about one action of it, the synchronized attempt start. The lifted Model carries the claim
// as Go declares it: the same selector, the same path, key for key, and an answer that read the claim
// on a step, so the verification is not one for want of a firing.
func TestActivityCrossEntityClaimIsCompared(t *testing.T) {
	baseline := standaloneactivity.StoppedWorkerStartsNothing
	answer, err := baseline.Answer()
	require.NoError(t, err)
	require.Equal(t, umpire.VerifiedWithinLimits, answer.Outcome)
	require.True(t, answer.Exercised)

	m := activityModel(t)
	started := admProperty(m, "standaloneActivity", "startedByPollingWorker")
	require.NotNil(t, started)
	require.Equal(t, "attemptStart", started.GetWhenAction())
	require.False(t, started.GetTransition())
	path := admScenario(m, "standaloneActivity", "stoppedBeforeRetry")
	require.NotNil(t, path)
	require.Equal(t, baseline.Scenario.Actions, path.GetKeys())
	require.Empty(t, path.GetActions())

	got := receiptOf(t, checked(t, m), queryReceipt(baseline))
	table := goTable(t, standaloneactivity.StandaloneActivity)
	composed := ClaimKey{Family: string(table.Family), Owner: "standaloneActivity"}
	require.Equal(t, []any{Verified, true, answer.Explored, answer.Expanded},
		[]any{got.Kind, got.Exercised, got.Explored, got.Expanded})
	composed.Name = "startedByPollingWorker"
	require.Equal(t, composed, got.Property)
	composed.Name = "stoppedBeforeRetry"
	require.Equal(t, composed, got.Scenario)
	require.Equal(t, []any{table.Family.Target(table.OwnerName()), table.TargetFingerprint()}, []any{got.Target, got.Fingerprint})
	require.Empty(t, got.Assumptions)
	require.Empty(t, got.Holes)
}

// excludedClaims is what Go declares and the comparison leaves out, each with why. It is empty: every
// Property, Scenario and Query of the baseline is compared. A declaration that must be left out is
// named here with its reason, and TestActivityClaimDomainIsWholeOrExcluded fails on one that is not.
var excludedClaims = map[string]string{}

// goDeclaredClaims reads model/go/standaloneactivity's source for every Property, Scenario and Query
// it declares by a literal name, as "<kind> <name>". indirect counts the declarations whose name is
// no literal, which this reading cannot follow: the Scenario inside the `path` helper, whose callers
// are read instead.
func goDeclaredClaims(t *testing.T) (declared []string, indirect int) {
	t.Helper()
	dir := filepath.Join("..", "..", "go", "standaloneactivity")
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	require.NoError(t, err)
	kinds := map[string]string{"Property": "property", "Scenario": "scenario", "path": "scenario",
		"Find": "query", "Verify": "query", "VerifyRefined": "query"}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		require.NoError(t, err)
		ast.Inspect(parsed, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			var callee string
			switch fn := call.Fun.(type) {
			case *ast.SelectorExpr:
				callee = fn.Sel.Name
			case *ast.Ident:
				callee = fn.Name
			default:
			}
			kind, ok := kinds[callee]
			if !ok {
				return true
			}
			if name, ok := call.Args[0].(*ast.BasicLit); ok && name.Kind == token.STRING {
				unquoted, err := strconv.Unquote(name.Value)
				require.NoError(t, err)
				declared = append(declared, kind+" "+unquoted)
			} else {
				indirect++
			}
			return true
		})
	}
	return declared, indirect
}

// Every Property, Scenario and Query the Go baseline declares is either compared above or excluded
// by name with its reason, and the lifted Model carries exactly the compared ones. A declaration
// added to either side, or left out of the lift, fails here rather than passing unseen.
func TestActivityClaimDomainIsWholeOrExcluded(t *testing.T) {
	declared, indirect := goDeclaredClaims(t)
	require.Equal(t, 1, indirect, "a declaration this test cannot read by name")

	compared := map[string]bool{}
	for _, q := range comparedQueries() {
		compared["query "+q.Name] = true
		compared["property "+q.Property.Name] = true
		compared["scenario "+q.Scenario.Name] = true
	}
	var unaccounted, both []string
	seen := map[string]bool{}
	for _, claim := range declared {
		seen[claim] = true
		_, excluded := excludedClaims[claim]
		switch {
		case compared[claim] && excluded:
			both = append(both, claim)
		case !compared[claim] && !excluded:
			unaccounted = append(unaccounted, claim)
		default:
		}
	}
	require.Empty(t, unaccounted, "declared by Go, and neither compared nor excluded")
	require.Empty(t, both, "compared and excluded")
	for claim := range excludedClaims {
		require.True(t, seen[claim], "%s is excluded and Go does not declare it", claim)
	}
	// Only the pairing made here is compared without a declaration of Go's.
	var undeclared []string
	for claim := range compared {
		if !seen[claim] {
			undeclared = append(undeclared, claim)
		}
	}
	require.Equal(t, []string{"query cancelRequest"}, undeclared)

	// The lifted Model carries each compared claim and no other.
	m := activityModel(t)
	lifted := map[string]bool{}
	for _, p := range m.GetProperties() {
		lifted["property "+p.GetName()] = true
	}
	for _, s := range m.GetScenarios() {
		lifted["scenario "+s.GetName()] = true
	}
	for _, q := range m.GetQueries() {
		lifted["query "+q.GetName()] = true
	}
	require.Equal(t, compared, lifted)
	require.Len(t, declared, 10+1+8+1+10+1, "ten Properties, eight Scenarios and ten Queries of the machines, and the cross-entity claim's three")
}
