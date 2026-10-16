package framework.dsl

import scala.annotation.{publicInBinary, unused}
import scala.reflect.{ClassTag, TypeTest}

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
final class Choice private[framework] ()

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
    require(
      alternatives.count(_._1 eq token) == 1,
      "a choice names one alternative of a choose"
    )
    require(steps.sizeIs <= 1, "each alternative of a choose is at most one step")
  alternatives.flatMap(_._2).toList

// A declared, derived or composed machine.
trait Model:
  def name: String

// The name an object form takes from its object: `object OrderProduct` is `orderProduct`.
private[framework] def objectName(of: AnyRef): String =
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

  // Whether the object mixes in its own phase projection, `Phased[State, Phase](_.phase)`
  // (model/framework/dsl/Syntax.scala), which a derived machine or derived composition may not.
  private[framework] def declaresPhase: Boolean = false

  // A Property or a Scenario is named after the `val` that declares it (`val completes =
  // orderProduct.property expect: ...`), or by the name it is given where it is declared without
  // one, such as in a list or inside a function over a machine. A Scenario that names no start starts
  // in its machine's one declared start, or for a composition in its members' starts. Inside a
  // machine or composition object they name it implicitly: `property after(c) expect: ...`,
  // `scenario.actions(...)`.
  def property: PropertyBuilder[S, Outcome, Fact] = PropertyBuilder("", this, None)
  def property(name: String): PropertyBuilder[S, Outcome, Fact] = PropertyBuilder(name, this, None)
  def scenario: ScenarioBuilder[S] = ScenarioBuilder("", this, None)
  def scenario(name: String): ScenarioBuilder[S] = ScenarioBuilder(name, this, None)

// What a derivation binds in its source's place: one action's step function, `action ~> f`, or the
// rules of one action, `on(action) { where(g) ~> effect }` (model/framework/dsl/Syntax.scala).
sealed trait Rebinding[S, O, F]

// An action bound to its step function.
final case class StepBinding[S, O, F](decl: ActionDecl, function: AnyRef)
    extends Rebinding[S, O, F]:
  private var linked = Option.empty[Captured[AnyRef]]
  private[framework] var derivationEffect = Option.empty[Either[String, RuleBindings.Packet]]
  private var publishedDerivation = Option.empty[Either[String, RuleBindings.Packet]]
  private[framework] def publishDerivation(value: Either[String, RuleBindings.Packet]): Unit =
    require(
      publishedDerivation.isEmpty && derivationEffect.isEmpty,
      "a binding retains derivation evidence once"
    )
    publishedDerivation = Some(value)
    derivationEffect = publishedDerivation
  private[framework] def derivationPublished: Boolean =
    publishedDerivation.zip(derivationEffect).exists((before, after) => before.eq(after))
  @publicInBinary private[framework] def retainDerivation(data: Seq[String]): StepBinding[S, O, F] =
    publishDerivation(
      if data.headOption.contains("refused-rule-effect-v1") then Left(data(1))
      else Right(RuleBindings.packet(data, "derived binding"))
    )
    this
  private[framework] def captured: Option[Captured[AnyRef]] = linked
  private def attach(value: Captured[AnyRef]): StepBinding[S, O, F] =
    require(linked.isEmpty, "a step binding is captured once before publication")
    linked = Some(value)
    this

  @publicInBinary private[framework] def retainOrigin(
      owner: String,
      machine: String,
      named: Boolean,
      position: String = ""
  ): StepBinding[S, O, F] =
    val value = linked.getOrElse(
      throw IllegalArgumentException("a binding origin requires captured function metadata")
    )
    require(value.origin.isEmpty, "a step binding retains its origin once before publication")
    linked = Some(
      value.copy(origin =
        Some(
          BindingOrigin(
            owner,
            machine,
            named,
            decl.sourceIdentity,
            decl,
            Option(position)
              .filter(_.nonEmpty)
              .map(p =>
                io.temporal.server.api.umpire.v1.Position
                  .parseFrom(java.util.Base64.getDecoder.decode(p))
              )
          )
        )
      )
    )
    this

object StepBinding:
  private[framework] def retaining[S, O, F](
      decl: ActionDecl,
      f: AnyRef,
      captured: Option[Captured[AnyRef]]
  ): StepBinding[S, O, F] =
    val binding = StepBinding[S, O, F](decl, f)
    captured.fold(binding)(binding.attach)

  private[framework] def linking[S, O, F](
      decl: ActionDecl,
      f: AnyRef,
      root: String,
      chunks: Seq[String]
  ): StepBinding[S, O, F] =
    val linked = Reify.captured(f, root, chunks)
    StepBinding[S, O, F](decl, f).attach(linked)

// One rule of a machine's `rules` (model/framework/dsl/Syntax.scala): the action, or one class of it, it
// fires, the condition under which it fires, and its effect, over the state and the class's inputs
// in declaration order. `index` is its place among its machine's rules, from 1, and `heading` and
// `action` are how a refusal names it and its action.
final case class Rule[S, O, F] private[framework] (
    index: Int,
    heading: String,
    action: String,
    decl: ActionDecl,
    values: Option[List[Any]],
    guard: S => Boolean,
    effect: (S, List[Any]) => List[Step[S, O, F]]
):
  // Whether it fires the class of the inputs `inputs`.
  private[framework] var retainedRule = Option.empty[RuleBindings.Retained]
  private[framework] var adaptedRule = Option.empty[DerivationBindings.Adaptation]
  private var publishedRule = Option.empty[RuleBindings.Retained]
  private[framework] def publishCapture(captured: Option[RuleBindings.Registration]): Unit =
    require(
      publishedRule.isEmpty && retainedRule.isEmpty,
      "a Rule retains capture once before publication"
    )
    publishedRule = captured.map(RuleBindings.Retained(this, _))
    retainedRule = publishedRule
  private[framework] def capturePublished: Boolean =
    publishedRule.zip(retainedRule).exists((before, after) => before.eq(after))
  def fires(inputs: List[Any]): Boolean = values.forall(java.util.Objects.equals(_, inputs))

// The rules of one `on` block, which a derivation binds in its source's place.
final case class RuleGroup[S, O, F] private[framework] (rules: Vector[Rule[S, O, F]])
    extends Rebinding[S, O, F]:
  private[framework] var derivationOrigin = Option.empty[DerivationBindings.GroupOrigin]
  private var publishedOrigin = Option.empty[DerivationBindings.GroupOrigin]
  private[framework] def publishOrigin(value: Option[DerivationBindings.GroupOrigin]): Unit =
    require(
      publishedOrigin.isEmpty && derivationOrigin.isEmpty,
      "a RuleGroup retains origin once before publication"
    )
    publishedOrigin = value
    derivationOrigin = value
  private[framework] def originPublished: Boolean =
    publishedOrigin.zip(derivationOrigin).exists((before, after) => before.eq(after))

// What a machine binds one action to: a step function, or the rules that lower to one.
private[framework] enum Bound[S, O, F]:
  case Function(f: AnyRef, captured: Option[Captured[AnyRef]] = None)
  case Ruled(rules: Vector[Rule[S, O, F]])
  case Disabled()

  private[framework] var retainedDisabled = Option.empty[RuleBindings.Disabled]

// `action ~> stepFunction` receives the actual native input. Legacy tuple bindings remain below
// until caller migration; each form checks the callback's input types.
extension [I](a: Action[I])
  @scala.annotation.targetName("nativeBinding")
  infix def ~>[S, O, F](f: (S, I) => List[Step[S, O, F]]): StepBinding[S, O, F] =
    StepBinding(a.decl, f)

extension (a: Action[EmptyTuple])
  infix def ~>[S, O, F](
      @capture("capturedBinding0") f: S => List[Step[S, O, F]]
  ): StepBinding[S, O, F] = StepBinding(a.decl, f)
  private[framework] def capturedBinding0[S, O, F](
      f: S => List[Step[S, O, F]],
      root: String,
      chunks: Seq[String]
  ): StepBinding[S, O, F] = StepBinding.linking(a.decl, f, root, chunks)

extension [A](a: LegacyAction[A *: EmptyTuple])
  infix def ~>[S, O, F](
      @capture("capturedBinding1") f: (S, A) => List[Step[S, O, F]]
  ): StepBinding[S, O, F] =
    StepBinding(a.decl, f)
  private[framework] def capturedBinding1[S, O, F](
      f: (S, A) => List[Step[S, O, F]],
      root: String,
      chunks: Seq[String]
  ): StepBinding[S, O, F] = StepBinding.linking(a.decl, f, root, chunks)

extension [A, B](a: LegacyAction[(A, B)])
  infix def ~>[S, O, F](
      @capture("capturedBinding2") f: (S, A, B) => List[Step[S, O, F]]
  ): StepBinding[S, O, F] =
    StepBinding(a.decl, f)
  private[framework] def capturedBinding2[S, O, F](
      f: (S, A, B) => List[Step[S, O, F]],
      root: String,
      chunks: Seq[String]
  ): StepBinding[S, O, F] = StepBinding.linking(a.decl, f, root, chunks)

extension [A, B, C](a: LegacyAction[(A, B, C)])
  infix def ~>[S, O, F](
      @capture("capturedBinding3") f: (S, A, B, C) => List[Step[S, O, F]]
  ): StepBinding[S, O, F] =
    StepBinding(a.decl, f)
  private[framework] def capturedBinding3[S, O, F](
      f: (S, A, B, C) => List[Step[S, O, F]],
      root: String,
      chunks: Seq[String]
  ): StepBinding[S, O, F] = StepBinding.linking(a.decl, f, root, chunks)

extension [A, B, C, D](a: LegacyAction[(A, B, C, D)])
  infix def ~>[S, O, F](
      @capture("capturedBinding4") f: (S, A, B, C, D) => List[Step[S, O, F]]
  ): StepBinding[S, O, F] =
    StepBinding(a.decl, f)
  private[framework] def capturedBinding4[S, O, F](
      f: (S, A, B, C, D) => List[Step[S, O, F]],
      root: String,
      chunks: Seq[String]
  ): StepBinding[S, O, F] = StepBinding.linking(a.decl, f, root, chunks)

extension [A, B, C, D, E, G](a: LegacyAction[(A, B, C, D, E, G)])
  infix def ~>[S, O, F](
      @capture("capturedBinding6") f: (S, A, B, C, D, E, G) => List[Step[S, O, F]]
  ): StepBinding[S, O, F] =
    StepBinding(a.decl, f)
  private[framework] def capturedBinding6[S, O, F](
      f: (S, A, B, C, D, E, G) => List[Step[S, O, F]],
      root: String,
      chunks: Seq[String]
  ): StepBinding[S, O, F] = StepBinding.linking(a.decl, f, root, chunks)

// The owner of a machine's sections, which its `rules` read the machine's types from:
// `object rules extends Rules:` in `object OrderProduct extends Machine[...], Phased[...](_.phase)`,
// whose rules read the phase projection from the machine's `Phasing`.
final class Owner[S, O, F] private[framework] (val machine: Machine[S, O, F])

