package umpire.migrate

import scala.meta.*
import scalafix.v1.{SyntacticDocument, SyntacticRule}
import scalafix.internal.v1.Rules
import java.nio.file.{Files, Path}
import java.lang.reflect.InvocationTargetException

class FacetWrappingTest extends munit.FunSuite:
  test("independent compiled inventory refuses both missing and additional declared machines"):
    val module =
      Class.forName("umpire.migrate.RewriteFacetWrappingMini$").getField("MODULE$").get(null)
    val method = try module.getClass.getMethod("verifyInventory", classOf[List[?]])
    catch
      case _: NoSuchMethodException =>
        fail("the independent selected-machine inventory gate is missing")
    val names = List(
      "temporal.actors.worker.Polling$",
      "temporal.foundations.taskqueue.product.TaskQueueProduct$",
      "temporal.foundations.taskqueue.protocol.TaskQueueProtocol$",
      "temporal.features.activity.mini.product.ActivityProduct$",
      "temporal.features.activity.mini.protocol.ActivityProtocol$",
      "temporal.features.activity.mini.protocol.AnswerDispatch$"
    )
    method.invoke(module, names)
    for changed <- List(
        names.tail,
        names :+ "temporal.features.activity.mini.protocol.Unaccounted$"
      )
    do
      val error = intercept[InvocationTargetException](method.invoke(module, changed))
      assert(error.getCause.isInstanceOf[IllegalArgumentException])

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

  test("a namespace at the machine's final source boundary does not consume the outer newline"):
    val source = prefix + """
object AnswerDispatch extends Machine[AnswerState, Outcome, AnswerFact], Phased[AnswerState, AnswerPhase]:
  object rules extends Facets(answering.rules)
  object answering:
    object rules extends Rules:
      disabled(client.cancel)
"""
    val want = prefix + """
object AnswerDispatch extends Machine[AnswerState, Outcome, AnswerFact], Phased[AnswerState, AnswerPhase], Answering:
  object rules extends Facets(answering.rules)

trait Answering extends Facet[AnswerState, Outcome, AnswerFact], Phased[AnswerState, AnswerPhase]:

  object answering:
    object rules extends Rules:
      disabled(client.cancel)
"""
    val result = rewrite(source)
    assertEquals(result.diagnostics, Nil)
    assertEquals(result.fixed, want)

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

  private val acceptedMachine = expected.take(expected.indexOf("\n\ntrait Answering")) + "\n"
  private val acceptedFacet = prefix + "\n" + expected.drop(expected.indexOf("trait Answering"))
  private def scratch() =
    val directory = Files.createTempDirectory(Path.of(".flow/tmp"), "facet-wrapping-file-test.")
    Files.writeString(directory.resolve("AnswerDispatch.scala"), legacy)
    directory
  private def split(
      directory: Path,
      fixed: String = expected,
      facet: String = "Answering.scala",
      before: String = legacy,
      pinMachine: String = acceptedMachine,
      pinFacet: String = acceptedFacet
  ) =
    val module = try
      Class.forName("umpire.migrate.RewriteFacetWrappingMini$").getField("MODULE$").get(null)
    catch
      case _: ClassNotFoundException =>
        fail("the compiled copied-source companion operation is missing")
    val method = module.getClass.getMethod(
      "writeCompanion",
      classOf[Path],
      classOf[String],
      classOf[String],
      classOf[String],
      classOf[String],
      classOf[String],
      classOf[String]
    )
    try
      method.invoke(
        module,
        directory,
        "AnswerDispatch.scala",
        before,
        fixed,
        facet,
        pinMachine,
        pinFacet
      )
    catch case e: InvocationTargetException => throw e.getCause

  test(
    "companion operation writes the matching facet and preserves the independently pinned machine"
  ):
    val directory = scratch()
    split(directory)
    assertEquals(Files.readString(directory.resolve("AnswerDispatch.scala")), acceptedMachine)
    assertEquals(Files.readString(directory.resolve("Answering.scala")), acceptedFacet)

  test("a companion collision refuses before either file changes"):
    val directory = scratch()
    val target = directory.resolve("Answering.scala")
    Files.writeString(target, "unrelated destination\n")
    intercept[IllegalArgumentException](split(directory))
    assertEquals(Files.readString(directory.resolve("AnswerDispatch.scala")), legacy)
    assertEquals(Files.readString(target), "unrelated destination\n")

  test("wrong destination and semantic corruption refuse without publishing a companion"):
    for (destination, fixed) <- List(
        "../escaped.scala" -> expected,
        "Other.scala" -> expected,
        "Answering.scala" -> expected
          .replace("AnswerFact.acceptedCurrent", "AnswerFact.acceptedStale")
      )
    do
      val directory = scratch()
      intercept[IllegalArgumentException](split(directory, fixed, destination))
      assertEquals(Files.readString(directory.resolve("AnswerDispatch.scala")), legacy)
      assert(!Files.exists(directory.resolve("Answering.scala")))

  test("machine and dangling companion symlinks refuse without modifying their targets"):
    val directory = scratch()
    val saved = directory.resolve("saved.txt")
    Files.move(directory.resolve("AnswerDispatch.scala"), saved)
    Files.createSymbolicLink(directory.resolve("AnswerDispatch.scala"), saved.toAbsolutePath)
    intercept[IllegalArgumentException](split(directory))
    assertEquals(Files.readString(saved), legacy)
    assert(!Files.exists(directory.resolve("Answering.scala")))
    val second = scratch()
    Files.createSymbolicLink(
      second.resolve("Answering.scala"),
      second.resolve("absent.txt").toAbsolutePath
    )
    intercept[IllegalArgumentException](split(second))
    assertEquals(Files.readString(second.resolve("AnswerDispatch.scala")), legacy)
    assert(Files.isSymbolicLink(second.resolve("Answering.scala")))

  test("unaccounted global declarations cannot be erased by accepted-layout restoration"):
    val before = legacy.replace("import framework.*", "import framework.*\nval unaccounted = 41")
    val fixed = expected.replace("import framework.*", "import framework.*\nval unaccounted = 41")
    val directory = scratch()
    Files.writeString(directory.resolve("AnswerDispatch.scala"), before)
    intercept[IllegalArgumentException](split(directory, fixed, before = before))
    assertEquals(Files.readString(directory.resolve("AnswerDispatch.scala")), before)
    assert(!Files.exists(directory.resolve("Answering.scala")))

  test("seed imports add only the missing selected name, preserving the machine's Outcome binding"):
    val baseImport = "import framework.*\nimport framework.outcomes.Outcome"
    val pinnedMachine = acceptedMachine.replace("import framework.*", baseImport)
    val pinnedFacet = acceptedFacet.replace(
      "import framework.*",
      "import framework.*\nimport framework.outcomes.{Outcome, Rejection}"
    )
    def merged(source: String) = source
      .replace("import framework.*", baseImport)
      .replace(
        "\nobject AnswerDispatch",
        "\nimport framework.outcomes.Rejection\nobject AnswerDispatch"
      )
    val before = merged(legacy)
    val fixed = merged(expected)
    val directory = scratch()
    Files.writeString(directory.resolve("AnswerDispatch.scala"), before)
    split(directory, fixed, before = before, pinMachine = pinnedMachine, pinFacet = pinnedFacet)
    assertEquals(Files.readString(directory.resolve("AnswerDispatch.scala")), pinnedMachine)
    assertEquals(Files.readString(directory.resolve("Answering.scala")), pinnedFacet)
