---
satisfies: [R6]
---
# fn-247-one-language-for-the-framework-a-rust.9 Adopt rustix and tempfile in the process crate

## Description
Adopt the R18-admitted `tempfile` 3.27 and `rustix` 1.1 in `umpire-backend-process` to shrink its unsafe surface (R6), after .4 settles the supervisor. Behavior is unchanged.

**Size:** S
**Files:** `model/umpire/backends/process/{Cargo.toml,src/support.rs,src/syscall.rs,src/syscall/**}`, `model/{Cargo.toml,Cargo.lock}`.
**Touches:** [model/umpire/backends/process/**, model/umpire/backends/tlc/**, model/umpire/backends/apalache/**, model/Cargo.toml, model/Cargo.lock]

### Approach

- Native Scratch ownership may use a path accessor; mechanically migrate affected TLC/Apalache callers if needed instead of preserving a tuple layout solely for backward compatibility. Keep this adaptation narrow and retain all functional process controls.

- Shared Cargo.toml/Cargo.lock mutations have one conductor-assigned owner. Task .9 may implement concurrently with .2, .5 or .6 in an isolated checkout with its own Cargo files and build target. The conductor serializes reconciliation of Cargo edits in the integration checkout and gathers combined evidence before completion; accept ordinary merge/rework risk rather than serializing whole tasks. No extra dependency edge or full gate is required.

- First record the audited unsafe site count (`unsafe` blocks/fns in `src/**`) and the before/after package/feature inventory; both crates are already resolved, so no new resolution is expected. Request only needed features (`rustix`: process/fs/system as used) and keep them out of guest closures.
- `support.rs`: create scratch directories with `tempfile::Builder` (prefix, 0700 permissions, `tempdir_in`, explicit `close()` so cleanup failure stays observable). The symlink-refusing workspace-relative walk stays hand-written.
- `syscall.rs`, parent side only: replace raw libc with rustix safe wrappers for `fcntl` dupfd-cloexec, `getrlimit`/close limit, `waitpid` in `try_wait`/`Drop`/owned-child reaping, `getuid`/`geteuid`, Linux `prctl` subreaper, and similar calls where a wrapper exists. Keep raw libc where rustix has no wrapper and record why.
- The forked supervisor and pre-exec child stay raw libc through the existing audited call table: no rustix, allocation, locking or destructors after fork.
- Keep crate-local `unsafe_code=deny` confined to the syscall module and workspace `forbid` everywhere else.

### Investigation targets

**Required**:
- `model/umpire/backends/process/src/support.rs:52` — scratch creation and workspace walk.
- `model/umpire/backends/process/src/syscall.rs:102` — parent `try_wait`/dupfd/rlimit/subreaper/reap calls.
- `model/umpire/backends/process/src/syscall/calls.rs` — child-side audited call table (unchanged).

### Quick commands

```bash
cd model && mise exec -- cargo test -p umpire-backend-process --locked
cd model && mise exec -- cargo clippy -p umpire-backend-process --all-targets -- -D warnings
```

## Acceptance
- [ ] Scratch directories use `tempfile::Builder` (prefix, 0700, `tempdir_in`, explicit close); the symlink-refusing walk and scratch cleanup-failure reporting are unchanged.
- [ ] Parent-side syscalls use rustix safe wrappers wherever one exists; remaining raw libc calls are listed with reasons; the forked child path remains raw libc.
- [ ] Recorded unsafe site count is lower after than before; unsafe stays confined to the syscall module and no other crate's lint policy weakens.
- [ ] Every existing process control (refusal, launch, timeout, cancellation, signal, owner EOF, Linux descendant cleanup, Darwin admission, scratch cleanup) passes with the same outcome; before/after inventory shows no new resolved package and no guest-closure change.



## Preparation evidence

Read-only research: `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/planning/parent-safe-adoption-research.md`. Use its parent/child boundary and exact resolved-API cautions; the lexical unsafe count is a reproducible source baseline, not an expanded-code or runtime proof. Existing guest build-tool reachability of rustix/tempfile is not a new guest runtime dependency; distinguish target/build contexts in closure accounting. No extra full gate or old-byte comparison is introduced.

## Done summary
Adopted already resolved rustix1.1.5 and tempfile3.27.0 in the process crate: Scratch privately owns TempDir, creates0700 run- directories and explicitly closes with observable cleanup/prior-result errors; parent identity/dupfd/limits/waits and fixture ownership calls use safe wrappers. TLC/Apalache path-access migrations are mechanical. Raw child production code and audited call table remain byte-identical; workspace forbid and syscall-only unsafe allowance are unchanged.

baseline: green (25 process tests and scoped process Clippy). Final isolated checks passed29 process+21 backend library tests and2 actual pinned backend controls; process/backend Clippy and scoped format passed. Root's source-bound joined checks passed75 tests (54 xtask/process+21 backend) and four-package all-target Clippy with exit0. Exact commands, logs and source hashes are committed under task9; hashes still match after review. The final Verify classifier returned FULL/unmatched for conductor-owned MILESTONES.md; task9 has focused gates and no full Model gate. Reused unchanged-source observations without extra full run, fabricated skip or receipt. Task3 retains the final combined required pipeline.

Unsafe accounting falls77→67 through10 actual parent-block removals; no grouping/moving/count reclassification. Package inventory stays145→145 with no new identities; host process/prctl feature additions are recorded. Six guest normal topologies remain identical; target-specific wasm trees exclude the process crate/feature. Existing host-build tempfile/rustix reachability is distinguished from runtime closure. Native evidence is Linux/aarch64/glibc2.43 uid/euid1000; no native Darwin/s390x/SPARC proof is claimed.

Three fresh read-only codex:gpt-6.1-sol:high draws returned SHIP with no findings or uncovered R6 obligations; backend finalizer exited0 with SHIP. Reviewers are the same model family as the writer under the explicit pin. Their attempted test reruns hit the read-only Cargo lock, so they checked committed source/log hashes and evidence; independent executions are not claimed. No review source fixes or nontrivial fix-memory capture were needed. Durable audit/commands/inventories and review provenance: .flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task9/; review-receipt.json and review/coordinator.md identify the exact reviewed scope.

Original source checkpoint9723353bcb was integrated by root as2b13fc6d29. Review base75f7749e6e excludes task2 implementation. Root's later admission/evidence bookkeeping in this range is not task9 source. No Model/DSL/protocol/guest meaning changed, no MILESTONES edit or remote operation was made by this worker. The process route changes Scratch callers to path(); feature-map work remains conductor-owned.

stage: impl-review - ran (three first-round SHIP draws, backend finalizer SHIP; task9/review-receipt.json)
Tier: session (jev-unavailable(no_key))

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9723353bcb521a2b0941d5acf9b573107fc5756c, 2b13fc6d29fa20d0742606b239b68b0d87fbd8c2, 2c3c308ee27d2920063dcd29777c5ef084484381
- Tests: cd model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2479-target mise exec -- cargo test -p umpire-backend-process --locked, cd model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2479-target mise exec -- cargo clippy -p umpire-backend-process --all-targets -- -D warnings, cd model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2479-target mise exec -- cargo test -p umpire-tlc -p umpire-apalache --lib --locked, cd model && MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false UMPIRE_BACKENDS=require CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2479-target mise exec -- cargo test -p umpire-tlc --test controls --locked real_tlc_initial_invariant_has_a_typed_counterexample -- --exact, cd model && MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false UMPIRE_APALACHE=require CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2479-target mise exec -- cargo test -p umpire-apalache --test controls --locked real_postcondition_filter_excludes_unrelated_actions -- --exact, cd model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2479-target mise exec -- cargo clippy -p umpire-tlc -p umpire-apalache --all-targets -- -D warnings, cd model && mise exec -- cargo fmt -p umpire-backend-process -p umpire-tlc -p umpire-apalache -- --check, cd model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2479-target mise exec -- cargo metadata --locked --offline --format-version 1, cd /Users/stephan/Workspace/skunkworks/umpire/temporal/model && BASH_ENV=/dev/null CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2479-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo test -p xtask -p umpire-backend-process --locked, cd /Users/stephan/Workspace/skunkworks/umpire/temporal/model && BASH_ENV=/dev/null CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2479-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo test -p umpire-tlc -p umpire-apalache --lib --locked, cd /Users/stephan/Workspace/skunkworks/umpire/temporal/model && BASH_ENV=/dev/null CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2479-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo clippy -p xtask -p umpire-backend-process -p umpire-tlc -p umpire-apalache --all-targets --locked -- -D warnings
- PRs: