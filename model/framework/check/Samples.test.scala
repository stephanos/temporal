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
           |  compile) printf '/stand-in/classes\n' ;;
           |esac
           |for argument in "$$@"; do
           |  case "$$argument" in *.json) printf '{"type":"framework.dsl.Step"}\n' > "$$argument" ;; esac
           |done""".stripMargin
      )
      .tool("go", "printf 'go %s\n' " + '"' + "$*" + '"' + s" >> '$calls'")
    val output = Files.createDirectories(root.resolve("output"))

    val checked = Samples.check(stubs.tools(root), output)

    assertEquals(checked.map(_.name), Vector("Alpha", "Zebra"))
    val commands = Files.readAllLines(calls).toArray.toVector.map(_.toString)
    assert(
      commands.filter(_.startsWith("scala-cli ")).forall(!_.contains("--server=false")),
      commands.mkString("\n")
    )
    assertEquals(commands.count(_.startsWith("scala-cli --power package ")), 1)
    assertEquals(commands.count(_.startsWith("scala-cli compile ")), 1)
    assertEquals(commands.count(_.startsWith("scala-cli run ")), 2)
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
