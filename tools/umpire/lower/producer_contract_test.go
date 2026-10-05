package lower

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
)

// producerContractFixture is what the producer is handed for a Query of the Nexus caller Model once
// the lowerer has checked it: the Query, its identity, the realization and the source.
func producerContractFixture(t *testing.T, name string) (*umpiremodel.Query, Identity, *cp.Realization, cp.Source) {
	t.Helper()
	p, err := NewProducer(loaded(t, "nexus-caller"))
	require.NoError(t, err)
	a, standing, err := p.ask(name)
	require.NoError(t, err)
	require.Equal(t, Lowered, standing)
	identity := nexusIdentity(name)
	l, problems := p.check(a, identity)
	require.Empty(t, problems)
	return l.query, identity, l.realization, p.source(a.q)
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
	carries := func(name string) bool {
		query, identity, realization, source := producerContractFixture(t, name)
		controller := &realization.Plan.Entrypoints[0]
		controller.Items = append(slices.Clone(controller.Items), cp.WhenOnPath{Keys: []string{"backoff"},
			Node: func(cp.Placement, []cp.EvidenceRule) *testpilotspb.InstructionNode {
				return cp.Node("after-backoff", cp.InvokeRPC("temporal.workflow-service",
					"/temporal.api.workflowservice.v1.WorkflowService/DescribeNamespace", nil, nil))
			}})
		produced, err := cp.Produce(query, identity, realization, source)
		require.NoError(t, err)
		return slices.Contains(instructions(produced, "controller"), "after-backoff")
	}
	require.True(t, carries("retry"), "the retry path takes the backoff")
	require.False(t, carries("syncCompletion"), "the synchronous path does not")
}

// Preflight refuses what Produce refuses, in Produce's words, and writes nothing: a Query Produce
// lowers passes it, and one whose path records a fact no evidence source names does not.
func TestPreflightRefusesWhatProduceRefuses(t *testing.T) {
	query, identity, sound, source := producerContractFixture(t, "syncCompletion")
	require.NoError(t, cp.Preflight(query, identity, sound))

	_, _, unsourced, _ := producerContractFixture(t, "syncCompletion")
	unsourced.Sources = slices.DeleteFunc(slices.Clone(unsourced.Sources), func(s *cp.EvidenceSource) bool {
		return s.EventKind == "nexusOperationCompleted"
	})
	_, want := cp.Produce(query, identity, unsourced, source)
	require.EqualError(t, want, "nexusOperationCompleted: evidence.kind-unknown")
	require.Equal(t, want, cp.Preflight(query, identity, unsourced))
}
