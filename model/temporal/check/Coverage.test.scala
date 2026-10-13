package temporal.check

import framework.check.*

class CoverageSuite extends munit.FunSuite:
  private val root = Tools.here.directory
  private lazy val declared = Coverage.declared(root)
  private def scope(paths: String*) = Scope.of(root, paths, declared, Coverage)
  private def names(scope: Scope) = scope.suites.map(_._1.name).toSet
  private def module(name: String) = declared.filter(_.module.name == name).map(_.name).toSet

  test("every suite of every module is in the registry, and the registry names only suites"):
    val found = declared.map(_.name)
    assertEquals(found.distinct.size, found.size, "two suites share a class name")
    assertEquals(found.filterNot(Coverage.suites.contains), Vector.empty)
    assertEquals((Coverage.suites.keySet -- found).toVector.sorted, Vector.empty)
    assertEquals(declared.map(_.module.name).toSet, Set("models", "lifter", "check", "migrate"))

  test("an unregistered suite or a registry entry no suite declares is refused, by name"):
    val extra =
      declared.head.copy(
        name = "temporal.check.UnregisteredSuite",
        file = "model/temporal/check/X.scala"
      )
    val refused = intercept[GateError](
      Scope.of(root, Seq("model/temporal/check/Gate.scala"), declared :+ extra, Coverage)
    )
    assert(
      refused.getMessage.contains(
        "suite temporal.check.UnregisteredSuite (model/temporal/check/X.scala)"
      )
    )
    val missing = intercept[GateError](
      Scope.of(root, Nil, declared.filterNot(_.name == "framework.check.ToolsSuite"), Coverage)
    )
    assert(missing.getMessage.contains("names framework.check.ToolsSuite, which no suite declares"))

  test("a change under the mini feature selects only the mini feature and the suites that read it"):
    val mini = scope("model/temporal/features/activity_mini/protocol/Answering.scala")
    assertEquals(mini.features, Vector("activity_mini"))
    assertEquals(mini.behind, Vector.empty)
    assertEquals(mini.skipped, Vector.empty)
    val selected = names(mini)
    for suite <- selected do
      val covers = Coverage.suites(suite)
      assert(covers.features("activity_mini") || covers.anyFeature, suite)
    assert(selected("framework.export.MiniLayoutFixtures"))
    assert(selected("framework.dsl.AnswerEnteringTest"))
    assert(selected("temporal.migrate.QueueAliasTest"))
    assert(!selected("framework.lint.AdmissionSuite"))
    assert(!selected("framework.dsl.RulesTest"))
    assert(!selected("temporal.check.GateSuite"))

  test("a lifter change selects the lifter's suites and every current feature, and no Model suite"):
    val lifter = scope("model/framework/export/Lift.scala")
    assertEquals(lifter.features, Vector("activity_mini"))
    val selected = names(lifter)
    assert(module("lifter").subsetOf(selected))
    assertEquals(
      selected -- module("lifter"),
      Set(
        "temporal.check.CapabilityVocabularySuite",
        "framework.check.lint.CommentRuleSuite",
        "temporal.check.CoverageSuite",
        "temporal.check.GateLintSuite",
        "framework.check.SamplesSuite",
        "framework.check.lint.SyntaxRuleSuite"
      )
    )

  test("a docs-only change selects nothing"):
    val docs =
      scope("MILESTONES.md", "model/README.md", ".flow/tasks/x.md", "tools/umpire/README.md")
    assert(docs.empty)
    assertEquals(docs.suites, Vector.empty)
    assert(docs.why.forall(_._2.endsWith("selects no feature")), docs.why)

  test("a change to a behind feature checks nothing of it and names it behind"):
    val behind =
      scope("model/temporal/features/activity/standalone/protocol/ActivityProtocol.scala")
    assertEquals(behind.features, Vector.empty)
    assertEquals(behind.behind, Vector("activity/standalone"))
    assert(behind.skipped.map(_._1.name).contains("framework.ActivityRetryRegression"))
    assert(behind.suites.forall((d, _) => Coverage.suites(d.name).anyFeature), behind.suites)

  test("a kind-level file selects every feature of the kind, and a shared file every current one"):
    val kind = scope("model/temporal/features/activity/Activity.scala")
    assertEquals(kind.features, Vector("activity_mini"))
    assertEquals(kind.behind, Vector("activity/standalone"))
    assertEquals(scope("model/temporal/Bounds.scala").features, Vector("activity_mini"))
    assert(names(scope("model/SEMANTICS.md")) == Set("framework.check.lint.GuideSourcesTest"))

  test("the fn-169 homes of the lifter, the checks and the harness keep their areas"):
    assertEquals(Coverage.area("model/framework/export/Lift.scala"), Area.Lifter)
    assertEquals(Coverage.area("model/framework/lint/Order.scala"), Area.Lifter)
    assertEquals(Coverage.area("model/framework/inspect/Index.scala"), Area.Lifter)
    assertEquals(Coverage.area("model/temporal/export/Realizations.scala"), Area.Lifter)
    assertEquals(Coverage.area("model/framework/check/Gate.scala"), Area.Check)
    assertEquals(Coverage.area("model/temporal/check/Gate.scala"), Area.Check)
    assertEquals(Coverage.area("model/framework/migrate/Harness.scala"), Area.Migrate)
    assertEquals(Coverage.area("model/temporal/migrate/rules/Rule.scala"), Area.Migrate)
    assertEquals(Coverage.area("model/framework/dsl/Machine.scala"), Area.Framework)
    assertEquals(Coverage.area("model/temporal/foundations/taskqueue/Queue.scala"), Area.Shared)
    assertEquals(Coverage.area("model/temporal/check/baselines/activity_mini.map"), Area.Baselines)
    assertEquals(Coverage.area("model/build/model-scala.jar"), Area.Outside)
    assertEquals(Coverage.area("model/framework/README.md"), Area.Docs)
    assertEquals(Coverage.area("AGENTS.md"), Area.Docs)
    assertEquals(Coverage.area("Makefile"), Area.Make)
    assertEquals(Coverage.area("tools/umpire/ir/reader.go"), Area.Outside)

  test("every current feature has a baseline and a recorded rename map"):
    for feature <- Coverage.current do
      val baseline = Coverage.baselines(feature)
      assert(java.nio.file.Files.isRegularFile(root.resolve(baseline.renames)), baseline.renames)

  test("change-scoped suites use Bloop for tools and normal compilation for Temporal Models"):
    val record = Scratch.directory("change-check-compiler")
    val calls = record.resolve("calls")
    val tools = Stubs(Scratch.directory("change-check-compiler-tools"))
      .tool("scala-cli", s"""printf '%s\n' "$$*" >> '$calls'; echo 'OK (1 test)'""")
      .tools(root)
    val models = Coverage.modules.find(_.models).get
    val checks = Coverage.modules.find(_.name == "check").get
    val selected = Scope(
      Vector.empty,
      Vector.empty,
      Vector.empty,
      Vector.empty,
      Vector(
        Declared(
          models,
          "model/framework/dsl/Rules.test.scala",
          "framework.dsl.RulesTest"
        ) -> "selected",
        Declared(
          checks,
          "model/framework/check/Samples.test.scala",
          "framework.check.SamplesSuite"
        ) -> "selected"
      ),
      Vector.empty,
      Set.empty
    )
    val out = java.io.PrintStream(java.io.ByteArrayOutputStream())
    ChangeCheck(tools, out, record, Coverage).run(selected)
    val commands = java.nio.file.Files.readAllLines(calls)
    assertEquals(commands.size(), 2)
    assert(commands.get(0).contains("--server=false"), commands.get(0))
    assert(!commands.get(1).contains("--server=false"), commands.get(1))
