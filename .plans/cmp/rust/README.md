# Rust sample

The two Models of `SPEC.md`, authored in Rust against a sketched `umpire` crate. Nothing here has
been compiled; the framework bodies are `todo!()` where the spec allows and the samples are written
as they would be in a crate laid out like this:

```
umpire/            crate `umpire`        lib.rs: Step, Machine, Finite, Refinement, Property, ...
umpire/macros.rs   crate `umpire-macros` the proc-macros, re-exported by `umpire`
worker.rs          crate `temporal-model`, src/worker.rs
nexus_caller.rs    src/nexus_caller.rs   Model 1
standalone_activity.rs                   Model 2
pins.rs            tests/pins.rs         the pins, plus trybuild fixtures under tests/ui/
```

## The DSL mechanism

Procedural macros over `syn` and `quote`, one per block of the Lean grammar: `entity!`, `action!`,
`observation!`, `machine!`, `compose!`, `property!`, `scenario!`, `limits!`, `query!`, `set!`, and
`#[derive(Finite)]`. Step functions and predicates are plain Rust: a `fn` returning
`Vec<Step<S, O, F>>` with an exhaustive `match`, a non-capturing closure for `holds:`.

Why proc-macros rather than builders: the declarative blocks are the part that should read like
the Lean, and a builder chain would put `.step("handlerReply", ...)` strings where the Lean has
names. A macro keeps every name a Rust path, and that is the whole trick of the design:

- **What a block can say about itself, the macro checks.** An `unobservable:` entry that is not a
  timer, a timer with no `steps:` line, a duplicate step, `refines:` without `map:`, a `set!` whose
  purpose and fields disagree, a `holds:` closure with the wrong arity for its claim. Each is a
  `syn::Error::new_spanned(token, message)`, so the caret sits under the author's token and the
  message is ours.
- **What lives in another item, rustc checks.** A proc-macro sees one invocation and nothing else,
  so `machine!` cannot know which `action!` blocks exist. Instead, `steps: { handlerReply: f }`
  expands to `handlerReply::bind(f)` with `quote_spanned!` at the author's span. An undeclared
  action is E0433 on that token; a step function whose signature disagrees with the action's
  declared inputs is E0593 on the function name, because `bind`'s parameter list is generated
  from the `input:` fields. Same for phases (`starts: [Unscheduled]` becomes
  `<<S as Phased>::Phase>::Unscheduled`), facts under `evidence:`, and every action a `scenario!`
  or `when:` names.

The generated items keep the spec's names exactly. `action! { handlerReply .. }` emits
`pub struct handlerReply;` (Diesel's `table!` names column types the same way), `machine! {
nexusProtocol .. }` emits `pub static nexusProtocol: LazyLock<Machine<..>>` and, because Rust keeps
types and values in separate namespaces, a twin `pub enum nexusProtocol {}` implementing `Typed`
so that `property!` and `query!` can spell their static's type without knowing the state type.

## Where each check runs

| Check | When | How |
|---|---|---|
| Exhaustive `match` in a step function | `cargo check` | rustc E0004, missing pattern spelled out |
| Step function signature vs declared inputs | `cargo check` | rustc E0593/E0631 at the `steps:` line |
| Undeclared action, phase, fact, entity, query, limits | `cargo check` | rustc E0433/E0599/E0425 at the token |
| Block shape (timers, unobservable, set purpose, claim arity) | `cargo check` | proc-macro `syn::Error` at the token |
| State-space size | `cargo check --tests` | `Finite::CARDINALITY` is a `const`; `const _: () = assert!(..)` in `tests/pins.rs`, no crate |
| Finite state table, ends, action catalog, reachability | `cargo test` | `Machine::table()` built once via `LazyLock` |
| Refinement (every protocol row is a stutter or a product transition between its mapped states) | `cargo test` | `Machine::refinement().rejected == None` |
| Search: find and verify Queries | `cargo test` | `Query::run()` |
| A product Property read on a protocol Scenario | `cargo test` | `Admission` error from `Query::run()` |
| Error message text | `cargo test` | `trybuild` compile-fail fixtures with `.stderr` files |

Lean runs the last five at elaboration. Rust cannot: `const fn` has no heap, a `Machine` boxes its
step functions, and the refinement needs both tables. What it can do at compile time is anything
expressible as a `const`, which covers the counts in the spec's pins but not the rows.

## What an author's error looks like

A missing arm in `protocol_handler_reply_step` (line numbers illustrative):

```
error[E0004]: non-exhaustive patterns: `Reply::HandlerError { retryable: false }` not covered
   --> src/nexus_caller.rs:318:11
    |
318 |     match reply {
    |           ^^^^^ pattern `Reply::HandlerError { retryable: false }` not covered
    |
note: `Reply` defined here
   --> src/nexus_caller.rs:52:10
    |
52  | pub enum Reply {
    |          ^^^^^
...
57  |     HandlerError { retryable: bool },
    |     ------------ not covered
    = note: the matched value is of type `Reply`
help: ensure that all possible cases are being handled by adding a match arm with a wildcard pattern or an explicit pattern as shown
    |
323 ~         HandlerError { retryable: true } => vec![Step { .. }],
324 ~         Reply::HandlerError { retryable: false } => todo!(),
    |
```

A `steps:` line naming an undeclared action (the macro forwarded the token, rustc resolved it):

```
error[E0433]: failed to resolve: use of undeclared type `handlerRepli`
   --> src/nexus_caller.rs:427:9
    |
427 |         handlerRepli: protocol_handler_reply_step,
    |         ^^^^^^^^^^^^ use of undeclared type `handlerRepli`
    |
help: a struct with a similar name exists
    |
427 |         handlerReply: protocol_handler_reply_step,
    |         ~~~~~~~~~~~~
```

A step function bound to the wrong action (`handlerReply: transport_fault_step`):

```
error[E0593]: function is expected to take 2 arguments, but it takes 1 argument
   --> src/nexus_caller.rs:427:23
    |
427 |         handlerReply: transport_fault_step,
    |                       ^^^^^^^^^^^^^^^^^^^^ expected function that takes 2 arguments
    |
note: required by a bound in `handlerReply::bind`
```

A block that disagrees with itself (the macro's own `syn::Error`):

```
error: `backof` is listed under `unobservable:` but is not one of this machine's timers (backoff, scheduleToClose, scheduleToStart, startToClose)
   --> src/nexus_caller.rs:411:20
    |
411 |     unobservable: [backof]
    |                    ^^^^^^
```

A Query that does not find its claim, or a refinement that does not hold, is a failed test. This is
what the activity refinement pin printed under the first draft of `SPEC.md`, which mapped
`pauseRequested` to `paused`; the spec was revised on the product side and the pin now passes:

```
---- activity_pins::refinement_passes stdout ----
assertion `left == right` failed
  left: Some(RefinementError { row: "pauseRequested-1-unset-unset-unset-control-unpause", reason: "maps paused -> started on control-unpause and activityProduct has no transition between them and it is not a stutter" })
 right: None
```

## Toolchain and loop

Stable Rust, `cargo`. Dependencies: `syn` (full), `quote`, `proc-macro2` for the macro crate;
`trybuild` as a dev-dependency (see "Libraries to leverage" for the rest). `cargo check` gives the compile-time tier
in about a second incrementally with a warm target dir; rust-analyzer expands proc-macros, so
every error above shows inline as you type, at the token. `cargo test` runs the tables, refinement
and search. `cargo expand` shows what a block became when the expansion itself is in doubt.

The slow edge is the macro crate: an edit to `macros.rs` recompiles `umpire-macros`, then `umpire`,
then every model crate, and `syn` with `full` features is a ten-to-twenty-second cold build.
Authoring a Model does not touch it; extending the DSL does.

## Honest notes

What Rust made easy:

- **Enums with payloads plus exhaustive `match`.** `Reply::HandlerError { retryable: bool }` is one
  variant and two classes, exactly the Lean granularity, and `#[derive(Finite)]` enumerates the
  payload so nothing is listed twice. A forgotten arm is a compile error that names the pattern.
- **Struct update syntax.** `ProtocolState { phase, ..*state }` is Lean's `{ state with phase }`.
- **`const` state literals.** `SUCCEEDED_ON_RETRY` is a `const`, and `Bounded<2>` gives the
  `Fin (attemptBound + 1)` type with a `const fn saturating_succ`.
- **Span-pinned errors for free.** Every name in a block is a Rust path at the author's span, so
  rustc's resolver is the DSL's name checker and rust-analyzer's UI is the DSL's UI.
- **Ordinary tests.** The pins are `#[test]` functions; `trybuild` pins error texts the way
  `#guard_msgs` does.

## Libraries to leverage

Maintenance checked on 2026-09-29 with `gh api repos/<owner>/<repo>` (`pushed_at`, `archived`).
The rule: no push in the last twelve months, or archived, is "not maintained, no-go". Only the
maintained rows are recommendations.

| Crate | Repository | Last push | Status | What it would replace |
|---|---|---|---|---|
| `proptest` + `proptest-state-machine` | proptest-rs/proptest | 2026-09-27 | maintained | The exploratory set's random walks over a `Machine`, and the reference-vs-system harness that drives a realization from the table: `ReferenceStateMachine` is our `Machine`, `StateMachineTest` is the Testpilot bridge. Not the exhaustive table or search. |
| `quickcheck` | BurntSushi/quickcheck | 2026-04-03 | maintained | Same role as `proptest` with less shrinking control and no state-machine layer; second choice. |
| `bolero` | camshaft/bolero | 2026-09-29 | maintained | One harness over `proptest`, libFuzzer and AFL for the exploratory set when a fuzzer-driven walk is wanted. |
| `petgraph` | petgraph/petgraph | 2026-09-27 | maintained | `Machine::reachable`, `stuck`, and the bounded search's graph walk: build the table as a `DiGraph<S, ActionClass>` and use its BFS, DFS, SCC and dominator algorithms. |
| `kani` | model-checking/kani | 2026-09-29 | maintained | Marginal. A bounded model checker for Rust code; it could prove `terminal_is_final` over every state without our enumerator, but for finite domains exhaustive `Finite::all()` already does that. |
| `loom`, `shuttle` | tokio-rs/loom, awslabs/shuttle | 2026-02-20, 2026-09-29 | maintained, not applicable | Both explore thread interleavings of Rust code under test. The Model has no concurrency of its own; they would matter only for a Rust realization runtime, which is out of scope. |
| `prost` + `prost-build` | tokio-rs/prost | 2026-08-03 | maintained | The protobuf Case format and the `schema:` strings: generated message types instead of string literals, so a renamed message is a compile error. |
| `tonic` | hyperium/tonic | 2026-09-29 | maintained | The gRPC client to the Go Testpilot runtime, if the search ever drives it directly. |
| `syn`, `quote`, `proc-macro2` | dtolnay/syn, dtolnay/quote, dtolnay/proc-macro2 | 2026-09-29, 2026-08-22, 2026-08-25 | maintained | The macro crate as sketched. |
| `darling` | TedDriggs/darling | 2026-09-11 | maintained | The hand-written `Parse` impls and `check_*` functions, if the DSL is restated as attributes (`#[machine(for = operation, starts = ..)]`): darling generates the field parsing, unknown-key errors and duplicate-key errors with spans. |
| `trybuild` | dtolnay/trybuild | 2026-09-08 | maintained | The `#guard_msgs` tier: compile-fail fixtures with pinned `.stderr`. |
| `insta` | mitsuhiko/insta | 2026-09-27 | maintained | The golden fixtures (`CallerExploratoryCoverage.json`): snapshot the table, the refinement rows and the coverage targets, review diffs with `cargo insta`. |
| `strum` | Peternator7/strum | 2026-03-07 | maintained | `Finite::key()` and `all()` for payload-free enums via `IntoStaticStr` and `EnumIter`. Our derive is still needed for payload variants and structs, so `strum` shrinks it rather than replaces it. |
| `serde_json` + `serde_json_canonicalizer` | serde-rs/json, evik42/serde-json-canonicalizer | 2026-08-08, 2026-02-20 | maintained | RFC 8785 canonical JSON of the table for the Behavior Fingerprint, so the hash is stable across map order. `serde_jcs` (l1h3r/serde_jcs, 2026-03-25) is the smaller alternative. |
| `sha2` or `blake3` | RustCrypto/hashes, BLAKE3-team/BLAKE3 | 2026-09-22, 2026-09-10 | maintained | The fingerprint hash over the canonical JSON. |
| `stateright` | stateright/stateright | 2025-07-27 | not maintained, no-go | Would have been the explicit-state model checker: `Model` trait, BFS/DFS checker, TLA+-style properties, a web explorer. Last push fourteen months ago, so the `Machine::table` and search stay ours. |
| `static_assertions` | nvzqz/static-assertions | 2023-11-18 | not maintained, no-go | `const_assert_eq!`. Unneeded: `const _: () = assert!(..)` is stable Rust, and `pins.rs` uses that. |
| `enum-iterator` | stephaneyfx/enum-iterator | 2025-09-08 | not maintained, no-go | `Sequence::all()` for enums and structs, close to `Finite::all()`. Just past the twelve-month line; `strum` plus our derive covers it. |
| `heck` | withoutboats/heck | 2025-08-09 | not maintained, no-go | Case conversion in the macros (`caller` to `Caller`). Two ten-line helpers in `macros.rs` do it. |
| `json-canon` | ahdinosaur/json-canon | 2023-05-23 | not maintained, no-go | Canonical JSON; superseded by `serde_json_canonicalizer`. |

The sketched framework stays on `syn`/`quote`/`proc-macro2` and `trybuild`; the first additions
a real build would make are `petgraph` for the search, `insta` for the fixtures, `prost` for the
Case format, and `proptest-state-machine` for the exploratory set.

What it made awkward:

- **Ownership noise in declarative code.** `&*syncCompletion as &dyn AnyQuery`, `vec![]`,
  `*state`, `Box<dyn Fn>`, `LazyLock` around every static, `+ Send + Sync + 'static` bounds. The
  Lean writes `queries: [syncCompletion, ...]` and the types are inferred; a Rust `static` needs
  its type spelled, which is why `machine!` grew `outcome:` and `facts:` lines the Lean does not
  have, and why every generated static has a `Typed` twin.
- **Qualified variants.** `ProtocolFact::NexusOperationCompleted` where Lean writes
  `.nexusOperationCompleted`. Glob-importing the input domains (`use Reply::*`) keeps scenarios
  short, but the phase and fact enums share variant names and must stay qualified.
- **`#[allow(non_camel_case_types)]` everywhere.** Keeping the spec's names exactly means lowercase
  types and camelCase statics, each with a lint allow the macro must emit.
- **A proc-macro cannot see across invocations.** The "undeclared action" error is rustc's message
  about a missing type, not ours about a missing action. A reader who has not seen the expansion
  may not know why `handlerReply` is a type.
- **Refinement and search are tests, not compile errors.** A Model with a broken refinement builds
  and ships until `cargo test` runs. Whether a product Property is readable on a protocol Scenario
  is decided at admission (`Query::run`), not by the type system; encoding it in types was possible
  but would have doubled the generics on `Property` and `Query`.
- **Proc-macro maintenance.** Each block has a `Parse` impl, a `check` function, an expansion, and a
  set of `trybuild` fixtures. A grammar change touches all four, and the fixtures that freeze
  rustc's own wording need refreshing when the toolchain changes its diagnostics. The `todo!()`s in `macros.rs`
  stand for roughly six hundred lines of parser that a full implementation would carry.
- **Compile times and a second language.** A Go monorepo gains a Cargo workspace, a Rust
  toolchain in CI, and contributors who need to read proc-macro code to change the DSL. The model
  layer would be the only Rust in the tree.