// The projection of a machine's or composition's state onto its phase, which its sections read as a
// given: `when(Phase.placed)` in its `rules` tests it. The object's `Phased[State, Phase](_.phase)`
// (model/framework/dsl/Syntax.scala) gives it, and a derived machine gives its source's, with the source's
// phase type.
final class Phasing[S, P] private[framework] (
    private[framework] val projection: S => P,
    private[framework] val writer: (S, P) => S = Phasing.Unwritten
):
  // Reads the phase a lifecycle reader tests, including a phase nested in a composed state.
  def phase(state: S): P = projection(state)

  // The state with its phase replaced by `p`, through the writer its `Phased` declares, which must
  // give a state the projection reads `p` of.
  private[framework] def write(state: S, p: P): S =
    val written = writer(state, p)
    require(
      java.util.Objects.equals(projection(written), p),
      s"the phase writer gives $written for $p, whose phase it does not read as $p: write the " +
        "field the projection reads, `Phased[State, Phase](_.phase, (s, p) => s.copy(phase = p))`"
    )
    written

  // The phase cases with a role, refused when none has it, naming the reader and the phase type.
  def roleCases[R](owner: String)(using
      finite: Finite[P],
      witness: TypeTest[P, R],
      phaseType: ClassTag[P],
      role: ClassTag[R]
  ): IndexedSeq[P] =
    framework.dsl.roleCases[P, R](owner, role.runtimeClass.getSimpleName)(using
      finite,
      witness,
      phaseType
    )

object Phasing:

  private[framework] object Unwritten extends ((Any, Any) => Nothing):
    def apply(state: Any, phase: Any): Nothing =
      throw IllegalStateException(
        s"enters($phase) writes the phase through the writer the machine's Phased declares, and it " +
          "declares none: `Phased[State, Phase](_.phase, (s, p) => s.copy(phase = p))`"
      )

  // The phasing of an object that declares no phase: its phase type is `Nothing`, so `when(...)` of
  // its rules does not compile, and its projection is never called.
  given unphased[S]: Phasing[S, Nothing] =
    Phasing(_ => throw IllegalStateException("this machine or composition declares no phase"))

// What a machine a derivation makes is made from: `m.rebind(...)` is made from `m`. Its static type
// keeps the source's object type, so a derived machine reads its source's phase.
trait DerivedFrom[+M]:
  private[framework] def source: M

// Evidence that `T`, the source of a derivation, projects its state `S` onto the phase type `P`:
// through its own `Phased` (whose companion gives that evidence), through the source it is derived
// from in turn, or, where it declares no phase, onto `Nothing`.
final class Inherited[-T, S, P] private[framework] (
    private[framework] val projection: T => S => P,
    private[framework] val end: T => S => Boolean,
    private[framework] val writer: T => (S, P) => S = (_: T) => Phasing.Unwritten
)

object Inherited extends Inherited.Unphased:
  // A derivation, `m.rebind(...)`: its source's.
  given derivation[S, P, M, T <: DerivedFrom[M]](using
      source: Inherited[M, S, P]
  ): Inherited[T, S, P] =
    Inherited(
      t => source.projection(t.source),
      t => source.end(t.source),
      t => source.writer(t.source)
    )

  // A derived machine, `object M extends Derived(m.rebind(...))`: its own, which is its source's.
  given derived[S, P, T <: Derived[S, ?, ?, P]]: Inherited[T, S, P] =
    Inherited(_.sourcePhasing.projection, _.end, _.sourcePhasing.writer)

  given composed[S <: Product, P, T <: DerivedComposition[S, P]]: Inherited[T, S, P] =
    Inherited(_.sourcePhasing.projection, _.end)

  // The evidence of a source that declares no phase, below every other: a given defined in an
  // object that extends this trait, such as `Phased`'s companion, is preferred to it.
  trait Unphased:
    // A source that declares no phase projects its state onto `Nothing`.
    given unphased[T, S]: Inherited[T, S, Nothing] =
      Inherited(
        _ => Phasing.unphased[S].projection,
        _ => _ => throw IllegalStateException("this derivation declares no phased stopping point")
      )

// What a `capabilities` section declares the capabilities of: the machine or
// composition whose object it sits in, with the outcomes and facts its steps answer and record.
final class Declaring[S, O, F] private[framework] (val model: Declares[S])

// What a machine binds each action to, in the order the first binding of each names it: a step
// function, rules, or nothing. A machine's `rules` (model/framework/dsl/Syntax.scala) is the one a Model
// writes; a derivation keeps what it makes as built, and the core `Bindings` binds step functions.
abstract class RuleBook[S, O, F]:
  private[framework] def table: Vector[(ActionDecl, Bound[S, O, F])]
  private[framework] def facetSource: Option[String] = None
  private[framework] def sectionName: String = "<rules>"
  private[framework] def actionName(decl: ActionDecl): String = decl.toString
  private[framework] def facetOrigin: Option[FacetBindings.Origin] = None

// A machine's rules in the core: one step function per action, bound by hand,
// `object rules extends Bindings(clerk.ship ~> shipStep, courier.strike ~> (_ => Nil))`, which is
// what `Rules` lower to. It is the spelling of the IR generator's core fixtures, whose step
// functions are written out; a Model says when each action fires in `Rules`, and the IR generator
// refuses a hand-bound step function in a Model (fn-126 R17).
abstract class Bindings[S, O, F](using @unused owner: Owner[S, O, F])(
    bindings: StepBinding[S, O, F]*
) extends RuleBook[S, O, F]:
  private[framework] def table: Vector[(ActionDecl, Bound[S, O, F])] =
    bindings.toVector.map(b => b.decl -> Bound.Function[S, O, F](b.function, b.captured))

// A step function's results for the state and one class's inputs in declaration order, whatever
// the function's arity: `f(s, inputs*)`, the function `action ~> f` binds.
private[framework] trait InputEffect[S, O, F] extends ((S, List[Any]) => List[Step[S, O, F]]):
  def apply(state: S, cls: Class): List[Step[S, O, F]] = apply(state, cls.values)

