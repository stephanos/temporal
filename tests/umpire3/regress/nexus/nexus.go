package nexus

import (
	"runtime"

	"go.temporal.io/server/tests/umpire3/compiler"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/regress"
)

type OperationHandle struct {
	identifier string
	identity   regress.Symbol
}

func Operation(identifier string) OperationHandle {
	return OperationHandle{identifier: identifier, identity: regress.Identity(identifier + "-id")}
}

func (o OperationHandle) Resource() regress.Resource {
	return regress.NexusOperation(o.identifier)
}

func (o OperationHandle) Schedule(options ...regress.ActionOption) regress.Term {
	return regress.ActionAt(callerSource(), o.action("schedule"), protocol.ActionKindScheduleOperation, options...)
}

func (o OperationHandle) Dispatch(options ...regress.ActionOption) regress.Term {
	source := callerSource()
	action := o.action("dispatch")
	return compiler.OnePathAt(source,
		regress.ActionAt(source, action, protocol.ActionKindDispatchTask, options...),
		regress.BindAt(source, o.identity,
			regress.Project(action, "operation-id", protocol.SemanticTypeIDIdentity)),
	)
}

func (o OperationHandle) RequestCancellation(options ...regress.ActionOption) regress.Term {
	return regress.ActionAt(callerSource(), o.action("request-cancellation"), protocol.ActionKindRequestCancellation, options...)
}

func (o OperationHandle) CommitCancellation(options ...regress.ActionOption) regress.Term {
	return o.identityAction("commit-cancellation", protocol.ActionKindCommitCancellation, options...)
}

func (o OperationHandle) AcquireOwnership(options ...regress.ActionOption) regress.Term {
	return o.identityAction("acquire-ownership", protocol.ActionKindAcquireOwnership, options...)
}

func (o OperationHandle) Retry(options ...regress.ActionOption) regress.Term {
	return o.identityAction("retry", protocol.ActionKindRetryTask, options...)
}

func (o OperationHandle) WorkerReturnsSuccess(options ...regress.ActionOption) regress.Term {
	return o.identityAction("worker-success", protocol.ActionKindWorkerReturnsSuccess, options...)
}

func (o OperationHandle) PersistSuccess(options ...regress.ActionOption) regress.Term {
	return o.identityAction("persist-success", protocol.ActionKindPersistSuccess, options...)
}

func (o OperationHandle) CancelWithRetry() regress.Term {
	return compiler.OnePathAt(callerSource(),
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

func (o OperationHandle) CancellationSafety() regress.Term {
	return regress.RequireAt(callerSource(), protocol.PropertyIDNexusCancellationWonExcludesSuccess)
}

func Regression(identifier string, operation OperationHandle, root regress.Term) regress.Scenario {
	return regress.NewScenario(identifier, protocol.TargetIDNexusCancellation,
		[]regress.Resource{operation.Resource(), regress.NexusWorker(operation.identifier + "-worker")}, root)
}

func (o OperationHandle) identityAction(
	suffix string,
	kind protocol.ActionKind,
	options ...regress.ActionOption,
) regress.Term {
	options = append([]regress.ActionOption{regress.WithArgument("operation", o.identity.Value())}, options...)
	return regress.ActionAt(callerSource(), o.action(suffix), kind,
		options...)
}

func (o OperationHandle) action(suffix string) string {
	return o.identifier + "-" + suffix
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
