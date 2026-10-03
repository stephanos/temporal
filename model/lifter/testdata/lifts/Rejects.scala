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
  query("crossedRead").verify(flips).in(flipping)(using Reads.through(refining, second)) limits one

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
  channel[Int]("counts", capacity = 1, order = Order.fifo, loss = Loss.reliable)(using Finite.of(1, 5))

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
    Finite.of(Nil, List(Note.ping)))

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
