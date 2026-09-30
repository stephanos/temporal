// -*- mode: quint -*-
//
// # The standalone activity Model
//
// A Temporal activity started directly through `StartActivityExecution`, with no workflow.
// Grounded in `chasm/lib/activity/statemachine.go` and `proto/v1/activity_state.proto`. Reset is
// deferred (like cancellation in the Nexus Model) and not modeled. Heartbeat timeout is not
// modeled. Standalone activities write no history events, so every fact is a status read
// through `DescribeActivityExecution` or a result read through `PollActivityExecution`.
//
// Same layout as nexus_caller.qnt: vocabulary → product table and machine → protocol table and
// machine → properties, scenarios, queries, sets → composition.
//
// Typecheck:  quint typecheck standalone_activity.qnt
// Scenarios:  quint test standalone_activity.qnt --main activityProtocol
// Explore:    quint run standalone_activity.qnt --main activityProtocol --invariant properties --mbt
// Verify:     quint verify standalone_activity.qnt --main activityProtocol --invariant refinesProduct

// ### Entities and the input domains
//
// `AttemptResult`'s members carry an `Attempt` prefix for the same reason `Resolution`'s carry
// `Resolved` in the Nexus Model: the phases `Completed`, `Failed`, `Canceled` own the bare names.
module activityVocabulary {
  import umpire.* from "./umpire"

  pure val activity: Entity = { name: "activity", key: "activityId", refer: Set() }

  type AttemptResult = AttemptCompleted | AttemptFailed(bool) | AttemptCanceled

  type Control = Pause | Unpause | RequestCancel | Terminate

  pure val attemptResults: Set[AttemptResult] =
    Set(AttemptCompleted, AttemptFailed(true), AttemptFailed(false), AttemptCanceled)

  pure val controls: Set[Control] = Set(Pause, Unpause, RequestCancel, Terminate)

  // ### Actions (the part Quint cannot check: party, entity, schema, examples)
  pure val actions: str -> ActionDecl = Map(
    "start" -> { party: Caller, creates: "activity", on: "",
      schema: "temporal.api.workflowservice.v1.StartActivityExecutionRequest", examples: Set() },
    // The worker's poll receives the task (`PollActivityTaskQueue`).
    "attemptStart" -> { party: Worker, creates: "", on: "activity",
      schema: "temporal.api.workflowservice.v1.PollActivityTaskQueueResponse", examples: Set() },
    "attemptResult" -> { party: Worker, creates: "", on: "activity",
      schema: "RespondActivityTaskCompletedRequest | RespondActivityTaskFailedRequest | RespondActivityTaskCanceledRequest",
      examples: Set(("AttemptFailed(false)", "ApplicationFailure nonRetryable"),
                    ("AttemptFailed(true)", "ApplicationFailure retryable")) },
    "control" -> { party: Caller, creates: "", on: "activity",
      schema: "PauseActivityExecutionRequest | UnpauseActivityExecutionRequest | RequestCancelActivityExecutionRequest | TerminateActivityExecutionRequest",
      examples: Set() },
    "workerStop" -> decl(Worker),
  )

  // ### The derived observation: the attempt count, read from `DescribeActivityExecution`.
  pure val attemptCount: (str, str) = ("activity", "attempt")
}
