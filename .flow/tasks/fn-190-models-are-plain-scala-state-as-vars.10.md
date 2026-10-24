---
satisfies: [R8, R14, R15]
---
# fn-190-models-are-plain-scala-state-as-vars.10 Phase 2: samples lift and model-check; Go budget, start list and vacuity; oracles and equivalence

## Description
> Deferred: the parent spec's 2026-10-21 execution hold applies. These criteria preserve the filed scope; they do not authorize work on the frozen Scala tree. Revalidate against current Rust/successor work before resuming.

**Recut 2026-10-18 (one-language series):** The Go parts are dropped: budget and vacuity already live in the Scala checker (fn-196.2) and there is no Go model-check after fn-201. What remains: the six samples lift in-process through `framework/lift`, and IR `Query.bounds` is removed. If nothing else remains, close into fn-201/fn-195.4.

**Phase 2 (IR and plugin), owner phasing 2026-10-17.** The IR, Go and equivalence half of fn-190.7, moved here.

The six rewritten samples compile with the plugin under the native strict flags and lift. R14/R15's consumer side: the Go explorer iterates the start list, reports vacuity in the spec's shapes; the derived negative-control machines and refuted claims are `assertRefuted` in the sample tests; remove the bounds field from the IR `Query` message; give the Scala checker a budget argument (default plus per-feature overrides in the gate's configuration, none in a Model); every result carries the budget and completion, a cut exploration answers unknown; record the budget in the inventory of intended answers. Regenerate the retained oracles once. Prove equivalence: the six samples' reachable graphs are unchanged and every Query answer on the default budget matches the inventory; a changed answer is a finding to fix, not a regeneration. Needs fn-173.4's native-collection IR schema for Set/Map/Int state, and the owner's release of the schema/Go hold.

**Size:** L

### Verification
- Six samples compile and lift; the four native sample suites pass on the evaluator (fn-190.3).
- Graph comparison per sample recorded; the samples model-check stage (fn-191.3) passes on the default budget with every answer matching the inventory.
## Acceptance
- [ ] Reconcile the 2026-10-18 recut before resumption: no Go explorer/budget work is reintroduced; record which remaining obligations belong to fn-201/fn-195.4.
- [ ] All six samples compile and lift with preserved reachable-graph and intended-answer evidence; changed answers fail rather than being regenerated as accepted.
- [ ] Remaining Query.bounds retirement and budget/completion/vacuity reporting have explicit owning evidence; cut exploration remains unknown.
- [ ] Deferred backend/schema obligations remain open unless their actual successor proof is complete.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: