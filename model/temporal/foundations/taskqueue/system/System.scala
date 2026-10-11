// The task queue's System: its provider, the route one message takes through history and matching
// (fn-126 decision 16). The level's own file holds TaskQueueSystem, the detailed provider whose
// refinement says what the opaque contract (product/Product.scala) reads of it, and the providers
// derived from it, LossyMatchingQueue, ForgetfulQueue and VolatileQueue.
package temporal
package foundations.taskqueue
package system

import framework.*
import framework.Finite.given
import Bounds.{seven, twelve}
import product.{TaskQueueProduct, TaskQueueProductUnderStorageLoss}
import QueueOutcome.given

val _ = () => summon[CanEqual[Custody, Custody]]

// ### The detailed provider, and the providers derived from it. The violating providers each differ
// from the detailed provider in what one ordinary crash does, and neither assumes storage loss. They
// are negative controls a feature may compose its own design with: the replacement is what must fail.
// format: off
// The detailed provider, under ordinary crashes and lost answers to the poller.
object TaskQueueSystem
    extends Machine[QueueDetail, QueueOutcome, QueueFact],
      Phased[QueueDetail, Custody](_.custody),
      FailureModel, Queueing:
  val init = states.idleQueue
  override def end(d: State) = states.queueEnds(d)
  object states:
    val idleQueue =
      QueueDetail(custody = Custody.nowhere, polled = false, delivered = Delivered.never)
    def queueEnds(d: State) = d.custody == Custody.nowhere // scalafix:ok DisableSyntax.==; typed equality
    def oneMoreDelivery(d: Delivered) = d match
      case Delivered.never                                  => Delivered.once
      case _: Delivered.once.type | _: Delivered.twice.type => Delivered.twice
// format: on
    // Matching holds a task a poll can take: reserved for a waiting poller, or persisted.
    def matchable(c: Custody) = c.in(Custody.reserved, Custody.persisted)
    // format: off
    // A custodian holds the message.
    def held(s: State) = s.custody != Custody.nowhere // scalafix:ok DisableSyntax.!=; typed equality

    // Matching holds the task, no poller holds it, and it was not yet handed out twice.
    def deliverable(s: State) =
      matchable(s.custody) && !s.polled && s.delivered != Delivered.twice // scalafix:ok DisableSyntax.!=; typed equality
    // format: on
    // A poller holds a task matching handed out, whose consumer may now answer.
    def answerable(s: State) =
      s.polled && s.custody != Custody.nowhere && s.delivered != Delivered.never // scalafix:ok DisableSyntax.!=; typed equality
  // format: off
  object refinement extends Refinement(TaskQueueProduct):
    // The interface state a detailed state stands for: a message is outstanding while anyone holds
    // it.
    def toProduct(d: State) =
      if d.custody == Custody.nowhere then QueueView(Outstanding.empty) // scalafix:ok DisableSyntax.==; typed equality
      else
        d.delivered match
          case Delivered.never => QueueView(Outstanding.committed)
          case Delivered.once  => QueueView(Outstanding.deliveredOnce)
          case Delivered.twice => QueueView(Outstanding.deliveredTwice)
  // format: on
    def visible(f: QueueFact) = f match
      case QueueFact.enqueueCommitted | QueueFact.enqueueFailed | QueueFact.delivered |
          QueueFact.acknowledged | QueueFact.storageLost =>
        true
      case QueueFact.addInvoked | QueueFact.taskPersisted | QueueFact.matchReserved |
          QueueFact.crashed | QueueFact.ackLost =>
        false
    // format: off
    def visibleOutcomes(o: QueueOutcome) = o != QueueOutcome.internal // scalafix:ok DisableSyntax.!=; typed equality
    // format: on

  object rules extends Facets(queueing.rules)

  // What a provider promises.
  object properties:
    // The claims of the provider `m`: a delivery hands the message out, and a message a custodian holds
    // stays held until it is acknowledged. Each takes its name explicitly, so every provider's
    // instance keeps the name its checks read.
    def queueClaims(m: Machine[State, QueueOutcome, QueueFact]) = QueueClaims(
      m.property("delivers") when queue.deliver holds (after => after.records(QueueFact.delivered)),
      m.property("committedStays")
        .stays(_.custody != Custody.nowhere) // scalafix:ok DisableSyntax.!=; typed equality
        .unless(_.records(QueueFact.acknowledged))
    )

  // The crash cuts: one crash at each point of the route, and the delivery that must still follow it.
  object queries:
    // One crash after the invocation, after the sync match, after persistence and after a delivery,
    // declared on the provider `m`. After the acknowledgment nothing is left to deliver.
    def providerQueries(m: Machine[State, QueueOutcome, QueueFact]) =
      val claims = properties.queueClaims(m)
      val crashAfterInvocation = m.scenario.actions(
        queue.enqueue,
        queue.addActivityTask,
        fault.crash,
        queue.addActivityTask,
        queue.persistTask,
        queue.deliver
      )
      val crashAfterSyncMatch = m.scenario.actions(
        queue.enqueue,
        queue.addActivityTask,
        queue.syncMatch,
        fault.crash,
        queue.addActivityTask,
        queue.syncMatch,
        queue.deliver
      )
      val crashAfterPersistence = m.scenario.actions(
        queue.enqueue,
        queue.addActivityTask,
        queue.persistTask,
        fault.crash,
        queue.deliver
      )
      val crashAfterDelivery = m.scenario.actions(
        queue.enqueue,
        queue.addActivityTask,
        queue.persistTask,
        queue.deliver,
        fault.crash,
        queue.deliver
      )
      val crashAfterAcknowledgment = m.scenario.actions(
        queue.enqueue,
        queue.addActivityTask,
        queue.persistTask,
        queue.deliver,
        queue.acknowledge,
        fault.crash
      )
      val any = m.scenario.free
      Vector(
        query(s"${m.name}.crashAfterInvocation") find claims.delivers in
          crashAfterInvocation limits seven,
        query(s"${m.name}.crashAfterSyncMatch") find claims.delivers in
          crashAfterSyncMatch limits seven,
        query(s"${m.name}.crashAfterPersistence") find claims.delivers in
          crashAfterPersistence limits seven,
        query(s"${m.name}.crashAfterDelivery") find claims.delivers in
          crashAfterDelivery limits seven,
        query(s"${m.name}.crashAfterAcknowledgment") verify claims.committedStays in
          crashAfterAcknowledgment limits seven,
        query verify claims.committedStays in any limits twelve
      )

    val matchingQueueQueries = providerQueries(TaskQueueSystem)
