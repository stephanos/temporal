// authoring: header
import {
  action,
  assertNever,
  compose,
  defineMachine,
  domain,
  entity,
  enumOf,
  limits,
  member,
  observation,
  property,
  query,
  restrict,
  same,
  saturatingSucc,
  scenario,
  set,
  struct,
  upTo,
  type Inputs,
  type Member,
  type StateOf,
  type Step,
} from "./umpire";
import { polling, workerStop } from "./worker";

/**
 * # The Nexus caller-side Model
 *
 * One workflow-scheduled Nexus operation, as the caller sees it: the product machine says what an
 * operation does, the protocol machine says how the server gets there and refines it, and the
 * functional set runs one Query per side effect that settles the operation, once per value of the
 * implementation switch. Step functions and predicates rather than rows, no cancellation (fn-79)
 * and no concurrency-limit setup parameter.
 *
 * The regions AUTHORING.md quotes are marked `// authoring: <name>`; a region runs to the next marker.
 *
 * Read from top to bottom: vocabulary → the two machines → what they promise → what the set asks.
 */

// authoring: entities

// An operation is scheduled by a caller workflow, and recorded data names one by its scheduled
// event: every history event of the operation carries that event's id.

export const workflow = entity("workflow");

export const operation = entity("operation", {
  refer: { caller: workflow },
  key: "scheduledEvent",
});

// authoring: domains

// A class is one member of a domain, and a constructor that carries finite fields contributes one
// class per assignment of them: `handlerError(retryable)` is one constructor and two classes, which
// is the granularity an example is written at and what mirrors a protobuf oneof.
//
// Plain enums are `as const` string unions; a constructor with fields is a discriminated union on
// `kind`, which is what `switch` narrows and `assertNever` checks.

export const Timeout = enumOf(["unset", "expires"]);
export type Timeout = Member<typeof Timeout>;

export type Reply =
  | { readonly kind: "syncSuccess" }
  | { readonly kind: "async" }
  | { readonly kind: "operationFailed" }
  | { readonly kind: "operationCanceled" }
  | { readonly kind: "handlerError"; readonly retryable: boolean };

export const Reply = domain<Reply>()([
  { kind: "syncSuccess" },
  { kind: "async" },
  { kind: "operationFailed" },
  { kind: "operationCanceled" },
  { kind: "handlerError", retryable: false },
  { kind: "handlerError", retryable: true },
]);

export const Resolution = enumOf(["succeeded", "failed", "canceled"]);
export type Resolution = Member<typeof Resolution>;

export const Delivery = enumOf(["accepted", "notFound"]);
export type Delivery = Member<typeof Delivery>;

// authoring: actions

// Parties are names the feature declares by using them: `caller`, `handler`, `network`, `worker`.
// The reserved party `system` is the server. A fault is an ordinary action of a declared party, and
// a timer is `system` behavior the machine owns, so neither is a separate kind.

export const schedule = action({
  name: "schedule",
  party: "caller",
  creates: operation,
  schema: "temporal.api.command.v1.ScheduleNexusOperationCommandAttributes",
  input: { scheduleToClose: Timeout, scheduleToStart: Timeout, startToClose: Timeout },
});

export const handlerReply = action({
  name: "handlerReply",
  party: "handler",
  on: operation,
  schema: "temporal.api.nexus.v1.StartOperationResponse | temporal.api.nexus.v1.HandlerError",
  input: { reply: Reply },
  examples: [
    [{ reply: { kind: "handlerError", retryable: false } }, "BadRequest"],
    [{ reply: { kind: "handlerError", retryable: true } }, "Internal"],
  ],
});

/** The Nexus HTTP completion carries no protobuf message, so it declares no schema and its classes are names the realization interprets. */
export const complete = action({
  name: "complete",
  party: "handler",
  on: operation,
  input: { resolution: Resolution },
  results: Delivery,
});

export const transportFault = action({ name: "transportFault", party: "network", on: operation });

