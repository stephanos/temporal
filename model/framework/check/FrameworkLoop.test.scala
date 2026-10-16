package framework.check

import java.nio.file.Files
import java.nio.file.Path

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

  private def fixture(): (Path, Tools) =
    val root = Scratch.directory("framework-loop")
    for (file, source) <- Seq(
        "model/framework/dsl/Core.test.scala" -> "package framework.dsl\nclass CoreSuite extends munit.FunSuite\n",
        "model/framework/check/Check.test.scala" -> "package framework.check\nclass CheckSuite extends munit.FunSuite\n",
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
    for suite <- Seq(
        "framework.dsl.CoreSuite",
        "framework.check.CheckSuite",
        "framework.export.DerivedIds"
      )
    do assert(text.contains(suite), text)
    assert(Files.isRegularFile(record.resolve("build/framework.jar")))
    assert(Files.isRegularFile(record.resolve("build/reify.jar")))
    assert(
      Files
        .readString(record.resolve("record.md"))
        .contains("PASS framework gate: 3 generic suites; 0 held suites")
    )

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
