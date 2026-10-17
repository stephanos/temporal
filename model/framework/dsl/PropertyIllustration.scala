package framework.dsl

import scala.annotation.publicInBinary
import scala.util.control.NonFatal

enum IllustrationKind derives CanEqual:
  case positive, counterexample

final class Illustration[S, O, F] private[framework] (
    val label: String,
    val because: String,
    val kind: IllustrationKind,
    val action: Class,
    val before: Option[S],
    val step: Step[S, O, F],
    val origin: String
)

final class SameStepProperty[S, O, F] private[framework] (
    declaration: PropertyDecl,
    individual: () => Option[RecordMachine[S, O, F]],
    val illustrations: Vector[Illustration[S, O, F]]
) extends Property(declaration):
  inline def illustrate(
      label: String,
      because: String,
      kind: IllustrationKind,
      action: Class,
      step: Step[S, O, F]
  ): SameStepProperty[S, O, F] =
    appended(
      label,
      because,
      kind,
      action,
      step,
      NativeLocation.here,
      StructuralValue.capture[S],
      StructuralValue.domain[O],
      StructuralValue.domain[F]
    )

  @publicInBinary private[framework] def appended(
      label: String,
      because: String,
      kind: IllustrationKind,
      action: Class,
      step: Step[S, O, F],
      origin: String,
      states: StructuralValue[S],
      outcomes: StructuralValue[O],
      facts: StructuralValue[F]
  ): SameStepProperty[S, O, F] =
    val value = new Illustration(label, because, kind, action, None, step, origin)
    val next = IllustrationAdmission.append(
      decl,
      illustrations,
      value,
      individual(),
      states,
      outcomes,
      facts
    )
    new SameStepProperty(IllustrationAdmission.copy(decl, next), individual, next)

final class TransitionProperty[S, O, F] private[framework] (
    declaration: PropertyDecl,
    individual: () => Option[RecordMachine[S, O, F]],
    val illustrations: Vector[Illustration[S, O, F]]
) extends Property(declaration):
  inline def illustrate(
      label: String,
      because: String,
      kind: IllustrationKind,
      action: Class,
      before: S,
      step: Step[S, O, F]
  ): TransitionProperty[S, O, F] =
    appended(
      label,
      because,
      kind,
      action,
      before,
      step,
      NativeLocation.here,
      StructuralValue.capture[S],
      StructuralValue.domain[O],
      StructuralValue.domain[F]
    )

  @publicInBinary private[framework] def appended(
      label: String,
      because: String,
      kind: IllustrationKind,
      action: Class,
      before: S,
      step: Step[S, O, F],
      origin: String,
      states: StructuralValue[S],
      outcomes: StructuralValue[O],
      facts: StructuralValue[F]
  ): TransitionProperty[S, O, F] =
    val value = new Illustration(label, because, kind, action, Some(before), step, origin)
    val next = IllustrationAdmission.append(
      decl,
      illustrations,
      value,
      individual(),
      states,
      outcomes,
      facts
    )
    new TransitionProperty(IllustrationAdmission.copy(decl, next), individual, next)

private[framework] object IllustrationAdmission:
  def copy[S, O, F](decl: PropertyDecl, values: Vector[Illustration[S, O, F]]): PropertyDecl =
    PropertyDecl(decl.name, decl.machine, decl.when, decl.holds, decl.holds2, values)

  def append[S, O, F](
      decl: PropertyDecl,
      previous: Vector[Illustration[S, O, F]],
      value: Illustration[S, O, F],
      individual: Option[RecordMachine[S, O, F]],
      states: StructuralValue[S],
      outcomes: StructuralValue[O],
      facts: StructuralValue[F]
  ): Vector[Illustration[S, O, F]] =
    val context = s"Property ${decl.name} illustration ${value.label} at ${value.origin}"
    try
      require(value.label != null && value.label.trim.nonEmpty, "label must be nonblank")
      require(value.because != null && value.because.trim.nonEmpty, "because must be nonblank")
      require(value.kind != null, "kind must be explicitly positive or counterexample")
      require(!previous.exists(_.label == value.label), "duplicate illustration label")
      val owner = individual.getOrElse:
        throw IllegalArgumentException(
          "illustrations require an individual machine; composed owners are unsupported"
        )
      require(owner eq decl.machine, "illustration belongs to another machine")
      require(value.action != null && value.step != null, "action and step must be supplied")
      require(
        owner.table.exists((action, _) => action eq value.action.decl),
        "action is not bound by this machine"
      )
      value.action.decl.nativeInputs match
        case Some(shape) => shape.input(value.action): Unit
        case None        =>
          require(
            value.action.decl.inputs.isEmpty && value.action.values.isEmpty,
            "action class has invalid inputs"
          )
      value.before.foreach(states.validate(_, owner, s"$context before state"))
      states.validate(value.step.state, owner, s"$context resulting state")
      outcomes.validate(value.step.outcome, owner, s"$context outcome")
      require(
        owner.fo.values.exists(owner.fo.equivalent(_, value.step.outcome)),
        "outcome is outside the owning machine domain"
      )
      require(value.step.facts != null, "facts must be supplied")
      value.step.facts.foreach: fact =>
        facts.validate(fact, owner, s"$context fact")
        require(
          owner.ff.values.exists(owner.ff.equivalent(_, fact)),
          "fact is outside the owning machine domain"
        )
      previous :+ value
    catch
      case NonFatal(error) =>
        throw IllegalArgumentException(s"$context: ${error.getMessage}", error)
