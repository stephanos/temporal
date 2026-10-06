package temporal
package features.nexus

import umpire.*
import temporal.Client
import io.temporal.api.nexus.v1.{HandlerError, StartOperationResponse}
import io.temporal.api.workflowservice.v1.TerminateNexusOperationExecutionRequest

// The handler's reply to the server's start request.
enum Reply derives Finite:
  case syncSuccess, async, operationFailed, operationCanceled
  case handlerError(retryable: Boolean)

// How an asynchronous completion settles the operation.
enum Resolution derives Finite:
  case succeeded, failed, canceled

// A step's outcome. The Product and System machines share these three members, and an outcome reads
// as the refined machine's outcome of the same name.
enum Outcome derives Finite:
  case accepted, notFound, alreadyCompleted

val operation = Entity()
object Inputs:
  val reply = input[Reply]
val resolution = input[Resolution]

object handler extends Actor:
  val reply = action(this)
    .on(operation)
    .input(Inputs.reply)
    .schema[StartOperationResponse]
    .schema[HandlerError]
    .example(Reply.handlerError(false), "BadRequest")
    .example(Reply.handlerError(true), "Internal")

  // The Nexus HTTP completion carries no protobuf message, so it declares no schema and its classes
  // are names the realization interprets. The result text is metadata of the action, not a domain a
  // state holds.
  val complete = action(this).on(operation).input(resolution).results("Delivery")

object network extends Actor:
  val fault = action(this).on(operation)
object client extends Client:
  val terminate = action(this).on(operation).schema[TerminateNexusOperationExecutionRequest]
object timers:
  val timeout = timer
given Ok[Outcome] = Ok(Outcome.accepted)
