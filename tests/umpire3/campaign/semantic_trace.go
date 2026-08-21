package campaign

import (
	"errors"

	umpire3runtime "go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func bindAcceptedSemanticTrace(
	experiment protocol.Experiment,
	result *umpire3runtime.Result,
) error {
	view, found, err := protocol.DefaultAttemptExecutionView(experiment)
	if err != nil {
		return err
	}
	if !found {
		return errors.New("experiment has no attempt execution view")
	}
	attempts, found := findAcceptedObservedAttempts(view, experiment.Actions, nil)
	if !found {
		return errors.New("experiment has no accepted observed attempt path")
	}
	trace, err := protocol.NewLiveSemanticTrace(experiment, view, attempts)
	if err != nil {
		return err
	}
	result.Trace = &trace
	return nil
}

func findAcceptedObservedAttempts(
	view protocol.AttemptExecutionView,
	actions []protocol.Action,
	prefix []protocol.ObservedAttempt,
) ([]protocol.ObservedAttempt, bool) {
	if len(prefix) == len(actions) {
		return append([]protocol.ObservedAttempt{}, prefix...), true
	}
	action := actions[len(prefix)]
	for _, outcome := range action.AllowedOutcomes {
		candidate := append(prefix, protocol.ObservedAttempt{
			Action: protocol.ActionKind(action.Kind), Outcome: outcome,
		})
		replay, err := view.ReplayObserved(candidate)
		if err != nil || !replay.Accepted {
			continue
		}
		if accepted, ok := findAcceptedObservedAttempts(view, actions, candidate); ok {
			return accepted, true
		}
	}
	return nil, false
}
