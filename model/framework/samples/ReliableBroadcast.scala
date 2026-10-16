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

  object Broadcast:
    given Finite[Broadcast] =
      given Finite[Inbox[Msg]] = wire.network.contents
      new Finite[Broadcast]:
        def values: IndexedSeq[Broadcast] = Finite.derived[Broadcast].values

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
    val broadcast = action(this).input[Payload]("payload")
    val send = action(this)

  object fault extends Actor:
    val crash = action(this)

  object process:
    val relay = internal.input[Proc]("process")
    val deliver = internal.input[Proc]("process")

  def relayFrom(state: Broadcast, p: Proc): List[Step[Broadcast, Outcome, Nothing]] =
    localOf(state, p) match
      case Local.relaying(payload) =>
        val sent = state.copy(network = state.network.send(Msg(other(p), payload)))
        enter(withLocal(sent, p, Local.relayed(payload)))
      case Local.idle         => stay(state)
      case Local.relayed(_)   => stay(state)
      case Local.delivered(_) => stay(state)

  def deliverAt(state: Broadcast, p: Proc): List[Step[Broadcast, Outcome, Nothing]] =
    localOf(state, p) match
      case Local.relayed(payload) => enter(withLocal(state, p, Local.delivered(payload)))
      case Local.idle             => stay(state)
      case Local.relaying(_)      => stay(state)
      case Local.delivered(_)     => stay(state)

  object Model extends Machine[Broadcast, Outcome, Nothing], FailureModel:
    val init = Broadcast(
      origin = Origin.idle,
      p1 = Local.idle,
      p2 = Local.idle,
      network = monitors.network.empty
    )
    // a run stops once nothing is in flight and no process is part way through
    def end(state: Broadcast) =
      !originSending(state) && state.network.isEmpty && settled(state.p1) && settled(state.p2)

    object monitors:
      val network = wire.network
      // where a run stops, the processes delivered the same messages
      val agreement = monitor[Broadcast, ReliableBroadcast.Outcome, Nothing, Boolean](false)(
        (_, _, after) => !agreed(after.state)
      )(_ == true).readAtEnds

    object effects:
      def begin(state: Broadcast, payload: Payload) =
        enter(
          state.copy(origin = Origin.sending(payload, Proc.p1))
        )
      def sendNext(state: Broadcast) = state.origin match
        case Origin.sending(payload, next) =>
          enter(
            state.copy(
              origin =
                if next == Proc.p1 then Origin.sending(payload, Proc.p2) else Origin.done(payload),
              network = state.network.send(Msg(next, payload))
            )
          )
        case Origin.idle       => stay(state)
        case Origin.done(_)    => stay(state)
        case Origin.crashed(_) => stay(state)
      def crash(state: Broadcast) = state.origin match
        case Origin.sending(payload, _) =>
          enter(
            state.copy(origin = Origin.crashed(payload))
          )
        case Origin.idle       => stay(state)
        case Origin.done(_)    => stay(state)
        case Origin.crashed(_) => stay(state)
      // On receiving m: forward m to all, then deliver m; a process already busy ignores it.
      def receive(state: Broadcast, msg: Msg) =
        if localOf(state, msg.receiver) == Local.idle then
          enter(
            withLocal(state, msg.receiver, Local.relaying(msg.payload))
          )
        else stay(state)
      // The naive algorithm: deliver m on receiving it, forwarding nothing.
      def receiveNaive(state: Broadcast, msg: Msg) =
        if localOf(state, msg.receiver) == Local.idle then
          enter(
            withLocal(state, msg.receiver, Local.delivered(msg.payload))
          )
        else stay(state)
      def relayP1(state: Broadcast) = relayFrom(state, Proc.p1)
      def relayP2(state: Broadcast) = relayFrom(state, Proc.p2)
      def deliverP1(state: Broadcast) = deliverAt(state, Proc.p1)
      def deliverP2(state: Broadcast) = deliverAt(state, Proc.p2)

    object rules extends Rules:
      on(sender.broadcast):
        where(_.origin == Origin.idle) ~> effects.begin
      on(sender.send):
        where(s => originSending(s) && !s.network.isFull) ~> effects.sendNext
      on(fault.crash):
        where(originSending) ~> effects.crash
      on(monitors.network.deliver):
        always ~> effects.receive
      on(process.relay(Proc.p1)):
        where(s => relaying(s.p1) && !s.network.isFull) ~> effects.relayP1
      on(process.relay(Proc.p2)):
        where(s => relaying(s.p2) && !s.network.isFull) ~> effects.relayP2
      on(process.deliver(Proc.p1)):
        where(s => relayed(s.p1)) ~> effects.deliverP1
      on(process.deliver(Proc.p2)):
        where(s => relayed(s.p2)) ~> effects.deliverP2

  object Naive
      extends Derived(Model.rebind(Model.monitors.network.deliver ~> Model.effects.receiveNaive)),
        NegativeControl

  val safety: Property[Broadcast] = Model.property.expect:
    (after: Expectation[Broadcast, Outcome, Nothing]) ?=> check(onlyBroadcast(after.step.state))
  // The naive algorithm delivers only what was broadcast too: what refutes it is the agreement
  // monitor, which a run that stops with one process delivered and the other not breaks.
  val naiveSafety: Property[Broadcast] = Naive.property.expect:
    (after: Expectation[Broadcast, Outcome, Nothing]) ?=> check(onlyBroadcast(after.step.state))
  // If I deliver a message, I have already forwarded it to all.
  val reliability: Property[Broadcast] = Model.property.across.expect:
    (before: Before[Broadcast], after: Expectation[Broadcast, Outcome, Nothing]) ?=>
      if deliveredValue(after.step.state.p1) != None && deliveredValue(before.state.p1) == None then
        check(relayed(before.state.p1))
      if deliveredValue(after.step.state.p2) != None && deliveredValue(before.state.p2) == None then
        check(relayed(before.state.p2))
  val survivesCrash: Property[Broadcast] = Model.property
    .after(process.deliver(Proc.p2))
    .expect: (after: Expectation[Broadcast, Outcome, Nothing]) ?=>
      check(everyoneDelivered(after.step.state))
      check(after.step.state.origin == Origin.crashed(Payload.hello))

  // once one process delivers, every process delivers the same, whatever else happens meanwhile:
  // the origin's sends or crash, four deliveries and the other process's forward and delivery
  val everyoneDelivers: Progress[Broadcast] =
    Model.leadsTo("everyoneDelivers")(someoneDelivered, everyoneDelivered, within = 9)

  val anyRun: Scenario[Broadcast] = Model.scenario.starts(Model.init).free
  val anyNaiveRun: Scenario[Broadcast] = Naive.scenario.starts(Model.init).free
  // the broadcast, two sends, four deliveries and two forwards and deliveries; a state that holds a
  // channel needs its total spelled: 9 origins x 7 x 7 locals x 70 network contents x 12 classes x
  // 11 steps
  val broadcastLimits = Limits(steps = 11, actions = 11, search = 1 << 20)
  val queries = Vector(
    query("safetyHolds") verify safety in anyRun limits broadcastLimits total 4074840,
    query("reliabilityHolds") verify reliability in anyRun limits broadcastLimits total 4074840,
    query(
      "deliveredDespiteCrash"
    ) find survivesCrash in anyRun limits broadcastLimits total 4074840,
    query(
      "refutesNaiveAgreement"
    ) verify naiveSafety in anyNaiveRun limits broadcastLimits total 4074840
  )