// The handler's worker stops polling. An action that names no entity is behavior no entity
// records: the Run records the fault, but nothing recorded names the operation, so the machines
// keep their state and record nothing at it. The declaration is the worker module's, imported
// rather than written twice, so the composition's `sync:` line pairs one action with itself.
export { workerStop };

// authoring: observation

// A retryable attempt failure writes no history event, so the attempt count is read back through
// `DescribeWorkflowExecution`. Every other evidence name resolves against the realization's
// catalog, which is why only a derived observation is declared.

export const pendingAttempts = observation("pendingAttempts", { on: operation, read: "attempts" });

// authoring: product

// What an operation does, with no account of how. Every Property written against it is carried to
// the protocol machine by the refinement declared there.

export const ProductPhase = enumOf(["scheduled", "started", "succeeded", "failed", "canceled", "timedOut"]);
export type ProductPhase = Member<typeof ProductPhase>;

export const ProductState = struct({ phase: ProductPhase });
export type ProductState = Member<typeof ProductState>;

// Outcomes and facts are only ever produced and compared, never enumerated, so they are types
// alone: the string unions the step functions' return types carry.
export type ProductOutcome = "accepted" | "notFound";

export type ProductFact =
  | "nexusOperationScheduled"
  | "nexusOperationStarted"
  | "nexusOperationCompleted"
  | "nexusOperationFailed"
  | "nexusOperationCanceled"
  | "nexusOperationTimedOut";

type ProductStep = Step<ProductState, ProductOutcome, ProductFact>;

function productStep(phase: ProductPhase, recorded: ProductFact): ProductStep[] {
  return [{ outcome: "accepted", state: { phase }, facts: [recorded] }];
}

/** The handler's reply to the server's start request. An operation that has not started yet is the only one a reply can move. */
export function handlerReplyStep(state: ProductState, { reply }: Inputs<typeof handlerReply.input>): ProductStep[] {
  if (state.phase !== "scheduled") return [];
  switch (reply.kind) {
    case "syncSuccess":
      return productStep("succeeded", "nexusOperationCompleted");
    case "async":
      return productStep("started", "nexusOperationStarted");
    case "operationFailed":
      return productStep("failed", "nexusOperationFailed");
    case "operationCanceled":
      return productStep("canceled", "nexusOperationCanceled");
    case "handlerError":
      // A retryable handler error leaves the operation where it is: the product machine does not
      // know about backing off, which is the whole of what the protocol machine adds.
      return reply.retryable ? [] : productStep("failed", "nexusOperationFailed");
    default:
      return assertNever(reply);
  }
}

/** The four phases the product machine ends on. */
export function productTerminal(state: ProductState): boolean {
  return (
    state.phase === "succeeded" || state.phase === "failed" || state.phase === "canceled" || state.phase === "timedOut"
  );
}

/** An asynchronous completion. A completion that arrives after the operation is over is not found, and changes nothing. */
export function completeStep(state: ProductState, { resolution }: Inputs<typeof complete.input>): ProductStep[] {
  if (productTerminal(state)) return [{ outcome: "notFound", state, facts: [] }];
  switch (resolution) {
    case "succeeded":
      return productStep("succeeded", "nexusOperationCompleted");
    case "failed":
      return productStep("failed", "nexusOperationFailed");
    case "canceled":
      return productStep("canceled", "nexusOperationCanceled");
    default:
      return assertNever(resolution);
  }
}

/** A transport fault is an ordinary action of the network. The product machine cannot see one: whether a delivery was retried is the protocol's account of how, not what. */
export function transportFaultStep(_state: ProductState): ProductStep[] {
  return [];
}

/**
 * The handler's worker stopping is a fault the Run records and the operation does not feel. The
 * product machine cannot see it, like the transport fault: a step that kept the state and recorded
 * nothing would be indistinguishable from a stutter, and the refinement would read every stutter
 * as this step.
 */
export function workerStopStep(_state: ProductState): ProductStep[] {
  return [];
}

/** One of the operation's deadlines firing. Which deadline is the protocol's account of how, so the product machine has one timer, and it fires while the operation runs. */
export function timeoutStep(state: ProductState): ProductStep[] {
  if (state.phase === "scheduled" || state.phase === "started") {
    return productStep("timedOut", "nexusOperationTimedOut");
  }
  return [];
}

