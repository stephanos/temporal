package workflow

import (
	"runtime"

	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/scenario"
)

type UpdateHandle struct {
	identifier string
	identity   scenario.Symbol
}

func Update(identifier string) UpdateHandle {
	return UpdateHandle{identifier: identifier, identity: scenario.Identity(identifier + "-id")}
}

func (u UpdateHandle) Resource() scenario.Resource {
	return scenario.WorkflowUpdate(u.identifier)
}

func (u UpdateHandle) Start(options ...scenario.ActionOption) scenario.Term {
	source := callerSource()
	action := u.action("start")
	return scenario.OnePathAt(source,
		scenario.ActionAt(source, action, protocol.ActionKindStartUpdate, options...),
		scenario.BindAt(source, u.identity,
			scenario.Project(action, "update-id", protocol.SemanticTypeIDIdentity)),
	)
}

func (u UpdateHandle) Dispatch(options ...scenario.ActionOption) scenario.Term {
	return u.actionWithIdentity("dispatch", protocol.ActionKindDispatchWorkflowTask, options...)
}

func (u UpdateHandle) Accept(options ...scenario.ActionOption) scenario.Term {
	return u.actionWithIdentity("accept", protocol.ActionKindAcceptUpdate, options...)
}

func (u UpdateHandle) RecordHistory(options ...scenario.ActionOption) scenario.Term {
	return u.actionWithIdentity("history", protocol.ActionKindRecordUpdateHistory, options...)
}

func (u UpdateHandle) CompleteTask(options ...scenario.ActionOption) scenario.Term {
	return u.actionWithIdentity("complete-task", protocol.ActionKindCompleteWorkflowTask, options...)
}

func (u UpdateHandle) Complete(options ...scenario.ActionOption) scenario.Term {
	return u.actionWithIdentity("complete", protocol.ActionKindCompleteUpdate, options...)
}

func (u UpdateHandle) Lifecycle() scenario.Term {
	return scenario.OnePathAt(callerSource(),
		u.Start(), u.Dispatch(), u.Accept(), u.RecordHistory(), u.CompleteTask(), u.Complete())
}

func (u UpdateHandle) CompletionThroughHistory() scenario.Term {
	return scenario.RequireAt(callerSource(), protocol.PropertyIDWorkflowUpdateAcceptedCompletesThroughHistory)
}

func Regression(identifier string, update UpdateHandle, root scenario.Term) scenario.Scenario {
	return scenario.NewScenario(identifier, protocol.TargetIDWorkflowUpdateLifecycle,
		[]scenario.Resource{scenario.Workflow(identifier + "-workflow"), update.Resource()}, root)
}

func (u UpdateHandle) actionWithIdentity(
	suffix string,
	kind protocol.ActionKind,
	options ...scenario.ActionOption,
) scenario.Term {
	options = append([]scenario.ActionOption{scenario.WithArgument("update", u.identity.Value())}, options...)
	return scenario.ActionAt(callerSource(), u.action(suffix), kind,
		options...)
}

func (u UpdateHandle) action(suffix string) string {
	return u.identifier + "-" + suffix
}

func callerSource() scenario.Source {
	programCounter, file, line, ok := runtime.Caller(2)
	if !ok {
		return scenario.Source{}
	}
	function := ""
	if details := runtime.FuncForPC(programCounter); details != nil {
		function = details.Name()
	}
	return scenario.Source{File: file, Line: line, Function: function}
}
