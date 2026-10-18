package framework.prototype

import framework.dsl.*
import framework.checker.{Checker, BudgetPolicy, Execution, Expectation, ReplayOperation, ReplayStep, ReplayExpectation, ReplayScript, Result, Verdict}
import scala.collection.mutable.ArrayBuffer
import scala.quoted.*

// Prototype only: these carriers adapt the shipped checker, not the public DSL.
abstract class ModelSuite[M <: NativeModel[State, ?, ?]](val model: M) extends munit.FunSuite:
  private var registration = "new"
  private var registrationFailure: Option[Throwable] = None
  private var section = ""
  private var operations = Option.empty[ArrayBuffer[ReplayOperation[State]]]
  private var targets = Option.empty[ArrayBuffer[Expectation[State]]]
  private val names = scala.collection.mutable.Set.empty[String]
  lazy val execution: Execution = Checker.forSuite(BudgetPolicy.fromEnvironment(sys.env), getClass.getName)

  final def outcome: model.Outcome = NativeClaims.answer(null).asInstanceOf[model.Outcome]
  final def records(fact: Any): Boolean = NativeClaims.facts.contains(fact)

  override def munitTests(): Seq[munit.Test] = synchronized:
    registration match
      case "ready" => ()
      case "failed" => throw registrationFailure.get
      case "loading" => throw IllegalStateException("recursive ModelSuite registration")
      case _ =>
        registration = "loading"
        try
          Vector("examples", "explorations", "conformance", "unit").foreach: name =>
            section = name
            getClass.getMethods.find(m => m.getName == name && m.getParameterCount == 0).foreach: getter =>
              val owner = getter.invoke(this)
              owner.getClass.getDeclaredFields.filterNot(f => f.isSynthetic || f.getName.contains("$")).foreach: field =>
                val member = owner.getClass.getMethod(field.getName).invoke(owner)
                require(member.isInstanceOf[Exploration], s"$name.${field.getName}: only tests and space vals belong in sections")
          section = ""
          registration = "ready"
        catch
          case error: Throwable =>
            section = ""
            val cause = error match
              case wrapped: java.lang.reflect.InvocationTargetException => wrapped.getCause
              case other => other
            registrationFailure = Some(cause)
            registration = "failed"
            throw cause
    super.munitTests()

  override def test(name: String)(body: => Any)(using location: munit.Location): Unit =
    register(name, None, "satisfied", None)(body)

  override def test(options: munit.TestOptions)(body: => Any)(using location: munit.Location): Unit =
    require(section == "unit", "test options belong in unit")
    super.test(options)(body)

  final def test(name: String, from: State, expectation: String = "satisfied", through: Option[String] = None)(using munit.Location): TestBuilder =
    new TestBuilder(name, Some(from), expectation, through)

  final def test(name: String, expectation: String, through: Option[String])(using munit.Location): TestBuilder =
    new TestBuilder(name, None, expectation, through)

  final def test(name: String, expectation: String)(using munit.Location): TestBuilder =
    new TestBuilder(name, None, expectation, None)

  final class TestBuilder(name: String, from: Option[State], expectation: String, through: Option[String])(using munit.Location):
    def apply(body: => Any): Unit = register(name, from, expectation, through)(body)

  private def register(name: String, from: Option[State], expectation: String, through: Option[String])(body: => Any)(using location: munit.Location): Unit =
    require(section.nonEmpty, s"$name: test outside examples/explorations/conformance/unit")
    val origin = Thread.currentThread().getStackTrace.iterator.map(_.getClassName).find: name =>
      Vector("examples", "explorations", "conformance", "unit").exists(s => name.endsWith(s"$$$s$$"))
    require(origin.forall(_.endsWith(s"$$$section$$")), s"$name: cross-section initialization during $section")
    val slug = name.toLowerCase.replaceAll("[^a-z0-9]+", "-").stripPrefix("-").stripSuffix("-")
    require(names.add(slug), s"$name: duplicate test slug $slug")
    val selected = section
    super.test(munit.TestOptions(s"$selected: $name")):
      if selected == "unit" then body
      else
        val steps = ArrayBuffer.empty[ReplayOperation[State]]
        operations = Some(steps)
        try
          val _ = body
          if steps.nonEmpty then
            require(from.nonEmpty || model.initialStates.size == 1, s"$name: multiple starts require from")
            val result = execution.replay(model, ReplayScript(steps.toVector, from.toSeq))
            assertEquals(result.verdict, Verdict.verified, result.toString)
          println(s"prototype $selected.$slug expectation=$expectation through=$through budget=${execution.budget}")
        finally operations = None

  final def step(target: Any): Unit = operations.getOrElse(throw IllegalStateException("step outside test")) += ReplayStep[State](target): Unit

  final def oneOf(first: Any, second: Any, rest: Any*): Unit =
    operations.get += ReplayStep[State](first, (second +: rest)*): Unit

  final def repeat(count: Int)(target: => Unit): Unit =
    require(count >= 1, "repeat requires a positive count")
    (0 until count).foreach(_ => target)

  final def capture(line: Int, text: String)(condition: => Boolean): Unit =
    val expectation = Expectation[State](line, text, _ => condition)
    targets match
      case Some(buffer) => buffer += expectation: Unit
      case None => operations.get += ReplayExpectation(expectation): Unit

  inline final def expect(inline body: Any): Unit = ${ExpectMacro.expand('this, 'body)}

  final class Exploration(val steps: Int, select: (Int, Class) => Boolean):
    private def space = Checker.space(model, steps)(select)
    def find(steps: Int)(body: => Unit): Unit =
      val conditions = ArrayBuffer.empty[Expectation[State]]
      targets = Some(conditions)
      try body finally targets = None
      require(conditions.nonEmpty, "find requires an expect target")
      val target = Expectation[State](conditions.head.line, conditions.map(_.text).mkString(" && "), s => conditions.forall(_.test(s)))
      val result = execution.find(space, steps)(target)
      assertEquals(result.verdict, Verdict.found, result.toString)
      println(result)
    def sample(count: Int, seed: Long): Result =
      val result = execution.sample(space, count, seed)
      assertEquals(result.verdict, Verdict.verified, result.toString)
      assertEquals(result.sampling.get.walks.size, count)
      println(result)
      result

  final def explore(steps: Int)(select: (Int, Class) => Boolean): Exploration =
    new Exploration(steps, select)

  final def any(action: Action[?]): (Int, Class) => Boolean =
    require(section == "explorations", "any belongs only in explorations")
    (_, cls) => cls.decl eq action.decl

  inline final def refuted[O, F](inline instance: NativeModel[State, O, F], inline claim: Any): Result =
    execution.refuted(instance, claim)

object ExpectMacro:
  def expand[M <: NativeModel[State, ?, ?]: Type](suite: Expr[ModelSuite[M]], body: Expr[Any])(using Quotes): Expr[Unit] =
    import quotes.reflect.*
    def statements(term: Term): List[Statement] = term match
      case Inlined(_, _, value) => statements(value)
      case Block(stats, last) => stats ++ List(last)
      case other => List(other)
    val checks = statements(body.asTerm).map:
      case term: Term if term.tpe <:< TypeRepr.of[Boolean] =>
        val condition = term.asExprOf[Boolean]
        val line = Expr(term.pos.startLine + 1)
        val text = Expr(term.pos.sourceCode.getOrElse(term.show))
        '{$suite.capture($line, $text)($condition)}
      case other => report.errorAndAbort("expect requires bare Boolean statements", other.pos)
    if checks.isEmpty then report.errorAndAbort("expect requires a Boolean statement")
    Expr.block(checks, '{()})
