/* The lifter: reads the typed trees (TASTy) of Scala Models compiled against model/scala's framework,
 * and emits the Umpire IR they declare. It lifts what authors wrote, as written: the `machine`
 * blocks, the action chains, and the step functions' bodies, including native `match`, `if`,
 * `copy` and local `val`s. Anything outside the subset stops the lift with the source position of
 * the construct, so a Model that cannot become IR is reported where it was written.
 *
 * TASTy is read after compilation rather than by a macro during it: a macro sees a function's body
 * only for definitions of its own compilation run, and there only after pattern matching has been
 * compiled away; TASTy keeps the typed tree with every `match` intact.
 */
package umpire.lift

import com.google.protobuf.util.JsonFormat
import java.nio.file.{Files, Path, Paths}
import java.util.zip.ZipFile
import scala.collection.mutable
import scala.jdk.CollectionConverters.*
import scala.quoted.*
import scala.tasty.inspector.*
import io.temporal.server.api.umpire.v1 as ir

/** A construct the IR cannot express, at the position the author wrote it. */
final case class LiftError(position: String, message: String) extends Exception(s"$position: $message")

class Lifter(roots: Set[String], sourcePrefix: String) extends Inspector:
  var model: Option[ir.Model] = None
  var error: Option[LiftError] = None

  // A lift error is kept rather than thrown through the compiler, which would report it as a crash.
  def inspect(using Quotes)(tastys: List[Tasty[quotes.type]]): Unit =
    try liftAll(tastys) catch case e: LiftError => error = Some(e)

  private def liftAll(using Quotes)(tastys: List[Tasty[quotes.type]]): Unit =
    import quotes.reflect.*

    // Every definition in the inspected files, by symbol, so a reference resolves to its body.
    val defs = mutable.Map.empty[Symbol, Definition]
    def isFunction(sym: Symbol): Boolean = defs.get(sym) match
      case Some(_: DefDef) => true
      case _               => false
    for t <- tastys do
      object index extends TreeTraverser:
        override def traverseTree(tree: Tree)(owner: Symbol): Unit =
          tree match
            case d: ValDef => defs(d.symbol) = d
            case d: DefDef => defs(d.symbol) = d
            case _         => ()
          super.traverseTree(tree)(owner)
      index.traverseTree(t.ast)(Symbol.spliceOwner)

    val types = mutable.LinkedHashMap.empty[String, ir.Type]
    val functions = mutable.LinkedHashMap.empty[String, ir.Function]
    val actions = mutable.LinkedHashMap.empty[String, ir.Action]
    val machines = mutable.LinkedHashMap.empty[String, ir.Machine]
    // The integer ranges a state's `Finite` given declares, by the state's type name.
    val intRanges = mutable.Map.empty[String, (Long, Long)]

    // TASTy records a source path relative to the build that compiled it; the prefix makes it
    // relative to the repository. A tree the lifter builds itself, such as the block left after a
    // `require`, has no span.
    def pos(t: Tree): ir.Position = scala.util.Try {
      val p = t.pos
      ir.Position.newBuilder().setFile(sourcePrefix + p.sourceFile.path).setLine(p.startLine + 1).build()
    }.getOrElse(ir.Position.getDefaultInstance)

    def where(t: Tree): String = s"${pos(t).getFile}:${pos(t).getLine}"
    def fail(t: Tree, message: String): Nothing = throw LiftError(where(t), message)

    // ### Types

    def isEnumCase(s: Symbol): Boolean = s.flags.is(Flags.Enum) && s.flags.is(Flags.Case)
    /** The enum a case belongs to: the class whose companion object declares the case. */
    def enumOf(caseSym: Symbol): Symbol = caseSym.owner.companionClass
    def fieldTypes(cls: Symbol): List[(String, TypeRepr)] =
      cls.caseFields.map(f => f.name -> cls.typeRef.memberType(f).widen)

    def typeRef(tpe: TypeRepr, at: Tree, owner: String = ""): ir.TypeRef =
      val t = tpe.dealias.widen
      val sym = t.typeSymbol
      if sym == defn.BooleanClass then ir.TypeRef.newBuilder().setBool(ir.Empty.getDefaultInstance).build()
      else if sym == defn.IntClass && owner.isEmpty then ir.TypeRef.newBuilder().setInt(ir.Empty.getDefaultInstance).build()
      else if sym == defn.IntClass then
        val (lo, hi) = intRanges.getOrElse(owner, fail(at, s"an Int field of $owner has no range: give the state a " +
          "`given Finite[Int] = Finite.upTo(...)` where its Finite is derived"))
        ir.TypeRef.newBuilder().setIntRange(ir.IntRange.newBuilder().setLow(lo).setHigh(hi)).build()
      else if sym.fullName == "scala.collection.immutable.List" then
        ir.TypeRef.newBuilder().setList(typeRef(t.typeArgs.head, at, owner)).build()
      else
        declareType(sym, at)
        ir.TypeRef.newBuilder().setNamed(sym.fullName).build()

    def declareType(sym: Symbol, at: Tree): Unit =
      if !types.contains(sym.fullName) && sym != defn.NothingClass then
        types(sym.fullName) = ir.Type.getDefaultInstance // placeholder against recursion
        val b = ir.Type.newBuilder().setName(sym.fullName).setPosition(scala.util.Try(pos(sym.tree)).getOrElse(pos(at)))
        if sym.flags.is(Flags.Enum) then
          val e = ir.Enum.newBuilder()
          for c <- sym.children do
            val cb = ir.Case.newBuilder().setName(c.name)
            if c.isClassDef then
              for (n, ft) <- fieldTypes(c) do cb.addFields(ir.Field.newBuilder().setName(n).setType(typeRef(ft, at, sym.fullName)))
            e.addCases(cb)
          b.setEnum(e)
        else if sym.flags.is(Flags.Case) then
          val r = ir.Record.newBuilder()
          for (n, ft) <- fieldTypes(sym) do r.addFields(ir.Field.newBuilder().setName(n).setType(typeRef(ft, at, sym.fullName)))
          b.setRecord(r)
        else fail(at, s"${sym.fullName} is neither an enum nor a case class, so it has no finite catalog")
        types(sym.fullName) = b.build()

    // ### Values folded at lift time: names, strings and bounds the declarations reference

    def resolve(t: Term): Term = t match
      case Typed(e, _)  => resolve(e)
      case Inlined(_, Nil, e) => resolve(e)
      case r: Ref if !r.symbol.flags.is(Flags.Param) && r.symbol.isValDef && defs.contains(r.symbol) =>
        defs(r.symbol) match
          case ValDef(_, _, Some(rhs)) if !isEnumCase(r.symbol) => resolve(rhs)
          case _                                              => t
      case _ => t

    def constString(t: Term): String = resolve(t) match
      case Literal(StringConstant(s)) => s
      // `x.name` of a case class value declared elsewhere: the argument its constructor got.
      case Select(qual, field) =>
        resolve(qual) match
          case Apply(Select(companion, "apply"), args) =>
            val cls = companion.tpe.typeSymbol.companionClass
            val i = cls.caseFields.indexWhere(_.name == field)
            if i < 0 then fail(t, s"cannot read .$field of this value") else constString(args(i))
          case other => fail(t, s"cannot fold ${other.show} to a string")
      // Party(...), Entity(...), Family(...), Observation(...): the name is the first argument.
      case Apply(Select(_, "apply"), first :: _) => constString(first)
      case other => fail(t, s"expected a string, got ${other.show}")

    def constInt(t: Term): Long = resolve(t) match
      case Literal(IntConstant(i)) => i.toLong
      case other                   => fail(t, s"expected an integer constant, got ${other.show}")

    // The `given Finite[S]` blocks: an Int field's range is the `Finite.upTo(bound)` in scope there.
    for (sym, d) <- defs do d match
      case ValDef(_, tpt, Some(Block(stats, _))) if tpt.tpe.typeSymbol.name == "Finite" =>
        val state = tpt.tpe.typeArgs.headOption.map(_.dealias.typeSymbol.fullName).getOrElse("")
        for case ValDef(_, _, Some(Apply(Select(_, "upTo"), List(bound)))) <- stats do intRanges(state) = (0L, constInt(bound))
      case _ => ()

    // ### Expressions

    def lit(v: ir.Value.Builder, at: Tree): ir.Expr =
      ir.Expr.newBuilder().setPosition(pos(at)).setLiteral(v).build()
    def expr(at: Tree)(f: ir.Expr.Builder => ir.Expr.Builder): ir.Expr = f(ir.Expr.newBuilder().setPosition(pos(at))).build()
    def enumLiteral(sym: Symbol, at: Tree): ir.Expr =
      declareType(enumOf(sym), at)
      lit(ir.Value.newBuilder().setEnum(ir.EnumValue.newBuilder().setType(enumOf(sym).fullName).setCase(sym.name)), at)
    def list(items: Seq[ir.Expr], at: Tree): ir.Expr = expr(at)(_.setList(ir.ListOf.newBuilder().addAllItems(items.asJava)))
    def text(s: String, at: Tree): ir.Expr = lit(ir.Value.newBuilder().setText(s), at)
    def varargs(t: Term): List[Term] = t match
      case Typed(Repeated(items, _), _) => items
      case Repeated(items, _)           => items
      case other                        => List(other)

    val binaryOps = Map("==" -> ir.Binary.Op.OP_EQ, "!=" -> ir.Binary.Op.OP_NE, "&&" -> ir.Binary.Op.OP_AND,
      "||" -> ir.Binary.Op.OP_OR, "<" -> ir.Binary.Op.OP_LT, "<=" -> ir.Binary.Op.OP_LE, ">" -> ir.Binary.Op.OP_GT,
      ">=" -> ir.Binary.Op.OP_GE, "+" -> ir.Binary.Op.OP_ADD, "-" -> ir.Binary.Op.OP_SUB, "++" -> ir.Binary.Op.OP_CONCAT)
    val stepType = "umpire.Step"

    def step(outcome: ir.Expr, state: ir.Expr, facts: ir.Expr, because: ir.Expr, at: Tree): ir.Expr =
      expr(at)(_.setConstruct(ir.Construct.newBuilder().setType(stepType).addAllArgs(List(outcome, state, facts, because).asJava)))

    /** The function a reference names, lifting its body on first use. */
    def callee(sym: Symbol, at: Tree): String =
      if !functions.contains(sym.fullName) then
        val d = defs.get(sym) match
          case Some(d: DefDef) => d
          case _               => fail(at, s"${sym.fullName} is not a function of the lifted sources")
        functions(sym.fullName) = ir.Function.getDefaultInstance
        functions(sym.fullName) = function(sym.fullName, d.termParamss.flatMap(_.params), d.rhs.get, d)
      sym.fullName

    def function(name: String, params: List[ValDef], body: Term, at: Tree): ir.Function =
      val b = ir.Function.newBuilder().setName(name).setPosition(pos(at))
      for p <- params do b.addParams(ir.Param.newBuilder().setName(p.name).setType(typeRef(p.tpt.tpe, p)))
      val (requires, rest) = stripContracts(body)
      requires.foreach(r => b.setRequires(lift(r)))
      b.setBody(lift(rest)).build()

    /** `{ require(p); body }.ensuring(q)` is `body` under precondition `p`: the contracts are what
      * Stainless proves, and the interpreter evaluates the body. */
    def stripContracts(t: Term): (Option[Term], Term) = t match
      case Apply(Select(Apply(TypeApply(e, _), List(body)), "ensuring"), _) if e.symbol.name == "Ensuring" => stripContracts(body)
      case Block(Apply(r, List(cond)) :: rest, e) if r.symbol.name == "require" =>
        (Some(cond), if rest.isEmpty then e else Block(rest, e))
      case other => (None, other)

    def lift(t: Term): ir.Expr = t match
      case Typed(e, _)             => lift(e)
      case Inlined(_, Nil, e)      => lift(e)
      case Block(Nil, e)           => lift(e)
      case Literal(BooleanConstant(b)) => lit(ir.Value.newBuilder().setBool(b), t)
      case Literal(IntConstant(i)) => lit(ir.Value.newBuilder().setInt(i), t)
      case Literal(StringConstant(s)) => text(s, t)

      // `copy` keeps a field its argument is the default getter for, and replaces the others. Named
      // arguments out of field order arrive as a block of synthetic vals, which are substituted back.
      case Block(stats, Apply(Select(base, "copy"), args))
          if stats.nonEmpty && stats.forall { case v: ValDef => v.name.contains("$"); case _ => false } =>
        val bound = stats.collect { case v: ValDef => v.symbol -> v.rhs.get }.toMap
        def unbind(a: Term): Term = a match
          case r: Ref if bound.contains(r.symbol) => bound(r.symbol)
          case _                                  => a
        copyOf(base, args.map {
          case NamedArg(n, v) => NamedArg(n, unbind(v))
          case a              => unbind(a)
        }, t)
      case Apply(Select(base, "copy"), args) => copyOf(base, args, t)

      case Block((v: ValDef) :: _, _) if v.symbol.flags.is(Flags.Mutable) =>
        fail(v, s"`var ${v.name}` has no IR form: a step function is one pure expression, so write the value it ends with")
      case While(_, _) => fail(t, "a loop has no IR form: a step function is one pure expression")
      case Block(ValDef(name, _, Some(rhs)) :: rest, e) =>
        expr(t)(_.setLet(ir.Let.newBuilder().setName(name).setValue(lift(rhs)).setBody(lift(Block(rest, e)))))

      case If(c, a, b) => expr(t)(_.setIf(ir.If.newBuilder().setCondition(lift(c)).setThen(lift(a)).setElse(lift(b))))

      case Match(scrutinee, cases) =>
        val m = ir.Match.newBuilder().setScrutinee(lift(scrutinee))
        for CaseDef(p, guard, body) <- cases do
          val c = ir.MatchCase.newBuilder().setPattern(pattern(p)).setBody(lift(body))
          guard.foreach(g => c.setGuard(lift(g)))
          m.addCases(c)
        expr(t)(_.setMatch(m))

      // The prelude's constructors, which both sides of the kernel supply.
      case Apply(TypeApply(Ident("step"), _), List(o, s, f)) if t.symbol.fullName.startsWith("umpire.prelude") =>
        step(lift(o), lift(s), lift(f), text("", t), t)
      case Apply(TypeApply(Ident("one" | "facts1"), _), List(x)) if t.symbol.fullName.startsWith("umpire.prelude") => list(List(lift(x)), t)
      case TypeApply(Ident("none" | "facts0"), _) if t.symbol.fullName.startsWith("umpire.prelude") => list(Nil, t)
      case Ident("Nil") => list(Nil, t)
      case Apply(TypeApply(Select(Ident("List"), "apply"), _), List(items)) => list(varargs(items).map(lift), t)

      // The framework's step record, with its default facts and explanation.
      case Apply(TypeApply(Select(companion, "apply"), _), args) if companion.tpe.typeSymbol.companionClass.fullName == stepType =>
        val all = args.map {
          case TypeApply(Select(_, name), _) if name.endsWith("$default$3") => list(Nil, t)
          case TypeApply(Select(_, name), _) if name.endsWith("$default$4") => text("", t)
          case a                                                           => lift(a)
        }
        step(all(0), all(1), all(2), all(3), t)


      case Apply(Select(recv, "contains"), List(x)) =>
        expr(t)(_.setBinary(ir.Binary.newBuilder().setOp(ir.Binary.Op.OP_CONTAINS).setLeft(lift(x)).setRight(lift(recv))))
      case Apply(TypeApply(Select(recv, "contains"), _), List(x)) =>
        expr(t)(_.setBinary(ir.Binary.newBuilder().setOp(ir.Binary.Op.OP_CONTAINS).setLeft(lift(x)).setRight(lift(recv))))

      case Select(recv, "unary_!") => expr(t)(_.setUnary(ir.Unary.newBuilder().setOp(ir.Unary.Op.OP_NOT).setOperand(lift(recv))))
      case Apply(Select(l, op), List(r)) if binaryOps.contains(op) =>
        expr(t)(_.setBinary(ir.Binary.newBuilder().setOp(binaryOps(op)).setLeft(lift(l)).setRight(lift(r))))
      case Apply(TypeApply(Select(l, "++"), _), List(r)) =>
        expr(t)(_.setBinary(ir.Binary.newBuilder().setOp(ir.Binary.Op.OP_CONCAT).setLeft(lift(l)).setRight(lift(r))))

      // A record, or an enum case with fields, built from its constructor.
      case Apply(Select(companion, "apply"), args) if companion.tpe.typeSymbol.companionClass.flags.is(Flags.Case) =>
        val cls = companion.tpe.typeSymbol.companionClass
        val c = ir.Construct.newBuilder().addAllArgs(args.map(lift).asJava)
        if cls.flags.is(Flags.Enum) then
          declareType(enumOf(cls), t)
          c.setType(enumOf(cls).fullName).setCase(cls.name)
        else
          declareType(cls, t)
          c.setType(cls.fullName)
        expr(t)(_.setConstruct(c))

      // A call of another function of the lifted sources.
      case Apply(fn, args) if isFunction(fn.symbol) =>
        val name = callee(fn.symbol, t)
        expr(t)(_.setCall(ir.Call.newBuilder().setFunction(name).addAllArgs(args.map(lift).asJava)))

      case r: Ref if isEnumCase(r.symbol) => enumLiteral(r.symbol, t)
      // A parameter, a local `val` or a pattern-bound name: every name a function's own scope owns.
      case r: Ref if r.symbol.maybeOwner.isDefDef => expr(t)(_.setVar(r.symbol.name))
      // A string read off a declared value, such as an observation's name, is a constant.
      case Select(recv, _) if t.tpe.widen <:< defn.StringClass.typeRef && !recv.symbol.maybeOwner.isDefDef =>
        text(constString(t), t)
      case Select(recv, field) if recv.tpe.widen.typeSymbol.caseFields.exists(_.name == field) =>
        expr(t)(_.setField(ir.FieldAccess.newBuilder().setBase(lift(recv)).setField(field)))
      // A value declared elsewhere: its definition, lifted in place.
      case r: Ref if r.symbol.isValDef && defs.contains(r.symbol) =>
        defs(r.symbol) match
          case ValDef(_, _, Some(rhs)) => lift(rhs)
          case _                       => fail(t, s"${r.symbol.fullName} has no definition to lift")

      case lambda @ Block(List(DefDef("$anonfun", List(TermParamClause(params)), _, Some(body))), _: Closure) =>
        val l = ir.Lambda.newBuilder().setBody(lift(body))
        for p <- params do l.addParams(ir.Param.newBuilder().setName(p.name).setType(typeRef(p.tpt.tpe, p)))
        expr(lambda)(_.setLambda(l))

      case other => fail(other, s"outside the liftable subset: ${other.show}")

    def copyOf(base: Term, args: List[Term], at: Tree): ir.Expr =
      val c = ir.Copy.newBuilder().setBase(lift(base))
      val fields = base.tpe.widen.typeSymbol.caseFields
      for (arg, i) <- args.zipWithIndex do arg match
        case NamedArg(name, value) => c.addUpdates(ir.NamedExpr.newBuilder().setName(name).setValue(lift(value)))
        case Select(_, getter) if getter.startsWith("copy$default$") => ()
        case TypeApply(Select(_, getter), _) if getter.startsWith("copy$default$") => ()
        case value => c.addUpdates(ir.NamedExpr.newBuilder().setName(fields(i).name).setValue(lift(value)))
      expr(at)(_.setCopy(c))

    def pattern(p: Tree): ir.Pattern = p match
      case Wildcard()             => ir.Pattern.newBuilder().setWildcard(ir.Empty.getDefaultInstance).build()
      case Bind(name, inner)      => ir.Pattern.newBuilder().setBind(ir.Bind.newBuilder().setName(name).setPattern(pattern(inner))).build()
      case Alternatives(ps)       => ir.Pattern.newBuilder().setAlternatives(ir.Alternatives.newBuilder().addAllPatterns(ps.map(pattern).asJava)).build()
      case Literal(BooleanConstant(b)) => ir.Pattern.newBuilder().setLiteral(ir.Value.newBuilder().setBool(b)).build()
      case r: Ref if isEnumCase(r.symbol) =>
        declareType(enumOf(r.symbol), p)
        ir.Pattern.newBuilder().setLiteral(ir.Value.newBuilder().setEnum(
          ir.EnumValue.newBuilder().setType(enumOf(r.symbol).fullName).setCase(r.symbol.name))).build()
      case Unapply(fun, _, fields) =>
        val cls = fun.symbol.owner.companionClass
        if !cls.flags.is(Flags.Enum) then fail(p, s"only enum cases are matched by constructor, not ${cls.fullName}")
        declareType(enumOf(cls), p)
        ir.Pattern.newBuilder().setCase(ir.CasePattern.newBuilder().setType(enumOf(cls).fullName).setCase(cls.name)
          .addAllFields(fields.map(pattern).asJava)).build()
      case TypedOrTest(inner, _) => pattern(inner)
      case other => fail(other, s"outside the liftable patterns: ${other.show}")

    // ### Declarations: actions and machines, from the framework calls that declare them

    /** A call's function name and its argument lists, outermost last, through type applications. */
    def call(t: Term): Option[(String, List[List[Term]])] = t match
      case Apply(fn, args) => call(fn).map((n, as) => (n, as :+ args))
      case TypeApply(fn, _) => call(fn)
      case Ident(n)        => Some(n -> Nil)
      case Select(_, n)    => Some(n -> Nil)
      case _               => None

    def action(ref: Term): String =
      val sym = resolveSymbol(ref)
      if !actions.contains(sym.fullName) then
        val d = defs.get(sym) match
          case Some(v: ValDef) => v
          case _               => fail(ref, s"${sym.fullName} is not an action declared in the lifted sources")
        actions(sym.fullName) = actionOf(sym.fullName, d.rhs.get)
      sym.fullName

    /** The symbol a reference finally names, through aliases such as `val workerStop = worker.workerStop`. */
    def resolveSymbol(ref: Term): Symbol = ref match
      case r: Ref => defs.get(r.symbol) match
        case Some(ValDef(_, _, Some(rhs: Ref))) => resolveSymbol(rhs)
        case _                                  => r.symbol
      case Typed(e, _) => resolveSymbol(e)
      case other       => fail(other, "expected a reference to a declared value")

    def actionOf(id: String, chain: Term): ir.Action =
      val b = ir.Action.newBuilder().setId(id).setPosition(pos(chain))
      def walk(t: Term): Unit = t match
        case Apply(Ident("action"), List(name, party)) =>
          b.setName(constString(name)).setParty(constString(party))
        case Apply(Ident("timer"), List(name)) =>
          b.setName(constString(name)).setParty("system").setTimer(true)
        case Apply(Select(inner, "on"), List(e))      => walk(inner); b.setOn(constString(e))
        case Apply(Select(inner, "creates"), List(e)) => walk(inner); b.setCreates(constString(e))
        case Apply(Select(inner, "results"), List(n)) => walk(inner); b.setResults(constString(n))
        case Apply(Select(inner, "schema"), List(names)) =>
          walk(inner); varargs(names).foreach(n => b.addSchemas(constString(n)))
        case Apply(Apply(TypeApply(Select(inner, "input"), List(tpt)), List(name)), _) =>
          walk(inner); b.addInputs(ir.Param.newBuilder().setName(constString(name)).setType(typeRef(tpt.tpe, t)))
        case Apply(Apply(TypeApply(Ident("example"), _), List(inner)), List(value, example)) =>
          walk(inner)
          b.addExamples(ir.Example.newBuilder().setValue(literalValue(value)).setExample(constString(example)))
        case other => fail(other, s"not a part of an action declaration: ${other.show}")
      walk(chain)
      b.build()

    /** A constant value, for an example: an enum case, with constant fields. */
    def literalValue(t: Term): ir.Value = resolve(t) match
      case Literal(BooleanConstant(v)) => ir.Value.newBuilder().setBool(v).build()
      case Literal(IntConstant(v))     => ir.Value.newBuilder().setInt(v).build()
      case r: Ref if isEnumCase(r.symbol) =>
        ir.Value.newBuilder().setEnum(ir.EnumValue.newBuilder().setType(enumOf(r.symbol).fullName).setCase(r.symbol.name)).build()
      case Apply(Select(companion, "apply"), args) if companion.tpe.typeSymbol.companionClass.flags.is(Flags.Enum) =>
        val cls = companion.tpe.typeSymbol.companionClass
        ir.Value.newBuilder().setEnum(ir.EnumValue.newBuilder().setType(enumOf(cls).fullName).setCase(cls.name)
          .addAllFields(args.map(literalValue).asJava)).build()
      case other => fail(other, s"an example is a constant value, not ${other.show}")

    /** A step binding's function: the one an eta-expanded lambda forwards to, or the lambda itself. */
    def stepFunction(fn: Term, machine: String, actionName: String): String = fn match
      case Block(List(DefDef("$anonfun", List(TermParamClause(params)), _, Some(Apply(target, args)))), _: Closure)
          if args.map(_.symbol) == params.map(_.symbol) && isFunction(target.symbol) =>
        callee(target.symbol, fn)
      case Block(List(DefDef("$anonfun", List(TermParamClause(params)), _, Some(body))), _: Closure) =>
        val name = s"$machine.$actionName"
        functions(name) = function(name, params, body, fn)
        name
      case other => fail(other, "a step binds a function")

    def machine(sym: Symbol, rhs: Term): ir.Machine =
      val name = sym.name
      rhs match
        // `source.restrict(family, name)(keep*)`: the source's steps for the kept actions only.
        case Apply(Apply(Select(source, "restrict"), List(family, newName)), List(keep)) =>
          val src = machineOf(resolveSymbol(source), source)
          val kept = varargs(keep).map(action).toSet
          ir.Machine.newBuilder(src).setFamily(constString(family)).setName(constString(newName)).setPosition(pos(rhs))
            .clearSteps().addAllSteps(src.getStepsList.asScala.filter(s => kept(s.getAction)).asJava)
            .clearUnobservable().clearRefines().build()
        case _ =>
          val (types, family, mname, body) = rhs match
            case Apply(Apply(Apply(TypeApply(Ident("machine"), tps), List(f, n)), List(ctx)), _) => (tps, f, n, ctx)
            case other => fail(other, "a machine is declared by `machine[S, O, F](family, name) { ... }`")
          val List(s, o, f) = types.map(_.tpe)
          val b = ir.Machine.newBuilder().setFamily(constString(family)).setName(constString(mname)).setPosition(pos(rhs))
            .setStateType(typeRef(s, rhs).getNamed).setOutcomeType(typeRef(o, rhs).getNamed)
          if f.dealias.typeSymbol != defn.NothingClass then b.setFactType(typeRef(f, rhs).getNamed)
          val stats = body match
            case Block(List(DefDef("$anonfun", _, _, Some(Block(stats, last)))), _: Closure) => stats :+ last
            case other => fail(other, "a machine's body is a block of declarations")
          for stat <- stats do call(stat.asInstanceOf[Term]) match
            case Some(("forEntity", List(List(e), _)))   => b.setEntity(constString(e))
            case Some(("starts", List(_, List(items))))  => varargs(items).foreach(i => b.addStarts(lift(i)))
            case Some(("ends", List(_, List(p))))        => b.setEnds(lift(p))
            case Some(("unobservable", List(List(ts), _))) => varargs(ts).foreach(a => b.addUnobservable(action(a)))
            case Some(("evidence", List(_, List(fn)))) =>
              val evidenceName = s"${b.getName}.evidence"
              fn match
                case Block(List(DefDef("$anonfun", List(TermParamClause(params)), _, Some(body))), _: Closure) =>
                  functions(evidenceName) = function(evidenceName, params, body, fn)
                case other => fail(other, "evidence is a function of the fact")
              b.setEvidence(evidenceName)
            case Some(("refines", List(_, List(product), List(map)))) =>
              val productName = machineOf(resolveSymbol(product), product).getName
              b.setRefines(ir.Refinement.newBuilder().setProduct(productName).setMap(stepFunction(map, b.getName, "refines")))
            case Some(("steps", List(_, List(bindings)))) =>
              for binding <- varargs(bindings) do binding match
                case Apply(TypeApply(Apply(TypeApply(Ident("~>"), _), List(a)), _), List(fn)) =>
                  val id = action(a)
                  b.addSteps(ir.StepBinding.newBuilder().setAction(id).setPosition(pos(binding))
                    .setFunction(stepFunction(fn, b.getName, actions(id).getName)))
                case Apply(TypeApply(Apply(Ident("~>"), List(a)), _), List(fn)) =>
                  val id = action(a)
                  b.addSteps(ir.StepBinding.newBuilder().setAction(id).setPosition(pos(binding))
                    .setFunction(stepFunction(fn, b.getName, actions(id).getName)))
                case other => fail(other, s"a step is `action ~> function`, not ${other.show}")
            case _ => stat match
              case Literal(UnitConstant()) => () // the block's trailing unit
              case _                       => fail(stat, s"not a machine declaration: ${stat.show}")
          b.build()

    def machineOf(sym: Symbol, at: Tree): ir.Machine =
      machines.getOrElseUpdate(sym.fullName, defs.get(sym) match
        case Some(ValDef(_, _, Some(rhs))) => machine(sym, rhs)
        case _                             => fail(at, s"${sym.fullName} is not a machine of the lifted sources"))

    for (sym, d) <- defs.toList.sortBy(_._1.fullName) if roots(sym.fullName) do machineOf(sym, d)

    val m = ir.Model.newBuilder().setSource("model/scala: " + roots.toList.sorted.mkString(", "))
    types.toList.sortBy(_._1).foreach((_, t) => m.addTypes(t))
    functions.toList.sortBy(_._1).foreach((_, f) => m.addFunctions(f))
    actions.toList.sortBy(_._1).foreach((_, a) => m.addActions(a))
    machines.values.toList.sortBy(_.getName).foreach(m.addMachines)
    model = Some(m.build())

