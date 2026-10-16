package framework.`export`

import java.nio.file.{Files, Path}
import java.util.Base64
import scala.concurrent.duration.*
import framework.check.{Output, RetainedScratch, Tools}
import framework.lint.ModulePolicy
import io.temporal.server.api.umpire.v1 as ir

class ConstructedReferencesSuite extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private lazy val scratch =
    retained(tools.directory.resolve(".flow/tmp"), "constructed-references.")
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
  private def settings(relaxed: Boolean, mode: String = "signature") = Seq(
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
    s"-P:umpire-reify:mode=$mode"
  ) ++
    (Seq(
      "-deprecation",
      "-feature",
      "-unchecked",
      "-Wvalue-discard",
      "-Wnonunit-statement",
      "-Wsafe-init",
      "-Wimplausible-patterns",
      "-language:strictEquality"
    )
      ++ (if relaxed then Nil else Seq("-Werror", "-Wunused:all")))
      .flatMap(o => Seq("--scalac-option", o))

  private def compile(dir: Path, name: String, extra: Seq[String] = Nil): (Path, String) =
    val out = scratch.resolve(s"$name.jar")
    val ran = tools.scalaCli(
      Seq(
        "compile",
        dir.toString,
        "--print-class-path",
        "--compilation-output",
        out.toString
      ) ++ settings(name == "oracle" || name == "rejects") ++ extra,
      Output.KeptApart
    )
    assertEquals(ran.exit, 0, ran.diagnostics)
    out -> ran.output.trim

  private def run(cp: String, main: String): String =
    val ran = tools.run("java", Seq("-cp", cp, main))
    assertEquals(ran.exit, 0, ran.diagnostics)
    ran.output

  private def fixture(source: String, folder: String): Path =
    val stored = tools.directory.resolve("model/framework/export/testdata/constructedReferences")
    val dir = Files.createDirectories(scratch.resolve(folder))
    Files.copy(stored.resolve(s"$source.scala"), dir.resolve(s"$source.scala")): Unit
    dir

  private def fixtures(name: String): Path = fixture(name, name)

  test("five native references match frozen old fairRefs in order in two separate JVMs"):
    val dir = fixtures("Native")
    val original =
      tools.directory.resolve("model/framework/export/testdata/lifts/ComposedProgress.scala")
    Files.copy(original, dir.resolve("ComposedProgress.scala")): Unit
    val (_, cp) = compile(dir, "oracle")
    val first = run(cp, "framework.references.Native")
    val second = run(cp, "framework.references.Native")
    assertEquals(first, second)
    val bytecode =
      tools.run("javap", Seq("-c", "-p", "-classpath", cp, "framework.references.Native$"))
    assertEquals(bytecode.exit, 0, bytecode.diagnostics)
    assert(bytecode.output.contains("Reify$.captured"), bytecode.output)
    val payload =
      first.linesIterator.find(_.startsWith("references:")).get.stripPrefix("references:")
    val native = ir.Model.parseFrom(Base64.getDecoder.decode(payload))
    val frozen = scalapb.json4s.JsonFormat.fromJsonString[ir.Model](
      Files.readString(
        tools.directory.resolve(
          "model/framework/export/testdata/lifts/expected/composedProgress.json"
        )
      )
    )
    assertEquals(native.assumptions.flatMap(_.fairRefs), frozen.assumptions.flatMap(_.fairRefs))
    val saved = Files.writeString(scratch.resolve("oracle.classpath"), cp)
    val out = scratch.resolve("current-reader.json")
    val lifted = InProcess.lift(
      tools.directory,
      Seq(
        s"${scratch.resolve("oracle.jar")}=$dir/",
        saved.toString,
        out.toString,
        "fixture.composedprogress.ComposedProgress$package$.drainsThrough",
        "fixture.composedprogress.ComposedProgress$package$.gradedThrough"
      ),
      policy = ModulePolicy.Generic
    )
    assertEquals(lifted.exit, 0, lifted.diagnostics)
    val current = scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(out))
    assertEquals(current.assumptions.flatMap(_.fairRefs), native.assumptions.flatMap(_.fairRefs))

    val strict = new ModulePolicy:
      val root = ""
      def permitted(from: String, to: String, test: Boolean): Option[Boolean] =
        Option.when(to.startsWith("io.temporal.server.api.umpire"))(false)
      def testModule(file: Path): Option[String] = None
    val refusedOutput = scratch.resolve("policy-refused.json")
    val refused = InProcess.lift(
      tools.directory,
      Seq(
        s"${scratch.resolve("oracle.jar")}=$dir/",
        saved.toString,
        refusedOutput.toString,
        "fixture.composedprogress.ComposedProgress$package$.drainsThrough"
      ),
      policy = strict
    )
    assertEquals(refused.exit, 1, refused.diagnostics)
    assert(refused.diagnostics.contains("R4.module-"), refused.diagnostics)
    assert(!Files.exists(refusedOutput))
    val repeated = InProcess.lift(
      tools.directory,
      Seq(
        s"${scratch.resolve("oracle.jar")}=$dir/",
        saved.toString,
        out.toString,
        "fixture.composedprogress.ComposedProgress$package$.drainsThrough",
        "fixture.composedprogress.ComposedProgress$package$.gradedThrough"
      ),
      policy = ModulePolicy.Generic
    )
    assertEquals(repeated.exit, 0, repeated.diagnostics)
    val afterRefusal = scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(out))
    assertEquals(afterRefusal, current)

  test("native coordinate ownership, derived replacement, refusals and evaluation counts"):
    val (_, cp) = compile(fixtures("Controls"), "controls")
    run(cp, "framework.references.Controls"): Unit

  test("full mode still refuses runtime endpoint inspection at the authored source"):
    val dir = fixture("Controls", "controls-full")
    val compiled = tools.scalaCli(
      Seq("compile", dir.toString) ++ settings(relaxed = true, mode = "full"),
      Output.KeptApart
    )
    assertEquals(compiled.exit, 1, compiled.diagnostics)
    assert(
      compiled.diagnostics.contains("Controls.scala:34:"),
      compiled.diagnostics
    )
    assert(
      compiled.diagnostics.contains("outside the liftable subset"),
      compiled.diagnostics
    )

  test("helper jars retain selectors and composed aliases without executing callback arguments"):
    val (helper, _) = compile(fixtures("Helpers"), "helpers")
    val (_, cp) = compile(fixtures("Client"), "client", Seq("--jar", helper.toString))
    run(cp, "framework.references.Client"): Unit

  test(
    "unchanged composed progress refusals retain native references and current-reader diagnostics"
  ):
    val dir = fixtures("Rejects")
    for name <- Seq("ComposedProgress", "ComposedProgressRejects") do
      Files.copy(
        tools.directory.resolve(s"model/framework/export/testdata/lifts/$name.scala"),
        dir.resolve(s"$name.scala")
      ): Unit
    val (jar, cp) = compile(dir, "rejects")
    run(cp, "framework.references.Rejects"): Unit
    val saved = Files.writeString(scratch.resolve("rejects.classpath"), cp)
    val roots = Seq(
      ("anotherOwner", "own composition", 17),
      ("onAMachine", "own composition", 20),
      ("mixedFairness", "both actions and composed steps", 24),
      ("spreadFairness", "one composition", 28),
      ("oneSidedClass", "takes inputs", 32),
      ("pairedOwn", "steps only with its pair", 36),
      ("unpairedSync", "no sync", 40),
      ("noStep", "at least one step", 45)
    )
      .map((name, reason, line) =>
        ("fixture.composedprogressrejects.ComposedProgressRejects$package$." + name, reason, line)
      ) :+
      (("fixture.composedprogressrejects.AssumingFlow", "composed steps", 22))
    for (root, reason, line) <- roots do
      val lifted = InProcess.lift(
        tools.directory,
        Seq(s"$jar=$dir/", saved.toString, scratch.resolve("rejected.json").toString, root),
        policy = ModulePolicy.Generic
      )
      assertEquals(lifted.exit, 1, s"$root\n${lifted.diagnostics}")
      assert(lifted.diagnostics.contains(reason), lifted.diagnostics)
      assert(
        lifted.diagnostics.contains(s"ComposedProgressRejects.scala:$line:"),
        lifted.diagnostics
      )
