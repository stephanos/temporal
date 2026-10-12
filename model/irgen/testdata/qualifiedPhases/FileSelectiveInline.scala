// format: off
package fixture.qualifiedphases

import framework.*
val markerFileSelectiveInline = (); import Stage.idle
val usedFileSelectiveInline = idle

object FileSelectiveInline extends Machine[Job, Said, Nothing], Phased[Job, Stage](_.stage):
  val init = Job(Stage.idle)
  override def end(s: Job) = true
  object rules extends Rules:
    on(hand.press)(when(Stage.idle) ~> (s => enter[Job, Said, Nothing](s)))
