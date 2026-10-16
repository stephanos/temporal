package framework.`export`

import java.nio.file.{Files, Path}
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import com.fasterxml.jackson.databind.JsonNode
import framework.check.{Output, RetainedScratch, Tools}

class ConstructedRealizationsSuite extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes

  private val tools = Tools.here
  private val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
  private lazy val scratch =
    retained(tools.directory.resolve(".flow/tmp"), "constructed-realizations.")

  private def source(dir: Path, body: String): Path =
    Files.createDirectories(dir): Unit
    Files.writeString(dir.resolve("Probe.scala"), body)

  private def namedSource(dir: Path, name: String, body: String): Path =
    Files.createDirectories(dir): Unit
    Files.writeString(dir.resolve(name), body)

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

  private def settings(jars: Seq[Path], reify: Boolean = true): Seq[String] =
    Seq(
      "--scala",
      "3.9.0",
      "--jvm",
      "27",
      "--server=false",
      "--dependency",
      "com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20"
    ) ++ jars.flatMap(jar => Seq("--jar", jar.toString)) ++
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
      ).flatMap(option => Seq("--scalac-option", option)) ++
      (if reify then
         Seq(
           "--scalac-option",
           s"-Xplugin:$plugin",
           "--scalac-option",
           "-P:umpire-reify:mode=realization"
         )
       else Seq.empty)

  final private case class Fixture(
      jar: Path,
      classpath: Path,
      source: Path,
      marker: Path
  )

  private lazy val protocolFixture =
    val marker = scratch.resolve("protocol-events.txt")
    val file = source(
      scratch.resolve("protocol"),
      """package fixture.constructedprotocol
        |import framework.dsl.*
        |import framework.realize.{ConstructedRealization, ProtocolProbe}
        |
        |object Subject extends Machine[Boolean, Boolean, Nothing]:
        |  val init = false
        |  def end(state: Boolean): Boolean = state
        |  object rules extends Rules
        |  object properties:
        |    val done = property expect(check(after.state))
        |  object queries:
        |    val any = scenario.free
        |    val done =
        |      (query find properties.done in any limits Limits("done", 1, 1, 8))
        |
        |val external = action(Actor("fixture"))
        |
        |private def mark(event: String): Unit =
        |  java.nio.file.Files.writeString(
        |    java.nio.file.Path.of("MARKER"),
        |    event + "\n",
        |    java.nio.file.StandardOpenOption.CREATE,
        |    java.nio.file.StandardOpenOption.APPEND
        |  ): Unit
        |
        |object Roots:
        |  private def build(
        |      label: String,
        |      cleanup: String = "",
        |      requiredAction: Option[String] = None,
        |      selected: Seq[AnyRef] = Seq.empty
        |  ): ConstructedRealization =
        |    mark("construct:" + label)
        |    ProtocolProbe(label, cleanup, "MARKER", requiredAction, selected)
        |
        |  lazy val first = build(
        |    "first",
        |    requiredAction = Some("fixture.constructedprotocol.external"),
        |    selected = Seq(Subject.queries.done)
        |  )
        |  lazy val second = build("second")
        |  lazy val stable = build("stable", "fixed")
        |  lazy val seeded =
        |    val random = scala.util.Random(java.lang.Long.getLong("fn174.seed", 0L))
        |    build("seeded", Vector.fill(8)(random.nextLong()).mkString(","))
        |  lazy val zzConversion: ConstructedRealization =
        |    mark("construct:zz-conversion")
        |    ProtocolProbe("zz-conversion", "", "MARKER", failConversion = true)
        |  lazy val zzFailure: ConstructedRealization =
        |    mark("construct:zz-failure")
        |    throw IllegalStateException("deliberate root failure")
        |  val unsupported = "not a realization"
        |""".stripMargin.replace("MARKER", marker.toString)
    )
    val converter = namedSource(
      file.getParent,
      "ProtocolRealization.scala",
      """package framework.realize
        |import fixture.constructedprotocol.Subject
        |import io.temporal.server.api.umpire.v1 as ir
        |
        |final class ProtocolProbe(
        |    label: String,
        |    cleanup: String,
        |    marker: String,
        |    requiredAction: Option[String] = None,
        |    selected: Seq[AnyRef] = Seq.empty,
        |    failConversion: Boolean = false
        |)
        |    extends ConstructedRealization:
        |  val machine = Subject
        |  val runs: Seq[AnyRef] = selected
        |  val dependencies = requiredAction match
        |    case Some(identity) => ConstructedRealization.Dependencies(
        |      Vector(ConstructedRealization.ActionDependency.Declaration(identity)),
        |      Vector.empty
        |    )
        |    case None => ConstructedRealization.Dependencies.empty
        |
        |  def toIR(context: ConstructedRealization.Context): ir.Realization =
        |    java.nio.file.Files.writeString(
        |      java.nio.file.Path.of(marker),
        |      "convert:" + label + "\n",
        |      java.nio.file.StandardOpenOption.CREATE,
        |      java.nio.file.StandardOpenOption.APPEND
        |    ): Unit
        |    if failConversion then throw IllegalStateException("deliberate conversion failure")
        |    require(
        |      context.sourcePackage.contains("fixture.constructedprotocol"),
        |      "declaring package was not supplied"
        |    )
        |    val metadataStatus =
        |      if context.metadata.machines.exists(_.name == machine.name) &&
        |          requiredAction.forall(identity => context.metadata.actions.exists(_.id == identity))
        |      then "metadata-ready"
        |      else "metadata-missing"
        |    ir.Realization(
        |      id = context.declaration,
        |      name = label,
        |      machine = machine.name,
        |      producer = metadataStatus,
        |      producerVersion = context.sourcePackage.get,
        |      cleanup = cleanup
        |    )
        |""".stripMargin
    )
    val contextual = namedSource(
      file.getParent,
      "Contextual.scala",
      """package framework.realize
        |import fixture.constructedprotocol.Subject
        |import io.temporal.server.api.umpire.v1 as ir
        |
        |object ContextualRoot extends ConstructedRealization:
        |  val machine = Subject
        |  val runs: Seq[AnyRef] = Seq.empty
        |  val dependencies = ConstructedRealization.Dependencies.empty
        |  def toIR(context: ConstructedRealization.Context): ir.Realization =
        |    val unresolved = String.valueOf(0.toChar) + "umpire-family-context:v1" + String.valueOf(0.toChar)
        |    ir.Realization(
        |      id = context.declaration,
        |      name = "contextual",
        |      machine = machine.name,
        |      producer = unresolved,
        |      evidence = Seq(
        |        ir.Evidence(
        |          id = unresolved + ".evidence.synthetic",
        |          records = "synthetic-record",
        |          source = unresolved + ".source.synthetic",
        |          fields = Seq(
        |            ir.EvidenceField(
        |              id = unresolved + ".field.operation",
        |              path = "payload.operation"
        |            )
        |          )
        |        )
        |      )
        |    )
        |""".stripMargin
    )
    val jar = scratch.resolve("protocol.jar")
    val built = tools.scalaCli(
      Seq(
        "compile",
        file.toString,
        converter.toString,
        contextual.toString,
        "--print-class-path",
        "--compilation-output",
        jar.toString
      ) ++
        settings(
          Seq(library, tools.directory.resolve("model/build/ir-scalapb.jar"))
        ),
      Output.KeptApart
    )
    built.orFail(): Unit
    Fixture(
      jar,
      Files
        .writeString(scratch.resolve("protocol.classpath"), built.output.trim),
      file,
      marker
    )

  private def arguments(
      fixture: Fixture,
      out: Path,
      roots: String*
  ): Seq[String] =
    Seq(
      s"${fixture.jar}=${fixture.source.getParent}/",
      fixture.classpath.toString,
      out.toString
    ) ++
      roots.map { root =>
        if root.startsWith("fixture.") then root
        else s"fixture.constructedprotocol.Roots$$.$root"
      }

  private def count(text: String, part: String): Int =
    text.sliding(part.length).count(_ == part)

  private def child(
      fixture: Fixture,
      out: Path,
      root: String,
      properties: Seq[String]
  ) =
    tools.run(
      "java",
      properties ++ Seq(
        "-cp",
        System.getProperty("java.class.path"),
        "framework.export.lift"
      ) ++ arguments(fixture, out, root)
    )

  final private case class Difference(line: Int, left: String, right: String)

  private def firstDifference(left: String, right: String): Option[Difference] =
    left.linesIterator
      .zipAll(right.linesIterator, "<missing>", "<missing>")
      .zipWithIndex
      .collectFirst {
        case ((a, b), index) if a != b => Difference(index + 1, a, b)
      }

  final private case class JsonDifference(
      path: String,
      expected: String,
      obtained: String
  )

  private def firstJsonDifference(
      expected: JsonNode,
      obtained: JsonNode,
      path: String = "$"
  ): Option[JsonDifference] =
    if expected.equals(obtained) then None
    else if expected.isObject && obtained.isObject then
      (expected.fieldNames().asScala.toSet ++ obtained
        .fieldNames()
        .asScala).toVector.sorted.iterator
        .flatMap(name =>
          firstJsonDifference(
            expected.path(name),
            obtained.path(name),
            s"$path.$name"
          )
        )
        .nextOption()
    else if expected.isArray && obtained.isArray then
      (0 until math.max(expected.size(), obtained.size())).iterator
        .flatMap(index =>
          firstJsonDifference(
            expected.path(index),
            obtained.path(index),
            s"$path[$index]"
          )
        )
        .nextOption()
    else Some(JsonDifference(path, expected.toString, obtained.toString))

  private def withoutPositions(value: JsonNode): JsonNode =
    if value.isObject then
      val result = mapper.createObjectNode()
      value
        .fields()
        .asScala
        .foreach: entry =>
          if entry.getKey != "position" then
            result.set[JsonNode](
              entry.getKey,
              withoutPositions(entry.getValue)
            ): Unit
      result
    else if value.isArray then
      val result = mapper.createArrayNode()
      value
        .elements()
        .asScala
        .foreach(item => result.add(withoutPositions(item)): Unit)
      result
    else value

  private def sha256(path: Path): String =
    java.util.HexFormat
      .of()
      .formatHex(
        java.security.MessageDigest
          .getInstance("SHA-256")
          .digest(Files.readAllBytes(path))
      )

  test(
    "action dependencies retain declarations, bindings, channels, and reject absent identity"
  ):
    val file = source(
      scratch.resolve("action-dependencies"),
      """package framework.realize
        |import framework.dsl.*
        |import framework.realize.ConstructedRealization.ActionDependency
        |import io.temporal.server.api.umpire.v1 as ir
        |
        |object DependencySubject extends Machine[Boolean, Boolean, Nothing]:
        |  val init = false
        |  def end(state: Boolean): Boolean = state
        |  object rules extends Rules
        |
        |final class DependencyHarness extends ConstructedRealization:
        |  val machine = DependencySubject
        |  val runs: Seq[AnyRef] = Seq.empty
        |  val dependencies = ConstructedRealization.Dependencies.empty
        |  def toIR(context: ConstructedRealization.Context): ir.Realization = ir.Realization()
        |  def dependency(value: ClassRef | Action[?]): ActionDependency = actionDependency(value)
        |
        |object DependencyProbe:
        |  private def declaration(value: ActionDependency): String = value match
        |    case ActionDependency.Declaration(identity) => identity
        |    case ActionDependency.Channel(identity, operation) =>
        |      throw IllegalStateException("unexpected channel " + identity + "." + operation)
        |
        |  def main(args: Array[String]): Unit =
        |    val harness = DependencyHarness()
        |    val original = action("step", Actor("fixture"))
        |    original.declaredAt("fixture.original"): Unit
        |    assert(declaration(harness.dependency(original())) == "fixture.original")
        |
        |    val bound = original.on(Entity("entity"))
        |    bound.sourceAt("fixture.bound"): Unit
        |    assert(declaration(harness.dependency(bound)) == "fixture.bound")
        |
        |    val inbox = channel[Boolean](1, Order.fifo, Loss.reliable)
        |    inbox.declaredAt("fixture.inbox"): Unit
        |    harness.dependency(inbox.deliver) match
        |      case ActionDependency.Channel(identity, operation) =>
        |        assert(identity == "fixture.inbox")
        |        assert(operation == "deliver")
        |      case other => throw IllegalStateException("unexpected dependency " + other)
        |
        |    val missing = action("missing", Actor("fixture"))
        |    val refused = try
        |      harness.dependency(missing): Unit
        |      throw IllegalStateException("missing action identity was accepted")
        |    catch case error: IllegalArgumentException => error
        |    assert(refused.getMessage.contains("has no retained source identity"), refused.getMessage)
        |""".stripMargin
    )
    val jar = scratch.resolve("action-dependencies.jar")
    val built = tools.scalaCli(
      Seq(
        "compile",
        file.toString,
        "--print-class-path",
        "--compilation-output",
        jar.toString
      ) ++
        settings(
          Seq(library, tools.directory.resolve("model/build/ir-scalapb.jar")),
          reify = false
        ),
      Output.KeptApart
    )
    built.orFail(): Unit
    val ran = tools.run(
      "java",
      Seq("-cp", built.output.trim, "framework.realize.DependencyProbe")
    )
    assertEquals(ran.exit, 0, ran.diagnostics)

  test(
    "constructed roots are read once in sorted order and failures publish no partial model"
  ):
    val fixture = protocolFixture
    val out = scratch.resolve("protocol.json")
    val kept = InProcess.lift(
      tools.directory,
      arguments(
        fixture,
        out,
        "second",
        "first",
        "first",
        "fixture.constructedprotocol.Subject$.queries"
      )
    )
    assertEquals(kept.exit, 0, kept.diagnostics)
    val events = Files.readAllLines(fixture.marker).asScala.toVector
    assertEquals(
      events.filter(_.startsWith("construct:")),
      Vector("construct:first", "construct:second")
    )
    assertEquals(
      events.filter(_.startsWith("convert:")),
      Vector("convert:first", "convert:second")
    )
    val model = mapper.readTree(Files.readString(out))
    assertEquals(
      model
        .path("realizations")
        .elements()
        .asScala
        .map(_.path("name").asText())
        .toVector,
      Vector("first", "second")
    )
    assertEquals(
      model
        .path("realizations")
        .elements()
        .asScala
        .map(_.path("producer").asText())
        .toSet,
      Set("metadata-ready")
    )
    assertEquals(
      model
        .path("realizations")
        .elements()
        .asScala
        .map(
          _.path("queries").elements().asScala.map(_.asText()).toVector
        )
        .toVector,
      Vector(Vector("done"), Vector.empty)
    )
    assertEquals(
      model
        .path("queries")
        .elements()
        .asScala
        .map(_.path("name").asText())
        .toVector,
      Vector("done")
    )

    Files.delete(fixture.marker)
    val failed = scratch.resolve("root-failure.json")
    val refused = InProcess.lift(
      tools.directory,
      arguments(
        fixture,
        failed,
        "first",
        "zzFailure",
        "fixture.constructedprotocol.Subject$.queries"
      )
    )
    assertEquals(refused.exit, 1, refused.diagnostics)
    assertEquals(
      Files.readAllLines(fixture.marker).asScala.toVector,
      Vector("construct:first", "construct:zz-failure")
    )
    assertEquals(
      count(refused.diagnostics, "while constructing root"),
      1,
      refused.diagnostics
    )
    assert(
      refused.diagnostics.contains("deliberate root failure"),
      refused.diagnostics
    )
    assert(!Files.exists(failed))

    Files.delete(fixture.marker)
    val conversion = scratch.resolve("conversion-failure.json")
    val conversionRefused = InProcess.lift(
      tools.directory,
      arguments(
        fixture,
        conversion,
        "first",
        "zzConversion",
        "fixture.constructedprotocol.Subject$.queries"
      )
    )
    assertEquals(conversionRefused.exit, 1, conversionRefused.diagnostics)
    assertEquals(
      Files.readAllLines(fixture.marker).asScala.toVector,
      Vector(
        "construct:first",
        "construct:zz-conversion",
        "convert:first",
        "convert:zz-conversion"
      )
    )
    assertEquals(
      count(conversionRefused.diagnostics, "deliberate conversion failure"),
      1
    )
    assertEquals(
      count(conversionRefused.diagnostics, "while constructing root"),
      1,
      conversionRefused.diagnostics
    )
    assert(!Files.exists(conversion))

    val unsupported = scratch.resolve("unsupported-root.json")
    val unsupportedResult = InProcess.lift(
      tools.directory,
      arguments(fixture, unsupported, "unsupported")
    )
    assertEquals(unsupportedResult.exit, 1, unsupportedResult.diagnostics)
    assert(
      unsupportedResult.diagnostics.contains(
        "is a java.lang.String; a root is"
      ),
      unsupportedResult.diagnostics
    )
    assert(!Files.exists(unsupported))

    val contextual = scratch.resolve("contextual-root.json")
    val contextualResult = InProcess.lift(
      tools.directory,
      Seq(
        s"${fixture.jar}=${fixture.source.getParent}/",
        fixture.classpath.toString,
        contextual.toString,
        "framework.realize.ContextualRoot"
      )
    )
    assertEquals(contextualResult.exit, 0, contextualResult.diagnostics)
    val contextualJson = Files.readString(contextual)
    assert(!contextualJson.contains("umpire-family-"), contextualJson)
    val resolved = mapper.readTree(contextualJson).path("realizations").path(0)
    assertEquals(resolved.path("producer").asText(), "framework.realize")
    val evidence = resolved.path("evidence").path(0)
    assertEquals(
      evidence.path("id").asText(),
      "framework.realize.evidence.synthetic"
    )
    assertEquals(evidence.path("records").asText(), "synthetic-record")
    assertEquals(
      evidence.path("source").asText(),
      "framework.realize.source.synthetic"
    )
    assertEquals(
      evidence.path("fields").path(0).path("id").asText(),
      "framework.realize.field.operation"
    )
    assertEquals(
      evidence.path("fields").path(0).path("path").asText(),
      "payload.operation"
    )

  test(
    "two JVM exports are identical and the seeded sensitivity control names its first difference"
  ):
    val fixture = protocolFixture
    def exported(root: String, name: String, seed: Long): String =
      val out = scratch.resolve(s"$root-$name.json")
      val ran = child(fixture, out, root, Seq(s"-Dfn174.seed=$seed"))
      assertEquals(ran.exit, 0, ran.diagnostics)
      Files.readString(out)

    val stableA = exported("stable", "a", 11)
    val stableB = exported("stable", "b", 29)
    assertEquals(firstDifference(stableA, stableB), None)

    val seededA = exported("seeded", "a", 11)
    val seededB = exported("seeded", "b", 29)
    val detected =
      firstDifference(seededA, seededB).getOrElse(
        fail("seeded disorder was not detected")
      )
    assert(
      detected.left.contains("cleanup") || detected.right.contains("cleanup"),
      detected.toString
    )

    val stableAfter = exported("stable", "after", 47)
    assertEquals(firstDifference(stableA, stableAfter), None)

  private def miniExport(name: String): String =
    val out = scratch.resolve(s"mini-$name")
    val ran = tools.run(
      "java",
      Seq(
        "-cp",
        System.getProperty("java.class.path"),
        "framework.export.lift",
        "--ir",
        ModelBuild.jarArgument,
        ModelBuild.classpath.toString,
        out.toString,
        "activity-mini"
      )
    )
    assertEquals(ran.exit, 0, ran.diagnostics)
    Files.readString(out.resolve("activity-mini.json"))

  test(
    FrameworkIntegration(
      "the real constructed activity mini matches its frozen oracle in two JVMs",
      "real activity-mini requires the held system Model and its realization kit"
    )
  ):
    val first = miniExport("first")
    val second = miniExport("second")
    assertEquals(firstDifference(first, second), None)
    assert(!first.contains("umpire-family-"), first)

    val model = mapper.readTree(first)
    val declared = model.path("realizations").elements().asScala.toVector
    assertEquals(declared.size, 1)
    val realization = declared.head
    assertEquals(
      realization.path("queries").elements().asScala.map(_.asText()).toVector,
      Vector("protocolCompleted", "retryCompleted")
    )
    // Old-host capture: model/build/history/lifter.9134648190401109745/coreDiscovery-out/
    // activity-mini.json at 25ffe44e08, sha256 201ac3d630af655671cb648c5e675ade296fcf998b30a3f2e52149a00321a570.
    // `jq -S '.realizations[0] | walk(if type=="object" then del(.position) else . end) |
    // del(.queries)'` produced this conversion-owned oracle, sha256
    // 199c89ac343fd64f63c93b2d015cf2995a90ffe6abc124493f9b905c18691b5e.
    val oracle = tools.directory.resolve(
      "model/framework/export/testdata/constructedRealizations/activity-mini-old.json"
    )
    assertEquals(Files.size(oracle), 50156L)
    assertEquals(
      sha256(oracle),
      "199c89ac343fd64f63c93b2d015cf2995a90ffe6abc124493f9b905c18691b5e"
    )
    val expected = mapper.readTree(Files.readString(oracle))
    val obtainedObject =
      realization.deepCopy[com.fasterxml.jackson.databind.node.ObjectNode]()
    obtainedObject.remove("queries"): Unit
    val obtained = withoutPositions(obtainedObject)

    assertEquals(firstJsonDifference(expected, obtained), None)

  test(
    FrameworkIntegration(
      "an opaque Temporal request selector refuses the constructed root and writes nothing",
      "request selector needs the held system realization kit and API jar"
    )
  ):
    val file = source(
      scratch.resolve("temporal-refusal"),
      """package fixture.constructedtemporal
        |import framework.dsl.*
        |import framework.realize.*
        |import temporal.realize.*
        |import io.temporal.api.workflowservice.v1.StartActivityExecutionRequest
        |import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.*
        |
        |object Subject extends Machine[Boolean, Boolean, Nothing]:
        |  val init = false
        |  def end(state: Boolean): Boolean = state
        |  object rules extends Rules
        |
        |private val selectNamespace: StartActivityExecutionRequest => String = _.namespace
        |private val opaque = rpc(workflowService, METHOD_START_ACTIVITY_EXECUTION) {
        |  field(selectNamespace) := Operand.text("opaque")
        |}
        |
        |object OpaqueRun extends Realizes(Subject):
        |  object controller extends Controller(everyCase(opaque))
        |""".stripMargin
    )
    val jar = scratch.resolve("temporal-refusal.jar")
    val built = tools.scalaCli(
      Seq(
        "compile",
        file.toString,
        "--print-class-path",
        "--compilation-output",
        jar.toString
      ) ++
        settings(
          Seq(
            ModelBuild.jar,
            ModelBuild.directory.resolve("api-scalapb.jar"),
            tools.directory.resolve("model/build/ir-scalapb.jar")
          ),
          reify = true
        ),
      Output.KeptApart
    )
    built.orFail(): Unit
    val classpath =
      Files.writeString(
        scratch.resolve("temporal-refusal.classpath"),
        built.output.trim
      )
    val out = scratch.resolve("temporal-refusal.json")
    val refused = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=${file.getParent}/,${ModelBuild.jarArgument}",
        classpath.toString,
        out.toString,
        "fixture.constructedtemporal.OpaqueRun"
      )
    )
    assertEquals(refused.exit, 1, refused.diagnostics)
    assert(
      refused.diagnostics.contains("fixture.constructedtemporal.OpaqueRun"),
      refused.diagnostics
    )
    assert(refused.diagnostics.contains("Probe.scala:"), refused.diagnostics)
    assert(refused.diagnostics.contains("selector"), refused.diagnostics)
    assert(!Files.exists(out))
