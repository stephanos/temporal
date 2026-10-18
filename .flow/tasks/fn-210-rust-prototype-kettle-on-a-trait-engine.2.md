---
satisfies: [R2, R4, R8, R11, R14, R16]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.2 framework-checker: BFS explore with budget, invariants per state, postconditions per edge, verify_all, refuted

## Description
**Note 2026-10-18 (R22):** cases will move to their owning crates in fn-210.10 (synthetic checker cases to `checker/cases/<name>/{model.rs,expected/}`, the runner to `umpire-testing`). Finish .2 in the current layout; keep each new case's Model in its own module so the move is mechanical.

**Note 2026-10-18 (R21):** the framework is renamed `umpire` (folder `model2/umpire/`, crates `umpire-*`) by fn-210.10 right after this task; keep using the current names until then.

**Deps 2026-10-18 (R18):** visited set = `IndexSet<State, FxBuildHasher>` (`indexmap`, `rustc-hash`): the insertion index is the `StateId`, iteration is discovery order, parents are a `Vec<(StateId, Action)>`; this is the `FullStates` impl of `Visited`. Vacuity tracking uses `fixedbitset`. Dev: `insta`, `proptest`.

**Recut 2026-10-18 (R11, R14, R16):** run each step under `catch_unwind` and refute on panic or overflow with path, action and message; keep order structural (catalog order, ordered starts, FIFO BFS, no hash iteration in output); put the visited set behind `Visited<S>` with a full-state implementation.

**Recut 2026-10-18 (R8, golden files):** deliver the behaviour runner and the oracle test, not assertions. Add `insta` (dev) and a `samples` test that runs every registered Model and snapshots `cases/<model>/expected/report.txt` and `paths/<claim>.txt`; give states a stable `Display` and sort whatever reaches a golden. Add `proptest` (dev) and a brute-force reference explorer over small random Models (2-3 small vars, 2-4 guarded actions) that must agree with `Checker::explore` on reached states and claim verdicts. Add `make model2-update`.

Implement `Checker::explore::<M>(budget)`: BFS from `starts()` with a `HashSet<M>` visited set and a parent map for paths; evaluate invariants on each new state and postconditions on each emitted edge with `before`, `after` and the action; `Result` with verdict, counts (starts, reached, emitted, depth, cut); `Result::verify_all() -> Report` with per-claim results and vacuity warnings (action never fired, `after` claim never triggered); `Checker::refuted(claim)` for negative controls. Hit the Scala note's Kettle counts: 81 starts, 283 states, depth 62.

**Files:** model2/framework/checker/src/lib.rs (+ explore.rs, report.rs), model2/framework/samples/tests/kettle.rs.

**Quick commands:** `mise exec -- cargo test -p framework-checker -p samples`
## Acceptance
- `cases/kettle/expected/report.txt` shows 81 starts, 283 states, depth 62, both claims verified, no vacuity; `paths/never_beeps.txt` holds the `!beeping` counterexample.
- Two consecutive runs produce byte-identical goldens.
- The oracle property test passes on 256 cases.
- A budget below one refuses; a small-budget case's report shows `cut: true` and unknown verdicts.
- A golden with no case fails the run.
## Done summary
The checker now explores world/state machines with bounded FIFO BFS, declaration-ordered safety reports, tri-state claims, caught effect/predicate panics, and counterexamples retaining action and choice identity. Kettle reaches 81 starts, 283 states, 424 emitted choices and depth 62; all three canonical safety/step claims verify, and never_beeps has the actual 62-step counterexample.

The FullStates implementation uses IndexSet with FxBuildHasher behind Visited, and fixedbitset tracks vacuity. Ordered rows retain outcomes, facts, rejecting choices and duplicate destinations. The 17 registered fixtures cover shared starts, cuts before start admission, known-target/self-loop postconditions, refutations surviving cuts, vacuity, holes, panics, overflow and linear clone work. One proptest oracle checks 256 random worlds against independent finite-universe closure, verdicts, rows and counterexample edges.

