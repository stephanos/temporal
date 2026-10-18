package fixture.constructedheader

import framework.dsl.*

// Opaque state was refused by the old reader at lift; native state refuses it at compile time.
object Opaque extends Machine[Domains.Left, Domains.Right, Nothing]:
  val init = Domains.zero
  def end(s: State) = true
  object rules extends Bindings()
object OpaqueSeparate extends Machine[Independent.Left, Independent.Right, Nothing]:
  val init = Independent.zero
  def end(s: State) = true
  object rules extends Bindings()
