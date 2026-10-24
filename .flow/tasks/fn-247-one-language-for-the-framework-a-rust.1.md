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
Backend consumers now share `umpire-backend-process`, preserving the existing implementation and TLA/TLC compatibility exports. Both ownership maps identify the new owner and explicitly retain the Python supervisor until task 4.

Admission, all 752 artifact hashes, 48 retained control mappings, package/features and observed API corpus evidence are in `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task1/`. Both moved sources compare byte-for-byte with `cf339e835809cca4ce3cc06a9dda5d66948e02eb`; guest SHA-256 remains `eaf6efd637e0e33b1a9f1e02b24cd8595c1f73706b5695fcde86151e884515f0`. The predecessor's canonical status is done, and its implementation/review/closure commits are ancestors. Cargo adds only the local crate, its existing thiserror dependency and three consumer edges; all existing packages, versions, checksums and feature sets are unchanged.

baseline: green (159 focused Rust tests and 6 Python process controls before edits). An initial focused launch lost its host-result handle and was inconclusive; one observed rerun supplied exit 0. The post-change exact task Quick command passed the same 159 Rust test identities, including 8 controls in the new process owner. The retained Python discovery suite passed all 6 controls. Format, scoped Clippy and scoped rustdoc with warnings denied passed. Compatibility and direct-owner invocation stdout/stderr compare exactly against the recorded pre-edit corpus. Verification used those observed post-change exits at the reviewed source commit and wrote a green focused unittest receipt; no source changes followed the passes.

Three explicit `codex:gpt-6.1-sol:high` draws returned SHIP; dispatcher and finalizer each exited 0. Reviewers are from the same model family as the writer. Their Cargo reruns were blocked by read-only build-lock permissions; they verified the retained passing logs, source equality and hash/corpus evidence. Linux descendant controls ran; native macOS execution is unclaimed. Full Model/backend/live gates and timing remain assigned to the spec's final task. Rust supervisor and Python removal remain assigned to task 4. No source worktree or remote operation was created.

stage: impl-review - ran (three SHIP draws; finalizer SHIP; receipt retained in task1/review-receipt.json)
Tier: session (jev-unavailable(no_key)); AGENTS implementer pin applies. Judge once already recorded .flow/tmp/fn2471-judge.json; no key retry/reroute.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4615afe62d8795f12c04b0cbc8e5522b23580678
- Tests: cd model && mise exec -- cargo test -p umpire-backend-process -p umpire-tla -p umpire-tlc -p umpire-apalache --locked, python3 -m unittest discover -s model/scripts/tests -p 'test_backend_process.py' -v, cd model && mise exec -- cargo fmt --all --check, cd model && mise exec -- cargo clippy -p umpire-backend-process -p umpire-tla -p umpire-tlc -p umpire-apalache --all-targets --locked -- -D warnings, cd model && RUSTDOCFLAGS="-D warnings" mise exec -- cargo doc -p umpire-backend-process -p umpire-tla -p umpire-tlc -p umpire-apalache --no-deps --locked, python3 .flow/tmp/fn2471/verify-extraction.py, python3 .flow/tmp/fn2471/run-corpus.py post; cmp .flow/tmp/fn2471/baseline-corpus.stdout .flow/tmp/fn2471/post-corpus.stdout; cmp .flow/tmp/fn2471/baseline-corpus.stderr .flow/tmp/fn2471/post-corpus.stderr, python3 .flow/tmp/fn2471/run-corpus.py post direct; cmp .flow/tmp/fn2471/baseline-corpus.stdout .flow/tmp/fn2471/post-direct-corpus.stdout; cmp .flow/tmp/fn2471/baseline-corpus.stderr .flow/tmp/fn2471/post-direct-corpus.stderr
- PRs: