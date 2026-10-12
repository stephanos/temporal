package umpire.irgen

import java.nio.file.Files
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import umpire.check.Tools

class InputDefaultsTest extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val scratch =
    Files.createTempDirectory(tools.directory.resolve(".flow/tmp"), "input-defaults.")
  private val json = new com.fasterxml.jackson.databind.ObjectMapper()
  // The packaged Models (ModelBuild): the framework this fixture reads, with the shared code and
  // the mini, which its lifts admit only where its roots reach them.
  private def modelJar = ModelBuild.jar
  private def classpath = ModelBuild.classpath
  private lazy val fixtureJar =
    val directory = Files.createDirectory(scratch.resolve("source"))
    Files.copy(
      tools.directory.resolve("model/irgen/testdata/inputDefaults/Defaults.scala"),
      directory.resolve("Defaults.scala")
    ): Unit
    Files.writeString(
      directory.resolve("project.scala"),
      "//> using scala 3.9.0\n//> using jvm 27\n"
    ): Unit
    val jar = directory.resolve("fixture.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          "--classpath",
          ModelBuild.classpathEntries,
          directory.toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    jar
  private def lift(root: String) =
    val out = scratch.resolve(root + ".json")
    val ran = InProcess.lift(
      tools.directory,
      Seq(
        s"$fixtureJar=model/irgen/testdata/inputDefaults/,$modelJar=model/",
        classpath.toString,
        out.toString,
        s"fixture.inputdefaults.Calls$$.${root}Query"
      )
    )
    (ran, out)

  test("empty, partial, full and reused bindings retain explicit finite values and input order"):
    val (ran, out) = lift("classes")
    assertEquals(ran.exit, 0, ran.diagnostics)
    val classes =
      json.readTree(Files.readString(out)).at("/scenarios/0/actions").elements().asScala.toVector
    def inputs(i: Int) = classes(i).path("inputs").elements().asScala.toVector
    assertEquals(classes.size, 15)
    assertEquals((0 until 6).map(i => inputs(i).size), 1 to 6)
    assertEquals(
      inputs(5).map(v =>
        if v.has("enum") then v.at("/enum/case").asText()
        else if v.has("bool") then v.path("bool").asText()
        else if v.has("int") then v.path("int").asText()
        else "record"
      ),
      Vector("second", "first", "false", "0", "failed", "record")
    )
    assertEquals(inputs(5)(4).at("/enum/fields/0/bool").asBoolean(), true)
    assertEquals(inputs(5)(5).at("/record/fields/0/enum/case").asText(), "second")
    assertEquals(inputs(6).head.at("/enum/case").asText(), "first")
    assertEquals(inputs(7)(3).path("int").asInt(), 2)
    assertEquals(inputs(8).map(_.at("/enum/case").asText()), Vector("Some", "None"))
    assertEquals(inputs(8).head.at("/enum/fields/0/enum/case").asText(), "second")
    assertEquals(inputs(9).map(_.at("/enum/case").asText()), Vector("second", "second"))
    assertEquals(inputs(10), inputs(5))
    assertEquals(classes(10).path("action").asText(), "fixture.inputdefaults.caller.six")
    assert(inputs(11).isEmpty && inputs(12).isEmpty)
    assertEquals(inputs(13), inputs(14))
    assertEquals(inputs(13).map(_.at("/enum/case").asText()), Vector("None", "Some"))

  test(
    "required omissions, duplicate aliases and foreign tokens refuse independently at their calls"
  ):
    for (root, message) <- List(
        "omitted" -> "input required of needs has no default",
        "partial" -> "input required of needs has no default",
        "duplicate" -> "given preferred twice",
        "foreign" -> "required is no input of two",
        "named" -> "input requiredNamed of named has no default",
        "message" -> "input message of mailboxDelivery has no default",
        "computed" -> "an input is declared by the token a val names"
      )
    do
      val (ran, out) = lift(root)
      assertEquals(ran.exit, 1, ran.diagnostics)
      assert(ran.diagnostics.contains(message), ran.diagnostics)
      val source = Files.readString(
        tools.directory.resolve("model/irgen/testdata/inputDefaults/Defaults.scala")
      )
      val line = source.linesIterator.toVector.indexWhere(
        _.trim.startsWith(
          if root == "computed" then "val computed = action(this)"
          else s"val $root = Model.scenario"
        )
      ) + 1
      assert(ran.diagnostics.contains(s"Defaults.scala:$line:"), ran.diagnostics)
      assert(!Files.exists(out))
