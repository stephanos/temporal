# Walkthrough: the standalone activity Model in Elixir

## 1. What this is

Umpire is model-based testing with three parts. A Model is the rulebook: a finite description of
what a Temporal feature may do. A playthrough is one path through the rulebook, chosen by a
search. A referee runs that path against a real server and checks that what the server recorded is
what the rulebook allowed.

A standalone activity is started directly with `StartActivityExecution`, with no workflow. It
writes no history events, so it is observed only through `DescribeActivityExecution` (status) and
`PollActivityExecution` (result).

`standalone_activity.ex` is that rulebook: vocabulary, two machines, claims, paths and test sets.
`lib/umpire/` checks it as it compiles; `test/pins_test.exs` holds 21 pins, passing on Elixir 1.20.4.

## 2. The language in five minutes

Elixir is a dynamically typed functional language on the Erlang VM (the BEAM). Recent versions
also infer types and report type errors as compile-time warnings. The walkthrough relies on these
features.

**Atoms, tuples and patterns.** An atom is a constant named by itself, such as `:completed`; the
sample uses one for every spec name. `{:failed, true}` is a 2-tuple. A pattern is a value with
holes: `{phase, _}` matches any 2-tuple, binds `phase`, and ignores the rest.

**`case` with guards.** `case` tries each `pattern -> result` arm in order. `when` adds a
condition, and `phase in [...]` tests membership in a literal list. An arm matching none of the
values is not an error when the file compiles, only a `CaseClauseError` when it runs. That gap is
the one the framework closes.

`standalone_activity.ex:458-460`
```elixir
      case {state.phase, state.startToClose} do
        {phase, :expires} when phase in [:started, :pauseRequested, :cancelRequested] ->
          moves(:timedOut, [{:statusTimedOut, :startToClose}])
```

**Keyword lists, structs, attributes.** `[party: :caller, on: :activity]` is a list of key-value
pairs, and its brackets can be dropped in a last argument, which is why declarations read like
configuration. A struct is a map with fixed keys tied to a module: `%ProductState{phase:
:started}`. `@name value` sets a compile-time module attribute.

`standalone_activity.ex:148`
```elixir
  @productTerminal [:completed, :failed, :canceled, :terminated, :timedOut]
```

**Macros.** `fn step -> ... end` is an anonymous function; the rest of the syntax is macros. A macro runs at compile time, receives its arguments as code (the AST, as data), and returns new
code. `quote` turns code into that data, and
`unquote` splices a value into it. Every `def...` declaration in the Model is an Umpire macro, in
the same style as `Ecto.Schema`'s `field` or Phoenix's `get`. The one detail that matters later is
timing. Elixir expands a whole module body first and evaluates it second, so a macro that needs a
value set earlier in the body must defer that work to the evaluation.

## 3. Vocabulary: entities, parties, actions, inputs

An **entity** is what a machine is about. The activity is named by the id its caller chose.

`standalone_activity.ex:37`
```elixir
  entity :activity, key: :activityId
```

A **domain** is a finite input type. A plain atom is one member. A constructor with finite fields
contributes one member per assignment of those fields. `AttemptResult` therefore has four members:
`:completed`, `{:failed, false}`, `{:failed, true}` and `:canceled`. `Control` has four plain ones.

`standalone_activity.ex:48-52`
```elixir
  domain AttemptResult, [:completed, {:failed, retryable: :boolean}, :canceled]

  domain Delivery, [:accepted, :notFound]

  domain Control, [:pause, :unpause, :requestCancel, :terminate]
```

Each `domain` line becomes a small module (`StandaloneActivity.AttemptResult`) with `values/0`,
so reuse is an `alias`: `Timeout` is the Nexus Model's.

A **party** is who performs an action: here `:caller` and `:worker`. The reserved party `:system`
owns the timers. An **action** is a named side effect of a party, with typed inputs. Its schema is
the protobuf message a realization would send, and its examples say which concrete value stands
for an input class.

`standalone_activity.ex:74-84`
```elixir
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
```

`start` takes three `Timeout` inputs, so it has 2 × 2 × 2 = 8 **action classes**. A class is one
action with one assignment of its inputs. `attemptResult` has four classes, `control` four, and
`attemptStart` and `workerStop` one each, since they take no input.

