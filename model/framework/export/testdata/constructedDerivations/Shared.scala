package fixture.constructedderivations

import framework.dsl.*
import framework.dsl.Finite.given

object hand extends Actor:
  val go = action(this)
  val neighbor = action(this)
  val more = action(this)
  val extra = action(this)
  val one = action(this).input[Boolean]("flag")
  val two = action(this).input[Boolean]("a").input[Boolean]("b")
  val three = action(this).input[Boolean]("a").input[Boolean]("b").input[Boolean]("c")
  val four =
    action(this).input[Boolean]("a").input[Boolean]("b").input[Boolean]("c").input[Boolean]("d")
  val five = action(this)
    .input[Boolean]("a")
    .input[Boolean]("b")
    .input[Boolean]("c")
    .input[Boolean]("d")
    .input[Boolean]("e")
  val six = action(this)
    .input[Boolean]("a")
    .input[Boolean]("b")
    .input[Boolean]("c")
    .input[Boolean]("d")
    .input[Boolean]("e")
    .input[Boolean]("f")

object Base extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object effects:
    @lifted def prefixed(prefix: Boolean)(s: Boolean): List[Step[Boolean, Boolean, Nothing]] =
      List(Step(prefix, s))
    @lifted def first(s: Boolean): List[Step[Boolean, Boolean, Nothing]] = List(Step(true, s))
    @lifted def second(s: Boolean): List[Step[Boolean, Boolean, Nothing]] = List(Step(true, s))
    @lifted def replacement(s: Boolean): List[Step[Boolean, Boolean, Nothing]] = List(
      Step(false, s)
    )
    @lifted def one(s: Boolean, flag: Boolean): List[Step[Boolean, Boolean, Nothing]] = List(
      Step(flag, s)
    )
    @lifted def two(s: Boolean, a: Boolean, b: Boolean): List[Step[Boolean, Boolean, Nothing]] =
      List(Step(a || b, s))
    @lifted def three(
        s: Boolean,
        a: Boolean,
        b: Boolean,
        c: Boolean
    ): List[Step[Boolean, Boolean, Nothing]] = List(Step(a || b || c, s))
    @lifted def four(
        s: Boolean,
        a: Boolean,
        b: Boolean,
        c: Boolean,
        d: Boolean
    ): List[Step[Boolean, Boolean, Nothing]] = List(Step(a || b || c || d, s))
    @lifted def five(
        s: Boolean,
        a: Boolean,
        b: Boolean,
        c: Boolean,
        d: Boolean,
        e: Boolean
    ): List[Step[Boolean, Boolean, Nothing]] = List(Step(a || b || c || d || e, s))
    @lifted def six(
        s: Boolean,
        a: Boolean,
        b: Boolean,
        c: Boolean,
        d: Boolean,
        e: Boolean,
        f: Boolean
    ): List[Step[Boolean, Boolean, Nothing]] = List(Step(a || b || c || d || e || f, s))
    @lifted def curried(s: Boolean)(flag: Boolean): List[Step[Boolean, Boolean, Nothing]] = List(
      Step(flag, s)
    )
  object rules extends Rules:
    on(hand.go) {
      where((s: Boolean) => s) ~> effects.first
      where((s: Boolean) => !s) ~> effects.first
    }
    on(hand.neighbor)(always ~> effects.second)

object Classes extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Rules:
    on(hand.one(false))(always ~> Base.effects.first)
    on(hand.one(true))(always ~> Base.effects.second)

object Disabled extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Rules:
    disabled(hand.go)

object Core extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Bindings(hand.go ~> Base.effects.first)

object Forwarded extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Rules:
    on(hand.go) {
      where((s: Boolean) => s) ~> ((s: Boolean) => Base.effects.first(s))
      where((s: Boolean) => !s) ~> ((other: Boolean) => Base.effects.first(other))
    }

object Distinct extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Rules:
    on(hand.go) {
      where((s: Boolean) => s) ~> Base.effects.first
      where((s: Boolean) => !s) ~> Base.effects.second
    }

object MixedClasses extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Rules:
    on(hand.one)(where((s: Boolean) => s) ~> Base.effects.first)
    on(hand.one(false))(where((s: Boolean) => !s) ~> Base.effects.first)

object ClassAmbiguous extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Rules:
    on(hand.one(false)) {
      where((s: Boolean) => s) ~> Base.effects.first
      where((s: Boolean) => !s) ~> Base.effects.second
    }
