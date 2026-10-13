package framework

import framework.Finite.given
// The block forms run as Scala: an `effect { }` block yields the steps of its method form, and an
// `is { }` block answers as its predicate. Field assignments, `record` and the one-argument
// `reject` compile only inside an effect block.

// A dotted file name holds no top-level definitions, so the fixture Model sits in an object.
object EffectsFixture:
  enum Phase:
    case idle, busy, paused

  final case class Job(phase: Phase, tries: UpTo[2])

  enum Said:
    case ok, notFound

  enum Note:
    case started, retried, paused

  given Ok[Said] = Ok(Said.ok)

  // The accessors the blocks read and assign the fields of a `Job` by.
  def phase(using v: View[Job]): Phase = v.get(_.phase)
  def phase_=(p: Phase)(using d: Draft[Job, ?, ?]): Unit = d.set(_.copy(phase = p))
  def tries(using v: View[Job]): UpTo[2] = v.get(_.tries)
  def tries_=(t: UpTo[2])(using d: Draft[Job, ?, ?]): Unit = d.set(_.copy(tries = t))

  object hand extends Actor:
    val press = action(this)

  object Subject extends Machine[Job, Said, Note], Phased[Job, Phase](_.phase):
    val init = Job(Phase.idle, UpTo(0))
    override def end(s: Job) = true

    object states:
      val busy = is(phase == Phase.busy)
      val retried = is(phase == Phase.busy && tries == UpTo[2](1))

    object effects:
      val start = effect {
        phase = Phase.busy
        record(Note.started)
      }
      val retry = effect { tries = UpTo(1) }
      val restart = effect {
        phase = Phase.busy
        tries = UpTo(2)
        record(Note.started)
        record(Note.retried, Note.paused)
      }
      val pause = effect { phase = Phase.paused }
      val keep = effect {}
      val missing = effect(reject(Said.notFound))

    object rules extends Rules:
      on(hand.press) {
        where(states.busy) ~> effects.pause
        when(Phase.idle) ~> effects.start
      }

  // A phase that declares, on each case, the status it is recorded as. An effect block that
  // assigns it records that status after its own facts.
  enum Status:
    case waiting, working, resting, retried

  enum Stage(val status: Status) extends Recorded[Status]:
    case waiting extends Stage(Status.waiting)
    case working extends Stage(Status.working)
    case resting extends Stage(Status.resting), Suspended

  final case class Task(stage: Stage, attempts: UpTo[2])

  def stage(using v: View[Task]): Stage = v.get(_.stage)
  def stage_=(p: Stage)(using d: Draft[Task, ?, Status]): Unit = d.set(p)(_.copy(stage = p))
  def attempts(using v: View[Task]): UpTo[2] = v.get(_.attempts)
  def attempts_=(a: UpTo[2])(using d: Draft[Task, ?, Status]): Unit = d.set(_.copy(attempts = a))

  object Tasker extends Machine[Task, Said, Status], Phased[Task, Stage](_.stage):
    val init = Task(Stage.waiting, UpTo(0))
    override def end(s: Task) = true

    object effects:
      val work = effect { stage = Stage.working }
      val retry = effect { attempts = UpTo(1) }
      val rest = effect {
        stage = Stage.resting
        record(Status.retried)
      }
      val missing = effect(reject(Said.notFound))

    object rules extends Rules:
      on(hand.press)(when(Stage.waiting) ~> effects.work)

  // The rules and a choose enter a phase by `enters`, through the writer the projection declares.
  object Entering
      extends Machine[Task, Said, Status],
        Phased[Task, Stage](_.stage, (s, p) => s.copy(stage = p)):
    val init = Task(Stage.waiting, UpTo(0))
    override def end(s: Task) = true

    object effects:
      val rested = choice
      val worked = choice
      def restOrWork(s: Task) = choose(rested -> enters(Stage.resting)(s), worked -> work(s))
      val work = effect { stage = Stage.working }

    object rules extends Rules:
      on(hand.press) {
        when(Stage.waiting) ~> enters(Stage.working)
        when(Stage.working) ~> effects.restOrWork
      }

  // A machine whose projection declares no writer, and one whose writer writes another field.
  object Unwritten extends Machine[Job, Said, Note], Phased[Job, Phase](_.phase):
    val init = Job(Phase.idle, UpTo(0))
    override def end(s: Job) = true

    object rules extends Rules:
      on(hand.press)(when(Phase.idle) ~> enters(Phase.busy))

  object Miswritten
      extends Machine[Job, Said, Note],
        Phased[Job, Phase](_.phase, (s, _) => s.copy(tries = UpTo(1))):
    val init = Job(Phase.idle, UpTo(0))
    override def end(s: Job) = true

    object rules extends Rules:
      on(hand.press)(when(Phase.idle) ~> enters(Phase.busy))

