---
satisfies: [R1]
---
# fn-214-progress-vocabulary-on-kettle-scopes.1 Assumptions: only and fair with explicit fairness and legacy compatibility

## Description
Implement R1 on the current flat attribute surface, preserving the legacy trait API.
This task owns the shared assumption representation and fairness integration before .2.

**Size:** M
**Files:** existing DSL assumption/machine APIs, macro helpers, progress admission, oracle and focused fixtures.
**Touches:** [model/umpire/dsl/src/scope.rs, model/umpire/dsl/src/lib.rs, model/umpire/dsl/src/machine.rs, model/umpire/dsl/macros/src/machine.rs, model/umpire/dsl/macros/src/action.rs, model/umpire/dsl/macros/tests/fixtures/**, model/umpire/dsl/macros/tests/ui/machine_scope_actor.stderr, model/umpire/dsl/tests/**, model/umpire/checker/src/explore.rs, model/umpire/checker/src/progress.rs, model/umpire/checker/src/refusal.rs, model/umpire/checker/tests/**, model/umpire/tests/fixtures/progress*/**, model/umpire/wasm/src/refusal.rs, model/umpire/wasm/tests/dispatch.rs, model/umpire/proto/README.md]

### Approach

- Extend the existing `Assumption` in `dsl/src/scope.rs`; retain the `Scope` alias and `Scope::without(&[..])` constructor. Do not rename the file or duplicate the assumption representation.
- Add optional admitted-action selection and explicit fair membership. `.only` replaces the prior only-selection; `.without` and `.fair` accumulate stable deduplicated entries. Restriction order does not affect only-minus-without admission.
- Preserve the static `Assumption::without` API and add receiver chaining through one extension trait, imported into generated modules. Rust permits that combination; the disposable `/tmp/fn214-associated-and-chain.rs` probe compiled and ran with Rust 1.99.1. Use actor-to-generated-Action conversion in the existing helpers and chaining methods. Keep generated syntax ordinary Rust, without new attributes or method/header rewriting.
- Replace `world.internal` fairness selection with the assumption's fair membership. Validate all three lists, including nonadmitted fair entries, against the catalog before search where registration metadata is already available. Preserve once-per-state generated evaluation and Unknown on inconsistent assumptions.
- Refuse empty admission only for an explicit restriction (`only` specified or nonempty exclusions). An unrestricted zero-action Model remains valid and can yield a deadlock. A valid press-only assumption with a disabled press is a deadlock, not an admission error.
- Update the independent brute-force oracle's fairness input and generated assumptions; it currently encodes implicit internal fairness in multiple places. Existing fairness-specific fixtures must explicitly declare their fair actions when their purpose needs them.
- Correct `Machine::internal` and derive documentation to mean environment-independent only. No new dependencies or workspace wiring.
- Carry the new EmptyAssumption refusal through the existing WASM bridge. Code 7 (`ScopeAction`) categorizes action-admission errors; preserve distinct unknown-action and empty-restriction diagnostics. Pin the empty-admission response through generated protobuf encode/decode and document this category in the protocol README. No schema, method ID or ABI change is needed.

### Investigation targets

**Required:**
- `model/umpire/dsl/src/scope.rs:1` - current representation and static constructor.
- `model/umpire/dsl/macros/src/machine.rs:612` - assumption helpers and actor conversion.
- `model/umpire/checker/src/progress.rs:14` - validation, fixed assumption evaluation and fair SCC checks.
- `model/umpire/checker/tests/oracle.rs` - independent reference fairness.
- `model/umpire/checker/tests/support/progress_evaluation.rs` - once-per-state and inconsistent-assumption controls.

**Optional:**
- `model/umpire/tests/fixtures/sample_kettle_trait/model.rs:123` - protected legacy constructor and scope builder.

### Quick commands

From `model/`, `mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros -p umpire-checker -p umpire-tests`.
From the repository root, run `make model-check` and record one `make model-time` output.
Timing uses the owner's 10 s inner / 30 s full soft targets; investigate material overruns, not millisecond noise.
## Acceptance
- [ ] Fixtures pin all/default, only, exclusion, chain-order independence, repeated-only replacement and stable repeated fair/without behavior; actor conversion works for constructors and chains.
- [ ] Fixtures prove internal actions are not fair by default, explicit fairness affects cycles, and fairness never admits an excluded action. The independent brute-force oracle exercises explicit fair sets.
- [ ] Unknown actions in every available registration/override list refuse naming the action even with zero starts. Method-produced lists are validated on real reached evaluations only; zero starts invoke no evaluator and preserve vacuous verification. Explicit empty admission refuses, while an unrestricted empty catalog and state-local disabled actions retain deadlock semantics.
- [ ] The canonical `sample_kettle` and `sample_kettle_trait` sources and all their existing snapshot metadata/payloads match the task baseline byte-for-byte. Legacy `Scope::without` and `Progress::within(scope)` still compile unchanged.
- [ ] Focused tests, `make model-check` and the recorded `make model-time` pass; macro refusal/expansion updates are scoped to the added API and semantics.
## Done summary
Model authors can compose `only`, `without` and explicit `fair` assumptions over actor actions while retaining `Scope::without(&[])` and registered `Progress::within(scope)`. The checker validates available action lists before search, preserves reached-state evaluation and zero-start vacuity, and treats internal actions as environment-independent without implicit fairness.

