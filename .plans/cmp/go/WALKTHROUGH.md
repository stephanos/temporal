# Walkthrough: the standalone activity Model in Go

## What this is

Umpire is model-based testing for Temporal. A Model is the rulebook: for every state the system can
be in and every move a participant can make, it says what may happen next and what the system must
show afterwards. A test is a playthrough of that rulebook against a real server, and the framework is
the referee that compares what the server did with what the rulebook allows.

A standalone activity is a Temporal activity started directly with `StartActivityExecution`. There is
no workflow around it, so it writes no history events. The only way to see it is to ask: the caller
reads its status through `DescribeActivityExecution` and its result through `PollActivityExecution`.

`standaloneactivity/standalone_activity.go` is the rulebook for one such activity: two machines (what
the caller sees, and how the server gets there), the claims they make, the paths the tests walk, and
the sets of tests to run. `standaloneactivity/pins_test.go` nails down what the rulebook says.

## The language in five minutes

The sample is Go. These are the only features the walkthrough relies on.

**Package-level variables** hold every declaration. `var activity = &umpire.Entity{...}` builds a
struct and takes its address with `&`, so other declarations refer to it by identity.

**Struct literals with named fields** are how machines and sets are written. Unnamed fields take
their zero value: `ProtocolState{Phase: Unstarted}` has attempts 0 and every deadline unset.

**Constants with `iota`** are Go's enums. In a `const` block `iota` counts from 0, so `Pause Control
= iota` is 0 and `Unpause` is 1. Constants are package-scoped, which is why some carry a prefix.

**Generics** put type parameters in square brackets: `umpire.Machine[ProtocolState, ProtocolOutcome,
ProtocolFact]`. Calls infer them from arguments, so `umpire.Bind1(control, controlStep)` writes none.

**A sealed interface** is an interface with an unexported method. Only the same package can
implement it, so its variants form a closed set. That is Go's sum type, spelled by hand.

**A type switch**, `switch result := result.(type) { case AttemptFailed: ... }`, branches on which
variant an interface value holds and binds it with that type inside the arm.

**Slices** are `[]T`; a `nil` slice is empty. Step functions return `nil` to say "not enabled".

**Struct tags** are string annotations on fields that reflection reads, like `umpire:"activity"`.
**A type alias** such as `type ProductStep = umpire.Step[...]` gives a long generic type a short name.

## Vocabulary: entities, parties, actions, inputs

An **entity** is the thing a machine is about. The activity is named by the id its caller chose.

`standaloneactivity/standalone_activity.go:33`
```go
var activity = &umpire.Entity{Name: "activity", Key: "activityId"}
```

A **party** is who acts. `umpire/umpire.go:22-31` declares them as string constants: `Caller`,
`Handler`, `Worker`, `Network`, `Operator` and `System`. This Model uses the caller, the worker, and
`System`, which is reserved for timers.

An **action** is a named side effect of a party. It may create or act on an entity, carry protobuf
messages, and take finite inputs. The five actions here are `start`, `attemptStart`, `attemptResult`,
`control` and `workerStop`.

`standaloneactivity/standalone_activity.go:74-80`
```go
var start = &umpire.Action3[common.Timeout, common.Timeout, common.Timeout]{
	Name:    "start",
	Party:   umpire.Caller,
	Creates: activity,
	Schema:  umpire.Schema(&workflowservice.StartActivityExecutionRequest{}),
	Inputs:  [3]string{"scheduleToClose", "scheduleToStart", "startToClose"},
}
```

`Action3` carries three inputs, `Action0` none, `Action1` one; the arity is in the type because Go has
no variadic type parameters. `umpire.Schema` takes real protobuf message values and reads their full
names back, so a renamed API message is a compile error. `attemptStart` (lines 83-88) is an `Action0`
of the worker on the activity, carrying `PollActivityTaskQueueResponse`.

An **input domain with payloads** is a sealed interface. `AttemptResult` has three constructors, and
`AttemptFailed` carries a boolean.

`standaloneactivity/standalone_activity.go:43-56`
```go
//sumtype:decl
type AttemptResult interface{ isAttemptResult() }

type (
	AttemptCompleted struct{}
	AttemptFailed    struct{ Retryable bool }
	AttemptCanceled  struct{}
)

func (AttemptCompleted) isAttemptResult() {}
func (AttemptFailed) isAttemptResult()    {}
func (AttemptCanceled) isAttemptResult()  {}