Insta owns reviewable binary snapshots with text payloads. Required .txt paths are symlinks to the single .snap.txt payload, with adjacent .snap metadata. A deliberate mismatch generated a pending snapshot; cargo insta accept updated its payload and alias, and the test passed. Missing aliases, metadata and payloads, plus orphan text and metadata, each failed with exit 101. The oracle rejected a deliberate verdict mutation, and the deep-chain golden failed before lazy path construction and passed afterward.

baseline: green at original pre-edit bb6162b4079905ccbfd803878040f93272b6e737 using the focused Quick command. The owner recut R9-R18 during execution; the known old-API compile-red was migrated under the conductor's instruction. Effective task base is 8be459b1ff449b64acf0caa8658fde07b90b50f4 after the revised .1 closure. Authorized owner planning and fixture-layout changes are included in the commit range.

Verification passed with make model2-check, the canonical cargo test --workspace, release-mode fixtures, cargo deny check bans, and scoped Flow validation. The final workspace log collected the oracle, samples runner and compilation fixture; it was not an empty selection. GREEN_RECEIPT is .flow/tmp/green-receipts/bbdf5afc-unittest.json. An earlier receipt attempt refused the owner's dirty MILESTONES.md and was not treated as a skip. Global Flow validation also exposed inherited duplicate headings in fn-137; scoped fn-210 validation has no errors, with deferred R22/R23 coverage warnings.

Logs live under .flow/tmp/fn210-2-*. Key records are closure-model2-check.log, closure-workspace.log, final-release.log, deny.log, review-fanout.log, review-finalize.log, review-rereview.log, insta-pending.log, insta-accept.log, insta-accepted-green.log, deep-red.log, deep-green.log, oracle-mutation-red.log and missing-report.*.log. The SHIP receipt is /tmp/impl-review-receipt-8f37faba39e2-fn-210-rust-prototype-kettle-on-a-trait-engine.2.json.

Serialization uses declaration-order claims, discovery-order states, and action Debug plus zero-based choice index. Kettle state Display is temperature=<n>, heating_on=<bool>, beeping=<bool>; action names retain UserPressButton, UserDepressButton, HeaterHeat and HeaterFailover. The 708 empty/disabled evaluations remain separate from the 424 emitted choices, giving 1,132 total evaluations. Scala's disabled catalog evaluations are excluded from its emitted row; rejecting domain outcomes remain edges as the choices fixture demonstrates.

WASI compilation passes. Panic recovery has native dev/release fixture proof; the target's default panic=abort prevents claiming WASI panic recovery. Progress remains task .3. The Umpire rename, owning-crate fixture layout and testing crate remain task .10; the TLC and Go-host slices remain their assigned tasks. No Scala source changed.

stage: impl-review - ran [2026-10-18T20:14:45Z..2026-10-18T20:22:34Z] (SHIP; three first-round draws, one re-review; codex:gpt-6.1-sol:high; same-family review)
Tier: project implementer gpt-6.1-sol at medium (judge unavailable no_key)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 8beb1725abd8225abaa5c0efcb6c5e1e4dd84cf5, f9088f7eefe574cd0c25d73cdb0b9987e8681fed, 5436c493cf90e546ddd01d0a8ec21082d74ef85d, bbdf5afc1f63859ca4c974e3aa1e6bc1006c75e3
- Tests: mise exec -- cargo test -p framework-checker -p samples, make model2-check, mise exec -- cargo test --workspace, mise exec -- cargo test -p samples --release --test kettle, mise exec -- cargo deny check bans, make model2-update, mise exec -- cargo insta accept --snapshot framework/samples/cases/zero_budget/expected/report.snap, negative controls: orphan text/metadata, missing alias/metadata/payload, oracle verdict mutation, deep-chain eager paths each failed with intended exit 101; controls restored
- PRs: