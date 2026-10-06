// Bounded channels, lifted with their declarations: a FIFO channel that may deliver a message once
// more when its acknowledgment is lost, and an unordered one that may lose a message. The lifter's
// tests lift `Relay` and `Tallying` and compare the IR with expected/channels.json.
package fixture.channels

import umpire.*

enum Note derives Finite:
  case ping, pong

enum Signal derives Finite:
  case up, down

val wire: Channel[Note] =
  channel[Note](capacity = 2, order = Order.fifo, loss = Loss.reliable, duplicates = 1)
val radio: Channel[Signal] =
  channel[Signal](capacity = 1, order = Order.unordered, loss = Loss.lossy)

enum Heard derives Finite:
  case nothing, note, signal

// The relay's state, named apart from the machine object `Relay`.
final case class RelayState(heard: Heard, wire: Inbox[Note], radio: Inbox[Signal])

given Finite[RelayState] =
  given wireContents: Finite[Inbox[Note]] = wire.contents
  given radioContents: Finite[Inbox[Signal]] = radio.contents
  Finite.derived

enum Outcome derives Finite:
  case accepted, dropped

val talk = action(Actor("fixture")).input[Note]("note")
val flash = action(Actor("fixture")).input[Signal]("signal")

def talkStep(r: RelayState, n: Note): List[Step[RelayState, Outcome, Nothing]] =
  if r.wire.isFull then Nil else List(Step(Outcome.accepted, r.copy(wire = r.wire.send(n))))

def flashStep(r: RelayState, s: Signal): List[Step[RelayState, Outcome, Nothing]] =
  if r.radio.isEmpty then List(Step(Outcome.accepted, r.copy(radio = r.radio.send(s)))) else Nil

// Receiving a note: the channel already dropped the delivered message.
def hear(r: RelayState, n: Note): List[Step[RelayState, Outcome, Nothing]] =
  if n == Note.ping then List(Step(Outcome.accepted, r.copy(heard = Heard.note))) else Nil

def tune(r: RelayState, s: Signal): List[Step[RelayState, Outcome, Nothing]] =
  List(Step(Outcome.accepted, r.copy(heard = Heard.signal)))

def fade(r: RelayState, s: Signal): List[Step[RelayState, Outcome, Nothing]] = List(
  Step(Outcome.dropped, r)
)

object Relay extends Machine[RelayState, Outcome, Nothing]:
  val init = RelayState(Heard.nothing, wire.empty, radio.empty)
  def end(relay: State) = true

  object rules
      extends Bindings(
        talk ~> talkStep,
        flash ~> flashStep,
        wire.deliver ~> hear,
        radio.deliver ~> tune,
        radio.lose ~> fade
      )

// A channel of bounded integers: the range its declaration names is its catalog of messages.
val tally: Channel[Int] =
  channel[Int](capacity = 1, order = Order.fifo, loss = Loss.reliable)(using
    Finite.upTo(2)
  )

final case class Tally(heard: Heard, counts: Inbox[Int])

given Finite[Tally] =
  given Finite[Inbox[Int]] = tally.contents
  Finite.derived

val count = timer

def countStep(t: Tally): List[Step[Tally, Outcome, Nothing]] =
  if t.counts.isEmpty then List(Step(Outcome.accepted, t.copy(counts = t.counts.send(2)))) else Nil

def counted(t: Tally, n: Int): List[Step[Tally, Outcome, Nothing]] =
  if n == 2 then List(Step(Outcome.accepted, t.copy(heard = Heard.note))) else Nil

object Tallying extends Machine[Tally, Outcome, Nothing]:
  val init = Tally(Heard.nothing, tally.empty)
  def end(tally: State) = true

  object rules extends Bindings(count ~> countStep, tally.deliver ~> counted)