var attemptResults = umpire.Sum[AttemptResult](AttemptCompleted{}, AttemptFailed{}, AttemptCanceled{})
```

An **action class** is one action with one assignment of its inputs. `umpire.Sum` lists the variants
and expands `AttemptFailed{}` into both boolean values, so `attemptResult` has four classes:
completed, failed(false), failed(true), canceled. `Control` (lines 58-65) is a plain `iota` enum with
`Pause`, `Unpause`, `RequestCancel` and `Terminate`, so `control` has four classes too.

The action binds its domain and maps two classes to concrete realization values.

`standaloneactivity/standalone_activity.go:99-104`
```go
	Input:   "result",
	Classes: attemptResults,
	Examples: map[AttemptResult]string{
		AttemptFailed{Retryable: false}: "ApplicationFailure nonRetryable",
		AttemptFailed{Retryable: true}:  "ApplicationFailure retryable",
	},
```

A **fault** is just an action. The worker stopping is an action of the `Worker` party that names no
entity, so no machine records anything at it.

`standaloneactivity/standalone_activity.go:121-122`
```go
// The worker stops polling. It names no entity, so the machines keep their state and record nothing.
var workerStop = &umpire.Action0{Name: "workerStop", Party: umpire.Worker}
```

Timers are `System` actions declared the same way at lines 124-130: `timeout`, `backoff`,
`scheduleToClose`, `scheduleToStart`, `startToClose`. A machine claims the ones it owns.

## State

The **product state** is one field: the phase the caller reads through Describe. `ProductPhase`
(lines 149-161) has nine members, from `ProductScheduled` to `ProductTimedOut`.

`standaloneactivity/standalone_activity.go:163-165`
```go
type ProductState struct {
	Phase ProductPhase
}
```

The **protocol state** adds what the server tracks: a bounded attempt count and whether each of
three deadlines will fire.

`standaloneactivity/standalone_activity.go:370-376`
```go
type ProtocolState struct {
	Phase           Phase
	Attempts        Attempts
	ScheduleToClose common.Timeout
	ScheduleToStart common.Timeout
	StartToClose    common.Timeout
}
```

Every field is finite because the framework enumerates all states to build a table. The protocol
`Phase` has twelve members, `Attempts` three values, each `Timeout` two: 12 × 3 × 2 × 2 × 2 = 288
states. The product has 9.

**Enumeration** goes through a `Values()` method. An integer enum gets it from a generator named in a
`//go:generate` line at the top of the file; the generator itself is not part of the sample. The
bounded counter writes it by hand, with a successor that stops at the bound instead of wrapping.

`standaloneactivity/standalone_activity.go:351-368`
```go
const attemptBound = 2

type Attempts uint8

func (Attempts) Values() []Attempts {
	values := make([]Attempts, 0, attemptBound+1)
	for i := Attempts(0); i <= attemptBound; i++ {
		values = append(values, i)
	}
	return values
}

func (a Attempts) saturatingSucc() Attempts {
	if a < attemptBound {
		return a + 1
	}
	return a
}
```

A struct state's domain comes from `umpire.Fields[ProtocolState]()` (`umpire/umpire.go:78-81`). Its
body is sketched. It would walk the struct's fields by reflection, call each field type's `Values()`,
and return every combination in field order. Nothing stops a step function from building an
`Attempts` of 7; the table build rejects such a row at test time.

## Step functions

A step function has the shape `(state, inputs) -> list of (outcome, next state, facts)`. The list
element is `umpire.Step` (`umpire/umpire.go:177-181`), a struct with `Outcome`, `State` and `Facts`.
One entry means the action is enabled and this is what happens; `nil` means not enabled.

The product `attemptResultStep` is small enough to read arm by arm.

`standaloneactivity/standalone_activity.go:220-244`
```go
func attemptResultStep(state ProductState, result AttemptResult) []ProductStep {
	if state.Phase != ProductStarted && state.Phase != ProductCancelRequested {
		return nil
	}
	switch result := result.(type) {
	case AttemptCompleted:
		return productStep(ProductCompleted, ProductStatusCompleted)
	case AttemptFailed:
		if !result.Retryable {
			return productStep(ProductFailed, ProductStatusFailed)
		}
		if state.Phase == ProductCancelRequested {
			return productStep(ProductCanceled, ProductStatusCanceled)
		}
		return productStep(ProductScheduled, ProductStatusScheduled)
	case AttemptCanceled:
		// A worker cancels only an attempt whose cancel was requested.
		if state.Phase != ProductCancelRequested {
			return nil
		}
		return productStep(ProductCanceled, ProductStatusCanceled)
	default:
		panic(fmt.Sprintf("unhandled AttemptResult %T", result))
	}
}
```

