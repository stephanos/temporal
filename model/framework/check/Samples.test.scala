package framework.check

import java.nio.file.Files

class SamplesSuite extends munit.FunSuite:
  test("sample checking discovers every source and compiles, exports and checks it"):
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
           |case "$$1" in
           |  compile) printf '/stand-in/classes:/a dependency/transitive.jar\n' ;;
           |esac
           |for argument in "$$@"; do
           |  case "$$argument" in *.json)
           |    printf '{"type":"framework.dsl.Step"}\n' > "$$argument"
           |    printf '{"type":"framework.Step"}\n' > "$${argument%.json}.check.json" ;;
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
    assertEquals(commands.count(_.startsWith("scala-cli --power package ")), 0)
    assertEquals(commands.count(_.startsWith("scala-cli compile ")), 1)
    assert(commands.head.contains(s"--compilation-output ${output.resolve("samples.jar")}"))
    assertEquals(
      Files.readString(output.resolve("samples.classpath")),
      "/stand-in/classes:/a dependency/transitive.jar\n"
    )
    assertEquals(commands.count(_.startsWith("scala-cli run ")), 1)
    assert(commands.exists(_.contains("samples.Alpha$.queries")), commands.mkString("\n"))
    assert(commands.exists(_.contains("samples.Zebra$.queries")), commands.mkString("\n"))
    val check = commands.find(_.startsWith("go run ")).getOrElse(fail("sample check did not run"))
    assert(check.indexOf("Alpha.check.json") < check.indexOf("Zebra.check.json"), check)
    assertEquals(
      Files.readString(output.resolve("Alpha.check.json")),
      "{\"type\":\"framework.Step\"}\n"
    )
    assert(
      Files.readString(output.resolve("check-samples.go")).contains("umpirespb.Query_FORM_VERIFY")
    )

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
        |      Files.writeString(out, "\"framework.dsl.Step\"\n" + args.drop(3).mkString("\n"))
        |      0
        |""".stripMargin
    ): Unit
    val actual = Tools.here.find("scala-cli")
    val stubs = Stubs(Scratch.directory("framework-sample-driver-tools"))
      .tool(
        "scala-cli",
        s"""case "$$1" in
           |  compile) printf 'stand-in.jar\n'; exit 0 ;;
           |esac
           |for argument in "$$@"; do
           |  case "$$argument" in *.scala) driver="$$argument" ;; esac
           |done
           |[ -n "$$driver" ] || { printf 'sample lift driver missing\n'; exit 1; }
           |while [ "$$1" != '--' ]; do shift; done
           |shift
           |exec '$actual' run '$fake' "$$driver" --scala 3.9.0 --main-class liftSamples -- "$$@"
           |""".stripMargin
      )
      .tool("go", "printf 'go must run only after all lifts pass\n' >> \"$GO_LOG\"")
    for failure <- Seq("none", "refused", "missing") do
      val output =
        Files.createDirectories(root.resolve(s"output $failure \"quoted\" \\path $$value"))
      val calls = output.resolve("lifts.log")
      val go = output.resolve("go.log")
      val tools = Tools(
        root,
        sys.env ++ Map(
          "PATH" -> (stubs.directory.toString + java.io.File.pathSeparator + sys.env("PATH")),
          "LIFT_LOG" -> calls.toString,
          "GO_LOG" -> go.toString,
          "LIFT_FAIL" -> failure
        )
      )
      if failure != "none" then
        val error = intercept[ToolError](Samples.check(tools, output))
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
          "\"framework.Step\"\nsamples.Zebra$.queries\nsamples.Zebra$.advances"
        )
      assertEquals(
        Files.readString(output.resolve("Alpha.check.json")),
        "\"framework.Step\"\nsamples.Alpha$.queries"
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
              s"${output.resolve("samples.jar")}=model/",
              output.resolve("samples.classpath").toString,
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
    for stage <- Seq("compile", "run") do
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
             |  printf 'stand-in.jar\n'
             |fi
             |exit 0""".stripMargin
        )
        .tool("go", s"printf 'invoked\n' > '$go'")
      val error = intercept[ToolError](Samples.check(stubs.tools(root), root.resolve("output")))
      assert(error.getMessage.contains("printed an error and exited 0"), error.getMessage)
      assert(error.getMessage.contains("sample compilation rejected"), error.getMessage)
      assert(!Files.exists(go))

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
