package combinedfrontier

import (
	"slices"
	"testing"

	"go.temporal.io/server/tools/gomadv3/evidence"
)

func TestCombinedFrontierExpandsEveryDimensionBreadthFirstAndDeduplicatesEvidenceOnly(t *testing.T) {
	config := testConfig()
	config.Parallel = 16
	state, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	root, ok := state.NextRound()
	if !ok || len(root.Candidates) != 1 || len(root.Candidates[0].Overrides) != 0 {
		t.Fatalf("root round = %#v, ok=%t", root, ok)
	}
	decisions := []Decision{
		testDecision(t, DimensionRuntime, 0, 2, 0),
		testDecision(t, DimensionScenario, 0, 3, 1),
		testDecision(t, DimensionNetwork, 0, 2, 0),
		testDecision(t, DimensionStorage, 0, 2, 0),
		testDecision(t, DimensionFault, 0, 2, 1),
		testDecision(t, DimensionCrash, 0, 2, 1),
	}
	sharedOutcome := evidence.HashBytes([]byte("shared outcome"))
	state, _, err = CommitRound(state, root, []Result{{
		CandidateSHA256: root.Candidates[0].SHA256,
		OutcomeSHA256:   sharedOutcome,
		Decisions:       decisions,
	}})
	if err != nil {
		t.Fatal(err)
	}
	summary := state.Summary()
	if summary.Pending != 7 || summary.SeenCandidates != 8 || summary.DeepestOverride != 1 || summary.DeduplicatedOutcomes != 1 {
		t.Fatalf("root expansion summary = %#v", summary)
	}
	for _, candidate := range state.Queue {
		if len(candidate.Overrides) != 1 {
			t.Fatalf("candidate is not breadth one: %#v", candidate)
		}
	}

	next, ok := state.NextRound()
	if !ok || len(next.Candidates) != 7 {
		t.Fatalf("next round = %#v, ok=%t", next, ok)
	}
	results := make([]Result, len(next.Candidates))
	for index, candidate := range next.Candidates {
		childDecisions := append([]Decision(nil), decisions...)
		if index == 0 {
			childDecisions = append(childDecisions, testDecision(t, DimensionStorage, 1, 2, 0))
		}
		childDecisions = forceObservedSelections(childDecisions, candidate.Overrides)
		results[index] = Result{CandidateSHA256: candidate.SHA256, OutcomeSHA256: sharedOutcome, Decisions: childDecisions}
	}
	state, _, err = CommitRound(state, next, results)
	if err != nil {
		t.Fatal(err)
	}
	summary = state.Summary()
	if summary.DeduplicatedOutcomes != 1 || summary.Pending == 0 || summary.SeenCandidates <= 8 {
		t.Fatalf("semantic deduplication pruned search: %#v", summary)
	}
	for _, candidate := range state.Queue {
		if len(candidate.Overrides) < 2 {
			t.Fatalf("queue is not breadth-first after the depth-one round: %#v", state.Queue)
		}
	}
}

func TestCombinedFrontierRejectsUnprovenForcedDecisionUnlessCandidateDiverged(t *testing.T) {
	state, err := New(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	root, _ := state.NextRound()
	decision := testDecision(t, DimensionScenario, 0, 2, 0)
	state, _, err = CommitRound(state, root, []Result{{
		CandidateSHA256: root.Candidates[0].SHA256,
		OutcomeSHA256:   evidence.HashBytes([]byte("root")),
		Decisions:       []Decision{decision},
	}})
	if err != nil {
		t.Fatal(err)
	}
	child, _ := state.NextRound()
	if len(child.Candidates) != 1 {
		t.Fatalf("child round = %#v", child)
	}
	if _, _, err := CommitRound(state, child, []Result{{
		CandidateSHA256: child.Candidates[0].SHA256,
		OutcomeSHA256:   evidence.HashBytes([]byte("missing override")),
	}}); err == nil {
		t.Fatal("CommitRound() accepted a result that did not prove its forced decision")
	}
	if _, _, err := CommitRound(state, child, []Result{{
		CandidateSHA256: child.Candidates[0].SHA256,
		OutcomeSHA256:   evidence.HashBytes([]byte("typed divergence")),
		Diverged:        true,
	}}); err != nil {
		t.Fatalf("CommitRound() rejected a typed control divergence: %v", err)
	}
}

func TestCombinedFrontierEnforcesDimensionDepthRunAndByteBounds(t *testing.T) {
	for _, test := range []struct {
		name      string
		configure func(*Config, State)
		decision  func(*testing.T) Decision
		want      StopReason
		omitted   func(Summary) uint64
	}{
		{
			name: "runs", configure: func(config *Config, _ State) { config.MaxRuns = 1 },
			decision: func(t *testing.T) Decision { return testDecision(t, DimensionRuntime, 0, 2, 0) },
			want:     StopMaxRuns, omitted: func(summary Summary) uint64 { return summary.OmittedByRunBound },
		},
		{
			name: "combined depth", configure: func(config *Config, _ State) { config.MaxForcedDecisions = 0 },
			decision: func(t *testing.T) Decision { return testDecision(t, DimensionRuntime, 0, 2, 0) },
			want:     StopDepthComplete, omitted: func(summary Summary) uint64 { return summary.OmittedByDepth },
		},
		{
			name: "dimension", configure: func(config *Config, _ State) { config.Limits.Runtime = 1 },
			decision: func(t *testing.T) Decision { return testDecision(t, DimensionRuntime, 1, 2, 0) },
			want:     StopDimensionComplete, omitted: func(summary Summary) uint64 { return summary.OmittedByDimension },
		},
		{
			name: "capacity", configure: func(config *Config, initial State) { config.MaxFrontierBytes = initial.Summary().PendingBytes },
			decision: func(t *testing.T) Decision { return testDecision(t, DimensionRuntime, 0, 2, 0) },
			want:     StopFrontierCapacity, omitted: func(summary Summary) uint64 { return summary.OmittedByCapacity },
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
			state, _, err = CommitRound(state, round, []Result{{
				CandidateSHA256: round.Candidates[0].SHA256,
				OutcomeSHA256:   evidence.HashBytes([]byte(test.name)),
				Decisions:       []Decision{test.decision(t)},
			}})
			if err != nil {
				t.Fatal(err)
			}
			summary := state.Summary()
			if summary.StopReason != test.want || test.omitted(summary) == 0 {
				t.Fatalf("summary = %#v, want stop %q with omissions", summary, test.want)
			}
		})
	}
}

func TestCombinedFrontierRoundSegmentReplaysByteIdentically(t *testing.T) {
	initial, err := New(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	round, _ := initial.NextRound()
	committed, segment, err := CommitRound(initial, round, []Result{{
		CandidateSHA256: round.Candidates[0].SHA256,
		OutcomeSHA256:   evidence.HashBytes([]byte("root")),
		Decisions:       []Decision{testDecision(t, DimensionFault, 0, 3, 1)},
	}})
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := ReplaySegment(initial, segment)
	if err != nil {
		t.Fatal(err)
	}
	left, err := evidence.CanonicalJSON(committed)
	if err != nil {
		t.Fatal(err)
	}
	right, err := evidence.CanonicalJSON(replayed)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(left, right) {
		t.Fatalf("replayed state differs:\n%s\n%s", left, right)
	}
	segment.Results[0].OutcomeSHA256 = evidence.HashBytes([]byte("corrupt"))
	if _, err := ReplaySegment(initial, segment); err == nil {
		t.Fatal("ReplaySegment() accepted a corrupted round")
	}
}

func TestValidateCandidateRejectsChangedForcedDecision(t *testing.T) {
	config := testConfig()
	state, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	round, ok := state.NextRound()
	if !ok {
		t.Fatal("combined frontier root round is unavailable")
	}
	decision := testDecision(t, DimensionScenario, 0, 2, 0)
	next, _, err := CommitRound(state, round, []Result{{
		CandidateSHA256: round.Candidates[0].SHA256, OutcomeSHA256: evidence.HashBytes([]byte("baseline")), Decisions: []Decision{decision},
	}})
	if err != nil {
		t.Fatal(err)
	}
	childRound, ok := next.NextRound()
	if !ok || len(childRound.Candidates) != 1 {
		t.Fatal("combined frontier child candidate is unavailable")
	}
	candidate := childRound.Candidates[0]
	if err := ValidateCandidate(config, candidate); err != nil {
		t.Fatalf("ValidateCandidate() error = %v", err)
	}
	candidate.Overrides[0].Selected = decision.Selected
	if err := ValidateCandidate(config, candidate); err == nil {
		t.Fatal("ValidateCandidate() accepted a changed forced decision")
	}
}

func testConfig() Config {
	return Config{
		ExecutionSHA256:    evidence.HashBytes([]byte("execution")),
		ControllerSHA256:   ImplementationSHA256(),
		BaseSeed:           7,
		Parallel:           8,
		MaxRuns:            64,
		MaxForcedDecisions: 8,
		MaxFrontierBytes:   1 << 20,
		MaxResultBytes:     1 << 20,
		FailureBudget:      8,
		Limits:             DimensionLimits{Runtime: 8, Scenario: 8, Network: 8, Storage: 8, Fault: 8, Crash: 8},
	}
}

func testDecision(t *testing.T, dimension Dimension, ordinal uint64, alternatives, selected uint32) Decision {
	t.Helper()
	identities := make([]evidence.SHA256, alternatives)
	for index := range identities {
		identities[index] = evidence.HashBytes([]byte{byte(dimensionOrder(dimension)), byte(ordinal), byte(index + 1)})
	}
	var controls [][]byte
	if dimension == DimensionRuntime {
		controls = make([][]byte, alternatives)
		for rank := range controls {
			if uint32(rank) != selected {
				controls[rank] = []byte{byte(rank + 1)}
			}
		}
	}
	decision, err := CanonicalControlledDecision(dimension, ordinal, evidence.HashBytes([]byte("site-"+string(dimension))), identities, controls, selected)
	if err != nil {
		t.Fatal(err)
	}
	return decision
}

func forceObservedSelections(decisions []Decision, overrides []ForcedDecision) []Decision {
	cloned := cloneDecisions(decisions)
	for index := range cloned {
		for _, override := range overrides {
			if override.Dimension == cloned[index].Dimension && override.Ordinal == cloned[index].Ordinal {
				cloned[index].Selected = override.Selected
				if cloned[index].Dimension == DimensionRuntime {
					for rank := range cloned[index].AlternativeControls {
						cloned[index].AlternativeControls[rank] = []byte{byte(rank + 1)}
					}
					cloned[index].AlternativeControls[override.Selected] = nil
				}
				cloned[index].Identity, _ = decisionIdentity(cloned[index])
			}
		}
	}
	return cloned
}
