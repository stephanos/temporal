package framework.realize

import com.google.protobuf.empty.Empty
import io.temporal.server.api.umpire.v1 as ir
import framework.dsl.{Action, ClassRef}

private def unsupported(declaration: String, value: Any): Nothing =
  throw IllegalArgumentException(
    s"$declaration: ${value.getClass.getName} has no constructed IR conversion"
  )

extension (value: Correlation)
  def toIR: ir.Correlation = ir.Correlation(
    projection = value.projection,
    run = value.run,
    operation = value.operation,
    observation = value.observation,
    events = value.events,
    buffered = value.buffered,
    keys = value.keys,
    support = value.support,
    work = value.work,
    eventSize = value.eventSize
  )

extension (value: Name) def toIR: ir.Name = ir.Name(value.prefix, value.fixture, value.suffix)

extension (value: RejectionCode[?])
  def toIR(declaration: String): ir.RejectionCode =
    import framework.dsl.outcomes.Rejection
    given CanEqual[Rejection, Rejection] = CanEqual.derived
    val reason: Rejection = value.rejection match
      case known: Rejection => known
      case _                =>
        throw IllegalArgumentException(
          s"$declaration: rejection reason has no constructed IR conversion"
        )
    val rejection = reason match
      case Rejection.notFound      => ir.RejectionCode.Rejection.REJECTION_NOT_FOUND
      case Rejection.alreadyExists =>
        ir.RejectionCode.Rejection.REJECTION_ALREADY_EXISTS
      case Rejection.failedPrecondition =>
        ir.RejectionCode.Rejection.REJECTION_FAILED_PRECONDITION
      case Rejection.invalidArgument =>
        ir.RejectionCode.Rejection.REJECTION_INVALID_ARGUMENT
    ir.RejectionCode(rejection, value.grpcCode)

extension (value: Field[?, ?])
  private[framework] def irPath(declaration: String): String =
    value.path.getOrElse(
      throw IllegalArgumentException(
        s"$declaration: opaque selector has no captured protobuf path"
      )
    )

extension (value: TypedProto[?])
  def toIR(declaration: String): ir.Proto =
    ir.Proto(
      message = value.message,
      fields = value.fields.map: field =>
        ir.ProtoField(
          protoFieldName(field.field.irPath(declaration), declaration),
          Some(field.value.toIR(declaration))
        )
    )

private def protoFieldName(path: String, declaration: String): String =
  path match
    case direct if !direct.exists(c => c == '.' || c == '[' || c == '<') =>
      direct
    case oneof if oneof.matches("[^.\\[<>]+<[^<>]+>") =>
      oneof.substring(oneof.indexOf('<') + 1, oneof.length - 1)
    case _ =>
      throw IllegalArgumentException(
        s"$declaration: $path is no direct protobuf literal field"
      )

extension (value: TypedProtoValue[?])
  def toIR(declaration: String): ir.ProtoValue = value.value match
    case literal: ProtoValue    => literal.toIR
    case message: TypedProto[?] =>
      ir.ProtoValue(ir.ProtoValue.Kind.Message(message.toIR(declaration)))
    case values: Vector[?] if value.message.nonEmpty =>
      ir.ProtoValue(ir.ProtoValue.Kind.Messages(ir.ProtoMessages(values.map:
        case message: TypedProto[?] => message.toIR(declaration)
        case other                  => unsupported(declaration, other))))
    case entries: Vector[?] =>
      ir.ProtoValue(ir.ProtoValue.Kind.Mapping(ir.ProtoMap(entries.map:
        case entry: TypedProtoEntry =>
          ir.ProtoEntry(entry.key, Some(entry.value.toIR(declaration)))
        case other => unsupported(declaration, other))))
    case other => unsupported(declaration, other)

extension (value: ProtoValue)
  def toIR: ir.ProtoValue = ir.ProtoValue(value match
    case ProtoValue.Text(text)     => ir.ProtoValue.Kind.Text(text)
    case ProtoValue.Flag(flag)     => ir.ProtoValue.Kind.Flag(flag)
    case ProtoValue.Number(number) => ir.ProtoValue.Kind.Number(number)
    case ProtoValue.EnumName(name) => ir.ProtoValue.Kind.EnumName(name)
    case ProtoValue.Utf8(text)     => ir.ProtoValue.Kind.Utf8(text)
    case ProtoValue.RoleId(role)   => ir.ProtoValue.Kind.RoleId(role)
    case ProtoValue.Named(name)    => ir.ProtoValue.Kind.Named(name.toIR))

