package framework.dsl

import scala.language.dynamics

final private[framework] case class SuppliedInput[I](value: I)

sealed private[framework] trait ActionInputs[I]:
  def domain: Finite[I]
  def values(value: I): List[Any]
  def defaultAt(index: Int): Option[Any]
  def omitted(decl: ActionDecl): Class
  final def classes: List[List[Any]] = domain.values.toList.map(values)
  final def classOf(decl: ActionDecl, value: I): Class =
    val fields = values(value)
    require(
      domain.values.exists(_.equals(value)),
      s"input value of ${decl.name} is outside its declared domain"
    )
    val result = Class(decl, fields)
    result.supplied = Some(SuppliedInput(value))
    result

private[framework] object ActionInputs:
  final class Scalar[I](val domain: Finite[I], default: Option[I]) extends ActionInputs[I]:
    def values(value: I): List[Any] = List(value)
    def defaultAt(index: Int): Option[Any] = if index == 0 then default else None
    def omitted(decl: ActionDecl): Class = classOf(
      decl,
      default.getOrElse(
        throw IllegalArgumentException(
          s"input ${decl.inputs.head} of ${decl.name} has no default: supply it"
        )
      )
    )

  final class Record[I <: Product](
      val domain: Finite[I],
      defaults: List[() => Option[Any]]
  )(using mirror: scala.deriving.Mirror.ProductOf[I])
      extends ActionInputs[I]:
    def values(value: I): List[Any] = value.productIterator.toList
    def defaultAt(index: Int): Option[Any] = defaults.lift(index).flatMap(_())
    def omitted(decl: ActionDecl): Class =
      val fields = defaults.zipWithIndex.map: (default, index) =>
        default().getOrElse(
          throw IllegalArgumentException(s"input ${decl.inputs(index)} has no default: supply it")
        )
      classOf(decl, mirror.fromProduct(Tuple.fromArray(fields.toArray)))

final class RecordAction[I <: Product] private[framework] (
    declaration: ActionDecl,
    private val shape: ActionInputs.Record[I]
) extends Action[I](declaration, Some(shape)):
  override infix def on(entity: Entity): RecordAction[I] =
    new RecordAction(decl.boundTo(decl.copy(on = Some(entity)), retainBinding = false), shape)
  override infix def creates(entity: Entity): RecordAction[I] =
    new RecordAction(decl.boundTo(decl.copy(creates = Some(entity)), retainBinding = false), shape)
  override infix def results(name: String): RecordAction[I] =
    new RecordAction(decl.boundTo(decl.copy(results = name)), shape)
  override def example(value: I, example: String): RecordAction[I] =
    new RecordAction(
      decl.boundTo(decl.copy(examples = decl.examples :+ ClassExample(value, example))),
      shape
    )

final class ScalarAction[I] private[framework] (
    declaration: ActionDecl,
    private val shape: ActionInputs.Scalar[I]
) extends Action[I](declaration, Some(shape)):
  override infix def on(entity: Entity): ScalarAction[I] =
    new ScalarAction(decl.boundTo(decl.copy(on = Some(entity)), retainBinding = false), shape)
  override infix def creates(entity: Entity): ScalarAction[I] =
    new ScalarAction(decl.boundTo(decl.copy(creates = Some(entity)), retainBinding = false), shape)
  override infix def results(name: String): ScalarAction[I] =
    new ScalarAction(decl.boundTo(decl.copy(results = name)), shape)
  override def example(value: I, example: String): ScalarAction[I] =
    new ScalarAction(
      decl.boundTo(decl.copy(examples = decl.examples :+ ClassExample(value, example))),
      shape
    )
