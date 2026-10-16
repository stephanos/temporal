package framework.lint

import framework.`export`.*

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*
import scala.concurrent.duration.*
import com.fasterxml.jackson.databind.{JsonNode, ObjectMapper}
import com.fasterxml.jackson.databind.node.ObjectNode
import framework.check.Tools
import FixtureBuild.buildFixture
import framework.check.RetainedScratch

class PropertyGroupsFixtures extends munit.FunSuite with RetainedScratch:
  override val munitTimeout = 10.minutes
  private val tools = Tools.here
  private val build = tools.directory.resolve("model/build")
  private val stored = "model/framework/export/testdata/propertyGroups/"
  private val scratch =
    retained(build.resolve("history"), "property-groups.")
  private val mapper = ObjectMapper()
  private lazy val frameworkJar =
    val out = Files.createDirectories(scratch.resolve("framework")).resolve("framework.jar")
    val _ = tools
      .buildFixture(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          tools.directory.resolve("model/framework/dsl").toString,
          tools.directory.resolve("model/framework/realize").toString,
          "-f",
          "-o",
          out.toString
        )
      )
      .orFail()
    out
  private lazy val frameworkClasspath =
    val entries = Files
      .readString(ModelBuild.classpath)
      .trim
      .split(java.io.File.pathSeparator)
      .filter(_.endsWith(".jar"))
    val out = scratch.resolve("framework.classpath")
    Files.writeString(out, entries.mkString(java.io.File.pathSeparator)): Unit
    out
  private lazy val commonJar = packaged("common")
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

  private def packaged(variant: String, accepted: Boolean = true): Path =
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
                "../../../../../build/model-scala.jar",
                frameworkJar.toString
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
        frameworkJar.toString,
        "--scala",
        "3.9.0"
      )
    val result = tools
      .buildFixture(
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
    Files.writeString(scratch.resolve(variant).resolve("compile.log"), result.output): Unit
    Files.writeString(scratch.resolve(variant).resolve("compile.rc"), result.exit.toString): Unit
    if accepted then result.orFail(): Unit
    else assertEquals(result.exit, 1, result.output)
    jar

  private def placementLift(variant: String, file: String): framework.check.Ran =
    Files.writeString(scratch.resolve("runtime-classpath"), sys.props("java.class.path")): Unit
    Files.writeString(
      scratch.resolve("tested-class-root"),
      classOf[PropertyGroupsFixtures].getProtectionDomain.getCodeSource.getLocation.toString
    ): Unit
    val jar = packaged(variant)
    val out = scratch.resolve(variant).resolve("ir")
    val result = InProcess.lift(
      tools.directory,
      Seq(
        "--ir",
        s"$jar=$stored$variant/,$commonJar=${stored}common/,$frameworkJar=model/framework/",
        frameworkClasspath.toString,
        out.toString,
        file
      )
    )
    Files.writeString(scratch.resolve(variant).resolve("lift.log"), result.output): Unit
    Files.writeString(scratch.resolve(variant).resolve("lift.rc"), result.exit.toString): Unit
    result

  test("placement admits valid empty omitted and one-sided groups with an independent inventory") {
    val result = placementLift("placement", "placement")
    assertEquals(result.exit, 0, result.output)
    val model = mapper.readTree(Files.readString(scratch.resolve("placement/ir/placement.json")))
    assertEquals(
      inventory(model, "properties"),
      List(
        "owner.aliased",
        "owner.bundleNever",
        "owner.bundleStays",
        "owner.direct",
        "owner.never",
        "owner.stays",
        "owner.unless",
        "safetyOnly.only"
      )
    )
    assertEquals(
      inventory(model, "progress"),
      List("livenessOnly.only", "owner.directProgress", "owner.factoryProgress")
    )
    assertEquals(model.path("monitors").size(), 1)
    val owner =
      model.path("machines").elements().asScala.find(_.path("name").asText().equals("owner")).get
    assertEquals(owner.path("monitors"), mapper.readTree("""["propertygroups.seal"]"""))
    assertEquals(model.path("machines").size(), 5)
    for p <- model.path("progress").elements().asScala do
      val name = p.path("name").asText()
      assertEquals(
        p.path("within").asInt(),
        Map("directProgress" -> 1, "factoryProgress" -> 2, "only" -> 3)(name)
      )
      assertEquals(p.path("assumptions"), mapper.readTree("""["propertygroups.fair"]"""))
  }

  test(
    "placement refuses every crossed direct alias factory bundle field subtype and monitor binding"
  ) {
    val result = placementLift("crossed", "crossed")
    assertEquals(result.exit, 1, result.output)
    val path = stored + "crossed/foundations/crossed/Crossed.scala"
    val expected = List(
      (14, "directProgress", "a progress claim", "liveness"),
      (16, "aliasProgress", "a progress claim", "liveness"),
      (17, "factoryProgress", "a progress claim", "liveness"),
      (18, "mixedBundle.progress", "a progress claim", "liveness"),
      (19, "reversedBundle.progress", "a progress claim", "liveness"),
      (20, "bundleAlias.progress", "a progress claim", "liveness"),
      (21, "selectedProgress", "a progress claim", "liveness"),
      (22, "scenarioBundle.scenario", "a Scenario", "queries"),
      (24, "directProperty", "a Property", "safety"),
      (25, "aliasProperty", "a Property", "safety"),
      (26, "factoryProperty", "a Property", "safety"),
      (27, "neverProperty", "a Property", "safety"),
      (28, "neverAlias", "a Property", "safety"),
      (29, "staysProperty", "a Property", "safety"),
      (30, "staysAlias", "a Property", "safety"),
      (31, "unlessProperty", "a Property", "safety"),
      (34, "subtypeBundle.never", "a Property", "safety"),
      (34, "subtypeBundle.stays", "a Property", "safety"),
      (35, "subtypeAlias.never", "a Property", "safety"),
      (35, "subtypeAlias.stays", "a Property", "safety"),
      (36, "selectedNever", "a Property", "safety"),
      (37, "mixedBundle.safety", "a Property", "safety"),
      (38, "reversedBundle.safety", "a Property", "safety"),
      (39, "watched", "a safety monitor reference", "safety"),
      (40, "monitorAlias", "a safety monitor reference", "safety"),
      (41, "monitorFactory", "a safety monitor reference", "safety"),
      (42, "queryBundle.query", "a Query", "queries"),
      (43, "misplacedEffect", "a step function", "effects"),
      (44, "misplacedStep", "a step function", "effects")
    )
    val actual = result.output.linesIterator.filter(_.startsWith("lift: ")).toList
    val diagnostics = expected.map { (line, binding, kind, group) =>
      val written = if line <= 22 then "safety" else "liveness"
      s"lift: $path:$line: $binding is $kind, declared in Crossed.properties.$written: it belongs in the `$group` object of its machine's `properties`" match
        case text if Set("queries", "effects")(group) =>
          text.replace("of its machine's `properties`", "of its machine's object")
        case text => text
    }
    assertEquals(actual.sorted, diagnostics.sorted, result.output)
    assert(!Files.exists(scratch.resolve("crossed/ir/crossed.json")))
  }

  test("placement retains source-attributed zero and negative progress bound refusal") {
    for (variant, file, owner, bound) <- List(
        ("zero", "zero", "Zero", 0),
        ("negativeBound", "negativebound", "Negativebound", -1)
      )
    do
      val result = placementLift(variant, file)
      assertEquals(result.exit, 1, result.output)
      assertEquals(
        result.output.linesIterator.filter(_.startsWith("lift: ")).toList,
        s"lift: the roots of $file.json did not lift:" :: List.fill(3)(
          s"lift: ${stored}$variant/foundations/$file/${
              if owner.equals("Zero") then "Zero" else "Negativebound"
            }.scala:12: progress invalid bounds itself within $bound steps; a bound is at least one step"
        )
      )
      assert(!Files.exists(scratch.resolve(s"$variant/ir/$file.json")))
  }

  test("missing bounds and incompatible Property and Progress factory owners fail compilation") {
    for (variant, fragments) <- List(
        "missing" -> List("Missing.scala:11:46", "within"),
        "incompatible" -> List(
          "Incompatible.scala:14:27",
          "Incompatible.scala:16:29",
          "propertygroups.State",
          "propertygroups.foundations.incompatible.Incompatible"
        )
      )
    do
      val _ = packaged(variant, accepted = false)
      val output = Files.readString(scratch.resolve(variant).resolve("compile.log"))
      for fragment <- fragments do assert(output.contains(fragment), output)
      val errors = output.linesIterator.filter(_.startsWith("-- [E")).toList
      val expected = if variant.equals("missing") then List("[E171]") else List("[E007]", "[E007]")
      assertEquals(errors.map(_.takeWhile(_ != ':').stripPrefix("-- ").take(6)), expected, output)
  }

  test("resolved Property and Progress factories and aliases retain their original owner") {
    val jar = packaged("wrongOwner")
    val cases = List(
      ("safety", "misplaced", 22, "a Property"),
      ("safety", "alias", 23, "a Property"),
      ("safety", "bundle", 24, "a Property"),
      ("safety", "laterOwner", 25, "a Property"),
      ("liveness", "misplaced", 27, "a progress claim"),
      ("liveness", "alias", 28, "a progress claim"),
      ("liveness", "laterOwner", 29, "a progress claim")
    )
    val results = cases.map { (group, binding, line, kind) =>
      val out = scratch.resolve(s"wrongOwner/$group-$binding.json")
      val result = InProcess.lift(
        tools.directory,
        Seq(
          s"$jar=${stored}wrongOwner/,$commonJar=${stored}common/,$frameworkJar=model/framework/",
          frameworkClasspath.toString,
          out.toString,
          s"propertygroups.foundations.wrongowner.WrongOwner$$.properties$$.$group$$.$binding"
        )
      )
      Files.writeString(scratch.resolve(s"wrongOwner/$group-$binding.log"), result.output): Unit
      (group, binding, line, kind, out, result)
    }
    for (group, binding, line, kind, out, result) <- results do
      assertEquals(result.exit, 1, result.output)
      val name = if binding.equals("bundle") then "bundle.never"
      else if binding.equals("laterOwner") then
        if group.equals("safety") then "laterOwner.closed" else "laterOwner.last"
      else binding
      assertEquals(
        result.output.linesIterator.filter(_.startsWith("lift: ")).toList,
        List(
          s"lift: ${stored}wrongOwner/foundations/wrongowner/Wrongowner.scala:$line: $name is $kind of home, declared in wrongOwner.properties.$group: it belongs in home.properties.$group"
        )
      )
      assert(!Files.exists(out))
  }

  test("flat authored Property and Progress declarations retain source-attributed refusal") {
    val result = placementLift("flat", "specimen-workflow")
    assertEquals(result.exit, 1, result.output)
    val path = stored + "flat/features/specimen/workflow/WorkflowSpecimen.scala"
    for (line, owner, binding, kind, group) <- List(
        (19, "Subject", "closed", "a Property", "safety"),
        (21, "Subject", "aliased", "a Property", "safety"),
        (22, "Subject", "bundle.open", "a Property", "safety"),
        (22, "Subject", "bundle.closed", "a Property", "safety"),
        (26, "Subject", "reaches", "a progress claim", "liveness"),
        (27, "Subject", "factory", "a progress claim", "liveness"),
        (36, "Joined", "member", "a Property", "safety")
      )
    do
      assert(
        result.output.contains(
          s"lift: $path:$line: $binding is $kind, declared directly in " +
            s"$owner.properties: it belongs in the `$group` object of its machine's `properties`"
        ),
        result.output
      )
    assert(!Files.exists(scratch.resolve("flat/ir/specimen-workflow.json")))
  }

  private lazy val flatJar = packaged("flat")
  private lazy val groupedJar = packaged("grouped")
  private lazy val negativeJar = packaged("negative")
  private def lifted(variant: String): Map[String, JsonNode] =
    val jar = groupedJar
    val out = scratch.resolve(variant).resolve("ir")
    val result = InProcess.lift(
      tools.directory,
      Seq(
        "--ir",
        s"$jar=$stored$variant/,$commonJar=${stored}common/,$frameworkJar=model/framework/",
        frameworkClasspath.toString,
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

  private lazy val groupedModels = lifted("grouped")
  private def inventory(model: JsonNode, kind: String): List[String] =
    model
      .path(kind)
      .elements()
      .asScala
      .map(p => s"${p.path("machine").asText()}.${p.path("name").asText()}")
      .toList
      .sorted

  test(
    "grouped automatic discovery matches the independent whole inventory"
  ) {
    assertEquals(
      inventory(groupedModels("specimen-workflow"), "properties"),
      expectedProperties
    )
    assertEquals(
      inventory(groupedModels("specimen-workflow"), "progress"),
      expectedProgress
    )
    assertEquals(
      inventory(groupedModels("lonely"), "progress"),
      List("lonely.onlyProgress")
    )
    assertEquals(inventory(groupedModels("lonely"), "properties"), Nil)
  }

  test(
    "a non-form owner with only bundled unqueried Progress declares its file"
  ) {
    assertEquals(
      inventory(groupedModels("bundleonly"), "progress"),
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
        s"$negativeJar=${stored}negative/,$commonJar=${stored}common/,$frameworkJar=model/framework/",
        frameworkClasspath.toString,
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
    assert(result.output.contains("ignored is a progress claim"), result.output)
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

  private def legacy(grouped: JsonNode, name: String): JsonNode =
    val copy = grouped.deepCopy[JsonNode]()
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
        if file.startsWith(stored + "grouped/") then
          val suffix = file.stripPrefix(stored + "grouped/")
          val lines = suffix match
            case "features/specimen/workflow/WorkflowSpecimen.scala" =>
              workflowLines
            case "foundations/lonely/Lonely.scala" | "foundations/bundleonly/Bundleonly.scala" =>
              foundationLines
            case other => fail(s"unmapped source $other")
          val inverse = lines.map(_.swap)
          assert(inverse.contains(line), s"unmapped source span $file:$line")
          p.put("file", stored + "flat/" + suffix): Unit
          p.put("line", inverse(line)): Unit
        else
          assertEquals(file, stored + "common/Shared.scala")
          assert(line >= 1 && line <= 37, s"unmapped shared span $file:$line")
      n.elements().asScala.foreach(walk)
    walk(copy)
    copy
      .asInstanceOf[ObjectNode]
      .put(
        "source", // scalafix:ok DisableSyntax.asInstanceOf
        "model: " + flatRoots(name).sorted.mkString(", ")
      ): Unit
    copy

  private def canonical(n: JsonNode): JsonNode =
    if n.isObject then
      val out = mapper.createObjectNode()
      n.fields()
        .asScala
        .toSeq
        .sortBy(_.getKey)
        .foreach(e => out.set(e.getKey, canonical(e.getValue)))
      out
    else if n.isArray then
      val out = mapper.createArrayNode()
      n.elements().asScala.foreach(v => out.add(canonical(v)))
      out
    else n.deepCopy[JsonNode]()

  private def digest(n: JsonNode): String =
    java.security.MessageDigest
      .getInstance("SHA-256")
      .digest(mapper.writeValueAsBytes(canonical(n)))
      .map(b => f"${b & 255}%02x")
      .mkString

  // fn-169.1 moved the fixtures from model/irgen/testdata: their positions name the
  // new path, and the DSL its new package, framework.dsl; each digest with the old path and
  // package is the one it had.
  private lazy val legacyHashes = Map(
    "bundleonly" -> "04ee185ced447d933465579745ae8a8df6a8a773dfa92a15e444658c40d7b6bd",
    "lonely" -> "7d8d9682440f07e0b043cdbe2e28cc3cc372d8cdcbea5d23be16a8a07daa4d89",
    "specimen-workflow" -> "4f63a7ce5085e806d8292ca1a6cb1be9b291fe76e678885e4f1c0c9043dd02d7"
  )

  test(
    "every identity predicate attachment and metadata field matches the frozen flat baseline under the source map"
  ) {
    for name <- Seq("specimen-workflow", "lonely", "bundleonly") do
      assertEquals(
        groupedModels(name).path("source").asText(),
        "model: " + groupedRoots(name).sorted.mkString(", ")
      )
      assertEquals(digest(legacy(groupedModels(name), name)), legacyHashes(name))
  }

  test(
    "dropping a whole group from both candidate inventories fails the frozen declaration baseline"
  ) {
    val dropped = groupedModels("specimen-workflow").deepCopy[JsonNode]()
    dropped
      .asInstanceOf[ObjectNode]
      .remove("properties"): Unit // scalafix:ok DisableSyntax.asInstanceOf
    assert(!inventory(dropped, "properties").equals(expectedProperties))
    assertNotEquals(
      digest(legacy(dropped, "specimen-workflow")),
      legacyHashes("specimen-workflow")
    )
  }

  test(
    "native construction checks every applicable finite predicate row and retains Query selection"
  ) {
    val dependencies = Files
      .readString(frameworkClasspath)
      .trim
      .split(java.io.File.pathSeparator)
      .filter(_.endsWith(".jar"))
      .toSeq
    for (variant, jar) <- Seq("flat" -> flatJar, "grouped" -> groupedJar) do
      val classpath = (Seq(
        jar.toString,
        commonJar.toString,
        frameworkJar.toString
      ) ++ dependencies).distinct.mkString(java.io.File.pathSeparator)
      val result = tools.run(
        "java",
        Seq("-cp", classpath, "framework.dsl.PropertyGroupsNative")
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
    for variant <- Seq("grouped") do
      val model = groupedModels("specimen-workflow")
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
                """[{"name":"before","type":{"named":"propertygroups.State"}},{"name":"after","type":{"named":"framework.dsl.Step"}}]"""
              case "propertygroups.seal.next" =>
                """[{"name":"boolean","type":{"bool":{}}},{"name":"state","type":{"named":"propertygroups.State"}},{"name":"after","type":{"named":"framework.dsl.Step"}}]"""
              case "propertygroups.seal.violated" =>
                """[{"name":"s","type":{"bool":{}}}]"""
              case _ =>
                """[{"name":"after","type":{"named":"framework.dsl.Step"}}]"""
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
        val ir = groupedModels(file)
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
