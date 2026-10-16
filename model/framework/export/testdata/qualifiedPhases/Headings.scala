package fixture.qualifiedphases

import framework.dsl.*
import Phase.low as quiet
import Phase.*
import sets.working

val unrelated = quiet
val other = high

object Accepted extends Machine[Job, Said, Nothing], Phased[Job, Stage](_.stage):
  val init = Job(Stage.idle)
  override def end(s: Job) = s.stage == Stage.done
  object effects:
    def keep(s: Job) = enter[Job, Said, Nothing](s.copy(stage = Stage.busy))
  object rules extends Rules:
    on(hand.press) {
      when(Stage.idle, Stage.busy) ~> effects.keep
      when[Closed] ~> effects.keep
    }
  object properties:
    val pressed = property when hand.press expect(check(after.state.stage == Stage.busy))

object QualifiedSet extends Machine[Job, Said, Nothing], Phased[Job, Stage](_.stage):
  val init = Job(Stage.idle)
  override def end(s: Job) = s.stage == Stage.done
  object rules extends Rules:
    on(hand.press)(when(sets.working) ~> Accepted.effects.keep)

object BarePhase extends Machine[Job, Said, Nothing], Phased[Job, Stage](_.stage):
  val init = Job(Stage.idle)
  override def end(s: Job) = true
  val idle: Stage = Stage.idle
  object rules extends Rules:
    on(hand.press)(when(idle) ~> Accepted.effects.keep)

object BareSet extends Machine[Job, Said, Nothing], Phased[Job, Stage](_.stage):
  val init = Job(Stage.idle)
  override def end(s: Job) = true
  object rules extends Rules:
    on(hand.press)(when(working) ~> Accepted.effects.keep)

object Inherited extends Derived(Accepted.restrict(hand.press)):
  object states:
    def terminal(s: Job) = s.stage == Stage.done
