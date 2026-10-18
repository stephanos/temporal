package framework.dsl

import scala.annotation.{publicInBinary, unused}

// One result of an action: the outcome, the next state and the facts it records. `because` is an
// optional explanation a generated table view shows beside the row; no fingerprint reads it.
// Covariant in its facts, so a step that records none, such as `stay(s)` or `reject(o, s)`, is
// a step of every machine of its states and outcomes, and its effect needs no stated
// result type.
final case class Step[S, O, +F](outcome: O, state: S, facts: List[F] = Nil, because: String = "")

// `steps.because("...")`: the explanation of each step written here, as `Step`'s `because` gives it.
// The lifter reads it only on a list of steps the step function writes out.
extension [S, O, F](steps: List[Step[S, O, F]])
  def because(reason: String): List[Step[S, O, F]] = steps.map(_.copy(because = reason))

// The name of one alternative of a step that can go more than one way, declared once by the val that
// names it, `val committed = choice`; the lifter names it after that val. A token is compared by
// identity, so two declarations are two names.
final class Choice private[framework] ():
  // `smashed -> { light = Lit.broken }`, an alternative of an effect of a Machine's vars: the block
  // runs only on the path that takes it.
  def ->(block: => Unit): Alternative = Alternative(this, () => block)

// One alternative of a `choose` in an effect of a Machine's vars: its token and its block.
final class Alternative private[framework] (
    private[framework] val token: Choice,
    private[framework] val run: () => Unit
)

// A new choice token: `val committed = choice`.
def choice: Choice = Choice()

// `choose(smashed -> { light = Lit.broken }, toggled -> { toggle() })`, in an effect of a Machine
// that declares its state as vars: the effect goes one way per alternative, in the order written,
// each a block of its own. The step it is in runs once per alternative, each run from the state the
// step started in and taking the alternatives it picked before, so each alternative is one step.
def choose(first: Alternative, second: Alternative, rest: Alternative*): Unit =
  NativeRuntime.installed.choose((first +: second +: rest).toVector)

private[framework] def choiceTokens(effect: AnyRef): Option[Vector[Choice]] = effect match
  case program: BudgetMetadata => program.nativeChoices
  case _                       => None

// A declared, derived or composed machine.
trait Model:
  def name: String
  def apply(target: Action[?] | Class | CompositionTarget): MemberStep =
    new MemberStep(this, target)

// The name an object form takes from its object: `object OrderProduct` is `orderProduct`.
private[framework] def objectName(of: AnyRef): String =
  objectName(of.getClass)

private[framework] def objectName(of: java.lang.Class[?]): String =
  val n = of.getSimpleName.stripSuffix("$")
  n.take(1).toLowerCase + n.drop(1)

// What a claim is declared on, by its state type `S`: a machine or a composition. A claim written once
// over `Declares[S]`, such as a function over `m: Declares[S]` and the predicates it reads, declares
// the same Property on either. `Outcome` and `Fact` are the types a step of it answers and records:
// the machine's own, or for a composition the strings of its composed keys.
trait Declares[S] extends Model:
  type Outcome
  type Fact

  private[framework] def illustrationMachine: Option[MachineDeclaration[S, Outcome, Fact]] = None

  // The vars its state is declared as, for a Machine and a derivation of one: what its effects,
  // guards and claims are evaluated on natively.
  private[framework] def stateVars: Option[Vars] = None

  private[framework] def reach: QueryBounds = frameworkReach

  // The state type, which a machine or composition object's members name through it, `def end(s:
  // State)`, so `extends MachineDeclaration[S, O, F]` is the one place it is written. The IR generator reads it
  // as the type it stands for. Its outcomes and facts are not given names here: inside an object
  // they would hide the feature's own `Outcome` and fact types.
  type State = S

  def scenario: ScenarioBuilder[S] = ScenarioBuilder("", this, None)
  def scenario(name: String): ScenarioBuilder[S] = ScenarioBuilder(name, this, None)

private[framework] trait NativeModel[S, O, F] extends Declares[S]:
  type Outcome = O
  type Fact = F
  def initialStates: Seq[S]
  def init: S
  def end(state: S): Boolean
  private[framework] def assumptions: List[EnvironmentAssumption]
  private[framework] def nativeEffects: Vector[(ActionDecl, (S, Class) => List[Step[S, O, F]])]

// What a derivation binds in its source's place: the native rules of one action.
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

// One rule of a machine's `transitions` (model/framework/dsl/Syntax.scala): the action, or one
// class of it, it fires, the condition under which it fires, and its effect, over the state and the
// class's inputs in declaration order. `index` is its place among its machine's rules, from 1, and
// `heading` and `action` are how a refusal names it and its action.
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

// What a machine a derivation makes is made from: `m.rebind(...)` is made from `m`. Its static type
// keeps the source's object type.
trait DerivedFrom[+M]:
  private[framework] def source: M

// What a `capabilities` section declares the capabilities of: the machine or
// composition whose object it sits in, with the outcomes and facts its steps answer and record.

// What a machine binds each action to, in the order the first binding of each names it: a step
// function, rules, or nothing. A machine's `transitions` (model/framework/dsl/Syntax.scala) is the
// one a Model writes; a derivation keeps what it makes as built, and the core `Bindings` binds step
// functions.
abstract class RuleBook[S, O, F]:
  private[framework] def table: Vector[(ActionDecl, Bound[S, O, F])]
  private[framework] def facetSource: Option[String] = None
  private[framework] def sectionName: String = "<transitions>"
  private[framework] def actionName(decl: ActionDecl): String = decl.toString
  private[framework] def facetOrigin: Option[FacetBindings.Origin] = None

// A step function's results for the state and one class's declared input: an effect the DSL builds
// (Programs), which the native runtime runs.
private[framework] trait InputEffect[S, O, F]
    extends ((S, List[Any]) => List[Step[S, O, F]])
    with BudgetMetadata:
  def budgets: Vector[NativeBudget[?]] = Vector.empty
  def nativeChoices: Option[Vector[Choice]] = None
  def apply(state: S, cls: Class): List[Step[S, O, F]] = apply(state, cls.values)

// A guard that reads its action's input as well as the state (Programs).
private[framework] trait InputGuard[S] extends (S => Boolean):
  final def apply(@unused state: S): Boolean =
    throw IllegalArgumentException("an input-reading guard requires its action input")
  def apply(state: S, cls: Class): Boolean

// A value as Umpire keys it in a class: a case by its name, followed by its fields, joined by `-`.
private[framework] def valueKey(v: Any): String = v match
  case p: Product if p.productArity > 0 =>
    (p.productPrefix +: p.productIterator.map(valueKey).toList).mkString("-")
  case other => other.toString

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

sealed abstract private[framework] class MachineDeclaration[S, O, F] private[framework] (using
    selectedState: Data[S],
    private[framework] val fo: Finite[O],
    private[framework] val ff: Finite[F]
)(using private[framework] val ss: StateData[S])
    extends NativeModel[S, O, F]:

  private[framework] val fs: Data[S] = selectedState

  final private[framework] def nativeEffects
      : Vector[(ActionDecl, (S, Class) => List[Step[S, O, F]])] =
    NativeRuntime.installed.machineEffects(this)

  final override private[framework] def illustrationMachine: Option[MachineDeclaration[S, O, F]] =
    Some(
      this
    )

  private var declaredInboxes = Vector.empty[InboxOwnership.Native]
  @publicInBinary final private[framework] def retainNativeInbox(
      entry: InboxOwnership.Native
  ): Unit =
    declaredInboxes :+= entry
  final private[framework] def nativeInboxes: Vector[InboxOwnership.Native] = declaredInboxes

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
  private[framework] def endpointSource: MachineDeclaration[?, ?, ?] = this
  private[framework] def evidenceSource: MachineDeclaration[?, ?, ?] = this
  private[framework] def evidenceMetadata: Option[MachineEvidence.Retained] = evidenceData.value
  final private[framework] def ownsEvidence(value: MachineEvidence.Retained): Boolean =
    evidenceData.owns(value)
  @publicInBinary final private[framework] def retainEvidence(data: Seq[String]): Unit =
    evidenceData.retain(this, data)
  private[framework] def ruleSource: MachineDeclaration[?, ?, ?] = this
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

  def initialStates: Seq[S] = Seq(init)

  // Whether the machine may end in `s`.
  def end(s: S): Boolean

  // When each action fires, and what it does then: `object transitions extends Transitions`.
  def transitions: RuleBook[S, O, F]

  // The object's name with its first letter lowered.
  def name: String = objectName(this)

  // The owner its `transitions` and other sections read its types from.
  protected given machineOwner: MachineDeclaration[S, O, F] = this

  // What its `capabilities` section declares the capabilities of: this machine.

  private var declaredDurability = Vector.empty[FieldDurability[S]]
  final protected def durability(fields: FieldDurability[S]*): Unit =
    declaredDurability ++= fields
  private[framework] def nativeDurability: Vector[FieldDurability[S]] = declaredDurability

  final protected inline def crashes(
      action: Action[EmptyTuple],
      outcome: O,
      facts: F*
  ): StepBinding[S, O, F] =
    crashBinding(machineOwner, action, Vector.empty, outcome, facts, NativeLocation.here)

  final protected inline def crashes(
      action: BudgetedAction[S, EmptyTuple],
      outcome: O,
      facts: F*
  ): StepBinding[S, O, F] =
    crashBinding(machineOwner, action.action, action.budgets, outcome, facts, NativeLocation.here)

  final private[framework] def assumptions: List[EnvironmentAssumption] =
    table
      .collect {
        case (decl, _) if decl.faultKind.contains(FaultKind.storageLoss) =>
          decl.storageLossAssumption
      }
      .toList
      .distinct

  // What each action is bound to, in the order its first rule names it.
  private var acquiredBook = Option.empty[RuleBook[S, O, F]]
  private var acquiredTable = Option.empty[DerivationBindings.Snapshot]
  private[framework] def acquiredRuleBook: Option[RuleBook[?, ?, ?]] = acquiredBook
  final private[framework] def acquireRuleBook(): RuleBook[S, O, F] =
    acquiredBook.getOrElse {
      val book = transitions
      acquiredBook = Some(book)
      book
    }
  private[framework] def acquiredRuleTable: Option[DerivationBindings.Snapshot] = acquiredTable
  final private[framework] def acquireRuleTable(
      table: Vector[(ActionDecl, Bound[S, O, F])]
  ): Vector[(ActionDecl, Bound[S, O, F])] =
    validateNativeTable(table)
    acquiredTable = Some(DerivationBindings.snapshot(table))
    table
  private[framework] def derivationParent: Option[MachineDeclaration[?, ?, ?]] = None
  private[framework] def derivationEvent: Option[DerivationBindings.Event] = None
  private[framework] def table: Vector[(ActionDecl, Bound[S, O, F])] =
    val book = transitions
    acquiredBook = Some(book)
    acquireRuleTable(book.table)

  private def boundBudgets(bound: Bound[S, O, F]): Vector[NativeBudget[?]] = bound match
    case Bound.Function(f, _) => budgetsOf(f)
    case Bound.Ruled(rules)   => rules.flatMap(rule => budgetsOf(rule.effect)).distinct
    case Bound.Disabled()     => Vector.empty

  private def boundGeneratedCrash(bound: Bound[S, O, F]): Boolean = bound match
    case Bound.Function(f, _) => generatedCrashOf(f)
    case Bound.Ruled(rules)   => rules.exists(rule => generatedCrashOf(rule.effect))
    case Bound.Disabled()     => false

  private def retainingBudgets(
      decl: ActionDecl,
      before: Bound[S, O, F],
      after: Bound[S, O, F]
  ): Bound[S, O, F] =
    val inherited = boundBudgets(before)
    if inherited.isEmpty || boundBudgets(after).nonEmpty then after
    else
      after match
        case Bound.Function(effect, captured) =>
          Bound.Function(Programs.budgeted[S, O, F](decl, effect, inherited), captured)
        case Bound.Ruled(rules) =>
          Bound.Ruled(rules.map: rule =>
            rule.copy(effect = Programs.Budgeted(Programs.inputEffect(rule.effect), inherited)))
        case Bound.Disabled() => after

  private def ownsBudget(owner: MachineDeclaration[?, ?, ?]): Boolean =
    this.eq(owner) || derivationParent.exists(_.ownsBudget(owner))

  private def validateNativeTable(entries: Vector[(ActionDecl, Bound[S, O, F])]): Unit =
    entries.foreach: (decl, bound) =>
      val context = s"Machine $name action ${decl.name}"
      decl.validateNativeFault(context)
      val budgets = boundBudgets(bound)
      require(
        budgets.size <= 1,
        s"$context: a binding declares its budget once at ${budgets.map(_.field.location).mkString(", ")}"
      )
      budgets.foreach: budget =>
        require(
          decl.faultKind.nonEmpty,
          s"$context: a budget requires a fault declaration at ${budget.field.location}"
        )
        require(
          (budget.declaration eq decl) && ownsBudget(budget.owner),
          s"$context: budget field ${budget.field.path.mkString(".")} belongs to another machine/action at ${budget.field.location}"
        )
    val crashes = entries.filter((decl, _) => decl.faultKind.contains(FaultKind.crash))
    crashes.foreach: (decl, _) =>
      require(
        crashes.count((other, _) => other.eq(decl)) == 1,
        s"Machine $name crash ${decl.name} at ${decl.nativeFaultLocation}: duplicate crash binding"
      )
    val fields = nativeDurability
    require(
      crashes.isEmpty || fields.nonEmpty,
      s"Machine $name crash ${crashes.headOption.fold("")(_._1.name)} at ${crashes.headOption.fold("native declaration")(_._1.nativeFaultLocation)} requires a durability classification"
    )
    require(
      fields.isEmpty || crashes.nonEmpty ||
        (declaredDurability.isEmpty && derivationParent.nonEmpty),
      s"Machine $name durability classification ${declaredDurability.headOption.orElse(fields.headOption).fold("")(_.field.path.mkString("."))} at ${declaredDurability.headOption.orElse(fields.headOption).fold("native declaration")(_.field.location)} requires a crash binding"
    )
    if fields.nonEmpty then
      FieldDurability.validate(
        this,
        fields,
        if crashes.nonEmpty then
          entries.flatMap((_, bound) => boundBudgets(bound).map(_.field.path))
        else Vector.empty,
        ownsBudget
      )
    val budgets =
      entries.flatMap((decl, bound) => boundBudgets(bound).map(budget => (decl, bound, budget)))
    for (decl, bound, budget) <- budgets if boundGeneratedCrash(bound) do
      require(
        !budgets.exists((other, _, selected) =>
          !other.eq(decl) && selected.field.path == budget.field.path
        ),
        s"Machine $name generated crash ${decl.name} own budget ${budget.field.path.mkString(".")} at ${budget.field.location} aliases another fault budget"
      )

  final private[framework] def nativeBudgets: Vector[NativeBudget[?]] =
    table.flatMap((_, bound) => boundBudgets(bound))

  final private[framework] def validateNativeDeclarations(): Unit =
    validateNativeTable(table)

  final private[framework] def validateNativeChoices(
      cls: Class,
      selected: Seq[Choice],
      location: String = "native performance"
  ): Unit =
    val context = s"Machine $name performance of ${cls.decl.name} at $location"
    val bound = table
      .find((decl, _) => decl eq cls.decl)
      .map(_._2)
      .getOrElse(
        throw IllegalArgumentException(s"$context: action is not bound by this machine")
      )
    val inventories = bound match
      case Bound.Function(f, _) => Vector(choiceTokens(f))
      case Bound.Ruled(rules) => rules.filter(_.fires(cls.values)).map(r => choiceTokens(r.effect))
      case Bound.Disabled()   => Vector.empty
    require(
      inventories.nonEmpty && inventories.forall(_.nonEmpty),
      s"$context: no retained native choice program for this class; compile native declarations " +
        "with -Yretain-trees and use effects with an available, static choice inventory"
    )
    require(
      selected.forall(token => inventories.exists(_.exists(_.exists(_ eq token)))),
      s"$context: selected choice is not declared by this action class"
    )

  // The machines this one is made from, which constructing it constructs as well
  // (IrFile.construct): the machine its `object refinement` refines, and for a derivation its
  // source and the machine a `refining` names.
  private[framework] def refinementEvidence: Option[Refinement[S, ?]] =
    Refinement
      .declaredBy(this)
      .map(_.asInstanceOf[Refinement[S, ?]]) // scalafix:ok DisableSyntax.asInstanceOf

  private[framework] def reaches: Seq[Model] = refinementEvidence.toSeq.map(_.of)

  private[framework] def coreOperations: Vector[CoreBindings.Operation] = Vector.empty

  private def built(
      table: => Vector[(ActionDecl, Bound[S, O, F])],
      keepRefinement: Boolean = true,
      operation: => Option[CoreBindings.Operation] = None,
      trace: => Option[DerivationBindings.Event] = None
  ): MachineDeclaration[S, O, F] & DerivedFrom[this.type] = // format: off
    Built[S, O, F, this.type](objectName(this), initialStates.toList, end, table, this, Seq.empty,
      if keepRefinement then refinementEvidence else None, operation.toVector, trace)(using // format: on
      fs,
      fo,
      ff
    )(using ss)

  // Replace the native rules of actions this machine binds, keeping its other declarations.
  // Each replacement supplies its complete guard and effect block.
  def rebind(
      replaced: (MachineDeclaration[S, O, F] ?=> Rebinding[S, O, F])*
  ): MachineDeclaration[S, O, F] & DerivedFrom[this.type] =
    rebindWith(replaced, None)

  @publicInBinary private[framework] def capturedRebind(
      replaced: (MachineDeclaration[S, O, F] ?=> Rebinding[S, O, F])*
  )(data: Seq[String]): MachineDeclaration[S, O, F] & DerivedFrom[this.type] =
    rebindWith(replaced, Some(DerivationBindings.operationSite(this, data)))

  private def rebindWith(
      replaced: Seq[MachineDeclaration[S, O, F] ?=> Rebinding[S, O, F]],
      site: Option[DerivationBindings.OperationSite]
  ): MachineDeclaration[S, O, F] & DerivedFrom[this.type] =
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
                val callback = Programs.Adapted[S, O, F](decl, f)
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
      before.map((d, b) => d -> by.get(d).fold(b)(retainingBudgets(d, b, _))),
      operation = Some(operation),
      trace = Some(DerivationBindings.Applied(operation))
    )

  // A machine that also binds actions this one does not, after its own bindings, by rules:
  // `extend(on(action) { effects.run() })`. The lifter refuses an action this machine binds
  // already, and a bare binding where this machine's actions are bound by rules.
  def extend(
      added: (MachineDeclaration[S, O, F] ?=> Rebinding[S, O, F])*
  ): MachineDeclaration[S, O, F] & DerivedFrom[this.type] =
    extendWith(added, None)

  @publicInBinary private[framework] def capturedExtend(
      added: (MachineDeclaration[S, O, F] ?=> Rebinding[S, O, F])*
  )(data: Seq[String]): MachineDeclaration[S, O, F] & DerivedFrom[this.type] =
    extendWith(added, Some(DerivationBindings.operationSite(this, data)))

  private def extendWith(
      added: Seq[MachineDeclaration[S, O, F] ?=> Rebinding[S, O, F]],
      site: Option[DerivationBindings.OperationSite]
  ): MachineDeclaration[S, O, F] & DerivedFrom[this.type] =
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
  def refining[PS, PO, PF](product: MachineDeclaration[PS, PO, PF])(
      map: S => PS
  ): MachineDeclaration[S, O, F] & DerivedFrom[this.type] =
    val refinement = new Refinement[S, PS](using machineOwner)(product):
      def toProduct(s: S): PS = map(s)
    Built[S, O, F, this.type](
      objectName(this),
      initialStates.toList,
      end,
      table,
      this,
      Seq(product),
      Some(refinement)
    )(using fs, fo, ff)(using ss)

  // A machine without this one's monitors and refinement: the same transitions, for a composition
  // a member's monitors may not watch (model/framework/SEMANTICS.md leaves that undefined).
  def unmonitored: MachineDeclaration[S, O, F] & DerivedFrom[this.type] =
    built(table, keepRefinement = false)

// A Machine: a transition relation over state declared as vars, with finite outcomes `O` and facts
// `F`. Declared as an object that is the machine, named after the object with its first letter
// lowered, whose `var`s, in declaration order, are its state, `State`, their initialisers its
// initial state, `init`:
//
// {{{
// object Kettle extends Machine[Outcome, Nothing]:
//   var temperature = 20
//   var heatingOn = false
//   object transitions extends Transitions: ...
//   object effects: ...
// }}}
//
// The compiler reads the vars where the object extends Machine (VarSchema): a Machine without vars,
// and a var whose type has no schema, do not compile. Its effects assign the vars, its guards and
// claims read them by name, and `end`, where it may stop, reads them too. The checker runs a step,
// guard or claim with its state restored into the vars, and the object reads as its initial state between
// them.
abstract class Machine[O, F](using Finite[O], Finite[F])(using schema: VarSchema)
    extends MachineDeclaration[State, O, F],
      NativeClaimWords[O, F],
      Lifted:
  private var holderName = Option.empty[String]
  final private[framework] def checkedHolder(holder: Option[String], location: String): Unit =
    synchronized:
      if holderName.isEmpty then holderName = holder
      require(
        holderName.nonEmpty,
        s"$location: ${getClass.getSimpleName}: a checked Model instance needs a stable val holder"
      )

  override def name: String = holderName.getOrElse(super.name)

  private lazy val instanceSchema = schema.bind(this)
  private lazy val declared: Vars = Vars(this, instanceSchema)
  private val initialDomains = scala.collection.mutable.LinkedHashMap.empty[String, Vector[Any]]

  final override private[framework] def stateVars: Option[Vars] = Some(declared)

  // The vars it declares, in declaration order.
  final def vars: Vector[Var] = instanceSchema.vars

  // The state its vars start in: their initialisers.
  final def init: State = declared.initial

  final override def initialStates: Seq[State] =
    instanceSchema.vars.foldLeft(Vector(init)): (states, variable) =>
      initialDomains
        .get(variable.name)
        .fold(states): domain =>
          states.flatMap(state =>
            domain.map(value => declared.updated(state, Seq(variable.name -> value)))
          )

  @publicInBinary private[framework] def registerInitial[A](name: String, domain: Vector[A]): A =
    require(domain.nonEmpty, s"$name: oneOf takes a non-empty domain")
    require(!initialDomains.contains(name), s"$name: oneOf declares one initial domain")
    initialDomains.update(name, domain.distinct)
    domain.head

  // Whether the machine may stop in the state its vars hold: anywhere, unless it says otherwise.
  def end: Boolean = true

  final def end(s: State): Boolean = declared.at(s)(end)

  // `evaluate` on the vars as they are in `state`: `Model.at(state)(temperature)`.
  final def at[A](state: State)(evaluate: => A): A = declared.at(state)(evaluate)

  // The outcome an effect rejects with and the facts it records, statements its effects make.
  protected given refusing: Refusing[O] = new Refusing[O]:
    private[framework] def refuse(outcome: O): Unit = declared.refuse(outcome)

  protected given recording: Recording[F] = new Recording[F]:
    private[framework] def add(recorded: Seq[F]): Unit = declared.add(recorded)

  // The durability of a state var across a crash: `durable(p1)` keeps it, `ephemeral(pending, 0)`
  // resets it, `inMemory(origin)(sending -> crashed)` replaces the values listed.
  final protected inline def durable[A](inline variable: A): FieldDurability[State] =
    FieldDurability.durable(this, StateField.captureVar(variable, this))

  final protected inline def ephemeral[A](inline variable: A, resetTo: A): FieldDurability[State] =
    FieldDurability.ephemeral(this, StateField.captureVar(variable, this), resetTo)

  final protected inline def inMemory[A](inline variable: A)(entries: (A, A)*)(using
      Data.Structural[A]
  ): FieldDurability[State] =
    FieldDurability.inMemory(this, StateField.captureVar(variable, this), entries.toVector)

  // The Inbox a state var holds for `channel`: `inbox(network, wire.network)`.
  final protected inline def inbox[M](inline variable: Inbox[M], channel: Channel[M]): Unit =
    ${ InboxAssociation.captureVar[M]('this, 'variable, 'channel) }

// A machine derived from another, declared as an object that is it:
// `object LateRecord extends Derived(OrderRecord.rebind(...))`, named after the object. The
// derivation is an expression of the core operations `rebind`, `extend`, `restrict`, `refining`,
// `assuming` and `unmonitored`; the object adds only the sections that are its own, `states`,
// `properties`, `capabilities` and `queries`, and reuses its source's effects by name. Its end is its
// source's.
abstract class Derived[S, O, F, M <: MachineDeclaration[S, O, F]](derivation: M)
    extends MachineDeclaration[S, O, F](using derivation.fs, derivation.fo, derivation.ff)(using
      derivation.ss
    ),
      ClaimWords,
      DerivedFrom[M]:
  final private[framework] def source: M = derivation
  override private[framework] def reach: QueryBounds = derivation.reach
  override private[framework] def nativeDurability: Vector[FieldDurability[S]] =
    derivation.nativeDurability ++ super.nativeDurability

  final override private[framework] def derivationParent: Option[MachineDeclaration[?, ?, ?]] =
    Some(
      derivation
    )
  final override private[framework] def acquiredRuleTable: Option[DerivationBindings.Snapshot] =
    derivation.acquiredRuleTable
  final override private[framework] def endpointSource: MachineDeclaration[?, ?, ?] =
    derivation.endpointSource
  final override private[framework] def evidenceSource: MachineDeclaration[?, ?, ?] =
    derivation.evidenceSource
  final override private[framework] def ruleSource: MachineDeclaration[?, ?, ?] =
    derivation.ruleSource
  override private[framework] def headerSchemaKeys: Vector[Option[String]] =
    super.headerSchemaKeys
      .zip(derivation.headerSchemaKeys)
      .map((own, inherited) => own.orElse(inherited))
  override private[framework] def factCases: Vector[(AnyRef, String)] =
    super.factCases ++ derivation.factCases
  final def init: S = derivation.init

  final override def initialStates: Seq[S] = derivation.initialStates
  final def end(s: S): Boolean = derivation.end(s)
  final def transitions: RuleBook[S, O, F] = derivation.transitions
  final override private[framework] def table: Vector[(ActionDecl, Bound[S, O, F])] =
    acquireRuleTable(derivation.table)
  final override private[framework] def coreOperations: Vector[CoreBindings.Operation] =
    derivation.coreOperations
  final override private[framework] def reaches: Seq[Model] =
    derivation +: refinementEvidence.toSeq.map(_.of)
  final override private[framework] def refinementEvidence: Option[Refinement[S, ?]] =
    super.refinementEvidence.orElse(derivation.refinementEvidence)
  final override private[framework] def stateVars: Option[Vars] = derivation.stateVars

// A machine a derivation makes from another: its name, its starts and ends, what it binds each
// action to, its source and the other machines it is made from. What else it declares only the
// lifter reads.
final private[framework] class Built[S, O, F, M <: Model](
    override val name: String,
    starts: => List[S],
    isEnd: S => Boolean,
    bound: => Vector[(ActionDecl, Bound[S, O, F])],
    private[framework] val source: M,
    also: Seq[Model],
    refinement: Option[Refinement[S, ?]],
    operations: => Vector[CoreBindings.Operation] = Vector.empty,
    trace: => Option[DerivationBindings.Event] = None
)(using Data[S], Finite[O], Finite[F])(using StateData[S])
    extends MachineDeclaration[S, O, F],
      DerivedFrom[M]:
  override private[framework] def nativeDurability: Vector[FieldDurability[S]] = source match
    case machine: MachineDeclaration[S, ?, ?] @unchecked => machine.nativeDurability
    case _                                               => super.nativeDurability
  override private[framework] def reach: QueryBounds = source match
    case owner: Declares[?] => owner.reach
    case _                  => super.reach

  override private[framework] def derivationParent: Option[MachineDeclaration[?, ?, ?]] =
    source match
      case machine: MachineDeclaration[?, ?, ?] => Some(machine)
      case _                                    => None
  override private[framework] lazy val derivationEvent: Option[DerivationBindings.Event] = trace
  override private[framework] def endpointSource: MachineDeclaration[?, ?, ?] = source match
    case machine: MachineDeclaration[?, ?, ?] => machine.endpointSource
    case _                                    => super.endpointSource
  override private[framework] def evidenceSource: MachineDeclaration[?, ?, ?] = source match
    case machine: MachineDeclaration[?, ?, ?] => machine.evidenceSource
    case _                                    => super.evidenceSource
  override private[framework] def ruleSource: MachineDeclaration[?, ?, ?] = source match
    case machine: MachineDeclaration[?, ?, ?] => machine.ruleSource
    case _                                    => super.ruleSource
  override private[framework] def headerSchemaKeys: Vector[Option[String]] = source match
    case machine: MachineDeclaration[?, ?, ?] => machine.headerSchemaKeys
    case _                                    => super.headerSchemaKeys
  override private[framework] def stateVars: Option[Vars] = source match
    case machine: MachineDeclaration[?, ?, ?] => machine.stateVars
    case _                                    => None
  override private[framework] def factCases: Vector[(AnyRef, String)] = source match
    case machine: MachineDeclaration[?, ?, ?] => super.factCases ++ machine.factCases
    case _                                    => super.factCases
  override lazy val initialStates: Seq[S] = starts
  def init: S = initialStates.head
  def end(s: S): Boolean = isEnd(s)

  object transitions extends RuleBook[S, O, F]:
    private[framework] def table: Vector[(ActionDecl, Bound[S, O, F])] = Vector.empty
  final override private[framework] lazy val table: Vector[(ActionDecl, Bound[S, O, F])] =
    acquireRuleTable(bound)
  final override private[framework] lazy val coreOperations: Vector[CoreBindings.Operation] =
    source match
      case machine: MachineDeclaration[?, ?, ?] => machine.coreOperations ++ operations
      case _                                    => operations
  final override private[framework] def reaches: Seq[Model] = source +: also
  final override private[framework] def refinementEvidence: Option[Refinement[S, ?]] = refinement
