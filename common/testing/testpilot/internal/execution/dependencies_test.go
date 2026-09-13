package execution

import (
	"context"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

func TestPrivateCoreImportBoundary(t *testing.T) {
	for _, packageName := range []string{"ir", "execution", "verification"} {
		command := exec.CommandContext(t.Context(), "go", "list", "-tags", "test_dep", "-deps", "go.temporal.io/server/common/testing/testpilot/internal/"+packageName)
		output, err := command.Output()
		require.NoError(t, err)
		for _, dependency := range strings.Fields(string(output)) {
			for _, prefix := range []string{"go.temporal.io/server/tools/umpire", "go.temporal.io/server/tests", "go.temporal.io/sdk"} {
				require.False(t, dependency == prefix || strings.HasPrefix(dependency, prefix+"/"), "forbidden dependency: %s", dependency)
			}
		}
	}
}

func TestInstructionsRunAfterTheirPredecessorUnlessAfterSaysOtherwise(t *testing.T) {
	c, catalog, policy := fixture(t)
	second := rpcNode("second")
	root := rpcNode("root")
	root.After = runsAfter("controller")
	join := rpcNode("join")
	join.After = runsAfter("controller", "second", "root")
	regardless := rpcNode("regardless")
	regardless.Guard = alwaysRuns()
	c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, second, root, join, regardless)
	c.Program.Cleanup.Instructions = []*testpilotspb.InstructionNode{rpcNode("release"), rpcNode("confirm")}
	prepared, err := Prepare(c, catalog, policy)
	require.NoError(t, err)

	type shape struct {
		dependencies []int
		succeeded    []string
	}
	shapeOf := func(n *node) shape {
		result := shape{dependencies: n.dependencies}
		for id := range successFacts(n.guard) {
			result.succeeded = append(result.succeeded, id)
		}
		slices.Sort(result.succeeded)
		return result
	}
	controller := prepared.graphs[0]
	require.Equal(t, []shape{
		{},
		{dependencies: []int{0}, succeeded: []string{"call"}},
		{},
		{dependencies: []int{1, 2}, succeeded: []string{"root", "second"}},
		{dependencies: []int{3}},
	}, []shape{shapeOf(controller.nodes[0]), shapeOf(controller.nodes[1]), shapeOf(controller.nodes[2]), shapeOf(controller.nodes[3]), shapeOf(controller.nodes[4])})
	require.Equal(t, []int{0, 2, 1, 3, 4}, controller.order)
	require.Nil(t, controller.nodes[4].guard, "a true guard runs regardless, like no guard")
	cleanup := prepared.cleanupGraph()
	require.Equal(t, shape{dependencies: []int{0}, succeeded: []string{"release"}}, shapeOf(cleanup.nodes[1]))
}

// The default guard makes a dependency's success statically known, exactly as an explicit success
// guard does, so a completion may consume the awaited handle and a finish may return the awaited
// value without writing that guard.
func TestTheDefaultGuardBindsDependencyOutcomes(t *testing.T) {
	c, catalog, policy := capabilityFixture(t)
	c.Program.Entrypoints[0].Instructions[2].Guard = nil
	c.Program.Entrypoints[1].Instructions[2].Guard = nil
	_, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
}

func TestTheDefaultGuardSkipsAfterAFailedDependency(t *testing.T) {
	c, catalog, policy := fixture(t)
	next := rpcNode("next")
	regardless := rpcNode("regardless")
	regardless.After = runsAfter("controller", "call")
	regardless.Guard = alwaysRuns()
	c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, next, regardless)
	p, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	var mu sync.Mutex
	var calls []string
	host := &schedulerHost{invoke: func(_ context.Context, c contract.Coordinate, _ proto.Message) (contract.EffectHandle, error) {
		mu.Lock()
		calls = append(calls, c.InstructionID)
		mu.Unlock()
		return &schedulerEffect{wait: func(context.Context) (contract.EffectResult, error) {
			if c.InstructionID == "call" {
				return contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE, ProtocolCode: "unavailable"}}, nil
			}
			return effectResponse(p, "ok"), nil
		}}, nil
	}}
	s, err := newScheduler(p, "run", "case", host, schedulerMonitor{}, time.Now)
	require.NoError(t, err)
	require.NoError(t, s.execute(context.Background()))
	require.Equal(t, []string{"call", "regardless"}, calls)
}

func TestPrepareRejectsAfterWithLocatedPaths(t *testing.T) {
	for name, tc := range map[string]struct {
		mutate   func(*testpilotspb.Case)
		category ir.ErrorCategory
		path     string
	}{
		"unknown instruction": {
			mutate: func(c *testpilotspb.Case) {
				c.Program.Entrypoints[0].Instructions[1].After = runsAfter("controller", "missing")
			},
			category: ir.Unknown, path: "program.entrypoints[controller].instructions[second].after.instructions[0]",
		},
		"itself": {
			mutate: func(c *testpilotspb.Case) {
				c.Program.Entrypoints[0].Instructions[1].After = runsAfter("controller", "second")
			},
			category: ir.Malformed, path: "program.entrypoints[controller].instructions[second].after.instructions[0]",
		},
		"duplicate": {
			mutate: func(c *testpilotspb.Case) {
				c.Program.Entrypoints[0].Instructions[1].After = runsAfter("controller", "call", "call")
			},
			category: ir.Malformed, path: "program.entrypoints[controller].instructions[second].after.instructions[1]",
		},
		"cycle": {
			mutate: func(c *testpilotspb.Case) {
				c.Program.Entrypoints[0].Instructions[0].After = runsAfter("controller", "second")
			},
			category: ir.Malformed, path: "program.entrypoints[controller].instructions[call].after",
		},
		"another entrypoint": {
			mutate: func(c *testpilotspb.Case) {
				other := proto.CloneOf(c.Program.Entrypoints[0])
				other.EntrypointId = "other"
				c.Program.Entrypoints = append(c.Program.Entrypoints, other)
				other.Instructions[1].After = runsAfter("controller", "call")
			},
			category: ir.Unsupported, path: "program.entrypoints[other].instructions[second].after.instructions[0]",
		},
		"into cleanup": {
			mutate: func(c *testpilotspb.Case) {
				c.Program.Cleanup.Instructions = []*testpilotspb.InstructionNode{rpcNode("release")}
				c.Program.Entrypoints[0].Instructions[1].After = runsAfter("cleanup", "release")
			},
			category: ir.Unsupported, path: "program.entrypoints[controller].instructions[second].after.instructions[0]",
		},
		"out of cleanup": {
			mutate: func(c *testpilotspb.Case) {
				release := rpcNode("release")
				release.After = runsAfter("controller", "call")
				c.Program.Cleanup.Instructions = []*testpilotspb.InstructionNode{release}
			},
			category: ir.Unsupported, path: "program.cleanup.instructions[release].after.instructions[0]",
		},
		"no entrypoint": {
			mutate:   func(c *testpilotspb.Case) { c.Program.Entrypoints[0].Instructions[1].After = runsAfter("", "call") },
			category: ir.Malformed, path: "program.entrypoints[controller].instructions[second].after.instructions[0]",
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, policy := fixture(t)
			c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, rpcNode("second"))
			tc.mutate(c)
			_, err := Prepare(c, catalog, policy)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, tc.category, diagnostic.Category)
			require.Equal(t, tc.path, diagnostic.Path)
		})
	}
}
