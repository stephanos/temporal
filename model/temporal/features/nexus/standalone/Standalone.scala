// The standalone Nexus operation Model: one operation a client starts directly through
// StartNexusOperationExecution, with no workflow around it, grounded in
// chasm/lib/nexusoperation/{operation.go,operation_statemachine.go}. The client starts, cancels
// and terminates it; the endpoint's handler answers it, synchronously or by starting it and
// completing it later. Its status is read back through DescribeNexusOperationExecution. No retry
// and no deadline is modeled: an attempt that fails retryably reads as scheduled, as BACKING_OFF
// does in Describe (RUNNING), and no Case sets a deadline it lives to see.
//
// Update this Model independently of the implementation. When conformance fails, ask a human
// rather than fitting the Model to the code.
//
// Read top to bottom: the types; the signature (the client, the handler, the operation and their
// actions); and last exports, its IR file. system/System.scala holds NexusSystem, its reading of
// closed rejection and the capabilities it implements. Realization.scala
// realizes it.
package temporal
package features.nexus
package standalone

import umpire.*
import temporal.Client
import system.NexusSystem
import io.temporal.api.workflowservice.v1.*

// ### Signature

// Named by the id the client chose: every request and read carries it.
val operation: Entity = Entity(key = "operationId")

// Who acts: each action is declared in the object of who takes it, and named after where it is
// declared, `temporal.features.nexus.standalone.client.start`.

// The client starts the operation, and requests its cancel or terminates it.
object client extends Client:
  val start = action(this).creates(operation).schema[StartNexusOperationExecutionRequest]
  val requestCancel =
    action(this).on(operation).schema[RequestCancelNexusOperationExecutionRequest]
  val terminate = temporal.features.nexus.client.terminate.on(operation)

// The endpoint's handler replies to the start, and completes an operation it started async.
object handler extends Actor:
  val reply = temporal.features.nexus.handler.reply.on(operation)
  val complete = temporal.features.nexus.handler.complete.on(operation)

given Ok[Outcome] = Ok(Outcome.accepted)

// ### The checked-in IR file of the standalone Nexus operation Model (umpire.irFile).

object exports:
  val nexusStandalone = irFile("nexus-standalone")(
    NexusSystem,
    NexusSystem.implements,
    NexusSystem.queries,
    OperationRealization.standalone
  )
