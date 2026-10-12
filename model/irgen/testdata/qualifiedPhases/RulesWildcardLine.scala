package fixture.qualifiedphases

import framework.*

object RulesWildcardLine extends Machine[Job, Said, Nothing], Phased[Job, Stage](_.stage):
  val init = Job(Stage.idle)
  override def end(s: Job) = true
  object rules extends Rules:
    import Stage.*
    val used = idle
    on(hand.press)(when(Stage.idle) ~> (s => enter[Job, Said, Nothing](s)))
