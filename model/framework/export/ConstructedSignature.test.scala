package framework.`export`

import java.nio.file.{Files, Path}
import java.util.Base64
import scala.concurrent.duration.*
import framework.check.{Output, RetainedScratch, Tools}
import framework.lint.ModulePolicy
import io.temporal.server.api.umpire.v1 as ir

class ConstructedSignatureSuite extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private lazy val scratch =
    retained(tools.directory.resolve(".flow/tmp"), "constructed-signature.")
  private lazy val library = ModelBuild.frameworkLibrary:
    val jar = scratch.resolve("framework.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          "model/framework/dsl",
          "model/framework/realize",
          "--server=false",
          "--library",
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    jar
  private lazy val plugin = ModelBuild.reifyPlugin:
    val jar = scratch.resolve("reify.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          "model/framework/reify/plugin",
          "--server=false",
          "--assembly",
          "--preamble=false",
          "--main-class-in-manifest=false",
          "--provided",
          "org.scala-lang:scala3-compiler_3",
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    jar
  private def settings = Seq(
    "--scala",
    "3.9.0",
    "--jvm",
    "27",
    "--server=false",
    "--dependency",
    "com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20",
    "--jar",
    library.toString,
    "--jar",
    tools.directory.resolve("model/build/ir-scalapb.jar").toString,
    "--scalac-option",
    s"-Xplugin:$plugin",
    "--scalac-option",
    "-P:umpire-reify:mode=signature"
  ) ++ Seq(
    "-Werror",
    "-deprecation",
    "-feature",
    "-unchecked",
    "-Wunused:all",
    "-Wvalue-discard",
    "-Wnonunit-statement",
    "-Wsafe-init",
    "-Wimplausible-patterns",
    "-language:strictEquality"
  ).flatMap(o => Seq("--scalac-option", o))

  private def source(name: String, body: String): Path =
    val dir = Files.createDirectories(scratch.resolve(name))
    Files.writeString(dir.resolve("Probe.scala"), body)

  private def compile(file: Path, name: String, extra: Seq[String] = Nil): (Path, String) =
    val jar = scratch.resolve(s"$name.jar")
    val ran = tools.scalaCli(
      Seq(
        "compile",
        file.toString,
        "--print-class-path",
        "--compilation-output",
        jar.toString
      ) ++ settings ++ extra,
      Output.KeptApart
    )
    assertEquals(ran.exit, 0, ran.diagnostics)
    (jar, ran.output.trim)

  private def run(classpath: String, main: String): String =
    val ran = tools.run("java", Seq("-cp", classpath, main))
    assertEquals(ran.exit, 0, ran.diagnostics)
    ran.output

  private def withoutPositions(model: ir.Model): ir.Model = model.copy(
    actions = model.actions.map(_.clearPosition),
    channels = model.channels.map(_.clearPosition),
    types = model.types.map(_.clearPosition)
  )

  private def closure(model: ir.Model): Seq[ir.Type] =
    val reached = scala.collection.mutable.Set.empty[String]
    def visit(ref: ir.TypeRef): Unit = ref.ref match
      case ir.TypeRef.Ref.Named(name) if reached.add(name) =>
        val value = model.types.find(_.name == name).get
        (value.getRecord.fields ++ value.getEnum.cases.flatMap(_.fields)).foreach(f =>
          visit(f.getType)
        )
      case _ => ()
    model.actions.flatMap(_.inputs).foreach(p => visit(p.getType))
    model.channels.foreach(c => visit(c.getMessage))
    model.types.filter(t => reached(t.name))

  test(
    "native signature matches independent old actions, channels and referenced closure in separate JVMs"
  ):
    val stored = tools.directory.resolve("model/framework/export/testdata/constructedSignature")
    val dir = Files.createDirectories(scratch.resolve("oracle"))
    for name <- Seq("Shapes.scala", "Native.scala") do
      Files.copy(stored.resolve(name), dir.resolve(name)): Unit
    val (jar, classpath) = compile(dir, "oracle")
    val first = run(classpath, "framework.signature.Native")
    val second = run(classpath, "framework.signature.Native")
    def payload(output: String, prefix: String): Array[Byte] =
      Base64.getDecoder.decode(
        output.linesIterator.find(_.startsWith(prefix)).get.stripPrefix(prefix)
      )
    assertEquals(payload(first, "signature:").toVector, payload(second, "signature:").toVector)
    assertEquals(payload(first, "defaults:").toVector, payload(second, "defaults:").toVector)
    val native = ir.Model.parseFrom(payload(first, "signature:"))
    val original = scalapb.json4s.JsonFormat
      .fromJsonString[ir.Model](Files.readString(stored.resolve("original.json")))
    assertEquals(
      withoutPositions(native),
      withoutPositions(original.copy(types = closure(original)))
    )
    assertEquals(
      original.types.map(_.name).toSet -- closure(original).map(_.name),
      Set("scala.None")
    )
    val oldDefault = scalapb.json4s.JsonFormat
      .fromJsonString[ir.ActionClass](Files.readString(stored.resolve("original-defaults.json")))
    val nativeDefault = ir.ActionClass.parseFrom(payload(first, "defaults:"))
    assertEquals(nativeDefault.inputs.head, oldDefault.inputs.head)
    // fn-174.24, kept explicit: the historical reader typed the nested default `Some(None)`'s inner
    // value as `scala.None`; the native record default keeps its declared `Option[Mode]` type with
    // case `None`. original-defaults.json stays the unchanged historical output, not the contract.
    assertEquals(oldDefault.inputs(1).getEnum.fields.head.getEnum.`type`, "scala.None")
    assertEquals(
      nativeDefault.inputs(1).getEnum.fields.head.getEnum.`type`,
      "scala.Option[fixture.signature.Mode]"
    )
    assertEquals(nativeDefault.inputs(1).getEnum.fields.head.getEnum.`case`, "None")
    val cp = Files.writeString(scratch.resolve("oracle.classpath"), classpath)
    val out = scratch.resolve("current-reader.json")
    val lifted = InProcess.lift(
      tools.directory,
      Seq(s"$jar=$dir/", cp.toString, out.toString, "fixture.signature.Shapes$package$.root"),
      policy = ModulePolicy.Generic
    )
    assertEquals(lifted.exit, 0, lifted.diagnostics)
    val current = scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(out))
    assertEquals(
      withoutPositions(
        ir.Model(actions = current.actions, channels = current.channels, types = current.types)
      ),
      withoutPositions(original)
    )

  // Standalone `Input` tokens and appended input ladders are retired (model/framework/README.md,
  // "Inputs"): a token-declared input is a named scalar, and inputs once appended to a bound action
  // are fields of one native record, whose per-field nominal keys and defaults this still pins.
  test(
    "helper jars preserve nominal shared-witness keys through aliases, records and all action copies"
  ):
    for order <- Seq("first", "second") do
      val declarations = if order == "first" then
        "given first: Finite[First] = Finite.upTo(2)\n  given second: Finite[Second] = first"
      else "given second: Finite[Second] = Finite.upTo(2)\n  given first: Finite[First] = second"
      val helper = source(
        s"helper-$order",
        """package helper.signature
import framework.dsl.*
object Domains:
  opaque type First = Int
  opaque type Second = Int
  DECLARATIONS
  // Structural evidence for the opaque integers, which Option and record catalogs of them need.
  given Data.Structural[First] = Data.Structural.given_Structural_Int
  given Data.Structural[Second] = Data.Structural.given_Structural_Int
object Helpers:
  def named[A](base: Action[EmptyTuple], name: String)(using Finite[A]): ScalarAction[A] = base.input[A](name)
  def channelFor[A](using Finite[A]): Channel[A] = channel[A](1, Order.fifo, Loss.lossy)
  def forwardChannel[A](value: Channel[A]): Channel[A] = value
  def forward[A](action: ScalarAction[A]): ScalarAction[A] = action.results("Reply")
""".replace("DECLARATIONS", declarations)
      )
      val (helperJar, _) = compile(helper, s"helper-$order")
      val client = source(
        s"client-$order",
        """package framework.signatureclient
import framework.dsl.*
import helper.signature.{Domains, Helpers}
object Probe:
  type Alias = Domains.First
  final case class Envelope(message: Alias)
  final case class Extended(message: Alias, flag: Boolean, token: Alias)
  final case class Appended(message: Alias, optional: Option[Domains.Second], flag: Boolean = false)
  val first = Helpers.named[Alias](action("first", Actor("caller")), "message")
  val second = Helpers.named[Domains.Second](action("second", Actor("caller")), "message")
  val alias = first
  val copied = Helpers.forward(alias).example(0.asInstanceOf[Alias], "zero")
  val bound = copied.on(Entity("record")).creates(Entity("created"))
  val firstWire = Helpers.channelFor[Alias]
  val secondWire = Helpers.channelFor[Domains.Second]
  val forwardedWire = Helpers.forwardChannel(firstWire)
  val nestedWire = Helpers.channelFor[Envelope]
  val optionalAction = action("optional", Actor("caller")).input[Option[Alias]]("optional")
  val canonicalOptional =
    action("canonical", Actor("caller")).input[Option[Domains.First]]("optional")
  val extended = action("extended", Actor("caller")).inputs[Extended].on(Entity("record"))
  val appended = action("appended", Actor("caller")).inputs[Appended].on(Entity("record"))
  def main(args: Array[String]): Unit =
    val originals = if "ORDER" == "first" then Vector(first, second) else Vector(second, first)
    originals.foreach(_.toIR: Unit)
    assert(Domains.first eq Domains.second)
    def generic[A](f: Finite[A]): Option[FiniteSchema] = f.schema
    assert(generic(Domains.first).isEmpty)
    assert(first.decl.schemaKeys.head.contains("helper.signature.Domains.First"))
    assert(second.decl.schemaKeys.head.contains("helper.signature.Domains.Second"))
    assert(alias.decl eq first.decl)
    assert(bound.decl.sourceIdentity == first.decl.sourceIdentity)
    assert(bound.decl.sourceBinding.contains("framework.signatureclient.Probe.bound"))
    assert(bound.decl.schemaKeys == first.decl.schemaKeys)
    assert(bound.toIR.inputs.head.getType.getIntRange.high == 2)
    assert(bound.toIR.examples.head.getValue.getInt == 0)
    assert(firstWire.schemaKey.contains("helper.signature.Domains.First"))
    assert(secondWire.schemaKey.contains("helper.signature.Domains.Second"))
    assert(forwardedWire eq firstWire)
    assert(forwardedWire.toIR.id == "framework.signatureclient.Probe.firstWire")
    assert(forwardedWire.toIR.name == "firstWire")
    assert(forwardedWire.deliver.toIR.id == "framework.signatureclient.Probe.firstWire.deliver")
    try
      firstWire.declaredAt("fixture.invalid"): Unit
      throw new AssertionError("authored declaration conflict accepted")
    catch
      case error: IllegalArgumentException => assert(error.getMessage.contains("conflicts with source identity"))
    assert(firstWire.toIR.id == "framework.signatureclient.Probe.firstWire")
    assert(firstWire.deliver.toIR.delivers == "framework.signatureclient.Probe.firstWire")
    assert(secondWire.lose.toIR.loses == "framework.signatureclient.Probe.secondWire")
    assert(extended.toIR.inputs.map(_.name) == Seq("message", "flag", "token"))
    assert(extended.decl.schemaKeys.head.contains("helper.signature.Domains.First"))
    assert(extended.decl.schemaKeys.last.contains("helper.signature.Domains.First"))
    assert(appended.toIR.inputs.map(_.getType.ref.isBool) == Seq(false, false, true))
    assert(appended.decl.schemaKeys.head == first.decl.schemaKeys.head)
    assert(appended.decl.schemaKeys(1).exists(k => k.startsWith("scala.Option[helper.signature.Domains") && k.endsWith(".Second]")))
    assert(appended.decl.schemaKeys.last.contains("scala.Boolean"))
    assert(appended.inputMetadata.last.default.get.getBool == false)
    assert(appended.inputMetadata.head.default.isEmpty)
    assert(optionalAction.toIR.inputs.head.getType.ref.isNamed)
    assert(optionalAction.decl.schemaKeys == canonicalOptional.decl.schemaKeys)
    assert(nestedWire.toIR.getMessage.getNamed == "framework.signatureclient.Probe.Envelope")
    assert(Signature.collect(Vector(optionalAction), Vector(nestedWire)).types.flatMap(_.getRecord.fields).head.getType.getIntRange.high == 2)
""".replace("ORDER", order)
      )
      val (_, cp) = compile(client, s"client-$order", Seq("--jar", helperJar.toString))
      run(cp, "framework.signatureclient.Probe"): Unit

  test("empty, restricted and throwing witnesses retain structural shape without channel contents"):
    val file = source(
      "catalogs",
      """package framework.signaturecatalogs
import framework.dsl.*
enum Mode:
  case first, second
// A values-only witness; the reify plugin does not compile an anonymous `new Finite[T]:`.
final class Unread[T](message: String)(using Data.Structural[T]) extends Finite[T]:
  def values = throw new AssertionError(message)
object Probe:
  val throwing: Finite[Mode] = Unread[Mode]("shape enumerated catalog")
  val empty = Finite.of[Mode]()
  val restricted = Finite.of[Mode](Mode.second)
  val actionOnly = action("send", Actor("caller")).input[Mode]("mode")(using throwing)
  val wire = channel[Mode](2, Order.unordered, Loss.lossy, 1)(using throwing)
  val emptyAction = action("empty", Actor("caller")).input[Mode]("mode")(using empty)
  val restrictedAction = action("restricted", Actor("caller")).input[Mode]("mode")(using restricted)
  def main(args: Array[String]): Unit =
    val model = Signature.collect(Vector(actionOnly, emptyAction, restrictedAction, wire.deliver, wire.lose), Vector(wire))
    assert(model.types.size == 1)
    assert(model.types.head.getEnum.cases.map(_.name) == Seq("first", "second"))
    assert(wire.toIR.duplicates == 1)
    assert(actionOnly.toIR.inputs.head.getType.getNamed == "framework.signaturecatalogs.Mode")
"""
    )
    val (_, cp) = compile(file, "catalogs")
    run(cp, "framework.signaturecatalogs.Probe"): Unit

  test("plain Int channel messages refuse contextually while plain Int action inputs retain Int"):
    val file = source(
      "int",
      """package framework.signatureint
import framework.dsl.*
object Probe:
  given Finite[Int] = Finite.upTo(2)
  val send = action("send", Actor("caller")).input[Int]("count")
  val wire = channel[Int](1, Order.fifo, Loss.reliable)
  def main(args: Array[String]): Unit =
    assert(send.toIR.inputs.head.getType.ref.isInt)
    for convert <- Seq[() => Any](() => wire.toIR, () => wire.deliver.toIR, () => wire.lose.toIR) do
      try
        convert(): Unit
        throw new AssertionError("plain Int channel accepted")
      catch
        case error: IllegalArgumentException =>
          assert(error.getMessage.contains("framework.signatureint.Probe.wire"))
          assert(error.getMessage.contains("Int with no declared bound"))
"""
    )
    val (_, cp) = compile(file, "int")
    run(cp, "framework.signatureint.Probe"): Unit
