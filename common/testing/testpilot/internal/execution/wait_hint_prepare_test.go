package execution

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
)

const (
	pollPath         = "program.entrypoints[controller].instructions[pending-attempts]"
	pollTimeoutPath  = pollPath + ".limits.timeout_milliseconds"
	pollIntervalPath = pollPath + ".instruction.read_evidence.poll_interval_milliseconds"
)

func waitHint(id string, line int32, atMost int64) *testpilotspb.WaitHint {
	return &testpilotspb.WaitHint{HintId: id, Source: &testpilotspb.SourceLocation{Path: "model/temporal/realize/Behavior.scala", Line: line}, AtMostMilliseconds: atMost}
}

// hintedFixture is evidenceFixture with its poll bounded by two hints whose bounds sum to the
// timeout it writes, under a Profile whose defaults differ from it, and writing no attempts so the
// default supplies them.
func hintedFixture(t *testing.T) (*testpilotspb.Case, *ir.Catalog, Profile) {
	t.Helper()
	source, catalog, policy := evidenceFixture(t)
	policy.InstructionDefaults = contract.InstructionDefaults{TimeoutMilliseconds: 10000, MaxAttempts: 2}
	poll := source.Program.Entrypoints[0].Instructions[2]
	poll.Limits = &testpilotspb.InstructionLimits{Timeout: &testpilotspb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: 3000}}
	poll.WaitHints = []*testpilotspb.WaitHint{
		waitHint("temporal.realize.cause.delivery", 12, 2000),
		waitHint("temporal.realize.visibility.pauseActivityExecution.describeActivityExecution", 20, 1000),
	}
	return source, catalog, policy
}

func hintedPoll(c *testpilotspb.Case) *testpilotspb.InstructionNode {
	return c.Program.Entrypoints[0].Instructions[2]
}

func setTimeout(n *testpilotspb.InstructionNode, timeout int64) {
	n.Limits.Timeout = &testpilotspb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: timeout}
}

func TestPrepareAdmitsHintedPollAtTheSumOfItsHints(t *testing.T) {
	t.Run("the sum, not the Profile default", func(t *testing.T) {
		source, catalog, policy := hintedFixture(t)
		prepared, err := Prepare(source, catalog, policy)
		require.NoError(t, err)
		poll := prepared.graphs[0].nodes[2]
		require.EqualValues(t, 3000, poll.timeoutMilliseconds)
		require.EqualValues(t, 2, poll.maxAttempts)
		require.EqualValues(t, 10, poll.pollIntervalMilliseconds)
		require.False(t, poll.once)
	})
	t.Run("one hint id twice", func(t *testing.T) {
		// A wait may sum the same kind of cause twice.
		source, catalog, policy := hintedFixture(t)
		hintedPoll(source).WaitHints[1].HintId = hintedPoll(source).WaitHints[0].HintId
		prepared, err := Prepare(source, catalog, policy)
		require.NoError(t, err)
		require.EqualValues(t, 3000, prepared.graphs[0].nodes[2].timeoutMilliseconds)
	})
}

