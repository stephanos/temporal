package fixture.constructedderivations

import framework.dsl.*
import framework.dsl.Finite.given

object hand extends Actor:
  val go = action(this)
  val neighbor = action(this)
  val more = action(this)
  val extra = action(this)
  val one = action(this).input[Boolean]("flag")
  val two = action(this).inputs[Two]
  val three = action(this).inputs[Three]
  val four =
    action(this).inputs[Four]
  val five = action(this)
    .inputs[Five]
  // Native actions declare one record for several inputs (fn-165), each keeping the retired
  // input chain's names, order and types, one IR input a field; the helpers keep their flat
  // parameters and the rules read the record's fields into them. The records take the
  // chains' lines, so every frozen position keeps its line.
  val six = action(this)
    .inputs[Six]
final case class Two(a: Boolean, b: Boolean)
final case class Three(a: Boolean, b: Boolean, c: Boolean)
final case class Four(a: Boolean, b: Boolean, c: Boolean, d: Boolean)
final case class Five(a: Boolean, b: Boolean, c: Boolean, d: Boolean, e: Boolean)
final case class Six(a: Boolean, b: Boolean, c: Boolean, d: Boolean, e: Boolean, f: Boolean)

object Base extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object effects:
    def prefixed(prefix: Boolean)(s: Boolean): List[Step[Boolean, Boolean, Nothing]] =
      List(Step(prefix, s))
    def first(s: Boolean): List[Step[Boolean, Boolean, Nothing]] = List(Step(true, s))
    def second(s: Boolean): List[Step[Boolean, Boolean, Nothing]] = List(Step(true, s))
    def replacement(s: Boolean): List[Step[Boolean, Boolean, Nothing]] = List(
      Step(false, s)
    )
    def one(s: Boolean, flag: Boolean): List[Step[Boolean, Boolean, Nothing]] = List(
      Step(flag, s)
    )
    def two(s: Boolean, a: Boolean, b: Boolean): List[Step[Boolean, Boolean, Nothing]] =
      List(Step(a || b, s))
    def three(
        s: Boolean,
        a: Boolean,
        b: Boolean,
        c: Boolean
    ): List[Step[Boolean, Boolean, Nothing]] = List(Step(a || b || c, s))
    def four(
        s: Boolean,
        a: Boolean,
        b: Boolean,
        c: Boolean,
        d: Boolean
    ): List[Step[Boolean, Boolean, Nothing]] = List(Step(a || b || c || d, s))
    def five(
        s: Boolean,
        a: Boolean,
        b: Boolean,
        c: Boolean,
        d: Boolean,
        e: Boolean
    ): List[Step[Boolean, Boolean, Nothing]] = List(Step(a || b || c || d || e, s))
    def six(
        s: Boolean,
        a: Boolean,
        b: Boolean,
        c: Boolean,
        d: Boolean,
        e: Boolean,
        f: Boolean
    ): List[Step[Boolean, Boolean, Nothing]] = List(Step(a || b || c || d || e || f, s))
    def curried(s: Boolean)(flag: Boolean): List[Step[Boolean, Boolean, Nothing]] = List(
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
