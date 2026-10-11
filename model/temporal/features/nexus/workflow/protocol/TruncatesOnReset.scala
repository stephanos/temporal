package temporal
package features.nexus
package workflow
package protocol

import framework.*
import framework.Finite.given CanEqual[?, ?]

// The corrected policy with a faulty reset that does not reapply what the original run recorded. A
// deliberately wrong design.
val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object TruncatesOnReset
    extends Derived(
      RetainAndRoute.rebind(
        callerSide.reset ~> (RejectAfterClose.closing.effects.reset(Reset.truncates, _))
      )
    ),
      NegativeControl:

  object queries:
    val truncatesOnResetQueries = RejectAfterClose.queries.designQueries(TruncatesOnReset)
