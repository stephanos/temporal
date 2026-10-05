// Bounded channels, lifted with their declarations: a FIFO channel that may deliver a message once
// more when its acknowledgment is lost, and an unordered one that may lose a message. The lifter's
// tests lift `relay` and compare the IR with expected/channels.json.
package fixture.channels

import umpire.*

given Family = Family("fixture.channels")

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

final case class Relay(heard: Heard, wire: Inbox[Note], radio: Inbox[Signal])

given Finite[Relay] =
  given wireContents: Finite[Inbox[Note]] = wire.contents
  given radioContents: Finite[Inbox[Signal]] = radio.contents
  Finite.derived

enum Outcome derives Finite:
  case accepted, dropped

val talk = action(Party("fixture")).input[Note]("note")
val flash = action(Party("fixture")).input[Signal]("signal")

def talkStep(r: Relay, n: Note): List[Step[Relay, Outcome, Nothing]] =
  if r.wire.isFull then Nil else List(Step(Outcome.accepted, r.copy(wire = r.wire.send(n))))

def flashStep(r: Relay, s: Signal): List[Step[Relay, Outcome, Nothing]] =
  if r.radio.isEmpty then List(Step(Outcome.accepted, r.copy(radio = r.radio.send(s)))) else Nil

/** Receiving a note: the channel already dropped the delivered message. */
def hear(r: Relay, n: Note): List[Step[Relay, Outcome, Nothing]] =
  if n == Note.ping then List(Step(Outcome.accepted, r.copy(heard = Heard.note))) else Nil

def tune(r: Relay, s: Signal): List[Step[Relay, Outcome, Nothing]] =
  List(Step(Outcome.accepted, r.copy(heard = Heard.signal)))

def fade(r: Relay, s: Signal): List[Step[Relay, Outcome, Nothing]] = List(Step(Outcome.dropped, r))

val relay = machine[Relay, Outcome, Nothing] {
  starts(Relay(Heard.nothing, wire.empty, radio.empty))
  ends(_ => true)
  steps(
    talk ~> talkStep,
    flash ~> flashStep,
    wire.deliver ~> hear,
    radio.deliver ~> tune,
    radio.lose ~> fade
  )
}

/** A channel of bounded integers: the range its declaration names is its catalog of messages. */
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

val tallying = machine[Tally, Outcome, Nothing] {
  starts(Tally(Heard.nothing, tally.empty))
  ends(_ => true)
  steps(count ~> countStep, tally.deliver ~> counted)
}
