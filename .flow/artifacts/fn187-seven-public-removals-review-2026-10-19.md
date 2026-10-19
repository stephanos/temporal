# Seven public-root removals: periodic source audit

Scoped delta: archived preimages in
`/home/agent/wt/evidence/fn187/r3-current/remaining-public-deletion-named-entry-2026-10-19/preimages/files`
to the four current files in `/home/agent/wt/wasm-fn187-functional`.
These reports are source-only periodic audits, not task-wide implementation
review verdicts or substitutes for final five-mode conservation. Reports below
are retained verbatim. Both were dispatched as fresh `quality-auditor` contexts;
the routing judge was unavailable (`no_key`).

### Correctness axis

## Quality Audit — Correctness axis: seven public test deletions

Files changed: 4 · Critical 0 · Should Fix 0 · Consider 0. No correctness findings.

- **Executed:** static comparisons confirm exactly 295 deleted lines; every retained byte matches the archived preimages. All guarded source hashes match. All 142 private binding keys/reasons and corresponding functions remain.
- **Executed:** cluster wire exactly matches the original 8,388,706 bytes; storage payload exactly matches all 1,259,101 bytes, 400 operations, 20 ordered chunks and final end frame.
- **Walked:** `tools/gomad/internal/backends/wasm/wasmhost/src/cluster/transport_input_corpus.rs:38` preserves shared-stream/fresh-reader ordering, capacity classification, unpoisoned transport and exact next-frame bytes; its six controls reject changed observations/input.
- **Walked:** `tools/gomad/internal/backends/wasm/wasmhost/src/storage_transport/vector_corpus.rs:41` preserves descriptor validation, maximum 1 refusal, missing-end refusal, original maximum 18,892,800, successful completion and count 400; six controls exercise these distinctions.
- **Walked:** `tools/gomad/internal/backends/wasm/wasmhost/tests/common/helper_feature_corpus.rs:231` and `:296` retain all 11 feature cases’ original assertions, including positive raw timings, exact counts/transcripts, fuel/configuration and pre-start rejection. Raw-profile and LLVM diagnostic controls remain.
- **Walked:** `tools/gomad/internal/backends/wasm/wasmhost/src/cluster/mod.rs:331` refuses diagnostic clusters unconditionally after validation; both original and replacement limits satisfy validation. Retained host/private tests are unchanged. Reversing `$hot_entry` reproduces the recorded original hash.

No builds, helper execution or tests run. Final feature conservation remains root-owned; no task-wide shipping verdict issued. Same-family writer/reviewer disclosed.

### Standards axis

## Quality Audit — Standards axis: seven public-root removals

### Summary

- Files changed: 4 · Should Fix 0 · Consider 0 · Blocking: none possible (standards axis)
- No standards findings in the scoped deletion or immediate fixture/classification support.

### What's Good

- **Executed:** archived-preimage comparisons show exactly 295 deleted lines and zero additions. Current file hashes and all 31 unchanged guarded inputs match the packet; no new lint exemptions.
- **Executed:** every selected functional root maps to corpus, invariants, or machinery across all five modes. All 142 original private binding keys and reasons remain unchanged.

### Limits

Source-only audit; no builds, tests, runtime commands, or repository mutations. Final all-feature conservation remains root-owned. Reviewer and writer are from the same model family; the alternate judge was unavailable (`no_key`).

Correctness: 0 findings; worst tier none.
Standards: 0 findings; worst tier none.
