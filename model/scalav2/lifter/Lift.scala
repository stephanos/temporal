/* The lifter: reads the typed trees (TASTy) of Scala Models compiled against the umpire framework,
 * and emits the Umpire IR they declare. It lifts what authors wrote, as written: the `machine`
 * blocks, the action chains, and the step functions' bodies, including native `match`, `if`,
 * `copy` and local `val`s. Anything outside the subset stops the lift with the source position of
 * the construct, so a Model that cannot become IR is reported where it was written.
 *
 * TASTy is read after compilation rather than by a macro during it: a macro sees a function's body
 * only for definitions of its own compilation run, and there only after pattern matching has been
 * compiled away; TASTy keeps the typed tree with every `match` intact.
 */
// The declarations checks read over the machines -- compositions, channels, monitors, assumptions,
// holes, Properties, Scenarios, Queries and progress claims -- are folded at lift time: one written
// through a helper function, such as a list of Queries per design, is lifted once per call, with the
// call's arguments bound.
//
// A realization is emitted as written: its declarations are data, so each constructor becomes the IR
// message of its name and each argument the field of its parameter's name.
package umpire.lift

import com.google.protobuf.Descriptors.FieldDescriptor
import com.google.protobuf.Message
import com.google.protobuf.util.JsonFormat
import java.nio.file.{Files, Path, Paths}
import java.util.zip.ZipFile
import scala.collection.mutable
import scala.jdk.CollectionConverters.*
import scala.quoted.*
import scala.tasty.inspector.*
import io.temporal.server.api.modelir.v1 as ir

/** A construct the IR cannot express, at the position the author wrote it. */
final case class LiftError(position: String, message: String)
    extends Exception(s"$position: $message")

/**
 * What a declaration folds to at lift time: a name or text the IR uses, or a declaration still being
 * built by the calls chained onto it.
 */
private enum Decl:
  case Text(value: String)

  /** A machine or a composition, by name. */
  case Model(name: String)
  case Items(items: List[Decl])
  case PropertyOn(machine: String, name: String, when: Option[Either[ir.ActionClass, String]])
  case ScenarioOn(machine: String, name: String, start: Option[ir.Expr])

  /** A declared Property or Scenario. */
  case Claim(ref: ir.ClaimRef)
  case QueryNamed(name: String)
  case QueryOn(name: String, form: ir.Query.Form, property: ir.ClaimRef)
  case QueryIn(
      name: String,
      form: ir.Query.Form,
      property: ir.ClaimRef,
      scenario: ir.ClaimRef,
      through: Option[String]
  )

  /** How a Scenario reads a Property: the machine whose refinement it reads through, if any. */
  case Reading(through: Option[String])
  case Bounds(limits: ir.Limits)

  /** A declared Query or progress claim. */
  case Declared(name: String)

class Lifter(roots: Seq[String], prefixes: Map[String, String]) extends Inspector:
  val models = mutable.ArrayBuffer.empty[ir.Model]
  val errors = mutable.ArrayBuffer.empty[LiftError]

  // A lift error is kept rather than thrown through the compiler, which would report it as a crash.
  def inspect(using Quotes)(tastys: List[Tasty[quotes.type]]): Unit =
    try liftAll(tastys)
    catch case e: LiftError => errors += e

  private def liftAll(using Quotes)(tastys: List[Tasty[quotes.type]]): Unit =
    import quotes.reflect.*

    // Every definition in the inspected files, by symbol, so a reference resolves to its body.
    val defs = mutable.Map.empty[Symbol, Definition]
    def isFunction(sym: Symbol): Boolean = defs.get(sym) match
      case Some(_: DefDef) => true
      case _               => false
    // The directory, relative to the repository, that each source's build-relative path is under.
    val sourceRoots = mutable.Map.empty[String, String]
    for t <- tastys do
      object index extends TreeTraverser:
        override def traverseTree(tree: Tree)(owner: Symbol): Unit =
          tree match
            case d: ValDef => defs(d.symbol) = d
            case d: DefDef => defs(d.symbol) = d
            case _         => ()
          super.traverseTree(tree)(owner)
      index.traverseTree(t.ast)(Symbol.spliceOwner)
      val prefix =
        prefixes.collectFirst { case (tasty, p) if t.path.endsWith(tasty) => p }.getOrElse("")
      scala.util.Try(t.ast.pos.sourceFile.path).foreach(path => sourceRoots(path) = prefix)

    val types = mutable.LinkedHashMap.empty[String, ir.Type]
    val functions = mutable.LinkedHashMap.empty[String, ir.Function]
    val actions = mutable.LinkedHashMap.empty[String, ir.Action]
    val machines = mutable.LinkedHashMap.empty[String, ir.Machine]
    val channels = mutable.LinkedHashMap.empty[String, ir.Channel]
    val monitors = mutable.LinkedHashMap.empty[String, ir.Monitor]
    val assumptions = mutable.LinkedHashMap.empty[String, ir.Assumption]
    val holes = mutable.LinkedHashMap.empty[String, ir.Hole]
    val compositions = mutable.LinkedHashMap.empty[String, ir.Composition]
    val properties = mutable.LinkedHashMap.empty[(String, String), ir.Property]
    val scenarios = mutable.LinkedHashMap.empty[(String, String), ir.Scenario]
    val queries = mutable.LinkedHashMap.empty[String, ir.Query]
    val progress = mutable.LinkedHashMap.empty[(String, String), ir.Progress]
    val realizations = mutable.LinkedHashMap.empty[String, ir.Realization]
    // The integer ranges a state's `Finite` given declares, by the state's type name.
    val intRanges = mutable.Map.empty[String, (Long, Long)]
    // The range an opaque type's own `Finite` given declares, by the type's name.
    val opaqueRanges = mutable.Map.empty[String, (Long, Long)]
    // The channel each `Inbox` field of a state holds, by the state's type name and the message type's.
    val channelFields = mutable.Map.empty[(String, String), Symbol]
    // What each value of the lifted sources folded to, so a declaration is lifted once.
    val folded = mutable.Map.empty[Symbol, Decl]
    // The functions whose bodies are being lifted, so a function that calls itself is refused.
    val lifting = mutable.Set.empty[String]
    val binaryOps = Map(
      "==" -> ir.Binary.Op.OP_EQ,
      "!=" -> ir.Binary.Op.OP_NE,
      "&&" -> ir.Binary.Op.OP_AND,
      "||" -> ir.Binary.Op.OP_OR,
      "<" -> ir.Binary.Op.OP_LT,
      "<=" -> ir.Binary.Op.OP_LE,
      ">" -> ir.Binary.Op.OP_GT,
      ">=" -> ir.Binary.Op.OP_GE,
      "+" -> ir.Binary.Op.OP_ADD,
      "-" -> ir.Binary.Op.OP_SUB,
      "++" -> ir.Binary.Op.OP_CONCAT
    )
    val stepType = "umpire.Step"
    val constOps = Map[String, (Long, Long) => Long](
      "+" -> (_ + _),
      "-" -> (_ - _),
      "*" -> (_ * _),
      "<<" -> (_ << _)
    )
    val noneModule = Symbol.requiredModule("scala.None")
    val someModule = Symbol.requiredModule("scala.Some")

    // TASTy records a source path relative to the build that compiled it; the prefix makes it
    // relative to the repository. A tree the lifter builds itself, such as the block left after a
    // `require`, has no span.
    // The prefix is the one of the jar the source came from; one with `%s` in it names the stored
    // file the build-relative path stands in for, such as a fixture materialized for its build.
    def pos(t: Tree): ir.Position = scala.util
      .Try {
        val p = t.pos
        val path = p.sourceFile.path
        val prefix = sourceRoots.getOrElse(path, "")
        ir.Position
          .newBuilder()
          .setFile(if prefix.contains("%s") then prefix.replace("%s", path) else prefix + path)
          .setLine(p.startLine + 1)
          .build()
      }
      .getOrElse(ir.Position.getDefaultInstance)

    def where(t: Tree): String = s"${pos(t).getFile}:${pos(t).getLine}"
    def fail(t: Tree, message: String): Nothing = throw LiftError(where(t), message)

    // ### Types

    def isEnumCase(s: Symbol): Boolean = s.flags.is(Flags.Enum) && s.flags.is(Flags.Case)

    /** The enum a case belongs to: the class whose companion object declares the case. */
    def enumOf(caseSym: Symbol): Symbol = caseSym.owner.companionClass
    def fieldTypes(cls: Symbol): List[(String, TypeRepr)] =
      cls.caseFields.map(f => f.name -> cls.typeRef.memberType(f).widen)
    def isList(sym: Symbol): Boolean = sym == defn.RepeatedParamClass ||
      sym.fullName == "scala.collection.immutable.List" || sym.fullName == "scala.collection.immutable.Seq"
    def isNamed(tpe: TypeRepr, name: String): Boolean =
      tpe.widen.dealias.typeSymbol.fullName == name
    def messageType(inbox: TypeRepr): String =
      inbox.widen.dealias.typeArgs.head.dealias.typeSymbol.fullName
    def named(name: String): ir.TypeRef = ir.TypeRef.newBuilder().setNamed(name).build()

    def typeRef(tpe: TypeRepr, at: Tree, owner: String = ""): ir.TypeRef =
      // An opaque type is read before its alias is resolved: its range is its own.
      val opaque = tpe.widen.typeSymbol
      if opaque.flags.is(Flags.Opaque) then
        val (lo, hi) = opaqueRanges.getOrElse(
          opaque.fullName,
          fail(
            at,
            s"${opaque.fullName} has no range: give it a " +
              "`given Finite[...] = Finite.upTo(...)` beside the type"
          )
        )
        ir.TypeRef.newBuilder().setIntRange(ir.IntRange.newBuilder().setLow(lo).setHigh(hi)).build()
      else
        val t = tpe.dealias.widen
        val sym = t.typeSymbol
        if sym == defn.BooleanClass then
          ir.TypeRef.newBuilder().setBool(ir.Empty.getDefaultInstance).build()
        else if sym == defn.IntClass && owner.isEmpty then
          ir.TypeRef.newBuilder().setInt(ir.Empty.getDefaultInstance).build()
        else if sym == defn.IntClass then
          val (lo, hi) = intRanges.getOrElse(
            owner,
            fail(
              at,
              s"an Int field of $owner has no range: give the state a " +
                "`given Finite[Int] = Finite.upTo(...)` where its Finite is derived"
            )
          )
          ir.TypeRef
            .newBuilder()
            .setIntRange(ir.IntRange.newBuilder().setLow(lo).setHigh(hi))
            .build()
        else if isList(sym) then
          ir.TypeRef.newBuilder().setList(typeRef(t.typeArgs.head, at, owner)).build()
        else if sym.fullName == "scala.Option" then named(optionType(t.typeArgs.head, at))
        else if sym.fullName == "umpire.Inbox" then
          if owner.isEmpty then
            fail(at, "an Inbox is held in a state field; a function reads it from the state")
          val channel = channelFields.getOrElse(
            (owner, messageType(t)),
            fail(
              at,
              s"an Inbox field of $owner names no " +
                "channel: give the state a `given Finite[Inbox[M]] = <channel>.contents` where its Finite is declared"
            )
          )
          ir.TypeRef.newBuilder().setChannel(channelOf(channel, at)).build()
        else if sym.fullName == stepType then named(stepType)
        else
          declareType(sym, at)
          named(sym.fullName)

    def declareType(sym: Symbol, at: Tree): Unit =
      if !types.contains(sym.fullName) && sym != defn.NothingClass then
        types(sym.fullName) = ir.Type.getDefaultInstance // placeholder against recursion
        val b = ir.Type
          .newBuilder()
          .setName(sym.fullName)
          .setPosition(scala.util.Try(pos(sym.tree)).getOrElse(pos(at)))
        def field(n: String, ft: TypeRepr): ir.Field =
          // A list has no bound, so no finite type has one as a field.
          if isList(ft.dealias.widen.typeSymbol) then
            fail(
              scala.util.Try(sym.tree).getOrElse(at),
              s"${sym.fullName}.$n is a list, which has no bound: a state " +
                "holds messages in a channel's Inbox"
            )
          ir.Field.newBuilder().setName(n).setType(typeRef(ft, at, sym.fullName)).build()
        if sym.flags.is(Flags.Enum) then
          val e = ir.Enum.newBuilder()
          for c <- sym.children do
            val cb = ir.Case.newBuilder().setName(c.name)
            if c.isClassDef then for (n, ft) <- fieldTypes(c) do cb.addFields(field(n, ft))
            e.addCases(cb)
          b.setEnum(e)
        else if sym.flags.is(Flags.Case) then
          val r = ir.Record.newBuilder()
          for (n, ft) <- fieldTypes(sym) do r.addFields(field(n, ft))
          b.setRecord(r)
        else
          fail(
            at,
            s"${sym.fullName} is neither an enum nor a case class, so it has no finite catalog"
          )
        types(sym.fullName) = b.build()

    /**
     * An optional value's type: an enum of `None` and `Some(value)`, one per type of value, named
     * after it, so a value keys as `None` or `Some-<key>` as the framework keys it.
     */
    def optionType(arg: TypeRepr, at: Tree): String =
      val value = typeRef(arg, at)
      val argName = value.getRefCase match
        case ir.TypeRef.RefCase.NAMED     => value.getNamed
        case ir.TypeRef.RefCase.BOOL      => "scala.Boolean"
        case ir.TypeRef.RefCase.INT_RANGE => arg.widen.typeSymbol.fullName
        case _                            =>
          fail(
            at,
            s"an optional ${arg.show} has no finite catalog: give its value an enum, a record, a " +
              "Boolean or an opaque type with a range"
          )
      val name = s"scala.Option[$argName]"
      if !types.contains(name) then
        types(name) = ir.Type
          .newBuilder()
          .setName(name)
          .setPosition(pos(at))
          .setEnum(
            ir.Enum
              .newBuilder()
              .addCases(ir.Case.newBuilder().setName("None"))
              .addCases(
                ir.Case
                  .newBuilder()
                  .setName("Some")
                  .addFields(ir.Field.newBuilder().setName("value").setType(value))
              )
          )
          .build()
      name

    /**
     * The value type of an optional value's type: `Option[T]`, or `Some[T]`, also inside a union such
     * as the `Some[T] | None` of a match with both.
     */
    def optionArg(tpe: TypeRepr, at: Tree): TypeRepr =
      def arg(t: TypeRepr): Option[TypeRepr] = t.widen.dealias match
        case AppliedType(o, List(a))
            if o.typeSymbol.fullName == "scala.Option" || o.typeSymbol.fullName == "scala.Some" =>
          Some(a)
        case OrType(l, r) => arg(l).orElse(arg(r))
        case _            => None
      arg(tpe).getOrElse(
        fail(
          at,
          s"None has no type here: ${tpe.widen.show} is not optional, so write `None: Option[T]`"
        )
      )

    // ### Values folded at lift time: names, strings and bounds the declarations reference

    def resolve(t: Term): Term = t match
      case Typed(e, _)        => resolve(e)
      case Inlined(_, Nil, e) => resolve(e)
      case NamedArg(_, e)     => resolve(e)
      case r: Ref
          if !r.symbol.flags.is(Flags.Param) && r.symbol.isValDef && defs.contains(r.symbol) =>
        defs(r.symbol) match
          case ValDef(_, _, Some(rhs)) if !isEnumCase(r.symbol) => resolve(rhs)
          case _                                                => t
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
      case other                                 => fail(t, s"expected a string, got ${other.show}")

    def constInt(t: Term): Long = resolve(t) match
      case Literal(IntConstant(i)) => i.toLong
      // Operators on constants the compiler leaves unfolded, such as `1 << 20`.
      case Apply(Select(a, op), List(b)) if constOps.contains(op) =>
        constOps(op)(constInt(a), constInt(b))
      case other => fail(t, s"expected an integer constant, got ${other.show}")

    /** An argument the default of its parameter supplies. */
    def isDefault(t: Term): Boolean = t match
      case TypeApply(Select(_, n), _) => n.contains("$default$")
      case Select(_, n)               => n.contains("$default$")
      case _                          => false

    // ### Expressions

    def lit(v: ir.Value.Builder, at: Tree): ir.Expr =
      ir.Expr.newBuilder().setPosition(pos(at)).setLiteral(v).build()
    def expr(at: Tree)(f: ir.Expr.Builder => ir.Expr.Builder): ir.Expr =
      f(ir.Expr.newBuilder().setPosition(pos(at))).build()
    def enumLiteral(sym: Symbol, at: Tree): ir.Expr =
      declareType(enumOf(sym), at)
      lit(
        ir.Value
          .newBuilder()
          .setEnum(ir.EnumValue.newBuilder().setType(enumOf(sym).fullName).setCase(sym.name)),
        at
      )
    def list(items: Seq[ir.Expr], at: Tree): ir.Expr =
      expr(at)(_.setList(ir.ListOf.newBuilder().addAllItems(items.asJava)))
    def text(s: String, at: Tree): ir.Expr = lit(ir.Value.newBuilder().setText(s), at)
    def varargs(t: Term): List[Term] = t match
      case Typed(Repeated(items, _), _) => items
      case Repeated(items, _)           => items
      case other                        => List(other)

    def step(
        outcome: ir.Expr,
        state: ir.Expr,
        facts: ir.Expr,
        because: ir.Expr,
        at: Tree
    ): ir.Expr =
      expr(at)(
        _.setConstruct(
          ir.Construct
            .newBuilder()
            .setType(stepType)
            .addAllArgs(List(outcome, state, facts, because).asJava)
        )
      )

    /** The function a reference names, lifting its body on first use. */
    def callee(sym: Symbol, at: Tree): String =
      if lifting(sym.fullName) then
        fail(
          at,
          s"${sym.name} calls itself, directly or through another function: a recursive function has no IR " +
            "form, so write the bounded computation out"
        )
      if !functions.contains(sym.fullName) then
        val d = defs.get(sym) match
          case Some(d: DefDef) => d
          case _ => fail(at, s"${sym.fullName} is not a function of the lifted sources")
        functions(sym.fullName) = ir.Function.getDefaultInstance
        lifting += sym.fullName
        functions(sym.fullName) =
          function(sym.fullName, d.termParamss.flatMap(_.params), d.rhs.get, d)
        lifting -= sym.fullName
      sym.fullName

    def function(name: String, params: List[ValDef], body: Term, at: Tree): ir.Function =
      val b = ir.Function.newBuilder().setName(name).setPosition(pos(at))
      for p <- params do
        b.addParams(ir.Param.newBuilder().setName(p.name).setType(typeRef(p.tpt.tpe, p)))
      val (requires, rest) = stripContracts(body)
      requires.foreach(r => b.setRequires(lift(r)))
      b.setBody(lift(rest)).build()

    /**
     * `{ require(p); body }.ensuring(q)` is `body` under precondition `p`: the contracts are what
     * Stainless proves, and the interpreter evaluates the body.
     */
    def stripContracts(t: Term): (Option[Term], Term) = t match
      case Apply(Select(Apply(TypeApply(e, _), List(body)), "ensuring"), _)
          if e.symbol.name == "Ensuring" =>
        stripContracts(body)
      case Block(Apply(r, List(cond)) :: rest, e) if r.symbol.name == "require" =>
        (Some(cond), if rest.isEmpty then e else Block(rest, e))
      case other => (None, other)

    /**
     * Whether a function's own scope owns the name: a parameter, a local `val`, or a name a pattern
     * binds, also inside a local `val`'s right-hand side.
     */
    def local(sym: Symbol): Boolean =
      val owner = sym.maybeOwner
      !owner.isNoSymbol && (owner.isDefDef || (owner.isValDef && local(owner)))

    /** Whether named arguments out of parameter order arrived as a block of synthetic vals. */
    def synthetic(stats: List[Statement]): Boolean =
      stats.nonEmpty && stats.forall { case v: ValDef => v.name.contains("$"); case _ => false }

    def lift(t: Term, expected: Option[TypeRepr] = None): ir.Expr = t match
      // The arguments a varargs parameter collects: the list they make.
      case Typed(Repeated(items, elem), _) => list(items.map(i => lift(i, Some(elem.tpe))), t)
      case Typed(e, tpt)                   => lift(e, expected.orElse(Some(tpt.tpe)))
      case Inlined(_, Nil, e)              => lift(e, expected)
      case Block(Nil, e)                   => lift(e, expected)
      case NamedArg(_, e)                  => lift(e, expected)
      case Literal(BooleanConstant(b))     => lit(ir.Value.newBuilder().setBool(b), t)
      case Literal(IntConstant(i))         => lit(ir.Value.newBuilder().setInt(i), t)
      case Literal(StringConstant(s))      => text(s, t)

      // `copy` keeps a field its argument is the default getter for, and replaces the others. Named
      // arguments out of field order arrive as a block of synthetic vals, which are substituted back.
      // A constructor's or a function's named arguments arrive the same way.
      case Block(stats, _: Apply) if synthetic(stats) =>
        arguments(t) match
          case Apply(Select(base, "copy"), args) => copyOf(base, args, t)
          case call                              => lift(call, expected)
      case Apply(Select(base, "copy"), args) => copyOf(base, args, t)

      case Block((v: ValDef) :: _, _) if v.symbol.flags.is(Flags.Mutable) =>
        fail(
          v,
          s"`var ${v.name}` has no IR form: a step function is one pure expression, so write the value it ends with"
        )
      case While(_, _) => fail(t, "a loop has no IR form: a step function is one pure expression")
      case Block(ValDef(name, _, Some(rhs)) :: rest, e) =>
        expr(t)(
          _.setLet(
            ir.Let
              .newBuilder()
              .setName(name)
              .setValue(lift(rhs))
              .setBody(lift(Block(rest, e), expected))
          )
        )

      case If(c, a, b) =>
        val branch = expected.orElse(Some(t.tpe))
        expr(t)(
          _.setIf(
            ir.If
              .newBuilder()
              .setCondition(lift(c))
              .setThen(lift(a, branch))
              .setElse(lift(b, branch))
          )
        )

      case Match(scrutinee, cases) =>
        val m = ir.Match.newBuilder().setScrutinee(lift(scrutinee))
        for CaseDef(p, guard, body) <- cases do
          val c = ir.MatchCase
            .newBuilder()
            .setPattern(pattern(p, scrutinee.tpe))
            .setBody(lift(body, expected.orElse(Some(t.tpe))))
          guard.foreach(g => c.setGuard(lift(g)))
          m.addCases(c)
        expr(t)(_.setMatch(m))

      // The prelude's constructors, which both sides of the kernel supply.
      case Apply(TypeApply(Ident("step"), _), List(o, s, f))
          if t.symbol.fullName.startsWith("umpire.prelude") =>
        step(lift(o), lift(s), lift(f), text("", t), t)
      case Apply(TypeApply(Ident("one" | "facts1"), _), List(x))
          if t.symbol.fullName.startsWith("umpire.prelude") =>
        list(List(lift(x)), t)
      case TypeApply(Ident("none" | "facts0"), _)
          if t.symbol.fullName.startsWith("umpire.prelude") =>
        list(Nil, t)
      case Ident("Nil")                                                        => list(Nil, t)
      case Apply(TypeApply(Select(Ident("List"), "apply"), elem), List(items)) =>
        list(varargs(items).map(i => lift(i, elem.headOption.map(_.tpe))), t)

      // The framework's step record, with its default facts and explanation.
      case Apply(TypeApply(Select(companion, "apply"), _), args)
          if companion.tpe.typeSymbol.companionClass.fullName == stepType =>
        val all = args.map {
          case TypeApply(Select(_, name), _) if name.endsWith("$default$3") => list(Nil, t)
          case TypeApply(Select(_, name), _) if name.endsWith("$default$4") => text("", t)
          case a                                                            => lift(a)
        }
        step(all(0), all(1), all(2), all(3), t)

      // An optional value. `None` takes its type from where it is used.
      case r: Ref if r.symbol == noneModule =>
        val name = optionType(optionArg(expected.getOrElse(r.tpe), t), t)
        lit(
          ir.Value.newBuilder().setEnum(ir.EnumValue.newBuilder().setType(name).setCase("None")),
          t
        )
      case Apply(TypeApply(Select(some, "apply"), List(arg)), List(x))
          if some.symbol == someModule =>
        val c = ir.Construct
          .newBuilder()
          .setType(optionType(arg.tpe, t))
          .setCase("Some")
          .addArgs(lift(x, Some(arg.tpe)))
        expr(t)(_.setConstruct(c))

      // `a.min(b)` and `a.max(b)` of integers: the smaller or the larger, as a conditional.
      case Apply(Select(Apply(Ident("intWrapper"), List(a)), op @ ("min" | "max")), List(b)) =>
        val keep = if op == "min" then ir.Binary.Op.OP_LE else ir.Binary.Op.OP_GE
        val (l, r) = (lift(a), lift(b))
        val c = expr(t)(_.setBinary(ir.Binary.newBuilder().setOp(keep).setLeft(l).setRight(r)))
        expr(t)(_.setIf(ir.If.newBuilder().setCondition(c).setThen(l).setElse(r)))
      // A varargs parameter read as the list it is.
      case Select(recv, "toList") if isList(recv.tpe.widen.dealias.typeSymbol) => lift(recv)

      // What a channel holds: its operations, and a channel holding nothing.
      case Select(channel, "empty") if isNamed(channel.tpe, "umpire.Channel") =>
        lit(ir.Value.newBuilder().setList(ir.ListValue.getDefaultInstance), t)
      case Apply(Select(recv, "send"), List(m)) if isNamed(recv.tpe, "umpire.Inbox") =>
        inbox(ir.Inbox.Op.OP_SEND, recv, Some(m), t)
      case Select(recv, "isEmpty") if isNamed(recv.tpe, "umpire.Inbox") =>
        inbox(ir.Inbox.Op.OP_IS_EMPTY, recv, None, t)
      case Select(recv, "isFull") if isNamed(recv.tpe, "umpire.Inbox") =>
        inbox(ir.Inbox.Op.OP_IS_FULL, recv, None, t)
      // A declared hole, where a step reaches it.
      case Select(h, "reached") if isNamed(h.tpe, "umpire.Hole") =>
        expr(t)(_.setHole(holeOf(resolveSymbol(h), t)))

      case Apply(Select(recv, "contains"), List(x)) =>
        expr(t)(
          _.setBinary(
            ir.Binary
              .newBuilder()
              .setOp(ir.Binary.Op.OP_CONTAINS)
              .setLeft(lift(x))
              .setRight(lift(recv))
          )
        )
      case Apply(TypeApply(Select(recv, "contains"), _), List(x)) =>
        expr(t)(
          _.setBinary(
            ir.Binary
              .newBuilder()
              .setOp(ir.Binary.Op.OP_CONTAINS)
              .setLeft(lift(x))
              .setRight(lift(recv))
          )
        )

      case Select(recv, "unary_!") =>
        expr(t)(_.setUnary(ir.Unary.newBuilder().setOp(ir.Unary.Op.OP_NOT).setOperand(lift(recv))))
      case Apply(Select(l, op), List(r)) if binaryOps.contains(op) =>
        expr(t)(
          _.setBinary(
            ir.Binary
              .newBuilder()
              .setOp(binaryOps(op))
              .setLeft(lift(l, Some(r.tpe)))
              .setRight(lift(r, Some(l.tpe)))
          )
        )
      case Apply(TypeApply(Select(l, "++"), _), List(r)) =>
        expr(t)(
          _.setBinary(
            ir.Binary.newBuilder().setOp(ir.Binary.Op.OP_CONCAT).setLeft(lift(l)).setRight(lift(r))
          )
        )

      // A record, or an enum case with fields, built from its constructor.
      case Apply(Select(companion, "apply"), args)
          if companion.tpe.typeSymbol.companionClass.flags.is(Flags.Case) =>
        val cls = companion.tpe.typeSymbol.companionClass
        val fields = fieldTypes(cls).map(_._2)
        val c = ir.Construct
          .newBuilder()
          .addAllArgs(args.zipWithIndex.map((a, i) => lift(a, fields.lift(i))).asJava)
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
        val params = defs(fn.symbol) match
          case d: DefDef => d.termParamss.flatMap(_.params).map(_.tpt.tpe)
          case _         => Nil
        expr(t)(
          _.setCall(
            ir.Call
              .newBuilder()
              .setFunction(name)
              .addAllArgs(args.zipWithIndex.map((a, i) => lift(a, params.lift(i))).asJava)
          )
        )

      case r: Ref if isEnumCase(r.symbol) => enumLiteral(r.symbol, t)
      // A parameter, a local `val` or a pattern-bound name: every name a function's own scope owns.
      case r: Ref if local(r.symbol) => expr(t)(_.setVar(r.symbol.name))
      // A string read off a declared value, such as an observation's name, is a constant.
      case Select(recv, _) if t.tpe.widen <:< defn.StringClass.typeRef && !local(recv.symbol) =>
        text(constString(t), t)
      case Select(recv, field)
          if recv.tpe.widen.dealias.typeSymbol.caseFields.exists(_.name == field) =>
        expr(t)(_.setField(ir.FieldAccess.newBuilder().setBase(lift(recv)).setField(field)))
      // A value declared elsewhere: its definition, lifted in place.
      case r: Ref if r.symbol.isValDef && defs.contains(r.symbol) =>
        defs(r.symbol) match
          case ValDef(_, _, Some(rhs)) => lift(rhs, expected)
          case _                       => fail(t, s"${r.symbol.fullName} has no definition to lift")

      case lambda @ Block(
            List(DefDef("$anonfun", List(TermParamClause(params)), _, Some(body))),
            _: Closure
          ) =>
        val l = ir.Lambda.newBuilder().setBody(lift(body))
        for p <- params do
          l.addParams(ir.Param.newBuilder().setName(p.name).setType(typeRef(p.tpt.tpe, p)))
        expr(lambda)(_.setLambda(l))

      case other => fail(other, s"outside the liftable subset: ${other.show}")

    def copyOf(base: Term, args: List[Term], at: Tree): ir.Expr =
      val c = ir.Copy.newBuilder().setBase(lift(base))
      val fields = fieldTypes(base.tpe.widen.typeSymbol)
      for (arg, i) <- args.zipWithIndex do
        arg match
          case NamedArg(name, value) =>
            c.addUpdates(
              ir.NamedExpr
                .newBuilder()
                .setName(name)
                .setValue(lift(value, fields.find(_._1 == name).map(_._2)))
            )
          case Select(_, getter) if getter.startsWith("copy$default$")               => ()
          case TypeApply(Select(_, getter), _) if getter.startsWith("copy$default$") => ()
          case value                                                                 =>
            c.addUpdates(
              ir.NamedExpr
                .newBuilder()
                .setName(fields(i)._1)
                .setValue(lift(value, Some(fields(i)._2)))
            )
      expr(at)(_.setCopy(c))

    def pattern(p: Tree, scrutinee: TypeRepr): ir.Pattern = p match
      case Wildcard() => ir.Pattern.newBuilder().setWildcard(ir.Empty.getDefaultInstance).build()
      case Bind(name, inner) =>
        ir.Pattern
          .newBuilder()
          .setBind(ir.Bind.newBuilder().setName(name).setPattern(pattern(inner, scrutinee)))
          .build()
      case Alternatives(ps) =>
        ir.Pattern
          .newBuilder()
          .setAlternatives(
            ir.Alternatives.newBuilder().addAllPatterns(ps.map(pattern(_, scrutinee)).asJava)
          )
          .build()
      case Literal(BooleanConstant(b)) =>
        ir.Pattern.newBuilder().setLiteral(ir.Value.newBuilder().setBool(b)).build()
      case r: Ref if r.symbol == noneModule =>
        ir.Pattern
          .newBuilder()
          .setLiteral(
            ir.Value
              .newBuilder()
              .setEnum(
                ir.EnumValue
                  .newBuilder()
                  .setType(optionType(optionArg(scrutinee, p), p))
                  .setCase("None")
              )
          )
          .build()
      case r: Ref if isEnumCase(r.symbol) =>
        declareType(enumOf(r.symbol), p)
        ir.Pattern
          .newBuilder()
          .setLiteral(
            ir.Value
              .newBuilder()
              .setEnum(
                ir.EnumValue.newBuilder().setType(enumOf(r.symbol).fullName).setCase(r.symbol.name)
              )
          )
          .build()
      case Unapply(TypeApply(fun, List(arg)), _, List(inner))
          if fun.symbol.owner.companionClass.fullName == "scala.Some" =>
        ir.Pattern
          .newBuilder()
          .setCase(
            ir.CasePattern
              .newBuilder()
              .setType(optionType(arg.tpe, p))
              .setCase("Some")
              .addFields(pattern(inner, arg.tpe))
          )
          .build()
      case Unapply(fun, _, fields) =>
        val cls = fun.symbol.owner.companionClass
        if !cls.flags.is(Flags.Enum) then
          fail(p, s"only enum cases are matched by constructor, not ${cls.fullName}")
        declareType(enumOf(cls), p)
        val types = fieldTypes(cls).map(_._2)
        ir.Pattern
          .newBuilder()
          .setCase(
            ir.CasePattern
              .newBuilder()
              .setType(enumOf(cls).fullName)
              .setCase(cls.name)
              .addAllFields(
                fields.zipWithIndex
                  .map((f, i) => pattern(f, types.lift(i).getOrElse(scrutinee)))
                  .asJava
              )
          )
          .build()
      case TypedOrTest(inner, tpt) => pattern(inner, tpt.tpe)
      case other                   => fail(other, s"outside the liftable patterns: ${other.show}")

    // ### Declarations: actions and machines, from the framework calls that declare them

    /** A call's function name and its argument lists, outermost last, through type applications. */
    def call(t: Term): Option[(String, List[List[Term]])] = t match
      case Apply(fn, args)  => call(fn).map((n, as) => (n, as :+ args))
      case TypeApply(fn, _) => call(fn)
      case Ident(n)         => Some(n -> Nil)
      case Select(_, n)     => Some(n -> Nil)
      case _                => None

    def action(ref: Term): String = ref match
      // A channel's delivery or loss: an action its declaration implies.
      case Select(channel, op @ ("deliver" | "lose")) if isNamed(channel.tpe, "umpire.Channel") =>
        channelAction(resolveSymbol(channel), op, ref)
      case _ =>
        val sym = resolveSymbol(ref)
        if !actions.contains(sym.fullName) then
          val d = defs.get(sym) match
            case Some(v: ValDef) => v
            case _ => fail(ref, s"${sym.fullName} is not an action declared in the lifted sources")
          actions(sym.fullName) = actionOf(sym.fullName, d.rhs.get)
        sym.fullName

    /** A stable path, `a` or `a.b.c`: what an alias is a name for. */
    def path(t: Term): Boolean = t match
      case Ident(_)     => true
      case Select(q, _) => path(q)
      case _            => false

    /** The symbol a reference finally names, through aliases such as `val workerStop = worker.workerStop`. */
    def resolveSymbol(ref: Term): Symbol = ref match
      case r: Ref =>
        defs.get(r.symbol) match
          case Some(ValDef(_, _, Some(rhs: Ref))) if path(rhs) => resolveSymbol(rhs)
          case _                                               => r.symbol
      case Typed(e, _) => resolveSymbol(e)
      case other       => fail(other, "expected a reference to a declared value")

    def actionOf(id: String, chain: Term): ir.Action =
      val b = ir.Action.newBuilder().setId(id).setPosition(pos(chain))
      def walk(t: Term): Unit = t match
        case Apply(Ident("action"), List(name, party)) =>
          b.setName(constString(name)).setParty(constString(party))
        case Apply(Ident("timer"), List(name)) =>
          b.setName(constString(name)).setParty("system").setTimer(true)
        case Apply(Ident("internal"), List(name)) =>
          b.setName(constString(name)).setParty("system").setInternal(true)
        case Apply(Select(inner, "on"), List(e))      => walk(inner); b.setOn(constString(e))
        case Apply(Select(inner, "creates"), List(e)) => walk(inner); b.setCreates(constString(e))
        case Apply(Select(inner, "results"), List(n)) => walk(inner); b.setResults(constString(n))
        case Apply(Select(inner, "schema"), List(names)) =>
          walk(inner); varargs(names).foreach(n => b.addSchemas(constString(n)))
        case Apply(Apply(TypeApply(Select(inner, "input"), List(tpt)), List(name)), _) =>
          walk(inner);
          b.addInputs(ir.Param.newBuilder().setName(constString(name)).setType(typeRef(tpt.tpe, t)))
        case Apply(Apply(TypeApply(Ident("example"), _), List(inner)), List(value, example)) =>
          walk(inner)
          b.addExamples(
            ir.Example.newBuilder().setValue(literalValue(value)).setExample(constString(example))
          )
        case other => fail(other, s"not a part of an action declaration: ${other.show}")
      walk(chain)
      b.build()

    /** A constant value, for an example: an enum case, with constant fields. */
    def literalValue(t: Term): ir.Value = resolve(t) match
      case Literal(BooleanConstant(v))    => ir.Value.newBuilder().setBool(v).build()
      case Literal(IntConstant(v))        => ir.Value.newBuilder().setInt(v).build()
      case r: Ref if isEnumCase(r.symbol) =>
        ir.Value
          .newBuilder()
          .setEnum(
            ir.EnumValue.newBuilder().setType(enumOf(r.symbol).fullName).setCase(r.symbol.name)
          )
          .build()
      case Apply(Select(companion, "apply"), args)
          if companion.tpe.typeSymbol.companionClass.flags.is(Flags.Enum) =>
        val cls = companion.tpe.typeSymbol.companionClass
        ir.Value
          .newBuilder()
          .setEnum(
            ir.EnumValue
              .newBuilder()
              .setType(enumOf(cls).fullName)
              .setCase(cls.name)
              .addAllFields(args.map(literalValue).asJava)
          )
          .build()
      case other => fail(other, s"an example is a constant value, not ${other.show}")

    /** A step binding's function: the one an eta-expanded lambda forwards to, or the lambda itself. */
    def stepFunction(fn: Term, machine: String, actionName: String): String = fn match
      // A lambda written as a block, `holds { s => ... }`.
      case Block(Nil, e)      => stepFunction(e, machine, actionName)
      case Typed(e, _)        => stepFunction(e, machine, actionName)
      case Inlined(_, Nil, e) => stepFunction(e, machine, actionName)
      case Block(
            List(DefDef("$anonfun", List(TermParamClause(params)), _, Some(Apply(target, args)))),
            _: Closure
          ) if args.map(_.symbol) == params.map(_.symbol) && isFunction(target.symbol) =>
        callee(target.symbol, fn)
      case Block(
            List(DefDef("$anonfun", List(TermParamClause(params)), _, Some(body))),
            _: Closure
          ) =>
        val name = s"$machine.$actionName"
        functions(name) = function(name, params, body, fn)
        name
      case other => fail(other, "a step binds a function")

    def machine(rhs: Term): ir.Machine =
      rhs match
        // `source.restrict(family, name)(keep*)`: the source's steps for the kept actions only.
        case Apply(Apply(Select(source, "restrict"), List(family, newName)), List(keep)) =>
          val src = machineOf(resolveSymbol(source), source)
          val kept = varargs(keep).map(action).toSet
          ir.Machine
            .newBuilder(src)
            .setFamily(constString(family))
            .setName(constString(newName))
            .setPosition(pos(rhs))
            .clearSteps()
            .addAllSteps(src.getStepsList.asScala.filter(s => kept(s.getAction)).asJava)
            .clearUnobservable()
            .clearRefines()
            .build()
        case _ =>
          val (s, o, f, family, mname, body) = rhs match
            case Apply(
                  Apply(Apply(TypeApply(Ident("machine"), List(s, o, f)), List(fam, n)), List(ctx)),
                  _
                ) =>
              (s.tpe, o.tpe, f.tpe, fam, n, ctx)
            case other =>
              fail(other, "a machine is declared by `machine[S, O, F](family, name) { ... }`")
          val b = ir.Machine
            .newBuilder()
            .setFamily(constString(family))
            .setName(constString(mname))
            .setPosition(pos(rhs))
            .setStateType(typeRef(s, rhs).getNamed)
            .setOutcomeType(typeRef(o, rhs).getNamed)
          if f.dealias.typeSymbol != defn.NothingClass then b.setFactType(typeRef(f, rhs).getNamed)
          val stats = body match
            case Block(List(DefDef("$anonfun", _, _, Some(Block(stats, last)))), _: Closure) =>
              stats :+ last
            case other => fail(other, "a machine's body is a block of declarations")
          val visible = mutable.ArrayBuffer.empty[String]
          val visibleOutcomes = mutable.ArrayBuffer.empty[String]
          for stat <- stats do
            val decl = stat match
              case term: Term => call(term)
              case _          => None
            decl match
              case Some(("forEntity", List(List(e), _)))  => b.setEntity(constString(e))
              case Some(("starts", List(_, List(items)))) =>
                varargs(items).foreach(i => b.addStarts(lift(i)))
              case Some(("ends", List(_, List(p))))          => b.setEnds(lift(p))
              case Some(("unobservable", List(List(ts), _))) =>
                varargs(ts).foreach(a => b.addUnobservable(action(a)))
              case Some(("evidence", List(_, List(fn)))) =>
                val evidenceName = s"${b.getName}.evidence"
                fn match
                  case Block(
                        List(DefDef("$anonfun", List(TermParamClause(params)), _, Some(body))),
                        _: Closure
                      ) =>
                    functions(evidenceName) = function(evidenceName, params, body, fn)
                  case other => fail(other, "evidence is a function of the fact")
                b.setEvidence(evidenceName)
              case Some(("refines", List(_, List(product), List(map)))) =>
                val productName = machineOf(resolveSymbol(product), product).getName
                b.setRefines(
                  ir.Refinement
                    .newBuilder()
                    .setProduct(productName)
                    .setMap(stepFunction(map, b.getName, "refines"))
                )
              case Some(("visible", List(_, List(fn)))) =>
                visible += stepFunction(fn, b.getName, "visible")
              case Some(("visibleOutcomes", List(_, List(fn)))) =>
                visibleOutcomes += stepFunction(fn, b.getName, "visibleOutcomes")
              case Some(("monitors", List(_, List(ms)))) =>
                varargs(ms).foreach(m => b.addMonitors(monitorOf(resolveSymbol(m), m)))
              case Some(("assumes", List(List(as), _))) =>
                varargs(as).foreach(a => b.addAssumes(assumptionOf(resolveSymbol(a), a)))
              case Some(("steps", List(_, List(bindings)))) =>
                for binding <- varargs(bindings) do
                  binding match
                    case Apply(TypeApply(Apply(TypeApply(Ident("~>"), _), List(a)), _), List(fn)) =>
                      val id = action(a)
                      b.addSteps(
                        ir.StepBinding
                          .newBuilder()
                          .setAction(id)
                          .setPosition(pos(binding))
                          .setFunction(stepFunction(fn, b.getName, actions(id).getName))
                      )
                    case Apply(TypeApply(Apply(Ident("~>"), List(a)), _), List(fn)) =>
                      val id = action(a)
                      b.addSteps(
                        ir.StepBinding
                          .newBuilder()
                          .setAction(id)
                          .setPosition(pos(binding))
                          .setFunction(stepFunction(fn, b.getName, actions(id).getName))
                      )
                    case other => fail(other, s"a step is `action ~> function`, not ${other.show}")
              case _ =>
                stat match
                  case Literal(UnitConstant()) => () // the block's trailing unit
                  case _ => fail(stat, s"not a machine declaration: ${stat.show}")
          for v <- visible do
            if !b.hasRefines then
              fail(
                rhs,
                s"${b.getName} names the facts a refined machine sees, and declares no refinement"
              )
            b.setRefines(b.getRefines.toBuilder.setVisible(v))
          for v <- visibleOutcomes do
            if !b.hasRefines then
              fail(
                rhs,
                s"${b.getName} names the outcomes a refined machine sees, and declares no refinement"
              )
            b.setRefines(b.getRefines.toBuilder.setVisibleOutcomes(v))
          checkChannels(b, s.dealias.typeSymbol.fullName, rhs)
          b.build()

    /**
     * A machine binds each channel's actions only for a channel its state holds, and says what losing
     * a message of a lossy one does.
     */
    def checkChannels(b: ir.Machine.Builder, state: String, at: Tree): Unit =
      val fields = types
        .get(state)
        .toList
        .flatMap(t =>
          t.getRecord.getFieldsList.asScala ++
            t.getEnum.getCasesList.asScala.flatMap(_.getFieldsList.asScala)
        )
        .filter(_.getType.hasChannel)
      for (c, fs) <- fields.groupBy(_.getType.getChannel).toList.sortBy(_._1) if fs.size > 1 do
        fail(
          at,
          s"$state holds channel ${channels(c).getName} in ${fs.map(_.getName).mkString(" and ")}; a state " +
            "holds a channel in one field"
        )
      val held = fields.map(_.getType.getChannel).toSet
      val bound = b.getStepsList.asScala.map(s => actions(s.getAction))
      for a <- bound; c <- Seq(a.getDelivers, a.getLoses) if c.nonEmpty && !held(c) do
        fail(
          at,
          s"${b.getName} binds ${a.getName}, and its state holds no ${channels(c).getName} channel"
        )
      for c <- held.toList.sorted if channels(c).getLossy && !bound.exists(_.getLoses == c) do
        fail(
          at,
          s"${b.getName} holds the lossy channel ${channels(c).getName} and binds no ${channels(c).getName}Loss " +
            "step: a lossy channel's loss is a step whose meaning the machine says"
        )
      for c <- held.toList.sorted if !channels(c).getLossy && bound.exists(_.getLoses == c) do
        fail(
          at,
          s"${b.getName} binds ${channels(c).getName}Loss, and ${channels(c).getName} is reliable"
        )

    /** A lifted machine, by the name the IR gives it. */
    def machineNamed(name: String): Option[ir.Machine] = machines.values.find(_.getName == name)

    def machineOf(sym: Symbol, at: Tree): ir.Machine =
      machines.getOrElseUpdate(
        sym.fullName,
        defs.get(sym) match
          case Some(ValDef(_, _, Some(rhs))) => machine(rhs)
          case _ => fail(at, s"${sym.fullName} is not a machine of the lifted sources")
      )

    /** A declaration value of the lifted sources: its definition. */
    def valDef(sym: Symbol, at: Tree, kind: String): ValDef = defs.get(sym) match
      case Some(v @ ValDef(_, _, Some(_))) => v
      case _ => fail(at, s"${sym.fullName} is not $kind declared by a val of the lifted sources")

    /**
     * A call whose named arguments out of parameter order arrived as a block of synthetic vals, with
     * the vals substituted back into it.
     */
    def arguments(t: Term): Term = t match
      case Block(stats, call: Apply) if synthetic(stats) =>
        val bound = stats.collect { case v: ValDef => v.symbol -> v.rhs.get }.toMap
        def unbind(a: Term): Term = a match
          case r: Ref if bound.contains(r.symbol) => bound(r.symbol)
          case NamedArg(n, v)                     => NamedArg(n, unbind(v))
          case Apply(fn, args)                    => Apply.copy(a)(unbind(fn), args.map(unbind))
          case TypeApply(fn, targs)               => TypeApply.copy(a)(unbind(fn), targs)
          case _                                  => a
        unbind(call)
      case _ => t

    // ### Channels, monitors, assumptions and holes

    /** A channel, from its `channel[M](name, capacity, order, loss, duplicates)` declaration. */
    def channelOf(sym: Symbol, at: Tree): String =
      if !channels.contains(sym.fullName) then
        val d = valDef(sym, at, "a channel")
        val (message, name, capacity, order, loss, duplicates, finite) = arguments(d.rhs.get) match
          case Apply(
                Apply(TypeApply(Ident("channel"), List(m)), List(n, c, o, l, dups)),
                List(f)
              ) =>
            (m, n, c, o, l, dups, f)
          case other =>
            fail(
              other,
              "a channel is declared by `channel[M](name, capacity, order, loss, duplicates)`"
            )
        val (cap, dup) =
          (constInt(capacity), if isDefault(duplicates) then 0L else constInt(duplicates))
        if cap < 1 then
          fail(
            d,
            s"channel ${constString(name)} holds at most $cap messages; a channel holds at least one"
          )
        if dup < 0 then
          fail(
            d,
            s"channel ${constString(name)} delivers a message $dup more times than once; no fewer"
          )
        val n = constString(name)

        /** The case of a framework enum a policy argument names; anything computed is refused. */
        def policy[V](arg: Term, enumName: String, cases: Map[String, V]): V = resolve(arg) match
          case r: Ref if isEnumCase(r.symbol) && enumOf(r.symbol).fullName == s"umpire.$enumName" =>
            cases(r.symbol.name)
          case other =>
            fail(
              d,
              s"channel $n's ${enumName.toLowerCase} is ${other.show}, not a case of $enumName: a channel " +
                s"names its ${enumName.toLowerCase} as ${cases.keys.toList.sorted.map(c => s"$enumName.$c").mkString(" or ")}"
            )
        channels(sym.fullName) = ir.Channel
          .newBuilder()
          .setId(sym.fullName)
          .setName(n)
          .setPosition(pos(d))
          .setMessage(messageRef(message.tpe, finite, d, n))
          .setCapacity(cap.toInt)
          .setOrder(
            policy(
              order,
              "Order",
              Map(
                "fifo" -> ir.Channel.Order.ORDER_FIFO,
                "unordered" -> ir.Channel.Order.ORDER_UNORDERED
              )
            )
          )
          .setLossy(policy(loss, "Loss", Map("reliable" -> false, "lossy" -> true)))
          .setDuplicates(dup.toInt)
          .build()
      sym.fullName

    /**
     * A channel's message type, as its finite catalog: a named type, the Booleans, an opaque type's
     * own range, or, for Int, the range the channel's `Finite.upTo(n)` declares. The IR has no catalog
     * for any other Int or for a list, so those are refused at the declaration.
     */
    def messageRef(message: TypeRepr, finite: Term, at: Tree, channel: String): ir.TypeRef =
      val t = message.dealias.widen
      if !message.widen.typeSymbol.flags.is(Flags.Opaque) && t.typeSymbol == defn.IntClass then
        resolve(finite) match
          case Apply(upTo @ Select(_, "upTo"), List(bound))
              if upTo.symbol.owner.fullName.startsWith("umpire.Finite") =>
            ir.TypeRef
              .newBuilder()
              .setIntRange(ir.IntRange.newBuilder().setLow(0).setHigh(constInt(bound)))
              .build()
          case other =>
            fail(
              at,
              s"channel $channel's messages are Int, and their catalog ${other.show} is not `Finite.upTo(n)`: " +
                "the IR carries an Int only as a range from 0, so declare the channel with one or give its messages " +
                "an opaque type with a range"
            )
      else if isList(t.typeSymbol) then
        fail(
          at,
          s"channel $channel's messages are lists, which have no finite catalog in the IR: give them an " +
            "enum or a record"
        )
      else typeRef(message, at)

    /** A channel's delivery or loss of one message, the action's one input. */
    def channelAction(sym: Symbol, op: String, at: Tree): String =
      val channel = channels(channelOf(sym, at))
      val id = s"${channel.getId}.$op"
      if !actions.contains(id) then
        val b = ir.Action
          .newBuilder()
          .setId(id)
          .setPosition(channel.getPosition)
          .setParty("system")
          .setInternal(true)
          .addInputs(ir.Param.newBuilder().setName("message").setType(channel.getMessage))
        if op == "deliver" then b.setName(s"${channel.getName}Delivery").setDelivers(channel.getId)
        else b.setName(s"${channel.getName}Loss").setLoses(channel.getId)
        actions(id) = b.build()
      id

    /**
     * The channel an `Inbox` value belongs to: the one the state field it is read from holds, or else
     * the only channel of its message type.
     */
    def inboxChannel(recv: Term): String =
      val message = messageType(recv.tpe)
      val owned = recv match
        case Select(base, _) =>
          channelFields.get((base.tpe.widen.dealias.typeSymbol.fullName, message)).toList
        case _ => Nil
      val candidates = if owned.nonEmpty then owned
      else channelFields.collect { case ((_, m), c) if m == message => c }.toList.distinct
      candidates match
        case List(c) => channelOf(c, recv)
        case Nil     => fail(recv, s"no state holds a channel of $message in an Inbox field")
        case many    =>
          fail(
            recv,
            s"channels ${many.map(_.name).sorted.mkString(", ")} all hold $message; read the " +
              "Inbox from the state field that holds it"
          )

    def inbox(op: ir.Inbox.Op, recv: Term, message: Option[Term], at: Tree): ir.Expr =
      val b = ir.Inbox.newBuilder().setOp(op).setChannel(inboxChannel(recv)).setContents(lift(recv))
      message.foreach(m => b.setMessage(lift(m)))
      expr(at)(_.setInbox(b))

    /**
     * A monitor, from its `monitor[S, O, F, M](name, initial)(next)(violated)` declaration and the
     * evaluation point chained onto it.
     */
    def monitorOf(sym: Symbol, at: Tree): String =
      if !monitors.contains(sym.fullName) then
        val d = valDef(sym, at, "a monitor")
        val id = sym.fullName
        val b = ir.Monitor.newBuilder().setId(id).setPosition(pos(d))
        def walk(t: Term): Unit = t match
          case Select(inner, "readAtEnds") => walk(inner); b.setAtEnds(ir.Empty.getDefaultInstance)
          case Apply(Select(inner, "readAfter"), List(f)) =>
            walk(inner); b.setAfter(stepFunction(f, id, "after"))
          case Apply(
                Apply(
                  Apply(
                    Apply(TypeApply(Ident("monitor"), List(_, _, _, m)), List(name, initial)),
                    List(next)
                  ),
                  List(violated)
                ),
                _
              ) =>
            b.setName(constString(name))
              .setState(typeRef(m.tpe, t))
              .setInitial(lift(initial, Some(m.tpe)))
              .setNext(stepFunction(next, id, "next"))
              .setViolated(stepFunction(violated, id, "violated"))
              .setEveryStep(ir.Empty.getDefaultInstance)
          case other => fail(other, s"not a part of a monitor declaration: ${other.show}")
        walk(d.rhs.get)
        if b.getState.hasList || b.getState.hasInt then
          fail(d, s"monitor ${b.getName}'s state has no finite catalog")
        for other <- monitors.values if other.getName == b.getName do
          fail(
            d,
            s"two monitors are named ${b.getName}: ${other.getId} and $id, and would share one Definition ID"
          )
        monitors(id) = b.build()
      sym.fullName

    /** An assumption, from `assume(name)` and the fairness chained onto it. */
    def assumptionOf(sym: Symbol, at: Tree): String =
      if !assumptions.contains(sym.fullName) then
        val d = valDef(sym, at, "an assumption")
        val b = ir.Assumption.newBuilder().setId(sym.fullName).setPosition(pos(d))
        def walk(t: Term): Unit = t match
          case Apply(Ident("assume"), List(name))     => b.setName(constString(name))
          case Apply(Select(inner, "fair"), List(as)) =>
            walk(inner); varargs(as).foreach(a => b.addFair(action(a)))
          case other => fail(other, s"not a part of an assumption declaration: ${other.show}")
        walk(d.rhs.get)
        assumptions(sym.fullName) = b.build()
      sym.fullName

    /** A hole, from `hole(name)`. */
    def holeOf(sym: Symbol, at: Tree): String =
      if !holes.contains(sym.fullName) then
        val d = valDef(sym, at, "a hole")
        d.rhs.get match
          case Apply(Ident("hole"), List(name)) =>
            holes(sym.fullName) = ir.Hole
              .newBuilder()
              .setId(sym.fullName)
              .setName(constString(name))
              .setPosition(pos(d))
              .build()
          case other => fail(other, "a hole is declared by `hole(name)`")
      sym.fullName

    // ### Realizations: declarations written as data, emitted by name

    /** A term, and what the helper function parameters and local vals it names are bound to. */
    final class Bound(val term: Term, val env: Map[Symbol, Bound])

    /** A value the IR names rather than writes out: a machine, a channel, an action or a class. */
    def namedByIR(tpe: TypeRepr): Boolean =
      Set("umpire.Machine", "umpire.Channel", "umpire.Action", "umpire.Class")(
        tpe.widen.dealias.typeSymbol.fullName
      )

    /** A call's function and its arguments, through every argument list. */
    def applied(t: Term): Option[(Term, List[Term])] = t match
      case Apply(fn, args) =>
        applied(fn).map((f, as) => (f, as ++ args)).orElse(Some(fn -> args))
      case TypeApply(fn, _) => applied(fn).orElse(Some(fn -> Nil))
      case _                => None

    /**
     * What a term is once the names it goes through are followed: a helper function of the lifted
     * sources by its body with its parameters bound, a val by its definition.
     */
    def reduce(b: Bound): Bound = b.term match
      case Typed(e, _)                                => reduce(Bound(e, b.env))
      case Inlined(_, Nil, e)                         => reduce(Bound(e, b.env))
      case NamedArg(_, e)                             => reduce(Bound(e, b.env))
      case Block(stats, _: Apply) if synthetic(stats) =>
        reduce(Bound(arguments(b.term), b.env))
      case Block(stats, e) =>
        val inner = stats.foldLeft(b.env) {
          case (acc, v @ ValDef(_, _, Some(rhs))) =>
            acc + (v.symbol -> Bound(rhs, acc))
          case (_, other) => fail(other, s"not a declaration: ${other.show}")
        }
        reduce(Bound(e, inner))
      case r: Ref if b.env.contains(r.symbol) => reduce(b.env(r.symbol))
      case r: Ref if isFunction(r.symbol)     =>
        defs(r.symbol) match
          case d: DefDef => reduce(Bound(d.rhs.get, Map.empty))
          case _         => b
      case r: Ref
          if !isEnumCase(r.symbol) && !namedByIR(
            r.tpe
          ) && r.symbol.isValDef && defs.contains(
            resolveSymbol(r)
          ) =>
        defs(resolveSymbol(r)) match
          case ValDef(_, _, Some(rhs)) => reduce(Bound(rhs, Map.empty))
          case _                       => b
      case t =>
        applied(t) match
          case Some((fn, args)) if isFunction(fn.symbol) =>
            defs(fn.symbol) match
              case d: DefDef =>
                val params = d.termParamss.flatMap(_.params).map(_.symbol)
                reduce(
                  Bound(d.rhs.get, params.zip(args.map(Bound(_, b.env))).toMap)
                )
              case _ => b
          case _ => b

    def snake(name: String): String =
      name
        .flatMap(c => if c.isUpper then s"_${c.toLower}" else c.toString)
        .stripPrefix("_")

    /**
     * The declaration a reduced term writes: the name of the class or the case it constructs, and
     * its arguments by parameter name. An argument its parameter's default supplies is left out, so
     * the IR leaves that field unset.
     */
    def written(b: Bound): (String, List[(String, Bound)]) =
      def vocabulary(sym: Symbol): Unit =
        if !sym.fullName.startsWith("umpire.realize.") then
          fail(b.term, s"not a realization declaration: ${b.term.show}")
      b.term match
        case r: Ref if isEnumCase(r.symbol) =>
          vocabulary(r.symbol)
          (r.symbol.name, Nil)
        case t =>
          applied(t) match
            case Some((fn, args)) if fn.symbol.name == "apply" =>
              val cls = fn.symbol.owner.companionClass
              vocabulary(cls)
              val params =
                fn.symbol.paramSymss.flatten.filter(_.isTerm).map(_.name)
              (
                cls.name,
                params.zip(args).collect {
                  case (p, a) if !isDefault(a) => p -> Bound(a, b.env)
                }
              )
            case _ => fail(t, s"not a realization declaration: ${t.show}")

    /** A string a declaration names: a constant, or the IR's name of a machine or a channel. */
    def textOfBound(b0: Bound): String =
      val b = reduce(b0)
      b.term match
        case Literal(StringConstant(s))                 => s
        case r: Ref if isNamed(r.tpe, "umpire.Machine") =>
          machineOf(resolveSymbol(r), r).getName
        case r: Ref if isNamed(r.tpe, "umpire.Channel") =>
          channelOf(resolveSymbol(r), r)
        case Apply(
              Select(Apply(Select(sc, "apply"), List(parts)), "s"),
              List(args)
            ) if sc.symbol.fullName == "scala.StringContext" =>
          val texts = varargs(args).map(a => textOfBound(Bound(a, b.env)))
          varargs(parts)
            .map(constString)
            .zipAll(texts, "", "")
            .map(_ + _)
            .mkString
        case Apply(Select(l, "+"), List(r)) =>
          textOfBound(Bound(l, b.env)) + textOfBound(Bound(r, b.env))
        case other => fail(other, s"expected a string, got ${other.show}")

    /** The items of a sequence a declaration writes out. */
    def itemsOf(b0: Bound): List[Bound] =
      val b = reduce(b0)
      b.term match
        case Repeated(items, _) => items.map(Bound(_, b.env))
        case Apply(
              TypeApply(Select(Ident("Vector" | "List" | "Seq"), "apply"), _),
              List(items)
            ) =>
          itemsOf(Bound(items, b.env))
        case TypeApply(Select(Ident("Vector" | "List" | "Seq"), "empty"), _) =>
          Nil
        case Ident("Nil")                                  => Nil
        case Apply(TypeApply(Select(l, "++"), _), List(r)) =>
          itemsOf(Bound(l, b.env)) ++ itemsOf(Bound(r, b.env))
        case Apply(TypeApply(Select(l, ":+"), _), List(x)) =>
          itemsOf(Bound(l, b.env)) :+ Bound(x, b.env)
        case other =>
          fail(other, s"expected a sequence written out, got ${other.show}")

    /** One value of a field: a message of the field's type, a class, or a constant. */
    def valueOf(f: FieldDescriptor, into: Message.Builder, b0: Bound): Object =
      val b = reduce(b0)
      f.getJavaType match
        case FieldDescriptor.JavaType.MESSAGE if f.getMessageType.getName == "ActionClass" =>
          classOf(b.term)
        case FieldDescriptor.JavaType.MESSAGE =>
          val sub = into.newBuilderForField(f)
          declaration(b, sub)
          sub.build()
        case FieldDescriptor.JavaType.STRING  => textOfBound(b)
        case FieldDescriptor.JavaType.BOOLEAN =>
          b.term match
            case Literal(BooleanConstant(v)) => Boolean.box(v)
            case other                       =>
              fail(other, s"expected true or false, got ${other.show}")
        case FieldDescriptor.JavaType.LONG | FieldDescriptor.JavaType.INT =>
          val n = b.term match
            case Literal(LongConstant(v)) => v
            case other                    => constInt(other)
          if f.getJavaType == FieldDescriptor.JavaType.LONG then Long.box(n)
          else Int.box(n.toInt)
        case FieldDescriptor.JavaType.ENUM =>
          b.term match
            case r: Ref if isEnumCase(r.symbol) =>
              val name =
                s"${snake(f.getEnumType.getName)}_${snake(r.symbol.name)}".toUpperCase
              Option(f.getEnumType.findValueByName(name))
                .getOrElse(
                  fail(
                    r,
                    s"${r.symbol.name} is no ${f.getEnumType.getName} of the IR"
                  )
                )
            case other =>
              fail(other, s"expected an enum case, got ${other.show}")
        case other =>
          fail(
            b.term,
            s"the IR field ${f.getName} of kind $other is not written out"
          )

    /** Sets the field a parameter names. An optional argument that is `None` leaves it unset. */
    def fieldOf(into: Message.Builder, f: FieldDescriptor, b: Bound): Unit =
      if f.isRepeated then itemsOf(b).foreach(i => into.addRepeatedField(f, valueOf(f, into, i)))
      else
        reduce(b).term match
          case r: Ref if r.symbol == noneModule => ()
          case Apply(TypeApply(Select(some, "apply"), _), List(x)) if some.symbol == someModule =>
            into.setField(f, valueOf(f, into, Bound(x, reduce(b).env)))
          case _ => into.setField(f, valueOf(f, into, b))

    /**
     * Emits one declaration into the IR message of its kind. A constructor named after a member of
     * one of the message's oneofs writes that member; any other writes the fields its parameters
     * name, and a parameter named after a oneof takes the member its argument writes.
     */
    def declaration(b0: Bound, into: Message.Builder): Unit =
      val b = reduce(b0)
      val d = into.getDescriptorForType
      Option(d.findFieldByName("position"))
        .foreach(into.setField(_, pos(b.term)))
      val (name, args) = written(b)
      def member(n: String): Option[FieldDescriptor] =
        Option(d.findFieldByName(snake(n))).filter(f => Option(f.getContainingOneof).nonEmpty)
      def write(f: FieldDescriptor, as: List[(String, Bound)], at: Term): Unit =
        if f.getJavaType != FieldDescriptor.JavaType.MESSAGE then
          as match
            case List((_, a)) => into.setField(f, valueOf(f, into, a))
            case _            => fail(at, s"${f.getName} takes one value")
        else if f.getMessageType.getFields.isEmpty then
          into.setField(f, into.newBuilderForField(f).build())
        else
          val sub = into.newBuilderForField(f)
          as match
            case List((p, a))
                if Option(sub.getDescriptorForType.findFieldByName(snake(p))).isEmpty =>
              declaration(a, sub)
            case _ =>
              Option(sub.getDescriptorForType.findFieldByName("position"))
                .foreach(sub.setField(_, pos(at)))
              fields(sub, as, at)
          into.setField(f, sub.build())
      def fields(
          m: Message.Builder,
          as: List[(String, Bound)],
          at: Term
      ): Unit =
        val md = m.getDescriptorForType
        for (p, a) <- as do
          Option(md.findFieldByName(snake(p))) match
            case Some(f) => fieldOf(m, f, a)
            case None
                if m.eq(into) && d.getOneofs.asScala.exists(
                  _.getName == snake(p)
                ) =>
              val chosen = reduce(a)
              val (n, inner) = written(chosen)
              write(
                member(n).getOrElse(
                  fail(chosen.term, s"$n is no $p of ${d.getName} in the IR")
                ),
                inner,
                chosen.term
              )
            case None => fail(at, s"${md.getName} has no $p in the IR")
      member(name) match
        case Some(f) => write(f, args, b.term)
        case None    => fields(into, args, b.term)

    def realizationOf(sym: Symbol, at: Tree): ir.Realization =
      realizations.getOrElseUpdate(
        sym.fullName, {
          val r = ir.Realization.newBuilder()
          declaration(
            Bound(valDef(sym, at, "a realization").rhs.get, Map.empty),
            r
          )
          r.setId(sym.fullName).build()
        }
      )

    // ### Compositions

    /** `"field" -> value`. */
    def arrow(t: Term): (Term, Term) = t match
      case Apply(
            TypeApply(Select(Apply(TypeApply(Ident("ArrowAssoc"), _), List(k)), "->"), _),
            List(v)
          ) =>
        (k, v)
      case other => fail(other, s"expected `\"member\" -> value`, not ${other.show}")

    def compositionOf(sym: Symbol, at: Tree): ir.Composition =
      compositions.getOrElseUpdate(
        sym.fullName,
        composition(valDef(sym, at, "a composition").rhs.get)
      )

    /**
     * A composition, from `compose[S](family, name)(members*)` and the syncs, ends and replacements
     * chained onto it.
     */
    def composition(rhs: Term): ir.Composition =
      val b = ir.Composition.newBuilder().setPosition(pos(rhs))
      val replaced = mutable.ArrayBuffer.empty[(String, String)]
      def move(t: Term): ir.SyncMove =
        val (member, a) = arrow(t)
        ir.SyncMove
          .newBuilder()
          .setMember(constString(member))
          .setAction(actions(action(a)).getName)
          .build()
      def walk(t: Term): Unit = t match
        case Apply(Select(inner, "sync"), List(name, first, second)) =>
          walk(inner)
          b.addSyncs(
            ir.Sync
              .newBuilder()
              .setName(constString(name))
              .setFirst(move(first))
              .setSecond(move(second))
          )
        case Apply(Select(inner, "ends"), List(p)) => walk(inner); b.setEnds(lift(p))
        case Apply(Select(inner, "replaces"), List(field, opaque)) =>
          walk(inner)
          replaced += constString(field) -> machineOf(resolveSymbol(opaque), opaque).getName
        case Apply(
              Apply(Apply(TypeApply(Ident("compose"), List(s)), List(family, name)), List(members)),
              _
            ) =>
          b.setFamily(constString(family))
            .setName(constString(name))
            .setStateType(typeRef(s.tpe, t).getNamed)
          for m <- varargs(members) do
            val (field, member) = arrow(m)
            b.addMembers(
              ir.Member
                .newBuilder()
                .setField(constString(field))
                .setMachine(machineOf(resolveSymbol(member), member).getName)
            )
        case other => fail(other, s"not a part of a composition declaration: ${other.show}")
      walk(rhs)
      val fields = b.getMembersList.asScala.map(_.getField)
      for
        s <- b.getSyncsList.asScala; m <- Seq(s.getFirst, s.getSecond)
        if !fields.contains(m.getMember)
      do fail(rhs, s"sync ${s.getName} names ${m.getMember}, which is not a member of ${b.getName}")
      for (field, opaque) <- replaced do
        val i = fields.indexOf(field)
        if i < 0 then fail(rhs, s"$field replaces $opaque, and no member fills $field")
        val member = b.getMembers(i).getMachine
        if !machineNamed(member).exists(_.getRefines.getProduct == opaque) then
          fail(rhs, s"$field replaces $opaque, and its member $member does not refine $opaque")
        b.setMembers(i, b.getMembers(i).toBuilder.setReplaces(opaque))
      b.build()

    // ### Claims: Properties, Scenarios, Queries and progress, folded with their arguments bound

    def modelName(d: Decl, at: Tree): String = d match
      case Decl.Model(name) => name
      case other            => fail(at, s"expected a machine or a composition, got $other")
    def textOf(d: Decl, at: Tree): String = d match
      case Decl.Text(s) => s
      case other        => fail(at, s"expected a string, got $other")
    def claimOf(d: Decl, at: Tree): ir.ClaimRef = d match
      case Decl.Claim(ref) => ref
      case other           => fail(at, s"expected a Property or a Scenario, got $other")
    def claim(machine: String, name: String): ir.ClaimRef =
      ir.ClaimRef.newBuilder().setMachine(machine).setName(name).build()

    /** One class of an action: the action bare, or applied to one value per input. */
    def classOf(t: Term): ir.ActionClass = t match
      case Typed(e, _)                                                    => classOf(e)
      case Apply(Apply(fn, List(a)), values) if fn.symbol.name == "apply" =>
        ir.ActionClass
          .newBuilder()
          .setAction(action(a))
          .addAllInputs(values.map(literalValue).asJava)
          .build()
      case ref => ir.ActionClass.newBuilder().setAction(action(ref)).build()

    /**
     * Keeps the first declaration under a key and refuses a different second one, which would share
     * its Definition ID.
     */
    def register[K, V](
        into: mutable.LinkedHashMap[K, V],
        key: K,
        value: V,
        at: Tree,
        what: String
    ): Unit =
      into.get(key) match
        case Some(existing) if existing != value =>
          fail(at, s"$what is declared twice, and both would share one Definition ID")
        case _ => into(key) = value

    def limitsOf(t: Term): ir.Limits = arguments(t) match
      case Apply(Select(companion, "apply"), List(name, steps, actions, search))
          if companion.tpe.typeSymbol.companionClass.fullName == "umpire.Limits" =>
        val l = ir.Limits
          .newBuilder()
          .setName(constString(name))
          .setSteps(constInt(steps).toInt)
          .setActions(constInt(actions).toInt)
          .setSearch(constInt(search).toInt)
          .build()
        for
          (bound, v) <- Seq(
            "steps" -> l.getSteps,
            "actions" -> l.getActions,
            "search" -> l.getSearch
          ) if v < 0
        do fail(t, s"limits ${l.getName} declare $v $bound; a bound is at least 0")
        l
      case other =>
        fail(
          other,
          s"Limits are declared by `Limits(name, steps, actions, search)`, not ${other.show}"
        )

    def query(
        name: String,
        form: ir.Query.Form,
        p: ir.ClaimRef,
        s: ir.ClaimRef,
        through: Option[String],
        limits: ir.Limits,
        at: Tree
    ): Decl =
      through match
        case Some(m) =>
          if m != s.getMachine then
            fail(
              at,
              s"$name reads its Property through the refinement of $m, and its Scenario runs on ${s.getMachine}"
            )
          val refined = machineNamed(m).map(_.getRefines.getProduct).filter(_.nonEmpty)
          if !refined.contains(p.getMachine) then
            fail(
              at,
              s"$name reads ${p.getName}, a Property of ${p.getMachine}, through the refinement of $m, which " +
                s"refines ${refined.getOrElse("nothing")}"
            )
        case None =>
          if p.getMachine != s.getMachine then
            fail(
              at,
              s"$name pairs ${p.getName}, a Property of ${p.getMachine}, with ${s.getName}, a Scenario of " +
                s"${s.getMachine}, and reads it through no refinement"
            )
      val q = ir.Query
        .newBuilder()
        .setName(name)
        .setPosition(pos(at))
        .setForm(form)
        .setProperty(p)
        .setScenario(s)
        .setThrough(through.nonEmpty)
        .setLimits(limits)
        .build()
      register(queries, name, q, at, s"Query $name")
      Decl.Declared(name)

    /**
     * What a declaration of the lifted sources folds to, with the helper function parameters bound
     * in `env`.
     */
    def fold(t: Term, env: Map[Symbol, Decl]): Decl = t match
      case Typed(e, _)                                => fold(e, env)
      case Inlined(_, Nil, e)                         => fold(e, env)
      case NamedArg(_, e)                             => fold(e, env)
      case Block(stats, _: Apply) if synthetic(stats) => fold(arguments(t), env)
      case Block(stats, e)                            =>
        val inner = stats.foldLeft(env) {
          case (acc, v @ ValDef(_, _, Some(rhs))) => acc + (v.symbol -> fold(rhs, acc))
          case (_, other) => fail(other, s"not a declaration: ${other.show}")
        }
        fold(e, inner)
      case Literal(StringConstant(s))       => Decl.Text(s)
      case r: Ref if env.contains(r.symbol) => env(r.symbol)
      // `s"..."`, with each argument folded to its text.
      case Apply(Select(Apply(Select(sc, "apply"), List(parts)), "s"), List(args))
          if sc.symbol.fullName == "scala.StringContext" =>
        val texts = varargs(args).map(a => textOf(fold(a, env), a))
        Decl.Text(varargs(parts).map(constString).zipAll(texts, "", "").map(_ + _).mkString)
      case Select(m, "name")
          if isNamed(m.tpe, "umpire.Machine") || isNamed(m.tpe, "umpire.Composition") =>
        Decl.Text(modelName(fold(m, env), m))
      case Apply(TypeApply(Select(Ident("Vector" | "List"), "apply"), _), List(items)) =>
        Decl.Items(varargs(items).map(fold(_, env)))

      case Apply(Apply(TypeApply(Ident("property"), _), List(m)), List(name)) =>
        Decl.PropertyOn(modelName(fold(m, env), m), textOf(fold(name, env), name), None)
      case Apply(Select(b, "when"), List(c)) =>
        fold(b, env) match
          case p: Decl.PropertyOn => p.copy(when = Some(Left(classOf(c))))
          case other              => fail(t, s"when restricts a Property, not $other")
      case Apply(Select(b, "whenAction"), List(a)) =>
        fold(b, env) match
          case p: Decl.PropertyOn => p.copy(when = Some(Right(constString(a))))
          case other              => fail(t, s"whenAction restricts a Property, not $other")
      case Apply(Select(b, op @ ("holds" | "holdsAcross")), List(f)) =>
        fold(b, env) match
          case Decl.PropertyOn(m, name, when) =>
            val p = ir.Property
              .newBuilder()
              .setMachine(m)
              .setName(name)
              .setPosition(pos(t))
              .setHolds(stepFunction(f, s"$m.property", name))
              .setTransition(op == "holdsAcross")
            when.foreach(_.fold(p.setWhenClass, p.setWhenAction))
            register(properties, (m, name), p.build(), t, s"Property $name of $m")
            Decl.Claim(claim(m, name))
          case other => fail(t, s"$op finishes a Property, not $other")

      case Apply(Apply(TypeApply(Ident("scenario"), _), List(m)), List(name)) =>
        Decl.ScenarioOn(modelName(fold(m, env), m), textOf(fold(name, env), name), None)
      case Apply(Select(b, "starts"), List(s)) =>
        fold(b, env) match
          case sc: Decl.ScenarioOn => sc.copy(start = Some(lift(s)))
          case other               => fail(t, s"starts begins a Scenario, not $other")
      case Apply(Select(b, op @ ("actions" | "actionKeys")), List(items)) =>
        scenario(fold(b, env), t) { s =>
          if op == "actions" then varargs(items).foreach(c => s.addActions(classOf(c)))
          else varargs(items).foreach(k => s.addKeys(constString(k)))
        }
      case Select(b, "free") => scenario(fold(b, env), t)(_.setFree(true))

      case Apply(Ident("query"), List(name)) => Decl.QueryNamed(textOf(fold(name, env), name))
      case Apply(TypeApply(Select(q, form @ ("find" | "verify")), _), List(p)) =>
        fold(q, env) match
          case Decl.QueryNamed(name) =>
            val f = if form == "find" then ir.Query.Form.FORM_FIND else ir.Query.Form.FORM_VERIFY
            Decl.QueryOn(name, f, claimOf(fold(p, env), p))
          case other => fail(t, s"$form asks a Query, not $other")
      case Apply(Apply(TypeApply(Select(q, "in"), _), List(s)), List(reads)) =>
        (fold(q, env), fold(reads, env)) match
          case (Decl.QueryOn(name, form, p), Decl.Reading(through)) =>
            Decl.QueryIn(name, form, p, claimOf(fold(s, env), s), through)
          case (other, _) => fail(t, s"in gives a Query its Scenario, not $other")
      case Apply(Select(q, "explore"), List(space)) =>
        fold(q, env) match
          case Decl.Declared(name) if queries.contains(name) =>
            val value = ir.Exploration.newBuilder()
            declaration(Bound(space, Map.empty), value)
            queries(name) = queries(name).toBuilder.setExploration(value).build()
            Decl.Declared(name)
          case other => fail(t, s"explore declares a Query's finite variations, not $other")
      case Apply(Select(q, "expect"), List(expected)) =>
        fold(q, env) match
          case Decl.Declared(name) if queries.contains(name) =>
            val value = ir.RunExpectation.newBuilder()
            declaration(Bound(expected, Map.empty), value)
            queries(name) = queries(name).toBuilder.setExpectedRun(value).build()
            Decl.Declared(name)
          case other => fail(t, s"expect declares a Query's live assessment, not $other")
      case Apply(Select(q, "limits"), List(l)) =>
        fold(q, env) match
          case Decl.QueryIn(name, form, p, s, through) =>
            val limits = fold(l, env) match
              case Decl.Bounds(limits) => limits
              case other               => fail(l, s"expected Limits, got $other")
            query(name, form, p, s, through, limits, t)
          case other => fail(t, s"limits bounds a Query, not $other")
      case Apply(TypeApply(Select(Ident("Reads"), "through"), _), List(m, _)) =>
        Decl.Reading(Some(modelName(fold(m, env), m)))
      case TypeApply(Ident("identity"), _) if t.symbol.owner.fullName.startsWith("umpire.Reads") =>
        Decl.Reading(None)
      case Apply(Select(companion, "apply"), _)
          if companion.tpe.typeSymbol.companionClass.fullName == "umpire.Limits" =>
        Decl.Bounds(limitsOf(t))

      case Apply(
            Apply(Apply(TypeApply(Ident("leadsTo"), _), List(m)), List(name)),
            List(from, to, within, under)
          ) =>
        val (machine, n) = (modelName(fold(m, env), m), textOf(fold(name, env), name))
        val steps = constInt(within)
        if steps < 1 then
          fail(t, s"progress $n bounds itself within $steps steps; a bound is at least one step")
        val p = ir.Progress
          .newBuilder()
          .setMachine(machine)
          .setName(n)
          .setPosition(pos(t))
          .setFrom(stepFunction(from, s"$machine.progress.$n", "from"))
          .setTo(stepFunction(to, s"$machine.progress.$n", "to"))
          .setWithin(steps.toInt)
          .addAllAssumptions(varargs(under).map(a => assumptionOf(resolveSymbol(a), a)).asJava)
        register(progress, (machine, n), p.build(), t, s"progress $n of $machine")
        Decl.Declared(n)

      // A value declared elsewhere, folded once: a machine or composition by its name, anything else
      // by its definition.
      case r: Ref if !isFunction(r.symbol) && defs.contains(resolveSymbol(r)) =>
        val sym = resolveSymbol(r)
        folded.getOrElseUpdate(
          sym,
          valDef(sym, r, "a declaration") match
            case d if isNamed(d.tpt.tpe, "umpire.Machine") => Decl.Model(machineOf(sym, r).getName)
            case d if isNamed(d.tpt.tpe, "umpire.Composition") =>
              Decl.Model(compositionOf(sym, r).getName)
            case d => fold(d.rhs.get, Map.empty)
        )
      // A helper function of the lifted sources that declares: its body, with its arguments bound.
      case Apply(fn, args) if isFunction(fn.symbol) =>
        defs(fn.symbol) match
          case d: DefDef =>
            val params = d.termParamss.flatMap(_.params).map(_.symbol)
            fold(d.rhs.get, params.zip(args.map(fold(_, env))).toMap)
          case _ => fail(t, s"${fn.symbol.fullName} is not a function of the lifted sources")
      case other => fail(other, s"not a declaration the IR carries: ${other.show}")

    def scenario(d: Decl, at: Tree)(f: ir.Scenario.Builder => Unit): Decl = d match
      case Decl.ScenarioOn(m, name, start) =>
        val s = ir.Scenario
          .newBuilder()
          .setMachine(m)
          .setName(name)
          .setPosition(pos(at))
          .setStart(start.getOrElse(fail(at, s"Scenario $name of $m names no start")))
        f(s)
        register(scenarios, (m, name), s.build(), at, s"Scenario $name of $m")
        Decl.Claim(claim(m, name))
      case other => fail(at, s"expected a Scenario, got $other")

    /** A root: a machine, a composition, a Query, a list of Queries, a progress claim, or a realization. */
    def liftRoot(root: String): Unit =
      val sym = defs.keys
        .find(s => s.isValDef && s.fullName == root)
        .getOrElse(throw LiftError(s"root $root", "names no declaration of the lifted sources"))
      val d = valDef(sym, sym.tree, "a declaration")
      if isNamed(d.tpt.tpe, "umpire.Machine") then machineOf(sym, d)
      else if isNamed(d.tpt.tpe, "umpire.Composition") then compositionOf(sym, d)
      else if isNamed(d.tpt.tpe, "umpire.realize.Realization") then realizationOf(sym, d)
      else
        val kind = d.tpt.tpe.widen.dealias
        val claims = Set("umpire.Query", "umpire.Progress")
        val listed =
          isList(kind.typeSymbol) || kind.typeSymbol.fullName == "scala.collection.immutable.Vector"
        if !claims(kind.typeSymbol.fullName) && !(listed && kind.typeArgs.headOption
            .exists(a => isNamed(a, "umpire.Query")))
        then
          fail(
            d,
            s"$root is a ${kind.show}; a root is a machine, a composition, a Query, a list of Queries, a progress claim or a realization"
          )
        fold(Ref(sym), Map.empty)

    // The `given Finite[S]` blocks: an Int field's range is the `Finite.upTo(bound)` in scope there.
    // An Inbox field's channel is the one whose contents are in scope there, and an opaque type's own
    // given is its range.
    def finiteOf(tpt: TypeTree): Option[TypeRepr] =
      if tpt.tpe.typeSymbol.name == "Finite" then tpt.tpe.typeArgs.headOption else None
    for d <- defs.values do
      d match
        case ValDef(_, tpt, Some(Block(stats, _))) if finiteOf(tpt).nonEmpty =>
          val state = finiteOf(tpt).get.dealias.typeSymbol.fullName
          for case ValDef(_, _, Some(Apply(Select(_, "upTo"), List(bound)))) <- stats do
            intRanges(state) = (0L, constInt(bound))
          for
            case ValDef(_, inbox, Some(Select(channel, "contents"))) <- stats; m <- finiteOf(inbox)
          do channelFields((state, messageType(m))) = resolveSymbol(channel)
        case ValDef(_, tpt, Some(Apply(Select(_, "upTo"), List(bound))))
            if finiteOf(tpt).exists(_.typeSymbol.flags.is(Flags.Opaque)) =>
          opaqueRanges(finiteOf(tpt).get.typeSymbol.fullName) = (0L, constInt(bound))
        case _ => ()

    // Every root is lifted, and one that fails is reported without stopping the others. Nothing is
    // written once one failed, so what a failed root left behind matters only as the functions it
    // was still lifting.
    for root <- roots.distinct.sorted do
      try liftRoot(root)
      catch
        case e: LiftError =>
          errors += e
          lifting.clear()

    val m = ir.Model.newBuilder().setSource("model/scalav2: " + roots.toList.sorted.mkString(", "))
    types.toList.sortBy(_._1).foreach((_, t) => m.addTypes(t))
    functions.toList.sortBy(_._1).foreach((_, f) => m.addFunctions(f))
    actions.toList.sortBy(_._1).foreach((_, a) => m.addActions(a))
    machines.values.toList.sortBy(_.getName).foreach(m.addMachines)
    channels.toList.sortBy(_._1).foreach((_, c) => m.addChannels(c))
    monitors.toList.sortBy(_._1).foreach((_, x) => m.addMonitors(x))
    assumptions.toList.sortBy(_._1).foreach((_, a) => m.addAssumptions(a))
    holes.toList.sortBy(_._1).foreach((_, h) => m.addHoles(h))
    compositions.values.toList.sortBy(_.getName).foreach(m.addCompositions)
    properties.toList.sortBy(_._1).foreach((_, p) => m.addProperties(p))
    scenarios.toList.sortBy(_._1).foreach((_, s) => m.addScenarios(s))
    queries.toList.sortBy(_._1).foreach((_, q) => m.addQueries(q))
    progress.toList.sortBy(_._1).foreach((_, p) => m.addProgress(p))
    realizations.toList.sortBy(_._1).foreach((_, r) => m.addRealizations(r))
    models += m.build()

