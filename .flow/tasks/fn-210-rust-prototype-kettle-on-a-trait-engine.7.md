---
satisfies: [R7, R24]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.7 DESIGN.md: authoring, size, speed, tooling comparison and the recommendation

## Description
**Shared fixtures 2026-10-18 (R29):** end-to-end fixtures live in `umpire/tests/fixtures/<name>/` after .13; their expected files select the consumers (checker, TLC, Apalache).

**Recut 2026-10-18 (R24):** the recommendation follows R24 alone (R20 on the stock toolchain: Rust; otherwise Scala, with the blocker). Authoring, speed (against the Scala note's pre-fix numbers, labelled so), lift size, build time and diagnostics are reported as information for sizing the follow-up work. .9 has already written the verdict section; this task completes the rest of the note.

**Recut 2026-10-18 (R19, R20):** also judge the two vertical slices: how much of `umpire-lift` the token route needed against the Scala plugin and lifter, whether TLC agreed first time, the wasm target that worked, module size, load time and per-call latency against a JVM call, and what each slice says about extending the prototype.

**Owner 2026-10-18:** as the last step of fn-210, move the surface comparison (the options README's table and each option's rejection reason) into DESIGN.md, then delete `model2/umpire/samples/options/`.

Write `model2/DESIGN.md`: Kettle.rs vs Kettle.scala line by line (constructs better/worse), crate line counts vs the Scala counterparts (dsl, checker, samples), `verify_all` wall time vs the note's 0.37 s (and the JVM's first-call cost), clean and incremental build times, rust-analyzer behaviour inside the impl, what macro_rules helpers (if any) earned their place, and a recommendation: extend to Bank and Voting, or stop. This note is the input to un-deferring the other specs.

**Files:** model2/DESIGN.md, MILESTONES.md.

**Quick commands:** `mise exec -- cargo test --workspace --release`; `hyperfine` or `time` on the test binary
## Acceptance
- Note present with every section and numbers.
- MILESTONES names the recommendation.
- `model2/umpire/samples/options/` is deleted after DESIGN.md carries its comparison.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
