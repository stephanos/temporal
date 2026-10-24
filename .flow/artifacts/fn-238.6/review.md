# fn-238.6 review: interpreted checker by default (db9df02bb0, e9f71b00ae, 39cdae2691, be59562661)

Reviewer: independent, fresh context, 2026-10-24. Base c3c95eb00b, branch `work/fn2386-default`, worktree `.worktrees/fn2385-interpreted` (left clean at be59562661).

## Verdict: NEEDS_WORK

The first three commits (the per-claim callbacks, reading fields without the cache, and trying arms by their written guards) are correct, and I would ship them. The switch commit be59562661 should not land yet, for three reasons:
- the gate as written is not met (the sets half is unmeasured);
- the wide-shape ratio does not reproduce under 2x with any margin;
- the switch misses the flagship admitted Model, and the docs overstate its coverage.

## What I verified (no finding)

- **Written-guard recovery is exact.**
  - `Branch` has one constructor, `front/src/body/effects.rs:86`, with `effective = previous.not().and(guard)` and `previous = previous.or(guard)` (`:66-67`).
  - `written_guards` (`checker/src/interpret.rs:741-756`) rebuilds `earlier` with the same `Expr::or`, `not` and `binary` constant folds, so the structural `==` (including `ty`) matches front output exactly.
  - Literal-false guards and catch-alls are handled. After a catch-all the effective guard is `false` and does not match, so all later arms fall back.
  - A mutant tree whose left conjunct differs also falls back, and `earlier` becomes `None`.
  - `checked_step` (`:329-368`) tries arms in order and `break`s on the first that holds. `And` and `Or` short-circuit (`:840-845`).
  - Guards are built before the body's definitions and never contain `Local`, so arm-local memo state cannot differ.
  - An earlier guard that would panic has already panicked in its own arm.
  - So first-match semantics, evaluation order and panics are preserved.
- **Dropping the field cache is behaviour-neutral.**
  - A `Frame` holds only `&S` for the before and after states.
  - `next` is a clone, mutated only after every evaluation of the arm.
  - `__umpire_get` is a pure generated `match`.
  - Self-loops, where before and after are the same, are both immutable reads.
- **The callbacks are sound.**
  - Each callback is a non-capturing closure with a literal index, coerced to a `fn` pointer. It is emitted only for admitted modules (`dsl/macros/src/lifted.rs:18-19`), `#[cfg(not(target_arch = "wasm32"))]` and `#[doc(hidden)]`, under `umpire_dsl::__lifted`. There is no author-visible surface.
  - Callbacks are monomorphised only for `W = Interpreted<M>`.
  - The count check in `registered` (`interpret.rs:445-458`) makes `self.program.*[index]` in-bounds.
  - The 70-claim control interprets and agrees.
- **The differential is still a real check.** `interpreted()` compares `world()` compiled against `Interpreted::new(world())` (`testing/src/interpreted.rs:25-45`).
  - `interpreted_agrees_with_compiled` asserts that every admitted fixture was interpreted (`fixtures/fixtures.rs:92-96`).
  - `__checked!` uses the same `Probe` impl block and bounds, so it selects the same way.
- **Refused Models, the guest and typed replay stay compiled.**
  - A refused Model has no `Interprets` impl, so it takes the `Unadmitted` path, which renders the compiled files.
  - The guest (wasm32) has no `Interprets`.
  - `testing/src/backend.rs` is untouched.
  - `Bound::check` uses the compiled `Bound::native`.
- **The compiled path is equivalent to the old `run()` bodies.** Bound-form rows are always rendered; plain-form rows only at 3 states or fewer. The negative claim goes through `refuted_from` with the default budget.
- **Goldens.** `sha256sum -c .flow/tmp/fn-238.6/baseline.sha256` (800 files) passes on the worktree.
- **Tests.** `cargo test -p umpire-fixtures --test interpreted`: 7 pass, timing ignored. `-p umpire-checker -p umpire-dsl-macros`: all ok.
- **Owner rules.** No new dependencies (no `Cargo.toml` changed). No Temporal under `model/umpire`. Expansion snapshots are unchanged, because `lifted::embed` is not rendered.
- **Benchmark hygiene.**
  - Release profile for both sides: thin LTO, `codegen-units = 1`, overflow checks.
  - A `compare` warm-up runs per shape, and the order alternates.
  - Both sides get the same starts and budget.
  - `Interpreted` holds no cross-run cache.

