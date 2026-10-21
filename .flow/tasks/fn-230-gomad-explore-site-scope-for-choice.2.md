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
R2/R7/R8/R10 accepted. Scope filters proposal generation only; all original decisions stay seed-driven and retained. Authenticated round ledgers prove only matching select/runnable sites are forced, per-round admitted/skipped counts persist, and changed patterns/resolution digests refuse with status2. Under budget3 the actual fixture reaches depth5 scoped versus2 unscoped. Empty scope retains its base despite keep-successes=none.

Public runs /home/agent/wt/evidence/fn230-site-scope-public-03 and -04 each pass34 command receipts with byte-identical observation SHA256 e895c690cdc2ebe767b37ae8e5ba833f22a905deaf5cb3bb119afeb4d77485c7. Independent site_scope_review authenticated actual artifacts/journals and found no blocking correctness findings. All9 unscoped Go/Rust controller cases repeat raw outputs byte-for-byte and retain unchanged tracked goldens-choice-controller.json state/segment hashes. Evidence: workspace-parent .build-cache/fn230-site-scope/choice-controller.

Root integrated Go choice/generator/toolchain/determinism tests pass. Integrated734 helper full tests and all-target Clippy plus host feature62/62 tests and all-target Clippy all exit0; receipt source-freeze authenticates unchanged relevant source trees despite later metadata/Python-only commits. Raw argv/status/stdout are /home/agent/wt/evidence/fn232-replay/integrated734-*.

stage: impl-review - ran (fresh site_scope_review, same-family session-model fallback)

Task1's public plan receiving requirement stays open under fn232.11. Final whole-system qualification stays fn232.13; neither is waived or claimed by this task closure.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 97ac127b01, 38572d7e49, 734bc7c566
- Tests: public scope runs03/04:34 commands each exit0; observations SHA256 e895c690cdc2ebe767b37ae8e5ba833f22a905deaf5cb3bb119afeb4d77485c7 identical, choice-controller-check.py:9 fresh Go/Rust cases, raw repeats and unchanged goldens exit0, integrated734 helper fulltests/alltargetClippy and host featurelib62tests/alltargetClippy: all exit0 with source-freeze0, root Go choice/generator/toolchain/determinism packages: exit0, independent site_scope_review: no blocking findings; actual journal/artifact authentication rerun0
- PRs: