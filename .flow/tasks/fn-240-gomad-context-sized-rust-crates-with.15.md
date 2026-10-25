# fn-240-gomad-context-sized-rust-crates-with.15 gomad-process: one shared custody crate for cycle, launcher, stress, backstop and measurement

## Description
One shared process crate replaces five separate implementations.

Today (2026-10-25), each of these spawns, times out, kills and waits on its own, and none uses another's code:
- `cycle/process` (custody supervisor: a fork-time syscall-only child, process group/session, descendant reaping via `waitid`, `prctl`)
- `campaign/launcher.rs` (577 lines)
- `stress/process.rs` (268)
- `backstop.rs` (161)
- `measurement/cancellation.rs` (105)
Only the cycle reaps descendants, so the other four can leak grandchildren on timeout.

Do:
1. Amend the Amendment 4 layout. Add `gomad-process` as a low-layer crate (depends only on `rustix`/`libc` and `std`; no wasmtime, no evidence types) and update the dependency graph and crate table. Without this amendment, R10 treats the crate as a finding.
2. Move the custody supervisor (`sys.rs`) and a lean API into it: spawn with argv/cwd/exact environment, deadline, kill the group, reap descendants, resource usage, and the lean result type from fn-242.53.
3. Port the launcher, stress process, backstop child and measurement cancellation onto it, so all four gain descendant reaping.
4. Keep the Linux-only syscalls behind one platform module in this crate, so fn-248's Darwin parity has a single place to implement.
5. Expected size: ≈1.5–2k lines total versus ≈4.8k today (estimate; record the actual counts).

Ordering:
- After fn-242.53, the receipt-field review (cross-spec), so the crate exposes the lean type.
- After fn-240.1 (survey) and fn-240.5 (foundations).
- Before fn-240.9 (gomad-install / gomad-execution, which hold the backstop and launcher) and fn-240.12 (gomad-stress).
- Coordinate with fn-254.5 (shuttle seams) and fn-248 (Darwin).
## Acceptance
- `gomad-process` is in the amended Amendment 4 graph and table; no other crate calls `fork`, `setsid`, `setpgid`, `kill_process_group`, `waitid` or `Command::spawn` for supervised children (grep audit with an allow-list).
- A golden fixture shows a timed-out grandchild being reaped for each of the launcher, stress, backstop and measurement callers, where today it leaks.
- The cycle's custody mutation controls still fail as before; the before/after line counts are recorded.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
