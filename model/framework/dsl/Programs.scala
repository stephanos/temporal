package framework.dsl

// The callables the DSL builds from what a Model writes, where a binding, a rule or a choice needs
// one: each holds the parts it was built from, the code the Model wrote and the declarations it
// reads, and runs nothing itself. Calling one hands it to the product checker through NativeRuntime;
// the lifter reads its declaration references.
private[framework] object Programs:
  final class NativeBlock(val body: () => Unit, val nativeChoices: Option[Vector[Choice]])
      extends (() => Unit),
        BudgetMetadata:
    val budgets: Vector[NativeBudget[?]] = Vector.empty
    def apply(): Unit = NativeRuntime.installed.invoke(body)

  final class NativeInputBlock[I](val body: I => Unit, val nativeChoices: Option[Vector[Choice]])
      extends (I => Unit),
        BudgetMetadata:
    val budgets: Vector[NativeBudget[?]] = Vector.empty
    def apply(input: I): Unit = NativeRuntime.installed.invoke(body, input)

  final class WitnessPredicate[S, O, F](
      val recorded: Step[S, O, F] => Boolean,
      val conditional: Vector[Step[S, O, F] => Boolean],
      val ending: Option[S => Boolean]
  ) extends (Step[S, O, F] => Boolean):
    def apply(step: Step[S, O, F]): Boolean = NativeRuntime.installed.call[Boolean](this, step)

  final class ConditionalFact[S, O, F](
      val condition: S => Boolean,
      val fact: Step[S, O, F] => Boolean
  ) extends (Step[S, O, F] => Boolean):
    def apply(step: Step[S, O, F]): Boolean = NativeRuntime.installed.call[Boolean](this, step)

  final class Never[S, O, F](val target: Step[S, O, F] => Boolean)
      extends (Step[S, O, F] => Boolean):
    def apply(step: Step[S, O, F]): Boolean = NativeRuntime.installed.call[Boolean](this, step)

  final class NeverFrom[S, O, F](val origin: S => Boolean, val target: Step[S, O, F] => Boolean)
      extends ((S, Step[S, O, F]) => Boolean):
    def apply(before: S, step: Step[S, O, F]): Boolean =
      NativeRuntime.installed.call[Boolean](this, before, step)

  final class Stays[S, O, F](
      val condition: S => Boolean,
      val release: Option[Step[S, O, F] => Boolean]
  ) extends ((S, Step[S, O, F]) => Boolean):
    def apply(before: S, step: Step[S, O, F]): Boolean =
      NativeRuntime.installed.call[Boolean](this, before, step)

  final class Keeps[S, O, F, V](
      val over: S => Boolean,
      val projection: S => V,
      val equivalent: (V, V) => Boolean
  ) extends ((S, Step[S, O, F]) => Boolean):
    def apply(before: S, step: Step[S, O, F]): Boolean =
      NativeRuntime.installed.call[Boolean](this, before, step)

  final class Sticky[S, O, F](
      val same: Option[Step[S, O, F] => Boolean],
      val across: Option[(S, Step[S, O, F]) => Boolean]
  ) extends ((Boolean, S, Step[S, O, F]) => Boolean):
    def apply(broken: Boolean, before: S, step: Step[S, O, F]): Boolean =
      NativeRuntime.installed.stickyHolds(this, broken, before, step)

  final class StickyVar(val promise: () => Boolean) extends (Boolean => Boolean):
    def apply(broken: Boolean): Boolean = NativeRuntime.installed.call[Boolean](this, broken)
  // A retained effect callback read with its action's declared input.
  final class Adapted[S, O, F](val decl: ActionDecl, val function: AnyRef)
      extends InputEffect[S, O, F]:
    require(
      decl.nativeInputs.nonEmpty || decl.inputs.isEmpty,
      s"action ${decl.name} has no declared input shape"
    )
    override def budgets: Vector[NativeBudget[?]] = budgetsOf(function)
    override def nativeChoices: Option[Vector[Choice]] = choiceTokens(function)
    override def generatedCrash: Boolean = generatedCrashOf(function)
    def apply(state: S, values: List[Any]): List[Step[S, O, F]] =
      NativeRuntime.installed.evaluateEffect(this, state, values)
    override def apply(state: S, cls: Class): List[Step[S, O, F]] =
      NativeRuntime.installed.effectOn(this, state, cls)

  // An effect of the state alone.
  final class OfState[S, O, F](val function: S => List[Step[S, O, F]]) extends InputEffect[S, O, F]:
    override def budgets: Vector[NativeBudget[?]] = budgetsOf(function)
    override def nativeChoices: Option[Vector[Choice]] = choiceTokens(function)
    override def generatedCrash: Boolean = generatedCrashOf(function)
    def apply(state: S, values: List[Any]): List[Step[S, O, F]] =
      NativeRuntime.installed.evaluateEffect(this, state, values)

  // An effect of the state and a class's inputs, as written.
  final class Plain[S, O, F](val function: (S, List[Any]) => List[Step[S, O, F]])
      extends InputEffect[S, O, F]:
    def apply(state: S, values: List[Any]): List[Step[S, O, F]] =
      NativeRuntime.installed.evaluateEffect(this, state, values)

  // An effect that consumes the budgets selected.
  final class Budgeted[S, O, F](val effect: InputEffect[S, O, F], selected: Vector[NativeBudget[?]])
      extends InputEffect[S, O, F]:
    override val budgets: Vector[NativeBudget[?]] = selected
    override val nativeChoices: Option[Vector[Choice]] = effect.nativeChoices
    override val generatedCrash: Boolean = effect.generatedCrash
    def apply(state: S, values: List[Any]): List[Step[S, O, F]] =
      NativeRuntime.installed.evaluateEffect(this, state, values)
    override def apply(state: S, cls: Class): List[Step[S, O, F]] =
      NativeRuntime.installed.effectOn(this, state, cls)

  // A transition block of a Machine's vars: its steps are the paths of the block's run from a
  // state, where the run reaches an effect call or a rejection.
  final class Run[O, F](val machine: Model, val accepted: O, val body: () => Unit)
      extends InputEffect[State, O, F]:
    override val nativeChoices: Option[Vector[Choice]] = choiceTokens(body)
    def apply(state: State, values: List[Any]): List[Step[State, O, F]] =
      NativeRuntime.installed.evaluateEffect(this, state, values)

  // A transition block of a Machine's vars that reads its action's input.
  final class InputRun[O, F, I](
      val decl: ActionDecl,
      val shape: ActionInputs[?],
      val machine: Model,
      val accepted: O,
      val body: I => Unit
  ) extends InputEffect[State, O, F]:
    override val nativeChoices: Option[Vector[Choice]] = choiceTokens(body)
    def apply(state: State, values: List[Any]): List[Step[State, O, F]] =
      NativeRuntime.installed.evaluateEffect(this, state, values)
    override def apply(state: State, cls: Class): List[Step[State, O, F]] =
      NativeRuntime.installed.effectOn(this, state, cls)

  // A call of an effect of a Machine's vars, which takes one step: the rejection it makes, or the
  // accepted outcome into the vars it assigned.
  final class VarStep[O, F](val vars: Vars, val accepted: O, val body: () => Unit)
      extends (State => List[Step[State, O, F]]):
    def apply(state: State): List[Step[State, O, F]] =
      NativeRuntime.installed.call[List[Step[State, O, F]]](this, state)

  // A call of an effect of a Machine's vars on the action's one input.
  final class VarInputStep[I, O, F](val vars: Vars, val accepted: O, val body: I => Unit)
      extends ((State, I) => List[Step[State, O, F]]):
    def apply(state: State, input: I): List[Step[State, O, F]] =
      NativeRuntime.installed.call[List[Step[State, O, F]]](this, state, input)

  // A binding's step function that consumes the budgets selected, of the state alone.
  final class BudgetedState[S, O, F](val effect: AnyRef, val budgets: Vector[NativeBudget[?]])
      extends (S => List[Step[S, O, F]]),
        BudgetMetadata:
    val nativeChoices: Option[Vector[Choice]] = choiceTokens(effect)
    override val generatedCrash: Boolean = generatedCrashOf(effect)
    def apply(state: S): List[Step[S, O, F]] =
      NativeRuntime.installed.call[List[Step[S, O, F]]](this, state)

  // A binding's step function that consumes the budgets selected, of the state and the input.
  final class BudgetedInput[S, I, O, F](val effect: AnyRef, val budgets: Vector[NativeBudget[?]])
      extends ((S, I) => List[Step[S, O, F]]),
        BudgetMetadata:
    val nativeChoices: Option[Vector[Choice]] = choiceTokens(effect)
    override val generatedCrash: Boolean = generatedCrashOf(effect)
    def apply(state: S, input: I): List[Step[S, O, F]] =
      NativeRuntime.installed.call[List[Step[S, O, F]]](this, state, input)

  // A guard that holds where both hold: `where(g).where(h)`.
  final class Both[S](val left: S => Boolean, val right: S => Boolean) extends (S => Boolean):
    def apply(state: S): Boolean =
      NativeRuntime.installed.call[Boolean](this, state)

  // A guard that holds where both hold, one of them reading the action's input.
  final class BothOnInput[S](val left: S => Boolean, val right: S => Boolean) extends InputGuard[S]:
    def apply(state: S, cls: Class): Boolean = NativeRuntime.installed.holds(this, state, cls)

  // A guard of the state and the action's declared input.
  final class OnInput[S, A](val shape: ActionInputs[?], val condition: (S, A) => Boolean)
      extends InputGuard[S]:
    def apply(state: S, cls: Class): Boolean = NativeRuntime.installed.holds(this, state, cls)

  // A guard that holds where an effect takes a step.
  final class Fires[S](val effect: InputEffect[S, ?, ?]) extends InputGuard[S]:
    def apply(state: S, cls: Class): Boolean = NativeRuntime.installed.holds(this, state, cls)

  // A condition of a Machine's vars and the action's input, read with the state restored into the
  // vars.
  final class VarCondition[A](val vars: Vars, val condition: A => Boolean)
      extends ((State, A) => Boolean):
    def apply(state: State, input: A): Boolean =
      NativeRuntime.installed.call[Boolean](this, state, input)

  // A step function a binding may give its budgets: its own where it has none to inherit.
  def budgeted[S, O, F](
      declaration: ActionDecl,
      effect: AnyRef,
      selected: Vector[NativeBudget[?]]
  ): AnyRef =
    if selected.isEmpty then effect
    else if declaration.nativeInputs.nonEmpty then BudgetedInput[S, Any, O, F](effect, selected)
    else BudgetedState[S, O, F](effect, selected)

  // A plain effect of the state and a class's inputs as an InputEffect.
  def inputEffect[S, O, F](value: (S, List[Any]) => List[Step[S, O, F]]): InputEffect[S, O, F] =
    value match
      case adapted: InputEffect[S, O, F] @unchecked => adapted
      case other                                    => Plain(other)
