package framework.`export`

import java.nio.file.Files
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import framework.check.{RetainedScratch, Tools}
import FixtureBuild.buildFixture

private[`export`] object FixtureDeclarations:
  def apply(model: com.fasterxml.jackson.databind.JsonNode, fixture: String): (
      com.fasterxml.jackson.databind.JsonNode, String => String, String => String
  ) =
    import com.fasterxml.jackson.databind.JsonNode
    import com.fasterxml.jackson.databind.node.ObjectNode
    def strip(n: JsonNode): Unit =
      n match
        case o: ObjectNode =>
          o.remove(java.util.List.of("position", "source")): Unit
        case _ => ()
      n.elements().asScala.foreach(strip)
    val functions = model
      .path("functions")
      .elements()
      .asScala
      .map { f =>
        val body = f.deepCopy[ObjectNode]()
        body.remove("name")
        strip(body)
        f.path("name").asText() -> body
      }
      .toMap
    def deref(o: JsonNode, fields: String*): Unit = o match
      case o: ObjectNode =>
        for field <- fields if o.path(field).asText().nonEmpty do
          o.set(field, functions(o.path(field).asText())): Unit
      case _ => ()
    def find(kind: String, name: String): ObjectNode =
      model
        .path(kind)
        .elements()
        .asScala
        .find(_.path("name").asText() == name)
        .getOrElse(throw AssertionError(s"the $fixture fixture lifted no $kind named $name"))
        .deepCopy[ObjectNode]()
    def machine(name: String): String =
      val m = find("machines", name)
      m.remove(java.util.List.of("name"))
      deref(m, "evidence")
      m.path("steps").elements().asScala.foreach(deref(_, "function"))
      deref(m.path("refines"), "map", "visible", "visibleOutcomes")
      strip(m)
      m.toPrettyString
    def property(name: String): String =
      val p = find("properties", name)
      p.remove(java.util.List.of("name", "machine"))
      deref(p, "holds")
      strip(p)
      p.toPrettyString
    (model, machine, property)

class SugarFixtures extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private lazy val scratch = retained(tools.directory.resolve(".flow/tmp"), "sugar-fixtures.")

  private def declarations(fixture: String, roots: Seq[String]) =
    val dir = Files.createDirectory(scratch.resolve("fixture"))
    Files.copy(tools.directory.resolve("model/framework/export/testdata/lifts/Sugar.scala"), dir.resolve("Sugar.scala")): Unit
    Files.writeString(dir.resolve("project.scala"), "//> using scala 3.9.0\n//> using jvm 27\n"): Unit
    val jar = dir.resolve("fixture.jar")
    tools.buildFixture(Seq("--power", "package", "--server=false", "--library", "--classpath",
      ModelBuild.classpathEntries, dir.toString, "-f", "-o", jar.toString)).orFail(): Unit
    val out = dir.resolve("fixture.json")
    val pkg = s"fixture.$fixture.${fixture.capitalize}$$package$$"
    val names = roots.map(r => if r.startsWith("fixture.") then r else s"$pkg.$r")
    InProcess.lift(tools.directory, Seq(s"$jar=model/framework/export/testdata/lifts/,${ModelBuild.jarArgument}",
      ModelBuild.classpath.toString, out.toString) ++ names).orFail(): Unit
    val model = new com.fasterxml.jackson.databind.ObjectMapper().readTree(Files.readString(out))
    FixtureDeclarations(model, fixture)

  // Each sugar form beside its core spelling (lifts/Sugar.scala).
  test(
    "enter, stay, disabled, because, in, implies, records and sticky lift as their core forms do"
  ):
    val (model, machine, property) =
      declarations(
        "sugar",
        Seq(
          "fixture.sugar.Sugared",
          "fixture.sugar.Cored",
          "claims",
          "fixture.sugar.Watched"
        )
      )
    assertEquals(machine("sugared"), machine("cored"))
    for form <- Seq("records", "implies", "paused") do
      assertEquals(property(s"${form}Sugar"), property(s"${form}Core"), form)
    // Each sticky monitor beside its `monitor` spelling: one monitor but for its ID, name, position
    // and the names of its functions, whose bodies are compared in their place.
    def monitor(name: String): String =
      import com.fasterxml.jackson.databind.node.ObjectNode
      val m = model
        .path("monitors")
        .elements()
        .asScala
        .find(_.path("name").asText() == name)
        .getOrElse(fail(s"the sugar fixture lifted no monitor named $name"))
        .deepCopy[ObjectNode]()
      m.remove(java.util.List.of("id", "name", "position"))
      m.path("initial") match
        case i: ObjectNode => i.remove("position"): Unit
        case _             => ()
      for field <- Seq("next", "violated") do
        val f = model
          .path("functions")
          .elements()
          .asScala
          .find(
            _.path("name").asText() == m.path(field).asText()
          )
          .get
          .deepCopy[ObjectNode]()
        f.remove(java.util.List.of("name", "position"))
        f.findParents("position").asScala.foreach {
          case o: ObjectNode => o.remove("position"): Unit
          case _             => ()
        }
        m.set(field, f): Unit
      m.toPrettyString
    for (sugar, spelled) <- Seq(
        "refusedOnce" -> "refusedOnceSpelled",
        "retriedLost" -> "retriedLostSpelled"
      )
    do assertEquals(monitor(sugar), monitor(spelled), sugar)
    assert(
      monitor("refusedOnce").contains("neverRefused"),
      monitor("refusedOnce")
    )
    val functions = model
      .path("functions")
      .elements()
      .asScala
      .map(_.path("name").asText())
      .toList
    assert(
      functions.forall(!_.startsWith("framework.")),
      s"a framework definition was lifted as a function: ${functions.mkString(", ")}"
    )
    val resume = model
      .path("functions")
      .elements()
      .asScala
      .find(_.path("name").asText().endsWith(".resumeSugar"))
      .get
    // R4 refuses a whole-case wildcard on a Model enum, and the earlier guarded arm leaves no typed
    // remainder, so the fallback arm names every case as written.
    val cases = Seq("idle", "running", "paused", "done")
      .map(c => s"""{"literal":{"enum":{"type":"fixture.sugar.Phase","case":"$c"}}}""")
    assertEquals(
      resume.at("/body/match/cases/1/pattern").toString,
      s"""{"alternatives":{"patterns":[${cases.mkString(",")}]}}""",
      "a fallback arm that names every case lifts as their alternatives"
    )

