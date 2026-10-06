// A Model with every name taken from its val or object: the machines state their types once, the
// composition names its members, syncs and Scenario classes by field selectors, the Scenarios start
// where their machines do, the evidence lists only the fact whose evidence is not its name, and the
// refined read needs no given. Every Definition ID is its declaration's fully qualified name, the
// objects it sits in included, and every family the package fixture.captured. The lifter's tests
// compare its IR with expected/captured.json and check those IDs and names.
package fixture.captured

import umpire.*
import umpire.realize.*, temporal.realize.{Role, RoleKind}
import temporal.server.api.testpilot.v1.CorrelatedEvidence
import Entities.{entry, lostData}

enum Kept derives Finite:
  case nothing, held

// The store's state, named apart from the machine object `Store`.
final case class StoreState(kept: Kept) derives Finite

enum Outcome derives Finite:
  case accepted, deferred

enum Fact derives Finite:
  case stored, staged
  case lost(hard: Boolean)

// Entities and an observation named after their vals. An entity that refers to another is
// compiled as a block that binds the reference before the call, which the lifter reads through.
// They sit in an object of their own, which the machine objects read while they initialize.
object Entities:
  val owner: Entity = Entity()
  val entry: Entity = Entity(key = "entryId", refer = Map("owner" -> owner))
  val lostData: Observation = Observation(on = entry, read = "lost")

// The actions, grouped by who takes them (fn-126 R14). An actor object is the actor of its name, and
// each action is named after where it is declared, fixture.captured.client.put, as the lifter's
// tests check.

// The actor `client`, whose members are the actions it takes.
object client extends Actor:
  val put = action(this)

// The system's own steps: a flush and a timer.
object background:
  val flush = internal
  val expire = timer

// A fault, taken by an actor named in place.
object faults:
  val crash = action(Actor("fault"))

val storeOpaque = assume
val flushRuns = assume.fair(background.flush)
val crashUnmodeled = hole

def putStore(s: StoreState): List[Step[StoreState, Outcome, Fact]] =
  if s.kept == Kept.nothing then
    List(Step(Outcome.accepted, StoreState(Kept.held), List(Fact.stored)))
  else Nil

def expireStore(s: StoreState): List[Step[StoreState, Outcome, Fact]] =
  if s.kept == Kept.held then List(Step(Outcome.deferred, s, List(Fact.lost(false)))) else Nil

// The opaque provider: only the fact whose evidence is not its name has a line.
object Store extends Machine[StoreState, Outcome, Fact]:
  val entity = entry
  val init = StoreState(Kept.nothing)
  def end(s: State) = s.kept == Kept.held
  val evidence: PartialFunction[Fact, String] = { case Fact.lost(_) => lostData.name }

  object monitors:
    val opaque = storeOpaque

  object rules extends Bindings(client.put ~> putStore, background.expire ~> expireStore)

enum Stage derives Finite:
  case empty, staged, durable

// The disk's state, named apart from the machine object `Disk`.
final case class DiskState(stage: Stage) derives Finite

def putDisk(d: DiskState): List[Step[DiskState, Outcome, Fact]] = d.stage match
  case Stage.empty => List(Step(Outcome.accepted, DiskState(Stage.staged), List(Fact.stored)))
  case Stage.staged | Stage.durable => Nil

def flushDisk(d: DiskState): List[Step[DiskState, Outcome, Fact]] = d.stage match
  case Stage.staged => List(Step(Outcome.deferred, DiskState(Stage.durable), List(Fact.staged)))
  case _            => Nil

def crashDisk(d: DiskState): List[Step[DiskState, Outcome, Fact]] =
  if d.stage == Stage.staged then crashUnmodeled.reached else Nil

enum Seen derives Finite:
  case never, once, twice

def countStored(seen: Seen, before: DiskState, after: Step[DiskState, Outcome, Fact]): Seen =
  if !after.facts.contains(Fact.stored) then seen
  else
    seen match
      case Seen.never             => Seen.once
      case Seen.once | Seen.twice => Seen.twice

val storedOnce =
  monitor[DiskState, Outcome, Fact, Seen](Seen.never)(countStored)(seen => seen == Seen.twice)

// A monitor in an object of its own, named after it: fixture.captured.Watched.storedTwice.
object Watched:
  val storedTwice =
    monitor[DiskState, Outcome, Fact, Seen](Seen.never)(countStored)(seen => seen == Seen.twice)

// The detailed provider, its types stated once in its object's parent.
object Disk extends Machine[DiskState, Outcome, Fact], FailureModel:
  val init = DiskState(Stage.empty)
  def end(d: State) = d.stage != Stage.staged
  val evidence: PartialFunction[Fact, String] = { case Fact.lost(_) => "lostData" }

  // What the store sees of a disk: whether it holds anything.
  object refinement extends Refinement(Store):
    def toProduct(d: DiskState): StoreState =
      if d.stage == Stage.empty then StoreState(Kept.nothing) else StoreState(Kept.held)
    val visible = (f: Fact) => f == Fact.stored

  object monitors:
    val once = storedOnce
    val twice = Watched.storedTwice

  object rules
      extends Bindings(
        client.put ~> putDisk,
        background.flush ~> flushDisk,
        faults.crash ~> crashDisk
      )

