"""
# What the two Models say

The claims below are about the tables, because the tables are what Search, the Behavior Fingerprint
and Contract lowering read. A `@match` arm that stopped saying what it says would fail here.

The Lean file pins with `#guard` at elaboration time. Here the tables, the refinements and the Query
results were already computed once, when the Model modules were precompiled (each is a `const`), so
this file only *reads* them: a red `@test` is a claim about a table that already built. A Model
whose refinement is rejected or whose Query finds nothing never gets this far; it fails at
`using TemporalModels`, which is the compile-time path.

Run with `julia --project -e 'using Pkg; Pkg.test()'` or, in a Revise session, `include("pins.jl")`.
"""

using Test
using Umpire
import Moshi

# In the real package these three `include`s are `src/TemporalModels.jl`; the wrapper module is what
# makes `using ..Worker` resolve inside the two Models.
module TemporalModels
    include("worker.jl")
    include("nexus_caller.jl")
    include("standalone_activity.jl")
end

# Julia has no `open`, a module exports nothing unless told to, and `using` is only legal at a
# module's top level. So each Model's pins live in a module of their own, which is also what keeps
# the two Models' `Phase`, `attemptBound` and `retry_result` from colliding.
module NexusPins

using Test, Umpire
using ..TemporalModels.NexusCaller: Phase, ProductPhase, Reply, Resolution, Timeout, TimeoutType,
    ProtocolFact, ProtocolOutcome, ProductState, ProtocolState, TStep, attemptBound,
    nexusProduct, nexusProtocol, handlerReplyStep, workerStopStep, protocolHandlerReplyStep,
    protocolCompleteStep, protocolWorkerStopStep, startToCloseStep, scheduleToCloseStep,
    scheduleToStartStep,
    syncCompletion_result, asyncCompletion_result, asyncFailure_result, handlerError_result,
    retry_result, scheduleToStartTimeout_result, startToCloseTimeout_result, terminalHolds_result,
    stoppedWorkerRepliesNothing_result

# A state written the way a reader names one: the phase, and whichever fields are not at the value
# the operation begins with.
at(phase::Phase.T; attempts = 0, scheduleToClose = Timeout.unset, scheduleToStart = Timeout.unset,
   startToClose = Timeout.unset) =
    ProtocolState(; phase, attempts, scheduleToClose, scheduleToStart, startToClose)

@testset "Nexus caller" begin
    @testset "the product machine" begin
        # Six phases, and the four the design ends on.
        @test length(nexusProduct.table.states) == 6
        @test length(nexusProduct.ends) == 4
        # Every action class the machine steps on: six replies, three resolutions, the two faults it
        # cannot see, and the one timer.
        @test length(Umpire.actionKeys(nexusProduct)) == 12
        # A retryable handler error is invisible here: it is the protocol machine that backs off.
        @test handlerReplyStep(ProductState(phase = ProductPhase.scheduled), Reply.handlerError(true)) == []
        # What the Model actually reaches: every phase.
        @test Set(s.phase for s in Umpire.reachable(nexusProduct)) == Set(instances(ProductPhase.T))
        @test Umpire.stuck(nexusProduct) === nothing
    end

    @testset "the protocol machine" begin
        @test length(nexusProtocol.table.states) == 8 * (attemptBound + 1) * 2 * 2 * 2   # 192
        @test length(nexusProtocol.ends) == 4 * (attemptBound + 1) * 2 * 2 * 2           # 96
        # Eight schedule classes, six replies, three resolutions, the two faults, four timers.
        @test length(Umpire.actionKeys(nexusProtocol)) == 8 + 6 + 3 + 1 + 1 + 4        # 23
        @test nexusProtocol.starts == [at(Phase.unscheduled)]

        # The retryable error backs off and raises the count; at the bound it saturates.
        @test protocolHandlerReplyStep(at(Phase.scheduled), Reply.handlerError(true)) ==
            [TStep(outcome = ProtocolOutcome.accepted, state = at(Phase.backingOff; attempts = 1),
                   facts = [ProtocolFact.pendingAttempts])]
        @test only(protocolHandlerReplyStep(at(Phase.scheduled; attempts = attemptBound),
                                            Reply.handlerError(true))).state.attempts == attemptBound

        # A completion before a start records Started first; after one, the completion alone.
        @test only(protocolCompleteStep(at(Phase.backingOff; attempts = 1), Resolution.succeeded)).facts ==
            [ProtocolFact.nexusOperationStarted, ProtocolFact.nexusOperationCompleted]
        @test only(protocolCompleteStep(at(Phase.started), Resolution.succeeded)).facts ==
            [ProtocolFact.nexusOperationCompleted]
        @test protocolCompleteStep(at(Phase.timedOut), Resolution.succeeded) ==
            [TStep(outcome = ProtocolOutcome.notFound, state = at(Phase.timedOut))]

        # Each timer fires in its own phases, and only when the schedule command set it.
        @test startToCloseStep(at(Phase.scheduled; startToClose = Timeout.expires)) == []
        @test only(startToCloseStep(at(Phase.started; startToClose = Timeout.expires))).state.phase == Phase.timedOut
        @test scheduleToCloseStep(at(Phase.started)) == []
        @test only(scheduleToStartStep(at(Phase.scheduled; scheduleToStart = Timeout.expires))).facts ==
            [ProtocolFact.nexusOperationTimedOut(TimeoutType.scheduleToStart)]

        # The worker stop is a stutter here and invisible on the product.
        @test protocolWorkerStopStep(at(Phase.scheduled; scheduleToStart = Timeout.expires)) ==
            [TStep(outcome = ProtocolOutcome.accepted, state = at(Phase.scheduled; scheduleToStart = Timeout.expires))]
        @test workerStopStep(ProductState(phase = ProductPhase.scheduled)) == []

        @test Umpire.stuck(nexusProtocol) === nothing
    end

    @testset "the refinement" begin
        # `refines = nexusProduct` with `map = productOf` walked every protocol row through the map
        # when the const was built; a rejected row would have failed `using`. Pin the shape anyway.
        @test nexusProtocol.refinement.rejected === nothing
        @test length(nexusProtocol.refinement.rows) == length(Umpire.transitions(nexusProtocol))
        @test nexusProtocol.refinement.rows["scheduled-0-unset-unset-unset-handlerReply-async"] ==
            "scheduled-handlerReply-async"
        # The backoff timer is a product stutter; a protocol timer resolves to the product's one timer.
        @test nexusProtocol.refinement.rows["backingOff-1-unset-unset-unset-backoff"] === nothing
        @test nexusProtocol.refinement.rows["started-0-unset-unset-expires-startToClose"] ==
            "started-timeout"
    end

    @testset "the Queries" begin
        # Every find-Query's Scenario reaches its Property; the verify-Query holds on every path.
        for result in (syncCompletion_result, asyncCompletion_result, asyncFailure_result,
                       handlerError_result, retry_result, scheduleToStartTimeout_result,
                       startToCloseTimeout_result)
            @test result isa Umpire.Found
        end
        @test terminalHolds_result isa Umpire.Verified
        @test length(retry_result.witness.path) == 4
        @test [Umpire.key(c) for (c, _) in scheduleToStartTimeout_result.witness.path] ==
            ["schedule-unset-expires-unset", "workerStop", "scheduleToStart"]
        @test stoppedWorkerRepliesNothing_result isa Umpire.Verified
    end
