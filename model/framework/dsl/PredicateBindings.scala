package framework.dsl

private[framework] trait PredicateBindings:
  def validate(owner: Model): Unit
  def sourceParameters: Vector[String]
  // The predicate the reify plugin captured at its expectation block, when it compiled one.
  def captured: Option[Captured[AnyRef]] = None

final private[framework] class MemberBinding(
    val field: String,
    val machine: Model
):
  def validate(owner: Model): Unit =
    val declared = owner match
      case composition: Composition[?] => composition.memberFields
      case _ => throw IllegalArgumentException("a selected member fact belongs to a composition")
    require(
      declared.exists((name, model) => name == field && (model eq machine)),
      s"$field is not bound to ${machine.name} in ${owner.name}"
    )

@scala.annotation.publicInBinary
private[framework] object PredicateBindings:
  @scala.annotation.publicInBinary
  private[framework] def block[T](
      body: T,
      members: Vector[MemberBinding],
      parameters: Vector[String],
      predicates: Vector[AnyRef]
  ): BlockPredicate[T] = new BlockPredicate(body, members, parameters, predicates)

  @scala.annotation.publicInBinary
  private[framework] def member(field: String, machine: Model): MemberBinding =
    new MemberBinding(field, machine)

  // The captured predicate of an expectation, `expect: ...` compiled by the reify plugin.
  def captured(predicate: AnyRef): Option[Captured[AnyRef]] = predicate match
    case retained: PredicateBindings => retained.captured
    case _                           => None

  def validate(predicate: AnyRef, owner: Model): Unit = predicate match
    case retained: PredicateBindings => retained.validate(owner)
    case _                           => ()

final private[framework] class BlockPredicate[
    T
] @scala.annotation.publicInBinary private[framework] (
    val body: T,
    val members: Vector[MemberBinding],
    val parameters: Vector[String],
    val predicates: Vector[AnyRef]
)
