// Standalone activity terminal deadlines, including both orders of competing scheduling deadlines.
package temporal
package features.activity
package standalone
package system

import framework.*
import framework.realize.Reason
import temporal.realize.inconclusive
import Bounds.three
import actors.worker.worker as process
import Timeout.expires

object Timeouts extends Derived(ActivitySystem.rebind()):
  object properties:
    // Terminal deadline witnesses record which deadline settled the activity. Start-to-close
    // uses a one-attempt start.
    val scheduleToStartFires = property when deadline.scheduleToStart holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(Fact.statusTimedOut(TimeoutType.scheduleToStart))
    }

    val startToCloseFires = property when deadline.startToClose holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(Fact.statusTimedOut(TimeoutType.startToClose))
    }

  object queries:
    val scheduleToStartExpires = scenario.actions(
      client.start(scheduleToStart := expires),
      process.stop,
      deadline.scheduleToStart
    )
    val startToCloseExpires = scenario.actions(
      client.start(startToClose := expires, maxAttempts := MaxAttempts.one),
      worker.poll,
      deadline.startToClose
    )

    val scheduleToStartTimeout =
      (query find properties.scheduleToStartFires in scheduleToStartExpires limits three)
        .expect(inconclusive(Reason.neverEvaluated))
    val startToCloseTimeout =
      query find properties.startToCloseFires in startToCloseExpires limits three

object CompetingTimeouts extends Derived(ActivitySystem.rebind()):
  object properties:
    val scheduleToStartFires = property when deadline.scheduleToStart holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(Fact.statusTimedOut(TimeoutType.scheduleToStart))
    }

    val scheduleToCloseFires = property when deadline.scheduleToClose holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(Fact.statusTimedOut(TimeoutType.scheduleToClose))
    }

  object queries:
    // Neither deadline is ordered before the other, so each firing is a trace of its own. Both
    // deadlines are set and no attempt started, so either may fire first.
    val bothDeadlinesStartFirst = scenario.actions(
      client.start(scheduleToClose := Timeout.expires, scheduleToStart := Timeout.expires),
      deadline.scheduleToStart
    )
    val bothDeadlinesCloseFirst = scenario.actions(
      client.start(scheduleToClose := Timeout.expires, scheduleToStart := Timeout.expires),
      deadline.scheduleToClose
    )

    val competingTimers = Vector(
      query("competingTimers.scheduleToStartFirst") find
        properties.scheduleToStartFires in
        bothDeadlinesStartFirst limits three,
      query("competingTimers.scheduleToCloseFirst") find
        properties.scheduleToCloseFires in
        bothDeadlinesCloseFirst limits three
    )
