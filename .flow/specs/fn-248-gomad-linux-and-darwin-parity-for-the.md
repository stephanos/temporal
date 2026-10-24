# Gomad: Linux and Darwin parity for the Rust host

## Goal & Context

Owner, 2026-10-24: "can we remove/reduce the system-dependent parts? want near identical code for linux and darwin". Today `wasmhost` builds on macOS but `gomad` does not: `cargo check` fails with 4 library errors and 6 library-test errors (`Mode::from_raw_mode` takes `u16` on Darwin, `st_dev` is `i32`, and `rustix::fs::mkfifoat` doesn't exist there). Behind that, Linux-only code is concentrated in a few places:

- **Process supervisor** (`gomad/src/cycle/process/sys.rs`, `domain.rs`): hand-written C bindings with magic numbers (`prctl(36,…)` for the child subreaper, `fcntl(9, 2, 1)` for close-on-exec, `sigaction` as `[u64; 32]`, `_Fork`, `close_range`, `__errno_location`), plus `/proc/thread-self/children` and `/proc/{pid}/task` enumeration.
- **Resource accounting**: two separate per-OS `Rusage` struct definitions (`gomad/src/qualify/usage.rs`, `wasmhost/src/metrics.rs`), `/proc/self/status` peak RSS, `/proc/self/clear_refs` and glibc `malloc_trim`.
- **Host facts**: `/proc/meminfo` parsed in three places (`measurement/host.rs`, `cycle/scheduler/admission.rs`, `stress/mod.rs` via `cat`), plus `/proc/cpuinfo`, `/proc/sys/kernel/random/boot_id` and `/proc/loadavg`.
- **Executable path**: `/proc/self/exe` in `wasmhost/src/cache.rs`, with a `cfg` fallback.
- **Signals**: a raw `signal()` binding in `sweep/process.rs`.

## Architecture & Data Models

- One `host` module (in the crate fn-240 assigns) owns every OS query: available and total memory, CPU model, boot identity, load average and peak RSS. It reads `/proc` on Linux and `sysctlbyname`/`host_statistics64` on Darwin, through `libc`. Callers never read `/proc` or call `sysctl` directly. `cfg(target_os)` appears only inside this module and the supervisor's custody backend.
- `libc` (pinned `=`) replaces every hand-written C declaration. `libc::getrusage` replaces both `Rusage` structs, with the `ru_maxrss` unit difference (KiB on Linux, bytes on Darwin) normalized in one place.
- `std::env::current_exe()` replaces `/proc/self/exe`.
- `signal-hook` (pinned `=`) replaces the raw SIGINT/SIGTERM handler in `sweep`.
- **Process supervisor.** One portable double-fork supervisor through `libc`: new session and process group per leaf, a report pipe, `wait4` rusage, and kill-the-group cleanup. A small custody backend adds Linux's subreaper and `/proc` descendant sweep. On Darwin the backend uses `proc_listchildpids`/`proc_pidinfo` (libproc) for descendant enumeration. Darwin has no subreaper: a descendant that calls `setsid` and is reparented to launchd cannot be reclaimed there. The spec records that limit rather than hiding it.

## API Contracts

- CLI, records and outputs are unchanged on Linux. On Darwin every verb runs, and records name the platform in the host facts they already carry.
- `cargo test --workspace` passes on Linux and on macOS (aarch64). Tests that need Linux-only facilities are marked `#[cfg_attr(not(target_os = "linux"), ignore = "<reason>")]` with the reason. The ignored set is listed in `.flow/artifacts/fn-248/darwin-ignored.md` and may only shrink.
- Qualification resource floors, timing policy and final qualification stay Linux-qualified (fn-246). Darwin parity covers development, tests and ordinary verbs.

## Edge Cases & Constraints

- The supervisor's fork-to-exec rule stays: no allocation, locks or unwinding between fork and exec/_exit. `libc` calls only; no `nix` wrappers that allocate.
- The `malloc_trim` call stays glibc-only behind one function; elsewhere it's a documented no-op.
- Peak-RSS reset (`clear_refs`) has no Darwin equivalent. Callers that need it get an explicit "unsupported" value, never a silently stale peak.
- No new `cfg(target_os)` outside the `host` module and the custody backend. Errors: a `/proc` path or OS `cfg` elsewhere is a finding.

## Acceptance Criteria

- **R1:** `cargo check --workspace --all-targets` and `cargo test --workspace` pass on macOS aarch64 and Linux. Darwin-ignored tests are listed with reasons. Errors: an ignore without a reason, or growth of the list, is a finding.
- **R2:** One `host` module owns all OS queries (memory, CPU, boot identity, load, peak RSS, executable path via `current_exe`), with tests on both platforms. No `/proc` path or `cfg(target_os)` remains outside it and the custody backend. Errors: a surviving `/proc` read or duplicated meminfo parser elsewhere is a finding.
- **R3:** Every hand-written `extern "C"` block is replaced by `libc`, and `getrusage` has one portable implementation. Errors: a surviving hand-written C declaration or `Rusage` struct is a finding.
- **R4:** `sweep` signal handling uses `signal-hook`. Errors: a raw `signal()` binding is a finding.
- **R5:** The cycle supervisor runs on both platforms through `libc`, with a per-OS custody backend. Its existing custody, cleanup, spawn-ambiguity and accounting tests pass on Linux unchanged and on Darwin, apart from tests that need the subreaper, which are listed. The Darwin setsid-escape limit is documented in `ARCHITECTURE.md`. Errors: a Linux custody guarantee weakened to match Darwin is a finding.

## Boundaries

- No change to Linux qualification guarantees; Darwin does not become a qualification platform.
- No Windows support.
- No CI run, PR or push is authorized by this spec.

## Decision Context

Owner, 2026-10-24 crate review, items 2, 14 and 15 (libc supervisor, signal-hook, getrusage) were folded in here, and the owner asked for near-identical Linux and Darwin code. Rejected: `sysinfo` (heavy, samples far more than needed), `nix` in the supervisor (its `execve` allocates between fork and exec), and `procfs` (Linux-only, so no help with parity).

Sequencing (owner, 2026-10-24): after fn-242 (its tasks .13 and .15 just landed the supervisor and admission code) and before fn-240, so the extraction moves portable code. It can run in parallel with fn-247.

Owner decision (2026-10-24): Darwin's weaker descendant custody (no subreaper; a `setsid` descendant can escape cleanup) is acceptable for development runs. The cycle verb dispatches on Darwin; the limit is documented, not refused.

