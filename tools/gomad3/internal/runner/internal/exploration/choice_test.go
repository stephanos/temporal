package exploration

import (
	"crypto/sha256"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
)

func TestExplorationExpandsAllNonSelectedRanksBreadthFirst(t *testing.T) {
	config := choiceTestConfig()
	config.Parallel = 2
	state, err := NewChoice(config)
	if err != nil {
		t.Fatal(err)
	}
	rootRound, ok := state.NextRound()
	if !ok || len(rootRound.Candidates) != 1 || rootRound.Candidates[0].ForcedDepth != 0 {
		t.Fatalf("root round = %#v, ok=%t", rootRound, ok)
	}
	trace := testTape(t, config.Execution,
		choiceTestDecision(t, 0, choice.KindRunnable, 1, 0),
		choiceTestDecision(t, 1, choice.KindSelectPoll, 3, 1),
		choiceTestDecision(t, 2, choice.KindRunnable, 2, 0),
	)
	state, _, err = CommitChoiceRound(state, rootRound, []ChoiceResult{testResult(rootRound.Candidates[0], trace, "root")})
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
	config := choiceTestConfig()
	config.Parallel = 1
	state, err := NewChoice(config)
	if err != nil {
		t.Fatal(err)
	}
	rootRound, _ := state.NextRound()
	rootTrace := testTape(t, config.Execution, choiceTestDecision(t, 0, choice.KindRunnable, 2, 0))
	sharedOutcome := record.HashBytes([]byte("shared outcome"))
	state, _, err = CommitChoiceRound(state, rootRound, []ChoiceResult{{CandidateSHA256: rootRound.Candidates[0].SHA256, OutcomeSHA256: sharedOutcome, Trace: &rootTrace}})
	if err != nil {
		t.Fatal(err)
	}
	childRound, _ := state.NextRound()
	childTrace := testTape(t, config.Execution,
		choiceTestDecision(t, 0, choice.KindRunnable, 2, 1),
		choiceTestDecision(t, 1, choice.KindSelectPoll, 2, 0),
	)
	state, _, err = CommitChoiceRound(state, childRound, []ChoiceResult{{CandidateSHA256: childRound.Candidates[0].SHA256, OutcomeSHA256: sharedOutcome, Trace: &childTrace}})
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
		configure func(*ChoiceConfig, ChoiceState)
		trace     func(*testing.T, choice.ExecutionIdentity) choice.ReplayPlan
		want      ChoiceStopReason
		omitted   func(ChoiceSummary) uint64
	}{
		{
			name: "executions", configure: func(config *ChoiceConfig, _ ChoiceState) { config.MaxExecutions = 1 },
			trace: func(t *testing.T, identity choice.ExecutionIdentity) choice.ReplayPlan {
				return testTape(t, identity, choiceTestDecision(t, 0, choice.KindRunnable, 2, 0))
			},
			want: ChoiceStopMaxExecutions, omitted: func(summary ChoiceSummary) uint64 { return summary.OmittedByExecutionBound },
		},
		{
			name: "depth", configure: func(config *ChoiceConfig, _ ChoiceState) { config.MaxChoiceDepth = 1 },
			trace: func(t *testing.T, identity choice.ExecutionIdentity) choice.ReplayPlan {
				return testTape(t, identity,
					choiceTestDecision(t, 0, choice.KindRunnable, 2, 0),
					choiceTestDecision(t, 1, choice.KindRunnable, 2, 0),
				)
			},
			want: ChoiceStopDepthComplete, omitted: func(summary ChoiceSummary) uint64 { return summary.OmittedByDepth },
		},
		{
			name: "capacity", configure: func(config *ChoiceConfig, initial ChoiceState) {
				config.MaxExplorationBytes = initial.Summary().PendingBytes
			},
			trace: func(t *testing.T, identity choice.ExecutionIdentity) choice.ReplayPlan {
				return testTape(t, identity, choiceTestDecision(t, 0, choice.KindRunnable, 2, 0))
			},
			want: ChoiceStopExplorationCapacity, omitted: func(summary ChoiceSummary) uint64 { return summary.OmittedByCapacity },
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := choiceTestConfig()
			initial, err := NewChoice(config)
			if err != nil {
				t.Fatal(err)
			}
			test.configure(&config, initial)
			state, err := NewChoice(config)
			if err != nil {
				t.Fatal(err)
			}
			round, _ := state.NextRound()
			trace := test.trace(t, config.Execution)
			state, _, err = CommitChoiceRound(state, round, []ChoiceResult{testResult(round.Candidates[0], trace, test.name)})
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
					choiceTestDecision(t, 0, choice.KindRunnable, 2, prefix.Decisions[0].Selected),
					choiceTestDecision(t, 1, choice.KindRunnable, 2, 0),
				}
				childTrace := testTape(t, config.Execution, decisions...)
				state, _, err = CommitChoiceRound(state, next, []ChoiceResult{testResult(next.Candidates[0], childTrace, "depth-child")})
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

func TestExplorationStartOrdinalExpandsOnlyFromTheStart(t *testing.T) {
	config := choiceTestConfig()
	config.StartOrdinal = 2
	config.MaxChoiceDepth = 1
	state, err := NewChoice(config)
	if err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	trace := testTape(t, config.Execution,
		choiceTestDecision(t, 0, choice.KindRunnable, 2, 0),
		choiceTestDecision(t, 1, choice.KindSelectPoll, 3, 1),
		choiceTestDecision(t, 2, choice.KindRunnable, 2, 0),
		choiceTestDecision(t, 3, choice.KindRunnable, 3, 0),
	)
	state, _, err = CommitChoiceRound(state, round, []ChoiceResult{testResult(round.Candidates[0], trace, "root")})
	if err != nil {
		t.Fatal(err)
	}
	summary := state.Summary()
	if summary.StartOrdinal != 2 || summary.Pending != 1 || summary.OmittedByDepth != 2 || summary.DeepestPrefix != 3 {
		t.Fatalf("summary = %#v", summary)
	}
	prefix, err := state.Queue[0].PrefixReplayPlan(config.Execution)
	if err != nil {
		t.Fatal(err)
	}
	if len(prefix.Decisions) != 3 || prefix.Decisions[0] != trace.Decisions[0] || prefix.Decisions[1] != trace.Decisions[1] || !prefix.Decisions[2].RankOverride || prefix.Decisions[2].Selected != 1 {
		t.Fatalf("forced prefix = %#v", prefix.Decisions)
	}
}

func TestExplorationReportsStartAtOrPastRootTrace(t *testing.T) {
	for _, test := range []struct {
		name    string
		start   uint64
		pending uint64
		want    ChoiceStopReason
	}{
		{name: "before end", start: 1, pending: 1},
		{name: "at end", start: 2, want: ChoiceStopStartUnreached},
		{name: "past end", start: 9, want: ChoiceStopStartUnreached},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := choiceTestConfig()
			config.StartOrdinal = test.start
			state, err := NewChoice(config)
			if err != nil {
				t.Fatal(err)
			}
			round, _ := state.NextRound()
			trace := testTape(t, config.Execution, choiceTestDecision(t, 0, choice.KindRunnable, 2, 0), choiceTestDecision(t, 1, choice.KindRunnable, 2, 0))
			state, _, err = CommitChoiceRound(state, round, []ChoiceResult{testResult(round.Candidates[0], trace, "root")})
			if err != nil {
				t.Fatal(err)
			}
			if summary := state.Summary(); summary.StopReason != test.want || summary.Pending != test.pending || summary.BoundedComplete {
				t.Fatalf("summary = %#v, want stop %q with %d pending", summary, test.want, test.pending)
			}
		})
	}
}

func TestExplorationRejectsCandidateAlteringADecisionBeforeTheStart(t *testing.T) {
	config := choiceTestConfig()
	trace := testTape(t, config.Execution, choiceTestDecision(t, 0, choice.KindRunnable, 2, 0), choiceTestDecision(t, 1, choice.KindRunnable, 2, 0))
	prefix, err := choice.BuildRankPrefix(trace, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	config.StartOrdinal = 2
	if _, err := choiceNewCandidate(config, &prefix, "", ""); err == nil || !strings.Contains(err.Error(), "before start ordinal 2") {
		t.Fatalf("newCandidate() error = %v", err)
	}
	config.StartOrdinal = 1
	if _, err := choiceNewCandidate(config, &prefix, "", ""); err != nil {
		t.Fatal(err)
	}
}

func TestExplorationSkipsNoOpSelectPollDecisions(t *testing.T) {
	config := choiceTestConfig()
	state, err := NewChoice(config)
	if err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	// A two-case select with one ready case between two runnable decisions.
	trace := testTapeRecords(t, config.Execution,
		choiceTestDecision(t, 0, choice.KindRunnable, 2, 0).Record(),
		testSiteDecision(t, 1, choice.KindSelectPoll, 40, 2, 1).Record(),
		testSelectResult(t, 2, 1, 40, 2, choice.SelectReadiness{Known: true, Ready: 1}),
		choiceTestDecision(t, 3, choice.KindRunnable, 3, 0).Record(),
	)
	state, _, err = CommitChoiceRound(state, round, []ChoiceResult{testResult(round.Candidates[0], trace, "root")})
	if err != nil {
		t.Fatal(err)
	}
	summary := state.Summary()
	if summary.Pending != 3 || summary.OmittedBySelectReadiness != 1 || summary.OmittedByDepth != 0 || summary.StopReason != "" {
		t.Fatalf("summary = %#v", summary)
	}
	for _, candidate := range state.Queue {
		prefix, err := candidate.PrefixReplayPlan(config.Execution)
		if err != nil {
			t.Fatal(err)
		}
		if overridden := prefix.Decisions[len(prefix.Decisions)-1]; overridden.Kind == choice.KindSelectPoll {
			t.Fatalf("candidate %s overrides the no-op select-poll decision", candidate.SHA256)
		}
	}
}

func TestExplorationExpandsSelectPollDecisionsOutsideTheNoOpShapes(t *testing.T) {
	oneReady := choice.SelectReadiness{Known: true, Ready: 1}
	twoCasePoll := func(t *testing.T) []choice.Record {
		return []choice.Record{testSiteDecision(t, 0, choice.KindSelectPoll, 40, 2, 1).Record()}
	}
	for _, test := range []struct {
		name    string
		records func(*testing.T) []choice.Record
		pending uint64
		omitted uint64
	}{
		{
			name: "listed shape", pending: 0, omitted: 1,
			records: func(t *testing.T) []choice.Record {
				return append(twoCasePoll(t), testSelectResult(t, 1, 0, 40, 2, oneReady))
			},
		},
		{
			name: "unknown readiness", pending: 1,
			records: twoCasePoll,
		},
		{
			name: "unlisted shape with the same ready count", pending: 1,
			records: func(t *testing.T) []choice.Record {
				return append(twoCasePoll(t), testSelectResult(t, 1, 0, 40, 2, choice.SelectReadiness{Known: true, Ready: 1, RepeatedChannel: true}))
			},
		},
		{
			name: "two ready", pending: 1,
			records: func(t *testing.T) []choice.Record {
				return append(twoCasePoll(t), testSelectResult(t, 1, 0, 40, 2, choice.SelectReadiness{Known: true, Ready: 2}))
			},
		},
		{
			name: "three polled cases", pending: 3,
			records: func(t *testing.T) []choice.Record {
				return append(twoCasePoll(t), testSiteDecision(t, 1, choice.KindSelectPoll, 40, 3, 2).Record(), testSelectResult(t, 2, 0, 40, 3, oneReady))
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := choiceTestConfig()
			state, err := NewChoice(config)
			if err != nil {
				t.Fatal(err)
			}
			round, _ := state.NextRound()
			trace := testTapeRecords(t, config.Execution, test.records(t)...)
			state, _, err = CommitChoiceRound(state, round, []ChoiceResult{testResult(round.Candidates[0], trace, test.name)})
			if err != nil {
				t.Fatal(err)
			}
			summary := state.Summary()
			if summary.Pending != test.pending || summary.OmittedBySelectReadiness != test.omitted {
				t.Fatalf("summary = %#v, want %d pending and %d omitted", summary, test.pending, test.omitted)
			}
			if test.pending == 0 && (summary.StopReason != ChoiceStopExhausted || !summary.BoundedComplete) {
				t.Fatalf("summary = %#v, want exhaustion", summary)
			}
		})
	}
}

func TestExplorationRejectsIncompleteExecutionIdentity(t *testing.T) {
	config := choiceTestConfig()
	config.Execution.TargetSHA256 = [sha256.Size]byte{}
	if _, err := NewChoice(config); err == nil {
		t.Fatal("New() accepted an incomplete execution identity")
	}
}

func TestExplorationRoundSegmentReplaysByteIdentically(t *testing.T) {
	config := choiceTestConfig()
	initial, err := NewChoice(config)
	if err != nil {
		t.Fatal(err)
	}
	round, _ := initial.NextRound()
	// The no-op select keeps the replayed round under the rule it was
	// committed with.
	trace := testTapeRecords(t, config.Execution,
		choiceTestDecision(t, 0, choice.KindRunnable, 3, 1).Record(),
		testSiteDecision(t, 1, choice.KindSelectPoll, 40, 2, 0).Record(),
		testSelectResult(t, 2, 1, 40, 2, choice.SelectReadiness{Known: true, Default: true}),
	)
	committed, segment, err := CommitChoiceRound(initial, round, []ChoiceResult{testResult(round.Candidates[0], trace, "root")})
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := ChoiceReplaySegment(initial, segment)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.OmittedBySelectReadiness != 1 || len(replayed.Queue) != 2 {
		t.Fatalf("replayed state = %#v", replayed.Summary())
	}
	committedBytes, err := json.Marshal(committed)
	if err != nil {
		t.Fatal(err)
	}
	replayedBytes, err := json.Marshal(replayed)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(committedBytes, replayedBytes) {
		t.Fatalf("replayed state differs:\n%s\n%s", committedBytes, replayedBytes)
	}
	segmentBytes, err := json.Marshal(segment)
	if err != nil || segment.SHA256 != record.DomainHash(choiceRoundSegmentDomain, segmentBytesWithoutIdentity(t, segment)) {
		t.Fatalf("segment identity = %q, bytes=%s, error=%v", segment.SHA256, segmentBytes, err)
	}
}

func choiceTestConfig() ChoiceConfig {
	return ChoiceConfig{
		Execution: choice.ExecutionIdentity{
			TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: strings.Repeat("a", 64),
			GOOS: "darwin", GOARCH: "arm64", ImplementationSHA256: sha256.Sum256([]byte("controller")),
		},
		ControllerSHA256: ChoiceImplementationSHA256(),
		BaseSeed:         7, Parallel: 4, MaxExecutions: 32, MaxChoiceDepth: 8, MaxExplorationBytes: 1 << 20,
		FailurePolicy: PolicyAll, FailureBudget: 1,
	}
}

func TestChoiceIdentitiesUseCompleteRepeatableStdlibProjections(t *testing.T) {
	config := choiceTestConfig()
	root, err := choiceNewCandidate(config, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	projection := candidateIdentityProjection{
		TargetSHA256:         record.SHA256FromSum(config.Execution.TargetSHA256),
		ToolchainBuildKey:    config.Execution.ToolchainBuildKey,
		GOOS:                 config.Execution.GOOS,
		GOARCH:               config.Execution.GOARCH,
		ImplementationSHA256: record.SHA256FromSum(config.Execution.ImplementationSHA256),
		ControllerSHA256:     config.ControllerSHA256,
		BaseSeed:             record.Uint64String(config.BaseSeed),
		Decisions:            []decisionIdentityProjection{},
	}
	encoded, err := json.Marshal(projection)
	if err != nil {
		t.Fatal(err)
	}
	if want := record.DomainHash(choiceCandidateDomain, encoded); root.SHA256 != want {
		t.Fatalf("candidate identity = %s, want stdlib projection %s", root.SHA256, want)
	}

	for range 32 {
		repeated, err := choiceNewCandidate(config, nil, "", "")
		if err != nil || repeated.SHA256 != root.SHA256 {
			t.Fatalf("repeated candidate = %#v, error = %v; want %s", repeated, err, root.SHA256)
		}
	}

	for _, test := range []struct {
		name   string
		change func(*ChoiceConfig)
	}{
		{name: "target", change: func(config *ChoiceConfig) { config.Execution.TargetSHA256 = sha256.Sum256([]byte("other target")) }},
		{name: "toolchain", change: func(config *ChoiceConfig) { config.Execution.ToolchainBuildKey = strings.Repeat("b", 64) }},
		{name: "goos", change: func(config *ChoiceConfig) { config.Execution.GOOS = "linux" }},
		{name: "goarch", change: func(config *ChoiceConfig) { config.Execution.GOARCH = "amd64" }},
		{name: "choice implementation", change: func(config *ChoiceConfig) {
			config.Execution.ImplementationSHA256 = sha256.Sum256([]byte("other choice implementation"))
		}},
		{name: "controller", change: func(config *ChoiceConfig) { config.ControllerSHA256 = record.HashBytes([]byte("other controller")) }},
		{name: "base seed", change: func(config *ChoiceConfig) { config.BaseSeed++ }},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := config
			test.change(&changed)
			candidate, err := choiceNewCandidate(changed, nil, "", "")
			if err != nil {
				t.Fatal(err)
			}
			if candidate.SHA256 == root.SHA256 {
				t.Fatalf("candidate identity did not bind %s", test.name)
			}
		})
	}
}

func TestChoiceRoundIdentityIsStableAcrossFrontierMapIteration(t *testing.T) {
	config := choiceTestConfig()
	initial, err := NewChoice(config)
	if err != nil {
		t.Fatal(err)
	}
	round, _ := initial.NextRound()
	trace := testTape(t, config.Execution,
		choiceTestDecision(t, 0, choice.KindRunnable, 8, 0),
		choiceTestDecision(t, 1, choice.KindRunnable, 8, 0),
	)
	result := []ChoiceResult{testResult(round.Candidates[0], trace, "repeatable frontier")}
	var wantState, wantSegment record.SHA256
	for iteration := range 64 {
		next, segment, err := CommitChoiceRound(initial, round, result)
		if err != nil {
			t.Fatal(err)
		}
		stateIdentity, err := ChoiceStateSHA256(next)
		if err != nil {
			t.Fatal(err)
		}
		if iteration == 0 {
			wantState, wantSegment = stateIdentity, segment.SHA256
			continue
		}
		if stateIdentity != wantState || segment.SHA256 != wantSegment {
			t.Fatalf("iteration %d identities = %s/%s, want %s/%s", iteration, stateIdentity, segment.SHA256, wantState, wantSegment)
		}
	}
}

func choiceTestDecision(t *testing.T, ordinal uint64, kind choice.Kind, alternatives, selected uint32) choice.Decision {
	t.Helper()
	return testSiteDecision(t, ordinal, kind, ordinal+1, alternatives, selected)
}

func testSiteDecision(t *testing.T, ordinal uint64, kind choice.Kind, site uint64, alternatives, selected uint32) choice.Decision {
	t.Helper()
	identities := make([][sha256.Size]byte, alternatives)
	for index := range identities {
		identities[index] = sha256.Sum256([]byte{byte(ordinal), byte(index + 1)})
	}
	decision, err := choice.CanonicalDecision(ordinal, kind, site, false, identities, identities[selected], 0)
	if err != nil {
		t.Fatal(err)
	}
	return decision
}

// testSelectResult is the observation a select records once it has locked
// its channels; origin is the record ordinal of its first poll decision.
func testSelectResult(t *testing.T, ordinal, origin, site uint64, polled uint32, readiness choice.SelectReadiness) choice.Record {
	t.Helper()
	word, err := readiness.Word()
	if err != nil {
		t.Fatal(err)
	}
	alternatives := polled
	if readiness.Default {
		alternatives++
	}
	return choice.Record{Ordinal: ordinal, Kind: choice.KindSelectResult, Flags: choice.FlagObservation, SiteOffset: site, Alternatives: alternatives, Data: polled, Readiness: word, Origin: origin}
}

func testTape(t *testing.T, identity choice.ExecutionIdentity, decisions ...choice.Decision) choice.ReplayPlan {
	t.Helper()
	records := make([]choice.Record, len(decisions))
	for index, decision := range decisions {
		records[index] = decision.Record()
	}
	return testTapeRecords(t, identity, records...)
}

