package umpire.irgen

import java.nio.file.Files
import scala.concurrent.duration.*
import umpire.check.Tools

class StatusTableSeamFixtures extends munit.FunSuite:
  override val munitTimeout = 10.minutes
  private val tools = Tools.here
  private val build = tools.directory.resolve("model/build")
  private val stored = "model/irgen/testdata/statusTableSeam/"
  private def lift(name: String) =
    val scratch =
      Files.createTempDirectory(build.resolve("history"), "status-seam.")
    for file <- Vector(
        "project.scala",
        "Kinds.scala",
        "Statuses.scala",
        name + ".scala"
      )
    do
      Files.writeString(
        scratch.resolve(file),
        Files
          .readString(tools.directory.resolve(stored + file))
          .replace(
            "../../../build/model-scala.jar",
            build.resolve("model-scala.jar").toString
          )
          .replace("../../../build/api-scalapb.jar", build.resolve("api-scalapb.jar").toString)
      ): Unit
    val jar = scratch.resolve("fixture.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          "--library",
          scratch.toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    tools.run(
      java.nio.file.Path
        .of(System.getProperty("java.home"), "bin", "java")
        .toString,
      Seq(
        "-cp",
        System.getProperty("java.class.path"),
        "umpire.irgen.lintConventions",
        s"$jar=$stored,${build.resolve("model-scala.jar")}=model/",
        build.resolve("model-scala.classpath").toString
      )
    )

  test(
    "a generic capability status-table binding keeps the narrow recorded layering seam"
  ):
    val result = lift("Positive")
    result.orFail(): Unit

  test(
    "wrong section extra field blocks and calls retain exact layering refusals"
  ):
    val result = lift("Negative")
    assert(result.failed, result.output)
    for line <- Vector(11, 15, 17, 20, 23, 27, 28, 32) do
      assert(
        result.output.linesIterator.exists(s =>
          s.contains(s"Negative.scala:$line: R4.module-reference")
        ),
        result.output
      )
    assert(
      result.output.linesIterator.exists(s =>
        s.contains("Negative.scala:27: R4.module-reference") && s.contains(
          "io.temporal.api"
        )
      ),
      result.output
    )
