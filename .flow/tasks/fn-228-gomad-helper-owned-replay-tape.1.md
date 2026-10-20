---
satisfies: [R8]
---
# fn-228-gomad-helper-owned-replay-tape.1 Rust replay-plan projection with a Go/Rust conformance corpus

**Size:** M
**Touches:** tools/gomad/internal/backends/wasm/wasmhost/src/model/replay_plan.rs, tools/gomad/internal/backends/wasm/wasmhost/src/model.rs, tools/gomad/internal/backends/wasm/wasmhost/src/model/mod.rs (if module wiring requires it), tools/gomad/internal/backends/wasm/wasmhost/tests/replay_plan.rs (optional external integration path; prefer a colocated corpus test in src/model/replay_plan.rs reusing the existing Decision type), tools/gomad/internal/backends/wasm/wasmhost/tests/testdata/replay_plan/**, tools/gomad/internal/model/choice/tape_conformance_test.go, tools/gomad/internal/backends/wasm/Makefile, tools/gomad/Makefile

## Description
Add a `replay_plan` module to the helper crate that projects a complete v3 choice trace into the replay plan exactly as `choice.ProjectReplayPlan` does: drop observation and single-alternative records, carry each select_result's readiness onto the poll decisions it names (polled-1 consecutive records from origin, site and alternatives count checks), encode and hash the plan with the execution identity. Reuse the existing Rust `Decision` type and SHA-256 helpers. Create a corpus of retained trace fixtures (take them from existing Go choice test fixtures plus one storage-crash interrupted trace) with expected plan bytes and digest; a Rust test projects every fixture, a Go test projects the same fixtures through `ProjectReplayPlan` and compares bytes. Extend that same oracle corpus to bounded typed prefix cutoffs and final rank controls, matching `Prefix`, `BuildRankPrefix` and `BuildForcedRankPrefix` canonical bytes/digests and readiness. Keep rank overrides out of trace evidence. Wire regeneration into the existing two-pass corpus workflow in the Makefile.

## Acceptance
Every fixture projects to byte-equal plan bytes and equal digest on both sides, including zero/short/full prefixes, final forced rank and reconstruction intent, and rejected out-of-range/incorrectly placed controls; corrupt fixtures (result naming decisions before origin, broken poll pattern, double-named decision) are rejected on both sides with the same classification; `make validate` regenerates the corpus byte-identically on a second pass.

## Done summary
R8: Rust projects the complete choice trace with shared Decision/readiness rules and typed prefix/final-rank controls. Go/Rust corpus59fixtures agrees on bytes/digests and refusals; two-pass corpus regeneration is byte-identical. Combined canonical Quick at0561 passed; all71 helper inputs match deletion-free12cdc and fresh helper digestab405. Prior full Rust223pass/2existingignored and seven-mode lint are equivalent-input reused evidence, not claimed freshcombined fullsuite. Independent combined source review SHIP07ebfdb4ee. See receiver FINAL_CUSTODY.md and tracked replay identity ledger.

Evidence index: /home/agent/wt/evidence/fn232/fn228-replay-inputs-2026-10-20/FINAL_CUSTODY.md; original producer handovers and narrow review-fix appends remain authenticated independently. Source acceptance only; spec closure requires separate completion review.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 8d37d31bc9, 1ef7bd73b5, 0561e32514
- Tests: make validate; Go replay-plan conformance corpus59; Rust replay_plan, two-pass corpus regeneration byte comparison, combined Quick exit0; helper source/pin manifest verification
- PRs: