// Fenced (fn-163): a delivery of a stale attempt is fenced off, a server-side promise of the dispatch
// protocol. A system capability, named for its one use (the held dispatch race): the server meets the
// stale message and rejects it rather than starting the attempt. Its law lowers to the IR's Property.
package temporal.capabilities.system

import framework.*

object Fenced:
  // A delivery the server met and rejected records its rejection.
  def staleDeliveryRejected[S](m: Declares[S])(deliver: ClassRef, rejected: m.Fact): Property[S] =
    m.property("staleDeliveryRejected") when deliver holds (_.records(rejected))
