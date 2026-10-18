package framework.dsl

import framework.checker.Checker
import framework.dsl.Finite.given

object ParametricMachinesFixture:
  enum Answer:
    case ok
  given Ok[Answer] = Ok(Answer.ok)
  enum Address:
    case alice, bob
  final case class Ledger[D](balances: Map[D, Int])
  final case class Transfer[D](to: D, amount: Boolean)

  class Model[D: Finite](holder: D, supply: Int) extends Machine[Answer, Nothing]:
    var ledger = Ledger(Map(holder -> supply))
    var opening = oneOf(0 to supply)

    object client extends Actor:
      lazy val send = action.inputs[Transfer[D]]()

    object transitions extends Transitions:
      on(client.send): input =>
        effects.send(input)

    object effects:
      def send(input: Transfer[D]) =
        ledger = Ledger(Map(input.to -> supply))

    object invariants:
      val supplyHeld =
        check(ledger.balances.values.sum == supply)
      val smallSupply =
        check(supply == 1)

    object postconditions:
      val openingUnchanged =
        check(opening == before.opening)
      val typedLedger =
        check(ledger.balances.values.sum == before.ledger.balances.values.sum)

    object monitors:
      val largeSupply = monitor(false)(seen => seen || supply == 2)(identity)

    object progress:
      val becomesSmall = (whenever(true):
        eventually(supply == 1)
      ) in only(client.send).fair(client.send)

  class Constants(enabled: Boolean, supply: Int, choices: Set[Int])
      extends Machine[Answer, Nothing]:
    var opening = oneOf(if enabled then choices else Set(supply))

    object transitions extends Transitions

  class Domain[D: Finite](initial: Set[D]) extends Machine[Answer, Nothing]:
    var selected = initial

    object transitions extends Transitions

  class MapConstants(entries: Map[Int, Set[Boolean]]) extends Machine[Answer, Nothing]:
    var selected = entries

    object transitions extends Transitions

  class Owned[D: Finite](holder: D) extends Machine[Answer, Nothing]:
    enum Status:
      case waiting, ready
    final case class Record(holder: D, status: Status)
    var record = Record(holder, Status.waiting)

    object client extends Actor:
      lazy val advance = action

    object transitions extends Transitions:
      on(client.advance):
        effects.advance()

    object effects:
      def advance() =
        record = Record(holder, Status.ready)

    object postconditions:
      val typedRecord =
        val previous: Record = before.record
        check(record.holder == previous.holder)
        check(record.status == Status.ready)

  class Multiple[A: Finite, B: Finite](left: A, right: B) extends Machine[Answer, Nothing]:
    var selected = Map(left -> Set(right))

    object transitions extends Transitions

  trait BorrowedClaims extends Facet[Answer, Nothing]:
    var active: Boolean

    object invariants:
      val impossible =
        check(active)

  class Assembled extends Machine[Answer, Nothing], BorrowedClaims:
    var active = false

    object transitions extends Transitions

  final class Acquisition:
    var reads = 0
    var catalogs = 0
    val domain: Finite[Int] = new Finite[Int]:
      def values =
        catalogs += 1
        Vector(0)
    def fail(): Unit = throw AssertionError("failed class claim continued")
    class Model(enabled: Boolean) extends Machine[Answer, Nothing]:
      var active = false

      object transitions extends Transitions

      object invariants:
        val first =
          reads += 1
          check(domain.values.nonEmpty)
          check(false)
          fail()
        val adjacent =
          check(true)
        val scoped =
          check(enabled) in starts()

  object Pair extends Composition[Nothing]:
    val left = Model(Address.alice, 1)
    val right = Model(Address.bob, 2)

    object postconditions:
      val typedMember =
        val previous: Ledger[Address] = before.left.ledger
        check(previous.balances.values.sum == left.ledger.balances.values.sum)

