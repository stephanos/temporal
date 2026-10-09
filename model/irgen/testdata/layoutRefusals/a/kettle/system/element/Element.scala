// A folder below a level's: a zoom-in sits in system/ itself, one file per subject.
package fixture.features.kettle
package system
package element

import framework.*

object Element extends Machine[Kettle, Outcome, Nothing]:
  val init = Kettle(hot = false)
  def end(s: State) = true

  object effects:
    def boiled(s: State): List[KettleStep] = enter(s.copy(hot = true))

  object rules extends Rules:
    on(cook.boil)(where(!_.hot) ~> effects.boiled)
