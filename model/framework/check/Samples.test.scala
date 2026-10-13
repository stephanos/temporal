package framework.check

import java.nio.file.Files
import scala.concurrent.duration.*

class SamplesSuite extends munit.FunSuite:
  override def munitTimeout: Duration =
    if sys.env.contains("SAMPLE_REPORT_INPUT") then 10.minutes else super.munitTimeout

  test("sample builds reuse across outputs and invalidate changed inputs and damaged outputs"):
    val root = Scratch.directory("framework-samples")
    val samples = Files.createDirectories(root.resolve("model/framework/samples"))
    for name <- Seq("Zebra.scala", "Alpha.scala", "Ignored.test.scala", "README.md") do
      Files.writeString(samples.resolve(name), "package samples\n"): Unit
    Files.createDirectories(root.resolve("model/framework/dsl")): Unit
    Files.createDirectories(root.resolve("model/framework/realize")): Unit
    Files.createDirectories(root.resolve("model/framework/export")): Unit
    val calls = root.resolve("calls")
    val toolDirectory = Scratch.directory("framework-sample-tools")
    val stubs = Stubs(toolDirectory)
      .tool("sample-checker", s"printf 'checker %s\\n' \"$$*\" >> '$calls'")
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
      .tool(
        "go",
        s"""printf 'go %s\n' "$$*" >> '$calls'
           |while [ "$$1" != '-o' ]; do shift; done
           |shift
           |/bin/cp '${toolDirectory.resolve("sample-checker")}' "$$1"
           |""".stripMargin
      )
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
    assertEquals(commands.count(_.startsWith("go build ")), 1)
    val check = commands.find(_.startsWith("checker ")).getOrElse(fail("sample check did not run"))
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
    assertEquals(again.linesIterator.count(_.startsWith("go build ")), 2)
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
    val toolDirectory = Scratch.directory("framework-sample-driver-tools")
    val stubs = Stubs(toolDirectory)
      .tool("sample-checker", "printf 'go must run only after all lifts pass\n' >> \"$GO_LOG\"")
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
      .tool(
        "go",
        s"""while [ "$$1" != '-o' ]; do shift; done
           |shift
           |/bin/cp '${toolDirectory.resolve("sample-checker")}' "$$1"
           |""".stripMargin
      )
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

  test(
    "the generated Go checker overlaps at most two checks, joins failures and orders diagnostics"
  ):
    val root = Scratch.directory("framework-native-checker")
    def file(name: String, contents: String): Unit =
      val path = root.resolve(name)
      Files.createDirectories(path.getParent): Unit
      Files.writeString(path, contents): Unit
    file("go.mod", "module samplecheckerfixture\n\ngo 1.27.0\n")
    file("main.go", Samples.checker.replace("MODULE", "samplecheckerfixture"))
    file(
      "api/umpire/v1/query.go",
      """package umpire
        |const Query_FORM_VERIFY = 1
        |type Query struct { Name string; Form int }
        |func (q *Query) GetName() string { return q.Name }
        |func (q *Query) GetForm() int { return q.Form }
        |""".stripMargin
    )
    file(
      "tools/umpire/ir/model.go",
      """package ir
        |import (
        |  "fmt"
        |  p "samplecheckerfixture/api/umpire/v1"
        |)
        |type Model struct { Name string; Queries []*p.Query }
        |func (m *Model) GetQueries() []*p.Query { return m.Queries }
        |func Load(path string) (*Model, error) {
        |  if path == "load" { return nil, fmt.Errorf("invalid IR") }
        |  queries := []*p.Query{{Name: "find"}, {Name: "verify", Form: 1}, {Name: "refutesBad"}}
        |  if path == "empty" { queries = nil }
        |  return &Model{Name: path, Queries: queries}, nil
        |}
        |""".stripMargin
    )
    file(
      "tools/umpire/check/check.go",
      """package check
        |import (
        |  "fmt"
        |  "os"
        |  "sync"
        |  "sync/atomic"
        |  "time"
        |  "samplecheckerfixture/tools/umpire/ir"
        |)
        |type Kind string
        |const (Found Kind = "found"; Verified Kind = "verified"; Counterexample Kind = "counterexample")
        |type Subject string
        |const (QuerySubject Subject = "query"; ProgressSubject Subject = "progress")
        |type Key struct { Owner, Name string }
        |type Receipt struct { Subject Subject; Key Key; Kind Kind; Explanation string }
        |type Report struct { Receipts []Receipt }
        |var DefaultScope struct{}
        |var active atomic.Int32
        |var both = make(chan struct{})
        |var release = make(chan struct{})
        |var once sync.Once
        |var logLock sync.Mutex
        |func record(line string) {
        |  logLock.Lock()
        |  defer logLock.Unlock()
        |  f, err := os.OpenFile(os.Getenv("CHECK_LOG"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
        |  if err != nil { panic(err) }
        |  defer f.Close()
        |  if _, err := fmt.Fprintln(f, line); err != nil { panic(err) }
        |}
        |func Check(model *ir.Model, _ struct{}) Report {
        |  count := active.Add(1)
        |  if count > 2 { panic("more than two concurrent checks") }
        |  record("start " + model.Name)
        |  if count == 2 { once.Do(func() { close(both) }) }
        |  defer func() { active.Add(-1); record("done " + model.Name) }()
        |  failedOverlap := false
        |  if model.Name == "positive" || model.Name == "slow" {
        |    select {
        |    case <-both:
        |    case <-time.After(5*time.Second): failedOverlap = true
        |    }
        |  }
        |  if model.Name == "slow" && !failedOverlap {
        |    select {
        |    case <-release:
        |    case <-time.After(5*time.Second): failedOverlap = true
        |    }
        |  }
        |  if model.Name == "tail" { close(release) }
        |  receipts := []Receipt{
        |    {Subject: QuerySubject, Key: Key{Name: "find"}, Kind: Found},
        |    {Subject: QuerySubject, Key: Key{Name: "verify"}, Kind: Verified},
        |    {Subject: QuerySubject, Key: Key{Name: "refutesBad"}, Kind: Counterexample},
        |    {Subject: ProgressSubject, Key: Key{Owner: "fixture", Name: "advances"}, Kind: Verified},
        |  }
        |  if model.Name == "missing" { receipts = receipts[1:] }
        |  if model.Name == "wrong" || failedOverlap { receipts[0].Kind = Counterexample }
        |  if model.Name == "progress" { receipts[3].Kind = Counterexample }
        |  return Report{Receipts: receipts}
        |}
        |""".stripMargin
    )
    val tools = Tools.here.inDirectory(root)
    val executable = root.resolve("checker")
    tools.run("go", Seq("build", "-o", executable.toString, "main.go")).orFail(): Unit
    val log = root.resolve("checks.log")
    val paths = Seq("positive", "slow", "load", "empty", "missing", "wrong", "progress", "tail")
    val ran = tools
      .withEnvironment("CHECK_LOG" -> log.toString)
      .run(executable.toString, paths, Output.KeptApart)
    assertEquals(ran.exit, 1)
    val events = Files.readAllLines(log).toArray.toVector.map(_.toString)
    assertEquals(events.take(2).toSet, Set("start positive", "start slow"))
    assertEquals(
      events.filter(_.startsWith("done ")).map(_.stripPrefix("done ")).toSet,
      paths.filterNot(_ == "load").toSet
    )
    assert(events.indexOf("done tail") < events.indexOf("done slow"), events.mkString("\n"))
    assertEquals(
      ran.output.linesIterator.map(_.takeWhile(_ != ':')).toVector,
      Vector("positive", "slow", "missing", "wrong", "progress", "tail")
    )
    val errors = ran.standardError.linesIterator.toVector
    assertEquals(
      errors.map(_.takeWhile(_ != ':')),
      Vector("load", "empty", "missing", "wrong", "progress")
    )
    assert(errors.exists(_.contains("invalid IR")))
    assert(errors.exists(_.contains("no Query to check")))
    assert(errors.exists(_.contains("produced no receipt")))
    assert(errors.exists(_.contains("want found")))
    assert(errors.exists(_.contains("want verified")))

  test("the generated parallel checker preserves complete real sample reports"):
    val input = sys.env.get("SAMPLE_REPORT_INPUT")
    assume(input.nonEmpty, "real sample report comparison is selected with SAMPLE_REPORT_INPUT")
    val root = Tools.here.directory
    val directory = java.nio.file.Path.of(input.get).toAbsolutePath
    val baseline = java.nio.file.Path.of(sys.env("SAMPLE_REPORT_BASE")).toAbsolutePath
    val paths =
      Seq("Bank", "Kettle", "ReliableBroadcast", "TwoLayeredCache", "TwoPhaseCommit", "Voting")
        .map(name => directory.resolve(name + ".check.json").toString)
    val source = directory.resolve("check-reports.go")
    val executable = directory.resolve("check-reports")
    val module =
      Files.readString(root.resolve("go.mod")).linesIterator.next().stripPrefix("module ")
    val capture =
      """report := check.Check(model, check.DefaultScope)
        |    encoded, err := json.MarshalIndent(report, "", "  ")
        |    if err != nil { panic(err) }
        |    if err := os.WriteFile(path+".receipts.json", append(encoded, '\n'), 0600); err != nil { panic(err) }
        |""".stripMargin
    Files.writeString(
      source,
      Samples.checker
        .replace("MODULE", module)
        .replace("\"bytes\"", "\"bytes\"\n  \"encoding/json\"")
        .replace("report := check.Check(model, check.DefaultScope)", capture)
    ): Unit
    Tools.here.run("go", Seq("build", "-o", executable.toString, source.toString)).orFail(): Unit
    val start = System.nanoTime
    Tools.here.run(executable.toString, paths).orFail(): Unit
    println(s"parallel report checker wall seconds: ${(System.nanoTime - start) / 1e9}")
    for name <- paths.map(path =>
        java.nio.file.Path.of(path).getFileName.toString.stripSuffix(".check.json")
      )
    do
      for suffix <- Seq(".json", ".check.json", ".check.json.receipts.json") do
        assert(
          java.util.Arrays.equals(
            Files.readAllBytes(baseline.resolve(name + suffix)),
            Files.readAllBytes(directory.resolve(name + suffix))
          ),
          s"$name$suffix differs from the serial baseline"
        )
    println("parallel sample parity: 6 raw IRs, 6 adapted IRs, 6 complete Reports identical")

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
