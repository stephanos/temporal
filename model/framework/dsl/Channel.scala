package framework.dsl

import scala.annotation.publicInBinary

// The order a channel delivers the messages it holds in.
enum Order derives CanEqual:
  // The order they were sent in.
  case fifo

  // Any held message may be delivered next.
  case unordered

// Whether a channel may lose a message it holds. A lossy channel's loss is a step the machine
// binds, as every fault is an action.
enum Loss derives CanEqual:
  case reliable, lossy

// One message a channel holds, and how many more times than once it has been delivered.
final case class Delivery[M](message: M, redeliveries: Int)

// A bounded channel: the messages one machine's state holds between their send and their delivery,
// declared with its capacity, its order, whether it loses messages, and how many more times than
// once it may deliver one whose acknowledgment was lost.
//
// A machine holds it in a state field of type `Inbox[M]`, sends with `Inbox.send`, and binds
// `deliver` to what receiving a message does and, for a lossy channel, `lose` to what losing one
// does. When a message can be delivered or lost, and the redelivery after a lost acknowledgment,
// are derived from this declaration by the IR interpreter (model/framework/SEMANTICS.md, Channels).
final class Channel[M] private[framework] (
    val name: String,
    val capacity: Int,
    val order: Order,
    val loss: Loss,
    val duplicates: Int
)(using val messages: Finite[M]):
  private[framework] val comparisonIdentity: Long = Channel.nextIdentity.getAndIncrement()
  private given Data.Structural[M] = messages.structural
  private var source = Option.empty[String]
  private[framework] var schemaKey = Option.empty[String]
  private[framework] var spelling = Option.empty[String]
  private[framework] def sourceIdentity: Option[String] = source

  @publicInBinary private[framework] def sourceAt(id: String): Channel[M] =
    if source.isEmpty then declaredAt(id): Unit
    this

  def toIR: io.temporal.server.api.umpire.v1.Channel = Signature.channel(this)

  @publicInBinary private[framework] def retainName(id: String, name: String): Channel[M] =
    if source.contains(id) then
      require(spelling.forall(_ == name), s"channel $id has conflicting declaration names")
      spelling = Some(name)
      val declared = if this.name.nonEmpty then this.name else name
      deliver.decl.channelName = Some(declared)
      lose.decl.channelName = Some(declared)
    this

  @publicInBinary private[framework] def retainInputSchema(
      key: String,
      root: Seq[String],
      types: Seq[String],
      inboxes: Seq[String]
  ): Channel[M] =
    require(
      schemaKey.forall(_ == key),
      s"channel ${source.getOrElse(name)} has conflicting schema identity"
    )
    try messages.retainSchema(key, root, types, inboxes): Unit
    catch
      case error: IllegalArgumentException =>
        throw IllegalArgumentException(
          s"channel ${source.getOrElse(name)} message: ${error.getMessage}",
          error
        )
    schemaKey = Some(key)
    deliver.retainInputSchema(0, key, root, types, inboxes): Unit
    lose.retainInputSchema(0, key, root, types, inboxes): Unit
    this
  @publicInBinary private[framework] def declaredAt(id: String): Channel[M] =
    require(
      source.forall(_.equals(id)),
      s"channel $id conflicts with source identity ${source.getOrElse("")}"
    )
    source = Some(id)
    deliver.declaredAt(s"$id.deliver"): Unit
    lose.declaredAt(s"$id.lose"): Unit
    deliver.decl.declaredByChannel(id, "deliver")
    lose.decl.declaredByChannel(id, "lose")
    this
  // Every entry a channel can hold, messages in catalog order and each one's redeliveries within.
  private[framework] lazy val entries: Vector[Delivery[M]] =
    for m <- messages.values.toVector; r <- (0 to duplicates).toVector yield Delivery(m, r)

  // Everything the channel can hold, for the state field that holds it: no message, then every
  // sequence of one entry, of two, up to the capacity, the last entry varying fastest. Unordered
  // contents are the sequences whose entries are in structural order, one per multiset.
  lazy val contents: Finite[Inbox[M]] =
    val ordered =
      if order == Order.fifo then entries else entries.sortWith(compareDelivery(_, _) < 0)
    val sequences = (0 to capacity).toList.flatMap { n =>
      (1 to n).foldLeft(List(List.empty[Delivery[M]])) { (prefixes, _) =>
        for prefix <- prefixes; e <- ordered.toList yield prefix :+ e
      }
    }
    val held = if order == Order.fifo then sequences
    else sequences.filter(s => s.zip(s.drop(1)).forall((a, b) => compareDelivery(a, b) <= 0))
    Finite.of(held.map(Inbox(this, _))*)

  private[framework] def compareDelivery(left: Delivery[M], right: Delivery[M]): Int =
    val message = messages.compare(left.message, right.message)
    if message != 0 then message else Integer.compare(left.redeliveries, right.redeliveries)

  // The channel holding nothing.
  val empty: Inbox[M] = Inbox(this, Nil)

  private def scalarAction(decl: ActionDecl): ScalarAction[M] =
    val shape = new ActionInputs.Scalar[M](messages, None)
    decl.nativeInputs = Some(shape)
    new ScalarAction(decl, shape)

  // The delivery of one held message, the action's one input.
  val deliver: ScalarAction[M] = scalarAction(
    ActionDecl(
      s"${name}Delivery",
      Actor.system,
      inputs = List("message"),
      domains = List(messages),
      internal = true,
      delivers = name
    )
  )

  // The loss of one held message, for a lossy channel.
  val lose: ScalarAction[M] = scalarAction(
    ActionDecl(
      s"${name}Loss",
      Actor.system,
      inputs = List("message"),
      domains = List(messages),
      internal = true,
      loses = name
    )
  )

  override def toString: String = s"channel $name"

