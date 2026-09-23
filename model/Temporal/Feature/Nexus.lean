import Temporal.Feature.Nexus.Caller.Model
import Temporal.Feature.Nexus.Pair.Model
import Temporal.Feature.Nexus.Control.Model

/-!
# The Nexus Model

This is the single Nexus entry import. The feature is authored with the commands:

1. `Temporal.Feature.Nexus.Caller.Model` is the caller Model: the operation entity, its actions
   with their parties, the recorded events, the product machine `nexusProduct` (what an operation
   does) and the protocol machine `nexusProtocol` (how the caller drives it, with deadlines and
   retries), the Properties, the Scenarios, the Queries, the query sets and the Cases.
2. `Temporal.Feature.Nexus.Pair.Model` is the typed example over two instances of the operation:
   one machine that keeps the asynchronous success path, one field relation between the completed
   and scheduled events, and its Case.
3. `Temporal.Feature.Nexus.Control.Model` is the negative control: the caller Model's entity and
   actions over a machine that adds one row the platform never takes, so that its one Case's Run
   is violated on purpose, for the replay of fn-22 to admit, rerun and reduce. It enters no set of
   the caller Model and no regression view.
4. `Temporal.System.Nexus.Core` and `Temporal.System.Nexus.ImplementationLink` are the
   independently authored System mechanism and its correspondence with the product machine.

`Temporal.Feature.NexusTests` is the compiled facade-only walkthrough. The Caller Model's own tests
(`Temporal.Feature.Nexus.Caller.Tests`) and `COVERAGE.md` beside it are the coverage record.
-/
