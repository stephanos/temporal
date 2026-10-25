---
satisfies: [R3]
---
# fn-252-quick-consolidation-wins-in-model-code.3 Delete committed spike output; no copied Kettle in the spike

## Description
Implement **Spike output (A6)**: delete `design/native-symbolic/out/` and `out-big/`, ignore them, and include the shared sample_kettle model by `#[path]` instead of the copies.

## Focused source-lane contract

Owner-authorized early source lane (2026-10-25), ahead of whole fn248. Parent fn252 keeps its whole-fn248 prerequisite, and no other fn252 task is admitted by this work.

**Touches:** [model/design/native-symbolic/**, model/umpire/dev/fixtures/fixtures/sample_kettle/model.rs]

Reuse the actual shared sample_kettle source without changing its baseline threshold100. Generate the big threshold1_000_000 variant privately, with missing/ambiguous declaration guards. Preserve distinct claim99/heat2e9 controls and strict native witness refusal. Necessary current frontend/witness API repairs and standalone manifest/lock reconciliation are within scope. Regenerate actual baseline/control outputs and retain source/solver evidence; incomplete/timeout is not solver agreement. Focused checks: `bash -n model/design/native-symbolic/run.sh` and locked offline Cargo check/test for `model/design/native-symbolic/Cargo.toml`, with formatting. Retain prior actual solver runs with their exact source limitations rather than repeating big exploration or running a full Model gate. Whole-spec R9/current guest publication remain deferred.

## Acceptance
- [ ] No committed spike output; `run.sh` still regenerates it.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
