package protocol

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
)

const AttemptViewFormatVersion = "umpire3/attempt-view/v1"

type ActionOutcome string

const (
	ActionOutcomeApplied          ActionOutcome = "applied"
	ActionOutcomeSuppressed       ActionOutcome = "suppressed"
	ActionOutcomeRejected         ActionOutcome = "rejected"
	ActionOutcomeRetried          ActionOutcome = "retried"
	ActionOutcomeFaultIntercepted ActionOutcome = "fault-intercepted"
)

type AttemptOutcomeMapping struct {
	Outcome     ActionOutcome     `json:"outcome"`
	Guard       FirstOrderFormula `json:"guard"`
	Transitions []ActionKind      `json:"transitions"`
}

type AttemptMapping struct {
	Action   ActionKind              `json:"action"`
	Outcomes []AttemptOutcomeMapping `json:"outcomes"`
}

type AttemptView struct {
	FormatVersion          string             `json:"formatVersion"`
	Target                 TargetID           `json:"target"`
	Property               PropertyID         `json:"property"`
	World                  string             `json:"world"`
	Variant                string             `json:"variant"`
	SemanticHash           string             `json:"semanticHash"`
	FirstOrderSemanticHash string             `json:"firstOrderSemanticHash"`
	CanonicalModel         string             `json:"canonicalModel"`
	Relation               FirstOrderRelation `json:"relation"`
	Attempts               []AttemptMapping   `json:"attempts"`
}

type AttemptRequest struct {
	Action   ActionKind
	Outcomes []ActionOutcome
}

type ObservedAttempt struct {
	Action  ActionKind
	Outcome ActionOutcome
}

type AttemptReplay struct {
	Accepted        bool
	RejectedAction  ActionKind
	RejectedOutcome ActionOutcome
	LiveOnlyActions []ActionKind
}

func (v AttemptView) Outcomes(action ActionKind) ([]ActionOutcome, bool) {
	mapping, found := findAttemptMapping(v.Attempts, action)
	if !found {
		return nil, false
	}
	outcomes := make([]ActionOutcome, len(mapping.Outcomes))
	for index, outcome := range mapping.Outcomes {
		outcomes[index] = outcome.Outcome
	}
	return outcomes, true
}

//go:embed generated/nexus-cancellation.attempt.json
var defaultNexusAttemptJSON []byte

//go:embed generated/nexus-cancellation-mutated.attempt.json
var defaultNexusMutatedAttemptJSON []byte

func DecodeAttemptView(reader io.Reader, limit int64) (AttemptView, error) {
	var view AttemptView
	if err := decodeStrictJSON(reader, limit, "attempt view", &view); err != nil {
		return AttemptView{}, err
	}
	if err := view.Validate(); err != nil {
		return AttemptView{}, err
	}
	return view, nil
}

func DefaultAttemptView(target TargetID, variant string) (AttemptView, bool, error) {
	var encoded []byte
	switch {
	case target == TargetIDNexusCancellation && variant == "sound":
		encoded = defaultNexusAttemptJSON
	case target == TargetIDNexusCancellation && variant == "stale-completion-guard-removed":
		encoded = defaultNexusMutatedAttemptJSON
	default:
		return AttemptView{}, false, nil
	}
	view, err := DecodeAttemptView(bytes.NewReader(encoded), DefaultDecodeLimit)
	if err != nil {
		return AttemptView{}, true, err
	}
	firstOrder, found, err := DefaultFirstOrderView(target, variant)
	if err != nil || !found {
		return AttemptView{}, true, errors.New("attempt view has no matching first-order view")
	}
	if err := view.ValidateAgainst(firstOrder); err != nil {
		return AttemptView{}, true, err
	}
	return view, true, nil
}

func (v AttemptView) CanonicalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

func (v AttemptView) Validate() error {
	if v.FormatVersion != AttemptViewFormatVersion || v.Target == "" || v.Property == "" ||
		v.World == "" || v.Variant == "" || !validHash(v.SemanticHash) ||
		!validHash(v.FirstOrderSemanticHash) || v.CanonicalModel == "" || len(v.Attempts) == 0 {
		return errors.New("complete attempt view identity and provenance are required")
	}
	if err := v.Relation.validate(); err != nil {
		return err
	}
	actions := make(map[ActionKind]struct{}, len(v.Attempts))
	for _, attempt := range v.Attempts {
		if attempt.Action == "" || len(attempt.Outcomes) == 0 {
			return errors.New("complete attempt action and outcomes are required")
		}
		if _, duplicate := actions[attempt.Action]; duplicate {
			return fmt.Errorf("duplicate attempt action %q", attempt.Action)
		}
		actions[attempt.Action] = struct{}{}
		outcomes := make(map[ActionOutcome]struct{}, len(attempt.Outcomes))
		for _, mapping := range attempt.Outcomes {
			if !validActionOutcome(mapping.Outcome) {
				return fmt.Errorf("attempt action %q has unknown outcome %q", attempt.Action, mapping.Outcome)
			}
			if _, duplicate := outcomes[mapping.Outcome]; duplicate {
				return fmt.Errorf("attempt action %q has duplicate outcome %q", attempt.Action, mapping.Outcome)
			}
			outcomes[mapping.Outcome] = struct{}{}
			if mapping.Transitions == nil {
				return fmt.Errorf("attempt action %q outcome %q has no transition list", attempt.Action, mapping.Outcome)
			}
		}
	}
	return nil
}

