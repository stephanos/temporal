package caseproducer_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	cp "go.temporal.io/server/model/go/caseproducer"
	"go.temporal.io/server/model/go/nexuscaller"
	"go.temporal.io/server/model/go/umpire"
	"google.golang.org/protobuf/testing/protocmp"
)

// keyedPredicates are the Nexus caller's functional Properties as they read a step's keys: what
// model/go/nexuscaller/claims.go says of the typed step. A state's key spells its phase first.
var keyedPredicates = map[string]func(umpire.Result) (bool, error){
	"syncSucceeds":         phaseAndFact("succeeded", "nexusOperationCompleted"),
	"completionSucceeds":   phaseAndFact("", "nexusOperationCompleted"),
	"completionFails":      phaseAndFact("", "nexusOperationFailed"),
	"handlerErrorFails":    phaseAndFact("failed", "nexusOperationFailed"),
	"scheduleToStartFires": phaseAndFact("timedOut", "nexusOperationTimedOut-scheduleToStart"),
	"startToCloseFires":    phaseAndFact("timedOut", "nexusOperationTimedOut-startToClose"),
	"retrySucceeds": func(r umpire.Result) (bool, error) {
		return r.State == "succeeded-1-unset-unset-unset" && slices.Contains(r.Facts, "nexusOperationCompleted"), nil
	},
}

func phaseAndFact(phase, fact string) func(umpire.Result) (bool, error) {
	return func(r umpire.Result) (bool, error) {
		first, _, _ := strings.Cut(r.State, "-")
		return (phase == "" || first == phase) && slices.Contains(r.Facts, fact), nil
	}
}

// keyed is a typed find Query as the same Query over a table that carries only keys: the table's
// rows, state fields and Abstraction Claims, a Property that reads keys, and the pinned schedule.
func keyed(t *testing.T, q *umpire.Query) *umpire.Query {
	t.Helper()
	typed, err := q.Scenario.Machine.Table()
	require.NoError(t, err)
	spec := umpire.TableSpec{Machine: typed.Machine, Owner: typed.Owner, Family: typed.Family, States: typed.States,
		Actions: typed.Actions, Outcomes: typed.Outcomes, Facts: typed.Facts, Starts: typed.Starts, Ends: typed.Ends,
		StateFields: typed.StateFields, Entity: typed.Entity, Evidence: typed.Evidence, RefinedField: nexuscaller.NexusProduct.Name(),
		FieldValues: map[string][]umpire.Atom{}, Claims: typed.Claims()}
	for _, r := range typed.Rows {
		row := umpire.Row{Key: r.Key, Source: r.Source, Action: r.Action}
		for _, res := range r.Results {
			row.Results = append(row.Results, umpire.Result{Outcome: res.Outcome, State: res.State, Facts: res.Facts})
		}
		spec.Rows = append(spec.Rows, row)
	}
	for _, state := range typed.States {
		spec.FieldValues[state] = typed.FieldValues(state)
	}
	table := umpire.NewTable(spec)
	require.NoError(t, table.Err())
	holds, ok := keyedPredicates[q.Property.Name]
	require.True(t, ok, "no keyed reading of %s", q.Property.Name)
	return umpire.KeyFind(q.Name, umpire.KeyProperty(table, q.Property.Name, q.Property.Triggers, "", holds),
		umpire.KeyScenario(table, q.Scenario.Name, q.Scenario.Start, q.Scenario.Actions...), q.Limits)
}

// A Query over a table's keys produces the Case the typed Query produces: the producer reads a table,
// a witness and a Property's clauses, whichever way the Model declared them.
func TestAKeyLevelQueryProducesTheCaseOfTheTypedOne(t *testing.T) {
	realization := nexuscaller.AsyncNexus("umpire.case.service", "complete")
	for _, q := range nexuscaller.FunctionalQueries {
		t.Run(q.Name, func(t *testing.T) {
			identity := cp.IdentityFor("temporal.case", "nexusCallerTests", q.Name)
			want, err := cp.Produce(q, identity, realization, nexuscaller.ModelSource)
			require.NoError(t, err)
			got, err := cp.Produce(keyed(t, q), identity, realization, nexuscaller.ModelSource)
			require.NoError(t, err)
			require.Empty(t, cmp.Diff(want, got, protocmp.Transform()))
		})
	}
}

func instructions(c *testpilotspb.Case, entrypoint string) []string {
	var out []string
	for _, e := range c.GetProgram().GetEntrypoints() {
		for _, n := range e.GetInstructions() {
			if e.GetEntrypointId() == entrypoint {
				out = append(out, n.GetInstructionId())
			}
		}
	}
	return out
}

// A node placed where the path performs a class is carried for a class no binding performs too: the
// backoff timer is the system's step, and a Case whose path takes it carries the node.
func TestANodeIsPlacedForAClassOfThePathNoBindingPerforms(t *testing.T) {
	realization := nexuscaller.AsyncNexus("umpire.case.service", "complete")
	controller := &realization.Plan.Entrypoints[0]
	controller.Items = append(slices.Clone(controller.Items), cp.WhenOnPath{Keys: []string{"backoff"},
		Node: func(cp.Placement, []cp.EvidenceRule) *testpilotspb.InstructionNode {
			return cp.Node("after-backoff", cp.InvokeRPC("temporal.workflow-service",
				"/temporal.api.workflowservice.v1.WorkflowService/DescribeNamespace", nil, nil))
		}})
	carries := func(q *umpire.Query) bool {
		produced, err := cp.Produce(q, cp.IdentityFor("temporal.case", "nexusCallerTests", q.Name), realization, nexuscaller.ModelSource)
		require.NoError(t, err)
		return slices.Contains(instructions(produced, "controller"), "after-backoff")
	}
	require.True(t, carries(nexuscaller.Retry), "the retry path takes the backoff")
	require.False(t, carries(nexuscaller.SyncCompletion), "the synchronous path does not")
}

// Preflight refuses what Produce refuses, in Produce's words, and writes nothing: a Query Produce
// lowers passes it, and one whose path records a fact no evidence source names does not.
func TestPreflightRefusesWhatProduceRefuses(t *testing.T) {
	identity := cp.IdentityFor("temporal.case", "nexusCallerTests", nexuscaller.SyncCompletion.Name)
	sound := nexuscaller.AsyncNexus("umpire.case.service", "complete")
	require.NoError(t, cp.Preflight(nexuscaller.SyncCompletion, identity, sound))

	unsourced := nexuscaller.AsyncNexus("umpire.case.service", "complete")
	unsourced.Sources = slices.DeleteFunc(slices.Clone(unsourced.Sources), func(s *cp.EvidenceSource) bool {
		return s.EventKind == "nexusOperationCompleted"
	})
	_, want := cp.Produce(nexuscaller.SyncCompletion, identity, unsourced, nexuscaller.ModelSource)
	require.EqualError(t, want, "nexusOperationCompleted: evidence.kind-unknown")
	require.Equal(t, want, cp.Preflight(nexuscaller.SyncCompletion, identity, unsourced))
}
