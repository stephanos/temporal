package fixture.qualifiedphases

import framework.*
import Stage.*
val usedFileWildcardLine = idle

object FileWildcardLine extends Machine[Job, Said, Nothing], Phased[Job, Stage](_.stage):
  val init = Job(Stage.idle)
  override def end(s: Job) = true
  object rules extends Rules:
    on(hand.press)(when(Stage.idle) ~> (s => enter[Job, Said, Nothing](s)))
