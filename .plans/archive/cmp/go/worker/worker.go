// Package worker is the worker entity.
//
// The worker of one task queue, as an entity of its own rather than a stutter row on the machines of
// the work it serves. A polling worker serves its queue; the `worker` party stops it and resumes it.
// The machine is the one a composition synchronizes with: a workflow or an operation whose progress
// needs a worker names `serve` beside its own action, and a stopped worker has no row for it.
//
// The package declares no set, Case, or Query: nothing here is realized on its own, and the
// Properties about a worker are the cross-entity ones a composition states.
package worker

import "go.temporal.io/umpire/model/umpire"

//go:generate go run ../umpire/cmd/finite -type=Phase,WorkerOutcome

// authoring: worker

// ### Entities and domains

// Worker is named by the task queue it polls: the handler's worker and the workflow's worker are
// two instances of this entity, told apart by their queue.
var Worker = &umpire.Entity{Name: "worker", Key: "taskQueue"}

type Phase uint8

const (
	PhasePolling Phase = iota
	PhaseStopped
)

type WorkerState struct {
	Phase Phase
}

type WorkerOutcome uint8

const (
	Accepted WorkerOutcome = iota
)

// WorkerFact has no members: a worker records nothing of its own. Its stop and resume are faults the
// Run records against no entity, and what it serves is recorded by the work it serves. An interface
// nothing implements is Go's empty inductive.
//
//sumtype:decl
type WorkerFact interface{ isWorkerFact() }

// ### The actions
//
// The two faults are the `worker` party's and name no entity, as the outage machine spells them. The
// serve action is the worker's own and takes no input, so a composition may synchronize it with an
// action of any class.

var WorkerStop = &umpire.Action0{Name: "workerStop", Party: umpire.Worker}

var WorkerResume = &umpire.Action0{Name: "workerResume", Party: umpire.Worker}

var Serve = &umpire.Action0{Name: "serve", Party: umpire.Worker, On: Worker}

// authoring: polling

// ### The machine

type workerStep = umpire.Step[WorkerState, WorkerOutcome, WorkerFact]

// A polling worker stops; a stopped one has nothing to stop.
func stopStep(state WorkerState) []workerStep {
	if state.Phase != PhasePolling {
		return nil
	}
	return []workerStep{{Outcome: Accepted, State: WorkerState{Phase: PhaseStopped}}}
}

// A stopped worker resumes polling; a polling one has nothing to resume.
func resumeStep(state WorkerState) []workerStep {
	if state.Phase != PhaseStopped {
		return nil
	}
	return []workerStep{{Outcome: Accepted, State: WorkerState{Phase: PhasePolling}}}
}

// A polling worker serves and keeps polling; a stopped one serves nothing.
func serveStep(state WorkerState) []workerStep {
	if state.Phase != PhasePolling {
		return nil
	}
	return []workerStep{{Outcome: Accepted, State: state}}
}

// Polling has no natural end: a worker may be left polling or stopped.
var Polling = &umpire.Machine[WorkerState, WorkerOutcome, WorkerFact]{
	Name:   "polling",
	For:    Worker,
	States: umpire.Fields[WorkerState](),
	Starts: []WorkerState{{Phase: PhasePolling}},
	Ends:   func(WorkerState) bool { return true },
	Steps: umpire.Steps(
		umpire.Bind0(WorkerStop, stopStep),
		umpire.Bind0(WorkerResume, resumeStep),
		umpire.Bind0(Serve, serveStep),
	),
}

// authoring: end
