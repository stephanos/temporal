---
satisfies: [R4]
---
# fn-254-fast-validation-gates-and-a-real-inner.3 Parallel test execution across binaries (cargo-nextest, owner-approved)

Touches: [model/.config/nextest.toml, model/mise.toml, Makefile, model/scripts/time-gate.sh]

## Description
Run tests across all binaries at once instead of one binary after another (spec R4). Owner approved cargo-nextest, pinned via mise (2026-10-25).

**Size:** M
**Files:** `model/.config/nextest.toml` (new), `model/mise.toml` (pinned tool), `Makefile` (`tests` stage), `model/scripts/time-gate.sh`.
**Touches:** [model/.config/nextest.toml, model/mise.toml, Makefile, model/scripts/time-gate.sh]

### Approach
- Pin cargo-nextest through mise. Replace the `tests` stage with `cargo nextest run --workspace --locked`, plus `cargo test --doc --workspace` (nextest does not run doctests).
- Test groups: TLC/Apalache/JVM-backed tests (filter by binary or name, e.g. `real_tlc_*`, `actual_tlc_*`, fixtures with `UMPIRE_BACKENDS=require`) get a bounded group so JVM heaps fit; everything else uses all cores.
- Keep fail-fast semantics for the gate. Keep `--quiet`-equivalent output, and emit the slow-test report so the timings stay visible.
- **Equivalence pin:** the full test list, including ignored tests, from `cargo test -- --list` and from nextest's list is identical.

### Investigation targets
**Required:**
- `Makefile:1290-1298` — tests, wasm, Go stages
- `model/design/gate-speed.md` — baseline per-binary times
- `model/mise.toml` — tool pins

### Acceptance
- [ ] The test list (with ignored) is identical before and after; doctests still run.
- [ ] The `tests` stage wall time is recorded against the baseline; JVM tests never exceed their group limit.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
