package framework.`export`

import java.nio.file.{Files, Path}
import scala.concurrent.duration.*
import framework.check.{Output, RetainedScratch, Tools}

class ConstructedNamesSuite extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private lazy val scratch =
    retained(tools.directory.resolve(".flow/tmp"), "constructed-names.")
  private def source(dir: Path, body: String): Path =
    Files.createDirectories(dir): Unit
    Files.writeString(dir.resolve("Probe.scala"), body)

  private def namedSource(dir: Path, name: String, body: String): Path =
    Files.createDirectories(dir): Unit
    Files.writeString(dir.resolve(name), body)

  private def strip(node: com.fasterxml.jackson.databind.JsonNode): Unit =
    import scala.jdk.CollectionConverters.*
    if node.isObject then
      node
        .asInstanceOf[com.fasterxml.jackson.databind.node.ObjectNode]
        .remove("position"): Unit
    node.elements().asScala.foreach(strip)

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

  private lazy val pluginFramework =
    val jar = scratch.resolve("framework-with-reify.jar")
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
          jar.toString,
          "--scalac-option",
          s"-Xplugin:$plugin"
        )
      )
      .orFail(): Unit
    jar

  private def settingsWith(framework: Path) = Seq(
    "--scala",
    "3.9.0",
    "--jvm",
    "27",
    "--server=false",
    "--dependency",
    "com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20",
    "--jar",
    framework.toString,
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

  private def settings = settingsWith(library)

  private val probe = """package client.naming
import framework.realize.*
import com.google.protobuf.duration.Duration
object Probe:
  private given CanEqual[Instruction, Instruction] = CanEqual.derived
  val original: Instruction = Instruction.Hold("gate")
  val configured = command(original)
  val alias: Instruction = original
  def scripted(call: Instruction): Script = script("controller", Activation.Controller)(everyCase(call))
  def twice(call: Instruction): Script = scripted(call)
  def changed(call: Instruction.Hold): String = call.copy(control = "changed").ref.get.spelling
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
  val held = Instruction.Hold("gate")
  val heldAlias = held
  def main(args: Array[String]): Unit =
    assert(original eq alias)
    assert(original == alias)
    assert(configured.toIR("client.naming.realization").id == "configured")
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
    val ids = values.map(_.items.head.command.get).map(_.toIR("client.naming.realization").id)
    assert(ids == Vector("original", "alias", "alias", "alias", "alias", "original"), ids.toString)
    assert(many(original, alias).map(_.command.get).map(_.toIR("client.naming.realization").id) == Vector("original", "alias"))
    assert(original.ref.isEmpty)
    val rpcIds = Vector(everyCase(variant), everyCase(extendedCall)).map(_.command.get).map(_.toIR("client.naming.realization").id)
    assert(rpcIds == Vector("original-call", "extended-call"), rpcIds.toString)
    assert(variant.assign.size == 1 && extendedCall.assign.size == 1)
    assert(variant.method eq originalCall.method)
    assert(evaluations == 1)
    assert(changed(heldAlias) == "heldAlias")
    println(ids.mkString(","))
"""

  private lazy val familyHelpers =
    val dir = scratch.resolve("family-helpers")
    source(
      dir.resolve("authored"),
      """package fixture.shared
import framework.realize.family
object Helpers:
  val shared = family + ".shared"
  def inner(prefix: String): String = prefix + ".inner"
  def outer(prefix: String): String = inner(prefix)
  def owned(): String = family + ".owned"
  def inspect(prefix: String): Boolean = prefix.isEmpty
  def inspectOuter(prefix: String): Boolean = inspect(prefix)
  def inspectCurried(@scala.annotation.unused ignored: String)(prefix: String): Boolean = prefix.isEmpty
  def innerCurried(prefix: String)(suffix: String): String = prefix + suffix
  def curried(prefix: String)(suffix: String): String = innerCurried(prefix)(suffix)
  @framework.dsl.lifted def body(@scala.annotation.unused flag: Boolean): String = family
  def inspectPattern(prefix: String): Boolean = prefix match
    case alias => alias.isEmpty
"""
    ): Unit
    source(
      dir.resolve("vocabulary"),
      """package framework.realize
val familyShared = family + ".vocabulary"
def familyOwned(): String = familyShared
"""
    ): Unit
    val jar = scratch.resolve("family-helpers.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          dir.toString,
          "--library",
          "-f",
          "-o",
          jar.toString
        ) ++
          settings ++ Seq("--scalac-option", s"-Xplugin:$plugin")
      )
      .orFail(): Unit
    jar

  test(
    "the family phase accepts only the exact framework and generated dependency carriers"
  ):
    val wrappers = source(
      scratch.resolve("family-production-carriers").resolve("framework"),
      """package framework.realize
val raw = new Evidence(
  "id",
  null.asInstanceOf[Fact],
  family,
  null.asInstanceOf[Recorded],
  null.asInstanceOf[Commitment]
)
val reference = new EvidenceRef[Any, Any](
  raw,
  null.asInstanceOf[Field[Any, ?]],
  Vector.empty
)
val typed = new TypedEvidence[Any](raw, None, Vector.empty)
"""
    )
    val generated = source(
      scratch.resolve("family-production-carriers").resolve("generated"),
      """package fixture.generated
import framework.realize.family
import temporal.server.api.testpilot.v1.InstructionOutcome
def contextual(using suffix: String): String = family + suffix
def status(prefix: String): String =
  prefix + InstructionOutcome.defaultInstance.status.toString
val kept = status(family)
val keptContext = contextual(using ".context")
"""
    )
    val compiled = tools.scalaCli(
      Seq("compile", wrappers.toString, generated.toString) ++
        settingsWith(pluginFramework) ++
        Seq(
          "--jar",
          tools.directory.resolve("model/build/api-scalapb.jar").toString,
          "--scalac-option",
          s"-Xplugin:$plugin"
        )
    )
    assertEquals(compiled.exit, 0, compiled.diagnostics)
    val apiSpoof = source(
      scratch.resolve("family-production-carriers").resolve("api-spoof"),
      """package temporal.server.api.testpilot.v1
object OpaqueGenerated:
  def apply(value: String): String = value
"""
    )
    val apiSpoofJar = scratch.resolve("api-spoof.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          apiSpoof.toString,
          "--library",
          "-f",
          "-o",
          apiSpoofJar.toString
        ) ++
          settingsWith(pluginFramework)
      )
      .orFail(): Unit
    val apiSpoofCaller = source(
      scratch.resolve("family-production-carriers").resolve("api-spoof-caller"),
      """package fixture.generated
import framework.realize.family
val kept = temporal.server.api.testpilot.v1.OpaqueGenerated(family)
"""
    )
    val apiSpoofResult = tools.scalaCli(
      Seq("compile", apiSpoofCaller.toString) ++ settingsWith(
        pluginFramework
      ) ++
        Seq(
          "--jar",
          tools.directory.resolve("model/build/api-scalapb.jar").toString,
          "--jar",
          apiSpoofJar.toString,
          "--scalac-option",
          s"-Xplugin:$plugin"
        )
    )
    assertEquals(apiSpoofResult.exit, 1, apiSpoofResult.diagnostics)
    assert(
      apiSpoofResult.diagnostics.contains(
        "OpaqueGenerated$.apply has no family forwarding contract"
      ),
      apiSpoofResult.diagnostics
    )
    val inspections = Seq(
      (
        "reference",
        """def inspect(value: EvidenceRef[?, ?]): Boolean = value.evidence.source.isEmpty
val invalid = inspect(reference)
""",
        "family cannot be inspected"
      ),
      (
        "typed",
        """def inspect(value: TypedEvidence[?]): Boolean = value.evidence.source.isEmpty
val invalid = inspect(typed)
""",
        "family cannot be inspected"
      ),
      (
        "context-function",
        """val contextual: String ?=> String = family + summon[String]
val invalid = contextual(using ".context").isEmpty
""",
        "scala.ContextFunction1.apply has no family forwarding contract"
      )
    )
    inspections.foreach { (name, body, refusal) =>
      val inspected = source(
        scratch
          .resolve("family-production-carriers")
          .resolve(s"inspected-$name"),
        s"""package fixture.inspected
import framework.realize.*
$body"""
      )
      val refused = tools.scalaCli(
        Seq("compile", wrappers.toString, inspected.toString) ++
          settingsWith(pluginFramework) ++ Seq(
            "--scalac-option",
            s"-Xplugin:$plugin"
          )
      )
      assertEquals(refused.exit, 1, refused.diagnostics)
      assert(
        refused.diagnostics.contains("Probe.scala:4:"),
        refused.diagnostics
      )
      assert(
        refused.diagnostics.contains(refusal),
        refused.diagnostics
      )
      assert(
        !refused.diagnostics.contains("family constructor contract"),
        refused.diagnostics
      )
      assert(
        !refused.diagnostics.contains("no validated model helper contract"),
        refused.diagnostics
      )
    }

  test("an ordinary authored Finite val retains its schema metadata"):
    val client = source(
      scratch.resolve("authored-schema-client"),
      """package fixture.schemaclient
import framework.dsl.*
enum Mode:
  case ready, stopped
final case class State(mode: Mode)
object Consumer:
  def retain(raw: Finite[State]): Any =
    val authored: Finite[State] = raw
    authored
"""
    )
    val clientJar = scratch.resolve("authored-schema-client.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          client.toString,
          "--library",
          "-f",
          "-o",
          clientJar.toString
        ) ++ settings ++ Seq("--scalac-option", s"-Xplugin:$plugin")
      )
      .orFail(): Unit
    val runner = source(
      scratch.resolve("authored-schema-runner"),
      """package fixture.schemaclient
import framework.dsl.*
object Probe:
  def main(args: Array[String]): Unit =
    val raw: Finite[State] = new Finite[State]:
      def values: IndexedSeq[State] = Finite.derived[State].values
    assert(raw.schema.isEmpty)
    val retained = Consumer.retain(raw).asInstanceOf[Finite[State]]
    assert(retained eq raw)
    assert(retained.schema.get.identity == "fixture.schemaclient.State")
    assert(retained.schema.get.root.getNamed == "fixture.schemaclient.State")
"""
    )
    val ran = tools.scalaCli(
      Seq(
        "run",
        runner.toString,
        "--main-class",
        "fixture.schemaclient.Probe"
      ) ++
        settings ++ Seq("--jar", clientJar.toString)
    )
    assertEquals(ran.exit, 0, ran.diagnostics)

  test("realization mode names only direct Entity declarations"):
    val client = source(
      scratch.resolve("realization-entity-names"),
      """package framework.fixture
import framework.dsl.*
val implicitName = Entity(key = "id")
val explicitName = Entity("given", key = "id")
val explicitEmpty = new Entity("", "empty-id")
val referred = Entity(key = "referred-id", refer = Map("parent" -> implicitName))
val aliasName = implicitName
val copiedName = implicitName.copy(key = "copy-id")
val copiedEmpty = implicitName.copy(name = "")
val unrelatedLimits = Limits("", 1, 1, 8)
val unrelatedObservation = Observation(on = implicitName)
val unrelatedAssumption = assume
val unrelatedHole = hole
val unrelatedAction = action(Actor("actor"))
val unrelatedChannel = channel[Boolean](1, Order.fifo, Loss.reliable)
object Probe:
  def main(arguments: Array[String]): Unit =
    assert(implicitName.sourceName == "implicitName")
    assert(explicitName.sourceName == "given")
    assert(explicitEmpty.sourceName == "")
    assert(referred.sourceName == "referred")
    assert(aliasName.sourceName == "implicitName")
    assert(copiedName.sourceName == "implicitName")
    assert(copiedEmpty.sourceName == "")
    val missing = Seq(
      scala.util.Try(unrelatedLimits.sourceName),
      scala.util.Try(unrelatedObservation.sourceName),
      scala.util.Try(unrelatedAssumption.sourceName),
      scala.util.Try(unrelatedHole.sourceName),
      scala.util.Try(unrelatedAction.toIR),
      scala.util.Try(unrelatedChannel.toIR)
    ).map(_.failed.get.getMessage)
    assert(missing.forall(_.contains("no retained")), missing)
"""
    )
    val ran = tools.scalaCli(
      Seq("run", client.toString, "--main-class", "framework.fixture.Probe") ++
        settings ++ Seq(
          "--scalac-option",
          s"-Xplugin:$plugin",
          "--scalac-option",
          "-P:umpire-reify:mode=realization"
        )
    )
    assertEquals(ran.exit, 0, ran.diagnostics)

  test("the reify cache key accepts exactly one lowercase SHA-256 digest"):
    val digest = "0" * 64
    var attempt = 0
    def compile(options: String*) =
      attempt += 1
      val client = source(
        scratch.resolve(s"reify-cache-key-$attempt"),
        s"""package fixture.cachekey
object Probe$attempt
"""
      )
      tools.scalaCli(
        Seq("compile", client.toString) ++ settings ++
          Seq("--scalac-option", s"-Xplugin:$plugin") ++
          options.flatMap(option => Seq("--scalac-option", s"-P:umpire-reify:$option"))
      )
    val accepted = compile(s"cache-key=$digest")
    assertEquals(accepted.exit, 0, accepted.diagnostics)
    val refused = Seq(
      Seq("cache-key=short"),
      Seq(s"cache-key=${"A" * 64}"),
      Seq(s"cache-key=$digest", s"cache-key=${"1" * 64}")
    )
    refused.foreach { options =>
      val result = compile(options*)
      assertEquals(result.exit, 1, result.diagnostics)
      assert(
        result.diagnostics.contains("invalid umpire-reify cache key"),
        result.diagnostics
      )
    }

  test("an authored family sentinel remains forbidden"):
    val escape = 92.toChar
    val sentinel = s"${escape}u0000umpire-family-context:v1${escape}u0000"
    val file = source(
      scratch.resolve("family-authored-sentinel"),
      s"""package framework.realize
object Impostor:
  val forbidden = "$sentinel"
"""
    )
    val compiled = tools.scalaCli(
      Seq("compile", file.toString) ++ settings ++
        Seq("--scalac-option", s"-Xplugin:$plugin")
    )
    assertEquals(compiled.exit, 1, compiled.diagnostics)
    assert(
      compiled.diagnostics.contains(
        "reserved family provenance cannot be authored"
      ),
      compiled.diagnostics
    )

    val spoof = namedSource(
      scratch.resolve("family-authored-sentinel-owner-spoof"),
      "Family.scala",
      s"""package framework.realize
object Family:
  val forbidden = "$sentinel"
"""
    )
    val spoofed = tools.scalaCli(
      Seq("compile", spoof.toString) ++ settings ++
        Seq("--scalac-option", s"-Xplugin:$plugin")
    )
    assertEquals(spoofed.exit, 1, spoofed.diagnostics)
    assert(
      spoofed.diagnostics.contains(
        "reserved family provenance cannot be authored"
      ) &&
        spoofed.diagnostics.contains("Family.scala:3"),
      spoofed.diagnostics
    )

  test(
    "framework naming contracts do not exempt authored subtypes or same-named helpers"
  ):
    val subtype = source(
      scratch.resolve("naming-contract-subtype"),
      """package fixture.naming
import framework.realize.*
final class CustomInstruction extends Instruction:
  val invalid = everyCase(Instruction.Hold("gate"))
"""
    )
    val subtypeResult = tools.scalaCli(
      Seq("compile", subtype.toString) ++ settingsWith(pluginFramework) ++
        Seq("--scalac-option", s"-Xplugin:$plugin")
    )
    assertEquals(subtypeResult.exit, 1, subtypeResult.diagnostics)
    assert(
      subtypeResult.diagnostics.contains("a command is named after the val"),
      subtypeResult.diagnostics
    )

    val helper = source(
      scratch.resolve("naming-contract-helper"),
      """package fixture.opaque
import framework.realize.Instruction
object Helpers:
  def commanded(value: Instruction): Instruction = value
"""
    )
    val helperJar = scratch.resolve("naming-contract-helper.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          helper.toString,
          "--library",
          "-f",
          "-o",
          helperJar.toString
        ) ++
          settingsWith(pluginFramework) ++
          Seq("--scalac-option", s"-Xplugin:$plugin")
      )
      .orFail(): Unit
    val caller = source(
      scratch.resolve("naming-contract-caller"),
      """package fixture.naming
import framework.realize.*
val invalid = everyCase(fixture.opaque.Helpers.commanded(Instruction.Hold("gate")))
"""
    )
    val helperResult = tools.scalaCli(
      Seq("compile", caller.toString) ++ settingsWith(pluginFramework) ++
        Seq("--jar", helperJar.toString, "--scalac-option", s"-Xplugin:$plugin")
    )
    assertEquals(helperResult.exit, 1, helperResult.diagnostics)
    assert(
      helperResult.diagnostics.contains("a command is named after the val"),
      helperResult.diagnostics
    )

    val scriptsSpoof = namedSource(
      scratch.resolve("naming-contract-scripts-owner-spoof"),
      "Scripts.scala",
      """package framework.realize
private def commanded(value: Instruction): Instruction = value
def accept(value: Instruction): Instruction = value
val invalid = accept(commanded(Instruction.Hold("gate")))
"""
    )
    val scriptsResult = tools.scalaCli(
      Seq("compile", scriptsSpoof.toString) ++ settingsWith(pluginFramework) ++
        Seq("--scalac-option", s"-Xplugin:$plugin")
    )
    assertEquals(scriptsResult.exit, 1, scriptsResult.diagnostics)
    assert(
      scriptsResult.diagnostics.contains("a command is named after the val") &&
        scriptsResult.diagnostics.contains("Scripts.scala:4"),
      scriptsResult.diagnostics
    )

    val namedSpoof = namedSource(
      scratch.resolve("naming-contract-named-owner-spoof"),
      "Named.scala",
      """package framework.realize
trait Named:
  def copied(value: Instruction): Instruction = value
object Authored extends Named:
  val invalid = everyCase(copied(Instruction.Hold("gate")))
"""
    )
    val namedResult = tools.scalaCli(
      Seq("compile", namedSpoof.toString) ++ settingsWith(pluginFramework) ++
        Seq("--scalac-option", s"-Xplugin:$plugin")
    )
    assertEquals(namedResult.exit, 1, namedResult.diagnostics)
    assert(
      namedResult.diagnostics.contains("a command is named after the val") &&
        namedResult.diagnostics.contains("Named.scala:5"),
      namedResult.diagnostics
    )

    val describedSpoof = namedSource(
      scratch.resolve("naming-contract-described-status-spoof"),
      "Modules.scala",
      """package temporal.realize
import framework.realize.*
final class DescribedStatus:
  def await(value: String): Instruction = Instruction.Hold(value).named("forged", "", "")
def accept(value: Instruction): Instruction = value
val invalid = accept(DescribedStatus().await("gate"))
"""
    )
    val describedResult = tools.scalaCli(
      Seq("compile", describedSpoof.toString) ++ settingsWith(
        pluginFramework
      ) ++
        Seq("--scalac-option", s"-Xplugin:$plugin")
    )
    assertEquals(describedResult.exit, 1, describedResult.diagnostics)
    assert(
      describedResult.diagnostics.contains(
        "a command is named after the val"
      ) &&
        describedResult.diagnostics.contains("Modules.scala:6"),
      describedResult.diagnostics
    )

    val invalidAwaitBodies = Seq(
      "return" -> "if value.isEmpty then return poll\n    poll.named(\"await-completed\", \"\", \"\")",
      "try" -> "try poll.named(\"await-completed\", \"\", \"\") catch case _: Throwable => poll",
      "unnamed" -> "poll"
    )
    invalidAwaitBodies.foreach { (name, body) =>
      val file = namedSource(
        scratch
          .resolve(s"naming-contract-invalid-await-$name")
          .resolve("model/temporal/realize"),
        "Modules.scala",
        s"""package temporal.realize
import framework.realize.*
final class DescribedStatus:
  def await(@scala.annotation.unused value: String): Instruction.TypedPoll[Any, Any] =
    val poll = null.asInstanceOf[Instruction.TypedPoll[Any, Any]]
    $body
def accept(value: Instruction): Instruction = value
val invalid = accept(DescribedStatus().await("gate"))
"""
      )
      val result = tools.scalaCli(
        Seq("compile", file.toString) ++ settingsWith(pluginFramework) ++
          Seq("--scalac-option", s"-Xplugin:$plugin")
      )
      assertEquals(result.exit, 1, result.diagnostics)
      assert(
        result.diagnostics.contains("a command is named after the val") &&
          result.diagnostics.contains("Modules.scala:"),
        result.diagnostics
      )
    }

    val canonicalDir = scratch
      .resolve("naming-contract-packaged-await")
      .resolve("model/temporal/realize")
    val canonical = namedSource(
      canonicalDir,
      "Modules.scala",
      """package temporal.realize
import framework.realize.*
final class DescribedStatus:
  def await(@scala.annotation.unused value: String): Instruction.TypedPoll[Any, Any] =
    val poll = null.asInstanceOf[Instruction.TypedPoll[Any, Any]]
    poll.named("await-completed", "", "")
"""
    )
    val canonicalJar = scratch.resolve("naming-contract-packaged-await.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          canonical.toString,
          "--library",
          "-f",
          "-o",
          canonicalJar.toString
        ) ++ settingsWith(pluginFramework) ++
          Seq("--scalac-option", s"-Xplugin:$plugin")
      )
      .orFail(): Unit
    val packagedCaller = source(
      scratch.resolve("naming-contract-packaged-await-caller"),
      """package fixture.packagedawait
import framework.realize.*
import temporal.realize.DescribedStatus
def accept(value: Instruction): Instruction = value
val valid = accept(DescribedStatus().await("gate"))
"""
    )
    val packagedResult = tools.scalaCli(
      Seq("compile", packagedCaller.toString) ++ settingsWith(
        pluginFramework
      ) ++
        Seq(
          "--jar",
          canonicalJar.toString,
          "--scalac-option",
          s"-Xplugin:$plugin"
        )
    )
    assertEquals(packagedResult.exit, 0, packagedResult.diagnostics)

    val commandedCaller = source(
      scratch.resolve("naming-contract-packaged-commanded-caller"),
      """package fixture.packagedcommanded
import framework.realize.*
val retained = Instruction.Hold("gate").named("complete", "fixture.complete", "Probe.scala:3")
@main def check(): Unit =
  assert(everyCase(retained).command.get.toIR("fixture.root").id == "complete")
"""
    )
    val commandedResult = tools.scalaCli(
      Seq("run", commandedCaller.toString) ++ settingsWith(pluginFramework)
    )
    assertEquals(commandedResult.exit, 0, commandedResult.diagnostics)

  test(
    "named arguments retain direct family references before native construction"
  ):
    val file = source(
      scratch.resolve("named-family"),
      """package fixture.namedfamily
import framework.realize.family
def pass(name: String): String = name
val direct = family
val named = pass(name = family)
"""
    )
    val jar = scratch.resolve("named-family.jar")
    val built = tools.scalaCli(
      Seq("compile", file.toString, "--compilation-output", jar.toString) ++
        settings ++ Seq("--scalac-option", s"-Xplugin:$plugin")
    )
    assertEquals(built.exit, 0, built.diagnostics)
    val native = source(
      scratch.resolve("named-family-native"),
      """package fixture.namedfamily
object Probe:
  def main(args: Array[String]): Unit =
    assert(direct == "fixture.namedfamily")
    assert(named == "fixture.namedfamily")
"""
    )
    val ran = tools.scalaCli(
      Seq(
        "run",
        native.toString,
        "--main-class",
        "fixture.namedfamily.Probe"
      ) ++
        settings ++ Seq("--jar", jar.toString)
    )
    assertEquals(ran.exit, 0, ran.diagnostics)

  test(
    "family identities retain compiler package provenance across helper jars and declaring packages"
  ):
    val dir = scratch.resolve("families")
    val files = Vector("first", "second").map { pkg =>
      source(
        dir.resolve(pkg),
        s"""package fixture.$pkg
import framework.dsl.*
import framework.realize.*
import fixture.shared.Helpers
object ${pkg.capitalize}Subject extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean): Boolean = s
  object rules extends Rules
val root = Realization(name = "$pkg", machine = ${pkg.capitalize}Subject, producer = family + ".direct",
  producerVersion = Helpers.curried(Helpers.outer(family))(""), roles = Vector.empty,
  correlation = Correlation(Helpers.owned(), Helpers.shared, familyOwned(),
    ".literal", 8, 8, 2, 16, 1000, 512), scripts = Vector.empty, cleanup = ".payload")
object ContextProbe:
  def main(args: Array[String]): Unit =
    val error = try
      root.toIR("fixture.$pkg.root", io.temporal.server.api.umpire.v1.Model()): Unit
      throw IllegalStateException("missing context was accepted")
    catch case error: IllegalArgumentException => error
    assert(error.getMessage.contains("family has no declaring package context"))
"""
      )
    }
    val jar = scratch.resolve("families.jar")
    val built = tools.scalaCli(
      Seq(
        "compile",
        dir.toString,
        "--print-class-path",
        "--compilation-output",
        jar.toString
      ) ++ settings ++
        Seq(
          "--jar",
          familyHelpers.toString,
          "--scalac-option",
          s"-Xplugin:$plugin"
        ),
      Output.KeptApart
    )
    built.orFail(): Unit
    val cp = Files.writeString(
      scratch.resolve("families.classpath"),
      built.output.trim
    )
    val out = scratch.resolve("families.json")
    val exported = InProcess.lift(
      tools.directory,
      Seq(s"$jar=$dir/", cp.toString, out.toString) ++ files.map { file =>
        s"fixture.${file.getParent.getFileName}.Probe$$package$$.root"
      }
    )
    assertEquals(exported.exit, 0, exported.diagnostics)
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    val expected = mapper.readTree(
      Files.readString(
        tools.directory.resolve(
          "model/framework/export/testdata/constructed/family-original.json"
        )
      )
    )
    val actual = mapper.readTree(Files.readString(out)).get("realizations")
    assertEquals(actual, expected)
    val missingContext = tools.scalaCli(
      Seq("run", dir.toString, "--main-class", "fixture.first.ContextProbe") ++
        settings ++ Seq(
          "--jar",
          familyHelpers.toString,
          "--scalac-option",
          s"-Xplugin:$plugin"
        )
    )
    assertEquals(missingContext.exit, 0, missingContext.diagnostics)
    val independent = Vector("a", "b").map { run =>
      val output = scratch.resolve(s"families-$run.json")
      val exported = tools.run(
        "java",
        Seq(
          "-cp",
          System.getProperty("java.class.path"),
          "framework.export.lift",
          s"$jar=$dir/",
          cp.toString,
          output.toString,
          "fixture.first.Probe$package$.root",
          "fixture.second.Probe$package$.root"
        )
      )
      assertEquals(exported.exit, 0, exported.diagnostics)
      Files.readString(output)
    }
    assertEquals(independent(0), independent(1))

  test(
    "contextual family stays opaque through its exact collection and poll carriers"
  ):
    val retained = source(
      scratch.resolve("contextual-family-carriers"),
      """package framework.realize
val inputs: Seq[Int] = Vector(0)
val roles: Vector[String] = inputs.map(_ => contextualFamily).toVector
def evidenceId(kind: String): String = contextualFamily + ".evidence." + kind
val poll = new Instruction.TypedPoll[Any, Any](
  null.asInstanceOf[EvidenceRef[Any, Any]],
  contextualFamily,
  Vector.empty,
  null.asInstanceOf[Condition[Any]]
)
val named = poll.named("poll", "", "")
object HostShadowProbe:
  def main(arguments: Array[String]): Unit =
    val host = Class.forName("framework.realize.Realize$package$")
    val module = host.getField("MODULE$").get(null)
    try
      host.getMethod("contextualFamily").invoke(module): Unit
      throw AssertionError("plugin-free contextualFamily returned")
    catch
      case failure: java.lang.reflect.InvocationTargetException =>
        assert(failure.getCause.isInstanceOf[IllegalStateException])
    val child = Class.forName("framework.realize.Probe$package$")
    val childModule = child.getField("MODULE$").get(null)
    child.getMethod("evidenceId", classOf[String]).invoke(childModule, "done"): Unit
"""
    )
    val carrierJar = scratch.resolve("contextual-family-carriers.jar")
    val compiled = tools.scalaCli(
      Seq(
        "compile",
        "model/framework/dsl",
        "model/framework/realize",
        retained.toString,
        "--server=false",
        "--print-class-path",
        "--compilation-output",
        carrierJar.toString
      ) ++ Seq("--scalac-option", s"-Xplugin:$plugin")
    )
    assertEquals(compiled.exit, 0, compiled.diagnostics)
    val shadowed = tools.run(
      "java",
      Seq(
        "-cp",
        Seq(
          library.toString,
          System.getProperty("java.class.path"),
          carrierJar.toString,
          compiled.output.trim
        ).mkString(java.io.File.pathSeparator),
        "framework.realize.HostShadowProbe"
      )
    )
    assertEquals(shadowed.exit, 0, shadowed.diagnostics)

    val inspected = source(
      scratch.resolve("contextual-family-inspection"),
      """package fixture.contextual
import framework.realize.*
val contains = contextualFamily.contains("family")
val point = contextualFamily.codePointAt(0)
val branch = if contextualFamily.isEmpty then "empty" else "set"
val printed = println(contextualFamily)
val values: Seq[String] = Vector(contextualFamily)
val mapped = values.map(value => println(value)).toVector
val projection = Correlation(contextualFamily, "run", "operation", "value", 8, 8, 2, 16, 1000, 512).toIR.projection.isEmpty
"""
    )
    val refused = tools.scalaCli(
      Seq("compile", inspected.toString) ++ settingsWith(pluginFramework) ++
        Seq("--scalac-option", s"-Xplugin:$plugin")
    )
    assertEquals(refused.exit, 1, refused.diagnostics)
    Vector(3, 4, 5, 6, 8, 9).foreach { line =>
      assert(
        refused.diagnostics.contains(s"Probe.scala:$line:"),
        refused.diagnostics
      )
    }
    assert(
      refused.diagnostics.contains("family cannot be inspected"),
      refused.diagnostics
    )
    assert(
      refused.diagnostics.contains(
        "receiver of scala.collection.IterableOps.map"
      ),
      refused.diagnostics
    )
    assert(
      refused.diagnostics.contains(
        "Correlation$.apply has no family forwarding contract"
      ),
      refused.diagnostics
    )

    val boundaries = source(
      scratch.resolve("contextual-family-carrier-boundaries"),
      """package framework.realize
val poll = new Instruction.TypedPoll[Any, Any](null.asInstanceOf[EvidenceRef[Any, Any]], contextualFamily, Vector.empty, null.asInstanceOf[Condition[Any]])
val omitted = poll.named("omitted")
val other = Instruction.Hold(contextualFamily).named("other", "", "")
val widened: Instruction = poll
val widenedNamed = widened.named("widened", "", "")
val taintedName = poll.named(contextualFamily, "", "")
"""
    )
    val bounded = tools.scalaCli(
      Seq("compile", boundaries.toString) ++ settingsWith(pluginFramework) ++
        Seq("--scalac-option", s"-Xplugin:$plugin")
    )
    assertEquals(bounded.exit, 1, bounded.diagnostics)
    Vector(3, 4, 6, 7).foreach { line =>
      assert(
        bounded.diagnostics.contains(s"Probe.scala:$line:"),
        bounded.diagnostics
      )
    }
    assert(
      bounded.diagnostics.contains("family cannot supply a retained name") ||
        bounded.diagnostics.contains("family forwarding contract"),
      bounded.diagnostics
    )

  test(
    "family inspection refuses through local aliases and source or jar helper parameters"
  ):
    val file = source(
      scratch.resolve("family-refused"),
      """package fixture.refused
import framework.realize.*
import fixture.shared.Helpers
val direct = family.isEmpty
val cut = family.substring(1)
val alias = family
val aliasLength = alias.length
def inspect(prefix: String): Boolean =
  val local = prefix
  local.isEmpty
def outer(prefix: String): Boolean = inspect(prefix)
val indirect = outer(family)
val foreign = Helpers.inspectOuter(family)
val data = family.hashCode
def curried(@scala.annotation.unused ignored: String)(prefix: String): Boolean = prefix.isEmpty
val curriedUse = curried("literal")(family)
val jarCurried = Helpers.inspectCurried("literal")(family)
val returned = Helpers.outer(family).substring(1)
val owned = Helpers.owned().length
val guarded = if Helpers.shared.isEmpty then ".empty" else ".notempty"
final case class Box(prefix: String)
val boxed = Box(family).prefix.isEmpty
object Checks:
  require(family.isEmpty)
val hidden = script(family, Activation.Controller)().id.isEmpty
val patterned = Helpers.inspectPattern(family)
val caught = try throw IllegalArgumentException(family) catch case error: IllegalArgumentException => error.getMessage
val contextualCapture = framework.dsl.Reify.function((_: Boolean) => familyOwned())
val namedGuard = if outer(prefix = family) then "empty" else "nonempty"
"""
    )
    val ran = tools.scalaCli(
      Seq("compile", file.toString) ++ settings ++
        Seq(
          "--jar",
          familyHelpers.toString,
          "--scalac-option",
          s"-Xplugin:$plugin"
        )
    )
    assertEquals(ran.exit, 1, ran.diagnostics)
    Vector(4, 5, 7, 12, 13, 14, 16, 17, 18, 19, 20, 22, 24, 25, 26, 27, 28, 29)
      .foreach { line =>
        assert(ran.diagnostics.contains(s"Probe.scala:$line:"), ran.diagnostics)
      }
    assert(
      ran.diagnostics.contains("family cannot be inspected"),
      ran.diagnostics
    )
    assert(
      ran.diagnostics.contains(
        "contextual family cannot enter captured function IR"
      ),
      ran.diagnostics
    )

  test(
    "ordinary source family has equal stored body IR and native value across a jar"
  ):
    val file = source(
      scratch.resolve("family-body"),
      """package fixture.body
import framework.dsl.Reify
import fixture.shared.Helpers
object Probe:
  def main(args: Array[String]): Unit =
    val function = Reify.function(Helpers.body)
    assert(function.getBody.getLiteral.getText == "fixture.shared")
    val callback = Reify.function((_: Boolean) => framework.realize.family)
    assert(callback.getBody.getLiteral.getText == "fixture.body")
"""
    )
    val ran = tools.scalaCli(
      Seq("run", file.toString, "--main-class", "fixture.body.Probe") ++
        settings ++ Seq(
          "--jar",
          familyHelpers.toString,
          "--scalac-option",
          s"-Xplugin:$plugin"
        )
    )
    assertEquals(ran.exit, 0, ran.diagnostics)
    val native = source(
      scratch.resolve("family-native-body"),
      """package fixture.body
object Probe:
  def main(args: Array[String]): Unit =
    assert(fixture.shared.Helpers.body(true) == "fixture.shared")
"""
    )
    val evaluated = tools.scalaCli(
      Seq("run", native.toString, "--main-class", "fixture.body.Probe") ++
        settings ++ Seq("--jar", familyHelpers.toString)
    )
    assertEquals(evaluated.exit, 0, evaluated.diagnostics)

  test(
    "an unrewritten family getter refuses before inspection and publishes no output"
  ):
    val marker = scratch.resolve("unrewritten-initialized.txt")
    val file = source(
      scratch.resolve("family-unrewritten"),
      """package fixture.unrewritten
import framework.dsl.*
import framework.realize.*
val initialized = java.nio.file.Files.writeString(java.nio.file.Path.of("MARKER"), "initialized")
object Subject extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean): Boolean = s
  object rules extends Rules
def build(producer: String): Realization = Realization(machine = Subject, producer = producer,
  producerVersion = "1", roles = Vector.empty,
  correlation = Correlation("p", "r", "o", "v", 8, 8, 2, 16, 1000, 512), scripts = Vector.empty)
lazy val first = build("literal")
lazy val failed = build(if family.isEmpty then ".empty" else ".nonempty")
lazy val foreign = build(if familyOwned().isEmpty then ".empty" else ".nonempty")
""".replace("MARKER", marker.toString)
    )
    val jar = scratch.resolve("family-unrewritten.jar")
    val built = tools.scalaCli(
      Seq(
        "compile",
        file.toString,
        "--print-class-path",
        "--compilation-output",
        jar.toString
      ) ++ settings ++ Seq("--jar", familyHelpers.toString),
      Output.KeptApart
    )
    built.orFail(): Unit
    val cp = Files.writeString(
      scratch.resolve("family-unrewritten.classpath"),
      built.output.trim
    )
    Vector("first", "failed", "foreign").foreach { name =>
      val out = scratch.resolve(s"family-unrewritten-$name.json")
      val refused = InProcess.lift(
        tools.directory,
        Seq(
          s"$jar=${file.getParent}/",
          cp.toString,
          out.toString,
          s"fixture.unrewritten.Probe$$package$$.$name"
        )
      )
      assertEquals(refused.exit, 1, refused.diagnostics)
      assert(
        refused.diagnostics.contains("compiler source provenance"),
        refused.diagnostics
      )
      assert(
        refused.diagnostics.contains(
          s"fixture.unrewritten.Probe$$package$$.$name"
        ),
        refused.diagnostics
      )
      assert(!Files.exists(out))
      assert(!Files.exists(marker))
    }

  test(
    "compiler provenance carriers remain internal and cannot replace structural proof"
  ):
    val file = source(
      scratch.resolve("private-provenance-carriers"),
      """package fixture.contracts
import framework.dsl.{familyFlow, namedResult}
@familyFlow("v1", "-2")
def forgedFamily: String = "forged"
@namedResult
def forgedNamed: String = "forged"
"""
    )
    val refused = tools.scalaCli(Seq("compile", file.toString) ++ settings)
    assertEquals(refused.exit, 1, refused.diagnostics)
    assert(refused.diagnostics.contains("E006"), refused.diagnostics)
    Vector("familyFlow", "namedResult", "Probe.scala:3:", "Probe.scala:5:")
      .foreach { text =>
        assert(refused.diagnostics.contains(text), refused.diagnostics)
      }

    val forged = source(
      scratch.resolve("forged-provenance-carrier"),
      """package framework.realize
import framework.dsl.{familyFlow, namedResult}
@familyFlow("v1", "-2")
def forged(prefix: String): String = if prefix.isEmpty then "empty" else "forged"
@namedResult
def forgedNamed: String = "forged"
val invalid = forged(family)
"""
    )
    val rejected = tools.scalaCli(
      Seq("compile", forged.toString) ++ settingsWith(pluginFramework) ++
        Seq("--scalac-option", s"-Xplugin:$plugin")
    )
    assertEquals(rejected.exit, 1, rejected.diagnostics)
    assert(
      rejected.diagnostics.contains("E173") ||
        rejected.diagnostics.contains("cannot be accessed"),
      rejected.diagnostics
    )
    Vector("familyFlow", "namedResult", "Probe.scala:3:", "Probe.scala:5:")
      .foreach { text =>
        assert(rejected.diagnostics.contains(text), rejected.diagnostics)
      }

  test(
    "an opaque jar helper cannot erase symbolic family before a validated root reads it"
  ):
    val helper = source(
      scratch.resolve("family-opaque-helper"),
      """package fixture.opaque
object Helper:
  val hiddenValue = if framework.realize.familyOwned().isEmpty then "empty" else "branch"
  def hidden(): String = hiddenValue
"""
    )
    val jar = scratch.resolve("family-opaque-helper.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          helper.toString,
          "--library",
          "-f",
          "-o",
          jar.toString
        ) ++
          settings ++ Seq("--jar", familyHelpers.toString)
      )
      .orFail(): Unit
    val spoof = source(
      scratch.resolve("family-opaque-spoof"),
      """package scala.umpireopaque
object Helper:
  def hidden(): String = fixture.opaque.Helper.hidden()
"""
    )
    val spoofJar = scratch.resolve("family-opaque-spoof.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          spoof.toString,
          "--library",
          "-f",
          "-o",
          spoofJar.toString
        ) ++
          settings ++ Seq("--jar", jar.toString)
      )
      .orFail(): Unit
    Vector(
      ("fixture.opaque.Helper.hidden()", "hidden", "hidden"),
      ("fixture.opaque.Helper.hiddenValue", "hiddenValue", "hiddenValue"),
      ("scala.umpireopaque.Helper.hidden()", "spoof", "hidden")
    ).foreach { (expression, name, method) =>
      val caller = source(
        scratch.resolve(s"family-opaque-root-$name"),
        s"""package fixture.opaqueclient
import framework.dsl.*
import framework.realize.*
object Subject extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean): Boolean = s
  object rules extends Rules
val root = Realization(machine = Subject, producer = $expression, producerVersion = "1",
  roles = Vector.empty, correlation = Correlation("p", "r", "o", "v", 8, 8, 2, 16, 1000, 512),
  scripts = Vector.empty)
"""
      )
      val refused = tools.scalaCli(
        Seq("compile", caller.toString) ++ settings ++
          Seq(
            "--jar",
            familyHelpers.toString,
            "--jar",
            jar.toString,
            "--jar",
            spoofJar.toString,
            "--scalac-option",
            s"-Xplugin:$plugin"
          )
      )
      assertEquals(refused.exit, 1, refused.diagnostics)
      assert(
        refused.diagnostics.contains("Probe.scala:8:"),
        refused.diagnostics
      )
      assert(
        refused.diagnostics.contains(
          s"Helper$$.$method has no validated model helper contract"
        ),
        refused.diagnostics
      )
    }

  test(
    "a warm helper-only edit invalidates family contracts and the dependent unit"
  ):
    val dir = scratch.resolve("family-warm")
    val daemon = retained(Path.of("/tmp"), "family-warm.")
    Files.setPosixFilePermissions(
      daemon,
      java.nio.file.attribute.PosixFilePermissions.fromString("rwx------")
    ): Unit
    val warm = settings.filterNot(_ == "--server=false") ++
      Seq(
        "--bloop-daemon-dir",
        daemon.toString,
        "--verbose",
        "--scalac-option",
        "-Ycheck:typer",
        "--scalac-option",
        s"-Xplugin:$plugin"
      )
    source(
      dir,
      """package fixture.warm
import framework.realize.family
val value = Helper.forward(family)
"""
    ): Unit
    val helper = dir.resolve("Helper.scala")
    def body(expression: String): Unit =
      Files.writeString(
        helper,
        s"""package fixture.warm
object Helper:
  def forward(prefix: String): String = $expression
"""
      ): Unit
    try
      body("prefix + \".kept\"")
      tools.scalaCli(Seq("compile", dir.toString) ++ warm).orFail(): Unit
      body("if prefix.isEmpty then \"empty\" else \"erased\"")
      val refused = tools.scalaCli(Seq("compile", dir.toString) ++ warm)
      assert(refused.failed, refused.diagnostics)
      assert(
        refused.diagnostics.contains("family cannot be inspected"),
        refused.diagnostics
      )
      assert(
        refused.diagnostics.contains("Probe.scala after phase typer"),
        refused.diagnostics
      )
      body("prefix + \".kept\"")
      tools.scalaCli(Seq("compile", dir.toString) ++ warm).orFail(): Unit
    finally
      tools
        .run(
          "scala-cli",
          Seq("--power", "bloop", "exit", "--bloop-daemon-dir", daemon.toString)
        )
        .orFail(): Unit

  test(
    "the real names phase retains references through helpers, defaults, constructors and varargs"
  ):
    val file = source(scratch.resolve("native"), probe)
    val ran = tools.scalaCli(
      Seq("run", file.toString, "--main-class", "client.naming.Probe") ++
        settings ++ Seq("--scalac-option", s"-Xplugin:$plugin")
    )
    assertEquals(ran.exit, 0, ran.diagnostics)
    assert(
      ran.output.contains("original,alias,alias,alias,alias,original"),
      ran.diagnostics
    )

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
    assert(
      ran.diagnostics.contains("a command is named after the val"),
      ran.diagnostics
    )

    val conversion = source(
      scratch.resolve("refused-conversion"),
      """package client.naming
import framework.realize.*
val held = Instruction.Hold("gate")
def makeCommand: Command = Command("", held)
val invalid = makeCommand.toIR("client.naming.realization")
"""
    )
    val converted = tools.scalaCli(
      Seq("compile", conversion.toString) ++ settings ++
        Seq("--scalac-option", s"-Xplugin:$plugin")
    )
    assertEquals(converted.exit, 1, converted.diagnostics)
    assert(
      converted.diagnostics.contains("Probe.scala:5:"),
      converted.diagnostics
    )
    assert(
      converted.diagnostics.contains("a command is named after the val"),
      converted.diagnostics
    )

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
    val cp = Files.writeString(
      scratch.resolve("identities.classpath"),
      built.output.trim
    )
    val out = scratch.resolve("identities.json")
    val exported = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=${file.getParent}/",
        cp.toString,
        out.toString,
        "client.identities.Probe$.root"
      )
    )
    assertEquals(exported.exit, 0, exported.diagnostics)
    val json = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(Files.readString(out))
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

  test(
    "derived actions retain canonical identity and captured entity bindings"
  ):
    val stored =
      tools.directory.resolve("model/framework/export/testdata/constructed")
    val dir = scratch.resolve("bindings")
    Files.createDirectories(dir): Unit
    Files.copy(
      stored.resolve("Bindings.scala"),
      dir.resolve("Bindings.scala")
    ): Unit
    val jar = scratch.resolve("bindings.jar")
    val built = tools.scalaCli(
      Seq(
        "compile",
        dir.toString,
        "--print-class-path",
        "--compilation-output",
        jar.toString
      ) ++
        settings ++ Seq("--scalac-option", s"-Xplugin:$plugin"),
      Output.KeptApart
    )
    built.orFail(): Unit
    val cp = Files.writeString(
      scratch.resolve("bindings.classpath"),
      built.output.trim
    )
    val out = scratch.resolve("bindings.json")
    val exported = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=$dir/",
        cp.toString,
        out.toString,
        "fixture.bindings.Bindings$package$.root"
      )
    )
    assertEquals(exported.exit, 0, exported.diagnostics)
    val json = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(Files.readString(out))
    val actions = json.get("actions")
    import scala.jdk.CollectionConverters.*
    assertEquals(
      actions
        .elements()
        .asScala
        .map(a =>
          (
            a.get("id").asText(),
            a.path("on").asText(),
            a.path("creates").asText()
          )
        )
        .toSet,
      Set(
        ("fixture.bindings.caller.configure", "", ""),
        ("fixture.bindings.caller.update", "record", ""),
        ("fixture.bindings.caller.open", "", "record"),
        ("fixture.bindings.caller.inspect", "explicit", "record")
      )
    )
    strip(json)
    val expected = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(Files.readString(stored.resolve("bindings-original.json")))
    Vector("actions", "realizations").foreach(key => assertEquals(json.get(key), expected.get(key)))
    val native = tools.run(
      "java",
      Seq("-cp", built.output.trim, "fixture.bindings.NativeProbe")
    )
    assertEquals(native.exit, 0, native.diagnostics)

  test(
    "parameterized fact companions retain owning machine identity through helpers"
  ):
    val dir = scratch.resolve("fact-companions")
    Files.createDirectories(dir): Unit
    val file = Files.copy(
      tools.directory.resolve(
        "model/framework/export/testdata/constructed/Facts.scala"
      ),
      dir.resolve("Facts.scala")
    )
    val jar = scratch.resolve("fact-companions.jar")
    val built = tools.scalaCli(
      Seq(
        "compile",
        file.toString,
        "--print-class-path",
        "--compilation-output",
        jar.toString
      ) ++
        settings ++ Seq("--scalac-option", s"-Xplugin:$plugin"),
      Output.KeptApart
    )
    built.orFail(): Unit
    val cp = Files.writeString(
      scratch.resolve("fact-companions.classpath"),
      built.output.trim
    )
    val out = scratch.resolve("fact-companions.json")
    val kept = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=$dir/",
        cp.toString,
        out.toString,
        "fixture.factcases.Facts$package$.root"
      )
    )
    assertEquals(kept.exit, 0, kept.diagnostics)
    val json = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(Files.readString(out))
    assertEquals(
      json
        .get("realizations")
        .get(0)
        .get("evidence")
        .get(0)
        .get("records")
        .asText(),
      "recorded"
    )
    strip(json)
    val expected = new com.fasterxml.jackson.databind.ObjectMapper().readTree(
      Files.readString(
        tools.directory.resolve(
          "model/framework/export/testdata/constructed/facts-original.json"
        )
      )
    )
    assertEquals(json.get("realizations"), expected)
    val refusedFile = scratch.resolve("foreign-companion.json")
    val refused = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=$dir/",
        cp.toString,
        refusedFile.toString,
        "fixture.factcases.Facts$package$.foreign"
      )
    )
    assertEquals(refused.exit, 1, refused.diagnostics)
    assert(
      refused.diagnostics.contains("native machine fact catalog"),
      refused.diagnostics
    )
    assert(refused.diagnostics.contains("Facts.scala:"), refused.diagnostics)
    assert(
      refused.diagnostics.contains("fixture.factcases.foreign"),
      refused.diagnostics
    )
    assert(!Files.exists(refusedFile))
    val native = tools.run(
      "java",
      Seq("-cp", built.output.trim, "fixture.factcases.NativeProbe")
    )
    assertEquals(native.exit, 0, native.diagnostics)

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
    assert(
      ran.diagnostics.contains("a command is named after the val"),
      ran.diagnostics
    )

  test(
    "a constructed generic realization equals the independent old-host realization"
  ):
    val stored =
      tools.directory.resolve("model/framework/export/testdata/constructed")
    val fixture = scratch.resolve("oracle")
    Files.createDirectories(fixture): Unit
    val file =
      Files.copy(stored.resolve("Probe.scala"), fixture.resolve("Probe.scala"))
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
    val expected =
      mapper.readTree(Files.readString(stored.resolve("original.json")))
    val actual =
      mapper.readTree(Files.readString(out)).get("realizations").get(0)
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
    val cp = Files.writeString(
      scratch.resolve("computed.classpath"),
      built.output.trim
    )
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
      refused.diagnostics.contains(
        "while constructing root client.computed.Roots$.failed"
      ),
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
    val stored = tools.directory.resolve(
      "model/framework/export/testdata/constructed/Probe.scala"
    )
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
    val cp =
      Files.writeString(scratch.resolve("facts.classpath"), built.output.trim)
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
    val json = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(Files.readString(out))
    assertEquals(
      json
        .get("realizations")
        .get(0)
        .get("evidence")
        .get(0)
        .get("records")
        .asText(),
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
    assert(
      refused.diagnostics.contains("native machine fact catalog"),
      refused.diagnostics
    )
    assert(
      refused.diagnostics.contains("fixture.naming.foreignRealization"),
      refused.diagnostics
    )
    assert(!Files.exists(failed))
