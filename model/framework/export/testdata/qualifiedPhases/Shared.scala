package fixture.qualifiedphases

import framework.dsl.*

enum Stage:
  case idle extends Stage, Waiting
  case busy extends Stage, Held
  case done extends Stage, Succeeded
enum Phase:
  case low, high
final case class Job(stage: Stage)
enum Said:
  case ok
given Ok[Said] = Ok(Said.ok)
object hand extends Actor:
  val press = action(this)
object sets:
  def working(p: Stage): Boolean = p.in(Stage.idle, Stage.busy)
