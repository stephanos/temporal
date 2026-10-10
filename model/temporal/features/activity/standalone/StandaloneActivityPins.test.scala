package framework

import framework.Finite.given
// What the standalone activity Model's effects do, run as Scala.

import temporal.features.activity.{failure, Failure}
import temporal.features.activity.standalone.{
  activity,
  activityWorker,
  client,
  completionClient,
  system,
  ResetPause
}
import temporal.features.activity.{deadline, timers, Timeout, TimeoutType}
import temporal.features.activity.standalone.product
import temporal.actors.worker.worker as process
import framework.outcomes.Outcome
import system.{ActivityRecord, ActivitySystem, LostStartAnswer}

class StandaloneActivityPins extends munit.FunSuite:
  val _ = () => summon[CanEqual[system.Phase, system.Phase]]
  type Loss =
    system.AdmissionResponseState => List[
      Step[system.AdmissionResponseState, Outcome, system.AdmissionResponseFact]
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
        rules.filter(r =>
          r.decl == binding.decl && r.inputs.forall(_.equals(inputs)) && r.guard(state)
        ) // scalafix:ok DisableSyntax.==
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
    List[ExpectedRule[product.State, product.Fact]](
      ExpectedRule(
        client.start.decl,
        None,
        _.phase == product.Phase.scheduled, // scalafix:ok DisableSyntax.==
        s => accepted(s, product.Fact.statusScheduled)
      ),
      ExpectedRule(
        activityWorker.poll.decl,
        None,
        _.phase == product.Phase.scheduled, // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = product.Phase.started), product.Fact.statusStarted)
      ),
      ExpectedRule(
        activityWorker.respondCompleted.decl,
        None,
        s => s.phase == product.Phase.started || s.phase == product.Phase.cancelRequested, // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = product.Phase.completed), product.Fact.statusCompleted)
      ),
      ExpectedRule(
        activityWorker.respondFailed.decl,
        Some(List(Failure.fatal)),
        _.phase == product.Phase.started, // scalafix:ok DisableSyntax.==
        s =>
          accepted(s.copy(phase = product.Phase.failed), product.Fact.statusFailed) ++
            accepted(s.copy(phase = product.Phase.scheduled), product.Fact.statusScheduled) ++
            accepted(s.copy(phase = product.Phase.paused), product.Fact.statusPaused)
      ),
      ExpectedRule(
        activityWorker.respondFailed.decl,
        Some(List(Failure.fatal)),
        _.phase == product.Phase.cancelRequested, // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = product.Phase.failed), product.Fact.statusFailed)
      ),
      ExpectedRule(
        activityWorker.respondFailed.decl,
        Some(List(Failure.retryable)),
        _.phase == product.Phase.started, // scalafix:ok DisableSyntax.==
        s =>
          accepted(s.copy(phase = product.Phase.scheduled), product.Fact.statusScheduled) ++
            accepted(s.copy(phase = product.Phase.paused), product.Fact.statusPaused) ++
            accepted(s.copy(phase = product.Phase.failed), product.Fact.statusFailed)
      ),
      ExpectedRule(
        activityWorker.respondFailed.decl,
        Some(List(Failure.retryable)),
        _.phase == product.Phase.cancelRequested, // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = product.Phase.canceled), product.Fact.statusCanceled)
      ),
      ExpectedRule(
        activityWorker.respondCanceled.decl,
        None,
        _.phase == product.Phase.cancelRequested, // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = product.Phase.canceled), product.Fact.statusCanceled)
      ),
      ExpectedRule(
        activityWorker.respondCanceled.decl,
        None,
        _.phase == product.Phase.started, // scalafix:ok DisableSyntax.==
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
        _.phase == product.Phase.scheduled, // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = product.Phase.paused), product.Fact.statusPaused)
      ),
      ExpectedRule(
        client.pause.decl,
        None,
        _.phase == product.Phase.started, // scalafix:ok DisableSyntax.==
        s =>
          accepted(s.copy(phase = product.Phase.paused), product.Fact.statusPaused) ++
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
        s => s.phase == product.Phase.paused || s.phase == product.Phase.cancelRequested, // scalafix:ok DisableSyntax.==
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
        _.phase == product.Phase.paused, // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = product.Phase.scheduled), product.Fact.statusScheduled)
      ),
      ExpectedRule(
        client.unpause.decl,
        None,
        _.phase == product.Phase.started, // scalafix:ok DisableSyntax.==
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
        s => s.phase == product.Phase.scheduled || s.phase == product.Phase.cancelRequested, // scalafix:ok DisableSyntax.==
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
        s => s.phase == product.Phase.scheduled || s.phase == product.Phase.paused, // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = product.Phase.canceled), product.Fact.statusCanceled)
      ),
      ExpectedRule(
        client.requestCancel.decl,
        None,
        _.phase == product.Phase.started, // scalafix:ok DisableSyntax.==
        s =>
          accepted(
            s.copy(phase = product.Phase.cancelRequested),
            product.Fact.statusCancelRequested
          )
      ),
      ExpectedRule(
        client.requestCancel.decl,
        None,
        _.phase == product.Phase.cancelRequested, // scalafix:ok DisableSyntax.==
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
        s => accepted(s.copy(phase = product.Phase.terminated), product.Fact.statusTerminated)
      ),
      ExpectedRule(
        timers.timeout.decl,
        None,
        s => productLive(s.phase),
        s => accepted(s.copy(phase = product.Phase.timedOut), product.Fact.statusTimedOut)
      )
    ) ++ List[ExpectedRule[product.State, product.Fact]](
      ExpectedRule(
        activityWorker.heartbeat.decl,
        None,
        s => s.phase == product.Phase.started || s.phase == product.Phase.cancelRequested, // scalafix:ok DisableSyntax.==
        s => accepted(s, product.Fact.heartbeatReceived)
      ),
      ExpectedRule(
        activityWorker.heartbeat.decl,
        None,
        s => s.phase != product.Phase.started && s.phase != product.Phase.cancelRequested, // scalafix:ok DisableSyntax.!=
        rejectedNotFound
      ),
      ExpectedRule(
        deadline.heartbeat.decl,
        None,
        _.phase == product.Phase.started, // scalafix:ok DisableSyntax.==
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
        _.phase == product.Phase.cancelRequested, // scalafix:ok DisableSyntax.==
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
      ExpectedRule(client.reset.decl, None, s => closedProduct(s.phase), rejectedNotFound),
      ExpectedRule(
        client.reset.decl,
        None,
        _.phase == product.Phase.scheduled, // scalafix:ok DisableSyntax.==
        s => accepted(s, product.Fact.statusScheduled)
      ),
      ExpectedRule(
        client.reset.decl,
        None,
        _.phase == product.Phase.cancelRequested, // scalafix:ok DisableSyntax.==
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
        _.phase == product.Phase.paused, // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = product.Phase.scheduled), product.Fact.statusScheduled)
      ),
      ExpectedRule(
        client.reset.decl,
        Some(List(ResetPause.resume)),
        _.phase == product.Phase.started, // scalafix:ok DisableSyntax.==
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
        _.phase == product.Phase.paused, // scalafix:ok DisableSyntax.==
        s => accepted(s, product.Fact.statusPaused)
      ),
      ExpectedRule(
        client.reset.decl,
        Some(List(ResetPause.keepPaused)),
        _.phase == product.Phase.started, // scalafix:ok DisableSyntax.==
        s =>
          accepted(s, product.Fact.statusStarted) ++ accepted(s, product.Fact.statusPaused) ++
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
    val unheld = List(completionClient.respondFailedByID, completionClient.respondCanceledByID)
      .map(a =>
        ExpectedRule[product.State, product.Fact](
          a.decl,
          None,
          s =>
            s.phase == product.Phase.scheduled || s.phase == product.Phase.paused, // scalafix:ok DisableSyntax.==
          rejectedNotFound
        )
      )
    closed ++ unheld ++ List[ExpectedRule[product.State, product.Fact]](
      ExpectedRule(
        completionClient.respondCompletedByID.decl,
        None,
        s => productLive(s.phase),
        s => accepted(s.copy(phase = product.Phase.completed), product.Fact.statusCompleted)
      ),
      ExpectedRule(
        completionClient.respondFailedByID.decl,
        Some(List(Failure.fatal)),
        _.phase == product.Phase.started, // scalafix:ok DisableSyntax.==
        s =>
          accepted(s.copy(phase = product.Phase.failed), product.Fact.statusFailed) ++
            accepted(s.copy(phase = product.Phase.scheduled), product.Fact.statusScheduled) ++
            accepted(s.copy(phase = product.Phase.paused), product.Fact.statusPaused)
      ),
      ExpectedRule(
        completionClient.respondFailedByID.decl,
        Some(List(Failure.fatal)),
        _.phase == product.Phase.cancelRequested, // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = product.Phase.failed), product.Fact.statusFailed)
      ),
      ExpectedRule(
        completionClient.respondFailedByID.decl,
        Some(List(Failure.retryable)),
        _.phase == product.Phase.started, // scalafix:ok DisableSyntax.==
        s =>
          accepted(s.copy(phase = product.Phase.scheduled), product.Fact.statusScheduled) ++
            accepted(s.copy(phase = product.Phase.paused), product.Fact.statusPaused) ++
            accepted(s.copy(phase = product.Phase.failed), product.Fact.statusFailed)
      ),
      ExpectedRule(
        completionClient.respondFailedByID.decl,
        Some(List(Failure.retryable)),
        _.phase == product.Phase.cancelRequested, // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = product.Phase.canceled), product.Fact.statusCanceled)
      ),
      ExpectedRule(
        completionClient.respondCanceledByID.decl,
        None,
        _.phase == product.Phase.cancelRequested, // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = product.Phase.canceled), product.Fact.statusCanceled)
      ),
      ExpectedRule(
        completionClient.respondCanceledByID.decl,
        None,
        _.phase == product.Phase.started, // scalafix:ok DisableSyntax.==
        s => rejectedBecause(s, "invalidArgument", "cancellation was not requested")
      )
    )

  private def closedProduct(p: product.Phase): Boolean =
    p == product.Phase.completed || p == product.Phase.failed || p == product.Phase.canceled || p == product.Phase.terminated || p == product.Phase.timedOut // scalafix:ok DisableSyntax.==

  private def productLive(p: product.Phase): Boolean =
    p == product.Phase.scheduled || p == product.Phase.started || p == product.Phase.paused || p == product.Phase.cancelRequested // scalafix:ok DisableSyntax.==

  private lazy val systemRules: List[ExpectedRule[system.State, system.Fact]] =
    List[ExpectedRule[system.State, system.Fact]](
      ExpectedRule(
        activityWorker.poll.decl,
        None,
        s => s.phase == system.Phase.scheduled && s.dispatch == system.Dispatch.now, // scalafix:ok DisableSyntax.==
        s =>
          accepted(
            s.copy(
              phase = system.Phase.started,
              attempts = UpTo[2](((s.attempts: Int) + 1).min(2))
            ),
            system.Fact.statusStarted,
            system.Fact.attemptCount
          )
      ),
      ExpectedRule(
        activityWorker.respondCompleted.decl,
        None,
        s => heldSystem(s.phase) || resetSystem(s.phase),
        s => accepted(s.copy(phase = system.Phase.completed), system.Fact.statusCompleted)
      ),
      ExpectedRule(
        activityWorker.respondFailed.decl,
        Some(List(Failure.fatal)),
        s => heldSystem(s.phase),
        s => accepted(s.copy(phase = system.Phase.failed), system.Fact.statusFailed)
      ),
      ExpectedRule(
        activityWorker.respondFailed.decl,
        Some(List(Failure.retryable)),
        s => s.phase == system.Phase.started && retriesRemain(s), // scalafix:ok DisableSyntax.==
        s =>
          acceptedBecause(
            s.copy(phase = system.Phase.scheduled, dispatch = system.Dispatch.backoff),
            "a retryable attempt backs off; the client reads scheduled again",
            system.Fact.statusScheduled,
            system.Fact.attemptCount
          )
      ),
      ExpectedRule(
        activityWorker.respondFailed.decl,
        Some(List(Failure.retryable)),
        _.phase == system.Phase.cancelRequested, // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = system.Phase.canceled), system.Fact.statusCanceled)
      ),
      ExpectedRule(
        activityWorker.respondFailed.decl,
        Some(List(Failure.retryable)),
        s =>
          s.phase == system.Phase.pauseRequested && retriesRemain(
            s
          ), // scalafix:ok DisableSyntax.==
        s =>
          accepted(
            s.copy(phase = system.Phase.paused, dispatch = system.Dispatch.backoff),
            system.Fact.statusPaused,
            system.Fact.attemptCount
          )
      ),
      ExpectedRule(
        activityWorker.respondFailed.decl,
        Some(List(Failure.retryable)),
        s =>
          (s.phase == system.Phase.started || s.phase == system.Phase.pauseRequested) && !retriesRemain(
            s
          ), // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = system.Phase.failed), system.Fact.statusFailed)
      ),
      ExpectedRule(
        activityWorker.respondCanceled.decl,
        None,
        _.phase == system.Phase.cancelRequested, // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = system.Phase.canceled), system.Fact.statusCanceled)
      ),
      ExpectedRule(
        activityWorker.respondCanceled.decl,
        None,
        s =>
          s.phase == system.Phase.started || s.phase == system.Phase.pauseRequested || resetSystem(
            s.phase
          ), // scalafix:ok DisableSyntax.==
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
        _.phase == system.Phase.scheduled, // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = system.Phase.paused), system.Fact.statusPaused)
      ),
      ExpectedRule(
        client.pause.decl,
        None,
        _.phase == system.Phase.started, // scalafix:ok DisableSyntax.==
        s =>
          acceptedBecause(
            s.copy(phase = system.Phase.pauseRequested),
            "the worker learns of the pause on its next heartbeat",
            system.Fact.statusPaused
          )
      ),
      ExpectedRule(
        client.pause.decl,
        None,
        s =>
          s.phase == system.Phase.paused || s.phase == system.Phase.pauseRequested || s.phase == system.Phase.cancelRequested || // scalafix:ok DisableSyntax.==
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
        _.phase == system.Phase.paused, // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = system.Phase.scheduled), system.Fact.statusScheduled)
      ),
      ExpectedRule(
        client.unpause.decl,
        None,
        _.phase == system.Phase.pauseRequested, // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = system.Phase.started), system.Fact.statusStarted)
      ),
      ExpectedRule(
        client.unpause.decl,
        None,
        s =>
          s.phase == system.Phase.scheduled || s.phase == system.Phase.started || s.phase == system.Phase.cancelRequested || // scalafix:ok DisableSyntax.==
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
        s => s.phase == system.Phase.scheduled || s.phase == system.Phase.paused, // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = system.Phase.canceled), system.Fact.statusCanceled)
      ),
      ExpectedRule(
        client.requestCancel.decl,
        None,
        s =>
          s.phase == system.Phase.started || s.phase == system.Phase.pauseRequested || resetSystem(
            s.phase
          ), // scalafix:ok DisableSyntax.==
        s =>
          accepted(s.copy(phase = system.Phase.cancelRequested), system.Fact.statusCancelRequested)
      ),
      ExpectedRule(
        client.requestCancel.decl,
        None,
        _.phase == system.Phase.cancelRequested, // scalafix:ok DisableSyntax.==
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
        s => accepted(s.copy(phase = system.Phase.terminated), system.Fact.statusTerminated)
      ),
      ExpectedRule(process.stop.decl, None, _ => true, s => accepted(s)),
      ExpectedRule(
        timers.startDelay.decl,
        None,
        s =>
          liveSystem(
            s.phase
          ) && s.dispatch == system.Dispatch.startDelay, // scalafix:ok DisableSyntax.==
        s => acceptedBecause(s.copy(dispatch = system.Dispatch.now), nominalTimeWindow)
      ),
      ExpectedRule(
        timers.backoff.decl,
        None,
        s =>
          liveSystem(
            s.phase
          ) && s.dispatch == system.Dispatch.backoff, // scalafix:ok DisableSyntax.==
        s => acceptedBecause(s.copy(dispatch = system.Dispatch.now), nominalTimeWindow)
      ),
      ExpectedRule(
        deadline.scheduleToClose.decl,
        None,
        s =>
          liveSystem(
            s.phase
          ) && s.scheduleToClose == Timeout.expires && s.dispatch != system.Dispatch.startDelay, // scalafix:ok DisableSyntax.==, DisableSyntax.!=
        s =>
          acceptedBecause(
            s.copy(phase = system.Phase.timedOut),
            nominalTimeWindow,
            system.Fact.statusTimedOut(TimeoutType.scheduleToClose)
          )
      ),
      ExpectedRule(
        deadline.scheduleToStart.decl,
        None,
        s =>
          waitingSystem(
            s.phase
          ) && s.scheduleToStart == Timeout.expires && s.dispatch == system.Dispatch.now, // scalafix:ok DisableSyntax.==
        s =>
          acceptedBecause(
            s.copy(phase = system.Phase.timedOut),
            nominalTimeWindow,
            system.Fact.statusTimedOut(TimeoutType.scheduleToStart)
          )
      ),
      ExpectedRule(
        deadline.startToClose.decl,
        None,
        s =>
          heldSystem(
            s.phase
          ) && s.startToClose == Timeout.expires &&
            (s.phase == system.Phase.cancelRequested || !retriesRemain(
              s
            )), // scalafix:ok DisableSyntax.==
        s =>
          acceptedBecause(
            s.copy(phase = system.Phase.timedOut),
            nominalTimeWindow,
            system.Fact.statusTimedOut(TimeoutType.startToClose)
          )
      ),
      ExpectedRule(
        deadline.startToClose.decl,
        None,
        s =>
          s.phase == system.Phase.started && s.startToClose == Timeout.expires && retriesRemain(
            s
          ), // scalafix:ok DisableSyntax.==
        s =>
          acceptedBecause(
            s.copy(phase = system.Phase.scheduled, dispatch = system.Dispatch.backoff),
            nominalTimeWindow,
            system.Fact.statusScheduled,
            system.Fact.attemptCount
          )
      ),
      ExpectedRule(
        deadline.startToClose.decl,
        None,
        s =>
          s.phase == system.Phase.pauseRequested && s.startToClose == Timeout.expires && retriesRemain(
            s
          ), // scalafix:ok DisableSyntax.==
        s =>
          acceptedBecause(
            s.copy(phase = system.Phase.paused, dispatch = system.Dispatch.backoff),
            nominalTimeWindow,
            system.Fact.statusPaused,
            system.Fact.attemptCount
          )
      )
    ) ++ List[ExpectedRule[system.State, system.Fact]](
      ExpectedRule(
        activityWorker.heartbeat.decl,
        None,
        s => heldSystem(s.phase) || resetSystem(s.phase),
        s => accepted(s, system.Fact.heartbeatReceived)
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
            (s.phase == system.Phase.cancelRequested || !retriesRemain(
              s
            )), // scalafix:ok DisableSyntax.==
        s =>
          acceptedBecause(
            s.copy(phase = system.Phase.timedOut),
            nominalTimeWindow,
            system.Fact.statusTimedOut(TimeoutType.heartbeat),
            system.Fact.heartbeatTimedOut
          )
      ),
      ExpectedRule(
        deadline.heartbeat.decl,
        None,
        s =>
          s.phase == system.Phase.started && s.heartbeat == Timeout.expires && retriesRemain(
            s
          ), // scalafix:ok DisableSyntax.==
        s =>
          acceptedBecause(
            s.copy(phase = system.Phase.scheduled, dispatch = system.Dispatch.backoff),
            nominalTimeWindow,
            system.Fact.statusScheduled,
            system.Fact.attemptCount,
            system.Fact.heartbeatTimedOut
          )
      ),
      ExpectedRule(
        deadline.heartbeat.decl,
        None,
        s =>
          s.phase == system.Phase.pauseRequested && s.heartbeat == Timeout.expires && retriesRemain(
            s
          ), // scalafix:ok DisableSyntax.==
        s =>
          acceptedBecause(
            s.copy(phase = system.Phase.paused, dispatch = system.Dispatch.backoff),
            nominalTimeWindow,
            system.Fact.statusPaused,
            system.Fact.attemptCount,
            system.Fact.heartbeatTimedOut
          )
      )
    ) ++ startRules ++ systemServiceRules ++ systemResetRules

  // A reset applies at once to a waiting or paused activity, is recorded until a held attempt ends,
  // and then applies on its failure or its own deadline (operator_commands.go:428-563).
  private def systemResetRules: List[ExpectedRule[system.State, system.Fact]] =
    val learns = "the worker learns of the reset on its next heartbeat"
    val ends = "the deferred reset applies as the attempt ends"
    def cleared(s: system.State, phase: system.Phase) = s.copy(
      phase = phase,
      attempts = UpTo(0),
      dispatch =
        if s.dispatch == system.Dispatch.backoff then system.Dispatch.now
        else s.dispatch // scalafix:ok DisableSyntax.==
    )
    def restarted(s: system.State) =
      s.copy(
        phase =
          if s.phase == system.Phase.resetKeepingPause then system.Phase.paused
          else system.Phase.scheduled, // scalafix:ok DisableSyntax.==
        attempts = UpTo(0),
        dispatch = system.Dispatch.now
      )
    def landingFact(s: system.State) =
      if s.phase == system.Phase.resetKeepingPause then system.Fact.statusPaused
      else system.Fact.statusScheduled // scalafix:ok DisableSyntax.==
    List[ExpectedRule[system.State, system.Fact]](
      ExpectedRule(
        client.reset.decl,
        None,
        s =>
          s.phase == system.Phase.unstarted || closedSystem(
            s.phase
          ), // scalafix:ok DisableSyntax.==
        rejectedNotFound
      ),
      ExpectedRule(
        client.reset.decl,
        None,
        _.phase == system.Phase.scheduled, // scalafix:ok DisableSyntax.==
        s => accepted(cleared(s, system.Phase.scheduled), system.Fact.statusScheduled)
      ),
      ExpectedRule(
        client.reset.decl,
        None,
        _.phase == system.Phase.started, // scalafix:ok DisableSyntax.==
        s =>
          acceptedBecause(
            s.copy(phase = system.Phase.resetRequested),
            learns,
            system.Fact.statusStarted
          )
      ),
      ExpectedRule(
        client.reset.decl,
        None,
        _.phase == system.Phase.cancelRequested, // scalafix:ok DisableSyntax.==
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
        _.phase == system.Phase.paused, // scalafix:ok DisableSyntax.==
        s => accepted(cleared(s, system.Phase.scheduled), system.Fact.statusScheduled)
      ),
      ExpectedRule(
        client.reset.decl,
        Some(List(ResetPause.resume)),
        _.phase == system.Phase.pauseRequested, // scalafix:ok DisableSyntax.==
        s =>
          acceptedBecause(
            s.copy(phase = system.Phase.resetRequested),
            learns,
            system.Fact.statusStarted
          )
      ),
      ExpectedRule(
        client.reset.decl,
        Some(List(ResetPause.keepPaused)),
        _.phase == system.Phase.paused, // scalafix:ok DisableSyntax.==
        s => accepted(cleared(s, system.Phase.paused), system.Fact.statusPaused)
      ),
      ExpectedRule(
        client.reset.decl,
        Some(List(ResetPause.keepPaused)),
        _.phase == system.Phase.pauseRequested, // scalafix:ok DisableSyntax.==
        s =>
          acceptedBecause(
            s.copy(phase = system.Phase.resetKeepingPause),
            learns,
            system.Fact.statusPaused
          )
      )
    ) ++ List(activityWorker.respondFailed, completionClient.respondFailedByID).map(a =>
      ExpectedRule[system.State, system.Fact](
        a.decl,
        None,
        s => resetSystem(s.phase),
        s => acceptedBecause(restarted(s), ends, landingFact(s), system.Fact.attemptCount)
      )
    ) ++ List[ExpectedRule[system.State, system.Fact]](
      ExpectedRule(
        deadline.startToClose.decl,
        None,
        s =>
          resetSystem(s.phase) && s.startToClose == Timeout.expires, // scalafix:ok DisableSyntax.==
        s => acceptedBecause(restarted(s), ends, landingFact(s), system.Fact.attemptCount)
      ),
      ExpectedRule(
        deadline.heartbeat.decl,
        None,
        s => resetSystem(s.phase) && s.heartbeat == Timeout.expires, // scalafix:ok DisableSyntax.==
        s =>
          acceptedBecause(
            restarted(s),
            nominalTimeWindow,
            landingFact(s),
            system.Fact.attemptCount,
            system.Fact.heartbeatTimedOut
          )
      )
    )

  private def systemServiceRules: List[ExpectedRule[system.State, system.Fact]] =
    val absentOrClosed =
      List(
        completionClient.respondCompletedByID,
        completionClient.respondFailedByID,
        completionClient.respondCanceledByID
      )
        .map(a =>
          ExpectedRule[system.State, system.Fact](
            a.decl,
            None,
            s =>
              s.phase == system.Phase.unstarted || closedSystem(
                s.phase
              ), // scalafix:ok DisableSyntax.==
            rejectedNotFound
          )
        )
    val unheld = List(completionClient.respondFailedByID, completionClient.respondCanceledByID)
      .map(a =>
        ExpectedRule[system.State, system.Fact](
          a.decl,
          None,
          s =>
            s.phase == system.Phase.scheduled || s.phase == system.Phase.paused, // scalafix:ok DisableSyntax.==
          rejectedNotFound
        )
      )
    absentOrClosed ++ unheld ++ List[ExpectedRule[system.State, system.Fact]](
      ExpectedRule(
        completionClient.respondCompletedByID.decl,
        None,
        s => liveSystem(s.phase),
        s => accepted(s.copy(phase = system.Phase.completed), system.Fact.statusCompleted)
      ),
      ExpectedRule(
        completionClient.respondFailedByID.decl,
        Some(List(Failure.fatal)),
        s => heldSystem(s.phase),
        s => accepted(s.copy(phase = system.Phase.failed), system.Fact.statusFailed)
      ),
      ExpectedRule(
        completionClient.respondFailedByID.decl,
        Some(List(Failure.retryable)),
        s => s.phase == system.Phase.started && retriesRemain(s), // scalafix:ok DisableSyntax.==
        s =>
          acceptedBecause(
            s.copy(phase = system.Phase.scheduled, dispatch = system.Dispatch.backoff),
            "a retryable attempt backs off; the client reads scheduled again",
            system.Fact.statusScheduled,
            system.Fact.attemptCount
          )
      ),
      ExpectedRule(
        completionClient.respondFailedByID.decl,
        Some(List(Failure.retryable)),
        s =>
          s.phase == system.Phase.pauseRequested && retriesRemain(
            s
          ), // scalafix:ok DisableSyntax.==
        s =>
          accepted(
            s.copy(phase = system.Phase.paused, dispatch = system.Dispatch.backoff),
            system.Fact.statusPaused,
            system.Fact.attemptCount
          )
      ),
      ExpectedRule(
        completionClient.respondFailedByID.decl,
        Some(List(Failure.retryable)),
        s =>
          (s.phase == system.Phase.started || s.phase == system.Phase.pauseRequested) && !retriesRemain(
            s
          ), // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = system.Phase.failed), system.Fact.statusFailed)
      ),
      ExpectedRule(
        completionClient.respondFailedByID.decl,
        Some(List(Failure.retryable)),
        _.phase == system.Phase.cancelRequested, // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = system.Phase.canceled), system.Fact.statusCanceled)
      ),
      ExpectedRule(
        completionClient.respondCanceledByID.decl,
        None,
        _.phase == system.Phase.cancelRequested, // scalafix:ok DisableSyntax.==
        s => accepted(s.copy(phase = system.Phase.canceled), system.Fact.statusCanceled)
      ),
      ExpectedRule(
        completionClient.respondCanceledByID.decl,
        None,
        s =>
          s.phase == system.Phase.started || s.phase == system.Phase.pauseRequested || resetSystem(
            s.phase
          ), // scalafix:ok DisableSyntax.==
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
      _.phase == system.Phase.unstarted, // scalafix:ok DisableSyntax.==
      _ =>
        accepted(
          system.State(
            phase = system.Phase.scheduled,
            dispatch = if delay == Timeout.expires then system.Dispatch.startDelay
            else system.Dispatch.now, // scalafix:ok DisableSyntax.==
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
    p == system.Phase.scheduled || p == system.Phase.started || p == system.Phase.paused ||
      p == system.Phase.pauseRequested || p == system.Phase.cancelRequested || resetSystem(
        p
      ) // scalafix:ok DisableSyntax.==

  private def resetSystem(p: system.Phase): Boolean =
    p == system.Phase.resetRequested || p == system.Phase.resetKeepingPause // scalafix:ok DisableSyntax.==

  private def waitingSystem(p: system.Phase): Boolean =
    p == system.Phase.scheduled // scalafix:ok DisableSyntax.==

  private def heldSystem(p: system.Phase): Boolean =
    p == system.Phase.started || p == system.Phase.pauseRequested ||
      p == system.Phase.cancelRequested // scalafix:ok DisableSyntax.==

  private def closedSystem(p: system.Phase): Boolean =
    p == system.Phase.completed || p == system.Phase.failed || p == system.Phase.canceled || p == system.Phase.terminated || p == system.Phase.timedOut // scalafix:ok DisableSyntax.==

  test("the standalone signature declares one action for every RPC") {
    assertEquals(client.start.decl.creates, Some(activity))
    assertEquals(client.start.decl.domains.size, 6)

    val controls = List(client.pause, client.unpause, client.requestCancel, client.terminate)
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
        activityWorker.poll,
        activityWorker.heartbeat,
        activityWorker.respondCompleted,
        activityWorker.respondFailed,
        activityWorker.respondCanceled,
        completionClient.respondCompletedByID,
        completionClient.respondFailedByID,
        completionClient.respondCanceledByID,
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
        activityWorker.poll,
        activityWorker.heartbeat,
        activityWorker.respondCompleted,
        activityWorker.respondFailed,
        activityWorker.respondCanceled,
        completionClient.respondCompletedByID,
        completionClient.respondFailedByID,
        completionClient.respondCanceledByID,
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
      .find(
        // scalafix:off DisableSyntax.==; strictEquality checks native comparisons
        _.decl == temporal.foundations.taskqueue.fault.ackLoss.decl
        // scalafix:on DisableSyntax.==
      )
      .get
      .function
      .asInstanceOf[Loss] // scalafix:ok DisableSyntax.asInstanceOf
    assertEquals(loss(LostStartAnswer.init), choices)
    for choice <- choices do assertEquals(loss(choice.state), Nil)
    assert(choices.forall(c => LostStartAnswer.end(c.state)))
    assertEquals(
      choices.map(_.facts),
      List(List(system.AdmissionResponseFact.attemptAdmitted), Nil)
    )
  }
