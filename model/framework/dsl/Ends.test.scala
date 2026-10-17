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

  object PausedEnd extends RecordMachine[Snapshot, Answer, Nothing]:
    val init = Snapshot(Phase.absent)
    def end(s: State) = s.phase == Phase.paused

    object transitions extends Rules

  object ClosedRecord extends RecordMachine[Snapshot, Answer, Nothing]:
    val init = Snapshot(Phase.absent)
    def end(s: State) = s.phase.in[Closed]

    object transitions extends Rules

  object PausedPair extends Composition[Pair](_.left -> ClosedRecord, _.right -> ClosedRecord):
    def end(s: State) = s.left.phase == Phase.paused

    object syncs extends Syncs

  object DerivedPair extends DerivedComposition(PausedPair.withMember(_.right -> PausedEnd))

  object PlainDerivedPair extends Composition(PausedPair.withMember(_.right -> PausedEnd))

  object OpenEnd extends RecordMachine[OpenPhase, Answer, Nothing]:
    val init = OpenPhase.waiting
    def end(s: State) = s == OpenPhase.waiting

    object transitions extends Rules

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
      val s = Snapshot(p)
      assertEquals(PausedEnd.end(s), p == Phase.paused, p.toString)
      assertEquals(ClosedRecord.end(s), p.in[Closed], p.toString)
      assertEquals(PausedPair.end(Pair(s, ClosedRecord.init)), p == Phase.paused, p.toString)
    assert(OpenEnd.end(OpenPhase.waiting))

  test("a derived composition keeps its source's members, and the member it adds"):
    assertEquals(DerivedPair.name, "derivedPair")
    assertEquals(DerivedPair.members, Vector(ClosedRecord, ClosedRecord, PausedEnd))
    assertEquals(PlainDerivedPair.name, "plainDerivedPair")
    assertEquals(PlainDerivedPair.members.last, PausedEnd)
