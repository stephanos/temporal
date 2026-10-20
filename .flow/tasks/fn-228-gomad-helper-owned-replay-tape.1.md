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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
