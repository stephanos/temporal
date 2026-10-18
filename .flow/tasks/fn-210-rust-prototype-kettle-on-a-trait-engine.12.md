---
satisfies: [R25, R27]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.12 Spell and Refusal: canonical value spelling in every output; misuse returns Result

## Description
Apply R25 and R27 to the code .2 delivered, before .3, .4 and .9 build on it.

- Add `trait Spell { fn spell(&self, out: &mut String); }` to `umpire-dsl` with impls for `bool`, the integer types, `Outcome`, `Option`, tuples, and (behind the `imbl` dependency when it is admitted into dsl) `imbl` maps and sets with sorted keys; records spell `Name { field: value, ... }`, enum values `Type::Variant` with fields like records.
- Require `State: Spell` and `Action: Spell` on `Machine`; implement it by hand for the trait Kettle and every checker fixture Model (derive comes in .4); delete Kettle's hand-written `Display` and every `{:?}` or `Display` that reaches an output in `umpire-testing` and `umpire-checker`'s report and path rendering.
- `Checker::explore`, `Checker::refuted` and the other questions return `Result<_, Refusal>`; `Refusal` is a `thiserror` enum (`Budget { given }`, and the variants the existing asserts guard); remove the `assert!`s and the `# Panics` docs.
- Re-render all goldens with `make model2-update`; the diff is the spelling change only (counts, order and verdicts unchanged) and is reviewed as such. A zero-budget fixture now pins the refusal text.

**Files:** model2/umpire/dsl/src/{lib.rs,spell.rs,machine.rs}, model2/umpire/checker/src/{explore.rs,report.rs,refusal.rs}, model2/umpire/testing/src/render.rs, model2/umpire/samples/src/kettle.rs, model2/umpire/checker/tests/fixtures/*/model.rs, all `expected/` goldens.

**Quick commands:** `make model2-check`; `grep -rn '{:?}' model2/umpire/testing model2/umpire/checker/src` returns nothing that reaches output

## Acceptance
- Every golden spells values through `Spell` (records with type name and braces, enums as `Type::Variant`); counts, order and verdicts unchanged.
- No `Debug` or hand-written `Display` reaches an output.
- A zero budget returns `Refusal::Budget { given: 0 }`, pinned by a fixture; no `assert!` guards API input.
- `make model2-check` passes.

## Done summary
Umpire authors now get one canonical Spell representation for states, actions, outcomes and facts in reports, rows and counterexamples. Checker questions return typed Refusal errors for zero budgets and unknown state ids, while holes, failed claims, effect panics and overflow retain their report verdicts.

R25 requires Spell on all four Machine value types. The DSL implements bool, every integer type, Outcome, Option, unit and tuples through arity twelve, references, strings, chars, slices, vectors and arrays. imbl is not a DSL dependency, so its conditional implementation remains outside this task. Handwritten fixture and Kettle implementations retain their actual Rust type, variant and field names. Reports and paths implement Spell directly; their previous Display implementations and the Debug model-value renderers are removed. assert_verified retains its explicit assertion over model verdicts and spells its diagnostics through Spell.

R27 covers explore, refuted, verify_all and rows with Result returns. The zero_budget fixture pattern-matches Refusal::Budget { given: 0 } and snapshots its actual error text. invalid_state pins Refusal::State for an unadmitted id. spell_values renders structured state, action, domain outcome and fact types without Debug or Display, including a valid state with no outgoing rows. DSL leaf goldens cover literal spelling and escaping; trybuild diagnostics pin missing Spell for state and action types. The existing oracle keeps every behavioral assertion unchanged and only adapts to Spell and Result.

baseline: green from make model2-check before any implementation edit at 1aa4ce8959f91b249677de0adc79151f6697993c. Zero-budget and invalid-state fixture API controls failed before implementation; the DSL missing-Spell tests first failed because unsupported Models compiled, and the leaf test first failed because Spell was absent. The joined fixture run failed on the expected canonical report spelling mismatch. make model2-update generated the accepted insta payloads, aliases, metadata and trybuild diagnostics.

Strict comparison of all 56 previous golden payloads accepts exactly the declared spelling substitutions for 55 payloads and the intentional zero-budget refusal text change for one. Counts, state/action/choice/path order and verdicts are unchanged. The comparison script is .flow/tmp/fn210-12-golden-parity.py. Final verification passed make model2-check, PROPTEST_CASES=256 oracle, release checker fixtures including effect panic and overflow, cargo deny check bans and the requested Debug-format scan. Logs are .flow/tmp/fn210-12-closure-{gate,oracle,release,bans}.log. The first gate receipt attempt returned NO_RECEIPT because concurrent owner edits left MILESTONES.md dirty. After the owner's separate checkpoint, the clean tree accepted .flow/tmp/green-receipts/ed1c8279-unittest.json. No receipt skip or inferred pass is claimed.

stage: impl-review - ran (SHIP; three first-round lenses; codex:gpt-6.1-sol:high; same-family review)
Tier: project implementer gpt-6.1-sol at medium (judge unavailable no_key)

Review receipt: /tmp/impl-review-receipt-8f37faba39e2-fn-210-rust-prototype-kettle-on-a-trait-engine.12.json. Each lens returned SHIP without findings. Their build attempts were inconclusive because the read-only sandbox could not open Cargo's build lock; the worker's fresh final gates passed afterward. Memory capture did not run because no NEEDS_WORK-to-SHIP transition occurred.

The DSL/sample lane ran in an isolated task-owned worktree and joined before staging or gates. Its temporary copies and clean branch were removed after their implementation was committed on umpire and branch ancestry was verified. The task introduces no macro, progress, wasm or shared-fixture migration behavior. Implementation checkpoint 6697652276 also preserves concurrent owner fn-210 planning and MILESTONES changes. Owner checkpoint c7d5baf54a contains the separately approved fn-211 plan; ed1c8279d8 contains the owner's R30 timing helper, README, Makefile and later planning. The .12 source remains exactly the reviewed implementation at 6697652276.

stage: plan-sync - skipped(config: planSync.enabled != true)

R30 timing output from make model2-time (exit 0), recorded in .flow/tmp/fn210-12-timing.log. This is the owner's timing helper; the separately observed command exit codes above establish correctness.

```text
step (warm)                         time
fmt --check                       0.075s
clippy -D warnings                0.085s
doc (warnings denied)             0.083s
test --workspace                  0.301s
wasm32-wasip2 build               0.046s
inner loop (checker edit+test)    0.748s
gate total                         0.59s  (budget 30s)
inner loop                        0.748s  (budget 3s)
```
## Evidence
- Commits: 669765227666d16956e559f1f5205a0a5930f56e, c7d5baf54aca60822e8ebae131f02e37d631a34f, ed1c8279d8c7e92737c9cb2998f330160893cfa1
- Tests: make model2-check, make model2-update, PROPTEST_CASES=256 mise exec -- cargo test --manifest-path model2/Cargo.toml -p umpire-checker --test oracle, mise exec -- cargo test --manifest-path model2/Cargo.toml --release -p umpire-checker --test fixtures, mise exec -- cargo deny --manifest-path model2/Cargo.toml check bans, python3 .flow/tmp/fn210-12-golden-parity.py 1aa4ce8959f91b249677de0adc79151f6697993c, grep -rn '{:?}' model2/umpire/testing model2/umpire/checker/src (no matches), make model2-time
- PRs: