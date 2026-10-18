package framework.`export`

import java.nio.file.{Files, Path}
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import framework.check.{RetainedScratch, Tools}
import FixtureBuild.buildFixture

class DeclarationSugarFixtures extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private lazy val scratch = retained(tools.directory.resolve(".flow/tmp"), "declaration-sugar.")
  private lazy val fixture: (Path, Path) =
    val source = Files.createDirectory(scratch.resolve("fixture"))
    Files.copy(
      tools.directory.resolve("model/framework/export/testdata/lifts/Patterns.scala"),
      source.resolve("Patterns.scala")
    ): Unit
    Files.writeString(
      source.resolve("project.scala"),
      "//> using scala 3.9.0\n//> using jvm 27\n"
    ): Unit
    val jar = source.resolve("fixture.jar")
    tools
      .buildFixture(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          "--classpath",
          ModelBuild.classpathEntries,
          source.toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    (source, jar)

  private def declarations(name: String, roots: Seq[String]) =
    val (source, jar) = fixture
    val out = source.resolve(roots.head + ".json")
    val named = roots.map(root => s"fixture.$name.${name.capitalize}$$package$$.$root")
    InProcess
      .lift(
        tools.directory,
        Seq(
          s"$jar=$source/,${ModelBuild.jarArgument}",
          ModelBuild.classpath.toString,
          out.toString
        ) ++ named
      )
      .orFail(): Unit
    val model = new com.fasterxml.jackson.databind.ObjectMapper().readTree(Files.readString(out))
    FixtureDeclarations(model, name)

  // fn-112.4: claim patterns and shared claims (lifts/Patterns.scala).
  // Each claim pattern beside its lambda spelling, on the machine and on the composition, and each
  // claim written once over `Declares[S]` beside the lambda it stands for on either.
  test(
    "once/keeps, never, never/from, stays, stays/unless and records over a member lift as their lambdas do"
  ):
    import com.fasterxml.jackson.databind.JsonNode
    import com.fasterxml.jackson.databind.node.{ArrayNode, ObjectNode}
    def population(roots: Seq[String]): JsonNode =
      val pieces = roots.map(root => declarations("patterns", Seq(root))._1)
      assertEquals(pieces.map(_.path("queries").size()).sum, 23)
      val combined = pieces.head
      for piece <- pieces.tail; kind <- Seq("properties", "functions") do
        val rows =
          combined
            .path(kind)
            .asInstanceOf[ArrayNode] // scalafix:ok DisableSyntax.asInstanceOf
        piece.path(kind).elements().asScala.foreach(row => rows.add(row): Unit)
      combined
    val model = population(Seq("patternClaims", "inlineClaims", "sharedClaims"))
    val cores =
      population(Seq("coreClaims", "inlineCoreClaims", "sharedCoreClaims"))
    for kind <- Seq("properties", "functions") do
      val rows = model
        .path(kind)
        .asInstanceOf[ArrayNode] // scalafix:ok DisableSyntax.asInstanceOf
      cores.path(kind).elements().asScala.foreach(row => rows.add(row): Unit)
    val functions =
      model
        .path("functions")
        .elements()
        .asScala
        .map(f => f.path("name").asText() -> f)
        .toMap
    // Each child of a tree rewritten, the tree itself kept unless `f` replaces it.
    def rewrite(n: JsonNode)(f: PartialFunction[JsonNode, JsonNode]): JsonNode =
      f.applyOrElse(
        n,
        {
          case o: ObjectNode =>
            for k <- o.fieldNames().asScala.toList do o.set(k, rewrite(o.get(k))(f)): Unit
            o
          case a: ArrayNode =>
            for i <- 0 until a.size do a.set(i, rewrite(a.get(i))(f)): Unit
            a
          case other => other
        }
      )
    // A pattern lifts a lambda of its own as a function named after the Property's and the word that
    // takes it, `<holds>.never`, as `holds` lifts one; such a call is read as the lambda's body, so a
    // claim written with one compares with the lambda that calls the def inside it.
    def inlined(n: JsonNode, holds: String): JsonNode = rewrite(n) {
      case o: ObjectNode if o.path("call").path("function").asText().startsWith(s"$holds.") =>
        val f = functions(o.path("call").path("function").asText())
        val param = f.path("params").get(0).path("name").asText()
        val arg = inlined(o.path("call").path("args").get(0), holds)
        rewrite(f.path("body").deepCopy[JsonNode]()) {
          case v: ObjectNode if v.path("var").asText() == param =>
            arg.deepCopy[JsonNode]()
        }
    }
    def strip(n: JsonNode): Unit =
      n match
        case o: ObjectNode =>
          o.remove(java.util.List.of("position", "source")): Unit
        case _ => ()
      n.elements().asScala.foreach(strip)
    def found(machine: String, name: String): JsonNode =
      model
        .path("properties")
        .elements()
        .asScala
        .find(p =>
          p.path("machine")
            .asText() == machine && p.path("name").asText() == name
        )
        .getOrElse(
          fail(s"the patterns fixture lifted no Property $name of $machine")
        )
    def property(machine: String, name: String): String =
      val p = found(machine, name).deepCopy[ObjectNode]()
      val holds = p.path("holds").asText()
      val f = functions(holds).deepCopy[ObjectNode]()
      f.remove("name")
      p.set("holds", inlined(f, holds)): Unit
      p.remove(java.util.List.of("name", "machine"))
      strip(p)
      p.toPrettyString
    // Each form by its Property on the job and on the pair, and whether it is a transition one.
    val forms = Seq(
      ("doneKeeps", "pairDoneKeeps", true),
      ("noTwo", "pairNoTwo", false),
      ("notStartedWhilePaused", "pairNotStartedWhilePaused", true),
      ("doneStays", "pairDoneStays", true),
      ("activeStays", "pairActiveStays", true)
    )
    val shared =
      Seq(
        "notAdmittedWhilePaused" -> true,
        "atMostOneActive" -> false,
        "terminalStays" -> true
      )
    for
      (onJob, onPair, transition) <- forms
      (machine, name) <- Seq("job" -> onJob, "pair" -> onPair)
    do
      assertEquals(
        property(machine, name),
        property(machine, s"${name}Core"),
        name
      )
      assertEquals(
        found(machine, name).path("transition").asBoolean,
        transition,
        name
      )
    for name <- Seq("pairStarted", "lampFlipped") do
      assertEquals(
        property("pair", name),
        property("pair", s"${name}Core"),
        name
      )
    for (name, transition) <- shared; machine <- Seq("job", "pair") do
      assertEquals(
        property(machine, name),
        property(machine, s"${name}Core"),
        s"$machine.$name"
      )
      assertEquals(
        found(machine, name).path("transition").asBoolean,
        transition,
        name
      )
    for (name, transition) <- Seq(
        "doneKeepsInline" -> true,
        "noTwoInline" -> false,
        "notStartedInline" -> true,
        "doneStaysInline" -> true,
        "activeStaysInline" -> true
      )
    do
      assertEquals(property("job", name), property("job", s"${name}Core"), name)
      assertEquals(
        found("job", name).path("transition").asBoolean,
        transition,
        name
      )
    val names = functions.keys.toList.sorted
    assert(
      names.forall(!_.startsWith("framework.")),
      s"a framework definition was lifted as a function: ${names.mkString(", ")}"
    )
