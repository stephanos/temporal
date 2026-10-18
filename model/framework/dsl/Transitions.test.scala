package framework.dsl

import framework.checker.Effects.*
import framework.dsl.Finite.given
import framework.dsl.outcomes.{Outcome, Rejection}

// Transitions as plain control flow over a Machine's vars: an `on` block is an `if`/`match` chain
// whose leaves call an effect, reject or are `()`, run natively with the state restored into the
// vars, one rule per block.
object TransitionsFixture:
  enum Lit:
    case off, on
    case broken extends Lit, Terminated

  enum Knob:
    case up, down

  enum Said:
    case ok, broken, busy

  enum Note:
    case lit, darkened

  given Ok[Said] = Ok(Said.ok)

  object hand extends Actor:
    val press = action
    val smash = action
    val fix = action
    val set = action.input[Lit](name = "light")
    val turn = action.input[Knob](name = "knob")
    val knock = action
    val tap = action
    val strike = action
    val wave = action

  object fault extends Actor:
    val drop = action.crash

  // A lamp worn out by three presses, which a smash breaks or toggles and a fix repairs while it is
  // unpressed.
  object Lamp extends Machine[Said, Note]:
    var light = Lit.off
    var presses: UpTo[3] = UpTo(0)

    durability(durable(light), ephemeral(presses, UpTo[3](0)))

    object transitions extends Transitions:
      on(hand.press):
        light match
          case Lit.off | Lit.on if presses < 3 => effects.toggle()
          case _: Terminated                   => reject(Said.broken)
          case _                               => ()
      on(hand.smash):
        if light == Lit.on then effects.smash()
        else if light == Lit.off then effects.breakOff()
      // first match: a broken, unpressed lamp is repaired; any other broken one is busy
      on(hand.fix):
        light match
          case Lit.broken if presses == 0 => effects.repair()
          case _: Terminated              => reject(Said.busy)
          case _                          => ()
      on(hand.set): to =>
        if to != light && light != Lit.broken then effects.set(to)
      on(hand.turn(Knob.up)):
        if light == Lit.off then effects.flick()
      on(hand.turn(Knob.down)):
        if light == Lit.on then effects.flick()
      from(hand):
        on(hand.knock, hand.tap):
          if light == Lit.broken then reject(Said.busy) else effects.flick()
      disabled(hand.strike)
      on(hand.wave):
        effects.flick()
      crashes(fault.drop, Said.ok)

    object effects:
      val smashed = choice
      val toggled = choice

      def toggle() =
        presses = UpTo(presses + 1)
        if light == Lit.on then
          light = Lit.off
          record(Note.darkened)
        else
          light = Lit.on
          record(Note.lit)
      def smash() =
        choose(smashed -> { light = Lit.broken }, toggled -> { if presses < 3 then toggle() })
      def breakOff() =
        light = Lit.broken
      def repair() =
        light = Lit.off
      def set(to: Lit) =
        light = to
      def flick() =
        light = if light == Lit.on then Lit.off else Lit.on

  // The spec's retry phase table (fn-190, API Contracts), as a `match` over a phase var and in the
  // hidden `where` form over a phase record, with the same effects.
  enum Phase:
    case unstarted, scheduled, started, paused
    case cancelRequested, pauseRequested, resetRequested, resetKeepingPause
    case completed extends Phase, Succeeded
    case canceled extends Phase, Canceled
    case failed extends Phase, Failed

  enum Failure:
    case retryable, fatal

  object activityWorker extends Actor:
    val respondFailed = action.input[Failure](name = "failure")

  object completionClient extends Actor:
    val respondFailedByID = action

  object RetryVars extends Machine[Outcome, Nothing]:
    var phase = Phase.unstarted
    var retries: UpTo[2] = UpTo(2)

    def retriesRemaining: Boolean = retries > 0
    def exhausted: Boolean = retries == 0

    object transitions extends Transitions:
      on(activityWorker.respondFailed(Failure.retryable)):
        phase match
          case Phase.started if retriesRemaining                 => effects.backOff()
          case Phase.cancelRequested                             => effects.cancel()
          case Phase.pauseRequested if retriesRemaining          => effects.backOffPaused()
          case Phase.started | Phase.pauseRequested if exhausted => effects.fail()
          case _                                                 => ()
      on(completionClient.respondFailedByID):
        phase match
          case Phase.unstarted | Phase.scheduled | Phase.paused => reject(Rejection.notFound)
          case _: Closed                                        => reject(Rejection.notFound)
          case Phase.resetRequested | Phase.resetKeepingPause   => effects.applyReset()
          case _                                                => ()

    object effects:
      def backOff() =
        phase = Phase.scheduled
        retries = UpTo(retries - 1)
      def cancel() =
        phase = Phase.canceled
      def backOffPaused() =
        phase = Phase.paused
        retries = UpTo(retries - 1)
      def fail() =
        phase = Phase.failed
      def applyReset() =
        phase = Phase.scheduled
        retries = UpTo(2)

  final case class Retry(phase: Phase, retries: UpTo[2])

  object RetryRecord extends RecordMachine[Retry, Outcome, Nothing]:
    val init = Retry(Phase.unstarted, UpTo(2))
    def end(s: Retry) = s.phase.in[Closed]

    object states:
      def retriesRemaining(s: Retry): Boolean = s.retries > 0
      def exhausted(s: Retry): Boolean = s.retries == 0

    object effects:
      def backOff(s: Retry) =
        enter[Retry, Outcome, Nothing](
          s.copy(phase = Phase.scheduled, retries = UpTo(s.retries - 1))
        )
      def cancel(s: Retry) = enter[Retry, Outcome, Nothing](s.copy(phase = Phase.canceled))
      def backOffPaused(s: Retry) =
        enter[Retry, Outcome, Nothing](s.copy(phase = Phase.paused, retries = UpTo(s.retries - 1)))
      def fail(s: Retry) = enter[Retry, Outcome, Nothing](s.copy(phase = Phase.failed))
      def applyReset(s: Retry) =
        enter[Retry, Outcome, Nothing](s.copy(phase = Phase.scheduled, retries = UpTo(2)))

    object transitions extends Rules:
      on(activityWorker.respondFailed(Failure.retryable)):
        where(_.phase == Phase.started).where(states.retriesRemaining) ~> effects.backOff
        where(_.phase == Phase.cancelRequested) ~> effects.cancel
        where(_.phase == Phase.pauseRequested).where(states.retriesRemaining) ~>
          effects.backOffPaused
        where(_.phase.in(Phase.started, Phase.pauseRequested)).where(states.exhausted) ~>
          effects.fail
      on(completionClient.respondFailedByID):
        where(_.phase.in(Phase.unstarted, Phase.scheduled, Phase.paused)) ~>
          rejects(Rejection.notFound)
        where(_.phase.in[Closed]) ~> rejects(Rejection.notFound)
        where(_.phase.in(Phase.resetRequested, Phase.resetKeepingPause)) ~> effects.applyReset

