// The declarations checks read over a Model, lifted with it: an internal step, a declared hole beside
// disabled rows, assumptions, a monitor read at each evaluation point, the opaque machine a detailed
// one refines under a visible projection, a progress claim, a composition in which the detailed
// machine replaces the opaque one, and Properties, Scenarios and Queries on machines and compositions.
// The tests lift `queries` and `durableEventually` and compare the IR with expected/declarations.json.
package fixture.declarations

import umpire.*

val Family: umpire.Family = umpire.Family("fixture.declarations")

enum Kept derives Finite:
  case nothing, held

final case class Store(kept: Kept) derives Finite

enum Outcome derives Finite:
  case accepted, deferred

enum Fact derives Finite:
  case stored, staged

val put = action("put", Party("client"))
val flush = internal("flush")
val crash = action("crash", Party("fault"))

val storeOpaque: Assumption = assume("storeOpaque")
val flushRuns: Assumption = assume("flushEventuallyRuns").fair(flush)
val crashUnmodeled: Hole = hole("crashUnmodeled")

def evidenceOf(f: Fact): String = f match
  case Fact.stored => "stored"
  case Fact.staged => "staged"

/** The opaque provider: all its clients may rely on. */
val store: Machine[Store, Outcome, Fact] =
  machine[Store, Outcome, Fact](Family, "store") {
    assumes(storeOpaque)
    starts(Store(Kept.nothing))
    ends(s => s.kept == Kept.held)
    evidence(evidenceOf)
    steps(
      put ~> (s =>
        if s.kept == Kept.nothing then
          List(Step(Outcome.accepted, Store(Kept.held), List(Fact.stored)))
        else Nil
      )
    )
  }

enum Stage derives Finite:
  case empty, staged, durable

final case class Disk(stage: Stage) derives Finite

def stored(d: Disk): Store =
  if d.stage == Stage.empty then Store(Kept.nothing) else Store(Kept.held)

def putStep(d: Disk): List[Step[Disk, Outcome, Fact]] = d.stage match
  case Stage.empty => List(Step(Outcome.accepted, Disk(Stage.staged), List(Fact.stored)))
  case Stage.staged | Stage.durable => Nil

/** A stutter of the store: staging becomes durable, and the store sees nothing of it. */
def flushStep(d: Disk): List[Step[Disk, Outcome, Fact]] = d.stage match
  case Stage.staged => List(Step(Outcome.deferred, Disk(Stage.durable), List(Fact.staged)))
  case _            => Nil

/** What a crash does to staged data is left unknown; elsewhere a crash is disabled. */
def crashStep(d: Disk): List[Step[Disk, Outcome, Fact]] =
  if d.stage == Stage.staged then crashUnmodeled.reached else Nil

enum Seen derives Finite:
  case never, once, twice

def countStored(seen: Seen, before: Disk, after: Step[Disk, Outcome, Fact]): Seen =
  if !after.facts.contains(Fact.stored) then seen
  else
    seen match
      case Seen.never             => Seen.once
      case Seen.once | Seen.twice => Seen.twice

val storedOnce: Monitor[Disk, Outcome, Fact, Seen] =
  monitor[Disk, Outcome, Fact, Seen]("storedOnce", Seen.never)(countStored)(seen =>
    seen == Seen.twice
  )

val endsDurable: Monitor[Disk, Outcome, Fact, Boolean] =
  monitor[Disk, Outcome, Fact, Boolean]("endsDurable", false)((_, _, after) =>
    after.state.stage == Stage.durable
  )(durable => !durable).readAtEnds

val stagedBeforeDurable: Monitor[Disk, Outcome, Fact, Boolean] =
  monitor[Disk, Outcome, Fact, Boolean]("stagedBeforeDurable", false)((seen, _, after) =>
    seen || after.facts.contains(Fact.staged)
  )(seen => !seen).readAfter(after => after.state.stage == Stage.durable)

/** The detailed provider. */
val disk: Machine[Disk, Outcome, Fact] =
  machine[Disk, Outcome, Fact](Family, "disk") {
    refines(store)(stored)
    visible(f => f == Fact.stored)
    visibleOutcomes(o => o == Outcome.accepted)
    monitors(storedOnce, endsDurable, stagedBeforeDurable)
    starts(Disk(Stage.empty))
    ends(d => d.stage != Stage.staged)
    evidence(evidenceOf)
    steps(put ~> putStep, flush ~> flushStep, crash ~> crashStep)
  }

val durableEventually: Progress[Disk] =
  disk.leadsTo("durableEventually")(
    d => d.stage == Stage.staged,
    d => d.stage == Stage.durable,
    within = 2,
    flushRuns
  )

final case class Pair(front: Store, back: Store)

final case class DetailedPair(front: Store, back: Disk)

val pair: Composition[Pair] =
  compose[Pair](Family, "pair")("front" -> store, "back" -> store)
    .sync("putBoth", "front" -> put, "back" -> put)
    .ends(p => p.front.kept == p.back.kept)

val detailedPair: Composition[DetailedPair] =
  compose[DetailedPair](Family, "detailedPair")("front" -> store, "back" -> disk)
    .sync("putBoth", "front" -> put, "back" -> put)
    .replaces("back", store)
    .ends(p => p.front.kept == Kept.held)

val durableStays: Property[Disk] = disk.property("durableStays") holdsAcross { (before, after) =>
  before.stage != Stage.durable || after.state.stage == Stage.durable
}
val putStores: Property[Store] =
  store.property("putStores") when put holds (after => after.facts.contains(Fact.stored))
val putAccepted: Property[Disk] =
  disk.property("putAccepted").whenAction("put") holds (after => after.outcome == Outcome.accepted)
val keptTogether: Property[Pair] =
  pair.property("keptTogether") holds (after => after.state.front.kept == after.state.back.kept)
val frontHeld: Property[DetailedPair] =
  detailedPair.property("frontHeld") holds (after => after.state.front.kept == Kept.held)

val putThenFlush: Scenario[Disk] =
  disk.scenario("putThenFlush").starts(Disk(Stage.empty)).actions(put, flush)
val anyDisk: Scenario[Disk] = disk.scenario("any").starts(Disk(Stage.empty)).free
val putOnce: Scenario[Store] = store.scenario("putOnce").starts(Store(Kept.nothing)).actions(put)
val anyPair: Scenario[Pair] =
  pair.scenario("any").starts(Pair(Store(Kept.nothing), Store(Kept.nothing))).free
val bothPut: Scenario[DetailedPair] =
  detailedPair
    .scenario("bothPut")
    .starts(DetailedPair(Store(Kept.nothing), Disk(Stage.empty)))
    .actionKeys("putBoth")

val two: Limits = Limits("two", steps = 2, actions = 2, search = 64)

val queries: Vector[Query] = Vector(
  query("durableStays") verify durableStays in anyDisk limits two,
  query("putAccepted") verify putAccepted in putThenFlush limits two,
  query("putStores") find putStores in putOnce limits two,
  query("putStoresThroughDisk")
    .verify(putStores)
    .in(putThenFlush)(using Reads.through(disk, store)) limits two,
  query("keptTogether") verify keptTogether in anyPair limits two,
  query("bothPut") find frontHeld in bothPut limits two
)
