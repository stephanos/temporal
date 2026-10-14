package framework.`export`

import java.nio.file.{Files, Path}
import scala.concurrent.duration.*
import framework.check.{Output, RetainedScratch, Tools}

class ConstructedNamesSuite extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private lazy val scratch = retained(tools.directory.resolve(".flow/tmp"), "constructed-names.")
  private def source(dir: Path, body: String): Path =
    Files.createDirectories(dir): Unit
    Files.writeString(dir.resolve("Probe.scala"), body)

  private lazy val library =
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

  private lazy val plugin =
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
    tools.directory.resolve("model/build/ir-scalapb.jar").toString
  ) ++
    Seq(
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

  private val probe = """package client.naming
import framework.realize.*
import com.google.protobuf.duration.Duration
object Probe:
  private given CanEqual[Instruction, Instruction] = CanEqual.derived
  val original: Instruction = Instruction.Hold("gate")
  val alias: Instruction = original
  def scripted(call: Instruction): Script = script("controller", Activation.Controller)(everyCase(call))
  def twice(call: Instruction): Script = scripted(call)
  def many(calls: Instruction*): Vector[Item] = calls.toVector.map(everyCase)
  final case class Box(call: Instruction)
  def defaulted(call: Instruction = original): Script = scripted(call)
  def identityAny(call: Any): Any = call
  def identityRef(call: AnyRef): AnyRef = call
  def identityObject(call: Object): Object = call
  def identityMatchable(call: Matchable): Matchable = call
  def defaultAny(call: Any = alias): Any = call
  def defaultRef(call: AnyRef = alias): AnyRef = call
  def defaultTopVector(calls: Vector[Any] = Vector[Any](alias)): Vector[Any] = calls
  private var evaluations = 0
  private val method = io.grpc.MethodDescriptor.newBuilder[Duration, Duration]()
    .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
    .setFullMethodName("example.Clock/Read")
    .setRequestMarshaller(new scalapb.grpc.Marshaller(Duration))
    .setResponseMarshaller(new scalapb.grpc.Marshaller(Duration)).build()
  val originalCall = rpc("clock", method) { evaluations += 1 }
  val variant = originalCall.withFields { field(_.seconds) := Operand.number(2) }
  val extendedCall = originalCall.extended { field(_.seconds) := Operand.number(3) }
  def main(args: Array[String]): Unit =
    assert(original eq alias)
    assert(original == alias)
    assert(identityAny(alias).asInstanceOf[AnyRef] eq original)
    assert(identityRef(alias) eq original)
    assert(identityObject(alias) eq original)
    assert(identityMatchable(alias).asInstanceOf[AnyRef] eq original)
    println(alias)
    assert(defaultAny().asInstanceOf[AnyRef] eq original)
    assert(defaultRef() eq original)
    assert(defaultTopVector().head.asInstanceOf[AnyRef] eq original)
    assert(Vector[Any](alias).head.asInstanceOf[AnyRef] eq original)
    val values = Vector(scripted(original), scripted(alias), twice(alias),
      scripted(call = alias), scripted(Box(alias).call), defaulted())
    val ids = values.map(_.items.head.command.get.toIR("client.naming.realization").id)
    assert(ids == Vector("original", "alias", "alias", "alias", "alias", "original"), ids.toString)
    assert(many(original, alias).map(_.command.get.toIR("client.naming.realization").id) == Vector("original", "alias"))
    assert(original.ref.isEmpty)
    val rpcIds = Vector(everyCase(variant), everyCase(extendedCall)).map(_.command.get.toIR("client.naming.realization").id)
    assert(rpcIds == Vector("original-call", "extended-call"), rpcIds.toString)
    assert(variant.assign.size == 1 && extendedCall.assign.size == 1)
    assert(variant.method eq originalCall.method)
    assert(evaluations == 1)
    println(ids.mkString(","))
"""

  test(
    "the real names phase retains references through helpers, defaults, constructors and varargs"
  ):
    val file = source(scratch.resolve("native"), probe)
    val ran = tools.scalaCli(
      Seq("run", file.toString, "--main-class", "client.naming.Probe") ++
        settings ++ Seq("--scalac-option", s"-Xplugin:$plugin")
    )
    assertEquals(ran.exit, 0, ran.diagnostics)
    assert(ran.output.contains("original,alias,alias,alias,alias,original"), ran.diagnostics)

  test("a nonreference command argument refuses at its authored line"):
    val file = source(
      scratch.resolve("refused"),
      """package client.naming
import framework.realize.*
val invalid = everyCase(Instruction.Hold("gate"))
"""
    )
    val ran = tools.scalaCli(
      Seq("compile", file.toString) ++ settings ++
        Seq("--scalac-option", s"-Xplugin:$plugin")
    )
    assertEquals(ran.exit, 1, ran.diagnostics)
    assert(ran.diagnostics.contains("Probe.scala:3:"), ran.diagnostics)
    assert(ran.diagnostics.contains("a command is named after the val"), ran.diagnostics)

  test(
    "authored action and channel identities survive aliases, derived wrappers and forward references"
  ):
    val file = source(
      scratch.resolve("identities"),
      """package client.identities
import framework.dsl.*
import framework.realize.*
import io.temporal.server.api.umpire.v1 as ir
object Subject extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean): Boolean = s
  object rules extends Rules
object Probe:
  val original = action("step", Actor("caller"))
  val alias = original
  val bound = original.on(Entity("entity"))
  val created = original.creates(Entity("entity"))
  val derived = action("typed", Actor("caller")).input[Boolean]("flag").results("result")
  val forward = Later.original
  val tick = timer
  val hidden = internal
  val inbox = channel[Boolean](capacity = 2, order = Order.fifo, loss = Loss.reliable)
  val inboxAlias = inbox
  val hold = Instruction.Hold("gate")
  val root = Realization(machine = Subject, producer = "fixture", producerVersion = "1",
    roles = Vector.empty, correlation = Correlation("p", "r", "o", "v", 8, 8, 2, 16, 1000, 512),
    scripts = Vector(script("controller", Activation.Controller)(
      onPath(alias, forward, inboxAlias.deliver(true), inboxAlias.lose(false), tick, hidden)(hold),
      perform(derived(true) -> hold))),
    controls = Vector(Actuator("delivery", ControlKind.HoldDelivery(inboxAlias))))
  def main(args: Array[String]): Unit =
    assert(original.decl eq alias.decl)
    assert(inbox eq inboxAlias)
    assert(forward.decl eq Later.original.decl)
    assert(bound.decl.on.nonEmpty && created.decl.creates.nonEmpty)
    val ids = Vector("client.identities.Probe.original", "client.identities.Later.original",
      "client.identities.Probe.inbox.deliver", "client.identities.Probe.inbox.lose",
      "client.identities.Probe.tick", "client.identities.Probe.hidden")
    val bool = ir.TypeRef(ir.TypeRef.Ref.Bool(com.google.protobuf.empty.Empty()))
    val parameter = ir.Param("flag", Some(bool))
    val definitions = ids.zipWithIndex.map((id, n) => ir.Action(id = id,
      inputs = if n == 2 || n == 3 then Vector(parameter) else Vector.empty)) :+
      ir.Action(id = "client.identities.Probe.derived", inputs = Vector(parameter))
    val result = root.toIR("client.identities.Probe.root", ir.Model(actions = definitions))
    assert(result.scripts.head.items.head.when.map(_.action) == ids)
    val performance = result.scripts.head.items(1).performs.head.getStep
    assert(performance.action == "client.identities.Probe.derived")
    assert(performance.inputs.head.getBool)
    println("identities retained")
object Later:
  val original = action("forward", Actor("caller"))
"""
    )
    val ran = tools.scalaCli(
      Seq("run", file.toString, "--main-class", "client.identities.Probe") ++
        settings ++ Seq("--scalac-option", s"-Xplugin:$plugin")
    )
    assertEquals(ran.exit, 0, ran.diagnostics)
    assert(ran.output.contains("identities retained"), ran.diagnostics)
    val jar = scratch.resolve("identities.jar")
    val built = tools.scalaCli(
      Seq(
        "compile",
        file.toString,
        "--print-class-path",
        "--compilation-output",
        jar.toString
      ) ++ settings ++ Seq("--scalac-option", s"-Xplugin:$plugin"),
      Output.KeptApart
    )
    built.orFail(): Unit
    val cp = Files.writeString(scratch.resolve("identities.classpath"), built.output.trim)
    val out = scratch.resolve("identities.json")
    val exported = InProcess.lift(
      tools.directory,
      Seq(s"$jar=${file.getParent}/", cp.toString, out.toString, "client.identities.Probe$.root")
    )
    assertEquals(exported.exit, 0, exported.diagnostics)
    val json = new com.fasterxml.jackson.databind.ObjectMapper().readTree(Files.readString(out))
    val realization = json.get("realizations").get(0)
    assertEquals(
      realization.get("controls").get(0).get("holdDelivery").asText(),
      "client.identities.Probe.inbox"
    )
    assertEquals(
      realization
        .get("scripts")
        .get(0)
        .get("items")
        .get(1)
        .get("performs")
        .get(0)
        .get("step")
        .get("action")
        .asText(),
      "client.identities.Probe.derived"
    )

  test("a nonreference argument inside an authored lambda is still refused"):
    val file = source(
      scratch.resolve("lambda-refused"),
      """package client.naming
import framework.realize.*
val invalid = Vector(1).map(_ => everyCase(Instruction.Hold("gate")))
"""
    )
    val ran = tools.scalaCli(
      Seq("compile", file.toString) ++ settings ++
        Seq("--scalac-option", s"-Xplugin:$plugin")
    )
    assertEquals(ran.exit, 1, ran.diagnostics)
    assert(ran.diagnostics.contains("Probe.scala:3:"), ran.diagnostics)
    assert(ran.diagnostics.contains("a command is named after the val"), ran.diagnostics)

  test("a constructed generic realization equals the independent old-host realization"):
    val stored = tools.directory.resolve("model/framework/export/testdata/constructed")
    val fixture = scratch.resolve("oracle")
    Files.createDirectories(fixture): Unit
    val file = Files.copy(stored.resolve("Probe.scala"), fixture.resolve("Probe.scala"))
    val jar = scratch.resolve("oracle.jar")
    val built = tools.scalaCli(
      Seq(
        "compile",
        file.toString,
        "--print-class-path",
        "--compilation-output",
        jar.toString
      ) ++ settings ++ Seq("--scalac-option", s"-Xplugin:$plugin"),
      Output.KeptApart
    )
    built.orFail(): Unit
    val cp = built.output.trim
    val classpath = Files.writeString(scratch.resolve("oracle.classpath"), cp)
    val out = scratch.resolve("oracle.json")
    val ran = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=$fixture/",
        classpath.toString,
        out.toString,
        "fixture.naming.Probe$package$.realization"
      )
    )
    assertEquals(ran.exit, 0, ran.diagnostics)
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    val expected = mapper.readTree(Files.readString(stored.resolve("original.json")))
    def strip(node: com.fasterxml.jackson.databind.JsonNode): Unit =
      import scala.jdk.CollectionConverters.*
      if node.isObject then
        node.asInstanceOf[com.fasterxml.jackson.databind.node.ObjectNode].remove("position"): Unit
      node.elements().asScala.foreach(strip)
    val actual = mapper.readTree(Files.readString(out)).get("realizations").get(0)
    strip(actual)
    val deterministic = Vector("a", "b").map { name =>
      val output = scratch.resolve(s"deterministic-$name.json")
      val args = Seq(
        s"$jar=$fixture/",
        classpath.toString,
        output.toString,
        "fixture.naming.Probe$package$.realization"
      )
      val process = tools.run(
        "java",
        Seq(
          s"-Dfn174.determinism.seed=$name",
          "-cp",
          System.getProperty("java.class.path"),
          "framework.export.lift"
        ) ++ args
      )
      assertEquals(process.exit, 0, process.diagnostics)
      Files.readString(output)
    }
    assertEquals(deterministic(0), deterministic(1))
    assertEquals(actual, expected)

  test(
    "the actual driver constructs each root once in sorted order and publishes nothing after failure"
  ):
    val marker = scratch.resolve("constructions.txt")
    val file = source(
      scratch.resolve("computed"),
      """package client.computed
import framework.dsl.*
import framework.realize.*
object Subject extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean): Boolean = s
  object rules extends Rules
object Roots:
  private def build(name: String): Realization =
    java.nio.file.Files.writeString(java.nio.file.Path.of("MARKER"), name + "\n",
      java.nio.file.StandardOpenOption.CREATE, java.nio.file.StandardOpenOption.APPEND): Unit
    Realization(machine = Subject, producer = "computed", producerVersion = "1",
      roles = Vector.empty, correlation = Correlation("p", "r", "o", "v", 8, 8, 2, 16, 1000, 512),
      scripts = Vector.empty)
  lazy val first = build("first")
  private lazy val second = build("second")
  lazy val exported: Realization = second
  object Nested:
    lazy val third = build("third")
  lazy val failed: Realization = throw IllegalStateException("deliberate native construction failure")
""".replace("MARKER", marker.toString)
    )
    val jar = scratch.resolve("computed.jar")
    val built = tools.scalaCli(
      Seq(
        "compile",
        file.toString,
        "--print-class-path",
        "--compilation-output",
        jar.toString
      ) ++ settings ++ Seq("--scalac-option", s"-Xplugin:$plugin"),
      Output.KeptApart
    )
    built.orFail(): Unit
    val cp = Files.writeString(scratch.resolve("computed.classpath"), built.output.trim)
    val out = scratch.resolve("computed.json")
    val roots = Seq(
      "client.computed.Roots$.second",
      "client.computed.Roots$.first",
      "client.computed.Roots$.Nested$.third",
      "client.computed.Roots$.first"
    )
    val ran = InProcess.lift(
      tools.directory,
      Seq(s"$jar=${file.getParent}/", cp.toString, out.toString) ++ roots
    )
    assertEquals(ran.exit, 0, ran.diagnostics)
    assertEquals(Files.readString(marker), "third\nfirst\nsecond\n")
    val failed = scratch.resolve("failed.json")
    val refused = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=${file.getParent}/",
        cp.toString,
        failed.toString,
        "client.computed.Roots$.first",
        "client.computed.Roots$.failed"
      )
    )
    assertEquals(refused.exit, 1, refused.diagnostics)
    assert(
      refused.diagnostics.contains("while constructing root client.computed.Roots$.failed"),
      refused.diagnostics
    )
    assert(
      refused.diagnostics.contains("deliberate native construction failure"),
      refused.diagnostics
    )
    assert(!Files.exists(failed))

  test(
    "actual evidence construction accepts machine facts and refuses foreign cases with the same spelling"
  ):
    val stored = tools.directory.resolve("model/framework/export/testdata/constructed/Probe.scala")
    val file = source(
      scratch.resolve("facts"),
      Files.readString(stored) + """
val recordedSource = Recorded.runEvent[com.google.protobuf.duration.Duration](
  EventKind.diagnostic, "controller", "original", Operand.runKey())
val retainedFact = Evidence.runEvent("record", Fact.recorded, "fixture", recordedSource, Commitment.durable)
val factRealization = realization.copy(evidence = Vector(retainedFact))
enum ForeignFact:
  case recorded
val foreignFact = Evidence.runEvent("foreign", ForeignFact.recorded, "fixture", recordedSource, Commitment.durable)
val foreignRealization = realization.copy(evidence = Vector(foreignFact))
"""
    )
    val jar = scratch.resolve("facts.jar")
    val built = tools.scalaCli(
      Seq(
        "compile",
        file.toString,
        "--print-class-path",
        "--compilation-output",
        jar.toString
      ) ++ settings ++ Seq("--scalac-option", s"-Xplugin:$plugin"),
      Output.KeptApart
    )
    built.orFail(): Unit
    val cp = Files.writeString(scratch.resolve("facts.classpath"), built.output.trim)
    val out = scratch.resolve("facts.json")
    val kept = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=${file.getParent}/",
        cp.toString,
        out.toString,
        "fixture.naming.Probe$package$.factRealization"
      )
    )
    assertEquals(kept.exit, 0, kept.diagnostics)
    val json = new com.fasterxml.jackson.databind.ObjectMapper().readTree(Files.readString(out))
    assertEquals(
      json.get("realizations").get(0).get("evidence").get(0).get("records").asText(),
      "recorded"
    )
    val failed = scratch.resolve("foreign-facts.json")
    val refused = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=${file.getParent}/",
        cp.toString,
        failed.toString,
        "fixture.naming.Probe$package$.foreignRealization"
      )
    )
    assertEquals(refused.exit, 1, refused.diagnostics)
    assert(refused.diagnostics.contains("native machine fact catalog"), refused.diagnostics)
    assert(refused.diagnostics.contains("fixture.naming.foreignRealization"), refused.diagnostics)
    assert(!Files.exists(failed))
