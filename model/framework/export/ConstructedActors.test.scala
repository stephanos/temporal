package framework.`export`

import java.nio.file.{Files, Path}
import java.util.Base64
import scala.concurrent.duration.*
import framework.check.{Output, RetainedScratch, Tools}
import io.temporal.server.api.umpire.v1 as ir

class ConstructedActorsSuite extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private lazy val scratch = retained(tools.directory.resolve(".flow/tmp"), "constructed-actors.")
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
    s"-Xplugin:$plugin"
  )
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
  private def run(cp: String, main: String): String =
    val result = tools.run("java", Seq("-cp", cp, main))
    assertEquals(result.exit, 0, result.diagnostics)
    result.output
  private def payload(output: String, label: String): ir.Model = ir.Model.parseFrom(
    Base64.getDecoder.decode(
      output.linesIterator.find(_.startsWith(label + ":")).get.stripPrefix(label + ":")
    )
  )
  private def frozen(path: String): ir.Model = scalapb.json4s.JsonFormat
    .fromJsonString[ir.Model](Files.readString(tools.directory.resolve(path)))

  private val stored = tools.directory.resolve("model/framework/export/testdata/constructedActors")
  private def sources(names: Seq[String], folder: String): Path =
    val dir = Files.createDirectories(scratch.resolve(folder))
    names.foreach(name => Files.copy(stored.resolve(name), dir.resolve(name)): Unit)
    dir

  test("complete Actor Actions match frozen old output and current-reader replay in two JVMs"):
    val dir = sources(Seq("ConstructedActors.scala", "Native.scala"), "oracle")
    val (jar, cp) = compile(dir, "oracle")
    val first = run(cp, "framework.actorcontrols.Native")
    assertEquals(first, run(cp, "framework.actorcontrols.Native"))
    assert(first.contains("controls:passed"))
    val original = frozen("model/framework/export/testdata/constructedActors/original.json")
    val native = payload(first, "actors")
    assertEquals(native.actions, original.actions.map(_.clearPosition))
    val classpath = Files.writeString(scratch.resolve("oracle.classpath"), cp)
    val output = scratch.resolve("current-reader.json")
    val replay = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=$dir/",
        classpath.toString,
        output.toString,
        "fixture.constructedactor.ConstructedActors$package$.progress"
      )
    )
    assertEquals(replay.exit, 0, replay.diagnostics)
    val current = scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(output))
    assertEquals(current.actions.map(_.clearPosition), original.actions.map(_.clearPosition))

  test("actual old Actor admission qualifies new constructors helpers aliases and inheritance"):
    val dir = sources(Seq("ConstructedActors.scala", "Forms.scala"), "forms")
    val (jar, cp) = compile(dir, "forms")
    val first = run(cp, "fixture.constructedactor.Forms")
    assertEquals(first, run(cp, "fixture.constructedactor.Forms"))
    val native = payload(first, "forms")
    val classpath = Files.writeString(scratch.resolve("forms.classpath"), cp)
    for (name, accepted) <- Seq(
        "imported" -> true,
        "namedFactory" -> true,
        "emptyAliased" -> true,
        "constructed" -> false,
        "namedConstructor" -> false,
        "omittedConstructor" -> false,
        "moduleAlias" -> false,
        "helped" -> false,
        "inherited" -> false
      )
    do
      val result = InProcess.lift(
        tools.directory,
        Seq(
          s"$jar=$dir/",
          classpath.toString,
          scratch.resolve(s"$name.json").toString,
          s"fixture.constructedactor.Forms$$package$$.${name}Progress"
        )
      )
      assertEquals(result.exit, if accepted then 0 else 1, result.diagnostics)
      println(s"old-actor-$name: exit ${result.exit}\n${result.diagnostics}")
      if accepted then
        val model = scalapb.json4s.JsonFormat
          .fromJsonString[ir.Model](Files.readString(scratch.resolve(s"$name.json")))
        val id = s"fixture.constructedactor.Forms.$name"
        assertEquals(
          native.actions.find(_.id == id),
          model.actions.find(_.id == id).map(_.clearPosition)
        )
        assert(native.actions.exists(_.id == id))
      if !accepted then
        assert(
          result.diagnostics.contains("expected a string") ||
            result.diagnostics.contains("which is no object"),
          result.diagnostics
        )

  test(
    "helper jar modules retain exact spelling while factories and subclasses keep qualified support"
  ):
    val helper = sources(Seq("Helpers.scala"), "helper")
    val (helperJar, _) = compile(helper, "helper")
    val client = sources(Seq("Client.scala"), "client")
    val (clientJar, cp) = compile(client, "client", Seq("--jar", helperJar.toString))
    val first = run(cp, "framework.actorclient.Client")
    assertEquals(first, run(cp, "framework.actorclient.Client"))
    assert(first.contains("helper-controls:passed"))
    val classpath = Files.writeString(scratch.resolve("helper.classpath"), cp)
    for (root, accepted) <- Seq("progress" -> true, "helperProgress" -> false) do
      val output = scratch.resolve(s"$root.json")
      val old = InProcess.lift(
        tools.directory,
        Seq(
          s"$clientJar=$client/,$helperJar=$helper/",
          classpath.toString,
          output.toString,
          s"framework.actorclient.Client$$package$$.$root"
        )
      )
      assertEquals(old.exit, if accepted then 0 else 1, old.diagnostics)
      println(s"old-actor-jar-$root: exit ${old.exit}\n${old.diagnostics}")
      if accepted then
        val model = scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(output))
        assertEquals(payload(first, "direct").actions, model.actions.map(_.clearPosition))
      else assert(old.diagnostics.contains("expected a string"), old.diagnostics)
    val subclass = sources(Seq("Subclass.scala"), "subclass")
    val subclassJar = scratch.resolve("subclass.jar")
    val arguments = Seq(
      "compile",
      subclass.toString,
      "--print-class-path",
      "--compilation-output",
      subclassJar.toString
    )
    val dependencies = Seq("--jar", helperJar.toString, "--jar", clientJar.toString)
    val refused = tools.scalaCli(arguments ++ settings ++ dependencies, Output.KeptApart)
    assertEquals(refused.exit, 1, refused.diagnostics)
    assert(refused.diagnostics.contains("no validated model helper contract"), refused.diagnostics)
    println(s"native-actor-jar-subclass: exit ${refused.exit}\n${refused.diagnostics}")
    val built = tools.scalaCli(arguments ++ settings.dropRight(2) ++ dependencies, Output.KeptApart)
    assertEquals(built.exit, 0, built.diagnostics)
    val oldClasspath = Files.writeString(scratch.resolve("subclass.classpath"), built.output.trim)
    val old = InProcess.lift(
      tools.directory,
      Seq(
        s"$subclassJar=$subclass/,$clientJar=$client/,$helperJar=$helper/",
        oldClasspath.toString,
        scratch.resolve("subclass-old.json").toString,
        "framework.actorclient.Subclass$package$.subclassProgress"
      )
    )
    assertEquals(old.exit, 1, old.diagnostics)
    assert(old.diagnostics.contains("expected a string"), old.diagnostics)
    println(s"old-actor-jar-subclass: exit ${old.exit}\n${old.diagnostics}")
