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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