The guard says a result only lands on an attempt a worker holds. A completed attempt completes the
activity and the status reads completed. A non-retryable failure fails it. A retryable failure of a
cancel-requested attempt honors the cancel; otherwise the status reads scheduled again, which is what
CHASM's `TransitionRescheduled` does. A worker's cancel is accepted only when the caller asked for it.
`productStep` (lines 193-195) wraps one accepted step with one fact.

The protocol version dispatches on phase as well as result, because three source phases react
differently to a retryable failure.

`standaloneactivity/standalone_activity.go:477-503`
```go
func protocolAttemptResultStep(state ProtocolState, result AttemptResult) []ProtocolStep {
	if !held(state.Phase) {
		return nil
	}
	switch result := result.(type) {
	case AttemptCompleted:
		return moves(state, Completed, StatusCompleted{})
	case AttemptFailed:
		if !result.Retryable {
			return moves(state, Failed, StatusFailed{})
		}
		if state.Phase == CancelRequested {
			return moves(state, Canceled, StatusCanceled{})
		}
		if state.Phase == PauseRequested {
			return moves(state, Paused, StatusPaused{})
		}
		return moves(state, BackingOff, AttemptCount{})
	case AttemptCanceled:
		if state.Phase != CancelRequested {
			return nil
		}
		return moves(state, Canceled, StatusCanceled{})
	default:
		panic(fmt.Sprintf("unhandled AttemptResult %T", result))
	}
}
```

`held` (lines 433-435) is `Started`, `PauseRequested` or `CancelRequested`: the phases in which a
worker holds an attempt. A retryable failure from `Started` backs off, mirroring
`TransitionRescheduled`. From `CancelRequested` it cancels, because in `statemachine.go`
CANCEL_REQUESTED is a source of Canceled and not of Rescheduled. From `PauseRequested` it pauses,
mirroring `TransitionAttemptFailedWhilePauseRequested`. `moves` (lines 437-440) copies the state,
changes the phase and attaches the facts; Go passes structs by value, so the copy is free.

**Exhaustiveness** is not the compiler's job in Go. Two linters do it: `exhaustive` for `switch` on
integer enums and `go-check-sumtype` for type switches on interfaces marked `//sumtype:decl`. Both
report a missing arm even when a `default` is present. The `default: panic(...)` arm exists because
the compiler still wants a terminating statement after a switch it cannot prove complete. Without the
linters, nothing checks the arms.

## The machine and its table

A machine is a struct literal that ties the state type, starts and ends, the timers it owns, the
evidence its facts stand for, and the step functions together.

`standaloneactivity/standalone_activity.go:612-620`
```go
var ActivityProtocol = &umpire.Machine[ProtocolState, ProtocolOutcome, ProtocolFact]{
	Name:         "activityProtocol",
	For:          activity,
	States:       umpire.Fields[ProtocolState](),
	Refines:      umpire.Refines(ActivityProduct, productOf),
	Starts:       []ProtocolState{{Phase: Unstarted}},
	Ends:         func(s ProtocolState) bool { return terminalPhase(s.Phase) },
	Timers:       []umpire.Action{backoff, scheduleToClose, scheduleToStart, startToClose},
	Unobservable: []umpire.Action{backoff},
```

`Starts` is one state; the zero values of the other fields are what the activity begins with. `Ends`
is a predicate rather than a list, since Go has no phase-only view of a struct. `Timers` are the
`System` actions this machine owns and `Unobservable` the timers that record nothing.

`standaloneactivity/standalone_activity.go:621-636`
```go
	Evidence: map[ProtocolFact]string{
		StatusScheduled{}:       "statusScheduled",
		StatusStarted{}:         "statusStarted",
		StatusPaused{}:          "statusPaused",
		StatusCancelRequested{}: "statusCancelRequested",
		StatusCompleted{}:       "statusCompleted",
		StatusFailed{}:          "statusFailed",
		StatusCanceled{}:        "statusCanceled",
		StatusTerminated{}:      "statusTerminated",
		StatusTimedOut{}:        "statusTimedOut",
		AttemptCount{}:          attemptCount.Name,
	},
	Steps: umpire.Steps(
		umpire.Bind3(start, startStep),
		umpire.Bind0(attemptStart, protocolAttemptStartStep),
		umpire.Bind1(attemptResult, protocolAttemptResultStep),
```

