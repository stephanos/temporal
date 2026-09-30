# Walkthrough: the standalone activity Model in TypeScript

## What this is

Umpire is a model-based testing layer. A Model is the rulebook for one piece of Temporal behavior:
which moves exist, what each move does to the state, and what evidence each move leaves. A test run
is a playthrough of that rulebook against a real server, and a referee compares what the server
recorded with what the rulebook allowed. A standalone activity is an activity started directly with
`StartActivityExecution`, with no workflow around it; it writes no history events, so everything the
caller learns comes from `DescribeActivityExecution` and `PollActivityExecution`. The file
`standalone_activity.ts` is the rulebook for one such activity, written as ordinary TypeScript
against the small framework in `umpire.ts`.

## The language in five minutes

TypeScript is JavaScript plus a static type checker, `tsc`. Types exist only at compile time; at
runtime there are plain objects, arrays and functions. Six features carry this walkthrough.

**String literal unions.** A type can be a fixed set of strings. `enumOf` takes a list and returns a
value whose type remembers every member. The `typeof` operator reads a value's type, so the same
name can be a runtime value and a compile-time type:

`standalone_activity.ts:77-78`
```ts
export const Control = enumOf(["pause", "unpause", "requestCancel", "terminate"]);
export type Control = Member<typeof Control>;
```

**Discriminated unions.** An object type with a `kind` field can be one of several shapes. Inside a
`switch (x.kind)` the compiler narrows `x` to the matching shape, so `result.retryable` is only
readable in the `failed` arm:

`standalone_activity.ts:62-65`
```ts
export type AttemptResult =
  | { readonly kind: "completed" }
  | { readonly kind: "failed"; readonly retryable: boolean }
  | { readonly kind: "canceled" };
```

**Template literal types.** A string type can be built from other string types. Here a fact with a
payload stays a string, so equality and `includes` work without helpers:

`standalone_activity.ts:339`
```ts
  | `statusTimedOut:${TimeoutType}`
```

**Destructuring and spread.** `{ result }` in a parameter list pulls the field out of the argument
object; `{ ...state, phase }` copies an object with one field replaced:

`standalone_activity.ts:367-369`
```ts
function moves(state: ProtocolState, phase: Phase, recorded: readonly ProtocolFact[]): ProtocolStep[] {
  return [{ outcome: "accepted", state: { ...state, phase }, facts: recorded }];
}
```

**Generic builders.** A function like `defineMachine({...})` has type parameters that the compiler
infers from the object you pass. Because those parameters are marked `const`, the compiler keeps the
exact strings you wrote, and later fields are typed by earlier ones.

**`satisfies`.** `{...} satisfies ProtocolState` (line 589) checks a literal against a type
without widening the literal's own type.

## Vocabulary: entities, parties, actions, inputs

An **entity** is what a machine is about. The activity is named by the id the caller chose, so
line 52 declares `entity("activity", { key: "activityId" })`.

A **party** is who performs an action. This Model uses `caller` (starts and controls the activity)
and `worker` (runs attempts). `system` is reserved for timers, which a machine declares itself.

An **action** is a named side effect of a party. The caller's `start` creates the activity and
carries three finite inputs, the deadlines:

`standalone_activity.ts:85-91`
```ts
export const start = action({
  name: "start",
  party: "caller",
  creates: activity,
  schema: "temporal.api.workflowservice.v1.StartActivityExecutionRequest",
  input: { scheduleToClose: Timeout, scheduleToStart: Timeout, startToClose: Timeout },
});
```

`attemptStart` (lines 94-99) is the worker's poll receiving the task; it has no input.
`attemptResult` (lines 101-114) is the worker's answer, with input `{ result: AttemptResult }` and
two `examples` mapping the `failed` classes to a retryable or non-retryable `ApplicationFailure`:

`standalone_activity.ts:109-113`
```ts
  input: { result: AttemptResult },
  examples: [
    [{ result: { kind: "failed", retryable: false } }, "ApplicationFailureNonRetryable"],
    [{ result: { kind: "failed", retryable: true } }, "ApplicationFailureRetryable"],
  ],
```

The four caller controls (lines 117-128) are one action, `control`, with input
`{ control: Control }` and `results: Delivery`, because they share one behavior: a control on a
finished activity is `notFound`.

