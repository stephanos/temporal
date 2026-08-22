package conformance

import (
	"encoding/json"
	"errors"

	sharedconformance "go.temporal.io/server/tools/common/formal/conformance"
	"go.temporal.io/server/tools/common/formal/model"
	"go.temporal.io/server/tools/gomad4/trace"
	"go.temporal.io/server/tools/gomad4/virtualtime"
)

type Limits = sharedconformance.Limits

type Report = sharedconformance.Report

type Divergence = sharedconformance.Divergence[
	virtualtime.StateSnapshot,
	virtualtime.ObservableDelta,
]

func Replay(corpus trace.Corpus, limits Limits) (Report, error) {
	return sharedconformance.Replay(corpus, adapter(), limits)
}

func adapter() sharedconformance.Adapter[
	virtualtime.StateSnapshot,
	virtualtime.Action,
	virtualtime.ObservableDelta,
	virtualtime.State,
	virtualtime.RejectionCode,
] {
	return sharedconformance.Adapter[
		virtualtime.StateSnapshot,
		virtualtime.Action,
		virtualtime.ObservableDelta,
		virtualtime.State,
		virtualtime.RejectionCode,
	]{
		Subject:  "virtual time",
		Restore:  virtualtime.Restore,
		Snapshot: virtualtime.State.Snapshot,
		Identity: virtualtime.State.Identity,
		Step: func(state virtualtime.State, action virtualtime.Action) (model.Transition[virtualtime.State, virtualtime.Action, virtualtime.ObservableDelta], error) {
			transition, err := virtualtime.Step(state, action)
			if err != nil {
				return model.Transition[virtualtime.State, virtualtime.Action, virtualtime.ObservableDelta]{}, err
			}
			return model.Transition[virtualtime.State, virtualtime.Action, virtualtime.ObservableDelta]{
				PostState: transition.PostState,
				Observation: model.Observation[virtualtime.Action, virtualtime.ObservableDelta]{
					Action: transition.Action, PreStateIdentity: transition.PreStateIdentity,
					PostStateIdentity: transition.PostStateIdentity, ObservableDelta: transition.Delta,
				},
			}, nil
		},
		RejectionCode: func(err error) (virtualtime.RejectionCode, bool) {
			var rejection *virtualtime.Rejection
			if !errors.As(err, &rejection) {
				return "", false
			}
			return rejection.Code, true
		},
		EqualObservableDelta: equalObservableDelta,
	}
}

func equalObservableDelta(left, right virtualtime.ObservableDelta) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftJSON) == string(rightJSON)
}
