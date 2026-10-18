package framework.check

import java.nio.file.{Files, Path}

class BudgetGateTest extends munit.FunSuite:
  private def suite(root: Path, name: String): Path =
    val path = root.resolve(s"model/framework/dsl/$name.test.scala")
    Files.createDirectories(path.getParent): Unit
    Files.writeString(path, s"package fixture\nclass $name extends munit.FunSuite\n"): Unit
    path

  test("gate policy validation follows source discovery, including unselected suites"):
    val root = Scratch.directory("budget-discovery")
    val source = suite(root, "NewSuite")
    val tools = Tools(root, Map("UMPIRE_CHECKER_BUDGETS" -> "fixture.NewSuite=7"))
    tools.withCheckerBudgets(FrameworkSuites.declared(root).map(_.name).toSet)
    Files.delete(source)
    val error = intercept[GateError]:
      tools.withCheckerBudgets(FrameworkSuites.declared(root).map(_.name).toSet)
    assert(error.getMessage.contains("fixture.NewSuite"))
    assert(error.getMessage.contains("UMPIRE_CHECKER_BUDGETS"))

  test("a default-package source-discovered suite accepts its exact override name"):
    val root = Scratch.directory("budget-default-package")
    val source = suite(root, "SimpleSuite")
    Files.writeString(source, "class SimpleSuite extends munit.FunSuite\n")
    val tools = Tools(root, Map("UMPIRE_CHECKER_BUDGETS" -> "SimpleSuite=7"))
    tools.withCheckerBudgets(FrameworkSuites.declared(root).map(_.name).toSet)

  test("validated normalized policy reaches child tools through their existing environment"):
    val root = Scratch.directory("budget-transport")
    val stubs = Stubs(Scratch.directory("budget-tools"))
      .tool("scala-cli", "printf '%s' \"$UMPIRE_CHECKER_BUDGETS\"")
    val tools = stubs.tools(
      root,
      "UMPIRE_CHECKER_BUDGETS" -> "fixture.SecondSuite=19\ndefault=31\nfixture.FirstSuite=7"
    )
    val testing = tools.withCheckerBudgets(Set("fixture.FirstSuite", "fixture.SecondSuite"))
    assertEquals(
      testing.scalaCli(Seq("run")).orFail().output,
      "default=31\nfixture.FirstSuite=7\nfixture.SecondSuite=19"
    )

  test("framework and change gates reject unknown policy before their first command"):
    val root = Scratch.directory("budget-before-command")
    suite(root, "KnownSuite")
    val stubs = Stubs(Scratch.directory("budget-refusal-tools"))
      .tool("scala-cli", s"printf called > '$root/calls'")
    val tools = stubs.tools(root, "UMPIRE_CHECKER_BUDGETS" -> "fixture.UnknownSuite=7")
    val record = Files.createDirectory(root.resolve("record"))
    val out = new java.io.PrintStream(java.io.OutputStream.nullOutputStream())
    val generic = intercept[GateError](FrameworkLoop(tools, out, record).check())
    assert(generic.getMessage.contains("fixture.UnknownSuite"))
    val scope = Scope(
      Vector.empty,
      Vector.empty,
      Vector.empty,
      Vector.empty,
      Vector.empty,
      Vector.empty,
      Set.empty
    )
    val changed = intercept[GateError](ChangeCheck(tools, out, record, FrameworkSuites).run(scope))
    assert(changed.getMessage.contains("fixture.UnknownSuite"))
    assert(!Files.exists(root.resolve("calls")))

  test("every framework module receives the normalized policy including a known held override"):
    val root = Scratch.directory("budget-framework")
    suite(root, "NewSuite")
    for (file, source) <- Vector(
        "model/framework/check/Check.test.scala" -> "package fixture\nclass CheckSuite extends munit.FunSuite\n",
        "model/framework/check/lint/Guide.test.scala" -> "package framework.check.lint\nclass GuideSourcesTest extends munit.FunSuite\n",
        "model/framework/export/Ids.test.scala" -> "package framework.`export`\nclass DerivedIds extends munit.FunSuite\n",
        "model/build/ir-scalapb.jar" -> "IR"
      )
    do
      val path = root.resolve(file)
      Files.createDirectories(path.getParent): Unit
      Files.writeString(path, source): Unit
    val stubs = Stubs(Scratch.directory("budget-framework-tools")).tool(
      "scala-cli",
      s"""case "$$*" in
         |  *--print-class-path*)
         |    while [ "$$1" != --compilation-output ]; do shift; done
         |    shift; /usr/bin/touch "$$1"; printf '%s:$root/model/build/ir-scalapb.jar' "$$1" ;;
         |  *--assembly*)
         |    while [ "$$1" != -o ]; do shift; done
         |    shift; /usr/bin/touch "$$1" ;;
         |  *JUnitCore*)
         |    printf '%s\n---\n' "$$UMPIRE_CHECKER_BUDGETS" >> '$root/policies'
         |    printf 'OK (1 test)\n' ;;
         |esac""".stripMargin
    )
    val tools = stubs.tools(
      root,
      "UMPIRE_CHECKER_BUDGETS" -> "framework.check.lint.GuideSourcesTest=5\nfixture.NewSuite=7\ndefault=31"
    )
    val record = Files.createDirectory(root.resolve("record"))
    FrameworkLoop(tools, new java.io.PrintStream(java.io.OutputStream.nullOutputStream()), record)
      .check()
    val policies = Files.readString(root.resolve("policies")).split("---\n").toVector
    assertEquals(policies.size, 3)
    assert(
      policies.forall(
        _ ==
          "default=31\nfixture.NewSuite=7\nframework.check.lint.GuideSourcesTest=5\n"
      )
    )
