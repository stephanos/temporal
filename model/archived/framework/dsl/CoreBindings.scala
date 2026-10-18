package framework.dsl

import scala.collection.mutable
import io.temporal.server.api.umpire.v1 as ir

private[framework] object CoreBindings:
  final case class Operation(
      name: String,
      before: Vector[ActionDecl],
      operands: Vector[ActionDecl],
      unsupported: Boolean,
      source: Option[MachineDeclaration[?, ?, ?]] = None,
      sourceTable: Vector[(ActionDecl, Bound[?, ?, ?])] = Vector.empty,
      rawOperands: Vector[Rebinding[?, ?, ?]] = Vector.empty,
      sourceEvidence: Option[DerivationBindings.Snapshot] = None,
      operandEvidence: Vector[DerivationBindings.OperandEvidence] = Vector.empty,
      nativeEvidence: Option[DerivationBindings.Snapshot] = None,
      site: Option[DerivationBindings.OperationSite] = None,
      nativeSourceEvidence: Vector[DerivationBindings.Snapshot] = Vector.empty
  )
  final case class Contribution(
      steps: Vector[ir.StepBinding],
      functions: Vector[ir.Function],
      types: Vector[ir.Type],
      channels: Vector[String],
      holes: Vector[String]
  )
  final case class Dependencies(
      functions: Vector[ir.Function],
      types: Vector[ir.Type],
      channels: Vector[String],
      holes: Vector[String]
  )
  final case class CurriedCall(
      call: ir.Call,
      function: ir.Function,
      groups: Vector[Int],
      owner: Option[String] = None
  )

  def operation[S, O, F](
      name: String,
      before: Vector[(ActionDecl, Bound[S, O, F])],
      operands: Vector[Rebinding[S, O, F]],
      source: Option[MachineDeclaration[?, ?, ?]] = None,
      sourceEvidence: Option[DerivationBindings.Snapshot] = None,
      operandEvidence: Vector[DerivationBindings.OperandEvidence] = Vector.empty,
      nativeEvidence: Option[DerivationBindings.Snapshot] = None,
      site: Option[DerivationBindings.OperationSite] = None,
      nativeSourceEvidence: Vector[DerivationBindings.Snapshot] = Vector.empty
  ): Operation =
    val actions = operands.flatMap:
      case StepBinding(decl, _) => Vector(decl)
      case RuleGroup(rules)     => rules.map(_.decl).distinct
    Operation(
      name,
      before.map(_._1),
      actions,
      operands.exists(_.isInstanceOf[RuleGroup[?, ?, ?]]) ||
        before.exists { (_, bound) =>
          bound match
            case Bound.Function(_, _) => false
            case _                    => true
        },
      source,
      before,
      operands,
      sourceEvidence,
      operandEvidence,
      nativeEvidence,
      site,
      nativeSourceEvidence
    )

  private def walk(value: Any)(visit: Any => Unit): Unit =
    visit(value)
    value match
      case values: Seq[?]   => values.foreach(walk(_)(visit))
      case product: Product => product.productIterator.foreach(walk(_)(visit))
      case _                => ()

  def calls(value: Any): Vector[ir.Call] =
    val result = Vector.newBuilder[ir.Call]
    walk(value) {
      case call: ir.Call => result += call: Unit
      case _             => ()
    }
    result.result()

  def reachedProofs(
      roots: Seq[String],
      functions: Seq[ir.Function],
      proofs: Vector[CurriedCall]
  ): Vector[CurriedCall] =
    val reached = mutable.Set.empty[String]
    def visit(name: String): Unit =
      if reached.add(name) then
        functions.find(_.name == name).foreach(f => calls(f).foreach(c => visit(c.function)))
    roots.foreach(visit)
    proofs.filter(_.owner.exists(reached))

  def renameVariables(value: ir.Expr, variables: Map[String, String]): ir.Expr =
    rename(value, "", "", variables)

  private def rename(
      value: ir.Expr,
      source: String,
      target: String,
      variables: Map[String, String] = Map.empty
  ): ir.Expr =
    def next(value: ir.Expr): ir.Expr = rename(value, source, target, variables)
    def within(value: ir.Expr, bound: Iterable[String]): ir.Expr =
      rename(value, source, target, variables -- bound)
    def bindings(pattern: ir.Pattern): Set[String] = pattern.kind match
      case ir.Pattern.Kind.Bind(bound) =>
        bound.pattern.fold(Set.empty[String])(bindings) + bound.name
      case ir.Pattern.Kind.Case(c)         => c.fields.flatMap(bindings).toSet
      case ir.Pattern.Kind.Alternatives(a) => a.patterns.flatMap(bindings).toSet
      case _                               => Set.empty
    import ir.Expr.Kind as E
    value.copy(kind = value.kind match
      case E.Var(name)  => E.Var(variables.getOrElse(name, name))
      case E.Call(call) =>
        E.Call(
          call.copy(
            function = if call.function == source then target else call.function,
            args = call.args.map(next)
          )
        )
      case E.Field(field)         => E.Field(field.copy(base = field.base.map(next)))
      case E.Construct(construct) => E.Construct(construct.copy(args = construct.args.map(next)))
      case E.Copy(copy)           =>
        E.Copy(
          copy.copy(
            base = copy.base.map(next),
            updates = copy.updates.map(u => u.copy(value = u.value.map(next)))
          )
        )
      case E.Unary(unary)   => E.Unary(unary.copy(operand = unary.operand.map(next)))
      case E.Binary(binary) =>
        E.Binary(binary.copy(left = binary.left.map(next), right = binary.right.map(next)))
      case E.If(branch) =>
        E.If(
          branch.copy(
            condition = branch.condition.map(next),
            `then` = branch.`then`.map(next),
            `else` = branch.`else`.map(next)
          )
        )
      case E.Match(matched) =>
        E.Match(
          matched.copy(
            scrutinee = matched.scrutinee.map(next),
            cases = matched.cases.map { c =>
              val bound = c.pattern.fold(Set.empty[String])(bindings)
              c.copy(guard = c.guard.map(within(_, bound)), body = c.body.map(within(_, bound)))
            }
          )
        )
      case E.Let(let) =>
        E.Let(let.copy(value = let.value.map(next), body = let.body.map(within(_, List(let.name)))))
      case E.List(list)     => E.List(list.copy(items = list.items.map(next)))
      case E.Lambda(lambda) =>
        E.Lambda(lambda.copy(body = lambda.body.map(within(_, lambda.params.map(_.name)))))
      case E.Inbox(inbox) =>
        E.Inbox(inbox.copy(contents = inbox.contents.map(next), message = inbox.message.map(next)))
      case other => other)

  def collect[S, O, F](machine: MachineDeclaration[S, O, F]): Contribution =
    val context = s"machine ${machine.name} core bindings"
    def fail(message: String): Nothing = throw IllegalArgumentException(s"$context: $message")
    def identity(decl: ActionDecl): String = decl.sourceIdentity
      .getOrElse(fail(s"action ${decl.name} has no retained source identity"))
    machine.coreOperations.foreach { operation =>
      val before = operation.before.map(identity)
      val operands = operation.operands.map(identity)
      if operands.distinct.size != operands.size then
        fail(s"${operation.name} binds an action twice")
      operands.foreach { id =>
        if operation.name == "rebind" && !before.contains(id) then
          fail(s"rebind action $id which the source does not bind")
        if operation.name == "extend" && before.contains(id) then
          fail(s"extend action $id which the source binds already")
      }
      if operation.unsupported then
        fail(s"${operation.name} rule/effect contributions are unsupported")
    }
    collectEntries(DerivationBindings.acquiredTable(machine, context), context)

  def collectEntries(
      table: Vector[(ActionDecl, Bound[?, ?, ?])],
      context: String
  ): Contribution =
    def fail(message: String): Nothing = throw IllegalArgumentException(s"$context: $message")
    def identity(decl: ActionDecl): String = decl.sourceIdentity
      .getOrElse(fail(s"action ${decl.name} has no retained source identity"))
    val functions = mutable.LinkedHashMap.empty[String, ir.Function]
    val types = mutable.LinkedHashMap.empty[String, ir.Type]
    val roots = mutable.Map.empty[String, BindingOrigin]
    val steps = table.map { (decl, bound) =>
      val capture = bound match
        case Bound.Function(f, Some(captured)) if f.eq(captured.callable) => captured
        case Bound.Function(_, Some(_))                                   =>
          fail(s"action ${identity(decl)} has stale callable metadata")
        case Bound.Function(_, None) =>
          fail(s"action ${identity(decl)} has no captured function metadata")
        case Bound.Ruled(_) =>
          fail(s"action ${identity(decl)} has unsupported rules, not a complete step function")
        case Bound.Disabled() =>
          fail(s"action ${identity(decl)} has an unsupported disabled contribution")
      val origin = capture.origin.getOrElse(
        fail(s"action ${identity(decl)} has no compiler-owned binding origin")
      )
      if !origin.action.contains(identity(decl)) then
        fail(s"action ${identity(decl)} has a stale binding origin")
      if !origin.declaration.eq(decl) then
        fail(s"action ${identity(decl)} has a stale binding declaration")
      if origin.owner.isEmpty || origin.machine.isEmpty then
        fail(s"action ${identity(decl)} has no retained binding owner")
      // Binding identity and typed inputs need no example/catalog evaluation.
      val action = Signature.action(decl.boundTo(decl.copy(examples = Nil)))
      val arity = if decl.nativeInputs.nonEmpty then 2 else 1
      if capture.function.params.size != arity then
        fail(s"action ${action.id} has inconsistent captured arity")
      val root = if origin.named then capture.root
      else s"${origin.machine}.${action.name}"
      val historicalDuplicate = !origin.named && roots
        .get(root)
        .exists(previous =>
          previous.owner == origin.owner && previous.machine == origin.machine && previous.action == origin.action && previous.declaration
            .eq(origin.declaration) && !previous.named
        )
      capture.model.functions.foreach { function =>
        val rebased = if origin.named then function
        else
          function.copy(
            name = if function.name == capture.root then root else function.name,
            body = function.body.map(rename(_, capture.root, root)),
            requires = function.requires.map(rename(_, capture.root, root))
          )
        if functions.get(rebased.name).exists(prior => !prior.equals(rebased)) &&
          !(historicalDuplicate && function.name == capture.root)
        then fail(s"function ${rebased.name} has conflicting retained definitions")
        functions(rebased.name) = rebased
      }
      roots(root) = origin
      capture.model.types.foreach { value =>
        if types.get(value.name).exists(prior => !prior.clearPosition.equals(value.clearPosition))
        then fail(s"type ${value.name} has conflicting retained definitions")
        types.getOrElseUpdate(value.name, value): Unit
      }
      ir.StepBinding(action.id, root, origin.position)
    }
    val closure = dependencies(
      Nil,
      steps.map(_.function),
      functions.values.toVector,
      types.values.toVector,
      context
    )
    Contribution(steps, closure.functions, closure.types, closure.channels, closure.holes)

  def dependencies(
      expressions: Seq[ir.Expr],
      roots: Seq[String],
      retainedFunctions: Seq[ir.Function],
      retainedTypes: Seq[ir.Type],
      context: String,
      curriedCalls: Seq[CurriedCall] = Nil
  ): Dependencies =
    def fail(message: String): Nothing = throw IllegalArgumentException(s"$context: $message")
    val functions = mutable.LinkedHashMap.empty[String, ir.Function]
    val types = mutable.LinkedHashMap.empty[String, ir.Type]
    retainedFunctions.foreach { value =>
      if value.name.isEmpty then fail("function has no retained name")
      if functions.get(value.name).exists(prior => !prior.equals(value)) then
        fail(s"function ${value.name} has conflicting retained definitions")
      functions.getOrElseUpdate(value.name, value): Unit
    }
    retainedTypes.foreach { value =>
      if value.name.isEmpty then fail("type has no retained name")
      if types.get(value.name).exists(prior => !prior.clearPosition.equals(value.clearPosition))
      then fail(s"type ${value.name} has conflicting retained definitions")
      types.getOrElseUpdate(value.name, value): Unit
    }
    curriedCalls.foreach { proof =>
      if !functions.get(proof.function.name).exists(_.equals(proof.function)) ||
        proof.call.function != proof.function.name || proof.groups.size < 2 ||
        proof.groups
          .exists(_ < 0) || proof.groups.map(_.toLong).sum != proof.function.params.size.toLong ||
        proof.call.args.size >= proof.function.params.size ||
        proof.groups.last != proof.call.args.size
      then fail("stale or malformed retained curried-call evidence")
    }
    val consumedCurriedCalls = mutable.Set.empty[CurriedCall]
    val reachableFunctions = mutable.Set.empty[String]
    val reachableTypes = mutable.Set.empty[String]
    val channels = mutable.Set.empty[String]
    val holes = mutable.Set.empty[String]
    def external(id: String, target: mutable.Set[String], kind: String): Unit =
      if id.isEmpty then fail(s"empty external $kind reference")
      target.add(id): Unit
    def tpe(name: String): Unit =
      if name != "scala.Nothing" && name != "framework.dsl.Step" && reachableTypes.add(name) then
        val value = types.getOrElse(name, fail(s"type $name has no retained definition"))
        if value.shape.isEmpty then fail(s"type $name has no retained shape")
        scan(value)
    def function(name: String): Unit =
      if reachableFunctions.add(name) then
        val value = functions.getOrElse(name, fail(s"callee/root $name has no retained definition"))
        if value.body.isEmpty then fail(s"function $name has no retained body")
        parameters(value.params)
        validate(value.getBody, value.params.map(_.name).toSet)
        value.requires.foreach(validate(_, value.params.map(_.name).toSet))
        scan(value, Some(name))
    def parameters(values: Seq[ir.Param]): Unit =
      if values.exists(_.name.isEmpty) || values.map(_.name).distinct.size != values.size then
        fail("missing or duplicate retained parameter name")
    def required[A](value: Option[A], kind: String): A =
      value.getOrElse(fail(s"missing retained $kind"))
    def pattern(value: ir.Pattern): Set[String] = value.kind match
      case ir.Pattern.Kind.Bind(bound) =>
        if bound.name.isEmpty then fail("missing retained pattern binding")
        pattern(required(bound.pattern, "bound pattern")) + bound.name
      case ir.Pattern.Kind.Case(c)         => c.fields.flatMap(pattern).toSet
      case ir.Pattern.Kind.Alternatives(a) => a.patterns.flatMap(pattern).toSet
      case kind if kind.isEmpty            => fail("missing retained pattern")
      case _                               => Set.empty
    def validate(value: ir.Expr, scope: Set[String]): Unit =
      def next(value: ir.Expr): Unit = validate(value, scope)
      import ir.Expr.Kind as E
      value.kind match
        case E.Var(name) =>
          if name.isEmpty || !scope(name) then fail(s"variable $name has no retained binding")
        case E.Field(field) =>
          if field.field.isEmpty then fail("missing retained field name")
          next(required(field.base, "field base"))
        case E.Call(call)           => call.args.foreach(next)
        case E.Construct(construct) => construct.args.foreach(next)
        case E.Copy(copy)           =>
          next(required(copy.base, "copy base"))
          copy.updates.foreach { update =>
            if update.name.isEmpty then fail("missing retained update name")
            next(required(update.value, "update value"))
          }
        case E.Unary(unary) =>
          if unary.op.isUnrecognized || unary.op.value == 0 then
            fail("missing retained unary operator")
          next(required(unary.operand, "unary operand"))
        case E.Binary(binary) =>
          if binary.op.isUnrecognized || binary.op.value == 0 then
            fail("missing retained binary operator")
          next(required(binary.left, "binary left operand"))
          next(required(binary.right, "binary right operand"))
        case E.If(branch) =>
          next(required(branch.condition, "condition"))
          next(required(branch.`then`, "then branch"))
          next(required(branch.`else`, "else branch"))
        case E.Match(matched) =>
          next(required(matched.scrutinee, "match scrutinee"))
          if matched.cases.isEmpty then fail("missing retained match cases")
          matched.cases.foreach { c =>
            val bound = scope ++ pattern(required(c.pattern, "match pattern"))
            c.guard.foreach(validate(_, bound))
            validate(required(c.body, "match body"), bound)
          }
        case E.Let(let) =>
          if let.name.isEmpty then fail("missing retained let name")
          next(required(let.value, "let value"))
          validate(required(let.body, "let body"), scope + let.name)
        case E.List(list)     => list.items.foreach(next)
        case E.Lambda(lambda) =>
          parameters(lambda.params)
          validate(required(lambda.body, "lambda body"), scope ++ lambda.params.map(_.name))
        case E.Inbox(inbox) =>
          next(required(inbox.contents, "Inbox contents"))
          if inbox.op.value == 1 then next(required(inbox.message, "Inbox message"))
        case kind if kind.isEmpty => fail("missing retained expression")
        case _                    => ()
    def scan(value: Any, owner: Option[String] = None): Unit = walk(value) {
      case call: ir.Call =>
        function(call.function)
        if functions(call.function).params.size != call.args.size then
          curriedCalls.find(proof => proof.call.equals(call) && proof.owner == owner) match
            case Some(proof) => consumedCurriedCalls.add(proof): Unit
            case None => fail(s"callee ${call.function} has inconsistent retained argument arity")
      case ref: ir.TypeRef =>
        ref.ref match
          case ir.TypeRef.Ref.Named(name) => tpe(name)
          case ir.TypeRef.Ref.Channel(id) => external(id, channels, "channel")
          case ref if ref.isEmpty         => fail("missing retained type reference")
          case _                          => ()
      case construct: ir.Construct                      => tpe(construct.`type`)
      case literal: ir.EnumValue                        => tpe(literal.`type`)
      case literal: ir.RecordValue                      => tpe(literal.`type`)
      case pattern: ir.CasePattern                      => tpe(pattern.`type`)
      case value: ir.Enum if value.cases.isEmpty        => fail("missing retained enum cases")
      case value: ir.Case if value.name.isEmpty         => fail("missing retained enum case name")
      case value: ir.Value if value.kind.isEmpty        => fail("missing retained literal value")
      case range: ir.IntRange if range.low > range.high => fail("invalid retained integer range")
      case param: ir.Param if param.`type`.isEmpty      =>
        fail(s"parameter ${param.name} has no retained type")
      case field: ir.Field if field.name.isEmpty || field.`type`.isEmpty =>
        fail(s"field ${field.name} has no retained type")
      case expr: ir.Expr =>
        expr.kind match
          case ir.Expr.Kind.Hole(id) => external(id, holes, "hole")
          case kind if kind.isEmpty  => fail("missing retained expression")
          case _                     => ()
      case inbox: ir.Inbox => external(inbox.channel, channels, "channel")
      case _               => ()
    }
    expressions.foreach { value => validate(value, Set.empty); scan(value) }
    roots.foreach(function)
    if curriedCalls.exists(proof => !consumedCurriedCalls(proof)) then
      fail("retained curried-call evidence does not name a reachable exact call")
    Dependencies(
      reachableFunctions.toVector.sorted.map(functions),
      reachableTypes.toVector.sorted.map(types),
      channels.toVector.sorted,
      holes.toVector.sorted
    )
