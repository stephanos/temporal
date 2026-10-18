package framework.dsl

import framework.checker.Effects.*

object BoundedIntegersFixture:

  object counter extends Actor:
    val tick = action

  given Ok[Boolean] = Ok(false)

  object Bounded extends Machine[Boolean, Nothing]:
    var count: UpTo[2] = UpTo(0)
    override def end: Boolean = count == 2

    object transitions extends Transitions:
      on(counter.tick):
        if count < 2 then effects.increment()

    object effects:
      def increment(): Unit =
        count = UpTo(count + 1)

class BoundedIntegersSuite extends munit.FunSuite:
  final val one = 1
  final case class Counters(small: UpTo[one.type], large: UpTo[3], zero: UpTo[0])

  test("bounds belong to fields and their product keeps catalog order"):
    assertEquals(Finite[UpTo[0]].values.toList, List(0))
    assertEquals(
      Finite[Counters].values.map(c => (c.small: Int, c.large: Int, c.zero: Int)).toList,
      for small <- List(0, 1); large <- List(0, 1, 2, 3) yield (small, large, 0)
    )
    assertEquals(UpTo[3](3): Int, 3)
    for n <- List(-1, 4) do intercept[IllegalArgumentException](UpTo[3](n): Unit): Unit

  test("generic scalar Int catalogs remain legitimate"):
    given Finite[Int] = Finite.upTo(2)
    assertEquals(Finite[Int].values.toList, List(0, 1, 2))

  test("a machine's UpTo[N] var constructs and steps"):
    import BoundedIntegersFixture.*
    assert(summon[Data[UpTo[2]]].compare(UpTo[2](1), UpTo[2](2)) < 0)
    val binding = Bounded.bindings.find(_.decl eq counter.tick.decl).get
    val step = effectOf[State, Boolean, Nothing](binding.decl, binding.function)
    assertEquals(Bounded.init, Bounded.init.copy(count = UpTo[2](0)))
    assertEquals(step(Bounded.init, Nil).map(_.state), List(Bounded.init.copy(count = UpTo[2](1))))
    assertEquals(
      step(Bounded.init.copy(count = UpTo[2](1)), Nil).map(_.state),
      List(Bounded.init.copy(count = UpTo[2](2)))
    )
    assertEquals(step(Bounded.init.copy(count = UpTo[2](2)), Nil), Nil)
    assert(Bounded.end(Bounded.init.copy(count = UpTo[2](2))))
