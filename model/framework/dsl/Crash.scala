package framework.dsl

import scala.annotation.publicInBinary

final private[framework] class NativeCrash[S, O, F](
    owner: Machine[S, O, F],
    declaration: ActionDecl,
    outcome: O,
    facts: List[F],
    val budgets: Vector[NativeBudget[?]],
    location: String
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
    val consume = budgets.headOption.fold(Option(Vector.empty[(List[String], Any)])): budget =>
      budget
        .asInstanceOf[NativeBudget[S]]
        .consumption(state, location)
        .map(Vector(_)) // scalafix:ok DisableSyntax.asInstanceOf
    consume.fold(Nil): replacements =>
      List(Step(outcome, owner.resetAfterCrash(state, replacements), facts))

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
