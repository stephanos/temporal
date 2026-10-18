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
import framework.dsl.Lifted
object LastHelper extends Lifted:
  def invert(s: Boolean): Boolean = !s
  def forward(s: Boolean): Boolean = invert(s)
"""
    )
    val ran = tools.scalaCli(
      Seq("run", both.toString, "--main-class", "framework.boundary.SameRunCaller")
        ++ settings ++ pluginOption
    )
    assertEquals(ran.exit, 0, ran.diagnostics)

  test("another source unit needs stored helper IR while inline sugar uses the compiler"):
    val helper = scratch.resolve("unmarked-helper")
    source(
      helper,
      "Helper.scala",
      """package framework.boundary
import framework.dsl.*
object Unmarked:
  inline def eligible(s: Boolean): Boolean = !s
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
      if name == "sugar" then assertEquals(ran.exit, 0, ran.diagnostics)
      else
        assert(ran.failed, s"$name: ${ran.diagnostics}")
        assert(ran.diagnostics.contains(s"framework.boundary.Unmarked$$.$reached"), ran.diagnostics)
        assert(
          ran.diagnostics.contains(
            s"framework.boundary.Unmarked$$.$reached is reached from another file: define it in an object that extends Lifted"
          ),
          ran.diagnostics
        )
        assert(ran.diagnostics.contains("Caller.scala:4:"), ran.diagnostics)
      assert(!ran.diagnostics.contains("unhandled exception"), ran.diagnostics)

  test("stored helpers may reach an unmarked helper in their own file"):
    val dir = scratch.resolve("helper-locality")
    source(
      dir,
      "ABridge.scala",
      """package framework.boundary
import framework.dsl.*
object Bridge extends Lifted:
  def forward(value: Boolean): Boolean = LocalVocabulary.forward(value)
"""
    )
    source(
      dir,
      "ZVocabulary.scala",
      """package framework.boundary
import framework.dsl.*
object LocalLeaf:
  def invert(value: Boolean): Boolean = !value
object LocalVocabulary extends Lifted:
  def forward(value: Boolean): Boolean = LocalLeaf.invert(value)
"""
    )
    source(
      dir,
      "Caller.scala",
      """package framework.boundary
import framework.dsl.*
object LocalityCaller:
  def main(args: Array[String]): Unit =
    val captured = Reify.capture(Bridge.forward)
    assert(captured.model.functions.size == 3)
    assert(captured.model.functions.find(_.name.endsWith(".invert")).get.getBody.getUnary.op.name == "OP_NOT")
    assert(captured.callable(false))
"""
    )
    val ran = tools.scalaCli(
      Seq(
        "run",
        dir.toString,
        "--main-class",
        "framework.boundary.LocalityCaller"
      ) ++ settings ++ pluginOption
    )
    assertEquals(ran.exit, 0, ran.diagnostics)

    source(
      dir,
      "ABridge.scala",
      """package framework.boundary
import framework.dsl.*
object Bridge extends Lifted:
  def forward(value: Boolean): Boolean = InlineBridge.forward(value)
"""
    )
    source(
      dir,
      "ZVocabulary.scala",
      """package framework.boundary
object LocalLeaf:
  def invert(value: Boolean): Boolean = !value
object InlineBridge:
  inline def forward(value: Boolean): Boolean = LocalLeaf.invert(value)
"""
    )
    val refused = tools.scalaCli(Seq("compile", dir.toString) ++ settings ++ pluginOption)
    assert(refused.exit != 0, refused.diagnostics)
    assert(
      refused.diagnostics.contains(
        "framework.boundary.LocalLeaf$.invert is reached from another file: define it in an object that extends Lifted"
      ),
      refused.diagnostics
    )
    assert(refused.diagnostics.contains("ABridge.scala:4:"), refused.diagnostics)

  test("same-file tree lifting agrees with the stored helper body"):
    val dir = scratch.resolve("tree-stored")
    source(
      dir,
      "Helper.scala",
      """package framework.boundary
import framework.dsl.*
object TreeHelper extends Lifted:
  def invert(value: Boolean): Boolean = !value
  val local = Reify.function(invert)
  inline def twice(value: Boolean): Boolean = value && value
  def strict(value: Boolean): Boolean = twice(invert(value))
  val localStrict = Reify.function(strict)
  def collisions(value: Boolean): Boolean =
    val boolean = value
    twice(twice(invert(value))) && twice(invert(boolean))
  val localCollisions = Reify.function(collisions)
object StepHelper:
  def keep(value: Boolean): List[Step[Boolean, Boolean, Nothing]] = List(Step(true, value, Nil, ""))
"""
    )
    source(
      dir,
      "Caller.scala",
      """package framework.boundary
import framework.dsl.*
object StoredCaller:
  def main(args: Array[String]): Unit =
    val stored = Reify.function(TreeHelper.invert)
    assert(stored.clearPosition == TreeHelper.local.clearPosition)
    assert(stored.getBody.getUnary.op.name == "OP_NOT")
    val strict = Reify.function(TreeHelper.strict)
    assert(strict.clearPosition == TreeHelper.localStrict.clearPosition, s"stored: $strict\ntree: ${TreeHelper.localStrict}")
    assert(strict.getBody.getLet.getValue.getCall.function.endsWith(".invert"))
    val collisions = Reify.function(TreeHelper.collisions)
    assert(collisions.clearPosition == TreeHelper.localCollisions.clearPosition)
    assert(collisions.getBody.getLet.name == "boolean")
    val conjunction = collisions.getBody.getLet.getBody.getBinary
    val temporaries = Seq(conjunction.getLeft.getLet, conjunction.getLeft.getLet.getValue.getLet, conjunction.getRight.getLet)
    assert(temporaries.map(_.name) == Seq("boolean2", "boolean3", "boolean4"))
    for temporary <- temporaries do
      assert(temporary.getBody.getBinary.getLeft.getVar == temporary.name)
      assert(temporary.getBody.getBinary.getRight.getVar == temporary.name)
    assert(conjunction.getRight.getLet.getValue.getCall.args.head.getVar == "boolean")
    assert(TreeHelper.collisions(false) && !TreeHelper.collisions(true))
    val steps = Reify.capture(StepHelper.keep)
    assert(steps.model.functions.map(_.name) == Seq("framework.boundary.StepHelper$.keep"))
    assert(steps.function.getBody.getList.items.head.getConstruct.args.head.getLiteral.getBool)
    assert(steps.callable(false).head.state == false)
"""
    )
    val ran = tools.scalaCli(
      Seq(
        "run",
        dir.toString,
        "--main-class",
        "framework.boundary.StoredCaller"
      ) ++ settings ++ pluginOption
    )
    assertEquals(ran.exit, 0, ran.diagnostics)

  test("Machine marks authored helpers in arbitrarily named nested containers"):
    val dir = scratch.resolve("machine-marker")
    source(
      dir,
      "Subject.scala",
      """package framework.boundary
import framework.dsl.*
given Ok[Boolean] = Ok(true)
class Subject extends Machine[Boolean, Nothing]:
  var flag = false
  override def end: Boolean = false
  object transitions extends Transitions
  object vocabulary:
    object nested:
      def invert(value: Boolean): Boolean = !value
class Endpoint extends RecordMachine[Boolean, Boolean, Nothing]:
  val init = false
  def end(value: Boolean): Boolean = !value
  object transitions extends Bindings[Boolean, Boolean, Nothing]()
object Subjects:
  val subject = new Subject
  val endpoint = new Endpoint
"""
    )
    source(
      dir,
      "Caller.scala",
      """package framework.boundary
import framework.dsl.*
object MachineCaller:
  def main(args: Array[String]): Unit =
    val captured = Reify.capture(Subjects.subject.vocabulary.nested.invert)
    assert(captured.function.getBody.getUnary.op.name == "OP_NOT")
    assert(captured.callable(false))
    val endpoint = Reify.capture(Subjects.endpoint.end)
    assert(endpoint.function.getBody.getUnary.op.name == "OP_NOT")
    assert(endpoint.callable(false))
"""
    )
    val ran = tools.scalaCli(
      Seq(
        "run",
        dir.toString,
        "--main-class",
        "framework.boundary.MachineCaller"
      ) ++ settings ++ pluginOption
    )
    assertEquals(ran.exit, 0, ran.diagnostics)

  test("a local def in a marked container is not eagerly stored"):
    val dir = scratch.resolve("local-helper")
    source(
      dir,
      "Helper.scala",
      """package framework.boundary
import framework.dsl.*
object LocalHelper extends Lifted:
  inline def run(value: Boolean): Boolean =
    def local(): Boolean =
      var result = false
      result = value
      result
    local()
object LocalCaller:
  def main(args: Array[String]): Unit = assert(LocalHelper.run(true))
"""
    )
    val ran = tools.scalaCli(
      Seq(
        "run",
        dir.toString,
        "--main-class",
        "framework.boundary.LocalCaller"
      ) ++ settings ++ pluginOption
    )
    assertEquals(ran.exit, 0, ran.diagnostics)

  test("a marked definition compiled without the plugin has no stored body"):
    val helper = scratch.resolve("without-plugin-helper")
    source(
      helper,
      "Helper.scala",
      """package framework.boundary
import framework.dsl.Lifted
object Unstored extends Lifted:
  def invert(value: Boolean): Boolean = !value
"""
    )
    val jar = scratch.resolve("without-plugin.jar")
    tools
      .scalaCli(
        Seq("--power", "package", "--library", helper.toString, "-o", jar.toString) ++ settings
      )
      .orFail(): Unit
    val caller = scratch.resolve("without-plugin-caller")
    source(
      caller,
      "Caller.scala",
      """package framework.boundary
import framework.dsl.Reify
object UnstoredCaller:
  val captured = Reify.function(Unstored.invert)
"""
    )
    val ran = tools.scalaCli(
      Seq("compile", caller.toString, "--jar", jar.toString) ++ settings ++ pluginOption
    )
    assert(ran.failed, ran.diagnostics)
    assert(
      ran.diagnostics.contains(
        "framework.boundary.Unstored$.invert is reached from another file: define it in an object that extends Lifted"
      ),
      ran.diagnostics
    )
    assert(ran.diagnostics.contains("Caller.scala:4:"), ran.diagnostics)
    assert(!ran.diagnostics.contains("unhandled exception"), ran.diagnostics)

  test("unsupported bodies refuse at their tree in captures and lifted definitions"):
    val bodies = Seq(
      "definition" -> """package framework.boundary
import framework.dsl.Lifted
object Broken extends Lifted:
  def broken(s: Boolean): Boolean =
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

  test("recursive marked helpers keep the body lifter's refusal"):
    val dir = scratch.resolve("recursive-helper")
    source(
      dir,
      "Recursive.scala",
      """package framework.boundary
import framework.dsl.Lifted
object Recursive extends Lifted:
  def first(value: Boolean): Boolean = second(value)
  def second(value: Boolean): Boolean = first(value)
"""
    )
    val ran = tools.scalaCli(Seq("compile", dir.toString) ++ settings ++ pluginOption)
    assert(ran.failed, ran.diagnostics)
    assert(
      ran.diagnostics.contains(
        "calls itself, directly or through another function: a recursive function has no IR form"
      ),
      ran.diagnostics
    )
    assert(!ran.diagnostics.contains("unhandled exception"), ran.diagnostics)

  test("a def of two or more parameter lists is refused at its call in lifted code"):
    for (name, text, line, count) <- Seq(
        (
          "capture",
          """package framework.boundary
import framework.dsl.Reify
object Curried:
  def warmBy(reading: Int)(step: Int): Int = reading + step
  val captured = Reify.function((s: Int) => warmBy(s)(1))
""",
          5,
          "two"
        ),
        (
          "definition",
          """package framework.boundary
import framework.dsl.Lifted
object Curried extends Lifted:
  def warmBy(reading: Int)(step: Int)(scale: Int): Int = (reading + step) * scale
  def heat(s: Int): Int = warmBy(s)(1)(2)
""",
          5,
          "three"
        )
      )
    do
      val dir = scratch.resolve(s"curried-$name")
      source(dir, "Curried.scala", text)
      val ran = tools.scalaCli(Seq("compile", dir.toString) ++ settings ++ pluginOption)
      assert(ran.failed, ran.diagnostics)
      assert(
        ran.diagnostics.contains(
          s"framework.boundary.Curried.warmBy has $count parameter lists: write it with one"
        ),
        ran.diagnostics
      )
      assert(ran.diagnostics.contains(s"Curried.scala:$line:"), ran.diagnostics)
      assert(!ran.diagnostics.contains("unhandled exception"), ran.diagnostics)

  test("a using clause does not count as a parameter list and its arguments follow the list's"):
    val dir = scratch.resolve("using-clause")
    source(
      dir,
      "UsingClause.scala",
      """package framework.boundary
import framework.dsl.Reify
object UsingClause:
  def warmBy(reading: Int)(using step: Int): Int = reading + step
  def main(args: Array[String]): Unit =
    val captured = Reify.capture((s: Int) => warmBy(s)(using 1))
    assert(captured.callable(1) == 2)
    val helper = captured.model.functions.find(_.name.endsWith(".warmBy")).get
    assert(helper.params.map(_.name) == Seq("reading", "step"), helper.params.toString)
    assert(captured.function.getBody.getCall.args.size == 2, captured.function.toString)
    println("using-clause-accepted")
"""
    )
    val ran = tools.scalaCli(
      Seq("run", dir.toString, "--main-class", "framework.boundary.UsingClause")
        ++ settings ++ pluginOption
    )
    assertEquals(ran.exit, 0, ran.diagnostics)
    assert(ran.output.contains("using-clause-accepted"), ran.diagnostics)

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
import framework.dsl.Lifted
object Large extends Lifted:
  def all(s: Boolean): Boolean = ${balanced(10)}
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
import framework.dsl.Lifted
object WarmHelper extends Lifted:
  def result(s: Boolean): Boolean = $body
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
      println(first.output.trim)
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
      println(second.output.trim)
      println(
        second.diagnostics.linesIterator.filter(_.contains("after phase typer")).mkString("\n")
      )
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
      println(recovered.output.trim)
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
import framework.dsl.Lifted
case class Leaf(flag: Boolean)
case class Record(leaf: Leaf, optional: Option[Leaf])
object Records extends Lifted:
  def flag(r: Record): Boolean = r.leaf.flag
  def again(r: Record): Boolean = flag(r)
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
object Choices extends Lifted:
  val first = choice
  val second = choice
  def single(s: Boolean): List[Step[Boolean, ChoiceOutcome, Nothing]] =
    List(Step(ChoiceOutcome.ok, !s, Nil, ""))
  def forward(s: Boolean): List[Step[Boolean, ChoiceOutcome, Nothing]] = single(s)
  def multiple(s: Boolean): List[Step[Boolean, ChoiceOutcome, Nothing]] =
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
          "  inline def forwarded(after: Step[Pair, Boolean, String], member: Pair => Member): Boolean = after.records(member, RecordedFact.noted)\n",
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
