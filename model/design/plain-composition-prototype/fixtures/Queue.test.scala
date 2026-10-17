package framework.dsl
package compose

import framework.dsl.Finite.given

// SEMANTICS.md §17's own example (activity_mini's `Queue`) cut down: an activity over its task
// queue, where the worker's poll is the queue's delivery. Each action is bound by one member, so the
// sync names the actions alone, and a claim about the poll names the sync's step by the poll.
object Scheduling:
  enum Phase derives CanEqual:
    case scheduled, started

  enum Outcome:
    case accepted

  given Ok[Outcome] = Ok(Outcome.accepted)

  object activityWorker extends Actor:
    val poll = action

  object matching extends Internal:
    val schedule = step
    val deliver = step

  object Activity extends Machine[Outcome, Nothing]:
    var phase = Phase.scheduled

    object transitions extends Transitions:
      on(activityWorker.poll):
        if phase == Phase.scheduled then effects.start()

    object effects:
      def start() =
        phase = Phase.started

  object TaskQueue extends Machine[Outcome, Nothing]:
    var outstanding: UpTo[2] = UpTo(0)

    object transitions extends Transitions:
      on(matching.schedule):
        if outstanding < 2 then effects.schedule()
      on(matching.deliver):
        if outstanding > 0 then effects.deliver()

    object effects:
      def schedule() =
        outstanding = UpTo(outstanding + 1)
      def deliver() =
        outstanding = UpTo(outstanding - 1)

  // Members that record no facts: the composition's fact type is Nothing.
  object Queue extends Composition[Nothing]:
    val activity = Activity
    val queue = TaskQueue

    override def end = activity.phase == Phase.started

    object syncs:
      val admit = sync(activityWorker.poll, matching.deliver)

    object postconditions:
      // the poll steps only with the delivery, so `after(poll)` is about the sync's step
      val deliveredWasScheduled =
        after(activityWorker.poll):
          check(before.queue.outstanding > 0)
          check(queue.outstanding == before.queue.outstanding - 1)

    object queries:
      val startedWithOneLeft =
        after(syncs.admit):
          check(activity.phase == Phase.started && queue.outstanding == 1)
      val startedWithOneLeftFound = find(startedWithOneLeft)
