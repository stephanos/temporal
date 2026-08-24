package minimizer

import (
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
	simulationengine "go.temporal.io/server/tools/gomad3/runner/internal/exploration/simulation"
)

func TestStateReducesScheduleAndFaultEntriesDeterministically(t *testing.T) {
	config := testConfig()
	original := testCandidate(t, config,
		testForced(t, simulationengine.DimensionRuntime, 0),
		testForced(t, simulationengine.DimensionRuntime, 1),
		testForced(t, simulationengine.DimensionFault, 0),
		testForced(t, simulationengine.DimensionFault, 1),
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
		accepted := hasOverride(attempt.Candidate, simulationengine.DimensionRuntime, 0) && hasOverride(attempt.Candidate, simulationengine.DimensionFault, 0)
		state, err = Commit(state, attempt, accepted)
		if err != nil {
			t.Fatal(err)
		}
	}

	if state.StopReason != StopMinimal || state.Attempts == 0 || len(state.Accepted) != 2 {
		t.Fatalf("minimizer state = %#v", state)
	}
	if len(state.Current.Overrides) != 2 || !hasOverride(state.Current, simulationengine.DimensionRuntime, 0) || !hasOverride(state.Current, simulationengine.DimensionFault, 0) {
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
		testForced(t, simulationengine.DimensionRuntime, 0),
		testForced(t, simulationengine.DimensionFault, 0),
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
	state, err := New(config, testCandidate(t, config, testForced(t, simulationengine.DimensionRuntime, 0)), 2)
	if err != nil {
		t.Fatal(err)
	}
	state.Config.Parallel = 0
	state.SHA256, err = stateIdentity(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(state); err == nil {
		t.Fatal("Validate() accepted an invalid simulation-exploration config")
	}
}

func TestCloneStateDoesNotAliasNestedControlsOrReductions(t *testing.T) {
	config := testConfig()
	forced := testForced(t, simulationengine.DimensionRuntime, 0)
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

func testConfig() simulationengine.Config {
	return simulationengine.Config{
		ExecutionSHA256: record.HashBytes([]byte("execution")), ControllerSHA256: simulationengine.ImplementationSHA256(), BaseSeed: 7,
		Parallel: 1, MaxExecutions: 16, MaxForcedDecisions: 16, MaxExplorationBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 1,
		Limits: simulationengine.DimensionLimits{Runtime: 16, Scenario: 16, Network: 16, Storage: 16, Fault: 16, Crash: 16},
	}
}

func testForced(t *testing.T, dimension simulationengine.Dimension, ordinal uint64) simulationengine.ForcedDecision {
	t.Helper()
	forced, err := simulationengine.CanonicalForcedDecision(simulationengine.ForcedDecision{
		Dimension: dimension, Ordinal: ordinal, SiteSHA256: record.HashBytes([]byte(string(dimension) + "-site")),
		Alternatives: 2, AlternativeSetSHA256: record.HashBytes([]byte(string(dimension) + "-alternatives")),
		Selected: 1, SelectedSHA256: record.HashBytes([]byte(string(dimension) + "-selected")),
		Control: func() []byte {
			if dimension == simulationengine.DimensionRuntime {
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

func testCandidate(t *testing.T, config simulationengine.Config, overrides ...simulationengine.ForcedDecision) simulationengine.Candidate {
	t.Helper()
	candidate, err := simulationengine.CanonicalCandidate(config, overrides, "")
	if err != nil {
		t.Fatal(err)
	}
	return candidate
}

func hasOverride(candidate simulationengine.Candidate, dimension simulationengine.Dimension, ordinal uint64) bool {
	for _, override := range candidate.Overrides {
		if override.Dimension == dimension && override.Ordinal == ordinal {
			return true
		}
	}
	return false
}