## Findings

1. **Blocker. The gate is not met as written, yet the switch is made.**
   - **Where:** `be59562661`; `.flow/tasks/fn-238-shared-front-end-one-model-reading-for.6.md` (Description: "If the gate fails, stop … leave the compiled checker as default"); `model/DESIGN.md:100-102`.
   - **Problem:** The gate's per-construct half (sets at most 25% above A) is unmeasured, and the handoff marks it "deferred". The task allows the switch only once the gate passes, and the owner has not accepted the deferral. The conductor's instruction "switch if the ratio meets 2×" is not owner consent to drop half of an owner-set gate.
   - **Fix:** Hold be59562661, which is separable, until the owner records acceptance of "sets deferred to the first set construct" in the spec or design and the quiet measurement passes (finding 2). Otherwise keep compiled as the default and land only the first three commits.

2. **Major. The wide-shape ratio is marginal and does not reproduce under 2×.**
   - **Where:** `fixtures/interpreted.rs:391-430`; `.flow/tmp/fn-238.6/bench-head.log`.
   - **Evidence:** The `timing` test at scale 1 on the same host (load about 31-34, release, worktree target), run four times:

     | Run | Kettle | Wide |
     | --- | --- | --- |
     | 1 | 1.78 | **2.31** |
     | 2 | 1.79 | 1.84 |
     | 3 | 1.82 | **2.36** |
     | 4 | 1.77 | 1.72 |

     - Kettle is stable at about 1.8×. Wide exceeds 2× in 2 of 4 runs, and its compiled median alone swings from 1.51 to 2.19 ms.
     - The handoff's 1.80× is one draw from a wide distribution.
   - **Failure scenario:** The quiet run lands at 1.9×, the switch ships, and normal noise or the next construct pushes it past 2×. Or the quiet run fails, and the gate decision was made on an unrepresentative number.
   - **Fix:** Take the decision only on the quiet run.
     - Report the median of per-pair ratios (`interpreted[i] / compiled[i]` within each interleaved pair), not the ratio of two independent medians.
     - Add an IQR or p10-p90.
     - Give wide more repetitions; scale 5 is only 1,000 pairs.
     - Require the upper quartile, not the median, to be at most 2×, or record the margin explicitly.

3. **Major. The benchmark omits the setup that every interpreted `run()` pays.**
   - **Where:** `fixtures/interpreted.rs:395` (`Interpreted::new` hoisted out of timing) versus `testing/src/interpreted.rs:233` (`Admitted::checked` calls `Interpreted::new` on every `run()`, and the runner calls `run` twice for its determinism check).
   - **Evidence:** A scratch release binary outside the repo (same profile; `explore_within` on Kettle, 2,000 iterations, medians) measured `Interpreted::new` at **13.5 µs**:

     | Run | Explore only | Explore plus setup |
     | --- | --- | --- |
     | 1 | 1.69× | 1.83× |
     | 2 | 1.85× | **2.01×** |

     Setup is about +0.15× on Kettle.
   - **Problem:** The gate says "whole-checker". The checker as the fixture runner uses it includes construction, which reads the embedded tree and resolves it.
   - **Fix:** Either include `Interpreted::new` in the timed interpreted side, or state explicitly in the gate record that the ratio is exploration-only and give the setup cost beside it.

4. **Major. The switch misses admitted Models, and the docs claim it covers them.**
   - **Where:** `temporal/feature/activity/tests.rs:16-25`; `testing/src/bound.rs:70-72`; `fixtures/fixtures/action_inputs/tests.rs:24`; `fixtures/fixtures/parameterized_world/tests.rs:22`; `model/README.md:797-800`; `model/DESIGN.md:102-104`.
   - **Problem:**
     - The Temporal activity is admitted: its `run()` calls `umpire_testing::interpreted`. But it renders its checker goldens from compiled `Checker::explore(&world(), 64)`. The most important admitted Model is therefore not on the interpreted path.
     - `Bound::run`, used for the additional selected Worlds of `action_inputs` and `parameterized_world`, stays compiled. Those Worlds are not differential-checked either, because the hook covers only `$world`.
     - README says "a fixture's checker files (`run()`) come from the interpreter when its Model is admitted", and DESIGN says the same. Both are false for these cases.
   - **Fix:** Expose the probe as a helper, for example `umpire_testing::checked(world, budget, negative, rows)` behind `__checked!`.
     - Use it in the activity's `run()`, keeping its extra expansion and coverage files.
     - Optionally route `Bound::run` through it (with `__interpreted_probe` autoref at the call site, which `bound_for` can do because it is macro-generated).
     - Otherwise narrow README and DESIGN to "`fixture!`-generated `run()` only" and list the exceptions.

