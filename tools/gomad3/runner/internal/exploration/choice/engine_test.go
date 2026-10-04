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

func TestExplorationStartOrdinalExpandsOnlyFromTheStart(t *testing.T) {
	config := testConfig()
	config.StartOrdinal = 2
	config.MaxChoiceDepth = 1
	state, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	trace := testTape(t, config.Execution,
		testDecision(t, 0, choice.KindRunnable, 2, 0),
		testDecision(t, 1, choice.KindSelectPoll, 3, 1),
		testDecision(t, 2, choice.KindRunnable, 2, 0),
		testDecision(t, 3, choice.KindRunnable, 3, 0),
	)
	state, _, err = CommitRound(state, round, []Result{testResult(round.Candidates[0], trace, "root")})
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
		want    StopReason
	}{
		{name: "before end", start: 1, pending: 1},
		{name: "at end", start: 2, want: StopStartUnreached},
		{name: "past end", start: 9, want: StopStartUnreached},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := testConfig()
			config.StartOrdinal = test.start
			state, err := New(config)
			if err != nil {
				t.Fatal(err)
			}
			round, _ := state.NextRound()
			trace := testTape(t, config.Execution, testDecision(t, 0, choice.KindRunnable, 2, 0), testDecision(t, 1, choice.KindRunnable, 2, 0))
			state, _, err = CommitRound(state, round, []Result{testResult(round.Candidates[0], trace, "root")})
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
	config := testConfig()
	trace := testTape(t, config.Execution, testDecision(t, 0, choice.KindRunnable, 2, 0), testDecision(t, 1, choice.KindRunnable, 2, 0))
	prefix, err := choice.BuildRankPrefix(trace, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	config.StartOrdinal = 2
	if _, err := newCandidate(config, &prefix, "", ""); err == nil || !strings.Contains(err.Error(), "before start ordinal 2") {
		t.Fatalf("newCandidate() error = %v", err)
	}
	config.StartOrdinal = 1
	if _, err := newCandidate(config, &prefix, "", ""); err != nil {
		t.Fatal(err)
	}
}

func TestExplorationSkipsNoOpSelectPollDecisions(t *testing.T) {
	config := testConfig()
	state, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	round, _ := state.NextRound()
	// A two-case select with one ready case between two runnable decisions.
	trace := testTapeRecords(t, config.Execution,
		testDecision(t, 0, choice.KindRunnable, 2, 0).Record(),
		testSiteDecision(t, 1, choice.KindSelectPoll, 40, 2, 1).Record(),
		testSelectResult(t, 2, 1, 40, 2, choice.SelectReadiness{Known: true, Ready: 1}),
		testDecision(t, 3, choice.KindRunnable, 3, 0).Record(),
	)
	state, _, err = CommitRound(state, round, []Result{testResult(round.Candidates[0], trace, "root")})
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
			config := testConfig()
			state, err := New(config)
			if err != nil {
				t.Fatal(err)
			}
			round, _ := state.NextRound()
			trace := testTapeRecords(t, config.Execution, test.records(t)...)
			state, _, err = CommitRound(state, round, []Result{testResult(round.Candidates[0], trace, test.name)})
			if err != nil {
				t.Fatal(err)
			}
			summary := state.Summary()
			if summary.Pending != test.pending || summary.OmittedBySelectReadiness != test.omitted {
				t.Fatalf("summary = %#v, want %d pending and %d omitted", summary, test.pending, test.omitted)
			}
			if test.pending == 0 && (summary.StopReason != StopExhausted || !summary.BoundedComplete) {
				t.Fatalf("summary = %#v, want exhaustion", summary)
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
	// The no-op select keeps the replayed round under the rule it was
	// committed with.
	trace := testTapeRecords(t, config.Execution,
		testDecision(t, 0, choice.KindRunnable, 3, 1).Record(),
		testSiteDecision(t, 1, choice.KindSelectPoll, 40, 2, 0).Record(),
		testSelectResult(t, 2, 1, 40, 2, choice.SelectReadiness{Known: true, Default: true}),
	)
	committed, segment, err := CommitRound(initial, round, []Result{testResult(round.Candidates[0], trace, "root")})
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := ReplaySegment(initial, segment)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.OmittedBySelectReadiness != 1 || len(replayed.Queue) != 2 {
		t.Fatalf("replayed state = %#v", replayed.Summary())
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

func TestExplorationFailurePolicyBaseline(t *testing.T) {
	for _, test := range []struct {
		name      string
		policy    FailurePolicy
		budget    uint64
		distinct  bool
		maxRuns   uint64
		wantStop  StopReason
		wantQueue int
		wantSeen  int
		wantOmit  uint64
		wantState record.SHA256
		wantRound record.SHA256
	}{
		{name: "all", policy: PolicyAll, budget: 1, distinct: true, maxRuns: 32, wantQueue: 5, wantSeen: 11,
			wantState: "sha256:29c4dc08452b7a9c12c30903218e5c5713bb0674d7942bd05d57711aa0cb8812", wantRound: "sha256:d79236e601b0d4d3d311c978546064b692d2aaad5d8cfcfe192a1744e2117a01"},
		{name: "first", policy: PolicyFirst, budget: 1, distinct: true, maxRuns: 32, wantStop: StopFirstFailure, wantQueue: 3, wantSeen: 9,
			wantState: "sha256:69d352959f1ebdaaec6279e5867daa536c2916089012ca7e12a9f4ee028779c7", wantRound: "sha256:cd9f2e71a90ba558444242bfca04c258d1ada489a8a19d74a283e0df3f0157da"},
		{name: "budget repeated below threshold", policy: PolicyBudget, budget: 2, maxRuns: 32, wantQueue: 5, wantSeen: 11,
			wantState: "sha256:ad662230644933921d3511069dbdd3eee5a80e5a4f4f643db927520e56e91572", wantRound: "sha256:8e90e14dab1db2dc893780a122765b46cd3568647f97ec13d3e178cc160dac83"},
		{name: "budget distinct reaches threshold", policy: PolicyBudget, budget: 2, distinct: true, maxRuns: 32, wantStop: StopFailureBudget, wantQueue: 3, wantSeen: 9,
			wantState: "sha256:37aa647022748626a86853c558a4b18a30c0b21e56be9eb1f740053040383dcf", wantRound: "sha256:259ad174f01508c79720a50b5780451ab28479ef7522ac1fbc3b7a0da6c89c9a"},
		{name: "budget distinct below threshold", policy: PolicyBudget, budget: 3, distinct: true, maxRuns: 32, wantQueue: 5, wantSeen: 11,
			wantState: "sha256:bfb984e3d3b5cadeb6284f88342c0ec010024bfc586d90d3e7baad4c72ce1724", wantRound: "sha256:04d682021b88e11f50290d9c480be9cb10aa95f25a055f4d258bd910e4ce20c4"},
		{name: "budget one", policy: PolicyBudget, budget: 1, distinct: true, maxRuns: 32, wantStop: StopFailureBudget, wantQueue: 3, wantSeen: 9,
			wantState: "sha256:1b1c5fff432f820a56902b1c7fe6e91a72aedfd696d6a8c943d7162f78a2d893", wantRound: "sha256:e0fe60a8086f9be356128259c1f6157eee1e719cb8e045c1a199c37431262114"},
		{name: "first bounded children", policy: PolicyFirst, budget: 1, distinct: true, maxRuns: 8, wantStop: StopFirstFailure, wantQueue: 2, wantSeen: 8, wantOmit: 1,
			wantState: "sha256:c6f2a7990fbb953456acc42b99915236936c8697457124702973efa76a8158a7", wantRound: "sha256:74925aad7beecf12e50bf04d61075c6491deeb501e3458f6d3dbd99695b1a1a4"},
		{name: "manually invalid policy", policy: FailurePolicy("invalid"), budget: 1, distinct: true, maxRuns: 32, wantQueue: 5, wantSeen: 11,
			wantState: "sha256:89bf69a4c85ef03f3663a2963e37a8c78d9ef15393b49bf58eca78c59b3e75f2", wantRound: "sha256:eb11a099847a5b68fe653378c0cd7c1741a0c2d8667205678944f11377b2f746"},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, round, results := failurePolicyRound(t, test.policy, test.budget, test.maxRuns, test.distinct)
			next, segment, err := CommitRound(state, round, results)
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
			digest, err := StateSHA256(next)
			if err != nil {
				t.Fatal(err)
			}
			if digest != test.wantState || segment.SHA256 != test.wantRound {
				t.Fatalf("state/segment identities = %s/%s, want %s/%s", digest, segment.SHA256, test.wantState, test.wantRound)
			}
			replayed, err := ReplaySegment(state, segment)
			if err != nil {
				t.Fatal(err)
			}
			replayedDigest, err := StateSHA256(replayed)
			if err != nil || replayedDigest != digest {
				t.Fatalf("replayed identity = %q, error = %v", replayedDigest, err)
			}
		})
	}
}

func failurePolicyRound(t *testing.T, policy FailurePolicy, budget, maxRuns uint64, distinct bool) (State, Round, []Result) {
	t.Helper()
	config := testConfig()
	config.Parallel, config.MaxExecutions = 5, maxRuns
	config.FailurePolicy, config.FailureBudget = policy, budget
	if policy == FailurePolicy("invalid") {
		config.FailurePolicy = PolicyAll
	}
	state, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := state.NextRound()
	trace := testTape(t, config.Execution, testDecision(t, 0, choice.KindRunnable, 7, 0))
	state, _, err = CommitRound(state, root, []Result{testResult(root.Candidates[0], trace, "policy root")})
	if err != nil {
		t.Fatal(err)
	}
	state.Config.FailurePolicy = policy
	round, ok := state.NextRound()
	if !ok || len(round.Candidates) != 5 {
		t.Fatalf("policy round = %#v, available = %t", round, ok)
	}
	results := make([]Result, 5)
	for index, candidate := range round.Candidates {
		results[index] = Result{CandidateSHA256: candidate.SHA256, OutcomeSHA256: record.HashBytes([]byte{byte(index)})}
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
			observed = testDecision(t, 0, choice.KindRunnable, 7, selected)
			if observed.Selected == prefix.Decisions[0].Selected {
				break
			}
		}
		childTrace := testTape(t, config.Execution, observed, testDecision(t, 1, choice.KindRunnable, 3, 0))
		results[index].Trace = &childTrace
	}
	return state, round, results
}

func TestExplorationValidatesSiblingsAfterPolicyStop(t *testing.T) {
	for _, policy := range []FailurePolicy{PolicyFirst, PolicyBudget} {
		for _, test := range []struct {
			name   string
			change func(*Result)
			want   string
		}{
			{name: "candidate", change: func(result *Result) { result.CandidateSHA256 = "" }, want: "result 4 does not match candidate"},
			{name: "outcome", change: func(result *Result) { result.OutcomeSHA256 = "" }, want: "result 4 outcome:"},
			{name: "failure", change: func(result *Result) { result.Failed = true; result.FailureSHA256 = "" }, want: "result 4 failure signature:"},
			{name: "trace", change: func(result *Result) { result.Trace = &choice.ReplayPlan{} }, want: "validate choice exploration result 4 trace:"},
			{name: "prefix", change: func(result *Result) {
				trace := testTape(t, testConfig().Execution, testDecision(t, 0, choice.KindRunnable, 7, 0))
				result.Trace = &trace
			}, want: "validate choice exploration result 4 prefix:"},
			{name: "divergence with outcome", change: func(result *Result) { result.Divergence = &choice.Divergence{} }, want: "result 4 divergence contains outcome evidence"},
		} {
			t.Run(string(policy)+"/"+test.name, func(t *testing.T) {
				state, round, results := failurePolicyRound(t, policy, 1, 32, true)
				before, err := canonicaljson.CanonicalJSON(state)
				if err != nil {
					t.Fatal(err)
				}
				test.change(&results[4])
				next, segment, err := CommitRound(state, round, results)
				if err == nil || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("post-stop sibling error = %v, want %q", err, test.want)
				}
				if next.Config.ControllerSHA256 != "" || len(next.Queue) != 0 || next.LogicalExecutions != 0 || segment.Schema != "" || len(segment.Results) != 0 {
					t.Fatalf("failed transaction returned state = %#v, segment = %#v", next, segment)
				}
				after, err := canonicaljson.CanonicalJSON(state)
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
			config := testConfig()
			config.FailurePolicy, config.FailureBudget = test.policy, test.budget
			if _, err := New(config); err == nil || err.Error() != test.want {
				t.Fatalf("New() error = %v, want %q", err, test.want)
			}
		})
	}
}