An **action class** is one action with one assignment of its inputs. `attemptResult` has four
classes because `AttemptResult` has four members: `completed`, `failed(false)`, `failed(true)`,
`canceled`. `control` has four classes, one per `Control` member. `start` has eight, one per
assignment of three two-valued deadlines. Classes are what Scenarios list and what `when:` names.

TypeScript cannot enumerate a union type at runtime, so the classes of a payload-carrying domain are
listed by hand and the compiler checks the list against the type:

`standalone_activity.ts:67-72`
```ts
export const AttemptResult = domain<AttemptResult>()([
  { kind: "completed" },
  { kind: "failed", retryable: false },
  { kind: "failed", retryable: true },
  { kind: "canceled" },
]);
```

The compiler rejects a member that is not an `AttemptResult` and a list missing a `kind`; it cannot
tell that both `retryable` values are present. That is a runtime pin.

A **fault** is just an action of a declared party. `workerStop` is the worker's, names no entity,
and is re-exported from the worker module at line 132 rather than declared twice. Nothing recorded
names the activity, so the machines keep their state and record nothing at it.

An **observation** is a derived read with no history event. Line 139 declares `attemptCount`, read
from the `attempt` field of `DescribeActivityExecution`.

## State

There are two machines and therefore two state types. The product state is just a phase:

`standalone_activity.ts:147-161`
```ts
export const ProductPhase = enumOf([
  "scheduled",
  "started",
  "paused",
  "cancelRequested",
  "completed",
  "failed",
  "canceled",
  "terminated",
  "timedOut",
]);
export type ProductPhase = Member<typeof ProductPhase>;

export const ProductState = struct({ phase: ProductPhase });
export type ProductState = Member<typeof ProductState>;
```

The protocol state adds the attempt count and the three deadlines:

`standalone_activity.ts:317-326`
```ts
export const attemptBound = 2;

export const ProtocolState = struct({
  phase: Phase,
  attempts: upTo(attemptBound),
  scheduleToClose: Timeout,
  scheduleToStart: Timeout,
  startToClose: Timeout,
});
export type ProtocolState = Member<typeof ProtocolState>;
```

Every field is finite so the framework can list every state and every row. The mechanism is
ordinary runtime code, not reflection: `struct` takes a record of finite fields and builds the
cartesian product in field order, and the state type is read back from the value.

`umpire.ts:82-92`
```ts
export function struct<const Fields extends Record<string, Finite<unknown>>>(
  fields: Fields,
): Finite<{ readonly [K in keyof Fields]: Member<Fields[K]> }> {
  type S = { readonly [K in keyof Fields]: Member<Fields[K]> };
  const names = Object.keys(fields) as (keyof Fields & string)[];
  const members = names.reduce<Record<string, unknown>[]>(
    (acc, name) => acc.flatMap((partial) => fields[name].members.map((v) => ({ ...partial, [name]: v }))),
    [{}],
  ) as S[];
  return { members, key: (s) => names.map((n) => fields[n].key(s[n])).join("-") };
}
```

So `ProductState.members` has 9 entries and `ProtocolState.members` has 12 × 3 × 2 × 2 × 2 = 288.
Each member also has a canonical key, `started-1-unset-unset-unset` style, which is how two states
are compared, since `===` on objects in JavaScript is identity.

The attempt count is bounded at `attemptBound` and moves with `saturatingSucc` (`umpire.ts:73-75`),
which returns `Math.min(n + 1, bound)`. Arithmetic in TypeScript widens to `number`, so the helper
carries a cast; the enumeration is what keeps the bound honest. `UpTo<2>` is the type `0 | 1 | 2`,
computed by a small type-level recursion in `umpire.ts:59-62`.

## Step functions

A step function has the shape `(state, inputs) -> list of steps`, where each step is an outcome, a
next state and the facts recorded. An empty list means the action is not enabled from that state.

`umpire.ts:201-207`
```ts
export interface Step<S, O, F> {
  readonly outcome: O;
  readonly state: S;
  readonly facts: readonly F[];
}

export type StepFn<S, O, F, In> = (state: S, input: In) => readonly Step<S, O, F>[];
```

The product's answer to an attempt, arm by arm:

