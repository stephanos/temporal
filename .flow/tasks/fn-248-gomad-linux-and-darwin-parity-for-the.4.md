---
satisfies: [R5]
---
# fn-248-gomad-linux-and-darwin-parity-for-the.4 Portable cycle supervisor on libc with per-OS custody backend

## Description
Rewrite `cycle/process/sys.rs` declarations onto `libc` constants and types (`PR_SET_CHILD_SUBREAPER`, `F_SETFD`/`FD_CLOEXEC`, `sigaction`, `sigset_t`, errno), keeping the fork-to-exec no-allocation rule. Split the custody backend: Linux keeps the subreaper, `close_range` and `/proc/thread-self/children` / `/proc/{pid}/task` enumeration; Darwin uses a process group per leaf, `proc_listchildpids`/`proc_pidinfo` for descendants and fd closing by loop. Document the Darwin setsid-escape limit in `ARCHITECTURE.md`. Owner decision (2026-10-24): the Darwin setsid-escape limit is acceptable for development runs; dispatch on Darwin, do not refuse.
## Acceptance
- [ ] Existing custody, cleanup, spawn-ambiguity and accounting tests pass unchanged on Linux and on Darwin, except listed subreaper-dependent tests.
- [ ] No Linux custody guarantee weakened; no hand-written C declaration remains in `cycle/process`.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
