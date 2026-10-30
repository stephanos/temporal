# Deadline countdowns

Status: measured fn-253.2 spike verdict, 2026-10-29. The proposed abstraction compiles natively; its rewritten Models are **not yet admitted** by the front end.

## Decision

Adopt library-owned `umpire_dsl::countdown::Ticks<const N: u8>` and `Remaining<const N: u8>`, with the same Model-selected compile-time bound `N` in `1..=255`. Keep Temporal timeout semantics as a documented authoring pattern with named per-Model effects. Do not add a registered project library for this use.

The shared pair ties positive configured values (`1..=N`) and live remaining values (`0..=N`) to one bound. The trait alternative preserves today's local nominal types, but does not tie their ranges together and still needs trait/associated-type admission. Neither representation is currently admitted in its proposed call/type forms. The shared pair needs imported nominal/const specialization; choosing it is a scoped design decision for .3/.4, not a claim that generic input support exists.

Local helpers alone do not provide the proposed abstraction today. A no-argument immutable `self.armed()` helper is admitted. `Self::arm(value)`, `Self::tick(value)` and positional receiver-helper arguments in handlers are refused. Initial-only calls are admitted because computed starts are not body-lowered; that success does not establish handler or claim support.

## Fixed native surface for .3

All items belong in one `umpire-dsl` module, `author/dsl/src/countdown.rs`. The spike separates the comparative trait adapter, shared types and chosen functions only to isolate probes; those scratch module names are not public API recommendations.

```rust
pub struct Ticks<const N: u8>(pub u8);
pub struct Remaining<const N: u8> { pub ticks: u8 }
// Both implement Finite with the domains specified above.
// Both expose contains(value: u8) -> bool for their respective ranges.

pub fn arm<const N: u8>(configured: Option<Ticks<N>>) -> Option<Remaining<N>>;
pub fn tick<const N: u8>(remaining: Option<Remaining<N>>) -> Option<Remaining<N>>;
pub fn armed<const N: u8>(remaining: Option<Remaining<N>>) -> bool;
pub fn expired<const N: u8>(remaining: Option<Remaining<N>>) -> bool;
pub fn within<const N: u8>(remaining: Option<Remaining<N>>, configured: Option<Ticks<N>>);
pub fn descends<const N: u8>(after: Option<Remaining<N>>, before: Option<Remaining<N>>);
```

`arm` copies a present configured count and preserves absence. `tick` preserves absence, saturates at zero and subtracts one otherwise. `armed(None)` and `expired(None)` are false; `Some(0)` is armed and expired. Raw public fields permit forged values: `Finite` enumeration is not constructor validation, and the value functions do not sanitize invalid inputs.

The unit-returning `within` and `descends` laws use `check!` in a claim frame, following `closable`. `within` checks `N > 0`, checks every present configuration is in `1..=N`, and checks every present remaining value is positive and no greater than its present configuration. An absent remaining value may accompany an absent or valid configuration. This is for Models clearing expiry in the same step; it deliberately rejects states retaining an armed zero. `descends` checks exactly `after == tick(before)`. It does not cover retries, reset, terminal clearing, range validity, time, fairness or timeout priority. The Model must select the applicable transition and separately state those rules.

The measured candidate uses guarded presence tests and eager `unwrap_or`, not Option combinators or Option patterns. Its full source is [countdown-proposed.rs](../../.flow/artifacts/fn-253-reusable-deadline-countdowns-for-models/task2/probes/countdown-proposed.rs). Comparative scratch predicates ending in `_holds` are boolean mechanics sketches, not the chosen delegated-law API. Native tests evaluate the chosen laws through actual `claim::evaluate` frames; an unframed function call is not evidence that a claim held.

## Measured experiment

Baseline: `02d4f55660e4da54612d703f16bd92170c742539`. The fn-246.15 timeout/retry closure `9eefdd2f94cf890cfc7ad37b677bd5abcbb94e1f` is an ancestor. The independent fn-246.13 cancellation correction is not integrated. In this baseline, pending local expiry still closes Canceled; later production work must reanchor after both required closures. This spike changes no production Model or golden and supplies no migration baseline.

The runner verified 21 exact outcome expectations and expected message fragments: four admitted controls, 16 Model refusals, and one deliberately malformed parse control. Four complete activity variants compile natively: unchanged baseline, local helpers, local Finite wrappers plus trait adapter, and imported shared const-generic types. The candidate project-helper body compiles in the native harness. The whole `umpire-temporal` crate was not checked for that candidate.

This distinction is observable: the `#[machine]` macro calls `front::admit_checks`, embeds its Result, and generates `Interprets` only on success (`author/dsl/macros/src/lifted.rs:10–24`). Successful Rust compilation can therefore coexist with a source-admission refusal. No interpreted execution or TLC equivalence is claimed for these refused Models.

Native controls verify configured/remaining Finite domains, absent/zero/positive/maximum values, raw invalid values, invalid `N=0`, actual unit-law verdicts, simultaneous-expiry priority and global-budget preservation on retry. The standalone Rust mechanics suites pass one and two tests. The finite/law harness also checks bound 255 and the final missing-configuration/absent descent cases. These controls establish the candidate mechanics, not full activity behavior equivalence or a checker's answer.

The counted preparer rewrites initial/worker arming (Poll, retry and Heartbeat), deadline/decrement, three split invariants (`policy_in_range`, `attempt_within_policy`, `countdown_bounds`), descent laws and Tick priority. It preserves the baseline's timeout/cancellation branches. Every textual mutation asserts its match count. [Rewrite counts](../../.flow/artifacts/fn-253-reusable-deadline-countdowns-for-models/task2/rewrite-counts.json) and [line counts](../../.flow/artifacts/fn-253-reusable-deadline-countdowns-for-models/task2/line-counts.json) bind those edits.

| Complete Model source | Lines |
| --- | ---: |
| Baseline | 730 |
| Local value helpers | 580 |
| Local wrappers + trait library adapter | 570 |
| Shared bounded types + source library calls | 553 |

These are unformatted candidate Model line counts, excluding the library and harness. They are neither a project-wide size saving nor performance/equivalence evidence. Production invariant coverage, forged-zero boundaries, mutant ownership and equivalence remain later task obligations.

## Observed admission results

Positions below are one-based in generated probe files reconstructed by `prepare.py`. Paths in the final column are under `model/umpire/front/src/`. Each full rewrite reports its first refusal; independent cases expose the later syntax/type classes rather than treating that first error as exhaustive. The parse-error case checks the reporter and does not request a compiler change. Alias and unused-generic controls do not promise new supported syntax.

| Probe | Result / input position | Exact message | Front-end owner |
| --- | --- | --- | --- |
| `activity-baseline.rs` | admitted, — | — | `positive control` |
| `activity-local.rs` | model_refusal, 213:29 | unsupported expression `Self :: arm (world . start_to_close)` | `body/expr.rs:300–325` |
| `activity-library.rs` | model_refusal, 145:5 | trait or generic impl in machine module is unsupported | `shape.rs:161–177` |
| `activity-shared.rs` | model_refusal, 26:32 | unsupported type `Option<Ticks<3>>` | `body/typer.rs:10–40; body/optional.rs:62–86` |
| `control.rs` | admitted, — | — | `positive control` |
| `local-arm.rs` | model_refusal, 23:57 | unsupported expression `Self :: arm (world . timeout)` | `body/expr.rs:300–325` |
| `local-arm-initial.rs` | admitted, — | — | `positive control` |
| `local-tick.rs` | model_refusal, 23:57 | unsupported expression `Self :: tick (self . left)` | `body/expr.rs:300–325` |
| `local-generic.rs` | model_refusal, 15:1 | bounded newtypes must be nongeneric | `body/optional.rs:10–12` |
| `local-alias.rs` | model_refusal, 15:1 | unsupported declaration in machine module | `shape.rs:70–79 (keep refused)` |
| `local-trait.rs` | model_refusal, 16:5 | trait or generic impl in machine module is unsupported | `shape.rs:161–177` |
| `unused-generic.rs` | admitted, — | — | `positive control` |
| `receiver-args.rs` | model_refusal, 25:9 | only pure helper calls with an optional trailing World are admitted | `body/mod.rs:224–245` |
| `library-assignment.rs` | model_refusal, 23:57 | unsupported expression `countdown :: tick (self . left)` | `body/expr.rs:300–325` |
| `library-condition.rs` | model_refusal, 23:48 | unsupported expression `countdown :: armed (self . left)` | `body/expr.rs:300–325` |
| `library-within-law.rs` | model_refusal, 26:56 | claim body supports check! and conditional checks only | `body/claim.rs:141–176` |
| `library-descends-law.rs` | model_refusal, 26:63 | claim body supports check! and conditional checks only | `body/claim.rs:141–176` |
| `project-library.rs` | model_refusal, 23:48 | unsupported expression `temporal_countdown :: armed (self . left)` | `body/expr.rs:300–325; library.rs:4–5` |
| `shared-ticks.rs` | model_refusal, 10:44 | unsupported type `SharedTicks<3>` | `body/typer.rs:10–40; body/optional.rs:62–86` |
| `shared-remaining.rs` | model_refusal, 12:41 | unsupported type `SharedRemaining<3>` | `body/typer.rs:10–40; body/optional.rs:62–86` |
| `parse-error.rs` | parse_error, 2:16 | cannot parse string into token stream | `syn parser control; no front-end change` |

[Exact observations and source snippets](../../.flow/artifacts/fn-253-reusable-deadline-countdowns-for-models/task2/observations.json) retain all input positions/messages. [Source survey](../../.flow/artifacts/fn-253-reusable-deadline-countdowns-for-models/task2/source-survey.md) distinguishes additional source-inferred limits from these measured refusals.

## Front-end work for .4

Read the generic countdown functions and laws from their authored source, as `closable` is read today. Do not lower them by function name or add Temporal-specific includes to the framework. Current `library.rs` has only the fixed closable/phased sources; claim delegation recognizes closable only.

The chosen form needs canonical imported bounded-type identities instantiated with the literal `N`, matching positive/zero-inclusive domains, constructor/payload/default/encode/decode handling, and refusal of invalid bounds. Manual native `Finite` implementations in the prototype are not already a front-end source-registration mechanism. Admit their authored finite-domain meaning or provide equivalent authored range metadata; do not infer those domains solely from `Ticks`/`Remaining` names. Value and law calls need shared argument/result frames, const substitution and source-body lowering, including `descends` calling `tick`. Existing Optional and Bounded IR are reusable where their semantic contracts fit; no new runtime value form was demonstrated necessary by this spike.

The local-wrapper trait option avoids imported nominal identities but still needs trait/associated-type and positional binding support. It is a measured alternative, not a free path through today's lifter. Extending local helpers alone likewise needs associated-call or positional-helper admission and would still duplicate the abstraction across Models. Unsupported aliases, globs and nesting remain explicit refusal controls under the parent spec.

## Temporal layer and independence

Choose a README authoring pattern, with `model/temporal/feature/activity/README.md` as the deadline/arming/expiry authority and the corrected activity Model as its executable consumer. The activity Model's per-Model expiry effects are `time_out` and `time_out_attempt`, alongside `deadline`, `retry` and `clear_terminal`. These effects retain the committed fn-246.13/.15 corrections. The pattern owns arming/reset, global-first expiry priority, retry rearming without replenishing the global budget, and corrected cancellation outcomes. These rules do not belong in generic countdown arithmetic.

The alternative has a real prospective compiled owner: crate `umpire-temporal`, `model/temporal/realization/Cargo.toml`, declared from its `src/lib.rs`. The disposable installer declares `pub mod countdown_spike;` there and supplies its source; the native control compiles the same body via a path module. That does not add front-end visibility. A real shared project library would require a generic source/identity registry passed through `front::read/admit`, proc-macro embedding and call resolution. There is no current registration API to name. That cost is not justified for this pattern, and no uncompiled helper file is proposed as an implementation.

The independent protocol may share generic mechanics under the existing spec. Whether it may share Temporal deadline semantics remains the parent's explicit owner decision. This spike recommends keeping that semantic layer independently authored; it does not authorize sharing it or hold R1's design/admission findings open pending that separate decision.

## Reproduction and limits

Durable assets: [probes/](../../.flow/artifacts/fn-253-reusable-deadline-countdowns-for-models/task2/probes/), including `prepare.py`, `install-stubs.py`, `run.sh`, the strict expectation checker, authored sources and pinned Cargo.lock. Generated probe sources and logs are archived alongside the results; scratch is not their only authority. Source hashes and tool identities are in the task evidence summary.

From this repository, with the recorded revision available and pinned tools/dependencies installed, create a fresh disposable checkout (refuse an existing path):

```sh
git worktree add --detach .worktrees/fn2532-spike 02d4f55660e4da54612d703f16bd92170c742539
mkdir -p .worktrees/fn2532-spike/.flow/tmp/fn253-spike
cp .flow/artifacts/fn-253-reusable-deadline-countdowns-for-models/task2/probes/* .worktrees/fn2532-spike/.flow/tmp/fn253-spike/
cd .worktrees/fn2532-spike
env -u BASH_ENV python3 .flow/tmp/fn253-spike/install-stubs.py
env -u BASH_ENV CARGO_BUILD_JOBS=1 CARGO_INCREMENTAL=0 CARGO_PROFILE_DEV_DEBUG=0 bash .flow/tmp/fn253-spike/run.sh
```

Rust is pinned to 1.99.1 (`58cf8f9c3`, aarch64-unknown-linux-gnu, LLVM 23.1.1); Cargo resolves only the archived lock with `--offline --locked`, and mise auto-install is disabled. The measured run used a task-owned target through `CARGO_TARGET_DIR`; it needs no JVM, checker result cache, server, credentials or network. The runner keeps its complete admission and per-variant native logs under `output/`; the reporter's exit status alone never establishes admission. `check-results.py` verifies every expected outcome and message and rejects missing/duplicate cases.

The original spike worktree is removed at handoff, with its exact disposable source modifications retained in evidence. For a reconstruction checkout, archive any results/source changes first, then use ordinary clean-worktree removal under repository policy. Do not force-remove another session's checkout. Full gates, goldens, interpreter/TLC agreement, the actual library implementation and activity migration remain .3–.8/.9/.10 work.
