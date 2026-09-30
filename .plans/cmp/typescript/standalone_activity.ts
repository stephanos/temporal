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
import { Timeout } from "./nexus_caller";
import { polling, workerStop } from "./worker";

/**
 * # The standalone activity Model
 *
 * One activity started directly through `StartActivityExecution`, with no workflow around it: the
 * product machine says what an activity does as `DescribeActivityExecution` reports it, the
 * protocol machine says how the server gets there and refines it, and the functional set runs one
 * Query per side effect that settles the activity. Grounded in `chasm/lib/activity/statemachine.go`;
 * reset is deferred, like cancellation in the Nexus caller Model, and the heartbeat timeout is not
 * modeled.
 *
 * A standalone activity writes no history event. Every fact below is a status read through
 * `DescribeActivityExecution`, or a result read through `PollActivityExecution`, so the evidence
 * lines of the machines name observations rather than events.
 *
 * Read from top to bottom: vocabulary → the two machines → what they promise → what the set asks.
 */

// authoring: entities

// An activity is named by the id the caller chose for it: every status read and every result read
// carries it, and no run id or event id is needed to tell two apart.

export const activity = entity("activity", { key: "activityId" });

// authoring: domains

// As in the caller Model, a constructor with a finite field contributes one class per assignment:
// `failed(retryable)` is two classes, which mirrors the retryable flag of an `ApplicationFailure`.
// `Timeout` is the caller Model's, reused.

export { Timeout };

export type AttemptResult =
  | { readonly kind: "completed" }
  | { readonly kind: "failed"; readonly retryable: boolean }
  | { readonly kind: "canceled" };

export const AttemptResult = domain<AttemptResult>()([
  { kind: "completed" },
  { kind: "failed", retryable: false },
  { kind: "failed", retryable: true },
  { kind: "canceled" },
]);

export const Delivery = enumOf(["accepted", "notFound"]);
export type Delivery = Member<typeof Delivery>;

export const Control = enumOf(["pause", "unpause", "requestCancel", "terminate"]);
export type Control = Member<typeof Control>;

// authoring: actions

// Parties: `caller` starts and controls the activity, `worker` runs its attempts, `system` owns
// the timers. The worker's stop is an ordinary action of the `worker` party, as in every other Model.

export const start = action({
  name: "start",
  party: "caller",
  creates: activity,
  schema: "temporal.api.workflowservice.v1.StartActivityExecutionRequest",
  input: { scheduleToClose: Timeout, scheduleToStart: Timeout, startToClose: Timeout },
});

/** The worker's poll receives the task for the current attempt. */
export const attemptStart = action({
  name: "attemptStart",
  party: "worker",
  on: activity,
  schema: "temporal.api.workflowservice.v1.PollActivityTaskQueueResponse",
});

export const attemptResult = action({
  name: "attemptResult",
  party: "worker",
  on: activity,
  schema:
    "temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest | " +
    "temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest | " +
    "temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest",
  input: { result: AttemptResult },
  examples: [
    [{ result: { kind: "failed", retryable: false } }, "ApplicationFailureNonRetryable"],
    [{ result: { kind: "failed", retryable: true } }, "ApplicationFailureRetryable"],
  ],
});

/** The four caller-side controls are one action with a finite input, because they share a result: a control on an activity that is over is not found. */
export const control = action({
  name: "control",
  party: "caller",
  on: activity,
  schema:
    "temporal.api.workflowservice.v1.PauseActivityExecutionRequest | " +
    "temporal.api.workflowservice.v1.UnpauseActivityExecutionRequest | " +
    "temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest | " +
    "temporal.api.workflowservice.v1.TerminateActivityExecutionRequest",
  input: { control: Control },
  results: Delivery,
});

// The worker stops polling. Nothing recorded names the activity, so the machines keep their state
// and record nothing at it. The declaration is the worker module's.
export { workerStop };

// authoring: observation