extension (value: Operand)
  def toIR(declaration: String): ir.Operand = ir.Operand(kind = value match
    case Operand.Literal(literal) => ir.Operand.Kind.Literal(literal.toIR)
    case Operand.Message(message) =>
      ir.Operand.Kind.Literal(
        ir.ProtoValue(ir.ProtoValue.Kind.Message(message.toIR(declaration)))
      )
    case Operand.Environment(binding)  => ir.Operand.Kind.Environment(binding)
    case Operand.Run                   => ir.Operand.Kind.Run(Empty())
    case Operand.LearnedValue(learned) => ir.Operand.Kind.LearnedValue(learned)
    case Operand.Projected             => ir.Operand.Kind.Projected(Empty()))

extension (value: TypedAssignment[?, ?])
  def toIR(declaration: String): ir.Assignment =
    ir.Assignment(
      value.target.irPath(declaration),
      Some(value.value.operand.toIR(declaration))
    )

  private[realize] def toIRAssignments(
      declaration: String
  ): Vector[ir.Assignment] =
    if !value.expandsMessageFields then Vector(value.toIR(declaration))
    else
      value.value.operand match
        case Operand.Message(message) =>
          assignedMessage(
            value.target.irPath(declaration),
            message,
            declaration
          )
        case other => unsupported(declaration, other)

private def assignedMessage(
    prefix: String,
    value: TypedProto[?],
    declaration: String
): Vector[ir.Assignment] =
  value.fields.flatMap: field =>
    val target = prefix + "." + protoFieldName(
      field.field.irPath(declaration),
      declaration
    )
    field.value.value match
      case nested: TypedProto[?] => assignedMessage(target, nested, declaration)
      case _                     =>
        Vector(
          ir.Assignment(
            target,
            Some(
              ir.Operand(kind = ir.Operand.Kind.Literal(field.value.toIR(declaration)))
            )
          )
        )

extension (value: TypedResponseRead[?, ?])
  def toIR(declaration: String): ir.ResponseRead = ir.ResponseRead(
    value.path.irPath(declaration),
    value.cardinality match
      case Cardinality.one  => ir.ResponseRead.Cardinality.CARDINALITY_ONE
      case Cardinality.each => ir.ResponseRead.Cardinality.CARDINALITY_EACH,
    value.targets.map(_.toIR)
  )

extension (value: Target)
  def toIR: ir.Target = ir.Target(value match
    case Target.Observe(id) => ir.Target.Target.Observe(id)
    case Target.Bind(id)    => ir.Target.Target.Bind(id)
    case Target.Lift(id)    => ir.Target.Target.Lift(id))

extension (value: Condition[?])
  def toIR(declaration: String): ir.Operand =
    def path = ir.Operand(kind =
      ir.Operand.Kind.Path(
        ir.PathOf(
          Some(Operand.Projected.toIR(declaration)),
          value.field.get.irPath(declaration)
        )
      )
    )
    ir.Operand(kind = value.operator match
      case ConditionOperator.present =>
        ir.Operand.Kind.Present(ir.Present(Some(path)))
      case ConditionOperator.equal =>
        ir.Operand.Kind.Equal(
          ir.Equal(Some(path), Some(value.value.get.operand.toIR(declaration)))
        )
      case ConditionOperator.greater =>
        ir.Operand.Kind.Greater(
          ir.Greater(
            Some(path),
            Some(value.value.get.operand.toIR(declaration))
          )
        )
      case ConditionOperator.not =>
        ir.Operand.Kind.Not(ir.Not(Some(value.children.head.toIR(declaration))))
      case ConditionOperator.all =>
        ir.Operand.Kind.All(ir.All(value.children.map(_.toIR(declaration)))))

extension (value: Command | Instruction)
  private[framework] def commandId(declaration: String): String = value match
    case c: Command if c.alias.nonEmpty => c.alias.get.commandId(declaration)
    case c: Command if c.id.nonEmpty    => c.id
    case c: Command                     =>
      c.ref.fold(c.instruction.commandId(declaration))(r => kebab(r.spelling))
    case r: Instruction.TypedRpc[?, ?] if r.sharedName.nonEmpty =>
      r.sharedName.get.commandId(declaration)
    case i: Instruction =>
      i.ref
        .map(r => kebab(r.spelling))
        .getOrElse(
          throw IllegalArgumentException(
            s"$declaration: a command has no retained declaring-val reference"
          )
        )

private[framework] def constructedCommandId(
    value: Command | Instruction,
    declaration: String
): String = value.commandId(declaration)

private[framework] def constructedCommandToIR(
    value: Command | Instruction,
    placeholder: Boolean,
    declaration: String
): ir.Command =
  def placeholderFor(instruction: Instruction): Instruction =
    val replacement = Instruction.Finish(Operand.Literal(ProtoValue.Text("")))
    instruction.ref.fold(replacement)(reference =>
      replacement.named(
        reference.spelling,
        reference.declaration,
        reference.location
      )
    )
  value match
    case command: Command =>
      val instruction =
        if placeholder then placeholderFor(command.instruction)
        else command.instruction
      val retained =
        if placeholder then command.copy(instruction = instruction) else command
      retained.toIR(declaration)
    case instruction: Instruction =>
      val retained =
        if placeholder then placeholderFor(instruction) else instruction
      Command("", retained).toIR(declaration)

private def kebab(name: String): String =
  name.flatMap(c => if c.isUpper then s"-${c.toLower}" else c.toString)

private def roleId(role: String | Addressee): String = role match
  case r: Addressee => r.id
  case s: String    => s

extension (value: Instruction)
  def toIR(declaration: String): ir.Command.Instruction = value match
    case c: Instruction.TypedRpc[?, ?] =>
      ir.Command.Instruction.Rpc(
        ir.Rpc(
          roleId(c.role),
          "/" + c.method.getFullMethodName.stripPrefix("/"),
          c.assign.flatMap(_.toIRAssignments(declaration)),
          c.reads.map(_.toIR(declaration))
        )
      )
    case p: Instruction.TypedPoll[?, ?] =>
      require(
        p.reads.isEmpty,
        s"$declaration: scoped poll response reads have no IR field"
      )
      ir.Command.Instruction.Poll(
        ir.Poll(
          p.evidence.evidence.id,
          roleId(p.role),
          p.assign.flatMap(_.toIRAssignments(declaration)),
          Some(p.until.toIR(declaration)),
          p.interval
        )
      )
    case Instruction.AwaitLearned(learned) =>
      ir.Command.Instruction.AwaitLearned(learned match
        case s: String  => s
        case l: Learned => l.id)
    case Instruction.AwaitCommand(command) =>
      ir.Command.Instruction.AwaitCommand(command match
        case s: String                  => s
        case c: (Command | Instruction) => c.commandId(declaration))
    case Instruction.Finish(result) =>
      ir.Command.Instruction.Finish(ir.Finish(Some(result.toIR(declaration))))
    case Instruction.Hold(control) =>
      ir.Command.Instruction.Hold(control match
        case s: String   => s
        case a: Actuator => a.id)
    case Instruction.Release(control) =>
      ir.Command.Instruction.Release(control match
        case s: String   => s
        case a: Actuator => a.id)
    case other => unsupported(declaration, other)

extension (value: Command)
  def toIR(declaration: String): ir.Command = ir.Command(
    id = value.commandId(declaration),
    after = value.after.map(a => ir.After(a.commands)),
    timeout = value.timeout,
    regardless = value.regardless,
    instruction = value.instruction.toIR(declaration),
    closes = value.closes.map:
      case s: String            => s
      case e: EvidenceRef[?, ?] => e.evidence.id
      case e: TypedEvidence[?]  => e.evidence.id
  )

