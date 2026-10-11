// The standalone activity Protocol's Lifecycle facet: the terminate, the worker process's stop, and
// what Describe reports of the activity.
package temporal
package features.activity
package standalone
package protocol

import framework.*
import framework.outcomes.{Outcome, Rejection}
import temporal.capabilities.product.*
import framework.realize.Reason
import temporal.realize.inconclusive
import actors.worker.worker as process

trait Lifecycle extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  object lifecycle:
    object effects:
      def terminate(s: State) = enter(s.copy(phase = Phase.terminated), Fact.statusTerminated)

      // Keeps the state and records nothing; the next step's evidence confirms it, a Known Gap.
      def keep(s: State) = stay(s)

    object rules extends Rules:
      on(client.terminate) {
        when[Closed] ~> rejects(Rejection.notFound)
        when[Live] ~> effects.terminate
      }
      on(process.stop) {
        always ~> effects.keep
      }

    // A terminate settles it, and DescribeActivityExecution reports its status by `activityStatus`.
    // The terminate find starts the activity and stops the worker before the control.
    object capabilities extends Capabilities:
      val terminable: Capability = Terminable(
        terminate = client.terminate,
        settled = Fact.statusTerminated,
        reach = Seq(client.start(), process.stop),
        expect = inconclusive(Reason.explanationsDisagree)
      )
      // The status table is the realization's: what DescribeActivityExecution reports while each
      // fact holds. It is the one Protocol read of the realization layer (an open layering
      // question, fn-160 R9).
      val describable: Capability = Describable(statusTable = realization.activityStatus)
