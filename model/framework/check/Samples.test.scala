package framework.check

import java.nio.file.Files

class SamplesSuite extends munit.FunSuite:
  test("sample builds reuse across outputs and invalidate changed inputs and damaged outputs"):
    val root = Scratch.directory("framework-samples")
    val samples = Files.createDirectories(root.resolve("model/framework/samples"))
    for name <- Seq("Zebra.scala", "Alpha.scala", "Ignored.test.scala", "README.md") do
      Files.writeString(samples.resolve(name), "package samples\n"): Unit
    Files.createDirectories(root.resolve("model/framework/dsl")): Unit
    Files.createDirectories(root.resolve("model/framework/realize")): Unit
    Files.createDirectories(root.resolve("model/framework/export")): Unit
    val calls = root.resolve("calls")
    val stubs = Stubs(Scratch.directory("framework-sample-tools"))
      .tool(
        "scala-cli",
        s"""printf 'scala-cli %s\n' "$$*" >> '$calls'
           |if [ -f '$root/reject' ] && [ "$$1" = compile ]; then
           |  printf '[error] source rejected\n'; exit 0
           |fi
           |case "$$1" in
           |  version) if [ -f '$root/version' ]; then while IFS= read -r line; do printf '%s\n' "$$line"; done < '$root/version'; fi ;;
           |  compile)
           |    while [ "$$1" != '--compilation-output' ]; do shift; done
           |    shift; printf 'jar' > "$$1"
           |    printf '$root\n' ;;
           |  --power)
           |    while [ "$$1" != '-o' ]; do shift; done
           |    shift; printf 'lifter' > "$$1" ;;
           |  run)
           |    for argument in "$$@"; do
           |      case "$$argument" in *.json) printf '{"type":"framework.dsl.Step"}\n' > "$$argument" ;; esac
           |    done ;;
           |esac
           |""".stripMargin
      )
      .tool(
        "java",
        s"""printf 'java %s\n' "$$*" >> '$calls'
           |for argument in "$$@"; do
           |  case "$$argument" in *.json)
           |    printf '{"type":"framework.dsl.Step"}\n' > "$$argument"
           |    ;;
           |  esac
           |done""".stripMargin
      )
      .tool("go", "printf 'go %s\n' " + '"' + "$*" + '"' + s" >> '$calls'")
    val output = Files.createDirectories(root.resolve("output with spaces"))

    val checked = Samples.check(stubs.tools(root), output)

    assertEquals(checked.map(_.name), Vector("Alpha", "Zebra"))
    val commands = Files.readAllLines(calls).toArray.toVector.map(_.toString)
    assert(
      commands.filter(_.startsWith("scala-cli ")).forall(!_.contains("--server=false")),
      commands.mkString("\n")
    )
    assertEquals(commands.count(_.startsWith("scala-cli --power package ")), 1)
    assertEquals(commands.count(_.startsWith("scala-cli compile ")), 1)
    val cache = root.resolve("model/build/sample-check/samples")
    assert(commands.exists(_.contains(s"--compilation-output ${cache.resolve("samples.jar")}")))
    assertEquals(
      Files.readString(cache.resolve("samples.classpath")),
      s"$root\n"
    )
    assertEquals(commands.count(_.startsWith("java -Xmx3g -jar ")), 1)
    assert(commands.exists(_.contains("samples.Alpha$.queries")), commands.mkString("\n"))
    assert(commands.exists(_.contains("samples.Zebra$.queries")), commands.mkString("\n"))
    val check = commands.find(_.startsWith("go run ")).getOrElse(fail("sample check did not run"))
    assert(check.indexOf("Alpha.check.json") < check.indexOf("Zebra.check.json"), check)
    assert(
      Files.readString(output.resolve("check-samples.go")).contains("umpirespb.Query_FORM_VERIFY")
    )

    val warm = root.resolve("different output")
    Samples.check(stubs.tools(root), warm): Unit
    val again = Files.readString(calls)
    assertEquals(again.linesIterator.count(_.startsWith("scala-cli compile ")), 1)
    assertEquals(again.linesIterator.count(_.startsWith("scala-cli --power package ")), 1)
    assertEquals(again.linesIterator.count(_.startsWith("java -Xmx3g -jar ")), 2)
    assert(Files.isRegularFile(warm.resolve("Alpha.json")))

    def rebuild(compile: Boolean, pack: Boolean)(change: => Unit): Unit =
      val previous = Files.readString(calls)
      val compiles = previous.linesIterator.count(_.startsWith("scala-cli compile "))
      val packages = previous.linesIterator.count(_.startsWith("scala-cli --power package "))
      change
      Samples.check(stubs.tools(root), warm): Unit
      val log = Files.readString(calls)
      assertEquals(
        log.linesIterator.count(_.startsWith("scala-cli compile ")),
        compiles + (if compile then 1 else 0)
      )
      assertEquals(
        log.linesIterator.count(_.startsWith("scala-cli --power package ")),
        packages + (if pack then 1 else 0)
      )
    rebuild(true, false):
      Files.writeString(samples.resolve("Alpha.scala"), "package samples\nobject Alpha\n"): Unit
    rebuild(true, false):
      Files.writeString(samples.resolve("Beta.scala"), "package samples\n"): Unit
    rebuild(true, false):
      Files.delete(samples.resolve("Beta.scala"))
    rebuild(true, false):
      Files.writeString(root.resolve("model/project.scala"), "//> using scala 3.9.0\n"): Unit
    rebuild(true, false):
      Files.writeString(root.resolve("model/project.scala"), ""): Unit
    rebuild(true, false):
      Files.delete(root.resolve("model/project.scala"))
    rebuild(true, false):
      Files.writeString(root.resolve("model/build/api-scalapb.jar"), "new api"): Unit
    rebuild(false, true):
      Files.writeString(root.resolve("model/build/ir-scalapb.jar"), "new ir"): Unit
    rebuild(false, true):
      Files.writeString(root.resolve("model/framework/export/Lift.scala"), "changed lifter"): Unit
    rebuild(true, true):
      Files.writeString(root.resolve("version"), "another scala-cli version\n"): Unit
    rebuild(true, false):
      Files.delete(cache.resolve("samples.jar"))
    rebuild(true, false):
      Files.delete(cache.resolve("samples.classpath"))
    rebuild(true, false):
      Files.writeString(cache.resolve("samples.jar"), "corrupt cached output"): Unit
    rebuild(false, true):
      Files.delete(root.resolve("model/build/sample-check/lifter/lift-samples.jar"))
    rebuild(true, false):
      Files.writeString(
        cache.resolve("samples.classpath"),
        root.resolve("missing.jar").toString
      ): Unit
    rebuild(true, false):
      Files.writeString(cache.resolve("samples.classpath"), "invalid\u0000path"): Unit
    rebuild(true, false):
      Files.write(cache.resolve("stamp"), Array(0xff.toByte)): Unit
    rebuild(false, true):
      Files.write(root.resolve("model/build/sample-check/lifter/stamp"), Array(0xff.toByte)): Unit

    Files.writeString(samples.resolve("Alpha.scala"), "changed source\n"): Unit
    Files.writeString(root.resolve("reject"), ""): Unit
    for _ <- 1 to 2 do
      val error = intercept[ToolError](Samples.check(stubs.tools(root), warm))
      assert(error.getMessage.contains("source rejected"), error.getMessage)
      assert(!Files.exists(cache.resolve("stamp")), "a failed build cannot be reused")
    Files.delete(root.resolve("reject"))
    Samples.check(stubs.tools(root), warm): Unit
    assert(Files.isRegularFile(cache.resolve("stamp")))

  test("the sample lift driver preserves roots and partial outputs with source-significant paths"):
    val root = Scratch.directory("framework-sample-driver")
    val sources = Files.createDirectories(root.resolve("model/framework/samples"))
    Files.writeString(sources.resolve("Alpha.scala"), "package samples\n"): Unit
    Files.writeString(
      sources.resolve("Zebra.scala"),
      "package samples\nobject Zebra:\n  val advances: Progress[State] = claim\n"
    ): Unit
    Files.writeString(sources.resolve("Zulu.scala"), "package samples\n"): Unit
    val fake = root.resolve("FakeLift.scala")
    Files.writeString(
      fake,
      """package framework.`export`
        |import java.nio.file.{Files, Path, StandardOpenOption}
        |object Switches:
        |  def fromEnvironment: Set[String] = Set.empty
        |object LiftRun:
        |  def run(args: Seq[String], err: java.io.PrintStream, rules: Set[String]): Int =
        |    val channel = java.nio.channels.FileChannel.open(
        |      Path.of(sys.env("CACHE_LOCK")), java.nio.file.StandardOpenOption.WRITE)
        |    try
        |      val lock = channel.tryLock()
        |      if lock != null then
        |        lock.release()
        |        throw IllegalStateException("the cache lock was released before the lift")
        |    finally channel.close()
        |    val out = Path.of(args(2))
        |    Files.writeString(Path.of(sys.env("LIFT_LOG")),
        |      args.mkString("|\u0001|") + "\n", StandardOpenOption.CREATE,
        |      StandardOpenOption.APPEND)
        |    if out.getFileName.toString == "Zebra.json" && sys.env("LIFT_FAIL") == "refused" then
        |      err.println("lift: Zebra refused")
        |      1
        |    else if out.getFileName.toString == "Zebra.json" && sys.env("LIFT_FAIL") == "missing" then
        |      0
        |    else
        |      Files.writeString(out, "{\"type\":\"framework.dsl.Step\",\"roots\":[" +
        |        args.drop(3).map(r => "\"" + r + "\"").mkString(",") + "]}\n")
        |      0
        |""".stripMargin
    ): Unit
    val actual = Tools.here.find("scala-cli")
    val javaTool = Tools.here.find("java")
    val stubs = Stubs(Scratch.directory("framework-sample-driver-tools"))
      .tool(
        "scala-cli",
        s"""case "$$1" in
           |  compile)
           |    while [ "$$1" != '--compilation-output' ]; do shift; done
           |    shift; printf 'jar' > "$$1"; printf '$root\n'; exit 0 ;;
           |  version) printf 'stub version\n'; exit 0 ;;
           |esac
           |for argument in "$$@"; do
           |  case "$$argument" in *.scala) driver="$$argument" ;; esac
           |done
           |[ -n "$$driver" ] || { printf 'sample lift driver missing\n'; exit 1; }
           |while [ "$$1" != '-o' ]; do shift; done
           |shift
           |exec '$actual' --power package '$fake' "$$driver" --scala 3.9.0 --main-class liftSamples --assembly --preamble=false -f -o "$$1"
           |""".stripMargin
      )
      .tool("java", s"exec '$javaTool' \"$$@\"")
      .tool("go", "printf 'go must run only after all lifts pass\n' >> \"$GO_LOG\"")
    for failure <- Seq("none", "refused", "missing") do
      val output =
        Files.createDirectories(root.resolve(s"output $failure \"quoted\" \\path $$value"))
      for name <- Seq("Zebra", "Zulu") if failure != "none" do
        Files.writeString(output.resolve(name + ".json"), "stale IR"): Unit
        Files.writeString(output.resolve(name + ".check.json"), "stale adaptation"): Unit
      val calls = output.resolve("lifts.log")
      val go = output.resolve("go.log")
      val tools = Tools(
        root,
        sys.env ++ Map(
          "PATH" -> (stubs.directory.toString + java.io.File.pathSeparator + sys.env("PATH")),
          "LIFT_LOG" -> calls.toString,
          "GO_LOG" -> go.toString,
          "LIFT_FAIL" -> failure,
          "CACHE_LOCK" -> root.resolve("model/build/sample-check/lock").toString
        )
      )
      if failure != "none" then
        val error =
          if failure == "missing" then intercept[GateError](Samples.check(tools, output))
          else intercept[ToolError](Samples.check(tools, output))
        val message =
          if failure == "refused" then "lift: Zebra refused" else "the lift of Zebra wrote no"
        assert(error.getMessage.contains(message), error.getMessage)
        assert(!Files.exists(output.resolve("Zebra.json")))
        assert(!Files.exists(output.resolve("Zebra.check.json")))
        assert(!Files.exists(output.resolve("Zulu.json")))
        assert(!Files.exists(output.resolve("Zulu.check.json")))
        assert(!Files.exists(go))
      else
        assertEquals(Samples.check(tools, output).map(_.name), Vector("Alpha", "Zebra", "Zulu"))
        assert(Files.isRegularFile(go))
        assertEquals(
          Files.readString(output.resolve("Zebra.check.json")),
          "{\"type\":\"framework.Step\",\"roots\":[\"samples.Zebra$.queries\",\"samples.Zebra$.advances\"]}\n"
        )
      assertEquals(
        Files.readString(output.resolve("Alpha.check.json")),
        "{\"type\":\"framework.Step\",\"roots\":[\"samples.Alpha$.queries\"]}\n"
      )
      val requests = Files
        .readAllLines(calls)
        .toArray
        .toVector
        .map(_.toString.split("\\|\u0001\\|", -1).toVector)
      assertEquals(
        requests.map(_.take(3)),
        (if failure == "none" then Vector("Alpha", "Zebra", "Zulu") else Vector("Alpha", "Zebra"))
          .map(name =>
            Vector(
              s"${root.resolve("model/build/sample-check/samples/samples.jar")}=model/",
              root.resolve("model/build/sample-check/samples/samples.classpath").toString,
              output.resolve(name + ".json").toString
            )
          )
      )
      assertEquals(
        requests.map(_.drop(3)),
        Vector(
          Vector("samples.Alpha$.queries"),
          Vector("samples.Zebra$.queries", "samples.Zebra$.advances")
        ) ++ (if failure == "none" then Vector(Vector("samples.Zulu$.queries")) else Vector.empty)
      )

  test("exit-zero compiler errors stop sample compilation or the batch lift before Go"):
    for stage <- Seq("compile", "--power") do
      val root = Scratch.directory(s"framework-sample-$stage-error")
      val sources = Files.createDirectories(root.resolve("model/framework/samples"))
      Files.writeString(sources.resolve("Alpha.scala"), "package samples\n"): Unit
      val go = root.resolve("go.log")
      val stubs = Stubs(Scratch.directory("framework-sample-error-tools"))
        .tool(
          "scala-cli",
          s"""if [ "$$1" = '$stage' ]; then
             |  printf '[\\033[31merror\\033[0m] sample compilation rejected\n' >&2
             |else
             |  if [ "$$1" = 'compile' ]; then
             |    while [ "$$1" != '--compilation-output' ]; do shift; done
             |    shift; printf 'jar' > "$$1"; printf '$root\n'
             |  fi
             |fi
             |exit 0""".stripMargin
        )
        .tool("java", "printf 'stub java\n'")
        .tool("go", s"printf 'invoked\n' > '$go'")
      val error = intercept[ToolError](Samples.check(stubs.tools(root), root.resolve("output")))
      assert(error.getMessage.contains("printed an error and exited 0"), error.getMessage)
      assert(error.getMessage.contains("sample compilation rejected"), error.getMessage)
      assert(!Files.exists(go))
      val stageDirectory = if stage == "compile" then "samples" else "lifter"
      assert(!Files.exists(root.resolve(s"model/build/sample-check/$stageDirectory/stamp")))

  test("a successful tool that writes no build artifact cannot publish a cache stamp"):
    for stage <- Seq("compile", "--power", "classpath") do
      val root = Scratch.directory("framework-sample-incomplete-build")
      val sources = Files.createDirectories(root.resolve("model/framework/samples"))
      Files.writeString(sources.resolve("Alpha.scala"), "package samples\n"): Unit
      val stubs = Stubs(Scratch.directory("framework-sample-incomplete-tools"))
        .tool(
          "scala-cli",
          s"""if [ "$$1" = compile ]; then
             |  if [ '$stage' != compile ]; then
             |    while [ "$$1" != '--compilation-output' ]; do shift; done
             |    shift; printf 'jar' > "$$1"
             |  fi
             |  if [ '$stage' = classpath ]; then printf '$root/missing.jar\n'; else printf '$root\n'; fi
             |fi
             |""".stripMargin
        )
        .tool("java", "printf 'stub java\n'")
      for _ <- 1 to 2 do
        val error = intercept[GateError](Samples.check(stubs.tools(root), root.resolve("output")))
        val message = if stage == "classpath" then "invalid classpath" else "incomplete outputs"
        assert(error.getMessage.contains(message), error.getMessage)
        val stageDirectory = if stage == "--power" then "lifter" else "samples"
        assert(!Files.exists(root.resolve(s"model/build/sample-check/$stageDirectory/stamp")))

  test("a sample's progress claims are the Progress vals its object declares as members"):
    val source =
      """object Tanks:
        |  val fair = assume.fair(pump.run)
        |  val drains: Progress[Tank] = Model.leadsTo("drains")(_.full, _.empty, within = 2, fair)
        |  object Model extends Machine[Tank, Outcome, Nothing]:
        |    val nested: Progress[Tank] = leadsTo("nested")(_.full, _.empty, within = 1)
        |  val fills: Progress[Tank] =
        |    Model.leadsTo("fills")(_.empty, _.full, within = 2, fair)
        |""".stripMargin
    assertEquals(Samples.progressClaims(source), Vector("drains", "fills"))
