// Every declaration that can take its name from its val, spelled the way Models were written before:
// each name and family written out, every start and every evidence line stated. Captured.scala is the
// same Model written with captured names, a given family, defaulted starts and evidence and a
// refinement read with no given. The lifter's tests lift both and require one IR of the two, but for
// positions and the owner of the types and functions each file declares.
package fixture.spelled

import umpire.*
import umpire.realize.*

val Family: umpire.Family = umpire.Family("fixture.spelled")

enum Kept derives Finite:
  case nothing, held

final case class Store(kept: Kept) derives Finite

enum Outcome derives Finite:
  case accepted, deferred

enum Fact derives Finite:
  case stored, staged
  case lost(hard: Boolean)

val put = action("put", Party("client"))
val flush = internal("flush")
val crash = action("crash", Party("fault"))
val expire = timer("expire")

val storeOpaque: Assumption = assume("storeOpaque")
val flushRuns: Assumption = assume("flushRuns").fair(flush)
val crashUnmodeled: Hole = hole("crashUnmodeled")

def putStore(s: Store): List[Step[Store, Outcome, Fact]] =
  if s.kept == Kept.nothing then List(Step(Outcome.accepted, Store(Kept.held), List(Fact.stored)))
  else Nil

def expireStore(s: Store): List[Step[Store, Outcome, Fact]] =
  if s.kept == Kept.held then List(Step(Outcome.deferred, s, List(Fact.lost(false)))) else Nil

/** The opaque provider, with every fact's evidence written out. */
val store: Machine[Store, Outcome, Fact] =
  machine[Store, Outcome, Fact](Family, "store") {
    assumes(storeOpaque)
    starts(Store(Kept.nothing))
    ends(s => s.kept == Kept.held)
    evidence {
      case Fact.stored  => "stored"
      case Fact.staged  => "staged"
      case Fact.lost(_) => "lostData"
    }
    steps(put ~> putStore, expire ~> expireStore)
  }

enum Stage derives Finite:
  case empty, staged, durable

final case class Disk(stage: Stage) derives Finite

def stored(d: Disk): Store =
  if d.stage == Stage.empty then Store(Kept.nothing) else Store(Kept.held)

def putDisk(d: Disk): List[Step[Disk, Outcome, Fact]] = d.stage match
  case Stage.empty => List(Step(Outcome.accepted, Disk(Stage.staged), List(Fact.stored)))
  case Stage.staged | Stage.durable => Nil

def flushDisk(d: Disk): List[Step[Disk, Outcome, Fact]] = d.stage match
  case Stage.staged => List(Step(Outcome.deferred, Disk(Stage.durable), List(Fact.staged)))
  case _            => Nil

def crashDisk(d: Disk): List[Step[Disk, Outcome, Fact]] =
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

/** The detailed provider: the facts with fields keep their line, the rest are written out. */
val disk: Machine[Disk, Outcome, Fact] =
  machine[Disk, Outcome, Fact](Family, "disk") {
    refines(store)(stored)
    visible(f => f == Fact.stored)
    monitors(storedOnce)
    starts(Disk(Stage.empty))
    ends(d => d.stage != Stage.staged)
    evidence {
      case Fact.stored  => "stored"
      case Fact.staged  => "staged"
      case Fact.lost(_) => "lostData"
    }
    steps(put ~> putDisk, flush ~> flushDisk, crash ~> crashDisk)
  }

/** A machine derived by keeping some of another's actions. */
val putOnly: Machine[Disk, Outcome, Fact] = disk.restrict(Family, "putOnly")(put)

final case class DetailedPair(front: Store, back: Disk)

val detailedPair: Composition[DetailedPair] =
  compose[DetailedPair](Family, "detailedPair")("front" -> store, "back" -> disk)
    .sync("put", "front" -> put, "back" -> put)
    .replaces("back", store)
    .ends(p => p.front.kept == Kept.held)

val putStores: Property[Store] =
  store.property("putStores") when put holds (after => after.facts.contains(Fact.stored))
val durableStays: Property[Disk] = disk.property("durableStays") holdsAcross { (before, after) =>
  before.stage != Stage.durable || after.state.stage == Stage.durable
}
val frontHeld: Property[DetailedPair] =
  detailedPair.property("frontHeld") holds (after => after.state.front.kept == Kept.held)

