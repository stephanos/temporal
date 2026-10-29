# fn-253.2 source admission survey

Source-only observations at `02d4f55660` (2026-10-28). Paths below are relative to the Temporal repository. No Cargo, builds, tests, JVM, downloads, production edits, worktrees or Flow mutations ran in this survey. Probe outcomes and exact rewritten-source coordinates belong to the conductor's measured probes. These are admission-path predictions, not executed refusals or a review verdict.

Read task/spec through `flowctl cat fn-253.2`, `cat fn-253`, plus `cat fn-246.13`; inspected root AGENTS, model README, UMPIRE4 spec/vision/module map. The installed wrapper's BASH_ENV resets child-shell cwd. The reliable read route is `cd <temporal> && env -u BASH_ENV python3 /home/agent/.codex/scripts/flowctl.py cat <id>`.

## Current support and refusal owners

In this table `F` means `model/umpire/front/src/`. The first refusal depends on declaration order and exact probe shape. Each row gives the source node used to locate the predicted refusal.

| Proposed source | Current path / predicted refusal | Owner and location node |
| --- | --- | --- |
| `fn arm(ticks: ...)` as a module free function | Module functions are allowed only when selected by an action `#[domain]`. Otherwise `unsupported declaration in machine module`. | `F/shape.rs:34` computes domain names; `:70`–`:79` rejects at the function item. |
| `Self::arm(x)` / `Model::arm(x)` in assignment, local initializer or condition | Call expressions dispatch only optional constructors, bounded calls and vector constructors. An ordinary associated helper reaches `unsupported expression `...``. | `F/body/expr.rs:300`–`:325`, call node. Effect assignment `F/body/effects.rs:177`, local initializer `:144`, condition `:204` all use this expression dispatcher. |
| `self.arm(x)` with immutable receiver and value argument | Whole-state method lookup succeeds, but `helper_env` rejects positional arguments with `only pure helper calls with an optional trailing World are admitted`. | `F/body/expr.rs:331`–`:343`; `F/body/mod.rs:224`–`:245`, method declaration node (`:235`). |
| `self.armed()` / `self.armed(world)` | Admitted when immutable, non-generic and a single expression body; only the selected trailing World can be passed. Value returns already exist in this narrow form. | `F/body/expr.rs:331`–`:343`; `F/body/mod.rs:239`–`:245`, `:291`–`:295`. Wrong World argument locates at that argument and says `helper World argument must be the selected World parameter`. |
| Mutable receiver in expression | `only pure helpers are admitted in expressions`. | `F/body/expr.rs:334`–`:335`, call node. |
| Pure helper with `let`, statements then result, or explicit return | `expected a single expression body`. A nested `if/else` expression works; a multi-statement block does not. | `F/body/mod.rs:291`–`:295`, whole helper block; value-producing if also requires else at `F/body/expr.rs:240`–`:251`. |
| Enum helper with arguments | `only parameterless pure enum helpers are admitted`. | `F/body/expr.rs:353`–`:368`, call node. Bounded wrapper receivers are not enums and first reach `unsupported method `name`` at `:353`–`:354`. |
| Generic local helper | `async, unsafe or generic helper is unsupported`. | `F/shape.rs:214`–`:221`, method item. Recursively expanding admitted expression helpers over depth 32 refuses at `F/body/expr.rs:35`–`:36`. |
| `self.helper(...)` as an effect statement | Requires `#[effect]`, receiver and unit result; args remain optional trailing World only. | `F/body/effects.rs:270`–`:303`. `only #[effect] calls with an optional trailing World are admitted` at call (`:277`); `effect helpers must return unit` at result type (`:285`); argument binding reuses `F/body/mod.rs:224`. |
| `countdown::arm/tick/expired/armed` in expressions | No ordinary library call dispatch exists; call reaches `unsupported expression`. | `F/body/expr.rs:300`–`:325`. Existing sources are only CLOSABLE and PHASED in `F/library.rs:4`–`:5`. |
| `countdown::within/descends(...);` as claim statement | `claim body supports check! and conditional checks only`. Current library exception recognizes only DSL paths beginning `closable::`. | `F/body/claim.rs:141`–`:146`, statement node; `:165`–`:176` resolves the existing law. In `check!(countdown::...)`, expression dispatch refuses instead. |
| Claim-local `let` or ordinary value helper statement | Same claim-body refusal. Conditional claim else-if refuses separately as `claim else-if is unsupported`. | `F/body/claim.rs:109`–`:147`, statement/else expression nodes. |
| Unknown `closable` member / wrong arity | `unknown `umpire_dsl::closable` law `name`` / ``umpire_dsl::closable::name` takes N arguments`. | `F/body/claim.rs:174`–`:179`, call node. Library-body failures relocate to call with `in `name`: ...` (`:230`–`:232`). |
| Imported `Ticks<3>` / `Remaining<3>` or nongeneric external nominal wrapper | Imports do not populate bounded descriptors. Type dispatch accepts only local bare nominal bounded names, then enums. Expected `unsupported type `...`` at selected field type. | `F/body/typer.rs:10`–`:40`; `F/shape.rs:83`–`:87` registers local derived structs; `F/body/optional.rs:62`–`:86` checks optional payload through `ty`. An earlier local declaration can change the first refusal. |
| Local const-generic `#[derive(Finite)] struct Ticks<const N: ...>` | `bounded newtypes must be nongeneric`. | `F/body/optional.rs:10`–`:12`, struct item. Derive's bounded range additionally requires primitive backing, inclusive literal bounds (`F/declare/fields.rs:103`–`:130`). `#[inputs(1..=N)]` fails `a Finite range bound must be an integer literal` at N (`:123`). |
| Trait over local Finite wrappers | Existing wrappers retain usable nominal descriptors. New trait impl still refuses `trait or generic impl in machine module is unsupported`; only known Phase/Phased impls are admitted. | `F/shape.rs:161`–`:177`, impl item. A custom `impl Finite` has the same failure. Trait declaration itself fails `unsupported declaration in machine module` at `:79`. |
| Optional arithmetic / map / unwrap / expect / patterns | Only `is_some`, `is_none`, eager `unwrap_or` are optional methods. Others say `unsupported optional method; use is_some, is_none or eager unwrap_or`. | `F/body/optional.rs:261`–`:294`; generic optional method refusal at `:270`–`:271`. `matches!(optional, ...)` refuses `optional patterns are unsupported; use presence tests` in `F/body/expr.rs:404`–`:406`. |
| Wrapper `Ticks::arm(...)` / `Ticks::<N>(...)` | Recognized local bounded name invokes its restricted dispatcher. Only direct tuple construction and generated `contains` are admitted. | `F/body/optional.rs:213`–`:258`. `unsupported bounded associated call` at call (`:258`); any generic/path qualification/shadowing refuses `bounded constructor requires the local nominal type` (`:225`–`:230`); arity says `bounded construction and contains require one backing value` (`:232`–`:233`). |
| Aliases / glob / nested library source | No general source resolver. Machine nested modules/type aliases fail declaration admission; non-DSL globs fail macro-rebinding guard. Existing renamed imports are enumerated, so future blanket alias refusal must be explicit. | `F/shape.rs:70`–`:79`, `:472`–`:495`, `:504`–`:513`. |

`Diagnostic` reports one-based line/column and message (`model/umpire/ir/src/lib.rs:155`–`:161`), not a public Span. `Locate::at` chooses the first token of the source node (`F/lib.rs:44`–`:55`). Report measured line/column from the exact probe bytes; the owner anchors above identify the code issuing it.

## Starts and native compilation

`#[starts] initial` is special. The front records `Start::Computed` and skips inspecting its body (`F/shape.rs:223`–`:225`, `:259`–`:260`, `:365`), while native generation invokes it (`model/umpire/author/dsl/macros/src/machine.rs:680`). A local or library call used only inside initial therefore does not prove expression admission in handlers, effects or claims.

The actual macro path is `model/umpire/author/dsl/macros/src/lib.rs:59`–`:69`, declaration read then native `generate`, then `lifted::embed`. `macros/src/lifted.rs:17`–`:24` runs `front::admit_checks` during expansion and embeds its Result. Only Ok generates `Interprets` (`:19`–`:20`). A subset Err is retained as data rather than propagated by `?`; declaration errors can still abort the macro before native generation. Thus a successful native compilation can coexist with a recorded subset refusal, although compilation does execute front admission. Native compilation is not independent of the front path.

## Temporal helper ownership and minimum future work

| Candidate | Concrete current owner or missing admission |
| --- | --- |
| Shared compiled Temporal helper | Existing crate `umpire-temporal`, manifest `model/temporal/realization/Cargo.toml:2`. Its `src/lib.rs:59`–`:70` uses path modules for foundation and feature/activity/temporal.rs; that file includes model.rs at `model/temporal/feature/activity/temporal.rs:4`. A future helper at `model/temporal/capabilities/system/deadline.rs` needs a real module declaration from this crate, e.g. `#[path = "../../capabilities/system/mod.rs"] pub mod system_capabilities;` plus `mod deadline` under that module. A loose .rs file has no compiled owner. This describes a possible wiring, not an existing module. |
| Generic project-library registration | None exists in front::read/admit signatures (`F/lib.rs:74`, `:80`, `:108`) or Environment (`F/body/mod.rs:84`–`:108`). `library.rs` hardcodes two include_str sources. A compiled module declaration alone cannot supply source resolution. A future caller-supplied registry must pass canonical identity + authored source/AST through direct source reads and proc-macro embedding, resolve selected imports, type argument/result frames, inline source bodies and relocate errors at the call. Keep Temporal source in its owner; do not add Temporal-specific include_str into framework. |
| README pattern | `model/temporal/feature/activity/README.md:11`–`:38` already owns arming/reset and priority/cancellation explanations. Consume named per-Model effects in existing activity `deadline`, `retry`, `expire_attempt`, `clear_terminal` (`model.rs:237`, `:289`, `:301`, `:314`), preserving independent protocol authorship. New effect names remain a spike decision. Existing README pending outcomes are pre-fn24613 semantics and need reanchoring after that correction. |

