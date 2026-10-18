package framework.check

import java.nio.file.Files
import java.nio.file.Path
import scala.jdk.CollectionConverters.*

class FrameworkLoopSuite extends munit.FunSuite:
  test("the Make entrypoint selects only the generic runner without system prerequisites"):
    val root = Scratch.directory("framework-make")
    Files.copy(Tools.here.directory.resolve("Makefile"), root.resolve("Makefile"))
    Files.writeString(root.resolve("go.mod"), "module fixture\n")
    val stubs = Stubs(Scratch.directory("framework-make-tools"))
      .tool("git", s"printf '%s\\n' '$root'")
      .tool("find", "echo 'system source discovery is forbidden' >&2; exit 9")
      .tool("go", "echo 'system tool discovery is forbidden' >&2; exit 9")
    val tools = Tools(root, sys.env.updated("PATH", s"${stubs.directory}:/usr/bin:/bin"))
    val ran = tools.run("make", Seq("-n", "check-framework"))
    assertEquals(ran.exit, 0, ran.diagnostics)
    assert(ran.output.contains("-M framework.check.checkFramework"), ran.output)
    assert(!ran.output.contains("system source discovery is forbidden"), ran.output)
    assert(!ran.output.contains("system tool discovery is forbidden"), ran.output)
    assert(!ran.output.contains("model/temporal"), ran.output)
    assert(!ran.output.contains("api-scalapb.jar"), ran.output)
    assert(!ran.output.contains("model/project.scala"), ran.output)

  test("unknown export suites fail classification instead of silently earning a hold"):
    val module = FrameworkSuites.modules.find(_.name == "export").get
    val error = intercept[GateError]:
      FrameworkSuites.held(
        Declared(module, "model/framework/export/New.test.scala", "framework.export.NewSuite")
      )
    assert(error.getMessage.contains("unclassified framework suite"))

  test("the expectation lifter and capture suites are classified generic"):
    val module = FrameworkSuites.modules.find(_.name == "export").get
    for (file, suite) <- Seq(
        "model/framework/export/Expect.test.scala" -> "framework.export.ExpectFixtures",
        "model/framework/export/ExpectCapture.test.scala" -> "framework.export.ExpectCaptureSuite"
      )
    do assertEquals(FrameworkSuites.held(Declared(module, file, suite)), None)

  private def fixture(): (Path, Tools) =
    val root = Scratch.directory("framework-loop")
    for (file, source) <- Seq(
        "model/framework/dsl/Core.test.scala" -> "package framework.dsl\nclass CoreSuite extends munit.FunSuite\n",
        "model/framework/check/Check.test.scala" -> "package framework.check\nclass CheckSuite extends munit.FunSuite\n",
        "model/framework/samples/Bank.test.scala" -> "package framework.dsl\nclass BankTest extends munit.FunSuite\n",
        "model/framework/samples/Kettle.test.scala" -> "package framework.dsl\nclass KettleTest extends munit.FunSuite\n",
        "model/framework/samples/ReliableBroadcast.test.scala" -> "package framework.dsl\nclass ReliableBroadcastTest extends munit.FunSuite\n",
        "model/framework/samples/TwoLayeredCache.test.scala" -> "package framework.dsl\nclass TwoLayeredCacheTest extends munit.FunSuite\n",
        "model/framework/samples/TwoPhaseCommit.test.scala" -> "package framework.dsl\nclass TwoPhaseCommitTest extends munit.FunSuite\n",
        "model/framework/samples/Voting.test.scala" -> "package framework.dsl\nclass VotingTest extends munit.FunSuite\n",
        "model/framework/samples/Inputs.test.scala" -> "package framework.dsl\nclass InputsTest extends munit.FunSuite\n",
        "model/framework/samples/VotingBehavior.test.scala" -> "package framework.dsl\nclass VotingBehaviorTest extends munit.FunSuite\n",
        "model/framework/export/Ids.test.scala" -> "package framework.`export`\nclass DerivedIds extends munit.FunSuite\n",
        "model/build/ir-scalapb.jar" -> "existing IR",
        "model/build/api-scalapb.jar" -> "poison API",
        "model/build/model-scala.jar" -> "poison Models"
      )
    do
      val path = root.resolve(file)
      Files.createDirectories(path.getParent): Unit
      Files.writeString(path, source): Unit
    val stubs = Stubs(Scratch.directory("framework-loop-tools"))
      .tool(
        "scala-cli",
        s"""printf '%s\n' "$$*" >> '$root/calls'
        |case "$$*" in
        |  *model/project.scala*|*model/temporal*|*model-scala.jar*|*api-scalapb.jar*) exit 7 ;;
        |esac
        |case "$$*" in
        |  *--print-class-path*)
        |    while [ "$$1" != --compilation-output ]; do shift; done
        |    shift; /usr/bin/touch "$$1"
        |    if [ -f '$root/poison' ]; then printf '%s:$root/model/build/api-scalapb.jar\n' "$$1"
        |    else printf '%s:$root/model/build/ir-scalapb.jar\n' "$$1"; fi ;;
        |  *--assembly*)
        |    while [ "$$1" != -o ]; do shift; done
        |    shift; /usr/bin/touch "$$1" ;;
        |  *org.junit.runner.JUnitCore*)
        |    [ -s "$$UMPIRE_FRAMEWORK_BUILD/framework.classpath" ] || exit 9
        |    case "$$*" in
        |      *framework.check.CheckSuite*)
        |        if [ -f '$root/held' ]; then
        |          printf 'HELD test go case: runs Go\nHELD test go case: runs Go\n'
        |        fi ;;
        |    esac
        |    if [ -f '$root/empty' ]; then printf 'OK (0 tests)\n'
        |    else printf 'OK (1 test)\n'; fi ;;
        |  *) exit 8 ;;
        |esac
        |""".stripMargin
      )
    root -> stubs.tools(root)

  test("fresh generic artifacts and exact discovered suites reach every module command"):
    val (root, tools) = fixture()
    val record = Files.createDirectory(root.resolve("record"))
    FrameworkLoop(tools, new java.io.PrintStream(java.io.OutputStream.nullOutputStream()), record)
      .check()
    val commands = Files.readAllLines(root.resolve("calls"))
    assertEquals(commands.size(), 5)
    val text = commands.toString
    assert(
      text.contains(s"--scalac-option -sourceroot --scalac-option ${root.resolve("model")}"),
      text
    )
    val samples = Seq(
      "framework.dsl.BankTest",
      "framework.dsl.KettleTest",
      "framework.dsl.ReliableBroadcastTest",
      "framework.dsl.TwoLayeredCacheTest",
      "framework.dsl.TwoPhaseCommitTest",
      "framework.dsl.VotingTest",
      "framework.dsl.InputsTest",
      "framework.dsl.VotingBehaviorTest"
    )
    val selectors = commands.asScala.toVector.flatMap(_.split(" ").toVector)
    for suite <- Seq(
        "framework.dsl.CoreSuite",
        "framework.check.CheckSuite",
        "framework.export.DerivedIds"
      ) ++ samples
    do assertEquals(selectors.count(_ == suite), 1, text)
    val compiled = commands.asScala.find(_.contains("--print-class-path")).get
    assert(!compiled.contains("model/framework/samples"), compiled)
    val core = commands.asScala.find(_.contains("framework.dsl.CoreSuite")).get
    assert(core.contains("model/framework/samples"), core)
    for suite <- samples do assert(core.contains(suite), core)
    assert(Files.isRegularFile(record.resolve("build/framework.jar")))
    assert(Files.isRegularFile(record.resolve("build/reify.jar")))
    val recorded = Files.readString(record.resolve("record.md"))
    assert(recorded.contains("PASS framework gate: 11 generic suites; 0 held suites"), recorded)
    for suite <- samples do
      assert(recorded.contains(s"GENERIC $suite (model/framework/samples/"), recorded)

  test("the native samples are core suites; the reusable library includes the product checker"):
    val core = FrameworkSuites.modules.find(_.name == "core").get
    assert(core.suiteRoots.contains("model/framework/samples"))
    assert(core.inputs.contains("model/framework/samples"))
    assert(core.inputs.contains("model/framework/checker"))
    assertEquals(
      ModelSources.frameworkCore,
      Seq("model/framework/dsl", "model/framework/checker", "model/framework/realize")
    )
    val declared = FrameworkSuites.declared(Tools.here.directory)
    val samples = declared.filter(_.file.startsWith("model/framework/samples/"))
    assertEquals(
      samples.map(_.name).sorted,
      Vector(
        "framework.dsl.BankTest",
        "framework.dsl.InputsTest",
        "framework.dsl.KettleTest",
        "framework.dsl.ReliableBroadcastTest",
        "framework.dsl.TwoLayeredCacheTest",
        "framework.dsl.TwoPhaseCommitTest",
        "framework.dsl.VotingBehaviorTest",
        "framework.dsl.VotingTest"
      )
    )
    assert(samples.forall(_.module == core))
    assertEquals(declared.map(_.name).distinct.size, declared.size)

  test("held Go cases are recorded once with their reason and earn no pass credit"):
    val (root, tools) = fixture()
    Files.writeString(root.resolve("held"), "")
    val record = Files.createDirectory(root.resolve("record"))
    FrameworkLoop(tools, new java.io.PrintStream(java.io.OutputStream.nullOutputStream()), record)
      .check()
    val lines = Files.readAllLines(record.resolve("record.md")).asScala.toVector
    assertEquals(lines.count(_ == "HELD test go case: runs Go"), 1, lines.mkString("\n"))
    assert(
      lines.contains(
        "PASS check: 1 suites, 1 tests; ignored integration cases earn no pass credit"
      ),
      lines.mkString("\n")
    )

  // The Go-running samples cases are held only in framework mode, before their bodies run. Run this
  // once with UMPIRE_FRAMEWORK_BUILD set (it then executes the two cases, with poisoned report
  // inputs, and proves both are ignored) and once without (it inspects registration only).
  test("only the two Go-running samples cases are held, and only in framework mode"):
    val goCases = Set(
      "the generated Go checker overlaps at most two checks, joins failures and orders diagnostics",
      "the generated parallel checker preserves complete real sample reports"
    )
    val framework = sys.env.contains("UMPIRE_FRAMEWORK_BUILD")
    val registered = SamplesSuite().munitTests()
    def base(name: String) = name.takeWhile(_ != '[').trim
    val ignored = registered.filter(_.tags.contains(munit.Ignore)).map(t => base(t.name)).toSet
    assertEquals(goCases -- registered.map(t => base(t.name)), Set.empty[String])
    if framework then
      assertEquals(ignored, goCases)
      for test <- registered if goCases(base(test.name)) do
        assert(test.name.endsWith(s"[framework-held: ${GoHeld.reason}]"), test.name)
      val selected = registered.filter(t => goCases(base(t.name))).map(_.name).toSet
      val request = org.junit.runner.Request
        .aClass(classOf[SamplesSuite])
        .filterWith(new org.junit.runner.manipulation.Filter:
          def shouldRun(d: org.junit.runner.Description): Boolean =
            d.isSuite || selected(d.getMethodName)
          def describe: String = "the Go-running samples cases")
      val result = org.junit.runner.JUnitCore().run(request)
      assertEquals(result.getFailureCount, 0, result.getFailures.toString)
      assertEquals(result.getRunCount, 0)
      assertEquals(result.getIgnoreCount, 2)
    else
      assertEquals(ignored, Set.empty[String])
      for test <- registered if goCases(test.name) do assert(!test.name.contains("framework-held"))

  test("a poisoned compiler classpath or zero executed tests cannot pass the gate"):
    for (marker, message) <- Seq(
        "poison" -> "system-dependent classpath",
        "empty" -> "no executed tests"
      )
    do
      val (root, tools) = fixture()
      Files.writeString(root.resolve(marker), "")
      val record = Files.createDirectory(root.resolve("record"))
      val error = intercept[GateError]:
        FrameworkLoop(
          tools,
          new java.io.PrintStream(java.io.OutputStream.nullOutputStream()),
          record
        ).check()
      assert(error.getMessage.contains(message), error.getMessage)
