# Disposable countdown probes

Source revision: 02d4f55660e4da54612d703f16bd92170c742539. Includes fn-246.15 retry correction; excludes fn-246.13 cancellation correction. Never use this checkout as a production migration baseline.

Only run in .worktrees/fn2532-spike at that exact revision. The original disposable branch is work/fn2532-spike; a fresh detached checkout at the pinned revision and the same path also works. Copy the archived probes, including Cargo.lock, into that checkout's .flow/tmp/fn253-spike directory, then run:

    env -u BASH_ENV python3 .flow/tmp/fn253-spike/install-stubs.py
    env -u BASH_ENV bash .flow/tmp/fn253-spike/run.sh

The installer changes the disposable DSL crate and declares a candidate countdown_spike module in the actual umpire-temporal crate at model/temporal/realization/src/lib.rs. The module body is also compiled by the independent native project-library control, avoiding that crate's unrelated API build prerequisites. This independent control proves the helper body compiles, not that the whole Temporal crate was checked. Reinstallation only permits the exact declarations this installer owns. The preparer reads the immutable baseline through git show; every baseline transformation asserts its exact match count and records those counts. No production Model is overwritten.

admission.rs emits JSON lines containing input path, outcome, one-based line/column, exact source line and message. It distinguishes parse errors, Model refusals, hook refusals and admission using the current public umpire_front::read API. Current Diagnostic has line, column, message fields; no span member. Successful reporting alone proves nothing. check-results.py asserts every outcome against manifest.json, including actual baseline and no-argument receiver controls.

Full baseline/local/library/shared native typechecks run separately from admission; the macro can embed a refused interpretation while still generating compilable native code. The library variant retains Model-owned Finite wrappers and adds trait adapters. The shared variant removes both local wrappers and uses imported Ticks<3>/Remaining<3> from the disposable DSL crate. Their first refusals are isolated from expression/law call refusals by independent minimal probes.

countdown-proposed.rs is the proposed shared-type .3 surface: guarded eager unwrap_or bodies and unit-returning within/descends laws using check!, following closable. shared-types.rs provides actual generic Finite domains, with N chosen from 1..=255 by the Model; its N=0 native boundary controls document an empty configured domain and singleton zero remaining domain, while within refuses that invalid bound. countdown.rs and trait_countdown.rs are comparative mechanics sketches; their bool predicates are named within_holds/descends_holds. A bool helper is not an admitted delegated law. Proposed signatures: arm(Option<Ticks<N>>) -> Option<Remaining<N>>, tick(Option<Remaining<N>>) -> Option<Remaining<N>>, armed(Option<Remaining<N>>) -> bool, expired(Option<Remaining<N>>) -> bool, within(Option<Remaining<N>>, Option<Ticks<N>>) -> (), descends(Option<Remaining<N>>, Option<Remaining<N>>) -> (). Trait adapter signatures remain in trait_countdown.rs.

Initial-only associated arm calls are a positive control: source reads retain computed starts as constructor data, so this is not evidence that handler expressions can call associated helpers. local-arm-initial and unused-generic are explicit positive controls; local-arm exercises a handler call, and local-generic derives Finite to exercise the bounded-newtype admission path.

No command runs TLC, modifies Flow, commits, pushes, or downloads dependencies. Rust is pinned to rust@1.99.1; Cargo uses --offline --locked with the archived standalone Cargo.lock, and caller CARGO_TARGET_DIR is forwarded. Every required dependency must already exist in the local Cargo cache. Native mechanics and admitted source alone do not prove full native/TLC equivalence.

prepare.py records exact total and selected method line counts in line-counts.json, separately from its asserted replacement counts in rewrite-counts.json. The full baseline and all rewrite inputs are regenerated from the pinned revision; neither generated source nor diagnostic positions are reformatted.

The worktree and its DSL modifications are disposable; only these source assets are copied to the canonical .flow/tmp directory. The verdict must archive these assets or embed reconstruction instructions before scratch expiry.
