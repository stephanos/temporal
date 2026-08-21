package minimizer

import (
	"bytes"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/runner/internal/combinedfrontier"
)

const Schema = "gomadv3.minimizer-state/v1"

type ReductionKind string

const (
	ReductionScheduleSuffix ReductionKind = "schedule_suffix"
	ReductionScheduleRange  ReductionKind = "schedule_range"
	ReductionFaultEntries   ReductionKind = "fault_entries"
)

type StopReason string

const (
	StopMinimal       StopReason = "minimal"
	StopAttemptBudget StopReason = "attempt_budget"
)

type DecisionReference struct {
	Dimension combinedfrontier.Dimension `json:"dimension"`
	Ordinal   uint64                     `json:"ordinal"`
	Identity  evidence.SHA256            `json:"identity"`
}

type Reduction struct {
	Kind         ReductionKind       `json:"kind"`
	BeforeSHA256 evidence.SHA256     `json:"before_sha256"`
	AfterSHA256  evidence.SHA256     `json:"after_sha256"`
	Removed      []DecisionReference `json:"removed"`
}

type Attempt struct {
	Index     uint64                     `json:"index"`
	Reduction Reduction                  `json:"reduction"`
	Candidate combinedfrontier.Candidate `json:"candidate"`
}

type State struct {
	Schema        string                     `json:"schema"`
	Config        combinedfrontier.Config    `json:"config"`
	Original      combinedfrontier.Candidate `json:"original"`
	Current       combinedfrontier.Candidate `json:"current"`
	AttemptBudget uint64                     `json:"attempt_budget"`
	Attempts      uint64                     `json:"attempts"`
	Evaluated     []evidence.SHA256          `json:"evaluated"`
	Accepted      []Reduction                `json:"accepted"`
	StopReason    StopReason                 `json:"stop_reason,omitempty"`
	SHA256        evidence.SHA256            `json:"sha256"`
}

func ImplementationSHA256() evidence.SHA256 {
	return evidence.DomainHash("gomadv3-minimizer-controller/v1", []byte("suffix/ranges/faults;deterministic;bounded/v1"))
}

func New(config combinedfrontier.Config, original combinedfrontier.Candidate, attemptBudget uint64) (State, error) {
	if attemptBudget == 0 {
		return State{}, errors.New("minimizer attempt budget must be positive")
	}
	canonical, err := combinedfrontier.CanonicalCandidate(config, original.Overrides, original.ParentSHA256)
	if err != nil {
		return State{}, err
	}
	if !sameCandidate(canonical, original) {
		return State{}, errors.New("minimizer candidate identity does not match its contents")
	}
	state := State{
		Schema: Schema, Config: config, Original: cloneCandidate(canonical), Current: cloneCandidate(canonical),
		AttemptBudget: attemptBudget, Evaluated: []evidence.SHA256{}, Accepted: []Reduction{},
	}
	return seal(state)
}

func Next(state State) (Attempt, bool, error) {
	if err := Validate(state); err != nil {
		return Attempt{}, false, err
	}
	if state.StopReason != "" {
		return Attempt{}, false, nil
	}
	proposals, err := proposals(state)
	if err != nil {
		return Attempt{}, false, err
	}
	if len(proposals) == 0 {
		return Attempt{}, false, errors.New("active minimizer state has no proposal")
	}
	return Attempt{Index: state.Attempts, Reduction: proposals[0].Reduction, Candidate: proposals[0].Candidate}, true, nil
}

func Commit(state State, attempt Attempt, accepted bool) (State, error) {
	expected, ok, err := Next(state)
	if err != nil {
		return State{}, err
	}
	if !ok || !sameAttempt(expected, attempt) {
		return State{}, errors.New("minimizer attempt does not match the current state")
	}
	next := cloneState(state)
	next.Attempts++
	next.Evaluated = append(next.Evaluated, attempt.Candidate.SHA256)
	if accepted {
		next.Current = attempt.Candidate
		next.Accepted = append(next.Accepted, attempt.Reduction)
	}
	next.StopReason = ""
	next.SHA256 = ""
	return seal(next)
}

func Encode(state State) ([]byte, error) {
	if err := Validate(state); err != nil {
		return nil, err
	}
	return evidence.CanonicalJSON(state)
}

