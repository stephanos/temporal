// Exclusive (fn-163): at most one attempt of an entity is active at a time, a server-side promise of
// its dispatch protocol. A system capability, named for its one use (the activity's admission
// designs and their compositions with the task queue): the name says which promise a design makes,
// and its one definition is here. Its state law lowers to the IR's Property. Its history law, that
// no run admits a second attempt while one is active as the steps' records count it, is the
// record's `atMostOneActiveAttempt` monitor, written in its `monitors` section (fn-163 R4).
package temporal.capabilities.system

import framework.dsl.*

object Exclusive:
  // No step leaves two admitted attempts active, as the design's state counts them.
  def atMostOneActive[S](m: Declares[S])(twoActive: S => Boolean): Property[S] =
    m.property("atMostOneActive").never(s => twoActive(s.state))
