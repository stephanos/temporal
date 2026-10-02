package umpire.gate

import java.io.{ByteArrayOutputStream, PrintStream}
import java.nio.file.{Files, Path}
import java.nio.file.attribute.FileTime

class GateSuite extends munit.FunSuite:
  private val schemaFile = "proto/internal/temporal/server/api/umpire/v1/ir.proto"

  /** What a run of the gate's command line answered: its status and what it printed. */
  final private case class Answer(status: Int, out: String, err: String)
  private def gate(tools: => Tools, arguments: String*): Answer =
    val (out, err) = (ByteArrayOutputStream(), ByteArrayOutputStream())
    val status = Gate.main(arguments, tools, PrintStream(out), PrintStream(err))
    Answer(status, out.toString, err.toString)

  /**
   * A repository with an IR schema and its generated Go code, and stand-ins for the tools: each
   * records its command line in `log`, and scala-cli writes the file it is asked to package.
   */
  final private class Repository(scalaCli: String = ""):
    val root: Path = Files.createTempDirectory("umpire-gate-repository")
    val log: Path = root.resolve("tools.log")
    val schema: Path = root.resolve(schemaFile)
    Files.createDirectories(schema.getParent)
    Files.writeString(schema, "syntax = \"proto3\";\n")
    private val generated =
      Files.createDirectories(root.resolve("api/umpire/v1")).resolve("ir.pb.go")
    Files.writeString(generated, "package umpire\n")
    Files.setLastModifiedTime(
      generated,
      FileTime.fromMillis(Files.getLastModifiedTime(schema).toMillis + 1000)
    )
    Files.createDirectories(root.resolve("model/gen"))
    private val record = "echo \"${0##*/} $*\" >> \"$TOOLS_LOG\""
    private val packaging =
      "out=; previous=; for a in \"$@\"; do [ \"$previous\" = -o ] && out=$a; previous=$a; done; [ -n \"$out\" ] && : > \"$out\""
    private val stubs = Stubs(Files.createDirectory(root.resolve("bin")))
      .tool("scala-cli", s"$record\n$packaging\n$scalaCli")
      .tool("protoc", record)
      .tool("go", record)
    val tools: Tools = stubs.tools(root, "TOOLS_LOG" -> log.toString)
    def ran: Seq[String] =
      if Files.exists(log) then Files.readString(log).linesIterator.toSeq else Nil
    def jar: Path = root.resolve("model/gen/ir-proto.jar")
    def stamp: String = Files.readString(root.resolve("model/gen/ir.stamp"))

  test("the command line is refused when it names an unknown or a contradictory flag"):
    def unreachable: Tools = fail("a refused command line runs no tool")
    for arguments <- Seq(
        Seq("--verbose"),
        Seq("--if-stale"),
        Seq("--generate-ir", "--update"),
        Seq("--generate-ir", "--skip-go-checks")
      )
    do assertEquals(gate(unreachable, arguments*), Answer(2, "", Gate.usage + "\n"))

  test("the IR's classes are packaged when the schema changed, and with --if-stale only then"):
    val repository = Repository()
    assertEquals(gate(repository.tools, "--generate-ir", "--if-stale").status, 0)
    assert(Files.isRegularFile(repository.jar))
    // The stamp is the SHA-256 of the schema's content.
    assertEquals(
      repository.stamp,
      "26695965cd692d9dce08efe0b2e1f745c3be7c19631b56873c8956b8d486cf17\n"
    )
    repository.ran match
      case Seq(protoc, packaged) =>
        assert(protoc.startsWith("protoc --proto_path=proto/internal --java_out="), protoc)
        assert(protoc.endsWith(" temporal/server/api/umpire/v1/ir.proto"), protoc)
        assert(packaged.startsWith("scala-cli --power package --library "), packaged)
        assert(packaged.contains(s" -f -o ${repository.jar} "), packaged)
      case ran => fail(s"protoc and scala-cli each run once, not $ran")

    assertEquals(gate(repository.tools, "--generate-ir", "--if-stale"), Answer(0, "", ""))
    assertEquals(repository.ran.size, 2, "a current jar is not packaged again")

    val before = repository.stamp
    Files.writeString(repository.schema, "syntax = \"proto3\";\nmessage Model {}\n")
    assertEquals(
      gate(repository.tools, "--generate-ir", "--if-stale").out,
      "generated model/gen/ir-proto.jar\n"
    )
    assertEquals(repository.ran.size, 4)
    assertNotEquals(repository.stamp, before)

    assertEquals(gate(repository.tools, "--generate-ir").status, 0)
    assertEquals(
      repository.ran.size,
      6,
      "without --if-stale the jar is packaged whatever its stamp"
    )

    Files.delete(repository.jar)
    assertEquals(gate(repository.tools, "--generate-ir", "--if-stale").status, 0)
    assertEquals(repository.ran.size, 8, "a missing jar is stale whatever its stamp")

  test("a missing schema fails with its path"):
    val repository = Repository()
    Files.delete(repository.schema)
    assertEquals(
      gate(repository.tools, "--generate-ir"),
      Answer(1, "", s"gate: the IR schema ${repository.schema} is missing\n")
    )

  test("the gate stops before it builds when a tool is missing, and names it"):
    for tool <- Seq("scala-cli", "protoc", "go") do
      val repository = Repository()
      Files.delete(repository.root.resolve("bin").resolve(tool))
      val answer = gate(repository.tools)
      assertEquals(answer.status, 1)
      assert(
        answer.err.startsWith(s"gate: $tool is missing: no directory of the PATH holds it"),
        answer.err
      )
      assertEquals(repository.ran, Nil)

  test("the gate stops at a build that prints an error and exits 0"):
    val werror =
      """case " $* " in *" test "*) printf '[\033[31merror\033[0m] ./model/umpire/Machine.scala:71:3\n';; esac"""
    val repository = Repository(scalaCli = werror)
    val answer = gate(repository.tools)
    assertEquals(answer.status, 1)
    assert(answer.err.startsWith("gate: scala-cli printed an error and exited 0 in "), answer.err)
    assert(answer.err.contains("[error] ./model/umpire/Machine.scala:71:3"), answer.err)
    // The framework alone compiled; the test of the Models was the last tool run.
    assertEquals(
      repository.ran.drop(2),
      Seq(
        "scala-cli compile model/project.scala model/umpire --suppress-outdated-dependency-warning",
        "scala-cli test model/project.scala model/umpire model/temporal --suppress-outdated-dependency-warning"
      )
    )

  test("the gate stops when the IR's Go code is older than its schema"):
    val repository = Repository()
    Files.setLastModifiedTime(
      repository.schema,
      FileTime.fromMillis(System.currentTimeMillis() + 60000)
    )
    val answer = gate(repository.tools)
    assertEquals(answer.status, 1)
    assertEquals(
      answer.err,
      s"gate: api/umpire/v1/ir.pb.go is older than the IR schema $schemaFile; run make protoc\n"
    )

  /** A checked-in tree and a produced one, each with the given files. */
  final private class Trees(checkedIn: Map[String, String], produced: Map[String, String]):
    val root: Path = Files.createTempDirectory("umpire-gate-trees")
    val tree: Path = Files.createDirectories(root.resolve("model/ir"))
    val lifted: Path = Files.createDirectories(root.resolve("model/gen/lifted"))
    checkedIn.foreach((name, text) => Files.writeString(tree.resolve(name), text))
    produced.foreach((name, text) => Files.writeString(lifted.resolve(name), text))
    // A write is seen by the time it leaves, so the files start in the past.
    checkedIn.keys.foreach(name =>
      Files.setLastModifiedTime(tree.resolve(name), FileTime.fromMillis(1000))
    )
    def held: Map[String, (String, Long)] = checkedIn.keySet
      .union(produced.keySet)
      .map(tree.resolve)
      .filter(Files.exists(_))
      .map(file =>
        file.getFileName.toString -> (
          Files.readString(file),
          Files.getLastModifiedTime(file).toMillis
        )
      )
      .toMap
    def settle(update: Boolean): Unit = Gate.settle(tree, lifted, root, update)

  test("a check passes on a current tree and writes nothing"):
    val trees = Trees(
      Map("a.json" -> "{}\n", "b.json" -> "[]\n"),
      Map("a.json" -> "{}\n", "b.json" -> "[]\n")
    )
    trees.settle(update = false)
    assertEquals(trees.held, Map("a.json" -> ("{}\n", 1000L), "b.json" -> ("[]\n", 1000L)))

  test(
    "a check names every stale, missing and left-over file, where it differs, and writes nothing"
  ):
    val trees = Trees(
      Map("a.json" -> "{\n  \"line\": 1\n}\n", "b.json" -> "[]\n", "old.json" -> "{}\n"),
      Map("a.json" -> "{\n  \"line\": 2\n}\n", "b.json" -> "[]\n", "new.json" -> "{}\n")
    )
    val before = trees.held
    val message = intercept[GateError](trees.settle(update = false)).getMessage
    assertEquals(
      message,
      s"""model/ir/a.json is stale: line 2 is `  "line": 1` and lifts to `  "line": 2`
         |model/ir/new.json is missing
         |model/ir/old.json is checked in and nothing produces it: remove it or name its roots
         |rerun with --update (make umpire-gen-model); the lifted files are in ${trees.lifted}""".stripMargin
    )
    assertEquals(trees.held, before)

  test("an update writes the files that changed and leaves the others as they are"):
    val trees = Trees(
      Map("a.json" -> "{}\n", "b.json" -> "[]\n"),
      Map("a.json" -> "{ }\n", "b.json" -> "[]\n", "c.json" -> "1\n")
    )
    trees.settle(update = true)
    val held = trees.held
    assertEquals(
      held.view.mapValues(_._1).toMap,
      Map("a.json" -> "{ }\n", "b.json" -> "[]\n", "c.json" -> "1\n")
    )
    assertEquals(held("b.json")._2, 1000L)
    trees.settle(update = false)

  test("an update writes nothing while a checked-in file is produced by nothing"):
    val trees = Trees(Map("a.json" -> "{}\n", "old.json" -> "{}\n"), Map("a.json" -> "{ }\n"))
    val before = trees.held
    val message = intercept[GateError](trees.settle(update = true)).getMessage
    assertEquals(
      message,
      "model/ir/old.json is checked in and nothing produces it: remove it or name its roots"
    )
    assertEquals(trees.held, before)
