//nolint:revive // The package name is the public Umpire3 runtime.Run seam.
package runtime

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"go.temporal.io/server/tests/umpire3/protocol"
)

type ExecuteCandidate func(context.Context, protocol.Experiment) (Result, error)

func MinimizeActions(ctx context.Context, experiment protocol.Experiment, execute ExecuteCandidate) (protocol.Experiment, error) {
	if execute == nil {
		return protocol.Experiment{}, errors.New("candidate executor is required")
	}
	if err := experiment.Validate(); err != nil {
		return protocol.Experiment{}, fmt.Errorf("validate original experiment: %w", err)
	}
	result, err := execute(ctx, experiment)
	if err != nil {
		return protocol.Experiment{}, fmt.Errorf("execute original experiment: %w", err)
	}
	if !isRequestedViolation(experiment, result) {
		return protocol.Experiment{}, errors.New("original experiment does not produce the requested property violation")
	}
	violationCheckpoint := result.Claim.Checkpoint

	minimized := experiment
	for index := 0; index < len(minimized.Actions) && len(minimized.Actions) > 1; {
		candidate := minimized
		candidate.Actions = append([]protocol.Action(nil), minimized.Actions[:index]...)
		candidate.Actions = append(candidate.Actions, minimized.Actions[index+1:]...)
		if candidate.Scope.Bounds.MaxDepth > len(candidate.Actions) {
			candidate.Scope.Bounds.MaxDepth = len(candidate.Actions)
		}
		if err := candidate.Validate(); err != nil {
			index++
			continue
		}
		result, err := execute(ctx, candidate)
		if err != nil {
			return protocol.Experiment{}, fmt.Errorf("execute candidate without action %q: %w", minimized.Actions[index].Identifier, err)
		}
		if isRequestedViolation(experiment, result) && result.Claim.Checkpoint == violationCheckpoint {
			minimized = candidate
			index = 0
			continue
		}
		index++
	}
	return minimized, nil
}

func MinimizeExperiment(ctx context.Context, experiment protocol.Experiment, execute ExecuteCandidate) (protocol.Experiment, error) {
	minimized, err := MinimizeActions(ctx, experiment, execute)
	if err != nil {
		return protocol.Experiment{}, err
	}
	baseline, err := execute(ctx, minimized)
	if err != nil {
		return protocol.Experiment{}, err
	}
	checkpoint := baseline.Claim.Checkpoint
	preserves := func(candidate protocol.Experiment) (bool, error) {
		result, err := execute(ctx, candidate)
		if err != nil {
			return false, err
		}
		return isRequestedViolation(experiment, result) && result.Claim.Checkpoint == checkpoint, nil
	}

	for index := 0; index < len(minimized.Resources) && len(minimized.Resources) > 1; {
		candidate := minimized
		candidate.Resources = append([]protocol.Resource(nil), minimized.Resources[:index]...)
		candidate.Resources = append(candidate.Resources, minimized.Resources[index+1:]...)
		preserved, err := preserves(candidate)
		if err != nil {
			return protocol.Experiment{}, err
		}
		if preserved {
			minimized = candidate
			index = 0
			continue
		}
		index++
	}
	for actionIndex := range minimized.Actions {
		for mapKind := 0; mapKind < 2; mapKind++ {
			field := minimized.Actions[actionIndex].Arguments
			if mapKind == 1 {
				field = minimized.Actions[actionIndex].Bindings
			}
			keys := make([]string, 0, len(field))
			for key := range field {
				keys = append(keys, key)
			}
			slices.Sort(keys)
			for _, key := range keys {
				currentField := minimized.Actions[actionIndex].Arguments
				if mapKind == 1 {
					currentField = minimized.Actions[actionIndex].Bindings
				}
				if _, exists := currentField[key]; !exists {
					continue
				}
				candidate := minimized
				candidate.Actions = append([]protocol.Action(nil), minimized.Actions...)
				candidateMap := make(map[string]string, len(currentField)-1)
				for candidateKey, value := range currentField {
					if candidateKey != key {
						candidateMap[candidateKey] = value
					}
				}
				if mapKind == 0 {
					candidate.Actions[actionIndex].Arguments = candidateMap
				} else {
					candidate.Actions[actionIndex].Bindings = candidateMap
				}
				preserved, err := preserves(candidate)
				if err != nil {
					return protocol.Experiment{}, err
				}
				if preserved {
					minimized = candidate
				}
			}
		}
	}
	return minimized, nil
}

func isRequestedViolation(experiment protocol.Experiment, result Result) bool {
	return result.Claim.Kind == ClaimViolating && result.Claim.Property == experiment.Property.Identifier
}
