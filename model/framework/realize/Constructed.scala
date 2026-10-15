package framework.realize

import framework.dsl.{Action, ActionDecl, ClassRef, Machine}
import io.temporal.server.api.umpire.v1 as ir

// The framework-facing value of any realization kit. A kit owns its conversion; the export driver
// owns dependency lifting, Query selection and final Model assembly.
trait ConstructedRealization:
  def machine: Machine[?, ?, ?]
  def runs: Seq[AnyRef]
  def dependencies: ConstructedRealization.Dependencies
  def toIR(context: ConstructedRealization.Context): ir.Realization

  final protected def actionDependency(
      value: ClassRef | Action[?]
  ): ConstructedRealization.ActionDependency =
    ConstructedRealization.actionDependency(value)

  final protected def factName(
      value: Fact,
      context: ConstructedRealization.Context
  ): String =
    ConstructedRealization.factName(machine, value, context)

object ConstructedRealization:
  final case class Context(
      declaration: String,
      metadata: ir.Model,
      sourcePackage: Option[String]
  )

  enum ActionDependency:
    case Declaration(identity: String)
    case Channel(identity: String, operation: String)

  final case class Dependencies(
      actions: Vector[ActionDependency],
      channels: Vector[String]
  ):
    def ++(other: Dependencies): Dependencies =
      Dependencies(actions ++ other.actions, (channels ++ other.channels).distinct.sorted)

  object Dependencies:
    val empty: Dependencies = Dependencies(Vector.empty, Vector.empty)

  def dependencies(value: Realization): Dependencies =
    val actions = value.sourceActions.map(actionDependency)
    Dependencies(actions, value.sourceChannels.distinct.sorted)

  private[realize] def actionDependency(
      value: ClassRef | Action[?]
  ): ActionDependency =
    val declaration = value match
      case action: Action[?]              => action.decl
      case reference: framework.dsl.Class => reference.decl
    actionDependency(declaration)

  private def actionDependency(declaration: ActionDecl): ActionDependency =
    declaration.sourceChannel match
      case Some((channel, operation)) => ActionDependency.Channel(channel, operation)
      case None                       =>
        val identity = declaration.sourceBinding
          .orElse(declaration.sourceIdentity)
          .getOrElse:
            throw IllegalArgumentException(
              s"action ${declaration.name} has no retained source identity"
            )
        ActionDependency.Declaration(identity)

  private def factName(
      machine: Machine[?, ?, ?],
      value: Fact,
      context: Context
  ): String =
    new Construction(
      context.declaration,
      context.metadata,
      machine.name,
      Some(machine.ff.values),
      machine.factCases
    ).fact(value)
