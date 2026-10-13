package minimizer

import (
	"errors"
	"slices"

	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/exploration"
)

const choiceSchema = "gomad3.minimizer-state/choice-prefix/v1"

func ChoiceImplementationSHA256() record.SHA256 {
	return record.DomainHash("gomad3-minimizer-controller/choice-prefix/v1", []byte("effective-prefix-suffixes;deterministic;bounded/v1"))
}

func NewChoice(tape choice.ReplayPlan, attemptBudget uint64) (State, error) {
	if attemptBudget == 0 {
		return State{}, errors.New("minimizer attempt budget must be positive")
	}
	validated, err := choice.ValidateReplayPlan(tape, tape.Identity)
	if err != nil {
		return State{}, err
	}
	original, err := choiceCandidate(validated, uint64(len(validated.Decisions)), "")
	if err != nil {
		return State{}, err
	}
	return seal(State{Schema: choiceSchema, ChoiceTape: &validated, Original: original, Current: cloneCandidate(original), AttemptBudget: attemptBudget, Evaluated: []record.SHA256{}, Accepted: []Reduction{}})
}

func choiceCandidate(tape choice.ReplayPlan, length uint64, parent record.SHA256) (exploration.SimulationCandidate, error) {
	prefix, err := tape.Prefix(length)
	if err != nil {
		return exploration.SimulationCandidate{}, err
	}
	control := record.ChoicePrefixControl{Length: record.Uint64String(length), SHA256: record.SHA256FromSum(prefix.SHA256)}
	identity, err := control.CandidateSHA256()
	if err != nil {
		return exploration.SimulationCandidate{}, err
	}
	overrides := make([]exploration.ForcedDecision, length)
	for index, decision := range prefix.Decisions {
		overrides[index] = exploration.ForcedDecision{Dimension: exploration.DimensionRuntime, Ordinal: decision.Ordinal, Identity: record.SHA256FromSum(decision.SelectedIdentity)}
	}
	return exploration.SimulationCandidate{SHA256: identity, ParentSHA256: parent, Overrides: overrides}, nil
}

func choiceProposals(state State, seen map[record.SHA256]struct{}) ([]proposal, error) {
	for length := uint64(0); length < uint64(len(state.Current.Overrides)); length++ {
		candidate, err := choiceCandidate(*state.ChoiceTape, length, state.Current.SHA256)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[candidate.SHA256]; exists {
			continue
		}
		removed := make([]DecisionReference, len(state.Current.Overrides)-int(length))
		for index, override := range state.Current.Overrides[length:] {
			removed[index] = DecisionReference{Dimension: override.Dimension, Ordinal: override.Ordinal, Identity: override.Identity}
		}
		return []proposal{{Candidate: candidate, Reduction: Reduction{Kind: ReductionScheduleSuffix, BeforeSHA256: state.Current.SHA256, AfterSHA256: candidate.SHA256, Removed: removed}}}, nil
	}
	return nil, nil
}

func validateChoiceState(state State) error {
	if state.Config != (exploration.SimulationConfig{}) {
		return errors.New("choice minimizer cannot carry simulation configuration")
	}
	tape, err := choice.ValidateReplayPlan(*state.ChoiceTape, state.ChoiceTape.Identity)
	if err != nil {
		return err
	}
	if tape.SourceTraceSHA256 != state.ChoiceTape.SourceTraceSHA256 || !slices.Equal(tape.Decisions, state.ChoiceTape.Decisions) || !slices.Equal(tape.Readiness, state.ChoiceTape.Readiness) {
		return errors.New("choice minimizer cached tape differs from its bound bytes")
	}
	if state.Attempts > uint64(len(tape.Decisions)) || len(state.Accepted) > 1 {
		return errors.New("choice minimizer attempts exceed effective prefix search")
	}
	for index, identity := range state.Evaluated {
		candidate, err := choiceCandidate(tape, uint64(index), "")
		if err != nil {
			return err
		}
		if identity != candidate.SHA256 {
			return errors.New("choice minimizer evaluated prefixes are not in controller order")
		}
	}
	original, err := choiceCandidate(tape, uint64(len(tape.Decisions)), "")
	if err != nil {
		return err
	}
	if !sameCandidate(original, state.Original) {
		return errors.New("choice minimizer original is not its complete tape")
	}
	current := original
	for _, reduction := range state.Accepted {
		if reduction.Kind != ReductionScheduleSuffix || len(reduction.Removed) == 0 || len(reduction.Removed) > len(current.Overrides) {
			return errors.New("choice minimizer accepted an ineffective reduction")
		}
		length := len(current.Overrides) - len(reduction.Removed)
		next, err := choiceCandidate(tape, uint64(length), current.SHA256)
		if err != nil {
			return err
		}
		if reduction.BeforeSHA256 != current.SHA256 || reduction.AfterSHA256 != next.SHA256 || !slices.Contains(state.Evaluated, next.SHA256) {
			return errors.New("choice minimizer reduction identity changed")
		}
		for index, removed := range reduction.Removed {
			decision := current.Overrides[length+index]
			if removed != (DecisionReference{Dimension: decision.Dimension, Ordinal: decision.Ordinal, Identity: decision.Identity}) {
				return errors.New("choice minimizer removed controls are not its suffix")
			}
		}
		current = next
	}
	if !sameCandidate(current, state.Current) {
		return errors.New("choice minimizer current candidate does not follow its reductions")
	}
	if len(state.Accepted) != 0 && uint64(len(current.Overrides))+1 != state.Attempts {
		return errors.New("choice minimizer continued after its first accepted prefix")
	}
	minimal := len(state.Accepted) != 0 || state.Attempts == uint64(len(tape.Decisions))
	wantStop := StopReason("")
	if state.Attempts == state.AttemptBudget {
		wantStop = StopAttemptBudget
	} else if minimal {
		wantStop = StopMinimal
	}
	if state.StopReason != wantStop {
		return errors.New("choice minimizer stop does not match effective prefix search")
	}
	return nil
}
