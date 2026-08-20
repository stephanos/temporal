package callback

import (
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/regress"
)

type Handle struct {
	identifier string
}

func Callback(identifier string) Handle {
	return Handle{identifier: identifier}
}

func (c Handle) Resource() regress.Resource {
	return regress.Callback(c.identifier)
}

func (c Handle) Register(options ...regress.ActionOption) regress.Term {
	return regress.RegisterCallback(c.identifier+"-register", options...)
}

func (c Handle) Respond(options ...regress.ActionOption) regress.Term {
	return regress.RecordCallbackResponse(c.identifier+"-respond", options...)
}

func ReferenceRegression(identifier string, callback Handle, root regress.Term) regress.Scenario {
	return regress.NewScenario(identifier, protocol.TargetIDIntegrationCallbackNexus,
		[]regress.Resource{callback.Resource()}, root)
}

func ResponseRegression(identifier string, callback Handle, root regress.Term) regress.Scenario {
	return regress.NewScenario(identifier, protocol.TargetIDIntegrationCallbackWorkflow,
		[]regress.Resource{callback.Resource()}, root)
}
