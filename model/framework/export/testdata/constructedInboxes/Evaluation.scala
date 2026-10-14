package framework.inboxevaluation

import framework.dsl.*

enum Message:
  case ping
object Counts:
  var receivers = 0
  var catalogs = 0
given message: Finite[Message] = new Finite[Message]:
  def values: IndexedSeq[Message] =
    Counts.catalogs += 1
    Vector(Message.ping)
object Receiver:
  Counts.receivers += 1
  val wire = channel[Message](capacity = 1, order = Order.fifo, loss = Loss.reliable)
final case class LazyState(box: Inbox[Message])
given lazyState: Finite[LazyState] =
  given contents: Finite[Inbox[Message]] = Receiver.wire.contents
  new Finite[LazyState]:
    def values = Finite.derived[LazyState].values
final case class EagerState(box: Inbox[Message])
given eagerState: Finite[EagerState] =
  given contents: Finite[Inbox[Message]] = Receiver.wire.contents
  Finite.derived
final case class ThrowingState(box: Inbox[Message])
given throwingState: Finite[ThrowingState] =
  given contents: Finite[Inbox[Message]] = Receiver.wire.contents
  new Finite[ThrowingState]:
    def values = throw new AssertionError(s"state catalog evaluated: $contents")
object Evaluation:
  def main(args: Array[String]): Unit =
    val witness = summon[Finite[LazyState]]
    val structural = witness.schema.get
    assert(Counts.receivers == 0 && Counts.catalogs == 0)
    for _ <- 0 until 2 do
      val result =
        Signature.stateSchema(witness, Some("framework.inboxevaluation.LazyState"), "state lazy")
      assert(
        result.types
          .find(_.name == structural.identity)
          .get
          .getRecord
          .fields
          .head
          .getType
          .getChannel ==
          "framework.inboxevaluation.Receiver.wire"
      )
      assert(Counts.receivers == 1 && Counts.catalogs == 0)
      assert(witness.schema.get == structural)
      assert(witness eq summon[Finite[LazyState]])
    assert(witness.values.size == 2)
    assert(Counts.catalogs == 1)
    val eager = summon[Finite[EagerState]]
    assert(eager.values.size == 2)
    assert(Counts.catalogs == 1)
    Signature.stateSchema(eager, Some("framework.inboxevaluation.EagerState"), "state eager"): Unit
    val throwing = summon[Finite[ThrowingState]]
    Signature.stateSchema(
      throwing,
      Some("framework.inboxevaluation.ThrowingState"),
      "state throwing"
    ): Unit
    assert(Counts.receivers == 1 && Counts.catalogs == 1)
    println("evaluation:constructed=0,schema=0,receiver=1,lazy-values=1,eager-values=1,throwing=1")
