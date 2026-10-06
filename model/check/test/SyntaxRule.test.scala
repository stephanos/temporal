package umpire.check

import java.io.{ByteArrayOutputStream, PrintStream}
import java.nio.file.{Files, Path}

class SyntaxRuleSuite extends munit.FunSuite:
  // A repository of the files given, each a path under its root and its text.
  private def repository(files: (String, String)*): Path =
    val root = Files.createTempDirectory("umpire-syntax-rule")
    for (file, text) <- files do
      val path = root.resolve(file)
      Files.createDirectories(path.getParent)
      Files.writeString(path, text)
    root

  private def findings(files: (String, String)*): Vector[String] =
    SyntaxRule.findings(repository(files*))

  // The framework's sugar as model/umpire/Syntax.scala declares it, documented.
  private val syntax =
    """// Sugar of the framework.
      |package umpire
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
      |  private[umpire] def internal: Int = 1
      |""".stripMargin

  test("a documented Syntax.scala, its private helpers and its locals pass"):
    assertEquals(findings("model/umpire/Syntax.scala" -> syntax), Vector.empty)

  test("a Syntax.scala definition without a doc comment, or whose doc names no core form, fails"):
    val source =
      """package umpire
        |
        |def enter(n: Int): Int = n
        |
        |// One step: what it means, without its core spelling.
        |def stay(n: Int): Int = n
        |
        |// Core form: without a code span.
        |val disabled: List[Nothing] = Nil
        |
        |private val hidden = 0 // A remark after code, no doc. Core form: `x`.
        |
        |extension (a: Boolean) infix def implies(b: => Boolean): Boolean = !a || b
        |
        |extension (a: Int)
        |  // Documented. Core form: `a + 1`.
        |  def next: Int = a + 1
        |  def previous: Int = a - 1
        |
        |object Sugar:
        |  given Ordering[Int] = Ordering.Int
        |""".stripMargin
    val found = findings("model/temporal/realize/Syntax.scala" -> source)
    assertEquals(
      found.map(_.takeWhile(_ != ' ')),
      Vector(3, 6, 9, 13, 18, 20, 21).map(line => s"model/temporal/realize/Syntax.scala:$line:")
    )
    assert(found.head.contains("`enter` in a Syntax.scala has no doc comment"), found.head)
    assert(
      found(1).contains("the doc comment of `stay` in a Syntax.scala names no core form"),
      found(1)
    )
    assert(found(5).contains("`Sugar` in a Syntax.scala has no doc comment"), found(5))
    assert(found(6).contains("an anonymous given"), found(6))

  test("an extension's first definition may be documented above the extension"):
    val source =
      """package umpire
        |
        |// `a implies b`. Core form: `!a || b`.
        |extension (a: Boolean)
        |  infix def implies(b: => Boolean): Boolean = !a || b
        |""".stripMargin
    assertEquals(findings("model/umpire/Syntax.scala" -> source), Vector.empty)

  test("the lifter's Syntax trait documents its public members, not its private ones"):
    val source =
      """package umpire.irgen
        |
        |// The lifting of the sugar.
        |private[irgen] trait Syntax:
        |  self: Lifting =>
        |  import ctx.*
        |
        |  private val owner = "umpire.Syntax$package$"
        |
        |  // Hook: a sugar form lifted. Core form: `s.facts.contains(f)` lifts the same.
        |  def sugar(t: Term): Expr = t match
        |    case _ => fail(t)
        |
        |  // Hook: whether the term is a sugar form.
        |  def sugared(t: Term): Boolean = true
        |""".stripMargin
    val found = findings("model/irgen/Syntax.scala" -> source)
    assertEquals(found.map(_.takeWhile(_ != ' ')), Vector("model/irgen/Syntax.scala:15:"))
    assert(found.head.contains("`sugared`"), found.head)

  test("a sugar name defined at the top level, in an object or as an extension fails"):
    val umpire =
      """package umpire
        |
        |infix def implies(a: Boolean, b: Boolean): Boolean = !a || b
        |
        |object Steps:
        |  @annotation.targetName("assign")
        |  infix def :=(n: Int): Int = n
        |  val disabled = Nil
        |
        |extension [A](value: A) def in(xs: A*): Boolean = xs.contains(value)
        |""".stripMargin
    val temporal =
      """package temporal.features.activity.standalone
        |
        |extension (m: Machine)
        |  // Doc comments may say once, never and keeps.
        |  def never(p: Boolean): Boolean = !p
        |  inline def once(p: Boolean): Boolean = p
        |
        |private def sticky(p: Boolean): Boolean = p
        |""".stripMargin
    val lifter =
      """package umpire.irgen
        |
        |object Matching:
        |  object Inner:
        |    def unless(t: Term): Boolean = false
        |""".stripMargin
    val found = findings(
      "model/umpire/Steps.scala" -> umpire,
      "model/temporal/features/activity/standalone/Standalone.scala" -> temporal,
      "model/irgen/Matching.scala" -> lifter
    )
    assertEquals(
      found.map(_.takeWhile(_ != ' ')),
      Vector(
        "model/irgen/Matching.scala:5:",
        "model/temporal/features/activity/standalone/Standalone.scala:5:",
        "model/temporal/features/activity/standalone/Standalone.scala:6:",
        "model/temporal/features/activity/standalone/Standalone.scala:8:",
        "model/umpire/Steps.scala:3:",
        "model/umpire/Steps.scala:7:",
        "model/umpire/Steps.scala:8:",
        "model/umpire/Steps.scala:10:"
      )
    )
    assert(found.head.contains("move it into model/irgen/Syntax.scala"), found.head)
    assert(found(1).contains("move it into model/temporal/realize/Syntax.scala"), found(1))
    assert(found(4).contains("`implies` is a sugar name defined outside a Syntax.scala"), found(4))

  test("class members, constructor parameters, locals and comments are not sugar"):
    val claims =
      """package umpire
        |
        |final class QueryOn[P] private[umpire] (name: String, p: Property[P]):
        |  // The scenario clause: `query find p in s`. Sugar words in a doc: enter, implies.
        |  infix def in[S](s: Scenario[S]): QueryIn = QueryIn(name, p.decl, s.decl)
        |
        |final class Progress[S] private[umpire] (
        |    val name: String,
        |    val from: S => Boolean,
        |    val to: S => Boolean
        |)
        |
        |final class Evidence private[realize] (
        |    val records: String,
        |    val from: Recorded
        |) extends Base:
        |  def stay: Int = 1
        |
        |enum Delivered:
        |  case never, once, twice
        |
        |// def implies(a: Boolean, b: Boolean) = !a || b
        |val text = "def enter(s: S) = s"
        |
        |def providerQueries(m: Machine): Vector[Query] =
        |  val stays = m.property("committedStays") holdsAcross committedStays
        |  def once(p: Boolean): Boolean = p
        |  object local:
        |    def keeps: Int = 1
        |  Vector(stays)
        |""".stripMargin
    assertEquals(findings("model/umpire/Claims.scala" -> claims), Vector.empty)

  test("fixtures, tests and build output are not read"):
    val sugar = "package x\n\ndef implies(a: Boolean, b: Boolean): Boolean = !a || b\n"
    assertEquals(
      findings(
        "model/irgen/testdata/lifts/Model.scala" -> sugar,
        "model/irgen/test/Lift.test.scala" -> sugar,
        "model/temporal/features/activity/standalone/Pins.test.scala" -> sugar,
        "model/umpire/.scala-build/Gen.scala" -> sugar
      ),
      Vector.empty
    )

  test("a core file that imports a Syntax module or a name of its sugar fails"):
    val lifter =
      """package umpire.irgen
        |
        |import scala.collection.mutable
        |import umpire.irgen.Syntax
        |
        |trait Lifting:
        |  self: Syntax =>
        |  def lifted(t: Term) = if sugared(t) then sugar(t) else plain(t)
        |""".stripMargin
    val lifterSyntax =
      """package umpire.irgen
        |
        |// The lifting of the sugar.
        |private[irgen] trait Syntax:
        |  // Hook: a sugar form. Core form: `xs.contains(x)` lifts the same.
        |  def sugared(t: Term): Boolean = true
        |
        |  // Hook: its IR. Core form: `xs.contains(x)` lifts the same.
        |  def sugar(t: Term): Expr = ???
        |""".stripMargin
    val umpire =
      """package umpire
        |
        |import umpire.{
        |  Step,
        |  enter
        |}
        |import umpire.Syntax$package.*
        |
        |def core: Int = 1
        |""".stripMargin
    val found = findings(
      "model/umpire/Syntax.scala" -> syntax,
      "model/umpire/Core.scala" -> umpire,
      "model/irgen/Syntax.scala" -> lifterSyntax,
      "model/irgen/Lifting.scala" -> lifter
    )
    assertEquals(
      found,
      Vector(
        "model/irgen/Lifting.scala:4: syntax rule: a core file imports the sugar of " +
          "model/irgen/Syntax.scala (`Syntax`): remove the import and write the core form",
        "model/umpire/Core.scala:3: syntax rule: a core file imports the sugar of " +
          "model/umpire/Syntax.scala (`enter`): remove the import and write the core form",
        "model/umpire/Core.scala:7: syntax rule: a core file imports the sugar of " +
          "model/umpire/Syntax.scala (`Syntax$package`): remove the import and write the core form"
      )
    )

  test(
    "a core file of the framework that names its sugar fails, unless the core declares the name"
  ):
    val core =
      """package umpire
        |
        |// Doc comments may say `enter(s)` and `a implies b`.
        |def step[S](s: S)(using Ok[String]): List[Step[S, String, Nothing]] =
        |  val entered = "enter"
        |  if s.isInstanceOf[Int] implies true then enter(s) else disabled
        |
        |final class QueryOn[P](name: String):
        |  infix def in[S](s: Scenario[S]): QueryIn = QueryIn(name, s)
        |
        |def scenario(q: QueryOn[Int], s: Scenario[Int]): QueryIn = q in s
        |""".stripMargin
    val found = findings("model/umpire/Syntax.scala" -> syntax, "model/umpire/Core.scala" -> core)
    assertEquals(
      found.map(_.takeWhile(_ != ' ')),
      Vector(4, 6, 6, 6).map(line => s"model/umpire/Core.scala:$line:")
    )
    assertEquals(
      found.map(_.replaceAll(".*uses `([^`]+)`.*", "$1")),
      Vector("Ok", "implies", "enter", "disabled")
    )
    assert(found.head.contains("sugar defined in model/umpire/Syntax.scala"), found.head)

  test("the lifter's core may name its Syntax trait's hooks"):
    val lifterSyntax =
      """package umpire.irgen
        |
        |// The lifting of the sugar.
        |private[irgen] trait Syntax:
        |  // Hook: whether it is sugar. Core form: `xs.contains(x)` lifts the same.
        |  def sugared(t: Term): Boolean = true
        |""".stripMargin
    val core =
      """package umpire.irgen
        |
        |trait Lifting extends Syntax:
        |  def lifted(t: Term): Boolean = sugared(t)
        |""".stripMargin
    assertEquals(
      findings("model/irgen/Syntax.scala" -> lifterSyntax, "model/irgen/Lifting.scala" -> core),
      Vector.empty
    )

  test("the gate's --check-syntax prints each finding on a line of its own and fails"):
    def run(root: Path): (Int, String) =
      val (out, err) = (ByteArrayOutputStream(), ByteArrayOutputStream())
      val status =
        Gate.main(Seq("--check-syntax"), Tools(root, Map.empty), PrintStream(out), PrintStream(err))
      assertEquals(out.toString, "")
      (status, err.toString)
    val sugar = "package umpire\n\ndef implies(a: Boolean, b: Boolean): Boolean = !a || b\n"
    val (failed, printed) = run(repository("model/umpire/Logic.scala" -> sugar))
    assertEquals(failed, 1)
    assert(printed.startsWith("model/umpire/Logic.scala:3: syntax rule: "), printed)
    assertEquals(printed.linesIterator.size, 1)
    assertEquals(run(repository("model/umpire/Syntax.scala" -> syntax)), (0, ""))

  test("the repository's sugar keeps to the rule"):
    assertEquals(SyntaxRule.findings(Tools.here.directory), Vector.empty)