`standalone_activity.ts:206-221`
```ts
export function attemptResultStep(state: ProductState, { result }: Inputs<typeof attemptResult.input>): ProductStep[] {
  if (state.phase !== "started" && state.phase !== "cancelRequested") return [];
  switch (result.kind) {
    case "completed":
      return productStep("completed", "statusCompleted");
    case "failed":
      if (!result.retryable) return productStep("failed", "statusFailed");
      return state.phase === "cancelRequested"
        ? productStep("canceled", "statusCanceled")
        : productStep("scheduled", "statusScheduled");
    case "canceled":
      return state.phase === "cancelRequested" ? productStep("canceled", "statusCanceled") : [];
    default:
      return assertNever(result);
  }
}
```

The guard says an answer only lands while a worker holds the attempt, which the product sees as
`started` or `cancelRequested`. `completed` settles the activity. A non-retryable failure settles
it as failed. A retryable failure is visible, unlike in the Nexus Model: under a cancel request it
settles as canceled, otherwise Describe reads `SCHEDULED` again, matching CHASM's
`TransitionRescheduled`. A `canceled` answer is honored only when cancellation was requested.

The protocol version dispatches on the phase as well as the result, because the protocol has three
phases in which a worker holds the attempt and they react differently to a retryable failure:

`standalone_activity.ts:403-426`
```ts
export function protocolAttemptResultStep(
  state: ProtocolState,
  { result }: Inputs<typeof attemptResult.input>,
): ProtocolStep[] {
  if (!attemptHeld(state.phase)) return [];
  switch (result.kind) {
    case "completed":
      return moves(state, "completed", ["statusCompleted"]);
    case "failed":
      if (!result.retryable) return moves(state, "failed", ["statusFailed"]);
      switch (state.phase) {
        case "cancelRequested":
          return moves(state, "canceled", ["statusCanceled"]);
        case "pauseRequested":
          return moves(state, "paused", ["statusPaused"]);
        default:
          return moves(state, "backingOff", ["attemptCount"]);
      }
    case "canceled":
      return state.phase === "cancelRequested" ? moves(state, "canceled", ["statusCanceled"]) : [];
    default:
      return assertNever(result);
  }
}
```

From `started` a retryable failure goes to `backingOff` and records only the attempt count, since
the caller sees nothing else until the backoff timer reschedules. From `cancelRequested` it settles
as canceled. From `pauseRequested` it lands in `paused`, mirroring CHASM's
`TransitionAttemptFailedWhilePauseRequested`.

Exhaustiveness is enforced by the compiler through `assertNever` (`umpire.ts:103-105`), a function
whose parameter has type `never`. Only a fully-narrowed `default` branch can supply that. Remove
the `canceled` case and `tsc` reports error TS2345 at the `assertNever(result)` call, because
`result` still has a possible shape. The inner `switch (state.phase)` uses `default` for `started`, so it is not exhaustive by design.

## The machine and its table

A machine ties the state, the actions and the step functions together:

`standalone_activity.ts:516-525`
```ts
export const activityProtocol = defineMachine({
  name: "activityProtocol",
  for: activity,
  state: ProtocolState,
  refines: { machine: activityProduct, map: productOf },
  starts: ["unstarted"],
  ends: ["completed", "failed", "canceled", "terminated", "timedOut"],
  actions: [start, attemptStart, attemptResult, control, workerStop],
  timers: ["backoff", "scheduleToClose", "scheduleToStart", "startToClose"],
  unobservable: ["backoff"],
```

The declaration continues (lines 526-549) with an `evidence` block mapping each of the ten fact
names to an observation name, and a `steps` block with one step function per action and per timer,
`start: startStep` through `startToClose: startToCloseStep`.

`starts` and `ends` are phases. `timers` are actions of the reserved `system` party that the machine
owns; `unobservable` names the timers that record nothing. `steps` must have exactly one entry per
action and per timer. The compiler enforces this through the `StepsOf` type (`umpire.ts:220-222`),
which builds an object type keyed by the action names in `actions` and the strings in `timers`. An
extra key such as `awaitFinish` is error TS2353; a missing `backoff` is TS2741.

The finite table is derived at load time, on first read, by running every step function on every
state for every action class. This body is sketched, not written:

`umpire.ts:363-371`
```ts
function withTables<M extends AnyMachine>(machine: Omit<M, "table" | "actionKeys" | "reachable" | "endStates">): M {
  // elided: define `table`, `actionKeys`, `reachable` and `endStates` as memoized getters.
  //   actionKeys = [...machine.actions.flatMap(classesOf).map(c => c.key), ...machine.timers].sort()
  //   table.rows  = for each state in machine.state.members, for each class: steps[name](state, input)
  //                 -> one Row per returned Step, keyed by state.key + "-" + class.key
  //   reachable   = BFS from the start states over table.rows
  //   endStates   = machine.state.members.filter(s => machine.ends.some(p => machine.at(s, p)))
  return machine as M;
}
```

