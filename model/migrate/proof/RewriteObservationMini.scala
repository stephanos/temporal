package umpire.migrate

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*
import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

object RewriteObservationMini:
  def main(args: Array[String]): Unit =
    require(args.length == 1, "provide the isolated copied-mini protocol directory")
    val directory = Path.of(args(0)).toRealPath()
    require(
      directory != Path.of("model/temporal/features/activity_mini/protocol").toRealPath(),
      "refuse authored mini"
    )
    val scratch = directory.getParent.getParent.getParent.getParent.getParent
    val probe = Files.createDirectory(scratch.resolve("observation-probe"))
    val accepted = Files
      .readString(Path.of("model/irgen/testdata/observations/Observations.scala"))
      .replace("Observation(", "_root_.framework.Observation(")
    val legacy = accepted
      .replace(
        "Observation(on = order)",
        "Observation(on = order, read = \"obsolete attempt label\")"
      )
      .replace(
        "Observation(\"explicit-info\", order)",
        "Observation(\"explicit-info\", order, \"obsolete label\")"
      )
    require(legacy != accepted, "scratch probe must exercise actual retired constructors")
    val rules = Rules(List(new Fn159Task6ObservationRead))
    def rewrite(source: String) =
      val tree = dialects.Scala3(Input.String(source)).parse[Source].get
      rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)
    val result = rewrite(legacy)
    require(result.diagnostics.isEmpty, result.diagnostics.map(_.message).mkString("; "))
    require(result.fixed == accepted, "restore exact independently accepted probe source")
    val again = rewrite(result.fixed)
    require(again.diagnostics.isEmpty && again.fixed == accepted, "probe migration must converge")
    val effectful = legacy.replace("\"obsolete attempt label\"", "label()")
    val refused = rewrite(effectful)
    require(
      refused.diagnostics.nonEmpty && refused.fixed == effectful,
      "refuse effectful label atomically"
    )
    Files.writeString(probe.resolve("accepted.txt"), accepted): Unit
    Files.writeString(probe.resolve("legacy.txt"), legacy): Unit
    Files.writeString(probe.resolve("fixed.txt"), result.fixed): Unit
    Files.writeString(
      probe.resolve("project.scala"),
      "//> using scala 3.9.0\n//> using jvm 27\n"
    ): Unit
    val path = probe.resolve("Observations.scala")
    val classpath = Files.readString(scratch.resolve("before.classpath")).trim
    def run(arguments: Seq[String]): Unit =
      val process = ProcessBuilder((Seq("mise", "exec", "--", "scala-cli") ++ arguments).asJava)
        .directory(Path.of("").toAbsolutePath.toFile)
        .inheritIO()
        .start()
      require(process.waitFor() == 0, arguments.mkString(" "))
    for (stage, source) <- List("accepted" -> accepted, "fixed" -> result.fixed) do
      Files.writeString(path, source): Unit
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
          "model/irgen",
          "-M",
          "umpire.irgen.lift",
          "--",
          s"$jar=model/irgen/testdata/observations/,${scratch.resolve("before.jar")}=model/",
          scratch.resolve("before.classpath").toString,
          probe.resolve(stage + ".json").toString,
          "fixture.observations.Observations$package$.run"
        )
      )
    require(
      Files.mismatch(probe.resolve("accepted.json"), probe.resolve("fixed.json")) == -1L,
      "complete probe metadata/positions must match"
    )
    val lifted = Files.readString(probe.resolve("fixed.json"))
    require(
      lifted.contains("\"id\": \"inferred\"") && lifted.contains("\"id\": \"explicit-info\""),
      "independent probe identities"
    )
    require(
      lifted.contains("temporal.api.activity.v1.ActivityExecutionInfo") && lifted.contains(
        "\"path\": \"info\""
      ),
      "independent typed message and actual response path"
    )
    println(
      s"PASS: isolated Observation probe transformed, converged, compiled and lifted at $probe; authored mini has no Observation"
    )
