package framework.checker

import framework.dsl.*

// The native execution of what a Machine's transitions bind (fn-190.15 moved it out of
// framework.dsl): the step function each action's binding lowers to, the effects, guards, choices
// and crashes the DSL builds (Programs), the budgets a fault step consumes and the classes of an
// action.
private[framework] object Effects:
  // Whether a guard holds of the state and one class of its action: a guard that reads the action's
  // input reads the class's.
  private[framework] def guardHolds[S](guard: S => Boolean, state: S, cls: Class): Boolean =
    guard match
      case input: InputGuard[S] @unchecked => input(state, cls)
      case _                               => guard(state)

  // A binding's step function, read with the action's declared input.
  private[framework] def effectOf[S, O, F](decl: ActionDecl, f: AnyRef): InputEffect[S, O, F] =
    Programs.Adapted[S, O, F](decl, f)

  // The step function an action's binding lowers to: its own, or for rules the
  // effect of the first rule that fires, as `if g1(s) then e1(s) else if g2(s) then e2(s) else Nil`
  // lifts, or none where it is disabled.
  private[framework] def stepFunction[S, O, F](decl: ActionDecl, bound: Bound[S, O, F]): AnyRef =
    def run(s: S, cls: Class): List[Step[S, O, F]] = bound match
      case Bound.Ruled(rules) =>
        rules.iterator
          .filter(_.fires(cls.values))
          .map: rule =>
            def execute(): List[Step[S, O, F]] = rule.effect match
              case effect: InputEffect[S, O, F] @unchecked => effect(s, cls)
              case effect                                  => effect(s, cls.values)
            rule.guard match
              case fires: Programs.Fires[S] @unchecked if fires.effect eq rule.effect =>
                val steps = execute()
                Option.when(steps.nonEmpty)(steps)
              case guard if guardHolds(guard, s, cls) => Some(execute())
              case _                                  => None
          .collectFirst { case Some(steps) => steps }
          .getOrElse(Nil)
      case _ => Nil
    bound match
      case Bound.Function(f, _)            => f
      case _ if decl.nativeInputs.nonEmpty =>
        def adapted[I](shape: ActionInputs[I]): (S, I) => List[Step[S, O, F]] =
          (state, input) => run(state, shape.classOf(decl, input))
        adapted(decl.nativeInputs.get)
      case _ =>
        require(decl.inputs.isEmpty, s"action ${decl.name} has no declared input shape")
        (s: S) => run(s, Class(decl, Nil))

  // One step function per action of a machine: the rules of each, lowered (Rules.lowered).
  extension [S, O, F](machine: MachineDeclaration[S, O, F])
    private[framework] def bindings: List[StepBinding[S, O, F]] =
      machine.table.map { (decl, bound) =>
        val captured = bound match
          case Bound.Function(_, captured) => captured
          case _                           => None
        StepBinding.retaining[S, O, F](decl, stepFunction(decl, bound), captured)
      }.toList

  // Every class of an action: each assignment of its inputs, in catalog order.
  private[framework] def classesOf(decl: ActionDecl): List[List[Any]] =
    decl.nativeInputs.fold(
      decl.domains.foldLeft(List(List.empty[Any])): (prefixes, domain) =>
        for prefix <- prefixes; v <- domain.values.toList yield prefix :+ v
    )(_.classes)

  private[framework] def inputClassesOf(decl: ActionDecl): List[Class] =
    decl.nativeInputs.fold(classesOf(decl).map(Class(decl, _)))(_.inputClasses(decl))

  // What a fault step consumes of its budget in `state`: the budget field's next value, or nothing
  // where the budget is spent.
  private[framework] def consumption[S](
      budget: NativeBudget[S],
      state: S,
      location: String
  ): Option[(List[String], Any)] =
    val field = budget.field
    val writable = field.asInstanceOf[StateField[S, Any]] // scalafix:ok DisableSyntax.asInstanceOf
    val context =
      s"fault ${budget.declaration.name} at $location budget ${field.path.mkString(".")} at ${field.location}"
    writable.get(state) match
      case available: Boolean =>
        Option.when(available)(field.path -> false)
      case remaining: Int =>
        require(remaining >= 0, s"$context: negative own budget $remaining")
        Option.when(remaining > 0)(field.path -> (remaining - 1))
      case _ => throw IllegalArgumentException(s"$context: unsupported budget value")

  // The state a crash leaves: the fields its durability keeps, the others reset, and the budget
  // consumed.
  private[framework] def resetAfterCrash[S](
      machine: MachineDeclaration[S, ?, ?],
      state: S,
      budget: Vector[(List[String], Any)]
  ): S =
    val durability = machine.nativeDurability
    val replacements = durability.flatMap(_.replacement(state)) ++ budget
    state match
      case _: State =>
        replacements
          .groupBy(_._1.head)
          .toVector
          .sortBy(_._1)
          .foldLeft(state):
            case (current, (root, changes)) =>
              val selected = durability
                .find(_.field.path.head == root)
                .getOrElse(
                  throw IllegalArgumentException(s"unclassified crash field $root")
                )
              selected.field.rebuild(current, changes)
      case _ => durability.headOption.fold(state)(_.field.rebuild(state, replacements))

// The callables the DSL builds (Programs), run.
private[framework] object ProgramRuntime:
  import Effects.{consumption, guardHolds, resetAfterCrash}

  def evaluateEffect[S, O, F](
      program: InputEffect[S, O, F],
      state: S,
      values: List[Any]
  ): List[Step[S, O, F]] = program match
    case adapted: Programs.Adapted[S, O, F] @unchecked =>
      adapted.decl.nativeInputs match
        case Some(_) => effectOn(adapted, state, Class(adapted.decl, values))
        case None    =>
          adapted.function
            .asInstanceOf[S => List[Step[S, O, F]]](state) // scalafix:ok DisableSyntax.asInstanceOf
    case of: Programs.OfState[S, O, F] @unchecked        => of.function(state)
    case plain: Programs.Plain[S, O, F] @unchecked       => plain.function(state, values)
    case budgeted: Programs.Budgeted[S, O, F] @unchecked => budgeted.effect(state, values)
    case run: Programs.Run[?, ?]                         =>
      Vars
        .of(run.machine)
        .steps(state, run.accepted, fires = false)(run.body())
        .asInstanceOf[List[Step[S, O, F]]] // scalafix:ok DisableSyntax.asInstanceOf
    case run: Programs.InputRun[?, ?, ?] => effectOn(program, state, Class(run.decl, values))
    case other                           =>
      throw IllegalArgumentException(s"$other is no effect the DSL builds")

  def effectOn[S, O, F](
      program: InputEffect[S, O, F],
      state: S,
      cls: Class
  ): List[Step[S, O, F]] = program match
    case adapted: Programs.Adapted[S, O, F] @unchecked if adapted.decl.nativeInputs.nonEmpty =>
      def run[I](shape: ActionInputs[I]): List[Step[S, O, F]] =
        adapted.function
          .asInstanceOf[(S, I) => List[Step[S, O, F]]] // scalafix:ok DisableSyntax.asInstanceOf
          .apply(state, shape.input(cls))
      run(adapted.decl.nativeInputs.get)
    case budgeted: Programs.Budgeted[S, O, F] @unchecked => budgeted.effect(state, cls)
    case run: Programs.InputRun[?, ?, ?]                 => inputRun(run, state, cls)
    case other                                           => evaluateEffect(other, state, cls.values)

  private def inputRun[S, O, F, I](
      run: Programs.InputRun[?, ?, I],
      state: S,
      cls: Class
  ): List[Step[S, O, F]] =
    val input = run.shape.input(cls).asInstanceOf[I] // scalafix:ok DisableSyntax.asInstanceOf
    Vars
      .of(run.machine)
      .steps(state, run.accepted, fires = false)(run.body(input))
      .asInstanceOf[List[Step[S, O, F]]] // scalafix:ok DisableSyntax.asInstanceOf

  def holds[S](program: InputGuard[S], state: S, cls: Class): Boolean = program match
    case both: Programs.BothOnInput[S] @unchecked =>
      guardHolds(both.left, state, cls) && guardHolds(both.right, state, cls)
    case input: Programs.OnInput[S, ?] @unchecked => onInput(input, state, cls)
    case fires: Programs.Fires[S] @unchecked      => fires.effect(state, cls).nonEmpty
    case other                                    =>
      throw IllegalArgumentException(s"$other is no guard the DSL builds")

  private def onInput[S, A](program: Programs.OnInput[S, A], state: S, cls: Class): Boolean =
    program.condition(
      state,
      program.shape.input(cls).asInstanceOf[A] // scalafix:ok DisableSyntax.asInstanceOf
    )

  def call(program: AnyRef, argument: Any): Any = program match
    case claim: Programs.WitnessPredicate[Any, Any, Any] @unchecked =>
      val step =
        argument.asInstanceOf[Step[Any, Any, Any]] // scalafix:ok DisableSyntax.asInstanceOf
      claim.recorded(step) && claim.conditional.forall(_(step)) && claim.ending.forall(
        _(step.state)
      )
    case clause: Programs.ConditionalFact[Any, Any, Any] @unchecked =>
      val step =
        argument.asInstanceOf[Step[Any, Any, Any]] // scalafix:ok DisableSyntax.asInstanceOf
      !clause.condition(step.state) || clause.fact(step)
    case never: Programs.Never[Any, Any, Any] @unchecked =>
      !never.target(
        argument.asInstanceOf[Step[Any, Any, Any]]
      ) // scalafix:ok DisableSyntax.asInstanceOf
    case sticky: Programs.StickyVar =>
      argument.asInstanceOf[Boolean] || !sticky.promise() // scalafix:ok DisableSyntax.asInstanceOf
    case step: Programs.VarStep[?, ?] => step.vars.step(argument, step.accepted)(step.body())
    case budgeted: Programs.BudgetedState[?, ?, ?] =>
      budgeted.effect.asInstanceOf[Any => Any](argument) // scalafix:ok DisableSyntax.asInstanceOf
    case both: Programs.Both[?] =>
      val (left, right) =
        (both.left, both.right)
          .asInstanceOf[(Any => Boolean, Any => Boolean)] // scalafix:ok DisableSyntax.asInstanceOf
      left(argument) && right(argument)
    case crash: NativeCrash[?, ?, ?] => crashed(crash, argument)
    case other => throw IllegalArgumentException(s"$other is no step function the DSL builds")

  def call(program: AnyRef, first: Any, second: Any): Any = program match
    case never: Programs.NeverFrom[Any, Any, Any] @unchecked =>
      !never.origin(first) || !never.target(
        second.asInstanceOf[Step[Any, Any, Any]]
      ) // scalafix:ok DisableSyntax.asInstanceOf
    case stays: Programs.Stays[Any, Any, Any] @unchecked =>
      val step = second.asInstanceOf[Step[Any, Any, Any]] // scalafix:ok DisableSyntax.asInstanceOf
      !stays.condition(first) || stays.condition(step.state) || stays.release.exists(_(step))
    case keeps: Programs.Keeps[Any, Any, Any, Any] @unchecked =>
      val step = second.asInstanceOf[Step[Any, Any, Any]] // scalafix:ok DisableSyntax.asInstanceOf
      !keeps.over(first) || keeps.equivalent(keeps.projection(step.state), keeps.projection(first))
    case step: Programs.VarInputStep[?, ?, ?]         => varInputStep(step, first, second)
    case budgeted: Programs.BudgetedInput[?, ?, ?, ?] =>
      budgeted.effect
        .asInstanceOf[(Any, Any) => Any](first, second) // scalafix:ok DisableSyntax.asInstanceOf
    case condition: Programs.VarCondition[?] => varCondition(condition, first, second)
    case other => throw IllegalArgumentException(s"$other is no step function the DSL builds")

  private def varInputStep[I](step: Programs.VarInputStep[I, ?, ?], state: Any, input: Any): Any =
    step.vars.step(state, step.accepted)(
      step.body(input.asInstanceOf[I]) // scalafix:ok DisableSyntax.asInstanceOf
    )

  private def varCondition[A](condition: Programs.VarCondition[A], state: Any, input: Any): Any =
    condition.vars.at(state)(
      condition.condition(input.asInstanceOf[A]) // scalafix:ok DisableSyntax.asInstanceOf
    )

  private def crashed[S, O, F](crash: NativeCrash[S, O, F], argument: Any): List[Step[S, O, F]] =
    val state = argument.asInstanceOf[S] // scalafix:ok DisableSyntax.asInstanceOf
    val consume = crash.budgets.headOption.fold(Option(Vector.empty[(List[String], Any)])):
      budget =>
        consumption(
          budget.asInstanceOf[NativeBudget[S]], // scalafix:ok DisableSyntax.asInstanceOf
          state,
          crash.location
        ).map(Vector(_))
    consume.fold(Nil): replacements =>
      List(Step(crash.outcome, resetAfterCrash(crash.owner, state, replacements), crash.facts))
