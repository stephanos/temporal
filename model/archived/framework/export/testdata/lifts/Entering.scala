// `enters(Phase.x)`, the framework step that enters a phase (fn-159.7), beside the effects it
// stands for: `Entering` enters its phases by `enters` in a rule and in a `choose` alternative,
// `Cored` by the effects written out. The lifter's tests lift both and require each rule's arm to be
// the steps of the effect it stands for, and the two choose functions one function. `resting`
// mixes in its role, so its type is narrower than the phase type, and still records its status.
package fixture.entering

import framework.dsl.*

enum Status:
  case waiting, working, resting

enum Stage(val status: Status) extends Recorded[Status]:
  case waiting extends Stage(Status.waiting)
  case working extends Stage(Status.working)
  case resting extends Stage(Status.resting), Suspended

final case class Task(stage: Stage, tries: UpTo[1])

enum Outcome:
  case accepted, refused

given Ok[Outcome] = Ok(Outcome.accepted)

object hand extends Actor:
  val press = action(this)
  val drop = action(this)

object Entering
    extends Machine[Task, Outcome, Status],
      Phased[Task, Stage](_.stage, (s, p) => s.copy(stage = p)):
  val init = Task(Stage.waiting, UpTo(0))
  override def end(s: State) = true

  object effects:
    val rested = choice
    val retried = choice
    def restOrRetry(s: Task) = choose(
      rested -> enters(Stage.resting)(s),
      retried -> enter(s.copy(tries = UpTo(1)))
    )

  object rules extends Rules:
    on(hand.press) {
      when(Stage.waiting) ~> enters(Stage.working)
      when(Stage.working) ~> effects.restOrRetry
    }
    on(hand.drop)(when(Stage.resting) ~> enters(Stage.waiting))

object Cored extends Machine[Task, Outcome, Status], Phased[Task, Stage](_.stage):
  val init = Task(Stage.waiting, UpTo(0))
  override def end(s: State) = true

  object effects:
    val rested = choice
    val retried = choice
    def restOrRetry(s: Task) = choose(
      rested -> enter(s.copy(stage = Stage.resting), Status.resting),
      retried -> enter(s.copy(tries = UpTo(1)))
    )
    def work(s: Task) = enter(s.copy(stage = Stage.working), Status.working)
    def settle(s: Task) = enter(s.copy(stage = Stage.waiting), Status.waiting)

  object rules extends Rules:
    on(hand.press) {
      when(Stage.waiting) ~> effects.work
      when(Stage.working) ~> effects.restOrRetry
    }
    on(hand.drop)(when(Stage.resting) ~> effects.settle)