export const nexusProduct = defineMachine({
  name: "nexusProduct",
  for: operation,
  state: ProductState,
  starts: ["scheduled"],
  ends: ["succeeded", "failed", "canceled", "timedOut"],
  actions: [handlerReply, complete, transportFault, workerStop],
  timers: ["timeout"],
  evidence: {
    nexusOperationStarted: "nexusOperationStarted",
    nexusOperationCompleted: "nexusOperationCompleted",
    nexusOperationFailed: "nexusOperationFailed",
    nexusOperationCanceled: "nexusOperationCanceled",
    nexusOperationTimedOut: "nexusOperationTimedOut",
  },
  steps: {
    handlerReply: handlerReplyStep,
    complete: completeStep,
    transportFault: transportFaultStep,
    workerStop: workerStopStep,
    timeout: timeoutStep,
  },
});

// authoring: protocol

// How the server gets there: the retry the product machine cannot see, the three timers the
// schedule command sets, and the attempt count a retryable failure raises. Written against the
// same actions, so a Property proved on the product machine is carried here by the refinement.
//
// The machine begins before the operation exists: a state structure has no "no instance yet"
// member, so `unscheduled` is that member, and it is what makes the three deadline fields
// reachable at anything but their first value -- the schedule command is what sets them.
//
// Not here, for reasons recorded rather than silent: the `cancel` field and its rows (fn-79), and
// the concurrency-limit rejection. The limit exists, but a step function does not read the setup,
// the key and value differ per switch value, and the rejection names no operation, so it is not
// modeled until a Query needs it.

export const Phase = enumOf([
  "unscheduled",
  "scheduled",
  "backingOff",
  "started",
  "succeeded",
  "failed",
  "canceled",
  "timedOut",
]);
export type Phase = Member<typeof Phase>;

/** Which timer fired. The history event records it, so a Contract that did not check it would pass a run that timed out on the wrong deadline. */
export const TimeoutType = enumOf(["scheduleToClose", "scheduleToStart", "startToClose"]);
export type TimeoutType = Member<typeof TimeoutType>;

/** The attempt count is bounded by the Limits in the design; nothing wires the Limits into a machine's state, so the bound is written here and the saturating successor keeps a retry inside it. */
export const attemptBound = 2;

export const ProtocolState = struct({
  phase: Phase,
  attempts: upTo(attemptBound),
  scheduleToClose: Timeout,
  scheduleToStart: Timeout,
  startToClose: Timeout,
});
export type ProtocolState = Member<typeof ProtocolState>;

export type ProtocolOutcome = "accepted" | "notFound";

// A fact with a field is a template-literal string, `nexusOperationTimedOut:scheduleToStart`, so
// `facts.includes` and deep equality need no helper; the evidence line names the prefix.
export type ProtocolFact =
  | "nexusOperationScheduled"
  | "nexusOperationStarted"
  | "nexusOperationCompleted"
  | "nexusOperationFailed"
  | "nexusOperationCanceled"
  | `nexusOperationTimedOut:${TimeoutType}`
  | "pendingAttempts";

type ProtocolStep = Step<ProtocolState, ProtocolOutcome, ProtocolFact>;

/** The four phases the design ends on. A completion that arrives after one of them is not found. */
export function terminalPhase(phase: Phase): boolean {
  return phase === "succeeded" || phase === "failed" || phase === "canceled" || phase === "timedOut";
}

/** Scheduled and not yet over: the phases a completion resolves and a timer can fire in. */
export function running(phase: Phase): boolean {
  return phase === "scheduled" || phase === "backingOff" || phase === "started";
}

function moves(state: ProtocolState, phase: Phase, recorded: readonly ProtocolFact[]): ProtocolStep[] {
  return [{ outcome: "accepted", state: { ...state, phase }, facts: recorded }];
}

/**
 * The caller's schedule command. It names the operation's three deadlines, and every one of them
 * is a state field because whether a timer fires is a question about the operation and not about
 * the command that started it.
 */
export function scheduleStep(
  state: ProtocolState,
  { scheduleToClose, scheduleToStart, startToClose }: Inputs<typeof schedule.input>,
): ProtocolStep[] {
  if (state.phase !== "unscheduled") return [];
  return [
    {
      outcome: "accepted",
      state: { phase: "scheduled", attempts: 0, scheduleToClose, scheduleToStart, startToClose },
      facts: ["nexusOperationScheduled"],
    },
  ];
}

/**
 * The handler's reply to the server's start request. What the product machine cannot see is the
 * last arm: a retryable failure backs the operation off and raises its attempt count, and the count
 * is read back through the `pendingAttempts` observation because no history event records it.
 */
export function protocolHandlerReplyStep(
  state: ProtocolState,
  { reply }: Inputs<typeof handlerReply.input>,
): ProtocolStep[] {
  if (state.phase !== "scheduled") return [];
  switch (reply.kind) {
    case "syncSuccess":
      return moves(state, "succeeded", ["nexusOperationCompleted"]);
    case "async":
      return moves(state, "started", ["nexusOperationStarted"]);
    case "operationFailed":
      return moves(state, "failed", ["nexusOperationFailed"]);
    case "operationCanceled":
      return moves(state, "canceled", ["nexusOperationCanceled"]);
    case "handlerError":
      if (!reply.retryable) return moves(state, "failed", ["nexusOperationFailed"]);
      return [
        {
          outcome: "accepted",
          state: { ...state, phase: "backingOff", attempts: saturatingSucc(state.attempts, attemptBound) },
          facts: ["pendingAttempts"],
        },
      ];
    default:
      return assertNever(reply);
  }
}

/** A transport fault is the same failure arriving as a dropped delivery rather than as a reply. */
export function protocolTransportFaultStep(state: ProtocolState): ProtocolStep[] {
  if (state.phase !== "scheduled") return [];
  return [
    {
      outcome: "accepted",
      state: { ...state, phase: "backingOff", attempts: saturatingSucc(state.attempts, attemptBound) },
      facts: ["pendingAttempts"],
    },
  ];
}

/**
 * The handler's worker stopping is a fault the Run records and the operation does not feel, so the
 * step keeps the state and records nothing. On a path it is confirmed by the evidence of the step
 * after it, and the Case says so in a Known Gap.
 */
export function protocolWorkerStopStep(state: ProtocolState): ProtocolStep[] {
  return [{ outcome: "accepted", state, facts: [] }];
}

/**
 * An asynchronous completion. Before a start, the server records a Started event first, which is
 * why the evidence is two facts and not one -- and why the product machine, which has no
 * `backingOff` phase to have skipped, could write the completion alone.
 */
export function protocolCompleteStep(
  state: ProtocolState,
  { resolution }: Inputs<typeof complete.input>,
): ProtocolStep[] {
  if (terminalPhase(state.phase)) return [{ outcome: "notFound", state, facts: [] }];
  if (state.phase === "unscheduled") return [];
  const startedFirst: ProtocolFact[] = state.phase === "started" ? [] : ["nexusOperationStarted"];
  switch (resolution) {
    case "succeeded":
      return moves(state, "succeeded", [...startedFirst, "nexusOperationCompleted"]);
    case "failed":
      return moves(state, "failed", [...startedFirst, "nexusOperationFailed"]);
    case "canceled":
      return moves(state, "canceled", [...startedFirst, "nexusOperationCanceled"]);
    default:
      return assertNever(resolution);
  }
}

/** The backoff timer. It is what makes `backingOff` a phase the operation leaves rather than a state it is stuck in, and it records nothing: a retry writes no history event. */
export function backoffStep(state: ProtocolState): ProtocolStep[] {
  if (state.phase !== "backingOff") return [];
  return moves(state, "scheduled", []);
}

