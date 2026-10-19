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
TBD

## Evidence
- Commits:
- Tests:
- PRs: