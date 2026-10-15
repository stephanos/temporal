package fixture.rulespartial

import framework.dsl.*

object Library extends Machine[Boolean, Boolean, Nothing]:
  given Ok[Boolean] = Ok(true)
  val init = false
  def end(s: Boolean) = s
  object effects:
    @lifted def prefixed(prefix: Boolean)(s: Boolean) =
      stay[Boolean, Boolean, Nothing](prefix && s)
  object rules extends Rules