class TransitionsTest extends munit.FunSuite:
  import TransitionsFixture.*

  private def run[S, O, F](machine: RecordMachine[S, O, F], state: S, cls: Class) =
    val binding = machine.bindings.find(_.decl eq cls.decl).get
    effectOf[S, O, F](binding.decl, binding.function)(state, cls)

  private val off = Lamp.init

  private def lamp(light: Lit, presses: Int) = off.copy(light = light, presses = UpTo[3](presses))

  private def steps(light: Lit, presses: Int, cls: Class) = run(Lamp, lamp(light, presses), cls)

  test("an if fires its effect where its condition holds, and is disabled where it has no else"):
    assertEquals(steps(Lit.off, 0, hand.set(Lit.on)), List(Step(Said.ok, lamp(Lit.on, 0))))
    assertEquals(steps(Lit.off, 0, hand.set(Lit.off)), Nil)
    assertEquals(steps(Lit.broken, 0, hand.set(Lit.on)), Nil)
    assertEquals(steps(Lit.off, 1, hand.smash()), List(Step(Said.ok, lamp(Lit.broken, 1))))
    assertEquals(steps(Lit.broken, 1, hand.smash()), Nil)

  test("a bare effect call fires in every state"):
    assertEquals(steps(Lit.off, 2, hand.wave()), List(Step(Said.ok, lamp(Lit.on, 2))))
    assertEquals(steps(Lit.broken, 2, hand.wave()), List(Step(Said.ok, lamp(Lit.on, 2))))

  test("a match is a phase table: case a | b with a guard, a role, and case _ => ()"):
    assertEquals(
      steps(Lit.off, 0, hand.press()),
      List(Step(Said.ok, lamp(Lit.on, 1), List(Note.lit)))
    )
    assertEquals(
      steps(Lit.on, 1, hand.press()),
      List(Step(Said.ok, lamp(Lit.off, 2), List(Note.darkened)))
    )
    assertEquals(steps(Lit.on, 3, hand.press()), Nil)
    assertEquals(steps(Lit.broken, 2, hand.press()), List(Step(Said.broken, lamp(Lit.broken, 2))))

  test("the first matching case wins, as in Scala"):
    assertEquals(steps(Lit.broken, 0, hand.fix()), List(Step(Said.ok, lamp(Lit.off, 0))))
    assertEquals(steps(Lit.broken, 2, hand.fix()), List(Step(Said.busy, lamp(Lit.broken, 2))))
    assertEquals(steps(Lit.on, 0, hand.fix()), Nil)

  test("an input is the block's parameter, and a class target fires that class alone"):
    assertEquals(steps(Lit.on, 2, hand.set(Lit.off)), List(Step(Said.ok, lamp(Lit.off, 2))))
    assertEquals(steps(Lit.off, 0, hand.turn(Knob.up)), List(Step(Said.ok, lamp(Lit.on, 0))))
    assertEquals(steps(Lit.on, 0, hand.turn(Knob.up)), Nil)
    assertEquals(steps(Lit.on, 0, hand.turn(Knob.down)), List(Step(Said.ok, lamp(Lit.off, 0))))
    assertEquals(steps(Lit.off, 0, hand.turn(Knob.down)), Nil)

  test("an effect's choose takes one step per alternative, each from the state it started in"):
    assertEquals(
      steps(Lit.on, 1, hand.smash()),
      List(
        Step(Said.ok, lamp(Lit.broken, 1)),
        Step(Said.ok, lamp(Lit.off, 2), List(Note.darkened))
      )
    )
    assertEquals(
      steps(Lit.on, 3, hand.smash()),
      List(Step(Said.ok, lamp(Lit.broken, 3)), Step(Said.ok, lamp(Lit.on, 3)))
    )
    assertEquals((Lamp.light, Lamp.presses: Int), (Lit.off, 0))

  test("a choose outside an effect, and a choice named twice, are refused"):
    val outside = intercept[IllegalStateException](Lamp.effects.smash())
    assert(outside.getMessage.contains("choose is a statement of an effect"), outside.getMessage)
    val twice = intercept[IllegalArgumentException](
      choose(Lamp.effects.smashed -> (), Lamp.effects.smashed -> ())
    )
    assert(
      twice.getMessage.contains("a choice names one alternative of a choose"),
      twice.getMessage
    )

  test("one block fans out to each of its targets, from groups, and disabled binds no step"):
    for target <- List(hand.knock(), hand.tap()) do
      assertEquals(steps(Lit.off, 1, target), List(Step(Said.ok, lamp(Lit.on, 1))))
      assertEquals(steps(Lit.broken, 1, target), List(Step(Said.busy, lamp(Lit.broken, 1))))
    assertEquals(steps(Lit.on, 1, hand.strike()), Nil)
    val table = Lamp.transitions.table.toMap
    assert(table(hand.strike.decl).isInstanceOf[Bound.Disabled[?, ?, ?]])

  test("a generated crash keeps the durable vars and resets the ephemeral ones"):
    assertEquals(steps(Lit.on, 2, fault.drop()), List(Step(Said.ok, lamp(Lit.on, 0))))

  test("each block is one rule, in declaration order"):
    val table = Lamp.transitions.table
    assertEquals(
      table.map((decl, _) => Lamp.transitions.actionName(decl)),
      Vector("press", "smash", "fix", "set", "turn", "knock", "tap", "strike", "wave", "drop")
    )
    val bound = table.toMap
    bound(hand.press.decl) match
      case Bound.Ruled(rules) => assertEquals(rules.map(_.heading), Vector("on"))
      case other              => fail(s"press is bound to $other")
    bound(hand.turn.decl) match
      case Bound.Ruled(rules) =>
        assertEquals(rules.map(_.values), Vector(Some(List(Knob.up)), Some(List(Knob.down))))
      case other => fail(s"turn is bound to $other")

  test("the retry phase table as a match gives the steps of its where form, in every state"):
    val classes =
      inputClassesOf(activityWorker.respondFailed.decl) ++
        inputClassesOf(completionClient.respondFailedByID.decl)
    val compared = for
      phase <- Phase.values.toList
      retries <- 0 to 2
      cls <- classes
    yield
      val start = RetryVars.init.copy(phase = phase, retries = UpTo[2](retries))
      val matched = run(RetryVars, start, cls).map: step =>
        (
          step.outcome,
          step.state.selectDynamic("phase"),
          step.state.selectDynamic("retries"),
          step.facts
        )
      val phased = run(RetryRecord, Retry(phase, UpTo(retries)), cls).map: step =>
        (step.outcome, step.state.phase: Any, step.state.retries: Any, step.facts)
      assertEquals(matched, phased, s"$phase, $retries retries, $cls")
      matched
    // the retryable failure fires in 9 of the 27 states, the failure by id in 24 of them
    assertEquals(compared.count(_.nonEmpty), 33)
    assert(compared.flatten.map(_._1).contains(Outcome.rejected(Rejection.notFound)))

  test("an assignment to a var in a transition is refused"):
    val assigned = compileErrors(
      "object M extends Machine[Outcome, Nothing]:\n  var x = 1\n  object transitions extends Transitions:\n    on(TransitionsFixture.hand.press):\n      if x == 1 then x = 2\n"
    )
    assert(
      assigned.contains("a transition says when; write the state change in effects"),
      assigned
    )

  test("a transition leaf that is no effect call, rejection or () is refused"):
    val leaf = compileErrors(
      "object M extends Machine[Outcome, Nothing]:\n  var x = 1\n  def helper(): Unit = ()\n  object transitions extends Transitions:\n    on(TransitionsFixture.hand.press):\n      if x == 1 then helper()\n"
    )
    assert(
      leaf.contains(
        "a transition ends in an effect call, a rejection or (): helper() is none of these"
      ),
      leaf
    )
