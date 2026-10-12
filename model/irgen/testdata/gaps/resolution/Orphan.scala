// fn-167.2: a gap in a package no IR file is collected from (see Gapfile.scala).
package fixture.gaporphan

import framework.*

enum Spare:
  case only

given Ok[Spare] = Ok(Spare.only)

object Idle extends Machine[Spare, Spare, Nothing]:
  val init = Spare.only
  def end(s: State) = true
  object rules extends Bindings()

object extras:
  @gap(unasked)(because = "nothing lifts it")
  val orphan = Idle.property.holds(_ => true)