R1 coverage lives in the DSL `assumptions` fixture, checker `progress_explicit_fairness` and `progress_assumption_validation` fixtures, the fixed-evaluation controls and the independent brute-force oracle. They pin replacement, stable deduplication, admission order, unknown actions in every list including nonadmitted fair entries, explicit empty restrictions, disabled-action and empty-catalog deadlocks, and generated callback counts. The WASM dispatch test pins empty admission as code 7 with its distinct diagnostic through generated protobuf encoding and decoding.

Both canonical Kettle directories match base a37fe4a667ea140800689e82a2edfb52ee481298 byte-for-byte. Verification compared the complete filesystem path inventory and every file's Git blob hash, covering sources, snapshot metadata and payloads. Macro snapshot changes cover helper expansion, internal documentation and the compiler's `model::without` qualification.

baseline: green after a clean focused suite, `make model-check` and `make model-time`. The first model-check attempt exited 2 while an oracle child edit raced the baseline format check; restoring that exact edit established the clean baseline. The contaminated observation remains in `.flow/tmp/fn214-1-baseline-model-check.log`; the clean gate log is `.flow/tmp/fn214-1-baseline-model-check-clean.log`.

RED evidence is recorded in `.flow/tmp/fn214-1-red-api.log`, `fn214-1-red-semantics.log`, `fn214-1-red-implicit-fairness.log` and `fn214-1-red-wasm-refusal.log`. The semantic negative control observed implicit fairness incorrectly verify `internal_is_not_fair`. The full-gate WASM compile failure remains in `.flow/tmp/fn214-1-model-check-pre-wasm.log`.

The focused suite, WASM dispatch suite and full model gate passed. Post-change timing passed at 2.679 s inner and 3.661 s full against the 10 s / 30 s soft targets. Logs are `.flow/tmp/fn214-1-focused.log`, `fn214-1-wasm-dispatch.log`, `fn214-1-model-check.log` and `fn214-1-model-time.log`. Final native/WASI clippy, formatting and denied-warning documentation checks passed in `fn214-1-final-hygiene.log`.

GATE_SKIPPED:unittest:green-receipt 79b05c24 - baseline reused from prior post-gate pass
GATE_SKIPPED:model-check:green-receipt 79b05c24 - baseline reused from prior post-gate pass
GATE_SKIPPED:model-time:green-receipt 79b05c24 - baseline reused from prior post-gate pass

stage: impl-review - ran [2026-10-19T16:04:19Z..2026-10-19T16:06:19Z] (codex:gpt-6.1-sol:high, SHIP on all three axes, zero findings)

Review receipt: `/tmp/impl-review-receipt-8f37faba39e2-fn-214-progress-vocabulary-on-kettle-scopes.1.json`. The reviewer and writer use the same model family under explicit project routing. Reviewers' read-only Cargo lock failures limited their checks to static inspection; the worker's actual gates passed. A transient abandoned fanout journal was superseded by the original foreground command's successful completion and finalization, without a retry or counter reset. Clean first-pass review required no memory capture.

Tier: session (jev-unavailable(no_key)); explicitprojectroutingpreserved

Two bounded subagents updated the oracle and authored the admission fixture. Both returned before staging and verification. The authoring route now accepts `only([...]).without([...]).fair([...])` inside generated machine modules. No follow-up work is required for R1.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 79b05c248963a85e098483d9ff20141b5e35994f
- Tests: mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros -p umpire-checker -p umpire-tests, mise exec -- cargo test -p umpire-wasm --test dispatch, make model-check, make model-time, mise exec -- cargo fmt --all --check, mise exec -- cargo clippy --workspace --all-targets --all-features -- -D warnings, RUSTDOCFLAGS="-D warnings" mise exec -- cargo doc --workspace --no-deps --quiet, mise exec -- cargo clippy --target wasm32-wasip1 -p umpire-dsl -p umpire-checker --lib --locked -- -D warnings, GATE_SKIPPED:unittest:green-receipt 79b05c24 - baseline reused from prior post-gate pass, GATE_SKIPPED:model-check:green-receipt 79b05c24 - baseline reused from prior post-gate pass, GATE_SKIPPED:model-time:green-receipt 79b05c24 - baseline reused from prior post-gate pass
- PRs: