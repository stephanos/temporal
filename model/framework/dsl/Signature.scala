package framework.dsl

import scala.collection.mutable
import framework.realize.Construction
import io.temporal.server.api.umpire.v1 as ir

private[framework] object Signature:
  private given CanEqual[ir.Type, ir.Type] = CanEqual.derived
  final case class InputMetadata(token: Option[Input[?]], default: Option[ir.Value])

  private def fail(context: String, message: String): Nothing =
    throw IllegalArgumentException(s"$context: $message")

  private def schema(domain: Finite[?], key: Option[String], context: String): FiniteSchema =
    val retained =
      key.flatMap(domain.schemaFor).getOrElse(fail(context, "no retained typed schema"))
    if retained.unresolvedInboxes.nonEmpty || retained.root.ref.isChannel then
      fail(context, "unresolved Inbox references require state-field channel ownership")
    val definitions = types(Vector(retained), context).map(t => t.name -> t).toMap
    val visited = mutable.Set.empty[String]
    def visit(ref: ir.TypeRef): Unit = ref.ref match
      case ir.TypeRef.Ref.Named(name) if name != "scala.Nothing" && visited.add(name) =>
        val value =
          definitions.getOrElse(name, fail(context, s"type $name has no retained definition"))
        (value.getRecord.fields ++ value.getEnum.cases.flatMap(_.fields)).foreach(f =>
          visit(f.getType)
        )
      case ir.TypeRef.Ref.Channel(_) =>
        fail(context, "unresolved Inbox references require state-field channel ownership")
      case _ => ()
    visit(retained.root)
    retained

  def stateSchema(domain: Finite[?], key: Option[String], context: String): FiniteSchema =
    val retained =
      key.flatMap(domain.schemaFor).getOrElse(fail(context, "no retained typed schema"))
    retained.copy(types = stateTypes(Seq(domain -> key), context), unresolvedInboxes = Map.empty)

  def stateTypes(selections: Seq[(Finite[?], Option[String])], context: String): Vector[ir.Type] =
    val selected = selections.map { (domain, key) =>
      val retained =
        key.flatMap(domain.schemaFor).getOrElse(fail(context, "no retained typed schema"))
      retained -> domain.inboxesFor(retained.identity)
    }
    val definitions = types(selected.map(_._1), context).map(t => t.name -> t).toMap
    val ownership = mutable.LinkedHashMap.empty[InboxOwnership.Field, InboxOwnership.Entry]
    selected.foreach { (retained, evidence) =>
      evidence.foreach { (coordinate, entry) =>
        val at = s"$context schema ${retained.identity} field ${coordinate.context}"
        if !retained.root.ref.named.contains(coordinate.owner) then
          fail(at, "ownership belongs to another declaring type")
        val shape = definitions.getOrElse(coordinate.owner, fail(at, "missing declaring type"))
        val fields = if coordinate.caseName.isEmpty then shape.getRecord.fields
        else shape.getEnum.cases.find(_.name == coordinate.caseName).toSeq.flatMap(_.fields)
        val field = fields
          .find(_.name == coordinate.name)
          .getOrElse(fail(at, "extra or wrong case/field evidence"))
        if !field.getType.ref.channel.contains(entry.message) || !retained.unresolvedInboxes
            .contains(entry.message)
        then fail(at, "Inbox message key does not match the typed field")
        if ownership.get(coordinate).exists(prior => !prior.agrees(entry)) then
          fail(at, "conflicting Inbox ownership")
        ownership.getOrElseUpdate(coordinate, entry): Unit
      }
      retained.root.ref.named.flatMap(definitions.get).foreach { root =>
        val coordinates = root.getRecord.fields.map("" -> _) ++
          root.getEnum.cases.flatMap(c => c.fields.map(c.name -> _))
        coordinates.foreach { (caseName, field) =>
          if field.getType.ref.isChannel then
            val coordinate = InboxOwnership.Field(root.name, caseName, field.name)
            if !evidence.contains(coordinate) then
              fail(
                s"$context schema ${retained.identity} field ${coordinate.context}",
                "no retained Inbox field ownership on the selected witness"
              )
        }
      }
    }
    val resolved = mutable.LinkedHashMap.empty[String, ir.Type]
    val visited = mutable.Set.empty[String]
    def field(value: ir.Field, owner: String, caseName: String): ir.Field =
      val coordinate = InboxOwnership.Field(owner, caseName, value.name)
      val at = s"$context field ${coordinate.context}"
      value.getType.ref match
        case ir.TypeRef.Ref.Channel(message) =>
          val entry = ownership.getOrElse(coordinate, fail(at, "no retained Inbox field ownership"))
          val channel = try entry.channel
          catch
            case scala.util.control.NonFatal(error) =>
              fail(at, s"Channel supplier failed: ${error.getMessage}")
          val id =
            channel.sourceIdentity.getOrElse(fail(at, "channel has no retained source identity"))
          if id != entry.declaration then
            fail(at, s"channel identity $id differs from ${entry.declaration}")
          if !channel.schemaKey.contains(message) then
            fail(at, "channel message key does not match the typed field")
          val messageSchema = try channelSchema(channel)
          catch case error: IllegalArgumentException => fail(at, error.getMessage)
          selected.filter(_._1.root.ref.named.contains(owner)).foreach { (schema, _) =>
            if !schema.unresolvedInboxes.get(message).contains(messageSchema.root) then
              fail(at, "channel message schema does not match the typed field")
          }
          val messageDefinitions = types(Seq(messageSchema), at)
          messageDefinitions.foreach { definition =>
            if definitions.get(definition.name).forall(_ != definition) then
              fail(at, s"message type ${definition.name} has incomplete or conflicting closure")
          }
          visit(messageSchema.root, at)
          value.copy(`type` = Some(ir.TypeRef(ir.TypeRef.Ref.Channel(id))))
        case _ =>
          visit(value.getType, at)
          value
    def visit(ref: ir.TypeRef, at: String): Unit = ref.ref match
      case ir.TypeRef.Ref.Named(name) if name != "scala.Nothing" && visited.add(name) =>
        val value = definitions.getOrElse(name, fail(at, s"type $name has no retained definition"))
        val next = value.shape match
          case ir.Type.Shape.Record(record) =>
            value.copy(shape =
              ir.Type.Shape.Record(record.copy(fields = record.fields.map(field(_, name, ""))))
            )
          case ir.Type.Shape.Enum(enumValue) =>
            value.copy(shape =
              ir.Type.Shape.Enum(
                enumValue.copy(cases =
                  enumValue.cases.map(c => c.copy(fields = c.fields.map(field(_, name, c.name))))
                )
              )
            )
          case _ => fail(context, s"type $name has no retained shape")
        val held = (next.getRecord.fields ++ next.getEnum.cases.flatMap(_.fields))
          .flatMap(_.getType.ref.channel)
        if held.distinct.size != held.size then
          fail(s"$context type $name", "a state holds a channel in one field")
        resolved(name) = next
      case ir.TypeRef.Ref.Channel(_) => fail(context, "an Inbox requires a declaring state field")
      case _                         => ()
    selected.foreach((schema, _) => visit(schema.root, s"$context schema ${schema.identity}"))
    definitions.keys.toVector.sorted.map(name =>
      resolved.getOrElse(name, fail(context, s"type $name has incomplete closure"))
    )

  private def actionId(decl: ActionDecl): String =
    decl.sourceIdentity.getOrElse(fail(s"action ${decl.name}", "no retained source identity"))

  def input(value: Input[?], declaration: String): ir.Param =
    ir.Param(
      value.name,
      Some(schema(value.domain, value.schemaKey, s"$declaration input ${value.name}").root)
    )

  private def schemas(decl: ActionDecl): Vector[FiniteSchema] =
    val id = actionId(decl)
    require(decl.inputs.size == decl.domains.size, s"action $id has inconsistent input metadata")
    val tokens = decl.tokens.flatten
    if tokens.indices.exists(i => tokens.take(i).exists(_ eq tokens(i))) then
      fail(s"action $id", "declares the input token twice: an action takes each input once")
    decl.inputs
      .zip(decl.domains)
      .zipWithIndex
      .map { (input, index) =>
        val key = decl.schemaKeys
          .lift(index)
          .flatten
          .orElse(decl.tokens.lift(index).flatten.flatMap(_.schemaKey))
        val selected = schema(input._2, key, s"action $id input ${input._1}")
        decl.sourceChannel.fold(selected)((channel, _) => boundedMessage(selected, channel))
      }
      .toVector

  private def types(values: Seq[FiniteSchema], context: String): Vector[ir.Type] =
    val collected = mutable.LinkedHashMap.empty[String, ir.Type]
    values.flatMap(_.types).foreach { value =>
      val clean = value.clearPosition
      if collected.get(value.name).exists(_ != clean) then
        fail(context, s"type ${value.name} has conflicting definitions")
      collected(value.name) = clean
    }
    collected.values.toVector

  def action(decl: ActionDecl): ir.Action =
    val id = actionId(decl)
    val selected = schemas(decl)
    val metadata = ir.Model(types = types(selected, s"action $id"))
    val ctx = new Construction(s"action $id", metadata, "")
    val name = actionName(decl)
    def entity(value: Option[Entity]): String = value.fold("") { e =>
      try e.sourceName
      catch case error: IllegalArgumentException => fail(s"action $id", error.getMessage)
    }
    ir.Action(
      id = id,
      name = name,
      actor = decl.actor.irName(s"action $id"),
      on = entity(decl.on),
      creates = entity(decl.creates),
      inputs = decl.inputs.zip(selected).map((name, schema) => ir.Param(name, Some(schema.root))),
      results = decl.results,
      timer = decl.timer,
      internal = decl.internal,
      delivers = decl.sourceChannel.filter(_._2 == "deliver").fold(decl.delivers)(_._1),
      loses = decl.sourceChannel.filter(_._2 == "lose").fold(decl.loses)(_._1),
      examples = decl.examples.map { example =>
        require(selected.size == 1, s"action $id examples require one input")
        require(
          decl.domains.head.values.exists(_.equals(example.value)),
          s"action $id example does not belong to its native action input catalog"
        )
        ir.Example(Some(ctx.inputValue(example.value, selected.head.root)), example.example)
      }
    )

  def actionName(decl: ActionDecl): String =
    val id = actionId(decl)
    decl.sourceChannel match
      case Some((_, operation)) =>
        val channel =
          decl.channelName.getOrElse(fail(s"action $id", "no retained channel declaration name"))
        channel + (if operation == "deliver" then "Delivery" else "Loss")
      case _ =>
        decl.declarationName.effective(decl.name, s"action $id")

  def assumption(value: Assumption): ir.Assumption =
    val id = value.declarationName.identity(s"assumption ${value.name}")
    val context = s"assumption $id"
    val segments = if value.fairSegments.nonEmpty then value.fairSegments
    else
      Vector(
        Assumption.Fairness.Actions(value.fairness),
        Assumption.Fairness.Composed(value.composed)
      )
    val fair = segments
      .collect { case Assumption.Fairness.Actions(actions) => actions }
      .flatten
      .map(actionId)
    val refs = segments
      .collect { case Assumption.Fairness.Composed(actions) => actions }
      .flatten
      .map(_.toIR(context))
    if fair.nonEmpty && refs.nonEmpty then
      fail(context, "makes both actions and composed steps fair")
    val owners = refs.map(_.composition).distinct
    if owners.size > 1 then
      fail(
        context,
        s"makes steps of ${owners.mkString(" and ")} fair; composed steps are of one composition"
      )
    ir.Assumption(id = id, name = value.sourceName, fair = fair, fairRefs = refs)

  def hole(value: Hole): ir.Hole =
    val id = value.declarationName.identity(s"hole ${value.name}")
    ir.Hole(id = id, name = value.sourceName)

  def limits(value: Limits): ir.Limits =
    val name = value.sourceName
    Seq("steps" -> value.steps, "actions" -> value.actions, "search" -> value.search).foreach {
      (bound, count) =>
        if count < 0 then fail(s"limits $name", s"declare $count $bound; a bound is at least 0")
    }
    ir.Limits(name, value.steps, value.actions, value.search)

  def inputs(decl: ActionDecl): Vector[InputMetadata] =
    val id = actionId(decl)
    val selected = schemas(decl)
    val ctx = new Construction(s"action $id", ir.Model(types = types(selected, s"action $id")), "")
    selected.zipWithIndex.map { (schema, index) =>
      val token = decl.tokens.lift(index).flatten
      InputMetadata(
        token,
        token.flatMap(_.default).map { value =>
          require(
            decl.domains(index).values.exists(_.equals(value)),
            s"action $id input ${decl.inputs(index)} default does not belong to its native action input catalog"
          )
          ctx.inputValue(value, schema.root)
        }
      )
    }

  private def channelSchema(value: Channel[?]): FiniteSchema =
    val id =
      value.sourceIdentity.getOrElse(fail(s"channel ${value.name}", "no retained source identity"))
    val selected = schema(value.messages, value.schemaKey, s"channel $id message")
    boundedMessage(selected, id)

  private def boundedMessage(selected: FiniteSchema, id: String): FiniteSchema =
    if selected.root.ref.isInt then
      fail(
        s"channel $id",
        "messages are Int with no declared bound: write channel[UpTo[N]] or a named opaque range"
      )
    selected

  def channel(value: Channel[?]): ir.Channel =
    val selected = channelSchema(value)
    val id = value.sourceIdentity.get
    ir.Channel(
      id = id,
      name = if value.name.nonEmpty then value.name
      else value.spelling.getOrElse(fail(s"channel $id", "no retained declaration name")),
      message = Some(selected.root),
      capacity = value.capacity,
      order = value.order match
        case Order.fifo      => ir.Channel.Order.ORDER_FIFO
        case Order.unordered => ir.Channel.Order.ORDER_UNORDERED,
      lossy = value.loss == Loss.lossy,
      duplicates = value.duplicates
    )

  def collect(
      actions: Seq[Action[?]],
      channels: Seq[Channel[?]],
      assumptions: Seq[Assumption] = Nil,
      holes: Seq[Hole] = Nil,
      limits: Seq[Limits] = Nil
  ): ir.Model =
    def unique[A](values: Seq[A], key: A => String): Vector[A] =
      val collected = mutable.LinkedHashMap.empty[String, A]
      values.foreach { value =>
        val id = key(value)
        if collected.get(id).exists(prior => !prior.equals(value)) then
          fail(s"signature $id", "conflicting definitions")
        collected(id) = value
      }
      collected.toVector.sortBy(_._1).map(_._2)
    def named[A](kind: String, values: Seq[A], id: A => String, name: A => String): Vector[A] =
      val result = unique(values, id)
      val names = mutable.LinkedHashMap.empty[String, String]
      result.foreach { value =>
        val spelling = name(value)
        names.get(spelling).foreach { other =>
          fail(s"signature $kind", s"two $kind are named $spelling: $other and ${id(value)}")
        }
        names(spelling) = id(value)
      }
      result
    try unique(limits.map(_.toIR), _.name): Unit
    catch case error: IllegalArgumentException => fail("signature limits", error.getMessage)
    ir.Model(
      actions = unique(actions.map(_.toIR), _.id),
      channels = unique(channels.map(_.toIR), _.id),
      assumptions = named("assumptions", assumptions.map(_.toIR), _.id, _.name),
      holes = named("holes", holes.map(_.toIR), _.id, _.name),
      types =
        types(actions.flatMap(a => schemas(a.decl)) ++ channels.map(channelSchema), "signature")
          .sortBy(_.name)
    )
