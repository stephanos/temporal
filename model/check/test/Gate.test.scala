package umpire.check

import java.io.{ByteArrayOutputStream, PrintStream}
import java.nio.file.{Files, Path}
import java.nio.file.attribute.FileTime

class GateSuite extends munit.FunSuite:
  private val schemaFile = "proto/internal/temporal/server/api/umpire/v1/ir.proto"

  // What a run of the gate's command line answered: its status and what it printed.
  final private case class Answer(status: Int, out: String, err: String)
  private def gate(tools: => Tools, arguments: String*): Answer =
    val (out, err) = (ByteArrayOutputStream(), ByteArrayOutputStream())
    val status = Gate.main(arguments, tools, PrintStream(out), PrintStream(err))
    Answer(status, out.toString, err.toString)

  // A repository with an IR schema and its generated Go code, and stand-ins for the tools: each
  // records its command line in `log`, and scala-cli writes the file it is asked to package, as an
  // executable, since one of them is protoc's plugin.
  final private class Repository(scalaCli: String = "", go: String = passingVocabulary):
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
    Files.createDirectories(root.resolve("model/build"))
    val apiDescriptor: Path = root.resolve("proto/api.binpb")
    Files.writeString(apiDescriptor, "api descriptors")
    Files.writeString(root.resolve("go.mod"), "module example\n")
    Files.writeString(root.resolve("go.sum"), "")
    val getproto: Path = root.resolve("cmd/tools/getproto/main.go")
    Files.createDirectories(getproto.getParent)
    Files.writeString(getproto, "package main\n")
    Files.writeString(getproto.getParent.resolve("files.go"), "package main\n")
    val gateSource: Path = root.resolve("model/check/Gate.scala")
    Files.createDirectories(gateSource.getParent)
    Files.writeString(gateSource, "package umpire.check\n")
    Files.writeString(gateSource.getParent.resolve("project.scala"), "//> using scala 3.9.0\n")
    val testpilot = root.resolve("proto/internal/temporal/server/api/testpilot/v1/case.proto")
    Files.createDirectories(testpilot.getParent)
    Files.writeString(testpilot, "syntax = \"proto3\";\n")
    private val record = "echo \"${0##*/} $*\" >> \"$TOOLS_LOG\""
    private val packaging =
      "out=; previous=; for a in \"$@\"; do [ \"$previous\" = -o ] && out=$a; previous=$a; done; [ -n \"$out\" ] && : > \"$out\" && /bin/chmod +x \"$out\""
    private val stubs = Stubs(Files.createDirectory(root.resolve("bin")))
      .tool("scala-cli", s"$record\n$packaging\n$scalaCli\ntrue")
      .tool(
        "protoc",
        s"$record\ncase \" $$* \" in *\" --descriptor_set_out=\"*) for a in \"$$@\"; do case \"$$a\" in --descriptor_set_out=*) printf current > \"$${a#*=}\";; esac; done;; esac"
      )
      .tool("make", record)
      .tool(
        "go",
        s"$record\ncase \" $$* \" in *\" --linked-api \"*) out=; previous=; for a in \"$$@\"; do [ \"$$previous\" = --out ] && out=$$a; previous=$$a; done; printf 'linked' > \"$$out\"; printf 'temporal/api/workflowservice/v1/service.proto\\ntemporal/server/api/testpilot/v1/case.proto\\n'; exit 0;; esac\n$go"
      )
    val tools: Tools = stubs.tools(root, "TOOLS_LOG" -> log.toString)
    def ran: Seq[String] =
      if Files.exists(log) then Files.readString(log).linesIterator.toSeq else Nil
    def jar: Path = root.resolve("model/build/ir-scalapb.jar")
    def plugin: Path = root.resolve("model/build/protoc-gen-scala")
    def stampFile: Path = root.resolve("model/build/ir.stamp")
    def stamp: String = Files.readString(stampFile)
    def apiJar: Path = root.resolve("model/build/api-scalapb.jar")
    def apiStampFile: Path = root.resolve("model/build/api.stamp")
    def apiStamp: String = Files.readString(apiStampFile)

  // As `go test -v -run` of the vocabulary check, go prints what the test's run printed.
  private def vocabulary(printed: String, exit: Int = 0) =
    s"""case " $$* " in *" -run "*) printf '%b\\n' '$printed'; exit $exit;; esac"""
  private def passingVocabulary =
    vocabulary("--- PASS: TestModelNamesNoRetiredFrontEnd (0.05s)\\nPASS")
  private val vocabularyCheck =
    "go test -count=1 -tags test_dep -v -run ^TestModelNamesNoRetiredFrontEnd$ ./tools/umpire/ir"
  private val lintRun = "go run ./tools/umpire/cmd/umpire-lint"

  test("the command line is refused when it names an unknown or a contradictory flag"):
    def unreachable: Tools = fail("a refused command line runs no tool")
    for arguments <- Seq(
        Seq("--verbose"),
        Seq("--if-stale"),
        Seq("--generate-ir", "--update"),
        Seq("--generate-ir", "--skip-go-checks"),
        Seq("--check-syntax", "--update"),
        Seq("--check-syntax", "--generate-ir"),
        Seq("--check-syntax", "--check-comments"),
        Seq("--check-comments", "--update")
      )
    do assertEquals(gate(unreachable, arguments*), Answer(2, "", Gate.usage + "\n"))

  test("the IR's classes are packaged when the schema changed, and with --if-stale only then"):
    val repository = Repository()
    assertEquals(gate(repository.tools, "--generate-ir", "--if-stale").status, 0)
    assert(Files.isRegularFile(repository.jar))
    // The stamp is the SHA-256 of the schema's content and the versions of the generator and Scala.
    assertEquals(
      repository.stamp,
      "26695965cd692d9dce08efe0b2e1f745c3be7c19631b56873c8956b8d486cf17 scalapb:0.11.20 scala:3.9.0\n"
    )
    repository.ran match
      case Seq(plugin, protoc, packaged) =>
        assert(plugin.startsWith("scala-cli --power package "), plugin)
        assert(
          plugin.contains(
            " --scala 3.9.0 --dep com.thesamet.scalapb::compilerplugin:0.11.20" +
              s" --main-class scalapb.ScalaPbCodeGenerator -f -o ${repository.plugin} "
          ),
          plugin
        )
        assert(
          protoc.startsWith(
            s"protoc --plugin=protoc-gen-scala=${repository.plugin} --proto_path=proto/internal" +
              " --scala_out=flat_package,scala3_sources:"
          ),
          protoc
        )
        assert(protoc.endsWith(" temporal/server/api/umpire/v1/ir.proto"), protoc)
        assert(packaged.startsWith("scala-cli --power package --library "), packaged)
        assert(packaged.contains(" --scala 3.9.0 "), packaged)
        assert(
          packaged.contains(" --dep com.thesamet.scalapb::scalapb-runtime:0.11.20 "),
          packaged
        )
        assert(packaged.contains(s" -f -o ${repository.jar} "), packaged)
      case ran => fail(s"the plugin, protoc and the jar each run once, not $ran")

    assertEquals(gate(repository.tools, "--generate-ir", "--if-stale"), Answer(0, "", ""))
    assertEquals(repository.ran.size, 3, "a current jar is not packaged again")

    val before = repository.stamp
    Files.writeString(repository.schema, "syntax = \"proto3\";\nmessage Model {}\n")
    assertEquals(
      gate(repository.tools, "--generate-ir", "--if-stale").out,
      "generated model/build/ir-scalapb.jar\n"
    )
    assertEquals(repository.ran.size, 6)
    assertNotEquals(repository.stamp, before)

    assertEquals(gate(repository.tools, "--generate-ir").status, 0)
    assertEquals(
      repository.ran.size,
      9,
      "without --if-stale the jar is packaged whatever its stamp"
    )

    Files.delete(repository.jar)
    assertEquals(gate(repository.tools, "--generate-ir", "--if-stale").status, 0)
    assertEquals(repository.ran.size, 12, "a missing jar is stale whatever its stamp")

  test("the IR's classes are packaged again when a generator's version changed"):
    val repository = Repository()
    assertEquals(gate(repository.tools, "--generate-ir", "--if-stale").status, 0)
    Files.writeString(
      repository.stampFile,
      repository.stamp.replace("scalapb:0.11.20", "scalapb:0.11.19")
    )
    assertEquals(gate(repository.tools, "--generate-ir", "--if-stale").status, 0)
    val again = repository.ran.drop(3)
    assertEquals(again.size, 3, "a stamp of another generator is stale")
    for (ran, tool) <- again.zip(Seq("scala-cli", "protoc --plugin", "scala-cli")) do
      assert(ran.startsWith(tool), s"the plugin and the jar are packaged again: $again")
    assert(repository.stamp.endsWith(" scalapb:0.11.20 scala:3.9.0\n"), repository.stamp)

  test("a plugin scala-cli did not package is named, and nothing is generated with it"):
    // scala-cli exits 0 and leaves no launcher behind.
    val repository =
      Repository(scalaCli = "case \" $* \" in *\" --main-class \"*) /bin/rm -f \"$out\";; esac")
    assertEquals(
      gate(repository.tools, "--generate-ir"),
      Answer(
        1,
        "",
        "gate: the ScalaPB plugin model/build/protoc-gen-scala is missing: " +
          "scala-cli did not package it\n"
      )
    )
    assertEquals(repository.ran.size, 1, "protoc does not run without the plugin")
    assert(!Files.exists(repository.stampFile), "a failed generation leaves no stamp")

    // A launcher of an earlier run is not taken for one scala-cli packaged now.
    val earlier = Repository()
    assertEquals(gate(earlier.tools, "--generate-ir").status, 0)
    val again =
      Repository(scalaCli = "case \" $* \" in *\" --main-class \"*) /bin/rm -f \"$out\";; esac")
    Files.createDirectories(again.plugin.getParent)
    Files.copy(earlier.plugin, again.plugin)
    assertEquals(
      gate(again.tools, "--generate-ir"),
      Answer(
        1,
        "",
        "gate: the ScalaPB plugin model/build/protoc-gen-scala is missing: " +
          "scala-cli did not package it\n"
      )
    )
    assertEquals(again.ran.size, 1, "protoc does not run with an earlier run's plugin")

  test("a jar scala-cli did not package fails with the jar and the schema named"):
    // scala-cli exits 0 without writing the jar.
    val missing =
      Repository(scalaCli = "case \"$out\" in *ir-scalapb.jar) /bin/rm -f \"$out\";; esac")
    assertEquals(
      gate(missing.tools, "--generate-ir"),
      Answer(
        1,
        "",
        s"gate: model/build/ir-scalapb.jar is missing: the IR schema $schemaFile was not packaged\n"
      )
    )
    assert(!Files.exists(missing.stampFile), "a failed generation leaves no stamp")

    // A jar of an earlier schema is not taken for one scala-cli packaged now.
    val stale = Repository()
    assertEquals(gate(stale.tools, "--generate-ir").status, 0)
    Files.writeString(stale.schema, "syntax = \"proto3\";\nmessage Model {}\n")
    Files.writeString(
      stale.root.resolve("bin/scala-cli"),
      "\ncase \"$out\" in *ir-scalapb.jar) /bin/rm -f \"$out\";; esac\n",
      java.nio.file.StandardOpenOption.APPEND
    )
    assertEquals(
      gate(stale.tools, "--generate-ir", "--if-stale"),
      Answer(
        1,
        "",
        s"gate: model/build/ir-scalapb.jar is missing: the IR schema $schemaFile was not packaged\n"
      )
    )
    assert(!Files.exists(stale.jar), "the earlier jar is gone, not kept as current")

  test("a missing schema fails with its path"):
    val repository = Repository()
    Files.delete(repository.schema)
    assertEquals(
      gate(repository.tools, "--generate-ir"),
      Answer(1, "", s"gate: the IR schema ${repository.schema} is missing\n")
    )

  test("the API jar is built once for descriptors and tools, and survives a Model edit"):
    val repository = Repository()
    val initial = gate(repository.tools, "--generate-api", "--if-stale")
    assertEquals(initial.status, 0, initial.err)
    assert(Files.isRegularFile(repository.apiJar))
    assert(
      repository.ran.exists(line =>
        line.contains("--descriptor_set_out=") &&
          line.contains("temporal/server/api/testpilot/v1/case.proto")
      )
    )
    assert(repository.ran.exists(_.contains("--current-internal")))
    val stamp = repository.apiStamp
    val jarModified = Files.getLastModifiedTime(repository.apiJar)
    val ran = repository.ran.size
    Files.createDirectories(repository.root.resolve("model/temporal"))
    Files.writeString(repository.root.resolve("model/temporal/Model.scala"), "object Model\n")
    val unchanged = gate(repository.tools, "--generate-api", "--if-stale")
    assertEquals(unchanged.status, 0, unchanged.err)
    assertEquals(
      repository.ran.size,
      ran + 5,
      "only make, current protos, linked closure and version probes run"
    )
    assertEquals(repository.apiStamp, stamp)
    assertEquals(Files.getLastModifiedTime(repository.apiJar), jarModified)

    Files.writeString(repository.apiDescriptor, "changed descriptors")
    val changed = gate(repository.tools, "--generate-api", "--if-stale")
    assertEquals(changed.status, 0, changed.err)
    assertNotEquals(repository.apiStamp, stamp)
    assertEquals(repository.ran.count(_.contains("--linked-api")), 3)

    val closureStamp = repository.apiStamp
    val goStub = repository.root.resolve("bin/go")
    Files.writeString(
      goStub,
      Files.readString(goStub).replace("printf 'linked' >", "printf 'linked2' >")
    )
    assertEquals(gate(repository.tools, "--generate-api", "--if-stale").status, 0)
    assertNotEquals(repository.apiStamp, closureStamp)

    val next = repository.apiStamp
    Files.writeString(repository.getproto, "package main // changed\n")
    assertEquals(gate(repository.tools, "--generate-api", "--if-stale").status, 0)
    assertNotEquals(repository.apiStamp, next)

    val sourceStamp = repository.apiStamp
    Files.writeString(repository.gateSource, "package umpire.check // generation changed\n")
    val builds = repository.ran.count(_.contains("--scala_out=flat_package,scala3_sources,grpc"))
    assertEquals(gate(repository.tools, "--generate-api", "--if-stale").status, 0)
    assertNotEquals(repository.apiStamp, sourceStamp)
    assertEquals(
      repository.ran.count(_.contains("--scala_out=flat_package,scala3_sources,grpc")),
      builds + 1
    )
    assertEquals(gate(repository.tools, "--generate-api", "--if-stale").status, 0)
    assertEquals(
      repository.ran.count(_.contains("--scala_out=flat_package,scala3_sources,grpc")),
      builds + 1
    )

    val toolStamp = repository.apiStamp
    Files.writeString(
      repository.root.resolve("bin/protoc"),
      "\ncase \" $* \" in *\" --version \"*) echo libprotoc-30;; esac\n",
      java.nio.file.StandardOpenOption.APPEND
    )
    assertEquals(gate(repository.tools, "--generate-api", "--if-stale").status, 0)
    assertNotEquals(repository.apiStamp, toolStamp)

    val options = repository.apiStamp.replace("flat_package,scala3_sources,grpc", "flat_package")
    Files.writeString(repository.apiStampFile, options)
    assertEquals(gate(repository.tools, "--generate-api", "--if-stale").status, 0)
    assert(repository.apiStamp.contains("flat_package,scala3_sources,grpc"))

  test("the API generation names missing and stale descriptor remedies"):
    val missing = Repository()
    Files.delete(missing.apiDescriptor)
    val absent = gate(missing.tools, "--generate-api", "--if-stale")
    assertEquals(absent.status, 1)
    assert(absent.err.contains("make proto/api.binpb"), absent.err)

    val stale = Repository()
    Files.writeString(stale.root.resolve("bin/make"), "#!/bin/sh\nexit 1\n")
    val old = gate(stale.tools, "--generate-api", "--if-stale")
    assertEquals(old.status, 1)
    assert(old.err.contains("make proto/api.binpb"), old.err)

  test("failed API packaging cannot leave a current stamp or accept an old jar"):
    val repository = Repository()
    val initial = gate(repository.tools, "--generate-api", "--if-stale")
    assertEquals(initial.status, 0, initial.err)
    Files.writeString(repository.apiDescriptor, "changed descriptors")
    Files.writeString(
      repository.root.resolve("bin/scala-cli"),
      "\ncase \"$out\" in *api-scalapb.jar) /bin/rm -f \"$out\";; esac\n",
      java.nio.file.StandardOpenOption.APPEND
    )
    val failed = gate(repository.tools, "--generate-api", "--if-stale")
    assertEquals(failed.status, 1)
    assert(failed.err.contains("model/build/api-scalapb.jar"), failed.err)
    assert(!Files.isRegularFile(repository.apiStampFile))

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
    // The framework alone compiled; the test of the Models was the last build run.
    assertEquals(
      repository.ran.filter(_.startsWith("scala-cli ")).takeRight(2),
      Seq(
        "scala-cli compile model/project.scala model/umpire --suppress-outdated-dependency-warning",
        "scala-cli test --server=false model/project.scala model/umpire model/temporal --suppress-outdated-dependency-warning"
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

  // The IR files the stand-in lifter writes, as the Models declare them with `irFile`.
  private val irFiles = Seq(
    "nexus-workflow.json",
    "nexus-workflow-control.json",
    "activity-standalone.json",
    "activity-system.json",
    "activity-race.json",
    "nexus-workflow-close.json"
  )

  // As the lifter's one run (`lift --ir <jars> <classpath> <directory>`), scala-cli writes every IR
  // file into the directory its fourth argument names, or, when FAILING_LIFTS names some of them,
  // refuses each under a line naming it and writes none; it also records the update its test of the
  // lifter was given.
  private val lifting =
    s"""case " $$* " in *" test model/irgen "*) echo "lifter update=[$$UMPIRE_LIFTER_UPDATE]" >> "$$TOOLS_LOG";; esac
      |program=; n=0
      |for a in "$$@"; do
      |  if [ -n "$$program" ]; then
      |    n=$$((n + 1))
      |    if [ $$n -eq 4 ]; then
      |      if [ -n "$$FAILING_LIFTS" ]; then
      |        for f in $$FAILING_LIFTS; do echo "lift: the roots of $$f did not lift:"; echo "lift: $$f: no IR form"; done
      |        exit 1
      |      fi
      |      for f in ${irFiles.mkString(" ")}; do echo '{"lifted": true}' > "$$a/$$f"; done
      |    fi
      |  fi
      |  [ "$$a" = -- ] && program=yes
      |done
      |exit 0""".stripMargin

  private def checkedIn(repository: Repository): Map[String, String] =
    irFiles
      .map(file => file -> Files.readString(repository.root.resolve("model/ir").resolve(file)))
      .toMap

  // A repository whose model/ir holds every file the gate lifts, each of them stale.
  private def staleRepository(): Repository =
    val repository = Repository(scalaCli = lifting)
    val ir = Files.createDirectories(repository.root.resolve("model/ir"))
    irFiles.foreach(file => Files.writeString(ir.resolve(file), "{}\n"))
    repository

  test("a check of a stale tree fails, names every stale file and rewrites nothing"):
    val repository = staleRepository()
    val answer =
      gate(repository.tools.withEnvironment("UMPIRE_LIFTER_UPDATE" -> "1"), "--skip-go-checks")
    assertEquals(answer.status, 1)
    for file <- irFiles do
      assert(answer.err.contains(s"model/ir/$file is stale: line 1 is `{}`"), answer.err)
    assert(answer.err.contains("rerun with --update (make umpire-gen-model)"), answer.err)
    assertEquals(checkedIn(repository).values.toSet, Set("{}\n"))
    // An update the environment asks for does not reach a check's test of the lifter.
    assert(repository.ran.contains("lifter update=[]"), repository.ran.mkString("\n"))
    // A check lowers the checked-in IR beside the build, so its Cases were checked too; the IR is
    // not linted while it is stale.
    assertEquals(
      repository.ran.filter(line => line.startsWith("go ") && !line.contains("--linked-api")),
      Seq(vocabularyCheck, "go run ./tools/umpire/cmd/umpire-gen-cases")
    )

  test(
    "an update rewrites the stale tree, passes the update on, and the check that follows passes"
  ):
    val repository = staleRepository()
    assertEquals(gate(repository.tools, "--update").status, 0)
    assertEquals(checkedIn(repository).values.toSet, Set("{\"lifted\": true}\n"))
    assert(repository.ran.contains("lifter update=[1]"), repository.ran.mkString("\n"))
    assertEquals(
      repository.ran.filter(line => line.startsWith("go ") && !line.contains("--linked-api")),
      Seq(
        vocabularyCheck,
        "go run ./tools/umpire/cmd/umpire-gen-cases --update",
        s"$lintRun --update",
        "go vet -tags test_dep ./tools/umpire/...",
        "go test -count=1 -tags test_dep ./tools/umpire/..."
      )
    )
    val check = gate(repository.tools, "--skip-go-checks")
    assertEquals(check.status, 0, check.err)
    // A check that skips the Go checks still holds the vocabulary, and lints the IR it settled.
    assertEquals(
      repository.ran
        .filter(line => line.startsWith("go ") && !line.contains("--linked-api"))
        .takeRight(3),
      Seq(vocabularyCheck, "go run ./tools/umpire/cmd/umpire-gen-cases", lintRun)
    )
    assert(check.out.contains("== lint every IR file and print its coverage"), check.out)

  // A repository whose model/ir is current, so only the vocabulary check can stop the gate.
  private def currentRepository(go: String): Repository =
    val repository = Repository(scalaCli = lifting, go = go)
    val ir = Files.createDirectories(repository.root.resolve("model/ir"))
    irFiles.foreach(file => Files.writeString(ir.resolve(file), "{\"lifted\": true}\n"))
    repository

  test("package-only Model headers use the normal compiler, without ignoring compiler errors"):
    val headerCompiler =
      """case " $* " in *" package "*)
        |  case "$out" in */model-scala.jar)
        |    case " $* " in *" --server=false "*) ;; *) echo '[error] No input files provided'; exit 0;; esac
        |  ;; esac
        |;; esac
        |case " $* " in *" test model/project.scala model/umpire model/temporal "*)
        |  case " $* " in *" --server=false "*) ;; *) echo '[error] No class, trait or object is defined in the compilation unit.'; exit 0;; esac
        |;; esac
        |case " $* " in *" compile --print-class-path model/project.scala model/umpire model/temporal "*)
        |  case " $* " in *" --server=false "*) ;; *) echo '[error] No class, trait or object is defined in the compilation unit.'; exit 0;; esac
        |;; esac
        |""".stripMargin
    val repository = Repository(scalaCli = headerCompiler + lifting)
    val ir = Files.createDirectories(repository.root.resolve("model/ir"))
    irFiles.foreach(file => Files.writeString(ir.resolve(file), "{\"lifted\": true}\n"))
    val answer = gate(repository.tools, "--skip-go-checks")
    assertEquals(answer.status, 0, answer.err)
    val packaged = repository.ran.find(_.contains("model-scala.jar")).get
    assert(packaged.contains(" --server=false "), packaged)
    assert(
      repository.ran
        .find(line => line.contains(" test ") && line.contains("model/temporal"))
        .get
        .contains(" --server=false ")
    )
    assert(repository.ran.find(_.contains(" --print-class-path ")).get.contains(" --server=false "))

  test("the gate stops at a mention the vocabulary check finds, and shows where"):
    val found = "model/umpire/Table.scala:7\\n--- FAIL: TestModelNamesNoRetiredFrontEnd (0.05s)"
    val repository = currentRepository(vocabulary(found, exit = 1))
    val answer = gate(repository.tools, "--skip-go-checks")
    assertEquals(answer.status, 1)
    assert(answer.err.startsWith("gate: go exited 1 in "), answer.err)
    assert(answer.err.contains("model/umpire/Table.scala:7"), answer.err)
    // It is the first thing the gate runs: nothing was built or lifted before it.
    assertEquals(repository.ran, Seq(vocabularyCheck))

  test("the gate stops when the vocabulary check did not run"):
    val unmatched = "ok  \\tgo.temporal.io/server/tools/umpire/ir\\t0.4s [no tests to run]"
    val repository = currentRepository(vocabulary(unmatched))
    val answer = gate(repository.tools, "--skip-go-checks")
    assertEquals(answer.status, 1)
    assert(
      answer.err.startsWith(
        "gate: the vocabulary check TestModelNamesNoRetiredFrontEnd did not run:\nok  "
      ),
      answer.err
    )
    assertEquals(repository.ran.filter(_.startsWith("go ")), Seq(vocabularyCheck))

  test("the gate rejects a proto literal in a Model before generating classes"):
    val repository = currentRepository(passingVocabulary)
    val model = repository.root.resolve("model/temporal/Model.scala")
    Files.createDirectories(model.getParent)
    Files.writeString(
      model,
      "val method = \"/temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution\"\n"
    )
    val answer = gate(repository.tools, "--skip-go-checks")
    assertEquals(answer.status, 1)
    assert(answer.err.contains("model/temporal/Model.scala:1: method is free text"), answer.err)
    assertEquals(repository.ran, Seq(vocabularyCheck))

  test("the one lift reports every IR file that failed, each by name, and nothing is rewritten"):
    val repository = staleRepository()
    val failing =
      repository.tools.withEnvironment(
        "FAILING_LIFTS" -> "activity-standalone.json nexus-workflow-close.json"
      )
    val answer = gate(failing, "--update", "--skip-go-checks")
    assertEquals(answer.status, 1)
    assert(
      answer.err.contains(
        "the Models' IR files did not lift:\nlift: the roots of activity-standalone.json did not lift:\n" +
          "lift: activity-standalone.json: no IR form"
      ),
      answer.err
    )
    assert(
      answer.err.contains("lift: the roots of nexus-workflow-close.json did not lift:"),
      answer.err
    )
    assert(!answer.err.contains("nexus-workflow.json"), answer.err)
    // The gate names no root: the lifter finds the IR files the Models declare.
    val lifts = repository.ran.filter(_.contains("--main-class umpire.irgen.lift"))
    assertEquals(lifts.size, 1, lifts.mkString("\n"))
    assert(
      lifts.head.matches(
        """scala-cli run model/irgen --main-class umpire.irgen.lift .*-- --ir \S+=model/ \S+ \S+"""
      ),
      lifts.head
    )
    assertEquals(checkedIn(repository).values.toSet, Set("{}\n"))

  test("a check reports Cases that fail with their output, with every other failure"):
    val cases =
      s"""case " $$* " in *"/umpire-gen-cases "*) echo "model/cases/a.json is stale"; exit 1;; esac
         |$passingVocabulary""".stripMargin
    val current = currentRepository(cases)
    val answer = gate(current.tools, "--skip-go-checks")
    assertEquals(answer.status, 1)
    assert(answer.err.startsWith("gate: go exited 1 in "), answer.err)
    assert(answer.err.contains("model/cases/a.json is stale"), answer.err)
    // The lifts beside it ended and were reported before the Cases.
    assert(
      answer.out.indexOf("== lift every IR file the Models declare") <
        answer.out.indexOf("== check every lowered Case"),
      answer.out
    )
    val stale = Repository(scalaCli = lifting, go = cases)
    val ir = Files.createDirectories(stale.root.resolve("model/ir"))
    irFiles.foreach(file => Files.writeString(ir.resolve(file), "{}\n"))
    val both = gate(stale.tools, "--skip-go-checks")
    assertEquals(both.status, 1)
    assert(both.err.contains("model/ir/activity-standalone.json is stale"), both.err)
    assert(both.err.contains("model/cases/a.json is stale"), both.err)

  test("a check lints the settled IR before the Go checks, and a failing lint fails the gate"):
    val current = currentRepository(passingVocabulary)
    val check = gate(current.tools)
    assertEquals(check.status, 0, check.err)
    assertEquals(
      current.ran.filter(line => line.startsWith("go ") && !line.contains("--linked-api")),
      Seq(
        vocabularyCheck,
        "go run ./tools/umpire/cmd/umpire-gen-cases",
        lintRun,
        "go vet -tags test_dep ./tools/umpire/...",
        "go test -count=1 -tags test_dep ./tools/umpire/..."
      )
    )
    val lint =
      s"""case " $$* " in *"/umpire-lint "*) echo "umpire-lint: 1 unaccepted"; exit 1;; esac
         |$passingVocabulary""".stripMargin
    for arguments <- Seq(Seq(), Seq("--skip-go-checks"), Seq("--update", "--skip-go-checks")) do
      val failing = currentRepository(lint)
      val answer = gate(failing.tools, arguments*)
      assertEquals(answer.status, 1, arguments.mkString(" "))
      assert(answer.err.startsWith("gate: go exited 1 in "), answer.err)
      // An update forwards the law waivers into the accepted findings as it lints.
      val ranLint = if arguments.contains("--update") then s"$lintRun --update" else lintRun
      assert(answer.err.endsWith(s"$ranLint\n"), answer.err)
      // The Go checks follow the lint, and do not run after it failed.
      assertEquals(
        failing.ran.filter(line => line.startsWith("go ") && !line.contains("--linked-api")).last,
        ranLint
      )

  test("a file the gate cannot read fails with its path, not a stack trace"):
    val repository = Repository(scalaCli = lifting)
    val answer = gate(repository.tools, "--skip-go-checks")
    assertEquals(answer.status, 1)
    assertEquals(
      answer.err,
      s"gate: java.nio.file.NoSuchFileException: ${repository.root.resolve("model/ir")}\n"
    )

  // A checked-in tree and a produced one, each with the given files.
  final private class Trees(checkedIn: Map[String, String], produced: Map[String, String]):
    val root: Path = Files.createTempDirectory("umpire-gate-trees")
    val tree: Path = Files.createDirectories(root.resolve("model/ir"))
    val lifted: Path = Files.createDirectories(root.resolve("model/build/lifted"))
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
         |model/ir/old.json is checked in and nothing produces it: remove it or declare it with irFile
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

  test("accepted lint findings beside an IR file are no orphan, in a check and in an update"):
    val accepted = "{\"accepted\": []}\n"
    val trees = Trees(
      Map("a.json" -> "{}\n", "a.lint.json" -> accepted, "gone.lint.json" -> accepted),
      Map("a.json" -> "{}\n")
    )
    trees.settle(update = false)
    trees.settle(update = true)
    assertEquals(
      trees.held.view.mapValues(_._1).toMap,
      Map("a.json" -> "{}\n", "a.lint.json" -> accepted, "gone.lint.json" -> accepted)
    )
    // Every other file is still held to what was produced.
    val left = Trees(
      Map("a.json" -> "{}\n", "a.lint.json" -> accepted, "a.laws.json" -> "{}\n"),
      Map("a.json" -> "{}\n")
    )
    assertEquals(
      intercept[GateError](left.settle(update = true)).getMessage,
      "model/ir/a.laws.json is checked in and nothing produces it: " +
        "remove it or declare it with irFile"
    )

  test("an update writes nothing while a checked-in file is produced by nothing"):
    val trees = Trees(Map("a.json" -> "{}\n", "old.json" -> "{}\n"), Map("a.json" -> "{ }\n"))
    val before = trees.held
    val message = intercept[GateError](trees.settle(update = true)).getMessage
    assertEquals(
      message,
      "model/ir/old.json is checked in and nothing produces it: remove it or declare it with irFile"
    )
    assertEquals(trees.held, before)