// A retried attempt writes nothing the caller can see except the attempt count that
// `DescribeActivityExecution` reports, so it is the one derived observation.

export const attemptCount = observation("attemptCount", { on: activity, read: "attempt" });

// authoring: product

// What the caller sees through `DescribeActivityExecution`, with no account of how. A pause is one
// phase here whether or not a worker still holds the attempt. Unlike Nexus, the retry is visible:
// a retryable failure reads as SCHEDULED again (`TransitionRescheduled`), with a higher attempt count.

export const ProductPhase = enumOf([
  "scheduled",
  "started",
  "paused",
  "cancelRequested",
  "completed",
  "failed",
  "canceled",
  "terminated",
  "timedOut",
]);
export type ProductPhase = Member<typeof ProductPhase>;

export const ProductState = struct({ phase: ProductPhase });
export type ProductState = Member<typeof ProductState>;

export type ProductOutcome = "accepted" | "notFound";

export type ProductFact =
  | "statusScheduled"
  | "statusStarted"
  | "statusPaused"
  | "statusCancelRequested"
  | "statusCompleted"
  | "statusFailed"
  | "statusCanceled"
  | "statusTerminated"
  | "statusTimedOut";

type ProductStep = Step<ProductState, ProductOutcome, ProductFact>;

function productStep(phase: ProductPhase, recorded: ProductFact): ProductStep[] {
  return [{ outcome: "accepted", state: { phase }, facts: [recorded] }];
}

/** The phases the product machine ends on. */
export function productTerminal(state: ProductState): boolean {
  return (
    state.phase === "completed" ||
    state.phase === "failed" ||
    state.phase === "canceled" ||
    state.phase === "terminated" ||
    state.phase === "timedOut"
  );
}

/** A worker takes the attempt of a scheduled activity. */
export function attemptStartStep(state: ProductState): ProductStep[] {
  if (state.phase !== "scheduled") return [];
  return productStep("started", "statusStarted");
}

/**
 * The worker's answer to the attempt. A retryable failure is visible, unlike in the Nexus Model:
 * from started the activity reads as scheduled again (`TransitionRescheduled`), and under a cancel
 * request it settles as canceled. What the product does not see is the backoff in between, which
 * is what the protocol machine adds. A canceled answer settles only an activity whose
 * cancellation was requested.
 */
export function attemptResultStep(state: ProductState, { result }: Inputs<typeof attemptResult.input>): ProductStep[] {
  if (state.phase !== "started" && state.phase !== "cancelRequested") return [];
  switch (result.kind) {
    case "completed":
      return productStep("completed", "statusCompleted");
    case "failed":
      if (!result.retryable) return productStep("failed", "statusFailed");
      return state.phase === "cancelRequested"
        ? productStep("canceled", "statusCanceled")
        : productStep("scheduled", "statusScheduled");
    case "canceled":
      return state.phase === "cancelRequested" ? productStep("canceled", "statusCanceled") : [];
    default:
      return assertNever(result);
  }
}

/** A control on an activity that is over is not found and changes nothing. */
export function controlStep(state: ProductState, { control }: Inputs<typeof control.input>): ProductStep[] {
  if (productTerminal(state)) return [{ outcome: "notFound", state, facts: [] }];
  switch (control) {
    case "pause":
      return state.phase === "scheduled" || state.phase === "started" ? productStep("paused", "statusPaused") : [];
    case "unpause":
      return state.phase === "paused" ? productStep("scheduled", "statusScheduled") : [];
    case "requestCancel":
      return state.phase === "scheduled" ||
        state.phase === "started" ||
        state.phase === "paused" ||
        state.phase === "cancelRequested"
        ? productStep("cancelRequested", "statusCancelRequested")
        : [];
    case "terminate":
      return productStep("terminated", "statusTerminated");
    default:
      return assertNever(control);
  }
}

/** The worker stopping is a fault the Run records and the activity does not feel. */
export function workerStopStep(_state: ProductState): ProductStep[] {
  return [];
}

/** One of the activity's deadlines firing. Which deadline is the protocol's account of how. */
export function timeoutStep(state: ProductState): ProductStep[] {
  if (
    state.phase === "scheduled" ||
    state.phase === "started" ||
    state.phase === "cancelRequested" ||
    state.phase === "paused"
  ) {
    return productStep("timedOut", "statusTimedOut");
  }
  return [];
}

export const activityProduct = defineMachine({
  name: "activityProduct",
  for: activity,
  state: ProductState,
  starts: ["scheduled"],
  ends: ["completed", "failed", "canceled", "terminated", "timedOut"],
  actions: [attemptStart, attemptResult, control, workerStop],
  timers: ["timeout"],
  evidence: {
    statusScheduled: "statusScheduled",
    statusStarted: "statusStarted",
    statusPaused: "statusPaused",
    statusCancelRequested: "statusCancelRequested",
    statusCompleted: "statusCompleted",
    statusFailed: "statusFailed",
    statusCanceled: "statusCanceled",
    statusTerminated: "statusTerminated",
    statusTimedOut: "statusTimedOut",
  },
  steps: {
    attemptStart: attemptStartStep,
    attemptResult: attemptResultStep,
    control: controlStep,
    workerStop: workerStopStep,
    timeout: timeoutStep,
  },
});

// authoring: protocol

// How the server gets there: the retry the product machine cannot see, the pause a running attempt
// turns into a pause request, the three timers the start request sets, and the attempt count. The
// machine begins before the activity exists, so `unstarted` is a phase and the start request is
// what sets the deadlines.

export const Phase = enumOf([
  "unstarted",
  "scheduled",
  "backingOff",
  "started",
  "paused",
  "pauseRequested",
  "cancelRequested",
  "completed",
  "failed",
  "canceled",
  "terminated",
  "timedOut",
]);
export type Phase = Member<typeof Phase>;

export const TimeoutType = enumOf(["scheduleToClose", "scheduleToStart", "startToClose"]);
export type TimeoutType = Member<typeof TimeoutType>;

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

export type ProtocolFact =
  | "statusScheduled"
  | "statusStarted"
  | "statusPaused"
  | "statusCancelRequested"
  | "statusCompleted"
  | "statusFailed"
  | "statusCanceled"
  | "statusTerminated"
  | `statusTimedOut:${TimeoutType}`
  | "attemptCount";

type ProtocolStep = Step<ProtocolState, ProtocolOutcome, ProtocolFact>;

export function terminalPhase(phase: Phase): boolean {
  return (
    phase === "completed" || phase === "failed" || phase === "canceled" || phase === "terminated" || phase === "timedOut"
  );
}

/** Started and not over: the phases a deadline can fire in. */
export function running(phase: Phase): boolean {
  return (
    phase === "scheduled" ||
    phase === "backingOff" ||
    phase === "started" ||
    phase === "paused" ||
    phase === "pauseRequested" ||
    phase === "cancelRequested"
  );
}

/** A worker holds the attempt: the phases a start-to-close deadline covers and a worker's answer settles. */
export function attemptHeld(phase: Phase): boolean {
  return phase === "started" || phase === "pauseRequested" || phase === "cancelRequested";
}

function moves(state: ProtocolState, phase: Phase, recorded: readonly ProtocolFact[]): ProtocolStep[] {
  return [{ outcome: "accepted", state: { ...state, phase }, facts: recorded }];
}

export function startStep(
  state: ProtocolState,
  { scheduleToClose, scheduleToStart, startToClose }: Inputs<typeof start.input>,
): ProtocolStep[] {
  if (state.phase !== "unstarted") return [];
  return [
    {
      outcome: "accepted",
      state: { phase: "scheduled", attempts: 0, scheduleToClose, scheduleToStart, startToClose },
      facts: ["statusScheduled"],
    },
  ];
}

