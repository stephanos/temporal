## What the two Models say
##
## The pins are about the tables, because the tables are what Search and the refinement read. The
## `static:` blocks run in the compile-time VM: a pin there fails the build of this module, which is
## the Nim equivalent of a Lean `#guard`. The `unittest` suites below check the same values at run
## time, where a failure prints the value it saw; the two share every proc they call, so they
## cannot disagree.

import std/[unittest, options, sequtils, algorithm, tables]
import umpire
import nexus_caller as Nexus
import standalone_activity as Activity

# ── Compile-time pins ─────────────────────────────────────────────────────────────────────────────

static:
  # Six phases, and the four the design ends on.
  doAssert Nexus.nexusProduct.table.states.len == 6
  doAssert Nexus.nexusProduct.ends.len == 4
  # Eight phases, three attempt counts and three deadlines.
  doAssert Nexus.nexusProtocol.table.states.len == 8 * (Nexus.attemptBound + 1) * 2 * 2 * 2
  # `refines: nexusProduct` was walked when `nexusProtocol` was built; this pin only restates it.
  doAssert Nexus.nexusProtocol.refinement.get.rejected.isNone
  # Nine product phases; twelve protocol phases.
  doAssert Activity.activityProduct.table.states.len == 9
  doAssert Activity.activityProtocol.table.states.len == 12 * 3 * 8
  doAssert Activity.activityProtocol.refinement.get.rejected.isNone

# ── Nexus ─────────────────────────────────────────────────────────────────────────────────────────

suite "nexus product machine":
  test "six states, four ends":
    check nexusProduct.table.states.len == 6
    check nexusProduct.ends.len == 4

  test "every action class the machine steps on":
    # Six replies, three resolutions, the two faults it cannot see, and the one timer.
    check nexusProduct.catalog.len == 12

  test "a retryable handler error is invisible here":
    check handlerReplyStep(Nexus.ProductState(phase: Nexus.scheduled),
      handlerError(retryable = true)) == @[]

  test "every phase is reachable from the start":
    check nexusProduct.reachable.mapIt(it.phase).sorted == enumerate(Nexus.ProductPhase)
    check nexusProduct.stuck.isNone

suite "nexus protocol machine":
  ## A state written the way a reader names one: the phase, and whichever fields are not at the
  ## value the operation begins with.
  proc at(phase: Nexus.Phase, attempts: Nexus.Attempts = 0, scheduleToClose = unset,
      scheduleToStart = unset, startToClose = unset): Nexus.ProtocolState =
    Nexus.ProtocolState(phase: phase, attempts: attempts, scheduleToClose: scheduleToClose,
      scheduleToStart: scheduleToStart, startToClose: startToClose)

  test "8 * 3 * 2 * 2 * 2 states, 4 * 3 * 8 ends":
    check nexusProtocol.table.states.len == 192
    check nexusProtocol.ends.len == 96

  test "23 action classes, catalog in canonical order":
    # Eight schedule commands, six replies, three resolutions, two faults, four timers.
    check nexusProtocol.catalog.len == 8 + 6 + 3 + 1 + 1 + 4
    check nexusProtocol.catalog[0 .. 1] == @["backoff", "complete-canceled"]

  test "the machine begins before the operation exists":
    check nexusProtocol.starts == @[at(unscheduled)]

  test "a retryable error backs off and raises the count, saturating":
    check protocolHandlerReplyStep(at(Nexus.scheduled), handlerError(true)) ==
      @[Nexus.ProtocolStep(outcome: Nexus.accepted, state: at(backingOff, 1),
        facts: @[pendingAttempts])]
    check protocolHandlerReplyStep(at(Nexus.scheduled, attempts = attemptBound),
      handlerError(true)).mapIt(it.state.attempts) == @[Nexus.Attempts(attemptBound)]

  test "a completion before the start records Started first":
    check protocolCompleteStep(at(backingOff, 1), Nexus.succeeded).mapIt(it.facts).concat ==
      @[nexusOperationStarted, nexusOperationCompleted]
    check protocolCompleteStep(at(Nexus.started), Nexus.succeeded).mapIt(it.facts).concat ==
      @[nexusOperationCompleted]
    check protocolCompleteStep(at(Nexus.timedOut), Nexus.succeeded) ==
      @[Nexus.ProtocolStep(outcome: Nexus.notFound, state: at(Nexus.timedOut))]

  test "which timer fired is recorded":
    check scheduleToStartStep(at(Nexus.scheduled, scheduleToStart = expires))
      .mapIt(it.facts).concat == @[nexusOperationTimedOut(scheduleToStart)]
    check startToCloseStep(at(Nexus.scheduled, startToClose = expires)) == @[]

  test "the refinement passes and reads a retry as a stutter":
    let r = nexusProtocol.refinement.get
    check r.rejected.isNone
    check r.rows.len == nexusProtocol.transitions.len
    check r.rows.toTable["scheduled-0-unset-unset-unset-handlerReply-async"] ==
      some("handlerReply-async")
    check r.rows.toTable["scheduled-0-unset-unset-unset-handlerReply-handlerError-true"].isNone
    check r.rows.toTable["started-0-unset-unset-expires-startToClose"] == some("timeout")

suite "nexus queries":
  test "every find-query's scenario reaches its property":
    for q in [syncCompletion, asyncCompletion, asyncFailure, Nexus.handlerError, retry,
        scheduleToStartTimeout, startToCloseTimeout]:
      check q.rejected.isNone
      check q.witness.len > 0
  test "the product claim verifies on the async path":
    check terminalHolds.mode == verify
    check terminalHolds.rejected.isNone
  test "a stopped worker replies nothing":
    check stoppedWorkerRepliesNothing.rejected.isNone

# ── Standalone activity ───────────────────────────────────────────────────────────────────────────

suite "activity machines":
  test "product: nine states, five ends":
    check activityProduct.table.states.len == 9
    check activityProduct.ends.len == 5
  test "protocol: 12 * 3 * 8 states, 5 * 3 * 8 ends":
    check activityProtocol.table.states.len == 288
    check activityProtocol.ends.len == 120
  test "a cancellation nobody requested has no row":
    check protocolAttemptResultStep(
      Activity.ProtocolState(phase: Activity.Phase.started), Activity.canceled) == @[]
  test "the refinement passes":
    check activityProtocol.refinement.get.rejected.isNone

suite "activity queries":
  test "every find-query's scenario reaches its property":
    for q in [completion, nonRetryableFailure, Activity.retry, cancel, Activity.terminate,
        pauseResume, Activity.scheduleToStartTimeout, Activity.startToCloseTimeout]:
      check q.rejected.isNone
  test "the two transition claims verify":
    check Activity.terminalHolds.rejected.isNone
    check pauseHolds.rejected.isNone
  test "a stopped worker starts nothing":
    check stoppedWorkerStartsNothing.rejected.isNone
