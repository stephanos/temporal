package temporal
package features.nexus

import umpire.*
import temporal.Client

// The handler's reply to the server's start request.
enum Reply derives Finite:
  case syncSuccess, async, operationFailed, operationCanceled
  case handlerError(retryable: Boolean)

// How an asynchronous completion settles the operation.
enum Resolution derives Finite:
  case succeeded, failed, canceled

// The Product and System levels share the framework outcome, so a refinement reads the same value
// at both levels.

val operation = Entity()
object Inputs:
  val reply = input[Reply]
val resolution = input[Resolution]

object handler extends Actor:
  val reply = action(this)
    .on(operation)
    .input(Inputs.reply)
    .example(Reply.handlerError(false), "BadRequest")
    .example(Reply.handlerError(true), "Internal")

  // Its classes are names the realization interprets. The result text is metadata of the action,
  // not a domain a state holds.
  val complete = action(this).on(operation).input(resolution).results("Delivery")

object network extends Actor:
  val fault = action(this).on(operation)
object client extends Client:
  val terminate = action(this).on(operation)
object timers:
  val timeout = timer
