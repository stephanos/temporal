// Composition progress claims and composed fairness the lifter refuses (fn-150), each a root of its
// own: fairness on another composition than the claim's, on a machine's claim or a machine's
// assumptions, mixed with a machine's actions or spread over two compositions, a sync class named
// from the side that cannot supply its inputs, a member action stepping only with its pair, an action
// no sync pairs, and a bound of no step.
package fixture.composedprogressrejects

import framework.dsl.*
import fixture.composedprogress.{drain, fill, flowing, send, Line, Pipeline, Tanks}

object Mirror extends Composition[Line](_.left -> Tanks, _.right -> Tanks):
  def end(l: State) = true
  object syncs extends Syncs:
    sync("mirrored", _.left -> send, _.right -> fill)

val anotherOwner: Progress[Line] =
  Mirror.leadsTo("anotherOwner")(_ => true, _ => true, within = 1, flowing)

val onAMachine =
  Tanks.leadsTo("onAMachine")(_ => true, _ => true, within = 1, flowing)

object AssumingFlow extends Derived(Tanks.assuming(flowing))

val mixed = assume.fair(send).fair(Pipeline.own(_.right, send))
val mixedFairness: Progress[Line] =
  Pipeline.leadsTo("mixedFairness")(_ => true, _ => true, within = 1, mixed)

val spread = assume.fair(Pipeline.own(_.right, send), Mirror.own(_.right, send))
val spreadFairness: Progress[Line] =
  Pipeline.leadsTo("spreadFairness")(_ => true, _ => true, within = 1, spread)

val oneSided = assume.fair(Pipeline.synced(_.left -> send()))
val oneSidedClass: Progress[Line] =
  Pipeline.leadsTo("oneSidedClass")(_ => true, _ => true, within = 1, oneSided)

val paired = assume.fair(Pipeline.own(_.left, send))
val pairedOwn: Progress[Line] =
  Pipeline.leadsTo("pairedOwn")(_ => true, _ => true, within = 1, paired)

val unpaired = assume.fair(Pipeline.synced(_.left -> drain))
val unpairedSync: Progress[Line] =
  Pipeline.leadsTo("unpairedSync")(_ => true, _ => true, within = 1, unpaired)

val noStep: Progress[Line] =
  Pipeline.leadsTo("noStep")(_ => true, _ => true, within = 0)
