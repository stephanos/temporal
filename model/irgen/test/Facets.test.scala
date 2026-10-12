package umpire.irgen

import java.nio.file.{Files, Path}
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import umpire.check.Tools
import com.fasterxml.jackson.databind.{JsonNode, ObjectMapper}
import com.fasterxml.jackson.databind.node.{ArrayNode, ObjectNode}

class FacetsTest extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val scratch = Files.createTempDirectory(tools.directory.resolve(".flow/tmp"), "facet-contract.")
  private val json = ObjectMapper()
  private val source = Files.readString(tools.directory.resolve("model/irgen/testdata/facetContract/Contract.scala"))
  private lazy val frameworkJar =
    val jar = scratch.resolve("framework.jar")
    tools.scalaCli(Seq("--power", "package", "--server=false", "--library", "model/project.scala", "model/framework", "-f", "-o", jar.toString)).orFail(): Unit
    jar
  private lazy val classpath =
    val ran = tools.scalaCli(Seq("compile", "--server=false", "--print-class-path", "model/project.scala", "model/framework")).orFail()
    val cp = ran.diagnostics.trim.split("\n").last
    val path = scratch.resolve("framework.classpath")
    Files.writeString(path, s"$frameworkJar:${cp.substring(cp.indexOf(':') + 1)}"): Unit
    path

  private def packaged(name: String, files: Map[String, String]): (Path, Path) =
    val dir = Files.createDirectory(scratch.resolve(name))
    for (path, text) <- files do
      val target = dir.resolve(path)
      Files.createDirectories(target.getParent): Unit
      Files.writeString(target, text): Unit
    Files.writeString(dir.resolve("project.scala"), "//> using scala 3.9.0\n//> using jvm 27\n"): Unit
    val jar = dir.resolve("fixture.jar")
    tools.inDirectory(dir).scalaCli(Seq("--power", "package", "--server=false", "--library", "--classpath", Files.readString(classpath).trim, ".", "-f", "-o", jar.toString)).orFail(): Unit
    (dir, jar)

  private lazy val original = packaged("original", Map("Contract.scala" -> source))
  private def lift(built: (Path, Path), name: String, roots: Seq[String], rules: Set[String] = Set.empty) =
    val (dir, jar) = built
    val out = dir.resolve(name + ".json")
    val ran = InProcess.lift(tools.directory, Seq(s"$jar=$dir/,$frameworkJar=model/", classpath.toString, out.toString) ++ roots, rules)
    (ran, out)
  private def root(name: String) = s"fixture.facetcontract.$name"
  private val both = Set("facets-only", "facet-vocabulary")
  private def refused(built: (Path, Path), name: String, selected: String, text: Seq[String], file: String, needle: String, rules: Set[String] = both): Unit =
    val (ran, out) = lift(built, name, Seq(root(selected)), rules)
    assertEquals(ran.exit, 1, ran.diagnostics)
    for expected <- text do assert(ran.diagnostics.contains(expected), ran.diagnostics)
    val lines = Files.readString(built._1.resolve(file)).linesIterator.toVector
    val line = lines.indexWhere(_.contains(needle)) + 1
    assert(line > 0)
    assert(ran.diagnostics.contains(s"$file:$line:"), ran.diagnostics)
    assert(!Files.exists(out))

  test("core Bindings refuses independently of misplaced effects and capabilities"):
    val (off, _) = lift(original, "core-off", Seq(root("Core")))
    assertEquals(off.exit, 0, off.diagnostics)
    refused(original, "core-on", "Core", Seq("outside its facets"), "Contract.scala", "object rules extends Bindings")
    val changed = source.replace("object rules extends Bindings(hand.press ~> Whole.effects.press)", "object effects:\n    def press(s: State) = Whole.effects.press(s)\n  object rules extends Bindings(hand.press ~> effects.press)\n  object capabilities extends Capabilities:\n    val own: Capability = Holdable(rejected = Outcome.ok)")
    val built = packaged("core-compound", Map("Contract.scala" -> changed))
    refused(built, "core-compound-on", "Core", Seq("effects outside its facets"), "Contract.scala", "  object effects:", Set("facets-only"))

  test("FacetCapabilities refuses own declarations and waivers separately from plain Capabilities"):
    for (name, body, needle) <- List(
      ("declaration", "val own: Capability = Holdable(rejected = Outcome.ok)", "val own:"),
      ("waiver", "except(Holdable.keeps[State], because = \"own-body waiver\")", "except(Holdable"),
      ("override", "overriding(Holdable.keeps[State] -> properties.keeps, because = \"own-body override\")", "overriding(Holdable")
    ) do
      val withProperty = source.replace("  object capabilities extends FacetCapabilities(pressing.capabilities)", "  object properties:\n    def keeps(m: Declares[State])(rejected: m.Outcome): Property[State] = m.property(\"keeps\").holds(_.outcome == rejected)\n  object capabilities extends FacetCapabilities(pressing.capabilities):\n    " + body)
      val built = packaged("own-" + name, Map("Contract.scala" -> withProperty))
      refused(built, name, "Faceted", Seq("capabilities", "outside its facets"), "Contract.scala", needle, Set("facets-only"))
    val plain = packaged("plain", Map("Contract.scala" -> source.replace("FacetCapabilities(pressing.capabilities)", "Capabilities")))
    refused(plain, "plain", "Faceted", Seq("capabilities outside its facets"), "Contract.scala", "object capabilities extends Capabilities", Set("facets-only"))

  test("non-facet arguments are refused independently by the lifter"):
    refused(original, "unowned", "Unowned", Seq("Facets lists", "loose", "rules of its facets"), "Contract.scala", "Facets(pressing.rules, loose)")

  test("ordinary, timer, fault and disabled duplicate owners retain action and facet names"):
    for (name, action, disabled) <- List(("ordinary", "hand.press", false), ("timer", "clock.tick", false), ("fault", "fault.stop", false), ("disabled", "hand.press", true)) do
      val first = if disabled then "disabled(hand.press)" else s"on($action) {\n        when(Phase.off) ~> effects.settle\n      }"
      val second = if disabled then "disabled(hand.press)" else s"on($action) {\n        when(Phase.on) ~> effects.settle\n      }"
      val changed = source.replace("on(hand.press) {\n        when(Phase.off) ~> effects.settle\n      }", first).replace("on(hand.smash) {\n        when(Phase.on) ~> effects.settle\n      }", second)
      val built = packaged("duplicate-" + name, Map("Contract.scala" -> changed))
      val (ran, out) = lift(built, name, Seq(root("Faceted")), both)
      assertEquals(ran.exit, 1, ran.diagnostics)
      for expected <- Seq(action.split('.').last, "pressing", "breaking", "belongs to one facet") do assert(ran.diagnostics.contains(expected), ran.diagnostics)
      assert(!Files.exists(out))

  test("ordered facet union and core functions agree under a closed declaration map"):
    val (ran, out) = lift(original, "equivalent", Seq(root("Faceted"), root("Whole"), root("Faceted$.capabilities"), root("Whole$.capabilities"), root("Faceted$.queries"), root("Whole$.queries")))
    assertEquals(ran.exit, 0, ran.diagnostics)
    val ir = json.readTree(Files.readString(out))
    val renames = Map("faceted" -> "whole", "faceted.rules.press" -> "whole.rules.press", "faceted.rules.smash" -> "whole.rules.smash", "faceted.evidence" -> "whole.evidence", "fixture.facetcontract.Pressing.pressing$.effects$.settle" -> "fixture.facetcontract.Whole$.effects$.press", "fixture.facetcontract.Breaking.breaking$.effects$.settle" -> "fixture.facetcontract.Whole$.effects$.smash")
    def meaning(node: JsonNode): JsonNode = node match
      case o: ObjectNode =>
        val copy = json.createObjectNode()
        for e <- o.fields().asScala if e.getKey != "position" do copy.set[JsonNode](e.getKey, meaning(e.getValue)): Unit
        copy
      case a: ArrayNode =>
        val copy = json.createArrayNode()
        for v <- a.elements().asScala do copy.add(meaning(v)): Unit
        copy
      case v if v.isTextual => json.getNodeFactory.textNode(renames.getOrElse(v.asText(), v.asText()))
      case v => v
    def named(section: String, name: String) = ir.path(section).elements().asScala.find(_.path("name").asText() == name).get
    assertEquals(meaning(named("machines", "faceted")), meaning(named("machines", "whole")))
    for (from, to) <- renames if from != "faceted" do
      val left = named("functions", from)
      val right = named("functions", to)
      assert(left.path("position").path("line").asInt() > 0)
      assert(right.path("position").path("line").asInt() > 0)
      assertEquals(meaning(left), meaning(right), s"$from -> $to")
    assertEquals(ir.path("machines").get(0).path("steps").elements().asScala.map(_.path("action").asText()).toVector, Vector("fixture.facetcontract.hand.press", "fixture.facetcontract.hand.smash"))
    val properties = ir.path("properties").elements().asScala.toVector
    assertEquals(properties.map(_.path("name").asText()).toSet, Set("faceted.keeps", "whole.keeps"))

  test("selected imported actions are not mandatory; derived and composed subjects stay exempt"):
    val (ran, _) = lift(original, "exempt", Seq(root("Faceted"), root("Disabled"), root("DerivedControl"), root("ComposedControl")), both)
    assertEquals(ran.exit, 0, ran.diagnostics)

  private lazy val layoutFiles =
    val base = tools.directory.resolve("model/irgen/testdata/facetContract/layout")
    val paths = Files.walk(base)
    try paths.iterator().asScala.filter(p => p.toString.endsWith(".scala")).map(p => base.relativize(p).toString -> Files.readString(p)).toMap
    finally paths.close()
  private lazy val layout = packaged("layout", layoutFiles)
  private def layoutLift(built: (Path, Path), name: String) = lift(built, name,
    Seq("fixture.features.contract.protocol.ContractProtocol", "fixture.features.contract.product.ContractProduct$.queries", "fixture.features.contract.protocol.ContractProtocol$.queries"), both)

  test("recognized level files accept the matching single facet and machine-wide declarations"):
    val (ran, out) = layoutLift(layout, "accepted")
    assertEquals(ran.exit, 0, ran.diagnostics)
    val ir = json.readTree(Files.readString(out))
    assertEquals(ir.path("machines").elements().asScala.map(_.path("stateType").asText()).toSet,
      Set("fixture.features.contract.product.QueueView", "fixture.features.contract.protocol.QueueDetail"))
    assertEquals(ir.path("queries").size(), 2)
    assertEquals(ir.path("monitors").size(), 1)

  test("actual facet files refuse a misnamed file and a second facet independently"):
    val path = "contract/product/Pressing.scala"
    for (name, files, message, needle) <- List(
      ("misnamed", (layoutFiles - path) + ("contract/product/Wrong.scala" -> layoutFiles(path)), "file is named after it", "trait Pressing"),
      ("second", layoutFiles.updated(path, layoutFiles(path) + "\ntrait Breaking extends Facet[QueueView, Outcome, Fact]\n"), "second facet", "trait Breaking")
    ) do
      val built = packaged("layout-" + name, files)
      val (ran, _) = layoutLift(built, name)
      assertEquals(ran.exit, 1, ran.diagnostics)
      assert(ran.diagnostics.contains(message), ran.diagnostics)
      val file = if name == "misnamed" then "contract/product/Wrong.scala" else path
      val line = files(file).linesIterator.toVector.indexWhere(_.contains(needle)) + 1
      assert(ran.diagnostics.contains(s"$file:$line"), ran.diagnostics)

  test("facet declarations outside or in the wrong sections refuse with their own positions"):
    val path = "contract/product/Pressing.scala"
    for (name, replacement, needle, expected) <- List(
      ("outer-effect", "  def stray(s: State) = enter(s, Fact.changed)\n  object pressing:", "def stray", "effects"),
      ("inner-effect", "  object pressing:\n    def stray(s: State) = enter(s, Fact.changed)", "def stray", "effects"),
      ("wrong-effect", "  object pressing:\n    object states:\n      def stray(s: State) = enter(s, Fact.changed)", "def stray", "effects"),
      ("action", "  object pressing:\n    val own = action(Actor(\"local\"))", "val own", "signature"),
      ("outer-rules", "  object rules extends Rules:\n    disabled(hand.press)\n  object pressing:", "object rules", "facet"),
      ("capability", "  object pressing:\n    val own: CapabilityOf[QueueView, Outcome, Fact] = fixture.facetcontract.Holdable(rejected = Outcome.ok)", "val own", "capabilities")
    ) do
      val changed = layoutFiles(path).replace("  object pressing:", replacement)
      val built = packaged("placement-" + name, layoutFiles.updated(path, changed) + ("Contract.scala" -> source))
      val (ran, _) = layoutLift(built, name)
      assertEquals(ran.exit, 1, ran.diagnostics)
      assert(ran.diagnostics.contains(expected), ran.diagnostics)
      val line = changed.linesIterator.toVector.indexWhere(_.contains(needle)) + 1
      assert(ran.diagnostics.contains(s"$path:$line:"), ran.diagnostics)

  test("unknown vocabulary and crossed ordinary non-State levels are refused independently"):
    val path = "contract/protocol/Pressing.scala"
    for (name, vocab) <- List("crossed" -> "case pressing, breaking", "unknown" -> "case pressing") do
      val files = layoutFiles
        .updated("contract/Contract.scala", layoutFiles("contract/Contract.scala").replace("case pressing, breaking", vocab))
        .updated("contract/protocol/Protocol.scala", layoutFiles("contract/protocol/Protocol.scala").replace("Pressing", "Breaking").replace("pressing.rules", "breaking.rules")) - path
      val renamed = files + ("contract/protocol/Breaking.scala" -> layoutFiles(path).replace("Pressing", "Breaking").replace("object pressing", "object breaking"))
      val built = packaged("pair-" + name, renamed)
      val (ran, _) = layoutLift(built, name)
      assertEquals(ran.exit, 1, ran.diagnostics)
      for expected <- (if name == "crossed" then Seq("press", "pressing", "breaking", "same-named") else Seq("breaking", "vocabulary")) do assert(ran.diagnostics.contains(expected), ran.diagnostics)

  test("a genuinely separate design state retains its exemption"):
    val existing = Files.readString(tools.directory.resolve("model/irgen/testdata/facetRules/FacetRules.scala"))
    val built = packaged("design", Map("FacetRules.scala" -> existing))
    val (ran, _) = lift(built, "design", Seq("fixture.facetrules.Design"), both)
    assertEquals(ran.exit, 0, ran.diagnostics)

  test("an unliftable facet section names its actual facet and declaration"):
    val changed = source.replace("def settle(s: State) = pressing.effects.settle(s)", "def settle(s: State) =\n        var next = s\n        next = s.copy(phase = Phase.broken)\n        enter[State, Outcome, Fact](next, Fact.changed)")
    val built = packaged("unliftable", Map("Contract.scala" -> changed))
    val (ran, _) = lift(built, "unliftable", Seq(root("Faceted")), both)
    assertEquals(ran.exit, 1, ran.diagnostics)
    assert(ran.diagnostics.contains("next") && ran.diagnostics.contains("Contract.scala:"), ran.diagnostics)
