// Sugar: definitions whose meaning a core declaration already expresses, kept for readability. Each
// names the core form it stands for. Inline definitions expand through the Scala compiler;
// declaration witnesses, rules and blocks still await constructed-value export. Among them are the
// block forms a machine object's section `val`s are written in, `is:` for a predicate of the
// state, `expect:` for requirements on its resulting step, and `effect:` for a step function, with `record` and
// `reject(outcome)` statements, the `View` and `Draft` the field accessors of the state read and
// assign through, and `Recorded`, the status fact a phase enum's case declares.
package framework.dsl

import scala.annotation.{implicitNotFound, targetName, unused}
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

// What `after` is in a postcondition of a Machine (ClaimWords): the block about the steps of
// `target`, `after(heater.failover): check(beeping)`, which must hold an assertion. Core form:
// `!(a.action == target) || requirements`.
final class StepTarget:
  inline def apply(target: Action[?] | Class | CompositionTarget)(inline block: Unit): Unit =
    ${ ClaimMacros.targeted('target, 'block) }

// One assertion of a claim, reported on its own where it fails: `val neverBeeps = check(!beeping)` in
// `invariants`, or in a block of the retired `expect`, a requirement whose failure skips the rest. It
// records nothing outside an evaluation, and is refused in `progress`. Core form:
// `condition && remainingRequirements`.
inline def check(inline condition: Boolean): Unit = ${ ClaimMacros.check('condition) }

// A safety claim with its own environment. Core form:
// `ScopedClaim(machine, section, member, scenario)(block)`.
extension (inline block: Unit)
  inline infix def in(inline scenario: Scenario): ScopedClaim =
    ${ ClaimMacros.scoped('block, 'scenario) }

  // Refuses a non-scenario environment. Core form: `scala.compiletime.error(...)`.
  inline infix def in(@scala.annotation.unused other: Any): Nothing =
    scala.compiletime.error("in takes a scenario: only(...), fair(...), starts(...) or a named one")

// What `records` is in a claim or an effect of a Machine whose facts are `F` (ClaimWords): in a
// claim the assertion that the step recorded the fact, in an effect a fact the step records.
// Core form: `a.facts.contains(fact)`, or the facts of `Step(outcome, state, facts)`.
final class StepRecords[F]:
  inline def apply(inline fact: F): Unit = ${ ClaimMacros.fact('fact) }

// The outcome the step answered, in a postcondition or a monitor, `if outcome != Outcome.accepted then
// check(balances == before.balances)`, typed with the outcomes of the Machine it is in. Core form:
// `a.outcome` in `(b, a) => predicate(b, a)`.
transparent inline def outcome: Any = ${ ClaimMacros.outcome }

// What `oneOf` is in a claim: its branches, of which there are at least two. Core form:
// `firstBranch || secondBranch || rest`.
final class ClaimAlternatives:
  inline def apply(inline body: Branching ?=> Unit): Unit = ${ ClaimMacros.alternatives('body) }

// The alternatives of a var's initial value. Core form: `Machine.registerInitial(name, domain)`.
final class InitialAlternatives:
  inline def apply[A](domain: Iterable[A]): A = ${ StartStatesMacro.initialize('domain) }

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

// What `branch` is in a claim's `oneOf`. Core form: `(first && second)`.
final class ClaimBranch:
  inline def apply(inline body: Unit): Unit = ${ ClaimMacros.branchBlock('body) }

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
    val base = TypeRepr.of[NativeModel[?, ?, ?]].typeSymbol
    owners.find(s => s.isClassDef && !s.equals(base) && s.typeRef.baseClasses.contains(base))

  // The outcome (0) or fact (1) type of the Machine the expansion sits in.
  private def machineType(using q: Quotes)(index: Int, word: String): q.reflect.TypeRepr =
    import q.reflect.*
    val base = TypeRepr.of[NativeModel[?, ?, ?]].typeSymbol
    machine
      .map(_.typeRef.baseType(base))
      .collect { case AppliedType(_, List(_, o, f)) => if index == 0 then o else f }
      .getOrElse(report.errorAndAbort(s"$word reads a step of a Machine: write it in one"))

  private def machineRef(using q: Quotes)(cls: q.reflect.Symbol): Expr[NativeModel[?, ?, ?]] =
    import q.reflect.*
    val ref = if cls.flags.is(Flags.Module) then Ref(cls.companionModule) else This(cls)
    ref.asExprOf[NativeModel[?, ?, ?]]

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
  private[framework] def scoped(block: Expr[Unit], scenario: Expr[Scenario])(using
      q: Quotes
  ): Expr[ScopedClaim] =
    import q.reflect.*
    val section = owners
      .find(s => s.isClassDef && claimSections(s.name))
      .getOrElse(
        report.errorAndAbort(
          "an invariant or postcondition belongs in invariants or postconditions"
        )
      )
    if !asserts(block.asTerm) then report.errorAndAbort(emptyBlock, block.asTerm.pos)
    val owner = machineRef(
      machine.getOrElse(
        report.errorAndAbort("a claim belongs in a Machine")
      )
    )
    '{ ScopedClaim($owner, ${ Expr(section.name.stripSuffix("$")) }, $member, $scenario)($block) }

  @scala.annotation.publicInBinary
  private[framework] def check(condition: Expr[Boolean])(using q: Quotes): Expr[Unit] =
    import q.reflect.*
    if inside(Set("progress$", "properties$", "safety$", "liveness$")) then
      report.errorAndAbort("an invariant or postcondition belongs in invariants or postconditions")
    val text = Expr(condition.asTerm.pos.sourceCode.getOrElse(condition.show))
    '{ NativeClaims.assertion($condition, $member, $line, $text) }

  @scala.annotation.publicInBinary
  private[framework] def after(using q: Quotes): Expr[Any] =
    import q.reflect.*
    if !inside(Set("postconditions$", "queries$")) then
      report.errorAndAbort("after(action) is a postcondition: write it in postconditions")
    '{ new StepTarget() }

  @scala.annotation.publicInBinary
  private[framework] def targeted(
      target: Expr[Action[?] | Class | CompositionTarget],
      block: Expr[Unit]
  )(using
      q: Quotes
  ): Expr[Unit] =
    import q.reflect.*
    if !asserts(block.asTerm) then report.errorAndAbort(emptyBlock, block.asTerm.pos)
    val position = target.asTerm.pos
    val location = Expr(s"${position.sourceFile.name}:${position.startLine + 1}")
    '{ NativeClaims.targeting($target, $member, $location)($block) }

  @scala.annotation.publicInBinary
  private[framework] def before(using q: Quotes): Expr[Any] =
    import q.reflect.*
    if !inside(stepSections) then
      report.errorAndAbort("before is the state a step started from; use it in a postcondition")
    val cls = machine.getOrElse(
      report.errorAndAbort("before is the state a step of a Machine started from")
    )
    val record = if CompositionMacros.enclosing.nonEmpty then CompositionMacros.stateType(cls)
    else
      VarSchemaMacro
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
    machineType(1, "records").asType match
      case '[f] => '{ new StepRecords[f]() }

  @scala.annotation.publicInBinary
  private[framework] def fact[F: Type](value: Expr[F])(using q: Quotes): Expr[Unit] =
    val claim = Expr(inside(claimSections ++ Set("monitors$")))
    '{ NativeClaims.recordFact($value, $member, $line, $claim) }

  @scala.annotation.publicInBinary
  private[framework] def oneOf(using q: Quotes): Expr[Any] =
    if StartStatesMacro.initializer.nonEmpty then '{ new InitialAlternatives() }
    else '{ new ClaimAlternatives() }

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
    if Expr.summon[Branching].isEmpty then
      report.errorAndAbort("branch is an alternative of oneOf: write it inside oneOf")
    '{ new ClaimBranch() }

  @scala.annotation.publicInBinary
  private[framework] def branchBlock(body: Expr[Unit])(using q: Quotes): Expr[Unit] =
    import q.reflect.*
    if !asserts(body.asTerm) then report.errorAndAbort(emptyBlock, body.asTerm.pos)
    '{ NativeClaims.alternative($body) }

// The status fact a phase enum's case is recorded as, declared on each case through the enum's
// parameter, which has no default, so a case without one does not compile:
// `enum Phase(val status: Fact) extends Recorded[Fact]`, `case started extends Phase(statusStarted)`.
// Assigning such a value records nothing: an effect records the status as it records any fact,
// `phase = Phase.started` then `record(phase.status)`. Core form: the status fact each step
// writes by hand, `enter(s.copy(phase = started), statusStarted)`.
trait Recorded[F]:
  def status: F

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
extension [A](value: A) inline def in(first: A, rest: A*): Boolean = (first +: rest).contains(value)

// `phase.in[Closed]`: whether the phase has the role `Closed` (model/framework/dsl/Roles.scala), the roles
// its case declares and the broader ones they extend. Written dotted, never infix. Core form:
// `List(<the cases of Closed, in declaration order>).contains(phase)`.
extension [A](value: A)
  inline def in[R](using role: TypeTest[A, R]): Boolean = role.unapply(value).nonEmpty

// `a implies b`: `b` holds wherever `a` does. `b` is read only where `a` holds, so a hole it reaches
// is not reached where `a` is false. Core form: `!a || b`.
extension (a: Boolean) inline infix def implies(b: => Boolean): Boolean = !a || b

// `after.records(fact)`: whether the step records the fact, which is of the step's own fact type:
// a step's facts are covariant, so the evidence keeps a fact of another type from widening it.
// Core form: `after.facts.contains(fact)`.
extension [S, O, F](step: Step[S, O, F])
  // A fact of the step's own type. Core form: `step.facts.contains(fact)`.
  inline def records[G](fact: G)(using G <:< F): Boolean = step.facts.contains(fact)

// `m.witness(a, b).records(fact)`: one path-and-outcome declaration, implicitly on the enclosing
// machine or composition. Its default search budget is 512, with depth and schedule length equal
// to the path length. Core form:
// `query find m.property.when(b).sameStep(_.facts.contains(fact)) in m.scenario.actions(a, b) limits bounds`.
extension [S](owner: Declares[S])
  def witness(
      first: ClassRef | CompositionTarget,
      rest: (ClassRef | CompositionTarget)*
  ): WitnessBuilder[S, owner.Outcome, owner.Fact] =
    WitnessBuilder(owner, "", first +: rest.toVector)

  // A witness named explicitly, such as one in a list. Core form:
  // `query("done") find m.property("done").when(b).sameStep(_.facts.contains(fact)) in m.scenario("done").actions(a, b) limits bounds`.
  def witness(name: String)(
      first: ClassRef | CompositionTarget,
      rest: (ClassRef | CompositionTarget)*
  ): WitnessBuilder[S, owner.Outcome, owner.Fact] =
    WitnessBuilder(owner, name, first +: rest.toVector)

// A witness on the enclosing machine or composition, named after its val. Core form:
// `query find property.when(last).sameStep(_.facts.contains(fact)) in scenario.actions(path*) limits bounds`.
transparent inline def witness(
    first: ClassRef | CompositionTarget,
    rest: (ClassRef | CompositionTarget)*
): Any = ${ QueryMacros.witness('first, 'rest, '{ "" }) }

// A witness on the enclosing owner with an explicit name. Core form:
// `query(name) find property(name).when(last).sameStep(_.facts.contains(fact)) in scenario(name).actions(path*) limits bounds`.
transparent inline def witness(name: String)(
    first: ClassRef | CompositionTarget,
    rest: (ClassRef | CompositionTarget)*
): Any = ${ QueryMacros.witness('first, 'rest, 'name) }

// Finishes a witness with a fact of its declaring machine. Core form:
// `m.property.when(last).sameStep(step => step.facts.contains(fact))`, joined to the pinned Scenario
// by a find Query.
final class WitnessFacts[S, O, F] private[framework] (
    private[framework] val builder: WitnessBuilder[S, O, F]
):
  // A primary fact of the witness's machine. Core form: `builder.recording(_.facts.contains(fact))`.
  def records(fact: F): Witness[S, O, F] =
    builder.recording(_.facts.contains(fact))

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
def sticky[S, O, F](
    promise: Step[S, O, F] => Boolean
): Monitor[S, O, F, Boolean] =
  monitor[S, O, F, Boolean](false)(Programs.Sticky(Some(promise), None))(identity)

// `val ownerAcknowledgment = stickyAcross(ackOnlyWhenKept)`: `sticky` of a promise about the state
// before a step and the step. Core form:
// `monitor[S, O, F, Boolean](false)((broken, before, after) => broken || !promise(before, after))(broken => broken)`.
def stickyAcross[S, O, F](
    promise: (S, Step[S, O, F]) => Boolean
): Monitor[S, O, F, Boolean] =
  monitor[S, O, F, Boolean](false)(Programs.Sticky(None, Some(promise)))(identity)

// Whether a retained rule book is the machine's direct native table.
private[framework] def isDirectRuleBook(book: RuleBook[?, ?, ?]): Boolean =
  book.isInstanceOf[RuleTable[?, ?, ?]]

// A native replacement block. Core form: `derivedTransition(owner, ok, action.decl, None, code, body)`.
inline def on[O, F](using
    owner: MachineDeclaration[State, O, F],
    ok: Ok[O]
)(
    inline action: Action[EmptyTuple]
)(inline body: Unit): RuleGroup[State, O, F] =
  RuleSource.qualified(action, "on", "<declarer>")
  derivedTransition(owner, ok, action.decl, None, codeOf(action), TransitionMacros.block(body))

// A native input replacement. Core form: `derivedInputTransition(owner, ok, action.decl, code, body)`.
inline def on[O, F, I](using
    owner: MachineDeclaration[State, O, F],
    ok: Ok[O]
)(
    inline action: Action[I]
)(inline body: I => Unit)(using NotGiven[I =:= EmptyTuple]): RuleGroup[State, O, F] =
  RuleSource.qualified(action, "on", "<declarer>")
  derivedInputTransition(owner, ok, action.decl, codeOf(action), TransitionMacros.input(body))

// A budgeted native replacement. Core form: `derivedRule(owner, target.action.decl, None, code, program)`.
@targetName("onBudgetedWithoutInput")
inline def on[O, F](using
    owner: MachineDeclaration[State, O, F],
    ok: Ok[O]
)(
    inline action: BudgetedAction[State, EmptyTuple]
)(inline body: Unit): RuleGroup[State, O, F] =
  RuleSource.qualified(action, "on", "<declarer>")
  val target = action
  derivedRule(
    owner,
    target.action.decl,
    None,
    RuleSource.spelling(action),
    Programs.Budgeted(
      Programs.Run[O, F](owner, ok.outcome, TransitionMacros.block(body)),
      target.budgets
    )
  )

// A budgeted input replacement. Core form: `derivedInputTransition(owner, ok, target.action.decl, code, body, target.budgets)`.
@targetName("onBudgetedWithInput")
inline def on[O, F, I](using
    owner: MachineDeclaration[State, O, F],
    ok: Ok[O]
)(
    inline action: BudgetedAction[State, I]
)(inline body: I => Unit)(using NotGiven[I =:= EmptyTuple]): RuleGroup[State, O, F] =
  RuleSource.qualified(action, "on", "<declarer>")
  val target = action
  derivedInputTransition(
    owner,
    ok,
    target.action.decl,
    RuleSource.spelling(action),
    TransitionMacros.input(body),
    target.budgets
  )

// When each action of a Machine that declares its state as vars fires, written as the Machine
// object's `object transitions extends Transitions`: blocks of plain control flow over the vars,
// `on(user.pressButton): if !heatingOn then effects.pressButton()`, whose leaves each end the block
// one way, a call of an effect, `reject(r)` or `()`. A block fires where it reaches an effect
// call, answers the rejection where it reaches a `reject`, and is disabled where it reaches `()`,
// as an `if` without an `else` does. A `match` is a phase table, `case a | b`, `case a if g`,
// `case _: Role` and `case _ => ()`, the first matching case winning as in Scala. An input is the
// block's parameter, `on(client.send): s => effects.send(s)`. A leaf never assigns a var, which is
// what effects do: the block says when the action fires and its effect what it does. Blocks name
// their action, a class of it, or two to four of them alike. `from`, `disabled` and `crashes`
// bind native rules too. Core form: `RuleTable[State, O, F]` with native Programs callbacks.
abstract class Transitions[O, F](using
    owner: MachineDeclaration[State, O, F],
    ok: Ok[O],
    source: FacetSource
) extends RuleTable[State, O, F]:
  // The block of a whole action without inputs.
  @targetName("onWithoutInput")
  inline def on(inline a: Action[EmptyTuple])(inline body: Unit): Unit =
    RuleSource.qualified(a, "on", groupName)
    transition(a.decl, None, codeOf(a), TransitionMacros.block(body))

  @targetName("onBudgetedWithoutInput")
  inline def on(inline a: BudgetedAction[State, EmptyTuple])(inline body: Unit): Unit =
    RuleSource.qualified(a, "on", groupName)
    val target = a
    budgetedTransition(
      target.action.decl,
      RuleSource.spelling(a),
      TransitionMacros.block(body),
      target.budgets
    )

  @scala.annotation.publicInBinary
  private[framework] def budgetedTransition(
      decl: ActionDecl,
      code: String,
      body: () => Unit,
      budgets: Vector[NativeBudget[?]]
  ): Unit =
    register(decl, None, code)(
      Programs.Budgeted(Programs.Run[O, F](owner, ok.outcome, body), budgets)
    )

  // The block of a whole action whose input is the block's parameter, `s => effects.send(s)`.
  inline def on[I](inline a: Action[I])(inline body: I => Unit)(using
      NotGiven[I =:= EmptyTuple]
  ): Unit =
    RuleSource.qualified(a, "on", groupName)
    inputTransition[I](a.decl, codeOf(a), TransitionMacros.input[I](body))

  @targetName("onBudgetedWithInput")
  inline def on[I](inline a: BudgetedAction[State, I])(inline body: I => Unit)(using
      NotGiven[I =:= EmptyTuple]
  ): Unit =
    RuleSource.qualified(a, "on", groupName)
    val target = a
    inputTransition[I](
      target.action.decl,
      RuleSource.spelling(a),
      TransitionMacros.input[I](body),
      target.budgets
    )

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

  // Actions no state enables, which the machine binds all the same.
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

  inline def crashes(
      inline action: BudgetedAction[State, EmptyTuple],
      outcome: O,
      facts: F*
  ): Unit =
    RuleSource.qualified(action, "crashes", groupName)
    val target = action
    registerCrash(
      crashBinding(owner, target.action, target.budgets, outcome, facts, NativeLocation.here),
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
    register(decl, values, code)(Programs.Run[O, F](owner, ok.outcome, body))

  @scala.annotation.publicInBinary
  private[framework] def inputTransition[I](
      decl: ActionDecl,
      code: String,
      body: I => Unit,
      budgets: Vector[NativeBudget[?]] = Vector.empty
  ): Unit =
    val shape = decl.nativeInputs.getOrElse(
      throw IllegalArgumentException(
        s"on(${writtenAction(code)}) reads an input it does not declare"
      )
    )
    val effect = Programs.InputRun[O, F, I](decl, shape, owner, ok.outcome, body)
    register(decl, None, code)(if budgets.isEmpty then effect
    else Programs.Budgeted(effect, budgets))

  private def register(decl: ActionDecl, values: Option[List[Any]], code: String)(
      effect: InputEffect[State, O, F]
  ): Unit =
    openBlock(decl, code):
      bindRule(decl, values, "on", Programs.Fires(effect), effect, code, None)

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
inline def only(actions: (Action[?] | Class | CompositionTarget)*): Scenario =
  Scenario.of(QueryMacros.enclosing, NativeLocation.declaredName).only(actions*)

// `without(user.depressButton)`: the scenario that admits every action but these, named after its
// `val` where it is one. Core form: `scenario.free` restricted to the actions not listed.
inline def without(actions: (Action[?] | Class | CompositionTarget)*): Scenario =
  Scenario.of(QueryMacros.enclosing, NativeLocation.declaredName).without(actions*)

// `fair(rm.commit)`: the scenario that makes these actions weakly fair, an action that stays enabled
// is eventually taken, and admits every action. Core form: `scenario.free` assuming `fair(actions)`.
inline def fair(actions: (Action[?] | Class | CompositionTarget)*): Scenario =
  Scenario.of(QueryMacros.enclosing, NativeLocation.declaredName).fair(actions*)

// `starts(temperature = -40)`: the scenario that starts in the Machine's initial state with the
// vars named changed, checked against its vars when it is checked. Core form:
// `scenario.starts(init.copy(temperature = -40)).free`.
inline def starts: Scenario.Starts =
  Scenario.of(QueryMacros.enclosing, NativeLocation.declaredName).starts

// `actions(client.write(), client.read())`: the scenario pinned to exactly these classes, in order.
// Core form: `scenario.actions(classes*)`.
inline def actions(cs: (ClassRef | CompositionTarget)*): Scenario =
  Scenario.of(QueryMacros.enclosing, NativeLocation.declaredName).actions(cs*)

// The compile-time half of `verify`, `find` and the scenario words: what claim a check names, and
// the machine a scenario is built in.
@scala.annotation.publicInBinary
private[framework] object QueryMacros:
  import scala.quoted.*

  @scala.annotation.publicInBinary
  private[framework] def witness(
      first: Expr[ClassRef | CompositionTarget],
      rest: Expr[Seq[ClassRef | CompositionTarget]],
      name: Expr[String]
  )(using q: Quotes): Expr[Any] =
    import q.reflect.*
    val base = TypeRepr.of[NativeModel[?, ?, ?]].typeSymbol
    val cls = Iterator
      .iterate(Symbol.spliceOwner)(_.owner)
      .takeWhile(!_.isNoSymbol)
      .find(s => s.isClassDef && !s.equals(base) && s.typeRef.baseClasses.contains(base))
      .getOrElse(
        report.errorAndAbort(
          "witness belongs in a Machine or Composition; elsewhere use owner.witness"
        )
      )
    cls.typeRef.baseType(base).typeArgs.map(_.asType) match
      case List('[s], '[o], '[f]) =>
        val owner = This(cls).asExprOf[Declares[s]]
        '{ WitnessBuilder.declared[s, o, f]($owner, $name, $first +: $rest.toVector) }
      case _ => report.errorAndAbort("witness reads its owner's state, outcome and fact types")

  private val blockSections = Set("invariants$", "postconditions$", "queries$")

  // The Machine object the expansion sits in, if any.
  @scala.annotation.publicInBinary
  private[framework] inline def enclosing: Option[Model] = ${ enclosingImpl }

  @scala.annotation.publicInBinary
  private[framework] def enclosingImpl(using q: Quotes): Expr[Option[Model]] =
    import q.reflect.*
    val base = TypeRepr.of[NativeModel[?, ?, ?]].typeSymbol
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
    val named = progressClaim("a progress claim")
    val steps = bound(within)
    '{
      LeadsTo.declared(
        $named,
        $enclosingImpl,
        () => $from,
        () => $to,
        Some($steps),
        Scenario.of($enclosingImpl, None)
      )
    }

  @scala.annotation.publicInBinary
  private[framework] def whenever(from: Expr[Boolean], deadline: Expr[Whenever ?=> Deadline])(using
      q: Quotes
  ): Expr[LeadsTo] =
    val named = progressClaim("whenever, a progress claim,")
    '{
      val promised = $deadline(using Whenever.block)
      LeadsTo.declared(
        $named,
        $enclosingImpl,
        () => $from,
        promised.to,
        promised.within,
        Scenario.of($enclosingImpl, None)
      )
    }

  @scala.annotation.publicInBinary
  private[framework] def eventually(to: Expr[Boolean], within: Expr[Int])(using
      q: Quotes
  ): Expr[Deadline] =
    val steps = bound(within)
    '{ Deadline.declared(() => $to, Some($steps)) }

  // The name of the progress claim being declared, refused outside `progress`.
  private def progressClaim(what: String)(using q: Quotes): Expr[String] =
    import q.reflect.*
    val owners = Iterator.iterate(Symbol.spliceOwner)(_.owner).takeWhile(!_.isNoSymbol).toList
    if !owners.exists(s => s.isClassDef && s.name == "progress$") then
      report.errorAndAbort(s"$what belongs in progress")
    Expr(
      owners
        .find(s => s.isValDef && !s.flags.is(Flags.Synthetic) && !s.name.contains("$"))
        .fold("")(_.name)
    )

  private def bound(within: Expr[Int])(using q: Quotes): Expr[Int] =
    import q.reflect.*
    def unwrap(term: Term): Term = term match
      case Inlined(_, Nil, inner) => unwrap(inner)
      case Typed(inner, _)        => unwrap(inner)
      case other                  => other
    val classes = Iterator
      .iterate(Symbol.spliceOwner)(_.owner)
      .takeWhile(!_.isNoSymbol)
      .filter(s => s.isClassDef && !s.flags.is(Flags.Module))
      .toList
    def constructorParameter(term: Term): Boolean =
      val member = term.symbol
      classes.exists: cls =>
        val field = cls.fieldMember(member.name)
        val immutable = !member.flags.is(Flags.Mutable) &&
          (!field.exists || !field.flags.is(Flags.Mutable))
        val parameter =
          member.flags.is(Flags.ParamAccessor) && member.owner.equals(cls) ||
            member.flags.is(Flags.Param) && member.owner.equals(cls.primaryConstructor)
        val own = term match
          case Select(receiver: This, _) => receiver.symbol.equals(cls)
          case _: Ident                  => true
          case _                         => false
        immutable && parameter && own
    val constructorSize = unwrap(within.asTerm) match
      case Select(receiver, "size") =>
        val parameter = unwrap(receiver)
        parameter.tpe.widen <:< TypeRepr.of[scala.collection.immutable.Set[?]] &&
        constructorParameter(parameter)
      case _ => false
    within.value match
      case Some(n) if n > 0        => within
      case None if constructorSize =>
        '{
          val steps = $within
          require(steps > 0, "within is a positive number of steps")
          steps
        }
      case _ =>
        report.errorAndAbort(
          "within is a positive number of steps, written as a literal or an immutable constructor Set parameter's size",
          within.asTerm.pos
        )

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
    def selectedOwner(machine: Symbol): Option[Expr[Model]] =
      if !machine.isClassDef || !(machine.typeRef <:< TypeRepr.of[Model]) then None
      else if machine.flags.is(Flags.Module) then Some(Ref(machine.companionModule).asExprOf[Model])
      else
        term match
          case Select(section, _) =>
            unwrap(section) match
              case Select(receiver, _) if receiver.tpe <:< TypeRepr.of[Model] =>
                Some(receiver.asExprOf[Model])
              case _ => None
          case _ => None
    if tpe <:< TypeRepr.of[Property] then
      val member = term.symbol
      val section = if member.exists then member.owner else Symbol.noSymbol
      val machine = if section.exists then section.owner else Symbol.noSymbol
      val withinSection = Iterator
        .iterate(Symbol.spliceOwner)(_.owner)
        .takeWhile(!_.isNoSymbol)
        .exists(_.equals(section))
      val selected =
        if !withinSection && member.exists && member.isValDef && section.isClassDef &&
          section.flags.is(Flags.Module) && blockSections(section.name) &&
          selectedOwner(machine).nonEmpty
        then
          val owner = selectedOwner(machine).get
          '{
            NativeClaims.property(
              $owner,
              ${ Expr(section.name.stripSuffix("$")) },
              ${ Expr(member.name) }
            )
          }
        else term.asExprOf[Property]
      '{ ClaimQuery.property($form, $selected) }
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
      val owner = selectedOwner(machine).getOrElse(refused)
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
// The one rule a derivation's block of a Machine's vars binds: it holds where the block's run takes
// a step, as a block of `transitions` does.
@scala.annotation.publicInBinary
private[framework] def derivedTransition[O, F](
    owner: MachineDeclaration[State, O, F],
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
    Programs.Run[O, F](owner, ok.outcome, body)
  )

@scala.annotation.publicInBinary
private[framework] def derivedInputTransition[O, F, I](
    owner: MachineDeclaration[State, O, F],
    ok: Ok[O],
    decl: ActionDecl,
    code: String,
    body: I => Unit,
    budgets: Vector[NativeBudget[?]] = Vector.empty
): RuleGroup[State, O, F] =
  val shape = decl.nativeInputs.getOrElse(
    throw IllegalArgumentException(s"on(${writtenAction(code)}) reads an input it does not declare")
  )
  val effect = Programs.InputRun[O, F, I](decl, shape, owner, ok.outcome, body)
  derivedRule(
    owner,
    decl,
    None,
    code,
    if budgets.isEmpty then effect else Programs.Budgeted(effect, budgets)
  )

@scala.annotation.publicInBinary
private[framework] def derivedRule[O, F](
    owner: MachineDeclaration[State, O, F],
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
  val rule = Rule[State, O, F](1, "on", action, decl, values, Programs.Fires(effect), effect)
  rule.publishCapture(None)
  val group = RuleGroup(Vector(rule))
  group.publishOrigin(None)
  group
