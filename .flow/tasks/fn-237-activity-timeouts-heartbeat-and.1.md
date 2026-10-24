---
satisfies: [R1, R2]
---
# fn-237-activity-timeouts-heartbeat-and.1 Optional handler facts through native adapters and independent TLC replay

## Description
Moved from fn-227.9 on 2026-10-21.

Deliver the bounded fact-result interface in the tracked deadline-fact contract, independently of activity policy. This is facts checkpoint F, allocated as fn-237-activity-timeouts-heartbeat-and.1 (formerly fn-227-parameterized-models-declared-worlds.9). fn-227 approved its three amendments in `bbf0835fbf`; earlier contract notes calling those amendments pending are superseded. Source anchors predate the completed native work and must be reanchored at execution.

**Size:** M. One return/branch-result adaptation across five existing production owners; existing enum/sequence decoding and Debug replay already carry facts. Compile, source, shared-fixture and replay controls close that same contract. No collector, TLS recorder, fact catalog or general result framework.
**Dependencies:** After fn-227.7 (cross-spec prerequisite, satisfied through this spec's dependency on fn-227). Its typed selected-World/source binding is exercised by F's declared-World fixture. No dependency on fn-227.3 or O (fn-237.2); serial scheduling handles overlapping source ownership.
**Files:** `model/umpire/dsl/macros/src/machine.rs`; `model/umpire/lift/src/{shape,tree,emit}.rs`; `model/umpire/lift/src/body/effects.rs`; new macro controls under `model/umpire/dsl/macros/tests/{pass,ui,fixtures}/`; `model/umpire/lift/tests/source.rs` and new owned source controls; new shared fixture `model/umpire/tests/fixtures/optional_facts/{model.rs,tests.rs,expected/**}`; replay tests in `model/umpire/testing/src/backend.rs`; `model/README.md`.
**Touches:** [model/umpire/dsl/macros/**, model/umpire/lift/**, model/umpire/tests/fixtures/optional_facts/**, model/umpire/testing/src/backend.rs, model/README.md, model/temporal/hosts/go/umpire_temporal_guest.wasm]

### Approach

1. Pin existing unit-handler expansions and no-fact source artifacts. Admit exactly Option<F> for one local nongeneric nonempty fieldless Debug enum. Select the machine Fact and opaque return constraint from that carrier; reject inconsistent/unsupported carriers using existing located diagnostics.
2. Capture the authored optional result inside the existing unit outcome/stay evaluation closure and construct zero-or-one facts only when the step is emitted. Keep state-change/reject/stay enabledness and reject-after-mutation rules. Unit effect helpers remain unit; no facts enter the outcome TLS frame. Direct reject! in an optional handler retains its Rust return-type failure.
3. Independently admit only None, Some(F::Variant), or exhaustive tail conditional trees over those leaves after supported effects. Lower branch facts in the same SSA environment as state updates. Reuse existing conditional/sequence expressions and Seq<Enum>; preserve exact old Seq<Unit> metadata and no-fact emission. Ordinary edges assign an empty sequence every time.
4. Add generic A/B/C/D fixture traces and false claims that force replay across all four values. Preserve authored first-matching guards. Test source-only fact substitution with native rows held constant. Reuse current typed decode and native Debug bridge; native rows supply no source branch meaning.

### Investigation targets

Required, reanchor before coding:

- `.flow/artifacts/fn-227-parameterized-models-declared-worlds/deadline-fact-contract.md` for the complete bounded grammar and current production/test owners.
- `model/umpire/dsl/macros/src/machine.rs` for handler return admission and all reject/stay adapter combinations.
- `model/umpire/lift/src/body/effects.rs` for source conditional SSA and helper boundaries.
- `model/umpire/lift/src/emit.rs` for real-edge fact assignment and observer-only preservation.
- `model/umpire/testing/src/backend.rs` for exact fact replay and observer controls.

### Verification

Quick from `model/`: `mise exec -- cargo test -p umpire-dsl-macros -p umpire-lift -p umpire-testing -p umpire-tests`; repeat admitted shared-fixture tests with `UMPIRE_BACKENDS=require`. Run scoped formatting and strict clippy. First capture the pre-change optional-result/source refusals, then positive and mutation-negative results.

Full from repository root: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`. If linked guest bytes change, run `make umpire-guest`, then the embedded slice/gap checks. Compare all pre-existing snapshots, including metadata, to the pinned original manifest; no activity or other old golden updates. Retain actual counts, replay traces and soft-budget misses.

### Ordering note (2026-10-24)

Starts after fn-244.1, the serialized layout freeze, so it works in the new crate paths (for example `author/dsl/macros` and `backends/tla`).
## Acceptance
- [ ] Public/private and unit/declared-World machines mix unit and optional handlers correctly. Unsupported returns, payload/discriminant/alias/nested carriers and mixed carriers fail the appropriate compile/source controls.
- [ ] Native changed-None, unchanged-Some, ordinary empty, stay, rejection and invalid-mutation controls preserve enabledness and exact fact cardinality.
- [ ] Independent source facts follow the authored SSA branch; source-only swaps alter emitted facts; unsupported returns/helpers refuse at their source; no previous fact leaks into an ordinary edge.
- [ ] Required TLC executes all four value traces and false-claim replay. Wrong valid enum, omitted/duplicate/extra ordinary-edge facts fail replay; unknown values fail decoding.
- [ ] Old unit expansions and all pre-existing artifacts/metadata remain byte-identical. New fixture goldens, focused/full gates and actual timing evidence are retained.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
