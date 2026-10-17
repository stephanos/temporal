package framework.`export`

import java.nio.file.Files
import scala.concurrent.duration.*
import framework.check.{RetainedScratch, Tools}

class InlineSugarSuite extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private lazy val scratch =
    retained(tools.directory.resolve(".flow/tmp"), "inline-sugar.")
  private lazy val plugin = ModelBuild.reifyPlugin:
    val jar = scratch.resolve("reify.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--assembly",
          "--preamble=false",
          "--main-class-in-manifest=false",
          "--provided",
          "org.scala-lang:scala3-compiler_3",
          "model/framework/reify/plugin",
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    jar

  private def compile(name: String, body: String, run: Boolean = false) =
    val directory = Files.createDirectory(scratch.resolve(name))
    Files.writeString(directory.resolve("Sugar.scala"), body): Unit
    tools.scalaCli(
      Seq(
        if run then "run" else "compile",
        "--server=false",
        "--scala",
        "3.9.0",
        "--jvm",
        "27",
        "--classpath",
        ModelBuild.classpathEntries,
        "--scalac-option",
        s"-Xplugin:$plugin",
        directory.toString
      )
    )

  test(
    "inline helpers lift their core body and retain call-site role evidence"
  ):
    val result = compile(
      "core",
      """package framework.inlinesugar
import framework.dsl.*
enum Phase:
  case waiting
  case done extends Phase, Succeeded
trait RoleFacet[A]:
  inline def closed(value: A)(using scala.reflect.TypeTest[A, Closed]): Boolean = value.in[Closed]
object Check extends RoleFacet[Phase]:
  inline def invert(value: Boolean): Boolean = !value
  def main(args: Array[String]): Unit =
    val inverted = Reify.function((value: Boolean) => invert(value))
    assert(inverted.getBody.getUnary.op.name == "OP_NOT")
    val membership = Reify.function((phase: Phase) => closed(phase))
    assert(membership.getBody.getBinary.op.name == "OP_CONTAINS")
""",
      run = true
    )
    assertEquals(result.exit, 0, result.diagnostics)

  test("compiler inline bindings keep strict argument order and by-name laziness"):
    val result = compile(
      "arguments",
      """package framework.inlinesugar
import framework.dsl.*
object Check:
  def first(value: Boolean): Boolean = !value
  def second(value: Boolean): Boolean = value
  inline def swapped(a: Boolean, b: Boolean): Boolean = b && a && a
  inline def ignored(a: Boolean): Boolean = true
  inline def delayed(a: => Boolean): Boolean = false && a
  inline def repeated(a: => Boolean): Boolean = a && a
  def main(args: Array[String]): Unit =
    val strict = Reify.function((value: Boolean) => swapped(first(value), second(value)))
    val outer = strict.getBody.getLet
    assert(outer.getValue.getCall.function.endsWith(".first"))
    assert(outer.getBody.getLet.getValue.getCall.function.endsWith(".second"))
    val unused = Reify.function((value: Boolean) => ignored(first(value)))
    assert(unused.getBody.getLet.getValue.getCall.function.endsWith(".first"))
    val lazyResult = Reify.capture((value: Boolean) => delayed(first(value)))
    assert(!lazyResult.callable(false))
    val lazyBody = lazyResult.function.getBody.getBinary
    assert(lazyBody.op.name == "OP_AND")
    assert(!lazyBody.getLeft.getLiteral.getBool)
    assert(lazyBody.getRight.getCall.function.endsWith(".first"))
    val repeatedResult = Reify.function((value: Boolean) => repeated(first(value)))
    assert(repeatedResult.getBody.kind.isBinary)
    val repeatedBody = repeatedResult.getBody.getBinary
    assert(repeatedBody.op.name == "OP_AND")
    assert(repeatedBody.getLeft.getCall.function.endsWith(".first"))
    assert(repeatedBody.getRight.getCall.function.endsWith(".first"))
""",
      run = true
    )
    assertEquals(result.exit, 0, result.diagnostics)

  test("inline DSL sugar retains its core steps and predicates"):
    val result = compile(
      "primitives",
      """package framework.inlinesugar
import framework.dsl.*
object Primitives:
  given Ok[Boolean] = Ok(true)
  def main(args: Array[String]): Unit =
    val entering = Reify.function((state: Boolean) => enter[Boolean, Boolean, String](state, "first", "last"))
    val facts = entering.getBody.getLet
    assert(facts.getValue.getList.items.map(_.getLiteral.getText) == Seq("first", "last"))
    val arguments = facts.getBody.getList.items.head.getConstruct.args
    assert(arguments.head.getLiteral.getBool)
    assert(arguments(2).getVar == facts.name)
    val staying = Reify.function((state: Boolean) => stay[Boolean, Boolean, String](state))
    assert(staying.getBody.getList.items.size == 1)
    val rejecting = Reify.function((state: Boolean) => reject(false, state))
    assert(!rejecting.getBody.getList.items.head.getConstruct.args.head.getLiteral.getBool)
    val empty = Reify.function((state: Boolean) => disabled)
    assert(empty.getBody.getList.items.isEmpty)
    val member = Reify.function((state: Boolean) => state.in(true, false))
    val tail = member.getBody.getLet
    assert(tail.getValue.getList.items.map(_.getLiteral.getBool) == Seq(false))
    assert(tail.getBody.getBinary.op.name == "OP_CONTAINS")
    val implication = Reify.function((state: Boolean) => state implies false)
    assert(implication.getBody.getBinary.op.name == "OP_OR")
    val recorded = Reify.function((step: Step[Boolean, Boolean, String]) => step.records("first"))
    assert(recorded.getBody.getBinary.op.name == "OP_CONTAINS")
""",
      run = true
    )
    assertEquals(result.exit, 0, result.diagnostics)

  test("a public step fact read cannot widen the receiver's fact type"):
    val result = compile(
      "foreign-fact",
      """package publicclient
import framework.dsl.*
object Foreign:
  val step: Step[Boolean, Boolean, String] = Step(true, false, List("own"))
  val own: Boolean = step.records("own")
  val foreign: Boolean = step.records(true)
"""
    )
    assert(result.failed, result.diagnostics)
    assert(result.diagnostics.contains("Cannot prove that Boolean <:< String"), result.diagnostics)

  test("an unsupported inline body names its sugar at the call site"):
    val result = compile(
      "refused",
      """package framework.inlinesugar
import framework.dsl.Reify
object Broken:
  inline def selected(value: Boolean): Boolean = System.nanoTime() > 0 && value
  val captured = Reify.function((value: Boolean) => selected(value))
"""
    )
    assert(result.failed, result.diagnostics)
    assert(
      result.diagnostics.contains(
        "sugar framework.inlinesugar.Broken$.selected"
      ),
      result.diagnostics
    )
    assert(result.diagnostics.contains("Sugar.scala:5:"), result.diagnostics)
    assert(
      !result.diagnostics.contains("unhandled exception"),
      result.diagnostics
    )
