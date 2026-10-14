package framework.`export`

import java.nio.file.{Files, Path}
import java.util.Base64
import scala.concurrent.duration.*
import framework.check.{RetainedScratch, Tools}
import io.temporal.server.api.umpire.v1 as ir

class FiniteSchemaSuite extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private lazy val scratch =
    retained(tools.directory.resolve(".flow/tmp"), "finite-schema.")
  private def source(name: String, body: String): Path =
    val dir = Files.createDirectories(scratch.resolve(name))
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
    tools.directory.resolve("model/build/ir-scalapb.jar").toString,
    "--scalac-option",
    s"-Xplugin:$plugin"
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
    )
      .flatMap(o => Seq("--scalac-option", o))

  private val custom = """package fixture.schemas
import framework.dsl.*
import framework.dsl.Finite.given
enum Mode:
  case ready, stopped
final case class Broadcast(mode: Mode, flag: Boolean)
object Broadcast:
  given Finite[Broadcast] = new Finite[Broadcast]:
    def values: IndexedSeq[Broadcast] = Finite.derived[Broadcast].values
object Probe:
  def main(args: Array[String]): Unit =
    val witness = summon[Finite[Broadcast]]
    val schema = witness.schema.get
    assert(schema.root.getNamed == "fixture.schemas.Broadcast")
    assert(schema.types.map(_.name).toSet == Set("fixture.schemas.Broadcast", "fixture.schemas.Mode"))
    assert(witness.values.size == 4)
    val empty: Finite[Broadcast] = new Finite[Broadcast]:
      def values = throw new AssertionError("schema read enumerated catalog")
    assert(empty.schema.get == schema)
    val restricted = Finite.of[Mode](Mode.stopped, Mode.ready)
    assert(restricted.schema.get.types.head.getEnum.cases.map(_.name) == Seq("ready", "stopped"))
    assert(restricted.values == IndexedSeq(Mode.stopped, Mode.ready))
    val absent = Finite.of[Mode]()
    assert(absent.schema.get == restricted.schema.get)
    assert(absent.values.isEmpty)
    val before = Finite.of[Mode](Mode.ready)
    val unrelated = Finite.of[Broadcast]()
    assert(before.schema.get == restricted.schema.get)
    assert(unrelated.schema.get == schema)
    assert(Finite[Nothing].schema.get.types.isEmpty)
    assert(Finite[Boolean].schema.get.root.ref.isBool)
"""

  test(
    "custom values-only witnesses and empty catalogs retain structure without reading values"
  ):
    val file = source("custom", custom)
    val ran =
      tools.scalaCli(
        Seq(
          "run",
          file.toString,
          "--main-class",
          "fixture.schemas.Probe"
        ) ++ settings
      )
    assertEquals(ran.exit, 0, ran.diagnostics)

  test(
    "invalid bounds, fields, recursive schemas and name collisions refuse at the requesting source"
  ):
    for (name, declarations, target, expected) <- Seq(
        (
          "recursive",
          "final case class Loop(next: Option[Loop])",
          "Loop",
          "recursive finite schema"
        ),
        (
          "channel-recursive",
          "final case class Loop(next: Inbox[Loop])",
          "Loop",
          "recursive finite schema"
        ),
        (
          "collision",
          "object Owner { enum X { case one } }; enum `Owner.X` { case two }; final case class Both(a: Owner.X, b: `Owner.X`)",
          "Both",
          "would both be named"
        ),
        ("negative", "", "UpTo[-1]", "UpTo[-1] has no values"),
        ("nonconstant", "", "UpTo[Int]", "UpTo bounds a counter by a constant"),
        (
          "unbounded",
          "final case class Bad(unbounded: Int)",
          "Bad",
          "Int field with no declared bound"
        ),
        (
          "list",
          "final case class Bad(unbounded: List[Boolean])",
          "Bad",
          "list, which has no bound"
        )
      )
    do
      val file = source(
        name,
        s"""package fixture.refused
import framework.dsl.*
$declarations
val witness: Finite[$target] = new Finite[$target]:
  def values = IndexedSeq.empty
"""
      )
      val built = tools.scalaCli(Seq("compile", file.toString) ++ settings)
      assertEquals(built.exit, 1, built.diagnostics)
      assert(built.diagnostics.contains(expected), built.diagnostics)
      assert(built.diagnostics.contains("Probe.scala:"), built.diagnostics)

  test(
    "native schemas match independently retained old type messages and cross helper jars"
  ):
    val original = Files.readString(
      tools.directory.resolve(
        "model/framework/export/testdata/finiteSchemas/Shapes.scala"
      )
    )
    val shapes = source("shapes", original)
    val helper = source(
      "schema-helper",
      """package fixture.schemaoracle
import framework.dsl.*
object SchemaHelper:
  def nested[A](using f: Finite[Option[Option[A]]]): Finite[Option[Option[A]]] = f
  val retained: Finite[State] = new Finite[State]:
    def values = throw new AssertionError("helper schema enumerated values")
"""
    )
    val jar = scratch.resolve("shapes.jar")
    val built = tools.scalaCli(
      Seq(
        "--power",
        "package",
        shapes.toString,
        helper.toString,
        "--library",
        "-f",
        "-o",
        jar.toString
      ) ++ settings
    )
    assertEquals(built.exit, 0, built.diagnostics)
    val native = source(
      "schema-consumer",
      """package fixture.schemaoracle
import framework.dsl.*
import java.util.Base64
import io.temporal.server.api.umpire.v1 as ir
final case class ForeignBound(value: Option[bounded.Retries])
object other:
  opaque type Retries = Int
  object Retries:
    given Finite[Retries] = Finite.upTo(2)
object Native:
  def main(args: Array[String]): Unit =
    val schema = SchemaHelper.retained.schema.get
    val rebound = Finite[State].schema.get
    assert(schema == rebound)
    val bound = Finite[ForeignBound].schema.get
    assert(bound.types.find(_.name == "scala.Option[fixture.schemaoracle.bounded$.Retries]").get
      .getEnum.cases(1).fields.head.getType.getIntRange.high == 2)
    assert(Finite[bounded.Retries].schema.get.identity == "fixture.schemaoracle.bounded.Retries")
    assert(Finite[other.Retries].schema.get.identity == "fixture.schemaoracle.other.Retries")
    assert(Finite[UpTo[2]].schema.get.identity == "framework.dsl.Domain$package$.UpTo[2]")
    assert(SchemaHelper.nested[Mode].schema.get.root.getNamed == "scala.Option[scala.Option[fixture.schemaoracle.Mode]]")
    assert(Finite.of(Mode.stopped(true)).schema.get.root.getNamed == "fixture.schemaoracle.Mode")
    println("schema:" + Base64.getEncoder.encodeToString(ir.Model(types = schema.types).toByteArray))
"""
    )
    val ran = tools.scalaCli(
      Seq(
        "run",
        native.toString,
        "--main-class",
        "fixture.schemaoracle.Native",
        "--jar",
        jar.toString
      ) ++ settings
    )
    assertEquals(ran.exit, 0, ran.diagnostics)
    val actual = ir.Model.parseFrom(
      Base64.getDecoder.decode(
        ran.output.linesIterator
          .find(_.startsWith("schema:"))
          .get
          .stripPrefix("schema:")
      )
    )
    val expected = scalapb.json4s.JsonFormat.fromJsonString[ir.Model](
      Files.readString(
        tools.directory.resolve(
          "model/framework/export/testdata/finiteSchemas/original-types.json"
        )
      )
    )
    assertEquals(actual.types.sortBy(_.name), expected.types.sortBy(_.name))

  test(
    "unresolved Inbox references retain message closure without enumerating channel contents"
  ):
    val file = source(
      "inboxes",
      """package fixture.inboxes
import framework.dsl.*
enum Message:
  case first, second
val firstWire = channel[Message](capacity = 100000, order = Order.fifo, loss = Loss.reliable)
val secondWire = channel[Message](capacity = 100000, order = Order.unordered, loss = Loss.lossy)
final case class Mail(first: Inbox[Message], second: Inbox[Message])
final case class BoundedMail(small: Inbox[UpTo[1]], large: Inbox[UpTo[3]])
object Scalars:
  opaque type First = Int
  opaque type Second = Int
  given Finite[First] = Finite.upTo(2)
  given Finite[Second] = Finite.upTo(2)
final case class OpaqueMail(first: Inbox[Scalars.First], second: Inbox[Scalars.Second])
val witness: Finite[Mail] = new Finite[Mail]:
  def values = throw new AssertionError("schema enumerated channel contents")
object Probe:
  def main(args: Array[String]): Unit =
    val schema = witness.schema.get
    assert(schema.unresolvedInboxes.keySet == Set("fixture.inboxes.Message"))
    assert(schema.unresolvedInboxes("fixture.inboxes.Message").getNamed == "fixture.inboxes.Message")
    val fields = schema.types.find(_.name == "fixture.inboxes.Mail").get.getRecord.fields
    assert(fields.map(_.getType.getChannel) == Seq("fixture.inboxes.Message", "fixture.inboxes.Message"))
    assert(schema.types.exists(_.name == "fixture.inboxes.Message"))
    val state = Mail(firstWire.empty, secondWire.empty)
    assert(state.first.channel eq firstWire)
    assert(state.second.channel eq secondWire)
    val bounded: Finite[BoundedMail] = new Finite[BoundedMail]:
      def values = throw new AssertionError("schema enumerated bounded channel contents")
    val boundedSchema = bounded.schema.get
    assert(boundedSchema.unresolvedInboxes.keySet == Set("framework.dsl.Domain$package$.UpTo[1]", "framework.dsl.Domain$package$.UpTo[3]"))
    assert(boundedSchema.unresolvedInboxes.values.map(_.getIntRange.high).toSet == Set(1L, 3L))
    val opaque: Finite[OpaqueMail] = new Finite[OpaqueMail]:
      def values = throw new AssertionError("schema enumerated opaque channel contents")
    val opaqueSchema = opaque.schema.get
    assert(opaqueSchema.unresolvedInboxes.keySet == Set("fixture.inboxes.Scalars.First", "fixture.inboxes.Scalars.Second"))
    assert(opaqueSchema.unresolvedInboxes.values.forall(_.getIntRange.high == 2))
"""
    )
    val ran =
      tools.scalaCli(
        Seq(
          "run",
          file.toString,
          "--main-class",
          "fixture.inboxes.Probe"
        ) ++ settings
      )
    assertEquals(ran.exit, 0, ran.diagnostics)

  test("schema closure bytes survive the classfile string limit"):
    val suffix = "x" * 85
    val enums = (0 until 40)
      .map { i =>
        s"enum D$i:\n  case " + (0 until 20)
          .map(j => s"c${j}_$suffix")
          .mkString(", ")
      }
      .mkString("\n")
    val fields = (0 until 40).map(i => s"d$i: D$i").mkString(", ")
    val file = source(
      "large",
      s"""package fixture.large
import framework.dsl.*
import io.temporal.server.api.umpire.v1 as ir
$enums
final case class Bundle($fields)
val witness: Finite[Bundle] = new Finite[Bundle]:
  def values = throw new AssertionError("schema enumerated a product")
object Probe:
  def main(args: Array[String]): Unit =
    val schema = witness.schema.get
    assert(ir.Model(types = schema.types).toByteArray.length > 65535)
    assert(schema.types.size == 41)
    assert(schema.types.filterNot(_.name.endsWith(".Bundle")).forall(_.getEnum.cases.size == 20))
    assert(schema.types.find(_.name == "fixture.large.D39").get.getEnum.cases.last.name == "c19_$suffix")
"""
    )
    val ran =
      tools.scalaCli(
        Seq(
          "run",
          file.toString,
          "--main-class",
          "fixture.large.Probe"
        ) ++ settings
      )
    assertEquals(ran.exit, 0, ran.diagnostics)

  test(
    "opaque domains may share one native witness while retaining distinct typed schemas"
  ):
    val file = source(
      "opaque-alias",
      """package fixture.opaquealias
import framework.dsl.*
object Domains:
  opaque type First = Int
  opaque type Second = Int
  given first: Finite[First] = Finite.upTo(2)
  given second: Finite[Second] = first
object Probe:
  def selected(schema: Option[FiniteSchema]): String = schema.get.identity
  def untyped[A](value: Finite[A]): Option[FiniteSchema] = value.schema
  def main(args: Array[String]): Unit =
    val first = summon[Finite[Domains.First]]
    val second = summon[Finite[Domains.Second]]
    assert(first eq second)
    assert(first.values.size == 3 && second.values.size == 3)
    assert(untyped(first).isEmpty)
    assert(first.schema.get.identity == "fixture.opaquealias.Domains.First")
    assert(second.schema.get.identity == "fixture.opaquealias.Domains.Second")
    assert(first.schema.get.root.getIntRange.high == 2)
    assert(second.schema.get.root.getIntRange.high == 2)
    assert(selected(schema = first.schema) == "fixture.opaquealias.Domains.First")
    assert(selected(schema = second.schema) == "fixture.opaquealias.Domains.Second")
"""
    )
    val ran = tools.scalaCli(
      Seq(
        "run",
        file.toString,
        "--main-class",
        "fixture.opaquealias.Probe"
      ) ++ settings
    )
    assertEquals(ran.exit, 0, ran.diagnostics)

  test(
    "unchanged samples compile and ReliableBroadcast retains its values-only witness schema"
  ):
    val native = source(
      "original-sample",
      """package fixture.originalsample
import framework.dsl.*
import samples.ReliableBroadcast
object Probe:
  def main(args: Array[String]): Unit =
    val witness = Finite[ReliableBroadcast.Broadcast]
    val schema = witness.schema.get
    assert(schema.identity == "samples.ReliableBroadcast.Broadcast")
    val state = schema.types.find(_.name == schema.identity).get
    assert(state.getRecord.fields.map(_.name) == Seq("origin", "p1", "p2", "network"))
    assert(schema.types.exists(_.name == "samples.ReliableBroadcast.Msg"))
    assert(schema.unresolvedInboxes.nonEmpty)
"""
    )
    val sample = tools.directory.resolve("model/framework/samples")
    val ran = tools.scalaCli(
      Seq(
        "run",
        sample.toString,
        native.toString,
        "--main-class",
        "fixture.originalsample.Probe"
      ) ++ settings
    )
    assertEquals(ran.exit, 0, ran.diagnostics)

  test(
    "the existing signature reader accepts plugin-retained opaque and Inbox witnesses"
  ):
    val file =
      tools.directory.resolve(
        "model/framework/export/testdata/boundedIntegers/Bounds.scala"
      )
    val jar = scratch.resolve("bounds.jar")
    val built = tools.scalaCli(
      Seq(
        "compile",
        file.toString,
        "--print-class-path",
        "--compilation-output",
        jar.toString
      ) ++ settings
    )
    assertEquals(built.exit, 0, built.diagnostics)
    val classpath = scratch.resolve("bounds.classpath")
    Files.writeString(classpath, built.output): Unit
    val out = scratch.resolve("bounds.json")
    val lifted = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=model/framework/export/testdata/boundedIntegers/",
        classpath.toString,
        out.toString,
        "fixture.boundedintegers.Fields",
        "fixture.boundedintegers.Options",
        "fixture.boundedintegers.Messages"
      )
    )
    assertEquals(lifted.exit, 0, lifted.diagnostics)
    val model =
      scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(out))
    val counters =
      model.types.find(_.name == "fixture.boundedintegers.Counters").get
    assertEquals(
      counters.getRecord.fields.map(_.getType.getIntRange.high),
      Seq(0L, 1L, 3L, 2L)
    )
    val mail = model.types.find(_.name == "fixture.boundedintegers.Mail").get
    assertEquals(
      mail.getRecord.fields.map(_.getType.getChannel),
      Seq("fixture.boundedintegers.small", "fixture.boundedintegers.large")
    )
