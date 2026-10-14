# Walkthrough: the standalone activity Model in Julia

## What this is

Umpire is model-based testing with three parts. A Model is the rulebook: it says which moves exist
and which moves are legal from each position. A Case is one playthrough: a sequence of moves the
test runtime performs against a real Temporal server. The referee compares what the server recorded
with what the rulebook allows. A standalone activity is a Temporal activity started directly with
`StartActivityExecution`, with no workflow around it. It writes no history events, so the referee
can only read its status through `DescribeActivityExecution` and its result through
`PollActivityExecution`. The file `standalone_activity.jl` is the rulebook for one such activity,
written twice at two levels of detail, plus the claims, paths, queries and test sets built on it.

## The language in five minutes

Julia is a dynamically typed language with optional type annotations, a strong macro system, and
multiple dispatch (a function can have many methods, chosen by the types of all arguments). The
sample leans on these features.

**Modules and qualified names.** Code lives in `module ... end`. Nothing is exported unless asked,
so most names in the Model are written qualified, like `Phase.started`.

```julia
module StandaloneActivity

using Umpire
```
(`standalone_activity.jl:13-15`)

**Namespaced enums** come from the EnumX package. `@enumx Control pause unpause ...` makes a module
`Control` with a type `Control.T` and members `Control.pause` and so on.

```julia
@enumx Control pause unpause requestCancel terminate
```
(`standalone_activity.jl:42`)

**Sum types with payloads** come from the Moshi package. `@data` declares a type `AttemptResult.Type`
whose variants may carry fields. A sum type is a value that is exactly one of several cases.

```julia
@data AttemptResult begin
    completed
    failed(retryable::Bool)
    canceled
end
```
(`standalone_activity.jl:36-40`)

**Pattern matching** is Moshi's `@match`. Each arm is `pattern => result`, or-patterns use `||`,
and `_` matches anything. The first example is in the step functions section.

**Keyword structs.** `Base.@kwdef struct` gives a struct a constructor that takes fields by name;
the state types in the State section are written with it.

**Early return.** `cond || return x` reads "unless cond, return x", and opens most step functions
(`standalone_activity.jl:152`). Anonymous functions are written `arg -> body` or
`(a, b) -> body`. `T[]` is an empty vector of element type `T`.

**Macros with a block.** Every Umpire command is a macro applied to a name and a `begin ... end`
block of `key = value` lines. `var"for"` is how a Julia keyword is used as a plain name.

```julia
@entity activity begin
    key = activityId
end
```
(`standalone_activity.jl:25-27`)

## Vocabulary: entities, parties, actions, inputs

An **entity** is the thing a machine is about. Here it is the activity, identified by the id the
caller chose. The `@entity` block above registers it in a framework table so later commands can
check that they name a declared entity.

A **party** is who acts: `caller`, `worker`, and the reserved `system` for timers. Parties are not
declared; an action names one.

An **action** is a named side effect of a party. It may create an entity or act on one, may take
finite inputs, and may carry a protobuf schema name and example payloads for the realization layer.

```julia
@action start begin
    party = caller
    creates = activity
    schema = "temporal.api.workflowservice.v1.StartActivityExecutionRequest"
    input = (
        scheduleToClose = Timeout.T,
        scheduleToStart = Timeout.T,
        startToClose = Timeout.T,
    )
end
```
(`standalone_activity.jl:51-60`)

`attemptResult` (`standalone_activity.jl:69-78`) is the worker's, acts `on = activity`, takes
`input = (result = AttemptResult.Type,)`, and carries an `examples` table mapping `failed(false)`
and `failed(true)` to the two `ApplicationFailure` payloads a realization would send.

