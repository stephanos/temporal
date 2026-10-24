---
satisfies: [R1, R4]
---
# fn-247-one-language-for-the-framework-a-rust.1 Extract shared process support and retain the post-predecessor equivalence baseline

## Description

Extract process support and establish the post-predecessor equivalence baseline (R1). Keep its existing supervisor until the appended Rust-supervisor task replaces it. This bounds import/ownership changes separately from fork-safety work.

**Size:** M
**Files:** `model/umpire/backends/{process,tla,tlc,apalache}`, `model/Cargo.toml`, `model/Cargo.lock`, `model/scripts/tests/test_backend_process.py`, `model/README.md`, `.plans/UMPIRE_MODULES.md`.
**Touches:** [model/umpire/backends/process/**, model/umpire/backends/tla/**, model/umpire/backends/tlc/**, model/umpire/backends/apalache/**, model/Cargo.toml, model/Cargo.lock, model/scripts/tests/test_backend_process.py, model/README.md, .plans/UMPIRE_MODULES.md]

## Approach

- Admission first: verify committed fn-237.3 implementation and Flow closure ancestors; record integrated source commit and exact current guest/golden/original hashes under task-owned evidence. Reanchor this plan's `94e47d7878` source references to that tip. Never use a pre-.3 guest baseline.
- Record deterministic process/API control outputs and a one-to-one Python/Rust-control inventory before changing owners. This recorded-invocation corpus is the refactor equivalence harness; source tests alone are not an observed baseline.
- Move the existing process module and supervisor without algorithm changes, update all imports/reexports and direct workspace dependencies, preserving the caller contracts in the parent spec. Adjust the existing Python test import until the next task ports it.
- Update process/TLA ownership in both active maps. Keep Cargo/lock edits in this serial owner; existing package coordinates/checksums cannot change.

## Investigation targets

**Required** (reanchor before coding):
- `model/umpire/backends/tla/src/process.rs:18` — public items, failures, capture and lifecycle.
- `model/umpire/backends/tla/src/lib.rs:10` — reexports.
- `model/umpire/backends/tlc/src/runner.rs:17` — shared imports and admission.
- `model/umpire/backends/apalache/src/lib.rs:15` — separate consumer.
- `model/scripts/tests/test_backend_process.py:23` — live control ownership.
**Optional**:
- `.plans/UMPIRE_MODULES.md:25` — current ownership map.

## Quick commands

```bash
cd model && mise exec -- cargo test -p umpire-backend-process -p umpire-tla -p umpire-tlc -p umpire-apalache --locked
```

The retained Python process suite runs at this intermediate checkpoint using its existing discovery entrypoint; record that result before deleting it in the supervisor task. Do not run a full model gate here.

## Acceptance

- [ ] Committed predecessor admission and exact post-.3 baseline evidence recorded before source changes; the named tests/control corpus has old outputs/statuses and hashes.
- [ ] Focused backend tests pass after the extraction; public names/signatures, compatibility reexports, error text and deterministic corpus compare equal.
- [ ] Process support has one owner; TLA+ retains only its emitter/admission work; Apalache has no TLC dependency; both active ownership maps match.
- [ ] Cargo/lock graph changes only add the local process crate and consumer edges; Python remains explicitly temporary until its replacement passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
