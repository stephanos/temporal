package framework.`export`

import java.nio.file.{Files, Path}
import java.nio.file.attribute.PosixFilePermissions
import java.util.Base64
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import com.fasterxml.jackson.databind.{JsonNode, ObjectMapper}
import framework.check.{Output, Ran, Tools}
import io.temporal.server.api.umpire.v1 as ir

class ConstructedRulesSuite extends munit.FunSuite:
  override val munitTimeout: Duration = 20.minutes
  private val tools = Tools.here
  private val mapper = new ObjectMapper()
  private lazy val scratch =
    Files.createTempDirectory(
      tools.directory.resolve(".flow/tmp"),
      "constructed-rules."
    )
  private val stored =
    tools.directory.resolve("model/framework/export/testdata/constructedRules")
  private lazy val library = ModelBuild.frameworkLibrary:
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
  private lazy val plugin = ModelBuild.reifyPlugin:
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
      extra: Seq[String] = Nil
  ): (Path, Path, String) =
    val dir = Files.createDirectories(scratch.resolve(name))
    names.foreach(n => Files.copy(stored.resolve(n), dir.resolve(n)): Unit)
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
    Files.writeString(
      scratch.resolve(s"$name.classpath"),
      built.output.trim
    ): Unit
    (dir, jar, built.output.trim)
  private val sourceNames =
    Seq(
      "Shared.scala",
      "Targets.scala",
      "Arities.scala",
      "Domains.scala",
      "Sections.scala"
    )
  private val roots = Seq(
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
  ).map("fixture.qualifiedrules." + _) ++
    Seq(
      "fixture.rulesshapes.Arities",
      "fixture.rulesshapes.PhaseRules",
      "fixture.rulesdomains.Domains",
      "fixture.rulessections.Sections",
      "fixture.rulessections.Rejecting"
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
  private def frozen(name: String): ir.Model =
    scalapb.json4s.JsonFormat
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
          .find(_.startsWith("rules:"))
          .get
          .stripPrefix("rules:")
      )
    )
    Files.writeString(
      scratch.resolve(s"$name.json"),
      scalapb.json4s.JsonFormat.toJsonString(model)
    ): Unit
    model

  test(
    "complete strict plugin-free final Rules fixtures freeze the independent old oracle"
  ):
    val (dir, jar, cp) = compile(sourceNames, "old", false)
    val value = lift(s"$jar=$dir/", cp, "old-complete", roots)
    assertEquals(value.machines.size, roots.size)
    assert(value.functions.size > 50 && value.types.size >= 5)
    assert(value.functions.forall(_.position.nonEmpty))
    assertEquals(comparable(value), comparable(frozen("original.json")))
    val (helperDir, helperJar, _) =
      compile(Seq("Helpers.scala"), "helpers-old", false)
    val (clientDir, clientJar, clientCp) =
      compile(
        Seq("Client.scala"),
        "client-old",
        false,
        Seq("--jar", helperJar.toString)
      )
    val helpers = lift(
      s"$clientJar=$clientDir/,$helperJar=$helperDir/",
      clientCp,
      "helpers-old-complete",
      Seq("fixture.rulesclient.Client", "fixture.rulesclient.Contracts")
    )
    assert(
      helpers.functions.exists(
        _.name == "fixture.ruleslibrary.Library$.states$.ready"
      )
    )
    assert(
      helpers.functions.exists(
        _.name == "fixture.ruleslibrary.Library$.effects$.keep"
      )
    )
    assert(!helpers.functions.exists(_.name.endsWith(".unused")))
    assertEquals(comparable(helpers), comparable(frozen("helper-old.json")))
    println(s"rules-oracle:$scratch")

  test(
    "complete isolated ordinary Rules match old ordered bindings and nodes in two JVMs and saved trees"
  ):
    val (dir, jar, cp) = compile(sourceNames :+ "Native.scala", "native", true)
    val value = native(cp, "framework.rulesnative.Native", "native")
    val old = projected(frozen("original.json"))
    assertEquals(comparable(value), comparable(old))
    assertEquals(nodePositions(value), nodePositions(old))
    val replay = lift(s"$jar=$dir/", cp, "replay", roots)
    assertEquals(comparable(replay), comparable(frozen("original.json")))
    assertEquals(nodePositions(replay), nodePositions(frozen("original.json")))
    assertEquals(comparable(projected(replay)), comparable(value))
    assertEquals(nodePositions(projected(replay)), nodePositions(value))
    println(s"rules-native-artifacts:$scratch")

  test(
    "definition-owned section values and strict helper contracts cross compilation units and jars"
  ):
    val (helperDir, helperJar, _) =
      compile(Seq("Helpers.scala"), "helpers-plugin", true)
    val (dir, jar, cp) = compile(
      Seq("Client.scala", "HelperNative.scala"),
      "client-plugin",
      true,
      Seq("--jar", helperJar.toString)
    )
    val value =
      native(cp, "framework.ruleshelpernative.Native", "helper-native")
    val old = projected(frozen("helper-old.json"))
    assertEquals(comparable(value), comparable(old))
    assertEquals(nodePositions(value), nodePositions(old))
    val replay = lift(
      s"$jar=$dir/,$helperJar=$helperDir/",
      cp,
      "helpers-replay",
      Seq("fixture.rulesclient.Client", "fixture.rulesclient.Contracts")
    )
    assertEquals(comparable(replay), comparable(frozen("helper-old.json")))
    assertEquals(
      nodePositions(replay),
      nodePositions(frozen("helper-old.json"))
    )
    val (_, _, localCp) =
      compile(
        Seq("Helpers.scala", "Client.scala", "HelperNative.scala"),
        "cross-file-plugin",
        true
      )
    assertEquals(
      comparable(
        native(
          localCp,
          "framework.ruleshelpernative.Native",
          "cross-file-native"
        )
      ),
      comparable(value)
    )

  test(
    "partial effects retain exact old call groups through foreign lifted method signatures"
  ):
    val (oldLibraryDir, oldLibraryJar, _) =
      compile(Seq("PartialLibrary.scala"), "partial-old-library", false)
    val (oldBridgeDir, oldBridgeJar, _) = compile(
      Seq("PartialBridge.scala"),
      "partial-old-bridge",
      false,
      Seq("--jar", oldLibraryJar.toString)
    )
    val selected =
      Seq(
        "Client",
        "LocalForwarder",
        "StoredForwarder",
        "Leading",
        "Middle",
        "Repeated",
        "Trailing",
        "Polymorphic",
        "Multiple",
        "Six"
      ).map("fixture.rulespartial." + _)
    val (oldDir, oldJar, oldCp) = compile(
      Seq("PartialClient.scala"),
      "partial-old-client",
      false,
      Seq("--jar", oldLibraryJar.toString, "--jar", oldBridgeJar.toString)
    )
    val old = lift(
      s"$oldJar=$oldDir/,$oldLibraryJar=$oldLibraryDir/,$oldBridgeJar=$oldBridgeDir/",
      oldCp,
      "partial-old",
      selected
    )
    assertEquals(old.machines.size, 10)
    assertEquals(
      old.functions.find(_.name.endsWith("effects$.prefixed")).get.params.size,
      2
    )
    val (_, helperJar, _) =
      compile(Seq("PartialLibrary.scala"), "partial-library", true)
    val (bridgeDir, bridgeJar, _) = compile(
      Seq("PartialBridge.scala"),
      "partial-bridge",
      true,
      Seq("--jar", helperJar.toString)
    )
    val (dir, jar, cp) = compile(
      Seq("PartialClient.scala", "PartialNative.scala"),
      "partial-client",
      true,
      Seq("--jar", helperJar.toString, "--jar", bridgeJar.toString)
    )
    val value =
      native(cp, "framework.rulespartialnative.Native", "partial-native")
    assertEquals(comparable(value), comparable(projected(old)))
    assertEquals(nodePositions(value), nodePositions(projected(old)))
    val replay = lift(
      s"$jar=$dir/,$helperJar=${scratch.resolve("partial-library")}/,$bridgeJar=$bridgeDir/",
      cp,
      "partial-replay",
      selected
    )
    assertEquals(comparable(replay), comparable(old))

  test(
    "definition-owned call-group proofs reject corrupt reached metadata without forcing unused imports"
  ):
    val (_, helperJar, _) = compile(Seq("PartialLibrary.scala"), "proof-library", true)
    val poison = Files.createDirectories(scratch.resolve("call-group-poison"))
    Files.writeString(
      poison.resolve("Poison.scala"),
      """package rulescallgrouptest
import java.util.Base64
import dotty.tools.dotc.ast.tpd
import dotty.tools.dotc.ast.tpd.*
import dotty.tools.dotc.core.Annotations.Annotation
import dotty.tools.dotc.core.Constants.Constant
import dotty.tools.dotc.core.Contexts.Context
import dotty.tools.dotc.core.Symbols.*
import dotty.tools.dotc.plugins.{PluginPhase, StandardPlugin}
import io.temporal.server.api.umpire.v1 as ir
final class Poison extends StandardPlugin:
  val name = "rules-call-group-poison"
  override val description = "adversarial definition-owned call-group test metadata"
  override def initialize(options: List[String])(using Context): List[PluginPhase] =
    val mode = options.head
    List(new PluginPhase:
      val phaseName = "rulesCallGroupPoison"
      override val runsAfter = Set("reifyStore")
      override val runsBefore = Set("pickler")
      override def transformDefDef(tree: tpd.DefDef)(using Context): tpd.Tree =
        if tree.symbol.fullName.toString == "fixture.rulespartialbridge.Bridge$.effects$.forwarded" then
          val marker = requiredClass("framework.dsl.ruleCallGroups")
          def strings(t: tpd.Tree): List[String] = t match
            case Literal(c) if c.value.isInstanceOf[String] => List(c.stringValue)
            case Typed(t, _) => strings(t)
            case SeqLiteral(items, _) => items.flatMap(strings)
            case Apply(_, args) => args.flatMap(strings)
            case _ => Nil
          val data = tree.symbol.getAnnotation(marker).get.arguments.flatMap(strings)
          require(data.size == 8 && data.head == "rule-call-groups-v2")
          val altered = mode match
            case "clean" => data
            case "version" => data.updated(0, "rule-call-groups-v0")
            case "root" => data.updated(1, "stale.root")
            case "model" | "conflict" => data.updated(2, "stale-model-fingerprint")
            case "owner" => data.updated(4, "stale.owner")
            case "call" =>
              val call = ir.Call.parseFrom(Base64.getDecoder.decode(data(5)))
              data.updated(5, Base64.getEncoder.encodeToString(call.copy(args = Nil).toByteArray))
            case "callee" =>
              val function = ir.Function.parseFrom(Base64.getDecoder.decode(data(6)))
              data.updated(6, Base64.getEncoder.encodeToString(function.copy(name = "stale.callee").toByteArray))
            case "groups" => data.updated(7, "2,1")
            case "groups-consistent" => data.updated(7, "0,1,1")
            case "groups-negative" => data.updated(7, "-1,2,1")
            case "bytes" => data.updated(5, "not-base64!")
            case _ => throw IllegalArgumentException(mode)
          if mode != "conflict" then tree.symbol.removeAnnotation(marker)
          tree.symbol.addAnnotation(Annotation(marker,
            List(SeqLiteral(altered.map(s => Literal(Constant(s))), TypeTree(defn.StringType))), tree.span))
        tree)
"""
    ): Unit
    val resources = Files.createDirectories(poison.resolve("resources"))
    Files.writeString(
      resources.resolve("plugin.properties"),
      "pluginClass=rulescallgrouptest.Poison\n"
    ): Unit
    val poisonJar = scratch.resolve("call-group-poison.jar")
    recorded(
      "call-group-poison",
      tools.scalaCli(
        Seq(
          "--power",
          "package",
          poison.toString,
          "--assembly",
          "--preamble=false",
          "--main-class-in-manifest=false",
          "--provided",
          "org.scala-lang:scala3-compiler_3",
          "--dependency",
          "org.scala-lang::scala3-compiler:3.9.0",
          "--resource-dir",
          resources.toString,
          "-o",
          poisonJar.toString
        ) ++ settings(false)
      )
    ).orFail(): Unit
    val source = Files.readString(stored.resolve("PartialClient.scala"))
    val preamble = source.take(source.indexOf("object Client"))
    val reachedSource = preamble + source.drop(source.indexOf("object StoredForwarder"))
    val unusedSource = source.take(source.indexOf("object LocalForwarder"))
    val nativeSource = """package framework.rulesproofnative
import framework.dsl.RuleBindings
object Native:
  def main(args: Array[String]): Unit =
    val value = RuleBindings.collect(fixture.rulespartial.StoredForwarder)
    assert(value.steps.size == 1)
    println(s"call-proof-clean:${args.length}")
"""
    for mode <- Seq(
        "clean",
        "version",
        "root",
        "model",
        "owner",
        "call",
        "callee",
        "groups",
        "groups-consistent",
        "groups-negative",
        "bytes",
        "conflict"
      )
    do
      val (definition, jar, _) = compile(
        Seq("PartialBridge.scala"),
        s"proof-$mode",
        true,
        Seq(
          "--jar",
          helperJar.toString,
          "--scalac-option",
          s"-Xplugin:$poisonJar",
          "--scalac-option",
          s"-P:rules-call-group-poison:$mode"
        )
      )
      val extra = Seq("--jar", helperJar.toString, "--jar", jar.toString)
      if mode == "version" || mode == "conflict" then
        val unused = Files.createDirectories(scratch.resolve(s"proof-unused-$mode"))
        Files.writeString(unused.resolve("Unused.scala"), unusedSource): Unit
        recorded(
          s"proof-unused-$mode",
          tools.scalaCli(
            Seq("compile", unused.toString) ++
              settings(true) ++ extra
          )
        ).orFail(): Unit
      val reached = Files.createDirectories(scratch.resolve(s"proof-reached-$mode"))
      Files.writeString(reached.resolve("Client.scala"), reachedSource): Unit
      Files.writeString(reached.resolve("Native.scala"), nativeSource): Unit
      val result = recorded(
        s"proof-reached-$mode",
        tools.scalaCli(
          Seq("compile", reached.toString, "--print-class-path") ++ settings(true) ++ extra,
          Output.KeptApart
        )
      )
      if mode == "clean" then
        result.orFail(): Unit
        recorded(
          "proof-clean-native",
          tools.run("java", Seq("-cp", result.output.trim, "framework.rulesproofnative.Native"))
        ).orFail(): Unit
      else
        assertEquals(result.exit, 1, result.diagnostics)
        val reason = mode match
          case "conflict"                   => "conflicting"
          case "version" | "root" | "model" => "malformed or stale"
          case "bytes"                      => "malformed"
          case _                            => "stale"
        assert(
          result.diagnostics.contains(s"has $reason compiler-owned curried-call metadata"),
          result.diagnostics
        )
        assert(
          result.diagnostics.contains(s"${definition.resolve("PartialBridge.scala")}:10:"),
          result.diagnostics
        )

  test(
    "independent old reader freezes direct-result contracts and parameter exhaustion"
  ):
    val (dir, jar, cp) = compile(Seq("Refusals.scala"), "refusals-old", false)
    val classpath =
      Files.writeString(scratch.resolve("refusals-old.classpath"), cp)
    for name <- Seq(
        "UnusedNil",
        "UsedDisabled",
        "UnusedList",
        "MatchFirst",
        "MatchLast",
        "RequireGuard",
        "EnsureGuard",
        "RequireEffect",
        "EnsureEffect"
      )
    do
      val result = recorded(
        s"old-$name",
        InProcess.frameworkLift(
          tools.directory,
          Seq(
            s"$jar=$dir/,$library=model/framework/",
            classpath.toString,
            scratch.resolve(s"old-$name.json").toString,
            s"fixture.rulesrefusals.$name"
          )
        )
      )
      assertEquals(result.exit, 1, result.diagnostics)
      assert(result.diagnostics.contains("Refusals.scala:"), result.diagnostics)
    val helper =
      lift(
        s"$jar=$dir/",
        cp,
        "helper-result-old",
        Seq("fixture.rulesrefusals.HelperResult")
      )
    assertEquals(helper.machines.size, 1)
    val exhausted = scala.util.Try(
      InProcess.frameworkLift(
        tools.directory,
        Seq(
          s"$jar=$dir/,$library=model/framework/",
          classpath.toString,
          scratch.resolve("exhausted-old.json").toString,
          "fixture.rulesrefusals.Exhausted"
        )
      )
    )
    Files.writeString(
      scratch.resolve("exhausted-old.observation"),
      exhausted.toString
    ): Unit
    assert(
      exhausted.isFailure || exhausted.toOption.exists(_.exit == 1),
      exhausted.toString
    )

  test(
    "old section values admit unused failures and refuse reached failures at their definitions"
  ):
    val (helperDir, helperJar, helperCp) =
      compile(Seq("Invalids.scala"), "invalids-old", false)
    val unused = lift(
      s"$helperJar=$helperDir/",
      helperCp,
      "unused-invalids-old",
      Seq("fixture.rulesinvalids.Invalids")
    )
    assertEquals(unused.machines.size, 1)
    for name <- Seq("InvalidGuard", "InvalidEffect") do
      val (dir, jar, cp) =
        compile(
          Seq(s"$name.scala"),
          s"$name-old",
          false,
          Seq("--jar", helperJar.toString)
        )
      val classpath =
        Files.writeString(scratch.resolve(s"$name-old.classpath"), cp)
      val result = recorded(
        s"$name-old",
        InProcess.frameworkLift(
          tools.directory,
          Seq(
            s"$jar=$dir/,$helperJar=$helperDir/,$library=model/framework/",
            classpath.toString,
            scratch.resolve(s"$name-old.json").toString,
            s"fixture.rulesinvalidclient.$name"
          )
        )
      )
      assertEquals(result.exit, 1, result.diagnostics)
      assert(result.diagnostics.contains("Invalids.scala:"), result.diagnostics)

  test(
    "section carriers remain lazy and located in same-unit cross-file and cross-jar consumers"
  ):
    val (_, helperJar, _) =
      compile(Seq("Invalids.scala"), "invalids-plugin", true)
    for name <- Seq("InvalidGuard", "InvalidEffect") do
      val expected = if name == "InvalidGuard" then "Invalids.scala:12"
      else "Invalids.scala:14"
      val root =
        if name == "InvalidGuard" then "states$.bad" else "effects$.bad"
      for mode <- Seq("same-unit", "cross-file", "cross-jar") do
        val dir = Files.createDirectories(scratch.resolve(s"$name-$mode"))
        if mode == "same-unit" then
          val helper = Files.readString(stored.resolve("Invalids.scala"))
          val client = Files.readString(stored.resolve(s"$name.scala"))
          def enclosed(source: String): String =
            val split = source.indexOf('\n')
            source.take(split) + " {" + source.drop(split) + "\n}\n"
          Files.writeString(
            dir.resolve("Invalids.scala"),
            enclosed(helper) + enclosed(client)
          ): Unit
        else
          Files.copy(
            stored.resolve(s"$name.scala"),
            dir.resolve(s"$name.scala")
          ): Unit
          if mode == "cross-file" then
            Files.copy(
              stored.resolve("Invalids.scala"),
              dir.resolve("Invalids.scala")
            ): Unit
        val result = recorded(
          s"$name-$mode",
          tools.scalaCli(
            Seq("compile", dir.toString) ++
              settings(true) ++ (if mode == "cross-jar" then Seq("--jar", helperJar.toString)
                                 else Nil)
          )
        )
        assertEquals(result.exit, 1, result.diagnostics)
        assert(result.diagnostics.contains(expected), result.diagnostics)
        assert(result.diagnostics.contains(root), result.diagnostics)
    val dir = Files.createDirectories(scratch.resolve("same-unit-success"))
    val text = Seq("Helpers.scala", "Client.scala", "HelperNative.scala").map { name =>
      val source = Files.readString(stored.resolve(name))
      val split = source.indexOf('\n')
      source.take(split) + " {" + source.drop(split) + "\n}\n"
    }.mkString
    Files.writeString(dir.resolve("Local.scala"), text): Unit
    val built = recorded(
      "same-unit-success",
      tools.scalaCli(
        Seq("compile", dir.toString, "--print-class-path") ++ settings(true),
        Output.KeptApart
      )
    )
    assertEquals(built.exit, 0, built.diagnostics)
    assertEquals(
      comparable(
        native(
          built.output.trim,
          "framework.ruleshelpernative.Native",
          "same-unit-native"
        )
      ),
      comparable(projected(frozen("helper-old.json")))
    )

  test(
    "a section-only edit invalidates dependent carriers on one warm compiler daemon"
  ):
    val dir = Files.createDirectories(scratch.resolve("warm-sections"))
    val daemon = Files.createTempDirectory(Path.of("/tmp"), "rules-warm.")
    Files.setPosixFilePermissions(
      daemon,
      PosixFilePermissions.fromString("rwx------")
    ): Unit
    Seq("Helpers.scala", "Client.scala", "HelperNative.scala").foreach { name =>
      Files.copy(stored.resolve(name), dir.resolve(name)): Unit
    }
    val warm = settings(true).filterNot(_ == "--server=false") ++ Seq(
      "--bloop-daemon-dir",
      daemon.toString,
      "--verbose",
      "--scalac-option",
      "-Ycheck:typer"
    )
    def change(body: String): Unit = Files.writeString(
      dir.resolve("Helpers.scala"),
      Files
        .readString(stored.resolve("Helpers.scala"))
        .replace("val ready = is(true)", s"val ready = is($body)")
    ): Unit
    def compiled(label: String): Ran = recorded(
      label,
      tools.scalaCli(
        Seq("compile", dir.toString, "--print-class-path") ++ warm,
        Output.KeptApart
      )
    )
    try
      for (body, index) <- Seq("true", "false").zipWithIndex do
        change(body)
        val built = compiled(s"warm-section-$index")
        assertEquals(built.exit, 0, built.diagnostics)
        val model = native(
          built.output.trim.linesIterator.toVector.last,
          "framework.ruleshelpernative.Native",
          s"warm-section-native-$index"
        )
        val ready = model.functions.find(_.name.endsWith(".states$.ready")).get
        assertEquals(ready.getBody.getLiteral.getBool, body == "true")
        if index == 1 then
          Seq("Helpers.scala", "Client.scala").foreach { name =>
            assert(
              built.diagnostics.contains(s"$name after phase typer"),
              built.diagnostics
            )
          }
      change("throw IllegalArgumentException(\"changed\")")
      val refused = compiled("warm-section-refused")
      assertEquals(refused.exit, 1, refused.diagnostics)
      assert(
        refused.diagnostics.contains("Helpers.scala:12"),
        refused.diagnostics
      )
      change("true")
      val recovered = compiled("warm-section-recovered")
      assertEquals(recovered.exit, 0, recovered.diagnostics)
      val model = native(
        recovered.output.trim.linesIterator.toVector.last,
        "framework.ruleshelpernative.Native",
        "warm-section-native-recovered"
      )
      assert(
        model.functions
          .find(_.name.endsWith(".states$.ready"))
          .get
          .getBody
          .getLiteral
          .getBool
      )
    finally
      recorded(
        "warm-section-exit",
        tools.run(
          "scala-cli",
          Seq("--power", "bloop", "exit", "--bloop-daemon-dir", daemon.toString)
        )
      ).orFail(): Unit

  test(
    "definition symbols expose versioned success and lazy located-refusal carriers across jars"
  ):
    val (_, helperJar, _) =
      compile(Seq("Helpers.scala"), "carrier-helpers", true)
    val dir = Files.createDirectories(scratch.resolve("carrier-inspector"))
    Files.writeString(
      dir.resolve("Inspect.scala"),
      """package framework.rulescarrier
import scala.quoted.*
import fixture.ruleslibrary.Library
object Inspect:
  inline def verify(): Unit = ${verifyImpl}
  def verifyImpl(using Quotes): Expr[Unit] =
    import quotes.reflect.*
    def strings(t: Tree): List[String] = t match
      case Literal(StringConstant(s)) => List(s)
      case Apply(_, args) => args.flatMap(strings)
      case Typed(value, _) => strings(value)
      case Repeated(values, _) => values.flatMap(strings)
      case _ => Nil
    def referenced(t: Term): Symbol = t match
      case Inlined(_, _, value) => referenced(value)
      case Typed(value, _) => referenced(value)
      case Block(Nil, value) => referenced(value)
      case value => value.symbol
    val refs: List[(Expr[Any], String, String)] = List(
      ('{Library.states.ready}, "is", "ok"),
      ('{Library.states.unused}, "is", "refused"),
      ('{Library.effects.keep}, "effect", "ok"))
    refs.foreach { (ref, form, status) =>
      val symbol = referenced(ref.asTerm)
      val annotation = symbol.annotations.find(_.tpe.typeSymbol.fullName == "framework.dsl.sectionIr")
        .getOrElse(report.errorAndAbort("missing section carrier: " + symbol.fullName))
      val data = strings(annotation)
      if data.take(4) != List("section-v1", symbol.fullName, form, status) then
        report.errorAndAbort("stale section symbol carrier: " + data)
      if status == "refused" && (!data(4).endsWith("Helpers.scala") || data(5) != "13") then
        report.errorAndAbort("missing definition-owned refusal position: " + data)
    }
    '{ () }
"""
    ): Unit
    val jar = scratch.resolve("carrier-inspector.jar")
    recorded(
      "carrier-inspector",
      tools.scalaCli(
        Seq(
          "compile",
          dir.toString,
          "--compilation-output",
          jar.toString
        ) ++ settings(
          false
        ) ++ Seq("--jar", helperJar.toString)
      )
    ).orFail(): Unit
    val client =
      Files.createDirectories(scratch.resolve("carrier-inspector-client"))
    Files.writeString(
      client.resolve("Check.scala"),
      """package framework.rulescarrier
object Check:
  def main(args: Array[String]): Unit = Inspect.verify()
"""
    ): Unit
    recorded(
      "carrier-inspector-client",
      tools.scalaCli(
        Seq("compile", client.toString) ++ settings(false) ++
          Seq("--jar", jar.toString, "--jar", helperJar.toString)
      )
    ).orFail(): Unit

  test(
    "named domain definitions retain exact positions across jars and reject stale position carriers"
  ):
    val lines =
      Files.readString(stored.resolve("Domains.scala")).linesIterator.toVector
    val boundary = lines.indexWhere(_.startsWith("object Domains extends"))
    assert(boundary > 0)
    val types = Files.createDirectories(scratch.resolve("domain-types"))
    val client = Files.createDirectories(scratch.resolve("domain-client"))
    val typeSource = types.resolve("DomainTypes.scala")
    Files.writeString(
      typeSource,
      lines
        .take(boundary)
        .map { line =>
          if line.startsWith("given Ok") then "" else line
        }
        .mkString("\n") + "\n"
    ): Unit
    Files.writeString(
      client.resolve("Domains.scala"),
      lines.zipWithIndex
        .map { (line, index) =>
          if index >= boundary || line.startsWith("package ") || line
              .startsWith("import ") ||
            line.startsWith("given Ok")
          then line
          else ""
        }
        .mkString("\n") + "\n"
    ): Unit
    Files.writeString(
      client.resolve("Native.scala"),
      """package framework.rulesdomainclient
import framework.dsl.RuleBindings
import io.temporal.server.api.umpire.v1 as ir
object Native:
  def main(args: Array[String]): Unit =
    val c = RuleBindings.collect(fixture.rulesdomains.Domains)
    val model = ir.Model(machines = Seq(ir.Machine(name = "domains", steps = c.steps)),
      functions = c.functions, types = c.types)
    println("rules:" + java.util.Base64.getEncoder.encodeToString(model.toByteArray))
    println(s"domain-controls:args=${args.length}")
"""
    ): Unit
    val jar = scratch.resolve("domain-types.jar")
    recorded(
      "domain-types",
      tools.scalaCli(
        Seq(
          "compile",
          types.toString,
          "--compilation-output",
          jar.toString
        ) ++ settings(true)
      )
    ).orFail(): Unit
    val built = recorded(
      "domain-client",
      tools.scalaCli(
        Seq("compile", client.toString, "--print-class-path") ++ settings(
          true
        ) ++ Seq(
          "--jar",
          jar.toString
        ),
        Output.KeptApart
      )
    )
    built.orFail(): Unit
    val value = native(
      built.output.trim,
      "framework.rulesdomainclient.Native",
      "domain-native"
    )
    val old = frozen("original.json")
    assertEquals(
      comparable(value),
      comparable(
        projected(old.copy(machines = old.machines.filter(_.name == "domains")))
      )
    )
    Seq("Mode" -> 6, "Pair" -> 9).foreach { (name, line) =>
      val t = value.types.find(_.name == s"fixture.rulesdomains.$name").get
      assertEquals(t.position, Some(ir.Position(typeSource.toString, line)))
    }
    val nestedTypes =
      Files.createDirectories(scratch.resolve("nested-domain-types"))
    val nestedClient =
      Files.createDirectories(scratch.resolve("nested-domain-client"))
    val nestedSource = nestedTypes.resolve("NestedTypes.scala")
    Files.writeString(
      nestedSource,
      Files
        .readString(typeSource)
        .replace(
          "enum Mode:\n  case plain\n  case tagged(flag: Boolean)\nfinal case class Pair(left: Boolean, right: Boolean)",
          "object Types:\n  enum Mode:\n    case plain\n    case tagged(flag: Boolean)\n  final case class Pair(left: Boolean, right: Boolean)\nimport Types.*"
        )
    ): Unit
    val nestedLines = Files
      .readString(client.resolve("Domains.scala"))
      .linesIterator
      .toVector
      .updated(3, "import fixture.rulesdomains.Types.*")
    Files.writeString(
      nestedClient.resolve("Domains.scala"),
      nestedLines.mkString("\n") + "\n"
    ): Unit
    Files.copy(
      client.resolve("Native.scala"),
      nestedClient.resolve("Native.scala")
    ): Unit
    val nestedJar = scratch.resolve("nested-domain-types.jar")
    recorded(
      "nested-domain-types",
      tools.scalaCli(
        Seq(
          "compile",
          nestedTypes.toString,
          "--compilation-output",
          nestedJar.toString
        ) ++ settings(true)
      )
    ).orFail(): Unit
    val nestedBuilt = recorded(
      "nested-domain-client",
      tools.scalaCli(
        Seq("compile", nestedClient.toString, "--print-class-path") ++ settings(
          true
        ) ++ Seq(
          "--jar",
          nestedJar.toString
        ),
        Output.KeptApart
      )
    )
    nestedBuilt.orFail(): Unit
    val nestedValue =
      native(
        nestedBuilt.output.trim,
        "framework.rulesdomainclient.Native",
        "nested-domain-native"
      )
    Seq("Mode" -> 7, "Pair" -> 10).foreach { (name, line) =>
      val t = nestedValue.types
        .find(_.name == s"fixture.rulesdomains.Types.$name")
        .get
      assertEquals(t.position, Some(ir.Position(nestedSource.toString, line)))
    }
    val poison =
      Files.createDirectories(scratch.resolve("domain-position-poison"))
    Files.writeString(
      poison.resolve("Poison.scala"),
      """package rulespositiontest
import dotty.tools.dotc.ast.tpd
import dotty.tools.dotc.ast.tpd.*
import dotty.tools.dotc.core.Annotations.Annotation
import dotty.tools.dotc.core.Constants.Constant
import dotty.tools.dotc.core.Contexts.Context
import dotty.tools.dotc.core.Symbols.*
import dotty.tools.dotc.plugins.{PluginPhase, StandardPlugin}
final class Poison extends StandardPlugin:
  val name = "rules-position-poison"
  override val description = "adversarial retained-position test metadata"
  override def initialize(options: List[String])(using Context): List[PluginPhase] =
    val data = options.head.split("~", -1).toList
    require(data.size == 4)
    List(new PluginPhase:
      val phaseName = "rulesPositionPoison"
      override val runsAfter = Set("reifyStore")
      override val runsBefore = Set("pickler")
      override def transformTypeDef(tree: tpd.TypeDef)(using Context): tpd.Tree =
        if tree.symbol.fullName.toString == "fixture.rulesdomains.Mode" then
          tree.symbol.addAnnotation(Annotation(requiredClass("framework.dsl.ruleTypePosition"),
            List(SeqLiteral(data.map(s => Literal(Constant(s))), TypeTree(defn.StringType))), tree.span))
        tree)
"""
    ): Unit
    val resources = Files.createDirectories(poison.resolve("resources"))
    Files.writeString(
      resources.resolve("plugin.properties"),
      "pluginClass=rulespositiontest.Poison\n"
    ): Unit
    val poisonJar = scratch.resolve("domain-position-poison.jar")
    recorded(
      "domain-position-poison",
      tools.scalaCli(
        Seq(
          "--power",
          "package",
          poison.toString,
          "--assembly",
          "--preamble=false",
          "--main-class-in-manifest=false",
          "--provided",
          "org.scala-lang:scala3-compiler_3",
          "--dependency",
          "org.scala-lang::scala3-compiler:3.9.0",
          "--resource-dir",
          resources.toString,
          "-o",
          poisonJar.toString
        ) ++ settings(false)
      )
    ).orFail(): Unit
    Seq(
      "rule-type-position-v0,fixture.rulesdomains.Mode,definition.scala,6",
      "rule-type-position-v1,stale.Mode,definition.scala,6",
      "rule-type-position-v1,fixture.rulesdomains.Mode,stale.scala,7",
      "rule-type-position-v1,fixture.rulesdomains.Mode,,0"
    ).zipWithIndex.foreach { (metadata, index) =>
      val forged =
        Files.createDirectories(scratch.resolve(s"domain-forged-$index"))
      Files.copy(typeSource, forged.resolve("DomainTypes.scala")): Unit
      val forgedJar = scratch.resolve(s"domain-forged-$index.jar")
      recorded(
        s"domain-forged-$index",
        tools.scalaCli(
          Seq(
            "compile",
            forged.toString,
            "--compilation-output",
            forgedJar.toString
          ) ++ settings(
            true
          ) ++ Seq(
            "--scalac-option",
            s"-Xplugin:$poisonJar",
            "--scalac-option",
            s"-P:rules-position-poison:${metadata.replace(',', '~')}"
          )
        )
      ).orFail(): Unit
      if index == 0 then
        val unused =
          Files.createDirectories(scratch.resolve("domain-unused-metadata"))
        Files.writeString(
          unused.resolve("Unused.scala"),
          """package fixture.rulesdomains
import framework.dsl.*
given Ok[Boolean] = Ok(true)
object Unused extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object effects:
    def keep(s: Boolean) = stay[Boolean, Boolean, Nothing](s)
  object rules extends Rules:
    on(hand.bool)(always ~> effects.keep)
"""
        ): Unit
        recorded(
          "domain-unused-metadata",
          tools.scalaCli(
            Seq("compile", unused.toString) ++
              settings(true) ++ Seq("--jar", forgedJar.toString)
          )
        ).orFail(): Unit
      val reached =
        Files.createDirectories(scratch.resolve(s"domain-stale-$index"))
      Files.writeString(
        reached.resolve("Domains.scala"),
        Files.readString(client.resolve("Domains.scala"))
      ): Unit
      val result = recorded(
        s"domain-stale-$index",
        tools.scalaCli(
          Seq("compile", reached.toString) ++
            settings(true) ++ Seq("--jar", forgedJar.toString)
        )
      )
      assertEquals(result.exit, 1, result.diagnostics)
      val reason = if index == 2 then "conflicting" else "malformed or stale"
      assert(
        result.diagnostics.contains(
          s"fixture.rulesdomains.Mode has $reason type-definition position metadata"
        ),
        result.diagnostics
      )
    }

  test(
    "opaque callback vals and unmarked foreign helper defs remain outside the stored boundary"
  ):
    for opaque <- Seq(false, true) do
      val dir =
        Files.createDirectories(scratch.resolve(if opaque then "opaque-library"
        else "unmarked-library"))
      val original = Files.readString(stored.resolve("Helpers.scala"))
      val text = if opaque then
        original.replace(
          "val ready = is(true)",
          "val ready: Boolean => Boolean = (s: Boolean) => s"
        )
      else original.replace("@lifted ", "")
      Files.writeString(dir.resolve("Helpers.scala"), text): Unit
      val jar = scratch.resolve(if opaque then "opaque-library.jar"
      else "unmarked-library.jar")
      recorded(
        s"boundary-library-$opaque",
        tools.scalaCli(
          Seq(
            "compile",
            dir.toString,
            "--compilation-output",
            jar.toString
          ) ++ settings(true)
        )
      ).orFail(): Unit
      val client =
        Files.createDirectories(scratch.resolve(s"boundary-client-$opaque"))
      Files.copy(
        stored.resolve("Client.scala"),
        client.resolve("Client.scala")
      ): Unit
      val result = recorded(
        s"boundary-client-$opaque",
        tools.scalaCli(
          Seq("compile", client.toString) ++
            settings(true) ++ Seq("--jar", jar.toString)
        )
      )
      assertEquals(result.exit, 1, result.diagnostics)
      assert(
        result.diagnostics.contains(if opaque then "ready" else "@lifted"),
        result.diagnostics
      )

  test(
    "all frozen direct-result and direct-contract refusals survive strict capture independently"
  ):
    val source = Files.readString(stored.resolve("Refusals.scala"))
    val preamble = source.take(source.indexOf("object UnusedNil"))
    val names = Seq(
      "UnusedNil",
      "UsedDisabled",
      "UnusedList",
      "MatchFirst",
      "MatchLast",
      "RequireGuard",
      "EnsureGuard",
      "RequireEffect",
      "EnsureEffect",
      "Exhausted",
      "HelperResult"
    )
    names.foreach { name =>
      val start = source.indexOf(s"object $name ")
      val next = source.indexOf("\nobject ", start + 1)
      val body =
        if next < 0 then source.drop(start) else source.slice(start, next)
      val dir = Files.createDirectories(scratch.resolve(s"isolated-$name"))
      val collision = if name == "Exhausted" then
        source.slice(
          source.indexOf("object collision"),
          source.indexOf("object HelperResult")
        )
      else ""
      Files.writeString(
        dir.resolve("Refusals.scala"),
        preamble + body + "\n" + collision
      ): Unit
      if name == "Exhausted" then
        Files.writeString(
          dir.resolve("Check.scala"),
          """package framework.rulesnegative
import framework.dsl.RuleBindings
object Check:
  def main(args: Array[String]): Unit =
    try
      RuleBindings.collect(fixture.rulesrefusals.Exhausted): Unit
      throw AssertionError("exhausted names accepted")
    catch
      case error: IllegalArgumentException =>
        assert(error.getMessage.contains("state parameter"), error.getMessage)
"""
        ): Unit
      val result = recorded(
        s"isolated-$name",
        tools.scalaCli(
          Seq("compile", dir.toString, "--print-class-path") ++ settings(true),
          Output.KeptApart
        )
      )
      assertEquals(
        result.exit,
        if name == "HelperResult" || name == "Exhausted" then 0 else 1,
        result.diagnostics
      )
      if name == "Exhausted" then
        recorded(
          "isolated-exhaustion-native",
          tools.run(
            "java",
            Seq("-cp", result.output.trim, "framework.rulesnegative.Check")
          )
        ).orFail(): Unit
      else if name != "HelperResult" then
        assert(
          result.diagnostics.contains("Refusals.scala:"),
          result.diagnostics
        )
        val reason =
          if name.startsWith("Unused") || name.startsWith(
              "Match"
            ) || name == "UsedDisabled"
          then "gives no step"
          else if name.startsWith("Ensure") then "ensuring"
          else "require"
        assert(result.diagnostics.contains(reason), result.diagnostics)
    }

  test(
    "compiler refuses unused and folded-away direct empty effects at the definition"
  ):
    val dir = Files.createDirectories(scratch.resolve("empty"))
    Files.writeString(
      dir.resolve("Empty.scala"),
      """package fixture.rulesempty
import framework.dsl.*
given Ok[Boolean] = Ok(true)
object hand extends Actor:
  val go = action(this)
object Empty extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object effects:
    def unused(s: Boolean): List[Step[Boolean, Boolean, Nothing]] =
      if false then Nil else stay[Boolean, Boolean, Nothing](s)
  object rules extends Rules:
    disabled(hand.go)
"""
    ): Unit
    val (oldDir, oldJar, oldCp) =
      val jar = scratch.resolve("empty-old.jar")
      val built = recorded(
        "empty-old.compile",
        tools.scalaCli(
          Seq(
            "compile",
            dir.toString,
            "--print-class-path",
            "--compilation-output",
            jar.toString
          ) ++ settings(false),
          Output.KeptApart
        )
      )
      built.orFail(): Unit
      (dir, jar, built.output.trim)
    val classpath =
      Files.writeString(scratch.resolve("empty-old.classpath"), oldCp)
    val old = recorded(
      "empty-old",
      InProcess.frameworkLift(
        tools.directory,
        Seq(
          s"$oldJar=$oldDir/",
          classpath.toString,
          scratch.resolve("empty-old.json").toString,
          "fixture.rulesempty.Empty"
        )
      )
    )
    assertEquals(old.exit, 1, old.diagnostics)
    assert(
      old.diagnostics.contains("unused is an effect and gives no step"),
      old.diagnostics
    )
    val result =
      recorded(
        "empty-plugin",
        tools.scalaCli(Seq("compile", dir.toString) ++ settings(true))
      )
    assertEquals(result.exit, 1, result.diagnostics)
    assert(
      result.diagnostics.contains("unused is an effect and gives no step"),
      result.diagnostics
    )
    assert(result.diagnostics.contains("Empty.scala:11"), result.diagnostics)

  test(
    "historical mixed qualification fixtures and authored subset checks retain standalone refusals"
  ):
    val dir = Files.createDirectories(scratch.resolve("qualified-originals"))
    Seq("Shared.scala", "Accepted.scala", "RuleRejects.scala").foreach { name =>
      Files.copy(
        tools.directory
          .resolve("model/framework/export/testdata/qualifiedRules")
          .resolve(name),
        dir.resolve(name)
      ): Unit
    }
    val jar = scratch.resolve("qualified-originals.jar")
    val built = recorded(
      "qualified-originals.compile",
      tools.scalaCli(
        Seq(
          "compile",
          dir.toString,
          "--print-class-path",
          "--compilation-output",
          jar.toString
        ) ++ settings(false),
        Output.KeptApart
      )
    )
    assertEquals(built.exit, 0, built.diagnostics)
    val cp = Files.writeString(
      scratch.resolve("qualified-originals.classpath"),
      built.output.trim
    )
    val names =
      (for
        form <- Seq("Whole", "Class", "Two", "Three", "Four");
        scope <- Seq("Outside", "InFrom")
      yield s"Bare$form$scope") ++ Seq(
        "ImportedWildcard",
        "ImportedSelective",
        "Foreign",
        "BareDisabled",
        "BareDisabledInFrom"
      )
    val refused = recorded(
      "qualified-originals",
      InProcess.frameworkLift(
        tools.directory,
        Seq(
          s"$jar=$dir/,$library=model/framework/",
          cp.toString,
          scratch.resolve("qualified-originals.json").toString
        ) ++
          names.map("fixture.qualifiedrules." + _)
      )
    )
    assertEquals(refused.exit, 1, refused.diagnostics)
    assertEquals(
      refused.diagnostics.linesIterator
        .count(_.contains("names its action bare")),
      12
    )
    assertEquals(
      refused.diagnostics.linesIterator.count(_.contains("a from only groups")),
      2
    )
    assert(
      refused.diagnostics.contains("hand declares no tick"),
      refused.diagnostics
    )
    val subset = Files.createDirectories(scratch.resolve("authored-subset"))
    Files.writeString(
      subset.resolve("Unsupported.scala"),
      """package fixture.rulessubset
import framework.dsl.*
object hand extends Actor:
  val go = action(this)
object Unsupported extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = s
  def mutation(s: Boolean): List[Step[Boolean, Boolean, Nothing]] =
    var out = List(Step[Boolean, Boolean, Nothing](true, s))
    out = out.reverse
    out
  object rules extends Bindings(hand.go ~> mutation)
"""
    ): Unit
    val subsetJar = scratch.resolve("authored-subset.jar")
    val compiled = recorded(
      "authored-subset.compile",
      tools.scalaCli(
        Seq(
          "compile",
          subset.toString,
          "--print-class-path",
          "--compilation-output",
          subsetJar.toString
        ) ++ settings(false),
        Output.KeptApart
      )
    )
    assertEquals(compiled.exit, 0, compiled.diagnostics)
    val subsetCp =
      Files.writeString(
        scratch.resolve("authored-subset.classpath"),
        compiled.output.trim
      )
    val checked = recorded(
      "authored-subset.lint",
      tools.run(
        "java",
        Seq(
          "-cp",
          System.getProperty("java.class.path"),
          "framework.export.lintConventions",
          s"$subsetJar=$subset/,$library=model/framework/",
          subsetCp.toString
        )
      )
    )
    assertEquals(checked.exit, 1, checked.diagnostics)
    assert(
      checked.diagnostics.contains("Unsupported.scala:9:"),
      checked.diagnostics
    )
    assert(checked.diagnostics.contains("var out"), checked.diagnostics)
    assert(
      checked.diagnostics.contains("model/SEMANTICS.md#liftable-subset"),
      checked.diagnostics
    )
    val lifted = recorded(
      "authored-subset.lift",
      InProcess.frameworkLift(
        tools.directory,
        Seq(
          s"$subsetJar=$subset/,$library=model/framework/",
          subsetCp.toString,
          scratch.resolve("authored-subset.json").toString,
          "fixture.rulessubset.Unsupported"
        )
      )
    )
    assertEquals(lifted.exit, 1, lifted.diagnostics)
    assert(
      lifted.diagnostics.contains("Unsupported.scala:9:"),
      lifted.diagnostics
    )
