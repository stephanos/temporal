---
satisfies: [R7, R24, R30, R34, R35, R37, R38, R39, R40]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.7 DESIGN.md: authoring, size, speed, tooling comparison and the recommendation

## Description
**Recut 2026-10-18 (R36-R39):** no TLC slice to judge (moved to fn-212). The recommendation carries R34, R35 and R37-R39 into the Case slice's plan: Testpilot at most 11,000 non-test lines, generic runtime at most 5,000, no CEL, worker asks the Model at callbacks (out-of-process workers through the runner's endpoint, one Model per run, a thin shim per SDK: R40), one command, Runs without profiles, receipts or provenance.

**Next slice (R34):** DESIGN.md's recommendation section carries R34's and R35's Testpilot targets (no CEL; generic runtime at most 5,000 lines, Testpilot at most 13,000) into the Case slice's plan.

**Feedback loop (R30):** keep the inner loop at 3 s or less and `make model-check` at 30 s or less, warm; paste `make model-time` output into the done summary; a breach is fixed here or excepted by the owner.

**Shared fixtures 2026-10-18 (R29):** end-to-end fixtures live in `umpire/tests/fixtures/<name>/` after .13; their expected files select the consumers (checker, TLC, Apalache).

**Recut 2026-10-18 (R24):** the recommendation follows R24 alone (R20 on the stock toolchain: Rust; otherwise Scala, with the blocker). Authoring, speed (against the Scala note's pre-fix numbers, labelled so), lift size, build time and diagnostics are reported as information for sizing the follow-up work. .9 has already written the verdict section; this task completes the rest of the note.

**Recut 2026-10-18 (R19, R20):** also judge the two vertical slices: how much of `umpire-lift` the token route needed against the Scala plugin and lifter, whether TLC agreed first time, the wasm target that worked, module size, load time and per-call latency against a JVM call, and what each slice says about extending the prototype.

**Owner 2026-10-18:** as the last step of fn-210, move the surface comparison (the options README's table and each option's rejection reason) into DESIGN.md, then delete `model/options/`.

Write `model/DESIGN.md`: Kettle.rs vs Kettle.scala line by line (constructs better/worse), crate line counts vs the Scala counterparts (dsl, checker, samples), `verify_all` wall time vs the note's 0.37 s (and the JVM's first-call cost), clean and incremental build times, rust-analyzer behaviour inside the impl, what macro_rules helpers (if any) earned their place, and a recommendation: extend to Bank and Voting, or stop. This note is the input to un-deferring the other specs.

**Files:** model/DESIGN.md, MILESTONES.md.

**Quick commands:** `cd model && mise exec -- cargo test --workspace --release`; `hyperfine` or `time` on the test binary

**Parameterized-Model readiness (owner, 2026-10-19):** DESIGN.md gets a section "Parameterized Models: readiness" recording that the engine is ready by construction (R9: the world is a value implementing `Machine`; parameters are its fields; `starts`, `actions`, `step` and every claim predicate receive `&M`; generic state is admitted by the trait bounds; only `Finite` is missing, R17), and that the attribute surface has five gaps, each to land with the first sample that needs it (the samples port is parked): (1) no declared world struct, `#[machine]` generates a unit `World`; (2) handlers and claims cannot see the world; (3) `#[starts(range)]` is literal, no parameter-dependent starts; (4) actions have no inputs or domains; (5) `#[machine]` refuses generic Models. Record the settled shape for closing them without changing Kettle: a `#[derive(World)]` struct in the module (absent means the unit world); the world as an explicit extra parameter `w: &World<..>` on handlers and claims, present when the method wants it and detected by arity like `before`; a `#[starts]` impl block for computed starts, `#[starts(range)]` kept for literal domains; action variants with fields for inputs, domains from `Finite` or the world; generics accepted on `Model`, `World` and the action enums. Numeric `.within(n)` is planned in fn-214, rather than implemented by fn-210; distinguish it from the current trait builder that supplies an environment assumption. Record that a runtime parameter-derived bound (`w.resource_managers.len()`) needs no further authoring shape once (2) and the bound builder land, and verify the statement against the final code. The write-up claims no more than Kettle proves.
## Acceptance
- Note present with every section and numbers.
- MILESTONES names the recommendation.
- `model/options/` is deleted after DESIGN.md carries its comparison.
- DESIGN.md has a "Parameterized Models: readiness" section: engine ready via R9 (with `Finite` as the one missing piece), the five attribute-surface gaps, and the settled closing shape (`#[derive(World)]`, `w: &World` by arity, `#[starts]` block, action inputs, generics), without claiming any of it is implemented.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
