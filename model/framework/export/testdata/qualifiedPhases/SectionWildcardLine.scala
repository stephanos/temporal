package fixture.qualifiedphases

import framework.dsl.*

object SectionWildcardLine extends Machine[Job, Said, Nothing], Phased[Job, Stage](_.stage):
  val init = Job(Stage.idle)
  override def end(s: Job) = true
  object states:
    import Stage.*
    val used = idle
  object effects:
    def keep(s: Job) = enter[Job, Said, Nothing](s)
  object rules extends Rules:
    on(hand.press)(when(Stage.idle) ~> effects.keep)
