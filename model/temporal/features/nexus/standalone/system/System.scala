package temporal
package features.nexus
package standalone
package system

import scala.annotation.unused
import umpire.*
import umpire.outcomes.{Outcome, Rejection}
import umpire.realize.Reason
import temporal.capabilities.*
import temporal.realize.inconclusive
import temporal.Bounds.three

// The statuses DescribeNexusOperationExecution reports: scheduled and started read RUNNING.
enum Phase derives Finite:
  case unstarted
  case scheduled extends Phase, Waiting
  case started extends Phase, Held
  case succeeded extends Phase, Succeeded
  case failed extends Phase, Failed
  case canceled extends Phase, Canceled
  case terminated extends Phase, Terminated

// 7 phases and whether a cancel was requested: 14 states.
final case class State(phase: Phase, cancelRequested: Boolean) derives Finite

// What a step records: the status it lands in, and the cancel request Describe reports.
enum Fact derives Finite:
  case statusScheduled, nexusOperationStarted, statusCancelRequested
  case nexusOperationCompleted, nexusOperationFailed, nexusOperationCanceled,
    nexusOperationTerminated

object NexusSystem extends Machine[State, Outcome, Fact], Phased[State, Phase](_.phase):
  import Phase.*

  val init = system.State(phase = unstarted, cancelRequested = false)
  // A path ends where the operation is over: `end` reads the Closed role.
  override def end(s: State) = s.phase.in[Closed]

  // The operation's status sets.
  object states:
    def phase(s: State): Phase = s.phase

    // Every phase after the start, live or over: the phases a control is answered in.
    def created(p: Phase): Boolean = p.in[Live] || p.in[Closed]

    // A control that repeats an accepted request id is answered OK even after close.
    val repeatedRequestsAnswer =
      "a repeated request id is answered OK after close: operation.go RequestCancel and Terminate"

  object refinement extends Refinement(product.NexusProduct):
    def toProduct(s: State): product.State = s.phase match
      case Phase.unstarted | Phase.scheduled => product.State(product.Phase.scheduled)
      case Phase.started                     => product.State(product.Phase.started)
      case Phase.succeeded                   => product.State(product.Phase.succeeded)
      case Phase.failed                      => product.State(product.Phase.failed)
      case Phase.canceled                    => product.State(product.Phase.canceled)
      case Phase.terminated                  => product.State(product.Phase.terminated)
    def visible(f: Fact): Boolean = f match
      case Fact.statusScheduled | Fact.statusCancelRequested => false
      case _                                                 => true
    def visibleOutcomes(@unused o: Outcome): Boolean = false

  object effects:
    import Fact.*

    def start(@unused s: State) =
      enter(system.State(phase = scheduled, cancelRequested = false), statusScheduled)

    // TransitionStarted, or a synchronous completion straight from scheduled; a canceled answer
    // settles it canceled (operation.go invocationResultCancel, onCanceled).
    def syncSuccess(s: State) = enter(s.copy(phase = succeeded), nexusOperationCompleted)
    def syncFailure(s: State) = enter(s.copy(phase = failed), nexusOperationFailed)
    def syncCanceled(s: State) = enter(s.copy(phase = canceled), nexusOperationCanceled)
    def async(s: State) = enter(s.copy(phase = started), nexusOperationStarted)

    // An async operation's completion; a canceled failure settles it canceled.
    def complete(s: State, r: Resolution) =
      r match
        case Resolution.succeeded => enter(s.copy(phase = succeeded), nexusOperationCompleted)
        case Resolution.failed    => enter(s.copy(phase = failed), nexusOperationFailed)
        case Resolution.canceled  => enter(s.copy(phase = canceled), nexusOperationCanceled)

    // RequestCancel records the request and leaves the operation live: it is sent to the handler
    // only once started (operation.go RequestCancel).
    def requestCancel(s: State) = enter(s.copy(cancelRequested = true), statusCancelRequested)

    // Terminate settles a live operation terminated (TransitionTerminated).
    def terminate(s: State) = enter(s.copy(phase = terminated), nexusOperationTerminated)

    // A control that repeats a request the operation took is the same request, answered OK.
    def repeated(s: State) = stay(s)

  object rules extends Rules:
    on(client.start) {
      when(unstarted) ~> effects.start
    }
    on(handler.reply(Reply.syncSuccess)) {
      when(scheduled) ~> effects.syncSuccess
    }
    on(handler.reply(Reply.operationFailed)) {
      when(scheduled) ~> effects.syncFailure
    }
    on(handler.reply(Reply.operationCanceled)) {
      when(scheduled) ~> effects.syncCanceled
    }
    on(handler.reply(Reply.async)) {
      when(scheduled) ~> effects.async
    }
    on(handler.complete) {
      when(started) ~> effects.complete
    }

    // A repeated cancel request is the same request; one of a closed operation is alreadyCompleted,
    // unless it repeats one the operation took (operation.go RequestCancel).
    on(client.requestCancel) {
      when(states.created).where(_.cancelRequested) ~> effects.repeated
      when[Closed].where(!_.cancelRequested) ~>
        rejects(Rejection.failedPrecondition).because("operation already completed")
      // Started and not over: the phases a control settles or records a request in.
      when[Live].where(!_.cancelRequested) ~> effects.requestCancel
    }

    // A repeated terminate of a terminated operation is the same request, answered OK; any other
    // control of a closed one is alreadyCompleted (operation.go Terminate).
    on(client.terminate) {
      when(terminated) ~> effects.repeated
      when(succeeded, failed, canceled) ~>
        rejects(Rejection.failedPrecondition).because("operation already completed")
      when(scheduled, started) ~> effects.terminate
    }

  // What the operation promises of its own: its reading of closed rejection, which its capabilities
  // put in place of the shared Property, and its recorded cancel request.
  object properties:
    // Cancellation requests are recorded while the operation is in flight. A cancel request records
    // the request and leaves the operation live. Its find starts from the start, for the same reason
    // as Terminable.terminateSettles.
    // This does not promise cancellation, which only the work's answer settles, or a second request's
    // answer (Nexus answers ErrCancellationAlreadyRequested).
    // See chasm/lib/nexusoperation/operation.go.
    val cancelIsRequested =
      property("nexusSystem.cancelIsRequested") when client.requestCancel holds (
        _.records(Fact.statusCancelRequested)
      )

    // A closed operation keeps its state, and answers a control alreadyCompleted, or OK where it
    // repeats a request the operation took, a recorded cancel or the terminate that closed it: the
    // operation's own reading of closedIsRejectedUniformly.
    def closedRejectsOrRepeats(m: Machine[State, Outcome, Fact])(
        rejected: Outcome
    ): Property[State] =
      m.property holdsAcross ((before, after) =>
        !before.phase.in[Closed] ||
          (after.state == before && (after.outcome == rejected || after.outcome == Outcome.accepted &&
            (before.cancelRequested || before.phase == Phase.terminated)))
      )

  // What the operation is, as the Properties in model/temporal/capabilities read it: it closes, a client
  // terminates it, and DescribeNexusOperationExecution reports its status.
  // It receives the Properties without listing them, each named `nexusSystem.<property>`. It reads the
  // realization, which reads this machine, so it waits in a section, which initializes on its first
  // use.
  //
  // The rejection is the operation's own: alreadyCompleted, a FailedPrecondition, where the activity
  // answers NotFound (operation.go ErrOperationAlreadyCompleted). Each same-step Property's find starts
  // the operation, which no handler answers, so it stays running, then takes the control. A Run
  // explains an unobserved control of a closed operation too, which records nothing, so a Run of a
  // terminate or cancel find leaves the claim inconclusive: its explanations disagree. The cancel
  // find is the operation's own, cancelIsRequested in queries.
  object capabilities extends Capabilities:
    val closable: Capability = Closable(
      rejected = Outcome.rejected(Rejection.failedPrecondition)
    )
    val terminable: Capability = Terminable(
      terminate = client.terminate,
      settled = Fact.nexusOperationTerminated,
      reach = Seq(client.start),
      expect = inconclusive(Reason.explanationsDisagree)
    )
    val describable: Capability = Describable(statusTable = operationStatus)
    overriding(
      Closable.closedIsRejectedUniformly -> properties.closedRejectsOrRepeats,
      because = states.repeatedRequestsAnswer
    )

  object queries:
    capabilities.bound(three)

    val asyncThenSucceeded = scenario.actions(
      client.start,
      handler.reply(Reply.async),
      handler.complete(Resolution.succeeded)
    )
    // The operation started, which no handler answers, then the cancel request.
    val cancelRequest = scenario.actions(client.start, client.requestCancel)

    val terminalHolds =
      query verify product.NexusProduct.properties.terminalIsFinal in asyncThenSucceeded limits three
    val cancelIsRequested =
      (query(
        "nexusSystem.cancelIsRequested"
      ) find properties.cancelIsRequested in cancelRequest limits three)
        .total(28)
        .expect(inconclusive(Reason.explanationsDisagree))
