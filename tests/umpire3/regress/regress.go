package regress

import (
	"runtime"
	"time"

	"go.temporal.io/server/tests/umpire3/compiler"
	"go.temporal.io/server/tests/umpire3/protocol"
)

type Term = compiler.Node
type Resource = compiler.Resource
type Scenario = compiler.Scenario
type Symbol = compiler.Symbol
type Projection = compiler.Projection
type FaultIntent = compiler.FaultIntent
type ActionOption = compiler.ActionOption

func NewScenario(identifier string, target protocol.TargetID, resources []Resource, root Term) Scenario {
	return Scenario{Identifier: identifier, Target: target, Resources: resources, Root: root}
}

func Identity(name string) Symbol {
	return Symbol{Name: name, Type: protocol.SemanticTypeIDIdentity}
}

func Project(producerAction, projection string, valueType protocol.SemanticTypeID) Projection {
	return compiler.Project(producerAction, projection, valueType)
}

func WithArgument(name string, value protocol.Value) ActionOption {
	return compiler.WithArgument(name, value)
}

func WithResponse(mode protocol.ResponseMode) ActionOption {
	return compiler.WithResponse(mode)
}

func WithBoundedBlock(duration time.Duration) ActionOption {
	return compiler.WithBoundedBlock(duration)
}

func Synchronously() ActionOption {
	return compiler.WithResponse(protocol.ResponseSynchronous)
}

func Asynchronously() ActionOption {
	return compiler.WithResponse(protocol.ResponseAsynchronous)
}

func Deferred() ActionOption {
	return compiler.WithResponse(protocol.ResponseDeferred)
}

func BlockingFor(duration time.Duration) ActionOption {
	return compiler.WithBoundedBlock(duration)
}

func FailingResponse() ActionOption {
	return compiler.WithResponse(protocol.ResponseFailure)
}

func Action(identifier string, kind protocol.ActionKind, options ...ActionOption) Term {
	return actionAtCaller(identifier, kind, options...)
}

func ActionAt(source compiler.Source, identifier string, kind protocol.ActionKind, options ...ActionOption) Term {
	return compiler.ActionAt(source, identifier, kind, options...)
}

func Bind(symbol Symbol, projection Projection) Term {
	return compiler.BindAt(callerSource(2), symbol, projection)
}

func BindAt(source compiler.Source, symbol Symbol, projection Projection) Term {
	return compiler.BindAt(source, symbol, projection)
}

func Require(property protocol.PropertyID) Term {
	return requireAtCaller(property)
}

func RequireAt(source compiler.Source, property protocol.PropertyID) Term {
	return compiler.RequireAt(source, property)
}

func FaultAt(source compiler.Source, identifier string, kind protocol.FaultKind, options ...ActionOption) FaultIntent {
	return compiler.FaultAt(source, identifier, kind, options...)
}

func ConfiguredFault(fault protocol.Fault) FaultIntent {
	return compiler.ConfiguredFault(fault)
}

func OnePath(children ...Term) Term {
	return compiler.OnePathAt(callerSource(2), children...)
}

func AllPaths(children ...Term) Term {
	return compiler.AllPathsAt(callerSource(2), children...)
}

func AnyOrder(children ...Term) Term {
	return compiler.AnyOrderAt(callerSource(2), children...)
}

func Before(before, after Term) Term {
	return compiler.BeforeAt(callerSource(2), before, after)
}

func During(fault FaultIntent, body Term) Term {
	return compiler.DuringAt(callerSource(2), fault, body)
}

func Repeat(count int, body Term) Term {
	return compiler.RepeatAt(callerSource(2), count, body)
}

func actionAtCaller(identifier string, kind protocol.ActionKind, options ...ActionOption) Term {
	return compiler.ActionAt(callerSource(3), identifier, kind, options...)
}

func faultAtCaller(identifier string, kind protocol.FaultKind, options ...ActionOption) FaultIntent {
	return compiler.FaultAt(callerSource(3), identifier, kind, options...)
}

func requireAtCaller(property protocol.PropertyID) Term {
	return compiler.RequireAt(callerSource(3), property)
}

func callerSource(skip int) compiler.Source {
	programCounter, file, line, ok := runtime.Caller(skip)
	if !ok {
		return compiler.Source{}
	}
	function := ""
	if details := runtime.FuncForPC(programCounter); details != nil {
		function = details.Name()
	}
	return compiler.Source{File: file, Line: line, Function: function}
}
