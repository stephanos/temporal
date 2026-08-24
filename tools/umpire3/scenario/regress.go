package scenario

import (
	"runtime"
	"time"

	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

func NewScenario(identifier string, target protocolcatalog.TargetID, resources []Resource, root Term) Scenario {
	return Scenario{Identifier: identifier, Target: target, Resources: resources, Root: root}
}

func Identity(name string) Symbol {
	return Symbol{Name: name, Type: protocolcatalog.SemanticTypeIDIdentity}
}

func Synchronously() ActionOption {
	return WithResponse(protocolexperiment.ResponseSynchronous)
}

func Asynchronously() ActionOption {
	return WithResponse(protocolexperiment.ResponseAsynchronous)
}

func Deferred() ActionOption {
	return WithResponse(protocolexperiment.ResponseDeferred)
}

func BlockingFor(duration time.Duration) ActionOption {
	return WithBoundedBlock(duration)
}

func FailingResponse() ActionOption {
	return WithResponse(protocolexperiment.ResponseFailure)
}

func Action(identifier string, kind protocolcatalog.ActionKind, options ...ActionOption) Term {
	return actionAtCaller(identifier, kind, options...)
}

func Bind(symbol Symbol, projection Projection) Term {
	return BindAt(callerSource(2), symbol, projection)
}

func BindIdentity(symbol Symbol, producerAction, projection string) Term {
	return BindAt(callerSource(2), symbol,
		Project(producerAction, projection, protocolcatalog.SemanticTypeIDIdentity))
}

func Require(property protocolcatalog.PropertyID) Term {
	return requireAtCaller(property)
}

func OnePath(children ...Term) Term {
	return OnePathAt(callerSource(2), children...)
}

func AllPaths(children ...Term) Term {
	return AllPathsAt(callerSource(2), children...)
}

func AnyOrder(children ...Term) Term {
	return AnyOrderAt(callerSource(2), children...)
}

func Before(before, after Term) Term {
	return BeforeAt(callerSource(2), before, after)
}

func During(fault FaultIntent, body Term) Term {
	return DuringAt(callerSource(2), fault, body)
}

func Repeat(count int, body Term) Term {
	return RepeatAt(callerSource(2), count, body)
}

func actionAtCaller(identifier string, kind protocolcatalog.ActionKind, options ...ActionOption) Term {
	return ActionAt(callerSource(3), identifier, kind, options...)
}

func faultAtCaller(identifier string, kind protocolcatalog.FaultKind, options ...FaultOption) FaultIntent {
	return FaultAt(callerSource(3), identifier, kind, options...)
}

func requireAtCaller(property protocolcatalog.PropertyID) Term {
	return RequireAt(callerSource(3), property)
}

func callerSource(skip int) Source {
	programCounter, file, line, ok := runtime.Caller(skip)
	if !ok {
		return Source{}
	}
	function := ""
	if details := runtime.FuncForPC(programCounter); details != nil {
		function = details.Name()
	}
	return Source{File: file, Line: line, Function: function}
}
