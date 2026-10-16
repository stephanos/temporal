package framework.`export`

import java.nio.file.{Files, Path}
import java.util.Base64
import scala.concurrent.duration.*
import framework.check.{Output, RetainedScratch, Tools}
import framework.lint.ModulePolicy
import io.temporal.server.api.umpire.v1 as ir

class ConstructedInboxesSuite extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private lazy val scratch = retained(tools.directory.resolve(".flow/tmp"), "constructed-inboxes.")
  private val stored = tools.directory.resolve("model/framework/export/testdata/constructedInboxes")
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
  private def settings = Seq(
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
    tools.directory.resolve("model/build/ir-scalapb.jar").toString,
    "--scalac-option",
    s"-Xplugin:$plugin",
    "--scalac-option",
    "-P:umpire-reify:mode=signature"
  ) ++ Seq(
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
  private def compile(dir: Path, name: String, extra: Seq[String] = Nil): (Path, String) =
    val jar = scratch.resolve(s"$name.jar")
    val built = tools.scalaCli(
      Seq(
        "compile",
        dir.toString,
        "--print-class-path",
        "--compilation-output",
        jar.toString
      ) ++ settings ++ extra,
      Output.KeptApart
    )
    assertEquals(built.exit, 0, built.diagnostics)
    jar -> built.output.trim
  private def sources(names: Seq[String], folder: String): Path =
    val dir = Files.createDirectories(scratch.resolve(folder))
    names.foreach(name => Files.copy(stored.resolve(name), dir.resolve(name)): Unit)
    dir
  private def lift(jar: Path, dir: Path, cp: String, name: String, roots: Seq[String]): ir.Model =
    val classpath = Files.writeString(scratch.resolve(s"$name.classpath"), cp)
    val output = scratch.resolve(s"$name.json")
    val result = InProcess.lift(
      tools.directory,
      Seq(s"$jar=$dir/", classpath.toString, output.toString) ++ roots,
      policy = ModulePolicy.Generic
    )
    assertEquals(result.exit, 0, result.diagnostics)
    scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(output))
  private val roots = Seq("First", "Second", "Bounded", "Opaque", "Choice", "Nested")
    .map(n => s"fixture.constructedinbox.${n}Machine")
  private def frozen(name: String): ir.Model =
    scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(stored.resolve(name)))
  private def run(cp: String, main: String): String =
    val result = tools.run("java", Seq("-cp", cp, main))
    assertEquals(result.exit, 0, result.diagnostics)
    result.output
  private def payload(output: String, label: String): ir.Model = ir.Model.parseFrom(
    Base64.getDecoder.decode(
      output.linesIterator.find(_.startsWith(label + ":")).get.stripPrefix(label + ":")
    )
  )

  test("current reader preserves the complete frozen type and channel contributions"):
    val dir = sources(Seq("Inboxes.scala"), "oracle")
    val (jar, cp) = compile(dir, "oracle")
    val old = lift(jar, dir, cp, "oracle", roots)
    assert(old.types.exists(_.name == "fixture.constructedinbox.First"))
    assertEquals(old.channels.size, 6)
    assertEquals(old.types.map(_.clearPosition), frozen("original.json").types.map(_.clearPosition))
    assertEquals(
      old.channels.map(_.clearPosition),
      frozen("original.json").channels.map(_.clearPosition)
    )
    println(s"old-inbox-oracle:$scratch")

  test("old aliases helper jars and storage refusals have actual admission controls"):
    val dir = sources(Seq("Inboxes.scala", "Forms.scala", "AliasNative.scala"), "forms")
    val (jar, cp) = compile(dir, "forms")
    val classpath = Files.writeString(scratch.resolve("forms.classpath"), cp)
    val nativeAlias = run(cp, "framework.inboxalias.AliasNative")
    assertEquals(nativeAlias, run(cp, "framework.inboxalias.AliasNative"))
    assert(nativeAlias.contains("native-refusals:passed"))
    val alias = payload(nativeAlias, "alias")
    val oldAlias = frozen("alias-old.json")
    assertEquals(alias.types, oldAlias.types.map(_.clearPosition))
    assertEquals(alias.channels, oldAlias.channels.map(_.clearPosition))
    for (root, accepted) <- Seq(
        "Alias" -> true,
        "Duplicate" -> false,
        "DuplicateCases" -> false,
        "Missing" -> false
      )
    do
      val output = scratch.resolve(s"$root.json")
      val result = InProcess.lift(
        tools.directory,
        Seq(
          s"$jar=$dir/",
          classpath.toString,
          output.toString,
          s"fixture.constructedinbox.${root}Machine"
        ),
        policy = ModulePolicy.Generic
      )
      println(s"old-inbox-$root:exit=${result.exit}\n${result.diagnostics}")
      assertEquals(result.exit, if accepted then 0 else 1, result.diagnostics)
      if !accepted then
        assert(
          result.diagnostics.contains(if root == "Missing" then "names no channel"
          else "holds a channel in one field"),
          result.diagnostics
        )
    val helperDir = sources(Seq("Helpers.scala"), "helper")
    val (helperJar, _) = compile(helperDir, "helper", Seq("--jar", jar.toString))
    val clientDir = sources(Seq("Client.scala", "HelperNative.scala"), "client")
    val (clientJar, clientCp) =
      compile(clientDir, "client", Seq("--jar", jar.toString, "--jar", helperJar.toString))
    val helperCp = Files.writeString(scratch.resolve("helper.classpath"), clientCp)
    val nativeHelper = run(clientCp, "framework.inboxhelper.HelperNative")
    assertEquals(nativeHelper, run(clientCp, "framework.inboxhelper.HelperNative"))
    val helper = payload(nativeHelper, "helper")
    val oldHelper = frozen("helper-old.json")
    assertEquals(helper.types, oldHelper.types.map(_.clearPosition))
    assertEquals(helper.channels, oldHelper.channels.map(_.clearPosition))
    val helperOutput = scratch.resolve("helper-old.json")
    val result = InProcess.lift(
      tools.directory,
      Seq(
        s"$clientJar=$clientDir/,$helperJar=$helperDir/,$jar=$dir/",
        helperCp.toString,
        helperOutput.toString,
        "fixture.constructedinbox.HelperMachine"
      ),
      policy = ModulePolicy.Generic
    )
    println(s"old-inbox-helper:exit=${result.exit}\n${result.diagnostics}")
    assertEquals(result.exit, 0, result.diagnostics)

  test("native state contributions match the complete frozen old messages in two JVMs"):
    val dir = sources(Seq("Inboxes.scala", "NativeBaseline.scala"), "native-baseline")
    val (jar, cp) = compile(dir, "native-baseline")
    val first = run(cp, "framework.inboxbaseline.NativeBaseline")
    assertEquals(first, run(cp, "framework.inboxbaseline.NativeBaseline"))
    val native = payload(first, "inboxes")
    val original = frozen("original.json")
    assertEquals(native.types, original.types.map(_.clearPosition))
    assertEquals(native.channels, original.channels.map(_.clearPosition))
    val replay = lift(jar, dir, cp, "native-replay", roots)
    assertEquals(replay.types.map(_.clearPosition), original.types.map(_.clearPosition))
    assertEquals(replay.channels.map(_.clearPosition), original.channels.map(_.clearPosition))

  test("compiled suppliers preserve local lazy Given and native catalog timing"):
    val dir = sources(Seq("Evaluation.scala"), "evaluation")
    val (_, cp) = compile(dir, "evaluation")
    val first = run(cp, "framework.inboxevaluation.Evaluation")
    assertEquals(first, run(cp, "framework.inboxevaluation.Evaluation"))
    assert(
      first.contains(
        "evaluation:constructed=0,schema=0,receiver=1,lazy-values=1,eager-values=1,throwing=1"
      )
    )

  test("unsupported lexical Inbox helpers and catalog aliases refuse without guessing"):
    val dir = sources(Seq("Unsupported.scala", "UnsupportedNative.scala"), "unsupported")
    val (jar, cp) = compile(dir, "unsupported")
    val first = run(cp, "framework.inboxunsupported.UnsupportedNative")
    assertEquals(first, run(cp, "framework.inboxunsupported.UnsupportedNative"))
    assert(first.contains("unsupported:passed"))
    val classpath = Files.writeString(scratch.resolve("unsupported.classpath"), cp)
    for name <- Seq("Forwarded", "Aliased") do
      val result = InProcess.lift(
        tools.directory,
        Seq(
          s"$jar=$dir/",
          classpath.toString,
          scratch.resolve(s"unsupported-$name.json").toString,
          s"fixture.inboxunsupported.${name}Machine"
        ),
        policy = ModulePolicy.Generic
      )
      assertEquals(result.exit, 1, result.diagnostics)
      assert(result.diagnostics.contains("names no channel"), result.diagnostics)
      println(s"old-inbox-unsupported-$name:exit=${result.exit}\n${result.diagnostics}")