class EffectsTest extends munit.FunSuite:
  import EffectsFixture.*
  val _ = () => summon[CanEqual[Phase, Phase]]

  type Effect = Job => List[Step[Job, Said, Note]]

  val states: List[Job] = Finite[Job].values.toList

  def same(block: Effect, method: Effect)(using munit.Location): Unit =
    for s <- states do assertEquals(block(s), method(s), s)

  test("an effect block yields the steps of its method form") {
    import Subject.effects.*
    same(start, s => enter(s.copy(phase = Phase.busy), Note.started))
    // Assigning one field of two keeps the other.
    same(retry, s => enter(s.copy(tries = UpTo(1))))
    // Several `record` calls record their facts in call order, as one `record` of them all.
    same(
      restart,
      s =>
        enter(s.copy(phase = Phase.busy, tries = UpTo(2)), Note.started, Note.retried, Note.paused)
    )
    same(pause, s => enter(s.copy(phase = Phase.paused)))
    same(keep, s => enter(s))
    same(missing, s => reject(Said.notFound, s))
  }

  test("an effect block that assigns a status records it after its own facts") {
    import Tasker.effects.*
    type Steps = Task => List[Step[Task, Said, Status]]
    def same(block: Steps, method: Steps)(using munit.Location): Unit =
      for s <- Finite[Task].values do assertEquals(block(s), method(s), s)
    // From a working task too: assigning the status it has still records it.
    same(work, s => enter(s.copy(stage = Stage.working), Status.working))
    // A block that does not assign the status records none.
    same(retry, s => enter(s.copy(attempts = UpTo(1))))
    same(rest, s => enter(s.copy(stage = Stage.resting), Status.retried, Status.resting))
    // Nor does one that rejects.
    same(missing, s => reject(Said.notFound, s))
  }

  test("an effect block that records the status its assignment records is refused") {
    given Owner[Task, Said, Status] = Owner(Tasker)
    val twice = effect {
      stage = Stage.resting
      record(Status.resting)
    }
    for state <- Finite[Task].values do
      val refused = intercept[IllegalArgumentException](twice(state))
      assert(
        refused.getMessage.contains(
          "an effect block records resting, which its assignment of a status already records: " +
            "drop the record(resting)"
        ),
        refused.getMessage
      )
  }

  test("an is block answers as its predicate") {
    for s <- states do
      assertEquals(Subject.states.busy(s), s.phase == Phase.busy, s)
      assertEquals(
        Subject.states.retried(s),
        s.phase == Phase.busy && s.tries == UpTo[2](1),
        s
      )
  }

  test("a rule binds an effect block and conditions on an is block") {
    val press =
      Subject.bindings.head.function.asInstanceOf[Effect] // scalafix:ok DisableSyntax.asInstanceOf
    assertEquals(
      press(Subject.init),
      List(Step(Said.ok, Job(Phase.busy, UpTo(0)), List(Note.started)))
    )
    assertEquals(press(Job(Phase.busy, UpTo(1))).map(_.state.phase), List(Phase.paused))
    assertEquals(press(Job(Phase.paused, UpTo(0))), Nil)
    for state <- states do
      val expected = state.phase match
        case Phase.idle   => Subject.effects.start(state)
        case Phase.busy   => Subject.effects.pause(state)
        case Phase.paused => Nil
      assertEquals(press(state), expected, state)
  }

  test("record, the one-argument reject and a field assignment compile only in an effect block") {
    given Owner[Job, Said, Note] = Owner(Subject)
    // The blocks compile here, so the errors below are the draft's.
    assert(is(phase == Phase.idle)(Subject.init))
    val draftOnly = "which only an `effect { }` block does"
    for (code, errors) <- List(
        "record(Note.started)" -> compileErrors("record(Note.started)"),
        "reject(Said.notFound)" -> compileErrors("reject(Said.notFound)"),
        "phase = Phase.busy" -> compileErrors("phase = Phase.busy"),
        "is { record }" -> compileErrors("is { record(Note.started); true }"),
        "is { reject }" -> compileErrors("is { reject(Said.notFound); true }"),
        "is { phase = }" -> compileErrors("is { phase = Phase.busy; true }")
      )
    do assert(errors.contains(draftOnly), s"$code: $errors")
  }

  test("enters takes the step its phase-only effect block takes, in a rule and in a choose") {
    type Steps = Task => List[Step[Task, Said, Status]]
    val press =
      Entering.bindings.head.function.asInstanceOf[Steps] // scalafix:ok DisableSyntax.asInstanceOf
    for s <- Finite[Task].values do
      val expected: List[Step[Task, Said, Status]] = s.stage match
        case Stage.waiting =>
          List(Step(Said.ok, s.copy(stage = Stage.working), List(Status.working)))
        case Stage.working =>
          List(
            Step(Said.ok, s.copy(stage = Stage.resting), List(Status.resting)),
            Step(Said.ok, s.copy(stage = Stage.working), List(Status.working))
          )
        case Stage.resting => Nil
      assertEquals(press(s), expected, s)
    // A phase whose enum declares no status records none.
    given Owner[Job, Said, Note] = Owner(Subject)
    given Phasing[Job, Phase] = Phasing(_.phase, (s, p) => s.copy(phase = p))
    for s <- states do
      assertEquals(
        enters(Phase.paused)(s),
        List(Step(Said.ok, s.copy(phase = Phase.paused), Nil)),
        s
      )
  }

  test("enters refuses a projection with no writer, and a writer that misses the phase") {
    type Steps = Job => List[Step[Job, Said, Note]]
    def press(m: Machine[Job, Said, Note]) =
      m.bindings.head.function.asInstanceOf[Steps] // scalafix:ok DisableSyntax.asInstanceOf
    val unwritten = intercept[IllegalStateException](press(Unwritten)(Unwritten.init))
    assert(
      unwritten.getMessage.contains(
        "enters(busy) writes the phase through the writer the machine's Phased declares, and it " +
          "declares none: `Phased[State, Phase](_.phase, (s, p) => s.copy(phase = p))`"
      ),
      unwritten.getMessage
    )
    val miswritten = intercept[IllegalArgumentException](press(Miswritten)(Miswritten.init))
    assert(
      miswritten.getMessage.contains("whose phase it does not read as busy"),
      miswritten.getMessage
    )
  }
