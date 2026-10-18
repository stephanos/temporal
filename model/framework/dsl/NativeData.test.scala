package framework.dsl

import framework.checker.*
import framework.checker.Effects.*
import framework.dsl.Finite.given
import scala.compiletime.testing.typeCheckErrors

object NativeDataFixture:
  enum Flag:
    case first, second
  final case class Pair(flag: Flag, count: Int)
  final case class Collections(seen: Set[Int], pending: List[Int], counts: Map[Int, Int])
  final case class Poison(flag: Flag, ready: Boolean)

  object caller extends Actor:
    val act = action
    val extra = action

class NativeDataTest extends munit.FunSuite:
  import NativeDataFixture.*
  given Ok[Boolean] = Ok(true)

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
    object Native extends Machine[Boolean, Nothing]:
      var held = Held(wire.empty, Nil)
      override def end: Boolean = held.inbox.isEmpty
      object transitions extends Transitions:
        disabled(wire.deliver)
    assertEquals(Native.bindings.size, 1)
    val held = Held(wire.empty.send(Flag.second).send(Flag.first), Nil)
    val sent = Native.init.copy(held = held)
    assertEquals(held.inbox.deliveries.map(_.message), List(Flag.first, Flag.second))
    assert(Data[State].equivalent(sent, sent.copy(held = held.copy())))
  }

  test("custom finite action order and unordered Inbox type order are independent") {
    given Finite[Int] = Finite.of(2, 0)
    val pick = action(Actor("inputs")).input[Int](name = "number")
    assertEquals(classesOf(pick.decl), List(List(2), List(0)))
    assert(Data[Int].compare(0, 2) < 0)
    val data: Data[Int] = Finite[Int]
    assert(data.compare(0, 2) < 0)
    object NumericOutcome extends Machine[Int, Nothing]:
      var collections = Collections(Set.empty, Nil, Map.empty)
      override def end: Boolean = true
      object transitions extends Transitions(using summon, Ok(0), summon):
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
    object Native extends Machine[Boolean, Nothing]:
      var seen = Set.empty[Int]
      var pending = List.empty[Int]
      var counts = Map.empty[Int, Int]
      override def end: Boolean = pending.isEmpty
      object effects:
        def append(): Unit =
          seen = seen.incl(2)
          pending = pending :+ 2
          counts = counts.updated(2, counts.getOrElse(2, 0) + 1)
      object transitions extends Transitions:
        on(caller.act):
          effects.append()
    val initial = Native.init
    assertEquals(Native.counts.get(2), None)
    assertEquals(Native.counts.getOrElse(2, 7), 7)
    intercept[NoSuchElementException](Native.counts(2)): Unit
    val run = effectOf[State, Boolean, Nothing](caller.act.decl, Native.bindings.head.function)
    val next = run(initial, Nil).head.state
    assertEquals(next, initial.copy(seen = Set(2), pending = List(2), counts = Map(2 -> 1)))
    assertEquals(stuck(initial).states, Seq(initial))
  }

  test("declared and rebound rules construct without enumerating state or invoking callbacks") {
    given Ok[Boolean] = Ok(true)
    given Finite[Poison] = new Finite[Poison]:
      def values = throw AssertionError("construction enumerated whole-state catalog")
    def guard(s: Poison): Boolean = throw AssertionError("construction invoked guard")
    object Native extends Machine[Boolean, Nothing]:
      var poison = Poison(Flag.first, false)
      override def end: Boolean = false
      object effects:
        def fail(): Unit = throw AssertionError("construction invoked effect")
      object transitions extends Transitions:
        on(caller.act):
          if guard(poison) then effects.fail()
        on(caller.act):
          if guard(poison) then effects.fail()
    assertEquals(Native.bindings.size, 1)
    val derived = Native.rebind(on(caller.act):
      if guard(Native.poison) then Native.effects.fail()
      else if guard(Native.poison) then Native.effects.fail())
    assertEquals(derived.bindings.size, 1)
    val retained = derived.table.collect { case (_, Bound.Ruled(rules)) => rules }.flatten
    assertEquals(retained.map(_.index), Vector(1))
    assert(retained.forall(_.decl eq caller.act.decl))
    val forwarded = Native.unmonitored
    val extended = Native.extend(on(caller.extra):
      if guard(Native.poison) then Native.effects.fail())
    val refined =
      Native.refining(Native)(_ => throw AssertionError("construction invoked refinement"))
    object Copied extends Derived(derived)
    for machine <- List(derived, forwarded, extended, refined, Copied) do
      assert(machine.fs eq Native.fs)
      assert(machine.fo eq Native.fo)
      assert(machine.ff eq Native.ff)
      assertEquals(machine.init, Native.init)
    assertEquals(extended.bindings.map(_.decl), List(caller.act.decl, caller.extra.decl))
    assert(refined.refinementEvidence.get.of eq Native)
    object Left extends Machine[Boolean, Nothing]:
      var ready = false
      object transitions extends Transitions
    object Right extends Machine[Boolean, Nothing]:
      var ready = false
      object transitions extends Transitions
    object CopiedLeft extends Derived(Left.unmonitored)
    object Product extends Composition[Nothing]:
      val left = Left
      val right = Right
    object Shared extends Composition[Nothing]:
      val left = Left
      val right = CopiedLeft
    assertEquals(Product.members, Vector(Left, Right))
    assert(intercept[IllegalArgumentException](Shared.init).getMessage.contains("same vars"))
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
      given Ok[Boolean] = Ok(true)
      object Bad extends Machine[Boolean, Nothing]:
        var text = "text"
        object transitions extends Transitions
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
      object Bad extends Machine[Int, Nothing]:
        var count = 0
        object transitions extends Transitions(using summon, Ok(0), summon)
    """).nonEmpty)
    assert(compileErrors("""summon[framework.dsl.Finite[List[Int]]]""").nonEmpty)
    assert(compileErrors("""
      import framework.dsl.*
      given Ok[Boolean] = Ok(true)
      object Bad extends Machine[Boolean, Int]:
        var count = 0
        object transitions extends Transitions
    """).nonEmpty)
    assert(compileErrors("""
      import framework.dsl.*
      object Bad extends Actor:
        val pick = action.input[Int](name = "number")
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
