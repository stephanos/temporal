package plain

import scala.quoted.*

// The compile-time half of the native runtime. Nothing here lifts to IR: it names declarations,
// reads the vars of a Machine (names, types, bounds, accessors) and checks transition leaves.
object Macros:

  // The val, var or def the expansion sits in: `val pressButton = action` gives "pressButton".
  inline def enclosingName: String = ${ enclosingNameImpl }

  private def named(using q: Quotes)(start: q.reflect.Symbol): q.reflect.Symbol =
    import q.reflect.*
    var s = start
    while s.isNoSymbol == false && (s.flags.is(Flags.Synthetic) || s.name.startsWith("$") ||
        s.name.startsWith("<") || s.name == "macro" || s.isClassDef && s.isAnonymousClass)
    do s = s.owner
    s

  def enclosingNameImpl(using Quotes): Expr[String] =
    import quotes.reflect.*
    Expr(named(Symbol.spliceOwner).name)

  // The enclosing classes, innermost first.
  private def classes(using q: Quotes)(start: q.reflect.Symbol): List[q.reflect.Symbol] =
    Iterator.iterate(start)(_.owner).takeWhile(!_.isNoSymbol).filter(_.isClassDef).toList

  private def facetClass(using q: Quotes)(start: q.reflect.Symbol): Option[q.reflect.Symbol] =
    import q.reflect.*
    val facet = TypeRepr.of[Facet].typeSymbol
    classes(start).find(c => !(c eq facet) && c.typeRef.baseClasses.contains(facet))

  // The state vars a class declares, in declaration order: concrete vars are mutable fields,
  // a facet's abstract `var light: Lit` is a mutable deferred getter.
  def stateVars(using q: Quotes)(cls: q.reflect.Symbol): List[q.reflect.Symbol] =
    import q.reflect.*
    cls.declarations.filter(s =>
      s.flags.is(Flags.Mutable) && !s.name.endsWith("_=") && !s.flags.is(Flags.Synthetic)
    )

  // The declared type of a var, alias intact: `UpTo[3]`, not `Int`.
  private def declaredType(using q: Quotes)(cls: q.reflect.Symbol, v: q.reflect.Symbol) =
    import q.reflect.*
    v.tree match
      case d: ValDef => d.tpt.tpe
      case d: DefDef => d.returnTpt.tpe
      case _         => cls.typeRef.memberType(v).widenByName

  private def boundOf(using q: Quotes)(tpe: q.reflect.TypeRepr): Option[Int] =
    import q.reflect.*
    tpe match
      case AppliedType(tycon, List(ConstantType(IntConstant(n))))
          if tycon.typeSymbol.name == "UpTo" =>
        Some(n)
      case _ => None

  // The schema of a Machine: one Var per state var, with typed accessors on the given instance.
  inline def schema[M <: Facet](machine: M): List[Var] = ${ schemaImpl[M]('machine) }

  def schemaImpl[M <: Facet: Type](machine: Expr[M])(using Quotes): Expr[List[Var]] =
    import quotes.reflect.*
    val cls = TypeRepr.of[M].typeSymbol
    val vars = stateVars(cls)
    if vars.isEmpty then report.errorAndAbort("a Machine declares at least one state var")
    Expr.ofList(vars.map { v =>
      val tpe = declaredType(cls, v)
      tpe.asType match
        case '[t] =>
          val read = Select(machine.asTerm, v).asExprOf[t]
          val name = Expr(v.name)
          val shown = Expr(tpe.show(using Printer.TypeReprShortCode))
          // the symbol's info, which is what a separately compiled Machine offers without trees
          val info =
            Expr(cls.typeRef.memberType(v).widenByName.show(using Printer.TypeReprShortCode))
          val bound = Expr(boundOf(tpe))
          '{
            Var(
              $name,
              $shown,
              $info,
              $bound,
              () => $read,
              (value: Any) => value.isInstanceOf[t],
              (value: Any) =>
                ${
                  Assign(Select(machine.asTerm, v), '{ value.asInstanceOf[t] }.asTerm)
                    .asExprOf[Unit]
                }
            )
          }
    })

  // `before`: the record of the step's start, refined with the enclosing Machine's vars.
  def beforeImpl(using Quotes): Expr[Any] =
    import quotes.reflect.*
    val owners = classes(Symbol.spliceOwner)
    if !owners.exists(_.name == "postconditions$") then
      report.errorAndAbort(
        "before is the state a step started from; use it in a postcondition"
      )
    val cls = facetClass(Symbol.spliceOwner).getOrElse(
      report.errorAndAbort("before is read inside a Machine")
    )
    val refined = stateVars(cls).foldLeft(TypeRepr.of[Record]) { (t, v) =>
      Refinement(t, v.name, declaredType(cls, v))
    }
    refined.asType match
      case '[t] => '{ Runtime.before.asInstanceOf[t] }

  // `check(c)`: records the result under the enclosing claim member, with its line and text.
  def checkImpl(condition: Expr[Boolean])(using Quotes): Expr[Unit] =
    import quotes.reflect.*
    val member = Expr(named(Symbol.spliceOwner).name)
    val line = Expr(condition.asTerm.pos.startLine + 1)
    val text = Expr(condition.asTerm.pos.sourceCode.getOrElse(condition.show))
    '{ Runtime.check($condition, $member, $line, $text) }

  private val leafRule = "a transition ends in an effect call, a rejection or ()"

  // `on(action): body`: refuses state assignments and non-effect leaves, wraps each effect call in
  // a `fired` mark and records the arms as first-match nested `if`s, for the probe's report.
  def onImpl(self: Expr[Facet], action: Expr[Action], body: Expr[Unit])(using Quotes): Expr[Unit] =
    import quotes.reflect.*
    val inTransitions = classes(Symbol.spliceOwner).exists(_.name == "transitions$")
    if !inTransitions then
      report.errorAndAbort("on(action) is declared in a Machine's or a facet's transitions object")
    val facet = classes(Symbol.spliceOwner)
      .dropWhile(_.name != "transitions$")
      .drop(1)
      .headOption
      .map(_.name.stripSuffix("$"))
      .getOrElse("")
    val facetSym = TypeRepr.of[Facet].typeSymbol

    def isState(s: Symbol) =
      s.flags.is(Flags.Mutable) && s.owner.isClassDef && s.owner.typeRef.baseClasses.contains(
        facetSym
      )
    // no assignment to state anywhere in a transition
    new TreeTraverser:
      override def traverseTree(tree: Tree)(owner: Symbol): Unit = tree match
        case Assign(lhs, _) if isState(lhs.symbol) =>
          report.error("a transition says when; write the state change in effects", tree.pos)
        case _ => super.traverseTree(tree)(owner)
    .traverseTree(body.asTerm)(Symbol.spliceOwner)

    def isEffect(fn: Term) =
      val owner = fn.symbol.owner
      owner.flags.is(Flags.Module) && owner.name == "effects$"
    def isReject(fn: Term) = fn.symbol.name == "reject" && fn.symbol.owner.equals(facetSym)
    def show(t: Tree) = t.show(using Printer.TreeShortCode)

    val arms = List.newBuilder[String]
    // `guards` are the conditions under which control reaches `t`, outermost first.
    def leaf(t: Term, guards: List[String]): Term = t match
      case Inlined(_, Nil, inner) => leaf(inner, guards)
      case Typed(inner, _)        => leaf(inner, guards)
      case Block(stats, expr) if stats.forall { case _: ValDef => true; case _ => false } =>
        Block(stats, leaf(expr, guards))
      case If(cond, thenp, elsep) =>
        val g = show(cond)
        If(cond, leaf(thenp, guards :+ g), leaf(elsep, guards :+ s"!($g)"))
      case Match(scrutinee, cases) =>
        val s = show(scrutinee)
        var earlier = List.empty[String]
        val rewritten = cases.map { c =>
          val pat = show(c.pattern)
          val g = c.guard.map(show)
          val matches = s"$s matches $pat" + g.map(x => s" if $x").getOrElse("")
          val here = guards ++ earlier.reverse.map(e => s"!($e)") :+ matches
          earlier = matches :: earlier
          CaseDef(c.pattern, c.guard, leaf(c.rhs, here))
        }
        Match(scrutinee, rewritten)
      case Literal(UnitConstant()) =>
        arms += s"${guards.mkString(" && ")} => disabled"
        t
      case call @ Apply(fn, _) if isEffect(fn) =>
        val effect = Expr(fn.symbol.name)
        arms += s"${guards.mkString(" && ")} => effects.${fn.symbol.name}"
        '{ Runtime.fire($effect); ${ call.asExprOf[Unit] } }.asTerm
      case Apply(fn, _) if isReject(fn) =>
        arms += s"${guards.mkString(" && ")} => ${show(t)}"
        t
      case other =>
        report.error(s"$leafRule: ${show(other)} is none of these", other.pos)
        other
    val rewritten = leaf(body.asTerm, Nil).asExprOf[Unit]
    val listed = Expr(arms.result())
    '{ $self.bind($action, ${ Expr(facet) }, () => $rewritten, $listed) }

final class Var(
    val name: String,
    val tpe: String,
    val info: String,
    val bound: Option[Int],
    val read: () => Any,
    val accepts: Any => Boolean,
    val write: Any => Unit
):
  override def toString: String = s"$name: $tpe"