private[framework] def effectOf[S, O, F](
    decl: ActionDecl,
    f: AnyRef
): InputEffect[S, O, F] =
  decl.nativeInputs match
    case Some(shape) => nativeEffect(decl, shape, f)
    case None        =>
      val run = legacyEffect[S, O, F](decl, f)
      new InputEffect[S, O, F]:
        def apply(state: S, values: List[Any]): List[Step[S, O, F]] = run(state, values)

private[framework] def nativeEffect[S, O, F, I](
    decl: ActionDecl,
    shape: ActionInputs[I],
    f: AnyRef
): InputEffect[S, O, F] =
  val callback =
    f.asInstanceOf[(S, I) => List[Step[S, O, F]]] // scalafix:ok DisableSyntax.asInstanceOf
  new InputEffect[S, O, F]:
    def apply(state: S, values: List[Any]): List[Step[S, O, F]] = apply(state, Class(decl, values))
    override def apply(state: S, cls: Class): List[Step[S, O, F]] =
      callback(state, shape.input(cls))

private def legacyEffect[S, O, F](
    decl: ActionDecl,
    f: AnyRef
): (S, List[Any]) => List[Step[S, O, F]] =
  // scalafix:off DisableSyntax.asInstanceOf
  type R = List[Step[S, O, F]]
  decl.domains.size match
    case 0 => (s, _) => f.asInstanceOf[S => R](s)
    case 1 => (s, i) => f.asInstanceOf[(S, Any) => R](s, i(0))
    case 2 => (s, i) => f.asInstanceOf[(S, Any, Any) => R](s, i(0), i(1))
    case 3 => (s, i) => f.asInstanceOf[(S, Any, Any, Any) => R](s, i(0), i(1), i(2))
    case 4 => (s, i) => f.asInstanceOf[(S, Any, Any, Any, Any) => R](s, i(0), i(1), i(2), i(3))
    case 5 =>
      (s, i) => f.asInstanceOf[(S, Any, Any, Any, Any, Any) => R](s, i(0), i(1), i(2), i(3), i(4))
    case 6 =>
      (s, i) =>
        f.asInstanceOf[(S, Any, Any, Any, Any, Any, Any) => R](
          s,
          i(0),
          i(1),
          i(2),
          i(3),
          i(4),
          i(5)
        )
    case n =>
      throw IllegalArgumentException(s"a step function supports at most six inputs, got $n")
  // scalafix:on DisableSyntax.asInstanceOf

// The step function an action's binding lowers to, of the action's arity: its own, or for rules the
// effect of the first rule that fires, as `if g1(s) then e1(s) else if g2(s) then e2(s) else Nil`
// lifts, or none where it is disabled.
private[framework] def stepFunction[S, O, F](decl: ActionDecl, bound: Bound[S, O, F]): AnyRef =
  def run(s: S, cls: Class): List[Step[S, O, F]] = bound match
    case Bound.Ruled(rules) =>
      rules
        .find(r => r.fires(cls.values) && r.guard(s))
        .fold(Nil): rule =>
          rule.effect match
            case effect: InputEffect[S, O, F] @unchecked => effect(s, cls)
            case effect                                  => effect(s, cls.values)
    case _ => Nil
  bound match
    case Bound.Function(f, _)            => f
    case _ if decl.nativeInputs.nonEmpty =>
      def adapted[I](shape: ActionInputs[I]): (S, I) => List[Step[S, O, F]] =
        (state, input) => run(state, shape.retained(decl, input))
      adapted(decl.nativeInputs.get)
    case _ =>
      def legacy(s: S, inputs: List[Any]): List[Step[S, O, F]] = run(s, Class(decl, inputs))
      decl.domains.size match
        case 0 => (s: S) => legacy(s, Nil)
        case 1 => (s: S, a: Any) => legacy(s, List(a))
        case 2 => (s: S, a: Any, b: Any) => legacy(s, List(a, b))
        case 3 => (s: S, a: Any, b: Any, c: Any) => legacy(s, List(a, b, c))
        case 4 => (s: S, a: Any, b: Any, c: Any, d: Any) => legacy(s, List(a, b, c, d))
        case 5 => (s: S, a: Any, b: Any, c: Any, d: Any, e: Any) => legacy(s, List(a, b, c, d, e))
        case 6 =>
          (s: S, a: Any, b: Any, c: Any, d: Any, e: Any, f: Any) =>
            legacy(s, List(a, b, c, d, e, f))
        case n =>
          throw IllegalArgumentException(s"a step function supports at most six inputs, got $n")

// Every class of an action: each assignment of its inputs, in catalog order.
private[framework] def classesOf(decl: ActionDecl): List[List[Any]] =
  decl.nativeInputs.fold(
    decl.domains.foldLeft(List(List.empty[Any])): (prefixes, domain) =>
      for prefix <- prefixes; v <- domain.values.toList yield prefix :+ v
  )(_.classes)

private[framework] def inputClassesOf(decl: ActionDecl): List[Class] =
  decl.nativeInputs.fold(classesOf(decl).map(Class(decl, _)))(_.inputClasses(decl))

// A value as Umpire keys it in a class: a case by its name, followed by its fields, joined by `-`.
private[framework] def valueKey(v: Any): String = v match
  case p: Product if p.productArity > 0 =>
    (p.productPrefix +: p.productIterator.map(valueKey).toList).mkString("-")
  case other => other.toString

