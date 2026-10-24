---
satisfies: [R1, R3]
---
# fn-247-one-language-for-the-framework-a-rust.4 Replace the extracted supervisor and process controls with allocation-free Rust ownership

## Description
Replace the extracted supervisor and all process controls with Rust (R1, R3). This is a serial successor to existing .1 and a prerequisite of existing .2.

**Size:** M
**Files:** `model/umpire/backends/process/{Cargo.toml,src/**,tests/**}`, `model/Cargo.toml`, `model/Cargo.lock`, `model/scripts/tests/test_backend_process.py`, backend consumer tests.
**Touches:** [model/umpire/backends/process/**, model/Cargo.toml, model/Cargo.lock, model/scripts/tests/test_backend_process.py, model/umpire/backends/tlc/**, model/umpire/backends/apalache/**]

### Approach

- Follow the parent spec's in-process supervisor design. Audit every post-fork branch, including error, signal and cleanup branches, against the allocation/lock/unwind prohibition; prepare executable PATH resolution, argv/env and all `/proc` parsing storage in the parent. Use `execve` with prepared bytes for the target, and `_exit` for every forked exit.
- Promote only pinned libc 0.2.191. Mirror the established runtime crate-local lint exception, maintaining other lint values and workspace forbid; isolate the unsafe allowance to the process syscall module and forbid it elsewhere.
- Preserve Python supervisor outcomes from the recorded corpus. Implement Linux subreaper/adopted-escaped-grandchild cleanup and Darwin prevention, retaining direct/private group ownership and the EOF channel. Do not promise recovery from SIGKILL of the supervisor itself.
- Port each Python process unittest and every Rust Python-generated child to Rust fixture modes. Include real owner-group TERM/KILL, supervisor TERM, cancellation, timeout, owner death and escaped grandchild PID disappearance before harness fallback cleanup. Retain safe-output, bounded dump/read, scratch and Java controls.
- Add a concurrent parent allocator/thread stress control and a cold focused test/consumer control showing no extra executable bootstrap or runtime Cargo is needed. Audit syscall failures as refusals/statuses, not panics. Delete the old process Python only after its counterpart ledger is complete.

### Investigation targets

**Required**:
- `model/umpire/backends/tla/src/process.py:12` — original platform/signal/cleanup contract (stable source baseline).
- `model/umpire/backends/tla/src/process.rs:198` — execution/capture contract, moved by .1.
- `model/scripts/tests/test_backend_process.py:23` — all old process controls.
- `model/umpire/protocol/runtime/Cargo.toml:24` — narrow lint override precedent.
- `model/Cargo.toml:104` — workspace forbid and lint parity.
**Optional**:
- `model/umpire/backends/tlc/src/runner.rs:771` — boundary/limit controls.

### Quick commands

```bash
cd model && mise exec -- cargo test -p umpire-backend-process --locked
cd model && mise exec -- cargo clippy -p umpire-backend-process --all-targets -- -D warnings
```

Run actual Linux lifecycle controls on Linux; run Darwin policy/native controls on the available platform and distinguish simulated coverage from native evidence. No full gate here.

## Acceptance
- [ ] Cold focused process tests and a minimal direct dependency consumer work without a prebuilt sibling binary; fork-path safety audit and multithreaded allocator stress control pass.
- [ ] Every old process test has a named Rust counterpart; actual Linux direct/escaped descendants are gone before fallback cleanup in timeout/cancel/owner-death/group-signal controls; status/output corpus matches.
- [ ] Output/inode/FIFO/symlink/dump/UTF-8/scratch/Java controls still pass; unsupported-platform and Darwin UID/eUID prevention failures are explicit.
- [ ] Workspace forbid is unchanged; only the process syscall module admits unsafe, with documented invariants; libc is exactly existing 0.2.191 and no other third-party dependency changes.
- [ ] Extracted process Python and Python child fixtures are removed after parity proof; unavailable native Darwin evidence is stated.


## Done summary
Replaced the extracted Python supervisor with in-process Rust ownership, preserving functional process execution and the task's R1 safety controls. Only the audited syscall subtree admits unsafe; the safe capture/scratch modules forbid it, workspace forbid is unchanged, and the sole dependency promotion is existing libc =0.2.191. All 14 original process controls have named Rust counterparts; both process Python files were removed after those controls passed.

baseline: green (8 Rust process tests, 6 Python process tests and scoped Clippy before edits). Final focused checks passed 25 process tests, 21 TLC/Apalache unit tests and one real required control per backend: 48 distinct passing identities. Linux direct/escaped descendants disappear before fallback cleanup for timeout, cancellation, owner death and group signals. The stress control runs 48 supervised execs alongside four allocator threads and a concurrent std::process spawn thread. A fresh lazy-linked direct dependency consumer runs with no runtime tools or sibling helper; focused Cargo tests require no prebuilt supervisor. Source hashes still match the passing reviewed source. The classifier returned FULL/unmatched for externally owned MILESTONES.md; task .4 defines no full gate, so no full receipt or fabricated skip was recorded, and no tests were repeated after SHIP. Final task .3 retains its combined required validation pipeline.

The primary three codex:gpt-6.1-sol:high draws returned NEEDS_WORK, reduced to two introduced findings. PATH execution now tries all parent-prepared candidates; raw clone handles s390x argument order and SPARC child identity. Their regression controls failed first, then passed. The single resumed primary review returned SHIP with command exit 0. Reviewers are from the same model family as the writer; read-only Cargo build-lock restrictions prevented their independent reruns, so they checked source hashes and retained passing evidence. Dispatcher and finalizer both exited 0. Two distinct nontrivial bug memories were captured.

Native evidence is Linux/aarch64, glibc 2.43, uid/euid 1000. Darwin prevention is covered by the actual shared Rust NPROC helper on Linux plus pinned source; no native macOS/SDK link, s390x or SPARC execution is claimed. Darwin private syscall entries have source/export evidence without an ABI stability guarantee. Forked cleanup/error paths use resolved cancellation-free interfaces, complete signal preparation and _exit; no child allocation, locks, formatting, general Rust helpers or unwinding is admitted. Java heap configuration is not an OS memory cap; regular-file output polling is not a strict kernel cap; supervisor-SIGKILL recovery is not promised.

Durable source, commands, exits, passing/failed logs, cold consumer, all 14 mappings, platform/ABI audits and review receipts are in .flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task4/. The initial raw-fork cold run deadlocked in inherited glibc state and exited 101 after exact task-owned supervisor cleanup; this failure, both libc-fork callback mutants, review regressions and intermediate tooling/source-fetch failures are retained honestly. The owner waived old-byte/backward-compatibility obligations during this task; prior corpus/hash passes are historical evidence, not current completion gates. No Model/feature/protocol/Go/guest/golden meaning or bytes were changed by this worker.

The fixed pre-edit base remains 60c65fcb2922e953a558541343c65a4998519e2f. Source checkpoints include 66151d1377, f4fa213d12 and reviewed fix 64b41a2692. Root's independent metadata commits and other externally owned plan edits in this range are not this implementation. Unrelated dirty paths were unstaged and preserved before task commits; no remote command was issued by this worker. Both owned worktrees/branches were removed normally; unknown external worktrees/caches are preserved. The in-process supervision route replaces Python; final task .3 owns the remaining ownership-map/prerequisite prose.

stage: impl-review - ran (three NEEDS_WORK draws, two deduplicated findings, single resumed primary SHIP; task4/review-receipt.json)
stage: memory-capture - ran (PATH candidate fallback; raw clone argument/return ABIs)
stage: plan-sync - skipped(config: planSync.enabled != true)
Tier: session (jev-unavailable(no_key)); AGENTS implementer pin applies. Judgedonce .flow/tmp/fn2474-judge.json; no keyretry/reroute.
## Evidence
- Commits: 02d91ccc35b80721b2e02706cdd2d8721cd2ce2a, 5e8c888bc60fc3daa755d7346691e460502ea933, 2057ee923e1f929dc40ad12fab7d57648fc8ac4e, 722b92cf18a8af80f2784250ddc8d991348b28ad, 7701881614bec7395b441fdf1e47490629923202, 2c9a6959086b4ac4ce736fe3d14f276705bff6ab, 00b148e86946316ee57cd8f1c30e169fd876202b, 8f529d5c81668748cc0d428cb69a796dc068938a, f3d7b1e8a140bb8a70f0656846e41bd00593bcd2, 2e89358c25140b0733cd74ef60d3e80cdb480a28, a68ca184900bef314caf2bc87b2bbf27377d9e92, 93c81ec8b98a64d3f71c921da595c205cf226cb7, 50f16e1779a0c8fe2eebb13f422f6426d73184c6, 7ea726fd96787b5cfde457c2a49f92929686987e, c87df4c9ae92a033c0326c0dd19982c02c78e654, c1f77e52bc6939ffe2c0936d4aa1c67d2e948605, 4243fcb9e9a0145f64980db5c6405286ef1e5f47, 66151d1377408f16c37ddaf6f86dc13bbab9f434, f4fa213d12f248819b3bc77214f9137160bb0682, 37aa1cc6f542836b41c246f04cc95dc7f4d08b93, 4e4acd3f4bb96552b619e8765e7417c9ea8be7ce, 64b41a26928ca891b7c955cdaf429c424f8a4918, 156b4b858b726520cbfee28351c50a79078d9b9f
- Tests: cd model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2474-reviewed-target mise exec -- cargo test -p umpire-backend-process --locked, cd model && mise exec -- cargo clippy -p umpire-backend-process --all-targets -- -D warnings, cd model && mise exec -- cargo test -p umpire-tlc -p umpire-apalache --lib --locked, cd model && UMPIRE_BACKENDS=require mise exec -- cargo test -p umpire-tlc --test controls real_tlc_initial_invariant_has_a_typed_counterexample --locked -- --exact, cd model && UMPIRE_APALACHE=require mise exec -- cargo test -p umpire-apalache --test controls real_evaluated_start_failure_cannot_disappear_into_an_empty_range --locked -- --exact, BASH_ENV=/dev/null bash .flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task4/run-cold-consumer.sh (exit 0; prior observed cold consumer)
- PRs: