package framework.dsl

import scala.annotation.publicInBinary

// The budget of one fault of a machine: the state field it is counted in, which the native runtime
// consumes on each fault step.
final private[framework] class NativeBudget[S](
    val owner: MachineDeclaration[S, ?, ?],
    val declaration: ActionDecl,
    val field: StateField[S, ?]
)

private[framework] trait BudgetMetadata:
  def budgets: Vector[NativeBudget[?]]
  def nativeChoices: Option[Vector[Choice]]
  def generatedCrash: Boolean = false

private[framework] def budgetsOf(effect: AnyRef): Vector[NativeBudget[?]] = effect match
  case metadata: BudgetMetadata => metadata.budgets
  case _                        => Vector.empty

private[framework] def generatedCrashOf(effect: AnyRef): Boolean = effect match
  case metadata: BudgetMetadata => metadata.generatedCrash
  case _                        => false

class BudgetedAction[S, I] @publicInBinary private[framework] (
    val action: Action[I],
    private[framework] val budgets: Vector[NativeBudget[?]]
):
  transparent inline def budgetedBy[O, F, A <: Boolean | Int](inline variable: A)(using
      owner: MachineDeclaration[State, O, F]
  ): BudgetedAction[State, I] =
    new BudgetedAction(
      action,
      budgets :+ new NativeBudget(owner, action.decl, StateField.captureVar(variable, owner))
    )

@publicInBinary
private[framework] object BudgetedAction:
  def variable[I, A](
      action: Action[I],
      field: StateField[State, A],
      owner: MachineDeclaration[State, ?, ?]
  ): BudgetedAction[State, I] =
    new BudgetedAction(action, Vector(new NativeBudget(owner, action.decl, field)))
