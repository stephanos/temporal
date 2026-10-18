package framework.dsl

import scala.annotation.publicInBinary

// A generated crash's step function: from any state, the outcome and facts it declares into the
// state its durability keeps, consuming its budget, which the native runtime takes.
final private[framework] class NativeCrash[S, O, F](
    private[framework] val owner: RecordMachine[S, O, F],
    private[framework] val declaration: ActionDecl,
    private[framework] val outcome: O,
    private[framework] val facts: List[F],
    val budgets: Vector[NativeBudget[?]],
    private[framework] val location: String
) extends (S => List[Step[S, O, F]]),
      BudgetMetadata:
  val nativeChoices: Option[Vector[Choice]] = None
  override val generatedCrash: Boolean = true
  require(
    declaration.nativeInputs.isEmpty,
    s"crashes at $location requires an action without inputs"
  )
  require(
    declaration.faultKind.contains(FaultKind.crash),
    s"crashes at $location requires a crash fault"
  )
  def apply(state: S): List[Step[S, O, F]] =
    NativeRuntime.installed.call[List[Step[S, O, F]]](this, state)

@publicInBinary
private[framework] def crashBinding[S, O, F](
    owner: Owner[S, O, F],
    action: Action[EmptyTuple],
    budgets: Vector[NativeBudget[?]],
    outcome: O,
    facts: Seq[F],
    location: String
): StepBinding[S, O, F] =
  StepBinding(
    action.decl,
    new NativeCrash(owner.machine, action.decl, outcome, facts.toList, budgets, location)
  )
