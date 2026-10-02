// Package worker is the worker entity: the worker of one task queue, as an entity of its own
// rather than a stutter row on the machines of the work it serves. A polling worker serves its
// queue; the worker party stops it and resumes it. The machine is the one a composition
// synchronizes with: a workflow or an operation whose progress needs a worker names serve beside
// its own action, and a stopped worker has no row for it.
//
// The package declares no set, Case, or Query: nothing here is realized on its own, and the
// Properties about a worker are the cross-entity ones a composition states.
//
// Ported from model/lean/Temporal/Feature/Worker/Model.lean.
package worker

import "go.temporal.io/server/model/go/umpire"

// Family is the root of this package's Definition IDs.
const Family umpire.Family = "temporal.worker"

// Party is the worker party, which stops and resumes the worker.
const Party umpire.Party = "worker"

// ### Entities and domains

// Worker is named by the task queue it polls: the handler's worker and the workflow's worker are
// two instances of this entity, told apart by their queue.
var Worker = &umpire.Entity{Name: "worker", Key: "taskQueue"}

// Phase is whether the worker polls.
type Phase string

const (
	Polling Phase = "polling"
	Stopped Phase = "stopped"
)

func (Phase) Values() []Phase { return []Phase{Polling, Stopped} }

// State is the worker's state.
type State struct {
	Phase Phase
}

// Outcome is every worker step's outcome.
type Outcome string

const Accepted Outcome = "accepted"

func (Outcome) Values() []Outcome { return []Outcome{Accepted} }

// Fact is empty: a worker records nothing of its own. Its stop and resume are faults the Run
// records against no entity, and what it serves is recorded by the work it serves.
//
//sumtype:decl
type Fact interface{ isFact() }

var _ = umpire.Sum[Fact]()

// Step is one worker step.
type Step = umpire.Step[State, Outcome, Fact]

// ### The actions
//
// The two faults are the worker party's and name no entity, as the outage machine spells them. The
// serve action is the worker's own and takes no input, so a composition may synchronize it with an
// action of any class.

var (
	WorkerStop   = umpire.NewAction0("workerStop", Party)
	WorkerResume = umpire.NewAction0("workerResume", Party)
	Serve        = umpire.NewAction0("serve", Party, umpire.On(Worker))
)

// ### The machine

// stopStep: a polling worker stops; a stopped one has nothing to stop.
func stopStep(s State) []Step {
	if s.Phase != Polling {
		return nil
	}
	return []Step{{Outcome: Accepted, State: State{Phase: Stopped}}}
}

// resumeStep: a stopped worker resumes polling; a polling one has nothing to resume.
func resumeStep(s State) []Step {
	if s.Phase != Stopped {
		return nil
	}
	return []Step{{Outcome: Accepted, State: State{Phase: Polling}}}
}

// serveStep: a polling worker serves and keeps polling; a stopped one serves nothing.
func serveStep(s State) []Step {
	if s.Phase != Polling {
		return nil
	}
	return []Step{{Outcome: Accepted, State: s}}
}

// PollingMachine is the worker. A worker has no natural end: it may be left polling or stopped.
var PollingMachine = umpire.NewMachine[State, Outcome, Fact](Family, "polling").
	For(Worker).
	Starts(State{Phase: Polling}).
	Ends(func(State) bool { return true }).
	Step0(WorkerStop, stopStep).
	Step0(WorkerResume, resumeStep).
	Step0(Serve, serveStep)
