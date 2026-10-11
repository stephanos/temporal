package temporal
package features.activity
package standalone
package protocol

import framework.*
import framework.Finite.given
import Bounds.five

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object TrustingRecordOverQueue
    extends DerivedComposition(RecordOverQueue.withMember(_.activity -> TrustingRecordMember)),
      NegativeControl:

  object capabilities extends OverQueueCapabilities(this)

  object queries:
    capabilities.bound(five)
    val trustingRecordOverQueueQueries =
      RecordOverQueue.queries.overQueueQueries(TrustingRecordOverQueue, capabilities)

// ### Over the detailed queue, which replaces the opaque one only within the composition: it holds
// where the detailed provider refines the interface it stands in for, and the checks then rely on
// that provider. Each later design swaps one member of the first for a provider of its interface.
