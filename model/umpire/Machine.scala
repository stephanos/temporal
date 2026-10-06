package umpire

import scala.annotation.unused

// One result of an action: the outcome, the next state and the facts it records. `because` is an
// optional explanation a generated table view shows beside the row; no fingerprint reads it.
// Covariant in its facts, so a step that records none, such as `stay(s)` or `reject(o, s)`, is a
// step of every machine of its states and outcomes, and its effect needs no stated result type.
final case class Step[S, O, +F](outcome: O, state: S, facts: List[F] = Nil, because: String = "")

// `steps.because("...")`: the explanation of each step written here, as `Step`'s `because` gives it.
// The lifter reads it only on a list of steps the step function writes out.
extension [S, O, F](steps: List[Step[S, O, F]])
  def because(reason: String): List[Step[S, O, F]] = steps.map(_.copy(because = reason))

// The name of one alternative of a step that can go more than one way, declared once by the val that
// names it, `val committed = choice`; the lifter names it after that val. A token is compared by
// identity, so two tokens are two names, as two input tokens are two inputs.
final class Choice private[umpire] ()

// A new choice token: `val committed = choice`.
def choice: Choice = Choice()

// `choose(committed -> enter(...), redelivered -> stay(s))`: the results of a step that can go more
// than one way, in the order written, each named by its token. The names are metadata no check reads
// (model/SEMANTICS.md, Named choices), so the results are the ones the same steps give written as an
// unnamed list. Each alternative is one step written out, or a call of a function that gives at most
// one, such as a step another action shares, and each token names one of them; an alternative whose
// function gives no step is not taken. A choose of one alternative does not compile, since one result
// is no choice, and a step function's several results are always a choose: the lifter refuses an
// unnamed list of several steps.
def choose[S, O, F](
    first: (Choice, List[Step[S, O, F]]),
    second: (Choice, List[Step[S, O, F]]),
    rest: (Choice, List[Step[S, O, F]])*
): List[Step[S, O, F]] =
  val alternatives = first +: second +: rest
  for (token, steps) <- alternatives do
    require(alternatives.count(_._1 eq token) == 1, "a choice names one alternative of a choose")
    require(steps.sizeIs <= 1, "each alternative of a choose is at most one step")
  alternatives.flatMap(_._2).toList

// A declared, derived or composed machine.
trait Model:
  def name: String

// The name an object form takes from its object: `object OrderProduct` is `orderProduct`. A
// law is named the same way (umpire.Law), and the lifter reads the same name from the source.
private[umpire] def objectName(of: AnyRef): String =
  val n = of.getClass.getSimpleName.stripSuffix("$")
  n.take(1).toLowerCase + n.drop(1)

// What a claim is declared on, by its state type `S`: a machine or a composition. A claim written once
// over `Declares[S]`, such as a function over `m: Declares[S]` and the predicates it reads, declares
// the same Property on either. `Outcome` and `Fact` are the types a step of it answers and records:
// the machine's own, or for a composition the strings of its composed keys.
trait Declares[S] extends Model:
  type Outcome
  type Fact

  // The state type, which a machine or composition object's members name through it, `def end(s:
  // State)`, so `extends Machine[S, O, F]` is the one place it is written. The IR generator reads it
  // as the type it stands for. Its outcomes and facts are not given names here: inside an object
  // they would hide the feature's own `Outcome` and fact types.
  type State = S

  // A Property or a Scenario is named after the `val` that declares it (`val completes =
  // orderProduct.property holds ...`), or by the name it is given where it is declared without
  // one, such as in a list or inside a function over a machine. A Scenario that names no start starts
  // in its machine's one declared start, or for a composition in its members' starts. Inside a
  // machine or composition object they name it implicitly: `property when c holds ...`,
  // `scenario.actions(...)`.
  def property: PropertyBuilder[S, Outcome, Fact] = PropertyBuilder("", this, None)
  def property(name: String): PropertyBuilder[S, Outcome, Fact] = PropertyBuilder(name, this, None)
  def scenario: ScenarioBuilder[S] = ScenarioBuilder("", this, None)
  def scenario(name: String): ScenarioBuilder[S] = ScenarioBuilder(name, this, None)

  // Declares this machine's or composition's capabilities, whose generated `verify` Queries run under
  // `limits`: `capabilities(limits = three)(...)`, which is `capabilities(this, limits)(...)`. A
  // machine or composition object declares its own in `object implements extends Implements(...)`;
  // this form serves a function that declares them for several designs.
  def capabilities(limits: Limits)(declared: CapabilityOf[S, Outcome, Fact]*)(using
      Catalog
  ): LawDeclaration[S] = umpire.capabilities(this, limits)(declared*)

  // Declares the capabilities of another machine or composition `m`, as the top-level
  // `capabilities(m, limits)(...)` does, from inside an object form, whose own `capabilities` hides
  // that one.
  def capabilities[T](m: Declares[T], limits: Limits)(
      declared: CapabilityOf[T, m.Outcome, m.Fact]*
  )(using Catalog): LawDeclaration[T] = umpire.capabilities(m, limits)(declared*)

// What a derivation binds in its source's place: one action's step function, `action ~> f`, or the
// rules of one action, `on(action) { where(g) ~> effect }` (model/umpire/Syntax.scala).
sealed trait Rebinding[S, O, F]

// An action bound to its step function.
final case class StepBinding[S, O, F](decl: ActionDecl, function: AnyRef) extends Rebinding[S, O, F]

// One rule of a machine's `rules` (model/umpire/Syntax.scala): the action, or one class of it, it
// fires, the condition under which it fires, and its effect, over the state and the class's inputs
// in declaration order. `index` is its place among its machine's rules, from 1, and `heading` and
// `action` are how a refusal names it and its action.
final case class Rule[S, O, F] private[umpire] (
    index: Int,
    heading: String,
    action: String,
    decl: ActionDecl,
    values: Option[List[Any]],
    guard: S => Boolean,
    effect: (S, List[Any]) => List[Step[S, O, F]]
):
  // Whether it fires the class of the inputs `inputs`.
  def fires(inputs: List[Any]): Boolean = values.forall(_ == inputs)

// The rules of one `on` block, which a derivation binds in its source's place.
final case class RuleGroup[S, O, F] private[umpire] (rules: Vector[Rule[S, O, F]])
    extends Rebinding[S, O, F]

// What a machine binds one action to: a step function, or the rules that lower to one.
private[umpire] enum Bound[S, O, F]:
  case Function(f: AnyRef)
  case Ruled(rules: Vector[Rule[S, O, F]])
  case Disabled()

// `action ~> stepFunction`. One extension per arity, each typed by the action's inputs, so a step
// function written for another action's inputs does not compile.
extension (a: Action[EmptyTuple])
  infix def ~>[S, O, F](f: S => List[Step[S, O, F]]): StepBinding[S, O, F] = StepBinding(a.decl, f)

extension [A](a: Action[A *: EmptyTuple])
  infix def ~>[S, O, F](f: (S, A) => List[Step[S, O, F]]): StepBinding[S, O, F] =
    StepBinding(a.decl, f)

extension [A, B](a: Action[(A, B)])
  infix def ~>[S, O, F](f: (S, A, B) => List[Step[S, O, F]]): StepBinding[S, O, F] =
    StepBinding(a.decl, f)

extension [A, B, C](a: Action[(A, B, C)])
  infix def ~>[S, O, F](f: (S, A, B, C) => List[Step[S, O, F]]): StepBinding[S, O, F] =
    StepBinding(a.decl, f)

// The owner of a machine's sections, which its `rules` read the machine's types from:
// `object rules extends Rules(_.phase)` in `object OrderProduct extends Machine[...]`.
final class Owner[S, O, F] private[umpire] (val machine: Machine[S, O, F])

// What an `implements` or `capabilities` section declares the capabilities of: the machine or
// composition whose object it sits in, with the outcomes and facts its steps answer and record.
final class Declaring[S, O, F] private[umpire] (val model: Declares[S])

// What a machine binds each action to, in the order the first binding of each names it: a step
// function, rules, or nothing. A machine's `rules` (model/umpire/Syntax.scala) is the one a Model
// writes; a derivation keeps what it makes as built, and the core `Bindings` binds step functions.
abstract class RuleBook[S, O, F]:
  private[umpire] def table: Vector[(ActionDecl, Bound[S, O, F])]

// A machine's rules in the core: one step function per action, bound by hand,
// `object rules extends Bindings(clerk.ship ~> shipStep, courier.strike ~> (_ => Nil))`, which is
// what `Rules` lower to. It is the spelling of the IR generator's core fixtures, whose step
// functions are written out; a Model says when each action fires in `Rules`, and the IR generator
// refuses a hand-bound step function in a Model (fn-126 R17).
abstract class Bindings[S, O, F](using @unused owner: Owner[S, O, F])(
    bindings: StepBinding[S, O, F]*
) extends RuleBook[S, O, F]:
  private[umpire] def table: Vector[(ActionDecl, Bound[S, O, F])] =
    bindings.toVector.map(b => b.decl -> Bound.Function[S, O, F](b.function))

// A step function's results for the state and one class's inputs in declaration order, whatever
// the function's arity: `f(s, inputs*)`, the function `action ~> f` binds.
private[umpire] def effectOf[S, O, F](
    decl: ActionDecl,
    f: AnyRef
): (S, List[Any]) => List[Step[S, O, F]] =
  // scalafix:off DisableSyntax.asInstanceOf
  type R = List[Step[S, O, F]]
  decl.domains.size match
    case 0 => (s, _) => f.asInstanceOf[S => R](s)
    case 1 => (s, i) => f.asInstanceOf[(S, Any) => R](s, i(0))
    case 2 => (s, i) => f.asInstanceOf[(S, Any, Any) => R](s, i(0), i(1))
    case _ => (s, i) => f.asInstanceOf[(S, Any, Any, Any) => R](s, i(0), i(1), i(2))
  // scalafix:on DisableSyntax.asInstanceOf

// The step function an action's binding lowers to, of the action's arity: its own, or for rules the
// effect of the first rule that fires, as `if g1(s) then e1(s) else if g2(s) then e2(s) else Nil`
// lifts, or none where it is disabled.
private[umpire] def stepFunction[S, O, F](decl: ActionDecl, bound: Bound[S, O, F]): AnyRef =
  def run(s: S, inputs: List[Any]): List[Step[S, O, F]] = bound match
    case Bound.Ruled(rules) =>
      rules.find(r => r.fires(inputs) && r.guard(s)).fold(Nil)(_.effect(s, inputs))
    case _ => Nil
  bound match
    case Bound.Function(f) => f
    case _                 =>
      decl.domains.size match
        case 0 => (s: S) => run(s, Nil)
        case 1 => (s: S, a: Any) => run(s, List(a))
        case 2 => (s: S, a: Any, b: Any) => run(s, List(a, b))
        case _ => (s: S, a: Any, b: Any, c: Any) => run(s, List(a, b, c))

// Every class of an action: each assignment of its inputs, in catalog order.
private[umpire] def classesOf(decl: ActionDecl): List[List[Any]] =
  decl.domains.foldLeft(List(List.empty[Any])): (prefixes, domain) =>
    for prefix <- prefixes; v <- domain.values.toList yield prefix :+ v

// A value as Umpire keys it in a class: a case by its name, followed by its fields, joined by `-`.
private[umpire] def valueKey(v: Any): String = v match
  case p: Product if p.productArity > 0 =>
    (p.productPrefix +: p.productIterator.map(valueKey).toList).mkString("-")
  case other => other.toString

// The refusal of two rules of one action class of `machine` that both fire in some state, naming
// the class, both rules by their place and heading, and the first such state in catalog order: rules
// say when an action fires, so no two of one class fire together (intended alternatives are one
// effect's `choose`). `added` is checked against each rule of `rules`.
private[umpire] def overlap[S, O, F](
    machine: String,
    states: Finite[S],
    rules: Seq[Rule[S, O, F]],
    added: Rule[S, O, F]
): Option[String] =
  val pairs = for
    earlier <- rules.view if earlier.decl == added.decl
    inputs <- classesOf(added.decl).view if earlier.fires(inputs) && added.fires(inputs)
    s <- states.values.view if earlier.guard(s) && added.guard(s)
  yield (earlier, inputs, s)
  pairs.headOption.map: (earlier, inputs, s) =>
    val cls = (added.action +: inputs.map(valueKey)).mkString("-")
    s"$machine fires $cls by two rules in $s: rule ${earlier.index}, ${earlier.heading}, and rule " +
      s"${added.index}, ${added.heading}: the rules of one action class hold in no common state, " +
      "so write alternatives as one effect that names each with `choose`"

