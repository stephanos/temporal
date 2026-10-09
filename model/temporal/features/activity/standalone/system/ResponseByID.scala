// Standalone activity settlement through service responses by activity ID.
package temporal
package features.activity
package standalone
package system

import framework.*
import framework.realize.Reason
import temporal.realize.inconclusive
import Bounds.{four, three}

// A closed activity answers a repeated By-ID call NotFound and records nothing, so a Run cannot rule
// out a silent rejected repeat: the explanations of each By-ID witness disagree, as terminate's do.
object ByIDCompletion extends Derived(ActivitySystem.unmonitored):
  object properties:
    val completedByID = property when service.respondCompletedByID holds { after =>
      after.state.phase == Phase.completed && after.records(Fact.statusCompleted)
    }
  object queries:
    val scheduledCompletion = scenario.actions(client.start(), service.respondCompletedByID)
    val scheduledCompletedByID =
      (query find properties.completedByID in scheduledCompletion limits three)
        .total(11232)
        .expect(inconclusive(Reason.explanationsDisagree))

object ByIDFailure extends Derived(ActivitySystem.unmonitored):
  object properties:
    val fatalFailureByID = property when service.respondFailedByID(Failure.fatal) holds { after =>
      after.state.phase == Phase.failed && after.records(Fact.statusFailed)
    }
  object queries:
    val heldFatalFailure = scenario.actions(
      client.start(),
      worker.poll,
      service.respondFailedByID(Failure.fatal)
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
    val canceledByID = property when service.respondCanceledByID holds { after =>
      after.state.phase == Phase.canceled && after.records(Fact.statusCanceled)
    }
  object queries:
    val heldCancellation = scenario.actions(
      client.start(),
      worker.poll,
      client.requestCancel,
      service.respondCanceledByID
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
