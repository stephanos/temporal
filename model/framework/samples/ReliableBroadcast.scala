package samples

import framework.dsl.*
import framework.dsl.Finite.given

// Modelled after https://github.com/quint-co/quint/blob/main/examples/classic/distributed/ReliableBroadcast/reliablebc.qnt
//
// The folklore reliable broadcast: in the presence of crashes, all correct processes should see the
// same messages. The origin sends a message to every process, and may crash part way through. So
// that a crash cannot split the processes, a process that receives a message first forwards it to
// every other process, and only then delivers it.
object ReliableBroadcast:
  enum Proc:
    case p1, p2

  enum Payload:
    case hello, goodbye

  // a message in flight: whom it is for and what it says
  final case class Msg(receiver: Proc, payload: Payload)

  // a process: waiting, forwarding what it received, forwarded, or delivered
  enum Local:
    case idle
    case relaying(payload: Payload)
    case relayed(payload: Payload)
    case delivered(payload: Payload)

  // the origin of the broadcast: before it, sending to `next`, done, or crashed part way through
  enum Origin:
    case idle
    case sending(payload: Payload, next: Proc)
    case done(payload: Payload)
    case crashed(payload: Payload)

  final case class Broadcast(origin: Origin, p1: Local, p2: Local, network: Inbox[Msg])

  // reliable, so that a message sent is a message received; unordered, as the network is
  object wire:
    val network: Channel[Msg] =
      channel[Msg](capacity = 4, order = Order.unordered, loss = Loss.reliable)

  def other(p: Proc): Proc = p match
    case Proc.p1 => Proc.p2
    case Proc.p2 => Proc.p1

  def localOf(state: Broadcast, p: Proc): Local = p match
    case Proc.p1 => state.p1
    case Proc.p2 => state.p2

  def withLocal(state: Broadcast, p: Proc, local: Local): Broadcast = p match
    case Proc.p1 => state.copy(p1 = local)
    case Proc.p2 => state.copy(p2 = local)

  // the payload the origin broadcast, if it began
  def broadcast(state: Broadcast): Option[Payload] = state.origin match
    case Origin.idle                => None
    case Origin.sending(payload, _) => Some(payload)
    case Origin.done(payload)       => Some(payload)
    case Origin.crashed(payload)    => Some(payload)

  def deliveredValue(local: Local): Option[Payload] = local match
    case Local.idle               => None
    case Local.relaying(_)        => None
    case Local.relayed(_)         => None
    case Local.delivered(payload) => Some(payload)

  def relaying(local: Local): Boolean = local match
    case Local.idle         => false
    case Local.relaying(_)  => true
    case Local.relayed(_)   => false
    case Local.delivered(_) => false

  def relayed(local: Local): Boolean = local match
    case Local.idle         => false
    case Local.relaying(_)  => false
    case Local.relayed(_)   => true
    case Local.delivered(_) => false

  def settled(local: Local): Boolean = local == Local.idle || deliveredValue(local) != None

  def originSending(state: Broadcast): Boolean = state.origin match
    case Origin.idle          => false
    case Origin.sending(_, _) => true
    case Origin.done(_)       => false
    case Origin.crashed(_)    => false

  // no process delivered a value or every process delivered the same one
  def agreed(state: Broadcast): Boolean =
    deliveredValue(state.p1) == deliveredValue(state.p2)

  def someoneDelivered(state: Broadcast): Boolean =
    deliveredValue(state.p1) != None || deliveredValue(state.p2) != None

  def everyoneDelivered(state: Broadcast): Boolean =
    deliveredValue(state.p1) != None && agreed(state)

  // I only deliver messages that have been sent
  def onlyBroadcast(state: Broadcast): Boolean =
    (deliveredValue(state.p1) == None || deliveredValue(state.p1) == broadcast(state)) &&
      (deliveredValue(state.p2) == None || deliveredValue(state.p2) == broadcast(state))

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
    var p1 = Local.idle
    var p2 = Local.idle
    var network = wire.network.empty

    inbox(network, wire.network)
    durability(
      inMemory(origin)(
        Origin.sending(Payload.hello, Proc.p1) -> Origin.crashed(Payload.hello),
        Origin.sending(Payload.hello, Proc.p2) -> Origin.crashed(Payload.hello),
        Origin.sending(Payload.goodbye, Proc.p1) -> Origin.crashed(Payload.goodbye),
        Origin.sending(Payload.goodbye, Proc.p2) -> Origin.crashed(Payload.goodbye)
      ),
      durable(p1),
      durable(p2),
      durable(network)
    )

    // the broadcast as one record, which the predicates above read
    def current: Broadcast = Broadcast(origin, p1, p2, network)

    // a run stops once nothing is in flight and no process is part way through
    override def end = !originSending(current) && network.isEmpty && settled(p1) && settled(p2)

    val anyRun = starts()

    object transitions extends Transitions:
      on(sender.broadcast): payload =>
        if origin == Origin.idle then effects.begin(payload)
      on(sender.send):
        if originSending(current) && !network.isFull then effects.sendNext()
      on(fault.crash):
        if originSending(current) then effects.crash()
      on(monitors.network.deliver): msg =>
        effects.receive(msg)
      on(process.relay(Proc.p1)):
        if relaying(p1) && !network.isFull then effects.relayP1()
      on(process.relay(Proc.p2)):
        if relaying(p2) && !network.isFull then effects.relayP2()
      on(process.deliver(Proc.p1)):
        if relayed(p1) then effects.deliverP1()
      on(process.deliver(Proc.p2)):
        if relayed(p2) then effects.deliverP2()

    object effects:
      def begin(payload: Payload): Unit = origin = Origin.sending(payload, Proc.p1)
      def sendNext(): Unit = origin match
        case Origin.sending(payload, next) =>
          origin =
            if next == Proc.p1 then Origin.sending(payload, Proc.p2) else Origin.done(payload)
          network = network.send(Msg(next, payload))
        case Origin.idle       => ()
        case Origin.done(_)    => ()
        case Origin.crashed(_) => ()
      def crash(): Unit = origin match
        case Origin.sending(payload, _) => origin = Origin.crashed(payload)
        case Origin.idle                => ()
        case Origin.done(_)             => ()
        case Origin.crashed(_)          => ()
      // On receiving m: forward m to all, then deliver m; a process already busy ignores it.
      def receive(msg: Msg): Unit =
        if localOf(current, msg.receiver) == Local.idle then
          setLocal(msg.receiver, Local.relaying(msg.payload))
      // The naive algorithm: deliver m on receiving it, forwarding nothing.
      def receiveNaive(msg: Msg): Unit =
        if localOf(current, msg.receiver) == Local.idle then
          setLocal(msg.receiver, Local.delivered(msg.payload))
      def relayP1(): Unit = relayFrom(Proc.p1)
      def relayP2(): Unit = relayFrom(Proc.p2)
      def deliverP1(): Unit = deliverAt(Proc.p1)
      def deliverP2(): Unit = deliverAt(Proc.p2)

      private def relayFrom(p: Proc): Unit = localOf(current, p) match
        case Local.relaying(payload) =>
          network = network.send(Msg(other(p), payload))
          setLocal(p, Local.relayed(payload))
        case Local.idle         => ()
        case Local.relayed(_)   => ()
        case Local.delivered(_) => ()

      private def deliverAt(p: Proc): Unit = localOf(current, p) match
        case Local.relayed(payload) => setLocal(p, Local.delivered(payload))
        case Local.idle             => ()
        case Local.relaying(_)      => ()
        case Local.delivered(_)     => ()

      private def setLocal(p: Proc, local: Local): Unit = p match
        case Proc.p1 => p1 = local
        case Proc.p2 => p2 = local

    object invariants:
      val safety = check(onlyBroadcast(current))

    object postconditions:
      // If I deliver a message, I have already forwarded it to all.
      val reliability =
        if deliveredValue(p1) != None && deliveredValue(before.p1) == None then
          check(relayed(before.p1))
        if deliveredValue(p2) != None && deliveredValue(before.p2) == None then
          check(relayed(before.p2))

    object monitors:
      val network = wire.network
      // where a run stops, the processes delivered the same messages
      val agreement = monitor(false)(_ => !agreed(current))(identity).readAtEnds

    object progress:
      // once one process delivers, every process delivers the same, whatever else happens
      // meanwhile: the origin's sends or crash, four deliveries and the other process's forward and
      // delivery
      val everyoneDelivers =
        leadsTo(from = someoneDelivered(current), to = everyoneDelivered(current), within = 9)

    object queries:
      // a find target: a run need not crash, so it is no promise and not checked by default
      val survivesCrash = after(process.deliver(Proc.p2)):
        check(everyoneDelivered(current))
        check(origin == Origin.crashed(Payload.hello))
      val safetyHolds = verify(invariants.safety) in anyRun
      val reliabilityHolds = verify(postconditions.reliability) in anyRun
      val deliveredDespiteCrash = find(survivesCrash) in anyRun

  // The naive algorithm, which delivers a message on receiving it and forwards nothing: the negative
  // control of agreement, which the sample tests refute.
  object Naive
      extends Derived(
        Model.rebind(
          on(Model.monitors.network.deliver): msg =>
            Model.effects.receiveNaive(msg)
        )
      ),
        NegativeControl:

    object invariants:
      // The naive algorithm delivers only what was broadcast too: what refutes it is the agreement
      // monitor, which a run that stops with one process delivered and the other not breaks.
      val naiveSafety = check(onlyBroadcast(Model.current))
