// Input tokens, inputs supplied by name and a bounded counter (fn-112.5). Each Scenario that
// supplies inputs by name, `token := value`, has a twin with the positional calls it stands for,
// and the lifter's tests require one IR of the two. The counted state is the standalone activity's
// protocol state with its counter bounded by its field's type, and the tests require the record the
// protocol state lifts to. The control action reports results by name, with no enum of that name.
package fixture.inputs

import temporal.standaloneactivity.{Phase, Timeout}
import umpire.*

given Family = Family("fixture.inputs")

enum Outcome derives Finite:
  case accepted

given Accepted[Outcome] = Accepted(Outcome.accepted)

/** The worker's answer: its first value, the default, is `completed`. */
enum Answer derives Finite:
  case completed
  case failed(retryable: Boolean)
  case canceled

enum Control derives Finite:
  case pause, unpause

// Three inputs of one type, told apart by their tokens.
val scheduleToClose = input[Timeout]
val scheduleToStart = input[Timeout]
val startToClose = input[Timeout]
val answer = input[Answer]
val urgent = input[Boolean]
val control = input[Control]

val start = action(Party("caller"))
  .input(scheduleToClose)
  .input(scheduleToStart)
  .input(startToClose)
val respond = action(Party("worker")).input(answer).input(urgent)
// Reports results by name: the IR keeps the text, and no `Delivery` enum is declared.
val steer = action(Party("caller")).input(control).results("Delivery")

/** The protocol state's fields, its attempt counter bounded by its type: 0, 1 and 2. */
final case class Counted(
    phase: Phase,
    attempts: UpTo[2],
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout
) derives Finite

type CountedStep = Step[Counted, Outcome, Nothing]

def started(s: Counted, close: Timeout, toStart: Timeout, toClose: Timeout): List[CountedStep] =
  if s.phase != Phase.unstarted then disabled
  else accept(Counted(Phase.scheduled, UpTo(0), close, toStart, toClose))

// A retried answer counts one more attempt, saturating at the bound.
def responded(s: Counted, a: Answer, u: Boolean): List[CountedStep] = a match
  case Answer.failed(true) if s.phase == Phase.started || u =>
    accept(s.copy(phase = Phase.backingOff, attempts = UpTo((s.attempts + 1).min(2))))
  case _ => disabled

def steered(s: Counted, c: Control): List[CountedStep] =
  if c == Control.pause && s.attempts < 2 then stay(s) else disabled

val counted = machine[Counted, Outcome, Nothing] {
  starts(Counted(Phase.unstarted, UpTo(0), Timeout.unset, Timeout.unset, Timeout.unset))
  ends(_ => true)
  steps(start ~> started, respond ~> responded, steer ~> steered)
}

val scheduled = counted.property holds (after => after.state.phase != Phase.completed)

// Partial and reordered calls by name, a default of an enum with fields and of a Boolean, and a
// token of a one-input action.
val byName = counted.scenario.actions(
  start(scheduleToStart := Timeout.expires),
  start(startToClose := Timeout.expires, scheduleToClose := Timeout.expires),
  start(
    scheduleToClose := Timeout.unset,
    scheduleToStart := Timeout.unset,
    startToClose := Timeout.expires
  ),
  respond(urgent := true),
  respond(answer := Answer.failed(true)),
  steer(control := Control.unpause)
)
val byPosition = counted.scenario.actions(
  start(Timeout.unset, Timeout.expires, Timeout.unset),
  start(Timeout.expires, Timeout.unset, Timeout.expires),
  start(Timeout.unset, Timeout.unset, Timeout.expires),
  respond(Answer.completed, true),
  respond(Answer.failed(true), false),
  steer(Control.unpause)
)

// A Property restricted to a class named by name, beside its positional twin.
val urgentByName = counted.property.when(respond(urgent := true)) holds
  (after => after.state.attempts <= 2)
val urgentByPosition = counted.property.when(respond(Answer.completed, true)) holds
  (after => after.state.attempts <= 2)

val six = Limits(steps = 6, actions = 6, search = 64)

val byNameQuery = query verify scheduled in byName limits six
val byPositionQuery = query verify scheduled in byPosition limits six
val urgentByNameQuery = query verify urgentByName in byName limits six
val urgentByPositionQuery = query verify urgentByPosition in byPosition limits six
