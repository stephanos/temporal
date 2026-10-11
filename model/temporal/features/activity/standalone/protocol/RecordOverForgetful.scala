package temporal
package features.activity
package standalone
package protocol
import framework.*
import framework.Finite.given
import foundations.taskqueue.protocol.{
  ForgetfulQueue,
}

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object RecordOverForgetful
    extends Composition(RecordOverMatching.withMember(_.queue -> ForgetfulQueue)),
      NegativeControl
