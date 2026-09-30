# authoring: header
defmodule Temporal.Feature.Worker do
  @moduledoc """
  The worker entity.

  The worker of one task queue, as an entity of its own rather than a stutter row on the machines
  of the work it serves. A polling worker serves its queue; the `worker` party stops it and
  resumes it. The machine is the one a composition synchronizes with: a workflow, an operation or
  an activity whose progress needs a worker names `serve` beside its own action, and a stopped
  worker has no row for it.

  The module declares no set, Case, or Query: nothing here is realized on its own, and the
  Properties about a worker are the cross-entity ones a composition states.
  """

  use Umpire.Model

  # authoring: worker

  ## Entities and domains

  # A worker is named by the task queue it polls: the handler's worker and the workflow's worker
  # are two instances of this entity, told apart by their queue.
  entity :worker, key: :taskQueue

  domain Phase, [:polling, :stopped]

  defstate WorkerState, phase: Phase

  domain WorkerOutcome, [:accepted]

  # A worker records nothing of its own: its stop and resume are faults the Run records against
  # no entity, and what it serves is recorded by the work it serves. An empty domain is finite,
  # with no members.
  domain WorkerFact, []

  ## The actions
  #
  # The two faults are the `worker` party's and name no entity, as the outage machine spells them.
  # The serve action is the worker's own and takes no input, so a composition may synchronize it
  # with an action of any class.

  action :workerStop, party: :worker

  action :workerResume, party: :worker

  action :serve, party: :worker, on: :worker

  # authoring: polling

  ## The machine

  # A worker has no natural end: it may be left polling or stopped.
  defmachine :polling,
    for: :worker,
    state: WorkerState,
    outcome: WorkerOutcome,
    facts: WorkerFact do
    starts [:polling]
    ends [:polling, :stopped]

    # A polling worker stops; a stopped one has nothing to stop.
    defstep workerStop(state) do
      case state.phase do
        :polling -> moves(:stopped, [])
        :stopped -> []
      end
    end

    # A stopped worker resumes polling; a polling one has nothing to resume.
    defstep workerResume(state) do
      case state.phase do
        :stopped -> moves(:polling, [])
        :polling -> []
      end
    end

    # A polling worker serves and keeps polling; a stopped one serves nothing.
    defstep serve(state) do
      case state.phase do
        :polling -> stay()
        :stopped -> []
      end
    end
  end

  # authoring: end
end
