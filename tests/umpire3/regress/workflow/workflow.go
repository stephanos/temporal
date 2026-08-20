package workflow

import (
	"runtime"

	"go.temporal.io/server/tests/umpire3/compiler"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/regress"
)

type UpdateHandle struct {
	identifier string
	identity   regress.Symbol
}

func Update(identifier string) UpdateHandle {
	return UpdateHandle{identifier: identifier, identity: regress.Identity(identifier + "-id")}
}

func (u UpdateHandle) Resource() regress.Resource {
	return regress.WorkflowUpdate(u.identifier)
}

func (u UpdateHandle) Start(options ...regress.ActionOption) regress.Term {
	source := callerSource()
	action := u.action("start")
	return compiler.OnePathAt(source,
		regress.ActionAt(source, action, protocol.ActionKindStartUpdate, options...),
		regress.BindAt(source, u.identity,
			regress.Project(action, "update-id", protocol.SemanticTypeIDIdentity)),
	)
}

func (u UpdateHandle) Dispatch(options ...regress.ActionOption) regress.Term {
	return u.actionWithIdentity("dispatch", protocol.ActionKindDispatchWorkflowTask, options...)
}

func (u UpdateHandle) Accept(options ...regress.ActionOption) regress.Term {
	return u.actionWithIdentity("accept", protocol.ActionKindAcceptUpdate, options...)
}

func (u UpdateHandle) RecordHistory(options ...regress.ActionOption) regress.Term {
	return u.actionWithIdentity("history", protocol.ActionKindRecordUpdateHistory, options...)
}

func (u UpdateHandle) CompleteTask(options ...regress.ActionOption) regress.Term {
	return u.actionWithIdentity("complete-task", protocol.ActionKindCompleteWorkflowTask, options...)
}

func (u UpdateHandle) Complete(options ...regress.ActionOption) regress.Term {
	return u.actionWithIdentity("complete", protocol.ActionKindCompleteUpdate, options...)
}

func (u UpdateHandle) Lifecycle() regress.Term {
	return compiler.OnePathAt(callerSource(),
		u.Start(), u.Dispatch(), u.Accept(), u.RecordHistory(), u.CompleteTask(), u.Complete())
}

func (u UpdateHandle) CompletionThroughHistory() regress.Term {
	return regress.RequireAt(callerSource(), protocol.PropertyIDWorkflowUpdateAcceptedCompletesThroughHistory)
}

func Regression(identifier string, update UpdateHandle, root regress.Term) regress.Scenario {
	return regress.NewScenario(identifier, protocol.TargetIDWorkflowUpdateLifecycle,
		[]regress.Resource{regress.Workflow(identifier + "-workflow"), update.Resource()}, root)
}

func (u UpdateHandle) actionWithIdentity(
	suffix string,
	kind protocol.ActionKind,
	options ...regress.ActionOption,
) regress.Term {
	options = append([]regress.ActionOption{regress.WithArgument("update", u.identity.Value())}, options...)
	return regress.ActionAt(callerSource(), u.action(suffix), kind,
		options...)
}

func (u UpdateHandle) action(suffix string) string {
	return u.identifier + "-" + suffix
}

func callerSource() compiler.Source {
	programCounter, file, line, ok := runtime.Caller(2)
	if !ok {
		return compiler.Source{}
	}
	function := ""
	if details := runtime.FuncForPC(programCounter); details != nil {
		function = details.Name()
	}
	return compiler.Source{File: file, Line: line, Function: function}
}
