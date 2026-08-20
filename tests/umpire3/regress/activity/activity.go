package activity

import (
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/regress"
)

type Handle struct {
	identifier string
}

func Activity(identifier string) Handle {
	return Handle{identifier: identifier}
}

func (a Handle) Resource() regress.Resource {
	return regress.Activity(a.identifier)
}

func (a Handle) Progress(options ...regress.ActionOption) regress.Term {
	return regress.ProgressEntity(a.identifier+"-progress", options...)
}

func Regression(identifier string, activity Handle, root regress.Term) regress.Scenario {
	return regress.NewScenario(identifier, protocol.TargetIDIntegrationActivityDelivery,
		[]regress.Resource{activity.Resource()}, root)
}
