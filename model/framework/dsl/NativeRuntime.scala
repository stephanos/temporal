package framework.dsl

final private[framework] case class SuiteCheck(
    evidence: Any,
    verified: Boolean,
    inconclusive: Boolean,
    diagnostics: String
)

private[framework] trait NativeSuiteExecution:
  def budget: Int
  def claims[M <: Machine[?, ?]](model: M): SuiteCheck
  def replay[M <: Machine[?, ?]](body: RunBody[M]): SuiteCheck
  def find[M <: Machine[?, ?]](model: M, request: SpaceFind[M], from: Option[State]): SuiteCheck
  def sample[M <: Machine[?, ?]](model: M, request: SpaceSample[M], from: Option[State]): SuiteCheck
  def refuted[M <: Machine[?, ?]](
      model: M,
      request: SpaceRefuted[M],
      from: Option[State]
  ): SuiteCheck

// Declaration operations implemented by the product checker. Keeping the interface here lets the
// lifter read declarations without loading the checker or evaluating authored callbacks.
private[framework] trait NativeRuntime:
  def suiteExecution(exactName: String): NativeSuiteExecution
  def runBody[M <: Machine[?, ?]](body: RunBody[M], budget: Int): Any
  def invoke(body: () => Unit): Unit
  def stateCompare(vars: Vars, left: State, right: State): Int
  def invoke[A, B, C, D](
      function: (A, B, C) => D,
      first: A,
      second: B,
      third: C
  ): D
  def replace[S](
      state: S,
      index: Int,
      value: Any,
      mirror: scala.deriving.Mirror.ProductOf[S]
  ): S
  def entries[M](channel: Channel[M]): Vector[Delivery[M]]
  def deliveryCompare[M](
      channel: Channel[M],
      left: Delivery[M],
      right: Delivery[M]
  ): Int
  def send[M](inbox: Inbox[M], message: M): Inbox[M]
  def equivalent[T](data: Data[T], left: T, right: T): Boolean
  def compareData[T](data: Data[T], left: T, right: T): Int
  def checkDurability[S](descriptor: FieldDurability[S]): Unit
  def resetDurability[S](descriptor: FieldDurability[S], state: S): S
  def durabilityReplacement[S](
      descriptor: FieldDurability[S],
      state: S
  ): Option[(List[String], Any)]
  def inputs: InputEvaluation
  def stickyHolds[S, O, F](
      program: Programs.Sticky[S, O, F],
      broken: Boolean,
      before: S,
      step: Step[S, O, F]
  ): Boolean
  def invoke[A, B](function: A => B, argument: A): B
  def invoke[A, B, C](function: (A, B) => C, first: A, second: B): C
  def monitorNext[M](
      next: M => M,
      current: M,
      before: State,
      step: Step[State, ?, ?]
  ): M
  def contents[M](channel: Channel[M]): Finite[Inbox[M]]
  def inboxCompare[M](
      message: Data.Structural[M],
      left: Inbox[M],
      right: Inbox[M]
  ): Int
  def sameStep[O, F](
      vars: Vars,
      predicate: Step[State, O, F] => Boolean
  ): Step[State, O, F] => Boolean
  def transition[O, F](
      vars: Vars,
      predicate: (State, Step[State, O, F]) => Boolean
  ): (State, Step[State, O, F]) => Boolean
  def compare[T](program: Data.Native[T], left: T, right: T): Int
  def catalog[T](
      mirror: scala.deriving.Mirror.Of[T],
      fields: List[Finite[?]]
  ): IndexedSeq[T]
  def optional[A](domain: Finite[A]): IndexedSeq[Option[A]]
  def range(high: Int): IndexedSeq[Int]
  def composition[F](model: Composition[F]): CompositionExecution[F]
  def machineEffects[S, O, F](
      model: MachineDeclaration[S, O, F]
  ): Vector[(ActionDecl, (S, Class) => List[Step[S, O, F]])]
  def section(owner: AnyRef, name: String): Vector[(String, Any)]
  def classesOf(decl: ActionDecl): List[Class]
  def selects(target: Any, cls: Class): Boolean
  // The callables the DSL builds (Programs): an effect on a state and a class's inputs, a guard
  // that reads its action's input, and any other on its arguments.
  def evaluateEffect[S, O, F](
      program: InputEffect[S, O, F],
      state: S,
      values: List[Any]
  ): List[Step[S, O, F]]
  def effectOn[S, O, F](
      program: InputEffect[S, O, F],
      state: S,
      cls: Class
  ): List[Step[S, O, F]]
  def holds[S](program: InputGuard[S], state: S, cls: Class): Boolean
  def call[R](program: AnyRef, argument: Any): R
  def call[R](program: AnyRef, first: Any, second: Any): R

  // The var snapshots of a Machine (Vars): its initial state, an evaluation on a state restored
  // into its vars, and the steps of an effect's run with its statements.
  def initial(vars: Vars): State
  def at[A](vars: Vars, state: Any, evaluate: () => A): A
  def steps[O, F](
      vars: Vars,
      state: Any,
      accepted: O,
      fires: Boolean,
      body: () => Unit
  ): List[Step[State, O, F]]
  def refuse(vars: Vars, outcome: Any): Unit
  def add(vars: Vars, recorded: Seq[Any]): Unit
  def fired(): Unit
  def choose(alternatives: Vector[Alternative]): Unit

  // The claim statements (NativeClaims).
  def assertion(
      condition: () => Boolean,
      member: String,
      line: Int,
      text: String
  ): Unit
  def recordFact(
      value: () => Any,
      member: String,
      line: Int,
      claim: Boolean
  ): Unit
  def alternatives(member: String, line: Int, body: () => Unit): Unit
  def alternative(body: () => Unit): Unit
  def abandon(member: String, reached: HoleReached): Unit
  def targeting(
      target: Any,
      member: String,
      location: String,
      block: () => Unit
  ): Unit
  def start(fallback: () => State): State
  def answer(fallback: () => Any): Any
  def facts: List[Any]
  def scoped(body: () => Unit): Unit
  def watching[A](before: Any, step: Step[?, ?, ?], body: () => A): A
  def property(machine: Model, section: String, member: String): Property

private[framework] object NativeRuntime:
  private val implementation = "framework.checker.Native$"

  // The runtime, where the class path has it.
  lazy val loaded: Option[NativeRuntime] =
    try
      Some(
        java.lang.Class
          .forName(implementation, true, getClass.getClassLoader)
          .getField("MODULE$")
          .get(null) // scalafix:ok DisableSyntax.null
          .asInstanceOf[NativeRuntime] // scalafix:ok DisableSyntax.asInstanceOf
      )
    catch case _: ClassNotFoundException => None

  // The runtime an evaluation needs.
  def installed: NativeRuntime = loaded.getOrElse(
    throw IllegalStateException(
      "native execution requires the product checker module (model/framework/checker) on the class path"
    )
  )
