package fixture.ruleslibrary

import framework.dsl.*

given Ok[Boolean] = Ok(true)
object hand extends Actor:
  val go = action(this)
object Library extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object states:
    val ready = is(true)
    val unused = is(throw IllegalArgumentException("unused"))
    def guarded(s: Boolean): Boolean =
      require(s)
      s
  object effects:
    val keep = effect {}
    def held(s: Boolean): List[Step[Boolean, Boolean, Nothing]] =
      require(s)
      stay[Boolean, Boolean, Nothing](s)
  object rules extends Rules:
    disabled(hand.go)