final private[framework] class Construction(
    val declaration: String,
    val model: ir.Model,
    val machine: String,
    val nativeFacts: Option[IndexedSeq[Any]] = None,
    val factCases: Vector[(AnyRef, String)] = Vector.empty
):
  def fail(message: String): Nothing = throw IllegalArgumentException(
    s"$declaration: $message"
  )

  def actionClass(value: ClassRef): ir.ActionClass =
    val (decl, values) = value match
      case c: framework.dsl.Class => c.decl -> c.values
      case a: Action[?]           => a.decl -> Nil
    val id =
      decl.sourceIdentity.getOrElse(
        fail(s"action ${decl.name} has no retained source identity")
      )
    val definition =
      model.actions
        .find(_.id == id)
        .getOrElse(fail(s"action $id has no signature metadata"))
    require(
      decl.on.fold("")(_.sourceName) == definition.on && decl.creates.fold("")(
        _.sourceName
      ) == definition.creates,
      s"$declaration: action $id has no exact entity binding metadata"
    )
    require(
      values.size == definition.inputs.size,
      s"$declaration: action $id has no exact class inputs"
    )
    require(
      decl.domains.size == values.size,
      s"$declaration: action $id has no native action input catalog"
    )
    values
      .zip(decl.domains)
      .zip(definition.inputs)
      .foreach: (input, parameter) =>
        require(
          input._2.values.exists(_.equals(input._1)),
          s"$declaration: action $id input ${parameter.name} does not belong to its native action input catalog"
        )
    ir.ActionClass(
      id,
      values.zip(definition.inputs).map((v, p) => inputValue(v, p.getType))
    )

  private def fields(value: Product, types: Seq[ir.Field]): Seq[ir.Value] =
    require(
      value.productArity == types.size,
      s"$declaration: ${value.productPrefix} has inconsistent fields"
    )
    value.productIterator
      .zip(types.iterator)
      .map((v, f) => inputValue(v, f.getType))
      .toVector

  private[framework] def inputValue(value: Any, tpe: ir.TypeRef): ir.Value =
    (value, tpe.ref) match
      case (b: Boolean, ir.TypeRef.Ref.Bool(_)) =>
        ir.Value(ir.Value.Kind.Bool(b))
      case (n: Int, ir.TypeRef.Ref.IntRange(_) | ir.TypeRef.Ref.Int(_)) =>
        ir.Value(ir.Value.Kind.Int(n.toLong))
      case (n: Long, ir.TypeRef.Ref.Int(_))       => ir.Value(ir.Value.Kind.Int(n))
      case (p: Product, ir.TypeRef.Ref.Named(id)) =>
        val declared =
          model.types
            .find(_.name == id)
            .getOrElse(fail(s"input type $id has no metadata"))
        declared.shape match
          case ir.Type.Shape.Enum(e) =>
            val c = e.cases
              .find(_.name == p.productPrefix)
              .getOrElse(fail(s"$id has no case ${p.productPrefix}"))
            ir.Value(
              ir.Value.Kind.Enum(ir.EnumValue(id, c.name, fields(p, c.fields)))
            )
          case ir.Type.Shape.Record(r) =>
            ir.Value(
              ir.Value.Kind.Record(ir.RecordValue(id, fields(p, r.fields)))
            )
          case _ => fail(s"input type $id has no constructed value form")
      case _ => fail(s"$value has no constructed input value form for $tpe")

  def fact(value: Fact): String = value match
    case s: String => s
    case reference =>
      val spelling = reference match
        case p: Product =>
          if !nativeFacts.exists(_.exists(_.equals(p))) then
            fail("fact does not belong to the native machine fact catalog")
          p.productPrefix
        case _ =>
          factCases
            .find((original, _) => original eq reference)
            .map(_._2)
            .getOrElse(
              fail(
                "fact does not belong to the native machine fact catalog; no retained case companion identity"
              )
            )
      val factType = model.machines
        .find(_.name == machine)
        .map(_.factType)
        .filter(_.nonEmpty)
        .getOrElse(fail(s"machine $machine has no fact-type metadata"))
      val cases = model.types
        .find(_.name == factType)
        .map(_.getEnum.cases)
        .getOrElse(fail(s"fact type $factType has no metadata"))
      if !cases.exists(_.name == spelling) then fail(s"$factType has no fact $spelling")
      spelling

extension (value: Script)
  private[realize] def toIR(ctx: Construction): ir.Script =
    val activation = value.activation match
      case _: Activation.Controller.type =>
        ir.Script.Activation.Controller(Empty())
      case other => unsupported(ctx.declaration, other)
    ir.Script(
      id = value.id,
      activation = activation,
      items = value.items.map(_.toIR(ctx))
    )

