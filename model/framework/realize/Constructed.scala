package framework.realize

import framework.dsl.{Action, ActionDecl, ClassRef, MachineDeclaration, NativeRuntime}
import io.temporal.server.api.umpire.v1 as ir
import scala.util.control.NonFatal

// The framework-facing value of any realization kit. A kit owns its conversion; the export driver
// owns dependency lifting, Query selection and final Model assembly.
trait ConstructedRealization:
  def machine: MachineDeclaration[?, ?, ?]
  def runs: Seq[AnyRef]
  protected def declaredPerformances: Vector[Performance]
  protected def declaredWaitedClasses: Vector[ClassRef] =
    throw UnsupportedOperationException("realization kit has no native waited-class projection")
  protected def declaredDependencies: ConstructedRealization.Dependencies
  def toIR(context: ConstructedRealization.Context): ir.Realization

  private var coverageResult =
    Option.empty[Either[Throwable, ConstructedRealization.NativeCoverage]]
  private var acquiringCoverage = false

  final private[framework] def nativeCoverage(
      expectedMachine: MachineDeclaration[?, ?, ?]
  ): ConstructedRealization.NativeCoverage = synchronized:
    require(machine eq expectedMachine, "realization belongs to another exact machine instance")
    def result: ConstructedRealization.NativeCoverage = coverageResult.get.fold(
      throw _,
      coverage =>
        require(
          coverage.owner eq expectedMachine,
          "native coverage belongs to another exact machine instance"
        )
        coverage
    )
    if coverageResult.nonEmpty then result
    else if acquiringCoverage then
      val error = IllegalStateException("native coverage acquisition reentry")
      coverageResult = Some(Left(error))
      throw error
    else
      acquiringCoverage = true
      try
        val performances = nativePerformances
        val declarations = expectedMachine.table.map(_._1)
        var catalogs = Vector.empty[(ActionDecl, Vector[framework.dsl.Class])]
        def validate(cls: framework.dsl.Class): Unit =
          ConstructedRealization.validateOwner(declarations, cls)
          if !catalogs.exists((decl, _) => decl eq cls.decl) then
            val decl = cls.decl
            require(
              decl.composedClasses.nonEmpty ||
                (decl.inputs.size == decl.domains.size &&
                  (decl.nativeInputs.nonEmpty || decl.inputs.isEmpty)),
              s"action ${decl.name} has a malformed native input catalog"
            )
            val classes = NativeRuntime.installed.classesOf(decl).toVector
            require(
              decl.composedClasses.nonEmpty || classes.forall(_.values.size == decl.inputs.size),
              s"action ${decl.name} has a malformed native input catalog"
            )
            catalogs :+= decl -> classes
          ConstructedRealization.validateClass(catalogs, cls)
        val performed = performances.map(p => ConstructedRealization.exactClass(p.step))
        performed.foreach(validate)
        val awaited = declaredWaitedClasses.map(ConstructedRealization.exactClass)
        awaited.foreach(validate)
        require(
          machine eq expectedMachine,
          "realization changed its exact machine instance during native coverage acquisition"
        )
        if coverageResult.isEmpty then
          coverageResult = Some(
            Right(
              new ConstructedRealization.NativeCoverage(
                expectedMachine,
                this,
                performed,
                awaited,
                declarations,
                catalogs
              )
            )
          )
        result
      catch
        case NonFatal(error) =>
          if coverageResult.isEmpty then coverageResult = Some(Left(error))
          result
      finally acquiringCoverage = false

  final def nativePerformances: Vector[Performance] =
    val performances = declaredPerformances
    machine.validateNativeDeclarations()
    performances.foreach: performance =>
      performance.validateDeclaration()
      performance.selectedChoices.foreach: selected =>
        val cls = performance.step match
          case reference: framework.dsl.Class => reference
          case action: Action[?]              => action.omittedClass
        machine.validateNativeChoices(cls, selected, performance.declarationLocation)
    performances

  final def dependencies: ConstructedRealization.Dependencies =
    val _ = nativePerformances
    declaredDependencies

  final protected def actionDependency(
      value: ClassRef | Action[?]
  ): ConstructedRealization.ActionDependency =
    ConstructedRealization.actionDependency(value)

  final protected def factName(
      value: Fact,
      context: ConstructedRealization.Context
  ): String =
    ConstructedRealization.factName(machine, value, context)

  final protected def commandToIR(
      value: Command | Instruction,
      placeholder: Boolean,
      declaration: String
  ): ir.Command =
    ConstructedRealization.commandToIR(value, placeholder, declaration)

  final protected def commandId(
      value: Command | Instruction,
      declaration: String
  ): String = ConstructedRealization.commandId(value, declaration)

object ConstructedRealization:
  final private[framework] class NativeCoverage private[realize] (
      val owner: MachineDeclaration[?, ?, ?],
      val realization: ConstructedRealization,
      val performed: Vector[framework.dsl.Class],
      val awaited: Vector[framework.dsl.Class],
      declarations: Vector[ActionDecl],
      catalogs: Vector[(ActionDecl, Vector[framework.dsl.Class])]
  ):
    def admit(alternatives: Vector[Vector[ClassRef]]): Unit =
      require(
        realization.machine eq owner,
        "realization belongs to another exact machine instance"
      )
      require(alternatives.nonEmpty, "a native path requires alternatives")
      alternatives.foreach: path =>
        path.foreach: ref =>
          val cls = exactClass(ref)
          validateOwner(declarations, cls)
          if catalogs.exists((decl, _) => decl eq cls.decl) then validateClass(catalogs, cls)
          require(
            (performed ++ awaited).exists(sameClass(_, cls)),
            s"class ${cls.decl.name}${cls.values} is neither performed nor awaited"
          )

  private def exactClass(value: ClassRef | Action[?]): framework.dsl.Class = value match
    case cls: framework.dsl.Class => cls
    case action: Action[?]        => action.omittedClass

  private def sameClass(left: framework.dsl.Class, right: framework.dsl.Class): Boolean =
    (left.decl eq right.decl) && left.values.equals(right.values)

  private def validateOwner(declarations: Vector[ActionDecl], cls: framework.dsl.Class): Unit =
    require(
      declarations.exists(_ eq cls.decl),
      s"class action ${cls.decl.name} has a foreign owner"
    )

  private def validateClass(
      catalogs: Vector[(ActionDecl, Vector[framework.dsl.Class])],
      cls: framework.dsl.Class
  ): Unit =
    val catalog = catalogs
      .find((decl, _) => decl eq cls.decl)
      .map(_._2)
      .getOrElse(
        throw IllegalArgumentException(s"class action ${cls.decl.name} has a foreign owner")
      )
    require(
      catalog.exists(sameClass(_, cls)),
      s"class ${cls.decl.name}${cls.values} inputs are outside the native catalog"
    )

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

  // The generic Command shell a kit fills with its own instruction variants. A placeholder keeps
  // the command's retained name while generic conversion handles every shared Command field.
  private[framework] def commandToIR(
      value: Command | Instruction,
      placeholder: Boolean,
      declaration: String
  ): ir.Command = constructedCommandToIR(value, placeholder, declaration)

  // The retained name of a Command or bare Instruction, without interpreting a kit instruction.
  private[framework] def commandId(value: Command | Instruction, declaration: String): String =
    constructedCommandId(value, declaration)

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
      machine: MachineDeclaration[?, ?, ?],
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
