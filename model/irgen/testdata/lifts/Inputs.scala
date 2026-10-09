// Input tokens, inputs supplied by name and a bounded counter (fn-112.5). Each Scenario that
// supplies inputs by name, `token := value`, or omits every input, `start()`, has a twin with the
// positional calls it stands for, and the lifter's tests require one IR of the two. The counted
// state bounds its attempt counter by its field's type, and the tests require the Int range it lifts
// to. The control action reports results by name, with no enum of that name.
package fixture.inputs

import framework.*

enum Outcome derives Finite:
  case accepted

given Ok[Outcome] = Ok(Outcome.accepted)

// Where the counted run is.
enum Stage derives Finite:
  case unstarted, scheduled, backingOff, started, completed

// Whether a start sets a deadline.
enum Deadline derives Finite:
  case unset, expires

// The worker's answer: its first value, the default, is `completed`.
enum Answer derives Finite:
  case completed
  case failed(retryable: Boolean)
  case abandoned

enum Control derives Finite:
  case pause, unpause

// Three inputs of one type, told apart by their tokens.
val closeBy = input[Deadline]
val startBy = input[Deadline]
val runBy = input[Deadline]
val answer = input[Answer]
val urgent = input[Boolean]
val control = input[Control]

val start = action(Actor("caller"))
  .input(closeBy)
  .input(startBy)
  .input(runBy)
val respond = action(Actor("worker")).input(answer).input(urgent)
// Reports results by name: the IR keeps the text, and no `Delivery` enum is declared.
val steer = action(Actor("caller")).input(control).results("Delivery")

// Five stages, a counter of 0 to 2 bounded by its type and three deadlines: 120 states. Named apart
// from the machine object `Counted`.
final case class CountedState(
    phase: Stage,
    attempts: UpTo[2],
    closeBy: Deadline,
    startBy: Deadline,
    runBy: Deadline
) derives Finite

type CountedStep = Step[CountedState, Outcome, Nothing]

def started(
    s: CountedState,
    close: Deadline,
    toStart: Deadline,
    toClose: Deadline
): List[CountedStep] =
  if s.phase != Stage.unstarted then disabled
  else enter(CountedState(Stage.scheduled, UpTo(0), close, toStart, toClose))

// A retried answer counts one more attempt, saturating at the bound.
def responded(s: CountedState, a: Answer, u: Boolean): List[CountedStep] = a match
  case Answer.failed(true) if s.phase == Stage.started || u =>
    enter(s.copy(phase = Stage.backingOff, attempts = UpTo((s.attempts + 1).min(2))))
  case _ => disabled

def steered(s: CountedState, c: Control): List[CountedStep] =
  if c == Control.pause && s.attempts < 2 then stay(s) else disabled

object Counted extends Machine[CountedState, Outcome, Nothing]:
  val init = CountedState(Stage.unstarted, UpTo(0), Deadline.unset, Deadline.unset, Deadline.unset)
  def end(counted: State) = true

  object rules extends Bindings(start ~> started, respond ~> responded, steer ~> steered)

val scheduled = Counted.property holds (after => after.state.phase != Stage.completed)

// Partial and reordered calls by name, a default of an enum with fields and of a Boolean, and a
// token of a one-input action.
val byName = Counted.scenario.actions(
  start(startBy := Deadline.expires),
  start(runBy := Deadline.expires, closeBy := Deadline.expires),
  start(
    closeBy := Deadline.unset,
    startBy := Deadline.unset,
    runBy := Deadline.expires
  ),
  respond(urgent := true),
  respond(answer := Answer.failed(true)),
  steer(control := Control.unpause)
)
val byPosition = Counted.scenario.actions(
  start(Deadline.unset, Deadline.expires, Deadline.unset),
  start(Deadline.expires, Deadline.unset, Deadline.expires),
  start(Deadline.unset, Deadline.unset, Deadline.expires),
  respond(Answer.completed, true),
  respond(Answer.failed(true), false),
  steer(Control.unpause)
)

// Every input omitted, `start()`: each at its domain's first value, beside its positional twin.
val omitted = Counted.scenario.actions(start(), respond(), steer())
val omittedByPosition = Counted.scenario.actions(
  start(Deadline.unset, Deadline.unset, Deadline.unset),
  respond(Answer.completed, false),
  steer(Control.pause)
)

// A Property restricted to a class named by name, beside its positional twin.
val urgentByName = Counted.property.when(respond(urgent := true)) holds
  (after => after.state.attempts <= 2)
val urgentByPosition = Counted.property.when(respond(Answer.completed, true)) holds
  (after => after.state.attempts <= 2)

val six = Limits(steps = 6, actions = 6, search = 64)

val byNameQuery = query verify scheduled in byName limits six total 720
val byPositionQuery = query verify scheduled in byPosition limits six total 720
val urgentByNameQuery = query verify urgentByName in byName limits six total 720
val urgentByPositionQuery = query verify urgentByPosition in byPosition limits six total 720
val omittedQuery = query verify scheduled in omitted limits six total 360
val omittedByPositionQuery = query verify scheduled in omittedByPosition limits six total 360
