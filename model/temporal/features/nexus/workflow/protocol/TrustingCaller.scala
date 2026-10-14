// The Nexus caller's control: a caller design that predicts success for a failed completion, which
// the forged-completion Query must refuse. A subject of the Protocol level, beside
// protocol/Protocol.scala (fn-126 decisions 16 and 22). It is the Protocol machine without its
// refinement, its completion forged and the caller's inspection added, declared rather than derived
// from NexusProtocol: a derivation lifts its source machines, and nexus-workflow-control.json holds this
// machine alone.
package temporal
package features.nexus
package workflow
package protocol

import framework.*
import framework.Finite.given
import framework.outcomes.Outcome
import framework.realize.{Alternative, Cleanup, Conformance, Disposition, Exploration, Reason}
import framework.realize.{PropertyOutcome, RunExpectation, Variation}
import temporal.Bounds.eight

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]
// ### Signature

// A failed callback completes the forged control's operation two ways: as the success the control
// forges, and as the failure the runtime still sends.
val forged = choice
val sent = choice

// ### The control

object TrustingCaller
    extends Machine[protocol.State, Outcome, protocol.Fact],
      Phased[protocol.State, Phase](_.phase, (s, p) => s.copy(phase = p)),
      NegativeControl,
      ForgedControl:
  val init = NexusProtocol.init
  val evidence: PartialFunction[Fact, String] = {
    case Fact.nexusOperationTimedOut(_) => "nexusOperationTimedOut"
    case Fact.pendingAttempts           => pendingAttempts.name
  }
  val unobservable = List(timers.backoff)

  object rules extends Facets(forgedControl.rules)

  object queries:
    // The forged control, which every modeled execution that explains the evidence refutes, on a
    // path that inspects the operation around a failed completion.
    val forgedCompletion =
      witness(
        caller.schedule(),
        caller.inspect,
        handler.reply(Reply.async),
        caller.inspect,
        handler.complete(Resolution.failed)
      )
        .records(Fact.nexusOperationCompleted)
        .limits(eight)
        .live(
          RunExpectation(
            Conformance.inconclusive,
            PropertyOutcome.violated,
            contract = PropertyOutcome.violated,
            disposition = Disposition.stoppedByMonitor,
            cleanup = Cleanup.succeeded,
            reason = Some(Reason.everyExplanationViolates),
            conformanceReason = Some(Reason.incomplete),
            because =
              "every modeled execution that explains the evidence of a failed completion refutes the forged success"
          )
        )
        .explore(
          Exploration(
            "nexusControl",
            Vector(
              Variation(
                1,
                Vector(
                  Alternative(
                    "twice",
                    20,
                    Vector(caller.inspect, caller.inspect)
                  ),
                  Alternative("once", 10, Vector(caller.inspect)),
                  Alternative("none", 0, Vector.empty)
                )
              )
            ),
            runs = 1,
            edits = 8,
            dropPrefix = true
          )
        )
