import Temporal.Feature.Worker.Model

/-!
# What the worker entity says

The machine's whole table read off the command's output: two phases, both terminal, a stop and a
resume moving between them, and a serve row only while the worker polls.
-/

namespace Temporal.Feature.Worker.Tests

open Umpire
open Umpire.Command
open Temporal.Feature.Worker

/-! ### The machine -/

#guard polling.table.states.length == 2
#guard polling.actionKeys == #["serve", "workerResume", "workerStop"]
#guard polling.stuck == none
#guard polling.starts.map polling.stateKeyFor == ["polling"]
#guard polling.ends.map polling.stateKeyFor == ["polling", "stopped"]

/- Every row as its key and the states it results in: a stopped worker has no serve row, and each
fault has a row only from the phase it leaves. -/
#guard polling.table.transitions.map (fun row =>
    (row.key, row.results.map fun result => polling.stateKeyFor result.state)) ==
  [("polling-serve", ["polling"]),
   ("polling-workerStop", ["stopped"]),
   ("stopped-workerResume", ["polling"])]

/- The declarations hang off the `temporal` root, with `Temporal.Feature` as scaffolding, as every
Temporal Model's do. -/
#guard [worker.id.value, workerStop.id.value, workerResume.id.value, serve.id.value,
    polling.targetId.value] ==
  ["temporal.worker.entity.worker", "temporal.worker.action.workerStop",
   "temporal.worker.action.workerResume", "temporal.worker.action.serve",
   "temporal.worker.target.polling"]

/-- info: 'Temporal.Feature.Worker.polling' depends on axioms: [propext] -/
#guard_msgs in
#print axioms polling

end Temporal.Feature.Worker.Tests