/** The worker's poll takes the attempt and raises the count the caller reads back. */
export function protocolAttemptStartStep(state: ProtocolState): ProtocolStep[] {
  if (state.phase !== "scheduled") return [];
  return [
    {
      outcome: "accepted",
      state: { ...state, phase: "started", attempts: saturatingSucc(state.attempts, attemptBound) },
      facts: ["statusStarted", "attemptCount"],
    },
  ];
}

/**
 * The worker's answer, by the phase it lands in. A retryable failure backs a started attempt off,
 * settles a cancel-requested one as canceled, and lands a pause-requested one in paused
 * (`TransitionAttemptFailedWhilePauseRequested`). A canceled answer is honored only under a cancel
 * request.
 */
export function protocolAttemptResultStep(
  state: ProtocolState,
  { result }: Inputs<typeof attemptResult.input>,
): ProtocolStep[] {
  if (!attemptHeld(state.phase)) return [];
  switch (result.kind) {
    case "completed":
      return moves(state, "completed", ["statusCompleted"]);
    case "failed":
      if (!result.retryable) return moves(state, "failed", ["statusFailed"]);
      switch (state.phase) {
        case "cancelRequested":
          return moves(state, "canceled", ["statusCanceled"]);
        case "pauseRequested":
          return moves(state, "paused", ["statusPaused"]);
        default:
          return moves(state, "backingOff", ["attemptCount"]);
      }
    case "canceled":
      return state.phase === "cancelRequested" ? moves(state, "canceled", ["statusCanceled"]) : [];
    default:
      return assertNever(result);
  }
}

/** The caller's controls. A pause of a held attempt is a request the worker learns of on its next heartbeat, so it is its own phase; the caller reads it as paused either way. */
export function protocolControlStep(state: ProtocolState, { control }: Inputs<typeof control.input>): ProtocolStep[] {
  if (terminalPhase(state.phase)) return [{ outcome: "notFound", state, facts: [] }];
  if (state.phase === "unstarted") return [];
  switch (control) {
    case "pause":
      if (state.phase === "scheduled" || state.phase === "backingOff") return moves(state, "paused", ["statusPaused"]);
      if (state.phase === "started") return moves(state, "pauseRequested", ["statusPaused"]);
      return [];
    case "unpause":
      if (state.phase === "paused") return moves(state, "scheduled", ["statusScheduled"]);
      if (state.phase === "pauseRequested") return moves(state, "started", ["statusStarted"]);
      return [];
    case "requestCancel":
      return moves(state, "cancelRequested", ["statusCancelRequested"]);
    case "terminate":
      return moves(state, "terminated", ["statusTerminated"]);
    default:
      return assertNever(control);
  }
}

/** The worker stopping keeps the state and records nothing; on a path it is confirmed by the evidence of the step after it, and the Case says so in a Known Gap. */
export function protocolWorkerStopStep(state: ProtocolState): ProtocolStep[] {
  return [{ outcome: "accepted", state, facts: [] }];
}

/** The backoff timer: a retry writes nothing the caller can read. */
export function backoffStep(state: ProtocolState): ProtocolStep[] {
  if (state.phase !== "backingOff") return [];
  return moves(state, "scheduled", []);
}

export function scheduleToCloseStep(state: ProtocolState): ProtocolStep[] {
  if (running(state.phase) && state.scheduleToClose === "expires") {
    return moves(state, "timedOut", ["statusTimedOut:scheduleToClose"]);
  }
  return [];
}

export function scheduleToStartStep(state: ProtocolState): ProtocolStep[] {
  if ((state.phase === "scheduled" || state.phase === "backingOff") && state.scheduleToStart === "expires") {
    return moves(state, "timedOut", ["statusTimedOut:scheduleToStart"]);
  }
  return [];
}

export function startToCloseStep(state: ProtocolState): ProtocolStep[] {
  if (attemptHeld(state.phase) && state.startToClose === "expires") {
    return moves(state, "timedOut", ["statusTimedOut:startToClose"]);
  }
  return [];
}

/**
 * How a protocol state reads as a product state: not yet started and backing off read as
 * scheduled; a pause request reads as started, because the worker still holds the attempt, so
 * every answer it can give is a product row from started and the request itself is a stutter;
 * paused is paused; every other phase is its namesake.
 */
export function productOf(state: ProtocolState): ProductState {
  switch (state.phase) {
    case "unstarted":
    case "scheduled":
    case "backingOff":
      return { phase: "scheduled" };
    case "started":
    case "pauseRequested":
      return { phase: "started" };
    case "paused":
      return { phase: "paused" };
    case "cancelRequested":
      return { phase: "cancelRequested" };
    case "completed":
      return { phase: "completed" };
    case "failed":
      return { phase: "failed" };
    case "canceled":
      return { phase: "canceled" };
    case "terminated":
      return { phase: "terminated" };
    case "timedOut":
      return { phase: "timedOut" };
    default:
      return assertNever(state.phase);
  }
}

export const activityProtocol = defineMachine({
  name: "activityProtocol",
  for: activity,
  state: ProtocolState,
  refines: { machine: activityProduct, map: productOf },
  starts: ["unstarted"],
  ends: ["completed", "failed", "canceled", "terminated", "timedOut"],
  actions: [start, attemptStart, attemptResult, control, workerStop],
  timers: ["backoff", "scheduleToClose", "scheduleToStart", "startToClose"],
  unobservable: ["backoff"],
  evidence: {
    statusScheduled: "statusScheduled",
    statusStarted: "statusStarted",
    statusPaused: "statusPaused",
    statusCancelRequested: "statusCancelRequested",
    statusCompleted: "statusCompleted",
    statusFailed: "statusFailed",
    statusCanceled: "statusCanceled",
    statusTerminated: "statusTerminated",
    statusTimedOut: "statusTimedOut",
    attemptCount: "attemptCount",
  },
  steps: {
    start: startStep,
    attemptStart: protocolAttemptStartStep,
    attemptResult: protocolAttemptResultStep,
    control: protocolControlStep,
    workerStop: protocolWorkerStopStep,
    backoff: backoffStep,
    scheduleToClose: scheduleToCloseStep,
    scheduleToStart: scheduleToStartStep,
    startToClose: startToCloseStep,
  },
});

// authoring: properties

// What the machines promise.

export const terminalIsFinal = property({
  name: "terminalIsFinal",
  machine: activityProduct,
  holds: (before, after) => !productTerminal(before.state) || after.state.phase === before.state.phase,
});

/** A paused activity is dispatched to no worker: nothing moves it straight to started. */
export const pausedIsNotDispatched = property({
  name: "pausedIsNotDispatched",
  machine: activityProduct,
  holds: (before, after) => before.state.phase !== "paused" || after.state.phase !== "started",
});

export const completes = property({
  name: "completes",
  machine: activityProtocol,
  when: attemptResult.of({ result: { kind: "completed" } }),
  holds: (step) => step.state.phase === "completed" && step.facts.includes("statusCompleted"),
});

export const nonRetryableFails = property({
  name: "nonRetryableFails",
  machine: activityProtocol,
  when: attemptResult.of({ result: { kind: "failed", retryable: false } }),
  holds: (step) => step.state.phase === "failed" && step.facts.includes("statusFailed"),
});

/** Completed on the second attempt of an activity with no deadline set. */
export const completedOnRetry = {
  phase: "completed",
  attempts: 2,
  scheduleToClose: "unset",
  scheduleToStart: "unset",
  startToClose: "unset",
} satisfies ProtocolState;

export const retryCompletes = property({
  name: "retryCompletes",
  machine: activityProtocol,
  when: attemptResult.of({ result: { kind: "completed" } }),
  holds: (step) => same(ProtocolState, step.state, completedOnRetry) && step.facts.includes("statusCompleted"),
});

export const cancelRequestedWhileStarted = property({
  name: "cancelRequestedWhileStarted",
  machine: activityProtocol,
  when: control.of({ control: "requestCancel" }),
  holds: (step) => step.state.phase === "cancelRequested" && step.facts.includes("statusCancelRequested"),
});

export const canceledByWorker = property({
  name: "canceledByWorker",
  machine: activityProtocol,
  when: attemptResult.of({ result: { kind: "canceled" } }),
  holds: (step) => step.state.phase === "canceled" && step.facts.includes("statusCanceled"),
});

export const terminated = property({
  name: "terminated",
  machine: activityProtocol,
  when: control.of({ control: "terminate" }),
  holds: (step) => step.state.phase === "terminated" && step.facts.includes("statusTerminated"),
});

export const scheduleToStartFires = property({
  name: "scheduleToStartFires",
  machine: activityProtocol,
  when: "scheduleToStart",
  holds: (step) => step.state.phase === "timedOut" && step.facts.includes("statusTimedOut:scheduleToStart"),
});

export const startToCloseFires = property({
  name: "startToCloseFires",
  machine: activityProtocol,
  when: "startToClose",
  holds: (step) => step.state.phase === "timedOut" && step.facts.includes("statusTimedOut:startToClose"),
});

// authoring: scenarios

// The paths the Queries run.

const noDeadline = start.of({ scheduleToClose: "unset", scheduleToStart: "unset", startToClose: "unset" });

export const completed = scenario({
  name: "completed",
  model: activityProtocol,
  starts: "unstarted",
  actions: [noDeadline, "attemptStart", attemptResult.of({ result: { kind: "completed" } })],
});

export const nonRetryable = scenario({
  name: "nonRetryable",
  model: activityProtocol,
  starts: "unstarted",
  actions: [noDeadline, "attemptStart", attemptResult.of({ result: { kind: "failed", retryable: false } })],
});

export const retriedThenCompleted = scenario({
  name: "retriedThenCompleted",
  model: activityProtocol,
  starts: "unstarted",
  actions: [
    noDeadline,
    "attemptStart",
    attemptResult.of({ result: { kind: "failed", retryable: true } }),
    "backoff",
    "attemptStart",
    attemptResult.of({ result: { kind: "completed" } }),
  ],
});

export const cancelRequestedThenCanceled = scenario({
  name: "cancelRequestedThenCanceled",
  model: activityProtocol,
  starts: "unstarted",
  actions: [
    noDeadline,
    "attemptStart",
    control.of({ control: "requestCancel" }),
    attemptResult.of({ result: { kind: "canceled" } }),
  ],
});

/** The worker stops before the start, so no attempt is in flight when the caller terminates. */
export const terminatedWhileScheduled = scenario({
  name: "terminatedWhileScheduled",
  model: activityProtocol,
  starts: "unstarted",
  actions: [noDeadline, "workerStop", control.of({ control: "terminate" })],
});

export const pausedThenCompleted = scenario({
  name: "pausedThenCompleted",
  model: activityProtocol,
  starts: "unstarted",
  actions: [
    noDeadline,
    control.of({ control: "pause" }),
    control.of({ control: "unpause" }),
    "attemptStart",
    attemptResult.of({ result: { kind: "completed" } }),
  ],
});

export const scheduleToStartExpires = scenario({
  name: "scheduleToStartExpires",
  model: activityProtocol,
  starts: "unstarted",
  actions: [
    start.of({ scheduleToClose: "unset", scheduleToStart: "expires", startToClose: "unset" }),
    "workerStop",
    "scheduleToStart",
  ],
});

export const startToCloseExpires = scenario({
  name: "startToCloseExpires",
  model: activityProtocol,
  starts: "unstarted",
  actions: [
    start.of({ scheduleToClose: "unset", scheduleToStart: "unset", startToClose: "expires" }),
    "attemptStart",
    "startToClose",
  ],
});

export const three = limits({ name: "three", steps: 3, actions: 3, search: 4096 });
export const four = limits({ name: "four", steps: 4, actions: 4, search: 32768 });
export const six = limits({ name: "six", steps: 6, actions: 6, search: 262144 });

