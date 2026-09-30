# authoring: header
defmodule Temporal.Feature.Nexus.Caller do
  @moduledoc """
  The Nexus caller-side Model.

  One workflow-scheduled Nexus operation, as the caller sees it: the product machine says what an
  operation does, the protocol machine says how the server gets there and refines it, and the
  functional set runs one Query per side effect that settles the operation, once per value of the
  implementation switch. Step functions and predicates rather than rows, no cancellation (fn-79)
  and no concurrency-limit setup parameter.

  Naming follows `Temporal.Feature.StandaloneActivity`: spec names are atoms, verbatim; domains,
  states and machines are nested modules; steps are snake_case functions of their machine's
  module. The query `:handlerError` and the reply `{:handlerError, true}` are one atom in two
  registries, so neither is renamed.

  Read from top to bottom: vocabulary, the two machines, what they promise, what the set asks.
  """

  use Umpire.Model

  alias Temporal.Feature.Worker

  # authoring: entities

  ## Entities
  #
  # An operation is scheduled by a caller workflow, and recorded data names one by its scheduled
  # event: every history event of the operation carries that event's id.

  entity :workflow

  entity :operation, refer: [caller: :workflow], key: :scheduledEvent

  # authoring: domains

  ## The input domains
  #
  # A class is one member of a domain, and a constructor that carries finite fields contributes
  # one class per assignment of them: `{:handlerError, retryable: :boolean}` is one constructor and
  # two classes, which is the granularity an example is written at and what mirrors a protobuf
  # oneof.

  domain Timeout, [:unset, :expires]

  domain Reply, [
    :syncSuccess,
    :async,
    :operationFailed,
    :operationCanceled,
    {:handlerError, retryable: :boolean}
  ]

  domain Resolution, [:succeeded, :failed, :canceled]

  domain Delivery, [:accepted, :notFound]

  # authoring: actions

  ## Actions
  #
  # Parties are names the feature declares by using them: `:caller`, `:handler`, `:network`,
  # `:worker`. The reserved party `:system` is the server. A fault is an ordinary action of a
  # declared party, and a timer is `:system` behavior the machine owns, so neither is a separate
  # kind.

  action :schedule,
    party: :caller,
    creates: :operation,
    schema: "temporal.api.command.v1.ScheduleNexusOperationCommandAttributes",
    input: [scheduleToClose: Timeout, scheduleToStart: Timeout, startToClose: Timeout]

  action :handlerReply,
    party: :handler,
    on: :operation,
    schema: "temporal.api.nexus.v1.StartOperationResponse | temporal.api.nexus.v1.HandlerError",
    input: [reply: Reply],
    examples: %{{:handlerError, false} => "BadRequest", {:handlerError, true} => "Internal"}

  # The Nexus HTTP completion carries no protobuf message, so it declares no schema and its
  # classes are names the realization interprets.
  action :complete,
    party: :handler,
    on: :operation,
    input: [resolution: Resolution],
    results: Delivery

  action :transportFault, party: :network, on: :operation

  # The handler's worker stops polling. An action that names no entity is behavior no entity
  # records: the Run records the fault, but nothing recorded names the operation, so the machines
  # keep their state and record nothing at it. Declared by the worker module.
  import_actions Worker, [:workerStop]

  # authoring: observation

  ## The derived observation
  #
  # A retryable attempt failure writes no history event, so the attempt count is read back through
  # `DescribeWorkflowExecution`. Every other evidence name resolves against the realization's
  # catalog, which is why only a derived observation is declared.

  observation :pendingAttempts, on: :operation, read: :attempts

  # authoring: product

  ## The product machine
  #
  # What an operation does, with no account of how. Every Property written against it is carried
  # to the protocol machine by the refinement declared there.

  domain ProductPhase, [:scheduled, :started, :succeeded, :failed, :canceled, :timedOut]

  defstate ProductState, phase: ProductPhase

  domain ProductOutcome, [:accepted, :notFound]

  domain ProductFact, [
    :nexusOperationScheduled,
    :nexusOperationStarted,
    :nexusOperationCompleted,
    :nexusOperationFailed,
    :nexusOperationCanceled,
    :nexusOperationTimedOut
  ]

  # The four phases the product machine ends on.
  @productTerminal [:succeeded, :failed, :canceled, :timedOut]

  defmachine :nexusProduct,
    for: :operation,
    state: ProductState,
    outcome: ProductOutcome,
    facts: ProductFact do
    starts [:scheduled]
    ends @productTerminal
    timers [:timeout]

    evidence nexusOperationStarted: :nexusOperationStarted,
             nexusOperationCompleted: :nexusOperationCompleted,
             nexusOperationFailed: :nexusOperationFailed,
             nexusOperationCanceled: :nexusOperationCanceled,
             nexusOperationTimedOut: :nexusOperationTimedOut

    # The handler's reply to the server's start request. An operation that has not started yet is
    # the only one a reply can move.
    defstep handlerReply(state, reply) do
      case {state.phase, reply} do
        {phase, _} when phase != :scheduled -> []
        {_, :syncSuccess} -> moves(:succeeded, [:nexusOperationCompleted])
        {_, :async} -> moves(:started, [:nexusOperationStarted])
        {_, :operationFailed} -> moves(:failed, [:nexusOperationFailed])
        {_, :operationCanceled} -> moves(:canceled, [:nexusOperationCanceled])
        # A retryable handler error leaves the operation where it is: the product machine does not
        # know about backing off, which is the whole of what the protocol machine adds.
        {_, {:handlerError, true}} -> []
        {_, {:handlerError, false}} -> moves(:failed, [:nexusOperationFailed])
      end
    end

    # An asynchronous completion. A completion that arrives after the operation is over is not
    # found, and changes nothing.
    defstep complete(state, resolution) do
      case {state.phase, resolution} do
        {phase, _} when phase in @productTerminal -> not_found()
        {_, :succeeded} -> moves(:succeeded, [:nexusOperationCompleted])
        {_, :failed} -> moves(:failed, [:nexusOperationFailed])
        {_, :canceled} -> moves(:canceled, [:nexusOperationCanceled])
      end
    end

    # A transport fault is an ordinary action of the network. The product machine cannot see one:
    # whether a delivery was retried is the protocol's account of how, not what.
    defstep transportFault(_state), do: []

    # The handler's worker stopping is a fault the Run records and the operation does not feel. The
    # product machine cannot see it, like the transport fault: a step that kept the state and
    # recorded nothing would be indistinguishable from a stutter, and the refinement would read
    # every stutter as this step.
    defstep workerStop(_state), do: []

    # One of the operation's deadlines firing. Which deadline is the protocol's account of how, so
    # the product machine has one timer, and it fires while the operation runs.
    defstep timeout(state) do
      case state.phase do
        phase when phase in [:scheduled, :started] -> moves(:timedOut, [:nexusOperationTimedOut])
        _ -> []
      end
    end
  end

  # authoring: protocol

  ## The protocol machine
  #
  # How the server gets there: the retry the product machine cannot see, the three timers the
  # schedule command sets, and the attempt count a retryable failure raises. Written against the
  # same actions, so a Property proved on the product machine is carried here by the refinement.
  #
  # The machine begins before the operation exists: a state struct has no "no instance yet"
  # member, so `:unscheduled` is that member, and it is what makes the three deadline fields
  # reachable at anything but their first value; the schedule command is what sets them.
  #
  # Not here, for reasons recorded rather than silent: the `cancel` field and its rows (fn-79),
  # and the concurrency-limit rejection. The limit exists (one dynamic-config key per
  # implementation, and the schedule command fails the workflow task at it without writing a
  # `NexusOperationScheduled` event), but a step function does not read the setup, the key and
  # value differ per switch value, and the rejection names no operation, so it is not modeled until
  # a Query needs it.

  domain Phase, [
    :unscheduled,
    :scheduled,
    :backingOff,
    :started,
    :succeeded,
    :failed,
    :canceled,
    :timedOut
  ]

  # Which timer fired. The history event records it, so a Contract that did not check it would
  # pass a run that timed out on the wrong deadline.
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

  domain ProtocolFact, [
    :nexusOperationScheduled,
    :nexusOperationStarted,
    :nexusOperationCompleted,
    :nexusOperationFailed,
    :nexusOperationCanceled,
    {:nexusOperationTimedOut, timeoutType: TimeoutType},
    :pendingAttempts
  ]

  defmachine :nexusProtocol,
    for: :operation,
    state: ProtocolState,
    outcome: ProtocolOutcome,
    facts: ProtocolFact do
    # The four phases the design ends on. A completion that arrives after one of them is not found.
    @terminal [:succeeded, :failed, :canceled, :timedOut]

    # Scheduled and not yet over: the phases a completion resolves and a timer can fire in.
    @running [:scheduled, :backingOff, :started]

    refines :nexusProduct, map: :productOf
    starts [:unscheduled]
    ends @terminal
    timers [:backoff, :scheduleToClose, :scheduleToStart, :startToClose]
    unobservable [:backoff]

    evidence nexusOperationScheduled: :nexusOperationScheduled,
             nexusOperationStarted: :nexusOperationStarted,
             nexusOperationCompleted: :nexusOperationCompleted,
             nexusOperationFailed: :nexusOperationFailed,
             nexusOperationCanceled: :nexusOperationCanceled,
             nexusOperationTimedOut: :nexusOperationTimedOut,
             pendingAttempts: :pendingAttempts

    # The caller's schedule command. It names the operation's three deadlines, and every one of
    # them is a state field because whether a timer fires is a question about the operation and
    # not about the command that started it.
    defstep schedule(state, schedule_to_close, schedule_to_start, start_to_close) do
      case state.phase do
        :unscheduled ->
          moves(:scheduled, [:nexusOperationScheduled],
            attempts: 0,
            scheduleToClose: schedule_to_close,
            scheduleToStart: schedule_to_start,
            startToClose: start_to_close
          )

        _ ->
          []
      end
    end

    # The handler's reply to the server's start request. What the product machine cannot see is
    # the last arm: a retryable failure backs the operation off and raises its attempt count, and
    # the count is read back through the `pendingAttempts` observation because no history event
    # records it.
    defstep handlerReply(state, reply) do
      case {state.phase, reply} do
        {phase, _} when phase != :scheduled ->
          []

        {_, :syncSuccess} ->
          moves(:succeeded, [:nexusOperationCompleted])

        {_, :async} ->
          moves(:started, [:nexusOperationStarted])

        {_, :operationFailed} ->
          moves(:failed, [:nexusOperationFailed])

        {_, :operationCanceled} ->
          moves(:canceled, [:nexusOperationCanceled])

        {_, {:handlerError, false}} ->
          moves(:failed, [:nexusOperationFailed])

        {_, {:handlerError, true}} ->
          moves(:backingOff, [:pendingAttempts], attempts: succ(state.attempts))
      end
    end

    # A transport fault is the same failure arriving as a dropped delivery rather than as a reply.
    defstep transportFault(state) do
      case state.phase do
        :scheduled -> moves(:backingOff, [:pendingAttempts], attempts: succ(state.attempts))
        _ -> []
      end
    end

    # The handler's worker stopping is a fault the Run records and the operation does not feel, so
    # the step keeps the state and records nothing. On a path it is confirmed by the evidence of
    # the step after it, and the Case says so in a Known Gap.
    defstep workerStop(state), do: stay()

    # An asynchronous completion. Before a start, the server records a Started event first, which
    # is why the evidence is two facts and not one, and why the product machine, which has no
    # `:backingOff` phase to have skipped, could write the completion alone. The Lean file
    # prepends the Started fact with `++`; here it is two arms per resolution, because a list
    # operator is outside the expression subset and a literal list is not.
    defstep complete(state, resolution) do
      case {state.phase, resolution} do
        {phase, _} when phase in @terminal -> not_found()
        {:unscheduled, _} -> []
        {:started, :succeeded} -> moves(:succeeded, [:nexusOperationCompleted])
        {:started, :failed} -> moves(:failed, [:nexusOperationFailed])
        {:started, :canceled} -> moves(:canceled, [:nexusOperationCanceled])
        {_, :succeeded} -> moves(:succeeded, [:nexusOperationStarted, :nexusOperationCompleted])
        {_, :failed} -> moves(:failed, [:nexusOperationStarted, :nexusOperationFailed])
        {_, :canceled} -> moves(:canceled, [:nexusOperationStarted, :nexusOperationCanceled])
      end
    end

    # The backoff timer. It is what makes `:backingOff` a phase the operation leaves rather than a
    # state it is stuck in, and it records nothing: a retry writes no history event.
    defstep backoff(state) do
      case state.phase do
        :backingOff -> moves(:scheduled, [])
        _ -> []
      end
    end

    # The schedule-to-close deadline covers the whole operation, so it fires in every running
    # phase, and only when the schedule command set it.
    defstep scheduleToClose(state) do
      case {state.phase, state.scheduleToClose} do
        {phase, :expires} when phase in @running ->
          moves(:timedOut, [{:nexusOperationTimedOut, :scheduleToClose}])

        {_, _} ->
          []
      end
    end

    # The schedule-to-start deadline covers the wait for the handler to accept, so it stops at the
    # start.
    defstep scheduleToStart(state) do
      case {state.phase, state.scheduleToStart} do
        {phase, :expires} when phase in [:scheduled, :backingOff] ->
          moves(:timedOut, [{:nexusOperationTimedOut, :scheduleToStart}])

        {_, _} ->
          []
      end
    end

    # The start-to-close deadline covers the handler's own work, so it begins at the start.
    defstep startToClose(state) do
      case {state.phase, state.startToClose} do
        {:started, :expires} -> moves(:timedOut, [{:nexusOperationTimedOut, :startToClose}])
        {_, _} -> []
      end
    end

    # How a protocol state reads as a product state. A phase of the same name is that phase;
    # backing off is still scheduled, because the product machine cannot see a retry; and an
    # operation not yet scheduled reads as scheduled, because the product machine begins there.
    # Every other field is hidden, which is what a map that does not read it says.
    defmap productOf(state) do
      case state.phase do
        phase when phase in [:unscheduled, :scheduled, :backingOff] -> %ProductState{phase: :scheduled}
        phase -> %ProductState{phase: phase}
      end
    end
  end

  # authoring: properties

  ## What the machines promise
  #
  # A same-step claim names the action it is about under `when:` and holds of the step that action
  # produces; a transition claim holds of the step before and the step after. A functional Query
  # realizes a same-step claim, because the Case's Contract is the claim's clause triggered by the
  # action the Case performs; a transition claim is searched and verified, never realized.

  # Once an operation is over, no step changes its phase. Declared on the product machine and read
  # on the protocol machine through the map.
  defproperty :terminalIsFinal,
    machine: :nexusProduct,
    holds: fn before, next ->
      before.state.phase not in @productTerminal or next.state.phase == before.state.phase
    end

  # A synchronous reply settles the operation as succeeded, and the completed event records it.
  defproperty :syncSucceeds,
    machine: :nexusProtocol,
    when: handlerReply(:syncSuccess),
    holds: fn step -> step.state.phase == :succeeded and :nexusOperationCompleted in step.facts end

  # An asynchronous reply starts the operation, and the started event records it.
  defproperty :asyncStarts,
    machine: :nexusProtocol,
    when: handlerReply(:async),
    holds: fn step -> step.state.phase == :started and :nexusOperationStarted in step.facts end

  # A successful completion is recorded by the completed event. Neither the phase nor the outcome
  # is fixed: a completion resolves any running phase, and `:accepted` is every earlier step's
  # outcome too, so a clause fixing it would be answered before the completion.
  defproperty :completionSucceeds,
    machine: :nexusProtocol,
    when: complete(:succeeded),
    holds: fn step -> :nexusOperationCompleted in step.facts end

  # A failed completion is recorded by the failed event.
  defproperty :completionFails,
    machine: :nexusProtocol,
    when: complete(:failed),
    holds: fn step -> :nexusOperationFailed in step.facts end

  # A non-retryable handler error settles the operation as failed, and the failed event records it.
  defproperty :handlerErrorFails,
    machine: :nexusProtocol,
    when: handlerReply({:handlerError, false}),
    holds: fn step -> step.state.phase == :failed and :nexusOperationFailed in step.facts end

  # Succeeded on the second attempt of an operation with no deadline set. A claim fixes one state,
  # so every field is named.
  # `struct!/2` rather than a `%ProtocolState{}` literal: the struct module is created while this
  # module body runs, and a literal at the top level is expanded before that.
  @succeededOnRetry struct!(ProtocolState, %{
    phase: :succeeded,
    attempts: 1,
    scheduleToClose: :unset,
    scheduleToStart: :unset,
    startToClose: :unset
  })

  # A synchronous reply to the retried attempt settles the operation as succeeded on its second
  # attempt: the count the retryable failure raised is still one, and the completed event records
  # the reply.
  defproperty :retrySucceeds,
    machine: :nexusProtocol,
    when: handlerReply(:syncSuccess),
    holds: fn step -> step.state == @succeededOnRetry and :nexusOperationCompleted in step.facts end

  # The schedule-to-start deadline settles an operation no handler started as timed out, and the
  # timed-out event records which deadline it was.
  defproperty :scheduleToStartFires,
    machine: :nexusProtocol,
    when: scheduleToStart,
    holds: fn step ->
      step.state.phase == :timedOut and {:nexusOperationTimedOut, :scheduleToStart} in step.facts
    end

  # The start-to-close deadline settles a started operation no handler completed as timed out.
  defproperty :startToCloseFires,
    machine: :nexusProtocol,
    when: startToClose,
    holds: fn step ->
      step.state.phase == :timedOut and {:nexusOperationTimedOut, :startToClose} in step.facts
    end

  # authoring: scenarios

  ## The paths the Queries run
  #
  # A protocol Scenario names its classed actions with their inputs and its start by its phase.
  # Each path below is one upstream functional test's shape: the schedule command with no deadline
  # set, then the side effects that settle the operation.

  defscenario :syncReplied,
    model: :nexusProtocol,
    starts: :unscheduled,
    actions: [schedule(:unset, :unset, :unset), handlerReply(:syncSuccess)]

  defscenario :asyncThenSucceeded,
    model: :nexusProtocol,
    starts: :unscheduled,
    actions: [schedule(:unset, :unset, :unset), handlerReply(:async), complete(:succeeded)]

  defscenario :asyncThenFailed,
    model: :nexusProtocol,
    starts: :unscheduled,
    actions: [schedule(:unset, :unset, :unset), handlerReply(:async), complete(:failed)]

  defscenario :nonRetryableError,
    model: :nexusProtocol,
    starts: :unscheduled,
    actions: [schedule(:unset, :unset, :unset), handlerReply({:handlerError, false})]

  # The retryable error backs the operation off; the backoff timer fires and records nothing; the
  # retried attempt is answered synchronously.
  defscenario :retriedThenSucceeded,
    model: :nexusProtocol,
    starts: :unscheduled,
    actions: [
      schedule(:unset, :unset, :unset),
      handlerReply({:handlerError, true}),
      backoff,
      handlerReply(:syncSuccess)
    ]

  # The schedule command sets the schedule-to-start deadline; the handler's worker stops, so
  # nothing answers the start request; the deadline fires. The worker stops after the schedule in
  # the operation's order, where the stop changes nothing; the realization stops it before the
  # workflow starts, where the stop cannot race the dispatch.
  defscenario :scheduleToStartExpires,
    model: :nexusProtocol,
    starts: :unscheduled,
    actions: [schedule(:unset, :expires, :unset), workerStop, scheduleToStart]

  # The schedule command sets the start-to-close deadline; the handler accepts asynchronously and
  # never completes; the deadline fires.
  defscenario :startToCloseExpires,
    model: :nexusProtocol,
    starts: :unscheduled,
    actions: [schedule(:unset, :unset, :expires), handlerReply(:async), startToClose]

  # Nine actions are enabled before the operation is scheduled and eleven once it is, so an exact
  # sequence of two is one among ninety-nine candidates, one of three among about a thousand and
  # one of four among about ten thousand. The search walks only the scenario's classes, so these
  # budgets are generous.
  deflimits :two, steps: 2, actions: 2, search: 512
  deflimits :three, steps: 3, actions: 3, search: 4096
  deflimits :four, steps: 4, actions: 4, search: 32_768

  # authoring: queries

  ## The Queries
  #
  # The design's seven: sync success, async reply then succeeded callback, async reply then failed
  # callback, non-retryable handler error, retryable handler error then sync success after one
  # backoff, schedule-to-start timeout with the handler's worker stopped, start-to-close timeout
  # after an asynchronous reply. Each finds its same-step claim on its path and is realized by the
  # set below. The product claim is verified over every trace of one path, outside the set,
  # because a verify Query realizes nothing.

  defquery :syncCompletion, find: :syncSucceeds, in: :syncReplied, limits: :two
  defquery :asyncCompletion, find: :completionSucceeds, in: :asyncThenSucceeded, limits: :three
  defquery :asyncFailure, find: :completionFails, in: :asyncThenFailed, limits: :three
  defquery :handlerError, find: :handlerErrorFails, in: :nonRetryableError, limits: :two
  defquery :retry, find: :retrySucceeds, in: :retriedThenSucceeded, limits: :four

  defquery :scheduleToStartTimeout,
    find: :scheduleToStartFires,
    in: :scheduleToStartExpires,
    limits: :three

  defquery :startToCloseTimeout,
    find: :startToCloseFires,
    in: :startToCloseExpires,
    limits: :three

  defquery :terminalHolds, verify: :terminalIsFinal, in: :asyncThenSucceeded, limits: :three

  # authoring: set

  ## The functional set
  #
  # Every party but `:system` is bound: the Case drives the caller, the handler and the worker, and
  # observes the network. The set repeats over the implementation switch, so each Query's Case
  # runs once under HSM and once under CHASM.

  defset :nexusCallerTests,
    purpose: :functional,
    bind: [caller: :driven, handler: :driven, network: :observed, worker: :driven],
    repeat: :implementation,
    queries: [
      :syncCompletion,
      :asyncCompletion,
      :asyncFailure,
      :handlerError,
      :retry,
      :scheduleToStartTimeout,
      :startToCloseTimeout
    ]

  ## The canary set
  #
  # A canary runs a Query against a deployment that performs the handler's part itself: the
  # handler is `:observed`, so the verifier reads which reply occurred and checks the machine
  # allows it. What admits a canary is that a deployment can close every gap its Case carries, and
  # every step of the sync and async completion paths records evidence; a path with a silent step
  # (the backoff, the worker stop) is a capability gap no deployment closes, so a canary naming it
  # is rejected.

  defset :nexusCallerCanary,
    purpose: :canary,
    bind: [caller: :driven, handler: :observed, network: :observed, worker: :driven],
    queries: [:syncCompletion, :asyncCompletion]

  ## The exploratory set
  #
  # An exploration covers the protocol machine rather than listing Queries. Its targets are the
  # rows an exploration within the budget's steps of a start can take, the results those rows
  # reach and the members of the classes their actions claim, each in the machine's catalog order
  # and cut at the budget's search count, so the enumeration is the same on every reading.

  defset :nexusCallerExploration,
    purpose: :exploratory,
    bind: [caller: :driven, handler: :driven, network: :observed, worker: :driven],
    machine: :nexusProtocol,
    cover: [:rows, :results, :classMembers],
    budget: :four

  # authoring: case
  #
  # The Cases (Case lowering and the realization) are out of scope for this sample; the Lean file
  # declares them here.

  # authoring: composition

  ## The operation and the handler's worker
  #
  # The protocol machine's worker stop is a stutter row: the operation cannot see its handler's
  # worker, so the schedule-to-start Scenario orders the stop before the request by convention.
  # Composed with the worker of the handler's task queue, the stop is the worker's own phase change
  # and every reply is the worker serving, so a reply has a row only while the worker polls. No set
  # names the composition; it is what the cross-entity claim is verified over.

  # The caller's view of the handler's worker: it stops and it serves. It never resumes, because
  # an action no `sync:` line names would stay executable on its own and admit a stop, a resume and
  # then a reply; the operation's timers settle every state a stop leaves.
  defmachine :handlerWorker, from: {Worker, :polling}, only: [:workerStop, :serve]

  defcompose :nexusCaller,
    for: [:operation, {Worker, :worker}],
    state: NexusCallerState,
    members: [operation: :nexusProtocol, worker: :handlerWorker],
    sync: [
      workerStop: operation.workerStop || worker.workerStop,
      handlerReply: operation.handlerReply || worker.serve
    ],
    starts: [operation: :unscheduled, worker: :polling],
    ends: [operation: [:succeeded, :failed, :canceled, :timedOut]]

  # Every reply, of any class, leaves the handler's worker polling: no handler replies while its
  # worker is stopped.
  defproperty :repliedByPollingWorker,
    machine: :nexusCaller,
    when: handlerReply,
    holds: fn step -> step.state.worker.phase == :polling end

  # A retryable reply backs the operation off; the handler's worker then stops, so the retried
  # attempt is never answered and the schedule-to-start deadline fires.
  defscenario :repliedThenStopped,
    model: :nexusCaller,
    starts: [operation: :unscheduled],
    actions: [
      operation.schedule(:unset, :expires, :unset),
      handlerReply({:handlerError, true}),
      workerStop,
      operation.scheduleToStart
    ]

  defquery :stoppedWorkerRepliesNothing,
    verify: :repliedByPollingWorker,
    in: :repliedThenStopped,
    limits: :four

  # authoring: end
end