// format: off
// The detailed provider with the storage-loss fault, which only its assumption allows. It refines
// the interface that allows the loss.
object LossyMatchingQueue
    extends Derived(
      TaskQueueSystem
        .extend(on(fault.storageLoss) {
          where(TaskQueueSystem.states.held) ~> TaskQueueSystem.queueing.effects.storageLossDetail
        })
        .refining(TaskQueueProductUnderStorageLoss)(TaskQueueSystem.refinement.toProduct)
        .assuming(storageLossAssumed)
    ),
      FailureModel:
  object properties:
    // Storage loss drops a committed message, and the queue records that it did. Only the lossy
    // provider binds the loss, so this is its own Property, not a claim of every provider.
    val storageLossDrops =
      property when fault.storageLoss holds { after =>
        after.state.custody == Custody.nowhere && after.records(QueueFact.storageLost) // scalafix:ok DisableSyntax.==; typed equality
      }
// format: on
  // The crash cuts of every provider, and the storage loss.
  object queries:
    val persistedThenLost = scenario.actions(
      queue.enqueue,
      queue.addActivityTask,
      queue.persistTask,
      fault.storageLoss
    )
    val lossyMatchingQueueQueries =
      TaskQueueSystem.queries.providerQueries(LossyMatchingQueue)
    val storageLossQuery =
      query("lossyMatchingQueue.storageLoss") find properties.storageLossDrops in
        persistedThenLost limits seven

// A crash that loses the message history dropped on invoking AddActivityTask.
object ForgetfulQueue
    extends Derived(TaskQueueSystem.rebind(fault.crash ~> TaskQueueSystem.queueing.effects.forgetfulCrash)),
      NegativeControl:

  object queries:
    val forgetfulQueueQueries =
      TaskQueueSystem.queries.providerQueries(ForgetfulQueue)

// A crash that wipes the tasks matching persisted.
object VolatileQueue
    extends Derived(TaskQueueSystem.rebind(fault.crash ~> TaskQueueSystem.queueing.effects.volatileCrash)),
      NegativeControl:

  object queries:
    val volatileQueueQueries = TaskQueueSystem.queries.providerQueries(VolatileQueue)
