package temporal.realize

import java.io.ByteArrayInputStream
import java.lang.reflect.Modifier
import scala.collection.mutable
import com.google.protobuf.empty.Empty
import io.grpc.MethodDescriptor
import io.temporal.server.api.umpire.v1 as ir
import scalapb.GeneratedMessage
import scalapb.descriptors.{
  Descriptor,
  FieldDescriptor,
  PMessage,
  PRepeated,
  PString,
  PValue,
  ScalaType
}
import framework.dsl.{Action, ActionDecl, Class, ClassRef}
import framework.realize.*

final private[realize] class TemporalConversion(value: Realizes[?, ?, ?]):
  private val sectionOrder =
    Vector("controller", "workers", "evidence", "serverSteps", "controls")
  private val derivedSectionOrder = Vector("changes")
  private val boundRoles =
    Set("temporal.workflow-service", "temporal.task-queue")

  final private case class Sections(
      controller: Vector[Item] = Vector.empty,
      workers: Vector[Script] = Vector.empty,
      evidence: Vector[Evidence | EvidenceRef[?, ?] | TypedEvidence[?]] = Vector.empty,
      serverSteps: Vector[
        ServerStep | ActivityExternalSettlement[?, ?] | ActivityExternalSettlement.Scheduled |
          ActivityResetSettlement
      ] = Vector.empty,
      controls: Vector[Actuator] = Vector.empty
  )

  final private case class Assembly(
      learned: Vector[Learned],
      observations: Vector[Observed],
      requiredSettings: Vector[RequiredSetting],
      sections: Sections
  )

  final private case class DeadlineCatalog(
      classes: mutable.LinkedHashMap[ActionDecl, Vector[Class]] = mutable.LinkedHashMap.empty,
      bases: mutable.LinkedHashMap[(String, Int), ir.TimeoutBasis] = mutable.LinkedHashMap.empty
  )

  def dependencies: ConstructedRealization.Dependencies =
    val assembly =
      assemble(value, value.getClass.getName.stripSuffix("$"), None)
    val core = dependencyCore(value, assembly)
    val base = ConstructedRealization.dependencies(core)
    val seen =
      mutable.LinkedHashSet.empty[ConstructedRealization.ActionDependency]
    val additional = additionalActions(assembly.sections)
      .map(value.temporalActionDependency)
      .filter(seen.add)
    base ++ ConstructedRealization.Dependencies(additional, Vector.empty)

  def convert(context: ConstructedRealization.Context): ir.Realization =
    val family = context.sourcePackage
      .filter(_.nonEmpty)
      .getOrElse(
        fail(context.declaration, "family has no declaring package context")
      )
    val assembly = assemble(value, context.declaration, Some(context))
    val machine = context.metadata.machines
      .find(_.name == value.machine.name)
      .getOrElse(
        fail(
          context.declaration,
          s"machine ${value.machine.name} has no metadata"
        )
      )
    val core = genericCore(value, assembly, family, machine.entity)
    val generic = genericToIR(core, context)
    val deadlines = DeadlineCatalog()
    val scripts = scriptsToIR(assembly.sections, core, context, deadlines)
    val evidence =
      assembly.sections.evidence.map(evidenceToIR(_, value, core, context))
    val roles = selectedRoles(scripts, generic.controls)
    val (stated, external, reset) = splitServerSteps(
      assembly.sections.serverSteps
    )
    val serverSteps = serverStepsToIR(
      scripts,
      stated,
      core,
      context,
      deadlines,
      machine
    )
    generic.copy(
      roles = roles,
      scripts = scripts,
      evidence = evidence,
      requiredSettings = assembly.requiredSettings.map(s => ir.RequiredSetting(s.key, s.value)),
      behavior = Some(temporalBehavior.asIR),
      serverSteps = serverSteps,
      externalSettlements = external.map(_.asIR(context)),
      resetSettlements = reset.map(_.asIR(core, context))
    )

  private def fail(declaration: String, message: String): Nothing =
    throw IllegalArgumentException(s"$declaration: $message")

  private def assemble(
      value: Realizes[?, ?, ?],
      declaration: String,
      context: Option[ConstructedRealization.Context]
  ): Assembly = value match
    case derived: DerivesFrom[?, ?, ?] =>
      val inherited = assemble(derived.base, declaration, context)
      val changes =
        sectionsOf[Changes](derived, derivedSectionOrder, declaration)
          .get("changes")
          .toVector
          .flatMap(_.changes)
      val controller = changes.foldLeft(inherited.sections.controller): (items, change) =>
        val target = commandId(change.step, declaration)
        val index = items.indexWhere(
          _.command.exists(c => commandId(c, declaration) == target)
        )
        if index < 0 then
          fail(
            declaration,
            s"the base controller has no step whose command is $target: a derived realization changes the steps its base has"
          )
        if change.replaces then items.patch(index, change.items, 1)
        else items.patch(index + 1, change.items, 0)
      inherited.copy(sections = inherited.sections.copy(controller = controller))
    case _ =>
      val sections = sectionsOf[AnyRef](value, sectionOrder, declaration)
      Assembly(
        value.learned,
        value.observations,
        value.requiredSettings,
        Sections(
          controller = sections.get("controller").toVector.flatMap {
            case section: Controller => section.items
            case _                   => Vector.empty
          },
          workers = sections.get("workers").toVector.flatMap {
            case section: Workers => section.scripts
            case _                => Vector.empty
          },
          evidence = sections.get("evidence").toVector.flatMap {
            case section: Evidences => section.items
            case _                  => Vector.empty
          },
          serverSteps = sections.get("serverSteps").toVector.flatMap {
            case section: ServerSteps => section.steps
            case _                    => Vector.empty
          },
          controls = sections.get("controls").toVector.flatMap {
            case section: Controls => section.actuators
            case _                 => Vector.empty
          }
        )
      )

  private def sectionsOf[A](
      value: Realizes[?, ?, ?],
      allowed: Vector[String],
      declaration: String
  ): Map[String, A] =
    val sectionTypes = Set[java.lang.Class[?]](
      classOf[Controller],
      classOf[Workers],
      classOf[Evidences],
      classOf[ServerSteps],
      classOf[Controls],
      classOf[Changes]
    )
    final case class Accessor(
        name: String,
        resultType: java.lang.Class[?],
        read: () => AnyRef
    )
    val fields = value.getClass.getDeclaredFields.toVector
      .filter(field =>
        Modifier.isPublic(
          field.getModifiers
        ) && !field.isSynthetic && field.getName != "MODULE$"
      )
      .map(field =>
        Accessor(
          field.getName,
          field.getType,
          () =>
            Option(field.get(value).asInstanceOf[AnyRef]).getOrElse(
              field.getType.getField("MODULE$").get(null).asInstanceOf[AnyRef]
            )
        )
      )
    val methods = value.getClass.getDeclaredMethods.toVector
      .filter(method =>
        Modifier.isPublic(
          method.getModifiers
        ) && method.getParameterCount == 0 && !method.isSynthetic
      )
      .map(method =>
        Accessor(
          method.getName,
          method.getReturnType,
          () => method.invoke(value).asInstanceOf[AnyRef]
        )
      )
    val accessors = (fields ++ methods).filter(accessor =>
      sectionTypes.exists(_.isAssignableFrom(accessor.resultType)) ||
        (accessor.resultType.getName.endsWith("$") && !accessor.name.contains(
          "$"
        ))
    )
    accessors.foreach: accessor =>
      if !allowed.contains(accessor.name) then
        fail(
          declaration,
          s"${accessor.name} is no section of a realization object, whose sections are ${allowed.mkString(", ")}, in that order"
        )
    accessors
      .groupBy(_.name)
      .foreach: (name, matching) =>
        if matching.sizeIs > 1 then
          fail(
            declaration,
            s"$name is declared more than once in a realization object"
          )
    val ordered = allowed.flatMap(name => accessors.find(_.name == name))
    ordered
      .map: accessor =>
        accessor.name -> accessor.read().asInstanceOf[A]
      .toMap

  private def dependencyCore(
      value: Realizes[?, ?, ?],
      assembly: Assembly
  ): Realization =
    Realization(
      machine = value.machine,
      producer = "",
      producerVersion = "",
      roles = Vector.empty,
      correlation = Correlation("", "", "", "", 0, 0, 0, 0, 0, 0),
      scripts = Vector(
        controller(assembly.sections.controller*)
      ) ++ assembly.sections.workers,
      learned = assembly.learned,
      observations = assembly.observations,
      evidence = assembly.sections.evidence,
      controls = assembly.sections.controls
    )

  private def additionalActions(
      sections: Sections
  ): Vector[ClassRef | Action[?]] =
    val activations = sections.workers.flatMap(_.activation match
      case activity: WorkerActivation.Activity => activity.starts
      case _                                   => Vector.empty)
    val deadlines =
      (sections.controller ++ sections.workers.flatMap(_.items)).flatMap:
        _.binding.toVector.collect { case binding: DeadlineBindings[?] =>
          binding.action
        }
    val server = sections.serverSteps.flatMap:
      case step: ServerStep               => Vector(step.step)
      case reset: ActivityResetSettlement => Vector(reset.timer)
      case _: ActivityExternalSettlement[?, ?] | _: ActivityExternalSettlement.Scheduled =>
        Vector.empty
    activations ++ deadlines ++ server

  private def genericCore(
      value: Realizes[?, ?, ?],
      assembly: Assembly,
      family: String,
      operation: String
  ): Realization =
    Realization(
      name = objectName(value),
      machine = value.machine,
      producer = family + ".testpilot",
      producerVersion = "1",
      roles = Vector.empty,
      correlation = Correlation(
        projection = family + ".projection",
        run = family + ".scope.run",
        operation = family + ".scope." + operation,
        observation = correlatedEvidence,
        events = 32,
        buffered = 16,
        keys = 8,
        support = 128,
        work = 1000000000,
        eventSize = 512
      ),
      scripts = Vector.empty,
      learned = assembly.learned,
      observations = assembly.observations,
      evidence = Vector.empty,
      controls = assembly.sections.controls,
      cleanup = "cleanup",
      rejectionCodes = rejectionCodes
    )

  private def objectName(value: AnyRef): String =
    val name = value.getClass.getSimpleName.stripSuffix("$")
    name.take(1).toLowerCase + name.drop(1)

  private def scriptsToIR(
      sections: Sections,
      core: Realization,
      context: ConstructedRealization.Context,
      deadlines: DeadlineCatalog
  ): Vector[ir.Script] =
    val scripts = Vector(controller(sections.controller*)) ++ sections.workers
    scripts.map(scriptToIR(_, core, context, deadlines))

  private def scriptToIR(
      value: Script,
      core: Realization,
      context: ConstructedRealization.Context,
      deadlines: DeadlineCatalog
  ): ir.Script =
    val activation = value.activation match
      case _: Activation.Controller.type =>
        ir.Script.Activation.Controller(Empty())
      case value: WorkerActivation.Workflow =>
        ir.Script.Activation.Workflow(
          ir.WorkflowActivation(
            Some(value.workflowType.toIR),
            roleId(value.worker),
            roleId(value.taskQueue)
          )
        )
      case value: WorkerActivation.NexusHandler =>
        ir.Script.Activation.NexusHandler(
          ir.NexusHandlerActivation(
            value.service,
            value.operation,
            roleId(value.worker),
            roleId(value.taskQueue)
          )
        )
      case value: WorkerActivation.Activity =>
        ir.Script.Activation.Activity(
          ir.ActivityActivation(
            Some(value.activityType.toIR),
            roleId(value.worker),
            roleId(value.taskQueue),
            value.starts.map(actionClass(_, core, context))
          )
        )
      case other =>
        fail(
          context.declaration,
          s"${other.getClass.getName} is no Temporal activation"
        )
    ir.Script(
      id = value.id,
      activation = activation,
      items = value.items.flatMap(itemToIR(_, core, context, deadlines))
    )

  private def itemToIR(
      value: Item,
      core: Realization,
      context: ConstructedRealization.Context,
      deadlines: DeadlineCatalog
  ): Vector[ir.Item] = value.binding match
    case Some(binding: DeadlineBindings[?]) =>
      Vector(deadlineItem(binding, core, context, deadlines))
    case Some(other) =>
      fail(
        context.declaration,
        s"${other.getClass.getName} is no Temporal performance binding"
      )
    case None =>
      val when = value.when.flatMap:
        case action: Action[?] =>
          if action.decl.inputs.nonEmpty then
            deadlines.classes.getOrElse(
              action.decl,
              fail(
                context.declaration,
                s"${action.name} has inputs, and no deadlines declaration of the realization before it binds its classes: name each class"
              )
            )
          else Vector(action.asInstanceOf[ClassRef])
        case cls: Class => Vector(cls)
      Vector(
        ir.Item(
          command = value.command.map(commandToIR(_, context)),
          when = when.map(actionClass(_, core, context)),
          performs = value.performs.map(p =>
            ir.Performance(
              step = Some(actionClass(p.step, core, context)),
              command = Some(commandToIR(p.command, context))
            )
          )
        )
      )

  private def actionClass(
      value: ClassRef,
      core: Realization,
      context: ConstructedRealization.Context
  ): ir.ActionClass =
    val probe = core.copy(
      scripts = Vector(
        script("probe", Activation.Controller)(Item(when = Vector(value)))
      )
    )
    genericToIR(probe, context).scripts.head.items.head.when.head

  private def commandToIR(
      value: Command | Instruction,
      context: ConstructedRealization.Context
  ): ir.Command =
    val instruction = value match
      case command: Command         => command.instruction
      case instruction: Instruction => instruction
    val placeholder = instruction match
      case _: WorkerInstruction | _: ReferencedWorkerInstruction => true
      case _                                                     => false
    val generic =
      this.value.temporalCommandToIR(value, placeholder, context.declaration)
    generic.copy(instruction = instructionToIR(instruction, generic.instruction, context))

  private def commandId(
      value: Command | Instruction,
      declaration: String
  ): String =
    this.value.temporalCommandId(value, declaration)

  private def instructionToIR(
      value: Instruction,
      generic: ir.Command.Instruction,
      context: ConstructedRealization.Context
  ): ir.Command.Instruction = value match
    case referenced: ReferencedWorkerInstruction =>
      workerInstructionToIR(referenced.instruction, context)
    case worker: WorkerInstruction       => workerInstructionToIR(worker, context)
    case rpc: Instruction.TypedRpc[?, ?] =>
      generic match
        case ir.Command.Instruction.Rpc(call) =>
          ir.Command.Instruction.Rpc(
            expandRequestBase(rpc.role, rpc.method, call, context)
          )
        case _ =>
          fail(
            context.declaration,
            "generic RPC conversion returned another instruction"
          )
    case poll: Instruction.TypedPoll[?, ?] =>
      generic match
        case ir.Command.Instruction.Poll(call) =>
          ir.Command.Instruction.Poll(
            expandRequestBase(poll, call, context)
          )
        case _ =>
          fail(
            context.declaration,
            "generic poll conversion returned another instruction"
          )
    case _ => generic

  private def workerInstructionToIR(
      value: WorkerInstruction,
      context: ConstructedRealization.Context
  ): ir.Command.Instruction =
    given CanEqual[WorkerInstruction, WorkerInstruction] = CanEqual.derived
    value match
      case WorkerInstruction.AttemptFailure(failure) =>
        ir.Command.Instruction.AttemptFailure(
          ir.AttemptFailure(Some(failure.toIR(context.declaration)))
        )
      case WorkerInstruction.AttemptCanceled =>
        ir.Command.Instruction.AttemptCanceled(Empty())
      case WorkerInstruction.AttemptHeartbeat(details) =>
        ir.Command.Instruction.AttemptHeartbeat(
          ir.AttemptHeartbeat(Some(details.toIR(context.declaration)))
        )
      case WorkerInstruction.AttemptWithheld(mode, external) =>
        ir.Command.Instruction.AttemptWithheld(
          ir.AttemptWithheld(
            ir.WithholdingMode.fromValue(mode.ordinal),
            external.fold("")(commandId(_, context.declaration))
          )
        )
      case WorkerInstruction.AwaitActivityPublication(publication) =>
        ir.Command.Instruction.AwaitActivityPublication(publication.id)
      case WorkerInstruction.Fault(role, kind) =>
        ir.Command.Instruction.Fault(
          ir.Fault(roleId(role), ir.Fault.Kind.fromValue(kind.ordinal + 1))
        )
      case WorkerInstruction.WorkflowCommand(command) =>
        ir.Command.Instruction.WorkflowCommand(
          ir.WorkflowCommand(Some(command.toIR(context.declaration)))
        )
      case WorkerInstruction.NexusReply(reply, binds) =>
        ir.Command.Instruction.NexusReply(
          ir.NexusReply(Some(reply.toIR(context.declaration)), learnedId(binds))
        )
      case WorkerInstruction.NexusCompletion(handle, result) =>
        ir.Command.Instruction.NexusCompletion(
          ir.NexusCompletion(
            learnedId(handle),
            Some(result.toIR(context.declaration))
          )
        )

  private def learnedId(value: String | Learned): String = value match
    case text: String     => text
    case learned: Learned => learned.id

  private def roleId(value: String | Addressee): String = value match
    case text: String    => text
    case role: Addressee => role.id

  private def expandRequestBase[
      Req <: GeneratedMessage,
      Rsp <: GeneratedMessage
  ](
      role: String | Addressee,
      method: MethodDescriptor[Req, Rsp],
      call: ir.Rpc,
      context: ConstructedRealization.Context
  ): ir.Rpc = role match
    case base: RequestBase =>
      val descriptor = requestDescriptor(method, context.declaration)
      call.copy(assign = baseAssignments(base, descriptor, call.assign, context) ++ call.assign)
    case _ => call

  private def expandRequestBase(
      poll: Instruction.TypedPoll[?, ?],
      call: ir.Poll,
      context: ConstructedRealization.Context
  ): ir.Poll = poll.role match
    case base: RequestBase =>
      val method = poll.evidence.evidence.from match
        case source: Recorded.TypedRead[?, ?, ?]   => source.method
        case source: Recorded.TypedSingle[?, ?, ?] => source.method
        case other                                 =>
          fail(
            context.declaration,
            s"${other.getClass.getName} has no request descriptor"
          )
      val descriptor = requestDescriptor(method, context.declaration)
      call.copy(assign = baseAssignments(base, descriptor, call.assign, context) ++ call.assign)
    case _ => call

  private def requestDescriptor(
      method: MethodDescriptor[?, ?],
      declaration: String
  ): Descriptor =
    val parsed = method.getRequestMarshaller.parse(
      ByteArrayInputStream(Array.emptyByteArray)
    )
    parsed match
      case message: GeneratedMessage => message.companion.scalaDescriptor
      case other                     =>
        fail(
          declaration,
          s"${other.getClass.getName} is no ScalaPB request message"
        )

  private def baseAssignments(
      base: RequestBase,
      request: Descriptor,
      own: Seq[ir.Assignment],
      context: ConstructedRealization.Context
  ): Vector[ir.Assignment] = base.fields.toVector.flatMap: (name, value) =>
    val direct = request.findFieldByName(name).map(_.name)
    val nested = request.fields.collect:
      case field
          if messageDescriptor(field).exists(
            _.findFieldByName(name).nonEmpty
          ) =>
        field.name + "." + name
    val path = (direct, nested) match
      case (Some(found), _)   => found
      case (None, Seq(found)) => found
      case _                  =>
        fail(
          context.declaration,
          s"the request base sets $name, which ${request.fullName} has neither as a field nor in one message field"
        )
    val assignment =
      ir.Assignment(path, Some(value.operand.toIR(context.declaration)))
    own.reverseIterator.find(_.target == path) match
      case Some(existing) if existing.equals(assignment) =>
        fail(
          context.declaration,
          s"the call assigns $path the value its request base gives it: drop the assignment"
        )
      case Some(_) => Vector.empty
      case None    => Vector(assignment)

  private def messageDescriptor(field: FieldDescriptor): Option[Descriptor] =
    field.scalaType match
      case ScalaType.Message(descriptor) if !field.isRepeated =>
        Some(descriptor)
      case _ => None

  private def deadlineItem(
      binding: DeadlineBindings[?],
      core: Realization,
      context: ConstructedRealization.Context,
      catalog: DeadlineCatalog
  ): ir.Item =
    val declaration = binding.action match
      case action: Action[?] => action.decl
      case preset: Class     => preset.decl
    val definitionProbe = binding.action match
      case preset: Class => preset
      case _: Action[?]  =>
        Class(declaration, retainedDefaults(declaration, context.declaration))
    val definition = actionClass(definitionProbe, core, context)
    val action = context.metadata.actions
      .find(_.id == definition.action)
      .getOrElse(
        fail(
          context.declaration,
          s"action ${definition.action} has no signature metadata"
        )
      )
    def timeout(parameter: ir.Param): Option[String] =
      parameter.getType.ref match
        case ir.TypeRef.Ref.Named(name) if name.endsWith(".Timeout") =>
          Some(name)
        case _ => None
    val retained = binding.action match
      case preset: Class => preset.values
      case _: Action[?]  =>
        if action.inputs.exists(parameter => timeout(parameter).isEmpty) then
          fail(
            context.declaration,
            s"deadlines for ${action.name} requires an explicit Class preset for non-Timeout inputs"
          )
        retainedDefaults(declaration, context.declaration)
    if retained.size != action.inputs.size then
      fail(
        context.declaration,
        s"deadlines preset of ${action.name} must supply every input"
      )
    action.inputs.zipWithIndex.foreach: (parameter, index) =>
      timeout(parameter).foreach: _ =>
        val unset = domainCase(declaration, index, "unset", context.declaration)
        if !java.util.Objects.equals(retained(index), unset) then
          fail(
            context.declaration,
            s"deadlines preset of ${action.name} must leave Timeout input ${parameter.name} at its default"
          )
    val fields = binding.fields.map: field =>
      val index = declaration.tokens.indexWhere(_.exists(_ eq field.input))
      val parameter = action.inputs.lift(index)
      val typeName = parameter
        .flatMap(timeout)
        .getOrElse(
          fail(
            context.declaration,
            s"${field.input.name} is no Timeout input of ${action.name}: a deadline names one of its action's Timeout inputs"
          )
        )
      val path = field.selection.path.getOrElse(
        fail(
          context.declaration,
          "opaque selector has no captured protobuf path"
        )
      )
      val basis = path.split('.').last match
        case "schedule_to_close_timeout" =>
          ir.TimeoutBasis.TIMEOUT_BASIS_SCHEDULE_TO_CLOSE
        case "start_to_close_timeout" =>
          ir.TimeoutBasis.TIMEOUT_BASIS_START_TO_CLOSE
        case "heartbeat_timeout" => ir.TimeoutBasis.TIMEOUT_BASIS_HEARTBEAT
        case _                   => ir.TimeoutBasis.TIMEOUT_BASIS_UNSPECIFIED
      catalog.bases
        .get(definition.action -> index)
        .foreach: previous =>
          if previous.value != basis.value then
            fail(
              context.declaration,
              s"deadlines for ${action.name} binds input ${parameter.get.name} to different timeout bases"
            )
      catalog.bases(definition.action -> index) = basis
      (index, typeName, path)
    val unsetIndex =
      binding.unset.map((input, _) => declaration.tokens.indexWhere(_.exists(_ eq input)))
    val classes = (0 until (1 << fields.size)).map: mask =>
      val selected = fields.zipWithIndex.collect {
        case (field, bit) if (mask & (1 << bit)) != 0 => field
      }
      val values = retained.indices.map: index =>
        if selected.exists(_._1 == index) then
          domainCase(declaration, index, "expires", context.declaration)
        else retained(index)
      val base = binding.unset match
        case Some((_, command)) if !selected.exists(field => unsetIndex.contains(field._1)) =>
          command
        case _ => binding.call
      val command = selected.foldLeft(commandLikeToIR(base, context)): (command, field) =>
        setDeadline(
          command,
          field._3,
          binding.value.toIR(context.declaration),
          context.declaration
        )
      Class(declaration, values.toList) -> command
    val prior = catalog.classes.getOrElse(declaration, Vector.empty)
    if classes.exists((value, _) => prior.contains(value)) then
      fail(
        context.declaration,
        s"deadlines for ${action.name} binds a class twice"
      )
    catalog.classes(declaration) = prior ++ classes.map(_._1)
    ir.Item(
      performs = classes.map((step, command) =>
        ir.Performance(
          step = Some(actionClass(step, core, context)),
          command = Some(command)
        )
      )
    )

  private def retainedDefaults(
      declaration: ActionDecl,
      owner: String
  ): List[Any] =
    declaration.tokens.zipWithIndex.map:
      case (Some(input), _) =>
        input.default.getOrElse(
          fail(
            owner,
            s"input ${input.name} of ${declaration.name} has no default: supply it"
          )
        )
      case (None, index) =>
        fail(
          owner,
          s"input ${declaration.inputs(index)} of ${declaration.name} has no default: supply it"
        )

  private def domainCase(
      declaration: ActionDecl,
      index: Int,
      name: String,
      owner: String
  ): Any = declaration
    .domains(index)
    .values
    .find:
      case product: Product => product.productPrefix == name
      case other            => other.toString == name
    .getOrElse(fail(owner, s"${declaration.inputs(index)} has no $name value"))

  private def commandLikeToIR(
      value: Command | Instruction,
      context: ConstructedRealization.Context
  ): ir.Command = commandToIR(value, context)

  private def setDeadline(
      command: ir.Command,
      path: String,
      value: ir.Proto,
      declaration: String
  ): ir.Command = command.instruction match
    case ir.Command.Instruction.Rpc(call) =>
      command.copy(instruction =
        ir.Command.Instruction.Rpc(
          call.copy(assign = call.assign ++ assignedMessage(path, value))
        )
      )
    case ir.Command.Instruction.WorkflowCommand(workflow) =>
      val updated = protoSet(
        workflow.getCommand,
        path.split('.').toList.map(_.replaceAll("^[^<]*<(.*)>$", "$1")),
        value,
        declaration
      )
      command.copy(instruction =
        ir.Command.Instruction.WorkflowCommand(
          workflow.copy(command = Some(updated))
        )
      )
    case _ =>
      fail(
        declaration,
        "deadlines sets a field of a call's request or of a workflow command's protobuf"
      )

  private def assignedMessage(
      prefix: String,
      value: ir.Proto
  ): Vector[ir.Assignment] =
    value.fields.toVector.flatMap: field =>
      field.getValue.kind match
        case ir.ProtoValue.Kind.Message(message) =>
          assignedMessage(prefix + "." + field.name, message)
        case _ =>
          Vector(
            ir.Assignment(
              prefix + "." + field.name,
              Some(ir.Operand(kind = ir.Operand.Kind.Literal(field.getValue)))
            )
          )

  private def protoSet(
      value: ir.Proto,
      path: List[String],
      inserted: ir.Proto,
      declaration: String
  ): ir.Proto = path match
    case List(last) =>
      value.copy(fields =
        value.fields :+ ir.ProtoField(
          last,
          Some(ir.ProtoValue(ir.ProtoValue.Kind.Message(inserted)))
        )
      )
    case head :: rest =>
      val index = value.fields.indexWhere(_.name == head)
      if index < 0 then
        fail(
          declaration,
          s"the workflow command's protobuf sets no $head to set a deadline in"
        )
      val field = value.fields(index)
      val nested = field.getValue.kind match
        case ir.ProtoValue.Kind.Message(message) => message
        case _                                   =>
          fail(
            declaration,
            s"the workflow command's protobuf field $head is no message"
          )
      value.copy(fields =
        value.fields.updated(
          index,
          field.copy(value =
            Some(
              ir.ProtoValue(
                ir.ProtoValue.Kind.Message(
                  protoSet(nested, rest, inserted, declaration)
                )
              )
            )
          )
        )
      )
    case Nil => value

  private def evidenceToIR(
      value: Evidence | EvidenceRef[?, ?] | TypedEvidence[?],
      owner: Realizes[?, ?, ?],
      core: Realization,
      context: ConstructedRealization.Context
  ): ir.Evidence = value match
    case typed: TypedEvidence[?]
        if typed.evidence.from
          .isInstanceOf[WorkflowHistory.TypedHistory[?, ?]] =>
      historyEvidenceToIR(typed, owner, core, context)
    case _ =>
      genericToIR(core.copy(evidence = Vector(value)), context).evidence.head

  private def genericToIR(
      value: Realization,
      context: ConstructedRealization.Context
  ): ir.Realization = value.toIR(context)

  private def historyEvidenceToIR(
      value: TypedEvidence[?],
      owner: Realizes[?, ?, ?],
      core: Realization,
      context: ConstructedRealization.Context
  ): ir.Evidence =
    val source =
      value.evidence.from.asInstanceOf[WorkflowHistory.TypedHistory[?, ?]]
    val attributes = source.attributes.path.getOrElse(
      fail(context.declaration, "opaque selector has no captured protobuf path")
    )
    val history = attributes match
      case value if value.contains("<") && value.endsWith(">") =>
        value.substring(value.indexOf('<') + 1, value.length - 1)
      case other => other
    val operation = value.operation.fold("")(
      _.path.getOrElse(
        fail(
          context.declaration,
          "opaque selector has no captured protobuf path"
        )
      )
    ) + source.operationKey.fold("")(key => "." + key)
    ir.Evidence(
      id = value.evidence.id,
      records = owner.temporalFactName(value.evidence.records, context),
      source = value.evidence.source,
      from = ir.Evidence.From.History(history),
      operation = operation,
      commitment = ir.Evidence.Commitment.fromValue(value.evidence.commitment.ordinal + 1),
      fields = value.fields.map(field =>
        ir.EvidenceField(
          id = field.id,
          path = field.path.path
            .getOrElse(
              fail(
                context.declaration,
                "opaque selector has no captured protobuf path"
              )
            ),
          role = field.role.fold(ir.EvidenceField.Role.ROLE_UNSPECIFIED)(role =>
            ir.EvidenceField.Role.fromValue(role.ordinal + 1)
          ),
          redacted = field.redacted
        )
      ),
      exhaustive = value.evidence.exhaustive,
      confirms = value.evidence.confirms.map(taking =>
        ir.Taking(
          step = Some(actionClass(taking.step, core, context)),
          occurrence = taking.occurrence
        )
      )
    )

  private def selectedRoles(
      scripts: Seq[ir.Script],
      controls: Seq[ir.Control]
  ): Vector[ir.Role] =
    val named = mutable.Set.empty[String]
    def walk(value: PValue): Unit = value match
      case PMessage(fields) =>
        fields.foreach:
          case (field, PString(id))
              if Set("role", "role_id", "worker", "task_queue", "environment")(
                field.name
              ) =>
            named += id
          case (_, nested) => walk(nested)
      case PRepeated(values) => values.foreach(walk)
      case _                 => ()
    scripts.foreach(script => walk(script.toPMessage))
    controls.foreach(control => walk(control.toPMessage))
    Vector(
      workflowService,
      caseWorker,
      taskQueue,
      handlerTaskQueue,
      nexusEndpoint
    )
      .filter(role =>
        boundRoles(role.id) || named(
          role.id
        ) || (role.resource.nonEmpty && named(role.resource))
      )
      .map(_.asIR)

  private def splitServerSteps(
      values: Vector[
        ServerStep | ActivityExternalSettlement[?, ?] | ActivityExternalSettlement.Scheduled |
          ActivityResetSettlement
      ]
  ): (
      Vector[ServerStep],
      Vector[
        ActivityExternalSettlement[?, ?] | ActivityExternalSettlement.Scheduled
      ],
      Vector[ActivityResetSettlement]
  ) =
    val stated = values.collect { case value: ServerStep => value }
    val external = Vector.newBuilder[
      ActivityExternalSettlement[?, ?] | ActivityExternalSettlement.Scheduled
    ]
    values.foreach:
      case value: ActivityExternalSettlement[?, ?]     => external += value
      case value: ActivityExternalSettlement.Scheduled => external += value
      case _                                           => ()
    val reset = values.collect { case value: ActivityResetSettlement => value }
    (stated, external.result(), reset)

  private def serverStepsToIR(
      scripts: Seq[ir.Script],
      stated: Seq[ServerStep],
      core: Realization,
      context: ConstructedRealization.Context,
      deadlines: DeadlineCatalog,
      machine: ir.Machine
  ): Vector[ir.ServerStep] =
    val activities = scripts.filter(_.activation.isActivity)
    val deliveries = activities
      .flatMap(_.getActivity.starts)
      .map(step =>
        ir.ServerStep(
          step = Some(step),
          kind = ir.CauseKind.CAUSE_KIND_DELIVERY
        )
      )
    val timerActions = machine.steps
      .map(_.action)
      .filter(id => context.metadata.actions.find(_.id == id).exists(_.timer))
    val backoff = timerActions
      .filter(id =>
        activities.nonEmpty && context.metadata.actions
          .find(_.id == id)
          .exists(_.name == "backoff")
      )
      .map(id =>
        ir.ServerStep(
          step = Some(ir.ActionClass(id)),
          kind = ir.CauseKind.CAUSE_KIND_TIMER,
          deadline = Some(firstRetryBackoff)
        )
      )
    val expired = scripts
      .flatMap(_.items)
      .flatMap(_.performs)
      .flatMap(_.step)
      .flatMap: step =>
        step.inputs.zipWithIndex.collect:
          case (input, index)
              if input.kind.`enum`
                .exists(value => value.`type`.endsWith(".Timeout") && value.`case` == "expires") =>
            val name = context.metadata.actions
              .find(_.id == step.action)
              .get
              .inputs(index)
              .name
            name -> deadlines.bases.getOrElse(
              step.action -> index,
              ir.TimeoutBasis.TIMEOUT_BASIS_UNSPECIFIED
            )
    val byInput = expired.groupMap(_._1)(_._2)
    val deadlineSteps = timerActions
      .filter(id =>
        context.metadata.actions
          .find(_.id == id)
          .exists(action => byInput.contains(action.name))
      )
      .map: id =>
        val action = context.metadata.actions.find(_.id == id).get
        val bases = byInput(action.name).distinct
        if bases.size != 1 then
          fail(
            context.declaration,
            s"timer ${action.name} has different selected timeout bases"
          )
        ir.ServerStep(
          step = Some(ir.ActionClass(id)),
          kind = ir.CauseKind.CAUSE_KIND_TIMER,
          deadline = Some(timerDeadline),
          timeoutBasis = bases.head
        )
    val derived = (deliveries ++ backoff ++ deadlineSteps).toVector
    stated.foldLeft(derived): (steps, value) =>
      val converted = value.asIR(core, context)
      val index = steps.indexWhere(_.step.equals(converted.step))
      if index < 0 then steps :+ converted
      else if steps(index)
          .copy(position = None)
          .equals(converted.copy(position = None))
      then
        fail(
          context.declaration,
          s"the realization states the server step it derives for ${converted.getStep.action}: drop it"
        )
      else steps.updated(index, converted)

  extension (value: Role)
    private def asIR: ir.Role =
      ir.Role(
        id = value.id,
        kind = ir.Role.Kind.fromValue(value.kind.ordinal + 1),
        namespace = value.namespace,
        resource = value.resource
      )

  extension (value: WaitBound)
    private def asIR: ir.WaitBound =
      ir.WaitBound(interval = Some(value.interval), atMost = Some(value.atMost))

  extension (value: ApiBehavior)
    private def asIR: ir.ApiBehavior = ir.ApiBehavior(
      visibility = value.visibility.map(_.asIR),
      causes = value.causes.map(_.asIR),
      attemptNumbering = value.attemptNumbering.map(numbering =>
        ir.AttemptNumbering(first = numbering.first, oneRun = numbering.oneRun)
      ),
      instructionDefaults = value.instructionDefaults.map(limit =>
        ir.InstructionLimit(
          timeout = Some(limit.timeout),
          attempts = limit.attempts
        )
      ),
      runOrderIsCausal = value.runOrderIsCausal
    )

  extension (value: Visibility)
    private def asIR: ir.Visibility =
      given CanEqual[Visible, Visible] = CanEqual.derived
      val read = methodName(value.read)
      val (write, id) = value.write match
        case method: MethodDescriptor[?, ?] =>
          val name = methodName(method)
          ir.Visibility.Write.Method(
            name
          ) -> s"visibility.${idPart(name)}.${idPart(read)}"
        case cause: CauseKind =>
          ir.Visibility.Write.Cause(causeKind(cause)) ->
            s"visibility.$cause.${idPart(read)}"
      ir.Visibility(
        id = id,
        write = write,
        read = read,
        eventuallyWithin = value.when match
          case Visible.atOnce            => None
          case Visible.eventually(bound) => Some(bound.asIR)
      )

  extension (value: CauseBound)
    private def asIR: ir.CauseBound = ir.CauseBound(
      id = s"cause.${value.kind}",
      kind = causeKind(value.kind),
      bound = Some(value.bound.asIR)
    )

  private def methodName(value: MethodDescriptor[?, ?]): String =
    "/" + value.getFullMethodName.stripPrefix("/")

  private def idPart(method: String): String =
    val bare = method.substring(method.lastIndexOf('/') + 1)
    bare.take(1).toLowerCase + bare.drop(1)

  extension (value: ServerStep)
    private def asIR(
        core: Realization,
        context: ConstructedRealization.Context
    ): ir.ServerStep = ir.ServerStep(
      step = Some(actionClass(value.step, core, context)),
      kind = causeKind(value.kind),
      deadline = value.deadline,
      timeoutBasis = ir.TimeoutBasis.fromValue(value.timeoutBasis.ordinal)
    )

  private def causeKind(value: CauseKind): ir.CauseKind =
    given CanEqual[CauseKind, CauseKind] = CanEqual.derived
    value match
      case CauseKind.activityAnswer    => ir.CauseKind.CAUSE_KIND_ACTIVITY_ANSWER
      case CauseKind.activityHeartbeat =>
        ir.CauseKind.CAUSE_KIND_ACTIVITY_HEARTBEAT
      case CauseKind.workflowTask => ir.CauseKind.CAUSE_KIND_WORKFLOW_TASK
      case CauseKind.handlerReply => ir.CauseKind.CAUSE_KIND_HANDLER_REPLY
      case CauseKind.delivery     => ir.CauseKind.CAUSE_KIND_DELIVERY
      case CauseKind.timer        => ir.CauseKind.CAUSE_KIND_TIMER

  extension (
      value: ActivityExternalSettlement[?, ?] | ActivityExternalSettlement.Scheduled
  )
    private def asIR(
        context: ConstructedRealization.Context
    ): ir.ActivityExternalSettlement =
      value match
        case settlement: ActivityExternalSettlement[?, ?] =>
          ir.ActivityExternalSettlement(
            carrier = commandId(settlement.carrier, context.declaration),
            activity = settlement.activity.id,
            attempt = settlement.attempt,
            pending = settlement.pending.id,
            held = commandId(settlement.held, context.declaration),
            answer = commandId(settlement.answer, context.declaration),
            requestCancel = settlement.requestCancel.fold("")(
              commandId(_, context.declaration)
            ),
            settlement = commandId(settlement.settlement, context.declaration),
            cleanup = Some(commandToIR(settlement.cleanup, context))
          )
        case settlement: ActivityExternalSettlement.Scheduled =>
          ir.ActivityExternalSettlement(
            carrier = commandId(settlement.carrier, context.declaration),
            answer = commandId(settlement.answer, context.declaration),
            settlement = commandId(settlement.settlement, context.declaration),
            cleanup = Some(commandToIR(settlement.cleanup, context))
          )

  extension (value: ActivityResetSettlement)
    private def asIR(
        core: Realization,
        context: ConstructedRealization.Context
    ): ir.ActivityResetSettlement = ir.ActivityResetSettlement(
      carrier = commandId(value.carrier, context.declaration),
      activity = value.activity.id,
      attempt = value.attempt,
      pending = value.pending.id,
      held = commandId(value.held, context.declaration),
      resetRequest = commandId(value.resetRequest, context.declaration),
      timer = Some(actionClass(value.timer, core, context)),
      freshAttempt = value.freshAttempt,
      settlement = commandId(value.settlement, context.declaration),
      cleanup = Some(commandToIR(value.cleanup, context))
    )
