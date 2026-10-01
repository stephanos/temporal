package goir

// What the close and reset designs share with model/go/nexuscaller, the Go baseline, and what they do
// not. The baseline models one run and neither a close nor a reset, so the one behavior both have is
// an open caller accepting an asynchronous completion: the baseline's `complete` rows of a started
// operation. That is compared here, row by row and Query by Query. Everything else the designs claim
// is an authored design promise with no baseline to agree with, and the inventories below fail on a
// claim of either side that is neither compared nor listed as what it is.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	"go.temporal.io/server/model/go/nexuscaller"
	"go.temporal.io/server/model/go/umpire"
)

// closeAccepted is the result of the first delivery of a completion to an open caller, as both sides
// spell it: the answer, and the history event that records the outcome.
type closeAccepted struct {
	Outcome string
	Facts   []string
}

// An open caller accepts each completion and its history records the outcome by the event the
// baseline names, on every design: the policies differ only once the caller closed or was reset.
func TestNexusCloseOpenCallerAcceptsACompletionAsTheGoModel(t *testing.T) {
	c := closeModel(t)
	product, protocol := goTable(t, nexuscaller.NexusProduct), goTable(t, nexuscaller.NexusProtocol)
	for resolution, from := range map[string]string{
		"succeeded": "open-none-done-succeeded-inFlight-succeeded-none-none",
		"failed":    "open-none-done-failed-inFlight-failed-none-none",
		// A handler cancels only what it was asked to cancel.
		"canceled": "open-requested-callerWorkflow-done-canceled-inFlight-canceled-none-none",
	} {
		class := "complete-" + resolution
		baseline := plainResults(t, product, "started-"+class)
		require.Len(t, baseline, 1, class)
		want := closeAccepted{baseline[0].Outcome, baseline[0].Facts}
		detailed := plainResults(t, protocol, "started-0-unset-unset-unset-"+class)
		require.Equal(t, want, closeAccepted{detailed[0].Outcome, detailed[0].Facts}, class)
		for design, mm := range c.built {
			first := plainResults(t, mm.Table, from+"-"+class)[0]
			require.Equal(t, want, closeAccepted{first.Outcome, first.Facts}, "%s %s", design, class)
			require.Empty(t, first.Because, "%s %s", design, class)
		}
	}
}

// The designs deliver the baseline's own `complete` action: one declaration, by the handler, on the
// operation, with the baseline's input and result domain.
func TestNexusCloseCompleteIsTheBaselinesAction(t *testing.T) {
	decl, ok := goTable(t, nexuscaller.NexusProduct).Decl("complete")
	require.True(t, ok)
	var lifted []actionSide
	for _, a := range closeModel(t).model.GetActions() {
		if a.GetName() != "complete" {
			continue
		}
		side := actionSide{Name: a.GetName(), Party: a.GetParty(), On: a.GetOn(), Results: a.GetResults()}
		for _, in := range a.GetInputs() {
			side.Inputs = append(side.Inputs, in.GetName())
		}
		lifted = append(lifted, side)
	}
	require.Equal(t, []actionSide{{Name: decl.Name, Party: string(decl.Party), On: decl.On.Name, Results: decl.Results,
		Inputs: decl.Inputs}}, lifted)
}

// closeBaselineQueries is the baseline's Queries the designs also declare, by the design's name for it.
var closeBaselineQueries = map[string]*umpire.Query{
	"asyncCompletion": nexuscaller.AsyncCompletion,
	"asyncFailure":    nexuscaller.AsyncFailure,
}

// closeStep is a witness's last step without what differs by construction: the state, which each side
// spells in its own fields.
type closeStep struct {
	Property, Outcome, Action, Answer string
	Facts                             []string
}

