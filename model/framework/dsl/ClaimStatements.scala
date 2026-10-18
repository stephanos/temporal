package framework.dsl

// The statements of a Machine's claim sections, which the claim words expand to (ClaimMacros), and
// the claims they declare. A member of `invariants` or `postconditions` is a `Unit` val whose block
// asserts as it runs; its assertions mean something only under an evaluation of its section, which
// the checker makes through NativeRuntime. Outside an evaluation an assertion records nothing and reads nothing, and so
// does every statement here where no runtime is on the class path.
private[framework] object NativeClaims:
  def selects(target: Any, cls: Class): Boolean = NativeRuntime.installed.selects(target, cls)
  // A claim written where it is checked rather than in a section, such as a negative control's in a
  // test, `invariant(check(!beeping))`: its block, run in each evaluation.
  final class Adhoc @scala.annotation.publicInBinary private[framework] (
      val name: String,
      val body: () => Unit
  )

  // A member of a claim section of `machine`, as `verify` and `find` name it.
  final class Ref private[framework] (val machine: Model, val section: String, val member: String):
    override def toString: String = s"${machine.name}.$section.$member"

  // An assertion of a claim.
  def assertion(condition: => Boolean, member: String, line: Int, text: String): Unit =
    NativeRuntime.loaded.foreach(_.assertion(() => condition, member, line, text))

  // A fact: in a claim, an assertion that the step under evaluation recorded it; in an effect, a
  // fact the step records, which only the step its transition runs records.
  def recordFact(value: => Any, member: String, line: Int, claim: Boolean): Unit =
    NativeRuntime.loaded match
      case Some(runtime) => runtime.recordFact(() => value, member, line, claim)
      case None          =>
        if !claim then
          throw IllegalStateException(
            "a fact an effect records is a statement of the step its transition runs"
          )

  // Alternatives, of which at least one holds: one assertion of the member.
  def alternatives(member: String, line: Int)(body: => Unit): Unit =
    NativeRuntime.loaded.foreach(_.alternatives(member, line, () => body))

  // An alternative cut short by a hole of the model; the rewritten section calls it.
  def abandon(member: String, reached: HoleReached): Unit =
    NativeRuntime.loaded.foreach(_.abandon(member, reached))

  // One alternative: it holds where none of its assertions failed.
  def alternative(body: => Unit): Unit = NativeRuntime.loaded.foreach(_.alternative(() => body))

  // A block about the steps of `target`, which runs where the step took it.
  def targeting(target: Any, member: String, location: String)(block: => Unit): Unit =
    NativeRuntime.loaded.foreach(_.targeting(target, member, location, () => block))

  // The state the step under evaluation started from, or `fallback` outside an evaluation.
  def start(fallback: => State): State =
    NativeRuntime.loaded.fold(fallback)(_.start(() => fallback))

  // The outcome the step under evaluation answered, or `fallback` outside an evaluation.
  def answer(fallback: => Any): Any = NativeRuntime.loaded.fold(fallback)(_.answer(() => fallback))

  // The facts the step under evaluation recorded, none outside an evaluation.
  def facts: List[Any] = NativeRuntime.loaded.fold(Nil)(_.facts)

  // The block of a scoped claim, which asserts only under an evaluation.
  def scoped(body: => Unit): Unit = NativeRuntime.loaded.foreach(_.scoped(() => body))

  // `body` on a step a monitor watches: the state the step reached restored into the vars, and the
  // state it started from, its outcome and its facts under evaluation.
  def watching[A](before: Any, step: Step[?, ?, ?])(body: => A): A =
    NativeRuntime.installed.watching(before, step, () => body)

  // The Property a claim section's member declares, `verify(invariants.consistency)`.
  def property(machine: Model, section: String, member: String): Property =
    NativeRuntime.installed.property(machine, section, member)
