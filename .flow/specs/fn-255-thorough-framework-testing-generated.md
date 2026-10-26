# Thorough framework testing: generated differential checks, fuzzing, mutants, Miri, coverage (model)

## Conversation Evidence

> user (2026-10-25): "are there advanced testing tools in Rust that allow us to test it more throoguly like Loom or determiniscally?"
> assistant: the framework has almost no concurrency (one cancel `AtomicBool`, one signal atomic, one interning `Mutex`, a WASM `thread_local!`), so Loom/Shuttle and deterministic-simulation tools have little to test. Proposed generated differential testing, decoder fuzzing, cargo-mutants, Miri, Kani and coverage.
> user (2026-10-25): "give list of options in tabular form and rate each from -3 to +3"
> assistant: 15 options rated. Generated differential testing +3; cargo-fuzz, bolero, cargo-mutants +2; coverage, Miri, Kani +1; Shuttle 0 now, +2 once checking goes parallel; Loom −1; deterministic simulation −3.
> user (2026-10-25): "1. Generated differential testing of the lifter and backends 2. Fuzzing the decoders that read untrusted bytes 3. Mutation testing of the framework itself, with cargo-mutants. 4. Miri 6. Coverage with cargo-llvm-cov — approved"

## Goal & Context

<!-- scope: business -->

The framework's central claim is that its checkers agree: compiled native, the interpreter and TLC on the lifted TLA+ give the same states and verdicts for every admitted Model. Today that claim rests on about 18 shared fixtures, the Temporal activity and hand-seeded mutants. Beyond the claim itself, the framework reads bytes it does not control (the Go host's requests to the guest, TLC's output and state dumps), has an unsafe subprocess/syscall boundary and a safe-Rust ABI ownership layer with unsafe export attributes, and has no measure of how well its own tests catch changes.

This spec adds five owner-approved techniques:
1. generated differential testing of the front end, lifter and backends;
2. fuzzing of untrusted-input decoders;
3. mutation testing of the framework with cargo-mutants;
4. Miri on ABI ownership/envelope logic and pure helpers beside the excluded syscall boundary;
5. coverage with cargo-llvm-cov to aim the other four.

They must not slow the inner loop (fn-254). Each finding becomes a pinned regression fixture.

## Architecture & Data Models

<!-- scope: technical -->

**Generated differential testing (R1).** Source generation is required to cover the front end and macro. The spike (task .1) chooses its batching and budgets and whether to add a tree-level layer:
- **Tree level:** a proptest strategy generates admitted trees (the IR the front end produces) within the admitted subset: fieldless enums, bounded integers and `Finite` wrappers, `Option` of those, guards, checked arithmetic, `reject!`/`stay!`, invariants, postconditions and optional facts. The interpreter and TLC (via the emitter) must agree on starts, states, edges, outcomes, facts and claim verdicts. It is fast, since no rustc runs per case.
- **Source level:** the generator prints `#[machine]` Rust source. A batch of N Models compiles into one scratch crate per run, then compiled, interpreted and TLC results must agree. This is the only layer that covers the front end and the macro, but rustc cost limits the case count.

For required-backend cases, compare complete start sets and labeled transition relations, including action identity, successor, outcome and facts, as well as states and verdicts. Existing TLC helpers compare state sets/verdicts and replay counterexamples; .1 must demonstrate a bounded complete-transition extraction method before .2 builds on them. Every selected backend must finish successfully. Identical refusals are failures, even though the existing interpreter helper accepts matching refusals.

Shrinking yields a minimal disagreeing Model. Every disagreement is pinned as a fixture under `model/umpire/dev/fixtures/fixtures/generated_<n>/`.

**Fuzzing (R2).** cargo-fuzz (libFuzzer) targets in a `fuzz/` package under `model/umpire`, on a pinned nightly:
- guest request decoding and dispatch, run natively (`protocol/runtime/src/runtime.rs::decode`, `Dispatch::prepare` and `abi.rs::Guest::dispatch_to` with actual arbitrary request bytes, `annotations.rs::AnnotationRuntime::prepare`);
- TLC output handling (`backends/tlc/src/runner.rs::{messages, parse, decode_trace}`; `decode.rs::{decode_dump, decode_state}`).

The oracle: a typed refusal or a valid value, never a panic, hang or unbounded allocation. Corpora are seeded from existing goldens and recorded TLC output.

**Mutation testing (R3).** cargo-mutants per framework crate, with an in-repo config of exclusions (generated code, `Debug`/`Display` impls, test support). A missed mutant is triaged as a test gap (add a test), an equivalent mutant (record it) or dead code (remove it).

**Miri (R4).** Native unit tests of `abi.rs`'s safe-Rust buffer allocation, release and call-envelope logic run under Miri; its unsafe export attributes do not make its bodies unsafe. The actual unsafe execution boundary is `backends/process/src/syscall.rs` and its `syscall/` children. The raw-syscall module (`backends/process/src/syscall/calls.rs`) cannot run under Miri (FFI and volatile function pointers), so it is excluded with a recorded reason. Pure helpers in `backends/process/src/syscall.rs` get tests that Miri can run. The ABI tests exercise shared production ownership/envelope logic through a native-sized internal seam, preserving the exported wasm32 ABI. Successful allocation, dispatch, response publication and release are asserted; a native pointer that cannot fit `u32` cannot turn these tests into refusal-only coverage. Expected invalid-release panics are tested through Rust methods rather than unwinding across `extern "C"`.

**Coverage (R5).** cargo-llvm-cov per framework crate: line and branch coverage, with a list of unreached refusal and lowering branches in the front end, IR, emitter and interpreter. The list feeds R1's generator weights and R2's corpora.

**Tiers (R6).** Everything is wired through the framework entry point from fn-254 R7 (in `model/umpire`):
- `model-quick`: none of this.
- `model-check`: fixed-seed interpreter/IR controls and stable deterministic corpus replay (each input once), plus compiled comparison where a source batch is selected. TLC is explicitly skipped in default mode; this is not three-way agreement.
- `model-validate`: bounded required-TLC differential cases, timed fuzzing and Miri within the 120 s share; all selected backends must complete. Longer campaigns run only in the scheduled tier.
- Scheduled only: cargo-mutants, coverage reports.

Each scheduled run has a time budget and reports completed and remaining work when capped. Spec closure requires bounded smoke/control runs and working scheduled commands, not completion of full mutation or coverage campaigns.

## API Contracts

<!-- scope: technical -->

