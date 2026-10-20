---
satisfies: [R1, R4, R5, R6]
---
# fn-228-gomad-helper-owned-replay-tape.3 Runner stops projecting: pass trace and expectations, compare helper terminal counts

**Size:** M
**Touches:** tools/gomad/internal/runner/replay_operation.go, tools/gomad/internal/backends/wasm/backend/inspection.go, tools/gomad/internal/backends/wasm/backend/lifecycle.go, tools/gomad/internal/backends/wasm/wasi/helper_model.go, tools/gomad/internal/backends/backend.go, adjacent replay/inspection tests

## Description
Change `ChoiceRequest` to carry trace bytes, trace digest, expected tape digest and expected decision count plus a separate bounded typed prefix cutoff/final-rank control instead of an execution-path `ReplayPlan`. Preserve every existing ChoiceCapability.ReplayPlan producer across exploration, minimization and backend adapters while the helper owns projection/admission; exact mode has no controls. Do not duplicate the helper Decision type or force controller rank overrides into observed trace evidence. `choiceCapabilityForArtifact` reads the trace payload and manifest expectations and no longer calls `DecodeStoredTrace`/`ProjectReplayPlan`; `inspection.go`'s request builder does the same. Extend the helper exit terminal with `trace_sha256`, `records`, `branching_records`; in `collectHelper` take those from the terminal and drop the exit-path `DecodeStoredTrace`, keeping the count and digest checks. `choiceTraceDivergence` compares manifest values against the terminal. For storage crashes, require the helper's cursor, admitted count and complete next-unconsumed decision/divergence when the cursor is below the count. Validate bounds, tape identity and next-decision ordinal; preserve Expected and tape_unconsumed without indexing a Runner-side tape. An end cursor requires no next decision; an out-of-range cursor is invalid. `ProjectInterruptedReplayPlan` may still project observed interrupted evidence but cannot supply the missing expected input decision. Inspection, publication and minimization decoders are untouched.

## Acceptance
Fresh baseline and candidate executions of every runner replay corpus scenario, each replaying its own identity-matching artifacts, yield the same match/divergence string and status; storage-crash cases with partial consumption preserve the complete Expected decision and unconsumed divergence, end-of-tape cases have no next decision, and out-of-range/missing/inconsistent next-decision terminals refuse; `go test -tags test_dep ./internal/runner/... ./internal/backends/wasm/...` passes; no call to `ProjectReplayPlan` remains on the replay execution path (grep in a test). Historical helper/model identity mismatches remain refusals; no expected digest is frozen to bypass changed source.

## Done summary
R1/R4/R5/R6: Execution requests now pass raw trace/expectations with helper terminal digest/count/cursor/complete next-decision validation; evidence decoders and full interrupted divergence remain. Review discovered verify-only bypass;07ebfdb4ee restores semantic decode/project ONLY there, with resealed malformed trace/tape/count negatives and zero executor calls. RED proves all3 corruptions; focused replay suite46top-level/138events0, whole Runner239top-level/1128events0 with oneexisting skip, read-only lint0. Root reran Replay runner/... exit0. Existing407corpus terminal/full-divergence values unchanged. Same-receipt independent re-review SHIP; execution remains helper-owned.

Evidence index: /home/agent/wt/evidence/fn232/fn228-replay-inputs-2026-10-20/FINAL_CUSTODY.md; original producer handovers and narrow review-fix appends remain authenticated independently. Source acceptance only; spec closure requires separate completion review.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1ef7bd73b5, 0561e32514, 07ebfdb4ee
- Tests: root pinned Go test -p=1 -tags test_dep -run Replay ./internal/runner/... exit0, verify-only red1/green0; whole Runner package0; changed-source lint0 (retained receipts), 407-case complete terminal/divergence comparison0
- PRs: