package framework.dsl

class BudgetInputShapeSuite extends munit.FunSuite:
  test("budgeted zero-input targets refuse binary guards and effects"):
    val refusals = List(
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        case class BudgetState(remaining: Int)
        object caller:
          val empty = action("empty", Actor("fault")).responseLoss
        object Subject extends Machine[BudgetState, Boolean, Nothing]:
          val init = BudgetState(1)
          def end(s: BudgetState): Boolean = false
          object rules extends Rules:
            on(caller.empty.budgetedBy(_.remaining)):
              where((s: BudgetState, in: EmptyTuple) => in == EmptyTuple) ~>
                ((s: BudgetState) => List(Step(true, s)))
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        case class BudgetState(remaining: Int)
        object caller:
          val empty = action("empty", Actor("fault")).responseLoss
        object Subject extends Machine[BudgetState, Boolean, Nothing]:
          val init = BudgetState(1)
          def end(s: BudgetState): Boolean = false
          object rules extends Rules:
            on(caller.empty.budgetedBy(_.remaining)):
              always ~> ((s: BudgetState, in: EmptyTuple) => List(Step(in == EmptyTuple, s)))
      """)
    )
    refusals.foreach(errors => assert(errors.nonEmpty, errors.toString))

  test("budgeted targets preserve state-only, record and scalar EmptyTuple callbacks"):
    case class BudgetState(remaining: Int)
    case class Request(flag: Boolean)
    object caller:
      val empty = action("empty", Actor("fault")).responseLoss
      val record = action("record", Actor("fault")).inputs[Request].responseLoss
      val unit = action("unit", Actor("fault")).input[EmptyTuple](name = "unit").responseLoss
    object Subject extends Machine[BudgetState, Boolean, Nothing]:
      val init = BudgetState(1)
      def end(s: BudgetState): Boolean = false
      object rules extends Rules:
        on(caller.empty.budgetedBy(_.remaining)):
          where(_.remaining > 0) ~> ((s: BudgetState) => List(Step(true, s)))
        on(caller.record.budgetedBy(_.remaining)):
          where((s, in) => s.remaining > 0 && in.flag) ~>
            ((s: BudgetState, in: Request) => List(Step(in.flag, s)))
        on(caller.unit.budgetedBy(_.remaining)):
          where((s, in) => s.remaining > 0 && in == EmptyTuple) ~>
            ((s: BudgetState, in: EmptyTuple) => List(Step(in == EmptyTuple, s)))
    for cls <- List(caller.empty(), caller.record(Request(true)), caller.unit(EmptyTuple)) do
      val binding = Subject.bindings.find(_.decl eq cls.decl).get
      val run = effectOf[BudgetState, Boolean, Nothing](binding.decl, binding.function)
      assertEquals(run(BudgetState(1), cls), List(Step(true, BudgetState(1))))
      assertEquals(run(BudgetState(0), cls), Nil)
