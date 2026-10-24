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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
