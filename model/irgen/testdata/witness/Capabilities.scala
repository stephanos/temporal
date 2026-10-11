package fixture.witness

import framework.*
import worker.{begin, finish}

final case class Settling[S, F](
    settle: ClassRef | Composed,
    settled: F,
    reach: Seq[ClassRef | Composed]
) extends CapabilityOf[S, Nothing, F]
object Settling extends CapabilityKind:
  def settles[S](m: Declares[S])(settle: ClassRef | Composed, settled: m.Fact): Property[S] =
    m.property when settle holds (_.records(settled))

final case class Alternatives[S, F](
    settle: ClassRef,
    settled: F,
    other: F,
    reach: Seq[ClassRef]
) extends CapabilityOf[S, Nothing, F]
object Alternatives extends CapabilityKind:
  def settles[S](m: Declares[S])(settle: ClassRef, settled: m.Fact, other: m.Fact): Property[S] =
    m.property when settle holds (after => after.records(settled) || after.records(other))

val capBounds = Limits("capBounds", steps = 2, actions = 2, search = 512)
object Capable extends Derived(Subject.restrict(begin, finish)):
  object capabilities extends Capabilities:
    val settling: Capability =
      Settling[Phase, Note](finish(Input.first), Note.completed, Seq(begin()))
  object queries:
    capabilities.bound(capBounds)

object CapableAlternative extends Derived(Subject.restrict(begin, finish)):
  object capabilities extends Capabilities:
    val settling: Capability = Alternatives[Phase, Note](
      finish(Input.first),
      Note.completed,
      Note.other,
      Seq(begin())
    )
  object queries:
    capabilities.bound(capBounds)

object CapableTogether extends Composition[Joined](_.left -> Subject, _.right -> Other):
  def end(s: State) = closed(s.left) && closed(s.right)
  object syncs extends Syncs
  object capabilities extends Capabilities:
    val settling: Capability = Settling[Joined, String](
      CapableTogether.own(_.left, finish(Input.first)),
      "left_completed",
      Seq()
    )
  object queries:
    capabilities.bound(capBounds)

val capabilityDuplicate = Capable
  .witness("capabilityDuplicate")(begin(), finish(Input.first))
  .records(Note.completed)
  .ends(closed)
  .limits(bounded)
  .live(expected)
val capabilityDifferentInput = Capable
  .witness("capabilityDifferentInput")(begin(), finish(Input.second))
  .records(Note.completed)
val capabilityDifferentFact = Capable
  .witness("capabilityDifferentFact")(begin(), finish(Input.first))
  .records(Note.other)
val capabilityAlternative = CapableAlternative
  .witness("capabilityAlternative")(begin(), finish(Input.first))
  .records(Note.completed)
val capabilityMemberDuplicate = CapableTogether
  .witness("capabilityMemberDuplicate")(
    CapableTogether.own(_.left, finish(Input.first))
  )
  .records(_.left -> Subject, Note.completed)
val capabilityDifferentMember = CapableTogether
  .witness("capabilityDifferentMember")(
    CapableTogether.own(_.right, finish(Input.first))
  )
  .records(_.right -> Other, Note.completed)