func TestPrepareRejectsHintedWaits(t *testing.T) {
	hintPath := func(i int, field string) string {
		return fmt.Sprintf("%s.wait_hints[%d].%s", pollPath, i, field)
	}
	for _, tc := range []struct {
		name     string
		mutate   func(*testpilotspb.Case, *Profile)
		expected ir.Error
	}{
		{"hinted without a timeout under a Profile default", func(c *testpilotspb.Case, _ *Profile) { hintedPoll(c).Limits.Timeout = nil },
			ir.Error{Category: ir.Malformed, Path: pollTimeoutPath, Detail: "a hinted wait writes its own timeout; no Profile default applies to it"}},
		{"timeout other than the sum", func(c *testpilotspb.Case, _ *Profile) { setTimeout(hintedPoll(c), 3001) },
			ir.Error{Category: ir.Malformed, Path: pollTimeoutPath, Detail: "hinted wait timeout 3001 ms is not 3000 ms, the sum of its wait hints' bounds"}},
		{"zero bound", func(c *testpilotspb.Case, _ *Profile) { hintedPoll(c).WaitHints[1].AtMostMilliseconds = 0 },
			ir.Error{Category: ir.Malformed, Path: hintPath(1, "at_most_milliseconds"), Detail: "wait hint requires a positive bound"}},
		{"negative bound", func(c *testpilotspb.Case, _ *Profile) { hintedPoll(c).WaitHints[1].AtMostMilliseconds = -1 },
			ir.Error{Category: ir.Malformed, Path: hintPath(1, "at_most_milliseconds"), Detail: "wait hint requires a positive bound"}},
		{"bounds overflowing their sum", func(c *testpilotspb.Case, _ *Profile) {
			hintedPoll(c).WaitHints[0].AtMostMilliseconds = math.MaxInt64
		}, ir.Error{Category: ir.LimitExceeded, Path: hintPath(1, "at_most_milliseconds"), Detail: "wait hint bounds overflow their sum"}},
		{"missing source", func(c *testpilotspb.Case, _ *Profile) { hintedPoll(c).WaitHints[0].Source = nil },
			ir.Error{Category: ir.Malformed, Path: hintPath(0, "source"), Detail: "wait hint requires the source path and line it is declared at"}},
		{"source without a path", func(c *testpilotspb.Case, _ *Profile) { hintedPoll(c).WaitHints[0].Source.Path = "" },
			ir.Error{Category: ir.Malformed, Path: hintPath(0, "source"), Detail: "wait hint requires the source path and line it is declared at"}},
		{"source without a line", func(c *testpilotspb.Case, _ *Profile) { hintedPoll(c).WaitHints[0].Source.Line = 0 },
			ir.Error{Category: ir.Malformed, Path: hintPath(0, "source"), Detail: "wait hint requires the source path and line it is declared at"}},
		{"empty id", func(c *testpilotspb.Case, _ *Profile) { hintedPoll(c).WaitHints[0].HintId = "" },
			ir.Error{Category: ir.Malformed, Path: hintPath(0, "hint_id"), Detail: "wait hint requires a valid identity"}},
		{"invalid id", func(c *testpilotspb.Case, _ *Profile) { hintedPoll(c).WaitHints[0].HintId = "cause delivery" },
			ir.Error{Category: ir.Malformed, Path: hintPath(0, "hint_id"), Detail: "wait hint requires a valid identity"}},
		{"hints on an instruction other than ReadEvidence", func(c *testpilotspb.Case, _ *Profile) {
			history := c.Program.Entrypoints[0].Instructions[0]
			history.WaitHints, hintedPoll(c).WaitHints = hintedPoll(c).WaitHints, nil
			setTimeout(history, 3000)
		}, ir.Error{Category: ir.Unsupported, Path: "program.entrypoints[controller].instructions[history].wait_hints", Detail: "only a polling ReadEvidence waits within wait hints"}},
		{"hints on a read once", func(c *testpilotspb.Case, _ *Profile) {
			read := hintedPoll(c).Instruction.GetReadEvidence()
			read.Once, read.PollIntervalMilliseconds = true, 0
		}, ir.Error{Category: ir.Malformed, Path: pollPath + ".wait_hints", Detail: "a read once does not wait, so no wait hint bounds it"}},
		{"bound above the ceiling", func(c *testpilotspb.Case, _ *Profile) {
			hintedPoll(c).WaitHints[0].AtMostMilliseconds = 29001
			setTimeout(hintedPoll(c), 30001)
		}, ir.Error{Category: ir.LimitExceeded, Path: pollTimeoutPath, Detail: "wait bound 30001 ms exceeds the Profile total duration ceiling 30000 ms"}},
		{"scaled bound above the scaled ceiling", func(c *testpilotspb.Case, p *Profile) {
			p.BoundScale = 150
			hintedPoll(c).WaitHints[0].AtMostMilliseconds = 29001
			setTimeout(hintedPoll(c), 30001)
		}, ir.Error{Category: ir.LimitExceeded, Path: pollTimeoutPath, Detail: "scaled wait bound 45002 ms (30001 ms declared, scaled by 150%) exceeds the scaled Profile total duration ceiling 45000 ms"}},
		{"cleanup bound above the cleanup ceiling", func(c *testpilotspb.Case, _ *Profile) {
			poll := hintedPoll(c)
			c.Program.Entrypoints[0].Instructions = c.Program.Entrypoints[0].Instructions[:2]
			c.Program.Cleanup.Instructions = []*testpilotspb.InstructionNode{poll}
			poll.WaitHints[0].AtMostMilliseconds = 4001
			setTimeout(poll, 5001)
		}, ir.Error{Category: ir.LimitExceeded, Path: "program.cleanup.instructions[pending-attempts].limits.timeout_milliseconds", Detail: "wait bound 5001 ms exceeds the Profile cleanup duration ceiling 5000 ms"}},
		{"interval above the declared timeout, within the scaled one", func(c *testpilotspb.Case, p *Profile) {
			// The interval is declared against the declared bound, so a scale admits no Case it
			// would refuse unscaled.
			p.BoundScale = 150
			hintedPoll(c).Instruction.GetReadEvidence().PollIntervalMilliseconds = 3001
		}, ir.Error{Category: ir.LimitExceeded, Path: pollIntervalPath, Detail: "poll interval exceeds the instruction timeout"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source, catalog, policy := hintedFixture(t)
			tc.mutate(source, &policy)
			_, err := Prepare(source, catalog, policy)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, tc.expected, *diagnostic)
		})
	}
}