The five actions are `start` (caller), `attemptStart` (the worker's poll receives the task),
`attemptResult` (the worker reports), `control` (caller: pause, unpause, requestCancel, terminate,
with a `Delivery` result of accepted or notFound), and `workerStop`. The last is declared in
`worker.jl`, names no entity, and is a fault: the worker process stops polling. A fault is not a
special kind of thing. It is an ordinary action of an ordinary party, and each machine says what it
does about it, which is usually nothing.

```julia
@action workerStop begin
    party = worker
end
```
(`worker.jl:47-49`)

An **action class** is one action with one concrete assignment of its inputs. `attemptResult` has
four classes: `completed`, `failed(false)`, `failed(true)`, `canceled`. `control` has four, one per
`Control` member. `start` has eight (two values for each of three timeouts). An action with no
input is one class. The framework's `classes` function (`Umpire.jl:182-186`) computes them as the
cartesian product of `finite(t)` over the action's input types, the same `finite` that enumerates
states below.

`failed(retryable::Bool)` is how Julia, through Moshi, spells a finite domain member with a payload.
The payload type must itself be finite, which is what makes the class count finite.

## State

Each machine has a state type. The product machine's state is one field:

```julia
Base.@kwdef struct ProductState
    phase::ProductPhase.T
end
```
(`standalone_activity.jl:123-125`)

The protocol machine's state adds a bounded attempt counter and the three deadlines the start
request set:

```julia
const attemptBound = 2

Base.@kwdef struct ProtocolState
    phase::Phase.T
    attempts::Fin{attemptBound}
    scheduleToClose::Timeout.T
    scheduleToStart::Timeout.T
    startToClose::Timeout.T
end
```
(`standalone_activity.jl:256-264`)

Every field is finite because the framework must list every state to build a table. `Fin{N}` is an
integer in `0:N`; Julia has no range types, so the bound is a type parameter:

```julia
"""The successor that stays inside the bound, so a retry cannot leave the finite state space."""
saturatingSucc(a::Fin{N}) where {N} = Fin{N}(min(a.n + 1, N))
```
(`Umpire.jl:53-54`)

The constructor rejects a value outside `0:N`, and `attemptStart` raises the count with
`saturatingSucc`, so a third retry stays at 2 instead of leaving the finite space.

Enumeration is one generic function, `finite`, with a method per representation. This is multiple
dispatch at work: the struct method never needs to know which enum package a field came from.

```julia
finite(::Type{T}) where {T<:Enum} = collect(instances(T))         # Base.@enum and EnumX both
finite(::Type{Bool}) = [false, true]
finite(::Type{Fin{N}}) where {N} = Fin{N}.(0:N)
```
(`Umpire.jl:65-67`)

```julia
    return vec([T(vals...) for vals in Iterators.product((finite(ft) for ft in fieldtypes(T))...)])
```
(`Umpire.jl:83`)

`instances` is Julia's built-in list of an enum's members. `fieldtypes` is reflection: the types of
a struct's fields. The struct method is the cartesian product of the fields' domains, so
`ProductState` has 9 states and `ProtocolState` has 12 phases × 3 attempt values × 2 × 2 × 2
timeouts = 288. Between those two quotes (`Umpire.jl:68-80`) sits the Moshi branch: one member per
variant, times every assignment of the variant's fields, which is what makes `failed(true)` and
`failed(false)` two members. The Moshi reflection names it uses were not verified against the package.

## Step functions

A step is what one action does from one state:

```julia
struct Step{S,O,F}
    outcome::O
    state::S
    facts::Vector{F}
    Step{S,O,F}(; outcome, state, facts = F[]) where {S,O,F} = new{S,O,F}(outcome, state, collect(F, facts))
end
```
(`Umpire.jl:132-137`)

A step function takes the state and the action's inputs and returns a vector of steps. An empty
vector means the action is not enabled there. Each Model aliases its step type (`PStep`, `TStep`)
and writes a one-line helper, `productStep` or `moves`, for the common single-step case
(`standalone_activity.jl:141-144`, `293-294`).

The product `attemptResultStep`, arm by arm:

```julia
function attemptResultStep(state::ProductState, result::AttemptResult.Type)::Vector{PStep}
    state.phase in (ProductPhase.started, ProductPhase.cancelRequested) || return PStep[]
    return @match result begin
        AttemptResult.completed     => productStep(ProductPhase.completed, ProductFact.statusCompleted)
        AttemptResult.failed(false) => productStep(ProductPhase.failed, ProductFact.statusFailed)
        AttemptResult.failed(true)  => state.phase == ProductPhase.started ?
            productStep(ProductPhase.scheduled, ProductFact.statusScheduled) :
            productStep(ProductPhase.canceled, ProductFact.statusCanceled)
        AttemptResult.canceled      => state.phase == ProductPhase.cancelRequested ?
            productStep(ProductPhase.canceled, ProductFact.statusCanceled) : PStep[]
    end
end
```
(`standalone_activity.jl:161-172`)

Line 162: a result only means something while an attempt is out, so from any phase other than
`started` or `cancelRequested` the action is not enabled. `completed` settles the activity and the
Describe status reads COMPLETED. `failed(false)` is a non-retryable failure. `failed(true)` from
`started` puts the activity back to SCHEDULED in Describe (the retry is visible, unlike in the Nexus
Model); from `cancelRequested` the retry is not taken and the activity is canceled. `canceled` is
honored only where a cancel was requested.

The protocol version dispatches on the phase as well as the result:

```julia
function protocolAttemptResultStep(state::ProtocolState, result::AttemptResult.Type)::Vector{TStep}
    p = state.phase
    p in (Phase.started, Phase.cancelRequested, Phase.pauseRequested) || return TStep[]
    return @match result begin
        AttemptResult.completed     => moves(state, Phase.completed, [ProtocolFact.statusCompleted])
        AttemptResult.failed(false) => moves(state, Phase.failed, [ProtocolFact.statusFailed])
        AttemptResult.failed(true)  => @match p begin
            Phase.started         => moves(state, Phase.backingOff, [ProtocolFact.attemptCount])
            Phase.cancelRequested => moves(state, Phase.canceled, [ProtocolFact.statusCanceled])
            Phase.pauseRequested  => moves(state, Phase.paused, [ProtocolFact.statusPaused])
        end
        AttemptResult.canceled => p == Phase.cancelRequested ?
            moves(state, Phase.canceled, [ProtocolFact.statusCanceled]) : TStep[]
    end
end
```
(`standalone_activity.jl:321-335`)

The three source phases react differently to a retryable failure because the server's own state
machine does. From `started` it backs off and retries (CHASM `TransitionRescheduled`). From
`cancelRequested` the cancel wins. From `pauseRequested` the failure lands in `paused`
(`TransitionAttemptFailedWhilePauseRequested`). `moves` is the protocol helper: copy the state with
a new phase, attach the facts. `with(state; phase)` is the framework's record update.

Exhaustiveness is not checked statically. Moshi's `@match` throws at run time when no arm matches.
What catches a missing arm is the table build, which calls every step function on every state and
every class, so every arm is exercised:

```julia
function build_table(::Type{S}, ::Type{O}, ::Type{F}, actions::Vector{Action},
                     steps::Dict{Symbol,<:Function}) where {S,O,F}
    states = finite(S)
    cls = reduce(vcat, classes.(actions); init = ActionClass[])
    rows = Dict{Tuple{S,ActionClass},Vector{Step{S,O,F}}}()
    sizehint!(rows, length(states) * length(cls))
    for s in states, c in cls
        rows[(s, c)] = steps[c.action](s, c.inputs...)::Vector{Step{S,O,F}}
    end
    return Table{S,O,F}(states, cls, rows)
end
```
(`Umpire.jl:209-219`)

The `::Vector{Step{S,O,F}}` on line 216 is a run-time type assertion: a step function returning the
wrong type fails here too.

## The machine and its table

The protocol machine declaration binds everything together:

```julia
@machine activityProtocol begin
    var"for" = activity
    state = ProtocolState
    outcome = ProtocolOutcome.T
    fact = ProtocolFact.Type
    refines = activityProduct
    map = productOf
    starts = [unstarted]
    ends = [completed, failed, canceled, terminated, timedOut]
    timers = [backoff, scheduleToClose, scheduleToStart, startToClose]
    unobservable = [backoff]
```
(`standalone_activity.jl:418-428`)

```julia
    steps = (
        start = startStep,
        attemptStart = protocolAttemptStartStep,
        attemptResult = protocolAttemptResultStep,
        control = protocolControlStep,
        workerStop = protocolWorkerStopStep,
        backoff = backoffStep,
        scheduleToClose = scheduleToCloseStep,
        scheduleToStart = scheduleToStartStep,
        startToClose = startToCloseStep,
    )
end
```
(`standalone_activity.jl:441-452`)

`starts` names phases; the framework picks the state with that phase and every other field at its
first value (`unstarted`, 0 attempts, all timeouts unset). `ends` names phases too, and every state
in those phases is an end, so there are 5 × 24 = 120 end states. `timers` are the `system` party's
actions this machine owns; they need no `@action` because they take no input. `unobservable` marks
timers that record nothing. `steps` binds each action to its function.

Between the two quotes (`standalone_activity.jl:429-440`) sits `evidence`, a table with one line per
fact such as `statusCompleted = statusCompleted` and `attemptCount = attemptCount`. It maps each
fact to what the referee reads. For a system with history events this would be an event name. Standalone activities have none, so every entry is a status field read through
`DescribeActivityExecution`, and `attemptCount` is the derived observation declared earlier:

```julia
@observation attemptCount begin
    on = activity
    read = attempt
end
```
(`standalone_activity.jl:97-100`)

At expansion time the macro checks that every `steps` key is a declared action or a listed timer and
that the block's keys are legal, and reports mistakes at their line. It then emits
`const activityProtocol = Umpire.build_machine(...)`. That function (`Umpire.jl:312-325`) looks
up each action's declaration, calls `build_table`, and when `refines` is given runs the refinement
check and throws if any row is rejected:

```julia
        m.refinement.rejected === nothing ||
            error("machine $name: refinement of $(refines.name) rejected at row `$(m.refinement.rejected)`")
```
(`Umpire.jl:321-322`)

When does this run? When the `const` is evaluated. If the Models are a Julia package, that is during
package precompilation and the result is cached, so a broken refinement makes `using` fail. In this
sample, `pins.jl` `include`s the files into a plain module at test time, so every table is built
when the tests load. The README's "compile time" column describes the package layout, not what the
sample as written does.

## Two levels and the refinement

The product machine says what the caller sees through Describe: nine statuses, no attempt count, no
backoff phase. The protocol machine says how the server gets there: `unstarted` before the request,
`backingOff` between a retryable failure and the retry, `pauseRequested` while a running attempt is
asked to pause, plus the deadlines. Properties are easier to state on the product; the refinement
carries them to the protocol.

`productOf` is the abstraction map:

```julia
function productOf(state::ProtocolState)::ProductState
    phase = @match state.phase begin
        Phase.unstarted || Phase.scheduled || Phase.backingOff => ProductPhase.scheduled
        Phase.started || Phase.pauseRequested => ProductPhase.started
        Phase.paused          => ProductPhase.paused
        Phase.cancelRequested => ProductPhase.cancelRequested
        Phase.completed       => ProductPhase.completed
        Phase.failed          => ProductPhase.failed
        Phase.canceled        => ProductPhase.canceled
        Phase.terminated      => ProductPhase.terminated
        Phase.timedOut        => ProductPhase.timedOut
    end
    return ProductState(phase = phase)
end
```
(`standalone_activity.jl:403-416`)

`pauseRequested` maps to `started`, not `paused`, because the worker still holds the attempt. Every
answer the worker can give from `pauseRequested` then maps onto a product row from `started`, and
the pause request itself becomes a stutter.

The rule: for every protocol transition from `s` to `s'`, either the mapped states are equal (a
stutter, the product did not move) or the product has some transition from `map(s)` to `map(s')`
under any action class. The check is fully written:

```julia
function check_refinement(proto::Machine, target::Machine, map::Function)
    rows = Dict{String,Union{Nothing,String}}()
    rejected = nothing
    for (s, c, st) in transitions(proto)
        rowkey = key(s) * "-" * key(c)
        before, after = map(s), map(st.state)
        if before == after
            rows[rowkey] = nothing                       # a product stutter
            continue
        end
        hit = findfirst(pc -> any(pst -> pst.state == after, target.table.rows[(before, pc)]),
                        target.table.classes)
        if hit === nothing
            rejected = something(rejected, rowkey)
            rows[rowkey] = "rejected"
        else
            rows[rowkey] = key(before) * "-" * key(target.table.classes[hit])
        end
    end
    return Refinement(target, map, rows, rejected)
end
```
(`Umpire.jl:283-303`)

One row by hand. Protocol state `started-1-unset-unset-unset` (phase started, one attempt, no
deadlines), class `attemptResult-failed-true`. The step goes to `backingOff-1-unset-unset-unset`
with fact `attemptCount`. `map` gives `started` before and `scheduled` after; they differ, so this
is not a stutter. The product table is searched for any class with a row from `started` to
`scheduled`: `attemptResult(failed(true))` from `started` provides it. The verdict recorded is
`started-attemptResult-failed-true`. Had the product been blind to retries, as the first spec draft
had it, this row would have been rejected and `build_machine` would have thrown.

The Lean checker is stricter: the matching product row must also share the outcome and its facts
must be among the protocol row's facts by evidence name. Under that rule this same row would need
`statusScheduled` beside `attemptCount`. This sample implements the mapped-states rule as the spec
states it.

## Properties

A property is a claim about a machine. A **same-step claim** names an action under `when` and holds
of the step that action produced. A **transition claim** has no `when` and holds of a step and the
step after it. The framework tells them apart by the arity of `holds`, checked with `hasmethod`
when the `Property` is built (`Umpire.jl:392-394`), so the author writes no annotation.

Same-step, on the protocol machine:

```julia
@property completes begin
    machine = activityProtocol
    when = attemptResult(completed)
    holds = step -> step.state.phase == Phase.completed && ProtocolFact.statusCompleted in step.facts
end
```
(`standalone_activity.jl:466-470`)

```julia
@property retryCompletes begin
    machine = activityProtocol
    when = attemptResult(completed)
    holds = step -> step.state == completedOnRetry && ProtocolFact.statusCompleted in step.facts
end
```
(`standalone_activity.jl:483-487`)

`completedOnRetry` is a full state literal with `attempts = 2`, so the claim pins the count as well
as the phase. Transition claims, on the product machine:

```julia
@property terminalIsFinal begin
    machine = activityProduct
    holds = (before, after) ->
        !productTerminal(before.state) || after.state.phase == before.state.phase
end
```
(`standalone_activity.jl:459-463`)

```julia
@property pausedIsNotDispatched begin
    machine = activityProduct
    holds = (before, after) ->
        before.state.phase != ProductPhase.paused || after.state.phase != ProductPhase.started
end
```
(`standalone_activity.jl:512-516`)

Both are declared on the product and read on the protocol through `productOf`.

## Scenarios and limits

A scenario is a path: a start phase and classed actions in order. Inputs are written bare, and the
framework resolves `completed` against `AttemptResult.Type` and `unset` against `Timeout.T`.

```julia
@scenario retriedThenCompleted begin
    model = activityProtocol
    starts = unstarted
    actions = [start(unset, unset, unset), attemptStart, attemptResult(failed(true)), backoff,
               attemptStart, attemptResult(completed)]
end
```
(`standalone_activity.jl:553-558`)

```julia
@scenario pausedThenCompleted begin
    model = activityProtocol
    starts = unstarted
    actions = [start(unset, unset, unset), control(pause), control(unpause), attemptStart,
               attemptResult(completed)]
end
```
(`standalone_activity.jl:573-578`)

Limits bound a search: how many steps deep, how many scenario actions, how many candidates to
examine before giving up. `six` is steps 6, actions 6, search 262144
(`standalone_activity.jl:604-608`); the six-action scenarios above need it.

## Queries

A query joins a property, a scenario and limits. `find` asks for one path of the scenario's shape on
which the property holds at its `when` step; that witness becomes a Case. `verify` asks that the
property hold on every candidate path; nothing is realized.

```julia
@query retry begin
    find = retryCompletes
    var"in" = retriedThenCompleted
    limits = six
end
```
(`standalone_activity.jl:626-630`)

```julia
@query pauseHolds begin
    verify = pausedIsNotDispatched
    var"in" = pausedThenCompleted
    limits = six
end
```
(`standalone_activity.jl:668-672`)

The macro emits two constants: the `Query` and `retry_result`, the outcome of running it. The search
itself is sketched in this sample:

```julia
function run(q::Query)
    # Sketched: iterative DFS over q.scenario.model.table.rows with the two cuts above.
    error("run: sketched")
end
```
(`Umpire.jl:475-478`)

What it would do: walk the table from the scenario's start over every enabled class, at most
`limits.steps` deep, examining at most `limits.search` candidates; for `find`, return the candidate
whose classes are the scenario's actions in order and whose `when` step satisfies the property; for
`verify`, check the property on every candidate along which the scenario's actions occur. On failure
it throws `SearchFailed`, whose message names the query, the limits, and the closest candidate with
the clause that failed. As written, every `_result` constant throws `run: sketched` at load.

## Sets

A set is what gets run. `bind` says per party whether the Case drives it (performs its actions) or
observes it (reads what happened and checks the machine allows it).

```julia
@set standaloneActivityTests begin
    purpose = functional
    bind = (
        caller = driven,
        worker = driven,
    )
    queries = [completion, nonRetryableFailure, retry, cancel, terminate, pauseResume,
               scheduleToStartTimeout, startToCloseTimeout]
end
```
(`standalone_activity.jl:680-688`)

The canary set (`standalone_activity.jl:695-702`) runs `completion` and `cancel` against a
deployment whose own worker attempts, so its `bind` has `worker = observed`. The exploratory set
(`standalone_activity.jl:706-715`) lists no queries; it names `machine = activityProtocol`,
`cover = rows | results | classMembers` and `budget = four`, and covers the machine's rows, the
results they reach and the members of their classes within that budget. The Nexus set has `repeat = implementation` to run once under each of the HSM and
CHASM implementations; this set has none because standalone activities exist only in CHASM.

## Composition with the worker

The worker is its own entity with a two-phase machine, `polling` and `stopped`, and three actions:
`workerStop`, `workerResume` and `serve`. Its step functions look like any other; `serveStep`
(`worker.jl:76-80`) keeps a polling worker polling and returns `WStep[]` for a stopped one.

The activity's view of it drops `workerResume`, so a stopped worker stays stopped:

```julia
@machine activityWorker begin
    from = Worker.polling
    restrict = [workerStop, serve]
end
```
(`standalone_activity.jl:724-727`)

The composition runs both machines side by side. A `sync` line fuses two member actions into one:

```julia
@compose standaloneActivity begin
    var"for" = [activity, Worker.worker]
    state = StandaloneActivityState
    members = (
        activity = activityProtocol,
        worker = activityWorker,
    )
    sync = (
        workerStop = activity.workerStop ∥ worker.workerStop,
        attemptStart = activity.attemptStart ∥ worker.serve,
    )
    starts = [activity.unstarted, worker.polling]
    ends = [activity.completed, activity.failed, activity.canceled, activity.terminated,
            activity.timedOut]
end
```
(`standalone_activity.jl:734-748`)

`∥` is a Unicode operator Julia's parser already knows. `attemptStart` now fires only when the
worker can `serve`, which is only while it polls. The composed claim:

```julia
@property startedByPollingWorker begin
    machine = standaloneActivity
    when = attemptStart
    holds = step -> step.state.worker.phase == Worker.Phase.polling
end
```
(`standalone_activity.jl:751-755`)

It is verified over `stoppedBeforeRetry` (`standalone_activity.jl:760-765`): start with a
schedule-to-start deadline, let the worker serve one attempt that fails retryably, stop the worker
during the backoff, let the deadline fire. The path performs an `attemptStart`, so the claim is
exercised on a real step rather than holding vacuously; the spec's first version of this scenario
never dispatched and was replaced for that reason. What it proves is that the activity's own
`workerStop` stutter is consistent with the real thing: with the worker modeled, no dispatch happens
after a stop, and the timer is what settles the activity. The composite table builder `build_compose` is sketched in this
sample (`Umpire.jl:364-370`); it would take the product of the member tables, pairing synced steps
and letting unsynced member steps move one member at a time.

## Pins

`pins.jl` reads the built tables and asserts the counts and rows the spec fixes.

```julia
        @test length(activityProtocol.table.states) == 12 * (attemptBound + 1) * 8   # 288
        @test length(activityProtocol.ends) == 5 * (attemptBound + 1) * 8            # 120
        # A canceled result with no cancel requested is not honored.
        @test protocolAttemptResultStep(at(Phase.started), AttemptResult.canceled) == []
        # A retryable failure while a cancel is requested settles as canceled, not backed off.
        @test only(protocolAttemptResultStep(at(Phase.cancelRequested), AttemptResult.failed(true))).state.phase ==
            Phase.canceled
```
(`pins.jl:154-160`)

The first two guard the state type: adding a field or a phase changes the count. The third guards
the `canceled` arm's guard. The fourth guards the `cancelRequested` branch of the retryable arm.

```julia
        @test activityProtocol.refinement.rejected === nothing
        @test length(activityProtocol.refinement.rows) == length(Umpire.transitions(activityProtocol))
```
(`pins.jl:168-169`)

These guard that the refinement walked every transition and rejected none. `at` is a small helper
that writes a protocol state by phase with the other fields defaulted.

## From model to running test

Nothing past this file is in the sample. In the full system, a `find` query's witness is lowered to
a Case: the classes along the path become the actions a Go runtime performs, and the `evidence`
lines along the path become the Contract the referee checks. A realization maps each class to a
concrete request (the `schema` and `examples` lines feed it). The Go Testpilot runtime performs the
Case against a server and returns a verdict per Contract clause. No realization exists yet for
standalone activities; the Model is ahead of the runtime here.

## Gaps and gradual growth

The Model says "not modeled" in several places, and each is a deliberate seam. Reset and the
heartbeat timeout are absent by declaration in the module header. `workerStop` is a stutter on the
protocol machine (keep state, record nothing) and an empty step on the product; the composition is
where it gets meaning. Empty steps (`TStep[]`) mark actions that are not enabled rather than
forbidden, so adding a new arm later is a local change. A new action is declared with `@action`,
given a step function per machine, added to `steps`, and the table, refinement and pins say whether
the rest of the Model still holds. The refinement in particular means the product can stay small
while the protocol grows.

## Mental model recap

- A Model is a rulebook: finite states, finite action classes, step functions that say what each
  class does from each state.
- The framework builds the full table by calling every step on every state; that is also what
  exercises every match arm.
- Two machines, one abstraction map; the refinement check proves the detailed one never does
  something the simple one forbids.
- Same-step properties become Contracts on a Case; transition properties are only verified.
- Scenarios are the paths tests take; queries search the table for them within limits.
- Sets decide who is driven and who is observed, and which queries ship as tests.
- Composition with the worker gives the `workerStop` stutter a real meaning.
- Facts are Describe status reads here, not history events; `evidence` says what to read.

## Where this implementation is weak

- **The search and the composite table are stubs.** `Umpire.run` and `build_compose` both call
  `error("sketched")`, so every `@query` result constant and the `@compose` constant throw at load.
  The README's `SearchFailed` example describes intended behaviour, not code that exists.
- **"Compile time" means package precompile, and the sample is not a package.** `pins.jl` includes
  the Model files into a plain module, so tables, refinements and queries all run at test time. The
  precompile story is real for a package layout but is not what the files as written do.
- **Library behaviour is unverified.** The Moshi reflection functions used by `finite` and `keypart`,
  `Moshi.Match.MatchError`, the named-field call variant `failed(retryable::Bool)`, and `@match` on
  EnumX values with `||` patterns are all plausible and none were run through a toolchain.
- **Fixed after review.** The reviewer found `ending`/`starting` comparing enum values to symbols,
  and `classof` unable to resolve `activity.start(...)` in the composition scenario. Both are fixed
  in the current files, along with `finite` calling singleton variants and the `.machine` leak in
  `@property`/`@scenario`. The registry of actions is a global in `Umpire` that would not survive a
  precompile boundary between packages; within one package load it works.
