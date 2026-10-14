/*
 * # What the two Models say
 *
 * The Kotlin counterpart of the Lean `#guard` pins. In Lean these are checked when the Model file is
 * elaborated; here they run under kotest, and so does everything the builders check at `init` time:
 * the first test to touch `nexusProduct` loads `NexusCallerKt`, which builds every machine, walks the
 * refinement and validates every Property, Scenario, Query and Set in the file. A builder failure
 * therefore surfaces as an `ExceptionInInitializerError` whose cause names the declaration.
 *
 * Both Models declare `three`, `four`, `retry`, `terminalHolds` and the phase types under the same
 * names, so the activity's are imported under aliases; Kotlin has no package alias.
 */
package temporal.feature.pins

import io.kotest.core.spec.style.FunSpec
import io.kotest.matchers.collections.shouldBeEmpty
import io.kotest.matchers.collections.shouldContainExactlyInAnyOrder
import io.kotest.matchers.collections.shouldHaveSize
import io.kotest.matchers.nulls.shouldBeNull
import io.kotest.matchers.shouldBe
import io.kotest.matchers.types.shouldBeInstanceOf
import temporal.feature.activity.AttemptResult
import temporal.feature.activity.activityProduct
import temporal.feature.activity.activityProtocol
import temporal.feature.activity.attemptBound as activityAttemptBound
import temporal.feature.activity.attemptResultStep
import temporal.feature.activity.cancel
import temporal.feature.activity.completion
import temporal.feature.activity.nonRetryableFailure
import temporal.feature.activity.pauseHolds
import temporal.feature.activity.pauseResume
import temporal.feature.activity.retry as activityRetry
import temporal.feature.activity.scheduleToStartTimeout as activityScheduleToStartTimeout
import temporal.feature.activity.startToCloseTimeout as activityStartToCloseTimeout
import temporal.feature.activity.terminalHolds as activityTerminalHolds
import temporal.feature.activity.terminate
import temporal.feature.activity.ProductPhase as ActivityProductPhase
import temporal.feature.activity.ProductState as ActivityProductState
import temporal.feature.nexus.caller.ProductPhase
import temporal.feature.nexus.caller.ProductState
import temporal.feature.nexus.caller.Reply
import temporal.feature.nexus.caller.asyncCompletion
import temporal.feature.nexus.caller.asyncFailure
import temporal.feature.nexus.caller.attemptBound
import temporal.feature.nexus.caller.handlerError
import temporal.feature.nexus.caller.handlerReplyStep
import temporal.feature.nexus.caller.nexusProduct
import temporal.feature.nexus.caller.nexusProtocol
import temporal.feature.nexus.caller.retry
import temporal.feature.nexus.caller.scheduleToStartTimeout
import temporal.feature.nexus.caller.startToCloseTimeout
import temporal.feature.nexus.caller.syncCompletion
import temporal.feature.nexus.caller.terminalHolds
import umpire.Outcome
import umpire.Query

/** A find-Query reaches its Property on its path; a verify-Query holds on every trace within its Limits. */
private fun Query.shouldFind() = run().shouldBeInstanceOf<Outcome.Found>()
private fun Query.shouldVerify() = run() shouldBe Outcome.VerifiedWithinLimits

class NexusCallerPins : FunSpec({

    context("the product machine") {
        test("six phases, and the four the design ends on") {
            nexusProduct.table.states shouldHaveSize 6
            nexusProduct.ends shouldHaveSize 4
        }

        test("every action class the machine steps on: six replies, three resolutions, two faults, one timer") {
            nexusProduct.actionKeys shouldHaveSize 12
        }

        test("a retryable handler error is invisible here: it is the protocol machine that backs off") {
            handlerReplyStep(ProductState(ProductPhase.Scheduled), Reply.HandlerError(retryable = true)).shouldBeEmpty()
        }

        test("what the Model actually reaches: every phase") {
            nexusProduct.reachable().map { it.phase } shouldContainExactlyInAnyOrder ProductPhase.entries
        }

        test("nothing is stuck") {
            nexusProduct.stuck.shouldBeNull()
        }
    }

    context("the protocol machine") {
        test("eight phases, three attempt counts and three deadlines, and the four terminal phases") {
            nexusProtocol.table.states shouldHaveSize 8 * (attemptBound + 1) * 2 * 2 * 2
            nexusProtocol.ends shouldHaveSize 4 * (attemptBound + 1) * 2 * 2 * 2
        }

        test("eight schedule commands, six replies, three resolutions, two faults and four timers") {
            nexusProtocol.actionKeys shouldHaveSize 8 + 6 + 3 + 1 + 1 + 4
        }

        test("the refinement onto the product machine is derived for every row and rejects none") {
            val refinement = checkNotNull(nexusProtocol.refinement)
            refinement.rejected.shouldBeNull()
            refinement.rows shouldHaveSize nexusProtocol.transitions.size
        }
    }

    context("the Queries") {
        test("each functional Query finds its claim on its path") {
            listOf(
                syncCompletion, asyncCompletion, asyncFailure, handlerError, retry,
                scheduleToStartTimeout, startToCloseTimeout,
            ).forEach { it.shouldFind() }
        }

        test("the product claim is verified over every trace of the asynchronous path") {
            terminalHolds.shouldVerify()
        }
    }
})

class StandaloneActivityPins : FunSpec({

    context("the product machine") {
        test("nine phases, and the five the design ends on") {
            activityProduct.table.states shouldHaveSize 9
            activityProduct.ends shouldHaveSize 5
        }
    }

    context("the protocol machine") {
        test("twelve phases, three attempt counts and three deadlines, and the five terminal phases") {
            activityProtocol.table.states shouldHaveSize 12 * (activityAttemptBound + 1) * 2 * 2 * 2
            activityProtocol.ends shouldHaveSize 5 * (activityAttemptBound + 1) * 2 * 2 * 2
        }

        test("a canceled response is refused while no cancel was requested") {
            attemptResultStep(ActivityProductState(ActivityProductPhase.Started), AttemptResult.Canceled).shouldBeEmpty()
        }

        // A pause requested of a running attempt reads as started, so the unpause is a stutter and the
        // attempt's result lands where a started attempt's would; the retry the product can see maps
        // the backoff row onto started -> scheduled.
        test("the refinement onto the product machine is derived for every row and rejects none") {
            val refinement = checkNotNull(activityProtocol.refinement)
            refinement.rejected.shouldBeNull()
            refinement.rows shouldHaveSize activityProtocol.transitions.size
        }
    }

    context("the Queries") {
        test("each functional Query finds its claim on its path") {
            listOf(
                completion, nonRetryableFailure, activityRetry, cancel, terminate, pauseResume,
                activityScheduleToStartTimeout, activityStartToCloseTimeout,
            ).forEach { it.shouldFind() }
        }

        test("the product claims are verified over every trace of their paths") {
            activityTerminalHolds.shouldVerify()
            pauseHolds.shouldVerify()
        }
    }
})
