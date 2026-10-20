---
satisfies: [R7, R9]
---
# fn-228-gomad-helper-owned-replay-tape.4 Reference session, corpora and docs: keep split-owned mode, regenerate transcripts, document the contract

**Size:** M
**Touches:** tools/gomad/internal/backends/wasm/wasi/runtime.go, tools/gomad/internal/backends/wasm/wasi/testdata/**, tools/gomad/internal/backends/wasm/wasmhost/tests/testdata/helper_transcript/**, tools/gomad/internal/backends/wasm/README.md, tools/gomad/ARCHITECTURE.md, tools/gomad/CLI.md, the existing fn-219 identity ledger and fn-232 task .13 qualification-transfer record

## Description
Verify the Go reference session still receives runtime ops under split-owned mode with the new request shape (it keeps its own tape from the Runner-side projection for the oracle only), regenerate the helper transcript corpus and any fixture whose config frame changed, run the two-pass byte-identity check, and update the WASM README replay section, ARCHITECTURE's evidence section and CLI.md to state that the helper owns tape projection and validation and the Runner compares terminal counts. Record all actual helper source/binary and WASI/model identity changes and their input causes in the existing identity ledger. Preserve the frozen pre-fn-228 Go baseline and matching execution inputs under fn-232 ownership for its fresh-equivalence harness. Record the outstanding final runtime/workload/replay qualification on the receiving fn-232 R12/task .13 before closing this task; no full Go-host qualification cycle is added to this prerequisite batch.

## Acceptance
Split-owned differential tests pass with exact, shortened-prefix and forced-rank controls on standalone and cluster paths; transcript corpus regenerates byte-identically on the second pass; docs describe the new ownership; actual helper/model rotations and unchanged trace/tape semantics are recorded; incompatible identities refuse; baseline references remain usable; the explicit receiving fn-232 R12/task .13 qualification obligation is recorded as pending rather than passed.

## Done summary
R7/R9: Split-owned lazy Go reference oracle remains for standalone/cluster exact/prefix/rank comparisons; transcripts regenerate byte-identically. Helper/WASI rotations and native transitive causes are measured separately in tracked replay ledger. Fresh old/new own-artifact seed7R7 comparison retained; zero-choice limitation explicit, separate nonzero oracle59corpus does not fabricate campaign coverage. Independent fn232 custody preserves source/executables/manifests/rawR7/combinedreceipts; final Rust runtime/workload/replay qualification explicitly remains pending232R12/.13 under3600s. Review fixes preserve helper/model identities; no artifact migration. Independent combined source re-review SHIP07ebfdb4ee.

Evidence index: /home/agent/wt/evidence/fn232/fn228-replay-inputs-2026-10-20/FINAL_CUSTODY.md; original producer handovers and narrow review-fix appends remain authenticated independently. Source acceptance only; spec closure requires separate completion review.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1ef7bd73b5, 0561e32514, 95caeab345, 07ebfdb4ee, eee312b9a0
- Tests: split-owned oracle comparisons; two-pass transcript corpus0, fresh own-lineage R7 install0/campaign1targetfailure/replay1reproduced; scoped limitations retained, receiver initial+combined manifests verified0; combined source gates0; review SHIP
- PRs: