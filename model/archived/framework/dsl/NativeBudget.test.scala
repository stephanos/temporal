package framework.dsl

import framework.checker.Effects.*

class NativeBudgetSuite extends munit.FunSuite:
  given Ok[Boolean] = Ok(true)

  object faults:
    private val actor = Actor("fault")
    val loss = action("loss", actor).responseLoss

  private def run(machine: MachineDeclaration[State, Boolean, Nothing], state: State) =
    machine.nativeEffects.head._2(state, faults.loss())

  test("budgets belong to each binding and leave authored state and callbacks intact"):
    var calls = 0
    class Subject(selectRemaining: Boolean) extends Machine[Boolean, Nothing]:
      var available = true
      var remaining = 2
      var bounded = UpTo[2](2)
      var other = false
      object transitions extends Transitions:
        if selectRemaining then
          on(faults.loss.budgetedBy(remaining)):
            effects.toggle()
        else
          on(faults.loss.budgetedBy(available)):
            effects.toggle()
      object effects:
        def toggle(): Unit =
          calls += 1
          other = !other
    val first = new Subject(false)
    val second = new Subject(true)
    assertEquals(first.nativeBudgets.head.field.path, List("available"))
    assertEquals(second.nativeBudgets.head.field.path, List("remaining"))
    assertEquals(calls, 0)
    for subject <- Vector(first, second) do
      assertEquals(run(subject, subject.init), List(Step(true, subject.init.copy(other = true))))
    assertEquals(calls, 2)

  test("completed acquisition refuses duplicate budgets and another machine's field binding"):
    object Source extends Machine[Boolean, Nothing]:
      var available = true
      lazy val target = faults.loss.budgetedBy(available)
      object transitions extends Transitions:
        on(Source.target):
          effects.noop()
      object effects:
        def noop(): Unit =
          ()
    object Foreign extends Machine[Boolean, Nothing]:
      var available = true
      object transitions extends Transitions:
        on(Source.target):
          effects.noop()
      object effects:
        def noop(): Unit =
          ()
    intercept[IllegalArgumentException](Foreign.bindings): Unit
    object Duplicate extends Machine[Boolean, Nothing]:
      var available = true
      var other = true
      object transitions extends Transitions:
        on(faults.loss.budgetedBy(available).budgetedBy(other)):
          effects.noop()
      object effects:
        def noop(): Unit =
          ()
    val duplicate = intercept[IllegalArgumentException](Duplicate.bindings)
    assert(duplicate.getMessage.contains("NativeBudget.test.scala"), duplicate.getMessage)

  test("native derivations retain budgets when replacing their effects"):
    object Subject extends Machine[Boolean, Nothing]:
      var bounded = UpTo[2](2)
      object transitions extends Transitions:
        on(faults.loss.budgetedBy(bounded)):
          effects.noop()
      object effects:
        def noop(): Unit =
          ()
        def replacement(): Unit =
          bounded = UpTo[2](0)
    assertEquals(Subject.nativeBudgets.head.field.path, List("bounded"))
    val rebound = Subject.rebind(on(faults.loss):
      Subject.effects.replacement())
    assertEquals(rebound.nativeBudgets.head.field.path, List("bounded"))
    assertEquals(
      run(rebound, Subject.init),
      List(Step(true, Subject.init.copy(bounded = UpTo[2](0))))
    )
