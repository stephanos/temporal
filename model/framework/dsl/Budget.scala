package framework.dsl

import scala.annotation.{publicInBinary, targetName, unused}
import scala.quoted.*

final private[framework] class NativeBudget[S](
    val owner: Machine[S, ?, ?],
    val declaration: ActionDecl,
    val field: StateField[S, ?]
):
  def consumption(state: S, location: String): Option[(List[String], Any)] =
    val writable = field.asInstanceOf[StateField[S, Any]] // scalafix:ok DisableSyntax.asInstanceOf
    val context =
      s"fault ${declaration.name} at $location budget ${field.path.mkString(".")} at ${field.location}"
    writable.get(state) match
      case available: Boolean =>
        Option.when(available)(field.path -> false)
      case remaining: Int =>
        require(remaining >= 0, s"$context: negative own budget $remaining")
        Option.when(remaining > 0)(field.path -> (remaining - 1))
      case _ => throw IllegalArgumentException(s"$context: unsupported budget value")

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

private[framework] def budgetEffect[S, O, F](
    effect: InputEffect[S, O, F],
    selected: Vector[NativeBudget[?]]
): InputEffect[S, O, F] = new InputEffect[S, O, F] with BudgetMetadata:
  override val budgets: Vector[NativeBudget[?]] = selected
  override val nativeChoices: Option[Vector[Choice]] = effect.nativeChoices
  override val generatedCrash: Boolean = effect.generatedCrash
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
            override val generatedCrash: Boolean = generatedCrashOf(effect)
            def apply(state: S, value: I): List[Step[S, O, F]] =
              callback(state, value)
        adapted(shape)
      case None =>
        val callback =
          effect.asInstanceOf[S => List[Step[S, O, F]]] // scalafix:ok DisableSyntax.asInstanceOf
        new (S => List[Step[S, O, F]]) with BudgetMetadata:
          val budgets: Vector[NativeBudget[?]] = selected
          val nativeChoices: Option[Vector[Choice]] = choiceTokens(effect)
          override val generatedCrash: Boolean = generatedCrashOf(effect)
          def apply(state: S): List[Step[S, O, F]] = callback(state)

class BudgetedAction[S, I] @publicInBinary private[framework] (
    val action: Action[I],
    private[framework] val budgets: Vector[NativeBudget[?]]
):
  transparent inline def budgetedBy[A <: Boolean | Int](
      inline selector: S => A
  ): BudgetedAction[S, I] = ${ BudgetedAction.captureAppend('this, 'selector) }

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