**Evidence** maps each fact to a name the realization can resolve. In a system with no history
events, those names are status reads: `statusCompleted` means "Describe returned COMPLETED", and
`attemptCount` is the derived observation declared at line 138 that reads the `attempt` field. The
key `StatusTimedOut{}` stands for all three of its classes. `BindN` pairs an action with a step whose
input types the compiler checks against the action's; six more bindings follow at lines 637-642.

**The table** is built at test time by `ActivityProtocol.Check(t)` (`umpire/umpire.go:247-250`),
whose body is sketched. It would enumerate `States`, call every bound step on every state and action
class, collect the rows, verify each landing state is in the domain and each recorded fact has an
evidence line, compute reachability from `Starts`, and run the refinement. It returns the `Table`
(`umpire.go:259-266`): states, ends, action keys, rows, reachable states and a stuck state if one
exists. Lean does all of this while compiling; Go does it a second later under `go test`.

## Two levels and the refinement

The **product machine** says what the caller sees. The **protocol machine** says how the server gets
there: backoff, pause requests a worker must yield to, three separate deadlines, the attempt count.
Writing both lets a claim be proved once on the small machine and carried to the large one.

`productOf` says how a protocol state reads as a product state. The interesting arms are the first
three; the rest map each phase to the product phase of the same name.

`standaloneactivity/standalone_activity.go:587-594`
```go
func productOf(state ProtocolState) ProductState {
	switch state.Phase {
	case Unstarted, Scheduled, BackingOff:
		return ProductState{Phase: ProductScheduled}
	case Started, PauseRequested:
		return ProductState{Phase: ProductStarted}
	case Paused:
		return ProductState{Phase: ProductPaused}
```

`PauseRequested` maps to `started`, not `paused`: the worker still holds the attempt, and the caller
sees the pause only once the worker yields. Backing off and not yet started both read as scheduled.
Every other field is hidden.

**The refinement rule** this sample implements is by mapped states, as the `Refines` field's comment
at `umpire/umpire.go:237-239` states: a protocol row from `s` to `s'` passes if `productOf(s) ==
productOf(s')` (a stutter) or the product table has any row from `productOf(s)` to `productOf(s')`,
under any action class. The walk itself is sketched (`umpire.go:550` panics). It would run inside
`Check` at test time, produce a `RefinementReport` mapping each protocol row key to a product row or a
stutter, and fail the test at the first row that is neither.

One row by hand. Protocol state `{Started, attempts 1, all unset}` on `attemptResult(failed true)`
goes to `{BackingOff, attempts 1, all unset}` recording `AttemptCount{}`. The map gives `started` and
`scheduled`. The product table has a row `started -> scheduled` on `attemptResult(failed true)`
(line 234), so the row passes.

The real Lean checker is stricter: the matching product row must have the same outcome, and its
facts must be among the protocol row's facts, compared by evidence name. Under that rule this row
would need `StatusScheduled{}` beside `AttemptCount{}` at line 494. The spec says the non-Lean
samples implement the mapped-states rule as stated, and this sample does.

## Properties

A **same-step claim** names the action class it is about under `When` and states what that step must
look like under `Holds`.

`standaloneactivity/standalone_activity.go:663-670`
```go
var completes = &protocolProperty{
	Name:    "completes",
	Machine: ActivityProtocol,
	When:    attemptResult.With(AttemptCompleted{}),
	Holds: func(step ProtocolStep) bool {
		return step.State.Phase == Completed && step.Records(StatusCompleted{})
	},
}
```

`protocolProperty` is an alias (line 417) for `umpire.Property[ProtocolState, ProtocolOutcome,
ProtocolFact]`. `step.Records` is a method on `Step` rather than `slices.Contains`, because Go's
inference will not unify a `[]ProtocolFact` with a `StatusCompleted{}` argument.

`retryCompletes` fixes the whole state by comparing against `completedOnRetry` (lines 683-689), a
value that names every field: completed, attempts 2, all deadlines unset.

