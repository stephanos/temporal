// The task queue: the durable queue between history's dispatch and a worker's poll, as a reusable
// entity of its own, beside actors/worker. Its opaque contract is what a feature may rely on; the
// matching provider is the route one message takes through history and matching, which refines it;
// and the forgetful and volatile providers are deliberately violating controls. Grounded in
// chasm/lib/activity/tasks.go (the dispatch task) and service/matching (AddActivityTask, sync match,
// the persisted task queue and its completion).
//
// Update this Model independently of the implementation. When conformance fails, ask a human
// rather than fitting the Model to the code.
//
// A feature composes a provider with its own machine and synchronizes its actions with enqueue,
// deliver and acknowledge; the standalone activity's history record does. The queue names nothing
// of a feature.
//
// The abstraction is bounded: one message at a time, delivered at most twice before its
// acknowledgment. It is not a general multi-message queue, and a check of it says nothing of two
// messages in flight at once.
//
// The queue has two levels, each in a folder of its own (fn-126 decision 16): its contract is its
// Product and its provider its Protocol. This file holds the types and the signature; each level
// folder holds its level file, its facet and one file per derived provider, grouped by name prefix
// (fn-160 R7; model/README.md, "The layers").
//
// A machine object reads its header (entity, init, end), then its sections in order: states,
// refinement, effects, monitors, rules, properties and queries.
package temporal
package foundations.taskqueue

import framework.dsl.*

// ### Types

// The one message the queue may hold, by how far its delivery got.
enum Outstanding:
  case empty, committed, deliveredOnce, deliveredTwice

final case class QueueView(outstanding: Outstanding)

// `internal` answers every step of a provider that its interface does not show.
enum QueueOutcome:
  case committed, failed, delivered, acknowledged, lost, internal

object QueueOutcome:
  // A provider's step behind the interface is the one that answers `internal`. Kept in the
  // companion, where only a step known to answer a QueueOutcome finds it, so a `choose` of a feature's
  // own steps still finds its own `Ok` alone.
  given Ok[QueueOutcome] = Ok(QueueOutcome.internal)

// The interface's events, and what a provider records of the steps behind them.
enum QueueFact:
  case enqueueCommitted, enqueueFailed, delivered, acknowledged, storageLost
  case addInvoked, taskPersisted, matchReserved, crashed, ackLost

// Who holds the message. `history` is the dispatch task alone; `invoked` an AddActivityTask in
// flight and `reserved` a sync match, both only in memory; `persisted` a task in matching's durable
// queue, which is what lets history's own task finish.
enum Custody:
  case nowhere, history, invoked, reserved, persisted

enum Delivered:
  case never, once, twice

// `polled` is a poller holding the task while the consumer decides.
final case class QueueDetail(custody: Custody, polled: Boolean, delivered: Delivered)

// The claims every provider of the detailed queue is held to, declared once by `queueClaims`.
final case class QueueClaims(delivers: Property[QueueDetail], committedStays: Property[QueueDetail])

// The task queue's facet vocabulary (fn-161), shared by its levels: each level is one facet.
enum FacetName:
  case queueing

// The design family its derived providers share (fn-160 R7): QueueForgetful, QueueVolatile, ...
enum FamilyName:
  case queue

// ### Signature

// Named by the task queue's name, as the worker that polls it is. Messages and their deliveries are
// the queue's state, not entities of their own.
val taskQueueEntity = Entity("taskQueue", key = "taskQueue")

val enqueueCommits = choice
val enqueueFails = choice

// The queue's own steps.
//
// The interface: what a feature may rely on of the durable queue. An enqueue commits or fails; a
// committed message is delivered up to twice before its acknowledgment; and no committed message
// is lost. A crash shows at this interface only as that second delivery.
//
// The detailed queue: history's dispatch task and matching's custody. The route of one message:
// history durably schedules the dispatch task; its Execute invokes AddActivityTask; matching either
// persists the task or reserves it for a waiting poller; a poll hands it out; and once the consumer
// has answered, matching completes it. Each step is its own transition, so a crash can fall between
// any two.
object queue:
  val enqueue = internal on taskQueueEntity
  val deliver = internal on taskQueueEntity
  val acknowledge = internal on taskQueueEntity

  val addActivityTask = internal on taskQueueEntity
  val persistTask = internal on taskQueueEntity
  val syncMatch = internal on taskQueueEntity

// The faults the providers suffer. Like the worker's stop and resume, they name no entity.
object fault extends Actor:
  // Committed storage may be lost. It is a fault of its own, apart from a crash, and only a machine
  // that assumes it has the step.
  val storageLoss = action(this)

  val crash = action(this)
  val ackLoss = action(this)

// Committed storage may be lost. No machine makes the assumption of its own: a derivation that binds
// the storage-loss step adds it.
val storageLossAssumed = assume("storageLoss")