// The refusal of two rules of one action class of `machine` that both fire in some state, naming
// the class, both rules by their place and heading, and the first such state in catalog order: rules
// say when an action fires, so no two of one class fire together (intended alternatives are one
// effect's `choose`). `added` is checked against each rule of `rules`.
private[framework] def overlap[S, O, F](
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
private[framework] def writtenAction(code: String): String =
  writtenPath(code).split('.').last

// The reference to the action a rule's source names, as written: `clerk.ship`, the action
// `buyer.change(Change.hold)` applies, or a bare `ship`.
private[framework] def writtenPath(code: String): String =
  // The type arguments of a class written by name, `apply[(A, B)](action)(...)`, name no action.
  val untyped = code.replaceAll("\\[[^\\]]*\\]", "")
  val path = "[A-Za-z_][A-Za-z0-9_$]*(?:\\.[A-Za-z_][A-Za-z0-9_$]*)*".r
  path
    .findAllIn(untyped)
    .map(_.stripSuffix(".apply"))
    .find(p => p.nonEmpty && p != "framework" && p != "framework.dsl" && p != "apply")
    .getOrElse(code)

// A machine: a transition relation over the finite state type `S` with outcomes `O` and facts `F`.
// Declared as an object that is the machine, named after the object with its first letter lowered:
//
// {{{
// object OrderProduct extends Machine[OrderState, Outcome, OrderFact]:
//   val init = OrderState(placed)
//   def end(s: OrderState) = states.terminal(s.phase)
//   object states: ...
//   object effects: ...
//   object rules extends Rules: ...
// }}}
//
// A machine whose rules name phases mixes in its phase projection, `extends Machine[OrderState,
// Outcome, OrderFact], Phased[OrderState, Phase](_.phase)` (model/framework/dsl/Syntax.scala), which its
// sections read; a machine that names no phase is a plain `Machine`.
//
// Its members are its header, the state it starts in, `init`, the states it may end in, `end`, and
// where it declares them, `entity` and `evidence`; then its sections, objects named after what they
// hold, in this order: `states`, its vocabulary (the named sets and projections of its states);
// `refinement`, where it refines another machine (framework.dsl.Refinement); `effects`; `monitors`;
// `rules`, which it must declare; `properties`; `capabilities`, its capabilities
// `framework.dsl.Capabilities`; and `queries`. Its `entity` is the one entity its
// actions are `on`, or create, unless it names another. The IR generator (model/framework/export) reads them
// from the source, and names each of its declarations after where it is declared: its package and
// the objects it sits in. At run time a machine is its init, its end and the step function each
// action's rules lower to: `rules` is the one section the machine itself reads, through the member
// it mixes in, so no section is found by reflection, and a section object, which initializes on
// its first use, is never read while the machine initializes.
//
// A member of `states` or `effects` is a def of the state, `def held(s: OrderState) =
// s.phase.in(…)`, `def ship(s: OrderState) = enter(s.copy(phase = shipped), Fact.statusShipped)`,
// or a `val` written as a block that reads, and in an effect assigns, the state's fields by name
// (model/framework/dsl/Syntax.scala): `val held = is { phase.in(…) }`, `val ship = effect { phase =
// shipped }`, which records the status the assigned case declares, where its enum declares one.
abstract class Machine[S, O, F](using
    private[framework] val fs: Finite[S],
    private[framework] val fo: Finite[O],
    private[framework] val ff: Finite[F]
) extends Declares[S]:
  type Outcome = O
  type Fact = F

  private val headerMetadata = new MachineHeaders.Metadata
  private val endpointData = new MachineEndpoints.Metadata
  private val evidenceData = new MachineEvidence.Metadata
  private val facetContextData = new FacetBindings.Metadata
  private var facetNamespaces = Vector.empty[(AnyRef, String, String, java.lang.Class[?])]
  @publicInBinary final private[framework] def retainFacetNamespace(
      namespace: AnyRef,
      traitName: String,
      namespaceName: String,
      traitClass: java.lang.Class[?]
  ): Unit = synchronized {
    require(
      namespace != null && traitName.nonEmpty && namespaceName.nonEmpty,
      "Machine Facets: missing namespace receiver identity"
    )
    require(
      (traitClass ne null) && traitClass.isInterface && traitClass.isInstance(this),
      "Machine Facets: namespace trait token does not own this assembly"
    )
    require(
      facetNamespaces.forall { (reference, owner, name, token) =>
        if reference.eq(namespace) || (owner == traitName && name == namespaceName) then
          reference.eq(namespace) && owner == traitName && name == namespaceName && token.eq(
            traitClass
          )
        else true
      },
      "Machine Facets: conflicting namespace receiver identity"
    )
    if !facetNamespaces.exists((reference, _, _, _) => reference.eq(namespace)) then
      facetNamespaces :+= ((namespace, traitName, namespaceName, traitClass))
  }
  final private[framework] def ownsFacetNamespace(
      namespace: AnyRef,
      traitName: String,
      namespaceName: String,
      traitClass: java.lang.Class[?]
  ): Boolean =
    facetNamespaces.exists((reference, owner, name, token) =>
      reference.eq(namespace) && owner == traitName && name == namespaceName && token.eq(traitClass)
    )
  private[framework] def facetContextMetadata: Option[FacetBindings.Context] =
    facetContextData.value
  @publicInBinary final private[framework] def retainFacetContext(data: Seq[String]): Unit =
    facetContextData.retain(this, data)
  private[framework] def endpointSource: Machine[?, ?, ?] = this
  private[framework] def evidenceSource: Machine[?, ?, ?] = this
  private[framework] def evidenceMetadata: Option[MachineEvidence.Retained] = evidenceData.value
  final private[framework] def ownsEvidence(value: MachineEvidence.Retained): Boolean =
    evidenceData.owns(value)
  @publicInBinary final private[framework] def retainEvidence(data: Seq[String]): Unit =
    evidenceData.retain(this, data)
  private[framework] def ruleSource: Machine[?, ?, ?] = this
  private[framework] def endpointMetadata: Option[MachineEndpoints.Retained] = endpointData.value
  @publicInBinary final private[framework] def retainEndpoints(
      owner: String,
      family: String,
      name: String,
      starts: Seq[String],
      end: Seq[String],
      closure: Seq[String]
  ): Unit = endpointData.retain(this, owner, family, name, starts, end, closure)
  private[framework] def headerIdentity: Option[(String, String)] = headerMetadata.identity
  private[framework] def headerSchemaKeys: Vector[Option[String]] = headerMetadata.keys
  @publicInBinary final private[framework] def retainHeaderIdentity(
      family: String,
      name: String
  ): Unit =
    headerMetadata.retainIdentity(family, name)
  @publicInBinary final private[framework] def retainHeaderSchema(
      index: Int,
      key: String,
      root: Seq[String],
      closure: Seq[String],
      inboxes: Seq[String]
  ): Unit =
    headerMetadata.retainSchema(Vector(fs, fo, ff), index, key, root, closure, inboxes)

  private var declaredFactCases = Vector.empty[(AnyRef, String)]
  private[framework] def factCases: Vector[(AnyRef, String)] = declaredFactCases
  @publicInBinary private[framework] def declaredFact(reference: AnyRef, spelling: String): Unit =
    declaredFactCases.find((value, _) => value eq reference) match
      case Some((_, prior)) => require(prior == spelling, s"conflicting fact case $spelling")
      case None             => declaredFactCases = declaredFactCases :+ (reference -> spelling)

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

  // What its `capabilities` section declares the capabilities of: this machine.
  protected given declaring: Declaring[S, O, F] = Declaring(this)

  // What each action is bound to, in the order its first rule names it.
  private var acquiredBook = Option.empty[RuleBook[S, O, F]]
  private var acquiredTable = Option.empty[DerivationBindings.Snapshot]
  private[framework] def acquiredRuleBook: Option[RuleBook[?, ?, ?]] = acquiredBook
  final private[framework] def acquireRuleBook(): RuleBook[S, O, F] =
    acquiredBook.getOrElse {
      val book = rules
      acquiredBook = Some(book)
      book
    }
  private[framework] def acquiredRuleTable: Option[DerivationBindings.Snapshot] = acquiredTable
  final private[framework] def acquireRuleTable(
      table: Vector[(ActionDecl, Bound[S, O, F])]
  ): Vector[(ActionDecl, Bound[S, O, F])] =
    acquiredTable = Some(DerivationBindings.snapshot(table))
    table
  private[framework] def derivationParent: Option[Machine[?, ?, ?]] = None
  private[framework] def derivationEvent: Option[DerivationBindings.Event] = None
  private[framework] def table: Vector[(ActionDecl, Bound[S, O, F])] =
    val book = rules
    acquiredBook = Some(book)
    acquireRuleTable(book.table)

  // The machines this one is made from, which constructing it constructs as well
  // (IrFile.construct): the machine its `object refinement` refines, and for a derivation its
  // source and the machine a `refining` names.
  private[framework] def refinementEvidence: Option[Refinement[S, ?]] =
    Refinement
      .declaredBy(this)
      .map(_.asInstanceOf[Refinement[S, ?]]) // scalafix:ok DisableSyntax.asInstanceOf

  private[framework] def phaseEvidence: Option[S => Any] =
    if !declaresPhase then None
    else
      getClass.getMethod("phased").invoke(this) match
        case phasing: Phasing[?, ?] =>
          Some(
            phasing.asInstanceOf[Phasing[S, ?]].projection // scalafix:ok DisableSyntax.asInstanceOf
          )
        case other =>
          throw IllegalStateException(s"$name declares an invalid phase projection: $other")

  private[framework] def reaches: Seq[Model] = refinementEvidence.toSeq.map(_.of)

  // One step function per action: the rules of each, lowered (Rules.lowered).
  private[framework] def bindings: List[StepBinding[S, O, F]] =
    table.map { (decl, bound) =>
      val captured = bound match
        case Bound.Function(_, captured) => captured
        case _                           => None
      StepBinding.retaining[S, O, F](decl, stepFunction(decl, bound), captured)
    }.toList

  private[framework] def coreOperations: Vector[CoreBindings.Operation] = Vector.empty

  private def built(
      table: => Vector[(ActionDecl, Bound[S, O, F])],
      keepRefinement: Boolean = true,
      operation: => Option[CoreBindings.Operation] = None,
      trace: => Option[DerivationBindings.Event] = None
  ): Machine[S, O, F] & DerivedFrom[this.type] = // format: off
    Built[S, O, F, this.type](objectName(this), List(init), end, table, this, Seq.empty,
      if keepRefinement then refinementEvidence else None, operation.toVector, trace)(using // format: on
      fs,
      fo,
      ff
    )

  // A machine that keeps the rows of the named actions and drops the rest, named after the object
  // that declares it, `object M extends Derived(m.restrict(...))`. It keeps the state type, starts
  // and ends, owns its own name and Definition IDs, and does not inherit a refinement.
  def restrict(keep: Action[?]*): Machine[S, O, F] & DerivedFrom[this.type] =
    val decls = keep.map(_.decl).toSet
    // It keeps its source's monitors and assumptions, which are about the state and the machine.
    lazy val before = DerivationBindings.snapshot(table)
    built(
      before.table.asInstanceOf[Vector[(ActionDecl, Bound[S, O, F])]].filter((d, _) => decls(d)),
      keepRefinement = false,
      trace = Some(DerivationBindings.Restricted(this, before, keep.map(_.decl).toVector))
    )

  // A machine that binds other step functions or rules to actions this one binds, each in its
  // place, and keeps everything else this machine declares. `action ~> effect` keeps that action's
  // rules, each with its guard and class, and replaces their effect, which the lifter refuses where
  // its whole-action rules have several effects (rules that each fire one class may differ, as the
  // new effect reads the class's inputs); `on(action) { where(g) ~> effect }` replaces its rules.
  // The lifter refuses an action this machine does not bind.
  def rebind(
      replaced: (Owner[S, O, F] ?=> Rebinding[S, O, F])*
  ): Machine[S, O, F] & DerivedFrom[this.type] =
    rebindWith(replaced, None)

  @publicInBinary private[framework] def capturedRebind(
      replaced: (Owner[S, O, F] ?=> Rebinding[S, O, F])*
  )(data: Seq[String]): Machine[S, O, F] & DerivedFrom[this.type] =
    rebindWith(replaced, Some(DerivationBindings.operationSite(this, data)))

  private def rebindWith(
      replaced: Seq[Owner[S, O, F] ?=> Rebinding[S, O, F]],
      site: Option[DerivationBindings.OperationSite]
  ): Machine[S, O, F] & DerivedFrom[this.type] =
    val operands = Vector.newBuilder[Rebinding[S, O, F]]
    val evidence = Vector.newBuilder[DerivationBindings.OperandEvidence]
    val nativeSources = Vector.newBuilder[DerivationBindings.Snapshot]
    lazy val beforeSnapshot = DerivationBindings.snapshot(table)
    lazy val before = beforeSnapshot.table.asInstanceOf[Vector[(ActionDecl, Bound[S, O, F])]]
    val by = replaced.foldLeft(Map.empty[ActionDecl, Bound[S, O, F]]): (by, r) =>
      val resolved = r(using machineOwner)
      operands += resolved
      evidence += DerivationBindings.operandEvidence(resolved)
      resolved match
        case binding @ StepBinding(decl, f) =>
          val inspected = table
          nativeSources += DerivationBindings.snapshot(inspected)
          val kept = inspected.collectFirst { case (`decl`, Bound.Ruled(rs)) => rs }
          by.updated(
            decl,
            kept.fold(Bound.Function[S, O, F](f, binding.captured))(rs =>
              Bound.Ruled(rs.map { original =>
                val callback = effectOf[S, O, F](decl, f)
                val adapted = original.copy(effect = callback)
                adapted.adaptedRule =
                  Some(DerivationBindings.Adaptation(adapted, original, binding, callback))
                adapted
              })
            )
          )
        case RuleGroup(rules) =>
          rules.groupBy(_.decl).foldLeft(by)((by, g) => by.updated(g._1, Bound.Ruled(g._2)))
    val retained = operands.result()
    val retainedEvidence = evidence.result()
    val nativeEvidence = DerivationBindings.snapshot(by.toVector)
    lazy val operation = CoreBindings.operation(
      "rebind",
      before,
      retained,
      Some(this),
      Some(beforeSnapshot),
      retainedEvidence,
      Some(nativeEvidence),
      site,
      nativeSources.result()
    )
    built(
      before.map((d, b) => d -> by.getOrElse(d, b)),
      operation = Some(operation),
      trace = Some(DerivationBindings.Applied(operation))
    )

  // A machine that also binds actions this one does not, after its own bindings, by rules:
  // `extend(on(action) { where(g) ~> effect })`. The lifter refuses an action this machine binds
  // already, and a bare binding where this machine's actions are bound by rules.
  def extend(
      added: (Owner[S, O, F] ?=> Rebinding[S, O, F])*
  ): Machine[S, O, F] & DerivedFrom[this.type] =
    extendWith(added, None)

  @publicInBinary private[framework] def capturedExtend(
      added: (Owner[S, O, F] ?=> Rebinding[S, O, F])*
  )(data: Seq[String]): Machine[S, O, F] & DerivedFrom[this.type] =
    extendWith(added, Some(DerivationBindings.operationSite(this, data)))

  private def extendWith(
      added: Seq[Owner[S, O, F] ?=> Rebinding[S, O, F]],
      site: Option[DerivationBindings.OperationSite]
  ): Machine[S, O, F] & DerivedFrom[this.type] =
    val acquired = added.map { r =>
      val operand = r(using machineOwner)
      operand -> DerivationBindings.operandEvidence(operand)
    }.toVector
    val operands = acquired.map(_._1)
    lazy val beforeSnapshot = DerivationBindings.snapshot(table)
    lazy val before = beforeSnapshot.table.asInstanceOf[Vector[(ActionDecl, Bound[S, O, F])]]
    val more = operands.flatMap(r =>
      r match
        case binding @ StepBinding(decl, f) =>
          Vector(decl -> Bound.Function[S, O, F](f, binding.captured))
        case RuleGroup(rules) => rules.groupBy(_.decl).toVector.map((d, rs) => d -> Bound.Ruled(rs))
    )
    val nativeEvidence = DerivationBindings.snapshot(more)
    lazy val operation = CoreBindings.operation(
      "extend",
      before,
      operands,
      Some(this),
      Some(beforeSnapshot),
      acquired.map(_._2),
      Some(nativeEvidence),
      site
    )
    built(
      before ++ more,
      operation = Some(operation),
      trace = Some(DerivationBindings.Applied(operation))
    )

  // A machine that refines `product` through `map` in place of the refinement this one declares,
  // and keeps the facts and outcomes that refinement lets the refined machine see.
  def refining[PS, PO, PF](product: Machine[PS, PO, PF])(
      map: S => PS
  ): Machine[S, O, F] & DerivedFrom[this.type] =
    val refinement = new Refinement[S, PS](using machineOwner)(product):
      def toProduct(s: S): PS = map(s)
    Built[S, O, F, this.type](
      objectName(this),
      List(init),
      end,
      table,
      this,
      Seq(product),
      Some(refinement)
    )(using fs, fo, ff)

  // A machine whose checks also make the assumptions named, each once, after this one's.
  def assuming(@unused added: Assumption*): Machine[S, O, F] & DerivedFrom[this.type] =
    built(table)

  // A machine without this one's monitors and refinement: the same transitions, for a composition
  // a member's monitors may not watch (model/SEMANTICS.md leaves that undefined).
  def unmonitored: Machine[S, O, F] & DerivedFrom[this.type] = built(table, keepRefinement = false)

// A machine derived from another, declared as an object that is it:
// `object LateRecord extends Derived(OrderRecord.rebind(...))`, named after the object. The
// derivation is an expression of the core operations `rebind`, `extend`, `restrict`, `refining`,
// `assuming` and `unmonitored`; the object adds only the sections that are its own, `states`,
// `properties`, `capabilities` and `queries`, and reuses its source's effects by name. It mixes in no
// `Phased` of its own: its sections read its source's phase projection, with its phase type.
//
// Its phase type `P` is its source's, read off the derivation's static type, `Nothing` where the
// source declares no phase.
abstract class Derived[S, O, F, P](derivation: Machine[S, O, F])(using
    source: Inherited[derivation.type, S, P]
) extends Machine[S, O, F](using derivation.fs, derivation.fo, derivation.ff):
  final override private[framework] def derivationParent: Option[Machine[?, ?, ?]] = Some(
    derivation
  )
  final override private[framework] def acquiredRuleTable: Option[DerivationBindings.Snapshot] =
    derivation.acquiredRuleTable
  final override private[framework] def endpointSource: Machine[?, ?, ?] = derivation.endpointSource
  final override private[framework] def evidenceSource: Machine[?, ?, ?] = derivation.evidenceSource
  final override private[framework] def ruleSource: Machine[?, ?, ?] = derivation.ruleSource
  override private[framework] def headerSchemaKeys: Vector[Option[String]] =
    super.headerSchemaKeys
      .zip(derivation.headerSchemaKeys)
      .map((own, inherited) => own.orElse(inherited))
  override private[framework] def factCases: Vector[(AnyRef, String)] =
    super.factCases ++ derivation.factCases
  require(
    !declaresPhase,
    s"$name is derived and keeps its source's phase: it mixes in no Phased of its own"
  )

  // The phase projection its sections read: its source's, with its phase type.
  private[framework] def sourcePhasing: Phasing[S, P] =
    Phasing(source.projection(derivation), source.writer(derivation))

  protected given derivedPhasing: Phasing[S, P] = sourcePhasing

  final def init: S = derivation.init
  final def end(s: S): Boolean = derivation.end(s)
  final def rules: RuleBook[S, O, F] = derivation.rules
  final override private[framework] def table: Vector[(ActionDecl, Bound[S, O, F])] =
    derivation.table
  final override private[framework] def coreOperations: Vector[CoreBindings.Operation] =
    derivation.coreOperations
  final override private[framework] def reaches: Seq[Model] =
    derivation +: refinementEvidence.toSeq.map(_.of)
  final override private[framework] def refinementEvidence: Option[Refinement[S, ?]] =
    super.refinementEvidence.orElse(derivation.refinementEvidence)
  final override private[framework] def phaseEvidence: Option[S => Any] = derivation.phaseEvidence

// A machine a derivation makes from another: its name, its starts and ends, what it binds each
// action to, its source and the other machines it is made from. What else it declares only the
// lifter reads.
final private[framework] class Built[S, O, F, M <: Model](
    override val name: String,
    starts: List[S],
    isEnd: S => Boolean,
    bound: => Vector[(ActionDecl, Bound[S, O, F])],
    private[framework] val source: M,
    also: Seq[Model],
    refinement: Option[Refinement[S, ?]],
    operations: => Vector[CoreBindings.Operation] = Vector.empty,
    trace: => Option[DerivationBindings.Event] = None
)(using Finite[S], Finite[O], Finite[F])
    extends Machine[S, O, F],
      DerivedFrom[M]:
  override private[framework] def derivationParent: Option[Machine[?, ?, ?]] = source match
    case machine: Machine[?, ?, ?] => Some(machine)
    case _                         => None
  override private[framework] lazy val derivationEvent: Option[DerivationBindings.Event] = trace
  override private[framework] def endpointSource: Machine[?, ?, ?] = source match
    case machine: Machine[?, ?, ?] => machine.endpointSource
    case _                         => super.endpointSource
  override private[framework] def evidenceSource: Machine[?, ?, ?] = source match
    case machine: Machine[?, ?, ?] => machine.evidenceSource
    case _                         => super.evidenceSource
  override private[framework] def ruleSource: Machine[?, ?, ?] = source match
    case machine: Machine[?, ?, ?] => machine.ruleSource
    case _                         => super.ruleSource
  override private[framework] def headerSchemaKeys: Vector[Option[String]] = source match
    case machine: Machine[?, ?, ?] => machine.headerSchemaKeys
    case _                         => super.headerSchemaKeys
  override private[framework] def factCases: Vector[(AnyRef, String)] = source match
    case machine: Machine[?, ?, ?] => super.factCases ++ machine.factCases
    case _                         => super.factCases
  def init: S = starts.head
  def end(s: S): Boolean = isEnd(s)

  object rules extends RuleBook[S, O, F]:
    private[framework] def table: Vector[(ActionDecl, Bound[S, O, F])] = Vector.empty
  final override private[framework] lazy val table: Vector[(ActionDecl, Bound[S, O, F])] =
    acquireRuleTable(bound)
  final override private[framework] lazy val coreOperations: Vector[CoreBindings.Operation] =
    source match
      case machine: Machine[?, ?, ?] => machine.coreOperations ++ operations
      case _                         => operations
  final override private[framework] def reaches: Seq[Model] = source +: also
  final override private[framework] def refinementEvidence: Option[Refinement[S, ?]] = refinement
  final override private[framework] def phaseEvidence: Option[S => Any] = source match
    case machine: Machine[?, ?, ?] =>
      machine.asInstanceOf[Machine[S, ?, ?]].phaseEvidence // scalafix:ok DisableSyntax.asInstanceOf
    case _ => None
