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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
