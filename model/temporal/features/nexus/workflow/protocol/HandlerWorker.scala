package temporal
package features.nexus
package workflow
package protocol

import framework.*
import framework.Finite.given
import temporal.actors.worker.worker

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object HandlerWorker
    extends Derived(temporal.actors.worker.Polling.restrict(worker.stop, worker.serve))

// ### The operation and the handler's worker
//
// The Protocol machine's worker stop is a stutter row: the operation cannot see its handler's
// worker, so the schedule-to-start Scenario orders the stop before the request explicitly.
// Composed with the worker of the handler's task queue, the stop is the worker's own phase change
// and every reply is the worker serving, so a reply has a row only while the worker polls. No
// functional Query reads the composition; it is what the cross-entity claim is verified over.
