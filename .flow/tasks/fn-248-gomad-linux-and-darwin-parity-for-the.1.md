---
satisfies: [R1]
---
# fn-248-gomad-linux-and-darwin-parity-for-the.1 Make gomad build and test on macOS

## Description
Fix the Darwin compile errors: `Mode::from_raw_mode` width (`campaign/prepare/compilation_cache.rs:389,394`), the `st_dev` comparison type (`install/derive_cache.rs:227`), and `rustix::fs::mkfifoat` in `cycle/process/fifo_tests.rs` and `measurement/retention_tests.rs` (use `libc::mkfifo` on a pinned path, or `mkfifoat` where available). Run `cargo test --workspace` on macOS aarch64. Mark tests that need Linux-only facilities `cfg_attr(not(target_os = "linux"), ignore = "<reason>")` and list them in `.flow/artifacts/fn-248/darwin-ignored.md`. Add a Make target that runs the suite on the host platform.

## Acceptance
- [ ] `cargo check --workspace --all-targets` and `cargo test --workspace` pass on macOS aarch64 and on Linux.
- [ ] Every Darwin-ignored test carries a reason and appears in the list.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
