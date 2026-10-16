package framework.dsl

import scala.collection.mutable
import io.temporal.server.api.umpire.v1 as ir

private[framework] object DerivationBindings:
  final case class OperationSite(
      source: Machine[?, ?, ?],
      owner: String,
      family: String,
      machine: String,
      kind: String,
      position: ir.Position,
      operands: Int
  )
  def operationSite(source: Machine[?, ?, ?], data: Seq[String]): OperationSite =
    try
      require(
        data.size == 7 && data.head == "derivation-operation-v1",
        "Machine derivation Rules: malformed operation callsite"
      )
      OperationSite(
        source,
        data(1),
        data(2),
        data(3),
        data(4),
        ir.Position.parseFrom(java.util.Base64.getDecoder.decode(data(5))),
        data(6).toInt
      )
    catch
      case error: IllegalArgumentException =>
        throw IllegalArgumentException(
          "Machine derivation Rules: malformed operation callsite",
          error
        )
  final case class Adaptation(
      rule: Rule[?, ?, ?],
      original: Rule[?, ?, ?],
      binding: StepBinding[?, ?, ?],
      callback: AnyRef
  )
  final case class GroupOrigin(
      group: RuleGroup[?, ?, ?],
      source: Machine[?, ?, ?],
      declaration: ActionDecl,
      packet: RuleBindings.Packet
  )
  final case class RowEvidence(
      rule: Rule[?, ?, ?],
      retained: Option[RuleBindings.Retained],
      adapted: Option[Adaptation],
      original: Option[RowEvidence],
      binding: Option[OperandEvidence]
  )
  final case class OperandEvidence(
      operand: Rebinding[?, ?, ?],
      captured: Option[Captured[AnyRef]],
      effect: Option[Either[String, RuleBindings.Packet]],
      group: Option[GroupOrigin],
      rows: Vector[RowEvidence]
  )
  final case class Snapshot(
      table: Vector[(ActionDecl, Bound[?, ?, ?])],
      rows: Vector[RowEvidence],
      disabled: Vector[(Bound[?, ?, ?], Option[RuleBindings.Disabled])]
  )
  def operandEvidence(operand: Rebinding[?, ?, ?]): OperandEvidence = operand match
    case binding: StepBinding[?, ?, ?] =>
      OperandEvidence(binding, binding.captured, binding.derivationEffect, None, Vector.empty)
    case group: RuleGroup[?, ?, ?] =>
      OperandEvidence(group, None, None, group.derivationOrigin, group.rules.map(rowEvidence))
  private def rowEvidence(rule: Rule[?, ?, ?]): RowEvidence =
    RowEvidence(
      rule,
      rule.retainedRule,
      rule.adaptedRule,
      rule.adaptedRule.map(a => rowEvidence(a.original)),
      rule.adaptedRule.map(a => operandEvidence(a.binding))
    )
  def snapshot(table: Vector[(ActionDecl, Bound[?, ?, ?])]): Snapshot =
    Snapshot(
      table,
      table.flatMap { (_, bound) =>
        bound match
          case Bound.Ruled(rows) => rows.map(rowEvidence)
          case _                 => Vector.empty
      },
      table.collect { case (_, bound @ Bound.Disabled()) => bound -> bound.retainedDisabled }
    )
  sealed trait Event
  final case class Applied(operation: CoreBindings.Operation) extends Event
  final case class Restricted(
      source: Machine[?, ?, ?],
      before: Snapshot,
      keep: Vector[ActionDecl]
  ) extends Event

  final private case class Entry(
      declaration: ActionDecl,
      step: ir.StepBinding,
      functions: Vector[ir.Function],
      types: Vector[ir.Type],
      proofs: Vector[CoreBindings.CurriedCall] = Vector.empty
  )
  final private case class Replay(
      entries: Vector[Entry],
      rootRules: Option[Map[ActionDecl, Vector[RuleBindings.Registration]]],
      ownRules: Map[ActionDecl, Vector[RuleBindings.Registration]] = Map.empty
  )

  final private class Evidence(context: String):
    def fail(message: String): Nothing = throw IllegalArgumentException(s"$context: $message")
    def same[A <: AnyRef](before: Option[A], after: Option[A]): Boolean =
      before.isEmpty && after.isEmpty || before.zip(after).exists((a, b) => a.eq(b))
    def checkedOperand(evidence: OperandEvidence): Unit = evidence.operand match
      case binding: StepBinding[?, ?, ?] =>
        if binding.derivationEffect.nonEmpty && !binding.derivationPublished then
          fail("stale binding publication provenance")
        if !same(evidence.captured, binding.captured) || !same(
            evidence.effect,
            binding.derivationEffect
          )
        then fail("stale captured operand evidence")
      case group: RuleGroup[?, ?, ?] =>
        if group.derivationOrigin.nonEmpty && !group.originPublished then
          fail("stale RuleGroup publication provenance")
        if !same(evidence.group, group.derivationOrigin) then
          fail("stale copied rule group provenance")
        evidence.rows.foreach(checkedRow)
    def checkedRow(evidence: RowEvidence): Unit =
      val rule = evidence.rule
      if rule.retainedRule.exists(
          _.registration.packet.effectIdentity.nonEmpty
        ) && !rule.capturePublished
      then fail("stale Rule publication provenance")
      if !same(evidence.retained, rule.retainedRule) || !same(evidence.adapted, rule.adaptedRule)
      then fail("stale live Rule/callback evidence")
      evidence.original.foreach(checkedRow)
      evidence.binding.foreach(checkedOperand)
      evidence.adapted.foreach { adapted =>
        if !adapted.rule.eq(rule) || !adapted.callback.eq(rule.effect) || !rule.decl.eq(
            adapted.original.decl
          ) ||
          !rule.values.eq(adapted.original.values) || !rule.guard.eq(adapted.original.guard)
        then fail("stale adapted Rule/class/guard/callback provenance")
      }
    def checkedSnapshot(evidence: Snapshot): Unit =
      evidence.rows.foreach(checkedRow)
      evidence.disabled.foreach((bound, retained) =>
        if !same(retained, bound.retainedDisabled) then fail("stale disabled source evidence")
      )

  def acquiredTable(
      machine: Machine[?, ?, ?],
      context: String
  ): Vector[(ActionDecl, Bound[?, ?, ?])] =
    val retained = machine.acquiredRuleTable.getOrElse {
      val table = machine.table
      machine.acquiredRuleTable.getOrElse(snapshot(table))
    }
    new Evidence(context).checkedSnapshot(retained)
    retained.table

  def collect(machine: Machine[?, ?, ?]): RuleBindings.Contribution =
    val context = "Machine derivation Rules"
    val evidence = new Evidence(context)
    def fail(message: String): Nothing = evidence.fail(message)
    def id(decl: ActionDecl): String =
      decl.sourceIdentity.getOrElse(fail("action has no retained identity"))
    def book(source: Machine[?, ?, ?]): RuleBook[?, ?, ?] =
      source.ruleSource.acquireRuleBook()
    def closure(entries: Vector[Entry]): CoreBindings.Dependencies =
      CoreBindings.dependencies(
        Nil,
        entries.map(_.step.function),
        entries.flatMap(_.functions),
        entries.flatMap(_.types),
        context,
        entries.flatMap(_.proofs).distinct
      )
    def validate(stage: Replay): Replay =
      closure(stage.entries): Unit
      stage
    def fromContribution(
        table: Vector[(ActionDecl, Bound[?, ?, ?])],
        steps: Vector[ir.StepBinding],
        functions: Vector[ir.Function],
        types: Vector[ir.Type]
    ): Vector[Entry] =
      require(steps.size == table.size, s"$context: inconsistent retained source table")
      table.map(_._1).zip(steps).map((decl, step) => Entry(decl, step, functions, types))
    def core(table: Vector[(ActionDecl, Bound[?, ?, ?])]): Vector[Entry] =
      val result = CoreBindings.collectEntries(table, context)
      fromContribution(table, result.steps, result.functions, result.types)
    def ruled(
        source: Machine[?, ?, ?],
        table: Vector[(ActionDecl, Bound[?, ?, ?])],
        declaring: Option[Machine[?, ?, ?]]
    ): (Vector[Entry], Map[ActionDecl, Vector[RuleBindings.Registration]]) =
      val rows = mutable.Map.empty[ActionDecl, Vector[RuleBindings.Registration]]
      var proofs = Vector.empty[CoreBindings.CurriedCall]
      val result = RuleBindings.collectEntries(
        source,
        table,
        book(source),
        declaring,
        (decl, rs) => rows(decl) = rs,
        value => proofs = value
      )
      fromContribution(table, result.steps, result.functions, result.types)
        .map(e =>
          e.copy(proofs = CoreBindings.reachedProofs(Seq(e.step.function), e.functions, proofs))
        ) -> rows.toMap
    def lowered(
        decl: ActionDecl,
        rows: Vector[RuleBindings.Registration],
        declaring: Machine[?, ?, ?]
    ): Entry =
      val action = Signature.action(decl.boundTo(decl.copy(examples = Nil)))
      val first = rows.headOption.getOrElse(fail(s"action ${action.id} has no captured Rule rows"))
      val name = declaring.headerIdentity.map(_._2).getOrElse(fail("pending deriving declaration"))
      val types = rows.flatMap(_.types).map { t =>
        val positions =
          rows.flatMap(_.packet.typePositions).filter(_.name == t.name).map(_.getPosition).distinct
        if positions.size > 1 then
          fail(s"type ${t.name} has conflicting retained definition positions")
        positions.headOption.fold(t)(p => t.copy(position = Some(p)))
      }
      val function = RuleLowering.lowered(
        name,
        first.packet.params.head.getType,
        action.name,
        action.inputs,
        rows.map(r =>
          RuleLowering.Row(r.cls, r.packet.guard, r.packet.effect, Some(r.packet.position))
        ),
        Some(first.packet.binding),
        types.map(t => t.name -> t).toMap,
        fail
      )
      val proofs = rows
        .flatMap(_.packet.curriedCalls)
        .map(p => if p.owner.isEmpty then p.copy(owner = Some(function.name)) else p)
      val functions = rows.flatMap(_.packet.model.functions) :+ function
      Entry(
        decl,
        ir.StepBinding(action.id, function.name, Some(first.packet.binding)),
        functions,
        types,
        CoreBindings.reachedProofs(Seq(function.name), functions, proofs)
      )
    def checkedCapture(
        binding: StepBinding[?, ?, ?],
        declaring: Machine[?, ?, ?]
    ): Captured[AnyRef] =
      val capture =
        binding.captured.getOrElse(fail("rebind unsupported: missing captured function metadata"))
      if !binding.function.eq(capture.callable) then fail("stale binding callback provenance")
      val origin = capture.origin.getOrElse(fail("missing compiler-owned binding origin"))
      if !origin.declaration.eq(binding.decl) || !origin.action.contains(id(binding.decl)) then
        fail("stale binding/action provenance")
      if declaring.getClass.getName != origin.owner.replace("$.", "$") ||
        !declaring.headerIdentity.exists(_._2 == origin.machine)
      then fail("stale deriving declaration/binding provenance")
      capture
    def checkedPacket(
        binding: StepBinding[?, ?, ?],
        declaring: Machine[?, ?, ?],
        source: Machine[?, ?, ?]
    ): RuleBindings.Packet =
      val capture = checkedCapture(binding, declaring)
      val origin = capture.origin.get
      val packet = binding.derivationEffect.getOrElse(
        fail("rebind unsupported: missing captured rule-effect metadata")
      ) match
        case Left(message) => fail(message)
        case Right(packet) => packet
      if !packet.derived || !declaring.headerIdentity.contains(packet.family -> packet.machine) ||
        declaring.getClass.getName != packet.owner.replace(
          "$.",
          "$"
        ) || origin.owner != packet.owner || origin.machine != packet.machine
      then fail("stale deriving declaration/binding provenance")
      if packet.effectIdentity.forall(_.isEmpty) then fail("missing historical effect identity")
      if !packet.params.headOption.exists(
          _.getType.equals(RuleBindings.schema(source.ruleSource, context).root)
        )
      then fail("inconsistent retained rule-effect state type")
      val action = Signature.action(binding.decl.boundTo(binding.decl.copy(examples = Nil)))
      if packet.params.isEmpty || !(packet.params.size == 1 || packet.params.size == action.inputs.size + 1) ||
        (packet.params.size > 1 && !packet.params.tail
          .map(_.getType)
          .equals(action.inputs.map(_.getType)))
      then fail("inconsistent rule-effect parameter arity/types")
      val instantiated = if packet.sites.isEmpty then packet
      else
        FacetBindings
          .instantiate(source.ruleSource, packet, packet.sites, packet.owner)
          .copy(owner = packet.owner, family = packet.family, machine = packet.machine)
      val names = RuleLowering.stateName(action.inputs, fail) +: action.inputs.map(_.name)
      val renamed = instantiated.params.map(_.name).zip(names).toMap
      val normalized = FacetBindings
        .renameRoot(instantiated, renamed)
        .copy(params = instantiated.params.zip(names).map((p, n) => p.copy(name = n)))
      val scope = ir.Function(
        "$derivation-effect-scope",
        params = normalized.params,
        body = Some(normalized.effect)
      )
      CoreBindings.dependencies(
        Nil,
        Seq(scope.name),
        normalized.model.functions :+ scope,
        normalized.model.types,
        context,
        normalized.curriedCalls.map(p =>
          if p.owner.isEmpty then p.copy(owner = Some(scope.name)) else p
        )
      ): Unit
      normalized
    def bare(
        binding: StepBinding[?, ?, ?],
        source: Machine[?, ?, ?],
        declaring: Machine[?, ?, ?],
        originals: Vector[RuleBindings.Registration]
    ): (Entry, Vector[RuleBindings.Registration]) =
      val replacement = checkedPacket(binding, declaring, source)
      val identities = originals.map(
        _.packet.effectIdentity.getOrElse(
          fail("rebind unsupported: missing historical source effect identity")
        )
      )
      val groups = if originals.forall(_.cls.nonEmpty) then originals.groupBy(_.cls).values.toVector
      else Vector(originals)
      if groups.exists(_.map(_.packet.effectIdentity).distinct.size > 1) then
        fail("rebind merges rules with different effects")
      if identities.exists(_.isEmpty) then fail("missing historical source effect identity")
      val rows = originals.map { r =>
        val placedPaths = mutable.Set.empty[String]
        def placed(value: ir.Expr, path: String = ""): ir.Expr =
          if r.packet.guardPlacements.contains(path) then placedPaths += path: Unit
          val children = value.kind match
            case ir.Expr.Kind.Binary(binary) =>
              ir.Expr.Kind.Binary(
                binary.copy(
                  left = binary.left.map(placed(_, path + "l")),
                  right = binary.right.map(placed(_, path + "r"))
                )
              )
            case other => other
          value.copy(
            kind = children,
            position =
              if r.packet.guardPlacements.contains(path) then Some(replacement.position)
              else value.position
          )
        val guard = placed(r.packet.guard)
        if placedPaths.toSet != r.packet.guardPlacements.toSet then
          fail("stale guard placement path provenance")
        r.copy(
          packet = r.packet.copy(
            owner = replacement.owner,
            family = replacement.family,
            machine = replacement.machine,
            binding = replacement.binding,
            position = replacement.position,
            guard = guard,
            effect = replacement.effect,
            effectIdentity = replacement.effectIdentity,
            model = ir.Model(
              functions = r.packet.model.functions ++ replacement.model.functions,
              types = r.packet.model.types ++ replacement.model.types
            ),
            typePositions = r.packet.typePositions ++ replacement.typePositions,
            curriedCalls = r.packet.curriedCalls.filter(p =>
              p.owner.nonEmpty || CoreBindings.calls(r.packet.guard).exists(_.equals(p.call))
            ) ++ replacement.curriedCalls
          ),
          types = r.types ++ replacement.model.types
        )
      }
      lowered(binding.decl, rows, declaring) -> rows
    def replay(
        current: Machine[?, ?, ?],
        observed: Vector[(ActionDecl, Bound[?, ?, ?])],
        declaring: Machine[?, ?, ?]
    ): Replay =
      current.derivationParent match
        case None =>
          val rows = observed.filter { (_, bound) =>
            bound match
              case Bound.Function(_, _) => false
              case _                    => true
          }
          if rows.nonEmpty || isDirectRuleBook(book(current)) || book(current)
              .isInstanceOf[Facets[?, ?, ?]]
          then
            val (entries, rules) = ruled(current, observed, None)
            validate(Replay(entries, Some(rules)))
          else validate(Replay(core(observed), None))
        case Some(parent) if current.headerIdentity.nonEmpty =>
          val result = replay(parent, observed, current)
          val ids = result.entries.map(_.declaration).toSet
          val rules = result.rootRules
            .map(_.filter((decl, _) => ids(decl)))
            .getOrElse(Map.empty) ++ result.ownRules
          result.copy(
            rootRules =
              if result.rootRules.nonEmpty || result.ownRules.nonEmpty then Some(rules) else None,
            ownRules = Map.empty
          )
        case Some(parent) =>
          current.derivationEvent match
            case Some(Restricted(source, before, keep)) =>
              if !source.eq(parent) then fail("stale restriction source")
              evidence.checkedSnapshot(before)
              val expected = before.table.filter((decl, _) => keep.contains(decl))
              if expected.size != observed.size || !expected.zip(observed).forall {
                  case ((decl, bound), (actual, result)) => decl.eq(actual) && bound.eq(result)
                }
              then fail("stale restriction native bound/callback identity")
              val stage = replay(parent, before.table, declaring)
              validate(
                stage.copy(entries = stage.entries.filter(e => keep.contains(e.declaration)))
              )
            case Some(Applied(operation)) =>
              if !operation.source.exists(_.eq(parent)) then fail("stale operation source")
              val site = operation.site.getOrElse(
                fail(s"${operation.name} unsupported: missing compiler-owned operation callsite")
              )
              if !site.source.eq(
                  parent
                ) || site.kind != operation.name || site.operands != operation.rawOperands.size ||
                !declaring.headerIdentity.contains(
                  site.family -> site.machine
                ) || declaring.getClass.getName != site.owner.replace("$.", "$") ||
                site.position.file.isEmpty || site.position.line <= 0
              then fail("stale deriving operation callsite/source/kind/arity provenance")
              val sourceEvidence = operation.sourceEvidence.getOrElse(
                fail("unsupported operation: missing immutable source evidence")
              )
              evidence.checkedSnapshot(sourceEvidence)
              operation.operandEvidence.foreach(evidence.checkedOperand)
              if operation.operandEvidence.size != operation.rawOperands.size || !operation.operandEvidence
                  .zip(operation.rawOperands)
                  .forall((e, r) => e.operand.eq(r))
              then fail("missing ordered operand evidence")
              val native = operation.nativeEvidence.getOrElse(
                fail("unsupported operation: missing native adaptation evidence")
              )
              evidence.checkedSnapshot(native)
              val nativeReads =
                if operation.name == "rebind" then
                  operation.rawOperands.count(_.isInstanceOf[StepBinding[?, ?, ?]])
                else 0
              if operation.nativeSourceEvidence.size != nativeReads then
                fail("missing ordered native source acquisition evidence")
              operation.nativeSourceEvidence.foreach { retained =>
                evidence.checkedSnapshot(retained)
                validate(replay(parent, retained.table, declaring)): Unit
              }
              val stage = replay(parent, operation.sourceTable, declaring)
              val operands = operation.rawOperands.map {
                case binding: StepBinding[?, ?, ?] =>
                  val entry = if operation.name == "extend" && stage.rootRules.nonEmpty then
                    fail("extend with Rules needs a rule group, not a bare binding")
                  else
                    stage.rootRules.flatMap(_.get(binding.decl)) match
                      case Some(rows) => bare(binding, parent, declaring, rows)
                      case None       =>
                        val retained = checkedCapture(binding, declaring)
                        val action =
                          Signature.action(binding.decl.boundTo(binding.decl.copy(examples = Nil)))
                        val parameterTypes =
                          RuleBindings.schema(parent.ruleSource, context).root +: action.inputs.map(
                            _.getType
                          )
                        val capture = Some(retained).map { captured =>
                          binding.derivationEffect.collect {
                            case Right(packet) if packet.sites.nonEmpty => packet
                          } match
                            case None    => captured
                            case Some(_) =>
                              val packet = checkedPacket(binding, declaring, parent)
                              val replacements = packet.model.functions.map(f => f.name -> f).toMap
                              captured.copy(model =
                                captured.model.copy(
                                  functions = captured.model.functions.map(f =>
                                    replacements.getOrElse(f.name, f)
                                  ) ++ packet.model.functions.filterNot(f =>
                                    captured.model.functions.exists(_.name == f.name)
                                  ),
                                  types = (captured.model.types ++ packet.model.types)
                                    .groupBy(_.name)
                                    .toVector
                                    .sortBy(_._1)
                                    .map { (name, definitions) =>
                                      if definitions.map(_.clearPosition).distinct.size != 1 then
                                        fail(s"type $name has conflicting retained definitions")
                                      definitions.head
                                    }
                                )
                              )
                        }
                        if !capture.get.function.params.map(_.getType).equals(parameterTypes) then
                          fail("inconsistent captured complete-function state/input types")
                        core(
                          Vector(binding.decl -> Bound.Function(binding.function, capture))
                        ).head -> Vector.empty
                  (binding.decl, entry._1, Option(entry._2).filter(_.nonEmpty))
                case group: RuleGroup[?, ?, ?] =>
                  val origin = group.derivationOrigin.getOrElse(
                    fail("unsupported rule group: missing captured group provenance")
                  )
                  if !origin.group.eq(group) || !origin.source.eq(
                      parent
                    ) || group.rules.isEmpty || group.rules.exists(r =>
                      !r.decl.eq(origin.declaration)
                    )
                  then fail("stale rule group/source/target provenance")
                  val (entries, captured) = ruled(
                    parent,
                    Vector(origin.declaration -> Bound.Ruled(group.rules)),
                    Some(declaring)
                  )
                  val rows =
                    captured.getOrElse(origin.declaration, fail("missing derived Rule rows"))
                  if rows.exists(r =>
                      r.packet.owner != origin.packet.owner || !r.packet.binding
                        .equals(origin.packet.binding)
                    )
                  then fail("stale group callsite provenance")
                  (origin.declaration, entries.head, Some(rows))
              }
              val declarations = operands.map(_._1)
              if declarations.map(id).distinct.size != declarations.size then
                fail(s"${operation.name} binds an action twice")
              declarations.foreach { decl =>
                val exists = stage.entries.exists(e => id(e.declaration) == id(decl))
                if operation.name == "rebind" && !exists then
                  fail(s"rebind action ${id(decl)} which the source does not bind")
                if operation.name == "extend" && exists then
                  fail(s"extend action ${id(decl)} which the source binds already")
              }
              val replacements = operands.map((decl, entry, _) => decl -> entry).toMap
              val entries = if operation.name == "extend" then stage.entries ++ operands.map(_._2)
              else stage.entries.map(e => replacements.getOrElse(e.declaration, e))
              if !observed.map(_._1).equals(entries.map(_.declaration)) then
                fail("stale native operation table order/targets")
              val expected = if operation.name == "extend" then sourceEvidence.table ++ native.table
              else
                sourceEvidence.table.map((decl, bound) =>
                  decl -> native.table.find(_._1.eq(decl)).fold(bound)(_._2)
                )
              if !expected.zip(observed).forall { case ((decl, bound), (actual, result)) =>
                  decl.eq(actual) && bound.eq(result)
                }
              then fail("stale native operation bound/callback identity")
              validate(
                stage.copy(
                  entries = entries,
                  ownRules =
                    stage.ownRules ++ operands.flatMap((decl, _, rows) => rows.map(decl -> _))
                )
              )
            case None => replay(parent, observed, declaring)
    val table = acquiredTable(machine, context)
    val result = replay(machine, table, machine)
    val reached = closure(result.entries)
    RuleBindings.Contribution(
      result.entries.map(_.step),
      reached.functions,
      reached.types,
      reached.channels,
      reached.holes
    )
