package gomadv3sim

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"sync"
)

type ScenarioDecisionKind string

const (
	ScenarioDecisionAction   ScenarioDecisionKind = "action"
	ScenarioDecisionChoose   ScenarioDecisionKind = "choose"
	ScenarioDecisionParallel ScenarioDecisionKind = "parallel"
)

type ScenarioDecision struct {
	Ordinal      uint64               `json:"ordinal"`
	ID           string               `json:"id"`
	Kind         ScenarioDecisionKind `json:"kind"`
	Occurrence   uint64               `json:"occurrence"`
	Alternatives []string             `json:"alternatives,omitempty"`
	Selected     uint64               `json:"selected,omitempty"`
	Identity     string               `json:"identity"`
}

type ScenarioStep struct {
	id     string
	run    Scenario
	record bool
}

type scenarioDecisionController interface {
	recordScenarioDecision(context.Context, ScenarioDecision) error
	chooseScenarioAlternative(context.Context, ScenarioDecision) (uint64, error)
}

func NewScenarioStep(id string, scenario Scenario) (ScenarioStep, error) {
	if err := validateID("scenario step ID", id); err != nil {
		return ScenarioStep{}, err
	}
	if scenario == nil {
		return ScenarioStep{}, errors.New("scenario step callback is nil")
	}
	return ScenarioStep{id: id, run: scenario, record: true}, nil
}

func Sequence(steps ...ScenarioStep) Scenario {
	steps = append([]ScenarioStep(nil), steps...)
	return func(ctx context.Context, cluster Cluster) error {
		if err := validateScenarioSteps(steps); err != nil {
			return err
		}
		for _, step := range steps {
			if err := executeScenarioStep(ctx, cluster, step); err != nil {
				return fmt.Errorf("scenario step %q: %w", step.id, err)
			}
		}
		return nil
	}
}

func Repeat(count uint64, step ScenarioStep) ScenarioStep {
	return ScenarioStep{
		id: "repeat-" + step.id,
		run: func(ctx context.Context, cluster Cluster) error {
			if count > MaximumScenarioActions {
				return &CapacityError{Resource: "scenario_actions", Required: count, Maximum: MaximumScenarioActions}
			}
			if err := validateScenarioStep(step); err != nil {
				return err
			}
			for index := uint64(0); index < count; index++ {
				if err := executeScenarioStep(ctx, cluster, step); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func Choose(id string, steps ...ScenarioStep) Scenario {
	steps = append([]ScenarioStep(nil), steps...)
	return func(ctx context.Context, cluster Cluster) error {
		if err := validateID("scenario choice ID", id); err != nil {
			return err
		}
		if len(steps) == 0 {
			return errors.New("scenario choice has no alternatives")
		}
		if err := validateScenarioSteps(steps); err != nil {
			return err
		}
		controller, err := scenarioController(cluster)
		if err != nil {
			return err
		}
		alternatives := scenarioStepIDs(steps)
		selected, err := controller.chooseScenarioAlternative(ctx, ScenarioDecision{ID: id, Kind: ScenarioDecisionChoose, Alternatives: alternatives})
		if err != nil {
			return err
		}
		if selected >= uint64(len(steps)) {
			return errors.New("scenario controller selected an invalid alternative")
		}
		step := steps[selected]
		if err := step.run(ctx, cluster); err != nil {
			return fmt.Errorf("scenario step %q: %w", step.id, err)
		}
		return nil
	}
}

func BoundedParallel(id string, maximum uint64, steps ...ScenarioStep) Scenario {
	steps = append([]ScenarioStep(nil), steps...)
	return func(ctx context.Context, cluster Cluster) error {
		if err := validateID("parallel scenario ID", id); err != nil {
			return err
		}
		if maximum == 0 {
			return errors.New("parallel scenario limit is zero")
		}
		if len(steps) == 0 {
			return errors.New("parallel scenario has no actions")
		}
		if uint64(len(steps)) > MaximumScenarioActions {
			return &CapacityError{Resource: "scenario_actions", Required: uint64(len(steps)), Maximum: MaximumScenarioActions}
		}
		if err := validateScenarioSteps(steps); err != nil {
			return err
		}
		controller, err := scenarioController(cluster)
		if err != nil {
			return err
		}
		if err := controller.recordScenarioDecision(ctx, ScenarioDecision{ID: id, Kind: ScenarioDecisionParallel, Alternatives: scenarioStepIDs(steps)}); err != nil {
			return err
		}
		for start := uint64(0); start < uint64(len(steps)); start += maximum {
			end := min(start+maximum, uint64(len(steps)))
			errorsByIndex := make([]error, end-start)
			var wait sync.WaitGroup
			for index := start; index < end; index++ {
				index := index
				wait.Add(1)
				go func() {
					defer wait.Done()
					errorsByIndex[index-start] = steps[index].run(ctx, cluster)
				}()
			}
			wait.Wait()
			for offset, runErr := range errorsByIndex {
				if runErr != nil {
					return fmt.Errorf("scenario step %q: %w", steps[start+uint64(offset)].id, runErr)
				}
			}
		}
		return nil
	}
}

func executeScenarioStep(ctx context.Context, cluster Cluster, step ScenarioStep) error {
	if err := validateScenarioStep(step); err != nil {
		return err
	}
	if step.record {
		controller, err := scenarioController(cluster)
		if err != nil {
			return err
		}
		if err := controller.recordScenarioDecision(ctx, ScenarioDecision{ID: step.id, Kind: ScenarioDecisionAction}); err != nil {
			return err
		}
	}
	return step.run(ctx, cluster)
}

func scenarioController(cluster Cluster) (scenarioDecisionController, error) {
	if cluster == nil {
		return nil, errors.New("simulation cluster is nil")
	}
	controller, ok := cluster.(scenarioDecisionController)
	if !ok {
		return nil, errors.New("simulation cluster does not support typed scenario composition")
	}
	return controller, nil
}

func validateScenarioSteps(steps []ScenarioStep) error {
	if uint64(len(steps)) > MaximumScenarioActions {
		return &CapacityError{Resource: "scenario_actions", Required: uint64(len(steps)), Maximum: MaximumScenarioActions}
	}
	seen := make(map[string]struct{}, len(steps))
	for _, step := range steps {
		if err := validateScenarioStep(step); err != nil {
			return err
		}
		if _, ok := seen[step.id]; ok {
			return fmt.Errorf("scenario step ID %q is duplicated", step.id)
		}
		seen[step.id] = struct{}{}
	}
	return nil
}

func validateScenarioStep(step ScenarioStep) error {
	if err := validateID("scenario step ID", step.id); err != nil {
		return err
	}
	if step.run == nil {
		return errors.New("scenario step callback is nil")
	}
	return nil
}

func scenarioStepIDs(steps []ScenarioStep) []string {
	identities := make([]string, len(steps))
	for index, step := range steps {
		identities[index] = step.id
	}
	return identities
}

func validateScenarioDecision(decision ScenarioDecision) error {
	if err := validateID("scenario decision ID", decision.ID); err != nil {
		return err
	}
	if decision.Occurrence == 0 {
		return errors.New("scenario decision occurrence is zero")
	}
	seen := make(map[string]struct{}, len(decision.Alternatives))
	for _, alternative := range decision.Alternatives {
		if err := validateID("scenario alternative ID", alternative); err != nil {
			return err
		}
		if _, ok := seen[alternative]; ok {
			return errors.New("scenario alternatives are duplicated")
		}
		seen[alternative] = struct{}{}
	}
	switch decision.Kind {
	case ScenarioDecisionAction:
		if len(decision.Alternatives) != 0 || decision.Selected != 0 {
			return errors.New("scenario action decision contains alternatives")
		}
	case ScenarioDecisionChoose:
		if len(decision.Alternatives) == 0 || decision.Selected >= uint64(len(decision.Alternatives)) {
			return errors.New("scenario choice decision is invalid")
		}
	case ScenarioDecisionParallel:
		if len(decision.Alternatives) == 0 || decision.Selected != 0 {
			return errors.New("parallel scenario decision is invalid")
		}
	default:
		return fmt.Errorf("scenario decision kind %q is invalid", decision.Kind)
	}
	if !validSHA256(decision.Identity) {
		return errors.New("scenario decision identity is invalid")
	}
	want, err := scenarioDecisionIdentity(decision)
	if err != nil {
		return err
	}
	if decision.Identity != want {
		return errors.New("scenario decision identity does not match its contents")
	}
	return nil
}

func scenarioDecisionIdentity(decision ScenarioDecision) (string, error) {
	decision.Identity = ""
	decision.Alternatives = append([]string(nil), decision.Alternatives...)
	return hashCanonical("gomadv3-scenario-decision/v1", decision)
}

func selectScenarioAlternative(seed, ordinal uint64, id string, alternatives uint64) uint64 {
	if alternatives == 0 {
		return 0
	}
	input := make([]byte, 0, len(id)+40)
	input = append(input, "gomadv3-scenario-choice/v1"...)
	input = append(input, 0)
	var encoded [8]byte
	binary.LittleEndian.PutUint64(encoded[:], seed)
	input = append(input, encoded[:]...)
	binary.LittleEndian.PutUint64(encoded[:], ordinal)
	input = append(input, encoded[:]...)
	input = append(input, id...)
	digest := sha256.Sum256(input)
	return binary.LittleEndian.Uint64(digest[:8]) % alternatives
}

func cloneScenarioDecisions(decisions []ScenarioDecision) []ScenarioDecision {
	cloned := make([]ScenarioDecision, len(decisions))
	for index, decision := range decisions {
		cloned[index] = decision
		cloned[index].Alternatives = append([]string(nil), decision.Alternatives...)
	}
	return cloned
}

func equalScenarioDecisions(left, right []ScenarioDecision) bool {
	return slices.EqualFunc(left, right, func(left, right ScenarioDecision) bool {
		return left.Ordinal == right.Ordinal && left.ID == right.ID && left.Kind == right.Kind && left.Occurrence == right.Occurrence && slices.Equal(left.Alternatives, right.Alternatives) && left.Selected == right.Selected && left.Identity == right.Identity
	})
}
