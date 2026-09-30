# Review: cmp/rust/

## Scores (1-5, 5 best)

1. **Spec fidelity: 4.** Both Models are complete, section order follows the Lean file, the revised Model 2 is applied (`pauseRequested -> Started` at `standalone_activity.rs:540`, visible retry rows at `standalone_activity.rs:207-210`), and every spec pin appears in `pins.rs`; the deduction is that enum variants and state fields use Rust casing (`SyncSuccess`, `Unset`, `schedule_to_close`) so scenario lines do not match the spec's spelling side by side, and `workerStop` is re-exported from the worker module (`nexus_caller.rs:123`) rather than declared per Model.
2. **Language plausibility: 4.** The Model files are idiomatic Rust (payload enums with exhaustive `match`, struct update syntax, `const` state literals, `LazyLock` statics, lowercase unit-struct action types with Diesel as precedent); the sketch has a few things that could not compile as written: the derive references a nonexistent `PhaseField` trait (`umpire/macros.rs:113`), `.span()` is called on `Path`/`Expr` without importing `syn::spanned::Spanned` (`umpire/macros.rs:431,563,565`), and glob-importing variants (`nexus_caller.rs:75-77`) is a pattern many Rust reviewers reject because a misspelled unit variant silently becomes a binding.
3. **Authoring readability: 4.** The declarative blocks (`machine!`, `scenario!`, one-line `query!`, `set!`) read as configuration and sit close to the Lean; the step functions carry `vec![Step { .. }]`, `ProtocolFact::` qualification and several lines past 100 columns (`nexus_caller.rs:350,394,453`), and `machine!` needs `outcome:`/`facts:` lines Lean does not have (acknowledged in README).
4. **Check story accuracy: 4.** The table at `README.md:49-60` is honest that table, refinement, search and admission are `cargo test`; two overclaims: the arity-mismatch illustration is labelled E0308 (`README.md:111`) where rustc emits E0593 for a wrong-arity fn against `impl Fn`, and the "cargo check" tier for `const _: () = assert!` pins lives in `tests/pins.rs` (`pins.rs:34-47`), which plain `cargo check` does not compile without `--tests`.
5. **Framework realism: 4.** Table enumeration (`umpire/lib.rs:300-317`), the mapped-state refinement check (`umpire/lib.rs:395-421`), admission (`umpire/lib.rs:480-499`) and the `action!`/`machine!`/`set!` expansions are written out; bounded search (`umpire/lib.rs:604`), reachability (`umpire/lib.rs:331`), `restrict`, `Compose::machine`, the struct/tuple `all()` and four of the block parsers are `todo!()`, and mapping a product claim's facts onto a protocol step is not addressed (`umpire/lib.rs:380,505`).
6. **README honesty and library table: 5.** Costs are plain (second toolchain in a Go monorepo, roughly six hundred lines of parser behind the `todo!()`s, refinement as a test not a compile error, ownership noise); spot checks match: `stateright/stateright` pushed_at 2025-07-27 (README says 2025-07-27, no-go), `proptest-rs/proptest` 2026-09-27 (matches), `dtolnay/trybuild` 2026-09-08 (matches), `stephaneyfx/enum-iterator` 2025-09-08 (matches). `petgraph/petgraph` could not be reached (connection reset).

## Verbatim snippets

`handlerReply` step, product machine (`nexus_caller.rs:184-198`):
```rust
pub fn handler_reply_step(state: &ProductState, reply: Reply) -> Vec<ProductStep> {
    if state.phase != ProductPhase::Scheduled {
        return vec![];
    }
    match reply {
        SyncSuccess => product_step(ProductPhase::Succeeded, ProductFact::NexusOperationCompleted),
        Async => product_step(ProductPhase::Started, ProductFact::NexusOperationStarted),
        OperationFailed => product_step(ProductPhase::Failed, ProductFact::NexusOperationFailed),
        OperationCanceled => product_step(ProductPhase::Canceled, ProductFact::NexusOperationCanceled),
        HandlerError { retryable: true } => vec![],
        HandlerError { retryable: false } => product_step(ProductPhase::Failed, ProductFact::NexusOperationFailed),
    }
}
```

`syncSucceeds` (`nexus_caller.rs:546-550`):
```rust
property! { syncSucceeds
    machine: nexusProtocol
    when: handlerReply(SyncSuccess)
    holds: |step| step.state.phase == Phase::Succeeded && step.facts.contains(&ProtocolFact::NexusOperationCompleted)
}
```

`syncReplied` and `syncCompletion` (`nexus_caller.rs:626-630`, `nexus_caller.rs:699`):
```rust
scenario! { syncReplied
    model: nexusProtocol
    starts: Unscheduled
    actions: [schedule(Unset, Unset, Unset), handlerReply(SyncSuccess)]
}

query! { syncCompletion find: syncSucceeds in: syncReplied limits: two }
```

`nexusCaller` compose (`nexus_caller.rs:779-789`):
```rust
compose! { nexusCaller
    for: [operation, worker::worker]
    state: NexusCallerState
    members: { operation: nexusProtocol, worker: handlerWorker }
    sync: {
        workerStop: operation.workerStop || worker.workerStop,
        handlerReply: operation.handlerReply || worker.serve,
    }
    starts: { operation: Unscheduled, worker: Polling }
    ends: { operation: [Succeeded, Failed, Canceled, TimedOut] }
}
```

## Line counts

