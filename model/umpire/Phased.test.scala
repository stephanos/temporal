package umpire

object PhasedFixture:
  enum Phase derives Finite:
    case absent
    case waiting extends Phase, Waiting
    case held extends Phase, Held
    case paused extends Phase, Suspended
    case done extends Phase, Succeeded
    case failed extends Phase, Failed
    case canceled extends Phase, Canceled
    case terminated extends Phase, Terminated
    case timedOut extends Phase, TimedOut

  enum OpenPhase derives Finite:
    case waiting extends OpenPhase, Waiting

  enum Answer derives Finite:
    case accepted

  final case class Snapshot(phase: Phase) derives Finite
  final case class Pair(left: Snapshot, right: Snapshot)

  object DefaultEnd extends Machine[Snapshot, Answer, Nothing], Phased[Snapshot, Phase](_.phase):
    val init = Snapshot(Phase.absent)
    object rules extends Rules

  object OverrideEnd extends Machine[Snapshot, Answer, Nothing], Phased[Snapshot, Phase](_.phase):
    val init = Snapshot(Phase.absent)
    override def end(s: State) = s.phase == Phase.paused
    object rules extends Rules

  object DefaultPair
      extends Composition[Pair](_.left -> DefaultEnd, _.right -> DefaultEnd),
        Phased[Pair, Phase](_.left.phase):
    object syncs extends Syncs

  object OverridePair
      extends Composition[Pair](_.left -> DefaultEnd, _.right -> DefaultEnd),
        Phased[Pair, Phase](_.left.phase):
    override def end(s: State) = s.left.phase == Phase.paused
    object syncs extends Syncs

  object DerivedPair extends DerivedComposition(DefaultPair.withMember(_.right -> OverrideEnd)):
    val phase = summon[Phasing[Pair, Phase]]

  object DerivedAgain extends DerivedComposition(DerivedPair.withMember(_.right -> DefaultEnd)):
    val phase = summon[Phasing[Pair, Phase]]

  object DerivedOverride extends DerivedComposition(OverridePair.withMember(_.right -> DefaultEnd))
  object DerivedOverrideAgain
      extends DerivedComposition(DerivedOverride.withMember(_.right -> OverrideEnd))

  object PlainPair extends Composition[Pair](_.left -> DefaultEnd, _.right -> DefaultEnd):
    def end(s: State) = true
    object syncs extends Syncs

  object PlainDerivedPair extends Composition(PlainPair.withMember(_.right -> OverrideEnd))

  object OpenEnd
      extends Machine[OpenPhase, Answer, Nothing],
        Phased[OpenPhase, OpenPhase](identity):
    val init = OpenPhase.waiting
    object rules extends Rules

  object OpenOverride
      extends Machine[OpenPhase, Answer, Nothing],
        Phased[OpenPhase, OpenPhase](identity):
    val init = OpenPhase.waiting
    override def end(s: State) = s == OpenPhase.waiting
    object rules extends Rules

class PhasedSuite extends munit.FunSuite:
  import PhasedFixture.*

  test("a declared Phased exposes its typed projection to capability claim readers"):
    import DefaultEnd.phased
    val phasing = summon[Phasing[Snapshot, Phase]]
    assertEquals(phasing.phase(Snapshot(Phase.held)), Phase.held)

  test("derived compositions retain the source's typed nested phase through chained derivations"):
    val state = Pair(Snapshot(Phase.done), Snapshot(Phase.waiting))
    assertEquals(DerivedPair.phase.phase(state), Phase.done)
    assertEquals(DerivedAgain.phase.phase(state), Phase.done)
    assert(DerivedPair.end(state))
    assert(DerivedAgain.end(state))
    assert(!DerivedOverrideAgain.end(state))
    assert(DerivedOverrideAgain.end(Pair(Snapshot(Phase.paused), Snapshot(Phase.waiting))))
    assertEquals(PlainDerivedPair.name, "plainDerivedPair")
    assertEquals(PlainDerivedPair.members.last, OverrideEnd)

  test("default ends accept exactly the Closed phases on machines and nested compositions"):
    val closed: Set[Phase] =
      Set(Phase.done, Phase.failed, Phase.canceled, Phase.terminated, Phase.timedOut)
    for p <- Phase.values do
      val s = Snapshot(p)
      assertEquals(DefaultEnd.end(s), closed(p), p.toString)
      assertEquals(DefaultPair.end(Pair(s, DefaultEnd.init)), closed(p), p.toString)

  test("an explicit end overrides Closed and needs no Closed case"):
    for p <- Phase.values do
      assertEquals(OverrideEnd.end(Snapshot(p)), p == Phase.paused, p.toString)
      assertEquals(
        OverridePair.end(Pair(Snapshot(p), DefaultEnd.init)),
        p == Phase.paused,
        p.toString
      )
    assert(OpenOverride.end(OpenPhase.waiting))

  test("a missing Closed case is refused only when the default end is first evaluated"):
    assertEquals(OpenEnd.init, OpenPhase.waiting)
    val error = intercept[IllegalArgumentException](OpenEnd.end(OpenEnd.init))
    assertEquals(
      error.getMessage,
      "requirement failed: openEnd's phase type umpire.PhasedFixture$OpenPhase has no Closed case"
    )
