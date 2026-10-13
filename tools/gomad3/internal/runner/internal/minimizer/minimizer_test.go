package minimizer

import (
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/exploration"
)

func TestStateReducesScheduleAndFaultEntriesDeterministically(t *testing.T) {
	config := testConfig()
	original := testCandidate(t, config,
		testForced(t, exploration.DimensionRuntime, 0),
		testForced(t, exploration.DimensionRuntime, 1),
		testForced(t, exploration.DimensionFault, 0),
		testForced(t, exploration.DimensionFault, 1),
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
		accepted := hasOverride(attempt.Candidate, exploration.DimensionRuntime, 0) && hasOverride(attempt.Candidate, exploration.DimensionFault, 0)
		state, err = Commit(state, attempt, accepted)
		if err != nil {
			t.Fatal(err)
		}
	}

	if state.StopReason != StopMinimal || state.Attempts == 0 || len(state.Accepted) != 2 {
		t.Fatalf("minimizer state = %#v", state)
	}
	if len(state.Current.Overrides) != 2 || !hasOverride(state.Current, exploration.DimensionRuntime, 0) || !hasOverride(state.Current, exploration.DimensionFault, 0) {
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
		testForced(t, exploration.DimensionRuntime, 0),
		testForced(t, exploration.DimensionFault, 0),
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
}

func TestValidateRejectsInvalidConfigWithSelfConsistentStateIdentity(t *testing.T) {
	config := testConfig()
	state, err := New(config, testCandidate(t, config, testForced(t, exploration.DimensionRuntime, 0)), 2)
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
	forced := testForced(t, exploration.DimensionRuntime, 0)
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

func TestStateIdentityBindsEveryStateInput(t *testing.T) {
	config := testConfig()
	state, err := New(config, testCandidate(t, config, testForced(t, exploration.DimensionRuntime, 0)), 8)
	if err != nil {
		t.Fatal(err)
	}
	mutations := []struct {
		name   string
		mutate func(*State)
	}{
		{name: "schema", mutate: func(state *State) { state.Schema += ".changed" }},
		{name: "config", mutate: func(state *State) { state.Config.BaseSeed++ }},
		{name: "original", mutate: func(state *State) { state.Original.ParentSHA256 = record.HashBytes([]byte("other original parent")) }},
		{name: "current", mutate: func(state *State) { state.Current.ParentSHA256 = record.HashBytes([]byte("other current parent")) }},
		{name: "attempt budget", mutate: func(state *State) { state.AttemptBudget++ }},
		{name: "attempts", mutate: func(state *State) { state.Attempts++ }},
		{name: "evaluated", mutate: func(state *State) { state.Evaluated = append(state.Evaluated, record.HashBytes([]byte("evaluated"))) }},
		{name: "accepted", mutate: func(state *State) {
			state.Accepted = append(state.Accepted, Reduction{
				Kind: ReductionScheduleSuffix, BeforeSHA256: record.HashBytes([]byte("before")), AfterSHA256: record.HashBytes([]byte("after")),
				Removed: []DecisionReference{{Dimension: exploration.DimensionRuntime, Identity: record.HashBytes([]byte("removed"))}},
			})
		}},
		{name: "stop reason", mutate: func(state *State) { state.StopReason = StopMinimal }},
	}
	for _, test := range mutations {
		t.Run(test.name, func(t *testing.T) {
			changed := cloneState(state)
			test.mutate(&changed)
			identity, err := stateIdentity(changed)
			if err != nil {
				t.Fatal(err)
			}
			if identity == state.SHA256 {
				t.Fatal("state input did not change its identity")
			}
		})
	}

	changedDigestOnly := cloneState(state)
	changedDigestOnly.SHA256 = record.HashBytes([]byte("not an identity input"))
	if identity, err := stateIdentity(changedDigestOnly); err != nil || identity != state.SHA256 {
		t.Fatalf("state identity included its own digest: %s, %v", identity, err)
	}
}

func TestStateIdentityRejectsInvalidOriginalString(t *testing.T) {
	config := testConfig()
	state, err := New(config, testCandidate(t, config, testForced(t, exploration.DimensionRuntime, 0)), 8)
	if err != nil {
		t.Fatal(err)
	}
	state.StopReason = StopReason(string([]byte{0xff}))
	if _, err := stateIdentity(state); err == nil {
		t.Fatal("stateIdentity() accepted invalid UTF-8")
	}
}

func testConfig() exploration.SimulationConfig {
	return exploration.SimulationConfig{
		ExecutionSHA256: record.HashBytes([]byte("execution")), ControllerSHA256: exploration.SimulationImplementationSHA256(), BaseSeed: 7,
		Parallel: 1, MaxExecutions: 16, MaxForcedDecisions: 16, MaxExplorationBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 1,
		Limits: exploration.DimensionLimits{Runtime: 16, Scenario: 16, Network: 16, Storage: 16, Fault: 16, Crash: 16},
	}
}

func testForced(t *testing.T, dimension exploration.Dimension, ordinal uint64) exploration.ForcedDecision {
	t.Helper()
	forced, err := exploration.CanonicalForcedDecision(exploration.ForcedDecision{
		Dimension: dimension, Ordinal: ordinal, SiteSHA256: record.HashBytes([]byte(string(dimension) + "-site")),
		Alternatives: 2, AlternativeSetSHA256: record.HashBytes([]byte(string(dimension) + "-alternatives")),
		Selected: 1, SelectedSHA256: record.HashBytes([]byte(string(dimension) + "-selected")),
		Control: func() []byte {
			if dimension == exploration.DimensionRuntime {
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

func testCandidate(t *testing.T, config exploration.SimulationConfig, overrides ...exploration.ForcedDecision) exploration.SimulationCandidate {
	t.Helper()
	candidate, err := exploration.CanonicalCandidate(config, overrides, "")
	if err != nil {
		t.Fatal(err)
	}
	return candidate
}

func hasOverride(candidate exploration.SimulationCandidate, dimension exploration.Dimension, ordinal uint64) bool {
	for _, override := range candidate.Overrides {
		if override.Dimension == dimension && override.Ordinal == ordinal {
			return true
		}
	}
	return false
}