// The two Queries of the baseline that find a completion recorded are found on every design with the
// baseline's Property, by the same last step. The baseline's path schedules the operation and has the
// handler reply first; a design's begins at a started operation, so only the delivery is compared.
func TestNexusCloseCompletionClaimsEqualTheGoModel(t *testing.T) {
	c := closeModel(t)
	for name, q := range closeBaselineQueries {
		answer, err := q.Answer()
		require.NoError(t, err)
		final := last(t, answer.Witness)
		want := closeStep{q.Property.Name, string(answer.Outcome), final.Action.Value, final.Outcome.Value, factsOf(final)}
		for _, design := range closeDesigns {
			got := closeQuery(t, c, design, name)
			final := last(t, got.Witness)
			require.Equal(t, want, closeStep{got.Property.Name, string(got.Kind), final.Action.Value, final.Outcome.Value,
				factsOf(final)}, "%s %s", design, name)
		}
	}
}

// A difference to judge, pinned so it is not lost: the baseline answers a completion that arrives after
// the operation is over `notFound`, and the specimen's designs answer the duplicate of a lost
// acknowledgment `accepted` (N5). The designs do reject a completion that arrives after the deadline,
// under their own answer.
func TestNexusCloseLateCompletionDiffersFromTheGoModel(t *testing.T) {
	c := closeModel(t)
	baseline := plainResults(t, goTable(t, nexuscaller.NexusProduct), "succeeded-complete-succeeded")
	require.Equal(t, []umpire.Result{{Outcome: "notFound", State: "succeeded", Facts: []string{}}}, baseline)
	duplicate := plainResults(t, c.built["retainAndRoute"].Table,
		"open-none-done-succeeded-inFlight-succeeded-none-original-succeeded-complete-succeeded")
	require.Equal(t, "accepted", duplicate[0].Outcome)
	late := plainResults(t, goTable(t, nexuscaller.NexusProduct), "timedOut-complete-succeeded")
	require.Equal(t, []umpire.Result{{Outcome: "notFound", State: "timedOut", Facts: []string{}}}, late)
	require.Equal(t, []umpire.Result{{Outcome: "rejectedPermanent", State: "open-none-done-succeeded-none-none-expired",
		Facts: []string{"completionDropped"}}},
		plainResults(t, c.built["retainAndRouteWithDeadline"].Table,
			"open-none-done-succeeded-inFlight-succeeded-none-expired-complete-succeeded"))
}

// closeBaselineClaims is what the baseline declares and the designs are compared on.
var closeBaselineClaims = []string{"property completionSucceeds", "property completionFails", "query asyncCompletion",
	"query asyncFailure"}

// closeExcluded is every other claim of the baseline, each with why no design is compared on it.
var closeExcluded = map[string]string{
	"scenario asyncThenSucceeded": "schedules and replies before the delivery; a design starts at a started operation, and the delivery is compared",
	"scenario asyncThenFailed":    "schedules and replies before the delivery; a design starts at a started operation, and the delivery is compared",

	"property terminalIsFinal": "about the phases of one run; a design's knowledge moves to the successor at a reset",
	"query terminalHolds":      "asks terminalIsFinal",

	"property syncSucceeds":         "the handler's reply to the start request, which no design models",
	"property asyncStarts":          "the handler's reply to the start request, which no design models",
	"property handlerErrorFails":    "the handler's reply to the start request, which no design models",
	"property retrySucceeds":        "the start request's retry, which no design models",
	"scenario syncReplied":          "the handler's reply to the start request, which no design models",
	"scenario nonRetryableError":    "the handler's reply to the start request, which no design models",
	"scenario retriedThenSucceeded": "the start request's retry, which no design models",
	"query syncCompletion":          "the handler's reply to the start request, which no design models",
	"query handlerError":            "the handler's reply to the start request, which no design models",
	"query retry":                   "the start request's retry, which no design models",

	"property scheduleToStartFires":   "a deadline of the start request; the designs' one deadline is schedule-to-close, which the baseline claims nothing of",
	"property startToCloseFires":      "a deadline no design models",
	"scenario scheduleToStartExpires": "a deadline of the start request",
	"scenario startToCloseExpires":    "a deadline no design models",
	"query scheduleToStartTimeout":    "a deadline of the start request",
	"query startToCloseTimeout":       "a deadline no design models",

	"property repliedByPollingWorker":   "about the handler's worker, which no design models",
	"scenario repliedThenStopped":       "about the handler's worker, which no design models",
	"query stoppedWorkerRepliesNothing": "about the handler's worker, which no design models",
}