A **fault** is not a separate kind: the worker stopping is an ordinary `:worker` action, declared
once in the worker's Model and shared with `import_actions Worker, [:workerStop]`. An
**observation** is evidence no status change records. Each poll raises the attempt count, and
Describe exposes it:

`standalone_activity.ex:107`
```elixir
  observation :attemptCount, on: :activity, read: :attempt
```

## 4. State

The product machine's state is a struct with one field, `phase`, drawn from a nine-member domain.
The protocol machine's state has five fields:

`standalone_activity.ex:285-292`
```elixir
  @attemptBound 2

  defstate ProtocolState,
    phase: Phase,
    attempts: 0..@attemptBound,
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout
```

Every field has a finite type: a domain module, `:boolean`, or a range. `0..@attemptBound` is the
range 0 to 2. Anything else is rejected when the Model compiles, so every state space is finite by
construction.

`defstate` turns this into the struct module `ProtocolState`. Its `values/0` is the cartesian
product of the field types, with the first field varying slowest: 12 × 3 × 2 × 2 × 2 = 288 states.
The product has 9. The enumeration is ordinary Elixir in the framework:

`lib/umpire/domain.ex:94-99`
```elixir
  # Built as maps tagged with `__struct__` rather than with `struct!/2`: the states are
  # enumerated before their module exists, so that they can be compiled into it as data.
  def values(%IR.StateType{module: module, fields: fields}) do
    for assignment <- product(Enum.map(fields, fn {_field, type} -> values(type) end)),
        do: Map.new([{:__struct__, module} | Enum.zip(Keyword.keys(fields), assignment)])
  end
```

The states are built as maps tagged with `__struct__` because of the timing rule from section 2.
The module does not exist yet when its states are listed, so the list is compiled into it as a
literal.

The attempt count is bounded. `succ(state.attempts)` is the saturating successor: at 2 it stays 2.
The framework reads the bound off the field's range while lowering the step, and rejects `succ` on
a field that is not a range.

## 5. Step functions

A step function takes a state and an action's inputs and returns a list of steps, each an outcome,
a next state and facts; `[]` means not enabled. `moves(phase, facts)` moves and records, `stay()`
keeps the state, and `not_found()` answers `:notFound`.

This is the product's `attemptResult`, read arm by arm:

`standalone_activity.ex:181-204`
```elixir
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
```

- **Lines 183 to 187.** A completion or non-retryable failure settles the activity.
- **Line 189.** A retryable failure is visible: Describe reads SCHEDULED again.
- **Lines 192 to 196.** Under a requested cancel, both a retry and a cancel response end canceled.
- **Lines 198 and 201.** A cancel with no request, or any other phase, is not enabled.

The protocol's `attemptResult` dispatches on the phase as well as the result. Three phases hold
an attempt, and each answers a retryable failure differently, mirroring CHASM's
`TransitionRescheduled` and `TransitionAttemptFailedWhilePauseRequested`:

`standalone_activity.ex:373-392`
```elixir
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
```

- **From `started`**, the activity backs off. The row records `:statusScheduled` as well as
  `:attemptCount`, because Describe reads SCHEDULED again. Section 7 explains why that fact matters.
- **From `cancelRequested`**, it is canceled.
- **From `pauseRequested`**, it is parked in `paused`.

**How exhaustiveness is enforced.** The macro does it, not the language. `defstep` lowers the
`case` into IR, where each pattern, guard and body is a tagged tuple. It also emits the real
function `ActivityProtocol.attempt_result/2`, with the author's patterns and guards and each body
replaced by the struct list it stands for. Both happen through unquote fragments
(`def unquote(step.fun)(...)`), as in `Phoenix.Router`.

When the machine module finishes, its hook builds the table: this step on all 288 states and all
four results, recording which clause answered each. A combination with no clause is a
`CompileError` listing the uncovered `{phase, result}` values, and so is a clause that never
answers first. The product's `control` step relies on the second rule: its comment notes that a
`{_, :requestCancel} -> []` fallback would be reported as dead.

The Elixir 1.20 type checker also reads the emitted function, whose head pins the struct and the
four input literals. It can warn about a missing field or an unreachable arm. It does not report a
missing arm.

## 6. The machine and its table

A machine declaration names its entity, its state type, its outcomes and facts, and then lists its
header lines and steps:

`standalone_activity.ex:310-325`
```elixir
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
```

