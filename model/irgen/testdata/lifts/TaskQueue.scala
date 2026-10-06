// An independent consumer of the task queue in temporal/shared/taskqueue: a job of its own, sent once
// and settled once, composed with the opaque queue, with the matching provider that replaces it, and
// with the forgetful provider as the negative control. It imports nothing of the standalone activity,
// so the queue is lifted and checked here as any other feature would rely on it. The lifter's tests
// compare the IR with expected/taskqueue.json, and tools/umpire/check pins every Query's answer.
package fixture.taskqueue

import temporal.shared.taskqueue.*
import temporal.shared.taskqueue.product.TaskQueueProduct
import temporal.shared.taskqueue.system.{ForgetfulQueue, TaskQueueSystem}
import umpire.*

// ### The job

enum JobPhase derives Finite:
  case idle, sent, started, settled

// The job's state, named apart from the machine object `Job`.
final case class JobState(phase: JobPhase) derives Finite

enum JobOutcome derives Finite:
  case accepted

given Ok[JobOutcome] = Ok(JobOutcome.accepted)

type JobStep = Step[JobState, JobOutcome, Nothing]

val send = internal
val start = internal
val settle = internal

// A failed enqueue leaves nothing outstanding, so the job may be sent again; the queue disables the
// enqueue of a second message while one is outstanding.
def sendStep(j: JobState): List[JobStep] =
  if j.phase.in(JobPhase.idle, JobPhase.sent) then enter(JobState(JobPhase.sent)) else disabled

// The queue may deliver a message twice, and the second delivery finds the job started.
def startStep(j: JobState): List[JobStep] = j.phase match
  case JobPhase.sent                    => enter(JobState(JobPhase.started))
  case JobPhase.started                 => stay(j)
  case JobPhase.idle | JobPhase.settled => disabled

def settleStep(j: JobState): List[JobStep] =
  if j.phase == JobPhase.started then enter(JobState(JobPhase.settled)) else disabled

object Job extends Machine[JobState, JobOutcome, Nothing]:
  val init = JobState(JobPhase.idle)
  def end(j: State) = j.phase == JobPhase.settled

  object rules extends Bindings(send ~> sendStep, start ~> startStep, settle ~> settleStep)

// ### The job over the opaque queue

final case class OverQueue(job: JobState, queue: QueueView)

object JobOverQueue extends Composition[OverQueue](_.job -> Job, _.queue -> TaskQueueProduct):
  def end(s: State) = s.job.phase == JobPhase.settled
  object syncs extends Syncs:
    sync(_.job -> send, _.queue -> queue.enqueue)
    sync(_.job -> start, _.queue -> queue.deliver)
    sync(_.job -> settle, _.queue -> queue.acknowledge)

val queueSettles =
  JobOverQueue.property("settles").whenAction(JobOverQueue.synced(_.job -> settle)) holds
    (_.state.job.phase == JobPhase.settled)

// Settling is the acknowledgment, so a settled job leaves no message outstanding.
val queueSettledLeavesNothing = JobOverQueue
  .property("settledLeavesNothing")
  .never(after =>
    after.state.job.phase == JobPhase.settled && after.state.queue.outstanding != Outstanding.empty
  )

val duplicateDelivery = JobOverQueue.scenario
  .actions(
    JobOverQueue.synced(_.job -> send),
    JobOverQueue.synced(_.job -> start),
    JobOverQueue.synced(_.job -> start),
    JobOverQueue.synced(_.job -> settle)
  )

val queueAny = JobOverQueue.scenario("any").free

// ### The job over a detailed provider, which replaces the opaque queue

final case class OverMatching(job: JobState, queue: QueueDetail)

object JobOverMatching
    extends Composition[OverMatching](_.job -> Job, _.queue -> TaskQueueSystem),
      FailureModel:
  def end(s: State) = s.job.phase == JobPhase.settled
  object syncs extends Syncs:
    sync(_.job -> send, _.queue -> queue.enqueue)
    sync(_.job -> start, _.queue -> queue.deliver)
    sync(_.job -> settle, _.queue -> queue.acknowledge)
    replaces(_.queue, TaskQueueProduct)

// The negative control: the replacement is what must fail. No Query asks it; the check of the
// member that stands in for the opaque queue refutes it.
object JobOverForgetful
    extends Composition(JobOverMatching.withMember(_.queue -> ForgetfulQueue)),
      NegativeControl

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
      c.own(_.queue, fault.crash),
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

val matchingQueries: Vector[Query] = overMatchingQueries(JobOverMatching)
val forgetfulQueries: Vector[Query] = overMatchingQueries(JobOverForgetful)
