---
satisfies: [R7, R9]
---
# fn-228-gomad-helper-owned-replay-tape.4 Reference session, corpora and docs: keep split-owned mode, regenerate transcripts, document the contract

Touches: tools/gomad/internal/backends/wasm/wasi/runtime.go, wasi/testdata/*, wasmhost/tests/testdata/helper_transcript/, tools/gomad/internal/backends/wasm/README.md, ARCHITECTURE.md, CLI.md

## Description
Verify the Go reference session still receives runtime ops under split-owned mode with the new request shape (it keeps its own tape from the Runner-side projection for the oracle only), regenerate the helper transcript corpus and any fixture whose config frame changed, run the two-pass byte-identity check, and update the WASM README replay section, ARCHITECTURE's evidence section and CLI.md to state that the helper owns tape projection and validation and the Runner compares terminal counts. Record the helper identity rotation in the qualification ledger per the qualification contract.

## Acceptance
Split-owned differential tests pass; transcript corpus regenerates byte-identically on the second pass; docs describe the new ownership; the rotated helper digest is recorded with the frozen-candidate qualification required by QUALIFICATION.md.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