extension (value: Item)
  private[realize] def toIR(ctx: Construction): ir.Item =
    require(
      value.binding.isEmpty,
      s"${ctx.declaration}: performance binding has no constructed conversion"
    )
    ir.Item(
      command = value.command.map(_.toIR(ctx.declaration)),
      when = value.when.map:
        case c: framework.dsl.Class                => ctx.actionClass(c)
        case a: Action[?] if a.decl.inputs.isEmpty =>
          ctx.actionClass(a.asInstanceOf[Action[EmptyTuple]])
        case a: Action[?] =>
          ctx.fail(
            s"action ${a.name} requires a retained expanded class catalog"
          ),
      performs = value.performs.map(p =>
        ir.Performance(
          step = Some(ctx.actionClass(p.step)),
          command = Some(p.command.toIR(ctx.declaration))
        )
      )
    )

extension (value: Recorded)
  private[realize] def toIR(ctx: Construction): ir.Evidence.From = value match
    case r: Recorded.TypedRead[?, ?, ?] =>
      ir.Evidence.From.Read(
        ir.ReadSource(
          "/" + r.method.getFullMethodName.stripPrefix("/"),
          r.path.irPath(ctx.declaration)
        )
      )
    case r: Recorded.TypedSingle[?, ?, ?] =>
      ir.Evidence.From.Single(
        ir.ReadSource(
          "/" + r.method.getFullMethodName.stripPrefix("/"),
          r.path.irPath(ctx.declaration)
        )
      )
    case r: Recorded.TypedRunEvent[?] =>
      val key = r.key match
        case k: RunKey              => k.operand.toIR(ctx.declaration)
        case k: ProjectedPath[?, ?] =>
          ir.Operand(kind =
            ir.Operand.Kind.Path(
              ir.PathOf(
                Some(k.origin.operand.toIR(ctx.declaration)),
                k.field.irPath(ctx.declaration)
              )
            )
          )
      val command = r.command match
        case s: String                  => s
        case c: (Command | Instruction) => c.commandId(ctx.declaration)
      def scriptId(s: String | Script): String = s match
        case s: String => s
        case s: Script => s.id
      val kind = ir.RunEventSource.Kind.fromValue(r.kind.ordinal + 1)
      ir.Evidence.From.RunEvent(
        ir.RunEventSource(
          kind,
          scriptId(r.script),
          command,
          Some(key),
          r.guard.map(_.toIR(ctx.declaration)),
          r.attempt.map(a => ir.AttemptOf(script = scriptId(a.script), number = a.number))
        )
      )
    case other => unsupported(ctx.declaration, other)

extension (value: TypedEvidenceField[?, ?])
  private[realize] def toIR(declaration: String): ir.EvidenceField =
    ir.EvidenceField(
      id = value.id,
      path = value.path.irPath(declaration),
      role = value.role.fold(ir.EvidenceField.Role.ROLE_UNSPECIFIED)(r =>
        ir.EvidenceField.Role.fromValue(r.ordinal + 1)
      ),
      redacted = value.redacted
    )

extension (value: Evidence)
  private[realize] def toIR(ctx: Construction): ir.Evidence = ir.Evidence(
    id = value.id,
    records = ctx.fact(value.records),
    source = value.source,
    from = value.from.toIR(ctx),
    commitment = ir.Evidence.Commitment.fromValue(value.commitment.ordinal + 1),
    exhaustive = value.exhaustive,
    confirms = value.confirms.map(t =>
      ir.Taking(step = Some(ctx.actionClass(t.step)), occurrence = t.occurrence)
    )
  )

extension (value: EvidenceRef[?, ?])
  private[realize] def toIR(ctx: Construction): ir.Evidence = value.evidence
    .toIR(ctx)
    .copy(
      operation = value.operation.irPath(ctx.declaration),
      fields = value.fields.map(_.toIR(ctx.declaration))
    )

extension (value: TypedEvidence[?])
  private[realize] def toIR(ctx: Construction): ir.Evidence = value.evidence
    .toIR(ctx)
    .copy(
      operation = value.operation.fold("")(_.irPath(ctx.declaration)),
      fields = value.fields.map(_.toIR(ctx.declaration))
    )

