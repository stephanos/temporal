package framework.`export`

import java.nio.file.{Files, Path}
import java.nio.file.attribute.PosixFilePermissions
import java.util.Base64
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import com.fasterxml.jackson.databind.{JsonNode, ObjectMapper}
import framework.check.{Output, Ran, Tools}
import io.temporal.server.api.umpire.v1 as ir

class ConstructedFacetsSuite extends munit.FunSuite:
  override val munitTimeout: Duration = 20.minutes
  private val tools = Tools.here
  private val mapper = new ObjectMapper()
  private lazy val scratch =
    Files.createTempDirectory(
      tools.directory.resolve(".flow/tmp"),
      "constructed-facets."
    )
  private val stored =
    tools.directory.resolve("model/framework/export/testdata/constructedFacets")
  private lazy val library =
    val jar = scratch.resolve("framework.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          "model/framework/dsl",
          "model/framework/realize",
          "--server=false",
          "--library",
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    jar
  private lazy val plugin =
    val jar = scratch.resolve("reify.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          "model/framework/reify/plugin",
          "--server=false",
          "--assembly",
          "--preamble=false",
          "--main-class-in-manifest=false",
          "--provided",
          "org.scala-lang:scala3-compiler_3",
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    jar
  private def settings(withPlugin: Boolean) = Seq(
    "--scala",
    "3.9.0",
    "--jvm",
    "27",
    "--server=false",
    "--dependency",
    "com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20",
    "--jar",
    library.toString,
    "--jar",
    tools.directory.resolve("model/build/ir-scalapb.jar").toString
  ) ++
    (if withPlugin then Seq("--scalac-option", s"-Xplugin:$plugin") else Nil) ++
    Seq(
      "-Werror",
      "-deprecation",
      "-feature",
      "-unchecked",
      "-Wunused:all",
      "-Wvalue-discard",
      "-Wnonunit-statement",
      "-Wsafe-init",
      "-Wimplausible-patterns",
      "-language:strictEquality"
    )
      .flatMap(o => Seq("--scalac-option", o))
  private def recorded(name: String, result: Ran): Ran =
    Files.writeString(
      scratch.resolve(s"$name.log"),
      s"exit=${result.exit}\n${result.diagnostics}"
    ): Unit
    Files.writeString(
      scratch.resolve(s"$name.command"),
      result.command.mkString(" ")
    ): Unit
    result
  private def compile(
      names: Seq[String],
      name: String,
      withPlugin: Boolean,
      extra: Seq[String] = Nil,
      changes: Map[String, String] = Map.empty
  ): (Path, Path, String) =
    val dir = Files.createDirectories(scratch.resolve(name))
    names.foreach { n =>
      changes.get(n) match
        case Some(source) => Files.writeString(dir.resolve(n), source): Unit
        case None         => Files.copy(stored.resolve(n), dir.resolve(n)): Unit
    }
    val jar = scratch.resolve(s"$name.jar")
    val built = recorded(
      s"$name.compile",
      tools.scalaCli(
        Seq(
          "compile",
          dir.toString,
          "--print-class-path",
          "--compilation-output",
          jar.toString
        ) ++ settings(withPlugin) ++ extra,
        Output.KeptApart
      )
    )
    assertEquals(built.exit, 0, built.diagnostics)
    built.orFail(): Unit
    Files.writeString(
      scratch.resolve(s"$name.classpath"),
      built.output.trim
    ): Unit
    (dir, jar, built.output.trim)

  private lazy val refusalOld: Unit =
    val (dir, jar, cp) = compile(Seq("Refusals.scala"), "refusals-old", false)
    Seq(
      ("Duplicate", "go is bound by the facets left and right", "      on(hand.go)(where(s => s)"),
      ("Repeated", "stop is disabled twice, or disabled and fired", "      disabled(fault.stop)"),
      ("EmptyEffect", "unused is an effect and gives no step here", "      def unused(s: Boolean)"),
      ("Unowned", "go has no owning facet", "  object rules extends Facets(loose)")
    ).foreach { (root, message, needle) =>
      val name = s"refusal-old-$root"
      val classpath = Files.writeString(scratch.resolve(s"$name.classpath"), cp)
      val output = scratch.resolve(s"$name.json")
      val result = recorded(
        name,
        InProcess.frameworkLift(
          tools.directory,
          Seq(
            s"$jar=$dir/,$library=model/framework/",
            classpath.toString,
            output.toString,
            s"fixture.facetsrefusals.$root"
          )
        )
      )
      assertEquals(result.exit, 1, result.diagnostics)
      assert(!Files.exists(output))
      val line = Files
        .readString(dir.resolve("Refusals.scala"))
        .linesIterator
        .toVector
        .indexWhere(_.startsWith(needle)) + 1
      assert(result.diagnostics.contains(s"Refusals.scala:$line:"), result.diagnostics)
      assert(result.diagnostics.contains(message), result.diagnostics)
    }
    val vocabulary = recorded(
      "refusal-old-vocabulary",
      InProcess.frameworkLift(
        tools.directory,
        Seq(
          s"$jar=$dir/,$library=model/framework/",
          scratch.resolve("refusal-old-Duplicate.classpath").toString,
          scratch.resolve("refusal-old-vocabulary.json").toString,
          "fixture.facetsrefusals.Duplicate"
        ),
        Set("facet-vocabulary")
      )
    )
    assertEquals(vocabulary.exit, 1, vocabulary.diagnostics)
    assert(vocabulary.diagnostics.contains("facet vocabulary"), vocabulary.diagnostics)
    val clients = Files
      .readString(stored.resolve("Clients.scala"))
      .replace(", (s, p) => s.copy(left = p)", "")
      .replace(", (s, p) => s.copy(right = p)", "")
    val (missingDir, missingJar, missingCp) =
      compile(sourceNames, "missing-writer-old", false, changes = Map("Clients.scala" -> clients))
    val missingClasspath =
      Files.writeString(scratch.resolve("missing-writer-old.classpath"), missingCp)
    roots.zip(Seq("a", "b")).foreach { (root, name) =>
      val missing = recorded(
        s"missing-writer-old-$name",
        InProcess.frameworkLift(
          tools.directory,
          Seq(
            s"$missingJar=$missingDir/,$library=model/framework/",
            missingClasspath.toString,
            scratch.resolve(s"missing-writer-old-$name.json").toString,
            root
          )
        )
      )
      assertEquals(missing.exit, 1, missing.diagnostics)
      assert(missing.diagnostics.contains("declares none"), missing.diagnostics)
    }
  private val sourceNames =
    Seq("Signature.scala", "Pressing.scala", "Clients.scala", "MultiClients.scala")
  private val roots = Seq("fixture.constructedfacets.A", "fixture.constructedfacets.B")
  private val contextual =
    roots ++ Seq("fixture.constructedfacets.MultiA", "fixture.constructedfacets.MultiB")
  private val contextualNames = Seq("a", "b", "ma", "mb")
  private val ordinarySources = Seq(
    "Shared.scala",
    "Targets.scala",
    "Arities.scala",
    "Domains.scala",
    "Sections.scala",
    "Contributors.scala"
  )
  private val ordinaryRoots = Seq(
    "WholeOutside",
    "WholeInFrom",
    "ClassOutside",
    "ClassInFrom",
    "TwoOutside",
    "TwoInFrom",
    "ThreeOutside",
    "ThreeInFrom",
    "FourOutside",
    "FourInFrom"
  ).map("fixture.facetstargets." + _) ++ Seq(
    "fixture.facetsshapes.Arities",
    "fixture.facetsshapes.PhaseRules",
    "fixture.facetsdomains.Domains",
    "fixture.facetssections.Sections",
    "fixture.facetssections.Rejecting",
    "fixture.facetscontributors.Forward",
    "fixture.facetscontributors.Reverse"
  )

  private def lift(
      sources: String,
      cp: String,
      name: String,
      selected: Seq[String]
  ): ir.Model =
    val classpath = Files.writeString(scratch.resolve(s"$name.classpath"), cp)
    val output = scratch.resolve(s"$name.json")
    val result = recorded(
      name,
      InProcess.frameworkLift(
        tools.directory,
        Seq(
          sources + s",$library=model/framework/",
          classpath.toString,
          output.toString
        ) ++ selected
      )
    )
    result.orFail(): Unit
    scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(output))

  private def withoutPositions(node: JsonNode): JsonNode =
    if node.isObject then
      val out = mapper.createObjectNode()
      for field <- node.fields().asScala if field.getKey != "position" do
        out.set[JsonNode](field.getKey, withoutPositions(field.getValue)): Unit
      out
    else if node.isArray then
      val out = mapper.createArrayNode()
      node.elements().asScala.foreach(n => out.add(withoutPositions(n)): Unit)
      out
    else node
  private def comparable(model: ir.Model): JsonNode =
    withoutPositions(
      mapper.readTree(scalapb.json4s.JsonFormat.toJsonString(model))
    )
  private def frozen(name: String): ir.Model = scalapb.json4s.JsonFormat
    .fromJsonString[ir.Model](Files.readString(stored.resolve(name)))
  private def projected(model: ir.Model): ir.Model =
    val functions = scala.collection.mutable.Set.empty[String]
    val types = scala.collection.mutable.Set.empty[String]
    def read(value: scalapb.GeneratedMessage): Unit =
      scan(mapper.readTree(scalapb.json4s.JsonFormat.toJsonString(value)))
    def function(name: String): Unit =
      if functions.add(name) then read(model.functions.find(_.name == name).get)
    def tpe(name: String): Unit =
      if name != "scala.Nothing" && name != "framework.dsl.Step" && types.add(
          name
        )
      then read(model.types.find(_.name == name).get)
    def scan(node: JsonNode): Unit =
      if node.isObject then
        if node.has("call") then function(node.path("call").path("function").asText())
        if node.has("named") then tpe(node.path("named").asText())
        for
          kind <- Seq("construct", "enum", "record", "case")
          if node.path(kind).has("type")
        do tpe(node.path(kind).path("type").asText())
        node.elements().asScala.foreach(scan)
      else if node.isArray then node.elements().asScala.foreach(scan)
    model.machines.flatMap(_.steps).foreach(s => function(s.function))
    ir.Model(
      machines = model.machines
        .map(m => ir.Machine(name = m.name, steps = m.steps))
        .sortBy(_.name),
      functions = model.functions.filter(f => functions(f.name)).sortBy(_.name),
      types = model.types.filter(t => types(t.name)).sortBy(_.name)
    )
  private def nodePositions(model: ir.Model): Map[String, (String, Int)] =
    val positions = scala.collection.mutable.Map.empty[String, (String, Int)]
    def scan(node: JsonNode, path: String): Unit =
      if node.isObject then
        if node.has("position") then
          val p = node.path("position")
          positions(path) = Path.of(p.path("file").asText()).getFileName.toString -> p
            .path("line")
            .asInt()
        node
          .fields()
          .asScala
          .filter(_.getKey != "position")
          .foreach(f => scan(f.getValue, s"$path.${f.getKey}"))
      else if node.isArray then
        node
          .elements()
          .asScala
          .zipWithIndex
          .foreach((n, i) => scan(n, s"$path[$i]"))
    scan(
      mapper.readTree(scalapb.json4s.JsonFormat.toJsonString(model)),
      "model"
    )
    positions.toMap
  private def native(cp: String, main: String, name: String): ir.Model =
    val first =
      recorded(s"$name.first", tools.run("java", Seq("-cp", cp, main)))
    val second =
      recorded(s"$name.second", tools.run("java", Seq("-cp", cp, main)))
    first.orFail(): Unit
    second.orFail(): Unit
    assertEquals(first.output, second.output)
    Files.writeString(scratch.resolve(s"$name.output"), first.output): Unit
    val model = ir.Model.parseFrom(
      Base64.getDecoder.decode(
        first.output.linesIterator
          .find(_.startsWith("facets:"))
          .get
          .stripPrefix("facets:")
      )
    )
    Files.writeString(
      scratch.resolve(s"$name.json"),
      scalapb.json4s.JsonFormat.toJsonString(model)
    ): Unit
    model

  private lazy val oldModels: Map[String, ir.Model] =
    val (dir, jar, cp) = compile(sourceNames, "old", false)
    val (facetDir, facetJar, _) =
      compile(Seq("Signature.scala", "Pressing.scala"), "old-facets", false)
    val (clientDir, clientJar, clientCp) =
      compile(
        Seq("Clients.scala", "MultiClients.scala"),
        "old-clients",
        false,
        Seq("--jar", facetJar.toString)
      )
    val selected =
      Seq(
        "a" -> roots.take(1),
        "b" -> roots.drop(1),
        "ab" -> roots,
        "ba" -> roots.reverse
      ) ++ contextual.drop(2).zip(contextualNames.drop(2)).map((root, name) => name -> Seq(root))
    val values = selected.flatMap { (name, chosen) =>
      Seq(
        s"same-$name" -> lift(s"$jar=$dir/", cp, s"old-same-$name", chosen),
        s"jar-$name" -> lift(
          s"$clientJar=$clientDir/,$facetJar=$facetDir/",
          clientCp,
          s"old-jar-$name",
          chosen
        )
      )
    }.toMap
    values.foreach { (name, model) =>
      assertEquals(model.machines.size, if name.endsWith("ab") || name.endsWith("ba") then 2 else 1)
      assert(model.functions.forall(_.position.nonEmpty))
      val expected = frozen(s"$name.json")
      assertEquals(comparable(model), comparable(expected), name)
      assertEquals(nodePositions(model), nodePositions(expected), name)
    }
    assertEquals(comparable(values("same-ab")), comparable(values("same-ba")))
    assertEquals(comparable(values("jar-ab")), comparable(values("jar-ba")))
    println(s"facets-old-oracles:$scratch")
    values

  private lazy val ordinaryOld: Map[String, ir.Model] =
    val (dir, jar, cp) = compile(ordinarySources, "ordinary-old", false)
    ordinaryRoots.map { root =>
      val name = root.split('.').last
      val model = lift(s"$jar=$dir/", cp, s"ordinary-old-$name", Seq(root))
      val expected = frozen(s"ordinary-$name.json")
      assertEquals(comparable(model), comparable(expected), root)
      assertEquals(nodePositions(model), nodePositions(expected), root)
      root -> model
    }.toMap

  private def warmSource: String = Files
    .readString(stored.resolve("Pressing.scala"))
    .replace(
      "def move(s: State) = enters(Phase.done)(s)",
      "def move(s: State) = enters(Phase.open)(s)"
    )

  private lazy val supplementaryOld: ir.Model =
    def refused(
        dir: Path,
        jar: Path,
        cp: String,
        name: String,
        root: String,
        message: String
    ): Unit =
      val classpath = Files.writeString(scratch.resolve(s"$name.classpath"), cp)
      val output = scratch.resolve(s"$name.json")
      val result = recorded(
        name,
        InProcess.frameworkLift(
          tools.directory,
          Seq(s"$jar=$dir/,$library=model/framework/", classpath.toString, output.toString, root)
        )
      )
      assertEquals(result.exit, 1, result.diagnostics)
      assert(result.diagnostics.contains(message), result.diagnostics)
      assert(!Files.exists(output))
    val (projectionDir, projectionJar, projectionCp) =
      compile(Seq("MissingProjection.scala"), "missing-projection-old", false)
    refused(
      projectionDir,
      projectionJar,
      projectionCp,
      "missing-projection-old",
      "fixture.facetsprojection.Missing",
      "reads no phase projection"
    )
    val placement = Files
      .readString(stored.resolve("Refusals.scala"))
      .replace("object left:", "object wrong:")
      .replace("left.rules", "wrong.rules")
    val (placementDir, placementJar, placementCp) = compile(
      Seq("Refusals.scala"),
      "placement-old",
      false,
      changes = Map("Refusals.scala" -> placement)
    )
    refused(
      placementDir,
      placementJar,
      placementCp,
      "placement-old",
      "fixture.facetsrefusals.Duplicate",
      "a facet's object is named after it"
    )
    val (dir, jar, cp) = compile(Seq("SectionOutcomes.scala"), "section-outcomes-old", false)
    val model =
      lift(s"$jar=$dir/", cp, "section-outcomes-safe-old", Seq("fixture.facetsoutcomes.Safe"))
    refused(dir, jar, cp, "section-outcomes-bad-old", "fixture.facetsoutcomes.Bad", "parseBoolean")
    assertEquals(comparable(model), comparable(frozen("section-outcomes-safe.json")))
    assertEquals(nodePositions(model), nodePositions(frozen("section-outcomes-safe.json")))
    model

  private lazy val warmOld: Map[String, ir.Model] =
    val changes = Map("Pressing.scala" -> warmSource)
    val (dir, jar, cp) = compile(sourceNames, "warm-old", false, changes = changes)
    val (facetDir, facetJar, _) =
      compile(Seq("Signature.scala", "Pressing.scala"), "warm-old-facets", false, changes = changes)
    val (clientDir, clientJar, clientCp) =
      compile(Seq("Clients.scala"), "warm-old-clients", false, Seq("--jar", facetJar.toString))
    roots
      .zip(Seq("a", "b"))
      .flatMap { (root, name) =>
        Seq(
          s"warm-same-$name" -> lift(s"$jar=$dir/", cp, s"warm-old-same-$name", Seq(root)),
          s"warm-jar-$name" -> lift(
            s"$clientJar=$clientDir/,$facetJar=$facetDir/",
            clientCp,
            s"warm-old-jar-$name",
            Seq(root)
          )
        )
      }
      .map { (name, model) =>
        val expected = frozen(s"$name.json")
        assertEquals(comparable(model), comparable(expected), name)
        assertEquals(nodePositions(model), nodePositions(expected), name)
        name -> model
      }
      .toMap

  test("independent complete old single and combined Facets Models freeze before native capture"):
    assertEquals(oldModels.size, 12)
    assertEquals(supplementaryOld.machines.size, 1)
    assertEquals(ordinaryOld.size, ordinaryRoots.size)
    refusalOld
    assertEquals(warmOld.size, 4)

  test(
    "old-admitted plain facet reaches existing collector and all ordinary forms retain their complete closure"
  ):
    val old = ordinaryOld
    val expectedRoots = Seq("Forward", "Reverse")
      .map { name =>
        val machine = frozen(s"ordinary-$name.json").machines.head
        val functions = machine.steps.map(step => mapper.writeValueAsString(step.function))
        s"${mapper.writeValueAsString(machine.name)} -> ${functions.mkString("Vector(", ", ", ")")}"
      }
      .mkString("Map(", ", ", ")(machine.headerIdentity.get._2)")
    val nativeSource = Files.readString(stored.resolve("ContributorNative.scala"))
    val inferredRoots =
      "machine.table.map((decl, _) => machine.headerIdentity.get._2 + \".rules.\" + decl.name)"
    assert(nativeSource.contains(inferredRoots))
    val (dir, jar, cp) = compile(
      ordinarySources ++ Seq("OrdinaryNative.scala", "ContributorNative.scala"),
      "ordinary-native",
      true,
      changes = Map("ContributorNative.scala" -> nativeSource.replace(inferredRoots, expectedRoots))
    )
    ordinaryRoots.foreach { root =>
      val name = root.split('.').last
      val main = if root.startsWith("fixture.facetscontributors.") then
        s"framework.facetscontributorsnative.Native$name"
      else s"framework.facetsordinarynative.Native$name"
      val value = native(cp, main, s"ordinary-native-$name")
      assertEquals(comparable(value), comparable(projected(old(root))))
      assertEquals(nodePositions(value), nodePositions(projected(old(root))))
      val replay = lift(s"$jar=$dir/", cp, s"ordinary-replay-$name", Seq(root))
      assertEquals(comparable(replay), comparable(old(root)))
      assertEquals(nodePositions(replay), nodePositions(old(root)))
    }

  test("isolated contextual Facets match single-root old Models in two JVMs and saved trees"):
    val old = oldModels
    val (dir, jar, cp) = compile(sourceNames :+ "Native.scala", "native", true)
    contextual.zip(contextualNames).foreach { (root, name) =>
      val value = native(cp, s"framework.facetsnative.Native${name.toUpperCase}", s"native-$name")
      assertEquals(comparable(value), comparable(projected(old(s"same-$name"))))
      assertEquals(nodePositions(value), nodePositions(projected(old(s"same-$name"))))
      val replay = lift(s"$jar=$dir/", cp, s"replay-$name", Seq(root))
      assertEquals(comparable(replay), comparable(old(s"same-$name")))
      assertEquals(nodePositions(replay), nodePositions(old(s"same-$name")))
    }
    Seq("ab" -> roots, "ba" -> roots.reverse).foreach { (name, selected) =>
      val replay = lift(s"$jar=$dir/", cp, s"replay-$name", selected)
      assertEquals(comparable(replay), comparable(old(s"same-$name")))
      assertEquals(nodePositions(replay), nodePositions(old(s"same-$name")))
    }
    val (facetDir, facetJar, _) =
      compile(Seq("Signature.scala", "Pressing.scala"), "facets", true)
    val (clientDir, clientJar, clientCp) =
      compile(
        Seq("Clients.scala", "MultiClients.scala", "Native.scala"),
        "clients",
        true,
        Seq("--jar", facetJar.toString)
      )
    contextual.zip(contextualNames).foreach { (root, name) =>
      val value =
        native(clientCp, s"framework.facetsnative.Native${name.toUpperCase}", s"jar-native-$name")
      assertEquals(comparable(value), comparable(projected(old(s"jar-$name"))))
      assertEquals(nodePositions(value), nodePositions(projected(old(s"jar-$name"))))
      val replay = lift(
        s"$clientJar=$clientDir/,$facetJar=$facetDir/",
        clientCp,
        s"jar-replay-$name",
        Seq(root)
      )
      assertEquals(comparable(replay), comparable(old(s"jar-$name")))
      assertEquals(nodePositions(replay), nodePositions(old(s"jar-$name")))
    }
    Seq("ab" -> roots, "ba" -> roots.reverse).foreach { (name, selected) =>
      val replay = lift(
        s"$clientJar=$clientDir/,$facetJar=$facetDir/",
        clientCp,
        s"jar-replay-$name",
        selected
      )
      assertEquals(comparable(replay), comparable(old(s"jar-$name")))
      assertEquals(nodePositions(replay), nodePositions(old(s"jar-$name")))
    }
    println(s"facets-native-artifacts:$scratch")

  test("an unused refused facet section does not force imported metadata"):
    val source = Files
      .readString(stored.resolve("SectionOutcomes.scala"))
      .replace("where(states.ready)", "always")
    val main = """package framework.facetssectionnative
import java.util.Base64
import framework.dsl.RuleBindings
import io.temporal.server.api.umpire.v1 as ir
object Native:
  def main(args: Array[String]): Unit =
    val machine = fixture.facetsoutcomes.Safe
    val value = RuleBindings.collect(machine)
    val model = ir.Model(
      machines = Seq(ir.Machine(name = machine.headerIdentity.get._2, steps = value.steps)),
      functions = value.functions.sortBy(_.name),
      types = value.types.sortBy(_.name)
    )
    println("facets:" + Base64.getEncoder.encodeToString(model.toByteArray))
    println(s"facets-controls:args=${args.length}")
"""
    val (dir, jar, cp) = compile(
      Seq("SectionOutcomes.scala", "SectionNative.scala"),
      "unused-section-native",
      true,
      changes = Map("SectionOutcomes.scala" -> source, "SectionNative.scala" -> main)
    )
    val value = native(cp, "framework.facetssectionnative.Native", "unused-section-native")
    val old = frozen("section-outcomes-safe.json")
    assertEquals(comparable(value), comparable(projected(old)))
    assertEquals(nodePositions(value), nodePositions(projected(old)))
    val replay =
      lift(s"$jar=$dir/", cp, "unused-section-replay", Seq("fixture.facetsoutcomes.Safe"))
    assertEquals(comparable(replay), comparable(old))
    assertEquals(nodePositions(replay), nodePositions(old))

  test("a reached facet section refuses at its definition during Rule template capture"):
    val dir = Files.createDirectories(scratch.resolve("reached-section"))
    Files.copy(stored.resolve("SectionOutcomes.scala"), dir.resolve("SectionOutcomes.scala")): Unit
    val result = recorded(
      "reached-section.compile",
      tools.scalaCli(Seq("compile", dir.toString) ++ settings(true), Output.KeptApart)
    )
    assertEquals(result.exit, 1, result.diagnostics)
    assert(result.diagnostics.contains("SectionOutcomes.scala:21:"), result.diagnostics)
    assert(result.diagnostics.contains("Reached.reached$.states$.ready"), result.diagnostics)
    assert(result.diagnostics.contains("parseBoolean"), result.diagnostics)

  test("unused and folded empty facet effects refuse before any concrete assembly exists"):
    val source = Files.readString(stored.resolve("Refusals.scala"))
    val start = source.indexOf("trait EmptyEffects")
    val end = source.indexOf("object EmptyEffect")
    val first = source.indexOf("trait Left")
    val library = (source.take(first) + source
      .slice(first, start)
      .map(c => if c == '\n' then c else ' ') + source.slice(start, end))
      .replace(
        "    object rules extends Rules:\n      on(hand.go)(always ~> ((s: Boolean) => stay[Boolean, Boolean, Nothing](s)))",
        "    object rules extends Rules\n"
      )
    Seq(
      "unused" -> library,
      "folded" -> library.replace(
        "if s then Nil else Nil",
        "if false then Nil else stay[Boolean, Boolean, Nothing](s)"
      )
    ).foreach { (name, text) =>
      val dir = Files.createDirectories(scratch.resolve(s"empty-facet-$name"))
      Files.writeString(dir.resolve("Refusals.scala"), text): Unit
      val result = recorded(
        s"empty-facet-$name",
        tools.scalaCli(Seq("compile", dir.toString) ++ settings(true), Output.KeptApart)
      )
      assertEquals(result.exit, 1, result.diagnostics)
      assert(result.diagnostics.contains("gives no step"), result.diagnostics)
      val line = text.linesIterator.toVector.indexWhere(_.contains("def unused")) + 1
      assert(result.diagnostics.contains(s"Refusals.scala:$line:"), result.diagnostics)
    }

  test("facet helper edits update definition-owned captures on one warm compiler daemon"):
    val dir = Files.createDirectories(scratch.resolve("warm-facets"))
    val daemon = Files.createTempDirectory(Path.of("/tmp"), "facets-warm.")
    Files.setPosixFilePermissions(daemon, PosixFilePermissions.fromString("rwx------")): Unit
    (sourceNames :+ "Native.scala").foreach(name =>
      Files.copy(stored.resolve(name), dir.resolve(name)): Unit
    )
    val warm = settings(true).filterNot(_ == "--server=false") ++ Seq(
      "--bloop-daemon-dir",
      daemon.toString,
      "--verbose",
      "--scalac-option",
      "-Ycheck:typer"
    )
    val observed = scala.collection.mutable.Map.empty[(Int, String), ir.Model]
    try
      Seq(false, true, false).zipWithIndex.foreach { (changed, index) =>
        val source =
          if changed then warmSource else Files.readString(stored.resolve("Pressing.scala"))
        Files.writeString(dir.resolve("Pressing.scala"), source): Unit
        Files.writeString(scratch.resolve(s"warm-source-$index.scala"), source): Unit
        val built = recorded(
          s"warm-facet-$index",
          tools
            .scalaCli(Seq("compile", dir.toString, "--print-class-path") ++ warm, Output.KeptApart)
        )
        assertEquals(built.exit, 0, built.diagnostics)
        built.orFail(): Unit
        val cp = built.output.linesIterator.filter(_.nonEmpty).toVector.last
        roots.zip(Seq("a", "b")).foreach { (_, name) =>
          val value = native(
            cp,
            s"framework.facetsnative.Native${name.toUpperCase}",
            s"warm-facet-$index-$name"
          )
          val old = frozen(if changed then s"warm-same-$name.json" else s"same-$name.json")
          assertEquals(comparable(value), comparable(projected(old)))
          assertEquals(nodePositions(value), nodePositions(projected(old)))
          if index == 1 then
            assertNotEquals(comparable(value), comparable(observed(0 -> name)), name)
          if index == 2 then
            assertEquals(comparable(value), comparable(observed(0 -> name)), name)
            assertEquals(nodePositions(value), nodePositions(observed(0 -> name)), name)
          observed(index -> name) = value
        }
        if index > 0 then
          assert(built.diagnostics.contains("Pressing.scala after phase typer"), built.diagnostics)
      }
    finally
      recorded(
        "warm-facet-exit",
        tools
          .run("scala-cli", Seq("--power", "bloop", "exit", "--bloop-daemon-dir", daemon.toString))
      ).orFail(): Unit
