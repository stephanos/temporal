package framework.reify

import io.temporal.server.api.umpire.v1 as ir
import io.temporal.server.api.umpire.v1.Expr.Kind as E

private[framework] trait BodySyntax:
  self: BodyEngine =>
  import ctx.*
  import ctx.given
  import Equality.given
  import ctx.quotes.reflect.*

  // Hook: a rule's effect written `rejects(why)` or `rejects(why).because(reason)`, lowered to the
  // steps its core form lifts to from the state `state`, or None for any other effect. Core form:
  // `rejects(why)` lifts as `reject(Outcome.rejected(why), s)` does,
  // `List(Step(Outcome.rejected(why), s))`, and `rejects(why).because(reason)` gives that step the
  // explanation `reason`.
  def rejection(effect: Term, state: ir.Expr): Option[ir.Expr] =
    val (rejects, reason) = unwrapped(effect) match
      case Apply(sel @ Select(inner, "because"), List(r))
          if sel.symbol.maybeOwner.fullName == "framework.dsl.Rejects" =>
        (unwrapped(inner), constString(r))
      case other => (other, "")
    sugarCall(rejects)
      .collect { case ("rejects", List(List(why), _)) => why }
      .map { why =>
        val rejected = sharedOutcome.children
          .find(_.name == "rejected")
          .getOrElse(fail(effect, "the shared Outcome has no case rejected"))
        declareType(sharedOutcome, effect)
        val outcome = expr(effect)(
          E.Construct(
            ir.Construct(
              `type` = irTypeName(sharedOutcome),
              `case` = rejected.name,
              args = Seq(lift(why, fieldTypes(rejected).headOption.map(_._2)))
            )
          )
        )
        list(
          Seq(
            step(
              outcome,
              state,
              list(Nil, effect),
              text(reason, effect),
              effect
            )
          ),
          effect
        )
      }

  // Top-level definitions of a file are members of its package object.
  protected val sugarOwner = "framework.dsl.Syntax$package$"

  // The sugar definition a term applies, if it applies one: its name and its argument lists.
  protected def sugarCall(t: Term): Option[(String, List[List[Term]])] = t match
    case Apply(fn, args)  => sugarCall(fn).map((n, as) => (n, as :+ args))
    case TypeApply(fn, _) => sugarCall(fn)
    case r: Ref if r.symbol.maybeOwner.fullName == sugarOwner =>
      Some(r.symbol.name -> Nil)
    // A guard of a machine's rules reads `phase.in(a, b)` as the rules repeat it.
    case r: Ref if r.symbol.maybeOwner.fullName == "framework.dsl.Rules" && r.symbol.name == "in" =>
      Some("in" -> Nil)
    case _ => None

  // The type arguments a sugar call is applied to, innermost first.
  protected def typeArgs(t: Term): List[TypeRepr] = t match
    case Apply(fn, _)        => typeArgs(fn)
    case TypeApply(_, targs) => targs.map(_.tpe)
    case _                   => Nil

  // Hook: whether the term applies a sugar form, which `sugar` lifts. Core form: none of its own;
  // `lift` asks it of every term before `sugar` lowers one, as in `case _ if sugared(t) => sugar(t)`.
  def sugared(t: Term): Boolean =
    sugarCall(t).nonEmpty || entersApplied(t).nonEmpty

  // `enters(p)(s)`, the step function `enters(p)` applied to a state: the call and the state.
  private def entersApplied(t: Term): Option[(Term, Term)] = t match
    case Apply(Select(inner, "apply"), List(state)) =>
      sugarCall(inner).collect { case ("enters", _) => (inner, state) }
    case _ => None

  // The function-valued rule effect remains open for declaration-level expansion.
  // Core form: `List(Step(outcome, state))`, with the phase in `state` replaced by the rule effect.
  def sugar(t: Term): ir.Expr = sugarCall(t) match
    case _ if entersApplied(t).nonEmpty =>
      val (call, state) = entersApplied(t).get
      entering(call, Right(state)).get
    case Some(("enters", _)) =>
      fail(
        t,
        "enters(...) is a step function of the state: a rule's effect, `when(...) ~> " +
          "enters(Phase.x)`, or applied to the state in a step function, `enters(Phase.x)(s)`"
      )
    case _ => fail(t, s"outside the liftable subset: ${t.show}")

  // A statement of an `effect { ... }` block: a field assignment by its setter, `phase = p`, which
  // for a field whose values declare their status records it, a `record(facts*)` or a
  // `reject(outcome)`.
  private enum EffectStatement(val at: Tree):
    case Assign(field: String, value: Term, call: Tree, recorded: Boolean)
        extends EffectStatement(call)
    case Record(facts: List[Term], call: Tree) extends EffectStatement(call)
    case Reject(outcome: Term, call: Tree) extends EffectStatement(call)

    def values: List[Term] = this match
      case Assign(_, value, _, _) => List(value)
      case Record(facts, _)       => facts
      case Reject(outcome, _)     => List(outcome)

    def written: String = this match
      case Assign(field, _, _, _) => s"the assignment of $field"
      case Record(_, _)           => "record(...)"
      case Reject(_, _)           => "reject(...)"

  private lazy val recordedClass = Symbol.requiredClass("framework.dsl.Recorded")

  // The status fact an enum case declares, the argument its case passes to the enum's parameter
  // `status`: `Fact.statusStarted` of `case started extends Phase(Fact.statusStarted)`.
  def declaredStatus(caseSym: Symbol, at: Tree): Term =
    val phase = enumOf(caseSym)
    val index = phase.primaryConstructor.paramSymss
      .find(_.headOption.exists(_.isTerm))
      .map(_.indexWhere(_.name == "status"))
      .filter(_ >= 0)
      .getOrElse(
        fail(
          at,
          s"${phase.name} declares the status of its cases by its parameter `status`, " +
            s"`enum ${phase.name}(val status: Fact) extends Recorded[Fact]`"
        )
      )
    object constructed extends TreeAccumulator[Option[Term]]:
      def foldTree(found: Option[Term], tree: Tree)(owner: Symbol) =
        found.orElse(tree match
          case Apply(fn, args) if fn.symbol == phase.primaryConstructor =>
            args.lift(index)
          case _ => foldOverTree(None, tree)(owner))
    val declared =
      defs.get(caseSym).orElse(sourceTree(caseSym))
    declared
      .flatMap(d => constructed.foldTree(None, d)(caseSym))
      .getOrElse(
        fail(at, s"${caseSym.name} of ${phase.name} declares no status")
      )

  def storedStatus(caseSym: Symbol, at: Tree): Option[(ir.Expr, String)] = None

  private def statusOf(caseSym: Symbol, at: Tree, fact: Option[TypeRepr]): (ir.Expr, String) =
    storedStatus(caseSym, at).getOrElse {
      val declared = declaredStatus(caseSym, at)
      val name = resolve(declared) match
        case r: Ref => r.symbol.name
        case other  => other.show
      lift(declared, fact) -> name
    }

  private def syntaxCall(fn: Term, name: String): Boolean =
    fn.symbol.name == name && fn.symbol.maybeOwner.fullName == sugarOwner

  // The statement of an effect block a term is, if it is one. A call of a def that takes the
  // block's draft and is no getter is a setter of the fixed shape, or refused, naming it.
  private def effectStatement(t: Term): Option[EffectStatement] = unwrapped(
    t
  ) match
    case c @ Apply(Apply(TypeApply(fn, _), List(_)), List(first, rest))
        if syntaxCall(fn, "record") =>
      Some(EffectStatement.Record(first :: varargs(rest), c))
    case c @ Apply(Apply(TypeApply(fn, _), List(_)), List(outcome)) if syntaxCall(fn, "reject") =>
      Some(EffectStatement.Reject(outcome, c))
    case c: Apply if accessor(c.symbol) && getterField(c.symbol).isEmpty =>
      val fn = c.symbol
      (setterShape(fn), c) match
        case (Some((field, false)), Apply(Apply(_, List(value)), List(_)))
            if value.tpe.widen.derivesFrom(recordedClass) =>
          fail(
            c,
            s"${fn.name}, declared at ${declaredAt(fn)}, assigns $field, whose values declare " +
              "their status, so it hands the draft the value whose status the step records: " +
              "`def phase_=(p: Phase)(using d: Draft[State, ?, Fact]): Unit = " +
              "d.set(p)(_.copy(phase = p))`"
          )
        case (Some((field, recorded)), Apply(Apply(_, List(value)), List(_))) =>
          Some(EffectStatement.Assign(field, value, c, recorded))
        case _ =>
          fail(
            c,
            s"${fn.name}, declared at ${declaredAt(fn)}, takes the state of a block and is no " +
              "field accessor: a setter replaces one field, " +
              "`def phase_=(p: Phase)(using d: Draft[State, ?, ?]): Unit = d.set(_.copy(phase = p))`"
          )
    case _ => None

  // An `effect { ... }` block's statements, lifted as the steps of its method form: the
  // assignments as one copy of the state, its fields in the order the state declares them; the facts
  // in the order recorded; and the ok outcome. The IR's copy of the state has no statement order, so
  // what would make the order matter is refused: a statement in a branch or a loop, a rejecting
  // block with another statement, a field assigned twice or read after it is assigned. So is any
  // statement but an assignment, `record` or `reject`. Core form: `effect { phase = p; record(f) }`
  // lifts as `enter(s.copy(phase = p), f)`, and `effect { reject(o) }` as `reject(o, s)`.
  def effectSteps(b: SectionBlock, body: Term): ir.Expr =
    import EffectStatement.*
    val block = s"the effect block of ${b.at.name}"
    def statements(t: Tree): List[Tree] = t match
      case term: Term =>
        unwrapped(term) match
          case Block(stats, e)         => stats.flatMap(statements) ++ statements(e)
          case Literal(UnitConstant()) => Nil
          case other                   => List(other)
      case other => List(other)
    // The first statement written inside a tree.
    def within(t: Tree): Option[(Term, EffectStatement)] =
      object first extends TreeAccumulator[Option[(Term, EffectStatement)]]:
        def foldTree(found: Option[(Term, EffectStatement)], tree: Tree)(
            owner: Symbol
        ) =
          found.orElse(tree match
            case term: Term if effectStatement(term).nonEmpty =>
              Some(term -> effectStatement(term).get)
            case _ => foldOverTree(None, tree)(owner))
      first.foldTree(None, t)(b.at.symbol)
    def named(t: Tree): String = t match
      case v: ValDef                                => s"the local val ${v.name}"
      case d: Definition                            => s"the local definition ${d.name}"
      case _: If                                    => "an `if`"
      case _: Match                                 => "a `match`"
      case _: While                                 => "a loop"
      case c: Apply if isNamed(c.tpe, "scala.Unit") =>
        s"the call of ${c.symbol.name}"
      case _ => "a bare expression"
    def inside(at: Tree, s: EffectStatement, place: String): Nothing =
      fail(
        at,
        s"${s.written} is inside $place of $block: an effect block's statements are straight-line, " +
          "so branch with a method-form effect, `def f(s: State) = ...`"
      )
    val lifted = statements(body).map { t =>
      val own = t match
        case term: Term => effectStatement(term)
        case _          => None
      own match
        case Some(s) =>
          for v <- s.values; (at, n) <- within(v) do inside(at, n, s"the value of ${s.written}")
          s
        case None =>
          for (at, n) <- within(t) do inside(at, n, named(t))
          fail(
            t,
            s"${named(t)} is a statement of $block, which holds field assignments, `record(...)` " +
              "and `reject(...)` and nothing else"
          )
    }
    for
      r <- lifted.collectFirst { case r: Reject => r };
      other <- lifted.find(_ ne r)
    do
      fail(
        other.at,
        s"$block rejects at ${where(r.at)}, so it holds that one `reject(...)` and nothing else, " +
          s"not ${other.written}"
      )
    val assigns = lifted.collect { case a: Assign => a }
    for
      (a, i) <- assigns.zipWithIndex;
      first <- assigns.take(i).find(_.field == a.field)
    do
      fail(
        a.at,
        s"$block assigns ${a.field} twice, at ${where(first.at)} and here: assign each field once"
      )
    // A field read after its assignment reads the assigned value at run time, the old one in the IR.
    for (s, i) <- lifted.zipWithIndex do
      val assigned = lifted.take(i).collect { case a: Assign => a.field }.toSet
      object reads extends TreeTraverser:
        override def traverseTree(tree: Tree)(owner: Symbol): Unit = tree match
          case Apply(fn, List(_)) if getterField(fn.symbol).exists(assigned) =>
            val field = getterField(fn.symbol).get
            fail(
              tree,
              s"$block reads $field after it assigns it: the IR's copy of the state has no " +
                s"statement order, so read $field before its assignment"
            )
          case _ => super.traverseTree(tree)(owner)
      s.values.foreach(v => reads.traverseTree(v)(b.at.symbol))
    val at = b.body
    def state = expr(at)(E.Var("s"))
    lifted match
      case List(Reject(outcome, call)) =>
        list(
          Seq(
            step(lift(outcome), state, list(Nil, call), text("", call), call)
          ),
          call
        )
      case _ =>
        val ok = b.call.usings
          .lift(1)
          .getOrElse(fail(at, s"$block is given no ok outcome"))
        val outcome = outcomeOf(ok, "effect")
        val values = assigns.map(a => a.field -> a.value).toMap
        val updates = fieldTypes(b.state.widen.typeSymbol).collect {
          case (field, tpe) if values.contains(field) =>
            ir.NamedExpr(field, Some(lift(values(field), Some(tpe))))
        }
        val entered =
          if updates.isEmpty then state
          else expr(at)(E.Copy(ir.Copy(Some(state), updates)))
        val fact = b.call.types.lift(2)
        val recorded = lifted.collect { case r: Record =>
          r.facts.map(f => (r, lift(f, fact)))
        }
        // Each assignment of a field whose values declare their status records the status of the
        // case it names, after the facts the block records.
        val statuses = assigns.filter(_.recorded).map { a =>
          val (status, name) = resolve(a.value) match
            case r: Ref if isEnumCase(r.symbol) =>
              statusOf(r.symbol, a.at, fact)
            case _ =>
              fail(
                a.value,
                s"$block assigns ${a.field} a value that names no case: the status it records " +
                  s"is the declared status of a case, so name the case, `${a.field} = Phase.paused`"
              )
          for (r, f) <- recorded.flatten if f.kind == status.kind
          do
            fail(
              r.at,
              s"$block records $name, which its assignment of ${a.field} already records: " +
                s"drop the record($name)"
            )
          status
        }
        val facts = recorded.flatten.map(_._2) ++ statuses
        list(Seq(step(outcome, entered, list(facts, at), text("", at), at)), at)
  // Hook: a rule's effect written `enters(Phase.x)`, or the steps of `enters(Phase.x)(s)`, lowered to
  // the steps its core form lifts to from the state `state`, the rules' state variable by its name
  // or a term, or None for any other effect. The phase is a case of the phase type, and the state it
  // enters is the writer the machine's `Phased` declares applied to the state and that case, its
  // own or for a derived machine its source's. Core form: `enters(Phase.paused)` lifts as
  // `enter(s.copy(phase = Phase.paused), statusPaused)` does, the status the case declares where
  // its enum is `Recorded`, and no fact otherwise.
  def entering(effect: Term, state: Either[String, Term]): Option[ir.Expr] =
    val call = unwrapped(effect)
    sugarCall(call).collect { case ("enters", List(List(phase), List(_, ok, phasing))) =>
      val at = call
      val written = resolve(phase) match
        case r: Ref if isEnumCase(r.symbol) => r.symbol
        case _                              =>
          fail(
            phase,
            s"enters names the case of the phase type it enters, `enters(Phase.paused)`, not " +
              phase.show
          )
      // A facet's given is read through the facet's `this`, whose type's symbol is the facet trait.
      val declared = phasing.tpe match
        case TermRef(ThisType(self), _) => self.typeSymbol
        case TermRef(prefix, _)         => prefix.typeSymbol
        case other                      => other.typeSymbol
      // A facet's sections read the projection of the machine that mixes the facet in.
      val owner =
        if declared.typeRef.derivesFrom(facetClass) && !objectForm(declared)
        then liftingMachine.getOrElse(declared)
        else declared
      val entered = contextualPhaseWrite(declared, state, phase, at).getOrElse {
        val (sp, pp, body) = phaseWriter(owner, at).flatMap(lambda) match
          case Some((List(sp, pp), body)) => (sp, pp, body)
          case _                          =>
            fail(
              at,
              s"enters(${written.name}) writes the phase through the writer the machine's Phased " +
                "declares, and it declares none: " +
                "`Phased[State, Phase](_.phase, (s, p) => s.copy(phase = p))`"
            )
        val bound = state match
          case Left(name) =>
            renamed(sp.symbol) = name
            Map(pp.symbol -> phase)
          case Right(term) => Map(sp.symbol -> term, pp.symbol -> phase)
        binding(Map.empty, Map.empty, bound)(lift(body))
      }
      val fact = typeArgs(call).lift(2)
      val facts =
        if phase.tpe.widen.derivesFrom(recordedClass) then List(statusOf(written, phase, fact)._1)
        else Nil
      list(
        Seq(
          step(
            outcomeOf(ok, "enters"),
            entered,
            list(facts, at),
            text("", at),
            at
          )
        ),
        at
      )
    }
