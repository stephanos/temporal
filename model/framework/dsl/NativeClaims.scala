package framework.dsl

import scala.collection.mutable

// The native evaluation of a Machine's claim sections in phase 1 (fn-190.4), until the IR evaluator
// replaces it (fn-190.3/.5). A member of `invariants` or `postconditions` is a `Unit` val whose block
// asserts as it runs: each assertion records its member, line, source text and result under the
// evaluation under way, and outside an evaluation records nothing. A section is evaluated on one
// state, or one step, by restoring the state into the Machine's vars and running the section
// object's initialiser again, from its class's bytes in a child class loader that sees every other
// class through its parent, so the members read the live vars. Evaluations of one Machine run one at
// a time, as its vars are one live state.
private[framework] object NativeClaims:
  // One assertion a claim made: the member it belongs to, its line and source text, and whether it
  // held.
  final case class Checked(member: String, line: Int, text: String, holds: Boolean)

  // One evaluation: of a state, or of a step from `origin` taking `taken`, answering `answered` and
  // recording `recorded`. It collects what the members asserted, which members are about the steps
  // of one target and which of those the step triggered, and the order the members ran in.
  final class Context(
      val origin: Option[State] = None,
      val taken: Option[Class] = None,
      val answered: Option[Any] = None,
      val recorded: List[Any] = Nil
  ):
    private val collecting = mutable.Stack(mutable.ArrayBuffer.empty[Checked])
    private val choosing = mutable.Stack.empty[mutable.ArrayBuffer[Boolean]]
    def checks: Vector[Checked] = collecting.last.toVector
    // The members about the steps of one target, with that target, and those this step triggered.
    val targeted: mutable.LinkedHashMap[String, Any] = mutable.LinkedHashMap.empty
    val triggered: mutable.LinkedHashSet[String] = mutable.LinkedHashSet.empty
    val seen: mutable.LinkedHashSet[String] = mutable.LinkedHashSet.empty

    private[NativeClaims] def add(checked: Checked): Unit =
      seen += checked.member: Unit
      collecting.top += checked: Unit

    private[NativeClaims] def alternatives(member: String, line: Int)(body: => Unit): Unit =
      seen += member: Unit
      choosing.push(mutable.ArrayBuffer.empty)
      val held =
        try
          body
          choosing.top.exists(identity)
        finally
          val _ = choosing.pop()
      add(Checked(member, line, "oneOf", held))

    private[NativeClaims] def alternative(body: => Unit): Unit =
      require(choosing.nonEmpty, "a branch is an alternative of a oneOf, written inside it")
      collecting.push(mutable.ArrayBuffer.empty)
      val held =
        try
          body
          collecting.top.forall(_.holds)
        finally
          val _ = collecting.pop()
      choosing.top += held: Unit

    // The first assertion of `member` that failed.
    def failure(member: String): Option[Checked] =
      checks.find(c => c.member == member && !c.holds)

    // Whether the evaluation is about `member`: every step, unless the member is about the steps
    // of one target, which this step then took.
    def applies(member: String): Boolean = !targeted.contains(member) || triggered(member)

  // The evaluations under way on this thread, innermost first.
  private val running = ThreadLocal.withInitial[List[Context]](() => Nil)

  def current: Option[Context] = running.get.headOption

  def within[A](context: Context)(body: => A): A =
    running.set(context :: running.get)
    try body
    finally running.set(running.get.drop(1))

  // An assertion of a claim: recorded under the evaluation under way, nothing outside one.
  def assertion(condition: => Boolean, member: String, line: Int, text: String): Unit =
    current.foreach(_.add(Checked(member, line, text, condition)))

  // A fact: in a claim, an assertion that the step under evaluation recorded it; in an effect, a
  // fact the step records. Outside both, a claim's assertion records nothing, and an effect's fact
  // is refused, as the effect runs only as its transition's step.
  def recordFact(value: Any, member: String, line: Int, claim: Boolean): Unit =
    current match
      case Some(c) => c.add(Checked(member, line, s"records($value)", c.recorded.contains(value)))
      case None    =>
        Vars.current match
          case Some(frame) => frame.facts = frame.facts :+ value
          case None        =>
            if !claim then
              throw IllegalStateException(
                "a fact an effect records is a statement of the step its transition runs"
              )

  // Alternatives, of which at least one holds: one assertion of the member.
  def alternatives(member: String, line: Int)(body: => Unit): Unit =
    current.foreach(_.alternatives(member, line)(body))

  // One alternative: it holds where none of its assertions failed.
  def alternative(body: => Unit): Unit = current.foreach(_.alternative(body))

  // A block about the steps of `target`, which runs where the step took it.
  def targeting(target: Any, member: String)(block: => Unit): Unit =
    current.foreach: c =>
      c.seen += member: Unit
      c.targeted.getOrElseUpdate(member, target): Unit
      if c.taken.exists(selects(target, _)) then
        c.triggered += member: Unit
        block

  // Whether a step of `cls` is a step of `target`: the class itself, or any class of an action.
  def selects(target: Any, cls: Class): Boolean = target match
    case c: Class     => (c.decl eq cls.decl) && java.util.Objects.equals(c.values, cls.values)
    case a: Action[?] => a.decl eq cls.decl
    case other        =>
      throw IllegalArgumentException(s"$other selects a composed step, which no native check reads")

  // The state the step under evaluation started from, or `fallback` outside an evaluation.
  def start(fallback: => State): State = current.flatMap(_.origin).getOrElse(fallback)

  // The outcome the step under evaluation answered, or `fallback` outside an evaluation.
  def answer(fallback: => Any): Any = current.flatMap(_.answered).getOrElse(fallback)

  // The facts the step under evaluation recorded, none outside an evaluation.
  def facts: List[Any] = current.fold(Nil)(_.recorded)

  // A claim written where it is checked rather than in a section, such as a negative control's in a
  // test, `invariant(check(!beeping))`: its block, run in each evaluation.
  final class Adhoc @scala.annotation.publicInBinary private[framework] (
      val name: String,
      val body: () => Unit
  )

  // A member of a claim section of `machine`, as `verify` and `find` name it.
  final class Ref private[framework] (val machine: Model, val section: String, val member: String):
    override def toString: String = s"${machine.name}.$section.$member"

  // The class a claim section of `owner` (a Machine object, or the object its sections are read
  // from) compiles to, where it declares one as a nested object. A section of an instance, such as
  // a facet's, is no static object, and its claims are not evaluated natively.
  def sectionClass(owner: AnyRef, section: String): Option[java.lang.Class[?]] =
    scala.util
      .Try(
        java.lang.Class.forName(owner.getClass.getName + section + "$", false, loaderOf(owner))
      )
      .toOption
      .map: cls =>
        val static = scala.util
          .Try(cls.getField("MODULE$"))
          .toOption
          .exists(f => java.lang.reflect.Modifier.isStatic(f.getModifiers))
        require(
          static || members(cls).isEmpty,
          s"${cls.getName} is a claim section of an instance, such as a facet's, which no native " +
            "check evaluates: declare the claims in the Machine object's own sections"
        )
        cls

  private def loaderOf(owner: AnyRef): ClassLoader = owner.getClass.getClassLoader

  // The claim members of a section: its `Unit` vals, in the order their getters are declared.
  def members(section: java.lang.Class[?]): Vector[String] =
    section.getDeclaredMethods.toVector
      .filter(m =>
        m.getParameterCount == 0 && java.lang.reflect.Modifier.isPublic(m.getModifiers) &&
          !m.isSynthetic && !m.getName.contains("$") &&
          m.getReturnType.equals(classOf[scala.runtime.BoxedUnit])
      )
      .map(_.getName)

  // Evaluates a section of `owner` on the vars of `vars` holding `now`, under `context`.
  def evaluate(vars: Vars, section: java.lang.Class[?], now: State, context: Context): Context =
    vars.at(now)(within(context)(Reload.initialise(section)))
    context

  // Evaluates an ad-hoc claim on the vars of `vars` holding `now`, under `context`.
  def evaluate(vars: Vars, claim: Adhoc, now: State, context: Context): Context =
    vars.at(now)(within(context)(claim.body()))
    context

  // Reruns a static object's initialiser: defines the class afresh in a child loader that sees every
  // other class through its parent.
  private object Reload:
    private val bytes = java.util.concurrent.ConcurrentHashMap[java.lang.Class[?], Array[Byte]]()

    def initialise(target: java.lang.Class[?]): Unit =
      val parent = target.getClassLoader
      val wanted = target.getName
      val code = bytes.computeIfAbsent(
        target,
        _ =>
          val stream = parent.getResourceAsStream(wanted.replace('.', '/') + ".class")
          try stream.readAllBytes()
          finally stream.close()
      )
      val loader = new ClassLoader(parent):
        override def loadClass(n: String, resolve: Boolean): java.lang.Class[?] =
          if n == wanted then
            Option(findLoadedClass(n)).getOrElse(defineClass(n, code, 0, code.length))
          else super.loadClass(n, resolve)
      try
        val _ = java.lang.Class.forName(wanted, true, loader)
      catch
        case failed: ExceptionInInitializerError => throw Option(failed.getCause).getOrElse(failed)
