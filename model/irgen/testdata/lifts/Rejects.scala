// Declarations the lifter must refuse before any IR is written, each at the line that declares it.
// The lifter's tests lift every root below, and one that names nothing, in one run, and compare the
// diagnostics with expected/rejects.txt. Its machines and compositions are objects; a machine whose
// step functions are written out binds them by hand in the core, `object rules extends Bindings(...)`.
package fixture.rejects

import umpire.*

val Family: umpire.Family = umpire.Family("fixture.rejects")

enum Note derives Finite:
  case ping

enum Outcome derives Finite:
  case accepted

val go = action(Party("fixture"))

/** A list in a state has no bound. */
final case class Unbounded(notes: List[Note])

given Finite[Unbounded] = Finite.of(Unbounded(Nil), Unbounded(List(Note.ping)))

object Hoarding extends Machine[Unbounded, Outcome, Nothing]:
  val init = Unbounded(Nil)
  def end(s: State) = true
  object rules extends Bindings(go ~> (_ => Nil))

/** A channel that holds nothing. */
val closed: Channel[Note] =
  channel[Note](capacity = 0, order = Order.fifo, loss = Loss.reliable)

final case class WaitingState(inbox: Inbox[Note])

given Finite[WaitingState] =
  given Finite[Inbox[Note]] = closed.contents
  Finite.derived

object Waiting extends Machine[WaitingState, Outcome, Nothing]:
  val init = WaitingState(closed.empty)
  def end(s: State) = true
  object rules extends Bindings(closed.deliver ~> ((w, _) => List(Step(Outcome.accepted, w))))

/** A lossy channel whose loss no step says the meaning of. */
val leaky: Channel[Note] =
  channel[Note](capacity = 1, order = Order.fifo, loss = Loss.lossy)

final case class ListeningState(inbox: Inbox[Note])

given Finite[ListeningState] =
  given Finite[Inbox[Note]] = leaky.contents
  Finite.derived

object Listening extends Machine[ListeningState, Outcome, Nothing]:
  val init = ListeningState(leaky.empty)
  def end(s: State) = true
  object rules extends Bindings(leaky.deliver ~> ((l, _) => List(Step(Outcome.accepted, l))))

/** One channel held in two fields: which of them a delivery takes from is undefined. */
final case class Twice(first: Inbox[Note], second: Inbox[Note])

given Finite[Twice] =
  given Finite[Inbox[Note]] = leaky.contents
  Finite.derived

object Doubled extends Machine[Twice, Outcome, Nothing]:
  val init = Twice(leaky.empty, leaky.empty)
  def end(s: State) = true
  object rules extends Bindings(leaky.lose ~> ((t, _) => List(Step(Outcome.accepted, t))))

/** A function that calls itself. */
def countdown(n: Int): Int = if n <= 0 then 0 else countdown(n - 1)

final case class CounterState(n: Int)

given Finite[CounterState] =
  given Finite[Int] = Finite.upTo(2)
  Finite.derived

object Counter extends Machine[CounterState, Outcome, Nothing]:
  val init = CounterState(2)
  def end(s: State) = true
  object rules
      extends Bindings(go ~> (c => List(Step(Outcome.accepted, CounterState(countdown(c.n))))))

final case class Flag(on: Boolean) derives Finite

val flip = action(Party("fixture"))

def flipStep(f: Flag): List[Step[Flag, Outcome, Nothing]] = List(
  Step(Outcome.accepted, Flag(!f.on))
)

object First extends Machine[Flag, Outcome, Nothing]:
  val init = Flag(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> flipStep)

object Second extends Machine[Flag, Outcome, Nothing]:
  val init = Flag(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> flipStep)

object Refining extends Machine[Flag, Outcome, Nothing]:
  val init = Flag(false)
  def end(s: State) = true
  object refinement extends Refinement(First):
    def toProduct(f: Flag) = f
  object rules extends Bindings(flip ~> flipStep)

val flips: Property[Flag] =
  Second.property holds (after => after.outcome == Outcome.accepted)
val flipping: Scenario[Flag] = Refining.scenario.starts(Flag(false)).actions(flip)
val secondFlips: Scenario[Flag] = Second.scenario.starts(Flag(false)).actions(flip)
val one: Limits = Limits(steps = 1, actions = 1, search = 8)

/** A Property read through a refinement its Scenario's machine does not declare. */
val crossedRead: Query =
  query.verify(flips).in(flipping) limits one

/** Limits below zero. */
val backwards: Limits = Limits(steps = -1, actions = 1, search = 8)
val negative: Query = query verify flips in secondFlips limits backwards

/** Two monitors under one name would share one Definition ID. */
val twiceFirst: Monitor[Flag, Outcome, Nothing, Boolean] =
  monitor[Flag, Outcome, Nothing, Boolean]("twice", false)((seen, _, _) => seen)(seen => seen)
val twiceSecond: Monitor[Flag, Outcome, Nothing, Boolean] =
  monitor[Flag, Outcome, Nothing, Boolean]("twice", true)((seen, _, _) => seen)(seen => seen)

object Watched extends Machine[Flag, Outcome, Nothing]:
  val init = Flag(false)
  def end(s: State) = true
  object monitors extends Section:
    val firstTwice = twiceFirst
    val secondTwice = twiceSecond
  object rules extends Bindings(flip ~> flipStep)

/** What a refined machine sees, on a machine that refines none. */
object Unrefined extends Machine[Flag, Outcome, Nothing]:
  val init = Flag(false)
  def end(s: State) = true
  def visible(f: Nothing) = true
  object rules extends Bindings(flip ~> flipStep)

final case class Flags(left: Flag, middle: Flag, right: Flag)

/** A replacement of a member the composition does not have. */
object Misplaced extends Composition[Flags](_.left -> First, _.right -> Refining):
  def end(s: State) = false
  object syncs extends Syncs:
    replaces(_.middle, First)

/** The outcomes a refined machine sees, on a machine that refines none. */
object UnrefinedOutcomes extends Machine[Flag, Outcome, Nothing]:
  val init = Flag(false)
  def end(s: State) = true
  def visibleOutcomes(o: Outcome) = true
  object rules extends Bindings(flip ~> flipStep)

/** Int messages whose catalog is not a range the IR can carry. */
val counts: Channel[Int] =
  channel[Int](capacity = 1, order = Order.fifo, loss = Loss.reliable)(using
    Finite.of(1, 5)
  )

final case class CountingState(inbox: Inbox[Int])

given Finite[CountingState] =
  given Finite[Inbox[Int]] = counts.contents
  Finite.derived

object Counting extends Machine[CountingState, Outcome, Nothing]:
  val init = CountingState(counts.empty)
  def end(s: State) = true
  object rules extends Bindings()

/** List messages, which have no finite catalog in the IR. */
val batches: Channel[List[Note]] =
  channel[List[Note]](capacity = 1, order = Order.fifo, loss = Loss.reliable)(using
    Finite.of(Nil, List(Note.ping))
  )

final case class BatchingState(inbox: Inbox[List[Note]])

given Finite[BatchingState] =
  given Finite[Inbox[List[Note]]] = batches.contents
  Finite.derived

object Batching extends Machine[BatchingState, Outcome, Nothing]:
  val init = BatchingState(batches.empty)
  def end(s: State) = true
  object rules extends Bindings()

/** A channel whose order is computed rather than named. */
val shuffled: Channel[Note] =
  channel[Note](capacity = 1, order = Order.fromOrdinal(1), loss = Loss.reliable)

final case class ShufflingState(inbox: Inbox[Note])

given Finite[ShufflingState] =
  given Finite[Inbox[Note]] = shuffled.contents
  Finite.derived

object Shuffling extends Machine[ShufflingState, Outcome, Nothing]:
  val init = ShufflingState(shuffled.empty)
  def end(s: State) = true
  object rules extends Bindings()

/** A channel whose loss is computed rather than named. */
val guessed: Channel[Note] =
  channel[Note](capacity = 1, order = Order.fifo, loss = Loss.valueOf("reliable"))

final case class GuessingState(inbox: Inbox[Note])

given Finite[GuessingState] =
  given Finite[Inbox[Note]] = guessed.contents
  Finite.derived

object Guessing extends Machine[GuessingState, Outcome, Nothing]:
  val init = GuessingState(guessed.empty)
  def end(s: State) = true
  object rules extends Bindings()

// Every fixture takes its family from this given, and its name from its object or val.
given umpire.Family = Family

final case class Lamp(lit: Boolean) derives Finite

def lampStep(l: Lamp): List[Step[Lamp, Outcome, Nothing]] = List(
  Step(Outcome.accepted, Lamp(!l.lit))
)

/** An owner that pins its Definition IDs twice. */
object PinnedTwice:
  given DefinitionScope = DefinitionScope("fixture.rejects.Former$package$")
  val again: DefinitionScope = DefinitionScope("fixture.rejects.Former$package$")
  val twiceTick = action(Party("fixture"))
  object PinnedTwice extends Machine[Lamp, Outcome, Nothing]:
    val init = Lamp(false)
    def end(s: State) = true
    object rules extends Bindings(twiceTick ~> lampStep)

/** A pin in an object nested in an object that pins its own. */
object PinsOuter:
  given DefinitionScope = DefinitionScope("fixture.rejects.Former$package$")
  object PinsInner:
    given DefinitionScope = DefinitionScope("fixture.rejects.Other$package$")
    val innerTick = action(Party("fixture"))
  object PinnedNested extends Machine[Lamp, Outcome, Nothing]:
    val init = Lamp(false)
    def end(s: State) = true
    object rules extends Bindings(PinsInner.innerTick ~> lampStep)

/** Two owners pinned to one former owner, each with an action of one name. */
object SharedA:
  given DefinitionScope = DefinitionScope("fixture.rejects.Shared$")
  val share = action(Party("fixture"))

object SharedB:
  given DefinitionScope = DefinitionScope("fixture.rejects.Shared$")
  val share = action(Party("fixture"))

object SharedIds extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(SharedA.share ~> lampStep, SharedB.share ~> lampStep)

/** An owner pinned to itself. */
object Self:
  given DefinitionScope = DefinitionScope("fixture.rejects.Self$")
  val selfTick = action(Party("fixture"))
  object PinnedSelf extends Machine[Lamp, Outcome, Nothing]:
    val init = Lamp(false)
    def end(s: State) = true
    object rules extends Bindings(selfTick ~> lampStep)

/** A pin computed rather than written as a literal. */
object Computed:
  given DefinitionScope = DefinitionScope("fixture.rejects." + "Former$package$")
  val computedTick = action(Party("fixture"))
  object PinnedComputed extends Machine[Lamp, Outcome, Nothing]:
    val init = Lamp(false)
    def end(s: State) = true
    object rules extends Bindings(computedTick ~> lampStep)

/** A family whose root is computed rather than written as a literal. */
object ComputedFamily:
  given umpire.Family = umpire.Family("fixture." + "rejects")
  object FamilyComputed extends Machine[Lamp, Outcome, Nothing]:
    val init = Lamp(false)
    def end(s: State) = true
    object rules extends Bindings(flip ~> lampStep)

/** Limits named after an anonymous given, a name the compiler made up. */
object Anonymous:
  given Limits = Limits(steps = 1, actions = 1, search = 8)
  val anonymous: Query = query verify flips in secondFlips limits summon[Limits]

/**
 * Two Queries of one Scenario and Property in a list, which no val names: both take the name
 * `second.secondFlips.flips`.
 */
val unnamedTwice: Vector[Query] = Vector(
  query verify flips in secondFlips limits one,
  query find flips in secondFlips limits one
)

/** A Property in the body of a function, which no val names. */
def unnamedChecks(m: Machine[Flag, Outcome, Nothing]): Vector[Query] = Vector(
  query("unnamedProperty") verify (m.property holds (after =>
    after.outcome == Outcome.accepted
  )) in secondFlips limits one
)
val unnamedProperty: Vector[Query] = unnamedChecks(Second)

/** A Scenario in a list, which no val names. */
val unnamedScenario: Vector[Query] = Vector(
  query("unnamedScenario") verify flips in Second.scenario.actions(flip) limits one
)

/** Two machines named alike in two objects. */
object TwinA:
  object Twin extends Machine[Lamp, Outcome, Nothing]:
    val init = Lamp(false)
    def end(s: State) = true
    object rules extends Bindings(flip ~> lampStep)

object TwinB:
  object Twin extends Machine[Lamp, Outcome, Nothing]:
    val init = Lamp(true)
    def end(s: State) = true
    object rules extends Bindings(flip ~> lampStep)

final case class Lamps(left: Lamp, right: Lamp)

object Twins extends Composition[Lamps](_.left -> TwinA.Twin, _.right -> TwinB.Twin):
  def end(s: State) = false
  object syncs extends Syncs

/** Two Queries named alike in two objects. */
object AskedA:
  val asked = query verify flips in secondFlips limits one

object AskedB:
  val asked = query find flips in secondFlips limits one

val askedTwice: Vector[Query] = Vector(AskedA.asked, AskedB.asked)

/** Two Limits named alike in two objects, with different bounds. */
object BoundA:
  val bound = Limits(steps = 1, actions = 1, search = 8)

