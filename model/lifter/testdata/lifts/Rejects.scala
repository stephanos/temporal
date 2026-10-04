// Declarations the lifter must refuse before any IR is written, each at the line that declares it.
// The lifter's tests lift every root below, and one that names nothing, in one run, and compare the
// diagnostics with expected/rejects.txt.
package fixture.rejects

import umpire.*

val Family: umpire.Family = umpire.Family("fixture.rejects")

enum Note derives Finite:
  case ping

enum Outcome derives Finite:
  case accepted

val go = action("go", Party("fixture"))

/** A list in a state has no bound. */
final case class Unbounded(notes: List[Note])

given Finite[Unbounded] = Finite.of(Unbounded(Nil), Unbounded(List(Note.ping)))

val unbounded: Machine[Unbounded, Outcome, Nothing] =
  machine[Unbounded, Outcome, Nothing](Family, "unbounded") {
    starts(Unbounded(Nil))
    ends(_ => true)
    steps(go ~> (_ => Nil))
  }

/** A channel that holds nothing. */
val closed: Channel[Note] =
  channel[Note]("closed", capacity = 0, order = Order.fifo, loss = Loss.reliable)

final case class Waiting(inbox: Inbox[Note])

given Finite[Waiting] =
  given Finite[Inbox[Note]] = closed.contents
  Finite.derived

val waiting: Machine[Waiting, Outcome, Nothing] =
  machine[Waiting, Outcome, Nothing](Family, "waiting") {
    starts(Waiting(closed.empty))
    ends(_ => true)
    steps(closed.deliver ~> ((w, _) => List(Step(Outcome.accepted, w))))
  }

/** A lossy channel whose loss no step says the meaning of. */
val leaky: Channel[Note] =
  channel[Note]("leaky", capacity = 1, order = Order.fifo, loss = Loss.lossy)

final case class Listening(inbox: Inbox[Note])

given Finite[Listening] =
  given Finite[Inbox[Note]] = leaky.contents
  Finite.derived

val listening: Machine[Listening, Outcome, Nothing] =
  machine[Listening, Outcome, Nothing](Family, "listening") {
    starts(Listening(leaky.empty))
    ends(_ => true)
    steps(leaky.deliver ~> ((l, _) => List(Step(Outcome.accepted, l))))
  }

/** One channel held in two fields: which of them a delivery takes from is undefined. */
final case class Twice(first: Inbox[Note], second: Inbox[Note])

given Finite[Twice] =
  given Finite[Inbox[Note]] = leaky.contents
  Finite.derived

val doubled: Machine[Twice, Outcome, Nothing] =
  machine[Twice, Outcome, Nothing](Family, "doubled") {
    starts(Twice(leaky.empty, leaky.empty))
    ends(_ => true)
    steps(leaky.lose ~> ((t, _) => List(Step(Outcome.accepted, t))))
  }

/** A function that calls itself. */
def countdown(n: Int): Int = if n <= 0 then 0 else countdown(n - 1)

final case class Counter(n: Int)

given Finite[Counter] =
  given Finite[Int] = Finite.upTo(2)
  Finite.derived

val counter: Machine[Counter, Outcome, Nothing] =
  machine[Counter, Outcome, Nothing](Family, "counter") {
    starts(Counter(2))
    ends(_ => true)
    steps(go ~> (c => List(Step(Outcome.accepted, Counter(countdown(c.n))))))
  }

final case class Flag(on: Boolean) derives Finite

val flip = action("flip", Party("fixture"))

def flipStep(f: Flag): List[Step[Flag, Outcome, Nothing]] = List(
  Step(Outcome.accepted, Flag(!f.on))
)

val first: Machine[Flag, Outcome, Nothing] =
  machine[Flag, Outcome, Nothing](Family, "first") {
    starts(Flag(false))
    ends(_ => true)
    steps(flip ~> flipStep)
  }

val second: Machine[Flag, Outcome, Nothing] =
  machine[Flag, Outcome, Nothing](Family, "second") {
    starts(Flag(false))
    ends(_ => true)
    steps(flip ~> flipStep)
  }

val refining: Machine[Flag, Outcome, Nothing] =
  machine[Flag, Outcome, Nothing](Family, "refining") {
    refines(first)(f => f)
    starts(Flag(false))
    ends(_ => true)
    steps(flip ~> flipStep)
  }

val flips: Property[Flag] =
  second.property("flips") holds (after => after.outcome == Outcome.accepted)
val flipping: Scenario[Flag] = refining.scenario("flipping").starts(Flag(false)).actions(flip)
val secondFlips: Scenario[Flag] = second.scenario("secondFlips").starts(Flag(false)).actions(flip)
val one: Limits = Limits("one", steps = 1, actions = 1, search = 8)

