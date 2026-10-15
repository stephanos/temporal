package fixture.constructedevidence

import framework.dsl.*
import framework.dsl.Finite.given

object UnmarkedEvidence extends Machine[Boolean, Boolean, Fact]:
  val init = false
  def end(s: Boolean) = s
  val evidence: Fact => String = fact => Foreign.text(fact)
  object rules extends Bindings()

object OpaqueEvidence extends Machine[Boolean, Boolean, Fact]:
  val init = false
  def end(s: Boolean) = s
  val evidence: Fact => String = Opaque.callable
  object rules extends Bindings()
