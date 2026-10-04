// Spelled.scala's Model with every name taken from its val: the family is a given, the machines state
// their types once, the composition names its members, syncs and Scenario classes by field
// selectors, the Scenarios start where their machines do, the evidence lists only the fact whose
// evidence is not its name, the refined read needs no given, and one DefinitionScope keeps every
// symbol-based Definition ID Spelled.scala's owner gives and every top-level type's name in
// fixture.spelled. The lifter's tests lift both and require one IR of the two, but for positions and
// the owner of the functions each file declares.
package fixture.captured

import umpire.*
import umpire.realize.*

given DefinitionScope = DefinitionScope("fixture.spelled.Spelled$package$")

given Family = Family("fixture.spelled")

enum Kept derives Finite:
  case nothing, held

final case class Store(kept: Kept) derives Finite

enum Outcome derives Finite:
  case accepted, deferred

enum Fact derives Finite:
  case stored, staged
  case lost(hard: Boolean)

val put = action(Party("client"))
val flush = internal
val crash = action(Party("fault"))
val expire = timer

val storeOpaque = assume
val flushRuns = assume.fair(flush)
val crashUnmodeled = hole

def putStore(s: Store): List[Step[Store, Outcome, Fact]] =
  if s.kept == Kept.nothing then List(Step(Outcome.accepted, Store(Kept.held), List(Fact.stored)))
  else Nil

def expireStore(s: Store): List[Step[Store, Outcome, Fact]] =
  if s.kept == Kept.held then List(Step(Outcome.deferred, s, List(Fact.lost(false)))) else Nil

/** The opaque provider: only the fact whose evidence is not its name has a line. */
val store = machine[Store, Outcome, Fact] {
  assumes(storeOpaque)
  starts(Store(Kept.nothing))
  ends(s => s.kept == Kept.held)
  evidence { case Fact.lost(_) => "lostData" }
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

val storedOnce =
  monitor[Disk, Outcome, Fact, Seen](Seen.never)(countStored)(seen => seen == Seen.twice)

/** The detailed provider, its types stated once as its val's type. */
val disk: Machine[Disk, Outcome, Fact] = machine {
  refines(store)(stored)
  visible(f => f == Fact.stored)
  monitors(storedOnce)
  starts(Disk(Stage.empty))
  ends(d => d.stage != Stage.staged)
  evidence { case Fact.lost(_) => "lostData" }
  steps(put ~> putDisk, flush ~> flushDisk, crash ~> crashDisk)
}

/** A machine derived by keeping some of another's actions. */
val putOnly = disk.restrict(put)

final case class DetailedPair(front: Store, back: Disk)

// Its sync is named after its first member's action, `put`.
val detailedPair =
  compose[DetailedPair](_.front -> store, _.back -> disk)
    .sync(_.front -> put, _.back -> put)
    .replaces(_.back, store)
    .ends(p => p.front.kept == Kept.held)

val putStores = store.property when put holds (after => after.facts.contains(Fact.stored))
val durableStays = disk.property holdsAcross { (before, after) =>
  before.stage != Stage.durable || after.state.stage == Stage.durable
}
val frontHeld = detailedPair.property holds (after => after.state.front.kept == Kept.held)

// Each starts in its machine's declared start, or in its members' starts. `put()` is the one class
// of an action with no input, as `put` is.
val putOnce = store.scenario.actions(put())
val putThenFlush = disk.scenario.actions(put, flush)
val bothPut = detailedPair.scenario.actions(detailedPair.synced(_.back -> put))

val two = Limits(steps = 2, actions = 2, search = 64)

val putStoresOnce = query find putStores in putOnce limits two total 2
val durableAfterFlush = query verify durableStays in putThenFlush limits two total 6
// A Property of the opaque store read on the disk, through the refinement the disk declares.
val putStoresThroughDisk = query verify putStores in putThenFlush limits two total 6
val bothPutHeld = query find frontHeld in bothPut limits two total 6

/**
 * Claims declared inside a function over a machine have no val: a Query named by neither is named
 * after its Scenario's machine, its Scenario and its Property, `disk.any.durableStays`, and a
 * Property and a Scenario in a list take the one explicit-name form.
 */
def anyQueries(m: Machine[Disk, Outcome, Fact]): Vector[Query] = Vector(
  query verify durableStays in m
    .scenario("any")
    .starts(Disk(Stage.empty))
    .free limits two total 18,
  query(s"${m.name}.any.everPut") find (m.property(s"${m.name}.everPut") holds (after =>
    after.facts.contains(Fact.stored)
  )) in m.scenario("anyPut").starts(Disk(Stage.empty)).free limits two total 18
)

/** A local val inside a function names its declaration. */
def localQueries(m: Machine[Disk, Outcome, Fact]): Vector[Query] =
  val stays = m.property holds (after => after.state.stage != Stage.empty)
  Vector(query("localStays") verify stays in putThenFlush limits two total 6)

val durableEventually =
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

val ledger = Realization(
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
val wire = channel[Kept](capacity = 1, order = Order.fifo, loss = Loss.lossy)

final case class Relay(heard: Kept, wire: Inbox[Kept])

given Finite[Relay] =
  given Finite[Inbox[Kept]] = wire.contents
  Finite.derived

enum Note derives Finite:
  case heard, missed

val send = action(Party("client"))

def sendStep(r: Relay): List[Step[Relay, Outcome, Note]] =
  if r.wire.isFull then Nil else List(Step(Outcome.accepted, r.copy(wire = r.wire.send(Kept.held))))

def hear(r: Relay, k: Kept): List[Step[Relay, Outcome, Note]] =
  List(Step(Outcome.accepted, r.copy(heard = k), List(Note.heard)))

def drop(r: Relay, k: Kept): List[Step[Relay, Outcome, Note]] =
  List(Step(Outcome.deferred, r, List(Note.missed)))

// Its facts are each confirmed by the evidence of their name, so it declares no evidence.
val relay = machine[Relay, Outcome, Note] {
  starts(Relay(Kept.nothing, wire.empty))
  ends(_ => true)
  steps(send ~> sendStep, wire.deliver ~> hear, wire.lose ~> drop)
}
