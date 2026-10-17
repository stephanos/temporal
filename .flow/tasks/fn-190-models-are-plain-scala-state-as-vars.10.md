---
satisfies: [R8, R14, R15]
---
# fn-190-models-are-plain-scala-state-as-vars.10 Phase 2: samples lift and model-check; Go budget, start list and vacuity; oracles and equivalence

## Description
**Phase 2 (IR and plugin), owner phasing 2026-10-17.** The IR, Go and equivalence half of fn-190.7, moved here.

The six rewritten samples compile with the plugin under the native strict flags and lift. R14/R15's consumer side: the Go explorer iterates the start list, reports vacuity in the spec's shapes; the derived negative-control machines and refuted claims are `assertRefuted` in the sample tests; remove the bounds field from the IR `Query` message; give the Go checker a budget argument (default plus per-feature overrides in the gate's configuration, none in a Model); every result carries the budget and completion, a cut exploration answers unknown; record the budget in the inventory of intended answers. Regenerate the retained oracles once. Prove equivalence: the six samples' reachable graphs are unchanged and every Query answer on the default budget matches the inventory; a changed answer is a finding to fix, not a regeneration. Needs fn-173.4's native-collection IR schema for Set/Map/Int state, and the owner's release of the schema/Go hold.

**Size:** L

### Verification
- Six samples compile and lift; the four native sample suites pass on the evaluator (fn-190.3).
- Graph comparison per sample recorded; the samples model-check stage (fn-191.3) passes on the default budget with every answer matching the inventory.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
