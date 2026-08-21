package minimizer

import (
	"testing"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/combinedfrontier"
)

func TestStateReducesScheduleAndFaultEntriesDeterministically(t *testing.T) {
	config := testConfig()
	original := testCandidate(t, config,
		testForced(t, combinedfrontier.DimensionRuntime, 0),
		testForced(t, combinedfrontier.DimensionRuntime, 1),
		testForced(t, combinedfrontier.DimensionFault, 0),
		testForced(t, combinedfrontier.DimensionFault, 1),
	)
	state, err := New(config, original, 32)
	if err != nil {
		t.Fatal(err)
	}
	for {
		attempt, ok, err := Next(state)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			break
		}
		accepted := hasOverride(attempt.Candidate, combinedfrontier.DimensionRuntime, 0) && hasOverride(attempt.Candidate, combinedfrontier.DimensionFault, 0)
		state, err = Commit(state, attempt, accepted)
		if err != nil {
			t.Fatal(err)
		}
	}

	if state.StopReason != StopMinimal || state.Attempts == 0 || len(state.Accepted) != 2 {
		t.Fatalf("minimizer state = %#v", state)
	}
	if len(state.Current.Overrides) != 2 || !hasOverride(state.Current, combinedfrontier.DimensionRuntime, 0) || !hasOverride(state.Current, combinedfrontier.DimensionFault, 0) {
		t.Fatalf("minimized candidate = %#v", state.Current)
	}
	if state.Accepted[0].Kind != ReductionScheduleSuffix || state.Accepted[1].Kind != ReductionFaultEntries {
		t.Fatalf("accepted reductions = %#v", state.Accepted)
	}
	if err := Validate(state); err != nil {
		t.Fatal(err)
	}
}

func TestStateStopsAtAttemptBudgetAndRoundTrips(t *testing.T) {
	config := testConfig()
	original := testCandidate(t, config,
		testForced(t, combinedfrontier.DimensionRuntime, 0),
		testForced(t, combinedfrontier.DimensionFault, 0),
	)
	state, err := New(config, original, 1)
	if err != nil {
		t.Fatal(err)
	}
	attempt, ok, err := Next(state)
	if err != nil || !ok {
		t.Fatalf("Next() = %#v, %t, %v", attempt, ok, err)
	}
	state, err = Commit(state, attempt, false)
	if err != nil {
		t.Fatal(err)
	}
	if state.StopReason != StopAttemptBudget || state.Attempts != 1 {
		t.Fatalf("bounded state = %#v", state)
	}
	encoded, err := Encode(state)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.SHA256 != state.SHA256 || decoded.StopReason != state.StopReason {
		t.Fatalf("decoded state = %#v, want %#v", decoded, state)
	}
}

func TestValidateRejectsInvalidConfigWithSelfConsistentStateIdentity(t *testing.T) {
	config := testConfig()
	state, err := New(config, testCandidate(t, config, testForced(t, combinedfrontier.DimensionRuntime, 0)), 2)
	if err != nil {
		t.Fatal(err)
	}
	state.Config.Parallel = 0
	state.SHA256, err = stateIdentity(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(state); err == nil {
		t.Fatal("Validate() accepted an invalid combined-frontier config")
	}
}

func TestCloneStateDoesNotAliasNestedControlsOrReductions(t *testing.T) {
	config := testConfig()
	forced := testForced(t, combinedfrontier.DimensionRuntime, 0)
	state, err := New(config, testCandidate(t, config, forced), 2)
	if err != nil {
		t.Fatal(err)
	}
	state.Accepted = []Reduction{{
		Kind: ReductionScheduleRange, BeforeSHA256: state.Original.SHA256, AfterSHA256: state.Current.SHA256,
		Removed: []DecisionReference{{Dimension: forced.Dimension, Ordinal: forced.Ordinal, Identity: forced.Identity}},
	}}
	cloned := cloneState(state)
	cloned.Original.Overrides[0].Control[0]++
	cloned.Current.Overrides[0].Control[0]++
	cloned.Accepted[0].Removed[0].Ordinal++
	if state.Original.Overrides[0].Control[0] != 1 || state.Current.Overrides[0].Control[0] != 1 || state.Accepted[0].Removed[0].Ordinal != 0 {
		t.Fatalf("clone mutated source state = %#v", state)
	}
}

func testConfig() combinedfrontier.Config {
	return combinedfrontier.Config{
		ExecutionSHA256: evidence.HashBytes([]byte("execution")), ControllerSHA256: combinedfrontier.ImplementationSHA256(), BaseSeed: 7,
		Parallel: 1, MaxRuns: 16, MaxForcedDecisions: 16, MaxFrontierBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 1,
		Limits: combinedfrontier.DimensionLimits{Runtime: 16, Scenario: 16, Network: 16, Storage: 16, Fault: 16, Crash: 16},
	}
}

func testForced(t *testing.T, dimension combinedfrontier.Dimension, ordinal uint64) combinedfrontier.ForcedDecision {
	t.Helper()
	forced, err := combinedfrontier.CanonicalForcedDecision(combinedfrontier.ForcedDecision{
		Dimension: dimension, Ordinal: ordinal, SiteSHA256: evidence.HashBytes([]byte(string(dimension) + "-site")),
		Alternatives: 2, AlternativeSetSHA256: evidence.HashBytes([]byte(string(dimension) + "-alternatives")),
		Selected: 1, SelectedSHA256: evidence.HashBytes([]byte(string(dimension) + "-selected")),
		Control: func() []byte {
			if dimension == combinedfrontier.DimensionRuntime {
				return []byte{byte(ordinal + 1)}
			}
			return nil
		}(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return forced
}

func testCandidate(t *testing.T, config combinedfrontier.Config, overrides ...combinedfrontier.ForcedDecision) combinedfrontier.Candidate {
	t.Helper()
	candidate, err := combinedfrontier.CanonicalCandidate(config, overrides, "")
	if err != nil {
		t.Fatal(err)
	}
	return candidate
}

func hasOverride(candidate combinedfrontier.Candidate, dimension combinedfrontier.Dimension, ordinal uint64) bool {
	for _, override := range candidate.Overrides {
		if override.Dimension == dimension && override.Ordinal == ordinal {
			return true
		}
	}
	return false
}
