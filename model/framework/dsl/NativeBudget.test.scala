package framework.dsl

class NativeBudgetSuite extends munit.FunSuite:
  final case class BudgetState(available: Boolean, remaining: Int, bounded: UpTo[2], other: Boolean)
  private val loss = action("loss", Actor("fault")).responseLoss

  test("budgets belong to each binding and leave authored state and callbacks intact"):
    class Subject(selectRemaining: Boolean) extends RecordMachine[BudgetState, Boolean, Nothing]:
      var calls = 0
      def init: BudgetState = throw AssertionError("init evaluated")
      def end(s: BudgetState): Boolean = throw AssertionError("end evaluated")
      object transitions
          extends Bindings(
            (if selectRemaining then loss.budgetedBy(_.remaining)
             else loss.budgetedBy(_.available)) ~>
              ((s: BudgetState) =>
                calls += 1; List(Step(true, s.copy(other = !s.other)))
              )
          )
    val first = new Subject(false)
    val second = new Subject(true)
    val state = BudgetState(true, 2, UpTo(2), false)
    assertEquals(first.nativeBudgets.head.field.path, List("available"))
    assertEquals(second.nativeBudgets.head.field.path, List("remaining"))
    assertEquals(first.calls + second.calls, 0)
    for subject <- Vector(first, second) do
      val binding = subject.bindings.head
      val result =
        effectOf[BudgetState, Boolean, Nothing](binding.decl, binding.function)(state, Nil)
      assertEquals(result, List(Step(true, BudgetState(true, 2, UpTo(2), true))))
      assertEquals(subject.calls, 1)

  test("completed acquisition refuses duplicate budgets and another machine's field binding"):
    class Source extends RecordMachine[BudgetState, Boolean, Nothing]:
      def init: BudgetState = throw AssertionError("init evaluated")
      def end(s: BudgetState): Boolean = throw AssertionError("end evaluated")
      val target = loss.budgetedBy(_.available)
      object transitions extends Bindings(target ~> ((s: BudgetState) => List(Step(true, s))))
    val source = new Source
    object Foreign extends RecordMachine[BudgetState, Boolean, Nothing]:
      def init: BudgetState = throw AssertionError("init evaluated")
      def end(s: BudgetState): Boolean = throw AssertionError("end evaluated")
      object transitions
          extends Bindings(source.target ~> ((s: BudgetState) => List(Step(true, s))))
    intercept[IllegalArgumentException](Foreign.bindings): Unit
    object Duplicate extends RecordMachine[BudgetState, Boolean, Nothing]:
      def init: BudgetState = throw AssertionError("init evaluated")
      def end(s: BudgetState): Boolean = throw AssertionError("end evaluated")
      object transitions
          extends Bindings(
            loss.budgetedBy(_.available).budgetedBy(_.other) ~> ((s: BudgetState) =>
              List(Step(true, s))
            )
          )
    val duplicate = intercept[IllegalArgumentException](Duplicate.bindings)
    assert(duplicate.getMessage.contains("NativeBudget.test.scala"), duplicate.getMessage)

  test("rules and derivations retain budgets without retaining replaced choice programs"):
    object targets:
      val loss = action("ruleLoss", Actor("fault")).responseLoss
    val first = choice
    val second = choice
    val program = choose[BudgetState, Boolean, Nothing](
      first -> ((s: BudgetState) => List(Step(true, s))),
      second -> ((_: BudgetState) => Nil)
    )
    object Subject extends RecordMachine[BudgetState, Boolean, Nothing]:
      val init = BudgetState(true, 2, UpTo(2), false)
      def end(s: BudgetState): Boolean = false
      object transitions extends Rules:
        on(targets.loss.budgetedBy(_.bounded)):
          always ~> program
    assertEquals(Subject.nativeBudgets.head.field.path, List("bounded"))
    Subject.validateNativeChoices(targets.loss(), Seq(first))
    val restricted = Subject.restrict(targets.loss)
    assertEquals(restricted.nativeBudgets.head.field.path, List("bounded"))
    val rebound = Subject.rebind(targets.loss ~> ((s: BudgetState) => List(Step(false, s))))
    assertEquals(rebound.nativeBudgets.head.field.path, List("bounded"))
    intercept[IllegalArgumentException](rebound.validateNativeChoices(targets.loss(), Seq(first)))
