// Union and presence, lifted as the finite types they are: an enum whose cases carry their own
// fields, an optional value, and counters with their own ranges.
// The tests lift `presence` and compare the IR with expected/presence.json.
package fixture.presence

import framework.dsl.*; import framework.dsl.Finite.given

enum Result:
  case succeeded, failed

// A union: the second case carries fields the first does not.
enum Report:
  case unsent
  case sent(result: Result, retried: Boolean)

object bounded:
  // A retry count, 0 to 2.
  opaque type Retries = Int

  object Retries:
    given Finite[Retries] = Finite.upTo(2)
    val none: Retries = 0
    val most: Retries = 2
    extension (r: Retries) def next: Retries = (r + 1).min(2)

import bounded.Retries

// Each counter names its own range.
final case class State(report: Report, kept: Option[Result], retries: Retries, polls: UpTo[1])

enum Outcome:
  case accepted, ignored

val send = action(Actor("fixture")).input[Result]("result")
val keep = action(Actor("fixture"))
val forget = action(Actor("fixture"))
val poll = timer

def sendStep(s: State, r: Result): List[Step[State, Outcome, Nothing]] = s.report match
  case Report.unsent        => List(Step(Outcome.accepted, s.copy(report = Report.sent(r, false))))
  case Report.sent(_, true) => Nil
  case Report.sent(first, false) => // format: off
    if s.retries == Retries.most then Nil
    else // format: on
      List(
        Step(Outcome.accepted, s.copy(report = Report.sent(first, true), retries = s.retries.next))
      )

// Keeps what was sent. The match binds `r` inside a local `val`, and `None` takes its type from the
// match.
def keepStep(s: State): List[Step[State, Outcome, Nothing]] =
  val sent = s.report match
    case Report.sent(r, _) => Some(r)
    case Report.unsent     => None // format: off
  if sent == None || s.kept == sent then Nil else List(Step(Outcome.accepted, s.copy(kept = sent)))
  // format: on
def forgetStep(s: State): List[Step[State, Outcome, Nothing]] = s.kept match
  case Some(_) => List(Step(Outcome.accepted, s.copy(kept = None)))
  case None    => List(Step(Outcome.ignored, s))

def pollStep(s: State): List[Step[State, Outcome, Nothing]] = // format: off
  if s.polls == 0 then List(Step(Outcome.accepted, s.copy(polls = UpTo(1)))) else Nil
  // format: on
object Presence extends Machine[State, Outcome, Nothing]:
  val init = State(Report.unsent, None, Retries.none, UpTo(0)) // format: off
  def end(s: State) = s.kept != None
  // format: on
  object rules
      extends Bindings(send ~> sendStep, keep ~> keepStep, forget ~> forgetStep, poll ~> pollStep)
