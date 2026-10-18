package framework.dsl

import framework.dsl.Finite.given

// Where a Machine or a composition may stop is the `end` it declares: a phase's Closed role is no
// default of its own, so a Machine whose end is its Closed phases says so.
object EndsFixture:
  enum Phase:
    case absent
    case waiting extends Phase, Waiting
    case held extends Phase, Held
    case paused extends Phase, Suspended
    case done extends Phase, Succeeded
    case failed extends Phase, Failed
    case canceled extends Phase, Canceled
    case terminated extends Phase, Terminated
    case timedOut extends Phase, TimedOut

  enum OpenPhase:
    case waiting extends OpenPhase, Waiting

  enum Answer:
    case accepted

  given Ok[Answer] = Ok(Answer.accepted)

  final case class Snapshot(phase: Phase)
  final case class Pair(left: Snapshot, right: Snapshot)

  // A Machine of vars whose end is its Closed phases, and one that may stop anywhere.
  object ClosedEnd extends Machine[Answer, Nothing]:
    var phase = Phase.absent

    override def end: Boolean = phase.in[Closed]

    object transitions extends Transitions

  object Anywhere extends Machine[Answer, Nothing]:
    var phase = Phase.absent

    object transitions extends Transitions

  object PausedEnd extends Machine[Answer, Nothing]:
    var phase = Phase.absent
    override def end = phase == Phase.paused

    object transitions extends Transitions

  object ClosedRecord extends Machine[Answer, Nothing]:
    var phase = Phase.absent
    override def end = phase.in[Closed]

    object transitions extends Transitions

  object RightEnd extends Machine[Answer, Nothing]:
    var phase = Phase.absent
    override def end = phase.in[Closed]

    object transitions extends Transitions

  object ReplacedEnd extends Derived(RightEnd.rebind())

  object PausedPair extends Composition[Nothing]:
    val left = ClosedEnd
    val right = RightEnd
    override def end = left.phase == Phase.paused

  object DerivedPair extends Composition(PausedPair.withMember(right = ReplacedEnd))

  object PlainDerivedPair extends Composition(DerivedPair.withMember(right = RightEnd))

  object OpenEnd extends Machine[Answer, Nothing]:
    var phase: OpenPhase = OpenPhase.waiting
    override def end = phase == OpenPhase.waiting

    object transitions extends Transitions

class EndsSuite extends munit.FunSuite:
  import EndsFixture.*

  test("a Machine of vars stops where its declared end holds, and anywhere by default"):
    val closed: Set[Phase] =
      Set(Phase.done, Phase.failed, Phase.canceled, Phase.terminated, Phase.timedOut)
    for p <- Phase.values do
      assertEquals(ClosedEnd.end(ClosedEnd.init.copy(phase = p)), closed(p), p.toString)
      assert(Anywhere.end(Anywhere.init.copy(phase = p)), p.toString)

  test("an explicit end decides where machines and compositions stop, and needs no Closed case"):
    for p <- Phase.values do
      assertEquals(PausedEnd.end(PausedEnd.init.copy(phase = p)), p == Phase.paused, p.toString)
      assertEquals(ClosedRecord.end(ClosedRecord.init.copy(phase = p)), p.in[Closed], p.toString)
      val joined = PausedPair.init.copy(left = ClosedEnd.init.copy(phase = p))
      assertEquals(PausedPair.end(joined), p == Phase.paused, p.toString)
      assertEquals(
        DerivedPair.end(DerivedPair.init.copy(left = ClosedEnd.init.copy(phase = p))),
        p == Phase.paused
      )
    assert(OpenEnd.end(OpenEnd.init))

  test("a derived composition keeps its source's members, and the member it adds"):
    assertEquals(DerivedPair.name, "derivedPair")
    assertEquals(DerivedPair.members, Vector(ClosedEnd, ReplacedEnd))
    assertEquals(PlainDerivedPair.name, "plainDerivedPair")
    assertEquals(PlainDerivedPair.members, Vector(ClosedEnd, RightEnd))