func testTapeRecords(t *testing.T, identity choice.ExecutionIdentity, records ...choice.Record) choice.ReplayPlan {
	t.Helper()
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

func testResult(candidate ChoiceCandidate, trace choice.ReplayPlan, outcome string) ChoiceResult {
	return ChoiceResult{CandidateSHA256: candidate.SHA256, OutcomeSHA256: record.HashBytes([]byte(outcome)), Trace: &trace}
}

func segmentBytesWithoutIdentity(t *testing.T, segment ChoiceRoundSegment) []byte {
	t.Helper()
	segment.SHA256 = ""
	encoded, err := json.Marshal(segment)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestExplorationFailurePolicyBaseline(t *testing.T) {
	for _, test := range []struct {
		name      string
		policy    FailurePolicy
		budget    uint64
		distinct  bool
		maxRuns   uint64
		wantStop  ChoiceStopReason
		wantQueue int
		wantSeen  int
		wantOmit  uint64
		wantState record.SHA256
		wantRound record.SHA256
	}{
		{name: "all", policy: PolicyAll, budget: 1, distinct: true, maxRuns: 32, wantQueue: 5, wantSeen: 11,
			wantState: "sha256:f48d53744225258a34a532ca15e0292d2f754dd97dc0ad0f89b40f359402e274", wantRound: "sha256:7d28600fcf094403de165b531566d052db274969b46b10247e6b1036139cd495"},
		{name: "first", policy: PolicyFirst, budget: 1, distinct: true, maxRuns: 32, wantStop: ChoiceStopFirstFailure, wantQueue: 3, wantSeen: 9,
			wantState: "sha256:407f80eb5092dd492fcc390bd1db5b4ca3128d491b17ce7dc7daf83215347864", wantRound: "sha256:9f1c836d9bf4e13b1e63ebf3e78d9cc9f7e2fd6c462bb955570a021165b7449d"},
		{name: "budget repeated below threshold", policy: PolicyBudget, budget: 2, maxRuns: 32, wantQueue: 5, wantSeen: 11,
			wantState: "sha256:177bd2fc687016e812e1f8a47b8589cd4c4e428891590d68be8b5860a4a962bc", wantRound: "sha256:53bd08ff393f6e1586005afe70afc74593db8e101075c9bcedbe63d8fe3957e0"},
		{name: "budget distinct reaches threshold", policy: PolicyBudget, budget: 2, distinct: true, maxRuns: 32, wantStop: ChoiceStopFailureBudget, wantQueue: 3, wantSeen: 9,
			wantState: "sha256:08ad4ff0f575127c05464e6ae906f7d6c2661d0164e8e91b0c01225bd520f497", wantRound: "sha256:33be59b15b29244b3da86180dfb9d69a31c9428265af89e40c8fc0716813b43b"},
		{name: "budget distinct below threshold", policy: PolicyBudget, budget: 3, distinct: true, maxRuns: 32, wantQueue: 5, wantSeen: 11,
			wantState: "sha256:2bef8f84e0f38e2ae26951af29e41dec569efdcbb20222d140c4b17b97fb790a", wantRound: "sha256:98ecb4eaa79ad7251119653395ccf0f749f39fe3820ea711e6133c35951fc3c5"},
		{name: "budget one", policy: PolicyBudget, budget: 1, distinct: true, maxRuns: 32, wantStop: ChoiceStopFailureBudget, wantQueue: 3, wantSeen: 9,
			wantState: "sha256:12d9732da1d6082df7e2cb1fe447a92d6996e0a0e4d7582b4ef4d14254371612", wantRound: "sha256:be04ea1de1a4f223c0d3388484c09963318d601c5ada6e5ba6015d1b12c0b1fc"},
		{name: "first bounded children", policy: PolicyFirst, budget: 1, distinct: true, maxRuns: 8, wantStop: ChoiceStopFirstFailure, wantQueue: 2, wantSeen: 8, wantOmit: 1,
			wantState: "sha256:298c772dbd549e80b29f4e0e5e1c67a76670ec58a9b67d94bb82507642b1cf77", wantRound: "sha256:74f72d85ff1943ca9ab6a74acbb63f453f0326327d47ed2d686b8deca7474506"},
		{name: "manually invalid policy", policy: FailurePolicy("invalid"), budget: 1, distinct: true, maxRuns: 32, wantQueue: 5, wantSeen: 11,
			wantState: "sha256:113b7297b22907680f8895fbe22c2d806e998c33259fe3f4fcc713a7bf4495c4", wantRound: "sha256:90e943e164f78685fb3f2530e18dde4d51d5d32a50f715131092d9082893244f"},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, round, results := failurePolicyRound(t, test.policy, test.budget, test.maxRuns, test.distinct)
			next, segment, err := CommitChoiceRound(state, round, results)
			if err != nil {
				t.Fatal(err)
			}
			wantFailures := 1
			if test.distinct {
				wantFailures = 2
			}
			if next.StopReason != test.wantStop || next.LogicalExecutions != 6 || next.CommittedRounds != 2 || len(next.Outcomes) != 6 || len(next.FailureSignatures) != wantFailures || len(next.Queue) != test.wantQueue || len(next.Seen) != test.wantSeen || next.OmittedByExecutionBound != test.wantOmit {
				t.Fatalf("committed policy state = %#v", next)
			}
			_, available := next.NextRound()
			if available != (test.wantStop == "") {
				t.Fatalf("next round available = %t, stop = %q", available, next.StopReason)
			}
			if segment.Results[4].Failed || segment.Results[4].OutcomeSHA256 != results[4].OutcomeSHA256 || len(segment.Results[4].TraceBytes) == 0 {
				t.Fatalf("trailing success = %#v", segment.Results[4])
			}
			digest, err := ChoiceStateSHA256(next)
			if err != nil {
				t.Fatal(err)
			}
			if digest != test.wantState || segment.SHA256 != test.wantRound {
				t.Fatalf("state/segment identities = %s/%s, want %s/%s", digest, segment.SHA256, test.wantState, test.wantRound)
			}
			replayed, err := ChoiceReplaySegment(state, segment)
			if err != nil {
				t.Fatal(err)
			}
			replayedDigest, err := ChoiceStateSHA256(replayed)
			if err != nil || replayedDigest != digest {
				t.Fatalf("replayed identity = %q, error = %v", replayedDigest, err)
			}
		})
	}
}

