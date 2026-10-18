package framework.dsl

import framework.checker.Effects.*

class BudgetInputShapeSuite extends munit.FunSuite:
  given Ok[Boolean] = Ok(true)

  test("budgeted zero-input targets refuse input callbacks"):
    val errors = compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      given Ok[Boolean] = Ok(true)
      object fault extends Actor("fault"):
        val loss = action.responseLoss
      object Subject extends Machine[Boolean, Nothing]:
        var remaining = 1
        object effects:
          def accept(): Unit = ()
        object transitions extends Transitions:
          on(fault.loss.budgetedBy(remaining)) { (_: EmptyTuple) => effects.accept() }
    """)
    assert(errors.nonEmpty, errors.toString)

  test("widened scalar EmptyTuple targets execute state-only blocks safely"):
    object fault:
      private val actor = Actor("fault")
      val unit = framework.dsl.action("unit", actor).input[EmptyTuple](name = "unit").responseLoss
    class Subject(budgeted: Boolean) extends Machine[Boolean, Nothing]:
      var remaining = 1
      object targets:
        val plain: Action[EmptyTuple] = fault.unit
        val selected = fault.unit.budgetedBy(remaining)
      object effects:
        def accept(): Unit = ()
      object transitions extends Transitions:
        if budgeted then
          on(targets.selected):
            effects.accept()
        else
          on(targets.plain):
            effects.accept()
    for budgeted <- List(true, false) do
      val subject = new Subject(budgeted)
      val binding = subject.bindings.head
      assertEquals(
        effectOf[State, Boolean, Nothing](binding.decl, binding.function)(
          subject.init,
          fault.unit(EmptyTuple)
        ),
        List(Step(true, subject.init))
      )

  test("budgeted targets preserve state-only, record and scalar input blocks"):
    case class Request(flag: Boolean)
    object fault:
      private val actor = Actor("fault")
      val empty = framework.dsl.action("empty", actor).responseLoss
      val request = framework.dsl.action("request", actor).inputs[Request].responseLoss
      val scalar = framework.dsl.action("scalar", actor).input[Boolean](name = "flag").responseLoss
    object Subject extends Machine[Boolean, Nothing]:
      var remaining = 1
      object effects:
        def accept(): Unit = ()
        def answer(flag: Boolean): Unit = reject(flag)
      object transitions extends Transitions:
        on(fault.empty.budgetedBy(remaining)):
          if remaining > 0 then effects.accept()
        on(fault.request.budgetedBy(remaining)): in =>
          if remaining > 0 && in.flag then effects.answer(in.flag)
        on(fault.scalar.budgetedBy(remaining)): in =>
          if remaining > 0 && in then effects.answer(in)
    for cls <- List(fault.empty(), fault.request(Request(true)), fault.scalar(true)) do
      val binding = Subject.bindings.find(_.decl eq cls.decl).get
      val run = effectOf[State, Boolean, Nothing](binding.decl, binding.function)
      assertEquals(run(Subject.init, cls), List(Step(true, Subject.init)))
      assertEquals(run(Subject.init.copy(remaining = 0), cls), Nil)
    for cls <- List(fault.request(Request(false)), fault.scalar(false)) do
      val binding = Subject.bindings.find(_.decl eq cls.decl).get
      assertEquals(
        effectOf[State, Boolean, Nothing](binding.decl, binding.function)(Subject.init, cls),
        Nil
      )
