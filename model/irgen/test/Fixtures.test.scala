package umpire.irgen

import TestEquality.given

import java.nio.file.{Files, Path}
import java.util.concurrent.Semaphore
import scala.collection.mutable
import scala.concurrent.{blocking, Await, Future}
import scala.concurrent.ExecutionContext.Implicits.global
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import umpire.check.{Ran, Tools}

private object LayoutSources:
  def materialize(
      from: Path,
      to: Path,
      modelJar: Path,
      excluding: Set[String]
  ): Path =
    Files.createDirectories(to)
    val jar = java.util.regex.Matcher.quoteReplacement(modelJar.toString)
    val stream = Files.walk(from)
    try
      for
        source <- stream.iterator.asScala.toList
        if source.toString.endsWith(".scala")
        if !excluding.exists(from.relativize(source).startsWith(_))
      do
        val copy = to.resolve(from.relativize(source).toString)
        Files.createDirectories(copy.getParent)
        Files.writeString(
          copy,
          Files
            .readString(source)
            .replaceAll("""(\.\./)+build/model-scala\.jar""", jar)
        )
    finally stream.close()
    to

class LayoutWitnessFixtures extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val build = tools.directory.resolve("model/build")

  test(
    "the layout witness lifts its exact core alongside verify and refinement"
  ):
    val scratch = Files.createTempDirectory(
      Files.createDirectories(build.resolve("history")),
      "layout-witness."
    )
    val modelJar = build.resolve("model-scala.jar")
    val stored = "model/irgen/testdata/layout/"
    val sources = LayoutSources.materialize(
      tools.directory.resolve(stored),
      scratch.resolve("sources"),
      modelJar,
      Set("kinds", "r4")
    )
    val jar = scratch.resolve("layout.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          sources.toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    val out = scratch.resolve("ir")
    val result = InProcess.lift(
      tools.directory,
      Seq(
        "--ir",
        s"$jar=$stored,$modelJar=model/",
        build.resolve("model-scala.classpath").toString,
        out.toString,
        "lamp"
      )
    )
    assertEquals(result.exit, 0, result.output)
    val lifted = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(Files.readString(out.resolve("lamp.json")))
    def rows(kind: String) = lifted.path(kind).elements().asScala.toSeq
    assertEquals(rows("machines").size, 5)
    assertEquals(rows("queries").size, 2)
    val verify = rows("queries").find(_.path("name").asText() == "lights").get
    assertEquals(verify.path("form").asText(), "FORM_VERIFY")
    assertEquals(
      verify.path("property").path("name").asText(),
      "switchingOnLights"
    )
    val system =
      rows("machines").find(_.path("name").asText() == "lampSystem").get
    assertEquals(system.path("refines").path("product").asText(), "lampProduct")
    val query =
      rows("queries").find(_.path("name").asText() == "switchedOff").get
    assertEquals(query.path("form").asText(), "FORM_FIND")
    for kind <- Seq("property", "scenario") do
      assertEquals(query.path(kind).path("machine").asText(), "lampProduct")
      assertEquals(query.path(kind).path("name").asText(), "switchedOff")
    assertEquals(query.path("limits").path("name").asText(), "switchedOff")
    assertEquals(query.path("limits").path("steps").asInt(), 2)
    assertEquals(query.path("limits").path("actions").asInt(), 2)
    assertEquals(query.path("limits").path("search").asInt(), 512)
    assertEquals(query.path("total").asText(), "4")
    assert(!query.has("expectedRun"))
    val scenario =
      rows("scenarios").find(_.path("name").asText() == "switchedOff").get
    assertEquals(scenario.path("machine").asText(), "lampProduct")
    assertEquals(
      scenario.path("start").path("construct").path("type").asText(),
      "fixture.features.lamp.product.State"
    )
    assertEquals(
      scenario
        .path("start")
        .path("construct")
        .path("args")
        .get(0)
        .path("literal")
        .path("enum")
        .path("case")
        .asText(),
      "dark"
    )
    assertEquals(
      scenario
        .path("actions")
        .elements()
        .asScala
        .map(_.path("action").asText())
        .toList,
      List(
        "fixture.features.lamp.user.switchOn",
        "fixture.features.lamp.user.switchOff"
      )
    )
    val property =
      rows("properties").find(_.path("name").asText() == "switchedOff").get
    assertEquals(property.path("machine").asText(), "lampProduct")
    assertEquals(
      property.path("whenClass").path("action").asText(),
      "fixture.features.lamp.user.switchOff"
    )
    assertEquals(
      property.path("holds").asText(),
      "lampProduct.property.switchedOff"
    )
    val holds =
      rows("functions").find(_.path("name") == property.path("holds")).get
    val predicate = holds.path("body").path("binary")
    assertEquals(predicate.path("op").asText(), "OP_AND")
    val records = predicate.path("left").path("binary")
    assertEquals(records.path("op").asText(), "OP_CONTAINS")
    assertEquals(
      records.path("left").path("literal").path("enum").path("type").asText(),
      "fixture.features.lamp.product.Fact"
    )
    assertEquals(
      records.path("left").path("literal").path("enum").path("case").asText(),
      "switchedOff"
    )
    assertEquals(
      records.path("right").path("field").path("field").asText(),
      "facts"
    )
    val ending = predicate.path("right").path("call")
    assertEquals(
      ending.path("function").asText(),
      "lampProduct.property.switchedOff.ends"
    )
    assertEquals(
      ending.path("args").get(0).path("field").path("field").asText(),
      "state"
    )
    val ends =
      rows("functions").find(_.path("name") == ending.path("function")).get
    val statePredicate = ends.path("body").path("binary")
    assertEquals(statePredicate.path("op").asText(), "OP_EQ")
    assertEquals(
      statePredicate.path("left").path("field").path("field").asText(),
      "phase"
    )
    assertEquals(
      statePredicate
        .path("right")
        .path("literal")
        .path("enum")
        .path("type")
        .asText(),
      "fixture.features.lamp.product.Phase"
    )
    assertEquals(
      statePredicate
        .path("right")
        .path("literal")
        .path("enum")
        .path("case")
        .asText(),
      "dark"
    )

// The lifter's fixtures under testdata: the Models it must lift, compared with the IR in
// testdata/lifts/expected, and the declarations the build or the lifter must refuse, at their lines.
//
// The fixtures build against the framework and the Temporal Models the gate packaged into
// model/build, so the gate runs these tests after it packaged them. With UMPIRE_LIFTER_UPDATE set,
// as the gate's --update sets it, the expected files are rewritten instead of compared.
class WitnessFixtures extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val build = tools.directory.resolve("model/build")
  private val stored = "model/irgen/testdata/witness/"
  private lazy val scratch = Files.createTempDirectory(
    Files.createDirectories(build.resolve("history")),
    "witness-lifter."
  )
  private lazy val jar: Path =
    val sources = Files.createDirectories(scratch.resolve("sources"))
    val stream =
      Files.newDirectoryStream(tools.directory.resolve(stored), "*.scala")
    try
      for file <- stream.asScala do
        Files.writeString(
          sources.resolve(file.getFileName),
          Files
            .readString(file)
            .replace(
              "../../../build/model-scala.jar",
              build.resolve("model-scala.jar").toString
            )
            .replace(
              "../../../build/api-scalapb.jar",
              build.resolve("api-scalapb.jar").toString
            )
        ): Unit
    finally stream.close()
    val out = scratch.resolve("witness.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          sources.toString,
          "-f",
          "-o",
          out.toString
        )
      )
      .orFail(): Unit
    out

  private def lift(name: String, roots: String*): Ran = InProcess.lift(
    tools.directory,
    Seq(
      s"$jar=$stored,${build.resolve("model-scala.jar")}=model/",
      build.resolve("model-scala.classpath").toString,
      scratch.resolve(s"$name.json").toString
    ) ++ roots
  )
  private def root(name: String) = s"fixture.witness.Witnesses$$package$$.$name"
  private def model(name: String, roots: String*) =
    val ran = lift(name, roots*)
    assertEquals(ran.exit, 0, ran.output)
    new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(Files.readString(scratch.resolve(s"$name.json")))

  test(
    "witness roots lift machine, derived, composition, list and section declarations"
  ):
    val lifted = model(
      "positive",
      Seq(
        "machine",
        "derived",
        "composed",
        "repeated",
        "reversed",
        "sync",
        "distinct",
        "distinctMembers",
        "listed",
        "distinctSync"
      ).map(root)*
    )
    assertEquals(lifted.path("queries").size(), 10)
    val section = model("section", "fixture.witness.Subject$.queries")
    val query = section.path("queries").get(0)
    assertEquals(query.path("name").asText(), "discovered")
    assertEquals(query.path("form").asText(), "FORM_FIND")
    for kind <- Seq("property", "scenario") do
      assertEquals(query.path(kind).path("machine").asText(), "subject")
      assertEquals(query.path(kind).path("name").asText(), "discovered")
    for kind <- Seq("properties", "scenarios") do
      assertEquals(
        section.path(kind).get(0).path("name").asText(),
        "discovered"
      )
    assertEquals(query.path("limits").path("name").asText(), "discovered")
    assertEquals(query.path("limits").path("steps").asInt(), 2)
    assertEquals(query.path("limits").path("actions").asInt(), 2)
    assertEquals(query.path("limits").path("search").asInt(), 512)

  test(
    "witness modifiers preserve live, exploration, bounds and total in either order"
  ):
    val original = model("modifiers", root("machine"))
    val reordered = model("reorderedModifiers", root("machineReordered"))
    for field <- Seq(
        "name",
        "form",
        "property",
        "scenario",
        "limits",
        "expectedRun",
        "exploration",
        "total"
      )
    do
      assertEquals(
        original.path("queries").get(0).path(field),
        reordered.path("queries").get(0).path(field),
        field
      )
    assertEquals(original.path("queries").get(0).path("total").asText(), "4")

  test(
    "conditional recorded facts lower to ordered ordinary fact/state clauses"
  ):
    val lifted = model("conditional", root("conditional"))
    val core = model("conditionalCore", root("conditionalCore"))
    val member = model("conditionalMember", root("conditionalMember"))
    assertEquals(
      lifted.path("queries").get(0).path("property"),
      core.path("queries").get(0).path("property")
    )
    assertEquals(
      lifted.path("scenarios").get(0).path("actions"),
      core.path("scenarios").get(0).path("actions")
    )
    val fn = lifted
      .path("functions")
      .elements()
      .asScala
      .find(_.path("name").asText() == "subject.property.conditional")
      .get
    assertEquals(fn.path("params").size(), 1)
    assert(fn.path("body").toString.contains("OP_OR"), fn.toString)
    assert(fn.path("body").toString.contains("other"), fn.toString)
    assert(
      member.path("functions").toString.contains("right_other"),
      member.toString
    )
    assert(
      !member.path("functions").toString.contains("left_other"),
      member.toString
    )

  test("absent primary and subsequent fact guards lift without positive duplicate credit"):
    val absent = model("absent", root("absent"))
    val core = model("absentCore", root("absentCore"))
    assertEquals(
      absent.path("queries").get(0).path("property"),
      core.path("queries").get(0).path("property")
    )
    assertEquals(
      absent.path("properties").get(0).path("whenClass"),
      core.path("properties").get(0).path("whenClass")
    )
    assert(absent.path("functions").toString.contains("OP_NOT"), absent.toString)
    val guarded = model("guarded", root("guarded"))
    assert(guarded.path("functions").toString.contains("OP_NOT"), guarded.toString)
    val member = model("absentMember", root("absentMember"))
    assert(member.path("functions").toString.contains("right_completed"), member.toString)
    val distinct = model(
      "capabilityAbsent",
      "fixture.witness.Capable$.capabilities",
      "fixture.witness.Capabilities$package$.capabilityAbsent"
    )
    assertEquals(distinct.path("queries").size(), 2)

  test("conditional and absent core twins preserve exact functions, helpers and authored spans"):
    import io.temporal.server.api.umpire.v1 as ir
    import ir.Expr.Kind as E
    val source = Files.readAllLines(tools.directory.resolve(stored + "Witnesses.scala")).asScala
    def line(name: String) = source.indexWhere(_.startsWith(s"val $name =")) + 1
    def expr(n: Int, e: E) = ir.Expr(Some(ir.Position(stored + "Witnesses.scala", n)), e)
    def variable(n: Int, name: String = "after") = expr(n, E.Var(name))
    def field(n: Int, base: ir.Expr, name: String) =
      expr(n, E.Field(ir.FieldAccess(Some(base), name)))
    def binary(n: Int, op: ir.Binary.Op, left: ir.Expr, right: ir.Expr) =
      expr(n, E.Binary(ir.Binary(op, Some(left), Some(right))))
    def not(n: Int, value: ir.Expr) = expr(n, E.Unary(ir.Unary(ir.Unary.Op.OP_NOT, Some(value))))
    def contains(n: Int, factLine: Int, stateLine: Int, fact: String, member: Boolean) =
      val value = if member then ir.Value(ir.Value.Kind.Text(fact))
      else ir.Value(ir.Value.Kind.Enum(ir.EnumValue("fixture.witness.Note", fact)))
      binary(
        n,
        ir.Binary.Op.OP_CONTAINS,
        expr(factLine, E.Literal(value)),
        field(stateLine, variable(stateLine), "facts")
      )
    val closed = "fixture.witness.Witnesses$package$.closed"
    def calls(n: Int, fn: String, state: ir.Expr) = expr(n, E.Call(ir.Call(fn, Seq(state))))
    def state(n: Int) = field(n, variable(n), "state")
    def function(name: String, position: Int, body: ir.Expr) =
      ir.Function(
        name,
        Some(ir.Position(stored + "Witnesses.scala", position)),
        Seq(ir.Param("after", Some(ir.TypeRef().withNamed("framework.Step")))),
        body = Some(body)
      )
    for name <- Seq("conditional", "conditionalMember", "absent", "guarded", "absentMember") do
      val sugar =
        scalapb.json4s.JsonFormat.fromJsonString[ir.Model](model(name, root(name)).toString)
      val core = scalapb.json4s.JsonFormat
        .fromJsonString[ir.Model](model(name + "Core", root(name + "Core")).toString)
      val member = name.endsWith("Member")
      val holds = (if member then "together" else "subject") + ".property." + name
      val s = line(name)
      val declaration = line(name + "Property")
      val c = declaration + (if name == "conditionalMember" then 3
                             else if name == "conditional" || name == "absentMember" then 2
                             else 1)
      val corePosition = if name == "conditionalMember" then declaration + 2
      else if name == "conditional" || member then declaration + 1
      else declaration
      val primaryFact = if name == "absent" then "other"
      else if member then if name == "absentMember" then "right_completed" else "left_completed"
      else "completed"
      def body(sugarForm: Boolean): ir.Expr =
        val anchor = if sugarForm then s else c
        val primaryLine = if sugarForm && name == "conditional" then s + 2
        else if sugarForm && name == "conditionalMember" then s + 2
        else if sugarForm && name == "absentMember" then s + 2
        else if sugarForm && name == "guarded" then s + 2
        else anchor
        val primary = contains(anchor, primaryLine, anchor, primaryFact, member)
        var result = if name.startsWith("absent") then not(anchor, primary) else primary
        if name == "guarded" then
          val guardLine = if sugarForm then s + 3 else c
          result = binary(
            anchor,
            ir.Binary.Op.OP_AND,
            result,
            not(
              guardLine,
              contains(guardLine, if sugarForm then guardLine else c + 1, anchor, "other", false)
            )
          )
        if name.startsWith("conditional") || name == "guarded" then
          val n = if sugarForm then s + (if name == "guarded" then 4 else 3)
          else if name == "guarded" then c + 1
          else c
          val condition = if member && sugarForm then
            calls(n, holds + ".recordsWhen1", state(anchor))
          else
            calls(
              n,
              closed,
              if member then field(n, state(n), "right") else state(if sugarForm then anchor else n)
            )
          val selected =
            if member then "right_other" else if name == "guarded" then "completed" else "other"
          val factLine =
            if !sugarForm && (name == "conditional" || name == "conditionalMember") then c + 1
            else n
          val recorded = contains(
            n,
            factLine,
            if !sugarForm && name == "guarded" then n else anchor,
            selected,
            member
          )
          result = binary(
            anchor,
            ir.Binary.Op.OP_AND,
            result,
            binary(n, ir.Binary.Op.OP_OR, not(n, condition), recorded)
          )
        if name == "conditional" || name == "absent" then
          val n = if sugarForm then if name == "conditional" then s + 4 else s
          else if name == "conditional" then c + 2
          else c
          result = binary(
            anchor,
            ir.Binary.Op.OP_AND,
            result,
            calls(n, closed, state(if sugarForm then anchor else n))
          )
        result
      assertEquals(sugar.functions.find(_.name == holds).get, function(holds, s, body(true)), name)
      assertEquals(
        core.functions.find(_.name == holds).get,
        function(holds, corePosition, body(false)),
        name + "Core"
      )
      val helpers = sugar.functions.filter(_.name.startsWith(holds + "."))
      if name == "conditionalMember" then
        val n = s + 3
        val helper = ir.Function(
          holds + ".recordsWhen1",
          Some(ir.Position(stored + "Witnesses.scala", n)),
          Seq(ir.Param("s", Some(ir.TypeRef().withNamed("fixture.witness.Joined")))),
          body = Some(calls(n, closed, field(n, variable(n, "s"), "right")))
        )
        assertEquals(helpers, Seq(helper))
      else assertEquals(helpers, Seq.empty)
      assertEquals(
        sugar.functions.filterNot(f => f.name == holds || f.name.startsWith(holds + ".")),
        core.functions.filterNot(_.name == holds),
        name + " shared exports"
      )
      assertEquals(sugar.queries.head.getLimits, core.queries.head.getLimits, name + " bounds")
      assertEquals(
        sugar.scenarios.head.actions,
        core.scenarios.head.actions,
        name + " ordered path"
      )

  test("witness source refusals retain the authored positions"):
    val authored =
      Files
        .readAllLines(tools.directory.resolve(stored + "WitnessRejects.scala"))
        .asScala
    for (name, message) <- Seq(
        "repeatedTail" -> "last class",
        "unnamed" -> "no val",
        "shortSteps" -> "steps",
        "shortActions" -> "actions",
        "wrongOwner" -> "composed class",
        "wrongComposition" -> "class of reversed",
        "wrongMember" -> "fills right",
        "wrongSharedState" -> "fills left",
        "repeatedComposed" -> "last class",
        "repeatedSync" -> "last class",
        "repeatedSyncInput" -> "last class",
        "unbound" -> "does not bind",
        "emptyName" -> "non-empty name",
        "noField" -> "names a member by one field"
      )
    do
      val ran = lift(name, s"fixture.witness.WitnessRejects$$package$$.$name")
      assertNotEquals(ran.exit, 0, ran.output)
      assert(
        ran.output.contains(s"lift: ${stored}WitnessRejects.scala:"),
        ran.output
      )
      assert(ran.output.contains(message), ran.output)
      val start = authored.indexWhere(_.startsWith("val " + name))
      assert(start >= 0, name)
      val following = authored.indexWhere(_.startsWith("val "), start + 1)
      val end = if following < 0 then authored.size else following
      assert(
        (start + 1 to end).exists(line =>
          ran.output.contains(s"lift: ${stored}WitnessRejects.scala:$line:")
        ),
        ran.output
      )
      if name.startsWith("repeated") then
        val (key, positions) = name match
          case "repeatedTail" =>
            ("fixture.witness.worker.finish-first", "2 and 3")
          case "repeatedComposed" => ("left_finish-first", "1 and 2")
          case "repeatedSync"     => ("both", "1 and 2")
          case _                  => ("finishBoth-first", "1 and 2")
        assert(ran.output.contains("last class " + key), ran.output)
        assert(ran.output.contains("path positions " + positions), ran.output)
        val positioned = s"${stored}WitnessRejects.scala:"
        assert(
          ran.output.sliding(positioned.length).count(_ == positioned) >= 3,
          ran.output
        )
        val token = name match
          case "repeatedTail"     => "finish(Input.first)"
          case "repeatedComposed" => "Together.own("
          case "repeatedSync"     => "Synced.synced("
          case _                  => "SyncedInput.synced("
        val arguments = authored.zipWithIndex
          .slice(start, end)
          .flatMap((text, index) =>
            Seq.fill(text.sliding(token.length).count(_ == token))(index + 1)
          )
        assertEquals(arguments.size, 2, name)
        assert(
          ran.output.contains(
            s"at $positioned${arguments.head} and $positioned${arguments.last}"
          ),
          ran.output
        )

  test("witness generated-name collisions retain both authored positions"):
    for (ordinary, witness) <- Seq(
        "collisionPropertyQuery" -> "collision",
        "collisionScenarioQuery" -> "collisionWithScenario",
        "collisionNamedQuery" -> "collisionWithQuery"
      )
    do
      for roots <- Seq(Seq(ordinary, witness), Seq(witness, ordinary)) do
        val ran = lift(
          witness,
          roots.map(n => s"fixture.witness.WitnessRejects$$package$$.$n")*
        )
        assertNotEquals(ran.exit, 0, ran.output)
        assert(
          ran.output.contains("both would share one Definition ID"),
          ran.output
        )
        assert(
          ran.output.contains(
            "here and at model/irgen/testdata/witness/WitnessRejects.scala:"
          ),
          ran.output
        )
        val source =
          Files
            .readAllLines(
              tools.directory.resolve(stored + "WitnessRejects.scala")
            )
            .asScala
        val declaration = ordinary match
          case "collisionPropertyQuery" => "collisionProperty"
          case "collisionScenarioQuery" => "collisionScenario"
          case _                        => "collisionNamedQuery: Query"
        val original =
          source.indexWhere(_.startsWith("val " + declaration + " =")) + 1
        val start = source.indexWhere(_.startsWith("val " + witness + " ="))
        val generated =
          source.indexWhere(_.contains("Subject.witness("), start) + 1
        assert(
          ran.output.contains(s"${stored}WitnessRejects.scala:$original"),
          ran.output
        )
        assert(
          ran.output.contains(s"${stored}WitnessRejects.scala:$generated"),
          ran.output
        )

  test(
    "witness duplicates read generated capability triples with exact inputs and members"
  ):
    for (owner, name) <- Seq(
        "Capable" -> "capabilityDuplicate",
        "CapableTogether" -> "capabilityMemberDuplicate"
      )
    do
      val capability = s"fixture.witness.$owner$$.capabilities"
      val witness = s"fixture.witness.Capabilities$$package$$.$name"
      for roots <- Seq(Seq(capability, witness), Seq(witness, capability)) do
        val ran = lift(name, roots*)
        assertNotEquals(ran.exit, 0, ran.output)
        assert(ran.output.contains("repeats capability Query"), ran.output)
        assert(
          ran.output.contains(
            "here and at model/irgen/testdata/witness/Capabilities.scala:"
          ),
          ran.output
        )
    val lifted = model(
      "distinctCapabilities",
      "fixture.witness.Capable$.capabilities",
      "fixture.witness.CapableTogether$.capabilities",
      "fixture.witness.Capabilities$package$.capabilityDifferentInput",
      "fixture.witness.Capabilities$package$.capabilityDifferentFact",
      "fixture.witness.Capabilities$package$.capabilityDifferentMember"
    )
    assertEquals(lifted.path("queries").size(), 5)
    val alternative = model(
      "alternativeCapability",
      "fixture.witness.CapableAlternative$.capabilities",
      "fixture.witness.Capabilities$package$.capabilityAlternative"
    )
    assertEquals(alternative.path("queries").size(), 2)

  test(
    "witness default-start identity ignores only synthetic constructor positions"
  ):
    import io.temporal.server.api.umpire.v1 as ir
    def field(value: String) = ir.Expr().withLiteral(ir.Value().withText(value))
    def start(at: Int, values: Seq[String], stateType: String = "Joined") =
      ir.Expr()
        .withPosition(ir.Position("source.scala", at))
        .withConstruct(
          ir.Construct(`type` = stateType, args = values.map(field))
        )
    val initial = start(1, Seq("open", "closed"))
    assert(
      WitnessIdentity.sameDefaultStart(initial, start(9, Seq("open", "closed")))
    )
    for different <- Seq(
        start(9, Seq("closed", "closed")),
        start(9, Seq("closed", "open")),
        start(9, Seq("open", "closed"), "OtherJoined")
      )
    do assert(!WitnessIdentity.sameDefaultStart(initial, different))

  test(
    "witness empty paths, missing records and foreign facts are compiler refusals"
  ):
    val sources = Files.createDirectories(scratch.resolve("compiler"))
    val from =
      tools.directory.resolve(stored + "compiler/CompilerRejects.scala")
    Files.writeString(
      sources.resolve("CompilerRejects.scala"),
      Files.readString(from)
    ): Unit
    val ran = tools.scalaCli(
      Seq(
        "compile",
        "--server=false",
        sources.toString,
        "--scala",
        "3.9.0",
        "--jvm",
        "27",
        "--jar",
        jar.toString,
        "--jar",
        build.resolve("model-scala.jar").toString
      )
    )
    assertNotEquals(ran.exit, 0, ran.output)
    for line <- Seq(7, 8, 9, 12, 14) do
      assert(ran.output.contains(s"CompilerRejects.scala:$line:"), ran.output)

  test(
    "witness source placement shares the Query order without moving invariant triples"
  ):
    val authored =
      Files.readString(
        tools.directory.resolve(stored + "layout/placement/Placement.scala")
      )
    for (name, source, refusal) <- Seq(
        ("placed", authored, None),
        (
          "misplaced",
          authored.replace(
            "  object properties:",
            "  object properties:\n    val misplaced = witness(worker.finish(Input.first)).records(Note.other)"
          ),
          Some("belongs in")
        ),
        (
          "reordered",
          authored.replace(
            "    val checked =",
            "    val late = scenario.actions(worker.begin())\n    val checked ="
          ),
          Some("its Scenarios")
        )
      )
    do
      val directory =
        Files.createDirectories(scratch.resolve(name).resolve("placement"))
      Files.writeString(directory.resolve("Placement.scala"), source): Unit
      val out = scratch.resolve(name + ".jar")
      tools
        .scalaCli(
          Seq(
            "--power",
            "package",
            "--server=false",
            "--library",
            directory.toString,
            "--scala",
            "3.9.0",
            "--jvm",
            "27",
            "--jar",
            jar.toString,
            "--jar",
            build.resolve("model-scala.jar").toString,
            "-f",
            "-o",
            out.toString
          )
        )
        .orFail(): Unit
      val result = InProcess.lift(
        tools.directory,
        Seq(
          s"$out=${stored}layout/placement/,$jar=$stored,${build.resolve("model-scala.jar")}=model/",
          build.resolve("model-scala.classpath").toString,
          scratch.resolve(name + ".json").toString,
          "fixture.features.placement.Placed$.queries"
        )
      )
      refusal match
        case None =>
          assertEquals(result.exit, 0, result.output)
          val lifted = new com.fasterxml.jackson.databind.ObjectMapper()
            .readTree(Files.readString(scratch.resolve(name + ".json")))
          assertEquals(
            lifted
              .path("queries")
              .elements()
              .asScala
              .map(_.path("form").asText())
              .toSet,
            Set("FORM_FIND", "FORM_VERIFY")
          )
        case Some(message) =>
          assertNotEquals(result.exit, 0, result.output)
          assert(result.output.contains("Placement.scala:"), result.output)
          assert(result.output.contains(message), result.output)

  test(
    "exact-composed witness selectors match an independently derived complete class catalog"
  ):
    val lifted =
      model(
        "exact",
        root("composed"),
        root("distinctMembers"),
        root("sync"),
        root("distinctSync")
      )
    val inputs = Seq("first", "second")
    val own = for
      field <- Seq("left", "right"); action <- Seq("begin", "finish")
      input <- if action == "finish" then inputs.map("-" + _) else Seq("")
    yield s"${field}_$action$input"
    val catalogs = Map(
      "together" -> own.toSet,
      "synced" -> (own.filterNot(_.endsWith("_begin")) :+ "both").toSet,
      "syncedInput" -> (own.filterNot(key =>
        key.startsWith("left_finish-") || key == "right_begin"
      ) ++ inputs.map("finishBoth-" + _)).toSet
    )
    val expected = Map(
      "composed" -> "left_finish-first",
      "distinctMembers" -> "right_finish-first",
      "sync" -> "both",
      "distinctSync" -> "finishBoth-second"
    )
    for property <- lifted.path("properties").elements().asScala do
      val name = property.path("name").asText()
      val selector = property.path("whenAction").asText()
      assertEquals(selector, "@exact-composed-class:" + expected(name))
      val key = selector.stripPrefix("@exact-composed-class:")
      assert(catalogs(property.path("machine").asText())(key))
      val legacy =
        catalogs(property.path("machine").asText()).map(_.takeWhile(_ != '-'))
      assert(!legacy(selector))
    for (source, key) <- Seq(
        "composedWrongInput" -> "left_finish-second",
        "composedWrongMember" -> "right_finish-first"
      )
    do
      val seeded = model(source, root(source))
      val selected =
        seeded.path("properties").get(0).path("whenAction").asText()
      assertEquals(selected, "@exact-composed-class:" + key)
      assertNotEquals(selected, "@exact-composed-class:" + expected("composed"))

  test(
    "witness and ordinary core twins preserve their complete declaration metadata"
  ):
    val source = Files
      .readAllLines(tools.directory.resolve(stored + "Witnesses.scala"))
      .asScala
    def line(declaration: String, fragment: String = ""): Int =
      val start = source.indexWhere(_.startsWith("val " + declaration + " "))
      assert(start >= 0, declaration)
      if fragment.isEmpty then start + 1
      else
        val next = source.indexWhere(_.startsWith("val "), start + 1)
        val end = if next < 0 then source.size else next
        val matches =
          source.zipWithIndex
            .slice(start, end)
            .filter((text, _) => text.contains(fragment))
        assertEquals(matches.size, 1, declaration + ": " + fragment)
        matches.head._2 + 1
    def compare(
        witness: String,
        core: String,
        spans: Seq[(String, Int, Int)]
    ): Unit =
      val actual = model(witness + "Twin", root(witness))
      val expected = model(witness + "Core", root(core))
      assertEquals(actual.path("source").asText(), "model: " + root(witness))
      actual
        .asInstanceOf[com.fasterxml.jackson.databind.node.ObjectNode]
        .put("source", "model: " + root(core)): Unit
      for (pointer, from, to) <- spans do
        val position = actual.at(pointer)
        assertEquals(
          position.path("file").asText(),
          stored + "Witnesses.scala",
          pointer
        )
        assertEquals(position.path("line").asInt(), from, pointer)
        assertEquals(expected.at(pointer).path("line").asInt(), to, pointer)
        position
          .asInstanceOf[com.fasterxml.jackson.databind.node.ObjectNode]
          .put("line", to): Unit
      assertEquals(actual, expected)
    val function = "/functions/4"
    val machine = line("machine")
    val records = line("machine", ".records(")
    val ends = line("machine", ".ends(")
    val body = line("coreProperty", "after.records")
    compare(
      "machine",
      "core",
      Seq(
        (function + "/position", machine, line("coreProperty", "Subject.property(")),
        (function + "/body/position", machine, body),
        (function + "/body/binary/left/position", machine, body),
        (function + "/body/binary/left/binary/left/position", records, body),
        (function + "/body/binary/left/binary/right/position", machine, body),
        (
          function + "/body/binary/left/binary/right/field/base/position",
          machine,
          body
        ),
        (function + "/body/binary/right/position", ends, body),
        (function + "/body/binary/right/call/args/0/position", machine, body),
        (
          function + "/body/binary/right/call/args/0/field/base/position",
          machine,
          body
        ),
        ("/properties/0/position", machine, line("coreProperty", "Subject.property(")),
        ("/scenarios/0/position", machine, line("coreScenario", "Subject.scenario(")),
        ("/queries/0/position", machine, line("core:", "query("))
      )
    )
    val composed = line("composed")
    val composedBody = line("composedProperty", "after.facts.contains")
    compare(
      "composed",
      "composedCore",
      Seq(
        (function + "/position", composed, line("composedProperty", ".when(")),
        (function + "/body/position", composed, composedBody),
        (
          function + "/body/binary/left/position",
          line("composed", ".records("),
          composedBody
        ),
        (function + "/body/binary/right/position", composed, composedBody),
        (
          function + "/body/binary/right/field/base/position",
          composed,
          composedBody
        ),
        ("/properties/0/position", composed, line("composedProperty")),
        ("/scenarios/0/position", composed, line("composedScenario")),
        ("/scenarios/0/start/position", composed, line("composedScenario")),
        ("/queries/0/position", composed, line("composedCore:", "query("))
      )
    )

class ActivityHeartbeatFixtures extends munit.FunSuite:
  override val munitTimeout: Duration = 5.minutes
  private val tools = Tools.here
  private val build = tools.directory.resolve("model/build")
  private val stored = "model/irgen/testdata/realizationRefusals/heartbeat/"
  private lazy val scratch =
    Files.createTempDirectory(
      Files.createDirectories(build.resolve("history")),
      "heartbeat-lifter."
    )

  private lazy val fixtureJar: Path =
    val sources = Files.createDirectories(scratch.resolve("sources"))
    val stream =
      Files.newDirectoryStream(tools.directory.resolve(stored), "*.scala")
    try
      for file <- stream.asScala do
        Files.writeString(
          sources.resolve(file.getFileName),
          Files
            .readString(file)
            .replace(
              "../../../../build/model-scala.jar",
              build.resolve("model-scala.jar").toString
            )
            .replace(
              "../../../../build/api-scalapb.jar",
              build.resolve("api-scalapb.jar").toString
            )
        )
    finally stream.close()
    val jar = scratch.resolve("heartbeat.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          sources.toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    jar

  private def lift(roots: String*): Ran =
    InProcess.lift(
      tools.directory,
      Seq(
        s"$fixtureJar=$stored,${build.resolve("model-scala.jar")}=model/",
        build.resolve("model-scala.classpath").toString,
        scratch.resolve("out.json").toString
      ) ++ roots
    )

  test(
    "external settlement references and publication bind without a timer or a learned-kind escape"
  ):
    val result = lift(
      "fixture.heartbeat.ExternalBindings",
      "fixture.heartbeat.ScheduledBindings"
    )
    assertEquals(result.exit, 0, result.output)
    val model = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(Files.readString(scratch.resolve("out.json")))
    val realizations = model
      .path("realizations")
      .elements()
      .asScala
      .toSeq
      .map(r => r.path("name").asText() -> r)
      .toMap
    val external = realizations("externalBindings")
    val bindings = external.path("externalSettlements").elements().asScala.toSeq
    assertEquals(bindings.size, 1)
    val binding = bindings.head
    assertEquals(binding.path("carrier").asText(), "external-start")
    assertEquals(binding.path("activity").asText(), "external-attempts")
    assertEquals(binding.path("attempt").asLong(), 1L)
    assertEquals(binding.path("pending").asText(), "pending-activity")
    assertEquals(binding.path("held").asText(), "external-held")
    assertEquals(binding.path("answer").asText(), "external-answer")
    assertEquals(binding.path("settlement").asText(), "external-terminal")
    assertEquals(
      binding.path("cleanup").path("id").asText(),
      "external-cleanup"
    )
    assertEquals(binding.path("cleanup").path("regardless").asBoolean(), true)
    assertEquals(external.path("learned").size(), 0)
    assertEquals(external.path("serverSteps").size(), 0)
    val scripts = external.path("scripts").elements().asScala.toSeq
    val controllerCommands =
      scripts.head.path("items").elements().asScala.toSeq.map(_.path("command"))
    assertEquals(
      controllerCommands(1).path("awaitActivityPublication").asText(),
      "pending-activity"
    )
    val pending =
      scripts(1).path("items").get(0).path("command").path("attemptWithheld")
    assertEquals(pending.path("mode").asText(), "WITHHOLDING_MODE_SDK_PENDING")
    assertEquals(pending.path("externalSettlement").asText(), "external-answer")
    val scheduled = realizations("scheduledBindings")
    assertEquals(scheduled.path("scripts").size(), 1)
    val scheduledBasis = scheduled.path("externalSettlements").get(0)
    assertEquals(scheduledBasis.path("answer").asText(), "scheduled-answer")
    for absent <- Seq("activity", "attempt", "pending", "held", "requestCancel")
    do assert(!scheduledBasis.has(absent), absent)

  test("controller-only completion refuses a typed failure method and request"):
    val result = tools.scalaCli(
      Seq(
        "compile",
        "--server=false",
        "model/irgen/testdata/realizationRefusals/externalWrongMethod"
      )
    )
    assertNotEquals(result.exit, 0)
    assert(result.output.contains("Wrong.scala:"), result.output)
    assert(
      result.output.contains("RespondActivityTaskFailedByIdRequest"),
      result.output
    )
    assert(
      result.output.contains("RespondActivityTaskCompletedByIdRequest"),
      result.output
    )

  test(
    "deadline basis comes from the selected request field, not the input name"
  ):
    val result = lift(
      "fixture.heartbeat.FieldBasis",
      "fixture.heartbeat.HeartbeatBasis",
      "fixture.heartbeat.CloseBasis",
      "fixture.heartbeat.UnsupportedBasis",
      "fixture.heartbeat.MissingBasis"
    )
    assertEquals(result.exit, 0, result.output)
    val model = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(Files.readString(scratch.resolve("out.json")))
    val bases = model
      .path("realizations")
      .elements()
      .asScala
      .map { r =>
        val timers = r.path("serverSteps").elements().asScala.toSeq
        assertEquals(timers.size, 1)
        r.path("name").asText() -> timers.head.path("timeoutBasis").asText()
      }
      .toMap
    assertEquals(
      bases,
      Map(
        "fieldBasis" -> "TIMEOUT_BASIS_START_TO_CLOSE",
        "heartbeatBasis" -> "TIMEOUT_BASIS_HEARTBEAT",
        "closeBasis" -> "TIMEOUT_BASIS_SCHEDULE_TO_CLOSE",
        "unsupportedBasis" -> "",
        "missingBasis" -> ""
      )
    )

  test(
    "unknown timeout basis and withholding mode are refused at their declarations"
  ):
    val storedEnums = "model/irgen/testdata/realizationRefusals/enumRefusals/"
    val jar = scratch.resolve("enums.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          tools.directory.resolve(storedEnums).toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    for (root, enumName) <- Seq(
        "unknownBasis" -> "TimeoutBasis",
        "unknownMode" -> "WithholdingMode"
      )
    do
      val result = InProcess.lift(
        tools.directory,
        Seq(
          s"$jar=$storedEnums",
          build.resolve("model-scala.classpath").toString,
          scratch.resolve(s"$root.json").toString,
          s"fixture.enumrefusals.Refusals$$package$$.$root"
        )
      )
      assertNotEquals(result.exit, 0)
      val refusal =
        result.output.linesIterator.filter(_.startsWith("lift:")).toSeq
      assertEquals(refusal.size, 1, result.output)
      assert(
        refusal.head.startsWith(s"lift: ${storedEnums}Refusals.scala:"),
        result.output
      )
      assert(
        refusal.head.endsWith(s"unknown is no $enumName of the IR"),
        result.output
      )
      assert(!Files.exists(scratch.resolve(s"$root.json")))

  test(
    "heartbeat details and both withholding modes retain their distinct wire meanings"
  ):
    val result = lift("fixture.heartbeat.Commands")
    assertEquals(result.exit, 0, result.output)
    val model = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(Files.readString(scratch.resolve("out.json")))
    val realization = model.path("realizations").elements().asScala.toSeq.head
    val commands = realization
      .path("scripts")
      .elements()
      .asScala
      .toSeq
      .head
      .path("items")
      .elements()
      .asScala
      .map(_.path("command"))
      .toSeq
    assertEquals(commands.size, 3)
    assertEquals(
      commands.head
        .path("attemptHeartbeat")
        .path("details")
        .path("message")
        .asText(),
      "temporal.api.common.v1.Payloads"
    )
    val details = commands.head
      .path("attemptHeartbeat")
      .path("details")
      .path("fields")
      .elements()
      .asScala
      .toSeq
    assertEquals(details.map(_.path("name").asText()), Seq("payloads"))
    val payloads = details.head
      .path("value")
      .path("messages")
      .path("values")
      .elements()
      .asScala
      .toSeq
    assertEquals(
      payloads.map(_.path("message").asText()),
      Seq("temporal.api.common.v1.Payload", "temporal.api.common.v1.Payload")
    )
    assertEquals(
      payloads.map(
        _.path("fields")
          .elements()
          .asScala
          .toSeq
          .find(_.path("name").asText() == "data")
          .get
          .path("value")
          .path("utf8")
          .asText()
      ),
      Seq("\"first\"", "\"second\"")
    )
    assert(commands(1).has("attemptWithheld"))
    assertEquals(commands(1).path("attemptWithheld").path("mode").asText(), "")
    assertEquals(
      commands(2).path("attemptWithheld").path("mode").asText(),
      "WITHHOLDING_MODE_SDK_PENDING"
    )
    val hints =
      realization.path("behavior").path("visibility").elements().asScala.toSeq
    val heartbeat = hints.filter(
      _.path("cause").asText() == "CAUSE_KIND_ACTIVITY_HEARTBEAT"
    )
    assertEquals(heartbeat.size, 1)
    assertEquals(
      heartbeat.head.path("eventuallyWithin").path("interval").asText(),
      "0.250s"
    )
    assertEquals(
      heartbeat.head.path("eventuallyWithin").path("atMost").asText(),
      "2s"
    )

class ExpectationFixtures extends munit.FunSuite:
  override val munitTimeout: Duration = 5.minutes
  private val tools = Tools.here
  private val build = tools.directory.resolve("model/build")
  private val stored = "model/irgen/testdata/expectations/"
  private lazy val scratch =
    Files.createTempDirectory(
      Files.createDirectories(build.resolve("history")),
      "expectation-lifter."
    )
  private lazy val fixtureJar: Path =
    val sources = Files.createDirectories(scratch.resolve("sources"))
    val stream =
      Files.newDirectoryStream(tools.directory.resolve(stored), "*.scala")
    try
      for file <- stream.asScala do
        Files.writeString(
          sources.resolve(file.getFileName),
          Files
            .readString(file)
            .replace(
              "../../../build/model-scala.jar",
              build.resolve("model-scala.jar").toString
            )
            .replace(
              "../../../build/api-scalapb.jar",
              build.resolve("api-scalapb.jar").toString
            )
        )
    finally stream.close()
    Files.writeString(
      sources.resolve("Realizations.scala"),
      Files.readString(
        tools.directory.resolve("model/irgen/testdata/lifts/Realizations.scala")
      )
    ): Unit
    val jar = scratch.resolve("expectations.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          sources.toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    jar

  private def root(name: String) =
    "fixture.expectations.Expectations$package$." + name
  private def lift(name: String, roots: String*): Ran = InProcess.lift(
    tools.directory,
    Seq(
      s"$fixtureJar=$stored,${build.resolve("model-scala.jar")}=model/",
      build.resolve("model-scala.classpath").toString,
      scratch.resolve(s"$name.json").toString
    ) ++ roots
  )
  private def model(name: String, roots: String*) =
    val result = lift(name, roots*)
    assertEquals(result.exit, 0, result.output)
    new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(Files.readString(scratch.resolve(s"$name.json")))
  private def query(name: String) =
    model(name, root(name)).path("queries").get(0)
  private def authoredLine(needle: String, after: String) =
    val lines = Files
      .readAllLines(tools.directory.resolve(stored + "Expectations.scala"))
      .asScala
    val start = lines.indexWhere(_.contains(needle))
    assert(start >= 0, needle)
    val offset = lines.drop(start).indexWhere(_.contains(after))
    assert(offset >= 0, after)
    start + offset + 1

  test(
    "expectation live and total retain triple and witness metadata in either order"
  ):
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    val expectedLimits =
      mapper.readTree("""{"name":"one","steps":1,"actions":1,"search":32}""")
    val expectedSatisfied = mapper.readTree("""{
      "conformance":"CONFORMANCE_CONFORMANT", "property":"OUTCOME_SATISFIED",
      "contract":"OUTCOME_SATISFIED", "disposition":"DISPOSITION_COMPLETED",
      "cleanup":"CLEANUP_SUCCEEDED"
    }""")
    val triples = model(
      "triples",
      Seq("modelOnly", "liveThenTotal", "totalThenLive").map(root)*
    )
      .path("queries")
      .elements()
      .asScala
      .toSeq
    assertEquals(
      triples.map(_.path("name").asText()).toSet,
      Set("modelOnly", "liveThenTotal", "totalThenLive")
    )
    assertEquals(
      triples.filter(_.has("expectedRun")).map(_.path("name").asText()).toSet,
      Set("liveThenTotal", "totalThenLive")
    )
    for q <- triples do
      assertEquals(q.path("form").asText(), "FORM_FIND")
      assertEquals(q.path("total").asText(), "2")
      assertEquals(q.path("property").path("name").asText(), "closed")
      assertEquals(q.path("scenario").path("name").asText(), "once")
      assertEquals(q.path("limits"), expectedLimits)
      if q.has("expectedRun") then assertEquals(q.path("expectedRun"), expectedSatisfied)
    val first = query("witnessLiveThenTotal")
    val second = query("witnessTotalThenLive")
    for q <- Seq(first, second) do
      assertEquals(q.path("form").asText(), "FORM_FIND")
      assertEquals(q.path("total").asText(), "2")
      assertEquals(q.path("limits"), expectedLimits)
      assertEquals(q.path("expectedRun"), expectedSatisfied)
      for kind <- Seq("property", "scenario") do
        assertEquals(
          q.path(kind).path("name").asText(),
          q.path("name").asText()
        )
        assertEquals(q.path(kind).path("machine").asText(), "subject")
    assertEquals(first.path("expectedRun"), second.path("expectedRun"))
    assertEquals(first.path("limits"), second.path("limits"))
    assertEquals(
      first.path("expectedRun"),
      triples.find(_.has("expectedRun")).get.path("expectedRun")
    )

  test(
    "expectation source explanations emit identical complete expectation IR"
  ):
    val first = query("first")
    val second = query("second")
    assertEquals(first.path("name").asText(), "explained")
    assertEquals(second.path("name"), first.path("name"))
    assert(first.has("expectedRun"))
    assertEquals(first.path("expectedRun"), second.path("expectedRun"))
    val expected = first.path("expectedRun")
    assertEquals(expected.path("monitors").size(), 1)
    assertEquals(
      expected.path("monitors").get(0).path("reason").asText(),
      "REASON_NEVER_EVALUATED"
    )
    assert(!expected.toString.contains("because"))
    assert(!expected.toString.contains("the monitor"))
    val complete = query("completeFirst")
    val twin = query("completeSecond")
    assertEquals(complete.path("name"), twin.path("name"))
    assertEquals(complete.has("expectedRun"), twin.has("expectedRun"))
    assertEquals(complete.path("expectedRun"), twin.path("expectedRun"))
    val expectedComplete =
      new com.fasterxml.jackson.databind.ObjectMapper().readTree("""{
      "conformance": "CONFORMANCE_INCONCLUSIVE",
      "property": "OUTCOME_INCONCLUSIVE",
      "contract": "OUTCOME_VIOLATED",
      "disposition": "DISPOSITION_STOPPED_BY_MONITOR",
      "cleanup": "CLEANUP_FAILED",
      "reason": "REASON_NEVER_EVALUATED",
      "conformanceReason": "REASON_NO_EVIDENCE",
      "monitors": [{"name":"watched", "outcome":"OUTCOME_INCONCLUSIVE",
        "reason":"REASON_NEVER_EVALUATED"}]
    }""")
    assertEquals(complete.path("expectedRun"), expectedComplete)
    assert(!complete.path("expectedRun").toString.contains("because"))

  test(
    "expectation existing monitor declarations lift by value and name alike"
  ):
    val queries = model(
      "existingMonitors",
      "fixture.realizations.Realizations$package$.heldByValue",
      "fixture.realizations.Realizations$package$.heldByName"
    ).path("queries")
      .elements()
      .asScala
      .toSeq
    assertEquals(queries.size, 2)
    val expectations = queries.map(_.path("expectedRun"))
    assertEquals(expectations.head, expectations.last)
    assertEquals(
      expectations.head
        .path("monitors")
        .elements()
        .asScala
        .map(_.path("name").asText())
        .toList,
      List("opensOnce", "staysOpen")
    )

  test(
    "expectation missing blank and whitespace explanations refuse with the authored Query"
  ):
    val results = Seq(
      "missing",
      "empty",
      "whitespace",
      "unicode",
      "contractMissing",
      "witnessMissing",
      "monitorMissing",
      "monitorEmpty",
      "monitorWhitespace",
      "monitorUnicode"
    ).map(name => name -> lift(name, root(name)))
    assert(
      results.forall(_._2.exit != 0),
      results
        .map((name, result) => s"$name exited ${result.exit}")
        .mkString(", ")
    )
    for (name, result) <- results do
      assert(result.output.contains(s"$name's live expectation"), result.output)
      assert(result.output.contains("says no reason why"), result.output)
      val line = name match
        case "monitorMissing" =>
          authoredLine("val monitorMissing", "MonitorExpectation(")
        case "monitorEmpty" | "monitorWhitespace" | "monitorUnicode" =>
          authoredLine("def watched", "MonitorExpectation(")
        case "missing" | "contractMissing" | "witnessMissing" =>
          authoredLine(s"val $name =", "RunExpectation(")
        case _ => authoredLine(s"val $name =", ".live(")
      assert(
        result.output.contains(s"lift: ${stored}Expectations.scala:$line:"),
        result.output
      )
      assert(!Files.exists(scratch.resolve(s"$name.json")))

  test(
    "expectation capability helpers require reasons and preserve live metadata"
  ):
    val results =
      Seq(
        "MissingCapability",
        "EmptyCapability",
        "WhitespaceCapability",
        "UnicodeCapability"
      )
        .map(name => name -> lift(name, s"fixture.expectations.$name$$.capabilities"))
    assert(
      results.forall(_._2.exit != 0),
      results
        .map((name, result) => s"$name exited ${result.exit}")
        .mkString(", ")
    )
    for (name, result) <- results do
      val owner = s"${name.head.toLower}${name.tail}"
      assert(
        result.output.contains(s"$owner.closes's live expectation"),
        result.output
      )
      assert(result.output.contains("says no reason why"), result.output)
      val line = if name == "MissingCapability" then
        authoredLine(s"object $name", "RunExpectation(")
      else authoredLine(s"object $name", "inconclusive(")
      assert(
        result.output.contains(s"lift: ${stored}Expectations.scala:$line:"),
        result.output
      )
    val helper = query("helperValid")
    val capability =
      model("capability", "fixture.expectations.ValidCapability$.capabilities")
        .path("queries")
        .elements()
        .asScala
        .find(_.has("expectedRun"))
        .get
    assertEquals(helper.path("expectedRun"), capability.path("expectedRun"))
    assertEquals(capability.path("form").asText(), "FORM_FIND")
    assertEquals(capability.path("name").asText(), "validCapability.closes")

class Fixtures extends munit.FunSuite:
  // A test builds a fixture with scala-cli and lifts it in a JVM of its own.
  override val munitTimeout: Duration = 20.minutes

  private val tools = Tools.here
  private val root = tools.directory
  private val build = root.resolve("model/build")
  private val testdata = root.resolve("model/irgen/testdata")
  private val expected = testdata.resolve("lifts/expected")
  private val modelJar = build.resolve("model-scala.jar")
  private val modelClasspath = build.resolve("model-scala.classpath")
  private val update = sys.env.get("UMPIRE_LIFTER_UPDATE").exists(_.nonEmpty)

  // Each run builds in a directory of its own under the ignored model/build/history, which is kept
  // for inspection: nothing is deleted, so another process's build state is never removed.
  private lazy val scratch =
    Files.createTempDirectory(
      Files.createDirectories(build.resolve("history")),
      "lifter."
    )
  private def lifted(name: String) = scratch.resolve(s"$name.json")

  // Every build and lifter JVM holds one of these while it runs, so the tests run side by side
  // within the memory of the machine. A test awaiting a lift holds none, so none waits on itself.
  private val slots = Semaphore(6, true)
  private def bounded[A](work: => A): A =
    slots.acquire()
    try work
    finally slots.release()

  // A test that builds or lifts is started in beforeAll, after the fixtures' lifts, and runs beside
  // the others; the test awaits it, so its assertions and its failure stay its own.
  private val started = mutable.ArrayBuffer.empty[() => Unit]
  private def concurrently(name: String)(body: => Unit)(using
      munit.Location
  ): Unit =
    lazy val running = Future(blocking(body))
    started += (() => running: Unit)
    test(name)(Await.result(running, munitTimeout))

  // A fixture's sources are stored under testdata, which the lifter's project.scala excludes from
  // its build, since some of them must not compile. `materialize` copies them into the scratch
  // directory, with its project's jar path resolved. The lifter maps the copies' positions back to
  // the stored files, and the build's diagnostics are mapped back in `refusals`.
  private def materialize(fixture: String): Path =
    val to = Files.createDirectories(scratch.resolve(fixture))
    val stream = Files.newDirectoryStream(testdata.resolve(fixture), "*.scala")
    try
      for from <- stream.asScala do
        Files.writeString(
          to.resolve(from.getFileName.toString),
          Files
            .readString(from)
            .replace("../../../build/model-scala.jar", modelJar.toString)
            .replace(
              "../../../build/api-scalapb.jar",
              build.resolve("api-scalapb.jar").toString
            )
        )
    finally stream.close()
    to

  // The lifter's positions of a materialized fixture: its stored files.
  private def stored(fixture: String) = s"model/irgen/testdata/$fixture/"

  // The lines of the stored files at which the fixture's build fails, as `<file>:<line>:<column>`.
  private def refusals(fixture: String): Seq[String] =
    val built = bounded(
      tools.scalaCli(Seq("compile", materialize(fixture).toString))
    )
    assert(built.failed, s"model/irgen/testdata/$fixture built")
    val at = """\[error\] \S*/([^/\s:]+\.scala):(\d+):(\d+)""".r
    built.errors.collect { case at(file, line, column) =>
      assert(
        Files.isRegularFile(testdata.resolve(fixture).resolve(file)),
        built.output
      )
      s"$file:$line:$column"
    }

  private def packaged(
      name: String,
      sources: Path,
      options: Seq[String] = Nil
  ): Path =
    val jar = scratch.resolve(s"$name.jar")
    bounded(
      tools.scalaCli(
        Seq(
          "--power",
          "package",
          "--library",
          sources.toString,
          "-f",
          "-o",
          jar.toString
        ) ++
          options
      )
    ).orFail(): Unit
    jar

  // The lifter is run as `lift` runs it, with the arguments of its command line, in this JVM
  // (InProcess).
  private def lift(arguments: String*): Ran =
    bounded(InProcess.lift(tools.directory, arguments))

  private def refused(ran: Ran): Seq[String] =
    ran.output.linesIterator.filter(_.startsWith("lift:")).toSeq

  private val fixtures: Seq[(String, Seq[String])] = Seq(
    "presence" -> Seq("fixture.presence.Presence"),
    "channels" -> Seq("fixture.channels.Relay", "fixture.channels.Tallying"),
    "declarations" -> Seq(
      "fixture.declarations.Declarations$package$.queries",
      "fixture.declarations.Declarations$package$.durableEventually"
    ),
    "admission" -> Seq(
      "fixture.specimens.admission.Admission$package$.currentQueries",
      "fixture.specimens.admission.Admission$package$.staleQueries"
    ),
    "realizations" -> Seq(
      "fixture.realizations.Realizations$package$.learnedRun",
      "fixture.realizations.Realizations$package$.runOpens",
      "fixture.realizations.Realizations$package$.pauseRace",
      "fixture.realizations.Realizations$package$.pauseRaceQuery",
      "fixture.realizations.Realizations$package$.doorRealization",
      "fixture.realizations.Realizations$package$.doorOpens",
      "fixture.realizations.Realizations$package$.errandRealization",
      "fixture.realizations.Realizations$package$.errandRetry",
      "fixture.realizations.Realizations$package$.errandWithdrawn",
      "fixture.realizations.Realizations$package$.tallyRealization",
      "fixture.realizations.Realizations$package$.tallyOpens"
    ),
    "taskqueue" -> Seq("queueQueries", "matchingQueries", "forgetfulQueries")
      .map("fixture.taskqueue.TaskQueue$package$." + _),
    // fn-134.2: capabilities sections, a shared set among them, and a Query reading a generated
    // Property (lifts/CapabilitySections.scala).
    "capabilitySections" -> (Seq("Task", "Chore", "TaskPair", "MirrorPair")
      .map(o => s"fixture.capabilitysections.$o$$.capabilities") :+
      "fixture.capabilitysections.Task$.queries"),
    "retryCapabilities" -> Seq("WaitingFailures", "ControlledFailures")
      .map(o => s"fixture.retrycapabilities.$o$$.capabilities"),
    // fn-129.3: one instance of a repeated capability Property overridden by name.
    "retrySelection" -> Seq("ControlledFailures", "SelectedFailures")
      .map(o => s"fixture.retrycapabilities.$o$$.capabilities"),
    "deadlineCapabilities" -> Seq("DeadlineTimers", "SimpleDeadlines")
      .map(o => s"fixture.deadlinecapabilities.$o$$.capabilities"),
    // fn-135.2: `is { }` vals and their def twins (lifts/Blocks.scala).
    "blocks" -> Seq("Blocked", "Defined").flatMap(o =>
      Seq(s"fixture.blocks.$o$$.capabilities", s"fixture.blocks.$o$$.queries")
    ),
    // fn-135.5: status facts derived from the phase an effect block assigns, and their written
    // twins (lifts/StatusFacts.scala).
    "statusFacts" -> Seq("Derived", "Written").map(o => s"fixture.statusfacts.$o"),
    // fn-136.1, fn-136.4: role tests beside the case sets they stand for (lifts/Roles.scala).
    "roles" -> Seq("Roled", "Listed", "Shortened", "Written").map(
      "fixture.roles." + _
    ),
    "captured" -> Seq(
      "queries",
      "diskQueries",
      "localDiskQueries",
      "Relay",
      "ledger",
      "PutOnly",
      "durableEventually"
    ).map(r =>
      if r.head.isUpper then s"fixture.captured.$r"
      else s"fixture.captured.Captured$$package$$.$r"
    ),
    // fn-118.2: API behavior hints and server steps (lifts/Hints.scala). The reader admits the
    // first and refuses each realization of the second at its line (tools/umpire/realization).
    "hints" -> Seq("keptBehavior", "ownBehavior").map(
      "fixture.hints.Hints$package$." + _
    ),
    // fn-126 decision 23: the objects that group actions, named after where they are declared
    // (lifts/Sections.scala).
    "sections" -> Seq("fixture.sections.Switch", "fixture.sections.Clapper"),
    // fn-126 R15, R16: a machine object, its core twin, derivations and a composition object
    // (lifts/Rules.scala); tools/umpire/interp holds the twins' tables equal.
    "rules" -> (Seq(
      "Switch",
      "Mirror",
      "Steady",
      "Loose",
      "Dimming",
      "Twins",
      "Unequal",
      "Dial",
      "Turned"
    )
      .map("fixture.rules." + _) ++ Seq(
      "fixture.rules.CoreSwitch",
      "fixture.rules.Switch$.queries$.pressing",
      "fixture.rules.Switch$.queries$.wornOut"
    )),
    // fn-139.1: machines on the shared outcomes, `rejects` beside its core spelling, a refinement
    // and a capability's `rejected` (lifts/Rejections.scala).
    "rejections" -> (Seq(
      "DoorProduct",
      "DoorSystem$.capabilities",
      "DoorSystem$.queries"
    )
      .map("fixture.rejections." + _)),
    // fn-150: a composition's progress claims and the composed steps their assumptions make fair
    // (lifts/ComposedProgress.scala).
    "composedProgress" -> Seq("drainsThrough", "gradedThrough")
      .map("fixture.composedprogress.ComposedProgress$package$." + _),
    "hintsRefused" -> Seq(
      "zeroInterval",
      "nonPositiveBound",
      "intervalOverBound",
      "unboundedStep",
      "timerNoDeadline",
      "deliveryDeadline"
    ).map("fixture.hints.Hints$package$." + _)
  )

  // fn-134.2: the fixtures whose capabilities sections write the waivers they state beside their
  // IR, which the model gate writes into the accepted findings.
  private val waived: Seq[String] = Seq("capabilitySections", "rejections")
  // The refusals of fn-112.4's typed composition selectors, in lifts/Rejects.scala.
  private val selectorRejects: Seq[String] = Seq(
    "MemberNoField",
    "SyncNoMember",
    "MemberIncompatible",
    "SyncUnbound",
    "WithNoMember",
    "WithIncompatible",
    "WithUnrefined",
    "WithUnsynced",
    "syncedNone",
    "syncedTwice",
    "ownSynced",
    "ownSpelledAlike",
    "mixedSchedule",
    "bareInputs",
    "whenClassInputs",
    "ReplacesSpare",
    "LoopPair",
    "syncedNoMember",
    "ownNoMember",
    "syncedNoField"
  )
  // The refusals of fn-112.4's claim patterns, records over a member and function-valued arguments.
  private val patternRejects: Seq[String] = Seq(
    "whenStays",
    "keptComputed",
    "fromVal",
    "sharedLambda",
    "recordsNoMember",
    "recordsForeignFact",
    "recordsUnfilled",
    "paramCalled"
  )

  // The refusals of fn-112.5's input tokens, inputs supplied by name and bounded counters.
  private val inputRejects: Seq[String] = Seq(
    "foreignToken",
    "suppliedTwice",
    "supplyKept",
    "namedNoTokens",
    "inputTwice",
    "tokenUnnamed",
    "upToNegative"
  )

  // The refusals of fn-112.11's Query totals, which an author may write and the lifter otherwise
  // computes (fn-126 decision 28).
  private val totalRejects: Seq[String] = Seq(
    "totaledTwice",
    "totalComputed",
    "totalKept",
    "totalNegative",
    "sharedComputed"
  )

  // The refusals of fn-120.1's named choices.
  private val choiceRejects: Seq[String] = Seq(
    "ChosenTwice",
    "SpelledTwice",
    "ChoiceHelper",
    "ChoiceDisabled",
    "ChoiceTwoSteps",
    "ChoiceIf",
    "ChoiceUnnamed",
    "ChoiceKept",
    "ChoiceKeptHelper"
  )

  // The refusals of fn-120.2's unnamed branching: several results written without a choose.
  private val unnamedRejects: Seq[String] =
    Seq("UnnamedList", "UnnamedJoin", "UnnamedInHelper")

  // The refusals of fn-120.5's levels: a step made where no step function is, one per declaration
  // kind whose Scala type admits one (model/SEMANTICS.md, Levels).
  private val levelRejects: Seq[String] = Seq(
    "LevelStart",
    "LevelEnds",
    "LevelEvidence",
    "LevelRefinement",
    "LevelMonitor",
    "LevelRequire",
    "levelProperty",
    "levelTransition",
    "levelPattern",
    "levelProgress",
    "LevelComposition",
    "levelScenario"
  )

  // The refusals of fn-112.10's names taken by default.
  private val defaultRejects: Seq[String] =
    Seq(
      "unnamedTwice",
      "SyncNamedTwice",
      "omittedEmpty",
      "watchUnnamed",
      "watchUnwatched",
      "watchElsewhere"
    )

  // The refusals of fn-112.12's claim bundles.
  private val bundleRejects: Seq[String] = Seq("mixedBundle")

  // The refusals of fn-112.9's script helpers, status tables and request scopes
  // (lifts/ScriptRejects.scala).
  private val scriptRejects: Seq[String] = Seq(
    "unnamedCommand",
    "performsNothing",
    "onNoPath",
    "unlistedStatus",
    "statusTwice",
    "notAField",
    "notAPolledField",
    "foreignFact",
    "literalTwice",
    "baseOverride",
    "awaitUnlisted",
    "describedFactTwice",
    "historyKeyless",
    "namedTwice",
    "deadlineNoTimeout"
  ).map("fixture.scriptrejects.ScriptRejects$package$." + _)

  // The refusals of fn-133.6's realization objects, each an object of lifts/ScriptRejects.scala.
  private val realizationObjectRejects: Seq[String] =
    Seq("Unordered", "Doubled", "Foreign", "Restated", "Replaced").map(
      "fixture.scriptrejects." + _
    )

  // The refusals of fn-134.2's capabilities sections, each a machine's section
  // (lifts/CapabilitySectionRejects.scala).
  private val sectionRejects: Seq[String] = Seq(
    "ReadsNoField",
    "UnboundParameter",
    "BoundTwice",
    "DeclaredTwice",
    "WaivedUnbrought",
    "EmptyReason",
    "Unbounded",
    "OverrideUnbrought",
    "OverrideWaived"
  ).map(o => s"fixture.capabilitysectionrejects.$o$$.capabilities")

  // The refusals of fn-139.3's `from` blocks, fn-159's bare actions, phases and imports in a
  // `from`, and fn-159.7's unwritten `enters`, each a machine object (lifts/RuleRejects.scala).
  private val ruleRejects: Seq[String] =
    Seq(
      "NestedFrom",
      "Foreign",
      "ImportedInFrom",
      "BareInFrom",
      "BareImported",
      "BareClass",
      "BareDisabled",
      "BarePhase",
      "BareSet",
      "Unwritten",
      "TwiceFaceted"
    ).map("fixture.rulerejects." + _)
  // The refusals of fn-135.2's `is { }` blocks and fn-135.3's `effect { }` blocks, each a machine
  // object (lifts/BlockRejects.scala).
  private val blockRejects: Seq[String] =
    Seq(
      "NestedIs",
      "HeaderIs",
      "DefIs",
      "ShapedAccessor",
      "BranchedEffect",
      "RejectingEffect",
      "TwiceAssigned",
      "ReadAfterAssigned",
      "LocalVal",
      "BareExpression",
      "OtherCall",
      "NestedEffect",
      "ShapedSetter",
      "RuleEffect"
    ).map("fixture.blockrejects." + _)

  // The refusals of fn-135.5's status assignments, each a machine object
  // (lifts/StatusFactRejects.scala).
  private val statusRejects: Seq[String] =
    Seq("ExplicitStatus", "ComputedStatus", "PlainStatusSetter").map(
      "fixture.statusfactrejects." + _
    )

  // The refusals of fn-136.1's role tests and role conflicts, each a machine object
  // (lifts/RoleRejects.scala).
  private val roleRejects: Seq[String] = Seq(
    "NoEnum",
    "NoCase",
    "NoRole",
    "StandAlone",
    "WithFields",
    "BothLiveClosed",
    "TwoLiveRoles",
    "TwoClosureRoles",
    // fn-136.4: the short spellings.
    "WhenNoProjection",
    "WhenNoRole",
    "WhenNoEnum",
    "InNoRole",
    "InNoEnum"
  ).map("fixture.rolerejects." + _)

  // The refusals of fn-118.2's API behavior hints (lifts/HintRejects.scala).
  private val hintRejects: Seq[String] =
    Seq("builtWrite", "builtRead", "elapsedOutOfRange")
      .map("fixture.hintrejects.HintRejects$package$." + _)

  // fn-126 decision 20: what a machine is for, as its markers say (lifts/MarkerRejects.scala).
  private val markerRejects: Seq[String] =
    Seq(
      "FaultUnmarked",
      "FaultlessFailure",
      "HopelessFailure$.queries$.brokenHopelessFailure",
      "UnrefutedControl$.queries$.foundUnrefutedControl",
      "ControlRefiner",
      "RefinedControl$.queries$.askedRefinedControl",
      "SystemicControl$.queries$.askedSystemicControl",
      "TornMarkers$.queries$.askedTornMarkers",
      "UnmarkedPair",
      "FaultlessPair",
      "DerivedControl$.queries$.foundDerivedControl"
    ).map("fixture.markers." + _)

  // The refusals of fn-150's composition progress claims and composed fairness
  // (lifts/ComposedProgressRejects.scala).
  private val composedProgressRejects: Seq[String] = Seq(
    "anotherOwner",
    "onAMachine",
    "mixedFairness",
    "spreadFairness",
    "oneSidedClass",
    "pairedOwn",
    "unpairedSync",
    "noStep"
  ).map(
    "fixture.composedprogressrejects.ComposedProgressRejects$package$." + _
  ) :+
    "fixture.composedprogressrejects.AssumingFlow"

  // A root of lifts/Rejects.scala: a machine or composition object, or an object nested in one of its
  // objects, by its object (`Hoarding`, `Anonymous$.anonymous`); a top-level val by its name.
  private def rejectsRoot(name: String): String =
    if name.head.isUpper then s"fixture.rejects.$name"
    else s"fixture.rejects.Rejects$$package$$.$name"

  private val rejected = Seq(
    "Hoarding",
    "closed",
    "Waiting",
    "Doubled",
    "Listening",
    "Counter",
    "crossedRead",
    "negative",
    "Watched",
    "Unrefined",
    "Misplaced",
    "noSuchRoot",
    "UnrefinedOutcomes",
    "Counting",
    "Batching",
    "Shuffling",
    "Guessing",
    "unnamedProperty",
    "unnamedScenario",
    "Twins",
    "askedTwice",
    "boundTwice",
    "Tapped",
    "Undeclared",
    "Unlisted",
    "Partial",
    "unrelatedRead",
    "AssumedTwice",
    "GapTwice",
    "RebindPush",
    "ExtendBound",
    "ReboundTwice",
    "AssumedAgain",
    "AssumingTwice",
    "RefinedNothing",
    "RefinedOtherwise",
    "LoopFirst",
    "Aliased",
    "Splatted",
    "Explained"
  ).map(
    rejectsRoot
  ) ++ (selectorRejects ++ patternRejects ++ inputRejects ++ totalRejects ++
    choiceRejects ++ unnamedRejects ++ bundleRejects ++ defaultRejects ++ levelRejects)
    .map(
      rejectsRoot
    ) ++ Seq(
    // A name the compiler made up and a computed ok outcome, refused in objects of their own.
    "Anonymous$.anonymous",
    "ComputedOk$.ComputedOk"
  ).map(rejectsRoot) ++ Seq(
    // fn-126 R15, R16 and decision 27: machine objects, rules and effects.
    "EffectOutside",
    "EmptyEffect",
    "Unheaded",
    "BlockTwice",
    "FromStatement",
    "PhaseLambda",
    "DisabledFired",
    "ExtendedBare",
    "RebindSeveral",
    "RebindUnbound",
    "Endless",
    "EndedTwice",
    "LooseRefinement",
    "LookalikePair",
    "NilEffect",
    "WatchesElsewhere",
    "RebindOneClass",
    "valMachine",
    "valComposition",
    "UnobservedRefiner",
    "IndexedQueries$.queries"
  ).map(rejectsRoot) ++ scriptRejects ++ realizationObjectRejects ++
    sectionRejects ++ hintRejects ++ markerRejects ++ blockRejects ++ statusRejects ++ roleRejects ++
    ruleRejects ++ composedProgressRejects

  private lazy val liftsJar = packaged("lifts", materialize("lifts"))
  private lazy val liftsJars = s"$liftsJar=${stored("lifts")},$modelJar=model/"

  // Each lift is its own JVM, so they run side by side.
  private lazy val lifts: Map[String, Future[Ran]] =
    val jars = liftsJars
    (fixtures :+ ("rejects" -> rejected)).map { (name, roots) =>
      name -> Future(
        blocking(
          lift(
            (Seq(
              jars,
              modelClasspath.toString,
              lifted(name).toString
            ) ++ roots)*
          )
        )
      )
    }.toMap
  private def ran(name: String): Ran = Await.result(lifts(name), munitTimeout)

  // The IR a fixture lifted to.
  private def ir(name: String): String =
    val lift = ran(name)
    assert(
      !lift.failed,
      s"the $name fixture did not lift:\n${lift.diagnostics}"
    )
    Files.readString(lifted(name))

  // The waivers a fixture's capabilities sections wrote beside its IR.
  private def waivers(name: String): String =
    ir(name): Unit
    Files.readString(scratch.resolve(s"$name.waivers.json"))

  // The declarations the lifter refused, one line each, and no IR.
  private def rejections(): String =
    val lift = ran("rejects")
    assert(
      lift.exit != 0 && !Files.exists(
        lifted("rejects")
      ),
      "the lifter wrote the rejected declarations' IR"
    )
    refused(lift).map(_ + "\n").mkString

  // The files of the expected directory that no fixture lifts to.
  private def leftOver(): Seq[String] =
    val stream = Files.list(expected)
    val held =
      try stream.iterator.asScala.map(_.getFileName.toString).toList.sorted
      finally stream.close()
    held.diff(
      fixtures.map(_._1 + ".json") ++ waived.map(
        _ + ".waivers.json"
      ) :+ "rejects.txt"
    )

  // An update rewrites the expected files only once every fixture lifted, every rejected
  // declaration was refused and no expected file is left over, so the tree it leaves is one whole
  // run's. The gate holds model/ir to the same rule.
  private lazy val everyLift: Unit =
    fixtures.foreach((name, _) => ir(name))
    waived.foreach(waivers)
    rejections(): Unit
    assertEquals(
      leftOver(),
      Nil,
      s"${root.relativize(expected)} holds files no fixture lifts to"
    )

  private def expect(file: String, lifted: => String): Unit =
    val path = expected.resolve(file)
    if update then
      everyLift
      if !Files.exists(path) || Files.readString(path) != lifted
      then Files.writeString(path, lifted): Unit
    else
      assert(
        Files.isRegularFile(path) && Files.readString(
          path
        ) == lifted,
        s"${root.relativize(path)} is stale; make umpire-gen-model rewrites it. Lifted: $scratch"
      )

  override def beforeAll(): Unit =
    for input <- Seq(modelJar, modelClasspath) do
      assert(
        Files.isRegularFile(input),
        s"$input is missing: the gate packages the Models before it runs these tests (make umpire-check-model)"
      )
    lifts: Unit
    started.foreach(_())

  concurrently(
    "the build refuses a warning -Werror makes an error, at its line"
  ):
    assertEquals(refusals("werror"), Seq("Steps.scala:21:58"))

  concurrently("the build refuses crossed types, at their lines"):
    assertEquals(
      refusals("crossed").sorted,
      Seq(
        "ActionInput.scala:15:28",
        "Crossed.scala:38:41",
        "Crossed.scala:41:24",
        "Crossed.scala:49:82",
        "NamedInput.scala:21:50",
        "NamedInput.scala:24:27",
        "NamedInput.scala:27:49",
        "NamedInput.scala:30:31",
        "OneChoice.scala:18:62",
        "Progress.scala:28:48",
        "Progress.scala:31:17",
        "Rejections.scala:17:60",
        "Rejections.scala:24:33",
        "Rules.scala:16:32",
        "Sections.scala:22:46",
        "Sections.scala:23:51",
        "Sections.scala:24:45",
        "Statuses.scala:12:3",
        "Sugar.scala:10:27",
        "Sugar.scala:13:86",
        "Sugar.scala:16:71",
        "Sugar.scala:19:68",
        "Sugar.scala:23:69"
      )
    )

  // fn-126 R15: a machine object declares its init, its end and its rules, and a derived machine
  // has none of its own.
  concurrently(
    "the build refuses a machine object without init, end or rules, at its line"
  ):
    assertEquals(
      refusals("objectForms").sorted,
      Seq(
        "Invalid.scala:31:8", // no init
        "Invalid.scala:37:8", // no end
        "Invalid.scala:43:8", // no rules
        "Invalid.scala:49:3" // a derived machine's own rules
      )
    )

  // fn-133 R9: a realization object is typed by its machine (testdata/typedRealizations).
  concurrently(
    "the build refuses a realization object's evidence or status of another machine"
  ):
    assertEquals(
      refusals("typedRealizations").sorted,
      Seq("Typed.scala:16:46", "Typed.scala:25:3")
    )

  // fn-133.8: `.schema[T]` is retired with a migration diagnostic (testdata/retiredSchema).
  concurrently("the build refuses a retired schema, at its line"):
    assertEquals(refusals("retiredSchema"), Seq("Retired.scala:9:3"))

  // fn-139.7: `in(...)` is membership alone; rule cases use `when(...)`.
  concurrently("the build refuses a retired rule-case in with its replacement"):
    val built = bounded(
      tools.scalaCli(Seq("compile", materialize("retiredRuleCase").toString))
    )
    assert(built.failed, "model/irgen/testdata/retiredRuleCase built")
    assertEquals(
      built.errors.collect {
        case line if line.contains("Retired.scala:26:7") => "Retired.scala:26:7"
      },
      Seq("Retired.scala:26:7")
    )
    assert(
      built.output.contains("write `when(...)` for a rule case"),
      built.output
    )

  concurrently(
    "the lifter refuses a compiled retired rule-case in with its replacement"
  ):
    val fixture = "retiredRuleCaseLifter"
    val jar = packaged("retired-rule-case-lifter", materialize(fixture))
    val out = lifted("retired-rule-case-lifter")
    val result = lift(
      s"$jar=${stored(fixture)}",
      modelClasspath.toString,
      out.toString,
      "fixture.retiredrulecaselifter.Retired"
    )
    assert(result.failed, "the lifter accepted the retired rule-case in")
    assert(
      !Files.exists(out),
      "the lifter wrote IR for the retired rule-case in"
    )
    val diagnostic = refused(result).mkString("\n")
    assert(diagnostic.contains("Retired.scala:30:"), diagnostic)
    assert(
      diagnostic.contains(
        "a case says where its action fires, `when(phases)`, `when(states.set)`, " +
          "`where(g)` or `always`"
      ),
      diagnostic
    )

  concurrently("the build refuses a non-finite state field, at its line"):
    assertEquals(refusals("nonfinite"), Seq("NonFinite.scala:5:47"))

  concurrently(
    "typed API declarations and direct constructors refuse mismatched roots"
  ):
    assertEquals(
      refusals("typedInvalid").sorted,
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
        "Invalid.scala:169:26",
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

  concurrently(
    "a realization's reference by value to a missing declaration fails to compile"
  ):
    assertEquals(
      refusals("referenceInvalid").sorted,
      Seq(
        "Invalid.scala:36:32",
        "Invalid.scala:38:44",
        "Invalid.scala:40:38",
        "Invalid.scala:42:41",
        "Invalid.scala:44:63",
        "Invalid.scala:46:38",
        "Invalid.scala:48:28"
      )
    )

  // fn-114 R3: each string-named or string-keyed form the DSL retired, written as it was.
  concurrently(
    "the retired string-named and string-keyed declaration forms do not compile"
  ):
    assertEquals(
      refusals("retiredNames").sorted,
      Seq(
        "Invalid.scala:31:27", // timer("expire")
        "Invalid.scala:34:29", // internal("flush")
        "Invalid.scala:37:20", // hole("crash")
        "Invalid.scala:40:33", // channel[Note]("wire", ...)
        "Invalid.scala:40:41",
        "Invalid.scala:43:38", // Lamp.restrict("pressOnly")(...)
        "Invalid.scala:50:28", // Composition[PairState]("pair")
        "Invalid.scala:51:49", // Composition[PairState]("left" -> ..., ...)
        "Invalid.scala:51:65",
        "Invalid.scala:55:23", // sync("pressBoth", "left" -> ..., ...)
        "Invalid.scala:55:40",
        "Invalid.scala:56:14", // replaces("left", Lamp)
        "Invalid.scala:59:20" // Lamp.scenario.actionKeys("press")
      ).sorted
    )

  concurrently(
    "only the unknown projected origin admits a dynamic message root"
  ):
    assertEquals(refusals("dynamicInvalid"), Seq("Invalid.scala:10:16"))

  // fn-118.2: a hint of a method the generated API lacks, or whose write, read, `when` or bound has
  // the wrong type, a hint built by its constructor, and a server step of no class.
  concurrently(
    "API behavior hints refuse unknown methods and mistyped arguments, at their lines"
  ):
    assertEquals(
      refusals("hintsInvalid").sorted,
      Seq(
        "Invalid.scala:22:20", // WorkflowServiceGrpc.METHOD_DESCRIBE_NOTHING
        "Invalid.scala:23:21", // "StartActivityExecution".visibleTo(...)
        "Invalid.scala:24:36", // start.visibleTo(DescribeActivityExecutionRequest, ...)
        "Invalid.scala:27:76", // visibleTo(describe, WaitBound(100, 1000)): interval
        "Invalid.scala:27:81", // its atMost
        "Invalid.scala:28:19", // Visibility(...)
        "Invalid.scala:30:54", // cause bound: interval
        "Invalid.scala:30:59", // its atMost
        "Invalid.scala:31:48", // boundedBy(Visible.atOnce)
        "Invalid.scala:34:30" // ServerStep(CauseKind.delivery, ...)
      ).sorted
    )

  concurrently(
    "retired string proto constructors refuse direct and helper-built names"
  ):
    val positions = refusals("retiredInvalid")
    assert(positions.size >= 12, positions.mkString(", "))
    assert(
      positions.exists(_.startsWith("Invalid.scala:15:")),
      positions.mkString(", ")
    )
    assert(
      positions.exists(_.startsWith("Invalid.scala:16:")),
      positions.mkString(", ")
    )
    assert(
      positions.exists(_.startsWith("Invalid.scala:29:")),
      positions.mkString(", ")
    )

  concurrently(
    "typed protobuf constants refuse mismatched fields, values and forged carriers"
  ):
    assertEquals(
      refusals("typedProtoInvalid").sorted,
      Seq(
        "Invalid.scala:11:20",
        "Invalid.scala:14:55",
        "Invalid.scala:19:26",
        "Invalid.scala:25:24",
        "Invalid.scala:28:36",
        "Invalid.scala:29:50",
        "Invalid.scala:30:19",
        "Invalid.scala:31:75",
        "Invalid.scala:32:75",
        "Invalid.scala:36:56",
        "Invalid.scala:39:45",
        "Invalid.scala:40:23",
        "Invalid.scala:41:25"
      ).sorted
    )

  // The typed selectors no live Model writes (lifts/Typed.scala); model/ir pins the rest.
  concurrently("typed paths and bound constants lift to their protobuf names"):
    val out = lifted("typed")
    val roots = Seq(
      "fixture.typed.Typed",
      "fixture.typed.Typed$package$.typedRealization"
    )
    val result =
      lift((Seq(liftsJars, modelClasspath.toString, out.toString) ++ roots)*)
    assert(!result.failed, result.diagnostics)
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    val model = mapper.readTree(Files.readString(out))
    val actions = model.path("actions")
    assertEquals(actions.size(), 1)
    // fn-133.8: an action names no carrier; the realization's binding derives it.
    assert(
      actions.get(0).path("schemas").isMissingNode,
      actions.get(0).toPrettyString
    )
    val realizations = model.path("realizations")
    assertEquals(realizations.size(), 1)
    val typed = realizations.get(0)
    val call = typed.at("/scripts/0/items/0/command/rpc")
    assertEquals(
      call.at("/assign/0/target").asText(),
      "heartbeat_timeout.seconds"
    )
    assertEquals(call.at("/assign/0/value/literal/number").asText(), "5")
    assertEquals(call.at("/reads/0/path").asText(), "run_id")
    assertEquals(
      typed.at("/scripts/0/items/1/command/rpc/reads/0/path").asText(),
      "history.events[*].event_id"
    )
    // Through `Option.map` the optional message is selected, not indexed as repeated.
    assertEquals(
      typed.at("/evidence/0/fields/0/path").asText(),
      "delivery_admission.delivery_id"
    )

  test("typed repeated reads preserve bare paths"):
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    val bare = mapper
      .readTree(ir("realizations"))
      .path("realizations")
      .get(1)
      .path("evidence")
      .get(0)
      .path("read")
      .path("path")
      .asText()
    assertEquals(bare, "executions")

  // fn-112.10: a monitor of a Query's expected Run named by value, as by its name. Lifted apart
  // from the realizations fixture: no realization runs the watched door.
  concurrently(
    "a monitor expectation names its monitor by value as by its name"
  ):
    val out = lifted("monitorExpectations")
    val roots =
      Seq("heldByValue", "heldByName").map(
        "fixture.realizations.Realizations$package$." + _
      )
    val result =
      lift((Seq(liftsJars, modelClasspath.toString, out.toString) ++ roots)*)
    assert(!result.failed, result.diagnostics)
    val queries = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(Files.readString(out))
      .path("queries")
    def expected(name: String) = queries
      .elements()
      .asScala
      .find(_.path("name").asText() == name)
      .getOrElse(fail(s"the realizations fixture lifted no Query $name"))
      .path("expectedRun")
    val byValue = expected("heldByValue")
    assertEquals(
      byValue
        .path("monitors")
        .elements()
        .asScala
        .map(_.path("name").asText())
        .toList,
      List("opensOnce", "staysOpen")
    )
    assertEquals(byValue.toPrettyString, expected("heldByName").toPrettyString)

  // fn-160: one IR file per feature, collected from its package by `lift --ir` in one run.
  private def liftIr(out: Path, jars: String, names: String*): Ran =
    lift((Seq("--ir", jars, modelClasspath.toString, out.toString) ++ names)*)

  private def listed(directory: Path): Seq[String] =
    if !Files.isDirectory(directory) then Nil
    else
      val stream = Files.list(directory)
      try stream.iterator.asScala.map(_.getFileName.toString).toList.sorted
      finally stream.close()

  // The phase projection a machine mixes in, with explicit Phased type arguments
  // (testdata/phasedMixin) or a typed lambda (testdata/phasedRules), lifts to one IR, byte for byte
  // (phasedRules/expected.json). Both are lifted at phasedRules' positions on the same lines.
  private lazy val phasedMixinJar =
    packaged("phasedMixin", materialize("phasedMixin"))

  concurrently(
    "Phased with type arguments or a typed lambda lifts byte for byte"
  ):
    val at = stored("phasedRules")
    def switchOf(fixture: String, jar: Path): String =
      val out = scratch.resolve(s"$fixture-out.json")
      val result =
        lift(
          s"$jar=$at,$modelJar=model/",
          modelClasspath.toString,
          out.toString,
          "fixture.phased.Switch"
        )
      assert(!result.failed, result.diagnostics)
      Files.readString(out)
    val written = switchOf(
      "phasedRules",
      packaged("phasedRules", materialize("phasedRules"))
    )
    assertEquals(switchOf("phasedMixin", phasedMixinJar), written)
    val shared = testdata.resolve("phasedRules/expected.json")
    if update then Files.writeString(shared, written): Unit
    else
      assert(
        Files.isRegularFile(shared) && Files.readString(
          shared
        ) == written,
        s"${root.relativize(shared)} is stale; UMPIRE_LIFTER_UPDATE=1 rewrites it"
      )

  concurrently("when names phases no projection reads, refused at its line"):
    val out = scratch.resolve("phasedNone-out.json")
    val result = lift(
      s"$phasedMixinJar=${stored("phasedMixin")},$modelJar=model/",
      modelClasspath.toString,
      out.toString,
      "fixture.phased.Unprojected"
    )
    assertEquals(
      refused(result),
      Seq(
        s"lift: ${stored("phasedMixin")}Phased.scala:61: when names phases, and unprojected reads no " +
          "phase projection: mix it into the machine, `Phased[State, Phase](_.phase)`"
      )
    )

  // A derived machine reads its source's projection, through each derivation, and a derived
  // composition its source's; derivation rules name no phase, so the lifter's record is read, in a
  // lift run here.
  concurrently(
    "a derived machine's and a derived composition's projection is their source's"
  ):
    val tastys = Files.createDirectories(scratch.resolve("phasedTasty"))
    val zip = java.util.zip.ZipFile(phasedMixinJar.toFile)
    val files =
      try
        zip.entries.asScala.toList
          .filter(_.getName.endsWith(".tasty"))
          .map { e =>
            val to = tastys.resolve(e.getName)
            Files.createDirectories(to.getParent)
            Files.copy(zip.getInputStream(e), to): Unit
            to.toString
          }
      finally zip.close()
    val classpath = phasedMixinJar.toString +: Files
      .readString(modelClasspath)
      .trim
      .split(java.io.File.pathSeparator)
      .toList
    val roots =
      Seq("Switch", "Stiffer", "Twins", "Lopsided").map("fixture.phased." + _)
    val lifter =
      Lifter(Target.Roots(roots), files.map(_ -> stored("phasedMixin")).toMap)
    bounded(
      scala.tasty.inspector.TastyInspector
        .inspectAllTastyFiles(files, Nil, classpath)(lifter)
    ): Unit
    assertEquals(lifter.errors.toList, Nil)
    val phases = lifter.phases
    assert(phases("switch").contains("light"), phases.toString)
    assert(phases("twins").contains("left"), phases.toString)
    assertEquals(phases.get("stiffer"), phases.get("switch"))
    assertEquals(phases.get("lopsided"), phases.get("twins"))

  // fn-126 decision 23: an ID derived from a name is unique in its package across the run, even
  // when one shared def declares both records at one position.
  concurrently(
    "the lifter refuses one shared def's Query of one name over two machines of a package"
  ):
    val jar = packaged("derivedTwins", materialize("derivedTwins"))
    val out = scratch.resolve("derivedTwins-out")
    val at = stored("derivedTwins") + "DerivedTwins.scala"
    val result = liftIr(out, s"$jar=${stored("derivedTwins")},$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    assertEquals(
      refused(result),
      Seq(
        "lift: the roots of derivedtwins.json did not lift:",
        s"lift: $at:32: Query anyLit is declared twice, and both would share one Definition ID"
      )
    )
    val models = Seq("Lamp" -> "lampQueries", "PlainLamp" -> "plainLampQueries").map {
      (machine, queries) =>
        val isolated = scratch.resolve(s"derivedTwins-$machine.json")
        val lifted = lift(
          s"$jar=${stored("derivedTwins")},$modelJar=model/",
          modelClasspath.toString,
          isolated.toString,
          s"fixture.features.derivedtwins.$machine$$.queries$$.$queries"
        )
        assertEquals(lifted.exit, 0, lifted.diagnostics)
        val model =
          scalapb.json4s.JsonFormat.fromJsonString[io.temporal.server.api.umpire.v1.Model](
            Files.readString(isolated)
          )
        assertEquals(model.queries.size, 1)
        assertEquals(model.queries.head.name, "anyLit")
        assertEquals(
          model.queries.head.getScenario.machine,
          s"${machine.head.toLower}${machine.tail}"
        )
        assertEquals(model.queries.head.getPosition.file, at)
        assertEquals(model.queries.head.getPosition.line, 32)
        isolated.toString -> model
    }
    assertEquals(models.map(_._2.queries.size), Seq(1, 1))
    assertEquals(
      derivedIdTwins(models).map(e => s"lift: ${e.position}: ${e.message}"),
      Seq(
        s"lift: $at:32: this declaration over plainLamp and the one over lamp derive the ID " +
          "fixture.features.derivedtwins.query.anyLit: name them apart, since an ID derived from a name is " +
          "unique in its package"
      )
    )

  // fn-126 R4: the declaration-order lint, before anything is lifted. Each kind at its line, and
  // none of the reads beside them that initialize nothing yet (initOrder/Forward.scala). Every other
  // lift of these tests, and the gate's of the Models, runs it too and is refused nothing.
  concurrently(
    "the declaration-order lint refuses a read before its declaration, a cycle, an order and a place"
  ):
    val jar = packaged("initOrder", materialize("initOrder"))
    val out = scratch.resolve("initOrder-out")
    val at = stored("initOrder")
    val result = liftIr(out, s"$jar=$at,$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    val forward = (line: Int) =>
      s"lift: ${at}Forward.scala:$line: late is read while Forward initializes, before it is " +
        s"declared at ${at}Forward.scala:18, so it is still null here: declare it before the " +
        "declaration that reads it"
    val feature = s"${at}InitOrder.scala"
    assertEquals(
      refused(result),
      Seq(
        // (a)
        forward(7),
        forward(17),
        // (d): a Model declaration beside the feature file
        s"lift: ${at}Forward.scala:23: stray is a Scenario, which a feature declares in its " +
          s"feature file, $feature: in the `queries` object of its machine's object there",
        s"lift: ${at}Forward.scala:29: strayInCompanion is a Scenario, which a feature declares " +
          s"in its feature file, $feature: in the `queries` object of its machine's object there",
        // (d): a machine in a type's companion, a Property at the top level, a machine object in
        // an object of the signature
        s"lift: $feature:22: BulbLamp holds a Model declaration inside Bulb, the companion of a " +
          "type: a machine object sits at the top level of a feature file, and its sections " +
          "directly in it",
        s"lift: $feature:28: loose is a Property, declared at the top level of a feature file: " +
          "it belongs in the `properties` object of its machine's object",
        s"lift: $feature:32: Inner holds a Model declaration inside Holder, an object of the " +
          "signature: a machine object sits at the top level of a feature file, and its " +
          "sections directly in it",
        // (c)
        s"lift: $feature:68: properties belongs before queries at $feature:65: object Backwards " +
          "reads its header, then states, then refinement, then effects, then monitors, then " +
          "rules, then properties, then capabilities, then queries",
        s"lift: $feature:71: Late belongs before Backwards at $feature:59: a feature file reads " +
          "its header, then its types, then its signature, then its machine and composition " +
          "objects",
        // (d)
        s"lift: $feature:77: extras holds a Model declaration in Misplaced, and is none of its " +
          "sections, states, refinement, effects, monitors, rules, properties, capabilities, " +
          "queries: its declarations belong in them",
        s"lift: $feature:82: lit is a Property, and belongs in the `properties` object of its " +
          "machine's object, not in Misplaced",
        s"lift: $feature:85: switchLit is declared over Switch, a machine object: it belongs in " +
          "Switch.properties",
        // (c): a Scenario after a Query; (d): a Query over another object's Scenario
        s"lift: $feature:99: late belongs before first at $feature:98: Asked.queries reads " +
          "its Scenarios, then its Queries",
        s"lift: $feature:100: borrowed is declared over flipped, which Switch.queries declares: " +
          "it belongs in Switch.queries",
        // (b) through a base class: the body of the class a `capabilities` section extends
        s"lift: ${at}Realization.scala:16: an initialization cycle: Dimmer.capabilities -> " +
          "DimmerRealization -> Dimmer.capabilities, each read while the one before it " +
          "initializes, so one of them is read half made: read it in a def, a lambda or a lazy " +
          "val, or move what is read into an object of its own"
      )
    )

  // fn-126 R4 (e) and R17: each kind at its line, and none of the correct object forms beside them,
  // nor two objects' timers of one name, which their objects name apart (sectionOrder/SectionOrder.scala).
  concurrently(
    "the declaration-order lint refuses sections out of order or out of place"
  ):
    val jar = packaged("sectionOrder", materialize("sectionOrder"))
    val out = scratch.resolve("sectionOrder-out")
    val at = stored("sectionOrder")
    val result = liftIr(out, s"$jar=$at,$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    val f = s"${at}SectionOrder.scala"
    val order =
      "object Backwards reads its header, then states, then refinement, then effects, " +
        "then monitors, then rules, then properties, then capabilities, then queries"
    val handBound =
      "a machine object says when each action fires in its `rules`, " +
        "`on(action) { when(p) ~> effects.x }`, and a derivation binds one in `rebind`"
    assertEquals(
      refused(result),
      Seq(
        s"lift: $f:39: the section capabilities sits at the top level of a feature file: a " +
          "machine's sections sit directly in its machine or composition object",
        s"lift: $f:101: states belongs before effects at $f:98: $order",
        s"lift: $f:107: rules belongs before properties at $f:104: $order",
        s"lift: $f:114: stray is a step function, and belongs in the `effects` object of its " +
          "machine's object, not in Misfiled",
        s"lift: $f:115: watched is a monitor, assumption, hole or channel, and belongs in the " +
          "`monitors` object of its machine's object, not in Misfiled",
        s"lift: $f:116: lit is vocabulary of Misfiled, declared outside its sections: it belongs " +
          "in the `states` object of its machine's object",
        s"lift: $f:117: toProduct is a member of Misfiled's refinement: declare it in " +
          "`object refinement extends Refinement(product)`, which holds the machine's refinement",
        s"lift: $f:118: strayBlock is a step function, and belongs in the `effects` object of its " +
          "machine's object, not in Misfiled",
        s"lift: $f:127: dim is a monitor, assumption, hole or channel, declared in " +
          "Misfiled.properties: it belongs in the `monitors` object of its machine's object",
        s"lift: $f:128: borrowed is declared over Switch, a machine object: it belongs in " +
          "Switch.properties",
        s"lift: $f:135: a step function is bound by hand, `action ~> step`, in HandBound: $handBound",
        s"lift: $f:139: kept is a step function, declared inside HandBound.effects.more: it " +
          "belongs in the `effects` object of its machine's object",
        s"lift: $f:153: armed is read while Guarded.rules initializes, before it is declared at " +
          s"$f:154, so it is still null here: declare it before the declaration that reads it",
        s"lift: $f:159: a step function is bound by hand, `action ~> step`, in Escaped: $handBound",
        s"lift: $f:166: a step function is bound by hand, `action ~> step`, in Cored: $handBound",
        s"lift: $f:174: syncs belongs before properties at $f:171: object BackwardsComposition " +
          "reads its header, then states, then refinement, then effects, then monitors, then " +
          "syncs, then properties, then capabilities, then queries"
      )
    )

  // A Model folder must name its feature file after itself (misnamed/Lamp.scala): a file of another
  // name in a folder with no feature file would otherwise be held to no reading order.
  concurrently(
    "the declaration-order lint refuses a Model in a folder with no feature file"
  ):
    val jar = packaged("misnamed", materialize("misnamed"))
    val out = scratch.resolve("misnamed-out")
    val at = stored("misnamed")
    val result = liftIr(out, s"$jar=$at,$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    val home =
      "a Model folder declares its Models in its feature file, the file named after " +
        s"the folder, in $at"
    assertEquals(
      refused(result),
      Seq(
        s"lift: ${at}Lamp.scala:16: Switch holds a Model declaration in a file not named after " +
          s"its folder: $home",
        s"lift: ${at}Lamp.scala:27: loose is a Property, declared in a file not named after its " +
          s"folder: $home",
        s"lift: ${at}Lamp.scala:32: Spare holds a Model declaration in a file not named after " +
          s"its folder: $home"
      )
    )

  concurrently("the lifter refuses a mapped read ending at a singular message"):
    val out = lifted("typedMapped")
    val result = lift(
      liftsJars,
      modelClasspath.toString,
      out.toString,
      "fixture.typed.Typed$package$.invalidMappedRealization"
    )
    assertNotEquals(result.exit, 0)
    assert(!Files.exists(out), "the lifter wrote an invalid mapped read")
    assert(
      refused(result).exists(
        _.contains("Recorded.read must end at a repeated message field")
      ),
      result.diagnostics
    )

  concurrently(
    "a generated enum helper refuses an unknown value before writing IR"
  ):
    val out = lifted("typedUnknownEnum")
    val result = lift(
      liftsJars,
      modelClasspath.toString,
      out.toString,
      "fixture.typed.Typed$package$.unknownEnumRealization"
    )
    assertNotEquals(result.exit, 0)
    assert(!Files.exists(out), "the lifter wrote an unknown generated enum")
    assert(
      refused(result).exists(_.contains("expected a generated enum case")),
      result.diagnostics
    )

  // Captured.scala takes every name from its val or object, its starts and evidence by default and
  // reads a refinement with no given; expected/captured.json pins its IR. Every symbol-based
  // Definition ID is its declaration's fully qualified name, the objects it sits in included, every
  // top-level type is named in its package and every family is that package (fn-126 decision 23).
  test(
    "the captured fixture's IDs, type names and families are where each is declared"
  ):
    val model =
      new com.fasterxml.jackson.databind.ObjectMapper().readTree(ir("captured"))
    val pkg = "fixture.captured."
    for
      kind <- Seq(
        "actions",
        "monitors",
        "assumptions",
        "holes",
        "channels",
        "realizations"
      )
      declared = model.path(kind)
    do
      assert(declared.size() > 0, s"Captured.scala declares no $kind")
      for d <- declared.elements().asScala do
        val id = d.path("id").asText()
        assert(
          id.startsWith(pkg) && !id.contains("$"),
          s"$id is not a qualified name in $pkg"
        )
    // A monitor at the top level, and one in `object Watched`.
    assertEquals(
      model
        .path("monitors")
        .elements()
        .asScala
        .map(_.path("id").asText())
        .toList
        .sorted,
      List("Watched.storedTwice", "storedOnce").map(pkg + _)
    )
    // Its actions sit in an actor object and in plain objects, which each name; the actor object
    // names the actor.
    assertEquals(
      model
        .path("actions")
        .elements()
        .asScala
        .map(a => a.path("id").asText() -> a.path("actor").asText())
        .toList
        .sorted,
      List(
        "background.expire" -> "system",
        "background.flush" -> "system",
        "client.put" -> "client",
        "faults.crash" -> "fault",
        "relaying.send" -> "client",
        "wire.deliver" -> "system",
        "wire.lose" -> "system"
      ).map((name, actor) => (pkg + name) -> actor).sorted
    )
    val types =
      model.path("types").elements().asScala.map(_.path("name").asText()).toList
    assert(types.nonEmpty, "Captured.scala declares no types")
    for name <- types do assert(name.startsWith(pkg), s"$name is not named in $pkg")
    for
      kind <- Seq("machines", "compositions");
      m <- model.path(kind).elements().asScala
    do
      assertEquals(
        m.path("family").asText(),
        "fixture.captured",
        m.path("name").asText()
      )

  // Sections.scala: a top-level action is named in its package, one in a top-level object by the
  // object too, and one in an actor directly in a machine's object by both objects; two objects tell
  // their actions of one name apart (fn-126 decision 23).
  test(
    "an action is named after the package and the objects it is declared in"
  ):
    val model =
      new com.fasterxml.jackson.databind.ObjectMapper().readTree(ir("sections"))
    val actions = model
      .path("actions")
      .elements()
      .asScala
      .map(a => a.path("id").asText() -> a.path("actor").asText())
      .toMap
    assertEquals(
      actions,
      Map(
        "fixture.sections.reset" -> "system",
        "fixture.sections.panel.flip" -> "panel",
        "fixture.sections.Switch.operator.press" -> "operator",
        "fixture.sections.leftHand.clap" -> "fixture",
        "fixture.sections.rightHand.clap" -> "fixture"
      )
    )

  // A machine object's members name its state type through the machine, `State`, as the type it
  // stands for: `def broken(s: State)` lifts as `def brokenLamp(s: Lamp)` does (lifts/Rules.scala).
  test(
    "a state type named through its machine, State, lifts as the type it stands for"
  ):
    import com.fasterxml.jackson.databind.node.ObjectNode
    val model =
      new com.fasterxml.jackson.databind.ObjectMapper().readTree(ir("rules"))
    def function(name: String) =
      val f = model
        .path("functions")
        .elements()
        .asScala
        .find(
          _.path("name").asText() == s"fixture.rules.Switch$$.states$$.$name"
        )
        .getOrElse(fail(s"no function $name"))
        .deepCopy[ObjectNode]()
      f.remove(java.util.List.of("name", "position"))
      f.findParents("position").asScala.foreach {
        case o: ObjectNode => o.remove("position"): Unit
        case _             => ()
      }
      f.toPrettyString
    assertEquals(function("broken"), function("brokenLamp"))

  // The root of the machine object `fixture.<fixture>.<Name>` a machine `name` is declared by.
  private def objectRoot(fixture: String)(name: String): String =
    s"fixture.$fixture.${name.capitalize}"

  // The machines and Properties of one lift of `roots` of the lifts fixture `fixture`, by name, each
  // as a pair compares it: without its name, its machine and positions, and with each function it
  // refers to in place of the function's name, so two spellings may name their functions apart. A
  // root of the Temporal Models, and a machine object of the fixture, is named in full.
  private def declarations(
      fixture: String,
      roots: Seq[String]
  ): (
      com.fasterxml.jackson.databind.JsonNode,
      String => String,
      String => String
  ) =
    import com.fasterxml.jackson.databind.JsonNode
    import com.fasterxml.jackson.databind.node.ObjectNode
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    val pkg = s"fixture.$fixture.${fixture.capitalize}$$package$$"
    val out = lifted(s"$fixture-pairs")
    val named =
      roots.map(r =>
        if r.startsWith("temporal.") || r.startsWith("fixture.") then r
        else s"$pkg.$r"
      )
    val result = lift(
      (Seq(liftsJars, modelClasspath.toString, out.toString) ++ named)*
    )
    assert(!result.failed, result.diagnostics)
    val model = mapper.readTree(Files.readString(out))
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
        .getOrElse(fail(s"the $fixture fixture lifted no $kind named $name"))
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

  // Each derived machine beside the machine it stands for, spelled out (lifts/Derived.scala).
  concurrently(
    "rebind, extend, refining, assuming and unmonitored lift to the machines they stand for"
  ):
    val pairs = Seq(
      "stiffLamp" -> "stiffLampSpelled",
      "faultyLamp" -> "faultyLampSpelled",
      "plainLamp" -> "plainLampSpelled",
      "plainStiffLamp" -> "plainStiffLampSpelled",
      "stiffPressOnly" -> "stiffPressOnlySpelled"
    )
    val (model, machine, _) =
      declarations(
        "derived",
        pairs.flatMap((a, b) => Seq(a, b)).map(objectRoot("derived"))
      )
    for (derived, spelled) <- pairs do assertEquals(machine(derived), machine(spelled), derived)
    def named(name: String) =
      model
        .path("machines")
        .elements()
        .asScala
        .find(_.path("name").asText() == name)
        .get
    def actions(name: String) = named(name)
      .path("steps")
      .elements()
      .asScala
      .map(s => s.path("action").asText().split('.').last)
      .toList
    assertEquals(
      actions("stiffLamp"),
      List("press", "wear"),
      "a rebound action keeps its place"
    )
    assertEquals(actions("faultyLamp"), List("press", "wear", "burnOut"))
    assertEquals(named("faultyLamp").path("family").asText(), "fixture.derived")
    assertEquals(
      named("faultyLamp")
        .path("assumes")
        .elements()
        .asScala
        .map(_.asText().split('.').last)
        .toList,
      List("lampOpaque", "burnOutAssumed")
    )
    assertEquals(
      named("faultyLamp").path("refines").path("product").asText(),
      "viewUnderFaults"
    )
    assert(
      !named("plainLamp").has("monitors") && !named("plainLamp").has("refines")
    )
    assertEquals(
      named("plainStiffLamp").path("steps").get(1).path("function").asText(),
      "plainStiffLamp.wear",
      "a lambda a derivation binds is named after the machine it declares"
    )

  // fn-139.3: the rules of `Grouped`, in `from` blocks with `when` cases and an `on` of two
  // classes, beside `Plain`'s, in `on` blocks with `in` cases (lifts/Grouped.scala).
  concurrently("from, when and an on of several targets lift as on and in do"):
    val (_, machine, _) =
      declarations(
        "grouped",
        Seq("Grouped", "Plain").map(objectRoot("grouped"))
      )
    assertEquals(
      machine("grouped")
        .replace("fixture.grouped.Grouped$", "fixture.grouped.Plain$"),
      machine("plain")
    )

  // fn-159.7: `enters(Phase.x)` in a rule lifts as the steps of the effect it stands for, and in a
  // choose alternative as the step written out (lifts/Entering.scala).
  concurrently(
    "enters lifts as the effect that sets the phase, in a rule and in a choose"
  ):
    import com.fasterxml.jackson.databind.JsonNode
    import com.fasterxml.jackson.databind.node.ObjectNode
    val (model, _, _) =
      declarations(
        "entering",
        Seq("Entering", "Cored").map(objectRoot("entering"))
      )
    def body(name: String): JsonNode =
      val f = model
        .path("functions")
        .elements()
        .asScala
        .find(_.path("name").asText() == name)
        .getOrElse(fail(s"no function $name"))
        .deepCopy[ObjectNode]()
      f.findParents("position").asScala.foreach {
        case o: ObjectNode => o.remove("position"): Unit
        case _             => ()
      }
      f.path("body")
    def text(n: JsonNode) =
      n.toPrettyString.replace(
        "fixture.entering.Cored$",
        "fixture.entering.Entering$"
      )
    // The arm of each rule: the then of each `if` in the order the rules are written.
    def arms(n: JsonNode): List[JsonNode] =
      val cond = n.path("if")
      if cond.isMissingNode then Nil
      else cond.path("then") :: arms(cond.path("else"))
    val (entering, cored) =
      (arms(body("entering.rules.press")), arms(body("cored.rules.press")))
    assertEquals(entering.size, 2)
    assertEquals(
      text(entering.head),
      text(body("fixture.entering.Cored$.effects$.work"))
    )
    assertEquals(text(entering(1)), text(cored(1)))
    assertEquals(
      text(arms(body("entering.rules.drop")).head),
      text(body("fixture.entering.Cored$.effects$.settle"))
    )
    assertEquals(
      text(body("fixture.entering.Entering$.effects$.restOrRetry")),
      text(body("fixture.entering.Cored$.effects$.restOrRetry"))
    )

  // fn-161.1: a machine assembled from facets lifts as the machine written as one object, but for
  // the names of its effects, which each facet's object qualifies (lifts/Facets.scala).
  concurrently(
    "a machine of facets lifts as the machine written as one object"
  ):
    val (model, machine, property) = declarations(
      "facets",
      Seq("Faceted", "Whole").map(objectRoot("facets")) ++
        Seq("Faceted", "Whole").flatMap(o =>
          Seq(
            s"fixture.facets.$o$$.capabilities",
            s"fixture.facets.$o$$.queries"
          )
        )
    )
    def named(m: String) =
      machine(m).replaceAll(
        "fixture\\.facets\\.[A-Za-z$.]*effects\\$\\.",
        "effects."
      )
    assertEquals(named("faceted"), named("whole"))
    // A facet's capabilities and Property are its machine's, as Whole's own are.
    assertEquals(property("smashes"), property("smashesWhole"))
    def generated(m: String) = model
      .path("properties")
      .elements()
      .asScala
      .filter(_.path("machine").asText() == m)
      .map(_.path("name").asText().stripPrefix(m + "."))
      .toSeq
      .sorted
    assertEquals(
      generated("faceted").filterNot(_.startsWith("smashes")),
      Seq("closedIsRejectedUniformly", "terminalStatesAreFinal")
    )
    assertEquals(
      generated("faceted").filterNot(_.startsWith("smashes")),
      generated("whole").filterNot(_.startsWith("smashes"))
    )

  // fn-161 R8 and R9: the switched facet rules refuse nothing while off, and each refuses its
  // specimen while on (facetRules/FacetRules.scala).
  concurrently(
    "facets-only and facet-vocabulary refuse their specimens when switched on"
  ):
    val jar = packaged("facetRules", materialize("facetRules"))
    val at = stored("facetRules") + "FacetRules.scala"
    def run(root: String, rules: String*): Ran =
      bounded(
        InProcess.lift(
          tools.directory,
          Seq(
            s"$jar=${stored("facetRules")},$modelJar=model/",
            modelClasspath.toString,
            scratch.resolve(s"facetRules-$root.json").toString,
            s"fixture.facetrules.$root"
          ),
          rules.toSet
        )
      )
    for root <- Seq("Agreeing", "Unfaceted", "Unnamed", "Crossed") do
      val off = run(root)
      assert(!off.failed, s"$root:\n${off.diagnostics}")
    val all = Seq("facets-only", "facet-vocabulary")
    assert(!run("Agreeing", all*).failed)
    val design = run("Design", all*)
    assert(
      !design.failed,
      "a custom-state design keeps its deliberate facet-owner exemption\n" + design.diagnostics
    )
    def refusal(root: String, rules: String*): String =
      val ran = run(root, rules*)
      assert(ran.failed, s"$root lifted with ${rules.mkString(",")} on")
      refused(ran).filter(_.contains(at)).mkString("\n")
    assert(
      refusal("Unfaceted", "facets-only").contains(
        "unfaceted binds its actions outside its facets: its rules are its facets' rules"
      )
    )
    assert(
      refusal("Unnamed", "facet-vocabulary").contains(
        "unnamed's facet smashing is not in its feature's facet vocabulary, " +
          "fixture.facetrules.FacetName (pressing, breaking)"
      )
    )
    assert(
      refusal("Crossed", "facet-vocabulary").contains(
        "press is owned by the facet breaking of crossed and by the facet pressing of agreeing, " +
          "which it refines"
      )
    )

  // Each sugar form beside its core spelling (lifts/Sugar.scala).
  concurrently(
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

  // fn-135.2: each `is { }` val of `Blocked` beside the def of `Defined` it stands for, read by a
  // rule's condition, a capability field, `end` and a claim (lifts/Blocks.scala). Every declaration
  // of one machine, with the other's name in its place and no positions, is the other's. fn-135.3:
  // so is each `effect { }` val beside its def, bound by a rule.
  test(
    "is and effect blocks lift as the defs they stand for, in a condition, a capability field, " +
      "end, a claim and a rule's effect"
  ):
    import com.fasterxml.jackson.databind.JsonNode
    import com.fasterxml.jackson.databind.node.ObjectNode
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    val model = mapper.readTree(ir("blocks"))
    def strip(n: JsonNode): Unit =
      n match
        case o: ObjectNode =>
          o.remove(java.util.List.of("position", "source")): Unit
        case _ => ()
      n.elements().asScala.foreach(strip)
    // The declarations of each kind that name the machine, as the other machine's, by their names.
    def declarations(machine: String): Map[String, Map[String, String]] =
      val named =
        (s: String) => s.replace(machine, "Defined").replace(machine.toLowerCase, "defined")
      Seq("machines", "functions", "properties", "scenarios", "queries").map { kind =>
        kind -> model
          .path(kind)
          .elements()
          .asScala
          .filter(_.toString.toLowerCase.contains(machine.toLowerCase))
          .map { d =>
            val copy = mapper.readTree(named(d.toString))
            strip(copy)
            Seq("name", "id")
              .map(copy.path(_).asText())
              .mkString("/") -> copy.toPrettyString
          }
          .toMap
      }.toMap
    val (blocked, defined) = (declarations("Blocked"), declarations("Defined"))
    for kind <- blocked.keys do assertEquals(blocked(kind), defined(kind), kind)
    assertEquals(
      blocked("functions").keySet.filter(_.contains("states")),
      Set("paused", "busy", "over").map(p => s"fixture.blocks.Defined$$.states$$.$p/"),
      "each is block is a function of its own"
    )
    assertEquals(
      blocked("functions").keySet.filter(_.contains("effects")),
      Set("start", "restart", "finish", "pause", "keep", "refuse")
        .map(e => s"fixture.blocks.Defined$$.effects$$.$e/"),
      "each effect block is a function of its own"
    )
    assert(
      blocked("properties").size == 2,
      blocked("properties").keys
    )

  // fn-135.5: each `effect { }` val of `Derived`, which records the status of the phase it assigns,
  // beside the def of `Written` that records it by hand (lifts/StatusFacts.scala). Every machine and
  // function of one, with the other's name in its place and no positions, is the other's, and the
  // phase enum that declares the statuses lifts as an enum of cases with no fields.
  test(
    "an effect block's derived status facts lift as the facts its method form writes"
  ):
    import com.fasterxml.jackson.databind.JsonNode
    import com.fasterxml.jackson.databind.node.ObjectNode
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    val model = mapper.readTree(ir("statusFacts"))
    def strip(n: JsonNode): Unit =
      n match
        case o: ObjectNode =>
          o.remove(java.util.List.of("position", "source")): Unit
        case _ => ()
      n.elements().asScala.foreach(strip)
    def declarations(machine: String): Map[String, String] =
      Seq("machines", "functions").flatMap { kind =>
        model
          .path(kind)
          .elements()
          .asScala
          .filter(_.toString.toLowerCase.contains(machine.toLowerCase))
          .map { d =>
            val named = d.toString
              .replace(machine, "Written")
              .replace(machine.toLowerCase, "written")
            val copy = mapper.readTree(named)
            strip(copy)
            s"$kind/${copy.path("name").asText()}" -> copy.toPrettyString
          }
      }.toMap
    val (derived, written) = (declarations("Derived"), declarations("Written"))
    assertEquals(derived, written)
    assertEquals(
      derived.keySet.filter(_.contains("effects")),
      Set("start", "resume", "retry", "pause", "refuse")
        .map(e => s"functions/fixture.statusfacts.Written$$.effects$$.$e"),
      "each effect block is a function of its own"
    )
    val phase = model
      .path("types")
      .elements()
      .asScala
      .find(
        _.path("name").asText() == "fixture.statusfacts.Phase"
      )
      .get
    assertEquals(
      phase.path("enum").toString,
      """{"cases":[{"name":"idle"},{"name":"running"},{"name":"paused"}]}""",
      "a phase that declares statuses lifts as the enum it is without them"
    )

  // fn-136.1: each role test of `Roled` beside the cases `Listed` names by hand, and fn-136.4's
  // `when[R]` and `p.in[R]` of `Shortened` beside `Written` (lifts/Roles.scala). Every declaration
  // of one machine, with the other's name in its place and no positions, is the other's, and an
  // enum's roles leave its IR type as it is without them.
  test("role prefix compatibility agrees on every finite phase and state"):
    val classpath = liftsJar.toString + java.io.File.pathSeparator +
      Files.readString(modelClasspath).trim
    val result = tools.run(
      Path.of(System.getProperty("java.home"), "bin", "java").toString,
      Seq("-cp", classpath, "fixture.roles.verifyRolePrefixes")
    )
    assertEquals(result.exit, 0, result.output)
    assert(
      result.output.contains("role prefixes equivalent: 8 phases, 64 states"),
      result.output
    )

  test(
    "a role test lifts as the explicit list or alternatives of cases that have the role"
  ):
    import com.fasterxml.jackson.databind.JsonNode
    import com.fasterxml.jackson.databind.node.ObjectNode
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    val model = mapper.readTree(ir("roles"))
    def strip(n: JsonNode): Unit =
      n match
        case o: ObjectNode =>
          o.remove(java.util.List.of("position", "source")): Unit
        case _ => ()
      n.elements().asScala.foreach(strip)
    def declarations(
        machine: String,
        as: String
    ): Map[String, Map[String, String]] =
      val named = (s: String) => s.replace(machine, as).replace(machine.toLowerCase, as.toLowerCase)
      Seq("machines", "functions").map { kind =>
        kind -> model
          .path(kind)
          .elements()
          .asScala
          .filter(_.toString.toLowerCase.contains(machine.toLowerCase))
          .map { d =>
            val copy = mapper.readTree(named(d.toString))
            strip(copy)
            copy.path("name").asText() -> copy.toPrettyString
          }
          .toMap
      }.toMap
    val (roled, listed) =
      (declarations("Roled", "Listed"), declarations("Listed", "Listed"))
    for kind <- roled.keys do assertEquals(roled(kind), listed(kind), kind)
    val (short, written) =
      (declarations("Shortened", "Written"), declarations("Written", "Written"))
    for kind <- short.keys do assertEquals(short(kind), written(kind), kind)
    assertEquals(
      short("functions").keySet.filter(_.contains("rules")),
      Set("written.rules.start", "written.rules.stop"),
      "each when[R] is lifted"
    )
    assertEquals(
      roled("functions").keySet.filter(_.contains("states")),
      Set(
        "closed",
        "waiting",
        "live",
        "expiring",
        "over",
        "settled",
        "stopping"
      )
        .map(p => s"fixture.roles.Listed$$.states$$.$p"),
      "each role test is lifted"
    )
    def cases(name: String) =
      model
        .path("types")
        .elements()
        .asScala
        .find(_.path("name").asText() == name)
        .get
        .path("enum")
    assertEquals(cases("fixture.roles.Phase"), cases("fixture.roles.Bare"))

  // fn-139.1: each step function of `DoorSystem`, whose rules reject with `rejects`, beside
  // `DoorProduct`'s, whose rules name effects that spell `reject(Outcome.rejected(r), s)`: with each
  // call of such an effect replaced by its body, and no names or positions, the two are one but for
  // the explanation `because` gives (lifts/Rejections.scala).
  test(
    "rejects lifts as the reject of the shared rejected outcome, explained by its because"
  ):
    import com.fasterxml.jackson.databind.JsonNode
    import com.fasterxml.jackson.databind.node.ObjectNode
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    val model = mapper.readTree(ir("rejections"))
    val functions =
      model
        .path("functions")
        .elements()
        .asScala
        .map(f => f.path("name").asText() -> f)
        .toMap
    val effects = "fixture.rejections.DoorProduct$.effects$."
    // Each node without positions, and each call of a rejecting effect replaced by its body, whose
    // parameter reads as the argument the call passes.
    def spelled(
        n: JsonNode,
        vars: Map[String, JsonNode] = Map.empty
    ): JsonNode =
      val callee = n.path("call").path("function").asText()
      if Seq("locked", "missing").exists(e => callee == effects + e)
      then
        val f = functions(callee)
        val param = f.path("params").get(0).path("name").asText()
        spelled(
          f.path("body"),
          Map(param -> spelled(n.path("call").path("args").get(0), vars))
        )
      else
        n match
          case o: ObjectNode if vars.contains(o.path("var").asText()) =>
            vars(o.path("var").asText())
          case o: ObjectNode =>
            val copy = o.deepCopy[ObjectNode]()
            copy.remove(java.util.List.of("position", "name"))
            copy
              .fields()
              .asScala
              .toList
              .foreach(e => copy.set(e.getKey, spelled(e.getValue, vars)): Unit)
            copy
          case a: com.fasterxml.jackson.databind.node.ArrayNode =>
            val copy = mapper.createArrayNode()
            a.elements().asScala.foreach(e => copy.add(spelled(e, vars)))
            copy
          case other => other
    def step(machine: String, action: String): String =
      spelled(functions(s"$machine.rules.$action")).toPrettyString
        .replace("DoorSystem$", "DoorProduct$")
    val explained = "\"text\" : \"the door is locked\""
    assert(
      step("doorSystem", "open").contains(explained),
      step("doorSystem", "open")
    )
    assertEquals(
      step("doorSystem", "open").replace(explained, "\"text\" : \"\""),
      step("doorProduct", "open")
    )
    for action <- Seq("close", "ring") do
      assertEquals(
        step("doorSystem", action),
        step("doorProduct", action),
        action
      )
    assert(step("doorProduct", "ring").contains("\"case\" : \"rejected\""))

  // fn-112.9: the script helpers and the Temporal kit beside the core records they stand for, and
  // the kit's `field(_.name) :=` beside `Assignment.typed` (lifts/Scripts.scala).
  concurrently(
    "script helpers, the Temporal kit and field := lift as the core records they stand for"
  ):
    import com.fasterxml.jackson.databind.JsonNode
    import com.fasterxml.jackson.databind.node.ObjectNode
    val pkg = "fixture.scripts.Scripts$package$."
    val out = lifted("scripts-pairs")
    val roots =
      Seq("helpers", "records", "sugaredRequest", "coredRequest").map(pkg + _)
    val result =
      lift((Seq(liftsJars, modelClasspath.toString, out.toString) ++ roots)*)
    assert(!result.failed, result.diagnostics)
    val model = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(Files.readString(out))
    def strip(n: JsonNode): Unit =
      n match
        case o: ObjectNode => o.remove("position"): Unit
        case _             => ()
      n.elements().asScala.foreach(strip)
    def realization(name: String): ObjectNode =
      val r = model
        .path("realizations")
        .elements()
        .asScala
        .find(_.path("name").asText() == name)
        .getOrElse(fail(s"the scripts fixture lifted no realization $name"))
        .deepCopy[ObjectNode]()
      r.remove(java.util.List.of("id", "name"))
      strip(r)
      r
    val helpers = realization("helpers")
    // This comparison is about the script helper surface. The Temporal kit now adds its shared
    // rejection metadata centrally, while the core-record spelling intentionally does not.
    helpers.remove("rejectionCodes")
    assertEquals(helpers.toPrettyString, realization("records").toPrettyString)
    def request(name: String) =
      realization(name).at("/scripts/0/items/0/command/rpc").toPrettyString
    assert(
      request("sugaredRequest").contains("task_queue.name"),
      request("sugaredRequest")
    )
    assertEquals(request("sugaredRequest"), request("coredRequest"))

  // fn-133.1: the kit's literal and call scopes beside the core records they stand for
  // (lifts/Literals.scala).
  concurrently(
    "proto, read, extended and a message for a request field lift as their core forms"
  ):
    import com.fasterxml.jackson.databind.JsonNode
    import com.fasterxml.jackson.databind.node.ObjectNode
    val pkg = "fixture.literals.Literals$package$."
    val out = lifted("literals-pairs")
    val roots = Seq("literalSugared", "literalCored").map(pkg + _)
    val result =
      lift((Seq(liftsJars, modelClasspath.toString, out.toString) ++ roots)*)
    assert(!result.failed, result.diagnostics)
    val model = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(Files.readString(out))
    def strip(n: JsonNode): Unit =
      n match
        case o: ObjectNode => o.remove("position"): Unit
        case _             => ()
      n.elements().asScala.foreach(strip)
    def realization(name: String): JsonNode =
      val r = model
        .path("realizations")
        .elements()
        .asScala
        .find(_.path("name").asText() == name)
        .getOrElse(fail(s"the literals fixture lifted no realization $name"))
        .deepCopy[ObjectNode]()
      r.remove(java.util.List.of("id", "name"))
      strip(r)
      r
    val sugared = realization("literalSugared")
    assertEquals(
      sugared.at("/scripts/0/items").size(),
      7,
      sugared.toPrettyString
    )
    assertEquals(
      sugared.toPrettyString,
      realization("literalCored").toPrettyString
    )

  concurrently(
    "equivalent authored comparison populations retain strict duplicate refusals"
  ):
    for (fixture, roots) <- Seq(
        "inputs" -> Seq("urgentByNameQuery", "urgentByPositionQuery"),
        "patterns" -> Seq("claims")
      )
    do
      val out = lifted(s"$fixture-combined-duplicate")
      val owner = if fixture == "inputs" then "Inputs" else "Patterns"
      val result = lift(
        (Seq(liftsJars, modelClasspath.toString, out.toString) ++
          roots.map(name => s"fixture.$fixture.$owner$$package$$.$name"))*
      )
      assert(result.failed, s"$fixture combined equivalent declarations lifted")
      assert(result.output.contains("states the promise"), result.output)

  // fn-112.5: input tokens, inputs supplied by name and a bounded counter (lifts/Inputs.scala).
  concurrently(
    "inputs supplied by name lift as their positional calls, and UpTo as the Int range"
  ):
    import com.fasterxml.jackson.databind.JsonNode
    val (model, _, property) = declarations(
      "inputs",
      Seq(
        "byNameQuery",
        "byPositionQuery",
        "urgentByNameQuery",
        "omittedQuery",
        "omittedByPositionQuery"
      )
    )
    val (_, _, positionalProperty) =
      declarations("inputs", Seq("urgentByPositionQuery"))
    def named(kind: String, name: String): JsonNode = model
      .path(kind)
      .elements()
      .asScala
      .find(_.path("name").asText() == name)
      .getOrElse(fail(s"the inputs fixture lifted no $kind named $name"))
    def actionsOf(scenario: String): String =
      named("scenarios", scenario).path("actions").toPrettyString
    // A call by name is its positional twin, class by class: partial, reordered and defaulted.
    assertEquals(actionsOf("byName"), actionsOf("byPosition"))
    // fn-112.10: a call that omits every input is its positional twin at every first value.
    assertEquals(actionsOf("omitted"), actionsOf("omittedByPosition"))
    assertEquals(
      property("urgentByName"),
      positionalProperty("urgentByPosition")
    )
    val respond = named("actions", "respond")
    assertEquals(
      respond
        .path("inputs")
        .elements()
        .asScala
        .map(_.path("name").asText())
        .toList,
      List("answer", "urgent"),
      "an input takes its token's val name"
    )
    // The control action reports results by name with no enum declared of that name.
    assertEquals(named("actions", "steer").path("results").asText(), "Delivery")
    assert(
      !model
        .path("types")
        .elements()
        .asScala
        .exists(_.path("name").asText().endsWith(".Delivery")),
      "results named a type"
    )
    // UpTo[2] lifts as the Int range 0..2, the record the live protocol state's attempt counter
    // lifts to in model/ir/activity-standalone.json, which the gate holds.
    assertEquals(
      named("types", "fixture.inputs.CountedState")
        .at("/record/fields/1/type")
        .toString,
      """{"intRange":{"high":"2"}}"""
    )

  // fn-112.11: the total each Query asserts (lifts/Totals.scala).
  concurrently(
    "a Query's total lifts infix, dotted, around expect and from a shared def's argument"
  ):
    import com.fasterxml.jackson.databind.node.ObjectNode
    val (model, _, _) =
      declarations("totals", Seq("queries", "lampTotals", "plainLampTotals"))
    def record(name: String): ObjectNode =
      val q = model
        .path("queries")
        .elements()
        .asScala
        .find(_.path("name").asText() == name)
        .getOrElse(fail(s"the totals fixture lifted no Query named $name"))
        .deepCopy[ObjectNode]()
      q.remove(java.util.List.of("name", "position"))
      q
    def total(name: String): String = record(name).path("total").asText()
    def without(name: String): String =
      val q = record(name)
      q.remove("total")
      q.toPrettyString
    for name <- Seq(
        "infixTotal",
        "dottedTotal",
        "totalThenExpect",
        "expectThenTotal"
      )
    do assertEquals(total(name), "4", name)
    assertEquals(
      record("dottedTotal").toPrettyString,
      record("infixTotal").toPrettyString
    )
    assertEquals(
      record("expectThenTotal").toPrettyString,
      record("totalThenExpect").toPrettyString
    )
    val unexpected = record("totalThenExpect")
    assert(unexpected.has("expectedRun"), "expect left no expected run")
    unexpected.remove("expectedRun")
    assertEquals(unexpected.toPrettyString, record("infixTotal").toPrettyString)
    // Each instance of the shared def carries the total its call supplied, and nothing else apart.
    assertEquals(total("lamp.anyLit"), "12")
    assertEquals(total("plainLamp.anyLit"), "4")
    assertEquals(
      without("plainLamp.anyLit").replace("plainLamp", "lamp"),
      without("lamp.anyLit")
    )
    // A Query that asserts no total takes the count the lifter computes, the one an author writes.
    assertEquals(
      record("countedTotal").toPrettyString,
      record("infixTotal").toPrettyString
    )
    val (counted, _, _) = declarations("totals", Seq("lampCounted"))
    val (plainCounted, _, _) = declarations("totals", Seq("plainLampCounted"))
    def countedTotal(m: com.fasterxml.jackson.databind.JsonNode, name: String) =
      m.path("queries")
        .elements()
        .asScala
        .find(_.path("name").asText() == name)
        .get
        .path("total")
    assertEquals(countedTotal(counted, "lamp.anyLit").asText(), "12")
    assertEquals(countedTotal(plainCounted, "plainLamp.anyLit").asText(), "4")

  // fn-112.12: claims one shared def declares together as a case-class bundle, read back by field
  // (lifts/Totals.scala), lift to the IR of the same claims declared directly on each machine.
  concurrently(
    "a bundle of claims built by its constructor and read by field lifts as its claims"
  ):
    import com.fasterxml.jackson.databind.JsonNode
    import com.fasterxml.jackson.databind.node.ObjectNode
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    def lifting(spelling: String): JsonNode =
      val out = lifted(s"bundle-$spelling")
      val roots = Seq("Lamp", "PlainLamp").map(m => s"fixture.totals.Totals$$package$$.$spelling$m")
      val result =
        lift((Seq(liftsJars, modelClasspath.toString, out.toString) ++ roots)*)
      assert(!result.failed, result.diagnostics)
      mapper.readTree(Files.readString(out).replace(spelling, "direct"))
    def strip(n: JsonNode): Unit =
      n match
        case o: ObjectNode =>
          o.remove(java.util.List.of("position", "source")): Unit
        case _ => ()
      n.elements().asScala.foreach(strip)
    val (bundled, direct) = (lifting("bundled"), lifting("direct"))
    val names = bundled
      .path("properties")
      .elements()
      .asScala
      .map(p => s"${p.path("machine").asText()}.${p.path("name").asText()}")
    assertEquals(
      names.toList.sorted,
      List(
        "lamp.directLit",
        "lamp.directUnlit",
        "plainLamp.directLit",
        "plainLamp.directUnlit"
      )
    )
    strip(bundled)
    strip(direct)
    assertEquals(bundled.toPrettyString, direct.toPrettyString)

  // fn-120.1, fn-120.2: named choices (lifts/Choices.scala). Each name is on the step its alternative
  // wrote, in the order written; an alternative that calls a function calls a copy of it, named
  // `<function>$<choice>`, whose every step carries the name, through the functions it calls in
  // turn, while the function itself keeps its unnamed steps for the other calls.
  concurrently(
    "choose names each alternative's step, and a copy of the function it calls"
  ):
    import com.fasterxml.jackson.databind.JsonNode
    import com.fasterxml.jackson.databind.node.ObjectNode
    val (model, _, _) = declarations("choices", Seq("fixture.choices.Chosen"))
    val prefix = "fixture.choices.Choices$package$."
    def function(name: String): JsonNode = model
      .path("functions")
      .elements()
      .asScala
      .find(
        _.path("name").asText() == prefix + name
      )
      .getOrElse(fail(s"the choices fixture lifted no function $name"))
    def all(n: JsonNode, keep: JsonNode => Boolean): List[JsonNode] =
      (if keep(n) then List(n) else Nil) ++ n
        .elements()
        .asScala
        .toList
        .flatMap(all(_, keep))
    def steps(f: JsonNode) = all(
      f,
      _.at("/construct/type").asText() == "framework.Step"
    )
    def choices(f: JsonNode) = steps(f).map(_.at("/construct/choice").asText())
    def calls(f: JsonNode) =
      all(f, _.has("call")).map(_.at("/call/function").asText())
    val written = Seq(
      "admitNamed" -> List("committed", "redelivered"),
      "pauseNamed" -> List("held", "dropped", "refused", "committed"),
      "pollNamed" -> List("committed", "held", "held", "dropped"),
      "answerNamed" -> List("committed", "refused"),
      "retryNamed" -> List("held"),
      "admitted$committed" -> List("committed", "committed"),
      "admitted$redelivered" -> List("redelivered", "redelivered"),
      "admitted" -> List("", ""),
      "resumeStep" -> Nil
    )
    for (name, expected) <- written do assertEquals(choices(function(name)), expected, name)
    // The alternatives in the order written: a copy's steps, the step written out, a copy's steps.
    val retry = function("retryNamed").path("body")
    assertEquals(retry.at("/binary/op").asText(), "OP_CONCAT")
    assertEquals(
      calls(retry),
      List("admitted$committed", "redeliveredStep$redelivered").map(prefix + _)
    )
    assertEquals(retry.at("/binary/left/binary/right/list/items").size(), 1)
    assertEquals(
      calls(function("redeliveredStep$redelivered")),
      List(prefix + "admitted$redelivered")
    )
    assertEquals(calls(function("resumeStep")), List(prefix + "admitted"))
    // A copy is its function but for the names: without its name, its choices and the suffix of the
    // copies it calls, it is the function, parameters, precondition and every branch alike.
    def unnamed(f: JsonNode, choice: String): JsonNode =
      val copy = f.deepCopy[JsonNode]()
      def strip(n: JsonNode): Unit =
        n match
          case o: ObjectNode =>
            o.remove(java.util.List.of("choice", "name")): Unit
            if o.has("function") then
              o.put(
                "function",
                o.path("function").asText().stripSuffix("$" + choice)
              ): Unit
          case _ => ()
        n.elements().asScala.foreach(strip)
      strip(copy)
      copy
    for (name, choice) <- Seq(
        "admitted" -> "committed",
        "admitted" -> "redelivered",
        "redeliveredStep" -> "redelivered"
      )
    do
      assertEquals(
        unnamed(function(s"$name$$$choice"), choice).toPrettyString,
        unnamed(function(name), choice).toPrettyString,
        s"$name$$$choice"
      )

  // fn-112.4, fn-112.7: typed composition selectors (lifts/Members.scala).
  // The production compositions of temporal/features/activity/standalone, written with typed
  // selectors and derived by `withMember`, keep their exact members, syncs, replacement targets and
  // Scenario keys; and the keys of names that carry separators, over two members that bind actions
  // spelled alike.
  concurrently(
    "typed members, syncs, replaces, withMember, synced and own keep the composed keys"
  ):
    import com.fasterxml.jackson.databind.JsonNode
    // Each design is an object, named as the composition is with the first letter raised: its
    // Queries in its `queries`, or the design itself where no Query runs over it.
    val designs = "temporal.features.activity.standalone.protocol."
    def designObject(d: String) = designs + d.head.toUpper + d.tail
    val overQueue = Seq("recordOverQueue", "trustingRecordOverQueue")
    val overMatching =
      Seq(
        "recordOverMatching",
        "trustingRecordOverMatching",
        "recordOverLossyMatching"
      )
    val unqueried = Seq("recordOverForgetful", "recordOverVolatile")
    val roots =
      (overQueue ++ overMatching).map(d => s"${designObject(d)}$$.queries$$.${d}Queries") ++
        unqueried.map(designObject) ++ Seq("switchQueries", "flickedBothOnce")
    val (model, _, _) = declarations("members", roots)
    def all(kind: String) = model.path(kind).elements().asScala.toList
    def named(name: String)(n: JsonNode) =
      n.path("name").asText() == name
    def texts(n: JsonNode, field: String) =
      n.elements().asScala.map(_.path(field).asText()).toList
    // Each derived design replaces the interface its own queue refines, not its base's.
    val replaced = Map(
      "recordOverQueue" -> "",
      "trustingRecordOverQueue" -> "",
      "recordOverMatching" -> "taskQueueProduct",
      "trustingRecordOverMatching" -> "taskQueueProduct",
      "recordOverForgetful" -> "taskQueueProduct",
      "recordOverVolatile" -> "taskQueueProduct",
      "recordOverLossyMatching" -> "taskQueueProductUnderStorageLoss"
    )
    for (d, target) <- replaced do
      val c =
        all("compositions").find(named(d)).getOrElse(fail(s"no composition $d"))
      assertEquals(
        texts(c.path("members"), "field"),
        List("activity", "queue"),
        d
      )
      assertEquals(
        texts(c.path("syncs"), "name"),
        List("dispatch", "admit", "settle"),
        d
      )
      assertEquals(
        c.path("members").get(1).path("replaces").asText(),
        target,
        d
      )
    def scenarioKeys(machine: String, name: String) = all("scenarios")
      .find(s => s.path("machine").asText() == machine && named(name)(s))
      .getOrElse(fail(s"no Scenario $name of $machine"))
      .path("keys")
      .elements()
      .asScala
      .map(_.asText())
      .toList
    val pause = "activity_pause"
    val invoked = List("dispatch", "queue_addActivityTask")
    for d <- overQueue do
      assertEquals(
        scenarioKeys(d, "staleDeliveryAfterPause"),
        List("dispatch", pause, "admit")
      )
      assertEquals(
        scenarioKeys(d, "admittedBeforePause"),
        List("dispatch", "admit", pause)
      )
      assertEquals(
        scenarioKeys(d, "duplicateDelivery"),
        List("dispatch", "admit", "admit")
      )
    for d <- overMatching do
      val persisted = invoked :+ "queue_persistTask"
      assertEquals(
        scenarioKeys(d, "staleDeliveryAfterPause"),
        persisted ++ List(pause, "admit")
      )
      assertEquals(
        scenarioKeys(d, "admittedBeforePause"),
        persisted ++ List("admit", pause)
      )
      assertEquals(
        scenarioKeys(d, "deliveredAgainAfterLostAck"),
        persisted ++ List("admit", "queue_ackLoss", "admit")
      )
      val matched = invoked :+ "queue_syncMatch"
      assertEquals(
        scenarioKeys(d, "crashAfterAdmissionCommit"),
        matched ++ List(
          "admit",
          "queue_crash",
          "queue_addActivityTask",
          "queue_syncMatch",
          "admit"
        )
      )
    def keys(name: String) =
      all("scenarios")
        .find(named(name))
        .get
        .path("keys")
        .elements()
        .asScala
        .map(_.asText())
        .toList
    assertEquals(
      keys("switchSchedule"),
      List(
        "tap_both-ways",
        "left_side_turn-on-high",
        "left_side_flick",
        "right_side_flick"
      )
    )
    def whenAction(name: String) =
      all("properties").find(named(name)).get.path("whenAction").asText()
    assertEquals(whenAction("turnedOn"), "left_side_turn-on")
    assertEquals(whenAction("tappedBoth"), "tap_both-ways")
    // fn-112.10: a sync named after its first member's action, kept by `withMember` and selected
    // by `synced` from either member.
    for d <- Seq("flicks", "leftFlicks") do
      val c =
        all("compositions").find(named(d)).getOrElse(fail(s"no composition $d"))
      assertEquals(texts(c.path("syncs"), "name"), List("flick"), d)
    assertEquals(keys("bothFlick"), List("flick"))
    assertEquals(whenAction("flickedBoth"), "flick")
    def bound(machine: String) = all("machines")
      .find(named(machine))
      .get
      .path("steps")
      .elements()
      .asScala
      .map(_.path("action").asText().stripPrefix("fixture.members."))
      .toList
    assertEquals(bound("leftSwitch"), List("Left.tap", "Left.flick", "turnOn"))
    assertEquals(bound("rightSwitch"), List("Right.tap", "Right.flick"))

  concurrently(
    "every Temporal realization carries the shared rejection-code table"
  ):
    val rootGroups = Seq(
      Seq(
        "temporal.features.activity.standalone.realization.StandaloneActivityRun",
        "temporal.features.activity.standalone.realization.HeldDispatchRun",
        "temporal.features.activity.standalone.realization.LostStartAnswerRun"
      ),
      Seq(
        "temporal.features.nexus.workflow.realization.WorkflowNexusRun",
        "temporal.features.nexus.workflow.realization.TrustingCallerRun"
      ),
      Seq("temporal.features.nexus.standalone.realization.StandaloneNexusRun")
    )
    val expected = List(
      "REJECTION_NOT_FOUND" -> "NOT_FOUND",
      "REJECTION_ALREADY_EXISTS" -> "ALREADY_EXISTS",
      "REJECTION_FAILED_PRECONDITION" -> "FAILED_PRECONDITION",
      "REJECTION_INVALID_ARGUMENT" -> "INVALID_ARGUMENT"
    )
    val realizations = rootGroups.zipWithIndex.flatMap { (roots, index) =>
      val (model, _, _) = declarations(s"rejectionCodes$index", roots)
      model.path("realizations").elements().asScala.toList
    }
    assertEquals(realizations.size, rootGroups.map(_.size).sum)
    for realization <- realizations do
      val codes = realization
        .path("rejectionCodes")
        .elements()
        .asScala
        .map(code => code.path("rejection").asText() -> code.path("grpcCode").asText())
        .toList
      assertEquals(codes, expected, realization.path("name").asText())

  concurrently(
    "the actual WithTaskQueue capability root folds its typed composed outcome"
  ):
    val (model, _, _) = declarations(
      "withTaskQueueCapability",
      Seq(
        "temporal.features.activity.standalone.protocol.RecordOverQueue$.capabilities"
      )
    )
    assert(model.path("properties").size() > 0)

  // fn-112.12: an independent consumer of the shared task queue (lifts/TaskQueue.scala).
  test(
    "a consumer of temporal/foundations/taskqueue lifts the queue and nothing of the activity"
  ):
    import com.fasterxml.jackson.databind.JsonNode
    val model = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(ir("taskqueue"))
    def all(kind: String) = model.path(kind).elements().asScala.toList
    def named(kind: String, name: String) =
      all(kind)
        .find(_.path("name").asText() == name)
        .getOrElse(fail(s"no $kind $name"))
    def names(n: JsonNode, field: String) =
      n.elements().asScala.map(_.path(field).asText()).toList
    // The queue's declarations are named where they are declared, temporal/foundations/taskqueue.
    val queue = "temporal.foundations.taskqueue."
    assertEquals(
      named("machines", "taskQueueProduct").path("entity").asText(),
      "taskQueue"
    )
    assertEquals(
      named("machines", "taskQueueProduct").path("family").asText(),
      queue + "product"
    )
    assertEquals(
      named("actions", "enqueue").path("id").asText(),
      queue + "queue.enqueue"
    )
    for (c, queue, target) <- Seq(
        ("jobOverQueue", "taskQueueProduct", ""),
        ("jobOverMatching", "taskQueueProtocol", "taskQueueProduct"),
        ("jobOverForgetful", "forgetfulQueue", "taskQueueProduct")
      )
    do
      val members = named("compositions", c).path("members")
      assertEquals(names(members, "machine"), List("job", queue), c)
      assertEquals(members.get(1).path("replaces").asText(), target, c)
    // No name of the activity's package is lifted, only the shared queue's.
    val fromActivity = model.toString.linesIterator
      .flatMap(
        """"(temporal\.features\.activity\.standalone[^"]*)"""".r
          .findAllMatchIn(_)
          .map(_.group(1))
      )
      .toSet
    assert(fromActivity.isEmpty, fromActivity.mkString(", "))
    assert(!all("machines").exists(_.path("entity").asText() == "activity"))

  // fn-112.4: claim patterns and shared claims (lifts/Patterns.scala).
  // Each claim pattern beside its lambda spelling, on the machine and on the composition, and each
  // claim written once over `Declares[S]` beside the lambda it stands for on either.
  concurrently(
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
          combined.path(kind).asInstanceOf[ArrayNode] // scalafix:ok DisableSyntax.asInstanceOf
        piece.path(kind).elements().asScala.foreach(row => rows.add(row): Unit)
      combined
    val model = population(Seq("patternClaims", "inlineClaims", "sharedClaims"))
    val cores = population(Seq("coreClaims", "inlineCoreClaims", "sharedCoreClaims"))
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

  concurrently(
    "the lifter refuses unrelated machines with one state type, at the Query line"
  ):
    val jar = packaged("samestate", materialize("samestate"))
    val lift = this.lift(
      s"$jar=${stored("samestate")}",
      modelClasspath.toString,
      "/dev/null",
      "fixture.samestate.SameState$package$.wrongPair"
    )
    assertNotEquals(lift.exit, 0)
    assertEquals(
      refused(lift),
      Seq(
        "lift: model/irgen/testdata/samestate/SameState.scala:22: wrongPair pairs property, a Property of first, with scenario, a Scenario of second, and reads it through no refinement"
      )
    )

  concurrently(
    "the lifter refuses a construct outside the subset, at its line"
  ):
    val jar = packaged("unsupported", materialize("unsupported"))
    val lift = this.lift(
      s"$jar=${stored("unsupported")}",
      modelClasspath.toString,
      "/dev/null",
      "temporal.fixture.Unsupported"
    )
    assertNotEquals(lift.exit, 0)
    val line =
      "lift: model/irgen/testdata/unsupported/Unsupported.scala:18: `var out` has no IR form"
    refused(lift) match
      case Seq(refusal) => assert(refusal.startsWith(line), refusal)
      case refusals     => fail(s"one refusal, not $refusals")

  // fn-133.3: a feature file writes instructions in the kit's lower-case forms and names commands
  // after their vals (testdata/features).
  concurrently(
    "a feature file's upper-case instruction and written-out command are refused"
  ):
    val jar = packaged("features", materialize("features"))
    val at = stored("features") + "Features.scala"
    val cases = Seq(
      "upperCase" -> (s"lift: ${at}:20: Fault is the core form of an instruction: a feature file " +
        "writes its lower-case form, `fault(...)`"),
      "spelledOut" -> (s"lift: ${at}:25: Command(\"stop-named\", ...) writes a command's name " +
        "out: a feature file names a command after its val, or shares another's with " +
        "`aliasOf(command)(instruction)`")
    )
    for (name, expected) <- cases do
      val out = lifted(s"features-$name")
      val lift = this.lift(
        s"$jar=${stored("features")},$modelJar=model/",
        modelClasspath.toString,
        out.toString,
        s"fixture.features.Features$$package$$.$name"
      )
      assertNotEquals(lift.exit, 0)
      assertEquals(refused(lift), Seq(expected))

  concurrently(
    "the realization emitter refuses unknown constructors and fields at their lines"
  ):
    val jar =
      packaged("realizationRefusals", materialize("realizationRefusals"))
    val cases = Seq(
      (
        "unknownConstructor",
        "lift: model/irgen/testdata/realizationRefusals/Refusals.scala:7: Unknown is no activation of Script in the IR"
      ),
      (
        "unknownField",
        "lift: model/irgen/testdata/realizationRefusals/Refusals.scala:10: Realization has no invented in the IR"
      )
    )
    for (name, expected) <- cases do
      val out = lifted(name)
      val lift = this.lift(
        s"$jar=${stored("realizationRefusals")}",
        modelClasspath.toString,
        out.toString,
        s"fixture.realizationRefusals.Refusals$$package$$.$name"
      )
      assertNotEquals(lift.exit, 0)
      assertEquals(refused(lift), Seq(expected))
      assert(!Files.exists(out), s"the lifter wrote the $name realization's IR")

  for (name, _) <- fixtures do
    test(s"the $name fixture lifts to its expected IR"):
      expect(s"$name.json", ir(name))

  for name <- waived do
    test(
      s"the $name fixture writes the waivers its capabilities sections state"
    ):
      expect(s"$name.waivers.json", waivers(name))

  // fn-134.2 (R4): a Property expanded from a capability Property carries the def's fully
  // qualified name and position as its origin, an overriding def's keeps the one it replaces, and
  // no other Property carries one. Holdable's Property, which reads Takeable's field too, is
  // brought to the machines that declare both, and not to Chore.
  test(
    "a generated Property's origin is the capability Property it was expanded from"
  ):
    val model = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(ir("capabilitySections"))
    val origins = model
      .path("properties")
      .elements()
      .asScala
      .map(p =>
        p.path("name").asText() -> Option
          .when(p.has("origin"))(p.path("origin").path("name").asText())
      )
      .toMap
    val kind = "fixture.capabilitysections."
    assertEquals(
      origins,
      Map(
        "task.sealedStaysSealed" -> Some(kind + "Sealable.sealedStaysSealed"),
        "task.sealedIsRefused" -> Some(kind + "Sealable.sealedIsRefused"),
        "task.heldIsNotTaken" -> Some(kind + "Holdable.heldIsNotTaken"),
        "chore.sealedIsRefused" -> Some(kind + "Sealable.sealedIsRefused"),
        "taskPair.sealedStaysSealed" -> Some(
          kind + "Sealable.sealedStaysSealed"
        ),
        "taskPair.heldIsNotTaken" -> Some(kind + "Holdable.heldIsNotTaken"),
        "mirrorPair.sealedStaysSealed" -> Some(
          kind + "Sealable.sealedStaysSealed"
        )
      )
    )
    val at = "model/irgen/testdata/lifts/CapabilitySections.scala"
    for p <- model.path("properties").elements().asScala do
      assertEquals(p.path("origin").path("position").path("file").asText(), at)
      assertEquals(p.path("position").path("file").asText(), at)

  test(
    "Retryable instances bind their own fields and generate bounded free transition verifies"
  ):
    val model = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(ir("retryCapabilities"))
    val properties = model.path("properties").elements().asScala.toSeq
    val names = properties.map(_.path("name").asText()).toSet
    val ordinary = Set(
      "attemptCountIsWithinPolicy",
      "failureEndsFailed",
      "failureReturnsToWaiting"
    )
    val controlled = ordinary ++ Set("failurePauses", "failureCancels")
    val expected =
      (for instance <- Set("eligible", "network"); property <- ordinary
      yield s"waitingFailures.$instance.$property") ++
        (for instance <- Set("eligible", "fatal"); property <- controlled
        yield s"controlledFailures.$instance.$property") ++
        Set(
          "waitingFailures.terminalStatesAreFinal",
          "waitingFailures.closedIsRejectedUniformly"
        )
    assertEquals(names, expected)
    for
      p <- properties
      if !p.path("name").asText().startsWith("waitingFailures.terminal") &&
        !p.path("name").asText().endsWith("closedIsRejectedUniformly")
    do
      val property = p.path("name").asText().split('.').last
      assertEquals(
        p.path("origin").path("name").asText(),
        s"temporal.capabilities.system.Retryable.$property"
      )
      if property.startsWith("failure") then
        assert(p.path("transition").asBoolean())
        assert(p.has("whenClass"))
    val scenarios = model.path("scenarios").elements().asScala.toSeq
    assertEquals(scenarios.map(_.path("name").asText()).toSet, expected)
    for s <- scenarios do
      assert(s.path("free").asBoolean())
      assert(!s.has("actions"))
    val queries = model.path("queries").elements().asScala.toSeq
    assertEquals(queries.map(_.path("name").asText()).toSet, expected)
    for q <- queries do
      assertEquals(q.path("form").asText(), "FORM_VERIFY")
      assert(!q.has("expectedRun"))
      val steps =
        if q.path("name").asText().endsWith("failureReturnsToWaiting") then 2
        else 3
      assertEquals(q.path("limits").path("steps").asInt(), steps)
      val slots = if q.path("name").asText().startsWith("waitingFailures.") then 54L
      else 240L
      assertEquals(q.path("total").asText().toLong, slots * steps)
    val functions = model
      .path("functions")
      .elements()
      .asScala
      .toSeq
      .map(f => f.path("name").asText() -> f)
      .toMap
    for (instance, count, maximum) <- Seq(
        ("eligible", "attempts", "maxOne"),
        ("network", "otherAttempts", "unlimited")
      )
    do
      val property = properties
        .find(
          _.path("name").asText() ==
            s"waitingFailures.$instance.attemptCountIsWithinPolicy"
        )
        .get
      val read =
        functions(property.path("holds").asText())
          .findValuesAsText("function")
          .asScala
          .toSet
      assert(read.exists(_.endsWith(s"WaitingSteps$$.$count")), read.toString)
      assert(read.exists(_.endsWith(s"WaitingSteps$$.$maximum")), read.toString)
    val selected = properties.filter(p =>
      p.path("name").asText().startsWith("waitingFailures.") &&
        p.path("name").asText().endsWith("failureEndsFailed")
    )
    assertEquals(
      selected.map(_.path("whenClass").path("action").asText()).toSet.size,
      2
    )

  test(
    "an override naming one instance replaces it alone and keeps every other instance's IR"
  ):
    val model = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(ir("retrySelection"))
    val properties = model.path("properties").elements().asScala.toSeq
    val functions =
      model
        .path("functions")
        .elements()
        .asScala
        .toSeq
        .map(f => f.path("name").asText() -> f)
        .toMap
    def property(name: String) = properties
      .find(_.path("name").asText() == name)
      .get
    def reads(name: String) =
      functions(property(name).path("holds").asText())
        .findValuesAsText("function")
        .asScala
        .toSet
    val overridden = property("selectedFailures.fatal.failureEndsFailed")
    assertEquals(
      overridden.path("origin").path("name").asText(),
      "temporal.capabilities.system.Retryable.failureEndsFailed"
    )
    def body(name: String) =
      val f = functions(property(name).path("holds").asText())
        .deepCopy[com.fasterxml.jackson.databind.node.ObjectNode]()
      f.remove("name")
      f.toString
        .replace("selectedFailures", "")
        .replace("controlledFailures", "")
    assertNotEquals(
      body("selectedFailures.fatal.failureEndsFailed"),
      body("controlledFailures.fatal.failureEndsFailed")
    )
    for (selected, base) <- Seq(
        "selectedFailures.eligible.failureEndsFailed" -> "controlledFailures.eligible.failureEndsFailed",
        "selectedFailures.fatal.failurePauses" -> "controlledFailures.fatal.failurePauses"
      )
    do
      def shape(name: String) =
        val p = property(name)
          .deepCopy[com.fasterxml.jackson.databind.node.ObjectNode]()
        p.remove(
          java.util.List.of("name", "machine", "position", "holds", "holds2")
        )
        p
      assertEquals(shape(selected), shape(base))
      assertEquals(reads(selected), reads(base))
    val queries =
      model
        .path("queries")
        .elements()
        .asScala
        .toSeq
        .map(q => q.path("name").asText() -> q)
        .toMap
    for
      name <- properties.map(_.path("name").asText())
      if name.startsWith("selectedFailures.")
    do
      val base =
        queries(name.replace("selectedFailures.", "controlledFailures."))
      assertEquals(queries(name).path("limits"), base.path("limits"))
      assertEquals(queries(name).path("total"), base.path("total"))

  test(
    "Retryable declarations refuse missing roles and invalid or ambiguous bindings at their lines"
  ):
    val cases = Seq(
      ("NoPhase", "capabilities", "noPhase", "no phase"),
      ("NoWaiting", "capabilities", "noWaiting", "Waiting"),
      ("NoFailed", "capabilities", "noFailed", "Failed"),
      ("NoSuspended", "capabilities", "noSuspended", "Suspended"),
      ("NoCanceled", "capabilities", "noCanceled", "Canceled"),
      ("UnboundFailure", "capabilities", "unboundFailure", "failure"),
      ("UnnamedCount", "capabilities", "unnamedCount", "attemptCount"),
      ("UnnamedPause", "capabilities", "unnamedPause", "pendingPause"),
      ("MalformedPause", "capabilities", "malformedPause", "pendingPause"),
      (
        "ComputedClassification",
        "capabilities",
        "computedClassification",
        "retryable"
      ),
      ("AmbiguousWaiver", "capabilities", "ambiguousWaiver", "ambiguous"),
      ("AmbiguousClaim", "queries", "ambiguousClaim", "ambiguous"),
      ("AmbiguousForeign", "capabilities", "ambiguousForeign", "shared"),
      (
        "UnknownInstance",
        "capabilities",
        "unknownInstance",
        "no capability its section declares"
      ),
      (
        "NonMatchingInstance",
        "capabilities",
        "nonMatchingInstance",
        "second does not bring"
      ),
      ("DuplicateInstance", "capabilities", "duplicateInstance", "first twice"),
      ("EmptySelection", "capabilities", "emptySelection", "no instance"),
      ("RedundantSelection", "capabilities", "redundantSelection", "drop `of`")
    )
    for (root, section, machine, field) <- cases do
      val out = lifted(s"retry-refused-$root")
      val ran = lift(
        liftsJars,
        modelClasspath.toString,
        out.toString,
        s"fixture.retrycapabilityrejects.$root$$.$section"
      )
      assertNotEquals(ran.exit, 0, ran.diagnostics)
      assert(!Files.exists(out))
      val errors = refused(ran)
      assertEquals(errors.size, 1, ran.diagnostics)
      val error = errors.head
      assert(
        error.matches(
          "lift: model/irgen/testdata/lifts/RetryCapabilityRejects.scala:[0-9]+: .*"
        ),
        error
      )
      assert(error.contains(machine), error)
      assert(error.contains(field), error)

  test(
    "Expirable instances retain their selected timer, own role, armed predicate and typed fact"
  ):
    val model =
      new com.fasterxml.jackson.databind.ObjectMapper()
        .readTree(ir("deadlineCapabilities"))
    val properties = model.path("properties").elements().asScala.toSeq
    val common = Set("firesInWindow", "deadlineTimesOut")
    val retries = common + "deadlineReturnsToWaiting"
    val names = (for p <- common yield s"deadlineTimers.schedule.$p") ++
      (for p <- retries + "deadlinePauses" yield s"deadlineTimers.start.$p") ++
      (for p <- retries yield s"simpleDeadlines.retry.$p")
    assertEquals(properties.map(_.path("name").asText()).toSet, names)
    val functions = model
      .path("functions")
      .elements()
      .asScala
      .toSeq
      .map(f => f.path("name").asText() -> f)
      .toMap
    for p <- properties do
      val name = p.path("name").asText()
      assert(p.path("transition").asBoolean())
      assert(p.has("whenClass"))
      assertEquals(
        p.path("origin").path("name").asText(),
        s"temporal.capabilities.system.Expirable.${name.split('.').last}"
      )
      assertEquals(
        p.path("position").path("file").asText(),
        "model/irgen/testdata/lifts/DeadlineCapabilities.scala"
      )
      assertEquals(
        p.path("origin").path("position").path("file").asText(),
        "model/temporal/capabilities/system/Expirable.scala"
      )
    for (instance, predicate, phase, kind) <- Seq(
        ("schedule", "dispatchArmed", "waiting", "schedule"),
        ("start", "armed", "held", "start")
      )
    do
      def body(property: String) = functions(
        properties
          .find(
            _.path("name").asText() == s"deadlineTimers.$instance.$property"
          )
          .get
          .path("holds")
          .asText()
      )
      val window = body("firesInWindow")
      assert(
        window
          .findValuesAsText("function")
          .asScala
          .exists(_.endsWith(s"Steps$$.$predicate"))
      )
      assert(
        window.findValuesAsText("case").asScala.contains(phase),
        window.toString
      )
      val terminalBody = body("deadlineTimesOut")
      if instance == "schedule" then
        val classification =
          terminalBody.findValues("literal").asScala.filter(_.has("bool"))
        assertEquals(classification.count(_.path("bool").asBoolean()), 0)
        val declaration = properties
          .find(
            _.path("name").asText() ==
              "deadlineTimers.schedule.deadlineTimesOut"
          )
          .get
          .path("position")
        val defaults = terminalBody
          .findParents("literal")
          .asScala
          .filter(e => e.path("literal").has("bool") && e.path("position") == declaration)
        assertEquals(defaults.size, 1)
        assertEquals(
          defaults.head.path("literal").path("bool").asBoolean(),
          false
        )
      val terminal = terminalBody.findValuesAsText("case").asScala
      assert(terminal.contains(kind), terminal.toString)
      assert(
        !terminal.contains(if kind == "start" then "schedule" else "start"),
        terminal.toString
      )
      val selector = properties
        .find(
          _.path("name").asText() ==
            s"deadlineTimers.$instance.firesInWindow"
        )
        .get
        .path("whenClass")
      assert(
        selector
          .path("action")
          .asText()
          .endsWith(if instance == "schedule" then "scheduleExpired"
          else "startExpired")
      )
      if instance == "schedule" then
        assert(selector.path("inputs").path(0).path("bool").asBoolean())
    for property <- Seq("deadlineReturnsToWaiting", "deadlinePauses") do
      val p = properties
        .find(_.path("name").asText() == s"deadlineTimers.start.$property")
        .get
      val facts =
        functions(p.path("holds").asText()).findValuesAsText("case").asScala
      assert(
        facts.contains("schedule") && facts.contains("start"),
        facts.toString
      )
    val scenarios = model.path("scenarios").elements().asScala.toSeq
    assertEquals(scenarios.map(_.path("name").asText()).toSet, names)
    assert(
      scenarios.forall(s => s.path("free").asBoolean() && !s.has("actions"))
    )
    val queries = model.path("queries").elements().asScala.toSeq
    assertEquals(queries.map(_.path("name").asText()).toSet, names)
    for q <- queries do
      val name = q.path("name").asText()
      assertEquals(q.path("form").asText(), "FORM_VERIFY")
      val steps =
        if name.startsWith("deadlineTimers.") && name.endsWith("firesInWindow")
        then 2
        else 3
      assertEquals(q.path("limits").path("steps").asInt(), steps)
      val combinations =
        if name.startsWith("deadlineTimers.") then 128L * 15L else 2L
      assertEquals(q.path("total").asText().toLong, combinations * steps)

  test(
    "Expirable refuses missing roles, invalid timer bindings and malformed typed fact families"
  ):
    for root <- Seq(
        "NativeTerminalRoles",
        "CompleteSubtypeBinding",
        "CompleteIntersectionBinding"
      )
    do
      val terminalOut = lifted(s"deadline-positive-$root")
      val terminal = lift(
        liftsJars,
        modelClasspath.toString,
        terminalOut.toString,
        s"fixture.deadlinecapabilityrejects.$root$$.capabilities"
      )
      assertEquals(terminal.exit, 0, terminal.diagnostics)
      val terminalModel =
        new com.fasterxml.jackson.databind.ObjectMapper()
          .readTree(Files.readString(terminalOut))
      val machine = root.head.toLower.toString + root.tail
      assertEquals(
        terminalModel
          .path("properties")
          .elements()
          .asScala
          .map(_.path("name").asText())
          .toSet,
        Set(
          s"$machine.deadline.firesInWindow",
          s"$machine.deadline.deadlineTimesOut"
        )
      )
    val cases = Seq(
      "PositiveFamily" -> "negated records",
      "DoubledFamily" -> "negated records",
      "IncompleteSubtypeBinding" -> "timeoutFacts",
      "IncompleteIntersectionBinding" -> "timeoutFacts",
      "WrongTypedFamily" -> "timeoutFacts",
      "NoPhase" -> "no phase",
      "NoCovered" -> "Held",
      "NoTimedOut" -> "TimedOut",
      "NoWaiting" -> "Waiting",
      "NoSuspended" -> "Suspended",
      "ForeignTimer" -> "timer",
      "UnboundTimer" -> "timer",
      "NonTimer" -> "timer",
      "RetryNoEligibility" -> "retriesRemaining",
      "EligibilityNonretry" -> "retriesRemaining",
      "ControlNonretry" -> "pendingCancel",
      "UnnamedArmed" -> "armed",
      "UnnamedPause" -> "pendingPause",
      "MalformedEligibility" -> "retriesRemaining",
      "ComputedClassification" -> "retryable",
      "EmptyFamily" -> "timeoutFacts",
      "IncompleteFamily" -> "timeoutFacts",
      "DuplicateFamily" -> "timeoutFacts",
      "MixedFamily" -> "timeoutFacts",
      "WrongFamily" -> "timeoutFacts",
      "MalformedFamily" -> "timeoutFacts",
      "NonconstantTimeout" -> "timeout",
      "DuplicateTimeout" -> "timeout",
      "AmbiguousWaiver" -> "ambiguous",
      "AmbiguousClaim" -> "ambiguous"
    )
    val source =
      Files
        .readAllLines(testdata.resolve("lifts/DeadlineCapabilityRejects.scala"))
        .asScala
    for (root, field) <- cases do
      val out = lifted(s"deadline-refused-$root")
      val section =
        if root == "AmbiguousClaim" then "queries"
        else "capabilities"
      val result = lift(
        liftsJars,
        modelClasspath.toString,
        out.toString,
        s"fixture.deadlinecapabilityrejects.$root$$.$section"
      )
      assertNotEquals(result.exit, 0, result.diagnostics)
      assert(!Files.exists(out))
      val errors = refused(result)
      assertEquals(errors.size, 1, result.diagnostics)
      val error = errors.head
      assert(
        error.matches(
          "lift: model/irgen/testdata/lifts/DeadlineCapabilityRejects.scala:[0-9]+: .*"
        ),
        error
      )
      assert(error.contains(root.head.toLower.toString + root.tail), error)
      assert(error.contains(field), error)
      if root == "DuplicateTimeout" then
        val start = source.indexWhere(_.startsWith("object DuplicateTimeout"))
        val first = source.indexWhere(_.contains("val first:"), start) + 1
        val second = source.indexWhere(_.contains("val second:"), start) + 1
        assert(error.contains("first") && error.contains("second"), error)
        assert(
          error.contains(s"DeadlineCapabilityRejects.scala:$first") &&
            error.contains(s"DeadlineCapabilityRejects.scala:$second"),
          error
        )

  test(
    "every rejected declaration is refused at its line, and no IR is written"
  ):
    val authored = rejections()
    Files.writeString(scratch.resolve("authored-rejects.txt"), authored): Unit
    expect("rejects.txt", authored)

  test("the complete authored refusal contract is independent of held managed comparisons"):
    val historicalBytes = Files.readAllBytes(expected.resolve("rejects.txt"))
    val digest = java.security.MessageDigest.getInstance("SHA-256").digest(historicalBytes)
    assertEquals(
      digest.map(b => f"${b & 0xff}%02x").mkString,
      "efdd6380f6266a955434f28a2c486065829f077e980b1136939bc1994112bf90"
    )
    val historical = Files.readAllLines(expected.resolve("rejects.txt")).asScala.toVector
    assertEquals(historical.size, 226)
    val prefix = "lift: model/irgen/testdata/lifts/"
    val adaptations = Map(
      (prefix + "HintRejects.scala:44: 9223372036854775807 milliseconds is outside the range of a protobuf Duration") ->
        (prefix + "HintRejects.scala:44: an elapsed bound is outside the protobuf Duration range"),
      (prefix + "ScriptRejects.scala:203: inspect is no action activitySystem binds: a realization binds its own machine's actions") ->
        (prefix + "ScriptRejects.scala:203: inspect is no action activityProtocol binds: a realization binds its own machine's actions"),
      (prefix + "ScriptRejects.scala:105: expires is a case of Timeout, and activitySystem records facts of temporal.features.activity.standalone.system.Fact: a realization names the facts its machine records") ->
        (prefix + "ScriptRejects.scala:105: expires is a case of Timeout, and activityProtocol records facts of temporal.features.activity.standalone.protocol.Fact: a realization names the facts its machine records")
    )
    assert(adaptations.keys.forall(old => historical.count(_ == old) == 1))
    val markerSource = Files.readAllLines(testdata.resolve("lifts/MarkerRejects.scala")).asScala
    val markerDeclarations = Map(
      35 -> "FaultUnmarked",
      41 -> "FaultlessFailure",
      47 -> "HopelessFailure",
      70 -> "UnrefutedControl",
      83 -> "RefinedControl",
      103 -> "SystemicControl",
      123 -> "TornMarkers",
      144 -> "UnmarkedPair",
      150 -> "FaultlessPair",
      158 -> "DerivedControl"
    )
    val markerCoordinates = markerDeclarations.map { (old, name) =>
      val matches = markerSource.zipWithIndex.filter((line, _) =>
        line.startsWith(s"object $name ") || line == s"object $name"
      )
      assertEquals(matches.size, 1, name)
      (prefix + s"MarkerRejects.scala:$old:") -> (prefix + s"MarkerRejects.scala:${matches.head._2 + 1}:")
    }
    def sourceLine(file: String, text: String): Int =
      val source = Files.readAllLines(testdata.resolve("lifts/" + file)).asScala
      val matches = source.zipWithIndex.filter((line, _) => line.trim == text)
      assertEquals(matches.size, 1, file + ": " + text)
      matches.head._2 + 1
    val formattedCoordinates = Map(
      (prefix + "HintRejects.scala:44:") -> (prefix + s"HintRejects.scala:${sourceLine("HintRejects.scala", "atMost = framework.realize.Elapsed.milliseconds(9223372036854775807L)")}:"),
      (prefix + "ScriptRejects.scala:219:") -> (prefix + s"ScriptRejects.scala:${sourceLine("ScriptRejects.scala", "ServerStep(")}:"),
      (prefix + "ScriptRejects.scala:226:") -> (prefix + s"ScriptRejects.scala:${sourceLine("ScriptRejects.scala", "object Replaced extends DerivesFrom(Restated, activityProtocol):") + 1}:")
    )
    val coordinates = markerCoordinates ++ formattedCoordinates
    val authored = historical.map { old =>
      val vocabulary = adaptations.getOrElse(old, old)
      coordinates.find((from, _) => vocabulary.startsWith(from)) match
        case Some((from, to)) => to + vocabulary.drop(from.length)
        case None             => vocabulary
    }
    val actual = rejections().linesIterator.toVector
    Files.writeString(
      scratch.resolve("authored-refusal-contract.txt"),
      authored.mkString("", "\n", "\n")
    ): Unit
    Files.writeString(
      scratch.resolve("authored-refusal-roots.txt"),
      rejected.mkString("", "\n", "\n")
    ): Unit
    assertEquals(actual, authored)

  test("the expected files are the ones the fixtures lift to"):
    assertEquals(leftOver(), Nil)

  concurrently("a declaration's identity does not move with its line"):
    val was = ir("declarations")
    val shifted = Files.createDirectories(scratch.resolve("shifted"))
    Files.writeString(
      shifted.resolve("Declarations.scala"),
      "// A line the declarations move down by.\n" * 3
        + Files.readString(scratch.resolve("lifts/Declarations.scala"))
    )
    Files.copy(
      scratch.resolve("lifts/project.scala"),
      shifted.resolve("project.scala")
    )
    val jar = packaged("shifted", shifted)
    val roots = fixtures.toMap.apply("declarations")
    val arguments =
      Seq(
        s"$jar=${stored("lifts")},$modelJar=model/",
        modelClasspath.toString,
        lifted("shifted").toString
      )
    val lift = this.lift((arguments ++ roots)*)
    assert(!lift.failed, lift.diagnostics)
    val now = Files.readString(lifted("shifted"))
    def lines(ir: String) = ir.replaceAll("\"line\": [0-9]+", "\"line\": _")
    assertNotEquals(
      now,
      was,
      "moving the declarations did not move their lines"
    )
    assert(
      lines(now) == lines(was),
      "moving the declarations down changed more than their lines"
    )

  concurrently("a jar's prefix may be an argument of its own"):
    val out = lifted("presence-prefix-argument")
    val lift = this.lift(
      (Seq(
        liftsJar.toString,
        modelClasspath.toString,
        out.toString,
        stored("lifts")
      )
        ++ fixtures.toMap.apply("presence"))*
    )
    assert(!lift.failed, lift.diagnostics)
    assertEquals(Files.readString(out), ir("presence"))

  concurrently("a lift without roots, or without its arguments, is refused"):
    val none = lift(
      s"$modelJar=model/",
      modelClasspath.toString,
      lifted("no-roots").toString
    )
    assertEquals(none.exit, 1)
    assertEquals(
      refused(none),
      Seq("lift: no roots: name the declarations to lift")
    )
    val usage = lift("only-one")
    assertEquals(usage.exit, 2)
    assert(
      usage.output.contains(
        "usage: lift <jar=prefix>,... <classpath file> <out.json> <root>..."
      ),
      usage.output
    )

  // ### fn-126 R20: the structure lint, over fixtures of features with their level folders

  concurrently(
    "a form binds a shared action's entity without redeclaring its identity"
  ):
    val jar = packaged("binding", materialize("binding"), Seq("--server=false"))
    val out = lifted("binding")
    val result = lift(
      s"$jar=${stored("binding")},$modelJar=model/",
      modelClasspath.toString,
      out.toString,
      "fixture.binding.WorkflowForm",
      "fixture.binding.StandaloneForm"
    )
    assert(!result.failed, result.diagnostics)
    val json = new com.fasterxml.jackson.databind.ObjectMapper()
      .readTree(Files.readString(out))
    val shared = json
      .path("actions")
      .elements()
      .asScala
      .filter(
        _.path("id").asText() == "fixture.binding.worker.take"
      )
      .toList
    assertEquals(shared.size, 1)
    assertEquals(shared.head.path("on").asText(), "job")
    assertEquals(shared.head.path("name").asText(), "take")
    assertEquals(shared.head.path("actor").asText(), "worker")
    assertEquals(
      shared.head.path("inputs").get(0).path("name").asText(),
      "ready"
    )
    assertEquals(
      shared.head.path("inputs").get(1).path("name").asText(),
      "resolution"
    )
    assertEquals(shared.head.path("results").asText(), "Delivery")
    val respond = json
      .path("actions")
      .elements()
      .asScala
      .find(
        _.path("id").asText() == "fixture.binding.worker.respond"
      )
      .get
    assertEquals(respond.path("on").asText(), "job")
    assertEquals(respond.path("examples").size(), 2)
    val create = json
      .path("actions")
      .elements()
      .asScala
      .find(
        _.path("id").asText() == "fixture.binding.caller.create"
      )
      .get
    assertEquals(create.path("creates").asText(), "job")
    for machine <- json.path("machines").elements().asScala do
      assertEquals(machine.path("entity").asText(), "job")
      assert(
        machine
          .path("steps")
          .elements()
          .asScala
          .exists(
            _.path("action").asText() == "fixture.binding.worker.take"
          )
      )
    val refusedOut = lifted("wrong-binding")
    val rejected = lift(
      s"$jar=${stored("binding")},$modelJar=model/",
      modelClasspath.toString,
      refusedOut.toString,
      "fixture.binding.WrongForm"
    )
    assertNotEquals(rejected.exit, 0)
    assert(!Files.exists(refusedOut))
    assert(
      refused(rejected).exists(_.contains("actions are on alien, job")),
      rejected.diagnostics
    )
    val conflictOut = lifted("conflicting-bindings")
    val conflict = lift(
      s"$jar=${stored("binding")},$modelJar=model/",
      modelClasspath.toString,
      conflictOut.toString,
      "fixture.binding.WorkflowForm",
      "fixture.binding.WrongForm"
    )
    assertNotEquals(conflict.exit, 0)
    assert(!Files.exists(conflictOut))
    assert(
      refused(conflict).exists(
        _.contains("conflicting entity bindings in this export")
      ),
      conflict.diagnostics
    )
    val cycleOut = lifted("cyclic-binding")
    val cycle = lift(
      s"$jar=${stored("binding")},$modelJar=model/",
      modelClasspath.toString,
      cycleOut.toString,
      "fixture.binding.CyclicForm"
    )
    assertEquals(cycle.exit, 1)
    assert(!Files.exists(cycleOut))
    assert(
      refused(cycle).exists(_.contains("cyclic action binding")),
      cycle.diagnostics
    )
    assert(!cycle.diagnostics.contains("StackOverflowError"))

  // A fixture whose features have folders, copied with them, its jar's path resolved.
  private def materializeTree(
      fixture: String,
      name: String = "",
      excluding: Set[String] = Set.empty
  ): Path =
    LayoutSources.materialize(
      testdata.resolve(fixture),
      scratch.resolve(if name.isEmpty then fixture else name),
      modelJar,
      excluding
    )

  // The template a new feature copies (layout/lamp): a feature with its two levels and a zoom-in,
  // which the lint refuses nothing of and the lifter lifts.
  concurrently("the layout template lifts with no refusal"):
    val tree = materializeTree("layout", excluding = Set("kinds", "r4"))
    val stream = Files.walk(tree)
    val inventory =
      try
        stream.iterator.asScala
          .filter(Files.isRegularFile(_))
          .map(tree.relativize(_).toString)
          .toList
          .sorted
      finally stream.close()
    assertEquals(
      inventory,
      List(
        "lamp/Lamp.scala",
        "lamp/product/Product.scala",
        "lamp/system/Bulb.scala",
        "lamp/system/Realization.scala",
        "lamp/system/System.scala",
        "project.scala"
      )
    )
    val jar = packaged("layout", tree, Seq("--server=false"))
    val explicit = scratch.resolve("layout-explicit.json")
    val direct = lift(
      s"$jar=${stored("layout")},$modelJar=model/",
      modelClasspath.toString,
      explicit.toString,
      "fixture.features.lamp.system.LampRealization$.system"
    )
    assertEquals(direct.exit, 0, direct.diagnostics)
    val out = scratch.resolve("layout-out")
    val result =
      liftIr(out, s"$jar=${stored("layout")},$modelJar=model/", "lamp")
    assertEquals(refused(result), Nil)
    assert(!result.failed, result.diagnostics)
    assertEquals(listed(out), Seq("lamp.json"))
    val lifted = Files.readString(out.resolve("lamp.json"))
    for machine <- Seq(
        "lampProduct",
        "onlyOn",
        "lampSystem",
        "onlyClosed",
        "bulb"
      )
    do
      assert(
        lifted.contains(s"\"$machine\""),
        s"lamp.json lifts no machine $machine"
      )
    val parsed =
      new com.fasterxml.jackson.databind.ObjectMapper().readTree(lifted)
    val realization = Option(parsed.get("realizations")).toSeq
      .flatMap(_.elements().asScala.toSeq)
    assertEquals(realization.size, 1)
    val explicitModel =
      new com.fasterxml.jackson.databind.ObjectMapper().readTree(Files.readString(explicit))
    assertEquals(parsed.path("realizations"), explicitModel.path("realizations"))
    assertEquals(
      realization.head.get("id").asText(),
      "fixture.features.lamp.system.LampRealization.system"
    )
    assertEquals(realization.head.get("machine").asText(), "lampSystem")
    assertEquals(
      realization.head.get("position").get("file").asText(),
      "model/irgen/testdata/layout/lamp/system/Realization.scala"
    )

  concurrently("typed generic realization alone requests a file while unrelated values do not"):
    val tree = materializeTree("coreDiscovery")
    val jar = packaged("coreDiscovery", tree, Seq("--server=false"))
    val out = scratch.resolve("coreDiscovery-out")
    val result = liftIr(out, s"$jar=${stored("coreDiscovery")},$modelJar=model/")
    assertEquals(result.exit, 0, result.diagnostics)
    assert(
      Files.isRegularFile(out.resolve("coreonly.json")),
      "the realization-only foundation was not collected"
    )
    assertEquals(
      listed(out),
      Seq(
        "activity-standalone.json",
        "activity-standalone.waivers.json",
        "coreonly.json",
        "nexus-standalone.json",
        "nexus-standalone.waivers.json",
        "nexus-workflow.json",
        "nexus-workflow.waivers.json",
        "taskqueue.json"
      )
    )
    assert(!Files.exists(out.resolve("quiet.json")), "an unrelated value requested an actor file")
    val mapper = new com.fasterxml.jackson.databind.ObjectMapper()
    val collected = mapper.readTree(Files.readString(out.resolve("coreonly.json")))
    val explicit = scratch.resolve("coreDiscovery-explicit.json")
    val direct = lift(
      s"$jar=${stored("coreDiscovery")},$modelJar=model/",
      modelClasspath.toString,
      explicit.toString,
      "fixture.foundations.coreonly.Runs$.only"
    )
    assertEquals(direct.exit, 0, direct.diagnostics)
    val expected = mapper.readTree(Files.readString(explicit))
    assertEquals(collected.path("realizations"), expected.path("realizations"))
    assertEquals(collected.path("realizations").size(), 1)
    assertEquals(
      collected.path("realizations").get(0).path("id").asText(),
      "fixture.foundations.coreonly.Runs.only"
    )
    assertEquals(collected.path("realizations").get(0).path("machine").asText(), "idle")

  concurrently(
    "kind scaffolds and their one or two forms lift with local or kind Products"
  ):
    val tree = materializeTree("layout/kinds", "kind-layout")
    val jar = packaged("kind-layout", tree, Seq("--server=false"))
    val out = scratch.resolve("kind-layout-out")
    val result = liftIr(
      out,
      s"$jar=$tree/,$modelJar=model/",
      "activity-standalone",
      "nexus-standalone",
      "nexus-workflow",
      "relay-standalone",
      "relay-workflow"
    )
    assertEquals(refused(result), Nil)
    assert(!result.failed, result.diagnostics)
    assertEquals(
      listed(out),
      Seq(
        "activity-standalone.json",
        "activity-standalone.waivers.json",
        "nexus-standalone.json",
        "nexus-standalone.waivers.json",
        "nexus-workflow.json",
        "nexus-workflow.waivers.json",
        "relay-standalone.json",
        "relay-workflow.json"
      )
    )
    for form <- Seq("standalone", "workflow") do
      val json = new com.fasterxml.jackson.databind.ObjectMapper()
        .readTree(Files.readString(out.resolve(s"relay-$form.json")))
      val machines = json.get("machines").elements().asScala.toSeq
      assertEquals(
        machines.map(_.get("name").asText()).toSet,
        Set("relayProduct", "relaySystem")
      )
      val actions = json.get("actions").elements().asScala.toSeq
      assertEquals(
        actions.map(_.get("id").asText()).toSet,
        Set(
          "fixture.features.relay.user.start",
          "fixture.features.relay.user.complete"
        )
      )
      assertEquals(
        actions
          .find(_.get("name").asText() == "start")
          .get
          .get("creates")
          .asText(),
        "task"
      )
      assertEquals(
        actions
          .find(_.get("name").asText() == "complete")
          .get
          .get("on")
          .asText(),
        "task"
      )
      assert(
        actions.forall(_.get("actor").asText() == "user")
      )
      assert(
        actions.forall(
          _.get("position").get("file").asText().endsWith("relay/Relay.scala")
        )
      )
      assert(
        machines.forall(_.get("entity").asText() == "task")
      )
      assertEquals(
        machines
          .find(
            _.get("name").asText() == "relaySystem"
          )
          .get
          .get("refines")
          .get("product")
          .asText(),
        "relayProduct"
      )

  private lazy val kindRefusals = Seq(
    (
      "missing-header",
      "nexus/Nexus.scala",
      None,
      "has no general feature file"
    ),
    (
      "wrong-header-package",
      "nexus/Nexus.scala",
      Some("package fixture.features.other\n"),
      "does not mirror"
    ),
    (
      "case-variant-header",
      "nexus/nexus.scala",
      Some("package fixture.features.nexus\n"),
      "one general feature file"
    ),
    (
      "general-vocabulary",
      "activity/Activity.scala",
      Some(
        "package fixture.features.activity\nimport framework.Finite\nenum Phase derives Finite:\n  case idle\n"
      ),
      "Phase is level vocabulary declared in activity's root feature file"
    ),
    (
      "wrong-form-package",
      "nexus/standalone/Mismatch.scala",
      Some(
        "package fixture.features.nexus.workflow\nfinal case class Marker(value: Boolean)\n"
      ),
      "does not mirror"
    ),
    (
      "deeper-form",
      "nexus/workflow/system/extra/Extra.scala",
      Some(
        "package fixture.features.nexus.workflow.system.extra\n" +
          "import framework.*\n" +
          "object Extra extends Derived(fixture.features.nexus.workflow.system.NexusSystem)\n"
      ),
      "no level folder"
    ),
    (
      "foreign-product",
      "relay/workflow/system/System.scala",
      Some(
        Files
          .readString(
            testdata.resolve("layout/kinds/relay/workflow/system/System.scala")
          )
          .replace(
            "fixture.features.relay.shared.product",
            "fixture.features.nexus.workflow.product"
          )
          .replace("RelayProduct", "NexusProduct")
      ),
      "does not refine RelayProduct"
    ),
    (
      "foreign-single-level",
      "nexus/standalone/StandaloneNexus.scala",
      Some(
        Files
          .readString(
            testdata.resolve(
              "layout/kinds/nexus/standalone/StandaloneNexus.scala"
            )
          )
          // Strict equality: the refinement compares phases with the evidence Finite supplies.
          .replace(
            "import framework.*\n",
            "import framework.*\nimport framework.Finite.given\n" +
              "val _ = () => summon[CanEqual[Phase, Phase]]\n"
          )
          .replace(
            "  object effects:",
            "  object refinement extends Refinement(fixture.features.relay.shared.product.RelayProduct):\n" +
              "    def toProduct(s: State) = fixture.features.relay.shared.product.State(\n" +
              "      if s.phase == Phase.idle then fixture.features.relay.shared.product.Phase.idle\n" +
              "      else fixture.features.relay.shared.product.Phase.done)\n\n" +
              "  object effects:"
          )
      ),
      "outside its kind"
    )
  )
  for (name, file, contents, diagnostic) <- kindRefusals do
    concurrently(s"kind classification refuses $name"):
      val variant = materializeTree("layout/kinds", s"kind-$name")
      val changed = variant.resolve(file)
      if name == "case-variant-header" then
        assume(
          !Files.exists(changed),
          "case-variant headers require a case-sensitive filesystem"
        )
      contents match
        case Some(text) =>
          Files.createDirectories(changed.getParent)
          Files.writeString(changed, text)
        case None => Files.delete(changed)
      val jar = packaged(s"kind-$name", variant, Seq("--server=false"))
      val out = scratch.resolve(s"kind-$name-out")
      val result = liftIr(out, s"$jar=$variant/,$modelJar=model/")
      assertNotEquals(result.exit, 0, name)
      assertEquals(listed(out), Nil)
      assert(
        refused(result).exists(message => message.contains(file) && message.contains(diagnostic)),
        s"$name did not report $diagnostic:\n${result.diagnostics}"
      )

  for (name, file, line, diagnostic) <- Seq(
      (
        "form-outside-kind",
        "standalone/Standalone.scala",
        3,
        "a form belongs inside a kind"
      ),
      (
        "machine-in-general-file",
        "nexus/Nexus.scala",
        5,
        "Misplaced is a machine object"
      ),
      (
        "second-general-file",
        "nexus/Extra.scala",
        1,
        "one general feature file"
      )
    )
  do
    concurrently(s"R4 refuses $name at its source line"):
      val tree = materializeTree("layout/kinds", s"r4-$name")
      val specimen = testdata.resolve(s"layout/r4/$name/$file")
      val changed = tree.resolve(file)
      Files.createDirectories(changed.getParent)
      Files.copy(
        specimen,
        changed,
        java.nio.file.StandardCopyOption.REPLACE_EXISTING
      )
      val jar = packaged(s"r4-$name", tree, Seq("--server=false"))
      val out = scratch.resolve(s"r4-$name-out")
      val result = liftIr(out, s"$jar=$tree/,$modelJar=model/")
      assert(
        refused(result).exists(message =>
          message.startsWith(s"lift: $tree/$file:$line:") && message.contains(
            diagnostic
          )
        ),
        s"$name did not report $file:$line: $diagnostic:\n${result.diagnostics}"
      )
      assertNotEquals(result.exit, 0, name)
      assertEquals(listed(out), Nil)

  concurrently("shared features refuse kind-form nesting"):
    val tree = materializeTree("layout/kinds", "shared-forms")
    val stream = Files.walk(tree.resolve("nexus"))
    try
      for file <- stream.iterator.asScala.toList if Files.isRegularFile(file) do
        Files.writeString(
          file,
          Files
            .readString(file)
            .replace("fixture.features.nexus", "fixture.foundations.nexus")
        )
    finally stream.close()
    val jar = packaged("shared-forms", tree, Seq("--server=false"))
    val out = scratch.resolve("shared-forms-out")
    val result = liftIr(out, s"$jar=$tree/,$modelJar=model/", "nexus-workflow")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    assert(
      refused(result).exists(message =>
        message.contains("nexus/workflow/") && message.contains(
          "no level folder"
        )
      ),
      result.diagnostics
    )

  concurrently(
    "a form's package mirrors its original parent kind before rebasing"
  ):
    val tree = materializeTree("layout/kinds", "wrong-parent-kind")
    Files.move(
      tree.resolve("nexus/workflow"),
      Files.createDirectories(tree.resolve("other")).resolve("workflow")
    )
    val jar = packaged("wrong-parent-kind", tree, Seq("--server=false"))
    val out = scratch.resolve("wrong-parent-kind-out")
    val result = liftIr(out, s"$jar=$tree/,$modelJar=model/", "nexus-workflow")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    assert(
      refused(result).exists(message =>
        message.contains("other/workflow/") && message.contains(
          "does not mirror"
        )
      ),
      result.diagnostics
    )

  for variantName <- Seq("unrefined-sibling", "derived-sibling") do
    concurrently(
      s"an inherited System requires its canonical primary with $variantName"
    ):
      val tree = materializeTree("layout/kinds", variantName)
      val system = tree.resolve("relay/workflow/system/System.scala")
      val text =
        if variantName == "unrefined-sibling" then
          Files
            .readString(system)
            .replaceAll("(?s)  object refinement .*?(?=  object effects:)", "")
            .replace(
              "import fixture.features.relay.shared.product.RelayProduct\n",
              ""
            )
        else
          "package fixture.features.relay.workflow\npackage system\nimport framework.*\n" +
            "import fixture.features.relay.shared.product.RelayProduct\nobject Copy extends Derived(RelayProduct.unmonitored)\n"
      Files.delete(system)
      Files.writeString(system.resolveSibling("Other.scala"), text)
      if variantName == "derived-sibling" then
        val feature = tree.resolve("relay/workflow/WorkflowRelay.scala")
        Files.writeString(
          feature,
          Files.readString(feature).replace("system.RelaySystem", "system.Copy")
        ): Unit
      val jar = packaged(variantName, tree, Seq("--server=false"))
      val out = scratch.resolve(s"$variantName-out")
      val result =
        liftIr(out, s"$jar=$tree/,$modelJar=model/", "relay-workflow")
      assertNotEquals(result.exit, 0)
      assertEquals(listed(out), Nil)
      assert(
        refused(result).exists(_.contains("system/System.scala is missing")),
        result.diagnostics
      )

  concurrently(
    "kind cores and forms refuse unknown or deeper folders containing only types"
  ):
    val tree = materializeTree("layout/kinds", "type-only-folders")
    val misplaced = Seq(
      "relay/extra/Extra.scala" -> "fixture.features.relay.extra",
      "relay/shared/product/extra/Extra.scala" -> "fixture.features.relay.shared.product.extra",
      "relay/workflow/system/extra/Extra.scala" -> "fixture.features.relay.workflow.system.extra"
    )
    for (file, pkg) <- misplaced do
      val path = tree.resolve(file)
      Files.createDirectories(path.getParent)
      Files.writeString(
        path,
        s"package $pkg\nfinal case class Extra(value: Boolean)\n"
      )
    val jar = packaged("type-only-folders", tree, Seq("--server=false"))
    val out = scratch.resolve("type-only-folders-out")
    val result = liftIr(out, s"$jar=$tree/,$modelJar=model/", "relay-workflow")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    for (file, _) <- misplaced do
      assert(
        refused(result).exists(message =>
          message.contains(file) && message.contains("no level folder")
        ),
        result.diagnostics
      )

  // R20 (a): the folders of a feature with two levels, a missing root feature file or level file,
  // a feature of one level with a subfolder, and a package that does not mirror its folder
  // (layoutRefusals/a).
  concurrently(
    "the structure lint refuses a feature's levels out of their folders"
  ):
    val jar = packaged("layoutRefusals-a", materializeTree("layoutRefusals/a"))
    val out = scratch.resolve("layoutRefusals-a-out")
    val at = stored("layoutRefusals/a")
    val result = liftIr(out, s"$jar=$at,$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    val two =
      "KettleSystem refines KettleProduct, so kettle has two levels, each in its folder: " +
        "the Product in product/Product.scala, the System in system/System.scala, beside the root " +
        "feature file named after the feature's folder, which holds shared types and the signature " +
        "(model/irgen/testdata/layout/lamp is the template)"
    val urn = two.replace(
      "KettleSystem refines KettleProduct, so kettle",
      "UrnSystem refines UrnProduct, so urn"
    )
    assertEquals(
      refused(result),
      Seq(
        s"lift: ${at}kettle/Kettle.scala:26: Stray is a machine object in kettle's root folder, " +
          s"${at}kettle/, whose feature file holds shared types and the signature " +
          "alone: a feature with two levels declares its machines in product/ and system/",
        s"lift: ${at}kettle/system/Heater.scala:12: $two; ${at}kettle/system/System.scala is " +
          "missing",
        s"lift: ${at}kettle/system/element/Element.scala:8: Element is a machine object in " +
          s"${at}kettle/system/element/, which is no level folder of kettle: a feature with two " +
          "levels keeps its Models in product/ and system/, one file per subject beside the " +
          "level's own file, with no folder below them",
        s"lift: ${at}tap/fittings/Washer.scala:5: ${at}tap/fittings/Washer.scala declares package " +
          s"fixture.features.tap, which its folder, ${at}tap/fittings/, does not mirror: a " +
          "feature's subpackages are its folders, named alike, since the structure lint reads a " +
          "source's package and the order lint its path",
        s"lift: ${at}tap/product/Product.scala:6: tap has no machine that refines another of its " +
          "own, so it has one level, whose Models sit in its feature file: it has no product/ " +
          "folder",
        s"lift: ${at}tap/valve/Valve.scala:8: tap has no machine that refines another of its " +
          "own, so it has one level, whose Models sit in its feature file: it has no valve/ folder",
        s"lift: ${at}urn/system/System.scala:11: $urn; ${at}urn/ has no root feature file",
        s"lift: ${at}urn/system/System.scala:11: $urn; ${at}urn/product/Product.scala is missing"
      )
    )

  // R20 (b): the level files name the feature's Product and System machines and own their
  // Phase, State and Fact vocabulary rather than leaving prefixed level types in the root.
  concurrently(
    "the structure lint refuses invalid level names and root-owned level vocabulary"
  ):
    val jar = packaged("layoutRefusals-b", materializeTree("layoutRefusals/b"))
    val out = scratch.resolve("layoutRefusals-b-out")
    val at = stored("layoutRefusals/b")
    val result = liftIr(out, s"$jar=$at,$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    assertEquals(
      refused(result),
      Seq(
        s"lift: ${at}boiler/product/Product.scala:6: BoilerContract is the Product machine in " +
          "boiler's product/Product.scala: name it BoilerProduct, after the feature and its " +
          "Product level",
        s"lift: ${at}boiler/system/System.scala:7: BoilerImplementation is the System machine in " +
          "boiler's system/System.scala: name it BoilerSystem, after the feature and its System " +
          "level",
        s"lift: ${at}kettle/product/Product.scala:6: KettleContract is the Product machine in " +
          "kettle's product/Product.scala: name it KettleProduct, after the feature and its " +
          "Product level",
        s"lift: ${at}pump/system/System.scala:7: PumpProtocol is the System machine in pump's " +
          "system/System.scala: name it PumpSystem, with the same prefix as PumpProduct",
        s"lift: ${at}reservoir/Reservoir.scala:6: ProductPhase is Product level vocabulary " +
          "declared in reservoir's root feature file: declare it as Phase in " +
          "product/Product.scala",
        s"lift: ${at}reservoir/Reservoir.scala:9: ProductState is Product level vocabulary " +
          "declared in reservoir's root feature file: declare it as State in " +
          "product/Product.scala",
        s"lift: ${at}reservoir/Reservoir.scala:11: ProductFact is Product level vocabulary " +
          "declared in reservoir's root feature file: declare it as Fact in " +
          "product/Product.scala",
        s"lift: ${at}reservoir/Reservoir.scala:14: Phase is level vocabulary declared in " +
          "reservoir's root feature file: declare it in product/Product.scala or " +
          "system/System.scala",
        s"lift: ${at}reservoir/Reservoir.scala:17: SystemState is System level vocabulary " +
          "declared in reservoir's root feature file: declare it as State in " +
          "system/System.scala",
        s"lift: ${at}reservoir/Reservoir.scala:19: SystemFact is System level vocabulary " +
          "declared in reservoir's root feature file: declare it as Fact in " +
          "system/System.scala",
        s"lift: ${at}reservoir/product/Product.scala:6: ReservoirProduct is reservoir's Product " +
          "machine, but product/Product.scala declares no Fact: each level owns Phase, State and " +
          "Fact in its level file",
        s"lift: ${at}reservoir/product/Product.scala:6: ReservoirProduct is reservoir's Product " +
          "machine, but product/Product.scala declares no Phase: each level owns Phase, State " +
          "and Fact in its level file",
        s"lift: ${at}reservoir/product/Product.scala:6: ReservoirProduct is reservoir's Product " +
          "machine, but product/Product.scala declares no State: each level owns Phase, State " +
          "and Fact in its level file",
        s"lift: ${at}reservoir/system/System.scala:7: ReservoirSystem is reservoir's System " +
          "machine, but system/System.scala declares no Fact: each level owns Phase, State and " +
          "Fact in its level file",
        s"lift: ${at}reservoir/system/System.scala:7: ReservoirSystem is reservoir's System " +
          "machine, but system/System.scala declares no Phase: each level owns Phase, State and " +
          "Fact in its level file",
        s"lift: ${at}reservoir/system/System.scala:7: ReservoirSystem is reservoir's System " +
          "machine, but system/System.scala declares no State: each level owns Phase, State and " +
          "Fact in its level file",
        s"lift: ${at}urn/system/System.scala:6: UrnSystem is the System machine in urn's " +
          "system/System.scala but does not refine UrnProduct from product/Product.scala"
      )
    )

  concurrently(
    "the structure lint requires independent unambiguous primary level machines"
  ):
    val jar = packaged("layoutRefusals-d", materializeTree("layoutRefusals/d"))
    val out = scratch.resolve("layoutRefusals-d-out")
    val at = stored("layoutRefusals/d")
    val result = liftIr(out, s"$jar=$at,$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    assertEquals(
      refused(result),
      Seq(
        s"lift: ${at}brazier/product/Product.scala:6: brazier's product/Product.scala declares " +
          "multiple primary Product machines: BrazierProduct, SpareProduct",
        s"lift: ${at}cistern/product/Product.scala:6: CisternContract is the Product machine in " +
          "cistern's product/Product.scala: name it CisternProduct, after the feature and its " +
          "Product level",
        s"lift: ${at}cistern/system/System.scala:6: cistern's system/System.scala declares no " +
          "primary System machine: declare one <Prefix>System",
        s"lift: ${at}furnace/system/System.scala:6: furnace's system/System.scala declares " +
          "multiple primary System machines: FurnaceSystem, SpareSystem",
        s"lift: ${at}pan/product/Product.scala:6: pan's product/Product.scala declares no " +
          "primary Product machine: declare one <Prefix>Product",
        s"lift: ${at}pan/system/System.scala:6: PanImplementation is the System machine in pan's " +
          "system/System.scala: name it PanSystem, after the feature and its System level",
        s"lift: ${at}stove/system/System.scala:6: StoveSystem is the System machine in stove's " +
          "system/System.scala but does not refine StoveProduct from product/Product.scala"
      )
    )

  // R20 (c): a machine's sections by their closed names, which say what each holds, and
  // one object exports per feature, in its root feature file (layoutRefusals/c); a level folder's
  // file is held to a feature file's order, which has no exports there.
  concurrently("the structure lint refuses an unnamed section"):
    val jar = packaged("layoutRefusals-c", materializeTree("layoutRefusals/c"))
    val out = scratch.resolve("layoutRefusals-c-out")
    val at = stored("layoutRefusals/c")
    val result = liftIr(out, s"$jar=$at,$modelJar=model/")
    assertNotEquals(result.exit, 0)
    assertEquals(listed(out), Nil)
    val product = s"${at}kiln/product/Product.scala"
    val none =
      "and none of its sections, states, refinement, effects, monitors, rules, syncs, " +
        "properties, capabilities, queries: a section of another name sits at the top " +
        "level of the " +
        "file, in the signature, and anything else in one of these"
    assertEquals(
      refused(result),
      Seq(
        s"lift: $product:17: timers is an object in KilnProduct $none",
        s"lift: $product:21: helpers is an object in KilnProduct $none",
        s"lift: ${at}pump/system/System.scala:21: Gauge belongs before PumpSystem at " +
          s"${at}pump/system/System.scala:7: a level folder's file reads its header, then its " +
          "types, then its signature, then its machine and composition objects"
      )
    )