object BoundB:
  val bound = Limits(steps = 2, actions = 1, search = 8)

val boundTwice: Vector[Query] = Vector(
  query("boundFirst") verify flips in secondFlips limits BoundA.bound,
  query("boundSecond") verify flips in secondFlips limits BoundB.bound
)

/** Two actions named alike in two objects, bound by one machine. */
object TapA:
  val tap = action(Party("fixture"))

object TapB:
  val tap = action(Party("fixture"))

object Tapped extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(TapA.tap ~> lampStep, TapB.tap ~> lampStep)

object OneStart extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> lampStep)

enum Dropped derives Finite:
  case kept
  case lost(hard: Boolean)

/** A machine that declares no evidence, of a fact with fields. */
object Undeclared extends Machine[Lamp, Outcome, Dropped]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings()

/** Evidence that names no line for a fact with fields. */
object Unlisted extends Machine[Lamp, Outcome, Dropped]:
  val init = Lamp(false)
  def end(s: State) = true
  val evidence: PartialFunction[Dropped, String] = { case Dropped.kept => "keptData" }
  object rules extends Bindings()

/** Evidence that covers only some values of a fact with fields. */
object Partial extends Machine[Lamp, Outcome, Dropped]:
  val init = Lamp(false)
  def end(s: State) = true
  val evidence: PartialFunction[Dropped, String] = { case Dropped.lost(true) => "hardLoss" }
  object rules extends Bindings()

/** A Property read on a machine of another state type, which refines nothing. */
val lampFlips = OneStart.scenario.actions(flip)
val unrelatedRead: Query = query verify flips in lampFlips limits one

/** Two assumptions named alike in two objects, assumed by one machine. */
object OpaqueA:
  val opaque = assume

object OpaqueB:
  val opaque = assume

object AssumedTwice extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object monitors extends Section:
    val opaqueA = OpaqueA.opaque
    val opaqueB = OpaqueB.opaque
  object rules extends Bindings()

/** Two holes named alike in two objects, reached by one machine. */
object GapA:
  val gap = hole

object GapB:
  val gap = hole

def gapStep(l: Lamp): List[Step[Lamp, Outcome, Nothing]] =
  if l.lit then GapA.gap.reached else GapB.gap.reached

object GapTwice extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> gapStep)

/** A derivation that rebinds an action its source does not bind. */
val push = action(Party("fixture"))
object RebindPush extends Derived(OneStart.rebind(push ~> lampStep))

/** A derivation that extends its source by an action it binds already. */
object ExtendBound extends Derived(OneStart.extend(flip ~> lampStep))

/** A derivation that binds one action twice. */
def darkStep(l: Lamp): List[Step[Lamp, Outcome, Nothing]] = List(
  Step(Outcome.accepted, Lamp(false))
)
object ReboundTwice extends Derived(OneStart.rebind(flip ~> lampStep, flip ~> darkStep))

/** A derivation that names an assumption its source assumes already. */
val lampOpaque = assume
object AssumingLamp extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object monitors extends Section:
    val opaque = lampOpaque
  object rules extends Bindings(flip ~> lampStep)
object AssumedAgain extends Derived(AssumingLamp.assuming(lampOpaque))

/** A derivation that names one assumption twice. */
val lampFaulty = assume
object AssumingTwice extends Derived(OneStart.assuming(lampFaulty, lampFaulty))

/** A refinement replaced where the source declares none. */
object RefinedNothing extends Derived(OneStart.refining(First)(l => Flag(l.lit)))

/** A refinement replaced by one of a machine whose types differ from the replaced one's. */
enum Seen derives Finite:
  case seen

final case class Glimpse(at: Seen) derives Finite

object Glance extends Machine[Glimpse, Seen, Nothing]:
  val init = Glimpse(Seen.seen)
  def end(s: State) = true
  object rules extends Bindings()
object FlagLamp extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object refinement extends Refinement(First):
    def toProduct(l: Lamp) = Flag(l.lit)
  object rules extends Bindings(flip ~> lampStep)
object RefinedOtherwise extends Derived(FlagLamp.refining(Glance)(_ => Glimpse(Seen.seen)))

/** Two machines derived from each other. */
object LoopFirst extends Derived[Lamp, Outcome, Nothing](LoopSecond.rebind(flip ~> darkStep))
object LoopSecond extends Derived[Lamp, Outcome, Nothing](LoopFirst.rebind(flip ~> lampStep))

/** Two machines that are aliases of each other. */
val aliasFirst: Machine[Lamp, Outcome, Nothing] = aliasSecond
val aliasSecond: Machine[Lamp, Outcome, Nothing] = aliasFirst
object Aliased extends Composition[Lamps](_.left -> aliasFirst, _.right -> OneStart):
  def end(s: State) = false
  object syncs extends Syncs

given Ok[Outcome] = Ok(Outcome.accepted)

/** `in` over a list passed whole, which names no members. */
val lit = List(true)
def litStep(l: Lamp): List[Step[Lamp, Outcome, Nothing]] =
  if l.lit.in(false, lit*) then enter(Lamp(true)) else disabled
object Splatted extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> litStep)

/** An explanation given to steps a helper function returns, not written out. */
def explainedStep(l: Lamp): List[Step[Lamp, Outcome, Nothing]] = lampStep(l).because("it flips")
object Explained extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> explainedStep)

/** An ok outcome a helper function computes, which `enter` cannot read. */
object ComputedOk:
  def okOf(o: Outcome): Ok[Outcome] = Ok(o)
  given Ok[Outcome] = okOf(Outcome.accepted)
  def computedStep(l: Lamp): List[Step[Lamp, Outcome, Nothing]] = enter(Lamp(!l.lit))
  object ComputedOk extends Machine[Lamp, Outcome, Nothing]:
    val init = Lamp(false)
    def end(s: State) = true
    object rules extends Bindings(flip ~> computedStep)

// ### Typed composition selectors (fn-112.4)

final case class Trio(left: Lamp, right: Lamp, spare: Lamp)

enum Dim derives Finite:
  case low, high

val dim = action(Party("fixture")).input[Dim]("level")
def dimStep(l: Lamp, level: Dim): List[Step[Lamp, Outcome, Nothing]] = List(
  Step(Outcome.accepted, Lamp(true))
)

/** A lamp that flips, dims and is pushed, and one that taps TapA's action spelled as TapB's. */
object Dimmer extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> lampStep, dim ~> dimStep, push ~> lampStep)
object TapLamp extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(TapA.tap ~> lampStep)

