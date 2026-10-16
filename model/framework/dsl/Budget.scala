package framework.dsl

import scala.annotation.{publicInBinary, targetName, unused}
import scala.quoted.*

final private[framework] class NativeBudget[S](
    val owner: Machine[S, ?, ?],
    val declaration: ActionDecl,
    val field: StateField[S, ?]
)

private[framework] trait BudgetMetadata:
  def budgets: Vector[NativeBudget[?]]
  def nativeChoices: Option[Vector[Choice]]

private[framework] def budgetsOf(effect: AnyRef): Vector[NativeBudget[?]] = effect match
  case metadata: BudgetMetadata => metadata.budgets
  case _                        => Vector.empty

private[framework] def budgetEffect[S, O, F](
    effect: InputEffect[S, O, F],
    selected: Vector[NativeBudget[?]]
): InputEffect[S, O, F] = new InputEffect[S, O, F] with BudgetMetadata:
  override val budgets: Vector[NativeBudget[?]] = selected
  override val nativeChoices: Option[Vector[Choice]] = effect.nativeChoices
  def apply(state: S, values: List[Any]): List[Step[S, O, F]] = effect(state, values)
  override def apply(state: S, cls: Class): List[Step[S, O, F]] = effect(state, cls)

private[framework] def budgetedFunction[S, O, F](
    declaration: ActionDecl,
    effect: AnyRef,
    selected: Vector[NativeBudget[?]]
): AnyRef =
  if selected.isEmpty then effect
  else
    declaration.nativeInputs match
      case Some(shape) =>
        def adapted[I](@unused input: ActionInputs[I]): AnyRef =
          val callback = effect
            .asInstanceOf[(S, I) => List[Step[S, O, F]]] // scalafix:ok DisableSyntax.asInstanceOf
          new ((S, I) => List[Step[S, O, F]]) with BudgetMetadata:
            val budgets: Vector[NativeBudget[?]] = selected
            val nativeChoices: Option[Vector[Choice]] = choiceTokens(effect)
            def apply(state: S, value: I): List[Step[S, O, F]] =
              callback(state, value)
        adapted(shape)
      case None =>
        val callback =
          effect.asInstanceOf[S => List[Step[S, O, F]]] // scalafix:ok DisableSyntax.asInstanceOf
        new (S => List[Step[S, O, F]]) with BudgetMetadata:
          val budgets: Vector[NativeBudget[?]] = selected
          val nativeChoices: Option[Vector[Choice]] = choiceTokens(effect)
          def apply(state: S): List[Step[S, O, F]] = callback(state)

final class BudgetedAction[S, I] @publicInBinary private[framework] (
    val action: Action[I],
    private[framework] val budgets: Vector[NativeBudget[?]]
):
  inline def budgetedBy[A <: Boolean | Int](inline selector: S => A): BudgetedAction[S, I] =
    append(StateField.capture(selector))

  @publicInBinary private[framework] def append[A](field: StateField[S, A]): BudgetedAction[S, I] =
    val owner =
      budgets.head.owner.asInstanceOf[Machine[S, ?, ?]] // scalafix:ok DisableSyntax.asInstanceOf
    new BudgetedAction(action, budgets :+ new NativeBudget(owner, action.decl, field))

  infix def ~>[O, F](effect: S => List[Step[S, O, F]])(using
      @unused empty: I =:= EmptyTuple
  ): StepBinding[S, O, F] =
    require(
      action.decl.nativeInputs.isEmpty,
      "a state-only binding requires an action without inputs"
    )
    StepBinding(action.decl, budgetedFunction[S, O, F](action.decl, effect, budgets))

  @targetName("budgetedInputBinding")
  infix def ~>[O, F](effect: (S, I) => List[Step[S, O, F]]): StepBinding[S, O, F] =
    require(action.decl.nativeInputs.nonEmpty, "an input binding requires a declared input")
    StepBinding(action.decl, budgetedFunction[S, O, F](action.decl, effect, budgets))

@publicInBinary
private[framework] object BudgetedAction:
  def capture[S: Type, I: Type, A: Type, O: Type, F: Type](
      action: Expr[Action[I]],
      selector: Expr[S => A],
      owner: Expr[Owner[S, O, F]]
  )(using Quotes): Expr[BudgetedAction[S, I]] =
    val field = StateField.captured(selector)
    '{
      val declared = $action
      new BudgetedAction(declared, Vector(new NativeBudget($owner.machine, declared.decl, $field)))
    }
