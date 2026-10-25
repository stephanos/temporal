---
satisfies: [R4]
---
# fn-254-fast-validation-gates-and-a-real-inner.3 Parallel test execution across binaries (cargo-nextest, owner-approved)


## Description
Run tests across all binaries at once instead of one binary after another (spec R4). Owner approved cargo-nextest, pinned via mise (2026-10-25).

**Size:** M
**Files:** framework-local nextest configuration and mise pins selected by .9 and installed by .13; Temporal nextest configuration, framework entry point and root delegation established by .10.
**Touches:** [model/umpire/.config/nextest.toml, model/umpire/mise.toml, model/umpire/dev/**, model/.config/nextest.toml, mise.toml, Makefile, model/scripts/time-gate.sh]

### Approach
- Pin cargo-nextest through mise. Replace the `tests` stage with `cargo nextest run --workspace --locked`, plus `cargo test --doc --workspace` (nextest does not run doctests).
- Test groups: TLC/Apalache/JVM-backed tests (filter by binary or name, e.g. `real_tlc_*`, `actual_tlc_*`, fixtures with `UMPIRE_BACKENDS=require`) get a bounded group so JVM heaps fit; everything else uses all cores.
- Keep fail-fast semantics for the gate. Keep `--quiet`-equivalent output, and emit the slow-test report so the timings stay visible.
- **Equivalence pin:** collect the pre-change Cargo package/target/full-name/ignored inventory and compare it to nextest under matching features/backend modes, using .4's checked migration mapping where applicable; include doctests separately. This correctness inventory is still required despite the owner's timing-baseline waiver. Reject missing, duplicated or unmatched identities. Bound the combined runner/test concurrency and demonstrate peak JVM count, including nested runner parallelism.

### Investigation targets
**Required:**
- `Makefile:1290-1298` — tests, wasm, Go stages
- `model/design/gate-speed.md` — baseline per-binary times
- `mise.toml` — current tool pins; .9/.13 establish the framework-local successor

## Owner validation cadence (2026-10-25)

Keep exact Cargo/nextest/ignored/doctest inventory parity, representative affected execution and combined JVM-limit controls now; unrelated full workspace execution waits for .8. Deferred checks remain final .8 requirements and are never reported green. This owner instruction supersedes earlier broad per-task verification commands.

## Acceptance
- [ ] The test list (with ignored) is identical before and after; doctests still run.
- [ ] The `tests` stage wall time is recorded against the baseline; JVM tests never exceed their group limit.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
