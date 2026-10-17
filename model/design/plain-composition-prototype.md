# Plain-Scala Compositions (fn-193.1): prototype and verdict

Status: prototype with a verdict, 2026-10-17. Scala 3.9.0, JVM 27, the framework's strict flags.
No framework source changed. The probe is kept at
[`plain-composition-prototype/`](plain-composition-prototype/README.md) with a README and `run.sh`;
its logs are `plain-composition-prototype/logs/{compile,test,report}.{log,status}`.

Unlike the Kettle probe, this one brings no Machine of its own. Its members are the framework's
plain `Machine`s, and it compiles against `model/framework/dsl` and `realize`'s main sources. The
new code is the composition layer: about 740 lines of DSL, macros and native runtime in package
`framework.dsl.compose`. The package gives it `private[framework]` access to `Vars`, `NativeClaims`,
`NativeCheck` and `effectOf`, which is the only framework hook it needs, and needs no change to them.

## 0. Answer in brief

**The plain form works natively on the framework's composition fixtures, with no state case class,
no selectors and no `->`.** 12 tests pass (`logs/test.status`). The run prints each composition's
members, syncs, reachable graph with every composed step, its verdicts, its witness, a refuted
negative control and the run-time refusals (`logs/report.log`).

- **Members are vals naming plain Machines**, and the composed state is their vars, read through
  the member (`left.phase`, `left.isClosed`). A macro derives the member schema where the object
  extends `Composition[F]`, as `VarSchema` does for a Machine's vars.
- **A sync steps both members in one step.** From the start, `started` runs Subject's and Other's
  `start` effects and records `[left(completed), right(completed)]`. The runtime has no new state
  machinery: each member steps and restores through its own `Vars`, and a claim is evaluated with
  every member's state restored at once.
- **Three changes to the spec's direction:**
  - `own` and `synced` are not needed. An action names its composed step.
  - Member-qualified actions are `left(action)`, not `activity.worker.poll`.
  - A Machine object can be a member only once.
- **The derivation keeps the source's fact type without `DerivedComposition`**, because the fact
  type is a type parameter that Scala infers from `Together.withMember(right = LateOther)`.

**Verdict: adopt the form, with the changes in §2.**

## 1. Recommended syntax

```scala
enum JoinedFact:                          // one case per member that records facts (fn-190 R17)
  case left(value: Fact)
  case right(value: Foreign)

object Together extends Composition[JoinedFact]:   // Composition[Nothing] if no member records
  val left = Subject                      // members: vals naming Machines, in declaration order
  val right = Other

  override def end = left.end && right.phase == Phase.closed   // reads member vars, like a Machine's

  object syncs:
    val started = sync(left(hand.start), right(hand.start))    // both bind start: name the member
    // each action bound by one member: sync(activityWorker.poll, matching.deliver)

  object invariants:
    val rightClosedWithLeft =
      if left.isClosed then check(right.phase == Phase.closed) // member helper: an ordinary call

  object postconditions:
    val startClosesBoth =
      after(syncs.started):                                    // a sync is its val
        check(before.left.phase == Phase.open)                 // before: a record of member records
        records(JoinedFact.left(Fact.completed))               // a member fact in its case
    val finishMovesLeftAlone =
      after(hand.finish):                                      // an unpaired action: its own step
        check(right.phase == before.right.phase)

  object queries:
    val reopened = after(hand.finish):
      records(JoinedFact.left(Fact.reopened))
    val reopenedFound = find(reopened)

object LateTogether extends Composition(Together.withMember(right = LateOther)):
  object invariants:                      // own claims; the replacement shares Other's vars
    val rightStaysOpen = check(Together.right.phase == Phase.open)

// a test's negative control
assertRefuted(Together, invariant(check(Together.left.phase == Together.right.phase)))
```

## 2. Verdict per Architecture bullet

| Bullet | Verdict | Evidence / what Scala forces |
| --- | --- | --- |
| Members as `val`s naming Machines; composed state is their vars read through the member; no case class, selectors or `->` | **works, with one restriction** | `val left = Subject` has type `Subject.type`, so `left.phase` and `left.isClosed` type-check. The schema macro reads the object's vals of Machine type in declaration order (`Reordered` lists `right` first; facts keep their case). It refuses at compile time: a val naming another object (`clock names hand, which is neither a Machine nor a Composition`); a member that records facts with no case for it in `F` (`Foreign declares a case for it, case left(value: Fact)`) or a case of the wrong type. **The restriction:** a Machine object's vars hold one state, so a Machine can be a member only once. `Repeated` (Subject twice) is refused at compile time (`left names Subject, which right names already`). Two members that share vars through a derivation are refused when the runtime reads them (`left and right hold the same vars (subjectAgain's are subject's)`). A member that is a `Derived` has no vars of its own on its val, so a claim reads it through its source (`Other.phase`). |
| `syncs` pair member actions by reference, `sync(activity.worker.poll, queue.deliver)` | **works, with a change** | Actions live in `Actor`/`Internal` objects beside the Machines (fn-190 R16) and are often bound by more than one machine, so there is no `activity.worker.poll` path. Instead: (1) `sync(activityWorker.poll, matching.deliver)`, where each action is bound by exactly one member (the Queue fixture), resolved by the binding; (2) `sync(left(hand.start), right(hand.start))` where two members bind it. `left(action)` is an extension `apply` on a Machine. A sync is a `val` in `object syncs`, named after the val, so `sync("admit", …)`, `Syncs` and `Composer` go. The runtime refuses, with file, line and column: an ambiguous end (`Refused.test.scala:26:21: sync started: left and right bind start; name the member, left(start)`), and an action the member does not bind (`… right (other) binds no finish`). It also refuses an action that two syncs pair, a case the suite does not exercise. A sync of a non-action does not type-check (`Required: framework.dsl.compose.Endpoint`). |
| `own` and `synced` take the member's action directly | **does not work as stated → alternative: drop both** | Without selectors there is nothing left for them to do. A claim or witness names a composed step by: the sync's val (`after(syncs.started)`); a member action that no sync pairs, which is its own step (`after(hand.finish)` gives `left.finish`); a paired action, which is the sync's step (`after(activityWorker.poll)` is `admit`); or `left(action)` where two members bind it. `Composed`, `own`, `synced` and `ComposedReference` go. `only`/`fair` should take the same targets, but this was not built. |
| Claims in fn-190 sections with `check`; member helpers as ordinary calls; member facts `Fact.<member>(f)` | **works** | `invariants`, `postconditions` and `queries` are evaluated exactly as a Machine's are: the framework's `NativeClaims.evaluate` and section reload, unchanged, with every member's state restored. `left.isClosed` is a plain call reading Subject's vars. `records(JoinedFact.left(Fact.completed))` checks the tagged fact, and an unwrapped `records(Fact.completed)` is a type error. `before.left.phase` is a `transparent inline` record of the members' State records, refined with each member's vars: `before.left.phse` is a type error, and `before` in `invariants` is refused. `after`, `records`, `before` and `find` are composition-side copies, because the framework's `ClaimWords`/`QueryMacros` look for an enclosing `RecordMachine`. fn-193.2 should widen that lookup rather than copy the words. Progress (`leadsTo`) was not built (§6). |
| A derivation replaces one member, `withMember(activity = LateRecord)` or a `rebind` form; `DerivedComposition` goes if the fact type is kept | **works; `DerivedComposition` goes** | `object LateTogether extends Composition(Together.withMember(right = LateOther))`. `withMember` is a `Dynamic` named update, as `state.copy(heatingOn = true)` is. It keeps the source's syncs, end and member order, and `summon[LateTogether.type <:< Composition[JoinedFact]]` compiles. This needs the fact type as a **type parameter**: a `type Fact` member cannot be inferred from a constructor argument, which is why `DerivedComposition` exists today. **The replacement must hold the replaced member's vars**, so it is that Machine or a derivation of it. Otherwise it is refused: `subject does not hold other's vars; a replacement is other or a derivation of it, so the claims that read right read it`. This rule takes the place of the lifter's "same state type" rule. It makes the source's claims correct on the derivation: checked on `LateTogether`, `Together.rightClosedWithLeft` is refuted at `started`, because LateOther stays open. A misspelt member is refused (`rigth is not a member of together`). |
| Native runtime: product of var snapshots; a sync runs both effects in one step; R20 checks run over it | **works; the explorer is new** | `Composed(Vector[State])`. An own step runs one member's binding from its State. A sync runs both members' bindings and takes the product of their results, with the first member's outcome and both members' facts tagged. Claims run under nested `Vars.at`. Results: Together has 3 states, all three claims verified and the witness `started, left.finish`; Queue has 6 states, its postcondition verified and the witness `queue.schedule, queue.schedule, admit`; `assertRefuted(Together, invariant(…))` gives `counterexample: started, left.finish`. Vacuity is reported: Together's `startClosesBoth` checked on `Apart` warns `no step takes its after target`. `NativeCheck.Search` is typed to a `RecordMachine`, so the probe has its own breadth-first search (§5). |

## 3. What of today's Composition machinery goes

- **`Composition[S <: Product]`, its `Mirror`, `Data[S]`, `StateData[S]`, `Composition.Empty`,
  `retain`/`selected` and `Shape.Members`**: the state is the members' vars, each of which already
  has its `Data`.
- **`MemberSelection`, `MemberSelector`** (`Retained`, `retainDirect`, `retainMove`, the
  direct-field check) and the `_.field -> Machine` pairing.
