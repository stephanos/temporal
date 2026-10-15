package fixture.rulespartial

import framework.dsl.*

object Library extends Machine[Boolean, Boolean, Nothing]:
  given Ok[Boolean] = Ok(true)
  val init = false
  def end(s: Boolean) = s
  object effects:
    @lifted def prefixed(prefix: Boolean)(s: Boolean) =
      stay[Boolean, Boolean, Nothing](prefix && s)
    @lifted def leading()(prefix: Boolean)(s: Boolean) =
      stay[Boolean, Boolean, Nothing](prefix && s)
    @lifted def middle(prefix: Boolean)()(s: Boolean) =
      stay[Boolean, Boolean, Nothing](prefix && s)
    @lifted def repeated()()(prefix: Boolean)()(s: Boolean) =
      stay[Boolean, Boolean, Nothing](prefix && s)
    @lifted def trailing(s: Boolean)() = stay[Boolean, Boolean, Nothing](s)
    @lifted def polymorphic[@scala.annotation.unused A](prefix: Boolean)(s: Boolean) =
      stay[Boolean, Boolean, Nothing](prefix && s)
    @lifted def multiple(a: Boolean)(b: Boolean)(s: Boolean) =
      stay[Boolean, Boolean, Nothing](a && b && s)
    @lifted def six(
        prefix: Boolean
    )(s: Boolean, a: Boolean, b: Boolean, c: Boolean, d: Boolean, e: Boolean, f: Boolean) =
      stay[Boolean, Boolean, Nothing](prefix && s && a && b && c && d && e && f)
  object rules extends Rules
