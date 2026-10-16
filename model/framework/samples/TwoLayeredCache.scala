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
    val write = action(this)
    val read = action(this)

  object timers:
    // layer 1 usually lives in memory
    val l1Expire = timer

  final case class Observation(event: Event, value: Option[Int])

  // the value a read returned: layer 1's after the read, which a layer-2 hit wrote back
  def readValue(after: Cache): Option[Int] = after.l1

  def observe(
      history: List[Observation],
      after: Step[Cache, Outcome, Event]
  ): List[Observation] =
    val event =
      if after.records(Event.wrote) then Some(Event.wrote)
      else
        after.facts.find(event =>
          event == Event.readL1 || event == Event.readL2 || event == Event.notFound
        )
    event match
      case Some(event) => history :+ Observation(event, readValue(after.state))
      case None        => history

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

  object Model extends Machine[Cache, Outcome, Event]:
    val init = Cache(l1 = None, l2 = None, num = 0)
    // a run stops where the layers agree
    def end(cache: Cache) = agreed(cache)

    object monitors:
      // read the latest write: no read returns a value older than the last write, and a miss is
      // older than any write
      val readAfterWrite = monitor[Cache, TwoLayeredCache.Outcome, Event, List[Observation]](Nil)(
        (history, _, after) => observe(history, after)
      )(readHistoryBroken)
      // a later write stores a greater value
      val writeAfterWrite = monitor[Cache, TwoLayeredCache.Outcome, Event, List[Observation]](Nil)(
        (history, _, after) => observe(history, after)
      )(writeHistoryBroken)

    object effects:
      def write(cache: Cache) =
        enter(
          cache.copy(
            l1 = Some(cache.num),
            l2 = Some(cache.num),
            num = cache.num + 1
          ),
          Event.wrote
        )
      // A buggy write that stores its value in layer 1 only.
      def writeL1Only(cache: Cache) =
        enter(
          cache.copy(l1 = Some(cache.num), num = cache.num + 1),
          Event.wrote
        )
      // If the value is not in layer 1, read it from layer 2 and write it back to layer 1;
      // otherwise it is not found.
      def read(cache: Cache) =
        if cache.l1 != None then enter(cache, Event.readL1)
        else if cache.l2 != None then enter(cache.copy(l1 = cache.l2), Event.readL2)
        else enter(cache, Event.notFound)
      def expire(cache: Cache) = enter(cache.copy(l1 = None))

    object rules extends Rules:
      on(client.write):
        where(_.num < 3) ~> effects.write
      on(client.read):
        always ~> effects.read
      on(timers.l1Expire):
        where(_.l1 != None) ~> effects.expire

  object WriteThroughL1
      extends Derived(Model.rebind(client.write ~> Model.effects.writeL1Only)),
        NegativeControl

  // after a read, both layers hold what it returned
  val readFillsL1: Property[Cache] =
    Model.property
      .after(client.read)
      .expect: (after: Expectation[Cache, Outcome, Event]) ?=>
        check(agreed(after.step.state))
  // Every write records itself, in the buggy cache too: what refutes it is the readAfterWrite
  // monitor, which a read of a value older than the last write breaks.
  val buggyWritesRecorded: Property[Cache] =
    WriteThroughL1.property
      .after(client.write)
      .expect: (after: Expectation[Cache, Outcome, Event]) ?=>
        records(Event.wrote)
  val servedFromL2: Property[Cache] =
    Model.property
      .after(client.read)
      .expect: (after: Expectation[Cache, Outcome, Event]) ?=>
        records(Event.readL2)

  // all layers contain the same latest written value, one step after they part
  val agreement: Progress[Cache] =
    Model.leadsTo("agreement")(cache => !agreed(cache), agreed, within = 1)

  val anyRun: Scenario[Cache] = Model.scenario.starts(Model.init).free
  val anyBuggyRun: Scenario[Cache] = WriteThroughL1.scenario.starts(Model.init).free
  // three writes, each expired and read back
  val history = Limits(steps = 9, actions = 9, search = 65536)
  val queries = Vector(
    query("readFillsL1Holds") verify readFillsL1 in anyRun limits history,
    query("readAfterExpiry") find servedFromL2 in anyRun limits history,
    query("refutesWriteThroughL1") verify buggyWritesRecorded in anyBuggyRun limits history
  )
