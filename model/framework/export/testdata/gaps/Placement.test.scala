package fixture.gaps

import framework.dsl.*

object TestOnly:
  @gap(unasked)(because = "tests are not source declarations")
  val property = Checked.property.holds(_ => true)
