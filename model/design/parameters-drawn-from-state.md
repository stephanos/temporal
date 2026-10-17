# Idea: parameters drawn from state, an Event-B `any … where` for internal steps

Status: idea, 2026-10-17. No spec, no source change. Recorded from the owner's review of a notation
survey (FizzBee, P, statecharts, Alloy, Event-B, Ivy, UPPAAL); the sibling idea from the same review,
correlated obligations, became fn-197.

## What Event-B writes

```
event Dispatch:
    any task, worker
    where task in queued, worker in available
    then queued.remove(task); available.remove(worker); assigned.add(task, worker)
```

The parameters are not inputs anybody supplies. The system picks any eligible pair from its own
state, and the model allows every pairing without naming a selection algorithm.

## What Umpire writes today

The same step is an action with inputs from a fixed catalog and a guard that filters:

```scala
final case class Dispatch(task: Task, worker: Worker)
object scheduler extends Actor:
  val dispatch = action.inputs[Dispatch]

on(scheduler.dispatch): in =>
  if queued.contains(in.task) && available.contains(in.worker) then
    effects.dispatch(in)
```

For a finite model the two are equivalent: the checker enumerates every task and worker, and the
guard disables the pairs that are not eligible. The difference is in what the enumeration means:

- Every pair is an action class. The tables and the lint see classes that are disabled in most
  states and report them as disabled rows; a Case can drive them; a `fair` names them.
- The guard repeats the eligibility test the effect then relies on, once per such step.
- An `Actor` takes it, where the dispatch is the system's own choice, which `Internal` is for. An
  `Internal` step has no input today, so it cannot pick.

## The form that would say it

A nondeterministic pick inside an internal step, with no input and no catalog:

```scala
object scheduler extends Internal:
  val dispatch = step

on(scheduler.dispatch):
  for task <- any(queued); worker <- any(available) do
    effects.dispatch(task, worker)
```

The checker branches once per eligible pair in the current state, as `choose` branches once per
named alternative. `choose` is the nearest thing today, but its alternatives are named and fixed at
declaration, not drawn from a collection; `any(collection)` would be a `choose` whose arms are the
collection's elements, unnamed because the element is the name. An empty collection is a disabled
step, not a hole.

## For and against

For:

- Dispatch, matching and task delivery are the system's choices, and the activity and task-queue
  models spell them as actor inputs with guards that restate eligibility.
- The disabled-row noise goes: there is no row for a pair that was never eligible.
- `fair(scheduler.dispatch)` reads as "the scheduler keeps dispatching", not as fairness over a
  catalog of pairs most of which are never enabled.

Against:

- Inputs are the one mechanism for nondeterminism, and the disabled rows are honest about what the
  environment could try; a disabled pair is a prohibition the table shows (SEMANTICS §6, modalities).
- Nothing drives it from a Case: an `Internal` step is observed, not performed, so the realization
  has no handle on which pair the system picked. That is right for a system choice and wrong the
  moment a test wants to force a pairing.
- The lifter needs a new node (a bound variable over a collection expression) and the IR a
  quantifier it does not have; `choose` lifts as a list of arms.
- It is a second spelling for something expressible today, which the DSL's rule on words
  (`progress-notation.md`, "When a claim gets a word") resists unless it changes what is checked. It
  changes what is enumerated, not what is checked.

## When to pick it up

Not before a Temporal model shows the pain: when the task-queue foundation or the activity dispatch
is rewritten in the plain-Scala form (fn-190, fn-193) and its guards restate eligibility in more
than one step, or its disabled rows bury real findings in the lint. Then the question is a short
prototype under `model/design/` with the dispatch step written both ways, and a decision on whether
`any` is a `choose` over a collection (no IR change beyond the arms) or a new quantifier.
