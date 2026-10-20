## Requirements Extracted

1. Provide choice-zero `Transition::from`, `Projection::step`/`steps`, `Candidate` with optional expected outcome, and `AnswerPlan::new`.
2. Remove `Realize::matches`; compare final replay outcomes in the Runtime. Mismatches remain Unexplained; absent expectations accept any outcome.
3. Provide `Observed` with method, typed response decoding, and gRPC error-code access.
4. Provide gRPC rejection mapping and service-based RPC method naming.
5. Provide `plain_text` enforcing exactly one `encoding = binary/plain` metadata entry.
6. Provide application-failure construction, reply-envelope packing, and typed callback unpacking.
7. Preserve refusal codes, precedence, and test-pinned messages; provide controls for helper refusal cases.
8. Convert `ActivityExecutionInfo` to `Phase` beside the Model, preserving six valid combinations and invalid-enum/state refusals.
9. Rewrite the activity binding using the constructors/helpers, imported actor enums, cloned payloads, and documented module inclusion; retain its public items.
10. Keep the activity binding body at most eighty lines excluding imports/docs.
11. Preserve Model, activity report/rows/states/lift goldens, and protocol wire bytes.
12. Preserve Go parity, dedicated live scenarios, and fn-218 cover tests.
13. Keep Temporal helpers outside generic `umpire-wasm`, without activity-specific names or unnecessary generalization.
14. Preserve R30 budgets; introduce no crates for this refactor, crate renames, Propose, world, or retryable-input behavior.

## Coverage Verification

1. **COVERED** — constructors and candidate representation: `model/umpire/wasm/src/runtime.rs:15`, `:22`, `:37`, `:59`; constructor controls: `model/umpire/wasm/tests/constructors.rs:6`.
2. **COVERED** — revised trait: `model/umpire/wasm/src/runtime.rs:67`; final-outcome comparison and Unexplained fallback: `:218`, `:243`. Mismatch control: `model/umpire/wasm/tests/support/mod.rs:115`; `model/umpire/wasm/tests/stateful.rs:181`. Stock and test implementations are adapted.
3. **COVERED** — `model/temporal/realization/src/support.rs:17`, `:25`, `:34`, `:42`.
4. **COVERED** — `model/temporal/realization/src/support.rs:52`, `:64`, `:78`; controls: `model/temporal/realization/tests/support.rs:53`, `:140`.
5. **COVERED** — `model/temporal/realization/src/support.rs:83`; metadata precedence controls: `model/temporal/realization/tests/support.rs:86`.
6. **COVERED** — `model/temporal/realization/src/support.rs:98`, `:118`, `:123`; controls: `model/temporal/realization/tests/support.rs:116`.
7. **COVERED** — helper refusal controls: `model/temporal/realization/tests/support.rs:28`, `:86`, `:116`; binding-specific precedence: `model/temporal/features/activity/temporal.rs:45`, `:53`, `:63`. Unknown gRPC codes produce no candidate at `:48`.
8. **COVERED** — realization-only sibling conversion: `model/temporal/features/activity/phase.rs:8`; exhaustive combination controls: `model/temporal/realization/tests/support.rs:154`.
9. **COVERED** — inclusion reason, imports, observation projection, payload normalization, and replies: `model/temporal/features/activity/temporal.rs:3`, `:10`, `:45`, `:87`; retained exports: `model/temporal/realization/src/lib.rs:9`. External-payload regression control: `model/temporal/realization/tests/realization.rs:358`.
10. **COVERED** — measured **80 body lines**; whole file decreases from 221 to 110 lines. Recorded measurement: `.flow/tasks/fn-229-realization-ergonomics-shared-temporal.2.md:53`.
11. **COVERED** — reviewed-range diff contains no Model, expected-output, generated-output, or protocol-tree changes. Report/rows and wire controls remain at `model/temporal/realization/tests/realization.rs:303`, `:272`.
12. **COVERED** — recorded integrated model gates, parity, cover tests, and six live scenarios: `.flow/tasks/fn-229-realization-ergonomics-shared-temporal.1.md:41`; `.flow/tasks/fn-229-realization-ergonomics-shared-temporal.2.md:64`.
13. **COVERED** — Temporal support is exported from `model/temporal/realization/src/lib.rs:4`; helpers contain no activity-specific names. Runtime changes remain generic.
14. **COVERED** — refactor adds no dependency or crate rename and introduces none of the excluded behavior. Recorded model-time: 15.443s gate / 1.883s inner loop at `.flow/tasks/fn-229-realization-ergonomics-shared-temporal.2.md:53`. The separate annotation crate is concurrent fn-217 work, flagged below.

Fresh focused `cargo test` and Clippy attempts both stopped at the read-only Cargo build lock. Passing runtime evidence above comes from committed task evidence; those passes were not independently reproduced here. `git diff --check` passed.

