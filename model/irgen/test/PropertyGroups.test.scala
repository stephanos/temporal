package umpire.irgen

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*
import scala.concurrent.duration.*
import com.fasterxml.jackson.databind.{JsonNode, ObjectMapper}
import com.fasterxml.jackson.databind.node.ObjectNode
import umpire.check.Tools

class PropertyGroupsFixtures extends munit.FunSuite:
  override val munitTimeout = 10.minutes
  private val tools = Tools.here
  private val build = tools.directory.resolve("model/build")
  private val stored = "model/irgen/testdata/propertyGroups/"
  private val scratch =
    Files.createTempDirectory(build.resolve("history"), "property-groups.")
  private val mapper = ObjectMapper()
  private lazy val commonJar = packaged("common")

  private def packaged(variant: String): Path =
    val sources =
      Files.createDirectories(scratch.resolve(variant).resolve("sources"))
    for part <- Seq(variant) do
      val from = tools.directory.resolve(stored + part)
      val stream = Files.walk(from)
      try
        for file <- stream.iterator.asScala if file.toString.endsWith(".scala")
        do
          val target = sources.resolve(from.relativize(file))
          Files.createDirectories(target.getParent): Unit
          Files.writeString(
            target,
            Files
              .readString(file)
              .replace(
                "../../../../build/model-scala.jar",
                build.resolve("model-scala.jar").toString
              )
          ): Unit
      finally stream.close()
    val jar = scratch.resolve(variant).resolve("fixture.jar")
    val extra = if variant.equals("common") then Seq.empty
    else
      Seq(
        "--jar",
        commonJar.toString,
        "--jar",
        build.resolve("model-scala.jar").toString,
        "--scala",
        "3.9.0"
      )
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
        ) ++ extra
      )
      .orFail(): Unit
    jar

  private lazy val flatJar = packaged("flat")
  private lazy val groupedJar = packaged("grouped")
  private lazy val negativeJar = packaged("negative")
  private def lifted(variant: String): Map[String, JsonNode] =
    val jar = if variant.equals("flat") then flatJar else groupedJar
    val out = scratch.resolve(variant).resolve("ir")
    val result = InProcess.lift(
      tools.directory,
      Seq(
        "--ir",
        s"$jar=$stored$variant/,$commonJar=${stored}common/,${build.resolve("model-scala.jar")}=model/",
        build.resolve("model-scala.classpath").toString,
        out.toString,
        "specimen-workflow",
        "lonely",
        "bundleonly"
      )
    )
    assertEquals(result.exit, 0, result.output)
    Seq("specimen-workflow", "lonely", "bundleonly")
      .map(name => name -> mapper.readTree(Files.readString(out.resolve(s"$name.json"))))
      .toMap

  private val expectedProperties = List(
    "joined.memberClosed",
    "subject.aliased",
    "subject.bundledClosed",
    "subject.bundledOpen",
    "subject.closed",
    "subject.stable"
  )
  private val expectedProgress =
    List("subject.eventuallyClosed", "subject.factoryProgress")
  private lazy val models =
    Seq("flat", "grouped").map(v => v -> lifted(v)).toMap
  private def inventory(model: JsonNode, kind: String): List[String] =
    model
      .path(kind)
      .elements()
      .asScala
      .map(p => s"${p.path("machine").asText()}.${p.path("name").asText()}")
      .toList
      .sorted

  test(
    "flat and grouped automatic discovery each match the independent whole inventory"
  ) {
    for variant <- Seq("flat", "grouped") do
      val discovered = models(variant)
      assertEquals(
        inventory(discovered("specimen-workflow"), "properties"),
        expectedProperties
      )
      assertEquals(
        inventory(discovered("specimen-workflow"), "progress"),
        expectedProgress
      )
      assertEquals(
        inventory(discovered("lonely"), "progress"),
        List("lonely.onlyProgress")
      )
      assertEquals(inventory(discovered("lonely"), "properties"), Nil)
  }

  test(
    "a non-form owner with only bundled unqueried Progress declares its file"
  ) {
    for variant <- Seq("flat", "grouped") do
      assertEquals(
        inventory(models(variant)("bundleonly"), "progress"),
        List("bundleOnly.bundledOnly")
      )
  }

  test(
    "arbitrary nested helper Progress remains refused rather than automatically exported"
  ) {
    val out = scratch.resolve("negative/ir")
    val result = InProcess.lift(
      tools.directory,
      Seq(
        "--ir",
        s"$negativeJar=${stored}negative/,$commonJar=${stored}common/,${build.resolve("model-scala.jar")}=model/",
        build.resolve("model-scala.classpath").toString,
        out.toString,
        "hidden"
      )
    )
    Files.writeString(scratch.resolve("negative/lift.log"), result.output): Unit
    Files.writeString(
      scratch.resolve("negative/lift.rc"),
      result.exit.toString
    ): Unit
    assertEquals(result.exit, 1, result.output)
    assert(result.output.contains("Hidden.properties.helper"), result.output)
    assert(result.output.contains("ignored is a Property"), result.output)
    assert(
      result.output.contains(
        "unexpected is a monitor, assumption, hole or channel"
      ),
      result.output
    )
    assert(
      result.output.contains(
        "notAttached is a monitor, assumption, hole or channel"
      ),
      result.output
    )
    assert(!Files.exists(out.resolve("hidden.json")))
  }

  private lazy val subject = "propertygroups.features.specimen.workflow.Subject"
  private lazy val joined = "propertygroups.features.specimen.workflow.Joined"
  private lazy val lonely = "propertygroups.foundations.lonely.Lonely"
  private lazy val bundleOnly =
    "propertygroups.foundations.bundleonly.BundleOnly"
  private lazy val flatRoots = Map(
    "specimen-workflow" -> List(
      joined,
      s"$joined$$.properties",
      subject,
      s"$subject$$.properties",
      s"$subject$$.properties$$.factory",
      s"$subject$$.properties$$.reaches",
      s"$subject$$.queries"
    ),
    "lonely" -> List(
      lonely,
      s"$lonely$$.properties",
      s"$lonely$$.properties$$.progress"
    ),
    "bundleonly" -> List(bundleOnly, s"$bundleOnly$$.properties")
  )
  private lazy val groupedRoots = Map(
    "specimen-workflow" -> List(
      joined,
      s"$joined$$.properties",
      s"$joined$$.properties$$.safety",
      subject,
      s"$subject$$.properties",
      s"$subject$$.properties$$.liveness",
      s"$subject$$.properties$$.liveness$$.factory",
      s"$subject$$.properties$$.liveness$$.reaches",
      s"$subject$$.properties$$.safety",
      s"$subject$$.queries"
    ),
    "lonely" -> List(
      lonely,
      s"$lonely$$.properties",
      s"$lonely$$.properties$$.liveness",
      s"$lonely$$.properties$$.liveness$$.progress"
    ),
    "bundleonly" -> List(
      bundleOnly,
      s"$bundleOnly$$.properties",
      s"$bundleOnly$$.properties$$.liveness"
    )
  )
  private lazy val workflowLines = (1 to 18).map(n => n -> n).toMap ++ Map(
    19 -> 21,
    20 -> 22,
    21 -> 23,
    22 -> 24,
    23 -> 25,
    24 -> 26,
    25 -> 27,
    26 -> 29,
    27 -> 30,
    28 -> 31,
    29 -> 32,
    30 -> 33,
    31 -> 34,
    32 -> 35,
    33 -> 36,
    34 -> 37,
    35 -> 38,
    36 -> 40
  )
  private lazy val foundationLines =
    (1 to 13).map(n => n -> n).toMap ++ Map(14 -> 15, 15 -> 16)

  private def mapped(flat: JsonNode, name: String): JsonNode =
    val copy = flat.deepCopy[JsonNode]()
    def walk(n: JsonNode): Unit =
      if n.has("position") then
        val p =
          n.path("position")
            .asInstanceOf[ObjectNode] // scalafix:ok DisableSyntax.asInstanceOf
        val file = p.path("file").asText()
        val line = p.path("line").asInt()
        val authored = tools.directory.resolve(file)
        assert(
          Files.isRegularFile(authored),
          s"source span has no authored file: $file"
        )
        assert(
          line > 0 && line <= Files.readAllLines(authored).size(),
          s"invalid authored span $file:$line"
        )
        if file.startsWith(stored + "flat/") then
          val suffix = file.stripPrefix(stored + "flat/")
          val lines = suffix match
            case "features/specimen/workflow/WorkflowSpecimen.scala" =>
              workflowLines
            case "foundations/lonely/Lonely.scala" | "foundations/bundleonly/Bundleonly.scala" =>
              foundationLines
            case other => fail(s"unmapped source $other")
          assert(lines.contains(line), s"unmapped source span $file:$line")
          p.put("file", stored + "grouped/" + suffix): Unit
          p.put("line", lines(line)): Unit
        else
          assertEquals(file, stored + "common/Shared.scala")
          assert(line >= 1 && line <= 37, s"unmapped shared span $file:$line")
      n.elements().asScala.foreach(walk)
    walk(copy)
    copy
      .asInstanceOf[ObjectNode]
      .put(
        "source", // scalafix:ok DisableSyntax.asInstanceOf
        "model: " + groupedRoots(name).sorted.mkString(", ")
      ): Unit
    copy

  test(
    "every identity, predicate, attachment and metadata field differs only by explicit source mapping"
  ) {
    for name <- Seq("specimen-workflow", "lonely", "bundleonly") do
      assertEquals(
        models("flat")(name).path("source").asText(),
        "model: " + flatRoots(name).sorted.mkString(", ")
      )
      assertEquals(
        models("grouped")(name).path("source").asText(),
        "model: " + groupedRoots(name).sorted.mkString(", ")
      )
      assertEquals(mapped(models("flat")(name), name), models("grouped")(name))
  }

  test(
    "dropping a whole group from both candidate inventories fails the frozen declaration baseline"
  ) {
    val left = models("flat")("specimen-workflow").deepCopy[JsonNode]()
    val right = models("grouped")("specimen-workflow").deepCopy[JsonNode]()
    for dropped <- Seq(left, right) do
      dropped
        .asInstanceOf[ObjectNode]
        .remove("properties"): Unit // scalafix:ok DisableSyntax.asInstanceOf
      assert(!inventory(dropped, "properties").equals(expectedProperties))
    assertEquals(mapped(left, "specimen-workflow"), right)
  }

  test(
    "native construction checks every applicable finite predicate row and retains Query selection"
  ) {
    val dependencies = Files
      .readString(build.resolve("model-scala.classpath"))
      .trim
      .split(java.io.File.pathSeparator)
      .filter(_.endsWith(".jar"))
      .toSeq
    for (variant, jar) <- Seq("flat" -> flatJar, "grouped" -> groupedJar) do
      val classpath = (Seq(
        jar.toString,
        commonJar.toString,
        build.resolve("model-scala.jar").toString
      ) ++ dependencies).distinct.mkString(java.io.File.pathSeparator)
      val result = tools.run(
        "java",
        Seq("-cp", classpath, "framework.PropertyGroupsNative")
      )
      Files.writeString(
        scratch.resolve(variant).resolve("native.log"),
        result.output
      ): Unit
      assertEquals(result.exit, 0, result.output)
      assertEquals(
        result.output.linesIterator
          .filter(_.startsWith("PROPERTY_GROUPS_NATIVE_OK"))
          .toList,
        List(
          "PROPERTY_GROUPS_NATIVE_OK predicateRows=30 monitorRows=8 properties=6 progress=4"
        )
      )
  }

  private def expression(n: JsonNode): String =
    if n.has("var") then n.path("var").asText()
    else if n.has("field") then
      val f = n.path("field")
      s"${expression(f.path("base"))}.${f.path("field").asText()}"
    else if n.has("literal") then
      val l = n.path("literal")
      if l.has("enum") then
        s"${l.path("enum").path("type").asText()}.${l.path("enum").path("case").asText()}"
      else if l.has("bool") then l.path("bool").asText()
      else fail(s"unexpected predicate literal $l")
    else if n.has("call") then
      val c = n.path("call")
      s"CALL(${c.path("function").asText()},${c.path("args").elements().asScala.map(expression).mkString(",")})"
    else if n.has("unary") then
      val u = n.path("unary")
      s"${u.path("op").asText()}(${expression(u.path("operand"))})"
    else if n.has("binary") then
      val b = n.path("binary")
      s"${b.path("op").asText()}(${expression(b.path("left"))},${expression(b.path("right"))})"
    else fail(s"unexpected predicate expression $n")

  private def evaluate(
      n: JsonNode,
      row: Map[String, String],
      helpers: Map[String, JsonNode] = Map.empty
  ): String =
    if n.has("call") then
      val c = n.path("call")
      assertEquals(
        c.path("args").elements().asScala.map(expression).toList,
        List("after")
      )
      val helper =
        helpers.getOrElse(
          c.path("function").asText(),
          fail(s"unpinned predicate call $c")
        )
      evaluate(helper, row)
    else if n.has("unary") then
      val u = n.path("unary")
      assertEquals(u.path("op").asText(), "OP_NOT")
      (!evaluate(u.path("operand"), row, helpers).equals("true")).toString
    else if n.has("binary") then
      val b = n.path("binary")
      val left = evaluate(b.path("left"), row, helpers)
      val right = evaluate(b.path("right"), row, helpers)
      (b.path("op").asText() match
        case "OP_EQ"  => left.equals(right)
        case "OP_NE"  => !left.equals(right)
        case "OP_AND" => left.equals("true") && right.equals("true")
        case other    => fail(s"unexpected predicate operation $other")
      ).toString
    else if n.has("literal") then expression(n)
    else
      row.getOrElse(
        expression(n),
        fail(s"missing predicate input ${expression(n)}")
      )

  test(
    "lifted predicates and metadata match independently authored finite truth tables"
  ) {
    val states =
      Vector("propertygroups.State.open", "propertygroups.State.closed")
    val afterRows =
      states.map(s =>
        Map(
          "after.state" -> s,
          "after.outcome" -> "propertygroups.Outcome.done"
        )
      )
    val closed = "OP_EQ(after.state,propertygroups.State.closed)"
    val predicates = List(
      ("subject.property.closed", closed, afterRows, Vector(false, true)),
      (
        "subject.property.aliased",
        "OP_NE(after.state,propertygroups.State.open)",
        afterRows,
        Vector(false, true)
      ),
      (
        "subject.property.bundledOpen",
        "OP_EQ(after.state,propertygroups.State.open)",
        afterRows,
        Vector(true, false)
      ),
      (
        "subject.property.bundledClosed",
        s"OP_AND($closed,OP_EQ(after.outcome,propertygroups.Outcome.done))",
        afterRows,
        Vector(false, true)
      ),
      (
        "subject.property.stable",
        "OP_EQ(before,after.state)",
        states.flatMap(before => afterRows.map(_ + ("before" -> before))),
        Vector(true, false, false, true)
      ),
      (
        "joined.property.memberClosed",
        "OP_NOT(CALL(joined.property.memberClosed.never,after))",
        states.map(s => Map("after.state.value" -> s)),
        Vector(false, true)
      )
    )
    for variant <- Seq("flat", "grouped") do
      val model = models(variant)("specimen-workflow")
      val propertyPositions = Map(
        "memberClosed" -> (
          s"$variant/features/specimen/workflow/WorkflowSpecimen.scala",
          if variant.equals("flat") then 36 else 40
        ),
        "closed" -> (
          s"$variant/features/specimen/workflow/WorkflowSpecimen.scala",
          if variant.equals("flat") then 19 else 21
        ),
        "stable" -> (
          s"$variant/features/specimen/workflow/WorkflowSpecimen.scala",
          if variant.equals("flat") then 20 else 22
        ),
        "aliased" -> ("common/Shared.scala", 21),
        "bundledOpen" -> ("common/Shared.scala", 25),
        "bundledClosed" -> ("common/Shared.scala", 26)
      )
      for p <- model.path("properties").elements().asScala do
        val name = p.path("name").asText()
        val (file, line) = propertyPositions(name)
        assertEquals(p.path("position").path("file").asText(), stored + file)
        assertEquals(p.path("position").path("line").asInt(), line)
        assert(
          Files
            .readAllLines(tools.directory.resolve(stored + file))
            .get(line - 1)
            .contains(s"property(\"$name\")")
        )
        assertEquals(
          p.path("holds").asText(),
          s"${p.path("machine").asText()}.property.$name"
        )
        assertEquals(p.path("transition").asBoolean(), name.equals("stable"))
        assertEquals(p.has("whenClass"), name.equals("closed"))
        if name.equals("closed") then
          assertEquals(
            p.path("whenClass").path("action").asText(),
            "propertygroups.tester.advance"
          )
      def body(ir: JsonNode, id: String): JsonNode =
        val found =
          ir.path("functions")
            .elements()
            .asScala
            .filter(_.path("name").asText().equals(id))
            .toList
        assertEquals(found.size, 1, id)
        val params =
          if id.contains(".progress.") then
            """[{"name":"state","type":{"named":"propertygroups.State"}}]"""
          else
            id match
              case "subject.property.stable" =>
                """[{"name":"before","type":{"named":"propertygroups.State"}},{"name":"after","type":{"named":"framework.Step"}}]"""
              case "propertygroups.seal.next" =>
                """[{"name":"boolean","type":{"bool":{}}},{"name":"state","type":{"named":"propertygroups.State"}},{"name":"after","type":{"named":"framework.Step"}}]"""
              case "propertygroups.seal.violated" =>
                """[{"name":"s","type":{"bool":{}}}]"""
              case _ =>
                """[{"name":"after","type":{"named":"framework.Step"}}]"""
        assertEquals(found.head.path("params"), mapper.readTree(params), id)
        found.head.path("body")
      val never = body(model, "joined.property.memberClosed.never")
      assertEquals(
        expression(never),
        "OP_NE(after.state.value,propertygroups.State.closed)"
      )
      val helpers = Map("joined.property.memberClosed.never" -> never)
      for (id, signature, rows, truth) <- predicates do
        val b = body(model, id)
        assertEquals(expression(b), signature)
        assertEquals(
          rows.map(r => evaluate(b, r, helpers).equals("true")),
          truth
        )
      val progress = List(
        ("specimen-workflow", "subject", "eventuallyClosed", 1),
        ("specimen-workflow", "subject", "factoryProgress", 2),
        ("lonely", "lonely", "onlyProgress", 3),
        ("bundleonly", "bundleOnly", "bundledOnly", 4)
      )
      for (file, owner, name, bound) <- progress do
        val ir = models(variant)(file)
        val p =
          ir.path("progress")
            .elements()
            .asScala
            .find(_.path("name").asText().equals(name))
            .get
        assertEquals(p.path("machine").asText(), owner)
        assertEquals(p.path("within").asInt(), bound)
        assertEquals(
          p.path("assumptions").elements().asScala.map(_.asText()).toList,
          List("propertygroups.fair")
        )
        for (part, target, truth) <- List(
            ("from", "open", Vector(true, false)),
            ("to", "closed", Vector(false, true))
          )
        do
          val id = s"$owner.progress.$name.$part"
          assertEquals(p.path(part).asText(), id)
          val b = body(ir, id)
          assertEquals(
            expression(b),
            s"OP_EQ(state,propertygroups.State.$target)"
          )
          assertEquals(
            states.map(s => evaluate(b, Map("state" -> s)).equals("true")),
            truth
          )
      val q = model.path("queries")
      assertEquals(q.size(), 1)
      assertEquals(q.get(0).path("name").asText(), "finished")
      assertEquals(q.get(0).path("form").asText(), "FORM_VERIFY")
      assertEquals(
        q.get(0).path("property"),
        mapper.readTree("""{"machine":"subject","name":"closed"}""")
      )
      assertEquals(
        q.get(0).path("scenario"),
        mapper.readTree("""{"machine":"subject","name":"any"}""")
      )
      assertEquals(
        q.get(0).path("limits"),
        mapper.readTree("""{"name":"small","steps":1,"actions":1,"search":8}""")
      )
      assertEquals(q.get(0).path("total").asText(), "2")
      val scenarios = model.path("scenarios")
      assertEquals(scenarios.size(), 1)
      assertEquals(scenarios.get(0).path("machine").asText(), "subject")
      assertEquals(scenarios.get(0).path("name").asText(), "any")
      assert(scenarios.get(0).path("free").asBoolean())
      assertEquals(scenarios.get(0).path("actions").size(), 0)
      assertEquals(
        scenarios.get(0).path("start").path("literal"),
        mapper.readTree(
          """{"enum":{"type":"propertygroups.State","case":"open"}}"""
        )
      )
      val monitors = model.path("monitors")
      assertEquals(monitors.size(), 1)
      assertEquals(monitors.get(0).path("id").asText(), "propertygroups.seal")
      assertEquals(
        monitors
          .get(0)
          .path("initial")
          .path("literal")
          .path("bool")
          .asBoolean(),
        false
      )
      assert(monitors.get(0).has("everyStep"))
      assertEquals(
        model.path("machines").get(0).path("monitors"),
        mapper.readTree("""["propertygroups.seal"]""")
      )
      assertEquals(
        model.path("compositions").get(0).path("members"),
        mapper.readTree("""[{"field":"value","machine":"subject"}]""")
      )
      assertEquals(
        model.path("assumptions"),
        mapper.readTree(
          s"""[{"id":"propertygroups.fair","name":"fair","position":{"file":"${stored}common/Shared.scala","line":12},"fair":["propertygroups.tester.advance"]}]"""
        )
      )
      val next = body(model, "propertygroups.seal.next")
      assertEquals(expression(next), closed)
      val monitorRows = Vector("false", "true").flatMap(m =>
        states.flatMap(before => afterRows.map(_ ++ Map("monitor" -> m, "before" -> before)))
      )
      assertEquals(
        monitorRows.map(r => evaluate(next, r).equals("true")),
        Vector(false, true, false, true, false, true, false, true)
      )
      val violated = body(model, "propertygroups.seal.violated")
      assertEquals(expression(violated), "s")
      assertEquals(
        Vector("false", "true").map(s => evaluate(violated, Map("s" -> s))),
        Vector("false", "true")
      )
  }
