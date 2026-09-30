package umpire

/** The order a channel delivers the messages it holds in. */
enum Order:
  /** The order they were sent in. */
  case fifo
  /** Any held message may be delivered next. */
  case unordered

/** Whether a channel may lose a message it holds. A lossy channel's loss is a step the machine
  * binds, as every fault is an action. */
enum Loss:
  case reliable, lossy

/** One message a channel holds, and how many more times than once it has been delivered. */
final case class Delivery[M](message: M, redeliveries: Int)

/** A bounded channel: the messages one machine's state holds between their send and their delivery,
  * declared with its capacity, its order, whether it loses messages, and how many more times than
  * once it may deliver one whose acknowledgment was lost.
  *
  * A machine holds it in a state field of type `Inbox[M]`, sends with `Inbox.send`, and binds
  * `deliver` to what receiving a message does and, for a lossy channel, `lose` to what losing one
  * does. When a message can be delivered or lost, and the redelivery after a lost acknowledgment,
  * are derived from this declaration by the IR interpreter (model/scalav2/SEMANTICS.md, Channels);
  * this framework's table does not derive them and refuses a machine that binds either. */
final class Channel[M] private[umpire] (
    val name: String,
    val capacity: Int,
    val order: Order,
    val loss: Loss,
    val duplicates: Int,
)(using val messages: Finite[M]):
  /** Every entry a channel can hold, messages in catalog order and each one's redeliveries within. */
  private[umpire] lazy val entries: Vector[Delivery[M]] =
    for m <- messages.values.toVector; r <- (0 to duplicates).toVector yield Delivery(m, r)

  /** Everything the channel can hold, for the state field that holds it: no message, then every
    * sequence of one entry, of two, up to the capacity, the last entry varying fastest. Unordered
    * contents are the sequences whose entries are in catalog order, one per multiset. */
  lazy val contents: Finite[Inbox[M]] =
    val index = entries.zipWithIndex.toMap
    val sequences = (0 to capacity).toList.flatMap { n =>
      (1 to n).foldLeft(List(List.empty[Delivery[M]])) { (prefixes, _) =>
        for prefix <- prefixes; e <- entries.toList yield prefix :+ e
      }
    }
    val held = if order == Order.fifo then sequences else sequences.filter(s => s.map(index) == s.map(index).sorted)
    Finite.of(held.map(Inbox(this, _))*)

  /** The channel holding nothing. */
  val empty: Inbox[M] = Inbox(this, Nil)

  /** The delivery of one held message, the action's one input. */
  val deliver: Action[M *: EmptyTuple] = Action(ActionDecl(s"${name}Delivery", Party.system, inputs = List("message"),
    domains = List(messages), internal = true, delivers = name))

  /** The loss of one held message, for a lossy channel. */
  val lose: Action[M *: EmptyTuple] = Action(ActionDecl(s"${name}Loss", Party.system, inputs = List("message"),
    domains = List(messages), internal = true, loses = name))

  override def toString: String = s"channel $name"

/** Declares a bounded channel of messages of `M`. */
def channel[M](name: String, capacity: Int, order: Order, loss: Loss, duplicates: Int = 0)(using
    Finite[M]
): Channel[M] =
  require(capacity >= 1, s"channel $name holds at most $capacity messages; a channel holds at least one")
  require(duplicates >= 0, s"channel $name delivers a message $duplicates more times; it cannot deliver fewer than once")
  Channel(name, capacity, order, loss, duplicates)

/** What a channel holds: its deliveries in send order, or in catalog order for an unordered channel,
  * so that sending the same messages in another order reaches the same state. */
final case class Inbox[M] private[umpire] (channel: Channel[M], deliveries: List[Delivery[M]]) extends Keyed:
  /** The contents with `m` added: at the end, or at its catalog position for an unordered channel.
    * Sending to a full channel leaves the channel's contents, so the step lands outside the domain. */
  def send(m: M): Inbox[M] =
    val d = Delivery(m, 0)
    if channel.order == Order.fifo then copy(deliveries = deliveries :+ d)
    else
      val index = channel.entries.indexOf(d)
      val (before, after) = deliveries.span(e => channel.entries.indexOf(e) <= index)
      copy(deliveries = before ++ (d :: after))

  def isEmpty: Boolean = deliveries.isEmpty

  /** Whether it holds as many messages as the channel's capacity. */
  def isFull: Boolean = deliveries.size >= channel.capacity

  /** Its deliveries, each a message followed by its redeliveries, as a list is spelled. */
  def key: String = deliveries.map(d => s"${Keys.of(d.message)}-${d.redeliveries}").mkString("[", ",", "]")
