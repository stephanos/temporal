/* The Nexus caller's domains and step functions: the part of the Model Stainless proves things
 * about (proofs/NexusLemmas.scala) and the framework enumerates (src/nexuscaller). It is written in
 * the Scala subset Stainless accepts: enums, case classes, Int and Boolean, total matches, and the
 * list and step constructors of kernel.prelude, which each side supplies for itself. Everything a
 * Model declares beyond its step functions lives in src/nexuscaller in ordinary Scala.
 *
 * Ported from model/lean/Temporal/Feature/Nexus/Caller/Model.lean and model/go/nexuscaller, in their
 * order; the comments carry over.
 */
package kernel.nexus

import kernel.prelude.*

// ### The input domains
//
// A class is one member of a domain, and a constructor that carries finite fields contributes one
// class per assignment of them: handlerError(retryable) is one constructor and two classes, which
// is the granularity an example is written at and what mirrors a protobuf oneof.

/** Whether the schedule command sets a deadline. */
enum Timeout:
  case unset, expires

/** The handler's reply to the server's start request. */
enum Reply:
  case syncSuccess, async, operationFailed, operationCanceled
  case handlerError(retryable: Boolean)

/** How an asynchronous completion settles the operation. */
enum Resolution:
  case succeeded, failed, canceled

/** A step's outcome. The product and protocol machines share the two members, and an outcome reads
  * as the refined machine's outcome of the same name. */
enum Outcome:
  case accepted, notFound

// ### The product machine
//
// What an operation does, with no account of how. Every Property written against it is carried to
// the protocol machine by the refinement declared there.

enum ProductPhase:
  case scheduled, started, succeeded, failed, canceled, timedOut

final case class ProductState(phase: ProductPhase)

enum ProductFact:
  case nexusOperationScheduled, nexusOperationStarted, nexusOperationCompleted, nexusOperationFailed,
    nexusOperationCanceled, nexusOperationTimedOut

type ProductStep = Step[ProductState, Outcome, ProductFact]

object Product:
  def productStep(phase: ProductPhase, recorded: ProductFact): Steps[ProductStep] =
    one(step(Outcome.accepted, ProductState(phase), facts1(recorded)))

  /** The handler's reply to the server's start request. An operation that has not started yet is
    * the only one a reply can move. */
  def handlerReplyStep(s: ProductState, reply: Reply): Steps[ProductStep] =
    if s.phase != ProductPhase.scheduled then none
    else reply match
      case Reply.syncSuccess       => productStep(ProductPhase.succeeded, ProductFact.nexusOperationCompleted)
      case Reply.async             => productStep(ProductPhase.started, ProductFact.nexusOperationStarted)
      case Reply.operationFailed   => productStep(ProductPhase.failed, ProductFact.nexusOperationFailed)
      case Reply.operationCanceled => productStep(ProductPhase.canceled, ProductFact.nexusOperationCanceled)
      // A retryable handler error leaves the operation where it is: the product machine does not
      // know about backing off, which is the whole of what the protocol machine adds.
      case Reply.handlerError(retryable) =>
        if retryable then none else productStep(ProductPhase.failed, ProductFact.nexusOperationFailed)

  /** The four phases the product machine ends on. */
  def productTerminal(s: ProductState): Boolean =
    s.phase == ProductPhase.succeeded || s.phase == ProductPhase.failed || s.phase == ProductPhase.canceled ||
      s.phase == ProductPhase.timedOut

  /** An asynchronous completion. A completion that arrives after the operation is over is not
    * found, and changes nothing. */
  def completeStep(s: ProductState, resolution: Resolution): Steps[ProductStep] =
    if productTerminal(s) then one(step(Outcome.notFound, s, facts0))
    else resolution match
      case Resolution.succeeded => productStep(ProductPhase.succeeded, ProductFact.nexusOperationCompleted)
      case Resolution.failed    => productStep(ProductPhase.failed, ProductFact.nexusOperationFailed)
      case Resolution.canceled  => productStep(ProductPhase.canceled, ProductFact.nexusOperationCanceled)

  /** A transport fault is an ordinary action of the network. The product machine cannot see one:
    * whether a delivery was retried is the protocol's account of how, not what. */
  def transportFaultStep(s: ProductState): Steps[ProductStep] = none

  /** The handler's worker stopping is a fault the Run records and the operation does not feel. The
    * product machine cannot see it, like the transport fault: a step that kept the state and
    * recorded nothing would be indistinguishable from a stutter, and the refinement would read every
    * stutter as this step. */
  def workerStopStep(s: ProductState): Steps[ProductStep] = none

  /** One of the operation's deadlines firing. Which deadline is the protocol's account of how, so the
    * product machine has one timer, and it fires while the operation runs. */
  def timeoutStep(s: ProductState): Steps[ProductStep] =
    if s.phase == ProductPhase.scheduled || s.phase == ProductPhase.started then
      productStep(ProductPhase.timedOut, ProductFact.nexusOperationTimedOut)
    else none