end

end # module NexusPins

module ActivityPins

using Test, Umpire
using ..TemporalModels.NexusCaller: Timeout
using ..TemporalModels.StandaloneActivity: Phase, ProductPhase, AttemptResult, Control, ProtocolFact,
    ProtocolState, attemptBound,
    activityProduct, activityProtocol, protocolAttemptResultStep, protocolControlStep,
    completion_result, nonRetryableFailure_result, retry_result, cancel_result, terminate_result,
    pauseResume_result, scheduleToStartTimeout_result, startToCloseTimeout_result,
    terminalHolds_result, pauseHolds_result, stoppedWorkerStartsNothing_result

at(phase::Phase.T; attempts = 0, scheduleToClose = Timeout.unset, scheduleToStart = Timeout.unset,
   startToClose = Timeout.unset) =
    ProtocolState(; phase, attempts, scheduleToClose, scheduleToStart, startToClose)

@testset "Standalone activity" begin
    @testset "the product machine" begin
        @test length(activityProduct.table.states) == 9
        @test length(activityProduct.ends) == 5
        @test Umpire.stuck(activityProduct) === nothing
    end

    @testset "the protocol machine" begin
        @test length(activityProtocol.table.states) == 12 * (attemptBound + 1) * 8   # 288
        @test length(activityProtocol.ends) == 5 * (attemptBound + 1) * 8            # 120
        # A canceled result with no cancel requested is not honored.
        @test protocolAttemptResultStep(at(Phase.started), AttemptResult.canceled) == []
        # A retryable failure while a cancel is requested settles as canceled, not backed off.
        @test only(protocolAttemptResultStep(at(Phase.cancelRequested), AttemptResult.failed(true))).state.phase ==
            Phase.canceled
        # A pause of a running attempt is only requested; unpause returns it to started.
        @test only(protocolControlStep(at(Phase.started), Control.pause)).state.phase == Phase.pauseRequested
        @test only(protocolControlStep(at(Phase.pauseRequested), Control.unpause)).state.phase == Phase.started
        @test Umpire.stuck(activityProtocol) === nothing
    end

    @testset "the refinement" begin
        @test activityProtocol.refinement.rejected === nothing
        @test length(activityProtocol.refinement.rows) == length(Umpire.transitions(activityProtocol))
    end

    @testset "the Queries" begin
        for result in (completion_result, nonRetryableFailure_result, retry_result, cancel_result,
                       terminate_result, pauseResume_result, scheduleToStartTimeout_result,
                       startToCloseTimeout_result)
            @test result isa Umpire.Found
        end
        @test terminalHolds_result isa Umpire.Verified
        @test pauseHolds_result isa Umpire.Verified
        @test stoppedWorkerStartsNothing_result isa Umpire.Verified
    end
end

end # module ActivityPins

@testset "authoring errors are pinned to their line" begin
    NC = TemporalModels.NexusCaller

    # An undeclared action in `steps` is rejected when the macro expands, not when the table builds.
    # `@macroexpand` runs the macro without evaluating the result, so the error is the macro's own.
    err = try
        @macroexpand @machine broken begin
            var"for" = operation
            state = NC.ProductState
            outcome = NC.ProductOutcome.T
            fact = NC.ProductFact.T
            starts = [scheduled]
            ends = [succeeded]
            steps = (handlerRepy = NC.handlerReplyStep,)
        end
        nothing
    catch e
        e
    end
    @test err isa LoadError && err.error isa Umpire.DSLError
    @test occursin("`steps` names `handlerRepy`", err.error.msg)
    @test err.error.line.line == (@__LINE__) - 8      # the `steps = (...)` line

    # A missing `@match` arm is a runtime error from Moshi, so it is the table build that catches it:
    # every state times every class calls the step function, and the arm that is missing is hit.
    partial(reply::NC.Reply.Type) = @match reply begin
        NC.Reply.syncSuccess => 1
        NC.Reply.async       => 2
    end
    @test_throws Moshi.Match.MatchError partial(NC.Reply.operationFailed)
end