Written out, it would call each step function 288 × 20 times for the protocol machine (288 states,
8 + 1 + 4 + 4 + 1 action classes plus 4 timers, 22 in all) and keep one row per returned step.
The pins then count states, ends and rows under vitest.

The `evidence` block maps each fact name to what a recorded run shows. For a system with history
events those would be event names. A standalone activity writes none, so every line here names an
observation: a status read through Describe, or the attempt count. The referee resolves each fact
in a step's `facts` list to one such read.

## Two levels and the refinement

The product machine says what an activity does as the caller sees it. The protocol machine says how
the server gets there: the `backingOff` phase between a retryable failure and the next poll, the
`pauseRequested` phase while a worker still holds a paused attempt, and the deadline fields. Both
are written against the same actions. A Property proved on the product is meant to carry over to
the protocol, and the refinement is what makes that legitimate.

`productOf` says how a protocol state reads as a product state:

`standalone_activity.ts:488-498`
```ts
export function productOf(state: ProtocolState): ProductState {
  switch (state.phase) {
    case "unstarted":
    case "scheduled":
    case "backingOff":
      return { phase: "scheduled" };
    case "started":
    case "pauseRequested":
      return { phase: "started" };
    case "paused":
      return { phase: "paused" };
```

The remaining arms map each phase to its namesake. Two arms matter. `unstarted` and `backingOff`
read as `scheduled` because the product begins there and cannot see a retry. `pauseRequested` reads
as `started`, not `paused`: the worker still holds the attempt, so every answer it can give is a
product row from `started`, and the request itself is a stutter. The first draft of this Model
mapped `pauseRequested` to `paused` and the refinement did not hold; the spec's revision note
records the fix.

The rule, as this sample implements it: for every protocol row from `s` to `s'`, either
`productOf(s)` equals `productOf(s')` (a stutter, the product saw nothing), or the product table has
some row from `productOf(s)` to `productOf(s')` under any action class. The Lean checker is
stricter: the matching product row must also have the same outcome and its facts must be among the
protocol row's facts, compared by evidence name. Under that rule one protocol row here would need
one more fact (`failed(true)` from `started` would record `statusScheduled` as well as
`attemptCount`); this sample implements the mapped-states rule as the spec states it.

The check is `checkRefinement` (`umpire.ts:501-506`), a runtime walk over both tables run under
vitest. Its body is sketched in a comment: for each protocol row, map both ends; equal means
stutter; otherwise look for any product row between the mapped states; none means rejected. The
match is not by action name, so a protocol timer row maps to the product's `timeout` row.

One row by hand. Take the protocol state `started`, attempts 1, all deadlines unset, and the class
`attemptResult-failed-true`. The protocol step function returns one step to `backingOff`, attempts
1, facts `["attemptCount"]`. Map both ends: `productOf(started) = {started}` and
`productOf(backingOff) = {scheduled}`. They differ, so this is not a stutter. Does the product have
a row from `started` to `scheduled`? Yes: `attemptResultStep({phase: "started"}, failed(true))`
returns `scheduled` with `statusScheduled`. The row passes. Under the first draft, where the product
returned `[]` for a retryable failure, there was no such row and the check rejected it.

## Properties

A **same-step claim** names an action class under `when` and asserts something about the step that
action produces. `completes` says a completed answer lands in `completed` and records it:

`standalone_activity.ts:568-573`
```ts
export const completes = property({
  name: "completes",
  machine: activityProtocol,
  when: attemptResult.of({ result: { kind: "completed" } }),
  holds: (step) => step.state.phase === "completed" && step.facts.includes("statusCompleted"),
});
```

`retryCompletes` fixes the whole state, so it can only be satisfied on the second attempt. State
equality goes through the canonical key with `same`, because `===` would compare object identity:

`standalone_activity.ts:591-596`
```ts
export const retryCompletes = property({
  name: "retryCompletes",
  machine: activityProtocol,
  when: attemptResult.of({ result: { kind: "completed" } }),
  holds: (step) => same(ProtocolState, step.state, completedOnRetry) && step.facts.includes("statusCompleted"),
});
```