- **`Syncs`, `Syncs.Statement`, `Syncs.declaredBy` and `Composer`**, replaced by vals in
  `object syncs`.
- **`Composed`, `own`, `synced`, `Move` and `ComposedReference`**: targets are actions, member steps
  and sync vals. `Assume.fair(Composed…)` and the `Composed` arm of `only`/`actions` go with them.
- **`DerivedComposition`, `DerivedFrom` on compositions, `Shape.Of`/`Shape.With` and
  `withMember(selector)`**, replaced by `Composition(derivation)` and `withMember(name = machine)`.
- **`Owner`/`Declaring`**: the probe needs neither. `Declaring` served only the `capabilities`
  section and `composedOutcome`, which are open (§6).
- **The record-state `property`/`scenario`/`query` surface** (`Declares[S]`): not used by any
  composition.

What the plain form adds: `sync`, `left(action)`, `withMember(name = …)`, and the composition's
reading of `after`/`records`/`before`/`find`. These replace about 480 lines (`Compose.scala`,
`ComposedReference.scala`, `MemberSelector.scala`, `MemberSelection.scala`).

## 4. Fixture changes this forces (for fn-193.2)

- `Expect.test.scala` `Repeated` (Subject twice) cannot be written. Its only assertion, that tagged
  facts survive member order, is `Reordered`'s, so delete it, along with the export testdata
  `witness/Witnesses.scala` `Repeated`.
- `Ends.test.scala` `PausedPair(ClosedRecord, ClosedRecord)` needs two distinct Machines. Its
  derivation replaces `right` with `PausedEnd`, which must then derive from the replaced Machine.
- `NativeData.test.scala` `Product(Native, Copied)`: `Copied` is a derivation of `Native`, so the
  two share vars and the composition is refused. Use a second Machine.
- `Rules.test.scala` `Odd = Twins.withMember(right -> Stuck)`: Stuck derives from Switch (left) and
  replaces Unfelt, so it is refused. Replace `left` instead.
- `IrFile.test.scala` `JoinedReplacement = JoinedSource.withMember(value = Original)` keeps working,
  since `Added` derives from `Original`.

## 5. Native runtime for fn-193.2

- **Members and schema at compile time:** the probe's `Schema` macro, given in the
  `Composition[F]` constructor. It emits the member names and a constructor call for each member's
  fact case, and refuses objects that are not Machines, repeated objects and missing or mistyped
  fact cases. The members' objects are read back through their getters by name, as `Vars` reads its
  accessors.
- **State and steps:** a composed state is the vector of member `State`s. Its starts are the
  product of each member's `initialStates`. An own step is `effectOf(binding)(state(i), cls)`. A
  sync is the product of both members' results over the product of their classes, with the first
  member's outcome and the tagged facts of both. A paired action never steps alone.
- **Claims:** restore every member with nested `Vars.at`, then `NativeClaims.evaluate` on the
  section, unchanged. A step frame carries the composed start (`before`) and what was taken
  (`after`). Shared vars between members are refused at construction.
- **Search:** do not copy the probe's breadth-first search. Generalise `NativeCheck.Search` over a
  successor function and a state type, so that compositions get R20's on-demand search, budgets,
  progress, monitors and pinned paths unchanged. Today `Search` reaches into
  `RecordMachine.bindings` and channels. The alternative, a synthetic `RecordMachine[Composed, …]`
  with one binding per own step and per sync, needs an `ActionDecl` for each sync and drags in the
  record surface that fn-190.16 deletes.
- **Compiler hazard:** an inline method that calls a `@publicInBinary` private constructor, used in
  the same compilation run, loops forever in Scala 3.9.0's inliner (README). The samples compile in
  the same run as the DSL, so the composition words should call public factories.

## 6. What the Phase-3 lifter must do (with fn-190.12)

- Read the member vals (name, the object named, declaration order) as the IR composition's members.
- Lift member reads in claims and `end` (`left.phase`) to field paths of the composed state
  (`s.left.phase`). Lift `before.left.phase` to the transition's first parameter.
- Lift a member helper call (`left.isClosed`) as that member's lifted def applied to its field. A
  read through another object that holds the member's vars, `Together.right.phase` or `Other.phase`
  in a derivation, resolves to the member whose vars it reads.
- Lift sync vals to IR syncs named after the val. Resolve each end to (member, action) by the rule
  the runtime uses: by binding, or by `left(action)`, refusing ambiguity. Resolve `after`/`only`/
  `fair` targets the same way to a sync or a member action. This replaces `ComposedReference` and
  the selector extraction in `export/Compositions.scala`.
- Lift `records(JoinedFact.left(f))` as the ordinary enum construction it is (R17). Go reads the
  tagged enum.
- Lift `withMember(right = X)` to a derived composition with that member replaced, checking that X
  is the replaced Machine or a derivation of it.
