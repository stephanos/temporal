package fixture.rulespartial

import framework.dsl.*

object Library extends Machine[Boolean, Boolean, Nothing]:
  given Ok[Boolean] = Ok(true)
  val init = false
  def end(s: Boolean) = s
  object effects:
    def prefixed(prefix: Boolean)(s: Boolean) =
      stay[Boolean, Boolean, Nothing](prefix && s)
    def leading()(prefix: Boolean)(s: Boolean) =
      stay[Boolean, Boolean, Nothing](prefix && s)
    def middle(prefix: Boolean)()(s: Boolean) =
      stay[Boolean, Boolean, Nothing](prefix && s)
    def repeated()()(prefix: Boolean)()(s: Boolean) =
      stay[Boolean, Boolean, Nothing](prefix && s)
    def trailing(s: Boolean)() = stay[Boolean, Boolean, Nothing](s)
    def polymorphic[@scala.annotation.unused A](prefix: Boolean)(s: Boolean) =
      stay[Boolean, Boolean, Nothing](prefix && s)
    def multiple(a: Boolean)(b: Boolean)(s: Boolean) =
      stay[Boolean, Boolean, Nothing](a && b && s)
    def six(
        prefix: Boolean
    )(s: Boolean, a: Boolean, b: Boolean, c: Boolean, d: Boolean, e: Boolean, f: Boolean) =
      stay[Boolean, Boolean, Nothing](prefix && s && a && b && c && d && e && f)
  object rules extends Rules
