---
satisfies: [R1, R2, R3, R4, R7, R8]
---
# fn-231-gomad-site-aware-minimizer-ordering-and.1 Scoped proposal ordering with checkpoint key and parent-scope default

**Touches:** tools/gomad/rust/gomad/src/campaign/**, tools/gomad/rust/gomad/src/artifact/**, tools/gomad/rust/gomad/src/evidence/**, tools/gomad/rust/gomad/src/minimize/**, tools/gomad/rust/gomad/src/cli/minimize.rs, tools/gomad/testdata/parity/**
**Semantic references only:** tools/gomad/internal/runner/internal/minimizer/{minimizer.go,choice.go,workspace.go}, runner/minimize_operation.go, cli/minimize.go

## Description
Implement once on the Rust host after fn-232.9 and the existing site-spec predecessors; the Go locations below document existing semantics only. Port the behavior into the Rust ownership surface above and extend the shared golden/behavioral corpus. Do not create or extend a Go-host implementation.
Accept `--site-scope` on minimize with the explore validator; default to the parent artifact's recorded scope when absent. Build the fingerprint→symbol map from the parent's retained module once and resolve the scope. In `proposals`, apply a stable sort key: all-out-of-scope first, mixed, then in-scope only, preserving existing order within a class; the choice-prefix minimizer is unchanged. Add the ordering key (patterns, resolved digest) to persisted state and the implementation digest so old state and changed keys are refused as today. Add a test with a fixed reduction set showing that both final candidates satisfy the same acceptance predicate, with a fixture where attempt counts differ. Do not require identical final candidates from different greedy proposal orders.

## Acceptance
Implement the parent authenticated search-context payload during Rust exploration retention and consume it through the shared opener. Pin the producing-round snapshot and parent-specific resolution before attempts. Test attached/detached artifacts, removed campaign context, legacy absent metadata, malformed/tampered present metadata, round-to-round resolution changes, explicit override and resume. Never guess provenance from artifact path or use a mutable final campaign summary.

Invalid scope exits 2 before any attempt; parent scope is used by default; the ordering test passes; resume with a changed key is refused with status 3; without any scope the attempt sequence is byte-identical to before. Execution/failure identities and the minimization evidence schema remain unchanged; the ordering/checkpoint key and implementation digest change deliberately, and legacy checkpoint state is refused.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
