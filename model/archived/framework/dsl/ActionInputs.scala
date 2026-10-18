package framework.dsl

import scala.language.dynamics

final private[framework] case class SuppliedInput[I](value: I)

sealed private[framework] trait ActionInputs[I]:
  def domain: Finite[I]
  final def values(value: I): List[Any] = NativeRuntime.installed.inputs.values(this, value)
  final def fromValues(values: List[Any]): I =
    NativeRuntime.installed.inputs.fromValues(this, values)
  final def defaultAt(index: Int): Option[Any] =
    NativeRuntime.installed.inputs.defaultAt(this, index)
  final def omitted(decl: ActionDecl): Class = NativeRuntime.installed.inputs.omitted(this, decl)
  final def classes: List[List[Any]] = NativeRuntime.installed.inputs.classes(this)
  final def inputClasses(decl: ActionDecl): List[Class] =
    NativeRuntime.installed.inputs.inputClasses(this, decl)
  final def input(cls: Class): I = NativeRuntime.installed.inputs.input(this, cls)
  final def retained(decl: ActionDecl, value: I): Class =
    NativeRuntime.installed.inputs.retained(this, decl, value)
  final def classOf(decl: ActionDecl, value: I): Class =
    NativeRuntime.installed.inputs.classOf(this, decl, value)

private[framework] object ActionInputs:
  final class Scalar[I](val domain: Finite[I], private[framework] val default: Option[I])
      extends ActionInputs[I]

  final class Record[I <: Product](
      val domain: Finite[I],
      private[framework] val defaults: List[() => Option[Any]],
      private[framework] val fields: List[Finite[?]],
      private[framework] val fieldSelection: Boolean = false
  )(using private[framework] val mirror: scala.deriving.Mirror.ProductOf[I])
      extends ActionInputs[I]

private[framework] trait InputEvaluation:
  def values[I](shape: ActionInputs[I], value: I): List[Any]
  def fromValues[I](shape: ActionInputs[I], values: List[Any]): I
  def defaultAt[I](shape: ActionInputs[I], index: Int): Option[Any]
  def omitted[I](shape: ActionInputs[I], decl: ActionDecl): Class
  def classes[I](shape: ActionInputs[I]): List[List[Any]]
  def inputClasses[I](shape: ActionInputs[I], decl: ActionDecl): List[Class]
  def input[I](shape: ActionInputs[I], cls: Class): I
  def retained[I](shape: ActionInputs[I], decl: ActionDecl, value: I): Class
  def classOf[I](shape: ActionInputs[I], decl: ActionDecl, value: I): Class

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