// The name of an action as its rule writes it, from the source the rule was given: the last name
// of `clerk.ship`, or of the action a class applies, `buyer.change(Change.hold)`.
private[umpire] def writtenAction(code: String): String =
  // The type arguments of a class written by name, `apply[(A, B)](action)(...)`, name no action.
  val untyped = code.replaceAll("\\[[^\\]]*\\]", "")
  val path = "[A-Za-z_][A-Za-z0-9_$]*(?:\\.[A-Za-z_][A-Za-z0-9_$]*)*".r
  path
    .findAllIn(untyped)
    .map(_.stripSuffix(".apply"))
    .find(p => p.nonEmpty && p != "umpire" && p != "apply")
    .fold(code)(_.split('.').last)

// A machine: a transition relation over the finite state type `S` with outcomes `O` and facts `F`.
// Declared as an object that is the machine, named after the object with its first letter lowered:
//
// {{{
// object OrderProduct extends Machine[OrderState, Outcome, OrderFact]:
//   val init = OrderState(placed)
//   def end(s: OrderState) = states.terminal(s.phase)
//   object states: ...
//   object effects: ...
//   object rules extends Rules(_.phase): ...
// }}}
//
// Its members are its header, the state it starts in, `init`, the states it may end in, `end`, and
// where it declares them, `entity` and `evidence`; then its sections, objects named after what they
// hold, in this order: `states`, its vocabulary (the named sets and projections of its states);
// `refinement`, where it refines another machine (umpire.Refinement); `effects`; `monitors`;
// `rules`, which it must declare; `properties`; `implements` or `capabilities`, its capabilities
// (umpire.Implements, umpire.Capabilities); and `queries`. Its `entity` is the one entity its
// actions are `on`, or create, unless it names another. The IR generator (model/irgen) reads them
// from the source, and names each of its declarations after where it is declared: its package and
// the objects it sits in. At run time a machine is its init, its end and the step function each
// action's rules lower to: `rules` is the one section the machine itself reads, through the member
// it implements, so no section is found by reflection, and a section object, which initializes on
// its first use, is never read while the machine initializes.
abstract class Machine[S, O, F](using
    private[umpire] val fs: Finite[S],
    private[umpire] val fo: Finite[O],
    private[umpire] val ff: Finite[F]
) extends Declares[S]:
  type Outcome = O
  type Fact = F

  // The state the machine starts in, named `init` as Quint and TLA+ name it.
  def init: S

  // Whether the machine may end in `s`.
  def end(s: S): Boolean

  // When each action fires, and what it does then: `object rules extends Rules`.
  def rules: RuleBook[S, O, F]

  // The object's name with its first letter lowered.
  def name: String = objectName(this)

  // The owner its `rules` and other sections read its types from.
  protected given machineOwner: Owner[S, O, F] = Owner(this)

  // What its `implements` or `capabilities` declares the capabilities of: this machine.
  protected given declaring: Declaring[S, O, F] = Declaring(this)

  // What each action is bound to, in the order its first rule names it.
  private[umpire] def table: Vector[(ActionDecl, Bound[S, O, F])] = rules.table

  // The machines this one is made from, which constructing it constructs as well
  // (IrFile.construct): the machine its `object refinement` refines, and for a derivation its
  // source and the machine a `refining` names.
  private[umpire] def reaches: Seq[Model] = Refinement.declaredBy(this).toSeq

  // One step function per action: the rules of each, lowered (Rules.lowered).
  private[umpire] def bindings: List[StepBinding[S, O, F]] =
    table.map((decl, bound) => StepBinding[S, O, F](decl, stepFunction(decl, bound))).toList

  private def built(
      table: => Vector[(ActionDecl, Bound[S, O, F])],
      also: Model*
  ): Machine[S, O, F] =
    Built(objectName(this), List(init), end, table, this +: also)(using fs, fo, ff)

  // A machine that keeps the rows of the named actions and drops the rest, named after the object
  // that declares it, `object M extends Derived(m.restrict(...))`. It keeps the state type, starts
  // and ends, owns its own name and Definition IDs, and does not inherit a refinement.
  def restrict(keep: Action[?]*): Machine[S, O, F] =
    val decls = keep.map(_.decl).toSet
    // It keeps its source's monitors and assumptions, which are about the state and the machine.
    built(table.filter((d, _) => decls(d)))

  // A machine that binds other step functions or rules to actions this one binds, each in its
  // place, and keeps everything else this machine declares. `action ~> effect` keeps that action's
  // rules, each with its guard and class, and replaces their effect, which the lifter refuses where
  // its whole-action rules have several effects (rules that each fire one class may differ, as the
  // new effect reads the class's inputs); `on(action) { where(g) ~> effect }` replaces its rules.
  // The lifter refuses an action this machine does not bind.
  def rebind(replaced: (Owner[S, O, F] ?=> Rebinding[S, O, F])*): Machine[S, O, F] =
    val by = replaced.foldLeft(Map.empty[ActionDecl, Bound[S, O, F]]): (by, r) =>
      r(using machineOwner) match
        case StepBinding(decl, f) =>
          val kept = table.collectFirst { case (`decl`, Bound.Ruled(rs)) => rs }
          by.updated(
            decl,
            kept.fold(Bound.Function[S, O, F](f))(rs =>
              Bound.Ruled(rs.map(_.copy(effect = effectOf[S, O, F](decl, f))))
            )
          )
        case RuleGroup(rules) =>
          rules.groupBy(_.decl).foldLeft(by)((by, g) => by.updated(g._1, Bound.Ruled(g._2)))
    built(table.map((d, b) => d -> by.getOrElse(d, b)))

  // A machine that also binds actions this one does not, after its own bindings, by rules:
  // `extend(on(action) { where(g) ~> effect })`. The lifter refuses an action this machine binds
  // already, and a bare binding where this machine's actions are bound by rules.
  def extend(added: (Owner[S, O, F] ?=> Rebinding[S, O, F])*): Machine[S, O, F] =
    val more = added.flatMap(r =>
      r(using machineOwner) match
        case StepBinding(decl, f) => Vector(decl -> Bound.Function[S, O, F](f))
        case RuleGroup(rules) => rules.groupBy(_.decl).toVector.map((d, rs) => d -> Bound.Ruled(rs))
    )
    built(table ++ more)

  // A machine that refines `product` through `map` in place of the refinement this one declares,
  // and keeps the facts and outcomes that refinement lets the refined machine see.
  def refining[PS, PO, PF](product: Machine[PS, PO, PF])(@unused map: S => PS): Machine[S, O, F] =
    built(table, product)

  // A machine whose checks also make the assumptions named, each once, after this one's.
  def assuming(@unused added: Assumption*): Machine[S, O, F] = built(table)

  // A machine without this one's monitors and refinement: the same transitions, for a composition
  // a member's monitors may not watch (model/SEMANTICS.md leaves that undefined).
  def unmonitored: Machine[S, O, F] = built(table)

