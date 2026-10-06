package umpire
// The block forms run as Scala: an `effect { }` block yields the steps of its method form, and an
// `is { }` block answers as its predicate. Field assignments, `record` and the one-argument
// `reject` compile only inside an effect block.

// A dotted file name holds no top-level definitions, so the fixture Model sits in an object.
object EffectsFixture:
  enum Phase derives Finite:
    case idle, busy, paused

  final case class Job(phase: Phase, tries: UpTo[2]) derives Finite

  enum Said derives Finite:
    case ok, notFound

  enum Note derives Finite:
    case started, retried, paused

  given Ok[Said] = Ok(Said.ok)

  // The accessors the blocks read and assign the fields of a `Job` by.
  def phase(using v: View[Job]): Phase = v.get(_.phase)
  def phase_=(p: Phase)(using d: Draft[Job, ?, ?]): Unit = d.set(_.copy(phase = p))
  def tries(using v: View[Job]): UpTo[2] = v.get(_.tries)
  def tries_=(t: UpTo[2])(using d: Draft[Job, ?, ?]): Unit = d.set(_.copy(tries = t))

  object hand extends Actor:
    val press = action(this)

  object Worker extends Machine[Job, Said, Note]:
    val init = Job(Phase.idle, UpTo(0))
    def end(s: Job) = true

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

    object rules extends Rules(_.phase):
      on(hand.press) {
        where(states.busy) ~> effects.pause
        in(Phase.idle) ~> effects.start
      }

class EffectsTest extends munit.FunSuite:
  import EffectsFixture.*

  type Effect = Job => List[Step[Job, Said, Note]]

  val states: List[Job] = Finite[Job].values.toList

  def same(block: Effect, method: Effect)(using munit.Location): Unit =
    for s <- states do assertEquals(block(s), method(s), s)

  test("an effect block yields the steps of its method form") {
    import Worker.effects.*
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

  test("an is block answers as its predicate") {
    for s <- states do
      assertEquals(Worker.states.busy(s), s.phase == Phase.busy, s)
      assertEquals(Worker.states.retried(s), s.phase == Phase.busy && s.tries == UpTo[2](1), s)
  }

  test("a rule binds an effect block and conditions on an is block") {
    val press =
      Worker.bindings.head.function.asInstanceOf[Effect] // scalafix:ok DisableSyntax.asInstanceOf
    assertEquals(
      press(Worker.init),
      List(Step(Said.ok, Job(Phase.busy, UpTo(0)), List(Note.started)))
    )
    assertEquals(press(Job(Phase.busy, UpTo(1))).map(_.state.phase), List(Phase.paused))
    assertEquals(press(Job(Phase.paused, UpTo(0))), Nil)
  }

  test("record, the one-argument reject and a field assignment compile only in an effect block") {
    given Owner[Job, Said, Note] = Owner(Worker)
    // The blocks compile here, so the errors below are the draft's.
    assert(is(phase == Phase.idle)(Worker.init))
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
