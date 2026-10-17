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

  private lazy val libraryJar = ModelBuild.frameworkLibrary:
    val libraryJar = scratch.resolve("capture.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          "model/framework/dsl",
          "model/framework/realize",
          "-f",
          "-o",
          libraryJar.toString
        )
      )
      .orFail(): Unit
    libraryJar

  private lazy val plugin = ModelBuild.reifyPlugin:
    val plugin = scratch.resolve("reify.jar")
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
          plugin.toString
        )
      )
      .orFail(): Unit
    plugin

  private def pluginOption = Seq("--scalac-option", s"-Xplugin:$plugin")
  private def settings = Seq(
    "--scala",
    "3.9.0",
    "--jvm",
    "27",
    "--server=false",
    "--dependency",
    "com.thesamet.scalapb::scalapb-runtime:0.11.20",
    "--jar",
    libraryJar.toString,
    "--jar",
    tools.directory.resolve("model/build/ir-scalapb.jar").toString
  )

  test("definition-site storage precedes a caller in another source unit"):
    val both = scratch.resolve("same-run")
    source(
      both,
      "ACaller.scala",
      """package framework.boundary
import framework.dsl.Reify
object SameRunCaller:
  def main(args: Array[String]): Unit =
    val captured = Reify.capture(LastHelper.forward)
    assert(captured.callable(false))
    assert(captured.model.functions.map(_.name).toSet == Set("framework.boundary.LastHelper$.forward", "framework.boundary.LastHelper$.invert"))
    assert(captured.model.functions.size == 2)
    val invert = captured.model.functions.find(_.name.endsWith(".invert")).get
    assert(invert.getBody.getUnary.op.name == "OP_NOT")
"""
    )
    source(
      both,
      "ZHelper.scala",
      """package framework.boundary
import framework.dsl.lifted
object LastHelper:
  @lifted def invert(s: Boolean): Boolean = !s
  @lifted def forward(s: Boolean): Boolean = invert(s)
"""
    )
    val ran = tools.scalaCli(
      Seq("run", both.toString, "--main-class", "framework.boundary.SameRunCaller")
        ++ settings ++ pluginOption
    )
    assertEquals(ran.exit, 0, ran.diagnostics)

  test("an unmarked helper or sugar in another source unit is refused"):
    val helper = scratch.resolve("unmarked-helper")
    source(
      helper,
      "Helper.scala",
      """package framework.boundary
import framework.dsl.sugar
object Unmarked:
  @sugar def eligible(s: Boolean): Boolean = !s
  def ordinary(s: Boolean): Boolean = !s
"""
    )
    val caller = scratch.resolve("unmarked-caller")
    for (name, argument, reached) <- Seq(
        ("helper", "Unmarked.ordinary", "ordinary"),
        ("sugar", "(s: Boolean) => Unmarked.eligible(s)", "eligible")
      )
    do
      source(
        caller,
        "Caller.scala",
        s"""package framework.boundary
import framework.dsl.Reify
object UnmarkedCaller:
  val captured = Reify.function($argument)
"""
      )
      val ran = tools.scalaCli(
        Seq("compile", caller.toString, helper.toString) ++ settings ++ pluginOption
      )
      assert(ran.failed, s"$name: ${ran.diagnostics}")
      assert(ran.diagnostics.contains(s"framework.boundary.Unmarked$$.$reached"), ran.diagnostics)
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
    source(
      dir,
      "Probe.scala",
      """package framework.boundary
import scala.compiletime.testing.typeCheckErrors
object TypeChecking:
  def main(args: Array[String]): Unit =
    val errors = typeCheckErrors("framework.dsl.Reify.function((s: Boolean) => { var n = 0; n = 1; s })")
    assert(errors.isEmpty, errors.toString)
    println("typeCheckErrors-does-not-run-plugin-phases")
"""
    )
    val ran = tools.scalaCli(
      Seq("run", dir.toString, "--main-class", "framework.boundary.TypeChecking")
        ++ settings ++ pluginOption
    )
    assertEquals(ran.exit, 0, ran.diagnostics)
    assert(ran.output.contains("typeCheckErrors-does-not-run-plugin-phases"), ran.diagnostics)

  test("a chunked carrier survives the classfile string limit"):
    def balanced(depth: Int): String =
      if depth == 0 then "s" else s"(${balanced(depth - 1)} && ${balanced(depth - 1)})"
    val helper = scratch.resolve("large-helper")
    source(
      helper,
      "Large.scala",
      s"""package framework.boundary
import framework.dsl.lifted
object Large:
  @lifted def all(s: Boolean): Boolean = ${balanced(10)}
"""
    )
    val caller = scratch.resolve("large-caller")
    source(
      caller,
      "Caller.scala",
      """package framework.boundary
import framework.dsl.Reify
object LargeCaller:
  def main(args: Array[String]): Unit =
    val captured = Reify.capture(Large.all)
    assert(captured.model.toByteArray.length > 65535)
    assert(captured.model.functions.size == 1)
    assert(captured.function.name == "framework.boundary.Large$.all")
    assert(captured.callable(true))
    assert(!captured.callable(false))
"""
    )
    val ran = tools.scalaCli(
      Seq(
        "run",
        caller.toString,
        helper.toString,
        "--main-class",
        "framework.boundary.LargeCaller"
      ) ++ settings ++ pluginOption
    )
    assertEquals(ran.exit, 0, ran.diagnostics)

  test("capture dispatch follows the annotated parameter and preserves native evaluation"):
    val dir = scratch.resolve("factory")
    source(
      dir,
      "Factory.scala",
      """package framework.boundary
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
"""
    )
    val ran = tools.scalaCli(
      Seq("run", dir.toString, "--main-class", "framework.boundary.Factory")
        ++ settings ++ pluginOption
    )
    assertEquals(ran.exit, 0, ran.diagnostics)

  test("a helper-only edit changes dependent captured IR on one warm Bloop daemon"):
    val dir = scratch.resolve("warm")
    val daemon = retained(Path.of("/tmp"), "reify-warm.")
    Files.createDirectories(daemon): Unit
    Files.setPosixFilePermissions(daemon, PosixFilePermissions.fromString("rwx------")): Unit
    val warmSettings = settings.filterNot(_ == "--server=false") ++
      Seq(
        "--bloop-daemon-dir",
        daemon.toString,
        "--verbose",
        "--scalac-option",
        "-Ycheck:typer"
      ) ++ pluginOption
    source(
      dir,
      "Caller.scala",
      """package framework.boundary
import framework.dsl.Reify
object WarmCaller:
  def main(args: Array[String]): Unit =
    val captured = Reify.capture(WarmHelper.result)
    println("captured-result=" + captured.function.getBody.getLiteral.getBool)
    assert(captured.function.getBody.getLiteral.getBool == captured.callable(false))
"""
    )
    def helper(body: String): Unit = source(
      dir,
      "Helper.scala",
      s"""package framework.boundary
import framework.dsl.lifted
object WarmHelper:
  @lifted def result(s: Boolean): Boolean = $body
"""
    )
    try
      helper("false")
      val first = tools
        .scalaCli(
          Seq("run", dir.toString, "--main-class", "framework.boundary.WarmCaller")
            ++ warmSettings
        )
        .orFail()
      assert(first.output.contains("captured-result=false"), first.diagnostics)
      helper("true")
      val second = tools
        .scalaCli(
          Seq("run", dir.toString, "--main-class", "framework.boundary.WarmCaller")
            ++ warmSettings
        )
        .orFail()
      assert(second.output.contains("captured-result=true"), second.diagnostics)
      for name <- Seq("Helper.scala", "Caller.scala") do
        assert(second.diagnostics.contains(s"$name after phase typer"), second.diagnostics)
      helper("{ var n = 0; n = 1; s }")
      val refused = tools.scalaCli(Seq("compile", dir.toString) ++ warmSettings)
      assert(refused.failed, refused.diagnostics)
      assert(refused.diagnostics.contains("`var n` has no IR form"), refused.diagnostics)
      helper("false")
      val recovered = tools
        .scalaCli(
          Seq("run", dir.toString, "--main-class", "framework.boundary.WarmCaller")
            ++ warmSettings
        )
        .orFail()
      assert(recovered.output.contains("captured-result=false"), recovered.diagnostics)
    finally
      tools
        .run("scala-cli", Seq("--power", "bloop", "exit", "--bloop-daemon-dir", daemon.toString))
        .orFail(): Unit

  test("imports from another source unit deduplicate the complete nested-record type closure"):
    val helper = scratch.resolve("records-helper")
    source(
      helper,
      "Records.scala",
      """package framework.boundary
import framework.dsl.lifted
case class Leaf(flag: Boolean)
case class Record(leaf: Leaf, optional: Option[Leaf])
object Records:
  @lifted def flag(r: Record): Boolean = r.leaf.flag
  @lifted def again(r: Record): Boolean = flag(r)
"""
    )
    val caller = scratch.resolve("records-caller")
    source(
      caller,
      "Caller.scala",
      """package framework.boundary
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
"""
    )
    val ran = tools.scalaCli(
      Seq(
        "run",
        caller.toString,
        helper.toString,
        "--main-class",
        "framework.boundary.RecordsCaller"
      ) ++ settings ++ pluginOption
    )
    assertEquals(ran.exit, 0, ran.diagnostics)

  test("the authored action binding captures typed IR without replacing its native function"):
    val dir = scratch.resolve("binding")
    source(
      dir,
      "Binding.scala",
      """package framework.boundary
import framework.dsl.*
enum Said:
  case ok
object BindingCaller:
  val press = action(Actor("user"))
  def main(args: Array[String]): Unit =
    val f: Boolean => List[Step[Boolean, Said, Nothing]] = s => List(Step(Said.ok, !s, Nil, ""))
    val binding: StepBinding[Boolean, Said, Nothing] = press ~> f
    val captured = binding.captured.get
    assert(binding.function.eq(f))
    assert(captured.callable.eq(f))
    assert(binding.function.asInstanceOf[Boolean => List[Step[Boolean, Said, Nothing]]](false).head.state)
    assert(captured.function.getBody.getList.items.head.getConstruct.args(1).getUnary.op.name == "OP_NOT")
    assert(binding.productArity == 2)
    assert(binding == StepBinding[Boolean, Said, Nothing](press.decl, f))
    val StepBinding(decl, native) = binding
    assert(decl == press.decl)
    assert(native.eq(f))
"""
    )
    val ran = tools.scalaCli(
      Seq("run", dir.toString, "--main-class", "framework.boundary.BindingCaller")
        ++ settings ++ pluginOption
    )
    assertEquals(ran.exit, 0, ran.diagnostics)

  test("stored choice copies keep their transitive bodies and refuse a multi-step helper"):
    val helper = scratch.resolve("choice-helper")
    source(
      helper,
      "Choices.scala",
      """package framework.boundary
import framework.dsl.*
enum ChoiceOutcome:
  case ok
object Choices:
  val first = choice
  val second = choice
  @lifted def single(s: Boolean): List[Step[Boolean, ChoiceOutcome, Nothing]] =
    List(Step(ChoiceOutcome.ok, !s, Nil, ""))
  @lifted def forward(s: Boolean): List[Step[Boolean, ChoiceOutcome, Nothing]] = single(s)
  @lifted def multiple(s: Boolean): List[Step[Boolean, ChoiceOutcome, Nothing]] =
    choose(first -> List(Step(ChoiceOutcome.ok, s, Nil, "")), second -> List(Step(ChoiceOutcome.ok, !s, Nil, "")))
"""
    )
    val caller = scratch.resolve("choice-caller")
    source(
      caller,
      "Caller.scala",
      """package framework.boundary
import framework.dsl.*
object ChoiceCaller:
  val outer = choice
  val other = choice
  def main(args: Array[String]): Unit =
    val captured = Reify.capture((s: Boolean) => choose(outer -> Choices.forward(s), other -> List(Step(ChoiceOutcome.ok, s, Nil, ""))))
    assert(captured.callable(false).map(_.state) == List(true, false))
    val original = "framework.boundary.Choices$.single"
    val forward = "framework.boundary.Choices$.forward"
    assert(captured.model.functions.map(_.name).toSet == Set(captured.root, original, forward, original + "$outer", forward + "$outer"))
    def function(name: String) = captured.model.functions.find(_.name == name).get
    assert(function(original).getBody.getList.items.head.getConstruct.choice.isEmpty)
    assert(function(original + "$outer").getBody.getList.items.head.getConstruct.choice == "outer")
    assert(function(forward + "$outer").getBody.getCall.function == original + "$outer")
"""
    )
    val ran = tools.scalaCli(
      Seq(
        "run",
        caller.toString,
        helper.toString,
        "--main-class",
        "framework.boundary.ChoiceCaller"
      ) ++ settings ++ pluginOption
    )
    assertEquals(ran.exit, 0, ran.diagnostics)
    source(
      caller,
      "Caller.scala",
      """package framework.boundary
import framework.dsl.*
object RefusedChoice:
  val outer = choice
  val other = choice
  val captured = Reify.capture((s: Boolean) => choose(outer -> Choices.multiple(s), other -> Choices.single(s)))
"""
    )
    val refused = tools.scalaCli(
      Seq("compile", caller.toString, helper.toString) ++ settings ++ pluginOption
    )
    assert(refused.failed, refused.diagnostics)
    assert(
      refused.diagnostics.contains(
        "the alternative outer of a choose calls multiple, which gives 2 steps"
      ),
      refused.diagnostics
    )
    assert(refused.diagnostics.contains("Caller.scala:6:"), refused.diagnostics)
    assert(!refused.diagnostics.contains("unhandled exception"), refused.diagnostics)

  test("nested records syntax reaches capture before selector inlining and keeps its native key"):
    val dir = scratch.resolve("nested-records")
    source(
      dir,
      "Nested.scala",
      """package framework.boundary
import framework.dsl.*
case class Member(flag: Boolean)
case class Pair(left: Member, right: Member)
enum RecordedFact:
  case noted
object NestedRecords:
  def main(args: Array[String]): Unit =
    val captured = Reify.capture((after: Step[Pair, Boolean, String]) => after.records(_.left, RecordedFact.noted))
    val state = Pair(Member(true), Member(false))
    assert(captured.callable(Step(true, state, List("left_noted"), "")))
    assert(!captured.callable(Step(true, state, List("right_noted"), "")))
    val body = captured.function.getBody
    assert(body.getBinary.op.name == "OP_CONTAINS")
    assert(body.getBinary.getLeft.getLiteral.getText == "left_noted")
    assert(body.getBinary.getRight.getField.field == "facts")
    assert(body.getBinary.getRight.getField.getBase.getVar == "after")
"""
    )
    val ran = tools.scalaCli(
      Seq("run", dir.toString, "--main-class", "framework.boundary.NestedRecords")
        ++ settings ++ pluginOption
    )
    assertEquals(ran.exit, 0, ran.diagnostics)

  // Multi-input chains are retired: a native callback takes the state and one declared input
  // (model/framework/README.md, "Inputs"), so several inputs are one record and the captured
  // function has two parameters, where it once had one a chained input.
  test("all typed binding shapes capture the original callable and preserve its native result"):
    val dir = scratch.resolve("binding-arities")
    val bindings = Seq(0, 1, 2, 3, 4, 6).map { arity =>
      val record = s"Inputs$arity"
      val (declaration, types, parameters, reads, arguments, params) = arity match
        case 0 => ("", "Boolean", "(s: Boolean)", "s", "false", 1)
        case 1 =>
          (
            ".input[Boolean](\"i1\")",
            "(Boolean, Boolean)",
            "(s: Boolean, a1: Boolean)",
            "s && a1",
            "false, true",
            2
          )
        case _ =>
          (
            s".inputs[$record]",
            s"(Boolean, $record)",
            s"(s: Boolean, in: $record)",
            ("s" +: (1 to arity).map(i => s"in.i$i")).mkString(" && "),
            s"false, $record(${List.fill(arity)("true").mkString(", ")})",
            2
          )
      s"""    val action$arity = action(Actor("user"))$declaration
    val f$arity: $types => List[Step[Boolean, ArityOutcome, Nothing]] = $parameters => List(Step(ArityOutcome.ok, $reads, Nil, ""))
    val binding$arity: StepBinding[Boolean, ArityOutcome, Nothing] = action$arity ~> f$arity
    val captured$arity = framework.bridge.BindingProbe.function(binding$arity)
    assert(captured$arity.params.size == $params)
    assert(binding$arity.function.eq(f$arity))
    assert(framework.bridge.BindingProbe.callable(binding$arity).eq(f$arity))
    assert(!f$arity($arguments).head.state)
    assert(!framework.bridge.BindingProbe.callable(binding$arity).asInstanceOf[$types => List[Step[Boolean, ArityOutcome, Nothing]]]($arguments).head.state)
"""
    }.mkString
    val records = Seq(2, 3, 4, 6)
      .map(arity =>
        s"final case class Inputs$arity(${(1 to arity).map(i => s"i$i: Boolean").mkString(", ")})"
      )
      .mkString("\n")
    source(
      dir,
      "Probe.scala",
      """package framework.bridge
import framework.dsl.StepBinding
import io.temporal.server.api.umpire.v1 as ir
object BindingProbe:
  def function[S, O, F](binding: StepBinding[S, O, F]): ir.Function = binding.captured.get.function
  def callable[S, O, F](binding: StepBinding[S, O, F]): AnyRef = binding.captured.get.callable
"""
    )
    source(
      dir,
      "Arities.scala",
      s"""package client.boundary
import framework.dsl.*
import framework.dsl.Finite.given
enum ArityOutcome:
  case ok
$records
object BindingArities:
  def main(args: Array[String]): Unit =
    val function = Reify.function((s: Boolean) => !s)
    assert(function.getBody.getUnary.op.name == "OP_NOT")
$bindings
"""
    )
    val ran = tools.scalaCli(
      Seq("run", dir.toString, "--main-class", "client.boundary.BindingArities")
        ++ settings ++ pluginOption
    )
    assertEquals(ran.exit, 0, ran.diagnostics)

  test("records refuses nested and forwarded plain selectors at the authored call"):
    for (name, extra, call, refusal) <- Seq(
        (
          "nested",
          "",
          "after.records(_.left.flag, RecordedFact.noted)",
          "records reads the facts of the member one field of the composed state names"
        ),
        (
          "forwarded",
          "  @sugar def forwarded(after: Step[Pair, Boolean, String], member: Pair => Member): Boolean = after.records(member, RecordedFact.noted)\n",
          "forwarded(after, _.left)",
          "selector capture cannot cross a plain function parameter"
        )
      )
    do
      val dir = scratch.resolve(s"bad-selector-$name")
      val text = s"""package framework.boundary
import framework.dsl.*
case class Member(flag: Boolean)
case class Pair(left: Member, right: Member)
enum RecordedFact:
  case noted
object BadSelector:
$extra  val captured = Reify.capture((after: Step[Pair, Boolean, String]) => $call)
"""
      source(dir, "Bad.scala", text)
      val line = text.linesIterator.zipWithIndex.find(_._1.contains("val captured")).get._2 + 1
      val ran = tools.scalaCli(Seq("compile", dir.toString) ++ settings ++ pluginOption)
      assert(ran.failed, ran.diagnostics)
      assert(ran.diagnostics.contains(refusal), ran.diagnostics)
      assert(ran.diagnostics.contains(s"Bad.scala:$line:"), ran.diagnostics)
      assert(!ran.diagnostics.contains("unhandled exception"), ran.diagnostics)

  test(
    "an absent or ambiguous capture implementation refuses instead of constructing a malformed call"
  ):
    for (name, implementations) <- Seq(
        "absent" -> "",
        "ambiguous" -> """  def target(f: Boolean => Boolean, root: String, chunks: Seq[String]): ir.Function = ir.Function.defaultInstance
  def target(f: Boolean => Boolean, root: String): ir.Function = ir.Function.defaultInstance
"""
      )
    do
      val dir = scratch.resolve(s"bad-implementation-$name")
      val text = s"""package framework.boundary
import framework.dsl.capture
import io.temporal.server.api.umpire.v1 as ir
object BadFactory:
  def apply[F](@capture("target") f: F): ir.Function = throw IllegalStateException("missing capture")
$implementations  val captured = apply((s: Boolean) => !s)
"""
      source(dir, "BadFactory.scala", text)
      val ran = tools.scalaCli(Seq("compile", dir.toString) ++ settings ++ pluginOption)
      assert(ran.failed, ran.diagnostics)
      assert(
        ran.diagnostics.contains(
          "capture implementation framework.boundary.BadFactory$.target is missing or not a unique method"
        ),
        ran.diagnostics
      )
      val line = text.linesIterator.zipWithIndex.find(_._1.contains("val captured")).get._2 + 1
      assert(ran.diagnostics.contains(s"BadFactory.scala:$line:"), ran.diagnostics)
      assert(!ran.diagnostics.contains("unhandled exception"), ran.diagnostics)
