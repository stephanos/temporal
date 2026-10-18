package fixture.finiteschemaclient

import framework.dsl.*
import fixture.finiteschemaroles.{Inner, Outcome, Phase, Snapshot}
import fixture.finiteschemahelper.Generic

object Inherited
    extends Generic[Snapshot, Outcome, Nothing],
      Phased[Snapshot, Phase](_.inner.phase):
  val init = Snapshot(Inner(Phase.waiting, false))
  object rules extends Bindings()
object DerivedInherited extends Derived(Inherited.unmonitored)
