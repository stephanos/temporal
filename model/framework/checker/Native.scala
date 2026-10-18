package framework.checker

import framework.dsl.*

// The native runtime the DSL reaches through its seam (framework/dsl/NativeRuntime.scala), by this
// object's name: each statement and evaluation the DSL declares, run here.
private[framework] object Native extends NativeRuntime:
  def invoke(body: () => Unit): Unit = body()
  def stateCompare(vars: Vars, left: State, right: State): Int =
    DomainRuntime.stateCompare(vars, left, right)
  def invoke[A, B, C, D](function: (A, B, C) => D, first: A, second: B, third: C): D =
    function(first, second, third)
  def replace[S](state: S, index: Int, value: Any, mirror: scala.deriving.Mirror.ProductOf[S]): S =
    val fields =
      state.asInstanceOf[Product].productIterator.toArray // scalafix:ok DisableSyntax.asInstanceOf
    fields(index) = value
    mirror.fromProduct(Tuple.fromArray(fields))
  def entries[M](channel: Channel[M]): Vector[Delivery[M]] = DomainRuntime.entries(channel)
  def deliveryCompare[M](channel: Channel[M], left: Delivery[M], right: Delivery[M]): Int =
    DomainRuntime.deliveryCompare(channel, left, right)
  def send[M](inbox: Inbox[M], message: M): Inbox[M] = DomainRuntime.send(inbox, message)
  def equivalent[T](data: Data[T], left: T, right: T): Boolean = data.compare(left, right) == 0
  def compareData[T](data: Data[T], left: T, right: T): Int = data.compare(left, right)
  def checkDurability[S](descriptor: FieldDurability[S]): Unit =
    DurabilityRuntime.validate(descriptor)
  def resetDurability[S](descriptor: FieldDurability[S], state: S): S =
    DurabilityRuntime.reset(descriptor, state)
  def durabilityReplacement[S](
      descriptor: FieldDurability[S],
      state: S
  ): Option[(List[String], Any)] = DurabilityRuntime.replacement(descriptor, state)
  def inputs: InputEvaluation = InputRuntime
  def stickyHolds[S, O, F](
      program: Programs.Sticky[S, O, F],
      broken: Boolean,
      before: S,
      step: Step[S, O, F]
  ): Boolean =
    broken || program.same.exists(f => !f(step)) || program.across.exists(f => !f(before, step))
  def invoke[A, B](function: A => B, argument: A): B = function(argument)
  def invoke[A, B, C](function: (A, B) => C, first: A, second: B): C = function(first, second)
  def monitorNext[M](next: M => M, current: M, before: State, step: Step[State, ?, ?]): M =
    ClaimEvaluation.watching(before, step)(next(current))
  def contents[M](channel: Channel[M]): Finite[Inbox[M]] = DomainRuntime.contents(channel)
  def inboxCompare[M](message: Data.Structural[M], left: Inbox[M], right: Inbox[M]): Int =
    DomainRuntime.inboxCompare(message, left, right)
  def sameStep[O, F](
      vars: Vars,
      predicate: Step[State, O, F] => Boolean
  ): Step[State, O, F] => Boolean = VarPredicates.sameStep(vars, predicate)
  def transition[O, F](
      vars: Vars,
      predicate: (State, Step[State, O, F]) => Boolean
  ): (State, Step[State, O, F]) => Boolean = VarPredicates.transition(vars, predicate)
  def compare[T](program: Data.Native[T], left: T, right: T): Int =
    DomainRuntime.compare(program, left, right)
  def catalog[T](mirror: scala.deriving.Mirror.Of[T], fields: List[Finite[?]]): IndexedSeq[T] =
    DomainRuntime.catalog(mirror, fields)
  def optional[A](domain: Finite[A]): IndexedSeq[Option[A]] = DomainRuntime.optional(domain)
  def range(high: Int): IndexedSeq[Int] = DomainRuntime.range(high)
  def composition[F](model: Composition[F]): CompositionExecution[F] = new CompositionRuntime(model)
  def machineEffects[S, O, F](
      model: MachineDeclaration[S, O, F]
  ): Vector[(ActionDecl, (S, Class) => List[Step[S, O, F]])] = NativeCheck.machineEffects(model)
  def section(owner: AnyRef, name: String): Vector[(String, Any)] = NativeCheck.section(owner, name)
  def classesOf(decl: ActionDecl): List[Class] = NativeCheck.classesOf(decl)
  def selects(target: Any, cls: Class): Boolean = ClaimEvaluation.selects(target, cls)
  def evaluateEffect[S, O, F](
      program: InputEffect[S, O, F],
      state: S,
      values: List[Any]
  ): List[Step[S, O, F]] = ProgramRuntime.evaluateEffect(program, state, values)
  def effectOn[S, O, F](
      program: InputEffect[S, O, F],
      state: S,
      cls: Class
  ): List[Step[S, O, F]] = ProgramRuntime.effectOn(program, state, cls)
  def holds[S](program: InputGuard[S], state: S, cls: Class): Boolean =
    ProgramRuntime.holds(program, state, cls)
  def call[R](program: AnyRef, argument: Any): R =
    ProgramRuntime.call(program, argument).asInstanceOf[R] // scalafix:ok DisableSyntax.asInstanceOf
  def call[R](program: AnyRef, first: Any, second: Any): R =
    ProgramRuntime
      .call(program, first, second)
      .asInstanceOf[R] // scalafix:ok DisableSyntax.asInstanceOf

  def initial(vars: Vars): State = VarRuntime.initial(vars)
  def at[A](vars: Vars, state: Any, evaluate: () => A): A = VarRuntime.at(vars, state)(evaluate())
  def steps[O, F](
      vars: Vars,
      state: Any,
      accepted: O,
      fires: Boolean,
      body: () => Unit
  ): List[Step[State, O, F]] = VarRuntime.steps[O, F](vars, state, accepted, fires)(body())
  def refuse(vars: Vars, outcome: Any): Unit = VarRuntime.refuse(vars, outcome)
  def add(vars: Vars, recorded: Seq[Any]): Unit = VarRuntime.add(vars, recorded)
  def fired(): Unit = VarRuntime.fired()
  def choose(alternatives: Vector[Alternative]): Unit = VarRuntime.choose(alternatives)

  def assertion(condition: () => Boolean, member: String, line: Int, text: String): Unit =
    ClaimEvaluation.assertion(condition(), member, line, text)
  def recordFact(value: () => Any, member: String, line: Int, claim: Boolean): Unit =
    ClaimEvaluation.recordFact(value(), member, line, claim)
  def alternatives(member: String, line: Int, body: () => Unit): Unit =
    ClaimEvaluation.alternatives(member, line)(body())
  def alternative(body: () => Unit): Unit = ClaimEvaluation.alternative(body())
  def abandon(member: String, reached: HoleReached): Unit = ClaimEvaluation.abandon(member, reached)
  def targeting(target: Any, member: String, location: String, block: () => Unit): Unit =
    ClaimEvaluation.targeting(target, member, location)(block())
  def start(fallback: () => State): State = ClaimEvaluation.start(fallback())
  def answer(fallback: () => Any): Any = ClaimEvaluation.answer(fallback())
  def facts: List[Any] = ClaimEvaluation.facts
  def scoped(body: () => Unit): Unit =
    if ClaimEvaluation.current.nonEmpty then ClaimEvaluation.boundary(body())
  def watching[A](before: Any, step: Step[?, ?, ?], body: () => A): A =
    ClaimEvaluation.watching(before, step)(body())
  def property(machine: Model, section: String, member: String): Property =
    ClaimEvaluation.property(machine, section, member)
