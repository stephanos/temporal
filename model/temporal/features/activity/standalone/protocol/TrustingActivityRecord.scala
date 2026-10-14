package temporal
package features.activity
package standalone
package protocol

import framework.*
import framework.Finite.given
import framework.outcomes.Outcome
import Bounds.five

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object TrustingActivityRecord
    extends Derived(
      ActivityRecord.rebind(on(activityWorker.poll) {
        always ~> ActivityRecord.admission.effects.admit
      })
    ),
      NegativeControl:

  object capabilities extends AdmissionCapabilities

  object queries:
    capabilities.bound(five)
    val trustingActivityRecordQueries =
      ActivityRecord.queries.admissionQueries(TrustingActivityRecord, capabilities)
