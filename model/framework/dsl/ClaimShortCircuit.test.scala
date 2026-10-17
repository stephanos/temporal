package framework.dsl

import framework.dsl.Finite.given
import framework.dsl.outcomes.Outcome

object ClaimShortCircuitFixture:

  object Guarded extends Machine[Outcome, Nothing]:
    var value: Option[Int] = None

    object transitions extends Transitions

    object invariants:
      val requiresValue =
        check(value.nonEmpty)
        check(value.get > 0)
      val independent = check(false)

  object Alternatives extends Machine[Outcome, Nothing]:
    var value: Option[Int] = None

    object transitions extends Transitions

    object invariants:
      val succeeds = oneOf:
        branch:
          check(false)
          check(value.get > 0)
        branch:
          check(true)
        branch:
          check(value.get > 0)
      val fails =
        oneOf:
          branch:
            check(false)
          branch:
            check(false)
        check(value.get > 0)
      val skipped =
        check(false)
        oneOf:
          branch:
            check(value.get > 0)
          branch:
            check(value.get > 0)
      val nested = oneOf:
        branch:
          oneOf:
            branch:
              check(false)
            branch:
              check(true)
          check(false)
          check(value.get > 0)
        branch:
          check(true)
      val independent = check(false)

  object Missing extends Machine[Outcome, Nothing]:
    var value = false
    val missing = hole

    object transitions extends Transitions

    object invariants:
      val reached = oneOf:
        branch:
          check(false)
        branch:
          check(missing.reached)

  object Recorded extends Machine[Outcome, Boolean]:
    var value: Option[Boolean] = None

    object transitions extends Transitions

    object invariants:
      val guarded =
        check(value.nonEmpty)
        records(value.get)

  object WholeMember extends Machine[Outcome, Nothing]:
    var value: Option[Int] = None

    object transitions extends Transitions

    object invariants:
      val helper =
        check(value.nonEmpty)
        val held = value.get
        check(held > 0)
      val guard =
        check(value.nonEmpty)
        if value.get > 0 then check(true)
      val independent =
        check(true)

  object BranchHelpers extends Machine[Outcome, Nothing]:
    var value: Option[Int] = None

    object transitions extends Transitions

    object invariants:
      val succeeds =
        oneOf:
          branch:
            check(value.nonEmpty)
            val held = value.get
            check(held > 0)
          branch:
            check(true)

    object queries:
      val guardedTarget =
        check(value.nonEmpty)
        val held = value.get
        check(held > 0)
      val findTarget =
        find(guardedTarget)

  object MutableHelper extends Machine[Outcome, Nothing]:
    var value: Option[Int] = None

    object transitions extends Transitions

    object invariants:
      private var counter = 0
      val guarded =
        check(value.nonEmpty)
        counter += 1
        val held = value.get
        check(held > 0)
      val independent =
        check(counter == 0)

  object ShadowedLocal extends Machine[Outcome, Nothing]:
    var value = false

    object transitions extends Transitions

    object invariants:
      val guarded =
        val independent = ()
        check(value)
        independent
      val independent =
        check(true)

  object MixedClaims extends Machine[Outcome, Nothing]:
    var value: Option[Int] = None

    object transitions extends Transitions

    object invariants:
      val guarded =
        check(value.nonEmpty)
        val held = value.get
        check(held > 0)
      val scoped =
        check(true) in only()

    object queries:
      val scopedQuery =
        verify(invariants.scoped)

  object PassingHelper extends Machine[Outcome, Nothing]:
    var value: Option[Int] = Some(1)

    object transitions extends Transitions

    object invariants:
      private var counter = 0
      val guarded =
        check(value.nonEmpty)
        counter += 1
        val held = value.get
        check(held > 0)
      val independent =
        check(counter == 1)

  object NakedHole extends Machine[Outcome, Nothing]:
    var value = false
    val missing = hole

    object transitions extends Transitions

    object invariants:
      val guarded =
        if missing.reached then check(value)
      val independent =
        check(true)

  object worker extends Actor:
    val advance = action

  object TargetHole extends Machine[Outcome, Nothing]:
    var value: Option[Int] = None
    val missing = hole

    object transitions extends Transitions:
      on(worker.advance):
        effects.advance()

    object effects:
      def advance() =
        value = None

    object postconditions:
      val guarded =
        after(worker.advance):
          if missing.reached then check(true)
        val held = value.get
        check(held > 0)
      val independent =
        check(true)

  object OrdinaryFailure extends Machine[Outcome, Nothing]:
    var value = false

    object transitions extends Transitions

    object invariants:
      private def fail(): Int =
        throw IllegalStateException("a reached helper failed")
      val guarded =
        check(true)
        val held = fail()
        check(held > 0)

  class InstanceClaims:

    object invariants:
      val independent = ()

  object ForwardField extends Machine[Outcome, Nothing]:
    var value: Option[Int] = None

    object transitions extends Transitions

    object invariants:
      val guarded =
        check(value.nonEmpty)
        counter += 1
        val held = value.get
        check(held > 0)
      private var counter = 0
      val independent =
        check(counter == 0)

  object ScopedQueryTarget extends Machine[Outcome, Nothing]:
    var value: Option[Int] = None

    object transitions extends Transitions:
      on(worker.advance):
        effects.advance()

    object effects:
      def advance() =
        value = None

    object queries:
      val guarded =
        check(value.nonEmpty)
        val held = value.get
        check(held > 0)
      val target =
        check(true) in only(worker.advance)
      val findTarget =
        find(target)