func Decode(encoded []byte) (State, error) {
	var state State
	if err := evidence.DecodeCanonicalJSON(encoded, &state); err != nil {
		return State{}, fmt.Errorf("decode minimizer state: %w", err)
	}
	if err := Validate(state); err != nil {
		return State{}, err
	}
	return state, nil
}

func Validate(state State) error {
	if state.Schema != Schema || state.AttemptBudget == 0 || state.Attempts > state.AttemptBudget || uint64(len(state.Evaluated)) != state.Attempts || uint64(len(state.Accepted)) > state.Attempts {
		return errors.New("minimizer state shape is invalid")
	}
	for _, candidate := range []combinedfrontier.Candidate{state.Original, state.Current} {
		canonical, err := combinedfrontier.CanonicalCandidate(state.Config, candidate.Overrides, candidate.ParentSHA256)
		if err != nil {
			return fmt.Errorf("validate minimizer candidate: %w", err)
		}
		if !sameCandidate(canonical, candidate) {
			return errors.New("minimizer candidate identity does not match its contents")
		}
	}
	for _, identity := range state.Evaluated {
		if _, err := identity.Bytes(); err != nil {
			return err
		}
	}
	for index, reduction := range state.Accepted {
		if err := validateReduction(reduction); err != nil {
			return fmt.Errorf("validate accepted reduction %d: %w", index, err)
		}
	}
	switch state.StopReason {
	case "", StopMinimal, StopAttemptBudget:
	default:
		return fmt.Errorf("unknown minimizer stop reason %q", state.StopReason)
	}
	want, err := stateIdentity(state)
	if err != nil {
		return err
	}
	if state.SHA256 != want {
		return errors.New("minimizer state identity changed")
	}
	if state.StopReason == StopAttemptBudget && state.Attempts != state.AttemptBudget {
		return errors.New("minimizer attempt-budget stop is premature")
	}
	return nil
}

type proposal struct {
	Reduction Reduction
	Candidate combinedfrontier.Candidate
}

func proposals(state State) ([]proposal, error) {
	seen := make(map[evidence.SHA256]struct{}, len(state.Evaluated)+1)
	seen[state.Current.SHA256] = struct{}{}
	for _, identity := range state.Evaluated {
		seen[identity] = struct{}{}
	}
	var result []proposal
	runtimeIndexes, scheduleIndexes, faultIndexes := proposalIndexes(state.Current)
	if err := appendRangeProposals(&result, seen, state, ReductionScheduleSuffix, runtimeIndexes, true); err != nil {
		return nil, err
	}
	if err := appendRangeProposals(&result, seen, state, ReductionScheduleRange, scheduleIndexes, false); err != nil {
		return nil, err
	}
	if err := appendRangeProposals(&result, seen, state, ReductionFaultEntries, faultIndexes, false); err != nil {
		return nil, err
	}
	return result, nil
}

func proposalIndexes(candidate combinedfrontier.Candidate) (runtime, schedule, fault []int) {
	for index, override := range candidate.Overrides {
		if override.Dimension == combinedfrontier.DimensionRuntime {
			runtime = append(runtime, index)
		}
		if override.Dimension == combinedfrontier.DimensionFault {
			fault = append(fault, index)
		} else {
			schedule = append(schedule, index)
		}
	}
	return runtime, schedule, fault
}

func appendRangeProposals(result *[]proposal, seen map[evidence.SHA256]struct{}, state State, kind ReductionKind, indexes []int, suffixOnly bool) error {
	for length := len(indexes); length > 0; length-- {
		maximumStart := len(indexes) - length
		firstStart := 0
		if suffixOnly {
			firstStart = maximumStart
		}
		for start := firstStart; start <= maximumStart; start++ {
			candidate, reduction, err := reducedCandidate(state.Config, state.Current, kind, indexes[start:start+length])
			if err != nil {
				return err
			}
			if _, exists := seen[candidate.SHA256]; exists {
				continue
			}
			seen[candidate.SHA256] = struct{}{}
			*result = append(*result, proposal{Reduction: reduction, Candidate: candidate})
		}
	}
	return nil
}

