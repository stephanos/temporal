// How Temporal's APIs behave between calls, declared once for every Temporal realization
// (.plans/API_BEHAVIOR_HINTS.md): when a write's effect is visible to a read, and how long each
// kind of asynchronous cause may take. Each hint is a claim about the server and cites the code it
// rests on; a wrong one hides a bug or wastes time, so a change to one changes its citation too.
//
// Only the pairs and causes an existing Query's path needs are declared. The lowering derives each
// read's wait from them (tools/umpire/lower/waits.go): a realization writes no interval and no
// deadline of its own.
package temporal.realize

import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.*

// The API behavior `temporalRealization` attaches to every Temporal realization.
val temporalBehavior: ApiBehavior = ApiBehavior(
  visibility = Vector(
    // A start persists the new activity, scheduled, before it returns, and a describe reads that
    // component under the execution's lease: chasm/lib/activity/handler.go:79-112, :200-203,
    // service/history/chasm_engine.go:221-233, :675-701.
    METHOD_START_ACTIVITY_EXECUTION.visibleTo(METHOD_DESCRIBE_ACTIVITY_EXECUTION, Visible.atOnce),
    // A pause of a scheduled activity applies PAUSED in its own transaction (of a started one it is
    // PAUSE_REQUESTED, which the path avoids): chasm/lib/activity/operator_commands.go:288-322,
    // statemachine.go:221-243.
    METHOD_PAUSE_ACTIVITY_EXECUTION.visibleTo(METHOD_DESCRIBE_ACTIVITY_EXECUTION, Visible.atOnce),
    // An unpause applies SCHEDULED in its own transaction and dispatches through the timer queue,
    // which the delivery bound covers: chasm/lib/activity/operator_commands.go:324-361,
    // statemachine.go:247-258, :659-680.
    METHOD_UNPAUSE_ACTIVITY_EXECUTION.visibleTo(METHOD_DESCRIBE_ACTIVITY_EXECUTION, Visible.atOnce),
    // A terminate applies TERMINATED in its own transaction, without waiting for a worker:
    // chasm/lib/activity/handler.go:333-342, activity.go:551-569, statemachine.go:161-175.
    METHOD_TERMINATE_ACTIVITY_EXECUTION.visibleTo(
      METHOD_DESCRIBE_ACTIVITY_EXECUTION,
      Visible.atOnce
    ),
    // A cancel request applies CANCEL_REQUESTED, and CANCELED where no attempt runs, in its own
    // transaction: chasm/lib/activity/handler.go:352-375, operator_commands.go:236-286,
    // statemachine.go:176-190, :552-564.
    METHOD_REQUEST_CANCEL_ACTIVITY_EXECUTION.visibleTo(
      METHOD_DESCRIBE_ACTIVITY_EXECUTION,
      Visible.atOnce
    ),
    // The frontend synthesizes the standalone component token; history persists the component
    // update before replying, and Describe reads it under a lease:
    // service/frontend/workflow_handler.go:1817-1832, :2060-2074, :2249-2264;
    // service/history/handler.go:425-429, :476-480, :527-531;
    // service/history/chasm_engine.go:437-456; chasm/lib/activity/activity.go:463-470,
    // :513-522, :539-547; chasm/lib/activity/handler.go:194-203.
    METHOD_RESPOND_ACTIVITY_TASK_COMPLETED_BY_ID.visibleTo(
      METHOD_DESCRIBE_ACTIVITY_EXECUTION,
      Visible.atOnce
    ),
    METHOD_RESPOND_ACTIVITY_TASK_FAILED_BY_ID.visibleTo(
      METHOD_DESCRIBE_ACTIVITY_EXECUTION,
      Visible.atOnce
    ),
    METHOD_RESPOND_ACTIVITY_TASK_CANCELED_BY_ID.visibleTo(
      METHOD_DESCRIBE_ACTIVITY_EXECUTION,
      Visible.atOnce
    ),
    // A worker's answer applies the attempt's outcome, or a retry's reschedule and count, in the
    // respond's transaction: service/history/handler.go:425-429, :476-480, :527-531,
    // chasm/lib/activity/activity.go:463, :513, :539, attempt.go:204-240.
    CauseKind.activityAnswer.visibleTo(METHOD_DESCRIBE_ACTIVITY_EXECUTION, Visible.atOnce),
    // SDK RecordHeartbeat returns void and can batch calls: go.temporal.io/sdk v1.48.0,
    // activity/activity.go:77-79, internal/internal_activity.go:397-413,
    // internal/internal_task_handlers.go:2182-2214. Describe proves persistence, not invocation:
    // chasm/lib/activity/activity.go:592-619, :261-273.
    CauseKind.activityHeartbeat.visibleTo(
      METHOD_DESCRIBE_ACTIVITY_EXECUTION,
      Visible.eventually(WaitBound(intervalMs = 250, atMostMs = 2000))
    ),
    // A start of a standalone Nexus operation persists it, scheduled, before it returns, and a
    // describe reads that component: chasm/lib/nexusoperation/frontend.go:65-95, handler.go:45-60,
    // :152-155, operation.go:702-736.
    METHOD_START_NEXUS_OPERATION_EXECUTION.visibleTo(
      METHOD_DESCRIBE_NEXUS_OPERATION_EXECUTION,
      Visible.atOnce
    ),
    // A terminate applies TERMINATED in its own transaction:
    // chasm/lib/nexusoperation/handler.go:293-307, operation.go:671-690,
    // operation_statemachine.go:277-299.
    METHOD_TERMINATE_NEXUS_OPERATION_EXECUTION.visibleTo(
      METHOD_DESCRIBE_NEXUS_OPERATION_EXECUTION,
      Visible.atOnce
    ),
    // A start persists the workflow's first events before it returns:
    // service/history/api/startworkflow/api.go:326-332.
    METHOD_START_WORKFLOW_EXECUTION.visibleTo(
      METHOD_GET_WORKFLOW_EXECUTION_HISTORY,
      Visible.atOnce
    ),
    // A workflow task's scheduled Nexus operation is never buffered and is persisted before the
    // respond returns: service/history/historybuilder/event_store.go:297-338,
    // service/history/api/respondworkflowtaskcompleted/api.go:694-700.
    CauseKind.workflowTask.visibleTo(METHOD_GET_WORKFLOW_EXECUTION_HISTORY, Visible.atOnce),
    // Matching hands a handler's reply to the waiting dispatch and returns; history's invocation
    // task records the attempt only afterwards: service/matching/matching_engine.go:2884-2909,
    // service/history/hsm/nexusoperations/executors.go:339, :528-575, statemachine.go:276-288.
    // The retry then waits 0.8-1 s (config.go:122-131, common/backoff/retrypolicy.go:181-193), so
    // the interval stays well below that.
    CauseKind.handlerReply.visibleTo(
      METHOD_DESCRIBE_WORKFLOW_EXECUTION,
      Visible.eventually(WaitBound(intervalMs = 250, atMostMs = 2000))
    )
  ),
  causes = Vector(
    // Matching hands the task to a waiting poll; after an unpause, or a retry's backoff timer, the
    // timer queue dispatches it: chasm/lib/activity/tasks.go:67-102, statemachine.go:393-420.
    CauseKind.delivery.boundedBy(WaitBound(intervalMs = 250, atMostMs = 3000)),
    // The Case's own worker answers an attempt as soon as it is delivered, with nothing to wait for
    // (common/testing/testpilot/temporal/worker/interpreter.go:240-300), and the respond applies the
    // answer in its transaction (service/history/handler.go:425-429): the bound is the round trip.
    CauseKind.activityAnswer.boundedBy(WaitBound(intervalMs = 250, atMostMs = 2000)),
    CauseKind.activityHeartbeat.boundedBy(WaitBound(intervalMs = 250, atMostMs = 2000)),
    // A workflow task is dispatched to the Case's worker and completed by it:
    // service/matching/matching_engine.go:586, :717.
    CauseKind.workflowTask.boundedBy(WaitBound(intervalMs = 250, atMostMs = 5000)),
    // Matching dispatches a Nexus task to the handler worker's poll, the Case's handler answers it
    // at once, and matching hands the reply to the waiting dispatch:
    // service/matching/matching_engine.go:2721-2802, :2884-2909,
    // common/testing/testpilot/temporal/worker/interpreter.go:326-430.
    CauseKind.handlerReply.boundedBy(WaitBound(intervalMs = 250, atMostMs = 5000)),
    // A timer fires at or after its deadline from the timer queue, and one closer than the queue's
    // maximum time shift (1 s) is pushed out to it: chasm/lib/activity/tasks.go:154-168,
    // service/history/timer_queue_active_task_executor.go:1134-1160,
    // service/history/shard/task_key_manager.go:96-101. The bound is the slack after the deadline.
    CauseKind.timer.boundedBy(WaitBound(intervalMs = 250, atMostMs = 3000))
  ),
  // An activity starts with no attempt, its schedule counts the first as 1, and each retry adds one
  // to the same activity execution, so every attempt is of its one run:
  // chasm/lib/activity/activity.go:196, statemachine.go:349-352, :681-686.
  attemptNumbering = Some(AttemptNumbering(first = 1, oneRun = true)),
  // A call whose response no read waits on, a Driver wait (AwaitLearned, AwaitCommand) or a control
  // (Fault, Hold, Release) has no API fact that bounds it (.plans/API_BEHAVIOR_HINTS.md, W-11, W-14,
  // W-15, W-19, W-20), so it is bounded by the limit Temporal Cases have always run such
  // instructions under: 10 s and one attempt, since no instruction is retried.
  instructionDefaults = Some(InstructionLimit(timeoutMs = 10000, attempts = 1)),
  // A read is made after the Run recorded what came before it, and what it reads is visible to it
  // at once or within the wait derived from the hints above; a Run Event records a call's answer or
  // a worker's answer once the server took it. So the Run's record order is the order of one
  // operation's evidence across its sources.
  runOrderIsCausal = true
)