Minimum work for local value helpers is ordinary associated-call dispatch plus positional parameter typing/binding in `helper_env`, immutable receiver enforcement and source-result typing. Existing single-expression helpers can inline nested if/else; statement-bodied pure functions need a sequential pure-body evaluator too. Minimum source-library work adds generic value/law call dispatch, bound generic parameter substitution and optional nominal conversion, sharing argument/result typing between expressions and claims. The local-wrapper trait form avoids replacing the current Bounded descriptor identity, but still needs a declared source-library trait/method contract and impl admission. Library-owned const-generic wrappers additionally need canonical imported nominal identities with evaluated N, instantiated backing/ranges, matching constructor/payload handling and encode/decode paths. These are future admission changes only; no .3/.4 implementation was made.

## Baseline counts, ancestry and independence

`model/temporal/feature/activity/model.rs` has 730 lines. Inclusive function spans exclude attributes and surrounding blank lines.

| Function | Lines | Count |
| --- | --- | --- |
| initial | 146–164 | 19 |
| worker | 182–234 | 53 |
| deadline | 237–286 | 50 |
| retry | 289–298 | 10 |
| expire_attempt | 301–311 | 11 |
| clear_terminal | 314–320 | 7 |
| decrement | 323–355 | 33 |
| state_bounds | 358–424 | 67 |
| countdowns_decrement | 497–551 | 55 |
| tick_priority | 588–658 | 71 |

Four configured budgets and four live optional budgets occur at model.rs:32–38 and :127–133. Exact single-line substring counts are 51 `unwrap_or(Remaining { ticks: 0 })` and 16 `unwrap_or(Ticks(1))`; multiline formatting means these are not semantic call counts. No rewritten-source counts were measured here.

`9eefdd2f94cf890cfc7ad37b677bd5abcbb94e1f` (fn-246.15 closure) is an ancestor of baseline `02d4f55660`. `aaba137190` (independent activity protocol and cancellation correction source) is not. Baseline `expire_attempt` still turns pending local expiry into Canceled (`model.rs:301`–`:304`); nonpending expiry retries below cap (`:305`–`:306`), confirming .15 behavior before .13's approved TimedOut correction. The retained README similarly says pending failure/local expiry closes Canceled (`README.md:21`, `:34`). Do not promote this older behavior into the abstraction's semantics.

The fn-253 spec's first Parked unknown remains an owner question. Generic countdown mechanics may be shared; copying Temporal deadline semantics into an independently authored protocol risks common-source agreement. fn-246.13 requires primary-protocol provenance rather than feature edges. This survey makes no permission decision about protocol use of the Temporal layer.