class ParametricMachinesTest extends munit.FunSuite:
  import ParametricMachinesFixture.*

  test("Finite-only generic fields and input records retain each instance's selected type"):
    val small = Model(Address.alice, 1)
    val large = Model(Address.bob, 2)
    assertEquals(small.initialStates.size, 2)
    assertEquals(large.initialStates.size, 3)
    assertNotEquals(small.init, large.init)
    assertEquals(Checker.explore(small).subject, "small")
    assertEquals(Checker.explore(large).subject, "large")
    assert(Checker.explore(small, budget = 4).complete)
    assert(!Checker.explore(small, budget = 3).complete)
    assert(Checker.explore(large, budget = 6).complete)
    assert(!Checker.explore(large, budget = 5).complete)
    assertEquals(small.ledger, Ledger(Map(Address.alice -> 1)))
    assertEquals(large.ledger, Ledger(Map(Address.bob -> 2)))
    val stays = Model[Boolean](false, 1)(using Finite.of(false))
    val moves = Model[Boolean](false, 1)(using Finite.of(true))
    assert(Checker.explore(stays, budget = 2).complete)
    assert(!Checker.explore(moves, budget = 2).complete)
    assert(Checker.explore(moves, budget = 4).complete)
    assertEquals(stays.ledger, Ledger(Map(false -> 1)))
    assertEquals(moves.ledger, Ledger(Map(false -> 1)))

  test("class-owned claims evaluate on their exact instance"):
    val small = Model(Address.alice, 1)
    val large = Model(Address.bob, 2)
    val first =
      Checker.verifyAll(small).results.map(result => result.subject -> result.verdict).toMap
    val second =
      Checker.verifyAll(large).results.map(result => result.subject -> result.verdict).toMap
    assertEquals(first("small.supplyHeld"), framework.checker.Verdict.verified)
    assertEquals(first("small.smallSupply"), framework.checker.Verdict.verified)
    assertEquals(second("large.supplyHeld"), framework.checker.Verdict.verified)
    assertEquals(second("large.smallSupply"), framework.checker.Verdict.refuted)
    assertEquals(first("small.typedLedger"), framework.checker.Verdict.verified)
    assertEquals(second("large.typedLedger"), framework.checker.Verdict.verified)

  test("runtime snapshots restore independent owners and refuse exact foreign states"):
    val small = Model(Address.alice, 1)
    val large = Model(Address.bob, 2)
    val initial = small.init
    val changed = initial.copy(opening = 1)
    small.at(changed):
      assertEquals(small.opening, 1)
      assertEquals(large.at(large.init)(large.opening), 0)
      assertEquals(small.opening, 1)
    assertEquals(small.opening, 0)
    assert(small.init.eq(initial))
    val error = intercept[IllegalArgumentException](small.at(large.init)(fail("foreign body ran")))
    assertEquals(
      error.getMessage,
      s"${large.init} is a state of another machine than ${small.name}"
    )
    assertEquals(large.ledger, Ledger(Map(Address.bob -> 2)))

  test("class-owned monitors and progress retain their instance's parameters"):
    val small = Model(Address.alice, 1)
    val large = Model(Address.bob, 2)
    assertEquals(
      Checker.monitor(small, small.monitors.largeSupply, small.initialStates).verdict,
      framework.checker.Verdict.verified
    )
    assertEquals(
      Checker.monitor(large, large.monitors.largeSupply, large.initialStates).verdict,
      framework.checker.Verdict.refuted
    )
    assertEquals(
      Checker.progress(small, small.progress.becomesSmall, small.initialStates).verdict,
      framework.checker.Verdict.verified
    )
    assertEquals(
      Checker.progress(large, large.progress.becomesSmall, large.initialStates).verdict,
      framework.checker.Verdict.refuted
    )

  test("constant initial domains keep ordinary Int Boolean Set[Int] and empty-domain semantics"):
    val choices = Constants(true, 9, Set(0, 1))
    val fixed = Constants(false, 9, Set.empty)
    assertEquals(choices.initialStates.map(_.selectDynamic("opening")).toSet, Set[Any](0, 1))
    assertEquals(fixed.initialStates.map(_.selectDynamic("opening")), Seq[Any](9))
    val nested = MapConstants(Map(3 -> Set(true, false)))
    assertEquals(nested.at(nested.init)(nested.selected), Map(3 -> Set(true, false)))
    assertEquals(Checker.explore(nested).verdict, framework.checker.Verdict.verified)
    assertEquals(
      intercept[IllegalArgumentException](Constants(true, 9, Set.empty)).getMessage,
      "requirement failed: opening: oneOf takes a non-empty domain"
    )

  test("aliases and helper checks retain the first checked stable holder"):
    val original = Model(Address.alice, 1)
    val alias = original
    assertEquals(Checker.explore(original).subject, "original")
    assertEquals(Checker.explore(alias).subject, "original")
    def helper(machine: NativeModel[State, Answer, Nothing]) = Checker.explore(machine)
    assertEquals(helper(alias).subject, "original")
    def factory() = Model(Address.bob, 2)
    val heldFactory = factory()
    assertEquals(Checker.explore(heldFactory).subject, "heldFactory")
    val direct = compileErrors("""
      import framework.dsl.ParametricMachinesFixture.*
      framework.checker.Checker.explore(Model(Address.alice, 1))
    """)
    assert(direct.contains("Model") && direct.contains("stable val holder"), direct)
    val computed = intercept[IllegalArgumentException](Checker.explore(factory())).getMessage
    assert(computed.contains("Model") && computed.contains("stable val holder"), computed)
    assert(computed.contains("ParametricMachines.test.scala:"), computed)

  test("selected abstract collection members refuse unsupported shapes without reading catalogs"):
    given Finite[String] = new Finite[String]:
      def values = throw AssertionError("unsupported domain catalog read")
    val unsupported = Domain(Set("text"))
    val error = intercept[IllegalArgumentException](Checker.explore(unsupported))
    assert(error.getMessage.contains("supported native members"), error.getMessage)

  test("selected Nothing keeps the lawful empty native collection"):
    val _ = summon[StateData[Set[Nothing]]]
    val empty = Domain[Nothing](Set.empty)
    assertEquals(Checker.explore(empty).verdict, framework.checker.Verdict.verified)
    assertEquals(empty.at(empty.init)(empty.selected), Set.empty[Nothing])
    val custom = new Data.Structural[Nothing]:
      def compare(left: Nothing, right: Nothing): Int = left
    val selected = new Finite[Nothing](using custom):
      def values = throw AssertionError("custom empty catalog read")
    val unsupported = new Domain[Nothing](Set.empty)(using selected)
    val error = intercept[IllegalArgumentException](Checker.explore(unsupported))
    assert(error.getMessage.contains("supported native members"), error.getMessage)

  test("class-owned vocabulary and two independent Finite bounds retain selected field types"):
    val owned = Owned(Address.alice)
    assertEquals(owned.at(owned.init)(owned.record.holder), Address.alice)
    assert(Checker.explore(owned, budget = 2).complete)
    assert(!Checker.explore(owned, budget = 1).complete)
    assertEquals(
      Checker.verifyAll(owned).results.map(_.verdict),
      Vector(framework.checker.Verdict.verified)
    )
    val multiple = Multiple(Address.alice, true)
    assertEquals(multiple.at(multiple.init)(multiple.selected), Map(Address.alice -> Set(true)))

  test("composition executes exact parameterized member snapshots and claims"):
    val result = Checker.explore(Pair, budget = 24)
    assert(result.complete)
    assert(!Checker.explore(Pair, budget = 23).complete)
    val report = Checker.verifyAll(result)
    val answers = report.results.map(r => r.subject -> r.verdict).toMap
    assertEquals(answers("pair.typedMember"), framework.checker.Verdict.verified)
    assertEquals(Pair.left.ledger, Ledger(Map(Address.alice -> 1)))
    assertEquals(Pair.right.ledger, Ledger(Map(Address.bob -> 2)))

  test("an already-bound instance keeps its holder through a concrete return helper"):
    val original = Model(Address.alice, 1)
    assertEquals(Checker.explore(original).subject, "original")
    def same(model: Model[Address]): Model[Address] = model
    assertEquals(Checker.explore(same(original)).subject, "original")

  test("claim references and direct sections refuse another instance of the same class"):
    val source = Model(Address.alice, 1)
    val target = Model(Address.bob, 2)
    val foreign = verify(source.invariants.smallSupply)
    val graph = framework.checker.NativeCheck.explore(target, target.initialStates)
    val error = intercept[IllegalArgumentException](
      framework.checker.NativeCheck.verify(graph, "borrowed", foreign.property)
    ).getMessage
    assert(error.contains("another Model instance"), error)
    val sections = intercept[IllegalArgumentException](
      framework.checker.NativeCheck.verifyAll(target, target.initialStates, sections = Some(source))
    ).getMessage
    assert(sections.contains("another Model instance"), sections)

  test("facet-owned Unit claims are refused before their inherited getter runs"):
    val assembled = new Assembled
    val error = intercept[IllegalArgumentException](Checker.verifyAll(assembled)).getMessage
    assert(error.contains("claim section of an instance"), error)

  test("class claim metadata skips Unit callbacks and evaluation keeps short-circuit boundaries"):
    val fixture = new Acquisition
    val model = new fixture.Model(true)
    val selected = verify(model.invariants.scoped)
    assertEquals(fixture.reads, 0)
    assertEquals(fixture.catalogs, 0)
    assertEquals(selected.property.machine, model)
    val report = Checker.verifyAll(model)
    val answers = report.results.map(r => r.subject -> r.verdict).toMap
    assertEquals(answers("model.first"), framework.checker.Verdict.refuted)
    assertEquals(answers("model.adjacent"), framework.checker.Verdict.verified)
    assertEquals(answers("model.scoped"), framework.checker.Verdict.verified)
    assertEquals(fixture.reads, 2)
    assertEquals(fixture.catalogs, 2)

  test("class-owned progress refuses another instance's captured parameters"):
    val source = Model(Address.alice, 1)
    val target = Model(Address.bob, 2)
    val error = intercept[IllegalArgumentException](
      Checker.progress(target, source.progress.becomesSmall, target.initialStates)
    ).getMessage
    assert(error.contains("another Model instance"), error)

    val shared = LeadsTo.declared(
      "shared",
      None,
      () => true,
      () => true,
      None,
      source.progress.becomesSmall.scenario
    )
    assertEquals(
      Checker.progress(target, shared, target.initialStates).verdict,
      framework.checker.Verdict.verified
    )

  test("temporary member receivers and helper-local aliases do not invent a holder"):
    class Container:
      val member = Model(Address.alice, 1)
    val container = new Container
    assertEquals(Checker.explore(container.member).subject, "member")
    val temporary =
      intercept[IllegalArgumentException](Checker.explore((new Container).member)).getMessage
    assert(temporary.contains("Model") && temporary.contains("stable val holder"), temporary)

  test("helper-local aliases preserve existing holders and refuse to invent internal names"):
    def helper(machine: NativeModel[State, Answer, Nothing]) =
      val internal = machine
      Checker.explore(internal)
    val error = intercept[IllegalArgumentException](helper(Model(Address.alice, 1))).getMessage
    assert(error.contains("Model") && error.contains("stable val holder"), error)
    val held = Model(Address.alice, 1)
    assertEquals(Checker.explore(held).subject, "held")
    assertEquals(helper(held).subject, "held")

  test(
    "constructor parameters require finite domains and liftable constants at their declarations"
  ):
    val unbounded = compileErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      given Ok[Boolean] = Ok(true)
      class Unbounded[D](domain: Set[D]) extends Machine[Boolean, Nothing]:
        var count = domain.size
        object transitions extends Transitions
    """)
    val path = compileErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      given Ok[Boolean] = Ok(true)
      class Bad(path: java.nio.file.Path) extends Machine[Boolean, Nothing]:
        var count = path.getNameCount
        object transitions extends Transitions
    """)
    val nested = compileErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      given Ok[Boolean] = Ok(true)
      class Bad(entries: Set[Map[Int, String]]) extends Machine[Boolean, Nothing]:
        var count = entries.size
        object transitions extends Transitions
    """)
    val callback = compileErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      given Ok[Boolean] = Ok(true)
      class Bad(callback: Int => Int) extends Machine[Boolean, Nothing]:
        var count = callback(0)
        object transitions extends Transitions
    """)
    assert(unbounded.contains("D") && unbounded.contains("Finite"), unbounded)
    for (error, name) <- List(path -> "path", nested -> "entries", callback -> "callback") do
      assert(error.contains(name) && error.contains("constant"), error)

  test("a shadowed outer finite domain does not license an unbounded inner type parameter"):
    val error = compileErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      given Ok[Boolean] = Ok(true)
      class Holder[D: Finite]:
        type Outside = D
        class Model[D](holder: D)(using Finite[Outside]) extends Machine[Boolean, Nothing]:
          var count = holder.hashCode
          object transitions extends Transitions
    """)
    assert(error.contains("D") && error.contains("Finite bound"), error)
