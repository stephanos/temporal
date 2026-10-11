// The standalone activity Protocol's Reset facet: a reset applies at once to a waiting or paused
// activity and waits for a held attempt to end, with its settlement witnesses.
package temporal
package features.activity
package standalone
package system

import framework.*
import framework.Finite.given
import framework.outcomes.{Outcome, Rejection}
import temporal.realize.satisfied
import Bounds.{five, four, six, three}
import Timeout.expires

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

trait Reset extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  this: Completion & Cancellation & Timeouts =>
  object reset:
    object effects:
      // A reset of a waiting or paused activity applies at once: its independent count restarts at
      // zero and a retry backoff is discarded (operator_commands.go:521-563).
      def resetWaiting(s: State) =
        enter(
          s.copy(
            phase = Phase.scheduled,
            attempts = UpTo(0),
            dispatch = ActivitySystem.states.resetDispatch(s.dispatch)
          ),
          Fact.statusScheduled
        )
      def resetPaused(s: State) =
        enter(
          s.copy(
            phase = Phase.paused,
            attempts = UpTo(0),
            dispatch = ActivitySystem.states.resetDispatch(s.dispatch)
          ),
          Fact.statusPaused
        )

      // A reset of a held attempt is recorded until the attempt ends; Describe still reads the
      // attempt as started, or pause-requested when the pause is kept (responses.go:55-62).
      def requestReset(s: State) =
        enter(s.copy(phase = Phase.resetRequested), Fact.statusStarted)
          .because("the worker learns of the reset on its next heartbeat")
      def requestResetKeepingPause(s: State) =
        enter(s.copy(phase = Phase.resetKeepingPause), Fact.statusPaused)
          .because("the worker learns of the reset on its next heartbeat")

      // A pending reset applies when the held attempt fails or one of its own deadlines ends it,
      // whatever the failure's retryability or the policy left: the next attempt is a first one,
      // dispatched at once (attempt.go:201-216, statemachine.go:317-345).
      def applyReset(s: State) =
        enter(
          s.copy(
            phase = ActivitySystem.states.landingPhase(s),
            attempts = UpTo(0),
            dispatch = Dispatch.now
          ),
          ActivitySystem.states.landingStatus(s),
          Fact.attemptCount
        )
          .because("the deferred reset applies as the attempt ends")

    // Cancel > Reset > Pause: a reset replaces a requested pause, and does not undo a cancellation
    // (model.go:156-158).
    object rules extends Rules:
      on(client.reset) {
        when(Phase.unstarted) ~> rejects(Rejection.notFound)
        when[Closed] ~> rejects(Rejection.notFound)
        when(Phase.scheduled) ~> effects.resetWaiting
        when(Phase.started) ~> effects.requestReset
        when(Phase.cancelRequested) ~> rejects(Rejection.failedPrecondition)
          .because(resetWithPendingCancellation)
        when(Phase.resetRequested, Phase.resetKeepingPause) ~> rejects(Rejection.failedPrecondition)
          .because(resetWithPendingReset)
      }
      on(client.reset(ResetPause.resume)) {
        when(Phase.paused) ~> effects.resetWaiting
        when(Phase.pauseRequested) ~> effects.requestReset
      }
      on(client.reset(ResetPause.keepPaused)) {
        when(Phase.paused) ~> effects.resetPaused
        when(Phase.pauseRequested) ~> effects.requestResetKeepingPause
      }

    // Model-only witnesses of a pending reset: it applies on a fatal failure, on an exhausted policy
    // and on an attempt's own deadline, keeps a requested pause, loses to a completion, a
    // cancellation and schedule-to-close, and outranks a pause and a repeated reset. No realization
    // runs them. A reset with keep_paused of an activity paused before any worker took it stays
    // paused, its count restarted: it leaves the state the pause made, so no Contract clause tells
    // the reset's step from the pause's, and it stays a Model-only witness.
    object properties:
      // A first attempt again, dispatched at once, of an activity with no deadline set.
      val restarted =
        ActivitySystem.init.copy(phase = Phase.scheduled)
      val resetOnFatal = property when activityWorker.respondFailed(Failure.fatal) holds {
        s => // format: off
        s.state == restarted && s.records(Fact.statusScheduled) && !s.records(Fact.statusFailed) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
      }
      val resetOnExhaustion = property when activityWorker.respondFailed(Failure.retryable) holds { s =>
        s.state == restarted.copy(maxAttempts = MaxAttempts.one) && s.records(Fact.statusScheduled) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
      } // format: on
      val resetOnTimeout = property when deadline.startToClose holds { s => // format: off
        s.state == restarted.copy(startToClose = Timeout.expires, maxAttempts = MaxAttempts.one) && // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        s.records(Fact.statusScheduled) && // format: on
        !s.records(Fact.statusTimedOut(TimeoutType.startToClose))
      } // format: off
      val resetKeepsPause = property when activityWorker.respondFailed(Failure.retryable) holds { s =>
        s.state == restarted.copy(phase = Phase.paused) && s.records(Fact.statusPaused) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
      } // format: on
      val completionWins = property when activityWorker.respondCompleted holds { s =>
        s.state.phase == Phase.completed && s.state.attempts == 1 && // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        s.records(Fact.statusCompleted)
      }
      val pauseLeavesResetPending = property when client.pause holds { s => // format: off
        s.state.phase == Phase.resetRequested && !s.records(Fact.statusPaused) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
      } // format: on
      val resetStaysPending = property when client.reset(ResetPause.resume) holds { s =>
        s.state.phase == Phase.resetRequested // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
      }

      // Paused before any attempt, with no deadline set, its count restarted.
      val keptPaused =
        ActivitySystem.init.copy(phase = Phase.paused) // format: off
      val resetKeptPaused = property when client.reset(ResetPause.keepPaused) holds { s =>
        s.state == keptPaused && s.records(Fact.statusPaused) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
      } // format: on

    object queries:
      val resetThenFatal = scenario.actions(
        client.start(),
        activityWorker.poll,
        client.reset(ResetPause.resume),
        activityWorker.respondFailed(Failure.fatal)
      )
      val resetThenExhausted = scenario.actions(
        client.start(maxAttempts := MaxAttempts.one),
        activityWorker.poll,
        client.reset(ResetPause.resume),
        activityWorker.respondFailed(Failure.retryable)
      )
      val resetThenTimedOut = scenario.actions(
        client.start(startToClose := expires, maxAttempts := MaxAttempts.one),
        activityWorker.poll,
        client.reset(ResetPause.resume),
        deadline.startToClose
      )
      val pausedResetThenFailed = scenario.actions(
        client.start(),
        activityWorker.poll,
        client.pause,
        client.reset(ResetPause.keepPaused),
        activityWorker.respondFailed(Failure.retryable)
      )
      val resetThenCompleted = scenario.actions(
        client.start(),
        activityWorker.poll,
        client.reset(ResetPause.resume),
        activityWorker.respondCompleted
      )
      val resetThenScheduleToClose = scenario.actions(
        client.start(scheduleToClose := expires),
        activityWorker.poll,
        client.reset(ResetPause.resume),
        deadline.scheduleToClose
      )
      val resetThenCanceled = scenario.actions(
        client.start(),
        activityWorker.poll,
        client.reset(ResetPause.resume),
        client.requestCancel
      )
      val resetThenPaused = scenario.actions(
        client.start(),
        activityWorker.poll,
        client.reset(ResetPause.resume),
        client.pause
      )
      val resetTwice = scenario.actions(
        client.start(),
        activityWorker.poll,
        client.reset(ResetPause.resume),
        client.reset(ResetPause.resume)
      )
      val pausedThenReset = scenario.actions(
        client.start(),
        client.pause,
        client.reset(ResetPause.keepPaused)
      )
      // A reset of a held attempt, applied when the attempt's heartbeat deadline ends it: the
      // server's next delivery is a first attempt again, which completes, though the policy allowed
      // one attempt.
      val resetThenHeartbeatTimedOut = scenario.actions(
        client.start(heartbeat := expires, maxAttempts := MaxAttempts.one),
        activityWorker.poll,
        client.reset(ResetPause.resume),
        deadline.heartbeat,
        activityWorker.poll,
        activityWorker.respondCompleted
      )
      val resetFatality = query find properties.resetOnFatal in resetThenFatal limits four
      val resetExhaustion = query find properties.resetOnExhaustion in resetThenExhausted limits four
      val resetTimeout = query find properties.resetOnTimeout in resetThenTimedOut limits four
      val resetKeptPause = query find properties.resetKeepsPause in pausedResetThenFailed limits five
      val resetCompletion = query find properties.completionWins in resetThenCompleted limits four
      val resetScheduleToClose =
        query find timeouts.properties.scheduleToCloseTimesOut in resetThenScheduleToClose limits four
      val resetCancellation =
        query find cancellation.properties.cancelRequested in resetThenCanceled limits four
      val resetOutranksPause =
        query find properties.pauseLeavesResetPending in resetThenPaused limits four
      val resetRepeated = query find properties.resetStaysPending in resetTwice limits four
      val keepPausedReset = query find properties.resetKeptPaused in pausedThenReset limits three
      val deferredResetCompletes =
        (query find completion.properties.completes in resetThenHeartbeatTimedOut limits six)
          .expect(satisfied)
