package nexus

import (
	"runtime"

	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	"go.temporal.io/server/tests/umpire3/scenario"
)

type OperationHandle struct {
	identifier string
	identity   scenario.Symbol
}

func Operation(identifier string) OperationHandle {
	return OperationHandle{identifier: identifier, identity: scenario.Identity(identifier + "-id")}
}

func (o OperationHandle) Resource() scenario.Resource {
	return scenario.NexusOperation(o.identifier)
}

func (o OperationHandle) Schedule(options ...scenario.ActionOption) scenario.Term {
	return scenario.ActionAt(callerSource(), o.action("schedule"), protocolcatalog.ActionKindScheduleOperation, options...)
}

func (o OperationHandle) Dispatch(options ...scenario.ActionOption) scenario.Term {
	source := callerSource()
	action := o.action("dispatch")
	return scenario.OnePathAt(source,
		scenario.ActionAt(source, action, protocolcatalog.ActionKindDispatchTask, options...),
		scenario.BindAt(source, o.identity,
			scenario.Project(action, "operation-id", protocolcatalog.SemanticTypeIDIdentity)),
	)
}

func (o OperationHandle) RequestCancellation(options ...scenario.ActionOption) scenario.Term {
	return scenario.ActionAt(callerSource(), o.action("request-cancellation"), protocolcatalog.ActionKindRequestCancellation, options...)
}

func (o OperationHandle) CommitCancellation(options ...scenario.ActionOption) scenario.Term {
	return o.identityAction("commit-cancellation", protocolcatalog.ActionKindCommitCancellation, options...)
}

func (o OperationHandle) AcquireOwnership(options ...scenario.ActionOption) scenario.Term {
	return o.identityAction("acquire-ownership", protocolcatalog.ActionKindAcquireOwnership, options...)
}

func (o OperationHandle) Retry(options ...scenario.ActionOption) scenario.Term {
	return o.identityAction("retry", protocolcatalog.ActionKindRetryTask, options...)
}

func (o OperationHandle) WorkerReturnsSuccess(options ...scenario.ActionOption) scenario.Term {
	return o.identityAction("worker-success", protocolcatalog.ActionKindWorkerReturnsSuccess, options...)
}

func (o OperationHandle) PersistSuccess(options ...scenario.ActionOption) scenario.Term {
	return o.identityAction("persist-success", protocolcatalog.ActionKindPersistSuccess, options...)
}

func (o OperationHandle) CancelWithRetry() scenario.Term {
	return scenario.OnePathAt(callerSource(),
		o.Schedule(),
		o.Dispatch(),
		o.RequestCancellation(),
		o.CommitCancellation(),
		o.AcquireOwnership(),
		o.Retry(),
		o.WorkerReturnsSuccess(),
		o.PersistSuccess(),
	)
}

func (o OperationHandle) CancellationSafety() scenario.Term {
	return scenario.RequireAt(callerSource(), protocolcatalog.PropertyIDNexusCancellationWonExcludesSuccess)
}

func Scenario(identifier string, operation OperationHandle, root scenario.Term) scenario.Scenario {
	return scenario.NewScenario(identifier, protocolcatalog.TargetIDNexusCancellation,
		[]scenario.Resource{operation.Resource(), scenario.NexusWorker(operation.identifier + "-worker")}, root)
}

func (o OperationHandle) identityAction(
	suffix string,
	kind protocolcatalog.ActionKind,
	options ...scenario.ActionOption,
) scenario.Term {
	options = append([]scenario.ActionOption{scenario.WithArgument("operation", o.identity.Value())}, options...)
	return scenario.ActionAt(callerSource(), o.action(suffix), kind,
		options...)
}

func (o OperationHandle) action(suffix string) string {
	return o.identifier + "-" + suffix
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
