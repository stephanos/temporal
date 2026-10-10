// Standalone activity settlement through the completion client's responses by activity ID.
package temporal
package features.activity
package standalone
package system

import framework.*
import framework.Finite.given
import framework.realize.Reason
import temporal.realize.inconclusive
import Bounds.{four, three}

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

// A closed activity answers a repeated By-ID call NotFound and records nothing, so a Run cannot rule
// out a silent rejected repeat: the explanations of each By-ID witness disagree, as terminate's do.
object ByIDCompletion extends Derived(ActivitySystem.unmonitored):

  object properties:
    val completedByID = property when completionClient.respondCompletedByID holds {
      after => // format: off
      after.state.phase == Phase.completed && after.records(Fact.statusCompleted) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
    } // format: on

  object queries:
    val scheduledCompletion =
      scenario.actions(client.start(), completionClient.respondCompletedByID)
    val scheduledCompletedByID =
      (query find properties.completedByID in scheduledCompletion limits three)
        .total(11232)
        .expect(inconclusive(Reason.explanationsDisagree))

object ByIDFailure extends Derived(ActivitySystem.unmonitored):

  object properties: // format: off
    val fatalFailureByID = property when completionClient.respondFailedByID(Failure.fatal) holds { after =>
      after.state.phase == Phase.failed && after.records(Fact.statusFailed) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
    } // format: on

  object queries:
    val heldFatalFailure = scenario.actions(
      client.start(),
      activityWorker.poll,
      completionClient.respondFailedByID(Failure.fatal)
    )
    val heldFailedByID =
      (query find properties.fatalFailureByID in heldFatalFailure limits three)
        .total(16848)
        .expect(inconclusive(Reason.explanationsDisagree))

object ByIDCancellation extends Derived(ActivitySystem.unmonitored):

  object properties:
    val cancelIsRequested =
      property("activitySystem.cancelIsRequested") when client.requestCancel holds (
        _.records(Fact.statusCancelRequested)
      )
    val canceledByID = property when completionClient.respondCanceledByID holds {
      after => // format: off
      after.state.phase == Phase.canceled && after.records(Fact.statusCanceled) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
    } // format: on

  object queries:
    val heldCancellation = scenario.actions(
      client.start(),
      activityWorker.poll,
      client.requestCancel,
      completionClient.respondCanceledByID
    )
    val heldCanceledByID =
      (query find properties.canceledByID in heldCancellation limits four)
        .total(22464)
        .expect(inconclusive(Reason.explanationsDisagree))
    val cancelIsRequested =
      (query(
        "activitySystem.cancelIsRequested"
      ) find properties.cancelIsRequested in heldCancellation limits four)
        .total(22464)
        .expect(inconclusive(Reason.explanationsDisagree))
