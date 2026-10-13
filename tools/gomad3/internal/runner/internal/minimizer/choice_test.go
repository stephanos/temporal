package minimizer

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/model/choice"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/exploration"
)

func TestChoiceModePreservesNativeControllerAndStateEncoding(t *testing.T) {
	if ImplementationSHA256() != "sha256:4275065fb08afaf4186985787d5e7bb2f889d66ef221fcd988ce514a6cb26ba3" || ChoiceImplementationSHA256() != "sha256:a2513a63ea16ae06368fe29a7594c28198eaa9723f96b154ba9e1f74a0e8820f" {
		t.Fatal("native or versioned choice controller identity changed")
	}
	config := testConfig()
	state, err := New(config, testCandidate(t, config, testForced(t, exploration.DimensionRuntime, 0)), 8)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if state.Schema != "gomad3.minimizer-state/v1" || state.ChoiceTape != nil || bytes.Contains(encoded, []byte("choice_tape")) {
		t.Fatalf("native state acquired choice-mode encoding: %s", encoded)
	}
}

func choiceState(t *testing.T) State {
	t.Helper()
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256([]byte("target")), ToolchainBuildKey: "cbeccfefbc62a2ca026d9dded0316ecedfce33bd46b5c71b6645e86b67a0713e", GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte("controller"))}
	var records []choice.Record
	for index := uint64(0); index < 3; index++ {
		alternatives := [][32]byte{sha256.Sum256([]byte("first")), sha256.Sum256([]byte("second"))}
		decision, err := choice.CanonicalDecision(index, choice.KindRunnable, index+1, false, alternatives, alternatives[0], 0)
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, decision.Record())
	}
	trace, err := choice.BuildTrace(records, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	tape, err := choice.ProjectReplayPlan(trace, identity)
	if err != nil {
		t.Fatal(err)
	}
	state, err := NewChoice(tape, 8)
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func TestChoiceValidateRejectsResealedImpossibleState(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*State)
	}{
		{"unexecuted prefix", func(s *State) {
			s.Evaluated = []record.SHA256{record.HashBytes([]byte("not a prefix"))}
			s.Attempts = 1
		}},
		{"premature minimal", func(s *State) { s.StopReason = StopMinimal }},
		{"cached decision drift", func(s *State) { s.ChoiceTape.Decisions[0].Selected ^= 1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := choiceState(t)
			tc.change(&state)
			var err error
			state.SHA256, err = stateIdentity(state)
			if err != nil {
				t.Fatal(err)
			}
			if err := Validate(state); err == nil {
				t.Fatal("resealed impossible prefix state admitted")
			}
		})
	}
}

func TestChoiceProposalsReduceEffectiveLongestPrefixOnly(t *testing.T) {
	state := choiceState(t)
	for _, want := range []int{0, 1, 2} {
		attempt, ok, err := Next(state)
		if err != nil || !ok || len(attempt.Candidate.Overrides) != want || attempt.Reduction.Kind != ReductionScheduleSuffix || len(attempt.Reduction.Removed) != 3-want {
			t.Fatalf("effective suffix %d: %+v ok=%v err=%v", want, attempt, ok, err)
		}
		for index, removed := range attempt.Reduction.Removed {
			if removed.Ordinal != uint64(want+index) {
				t.Fatalf("earlier override removed while longest prefix survives: %+v", attempt.Reduction)
			}
		}
		state, err = Commit(state, attempt, want == 2)
		if err != nil {
			t.Fatal(err)
		}
	}
	if state.StopReason != StopMinimal || len(state.Current.Overrides) != 2 || state.Attempts != 3 || len(state.Accepted) != 1 {
		t.Fatalf("effective prefix state: %+v", state)
	}
}