/** A member named by a selector of a field's field. */
object MemberNoField extends Composition[Lamps](_.left.lit -> OneStart, _.right -> OneStart):
  def end(s: State) = false
  object syncs extends Syncs

/** A sync whose selector names a field no member fills. */
object SyncNoMember extends Composition[Trio](_.left -> OneStart, _.right -> Dimmer):
  def end(s: State) = false
  object syncs extends Syncs:
    sync("flipSpare", _.left -> flip, _.spare -> flip)

/** A member whose machine is of another state type than its field. */
object MemberIncompatible extends Composition[Lamps](_.left -> OneStart, _.right -> First):
  def end(s: State) = false
  object syncs extends Syncs

/** A sync of an action spelled as the one its member binds, of another declaration. */
object SyncUnbound extends Composition[Lamps](_.left -> TapLamp, _.right -> OneStart):
  def end(s: State) = false
  object syncs extends Syncs:
    sync("tapFlip", _.left -> TapB.tap, _.right -> flip)

/** A member replaced where no member fills the field, or by a machine of another state type. */
object Spareless extends Composition[Trio](_.left -> OneStart, _.right -> Dimmer):
  def end(s: State) = false
  object syncs extends Syncs
object WithNoMember extends Composition(Spareless.withMember(_.spare -> OneStart))
object WithIncompatible extends Composition(Spareless.withMember(_.left -> First))

/** `synced` and `own` of a field no member fills, and `synced` of a field's field. */
val sparelessLit = Spareless.property holds (after => after.state.left.lit)
val sparelessStart = Trio(Lamp(false), Lamp(false), Lamp(false))
val syncedSpare =
  Spareless.scenario.starts(sparelessStart).actions(Spareless.synced(_.spare -> flip))
val syncedNoMember: Query = query verify sparelessLit in syncedSpare limits one
val ownSpare = Spareless.scenario.starts(sparelessStart).actions(Spareless.own(_.spare, flip))
val ownNoMember: Query = query verify sparelessLit in ownSpare limits one

/** A member that stands in for a machine, replaced by a machine that refines none. */
object StandIn extends Composition[Lamps](_.left -> FlagLamp, _.right -> OneStart):
  def end(s: State) = false
  object syncs extends Syncs:
    replaces(_.left, First)
object WithUnrefined extends Composition(StandIn.withMember(_.left -> OneStart))

/** A member a sync pairs, replaced by a machine that binds none of the actions it pairs. */
object FlipPair extends Composition[Lamps](_.left -> OneStart, _.right -> Dimmer):
  def end(s: State) = false
  object syncs extends Syncs:
    sync("flipBoth", _.left -> flip, _.right -> flip)
object WithUnsynced extends Composition(FlipPair.withMember(_.left -> TapLamp))

val flipPairLit = FlipPair.property holds (after => after.state.left.lit)
val flipPairFree = FlipPair.scenario.free

/** `synced` of a member action no sync pairs. */
val unsyncedPush = FlipPair.scenario.actions(FlipPair.synced(_.right -> push))
val syncedNone: Query = query verify flipPairLit in unsyncedPush limits one

/** `synced` of a member action two syncs pair. */
object FlipTrio
    extends Composition[Trio](_.left -> OneStart, _.right -> OneStart, _.spare -> OneStart):
  def end(s: State) = false
  object syncs extends Syncs:
    sync("flipRight", _.left -> flip, _.right -> flip)
    sync("flipSpare", _.left -> flip, _.spare -> flip)
val flipTrioLit = FlipTrio.property holds (after => after.state.left.lit)
val eitherFlip = FlipTrio.scenario.actions(FlipTrio.synced(_.left -> flip))
val syncedTwice: Query = query verify flipTrioLit in eitherFlip limits one

// ### Names a declaration takes by default (fn-112.10)

/** A sync named after its first member's action, which another sync of the composition is named. */
object SyncNamedTwice
    extends Composition[Trio](_.left -> OneStart, _.right -> OneStart, _.spare -> OneStart):
  def end(s: State) = false
  object syncs extends Syncs:
    sync("flip", _.left -> flip, _.right -> flip)
    sync(_.left -> flip, _.spare -> flip)

object emptied:
  /** A count whose range ends below its start, so it has no values. */
  opaque type Unreached = Int

  object Unreached:
    given Finite[Unreached] = Finite.upTo(-1)

val unreached = input[emptied.Unreached]
val reach = action(Party("fixture")).input(unreached)
def reachStep(l: Lamp, u: emptied.Unreached): List[Step[Lamp, Outcome, Nothing]] = Nil
object ReachLamp extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(reach ~> reachStep)
val reachLit = ReachLamp.property holds (after => after.state.lit)

/** `reach()`, whose input has no first value to take. */
val reachOnce = ReachLamp.scenario.actions(reach())
val omittedEmpty: Query = query verify reachLit in reachOnce limits one

/** A party no val declares, which states no name. */
val partyUnnamed = action(Party())
object PartyUnnamedLamp extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(partyUnnamed ~> lampStep)

/** A monitor a Query's expected Run names by a def, which no val declares. */
def flipWatch: Monitor[Flag, Outcome, Nothing, Boolean] =
  monitor[Flag, Outcome, Nothing, Boolean](false)((seen, _, _) => seen)(seen => seen)
val watchUnnamed: Query = (query verify flips in secondFlips limits one).expect(
  umpire.realize.RunExpectation(
    umpire.realize.Conformance.conformant,
    umpire.realize.PropertyOutcome.satisfied,
    umpire.realize.PropertyOutcome.satisfied,
    umpire.realize.Disposition.completed,
    umpire.realize.Cleanup.succeeded,
    monitors =
      Vector(umpire.realize.MonitorExpectation(flipWatch, umpire.realize.PropertyOutcome.satisfied))
  )
)

/** A monitor a Query's expected Run names by value, which no machine watches. */
val unwatched: Monitor[Flag, Outcome, Nothing, Boolean] =
  monitor[Flag, Outcome, Nothing, Boolean](false)((seen, _, _) => seen)(seen => seen)
val watchUnwatched: Query = (query verify flips in secondFlips limits one).expect(
  umpire.realize.RunExpectation(
    umpire.realize.Conformance.conformant,
    umpire.realize.PropertyOutcome.satisfied,
    umpire.realize.PropertyOutcome.satisfied,
    umpire.realize.Disposition.completed,
    umpire.realize.Cleanup.succeeded,
    monitors =
      Vector(umpire.realize.MonitorExpectation(unwatched, umpire.realize.PropertyOutcome.satisfied))
  )
)

/**
 * A monitor a Query's expected Run names by value, which another machine watches.
 */
val elsewhereWatch: Monitor[Flag, Outcome, Nothing, Boolean] =
  monitor[Flag, Outcome, Nothing, Boolean](false)((seen, _, _) => seen)(seen => seen)
object WatchingElsewhere extends Machine[Flag, Outcome, Nothing]:
  val init = Flag(false)
  def end(s: State) = true
  object monitors extends Section:
    val watch = elsewhereWatch
  object rules extends Bindings(flip ~> flipStep)
val elsewhereFlips: Property[Flag] =
  WatchingElsewhere.property holds (after => after.outcome == Outcome.accepted)
val watchElsewhere: Vector[Query] = Vector(
  query("watchedThere") verify elsewhereFlips in WatchingElsewhere
    .scenario("there")
    .starts(Flag(false))
    .actions(flip) limits one total 2,
  (query("watchedHere") verify flips in secondFlips limits one total 2).expect(
    umpire.realize.RunExpectation(
      umpire.realize.Conformance.conformant,
      umpire.realize.PropertyOutcome.satisfied,
      umpire.realize.PropertyOutcome.satisfied,
      umpire.realize.Disposition.completed,
      umpire.realize.Cleanup.succeeded,
      monitors = Vector(
        umpire.realize.MonitorExpectation(elsewhereWatch, umpire.realize.PropertyOutcome.satisfied)
      )
    )
  )
)

/** `own` of a member action a sync pairs. */
val ownFlip = FlipPair.scenario.actions(FlipPair.own(_.left, flip))
val ownSynced: Query = query verify flipPairLit in ownFlip limits one

val syncedLit = FlipPair.scenario.actions(FlipPair.synced(_.left.lit -> flip))
val syncedNoField: Query = query verify flipPairLit in syncedLit limits one

/** `own` of an action spelled as the one its member binds, of another declaration. */
object TapPair extends Composition[Lamps](_.left -> TapLamp, _.right -> OneStart):
  def end(s: State) = false
  object syncs extends Syncs
val tapPairLit = TapPair.property holds (after => after.state.left.lit)
val tapOther = TapPair.scenario.actions(TapPair.own(_.left, TapB.tap))
val ownSpelledAlike: Query = query verify tapPairLit in tapOther limits one

/** A composition's Scenario that lists a machine's class beside its composed classes. */
val mixedFlips = FlipPair.scenario.actions(FlipPair.synced(_.left -> flip), push)
val mixedSchedule: Query = query verify flipPairLit in mixedFlips limits one

/** A composition's Scenario that lists an action that takes inputs, not a class of it. */
val bareDim = FlipPair.scenario.actions(FlipPair.own(_.right, dim))
val bareInputs: Query = query verify flipPairLit in bareDim limits one

/** `whenAction` of a class, with its inputs, where it names every class of an action. */
val dimmedHigh = FlipPair.property.whenAction(FlipPair.own(_.right, dim(Dim.high))) holds
  (after => after.state.right.lit)
val whenClassInputs: Query = query verify dimmedHigh in flipPairFree limits one

/** A typed replacement of a field no member fills. */
object ReplacesSpare extends Composition[Trio](_.left -> FlagLamp, _.right -> OneStart):
  def end(s: State) = false
  object syncs extends Syncs:
    replaces(_.spare, First)

/** Two compositions derived from each other. */
object LoopPair extends Composition[Lamps](LoopPairBack.withMember(_.left -> OneStart))
object LoopPairBack extends Composition[Lamps](LoopPair.withMember(_.right -> OneStart))

// ### Claim patterns, records over a member and function-valued arguments (fn-112.4)

def lampLit(l: Lamp): Boolean = l.lit
def lampStepLit(after: Step[Lamp, Outcome, Nothing]): Boolean = after.state.lit

/** A claim pattern after `when`: a transition Property is about every step. */
val litAfterFlip = OneStart.property.when(flip).stays(lampLit)
val whenStays: Query = query verify litAfterFlip in lampFlips limits one

/** A `keeps` projection that computes a value rather than reading a field. */
val litKept = OneStart.property.once(lampLit).keeps(l => !l.lit)
val keptComputed: Query = query verify litKept in lampFlips limits one

/** `from` on a `never` kept in a val, not written directly after it. */
val neverLit = OneStart.property.never(lampStepLit)
val fromKept = neverLit.from(lampLit)
val fromVal: Query = query verify fromKept in lampFlips limits one

/** A lambda literal where a claim written once takes a predicate, which names no def to bind. */
def litStays[S](m: Declares[S])(lit: S => Boolean): Property[S] = m.property("litStays").stays(lit)
val litStaysLambda = litStays(OneStart)(l => l.lit)
val sharedLambda: Query = query verify litStaysLambda in lampFlips limits one

/** A composition's records whose selector names no member, and a fact its member does not record. */
object PatternLamps extends Composition[Lamps](_.left -> OneStart, _.right -> OneStart):
  def end(s: State) = false
  object syncs extends Syncs
val patternLampsFree = PatternLamps.scenario.free
val leftLitRecorded = PatternLamps.property holds (after => after.records(_.left.lit, Note.ping))
val recordsNoMember: Query = query verify leftLitRecorded in patternLampsFree limits one
val leftPinged = PatternLamps.property holds (after => after.records(_.left, Note.ping))
val recordsForeignFact: Query = query verify leftPinged in patternLampsFree limits one

/** A composition's records of a field no member fills. */
final case class SparedState(left: Lamp, spare: Lamp)
object Spared extends Composition[SparedState](_.left -> OneStart):
  def end(s: State) = false
  object syncs extends Syncs
val sparedFree = Spared.scenario.starts(SparedState(Lamp(false), Lamp(false))).free
val sparePinged = Spared.property holds (after => after.records(_.spare, Note.ping))
val recordsUnfilled: Query = query verify sparePinged in sparedFree limits one

/** A call of a function parameter in a function no declaring function binds it in. */
def litBy(after: Step[Lamp, Outcome, Nothing], lit: Lamp => Boolean): Boolean = lit(after.state)
val litByParam = OneStart.property holds (after => litBy(after, lampLit))
val paramCalled: Query = query verify litByParam in lampFlips limits one

// ### Input tokens, inputs supplied by name and bounded counters (fn-112.5)

val level = input[Dim]
val glow = input[Dim]
val shade = input[Dim]
val bright = action(Party("fixture")).input(level).input(glow)
val tint = action(Party("fixture")).input(shade)
def brightStep(l: Lamp, a: Dim, b: Dim): List[Step[Lamp, Outcome, Nothing]] =
  List(Step(Outcome.accepted, Lamp(a == b)))
def tintStep(l: Lamp, a: Dim): List[Step[Lamp, Outcome, Nothing]] =
  List(Step(Outcome.accepted, Lamp(a == Dim.high)))
object BrightLamp extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(bright ~> brightStep, tint ~> tintStep, dim ~> dimStep)
val brightLit = BrightLamp.property holds (after => after.state.lit)

/** A token of another action, of the input type this one takes. */
val foreignBright = BrightLamp.scenario.actions(bright(shade := Dim.high))
val foreignToken: Query = query verify brightLit in foreignBright limits one

/** One input supplied twice. */
val twiceBright = BrightLamp.scenario.actions(bright(level := Dim.high, level := Dim.low))
val suppliedTwice: Query = query verify brightLit in twiceBright limits one

/** A supply kept in a val, not written in the call. */
val keptLevel = level := Dim.high
val keptBright = BrightLamp.scenario.actions(bright(keptLevel))
val supplyKept: Query = query verify brightLit in keptBright limits one

/** An action whose inputs are declared by name strings, given a token. */
val strungDim = BrightLamp.scenario.actions(dim(level := Dim.high))
val namedNoTokens: Query = query verify brightLit in strungDim limits one

/** An action that declares one token twice. */
val doubleTint = action(Party("fixture")).input(shade).input(shade)
def doubleTintStep(l: Lamp, a: Dim, b: Dim): List[Step[Lamp, Outcome, Nothing]] =
  List(Step(Outcome.accepted, Lamp(a == b)))
object DoubleTintLamp extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(doubleTint ~> doubleTintStep)
val doubleTintLit = DoubleTintLamp.property holds (after => after.state.lit)
val doubleTintFree = DoubleTintLamp.scenario.free
val inputTwice: Query = query verify doubleTintLit in doubleTintFree limits one

/** An input token no val declares, so it has no name. */
val unnamedTint = action(Party("fixture")).input(input[Dim])
object UnnamedTintLamp extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(unnamedTint ~> tintStep)
val unnamedTintLit = UnnamedTintLamp.property holds (after => after.state.lit)
val unnamedTintFree = UnnamedTintLamp.scenario.free
val tokenUnnamed: Query = query verify unnamedTintLit in unnamedTintFree limits one

/** A counter bounded below zero, which has no values. */
final case class BelowState(count: UpTo[-1]) derives Finite
object Below extends Machine[BelowState, Outcome, Nothing]:
  val init = BelowState(UpTo(0))
  def end(s: State) = true
  object rules extends Bindings(flip ~> (b => List(Step(Outcome.accepted, b))))
val belowFree = Below.scenario.free
val belowAny = Below.property holds (after => after.state.count == 0)
val upToNegative: Query = query verify belowAny in belowFree limits one

// ### fn-112.11: the total each Query asserts (2 states of Flag x 1 scheduled flip = 2)

val twoStates: Int = 2

/** A Query that asserts no total. */
val untotaled: Query = query verify flips in secondFlips limits one

/** A Query that asserts its total twice. */
val totaledTwice: Query = query verify flips in secondFlips limits one total 2 total 2

/** A total the lifter would have to compute. */
val totalComputed: Query = query verify flips in secondFlips limits one total (twoStates * 1)

/** A total kept in a val. */
val totalKept: Query = query verify flips in secondFlips limits one total twoStates

/** A total below zero. */
val totalNegative: Query = query verify flips in secondFlips limits one total -2

/** A shared def's total computed at its call. */
def flipQueries(m: Machine[Flag, Outcome, Nothing], total: Int): Vector[Query] = Vector(
  query(s"${m.name}.flipsAgain") verify flips in m
    .scenario("flipsAgain")
    .actions(flip) limits one total total
)
val sharedComputed: Vector[Query] = flipQueries(Second, twoStates + 0)

// ### Named choices (fn-120.1)

type LampStep = Step[Lamp, Outcome, Nothing]

val lampOn = choice
val lampOff = choice
object Elsewhere:
  val lampOn = choice

/** One token naming two alternatives. */
def chosenTwiceStep(l: Lamp): List[LampStep] =
  choose(lampOn -> stay(l), lampOn -> enter(Lamp(!l.lit)))
object ChosenTwice extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> chosenTwiceStep)

/** Two tokens whose vals have one simple name. */
def spelledTwiceStep(l: Lamp): List[LampStep] =
  choose(lampOn -> stay(l), Elsewhere.lampOn -> enter(Lamp(!l.lit)))
object SpelledTwice extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> spelledTwiceStep)

/** An alternative whose function gives two steps, a choose of its own (lampBothStep, below). */
def choiceHelperStep(l: Lamp): List[LampStep] =
  choose(lampOn -> lampBothStep(l), lampOff -> stay(l))
object ChoiceHelper extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> choiceHelperStep)

/** An alternative with no step. */
def choiceDisabledStep(l: Lamp): List[LampStep] =
  choose(lampOn -> enter(Lamp(true)), lampOff -> disabled)
object ChoiceDisabled extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> choiceDisabledStep)

/** An alternative of two steps. */
def choiceTwoStepsStep(l: Lamp): List[LampStep] = choose(
  lampOn -> List(Step(Outcome.accepted, Lamp(true)), Step(Outcome.accepted, Lamp(false))),
  lampOff -> stay(l)
)
object ChoiceTwoSteps extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> choiceTwoStepsStep)

/** An alternative that is a conditional. */
def choiceIfStep(l: Lamp): List[LampStep] = choose(
  lampOn -> (if l.lit then stay(l) else enter(Lamp(true))),
  lampOff -> stay(l)
)
object ChoiceIf extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> choiceIfStep)

/** Choice tokens no val declares, so they have no name. */
def choiceUnnamedStep(l: Lamp): List[LampStep] =
  choose(choice -> stay(l), choice -> enter(Lamp(true)))
object ChoiceUnnamed extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> choiceUnnamedStep)

/** An alternative kept in a val, not written in the call. */
val keptOn: (Choice, List[LampStep]) = lampOn -> List(Step(Outcome.accepted, Lamp(true)))
def choiceKeptStep(l: Lamp): List[LampStep] = choose(keptOn, lampOff -> stay(l))
object ChoiceKept extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> choiceKeptStep)

// ### Claims a shared def declares together, as a bundle read back by field (fn-112.12)

/** A case class one field of which is no claim: it bundles none, so building it declares nothing. */
final case class Mixed(flipped: Property[Flag], count: Int)

def mixedLaws(m: Machine[Flag, Outcome, Nothing]): Mixed =
  Mixed(m.property("mixedFlips") holds (after => after.outcome == Outcome.accepted), 1)
def mixedQueries(m: Machine[Flag, Outcome, Nothing]): Vector[Query] =
  val laws = mixedLaws(m)
  Vector(query(s"${m.name}.mixedFlips") verify laws.flipped in secondFlips limits one total 2)
val mixedBundle: Vector[Query] = mixedQueries(Second)

// ### A type moved under a DefinitionScope keeps its former name (MovedRejects.scala, fn-112.12)

/** The gauge fixture.rejects keeps, whose name the Gauge MovedRejects.scala moved out still takes. */
enum Gauge derives Finite:
  case low, high

final case class Gauges(kept: Gauge, taken: moved.Gauge) derives Finite

val gaugeTick = action(Party("fixture"))
def gaugeStep(g: Gauges): List[Step[Gauges, Outcome, Nothing]] = List(Step(Outcome.accepted, g))

/** One machine whose state reads both gauges, which the IR would name alike. */
object MovedNameTaken extends Machine[Gauges, Outcome, Nothing]:
  val init = Gauges(Gauge.low, moved.Gauge.empty)
  def end(s: State) = true
  object rules extends Bindings(gaugeTick ~> gaugeStep)

// ### Functions a choose calls, and unnamed branching (fn-120.2)

/** A function a choose calls that gives two steps, each named already. */
def lampBothStep(l: Lamp): List[LampStep] = choose(lampOn -> stay(l), lampOff -> enter(Lamp(true)))

/** A function a choose calls that gives a step it keeps in a val, not one written out. */
def lampKeptStep(l: Lamp): List[LampStep] =
  val kept = enter(Lamp(!l.lit))
  if l.lit then kept else disabled
def choiceKeptHelperStep(l: Lamp): List[LampStep] =
  choose(lampOn -> lampKeptStep(l), lampOff -> stay(l))
object ChoiceKeptHelper extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> choiceKeptHelperStep)

/** Two results written as an unnamed list. */
def unnamedListStep(l: Lamp): List[LampStep] =
  List(Step(Outcome.accepted, l), Step(Outcome.accepted, Lamp(!l.lit)))
object UnnamedList extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> unnamedListStep)

/** Two results joined with `++`, here two helpers' steps. */
def unnamedJoinStep(l: Lamp): List[LampStep] = lampStep(l) ++ stay(l)
object UnnamedJoin extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> unnamedJoinStep)

/** An unnamed list in a function a step function calls, refused where it is written. */
def lampPairStep(l: Lamp): List[LampStep] =
  List(Step(Outcome.accepted, l), Step(Outcome.accepted, l))
def unnamedInHelperStep(l: Lamp): List[LampStep] = if l.lit then lampPairStep(l) else disabled
object UnnamedInHelper extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> unnamedInHelperStep)

// ### Expressions at the wrong level (fn-120 R13): a step made where no step function is

/** A start computed from a step. */
object LevelStart extends Machine[Lamp, Outcome, Nothing]:
  val init = if lampStep(Lamp(false)) == Nil then Lamp(true) else Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> lampStep)

/** An `end` that asks whether a step function gives a step. */
object LevelEnds extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(l: State) = lampStep(l) == Nil
  object rules extends Bindings(flip ~> lampStep)

/** Evidence that makes a step. */
object LevelEvidence extends Machine[Lamp, Outcome, Dropped]:
  val init = Lamp(false)
  def end(s: State) = true
  val evidence: PartialFunction[Dropped, String] = {
    case Dropped.kept    => if lampStep(Lamp(false)) == Nil then "keptData" else "keptAgain"
    case Dropped.lost(_) => "lostData"
  }
  object rules extends Bindings()

/** A refinement whose map reads a step. */
object LevelRefinement extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object refinement extends Refinement(OneStart):
    def toProduct(l: Lamp) = if lampStep(l) == Nil then l else Lamp(!l.lit)
  object rules extends Bindings(flip ~> lampStep)

/** A monitor whose next state asks which steps the step function gives. */
val levelWatch: Monitor[Lamp, Outcome, Nothing, Boolean] =
  monitor[Lamp, Outcome, Nothing, Boolean](false)((seen, before, after) =>
    seen || !lampStep(before).contains(after)
  )(seen => seen)
object LevelMonitor extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object monitors extends Section:
    val watch = levelWatch
  object rules extends Bindings(flip ~> lampStep)

/** A step function whose precondition asks whether another step function gives a step. */
def requiringStep(l: Lamp): List[LampStep] =
  require(lampStep(l) != Nil)
  lampStep(l)
object LevelRequire extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(flip ~> requiringStep)

/** A same-step Property that asks what the step function gives after the step. */
val levelHolds = OneStart.property holds (after => lampStep(after.state) == Nil)
val levelProperty: Query = query verify levelHolds in lampFlips limits one total 2

/** A transition Property that asks whether the step is one the step function gives. */
val levelAcross =
  OneStart.property holdsAcross ((before, after) => lampStep(before).contains(after))
val levelTransition: Query = query verify levelAcross in lampFlips limits one total 2

/** A claim pattern whose predicate makes a step. */
def lampStepped(after: Step[Lamp, Outcome, Nothing]): Boolean = lampStep(after.state) == Nil
val levelNever = OneStart.property.never(lampStepped)
val levelPattern: Query = query verify levelNever in lampFlips limits one total 2

/** A progress claim whose source asks whether a step function gives a step. */
val levelProgress = OneStart.leadsTo("levelSettles")(l => lampStep(l) == Nil, l => l.lit, 2)

/** A composition whose `end` asks whether a member's step function gives a step. */
object LevelComposition extends Composition[Lamps](_.left -> OneStart, _.right -> OneStart):
  def end(c: State) = lampStep(c.left) == Nil
  object syncs extends Syncs

/** A Scenario whose start is computed from a step. */
val levelStarted =
  OneStart.scenario.starts(if lampStep(Lamp(false)) == Nil then Lamp(true) else Lamp(false))
val levelLit = OneStart.property holds (after => after.state.lit)
val levelScenario: Query =
  query verify levelLit in levelStarted.actions(flip) limits one total 2

// ### Sections (fn-126 R14): transparent to Definition IDs, so where one may sit is narrow

/** A section in a section: a section sits at a file's top level or in a machine's object. */
object outerSection extends Section:
  object innerSection extends Section:
    val nestedTick = action(Party("fixture"))

object SectionNested extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(outerSection.innerSection.nestedTick ~> lampStep)

/** A section in an object that holds no machine. */
object Holder:
  object heldSection extends Section:
    val heldTick = action(Party("fixture"))

object SectionMisplaced extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(Holder.heldSection.heldTick ~> lampStep)

/** Two sections of one owner, each with an action of one name, which would share its ID. */
object leftHand extends Section:
  val clap = action(Party("fixture"))

object rightHand extends Section:
  val clap = action(Party("fixture"))

object SectionTwins extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(leftHand.clap ~> lampStep, rightHand.clap ~> lampStep)

/** A section that pins: its members take the IDs of the owner it stands in. */
object pinningSection extends Section:
  given DefinitionScope = DefinitionScope("fixture.rejects.Former$package$")
  val pinnedTick = action(Party("fixture"))

object SectionPinned extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object rules extends Bindings(pinningSection.pinnedTick ~> lampStep)

// ### Machine objects, rules and effects (fn-126 R15, R16): each refused at its line

enum Glow derives Finite:
  case dim, bright

final case class Bulb(glow: Glow) derives Finite

final case class Bulbs(a: Bulb, b: Bulb)

object bulbHand extends Actor:
  val squeeze = action(this)
  val twist = action(this)

/** A machine whose squeeze brightens a dim bulb and dims a bright one: two rules of squeeze. */
object Ruled extends Machine[Bulb, Outcome, Nothing]:
  val init = Bulb(Glow.dim)
  def end(s: Bulb) = true
  object effects extends Section:
    def brighten(s: Bulb) = enter[Bulb, Outcome, Nothing](Bulb(Glow.bright))
    def darken(s: Bulb) = enter[Bulb, Outcome, Nothing](Bulb(Glow.dim))
  object rules extends Rules(_.glow):
    in(Glow.dim)(bulbHand.squeeze ~> effects.brighten)
    in(Glow.bright)(bulbHand.squeeze ~> effects.darken)

/** A rule whose effect is no def of `effects`. */
object EffectOutside extends Machine[Bulb, Outcome, Nothing]:
  val init = Bulb(Glow.dim)
  def end(s: Bulb) = true
  def brighten(s: Bulb) = enter[Bulb, Outcome, Nothing](Bulb(Glow.bright))
  object rules extends Rules:
    when(_ => true)(bulbHand.squeeze ~> brighten)

/** An effect that gives no step in one branch: the rules say where it fires. */
object EmptyEffect extends Machine[Bulb, Outcome, Nothing]:
  val init = Bulb(Glow.dim)
  def end(s: Bulb) = true
  object effects extends Section:
    def brighten(s: Bulb) =
      if s.glow == Glow.bright then disabled else enter[Bulb, Outcome, Nothing](Bulb(Glow.bright))
  object rules extends Rules:
    when(_ => true)(bulbHand.squeeze ~> effects.brighten)

/** A rule under no heading. */
object Unheaded extends Machine[Bulb, Outcome, Nothing]:
  val init = Bulb(Glow.dim)
  def end(s: Bulb) = true
  object rules extends Rules:
    bulbHand.squeeze ~> Ruled.effects.brighten

/** A heading under a heading. */
object HeadingTwice extends Machine[Bulb, Outcome, Nothing]:
  val init = Bulb(Glow.dim)
  def end(s: Bulb) = true
  object rules extends Rules(_.glow):
    when(_ => true) {
      in(Glow.dim)(bulbHand.squeeze ~> Ruled.effects.brighten)
    }

/** An action both disabled and fired by a rule. */
object DisabledFired extends Machine[Bulb, Outcome, Nothing]:
  val init = Bulb(Glow.dim)
  def end(s: Bulb) = true
  object rules extends Rules(_.glow):
    disabled(bulbHand.squeeze)
    in(Glow.dim)(bulbHand.squeeze ~> Ruled.effects.brighten)

/** A bare binding where the source's actions are bound by rules: extend takes rules. */
object ExtendedBare extends Derived(Ruled.extend(bulbHand.twist ~> Ruled.effects.darken))

/** One effect in place of two rules' different effects. */
object RebindSeveral extends Derived(Ruled.rebind(bulbHand.squeeze ~> Ruled.effects.darken))

/** Rules for an action class the source does not bind. */
object RebindUnbound
    extends Derived(Ruled.rebind(when(_ => true)(bulbHand.twist ~> Ruled.effects.darken)))

/** A derivation of the ruled bulb, the member `EndedTwice` puts in place of another. */
object Colliding extends Derived(Ruled.restrict(bulbHand.squeeze))

/** A composition object that says nowhere where it ends. */
object Endless extends Composition[Bulbs](_.a -> Ruled, _.b -> Ruled):
  object syncs extends Syncs:
    sync(_.a -> bulbHand.squeeze, _.b -> bulbHand.squeeze)

object Paired extends Composition[Bulbs](_.a -> Ruled, _.b -> Ruled):
  def end(s: Bulbs) = true
  object syncs extends Syncs:
    sync(_.a -> bulbHand.squeeze, _.b -> bulbHand.squeeze)

/** A derived composition that declares its own end, which its source's is. */
object EndedTwice extends Composition(Paired.withMember(_.b -> Colliding)):
  def end(s: Bulbs) = false

/** A refinement's member written outside its `refinement` section. */
object LooseRefinement extends Machine[Bulb, Outcome, Nothing]:
  val init = Bulb(Glow.dim)
  def end(s: Bulb) = true
  def toProduct(s: Bulb) = s
  object rules extends Rules:
    when(_ => true)(bulbHand.squeeze ~> Ruled.effects.brighten)

/** A machine object whose name another package's object has too (CollidingRejects.scala). */
object Lookalike extends Machine[Bulb, Outcome, Nothing]:
  val init = Bulb(Glow.dim)
  def end(s: State) = true
  object rules extends Rules:
    when(_ => true)(bulbHand.squeeze ~> Ruled.effects.darken)

object LookalikePair extends Composition[Bulbs](_.a -> Lookalike, _.b -> elsewhere.Lookalike):
  def end(s: State) = true
  object syncs extends Syncs:
    sync(_.a -> bulbHand.squeeze, _.b -> bulbHand.squeeze)

/** An effect that gives no step, `Nil`, in one branch. */
object NilEffect extends Machine[Bulb, Outcome, Nothing]:
  val init = Bulb(Glow.dim)
  def end(s: State) = true
  object effects extends Section:
    def brighten(s: State) =
      if s.glow == Glow.bright then Nil else enter[Bulb, Outcome, Nothing](Bulb(Glow.bright))
  object rules extends Rules:
    when(_ => true)(bulbHand.squeeze ~> effects.brighten)

/** A monitor of another state type than the machine that watches it. */
val flagWatch =
  monitor[Flag, Outcome, Nothing, Boolean](false)((seen, _, _) => seen)(seen => seen)

object WatchesElsewhere extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: State) = true
  object monitors extends Section:
    val elsewhere = flagWatch
  object rules extends Bindings(flip ~> lampStep)
