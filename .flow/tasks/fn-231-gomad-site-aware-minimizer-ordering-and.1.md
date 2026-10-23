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
Implemented R1/R2/R3/R4/R7/R8: authenticated producing-round scope inheritance and detached retention, stable out/mixed/in proposal ordering, pinned checkpoint/context refusals, unchanged prefix proposals and full failure predicate, canonical evidence and execution/failure identities.

On immutable a07dd6e2c404ce9120c448e8568c18906fa2f9aebd87d11ce8d53a7bac8a861b from 8536a84a85, both public scope runs passed 55 commands and both fresh Go/Rust semantic lanes passed 119 commands. Each retained unchanged inputs/binary/source bindings and original budgets. The two generated semantic goldens are byte-identical (3095bc851f9fa9f821dac792e585f16228975a0ebd0d8dd0a0cafeaedf05f61e). Scope02 used driver 9808f8807c after an unrelated documentation-only commit; all 4570 compiled tools inputs were independently identical. Its earlier pre-execution HEAD refusal is preserved.

Integrated default156/fixture158 tests (+1 existing ignored each), both strict all-target Clippy and fmt passed. Genuine-parent provenance RED/GREEN and exact minimized replay prove the narrow source correction; fresh independent review SHIP, same GPT family. Full R-ID evidence is in .flow/artifacts/fn231-rust-minimizer-handover-2026-10-22.md; raw comparison: /home/agent/wt/evidence/fn231-public-provenancefix-02/two-run-comparison.json. Task2 golden field closeout and spec completion review remain separate; no final Rust qualification/retirement claim.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 2ac50dcabf56d3e51ddce9eb1a4d299628551ee4, 33a4ceea3af675e009eb224384fb03c5d6f8ad5b, faf8fd6a9fd9a736c9ab56b72d07a0a53cc4a06a, 8536a84a8595b0f5e27295b93885908bd65f07bb
- Tests: cargo +1.99.1 test --release --locked --offline --manifest-path tools/gomad/rust/Cargo.toml -p gomad --lib (156 passed), cargo +1.99.1 test --release --locked --offline --manifest-path tools/gomad/rust/Cargo.toml -p gomad --features determinism-fixture --lib (158 passed), default and determinism-fixture all-target strict Clippy; cargo fmt --all -- --check, /home/agent/wt/evidence/fn231-public-provenancefix-01/run-phase.py scope01 (55) and semantic01 (119): terminal 0, /home/agent/wt/evidence/fn231-public-provenancefix-02/run-phase.py scope02 (55) and semantic02 (119): terminal 0, byte-identical generated golden comparison SHA3095bc851f9fa9f821dac792e585f16228975a0ebd0d8dd0a0cafeaedf05f61e
- PRs: