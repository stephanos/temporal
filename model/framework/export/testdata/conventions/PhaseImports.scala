// fn-159.3: imports of a phase type's members, which the lifter refuses at their lines wherever they
// sit, wildcard or selective; the type itself, imported by name, is no member import.
package fixture.conventions.phases

import framework.*
import Stage.done

enum Stage:
  case idle, busy, done

final case class Job(stage: Stage)

enum Said:
  case ok

given Ok[Said] = Ok(Said.ok)

val last = done

object hand extends Actor:
  val press = action(this)

object Staged extends Machine[Job, Said, Nothing], Phased[Job, Stage](_.stage):
  import Stage.*
  val init = Job(idle)
  override def end(s: Job) = s.stage.in(Stage.done)

  object rules extends Rules:
    import Stage.busy
    on(hand.press)(when(Stage.idle) ~> (s => enter[Job, Said, Nothing](s.copy(stage = busy))))
