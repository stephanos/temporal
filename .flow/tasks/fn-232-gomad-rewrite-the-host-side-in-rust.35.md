---
satisfies: [R1]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.35 Release profile overflow-checks = true (umpire parity)

## Description
Every gomad build, test and lint command uses `--release`, and the workspace has no `[profile]` section, so integer overflow in host code wraps silently. Umpire's model workspace sets `[profile.release] overflow-checks = true` (fn-210 R11: same arithmetic answer in every profile), which turns a wrap into a panic. For gomad that panic becomes the R1 explicit-abort path, not a silently wrong fuel, budget, offset or size.

Do: add `[profile.release] overflow-checks = true` to `tools/gomad/rust/Cargo.toml`. Also decide on `debug = 1` (line tables for profiling, as umpire has) and record the decision; profiling is under the deferred perf window, so `debug` may be left unset. The check costs some runtime, but correctness takes priority over perf here.

Any overflow the qualification cycle turns up is a real bug: fix it with `checked_`/`saturating_`/explicit `wrapping_` arithmetic chosen per site, never by turning the check off. Intentional wrapping (hashes, CRC, PRNG) must already use `wrapping_*` or `Wrapping<T>`; audit the ~340 existing `wrapping_`/`checked_` sites only where the cycle fails.

Ordering: share one qualification cycle with fn-232.34 (the wasmtime bump) when both are ready. Both change host runtime behaviour, so neither lands on unit tests alone.

## Acceptance
- `[profile.release] overflow-checks = true` is in the workspace `Cargo.toml`; the `debug` decision is recorded in the task evidence.
- A green qualification cycle on head with the profile change; every overflow panic it surfaced is fixed at the site with the arithmetic intent made explicit, and listed in the evidence.
- `make -C tools/gomad test-harness test-rust-engine validate-wasm-determinism` is green.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
