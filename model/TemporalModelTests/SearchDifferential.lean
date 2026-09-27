import Umpire.Search.Tests.Differential
import Temporal.Feature.Nexus.Caller.Model
import Temporal.Feature.Nexus.Control.Model
import Temporal.Feature.Nexus.Pair.Model
import Temporal.Feature.Nexus.Success.RaceSyntaxTests
import Temporal.Feature.Nexus.Success.Tests
import Temporal.Feature.Nexus.Tests.Machines
import Temporal.Feature.System.Info.Model
import Temporal.Feature.Workflow.Outage.Model
import Temporal.Feature.Workflow.Start.Model

/-!
# Both search backends over every Temporal feature Query

The fn-88 R8 differential over every `query` block the Temporal feature Models and their test
modules declare, which the Umpire tests cannot import: each Query runs on `reference` and on `veil`
through `AdmittedQuery.searchWith`, compared as `Umpire.SearchTests.Differential` defines, beside
the Scenario automaton and Property monitor checks within its Limits (R14, R6). One line per Query
says which backend `searchWith .veil` ran and why, what each backend answered and explored --
candidate paths on `reference`, product states on `veil` -- and the checks' results. A Query whose
admission fails is listed with the reason; one over several instances whose search selects nothing
has no admission to re-run, so it is listed and not compared.
-/

namespace TemporalModelTests.SearchDifferential

/--
info: Temporal.Feature.Nexus.Caller.asyncCompletion: veil default, found 4 paths, found 4 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Caller.asyncFailure: veil default, found 4 paths, found 4 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Caller.handlerError: veil default, found 3 paths, found 3 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Caller.retry: veil default, found 5 paths, found 5 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Caller.scheduleToStartTimeout: veil default, found 4 paths, found 4 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Caller.startToCloseTimeout: veil default, found 4 paths, found 4 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Caller.syncCompletion: veil default, found 3 paths, found 3 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Caller.terminalHolds: veil default, verified-within-limits 4 paths, verified-within-limits 4 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Control.forgedCompletion: veil default, found 4 paths, found 4 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Pair.bothComplete: veil default, found 7 paths, found 7 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Success.RaceSyntax.cancellation: veil default, found 4 paths, found 4 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Success.RaceSyntax.cancellationVerified: veil default, verified-within-limits 4 paths, verified-within-limits 4 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Success.Tests.boundedCompletion: veil default, limit-reached 1 paths, limit-reached 1 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Success.Tests.completionHolds: veil default, verified-within-limits 3 paths, verified-within-limits 3 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Success.Tests.configuredQuery: veil default, found 3 paths, found 3 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Success.Tests.forkedCompletion: veil default, limit-reached 2 paths, limit-reached 2 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Success.Tests.probeQuery: veil default, found 3 paths, found 3 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Success.Tests.quickProbed: veil default, found 3 paths, found 3 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Success.Tests.renamedQuery: veil default, found 3 paths, found 3 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Success.Tests.secondCompletesFirst: veil default, found 5 paths, found 5 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Success.Tests.slowProbed: veil default, found 3 paths, found 3 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Success.Tests.twoCompletions: veil default, found 5 paths, found 5 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Success.Tests.twoCompletionsCutShort: not admitted: not selected: limit-reached
---
info: Temporal.Feature.Nexus.Success.Tests.twoTransitions: not admitted: the instances
---
info: Temporal.Feature.Nexus.Success.Tests.unevenCompletion: not admitted: the instances
---
info: Temporal.Feature.Nexus.Success.Tests.ungappedCompletion: veil default, found 3 paths, found 3 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Success.Tests.unsatisfiableCompletion: veil default, unsatisfiable 1 paths, unsatisfiable 1 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Success.Tests.verifiedCompletion: veil default, verified-within-limits 3 paths, verified-within-limits 3 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Success.completion: veil default, found 3 paths, found 3 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Tests.Machines.attemptCompletes: veil default, found 4 paths, found 4 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Tests.Machines.attemptOutOfBudget: veil default, limit-reached 1 paths, limit-reached 1 states; automaton ok, monitors ok
---
info: Temporal.Feature.Nexus.Tests.Machines.retryCompletes: veil default, found 4 paths, found 4 states; automaton ok, monitors ok
---
info: Temporal.Feature.System.Info.answered: veil default, found 2 paths, found 2 states; automaton ok, monitors ok
---
info: Temporal.Feature.Workflow.Outage.survived: veil default, found 5 paths, found 5 states; automaton ok, monitors ok
---
info: Temporal.Feature.Workflow.Start.started: veil default, found 2 paths, found 2 states; automaton ok, monitors ok
-/
#guard_msgs in
run_cmd Umpire.SearchTests.Differential.sweep [`Temporal]

end TemporalModelTests.SearchDifferential
