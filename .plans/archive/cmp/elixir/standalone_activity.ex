# authoring: header
defmodule Temporal.Feature.StandaloneActivity do
  @moduledoc """
  The standalone activity Model.

  A Temporal activity started directly through `StartActivityExecution`, with no workflow: the
  product machine says what the caller sees through `DescribeActivityExecution`, the protocol
  machine says how the server gets there and refines it. Grounded in
  `chasm/lib/activity/statemachine.go` and `proto/v1/activity_state.proto`. Reset is deferred
  (like cancellation in the Nexus Model) and not modeled. Heartbeat timeout is not modeled.

  Standalone activities write no history events, so every fact is a status read through
  `DescribeActivityExecution` or a result read through `PollActivityExecution`.

  Naming: every spec name is an atom, verbatim (`:attemptResult`, `:pauseRequested`,
  `:statusCompleted`). Atoms have no namespace, so `:completed` as a phase, as an `AttemptResult`
  member and as a scenario name never collide: each declaration kind is its own registry, and a
  reference is resolved in the registry its position implies. Domains and states are nested
  modules (`Phase`, `ProtocolState`); each machine is a nested module named by camelizing its
  atom (`ActivityProtocol`), holding one snake_case function per step (`attempt_result/2`).

  Read from top to bottom: vocabulary, the two machines, what they promise, what the set asks.
  """

  use Umpire.Model

  alias Temporal.Feature.Worker
  # The deadline domain is the Nexus Model's: a domain is a module, so reuse is an alias.
  alias Temporal.Feature.Nexus.Caller.Timeout

  # authoring: entities

  ## Entities
  #
  # An activity is named by the id the caller chose for it.

  entity :activity, key: :activityId

  # authoring: domains

  ## The input domains
  #
  # A class is one member of a domain, and a constructor that carries finite fields contributes
  # one class per assignment of them: `{:failed, retryable: :boolean}` is one constructor and two
  # classes, `{:failed, false}` and `{:failed, true}`, which is the granularity an example is
  # written at and what mirrors a protobuf oneof.

  domain AttemptResult, [:completed, {:failed, retryable: :boolean}, :canceled]

  domain Delivery, [:accepted, :notFound]

  domain Control, [:pause, :unpause, :requestCancel, :terminate]

  # authoring: actions

  ## Actions
  #
  # The caller starts and controls the activity; the worker's poll receives the task and its
  # response settles the attempt. A fault is an ordinary action of a declared party, and a timer
  # is `system` behavior the machine owns, so neither is a separate kind.

  action :start,
    party: :caller,
    creates: :activity,
    schema: "temporal.api.workflowservice.v1.StartActivityExecutionRequest",
    input: [scheduleToClose: Timeout, scheduleToStart: Timeout, startToClose: Timeout]

  # The worker's poll receives the task (`PollActivityTaskQueue`).
  action :attemptStart,
    party: :worker,
    on: :activity,
    schema: "temporal.api.workflowservice.v1.PollActivityTaskQueueResponse"

  action :attemptResult,
    party: :worker,
    on: :activity,
    schema:
      "RespondActivityTaskCompletedRequest | RespondActivityTaskFailedRequest | " <>
        "RespondActivityTaskCanceledRequest",
    input: [result: AttemptResult],
    examples: %{
      {:failed, false} => "ApplicationFailure nonRetryable",
      {:failed, true} => "ApplicationFailure retryable"
    }

  action :control,
    party: :caller,
    on: :activity,
    schema:
      "PauseActivityExecutionRequest | UnpauseActivityExecutionRequest | " <>
        "RequestCancelActivityExecutionRequest | TerminateActivityExecutionRequest",
    input: [control: Control],
    results: Delivery

  # The worker stops polling. An action that names no entity is behavior no entity records, so the
  # machines keep their state and record nothing at it. Declared by the worker module.
  import_actions Worker, [:workerStop]

  # authoring: observation

  ## The derived observation
  #
  # Each poll raises the attempt count, and no status names the count, so it is read back
  # through `DescribeActivityExecution`. Every other evidence name resolves against the
  # realization's catalog, which is why only a derived observation is declared.

  observation :attemptCount, on: :activity, read: :attempt

  # authoring: product

  ## The product machine
  #
  # What the caller sees through Describe, with no account of how. Every Property written against
  # it is carried to the protocol machine by the refinement declared there. Describe reads a retry,
  # so unlike the Nexus product this one sees a retryable failure; what it cannot see is the
  # backoff between the failure and the next poll, and the difference between a pause and a
  # requested one.

  domain ProductPhase, [
    :scheduled,
    :started,
    :paused,
    :cancelRequested,
    :completed,
    :failed,
    :canceled,
    :terminated,
    :timedOut
  ]

  defstate ProductState, phase: ProductPhase

  domain ProductOutcome, [:accepted, :notFound]

  domain ProductFact, [
    :statusScheduled,
    :statusStarted,
    :statusPaused,
    :statusCancelRequested,
    :statusCompleted,
    :statusFailed,
    :statusCanceled,
    :statusTerminated,
    :statusTimedOut
  ]

  # The five phases the product machine ends on. Read by the machine and by `terminalIsFinal`.
  @productTerminal [:completed, :failed, :canceled, :terminated, :timedOut]

  defmachine :activityProduct,
    for: :activity,
    state: ProductState,
    outcome: ProductOutcome,
    facts: ProductFact do
    starts [:scheduled]
    ends @productTerminal
    timers [:timeout]

    evidence statusScheduled: :statusScheduled,
             statusStarted: :statusStarted,
             statusPaused: :statusPaused,
             statusCancelRequested: :statusCancelRequested,
             statusCompleted: :statusCompleted,
             statusFailed: :statusFailed,
             statusCanceled: :statusCanceled,
             statusTerminated: :statusTerminated,
             statusTimedOut: :statusTimedOut

    # The worker's poll receives the task. Only a scheduled activity has one to receive.
    defstep attemptStart(state) do
      case state.phase do
        :scheduled -> moves(:started, [:statusStarted])
        _ -> []
      end
    end

    # The worker's response to a started attempt. Unlike the Nexus product, the retry is visible: a
    # retryable failure reads as SCHEDULED again through Describe, with a higher attempt count
    # (statemachine.go's `TransitionRescheduled`), and under a requested cancel it resolves the
    # activity as canceled. A cancel response counts only where a cancel was requested.
    defstep attemptResult(state, result) do
      case {state.phase, result} do
        {phase, :completed} when phase in [:started, :cancelRequested] ->
          moves(:completed, [:statusCompleted])

        {phase, {:failed, false}} when phase in [:started, :cancelRequested] ->
          moves(:failed, [:statusFailed])

        {:started, {:failed, true}} ->
          moves(:scheduled, [:statusScheduled])

        {:cancelRequested, {:failed, true}} ->
          moves(:canceled, [:statusCanceled])

        {:cancelRequested, :canceled} ->
          moves(:canceled, [:statusCanceled])

        {:started, :canceled} ->
          []

        {phase, _} when phase not in [:started, :cancelRequested] ->
          []
      end
    end

    # The caller's control requests. After the activity is over every one is not found and
    # changes nothing; a repeated cancel request is accepted again and reads the same status.
    defstep control(state, control) do
      case {state.phase, control} do
        {phase, _} when phase in @productTerminal ->
          not_found()

        {phase, :pause} when phase in [:scheduled, :started] ->
          moves(:paused, [:statusPaused])

        {_, :pause} ->
          []

        {:paused, :unpause} ->
          moves(:scheduled, [:statusScheduled])

        {_, :unpause} ->
          []

        # These four are every phase the first clause left, so no fallback follows: the macro
        # proves the list complete, and would report a `{_, :requestCancel} -> []` here as dead.
        {phase, :requestCancel} when phase in [:scheduled, :started, :paused, :cancelRequested] ->
          moves(:cancelRequested, [:statusCancelRequested])

        {_, :terminate} ->
          moves(:terminated, [:statusTerminated])
      end
    end

    # The worker stopping is a fault the Run records and the activity does not feel. The product
    # machine cannot see it: a step that kept the state and recorded nothing would be read by the
    # refinement as every stutter.
    defstep workerStop(_state), do: []

    # One of the activity's deadlines firing. Which deadline is the protocol's account of how, so
    # the product machine has one timer, and it fires while the activity is open.
    defstep timeout(state) do
      case state.phase do
        phase when phase in [:scheduled, :started, :cancelRequested, :paused] ->
          moves(:timedOut, [:statusTimedOut])

        _ ->
          []
      end
    end
  end

  # authoring: protocol

  ## The protocol machine
  #
  # How the server gets there: the backoff the product machine cannot see, the pause a started
  # attempt only requests, the three timers the start request sets, and the attempt count a poll
  # raises. Written against the same actions, so a Property proved on the product machine is
  # carried here by the refinement.
  #
  # The machine begins before the activity exists: `:unstarted` is the "no instance yet" member,
  # and it is what makes the deadline fields reachable at anything but their first value.

  domain Phase, [
    :unstarted,
    :scheduled,
    :backingOff,
    :started,
    :paused,
    :pauseRequested,
    :cancelRequested,
    :completed,
    :failed,
    :canceled,
    :terminated,
    :timedOut
  ]

  # Which timer fired. The Describe status does not say, so the fact carries it.
  domain TimeoutType, [:scheduleToClose, :scheduleToStart, :startToClose]

  # The attempt count is bounded by the Limits in the design; nothing wires the Limits into a
  # machine's state, so the bound is written here and `succ/1` saturates at it.
  @attemptBound 2

  defstate ProtocolState,
    phase: Phase,
    attempts: 0..@attemptBound,
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout

  domain ProtocolOutcome, [:accepted, :notFound]

  # As the product's, with the timed-out status carrying its type and the derived attempt count.
  domain ProtocolFact, [
    :statusScheduled,
    :statusStarted,
    :statusPaused,
    :statusCancelRequested,
    :statusCompleted,
    :statusFailed,
    :statusCanceled,
    :statusTerminated,
    {:statusTimedOut, timeoutType: TimeoutType},
    :attemptCount
  ]

  defmachine :activityProtocol,
    for: :activity,
    state: ProtocolState,
    outcome: ProtocolOutcome,
    facts: ProtocolFact do
    # The five phases the design ends on. A control that arrives after one of them is not found.
    @terminal [:completed, :failed, :canceled, :terminated, :timedOut]

    # Started and not yet over: the phases the schedule-to-close deadline covers.
    @running [:scheduled, :backingOff, :started, :paused, :pauseRequested, :cancelRequested]

    refines :activityProduct, map: :productOf
    starts [:unstarted]
    ends @terminal
    timers [:backoff, :scheduleToClose, :scheduleToStart, :startToClose]
    unobservable [:backoff]

    evidence statusScheduled: :statusScheduled,
             statusStarted: :statusStarted,
             statusPaused: :statusPaused,
             statusCancelRequested: :statusCancelRequested,
             statusCompleted: :statusCompleted,
             statusFailed: :statusFailed,
             statusCanceled: :statusCanceled,
             statusTerminated: :statusTerminated,
             statusTimedOut: :statusTimedOut,
             attemptCount: :attemptCount

    # The caller's start request. It names the activity's three deadlines, each a state field
    # because whether a timer fires is a question about the activity and not about the request
    # that started it.
    defstep start(state, schedule_to_close, schedule_to_start, start_to_close) do
      case state.phase do
        :unstarted ->
          moves(:scheduled, [:statusScheduled],
            attempts: 0,
            scheduleToClose: schedule_to_close,
            scheduleToStart: schedule_to_start,
            startToClose: start_to_close
          )

        _ ->
          []
      end
    end

    # The worker's poll receives the task and the attempt count rises. No status change records
    # the count, so it is read back through the `attemptCount` observation.
    defstep attemptStart(state) do
      case state.phase do
        :scheduled -> moves(:started, [:statusStarted, :attemptCount], attempts: succ(state.attempts))
        _ -> []
      end
    end

    # The worker's response to a started attempt. It dispatches on the phase as well as the result
    # because the three phases that hold an attempt answer a retryable failure differently. From
    # `:started` the activity backs off, and Describe reads SCHEDULED again with the raised count
    # (`TransitionRescheduled`), so the row records both. Under a requested cancel, statemachine.go
    # makes CANCEL_REQUESTED a source of Completed, Failed, Canceled, TimedOut and Terminated, and a
    # retryable failure there resolves the activity as canceled. Under a requested pause it parks
    # the activity (`TransitionAttemptFailedWhilePauseRequested`). A cancel response with no cancel
    # requested is not enabled.
    defstep attemptResult(state, result) do
      case {state.phase, result} do
        {:started, :completed} -> moves(:completed, [:statusCompleted])
        {:started, {:failed, false}} -> moves(:failed, [:statusFailed])
        {:started, {:failed, true}} -> moves(:backingOff, [:statusScheduled, :attemptCount])
        {:started, :canceled} -> []

        {:cancelRequested, :completed} -> moves(:completed, [:statusCompleted])
        {:cancelRequested, {:failed, false}} -> moves(:failed, [:statusFailed])
        {:cancelRequested, {:failed, true}} -> moves(:canceled, [:statusCanceled])
        {:cancelRequested, :canceled} -> moves(:canceled, [:statusCanceled])

        {:pauseRequested, :completed} -> moves(:completed, [:statusCompleted])
        {:pauseRequested, {:failed, false}} -> moves(:failed, [:statusFailed])
        {:pauseRequested, {:failed, true}} -> moves(:paused, [:statusPaused])
        {:pauseRequested, :canceled} -> []

        {phase, _} when phase not in [:started, :cancelRequested, :pauseRequested] -> []
      end
    end

    # The caller's control requests. A pause of a started attempt is only requested, and the
    # Describe status reads PAUSE_REQUESTED; the fact is `:statusPaused` for both, as the product
    # cannot tell them apart. A cancel request and a termination after the activity is over are not
    # found; a control before it exists is not enabled.
    defstep control(state, control) do
      case {state.phase, control} do
        {phase, :pause} when phase in [:scheduled, :backingOff] -> moves(:paused, [:statusPaused])
        {:started, :pause} -> moves(:pauseRequested, [:statusPaused])
        {_, :pause} -> []

        {:paused, :unpause} -> moves(:scheduled, [:statusScheduled])
        {:pauseRequested, :unpause} -> moves(:started, [:statusStarted])
        {_, :unpause} -> []

        {:unstarted, :requestCancel} -> []
        {phase, :requestCancel} when phase in @terminal -> not_found()
        {_, :requestCancel} -> moves(:cancelRequested, [:statusCancelRequested])

        {:unstarted, :terminate} -> []
        {phase, :terminate} when phase in @terminal -> not_found()
        {_, :terminate} -> moves(:terminated, [:statusTerminated])
      end
    end

    # The worker stopping is a fault the Run records and the activity does not feel, so the step
    # keeps the state and records nothing. On a path it is confirmed by the evidence of the step
    # after it.
    defstep workerStop(state), do: stay()

    # The backoff timer. It is what makes `:backingOff` a phase the activity leaves rather than a
    # state it is stuck in, and it records nothing: the retry was recorded when the attempt failed.
    defstep backoff(state) do
      case state.phase do
        :backingOff -> moves(:scheduled, [])
        _ -> []
      end
    end

    # The schedule-to-close deadline covers the whole activity, so it fires in every running
    # phase, and only when the start request set it.
    defstep scheduleToClose(state) do
      case {state.phase, state.scheduleToClose} do
        {phase, :expires} when phase in @running ->
          moves(:timedOut, [{:statusTimedOut, :scheduleToClose}])

        {_, _} ->
          []
      end
    end

    # The schedule-to-start deadline covers the wait for a worker to poll, so it stops at the start.
    defstep scheduleToStart(state) do
      case {state.phase, state.scheduleToStart} do
        {phase, :expires} when phase in [:scheduled, :backingOff] ->
          moves(:timedOut, [{:statusTimedOut, :scheduleToStart}])

        {_, _} ->
          []
      end
    end

    # The start-to-close deadline covers the attempt, requested pause or cancel included, so it
    # begins at the start.
    defstep startToClose(state) do
      case {state.phase, state.startToClose} do
        {phase, :expires} when phase in [:started, :pauseRequested, :cancelRequested] ->
          moves(:timedOut, [{:statusTimedOut, :startToClose}])

        {_, _} ->
          []
      end
    end

    # How a protocol state reads as a product state. Backing off is still scheduled, and an
    # activity not yet started reads as scheduled, because the product machine begins there. A
    # requested pause reads as started: the worker still holds the attempt, so every answer it can
    # give is a product row from started, and the request itself is a stutter. A phase of the same
    # name is that phase. Every other field is hidden, which is what a map that does not read it
    # says.
    defmap productOf(state) do
      case state.phase do
        phase when phase in [:unstarted, :scheduled, :backingOff] -> %ProductState{phase: :scheduled}
        :pauseRequested -> %ProductState{phase: :started}
        phase -> %ProductState{phase: phase}
      end
    end
  end

  # authoring: properties

  ## What the machines promise
  #
  # A same-step claim names the action it is about under `when:` and holds of the step that action
  # produces; a transition claim holds of the step before and the step after. A functional Query
  # realizes a same-step claim; a transition claim is searched and verified, never realized.
  # (`after` is reserved in Elixir, so a transition claim's second step is `next`.)

  # Once an activity is over, no step changes its phase. Declared on the product machine and read
  # on the protocol machine through the map.
  defproperty :terminalIsFinal,
    machine: :activityProduct,
    holds: fn before, next ->
      before.state.phase not in @productTerminal or next.state.phase == before.state.phase
    end

  # A completed attempt settles the activity as completed, and the status records it.
  defproperty :completes,
    machine: :activityProtocol,
    when: attemptResult(:completed),
    holds: fn step -> step.state.phase == :completed and :statusCompleted in step.facts end

  # A non-retryable failure settles the activity as failed, and the status records it.
  defproperty :nonRetryableFails,
    machine: :activityProtocol,
    when: attemptResult({:failed, false}),
    holds: fn step -> step.state.phase == :failed and :statusFailed in step.facts end

  # Completed on the second attempt of an activity with no deadline set. A claim fixes one state,
  # so every field is named.
  # `struct!/2` rather than a `%ProtocolState{}` literal: the struct module is created while this
  # module body runs, and a literal at the top level is expanded before that.
  @completedOnRetry struct!(ProtocolState, %{
    phase: :completed,
    attempts: 2,
    scheduleToClose: :unset,
    scheduleToStart: :unset,
    startToClose: :unset
  })

  # A completed retried attempt settles the activity as completed on its second attempt: the count
  # the second poll raised is two, and the status records the completion.
  defproperty :retryCompletes,
    machine: :activityProtocol,
    when: attemptResult(:completed),
    holds: fn step -> step.state == @completedOnRetry and :statusCompleted in step.facts end

  # A cancel request of a started attempt is recorded as requested; the attempt keeps running.
  defproperty :cancelRequestedWhileStarted,
    machine: :activityProtocol,
    when: control(:requestCancel),
    holds: fn step ->
      step.state.phase == :cancelRequested and :statusCancelRequested in step.facts
    end

  # The worker honors the cancel request and the status records it.
  defproperty :canceledByWorker,
    machine: :activityProtocol,
    when: attemptResult(:canceled),
    holds: fn step -> step.state.phase == :canceled and :statusCanceled in step.facts end

  # A termination settles the activity, and the status records it.
  defproperty :terminated,
    machine: :activityProtocol,
    when: control(:terminate),
    holds: fn step -> step.state.phase == :terminated and :statusTerminated in step.facts end

  # A paused activity is never dispatched: no step moves it straight to started. Declared on the
  # product machine and read on the protocol machine through the map.
  defproperty :pausedIsNotDispatched,
    machine: :activityProduct,
    holds: fn before, next -> before.state.phase != :paused or next.state.phase != :started end

  # The schedule-to-start deadline settles an activity no worker polled as timed out, and the
  # status records which deadline it was.
  defproperty :scheduleToStartFires,
    machine: :activityProtocol,
    when: scheduleToStart,
    holds: fn step ->
      step.state.phase == :timedOut and {:statusTimedOut, :scheduleToStart} in step.facts
    end

  # The start-to-close deadline settles a started attempt no worker answered as timed out.
  defproperty :startToCloseFires,
    machine: :activityProtocol,
    when: startToClose,
    holds: fn step ->
      step.state.phase == :timedOut and {:statusTimedOut, :startToClose} in step.facts
    end

  # authoring: scenarios

  ## The paths the Queries run
  #
  # A protocol Scenario names its classed actions with their inputs and its start by its phase.
  # The start request sets no deadline unless the path is about one. An action with no input is
  # written bare; the macro reads the list and never evaluates it.

  defscenario :completed,
    model: :activityProtocol,
    starts: :unstarted,
    actions: [start(:unset, :unset, :unset), attemptStart, attemptResult(:completed)]

  defscenario :nonRetryable,
    model: :activityProtocol,
    starts: :unstarted,
    actions: [start(:unset, :unset, :unset), attemptStart, attemptResult({:failed, false})]

  # The retryable failure backs the activity off; the backoff timer fires and records nothing; the
  # second poll starts the retried attempt, which completes.
  defscenario :retriedThenCompleted,
    model: :activityProtocol,
    starts: :unstarted,
    actions: [
      start(:unset, :unset, :unset),
      attemptStart,
      attemptResult({:failed, true}),
      backoff,
      attemptStart,
      attemptResult(:completed)
    ]

  defscenario :cancelRequestedThenCanceled,
    model: :activityProtocol,
    starts: :unstarted,
    actions: [
      start(:unset, :unset, :unset),
      attemptStart,
      control(:requestCancel),
      attemptResult(:canceled)
    ]

  # The worker stops, so nothing polls; the caller terminates the scheduled activity.
  defscenario :terminatedWhileScheduled,
    model: :activityProtocol,
    starts: :unstarted,
    actions: [start(:unset, :unset, :unset), workerStop, control(:terminate)]

  # Paused before any poll, resumed, then polled and completed.
  defscenario :pausedThenCompleted,
    model: :activityProtocol,
    starts: :unstarted,
    actions: [
      start(:unset, :unset, :unset),
      control(:pause),
      control(:unpause),
      attemptStart,
      attemptResult(:completed)
    ]

  # The start request sets the schedule-to-start deadline; the worker stops, so nothing polls; the
  # deadline fires.
  defscenario :scheduleToStartExpires,
    model: :activityProtocol,
    starts: :unstarted,
    actions: [start(:unset, :expires, :unset), workerStop, scheduleToStart]

  # The start request sets the start-to-close deadline; the worker polls and never answers; the
  # deadline fires.
  defscenario :startToCloseExpires,
    model: :activityProtocol,
    starts: :unstarted,
    actions: [start(:unset, :unset, :expires), attemptStart, startToClose]

  deflimits :three, steps: 3, actions: 3, search: 4096
  deflimits :four, steps: 4, actions: 4, search: 32_768
  deflimits :six, steps: 6, actions: 6, search: 262_144

  # authoring: queries

  ## The Queries
  #
  # Eight find their same-step claim on their path and are realized by the set below. The two
  # product claims are verified over every trace of one path each, outside the set, because a
  # verify Query realizes nothing. A verify whose claim never fires on its path is an error, not a
  # pass (`require_firing:` defaults to true).

  defquery :completion, find: :completes, in: :completed, limits: :three
  defquery :nonRetryableFailure, find: :nonRetryableFails, in: :nonRetryable, limits: :three
  defquery :retry, find: :retryCompletes, in: :retriedThenCompleted, limits: :six
  defquery :cancel, find: :canceledByWorker, in: :cancelRequestedThenCanceled, limits: :four
  defquery :terminate, find: :terminated, in: :terminatedWhileScheduled, limits: :three
  defquery :pauseResume, find: :completes, in: :pausedThenCompleted, limits: :six

  defquery :scheduleToStartTimeout,
    find: :scheduleToStartFires,
    in: :scheduleToStartExpires,
    limits: :three

  defquery :startToCloseTimeout,
    find: :startToCloseFires,
    in: :startToCloseExpires,
    limits: :three

  defquery :terminalHolds, verify: :terminalIsFinal, in: :completed, limits: :three
  defquery :pauseHolds, verify: :pausedIsNotDispatched, in: :pausedThenCompleted, limits: :six

  # authoring: set

  ## The functional set
  #
  # The Case drives the caller and the worker. No repeat: standalone activities are CHASM only.

  defset :standaloneActivityTests,
    purpose: :functional,
    bind: [caller: :driven, worker: :driven],
    queries: [
      :completion,
      :nonRetryableFailure,
      :retry,
      :cancel,
      :terminate,
      :pauseResume,
      :scheduleToStartTimeout,
      :startToCloseTimeout
    ]

  ## The canary set
  #
  # The worker is `:observed`: the deployment's own worker answers, and the verifier checks the
  # machine allows what it did. Both paths record evidence at every step, which is what admits a
  # canary; a path with a silent step (the backoff, the worker stop) is a gap no deployment
  # closes.

  defset :standaloneActivityCanary,
    purpose: :canary,
    bind: [caller: :driven, worker: :observed],
    queries: [:completion, :cancel]

  ## The exploratory set
  #
  # An exploration covers the protocol machine rather than listing Queries: the rows within the
  # budget's steps of a start, the results they reach and the members of the classes they claim.

  defset :standaloneActivityExploration,
    purpose: :exploratory,
    bind: [caller: :driven, worker: :driven],
    machine: :activityProtocol,
    cover: [:rows, :results, :classMembers],
    budget: :four

  # authoring: composition

  ## The activity and its worker
  #
  # The protocol machine's worker stop is a stutter row. Composed with the worker of the task
  # queue, the stop is the worker's own phase change and every poll is the worker serving, so an
  # attempt starts only while the worker polls. No set names the composition; it is what the
  # cross-entity claim is verified over.

  # The caller's view of the worker: it stops and it serves. It never resumes, because an action
  # no `sync:` line names would stay executable on its own and admit a stop, a resume and then a
  # poll; the activity's timers settle every state a stop leaves.
  defmachine :activityWorker, from: {Worker, :polling}, only: [:workerStop, :serve]

  defcompose :standaloneActivity,
    for: [:activity, {Worker, :worker}],
    state: StandaloneActivityState,
    members: [activity: :activityProtocol, worker: :activityWorker],
    sync: [
      workerStop: activity.workerStop || worker.workerStop,
      attemptStart: activity.attemptStart || worker.serve
    ],
    starts: [activity: :unstarted, worker: :polling],
    ends: [activity: [:completed, :failed, :canceled, :terminated, :timedOut]]

  # Every attempt starts with the worker polling: a stopped worker starts nothing.
  defproperty :startedByPollingWorker,
    machine: :standaloneActivity,
    when: attemptStart,
    holds: fn step -> step.state.worker.phase == :polling end

  # An attempt starts while the worker polls and fails retryably; the worker stops before the
  # retry, so the retried attempt never starts and the schedule-to-start deadline fires. The one
  # `attemptStart` on the path is what makes the claim fire, so the verify exercises it rather
  # than passing vacuously, as the earlier `stoppedBeforeDispatch` path did.
  defscenario :stoppedBeforeRetry,
    model: :standaloneActivity,
    starts: [activity: :unstarted],
    actions: [
      activity.start(:unset, :expires, :unset),
      attemptStart,
      activity.attemptResult({:failed, true}),
      activity.backoff,
      workerStop,
      activity.scheduleToStart
    ]

  defquery :stoppedWorkerStartsNothing,
    verify: :startedByPollingWorker,
    in: :stoppedBeforeRetry,
    limits: :six

  # authoring: end
end