A **transition claim** has no `when` and sees two consecutive steps. Both product claims here are
transition claims. `terminalIsFinal` (lines 555-559) says a terminal phase never changes;
`pausedIsNotDispatched` says nothing moves a paused activity straight to started:

`standalone_activity.ts:562-566`
```ts
export const pausedIsNotDispatched = property({
  name: "pausedIsNotDispatched",
  machine: activityProduct,
  holds: (before, after) => before.state.phase !== "paused" || after.state.phase !== "started",
});
```

The two shapes are two overloads of `property` in `umpire.ts:403-414`: with `when`, `holds` takes
one step; without it, two. The compiler picks the overload from the presence of `when`, so the
arrow function's parameters are typed without annotation.

## Scenarios and limits

A Scenario is a path: a start phase and a list of classed actions in order. An action with input is
spelled `action.of({...})`; an action without input, or a timer, is spelled by its bare name.

`standalone_activity.ts:653-665`
```ts
export const retriedThenCompleted = scenario({
  name: "retriedThenCompleted",
  model: activityProtocol,
  starts: "unstarted",
  actions: [
    noDeadline,
    "attemptStart",
    attemptResult.of({ result: { kind: "failed", retryable: true } }),
    "backoff",
    "attemptStart",
    attemptResult.of({ result: { kind: "completed" } }),
  ],
});
```

`noDeadline` is `start.of({...})` with all three deadlines unset, hoisted at line 637 because six
scenarios begin with it. The backoff timer appears in the path like any action; it records nothing.
`pausedThenCompleted` (lines 687-698) has the same shape: `noDeadline`, then
`control.of({ control: "pause" })`, `control.of({ control: "unpause" })`, `"attemptStart"` and a
completed answer.

Both `starts` and every element of `actions` are checked against the machine named in `model`: a
phase of another machine or an action it does not declare is a compile error.

Limits bound the search: how many steps a candidate may have, how many distinct actions, and how
many candidates to examine before giving up.

`standalone_activity.ts:722-724`
```ts
export const three = limits({ name: "three", steps: 3, actions: 3, search: 4096 });
export const four = limits({ name: "four", steps: 4, actions: 4, search: 32768 });
export const six = limits({ name: "six", steps: 6, actions: 6, search: 262144 });
```

## Queries

A Query pairs a Property with a Scenario under a limit. `find` asks for one trace of the path on
which the same-step claim holds; that trace becomes a test. `verify` asks that a claim hold on every
trace of the path; nothing is realized from it.

`standalone_activity.ts:735`
```ts
export const retry = query({ name: "retry", find: retryCompletes, in: retriedThenCompleted, limits: six });
```

`standalone_activity.ts:752`
```ts
export const pauseHolds = query({ name: "pauseHolds", verify: pausedIsNotDispatched, in: pausedThenCompleted, limits: six });
```

`pauseHolds` verifies a product Property on a protocol Scenario. The compiler allows it because a
Query's Property may belong to the machine or to the machine it refines, and the product Property is
read through `productOf` at each step.

The search is `run` (`umpire.ts:479-487`), a bounded breadth-first walk over the table from the
Scenario's start states. A candidate is a row sequence whose action classes are the Scenario's in
order, within the limits. `find` stops at the first candidate whose triggering step satisfies
`holds`; `verify` walks every candidate and every adjacent pair of rows; running over the budget is
`exhausted`. The body is sketched, and as shipped returns `exhausted` for everything.

When written, a failed `find` shows up as a vitest assertion with the result object in the diff:

```
FAIL pins.test.ts > standalone activity: the Queries > retry finds its claim on its path
AssertionError: expected { outcome: 'notFound' } to match object { outcome: 'found' }
```

A failed `verify` returns `violated` with the counterexample trace in `trace`.

## Sets

A set groups Queries for one purpose and says which parties the test harness drives and which it
only observes. The functional set `standaloneActivityTests` (lines 759-773) binds
`{ caller: "driven", worker: "driven" }` and lists the eight find-Queries. There is no `repeat: "implementation"` line, unlike the Nexus Model: standalone activities exist
only under CHASM, so there is no implementation switch to repeat over.

`standalone_activity.ts:775-789`
```ts
export const standaloneActivityCanary = set({
  name: "standaloneActivityCanary",
  purpose: "canary",
  bind: { caller: "driven", worker: "observed" },
  queries: [completion, cancel],
});

export const standaloneActivityExploration = set({
  name: "standaloneActivityExploration",
  purpose: "exploratory",
  bind: { caller: "driven", worker: "driven" },
  machine: activityProtocol,
  cover: ["rows", "results", "classMembers"],
  budget: four,
});
```

