package umpire.migrate

import scala.meta.*
import scalafix.v1.{SyntacticDocument, SyntacticRule}
import scalafix.internal.v1.Rules

class FacetWrappingTest extends munit.FunSuite:
  private def rewrite(source: String) =
    val rule = try
      Class
        .forName("umpire.migrate.Fn161Task4FacetWrapping")
        .getDeclaredConstructor()
        .newInstance()
        .asInstanceOf[SyntacticRule]
    catch case _: ClassNotFoundException => fail("the compiled facet-wrapping rule is missing")
    val tree = dialects.Scala3(Input.String(source)).parse[Source].get
    Rules(List(rule)).syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)

  private val prefix = """package temporal
package features.activity
package mini
package protocol

import framework.*
"""
  private val legacy = prefix + """
object AnswerDispatch extends Machine[AnswerState, Outcome, AnswerFact], Phased[AnswerState, AnswerPhase]:
  val init = AnswerState(AnswerPhase.waiting)
  object answering:
    object effects:
      def receive(s: AnswerState, answer: Answer) =
        enter(s.copy(phase = AnswerPhase.completed), AnswerFact.acceptedCurrent)
    object rules extends Rules:
      on(answers.deliver) { always ~> effects.receive }
  object rules extends Facets(answering.rules)
  object queries:
    val completed = witness(answers.deliver(Answer.firstCompleted)).limits(three)
"""
  private val expected = prefix + """
object AnswerDispatch extends Machine[AnswerState, Outcome, AnswerFact], Phased[AnswerState, AnswerPhase], Answering:
  val init = AnswerState(AnswerPhase.waiting)
  object rules extends Facets(answering.rules)
  object queries:
    val completed = witness(answers.deliver(Answer.firstCompleted)).limits(three)

trait Answering extends Facet[AnswerState, Outcome, AnswerFact], Phased[AnswerState, AnswerPhase]:

  object answering:
    object effects:
      def receive(s: AnswerState, answer: Answer) =
        enter(s.copy(phase = AnswerPhase.completed), AnswerFact.acceptedCurrent)
    object rules extends Rules:
      on(answers.deliver) { always ~> effects.receive }
"""

  test("owned namespace moves intact while machine declarations and aggregation stay ordered"):
    val result = rewrite(legacy)
    assertEquals(result.diagnostics, Nil)
    assertEquals(result.fixed, expected)
    assertEquals(rewrite(result.fixed).fixed, expected)
    assertEquals(rewrite(result.fixed).diagnostics, Nil)

  test("Completion preserves its typed Dispatching dependency, capability override and waiver"):
    val namespace = """  object completion:
    object effects:
      def fail(s: State) = enters(Phase.failed)(s)
    object rules extends Rules:
      on(activityWorker.respondFailed) { where(dispatching.states.waiting) ~> effects.fail }
    object capabilities extends Capabilities:
      val retryable: Capability = Retryable(failure)
      overriding(Retryable.failureReturnsToWaiting[State, Phase] -> own,
        because = "keep backoff distinct")
      except(Retryable.failureEndsFailed[State, Phase], because = "explicit exception")
"""
    val source = prefix + """
object ActivityProtocol extends Machine[State, Outcome, Fact], Phased[State, Phase], Dispatching:
""" + namespace + """  object rules extends Facets(dispatching.rules, completion.rules)
  object capabilities extends FacetCapabilities(dispatching.capabilities, completion.capabilities)
"""
    val want = prefix + """
object ActivityProtocol extends Machine[State, Outcome, Fact], Phased[State, Phase], Dispatching, Completion:
  object rules extends Facets(dispatching.rules, completion.rules)
  object capabilities extends FacetCapabilities(dispatching.capabilities, completion.capabilities)

trait Completion extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  this: Dispatching =>
""" + namespace
    val result = rewrite(source)
    assertEquals(result.diagnostics, Nil)
    assertEquals(result.fixed, want)

  test("unsupported ownership, aggregation, dependency and destination refuse atomically"):
    val refused = List(
      legacy.replace("AnswerState, Outcome, AnswerFact", "OtherState, Outcome, AnswerFact"),
      legacy.replace("Facets(answering.rules)", "Facets(other.rules, answering.rules)"),
      legacy.replace("object answering:", "object answering extends Other:"),
      legacy.replace("object effects:", "val own = action(actor)\n    object effects:"),
      legacy + "\ntrait Answering\n",
      legacy.replace("import framework.*", "import framework.*\nimport other.Facet"),
      legacy.replace("object rules extends Facets(answering.rules)", "object rules extends Rules"),
      prefix + """
object ActivityProtocol extends Machine[State, Outcome, Fact], Phased[State, Phase]:
  object completion:
    object rules extends Rules:
      disabled(client.cancel)
  object rules extends Facets(completion.rules)
"""
    )
    for source <- refused do
      val result = rewrite(source)
      assertEquals(result.fixed, source)
      assertEquals(result.diagnostics.size, 1, source)
      assert(result.diagnostics.head.message.contains("manually inventory"))

  test("current and unrelated machine namespaces are not rewritten"):
    for source <- List(expected, legacy.replace("package mini", "package other")) do
      val result = rewrite(source)
      assertEquals(result.fixed, source)
      assertEquals(result.diagnostics, Nil)

  test("nested machines refuse instead of producing a nested companion facet"):
    val nested = prefix + "object outer {\n" + legacy.stripPrefix(prefix) + "}\n"
    val result = rewrite(nested)
    assertEquals(result.fixed, nested)
    assertEquals(result.diagnostics.size, 1)

  test("leftover machine-owned effects refuse instead of partially wrapping"):
    val leftover = legacy.replace(
      "  val init =",
      "  object effects:\n    def stray(s: AnswerState) = stay(s)\n  val init ="
    )
    val result = rewrite(leftover)
    assertEquals(result.fixed, leftover)
    assertEquals(result.diagnostics.size, 1)

  test("an escaping core helper refuses without changing a second eligible machine"):
    val escaped = legacy
      .replace(
        "  val init =",
        "  def coreOnly(s: AnswerState) = s\n  val init ="
      )
      .replace("enter(s.copy", "enter(coreOnly(s).copy")
    val other = """
object ActivityProtocol extends Machine[State, Outcome, Fact], Phased[State, Phase], Dispatching:
  object completion:
    object rules extends Rules:
      disabled(client.cancel)
  object rules extends Facets(dispatching.rules, completion.rules)
"""
    val source = escaped + other
    val result = rewrite(source)
    assertEquals(result.fixed, source)
    assertEquals(result.diagnostics.size, 1)
