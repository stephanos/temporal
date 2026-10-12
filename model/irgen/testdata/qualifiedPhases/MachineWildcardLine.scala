package fixture.qualifiedphases

import framework.*

object MachineWildcardLine extends Machine[Job, Said, Nothing], Phased[Job, Stage](_.stage):
  import Stage.*
  val used = idle
  val init = Job(Stage.idle)
  override def end(s: Job) = true
  object rules extends Rules:
    on(hand.press)(when(Stage.idle) ~> (s => enter[Job, Said, Nothing](s)))
