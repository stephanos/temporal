package framework.`export`

import java.nio.file.Files
import scala.concurrent.duration.*
import framework.check.{RetainedScratch, Tools}
import FixtureBuild.buildFixture

class TypedDeclarationsTest extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val scratch = retained(tools.directory.resolve(".flow/tmp"), "typed-declarations.")

  test("canonical typed API declarations refuse mismatched roots at their authored positions"):
    val source = Files.createDirectory(scratch.resolve("fixture"))
    Files.writeString(
      source.resolve("Invalid.scala"),
      Files
        .readString(
          tools.directory.resolve("model/framework/export/testdata/typedInvalid/Invalid.scala")
        )
        .replace("../../../../build/model-scala.jar", ModelBuild.jar.toString)
        .replace(
          "../../../../build/api-scalapb.jar",
          ModelBuild.directory.resolve("api-scalapb.jar").toString
        )
    ): Unit
    val built = tools.buildFixture(Seq("compile", source.toString))
    assert(built.failed, built.output)
    val at = """\[error\] \S*/([^/\s:]+\.scala):(\d+):(\d+)""".r
    val positions = built.errors.collect { case at(file, line, column) =>
      s"$file:$line:$column"
    }
    assertEquals(
      positions.sorted,
      Seq(
        "Invalid.scala:107:7",
        "Invalid.scala:118:7",
        "Invalid.scala:128:3",
        "Invalid.scala:132:3",
        "Invalid.scala:136:68",
        "Invalid.scala:137:69",
        "Invalid.scala:140:3",
        "Invalid.scala:143:49",
        "Invalid.scala:144:3",
        "Invalid.scala:146:46",
        "Invalid.scala:155:3",
        "Invalid.scala:157:31",
        "Invalid.scala:160:3",
        "Invalid.scala:165:49",
        "Invalid.scala:169:27",
        "Invalid.scala:29:7",
        "Invalid.scala:43:9",
        "Invalid.scala:66:7",
        "Invalid.scala:81:21",
        "Invalid.scala:84:3",
        "Invalid.scala:89:5",
        "Invalid.scala:92:3",
        "Invalid.scala:93:5",
        "Invalid.scala:97:7"
      ).sorted
    )
