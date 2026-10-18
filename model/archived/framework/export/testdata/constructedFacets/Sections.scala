package fixture.facetssections

import framework.dsl.*
import framework.dsl.Finite.given

enum Fact:
  case opened, closed, attempted
enum Phase(val status: Fact) extends Recorded[Fact]:
  case open extends Phase(Fact.opened), Live
  case done extends Phase(Fact.closed), Closed
final case class Snapshot(phase: Phase, flag: Boolean)
given Ok[Boolean] = Ok(true)
def phase(using v: View[Snapshot]): Phase = v.get(_.phase)
def phase_=(p: Phase)(using d: Draft[Snapshot, ?, Fact]): Unit = d.set(p)(_.copy(phase = p))
def flag(using v: View[Snapshot]): Boolean = v.get(_.flag)
def flag_=(f: Boolean)(using d: Draft[Snapshot, ?, Fact]): Unit = d.set(_.copy(flag = f))
object hand extends Actor:
  val block = action(this)
  val enterPhase = action(this)
  val rejected = action(this)
  val stopped = action(this)

trait SectionsFacet extends Facet[Snapshot, Boolean, Fact], Phased[Snapshot, Phase]:
  object sectionsFacet:
    object states:
      val ready = is(phase == Phase.open && !flag)
    object effects:
      val close = effect {
        flag = true
        phase = Phase.done
        record(Fact.attempted)
      }
      val refuse = effect(reject(false))
    object rules extends Rules:
      on(hand.block):
        where(states.ready) ~> effects.close
        when[Closed] ~> effects.refuse
      on(hand.enterPhase)(when(Phase.open) ~> enters(Phase.done))
      disabled(hand.stopped)

object Sections
    extends Machine[Snapshot, Boolean, Fact],
      Phased[Snapshot, Phase](_.phase, (s, p) => s.copy(phase = p)),
      SectionsFacet:
  val init = Snapshot(Phase.open, false)
  object rules extends Facets(sectionsFacet.rules)

trait RejectingFacet extends Facet[Boolean, outcomes.Outcome, Nothing]:
  object rejectingFacet:
    object rules extends Rules:
      on(hand.rejected)(always ~> rejects(outcomes.Rejection.notFound))

object Rejecting extends Machine[Boolean, outcomes.Outcome, Nothing], RejectingFacet:
  val init = false
  def end(s: Boolean) = s
  object rules extends Facets(rejectingFacet.rules)