`standaloneactivity/standalone_activity.go:692-699`
```go
var retryCompletes = &protocolProperty{
	Name:    "retryCompletes",
	Machine: ActivityProtocol,
	When:    attemptResult.With(AttemptCompleted{}),
	Holds: func(step ProtocolStep) bool {
		return step.State == completedOnRetry && step.Records(StatusCompleted{})
	},
}
```

A **transition claim** has no `When`. It holds of every pair of consecutive steps.

`standaloneactivity/standalone_activity.go:654-660`
```go
var terminalIsFinal = &productProperty{
	Name:    "terminalIsFinal",
	Machine: ActivityProduct,
	Transition: func(before, after ProductStep) bool {
		return !productTerminal(before.State) || after.State.Phase == before.State.Phase
	},
}
```

`standaloneactivity/standalone_activity.go:732-738`
```go
var pausedIsNotDispatched = &productProperty{
	Name:    "pausedIsNotDispatched",
	Machine: ActivityProduct,
	Transition: func(before, after ProductStep) bool {
		return before.State.Phase != ProductPaused || after.State.Phase != ProductStarted
	},
}
```

Both are declared on the product machine and read on the protocol machine through `productOf`. The
`Property` struct (`umpire.go:309-318`) has both `Holds` and `Transition` fields; `Check` rejects a
property that sets both or neither, since the struct cannot.

## Scenarios and limits

A scenario is a start state and a list of classed actions in order. Two shared values keep the
paths short: `unstarted` is `ProtocolState{Phase: Unstarted}` and `noDeadlines` is
`start.With(common.Unset, common.Unset, common.Unset)` (lines 767-769).

`standaloneactivity/standalone_activity.go:791-799`
```go
var retriedThenCompleted = &protocolScenario{
	Name:   "retriedThenCompleted",
	Model:  ActivityProtocol,
	Starts: unstarted,
	Actions: []umpire.Class{
		noDeadlines, attemptStart, attemptResult.With(AttemptFailed{Retryable: true}),
		backoff, attemptStart, attemptResult.With(AttemptCompleted{}),
	},
}
```

A classed action is `action.With(inputs...)`, typed by the action. An action without inputs, like
`attemptStart` or `backoff`, is written bare. The retryable failure backs the attempt off, the backoff
timer fires and records nothing, and the second attempt completes with the count at 2.

`standaloneactivity/standalone_activity.go:824-827`
```go
	Actions: []umpire.Class{
		noDeadlines, control.With(Pause), control.With(Unpause), attemptStart,
		attemptResult.With(AttemptCompleted{}),
	},
```

**Limits** bound the search: how many steps a path may take, how many actions it may name, and how
many candidates to examine. Lines 852-856 declare `three` (3, 3, 4096), `four` (4, 4, 32768) and
`six` (6, 6, 262144); a six-action path like the retry needs `six`.

## Queries

A query pairs a property with a scenario under limits. **Find** asks for a path of the scenario's
shape on which the same-step claim holds at its action; a set later realizes it as a test. **Verify**
asks whether a transition claim holds over every trace of the path; it is never realized.

`standaloneactivity/standalone_activity.go:868` and `:875`
```go
	retry                  = &umpire.Query{Name: "retry", Find: retryCompletes, In: retriedThenCompleted, Limits: six}
	pauseHolds             = &umpire.Query{Name: "pauseHolds", Verify: pausedIsNotDispatched, In: pausedThenCompleted, Limits: six}
```

`Query` (`umpire/umpire.go:350-356`) is not generic. Its `Find`, `Verify` and `In` fields are erased
interfaces, because `pauseHolds` pairs a product property with a protocol scenario and Go's type
system cannot say "a property of a machine this scenario's machine refines". `Check` confirms the
pairing at test time.

The search runs under `go test` through `Answer` (`umpire.go:368-371`), whose body is sketched. It
would walk the table from the scenario's start along its action classes within the limits, and
return an `Answer` (`umpire.go:359-366`) with a witness path for `Find` or a counterexample for
`Verify`, plus a count of candidates and an `Explanation`.

What an author sees on failure is a failing subtest named after the query with the explanation as
its message: which scenario, which property, where the path's last step landed and why the claim did
not hold there, and how many candidates were searched.

## Sets

A set is what the runtime registers. It binds each party as **driven** (the test performs its
actions) or **observed** (the test reads what it did).