5. **Minor. "TLC agreement stays compiled" does not hold for plain-form fixtures.**
   - **Where:** `testing/src/lib.rs:113-121` (`__lift_hooks` `check` passes the fixture's `run` → `backend::check_machine(world, &module, folder, &run()?)`); `testing/src/backend.rs:314-348` (`agreement` takes `states.txt` and the report verdicts from those files); `model/README.md:800`; `model/DESIGN.md:103`; the handoff ("TLC agreement (`check`) … stay compiled").
   - **Problem:** For `fixture!(world, budget[, negative])`, TLC's state set and verdicts are now compared against interpreted files. Only trace replay uses a compiled graph (`check_machine` re-explores `world`). Bound-form fixtures (`Bound::check`) still compare against compiled. The comparison stays sound, since the differential enforces compiled == interpreted, but the stated invariant is wrong. A divergence would also surface as a TLC disagreement on the interpreted side, not only as the differential failure.
   - **Fix:** In `__lift_hooks`' `check`, render the compiled files for the agreement (the `Unadmitted`-style `files(&world, …)`), or correct the three documents.

6. **Minor. No negative test exercises the written-guard fallback.**
   - **Where:** `checker/src/interpret.rs:741-756`; `fixtures/interpreted.rs:271-347`.
   - **Problem:** None of the seeded mutants touches a guard, and every admitted tree has exactly the front's shape, so the `None` and fallback branch and the `earlier = None` propagation never run under test.
   - **Failure scenario:** A refactor makes the match strip the left conjunct even when it differs, for example by comparing only `kind`, or by not resetting `earlier` after a mismatch. A mutant whose arm 2 guard is `!(g1') && g2` is then evaluated as `g2`, and nothing fails.
   - **Fix:** Add seeded mutants on Kettle or `shared_returns`:
     - (a) replace arm k's left conjunct with `!(g1')`, so that the effective guard matters and the run must disagree with compiled exactly as a fully effective-guard evaluation would;
     - (b) mutate arm 1's guard while leaving later arms intact, so that later arms must keep their effective guards.

     Also add a small unit test of `written_guards` on front-shaped and non-front-shaped inputs, including a literal `false` and a catch-all followed by an arm.

7. **Minor. The claim-count refusal is tested only for extra invariants, and it narrows the mutant surface.**
   - **Where:** `fixtures/interpreted.rs:353-367`; `interpret.rs:445-458`.
   - **Problem:** The mismatch error for postconditions and progress claims, and for a tree with fewer claims, is untested. Separately, `from_tree` now refuses any mutant that adds or removes a claim; .5 accepted up to 64.
   - **Fix:** Add the two other kinds and the fewer-claims case to `every_claim_has_its_own_callback`. Note in DESIGN that claim-count mutants are refused by construction.

8. **Nit. Stale text.**
   - `checker/src/interpret.rs:335`: "One frame per step, so each field is read once however many arms are tried" is no longer true, since fields are now read every time.
   - `fixtures/interpreted.rs:3`: the module doc still lists "the claim cap".
   - `model/DESIGN.md:689` ("fn-238.5/.6 stay deferred absent an interpreted-checker need") and `model/design/shared-front-end.md:8-10` (Status: ".5 and .6 stay deferred") contradict the new "Interpreted by default" paragraph.
   - `model/DESIGN.md:100` cites 1.77× / 1.78× (the written-guards run), while the handoff's gate table cites head at 1.77× / 1.80×.

9. **Nit. Benchmark provenance.**
   - **Where:** `.flow/tmp/fn-238.6/bench.sh:13`.
   - **Problem:** The log stamps `git rev-parse --short HEAD` with no dirty marker. `bench-nocache.log` is stamped db9df02bb0 and `bench-written-guards.log` e9f71b00ae, yet each measured the then-uncommitted next change. A reader cannot tell which code produced which number.
   - **Fix:** Stamp `git describe --always --dirty` plus `git diff --stat HEAD`.
