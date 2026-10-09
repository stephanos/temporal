// Standalone activity reset settlement and deferred reset witnesses.
package temporal
package features.activity
package standalone
package system

import framework.*
import temporal.realize.satisfied
import Bounds.{four, three}
import Timeout.expires

// Model-only witnesses of a pending reset: it applies on a fatal failure, on an exhausted policy
// and on an attempt's own deadline, keeps a requested pause, loses to a completion, a cancellation
// and schedule-to-close, and outranks a pause and a repeated reset. No realization runs them.
object ResetSettlement extends Derived(ActivitySystem.unmonitored):
  object properties:
    // A first attempt again, dispatched at once, of an activity with no deadline set.
    val restarted =
      system.State(
        phase = Phase.scheduled,
        dispatch = Dispatch.now,
        attempts = UpTo(0),
        scheduleToClose = Timeout.unset,
        scheduleToStart = Timeout.unset,
        startToClose = Timeout.unset,
        heartbeat = Timeout.unset,
        maxAttempts = MaxAttempts.unlimited
      )
    val resetOnFatal = property when worker.respondFailed(Failure.fatal) holds { s =>
      s.state == restarted && s.records(Fact.statusScheduled) && !s.records(Fact.statusFailed)
    }
    val resetOnExhaustion = property when worker.respondFailed(Failure.retryable) holds { s =>
      s.state == restarted.copy(maxAttempts = MaxAttempts.one) && s.records(Fact.statusScheduled)
    }
    val resetOnTimeout = property when deadline.startToClose holds { s =>
      s.state == restarted.copy(startToClose = Timeout.expires, maxAttempts = MaxAttempts.one) &&
      s.records(Fact.statusScheduled) &&
      !s.records(Fact.statusTimedOut(TimeoutType.startToClose))
    }
    val resetKeepsPause = property when worker.respondFailed(Failure.retryable) holds { s =>
      s.state == restarted.copy(phase = Phase.paused) && s.records(Fact.statusPaused)
    }
    val completionWins = property when worker.respondCompleted holds { s =>
      s.state.phase == Phase.completed && s.state.attempts == 1 &&
      s.records(Fact.statusCompleted)
    }
    val scheduleToCloseStaysTerminal = property when deadline.scheduleToClose holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(Fact.statusTimedOut(TimeoutType.scheduleToClose))
    }
    val cancellationReplacesReset = property when client.requestCancel holds { s =>
      s.state.phase == Phase.cancelRequested && s.records(Fact.statusCancelRequested)
    }
    val pauseLeavesResetPending = property when client.pause holds { s =>
      s.state.phase == Phase.resetRequested && !s.records(Fact.statusPaused)
    }
    val resetStaysPending = property when client.reset(ResetPause.resume) holds { s =>
      s.state.phase == Phase.resetRequested
    }
  object queries:
    val resetThenFatal = scenario.actions(
      client.start(),
      worker.poll,
      client.reset(ResetPause.resume),
      worker.respondFailed(Failure.fatal)
    )
    val resetThenExhausted = scenario.actions(
      client.start(maxAttempts := MaxAttempts.one),
      worker.poll,
      client.reset(ResetPause.resume),
      worker.respondFailed(Failure.retryable)
    )
    val resetThenTimedOut = scenario.actions(
      client.start(startToClose := expires, maxAttempts := MaxAttempts.one),
      worker.poll,
      client.reset(ResetPause.resume),
      deadline.startToClose
    )
    val pausedResetThenFailed = scenario.actions(
      client.start(),
      worker.poll,
      client.pause,
      client.reset(ResetPause.keepPaused),
      worker.respondFailed(Failure.retryable)
    )
    val resetThenCompleted = scenario.actions(
      client.start(),
      worker.poll,
      client.reset(ResetPause.resume),
      worker.respondCompleted
    )
    val resetThenScheduleToClose = scenario.actions(
      client.start(scheduleToClose := expires),
      worker.poll,
      client.reset(ResetPause.resume),
      deadline.scheduleToClose
    )
    val resetThenCanceled = scenario.actions(
      client.start(),
      worker.poll,
      client.reset(ResetPause.resume),
      client.requestCancel
    )
    val resetThenPaused = scenario.actions(
      client.start(),
      worker.poll,
      client.reset(ResetPause.resume),
      client.pause
    )
    val resetTwice = scenario.actions(
      client.start(),
      worker.poll,
      client.reset(ResetPause.resume),
      client.reset(ResetPause.resume)
    )
    val resetFatality = query find properties.resetOnFatal in resetThenFatal limits four
    val resetExhaustion = query find properties.resetOnExhaustion in resetThenExhausted limits four
    val resetTimeout = query find properties.resetOnTimeout in resetThenTimedOut limits four
    val resetKeptPause = query find properties.resetKeepsPause in pausedResetThenFailed limits five
    val resetCompletion = query find properties.completionWins in resetThenCompleted limits four
    val resetScheduleToClose =
      query find properties.scheduleToCloseStaysTerminal in resetThenScheduleToClose limits four
    val resetCancellation =
      query find properties.cancellationReplacesReset in resetThenCanceled limits four
    val resetOutranksPause = query find properties.pauseLeavesResetPending in resetThenPaused limits
      four
    val resetRepeated = query find properties.resetStaysPending in resetTwice limits four

// A reset with keep_paused of an activity paused before any worker took it: it stays paused, its
// count restarted. It leaves the state the pause made, so no Contract clause tells the reset's step
// from the pause's, and it stays a Model-only witness; a retried activity's reset would rewind a
// delivered count, which only a held reset declares to the Driver.
object ResetKeepingPause extends Derived(ActivitySystem.unmonitored):
  object properties:
    // Paused before any attempt, with no deadline set, its count restarted.
    val keptPaused =
      system.State(
        phase = Phase.paused,
        dispatch = Dispatch.now,
        attempts = UpTo(0),
        scheduleToClose = Timeout.unset,
        scheduleToStart = Timeout.unset,
        startToClose = Timeout.unset,
        heartbeat = Timeout.unset,
        maxAttempts = MaxAttempts.unlimited
      )
    val resetKeptPaused = property when client.reset(ResetPause.keepPaused) holds { s =>
      s.state == keptPaused && s.records(Fact.statusPaused)
    }
  object queries:
    val pausedThenReset = scenario.actions(
      client.start(),
      client.pause,
      client.reset(ResetPause.keepPaused)
    )
    val keepPausedReset = query find properties.resetKeptPaused in pausedThenReset limits three

// A reset of a held attempt, applied when the attempt's heartbeat deadline ends it: the server's
// next delivery is a first attempt again, which completes, though the policy allowed one attempt.
object DeferredReset extends Derived(ActivitySystem.unmonitored):
  object properties:
    val resetAttemptCompletes = property when worker.respondCompleted holds { s =>
      s.state.phase == Phase.completed && s.records(Fact.statusCompleted)
    }
  object queries:
    val resetThenHeartbeatTimedOut = scenario.actions(
      client.start(heartbeat := expires, maxAttempts := MaxAttempts.one),
      worker.poll,
      client.reset(ResetPause.resume),
      deadline.heartbeat,
      worker.poll,
      worker.respondCompleted
    )
    val deferredResetCompletes =
      (query find properties.resetAttemptCompletes in resetThenHeartbeatTimedOut limits six)
        .expect(satisfied)
