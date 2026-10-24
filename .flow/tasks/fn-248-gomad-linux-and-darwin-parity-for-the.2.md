---
satisfies: [R2]
---
# fn-248-gomad-linux-and-darwin-parity-for-the.2 One host module for OS queries

## Description
Add a `host` module owning available/total memory, CPU model, boot identity, load average, peak RSS and executable path. Linux reads `/proc`; Darwin uses `libc::sysctlbyname` (`hw.memsize`, `kern.bootsessionuuid`, `machdep.cpu.brand_string`, `vm.loadavg`) and `host_statistics64` for available memory. Route `measurement/host.rs`, `cycle/scheduler/admission.rs`, `stress/mod.rs` (replace `cat /proc/...` subprocesses), `qualify/usage.rs`, `wasmhost/src/metrics.rs` and `wasmhost/src/cache.rs` (`std::env::current_exe()`) through it, keeping existing bounds and parse refusals. Peak-RSS reset returns an explicit unsupported value on Darwin.

## Acceptance
- [ ] No `/proc` path or `cfg(target_os)` outside the host module and the supervisor custody backend (tests excepted only where listed).
- [ ] Malformed-input tests for each parser run on both platforms.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
