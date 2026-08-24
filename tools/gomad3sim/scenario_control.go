package gomad3sim

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"unicode/utf8"
)

const ScenarioChoicePlanSchema = "gomad3.scenario-choice-plan/v1"
const MaximumScenarioChoicePlanBytes = 16 << 20

type ScenarioChoiceOverride struct {
	Ordinal      uint64   `json:"ordinal"`
	ID           string   `json:"id"`
	Occurrence   uint64   `json:"occurrence"`
	Alternatives []string `json:"alternatives"`
	Selected     uint64   `json:"selected"`
	Identity     string   `json:"identity"`
}

type ScenarioChoicePlan struct {
	Schema    string                   `json:"schema"`
	Overrides []ScenarioChoiceOverride `json:"overrides"`
	Identity  string                   `json:"identity"`
}

func NewScenarioChoicePlan(overrides []ScenarioChoiceOverride) (ScenarioChoicePlan, error) {
	if err := checkCapacity("scenario_choice_overrides", uint64(len(overrides)), MaximumScenarioDecisions); err != nil {
		return ScenarioChoicePlan{}, err
	}
	plan := ScenarioChoicePlan{Schema: ScenarioChoicePlanSchema, Overrides: cloneScenarioChoiceOverrides(overrides)}
	for index := range plan.Overrides {
		if plan.Overrides[index].Identity != "" {
			return ScenarioChoicePlan{}, errors.New("scenario choice override contains a record-owned identity")
		}
		identity, err := scenarioChoiceOverrideIdentity(plan.Overrides[index])
		if err != nil {
			return ScenarioChoicePlan{}, err
		}
		plan.Overrides[index].Identity = identity
	}
	if err := validateScenarioChoiceOverrides(plan.Overrides); err != nil {
		return ScenarioChoicePlan{}, err
	}
	identity, err := scenarioChoicePlanIdentity(plan)
	if err != nil {
		return ScenarioChoicePlan{}, err
	}
	plan.Identity = identity
	if _, err := EncodeScenarioChoicePlan(plan); err != nil {
		return ScenarioChoicePlan{}, err
	}
	return plan, nil
}

func EncodeScenarioChoicePlan(plan ScenarioChoicePlan) ([]byte, error) {
	if err := validateScenarioChoicePlan(plan); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(plan)
	if err != nil {
		return nil, fmt.Errorf("encode scenario choice plan: %w", err)
	}
	if len(encoded) > MaximumScenarioChoicePlanBytes {
		return nil, &CapacityError{Resource: "scenario_choice_plan_bytes", Required: uint64(len(encoded)), Maximum: MaximumScenarioChoicePlanBytes}
	}
	return encoded, nil
}

func DecodeScenarioChoicePlan(data []byte) (ScenarioChoicePlan, error) {
	if len(data) == 0 || len(data) > MaximumScenarioChoicePlanBytes {
		return ScenarioChoicePlan{}, fmt.Errorf("scenario choice plan must be between 1 and %d bytes", MaximumScenarioChoicePlanBytes)
	}
	if !utf8.Valid(data) {
		return ScenarioChoicePlan{}, errors.New("scenario choice plan is not valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var plan ScenarioChoicePlan
	if err := decoder.Decode(&plan); err != nil {
		return ScenarioChoicePlan{}, fmt.Errorf("decode scenario choice plan: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ScenarioChoicePlan{}, errors.New("scenario choice plan contains trailing JSON")
	}
	if err := validateScenarioChoicePlan(plan); err != nil {
		return ScenarioChoicePlan{}, err
	}
	canonical, err := json.Marshal(plan)
	if err != nil {
		return ScenarioChoicePlan{}, fmt.Errorf("canonicalize scenario choice plan: %w", err)
	}
	if !bytes.Equal(data, canonical) {
		return ScenarioChoicePlan{}, errors.New("scenario choice plan is not canonical JSON")
	}
	return plan, nil
}

func validateScenarioChoicePlan(plan ScenarioChoicePlan) error {
	if plan.Schema != ScenarioChoicePlanSchema {
		return fmt.Errorf("scenario choice plan schema = %q, want %q", plan.Schema, ScenarioChoicePlanSchema)
	}
	if err := checkCapacity("scenario_choice_overrides", uint64(len(plan.Overrides)), MaximumScenarioDecisions); err != nil {
		return err
	}
	if err := validateScenarioChoiceOverrides(plan.Overrides); err != nil {
		return err
	}
	if !validSHA256(plan.Identity) {
		return errors.New("scenario choice plan identity is invalid")
	}
	want, err := scenarioChoicePlanIdentity(plan)
	if err != nil {
		return err
	}
	if plan.Identity != want {
		return errors.New("scenario choice plan identity does not match its contents")
	}
	return nil
}

func validateScenarioChoiceOverrides(overrides []ScenarioChoiceOverride) error {
	for index, override := range overrides {
		if index != 0 && overrides[index-1].Ordinal >= override.Ordinal {
			return errors.New("scenario choice override ordinals must be strictly increasing")
		}
		if err := validateID("scenario choice override ID", override.ID); err != nil {
			return err
		}
		if override.Occurrence == 0 {
			return errors.New("scenario choice override occurrence is zero")
		}
		if len(override.Alternatives) < 2 {
			return errors.New("scenario choice override must contain at least two alternatives")
		}
		seen := make(map[string]struct{}, len(override.Alternatives))
		for _, alternative := range override.Alternatives {
			if err := validateID("scenario choice override alternative", alternative); err != nil {
				return err
			}
			if _, ok := seen[alternative]; ok {
				return errors.New("scenario choice override alternatives are duplicated")
			}
			seen[alternative] = struct{}{}
		}
		if override.Selected >= uint64(len(override.Alternatives)) {
			return errors.New("scenario choice override selected rank is invalid")
		}
		if !validSHA256(override.Identity) {
			return errors.New("scenario choice override identity is invalid")
		}
		want, err := scenarioChoiceOverrideIdentity(override)
		if err != nil {
			return err
		}
		if override.Identity != want {
			return errors.New("scenario choice override identity does not match its contents")
		}
	}
	return nil
}

func scenarioChoiceOverrideIdentity(override ScenarioChoiceOverride) (string, error) {
	override.Identity = ""
	override.Alternatives = append([]string(nil), override.Alternatives...)
	return hashCanonical("gomad3-scenario-choice-override/v1", override)
}

func scenarioChoicePlanIdentity(plan ScenarioChoicePlan) (string, error) {
	plan.Identity = ""
	plan.Overrides = cloneScenarioChoiceOverrides(plan.Overrides)
	return hashCanonical("gomad3-scenario-choice-plan/v1", plan)
}

func cloneScenarioChoicePlan(plan ScenarioChoicePlan) ScenarioChoicePlan {
	plan.Overrides = cloneScenarioChoiceOverrides(plan.Overrides)
	return plan
}

func cloneScenarioChoiceOverrides(overrides []ScenarioChoiceOverride) []ScenarioChoiceOverride {
	cloned := make([]ScenarioChoiceOverride, len(overrides))
	for index, override := range overrides {
		cloned[index] = override
		cloned[index].Alternatives = append([]string(nil), override.Alternatives...)
	}
	return cloned
}

func equalScenarioChoicePlan(left, right ScenarioChoicePlan) bool {
	return left.Schema == right.Schema && left.Identity == right.Identity && slices.EqualFunc(left.Overrides, right.Overrides, func(left, right ScenarioChoiceOverride) bool {
		return left.Ordinal == right.Ordinal && left.ID == right.ID && left.Occurrence == right.Occurrence && slices.Equal(left.Alternatives, right.Alternatives) && left.Selected == right.Selected && left.Identity == right.Identity
	})
}

func scenarioChoicePlanForSpec(spec Spec) (ScenarioChoicePlan, error) {
	if spec.ScenarioChoices == nil {
		return NewScenarioChoicePlan(nil)
	}
	plan := cloneScenarioChoicePlan(*spec.ScenarioChoices)
	if err := validateScenarioChoicePlan(plan); err != nil {
		return ScenarioChoicePlan{}, err
	}
	return plan, nil
}

func scenarioDecisionFromChoiceOverride(override ScenarioChoiceOverride) ScenarioDecision {
	decision := ScenarioDecision{
		Ordinal: override.Ordinal, ID: override.ID, Kind: ScenarioDecisionChoose, Occurrence: override.Occurrence,
		Alternatives: append([]string(nil), override.Alternatives...), Selected: override.Selected,
	}
	decision.Identity, _ = scenarioDecisionIdentity(decision)
	return decision
}

func scenarioChoiceOverrideMatchesDecision(override ScenarioChoiceOverride, decision ScenarioDecision) bool {
	return decision.Kind == ScenarioDecisionChoose && override.Ordinal == decision.Ordinal && override.ID == decision.ID &&
		override.Occurrence == decision.Occurrence && slices.Equal(override.Alternatives, decision.Alternatives)
}