// ### The protocol machine
//
// How the server gets there: the retry the product machine cannot see, the three timers the
// schedule command sets, and the attempt count a retryable failure raises. Written against the same
// actions, so a Property proved on the product machine is carried here by the refinement.
//
// The machine begins before the operation exists: a state structure has no "no instance yet"
// member, so unscheduled is that member, and it is what makes the three deadline fields reachable
// at anything but their first value -- the schedule command is what sets them.
//
// Not here, for reasons recorded rather than silent: the cancel field and its rows (fn-79), and the
// concurrency-limit rejection, which names no operation and is not modeled until a Query needs it.

enum Phase:
  case unscheduled, scheduled, backingOff, started, succeeded, failed, canceled, timedOut

/** Which timer fired. The history event records it, so a Contract that did not check it would pass a
  * run that timed out on the wrong deadline. */
enum TimeoutType:
  case scheduleToClose, scheduleToStart, startToClose

/** The attempt count is `0..attemptBound`: Lean's `Fin (attemptBound + 1)`. */
final case class ProtocolState(
    phase: Phase,
    attempts: Int,
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout,
)

enum ProtocolFact:
  case nexusOperationScheduled, nexusOperationStarted, nexusOperationCompleted, nexusOperationFailed,
    nexusOperationCanceled
  case nexusOperationTimedOut(timeoutType: TimeoutType)
  /** The attempt count, read through the observation of that name: no history event records it. */
  case pendingAttempts

type ProtocolStep = Step[ProtocolState, Outcome, ProtocolFact]

