package fixture.expectations

import framework.*
import framework.Finite.given
import framework.realize.*

enum Phase derives Finite:
  case open, closed
enum Said derives Finite:
  case accepted
enum Noted derives Finite:
  case closed
object reader extends Actor:
  val close = action(this)

object Subject extends Machine[Phase, Said, Noted]:
  val init = Phase.open
  def end(s: State) = s == Phase.closed
  object monitors:
    val watched = monitor[Phase, Said, Noted, Boolean](false)((seen, _, _) => seen)(seen => seen)
  object rules extends Bindings(reader.close ~> closing)

def closing(s: Phase): List[Step[Phase, Said, Noted]] =
  List(Step(Said.accepted, s, List(Noted.closed)))
val one = Limits("one", steps = 1, actions = 1, search = 32)
val closed = Subject.property("closed") when reader.close holds (_.records(Noted.closed))
val once = Subject.scenario("once").actions(reader.close)

val satisfied = RunExpectation(
  Conformance.conformant,
  PropertyOutcome.satisfied,
  PropertyOutcome.satisfied,
  Disposition.completed,
  Cleanup.succeeded
)

def inconclusive(because: String): RunExpectation = RunExpectation(
  Conformance.conformant,
  PropertyOutcome.inconclusive,
  PropertyOutcome.satisfied,
  Disposition.completed,
  Cleanup.succeeded,
  Some(Reason.neverEvaluated),
  because = because
)
def watched(because: String): RunExpectation = RunExpectation(
  Conformance.conformant,
  PropertyOutcome.satisfied,
  PropertyOutcome.satisfied,
  Disposition.completed,
  Cleanup.succeeded,
  monitors = Vector(
    MonitorExpectation(
      Subject.monitors.watched,
      PropertyOutcome.inconclusive,
      Some(Reason.neverEvaluated),
      because
    )
  )
)
def complete(because: String, monitorBecause: String): RunExpectation = RunExpectation(
  Conformance.inconclusive,
  PropertyOutcome.inconclusive,
  PropertyOutcome.violated,
  Disposition.stoppedByMonitor,
  Cleanup.failed,
  Some(Reason.neverEvaluated),
  Some(Reason.noEvidence),
  Vector(
    MonitorExpectation(
      Subject.monitors.watched,
      PropertyOutcome.inconclusive,
      Some(Reason.neverEvaluated),
      monitorBecause
    )
  ),
  because
)

val modelOnly = query("modelOnly") find closed in once limits one
val liveThenTotal = (query("liveThenTotal") find closed in once limits one).live(satisfied).total(2)
val totalThenLive = (query("totalThenLive") find closed in once limits one).total(2).live(satisfied)
val witnessLiveThenTotal = Subject
  .witness("witnessLiveThenTotal")(reader.close())
  .records(Noted.closed)
  .limits(one)
  .live(satisfied)
  .total(2)
val witnessTotalThenLive = Subject
  .witness("witnessTotalThenLive")(reader.close())
  .records(Noted.closed)
  .limits(one)
  .total(2)
  .live(satisfied)

// Both descriptions state the evaluation point the monitor's declaration defines.
val first =
  (query("explained") find closed in once limits one).live(watched("the monitor reads steps"))
val second = (query("explained") find closed in once limits one)
  .live(watched("the monitor watches every step"))
// An unplayed fixture has neither machine evidence nor a claim evaluation; the monitor reads steps.
val completeFirst = (query("complete") find closed in once limits one)
  .live(complete("the fixture has no evidence or claim evaluation", "the monitor reads steps"))
val completeSecond = (query("complete") find closed in once limits one)
  .live(complete("the fixture is not played or evaluated", "the monitor watches every step"))
val helperValid =
  (query("helperValid") find closed in once limits one).live(inconclusive("no claim evaluation"))
