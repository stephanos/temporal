---
satisfies: [R1]
---
# fn-221-rejecting-outcomes-reject-steps-phased.1 Outcomes and reject!: Rejection, Outcome, the outcome frame, macro edge rules and outcome-taking postconditions

## Description
Add `Rejection`, step `Outcome`, `outcome::{evaluate, reject}` and statement-form `reject!` to `umpire_dsl`, mirroring scoped `claim::{evaluate, fail}` and `check!`. The macro applies the parent spec's accepted/disabled/rejected/invalid edge table and permits an outcome-taking postcondition. Only modules using `reject!` switch their associated outcome from unit. The checker refuses rejection after mutation, naming the action and before state.

**Size:** M
**Touches:** [model/README.md, model/umpire/dsl/src/**, model/umpire/dsl/tests/**, model/umpire/dsl/macros/src/machine.rs, model/umpire/dsl/macros/tests/**, model/umpire/tests/fixtures/**, model/umpire/checker/src/**, model/umpire/checker/tests/**, model/umpire/wasm/src/refusal.rs, model/umpire/wasm/src/runtime.rs, model/umpire/wasm/tests/**]

### Approach

- Current fn-220 formatting supersedes the spec's Spell references: states/actions use Display, outcomes/facts use Debug. Derive the new outcomes' canonical Debug spellings rather than restoring Spell.
- The DSL already exports an aggregate `Outcome::{Verified,Refuted,Unknown}`. Reserve `Outcome` for the requested step outcome and rename that aggregate to `CheckOutcome`, updating its existing live consumers. Keep rendered aggregate variants and all pre-existing golden bytes unchanged. Do not combine these two concepts or keep a misleading old alias.
- New shared fixtures cover accepted, disabled, rejected self-loop, rejecting helper and rejecting-then-mutating refusal. Preserve all existing Model behavior and Kettle goldens; add a separate macro snapshot for rejecting modules while leaving nonrejecting snapshots unchanged.
- Follow claim's frame restoration on nested evaluation and unwinding. The R1 requirement that rejection outside an active frame is inert governs the parent text's contradictory optional debug-assert wording; direct out-of-frame use must not panic. Pin that behavior and helper return semantics.
- Add the minimal generic checker refusal path and its guest refusal-code mapping if needed. Do not invent a Model edge to represent the invalid implementation or allow changed-state rejection to pass as disabled. Root approves the listed test/caller/error-mapping scope expansions required for a green complete model gate.
- Explain any new checked-step/error interface to the conductor before committing a broad Machine API change, so the parallel TLC lane can adapt. Its worktree does not modify DSL/checker/Model source.
- Root approved `Machine::checked_step` with a default `Ok(self.step(..))` and a typed `StepError`, preserving manual implementations and nonrejecting macro snapshots. All checker/guest replay and Answer response-validation callers must use it; changed-state rejection is a typed refusal before publication, not a disabled edge, unexplained observation or WASM trap. The runtime/test scope expansion above covers these maintained callers and controls.

### Quick commands

Focused native fixtures/macro/negative controls first, then `make model-check`. The task acceptance and parent edge table remain authoritative. Preserve red/green evidence and existing golden identities.
## Acceptance
- [ ] R1 fixtures pass natively; rows/report spell outcomes; rejecting-then-mutating is a named checker refusal.
- [ ] Kettle and every pre-existing fixture golden and the existing macro snapshot are byte-identical.
- [ ] `make model-check` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
