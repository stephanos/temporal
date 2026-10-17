package samples

import framework.dsl.*
import framework.dsl.Finite.given

// Modelled after https://github.com/quint-co/quint/blob/main/examples/classic/distributed/TwoLayeredCache/two_layered_cache.qnt
//
// A two-layered cache of one key: a write stores its value in both layers, a read serves from layer
// 1, else from layer 2, writing the value back into layer 1, and layer 1 is volatile, so it may
// expire at any time. Every write stores a greater value than the one before, and no read may see
// an older value than the last write.
object TwoLayeredCache:
  // the cache stores the values 0, 1 and 2, written in that order
  final case class Cache(l1: Option[Int], l2: Option[Int], num: Int)

  // what each step did
  enum Event:
    case wrote, readL1, readL2, notFound

  enum Outcome:
    case accepted

  given Ok[Outcome] = Ok(Outcome.accepted)

  object client extends Actor:
    val write = action
    val read = action

  object timers extends Internal:
    // layer 1 usually lives in memory
    val l1Expire = timer

  final case class Observation(event: Event, value: Option[Int])

  // the value a read returned: layer 1's after the read, which a layer-2 hit wrote back
  def readValue(after: Cache): Option[Int] = after.l1

  def observe(
      history: List[Observation],
      after: Step[State, Outcome, Event]
  ): List[Observation] =
    val event =
      if after.records(Event.wrote) then Some(Event.wrote)
      else
        after.facts.find(event =>
          event == Event.readL1 || event == Event.readL2 || event == Event.notFound
        )
    event match
      case Some(event) =>
        history :+ Observation(event, readValue(Model.at(after.state)(Model.current)))
      case None => history

  def readHistoryBroken(history: List[Observation]): Boolean =
    history
      .foldLeft((Option.empty[Int], false)) { case ((lastWrite, broken), observation) =>
        if observation.event == Event.wrote then (observation.value, broken)
        else (lastWrite, broken || older(observation.value, lastWrite))
      }
      ._2

  def writeHistoryBroken(history: List[Observation]): Boolean =
    history
      .foldLeft((Option.empty[Int], false)) { case ((lastWrite, broken), observation) =>
        if observation.event == Event.wrote then
          (observation.value, broken || !older(lastWrite, observation.value))
        else (lastWrite, broken)
      }
      ._2

  // whether `value` is older than `than`: a miss is older than any value, and nothing is older than
  // no write at all
  def older(value: Option[Int], than: Option[Int]): Boolean = than match
    case None       => false
    case Some(last) =>
      value match
        case None       => true
        case Some(read) => read < last

  def agreed(cache: Cache): Boolean = cache.l1 == cache.l2

  object Model extends Machine[Outcome, Event]:
    // the two layers and the value the next write stores: empty, and the first write stores 0
    var l1 = Option.empty[Int]
    var l2 = Option.empty[Int]
    var num = 0

    // the cache as one record, which the predicates above read
    def current: Cache = Cache(l1, l2, num)

    // a run stops where the layers agree
    override def end = agreed(current)

    object transitions extends Rules:
      on(client.write):
        where(num < 3) ~> effects.write()
      on(client.read):
        always ~> effects.read()
      on(timers.l1Expire):
        where(l1 != None) ~> effects.expire()

    object effects:
      def write(): Unit =
        l1 = Some(num)
        l2 = Some(num)
        num = num + 1
        record(Event.wrote)
      // A buggy write that stores its value in layer 1 only.
      def writeL1Only(): Unit =
        l1 = Some(num)
        num = num + 1
        record(Event.wrote)
      // If the value is not in layer 1, read it from layer 2 and write it back to layer 1;
      // otherwise it is not found.
      def read(): Unit =
        if l1 != None then record(Event.readL1)
        else if l2 != None then
          l1 = l2
          record(Event.readL2)
        else record(Event.notFound)
      def expire(): Unit = l1 = None

    object postconditions:
      // after a read, both layers hold what it returned
      val readFillsL1: Property =
        property
          .after(client.read)
          .expect:
            check(agreed(current))
      val servedFromL2: Property =
        property
          .after(client.read)
          .expect:
            records(Event.readL2)

    object monitors:
      // read the latest write: no read returns a value older than the last write, and a miss is
      // older than any write
      val readAfterWrite = monitor[State, TwoLayeredCache.Outcome, Event, List[Observation]](Nil)(
        (history, _, after) => observe(history, after)
      )(readHistoryBroken)
      // a later write stores a greater value
      val writeAfterWrite = monitor[State, TwoLayeredCache.Outcome, Event, List[Observation]](Nil)(
        (history, _, after) => observe(history, after)
      )(writeHistoryBroken)

    object progress:
      // all layers contain the same latest written value, one step after they part
      val agreement: Progress[State] =
        Model.leadsTo("agreement")(!agreed(current), agreed(current), within = 1)

    object queries:
      val anyRun: Scenario = scenario.starts(init).free
      // three writes, each expired and read back
      val history = Limits(steps = 9, actions = 9, search = 65536)
      val readFillsL1Holds = query verify postconditions.readFillsL1 in anyRun limits history
      val readAfterExpiry = query find postconditions.servedFromL2 in anyRun limits history

  object WriteThroughL1
      extends Derived(Model.rebind(client.write ~> Model.effects.writeL1Only())),
        NegativeControl:

    object postconditions:
      // Every write records itself, in the buggy cache too: what refutes it is the readAfterWrite
      // monitor, which a read of a value older than the last write breaks.
      val buggyWritesRecorded: Property =
        property
          .after(client.write)
          .expect:
            records(Event.wrote)

    object queries:
      val anyBuggyRun: Scenario = scenario.starts(Model.init).free
      val refutesWriteThroughL1 =
        query verify postconditions.buggyWritesRecorded in anyBuggyRun limits Model.queries.history
