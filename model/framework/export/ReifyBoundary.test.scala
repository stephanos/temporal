package framework.`export`

import java.nio.file.{Files, Path}
import java.nio.file.attribute.PosixFilePermissions
import scala.concurrent.duration.*
import framework.check.{RetainedScratch, Tools}

class ReifyBoundarySuite extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private lazy val scratch = retained(tools.directory.resolve(".flow/tmp"), "reify-boundary.")

  private def source(directory: Path, name: String, text: String): Unit =
    Files.createDirectories(directory): Unit
    Files.writeString(directory.resolve(name), text): Unit

  private lazy val libraryJar =
    val libraryJar = scratch.resolve("capture.jar")
    tools.scalaCli(Seq("--power", "package", "--server=false", "--library",
      "model/framework/dsl", "model/framework/realize", "-f", "-o", libraryJar.toString)).orFail(): Unit
    libraryJar

  private lazy val plugin =
    val plugin = scratch.resolve("reify.jar")
    tools.scalaCli(Seq("--power", "package", "--server=false", "--assembly", "--preamble=false",
      "--main-class-in-manifest=false", "--provided", "org.scala-lang:scala3-compiler_3",
      "model/framework/reify/plugin", "-f", "-o", plugin.toString)).orFail(): Unit
    plugin

  private def pluginOption = Seq("--scalac-option", s"-Xplugin:$plugin")
  private def settings = Seq("--scala", "3.9.0", "--jvm", "27", "--server=false",
    "--dependency", "com.thesamet.scalapb::scalapb-runtime:0.11.20",
    "--jar", libraryJar.toString, "--jar", tools.directory.resolve("model/build/ir-scalapb.jar").toString)

  test("a jar helper keeps its real body, transitive callee and native callable"):
    val helper = scratch.resolve("helper")
    source(helper, "Helper.scala", """package framework.boundary
import framework.dsl.lifted
object Helper:
  @lifted def single(s: Boolean): Boolean = !s
  @lifted def viaSingle(s: Boolean): Boolean = single(s)
""")
    val helperJar = scratch.resolve("helper.jar")
    tools.scalaCli(Seq("--power", "package", "--library", helper.toString, "-f", "-o",
      helperJar.toString) ++ settings ++ pluginOption).orFail(): Unit
    val caller = scratch.resolve("caller")
    source(caller, "Caller.scala", """package framework.boundary
import framework.dsl.Reify
object Caller:
  def main(args: Array[String]): Unit =
    val captured = Reify.capture(Helper.viaSingle)
    val f = captured.function
    assert(f.name == "framework.boundary.Helper$.viaSingle", s"missing real helper IR: $f")
    assert(f.params.map(_.name) == Seq("s"), s"wrong parameters: $f")
    assert(f.getBody.getCall.function == "framework.boundary.Helper$.single", s"missing transitive call: $f")
    assert(captured.model.functions.map(_.name).toSet == Set("framework.boundary.Helper$.viaSingle", "framework.boundary.Helper$.single"))
    assert(captured.model.functions.size == 2)
    val single = captured.model.functions.find(_.name == "framework.boundary.Helper$.single").get
    assert(single.getBody.getUnary.op.name == "OP_NOT")
    assert(single.getBody.getUnary.getOperand.getVar == "s")
    assert(captured.model.types.isEmpty)
    assert(captured.callable(false))
    assert(Reify.function(Helper.viaSingle).name == f.name)
    assert(Helper.viaSingle(false))
    println("real-helper-body-and-native-callable")
""")
    val ran = tools.scalaCli(Seq("run", caller.toString, "--main-class", "framework.boundary.Caller",
      "--jar", helperJar.toString) ++ settings ++ pluginOption)
    assertEquals(ran.exit, 0, ran.diagnostics)
    assert(ran.output.contains("real-helper-body-and-native-callable"), ran.diagnostics)

  test("a jar sugar body binds its generic parameters in a real capture"):
    val kit = scratch.resolve("sugar-kit")
    source(kit, "Kit.scala", """package framework.boundary
import framework.dsl.sugar
object Kit:
  @sugar def selected[A](value: A, wanted: A): Boolean = value == wanted
""")
    val kitJar = scratch.resolve("kit.jar")
    tools.scalaCli(Seq("--power", "package", "--library", kit.toString, "-f", "-o", kitJar.toString)
      ++ settings).orFail(): Unit
    val caller = scratch.resolve("sugar-caller")
    source(caller, "Caller.scala", """package framework.boundary
import framework.dsl.Reify
object SugarCaller:
  def main(args: Array[String]): Unit =
    val f = Reify.capture((s: Boolean) => Kit.selected(s, false))
    assert(f.callable(false))
    assert(!f.callable(true))
    assert(f.model.functions.size == 1)
    assert(f.function.getBody.getBinary.op.name == "OP_EQ")
    assert(f.function.getBody.getBinary.getLeft.getVar == "s")
    assert(!f.function.getBody.getBinary.getRight.getLiteral.getBool)
""")
    val ran = tools.scalaCli(Seq("run", caller.toString, "--main-class", "framework.boundary.SugarCaller",
      "--jar", kitJar.toString) ++ settings ++ pluginOption)
    assertEquals(ran.exit, 0, ran.diagnostics)

  test("definition-site storage precedes a caller in another source unit"):
    val both = scratch.resolve("same-run")
    source(both, "ACaller.scala", """package framework.boundary
import framework.dsl.Reify
object SameRunCaller:
  def main(args: Array[String]): Unit =
    val captured = Reify.capture(LastHelper.forward)
    assert(captured.callable(false))
    assert(captured.model.functions.map(_.name).toSet == Set("framework.boundary.LastHelper$.forward", "framework.boundary.LastHelper$.invert"))
    assert(captured.model.functions.size == 2)
    val invert = captured.model.functions.find(_.name.endsWith(".invert")).get
    assert(invert.getBody.getUnary.op.name == "OP_NOT")
""")
    source(both, "ZHelper.scala", """package framework.boundary
import framework.dsl.lifted
object LastHelper:
  @lifted def invert(s: Boolean): Boolean = !s
  @lifted def forward(s: Boolean): Boolean = invert(s)
""")
    val ran = tools.scalaCli(Seq("run", both.toString, "--main-class", "framework.boundary.SameRunCaller")
      ++ settings ++ pluginOption)
    assertEquals(ran.exit, 0, ran.diagnostics)

  test("an unmarked foreign helper is refused even beside an eligible jar sugar"):
    val helper = scratch.resolve("unmarked-helper")
    source(helper, "Helper.scala", """package framework.boundary
import framework.dsl.sugar
object Unmarked:
  @sugar def eligible(s: Boolean): Boolean = !s
  def ordinary(s: Boolean): Boolean = !s
""")
    val jar = scratch.resolve("unmarked.jar")
    tools.scalaCli(Seq("--power", "package", "--library", helper.toString, "-f", "-o", jar.toString)
      ++ settings).orFail(): Unit
    val caller = scratch.resolve("unmarked-caller")
    source(caller, "Caller.scala", """package framework.boundary
import framework.dsl.Reify
object UnmarkedCaller:
  val captured = Reify.function(Unmarked.ordinary)
""")
    for (name, inputs, jars) <- Seq(
        ("jar", Seq(caller.toString), Seq("--jar", jar.toString)),
        ("same run", Seq(caller.toString, helper.toString), Seq.empty)
      )
    do
      val ran = tools.scalaCli(Seq("compile") ++ inputs ++ jars ++ settings ++ pluginOption)
      assert(ran.failed, s"$name: ${ran.diagnostics}")
      assert(ran.diagnostics.contains("framework.boundary.Unmarked$.ordinary"), ran.diagnostics)
      assert(ran.diagnostics.contains("mark its definition @lifted"), ran.diagnostics)
      assert(ran.diagnostics.contains("Caller.scala:4:"), ran.diagnostics)
      assert(!ran.diagnostics.contains("unhandled exception"), ran.diagnostics)

  test("unsupported bodies refuse at their tree in captures and lifted definitions"):
    val bodies = Seq(
      "definition" -> """package framework.boundary
import framework.dsl.lifted
object Broken:
  @lifted def broken(s: Boolean): Boolean =
    var count = 0
    count = 1
    s
""",
      "capture" -> """package framework.boundary
import framework.dsl.Reify
object Broken:
  val broken = Reify.function((s: Boolean) =>
    var count = 0
    count = 1
    s)
"""
    )
    for (name, text) <- bodies do
      val dir = scratch.resolve(s"unsupported-$name")
      source(dir, "Broken.scala", text)
      val ran = tools.scalaCli(Seq("compile", dir.toString) ++ settings ++ pluginOption)
      assert(ran.failed, ran.diagnostics)
      assert(ran.diagnostics.contains("`var count` has no IR form"), ran.diagnostics)
      assert(ran.diagnostics.contains("Broken.scala:5:"), ran.diagnostics)
      assert(!ran.diagnostics.contains("unhandled exception"), ran.diagnostics)

  test("typeCheckErrors does not replace compiling a plugin refusal fixture"):
    val dir = scratch.resolve("type-check-errors")
    source(dir, "Probe.scala", """package framework.boundary
import scala.compiletime.testing.typeCheckErrors
object TypeChecking:
  def main(args: Array[String]): Unit =
    val errors = typeCheckErrors("framework.dsl.Reify.function((s: Boolean) => { var n = 0; n = 1; s })")
    assert(errors.isEmpty, errors.toString)
    println("typeCheckErrors-does-not-run-plugin-phases")
""")
    val ran = tools.scalaCli(Seq("run", dir.toString, "--main-class", "framework.boundary.TypeChecking")
      ++ settings ++ pluginOption)
    assertEquals(ran.exit, 0, ran.diagnostics)
    assert(ran.output.contains("typeCheckErrors-does-not-run-plugin-phases"), ran.diagnostics)

  test("a chunked jar carrier survives the classfile string limit"):
    def balanced(depth: Int): String =
      if depth == 0 then "s" else s"(${balanced(depth - 1)} && ${balanced(depth - 1)})"
    val helper = scratch.resolve("large-helper")
    source(helper, "Large.scala", s"""package framework.boundary
import framework.dsl.lifted
object Large:
  @lifted def all(s: Boolean): Boolean = ${balanced(10)}
""")
    val jar = scratch.resolve("large.jar")
    tools.scalaCli(Seq("--power", "package", "--library", helper.toString, "-f", "-o", jar.toString)
      ++ settings ++ pluginOption).orFail(): Unit
    val caller = scratch.resolve("large-caller")
    source(caller, "Caller.scala", """package framework.boundary
import framework.dsl.Reify
object LargeCaller:
  def main(args: Array[String]): Unit =
    val captured = Reify.capture(Large.all)
    assert(captured.model.toByteArray.length > 65535)
    assert(captured.model.functions.size == 1)
    assert(captured.function.name == "framework.boundary.Large$.all")
    assert(captured.callable(true))
    assert(!captured.callable(false))
""")
    val ran = tools.scalaCli(Seq("run", caller.toString, "--main-class", "framework.boundary.LargeCaller",
      "--jar", jar.toString) ++ settings ++ pluginOption)
    assertEquals(ran.exit, 0, ran.diagnostics)

  test("capture dispatch follows the annotated parameter and preserves native evaluation"):
    val dir = scratch.resolve("factory")
    source(dir, "Factory.scala", """package framework.boundary
import framework.dsl.{capture, Captured, Reify}
object Factory:
  def authored[F](label: String)(@capture("linked") f: F): Captured[F] =
    throw IllegalStateException("capture phase did not run")
  def linked[F](label: String)(f: F, root: String, chunks: Seq[String]): Captured[F] =
    assert(label == "native")
    Reify.captured(f, root, chunks)
  def main(args: Array[String]): Unit =
    var evaluations = 0
    def label(): String =
      evaluations += 1
      "native"
    val captured: Captured[Boolean => Boolean] = authored(label())((s: Boolean) => !s)
    assert(evaluations == 1)
    assert(captured.callable(false))
    assert(captured.function.getBody.getUnary.op.name == "OP_NOT")
""")
    val ran = tools.scalaCli(Seq("run", dir.toString, "--main-class", "framework.boundary.Factory")
      ++ settings ++ pluginOption)
    assertEquals(ran.exit, 0, ran.diagnostics)

  test("a helper-only edit changes dependent captured IR on one warm Bloop daemon"):
    val dir = scratch.resolve("warm")
    val daemon = retained(Path.of("/tmp"), "reify-warm.")
    Files.createDirectories(daemon): Unit
    Files.setPosixFilePermissions(daemon, PosixFilePermissions.fromString("rwx------")): Unit
    val warmSettings = settings.filterNot(_ == "--server=false") ++
      Seq("--bloop-daemon-dir", daemon.toString, "--verbose", "--scalac-option", "-Ycheck:typer") ++ pluginOption
    source(dir, "Caller.scala", """package framework.boundary
import framework.dsl.Reify
object WarmCaller:
  def main(args: Array[String]): Unit =
    val captured = Reify.capture(WarmHelper.result)
    println("captured-result=" + captured.function.getBody.getLiteral.getBool)
    assert(captured.function.getBody.getLiteral.getBool == captured.callable(false))
""")
    def helper(body: String): Unit = source(dir, "Helper.scala", s"""package framework.boundary
import framework.dsl.lifted
object WarmHelper:
  @lifted def result(s: Boolean): Boolean = $body
""")
    try
      helper("false")
      val first = tools.scalaCli(Seq("run", dir.toString, "--main-class", "framework.boundary.WarmCaller")
        ++ warmSettings).orFail()
      assert(first.output.contains("captured-result=false"), first.diagnostics)
      helper("true")
      val second = tools.scalaCli(Seq("run", dir.toString, "--main-class", "framework.boundary.WarmCaller")
        ++ warmSettings).orFail()
      assert(second.output.contains("captured-result=true"), second.diagnostics)
      for name <- Seq("Helper.scala", "Caller.scala") do
        assert(second.diagnostics.contains(s"$name after phase typer"), second.diagnostics)
      helper("{ var n = 0; n = 1; s }")
      val refused = tools.scalaCli(Seq("compile", dir.toString) ++ warmSettings)
      assert(refused.failed, refused.diagnostics)
      assert(refused.diagnostics.contains("`var n` has no IR form"), refused.diagnostics)
    finally
      tools.run("scala-cli", Seq("--power", "bloop", "exit", "--bloop-daemon-dir", daemon.toString))
        .orFail(): Unit

  test("jar imports deduplicate the complete nested-record type closure"):
    val helper = scratch.resolve("records-helper")
    source(helper, "Records.scala", """package framework.boundary
import framework.dsl.lifted
case class Leaf(flag: Boolean)
case class Record(leaf: Leaf, optional: Option[Leaf])
object Records:
  @lifted def flag(r: Record): Boolean = r.leaf.flag
  @lifted def again(r: Record): Boolean = flag(r)
""")
    val jar = scratch.resolve("records.jar")
    tools.scalaCli(Seq("--power", "package", "--library", helper.toString, "-f", "-o", jar.toString)
      ++ settings ++ pluginOption).orFail(): Unit
    val caller = scratch.resolve("records-caller")
    source(caller, "Caller.scala", """package framework.boundary
import framework.dsl.Reify
object RecordsCaller:
  def main(args: Array[String]): Unit =
    val captured = Reify.capture((r: Record) => Records.flag(r) && Records.again(r))
    assert(captured.callable(Record(Leaf(true), None)))
    assert(!captured.callable(Record(Leaf(false), Some(Leaf(true)))))
    assert(captured.model.functions.size == 3)
    assert(captured.model.functions.map(_.name).distinct.size == 3)
    assert(captured.model.types.map(_.name).distinct.size == captured.model.types.size)
    assert(captured.model.types.map(_.name).toSet == Set("framework.boundary.Leaf", "framework.boundary.Record", "scala.Option[framework.boundary.Leaf]"))
    val flag = captured.model.functions.find(_.name == "framework.boundary.Records$.flag").get
    assert(flag.getBody.getField.field == "flag")
    assert(flag.getBody.getField.getBase.getField.field == "leaf")
""")
    val ran = tools.scalaCli(Seq("run", caller.toString, "--main-class", "framework.boundary.RecordsCaller",
      "--jar", jar.toString) ++ settings ++ pluginOption)
    assertEquals(ran.exit, 0, ran.diagnostics)
