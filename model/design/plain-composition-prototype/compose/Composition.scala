package framework.dsl
package compose

import scala.language.dynamics

// fn-193.1 probe: the plain composition form. A composition is an object whose members are vals
// naming plain Machines; its state is the members' vars, read through the member (`left.phase`).
// Nothing here changes framework sources: the probe sits in a subpackage of `framework.dsl`, so it
// reaches the native runtime's `private[framework]` parts (Vars, NativeClaims, NativeCheck) and its
// `Composition` shadows the record-state one for the fixtures written in the same package.
//
// {{{
// object Together extends Composition[JoinedFact]:
//   val left = Subject
//   val right = Other
//   object syncs:
//     val started = sync(left(hand.start), right(hand.start))
//   object invariants:
//     val rightClosedWithLeft = if left.isClosed then check(right.phase == Phase.closed)
// }}}

// One member of a composition as its declaration names it: the val and the object it names.
final class Member private[compose] (val name: String, val declared: RecordMachine[?, ?, ?]):
  override def toString: String = s"$name = ${declared.name}"

// The members a composition object declares as vals, in declaration order, and how each member's
// fact is wrapped in the composition's own fact enum (`JoinedFact.left(f)`). The compiler derives
// it where the object extends `Composition[F]` (ComposeMacros.schema); the members' objects are read
// back through the vals' getters, by name, when the runtime first needs them.
final class Schema[F] private[compose] (
    private[compose] val names: Vector[String],
    private[compose] val tags: Vector[Option[Any => F]]
)

object Schema:
  inline given derived[F]: Schema[F] = ${ ComposeMacros.schema[F] }

  @scala.annotation.publicInBinary
  private[compose] def of[F](names: List[String], tags: List[Option[Any => F]]): Schema[F] =
    Schema(names.toVector, tags.toVector)

// `Together.withMember(right = LateOther)`: the source with one member replaced, which a derived
// composition object extends, `object LateTogether extends Composition(Together.withMember(...))`.
final class Derivation[F] private[compose] (
    private[compose] val source: Composition[F],
    private[compose] val replaced: Map[String, RecordMachine[?, ?, ?]]
)

// A step of one member's action, where two members bind the action and a sync or a claim must say
// whose it is: `left(hand.start)`.
final class MemberStep private[compose] (val machine: RecordMachine[?, ?, ?], val action: Action[?]):
  override def toString: String = s"${machine.name}(${NativeCheck.actionName(action.decl)})"

extension (machine: RecordMachine[?, ?, ?])
  // `left(hand.start)`: the start that member `left` takes.
  def apply(action: Action[?]): MemberStep = MemberStep(machine, action)

// One end of a sync: an action one member binds, or a member's action.
type Endpoint = Action[?] | MemberStep

// A sync, named after its val: `val admit = sync(activityWorker.poll, queue.deliver)`. The inline
// `sync` calls a factory: an inline method that calls a `@publicInBinary` private constructor, used
// in the run that compiles it, sends Scala 3.9.0 into an endless loop in its inlining phase.
object Sync:
  def declared(first: Endpoint, second: Endpoint, name: Option[String], at: String): Sync =
    new Sync(first, second, name, at)

final class Sync private[compose] (
    val first: Endpoint,
    val second: Endpoint,
    private[compose] val declaredName: Option[String],
    private[compose] val at: String
):
  def name: String = declaredName.getOrElse(s"$first")
  override def toString: String = name

// What `after(...)` and a witness select: a sync, an action (its own step, or the sync that pairs
// it), or one member's action.
type Target = Sync | Action[?] | MemberStep

// A witness, `val reopenedFound = find(reopened)`: some step the target claim holds of and is about.
final class Find @scala.annotation.publicInBinary private[compose] (val target: String)

abstract class Composition[F] private (private[compose] val compositionShape: Composition.Shape[F])
    extends Model:
  // A composition of the members its vals name.
  def this()(using schema: Schema[F]) = this(Composition.Shape.Declared(schema))

  // A composition derived from another, keeping its fact type, syncs, end and member order.
  def this(derivation: Derivation[F]) = this(Composition.Shape.Derived(derivation))

  def name: String = objectName(this)

  // Whether the composition may stop where its members' vars are now: anywhere unless it says.
  def end: Boolean = compositionShape match
    case Composition.Shape.Derived(d) => d.source.end
    case _                            => true

  // `Together.withMember(right = LateOther)`: named updates checked against the members.
  final def withMember: Composition.Updates[F] = Composition.Updates(this)

  // `sync(left(hand.start), right(hand.start))` in `object syncs`, named after its val.
  final protected inline def sync(first: Endpoint, second: Endpoint): Sync =
    Sync.declared(first, second, NativeLocation.declaredName, NativeLocation.here)

  // `after(syncs.started):` a block about the steps of a target, in `postconditions` or `queries`.
  final protected inline def after(target: Target)(inline block: Unit): Unit =
    ComposeRuntime.targeting(target, NativeLocation.declaredName.getOrElse(""))(block)

  // `records(JoinedFact.left(Fact.completed))`: that the step recorded the tagged member fact.
  final protected inline def records(fact: F): Unit =
    NativeClaims.recordFact(fact, NativeLocation.declaredName.getOrElse(""), ComposeMacros.line, true)

  // `before.left.phase`: the members' states the step started from, typed by the members' vars.
  final protected transparent inline def before: Any = ${ ComposeMacros.before }

  // `find(reopened)` in `queries`: a witness of the claim block `reopened`.
  final protected inline def find(inline target: Unit): Find = ${ ComposeMacros.find('target) }

  // The members, each as declared and as it stands here (a derivation's replacement).
  private[compose] lazy val memberTable: Vector[(Member, RecordMachine[?, ?, ?])] = compositionShape match
    case Composition.Shape.Declared(schema) =>
      schema.names.map: n =>
        val value = getClass.getMethod(n).invoke(this) match
          case m: RecordMachine[?, ?, ?] => m
          case other                     =>
            throw IllegalArgumentException(s"$name.$n names $other, which is no Machine")
        (Member(n, value), value)
    case Composition.Shape.Derived(d) =>
      d.source.memberTable.map((member, actual) => (member, d.replaced.getOrElse(member.name, actual)))

  private[compose] def factTags: Vector[Option[Any => F]] = compositionShape match
    case Composition.Shape.Declared(schema) => schema.tags
    case Composition.Shape.Derived(d)       => d.source.factTags

  // The syncs: a declared composition's `object syncs`, a derivation's source's.
  private[compose] lazy val syncTable: Vector[Sync] = compositionShape match
    case Composition.Shape.Declared(_) =>
      NativeCheck.section(this, "syncs").collect { case (_, s: Sync) => s }
    case Composition.Shape.Derived(d) => d.source.syncTable

  // The composition its sections were declared on before this one: a derivation's source.
  private[compose] def derivedFrom: Option[Composition[F]] = compositionShape match
    case Composition.Shape.Derived(d) => Some(d.source)
    case _                            => None

object Composition:
  private[compose] enum Shape[F]:
    case Declared(schema: Schema[F])
    case Derived(derivation: Derivation[F])

  // The named updates of a composition's members, `withMember(right = LateOther)`. A replacement
  // holds the vars of the member it replaces (it is that Machine or a derivation of it), so the
  // source's claims, which read the member through its vars, read the replacement.
  final class Updates[F] private[compose] (composition: Composition[F]) extends Dynamic:
    def applyDynamicNamed(method: String)(named: (String, Any)*): Derivation[F] =
      require(method == "apply", s"withMember.$method is not a member update")
      val updates = named.map:
        case (n, m: RecordMachine[?, ?, ?]) => n -> m
        case (n, other)                     =>
          throw IllegalArgumentException(s"$n = $other: a member is replaced by a Machine")
      val members = composition.memberTable.map((m, actual) => m.name -> actual).toMap
      for (n, replacement) <- updates do
        val current = members.getOrElse(
          n,
          throw IllegalArgumentException(
            s"$n is not a member of ${composition.name}: its members are " +
              composition.memberTable.map(_._1.name).mkString(", ")
          )
        )
        if !(Vars.of(replacement) eq Vars.of(current)) then
          throw IllegalArgumentException(
            s"${composition.name}.withMember($n = ${replacement.name}): ${replacement.name} does " +
              s"not hold ${current.name}'s vars; a replacement is ${current.name} or a derivation " +
              s"of it, so the claims that read $n read it"
          )
      Derivation(composition, updates.toMap)
