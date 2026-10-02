package umpire.gate

import java.nio.file.{Files, Path}
import java.nio.file.attribute.PosixFilePermissions

/** A directory of stand-in tools, each a script that plays one behavior of the real tool. */
final class Stubs(val directory: Path):
  def tool(name: String, script: String): Stubs =
    val file = Files.writeString(directory.resolve(name), s"#!/bin/sh\n$script\n")
    Files.setPosixFilePermissions(file, PosixFilePermissions.fromString("rwxr-xr-x"))
    this

  /** The tools as a run in `in` finds them: nothing but the stand-ins is on the PATH. */
  def tools(in: Path, environment: (String, String)*): Tools =
    Tools(in, Map("PATH" -> directory.toString) ++ environment)

class ToolsSuite extends munit.FunSuite:
  private def stubs() = Stubs(Files.createTempDirectory("umpire-gate-stubs"))
  private val work = Files.createTempDirectory("umpire-gate-work")
  private def refused(body: => Any): String = intercept[ToolError](body).getMessage

  // scala-cli prints a -Werror failure with the bracket's word colored, and exits 0.
  private val printedError =
    """printf '[\033[31merror\033[0m] ./model/umpire/Machine.scala:71:3\n'"""

  test("a scala-cli run that prints an error and exits 0 failed"):
    val tools = stubs().tool("scala-cli", s"$printedError; exit 0").tools(work)
    val ran = tools.scalaCli(Seq("compile", "model/umpire"))
    assertEquals(ran.exit, 0)
    assertEquals(ran.errors, Seq("[error] ./model/umpire/Machine.scala:71:3"))
    assert(ran.failed)
    val message = refused(ran.orFail())
    assert(message.startsWith(s"scala-cli printed an error and exited 0 in $work: "), message)
    assert(message.contains("compile model/umpire"), message)
    assert(message.contains("[error] ./model/umpire/Machine.scala:71:3"), message)

  test("a nonzero exit failed, and the failure names the tool, the status and the command"):
    val tools = stubs().tool("go", "echo 'FAIL tools/umpire/model'; exit 3").tools(work)
    val ran = tools.run("go", Seq("test", "./tools/umpire/..."))
    assert(ran.failed)
    val message = refused(ran.orFail())
    assert(message.startsWith(s"go exited 3 in $work: "), message)
    assert(
      message.contains("test ./tools/umpire/...") && message.contains("FAIL tools/umpire/model"),
      message
    )

  test("a run that exits 0 and prints no error passed, whatever else it printed"):
    val script =
      "echo \"$@\"; echo '[hint] a newer version exists'; echo 'WARNING: deprecated'; echo warned >&2"
    val tools =
      stubs().tool("scala-cli", script).tool("go", "echo '[error] is only a word here'").tools(work)
    val ran = tools.scalaCli(Seq("run", "model/lifter", "--", "a.jar", "root")).orFail()
    // scala-cli's flag is its own, so it stands before the program's arguments.
    assertEquals(
      ran.output.linesIterator.next(),
      "run model/lifter --suppress-outdated-dependency-warning -- a.jar root"
    )
    assert(ran.output.contains("warned"), "standard error is kept with standard output")
    assertEquals(ran.diagnostics.linesIterator.toSeq.drop(1), Seq("warned"))
    assert(!tools.run("go", Seq("vet")).failed, "only scala-cli's printed errors are a verdict")

  test("standard output can be kept apart from standard error"):
    val tools = stubs().tool("scala-cli", "echo a.jar:b.jar; echo Compiling >&2").tools(work)
    assertEquals(
      tools.scalaCli(Seq("compile", "--print-class-path"), Output.KeptApart).output,
      "a.jar:b.jar\n"
    )

  test("the tool's environment and directory are the ones given"):
    val tools = stubs()
      .tool("go", "echo \"$GOFLAGS $PWD\"")
      .tools(work)
      .withEnvironment("GOFLAGS" -> "-tags=test_dep")
    assertEquals(tools.run("go", Nil).output.trim, s"-tags=test_dep ${work.toRealPath()}")

  for tool <- Seq("scala-cli", "protoc", "go") do
    test(s"a missing $tool fails with its name and the directories that were searched"):
      val empty = stubs()
      val message = refused(empty.tools(work).run(tool, Seq("--version")))
      assert(
        message.startsWith(
          s"$tool is missing: no directory of the PATH holds it (${empty.directory})"
        ),
        message
      )
      assert(message.contains("mise exec --"), message)

  test(
    "a tool is missing when the PATH is empty, when it is not executable and when it is a directory"
  ):
    assert(
      refused(Tools(work, Map.empty).find("go")).startsWith("go is missing: the PATH is empty")
    )
    val held = stubs()
    Files.writeString(held.directory.resolve("go"), "not executable")
    Files.createDirectory(held.directory.resolve("protoc"))
    assert(refused(held.tools(work).find("go")).startsWith("go is missing: "))
    assert(refused(held.tools(work).find("protoc")).startsWith("protoc is missing: "))
    val path = held.directory.resolve("bin/java").toString
    assertEquals(
      refused(held.tools(work).find(path)).takeWhile(_ != '.'),
      s"$path is missing: it is not an executable file"
    )

  test("a tool named by its path is run as given"):
    val held = stubs().tool("java", "echo ran")
    assertEquals(
      Tools(work, Map.empty).run(held.directory.resolve("java").toString, Nil).output,
      "ran\n"
    )

  test("a run in a directory that does not exist names the directory"):
    val gone = work.resolve("gone")
    val message = refused(stubs().tool("go", "true").tools(gone).run("go", Nil))
    assertEquals(message, s"go cannot run in $gone: it is not a directory")

  test("the repository is the nearest directory above that holds the gate"):
    val repository = Files.createTempDirectory("umpire-gate-repository")
    Files.createDirectories(repository.resolve("model/gate"))
    Files.writeString(repository.resolve("model/gate/project.scala"), "")
    val inside = Files.createDirectories(repository.resolve("model/lifter/test"))
    assertEquals(Tools.repository(inside), repository)
    assertEquals(Tools.repository(repository), repository)
    val message = refused(Tools.repository(work))
    assert(message.contains(s"no model/gate/project.scala in $work or above it"), message)
