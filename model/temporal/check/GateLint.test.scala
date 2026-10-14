package temporal.check

import java.io.{ByteArrayOutputStream, PrintStream}
import java.nio.file.{Files, Path}
import framework.check.{GateError, Scratch, Tools}
import framework.check.lint.Boundary

// The gate's own runs of the framework's source rules and of its settlement of selected outputs: what
// each command line reads and prints. The rules themselves are tested beside them
// (model/framework/check/lint/, ModelSources.test.scala).
class GateLintSuite extends munit.FunSuite:
  // A repository of the files given, each a path under its root and its text.
  private def repository(files: (String, String)*): Path =
    val root = Scratch.directory("umpire-gate-lint")
    for (file, text) <- files do
      val path = root.resolve(file)
      Files.createDirectories(path.getParent)
      Files.writeString(path, text)
    root

  // The framework's sugar as model/framework/Syntax.scala declares it, documented.
  private val syntax =
    """// Sugar of the framework.
      |package framework
      |
      |// The ok outcome of a machine. Core form: the outcome itself, `Outcome.accepted`.
      |final case class Ok[O](outcome: O)
      |
      |// One step with the ok outcome. Core form:
      |// `List(Step(Outcome.accepted, state, List(facts*)))`.
      |def enter[S, O, F](state: S, facts: F*)(using ok: Ok[O]): List[Step[S, O, F]] =
      |  List(Step(ok.outcome, state, facts.toList))
      |
      |// No step. Core form: `Nil`.
      |val disabled: List[Nothing] = Nil
      |
      |// `phase.in(a, b)`. Core form: `List(a, b).contains(phase)`.
      |extension [A](value: A) def in(first: A, rest: A*): Boolean = (first +: rest).contains(value)
      |
      |// `a implies b`. Core form: `!a || b`.
      |@deprecated("example")
      |extension (a: Boolean) infix def implies(b: => Boolean): Boolean = !a || b
      |
      |extension [S, O, F](step: Step[S, O, F])
      |  // `after.records(fact)`. Core form: `after.facts.contains(fact)`.
      |  def records(fact: F): Boolean = step.facts.contains(fact)
      |
      |private def helper(n: Int): Int =
      |  val in = n
      |  in + 1
      |
      |// Sugar spelled through an object. Core form: `Sugar.stay(s)` is its member's.
      |object Sugar:
      |  // One step that keeps the state. Core form: `List(Step(Outcome.accepted, s))`.
      |  def stay[S](s: S): List[S] = List(s)
      |
      |  private[framework] def internal: Int = 1
      |""".stripMargin

  test("the gate's --check-block-form reads the Temporal Models and names each finding"):
    val root = Scratch.directory("umpire-block-form-rule")
    def write(file: String, text: String): Path =
      val path = root.resolve(file)
      Files.createDirectories(path.getParent)
      Files.writeString(path, text)
    write(
      "model/temporal/features/kind/form/FormKind.scala",
      "object r:\n  on(a.b)(always ~> e)\n"
    ): Unit
    write(
      "model/framework/export/testdata/lifts/Rules.scala",
      "object r:\n  on(a.b)(always ~> e)\n"
    ): Unit
    val (out, err) = (ByteArrayOutputStream(), ByteArrayOutputStream())
    val status =
      Gate.main(
        Seq("--check-block-form"),
        Tools(root, Map.empty),
        PrintStream(out),
        PrintStream(err)
      )
    assertEquals(status, 1)
    assertEquals(
      err.toString.linesIterator
        .map(_.takeWhile(_ != ' '))
        .toVector,
      Vector("model/temporal/features/kind/form/FormKind.scala:2:")
    )
    assert(err.toString.contains("block-form rule: write the rule as a block"), err.toString)

  test("the gate's --check-comments reads all of model/, tests and fixtures too, not build output"):
    def run(root: Path): (Int, String) =
      val (out, err) = (ByteArrayOutputStream(), ByteArrayOutputStream())
      val status =
        Gate.main(
          Seq("--check-comments"),
          Tools(root, Map.empty),
          PrintStream(out),
          PrintStream(err)
        )
      assertEquals(out.toString, "")
      (status, err.toString)
    val (failed, printed) = run(
      repository(
        "model/framework/Machine.scala" -> "package framework\n\n/** A machine. */\ntrait Machine\n",
        "model/framework/export/testdata/lifts/Fixture.scala" -> "/* A fixture. */\nobject Fixture\n",
        "model/build/Generated.scala" -> "/** Generated. */\nobject Generated\n",
        "model/temporal/check/.scala-build/Cached.scala" -> "/* Cached. */\nobject Cached\n"
      )
    )
    assertEquals(failed, 1)
    assertEquals(
      printed.linesIterator
        .map(_.takeWhile(_ != ' '))
        .toVector,
      Vector(
        "model/framework/Machine.scala:3:",
        "model/framework/export/testdata/lifts/Fixture.scala:1:"
      )
    )
    assert(printed.contains("comment rule: write the comment as // lines"), printed)
    assertEquals(
      run(repository("model/framework/Machine.scala" -> "// A machine.\ntrait M\n")),
      (0, "")
    )

  test("the gate's --check-syntax prints each finding on a line of its own and fails"):
    def run(root: Path): (Int, String) =
      val (out, err) = (ByteArrayOutputStream(), ByteArrayOutputStream())
      val status =
        Gate.main(Seq("--check-syntax"), Tools(root, Map.empty), PrintStream(out), PrintStream(err))
      assertEquals(out.toString, "")
      (status, err.toString)
    val sugar = "package framework\n\ndef implies(a: Boolean, b: Boolean): Boolean = !a || b\n"
    val (failed, printed) = run(repository("model/framework/Logic.scala" -> sugar))
    assertEquals(failed, 1)
    assert(printed.startsWith("model/framework/Logic.scala:3: syntax rule: "), printed)
    assertEquals(printed.linesIterator.size, 1)
    assertEquals(run(repository("model/framework/Syntax.scala" -> syntax)), (0, ""))

  test("narrowed settlement checks only produced files and leaves unlisted files untouched"):
    val root = Scratch.directory("selected-ir-settlement")
    val tree = Files.createDirectories(root.resolve("ir"))
    val produced = Files.createDirectories(root.resolve("produced"))
    Files.writeString(tree.resolve("unlisted.json"), "unlisted\n")
    Files.writeString(produced.resolve("selected.json"), "selected\n")
    Gate.settle(tree, produced, root, update = true, selected = true)
    assertEquals(Files.readString(tree.resolve("unlisted.json")), "unlisted\n")
    assertEquals(Files.readString(tree.resolve("selected.json")), "selected\n")
    Gate.settle(tree, produced, root, update = false, selected = true)
    val failure = intercept[GateError](Gate.settle(tree, produced, root, update = false))
    assert(failure.getMessage.contains("unlisted.json"))

  test(
    "the gate's --check-boundary holds model/framework to Temporal's packages and prints each finding"
  ):
    def run(root: Path): (Int, String) =
      val (out, err) = (ByteArrayOutputStream(), ByteArrayOutputStream())
      val status =
        Gate.main(
          Seq("--check-boundary"),
          Tools(root, Map.empty),
          PrintStream(out),
          PrintStream(err)
        )
      assertEquals(out.toString, "")
      (status, err.toString)
    val (failed, printed) = run(
      repository(
        "model/framework/export/Lift.scala" ->
          "package framework.`export`\n\nimport temporal.realize.Kit\n",
        "model/framework/dsl/Gaps.test.scala" ->
          "package framework.dsl\n\nval e = io.temporal.api.enums.v1.EventType.EVENT_TYPE_UNSPECIFIED\n"
      )
    )
    assertEquals(failed, 1)
    assertEquals(
      printed.linesIterator.map(_.takeWhile(_ != ' ')).toVector,
      Vector("model/framework/dsl/Gaps.test.scala:3:", "model/framework/export/Lift.scala:3:")
    )
    assertEquals(
      run(
        repository(
          "model/framework/export/Lift.scala" ->
            "package framework.`export`\n\nimport io.temporal.server.api.umpire.v1 as ir\n"
        )
      ),
      (0, "")
    )

  test("the repository's framework keeps to its boundary"):
    assertEquals(Boundary.findings(Tools.here.directory, Gate.systemPackages), Vector.empty)
