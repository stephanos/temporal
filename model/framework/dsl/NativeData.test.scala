package framework.dsl

import framework.dsl.Finite.given
import scala.compiletime.testing.typeCheckErrors

object NativeDataFixture:
  enum Flag:
    case first, second
  final case class Pair(flag: Flag, count: Int)
  final case class Collections(seen: Set[Int], pending: List[Int], counts: Map[Int, Int])
  final case class Poison(flag: Flag, ready: Boolean)

  object caller extends Actor:
    val act = action(this)
    val extra = action(this)

class NativeDataTest extends munit.FunSuite:
  import NativeDataFixture.*

  private def ordered[A](values: List[A])(using data: Data[A]): Unit =
    for (left, i) <- values.zipWithIndex; (right, j) <- values.zipWithIndex do
      assertEquals(Integer.signum(data.compare(left, right)), Integer.signum(i.compare(j)))
      assertEquals(data.equivalent(left, right), i == j)

  test("structural comparison follows type order independently of finite catalogs") {
    ordered(List(false, true))
    ordered(List(Int.MinValue, -1, 0, 2, Int.MaxValue))
    ordered(List("", "a", "b", "\uffff", "\ud800\udc00"))
    assert(!Data[String].equivalent(0xd800.toChar.toString, 0xd801.toChar.toString))
    ordered(List(Flag.first, Flag.second))
    ordered(List(Pair(Flag.first, 0), Pair(Flag.first, 2), Pair(Flag.second, -1)))
    ordered(List[Option[Int]](None, Some(-1), Some(2)))
    ordered(List(List.empty[Int], List(0), List(0, 2), List(2)))
    ordered(List(Set.empty[Int], Set(0), Set(0, 2), Set(2)))
    ordered(List(Map.empty[Int, Int], Map(0 -> 1), Map(0 -> 2), Map(2 -> 0)))
    assert(Data[Set[Int]].equivalent(Set(0, 2), Set(2, 0)))
    assert(Data[Map[Int, Int]].equivalent(Map(0 -> 1, 2 -> 3), Map(2 -> 3, 0 -> 1)))

    given Finite[Poison] = new Finite[Poison]:
      def values = throw AssertionError("structural comparison enumerated states")
    ordered(List(Poison(Flag.first, false), Poison(Flag.first, true), Poison(Flag.second, false)))
  }

  test("Inbox state constructs and sends without enumerating its message catalog") {
    given Finite[Flag] = new Finite[Flag]:
      def values = throw AssertionError("Inbox construction enumerated messages")
    val wire = channel[Flag](2, Order.unordered, Loss.reliable)
    final case class Held(inbox: Inbox[Flag], received: List[Flag])
    object Native extends Machine[Held, Boolean, Nothing]:
      val init = Held(wire.empty, Nil)
      def end(s: Held) = s.inbox.isEmpty
      object rules extends Rules:
        disabled(wire.deliver)
    assertEquals(Native.bindings.size, 1)
    val sent = Native.init.copy(inbox = wire.empty.send(Flag.second).send(Flag.first))
    assertEquals(sent.inbox.deliveries.map(_.message), List(Flag.first, Flag.second))
    assert(Data[Held].equivalent(sent, sent.copy()))
  }

  test("custom finite action order and unordered Inbox type order are independent") {
    given Finite[Int] = Finite.of(2, 0)
    val pick = action(Actor("inputs")).input[Int](name = "number")
    assertEquals(classesOf(pick.decl), List(List(2), List(0)))
    assert(Data[Int].compare(0, 2) < 0)
    val data: Data[Int] = Finite[Int]
    assert(data.compare(0, 2) < 0)
    object NumericOutcome extends Machine[Collections, Int, Nothing]:
      val init = Collections(Set.empty, Nil, Map.empty)
      def end(s: Collections) = true
      object rules extends Rules:
        disabled(caller.act)
    assertEquals(NumericOutcome.fo.values.toList, List(2, 0))
    assertEquals(NumericOutcome.ff.values.toList, Nil)

    val unordered = channel[Int](2, Order.unordered, Loss.reliable, duplicates = 1)
    val expected = List(Delivery(0, 0), Delivery(2, 0))
    for inserted <- List(List(2, 0), List(0, 2)) do
      val inbox = inserted.foldLeft(unordered.empty)((held, message) => held.send(message))
      assertEquals(inbox.deliveries, expected)
      assert(inbox.isFull)
      assert(unordered.contents.values.exists(_.deliveries == expected))
    assertEquals(unordered.empty.send(2).send(2).deliveries, List.fill(2)(Delivery(2, 0)))
    assert(unordered.contents.values.exists(_.deliveries == List(Delivery(0, 1))))
    val overflow = unordered.empty.send(0).send(2).send(0)
    assert(overflow.isFull)
    assertEquals(overflow.deliveries.size, 3)
    assert(!unordered.contents.values.contains(overflow))
    assertEquals(
      unordered.contents.values.filter(_.deliveries.size == 1).map(_.deliveries.head).toList,
      List(Delivery(0, 0), Delivery(0, 1), Delivery(2, 0), Delivery(2, 1))
    )

    val fifo = channel[Int](2, Order.fifo, Loss.reliable)
    assertEquals(fifo.empty.send(2).send(0).deliveries, List(Delivery(2, 0), Delivery(0, 0)))
    assert(!Data[Inbox[Int]].equivalent(fifo.empty, unordered.empty))
    intercept[IllegalArgumentException](channel[Int](0, Order.fifo, Loss.reliable)): Unit
    intercept[IllegalArgumentException](channel[Int](1, Order.fifo, Loss.reliable, -1)): Unit
  }

  test("collection machine state uses native partial Map operations") {
    object Native extends Machine[Collections, Boolean, Nothing]:
      val init = Collections(Set.empty, Nil, Map.empty)
      def end(s: Collections) = s.pending.isEmpty
      object rules extends Rules:
        on(caller.act):
          always ~> ((s: Collections) =>
            List(
              Step(
                true,
                s.copy(
                  seen = s.seen.incl(2),
                  pending = s.pending :+ 2,
                  counts = s.counts.updated(2, s.counts.getOrElse(2, 0) + 1)
                )
              )
            )
          )
    val initial = Native.init
    assertEquals(initial.counts.get(2), None)
    assertEquals(initial.counts.getOrElse(2, 7), 7)
    intercept[NoSuchElementException](initial.counts(2)): Unit
    val run =
      effectOf[Collections, Boolean, Nothing](caller.act.decl, Native.bindings.head.function)
    val next = run(initial, Nil).head.state
    assertEquals(next, Collections(Set(2), List(2), Map(2 -> 1)))
    assertEquals(stuck(initial).states, Seq(initial))
  }

  test("declared and rebound rules construct without enumerating state or invoking callbacks") {
    given Finite[Poison] = new Finite[Poison]:
      def values = throw AssertionError("construction enumerated whole-state catalog")
    def guard(s: Poison): Boolean = throw AssertionError("construction invoked guard")
    def effect(s: Poison): List[Step[Poison, Boolean, Nothing]] =
      throw AssertionError("construction invoked effect")
    object Native extends Machine[Poison, Boolean, Nothing]:
      val init = Poison(Flag.first, false)
      def end(s: Poison) = false
      object rules extends Rules:
        on(caller.act):
          where(guard) ~> effect
          where(guard) ~> effect
    assertEquals(Native.bindings.size, 1)
    assert(Native.fs eq Finite[Poison])
    val derived = Native.rebind(
      on(caller.act):
        where(guard) ~> effect
        where(guard) ~> effect
    )
    assertEquals(derived.bindings.size, 1)
    val retained = derived.table.collect { case (_, Bound.Ruled(rules)) => rules }.flatten
    assertEquals(retained.map(_.index), Vector(1, 2))
    assert(retained.forall(_.decl eq caller.act.decl))
    val restricted = Native.restrict(caller.act)
    val extended = Native.extend(
      on(caller.extra):
        where(guard) ~> effect
    )
    val refined =
      Native.refining(Native)(_ => throw AssertionError("construction invoked refinement"))
    object Copied extends Derived(derived)
    for machine <- List(derived, restricted, extended, refined, Copied) do
      assert(machine.fs eq Native.fs)
      assert(machine.fo eq Native.fo)
      assert(machine.ff eq Native.ff)
      assertEquals(machine.init, Native.init)
    assertEquals(extended.bindings.map(_.decl), List(caller.act.decl, caller.extra.decl))
    assert(refined.refinementEvidence.get.of eq Native)
    final case class ProductState(left: Poison, right: Poison)
    object Product extends Composition[ProductState](_.left -> Native, _.right -> Copied)
    assertEquals(Product.members, Vector(Native, Copied))
  }

  test("collection monitor copies preserve their state evidence without running callbacks") {
    val initial = Collections(Set.empty, Nil, Map.empty)
    val watch = monitor[Collections, Boolean, Nothing, Collections](initial)((_, _, _) =>
      throw AssertionError("construction invoked monitor callback")
    )(_ => throw AssertionError("construction invoked monitor verdict"))
    val copies = List(watch.readAtEnds, watch.readAfter(_ => true))
    for copy <- copies do
      assertEquals(copy.initial, initial)
      assert(copy.states eq watch.states)
    assert(copies.head.atEnds)
    assert(copies(1).readAt.nonEmpty)
  }

  test(
    "native state admission rejects text and unsupported leaves even with custom finite evidence"
  ) {
    assertEquals(
      typeCheckErrors("""
      import framework.dsl.*
      case class Good(seen: Set[Int], pending: List[Int], counts: Map[Int, Option[Int]])
      summon[StateData[Good]]
      summon[StateData[Int]]
    """),
      Nil
    )
    assertEquals(
      typeCheckErrors("""
      import framework.dsl.*
      case class Held(box: Inbox[Boolean])
      case class Nested(held: Held, items: List[Int])
      summon[StateData[Nested]]
    """),
      Nil
    )
    val unsupported = List(
      compileErrors("""summon[framework.dsl.StateData[List[framework.dsl.Inbox[Boolean]]]]"""),
      compileErrors("""summon[framework.dsl.StateData[Set[framework.dsl.Inbox[Boolean]]]]"""),
      compileErrors("""summon[framework.dsl.StateData[Map[Int, framework.dsl.Inbox[Boolean]]]]"""),
      compileErrors("""summon[framework.dsl.StateData[Map[framework.dsl.Inbox[Boolean], Int]]]"""),
      compileErrors("""
        import framework.dsl.*
        case class Held(box: Inbox[Boolean])
        summon[StateData[List[Held]]]
      """),
      compileErrors("""
        import framework.dsl.*
        case class Recursive(next: Option[Recursive])
        summon[StateData[Recursive]]
      """)
    )
    unsupported.foreach(errors => assert(errors.nonEmpty))
    assert(compileErrors("""summon[framework.dsl.StateData[String]]""").nonEmpty)
    assert(compileErrors("""summon[framework.dsl.StateData[List[String]]]""").nonEmpty)
    assert(compileErrors("""summon[framework.dsl.StateData[Set[String]]]""").nonEmpty)
    assert(compileErrors("""summon[framework.dsl.StateData[Map[String, Int]]]""").nonEmpty)
    assert(compileErrors("""summon[framework.dsl.StateData[Map[Int, Option[String]]]]""").nonEmpty)
    assert(compileErrors("""summon[framework.dsl.StateData[Int => Int]]""").nonEmpty)
    assert(compileErrors("""
      import framework.dsl.*
      given Finite[String] = Finite.of("text")
      object Bad extends Machine[String, Boolean, Nothing]:
        val init = "text"
        def end(s: String) = true
        object rules extends Rules
    """).nonEmpty)
    assert(compileErrors("""
      import framework.dsl.*
      case class Bad(callback: Int => Int)
      summon[StateData[Bad]]
    """).nonEmpty)
    assert(compileErrors("""
      import framework.dsl.*
      case class Bad(text: Option[String])
      given Finite[Bad] = Finite.of(Bad(Some("text")))
      summon[StateData[Bad]]
    """).nonEmpty)
  }

  test("outcomes facts and action inputs still require finite catalogs") {
    assert(compileErrors("""
      import framework.dsl.*
      object Bad extends Machine[Int, Int, Nothing]:
        val init = 0
        def end(s: Int) = true
        object rules extends Rules
    """).nonEmpty)
    assert(compileErrors("""summon[framework.dsl.Finite[List[Int]]]""").nonEmpty)
    assert(compileErrors("""
      import framework.dsl.*
      object Bad extends Machine[Int, Boolean, Int]:
        val init = 0
        def end(s: Int) = true
        object rules extends Rules
    """).nonEmpty)
    assert(compileErrors("""
      import framework.dsl.*
      object Bad extends Actor:
        val pick = action(this).input[Int](name = "number")
    """).nonEmpty)
  }

  test("native Int arithmetic is non-authoritative 32-bit host smoke behavior") {
    val minimum = Int.MinValue
    val maximum = Int.MaxValue
    assertEquals(minimum, -2147483647 - 1)
    assertEquals(maximum, 2147483647)
    assertEquals(maximum + 1, minimum)
    assertEquals(minimum - 1, maximum)
    assertEquals(maximum * 2, -2)
    assertEquals(minimum / -1, minimum)
    val zero = 0
    intercept[ArithmeticException](maximum / zero): Unit
  }