/**
 * `lift <model.jar> <classpath file> <out.json> <source prefix> <root>...`: lift the machines named
 * by the roots, the fully qualified names of their `val`s, from the Temporal Models in the jar. The
 * prefix turns the sources' build-relative paths into repository-relative ones.
 */
// `lift <jar=prefix>,... <classpath file> <out.json> <root>...` reads several jars, each with the
// prefix of its own sources. Either way a root may also name a composition, a Query, a list of Queries
// or a progress claim; every jar's TASTy but the framework's is read; and every root is lifted and
// every refusal reported before anything is written.
@main def lift(args: String*): Unit =
  val (specs, classpathFile, out, roots) = args.toList match
    case jars :: classpath :: out :: roots if jars.contains("=") =>
      val specs = jars
        .split(",")
        .toList
        .map(_.split("=", 2) match
          case Array(jar, prefix) => (jar, prefix)
          case Array(jar)         => (jar, ""))
      (specs, classpath, out, roots)
    case jar :: classpath :: out :: prefix :: roots => (List(jar -> prefix), classpath, out, roots)
    case _                                          =>
      System.err.println("usage: lift <jar=prefix>,... <classpath file> <out.json> <root>...")
      sys.exit(2)
  val scratch = Files.createTempDirectory("umpire-lift")
  val prefixes = mutable.Map.empty[String, String]
  val tastys = specs.zipWithIndex.flatMap { case ((jar, prefix), i) =>
    val zip = ZipFile(jar)
    zip.entries.asScala
      .filter(e => !e.getName.startsWith("umpire/") && e.getName.endsWith(".tasty"))
      .map { e =>
        val entry = s"$i/${e.getName}"
        val p = scratch.resolve(entry)
        Files.createDirectories(p.getParent)
        Files.copy(zip.getInputStream(e), p)
        prefixes(entry) = prefix
        p.toString
      }
      .toList
  }.sorted
  val classpath = specs.map(_._1) ++ Files
    .readString(Path.of(classpathFile))
    .trim
    .split(java.io.File.pathSeparator)
    .toList
  if roots.isEmpty then
    System.err.println("lift: no roots: name the declarations to lift")
    sys.exit(1)
  val lifter = Lifter(roots, prefixes.toMap)
  TastyInspector.inspectAllTastyFiles(tastys, Nil, classpath)(lifter)
  if lifter.errors.nonEmpty then
    if sys.env.contains("LIFT_DEBUG") then lifter.errors.foreach(_.printStackTrace())
    lifter.errors.foreach(e => System.err.println(s"lift: ${e.getMessage}"))
    sys.exit(1)
  val json = JsonFormat
    .printer()
    .print(lifter.models.headOption.getOrElse(sys.error("lift: nothing was lifted")))
  Files.writeString(Paths.get(out), json + "\n")
