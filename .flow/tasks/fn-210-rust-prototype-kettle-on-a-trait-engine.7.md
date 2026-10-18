---
satisfies: [R7]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.7 DESIGN.md: authoring, size, speed, tooling comparison and the recommendation

## Description
**Owner 2026-10-18:** as the last step of fn-210, move the surface comparison (the options README's table and each option's rejection reason) into DESIGN.md, then delete `model2/framework/samples/options/`.

Write `model2/DESIGN.md`: Kettle.rs vs Kettle.scala line by line (constructs better/worse), crate line counts vs the Scala counterparts (dsl, checker, samples), `verify_all` wall time vs the note's 0.37 s (and the JVM's first-call cost), clean and incremental build times, rust-analyzer behaviour inside the impl, what macro_rules helpers (if any) earned their place, and a recommendation: extend to Bank and Voting, or stop. This note is the input to un-deferring the other specs.

**Files:** model2/DESIGN.md, MILESTONES.md.

**Quick commands:** `mise exec -- cargo test --workspace --release`; `hyperfine` or `time` on the test binary
## Acceptance
- Note present with every section and numbers.
- MILESTONES names the recommendation.
- `model2/framework/samples/options/` is deleted after DESIGN.md carries its comparison.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