object Channel:
  private val nextIdentity = new java.util.concurrent.atomic.AtomicLong()

// Declares a bounded channel of messages of `M`, named after the `val` that declares it: `val wire
// = channel[Note](capacity = 2, ...)`. The name is read by the lifter, so the channel's own `name`
// is empty here.
def channel[M](capacity: Int, order: Order, loss: Loss, duplicates: Int = 0)(using
    Finite[M]
): Channel[M] =
  require(
    capacity >= 1,
    s"a channel holds at most $capacity messages; a channel holds at least one"
  )
  require(
    duplicates >= 0,
    s"a channel delivers a message $duplicates more times; it cannot deliver fewer than once"
  )
  Channel("", capacity, order, loss, duplicates)

// What a channel holds: its deliveries in send order, or in structural order for an unordered channel,
// so that sending the same messages in another order reaches the same state.
final case class Inbox[M] private[framework] (channel: Channel[M], deliveries: List[Delivery[M]]):
  // The contents with `m` added: at the end, or at its structural position for an unordered channel.
  // Sending to a full channel leaves the channel's contents, so the step lands outside the domain.
  def send(m: M): Inbox[M] =
    val d = Delivery(m, 0)
    if channel.order == Order.fifo then copy(deliveries = deliveries :+ d)
    else
      val (before, after) = deliveries.span(e => channel.compareDelivery(e, d) <= 0)
      copy(deliveries = before ++ (d :: after))

  def isEmpty: Boolean = deliveries.isEmpty

  // Whether it holds as many messages as the channel's capacity.
  def isFull: Boolean = deliveries.size >= channel.capacity

object Inbox:
  private[framework] def structural[M](message: Data.Structural[M]): Data.Structural[Inbox[M]] =
    new Data.Structural[Inbox[M]]:
      def compare(left: Inbox[M], right: Inbox[M]): Int =
        val owner =
          java.lang.Long.compare(left.channel.comparisonIdentity, right.channel.comparisonIdentity)
        if owner != 0 then owner
        else
          val a = left.deliveries.iterator
          val b = right.deliveries.iterator
          var result = 0
          while result == 0 && a.hasNext && b.hasNext do
            val x = a.next()
            val y = b.next()
            result = message.compare(x.message, y.message)
            if result == 0 then result = Integer.compare(x.redeliveries, y.redeliveries)
          if result != 0 then result else java.lang.Boolean.compare(a.hasNext, b.hasNext)
