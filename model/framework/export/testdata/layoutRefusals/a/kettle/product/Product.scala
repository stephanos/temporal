package fixture.features.kettle
package product

import framework.*

object KettleProduct extends Machine[Kettle, Outcome, Nothing]:
  val init = Kettle(hot = false)
  def end(s: State) = true

  object effects:
    def boiled(s: State): List[KettleStep] = enter(s.copy(hot = true))

  object rules extends Rules:
    on(cook.boil)(where(!_.hot) ~> effects.boiled)
