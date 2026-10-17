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
import scala.compiletime.codeOf
import scala.reflect.{ClassTag, TypeTest}
import scala.util.NotGiven
import scala.language.implicitConversions

// The outcome `enter` and `stay` answer for a machine whose outcomes are `O`, declared once beside
// the outcome type: `given Ok[Outcome] = Ok(Outcome.accepted)`. Core form: the outcome itself,
// `Outcome.accepted`, written in each `Step`.
final case class Ok[O](outcome: O) extends Accepting[O]

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

// `where(g) ~> rejects(Rejection.notFound)`, in a rule of a machine on the shared outcomes: the
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
@implicitNotFound("before is the state a step started from; use it in a postcondition")
final class Before[S] private[framework] (val state: S)

// The typed step whose requirements an expectation reads. Inside a Machine, `after(action):` is
// the word of a postcondition (ClaimWords). Core form: `s` in `s.facts`.
def after[S, O, F](using context: Expectation[S, O, F]): Step[S, O, F] = context.step

// What `after` is in a postcondition of a Machine (ClaimWords): the block about the steps of
// `target`, `after(heater.failover): check(beeping)`, which must hold an assertion. Core form:
// `!(a.action == target) || requirements`.
final class StepTarget:
  inline def apply(target: Action[?] | Class | Composed)(inline block: Unit): Unit =
    ${ ClaimMacros.targeted('target, 'block) }

// The state before a transition's step in an `across` block: for a Machine that declares its state
// as vars, its State record, whose vars a postcondition reads by name, `before.temperature`; inside a
// Machine, the State record a postcondition's step started from (ClaimWords). Core form: `b` in
// `(b, a) => a.state == b`.
transparent inline def before[S](using context: Before[S]): Any =
  ${ ExpectationMacros.stateBefore[S]('context) }

// One assertion of a claim, reported on its own where it fails: `val neverBeeps = check(!beeping)` in
// `invariants`, or in a block of the retired `expect`, a requirement whose failure skips the rest. It
// records nothing outside an evaluation, and is refused in `progress`. Core form:
// `condition && remainingRequirements`.
inline def check(inline condition: Boolean): Unit = ${ ClaimMacros.check('condition) }

// Fact requirements specialized to the current step before typing a member selector. Inside a
// Machine, `records(fact)` asserts the fact in a claim and records it in an effect (ClaimWords).
// Core form: `s.facts.contains(fact)` or `s.facts.contains("member_fact")`.
def records[S, O, F](using context: Expectation[S, O, F]): RequirementFacts[S, O, F] =
  new RequirementFacts(context)

// What `records` is in a claim or an effect of a Machine whose facts are `F` (ClaimWords): in a
// claim the assertion that the step recorded the fact, in an effect a fact the step records.
// Core form: `a.facts.contains(fact)`, or the facts of `Step(outcome, state, facts)`.
final class StepRecords[F]:
  inline def apply(fact: F): Unit = ${ ClaimMacros.fact('fact) }

// The outcome the step answered, in a postcondition or a monitor, `if outcome != Outcome.accepted then
// check(balances == before.balances)`, typed with the outcomes of the Machine it is in. Core form:
// `a.outcome` in `(b, a) => predicate(b, a)`.
transparent inline def outcome: Any = ${ ClaimMacros.outcome }

// The callable fact requirement in a block. Core form: `s.facts.contains(fact)`.
final class RequirementFacts[S, O, F] private[framework] (
    private[framework] val context: Expectation[S, O, F]
):
  // A fact of this step's own machine. Core form: `s.facts.contains(fact)`.
  def apply(fact: F): Unit = context.require(context.step.facts.contains(fact))

  // A fact of the exact selected field and machine. Core form: `s.facts.contains("member_fact")`.
  inline def apply[MS, MO, MF](
      inline member: S => (MS, RecordMachine[MS, MO, MF]),
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

// At least one explicit branch must hold; later branches are skipped after success. Inside a
// Machine, the alternatives of a claim (ClaimWords). Core form:
// `firstBranch || secondBranch || remainingBranches`.
def oneOf[S, O, F](using
    context: Expectation[S, O, F]
)(body: Alternatives[S, O, F] ?=> Unit): Unit =
  context.require(new Alternatives(context.step).evaluate(body))

// What `oneOf` is in a block of the retired `expect`. Core form: `firstBranch || secondBranch`.
final class ExpectedAlternatives[S, O, F] @scala.annotation.publicInBinary private[framework] (
    context: Expectation[S, O, F]
):
  def apply(body: Alternatives[S, O, F] ?=> Unit): Unit =
    context.require(new Alternatives(context.step).evaluate(body))

// What `oneOf` is in a claim: its branches, of which there are at least two. Core form:
// `firstBranch || secondBranch || rest`.
final class ClaimAlternatives:
  inline def apply(inline body: Branching ?=> Unit): Unit = ${ ClaimMacros.alternatives('body) }

// The evidence a `branch` of a claim's `oneOf` reads, that it is written inside one. Core form: none
// of its own; it is the `||` between the branches.
final class Branching

// Explicit finite input catalogs. Core form: `Finite.of(values*)` with checked membership.
object domains:
  // Explicit choices in authored order. Core form: `Finite.of(values*)`.
  def oneOf[A](values: A*)(using Data.Structural[A]): Finite[A] = DomainChoices.create(values)
  // Integer choices in Range iteration order. Core form: `Finite.of(values*)`.
  def oneOf(values: Range): Finite[Int] = DomainChoices.range(values)

extension [In, A](field: InputField[In, A])
  // The catalog selected for this typed field. Core form: `InputBinding.make(field, domain)`.
  def in(domain: Finite[A]): InputBinding[In] = InputBinding.make(field, domain)

// Conjunctive requirements of one alternative, only inside oneOf; inside a Machine, an alternative
// of a claim's oneOf (ClaimWords). Core form: `(first && second)` as an operand of
// `firstBranch || secondBranch`.
def branch[S, O, F](using
    alternatives: Alternatives[S, O, F]
)(body: Expectation[S, O, F] ?=> Unit): Unit = alternatives.attempt(body)

// What `branch` is in a block of the retired `expect`. Core form: `(first && second)`.
final class ExpectedBranch[S, O, F] @scala.annotation.publicInBinary private[framework] (
    alternatives: Alternatives[S, O, F]
):
  def apply(body: Expectation[S, O, F] ?=> Unit): Unit = alternatives.attempt(body)

// What `branch` is in a claim's `oneOf`. Core form: `(first && second)`.
final class ClaimBranch:
  inline def apply(inline body: Unit): Unit = ${ ClaimMacros.branchBlock('body) }

@scala.annotation.publicInBinary
private[framework] def requirementFacts[S, O, F](
    context: Expectation[S, O, F]
): RequirementFacts[S, O, F] = new RequirementFacts(context)

@scala.annotation.publicInBinary
private[framework] def expectedAlternatives[S, O, F](
    context: Expectation[S, O, F]
): ExpectedAlternatives[S, O, F] = new ExpectedAlternatives(context)

@scala.annotation.publicInBinary
private[framework] def expectedBranch[S, O, F](
    alternatives: Alternatives[S, O, F]
): ExpectedBranch[S, O, F] = new ExpectedBranch(alternatives)

// The compile-time half of the claim words: each reads where it is written. Inside a block of the
// retired `expect`, whose expectation it finds as a given, it is the requirement it was; in a claim
// section of a Machine it asserts natively (NativeClaims), and outside the sections it belongs in it
// is refused with the spec's text.
@scala.annotation.publicInBinary
private[framework] object ClaimMacros:
  import scala.quoted.*

  private val claimSections = Set("invariants$", "postconditions$", "queries$")
  private val stepSections = Set("postconditions$", "monitors$", "queries$")
  private val emptyBlock = "a requirement block must not be empty or contain only helper vals"

  private def owners(using q: Quotes): List[q.reflect.Symbol] =
    import q.reflect.*
    Iterator.iterate(Symbol.spliceOwner)(_.owner).takeWhile(!_.isNoSymbol).toList

  private def inside(names: Set[String])(using q: Quotes): Boolean =
    owners.exists(s => s.isClassDef && names(s.name))

  // The member the expansion sits in: the val, or the def, of a section that declares the claim.
  private def member(using q: Quotes): Expr[String] =
    import q.reflect.*
    Expr(
      owners
        .find(s =>
          (s.isValDef || s.isDefDef) && !s.flags.is(Flags.Synthetic) && !s.name.contains("$") &&
            !s.name.startsWith("<") && s.name != "macro"
        )
        .fold("")(_.name)
    )

  private def line(using q: Quotes): Expr[Int] =
    Expr(q.reflect.Position.ofMacroExpansion.startLine + 1)

  private def machine(using q: Quotes): Option[q.reflect.Symbol] =
    import q.reflect.*
    val base = TypeRepr.of[RecordMachine[?, ?, ?]].typeSymbol
    owners.find(s => s.isClassDef && !s.equals(base) && s.typeRef.baseClasses.contains(base))

  // The outcome (0) or fact (1) type of the Machine the expansion sits in.
  private def machineType(using q: Quotes)(index: Int, word: String): q.reflect.TypeRepr =
    import q.reflect.*
    val base = TypeRepr.of[RecordMachine[?, ?, ?]].typeSymbol
    machine
      .map(_.typeRef.baseType(base))
      .collect { case AppliedType(_, List(_, o, f)) => if index == 0 then o else f }
      .getOrElse(report.errorAndAbort(s"$word reads a step of a Machine: write it in one"))

  private def machineRef(using q: Quotes)(cls: q.reflect.Symbol): Expr[RecordMachine[?, ?, ?]] =
    import q.reflect.*
    val ref = if cls.flags.is(Flags.Module) then Ref(cls.companionModule) else This(cls)
    ref.asExprOf[RecordMachine[?, ?, ?]]

  private def arguments(using q: Quotes)(tpe: q.reflect.TypeRepr, of: q.reflect.Symbol) =
    tpe.widen.baseType(of).typeArgs

  // The calls of a block to one of the claim words `named`, as written, or expanded (the claim
  // words are inline, and a block an enclosing word reads may hold either), not looking inside one.
  private def calls(using q: Quotes)(block: q.reflect.Term, named: Set[String]): Int =
    import q.reflect.*
    def word(s: Symbol): Option[String] =
      if !s.exists then None
      else if s.name == "apply" then
        Map(
          "ClaimAlternatives" -> "oneOf",
          "ClaimBranch" -> "branch",
          "StepRecords" -> "records",
          "StepTarget" -> "after"
        ).get(s.owner.name)
      else if s.owner.name == "NativeClaims$" then
        Map(
          "assertion" -> "check",
          "recordFact" -> "records",
          "alternatives" -> "oneOf",
          "alternative" -> "branch",
          "targeting" -> "after"
        ).get(s.name)
      else if s.name == "check" && s.owner.name.startsWith("Syntax$package") then Some("check")
      else None
    var found = 0
    new TreeTraverser:
      override def traverseTree(tree: Tree)(owner: Symbol): Unit = tree match
        case call: Apply if word(call.fun.symbol).exists(named) => found += 1
        case _                                                  => super.traverseTree(tree)(owner)
    .traverseTree(block)(Symbol.spliceOwner)
    found

  // Whether a block asserts anything: a check, a fact or alternatives.
  private def asserts(using q: Quotes)(block: q.reflect.Term): Boolean =
    calls(block, Set("check", "records", "oneOf")) > 0

  @scala.annotation.publicInBinary
  private[framework] def check(condition: Expr[Boolean])(using q: Quotes): Expr[Unit] =
    import q.reflect.*
    Expr.summon[Expectation[?, ?, ?]] match
      case Some(context) => '{ $context.require($condition) }
      case None          =>
        if inside(Set("progress$")) then
          report.errorAndAbort(
            "an invariant or postcondition belongs in invariants or postconditions"
          )
        val text = Expr(condition.asTerm.pos.sourceCode.getOrElse(condition.show))
        '{ NativeClaims.assertion($condition, $member, $line, $text) }

  @scala.annotation.publicInBinary
  private[framework] def after(using q: Quotes): Expr[Any] =
    import q.reflect.*
    Expr.summon[Expectation[?, ?, ?]] match
      case Some(context) =>
        arguments(context.asTerm.tpe, TypeRepr.of[Expectation[?, ?, ?]].typeSymbol) match
          case List(s, o, f) =>
            (s.asType, o.asType, f.asType) match
              case ('[s], '[o], '[f]) =>
                '{ ${ context.asInstanceOf[Expr[Expectation[s, o, f]]] }.step }
              case _ => report.errorAndAbort("a step's types are unknown here")
          case _ => '{ $context.step }
      case None =>
        if !inside(Set("postconditions$", "queries$")) then
          report.errorAndAbort("after(action) is a postcondition: write it in postconditions")
        '{ new StepTarget() }

  @scala.annotation.publicInBinary
  private[framework] def targeted(target: Expr[Action[?] | Class | Composed], block: Expr[Unit])(
      using q: Quotes
  ): Expr[Unit] =
    import q.reflect.*
    if !asserts(block.asTerm) then report.errorAndAbort(emptyBlock, block.asTerm.pos)
    '{ NativeClaims.targeting($target, $member)($block) }

  @scala.annotation.publicInBinary
  private[framework] def before(using q: Quotes): Expr[Any] =
    import q.reflect.*
    Expr.summon[Before[?]] match
      case Some(context) =>
        arguments(context.asTerm.tpe, TypeRepr.of[Before[?]].typeSymbol).headOption
          .getOrElse(TypeRepr.of[Any])
          .asType match
          case '[s] => ExpectationMacros.stateBefore[s](context.asExprOf[Before[s]])
      case None =>
        if !inside(stepSections) then
          report.errorAndAbort("before is the state a step started from; use it in a postcondition")
        val cls = machine.getOrElse(
          report.errorAndAbort("before is the state a step of a Machine started from")
        )
        val record = VarSchemaMacro
          .enclosingMachine(Symbol.spliceOwner)
          .fold(TypeRepr.of[State]): declared =>
            VarSchemaMacro
              .stateVars(declared)
              .foldLeft(TypeRepr.of[State]): (record, variable) =>
                Refinement(record, variable.name, VarSchemaMacro.declaredType(declared, variable))
        val owner = machineRef(cls)
        record.asType match
          case '[r] =>
            '{
              NativeClaims
                .start($owner.init.asInstanceOf[State]) // scalafix:ok DisableSyntax.asInstanceOf
                .asInstanceOf[r] // scalafix:ok DisableSyntax.asInstanceOf
            }

  @scala.annotation.publicInBinary
  private[framework] def outcome(using q: Quotes): Expr[Any] =
    import q.reflect.*
    if !inside(stepSections) then
      report.errorAndAbort(
        "outcome is what a step answered: read it in a postcondition or a monitor"
      )
    machineType(0, "outcome").asType match
      case '[o] =>
        val fallback = Expr.summon[Ok[o]].fold('{ None })(ok => '{ Some($ok.outcome) })
        '{
          NativeClaims
            .answer($fallback.orNull)
            .asInstanceOf[o] // scalafix:ok DisableSyntax.asInstanceOf
        }

  @scala.annotation.publicInBinary
  private[framework] def records(using q: Quotes): Expr[Any] =
    import q.reflect.*
    Expr.summon[Expectation[?, ?, ?]] match
      case Some(context) =>
        arguments(context.asTerm.tpe, TypeRepr.of[Expectation[?, ?, ?]].typeSymbol) match
          case List(s, o, f) =>
            (s.asType, o.asType, f.asType) match
              case ('[s], '[o], '[f]) =>
                '{ requirementFacts(${ context.asInstanceOf[Expr[Expectation[s, o, f]]] }) }
              case _ => report.errorAndAbort("a step's types are unknown here")
          case _ => report.errorAndAbort("records reads the step of an expectation")
      case None =>
        machineType(1, "records").asType match
          case '[f] => '{ new StepRecords[f]() }

  @scala.annotation.publicInBinary
  private[framework] def fact[F: Type](value: Expr[F])(using q: Quotes): Expr[Unit] =
    val claim = Expr(inside(claimSections ++ Set("monitors$")))
    '{ NativeClaims.recordFact($value, $member, $line, $claim) }

  @scala.annotation.publicInBinary
  private[framework] def oneOf(using q: Quotes): Expr[Any] =
    import q.reflect.*
    Expr.summon[Expectation[?, ?, ?]] match
      case Some(context) =>
        arguments(context.asTerm.tpe, TypeRepr.of[Expectation[?, ?, ?]].typeSymbol) match
          case List(s, o, f) =>
            (s.asType, o.asType, f.asType) match
              case ('[s], '[o], '[f]) =>
                '{
                  expectedAlternatives(${ context.asInstanceOf[Expr[Expectation[s, o, f]]] })
                }
              case _ => report.errorAndAbort("a step's types are unknown here")
          case _ => report.errorAndAbort("oneOf reads the step of an expectation")
      case None => '{ new ClaimAlternatives() }

  @scala.annotation.publicInBinary
  private[framework] def alternatives(body: Expr[Branching ?=> Unit])(using q: Quotes): Expr[Unit] =
    import q.reflect.*
    val branches = calls(body.asTerm, Set("branch"))
    if branches < 2 then
      report.errorAndAbort("oneOf requires at least two branches", body.asTerm.pos)
    '{ NativeClaims.alternatives($member, $line)($body(using new Branching())) }

  @scala.annotation.publicInBinary
  private[framework] def branch(using q: Quotes): Expr[Any] =
    import q.reflect.*
    Expr.summon[framework.dsl.Alternatives[?, ?, ?]] match
      case Some(alternatives) =>
        arguments(
          alternatives.asTerm.tpe,
          TypeRepr.of[framework.dsl.Alternatives[?, ?, ?]].typeSymbol
        ) match
          case List(s, o, f) =>
            (s.asType, o.asType, f.asType) match
              case ('[s], '[o], '[f]) =>
                '{
                  expectedBranch(${
                    alternatives.asInstanceOf[Expr[framework.dsl.Alternatives[s, o, f]]]
                  })
                }
              case _ => report.errorAndAbort("a step's types are unknown here")
          case _ => report.errorAndAbort("branch reads the step of an expectation")
      case None =>
        if Expr.summon[Branching].isEmpty then
          report.errorAndAbort("branch is an alternative of oneOf: write it inside oneOf")
        '{ new ClaimBranch() }

  @scala.annotation.publicInBinary
  private[framework] def branchBlock(body: Expr[Unit])(using q: Quotes): Expr[Unit] =
    import q.reflect.*
    if !asserts(body.asTerm) then report.errorAndAbort(emptyBlock, body.asTerm.pos)
    '{ NativeClaims.alternative($body) }

// An ordinary step predicate with statement requirements. Core form:
// `(s: Step[S, O, F]) => first(s) && second(s)`.
object expect:
  // Bare fields read the resulting state. Core form: `(s: Step[S, O, F]) => predicate(s)`.
  inline def apply[S, O, F](
      @capture("capturedApply") body: Expectation[S, O, F] ?=> Unit
  ): Step[S, O, F] => Boolean =
    ${ ExpectationMacros.same('body) }
  // The captured form of `apply`: the plugin passes the lifted root and its chunks.
  private[framework] inline def capturedApply[S, O, F](
      body: Expectation[S, O, F] ?=> Unit,
      root: String,
      chunks: Seq[String]
  ): Step[S, O, F] => Boolean =
    ${ ExpectationMacros.capturedSame('body, 'root, 'chunks) }
  // A transition predicate, with before-state evidence and resulting-state bare fields.
  // Core form: `(b: S, a: Step[S, O, F]) => predicate(b, a)`.
  inline def across[S, O, F](
      @capture("capturedAcross") body: (Before[S], Expectation[S, O, F]) ?=> Unit
  ): (S, Step[S, O, F]) => Boolean =
    ${ ExpectationMacros.transition('body) }
  // The captured form of `across`: the plugin passes the lifted root and its chunks.
  private[framework] inline def capturedAcross[S, O, F](
      body: (Before[S], Expectation[S, O, F]) ?=> Unit,
      root: String,
      chunks: Seq[String]
  ): (S, Step[S, O, F]) => Boolean =
    ${ ExpectationMacros.capturedTransition('body, 'root, 'chunks) }

// The captured predicate an expectation retains, its root and chunks, when the reify plugin
// compiled it.
private[framework] type ExpectationCapture = Option[(String, Seq[String])]

@scala.annotation.publicInBinary
private[framework] object ExpectationMacros:
  import scala.quoted.*

  // The state a transition's step started from, typed for a Machine that declares its state as vars
  // as its State record refined with the vars of the Machine the expansion sits in.
  @scala.annotation.publicInBinary
  private[framework] def stateBefore[S: Type](context: Expr[Before[S]])(using Quotes): Expr[Any] =
    import quotes.reflect.*
    if !(TypeRepr.of[S] =:= TypeRepr.of[State]) then '{ $context.state }
    else
      val record = VarSchemaMacro
        .enclosingMachine(Symbol.spliceOwner)
        .fold(TypeRepr.of[State]): machine =>
          VarSchemaMacro
            .stateVars(machine)
            .foldLeft(TypeRepr.of[State]): (record, variable) =>
              Refinement(record, variable.name, VarSchemaMacro.declaredType(machine, variable))
      record.asType match
        case '[r] => '{ $context.state.asInstanceOf[r] } // scalafix:ok DisableSyntax.asInstanceOf

  @scala.annotation.publicInBinary
  private[framework] def same[S: Type, O: Type, F: Type](body: Expr[Expectation[S, O, F] ?=> Unit])(
      using Quotes
  ): Expr[Step[S, O, F] => Boolean] = sameLinked(body, '{ None })

  @scala.annotation.publicInBinary
  private[framework] def transition[S: Type, O: Type, F: Type](
      body: Expr[(Before[S], Expectation[S, O, F]) ?=> Unit]
  )(using Quotes): Expr[(S, Step[S, O, F]) => Boolean] = transitionLinked(body, '{ None })

  @scala.annotation.publicInBinary
  private[framework] def property[S: Type, O: Type, F: Type](
      builder: Expr[PropertyBuilder[S, O, F]],
      body: Expr[Expectation[S, O, F] ?=> Unit]
  )(using Quotes): Expr[SameStepProperty[S, O, F]] =
    '{ $builder.sameStep(${ sameLinked(body, 'None) }) }

  @scala.annotation.publicInBinary
  private[framework] def transitionProperty[S: Type, O: Type, F: Type](
      builder: Expr[PropertyBuilder[S, O, F]],
      body: Expr[(Before[S], Expectation[S, O, F]) ?=> Unit]
  )(using Quotes): Expr[TransitionProperty[S, O, F]] =
    '{ $builder.transition(${ transitionLinked(body, 'None) }) }

  // The captured forms, which the reify plugin calls with the lifted root and its chunks.
  @scala.annotation.publicInBinary
  private[framework] def capturedSame[S: Type, O: Type, F: Type](
      body: Expr[Expectation[S, O, F] ?=> Unit],
      root: Expr[String],
      chunks: Expr[Seq[String]]
  )(using Quotes): Expr[Step[S, O, F] => Boolean] = sameLinked(body, '{ Some(($root, $chunks)) })

  @scala.annotation.publicInBinary
  private[framework] def capturedTransition[S: Type, O: Type, F: Type](
      body: Expr[(Before[S], Expectation[S, O, F]) ?=> Unit],
      root: Expr[String],
      chunks: Expr[Seq[String]]
  )(using Quotes): Expr[(S, Step[S, O, F]) => Boolean] =
    transitionLinked(body, '{ Some(($root, $chunks)) })

  @scala.annotation.publicInBinary
  private[framework] def capturedProperty[S: Type, O: Type, F: Type](
      builder: Expr[PropertyBuilder[S, O, F]],
      body: Expr[Expectation[S, O, F] ?=> Unit],
      root: Expr[String],
      chunks: Expr[Seq[String]]
  )(using Quotes): Expr[SameStepProperty[S, O, F]] =
    '{ $builder.sameStep(${ sameLinked(body, '{ Some(($root, $chunks)) }) }) }

  @scala.annotation.publicInBinary
  private[framework] def capturedTransitionProperty[S: Type, O: Type, F: Type](
      builder: Expr[PropertyBuilder[S, O, F]],
      body: Expr[(Before[S], Expectation[S, O, F]) ?=> Unit],
      root: Expr[String],
      chunks: Expr[Seq[String]]
  )(using Quotes): Expr[TransitionProperty[S, O, F]] =
    '{ $builder.transition(${ transitionLinked(body, '{ Some(($root, $chunks)) }) }) }

  private def sameLinked[S: Type, O: Type, F: Type](
      body: Expr[Expectation[S, O, F] ?=> Unit],
      linked: Expr[ExpectationCapture]
  )(using Quotes): Expr[Step[S, O, F] => Boolean] =
    val prepared = ExpectationBlock.validate(body)
    '{ sameExpectation($prepared, $linked) }

  private def transitionLinked[S: Type, O: Type, F: Type](
      body: Expr[(Before[S], Expectation[S, O, F]) ?=> Unit],
      linked: Expr[ExpectationCapture]
  )(using Quotes): Expr[(S, Step[S, O, F]) => Boolean] =
    val prepared = ExpectationBlock.validate(body)
    '{ transitionExpectation($prepared, $linked) }

@scala.annotation.publicInBinary
private[framework] def sameExpectation[S, O, F](
    prepared: BlockPredicate[Expectation[S, O, F] ?=> Unit],
    linked: ExpectationCapture
): Step[S, O, F] => Boolean =
  new (Step[S, O, F] => Boolean) with PredicateBindings:
    def apply(step: Step[S, O, F]): Boolean = new Expectation(step).evaluate(prepared.body)
    def validate(owner: Model): Unit =
      prepared.members.foreach(_.validate(owner))
      prepared.predicates.foreach(PredicateBindings.validate(_, owner))
    def sourceParameters: Vector[String] = prepared.parameters
    override lazy val captured: Option[Captured[AnyRef]] =
      linked.map((root, chunks) => Reify.captured[AnyRef](this, root, chunks))

@scala.annotation.publicInBinary
private[framework] def transitionExpectation[S, O, F](
    prepared: BlockPredicate[(Before[S], Expectation[S, O, F]) ?=> Unit],
    linked: ExpectationCapture
): (S, Step[S, O, F]) => Boolean =
  new ((S, Step[S, O, F]) => Boolean) with PredicateBindings:
    def apply(state: S, step: Step[S, O, F]): Boolean =
      new Expectation(step).evaluate(prepared.body(using new Before(state)))
    def validate(owner: Model): Unit =
      prepared.members.foreach(_.validate(owner))
      prepared.predicates.foreach(PredicateBindings.validate(_, owner))
    def sourceParameters: Vector[String] = prepared.parameters
    override lazy val captured: Option[Captured[AnyRef]] =
      linked.map((root, chunks) => Reify.captured[AnyRef](this, root, chunks))

// A Property builder with before-state evidence enabled. Core form:
// `builder.transition((b, a) => predicate(b, a))` (private core construction).
final class Across[S, O, F] private[framework] (
    private[framework] val builder: PropertyBuilder[S, O, F]
):
  // Finishes a transition Property. Core form: `builder.transition((b, a) => predicate(b, a))`.
  inline infix def expect(
      @capture("capturedExpect") body: (Before[S], Expectation[S, O, F]) ?=> Unit
  ): TransitionProperty[S, O, F] =
    ${ ExpectationMacros.transitionProperty('builder, 'body) }
  // The captured form of `expect`: the plugin passes the lifted root and its chunks.
  private[framework] inline def capturedExpect(
      body: (Before[S], Expectation[S, O, F]) ?=> Unit,
      root: String,
      chunks: Seq[String]
  ): TransitionProperty[S, O, F] =
    ${ ExpectationMacros.capturedTransitionProperty('builder, 'body, 'root, 'chunks) }

// The same-step expectation view of an ordinary Property builder. Core form:
// `builder.sameStep(s => predicate(s))` (private core construction).
final class Expecting[S, O, F] private[framework] (
    private[framework] val builder: PropertyBuilder[S, O, F]
):
  // Finishes a same-step Property. Core form: `builder.sameStep(s => predicate(s))`.
  inline infix def expect(
      @capture("capturedExpect") body: Expectation[S, O, F] ?=> Unit
  ): SameStepProperty[S, O, F] =
    ${ ExpectationMacros.property('builder, 'body) }
  // The captured form of `expect`: the plugin passes the lifted root and its chunks.
  private[framework] inline def capturedExpect(
      body: Expectation[S, O, F] ?=> Unit,
      root: String,
      chunks: Seq[String]
  ): SameStepProperty[S, O, F] =
    ${ ExpectationMacros.capturedProperty('builder, 'body, 'root, 'chunks) }

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
// Assigning such a value records nothing: an effect records the status as it records any fact,
// `phase = Phase.started` then `record(phase.status)`. Core form: the status fact each step
// writes by hand, `enter(s.copy(phase = started), statusStarted)`.
trait Recorded[F]:
  def status: F

// What an `effect:` block makes of one step while it runs: the state it enters, which each field
// assignment replaces with a copy, by the setters its state type declares, each of one shape:
// `def phase_=(p: Phase)(using d: Draft[State, ?, ?]): Unit = d.set(_.copy(phase = p))`; the facts
// it records; and the outcome it rejects with, if any. Each call of the effect makes a fresh draft,
// which never leaves the block. Core form: the arguments of the step the step function returns,
// `Step(outcome, s.copy(phase = p), List(facts*))`.
@implicitNotFound(
  "this assigns a field of ${S}, records a fact or rejects, which only an `effect:` block does: " +
    "outside one, return the steps, `enter(s.copy(...), fact)` or `reject(outcome, s)`"
)
final class Draft[S, O, F] private[framework] (start: S) extends View[S], Refusing[O], Recording[F]:
  private var current: S = start // scalafix:ok DisableSyntax.var
  private var facts: List[F] = Nil // scalafix:ok DisableSyntax.var
  private var rejection: Option[O] = None // scalafix:ok DisableSyntax.var

  private[framework] def state: S = current

  // The state with one field replaced: `d.set(_.copy(phase = p))`. Core form: `s.copy(phase = p)`.
  def set(update: S => S): Unit = current = update(current)

  // The step the block made, from the state `s` it started in: the rejection with `s`, or the ok
  // outcome with the state assigned and the facts in the order recorded.
  private[framework] def step(s: S, ok: Ok[O]): Step[S, O, F] = rejection match
    case Some(outcome) => Step(outcome, s)
    case None          => Step(ok.outcome, current, facts)

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

// `record(statusPaused)`, in an `effect:` block or an effect of a Machine that declares its state
// as vars: the step records these facts, after those an earlier `record` of the block recorded.
// Core form: the facts of `enter(s.copy(...), facts*)`.
def record[F](first: F, rest: F*)(using recording: Recording[F]): Unit =
  recording.add(first +: rest)

// `reject(Outcome.notFound)`, in an `effect:` block or an effect of a Machine that declares its
// state as vars: the step answers `outcome` and keeps the state it started from, recording
// nothing. Core form: `reject(Outcome.notFound, s)`.
def reject[O](outcome: O)(using refusing: Refusing[O]): Unit =
  refusing.refuse(outcome)

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
      inline member: S => (MS, RecordMachine[MS, MO, MF]),
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
      inline member: S => (MS, RecordMachine[MS, MO, MF]),
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
  // Requires an additional fact only when a condition of the vars holds after the step:
  // `recordsWhen(l1 != None, Event.readL1)`. Core form:
  // `prior(step) && (!condition(step.state) || step.facts.contains(fact))`.
  @targetName("recordsWhenVars")
  def recordsWhen[G](condition: => Boolean, fact: G)(using G <:< F, S <:< State): Witness[S, O, F] =
    claim.recordingWhen(
      s => summon[S <:< State](s).owner.at(s)(condition),
      _.facts.contains(fact)
    )

// Finishes a composition's witness with the fact of its selected member, whose machine fixes the
// fact type, as the composition's declaration selector does. Core form:
// `c.property.when(last).sameStep(step => step.facts.contains("member_fact"))`.
extension [S <: Product](builder: WitnessBuilder[S, String, String])
  // Finishes with an absent fact of the selected member. Core form:
  // `c.property.when(last).sameStep(step => !step.facts.contains("member_fact"))`.
  inline def excludes[MS, O, F](
      inline member: S => (MS, RecordMachine[MS, O, F]),
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
      inline member: S => (MS, RecordMachine[MS, O, F]),
      fact: F
  ): Witness[S, String, String] =
    val key = witnessMemberFact(claim.owner, MemberPath.retainDirect(member), fact)
    claim.requiring(step => !step.facts.contains(key))
  // Requires the selected member's fact under the composed after-state condition. Core form:
  // `prior(step) && (!condition(step.state) || step.facts.contains("member_fact"))`.
  inline def recordsWhen[MS, O, F](
      condition: S => Boolean,
      inline member: S => (MS, RecordMachine[MS, O, F]),
      fact: F
  ): Witness[S, String, String] =
    val key = witnessMemberFact(claim.owner, MemberPath.retainDirect(member), fact)
    claim.recordingWhen(condition, _.facts.contains(key))

@scala.annotation.publicInBinary
private[framework] def witnessMemberFact[S <: Product, MS, O, F](
    owner: Declares[S],
    member: S => (MS, RecordMachine[MS, O, F]),
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

// `property.once(chosen.nonEmpty).keeps(chosen)`, of a Machine that declares its state as vars:
// `over` and the value kept read them, with the state before a step, or after it, restored into
// them. Core form: `b.transition((before, after) => !over(before) || after.state.chosen == before.chosen)`.
extension [O, F](b: PropertyBuilder[State, O, F])
  @sugar @targetName("onceVars")
  def once(over: => Boolean): Once[State, O, F] =
    require(
      b.when.isEmpty,
      s"once declares ${b.name} about every step, so it takes no when: write the restriction into its predicates, or declare the Property with `expect`"
    )
    val vars = Vars.of(b.m)
    Once(b, (s: State) => vars.at(s)(over))

// The value of a Machine's vars a step from a state `over` holds of keeps: `keeps(chosen)`.
// Core form: `b.transition((before, after) => !over(before) || after.state.chosen == before.chosen)`.
extension [O, F](once: Once[State, O, F])
  @sugar @targetName("keepsVars")
  def keeps[V](projection: => V)(using CanEqual[V, V]): Property =
    val vars = Vars.of(once.b.m)
    once.keeps((s: State) => vars.at(s)(projection))

// What `once(over)` leaves to say: the value it keeps. Core form: `b.transition(predicate)`.
@sugar final class Once[S, O, F] private[framework] (
    private[framework] val b: PropertyBuilder[S, O, F],
    over: S => Boolean
):
  // The value a step from a state `over` holds of keeps, a field path such as `_.phase` or
  // `_.order.phase`. Core form:
  // `b.transition((before, after) => !over(before) || after.state.phase == before.phase)`.
  @sugar def keeps[V](projection: S => V)(using CanEqual[V, V]): Property =
    b.transition((before, after) => !over(before) || projection(after.state) == projection(before))

// `never(to)`, a same-step Property, which `from` turns into a transition one. Core form:
// `b.sameStep(after => !to(after))`.
@sugar final class Never[S, O, F] private[framework] (
    b: PropertyBuilder[S, O, F],
    to: Step[S, O, F] => Boolean
) extends Property(b.sameStep(after => !to(after)).decl):
  // No step from a state `before` holds of is one `to` holds of. Core form:
  // `b.transition((before, after) => !origin(before) || !to(after))`.
  @sugar def from(before: S => Boolean): Property =
    val origin = before
    b.transition((before, after) => !origin(before) || !to(after))

// `stays(p)`, a transition Property, which `unless` releases. Core form:
// `b.transition((before, after) => !p(before) || p(after.state))`.
@sugar final class Stays[S, O, F] private[framework] (
    b: PropertyBuilder[S, O, F],
    p: S => Boolean
) extends Property(b.transition((before, after) => !p(before) || p(after.state)).decl):
  // A step `release` holds of may leave `p`. Core form:
  // `b.transition((before, after) => !p(before) || p(after.state) || release(after))`.
  @sugar def unless(release: Step[S, O, F] => Boolean): Property =
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

// A case of an `on` block: where in the state space its action fires, `where(g)`, `always` or
// `where(g).where(h)`, and through `~>` the effect it has there, `where(placed) ~> effects.ship`.
// Core form: the arm `if g(s) then e(s, inputs) else ...` of the action's step function.
final class Case[S, O, F] private[framework] (
    private[framework] val heading: String,
    private[framework] val guard: S => Boolean
):
  // This case where `condition` also holds of the state, beyond its own condition:
  // `where(open).where(_.timer == Timeout.expires)`. Core form: `g(s) && condition(s)`.
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
  // `where(placed) ~> effects.ship`. Core form: the arm `if g(s) then ship(s)` of the action's step
  // function.
  infix def ~>(effect: S => List[Step[S, O, F]])(using
      firing: Firing[S, O, F, ?]
  ): Unit =
    firing.bind(this, stateEffect(effect))

  // The effect of the block's action, of one input, where this case holds, read of the state and
  // the input: `where(open) ~> effects.change`. Core form: the arm `if g(s) then change(s, c)`.
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

// The effects of a Machine that declares its state as vars, where a case of its rules holds: a call
// of an effect that assigns them, which takes one step, the rejection the effect makes or the
// accepted outcome into the vars it assigned. Core form: the arm `if g(s) then e(s, in)` of the
// action's step function.
object Case:
  // A call of an effect of a Machine's vars where the case holds. Core form: the arm
  // `if g(s) then e(s, in)` of the action's step function.
  extension [O, F](arm: Case[State, O, F])
    // `where(!heatingOn) ~> effects.pressButton()`. Core form: the arm `if g(s) then pressButton(s)`.
    @targetName("varEffect")
    infix def ~>(effect: => Unit)(using firing: Firing[State, O, F, ?], ok: Ok[O]): Unit =
      val vars = machineVars(firing)
      firing.bind(arm, stateEffect((s: State) => vars.step[O, F](s, ok.outcome)(effect)))

    // `always ~> effects.send`, an effect of the action's one input. Core form: the arm
    // `if g(s) then send(s, in)`.
    @targetName("varInputEffect")
    infix def ~>[I, A](effect: A => Unit)(using
        firing: Firing[State, O, F, I],
        @unused input: RuleInput[I, A],
        ok: Ok[O]
    ): Unit =
      val vars = machineVars(firing)
      firing.bind(
        arm,
        effectOf(firing.decl, (s: State, value: A) => vars.step[O, F](s, ok.outcome)(effect(value)))
      )

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
    private[framework] val source: Option[RecordMachine[S, O, F]] = None,
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

// The conditions of a Machine that declares its state as vars, which read them with the state
// restored into them. Core form: `if g(s, in) then e(s, in)`.
extension [O, F, I](cases: GuardCases[State, O, F, I])
  // A condition of the vars: `where(!heatingOn)`. Core form: `if condition(s) then e(s, in)`.
  @targetName("varCondition")
  def apply(condition: => Boolean): Case[State, O, F] =
    val vars = machineVars(cases.firing)
    Case("where", s => vars.at(s)(condition))

  // A condition of the vars and the action's input: `where((in: Request) => in.amount <= 2)`.
  // Core form: `if condition(s, in) then e(s, in)`.
  @targetName("varInputCondition")
  def apply[A](using input: RuleInput[I, A])(condition: A => Boolean): Case[State, O, F] =
    val vars = machineVars(cases.firing)
    Case(
      "where",
      cases.firing.inputGuard((s: State, value: A) => vars.at(s)(condition(value)))(using
        input.native
      )
    )

// The vars of the Machine whose rules a firing is in.
private def machineVars(firing: Firing[?, ?, ?, ?]): Vars =
  Vars.of(
    firing.source.getOrElse(
      throw IllegalArgumentException(s"Rule ${firing.action}: no Machine declares its vars here")
    )
  )

// A case that holds in every state: `always ~> effects.keep`. Core form:
// `if true then e(s) else ...`.
def always[S, O, F](using firing: Firing[S, O, F, ?]): Case[S, O, F] =
  val _ = firing
  Case("always", _ => true)

private[framework] def isDirectRuleBook(book: RuleBook[?, ?, ?]): Boolean =
  book.isInstanceOf[RuleTable[?, ?, ?]]

extension (@unused companion: MemberPath.type)
  private[framework] def field(member: Any): Option[String] = MemberPath.retainedField(member)

// When each action of a machine over a state record fires, written as the machine object's `object
// transitions extends Rules`: blocks of an action or action class,
// `on(clerk.ship): where(_.phase == Phase.placed) ~> effects.send`, whose cases each say where the
// action fires, `where(g)` of the state, `where(g).where(h)` of both or `always`, and what it does
// there, `~> effects.x`; and the actions
// no state enables, `disabled(courier.strike)`. A block fires a whole action, `on(buyer.change)`,
// one class of it, `on(buyer.change(Change.hold))`, whose effects then read the state alone, or
// several of them alike, `on(clerk.ship, clerk.cancel)`; an effect that takes arguments beyond the
// state binds them in place, `effects.timeOut(_, Expiry.close)`. Every target names its action
// through the object that declares it, `clerk.ship`, never by a bare name: the IR generator
// refuses a bare target, and a bare `disabled` target, at its line, and a block refuses as the rules
// construct any bare target, including imported names, using its original source spelling. The
// blocks of the actions one object declares may be grouped in
// `from(clerk):` with nested `on(clerk.ship):` blocks, which imports nothing. Where no
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
abstract class Rules[S, O, F](using
    owner: Owner[S, O, F],
    source: FacetSource
) extends RuleTable[S, O, F]:

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

  // Runs one block's cases, refusing a block in a block, and in a `from` an action its declarer
  // does not declare.
  private[framework] def block[I](
      decl: ActionDecl,
      values: Option[List[Any]],
      code: String,
      budgets: Vector[NativeBudget[?]] = Vector.empty
  )(cases: Firing[S, O, F, I] ?=> Unit): Unit =
    openBlock(decl, code):
      val firing =
        Firing[S, O, F, I](
          decl,
          values,
          writtenAction(code),
          (c, effect, captured) =>
            bindRule(decl, values, c.heading, c.guard, effect, code, captured),
          Some(owner.machine),
          facetOrigin,
          budgets
        )
      cases(using firing)

  // The cases of a whole action: `on(clerk.ship): where(placed) ~> effects.send`. Core form: the
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
  // `on(courier.report(Report.delivered)): where(sent) ~> effects.deliver`. Core form: the arms
  // `if g(s) && result == Report.delivered then deliver(s)` of the action's step function.
  inline def on(inline c: Class)(
      cases: Firing[S, O, F, EmptyTuple] ?=> Unit
  ): Unit =
    RuleSource.qualified(c, "on", groupName)
    val target = c
    block[EmptyTuple](target.decl, Some(target.values), RuleSource.spelling(c))(cases)

  // The same cases for two actions, or classes of them, whose effects read the state alone:
  // `on(clerk.hold, clerk.cancel): where(shipped) ~> rejects(Rejection.failedPrecondition)`.
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
      named: List[(Action[?] | Class, String)]
  )(cases: Firing[S, O, F, EmptyTuple] ?=> Unit): Unit =
    for (decl, values, code) <- blockTargets(named) do block[EmptyTuple](decl, values, code)(cases)

  // The blocks of the actions `declarer` declares, each naming its action through its declarer as
  // every block does: `from(clerk):` containing `on(clerk.ship):`. It only groups, and brings none of
  // the declarer's members into scope: a block means in a `from` what it means outside one, and the
  // IR generator refuses an `import clerk.*` in it. Core form: the blocks it holds,
  // `on(clerk.ship): ...`.
  inline def from(declarer: AnyRef)(inline body: Unit): Unit =
    grouped(declarer, RuleSource.imports(body))(body)

  // Actions no state enables, which the machine binds all the same, such as a courier's strike it does
  // not feel. Core form: `action ~> (_ => Nil)`.
  inline def disabled(inline actions: Action[?]*): Unit =
    disable(RuleSource.disabledLabels(actions), actions*)

  private[framework] def capturedDisabledRules(actions: Action[?]*)(data: Seq[String]): Unit =
    val retained =
      RuleBindings.retainedDisabled(owner.machine, actions.map(_.decl), data, facetOrigin)
    disable(retained.map(_._1), actions*)
    actions.zip(retained).foreach((action, row) => capturedDisabledRows(action.decl) = row._2)

  // `phase.in(a, b)` in a condition, as outside the rules. Core form: `List(a, b).contains(phase)`.
  extension [A](value: A)
    @sugar def in(first: A, rest: A*): Boolean = (first +: rest).contains(value)

  // `phase.in[Closed]` in a condition, as outside the rules. Core form:
  // `List(<the cases of Closed, in declaration order>).contains(phase)`.
  extension [A](value: A)
    @sugar def in[R](using role: TypeTest[A, R]): Boolean =
      role.unapply(value).nonEmpty

// The rules of one action a derivation binds in its source's place:
// `rebind(on(clerk.ship): always ~> OrderRecord.effects.send)`, each case `where(g)` or
// `always`, since a derivation's rules name no phase. Its action is `inline`, as a rule block's is
// (`Rules`), so that an overlap names it as written. Core form: the step function
// `action ~> ((s, i) => if g(s) then e(s, i) else Nil)`. For a Machine that declares its state as
// vars, the block is plain control flow whose input is its parameter,
// `rebind(on(client.send): s => Model.effects.sendFromNowhere(s))`.
inline def on[S, O, F, I](using
    owner: Owner[S, O, F]
)(inline a: Action[I])(
    inline cases: Firing[S, O, F, NativeInput[I]] ?=> (I => Unit) | Unit
)(using NotGiven[I =:= EmptyTuple]): RuleGroup[S, O, F] =
  RuleSource.qualified(a, "on", "<declarer>")
  DerivationMacros.derivedInput[S, O, F, I](owner, a.decl, codeOf(a), cases)

// A native scalar action's cases. Core form: `action ~> ((s, in) => if g(s, in) then e(s, in) else Nil)`.
@targetName("onScalarInput")
inline def on[S, O, F, I](using
    owner: Owner[S, O, F]
)(inline a: ScalarAction[I])(
    inline cases: Firing[S, O, F, NativeInput[I]] ?=> (I => Unit) | Unit
): RuleGroup[S, O, F] =
  RuleSource.qualified(a, "on", "<declarer>")
  DerivationMacros.derivedInput[S, O, F, I](owner, a.decl, codeOf(a), cases)

// A state-only action's cases. Core form: `action ~> (s => if g(s) then e(s) else Nil)`. For a
// Machine that declares its state as vars, the block is plain control flow over the vars whose leaves
// call an effect, reject or are `()`, as a block of `transitions` is,
// `rebind(on(tm.decideOnCommit): if Model.states.eagerlyCommits then Model.effects.decideOnCommit())`.
@targetName("onWithoutInput")
inline def on[S, O, F](using
    owner: Owner[S, O, F]
)(inline a: Action[EmptyTuple])(
    inline cases: Firing[S, O, F, EmptyTuple] ?=> Unit
): RuleGroup[S, O, F] =
  RuleSource.qualified(a, "on", "<declarer>")
  DerivationMacros.derived[S, O, F](owner, a.decl, codeOf(a), cases)

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

// When each action of a Machine that declares its state as vars fires, written as the Machine
// object's `object transitions extends Transitions`: blocks of plain control flow over the vars,
// `on(user.pressButton): if !heatingOn then effects.pressButton()`, whose leaves each end the block
// one way, a call of an effect, `reject(r)` or `()`. A block fires where it reaches an effect
// call, answers the rejection where it reaches a `reject`, and is disabled where it reaches `()`,
// as an `if` without an `else` does. A `match` is a phase table, `case a | b`, `case a if g`,
// `case _: Role` and `case _ => ()`, the first matching case winning as in Scala. An input is the
// block's parameter, `on(client.send): s => effects.send(s)`. A leaf never assigns a var, which is
// what effects do: the block says when the action fires and its effect what it does. Blocks name
// their action, a class of it, or two to four of them alike, as the blocks of `Rules` do, and
// `from`, `disabled` and `crashes` mean what they mean there. Core form: one step function per
// action, `action ~> ((s, in) => if g(s) then e(s, in) else Nil)`.
abstract class Transitions[O, F](using
    owner: Owner[State, O, F],
    ok: Ok[O],
    source: FacetSource
) extends RuleTable[State, O, F]:
  // The block of a whole action without inputs. Core form: `action ~> (s => if g(s) then e(s) else Nil)`.
  @targetName("onWithoutInput")
  inline def on(inline a: Action[EmptyTuple])(inline body: Unit): Unit =
    RuleSource.qualified(a, "on", groupName)
    transition(a.decl, None, codeOf(a), TransitionMacros.block(body))

  // The block of a whole action whose input is the block's parameter, `s => effects.send(s)`. Core
  // form: `action ~> ((s, in) => if g(s, in) then e(s, in) else Nil)`.
  inline def on[I](inline a: Action[I])(inline body: I => Unit)(using
      NotGiven[I =:= EmptyTuple]
  ): Unit =
    RuleSource.qualified(a, "on", groupName)
    inputTransition[I](a.decl, codeOf(a), TransitionMacros.input[I](body))

  // The block of one class of an action, which reads the vars alone. Core form: the arm
  // `if g(s) && in == c then e(s) else ...` of the action's step function.
  inline def on(inline c: Class)(inline body: Unit): Unit =
    RuleSource.qualified(c, "on", groupName)
    val target = c
    transition(
      target.decl,
      Some(target.values),
      RuleSource.spelling(c),
      TransitionMacros.block(body)
    )

  // The same block for two actions, or classes of them. Core form: one block per target.
  inline def on(inline a: Action[?] | Class, inline b: Action[?] | Class)(inline body: Unit): Unit =
    RuleSource.qualified(a, "on", groupName)
    RuleSource.qualified(b, "on", groupName, 1)
    each(
      List(a -> RuleSource.spelling(a), b -> RuleSource.spelling(b, 1)),
      TransitionMacros.block(body)
    )

  // The same block for three actions, or classes of them. Core form: one block per target.
  inline def on(
      inline a: Action[?] | Class,
      inline b: Action[?] | Class,
      inline c: Action[?] | Class
  )(inline body: Unit): Unit =
    RuleSource.qualified(a, "on", groupName)
    RuleSource.qualified(b, "on", groupName, 1)
    RuleSource.qualified(c, "on", groupName, 2)
    each(
      List(
        a -> RuleSource.spelling(a),
        b -> RuleSource.spelling(b, 1),
        c -> RuleSource.spelling(c, 2)
      ),
      TransitionMacros.block(body)
    )

  // The same block for four actions, or classes of them. Core form: one block per target.
  inline def on(
      inline a: Action[?] | Class,
      inline b: Action[?] | Class,
      inline c: Action[?] | Class,
      inline d: Action[?] | Class
  )(inline body: Unit): Unit =
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
      ),
      TransitionMacros.block(body)
    )

  // The blocks of the actions `declarer` declares, each naming its action through its declarer.
  // Core form: the blocks it holds, `on(clerk.ship): ...`.
  inline def from(declarer: AnyRef)(inline body: Unit): Unit =
    grouped(declarer, RuleSource.imports(body))(body)

  // Actions no state enables, which the machine binds all the same. Core form: `action ~> (_ => Nil)`.
  inline def disabled(inline actions: Action[?]*): Unit =
    disable(RuleSource.disabledLabels(actions), actions*)

  // A generated crash, which resets the vars its durability does not keep. Core form:
  // `on(action): effects.generatedReset()`.
  inline def crashes(inline action: Action[EmptyTuple], outcome: O, facts: F*): Unit =
    RuleSource.qualified(action, "crashes", groupName)
    registerCrash(
      crashBinding(owner, action, Vector.empty, outcome, facts, NativeLocation.here),
      RuleSource.spelling(action)
    )

  private def each(named: List[(Action[?] | Class, String)], body: () => Unit): Unit =
    for (decl, values, code) <- blockTargets(named) do transition(decl, values, code, body)

  // One rule per block: its steps are the paths of the block's run from a state, and it holds
  // where the run takes a step, which is where it reaches an effect call or a rejection. Running
  // the block to read its guard is safe, as a run only assigns vars and every run restores them.
  @scala.annotation.publicInBinary
  private[framework] def transition(
      decl: ActionDecl,
      values: Option[List[Any]],
      code: String,
      body: () => Unit
  ): Unit =
    register(decl, values, code)(
      new InputEffect[State, O, F]:
        def apply(state: State, values: List[Any]): List[Step[State, O, F]] =
          Vars.of(owner.machine).steps[O, F](state, ok.outcome, fires = false)(body())
    )

  @scala.annotation.publicInBinary
  private[framework] def inputTransition[I](decl: ActionDecl, code: String, body: I => Unit): Unit =
    val shape = decl.nativeInputs.getOrElse(
      throw IllegalArgumentException(
        s"on(${writtenAction(code)}) reads an input it does not declare"
      )
    )
    register(decl, None, code)(
      new InputEffect[State, O, F]:
        def apply(state: State, values: List[Any]): List[Step[State, O, F]] =
          apply(state, Class(decl, values))
        override def apply(state: State, cls: Class): List[Step[State, O, F]] =
          val input = shape.input(cls).asInstanceOf[I] // scalafix:ok DisableSyntax.asInstanceOf
          Vars.of(owner.machine).steps[O, F](state, ok.outcome, fires = false)(body(input))
    )

  private def register(decl: ActionDecl, values: Option[List[Any]], code: String)(
      effect: InputEffect[State, O, F]
  ): Unit =
    val guard = new InputGuard[State]:
      def apply(state: State, cls: Class): Boolean = effect(state, cls).nonEmpty
    openBlock(decl, code):
      bindRule(decl, values, "on", guard, effect, code, None)

// `reject(Rejection.notFound)`, in a transition or an effect of a Machine on the shared outcomes:
// the step answers `rejected(why)` and keeps the state it started from. Core form:
// `reject(Outcome.rejected(Rejection.notFound))`.
def reject(why: outcomes.Rejection)(using
    @implicitNotFound(
      "reject(Rejection) answers the shared framework.dsl.outcomes.Outcome, so it rejects in a " +
        "Machine on the shared outcomes: `import framework.dsl.outcomes.{Outcome, Rejection}`"
    ) refusing: Refusing[outcomes.Outcome]
): Unit = refusing.refuse(outcomes.Outcome.rejected(why))

// `verify(invariants.consistency) in anyVote`, in `queries`: a check that a claim of the Machine
// holds on every path of the scenario, named after its `val`; a claim is a member of `invariants`,
// `postconditions` or `progress`, or a find target declared in `queries`. Every claim is checked by
// default, so a Model writes one only where a realization names it or a scenario other than the
// default matters. Core form: `query verify claim in scenario limits bounds`.
inline def verify(inline claim: Any): ClaimQuery = ${ QueryMacros.verifying('claim) }

// `find(servedFromL2) in anyRun`, in `queries`: a check that some step of the scenario is one the
// claim is about and holds of, a witness, named after its `val`. Its target is a claim, typically a
// postcondition-shaped block declared in `queries` itself, `val servedFromL2 =
// after(client.read): records(Event.readL2)`, which is no promise and is not checked by default.
// Core form: `query find claim in scenario limits bounds`.
inline def find(inline claim: Any): ClaimQuery = ${ QueryMacros.finding('claim) }

// `only(heater.heat, heater.failover)`: the scenario that admits these actions and no others, named
// after its `val` where it is one. Core form: `scenario.free` restricted to the actions listed.
inline def only(actions: (Action[?] | Class | Composed)*): Scenario =
  Scenario.of(QueryMacros.enclosing, NativeLocation.declaredName).only(actions*)

// `fair(rm.commit)`: the scenario that makes these actions weakly fair, an action that stays enabled
// is eventually taken, and admits every action. Core form: `scenario.free` assuming `fair(actions)`.
inline def fair(actions: (Action[?] | Class | Composed)*): Scenario =
  Scenario.of(QueryMacros.enclosing, NativeLocation.declaredName).fair(actions*)

// `starts(temperature = -40)`: the scenario that starts in the Machine's initial state with the
// vars named changed, checked against its vars when it is checked. Core form:
// `scenario.starts(init.copy(temperature = -40)).free`.
inline def starts: Scenario.Starts =
  Scenario.of(QueryMacros.enclosing, NativeLocation.declaredName).starts

// `actions(client.write(), client.read())`: the scenario pinned to exactly these classes, in order.
// Core form: `scenario.actions(classes*)`.
inline def actions(cs: (ClassRef | Composed)*): Scenario =
  Scenario.of(QueryMacros.enclosing, NativeLocation.declaredName).actions(cs*)

// The compile-time half of `verify`, `find` and the scenario words: what claim a check names, and
// the machine a scenario is built in.
@scala.annotation.publicInBinary
private[framework] object QueryMacros:
  import scala.quoted.*

  private val blockSections = Set("invariants$", "postconditions$", "queries$")

  // The Machine object the expansion sits in, if any.
  @scala.annotation.publicInBinary
  private[framework] inline def enclosing: Option[Model] = ${ enclosingImpl }

  @scala.annotation.publicInBinary
  private[framework] def enclosingImpl(using q: Quotes): Expr[Option[Model]] =
    import q.reflect.*
    val base = TypeRepr.of[RecordMachine[?, ?, ?]].typeSymbol
    Iterator
      .iterate(Symbol.spliceOwner)(_.owner)
      .takeWhile(!_.isNoSymbol)
      .find(s => s.isClassDef && !s.equals(base) && s.typeRef.baseClasses.contains(base))
      .fold('{ None })(cls => '{ Some(${ This(cls).asExprOf[Model] }) })

  @scala.annotation.publicInBinary
  private[framework] def verifying(claim: Expr[Any])(using Quotes): Expr[ClaimQuery] =
    select(claim, finds = false)

  @scala.annotation.publicInBinary
  private[framework] def finding(claim: Expr[Any])(using Quotes): Expr[ClaimQuery] =
    select(claim, finds = true)

  @scala.annotation.publicInBinary
  private[framework] def leadsTo(from: Expr[Boolean], to: Expr[Boolean], within: Expr[Int])(using
      q: Quotes
  ): Expr[LeadsTo] =
    import q.reflect.*
    val owners = Iterator.iterate(Symbol.spliceOwner)(_.owner).takeWhile(!_.isNoSymbol).toList
    if !owners.exists(s => s.isClassDef && s.name == "progress$") then
      report.errorAndAbort("a progress claim belongs in progress")
    within.value match
      case Some(n) if n > 0 => ()
      case _                =>
        report.errorAndAbort(
          "within is a positive number of steps, written as a literal",
          within.asTerm.pos
        )
    val named = Expr(
      owners
        .find(s => s.isValDef && !s.flags.is(Flags.Synthetic) && !s.name.contains("$"))
        .fold("")(_.name)
    )
    '{
      LeadsTo.declared(
        $named,
        $enclosingImpl,
        () => $from,
        () => $to,
        $within,
        Scenario.of($enclosingImpl, None)
      )
    }

  private def select(claim: Expr[Any], finds: Boolean)(using q: Quotes): Expr[ClaimQuery] =
    import q.reflect.*
    val word = if finds then "find" else "verify"
    val form = if finds then '{ QueryForm.find } else '{ QueryForm.verify }
    def refused = report.errorAndAbort(
      s"$word takes an invariant, a postcondition or a progress claim",
      claim.asTerm.pos
    )
    def unwrap(t: Term): Term = t match
      case Inlined(_, Nil, inner) => unwrap(inner)
      case Typed(inner, _)        => unwrap(inner)
      case other                  => other
    val term = unwrap(claim.asTerm)
    val tpe = term.tpe.widen
    if tpe <:< TypeRepr.of[Property] then
      '{ ClaimQuery.property($form, ${ term.asExprOf[Property] }) }
    else if tpe <:< TypeRepr.of[LeadsTo] then
      if finds then
        report.errorAndAbort("a progress claim is verified, not found", claim.asTerm.pos)
      '{ ClaimQuery.progress($form, ${ term.asExprOf[LeadsTo] }) }
    else
      val member = term.symbol
      val section = if member.exists then member.owner else Symbol.noSymbol
      if !member.exists || !member.isValDef || !(tpe =:= TypeRepr.of[Unit]) ||
        !section.isClassDef || !section.flags.is(Flags.Module) || !blockSections(section.name)
      then refused
      val machine = section.owner
      if !machine.isClassDef || !machine.flags.is(Flags.Module) then refused
      val owner = Ref(machine.companionModule).asExprOf[Model]
      '{
        ClaimQuery.block(
          $form,
          $owner,
          ${ Expr(section.name.stripSuffix("$")) },
          ${ Expr(member.name) }
        )
      }

// The compile-time half of a derivation's block: in the rule words it binds those rules, and in
// plain control flow over a Machine's vars the one rule a transitions block would.
@scala.annotation.publicInBinary
private[framework] object DerivationMacros:
  import scala.quoted.*

  // A derivation's block of a whole action without inputs: written in the rule words, whose cases
  // read the block's Firing, it binds those rules; written as plain control flow over a Machine's
  // vars, which reads no Firing, it binds the one rule the block's run gives, as `transitions` does.
  private[framework] inline def derived[S, O, F](
      owner: Owner[S, O, F],
      decl: ActionDecl,
      code: String,
      inline cases: Firing[S, O, F, EmptyTuple] ?=> Unit
  ): RuleGroup[S, O, F] = ${ derivedImpl[S, O, F]('owner, 'decl, 'code, 'cases) }

  private def derivedImpl[S: Type, O: Type, F: Type](
      owner: Expr[Owner[S, O, F]],
      decl: Expr[ActionDecl],
      code: Expr[String],
      cases: Expr[Firing[S, O, F, EmptyTuple] ?=> Unit]
  )(using Quotes): Expr[RuleGroup[S, O, F]] =
    import quotes.reflect.*
    def unwrap(t: Term): Term = t match
      case Inlined(_, Nil, inner) => unwrap(inner)
      case Typed(inner, _)        => unwrap(inner)
      case other                  => other
    def reads(tree: Tree, parameter: Symbol): Boolean =
      var found = false
      new TreeTraverser:
        override def traverseTree(t: Tree)(owner: Symbol): Unit =
          if t.symbol.equals(parameter) then found = true
          else super.traverseTree(t)(owner)
      .traverseTree(tree)(Symbol.spliceOwner)
      found
    unwrap(cases.asTerm) match
      case Lambda(List(parameter), body) if !reads(body, parameter.symbol) =>
        if !(TypeRepr.of[S] =:= TypeRepr.of[State]) then
          report.errorAndAbort(
            "a block of plain control flow binds a Machine that declares its state as vars"
          )
        val ok = Expr
          .summon[Ok[O]]
          .getOrElse(
            report.errorAndAbort(
              s"a block of plain control flow takes its accepted outcome from a given Ok[${Type.show[O]}]"
            )
          )
        val run = TransitionMacros.leaves(body.asExprOf[Unit])
        '{
          derivedTransition[O, F](
            $owner.asInstanceOf[Owner[State, O, F]], // scalafix:ok DisableSyntax.asInstanceOf
            $ok,
            $decl,
            None,
            $code,
            () => $run
          ).asInstanceOf[RuleGroup[S, O, F]] // scalafix:ok DisableSyntax.asInstanceOf
        }
      case _ => '{ derivedRules[S, O, F, EmptyTuple]($owner, $decl, $code)($cases) }

  // A derivation's block of an action with an input: written in the rule words, it binds those
  // rules; written as plain control flow whose input is the block's parameter, `s => ...`, it binds
  // the one rule the block's run on the input gives.
  private[framework] inline def derivedInput[S, O, F, I](
      owner: Owner[S, O, F],
      decl: ActionDecl,
      code: String,
      inline cases: Firing[S, O, F, NativeInput[I]] ?=> (I => Unit) | Unit
  ): RuleGroup[S, O, F] = ${ derivedInputImpl[S, O, F, I]('owner, 'decl, 'code, 'cases) }

  private def derivedInputImpl[S: Type, O: Type, F: Type, I: Type](
      owner: Expr[Owner[S, O, F]],
      decl: Expr[ActionDecl],
      code: Expr[String],
      cases: Expr[Firing[S, O, F, NativeInput[I]] ?=> (I => Unit) | Unit]
  )(using Quotes): Expr[RuleGroup[S, O, F]] =
    import quotes.reflect.*
    def unwrap(t: Term): Term = t match
      case Inlined(_, Nil, inner) => unwrap(inner)
      case Typed(inner, _)        => unwrap(inner)
      case Block(Nil, inner)      => unwrap(inner)
      case other                  => other
    unwrap(cases.asTerm) match
      case Lambda(List(_), body) =>
        unwrap(body) match
          case function @ Lambda(List(_), _) =>
            if !(TypeRepr.of[S] =:= TypeRepr.of[State]) then
              report.errorAndAbort(
                "a block of plain control flow binds a Machine that declares its state as vars"
              )
            val ok = Expr
              .summon[Ok[O]]
              .getOrElse(
                report.errorAndAbort(
                  "a block of plain control flow takes its accepted outcome from a given " +
                    s"Ok[${Type.show[O]}]"
                )
              )
            val run = TransitionMacros.inputImpl[I](function.asExprOf[I => Unit])
            '{
              derivedInputTransition[O, F, I](
                $owner.asInstanceOf[Owner[State, O, F]], // scalafix:ok DisableSyntax.asInstanceOf
                $ok,
                $decl,
                $code,
                $run
              ).asInstanceOf[RuleGroup[S, O, F]] // scalafix:ok DisableSyntax.asInstanceOf
            }
          case _ => rules(owner, decl, code, cases)
      case _ => rules(owner, decl, code, cases)

  private def rules[S: Type, O: Type, F: Type, I: Type](
      owner: Expr[Owner[S, O, F]],
      decl: Expr[ActionDecl],
      code: Expr[String],
      cases: Expr[Firing[S, O, F, NativeInput[I]] ?=> (I => Unit) | Unit]
  )(using Quotes): Expr[RuleGroup[S, O, F]] =
    '{
      derivedRules[S, O, F, NativeInput[I]]($owner, $decl, $code): firing ?=>
        val _ = $cases(using firing)
    }

// The one rule a derivation's block of a Machine's vars binds: it holds where the block's run takes
// a step, as a block of `transitions` does.
@scala.annotation.publicInBinary
private[framework] def derivedTransition[O, F](
    owner: Owner[State, O, F],
    ok: Ok[O],
    decl: ActionDecl,
    values: Option[List[Any]],
    code: String,
    body: () => Unit
): RuleGroup[State, O, F] =
  derivedRule(
    owner,
    decl,
    values,
    code,
    new InputEffect[State, O, F]:
      def apply(state: State, values: List[Any]): List[Step[State, O, F]] =
        Vars.of(owner.machine).steps[O, F](state, ok.outcome, fires = false)(body())
  )

@scala.annotation.publicInBinary
private[framework] def derivedInputTransition[O, F, I](
    owner: Owner[State, O, F],
    ok: Ok[O],
    decl: ActionDecl,
    code: String,
    body: I => Unit
): RuleGroup[State, O, F] =
  val shape = decl.nativeInputs.getOrElse(
    throw IllegalArgumentException(s"on(${writtenAction(code)}) reads an input it does not declare")
  )
  derivedRule(
    owner,
    decl,
    None,
    code,
    new InputEffect[State, O, F]:
      def apply(state: State, values: List[Any]): List[Step[State, O, F]] =
        apply(state, Class(decl, values))
      override def apply(state: State, cls: Class): List[Step[State, O, F]] =
        val input = shape.input(cls).asInstanceOf[I] // scalafix:ok DisableSyntax.asInstanceOf
        Vars.of(owner.machine).steps[O, F](state, ok.outcome, fires = false)(body(input))
  )

private def derivedRule[O, F](
    owner: Owner[State, O, F],
    decl: ActionDecl,
    values: Option[List[Any]],
    code: String,
    effect: InputEffect[State, O, F]
): RuleGroup[State, O, F] =
  val action = writtenAction(code)
  require(
    writtenPath(code).contains('.'),
    s"on($action) names its action bare: name it through the object that declares it, " +
      s"`on(<declarer>.$action)`"
  )
  val _ = owner
  val guard = new InputGuard[State]:
    def apply(state: State, cls: Class): Boolean = effect(state, cls).nonEmpty
  val rule = Rule[State, O, F](1, "on", action, decl, values, guard, effect)
  rule.publishCapture(None)
  val group = RuleGroup(Vector(rule))
  group.publishOrigin(None)
  group

// A plain definition expanded from its body where a lifted function calls it. Core form: its `body`.
final class sugar extends scala.annotation.StaticAnnotation
