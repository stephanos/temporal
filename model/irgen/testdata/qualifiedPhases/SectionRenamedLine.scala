package fixture.qualifiedphases

import framework.*

object SectionRenamedLine extends Machine[Job, Said, Nothing], Phased[Job, Stage](_.stage):
  val init = Job(Stage.idle)
  override def end(s: Job) = true
  object states:
    import Stage.idle as waiting
    val used = waiting
  object rules extends Rules:
    on(hand.press)(when(Stage.idle) ~> (s => enter[Job, Said, Nothing](s)))
