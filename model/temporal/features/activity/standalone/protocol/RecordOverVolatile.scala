package temporal
package features.activity
package standalone
package protocol
import framework.*
import framework.Finite.given
import foundations.taskqueue.protocol.{
  VolatileQueue
}

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object RecordOverVolatile
    extends Composition(RecordOverMatching.withMember(_.queue -> VolatileQueue)),
      NegativeControl
