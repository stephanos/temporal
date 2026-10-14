// The Nexus caller-side Model: one workflow-scheduled Nexus operation, as the caller sees it. The
// Product machine says what an operation does, the System machine says how the server gets there
// and refines it, and the functional Queries are one per side effect that settles the operation.
// No cancellation (fn-79) and no concurrency-limit setup parameter.
//
// Update this Model independently of the implementation. When conformance fails, ask a human
// rather than fitting the Model to the code.
//
// The feature has two levels, each in a folder of its own, because different people read them
// (model/irgen/testdata/layout/lamp is the template):
//
//   - this file: the shared types; the signature (the entities, the inputs, the actors with their
//     actions, the derived observation, the timers and the bounds); and last exports, its IR files;
//   - ../product/Product.scala: Product Phase, State and Fact; NexusProduct, the kind's machine, what
//     an operation does;
//   - system/System.scala: System Phase, State, Fact, timer and composition types; NexusSystem, the
//     System machine that refines it; HandlerWorker, the handler's worker; and NexusCaller, the
//     System with that worker;
//   - system/TrustingCaller.scala: TrustingCaller, the forged control a caller must refuse;
//   - system/ClosePolicy.scala: the close and reset designs.
//
// A machine object reads its header (entity, init, end, evidence), then its sections in order:
// states, refinement, effects, rules, properties and queries. Realization.scala realizes it.
package temporal
package features.nexus
package workflow

import framework.*
import product.NexusProduct
import system.{HandlerWorker, NexusCaller, NexusSystem, TrustingCaller}

// ### Types
//
// A class is one member of a domain, and a constructor that carries finite fields contributes one
// class per assignment of them: handlerError(retryable) is one constructor and two classes, which
// is the granularity an example is written at and what mirrors a protobuf oneof.

// Whether the schedule command sets a deadline.
enum Timeout derives Finite:
  case unset, expires

// ### Signature
//
// Actors are the objects the feature declares for who acts. The reserved actor `system` is the
// server. A fault is an ordinary action of a declared actor, and a timer is system behavior the
// machine owns, so neither is a separate kind. Each actor is an object whose members are the actions
// it takes, and the timers are grouped in objects of their own; each action is named after where it
// is declared, `temporal.features.nexus.workflow.caller.schedule`.

// An operation is scheduled by a caller workflow, and recorded data names one by its scheduled
// event: every history event of the operation carries that event's id.

val workflow = Entity()
val operation = Entity(key = "scheduledEvent", refer = Map("caller" -> workflow))

// The inputs: the schedule's deadlines, which the deadline timers no longer collide with, the
// handler's reply and its completion's resolution.
val scheduleToClose = input[Timeout]
val scheduleToStart = input[Timeout]
val startToClose = input[Timeout]

// The caller workflow, which schedules the operation and inspects it.
object caller extends Actor:
  val schedule = action(this)
    .input(scheduleToClose)
    .input(scheduleToStart)
    .input(startToClose)
    .creates(operation)

  // The caller's inspection of its workflow, which only the forged control (system/) takes.
  val inspect = action(this).on(operation)

// The endpoint's handler, which replies to the start and completes the operation.
object handler extends Actor:
  val reply = temporal.features.nexus.handler.reply.on(operation)
  val complete = temporal.features.nexus.handler.complete.on(operation)

// The network between the caller and the handler, which can fail a transport.
object network extends Actor:
  val fault = temporal.features.nexus.network.fault.on(operation)

// The handler's worker stopping is the worker's own action, `worker.stop`: an action that
// names no entity is behavior no entity records. The Run records the fault, but nothing recorded
// names the operation, so the machines keep their state and record nothing at it.

// A retryable attempt failure writes no history event, so the attempt count is read back through
// DescribeWorkflowExecution. Every other evidence name resolves against the realization's catalog,
// which is why only a derived observation is declared.

val pendingAttempts = Observation(on = operation, read = "attempts")

// One of the operation's deadlines firing, as the product machine sees it, and the backoff.
object timers:
  val timeout = temporal.features.nexus.timers.timeout
  val backoff = timer

// The System's three deadlines, each armed by the schedule's input of its name.
object deadline:
  val scheduleToClose = timer
  val scheduleToStart = timer
  val startToClose = timer

// The bounds of the Queries, beside three and four (temporal.Bounds). Nine actions are enabled before
// the operation is scheduled and eleven once it is, so an exact sequence of two is found among
// ninety-nine candidates, one of three among about a thousand and one of four among about ten
// thousand.
val two = Limits(steps = 2, actions = 2, search = 512)
val control = Limits(steps = 8, actions = 8, search = 262144)

// ### The checked-in IR files of the Nexus caller Model and its close and reset designs (framework.irFile).

object exports:
  // The functional Queries and the realization that runs them are roots beside the machines: Go
  // lowers each Query's witness through the realization into a Testpilot Case (tools/umpire/lower).
  // The product claim read on a System path and the cross-entity Query, which carries the
  // composition with the handler's worker and its claim, are roots too.
  val nexusWorkflow = irFile("nexus-workflow")(
    NexusProduct,
    NexusSystem,
    HandlerWorker,
    NexusCaller,
    temporal.actors.worker.Polling,
    NexusSystem.queries,
    NexusCaller.queries,
    AsyncNexus,
    NexusSystem.capabilities
  )

  // The forged completion a caller must refuse, and the realization that offers it.
  val nexusWorkflowControl =
    irFile("nexus-workflow-control")(TrustingCaller.queries, ForgedControl)

  // The close and reset designs. Each design's Queries are a root, and so is each progress claim.
  val nexusWorkflowClose = irFile("nexus-workflow-close")(
    system.RejectAfterClose.queries,
    system.AckByOriginal.queries,
    system.RetainAndRoute.queries,
    system.ForgetsCancelOnReset.queries,
    system.TruncatesOnReset.queries,
    system.RetainAndRouteBoundedRetry.queries,
    system.RejectAfterCloseWithDeadline.queries,
    system.AckByOriginalWithDeadline.queries,
    system.RetainAndRouteWithDeadline.queries,
    system.RejectAfterClose.properties.rejectAfterCloseProgress,
    system.AckByOriginal.properties.ackByOriginalProgress,
    system.RetainAndRoute.properties.retainAndRouteProgress,
    system.RetainAndRouteBoundedRetry.properties.retainAndRouteBoundedRetryProgress,
    system.RejectAfterCloseWithDeadline.properties.rejectAfterCloseWithDeadlineProgress,
    system.AckByOriginalWithDeadline.properties.ackByOriginalWithDeadlineProgress,
    system.RetainAndRouteWithDeadline.properties.retainAndRouteWithDeadlineProgress,
    system.RetainAndRoute.properties.retainedReachesOwner,
    system.RetainAndRoute.properties.retainedWaitsWithoutRecovery,
    system.RetainAndRouteBoundedRetry.properties.retainedReachesOwnerBoundedRetry
  )
