package samples

import framework.dsl.*
import framework.dsl.Finite.given

// The folklore reliable broadcast: in the presence of crashes, all correct processes should see the
// same messages. The origin sends a message to every process, and may crash part way through. So
// that a crash cannot split the processes, a process that receives a message first forwards it to
// every other process, and only then delivers it.
object ReliableBroadcast:
  enum Proc:
    case p1, p2

  // every process, in the order the origin sends to them
  val procs: List[Proc] = Proc.values.toList

  enum Payload:
    case hello, goodbye

  // a message in flight: whom it is for and what it says
  final case class Msg(receiver: Proc, payload: Payload)

  // how far a process got with what it received: forwarding it, forwarded, or delivered
  enum Phase:
    case relaying, relayed, delivered

  // what a process received and how far it got with it
  final case class Received(payload: Payload, phase: Phase)

  // the origin of the broadcast: before it, sending to `next`, done, or crashed part way through
  enum Origin:
    case idle
    case sending(payload: Payload, next: Proc)
    case done(payload: Payload)
    case crashed(payload: Payload)

  // the origin, what each process received, and the messages in flight; a process absent from
  // `received` is idle
  final case class Broadcast(origin: Origin, received: Map[Proc, Received], network: Inbox[Msg])

  // reliable, so that a message sent is a message received; unordered, as the network is
  object wire:
    val network: Channel[Msg] =
      channel[Msg](capacity = 4, order = Order.unordered, loss = Loss.reliable)

  // every process but `p`
  def others(p: Proc): List[Proc] = procs.filter(_ != p)

  // the process the origin sends to after `p`, if any
  def following(p: Proc): Option[Proc] = procs.lift(p.ordinal + 1)

  // whether `network` has room for `count` more messages
  def hasRoom(network: Inbox[Msg], count: Int): Boolean =
    network.deliveries.size + count <= network.channel.capacity

  // the payload the origin broadcast, if it began
  def broadcast(state: Broadcast): Option[Payload] = state.origin match
    case Origin.idle                => None
    case Origin.sending(payload, _) => Some(payload)
    case Origin.done(payload)       => Some(payload)
    case Origin.crashed(payload)    => Some(payload)

  def sending(origin: Origin): Boolean = origin match
    case Origin.sending(_, _)                             => true
    case Origin.idle | Origin.done(_) | Origin.crashed(_) => false

  // how far `p` got, if it received anything
  def phaseOf(received: Map[Proc, Received], p: Proc): Option[Phase] =
    received.get(p).map(_.phase)

  // `p` forwarded what it received to every other process
  def forwarded(received: Map[Proc, Received], p: Proc): Boolean =
    phaseOf(received, p).exists(_ != Phase.relaying)

  // what `p` delivered, if it delivered
  def deliveredBy(received: Map[Proc, Received], p: Proc): Option[Payload] =
    received.get(p).filter(_.phase == Phase.delivered).map(_.payload)

  // the values the processes delivered, None for a process that delivered nothing
  def deliveredValues(state: Broadcast): Set[Option[Payload]] =
    procs.map(deliveredBy(state.received, _)).toSet

  // no process delivered a value or every process delivered the same one
  def agreed(state: Broadcast): Boolean =
    deliveredValues(state).size == 1

  def someoneDelivered(state: Broadcast): Boolean =
    deliveredValues(state).exists(_ != None)

  def everyoneDelivered(state: Broadcast): Boolean =
    agreed(state) && someoneDelivered(state)

  // I only deliver messages that have been sent
  def onlyBroadcast(state: Broadcast): Boolean =
    deliveredValues(state).forall(v => v == None || v == broadcast(state))

  enum Outcome:
    case accepted

  given Ok[Outcome] = Ok(Outcome.accepted)

  object sender extends Actor:
    val broadcast = action.input[Payload](name = "payload")
    val send = action

  object fault extends Actor:
    val crash = action.crash

  object process extends Internal:
    val relay = step.input[Proc](name = "process")
    val deliver = step.input[Proc](name = "process")

  object Model extends Machine[Outcome, Nothing], FailureModel:
    // the origin, each process and the messages in flight: nothing broadcast, nothing sent
    var origin = Origin.idle
    var received = Map.empty[Proc, Received]
    var network = wire.network.empty

    inbox(network, wire.network)
    durability(
      // a crash part way through stops the sending, whomever it was to reach next
      inMemory(origin)(
        (for payload <- Payload.values.toList; next <- procs
        yield Origin.sending(payload, next) -> Origin.crashed(payload))*
      ),
      durable(received),
      durable(network)
    )

    // the broadcast as one record, which the predicates above read
    def current: Broadcast =
      Broadcast(origin, received, network)

    // a run stops once nothing is in flight and no process is part way through
    override def end =
      !sending(origin) && network.isEmpty && received.values.forall(_.phase == Phase.delivered)

    object transitions extends Transitions:
      on(sender.broadcast): payload =>
        if origin == Origin.idle then effects.begin(payload)
      on(sender.send):
        if sending(origin) && !network.isFull then effects.sendNext()
      on(fault.crash):
        if sending(origin) then effects.crash()
      on(monitors.network.deliver): msg =>
        effects.receive(msg)
      on(process.relay): p =>
        if phaseOf(received, p) == Some(Phase.relaying) && hasRoom(network, others(p).size) then
          effects.relay(p)
      on(process.deliver): p =>
        if phaseOf(received, p) == Some(Phase.relayed) then effects.deliver(p)

    object effects:
      def begin(payload: Payload) =
        origin = Origin.sending(payload, procs.head)
      def sendNext() =
        origin match
          case Origin.sending(payload, next) =>
            origin = following(next).fold(Origin.done(payload))(Origin.sending(payload, _))
            network = network.send(Msg(next, payload))
          case Origin.idle | Origin.done(_) | Origin.crashed(_) => ()
      def crash() =
        origin match
          case Origin.sending(payload, _)                       => origin = Origin.crashed(payload)
          case Origin.idle | Origin.done(_) | Origin.crashed(_) => ()
      // On receiving m: forward m to all, then deliver m; a process already busy ignores it.
      def receive(msg: Msg) =
        if !received.contains(msg.receiver) then
          received = received.updated(msg.receiver, Received(msg.payload, Phase.relaying))
      def relay(p: Proc) =
        val payload = received(p).payload
        network = others(p).foldLeft(network)((sent, q) => sent.send(Msg(q, payload)))
        advance(p, Phase.relayed)
      def deliver(p: Proc) =
        advance(p, Phase.delivered)

      private def advance(p: Proc, phase: Phase) =
        received = received.updated(p, received(p).copy(phase = phase))

    object invariants:
      val safety =
        check(onlyBroadcast(current))

    object postconditions:
      // If I deliver a message, I have already forwarded it to all.
      val reliability =
        check(procs.forall(p => deliveredBy(received, p) == None || forwarded(before.received, p)))

    object monitors:
      val network =
        wire.network
      // where a run stops, the processes delivered the same messages
      val agreement =
        monitor(false)(_ => !agreed(current))(identity).readAtEnds

    object progress:
      // once one process delivers, every process delivers the same, whatever else happens
      // meanwhile: the origin's sends or crash, and the other process's forward and delivery
      val everyoneDelivers =
        whenever(someoneDelivered(current)):
          eventually(everyoneDelivered(current))

    object queries:
      // a find target: a run need not crash, so it is no promise and not checked by default
      val survivesCrash =
        after(process.deliver(Proc.p2)):
          check(everyoneDelivered(current))
          check(origin == Origin.crashed(Payload.hello))
      val deliveredDespiteCrash =
        find(survivesCrash)