/** `lift <model.jar> <classpath file> <out.json> <source prefix> <root>...`: lift the machines named
  * by the roots, the fully qualified names of their `val`s, from the Temporal Models in the jar. The
  * prefix turns the sources' build-relative paths into repository-relative ones. */
@main def lift(jar: String, classpathFile: String, out: String, sourcePrefix: String, roots: String*): Unit =
  val scratch = Files.createTempDirectory("umpire-lift")
  val zip = ZipFile(jar)
  val tastys = zip.entries.asScala.filter(e => e.getName.startsWith("temporal/") && e.getName.endsWith(".tasty")).map { e =>
    val p = scratch.resolve(e.getName)
    Files.createDirectories(p.getParent)
    Files.copy(zip.getInputStream(e), p)
    p.toString
  }.toList.sorted
  val classpath = jar :: Files.readString(Path.of(classpathFile)).trim.split(java.io.File.pathSeparator).toList
  val lifter = Lifter(roots.toSet, sourcePrefix)
  TastyInspector.inspectAllTastyFiles(tastys, Nil, classpath)(lifter)
  lifter.error.foreach { e =>
    if sys.env.contains("LIFT_DEBUG") then e.printStackTrace()
    System.err.println(s"lift: ${e.getMessage}")
    sys.exit(1)
  }
  val json = JsonFormat.printer().print(lifter.model.getOrElse(sys.error("lift: nothing was lifted")))
  Files.writeString(Paths.get(out), json + "\n")