func reducedCandidate(config combinedfrontier.Config, current combinedfrontier.Candidate, kind ReductionKind, removedIndexes []int) (combinedfrontier.Candidate, Reduction, error) {
	removedSet := make(map[int]struct{}, len(removedIndexes))
	removed := make([]DecisionReference, len(removedIndexes))
	for index, candidateIndex := range removedIndexes {
		removedSet[candidateIndex] = struct{}{}
		decision := current.Overrides[candidateIndex]
		removed[index] = DecisionReference{Dimension: decision.Dimension, Ordinal: decision.Ordinal, Identity: decision.Identity}
	}
	overrides := make([]combinedfrontier.ForcedDecision, 0, len(current.Overrides)-len(removedIndexes))
	for index, override := range current.Overrides {
		if _, remove := removedSet[index]; !remove {
			overrides = append(overrides, override)
		}
	}
	candidate, err := combinedfrontier.CanonicalCandidate(config, overrides, current.SHA256)
	if err != nil {
		return combinedfrontier.Candidate{}, Reduction{}, err
	}
	reduction := Reduction{Kind: kind, BeforeSHA256: current.SHA256, AfterSHA256: candidate.SHA256, Removed: removed}
	return candidate, reduction, validateReduction(reduction)
}

func validateReduction(reduction Reduction) error {
	switch reduction.Kind {
	case ReductionScheduleSuffix, ReductionScheduleRange, ReductionFaultEntries:
	default:
		return fmt.Errorf("unknown minimizer reduction %q", reduction.Kind)
	}
	for _, identity := range []evidence.SHA256{reduction.BeforeSHA256, reduction.AfterSHA256} {
		if _, err := identity.Bytes(); err != nil {
			return err
		}
	}
	if reduction.BeforeSHA256 == reduction.AfterSHA256 || len(reduction.Removed) == 0 {
		return errors.New("minimizer reduction does not remove a decision")
	}
	for _, removed := range reduction.Removed {
		if _, err := removed.Identity.Bytes(); err != nil {
			return err
		}
	}
	return nil
}

func seal(state State) (State, error) {
	state.SHA256 = ""
	if state.Attempts == state.AttemptBudget {
		state.StopReason = StopAttemptBudget
	} else {
		proposals, err := proposals(state)
		if err != nil {
			return State{}, err
		}
		if len(proposals) == 0 {
			state.StopReason = StopMinimal
		}
	}
	identity, err := stateIdentity(state)
	if err != nil {
		return State{}, err
	}
	state.SHA256 = identity
	return state, Validate(state)
}

func stateIdentity(state State) (evidence.SHA256, error) {
	state.SHA256 = ""
	encoded, err := evidence.CanonicalJSON(state)
	if err != nil {
		return "", err
	}
	return evidence.DomainHash("gomadv3-minimizer-state/v1", encoded), nil
}

func sameAttempt(left, right Attempt) bool {
	leftBytes, leftErr := evidence.CanonicalJSON(left)
	rightBytes, rightErr := evidence.CanonicalJSON(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftBytes, rightBytes)
}

func sameCandidate(left, right combinedfrontier.Candidate) bool {
	leftBytes, leftErr := evidence.CanonicalJSON(left)
	rightBytes, rightErr := evidence.CanonicalJSON(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftBytes, rightBytes)
}

func cloneState(state State) State {
	state.Original = cloneCandidate(state.Original)
	state.Current = cloneCandidate(state.Current)
	state.Evaluated = append([]evidence.SHA256(nil), state.Evaluated...)
	accepted := make([]Reduction, len(state.Accepted))
	for index, reduction := range state.Accepted {
		accepted[index] = reduction
		accepted[index].Removed = append([]DecisionReference(nil), reduction.Removed...)
	}
	state.Accepted = accepted
	return state
}

func cloneCandidate(candidate combinedfrontier.Candidate) combinedfrontier.Candidate {
	overrides := make([]combinedfrontier.ForcedDecision, len(candidate.Overrides))
	for index, override := range candidate.Overrides {
		overrides[index] = override
		overrides[index].Control = append([]byte(nil), override.Control...)
	}
	candidate.Overrides = overrides
	return candidate
}