object InvalidNativeClaimBoundaryFixture:

  object invariants:
    def unsupported(): scala.runtime.BoxedUnit =
      throw IllegalStateException("an unsupported getter was invoked")

object SameLineClaimBoundaryFixture extends Machine[Outcome, Nothing]:
  var value = false

  object transitions extends Transitions

  object invariants:
    // format: off
    val a = check(true); private val env = only(); val b = check(true); val scoped = check(true) in env
    // format: on

object FirstSameLineClaimBoundaryFixture extends Machine[Outcome, Nothing]:
  var value = false

  object transitions extends Transitions

  object invariants:
    // format: off
    private var env = only(); val a = check(true)
    // format: on
    val scoped =
      check(true) in env
    env = only()

class ClaimShortCircuitSuite extends munit.FunSuite:
  import ClaimShortCircuitFixture.*
  import NativeCheck.Verdict.*

  test("a failed requirement skips later assertion conditions in its member"):
    val report = NativeCheck.verifyAll(Guarded)
    assertEquals(report.verdicts, Map("requiresValue" -> refuted, "independent" -> refuted))
    assert(report.result("requiresValue").detail.last.endsWith(": value.nonEmpty"))

  test("oneOf skips failed branch assertions and alternatives after its first success"):
    assertEquals(
      NativeCheck.verifyAll(Alternatives).verdicts,
      Map(
        "succeeds" -> verified,
        "fails" -> refuted,
        "skipped" -> refuted,
        "nested" -> verified,
        "independent" -> refuted
      )
    )

  test("an actually evaluated hole leaves the claim unknown"):
    val result = NativeCheck.verifyAll(Missing).result("reached")
    assertEquals(result.verdict, unknown)
    assert(!result.complete)
    assert(result.detail.exists(_.contains("missing")))

  test("a failed requirement skips the argument of a later fact assertion"):
    assertEquals(NativeCheck.verifyAll(Recorded).verdicts, Map("guarded" -> refuted))

  test("a fact assertion evaluates its argument once when reached"):
    var evaluated = 0
    def fact =
      evaluated += 1
      true
    val context = NativeClaims.Context(recorded = List(true))
    NativeClaims.within(context):
      NativeClaims.recordFact(fact, "recorded", 1, true)
    assertEquals(evaluated, 1)
    assertEquals(context.checks.map(_.holds), Vector(true))

  test("a failed requirement skips helper vals and naked guards but preserves later members"):
    val report = NativeCheck.verifyAll(WholeMember)
    assertEquals(
      report.verdicts,
      Map("helper" -> refuted, "guard" -> refuted, "independent" -> verified)
    )

  test("a failed branch skips its helper vals before trying the next alternative"):
    assertEquals(NativeCheck.verifyAll(BranchHelpers).result("succeeds").verdict, verified)

  test("reading query declarations does not execute a find target's helper vals"):
    val results = NativeCheck.queries(BranchHelpers)
    assertEquals(results.map(_.verdict), Vector(refuted))

  test("a failed ad hoc requirement skips its helper vals"):
    val claim = NativeControls.invariant:
      check(WholeMember.value.nonEmpty)
      val held = WholeMember.value.get
      check(held > 0)
    val result = NativeControls.assertRefuted(WholeMember, claim)
    assertEquals(result.verdict, refuted)

  test("a section field assignment does not end its containing claim"):
    assertEquals(
      NativeCheck.verifyAll(MutableHelper).verdicts,
      Map("guarded" -> refuted, "independent" -> verified)
    )

  test("a local Unit val may share a later claim's name"):
    assertEquals(
      NativeCheck.verifyAll(ShadowedLocal).verdicts,
      Map("guarded" -> refuted, "independent" -> verified)
    )

  test("query declarations can reference a scoped claim beside a guarded Unit claim"):
    assertEquals(NativeCheck.queries(MixedClaims).map(_.verdict), Vector(verified))

  test("a direct scoped verify does not initialize guarded neighboring claims"):
    assertEquals(
      NativeCheck.query(verify(MixedClaims.invariants.scoped), "mixed.scoped").verdict,
      verified
    )

  test("an unsupported claim boundary retains its declaration diagnostic"):
    val failed = intercept[IllegalArgumentException]:
      NativeCheck.section(InvalidNativeClaimBoundaryFixture, "invariants")
    assert(
      failed.getMessage.contains(
        "native claim boundaries require each Unit val's initializer scope"
      )
    )

  test("a successful guard executes its helpers exactly once"):
    assertEquals(
      NativeCheck.verifyAll(PassingHelper).verdicts,
      Map("guarded" -> verified, "independent" -> verified)
    )

  test("a naked guard hole affects only its own member"):
    assertEquals(
      NativeCheck.verifyAll(NakedHole).verdicts,
      Map("guarded" -> unknown, "independent" -> verified)
    )

  test("a hole in an after block skips the rest of its member"):
    assertEquals(
      NativeCheck.verifyAll(TargetHole).verdicts,
      Map("guarded" -> unknown, "independent" -> verified)
    )

  test("ordinary helper failures still propagate"):
    val failed = intercept[IllegalStateException]:
      NativeCheck.verifyAll(OrdinaryFailure)
    assertEquals(failed.getMessage, "a reached helper failed")

  test("feature discovery skips unrelated instance sections"):
    assertEquals(NativeCheck.verifyAll(WholeMember).result("independent").verdict, verified)

  test("a Property alias outside a Model remains a query target"):
    val errors = compileErrors("""
      import framework.dsl.*
      object Shared:
        object invariants:
          val alias: Property = ???
      verify(Shared.invariants.alias)
    """)
    assertEquals(errors, "")

  test("a write to a later field does not split the current claim"):
    assertEquals(
      NativeCheck.verifyAll(ForwardField).verdicts,
      Map("guarded" -> refuted, "independent" -> verified)
    )

  test("ambiguous same-line claim boundaries fail before declaration helpers are skipped"):
    val failed = intercept[IllegalArgumentException]:
      NativeCheck.section(SameLineClaimBoundaryFixture, "invariants")
    assert(failed.getMessage.contains("native claim initializer has no distinct source line"))

  test("an ambiguous mutable initializer before the first claim fails before helpers are skipped"):
    val failed = intercept[IllegalArgumentException]:
      NativeCheck.section(FirstSameLineClaimBoundaryFixture, "invariants")
    assert(failed.getMessage.contains("native claim initializer has no distinct source line"))

  test(
    "a scoped find target in its query section initializes without recursive declaration loading"
  ):
    assertEquals(NativeCheck.queries(ScopedQueryTarget).map(_.verdict), Vector(found))
