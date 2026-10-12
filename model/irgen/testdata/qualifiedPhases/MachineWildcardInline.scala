// format: off
package fixture.qualifiedphases

import framework.*

object MachineWildcardInline extends Machine[Job, Said, Nothing], Phased[Job, Stage](_.stage):
  val marker = (); import Stage.*
  val used = idle
  val init = Job(Stage.idle)
  override def end(s: Job) = true
  object effects:
    def keep(s: Job) = enter[Job, Said, Nothing](s)
  object rules extends Rules:
    on(hand.press)(when(Stage.idle) ~> effects.keep)
