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
      typePositions: Vector[ir.Type] = Vector.empty,
      effectIdentity: Option[String] = None,
      derived: Boolean = false,
      sites: Vector[FacetBindings.Site] = Vector.empty,
      guardPlacements: Vector[String] = Vector.empty
  )
  final case class Registration(
      source: MachineDeclaration[?, ?, ?],
      declaration: ActionDecl,
      values: Option[List[Any]],
      cls: Option[Seq[ir.Value]],
      guard: AnyRef,
      effect: AnyRef,
      packet: Packet,
      types: Vector[ir.Type],
      facet: Option[FacetBindings.Retained] = None
  )
  final case class Retained(rule: Rule[?, ?, ?], registration: Registration)
  final case class Disabled(
      bound: Bound[?, ?, ?],
      source: MachineDeclaration[?, ?, ?],
      declaration: ActionDecl,
      owner: String,
      family: String,
      machine: String,
      position: ir.Position,
      typePositions: Vector[ir.Type] = Vector.empty,
      facet: Option[(FacetBindings.Origin, FacetBindings.Definition)] = None
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

  private def checkedGuardPlacements(packet: Packet, context: String): Unit =
    def exists(value: ir.Expr, path: String): Boolean =
      if path.isEmpty then true
      else
        value.kind match
          case ir.Expr.Kind.Binary(binary) =>
            (if path.head == 'l' then binary.left
             else if path.head == 'r' then binary.right
             else None) .exists(exists(_, path.tail))
          case _ => false
    require(
      packet.guardPlacements.distinct.size == packet.guardPlacements.size &&
        packet.guardPlacements.forall(exists(packet.guard, _)),
      s"$context: stale guard placement path provenance"
    )

  def packet(input: Seq[String], context: String): Packet =
    def fail(message: String): Nothing = throw IllegalArgumentException(s"$context: $message")
    try
      if input.headOption.contains("rule-guard-placements-v1") then
        val count = input(1).toInt
        require(
          count > 0 && count < input.size - 2,
          s"$context: incomplete guard placement payload"
        )
        val placements = input.drop(3 + count).toVector
        require(
          input(
            2 + count
          ).toInt == placements.size && placements.distinct.size == placements.size &&
            placements.forall(_.forall(c => c == 'l' || c == 'r')),
          s"$context: malformed guard placements"
        )
        val result = packet(input.slice(2, 2 + count), context).copy(guardPlacements = placements)
        checkedGuardPlacements(result, context)
        return result
      if input.headOption.contains("derived-rule-v1") then
        val count = input(1).toInt
        require(count > 0 && count <= input.size - 2, s"$context: incomplete derived Rule")
        return packet(input.slice(2, 2 + count), context).copy(
          derived = true,
          sites = FacetBindings.sites(input.drop(2 + count))
        )
      val identity =
        if input.headOption.contains("ordinary-rule-v3") then Some(input(1)) else None
      val data = identity.fold(input)(_ => "ordinary-rule-v2" +: input.drop(2))
      if data.headOption != Some("ordinary-rule-v2") || data.size < 10 then
        fail("missing or unsupported Rule payload")
      val count = data(6).toInt
      if count < 1 || count > 2 then fail("invalid retained Rule arity")
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
        positions,
        identity
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
      source: MachineDeclaration[?, ?, ?],
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

  private[framework] def schema(
      source: MachineDeclaration[?, ?, ?],
      context: String
  ): FiniteSchema =
    source.headerSchemaKeys.headOption.flatten
      .map(key => FiniteSchema.selected(source.fs, key, context))
      .getOrElse(throw IllegalArgumentException(s"$context: no retained state schema"))

  private def inputTypes(decl: ActionDecl, context: String): Vector[ir.Type] =
    decl.domains.zipWithIndex.flatMap { (domain, index) =>
      decl.schemaKeys
        .lift(index)
        .flatten
        .map(key => FiniteSchema.selected(domain, key, context))
        .getOrElse(throw IllegalArgumentException(s"$context: no retained input schema $index"))
        .types
    }.toVector

  private[framework] def inputParametersMatch(
      decl: ActionDecl,
      actual: Seq[ir.Param],
      flattened: Seq[ir.Param],
      context: String
  ): Boolean =
    if actual.isEmpty then true
    else
      val expected = decl.nativeInputs match
        case Some(record: ActionInputs.Record[?]) =>
          Vector(
            record.domain.schema
              .getOrElse(
                throw IllegalArgumentException(s"$context: no retained native input schema")
              )
              .root
          )
        case _ => flattened.map(_.getType)
      actual.map(_.getType).equals(expected)

  private def guardParametersMatch(guard: AnyRef, params: Seq[ir.Param]): Boolean = guard match
    case _: InputGuard[?] => params.size == 2
    case _                => true

  def registration[S, O, F](
      source: MachineDeclaration[S, O, F],
      decl: ActionDecl,
      values: Option[List[Any]],
      guard: S => Boolean,
      effect: (S, List[Any]) => List[Step[S, O, F]],
      packet: Packet,
      facet: Option[FacetBindings.Retained] = None
  ): Registration =
    val context = s"Machine ${packet.family}.${packet.machine} Rule"
    facet match
      case Some(retained) =>
        FacetBindings.checkedOrigin(source, retained.origin, retained.template.definition)
        require(retained.template.packet.equals(packet), s"$context: stale facet template packet")
        require(source.headerIdentity.nonEmpty, s"$context: pending facet assembly header")
      case None if packet.derived =>
        require(source.ruleSource.headerIdentity.nonEmpty, s"$context: pending derivation source")
        require(
          packet.owner.nonEmpty && packet.machine.nonEmpty,
          s"$context: missing derivation owner"
        )
      case None => checkedSource(source, packet.owner, packet.family, packet.machine, context)
    val action = Signature.action(decl.boundTo(decl.copy(examples = Nil)))
    val state = schema(source.ruleSource, context)
    require(
      packet.params.headOption.exists(_.getType.equals(state.root)),
      s"$context: wrong retained Rule state type"
    )
    require(
      packet.params.size == 1 || (decl.nativeInputs.nonEmpty && packet.params.size == 2),
      s"$context: wrong retained effect arity for ${action.id}"
    )
    require(
      guardParametersMatch(guard, packet.params),
      s"$context: input-reading guard needs retained input"
    )
    if packet.params.size > 1 then
      require(
        inputParametersMatch(decl, packet.params.tail, action.inputs, context),
        s"$context: wrong retained effect input types for ${action.id}"
      )
    val types = packet.model.types.toVector ++ state.types ++ inputTypes(decl, context)
    val native = new Construction(context, ir.Model(types = types), "")
    val cls = values.map { values =>
      require(values.size == action.inputs.size, s"$context: wrong class arity for ${action.id}")
      values.zip(action.inputs).map((value, input) => native.inputValue(value, input.getType))
    }
    Registration(source, decl, values, cls, guard, effect, packet, types, facet)

  def retainedDisabled[S, O, F](
      source: MachineDeclaration[S, O, F],
      declarations: Seq[ActionDecl],
      data: Seq[String],
      origin: Option[FacetBindings.Origin] = None
  ): Vector[(String, Bound[S, O, F])] =
    val context = "Machine disabled Rules"
    val (payload, facet) = if data.headOption.contains("facet-disabled-v1") then
      val (definition, ordinary, sites) = FacetBindings.unwrap(data, "facet-disabled-v1")
      require(sites.isEmpty, s"$context: disabled template has contextual sites")
      val token =
        origin.getOrElse(throw IllegalArgumentException(s"$context: no facet source token"))
      FacetBindings.checkedOrigin(source, token, definition)
      require(
        ordinary.lift(1).contains(definition.traitName),
        s"$context: stale disabled facet owner"
      )
      ordinary -> Some(token -> definition)
    else data -> None
    disabledRows(source, declarations, payload, facet)

  private def disabledRows[S, O, F](
      source: MachineDeclaration[S, O, F],
      declarations: Seq[ActionDecl],
      data: Seq[String],
      facet: Option[(FacetBindings.Origin, FacetBindings.Definition)]
  ): Vector[(String, Bound[S, O, F])] =
    val context = "Machine disabled Rules"
    require(
      data.headOption.contains("ordinary-disabled-v1") && data.size >= 6,
      s"$context: malformed retained disabled payload"
    )
    if facet.isEmpty then checkedSource(source, data(1), data(2), data(3), context)
    else require(source.headerIdentity.nonEmpty, s"$context: pending facet assembly header")
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
          positions,
          facet
        )
      )
      data(offset + 2 * index) -> bound
    }.toVector

  def collect(machine: MachineDeclaration[?, ?, ?]): Contribution =
    require(machine ne null, "Machine Rules: declaration is not initialized")
    if machine.derivationParent.nonEmpty then DerivationBindings.collect(machine)
    else
      require(
        machine.ruleSource.headerIdentity.nonEmpty,
        "Machine Rules: declaration registration is pending or absent"
      )
      val table = DerivationBindings.acquiredTable(machine, "Machine Rules")
      val book = machine.acquireRuleBook()
      collectEntries(machine, table, book)

  def collectEntries(
      source: MachineDeclaration[?, ?, ?],
      table: Vector[(ActionDecl, Bound[?, ?, ?])],
      book: RuleBook[?, ?, ?],
      declaring: Option[MachineDeclaration[?, ?, ?]] = None,
      capturedRows: (ActionDecl, Vector[Registration]) => Unit = (_, _) => (),
      capturedProofs: Vector[CoreBindings.CurriedCall] => Unit = _ => ()
  ): Contribution =
    val context = "Machine ordinary Rules"
    def fail(message: String): Nothing = throw IllegalArgumentException(s"$context: $message")
    if source == null || source.ruleSource.headerIdentity.isEmpty then
      fail("declaration registration is pending or absent")
    val contributors: Option[Vector[FacetBindings.Origin]] =
      if declaring.nonEmpty then None else FacetBindings.contributors(source.ruleSource, book)
    if contributors.isEmpty then
      book match
        case rules if isDirectRuleBook(rules) && rules.facetSource.isEmpty => ()
        case _ if declaring.nonEmpty                                       => ()
        case _ => fail("Facets or non-direct Rules contributions are unsupported")
    def checkedFacet(origin: FacetBindings.Origin, definition: FacetBindings.Definition): Unit =
      if !contributors.exists(_.exists(_.eq(origin))) then
        fail("foreign facet contributor RuleBook")
      FacetBindings.checkedOrigin(source.ruleSource, origin, definition)
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
    val steps = table.map { (decl, bound) =>
      val action = Signature.action(decl.boundTo(decl.copy(examples = Nil)))
      val firstCurried = curried.size
      val (name, position, state, rows) = bound match
        case Bound.Ruled(rules) =>
          if rules.isEmpty then fail(s"action ${action.id} has no retained Rule rows")
          val retained = rules.map { rule =>
            val retained = rule.retainedRule.getOrElse(
              fail(s"action ${action.id} has no captured Rule metadata")
            )
            val original = retained.registration
            checkedGuardPlacements(original.packet, context)
            if original.packet.effectIdentity.nonEmpty && !rule.capturePublished then
              fail("stale Rule publication provenance")
            if contributors.nonEmpty && original.facet.isEmpty && !original.packet.derived then
              fail("facet Rule row has no source provenance")
            val r = original.facet.fold(original) { facet =>
              checkedFacet(facet.origin, facet.template.definition)
              if !original.packet.equals(facet.template.packet) then
                fail("stale facet template packet")
              val packet = FacetBindings.instantiate(source.ruleSource, facet.template)
              original.copy(packet = packet, types = original.types ++ packet.model.types)
            }
            val row = if r.packet.derived && r.packet.sites.nonEmpty then
              val packet = FacetBindings
                .instantiate(source.ruleSource, r.packet, r.packet.sites, r.packet.owner)
                .copy(owner = r.packet.owner, family = r.packet.family, machine = r.packet.machine)
              r.copy(packet = packet, types = r.types ++ packet.model.types)
            else r
            if !retained.rule.eq(rule) || !r.source.eq(source) || !r.declaration.eq(decl) ||
              !rule.decl.eq(decl) || !r.values.eq(rule.values) || !r.guard.eq(
                rule.guard
              ) || !r.effect.eq(rule.effect)
            then fail(s"action ${action.id} has stale Rule/action/class/callback provenance")
            if r.packet.derived then
              checkedSource(
                declaring.getOrElse(fail("no live deriving declaration")),
                r.packet.owner,
                r.packet.family,
                r.packet.machine,
                context
              )
            else
              checkedSource(
                source.ruleSource,
                r.packet.owner,
                r.packet.family,
                r.packet.machine,
                context
              )
            if !r.packet.params.headOption
                .exists(_.getType.equals(schema(source.ruleSource, context).root)) ||
              !(r.packet.params.size == 1 ||
                (decl.nativeInputs.nonEmpty && r.packet.params.size == 2)) ||
              !guardParametersMatch(r.guard, r.packet.params) ||
              !inputParametersMatch(decl, r.packet.params.tail, action.inputs, context)
            then fail(s"action ${action.id} has inconsistent retained Rule parameter arity/types")
            val names = RuleLowering.stateName(action.inputs, fail) +: action.inputs.map(_.name)
            val renamed = r.packet.params.map(_.name).zip(names).toMap
            if r.packet.params.map(_.name).distinct.size != r.packet.params.size ||
              r.packet.params.exists(p => p.name.isEmpty || p.`type`.isEmpty)
            then fail("missing or duplicate retained Rule parameters")
            val loweredPacket = if row.facet.nonEmpty || row.packet.derived then
              FacetBindings.renameRoot(row.packet, renamed)
            else
              r.packet.copy(
                guard = CoreBindings.renameVariables(row.packet.guard, renamed),
                effect = CoreBindings.renameVariables(row.packet.effect, renamed),
                curriedCalls = row.packet.curriedCalls.map { proof =>
                  if proof.owner.nonEmpty then proof
                  else
                    proof.copy(call =
                      proof.call.copy(args =
                        proof.call.args.map(CoreBindings.renameVariables(_, renamed))
                      )
                    )
                }
              )
            functions ++= row.packet.model.functions
            types ++= row.types
            retainPositions(row.packet.typePositions)
            curried ++= loweredPacket.curriedCalls.map { proof =>
              if proof.owner.exists(owner => !row.packet.model.functions.exists(_.name == owner))
              then fail("stale curried-call evidence owner is not a retained helper")
              proof
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
            row.copy(packet =
              loweredPacket.copy(params =
                loweredPacket.params.zip(names).map((p, n) => p.copy(name = n))
              )
            ) -> RuleLowering.Row(
              r.cls,
              loweredPacket.guard,
              loweredPacket.effect,
              Some(r.packet.position)
            )
          }
          capturedRows(decl, retained.map(_._1))
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
          r.facet match
            case Some((origin, definition)) => checkedFacet(origin, definition)
            case None                       =>
              if contributors.nonEmpty then fail("facet disabled row has no source provenance")
              checkedSource(source, r.owner, r.family, r.machine, context)
          retainPositions(r.typePositions)
          val selected = schema(source, context)
          types ++= selected.types ++ inputTypes(decl, context)
          (
            if r.facet.nonEmpty then source.headerIdentity.get._2 else r.machine,
            r.position,
            selected.root,
            Vector.empty[RuleLowering.Row]
          )
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
    capturedProofs(curried.distinct.toVector)
    Contribution(steps, closure.functions, closure.types, closure.channels, closure.holes)