/** A Property read through a refinement its Scenario's machine does not declare. */
val crossedRead: Query =
  query("crossedRead").verify(flips).in(flipping) limits one

/** Limits below zero. */
val backwards: Limits = Limits("backwards", steps = -1, actions = 1, search = 8)
val negative: Query = query("negative") verify flips in secondFlips limits backwards

/** Two monitors under one name would share one Definition ID. */
val twiceFirst: Monitor[Flag, Outcome, Nothing, Boolean] =
  monitor[Flag, Outcome, Nothing, Boolean]("twice", false)((seen, _, _) => seen)(seen => seen)
val twiceSecond: Monitor[Flag, Outcome, Nothing, Boolean] =
  monitor[Flag, Outcome, Nothing, Boolean]("twice", true)((seen, _, _) => seen)(seen => seen)

val watched: Machine[Flag, Outcome, Nothing] =
  machine[Flag, Outcome, Nothing](Family, "watched") {
    monitors(twiceFirst, twiceSecond)
    starts(Flag(false))
    ends(_ => true)
    steps(flip ~> flipStep)
  }

/** What a refined machine sees, on a machine that refines none. */
val unrefined: Machine[Flag, Outcome, Nothing] =
  machine[Flag, Outcome, Nothing](Family, "unrefined") {
    visible(_ => true)
    starts(Flag(false))
    ends(_ => true)
    steps(flip ~> flipStep)
  }

final case class Flags(left: Flag, right: Flag)

/** A replacement of a member the composition does not have. */
val misplaced: Composition[Flags] =
  compose[Flags](Family, "misplaced")("left" -> first, "right" -> refining)
    .replaces("middle", first)

/** The outcomes a refined machine sees, on a machine that refines none. */
val unrefinedOutcomes: Machine[Flag, Outcome, Nothing] =
  machine[Flag, Outcome, Nothing](Family, "unrefinedOutcomes") {
    visibleOutcomes(_ => true)
    starts(Flag(false))
    ends(_ => true)
    steps(flip ~> flipStep)
  }

/** Int messages whose catalog is not a range the IR can carry. */
val counts: Channel[Int] =
  channel[Int]("counts", capacity = 1, order = Order.fifo, loss = Loss.reliable)(using
    Finite.of(1, 5)
  )

final case class Counting(inbox: Inbox[Int])

given Finite[Counting] =
  given Finite[Inbox[Int]] = counts.contents
  Finite.derived

val counting: Machine[Counting, Outcome, Nothing] =
  machine[Counting, Outcome, Nothing](Family, "counting") {
    starts(Counting(counts.empty))
    ends(_ => true)
  }

/** List messages, which have no finite catalog in the IR. */
val batches: Channel[List[Note]] =
  channel[List[Note]]("batches", capacity = 1, order = Order.fifo, loss = Loss.reliable)(using
    Finite.of(Nil, List(Note.ping))
  )

final case class Batching(inbox: Inbox[List[Note]])

given Finite[Batching] =
  given Finite[Inbox[List[Note]]] = batches.contents
  Finite.derived

val batching: Machine[Batching, Outcome, Nothing] =
  machine[Batching, Outcome, Nothing](Family, "batching") {
    starts(Batching(batches.empty))
    ends(_ => true)
  }

/** A channel whose order is computed rather than named. */
val shuffled: Channel[Note] =
  channel[Note]("shuffled", capacity = 1, order = Order.fromOrdinal(1), loss = Loss.reliable)

final case class Shuffling(inbox: Inbox[Note])

given Finite[Shuffling] =
  given Finite[Inbox[Note]] = shuffled.contents
  Finite.derived

val shuffling: Machine[Shuffling, Outcome, Nothing] =
  machine[Shuffling, Outcome, Nothing](Family, "shuffling") {
    starts(Shuffling(shuffled.empty))
    ends(_ => true)
  }

/** A channel whose loss is computed rather than named. */
val guessed: Channel[Note] =
  channel[Note]("guessed", capacity = 1, order = Order.fifo, loss = Loss.valueOf("reliable"))

final case class Guessing(inbox: Inbox[Note])

given Finite[Guessing] =
  given Finite[Inbox[Note]] = guessed.contents
  Finite.derived

val guessing: Machine[Guessing, Outcome, Nothing] =
  machine[Guessing, Outcome, Nothing](Family, "guessing") {
    starts(Guessing(guessed.empty))
    ends(_ => true)
  }

// The fixtures below take their names from their vals, in the family the given names.
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
  val pinnedTwice = machine[Lamp, Outcome, Nothing] {
    starts(Lamp(false))
    ends(_ => true)
    steps(twiceTick ~> lampStep)
  }

/** A pin in an object nested in an object that pins its own. */
object PinsOuter:
  given DefinitionScope = DefinitionScope("fixture.rejects.Former$package$")
  object PinsInner:
    given DefinitionScope = DefinitionScope("fixture.rejects.Other$package$")
    val innerTick = action(Party("fixture"))
  val pinnedNested = machine[Lamp, Outcome, Nothing] {
    starts(Lamp(false))
    ends(_ => true)
    steps(PinsInner.innerTick ~> lampStep)
  }

/** Two owners pinned to one former owner, each with an action of one name. */
object SharedA:
  given DefinitionScope = DefinitionScope("fixture.rejects.Shared$")
  val share = action(Party("fixture"))

object SharedB:
  given DefinitionScope = DefinitionScope("fixture.rejects.Shared$")
  val share = action(Party("fixture"))

val sharedIds = machine[Lamp, Outcome, Nothing] {
  starts(Lamp(false))
  ends(_ => true)
  steps(SharedA.share ~> lampStep, SharedB.share ~> lampStep)
}

/** An owner pinned to itself. */
object Self:
  given DefinitionScope = DefinitionScope("fixture.rejects.Self$")
  val selfTick = action(Party("fixture"))
  val pinnedSelf = machine[Lamp, Outcome, Nothing] {
    starts(Lamp(false))
    ends(_ => true)
    steps(selfTick ~> lampStep)
  }

/** A pin computed rather than written as a literal. */
object Computed:
  given DefinitionScope = DefinitionScope("fixture.rejects." + "Former$package$")
  val computedTick = action(Party("fixture"))
  val pinnedComputed = machine[Lamp, Outcome, Nothing] {
    starts(Lamp(false))
    ends(_ => true)
    steps(computedTick ~> lampStep)
  }

/** Limits named after an anonymous given, a name the compiler made up. */
object Anonymous:
  given Limits = Limits(steps = 1, actions = 1, search = 8)
  val anonymous: Query = query("anonymous") verify flips in secondFlips limits summon[Limits]

/** A Query in a list, which no val names. */
val unnamedInList: Vector[Query] = Vector(query verify flips in secondFlips limits one)

/** A Property in the body of a function, which no val names. */
def unnamedChecks(m: Machine[Flag, Outcome, Nothing]): Vector[Query] = Vector(
  query("unnamedProperty") verify (m.property holds (after =>
    after.outcome == Outcome.accepted
  )) in secondFlips limits one
)
val unnamedProperty: Vector[Query] = unnamedChecks(second)

/** Two machines named alike in two objects. */
object TwinA:
  val twin = machine[Lamp, Outcome, Nothing] {
    starts(Lamp(false))
    ends(_ => true)
    steps(flip ~> lampStep)
  }

object TwinB:
  val twin = machine[Lamp, Outcome, Nothing] {
    starts(Lamp(true))
    ends(_ => true)
    steps(flip ~> lampStep)
  }

final case class Lamps(left: Lamp, right: Lamp)

val twins = compose[Lamps]("left" -> TwinA.twin, "right" -> TwinB.twin)

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

val tapped = machine[Lamp, Outcome, Nothing] {
  starts(Lamp(false))
  ends(_ => true)
  steps(TapA.tap ~> lampStep, TapB.tap ~> lampStep)
}

/** A Scenario that names no start, of a machine that declares two. */
val twoStarts = machine[Lamp, Outcome, Nothing] {
  starts(Lamp(false), Lamp(true))
  ends(_ => true)
  steps(flip ~> lampStep)
}
val twoStartsLit = twoStarts.property holds (after => after.state.lit)
val eitherStart = twoStarts.scenario.actions(flip)
val unstarted: Query = query find twoStartsLit in eitherStart limits one

/** A composition's Scenario that names no start, where a member declares two. */
val oneStart = machine[Lamp, Outcome, Nothing] {
  starts(Lamp(false))
  ends(_ => true)
  steps(flip ~> lampStep)
}
val startsPair = compose[Lamps]("left" -> twoStarts, "right" -> oneStart)
val pairLit = startsPair.property holds (after => after.state.left.lit)
val eitherPair = startsPair.scenario.free
val unstartedPair: Query = query find pairLit in eitherPair limits one

enum Dropped derives Finite:
  case kept
  case lost(hard: Boolean)

/** A machine that declares no evidence, of a fact with fields. */
val undeclared = machine[Lamp, Outcome, Dropped] {
  starts(Lamp(false))
  ends(_ => true)
}

/** Evidence that names no line for a fact with fields. */
val unlisted = machine[Lamp, Outcome, Dropped] {
  starts(Lamp(false))
  ends(_ => true)
  evidence { case Dropped.kept => "keptData" }
}

/** Evidence that covers only some values of a fact with fields. */
val partial = machine[Lamp, Outcome, Dropped] {
  starts(Lamp(false))
  ends(_ => true)
  evidence { case Dropped.lost(true) => "hardLoss" }
}

/** A Property read on a machine of another state type, which refines nothing. */
val lampFlips = oneStart.scenario.actions(flip)
val unrelatedRead: Query = query verify flips in lampFlips limits one

/** Two assumptions named alike in two objects, assumed by one machine. */
object OpaqueA:
  val opaque = assume

object OpaqueB:
  val opaque = assume

val assumedTwice = machine[Lamp, Outcome, Nothing] {
  assumes(OpaqueA.opaque, OpaqueB.opaque)
  starts(Lamp(false))
  ends(_ => true)
}

/** Two holes named alike in two objects, reached by one machine. */
object GapA:
  val gap = hole

object GapB:
  val gap = hole

def gapStep(l: Lamp): List[Step[Lamp, Outcome, Nothing]] =
  if l.lit then GapA.gap.reached else GapB.gap.reached

val gapTwice = machine[Lamp, Outcome, Nothing] {
  starts(Lamp(false))
  ends(_ => true)
  steps(flip ~> gapStep)
}

/** A derivation that rebinds an action its source does not bind. */
val push = action(Party("fixture"))
val rebindUnbound = oneStart.rebind(push ~> lampStep)

/** A derivation that extends its source by an action it binds already. */
val extendBound = oneStart.extend(flip ~> lampStep)

/** A derivation that binds one action twice. */
def darkStep(l: Lamp): List[Step[Lamp, Outcome, Nothing]] = List(
  Step(Outcome.accepted, Lamp(false))
)
val reboundTwice = oneStart.rebind(flip ~> lampStep, flip ~> darkStep)

/** A derivation that names an assumption its source assumes already. */
val lampOpaque = assume
val assumingLamp = machine[Lamp, Outcome, Nothing] {
  assumes(lampOpaque)
  starts(Lamp(false))
  ends(_ => true)
  steps(flip ~> lampStep)
}
val assumedAgain = assumingLamp.assuming(lampOpaque)

/** A derivation that names one assumption twice. */
val lampFaulty = assume
val assumingTwice = oneStart.assuming(lampFaulty, lampFaulty)

/** A refinement replaced where the source declares none. */
val refinedNothing = oneStart.refining(first)(l => Flag(l.lit))

/** A refinement replaced by one of a machine whose types differ from the replaced one's. */
enum Seen derives Finite:
  case seen

final case class Glimpse(at: Seen) derives Finite

val glimpse = machine[Glimpse, Seen, Nothing] {
  starts(Glimpse(Seen.seen))
  ends(_ => true)
}
val flagLamp = machine[Lamp, Outcome, Nothing] {
  refines(first)(l => Flag(l.lit))
  starts(Lamp(false))
  ends(_ => true)
  steps(flip ~> lampStep)
}
val refinedOtherwise = flagLamp.refining(glimpse)(_ => Glimpse(Seen.seen))

/** Two machines derived from each other. */
val loopFirst: Machine[Lamp, Outcome, Nothing] = loopSecond.rebind(flip ~> darkStep)
val loopSecond: Machine[Lamp, Outcome, Nothing] = loopFirst.rebind(flip ~> lampStep)

/** Two machines that are aliases of each other. */
val aliasFirst: Machine[Lamp, Outcome, Nothing] = aliasSecond
val aliasSecond: Machine[Lamp, Outcome, Nothing] = aliasFirst
val aliased = compose[Lamps]("left" -> aliasFirst, "right" -> oneStart)

given Accepted[Outcome] = Accepted(Outcome.accepted)

/** `in` over a list passed whole, which names no members. */
val lit = List(true)
def litStep(l: Lamp): List[Step[Lamp, Outcome, Nothing]] =
  if l.lit.in(false, lit*) then accept(Lamp(true)) else disabled
val splatted = machine[Lamp, Outcome, Nothing] {
  starts(Lamp(false))
  ends(_ => true)
  steps(flip ~> litStep)
}

/** An explanation given to steps a helper function returns, not written out. */
def explainedStep(l: Lamp): List[Step[Lamp, Outcome, Nothing]] = lampStep(l).because("it flips")
val explained = machine[Lamp, Outcome, Nothing] {
  starts(Lamp(false))
  ends(_ => true)
  steps(flip ~> explainedStep)
}

/** An accepted outcome a helper function computes, which `accept` cannot read. */
object ComputedAccepted:
  def acceptedOf(o: Outcome): Accepted[Outcome] = Accepted(o)
  given Accepted[Outcome] = acceptedOf(Outcome.accepted)
  def computedStep(l: Lamp): List[Step[Lamp, Outcome, Nothing]] = accept(Lamp(!l.lit))
  val computedAccept = machine[Lamp, Outcome, Nothing] {
    starts(Lamp(false))
    ends(_ => true)
    steps(flip ~> computedStep)
  }
