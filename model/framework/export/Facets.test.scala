package framework.`export`

import java.nio.file.{Files, Path}
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import framework.check.Tools
import com.fasterxml.jackson.databind.{JsonNode, ObjectMapper}
import com.fasterxml.jackson.databind.node.{ArrayNode, ObjectNode}
import FixtureBuild.buildFixture
import framework.check.RetainedScratch

class FacetsTest extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val scratch =
    retained(tools.directory.resolve(".flow/tmp"), "facet-contract.")
  private val json = ObjectMapper()
  private val source =
    Files.readString(
      tools.directory.resolve("model/framework/export/testdata/facetContract/Contract.scala")
    )
  // The packaged Models (ModelBuild): the framework this fixture reads, with the shared code and
  // the mini, which its lifts admit only where its roots reach them.
  private def frameworkJar = ModelBuild.jar
  private def classpath = ModelBuild.classpath

  private def packaged(name: String, files: Map[String, String]): (Path, Path) =
    val dir = Files.createDirectory(scratch.resolve(name))
    for (path, text) <- files do
      val target = dir.resolve(path)
      Files.createDirectories(target.getParent): Unit
      Files.writeString(target, text): Unit
    Files.writeString(
      dir.resolve("project.scala"),
      "//> using scala 3.9.0\n//> using jvm 27\n"
    ): Unit
    val jar = dir.resolve("fixture.jar")
    tools
      .inDirectory(dir)
      .buildFixture(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          "--classpath",
          ModelBuild.classpathEntries,
          ".",
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    (dir, jar)

  private lazy val original = packaged(
    "original",
    Map(
      "Contract.scala" -> source,
      "Rows.scala" -> Files.readString(
        tools.directory.resolve("model/framework/export/testdata/facetContract/Rows.scala")
      )
    )
  )
  private def lift(
      built: (Path, Path),
      name: String,
      roots: Seq[String],
      rules: Set[String] = Set.empty
  ) =
    val (dir, jar) = built
    val out = dir.resolve(name + ".json")
    val ran = InProcess.lift(
      tools.directory,
      Seq(s"$jar=$dir/,$frameworkJar=model/", classpath.toString, out.toString) ++ roots,
      rules
    )
    (ran, out)
  private def root(name: String) = s"fixture.facetcontract.$name"
  private val both = Set("facets-only", "facet-vocabulary")
  private def refused(
      built: (Path, Path),
      name: String,
      selected: String,
      text: Seq[String],
      file: String,
      needle: String,
      rules: Set[String] = both
  ): Unit =
    val (ran, out) = lift(built, name, Seq(root(selected)), rules)
    assertEquals(ran.exit, 1, ran.diagnostics)
    for expected <- text do assert(ran.diagnostics.contains(expected), ran.diagnostics)
    val lines = Files.readString(built._1.resolve(file)).linesIterator.toVector
    val ownerLine = lines.indexWhere(_.startsWith(s"object $selected"))
    val line = lines.indexWhere(_.contains(needle), ownerLine.max(0)) + 1
    assert(line > 0)
    assert(ran.diagnostics.contains(s"$file:$line:"), ran.diagnostics)
    assert(!Files.exists(out))

  test("core Bindings refuses independently of misplaced effects and capabilities"):
    val (off, _) = lift(original, "core-off", Seq(root("Core")))
    assertEquals(off.exit, 0, off.diagnostics)
    refused(
      original,
      "core-on",
      "Core",
      Seq("outside its facets"),
      "Contract.scala",
      "object rules extends Bindings"
    )
    val changed = source.replace(
      "object rules extends Bindings(hand.press ~> Whole.effects.press)",
      "object effects:\n    def press(s: State) = Whole.effects.press(s)\n  object rules extends Bindings(hand.press ~> effects.press)\n  object capabilities extends Capabilities:\n    val own: Capability = Holdable(rejected = Outcome.ok)"
    )
    val built = packaged("core-compound", Map("Contract.scala" -> changed))
    refused(
      built,
      "core-compound-on",
      "Core",
      Seq("effects outside its facets"),
      "Contract.scala",
      "  object effects:",
      Set("facets-only")
    )

  for (name, body, needle) <- List(
      ("declaration", "val own: Capability = Holdable(rejected = Outcome.ok)", "val own:"),
      ("waiver", "except(Holdable.keeps[State], because = \"own-body waiver\")", "except(Holdable"),
      (
        "override",
        "overriding(Holdable.keeps[State] -> properties.keeps, because = \"own-body override\")",
        "overriding(Holdable"
      )
    )
  do
    test(s"machine FacetCapabilities refuses its own $name"):
      val withProperty = source.replace(
        "  object capabilities extends FacetCapabilities(pressing.capabilities)",
        "  object properties:\n    def keeps(m: Declares[State])(rejected: m.Outcome): Property[State] = m.property(\"keeps\").holds(_.outcome == rejected)\n  object capabilities extends FacetCapabilities(pressing.capabilities):\n    " + body
      )
      val built = packaged("own-" + name, Map("Contract.scala" -> withProperty))
      refused(
        built,
        name,
        "Faceted",
        Seq("capabilities", "outside its facets"),
        "Contract.scala",
        needle,
        Set("facets-only")
      )
  test("plain machine Capabilities is refused independently"):
    val plain = packaged(
      "plain",
      Map(
        "Contract.scala" -> source.replace(
          "FacetCapabilities(pressing.capabilities)",
          "Capabilities"
        )
      )
    )
    refused(
      plain,
      "plain",
      "Faceted",
      Seq("capabilities outside its facets"),
      "Contract.scala",
      "object capabilities extends Capabilities",
      Set("facets-only")
    )

  test("a FacetCapabilities subclass cannot supply non-facet declaration provenance"):
    val body = """abstract class SharedCaps(using Declaring[State, Outcome, Fact])(
    facet: Capabilities[State, Outcome, Fact]
) extends FacetCapabilities(facet):
  val inherited: Capability = Holdable(rejected = Outcome.ok)

"""
    val changed = source
      .replace("object Faceted", body + "object Faceted")
      .replace("FacetCapabilities(pressing.capabilities)", "SharedCaps(pressing.capabilities)")
    val built = packaged("inherited-caps", Map("Contract.scala" -> changed))
    val (ran, out) = lift(built, "inherited", Seq(root("Faceted")), both)
    assertEquals(ran.exit, 1, ran.diagnostics)
    assert(ran.diagnostics.contains("outside its facets"), ran.diagnostics)
    val line = changed.linesIterator.toVector.indexWhere(_.contains("val inherited:")) + 1
    assert(ran.diagnostics.contains(s"Contract.scala:$line:"), ran.diagnostics)
    assert(!Files.exists(out))

  test("unowned ordinary, timer, fault and disabled arguments refuse independently by the lifter"):
    for (name, target, binding) <- List(
        ("ordinary", "hand.smash", "on(hand.smash) { always ~> Whole.effects.smash }"),
        ("timer", "clock.tick", "on(clock.tick) { always ~> Whole.effects.smash }"),
        ("fault", "fault.stop", "on(fault.stop) { always ~> Whole.effects.smash }"),
        ("disabled", "hand.smash", "disabled(hand.smash)")
      )
    do
      val built = packaged(
        "unowned-" + name,
        Map("Contract.scala" -> source.replace("    disabled(hand.smash)", "    " + binding))
      )
      refused(
        built,
        name,
        "Unowned",
        Seq("Facets lists", "loose", s"${target.split('.').last} has no owning facet"),
        "Contract.scala",
        "Facets(pressing.rules, loose)"
      )

  test("ordinary, timer, fault and disabled duplicate owners retain action and facet names"):
    for (name, action, disabled) <- List(
        ("ordinary", "hand.press", false),
        ("timer", "clock.tick", false),
        ("fault", "fault.stop", false),
        ("disabled", "hand.press", true)
      )
    do
      val first = if disabled then "disabled(hand.press)"
      else s"on($action) {\n        when(Phase.off) ~> effects.settle\n      }"
      val second = if disabled then "disabled(hand.press)"
      else s"on($action) {\n        when(Phase.on) ~> effects.settle\n      }"
      val changed = source
        .replace("on(hand.press) {\n        when(Phase.off) ~> effects.settle\n      }", first)
        .replace("on(hand.smash) {\n        when(Phase.on) ~> effects.settle\n      }", second)
      val built = packaged("duplicate-" + name, Map("Contract.scala" -> changed))
      val (ran, out) = lift(built, name, Seq(root("Faceted")), both)
      assertEquals(ran.exit, 1, ran.diagnostics)
      for expected <- Seq(action.split('.').last, "pressing", "breaking", "belongs to one facet") do
        assert(ran.diagnostics.contains(expected), ran.diagnostics)
      assert(!Files.exists(out))

  test("ordered facet union and core functions agree under a closed declaration map"):
    val native = tools
      .inDirectory(original._1)
      .buildFixture(
        Seq(
          "run",
          "--server=false",
          "--classpath",
          ModelBuild.classpathEntries,
          ".",
          "-M",
          "framework.dsl.nativeRows"
        )
      )
    assertEquals(native.exit, 0, native.diagnostics)
    assert(
      native.diagnostics.contains("PASS: 6 complete native state/action pairs"),
      native.diagnostics
    )
    val (ran, out) = lift(
      original,
      "equivalent",
      Seq(
        root("Faceted"),
        root("Whole"),
        root("Faceted$.capabilities"),
        root("Whole$.capabilities"),
        root("Faceted$.queries"),
        root("Whole$.queries")
      )
    )
    assertEquals(ran.exit, 0, ran.diagnostics)
    val ir = json.readTree(Files.readString(out))
    val functionRenames = Map(
      "faceted.rules.press" -> "whole.rules.press",
      "faceted.rules.smash" -> "whole.rules.smash",
      "faceted.evidence" -> "whole.evidence",
      "faceted.property.faceted.keeps" -> "whole.property.whole.keeps",
      "fixture.facetcontract.Pressing.pressing$.effects$.settle" -> "fixture.facetcontract.Whole$.effects$.press",
      "fixture.facetcontract.Breaking.breaking$.effects$.settle" -> "fixture.facetcontract.Whole$.effects$.smash"
    )
    val renames = functionRenames ++ Map("faceted" -> "whole", "faceted.keeps" -> "whole.keeps")
    val lines = source.linesIterator.toVector
    def line(owner: String, needle: String): Int =
      val start = lines.indexWhere(_.startsWith(owner))
      assert(start >= 0, owner)
      val found = lines.indexWhere(_.contains(needle), start)
      assert(found >= 0, s"$owner: $needle")
      found + 1
    val lineMap = List(
      ("trait Pressing", "def settle", "object Whole", "def press"),
      ("trait Pressing", "on(hand.press)", "object Whole", "on(hand.press)"),
      ("trait Pressing", "when(Phase.off)", "object Whole", "when(Phase.off)"),
      ("trait Pressing", "val holdable", "object Whole", "val holdable"),
      ("trait Breaking", "def settle", "object Whole", "def smash"),
      ("trait Breaking", "on(hand.smash)", "object Whole", "on(hand.smash)"),
      ("trait Breaking", "when(Phase.on)", "object Whole", "when(Phase.on)"),
      ("object Faceted", "object Faceted", "object Whole", "object Whole"),
      ("object Faceted", "Phased[", "object Whole", "Phased["),
      ("object Faceted", "val init", "object Whole", "val init"),
      ("object Faceted", "object rules", "object Whole", "object rules")
    ).map((a, b, c, d) => line(a, b) -> line(c, d)).toMap
    val usedLines = scala.collection.mutable.Set.empty[Int]
    def meaning(node: JsonNode): JsonNode = node match
      case position: ObjectNode if position.has("file") && position.has("line") =>
        assertEquals(position.path("file").asText(), original._1.resolve("Contract.scala").toString)
        val at = position.path("line").asInt()
        assert(at > 0 && at <= lines.size && lines(at - 1).trim.nonEmpty)
        val copy = position.deepCopy[ObjectNode]()
        for replacement <- lineMap.get(at) do
          usedLines += at
          copy.put("line", replacement): Unit
        copy
      case o: ObjectNode =>
        val copy = json.createObjectNode()
        for e <- o.fields().asScala do copy.set[JsonNode](e.getKey, meaning(e.getValue)): Unit
        copy
      case a: ArrayNode =>
        val copy = json.createArrayNode()
        for v <- a.elements().asScala do copy.add(meaning(v)): Unit
        copy
      case v if v.isTextual =>
        json.getNodeFactory.textNode(renames.getOrElse(v.asText(), v.asText()))
      case v => v
    def named(section: String, name: String) =
      ir.path(section).elements().asScala.find(_.path("name").asText() == name).get
    assertEquals(meaning(named("machines", "faceted")), meaning(named("machines", "whole")))
    for (from, to) <- functionRenames do
      val left = named("functions", from)
      val right = named("functions", to)
      assert(left.path("position").path("line").asInt() > 0)
      assert(right.path("position").path("line").asInt() > 0)
      assertEquals(meaning(left), meaning(right), s"$from -> $to")
    assertEquals(
      ir.path("machines")
        .get(0)
        .path("steps")
        .elements()
        .asScala
        .map(_.path("action").asText())
        .toVector,
      Vector("fixture.facetcontract.hand.press", "fixture.facetcontract.hand.smash")
    )
    val properties = ir.path("properties").elements().asScala.toVector
    assertEquals(properties.map(_.path("name").asText()).toSet, Set("faceted.keeps", "whole.keeps"))
    for section <- Seq("properties", "scenarios", "queries") do
      assertEquals(meaning(named(section, "faceted.keeps")), meaning(named(section, "whole.keeps")))
    assertEquals(usedLines.toSet, lineMap.keySet)
    Files.writeString(
      original._1.resolve("equivalence-line-map.txt"),
      lineMap.toVector.sorted
        .map((a, b) => s"$a -> $b: ${lines(a - 1).trim} -> ${lines(b - 1).trim}")
        .mkString("\n")
    ): Unit

  test("selected imported actions are not mandatory; derived and composed subjects stay exempt"):
    val (ran, _) = lift(
      original,
      "exempt",
      Seq(root("Faceted"), root("Disabled"), root("DerivedControl"), root("ComposedControl")),
      both
    )
    assertEquals(ran.exit, 0, ran.diagnostics)

  private lazy val layoutFiles =
    val base = tools.directory.resolve("model/framework/export/testdata/facetContract/layout")
    val paths = Files.walk(base)
    try
      paths
        .iterator()
        .asScala
        .filter(p => p.toString.endsWith(".scala"))
        .map(p => base.relativize(p).toString -> Files.readString(p))
        .toMap
    finally paths.close()
  private lazy val layout = packaged("layout", layoutFiles)
  private def layoutLift(built: (Path, Path), name: String) = lift(
    built,
    name,
    Seq(
      "fixture.features.contract.protocol.ContractProtocol",
      "fixture.features.contract.product.ContractProduct$.queries",
      "fixture.features.contract.protocol.ContractProtocol$.queries"
    ),
    both
  )

  test("recognized level files accept the matching single facet and machine-wide declarations"):
    val (ran, out) = layoutLift(layout, "accepted")
    assertEquals(ran.exit, 0, ran.diagnostics)
    val ir = json.readTree(Files.readString(out))
    assertEquals(
      ir.path("machines").elements().asScala.map(_.path("stateType").asText()).toSet,
      Set(
        "fixture.features.contract.product.QueueView",
        "fixture.features.contract.protocol.QueueDetail"
      )
    )
    assertEquals(ir.path("queries").size(), 2)
    assertEquals(ir.path("monitors").size(), 1)

  test("actual facet files refuse a misnamed file and a second facet independently"):
    val path = "contract/product/Pressing.scala"
    for (name, files, message, needle) <- List(
        (
          "misnamed",
          (layoutFiles - path) + ("contract/product/Wrong.scala" -> layoutFiles(path)),
          "file is named after it",
          "trait Pressing"
        ),
        (
          "second",
          layoutFiles.updated(
            path,
            layoutFiles(path) + "\ntrait Breaking extends Facet[QueueView, Outcome, Fact]\n"
          ),
          "second facet",
          "trait Breaking"
        )
      )
    do
      val built = packaged("layout-" + name, files)
      val (ran, _) = layoutLift(built, name)
      assertEquals(ran.exit, 1, ran.diagnostics)
      assert(ran.diagnostics.contains(message), ran.diagnostics)
      val file = if name == "misnamed" then "contract/product/Wrong.scala" else path
      val line = files(file).linesIterator.toVector.indexWhere(_.contains(needle)) + 1
      assert(ran.diagnostics.contains(s"$file:$line"), ran.diagnostics)

  for (name, replacement, needle, expected) <- List(
      (
        "outer-effect",
        "  def stray(s: State) = enter(s, Fact.changed)\n  object pressing:",
        "def stray",
        "effects"
      ),
      (
        "inner-effect",
        "  object pressing:\n    def stray(s: State) = enter(s, Fact.changed)",
        "def stray",
        "effects"
      ),
      (
        "wrong-effect",
        "  object pressing:\n    object states:\n      def stray(s: State) = enter(s, Fact.changed)",
        "def stray",
        "effects"
      ),
      (
        "action",
        "  object pressing:\n    val own = action(Actor(\"local\"))",
        "val own",
        "signature"
      ),
      (
        "outer-rules",
        "  object strayRules extends Rules:\n    disabled(hand.press)\n  object pressing:",
        "object strayRules",
        "facet"
      ),
      (
        "capability",
        "  object pressing:\n    val own: CapabilityOf[QueueView, Outcome, Fact] = fixture.facetcontract.Holdable(rejected = Outcome.ok)",
        "val own",
        "capabilities"
      )
    )
  do
    test(s"facet placement $name refuses at its own position"):
      val path = "contract/product/Pressing.scala"
      val changed = layoutFiles(path).replace("  object pressing:", replacement)
      val built = packaged(
        "placement-" + name,
        layoutFiles.updated(path, changed) + ("Contract.scala" -> source)
      )
      val (ran, _) = layoutLift(built, name)
      assertEquals(ran.exit, 1, ran.diagnostics)
      assert(ran.diagnostics.contains(expected), ran.diagnostics)
      val line = changed.linesIterator.toVector.indexWhere(_.contains(needle)) + 1
      assert(ran.diagnostics.contains(s"$path:$line:"), ran.diagnostics)

  for
    level <- List("product", "protocol");
    // An unknown facet's file is named for a design family (fn-160 R7), so the family lint passes it
    // and the vocabulary refusal is the one left.
    (name, vocab) <- List(
      "crossed" -> "case pressing, breaking",
      "unknown" -> "case pressing\nenum FamilyName:\n  case breaking"
    )
  do
    test(s"$name vocabulary at $level in ordinary non-State levels is refused"):
      val path = s"contract/$level/Pressing.scala"
      val primary = if level == "product" then "Product.scala" else "Protocol.scala"
      val primaryPath = s"contract/$level/$primary"
      val files = layoutFiles
        .updated(
          "contract/Contract.scala",
          layoutFiles("contract/Contract.scala").replace("case pressing, breaking", vocab)
        )
        .updated(
          primaryPath,
          layoutFiles(primaryPath)
            .replace("Pressing", "Breaking")
            .replace("pressing.rules", "breaking.rules")
        ) - path
      val renamed = files + (s"contract/$level/Breaking.scala" -> layoutFiles(path)
        .replace("Pressing", "Breaking")
        .replace("object pressing", "object breaking"))
      val built = packaged(s"pair-$level-$name", renamed)
      val (ran, _) = layoutLift(built, name)
      assertEquals(ran.exit, 1, ran.diagnostics)
      for expected <-
          if name == "crossed" then Seq("press", "pressing", "breaking", "same-named")
          else Seq("breaking", "vocabulary")
      do assert(ran.diagnostics.contains(expected), ran.diagnostics)

  test("a genuinely separate design state retains its exemption"):
    val existing =
      Files.readString(
        tools.directory.resolve("model/framework/export/testdata/facetRules/FacetRules.scala")
      )
    val built = packaged("design", Map("FacetRules.scala" -> existing))
    val (ran, _) = lift(built, "design", Seq("fixture.facetrules.Design"), both)
    assertEquals(ran.exit, 0, ran.diagnostics)

  test("an unliftable facet section names its actual facet and declaration"):
    val changed = source.replace(
      "def settle(s: State) = pressing.effects.settle(s)",
      "def settle(s: State) =\n        var next = s\n        next = s.copy(phase = Phase.broken)\n        enter[State, Outcome, Fact](next, Fact.changed)"
    )
    val built = packaged("unliftable", Map("Contract.scala" -> changed))
    val (ran, _) = lift(built, "unliftable", Seq(root("Faceted")), both)
    assertEquals(ran.exit, 1, ran.diagnostics)
    assert(
      ran.diagnostics.contains("next") && ran.diagnostics.contains(
        "Contract.scala:"
      ) && ran.diagnostics.contains("breaking.effects"),
      ran.diagnostics
    )
