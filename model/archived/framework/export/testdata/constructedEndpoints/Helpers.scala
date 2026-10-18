package fixture.endpointlibrary

import framework.dsl.Lifted

object Helpers extends Lifted:
  def leaf(s: Boolean): Boolean = !s
  def forward(s: Boolean): Boolean = leaf(s)

object Unmarked:
  def unmarked(s: Boolean): Boolean = s
