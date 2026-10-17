package samples

import framework.dsl.*
import framework.dsl.Finite.given

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
    def current: Cache =
      Cache(l1, l2, num)

    // a run stops where the layers agree
    override def end =
      agreed(current)

    // What a step showed of the cache: the event it recorded, a write before a read, and the value
    // a read returned, layer 1's after the step, which a layer-2 hit wrote back.
    def observe(history: List[Observation]): List[Observation] =
      val event =
        if recorded(Event.wrote) then Some(Event.wrote)
        else
          facts.find(event =>
            event == Event.readL1 || event == Event.readL2 || event == Event.notFound
          )
      event.fold(history)(event => history :+ Observation(event, l1))

    object transitions extends Transitions:
      on(client.write):
        if num < 3 then effects.write()
      on(client.read):
        effects.read()
      on(timers.l1Expire):
        if l1 != None then effects.expire()

    object effects:
      def write() =
        l1 = Some(num)
        l2 = Some(num)
        num = num + 1
        records(Event.wrote)
      // If the value is not in layer 1, read it from layer 2 and write it back to layer 1;
      // otherwise it is not found.
      def read() =
        if l1 != None then records(Event.readL1)
        else if l2 != None then
          l1 = l2
          records(Event.readL2)
        else records(Event.notFound)
      def expire() =
        l1 = None

    object postconditions:
      // after a read, both layers hold what it returned
      val readFillsL1 =
        after(client.read):
          check(agreed(current))

    object monitors:
      // read the latest write: no read returns a value older than the last write, and a miss is
      // older than any write
      val readAfterWrite =
        monitor(List.empty[Observation])(observe)(readHistoryBroken)
      // a later write stores a greater value
      val writeAfterWrite =
        monitor(List.empty[Observation])(observe)(writeHistoryBroken)

    object progress:
      // all layers contain the same latest written value, one step after they part
      val agreement =
        whenever(!agreed(current)):
          eventually(agreed(current), within = 1)

    object queries:
      // a read served from layer 2: no promise of every read, so a find target here
      val servedFromL2 =
        after(client.read):
          records(Event.readL2)
      val readAfterExpiry =
        find(servedFromL2)