- `model/umpire/dev/generate` (or a module in `umpire-testing`, decided by .1): `proptest` strategies for admitted trees and source; `differential(tree|source) -> Result<(), Disagreement>`.
- `model/umpire/fuzz/`: cargo-fuzz package; one target per decoder; corpora under `fuzz/corpus/<target>/`.
- `model/umpire/.cargo/mutants.toml` (or the location fn-254.9 decides): exclusions and timeouts.
- Framework entry point subcommands (names follow fn-254.9's choice): `fuzz-replay`, `fuzz <target> --time <s>`, `mutants <crate>`, `miri`, `coverage`.
- All tool pins live in `model/umpire/mise.toml`, and runtime configuration lives within `model/umpire`; the standalone framework requires no parent configuration. New pinned tools via mise: a dated nightly toolchain (fuzz, Miri, branch coverage (bounded smoke and scheduled)), cargo-fuzz, cargo-mutants, cargo-llvm-cov.

## Edge Cases & Constraints

<!-- scope: technical -->

- **Generator soundness.** Generated Models must lie inside the admitted subset; a refusal by the front end counts as a generator bug, not a pass. Unbounded or huge state spaces are excluded by construction (small domains, bounded depth).
- **Differential, not oracle-free.** Required-TLC differential agreement passes only if every selected backend completes and agrees. Default-mode interpreter/native checks explicitly record TLC skipped and earn no TLC-agreement credit. A TLC tool error, timeout or unsupported refusal fails the case with its reason; it never counts as agreement.
- **Determinism.** Fixed-seed cases in `model-check` are reproducible; a failure prints the seed and the shrunk Model source.
- **Nightly.** Pin a dated nightly for fuzzing, Miri and branch coverage (bounded smoke and scheduled), isolated from stable production/gate builds. Default corpus replay uses a stable deterministic harness; it does not require libFuzzer or nightly.
- **Independence.** Generators must not reuse the front end's own lowering to construct expectations (no self-confirming oracle). Agreement between independent backends is the oracle.
- **Cost.** Nothing here enters `model-quick`. `model-check` additions are budgeted (proposal: under 30 s total) and measured against fn-254's baseline.
- **Framework standalone.** All harnesses live in `model/umpire`, use no Temporal crate, and pass fn-254.11's standalone control.

## Acceptance Criteria

<!-- scope: both -->

- **R1:** Generated source testing covers the macro/front end; tree testing is optional. Required-backend cases successfully complete and agree on complete starts, states, labeled edges, outcomes, facts and verdicts within the configured bounds. Errors: swapped-comparison, dropped-guard and outcome/fact-only mutations are caught and shrunk; the last leaves states/verdicts unchanged. A backend error, timeout or refusal fails, including matching refusals. Every real finding is pinned as a fixture.
- **R2:** cargo-fuzz targets exist for every decoder listed in the Architecture section, with seeded corpora. A timed run per target finds no panic, hang or unbounded allocation, or each finding is fixed and pinned as a corpus regression. Errors: a deliberately planted panic in a decoder is found within the target's time budget.
- **R3:** Committed cargo-mutants configuration and commands support each framework crate. A bounded smoke run proves execution and triage; scheduled capped campaigns report per-crate counts, completed/remaining work and triage of observed missed mutants. Full campaigns and exhaustive triage are not closure prerequisites. Errors: report controls distinguish missed, timed-out and unviable mutants; a timeout never counts as caught.
- **R4:** Shared production ABI buffer/envelope logic completes successful native-sized allocation, dispatch, publication and release under Miri without undefined behaviour. Invalid-range behavior is checked through Rust methods; syscall FFI exclusions and pure-helper coverage are recorded. Errors: a bounded test-only out-of-bounds control is reported by Miri; allocation returning zero cannot satisfy positive-path coverage.
- **R5:** Coverage commands support per-framework-crate reports of unreached refusal and lowering branches. A bounded representative smoke report drives a generator/corpus adjustment and records its before/after branch evidence. Scheduled capped campaigns produce broader per-crate reports with completed/remaining work; full campaigns are not closure prerequisites.
- **R6:** Tier wiring per the Architecture section through the framework entry point, documented in `model/umpire/README.md`, with measured `model-check` cost inside its budget. Errors: a corpus regression or fixed-seed disagreement fails `model-check` with its name and seed.
- **R7:** Every fn-255 step is bounded by work and by time, and none can degrade fn-254's goal:
  - **Work bounds** (hard, deterministic): generated Model size caps (states, edges, depth), case counts per tier, shrink-iteration cap, TLC launches per case, fuzz corpus-size cap, and replay running each input once. These come from one committed budget table (fn-254.12).
  - **Time bounds:** a per-case and per-input timeout; libFuzzer `-rss_limit_mb` and `-max_len`; tier shares of at most 30 s in `model-check` and 120 s in `model-validate`, measured warm on the fn-254 baseline host.
  - **Full campaigns** (cargo-mutants, coverage, long fuzz and generator runs) live only in a scheduled tier with a wall-clock cap. They are never part of `model-quick`, `model-check`, `model-validate` or a spec-close requirement. Closure uses only the fixed, separately capped mutation/coverage smoke controls in .4/.6.

  Errors: an over-cap generated Model, a planted over-budget work count and a planted slow fuzz input each fail with their stage name. If the re-measured gates miss fn-254 R6's targets with fn-255 included, fn-255 counts are cut, never the targets.

## Boundaries

<!-- scope: business -->

- Not adopted (owner, 2026-10-25): Kani, bolero, Loom, Shuttle, sanitizers, cargo-careful, deductive verifiers, deterministic simulation. Shuttle stays a candidate once the native checker or the TLC runner goes parallel.
- No Temporal Models or Temporal crates in any harness; the Temporal activity keeps its own fixtures.
- No change to Model semantics or checker verdicts, except fixes for bugs these techniques find, each pinned by a fixture.
- No remote fuzzing service or CI infrastructure.

## Decision Context

<!-- scope: both -->

2026-10-25, owner approved five techniques from the rated table: generated differential testing, decoder fuzzing, cargo-mutants, Miri and cargo-llvm-cov. These are new pinned tools (a nightly toolchain, cargo-fuzz, cargo-mutants, cargo-llvm-cov), approved together with the techniques.

Rejected:
- **Kani:** narrow, a heavy toolchain, and mostly covered by the Models' own checkers.
- **bolero:** only worth it if one harness has to serve both proptest and fuzzing; cargo-fuzz and proptest stay separate.
- **Loom / Shuttle:** no concurrency to explore yet.
- **Deterministic simulation:** the framework has no async and is already deterministic.

2026-10-25, owner: "ensure that the new testing steps are bounded and don't sabotage our fast validation gate goal" (R7). Work counts are enforced hard because wall-clock limits alone are flaky on a shared host; time is budgeted per stage through fn-254.12.

Placement follows MILESTONES: .1/.3/.4 may prepare disjoint source work before fn-254 closes. Bound every early probe/run explicitly; serialize shared pins and source fixes. Final tier wiring in .7 consumes committed fn-254.6/.10/.12 and registers the same measured bounds in its authoritative tables before gate activation.

Maintainability (plan review): duplication - repeated validate-tier wording and budget shares now refer to R7 and the authoritative fn-254.12 tables; structure - none identified.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Generated source testing covers the macro/front end; tree testing is optional. Required-backend cases successfully complete and agree on complete starts, states, labeled edges, outcomes, facts and verdicts within the configured bounds. Errors: swapped-comparison, dropped-guard and outcome/fact-only mutations are caught and shrunk; the last leaves states/verdicts unchanged. A backend error, timeout or refusal fails, including matching refusals. Every real finding is pinned as a fixture. | fn-255-thorough-framework-testing-generated.1, fn-255-thorough-framework-testing-generated.2 | — |
| R2 | cargo-fuzz targets exist for every decoder listed in the Architecture section, with seeded corpora. A timed run per target finds no panic, hang or unbounded allocation, or each finding is fixed and pinned as a corpus regression. Errors: a deliberately planted panic in a decoder is found within the target's time budget. | fn-255-thorough-framework-testing-generated.3 | — |
| R3 | Committed cargo-mutants configuration and commands support each framework crate. A bounded smoke run proves execution and triage; scheduled capped campaigns report per-crate counts, completed/remaining work and triage of observed missed mutants. Full campaigns and exhaustive triage are not closure prerequisites. Errors: report controls distinguish missed, timed-out and unviable mutants; a timeout never counts as caught. | fn-255-thorough-framework-testing-generated.4 | — |
| R4 | Shared production ABI buffer/envelope logic completes successful native-sized allocation, dispatch, publication and release under Miri without undefined behaviour. Invalid-range behavior is checked through Rust methods; syscall FFI exclusions and pure-helper coverage are recorded. Errors: a bounded test-only out-of-bounds control is reported by Miri; allocation returning zero cannot satisfy positive-path coverage. | fn-255-thorough-framework-testing-generated.5 | — |
| R5 | Coverage commands support per-framework-crate reports of unreached refusal and lowering branches. A bounded representative smoke report drives a generator/corpus adjustment and records its before/after branch evidence. Scheduled capped campaigns produce broader per-crate reports with completed/remaining work; full campaigns are not closure prerequisites. | fn-255-thorough-framework-testing-generated.6 | — |
| R6 | Tier wiring per the Architecture section through the framework entry point, documented in `model/umpire/README.md`, with measured `model-check` cost inside its budget. Errors: a corpus regression or fixed-seed disagreement fails `model-check` with its name and seed. | fn-255-thorough-framework-testing-generated.7 | — |
| R7 | Every fn-255 step is bounded by work and by time, and none can degrade fn-254's goal: | fn-255-thorough-framework-testing-generated.1, fn-255-thorough-framework-testing-generated.2, fn-255-thorough-framework-testing-generated.3, fn-255-thorough-framework-testing-generated.4, fn-255-thorough-framework-testing-generated.5, fn-255-thorough-framework-testing-generated.6, fn-255-thorough-framework-testing-generated.7 | — |

## Parked unknowns

- Whether tree-level generation can produce the admitted tree without going through the front end, or must print source and admit it (affects the independence constraint).
- Case counts and time budgets per tier; .1 and fn-254's baseline set them.
- Whether TLC's per-run cost (fn-254.2) makes TLC the bottleneck for generated cases; batch generated Models into one TLC module if so.
