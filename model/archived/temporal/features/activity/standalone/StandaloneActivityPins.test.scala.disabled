package framework

import framework.Finite.given
// What the standalone activity Model's effects do, run as Scala.

import temporal.features.activity.{failure, Failure}
import temporal.features.activity.standalone.{
  activity,
  activityWorker,
  client,
  completionClient,
  protocol,
  ResetPause
}
import temporal.features.activity.{deadline, timers, Timeout, TimeoutType}
import temporal.features.activity.standalone.product
import temporal.actors.worker.worker as process
import framework.outcomes.Outcome
import protocol.{ActivityProtocol, ActivityRecord, LostStartAnswer}

class StandaloneActivityPins extends munit.FunSuite:
  val _ = () => summon[CanEqual[protocol.Phase, protocol.Phase]]
  type Loss =
    protocol.AdmissionResponseState => List[
      Step[
        protocol.AdmissionResponseState,
        Outcome,
        protocol.AdmissionResponseFact
      ]
    ]

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

  private def observed[S, O, F](
      steps: List[Step[S, O, F]]
  ): List[ExpectedStep[S, F]] =
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
        binding.function
          .asInstanceOf[(S, Any) => List[Step[S, O, F]]](state, one)
      case List(one, two) =>
        binding.function
          .asInstanceOf[(S, Any, Any) => List[Step[S, O, F]]](state, one, two)
      case List(one, two, three) =>
        binding.function
          .asInstanceOf[(S, Any, Any, Any) => List[Step[S, O, F]]](
            state,
            one,
            two,
            three
          )
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
          .asInstanceOf[(S, Any, Any, Any, Any, Any, Any) => List[
            Step[S, O, F]
          ]](
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
        rules.filter(r =>
          r.decl == binding.decl && r.inputs.forall(_.equals(inputs)) && r
            .guard(state)
        )
      assert(
        matches.sizeIs <= 1,
        s"several pinned rows match ${binding.decl.name}$inputs in $state"
      )
      val expected =
        matches.headOption.fold(List.empty[ExpectedStep[S, F]])(_.run(state))
      assertEquals(
        observed(run(binding, state, inputs)),
        expected,
        s"${machine.name}: ${binding.decl.name}$inputs in $state"
      )

  private val productRules: List[ExpectedRule[product.State, product.Fact]] =
    List[ExpectedRule[product.State, product.Fact]](
      ExpectedRule(
        client.start.decl,
        None,
        _.phase == product.Phase.scheduled,
        s => accepted(s, product.Fact.statusScheduled)
      ),
      ExpectedRule(
        activityWorker.poll.decl,
        None,
        _.phase == product.Phase.scheduled,
        s =>
          accepted(
            s.copy(phase = product.Phase.started),
            product.Fact.statusStarted
          )
      ),
      ExpectedRule(
        activityWorker.respondCompleted.decl,
        None,
        s => s.phase == product.Phase.started || s.phase == product.Phase.cancelRequested,
        s =>
          accepted(
            s.copy(phase = product.Phase.completed),
            product.Fact.statusCompleted
          )
      ),
      ExpectedRule(
        activityWorker.respondFailed.decl,
        Some(List(Failure.fatal)),
        _.phase == product.Phase.started,
        s =>
          accepted(
            s.copy(phase = product.Phase.failed),
            product.Fact.statusFailed
          ) ++
            accepted(
              s.copy(phase = product.Phase.scheduled),
              product.Fact.statusScheduled
            ) ++
            accepted(
              s.copy(phase = product.Phase.paused),
              product.Fact.statusPaused
            )
      ),
      ExpectedRule(
        activityWorker.respondFailed.decl,
        Some(List(Failure.fatal)),
        _.phase == product.Phase.cancelRequested,
        s =>
          accepted(
            s.copy(phase = product.Phase.failed),
            product.Fact.statusFailed
          )
      ),
      ExpectedRule(
        activityWorker.respondFailed.decl,
        Some(List(Failure.retryable)),
        _.phase == product.Phase.started,
        s =>
          accepted(
            s.copy(phase = product.Phase.scheduled),
            product.Fact.statusScheduled
          ) ++
            accepted(
              s.copy(phase = product.Phase.paused),
              product.Fact.statusPaused
            ) ++
            accepted(
              s.copy(phase = product.Phase.failed),
              product.Fact.statusFailed
            )
      ),
      ExpectedRule(
        activityWorker.respondFailed.decl,
        Some(List(Failure.retryable)),
        _.phase == product.Phase.cancelRequested,
        s =>
          accepted(
            s.copy(phase = product.Phase.canceled),
            product.Fact.statusCanceled
          )
      ),
      ExpectedRule(
        activityWorker.respondCanceled.decl,
        None,
        _.phase == product.Phase.cancelRequested,
        s =>
          accepted(
            s.copy(phase = product.Phase.canceled),
            product.Fact.statusCanceled
          )
      ),
      ExpectedRule(
        activityWorker.respondCanceled.decl,
        None,
        _.phase == product.Phase.started,
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
        _.phase == product.Phase.scheduled,
        s =>
          accepted(
            s.copy(phase = product.Phase.paused),
            product.Fact.statusPaused
          )
      ),
      ExpectedRule(
        client.pause.decl,
        None,
        _.phase == product.Phase.started,
        s =>
          accepted(
            s.copy(phase = product.Phase.paused),
            product.Fact.statusPaused
          ) ++
            accepted(s, product.Fact.statusPaused) ++
            rejectedBecause(
              s,
              "failedPrecondition",
              "pause already requested (chasm/lib/activity/model/model.go:232)"
            )
      ),
      ExpectedRule(
        client.pause.decl,
        None,
        s => s.phase == product.Phase.paused || s.phase == product.Phase.cancelRequested,
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
        _.phase == product.Phase.paused,
        s =>
          accepted(
            s.copy(phase = product.Phase.scheduled),
            product.Fact.statusScheduled
          )
      ),
      ExpectedRule(
        client.unpause.decl,
        None,
        _.phase == product.Phase.started,
        s =>
          accepted(s, product.Fact.statusStarted) ++
            rejectedBecause(
              s,
              "failedPrecondition",
              "activity is not paused (chasm/lib/activity/model/model.go:251)"
            )
      ),
      ExpectedRule(
        client.unpause.decl,
        None,
        s => s.phase == product.Phase.scheduled || s.phase == product.Phase.cancelRequested,
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
        s => s.phase == product.Phase.scheduled || s.phase == product.Phase.paused,
        s =>
          accepted(
            s.copy(phase = product.Phase.canceled),
            product.Fact.statusCanceled
          )
      ),
      ExpectedRule(
        client.requestCancel.decl,
        None,
        _.phase == product.Phase.started,
        s =>
          accepted(
            s.copy(phase = product.Phase.cancelRequested),
            product.Fact.statusCancelRequested
          )
      ),
      ExpectedRule(
        client.requestCancel.decl,
        None,
        _.phase == product.Phase.cancelRequested,
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
        s =>
          accepted(
            s.copy(phase = product.Phase.terminated),
            product.Fact.statusTerminated
          )
      ),
      ExpectedRule(
        timers.timeout.decl,
        None,
        s => productLive(s.phase),
        s =>
          accepted(
            s.copy(phase = product.Phase.timedOut),
            product.Fact.statusTimedOut
          )
      )
    ) ++ List[ExpectedRule[product.State, product.Fact]](
      ExpectedRule(
        activityWorker.heartbeat.decl,
        None,
        s => s.phase == product.Phase.started || s.phase == product.Phase.cancelRequested,
        s => accepted(s, product.Fact.heartbeatReceived)
      ),
      ExpectedRule(
        activityWorker.heartbeat.decl,
        None,
        s => s.phase != product.Phase.started && s.phase != product.Phase.cancelRequested,
        rejectedNotFound
      ),
      ExpectedRule(
        deadline.heartbeat.decl,
        None,
        _.phase == product.Phase.started,
        s =>
          accepted(
            s.copy(phase = product.Phase.scheduled),
            product.Fact.statusScheduled,
            product.Fact.heartbeatTimedOut
          ) ++
            accepted(
              s.copy(phase = product.Phase.paused),
              product.Fact.statusPaused,
              product.Fact.heartbeatTimedOut
            ) ++
            accepted(
              s.copy(phase = product.Phase.timedOut),
              product.Fact.statusTimedOut,
              product.Fact.heartbeatTimedOut
            )
      ),
      ExpectedRule(
        deadline.heartbeat.decl,
        None,
        _.phase == product.Phase.cancelRequested,
        s =>
          accepted(
            s.copy(phase = product.Phase.timedOut),
            product.Fact.statusTimedOut,
            product.Fact.heartbeatTimedOut
          )
      )
    ) ++ productServiceRules ++ productResetRules

  private def productResetRules: List[ExpectedRule[product.State, product.Fact]] =
    val pending = "a reset is already pending (operator_commands.go:460-464)"
    List[ExpectedRule[product.State, product.Fact]](
      ExpectedRule(
        client.reset.decl,
        None,
        s => closedProduct(s.phase),
        rejectedNotFound
      ),
      ExpectedRule(
        client.reset.decl,
        None,
        _.phase == product.Phase.scheduled,
        s => accepted(s, product.Fact.statusScheduled)
      ),
      ExpectedRule(
        client.reset.decl,
        None,
        _.phase == product.Phase.cancelRequested,
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
        _.phase == product.Phase.paused,
        s =>
          accepted(
            s.copy(phase = product.Phase.scheduled),
            product.Fact.statusScheduled
          )
      ),
      ExpectedRule(
        client.reset.decl,
        Some(List(ResetPause.resume)),
        _.phase == product.Phase.started,
        s =>
          accepted(s, product.Fact.statusStarted) ++ rejectedBecause(
            s,
            "failedPrecondition",
            pending
          )
      ),
      ExpectedRule(
        client.reset.decl,
        Some(List(ResetPause.keepPaused)),
        _.phase == product.Phase.paused,
        s => accepted(s, product.Fact.statusPaused)
      ),
      ExpectedRule(
        client.reset.decl,
        Some(List(ResetPause.keepPaused)),
        _.phase == product.Phase.started,
        s =>
          accepted(s, product.Fact.statusStarted) ++ accepted(
            s,
            product.Fact.statusPaused
          ) ++
            rejectedBecause(s, "failedPrecondition", pending)
      )
    )

  private def productServiceRules: List[ExpectedRule[product.State, product.Fact]] =
    val closed =
      List(
        completionClient.respondCompletedByID,
        completionClient.respondFailedByID,
        completionClient.respondCanceledByID
      )
        .map(a =>
          ExpectedRule[product.State, product.Fact](
            a.decl,
            None,
            s => closedProduct(s.phase),
            rejectedNotFound
          )
        )
    val unheld = List(
      completionClient.respondFailedByID,
      completionClient.respondCanceledByID
    )
      .map(a =>
        ExpectedRule[product.State, product.Fact](
          a.decl,
          None,
          s => s.phase == product.Phase.scheduled || s.phase == product.Phase.paused,
          rejectedNotFound
        )
      )
    closed ++ unheld ++ List[ExpectedRule[product.State, product.Fact]](
      ExpectedRule(
        completionClient.respondCompletedByID.decl,
        None,
        s => productLive(s.phase),
        s =>
          accepted(
            s.copy(phase = product.Phase.completed),
            product.Fact.statusCompleted
          )
      ),
      ExpectedRule(
        completionClient.respondFailedByID.decl,
        Some(List(Failure.fatal)),
        _.phase == product.Phase.started,
        s =>
          accepted(
            s.copy(phase = product.Phase.failed),
            product.Fact.statusFailed
          ) ++
            accepted(
              s.copy(phase = product.Phase.scheduled),
              product.Fact.statusScheduled
            ) ++
            accepted(
              s.copy(phase = product.Phase.paused),
              product.Fact.statusPaused
            )
      ),
      ExpectedRule(
        completionClient.respondFailedByID.decl,
        Some(List(Failure.fatal)),
        _.phase == product.Phase.cancelRequested,
        s =>
          accepted(
            s.copy(phase = product.Phase.failed),
            product.Fact.statusFailed
          )
      ),
      ExpectedRule(
        completionClient.respondFailedByID.decl,
        Some(List(Failure.retryable)),
        _.phase == product.Phase.started,
        s =>
          accepted(
            s.copy(phase = product.Phase.scheduled),
            product.Fact.statusScheduled
          ) ++
            accepted(
              s.copy(phase = product.Phase.paused),
              product.Fact.statusPaused
            ) ++
            accepted(
              s.copy(phase = product.Phase.failed),
              product.Fact.statusFailed
            )
      ),
      ExpectedRule(
        completionClient.respondFailedByID.decl,
        Some(List(Failure.retryable)),
        _.phase == product.Phase.cancelRequested,
        s =>
          accepted(
            s.copy(phase = product.Phase.canceled),
            product.Fact.statusCanceled
          )
      ),
      ExpectedRule(
        completionClient.respondCanceledByID.decl,
        None,
        _.phase == product.Phase.cancelRequested,
        s =>
          accepted(
            s.copy(phase = product.Phase.canceled),
            product.Fact.statusCanceled
          )
      ),
      ExpectedRule(
        completionClient.respondCanceledByID.decl,
        None,
        _.phase == product.Phase.started,
        s =>
          rejectedBecause(
            s,
            "invalidArgument",
            "cancellation was not requested"
          )
      )
    )

  private def closedProduct(p: product.Phase): Boolean =
    p == product.Phase.completed || p == product.Phase.failed || p == product.Phase.canceled || p == product.Phase.terminated || p == product.Phase.timedOut

  private def productLive(p: product.Phase): Boolean =
    p == product.Phase.scheduled || p == product.Phase.started || p == product.Phase.paused || p == product.Phase.cancelRequested

  private lazy val systemRules: List[ExpectedRule[protocol.State, protocol.Fact]] =
    List[ExpectedRule[protocol.State, protocol.Fact]](
      ExpectedRule(
        activityWorker.poll.decl,
        None,
        s => s.phase == protocol.Phase.scheduled && s.dispatch == protocol.Dispatch.now,
        s =>
          accepted(
            s.copy(
              phase = protocol.Phase.started,
              attempts = UpTo[2](((s.attempts: Int) + 1).min(2))
            ),
            protocol.Fact.statusStarted,
            protocol.Fact.attemptCount
          )
      ),
      ExpectedRule(
        activityWorker.respondCompleted.decl,
        None,
        s => heldSystem(s.phase) || resetSystem(s.phase),
        s =>
          accepted(
            s.copy(phase = protocol.Phase.completed),
            protocol.Fact.statusCompleted
          )
      ),
      ExpectedRule(
        activityWorker.respondFailed.decl,
        Some(List(Failure.fatal)),
        s => heldSystem(s.phase),
        s =>
          accepted(
            s.copy(phase = protocol.Phase.failed),
            protocol.Fact.statusFailed
          )
      ),
      ExpectedRule(
        activityWorker.respondFailed.decl,
        Some(List(Failure.retryable)),
        s => s.phase == protocol.Phase.started && retriesRemain(s),
        s =>
          acceptedBecause(
            s.copy(
              phase = protocol.Phase.scheduled,
              dispatch = protocol.Dispatch.backoff
            ),
            "a retryable attempt backs off; the client reads scheduled again",
            protocol.Fact.statusScheduled,
            protocol.Fact.attemptCount
          )
      ),
      ExpectedRule(
        activityWorker.respondFailed.decl,
        Some(List(Failure.retryable)),
        _.phase == protocol.Phase.cancelRequested,
        s =>
          accepted(
            s.copy(phase = protocol.Phase.canceled),
            protocol.Fact.statusCanceled
          )
      ),
      ExpectedRule(
        activityWorker.respondFailed.decl,
        Some(List(Failure.retryable)),
        s =>
          s.phase == protocol.Phase.pauseRequested && retriesRemain(
            s
          ),
        s =>
          accepted(
            s.copy(
              phase = protocol.Phase.paused,
              dispatch = protocol.Dispatch.backoff
            ),
            protocol.Fact.statusPaused,
            protocol.Fact.attemptCount
          )
      ),
      ExpectedRule(
        activityWorker.respondFailed.decl,
        Some(List(Failure.retryable)),
        s =>
          (s.phase == protocol.Phase.started || s.phase == protocol.Phase.pauseRequested) && !retriesRemain(
            s
          ),
        s =>
          accepted(
            s.copy(phase = protocol.Phase.failed),
            protocol.Fact.statusFailed
          )
      ),
      ExpectedRule(
        activityWorker.respondCanceled.decl,
        None,
        _.phase == protocol.Phase.cancelRequested,
        s =>
          accepted(
            s.copy(phase = protocol.Phase.canceled),
            protocol.Fact.statusCanceled
          )
      ),
      ExpectedRule(
        activityWorker.respondCanceled.decl,
        None,
        s =>
          s.phase == protocol.Phase.started || s.phase == protocol.Phase.pauseRequested || resetSystem(
            s.phase
          ),
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
        s => closedSystem(s.phase),
        rejectedNotFound
      ),
      ExpectedRule(
        client.unpause.decl,
        None,
        s => closedSystem(s.phase),
        rejectedNotFound
      ),
      ExpectedRule(
        client.requestCancel.decl,
        None,
        s => closedSystem(s.phase),
        rejectedNotFound
      ),
      ExpectedRule(
        client.terminate.decl,
        None,
        s => closedSystem(s.phase),
        rejectedNotFound
      ),
      ExpectedRule(
        client.pause.decl,
        None,
        _.phase == protocol.Phase.scheduled,
        s =>
          accepted(
            s.copy(phase = protocol.Phase.paused),
            protocol.Fact.statusPaused
          )
      ),
      ExpectedRule(
        client.pause.decl,
        None,
        _.phase == protocol.Phase.started,
        s =>
          acceptedBecause(
            s.copy(phase = protocol.Phase.pauseRequested),
            "the worker learns of the pause on its next heartbeat",
            protocol.Fact.statusPaused
          )
      ),
      ExpectedRule(
        client.pause.decl,
        None,
        s =>
          s.phase == protocol.Phase.paused || s.phase == protocol.Phase.pauseRequested || s.phase == protocol.Phase.cancelRequested ||
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
        _.phase == protocol.Phase.paused,
        s =>
          accepted(
            s.copy(phase = protocol.Phase.scheduled),
            protocol.Fact.statusScheduled
          )
      ),
      ExpectedRule(
        client.unpause.decl,
        None,
        _.phase == protocol.Phase.pauseRequested,
        s =>
          accepted(
            s.copy(phase = protocol.Phase.started),
            protocol.Fact.statusStarted
          )
      ),
      ExpectedRule(
        client.unpause.decl,
        None,
        s =>
          s.phase == protocol.Phase.scheduled || s.phase == protocol.Phase.started || s.phase == protocol.Phase.cancelRequested ||
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
        s => s.phase == protocol.Phase.scheduled || s.phase == protocol.Phase.paused,
        s =>
          accepted(
            s.copy(phase = protocol.Phase.canceled),
            protocol.Fact.statusCanceled
          )
      ),
      ExpectedRule(
        client.requestCancel.decl,
        None,
        s =>
          s.phase == protocol.Phase.started || s.phase == protocol.Phase.pauseRequested || resetSystem(
            s.phase
          ),
        s =>
          accepted(
            s.copy(phase = protocol.Phase.cancelRequested),
            protocol.Fact.statusCancelRequested
          )
      ),
      ExpectedRule(
        client.requestCancel.decl,
        None,
        _.phase == protocol.Phase.cancelRequested,
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
        s =>
          accepted(
            s.copy(phase = protocol.Phase.terminated),
            protocol.Fact.statusTerminated
          )
      ),
      ExpectedRule(process.stop.decl, None, _ => true, s => accepted(s)),
      ExpectedRule(
        timers.startDelay.decl,
        None,
        s =>
          liveSystem(
            s.phase
          ) && s.dispatch == protocol.Dispatch.startDelay,
        s =>
          acceptedBecause(
            s.copy(dispatch = protocol.Dispatch.now),
            nominalTimeWindow
          )
      ),
      ExpectedRule(
        timers.backoff.decl,
        None,
        s =>
          liveSystem(
            s.phase
          ) && s.dispatch == protocol.Dispatch.backoff,
        s =>
          acceptedBecause(
            s.copy(dispatch = protocol.Dispatch.now),
            nominalTimeWindow
          )
      ),
      ExpectedRule(
        deadline.scheduleToClose.decl,
        None,
        s =>
          liveSystem(
            s.phase
          ) && s.scheduleToClose == Timeout.expires && s.dispatch != protocol.Dispatch.startDelay,
        s =>
          acceptedBecause(
            s.copy(phase = protocol.Phase.timedOut),
            nominalTimeWindow,
            protocol.Fact.statusTimedOut(TimeoutType.scheduleToClose)
          )
      ),
      ExpectedRule(
        deadline.scheduleToStart.decl,
        None,
        s =>
          waitingSystem(
            s.phase
          ) && s.scheduleToStart == Timeout.expires && s.dispatch == protocol.Dispatch.now,
        s =>
          acceptedBecause(
            s.copy(phase = protocol.Phase.timedOut),
            nominalTimeWindow,
            protocol.Fact.statusTimedOut(TimeoutType.scheduleToStart)
          )
      ),
      ExpectedRule(
        deadline.startToClose.decl,
        None,
        s =>
          heldSystem(
            s.phase
          ) && s.startToClose == Timeout.expires &&
            (s.phase == protocol.Phase.cancelRequested || !retriesRemain(
              s
            )),
        s =>
          acceptedBecause(
            s.copy(phase = protocol.Phase.timedOut),
            nominalTimeWindow,
            protocol.Fact.statusTimedOut(TimeoutType.startToClose)
          )
      ),
      ExpectedRule(
        deadline.startToClose.decl,
        None,
        s =>
          s.phase == protocol.Phase.started && s.startToClose == Timeout.expires && retriesRemain(
            s
          ),
        s =>
          acceptedBecause(
            s.copy(
              phase = protocol.Phase.scheduled,
              dispatch = protocol.Dispatch.backoff
            ),
            nominalTimeWindow,
            protocol.Fact.statusScheduled,
            protocol.Fact.attemptCount
          )
      ),
      ExpectedRule(
        deadline.startToClose.decl,
        None,
        s =>
          s.phase == protocol.Phase.pauseRequested && s.startToClose == Timeout.expires && retriesRemain(
            s
          ),
        s =>
          acceptedBecause(
            s.copy(
              phase = protocol.Phase.paused,
              dispatch = protocol.Dispatch.backoff
            ),
            nominalTimeWindow,
            protocol.Fact.statusPaused,
            protocol.Fact.attemptCount
          )
      )
    ) ++ List[ExpectedRule[protocol.State, protocol.Fact]](
      ExpectedRule(
        activityWorker.heartbeat.decl,
        None,
        s => heldSystem(s.phase) || resetSystem(s.phase),
        s => accepted(s, protocol.Fact.heartbeatReceived)
      ),
      ExpectedRule(
        activityWorker.heartbeat.decl,
        None,
        s => !heldSystem(s.phase) && !resetSystem(s.phase),
        rejectedNotFound
      ),
      ExpectedRule(
        deadline.heartbeat.decl,
        None,
        s =>
          heldSystem(s.phase) && s.heartbeat == Timeout.expires &&
            (s.phase == protocol.Phase.cancelRequested || !retriesRemain(
              s
            )),
        s =>
          acceptedBecause(
            s.copy(phase = protocol.Phase.timedOut),
            nominalTimeWindow,
            protocol.Fact.statusTimedOut(TimeoutType.heartbeat),
            protocol.Fact.heartbeatTimedOut
          )
      ),
      ExpectedRule(
        deadline.heartbeat.decl,
        None,
        s =>
          s.phase == protocol.Phase.started && s.heartbeat == Timeout.expires && retriesRemain(
            s
          ),
        s =>
          acceptedBecause(
            s.copy(
              phase = protocol.Phase.scheduled,
              dispatch = protocol.Dispatch.backoff
            ),
            nominalTimeWindow,
            protocol.Fact.statusScheduled,
            protocol.Fact.attemptCount,
            protocol.Fact.heartbeatTimedOut
          )
      ),
      ExpectedRule(
        deadline.heartbeat.decl,
        None,
        s =>
          s.phase == protocol.Phase.pauseRequested && s.heartbeat == Timeout.expires && retriesRemain(
            s
          ),
        s =>
          acceptedBecause(
            s.copy(
              phase = protocol.Phase.paused,
              dispatch = protocol.Dispatch.backoff
            ),
            nominalTimeWindow,
            protocol.Fact.statusPaused,
            protocol.Fact.attemptCount,
            protocol.Fact.heartbeatTimedOut
          )
      )
    ) ++ startRules ++ systemServiceRules ++ systemResetRules

  // A reset applies at once to a waiting or paused activity, is recorded until a held attempt ends,
  // and then applies on its failure or its own deadline (operator_commands.go:428-563).
  private def systemResetRules: List[ExpectedRule[protocol.State, protocol.Fact]] =
    val learns = "the worker learns of the reset on its next heartbeat"
    val ends = "the deferred reset applies as the attempt ends"
    def cleared(s: protocol.State, phase: protocol.Phase) = s.copy(
      phase = phase,
      attempts = UpTo(0),
      dispatch =
        if s.dispatch == protocol.Dispatch.backoff then protocol.Dispatch.now
        else s.dispatch
    )
    def restarted(s: protocol.State) =
      s.copy(
        phase =
          if s.phase == protocol.Phase.resetKeepingPause then protocol.Phase.paused
          else protocol.Phase.scheduled,
        attempts = UpTo(0),
        dispatch = protocol.Dispatch.now
      )
    def landingFact(s: protocol.State) =
      if s.phase == protocol.Phase.resetKeepingPause then protocol.Fact.statusPaused
      else protocol.Fact.statusScheduled
    List[ExpectedRule[protocol.State, protocol.Fact]](
      ExpectedRule(
        client.reset.decl,
        None,
        s =>
          s.phase == protocol.Phase.unstarted || closedSystem(
            s.phase
          ),
        rejectedNotFound
      ),
      ExpectedRule(
        client.reset.decl,
        None,
        _.phase == protocol.Phase.scheduled,
        s =>
          accepted(
            cleared(s, protocol.Phase.scheduled),
            protocol.Fact.statusScheduled
          )
      ),
      ExpectedRule(
        client.reset.decl,
        None,
        _.phase == protocol.Phase.started,
        s =>
          acceptedBecause(
            s.copy(phase = protocol.Phase.resetRequested),
            learns,
            protocol.Fact.statusStarted
          )
      ),
      ExpectedRule(
        client.reset.decl,
        None,
        _.phase == protocol.Phase.cancelRequested,
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
        _.phase == protocol.Phase.paused,
        s =>
          accepted(
            cleared(s, protocol.Phase.scheduled),
            protocol.Fact.statusScheduled
          )
      ),
      ExpectedRule(
        client.reset.decl,
        Some(List(ResetPause.resume)),
        _.phase == protocol.Phase.pauseRequested,
        s =>
          acceptedBecause(
            s.copy(phase = protocol.Phase.resetRequested),
            learns,
            protocol.Fact.statusStarted
          )
      ),
      ExpectedRule(
        client.reset.decl,
        Some(List(ResetPause.keepPaused)),
        _.phase == protocol.Phase.paused,
        s =>
          accepted(
            cleared(s, protocol.Phase.paused),
            protocol.Fact.statusPaused
          )
      ),
      ExpectedRule(
        client.reset.decl,
        Some(List(ResetPause.keepPaused)),
        _.phase == protocol.Phase.pauseRequested,
        s =>
          acceptedBecause(
            s.copy(phase = protocol.Phase.resetKeepingPause),
            learns,
            protocol.Fact.statusPaused
          )
      )
    ) ++ List(activityWorker.respondFailed, completionClient.respondFailedByID)
      .map(a =>
        ExpectedRule[protocol.State, protocol.Fact](
          a.decl,
          None,
          s => resetSystem(s.phase),
          s =>
            acceptedBecause(
              restarted(s),
              ends,
              landingFact(s),
              protocol.Fact.attemptCount
            )
        )
      ) ++ List[ExpectedRule[protocol.State, protocol.Fact]](
      ExpectedRule(
        deadline.startToClose.decl,
        None,
        s => resetSystem(s.phase) && s.startToClose == Timeout.expires,
        s =>
          acceptedBecause(
            restarted(s),
            ends,
            landingFact(s),
            protocol.Fact.attemptCount
          )
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
            protocol.Fact.attemptCount,
            protocol.Fact.heartbeatTimedOut
          )
      )
    )

  private def systemServiceRules: List[ExpectedRule[protocol.State, protocol.Fact]] =
    val absentOrClosed =
      List(
        completionClient.respondCompletedByID,
        completionClient.respondFailedByID,
        completionClient.respondCanceledByID
      )
        .map(a =>
          ExpectedRule[protocol.State, protocol.Fact](
            a.decl,
            None,
            s =>
              s.phase == protocol.Phase.unstarted || closedSystem(
                s.phase
              ),
            rejectedNotFound
          )
        )
    val unheld = List(
      completionClient.respondFailedByID,
      completionClient.respondCanceledByID
    )
      .map(a =>
        ExpectedRule[protocol.State, protocol.Fact](
          a.decl,
          None,
          s => s.phase == protocol.Phase.scheduled || s.phase == protocol.Phase.paused,
          rejectedNotFound
        )
      )
    absentOrClosed ++ unheld ++ List[
      ExpectedRule[protocol.State, protocol.Fact]
    ](
      ExpectedRule(
        completionClient.respondCompletedByID.decl,
        None,
        s => liveSystem(s.phase),
        s =>
          accepted(
            s.copy(phase = protocol.Phase.completed),
            protocol.Fact.statusCompleted
          )
      ),
      ExpectedRule(
        completionClient.respondFailedByID.decl,
        Some(List(Failure.fatal)),
        s => heldSystem(s.phase),
        s =>
          accepted(
            s.copy(phase = protocol.Phase.failed),
            protocol.Fact.statusFailed
          )
      ),
      ExpectedRule(
        completionClient.respondFailedByID.decl,
        Some(List(Failure.retryable)),
        s => s.phase == protocol.Phase.started && retriesRemain(s),
        s =>
          acceptedBecause(
            s.copy(
              phase = protocol.Phase.scheduled,
              dispatch = protocol.Dispatch.backoff
            ),
            "a retryable attempt backs off; the client reads scheduled again",
            protocol.Fact.statusScheduled,
            protocol.Fact.attemptCount
          )
      ),
      ExpectedRule(
        completionClient.respondFailedByID.decl,
        Some(List(Failure.retryable)),
        s =>
          s.phase == protocol.Phase.pauseRequested && retriesRemain(
            s
          ),
        s =>
          accepted(
            s.copy(
              phase = protocol.Phase.paused,
              dispatch = protocol.Dispatch.backoff
            ),
            protocol.Fact.statusPaused,
            protocol.Fact.attemptCount
          )
      ),
      ExpectedRule(
        completionClient.respondFailedByID.decl,
        Some(List(Failure.retryable)),
        s =>
          (s.phase == protocol.Phase.started || s.phase == protocol.Phase.pauseRequested) && !retriesRemain(
            s
          ),
        s =>
          accepted(
            s.copy(phase = protocol.Phase.failed),
            protocol.Fact.statusFailed
          )
      ),
      ExpectedRule(
        completionClient.respondFailedByID.decl,
        Some(List(Failure.retryable)),
        _.phase == protocol.Phase.cancelRequested,
        s =>
          accepted(
            s.copy(phase = protocol.Phase.canceled),
            protocol.Fact.statusCanceled
          )
      ),
      ExpectedRule(
        completionClient.respondCanceledByID.decl,
        None,
        _.phase == protocol.Phase.cancelRequested,
        s =>
          accepted(
            s.copy(phase = protocol.Phase.canceled),
            protocol.Fact.statusCanceled
          )
      ),
      ExpectedRule(
        completionClient.respondCanceledByID.decl,
        None,
        s =>
          s.phase == protocol.Phase.started || s.phase == protocol.Phase.pauseRequested || resetSystem(
            s.phase
          ),
        s =>
          rejectedBecause(
            s,
            "invalidArgument",
            "cancellation was not requested"
          )
      )
    )

  private val startRules: List[ExpectedRule[protocol.State, protocol.Fact]] =
    for
      scheduleClose <- Timeout.values.toList
      scheduleStart <- Timeout.values.toList
      startClose <- Timeout.values.toList
      heartbeat <- Timeout.values.toList
      delay <- Timeout.values.toList
      policy <- temporal.features.activity.standalone.MaxAttempts.values.toList
    yield ExpectedRule[protocol.State, protocol.Fact](
      client.start.decl,
      Some(
        List(scheduleClose, scheduleStart, startClose, heartbeat, delay, policy)
      ),
      _.phase == protocol.Phase.unstarted,
      _ =>
        accepted(
          protocol.State(
            phase = protocol.Phase.scheduled,
            dispatch =
              if delay == Timeout.expires then protocol.Dispatch.startDelay
              else protocol.Dispatch.now,
            attempts = UpTo(0),
            scheduleToClose = scheduleClose,
            scheduleToStart = scheduleStart,
            startToClose = startClose,
            heartbeat = heartbeat,
            maxAttempts = policy
          ),
          protocol.Fact.statusScheduled
        )
    )

  private def retriesRemain(s: protocol.State): Boolean = s.maxAttempts match
    case temporal.features.activity.standalone.MaxAttempts.unlimited => true
    case temporal.features.activity.standalone.MaxAttempts.one       => s.attempts < 1
    case temporal.features.activity.standalone.MaxAttempts.two       => s.attempts < 2

  private def liveSystem(p: protocol.Phase): Boolean =
    p == protocol.Phase.scheduled || p == protocol.Phase.started || p == protocol.Phase.paused ||
      p == protocol.Phase.pauseRequested || p == protocol.Phase.cancelRequested || resetSystem(
        p
      )

  private def resetSystem(p: protocol.Phase): Boolean =
    p == protocol.Phase.resetRequested || p == protocol.Phase.resetKeepingPause

  private def waitingSystem(p: protocol.Phase): Boolean =
    p == protocol.Phase.scheduled

  private def heldSystem(p: protocol.Phase): Boolean =
    p == protocol.Phase.started || p == protocol.Phase.pauseRequested ||
      p == protocol.Phase.cancelRequested

  private def closedSystem(p: protocol.Phase): Boolean =
    p == protocol.Phase.completed || p == protocol.Phase.failed || p == protocol.Phase.canceled || p == protocol.Phase.terminated || p == protocol.Phase.timedOut

  test("the standalone signature declares one action for every RPC") {
    assertEquals(client.start.decl.creates, Some(activity))
    assertEquals(client.start.decl.domains.size, 6)

    val controls =
      List(client.pause, client.unpause, client.requestCancel, client.terminate)
    for control <- controls do
      assertEquals(control.decl.on, Some(activity))
      assertEquals(control.decl.inputs, Nil)
      assertEquals(control.decl.results, "Delivery")

    assertEquals(activityWorker.poll.decl.on, Some(activity))
    assertEquals(activityWorker.respondCompleted.decl.on, Some(activity))
    assertEquals(activityWorker.respondCompleted.decl.inputs, Nil)
    assertEquals(activityWorker.respondCanceled.decl.on, Some(activity))
    assertEquals(activityWorker.respondCanceled.decl.inputs, Nil)
    assertEquals(activityWorker.respondFailed.decl.on, Some(activity))
    assertEquals(activityWorker.respondFailed.decl.tokens, List(Some(failure)))
    assertEquals(
      activityWorker.respondFailed.decl.domains.map(_.values.toList),
      List(List(Failure.fatal, Failure.retryable))
    )
    assertEquals(
      activityWorker.respondFailed.decl.examples,
      List(
        ClassExample(Failure.fatal, "ApplicationFailureNonRetryable"),
        ClassExample(Failure.retryable, "ApplicationFailureRetryable")
      )
    )
  }

  test("the activity machines bind actions in their exact facet order") {
    assertEquals(
      product.ActivityProduct.bindings.map(_.decl),
      List(
        client.start,
        activityWorker.poll,
        client.pause,
        client.unpause,
        client.requestCancel,
        activityWorker.respondCanceled,
        completionClient.respondCanceledByID,
        client.terminate,
        client.reset,
        activityWorker.heartbeat,
        deadline.heartbeat,
        activityWorker.respondCompleted,
        completionClient.respondCompletedByID,
        activityWorker.respondFailed,
        completionClient.respondFailedByID,
        timers.timeout
      ).map(_.decl)
    )
    assertEquals(
      ActivityProtocol.bindings.map(_.decl),
      List(
        client.start,
        activityWorker.poll,
        timers.startDelay,
        timers.backoff,
        client.pause,
        client.unpause,
        client.requestCancel,
        activityWorker.respondCanceled,
        completionClient.respondCanceledByID,
        client.terminate,
        process.stop,
        client.reset,
        activityWorker.heartbeat,
        deadline.heartbeat,
        activityWorker.respondCompleted,
        completionClient.respondCompletedByID,
        activityWorker.respondFailed,
        completionClient.respondFailedByID,
        deadline.scheduleToClose,
        deadline.scheduleToStart,
        deadline.startToClose
      ).map(_.decl)
    )
  }

  test(
    "the activity machines keep the recorded step table over every state and class"
  ) {
    assertStepTable(product.ActivityProduct, productRules)
    assertStepTable(ActivityProtocol, systemRules)
  }

  test(
    "one lost admission response consumes its budget for either durable outcome"
  ) {
    val choices =
      LostStartAnswer.responseLoss.effects.loseResponse(LostStartAnswer.init)
    assertEquals(
      choices.map(_.state.record),
      ActivityRecord.admission.effects.admit(ActivityRecord.init).map(_.state)
    )
    assertEquals(choices.map(_.state.lossAvailable), List(false, false))
    val loss = LostStartAnswer.bindings
      .find(
        _.decl == temporal.foundations.taskqueue.fault.ackLoss.decl
      )
      .get
      .function
      .asInstanceOf[Loss] // scalafix:ok DisableSyntax.asInstanceOf
    assertEquals(loss(LostStartAnswer.init), choices)
    for choice <- choices do assertEquals(loss(choice.state), Nil)
    assert(choices.forall(c => LostStartAnswer.end(c.state)))
    assertEquals(
      choices.map(_.facts),
      List(List(protocol.AdmissionResponseFact.attemptAdmitted), Nil)
    )
  }
