---
satisfies: [R2, R7, R8, R10]
---
# fn-230-gomad-explore-site-scope-for-choice.2 Frontier filtering, per-round counts and resume refusal

**Touches:** tools/gomad/rust/gomad/src/campaign/exploration/**, tools/gomad/rust/gomad/src/cli/resume.rs, tools/gomad/testdata/parity/**
**Semantic references only:** tools/gomad/internal/runner/choice_exploration_campaign.go, runner/internal/exploration/choice.go, runner/exploration_round.go, runner/resume.go

## Description
Implement once on the Rust host after fn-232.8 and the existing site-spec predecessors; the Go locations below document existing semantics only. Port the behavior into the Rust ownership surface above and extend the shared golden/behavioral corpus. Do not create or extend a Go-host implementation.
Apply the resolved fingerprint set where forced-alternative proposals are generated: select-kind decisions outside the set yield no proposals; unavailable symbols count as out of scope; runnable-kind decisions are out of scope in this task and counted as skipped. Symbolize newly seen sites per round and extend the set. Record `forced_in_scope` and `skipped_out_of_scope` per round in the summary and campaign record. `resume` compares persisted patterns and resolved digest and refuses a change with the existing changed-bound error.

## Acceptance
A scoped campaign on a fixture target forces only in-scope sites (asserted from the round ledger), reports per-round counts, reaches deeper prefixes than the unscoped run under the same budget (asserted on the fixture), and resume with a changed scope is refused with the existing status; unscoped behavior is byte-identical to before.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