/** The schedule-to-close deadline covers the whole operation, so it fires in every running phase -- and only when the schedule command set it. */
export function scheduleToCloseStep(state: ProtocolState): ProtocolStep[] {
  if (running(state.phase) && state.scheduleToClose === "expires") {
    return moves(state, "timedOut", ["nexusOperationTimedOut:scheduleToClose"]);
  }
  return [];
}

/** The schedule-to-start deadline covers the wait for the handler to accept, so it stops at the start. */
export function scheduleToStartStep(state: ProtocolState): ProtocolStep[] {
  if ((state.phase === "scheduled" || state.phase === "backingOff") && state.scheduleToStart === "expires") {
    return moves(state, "timedOut", ["nexusOperationTimedOut:scheduleToStart"]);
  }
  return [];
}

/** The start-to-close deadline covers the handler's own work, so it begins at the start. */
export function startToCloseStep(state: ProtocolState): ProtocolStep[] {
  if (state.phase === "started" && state.startToClose === "expires") {
    return moves(state, "timedOut", ["nexusOperationTimedOut:startToClose"]);
  }
  return [];
}

/**
 * How a protocol state reads as a product state. A phase of the same name is that phase; backing
 * off is still scheduled, because the product machine cannot see a retry; and an operation not yet
 * scheduled reads as scheduled, because the product machine begins there. Every other field is
 * hidden, which is what a map that does not read it says.
 */
export function productOf(state: ProtocolState): ProductState {
  switch (state.phase) {
    case "unscheduled":
    case "scheduled":
    case "backingOff":
      return { phase: "scheduled" };
    case "started":
      return { phase: "started" };
    case "succeeded":
      return { phase: "succeeded" };
    case "failed":
      return { phase: "failed" };
    case "canceled":
      return { phase: "canceled" };
    case "timedOut":
      return { phase: "timedOut" };
    default:
      return assertNever(state.phase);
  }
}

export const nexusProtocol = defineMachine({
  name: "nexusProtocol",
  for: operation,
  state: ProtocolState,
  refines: { machine: nexusProduct, map: productOf },
  starts: ["unscheduled"],
  ends: ["succeeded", "failed", "canceled", "timedOut"],
  actions: [schedule, handlerReply, complete, transportFault, workerStop],
  timers: ["backoff", "scheduleToClose", "scheduleToStart", "startToClose"],
  unobservable: ["backoff"],
  evidence: {
    nexusOperationScheduled: "nexusOperationScheduled",
    nexusOperationStarted: "nexusOperationStarted",
    nexusOperationCompleted: "nexusOperationCompleted",
    nexusOperationFailed: "nexusOperationFailed",
    nexusOperationCanceled: "nexusOperationCanceled",
    nexusOperationTimedOut: "nexusOperationTimedOut",
    pendingAttempts: "pendingAttempts",
  },
  steps: {
    schedule: scheduleStep,
    handlerReply: protocolHandlerReplyStep,
    complete: protocolCompleteStep,
    transportFault: protocolTransportFaultStep,
    workerStop: protocolWorkerStopStep,
    backoff: backoffStep,
    scheduleToClose: scheduleToCloseStep,
    scheduleToStart: scheduleToStartStep,
    startToClose: startToCloseStep,
  },
});

// authoring: properties

// A same-step claim names the action it is about under `when:` and holds of the step that action
// produces; a transition claim holds of the step before and the step after. A functional Query
// realizes a same-step claim, because the Case's Contract is the claim's clause triggered by the
// action the Case performs; a transition claim is searched and verified, never realized.

/** Once an operation is over, no step changes its phase. Declared on the product machine and read on the protocol machine through the map. */
export const terminalIsFinal = property({
  name: "terminalIsFinal",
  machine: nexusProduct,
  holds: (before, after) => !productTerminal(before.state) || after.state.phase === before.state.phase,
});

/** A synchronous reply settles the operation as succeeded, and the completed event records it. */
export const syncSucceeds = property({
  name: "syncSucceeds",
  machine: nexusProtocol,
  when: handlerReply.of({ reply: { kind: "syncSuccess" } }),
  holds: (step) => step.state.phase === "succeeded" && step.facts.includes("nexusOperationCompleted"),
});

/** An asynchronous reply starts the operation, and the started event records it. */
export const asyncStarts = property({
  name: "asyncStarts",
  machine: nexusProtocol,
  when: handlerReply.of({ reply: { kind: "async" } }),
  holds: (step) => step.state.phase === "started" && step.facts.includes("nexusOperationStarted"),
});

/**
 * A successful completion is recorded by the completed event. Neither the phase nor the outcome is
 * fixed: a completion resolves any running phase, and `accepted` is every earlier step's outcome
 * too, so a clause fixing it would be answered before the completion.
 */
export const completionSucceeds = property({
  name: "completionSucceeds",
  machine: nexusProtocol,
  when: complete.of({ resolution: "succeeded" }),
  holds: (step) => step.facts.includes("nexusOperationCompleted"),
});

/** A failed completion is recorded by the failed event. */
export const completionFails = property({
  name: "completionFails",
  machine: nexusProtocol,
  when: complete.of({ resolution: "failed" }),
  holds: (step) => step.facts.includes("nexusOperationFailed"),
});

/** A non-retryable handler error settles the operation as failed, and the failed event records it. */
export const handlerErrorFails = property({
  name: "handlerErrorFails",
  machine: nexusProtocol,
  when: handlerReply.of({ reply: { kind: "handlerError", retryable: false } }),
  holds: (step) => step.state.phase === "failed" && step.facts.includes("nexusOperationFailed"),
});

/** Succeeded on the second attempt of an operation with no deadline set. A claim fixes one state, so every field is named. */
export const succeededOnRetry = {
  phase: "succeeded",
  attempts: 1,
  scheduleToClose: "unset",
  scheduleToStart: "unset",
  startToClose: "unset",
} satisfies ProtocolState;

/**
 * A synchronous reply to the retried attempt settles the operation as succeeded on its second
 * attempt: the count the retryable failure raised is still one, and the completed event records
 * the reply.
 */
export const retrySucceeds = property({
  name: "retrySucceeds",
  machine: nexusProtocol,
  when: handlerReply.of({ reply: { kind: "syncSuccess" } }),
  holds: (step) => same(ProtocolState, step.state, succeededOnRetry) && step.facts.includes("nexusOperationCompleted"),
});

/** The schedule-to-start deadline settles an operation no handler started as timed out, and the timed-out event records which deadline it was. */
export const scheduleToStartFires = property({
  name: "scheduleToStartFires",
  machine: nexusProtocol,
  when: "scheduleToStart",
  holds: (step) => step.state.phase === "timedOut" && step.facts.includes("nexusOperationTimedOut:scheduleToStart"),
});

/** The start-to-close deadline settles a started operation no handler completed as timed out. */
export const startToCloseFires = property({
  name: "startToCloseFires",
  machine: nexusProtocol,
  when: "startToClose",
  holds: (step) => step.state.phase === "timedOut" && step.facts.includes("nexusOperationTimedOut:startToClose"),
});

// authoring: scenarios

// A protocol Scenario names its classed actions with their inputs and its start by its phase.
// Each path below is one upstream functional test's shape: the schedule command with no deadline
// set, then the side effects that settle the operation.

const noDeadline = schedule.of({ scheduleToClose: "unset", scheduleToStart: "unset", startToClose: "unset" });

export const syncReplied = scenario({
  name: "syncReplied",
  model: nexusProtocol,
  starts: "unscheduled",
  actions: [noDeadline, handlerReply.of({ reply: { kind: "syncSuccess" } })],
});

export const asyncThenSucceeded = scenario({
  name: "asyncThenSucceeded",
  model: nexusProtocol,
  starts: "unscheduled",
  actions: [noDeadline, handlerReply.of({ reply: { kind: "async" } }), complete.of({ resolution: "succeeded" })],
});

export const asyncThenFailed = scenario({
  name: "asyncThenFailed",
  model: nexusProtocol,
  starts: "unscheduled",
  actions: [noDeadline, handlerReply.of({ reply: { kind: "async" } }), complete.of({ resolution: "failed" })],
});