- **`starts`** gives the phase the machine begins in. The start state is the first state with
  that phase, so the other fields take their first values: attempts 0, every deadline `:unset`.
- **`ends`** gives the phases it may finish in.
- **`timers`** are the `:system` actions the machine owns. **`unobservable`** marks the backoff
  timer as recording nothing.
- **`evidence`**, on the lines after these, maps each fact to the name of the recorded thing
  that proves it.
- **`refines`** is the subject of section 7.

With no history events, every evidence name is a Describe status or the `attemptCount`
observation. The table is built during `mix compile`, in the machine module's `@before_compile`
hook, which runs just before a module's code is finalized:

`lib/umpire/table.ex:52-64`
```elixir
    {rows, hits, misses} =
      for state <- states, {action, inputs} = class <- classes, reduce: {[], %{}, %{}} do
        {rows, hits, misses} ->
          step = Map.fetch!(machine.steps, action)

          case Eval.step(step, state, inputs, machine) do
            {:fell_through, subject} ->
              {rows, hits, Map.update(misses, action, MapSet.new([subject]), &MapSet.put(&1, subject))}

            {:ok, index, steps} ->
              Enum.each(steps, &in_domain!(&1, step, index, domains))
              new = Enum.map(steps, &%Row{from: state, class: class, outcome: &1.outcome, to: &1.state, facts: &1.facts, clause: index})
              {Enum.reverse(new, rows), Map.update(hits, action, MapSet.new([index]), &MapSet.put(&1, index)), misses}
```

Each row is `(from, class, outcome, to, facts, clause)`. The same pass checks every produced value
against its domain, and the table is compiled into the module as `ActivityProtocol.table()`.

## 7. Two levels and the refinement

The product machine is what a caller can observe. The protocol machine is how the server gets
there: backoff, a pause that is only requested, three deadlines, an attempt count. Writing both
keeps the claims short. `terminalIsFinal` is stated once, on nine product states, and carried to
288 protocol states.

The link is the map `productOf`. Backing off and not yet started read as `scheduled`, and a
requested pause reads as `started`: the worker still holds the attempt, so every answer it can
give is a product row from `started`, and the request itself is a stutter.

`standalone_activity.ex:473-479`
```elixir
    defmap productOf(state) do
      case state.phase do
        phase when phase in [:unstarted, :scheduled, :backingOff] -> %ProductState{phase: :scheduled}
        :pauseRequested -> %ProductState{phase: :started}
        phase -> %ProductState{phase: phase}
      end
    end
```

The rule has two parts. Every protocol row `(s, class, s')` must be a **stutter**, meaning
`productOf(s) == productOf(s')`, or match some **product row** from `productOf(s)` to
`productOf(s')`. Rows are matched by mapped states, not by action name. The framework implements
the stricter rule the real Lean checker uses. A matching product row must also have the same
outcome, and its facts must appear among the protocol row's facts, compared by evidence name:

`lib/umpire/refinement.ex:71-80`
```elixir
    verdicts =
      Enum.map(table.rows, fn row ->
        {from, to} = {mapped[row.from], mapped[row.to]}

        cond do
          from == to -> {row, :stutter}
          match = Enum.find(Map.get(between, {from, to}, []), &matches?(&1, row, rule, evidence)) -> {row, {:matches, match}}
          true -> {row, {:rejected, from, to}}
        end
      end)
```

`lib/umpire/refinement.ex:88-91`
```elixir
  defp matches?(product_row, row, :strict, {evidence, product_evidence}) do
    product_row.outcome == row.outcome and
      MapSet.subset?(names(product_row.facts, product_evidence), names(row.facts, evidence))
  end
```

The check runs in the protocol machine's hook, right after its table, so a broken refinement
fails `mix compile` and names the first bad row.

One row by hand. The protocol row goes from `started`, attempts 1, under `attemptResult({:failed,
true})`, to `backingOff` with facts `[:statusScheduled, :attemptCount]`. `productOf` maps it from
`started` to `scheduled`, which is not a stutter. The product has one row between those states,
its own `attemptResult({:failed, true})`, with outcome `:accepted` and facts `[:statusScheduled]`.
The outcomes agree, and `statusScheduled` is among the protocol row's evidence names, so the row
matches. Without `:statusScheduled` on the protocol row, the strict rule would reject it. That is
the second revision note in SPEC.md, and a pin in section 13 checks it.