`standaloneactivity/standalone_activity.go:884-895`
```go
var StandaloneActivityTests = &umpire.Set{
	Name:    "standaloneActivityTests",
	Purpose: umpire.Functional,
	Bind: map[umpire.Party]umpire.Role{
		umpire.Caller: umpire.Driven,
		umpire.Worker: umpire.Driven,
	},
	Queries: []*umpire.Query{
		completion, nonRetryableFailure, retry, cancel, terminate, pauseResume,
		scheduleToStartTimeout, startToCloseTimeout,
	},
}
```

There is no `Repeat` line. The Nexus Model repeats each query under HSM and CHASM; standalone
activities exist only in CHASM, so there is nothing to repeat over.

The **canary** set runs against a deployment that performs the worker's part itself, so the worker is
observed, and it names only queries whose every step records evidence.

`standaloneactivity/standalone_activity.go:904-908`
```go
	Bind: map[umpire.Party]umpire.Role{
		umpire.Caller: umpire.Driven,
		umpire.Worker: umpire.Observed,
	},
	Queries: []*umpire.Query{completion, cancel},
```

The **exploratory** set names a machine and a budget instead of queries and covers its rows, results
and class members.

`standaloneactivity/standalone_activity.go:920-922`
```go
	Machine: ActivityProtocol,
	Cover:   umpire.Rows | umpire.Results | umpire.ClassMembers,
	Budget:  four,
```

## Composition with the worker

The protocol machine treats a worker stop as a stutter: the activity cannot see its worker. To prove a
claim across both, the Model composes with the worker entity from `worker/worker.go`.

`worker/worker.go:97-101`
```go
	Steps: umpire.Steps(
		umpire.Bind0(WorkerStop, stopStep),
		umpire.Bind0(WorkerResume, resumeStep),
		umpire.Bind0(Serve, serveStep),
	),
```

The `Polling` machine (lines 91-102) has a two-phase state, polling or stopped, starts polling and
may end either way. A polling worker stops or serves; a stopped one resumes. The composition restricts it to stop and
serve, so a stopped worker stays stopped and the activity's timers settle every state it leaves.

`standaloneactivity/standalone_activity.go:940-957`
```go
var ActivityWorker = umpire.Restrict("activityWorker", worker.Polling, worker.WorkerStop, worker.Serve)

type StandaloneActivityState struct {
	Activity ProtocolState      `umpire:"activity"`
	Worker   worker.WorkerState `umpire:"worker"`
}

var StandaloneActivity = &umpire.Compose[StandaloneActivityState]{
	Name:    "standaloneActivity",
	For:     []*umpire.Entity{activity, worker.Worker},
	Members: umpire.Members{"activity": ActivityProtocol, "worker": ActivityWorker},
	Sync: umpire.Sync{
		workerStop:   {"activity": workerStop, "worker": worker.WorkerStop},
		attemptStart: {"activity": attemptStart, "worker": worker.Serve},
	},
	Starts: []StandaloneActivityState{{Activity: unstarted, Worker: worker.WorkerState{Phase: worker.PhasePolling}}},
	Ends:   func(s StandaloneActivityState) bool { return terminalPhase(s.Activity.Phase) },
}
```

The composed state is a struct whose tags name the members. Each `Sync` line makes two member actions
fire as one composed step: the activity's `workerStop` with the worker's, and the activity's
`attemptStart` with the worker's `serve`. Since `serve` has no row when the worker is stopped, an
attempt start has no row either.

`standaloneactivity/standalone_activity.go:960-967`
```go
var startedByPollingWorker = &umpire.Property[StandaloneActivityState, umpire.Joint, umpire.Joint]{
	Name:    "startedByPollingWorker",
	Machine: StandaloneActivity,
	When:    attemptStart,
	Holds: func(step umpire.Composed[StandaloneActivityState]) bool {
		return step.State.Worker.Phase == worker.PhasePolling
	},
}
```

`When: attemptStart` with no `With` means every class of the action. `umpire.Joint` stands in for the
members' outcomes and facts, which Go cannot type as a tuple. The claim proves that no stopped worker
ever starts an attempt, which the protocol machine alone could not state. `stoppedWorkerStartsNothing`
(lines 986-991) verifies it over `stoppedBeforeRetry` (lines 972-984): the activity starts with a
schedule-to-start deadline, a polling worker starts an attempt that fails retryably, the backoff
fires, the worker stops, and the deadline fires before any retry. Because the path performs
`attemptStart` once, the claim is exercised rather than vacuously true; the earlier version stopped
the worker before any dispatch and never fired the claim.

## Pins