// A machine derived from another, declared as an object that is it:
// `object LateRecord extends Derived(OrderRecord.rebind(...))`, named after the object. The
// derivation is an expression of the core operations `rebind`, `extend`, `restrict`, `refining`,
// `assuming` and `unmonitored`; the object adds only the sections that are its own, `states`,
// `properties`, `implements` and `queries`, and reuses its source's effects by name.
abstract class Derived[S, O, F](derivation: Machine[S, O, F])
    extends Machine[S, O, F](using derivation.fs, derivation.fo, derivation.ff):
  final def init: S = derivation.init
  final def end(s: S): Boolean = derivation.end(s)
  final def rules: RuleBook[S, O, F] = derivation.rules
  final override private[umpire] def table: Vector[(ActionDecl, Bound[S, O, F])] =
    derivation.table
  final override private[umpire] def reaches: Seq[Model] =
    derivation +: Refinement.declaredBy(this).toSeq

// A machine a derivation makes from another: its name, its starts and ends, what it binds each
// action to and the machines it is made from. What else it declares only the lifter reads.
final private[umpire] class Built[S, O, F](
    override val name: String,
    starts: List[S],
    isEnd: S => Boolean,
    bound: => Vector[(ActionDecl, Bound[S, O, F])],
    from: Seq[Model]
)(using Finite[S], Finite[O], Finite[F])
    extends Machine[S, O, F]:
  def init: S = starts.head
  def end(s: S): Boolean = isEnd(s)
  object rules extends RuleBook[S, O, F]:
    private[umpire] def table: Vector[(ActionDecl, Bound[S, O, F])] = Vector.empty
  final override private[umpire] lazy val table: Vector[(ActionDecl, Bound[S, O, F])] = bound
  final override private[umpire] def reaches: Seq[Model] = from