A match need not be the same action. The protocol's retryable failure from `pauseRequested` goes
to `paused`. It maps from `started` to `paused`, and it matches the product's `control(:pause)`
row.

## 8. Properties

A **same-step claim** names an action under `when:` and holds of the step that action produces:

`standalone_activity.ex:500-503`
```elixir
  defproperty :completes,
    machine: :activityProtocol,
    when: attemptResult(:completed),
    holds: fn step -> step.state.phase == :completed and :statusCompleted in step.facts end
```

`retryCompletes` fixes a whole state, so every field is named:

`standalone_activity.ex:525-528`
```elixir
  defproperty :retryCompletes,
    machine: :activityProtocol,
    when: attemptResult(:completed),
    holds: fn step -> step.state == @completedOnRetry and :statusCompleted in step.facts end
```

`@completedOnRetry` names every field (completed, attempts 2, no deadline). It is built with
`struct!/2`, not a `%ProtocolState{}` literal, because of the timing rule: a literal at the top of
the module would be expanded before `ProtocolState` exists.

A **transition claim** has no `when:` and takes the step before and the step after. `after` is a
reserved word in Elixir, so the second argument is `next`:

`standalone_activity.ex:493-497`
```elixir
  defproperty :terminalIsFinal,
    machine: :activityProduct,
    holds: fn before, next ->
      before.state.phase not in @productTerminal or next.state.phase == before.state.phase
    end
```

`standalone_activity.ex:552-554`
```elixir
  defproperty :pausedIsNotDispatched,
    machine: :activityProduct,
    holds: fn before, next -> before.state.phase != :paused or next.state.phase != :started end
```

Both of these are declared on the product machine and checked on protocol paths through
`productOf`. Each `holds:` function is lowered to an IR expression for the search. It is also
emitted as an ordinary function (`StandaloneActivity.terminal_is_final/2`) so the type checker sees
it. The lowering accepts `==`, `!=`, `in`, `not in`, `and`, `or`, `not`, field paths and literals.
A call such as `Enum.member?` is rejected with its line.

## 9. Scenarios and limits

A **scenario** is a path to try: a start phase and a list of action classes. The list is read by
the macro and never evaluated. `start(:unset, :unset, :unset)` names one class of `start`,
`attemptStart` bare names the action with no input, and `backoff` names the timer.

`standalone_activity.ex:591-603`
```elixir
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
```

The path is a retry: the attempt fails retryably, the backoff timer returns it to `scheduled`, and
the second poll starts an attempt that completes. `pausedThenCompleted` has the same shape: start,
`control(:pause)`, `control(:unpause)`, `attemptStart`, `attemptResult(:completed)`. When the Model compiles, each class is
checked against the machine's classes. A misspelled class is an error there, with a suggestion.

**Limits** bound a search: the longest trace, the longest scenario admitted, and how many steps
the search may take before it gives up.

`standalone_activity.ex:647-649`
```elixir
  deflimits :three, steps: 3, actions: 3, search: 4096
  deflimits :four, steps: 4, actions: 4, search: 32_768
  deflimits :six, steps: 6, actions: 6, search: 262_144
```

## 10. Queries

A **find** query asks the search for a trace of the path on which the claim fires and holds. A
**verify** query checks the claim everywhere it fires, on every trace of the path.

`standalone_activity.ex:662`
```elixir
  defquery :retry, find: :retryCompletes, in: :retriedThenCompleted, limits: :six
```

`standalone_activity.ex:678`
```elixir
  defquery :pauseHolds, verify: :pausedIsNotDispatched, in: :pausedThenCompleted, limits: :six
```

`retry` finds `retryCompletes` at step 6 of `retriedThenCompleted`. `pauseHolds` verifies the
product's `pausedIsNotDispatched` on every transition of `pausedThenCompleted`, reading each state
through `productOf`.

Admission is checked when the Model compiles. A product claim may be read on a protocol path only
because `activityProtocol` refines `activityProduct`, and the path must fit the limits. The search
runs in `mix test`. It follows the path's classes through the table, branching where a row is
nondeterministic, and counts steps against the budget:

`lib/umpire/search.ex:80-89`
```elixir
      |> Enum.reduce(acc, fn row, acc ->
        acc = %{acc | nodes: acc.nodes + 1}
        if acc.nodes > ctx.limits.search, do: throw({:stop, %{acc | outcome: :exhausted}})

        step = %Step{outcome: row.outcome, state: row.to, facts: row.facts}
        trace = [{class, step} | trace]
        acc = claim(ctx, class, last, step, trace, acc)
        walk(ctx, step, trace, rest, acc)
      end)
    end
```

A verify whose claim never fired proved nothing. Here that is an error, `:vacuous`, because
`require_firing:` defaults to true. The Lean framework defaults it to false.

`lib/umpire/search.ex:67-70`
```elixir
  defp finish(%Result{outcome: outcome} = result, _query) when outcome != nil, do: result
  defp finish(result, %IR.Query{kind: :find}), do: %{result | outcome: :not_found}
  defp finish(%Result{fired: 0} = result, %IR.Query{kind: :verify, require_firing: true}), do: %{result | outcome: :vacuous}
  defp finish(result, %IR.Query{kind: :verify}), do: %{result | outcome: :verified_within_limits}
```

A failed query is a failed test: `assert_query/3` prints the counts and the trace where the claim
fired and failed. The README shows one.

## 11. Sets

A **set** is what a test run realizes. A **functional** set drives every party and runs the find
queries. A **canary** runs against a live deployment and **observes** the worker instead: the
deployment's own worker answers, and the verifier checks the machine allows what it did. An
**exploratory** set names a machine and coverage targets instead of queries.

`standalone_activity.ex:707-710`
```elixir
  defset :standaloneActivityCanary,
    purpose: :canary,
    bind: [caller: :driven, worker: :observed],
    queries: [:completion, :cancel]
```

The functional set binds both parties as driven and lists the eight find queries. The exploratory
set names `machine: :activityProtocol`, `cover: [:rows, :results, :classMembers]` and
`budget: :four`.

There is no `repeat: :implementation`, because standalone activities exist only in CHASM. The
compile-time hook checks that functional and canary sets hold only find queries and bind every
party their paths use. The canary rule that every step records evidence is only described, not
implemented, and `Umpire.Search.explore/2` is a sketch.

## 12. Composition with the worker

On its own, the protocol's `workerStop` is a stutter: the activity cannot see its worker. The
worker is an entity of its own, with a two-phase machine:

`worker.ex:62-68`
```elixir
    # A polling worker stops; a stopped one has nothing to stop.
    defstep workerStop(state) do
      case state.phase do
        :polling -> moves(:stopped, [])
        :stopped -> []
      end
    end
```

`workerResume` is the mirror image, and `serve` is `stay()` while polling and `[]` when stopped.

`activityWorker` restricts that machine to stopping and serving; it never resumes, because an
unsynchronized action stays executable on its own. Each `sync:` line makes two member actions fire
as one class, so an attempt starts only if the worker serves:

`standalone_activity.ex:736-747`
```elixir
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
```

`standalone_activity.ex:759-769`
```elixir
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
```

The query `stoppedWorkerStartsNothing` verifies the claim on this path within `:six`.

The claim `startedByPollingWorker` says that at every `attemptStart`, `step.state.worker.phase ==
:polling`; the composite state has one field per member. The composite table is 288 × 2 states,
built from the member tables when the Model compiles.

On this path one attempt starts while the worker polls, fails retryably, and the worker stops
before the retry, so the deadline fires. That single `attemptStart` makes the claim fire once, so
the verify is exercised. The path it replaced, `stoppedBeforeDispatch`, never started an attempt
and passed vacuously.

## 13. Pins

The pins freeze what the Model says. Most checks behind them would already have failed `mix
compile`; the pins record the answers so a change shows up in review. One pin is decided when
ExUnit compiles the test file, before any test runs:

`test/pins_test.exs:21-23`
```elixir
  # Decided at this file's compile time: twelve phases, three attempt counts and three deadlines.
  static_assert Activity.ProtocolState.size() == 12 * 3 * 2 * 2 * 2
  static_assert Nexus.ProtocolState.size() == 8 * 3 * 2 * 2 * 2
```

A row the spec calls out, checked on the compiled function:

`test/pins_test.exs:127-130`
```elixir
    # A cancel response with no cancel requested is not enabled.
    test "attemptResult(:canceled) from started is not enabled" do
      assert ActivityProtocol.attempt_result(at(ProtocolState, :started), :canceled) == []
    end
```

