package framework.dsl

object BoundedIntegersFixture:

  object counter extends Actor:
    val tick = action

  // A whole state that is a bounded counter, authored without explicit evidence.
  object Bounded extends RecordMachine[UpTo[2], Boolean, Nothing]:
    val init = UpTo[2](0)
    def end(s: UpTo[2]): Boolean = s == UpTo[2](2)

    object rules extends Rules:
      on(counter.tick):
        where(s => s < 2) ~> ((s: UpTo[2]) => List(Step(false, UpTo[2](s + 1))))

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

  test("a machine whose whole state is UpTo[N] constructs and steps"):
    import BoundedIntegersFixture.*
    assert(summon[Data[UpTo[2]]].compare(UpTo[2](1), UpTo[2](2)) < 0)
    val binding = Bounded.bindings.find(_.decl eq counter.tick.decl).get
    val step = effectOf[UpTo[2], Boolean, Nothing](binding.decl, binding.function)
    assertEquals(Bounded.init: Int, 0)
    assertEquals(step(UpTo[2](0), Nil).map(_.state: Int), List(1))
    assertEquals(step(UpTo[2](1), Nil).map(_.state: Int), List(2))
    assertEquals(step(UpTo[2](2), Nil), Nil)
    assert(Bounded.end(UpTo[2](2)))