// authoring: queries

export const completion = query({ name: "completion", find: completes, in: completed, limits: three });
export const nonRetryableFailure = query({
  name: "nonRetryableFailure",
  find: nonRetryableFails,
  in: nonRetryable,
  limits: three,
});
export const retry = query({ name: "retry", find: retryCompletes, in: retriedThenCompleted, limits: six });
export const cancel = query({ name: "cancel", find: canceledByWorker, in: cancelRequestedThenCanceled, limits: four });
export const terminate = query({ name: "terminate", find: terminated, in: terminatedWhileScheduled, limits: three });
export const pauseResume = query({ name: "pauseResume", find: completes, in: pausedThenCompleted, limits: six });
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
export const terminalHolds = query({ name: "terminalHolds", verify: terminalIsFinal, in: completed, limits: three });
export const pauseHolds = query({ name: "pauseHolds", verify: pausedIsNotDispatched, in: pausedThenCompleted, limits: six });

// authoring: set

// Standalone activities exist only under CHASM, so the functional set does not repeat over the
// implementation switch.

export const standaloneActivityTests = set({
  name: "standaloneActivityTests",
  purpose: "functional",
  bind: { caller: "driven", worker: "driven" },
  queries: [
    completion,
    nonRetryableFailure,
    retry,
    cancel,
    terminate,
    pauseResume,
    scheduleToStartTimeout,
    startToCloseTimeout,
  ],
});

export const standaloneActivityCanary = set({
  name: "standaloneActivityCanary",
  purpose: "canary",
  bind: { caller: "driven", worker: "observed" },
  queries: [completion, cancel],
});

export const standaloneActivityExploration = set({
  name: "standaloneActivityExploration",
  purpose: "exploratory",
  bind: { caller: "driven", worker: "driven" },
  machine: activityProtocol,
  cover: ["rows", "results", "classMembers"],
  budget: four,
});

// authoring: case

// The Cases (`case standaloneActivityCases realizes standaloneActivityTests as ...`) are the
// realization layer, out of scope for this comparison (SPEC.md).

// authoring: composition

// Composed with the worker of the activity's task queue, the stop is the worker's own phase change
// and every attempt start is the worker serving, so an attempt has a row only while the worker polls.

export const activityWorker = restrict(polling, ["workerStop", "serve"]);

export const standaloneActivity = compose({
  name: "standaloneActivity",
  members: { activity: activityProtocol, worker: activityWorker },
  sync: {
    workerStop: ["activity.workerStop", "worker.workerStop"],
    attemptStart: ["activity.attemptStart", "worker.serve"],
  },
  starts: ["activity.unstarted", "worker.polling"],
  ends: ["activity.completed", "activity.failed", "activity.canceled", "activity.terminated", "activity.timedOut"],
});

export type StandaloneActivityState = StateOf<typeof standaloneActivity>;

export const startedByPollingWorker = property({
  name: "startedByPollingWorker",
  machine: standaloneActivity,
  when: "attemptStart",
  holds: (step) => step.state.worker.phase === "polling",
});

/** An attempt starts while the worker polls and fails retryably; the worker then stops, so the retry is never dispatched and the schedule-to-start deadline fires. */
export const stoppedBeforeRetry = scenario({
  name: "stoppedBeforeRetry",
  model: standaloneActivity,
  starts: "activity.unstarted",
  actions: [
    member("activity", start.of({ scheduleToClose: "unset", scheduleToStart: "expires", startToClose: "unset" })),
    "attemptStart",
    member("activity", attemptResult.of({ result: { kind: "failed", retryable: true } })),
    "activity.backoff",
    "workerStop",
    "activity.scheduleToStart",
  ],
});

export const stoppedWorkerStartsNothing = query({
  name: "stoppedWorkerStartsNothing",
  verify: startedByPollingWorker,
  in: stoppedBeforeRetry,
  limits: six,
});

// authoring: end
