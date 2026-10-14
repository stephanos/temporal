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
      actor = decl.actor.name,
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
