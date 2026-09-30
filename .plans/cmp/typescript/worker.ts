// worker.ts — the worker entity.
//
// The worker of one task queue, as an entity of its own rather than a stutter row on the machines
// of the work it serves. A polling worker serves its queue; the `worker` party stops it and resumes
// it. The machine is the one a composition synchronizes with: a workflow or an operation whose
// progress needs a worker names `serve` beside its own action, and a stopped worker has no row for
// it. The module declares no set, Case or Query: the Properties about a worker are the cross-entity
// ones a composition states.

import { action, defineMachine, entity, enumOf, struct, type Member, type Step } from "./umpire";

// authoring: worker

/** A worker is named by the task queue it polls: the handler's worker and the workflow's worker are two instances, told apart by their queue. */
export const worker = entity("worker", { key: "taskQueue" });

export const Phase = enumOf(["polling", "stopped"]);
export type Phase = Member<typeof Phase>;

export const WorkerState = struct({ phase: Phase });
export type WorkerState = Member<typeof WorkerState>;

export type WorkerOutcome = "accepted";

/** A worker records nothing of its own: its stop and resume are faults the Run records against no entity, and what it serves is recorded by the work it serves. */
export type WorkerFact = never;

type WorkerStep = Step<WorkerState, WorkerOutcome, WorkerFact>;

// The two faults are the `worker` party's and name no entity. The serve action is the worker's own
// and takes no input, so a composition may synchronize it with an action of any class.

export const workerStop = action({ name: "workerStop", party: "worker" });
export const workerResume = action({ name: "workerResume", party: "worker" });
export const serve = action({ name: "serve", party: "worker", on: worker });

// authoring: polling

/** A polling worker stops; a stopped one has nothing to stop. */
export function stopStep(state: WorkerState): WorkerStep[] {
  if (state.phase !== "polling") return [];
  return [{ outcome: "accepted", state: { phase: "stopped" }, facts: [] }];
}

/** A stopped worker resumes polling; a polling one has nothing to resume. */
export function resumeStep(state: WorkerState): WorkerStep[] {
  if (state.phase !== "stopped") return [];
  return [{ outcome: "accepted", state: { phase: "polling" }, facts: [] }];
}

/** A polling worker serves and keeps polling; a stopped one serves nothing. */
export function serveStep(state: WorkerState): WorkerStep[] {
  if (state.phase !== "polling") return [];
  return [{ outcome: "accepted", state, facts: [] }];
}

/** A worker has no natural end: it may be left polling or stopped. */
export const polling = defineMachine({
  name: "polling",
  for: worker,
  state: WorkerState,
  starts: ["polling"],
  ends: ["polling", "stopped"],
  actions: [workerStop, workerResume, serve],
  evidence: {},
  steps: {
    workerStop: stopStep,
    workerResume: resumeStep,
    serve: serveStep,
  },
});

// authoring: end
