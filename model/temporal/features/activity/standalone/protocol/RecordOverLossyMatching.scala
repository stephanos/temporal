package temporal
package features.activity
package standalone
package protocol
import framework.*
import framework.Finite.given
import Bounds.twelve
import foundations.taskqueue.protocol.{
  LossyMatchingQueue,
}

// The corrected design where storage loss is assumed, over the interface that allows it.
val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object RecordOverLossyMatching
    extends DerivedComposition(RecordOverMatching.withMember(_.queue -> LossyMatchingQueue)),
      FailureModel:

  object capabilities extends OverMatchingCapabilities(this)

  object queries:
    capabilities.bound(twelve)
    val recordOverLossyMatchingQueries =
      RecordOverMatching.queries.overMatchingQueries(RecordOverLossyMatching, capabilities)