func failurePolicyRound(t *testing.T, policy FailurePolicy, budget, maxRuns uint64, distinct bool) (ChoiceState, ChoiceRound, []ChoiceResult) {
	t.Helper()
	config := choiceTestConfig()
	config.Parallel, config.MaxExecutions = 5, maxRuns
	config.FailurePolicy, config.FailureBudget = policy, budget
	if policy == FailurePolicy("invalid") {
		config.FailurePolicy = PolicyAll
	}
	state, err := NewChoice(config)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := state.NextRound()
	trace := testTape(t, config.Execution, choiceTestDecision(t, 0, choice.KindRunnable, 7, 0))
	state, _, err = CommitChoiceRound(state, root, []ChoiceResult{testResult(root.Candidates[0], trace, "policy root")})
	if err != nil {
		t.Fatal(err)
	}
	state.Config.FailurePolicy = policy
	round, ok := state.NextRound()
	if !ok || len(round.Candidates) != 5 {
		t.Fatalf("policy round = %#v, available = %t", round, ok)
	}
	results := make([]ChoiceResult, 5)
	for index, candidate := range round.Candidates {
		results[index] = ChoiceResult{CandidateSHA256: candidate.SHA256, OutcomeSHA256: record.HashBytes([]byte{byte(index)})}
		if index >= 1 && index <= 3 {
			results[index].Failed = true
			results[index].FailureSHA256 = record.HashBytes([]byte("failure A"))
			if index == 3 && distinct {
				results[index].FailureSHA256 = record.HashBytes([]byte("failure B"))
			}
			continue
		}
		prefix, err := candidate.PrefixReplayPlan(config.Execution)
		if err != nil {
			t.Fatal(err)
		}
		var observed choice.Decision
		for selected := range uint32(7) {
			observed = choiceTestDecision(t, 0, choice.KindRunnable, 7, selected)
			if observed.Selected == prefix.Decisions[0].Selected {
				break
			}
		}
		childTrace := testTape(t, config.Execution, observed, choiceTestDecision(t, 1, choice.KindRunnable, 3, 0))
		results[index].Trace = &childTrace
	}
	return state, round, results
}

func TestExplorationValidatesSiblingsAfterPolicyStop(t *testing.T) {
	for _, policy := range []FailurePolicy{PolicyFirst, PolicyBudget} {
		for _, test := range []struct {
			name   string
			change func(*ChoiceResult)
			want   string
		}{
			{name: "candidate", change: func(result *ChoiceResult) { result.CandidateSHA256 = "" }, want: "result 4 does not match candidate"},
			{name: "outcome", change: func(result *ChoiceResult) { result.OutcomeSHA256 = "" }, want: "result 4 outcome:"},
			{name: "failure", change: func(result *ChoiceResult) { result.Failed = true; result.FailureSHA256 = "" }, want: "result 4 failure signature:"},
			{name: "trace", change: func(result *ChoiceResult) { result.Trace = &choice.ReplayPlan{} }, want: "validate choice exploration result 4 trace:"},
			{name: "prefix", change: func(result *ChoiceResult) {
				trace := testTape(t, choiceTestConfig().Execution, choiceTestDecision(t, 0, choice.KindRunnable, 7, 0))
				result.Trace = &trace
			}, want: "validate choice exploration result 4 prefix:"},
			{name: "divergence with outcome", change: func(result *ChoiceResult) { result.Divergence = &choice.Divergence{} }, want: "result 4 divergence contains outcome evidence"},
		} {
			t.Run(string(policy)+"/"+test.name, func(t *testing.T) {
				state, round, results := failurePolicyRound(t, policy, 1, 32, true)
				before, err := json.Marshal(state)
				if err != nil {
					t.Fatal(err)
				}
				test.change(&results[4])
				next, segment, err := CommitChoiceRound(state, round, results)
				if err == nil || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("post-stop sibling error = %v, want %q", err, test.want)
				}
				if next.Config.ControllerSHA256 != "" || len(next.Queue) != 0 || next.LogicalExecutions != 0 || segment.Schema != "" || len(segment.Results) != 0 {
					t.Fatalf("failed transaction returned state = %#v, segment = %#v", next, segment)
				}
				after, err := json.Marshal(state)
				if err != nil || !slices.Equal(before, after) {
					t.Fatalf("failed transaction mutated input, error = %v", err)
				}
			})
		}
	}
}

