// API behavior hints and server steps, each written at the line that declares it (fn-118.2).
//
// `keptBehavior` keeps the kit's behavior and declares the steps no command performs: an attempt
// start is a delivery, a timeout a timer at the kit's deadline. `ownBehavior` declares a behavior
// of its own: a call visible to a read at once, a cause visible to it eventually within its bound,
// a hint declared by a val and named by value, which is written at the val's line, an eventual
// visibility held in a val, and a bound for each kind of server step it declares.
//
// The realizations under "Refused by the reader" lift, and the Go reader refuses each at its line
// (tools/umpire/ir/hints_fixture_test.go): a non-positive interval or bound, an interval
// greater than its bound, a server step of a kind the behavior bounds not, a timer step with no
// deadline and a deadline on a step that is no timer.
package fixture.hints

import umpire.realize.*
import temporal.realize.*
import temporal.features.activity.standalone.{activity, deadline, worker}
import temporal.features.activity.standalone.system.ActivitySystem as activitySystem
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.*

private def realizing(serverSteps: Vector[ServerStep], behavior: ApiBehavior) =
  temporalRealization(
    machine = activitySystem,
    operation = activity,
    roles = Vector(workflowService, taskQueue),
    scripts = Vector(controller()),
    evidence = Vector.empty,
    serverSteps = serverSteps,
    behavior = behavior
  )

/** The kit's behavior, which `temporalRealization` attaches, and a delivery and a timer step. */
val keptBehavior: Realization = temporalRealization(
  machine = activitySystem,
  operation = activity,
  roles = Vector(workflowService, taskQueue),
  scripts = Vector(controller()),
  evidence = Vector.empty,
  serverSteps = Vector(
    ServerStep(worker.poll, CauseKind.delivery),
    ServerStep(deadline.scheduleToStart, CauseKind.timer, deadlineMs)
  )
)

// Named by value below, and written here.
private val pauseSeen =
  METHOD_PAUSE_ACTIVITY_EXECUTION.visibleTo(METHOD_DESCRIBE_ACTIVITY_EXECUTION, Visible.atOnce)

private val slowly = Visible.eventually(WaitBound(intervalMs = 100, atMostMs = 1500))

/** A behavior of its own, which bounds each kind of its server steps. */
val ownBehavior: Realization = realizing(
  Vector(
    ServerStep(worker.poll, CauseKind.delivery),
    ServerStep(deadline.scheduleToStart, CauseKind.timer, deadlineMs)
  ),
  ApiBehavior(
    visibility = Vector(
      METHOD_START_ACTIVITY_EXECUTION.visibleTo(METHOD_DESCRIBE_ACTIVITY_EXECUTION, Visible.atOnce),
      CauseKind.activityAnswer.visibleTo(
        METHOD_DESCRIBE_ACTIVITY_EXECUTION,
        Visible.eventually(WaitBound(intervalMs = 50, atMostMs = 500))
      ),
      pauseSeen,
      METHOD_TERMINATE_ACTIVITY_EXECUTION.visibleTo(METHOD_DESCRIBE_ACTIVITY_EXECUTION, slowly)
    ),
    causes = Vector(
      CauseKind.delivery.boundedBy(WaitBound(intervalMs = 100, atMostMs = 1000)),
      CauseKind.timer.boundedBy(WaitBound(intervalMs = 100, atMostMs = 2000))
    )
  )
)

// ### Refused by the reader

/** A wait that never looks again. */
val zeroInterval: Realization = realizing(
  Vector.empty,
  ApiBehavior(
    visibility = Vector.empty,
    causes = Vector(CauseKind.delivery.boundedBy(WaitBound(intervalMs = 0, atMostMs = 1000)))
  )
)

/** Waits that end before they start: a negative bound and a bound of zero. */
val nonPositiveBound: Realization = realizing(
  Vector.empty,
  ApiBehavior(
    visibility = Vector(
      CauseKind.activityAnswer.visibleTo(
        METHOD_DESCRIBE_ACTIVITY_EXECUTION,
        Visible.eventually(WaitBound(intervalMs = 100, atMostMs = -1))
      )
    ),
    causes = Vector(CauseKind.timer.boundedBy(WaitBound(intervalMs = 100, atMostMs = 0)))
  )
)

/** A wait that looks less often than it lasts. */
val intervalOverBound: Realization = realizing(
  Vector.empty,
  ApiBehavior(
    visibility = Vector.empty,
    causes = Vector(CauseKind.timer.boundedBy(WaitBound(intervalMs = 2000, atMostMs = 1000)))
  )
)

/** A delivery step, and a behavior that bounds only timers. */
val unboundedStep: Realization = realizing(
  Vector(ServerStep(worker.poll, CauseKind.delivery)),
  ApiBehavior(
    visibility = Vector.empty,
    causes = Vector(CauseKind.timer.boundedBy(WaitBound(intervalMs = 100, atMostMs = 2000)))
  )
)

/** A timer step that names no deadline. */
val timerNoDeadline: Realization =
  realizing(Vector(ServerStep(deadline.scheduleToStart, CauseKind.timer)), temporalBehavior)

/** A delivery step that names a deadline, which only a timer has. */
val deliveryDeadline: Realization =
  realizing(
    Vector(ServerStep(worker.poll, CauseKind.delivery, deadlineMs)),
    temporalBehavior
  )
