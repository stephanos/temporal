// The declarations checks read over a Model, lifted with it: an internal step, a declared hole beside
// disabled rows, assumptions, a monitor read at each evaluation point, the opaque machine a detailed
// one refines under a visible projection, a progress claim, a composition in which the detailed
// machine replaces the opaque one, and Properties, Scenarios and Queries on machines and compositions.
// The tests lift `queries` and `durableEventually` and compare the IR with expected/declarations.json.
package fixture.declarations

import framework.*

enum Kept derives Finite:
  case nothing, held

final case class StoreState(kept: Kept) derives Finite

enum Outcome derives Finite:
  case accepted, deferred

enum Fact derives Finite:
  case stored, staged

val put = action(Actor("client"))
val flush = internal
val crash = action(Actor("fault"))

val storeOpaque: Assumption = assume
val flushRuns: Assumption = assume("flushEventuallyRuns").fair(flush)
val crashUnmodeled: Hole = hole

def evidenceOf(f: Fact): String = f match
  case Fact.stored => "stored"
  case Fact.staged => "staged"

// The opaque provider: all its clients may rely on.
object Store extends Machine[StoreState, Outcome, Fact]:
  val init = StoreState(Kept.nothing)
  def end(s: State) = s.kept == Kept.held
  val evidence: Fact => String = evidenceOf

  object monitors:
    val opaque = storeOpaque

  object rules
      extends Bindings(
        put ~> (s =>
          if s.kept == Kept.nothing then
            List(Step(Outcome.accepted, StoreState(Kept.held), List(Fact.stored)))
          else Nil
        )
      )

enum Stage derives Finite:
  case empty, staged, durable

final case class DiskState(stage: Stage) derives Finite

def putStep(d: DiskState): List[Step[DiskState, Outcome, Fact]] = d.stage match
  case Stage.empty => List(Step(Outcome.accepted, DiskState(Stage.staged), List(Fact.stored)))
  case Stage.staged | Stage.durable => Nil

// A stutter of the store: staging becomes durable, and the store sees nothing of it.
def flushStep(d: DiskState): List[Step[DiskState, Outcome, Fact]] = d.stage match
  case Stage.staged => List(Step(Outcome.deferred, DiskState(Stage.durable), List(Fact.staged)))
  case _            => Nil

// What a crash does to staged data is left unknown; elsewhere a crash is disabled.
def crashStep(d: DiskState): List[Step[DiskState, Outcome, Fact]] =
  if d.stage == Stage.staged then crashUnmodeled.reached else Nil

enum Seen derives Finite:
  case never, once, twice

def countStored(seen: Seen, before: DiskState, after: Step[DiskState, Outcome, Fact]): Seen =
  if !after.facts.contains(Fact.stored) then seen
  else
    seen match
      case Seen.never             => Seen.once
      case Seen.once | Seen.twice => Seen.twice

val storedOnce: Monitor[DiskState, Outcome, Fact, Seen] =
  monitor[DiskState, Outcome, Fact, Seen](Seen.never)(countStored)(seen => seen == Seen.twice)

val endsDurable: Monitor[DiskState, Outcome, Fact, Boolean] =
  monitor[DiskState, Outcome, Fact, Boolean](false)((_, _, after) =>
    after.state.stage == Stage.durable
  )(durable => !durable).readAtEnds

val stagedBeforeDurable: Monitor[DiskState, Outcome, Fact, Boolean] =
  monitor[DiskState, Outcome, Fact, Boolean](false)((seen, _, after) =>
    seen || after.facts.contains(Fact.staged)
  )(seen => !seen).readAfter(after => after.state.stage == Stage.durable)

// The detailed provider.
object Disk extends Machine[DiskState, Outcome, Fact], FailureModel:
  val init = DiskState(Stage.empty)
  def end(d: State) = d.stage != Stage.staged
  val evidence: Fact => String = evidenceOf

  // What the store sees of a disk: whether it holds anything.
  object refinement extends Refinement(Store):
    def toProduct(d: DiskState): StoreState =
      if d.stage == Stage.empty then StoreState(Kept.nothing) else StoreState(Kept.held)
    val visible = (f: Fact) => f == Fact.stored
    val visibleOutcomes = (o: Outcome) => o == Outcome.accepted

  object monitors:
    val once = storedOnce
    val durableAtEnd = endsDurable
    val stagedFirst = stagedBeforeDurable

  object rules extends Bindings(put ~> putStep, flush ~> flushStep, crash ~> crashStep)

val durableEventually: Progress[DiskState] =
  Disk.leadsTo("durableEventually")(
    d => d.stage == Stage.staged,
    d => d.stage == Stage.durable,
    within = 2,
    flushRuns
  )

final case class PairState(front: StoreState, back: StoreState)

final case class DetailedPairState(front: StoreState, back: DiskState)

object Pair extends Composition[PairState](_.front -> Store, _.back -> Store):
  def end(p: State) = p.front.kept == p.back.kept
  object syncs extends Syncs:
    sync("putBoth", _.front -> put, _.back -> put)

object DetailedPair
    extends Composition[DetailedPairState](_.front -> Store, _.back -> Disk),
      FailureModel:
  def end(p: State) = p.front.kept == Kept.held
  object syncs extends Syncs:
    sync("putBoth", _.front -> put, _.back -> put)
    replaces(_.back, Store)

val durableStays: Property[DiskState] = Disk.property holdsAcross { (before, after) =>
  before.stage != Stage.durable || after.state.stage == Stage.durable
}
val putStores: Property[StoreState] =
  Store.property when put holds (after => after.facts.contains(Fact.stored))
val putAccepted: Property[DiskState] =
  Disk.property.whenAction("put") holds (after => after.outcome == Outcome.accepted)
val keptTogether: Property[PairState] =
  Pair.property holds (after => after.state.front.kept == after.state.back.kept)
val frontHeld: Property[DetailedPairState] =
  DetailedPair.property holds (after => after.state.front.kept == Kept.held)

val putThenFlush: Scenario[DiskState] =
  Disk.scenario.starts(DiskState(Stage.empty)).actions(put, flush)
val anyDisk: Scenario[DiskState] = Disk.scenario("any").starts(DiskState(Stage.empty)).free
val putOnce: Scenario[StoreState] = Store.scenario.starts(StoreState(Kept.nothing)).actions(put)
val anyPair: Scenario[PairState] =
  Pair.scenario("any").starts(PairState(StoreState(Kept.nothing), StoreState(Kept.nothing))).free
val bothPut: Scenario[DetailedPairState] =
  DetailedPair.scenario
    .starts(DetailedPairState(StoreState(Kept.nothing), DiskState(Stage.empty)))
    .actions(DetailedPair.synced(_.front -> put))

val two: Limits = Limits(steps = 2, actions = 2, search = 64)

val queries: Vector[Query] = Vector(
  query("durableStays") verify durableStays in anyDisk limits two total 18,
  query("putAccepted") verify putAccepted in putThenFlush limits two total 6,
  query("putStores") find putStores in putOnce limits two total 2,
  query("putStoresThroughDisk")
    .verify(putStores)
    .in(putThenFlush) limits two total 6,
  query("keptTogether") verify keptTogether in anyPair limits two total 8,
  query("bothPut") find frontHeld in bothPut limits two total 6
)