func TestPrepareAdmitsReadOnceWithoutInterval(t *testing.T) {
	source, catalog, policy := evidenceFixture(t)
	hintedPoll(source).Instruction.GetReadEvidence().Once = true
	hintedPoll(source).Instruction.GetReadEvidence().PollIntervalMilliseconds = 0
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	read := prepared.graphs[0].nodes[2]
	require.True(t, read.once)
	require.Zero(t, read.pollIntervalMilliseconds)
	require.EqualValues(t, 1000, read.timeoutMilliseconds)
	require.Len(t, read.responseReads, 1)
}

func TestPrepareRejectsReadIntervals(t *testing.T) {
	for _, tc := range []struct {
		name     string
		once     bool
		interval int64
		expected ir.Error
	}{
		{"read once with an interval", true, 10, ir.Error{Category: ir.Malformed, Path: pollIntervalPath, Detail: "a read once has no poll interval"}},
		{"poll without an interval", false, 0, ir.Error{Category: ir.Malformed, Path: pollIntervalPath, Detail: "ReadEvidence requires a positive poll interval"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source, catalog, policy := evidenceFixture(t)
			read := hintedPoll(source).Instruction.GetReadEvidence()
			read.Once, read.PollIntervalMilliseconds = tc.once, tc.interval
			_, err := Prepare(source, catalog, policy)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, tc.expected, *diagnostic)
		})
	}
}

// A bound scale scales a hinted poll's timeout, rounding up, and the ceilings with it, and leaves
// every unhinted instruction's timeout as written.
func TestPrepareScalesOnlyHintedTimeouts(t *testing.T) {
	source, catalog, policy := hintedFixture(t)
	policy.BoundScale = 150
	hintedPoll(source).WaitHints[0].AtMostMilliseconds = 2001
	setTimeout(hintedPoll(source), 3001)
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	nodes := prepared.graphs[0].nodes
	require.EqualValues(t, 4502, nodes[2].timeoutMilliseconds)
	require.EqualValues(t, 1000, nodes[0].timeoutMilliseconds)
	require.EqualValues(t, 1000, nodes[1].timeoutMilliseconds)
	require.EqualValues(t, 45000, prepared.limits.MaxTotalDurationMilliseconds)
	require.EqualValues(t, 7500, prepared.limits.MaxCleanupDurationMilliseconds)
	require.EqualValues(t, 30000, policy.Limits.MaxTotalDurationMilliseconds, "the Profile's own limits stay as declared")
}

// An unhinted Case prepares the same instruction limits under every bound scale.
func TestPrepareLeavesUnhintedCasesUnscaled(t *testing.T) {
	for name, fixture := range map[string]func(*testing.T) (*testpilotspb.Case, *ir.Catalog, Profile){
		"evidence": evidenceFixture, "handle": handleFixture,
	} {
		t.Run(name, func(t *testing.T) {
			source, catalog, policy := fixture(t)
			unscaled, err := Prepare(source, catalog, policy)
			require.NoError(t, err)
			require.Same(t, policy.Limits, unscaled.limits)
			policy.BoundScale = 150
			scaled, err := Prepare(source, catalog, policy)
			require.NoError(t, err)
			require.Len(t, scaled.graphs, len(unscaled.graphs))
			for g, graph := range unscaled.graphs {
				require.Len(t, scaled.graphs[g].nodes, len(graph.nodes))
				for i, n := range graph.nodes {
					other := scaled.graphs[g].nodes[i]
					require.Equal(t, n.timeoutMilliseconds, other.timeoutMilliseconds, n.source.InstructionId)
					require.Equal(t, n.maxAttempts, other.maxAttempts, n.source.InstructionId)
					require.Equal(t, n.pollIntervalMilliseconds, other.pollIntervalMilliseconds, n.source.InstructionId)
					require.Equal(t, n.source.GetLimits().GetTimeoutMilliseconds(), n.timeoutMilliseconds, n.source.InstructionId)
				}
			}
		})
	}
}

// A scale widens only a hinted bound's ceiling: an unhinted timeout above the declared ceiling is
// refused under every scale, so whether a Case admits never depends on the scale.
func TestPrepareAdmitsUnhintedTimeoutsUnderTheDeclaredCeiling(t *testing.T) {
	for _, scale := range []contract.BoundScale{0, 150} {
		source, catalog, policy := evidenceFixture(t)
		policy.BoundScale = scale
		poll := source.Program.Entrypoints[0].Instructions[2]
		poll.Limits = &testpilotspb.InstructionLimits{Timeout: &testpilotspb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: policy.Limits.MaxTotalDurationMilliseconds + 1}, Attempts: &testpilotspb.InstructionLimits_MaxAttempts{MaxAttempts: 1}}
		_, err := Prepare(source, catalog, policy)
		require.ErrorContains(t, err, "instruction bounds exceed Profile ceilings", "scale %d", scale)
	}
}
