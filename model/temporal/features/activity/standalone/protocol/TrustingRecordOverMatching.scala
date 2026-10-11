package temporal
package features.activity
package standalone
package protocol
import framework.*
import framework.Finite.given
import Bounds.twelve

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object TrustingRecordOverMatching
    extends DerivedComposition(RecordOverMatching.withMember(_.activity -> TrustingRecordMember)),
      NegativeControl:

  object capabilities extends OverMatchingCapabilities(this)

  object queries:
    capabilities.bound(twelve)
    val trustingRecordOverMatchingQueries =
      RecordOverMatching.queries.overMatchingQueries(TrustingRecordOverMatching, capabilities)

// ### The corrected design over each violating provider: the replacement is what must fail.