// A machine derived by keeping some of another's actions.
object PutOnly extends Derived(Disk.restrict(client.put))

// The pair's state, named apart from the composition object `DetailedPair`.
final case class DetailedPairState(front: StoreState, back: DiskState)

// Its sync is named after its first member's action, `put`.
object DetailedPair
    extends Composition[DetailedPairState](_.front -> Store, _.back -> Disk),
      FailureModel:
  def end(p: State) = p.front.kept == Kept.held
  object syncs extends Syncs:
    sync(_.front -> client.put, _.back -> client.put)
    replaces(_.back, Store)

val putStores = Store.property when client.put holds (after => after.facts.contains(Fact.stored))
val durableStays = Disk.property holdsAcross { (before, after) =>
  before.stage != Stage.durable || after.state.stage == Stage.durable
}
val frontHeld = DetailedPair.property holds (after => after.state.front.kept == Kept.held)

// Each starts in its machine's declared start, or in its members' starts. `put()` is the one class
// of an action with no input, as `put` is.
val putOnce = Store.scenario.actions(client.put())
val putThenFlush = Disk.scenario.actions(client.put, background.flush)
val bothPut = DetailedPair.scenario.actions(DetailedPair.synced(_.back -> client.put))

val two = Limits(steps = 2, actions = 2, search = 64)

val putStoresOnce = query find putStores in putOnce limits two total 2
val durableAfterFlush = query verify durableStays in putThenFlush limits two total 6
// A Property of the opaque store read on the disk, through the refinement the disk declares.
val putStoresThroughDisk = query verify putStores in putThenFlush limits two total 6
val bothPutHeld = query find frontHeld in bothPut limits two total 6

// Claims declared inside a function over a machine have no val: a Query named by neither is named
// after its Scenario's machine, its Scenario and its Property, `disk.any.durableStays`, and a
// Property and a Scenario in a list take the one explicit-name form.
def anyQueries(m: Machine[DiskState, Outcome, Fact]): Vector[Query] = Vector(
  query verify durableStays in m
    .scenario("any")
    .starts(DiskState(Stage.empty))
    .free limits two total 18,
  query(s"${m.name}.any.everPut") find (m.property(s"${m.name}.everPut") holds (after =>
    after.facts.contains(Fact.stored)
  )) in m.scenario("anyPut").starts(DiskState(Stage.empty)).free limits two total 18
)

// A local val inside a function names its declaration.
def localQueries(m: Machine[DiskState, Outcome, Fact]): Vector[Query] =
  val stays = m.property holds (after => after.state.stage != Stage.empty)
  Vector(query("localStays") verify stays in putThenFlush limits two total 6)

val durableEventually =
  Disk.leadsTo("durableEventually")(
    d => d.stage == Stage.staged,
    d => d.stage == Stage.durable,
    within = 2,
    flushRuns
  )

val queries: Vector[Query] =
  Vector(putStoresOnce, durableAfterFlush, putStoresThroughDisk, bothPutHeld)
val diskQueries: Vector[Query] = anyQueries(Disk)
val localDiskQueries: Vector[Query] = localQueries(Disk)

val ledger = Realization(
  machine = Store,
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
  observations = Vector(Observed[CorrelatedEvidence]("correlated-evidence")),
  scripts = Vector(Script("controller", Activation.Controller, Vector.empty))
)

// A channel the relay holds, with its delivery and its loss.
val wire = channel[Kept](capacity = 1, order = Order.fifo, loss = Loss.lossy)

// The relay's state, named apart from the machine object `Relay`.
final case class RelayState(heard: Kept, wire: Inbox[Kept])

given Finite[RelayState] =
  given Finite[Inbox[Kept]] = wire.contents
  Finite.derived

enum Note derives Finite:
  case heard, missed

// The client's send, which names the actor object as its actor.
object relaying:
  val send = action(client)

def sendStep(r: RelayState): List[Step[RelayState, Outcome, Note]] =
  if r.wire.isFull then Nil else List(Step(Outcome.accepted, r.copy(wire = r.wire.send(Kept.held))))

def hear(r: RelayState, k: Kept): List[Step[RelayState, Outcome, Note]] =
  List(Step(Outcome.accepted, r.copy(heard = k), List(Note.heard)))

def drop(r: RelayState, k: Kept): List[Step[RelayState, Outcome, Note]] =
  List(Step(Outcome.deferred, r, List(Note.missed)))

// Its facts are each confirmed by the evidence of their name, so it declares no evidence.
object Relay extends Machine[RelayState, Outcome, Note]:
  val init = RelayState(Kept.nothing, wire.empty)
  def end(relay: State) = true

  object rules extends Bindings(relaying.send ~> sendStep, wire.deliver ~> hear, wire.lose ~> drop)
