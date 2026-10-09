package umpire
// What the standalone activity Model's effects do, run as Scala.

import temporal.features.activity.{failure, Failure}
import temporal.features.activity.standalone.{activity, client, service, system, worker, ResetPause}
import temporal.features.activity.{deadline, timers, Timeout, TimeoutType}
import temporal.features.activity.standalone.product
import temporal.actors.worker.worker as process
import umpire.outcomes.Outcome
import system.*

class StandaloneActivityPins extends munit.FunSuite:
  type Loss =
    AdmissionResponseState => List[Step[AdmissionResponseState, Outcome, AdmissionResponseFact]]

  final case class ExpectedStep[S, F](
      outcome: String,
      state: S,
      facts: List[F] = Nil,
      because: String = ""
  )

  final case class ExpectedRule[S, F](
      decl: ActionDecl,
      inputs: Option[List[Any]],
      guard: S => Boolean,
      run: S => List[ExpectedStep[S, F]]
  )

  private val nominalTimeWindow =
    "The configured time window is nominal, not proof that a real timer fired; the rule's guards select its applicability. Scheduling deadlines start after start delay; schedule-to-start also waits through retry backoff (chasm/lib/activity/model/model.go:300-310,312-335,355-376)."

  private def accepted[S, F](state: S, facts: F*): List[ExpectedStep[S, F]] =
    List(ExpectedStep("accepted", state, facts.toList))

  private def acceptedBecause[S, F](
      state: S,
      because: String,
      facts: F*
  ): List[ExpectedStep[S, F]] =
    List(ExpectedStep("accepted", state, facts.toList, because))

  private def rejectedNotFound[S, F](state: S): List[ExpectedStep[S, F]] =
    List(ExpectedStep("rejected(notFound)", state))

  private def rejectedBecause[S, F](
      state: S,
      why: String,
      because: String
  ): List[ExpectedStep[S, F]] =
    List(ExpectedStep(s"rejected($why)", state, because = because))

  private def observed[S, O, F](steps: List[Step[S, O, F]]): List[ExpectedStep[S, F]] =
    steps.map(s => ExpectedStep(s.outcome.toString, s.state, s.facts, s.because))

  private def run[S, O, F](
      binding: StepBinding[S, O, F],
      state: S,
      inputs: List[Any]
  ): List[Step[S, O, F]] =
    // scalafix:off DisableSyntax.asInstanceOf
    inputs match
      case Nil       => binding.function.asInstanceOf[S => List[Step[S, O, F]]](state)
      case List(one) =>
        binding.function.asInstanceOf[(S, Any) => List[Step[S, O, F]]](state, one)
      case List(one, two) =>
        binding.function.asInstanceOf[(S, Any, Any) => List[Step[S, O, F]]](state, one, two)
      case List(one, two, three) =>
        binding.function
          .asInstanceOf[(S, Any, Any, Any) => List[Step[S, O, F]]](state, one, two, three)
      case List(one, two, three, four) =>
        binding.function
          .asInstanceOf[(S, Any, Any, Any, Any) => List[Step[S, O, F]]](
            state,
            one,
            two,
            three,
            four
          )
      case List(one, two, three, four, five) =>
        binding.function
          .asInstanceOf[(S, Any, Any, Any, Any, Any) => List[Step[S, O, F]]](
            state,
            one,
            two,
            three,
            four,
            five
          )
      case List(one, two, three, four, five, six) =>
        binding.function
          .asInstanceOf[(S, Any, Any, Any, Any, Any, Any) => List[Step[S, O, F]]](
            state,
            one,
            two,
            three,
            four,
            five,
            six
          )
      case other => fail(s"the pinned activity action has ${other.size} inputs")
    // scalafix:on DisableSyntax.asInstanceOf

  private def assertStepTable[S, O, F](
      machine: Machine[S, O, F],
      rules: List[ExpectedRule[S, F]]
  )(using states: Finite[S]): Unit =
    for
      binding <- machine.bindings
      inputs <- classesOf(binding.decl)
      state <- states.values
    do
      val matches =
        rules.filter(r => r.decl == binding.decl && r.inputs.forall(_ == inputs) && r.guard(state))
      assert(
        matches.sizeIs <= 1,
        s"several pinned rows match ${binding.decl.name}$inputs in $state"
      )
      val expected = matches.headOption.fold(List.empty[ExpectedStep[S, F]])(_.run(state))
      assertEquals(
        observed(run(binding, state, inputs)),
        expected,
        s"${machine.name}: ${binding.decl.name}$inputs in $state"
      )

  private val productRules: List[ExpectedRule[product.State, product.Fact]] =
    import product.{Fact, Phase, State}
    import Phase.*
    List[ExpectedRule[State, Fact]](
      ExpectedRule(
        client.start.decl,
        None,
        _.phase == scheduled,
        s => accepted(s, Fact.statusScheduled)
      ),
      ExpectedRule(
        worker.poll.decl,
        None,
        _.phase == scheduled,
        s => accepted(s.copy(phase = started), Fact.statusStarted)
      ),
      ExpectedRule(
        worker.respondCompleted.decl,
        None,
        s => s.phase == started || s.phase == cancelRequested,
        s => accepted(s.copy(phase = completed), Fact.statusCompleted)
      ),
      ExpectedRule(
        worker.respondFailed.decl,
        Some(List(Failure.fatal)),
        _.phase == started,
        s =>
          accepted(s.copy(phase = failed), Fact.statusFailed) ++
            accepted(s.copy(phase = scheduled), Fact.statusScheduled) ++
            accepted(s.copy(phase = paused), Fact.statusPaused)
      ),
      ExpectedRule(
        worker.respondFailed.decl,
        Some(List(Failure.fatal)),
        _.phase == cancelRequested,
        s => accepted(s.copy(phase = failed), Fact.statusFailed)
      ),
      ExpectedRule(
        worker.respondFailed.decl,
        Some(List(Failure.retryable)),
        _.phase == started,
        s =>
          accepted(s.copy(phase = scheduled), Fact.statusScheduled) ++
            accepted(s.copy(phase = paused), Fact.statusPaused) ++
            accepted(s.copy(phase = failed), Fact.statusFailed)
      ),
      ExpectedRule(
        worker.respondFailed.decl,
        Some(List(Failure.retryable)),
        _.phase == cancelRequested,
        s => accepted(s.copy(phase = canceled), Fact.statusCanceled)
      ),
      ExpectedRule(
        worker.respondCanceled.decl,
        None,
        _.phase == cancelRequested,
        s => accepted(s.copy(phase = canceled), Fact.statusCanceled)
      ),
      ExpectedRule(
        worker.respondCanceled.decl,
        None,
        _.phase == started,
        s =>
          rejectedBecause(
            s,
            "invalidArgument",
            "cancellation was not requested (chasm/lib/activity/model/model.go:171)"
          )
      ),
      ExpectedRule(
        client.pause.decl,
        None,
        s => closedProduct(s.phase),
        rejectedNotFound
      ),
      ExpectedRule(
        client.unpause.decl,
        None,
        s => closedProduct(s.phase),
        rejectedNotFound
      ),
      ExpectedRule(
        client.requestCancel.decl,
        None,
        s => closedProduct(s.phase),
        rejectedNotFound
      ),
      ExpectedRule(
        client.terminate.decl,
        None,
        s => closedProduct(s.phase),
        rejectedNotFound
      ),
      ExpectedRule(
        client.pause.decl,
        None,
        _.phase == scheduled,
        s => accepted(s.copy(phase = paused), Fact.statusPaused)
      ),
      ExpectedRule(
        client.pause.decl,
        None,
        _.phase == started,
        s =>
          accepted(s.copy(phase = paused), Fact.statusPaused) ++
            accepted(s, Fact.statusPaused) ++
            rejectedBecause(
              s,
              "failedPrecondition",
              "pause already requested (chasm/lib/activity/model/model.go:232)"
            )
      ),
      ExpectedRule(
        client.pause.decl,
        None,
        s => s.phase == paused || s.phase == cancelRequested,
        s =>
          rejectedBecause(
            s,
            "failedPrecondition",
            "already paused or cancellation pending (chasm/lib/activity/model/model.go:232)"
          )
      ),
      ExpectedRule(
        client.unpause.decl,
        None,
        _.phase == paused,
        s => accepted(s.copy(phase = scheduled), Fact.statusScheduled)
      ),
      ExpectedRule(
        client.unpause.decl,
        None,
        _.phase == started,
        s =>
          accepted(s, Fact.statusStarted) ++
            rejectedBecause(
              s,
              "failedPrecondition",
              "activity is not paused (chasm/lib/activity/model/model.go:251)"
            )
      ),
      ExpectedRule(
        client.unpause.decl,
        None,
        s => s.phase == scheduled || s.phase == cancelRequested,
        s =>
          rejectedBecause(
            s,
            "failedPrecondition",
            "activity is not paused (chasm/lib/activity/model/model.go:251)"
          )
      ),
      ExpectedRule(
        client.requestCancel.decl,
        None,
        s => s.phase == scheduled || s.phase == paused,
        s => accepted(s.copy(phase = canceled), Fact.statusCanceled)
      ),
      ExpectedRule(
        client.requestCancel.decl,
        None,
        _.phase == started,
        s => accepted(s.copy(phase = cancelRequested), Fact.statusCancelRequested)
      ),
      ExpectedRule(
        client.requestCancel.decl,
        None,
        _.phase == cancelRequested,
        s =>
          rejectedBecause(
            s,
            "failedPrecondition",
            "cancellation already requested (chasm/lib/activity/model/model.go:201-202)"
          )
      ),
      ExpectedRule(
        client.terminate.decl,
        None,
        s => productLive(s.phase),
        s => accepted(s.copy(phase = terminated), Fact.statusTerminated)
      ),
      ExpectedRule(
        timers.timeout.decl,
        None,
        s => productLive(s.phase),
        s => accepted(s.copy(phase = timedOut), Fact.statusTimedOut)
      )
    ) ++ List[ExpectedRule[product.State, product.Fact]](
      ExpectedRule(
        worker.heartbeat.decl,
        None,
        s => s.phase == started || s.phase == cancelRequested,
        s => accepted(s, Fact.heartbeatReceived)
      ),
      ExpectedRule(
        worker.heartbeat.decl,
        None,
        s => s.phase != started && s.phase != cancelRequested,
        rejectedNotFound
      ),
      ExpectedRule(
        deadline.heartbeat.decl,
        None,
        _.phase == started,
        s =>
          accepted(s.copy(phase = scheduled), Fact.statusScheduled, Fact.heartbeatTimedOut) ++
            accepted(s.copy(phase = paused), Fact.statusPaused, Fact.heartbeatTimedOut) ++
            accepted(s.copy(phase = timedOut), Fact.statusTimedOut, Fact.heartbeatTimedOut)
      ),
      ExpectedRule(
        deadline.heartbeat.decl,
        None,
        _.phase == cancelRequested,
        s => accepted(s.copy(phase = timedOut), Fact.statusTimedOut, Fact.heartbeatTimedOut)
      )
    ) ++ productServiceRules ++ productResetRules

  private def productResetRules: List[ExpectedRule[product.State, product.Fact]] =
    import product.{Fact, Phase, State}
    import Phase.*
    val pending = "a reset is already pending (operator_commands.go:460-464)"
    List[ExpectedRule[State, Fact]](
      ExpectedRule(client.reset.decl, None, s => closedProduct(s.phase), rejectedNotFound),
      ExpectedRule(
        client.reset.decl,
        None,
        _.phase == scheduled,
        s => accepted(s, Fact.statusScheduled)
      ),
      ExpectedRule(
        client.reset.decl,
        None,
        _.phase == cancelRequested,
        s =>
          rejectedBecause(
            s,
            "failedPrecondition",
            "cannot reset an activity with a pending cancellation (operator_commands.go:458)"
          )
      ),
      ExpectedRule(
        client.reset.decl,
        Some(List(ResetPause.resume)),
        _.phase == paused,
        s => accepted(s.copy(phase = scheduled), Fact.statusScheduled)
      ),
      ExpectedRule(
        client.reset.decl,
        Some(List(ResetPause.resume)),
        _.phase == started,
        s => accepted(s, Fact.statusStarted) ++ rejectedBecause(s, "failedPrecondition", pending)
      ),
      ExpectedRule(
        client.reset.decl,
        Some(List(ResetPause.keepPaused)),
        _.phase == paused,
        s => accepted(s, Fact.statusPaused)
      ),
      ExpectedRule(
        client.reset.decl,
        Some(List(ResetPause.keepPaused)),
        _.phase == started,
        s =>
          accepted(s, Fact.statusStarted) ++ accepted(s, Fact.statusPaused) ++
            rejectedBecause(s, "failedPrecondition", pending)
      )
    )

  private def productServiceRules: List[ExpectedRule[product.State, product.Fact]] =
    import product.{Fact, Phase, State}
    import Phase.*
    val closed =
      List(service.respondCompletedByID, service.respondFailedByID, service.respondCanceledByID)
        .map(a =>
          ExpectedRule[State, Fact](a.decl, None, s => closedProduct(s.phase), rejectedNotFound)
        )
    val unheld = List(service.respondFailedByID, service.respondCanceledByID)
      .map(a =>
        ExpectedRule[State, Fact](
          a.decl,
          None,
          s => s.phase == scheduled || s.phase == paused,
          rejectedNotFound
        )
      )
    closed ++ unheld ++ List[ExpectedRule[State, Fact]](
      ExpectedRule(
        service.respondCompletedByID.decl,
        None,
        s => productLive(s.phase),
        s => accepted(s.copy(phase = completed), Fact.statusCompleted)
      ),
      ExpectedRule(
        service.respondFailedByID.decl,
        Some(List(Failure.fatal)),
        _.phase == started,
        s =>
          accepted(s.copy(phase = failed), Fact.statusFailed) ++
            accepted(s.copy(phase = scheduled), Fact.statusScheduled) ++
            accepted(s.copy(phase = paused), Fact.statusPaused)
      ),
      ExpectedRule(
        service.respondFailedByID.decl,
        Some(List(Failure.fatal)),
        _.phase == cancelRequested,
        s => accepted(s.copy(phase = failed), Fact.statusFailed)
      ),
      ExpectedRule(
        service.respondFailedByID.decl,
        Some(List(Failure.retryable)),
        _.phase == started,
        s =>
          accepted(s.copy(phase = scheduled), Fact.statusScheduled) ++
            accepted(s.copy(phase = paused), Fact.statusPaused) ++
            accepted(s.copy(phase = failed), Fact.statusFailed)
      ),
      ExpectedRule(
        service.respondFailedByID.decl,
        Some(List(Failure.retryable)),
        _.phase == cancelRequested,
        s => accepted(s.copy(phase = canceled), Fact.statusCanceled)
      ),
      ExpectedRule(
        service.respondCanceledByID.decl,
        None,
        _.phase == cancelRequested,
        s => accepted(s.copy(phase = canceled), Fact.statusCanceled)
      ),
      ExpectedRule(
        service.respondCanceledByID.decl,
        None,
        _.phase == started,
        s => rejectedBecause(s, "invalidArgument", "cancellation was not requested")
      )
    )

  private def closedProduct(p: product.Phase): Boolean =
    import product.Phase.*
    p == completed || p == failed || p == canceled || p == terminated || p == timedOut

  private def productLive(p: product.Phase): Boolean =
    import product.Phase.*
    p == scheduled || p == started || p == paused || p == cancelRequested

  private lazy val systemRules: List[ExpectedRule[system.State, system.Fact]] =
    import system.{Fact, Phase, State}
    import Phase.*
    List[ExpectedRule[State, Fact]](
      ExpectedRule(
        worker.poll.decl,
        None,
        s => s.phase == scheduled && s.dispatch == Dispatch.now,
        s =>
          accepted(
            s.copy(phase = started, attempts = UpTo[2](((s.attempts: Int) + 1).min(2))),
            Fact.statusStarted,
            Fact.attemptCount
          )
      ),
      ExpectedRule(
        worker.respondCompleted.decl,
        None,
        s => heldSystem(s.phase) || resetSystem(s.phase),
        s => accepted(s.copy(phase = completed), Fact.statusCompleted)
      ),
      ExpectedRule(
        worker.respondFailed.decl,
        Some(List(Failure.fatal)),
        s => heldSystem(s.phase),
        s => accepted(s.copy(phase = failed), Fact.statusFailed)
      ),
      ExpectedRule(
        worker.respondFailed.decl,
        Some(List(Failure.retryable)),
        s => s.phase == started && retriesRemain(s),
        s =>
          acceptedBecause(
            s.copy(phase = scheduled, dispatch = Dispatch.backoff),
            "a retryable attempt backs off; the client reads scheduled again",
            Fact.statusScheduled,
            Fact.attemptCount
          )
      ),
      ExpectedRule(
        worker.respondFailed.decl,
        Some(List(Failure.retryable)),
        _.phase == cancelRequested,
        s => accepted(s.copy(phase = canceled), Fact.statusCanceled)
      ),
      ExpectedRule(
        worker.respondFailed.decl,
        Some(List(Failure.retryable)),
        s => s.phase == pauseRequested && retriesRemain(s),
        s =>
          accepted(
            s.copy(phase = paused, dispatch = Dispatch.backoff),
            Fact.statusPaused,
            Fact.attemptCount
          )
      ),
      ExpectedRule(
        worker.respondFailed.decl,
        Some(List(Failure.retryable)),
        s => (s.phase == started || s.phase == pauseRequested) && !retriesRemain(s),
        s => accepted(s.copy(phase = failed), Fact.statusFailed)
      ),
      ExpectedRule(
        worker.respondCanceled.decl,
        None,
        _.phase == cancelRequested,
        s => accepted(s.copy(phase = canceled), Fact.statusCanceled)
      ),
      ExpectedRule(
        worker.respondCanceled.decl,
        None,
        s => s.phase == started || s.phase == pauseRequested || resetSystem(s.phase),
        s =>
          rejectedBecause(
            s,
            "invalidArgument",
            "cancellation was not requested (chasm/lib/activity/model/model.go:171)"
          )
      ),
      ExpectedRule(client.pause.decl, None, s => closedSystem(s.phase), rejectedNotFound),
      ExpectedRule(client.unpause.decl, None, s => closedSystem(s.phase), rejectedNotFound),
      ExpectedRule(client.requestCancel.decl, None, s => closedSystem(s.phase), rejectedNotFound),
      ExpectedRule(client.terminate.decl, None, s => closedSystem(s.phase), rejectedNotFound),
      ExpectedRule(
        client.pause.decl,
        None,
        _.phase == scheduled,
        s => accepted(s.copy(phase = paused), Fact.statusPaused)
      ),
      ExpectedRule(
        client.pause.decl,
        None,
        _.phase == started,
        s =>
          acceptedBecause(
            s.copy(phase = pauseRequested),
            "the worker learns of the pause on its next heartbeat",
            Fact.statusPaused
          )
      ),
      ExpectedRule(
        client.pause.decl,
        None,
        s =>
          s.phase == paused || s.phase == pauseRequested || s.phase == cancelRequested ||
            resetSystem(s.phase),
        s =>
          rejectedBecause(
            s,
            "failedPrecondition",
            "already paused, or a cancellation or reset pending (chasm/lib/activity/model/model.go:232)"
          )
      ),
      ExpectedRule(
        client.unpause.decl,
        None,
        _.phase == paused,
        s => accepted(s.copy(phase = scheduled), Fact.statusScheduled)
      ),
      ExpectedRule(
        client.unpause.decl,
        None,
        _.phase == pauseRequested,
        s => accepted(s.copy(phase = started), Fact.statusStarted)
      ),
      ExpectedRule(
        client.unpause.decl,
        None,
        s =>
          s.phase == scheduled || s.phase == started || s.phase == cancelRequested ||
            resetSystem(s.phase),
        s =>
          rejectedBecause(
            s,
            "failedPrecondition",
            "activity is not paused (chasm/lib/activity/model/model.go:251)"
          )
      ),
      ExpectedRule(
        client.requestCancel.decl,
        None,
        s => s.phase == scheduled || s.phase == paused,
        s => accepted(s.copy(phase = canceled), Fact.statusCanceled)
      ),
      ExpectedRule(
        client.requestCancel.decl,
        None,
        s => s.phase == started || s.phase == pauseRequested || resetSystem(s.phase),
        s => accepted(s.copy(phase = cancelRequested), Fact.statusCancelRequested)
      ),
      ExpectedRule(
        client.requestCancel.decl,
        None,
        _.phase == cancelRequested,
        s =>
          rejectedBecause(
            s,
            "failedPrecondition",
            "cancellation already requested (chasm/lib/activity/model/model.go:201-202)"
          )
      ),
      ExpectedRule(
        client.terminate.decl,
        None,
        s => liveSystem(s.phase),
        s => accepted(s.copy(phase = terminated), Fact.statusTerminated)
      ),
      ExpectedRule(process.stop.decl, None, _ => true, s => accepted(s)),
      ExpectedRule(
        timers.startDelay.decl,
        None,
        s => liveSystem(s.phase) && s.dispatch == Dispatch.startDelay,
        s => acceptedBecause(s.copy(dispatch = Dispatch.now), nominalTimeWindow)
      ),
      ExpectedRule(
        timers.backoff.decl,
        None,
        s => liveSystem(s.phase) && s.dispatch == Dispatch.backoff,
        s => acceptedBecause(s.copy(dispatch = Dispatch.now), nominalTimeWindow)
      ),
      ExpectedRule(
        deadline.scheduleToClose.decl,
        None,
        s =>
          liveSystem(
            s.phase
          ) && s.scheduleToClose == Timeout.expires && s.dispatch != Dispatch.startDelay,
        s =>
          acceptedBecause(
            s.copy(phase = timedOut),
            nominalTimeWindow,
            Fact.statusTimedOut(TimeoutType.scheduleToClose)
          )
      ),
      ExpectedRule(
        deadline.scheduleToStart.decl,
        None,
        s =>
          waitingSystem(
            s.phase
          ) && s.scheduleToStart == Timeout.expires && s.dispatch == Dispatch.now,
        s =>
          acceptedBecause(
            s.copy(phase = timedOut),
            nominalTimeWindow,
            Fact.statusTimedOut(TimeoutType.scheduleToStart)
          )
      ),
      ExpectedRule(
        deadline.startToClose.decl,
        None,
        s =>
          heldSystem(s.phase) && s.startToClose == Timeout.expires &&
            (s.phase == cancelRequested || !retriesRemain(s)),
        s =>
          acceptedBecause(
            s.copy(phase = timedOut),
            nominalTimeWindow,
            Fact.statusTimedOut(TimeoutType.startToClose)
          )
      ),
      ExpectedRule(
        deadline.startToClose.decl,
        None,
        s => s.phase == started && s.startToClose == Timeout.expires && retriesRemain(s),
        s =>
          acceptedBecause(
            s.copy(phase = scheduled, dispatch = Dispatch.backoff),
            nominalTimeWindow,
            Fact.statusScheduled,
            Fact.attemptCount
          )
      ),
      ExpectedRule(
        deadline.startToClose.decl,
        None,
        s => s.phase == pauseRequested && s.startToClose == Timeout.expires && retriesRemain(s),
        s =>
          acceptedBecause(
            s.copy(phase = paused, dispatch = Dispatch.backoff),
            nominalTimeWindow,
            Fact.statusPaused,
            Fact.attemptCount
          )
      )
    ) ++ List[ExpectedRule[system.State, system.Fact]](
      ExpectedRule(
        worker.heartbeat.decl,
        None,
        s => heldSystem(s.phase) || resetSystem(s.phase),
        s => accepted(s, Fact.heartbeatReceived)
      ),
      ExpectedRule(
        worker.heartbeat.decl,
        None,
        s => !heldSystem(s.phase) && !resetSystem(s.phase),
        rejectedNotFound
      ),
      ExpectedRule(
        deadline.heartbeat.decl,
        None,
        s =>
          heldSystem(s.phase) && s.heartbeat == Timeout.expires &&
            (s.phase == cancelRequested || !retriesRemain(s)),
        s =>
          acceptedBecause(
            s.copy(phase = timedOut),
            nominalTimeWindow,
            Fact.statusTimedOut(TimeoutType.heartbeat),
            Fact.heartbeatTimedOut
          )
      ),
      ExpectedRule(
        deadline.heartbeat.decl,
        None,
        s => s.phase == started && s.heartbeat == Timeout.expires && retriesRemain(s),
        s =>
          acceptedBecause(
            s.copy(phase = scheduled, dispatch = Dispatch.backoff),
            nominalTimeWindow,
            Fact.statusScheduled,
            Fact.attemptCount,
            Fact.heartbeatTimedOut
          )
      ),
      ExpectedRule(
        deadline.heartbeat.decl,
        None,
        s => s.phase == pauseRequested && s.heartbeat == Timeout.expires && retriesRemain(s),
        s =>
          acceptedBecause(
            s.copy(phase = paused, dispatch = Dispatch.backoff),
            nominalTimeWindow,
            Fact.statusPaused,
            Fact.attemptCount,
            Fact.heartbeatTimedOut
          )
      )
    ) ++ startRules ++ systemServiceRules ++ systemResetRules

  // A reset applies at once to a waiting or paused activity, is recorded until a held attempt ends,
  // and then applies on its failure or its own deadline (operator_commands.go:428-563).
  private def systemResetRules: List[ExpectedRule[system.State, system.Fact]] =
    import system.{Fact, Phase, State}
    import Phase.*
    val learns = "the worker learns of the reset on its next heartbeat"
    val ends = "the deferred reset applies as the attempt ends"
    def cleared(s: State, phase: Phase) = s.copy(
      phase = phase,
      attempts = UpTo(0),
      dispatch = if s.dispatch == Dispatch.backoff then Dispatch.now else s.dispatch
    )
    def restarted(s: State) =
      s.copy(
        phase = if s.phase == resetKeepingPause then paused else scheduled,
        attempts = UpTo(0),
        dispatch = Dispatch.now
      )
    def landingFact(s: State) =
      if s.phase == resetKeepingPause then Fact.statusPaused else Fact.statusScheduled
    List[ExpectedRule[State, Fact]](
      ExpectedRule(
        client.reset.decl,
        None,
        s => s.phase == unstarted || closedSystem(s.phase),
        rejectedNotFound
      ),
      ExpectedRule(
        client.reset.decl,
        None,
        _.phase == scheduled,
        s => accepted(cleared(s, scheduled), Fact.statusScheduled)
      ),
      ExpectedRule(
        client.reset.decl,
        None,
        _.phase == started,
        s => acceptedBecause(s.copy(phase = resetRequested), learns, Fact.statusStarted)
      ),
      ExpectedRule(
        client.reset.decl,
        None,
        _.phase == cancelRequested,
        s =>
          rejectedBecause(
            s,
            "failedPrecondition",
            "cannot reset an activity with a pending cancellation (operator_commands.go:458)"
          )
      ),
      ExpectedRule(
        client.reset.decl,
        None,
        s => resetSystem(s.phase),
        s =>
          rejectedBecause(
            s,
            "failedPrecondition",
            "cannot reset an activity with a pending reset (operator_commands.go:460-464)"
          )
      ),
      ExpectedRule(
        client.reset.decl,
        Some(List(ResetPause.resume)),
        _.phase == paused,
        s => accepted(cleared(s, scheduled), Fact.statusScheduled)
      ),
      ExpectedRule(
        client.reset.decl,
        Some(List(ResetPause.resume)),
        _.phase == pauseRequested,
        s => acceptedBecause(s.copy(phase = resetRequested), learns, Fact.statusStarted)
      ),
      ExpectedRule(
        client.reset.decl,
        Some(List(ResetPause.keepPaused)),
        _.phase == paused,
        s => accepted(cleared(s, paused), Fact.statusPaused)
      ),
      ExpectedRule(
        client.reset.decl,
        Some(List(ResetPause.keepPaused)),
        _.phase == pauseRequested,
        s => acceptedBecause(s.copy(phase = resetKeepingPause), learns, Fact.statusPaused)
      )
    ) ++ List(worker.respondFailed, service.respondFailedByID).map(a =>
      ExpectedRule[State, Fact](
        a.decl,
        None,
        s => resetSystem(s.phase),
        s => acceptedBecause(restarted(s), ends, landingFact(s), Fact.attemptCount)
      )
    ) ++ List[ExpectedRule[State, Fact]](
      ExpectedRule(
        deadline.startToClose.decl,
        None,
        s => resetSystem(s.phase) && s.startToClose == Timeout.expires,
        s => acceptedBecause(restarted(s), ends, landingFact(s), Fact.attemptCount)
      ),
      ExpectedRule(
        deadline.heartbeat.decl,
        None,
        s => resetSystem(s.phase) && s.heartbeat == Timeout.expires,
        s =>
          acceptedBecause(
            restarted(s),
            nominalTimeWindow,
            landingFact(s),
            Fact.attemptCount,
            Fact.heartbeatTimedOut
          )
      )
    )

  private def systemServiceRules: List[ExpectedRule[system.State, system.Fact]] =
    import system.{Fact, Phase, State}
    import Phase.*
    val absentOrClosed =
      List(service.respondCompletedByID, service.respondFailedByID, service.respondCanceledByID)
        .map(a =>
          ExpectedRule[State, Fact](
            a.decl,
            None,
            s => s.phase == unstarted || closedSystem(s.phase),
            rejectedNotFound
          )
        )
    val unheld = List(service.respondFailedByID, service.respondCanceledByID)
      .map(a =>
        ExpectedRule[State, Fact](
          a.decl,
          None,
          s => s.phase == scheduled || s.phase == paused,
          rejectedNotFound
        )
      )
    absentOrClosed ++ unheld ++ List[ExpectedRule[State, Fact]](
      ExpectedRule(
        service.respondCompletedByID.decl,
        None,
        s => liveSystem(s.phase),
        s => accepted(s.copy(phase = completed), Fact.statusCompleted)
      ),
      ExpectedRule(
        service.respondFailedByID.decl,
        Some(List(Failure.fatal)),
        s => heldSystem(s.phase),
        s => accepted(s.copy(phase = failed), Fact.statusFailed)
      ),
      ExpectedRule(
        service.respondFailedByID.decl,
        Some(List(Failure.retryable)),
        s => s.phase == started && retriesRemain(s),
        s =>
          acceptedBecause(
            s.copy(phase = scheduled, dispatch = Dispatch.backoff),
            "a retryable attempt backs off; the client reads scheduled again",
            Fact.statusScheduled,
            Fact.attemptCount
          )
      ),
      ExpectedRule(
        service.respondFailedByID.decl,
        Some(List(Failure.retryable)),
        s => s.phase == pauseRequested && retriesRemain(s),
        s =>
          accepted(
            s.copy(phase = paused, dispatch = Dispatch.backoff),
            Fact.statusPaused,
            Fact.attemptCount
          )
      ),
      ExpectedRule(
        service.respondFailedByID.decl,
        Some(List(Failure.retryable)),
        s => (s.phase == started || s.phase == pauseRequested) && !retriesRemain(s),
        s => accepted(s.copy(phase = failed), Fact.statusFailed)
      ),
      ExpectedRule(
        service.respondFailedByID.decl,
        Some(List(Failure.retryable)),
        _.phase == cancelRequested,
        s => accepted(s.copy(phase = canceled), Fact.statusCanceled)
      ),
      ExpectedRule(
        service.respondCanceledByID.decl,
        None,
        _.phase == cancelRequested,
        s => accepted(s.copy(phase = canceled), Fact.statusCanceled)
      ),
      ExpectedRule(
        service.respondCanceledByID.decl,
        None,
        s => s.phase == started || s.phase == pauseRequested || resetSystem(s.phase),
        s => rejectedBecause(s, "invalidArgument", "cancellation was not requested")
      )
    )

  private val startRules: List[ExpectedRule[system.State, system.Fact]] =
    for
      scheduleClose <- Timeout.values.toList
      scheduleStart <- Timeout.values.toList
      startClose <- Timeout.values.toList
      heartbeat <- Timeout.values.toList
      delay <- Timeout.values.toList
      policy <- temporal.features.activity.standalone.MaxAttempts.values.toList
    yield ExpectedRule[system.State, system.Fact](
      client.start.decl,
      Some(List(scheduleClose, scheduleStart, startClose, heartbeat, delay, policy)),
      _.phase == system.Phase.unstarted,
      _ =>
        accepted(
          system.State(
            phase = system.Phase.scheduled,
            dispatch = if delay == Timeout.expires then Dispatch.startDelay else Dispatch.now,
            attempts = UpTo(0),
            scheduleToClose = scheduleClose,
            scheduleToStart = scheduleStart,
            startToClose = startClose,
            heartbeat = heartbeat,
            maxAttempts = policy
          ),
          system.Fact.statusScheduled
        )
    )

  private def retriesRemain(s: system.State): Boolean = s.maxAttempts match
    case temporal.features.activity.standalone.MaxAttempts.unlimited => true
    case temporal.features.activity.standalone.MaxAttempts.one       => s.attempts < 1
    case temporal.features.activity.standalone.MaxAttempts.two       => s.attempts < 2

  private def liveSystem(p: system.Phase): Boolean =
    import system.Phase.*
    p == scheduled || p == started || p == paused ||
    p == pauseRequested || p == cancelRequested || resetSystem(p)

  private def resetSystem(p: system.Phase): Boolean =
    p == system.Phase.resetRequested || p == system.Phase.resetKeepingPause

  private def waitingSystem(p: system.Phase): Boolean =
    p == system.Phase.scheduled

  private def heldSystem(p: system.Phase): Boolean =
    p == system.Phase.started || p == system.Phase.pauseRequested ||
      p == system.Phase.cancelRequested

  private def closedSystem(p: system.Phase): Boolean =
    import system.Phase.*
    p == completed || p == failed || p == canceled || p == terminated || p == timedOut

  test("the standalone signature declares one action for every RPC") {
    assertEquals(client.start.decl.creates, Some(activity))
    assertEquals(client.start.decl.domains.size, 6)

    val controls = List(client.pause, client.unpause, client.requestCancel, client.terminate)
    for control <- controls do
      assertEquals(control.decl.on, Some(activity))
      assertEquals(control.decl.inputs, Nil)
      assertEquals(control.decl.results, "Delivery")

    assertEquals(worker.poll.decl.on, Some(activity))
    assertEquals(worker.respondCompleted.decl.on, Some(activity))
    assertEquals(worker.respondCompleted.decl.inputs, Nil)
    assertEquals(worker.respondCanceled.decl.on, Some(activity))
    assertEquals(worker.respondCanceled.decl.inputs, Nil)
    assertEquals(worker.respondFailed.decl.on, Some(activity))
    assertEquals(worker.respondFailed.decl.tokens, List(Some(failure)))
    assertEquals(
      worker.respondFailed.decl.domains.map(_.values.toList),
      List(List(Failure.fatal, Failure.retryable))
    )
    assertEquals(
      worker.respondFailed.decl.examples,
      List(
        ClassExample(Failure.fatal, "ApplicationFailureNonRetryable"),
        ClassExample(Failure.retryable, "ApplicationFailureRetryable")
      )
    )
  }

  test("the activity machines bind actions in actor-group order") {
    assertEquals(
      product.ActivityProduct.bindings.map(_.decl),
      List(
        client.start,
        client.pause,
        client.unpause,
        client.requestCancel,
        client.terminate,
        client.reset,
        worker.poll,
        worker.heartbeat,
        worker.respondCompleted,
        worker.respondFailed,
        worker.respondCanceled,
        service.respondCompletedByID,
        service.respondFailedByID,
        service.respondCanceledByID,
        process.stop,
        timers.timeout,
        deadline.heartbeat
      ).map(_.decl)
    )
    assertEquals(
      ActivitySystem.bindings.map(_.decl),
      List(
        client.start,
        client.pause,
        client.unpause,
        client.requestCancel,
        client.terminate,
        client.reset,
        worker.poll,
        worker.heartbeat,
        worker.respondCompleted,
        worker.respondFailed,
        worker.respondCanceled,
        service.respondCompletedByID,
        service.respondFailedByID,
        service.respondCanceledByID,
        process.stop,
        timers.startDelay,
        timers.backoff,
        deadline.scheduleToClose,
        deadline.scheduleToStart,
        deadline.startToClose,
        deadline.heartbeat
      ).map(_.decl)
    )
  }

  test("the activity machines keep the recorded step table over every state and class") {
    assertStepTable(product.ActivityProduct, productRules)
    assertStepTable(ActivitySystem, systemRules)
  }

  test("one lost admission response consumes its budget for either durable outcome") {
    val choices = LostStartAnswer.effects.loseResponse(LostStartAnswer.init)
    assertEquals(
      choices.map(_.state.record),
      ActivityRecord.effects.admit(ActivityRecord.init).map(_.state)
    )
    assertEquals(choices.map(_.state.lossAvailable), List(false, false))
    val loss = LostStartAnswer.bindings
      .find(_.decl == temporal.foundations.taskqueue.fault.ackLoss.decl)
      .get
      .function
      .asInstanceOf[Loss] // scalafix:ok DisableSyntax.asInstanceOf
    assertEquals(loss(LostStartAnswer.init), choices)
    for choice <- choices do assertEquals(loss(choice.state), Nil)
    assert(choices.forall(c => LostStartAnswer.end(c.state)))
    assertEquals(choices.map(_.facts), List(List(AdmissionResponseFact.attemptAdmitted), Nil))
  }
