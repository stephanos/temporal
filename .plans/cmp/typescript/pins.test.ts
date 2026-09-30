// pins.test.ts — what the two Models say, pinned.
//
// The Lean file checks these with `#guard` while it compiles; here they run under vitest. The
// claims are about the tables, because the tables are what Search, the fingerprint and Contract
// lowering read: a `switch` arm that stopped saying what it says would fail here. The type-level
// pins at the end run under `vitest --typecheck` (or `tsc --noEmit`), not at runtime.

import { describe, expect, expectTypeOf, it } from "vitest";
import { checkRefinement, defineMachine, run, scenario, type AnyMachine, type PhaseOf, type Query } from "./umpire";
import * as nexus from "./nexus_caller";
import * as activity from "./standalone_activity";

/** Every find-Query of a functional set. */
function findQueries(...queries: Query<AnyMachine>[]) {
  return queries.map((q) => [q.name, q] as const);
}

describe("nexus caller: the product machine", () => {
  const { nexusProduct, handlerReplyStep, workerStopStep, ProductPhase } = nexus;

  it("has six phases, and the four the design ends on", () => {
    expect(nexusProduct.table.states).toHaveLength(6);
    expect(nexusProduct.endStates).toHaveLength(4);
  });

  it("steps on twelve action classes: six replies, three resolutions, two faults it cannot see, one timer", () => {
    expect(nexusProduct.actionKeys).toHaveLength(12);
  });

  it("does not see a retryable handler error: the protocol machine is what backs off", () => {
    expect(handlerReplyStep({ phase: "scheduled" }, { reply: { kind: "handlerError", retryable: true } })).toEqual([]);
  });

  it("does not see the worker stop at all", () => {
    expect(workerStopStep({ phase: "scheduled" })).toEqual([]);
  });

  it("reaches every phase from its start", () => {
    const phases = new Set(nexusProduct.reachable.map((s) => s.phase));
    expect([...phases].sort()).toEqual([...ProductPhase.members].sort());
  });
});

describe("nexus caller: the protocol machine", () => {
  const { nexusProtocol, attemptBound, protocolHandlerReplyStep, protocolCompleteStep } = nexus;

  /** A state written the way a reader names one: the phase, and whichever fields are not at their first value. */
  const at = (phase: nexus.Phase, rest: Partial<Omit<nexus.ProtocolState, "phase">> = {}): nexus.ProtocolState => ({
    phase,
    attempts: 0,
    scheduleToClose: "unset",
    scheduleToStart: "unset",
    startToClose: "unset",
    ...rest,
  });

  it("has 8 * 3 * 2 * 2 * 2 states and 4 * 3 * 8 ends", () => {
    expect(nexusProtocol.table.states).toHaveLength(8 * (attemptBound + 1) * 2 * 2 * 2);
    expect(nexusProtocol.endStates).toHaveLength(4 * (attemptBound + 1) * 2 * 2 * 2);
  });

  it("steps on 8 + 6 + 3 + 1 + 1 + 4 action classes, in canonical order", () => {
    expect(nexusProtocol.actionKeys).toHaveLength(8 + 6 + 3 + 1 + 1 + 4);
    expect(nexusProtocol.actionKeys.slice(0, 2)).toEqual(["backoff", "complete-canceled"]);
  });

  it("backs off on a retryable handler error and raises the count, recorded by the derived observation", () => {
    expect(protocolHandlerReplyStep(at("scheduled"), { reply: { kind: "handlerError", retryable: true } })).toEqual([
      { outcome: "accepted", state: at("backingOff", { attempts: 1 }), facts: ["pendingAttempts"] },
    ]);
  });

  it("saturates the count rather than wrapping", () => {
    const steps = protocolHandlerReplyStep(at("scheduled", { attempts: attemptBound }), {
      reply: { kind: "handlerError", retryable: true },
    });
    expect(steps.map((s) => s.state.attempts)).toEqual([attemptBound]);
  });

  it("records the Started event first for a completion before the start, and not after", () => {
    expect(protocolCompleteStep(at("backingOff", { attempts: 1 }), { resolution: "succeeded" }).flatMap((s) => s.facts))
      .toEqual(["nexusOperationStarted", "nexusOperationCompleted"]);
    expect(protocolCompleteStep(at("started"), { resolution: "succeeded" }).flatMap((s) => s.facts))
      .toEqual(["nexusOperationCompleted"]);
  });

  it("does not find a completion after the operation is over", () => {
    expect(protocolCompleteStep(at("timedOut"), { resolution: "succeeded" })).toEqual([
      { outcome: "notFound", state: at("timedOut"), facts: [] },
    ]);
  });

  it("refines the product machine: every row maps to a product row or a product stutter", () => {
    expect(checkRefinement(nexusProtocol).rejected).toBeNull();
  });
});

describe("nexus caller: the Queries", () => {
  it.each(
    findQueries(
      nexus.syncCompletion,
      nexus.asyncCompletion,
      nexus.asyncFailure,
      nexus.handlerError,
      nexus.retry,
      nexus.scheduleToStartTimeout,
      nexus.startToCloseTimeout,
    ),
  )("%s finds its claim on its path", (_name, q) => {
    expect(run(q)).toMatchObject({ outcome: "found" });
  });

  it("verifies the product claim over every trace of the async path", () => {
    expect(run(nexus.terminalHolds)).toMatchObject({ outcome: "verified" });
  });

  it("verifies that no handler replies while its worker is stopped", () => {
    expect(run(nexus.stoppedWorkerRepliesNothing)).toMatchObject({ outcome: "verified" });
  });
});

describe("standalone activity: the machines", () => {
  const { activityProduct, activityProtocol, attemptBound, attemptResultStep, protocolAttemptResultStep } = activity;

  it("product: nine phases, five ends", () => {
    expect(activityProduct.table.states).toHaveLength(9);
    expect(activityProduct.endStates).toHaveLength(5);
  });

  it("protocol: 12 * 3 * 8 states, 5 * 3 * 8 ends", () => {
    expect(activityProtocol.table.states).toHaveLength(12 * (attemptBound + 1) * 8);
    expect(activityProtocol.endStates).toHaveLength(5 * (attemptBound + 1) * 8);
  });

  it("honors a canceled answer only under a cancel request, on both machines", () => {
    expect(attemptResultStep({ phase: "started" }, { result: { kind: "canceled" } })).toEqual([]);
    const started: activity.ProtocolState = {
      phase: "started",
      attempts: 1,
      scheduleToClose: "unset",
      scheduleToStart: "unset",
      startToClose: "unset",
    };
    expect(protocolAttemptResultStep(started, { result: { kind: "canceled" } })).toEqual([]);
  });

  it("refines the product machine: every row is a product stutter or maps to some product row", () => {
    // Holds under the revised spec: `pauseRequested` reads as started and a retryable failure is
    // a visible product row (started -> scheduled, cancelRequested -> canceled).
    expect(checkRefinement(activityProtocol).rejected).toBeNull();
  });
});

describe("standalone activity: the Queries", () => {
  it.each(
    findQueries(
      activity.completion,
      activity.nonRetryableFailure,
      activity.retry,
      activity.cancel,
      activity.terminate,
      activity.pauseResume,
      activity.scheduleToStartTimeout,
      activity.startToCloseTimeout,
    ),
  )("%s finds its claim on its path", (_name, q) => {
    expect(run(q)).toMatchObject({ outcome: "found" });
  });

  it.each([
    ["terminalHolds", activity.terminalHolds],
    ["pauseHolds", activity.pauseHolds],
    ["stoppedWorkerStartsNothing", activity.stoppedWorkerStartsNothing],
  ] as const)("%s verifies", (_name, q) => {
    expect(run(q)).toMatchObject({ outcome: "verified" });
  });
});

// ---------------------------------------------------------------------------------------------
// Type-level pins: what `tsc` rejects. These never execute; a wrong Model fails `vitest --typecheck`.
// ---------------------------------------------------------------------------------------------

describe("type-level pins", () => {
  it("derives the domains' types from their enumerations", () => {
    expectTypeOf<nexus.Reply["kind"]>().toEqualTypeOf<
      "syncSuccess" | "async" | "operationFailed" | "operationCanceled" | "handlerError"
    >();
    expectTypeOf<nexus.ProtocolState["attempts"]>().toEqualTypeOf<0 | 1 | 2>();
    expectTypeOf<PhaseOf<typeof nexus.nexusCaller>>().toEqualTypeOf<
      | `operation.${nexus.Phase}`
      | "worker.polling"
      | "worker.stopped"
    >();
  });

  it("rejects a step for an action no `action` declared", () => {
    defineMachine({
      name: "wrong",
      for: nexus.operation,
      state: nexus.ProductState,
      starts: ["scheduled"],
      ends: ["succeeded"],
      actions: [nexus.handlerReply],
      timers: ["timeout"],
      evidence: {},
      steps: {
        handlerReply: nexus.handlerReplyStep,
        timeout: nexus.timeoutStep,
        // @ts-expect-error TS2353: 'awaitFinish' does not exist in type 'StepsOf<...>'
        awaitFinish: nexus.timeoutStep,
      },
    });
  });

  it("rejects a Scenario whose actions or start belong to another Model", () => {
    scenario({
      name: "wrongAction",
      model: nexus.nexusProtocol,
      starts: "unscheduled",
      // @ts-expect-error TS2322: '"attemptStart"' is an action of activityProtocol, not nexusProtocol
      actions: ["attemptStart"],
    });
    scenario({
      name: "wrongPhase",
      model: nexus.nexusProtocol,
      // @ts-expect-error TS2322: '"polling"' is a phase of the worker, not of nexusProtocol
      starts: "polling",
      actions: [],
    });
  });
});
