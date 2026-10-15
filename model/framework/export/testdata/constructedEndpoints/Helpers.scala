package fixture.endpointlibrary

import framework.dsl.lifted

object Helpers:
  @lifted def leaf(s: Boolean): Boolean = !s
  @lifted def forward(s: Boolean): Boolean = leaf(s)
  def unmarked(s: Boolean): Boolean = s
