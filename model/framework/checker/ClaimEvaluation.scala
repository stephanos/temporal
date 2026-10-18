package framework.checker

import framework.dsl.*

import scala.collection.mutable

// The native evaluation of a Machine's claim sections. A member of `invariants` or `postconditions` is a `Unit` val whose block
// asserts as it runs: each assertion records its member, line, source text and result under the
// evaluation under way, and outside an evaluation records nothing. A section is evaluated on one
// state, or one step, by restoring the state into the Machine's vars and running the section
// object's initialiser again, from its class's bytes in a child class loader that sees every other
// class through its parent, so the members read the live vars. Evaluations of one Machine run one at
// a time, as its vars are one live state.
private[framework] object ClaimEvaluation:
  final private[framework] class Stopped extends scala.util.control.ControlThrowable
  private val stopped = new Stopped

  def boundary(body: => Unit): Unit =
    try body
    catch case _: Stopped => ()

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
    private val uncertain = mutable.Stack(mutable.ArrayBuffer.empty[(String, Hole)])
    private val choosing =
      mutable.Stack.empty[mutable.ArrayBuffer[(Boolean, Vector[(String, Hole)])]]
    private val choosingMembers = mutable.Stack.empty[String]
    private[ClaimEvaluation] var aborted = Option.empty[Hole]
    def checks: Vector[Checked] = collecting.last.toVector
    def missing(member: String): Vector[Hole] =
      aborted.toVector ++ uncertain.last.collect { case (`member`, hole) => hole }
    def missing: Vector[Hole] = aborted.toVector ++ uncertain.last.map(_._2)
    // The members about the steps of one target, with that target, and those this step triggered.
    val targeted: mutable.LinkedHashMap[String, Any] = mutable.LinkedHashMap.empty
    val triggered: mutable.LinkedHashSet[String] = mutable.LinkedHashSet.empty
    val seen: mutable.LinkedHashSet[String] = mutable.LinkedHashSet.empty

    private[ClaimEvaluation] def add(checked: Checked): Unit =
      seen += checked.member: Unit
      collecting.top += checked: Unit

    private[ClaimEvaluation] def failed(member: String): Boolean =
      collecting.top.exists(c => c.member == member && !c.holds) ||
        uncertain.top.exists(_._1 == member)

    private[ClaimEvaluation] def stop(member: String): Unit =
      if failed(member) then throw stopped

    private[ClaimEvaluation] def attempt(member: String)(body: => Unit): Unit =
      try body
      catch
        case HoleReached(hole) =>
          seen += member: Unit
          uncertain.top += ((member, hole)): Unit

    private[ClaimEvaluation] def alternatives(member: String, line: Int)(body: => Unit): Unit =
      if !failed(member) then
        seen += member: Unit
        choosing.push(mutable.ArrayBuffer.empty)
        choosingMembers.push(member): Unit
        val alternatives =
          try
            body
            choosing.top.toVector
          finally
            val _ = choosing.pop()
            val _ = choosingMembers.pop()
        if alternatives.exists((held, holes) => held && holes.isEmpty) then
          add(Checked(member, line, "oneOf", true))
        else
          val holes = alternatives.filter(_._1).flatMap(_._2)
          if holes.nonEmpty then uncertain.top ++= holes: Unit
          else add(Checked(member, line, "oneOf", false))

    private[ClaimEvaluation] def alternative(body: => Unit): Unit =
      require(choosing.nonEmpty, "a branch is an alternative of a oneOf, written inside it")
      if !choosing.top.exists((held, holes) => held && holes.isEmpty) then
        collecting.push(mutable.ArrayBuffer.empty)
        uncertain.push(mutable.ArrayBuffer.empty)
        val result =
          try
            boundary(attempt(choosingMembers.top)(body))
            (collecting.top.forall(_.holds), uncertain.top.toVector)
          finally
            val _ = collecting.pop()
            val _ = uncertain.pop()
        choosing.top += result: Unit

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
    current.foreach: c =>
      c.attempt(member):
        if !c.failed(member) then c.add(Checked(member, line, text, condition))
      c.stop(member)

  // A fact: in a claim, an assertion that the step under evaluation recorded it; in an effect, a
  // fact the step records. Outside both, a claim's assertion records nothing, and an effect's fact
  // is refused, as the effect runs only as its transition's step.
  def recordFact(value: => Any, member: String, line: Int, claim: Boolean): Unit =
    current match
      case Some(c) =>
        c.attempt(member):
          if !c.failed(member) then
            val fact = value
            c.add(Checked(member, line, s"records($fact)", c.recorded.contains(fact)))
        c.stop(member)
      case None =>
        VarRuntime.current match
          case Some(frame) => frame.facts = frame.facts :+ value
          case None        =>
            if !claim then
              throw IllegalStateException(
                "a fact an effect records is a statement of the step its transition runs"
              )

  // Alternatives, of which at least one holds: one assertion of the member.
  def alternatives(member: String, line: Int)(body: => Unit): Unit =
    current.foreach: c =>
      c.alternatives(member, line)(body)
      c.stop(member)

  def abandon(member: String, reached: HoleReached): Unit =
    current.foreach(_.attempt(member)(throw reached))

  // One alternative: it holds where none of its assertions failed.
  def alternative(body: => Unit): Unit = current.foreach(_.alternative(body))

  // A block about the steps of `target`, which runs where the step took it.
  def targeting(target: Any, member: String, location: String)(block: => Unit): Unit =
    current.foreach: c =>
      c.origin
        .map(_.owner.machine)
        .foreach:
          case composition: Composition[?] =>
            composition.validateTarget(target, s"$location: $member", required = false)
          case _ => ()
      c.seen += member: Unit
      c.targeted.getOrElseUpdate(member, target): Unit
      if c.taken.exists(selects(target, _)) then
        c.triggered += member: Unit
        c.attempt(member)(block)
        c.stop(member)

  // Whether a step of `cls` is a step of `target`: the class itself, or any class of an action.
  def selects(target: Any, cls: Class): Boolean = target match
    case _ if cls.decl.selectsComposed.nonEmpty => cls.decl.selectsComposed.get(target, cls)
    case c: Class     => (c.decl eq cls.decl) && java.util.Objects.equals(c.values, cls.values)
    case a: Action[?] => a.decl eq cls.decl
    case _: CompositionTarget => false
    case other                =>
      throw IllegalArgumentException(s"$other selects a composed step, which no native check reads")

  // The state the step under evaluation started from, or `fallback` outside an evaluation.
  def start(fallback: => State): State = current.flatMap(_.origin).getOrElse(fallback)

  // The outcome the step under evaluation answered, or `fallback` outside an evaluation.
  def answer(fallback: => Any): Any = current.flatMap(_.answered).getOrElse(fallback)

  // The facts the step under evaluation recorded, none outside an evaluation.
  def facts: List[Any] = current.fold(Nil)(_.recorded)

  // Evaluates `body` on a step a monitor watches: with the state the step reached restored into the
  // vars, and the state it started from, its outcome and its facts under evaluation.
  def watching[A](before: Any, step: Step[?, ?, ?])(body: => A): A = (before, step.state) match
    case (origin: State, now: State) =>
      now.owner.at(now)(within(Context(Some(origin), None, Some(step.outcome), step.facts))(body))
    case _ =>
      throw IllegalArgumentException(
        s"${step.state} is no state of a Machine that declares its vars: a monitor of vars watches one"
      )

  // Finds a section without invoking its getter. An instance section belongs to the exact Machine
  // class that declares its getter and supplies its constructor's outer receiver, never a facet.
  def sectionClass(
      owner: AnyRef,
      section: String,
      requireStatic: Boolean = false
  ): Option[java.lang.Class[?]] =
    scala.util
      .Try(
        java.lang.Class.forName(
          owner.getClass.getName.stripSuffix("$") + "$" + section + "$",
          false,
          loaderOf(owner)
        )
      )
      .toOption
      .orElse(
        owner.getClass.getMethods
          .find(m =>
            m.getName == section && m.getParameterCount == 0 &&
              m.getReturnType.getName.endsWith("$" + section + "$")
          )
          .map(_.getReturnType)
      )
      .map: cls =>
        val static = scala.util
          .Try(cls.getField("MODULE$"))
          .toOption
          .exists(f => java.lang.reflect.Modifier.isStatic(f.getModifiers))
        require(
          static || ownedSection(owner, cls, section) ||
            (!requireStatic && !Vector("invariants", "postconditions", "queries")
              .contains(section) && members(cls).isEmpty),
          s"${cls.getName} is a claim section of an instance, such as a facet's, which no native " +
            "check evaluates: declare the claims in the Machine object's own sections"
        )
        cls

  private def ownedSection(owner: AnyRef, cls: java.lang.Class[?], section: String): Boolean =
    owner.isInstanceOf[Machine[?, ?]] && owner.getClass.getDeclaredMethods.exists(m =>
      m.getName == section && m.getParameterCount == 0 && m.getReturnType.equals(cls)
    ) && cls.getConstructors.exists(c =>
      c.getParameterCount == 1 && c.getParameterTypes.head.equals(owner.getClass)
    )

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
  def requireOwner(vars: Vars, owner: AnyRef): Unit = owner match
    case machine: Machine[?, ?] if !machine.getClass.getName.endsWith("$") =>
      require(
        machine eq vars.machine,
        "a claim of another Model instance cannot read this Model's state"
      )
    case _ => ()

  def evaluate(vars: Vars, section: java.lang.Class[?], now: State, context: Context): Context =
    require(
      section.getFields.exists(f =>
        f.getName == "MODULE$" && java.lang.reflect.Modifier.isStatic(f.getModifiers)
      ),
      s"${section.getName} requires its declaring Model instance"
    )
    evaluate(vars, section, vars.machine, now, context)

  def evaluate(
      vars: Vars,
      section: java.lang.Class[?],
      owner: AnyRef,
      now: State,
      context: Context
  ): Context =
    requireOwner(vars, owner)
    try
      val _ = vars.at(now)(within(context)(Reload.initialise(section, owner = Some(owner))))
    catch case HoleReached(hole) => context.aborted = Some(hole)
    context

  // Evaluates an ad-hoc claim on the vars of `vars` holding `now`, under `context`.
  def evaluate(vars: Vars, claim: NativeClaims.Adhoc, now: State, context: Context): Context =
    try vars.at(now)(within(context)(boundary(claim.body())))
    catch case HoleReached(hole) => context.aborted = Some(hole)
    context

  // Reruns a section's initializer in a child loader that sees every other class through its parent.
  private object Reload:
    private val bytes =
      java.util.concurrent.ConcurrentHashMap[(java.lang.Class[?], Boolean), Array[Byte]]()

    def initialise(
        target: java.lang.Class[?],
        declarations: Boolean = false,
        owner: Option[AnyRef] = None
    ): AnyRef =
      val parent = target.getClassLoader
      val wanted = target.getName
      val code = bytes.computeIfAbsent(
        (target, declarations),
        _ =>
          val stream = parent.getResourceAsStream(wanted.replace('.', '/') + ".class")
          try NativeClaimSections.prepare(target, stream.readAllBytes(), declarations)
          finally stream.close()
      )
      val loader = new ClassLoader(parent):
        override def loadClass(n: String, resolve: Boolean): java.lang.Class[?] =
          if n == wanted then
            Option(findLoadedClass(n)).getOrElse(defineClass(n, code, 0, code.length))
          else super.loadClass(n, resolve)
      try
        val reloaded = java.lang.Class.forName(wanted, true, loader)
        scala.util.Try(reloaded.getField("MODULE$")).toOption match
          case Some(field) => field.get(null) // scalafix:ok DisableSyntax.null
          case None        =>
            val instance = owner.getOrElse(
              throw IllegalArgumentException(s"$wanted requires its Model instance")
            )
            reloaded.getConstructor(instance.getClass).newInstance(instance)
      catch
        case failed: ExceptionInInitializerError => throw Option(failed.getCause).getOrElse(failed)
        case failed: java.lang.reflect.InvocationTargetException =>
          throw Option(failed.getCause).getOrElse(failed)

  private val declared = java.util.concurrent.ConcurrentHashMap[java.lang.Class[?], AnyRef]()

  def declarations(section: java.lang.Class[?]): AnyRef =
    declared.computeIfAbsent(section, _ => Reload.initialise(section, declarations = true))

  def declarations(section: java.lang.Class[?], owner: AnyRef): AnyRef =
    if section.getConstructors.exists(_.getParameterCount == 1) then
      Reload.initialise(section, declarations = true, owner = Some(owner))
    else declarations(section)

  def property(machine: Model, section: String, member: String): Property =
    val cls = sectionClass(machine, section, requireStatic = true).getOrElse(
      throw IllegalArgumentException(s"${machine.name} has no $section section")
    )
    val module = declarations(cls, machine)
    module.getClass
      .getMethod(member)
      .invoke(module)
      .asInstanceOf[Property] // scalafix:ok DisableSyntax.asInstanceOf