A canary runs against a deployment that supplies its own worker, so the worker is `observed`: the
referee reads which answer occurred and checks the machine allows it. An exploratory set names a
machine and coverage targets instead of Queries; it generates paths within the budget to cover rows,
results and class members.

The three shapes are one discriminated union on `purpose` in `umpire.ts:520-533`, so a functional
set cannot carry `cover` and an exploratory one cannot carry `queries`.

## Composition with the worker

The protocol machine's `workerStop` is a stutter: it keeps the state and records nothing, because
the activity cannot see its worker. To state anything about the worker, the Model composes with a
worker machine `polling` from `worker.ts`. It has two phases, `polling` and `stopped`, and three
actions, `workerStop`, `workerResume` and `serve` (lines 33-35). A stopped worker serves nothing:

`worker.ts:52-55`
```ts
export function serveStep(state: WorkerState): WorkerStep[] {
  if (state.phase !== "polling") return [];
  return [{ outcome: "accepted", state, facts: [] }];
}
```

The composition uses a restriction of the worker to stop and serve. It never resumes: an action no
`sync` line names would stay executable on its own and admit a stop, a resume and then an attempt.

`standalone_activity.ts:801-812`
```ts
export const activityWorker = restrict(polling, ["workerStop", "serve"]);

export const standaloneActivity = compose({
  name: "standaloneActivity",
  members: { activity: activityProtocol, worker: activityWorker },
  sync: {
    workerStop: ["activity.workerStop", "worker.workerStop"],
    attemptStart: ["activity.attemptStart", "worker.serve"],
  },
  starts: ["activity.unstarted", "worker.polling"],
  ends: ["activity.completed", "activity.failed", "activity.canceled", "activity.terminated", "activity.timedOut"],
});
```

Each `sync` line makes two member actions fire as one. The composed `attemptStart` has a row only
when both members have one, and a stopped worker has no `serve` row, so no attempt starts while the
worker is stopped. The strings `activity.unstarted` and `worker.serve` are template-literal types
built from the member names, so a typo is a compile error. The composed state is one field per
member, `{ activity: ProtocolState; worker: WorkerState }`.

`standalone_activity.ts:816-821`
```ts
export const startedByPollingWorker = property({
  name: "startedByPollingWorker",
  machine: standaloneActivity,
  when: "attemptStart",
  holds: (step) => step.state.worker.phase === "polling",
});
```

`when: "attemptStart"` with no class means any class. The Scenario `stoppedBeforeRetry`
(lines 823-836) starts the activity with a schedule-to-start deadline, lets a first attempt start
and fail retryably while the worker polls, then stops the worker during the backoff so the retry is
never dispatched and the deadline fires. A synced action is spelled by its bare name
(`"attemptStart"`, `"workerStop"`); an unsynced member action is spelled
`member("activity", attemptResult.of({...}))` and a member timer `"activity.backoff"`. The Query
`stoppedWorkerStartsNothing` (lines 838-843) verifies the claim over every trace of that path under
limits `six`. An earlier version of this path never performed `attemptStart`, so the claim held
vacuously; the path now exercises it once, with the worker polling.

What this proves: in the composed table, every `attemptStart` row leaves the worker polling. That
is the cross-entity fact the protocol machine's stutter row could not express on its own. The
composition body in `umpire.ts:600-611` is sketched.

## Pins

The pins are vitest tests over the tables. Three of them:

`pins.test.ts:130-133`
```ts
  it("protocol: 12 * 3 * 8 states, 5 * 3 * 8 ends", () => {
    expect(activityProtocol.table.states).toHaveLength(12 * (attemptBound + 1) * 8);
    expect(activityProtocol.endStates).toHaveLength(5 * (attemptBound + 1) * 8);
  });
```

This guards the state space itself. Add a phase or a field and the count changes; the Behavior
Fingerprint reads this table, so the number is part of the Model's identity.

`pins.test.ts:135-136`
```ts
  it("honors a canceled answer only under a cancel request, on both machines", () => {
    expect(attemptResultStep({ phase: "started" }, { result: { kind: "canceled" } })).toEqual([]);
```

This guards one enablement decision that is easy to get wrong when editing the `canceled` arm: a
worker cannot cancel an activity nobody asked to cancel. The same test repeats the check on the
protocol step function.

