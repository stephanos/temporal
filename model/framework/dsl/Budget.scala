package framework.dsl

import scala.annotation.{publicInBinary, targetName, unused}
import scala.quoted.*

// The budget of one fault of a machine: the state field it is counted in, which the native runtime
// consumes on each fault step.
final private[framework] class NativeBudget[S](
    val owner: RecordMachine[S, ?, ?],
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
  transparent inline def budgetedBy[A <: Boolean | Int](
      inline selector: S => A
  ): BudgetedAction[S, I] = ${ BudgetedAction.captureAppend('this, 'selector) }

  @publicInBinary private[framework] def append[A](field: StateField[S, A]): BudgetedAction[S, I] =
    val owner =
      budgets.head.owner
        .asInstanceOf[RecordMachine[S, ?, ?]] // scalafix:ok DisableSyntax.asInstanceOf
    new BudgetedAction(action, budgets :+ new NativeBudget(owner, action.decl, field))

  private[framework] infix def ~>[O, F](effect: S => List[Step[S, O, F]])(using
      @unused empty: I =:= EmptyTuple
  ): StepBinding[S, O, F] =
    require(
      action.decl.nativeInputs.isEmpty,
      "a state-only binding requires an action without inputs"
    )
    StepBinding(action.decl, Programs.budgeted[S, O, F](action.decl, effect, budgets))

  @targetName("budgetedInputBinding")
  private[framework] infix def ~>[O, F](
      effect: (S, I) => List[Step[S, O, F]]
  ): StepBinding[S, O, F] =
    require(action.decl.nativeInputs.nonEmpty, "an input binding requires a declared input")
    StepBinding(action.decl, Programs.budgeted[S, O, F](action.decl, effect, budgets))

final class BudgetedScalarAction[S, I] @publicInBinary private[framework] (
    action: Action[I],
    budgets: Vector[NativeBudget[?]]
) extends BudgetedAction[S, I](action, budgets)

@publicInBinary
private[framework] object BudgetedAction:
  def capture[S: Type, I: Type, A: Type, O: Type, F: Type](
      action: Expr[Action[I]],
      selector: Expr[S => A],
      owner: Expr[Owner[S, O, F]]
  )(using Quotes): Expr[BudgetedAction[S, I]] =
    import quotes.reflect.*
    val field = StateField.captured(selector)
    if action.asTerm.underlyingArgument.tpe.widen <:< TypeRepr.of[ScalarAction[I]] then
      '{
        val declared = $action
        new BudgetedScalarAction(
          declared,
          Vector(new NativeBudget($owner.machine, declared.decl, $field))
        )
      }
    else
      '{
        val declared = $action
        new BudgetedAction(
          declared,
          Vector(new NativeBudget($owner.machine, declared.decl, $field))
        )
      }

  def captureAppend[S: Type, I: Type, A: Type](
      target: Expr[BudgetedAction[S, I]],
      selector: Expr[S => A]
  )(using Quotes): Expr[BudgetedAction[S, I]] =
    import quotes.reflect.*
    val field = StateField.captured(selector)
    if target.asTerm.underlyingArgument.tpe.widen <:< TypeRepr.of[BudgetedScalarAction[S, I]] then
      '{
        val appended = $target.append($field)
        new BudgetedScalarAction(appended.action, appended.budgets)
      }
    else '{ $target.append($field) }
