// The standalone activity Protocol's Completion facet: a held attempt's completion by its worker
// or by activity ID, and the completion claims every completing path asks.
package temporal
package features.activity
package standalone
package protocol

import framework.*
import framework.Finite.given
import framework.outcomes.{Outcome, Rejection}
import framework.realize.Reason
import temporal.realize.{inconclusive, satisfied}
import Bounds.three

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

trait Completion extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  object completion:
    object effects:
      def complete(s: State) = enter(s.copy(phase = Phase.completed), Fact.statusCompleted)

    // A worker's answer settles the attempt it holds; By-ID keeps its notFound checks and its
    // broader Live completion.
    object rules extends Rules:
      on(activityWorker.respondCompleted) {
        when[Held] ~> effects.complete
      }
      on(completionClient.respondCompletedByID) {
        when(Phase.unstarted) ~> rejects(Rejection.notFound)
        when[Closed] ~> rejects(Rejection.notFound)
        when[Live] ~> effects.complete
      }

    object properties:
      val completes: Property[State] =
        property when activityWorker.respondCompleted holds { s => // format: off
          s.state.phase == Phase.completed && s.records(Fact.statusCompleted) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        } // format: on

      val completedByID = property when completionClient.respondCompletedByID holds {
        after => // format: off
        after.state.phase == Phase.completed && after.records(Fact.statusCompleted) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
      } // format: on

    // A closed activity answers a repeated By-ID call NotFound and records nothing, so a Run cannot
    // rule out a silent rejected repeat: the explanations of each By-ID witness disagree, as
    // terminate's do.
    object queries:
      val completed = scenario.actions(
        client.start(),
        activityWorker.poll,
        activityWorker.respondCompleted
      )
      val scheduledCompletion =
        scenario.actions(client.start(), completionClient.respondCompletedByID)

      val completion =
        (query find properties.completes in completed limits three).expect(satisfied)
      val scheduledCompletedByID =
        (query find properties.completedByID in scheduledCompletion limits three)
          .total(11232)
          .expect(inconclusive(Reason.explanationsDisagree))