func TestExplorationRejectsInvalidFailurePolicyConfiguration(t *testing.T) {
	for _, test := range []struct {
		policy FailurePolicy
		budget uint64
		want   string
	}{
		{FailurePolicy("invalid"), 1, `unknown choice exploration failure policy "invalid"`},
		{PolicyBudget, 0, "choice exploration failure budget must be positive"},
		{PolicyFirst, 2, "choice exploration failure budget is only configurable in budget mode"},
		{PolicyAll, 2, "choice exploration failure budget is only configurable in budget mode"},
	} {
		t.Run(string(test.policy), func(t *testing.T) {
			config := choiceTestConfig()
			config.FailurePolicy, config.FailureBudget = test.policy, test.budget
			if _, err := NewChoice(config); err == nil || err.Error() != test.want {
				t.Fatalf("New() error = %v, want %q", err, test.want)
			}
		})
	}
}

func divergenceRound(t *testing.T, policy FailurePolicy) (ChoiceState, ChoiceRound, choice.Divergence) {
	t.Helper()
	config := choiceTestConfig()
	config.Parallel, config.FailurePolicy = 3, policy
	if policy == PolicyBudget {
		config.FailureBudget = 2
	}
	state, err := NewChoice(config)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := state.NextRound()
	trace := testTape(t, config.Execution, choiceTestDecision(t, 0, choice.KindRunnable, 5, 0))
	state, _, err = CommitChoiceRound(state, root, []ChoiceResult{testResult(root.Candidates[0], trace, "root")})
	if err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	prefix, err := round.Candidates[0].PrefixReplayPlan(config.Execution)
	if err != nil {
		t.Fatal(err)
	}
	expected := prefix.Decisions[0]
	observed := choiceTestDecision(t, 0, choice.KindRunnable, 5, expected.Selected)
	observed.AlternativeSetDigest = sha256.Sum256([]byte("changed alternatives"))
	return state, round, choice.Divergence{Ordinal: 0, Reason: choice.DivergenceAlternativeSet, Expected: &expected, Observed: &observed, TapeRecords: 1}
}

func TestExplorationCommitsDivergenceAndSiblingsUnderEveryPolicy(t *testing.T) {
	for _, policy := range []FailurePolicy{PolicyFirst, PolicyBudget, PolicyAll} {
		t.Run(string(policy), func(t *testing.T) {
			state, round, divergence := divergenceRound(t, policy)
			failure := record.HashBytes([]byte("failure"))
			results := []ChoiceResult{
				{CandidateSHA256: round.Candidates[0].SHA256, Divergence: &divergence},
				{CandidateSHA256: round.Candidates[1].SHA256, OutcomeSHA256: record.HashBytes([]byte("target failure")), Failed: true, FailureSHA256: failure},
				{CandidateSHA256: round.Candidates[2].SHA256, OutcomeSHA256: record.HashBytes([]byte("success"))},
			}
			next, segment, err := CommitChoiceRound(state, round, results)
			if err != nil {
				t.Fatal(err)
			}
			if next.LogicalExecutions != 4 || next.CommittedRounds != 2 || len(next.Outcomes) != 3 || !reflect.DeepEqual(next.FailureSignatures, []record.SHA256{failure}) || len(next.Queue) != 1 || len(next.Seen) != 5 {
				t.Fatalf("committed divergence = %#v", next)
			}
			if got := segment.Results[0]; got.OutcomeSHA256 != "" || got.Failed || got.FailureSHA256 != "" || len(got.TraceBytes) != 0 || !reflect.DeepEqual(got.Divergence.Divergence(), divergence) {
				t.Fatalf("divergent result = %#v", got)
			}
			_, available := next.NextRound()
			if policy == PolicyFirst && (available || next.StopReason != ChoiceStopFirstFailure) || policy != PolicyFirst && (!available || next.StopReason != "") {
				t.Fatalf("policy %s: %#v", policy, next)
			}
			replayed, err := ChoiceReplaySegment(state, segment)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(replayed, next) {
				t.Fatalf("replay = %#v, want %#v", replayed, next)
			}
			divergence.Observed.Alternatives++
			if segment.Results[0].Divergence.Observed.Alternatives != 5 {
				t.Fatal("segment aliases executor divergence")
			}
		})
	}
}

