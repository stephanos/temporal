package framework.dsl

import scala.quoted.*

// The compile-time half of a transition block: it checks the block's leaves and marks each effect
// call so the native runtime (NativeRuntime) sees that the block fired. A leaf is a call of a method
// of an `effects` object, the framework's `reject` or `()`, reached through `if`, `match`, blocks
// that open with vals and type ascriptions; Scala's own control flow then gives first match. An
// assignment to a var of a Machine anywhere in the block is refused: the effect assigns.
//
// Symbols are recognised by their names and owners, never by reference to the words that
// declare them, so that this core file names no sugar.
private[framework] object TransitionMacros:
  private val leafRule = "a transition ends in an effect call, a rejection or ()"
  private val assignment = "a transition says when; write the state change in effects"

  // A block without inputs, as the function its rule runs.
  inline def block(inline body: Unit): () => Unit = ${ blockImpl('body) }

  // A block whose input is its parameter, as the function its rule runs on the input.
  inline def input[I](inline body: I => Unit): I => Unit = ${ inputImpl[I]('body) }

  private def blockImpl(body: Expr[Unit])(using Quotes): Expr[() => Unit] =
    '{ new Programs.NativeBlock(() => ${ leaves(body) }, ${ choices(body) }) }

  private[framework] def inputImpl[I: Type](body: Expr[I => Unit])(using Quotes): Expr[I => Unit] =
    '{
      new Programs.NativeInputBlock[I](
        (in: I) => ${ leaves(Expr.betaReduce('{ $body(in) })) },
        ${ choices(body) }
      )
    }

  private def choices(body: Expr[?])(using Quotes): Expr[Option[Vector[Choice]]] =
    import quotes.reflect.*
    val tokens = scala.collection.mutable.ArrayBuffer.empty[Expr[Choice]]
    var complete = true
    def unwrapped(term: Term): Term = term match
      case Inlined(_, _, value) => unwrapped(value)
      case Typed(value, _)      => unwrapped(value)
      case other                => other
    def invocation(term: Term): (Symbol, Option[Term], List[Term]) = term match
      case Apply(function, arguments) =>
        val (symbol, receiver, preceding) = invocation(function)
        (symbol, receiver, preceding ++ arguments)
      case TypeApply(function, _)          => invocation(function)
      case selection @ Select(receiver, _) => (selection.symbol, Some(receiver), Nil)
      case other                           => (other.symbol, None, Nil)
    def stable(term: Term): Boolean = unwrapped(term) match
      case _: This        => true
      case reference: Ref =>
        !reference.symbol.flags.is(Flags.Mutable) &&
        (!reference.symbol.isDefDef || reference.symbol.flags.is(Flags.FieldAccessor) ||
          reference.symbol.flags.is(Flags.Module)) &&
        (reference match
          case Select(receiver, _) => stable(receiver)
          case _                   => true)
      case _ => false
    def inspect(tree: Tree, active: Set[Symbol]): Unit =
      new TreeTraverser:
        override def traverseTree(current: Tree)(owner: Symbol): Unit = current match
          case call: Apply =>
            val (method, receiver, arguments) = invocation(call)
            if method.owner.fullName == "framework.dsl.Choice" && method.name == "->" then
              receiver match
                case Some(token) if stable(token) => tokens += token.asExprOf[Choice]: Unit
                case _                            => complete = false
              arguments.foreach(inspect(_, active))
            else if method.owner.name == "effects$" then
              if active(method) then complete = false
              else
                scala.util.Try(method.tree).toOption match
                  case Some(definition: DefDef) if definition.rhs.nonEmpty && receiver.nonEmpty =>
                    val parameters = definition.termParamss.flatMap(_.params).map(_.symbol)
                    if parameters.size != arguments.size then complete = false
                    else
                      val substitutions = parameters.zip(arguments).toMap
                      val rewritten = new TreeMap:
                        override def transformTerm(term: Term)(owner: Symbol): Term = term match
                          case self: This if self.symbol.equals(method.owner) => receiver.get
                          case reference: Ref if substitutions.contains(reference.symbol) =>
                            substitutions(reference.symbol)
                          case reference: Ident if reference.symbol.owner.equals(method.owner) =>
                            Select(receiver.get, reference.symbol)
                          case _ => super.transformTerm(term)(owner)
                      inspect(
                        rewritten.transformTerm(definition.rhs.get)(Symbol.spliceOwner),
                        active + method
                      )
                  case _ => complete = false
            else
              if call.tpe <:< TypeRepr.of[Unit] && method.exists &&
                !method.fullName.startsWith("scala.") && !method.fullName.startsWith("java.") &&
                !method.fullName.startsWith("framework.dsl.") &&
                !method.name.endsWith("_=") &&
                !(method.owner.fullName == "<special-ops>" && method.name == "throw")
              then complete = false
              super.traverseTree(current)(owner)
          case _ => super.traverseTree(current)(owner)
      .traverseTree(tree)(Symbol.spliceOwner)
    inspect(body.asTerm, Set.empty)
    if complete then '{ Some(Vector[Choice](${ Varargs(tokens.toSeq) }*).distinct) } else '{ None }

  private[framework] def leaves(body: Expr[Unit])(using Quotes): Expr[Unit] =
    import quotes.reflect.*
    val machine = TypeRepr.of[Machine[?, ?]].typeSymbol

    def isState(s: Symbol): Boolean =
      s.flags.is(Flags.Mutable) && s.owner.isClassDef &&
        s.owner.typeRef.baseClasses.contains(machine)

    new TreeTraverser:
      override def traverseTree(tree: Tree)(owner: Symbol): Unit = tree match
        case Assign(lhs, _) if isState(lhs.symbol) => report.error(assignment, tree.pos)
        case _                                     => super.traverseTree(tree)(owner)
    .traverseTree(body.asTerm)(Symbol.spliceOwner)

    def callee(t: Term): Symbol = t match
      case Apply(fn, _)     => callee(fn)
      case TypeApply(fn, _) => callee(fn)
      case other            => other.symbol

    def isEffect(t: Term): Boolean =
      val owner = callee(t).owner
      owner.flags.is(Flags.Module) && owner.name == "effects$"

    def isRejection(t: Term): Boolean =
      val s = callee(t)
      s.name == "reject" && s.owner.name.startsWith("Syntax$package") &&
      s.owner.owner.fullName == "framework.dsl"

    def leaf(t: Term): Term = t match
      case Inlined(call, bindings, inner) => Inlined(call, bindings, leaf(inner))
      case Typed(inner, _)                => leaf(inner)
      case Block(stats, expr) if stats.forall { case _: ValDef => true; case _ => false } =>
        Block(stats, leaf(expr))
      case If(condition, thenp, elsep) => If(condition, leaf(thenp), leaf(elsep))
      case Match(scrutinee, cases)     =>
        Match(scrutinee, cases.map(c => CaseDef(c.pattern, c.guard, leaf(c.rhs))))
      case Literal(UnitConstant())                                         => t
      case call: Apply if isEffect(call) && call.tpe <:< TypeRepr.of[Unit] =>
        Block(List('{ Vars.fired() }.asTerm), call)
      case call: Apply if isRejection(call)      => call
      case Assign(lhs, _) if isState(lhs.symbol) => t
      case other                                 =>
        val shown = other.pos.sourceCode.getOrElse(other.show(using Printer.TreeShortCode))
        report.error(s"$leafRule: $shown is none of these", other.pos)
        other

    leaf(body.asTerm).asExprOf[Unit]
