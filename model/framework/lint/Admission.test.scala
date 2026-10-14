package framework.lint

import framework.`export`.*

import java.nio.file.{Files, Path}
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import framework.check.Tools
import FixtureBuild.buildFixture

// A lift admits the sources its roots reach (Admission.scala): a refused source no root reaches
// neither fails the lift nor is read, and one a root reaches is refused as before
// (testdata/admission: `reached` reaches nothing else, `reaching` reads `unrelated`, which the
// declaration-order lint refuses).
class AdmissionSuite extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val stored = "model/framework/export/testdata/admission/"
  private val build = tools.directory.resolve("model/build")
  private val scratch = framework.check.Scratch.directory("admission.")
  private val jar = scratch.resolve("fixture.jar")

  override def beforeAll(): Unit =
    val source = tools.directory.resolve(stored)
    Files.walk(source).iterator.asScala.filter(Files.isRegularFile(_)).foreach { from =>
      val to = scratch.resolve(source.relativize(from).toString)
      Files.createDirectories(to.getParent): Unit
      Files.writeString(
        to,
        Files
          .readString(from)
          .replace("../../../../build/model-scala.jar", build.resolve("model-scala.jar").toString)
      ): Unit
    }
    tools
      .buildFixture(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          scratch.toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit

  private def lift(root: String): (framework.check.Ran, Path) =
    val output = scratch.resolve(root + ".json")
    val ran = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=$stored,${build.resolve("model-scala.jar")}=model/",
        build.resolve("model-scala.classpath").toString,
        output.toString,
        root
      )
    )
    (ran, output)

  private val refusal =
    s"${stored}unrelated/Unrelated.scala:5: second is read while"

  test("a fixture lift does not report a refusal of a source its roots do not reach"):
    val (ran, output) = lift("fixture.admission.reached.Lamp")
    assertEquals(ran.exit, 0, ran.diagnostics)
    assert(Files.exists(output))
    assert(!ran.diagnostics.contains("Unrelated.scala"), ran.diagnostics)

  test("a source a root reaches is still admitted and refused at its line"):
    val (ran, output) = lift("fixture.admission.reaching.Switch")
    assertEquals(ran.exit, 1, ran.diagnostics)
    assert(!Files.exists(output))
    assert(ran.diagnostics.contains(refusal), ran.diagnostics)

  test("a lift of every IR file the sources declare admits every source, reached or not"):
    val out = scratch.resolve("every")
    val ran = InProcess.lift(
      tools.directory,
      Seq(
        "--ir",
        s"$jar=$stored,${build.resolve("model-scala.jar")}=model/",
        build.resolve("model-scala.classpath").toString,
        out.toString
      )
    )
    assertEquals(ran.exit, 1, ran.diagnostics)
    assert(ran.diagnostics.contains(refusal), ran.diagnostics)