val missing = (query("missing") find closed in once limits one).live(
  RunExpectation(
    Conformance.conformant,
    PropertyOutcome.inconclusive,
    PropertyOutcome.satisfied,
    Disposition.completed,
    Cleanup.succeeded,
    Some(Reason.neverEvaluated)
  )
)
val empty = (query("empty") find closed in once limits one).live(inconclusive(""))
val whitespace = (query("whitespace") find closed in once limits one).live(inconclusive(" \t\n"))
val unicode = (query("unicode") find closed in once limits one).live(inconclusive("\u2003"))
val contractMissing = (query("contractMissing") find closed in once limits one).live(
  RunExpectation(
    Conformance.conformant,
    PropertyOutcome.satisfied,
    PropertyOutcome.violated,
    Disposition.completed,
    Cleanup.succeeded
  )
)
val witnessMissing = Subject
  .witness("witnessMissing")(reader.close())
  .records(Noted.closed)
  .live(
    RunExpectation(
      Conformance.conformant,
      PropertyOutcome.inconclusive,
      PropertyOutcome.satisfied,
      Disposition.completed,
      Cleanup.succeeded,
      Some(Reason.neverEvaluated)
    )
  )
val monitorMissing = (query("monitorMissing") find closed in once limits one).live(
  RunExpectation(
    Conformance.conformant,
    PropertyOutcome.satisfied,
    PropertyOutcome.satisfied,
    Disposition.completed,
    Cleanup.succeeded,
    monitors = Vector(
      MonitorExpectation(
        Subject.monitors.watched,
        PropertyOutcome.inconclusive,
        Some(Reason.neverEvaluated)
      )
    )
  )
)
val monitorEmpty = (query("monitorEmpty") find closed in once limits one).live(watched(""))
val monitorWhitespace =
  (query("monitorWhitespace") find closed in once limits one).live(watched(" \t\n"))
val monitorUnicode =
  (query("monitorUnicode") find closed in once limits one).live(watched("\u2003"))

final case class Closing[S, F](
    close: ClassRef,
    closed: F,
    reach: Seq[ClassRef],
    expect: RunExpectation
) extends CapabilityOf[S, Nothing, F]
object Closing extends CapabilityKind:
  def closes[S](m: Declares[S])(close: ClassRef, closed: m.Fact): Property[S] =
    m.property when close holds (_.records(closed))

object MissingCapability extends Derived(Subject.restrict(reader.close)):
  object capabilities extends Capabilities:
    val closing: Capability = Closing[Phase, Noted](
      reader.close(),
      Noted.closed,
      Seq(),
      RunExpectation(
        Conformance.conformant,
        PropertyOutcome.inconclusive,
        PropertyOutcome.satisfied,
        Disposition.completed,
        Cleanup.succeeded,
        Some(Reason.neverEvaluated)
      )
    )
  object queries:
    capabilities.bound(one)
object EmptyCapability extends Derived(Subject.restrict(reader.close)):
  object capabilities extends Capabilities:
    val closing: Capability =
      Closing[Phase, Noted](reader.close(), Noted.closed, Seq(), inconclusive(""))
  object queries:
    capabilities.bound(one)
object WhitespaceCapability extends Derived(Subject.restrict(reader.close)):
  object capabilities extends Capabilities:
    val closing: Capability =
      Closing[Phase, Noted](reader.close(), Noted.closed, Seq(), inconclusive(" \t\n"))
  object queries:
    capabilities.bound(one)
object ValidCapability extends Derived(Subject.restrict(reader.close)):
  object capabilities extends Capabilities:
    val closing: Capability = Closing[Phase, Noted](
      reader.close(),
      Noted.closed,
      Seq(),
      inconclusive("no claim evaluation")
    )
  object queries:
    capabilities.bound(one)
object UnicodeCapability extends Derived(Subject.restrict(reader.close)):
  object capabilities extends Capabilities:
    val closing: Capability =
      Closing[Phase, Noted](reader.close(), Noted.closed, Seq(), inconclusive("\u2003"))
  object queries:
    capabilities.bound(one)
