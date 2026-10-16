// Sugar: definitions whose meaning a core declaration already expresses, kept for readability. Each
// names the core form it stands for. Marked definitions expand through the shared typed index;
// declaration witnesses, rules and blocks still await constructed-value export. Among them are the
// block forms a machine object's section `val`s are written in, `is:` for a predicate of the
// state, `expect:` for requirements on its resulting step, and `effect:` for a step function, with `record` and
// `reject(outcome)` statements, the `View` and `Draft` the field accessors of the state read and
// assign through, and `Recorded`, the status fact a phase enum's case declares.
package framework.dsl

import scala.annotation.{implicitNotFound, targetName, unused}
import scala.collection.mutable
import scala.compiletime.{codeOf, error}
import scala.reflect.{ClassTag, TypeTest}
import scala.util.NotGiven
import scala.language.implicitConversions

// The outcome `enter` and `stay` answer for a machine whose outcomes are `O`, declared once beside
// the outcome type: `given Ok[Outcome] = Ok(Outcome.accepted)`. Core form: the outcome itself,
// `Outcome.accepted`, written in each `Step`.
final case class Ok[O](outcome: O)

// The ok outcomes the framework declares itself, in the implicit scope of every `Ok`: a Model's own
// `given Ok` outranks them, and a machine on the shared outcomes declares none. Core form: the
// given beside the outcome type, `given Ok[Outcome] = Ok(Outcome.accepted)`.
object Ok:
  // `accepted`, the ok outcome of the shared outcomes (`outcomes`). Core form:
  // `given Ok[Outcome] = Ok(Outcome.accepted)`.
  given Ok[outcomes.Outcome] = Ok(outcomes.Outcome.accepted)

// One step with the ok outcome into `state`, recording `facts`. Core form:
// `List(Step(Outcome.accepted, state, List(facts*)))`.
@sugar def enter[S, O, F](state: S, facts: F*)(using ok: Ok[O]): List[Step[S, O, F]] =
  List(Step(ok.outcome, state, facts.toList))

// One step with the ok outcome that keeps the state and records nothing. Core form:
// `List(Step(Outcome.accepted, s))`.
@sugar def stay[S, O, F](s: S)(using ok: Ok[O]): List[Step[S, O, F]] =
  List(Step(ok.outcome, s))

// One step with the outcome `outcome` that keeps the state and records nothing, such as a request
// the machine refuses: `reject(Outcome.notFound, s)`. Core form: `List(Step(Outcome.notFound, s))`.
@sugar def reject[S, O](outcome: O, s: S): List[Step[S, O, Nothing]] = List(
  Step(outcome, s)
)

// No step: the action is disabled here. Core form: `Nil`.
@sugar val disabled: List[Nothing] = Nil

// The outcomes shared by every machine whose steps answer a request, accepted or rejected for a
// reason, which a machine adopts by name, `import framework.dsl.outcomes.{Outcome, Rejection}`, in place of
// its own. `import framework.*` does not open it, so a Model's own `Outcome` stays its own. Core form: a
// machine's own outcome enum, `enum Outcome: case accepted, notFound`.
object outcomes:
  // Why a request is rejected, loosely after the gRPC status codes a realization maps it to: the
  // entity is not there (`notFound`), a create collides with one that is (`alreadyExists`), its
  // state forbids the request (`failedPrecondition`), or the request is bad in every state
  // (`invalidArgument`). Core form: a case of a machine's own outcome enum, `case notFound`.
  enum Rejection:
    case notFound, alreadyExists, failedPrecondition, invalidArgument

  private[framework] def foldRejection[A](value: Any)(
      notFound: => A,
      alreadyExists: => A,
      failedPrecondition: => A,
      invalidArgument: => A
  ): Option[A] =
    given CanEqual[Rejection, Rejection] = CanEqual.derived
    value match
      case reason: Rejection =>
        reason match
          case Rejection.notFound           => Some(notFound)
          case Rejection.alreadyExists      => Some(alreadyExists)
          case Rejection.failedPrecondition => Some(failedPrecondition)
          case Rejection.invalidArgument    => Some(invalidArgument)
      case _ => None

  // What a step answers: the request is accepted, or rejected for the reason it names. Core form: a
  // machine's own outcome enum, `enum Outcome: case accepted, notFound`.
  enum Outcome:
    case accepted
    case rejected(why: Rejection)

// `when(...) ~> rejects(Rejection.notFound)`, in a rule of a machine on the shared outcomes: the
// effect that keeps the state, records nothing and answers `rejected(why)`, and
// `rejects(why).because("...")` the same effect with the server's explanation. Core form:
// `reject(Outcome.rejected(Rejection.notFound), s)`.
def rejects[S](why: outcomes.Rejection)(using
    @implicitNotFound(
      "rejects answers the shared framework.dsl.outcomes.Outcome, so it is an effect of a rule, in an `on` " +
        "block, of a machine on the shared outcomes: `import framework.dsl.outcomes.{Outcome, Rejection}`"
    ) @unused firing: Firing[S, outcomes.Outcome, ?, ?]
): Rejects[S] = Rejects(why)

// The effect `rejects(why)` writes: a function of the state alone, as every effect a rule names is.
// Core form: `(s: S) => reject(Outcome.rejected(why), s)`.
final class Rejects[S] private[framework] (why: outcomes.Rejection)
    extends (S => List[Step[S, outcomes.Outcome, Nothing]]):
  def apply(s: S): List[Step[S, outcomes.Outcome, Nothing]] =
    List(Step(outcomes.Outcome.rejected(why), s))

  // The same effect, its step explained by `reason`, as the IR row's `because` carries it. It
  // explains once: what it returns has no `because`. Core form:
  // `reject(Outcome.rejected(why), s).because(reason)`.
  def because(reason: String): S => List[Step[S, outcomes.Outcome, Nothing]] =
    s => List(Step(outcomes.Outcome.rejected(why), s, Nil, reason))

// The state an `is`, `effect` or `expect` block reads its fields of, by the accessors its state type
// declares, each of one shape: `def phase(using v: View[State]): Phase = v.get(_.phase)`.
// Core form: the state parameter of the predicate or step function, `s` in `s.phase`.
@implicitNotFound(
  "the fields of ${S} are read inside `is`, `effect` or `expect` blocks, which give the state they read: " +
    "outside a block, take the state as a parameter, `def f(s: State) = s.phase`"
)
sealed trait View[S]:
  private[framework] def state: S

  // The value of one field of the state: `v.get(_.phase)`. Core form: `s.phase`.
  def get[A](field: S => A): A = field(state)

// The view an `is:` block reads: the state it is asked of. Core form: the predicate's parameter.
final private class Fixed[S](private[framework] val state: S) extends View[S]

// The step an expectation checks; bare fields read its resulting state. Core form:
// the parameter `s` of `(s: Step[S, O, F]) => predicate(s)`.
final class Expectation[S, O, F] private[framework] (val step: Step[S, O, F]) extends View[S]:
  private[framework] def state: S = step.state
  private val failure = new scala.util.control.ControlThrowable {}
  private[framework] def require(condition: Boolean): Unit = if !condition then throw failure
  private[framework] def evaluate(body: Expectation[S, O, F] ?=> Unit): Boolean =
    try
      body(using this)
      true
    catch case signal: scala.util.control.ControlThrowable if signal eq failure => false

// The earlier state, available only in a transition expectation. Core form:
// `b` in `(b: S, a: Step[S, O, F]) => predicate(b, a)`.
final class Before[S] private[framework] (val state: S)

// The typed step whose requirements an expectation reads. Core form: `s` in `s.facts`.
def after[S, O, F](using context: Expectation[S, O, F]): Step[S, O, F] = context.step

// The state before a transition's step. Core form: `b` in `(b, a) => a.state == b`.
def before[S](using context: Before[S]): S = context.state

// One Boolean requirement; failure skips the remaining requirements. Core form:
// `condition && remainingRequirements`.
def check[S, O, F](condition: Boolean)(using context: Expectation[S, O, F]): Unit =
  context.require(condition)

// Fact requirements specialized to the current step before typing a member selector. Core form:
// `s.facts.contains(fact)` or `s.facts.contains("member_fact")`.
def records[S, O, F](using context: Expectation[S, O, F]): RequirementFacts[S, O, F] =
  new RequirementFacts(context)

// The callable fact requirement in a block. Core form: `s.facts.contains(fact)`.
final class RequirementFacts[S, O, F] private[framework] (
    private[framework] val context: Expectation[S, O, F]
):
  // A fact of this step's own machine. Core form: `s.facts.contains(fact)`.
  def apply(fact: F): Unit = context.require(context.step.facts.contains(fact))

  // A fact of the exact selected field and machine. Core form: `s.facts.contains("member_fact")`.
  inline def apply[MS, MO, MF](
      inline member: S => (MS, Machine[MS, MO, MF]),
      fact: MF
  )(using evidence: F =:= String): Unit =
    context.require(memberRecorded(context.step, MemberPath.captureBinding(member), fact))

// Explicit alternatives of one requirement. Core form: `first || second || rest`.
final class Alternatives[S, O, F] private[framework] (private[framework] val step: Step[S, O, F]):
  private val success = new scala.util.control.ControlThrowable {}
  private[framework] def attempt(body: Expectation[S, O, F] ?=> Unit): Unit =
    if new Expectation(step).evaluate(body) then throw success
  private[framework] def evaluate(body: Alternatives[S, O, F] ?=> Unit): Boolean =
    try
      body(using this)
      false
    catch case signal: scala.util.control.ControlThrowable if signal eq success => true

