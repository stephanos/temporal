// format: off
package fixture.qualifiedphases

import framework.*

object RulesRenamedInline extends Machine[Job, Said, Nothing], Phased[Job, Stage](_.stage):
  val init = Job(Stage.idle)
  override def end(s: Job) = true
  object effects:
    def keep(s: Job) = enter[Job, Said, Nothing](s)
  object rules extends Rules:
    val marker = (); import Stage.{idle as waiting}
    val used = waiting
    on(hand.press)(when(Stage.idle) ~> effects.keep)