| File | Lines |
|---|---|
| README.md | 226 |
| nexus_caller.rs | 814 |
| standalone_activity.rs | 864 |
| pins.rs | 298 |
| worker.rs | 108 |
| umpire/lib.rs | 805 |
| umpire/macros.rs | 827 |
| Total | 3942 |
| Two Model files (nexus + standalone) | 1678 |

## Red flags

- **Names not exactly as spec.** Variants are PascalCase (`SyncSuccess`, `Expires`, `Failed { retryable }`) and state fields snake_case (`schedule_to_close`). The Lean spelling is restored only at runtime by `Finite::key()` (`umpire/lib.rs:59-63`). A side-by-side reader sees `schedule(Unset, Unset, Unset)` against the spec's `schedule (unset, unset, unset)`.
- **Wrong rustc error code in the README.** The "step function bound to the wrong action" example (`README.md:108-119`) is shown as E0308. A fn item of the wrong arity passed to `impl Fn(&S, Reply)` is E0593; a wrong argument type is E0631. The location claim (at the `steps:` line) is sound, the code and message text are invented.
- **"cargo check" tier is really the test target.** The `const _: () = assert!(..)` pins (`pins.rs:34-47`) are in `tests/pins.rs`. They run under `cargo check --tests` or `cargo test`, not under a plain `cargo check` of the library, which `README.md:55` and `pins.rs:9` imply.
- **Sketch code that cannot compile in principle.** `umpire/macros.rs:113` uses `<Self as PhaseField>::Phase` where no `PhaseField` trait exists (the derive already has the field types at line 108). `umpire/macros.rs:431,563,565` call `.span()` on `Path`/`Expr` with no `Spanned` import. Tolerated by the spec, but a reader judging plausibility should know.
- **Refinement comment vs implementation.** `umpire/lib.rs:376` promises "the first in catalog order when several fit", but `check` iterates the product table in declaration order (`umpire/lib.rs:397-400`). Pass/fail is unaffected (matching is by mapped states); the reported product key for rows like nexus `scheduled -> failed` (three product classes fit) would differ from the Lean's.
- **Product claims over facts cannot be read through the map.** `Refines::map_erased` maps state only (`umpire/lib.rs:380,422-424`) and `holds_step` is `todo!()` (`umpire/lib.rs:505`). Both product properties here are state-only transition claims, so the gap is latent, not triggered.
- **`workerStop` is re-exported, not declared.** `nexus_caller.rs:123` and `standalone_activity.rs:113` `pub use worker::workerStop` where the Lean declares the action in the Nexus Model. Semantically the same class; structurally a deviation from "keep the Lean file's section order".
- **trybuild pins rustc text.** The `.stderr` fixtures (`pins.rs:257-270`) freeze rustc's own wording, which changes with toolchain releases. The README does not list this as a maintenance cost.
- **Composite scenario `starts:` is unexplained.** `starts: { operation: Unscheduled }` (`nexus_caller.rs:803`) must synthesize a `NexusCallerState` with the worker at its first value; `scenario!` is `todo!()` (`umpire/macros.rs:609-612`) and the doc comment describes only action lines.

## Strengths

- **The name-resolution design is real and consistently applied.** Every DSL name becomes a Rust path emitted with `quote_spanned!` at the author's token (`umpire/macros.rs:481-495`), so undeclared actions, phases, facts, entities and queries are rustc errors with the caret under the author's token, and rust-analyzer shows them. This is the strongest compile-time name-check story a non-Lean language can offer without a whole-program macro.
- **The refinement check is implemented to the spec's semantics.** `umpire/lib.rs:395-421` walks every protocol row through the map, treats equal mapped states as a stutter, and otherwise looks for any product transition between the mapped states, naming the rejected row. The README shows the failure this produced under the first spec draft (`README.md:135-140`), which is exactly the kind of evidence a decision maker wants.
- **Genuine compile-time state-space pins.** `Finite::CARDINALITY` as an associated `const` (`umpire/lib.rs:50`) lets `const _: () = assert!(ProtocolState::CARDINALITY == 8 * 3 * 2 * 2 * 2)` (`pins.rs:35`) fail the build, not a test.
- **pins.rs covers every spec pin and more.** All counts, the two `[]` pins, reachability, refinement pass plus five row lookups that pin the revised Model 2 mapping (`pins.rs:215-225`), every find-query, both verifies, the composition verifies, an admission-error pin (`pins.rs:276-298`), and a trybuild tier for error text.
- **Comments preserved.** The Lean's semantic comments (product vs protocol, stutter rows, faults as ordinary actions, why `workerStop` is `[]` on the product) are carried over nearly verbatim in both Model files.

## One-paragraph verdict

The Rust sample shows that a proc-macro DSL can make the declarative half of a Model read like configuration while keeping step functions as plain, exhaustively matched Rust, and it turns rustc and rust-analyzer into the DSL's name checker with token-pinned errors for free. State-space sizes are pinned at compile time; everything that needs a table (refinement, search, admission) is an ordinary `cargo test`, and the README says so plainly and backs it with a real failing-refinement output and a dated, spot-check-consistent library table. The single biggest reservation is the machinery under the sugar: the proc-macro crate is the largest file in the sample and the least written (four parsers, `scenario!`, `compose!`, `observation!` and `limits!` are `todo!()`), the README itself estimates six hundred more lines of parser, it brings a second toolchain and `syn`-literate maintainers into a Go monorepo, and its strongest checks still ship as tests rather than as build failures.