The second revision note, as a test. The test copies the protocol table, strips `:statusScheduled`
from every retry row out of `started`, and reruns the refinement under both rules:

`test/pins_test.exs:180-183`
```elixir
      assert %Refinement.Result{rejected: {%Table.Row{class: {:attemptResult, [{:failed, true}]}}, _, _}} =
               Refinement.check(stripped, product, mapped, :strict, evidence)

      assert %Refinement.Result{rejected: nil} = Refinement.check(stripped, product, mapped, :mapped_states, evidence)
```

The third revision note. The composition verify fires once. A second test rebuilds the old path as
IR and gets `:vacuous`:

`test/pins_test.exs:200-202`
```elixir
    test "stoppedWorkerStartsNothing verifies and its claim fires" do
      assert %Search.Result{fired: 1, paths: 1} = assert_query(Activity, :stoppedWorkerStartsNothing, :verified_within_limits)
    end
```

Each machine also has an agreement pin, `assert_agrees/2`, which runs the compiled step functions
and the IR interpreter on every row and requires the same steps.

## 14. From model to running test

This layer is not in the sample. In the Lean original, each query of a set is lowered to a
protobuf Case. A realization binds each class to real calls, such as a `StartActivityExecution`
request or a poll answered with a retryable `ApplicationFailure`. The Go Testpilot runtime drives a
server with it, and a verdict compares the recorded evidence with the witness. No realization
exists for standalone activities yet. Here the hand-off point would be `mix umpire.ir`, which
writes canonical JSON of the whole Model, tables included, for a Go core to lower.

## 15. Gaps and gradual growth

The Model says "not modeled" in several places, on purpose:

- **Reset** is deferred, as cancellation is in the Nexus Model.
- **The heartbeat timeout** is absent.
- **Stutter rows** such as the protocol's `workerStop` keep the state and record nothing.
- **Empty steps** such as the product's `workerStop` mean the product cannot see the action at all.

Growing the Model means adding a field, phase or action and letting the compile-time checks point
at every step that no longer covers its combinations. A heartbeat timer would be one new protocol
timer step, and the refinement would insist that it maps to the product's `timeout` row or to a
stutter. The new row reaches the IR, the JSON and the compiled function at once.

## 16. Mental model recap

- **The IR is the source of truth.** Each declaration is a macro that produces IR data.
- **Each machine is a nested module** whose step functions come from the same source as the IR.
- **Every type is finite**, so every table can be listed in full.
- **Exhaustiveness is enumeration**: every combination needs a clause, every clause needs a use.
- **The refinement runs at compile time**, with the strict rule on outcomes and facts.
- **Claims are data too**, lowered to expressions and read across the refinement.
- **Queries run in `mix test`**, and a verify that never fires fails.
- **Pins freeze the answers**, and an agreement pin keeps compiled code and IR in step.

## 17. Where this implementation is weak

The review (`cmp/.eval/elixir.md`, scores 5, 4, 4, 4, 5, 5) found no spec drift and no invented
APIs. Its red flags, and what became of them once the sample compiled:

- **The subset claimed to reject rebinding and did not.** `{x, x}` would have been an equality
  test on the BEAM and an overwrite in `Umpire.Eval`. This is now fixed: a name bound twice, or
  shadowing the state or an input, is a `CompileError`. The finding still stands as a warning about
  the design. Two implementations of pattern matching disagree in exactly such corners, and only
  `assert_agrees/2` guards the rest.
- **One README error text was not the checker's first report.** This is now fixed: the refinement
  example is the real `mix compile` output. The review also noted that search depth ignores
  `limits.actions`, which is enforced only at admission. A query built directly as IR, as the
  vacuity pin does, bypasses that check.
- **Less is checked than the headline suggests.** `defmap` gets totality and codomain checks but
  no dead-arm check; canary admission is only in comments; `explore/2` is a sketch. Compiling also
  showed `mix umpire.ir` crashing on value-keyed maps, fixed but not pinned by a golden file.
- **The review's main reservation.** The guarantee rests on a large hand-written macro framework:
  about 580 lines of lowering, an interpreter and an emitter, for a Go team to maintain. A Model
  edit also costs about five seconds of compile time, likely from tables compiled in as literals.