func TestExplorationRejectsInvalidCandidateDivergence(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*ChoiceResult)
	}{
		{"outcome", func(r *ChoiceResult) { r.OutcomeSHA256 = record.HashBytes([]byte("outcome")) }},
		{"failed", func(r *ChoiceResult) { r.Failed = true }},
		{"signature", func(r *ChoiceResult) { r.FailureSHA256 = record.HashBytes([]byte("signature")) }},
		{"trace", func(r *ChoiceResult) { r.Trace = &choice.ReplayPlan{} }},
		{"expected", func(r *ChoiceResult) { r.Divergence.Expected.Selected++ }},
		{"ordinal", func(r *ChoiceResult) { r.Divergence.Ordinal++ }},
		{"tape count", func(r *ChoiceResult) { r.Divergence.TapeRecords++ }},
		{"reason", func(r *ChoiceResult) { r.Divergence.Reason = choice.DivergenceSite }},
		{"missing expected", func(r *ChoiceResult) { r.Divergence.Expected = nil }},
		{"missing observed", func(r *ChoiceResult) { r.Divergence.Observed = nil }},
		{"observation beyond prefix boundary", func(r *ChoiceResult) {
			r.Divergence.Reason = choice.DivergenceObservation
			r.Divergence.Ordinal = 2
			r.Divergence.Expected = nil
			r.Divergence.Observed.Ordinal = 2
		}},
		{"observation at prefix boundary", func(r *ChoiceResult) {
			r.Divergence.Reason = choice.DivergenceObservation
			r.Divergence.Ordinal = 1
			r.Divergence.Expected = nil
			r.Divergence.Observed.Ordinal = 1
		}},
		{"observed ordinal", func(r *ChoiceResult) { r.Divergence.Observed.Ordinal++ }},
		{"rank override", func(r *ChoiceResult) { r.Divergence.Observed.RankOverride = true }},
		{"observed kind", func(r *ChoiceResult) { r.Divergence.Observed.Kind = 0 }},
		{"observed count", func(r *ChoiceResult) { r.Divergence.Observed.Alternatives = 0 }},
		{"observed selection", func(r *ChoiceResult) { r.Divergence.Observed.Selected = r.Divergence.Observed.Alternatives }},
		{"observed identity", func(r *ChoiceResult) { r.Divergence.Observed.SelectedIdentity = [32]byte{} }},
		{"observed digest", func(r *ChoiceResult) { r.Divergence.Observed.AlternativeSetDigest = [32]byte{} }},
		{"observed site", func(r *ChoiceResult) { r.Divergence.Observed.SiteMissing = true; r.Divergence.Observed.SiteOffset = 1 }},
		{"expected kind", func(r *ChoiceResult) { r.Divergence.Expected.Kind = 0 }},
		{"expected count", func(r *ChoiceResult) { r.Divergence.Expected.Alternatives = 0 }},
		{"expected rank override", func(r *ChoiceResult) { r.Divergence.Expected.RankOverride = false }},
		{"missing identity", func(r *ChoiceResult) {
			r.Divergence.Reason = choice.DivergenceIdentityMissing
			r.Divergence.Observed = nil
		}},
		{"duplicate identity", func(r *ChoiceResult) {
			r.Divergence.Reason = choice.DivergenceIdentityDuplicate
			r.Divergence.Observed = nil
		}},
		{"alternative capacity", func(r *ChoiceResult) {
			r.Divergence.Reason = choice.DivergenceAlternativeCapacity
			r.Divergence.Observed = nil
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, round, divergence := divergenceRound(t, PolicyAll)
			result := ChoiceResult{CandidateSHA256: round.Candidates[0].SHA256, Divergence: &divergence}
			test.change(&result)
			results := []ChoiceResult{result, {CandidateSHA256: round.Candidates[1].SHA256, OutcomeSHA256: record.HashBytes([]byte("success"))}, {CandidateSHA256: round.Candidates[2].SHA256, OutcomeSHA256: record.HashBytes([]byte("success"))}}
			if _, _, err := CommitChoiceRound(state, round, results); err == nil {
				t.Fatal("accepted invalid divergence")
			}
		})
	}
	state, _, divergence := divergenceRound(t, PolicyAll)
	initial, err := NewChoice(state.Config)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := initial.NextRound()
	if _, _, err := CommitChoiceRound(initial, root, []ChoiceResult{{CandidateSHA256: root.Candidates[0].SHA256, Divergence: &divergence}}); err == nil {
		t.Fatal("accepted root divergence")
	}
}

func TestExplorationRejectsPreviousSegmentSchemaVisibly(t *testing.T) {
	state, round, divergence := divergenceRound(t, PolicyAll)
	results := []ChoiceResult{{CandidateSHA256: round.Candidates[0].SHA256, Divergence: &divergence}, {CandidateSHA256: round.Candidates[1].SHA256, OutcomeSHA256: record.HashBytes([]byte("success"))}, {CandidateSHA256: round.Candidates[2].SHA256, OutcomeSHA256: record.HashBytes([]byte("success"))}}
	_, segment, err := CommitChoiceRound(state, round, results)
	if err != nil {
		t.Fatal(err)
	}
	segment.Schema = "gomad3.choice-exploration-round/v1"
	if _, err := ChoiceReplaySegment(state, segment); err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatalf("old schema error = %v", err)
	}
}