// At least one explicit branch must hold; later branches are skipped after success. Core form:
// `firstBranch || secondBranch || remainingBranches`.
def oneOf[S, O, F](using
    context: Expectation[S, O, F]
)(body: Alternatives[S, O, F] ?=> Unit): Unit =
  context.require(new Alternatives(context.step).evaluate(body))

// Explicit finite input catalogs. Core form: `Finite.of(values*)` with checked membership.
object domains:
  // Explicit choices in authored order. Core form: `Finite.of(values*)`.
  def oneOf[A](values: A*)(using Data.Structural[A]): Finite[A] = DomainChoices.create(values)
  // Integer choices in Range iteration order. Core form: `Finite.of(values*)`.
  def oneOf(values: Range): Finite[Int] = DomainChoices.range(values)

extension [In, A](field: InputField[In, A])
  // The catalog selected for this typed field. Core form: `InputBinding.make(field, domain)`.
  def in(domain: Finite[A]): InputBinding[In] = InputBinding.make(field, domain)

// Conjunctive requirements of one alternative, only inside oneOf. Core form:
// `(first && second)` as an operand of `firstBranch || secondBranch`.
def branch[S, O, F](using
    alternatives: Alternatives[S, O, F]
)(body: Expectation[S, O, F] ?=> Unit): Unit = alternatives.attempt(body)

