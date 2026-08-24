package choice

import (
	"crypto/sha256"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
)

func TestExplorationExpandsAllNonSelectedRanksBreadthFirst(t *testing.T) {
	config := testConfig()
	config.Parallel = 2
	state, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	rootRound, ok := state.NextRound()
	if !ok || len(rootRound.Candidates) != 1 || rootRound.Candidates[0].ForcedDepth != 0 {
		t.Fatalf("root round = %#v, ok=%t", rootRound, ok)
	}
	trace := testTape(t, config.Execution,
		testDecision(t, 0, choice.KindRunnable, 1, 0),
		testDecision(t, 1, choice.KindSelectPoll, 3, 1),
		testDecision(t, 2, choice.KindRunnable, 2, 0),
	)
	state, _, err = CommitRound(state, rootRound, []Result{testResult(rootRound.Candidates[0], trace, "root")})
	if err != nil {
		t.Fatal(err)
	}
	if got := state.Summary(); got.LogicalExecutions != 1 || got.Pending != 3 || got.SeenPrefixes != 4 || got.DeepestPrefix != 2 {
		t.Fatalf("summary = %#v", got)
	}
	depths := []uint64{state.Queue[0].ForcedDepth, state.Queue[1].ForcedDepth, state.Queue[2].ForcedDepth}
	if !slices.Equal(depths, []uint64{1, 1, 2}) {
		t.Fatalf("forced depths = %v", depths)
	}
	if state.Queue[0].SHA256 > state.Queue[1].SHA256 {
		t.Fatalf("same-depth candidates are not digest sorted: %#v", state.Queue)
	}
	next, ok := state.NextRound()
	if !ok || len(next.Candidates) != 2 || next.Candidates[0].ForcedDepth != 1 || next.Candidates[1].ForcedDepth != 1 {
		t.Fatalf("next round = %#v, ok=%t", next, ok)
	}
}

func TestExplorationDeduplicatesPrefixesAndOutcomesWithoutPruning(t *testing.T) {
	config := testConfig()
	config.Parallel = 1
	state, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	rootRound, _ := state.NextRound()
	rootTrace := testTape(t, config.Execution, testDecision(t, 0, choice.KindRunnable, 2, 0))
	sharedOutcome := record.HashBytes([]byte("shared outcome"))
	state, _, err = CommitRound(state, rootRound, []Result{{CandidateSHA256: rootRound.Candidates[0].SHA256, OutcomeSHA256: sharedOutcome, Trace: &rootTrace}})
	if err != nil {
		t.Fatal(err)
	}
	childRound, _ := state.NextRound()
	childTrace := testTape(t, config.Execution,
		testDecision(t, 0, choice.KindRunnable, 2, 1),
		testDecision(t, 1, choice.KindSelectPoll, 2, 0),
	)
	state, _, err = CommitRound(state, childRound, []Result{{CandidateSHA256: childRound.Candidates[0].SHA256, OutcomeSHA256: sharedOutcome, Trace: &childTrace}})
	if err != nil {
		t.Fatal(err)
	}
	summary := state.Summary()
	if summary.DeduplicatedOutcomes != 1 || summary.SeenPrefixes != 3 || summary.Pending != 1 || state.Queue[0].ForcedDepth != 2 {
		t.Fatalf("deduplicated exploration = %#v, queue=%#v", summary, state.Queue)
	}
}

func TestExplorationStopsAtExplicitRunDepthAndCapacityBounds(t *testing.T) {
	for _, test := range []struct {
		name      string
		configure func(*Config, State)
		trace     func(*testing.T, choice.ExecutionIdentity) choice.ReplayPlan
		want      StopReason
		omitted   func(Summary) uint64
	}{
		{
			name: "executions", configure: func(config *Config, _ State) { config.MaxExecutions = 1 },
			trace: func(t *testing.T, identity choice.ExecutionIdentity) choice.ReplayPlan {
				return testTape(t, identity, testDecision(t, 0, choice.KindRunnable, 2, 0))
			},
			want: StopMaxExecutions, omitted: func(summary Summary) uint64 { return summary.OmittedByExecutionBound },
		},
		{
			name: "depth", configure: func(config *Config, _ State) { config.MaxChoiceDepth = 1 },
			trace: func(t *testing.T, identity choice.ExecutionIdentity) choice.ReplayPlan {
				return testTape(t, identity,
					testDecision(t, 0, choice.KindRunnable, 2, 0),
					testDecision(t, 1, choice.KindRunnable, 2, 0),
				)
			},
			want: StopDepthComplete, omitted: func(summary Summary) uint64 { return summary.OmittedByDepth },
		},
		{
			name: "capacity", configure: func(config *Config, initial State) { config.MaxExplorationBytes = initial.Summary().PendingBytes },
			trace: func(t *testing.T, identity choice.ExecutionIdentity) choice.ReplayPlan {
				return testTape(t, identity, testDecision(t, 0, choice.KindRunnable, 2, 0))
			},
			want: StopExplorationCapacity, omitted: func(summary Summary) uint64 { return summary.OmittedByCapacity },
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := testConfig()
			initial, err := New(config)
			if err != nil {
				t.Fatal(err)
			}
			test.configure(&config, initial)
			state, err := New(config)
			if err != nil {
				t.Fatal(err)
			}
			round, _ := state.NextRound()
			trace := test.trace(t, config.Execution)
			state, _, err = CommitRound(state, round, []Result{testResult(round.Candidates[0], trace, test.name)})
			if err != nil {
				t.Fatal(err)
			}
			if test.name == "depth" {
				next, ok := state.NextRound()
				if !ok || len(next.Candidates) != 1 {
					t.Fatalf("depth round = %#v, ok=%t", next, ok)
				}
				prefix, err := next.Candidates[0].PrefixReplayPlan(config.Execution)
				if err != nil {
					t.Fatal(err)
				}
				decisions := []choice.Decision{
					testDecision(t, 0, choice.KindRunnable, 2, prefix.Decisions[0].Selected),
					testDecision(t, 1, choice.KindRunnable, 2, 0),
				}
				childTrace := testTape(t, config.Execution, decisions...)
				state, _, err = CommitRound(state, next, []Result{testResult(next.Candidates[0], childTrace, "depth-child")})
				if err != nil {
					t.Fatal(err)
				}
			}
			summary := state.Summary()
			if summary.StopReason != test.want || test.omitted(summary) == 0 {
				t.Fatalf("summary = %#v, want stop %q with omissions", summary, test.want)
			}
		})
	}
}

func TestExplorationRejectsIncompleteExecutionIdentity(t *testing.T) {
	config := testConfig()
	config.Execution.TargetSHA256 = [sha256.Size]byte{}
	if _, err := New(config); err == nil {
		t.Fatal("New() accepted an incomplete execution identity")
	}
}

func TestExplorationRoundSegmentReplaysByteIdentically(t *testing.T) {
	config := testConfig()
	initial, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	round, _ := initial.NextRound()
	trace := testTape(t, config.Execution, testDecision(t, 0, choice.KindRunnable, 3, 1))
	committed, segment, err := CommitRound(initial, round, []Result{testResult(round.Candidates[0], trace, "root")})
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := ReplaySegment(initial, segment)
	if err != nil {
		t.Fatal(err)
	}
	committedBytes, err := canonicaljson.CanonicalJSON(committed)
	if err != nil {
		t.Fatal(err)
	}
	replayedBytes, err := canonicaljson.CanonicalJSON(replayed)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(committedBytes, replayedBytes) {
		t.Fatalf("replayed state differs:\n%s\n%s", committedBytes, replayedBytes)
	}
	segmentBytes, err := canonicaljson.CanonicalJSON(segment)
	if err != nil || segment.SHA256 != record.DomainHash(roundSegmentDomain, segmentBytesWithoutIdentity(t, segment)) {
		t.Fatalf("segment identity = %q, bytes=%s, error=%v", segment.SHA256, segmentBytes, err)
	}
}

func testConfig() Config {
	return Config{
		Execution: choice.ExecutionIdentity{
			TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: strings.Repeat("a", 64),
			GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: sha256.Sum256([]byte("controller")),
		},
		ControllerSHA256: ImplementationSHA256(),
		BaseSeed:         7, Parallel: 4, MaxExecutions: 32, MaxChoiceDepth: 8, MaxExplorationBytes: 1 << 20,
		FailurePolicy: PolicyAll, FailureBudget: 1,
	}
}

func testDecision(t *testing.T, ordinal uint64, kind choice.Kind, alternatives, selected uint32) choice.Decision {
	t.Helper()
	identities := make([][sha256.Size]byte, alternatives)
	for index := range identities {
		identities[index] = sha256.Sum256([]byte{byte(ordinal), byte(index + 1)})
	}
	decision, err := choice.CanonicalDecision(ordinal, kind, ordinal+1, false, identities, identities[selected], 0)
	if err != nil {
		t.Fatal(err)
	}
	return decision
}

func testTape(t *testing.T, identity choice.ExecutionIdentity, decisions ...choice.Decision) choice.ReplayPlan {
	t.Helper()
	records := make([]choice.Record, len(decisions))
	for index, decision := range decisions {
		records[index] = decision.Record()
	}
	trace, err := choice.BuildTrace(records, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	tape, err := choice.ProjectReplayPlan(trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	return tape
}

func testResult(candidate Candidate, trace choice.ReplayPlan, outcome string) Result {
	return Result{CandidateSHA256: candidate.SHA256, OutcomeSHA256: record.HashBytes([]byte(outcome)), Trace: &trace}
}

func segmentBytesWithoutIdentity(t *testing.T, segment RoundSegment) []byte {
	t.Helper()
	segment.SHA256 = ""
	encoded, err := canonicaljson.CanonicalJSON(segment)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
