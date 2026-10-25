---
satisfies: [R4]
---
# fn-254-gomad-deeper-rust-host-testing-mutation.5 shuttle and loom for the host threading modules

## Description
R4: deterministic concurrency testing.

1. Add a test-only sync shim per crate: `#[cfg(shuttle)]` / `#[cfg(loom)]` re-exports of `thread`, `sync::{Mutex, Condvar, mpsc, atomic}`; production `std` otherwise.
2. The shuttle harnesses cover the surveyed shuttle targets (broker handoff, linked cluster, cancellation, backstop, slots, process registry), with randomised schedules and a budget. A failing schedule prints its seed, and replay from that seed is a committed regression test.
3. The loom models cover the small primitives the survey lists, with every interleaving explored under loom's bounds.
4. Process-level behaviour (the fn-232 R1 backstop killing the process) is out of scope for shuttle/loom and stays in its external harness.

Ordering: land each module's shim with its fn-240 crate cut (or, if this task runs first, in a form the cut can move unchanged). Cross-spec order recorded here per the survey's overlap list.

## Acceptance
- Each surveyed threading module runs under shuttle in `deep:shuttle`, and each surveyed primitive under loom in `deep:loom`.
- A seeded race (a removed memory-ordering fence, or a dropped condvar notify) is found and replays from its seed.
- Production builds contain no shim code (checked with `cargo tree -e features` and a `cfg` audit).


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
