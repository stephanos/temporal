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
  final case class Cache(l1: Option[UpTo[2]], l2: Option[UpTo[2]], num: UpTo[3])

  // what each step did, in place of a log of reads and writes
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

  // What a monitor remembers of the history: the last value written, and whether the law it
  // watches has broken.
  final case class Seen(lastWrite: Option[UpTo[2]], broken: Boolean)

  // the value a read returned: layer 1's after the read, which a layer-2 hit wrote back
  def readValue(after: Cache): Option[UpTo[2]] = after.l1

  // whether `value` is older than `than`: a miss is older than any value, and nothing is older than
  // no write at all
  def older(value: Option[UpTo[2]], than: Option[UpTo[2]]): Boolean = than match
    case None       => false
    case Some(last) =>
      value match
        case None       => true
        case Some(read) => read < last

  def agreed(cache: Cache): Boolean = cache.l1 == cache.l2

  object Model extends Machine[Cache, Outcome, Event]:
    val init = Cache(l1 = None, l2 = None, num = UpTo(0))
    // a run stops where the layers agree
    def end(cache: Cache) = agreed(cache)

    object monitors:
      // read the latest write: no read returns a value older than the last write, and a miss is
      // older than any write
      val readAfterWrite = monitor[Cache, TwoLayeredCache.Outcome, Event, Seen](
        Seen(lastWrite = None, broken = false)
      )((seen, _, after) =>
        if after.records(Event.wrote) then seen.copy(lastWrite = after.state.l1)
        else if after.records(Event.readL1) || after.records(Event.readL2) ||
          after.records(Event.notFound)
        then seen.copy(broken = seen.broken || older(readValue(after.state), seen.lastWrite))
        else seen
      )(_.broken)
      // a later write stores a greater value
      val writeAfterWrite = monitor[Cache, TwoLayeredCache.Outcome, Event, Seen](
        Seen(lastWrite = None, broken = false)
      )((seen, _, after) =>
        if after.records(Event.wrote) then
          Seen(
            lastWrite = after.state.l1,
            broken = seen.broken || !older(seen.lastWrite, after.state.l1)
          )
        else seen
      )(_.broken)

    object effects:
      def write(cache: Cache) =
        enter(
          cache.copy(
            l1 = Some(UpTo(cache.num)),
            l2 = Some(UpTo(cache.num)),
            num = UpTo(cache.num + 1)
          ),
          Event.wrote
        )
      // A buggy write that stores its value in layer 1 only.
      def writeL1Only(cache: Cache) =
        enter(
          cache.copy(l1 = Some(UpTo(cache.num)), num = UpTo(cache.num + 1)),
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
      on(client.write)(where(_.num < 3) ~> effects.write)
      on(client.read)(always ~> effects.read)
      on(timers.l1Expire)(where(_.l1 != None) ~> effects.expire)

  object WriteThroughL1
      extends Derived(Model.rebind(client.write ~> Model.effects.writeL1Only)),
        NegativeControl

  // after a read, both layers hold what it returned
  val readFillsL1: Property[Cache] =
    Model.property.when(client.read) holds (after => agreed(after.state))
  // Every write records itself, in the buggy cache too: what refutes it is the readAfterWrite
  // monitor, which a read of a value older than the last write breaks.
  val buggyWritesRecorded: Property[Cache] =
    WriteThroughL1.property.when(client.write) holds (after => after.records(Event.wrote))
  val servedFromL2: Property[Cache] =
    Model.property.when(client.read) holds (after => after.records(Event.readL2))

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
