package checker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
)

const AttemptViewFormatVersion = "umpire3/attempt-view/v1"

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

func validActionOutcome(outcome ActionOutcome) bool {
	return slices.Contains([]ActionOutcome{
		ActionOutcomeApplied,
		ActionOutcomeSuppressed,
		ActionOutcomeRejected,
		ActionOutcomeRetried,
		ActionOutcomeFaultIntercepted,
	}, outcome)
}
