package temporal
package features.nexus
package workflow
package protocol

import framework.*
import framework.Finite.given CanEqual[?, ?]

// The corrected policy with a faulty reset that forgets the cancel request, and with it who made
// it. A deliberately wrong design.
val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object ForgetsCancelOnReset
    extends Derived(
      RetainAndRoute.rebind(
        callerSide.reset ~> (RejectAfterClose.closing.effects.reset(Reset.forgetsCancel, _))
      )
    ),
      NegativeControl:

  object queries:
    val forgetsCancelOnResetQueries = RejectAfterClose.queries.designQueries(ForgetsCancelOnReset)