export const nonRetryableError = scenario({
  name: "nonRetryableError",
  model: nexusProtocol,
  starts: "unscheduled",
  actions: [noDeadline, handlerReply.of({ reply: { kind: "handlerError", retryable: false } })],
});

/** The retryable error backs the operation off; the backoff timer fires and records nothing; the retried attempt is answered synchronously. */
export const retriedThenSucceeded = scenario({
  name: "retriedThenSucceeded",
  model: nexusProtocol,
  starts: "unscheduled",
  actions: [
    noDeadline,
    handlerReply.of({ reply: { kind: "handlerError", retryable: true } }),
    "backoff",
    handlerReply.of({ reply: { kind: "syncSuccess" } }),
  ],
});

/**
 * The schedule command sets the schedule-to-start deadline; the handler's worker stops, so nothing
 * answers the start request; the deadline fires. The worker stops after the schedule in the
 * operation's order, where the stop changes nothing; the realization stops it before the workflow
 * starts, where the stop cannot race the dispatch.
 */
export const scheduleToStartExpires = scenario({
  name: "scheduleToStartExpires",
  model: nexusProtocol,
  starts: "unscheduled",
  actions: [
    schedule.of({ scheduleToClose: "unset", scheduleToStart: "expires", startToClose: "unset" }),
    "workerStop",
    "scheduleToStart",
  ],
});

/** The schedule command sets the start-to-close deadline; the handler accepts asynchronously and never completes; the deadline fires. */
export const startToCloseExpires = scenario({
  name: "startToCloseExpires",
  model: nexusProtocol,
  starts: "unscheduled",
  actions: [
    schedule.of({ scheduleToClose: "unset", scheduleToStart: "unset", startToClose: "expires" }),
    handlerReply.of({ reply: { kind: "async" } }),
    "startToClose",
  ],
});

// Nine actions are enabled before the operation is scheduled and eleven once it is, so an exact
// sequence of two is found among ninety-nine candidates, one of three among about a thousand and
// one of four among about ten thousand.

export const two = limits({ name: "two", steps: 2, actions: 2, search: 512 });
export const three = limits({ name: "three", steps: 3, actions: 3, search: 4096 });
export const four = limits({ name: "four", steps: 4, actions: 4, search: 32768 });

// authoring: queries

// The design's seven: sync success, async reply then succeeded callback, async reply then failed
// callback, non-retryable handler error, retryable handler error then sync success after one
// backoff, schedule-to-start timeout with the handler's worker stopped, start-to-close timeout
// after an asynchronous reply. Each finds its same-step claim on its path and is realized by the
// set below. The product claim is verified over every trace of one path, outside the set, because
// a `verify` Query realizes nothing.

export const syncCompletion = query({ name: "syncCompletion", find: syncSucceeds, in: syncReplied, limits: two });
export const asyncCompletion = query({
  name: "asyncCompletion",
  find: completionSucceeds,
  in: asyncThenSucceeded,
  limits: three,
});
export const asyncFailure = query({ name: "asyncFailure", find: completionFails, in: asyncThenFailed, limits: three });
export const handlerError = query({ name: "handlerError", find: handlerErrorFails, in: nonRetryableError, limits: two });
export const retry = query({ name: "retry", find: retrySucceeds, in: retriedThenSucceeded, limits: four });
export const scheduleToStartTimeout = query({
  name: "scheduleToStartTimeout",
  find: scheduleToStartFires,
  in: scheduleToStartExpires,
  limits: three,
});
export const startToCloseTimeout = query({
  name: "startToCloseTimeout",
  find: startToCloseFires,
  in: startToCloseExpires,
  limits: three,
});
export const terminalHolds = query({ name: "terminalHolds", verify: terminalIsFinal, in: asyncThenSucceeded, limits: three });

// authoring: set

// Every party but `system` is bound: the Case drives the caller, the handler and the worker, and
// observes the network. The set repeats over the implementation switch, so each Query's Case runs
// once under HSM and once under CHASM.