// goClaims reads a Go Model's source for every Property, Scenario and Query it declares by a literal
// name, as "<kind> <name>", and counts the declarations whose name is no literal.
func goClaims(t *testing.T, dir string) (declared []string, indirect int) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	require.NoError(t, err)
	kinds := map[string]string{"Property": "property", "Scenario": "scenario", "Find": "query", "Verify": "query",
		"VerifyRefined": "query"}
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
			fn, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			kind, ok := kinds[fn.Sel.Name]
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
// by name with its reason. A claim added to the baseline fails here rather than passing unseen.
func TestNexusCloseBaselineClaimsAreComparedOrExcluded(t *testing.T) {
	declared, indirect := goClaims(t, filepath.Join("..", "..", "go", "nexuscaller"))
	require.Zero(t, indirect, "a declaration this test cannot read by name")
	accounted := slices.Concat(closeBaselineClaims, slices.Collect(maps.Keys(closeExcluded)))
	require.ElementsMatch(t, declared, accounted)
	require.Len(t, declared, len(closeBaselineClaims)+len(closeExcluded), "a claim is compared or excluded, not both")
	for name, q := range closeBaselineQueries {
		require.Contains(t, closeBaselineClaims, "query "+q.Name)
		require.Equal(t, name, q.Name)
		require.Contains(t, closeBaselineClaims, "property "+q.Property.Name)
	}
}

// closePromises labels every Property the designs declare that the baseline does not: each is an
// authored design promise, by the oracle of specimens/nexus.md it states.
var closePromises = map[string]string{
	"outcomePreserved":            "promise 1: a decided outcome stays where an owner can learn it",
	"ackOnlyWhenKept":             "promise 2: an acknowledgment implies the outcome is kept",
	"closedHistoryIsFrozen":       "Close and reset: a closed run's history never changes",
	"handlerEffectIsIrreversible": "Close and reset: a reset cannot undo a handler effect",
	"knownIsTheHandlersOutcome":   "N5: the channel carries the handler's one result",
	"knowledgeIsFinal":            "N4, N5: a recorded outcome stays recorded, across a reset too",
	"finishesAfterClose":          "Close and reset: detached handler work continues after close",
	"requestedButUnreceived":      "the cancel intent apart from its delivery",
	"receivedButSucceeded":        "the cancel's receipt apart from the handler's effect",
	"canceledButUnknown":          "the handler's effect apart from the caller's knowledge",
	"completionCancels":           "the caller's knowledge of a canceled outcome",
	"rejectedTransiently":         "N1 with a transient rejection",
	"rejectedPermanently":         "N1, step 3",
	"lostAfterReset":              "N1, step 4",
	"reappliesRetained":           "N1', N6: reset after retention",
	"routedToSuccessor":           "N2', N6: reset before retention",
	"expiresWithNothingOwed":      "N8: the timeout that resolves a lost outcome",
	"noUnnecessaryWait":           "N8: a deadline fires only while the outcome is undecided or still owed",
	"expiresWhileOwed":            "N8: a deadline that beats the report is no loss",
	"lateCompletionIsDropped":     "N8: a completion after the deadline",
}

var closeDeclared = map[string]*regexp.Regexp{
	"machine":    regexp.MustCompile(`machine\[[^\]]+\]\(\s*Family,\s*"([^"]+)"\s*\)`),
	"monitor":    regexp.MustCompile(`monitor\[[^\]]+\]\(\s*"([^"]+)"`),
	"assumption": regexp.MustCompile(`assume\(\s*"([^"]+)"\s*\)`),
	"progress":   regexp.MustCompile(`\.leadsTo\(\s*"([^"]+)"\s*\)`),
	"property":   regexp.MustCompile(`\.property\(\s*"([^"]+)"\s*\)`),
	"scenario":   regexp.MustCompile(`\.scenario\(\s*"([^"]+)"\s*\)`),
	"query":      regexp.MustCompile(`query\(\s*s"\$\{m\.name\}\.([^"]+)"\s*\)`),
}

