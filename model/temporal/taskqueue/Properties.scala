package temporal
package taskqueue

import umpire.*

// ### What a provider promises

/** The laws every provider of the detailed queue is held to, declared once by `queueLaws`. */
final case class QueueLaws(delivers: Property[QueueDetail], committedStays: Property[QueueDetail])

/**
 * The laws of the provider `m`: a delivery hands the message out, and a message a custodian holds
 * stays held until it is acknowledged. Each takes its name explicitly, so every provider's instance
 * keeps the name its checks read.
 */
def queueLaws(m: Machine[QueueDetail, QueueOutcome, QueueFact]): QueueLaws = QueueLaws(
  m.property("delivers") when deliver holds (after => after.records(QueueFact.delivered)),
  m.property("committedStays")
    .stays(_.custody != Custody.nowhere)
    .unless(_.records(QueueFact.acknowledged))
)

/**
 * Storage loss drops a committed message, and the queue records that it did. Only the lossy provider
 * binds the loss, so this is its own Property, not a law of every provider.
 */
val storageLossDrops =
  lossyMatchingQueue.property when storageLoss holds { after =>
    after.state.custody == Custody.nowhere && after.records(QueueFact.storageLost)
  }
