// Union and presence, lifted as the finite types they are: an enum whose cases carry their own
// fields, an optional value, and a counter whose range is its own rather than the one every other Int
// field of the state shares. The tests lift `presence` and compare the IR with expected/presence.json.
package fixture.presence

import umpire.*

enum Result derives Finite:
  case succeeded, failed

/** A union: the second case carries fields the first does not. */
enum Report derives Finite:
  case unsent
  case sent(result: Result, retried: Boolean)

object bounded:
  /** A retry count, 0 to 2. */
  opaque type Retries = Int

  object Retries:
    given Finite[Retries] = Finite.upTo(2)
    val none: Retries = 0
    val most: Retries = 2
    extension (r: Retries) def next: Retries = (r + 1).min(2)

import bounded.Retries

/** `polls` takes the Int range the given below declares; `retries` keeps its own. */
final case class State(report: Report, kept: Option[Result], retries: Retries, polls: Int)

given Finite[State] =
  given Finite[Int] = Finite.upTo(1)
  Finite.derived

enum Outcome derives Finite:
  case accepted, ignored

val send = action("send", Party("fixture")).input[Result]("result")
val keep = action("keep", Party("fixture"))
val forget = action("forget", Party("fixture"))
val poll = timer("poll")

def sendStep(s: State, r: Result): List[Step[State, Outcome, Nothing]] = s.report match
  case Report.unsent        => List(Step(Outcome.accepted, s.copy(report = Report.sent(r, false))))
  case Report.sent(_, true) => Nil
  case Report.sent(first, false) =>
    if s.retries == Retries.most then Nil
    else
      List(
        Step(Outcome.accepted, s.copy(report = Report.sent(first, true), retries = s.retries.next))
      )

/**
 * Keeps what was sent. The match binds `r` inside a local `val`, and `None` takes its type from the
 * match.
 */
def keepStep(s: State): List[Step[State, Outcome, Nothing]] =
  val sent = s.report match
    case Report.sent(r, _) => Some(r)
    case Report.unsent     => None
  if sent == None || s.kept == sent then Nil else List(Step(Outcome.accepted, s.copy(kept = sent)))

def forgetStep(s: State): List[Step[State, Outcome, Nothing]] = s.kept match
  case Some(_) => List(Step(Outcome.accepted, s.copy(kept = None)))
  case None    => List(Step(Outcome.ignored, s))

def pollStep(s: State): List[Step[State, Outcome, Nothing]] =
  if s.polls == 0 then List(Step(Outcome.accepted, s.copy(polls = 1))) else Nil

val presence: Machine[State, Outcome, Nothing] =
  machine[State, Outcome, Nothing](Family("fixture.presence"), "presence") {
    starts(State(Report.unsent, None, Retries.none, 0))
    ends(s => s.kept != None)
    steps(send ~> sendStep, keep ~> keepStep, forget ~> forgetStep, poll ~> pollStep)
  }