Pins are ordinary Go tests that fix what the tables say.

`standaloneactivity/pins_test.go:31-32`
```go
	require.Len(t, table.States, 12*(attemptBound+1)*2*2*2)
	require.Len(t, table.Ends, 5*(attemptBound+1)*2*2*2)
```

These guard the state space: 288 states and 120 ends. A new phase or a wider bound changes both.

`standaloneactivity/pins_test.go:37-45`
```go
	// A retryable failure yields to whatever the caller asked for meanwhile.
	yields := func(phase Phase) Phase {
		steps := protocolAttemptResultStep(ProtocolState{Phase: phase, Attempts: 1}, AttemptFailed{Retryable: true})
		require.Len(t, steps, 1)
		return steps[0].State.Phase
	}
	require.Equal(t, BackingOff, yields(Started))
	require.Equal(t, Canceled, yields(CancelRequested))
	require.Equal(t, Paused, yields(PauseRequested))
```

This guards the three arms of the step function directly, so a refactor that merged them would fail.

`standaloneactivity/pins_test.go:52-55`
```go
func TestActivityProtocolRefinesProduct(t *testing.T) {
	report := ActivityProtocol.Refinement(t)
	require.Nil(t, report.Rejected)
}
```

This guards the refinement. A product change that left a protocol row without a counterpart, as the
first version of the spec did, would name the row in `Rejected`.

## From model to running test

Everything after this file is outside the sample. In the full system, each `Find` query of a
functional set is lowered to a Case: the witness path, the evidence names read off the machine's
`Evidence` lines at each step, and a contract clause from the property. A realization maps each
action class to a concrete call (`StartActivityExecution`, a worker poll, a Describe read) and each
evidence name to how to observe it. No realization exists yet for standalone activities. The Go
Testpilot runtime would then drive the driven parties, observe the rest, and return a verdict per
Case: pass, fail with the step and evidence that disagreed, or a known gap where a silent step could
not be confirmed.

## Gaps and gradual growth

The Model says "not modeled" in several places, on purpose. Reset is deferred, like cancellation in
the Nexus Model. The heartbeat timeout is absent. Late worker responses to a scheduled or paused
activity, which `statemachine.go` accepts, return `nil` here. The worker stop is a stutter on the
protocol machine and invisible on the product machine. `backoff` is `Unobservable`.

Every one of these is a `nil` return or a missing action, and that is what makes the Model growable.
To add an action: declare it as a `var` with its party, entity and inputs; write a step function for
each machine, returning `nil` where it is not enabled; add a `BindN` line to each machine; add an
`Evidence` entry for any new fact; write a property and a scenario if a query should find it; and add
a pin. The compiler checks the binding, the linters check the switches, and `go test` checks the
table, the evidence and the refinement.

## Mental model recap

- An action is a named side effect of a party; a class is an action with its inputs fixed; a fault
  and a timer are actions like any other.
- A step function maps a state and inputs to zero or one next step; `nil` means not enabled.
- Every state field is finite so the framework can enumerate the whole table: 9 product states, 288
  protocol states.
- The product machine is what the caller sees; the protocol machine is how the server gets there;
  `productOf` connects them and the refinement checks every protocol row against the product table.
- A same-step claim holds at one action's step; a transition claim holds across consecutive steps.
- A scenario is a path; a query finds or verifies a claim on it within limits; a set bundles queries
  for a purpose and binds parties as driven or observed.
- In Go, the compiler checks names and bindings, linters check exhaustiveness, and `go test` checks
  everything semantic.

## Where this implementation is weak

- **Nothing has been compiled or run.** Every framework body that builds a table, walks the
  refinement or searches is `panic("sketched")`. The error transcripts in the README are written by
  hand, not captured.
- **The `Values()` generator does not exist.** Every `//go:generate go run ../umpire/cmd/finite` line
  points at a command that is not in the sample. Without it, `umpire.Fields[ProtocolState]()` would
  fail at `Check` because `Phase` and `common.Timeout` have no `Values()` method.
- **Query pairing is untyped.** A property of one machine paired with a scenario of an unrelated
  machine compiles; only `Check` would reject it, and `Check` is sketched.
- **Exhaustiveness depends on linter defaults.** Both linters must run with
  `default-signifies-exhaustive` off for the `default: panic` arms to stay reportable. A lint
  configuration change silently removes the only exhaustiveness check; `go vet` alone checks nothing
  here.
