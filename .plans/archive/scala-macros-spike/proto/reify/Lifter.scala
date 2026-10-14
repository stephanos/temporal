package framework.reify

import scala.collection.mutable
import scala.quoted.*
import Json.{obj, Arr, Bool, Num, Str}

// The body lifter, hosted in a macro. It is written against `quotes.reflect`, the API the
// post-compile lifter (model/framework/export/Expressions.scala) already uses through the TASTy
// inspector; the differences are only where trees come from (`Symbol.tree`, which needs
// `-Yretain-trees` for defs outside the macro argument) and how a refusal is reported
// (`report.errorAndAbort` at the tree's position, so it is a compile error at the author's line).
final class Lifter(using val q: Quotes):
  import q.reflect.*

  private val functions = mutable.LinkedHashMap.empty[String, Json]
  private val lifting = mutable.Set.empty[String]
  private val renamed = mutable.Map.empty[Symbol, String]
  // A sugar call's parameters, bound to the call's arguments while its definition is lifted.
  private val bound = mutable.Map.empty[Symbol, Term]
  // The position the nodes of an expanded sugar body take: the call's (fn-141 R4).
  private var placedAt: Option[Json] = None

  private val stepType = "framework.dsl.Step"
  private val binaryOps = Map(
    "==" -> "OP_EQ", "!=" -> "OP_NE", "&&" -> "OP_AND", "||" -> "OP_OR", "<" -> "OP_LT",
    "<=" -> "OP_LE", ">" -> "OP_GT", ">=" -> "OP_GE", "+" -> "OP_ADD", "-" -> "OP_SUB",
    "++" -> "OP_CONCAT"
  )

  // ### Entry points

  def rule(actionExpr: Expr[Any], casesExpr: Expr[?]): (String, String) =
    val cases = casesExpr.asTerm
    val act = unwrap(actionExpr.asTerm)
    val actionName = act.symbol.name
    if !(act.tpe.widen.dealias.typeArgs.headOption.exists(_ =:= TypeRepr.of[EmptyTuple])) then
      fail(act, "spike: only actions without inputs are reified")
    val machineObj = Iterator
      .iterate(Symbol.spliceOwner)(_.maybeOwner)
      .takeWhile(!_.isNoSymbol)
      .find(s => s.flags.is(Flags.Module) && s.moduleClass.typeRef.baseClasses.exists(_.fullName == "framework.dsl.Machine"))
      .getOrElse(fail(act, "a rules section sits in a machine object"))
    val machineName =
      val n = machineObj.name.stripSuffix("$")
      n.take(1).toLowerCase + n.drop(1)
    val stateType =
      machineObj.moduleClass.typeRef.baseType(Symbol.requiredClass("framework.dsl.Machine")).typeArgs.head
    val stateName = "s"
    val at = Position.ofMacroExpansion
    // The cases: `(firing) ?=> where(g) ~> effect; ...`, each `~>` one arm of the step function.
    val arms = mutable.ListBuffer.empty[(Term, Term, Term)] // (arm tree, guard, effect)
    def collect(t: Term): Unit = unwrap(t) match
      case Block(List(DefDef(_, _, _, Some(body))), _: Closure) => collect(body)
      case Block(stats, e) =>
        stats.foreach { case s: Term => collect(s); case _ => () }
        collect(e)
      case arm @ Apply(Apply(Select(c, "~>"), List(effect)), _) => arms += ((arm, caseGuard(c), effect))
      case other => fail(other, s"a rule block holds cases `where(g) ~> effect`, not ${other.show}")
    collect(cases)
    def arm(armTree: Term, guard: Term, effect: Term, rest: Json): Json =
      val g = guard match
        case Literal(BooleanConstant(true)) => lit("bool", Bool(true), armTree)
        case _ =>
          lambda(guard) match
            case Some((List(p), body)) =>
              renamed(p.symbol) = stateName
              lift(body)
            case _ => fail(guard, s"a rule's condition is a function of the state, not ${guard.show}")
      val e = lambda(effect) match
        case Some((List(p), body)) =>
          renamed(p.symbol) = stateName
          unwrap(body) match
            case Apply(fn, _) if fn.symbol.isDefDef => lift(body)
            case other => fail(other, s"the effect of a rule of $machineName is a call of a def of `effects`, not ${other.show}")
        case _ => fail(effect, s"the effect of a rule of $machineName is a function, not ${effect.show}")
      expr(armTree)("if", obj("condition" -> g, "then" -> e, "else" -> rest))
    // Fold the arms right to left into one `if` chain ending in no step.
    val none = obj("position" -> position(at), "list" -> obj())
    val body = arms.toList.foldRight(none) { case ((t, g, e), rest) => arm(t, g, e, rest) }
    val name = s"$machineName.rules.$actionName"
    functions(name) = obj(
      "name" -> Str(name),
      "position" -> position(at),
      "params" -> Arr(List(obj("name" -> Str(stateName), "type" -> typeRef(stateType, act)))),
      "body" -> body
    )
    (name, render)

  def function(fExpr: Expr[Any]): (String, String) =
    unwrap(fExpr.asTerm) match
      case t if lambda(t).nonEmpty =>
        val (params, body) = lambda(t).get
        // A def passed by reference arrives as its eta-expansion, `x => d(x)`: lift `d` itself.
        unwrap(body) match
          case Apply(fn, args) if fn.symbol.isDefDef && args.map(a => unwrap(a).symbol) == params.map(_.symbol) =>
            val name = callee(fn.symbol, t)
            (name, render)
          case _ =>
            functions("<lambda>") = functionJson("<lambda>", params, body, t)
            ("<lambda>", render)
      case other => fail(other, s"reify a def or a lambda, not ${other.show}")

  // A def by symbol, for a host that is not a macro (the compiler-plugin probe).
  def defined(sym: Symbol, at: Tree): String =
    callee(sym, at)
    render

  private def render: String = Arr(functions.values.toList).render

  // ### Plumbing

  private def fail(t: Tree, message: String): Nothing = report.errorAndAbort(message, t.pos)

  private def unwrap(t: Term): Term = t match
    case Inlined(_, Nil, e) => unwrap(e)
    case Typed(e, _)        => unwrap(e)
    case Block(Nil, e)      => unwrap(e)
    case NamedArg(_, e)     => unwrap(e)
    case _                  => t

  private def lambda(t: Term): Option[(List[ValDef], Term)] = unwrap(t) match
    case Block(List(DefDef("$anonfun", List(TermParamClause(params)), _, Some(body))), _: Closure) =>
      Some((params, body))
    case _ => None

  private def caseGuard(c: Term): Term = unwrap(c) match
    case Apply(Apply(TypeApply(w, _), _), List(g)) if w.symbol.name == "where" => g
    case Apply(TypeApply(a, _), _) if a.symbol.name == "always"                => Literal(BooleanConstant(true))
    case other => fail(other, s"spike: a case is `where(g)` or `always`, not ${other.show}")

  // The IR file path: the spike takes it from the `model/` segment on; the real exporter would
  // relativize against the build's -sourceroot, as TASTy does.
  private def file(p: Position): String =
    val path = p.sourceFile.path
    val i = path.lastIndexOf("/model/")
    if i >= 0 then path.substring(i + 1) else path

  private def position(p: Position): Json = obj("file" -> Str(file(p)), "line" -> Num((p.startLine + 1).toString))
  private def pos(t: Tree): Json = placedAt.getOrElse(position(t.pos))
  private def expr(at: Tree)(kind: String, value: Json): Json = obj("position" -> pos(at), kind -> value)
  private def lit(kind: String, value: Json, at: Tree): Json = expr(at)("literal", obj(kind -> value))
  private def list(items: List[Json], at: Tree): Json =
    expr(at)("list", if items.isEmpty then obj() else obj("items" -> Arr(items)))
  private def binary(op: String, l: Json, r: Json, at: Tree): Json =
    expr(at)("binary", obj("op" -> Str(op), "left" -> l, "right" -> r))
  private def collection(op: String, recv: Json, args: List[Json], at: Tree): Json =
    expr(at)("collection", obj("op" -> Str(op), "receiver" -> recv, "args" -> Arr(args)))

  private def placing[A](at: Option[Json])(body: => A): A =
    val was = placedAt
    placedAt = at
    try body
    finally placedAt = was

  private def irTypeName(sym: Symbol): String = sym.fullName.replace("$.", ".").stripSuffix("$")

  private def typeRef(t: TypeRepr, at: Tree): Json =
    val w = t.widen.dealias
    if w =:= TypeRepr.of[Boolean] then obj("bool" -> obj())
    else if w =:= TypeRepr.of[Int] then obj("int" -> obj())
    else if w.typeSymbol.flags.is(Flags.Case) || w.typeSymbol.flags.is(Flags.Enum) then obj("named" -> Str(irTypeName(w.typeSymbol)))
    else fail(at, s"no IR type for ${t.show}")

  private def local(sym: Symbol): Boolean =
    val owner = sym.maybeOwner
    !owner.isNoSymbol && (owner.isDefDef || (owner.isValDef && local(owner)))

  private def nameOf(sym: Symbol): String = renamed.getOrElse(sym, sym.name)

  private def isEnumCase(sym: Symbol): Boolean = sym.flags.is(Flags.Enum) && (sym.isValDef || sym.flags.is(Flags.Module) || sym.isTerm)

  private def enumOf(sym: Symbol): Symbol =
    val t = sym.termRef.widen.dealias
    if t.typeSymbol.flags.is(Flags.Enum) && !t.typeSymbol.flags.is(Flags.Case) then t.typeSymbol
    else t.baseClasses.find(c => c.flags.is(Flags.Enum) && !c.flags.is(Flags.Case)).getOrElse(t.typeSymbol)

  private def sugar(sym: Symbol): Boolean = sym.annotations.exists(_.tpe.typeSymbol.fullName == "framework.dsl.sugar")

  // A definition's tree. Defs outside the macro argument need `-Yretain-trees`; without it the
  // DefDef arrives with no right-hand side and the reifier says so at the use.
  private def definition(sym: Symbol, at: Tree): Tree =
    val d = scala.util.Try(sym.tree).getOrElse(fail(at, s"${sym.fullName} has no tree"))
    d match
      case DefDef(_, _, _, None) | ValDef(_, _, None) =>
        fail(at, s"${sym.fullName} has no retained body: compile with -Yretain-trees")
      case _ => d

  private def functionJson(name: String, params: List[ValDef], body: Term, at: Tree): Json =
    obj(
      "name" -> Str(name),
      "position" -> position(at.pos),
      "params" -> Arr(params.map(p => obj("name" -> Str(nameOf(p.symbol)), "type" -> typeRef(p.tpt.tpe, p)))),
      "body" -> lift(body)
    )

  // A function a body calls, lifted on its first call.
  private def callee(sym: Symbol, at: Tree): String =
    val name = sym.fullName
    if lifting(name) then
      fail(at, s"${sym.name} calls itself, directly or through another function: a recursive function has no IR form, so write the bounded computation out")
    if !functions.contains(name) then
      definition(sym, at) match
        case d @ DefDef(_, _, _, Some(rhs)) =>
          lifting += name
          functions(name) = obj() // reserve the place, so the order is first-reached
          val saved = placedAt
          placedAt = None
          functions(name) = functionJson(name, d.termParamss.flatMap(_.params), rhs, d)
          placedAt = saved
          lifting -= name
        case _ => fail(at, s"$name is not a function of the lifted sources")
    name

  // A sugar call: its definition's body with the call's arguments bound, at the call's position.
  private def expand(call: Term): Json =
    def parts(t: Term, args: List[List[Term]]): (Term, List[List[Term]]) = t match
      case Apply(fn, as)     => parts(fn, as :: args)
      case TypeApply(fn, _)  => parts(fn, args)
      case _                 => (t, args)
    val (fn, argss) = parts(call, Nil)
    definition(fn.symbol, call) match
      case d @ DefDef(_, _, _, Some(rhs)) =>
        val params = d.termParamss.flatMap(_.params)
        params.zip(argss.flatten).foreach((p, a) => bound(p.symbol) = a)
        placing(Some(position(call.pos)))(lift(rhs))
      case _ => fail(call, s"sugar ${fn.symbol.name} has no definition to expand")

  // ### Expressions

  def lift(t: Term): Json = t match
    case r: Ref if bound.contains(r.symbol) =>
      // An argument of an expanded sugar call: lifted where it is written.
      placing(None)(lift(bound(r.symbol)))
    case Select(recv, "outcome") if recv.tpe.widen.dealias.typeSymbol.fullName == "framework.dsl.Ok" =>
      knownOutcome(recv)
    case Inlined(_, Nil, e) => lift(e)
    case Typed(Repeated(items, _), _) => list(items.map(lift), t)
    case Repeated(items, _)           => list(items.map(lift), t)
    case Typed(e, _)                  => lift(e)
    case Block(Nil, e)                => lift(e)
    case NamedArg(_, e)               => lift(e)
    case Literal(BooleanConstant(b))  => lit("bool", Bool(b), t)
    case Literal(IntConstant(i))      => lit("int", Str(i.toString), t)
    case Literal(StringConstant(s))   => lit("text", Str(s), t)

    case Apply(Select(base, "copy"), args) => copyOf(base, args, t)

    case Block((v: ValDef) :: _, _) if v.symbol.flags.is(Flags.Mutable) =>
      fail(v, s"`var ${v.name}` has no IR form: a step function is one pure expression, so write the value it ends with")
    case While(_, _) => fail(t, "a loop has no IR form: a step function is one pure expression")
    case Block((w: While) :: _, _) => fail(w, "a loop has no IR form: a step function is one pure expression")
    case Block(ValDef(name, _, Some(rhs)) :: rest, e) =>
      expr(t)("let", obj("name" -> Str(name), "value" -> lift(rhs), "body" -> lift(Block(rest, e))))

    case If(c, a, b) => expr(t)("if", obj("condition" -> lift(c), "then" -> lift(a), "else" -> lift(b)))

    case Match(scrutinee, cases) =>
      val lifted = cases.map { case CaseDef(p, guard, body) =>
        Json.Obj(List("pattern" -> pattern(p)) ++ guard.map(g => "guard" -> lift(g)) ++ List("body" -> lift(body)))
      }
      expr(t)("match", obj("scrutinee" -> lift(scrutinee), "cases" -> Arr(lifted)))

    case Ident("Nil") => list(Nil, t)
    case Apply(TypeApply(Select(Ident("List"), "apply"), _), List(items)) => list(varargs(items).map(lift), t)

    // The framework's step record, with its default facts and explanation.
    case Apply(TypeApply(Select(companion, "apply"), _), args)
        if companion.tpe.typeSymbol.companionClass.fullName == stepType =>
      val all = args.map {
        case TypeApply(Select(_, name), _) if name.endsWith("$default$3") => list(Nil, t)
        case TypeApply(Select(_, name), _) if name.endsWith("$default$4") => lit("text", Str(""), t)
        case a                                                            => lift(a)
      }
      expr(t)("construct", obj("type" -> Str(stepType), "args" -> Arr(all)))

    // A bounded counter written from an integer, `UpTo(n)`, is the integer: Go checks its range.
    case Apply(Apply(TypeApply(fn @ Select(_, "apply"), _), List(n)), _)
        if fn.symbol.owner.name.stripSuffix("$") == "UpTo" && fn.symbol.owner.fullName.startsWith("framework.dsl.") =>
      lift(n)
    // A varargs parameter read as the list it is.
    case Select(recv, "toList") if recv.tpe.widen.dealias.baseClasses.exists(_.fullName == "scala.collection.immutable.Seq") =>
      lift(recv)

    // Native collections (model/design/reachable-exploration.md §6.1, proposed `Collection` node).
    case Apply(Apply(TypeApply(Select(m, "updated"), _), List(k, v)), _) if isMap(m) =>
      collection("OP_UPDATED", lift(m), List(lift(k), lift(v)), t)
    case Apply(TypeApply(Select(m, "updated"), _), List(k, v)) if isMap(m) =>
      collection("OP_UPDATED", lift(m), List(lift(k), lift(v)), t)
    case Apply(TypeApply(Select(m, "getOrElse"), _), List(k, d)) if isMap(m) =>
      collection("OP_GET_OR", lift(m), List(lift(k), lift(d)), t)
    case Apply(Select(m, "removed"), List(k)) if isMap(m) =>
      collection("OP_REMOVED", lift(m), List(lift(k)), t)
    case Apply(Select(m, "contains"), List(k)) if isMap(m) =>
      collection("OP_HAS_KEY", lift(m), List(lift(k)), t)
    case Apply(Select(m, "apply"), List(k)) if isMap(m) =>
      collection("OP_GET", lift(m), List(lift(k)), t)
    case Apply(Select(s, "incl"), List(x)) if isSet(s) =>
      collection("OP_INCL", lift(s), List(lift(x)), t)
    case Apply(Select(s, "excl"), List(x)) if isSet(s) =>
      collection("OP_EXCL", lift(s), List(lift(x)), t)
    case Select(s, "size") if isSet(s) || isMap(s) =>
      collection("OP_SIZE", lift(s), Nil, t)
    case Apply(Select(recv, "contains"), List(x)) =>
      binary("OP_CONTAINS", lift(x), lift(recv), t)
    case Apply(TypeApply(Select(recv, "contains"), _), List(x)) =>
      binary("OP_CONTAINS", lift(x), lift(recv), t)
    case a: Apply if a.symbol.owner.fullName.startsWith("scala.collection") =>
      fail(a, s"outside the liftable subset: ${a.symbol.owner.name}.${a.symbol.name}")

    case Select(recv, "unary_!") => expr(t)("unary", obj("op" -> Str("OP_NOT"), "operand" -> lift(recv)))
    case Apply(Select(l, op), List(r)) if binaryOps.contains(op) => binary(binaryOps(op), lift(l), lift(r), t)

    // Sugar: expanded from its own definition; the reifier names no sugar.
    case a: Apply if sugar(a.symbol) => expand(a)

    // A record built from its constructor.
    case Apply(Select(companion, "apply"), args) if companion.tpe.typeSymbol.companionClass.flags.is(Flags.Case) =>
      val cls = companion.tpe.typeSymbol.companionClass
      expr(t)("construct", obj("type" -> Str(irTypeName(cls)), "args" -> Arr(args.map(lift))))

    // A call of another function of the lifted sources.
    case Apply(fn, args) if fn.symbol.isDefDef && !fn.symbol.owner.fullName.startsWith("scala") =>
      val name = callee(fn.symbol, t)
      expr(t)("call", obj("function" -> Str(name), "args" -> Arr(args.map(lift))))

    case r: Ref if isEnumCase(r.symbol) =>
      lit("enum", obj("type" -> Str(irTypeName(enumOf(r.symbol))), "case" -> Str(r.symbol.name)), t)
    case r: Ref if local(r.symbol) => expr(t)("var", Str(nameOf(r.symbol)))
    case Select(recv, field) if recv.tpe.widen.dealias.typeSymbol.caseFields.exists(_.name == field) =>
      expr(t)("field", obj("base" -> lift(recv), "field" -> Str(field)))
    // A value declared elsewhere: its definition, lifted in place, at its own position.
    case r: Ref if r.symbol.isValDef =>
      definition(r.symbol, r) match
        case ValDef(_, _, Some(rhs)) => placing(None)(lift(rhs))
        case _                       => fail(t, s"${r.symbol.fullName} has no definition to lift")

    case other => fail(other, s"outside the liftable subset: ${other.show}")

  private def isMap(t: Term): Boolean = t.tpe.widen.dealias.baseClasses.exists(_.fullName == "scala.collection.immutable.Map")
  private def isSet(t: Term): Boolean = t.tpe.widen.dealias.baseClasses.exists(_.fullName == "scala.collection.immutable.Set")

  private def varargs(t: Term): List[Term] = t match
    case Typed(Repeated(items, _), _) => items
    case Repeated(items, _)           => items
    case other                        => List(other)

  // `copy` keeps a field its argument is the default getter for, and replaces the others.
  private def copyOf(base: Term, args: List[Term], at: Term): Json =
    val fields = base.tpe.widen.dealias.typeSymbol.caseFields.map(_.name)
    val updates = args.zip(fields).collect {
      case (a, f) if !unwrap(a).symbol.name.contains("copy$default$") => obj("name" -> Str(f), "value" -> lift(a))
    }
    expr(at)("copy", obj("base" -> lift(base), "updates" -> Arr(updates)))

  // `ok.outcome`: the outcome of the `given Ok` the call resolved, lifted where the given writes it.
  private def knownOutcome(recv: Term): Json = unwrap(recv) match
    case r: Ref if bound.contains(r.symbol) => knownOutcome(bound(r.symbol))
    case r: Ref =>
      definition(r.symbol, r) match
        case ValDef(_, _, Some(rhs)) =>
          unwrap(rhs) match
            case Apply(_, List(outcome)) => placing(None)(lift(outcome))
            case other                   => fail(other, s"an Ok names its outcome, not ${other.show}")
        case DefDef(_, _, _, Some(rhs)) =>
          unwrap(rhs) match
            case Apply(_, List(outcome)) => placing(None)(lift(outcome))
            case other                   => fail(other, s"an Ok names its outcome, not ${other.show}")
        case _ => fail(r, "an Ok given has no definition")
    case other => fail(other, s"an Ok given is named, not ${other.show}")

  private def pattern(p: Tree): Json = p match
    case Wildcard()                     => obj("wildcard" -> obj())
    case Bind(n, Wildcard())            => obj("bind" -> obj("name" -> Str(n)))
    case r: Ref if isEnumCase(r.symbol) => obj("literal" -> obj("enum" -> obj("type" -> Str(irTypeName(enumOf(r.symbol))), "case" -> Str(r.symbol.name))))
    case other                          => fail(other, s"spike: pattern ${other.show} is not reified")
