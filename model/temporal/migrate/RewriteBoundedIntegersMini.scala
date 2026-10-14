package temporal.migrate

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*
import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

object RewriteBoundedIntegersMini:
  def main(args: Array[String]): Unit =
    require(args.length == 1, "provide the isolated copied-mini protocol directory")
    val directory = Path.of(args(0)).toRealPath()
    require(
      directory != Path.of("model/temporal/features/activity_mini/protocol").toRealPath(),
      "refuse authored mini"
    )
    val scratch = directory.getParent.getParent.getParent.getParent.getParent
    val rules = Rules(List(new Fn159Task9BoundedIntegers))
    def rewrite(source: String) =
      val tree = dialects.Scala3(Input.String(source)).parse[Source].get
      rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)
    for file <- Files.list(directory).iterator().asScala.filter(_.toString.endsWith(".scala")) do
      val source = Files.readString(file)
      val result = rewrite(source)
      require(result.diagnostics.isEmpty && result.fixed == source, s"already current mini: $file")
    val probe = Files.createDirectory(scratch.resolve("bounded-integer-legacy-probe"))
    val accepted = """package fixture.boundedprobe
import framework.*
import framework.Finite.given
enum Outcome:
  case accepted
val wire: Channel[UpTo[3]] = channel[UpTo[3]](capacity = 1, order = Order.fifo, loss = Loss.reliable)
final case class State(n: UpTo[2], inbox: Inbox[UpTo[3]])
given Finite[State] =
  given Finite[Inbox[UpTo[3]]] = wire.contents
  Finite.derived
val tick = timer
def step(s: State): List[Step[State, Outcome, Nothing]] =
  if s.inbox.isEmpty then List(Step(Outcome.accepted, s.copy(n = UpTo[2]((s.n + 1).min(2)), inbox = s.inbox.send(UpTo[3](2))))) else Nil
object Probe extends Machine[State, Outcome, Nothing]:
  val init = State(UpTo[2](0), wire.empty)
  def end(s: State) = true
  object rules extends Bindings(tick ~> step)
"""
    val legacy = accepted
      .replace(
        "val wire: Channel[UpTo[3]] = channel[UpTo[3]](capacity = 1, order = Order.fifo, loss = Loss.reliable)",
        "val wire: Channel[Int] = channel[Int](capacity = 1, order = Order.fifo, loss = Loss.reliable)(using Finite.upTo(3))"
      )
      .replace("n: UpTo[2]", "n: Int")
      .replace("Inbox[UpTo[3]]", "Inbox[Int]")
      .replace(
        "given Finite[State] =\n",
        "given Finite[State] =\n  given Finite[Int] = Finite.upTo(2)\n"
      )
      .replace("UpTo[2]((s.n + 1).min(2))", "(s.n + 1).min(2)")
      .replace("UpTo[3](2)", "2")
      .replace("UpTo[2](0)", "0")
    require(legacy != accepted, "labelled scratch probe must actually transform legacy syntax")
    val fixed = rewrite(legacy)
    require(fixed.diagnostics.isEmpty, fixed.diagnostics.map(_.message).mkString("; "))
    require(fixed.fixed == accepted, "exact independently authored accepted source")
    val again = rewrite(fixed.fixed)
    require(again.diagnostics.isEmpty && again.fixed == accepted, "idempotence")
    val unsafe = legacy.replace("Finite.upTo(2)", "Finite.upTo(limit)")
    val refused = rewrite(unsafe)
    require(refused.diagnostics.nonEmpty && refused.fixed == unsafe, "atomic unknown-bound refusal")
    for (name, source) <- List("legacy" -> legacy, "accepted" -> accepted, "fixed" -> fixed.fixed)
    do Files.writeString(probe.resolve(name + ".txt"), source): Unit
    Files.writeString(
      probe.resolve("project.scala"),
      "//> using scala 3.9.0\n//> using jvm 27\n"
    ): Unit
    val path = probe.resolve("Probe.scala")
    val classpath = Files.readString(scratch.resolve("before.classpath")).trim
    def run(arguments: Seq[String]): Unit =
      val process = ProcessBuilder((Seq("mise", "exec", "--", "scala-cli") ++ arguments).asJava)
        .directory(Path.of("").toAbsolutePath.toFile)
        .inheritIO()
        .start()
      require(process.waitFor() == 0, arguments.mkString(" "))
    // The probe compiles in the DSL's current spelling, as the sweep leaves it after
    // Fn169Task1DslPackage; the rule read the legacy one (run under OnLegacyDsl).
    for (stage, source) <- List("accepted" -> accepted, "fixed" -> fixed.fixed) do
      Files.writeString(path, OnLegacyDsl.forward(path, source)): Unit
      val jar = probe.resolve(stage + ".jar")
      run(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          "--classpath",
          classpath,
          probe.toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      run(
        Seq(
          "run",
          "--server=false",
          "model/framework/export",
          "model/temporal/export",
          "-M",
          "framework.export.lift",
          "--",
          s"$jar=$probe/,${scratch.resolve("before.jar")}=model/",
          scratch.resolve("before.classpath").toString,
          probe.resolve(stage + ".json").toString,
          "fixture.boundedprobe.Probe"
        )
      )
    require(
      Files.mismatch(probe.resolve("accepted.json"), probe.resolve("fixed.json")) == -1L,
      "complete accepted/fixed probe IR and positions"
    )
    println(
      s"PASS: genuine bounded-field/channel scratch legacy transformation, compilation and lift at $probe; copied mini already uses UpTo"
    )