export const nexusCallerTests = set({
  name: "nexusCallerTests",
  purpose: "functional",
  bind: { caller: "driven", handler: "driven", network: "observed", worker: "driven" },
  repeat: "implementation",
  queries: [
    syncCompletion,
    asyncCompletion,
    asyncFailure,
    handlerError,
    retry,
    scheduleToStartTimeout,
    startToCloseTimeout,
  ],
});

// A canary runs a Query against a deployment that performs the handler's part itself: the handler
// is `observed`, so the verifier reads which reply occurred and checks the machine allows it. What
// admits a canary is that a deployment can close every gap its Case carries, and every step of the
// sync and async completion paths records evidence; a path with a silent step -- the backoff, the
// worker stop -- is a capability gap no deployment closes, so a canary naming it is rejected.

export const nexusCallerCanary = set({
  name: "nexusCallerCanary",
  purpose: "canary",
  bind: { caller: "driven", handler: "observed", network: "observed", worker: "driven" },
  queries: [syncCompletion, asyncCompletion],
});

// An exploration covers the protocol machine rather than listing Queries. Its targets are the
// rows an exploration within the budget's steps of a start can take, the results those rows reach
// and the members of the classes their actions claim, each in the machine's catalog order and cut
// at the budget's search count, so the enumeration is the same on every reading.

export const nexusCallerExploration = set({
  name: "nexusCallerExploration",
  purpose: "exploratory",
  bind: { caller: "driven", handler: "driven", network: "observed", worker: "driven" },
  machine: nexusProtocol,
  cover: ["rows", "results", "classMembers"],
  budget: four,
});

// authoring: case

// The Cases (`case nexusCallerCases realizes nexusCallerTests as ...`) are the realization layer,
// out of scope for this comparison (SPEC.md).

// authoring: composition

// The protocol machine's worker stop is a stutter row: the operation cannot see its handler's
// worker, so the schedule-to-start Scenario orders the stop before the request by convention.
// Composed with the worker of the handler's task queue, the stop is the worker's own phase change
// and every reply is the worker serving, so a reply has a row only while the worker polls. No set
// names the composition; it is what the cross-entity claim is verified over.

/**
 * The caller's view of the handler's worker: it stops and it serves. It never resumes, because an
 * action no `sync:` line names would stay executable on its own and admit a stop, a resume and
 * then a reply; the operation's timers settle every state a stop leaves.
 */
export const handlerWorker = restrict(polling, ["workerStop", "serve"]);

export const nexusCaller = compose({
  name: "nexusCaller",
  members: { operation: nexusProtocol, worker: handlerWorker },
  sync: {
    workerStop: ["operation.workerStop", "worker.workerStop"],
    handlerReply: ["operation.handlerReply", "worker.serve"],
  },
  starts: ["operation.unscheduled", "worker.polling"],
  ends: ["operation.succeeded", "operation.failed", "operation.canceled", "operation.timedOut"],
});

/** `{ operation: ProtocolState; worker: WorkerState }`, derived from the members rather than declared. */
export type NexusCallerState = StateOf<typeof nexusCaller>;

/** Every reply, of any class, leaves the handler's worker polling: no handler replies while its worker is stopped. */
export const repliedByPollingWorker = property({
  name: "repliedByPollingWorker",
  machine: nexusCaller,
  when: "handlerReply",
  holds: (step) => step.state.worker.phase === "polling",
});

/** A retryable reply backs the operation off; the handler's worker then stops, so the retried attempt is never answered and the schedule-to-start deadline fires. */
export const repliedThenStopped = scenario({
  name: "repliedThenStopped",
  model: nexusCaller,
  starts: "operation.unscheduled",
  actions: [
    member("operation", schedule.of({ scheduleToClose: "unset", scheduleToStart: "expires", startToClose: "unset" })),
    handlerReply.of({ reply: { kind: "handlerError", retryable: true } }),
    "workerStop",
    "operation.scheduleToStart",
  ],
});

export const stoppedWorkerRepliesNothing = query({
  name: "stoppedWorkerRepliesNothing",
  verify: repliedByPollingWorker,
  in: repliedThenStopped,
  limits: four,
});

// authoring: end
