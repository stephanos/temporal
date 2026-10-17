package framework.dsl

import scala.quoted.*

@scala.annotation.publicInBinary
private[framework] object ExpectationBlock:
  def validate[T: Type](body: Expr[T])(using Quotes): Expr[BlockPredicate[T]] =
    import quotes.reflect.*
    def fail(tree: Tree, reason: String): Nothing =
      report.errorAndAbort(s"expect: $reason", tree.pos)

    def unwrapped(term: Term): Term = term match
      case Inlined(_, _, value) => unwrapped(value)
      case Typed(value, _)      => unwrapped(value)
      case other                => other

    def function(term: Term): Term = unwrapped(term) match
      case Lambda(_, value) => value
      case other            => fail(other, "write a contextual requirement block directly")

    def sourceFunction(term: Term, visited: Set[Symbol] = Set.empty): Term = unwrapped(term) match
      case lambda @ Lambda(parameters, value) =>
        unwrapped(value) match
          case Apply(Select(receiver, "apply"), arguments)
              if arguments.size == parameters.size && arguments
                .zip(parameters)
                .forall((argument, parameter) => argument.symbol.equals(parameter.symbol)) =>
            unwrapped(receiver) match
              case reference: Ident if !visited(reference.symbol) =>
                reference.symbol.tree match
                  case definition: ValDef if definition.rhs.nonEmpty =>
                    sourceFunction(definition.rhs.get, visited + reference.symbol)
                  case _ => lambda
              case _ => lambda
          case _ => lambda
      case other => other

    val original = sourceFunction(body.asTerm)
    val rootParameters = original match
      case Lambda(parameters, _) => parameters
      case other                 => fail(other, "write a contextual requirement block directly")
    val transition = rootParameters.exists(_.tpt.tpe.typeSymbol.fullName == "framework.dsl.Before")

    def name(term: Term): String = term match
      case Inlined(Some(call: Term), _, _) => name(call)
      case Inlined(_, _, value)            => name(value)
      case Apply(call, _)                  => name(call)
      case TypeApply(call, _)              => name(call)
      case other
          if other.symbol.exists && other.symbol.owner.fullName == "framework.dsl.RequirementFacts" && other.symbol.name == "apply" =>
        "records"
      case other
          if other.symbol.exists && other.symbol.owner.fullName == "framework.dsl.StepFacts" && other.symbol.name == "records" =>
        "records"
      case other
          if other.symbol.exists && other.symbol.owner.fullName == "framework.dsl.ExpectedAlternatives" && other.symbol.name == "apply" =>
        "oneOf"
      case other
          if other.symbol.exists && other.symbol.owner.fullName == "framework.dsl.ExpectedBranch" && other.symbol.name == "apply" =>
        "branch"
      case other if other.symbol.fullName.startsWith("framework.dsl.Syntax") => other.symbol.name
      case _                                                                 => ""

    def arguments(term: Term): List[Term] = term match
      case Inlined(Some(call: Term), _, _) => arguments(call)
      case Inlined(_, _, value)            => arguments(value)
      case Apply(call, values)             => arguments(call) ++ values
      case TypeApply(call, _)              => arguments(call)
      case Select(receiver, _)             => List(receiver)
      case _                               => Nil

    val forbidden = Set(
      "Step",
      "Property",
      "PropertyDecl",
      "PropertyBuilder",
      "Scenario",
      "ScenarioBuilder",
      "ScenarioDecl",
      "Query",
      "QueryDecl",
      "QueryOn",
      "QueryIn",
      "Witness",
      "WitnessBuilder",
      "Progress",
      "Monitor",
      "Assumption"
    )
    def wrongLevel(tpe: TypeRepr): Boolean =
      val widened = tpe.widen.dealias
      widened match
        case _: MethodType | _: PolyType => false
        case _                           =>
          widened.baseClasses.exists(s =>
            s.fullName.startsWith("framework.dsl.") && forbidden(s.name)
          ) ||
          (widened match
            case AppliedType(constructor, types)
                if constructor.typeSymbol.fullName.startsWith("scala.Function") =>
              types.lastOption.exists(wrongLevel)
            case AppliedType(_, types) => types.exists(wrongLevel)
            case _                     => false)

    val visited = scala.collection.mutable.Set.empty[Symbol]
    def calledSymbol(term: Term): Symbol = term match
      case Apply(call, _)       => calledSymbol(call)
      case TypeApply(call, _)   => calledSymbol(call)
      case Inlined(_, _, value) => calledSymbol(value)
      case other                => other.symbol
    val boundedConstructor = calledSymbol('{ UpTo[0](0) }.asTerm)
    val members = scala.collection.mutable.ArrayBuffer.empty[Expr[MemberBinding]]
    val predicates = scala.collection.mutable.ArrayBuffer.empty[Expr[AnyRef]]
    val helperScopes = scala.collection.mutable.Set.empty[Symbol]
    def expectationFunction(tpe: TypeRepr): Boolean =
      tpe.widen.baseClasses
        .filter(s => s.fullName == "scala.Function1" || s.fullName == "scala.Function2")
        .exists: function =>
          tpe.widen.baseType(function) match
            case AppliedType(_, types) =>
              types.lastOption.exists(_ =:= TypeRepr.of[Boolean]) &&
              types
                .dropRight(1)
                .lastOption
                .exists(
                  _.dealias.typeSymbol.fullName == "framework.dsl.Step"
                )
            case _ => false

    def retainPredicate(receiver: Ref): Unit =
      object accessible extends TreeTraverser:
        override def traverseTree(tree: Tree)(owner: Symbol): Unit =
          var scope = tree.symbol
          while scope.exists do
            if helperScopes(scope) then
              fail(receiver, "a forwarded predicate must be accessible outside its source helper")
            scope = scope.owner
          super.traverseTree(tree)(owner)
      accessible.traverseTree(receiver)(Symbol.spliceOwner)
      val _ = predicates += receiver.asExprOf[AnyRef]

    def retainMembers(term: Term): Unit =
      arguments(term).foreach: argument =>
        if argument.tpe.widen.typeSymbol.fullName.startsWith("scala.Function") then
          members += MemberPath
            .memberBinding(argument)
            .getOrElse(
              fail(argument, "a composed fact uses a direct field -> machine selector")
            )

    def pure(tree: Tree): Unit =
      object inspect extends TreeTraverser:
        override def traverseTree(tree: Tree)(owner: Symbol): Unit =
          tree match
            case value: Term
                if !transition && (name(
                  value
                ) == "before" || value.tpe.widen.dealias.typeSymbol.fullName == "framework.dsl.Before") =>
              fail(value, "before-state evidence is available only in across blocks")
            case value: Term if name(value) == "records" =>
              retainMembers(value)
              arguments(value).foreach(pure)
            case reference: Ref
                if wrongLevel(reference.tpe) &&
                  !reference.tpe.widen.dealias.baseClasses
                    .exists(_.fullName == "framework.dsl.Step") =>
              fail(reference, "a predicate cannot read a path claim or temporal declaration")
            case _: Assign | _: While | _: Return | _: Try =>
              fail(tree, "mutation and control statements are not predicate expressions")
            case value: ValDef if value.symbol.flags.is(Flags.Mutable) =>
              fail(value, "mutable helpers are not predicate expressions")
            case call: Apply =>
              call match
                case Apply(Select(receiver: Ref, "apply"), _)
                    if expectationFunction(receiver.tpe) =>
                  retainPredicate(receiver)
                case _ => ()
              val method = call.fun.symbol
              val operation = method.name
              if operation == "throw" || operation == "implies" then
                fail(call, s"$operation is not expectation vocabulary")
              if wrongLevel(call.tpe) && name(call) != "after" then
                fail(
                  call,
                  "a predicate cannot construct a path claim, temporal declaration or step"
                )
              if call.tpe.widen =:= TypeRepr.of[Unit] then
                fail(call, "an arbitrary Unit call is not a predicate expression")
              if method.exists && !visited(method) &&
                !method.equals(boundedConstructor) &&
                !method.fullName.startsWith("scala.") &&
                !method.fullName.startsWith("java.") &&
                !method.fullName.startsWith("framework.dsl.Syntax")
              then
                visited += method
                method.tree match
                  case definition: DefDef =>
                    helperScopes += method
                    definition.rhs.foreach(pure)
                    helperScopes -= method
                  case _ => ()
              super.traverseTree(tree)(owner)
            case definition: DefDef => definition.rhs.foreach(pure)
            case _                  => super.traverseTree(tree)(owner)
      inspect.traverseTree(tree)(Symbol.spliceOwner)

    def unit(term: Term): Boolean = unwrapped(term) match
      case Literal(UnitConstant()) => true
      case _                       => false

    def statements(term: Term): List[Statement] =
      (if Set("check", "records", "oneOf", "branch")(name(term)) then term
       else unwrapped(term)) match
        case Block(values, result) => values ++ (if unit(result) then Nil else List(result))
        case value if unit(value)  => Nil
        case other                 => List(other)

    def contextualArgument(term: Term): Term =
      arguments(term)
        .find(value => value.tpe.widen.typeSymbol.fullName.startsWith("scala.ContextFunction"))
        .getOrElse(fail(term, "a structural block requires its contextual body"))

    def requirements(term: Term): Unit =
      var required = false
      statements(term).foreach:
        case value: ValDef
            if !value.symbol.flags.is(Flags.Mutable) && value.tpt.tpe.widen <:< TypeRepr
              .of[Boolean] =>
          value.rhs.foreach(pure)
        case conditional: If =>
          pure(conditional.cond)
          requirements(conditional.thenp)
          if !unit(conditional.elsep) then requirements(conditional.elsep)
          required = true
        case value: Term if Set("check", "records")(name(value)) =>
          if name(value) == "records" then retainMembers(value)
          arguments(value).foreach(pure)
          required = true
        case value: Term if name(value) == "oneOf" =>
          val branches = statements(function(contextualArgument(value)))
          if branches.size < 2 then fail(value, "oneOf requires at least two branches")
          branches.foreach:
            case alternative: Term if name(alternative) == "branch" =>
              requirements(function(contextualArgument(alternative)))
            case other => fail(other, "oneOf contains only explicit branch blocks")
          required = true
        case other =>
          fail(
            other,
            "use check, records, if or oneOf; discarded expressions and declarations are not requirements"
          )
      if !required then
        fail(term, "a requirement block must not be empty or contain only helper vals")

    requirements(function(original))
    val bindings = Expr.ofSeq(members.toSeq)
    val forwarded = Expr.ofSeq(predicates.toSeq)
    // A context parameter the author left unnamed is named after its role, `before` or `after`,
    // which is what `before` and `after` read in the block.
    def sourceName(parameter: ValDef): String =
      if !parameter.symbol.flags.is(Flags.Synthetic) && !parameter.name.contains("$") then
        parameter.name
      else if parameter.tpt.tpe.typeSymbol.fullName == "framework.dsl.Before" then "before"
      else "after"
    val names = original match
      case Lambda(parameters, _) =>
        Expr.ofSeq(parameters.map(parameter => Expr(sourceName(parameter))))
      case other => fail(other, "write a contextual requirement block directly")
    '{ PredicateBindings.block[T]($body, $bindings.toVector, $names.toVector, $forwarded.toVector) }