func (v AttemptView) ValidateAgainst(firstOrder FirstOrderView) error {
	if err := v.Validate(); err != nil {
		return err
	}
	if err := firstOrder.Validate(); err != nil {
		return err
	}
	if v.Target != firstOrder.Target || v.Property != firstOrder.Property || v.World != firstOrder.World ||
		v.Variant != firstOrder.Variant || v.FirstOrderSemanticHash != firstOrder.SemanticHash ||
		v.CanonicalModel != firstOrder.CanonicalModel {
		return errors.New("attempt view identity does not match its first-order view")
	}
	sorts := make(map[string]FirstOrderSort, len(firstOrder.Sorts))
	for _, sort := range firstOrder.Sorts {
		sorts[sort.Identifier] = sort
	}
	fields := make(map[string]FirstOrderField, len(firstOrder.StateFields))
	for _, field := range firstOrder.StateFields {
		fields[field.Identifier] = field
	}
	modeled := make(map[ActionKind]struct{}, len(firstOrder.Actions))
	expected := make([]ActionKind, 0, len(firstOrder.Actions)+len(firstOrder.LiveOnlyActions))
	for _, action := range firstOrder.Actions {
		identifier := ActionKind(action.Identifier)
		modeled[identifier] = struct{}{}
		expected = append(expected, identifier)
	}
	expected = append(expected, firstOrder.LiveOnlyActions...)
	actual := make([]ActionKind, len(v.Attempts))
	for index, attempt := range v.Attempts {
		actual[index] = attempt.Action
		applied := false
		for _, mapping := range attempt.Outcomes {
			if err := validateFirstOrderFormula(mapping.Guard, fields, sorts); err != nil {
				return fmt.Errorf("validate attempt action %q outcome %q guard: %w",
					attempt.Action, mapping.Outcome, err)
			}
			for _, transition := range mapping.Transitions {
				if _, exists := modeled[transition]; !exists {
					return fmt.Errorf("attempt action %q outcome %q references unknown abstract transition %q",
						attempt.Action, mapping.Outcome, transition)
				}
			}
			if mapping.Outcome == ActionOutcomeApplied {
				expectedTransitions := []ActionKind{}
				if _, exists := modeled[attempt.Action]; exists {
					expectedTransitions = []ActionKind{attempt.Action}
				}
				if !slices.Equal(mapping.Transitions, expectedTransitions) {
					return fmt.Errorf("attempt action %q applied outcome has transitions %v, expected %v",
						attempt.Action, mapping.Transitions, expectedTransitions)
				}
				applied = true
			}
		}
		if !applied {
			return fmt.Errorf("attempt action %q has no applied outcome", attempt.Action)
		}
	}
	if !slices.Equal(actual, expected) {
		return fmt.Errorf("attempt actions %v do not exactly cover first-order and live-only actions %v",
			actual, expected)
	}
	return nil
}

func (v AttemptView) Replay(firstOrder FirstOrderView, requests []AttemptRequest) (AttemptReplay, error) {
	if err := v.ValidateAgainst(firstOrder); err != nil {
		return AttemptReplay{}, err
	}
	if len(requests) == 0 {
		return AttemptReplay{}, errors.New("attempt replay requires actions")
	}
	states, err := initialFirstOrderRuntimeStates(firstOrder)
	if err != nil {
		return AttemptReplay{}, err
	}
	liveOnlySet := make(map[ActionKind]struct{}, len(firstOrder.LiveOnlyActions))
	for _, action := range firstOrder.LiveOnlyActions {
		liveOnlySet[action] = struct{}{}
	}
	var liveOnly []ActionKind
	for _, request := range requests {
		attempt, found := findAttemptMapping(v.Attempts, request.Action)
		if !found {
			return AttemptReplay{}, fmt.Errorf("action %q has no Lean-derived attempt mapping", request.Action)
		}
		outcomes := request.Outcomes
		if len(outcomes) == 0 {
			outcomes = make([]ActionOutcome, len(attempt.Outcomes))
			for index, mapping := range attempt.Outcomes {
				outcomes[index] = mapping.Outcome
			}
		}
		if err := validateRequestedOutcomes(attempt, outcomes); err != nil {
			return AttemptReplay{}, err
		}
		nextStates := make([]firstOrderRuntimeState, 0, len(states)*len(outcomes))
		seen := make(map[string]struct{}, len(states)*len(outcomes))
		for _, state := range states {
			for _, outcome := range outcomes {
				mapping, _ := findAttemptOutcome(attempt.Outcomes, outcome)
				next, accepted := applyAttemptOutcome(firstOrder, state, mapping)
				if !accepted {
					continue
				}
				key := next.key(firstOrder.StateFields)
				if _, duplicate := seen[key]; duplicate {
					continue
				}
				seen[key] = struct{}{}
				nextStates = append(nextStates, next)
			}
		}
		if len(nextStates) == 0 {
			rejectedOutcome := ActionOutcome("")
			if len(outcomes) == 1 {
				rejectedOutcome = outcomes[0]
			}
			return AttemptReplay{
				RejectedAction: request.Action, RejectedOutcome: rejectedOutcome,
				LiveOnlyActions: sortedCompactActions(liveOnly),
			}, nil
		}
		if _, live := liveOnlySet[request.Action]; live {
			liveOnly = append(liveOnly, request.Action)
		}
		states = nextStates
	}
	return AttemptReplay{Accepted: true, LiveOnlyActions: sortedCompactActions(liveOnly)}, nil
}

func (v AttemptView) ReplayObserved(
	firstOrder FirstOrderView,
	attempts []ObservedAttempt,
) (AttemptReplay, error) {
	requests := make([]AttemptRequest, len(attempts))
	for index, attempt := range attempts {
		requests[index] = AttemptRequest{Action: attempt.Action, Outcomes: []ActionOutcome{attempt.Outcome}}
	}
	return v.Replay(firstOrder, requests)
}

func initialFirstOrderRuntimeStates(view FirstOrderView) ([]firstOrderRuntimeState, error) {
	states := make([]firstOrderRuntimeState, 0, len(view.Oracle.States))
	for _, state := range view.Oracle.States {
		runtimeState, err := newFirstOrderRuntimeState(view.StateFields, state)
		if err != nil {
			return nil, err
		}
		if evaluateFirstOrderFormula(runtimeState, view.Initial) {
			states = append(states, runtimeState)
		}
	}
	if len(states) == 0 {
		return nil, errors.New("first-order view has no oracle initial state")
	}
	return states, nil
}

func applyAttemptOutcome(
	view FirstOrderView,
	state firstOrderRuntimeState,
	mapping AttemptOutcomeMapping,
) (firstOrderRuntimeState, bool) {
	if !evaluateFirstOrderFormula(state, mapping.Guard) {
		return nil, false
	}
	next := state
	for _, identifier := range mapping.Transitions {
		action, found := findFirstOrderAction(view.Actions, string(identifier))
		if !found || !evaluateFirstOrderFormula(next, action.Guard) {
			return nil, false
		}
		next = applyFirstOrderUpdates(next, action.Updates)
	}
	return next, true
}

func validateRequestedOutcomes(mapping AttemptMapping, outcomes []ActionOutcome) error {
	seen := make(map[ActionOutcome]struct{}, len(outcomes))
	for _, outcome := range outcomes {
		if _, duplicate := seen[outcome]; duplicate {
			return fmt.Errorf("action %q repeats requested outcome %q", mapping.Action, outcome)
		}
		seen[outcome] = struct{}{}
		if _, found := findAttemptOutcome(mapping.Outcomes, outcome); !found {
			return fmt.Errorf("action %q outcome %q is not declared by its Lean-derived attempt mapping",
				mapping.Action, outcome)
		}
	}
	return nil
}

func findAttemptMapping(mappings []AttemptMapping, action ActionKind) (AttemptMapping, bool) {
	for _, mapping := range mappings {
		if mapping.Action == action {
			return mapping, true
		}
	}
	return AttemptMapping{}, false
}

func findAttemptOutcome(mappings []AttemptOutcomeMapping, outcome ActionOutcome) (AttemptOutcomeMapping, bool) {
	for _, mapping := range mappings {
		if mapping.Outcome == outcome {
			return mapping, true
		}
	}
	return AttemptOutcomeMapping{}, false
}

func validActionOutcome(outcome ActionOutcome) bool {
	return slices.Contains([]ActionOutcome{
		ActionOutcomeApplied,
		ActionOutcomeSuppressed,
		ActionOutcomeRejected,
		ActionOutcomeRetried,
		ActionOutcomeFaultIntercepted,
	}, outcome)
}