`pins.test.ts:147-151`
```ts
  it("refines the product machine: every row is a product stutter or maps to some product row", () => {
    // Holds under the revised spec: `pauseRequested` reads as started and a retryable failure is
    // a visible product row (started -> scheduled, cancelRequested -> canceled).
    expect(checkRefinement(activityProtocol).rejected).toBeNull();
  });
```

This is the check that failed on the first draft of the Model and would fail again if someone
changed `productOf` or removed a product row without thinking about the protocol.

## From model to running test

After this file, a Query that finds its trace is lowered to a Case: the trace's actions become the
steps a harness performs, the facts along the trace become the evidence a referee checks, and the
`bind` block says which party the harness plays. A realization maps each action class to concrete
calls (`StartActivityExecution`, a poll, `RespondActivityTaskFailed` with a retryable
`ApplicationFailure`) and each fact to a concrete read. No realization exists yet for standalone
activities. The Go Testpilot runtime executes Cases against a server and issues verdicts: pass,
fail with the first fact that did not match, or a Known Gap where a step recorded nothing. None of
that layer is in this sample; `standalone_activity.ts:791-794` marks where the Case declarations
would go.

## Gaps and gradual growth

The Model says "not modeled" in several places, on purpose. Reset is deferred. The heartbeat
timeout is not a timer here. `workerStop` on the protocol machine is a stutter row that records
nothing; on a path it is confirmed by the evidence of the step after it, and a Case carries a Known
Gap for it. The product's `workerStopStep` returns `[]` outright, because a product step that kept
the state and recorded nothing would be indistinguishable from every other stutter. An empty step
list is how any action says "not enabled here", so adding an action is safe by default: it does
nothing until a step function says otherwise.

That is what makes growth gradual. To add a heartbeat timeout: add `"heartbeat"` to `timers`,
write a step function with the phases it fires in, and add `` `statusTimedOut:heartbeat` `` to the
fact union if a new payload is needed. The compiler then demands the `steps` entry; the pins
demand the new counts; the refinement check demands that the product has a row for what the new
timer does. Nothing else needs to change.

## Mental model recap

- A Model is a rulebook. Actions are moves, step functions say what each move does from each state,
  and an empty list means the move is not available.
- Everything is finite: enums are `as const` lists, states are products of them, and the framework
  enumerates all 9 product and 288 protocol states from those lists at load time.
- Two machines: the product says what the caller sees, the protocol says how the server gets there.
  `productOf` maps protocol states down; the refinement check confirms every protocol row is a
  product stutter or a product row.
- Facts are what a step leaves behind. Here they are all status reads, because standalone activities
  write no history.
- A Property is a claim: same-step with `when`, transition without. A Scenario is a path. A Query
  pairs them, either to find a test or to verify a claim.
- Sets group Queries for a purpose and say who is driven and who is observed.
- Composition with the worker turns the stutter row into a real phase change, so cross-entity
  claims become checkable.
- The compiler checks shape (names, phases, exhaustiveness); vitest checks meaning (tables,
  refinement, search).

## Where this implementation is weak

- **It was never compiled.** The independent review found that `checkRefinement` could not accept
  the protocol machine because `Refinement.map` was a function-typed property, which is
  contravariant in its parameter under `strictFunctionTypes`. That has since been changed to a
  method signature (`umpire.ts:227`), the same fix `Finite.key` already used, but the file has still
  not been through `tsc`.
- **Outcome and fact inference is probably weaker than described.** `defineMachine` claims to
  infer `O` and `F` from the step functions' return types through `StepsOf`, a key-remapped mapped
  type. The review's assessment is that TypeScript does not infer through that shape, so `O` and
  `F` likely fall back to `string`, the `evidence` keys go unchecked, and `step.facts` in a
  `holds` function is `string[]`. The Models still compile, but the compile-time story in the README
  is narrower than written.
- **`run` returns a hard-coded `exhausted`** and `checkRefinement` a hard-coded pass. Every Query
  pin and both refinement pins would fail or pass vacuously as shipped. The algorithms are described
  in comments, not implemented.
- **The refinement route for product Properties is fragile.** `terminalHolds` and `pauseHolds` rely
  on `RefinedBy<M>` inferring the product state from an intersection of `refines?: Refinement<S,
  unknown>` and `refines: Refinement<S, S2>`; the review judged that inference may resolve to
  `unknown` and reject those Queries rather than allow them.