extension (value: Actuator)
  private[realize] def toIR(ctx: Construction): ir.Control =
    val kind = value.kind match
      case ControlKind.HoldDispatched(step) =>
        ir.Control.Kind.HoldDispatched(
          ir.HoldDispatched(Some(ctx.actionClass(step)))
        )
      case ControlKind.HoldDelivery(channel) =>
        ir.Control.Kind.HoldDelivery(
          channel.sourceIdentity.getOrElse(
            ctx.fail(
              s"control ${value.id}: channel has no retained source identity"
            )
          )
        )
    ir.Control(id = value.id, kind = kind, role = roleId(value.role))

extension (value: Realization)
  private[framework] def sourceActions: Vector[framework.dsl.ActionDecl] =
    def declaration(ref: ClassRef | Action[?]): framework.dsl.ActionDecl =
      ref match
        case c: framework.dsl.Class => c.decl
        case a: Action[?]           => a.decl
    val scriptActions = value.scripts
      .flatMap(_.items)
      .flatMap(i => i.when.map(declaration) ++ i.performs.map(p => declaration(p.step)))
    val evidenceActions = value.evidence.flatMap:
      case e: Evidence          => e.confirms.map(t => declaration(t.step))
      case e: EvidenceRef[?, ?] =>
        e.evidence.confirms.map(t => declaration(t.step))
      case e: TypedEvidence[?] =>
        e.evidence.confirms.map(t => declaration(t.step))
    val controlActions = value.controls.flatMap(_.kind match
      case ControlKind.HoldDispatched(step) => Vector(declaration(step))
      case ControlKind.HoldDelivery(_)      => Vector.empty)
    scriptActions ++ evidenceActions ++ controlActions

  private[framework] def sourceChannels: Vector[String] =
    value.controls.flatMap(_.kind match
      case ControlKind.HoldDelivery(channel) => channel.sourceIdentity.toVector
      case ControlKind.HoldDispatched(_)     => Vector.empty)

  def toIR(
      declaration: String,
      metadata: ir.Model = ir.Model()
  ): ir.Realization =
    value.toIRWithFamily(declaration, metadata, None)

  private[framework] def toIRWithFamily(
      declaration: String,
      metadata: ir.Model,
      sourcePackage: Option[String]
  ): ir.Realization =
    val ctx =
      new Construction(
        declaration,
        metadata,
        value.machine.name,
        Some(value.machine.ff.values),
        value.machine.factCases
      )
    value.machine.bindings: Unit
    require(
      value.roles.isEmpty,
      s"$declaration: roles require their owning kit's constructed conversion"
    )
    require(
      value.requiredSettings.isEmpty,
      s"$declaration: settings require their owning kit's constructed conversion"
    )
    require(
      value.behavior.isEmpty,
      s"$declaration: behavior requires its owning kit's constructed conversion"
    )
    require(
      value.serverSteps.isEmpty,
      s"$declaration: server steps require their owning kit's constructed conversion"
    )
    val result = ir.Realization(
      id = declaration,
      name =
        if value.name.nonEmpty then value.name else declaration.split('.').last,
      machine = value.machine.name,
      producer = value.producer,
      producerVersion = value.producerVersion,
      correlation = Some(value.correlation.toIR),
      scripts = value.scripts.map(_.toIR(ctx)),
      learned = value.learned.map(l =>
        ir.Learned(
          id = l.id,
          kind =
            if l.kind.ordinal == 0 then ir.Learned.Kind.KIND_TEXT
            else ir.Learned.Kind.KIND_HANDLE
        )
      ),
      observations = value.observations.map(o => ir.Observed(id = o.id, message = o.message)),
      controls = value.controls.map(_.toIR(ctx)),
      evidence = value.evidence.map:
        case e: Evidence          => e.toIR(ctx)
        case e: EvidenceRef[?, ?] => e.toIR(ctx)
        case e: TypedEvidence[?]  => e.toIR(ctx),
      cleanup = value.cleanup,
      rejectionCodes = value.rejectionCodes.map(_.toIR(declaration))
    )
    Family.resolve(result, declaration, sourcePackage)
