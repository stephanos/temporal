---
satisfies: [R7]
---
# fn-228-runner-and-run-the-one-go-host-loop.7 After committed fn-243.2: Runner and Run write-up, R33-R39 discharge and R40 proof checkpoint

## Description
Add the DESIGN.md section recording the vocabulary sentence, Propose, the Run schema, per-package counts before and after, live timings and make model-time; discharge independently earned R33 to R39 rows and record the delivered R40 architecture with its remaining unheld cancellation obligation assigned to fn-243.5; update README and MIGRATE for the runner package and renames. MILESTONES.md is the root agent's.

### Required integration contract

Document the actual bootstrap and polling protocol, exact replay event schema, immutable publication key, progress contract and before/after counts including relocated Runner code. Update current names and milestone references; retain historical evidence. R33-R40 close only with the recorded real live/replay and deletion-coverage proof.

### Dependency change (2026-10-23, revised 2026-10-24)

fn-228.5 is superseded by fn-243 (programmed workers) and closed with nothing built. Within this spec this task depends on .6; across specs it waits for fn-244.2 (done) and fn-243.2, so R40 is written once, in its final form: a worker in any language follows a Model-issued program, the Run records the executor kind and what happened, the Shim is superseded, the proposal vocabulary is action, bootstrap and read, and the hold fault replaces the worker gate. R37's live callback path is recorded as retired for Temporal (Answer stays for other systems). fn-243.4 checks the section afterwards and adds the design-note pointers.

### Ordering note (2026-10-24)

Runs after fn-244.2, which moves the Go Runner out of the generated protobuf tree, so the write-up names its final path.
## Acceptance
- [ ] Document the exact current atomic-Model refusal for unheld open-state cancellation, held-before-start and terminal support, and the deferred fn-243.5 proof. Keep fn-228's final R40/R7 obligation open until that evidence is committed; this task is the architecture documentation checkpoint.
- [ ] DESIGN.md section and R33 to R40 rows updated
- [ ] README and MIGRATE updated
- [ ] model-time pasted

## Done summary
Documented Runner and Run, earned R33–R39 discharge, final package names, protocol/replay/publication/progress contracts, comparable package counts and attributed Model/live timings. Final R40/fn-228 R7 remains open in fn-243.5 after independent fn-237 cooperative semantics/binding; the exact unheld cancellation refusal, original counterexample and non-Go recorded-not-proven gap remain explicit. Both parent specs are open.

Tier: session (jev-unavailable(no_key)). No actual implementer-model claim inferred from dispatch. Primary reviewer gpt-6.1-sol/high is the writer family; no cross-family independence claim.

Baseline: none (task/spec define no Quick commands). Documentation-only manual scope: DESIGN, README, MIGRATE and verification evidence, no executable edits. Independent checker verified 24 package counts, added local links/anchors, current crate names and the committed guest hash; git diff --check passed. CLI recount corrected helper60 to actual61; current expanded totals are6379/6748/7372/7405, original targets1759/3205 unchanged. Durable evidence: .flow/artifacts/fn-228.7/verification.md.

Classifier: exit1 FULL/unmatched for MIGRATE.md, model/DESIGN.md and model/README.md, not a tier-B pass. Explicit bounded docs-only scope exception: no new Model/live run; no fabricated GATE_SKIPPED or green gate receipt, and no baseline receipt reuse past the roadmap commit. Existing required model-time/live results are attributed only to committed fad276 source evidence; broad legacy tooling stays inherited red. The primary reviewers checked this checkpoint without new Model/live execution.

stage: impl-review - ran (first-round three-axis codex:gpt-6.1-sol:high; all SHIP, zero findings; finalized receipt /tmp/impl-review-receipt-8f37faba39e2-fn-228-runner-and-run-the-one-go-host-loop.7.json)
stage: plan-sync - skipped(config: planSync.enabled != true)

No source lanes/worktrees created; no push, PR, history rewrite or MILESTONES edit. This task is the documentation checkpoint, not either parent spec's closure.
## Evidence
- Commits: cdf3abf909e55eaaf6786659cc2332a5e5aeb3fb, f68fbe6b44693ccc884fc2d079df3ec6e542d1b7
- Tests: python3 .flow/tmp/fn2287-check.py (exit 0: 24 package counts, added links/anchors, names, guest hash), git diff --check (exit 0), flowctl gate classify --base 62fc6611eeaa2b39330263c2e5c4fa9eb25c6a55 (exit 1: FULL unmatched documentation paths; bounded manual docs-only scope exception recorded)
- PRs: