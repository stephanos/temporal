package temporal
package features.activity
package standalone
package protocol

import framework.*
import framework.Finite.given

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object TrustingRecordMember extends Derived(TrustingActivityRecord.unmonitored)

// ### Over the opaque queue. The dispatch is the queue's enqueue, a delivery is admission's one
// input, and admission's answer is the acknowledgment: a commit, its answer and the acknowledgment
// are three points a fault can fall between.
