package propertygroups

import framework.dsl.*

enum State:
  case open, closed
enum Outcome:
  case done

object tester extends Actor:
  val advance = action("advance", this)
val fair = assume("fair").fair(tester.advance)
val seal = monitor[State, Outcome, Nothing, Boolean]("seal", false)((_, _, after) =>
  after.state == State.closed
)(s => s)

def step(s: State): List[Step[State, Outcome, Nothing]] =
  List(Step(Outcome.done, State.closed))

def shared(m: Declares[State]): Property[State] =
  m.property("aliased").holds(after => after.state != State.open)

final case class Laws(open: Property[State], closed: Property[State])
def laws(m: Declares[State]): Laws = Laws(
  m.property("bundledOpen").holds(after => after.state == State.open),
  m.property("bundledClosed")
    .holds(after => after.state == State.closed && after.outcome == Outcome.done)
)

def eventual(m: Machine[State, Outcome, Nothing]): Progress[State] =
  m.leadsTo("factoryProgress")(_ == State.open, _ == State.closed, 2, fair)

final case class ProgressLaws(progress: Progress[State])
def progressLaws(m: Machine[State, Outcome, Nothing]): ProgressLaws =
  ProgressLaws(m.leadsTo("bundledOnly")(_ == State.open, _ == State.closed, 4, fair))

val limits = Limits("small", steps = 1, actions = 1, search = 8)
