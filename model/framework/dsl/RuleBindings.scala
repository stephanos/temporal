package framework.dsl

import java.util.Base64
import scala.collection.mutable
import framework.realize.Construction
import io.temporal.server.api.umpire.v1 as ir

private[framework] object RuleBindings:
  final case class Contribution(
      steps: Vector[ir.StepBinding],
      functions: Vector[ir.Function],
      types: Vector[ir.Type],
      channels: Vector[String],
      holes: Vector[String]
  )
  final case class Packet(
      owner: String,
      family: String,
      machine: String,
      binding: ir.Position,
      position: ir.Position,
      params: Vector[ir.Param],
      guard: ir.Expr,
      effect: ir.Expr,
      model: ir.Model,
      curriedCalls: Vector[CoreBindings.CurriedCall] = Vector.empty,
      typePositions: Vector[ir.Type] = Vector.empty
  )
  final case class Registration(
      source: Machine[?, ?, ?],
      declaration: ActionDecl,
      values: Option[List[Any]],
      cls: Option[Seq[ir.Value]],
      guard: AnyRef,
      effect: AnyRef,
      packet: Packet,
      types: Vector[ir.Type]
  )
  final case class Retained(rule: Rule[?, ?, ?], registration: Registration)
  final case class Disabled(
      bound: Bound[?, ?, ?],
      source: Machine[?, ?, ?],
      declaration: ActionDecl,
      owner: String,
      family: String,
      machine: String,
      position: ir.Position,
      typePositions: Vector[ir.Type] = Vector.empty
  )

  private def decode[A](text: String)(read: Array[Byte] => A): A =
    read(Base64.getDecoder.decode(text))

  private def validTypePosition(t: ir.Type): Boolean =
    t.name.nonEmpty && t.position.exists(p =>
      p.equals(ir.Position.defaultInstance) || (p.file.nonEmpty && p.line > 0)
    ) && t.copy(name = "", position = None).equals(ir.Type())

  private def typePositions(text: String, context: String): Vector[ir.Type] =
    val model = decode(text)(ir.Model.parseFrom)
    require(
      model.copy(types = Nil).equals(ir.Model()),
      s"$context: unrelated type-position declarations"
    )
    require(
      model.types.forall(validTypePosition),
      s"$context: malformed type-definition position"
    )
    model.types.toVector

  def packet(data: Seq[String], context: String): Packet =
    def fail(message: String): Nothing = throw IllegalArgumentException(s"$context: $message")
    try
      if data.headOption != Some("ordinary-rule-v2") || data.size < 10 then
        fail("missing or unsupported Rule payload")
      val count = data(6).toInt
      if count < 1 || count > 7 then fail("invalid retained Rule arity")
      val params = data.slice(7, 7 + count).map(decode(_)(ir.Param.parseFrom)).toVector
      val size = data(7 + count).toInt
      if size <= 0 || size >= data.size - 8 - count then fail("incomplete retained Rule payload")
      val fragments = decode(data.slice(8 + count, 8 + count + size).mkString)(ir.ListOf.parseFrom)
      if fragments.items.size != 2 then fail("Rule payload needs guard and effect Expr fragments")
      val modelAt = 8 + count + size
      val modelSize = data(modelAt).toInt
      if modelSize < 0 || modelSize > data.size - modelAt - 1 then fail("incomplete Rule closure")
      val model =
        decode(data.slice(modelAt + 1, modelAt + 1 + modelSize).mkString)(ir.Model.parseFrom)
      val positionAt = modelAt + 1 + modelSize
      val positionSize = data(positionAt).toInt
      if positionSize < 0 || positionSize > data.size - positionAt - 1 then
        fail("incomplete type positions")
      val positions =
        typePositions(data.slice(positionAt + 1, positionAt + 1 + positionSize).mkString, context)
      val proofs = data.drop(positionAt + 1 + positionSize)
      if proofs.size % 4 != 0 then fail("incomplete curried-call evidence")
      val curried = proofs
        .grouped(4)
        .map { group =>
          CoreBindings.CurriedCall(
            decode(group(0))(ir.Call.parseFrom),
            decode(group(1))(ir.Function.parseFrom),
            group(2).split(',').map(_.toInt).toVector,
            Option(group(3)).filter(_.nonEmpty)
          )
        }
        .toVector
      if !model.copy(functions = Nil, types = Nil).equals(ir.Model()) then
        fail("Rule closure contains unrelated declarations")
      Packet(
        data(1),
        data(2),
        data(3),
        decode(data(4))(ir.Position.parseFrom),
        decode(data(5))(ir.Position.parseFrom),
        params,
        fragments.items.head,
        fragments.items(1),
        model,
        curried,
        positions
      )
    catch
      case error: IllegalArgumentException =>
        throw IllegalArgumentException(
          s"$context: malformed retained Rule payload: ${error.getMessage}",
          error
        )
      case error: com.google.protobuf.InvalidProtocolBufferException =>
        throw IllegalArgumentException(s"$context: malformed retained Rule payload", error)
      case error: IndexOutOfBoundsException =>
        throw IllegalArgumentException(s"$context: incomplete retained Rule payload", error)

  private def checkedSource(
      source: Machine[?, ?, ?],
      owner: String,
      family: String,
      machine: String,
      context: String
  ): Unit =
    require(source ne null, s"$context: source is not initialized")
    require(owner.nonEmpty && machine.nonEmpty, s"$context: missing Rule provenance")
    require(
      source.getClass.getName == owner.replace("$.", "$"),
      s"$context: stale Rule owner $owner"
    )
    require(
      source.headerIdentity.contains(family -> machine),
      s"$context: pending or stale Rule declaration registration"
    )

  private def schema(source: Machine[?, ?, ?], context: String): FiniteSchema =
    source.headerSchemaKeys.headOption.flatten
      .flatMap(source.fs.schemaFor)
      .getOrElse(throw IllegalArgumentException(s"$context: no retained state schema"))

  private def inputTypes(decl: ActionDecl, context: String): Vector[ir.Type] =
    decl.domains.zipWithIndex.flatMap { (domain, index) =>
      decl.schemaKeys
        .lift(index)
        .flatten
        .orElse(decl.tokens.lift(index).flatten.flatMap(_.schemaKey))
        .flatMap(domain.schemaFor)
        .getOrElse(throw IllegalArgumentException(s"$context: no retained input schema $index"))
        .types
    }.toVector

  def registration[S, O, F](
      source: Machine[S, O, F],
      decl: ActionDecl,
      values: Option[List[Any]],
      c: Case[S, O, F],
      effect: (S, List[Any]) => List[Step[S, O, F]],
      packet: Packet
  ): Registration =
    val context = s"Machine ${packet.family}.${packet.machine} Rule"
    checkedSource(source, packet.owner, packet.family, packet.machine, context)
    val action = Signature.action(decl.boundTo(decl.copy(examples = Nil)))
    val state = schema(source, context)
    require(
      packet.params.headOption.exists(_.getType.equals(state.root)),
      s"$context: wrong retained Rule state type"
    )
    require(
      packet.params.size == 1 || packet.params.size == action.inputs.size + 1,
      s"$context: wrong retained effect arity for ${action.id}"
    )
    if packet.params.size > 1 then
      require(
        packet.params.tail.map(_.`type`).equals(action.inputs.map(_.`type`)),
        s"$context: wrong retained effect input types for ${action.id}"
      )
    val types = packet.model.types.toVector ++ state.types ++ inputTypes(decl, context)
    val native = new Construction(context, ir.Model(types = types), "")
    val cls = values.map { values =>
      require(values.size == action.inputs.size, s"$context: wrong class arity for ${action.id}")
      values.zip(action.inputs).map((value, input) => native.inputValue(value, input.getType))
    }
    Registration(source, decl, values, cls, c.guard, effect, packet, types)

  def disabled[S, O, F](
      source: Machine[S, O, F],
      declarations: Seq[ActionDecl],
      data: Seq[String]
  ): Vector[(String, Bound[S, O, F])] =
    val context = "Machine disabled Rules"
    require(
      data.headOption.contains("ordinary-disabled-v1") && data.size >= 6,
      s"$context: malformed retained disabled payload"
    )
    checkedSource(source, data(1), data(2), data(3), context)
    val (offset, positions) = try
      val count = data(4).toInt
      require(
        count >= 0 && count <= data.size - 5 && data.size - 5 - count == 2 * declarations.size,
        s"$context: malformed retained disabled payload"
      )
      (5 + count, typePositions(data.slice(5, 5 + count).mkString, context))
    catch
      case scala.util.control.NonFatal(error) =>
        throw IllegalArgumentException(s"$context: malformed retained disabled payload", error)
    declarations.zipWithIndex.map { (decl, index) =>
      val position = try decode(data(offset + 1 + 2 * index))(ir.Position.parseFrom)
      catch
        case scala.util.control.NonFatal(error) =>
          throw IllegalArgumentException(s"$context: malformed disabled position", error)
      val bound = Bound.Disabled[S, O, F]()
      bound.retainedDisabled = Some(
        Disabled(
          bound,
          source,
          decl,
          data(1),
          data(2),
          data(3),
          position,
          positions
        )
      )
      data(offset + 2 * index) -> bound
    }.toVector

  def collect(machine: Machine[?, ?, ?]): Contribution =
    require(machine ne null, "Machine Rules: declaration is not initialized")
    val source = machine.ruleSource
    val context = "Machine ordinary Rules"
    def fail(message: String): Nothing = throw IllegalArgumentException(s"$context: $message")
    if source == null || source.headerIdentity.isEmpty then
      fail("declaration registration is pending or absent")
    machine.coreOperations.foreach(operation =>
      fail(s"${operation.name} rule/effect contributions are unsupported")
    )
    source.rules match
      case rules: Rules[?, ?, ?, ?] if rules.facetSource.isEmpty => ()
      case _ => fail("Facets or non-direct Rules contributions are unsupported")
    val functions = mutable.ArrayBuffer.empty[ir.Function]
    val types = mutable.ArrayBuffer.empty[ir.Type]
    val curried = mutable.ArrayBuffer.empty[CoreBindings.CurriedCall]
    val positions = mutable.Map.empty[String, ir.Position]
    def retainPositions(locations: Seq[ir.Type]): Unit = locations.foreach { t =>
      if !validTypePosition(t) then fail("malformed type-definition position")
      if positions.get(t.name).exists(p => !p.equals(t.getPosition)) then
        fail(s"type ${t.name} has conflicting retained definition positions")
      positions(t.name) = t.getPosition
    }
    val steps = machine.table.map { (decl, bound) =>
      val action = Signature.action(decl.boundTo(decl.copy(examples = Nil)))
      val firstCurried = curried.size
      val (name, position, state, rows) = bound match
        case Bound.Ruled(rules) =>
          if rules.isEmpty then fail(s"action ${action.id} has no retained Rule rows")
          val retained = rules.map { rule =>
            val retained = rule.retainedRule.getOrElse(
              fail(s"action ${action.id} has no captured Rule metadata")
            )
            val r = retained.registration
            if !retained.rule.eq(rule) || !r.source.eq(source) || !r.declaration.eq(decl) ||
              !rule.decl.eq(decl) || !r.values.eq(rule.values) || !r.guard.eq(
                rule.guard
              ) || !r.effect.eq(rule.effect)
            then fail(s"action ${action.id} has stale Rule/action/class/callback provenance")
            checkedSource(source, r.packet.owner, r.packet.family, r.packet.machine, context)
            if !r.packet.params.headOption.exists(_.getType.equals(schema(source, context).root)) ||
              !(r.packet.params.size == 1 || r.packet.params.size == action.inputs.size + 1) ||
              (r.packet.params.size > 1 && !r.packet.params.tail
                .map(_.`type`)
                .equals(action.inputs.map(_.`type`)))
            then fail(s"action ${action.id} has inconsistent retained Rule parameter arity/types")
            val names = RuleLowering.stateName(action.inputs, fail) +: action.inputs.map(_.name)
            val renamed = r.packet.params.map(_.name).zip(names).toMap
            if r.packet.params.map(_.name).distinct.size != r.packet.params.size ||
              r.packet.params.exists(p => p.name.isEmpty || p.`type`.isEmpty)
            then fail("missing or duplicate retained Rule parameters")
            functions ++= r.packet.model.functions
            types ++= r.types
            retainPositions(r.packet.typePositions)
            curried ++= r.packet.curriedCalls.map { proof =>
              if proof.owner.nonEmpty then proof
              else
                proof.copy(call =
                  proof.call.copy(args =
                    proof.call.args.map(CoreBindings.renameVariables(_, renamed))
                  )
                )
            }
            r.cls.foreach { values =>
              if values.size != action.inputs.size then
                fail(s"wrong retained class arity for ${action.id}")
              values.zip(action.inputs).foreach { (value, input) =>
                if !RuleLowering
                    .valuesOf(input.getType, r.types.map(t => t.name -> t).toMap, fail)
                    .exists(_.equals(value))
                then fail(s"class of ${action.id} has a value outside its retained typed domain")
              }
            }
            r -> RuleLowering.Row(
              r.cls,
              CoreBindings.renameVariables(r.packet.guard, renamed),
              CoreBindings.renameVariables(r.packet.effect, renamed),
              Some(r.packet.position)
            )
          }
          val first = retained.head._1
          if retained.exists((r, _) =>
              r.packet.machine != first.packet.machine || r.packet.owner != first.packet.owner
            )
          then fail(s"action ${action.id} has conflicting Rule owners")
          (
            first.packet.machine,
            first.packet.binding,
            first.packet.params.head.getType,
            retained.map(_._2)
          )
        case off @ Bound.Disabled() =>
          val r = off.retainedDisabled.getOrElse(
            fail(s"action ${action.id} has no retained disabled provenance")
          )
          if !r.bound.eq(off) || !r.source.eq(source) || !r.declaration.eq(decl) then
            fail(s"action ${action.id} has stale disabled provenance")
          checkedSource(source, r.owner, r.family, r.machine, context)
          retainPositions(r.typePositions)
          val selected = schema(source, context)
          types ++= selected.types ++ inputTypes(decl, context)
          (r.machine, r.position, selected.root, Vector.empty[RuleLowering.Row])
        case _ => fail(s"action ${action.id} is not a direct ordinary Rule contribution")
      val function = RuleLowering.lowered(
        name,
        state,
        action.name,
        action.inputs,
        rows,
        Some(position),
        types.map(t => t.name -> t).toMap,
        fail
      )
      for index <- firstCurried until curried.size if curried(index).owner.isEmpty do
        curried(index) = curried(index).copy(owner = Some(function.name))
      functions += function
      ir.StepBinding(action.id, function.name, Some(position))
    }
    val locatedTypes =
      types.map(t => positions.get(t.name).fold(t)(p => t.copy(position = Some(p))))
    val closure = CoreBindings.dependencies(
      Nil,
      steps.map(_.function),
      functions.toVector,
      locatedTypes.toVector,
      context,
      curried.distinct.toVector
    )
    Contribution(steps, closure.functions, closure.types, closure.channels, closure.holes)
