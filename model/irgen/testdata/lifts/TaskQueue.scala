// An independent consumer of the task queue in temporal/shared/taskqueue: a job of its own, sent once
// and settled once, composed with the opaque queue, with the matching provider that replaces it, and
// with the forgetful provider as the negative control. It imports nothing of the standalone activity,
// so the queue is lifted and checked here as any other feature would rely on it. The lifter's tests
// compare the IR with expected/taskqueue.json, and tools/umpire/model pins every Query's answer.
package fixture.taskqueue

import temporal.shared.taskqueue.*
import temporal.shared.taskqueue.DispatchQueue.dispatchQueue
import temporal.shared.taskqueue.MatchingQueue.{forgetfulQueue, matchingQueue}
import umpire.*

given Family = Family("fixture.taskqueue")

// ### The job

enum JobPhase derives Finite:
  case idle, sent, started, settled

final case class Job(phase: JobPhase) derives Finite

enum JobOutcome derives Finite:
  case accepted

given Ok[JobOutcome] = Ok(JobOutcome.accepted)

type JobStep = Step[Job, JobOutcome, Nothing]

val send = internal
val start = internal
val settle = internal

/**
 * A failed enqueue leaves nothing outstanding, so the job may be sent again; the queue disables the
 * enqueue of a second message while one is outstanding.
 */
def sendStep(j: Job): List[JobStep] =
  if j.phase.in(JobPhase.idle, JobPhase.sent) then enter(Job(JobPhase.sent)) else disabled

/** The queue may deliver a message twice, and the second delivery finds the job started. */
def startStep(j: Job): List[JobStep] = j.phase match
  case JobPhase.sent                    => enter(Job(JobPhase.started))
  case JobPhase.started                 => stay(j)
  case JobPhase.idle | JobPhase.settled => disabled

def settleStep(j: Job): List[JobStep] =
  if j.phase == JobPhase.started then enter(Job(JobPhase.settled)) else disabled

val job = machine[Job, JobOutcome, Nothing] {
  starts(Job(JobPhase.idle))
  ends(j => j.phase == JobPhase.settled)
  steps(send ~> sendStep, start ~> startStep, settle ~> settleStep)
}

// ### The job over the opaque queue

final case class OverQueue(job: Job, queue: QueueView)

val jobOverQueue: Composition[OverQueue] =
  compose[OverQueue](_.job -> job, _.queue -> dispatchQueue)
    .sync(_.job -> send, _.queue -> queue.enqueue)
    .sync(_.job -> start, _.queue -> queue.deliver)
    .sync(_.job -> settle, _.queue -> queue.acknowledge)
    .ends(s => s.job.phase == JobPhase.settled)

val queueSettles =
  jobOverQueue.property("settles").whenAction(jobOverQueue.synced(_.job -> settle)) holds
    (_.state.job.phase == JobPhase.settled)

/** Settling is the acknowledgment, so a settled job leaves no message outstanding. */
val queueSettledLeavesNothing = jobOverQueue
  .property("settledLeavesNothing")
  .never(after =>
    after.state.job.phase == JobPhase.settled && after.state.queue.outstanding != Outstanding.empty
  )

val duplicateDelivery = jobOverQueue.scenario
  .actions(
    jobOverQueue.synced(_.job -> send),
    jobOverQueue.synced(_.job -> start),
    jobOverQueue.synced(_.job -> start),
    jobOverQueue.synced(_.job -> settle)
  )

val queueAny = jobOverQueue.scenario("any").free

// ### The job over a detailed provider, which replaces the opaque queue

final case class OverMatching(job: Job, queue: QueueDetail)

val jobOverMatching: Composition[OverMatching] =
  compose[OverMatching](_.job -> job, _.queue -> matchingQueue)
    .sync(_.job -> send, _.queue -> queue.enqueue)
    .sync(_.job -> start, _.queue -> queue.deliver)
    .sync(_.job -> settle, _.queue -> queue.acknowledge)
    .replaces(_.queue, dispatchQueue)
    .ends(s => s.job.phase == JobPhase.settled)

/** The negative control: the replacement is what must fail. */
val jobOverForgetful: Composition[OverMatching] =
  jobOverMatching.withMember(_.queue -> forgetfulQueue)

def overMatchingQueries(c: Composition[OverMatching]): Vector[Query] =
  val settles = c.property.whenAction(c.synced(_.job -> settle)) holds
    (_.state.job.phase == JobPhase.settled)
  val settledLeavesNothing = c.property
    .never(after =>
      after.state.job.phase == JobPhase.settled && after.state.queue.custody != Custody.nowhere
    )
  // History still holds its dispatch task after the crash, so it invokes AddActivityTask again.
  val crashAfterInvocation = c.scenario
    .actions(
      c.synced(_.job -> send),
      c.own(_.queue, queue.addActivityTask),
      c.own(_.queue, faults.crash),
      c.own(_.queue, queue.addActivityTask),
      c.own(_.queue, queue.persistTask),
      c.synced(_.job -> start),
      c.synced(_.job -> settle)
    )
  val any = c.scenario.free
  Vector(
    query(s"${c.name}.crashAfterInvocation") find settles in
      crashAfterInvocation limits seven total 840,
    query(s"${c.name}.any.settledLeavesNothing") verify settledLeavesNothing in
      any limits twelve total 11520
  )

val queueQueries: Vector[Query] = Vector(
  query("jobOverQueue.duplicateDelivery") find queueSettles in
    duplicateDelivery limits seven total 64,
  query verify queueSettledLeavesNothing in queueAny limits seven total 336
)

val matchingQueries: Vector[Query] = overMatchingQueries(jobOverMatching)
val forgetfulQueries: Vector[Query] = overMatchingQueries(jobOverForgetful)