// closeSource is every name the Scala sources of the designs declare, as "<kind> <name>".
func closeSource(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "scala", "temporal", "nexuscaller", "closepolicy", "*.scala"))
	require.NoError(t, err)
	require.NotEmpty(t, files)
	found := map[string]bool{}
	for _, file := range files {
		source, err := os.ReadFile(file)
		require.NoError(t, err)
		for kind, declares := range closeDeclared {
			for _, match := range declares.FindAllStringSubmatch(string(source), -1) {
				found[kind+" "+match[1]] = true
			}
		}
	}
	return slices.Sorted(maps.Keys(found))
}

// closeLifted is every name the lifted Model declares, a Query by its name without its machine's.
func closeLifted(m *modelirspb.Model) []string {
	found := map[string]bool{}
	for _, x := range m.GetMachines() {
		found["machine "+x.GetName()] = true
	}
	for _, x := range m.GetMonitors() {
		found["monitor "+x.GetName()] = true
	}
	for _, x := range m.GetAssumptions() {
		found["assumption "+x.GetName()] = true
	}
	for _, x := range m.GetProgress() {
		found["progress "+x.GetName()] = true
	}
	for _, x := range m.GetProperties() {
		found["property "+x.GetName()] = true
	}
	for _, x := range m.GetScenarios() {
		found["scenario "+x.GetName()] = true
	}
	for _, x := range m.GetQueries() {
		_, name, _ := strings.Cut(x.GetName(), ".")
		found["query "+name] = true
	}
	return slices.Sorted(maps.Keys(found))
}

// Nothing the designs declare is left unlifted or unasked: every machine, monitor, assumption,
// progress claim, Property, Scenario and Query of the Scala sources is in the lifted Model and no
// other is; every Property and Scenario is asked by a Query, every monitor watches every machine,
// every assumption is named by a result; and every Property is the baseline's or a labeled promise.
func TestNexusCloseEveryDeclarationIsLiftedAndAsked(t *testing.T) {
	c := closeModel(t)
	require.Equal(t, closeSource(t), closeLifted(c.model))

	asked := map[string]bool{}
	for _, q := range c.model.GetQueries() {
		asked["property "+q.GetProperty().GetMachine()+" "+q.GetProperty().GetName()] = true
		asked["scenario "+q.GetScenario().GetMachine()+" "+q.GetScenario().GetName()] = true
	}
	labeled := map[string]bool{}
	for _, p := range c.model.GetProperties() {
		require.True(t, asked["property "+p.GetMachine()+" "+p.GetName()], "no Query asks %s of %s", p.GetName(), p.GetMachine())
		_, promise := closePromises[p.GetName()]
		baseline := slices.Contains(closeBaselineClaims, "property "+p.GetName())
		require.NotEqual(t, promise, baseline, "%s is the baseline's or a design promise, and one of them", p.GetName())
		labeled[p.GetName()] = true
	}
	for name := range closePromises {
		require.True(t, labeled[name], "%s is labeled and no design declares it", name)
	}
	for _, s := range c.model.GetScenarios() {
		require.True(t, asked["scenario "+s.GetMachine()+" "+s.GetName()], "no Query runs %s of %s", s.GetName(), s.GetMachine())
	}

	var monitors []string
	for _, m := range c.model.GetMonitors() {
		monitors = append(monitors, m.GetId())
	}
	require.Len(t, monitors, 4)
	answered := map[string]bool{}
	relied := map[string]bool{}
	for _, r := range c.report.Receipts {
		answered[r.Key.Owner] = true
		for _, a := range r.Assumptions {
			relied[a] = true
		}
	}
	for _, m := range c.model.GetMachines() {
		require.ElementsMatch(t, monitors, m.GetMonitors(), m.GetName())
		require.True(t, answered[m.GetName()], m.GetName())
		require.NotEqual(t, "operation", m.GetEntity(), "%s is keyed by the run's scheduled event", m.GetName())
	}
	for _, a := range c.model.GetAssumptions() {
		require.True(t, relied[a.GetName()], "no result names %s", a.GetName())
	}
	require.Len(t, c.report.Receipts, len(c.model.GetQueries())+3*len(c.model.GetProgress()))
}