val putOnce: Scenario[Store] = store.scenario("putOnce").starts(Store(Kept.nothing)).actions(put)
val putThenFlush: Scenario[Disk] =
  disk.scenario("putThenFlush").starts(Disk(Stage.empty)).actions(put, flush)
val bothPut: Scenario[DetailedPair] =
  detailedPair
    .scenario("bothPut")
    .starts(DetailedPair(Store(Kept.nothing), Disk(Stage.empty)))
    .actionKeys("put")

val two: Limits = Limits("two", steps = 2, actions = 2, search = 64)

val putStoresOnce: Query = query("putStoresOnce") find putStores in putOnce limits two total 2
val durableAfterFlush: Query =
  query("durableAfterFlush") verify durableStays in putThenFlush limits two total 6
// A Property of the opaque store read on the disk, through the refinement the disk declares.
val putStoresThroughDisk: Query =
  query("putStoresThroughDisk") verify putStores in putThenFlush limits two total 6
val bothPutHeld: Query = query("bothPutHeld") find frontHeld in bothPut limits two total 6

/**
 * Claims declared inside a function over a machine take the one explicit-name form, with no val:
 * a computed name, a Property and a Scenario in a list.
 */
def anyQueries(m: Machine[Disk, Outcome, Fact]): Vector[Query] = Vector(
  query(s"${m.name}.any.durableStays") verify durableStays in m
    .scenario("any")
    .starts(Disk(Stage.empty))
    .free limits two total 18,
  query(s"${m.name}.any.everPut") find (m.property(s"${m.name}.everPut") holds (after =>
    after.facts.contains(Fact.stored)
  )) in m.scenario("anyPut").starts(Disk(Stage.empty)).free limits two total 18
)

/** A local val inside a function names its declaration. */
def localQueries(m: Machine[Disk, Outcome, Fact]): Vector[Query] =
  val stays = m.property("stays") holds (after => after.state.stage != Stage.empty)
  Vector(query("localStays") verify stays in putThenFlush limits two total 6)

val durableEventually: Progress[Disk] =
  disk.leadsTo("durableEventually")(
    d => d.stage == Stage.staged,
    d => d.stage == Stage.durable,
    within = 2,
    flushRuns
  )

val queries: Vector[Query] =
  Vector(putStoresOnce, durableAfterFlush, putStoresThroughDisk, bothPutHeld)
val diskQueries: Vector[Query] = anyQueries(disk)
val localDiskQueries: Vector[Query] = localQueries(disk)

val ledger: Realization = Realization(
  name = "ledger",
  machine = store,
  producer = "fixture.spelled",
  producerVersion = "1",
  roles = Vector(Role("fixture.spelled.server", RoleKind.endpoint)),
  correlation = Correlation(
    projection = "fixture.spelled.projection",
    run = "fixture.spelled.scope.run",
    operation = "fixture.spelled.scope.store",
    observation = "correlated-evidence",
    events = 8,
    buffered = 8,
    keys = 2,
    support = 16,
    work = 1000,
    eventSize = 512
  ),
  scripts = Vector(Script("controller", Activation.Controller, Vector.empty))
)

// A channel the relay holds, with its delivery and its loss.
val wire: Channel[Kept] =
  channel[Kept]("wire", capacity = 1, order = Order.fifo, loss = Loss.lossy)

final case class Relay(heard: Kept, wire: Inbox[Kept])

given Finite[Relay] =
  given Finite[Inbox[Kept]] = wire.contents
  Finite.derived

enum Note derives Finite:
  case heard, missed

val send = action("send", Party("client"))

def sendStep(r: Relay): List[Step[Relay, Outcome, Note]] =
  if r.wire.isFull then Nil else List(Step(Outcome.accepted, r.copy(wire = r.wire.send(Kept.held))))

def hear(r: Relay, k: Kept): List[Step[Relay, Outcome, Note]] =
  List(Step(Outcome.accepted, r.copy(heard = k), List(Note.heard)))

def drop(r: Relay, k: Kept): List[Step[Relay, Outcome, Note]] =
  List(Step(Outcome.deferred, r, List(Note.missed)))

val relay: Machine[Relay, Outcome, Note] =
  machine[Relay, Outcome, Note](Family, "relay") {
    starts(Relay(Kept.nothing, wire.empty))
    ends(_ => true)
    evidence {
      case Note.heard  => "heard"
      case Note.missed => "missed"
    }
    steps(send ~> sendStep, wire.deliver ~> hear, wire.lose ~> drop)
  }
