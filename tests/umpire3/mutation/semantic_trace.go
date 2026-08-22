package mutation

import (
	"errors"

	"go.temporal.io/server/tests/umpire3/checker/finite"
	checkertrace "go.temporal.io/server/tests/umpire3/checker/trace"
	umpire3execution "go.temporal.io/server/tests/umpire3/execution"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

func bindAcceptedSemanticTrace(
	experiment protocolexperiment.Experiment,
	result *umpire3execution.Result,
) error {
	view, found, err := finite.DefaultAttemptExecutionView(experiment)
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
	trace, err := checkertrace.NewLive(experiment, view, attempts)
	if err != nil {
		return err
	}
	result.Trace = &trace
	return nil
}

func findAcceptedObservedAttempts(
	view finite.AttemptExecutionView,
	actions []protocolexperiment.Action,
	prefix []finite.ObservedAttempt,
) ([]finite.ObservedAttempt, bool) {
	if len(prefix) == len(actions) {
		return append([]finite.ObservedAttempt{}, prefix...), true
	}
	action := actions[len(prefix)]
	for _, outcome := range action.AllowedOutcomes {
		candidate := append(prefix, finite.ObservedAttempt{
			Action: protocolcatalog.ActionKind(action.Kind), Outcome: outcome,
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