## Reverse Coverage (untraced changes)

The fn-229 implementation, tests, and pipeline records trace to the requirements above. These additional paths belong to concurrent work:

- `.flow/specs/fn-217-annotation-catalog-over-the-temporal.json` - UNRELATED_CHANGE - fn-217 review metadata.
- `.flow/specs/fn-218-test-space-patterns-cover-and-the-full.json` - UNRELATED_CHANGE - fn-218 completion metadata.
- `.flow/tasks/fn-217-annotation-catalog-over-the-temporal.2.json` - UNRELATED_CHANGE - fn-217 metadata.
- `.flow/tasks/fn-217-annotation-catalog-over-the-temporal.2.md` - UNRELATED_CHANGE - fn-217 annotation/build scope.
- `.flow/tasks/fn-217-annotation-catalog-over-the-temporal.3.json` - UNRELATED_CHANGE - fn-217 metadata.
- `.flow/tasks/fn-217-annotation-catalog-over-the-temporal.3.md` - UNRELATED_CHANGE - fn-217 live catalog integration requirements.
- `.flow/tasks/fn-217-annotation-catalog-over-the-temporal.4.json` - UNRELATED_CHANGE - fn-217 metadata.
- `.flow/tasks/fn-217-annotation-catalog-over-the-temporal.4.md` - UNRELATED_CHANGE - fn-217 gap-probe requirements.
- `.flow/tasks/fn-218-test-space-patterns-cover-and-the-full.1.json` - UNRELATED_CHANGE - fn-218 metadata.
- `.flow/tasks/fn-218-test-space-patterns-cover-and-the-full.2.json` - UNRELATED_CHANGE - fn-218 metadata.
- `.flow/tasks/fn-218-test-space-patterns-cover-and-the-full.3.json` - UNRELATED_CHANGE - fn-218 metadata.
- `.flow/tasks/fn-230-reproducible-guest-builds-a-system.1.json` - UNRELATED_CHANGE - fn-230 metadata.
- `.flow/tasks/fn-230-reproducible-guest-builds-a-system.1.md` - UNRELATED_CHANGE - reproducible guest-build requirements.
- `MILESTONES.md` - UNRELATED_CHANGE - mixed roadmap updates extending beyond fn-229.
- `model/Cargo.toml` - UNRELATED_CHANGE - registers fn-217’s annotation crate.
- `model/Cargo.lock` - UNRELATED_CHANGE - records fn-217’s annotation crate.
- `model/README.md` - UNRELATED_CHANGE - fn-218 pattern-grammar correction.
- `model/umpire/annotations/Cargo.toml` - UNDOCUMENTED_ADDITION - fn-217 crate outside this spec.
- `model/umpire/annotations/src/catalog.rs` - UNDOCUMENTED_ADDITION - typed annotation catalogs, validation, and filling.
- `model/umpire/annotations/src/gap.rs` - UNDOCUMENTED_ADDITION - annotation gap classification/reporting.
- `model/umpire/annotations/src/kind.rs` - UNDOCUMENTED_ADDITION - annotation constraint/tag vocabulary.
- `model/umpire/annotations/src/lib.rs` - UNDOCUMENTED_ADDITION - annotation crate exports.
- `model/umpire/annotations/src/types.rs` - UNDOCUMENTED_ADDITION - annotation kind/type witnesses.
- `model/umpire/annotations/src/value.rs` - UNDOCUMENTED_ADDITION - typed validation and generation.
- `model/umpire/annotations/tests/catalog.rs` - UNDOCUMENTED_ADDITION - annotation catalog controls.
- `model/umpire/annotations/tests/generation.rs` - UNDOCUMENTED_ADDITION - annotation generation controls.
- `model/umpire/annotations/tests/snapshots/catalog__admission.snap` - UNDOCUMENTED_ADDITION - annotation admission golden.
- `model/umpire/annotations/tests/snapshots/catalog__gaps.snap` - UNDOCUMENTED_ADDITION - annotation gap golden.
- `model/umpire/annotations/tests/snapshots/catalog__kind-violations.snap` - UNDOCUMENTED_ADDITION - annotation constraint golden.
- `model/umpire/annotations/tests/snapshots/catalog__presence-violations.snap` - UNDOCUMENTED_ADDITION - annotation presence golden.

## Gaps Found

None.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | met | `model/umpire/wasm/src/runtime.rs:15`, `:59`, `:218`; constructor and mismatch controls |
| R2 | met | `model/temporal/realization/src/support.rs:17`; `model/temporal/realization/tests/support.rs:28` |
| R3 | met | `model/temporal/features/activity/phase.rs:8`; `model/temporal/features/activity/temporal.rs:39`; task evidence `.2.md:64` |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict> - All requirements implemented (R-IDs all met or deferred)
