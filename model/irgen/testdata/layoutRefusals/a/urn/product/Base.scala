// The Product, in a file not named after its folder: product/Product.scala is missing.
package fixture.features.urn
package product

import umpire.*

object brewer extends Actor:
  val fill = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)

object UrnProduct extends Machine[Urn, Outcome, Nothing]:
  val init = Urn(full = false)
  def end(s: State) = true

  object effects extends Section:
    def filled(s: State): List[UrnStep] = enter(s.copy(full = true))

  object rules extends Rules:
    when(!_.full)(brewer.fill ~> effects.filled)