- Refuse a member var read outside a lifted scope, as fn-190.9 does for a Machine's vars.

## 7. Not proved here, and open questions for fn-193.2

- **Progress, monitors and scenarios over a composition** (`leadsTo … in only(syncs.admit)`,
  `fair(...)`, `starts`): not built. They follow from the generalised `Search` (§5) and the target
  resolution (§2).
- **A composition as a member** (nested compositions): the macro admits it, and the runtime refuses
  it (`declares no state vars`). Flattening the inner members is the obvious design.
- **The composed outcome**: the probe's step answers the stepping, or first, member's outcome
  untagged, so it needs no `O` parameter. Whether capabilities (`composedOutcome`) need a tagged
  outcome enum like the fact enum is open, as is whether compositions keep a `capabilities`
  section.
- **`replaces(_.queue, OpaqueProduct)`**, a member that stands in for an opaque machine, has no
  plain spelling yet. One candidate is `replaces(queue, OpaqueProduct)` in `object syncs`.
- **A derivation's own `before`**: the derived object declares no member vals, so its `before`
  record is empty in the probe. Either `withMember` returns a derivation that carries the source's
  type, so the macro can read its members, or a derived composition's own claims read the source
  (`Together.right.phase`), as in the probe. Should a derivation check its source's claims by
  default? The probe checks them on request (`sections = Some(Together)`), as a Machine's `Derived`
  does today.
- **Two members of one entity type** (the TwoPhaseCommit TODO's resource managers): the one-state
  rule refuses this. It would need a Machine declared once and instantiated per member, which is a
  new form and outside this spec.
- **Member ends**: the default `end` is "anywhere", as a Machine's is. The old composition required
  `end`. The other candidate default is that every member may end.

## 8. Rerun and recorded output

`bash "$PWD/model/design/plain-composition-prototype/run.sh"` from the repository root. Expected:
`compile: exit 0`, `test: exit 0, 12 passed, 0 failed`, `report: exit 0`. The main native suites
are unaffected: `mise exec -- scala-cli test --server=false model/framework/dsl
model/framework/realize model/framework/samples --require-tests` passes 637 tests, 0 failed, before
and after the probe. Excerpt of `logs/report.log`:

```text
== Together (Witness.test.scala): left and right both bind start, synced
members: left = subject { phase: Phase }; right = other { phase: Phase }
sync started: left.start with right.start
own steps: left.finish
reachable: 3 states, 2 steps
  0 (left = State(phase = open), right = State(phase = open))
     started -> 1  facts [left(completed), right(completed)]
  1 (left = State(phase = closed), right = State(phase = closed)) (may end)
     left.finish -> 2  facts [left(reopened)]
  2 (left = State(phase = open), right = State(phase = closed))
  together.invariants.rightClosedWithLeft: verified (budget 65536 states, 2 steps; exploration complete)
  together.postconditions.startClosesBoth: verified (budget 65536 states, 2 steps; exploration complete)
  together.postconditions.finishMovesLeftAlone: verified (budget 65536 states, 2 steps; exploration complete)
  together.reopenedFound: found (budget 65536 states, 2 steps; exploration complete)
    witness: started, left.finish
== Apart: no sync; Together's sections checked on it
  apart.invariants.rightClosedWithLeft: refuted (budget 65536 states, 2 steps; exploration complete)
    counterexample: left.start
  warning: apart.postconditions.startClosesBoth: no step takes its after target
== LateTogether: Together.withMember(right = LateOther)
members: left = subject { phase: Phase }; right = lateOther (in place of other) { phase: Phase }
     started -> 1  facts [left(completed), right(completed)]
  lateTogether.invariants.rightStaysOpen: verified (budget 65536 states, 1 steps; exploration complete)
   Together's sections checked on LateTogether:
  lateTogether.invariants.rightClosedWithLeft: refuted (budget 65536 states, 1 steps; exploration complete)
    counterexample: started
== Queue (SEMANTICS section 17's example, cut down): each action bound by one member
sync admit: activity.poll with queue.deliver
  queue.postconditions.deliveredWasScheduled: verified (budget 65536 states, 4 steps; exploration complete)
  queue.startedWithOneLeftFound: found (budget 65536 states, 4 steps; exploration complete)
    witness: queue.schedule, queue.schedule, admit
== negative control: invariant(check(left.phase == right.phase)) on Together
  together.sameSides: refuted (budget 65536 states, 2 steps; exploration complete)
    counterexample: started, left.finish
== refused when read
  Doubled: doubled: left and right hold the same vars (subjectAgain's are subject's): a Machine's vars hold one state, so each member holds vars of its own
  Ambiguous: Refused.test.scala:26:21: sync started: left and right bind start; name the member, left(start)
  Unbound: Refused.test.scala:34:22: sync finished: right (other) binds no finish
```
