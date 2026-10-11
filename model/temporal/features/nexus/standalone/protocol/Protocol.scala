package temporal
package features.nexus
package standalone
package protocol

import scala.annotation.unused
import framework.*
import temporal.capabilities.system.*
import framework.Finite.given CanEqual[?, ?]
import framework.outcomes.Outcome
import framework.realize.Reason
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

import Fact.{nexusOperationCompleted as NC, nexusOperationStarted as NS}
// 7 phases and whether a cancel was requested: 14 states.
final case class State(phase: Phase, cancelRequested: Boolean) derives Finite
import Fact.{nexusOperationCanceled as NX, nexusOperationFailed as NF}

// What a step records: the status it lands in, and the cancel request Describe reports.
enum Fact derives Finite:
  case statusScheduled, nexusOperationStarted, statusCancelRequested
  case nexusOperationCompleted, nexusOperationFailed, nexusOperationCanceled,
    nexusOperationTerminated

import Fact.nexusOperationTerminated as NT

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object NexusProtocol extends Machine[State, Outcome, Fact], Phased[State, Phase](_.phase), Operation:
  val init = protocol.State(phase = Phase.unstarted, cancelRequested = false)
  // A path ends where the operation is over: `end` reads the Closed role.
  override def end(s: State) = s.phase.in[Closed]

  // The operation's status sets.
  object states:
    def phase(s: State): Phase = s.phase

    // Every phase after the start, live or over: the phases a control is answered in.
    def created(p: Phase): Boolean = p.in[Live] || p.in[Closed]

    // Over, and a control repeats a request the operation took: a recorded cancel, or the
    // terminate that closed it.
    def closed(s: State): Boolean = s.phase.in[Closed]
    def repeats(s: State): Boolean = s.cancelRequested || s.phase == Phase.terminated // scalafix:ok DisableSyntax.==; typed equality

  object refinement extends Refinement(shared.product.NexusProduct):
    def toProduct(s: State): shared.product.State = s.phase match
      case Phase.unstarted | Phase.scheduled => shared.product.State(shared.product.Phase.scheduled)
      case Phase.started                     => shared.product.State(shared.product.Phase.started)
      case Phase.succeeded                   => shared.product.State(shared.product.Phase.succeeded)
      case Phase.failed                      => shared.product.State(shared.product.Phase.failed)
      case Phase.canceled                    => shared.product.State(shared.product.Phase.canceled)
      case Phase.terminated => shared.product.State(shared.product.Phase.terminated)
    def visible(f: Fact): Boolean = f match
      case Fact.statusScheduled | Fact.statusCancelRequested              => false
      case _: NS.type | _: NC.type | _: NF.type | _: NX.type | _: NT.type => true
    def visibleOutcomes(@unused o: Outcome): Boolean = false

  object rules extends Facets(operation.rules)

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
      property("nexusProtocol.cancelIsRequested") when client.requestCancel holds (
        _.records(Fact.statusCancelRequested)
      )

    // A closed operation keeps its state, and answers a control alreadyCompleted, or OK where it
    // repeats a request the operation took, a recorded cancel or the terminate that closed it: the
    // operation's own reading of closedIsRejectedUniformly.
    def closedRejectsOrRepeats(m: Machine[State, Outcome, Fact])(
        rejected: Outcome
    ): Property[State] =
      // The operation's controls are Idempotent (temporal.capabilities.system).
      Idempotent.closedRejectsOrRepeats(m)(states.closed, states.repeats, rejected, Outcome.accepted)
  object capabilities extends FacetCapabilities(operation.capabilities)
  // format: on
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
      query verify shared.product.NexusProduct.properties.terminalIsFinal in asyncThenSucceeded limits three
    val cancelIsRequested =
      (query(
        "nexusProtocol.cancelIsRequested"
      ) find properties.cancelIsRequested in cancelRequest limits three)
        .total(28)
        .live(inconclusive(Reason.explanationsDisagree, because = "a control of a closed operation is answered and records nothing, so a Run explains an unobserved one too, and the executions that explain it disagree"))