object Protocol:
  /** Bounds the attempt count. Nothing wires the Limits into a machine's state, so the bound is
    * written here and the saturating successor keeps a retry inside it. */
  val attemptBound: Int = 2

  def validAttempts(a: Int): Boolean = 0 <= a && a <= attemptBound

  /** A retry past the bound stays at it, rather than wrapping as `Fin` arithmetic would. */
  def saturatingSucc(a: Int): Int = {
    require(validAttempts(a))
    if a < attemptBound then a + 1 else a
  }.ensuring(validAttempts)

  /** The four phases the design ends on. A completion that arrives after one of them is not found. */
  def terminalPhase(p: Phase): Boolean =
    p == Phase.succeeded || p == Phase.failed || p == Phase.canceled || p == Phase.timedOut

  /** Scheduled and not yet over: the phases a completion resolves and a timer can fire in. */
  def running(p: Phase): Boolean = p == Phase.scheduled || p == Phase.backingOff || p == Phase.started

  def moves(s: ProtocolState, phase: Phase, recorded: Facts[ProtocolFact]): Steps[ProtocolStep] =
    one(step(Outcome.accepted, s.copy(phase = phase), recorded))

  /** The caller's schedule command. It names the operation's three deadlines, and every one of them
    * is a state field because whether a timer fires is a question about the operation and not about
    * the command that started it. */
  def scheduleStep(s: ProtocolState, scheduleToClose: Timeout, scheduleToStart: Timeout, startToClose: Timeout): Steps[ProtocolStep] =
    if s.phase != Phase.unscheduled then none
    else one(step(Outcome.accepted, ProtocolState(Phase.scheduled, 0, scheduleToClose, scheduleToStart, startToClose),
      facts1(ProtocolFact.nexusOperationScheduled)))

  /** The handler's reply to the server's start request. What the product machine cannot see is the
    * last arm: a retryable failure backs the operation off and raises its attempt count, and the
    * count is read back through the pendingAttempts observation because no history event records it. */
  def handlerReplyStep(s: ProtocolState, reply: Reply): Steps[ProtocolStep] = {
    require(validAttempts(s.attempts))
    if s.phase != Phase.scheduled then none
    else reply match
      case Reply.syncSuccess       => moves(s, Phase.succeeded, facts1(ProtocolFact.nexusOperationCompleted))
      case Reply.async             => moves(s, Phase.started, facts1(ProtocolFact.nexusOperationStarted))
      case Reply.operationFailed   => moves(s, Phase.failed, facts1(ProtocolFact.nexusOperationFailed))
      case Reply.operationCanceled => moves(s, Phase.canceled, facts1(ProtocolFact.nexusOperationCanceled))
      case Reply.handlerError(retryable) =>
        if !retryable then moves(s, Phase.failed, facts1(ProtocolFact.nexusOperationFailed))
        else moves(s.copy(attempts = saturatingSucc(s.attempts)), Phase.backingOff, facts1(ProtocolFact.pendingAttempts))
  }

  /** A transport fault is the same failure arriving as a dropped delivery rather than as a reply. */
  def transportFaultStep(s: ProtocolState): Steps[ProtocolStep] = {
    require(validAttempts(s.attempts))
    if s.phase != Phase.scheduled then none
    else moves(s.copy(attempts = saturatingSucc(s.attempts)), Phase.backingOff, facts1(ProtocolFact.pendingAttempts))
  }

  /** The handler's worker stopping is a fault the Run records and the operation does not feel, so the
    * step keeps the state and records nothing. On a path it is confirmed by the evidence of the step
    * after it, and the Case says so in a Known Gap. */
  def workerStopStep(s: ProtocolState): Steps[ProtocolStep] = one(step(Outcome.accepted, s, facts0))

  /** An asynchronous completion. Before a start, the server records a Started event first, which is
    * why the evidence is two facts and not one -- and why the product machine, which has no
    * backingOff phase to have skipped, could write the completion alone. */
  def completeStep(s: ProtocolState, resolution: Resolution): Steps[ProtocolStep] =
    if terminalPhase(s.phase) then one(step(Outcome.notFound, s, facts0))
    else if s.phase == Phase.unscheduled then none
    else
      val startedFirst: Facts[ProtocolFact] =
        if s.phase != Phase.started then facts1(ProtocolFact.nexusOperationStarted) else facts0
      resolution match
        case Resolution.succeeded => moves(s, Phase.succeeded, startedFirst ++ facts1(ProtocolFact.nexusOperationCompleted))
        case Resolution.failed    => moves(s, Phase.failed, startedFirst ++ facts1(ProtocolFact.nexusOperationFailed))
        case Resolution.canceled  => moves(s, Phase.canceled, startedFirst ++ facts1(ProtocolFact.nexusOperationCanceled))

  /** The backoff timer. It is what makes backingOff a phase the operation leaves rather than a state
    * it is stuck in, and it records nothing: a retry writes no history event. */
  def backoffStep(s: ProtocolState): Steps[ProtocolStep] =
    if s.phase != Phase.backingOff then none else moves(s, Phase.scheduled, facts0)

  /** The schedule-to-close deadline covers the whole operation, so it fires in every running phase
    * -- and only when the schedule command set it. */
  def scheduleToCloseStep(s: ProtocolState): Steps[ProtocolStep] =
    if running(s.phase) && s.scheduleToClose == Timeout.expires then
      moves(s, Phase.timedOut, facts1(ProtocolFact.nexusOperationTimedOut(TimeoutType.scheduleToClose)))
    else none

  /** The schedule-to-start deadline covers the wait for the handler to accept, so it stops at the
    * start. */
  def scheduleToStartStep(s: ProtocolState): Steps[ProtocolStep] =
    if (s.phase == Phase.scheduled || s.phase == Phase.backingOff) && s.scheduleToStart == Timeout.expires then
      moves(s, Phase.timedOut, facts1(ProtocolFact.nexusOperationTimedOut(TimeoutType.scheduleToStart)))
    else none

  /** The start-to-close deadline covers the handler's own work, so it begins at the start. */
  def startToCloseStep(s: ProtocolState): Steps[ProtocolStep] =
    if s.phase == Phase.started && s.startToClose == Timeout.expires then
      moves(s, Phase.timedOut, facts1(ProtocolFact.nexusOperationTimedOut(TimeoutType.startToClose)))
    else none

  /** How a protocol state reads as a product state. A phase of the same name is that phase; backing
    * off is still scheduled, because the product machine cannot see a retry; and an operation not
    * yet scheduled reads as scheduled, because the product machine begins there. Every other field
    * is hidden, which is what a map that does not read it says. */
  def productOf(s: ProtocolState): ProductState = s.phase match
    case Phase.unscheduled | Phase.scheduled | Phase.backingOff => ProductState(ProductPhase.scheduled)
    case Phase.started                                          => ProductState(ProductPhase.started)
    case Phase.succeeded                                        => ProductState(ProductPhase.succeeded)
    case Phase.failed                                           => ProductState(ProductPhase.failed)
    case Phase.canceled                                         => ProductState(ProductPhase.canceled)
    case Phase.timedOut                                         => ProductState(ProductPhase.timedOut)
