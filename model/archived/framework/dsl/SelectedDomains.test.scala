package nativeauthor

import framework.dsl.*
import framework.dsl.Finite.given
import framework.checker.{Checker, Verdict}
import scala.compiletime.testing.typeCheckErrors

object NativeSelectedDomainFixture:
  given Ok[Boolean] = Ok(true)

  enum Address:
    case alice, bob

  object Account:
    opaque type Value = Int
    val first: Value = 0
    given Finite[Value] = Finite.upTo(1)

  final case class Ledger[D](balances: Map[D, Int])
  final case class Transfer[D](to: D, enabled: Boolean)
  final case class Unsupported(label: String)

  class Model[D: Finite](holder: D) extends Machine[Boolean, Nothing]:
    var ledger = Ledger(Map(holder -> 1))
    var history = List(holder)
    var selected = Option(holder)

    object client extends Actor:
      lazy val send = action.inputs[Transfer[D]]()

    object transitions extends Transitions:
      on(client.send): input =>
        effects.send(input)

    object effects:
      def send(input: Transfer[D]): Unit =
        if input.enabled then
          ledger = Ledger(Map(input.to -> 1))
          history = List(input.to)
          selected = Some(input.to)

    object invariants:
      val supplyHeld =
        check(ledger.balances.values.sum == 1)

  object Poison:
    var catalogs = 0
    var guards = 0
    var effects = 0
    def reset(): Unit =
      catalogs = 0
      guards = 0
      effects = 0
    def catalog[A]: IndexedSeq[A] =
      catalogs += 1
      throw AssertionError("unsupported domain catalog evaluated")
    def guard: Boolean =
      guards += 1
      throw AssertionError("unsupported domain guard evaluated")
    def effect(): Unit =
      effects += 1
      throw AssertionError("unsupported domain effect evaluated")

  class Collection[D: Finite](initial: Set[D]) extends Machine[Boolean, Nothing]:
    var selected = initial

    object client extends Actor:
      lazy val send = action

    object transitions extends Transitions:
      on(client.send):
        if Poison.guard then effects.send()

    object effects:
      def send(): Unit = Poison.effect()

class NativeSelectedDomainTest extends munit.FunSuite:
  import NativeSelectedDomainFixture.*

  test("external Finite-only Models admit selected enum and opaque inputs and native state"):
    val enumOwner = Model(Address.alice)
    val opaqueOwner = Model(Account.first)
    assertEquals(enumOwner.ledger.balances.size, 1)
    assertEquals(opaqueOwner.ledger.balances.size, 1)
    assertEquals(Checker.explore(enumOwner).verdict, Verdict.verified)
    assertEquals(Checker.explore(opaqueOwner).verdict, Verdict.verified)
    assertEquals(
      Checker.verifyAll(enumOwner).results.map(r => r.subject -> r.verdict),
      Vector("enumOwner.supplyHeld" -> Verdict.verified)
    )
    assertEquals(
      Checker.verifyAll(opaqueOwner).results.map(r => r.subject -> r.verdict),
      Vector("opaqueOwner.supplyHeld" -> Verdict.verified)
    )
    assertEquals(enumOwner.ledger.balances, Map(Address.alice -> 1))
    assertEquals(opaqueOwner.ledger.balances, Map(Account.first -> 1))
    assertEquals(enumOwner.history, List(Address.alice))
    assertEquals(opaqueOwner.selected, Some(Account.first))

  test("external selected String collection refusal precedes catalogs guards and effects"):
    Poison.reset()
    given Finite[String] = new Finite[String]:
      def values = Poison.catalog[String]
    val unsupported = Collection(Set("text"))
    val error = intercept[IllegalArgumentException](Checker.explore(unsupported))
    assert(error.getMessage.contains("supported native members"), error.getMessage)
    assertEquals((Poison.catalogs, Poison.guards, Poison.effects), (0, 0, 0))

  test("external selected Inbox collection refusal precedes catalogs guards and effects"):
    Poison.reset()
    val wire = channel[Boolean](1, Order.fifo, Loss.reliable)
    given Finite[Inbox[Boolean]] = new Finite[Inbox[Boolean]]:
      def values = Poison.catalog[Inbox[Boolean]]
    val unsupported = Collection(Set(wire.empty))
    val error = intercept[IllegalArgumentException](Checker.explore(unsupported))
    assert(error.getMessage.contains("supported native members"), error.getMessage)
    assertEquals((Poison.catalogs, Poison.guards, Poison.effects), (0, 0, 0))

  test("external concrete collection admission retains the existing supported shapes"):
    assertEquals(
      typeCheckErrors("""
        import framework.dsl.*
        case class Good(seen: Set[Int], pending: List[Int], counts: Map[Int, Option[Int]])
        summon[StateData[Good]]
      """),
      Nil
    )
    val unsupported = List(
      compileErrors("""summon[framework.dsl.StateData[Set[String]]]"""),
      compileErrors("""summon[framework.dsl.StateData[List[framework.dsl.Inbox[Boolean]]]]"""),
      compileErrors("""summon[framework.dsl.StateData[Map[Int, framework.dsl.Inbox[Boolean]]]]""")
    )
    unsupported.foreach(errors => assert(errors.nonEmpty))

  test("external selected record leaves are validated without reading catalogs guards or effects"):
    Poison.reset()
    given Finite[Unsupported] = new Finite[Unsupported]:
      def values = Poison.catalog[Unsupported]
    val unsupported = Collection(Set(Unsupported("text")))
    val error = intercept[IllegalArgumentException](Checker.explore(unsupported))
    assert(error.getMessage.contains("supported native members"), error.getMessage)
    assertEquals((Poison.catalogs, Poison.guards, Poison.effects), (0, 0, 0))