// An ordinary step predicate with statement requirements. Core form:
// `(s: Step[S, O, F]) => first(s) && second(s)`.
object expect:
  // Bare fields read the resulting state. Core form: `(s: Step[S, O, F]) => predicate(s)`.
  inline def apply[S, O, F](body: Expectation[S, O, F] ?=> Unit): Step[S, O, F] => Boolean =
    ${ ExpectationMacros.same('body) }
  // A transition predicate, with before-state evidence and resulting-state bare fields.
  // Core form: `(b: S, a: Step[S, O, F]) => predicate(b, a)`.
  inline def across[S, O, F](
      body: (Before[S], Expectation[S, O, F]) ?=> Unit
  ): (S, Step[S, O, F]) => Boolean =
    ${ ExpectationMacros.transition('body) }

@scala.annotation.publicInBinary
private[framework] object ExpectationMacros:
  import scala.quoted.*

  @scala.annotation.publicInBinary
  private[framework] def same[S: Type, O: Type, F: Type](body: Expr[Expectation[S, O, F] ?=> Unit])(
      using Quotes
  ): Expr[Step[S, O, F] => Boolean] =
    val prepared = ExpectationBlock.validate(body)
    '{ sameExpectation($prepared) }

  @scala.annotation.publicInBinary
  private[framework] def transition[S: Type, O: Type, F: Type](
      body: Expr[(Before[S], Expectation[S, O, F]) ?=> Unit]
  )(using Quotes): Expr[(S, Step[S, O, F]) => Boolean] =
    val prepared = ExpectationBlock.validate(body)
    '{ transitionExpectation($prepared) }

  @scala.annotation.publicInBinary
  private[framework] def property[S: Type, O: Type, F: Type](
      builder: Expr[PropertyBuilder[S, O, F]],
      body: Expr[Expectation[S, O, F] ?=> Unit]
  )(using Quotes): Expr[SameStepProperty[S, O, F]] =
    val predicate = same(body)
    '{ $builder.sameStep($predicate) }

  @scala.annotation.publicInBinary
  private[framework] def transitionProperty[S: Type, O: Type, F: Type](
      builder: Expr[PropertyBuilder[S, O, F]],
      body: Expr[(Before[S], Expectation[S, O, F]) ?=> Unit]
  )(using Quotes): Expr[TransitionProperty[S, O, F]] =
    val predicate = transition(body)
    '{ $builder.transition($predicate) }

@scala.annotation.publicInBinary
private[framework] def sameExpectation[S, O, F](
    prepared: BlockPredicate[Expectation[S, O, F] ?=> Unit]
): Step[S, O, F] => Boolean =
  new (Step[S, O, F] => Boolean) with PredicateBindings:
    def apply(step: Step[S, O, F]): Boolean = new Expectation(step).evaluate(prepared.body)
    def validate(owner: Model): Unit =
      prepared.members.foreach(_.validate(owner))
      prepared.predicates.foreach(PredicateBindings.validate(_, owner))
    def sourceParameters: Vector[String] = prepared.parameters

@scala.annotation.publicInBinary
private[framework] def transitionExpectation[S, O, F](
    prepared: BlockPredicate[(Before[S], Expectation[S, O, F]) ?=> Unit]
): (S, Step[S, O, F]) => Boolean =
  new ((S, Step[S, O, F]) => Boolean) with PredicateBindings:
    def apply(state: S, step: Step[S, O, F]): Boolean =
      new Expectation(step).evaluate(prepared.body(using new Before(state)))
    def validate(owner: Model): Unit =
      prepared.members.foreach(_.validate(owner))
      prepared.predicates.foreach(PredicateBindings.validate(_, owner))
    def sourceParameters: Vector[String] = prepared.parameters

// A Property builder with before-state evidence enabled. Core form:
// `builder.transition((b, a) => predicate(b, a))` (private core construction).
final class Across[S, O, F] private[framework] (
    private[framework] val builder: PropertyBuilder[S, O, F]
):
  // Finishes a transition Property. Core form: `builder.transition((b, a) => predicate(b, a))`.
  inline infix def expect(
      body: (Before[S], Expectation[S, O, F]) ?=> Unit
  ): TransitionProperty[S, O, F] = ${ ExpectationMacros.transitionProperty('builder, 'body) }

// The same-step expectation view of an ordinary Property builder. Core form:
// `builder.sameStep(s => predicate(s))` (private core construction).
final class Expecting[S, O, F] private[framework] (
    private[framework] val builder: PropertyBuilder[S, O, F]
):
  // Finishes a same-step Property. Core form: `builder.sameStep(s => predicate(s))`.
  inline infix def expect(body: Expectation[S, O, F] ?=> Unit): SameStepProperty[S, O, F] =
    ${ ExpectationMacros.property('builder, 'body) }

// Makes expectation finishing available on a core builder. Core form: `builder.sameStep(predicate)`.
implicit def expectationBuilder[S, O, F](builder: PropertyBuilder[S, O, F]): Expecting[S, O, F] =
  new Expecting(builder)

// Property heads preserve the builder's exact class or whole-action restriction. Core form:
// `builder.when(c).sameStep(predicate)` or `builder.whenAction(action).sameStep(predicate)`.
extension [S, O, F](builder: PropertyBuilder[S, O, F])
  // Restricts to one exact action class. Core form: `builder.when(c)`.
  infix def after(c: ClassRef | Composed): PropertyBuilder[S, O, F] = builder.when(c)
  // Restricts to all classes of the action. Core form: `builder.whenAction(action)`.
  def afterAction(action: String): PropertyBuilder[S, O, F] = builder.whenAction(action)
  // Restricts to a whole composed action. Core form: `builder.whenAction(action)`.
  def afterAction(action: Composed): PropertyBuilder[S, O, F] = builder.whenAction(action)
  // Enables the before-state reading. Core form: `builder.transition(predicate)`.
  def across: Across[S, O, F] = new Across(builder)

// The status fact a phase enum's case is recorded as, declared on each case through the enum's
// parameter, which has no default, so a case without one does not compile:
// `enum Phase(val status: Fact) extends Recorded[Fact]`, `case started extends Phase(statusStarted)`.
// An `effect:` block that assigns such a value records its status after the facts the block
// records. Core form: the status fact each step writes by hand,
// `enter(s.copy(phase = started), statusStarted)`.
trait Recorded[F]:
  def status: F

// What an `effect:` block makes of one step while it runs: the state it enters, which each field
// assignment replaces with a copy, by the setters its state type declares, each of one shape:
// `def phase_=(p: Phase)(using d: Draft[State, ?, ?]): Unit = d.set(_.copy(phase = p))`, or for a
// field whose values are `Recorded`, `def phase_=(p: Phase)(using d: Draft[State, ?, Fact]): Unit =
// d.set(p)(_.copy(phase = p))`; the facts it records, then the status of the value such a setter
// assigned; and the outcome it rejects with, if any. Each call of the effect makes a fresh draft,
// which never leaves the block. Core form: the arguments of the step the step function returns,
// `Step(outcome, s.copy(phase = p), List(facts*))`.
@implicitNotFound(
  "this assigns a field of ${S}, records a fact or rejects, which only an `effect:` block does: " +
    "outside one, return the steps, `enter(s.copy(...), fact)` or `reject(outcome, s)`"
)
final class Draft[S, O, F] private[framework] (start: S) extends View[S]:
  private var current: S = start // scalafix:ok DisableSyntax.var
  private var facts: List[F] = Nil // scalafix:ok DisableSyntax.var
  private var rejection: Option[O] = None // scalafix:ok DisableSyntax.var
  private var statuses: List[F] = Nil // scalafix:ok DisableSyntax.var

  private[framework] def state: S = current

  // The state with one field replaced: `d.set(_.copy(phase = p))`. Core form: `s.copy(phase = p)`.
  def set(update: S => S): Unit = current = update(current)

  // The state with the field whose values declare their status replaced by `value`, which the step
  // records the status of: `d.set(p)(_.copy(phase = p))`. Core form: `s.copy(phase = p)`, and
  // `p.status` among the step's facts.
  def set(value: Recorded[F])(update: S => S): Unit =
    current = update(current)
    statuses = statuses :+ value.status

  // The step the block made, from the state `s` it started in: the rejection with `s`, or the ok
  // outcome with the state assigned and the facts in the order recorded, then the status of each
  // value assigned, which the block does not record itself.
  private[framework] def step(s: S, ok: Ok[O]): Step[S, O, F] = rejection match
    case Some(outcome) => Step(outcome, s)
    case None          =>
      for f <- statuses do
        require(
          !facts.contains(f),
          s"an effect block records $f, which its assignment of a status already records: " +
            s"drop the record($f)"
        )
      Step(ok.outcome, current, facts ++ statuses)

  private[framework] def add(recorded: Seq[F]): Unit = facts = facts ++ recorded

  private[framework] def refuse(outcome: O): Unit = rejection = Some(outcome)

// `val held = is: phase == Phase.held`: a predicate of the machine's state, its fields read by
// name. Core form: `def held(s: State) = s.phase == Phase.held`.
def is[S, O, F](using
    @unused owner: Owner[S, O, F]
)(body: View[S] ?=> Boolean): S => Boolean =
  s => body(using Fixed(s))

// `val pause = effect:` with `phase = Phase.paused` and `record(statusPaused)`: a step function of the
// machine's state alone, its fields assigned by name, and its facts recorded, or its outcome
// rejected, by statements. A block that neither records nor rejects enters the assigned state with
// no fact. Core form: `def pause(s: State) = enter(s.copy(phase = Phase.paused), statusPaused)`, or
// `reject(outcome, s)` for a block that rejects.
def effect[S, O, F](using
    @unused owner: Owner[S, O, F],
    ok: Ok[O]
)(
    body: Draft[S, O, F] ?=> Unit
): S => List[Step[S, O, F]] =
  s =>
    val draft = Draft[S, O, F](s)
    body(using draft)
    List(draft.step(s, ok))

// `when(Phase.held) ~> enters(Phase.completed)`, or `retryPaused -> enters(Phase.paused)(s)` as an
// alternative of a `choose`: the step function of the state alone that takes one step with the ok
// outcome into the state with its phase replaced by `phase`, recording the status the phase's case
// declares where its enum is `Recorded`, and nothing otherwise. It writes the phase through the
// writer the machine's projection declares, `Phased[State, Phase](_.phase, (s, p) => s.copy(phase = p))`, and a
// derived machine through its source's. It is what an effect whose whole body sets the phase,
// `effect: phase = Phase.completed`, does, so the rule shows the phase it enters and `effects`
// keeps the steps with logic. Core form: `enter(s.copy(phase = Phase.completed), statusCompleted)`,
// or `enter(s.copy(phase = p))` for a phase that declares no status.
// The phase's own type `C` may be narrower than the machine's phase type `P`, as a case that mixes
// in its role is (`Phase & Held`): the machine's `Phasing` fixes `P`.
def enters[S, O, F, P, C <: P](phase: C)(using
    @unused owner: Owner[S, O, F],
    ok: Ok[O],
    phasing: Phasing[S, P]
): S => List[Step[S, O, F]] =
  val status = phase match
    case recorded: Recorded[F @unchecked] => List(recorded.status)
    case _                                => Nil
  s => List(Step(ok.outcome, phasing.write(s, phase), status))

// `record(statusPaused)`, in an `effect:` block: the step records these facts, after those an
// earlier `record` of the block recorded. Core form: the facts of `enter(s.copy(...), facts*)`.
def record[S, O, F](using draft: Draft[S, O, F])(first: F, rest: F*): Unit =
  draft.add(first +: rest)

// `reject(Outcome.notFound)`, in an `effect:` block: the step answers `outcome` and keeps the
// state, recording nothing. Core form: `reject(Outcome.notFound, s)`.
def reject[S, O, F](using draft: Draft[S, O, F])(outcome: O): Unit =
  draft.refuse(outcome)

// `phase.in(a, b, c)`: whether the value is one of the members listed, of which there is at least
// one. Written dotted, never infix. Core form: `List(a, b, c).contains(phase)`.
extension [A](value: A) @sugar def in(first: A, rest: A*): Boolean = (first +: rest).contains(value)

// `phase.in[Closed]`: whether the phase has the role `Closed` (model/framework/dsl/Roles.scala), the roles
// its case declares and the broader ones they extend. Written dotted, never infix. Core form:
// `List(<the cases of Closed, in declaration order>).contains(phase)`.
extension [A](value: A)
  @sugar def in[R](using role: TypeTest[A, R]): Boolean = role.unapply(value).nonEmpty

// `a implies b`: `b` holds wherever `a` does. `b` is read only where `a` holds, so a hole it reaches
// is not reached where `a` is false. Core form: `!a || b`.
extension (a: Boolean) @sugar infix def implies(b: => Boolean): Boolean = !a || b

// `after.records(fact)`: whether the step records the fact, which is of the step's own fact type:
// a step's facts are covariant, so the evidence keeps a fact of another type from widening it.
// Core form: `after.facts.contains(fact)`.
final class StepFacts[S, O, F] private[framework] (private[framework] val step: Step[S, O, F]):
  // A fact of the step's own type. Core form: `step.facts.contains(fact)`.
  @sugar def records[G](fact: G)(using G <:< F): Boolean = step.facts.contains(fact)

  // A composition's selected member fact. Core form: `step.facts.contains("member_fact")`.
  @sugar inline def records[M](inline member: S => M, fact: Any)(using
      evidence: F =:= String,
      @unused notTypedMember: NotGiven[M <:< (Any, Model)]
  ): Boolean =
    step.facts.contains(
      evidence.flip(s"${MemberPath.capture(member, fact).field}_${valueKey(fact)}")
    )

  // A typed member fact, retaining its exact declaring field and machine. Core form:
  // `step.facts.contains("member_fact")`.
  @targetName("recordsTypedMember")
  inline def records[MS, MO, MF](
      inline member: S => (MS, Machine[MS, MO, MF]),
      fact: MF
  )(using evidence: F =:= String): Boolean =
    memberRecorded(step, MemberPath.captureBinding(member), fact)

// Makes Boolean fact reads available on an ordinary step. Core form: `step.facts.contains(fact)`.
implicit def stepFacts[S, O, F](step: Step[S, O, F]): StepFacts[S, O, F] = new StepFacts(step)

// `after.records(_.member, fact)`: whether a composition's step records the fact of the member the
// selector names, the composition form of `after.records(fact)`. Core form:
// `after.facts.contains("member_fact")`, the composed key `<field>_<fact>` a composition records.
@scala.annotation.publicInBinary
private[framework] def memberRecorded[S, O, F, MF](
    step: Step[S, O, F],
    member: MemberBinding,
    fact: MF
)(using evidence: F =:= String): Boolean =
  step.facts.contains(evidence.flip(s"${member.field}_${valueKey(fact)}"))

// A captured member's exact composed fact key. Core form: `step.facts.contains(key)`.
@scala.annotation.publicInBinary
@sugar private[framework] def recorded[S, O](
    step: Step[S, O, String],
    member: MemberPath,
    fact: Any
): Boolean =
  step.facts.contains(s"${member.field}_${valueKey(fact)}")

// `m.witness(a, b).records(fact)`: one path-and-outcome declaration, implicitly on the enclosing
// machine or composition. Its default search budget is 512, with depth and schedule length equal
// to the path length. Core form:
// `query find m.property.when(b).sameStep(_.facts.contains(fact)) in m.scenario.actions(a, b) limits bounds`.
extension [S](owner: Declares[S])
  def witness(
      first: ClassRef | Composed,
      rest: (ClassRef | Composed)*
  ): WitnessBuilder[S, owner.Outcome, owner.Fact] =
    WitnessBuilder(owner, "", first +: rest.toVector)

  // A witness named explicitly, such as one in a list. Core form:
  // `query("done") find m.property("done").when(b).sameStep(_.facts.contains(fact)) in m.scenario("done").actions(a, b) limits bounds`.
  def witness(name: String)(
      first: ClassRef | Composed,
      rest: (ClassRef | Composed)*
  ): WitnessBuilder[S, owner.Outcome, owner.Fact] =
    WitnessBuilder(owner, name, first +: rest.toVector)

// A witness on the enclosing machine or composition, named after its val. Core form:
// `query find property.when(last).sameStep(_.facts.contains(fact)) in scenario.actions(path*) limits bounds`.
def witness[S, O, F](
    first: ClassRef | Composed,
    rest: (ClassRef | Composed)*
)(using declaring: Declaring[S, O, F]): WitnessBuilder[S, O, F] =
  WitnessBuilder(declaring.model, "", first +: rest.toVector)

// A witness on the enclosing owner with an explicit name. Core form:
// `query(name) find property(name).when(last).sameStep(_.facts.contains(fact)) in scenario(name).actions(path*) limits bounds`.
def witness[S, O, F](name: String)(
    first: ClassRef | Composed,
    rest: (ClassRef | Composed)*
)(using declaring: Declaring[S, O, F]): WitnessBuilder[S, O, F] =
  WitnessBuilder(declaring.model, name, first +: rest.toVector)

// Finishes a witness with a fact of its declaring machine. Core form:
// `m.property.when(last).sameStep(step => step.facts.contains(fact))`, joined to the pinned Scenario
// by a find Query.
final class WitnessFacts[S, O, F] private[framework] (
    private[framework] val builder: WitnessBuilder[S, O, F]
):
  // A primary fact of the witness's machine. Core form: `builder.recording(_.facts.contains(fact))`.
  def records(fact: F): Witness[S, O, F] =
    builder.recording(_.facts.contains(fact))

  // A primary fact of an exact selected member. Core form:
  // `builder.recording(_.facts.contains("member_fact"))`.
  inline def records[MS, MO, MF](
      inline member: S => (MS, Machine[MS, MO, MF]),
      fact: MF
  )(using evidence: F =:= String): Witness[S, O, F] =
    val selected = MemberPath.captureBinding(member)
    selected.validate(builder.owner)
    builder.recording(step => memberRecorded(step, selected, fact))

// Makes primary fact finishing available on a witness builder. Core form:
// `builder.recording(_.facts.contains(fact))`.
implicit def witnessFacts[S, O, F](builder: WitnessBuilder[S, O, F]): WitnessFacts[S, O, F] =
  new WitnessFacts(builder)

// Finishes with an absent primary fact. Core form: `builder.recording(step => !step.facts.contains(fact))`.
extension [S, O, F](builder: WitnessBuilder[S, O, F])
  // Finishes with an absent primary fact. Core form:
  // `m.property.when(last).sameStep(step => !step.facts.contains(fact))`.
  def excludes[G](fact: G)(using G <:< F): Witness[S, O, F] =
    builder.recording(step => !step.facts.contains(fact))

// Appends typed after-step guards before the optional state ending. Core form:
// `prior(step) && !step.facts.contains(fact)` or `prior(step) && (!condition(step.state) || step.facts.contains(fact))`.
extension [S, O, F](claim: Witness[S, O, F])
  // Requires an additional absent fact in written guard order. Core form:
  // `prior(step) && !step.facts.contains(fact)`.
  def excludes[G](fact: G)(using G <:< F): Witness[S, O, F] =
    claim.requiring(step => !step.facts.contains(fact))
  // Requires an additional fact only when its after-state condition holds. Core form:
  // `prior(step) && (!condition(step.state) || step.facts.contains(fact))`.
  def recordsWhen[G](condition: S => Boolean, fact: G)(using
      G <:< F
  ): Witness[S, O, F] =
    claim.recordingWhen(condition, _.facts.contains(fact))

// Finishes a composition's witness with the fact of its selected member, whose machine fixes the
// fact type, as the composition's declaration selector does. Core form:
// `c.property.when(last).sameStep(step => step.facts.contains("member_fact"))`.
extension [S <: Product](builder: WitnessBuilder[S, String, String])
  // Finishes with an absent fact of the selected member. Core form:
  // `c.property.when(last).sameStep(step => !step.facts.contains("member_fact"))`.
  inline def excludes[MS, O, F](
      inline member: S => (MS, Machine[MS, O, F]),
      fact: F
  ): Witness[S, String, String] =
    val key = witnessMemberFact(builder.owner, MemberPath.retainDirect(member), fact)
    builder.recording(step => !step.facts.contains(key))

// Appends composed guards using the same exact typed member selector as the primary. Core form:
// `prior(step) && !step.facts.contains("member_fact")` or `prior(step) && (!condition(step.state) || step.facts.contains("member_fact"))`.
extension [S <: Product](claim: Witness[S, String, String])
  // Requires an absent fact from the selected member. Core form:
  // `prior(step) && !step.facts.contains("member_fact")`.
  inline def excludes[MS, O, F](
      inline member: S => (MS, Machine[MS, O, F]),
      fact: F
  ): Witness[S, String, String] =
    val key = witnessMemberFact(claim.owner, MemberPath.retainDirect(member), fact)
    claim.requiring(step => !step.facts.contains(key))
  // Requires the selected member's fact under the composed after-state condition. Core form:
  // `prior(step) && (!condition(step.state) || step.facts.contains("member_fact"))`.
  inline def recordsWhen[MS, O, F](
      condition: S => Boolean,
      inline member: S => (MS, Machine[MS, O, F]),
      fact: F
  ): Witness[S, String, String] =
    val key = witnessMemberFact(claim.owner, MemberPath.retainDirect(member), fact)
    claim.recordingWhen(condition, _.facts.contains(key))

@scala.annotation.publicInBinary
private[framework] def witnessMemberFact[S <: Product, MS, O, F](
    owner: Declares[S],
    member: S => (MS, Machine[MS, O, F]),
    fact: F
): String =
  val composition = owner match
    case c: Composition[S @unchecked] => c
    case _                            =>
      throw IllegalArgumentException(
        "a member fact belongs to a composition witness"
      )
  val machine = member(composition.mirror.fromProduct(Composition.Empty))._2
  val field = MemberPath
    .retainedField(member)
    .getOrElse(
      throw IllegalArgumentException("a member fact requires a retained direct field selector")
    )
  new MemberBinding(field, machine).validate(composition)
  s"${field}_${valueKey(fact)}"

// `m.property("terminalIsFinal").once(terminal).keeps(_.phase)`: once `over` holds of the state
// before a step, the step keeps the value the projection selects. Core form:
// `b.transition((before, after) => !over(before) || after.state.phase == before.phase)`.
extension [S, O, F](b: PropertyBuilder[S, O, F])
  @sugar
  def once(over: S => Boolean): Once[S, O, F] =
    require(
      b.when.isEmpty,
      s"once declares ${b.name} about every step, so it takes no when: write the restriction into its predicates, or declare the Property with `expect`"
    )
    Once(b, over)

  // `never(to)`: no step is one `to` holds of, the same-step invariant. Core form:
  // `b.sameStep(after => !to(after))`. `never(to).from(before)` says it of the steps from a state `before`
  // holds of only.
  @sugar
  def never(to: Step[S, O, F] => Boolean): Never[S, O, F] =
    require(
      b.when.isEmpty,
      s"never declares ${b.name} about every step, so it takes no when: write the restriction into its predicates, or declare the Property with `expect`"
    )
    PredicateBindings.validate(to, b.m)
    Never(b, to)

  // `stays(p)`: a step from a state `p` holds of keeps it holding. Core form:
  // `b.transition((before, after) => !p(before) || p(after.state))`. `stays(p).unless(release)` also
  // lets a step `release` holds of leave it.
  @sugar
  def stays(p: S => Boolean): Stays[S, O, F] =
    require(
      b.when.isEmpty,
      s"stays declares ${b.name} about every step, so it takes no when: write the restriction into its predicates, or declare the Property with `expect`"
    )
    Stays(b, p)

// What `once(over)` leaves to say: the value it keeps. Core form: `b.transition(predicate)`.
@sugar final class Once[S, O, F] private[framework] (
    b: PropertyBuilder[S, O, F],
    over: S => Boolean
):
  // The value a step from a state `over` holds of keeps, a field path such as `_.phase` or
  // `_.order.phase`. Core form:
  // `b.transition((before, after) => !over(before) || after.state.phase == before.phase)`.
  @sugar def keeps[V](projection: S => V)(using CanEqual[V, V]): Property[S] =
    b.transition((before, after) => !over(before) || projection(after.state) == projection(before))

// `never(to)`, a same-step Property, which `from` turns into a transition one. Core form:
// `b.sameStep(after => !to(after))`.
@sugar final class Never[S, O, F] private[framework] (
    b: PropertyBuilder[S, O, F],
    to: Step[S, O, F] => Boolean
) extends Property[S](b.sameStep(after => !to(after)).decl):
  // No step from a state `before` holds of is one `to` holds of. Core form:
  // `b.transition((before, after) => !origin(before) || !to(after))`.
  @sugar def from(before: S => Boolean): Property[S] =
    val origin = before
    b.transition((before, after) => !origin(before) || !to(after))

// `stays(p)`, a transition Property, which `unless` releases. Core form:
// `b.transition((before, after) => !p(before) || p(after.state))`.
@sugar final class Stays[S, O, F] private[framework] (
    b: PropertyBuilder[S, O, F],
    p: S => Boolean
) extends Property[S](b.transition((before, after) => !p(before) || p(after.state)).decl):
  // A step `release` holds of may leave `p`. Core form:
  // `b.transition((before, after) => !p(before) || p(after.state) || release(after))`.
  @sugar def unless(release: Step[S, O, F] => Boolean): Property[S] =
    PredicateBindings.validate(release, b.m)
    b.transition((before, after) => !p(before) || p(after.state) || release(after))

// Concrete input-field evidence before a selector is typed. Core form: `new InputFields[In]`.
def inputField[In <: Product]: InputFields[In] = new InputFields[In]

// A concrete input record's checked field selectors. Core form: `InputField(label, getter)`.
final class InputFields[In <: Product] private[framework] ():
  // Retains one direct field. Core form: the getter compiled for the concrete record owner.
  inline def apply[A](inline selector: In => A): InputField[In, A] =
    ${ InputFieldMacros.capture[In, A]('selector) }

// Fields attach only to declared records, including through precise generic forwarding.
extension [In <: Product](@unused action: RecordAction[In])
  // Anchors the declared record before typing `_.field`. Core form: `inputField[In]`.
  def field: InputFields[In] = inputField[In]

// A scalar input with its exact declaration label. Core form: `Action[In]` with a scalar catalog.
extension (a: Action[EmptyTuple])
  // A required scalar input. Core form: `InputSource.scalar(a, name, None, domain)`.
  def input[A](name: String)(using domain: Finite[A]): ScalarAction[A] =
    InputSource.scalar(a, name, None, domain)

  // A scalar input with a declared default. Core form: `InputSource.scalar(a, name, Some(default), domain)`.
  def input[A](name: String, default: A)(using domain: Finite[A]): ScalarAction[A] =
    InputSource.scalar(a, name, Some(default), domain)

  // An explicitly selected scalar catalog. Core form: `InputSource.scalarExplicit(a, name, None, domain)`.
  @targetName("inputWithDomain")
  def input[A](name: String, domain: Finite[A]): ScalarAction[A] =
    InputSource.scalarExplicit(a, name, None, domain)

  // A default within an explicitly selected catalog. Core form: `InputSource.scalarExplicit(a, name, Some(default), domain)`.
  @targetName("inputWithDefaultAndDomain")
  def input[A](name: String, default: A, domain: Finite[A]): ScalarAction[A] =
    InputSource.scalarExplicit(a, name, Some(default), domain)

// `val retainedOutcome = sticky(outcomePreserved)`: a monitor of a promise about every step that,
// once a step breaks it, stays broken, named after its `val`. Its verdict is read after every step.
// Core form:
// `monitor[S, O, F, Boolean](false)((broken, before, after) => broken || !promise(after))(broken => broken)`.
@scala.annotation.nowarn("msg=unused explicit parameter")
@sugar def sticky[S, O, F](
    promise: Step[S, O, F] => Boolean
): Monitor[S, O, F, Boolean] =
  monitor[S, O, F, Boolean](false)((broken, before, after) => broken || !promise(after))(broken =>
    broken
  )

// `val ownerAcknowledgment = stickyAcross(ackOnlyWhenKept)`: `sticky` of a promise about the state
// before a step and the step. Core form:
// `monitor[S, O, F, Boolean](false)((broken, before, after) => broken || !promise(before, after))(broken => broken)`.
@sugar def stickyAcross[S, O, F](
    promise: (S, Step[S, O, F]) => Boolean
): Monitor[S, O, F, Boolean] =
  monitor[S, O, F, Boolean](false)((broken, before, after) => broken || !promise(before, after))(
    broken => broken
  )

// A case of an `on` block: where in the state space its action fires, `when(Phase.placed)`,
// `when(Phase.open)`, `where(g)`, `always` or `when(Phase.open).where(g)`, and through `~>` the
// effect it has there, `when(Phase.placed) ~> effects.ship`. Core form: the arm `if g(s) then e(s, inputs) else ...` of the
// action's step function.
final class Case[S, O, F] private[framework] (
    private[framework] val heading: String,
    private[framework] val guard: S => Boolean
):
  // This case where `condition` also holds of the state, beyond its phases:
  // `when(Phase.open).where(_.timer == Timeout.expires)`. Core form: `g(s) && condition(s)`.
  def where(condition: S => Boolean): Case[S, O, F] =
    guard match
      case input: InputGuard[S] @unchecked =>
        val combined = new InputGuard[S]:
          def apply(s: S, cls: Class): Boolean = input(s, cls) && condition(s)
        Case(s"$heading.where", combined)
      case _ => Case(s"$heading.where", s => guard(s) && condition(s))

  // A condition over this action's retained input. Core form: `guard(s, in) && condition(s, in)`.
  def where[I](using
      firing: Firing[S, O, F, NativeInput[I]]
  )(
      condition: (S, I) => Boolean
  ): Case[S, O, F] =
    val input = firing.inputGuard(condition)
    val combined = new InputGuard[S]:
      def apply(s: S, cls: Class): Boolean = guardHolds(guard, s, cls) && input(s, cls)
    Case(s"$heading.where", combined)

  // The effect of the block's action where this case holds, read of the state alone:
  // `when(Phase.placed) ~> effects.ship`. Core form: the arm `if g(s) then ship(s)` of the action's step
  // function.
  infix def ~>(effect: S => List[Step[S, O, F]])(using
      firing: Firing[S, O, F, ?]
  ): Unit =
    firing.bind(this, stateEffect(effect))

  // The effect of the block's action, of one input, where this case holds, read of the state and
  // the input: `when(Phase.open) ~> effects.change`. Core form: the arm `if g(s) then change(s, c)`.
  @targetName("readsOne")
  infix def ~>[I, A](effect: (S, A) => List[Step[S, O, F]])(using
      firing: Firing[S, O, F, I],
      @unused input: RuleInput[I, A]
  ): Unit = firing.bind(this, effectOf(firing.decl, effect))

  private[framework] def capturedRule0(
      effect: S => List[Step[S, O, F]],
      data: Seq[String]
  )(using firing: Firing[S, O, F, ?]): Unit =
    firing.retaining(this, data)(this.~>(effect)(using firing))

  // The captured form of the input-reading `~>`: the plugin passes its retained data after the
  // effect, and the call keeps the type arguments and evidence the original call applies.
  private[framework] def capturedRule1[I, A](
      effect: (S, A) => List[Step[S, O, F]],
      data: Seq[String]
  )(using firing: Firing[S, O, F, I], input: RuleInput[I, A]): Unit =
    firing.retaining(this, data)(this.~>(effect)(using firing, input))

// The action, or the one class of it, an `on` block fires, and where its cases go, given to the
// block's cases. Core form: none of its own; it is the `action` of `action ~> stepFunction`, whose
// arms the block's cases are.
final class Firing[S, O, F, I] private[framework] (
    private[framework] val decl: ActionDecl,
    private[framework] val values: Option[List[Any]],
    private[framework] val action: String,
    add: (
        Case[S, O, F],
        (S, List[Any]) => List[Step[S, O, F]],
        Option[RuleBindings.Registration]
    ) => Unit,
    source: Option[Machine[S, O, F]] = None,
    origin: Option[FacetBindings.Origin] = None,
    budgets: Vector[NativeBudget[?]] = Vector.empty
):
  private var pending =
    Option.empty[(Case[S, O, F], RuleBindings.Packet, Option[FacetBindings.Template])]
  private var capturing = false

  private[framework] def inputGuard[A](condition: (S, A) => Boolean)(using
      I =:= NativeInput[A]
  ): InputGuard[S] =
    val shape = decl.nativeInputs.getOrElse(
      throw IllegalArgumentException("an input-reading guard requires a declared native input")
    )
    new InputGuard[S]:
      def apply(s: S, cls: Class): Boolean =
        condition(s, shape.input(cls).asInstanceOf[A]) // scalafix:ok DisableSyntax.asInstanceOf

  private[framework] def retaining(c: Case[S, O, F], data: Seq[String])(body: => Unit): Unit =
    require(!capturing, s"Rule $action: capture is already pending on this Firing")
    val template =
      if data.headOption.contains("facet-rule-v1") then Some(FacetBindings.template(data)) else None
    val payload = template.fold(RuleBindings.packet(data, s"Rule $action"))(_.packet)
    capturing = true
    pending = Some((c, payload, template))
    try
      body
      require(pending.isEmpty, s"Rule $action: Firing did not consume its capture")
    finally
      pending = None
      capturing = false

  private[framework] def bind(
      c: Case[S, O, F],
      effect: (S, List[Any]) => List[Step[S, O, F]]
  ): Unit =
    require(!capturing || pending.nonEmpty, s"Rule $action: Firing capture was consumed already")
    val captured = pending.map { (expected, packet, template) =>
      pending = None
      require(expected.eq(c), s"Rule $action: capture belongs to another Case")
      val machine =
        source.getOrElse(throw IllegalArgumentException(s"Rule $action: no direct Machine owner"))
      RuleBindings.registration(
        machine,
        decl,
        values,
        c.guard,
        effect,
        packet,
        template.map(t =>
          FacetBindings.Retained(
            origin.getOrElse(
              throw IllegalArgumentException(s"Rule $action: no facet RuleBook source")
            ),
            t
          )
        )
      )
    }
    val native = if budgets.isEmpty then effect
    else
      val adapted = effect match
        case value: InputEffect[S, O, F] @unchecked => value
        case value                                  =>
          new InputEffect[S, O, F]:
            def apply(state: S, values: List[Any]): List[Step[S, O, F]] = value(state, values)
      budgetEffect(adapted, budgets)
    add(c, native, captured)

// A case that holds where `condition` holds of the state: `where(states.held) ~> effects.settle`.
// Core form: `if condition(s) then e(s) else ...`.
def where[S, O, F, I](using firing: Firing[S, O, F, I]): GuardCases[S, O, F, I] =
  new GuardCases(firing)

// Conditions specialized to an action before typing their callbacks. Core form: `if g(s, in) then e(s, in)`.
final class GuardCases[S, O, F, I] private[framework] (
    private[framework] val firing: Firing[S, O, F, I]
):
  // A state condition. Core form: `if condition(s) then e(s, in)`.
  def apply(condition: S => Boolean): Case[S, O, F] = Case("where", condition)

extension [S, O, F, I](cases: GuardCases[S, O, F, I])
  // A state and native input condition. Core form: `if condition(s, in) then e(s, in)`.
  def apply[A](using input: RuleInput[I, A])(condition: (S, A) => Boolean): Case[S, O, F] =
    Case("where", cases.firing.inputGuard(condition)(using input.native))

// A case that holds in every state: `always ~> effects.keep`. Core form:
// `if true then e(s) else ...`.
def always[S, O, F](using firing: Firing[S, O, F, ?]): Case[S, O, F] =
  val _ = firing
  Case("always", _ => true)

// The projection of a machine's or composition's state onto its phase, declared once by mixing it
// into the object: `object OrderProduct extends Machine[OrderState, Outcome, OrderFact],
// Phased[OrderState, Phase](_.phase)`, or for a composition a nested phase,
// `Phased[OverQueue, Phase](_.order.phase)`. Both types are written: a trait parent's lambda takes
// no parameter type from the other parents. A machine whose rules or effects enter a phase by
// `enters(Phase.x)` also declares the writer that replaces the phase the projection reads,
// `Phased[OrderState, Phase](_.phase, (s, p) => s.copy(phase = p))`; the IR generator requires it to copy the
// state along the projection's field path. Its sections read it as a given: `when(Phase.placed)` and
// `when(states.terminal)` in `object rules extends Rules:` test it. It is optional, as a machine that
// names no phase is a plain `Machine`. A derived machine or derived composition mixes in none of
// its own: a derived machine reads its source's, with its phase type. Its default end is the Closed
// role, witnessed at the concrete parent declaration; an explicit end overrides it. The default's
// first evaluation refuses a phase with no Closed case, naming the object and phase type. A derived
// machine's final end prevents mixing it in, and a derived composition refuses its own projection
// as it initializes. Core form: the given `Phasing(_.phase)` the object's sections read.
trait Phased[S, P](
    private[framework] val projection: S => P,
    private[framework] val writer: (S, P) => S = Phasing.Unwritten
)(using
    Finite[P],
    TypeTest[P, Closed],
    ClassTag[P]
) extends Declares[S]:
  // The projection its sections and readers of their generated capability claims share.
  given phased: Phasing[S, P] = Phasing(projection, writer)

  private lazy val closedCases = phased.roleCases[Closed](name)

  // The default stopping point is closedness. A machine whose stopping point differs overrides it.
  // Core form: `projection(s).in(<the Closed cases of P>)`.
  def end(s: S): Boolean = closedCases.contains(projection(s))

  final override private[framework] def declaresPhase: Boolean = true

// The phase a derived machine reads through its source. Its evidence is preferred to the evidence
// of a source that declares no phase, which it extends. Core form: none of its own; it is the
// source's `Phasing(_.phase)`.
object Phased extends Inherited.Unphased:
  // A source that mixes in `Phased[S, P]` projects its state onto `P` by its own projection.
  // Core form: the source's `Phasing(_.phase)`.
  given source[S, P, T <: Phased[S, P]]: Inherited[T, S, P] =
    Inherited(_.projection, _.end, _.writer)

private[framework] def isDirectRuleBook(book: RuleBook[?, ?, ?]): Boolean =
  book.isInstanceOf[Rules[?, ?, ?, ?]]

extension (@unused companion: MemberPath.type)
  private[framework] def field(member: Any): Option[String] = MemberPath.retainedField(member)

// When each action of a machine fires, written as the machine object's `object rules extends
// Rules`, whose cases may name phases where the machine mixes in `Phased[State, Phase](_.phase)`
// through its given: blocks of an action or action class,
// `on(clerk.ship): when(Phase.placed) ~> effects.send`, whose cases each say where the action fires,
// `when(...)` of phases or of a named set of them, `where(g)` of the state,
// `when(...).where(g)` of both or `always`, and what it does there, `~> effects.x`; and the actions
// no state enables, `disabled(courier.strike)`. A block fires a whole action, `on(buyer.change)`,
// one class of it, `on(buyer.change(Change.hold))`, whose effects then read the state alone, or
// several of them alike, `on(clerk.ship, clerk.cancel)`; an effect that takes arguments beyond the
// state binds them in place, `effects.timeOut(_, Expiry.close)`. Every target names its action
// through the object that declares it, `clerk.ship`, never by a bare name: the IR generator
// refuses a bare target, and a bare `disabled` target, at its line, and a block refuses as the rules
// construct any bare target, including imported names, using its original source spelling. The
// blocks of the actions one object declares may be grouped in
// `from(clerk):` with nested `on(clerk.ship):` blocks, which imports nothing. A case names phases through
// their type, `when(Phase.placed)`. Where no
// case holds, the action is disabled: there is no catch-all. The cases of one action class hold in
// no common reached state, whichever blocks they sit in. Registration retains the rules without
// running their guards or effects; overlap checking belongs to the interpreter. Core form: one step
// function per action, the cases of the action in order,
// `action ~> ((s, i) => if g1(s) && c1(i) then e1(s, i) else if g2(s) then e2(s, i) else Nil)`, and
// `action ~> (_ => Nil)` for an action no state enables.
//
// `on` is `inline`, with an `inline` action: the one sanctioned exception to "no inline on the
// author surface" (.plans/DSL_OPERATORS.md, rule 5). An action's runtime name is `""`, since the
// lifter names it after its `val`, so `scala.compiletime.codeOf` names the action in an overlap
// message. TASTy is pickled before inlining, so the lifter reads the unexpanded call, and it refuses
// an expanded one loudly ("not a rule") rather than lift it.
abstract class Rules[S, O, F, P](using
    owner: Owner[S, O, F],
    phasing: Phasing[S, ? <: P],
    source: FacetSource
) extends RuleBook[S, O, F]:
  private var origin = Option.empty[FacetBindings.Origin]
  final override private[framework] def facetOrigin: Option[FacetBindings.Origin] = origin
  @scala.annotation.publicInBinary
  final private[framework] def retainFacetOrigin(
      assembly: Machine[?, ?, ?],
      namespace: AnyRef,
      traitClass: java.lang.Class[?],
      traitName: String,
      namespaceName: String,
      rulesName: String
  ): Unit = synchronized {
    require(assembly.eq(owner.machine), "Machine Facets: foreign RuleBook assembly source")
    require(
      source.facet.nonEmpty && traitName.nonEmpty && namespaceName.nonEmpty && rulesName.nonEmpty,
      "Machine Facets: missing facet definition identity"
    )
    require(
      (traitClass ne null) && traitClass.isInterface && traitClass.isInstance(assembly),
      "Machine Facets: facet trait token does not own this assembly"
    )
    require(
      assembly.ownsFacetNamespace(namespace, traitName, namespaceName, traitClass),
      "Machine Facets: foreign or unregistered facet namespace receiver"
    )
    val identity =
      new FacetSource.Identity(assembly, namespace, traitClass, traitName, namespaceName, rulesName)
    origin match
      case Some(previous) =>
        require(
          previous.source.identity.exists(old =>
            old.source.eq(assembly) && old.namespace.eq(namespace) && old.traitClass.eq(
              traitClass
            ) &&
              old.traitName == traitName && old.namespaceName == namespaceName && old.rulesName == rulesName
          ),
          "Machine Facets: conflicting RuleBook origin"
        )
      case None =>
        val provenance = new FacetSource(source.facet, source.section, Some(identity))
        origin = Some(new FacetBindings.Origin(this, provenance))
  }
  override private[framework] def facetSource: Option[String] = source.facet
  override private[framework] def sectionName: String = source.section
  override private[framework] def actionName(decl: ActionDecl): String = names(decl)
  private val phase: S => P = phasing.projection
  private val names = mutable.Map.empty[ActionDecl, String]
  private val written = mutable.ArrayBuffer.empty[Rule[S, O, F]]
  private val never = mutable.ArrayBuffer.empty[ActionDecl]
  private val capturedDisabled = mutable.Map.empty[ActionDecl, Bound[S, O, F]]
  private val order = mutable.LinkedHashSet.empty[ActionDecl]
  private val generatedCrashes = mutable.Set.empty[ActionDecl]
  private var open: Option[String] = None // scalafix:ok DisableSyntax.var
  // The `from` the rules are in, if any: its declarer's name and the actions it declares.
  private var within: Option[(String, Set[ActionDecl])] =
    None // scalafix:ok DisableSyntax.var

  // A generated crash uses the machine's retained field classification. Core form:
  // `on(action) { always ~> generatedReset }`.
  inline def crashes(inline action: Action[EmptyTuple], outcome: O, facts: F*): Unit =
    RuleSource.qualified(action, "crashes", groupName)
    registerCrash(
      crashBinding(owner, action, Vector.empty, outcome, facts, NativeLocation.here),
      RuleSource.spelling(action)
    )

  // The same reset consumes the firing crash's own declared budget. Core form:
  // `on(action.action) { always ~> generatedResetAndConsume }`.
  inline def crashes(inline action: BudgetedAction[S, EmptyTuple], outcome: O, facts: F*): Unit =
    RuleSource.qualified(action, "crashes", groupName)
    val target = action
    registerCrash(
      crashBinding(owner, target.action, target.budgets, outcome, facts, NativeLocation.here),
      RuleSource.spelling(action)
    )

  private[framework] def registerCrash(binding: StepBinding[S, O, F], spelling: String): Unit =
    require(
      !order.contains(binding.decl),
      s"crashes($spelling) at ${binding.decl.nativeFaultLocation}: duplicate crash binding"
    )
    block[EmptyTuple](binding.decl, None, spelling):
      always ~> binding.function
        .asInstanceOf[S => List[Step[S, O, F]]] // scalafix:ok DisableSyntax.asInstanceOf
    generatedCrashes += binding.decl: Unit

  // Runs one block's cases, refusing a block in a block, and in a `from` an action its declarer
  // does not declare.
  private[framework] def block[I](
      decl: ActionDecl,
      values: Option[List[Any]],
      code: String,
      budgets: Vector[NativeBudget[?]] = Vector.empty
  )(cases: Firing[S, O, F, I] ?=> Unit): Unit =
    val action = writtenAction(code)
    require(
      !generatedCrashes.contains(decl),
      s"on($action) at ${decl.nativeFaultLocation}: duplicate crash binding"
    )
    names(decl) = action
    require(
      writtenPath(code).contains('.'),
      s"on($action) names its action bare: name it through the object that declares it, " +
        s"`on(${within.fold("<declarer>")(_._1)}.$action)`"
    )
    require(
      open.isEmpty,
      s"on($action) sits in on(${open.get}): a block holds cases alone"
    )
    require(!never.contains(decl), s"$action is disabled and fired by a rule")
    for (declarer, declared) <- within do
      require(
        declared.contains(decl),
        s"on($action) sits in from($declarer), and $declarer declares no $action: a from holds " +
          "the blocks of the actions its declarer declares"
      )
    open = Some(action)
    val firing =
      Firing[S, O, F, I](
        decl,
        values,
        action,
        (c, effect, captured) => bind(decl, values, c, effect, code, captured),
        Some(owner.machine),
        origin,
        budgets
      )
    try cases(using firing)
    finally open = None

  // Registers a rule in declaration order.
  private def bind(
      decl: ActionDecl,
      values: Option[List[Any]],
      c: Case[S, O, F],
      effect: (S, List[Any]) => List[Step[S, O, F]],
      code: String,
      captured: Option[RuleBindings.Registration]
  ): Unit =
    val rule = Rule(
      written.size + 1,
      c.heading,
      writtenAction(code),
      decl,
      values,
      c.guard,
      effect
    )
    rule.publishCapture(captured)
    written += rule
    order += decl

  // The cases of a whole action: `on(clerk.ship): when(Phase.placed) ~> effects.send`. Core form: the
  // action's step function, whose arms are the cases in order.
  inline def on[I](inline a: Action[I])(
      cases: Firing[S, O, F, NativeInput[I]] ?=> Unit
  )(using NotGiven[I =:= EmptyTuple]): Unit =
    RuleSource.qualified(a, "on", groupName)
    block[NativeInput[I]](a.decl, None, codeOf(a))(cases)

  // Rules for the fault whose selected native field supplies its budget. Core form:
  // `on(a.action) { cases }`, retaining its binding-specific field metadata.
  inline def on[I](inline a: BudgetedAction[S, I])(
      cases: Firing[S, O, F, NativeInput[I]] ?=> Unit
  )(using NotGiven[I =:= EmptyTuple]): Unit =
    RuleSource.qualified(a, "on", groupName)
    val target = a
    block[NativeInput[I]](target.action.decl, None, RuleSource.spelling(a), target.budgets)(cases)

  @targetName("onBudgetedWithoutInput")
  inline def on(inline a: BudgetedAction[S, EmptyTuple])(
      cases: Firing[S, O, F, EmptyTuple] ?=> Unit
  ): Unit =
    RuleSource.qualified(a, "on", groupName)
    val target = a
    block[EmptyTuple](target.action.decl, None, RuleSource.spelling(a), target.budgets)(cases)

  @targetName("onBudgetedScalarInput")
  inline def on[I](inline a: BudgetedScalarAction[S, I])(
      cases: Firing[S, O, F, NativeInput[I]] ?=> Unit
  ): Unit =
    RuleSource.qualified(a, "on", groupName)
    val target = a
    block[NativeInput[I]](target.action.decl, None, RuleSource.spelling(a), target.budgets)(cases)

  @targetName("onBudgetedScalarEmptyTuple")
  inline def on(inline a: BudgetedScalarAction[S, EmptyTuple])(
      cases: Firing[S, O, F, NativeInput[EmptyTuple]] ?=> Unit
  ): Unit =
    RuleSource.qualified(a, "on", groupName)
    val target = a
    block[NativeInput[EmptyTuple]](
      target.action.decl,
      None,
      RuleSource.spelling(a),
      target.budgets
    )(cases)

  @targetName("onScalarInput")
  inline def on[I](inline a: ScalarAction[I])(
      cases: Firing[S, O, F, NativeInput[I]] ?=> Unit
  ): Unit =
    RuleSource.qualified(a, "on", groupName)
    block[NativeInput[I]](a.decl, None, codeOf(a))(cases)

  @targetName("onWithoutInput")
  inline def on(inline a: Action[EmptyTuple])(
      cases: Firing[S, O, F, EmptyTuple] ?=> Unit
  ): Unit =
    RuleSource.qualified(a, "on", groupName)
    block[EmptyTuple](a.decl, None, codeOf(a))(cases)

  @targetName("onScalarEmptyTuple")
  inline def on(inline a: ScalarAction[EmptyTuple])(
      cases: Firing[S, O, F, NativeInput[EmptyTuple]] ?=> Unit
  ): Unit =
    RuleSource.qualified(a, "on", groupName)
    block[NativeInput[EmptyTuple]](a.decl, None, codeOf(a))(cases)

  // The cases of one class of an action, whose effects read the state alone:
  // `on(courier.report(Report.delivered)): when(sent) ~> effects.deliver`. Core form: the arms
  // `if g(s) && result == Report.delivered then deliver(s)` of the action's step function.
  inline def on(inline c: Class)(
      cases: Firing[S, O, F, EmptyTuple] ?=> Unit
  ): Unit =
    RuleSource.qualified(c, "on", groupName)
    val target = c
    block[EmptyTuple](target.decl, Some(target.values), RuleSource.spelling(c))(cases)

  // The same cases for two actions, or classes of them, whose effects read the state alone:
  // `on(clerk.hold, clerk.cancel): when(shipped) ~> rejects(Rejection.failedPrecondition)`.
  // Fixed arities of two to four, since `codeOf` names no element of an `inline` varargs list.
  // Core form: one block per target, `on(clerk.hold): cases` and `on(clerk.cancel): cases`.
  inline def on(inline a: Action[?] | Class, inline b: Action[?] | Class)(
      cases: Firing[S, O, F, EmptyTuple] ?=> Unit
  ): Unit =
    RuleSource.qualified(a, "on", groupName)
    RuleSource.qualified(b, "on", groupName, 1)
    each(List(a -> RuleSource.spelling(a), b -> RuleSource.spelling(b, 1)))(cases)

  // The same cases for three actions, or classes of them. Core form: one block per target with
  // the same cases.
  inline def on(
      inline a: Action[?] | Class,
      inline b: Action[?] | Class,
      inline c: Action[?] | Class
  )(cases: Firing[S, O, F, EmptyTuple] ?=> Unit): Unit =
    RuleSource.qualified(a, "on", groupName)
    RuleSource.qualified(b, "on", groupName, 1)
    RuleSource.qualified(c, "on", groupName, 2)
    each(
      List(
        a -> RuleSource.spelling(a),
        b -> RuleSource.spelling(b, 1),
        c -> RuleSource.spelling(c, 2)
      )
    )(cases)

  // The same cases for four actions, or classes of them. Core form: one block per target with the
  // same cases.
  inline def on(
      inline a: Action[?] | Class,
      inline b: Action[?] | Class,
      inline c: Action[?] | Class,
      inline d: Action[?] | Class
  )(cases: Firing[S, O, F, EmptyTuple] ?=> Unit): Unit =
    RuleSource.qualified(a, "on", groupName)
    RuleSource.qualified(b, "on", groupName, 1)
    RuleSource.qualified(c, "on", groupName, 2)
    RuleSource.qualified(d, "on", groupName, 3)
    each(
      List(
        a -> RuleSource.spelling(a),
        b -> RuleSource.spelling(b, 1),
        c -> RuleSource.spelling(c, 2),
        d -> RuleSource.spelling(d, 3)
      )
    )(
      cases
    )

  // Runs the cases once for each target, as its own block, refusing a target named twice. The
  // cases run once per target, so each binds its own rules.
  private[framework] def each(
      targets: List[(Action[?] | Class, String)]
  )(cases: Firing[S, O, F, EmptyTuple] ?=> Unit): Unit =
    val named = targets.map:
      case (a: Action[?], code) => (a.decl, None, code)
      case (c: Class, code)     => (c.decl, Some(c.values), code)
    for ((decl, values, code), i) <- named.zipWithIndex do
      require(
        !named
          .take(i)
          .exists((d, v, _) => d == decl && java.util.Objects.equals(v, values)),
        s"${writtenAction(code)} is named twice in one on: name each action, or class of it, once"
      )
    for (decl, values, code) <- named do block[EmptyTuple](decl, values, code)(cases)

  // The blocks of the actions `declarer` declares, each naming its action through its declarer as
  // every block does: `from(clerk):` containing `on(clerk.ship):`. It only groups, and brings none of
  // the declarer's members into scope: a block means in a `from` what it means outside one, and the
  // IR generator refuses an `import clerk.*` in it. Core form: the blocks it holds,
  // `on(clerk.ship): ...`.
  private[framework] def groupName: String = within.fold("<declarer>")(_._1)

  inline def from(declarer: AnyRef)(inline body: Unit): Unit =
    grouped(declarer, RuleSource.imports(body))(body)

  private[framework] def grouped(declarer: AnyRef, imported: String)(body: => Unit): Unit =
    val name = declarer match
      case a: Actor => a.name
      case other    => objectName(other)
    require(
      imported.isEmpty,
      s"from($name) imports members at $imported: a from only groups, so drop the import and " +
        s"name each action through its declarer, `on($name.<action>)`"
    )
    require(
      open.isEmpty,
      s"from($name) sits in on(${open.get}): a block holds cases alone"
    )
    require(
      within.isEmpty,
      s"from($name) sits in from(${within.get._1}): a from holds on blocks alone"
    )
    within = Some(name -> declaredBy(declarer))
    try body
    finally within = None

  // The actions an object declares: the `Action` values among its public vals. An action records
  // no object that holds it, so the object's members are read.
  private def declaredBy(declarer: AnyRef): Set[ActionDecl] =
    declarer.getClass.getMethods.toSet
      .filter(m =>
        m.getParameterCount == 0 && classOf[Action[?]]
          .isAssignableFrom(m.getReturnType)
      )
      .map(_.invoke(declarer))
      .collect { case a: Action[?] => a.decl }

  // The retired rule-case spelling is refused where it is written, while `value.in(...)` remains
  // the membership predicate below. Core form: none; write `when(...)` for a rule case.
  inline def in[Q](@unused first: Q, @unused rest: Q*): Nothing =
    error("in(...) is membership alone: write `when(...)` for a rule case")

  // The retired role spelling is refused where it is written, while `value.in[R]` remains the
  // membership predicate below. Core form: none; write `when[R]` for a role rule case.
  inline def in[R]: Nothing =
    error("in[R] is membership alone: write `when[R]` for a role rule case")

  private[framework] def phases(
      set: P => Boolean,
      code: String
  ): Case[S, O, F] =
    Case(s"when(${code.trim})", s => set(phase(s)))

  // A case that holds in the phases listed, each named through its phase type:
  // `when(Phase.placed, Phase.open)`; the IR generator refuses a bare phase. Core form:
  // `List(p1, p2).contains(s.phase)`.
  def when[Q](first: Q, rest: Q*)(using PhasesOf[P, Q]): Case[S, O, F] =
    val phases = first +: rest
    Case(s"when(${phases.mkString(", ")})", s => phases.contains(phase(s)))

  // A case that holds in a named set of phases: `when(states.terminal)`. Core
  // form: `terminal(s.phase)`.
  inline def when(inline set: P => Boolean)(using
      PhasesOf[P, P]
  ): Case[S, O, F] =
    phases(set, codeOf(set))

  // A case that holds in the phases with the role `R` (model/framework/dsl/Roles.scala), as the projection
  // of the machine's `Phased` reads them: `when[Closed] ~> effects.notFound`; a machine that is
  // not `Phased` names no role. Core form:
  // `List(<the cases of R, in declaration order>).contains(s.phase)`.
  def when[R](using
      role: TypeTest[P, R],
      named: ClassTag[R],
      @unused projected: PhasesOf[P, P]
  ): Case[S, O, F] =
    Case(
      s"when[${named.runtimeClass.getSimpleName}]",
      s => role.unapply(phase(s)).nonEmpty
    )

  // Actions no state enables, which the machine binds all the same, such as a courier's strike it does
  // not feel. Core form: `action ~> (_ => Nil)`.
  inline def disabled(inline actions: Action[?]*): Unit =
    disable(RuleSource.disabledLabels(actions), actions*)

  private[framework] def capturedDisabledRules(actions: Action[?]*)(data: Seq[String]): Unit =
    val retained = RuleBindings.retainedDisabled(owner.machine, actions.map(_.decl), data, origin)
    disable(retained.map(_._1), actions*)
    actions.zip(retained).foreach((action, row) => capturedDisabled(action.decl) = row._2)

  private[framework] def disable(labels: Vector[String], actions: Action[?]*): Unit =
    for (a, label) <- actions.zip(labels) do
      require(
        !never.contains(a.decl) && !written.exists(_.decl == a.decl),
        "an action is disabled twice, or disabled and bound by a rule"
      )
      never += a.decl
      order += a.decl
      names(a.decl) = label

  // `phase.in(a, b)` in a condition, as outside the rules. Core form: `List(a, b).contains(phase)`.
  extension [A](value: A)
    @sugar def in(first: A, rest: A*): Boolean = (first +: rest).contains(value)

  // `phase.in[Closed]` in a condition, as outside the rules. Core form:
  // `List(<the cases of Closed, in declaration order>).contains(phase)`.
  extension [A](value: A)
    @sugar def in[R](using role: TypeTest[A, R]): Boolean =
      role.unapply(value).nonEmpty

  private[framework] def table: Vector[(ActionDecl, Bound[S, O, F])] =
    order.toVector.map: decl =>
      decl -> (if never.contains(decl) then
                 capturedDisabled.getOrElse(decl, Bound.Disabled[S, O, F]())
               else Bound.Ruled(written.filter(_.decl == decl).toVector))

// Evidence that a case's phases are of the type the rules' projection reads: the rules of a machine
// that mixes in `Phased[State, Phase](_.phase)` name phases with `when`, and the rules of
// one that is not `Phased`, whose phase type is `Nothing`, name none. Core form: none of its own;
// `when(p1, p2)` is `List(p1, p2).contains(s.phase)`.
@implicitNotFound(
  "when names phases of ${Q}, and these rules read phases of ${P}: mix the projection the " +
    "phases are of into the machine, `Phased[State, Phase](_.phase)`"
)
final class PhasesOf[P, Q] private ()

// The one projection a case's phases are of: its own. Core form: `List(p1, p2).contains(s.phase)`.
object PhasesOf:
  // Phases of the projection's own type, other than `Nothing`, or of a narrower one: a case that
  // takes a role, `case done extends Phase, Succeeded`, is a `Phase & Succeeded`. Core form:
  // `List(p1, p2).contains(s.phase)`, whose phases are of the type of `s.phase`.
  given [P, Q <: P](using NotGiven[P =:= Nothing]): PhasesOf[P, Q] = PhasesOf()

// The rules of one action a derivation binds in its source's place:
// `rebind(on(clerk.ship): always ~> OrderRecord.effects.send)`, each case `where(g)` or
// `always`, since a derivation's rules name no phase. Its action is `inline`, as a rule block's is
// (`Rules`), so that an overlap names it as written. Core form: the step function
// `action ~> ((s, i) => if g(s) then e(s, i) else Nil)`.
inline def on[S, O, F, I](using
    owner: Owner[S, O, F]
)(inline a: Action[I])(
    cases: Firing[S, O, F, NativeInput[I]] ?=> Unit
)(using NotGiven[I =:= EmptyTuple]): RuleGroup[S, O, F] =
  RuleSource.qualified(a, "on", "<declarer>")
  derivedRules(owner, a.decl, codeOf(a))(cases)

// A native scalar action's cases. Core form: `action ~> ((s, in) => if g(s, in) then e(s, in) else Nil)`.
@targetName("onScalarInput")
inline def on[S, O, F, I](using
    owner: Owner[S, O, F]
)(inline a: ScalarAction[I])(
    cases: Firing[S, O, F, NativeInput[I]] ?=> Unit
): RuleGroup[S, O, F] =
  RuleSource.qualified(a, "on", "<declarer>")
  derivedRules(owner, a.decl, codeOf(a))(cases)

// A state-only action's cases. Core form: `action ~> (s => if g(s) then e(s) else Nil)`.
@targetName("onWithoutInput")
inline def on[S, O, F](using
    owner: Owner[S, O, F]
)(inline a: Action[EmptyTuple])(
    cases: Firing[S, O, F, EmptyTuple] ?=> Unit
): RuleGroup[S, O, F] =
  RuleSource.qualified(a, "on", "<declarer>")
  derivedRules(owner, a.decl, codeOf(a))(cases)

// A declared EmptyTuple scalar's cases. Core form: `action ~> ((s, in) => if g(s, in) then e(s, in) else Nil)`.
@targetName("onScalarEmptyTuple")
inline def on[S, O, F](using
    owner: Owner[S, O, F]
)(inline a: ScalarAction[EmptyTuple])(
    cases: Firing[S, O, F, NativeInput[EmptyTuple]] ?=> Unit
): RuleGroup[S, O, F] =
  RuleSource.qualified(a, "on", "<declarer>")
  derivedRules(owner, a.decl, codeOf(a))(cases)

// The rules of a derivation's `on`, each named by its action as `written` spells it.
private[framework] def derivedRules[S, O, F, I](
    owner: Owner[S, O, F],
    decl: ActionDecl,
    written: String
)(cases: Firing[S, O, F, I] ?=> Unit): RuleGroup[S, O, F] =
  val action = writtenAction(written)
  require(
    writtenPath(written).contains('.'),
    s"on($action) names its action bare: name it through the object that declares it, " +
      s"`on(<declarer>.$action)`"
  )
  val rules = mutable.ArrayBuffer.empty[Rule[S, O, F]]
  val firing = Firing[S, O, F, I](
    decl,
    None,
    action,
    (c, effect, captured) =>
      val rule =
        Rule(rules.size + 1, c.heading, action, decl, None, c.guard, effect)
      rule.publishCapture(captured)
      rules += rule
    ,
    Some(owner.machine)
  )
  cases(using firing)
  val group = RuleGroup(rules.toVector)
  group.publishOrigin(
    rules.headOption
      .flatMap(_.retainedRule)
      .filter(_.registration.packet.derived)
      .map(r => DerivationBindings.GroupOrigin(group, owner.machine, decl, r.registration.packet))
  )
  group

// A plain definition expanded from its body where a lifted function calls it. Core form: its `body`.
final class sugar extends scala.annotation.StaticAnnotation