// A design records an outcome and the deadline by the history events the baseline's product machine
// names, and every fact of a design has its evidence line.
func TestNexusCloseEvidenceNamesTheBaselinesEvents(t *testing.T) {
	c := closeModel(t)
	shared := 0
	design := c.built["retainAndRouteWithDeadline"].Table.Evidence
	for _, line := range goTable(t, nexuscaller.NexusProduct).Evidence {
		if slices.Contains([]string{"nexusOperationCompleted", "nexusOperationFailed", "nexusOperationCanceled",
			"nexusOperationTimedOut"}, line[0]) {
			shared++
			require.Contains(t, design, line)
		}
	}
	require.Equal(t, 4, shared)
	want := [][2]string{{"workflowClosed", "workflowClosed"}, {"workflowReset", "workflowReset"},
		{"cancelRequested", "nexusOperationCancelRequested"}, {"cancelReceived", "cancelReceived"},
		{"handlerFinished", "handlerFinished"}, {"nexusOperationCompleted", "nexusOperationCompleted"},
		{"nexusOperationFailed", "nexusOperationFailed"}, {"nexusOperationCanceled", "nexusOperationCanceled"},
		{"nexusOperationTimedOut", "nexusOperationTimedOut"}, {"outcomeRetained", "outcomeRetained"},
		{"outcomeReapplied", "outcomeReapplied"}, {"completionDropped", "completionDropped"}}
	for name, mm := range c.built {
		require.Equal(t, want, mm.Table.Evidence, name)
	}
}

// closeClaim is a progress claim as the lifted Model declares it.
type closeClaim struct {
	From, To    string
	Within      int32
	Assumptions []string
}

// The progress claims are the specimen's: from a decided outcome to a state a path may end in within
// six steps, and from an outcome retained for a closed run to an owner that knows it. Each names the
// assumptions it is conditional on, and an assumption makes fair only what its name says.
func TestNexusCloseProgressClaimsAreTheSpecimens(t *testing.T) {
	m := closeModel(t).model
	const model, claims = "temporal.nexuscaller.closepolicy.Model$package$.", "temporal.nexuscaller.closepolicy.Claims$package$."
	names := map[string]string{}
	fair := map[string][]string{}
	for _, a := range m.GetAssumptions() {
		names[a.GetId()] = a.GetName()
		fair[a.GetName()] = a.GetFair()
	}
	const reporting, delivery, recovery = "handlerReportsUntilAckOrPermanent", "enabledDeliveryAndRecoveryActionsEventuallyRun",
		"currentOwnerEventuallyRecoversAndReappliesRetainedOutcome"
	require.Equal(t, map[string][]string{
		reporting:                              nil,
		delivery:                               {"temporal.nexuscaller.Model$package$.complete"},
		recovery:                               {model + "reset"},
		"transientRejectionEventuallyAccepted": nil,
		"retentionSurvivesCrash":               nil,
		"scheduleToCloseExpires":               nil,
	}, fair)

	reaches := closeClaim{model + "isDone", model + "settled", 6, []string{reporting, delivery}}
	retained := closeClaim{claims + "awaitingOwner", claims + "ownerKnowsOutcome", 2, []string{delivery, recovery}}
	unrecovered := retained
	unrecovered.Assumptions = []string{delivery}
	want := map[string]closeClaim{
		"retainAndRoute retainedReachesOwner":         retained,
		"retainAndRoute retainedWaitsWithoutRecovery": unrecovered,
		closeBounded + " retainedReachesOwner":        retained,
	}
	for _, design := range slices.Concat([]string{"rejectAfterClose", "ackByOriginal", "retainAndRoute", closeBounded}, closeDeadlineDesigns) {
		want[design+" outcomeReachesOwner"] = reaches
	}
	got := map[string]closeClaim{}
	for _, p := range m.GetProgress() {
		claim := closeClaim{From: p.GetFrom(), To: p.GetTo(), Within: p.GetWithin()}
		for _, a := range p.GetAssumptions() {
			claim.Assumptions = append(claim.Assumptions, names[a])
		}
		got[p.GetMachine()+" "+p.GetName()] = claim
	}
	require.Equal(t, want, got)
}
