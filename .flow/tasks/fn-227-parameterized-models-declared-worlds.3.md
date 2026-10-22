---
satisfies: [R3, R4]
---
# fn-227-parameterized-models-declared-worlds.3 Activity retries and retained-lifecycle refinement checkpoint

## Description
Deliver the retry/refinement consumer after phase/rejection checkpoint P (.11), preserving this original task identity. Extend the existing activity Model in place, preserving its direct deadline behavior until fn-237.3 (moved from .4 on 2026-10-21). Retain all original lifecycle evidence and prove the approved checkpoint relation; final Tick-era compatibility belongs to fn-237.3. The approved parent at `bbf0835fbf` governs these boundaries.

**Size:** M after P. Model retry state/actions, adapting the existing realization and proving one retained-graph comparison form one activity checkpoint. P separately owns phase/reject/helper/law source admission; no laws, native/TLC evidence or refinement control are omitted.
**Dependencies:** Retain fn-227-parameterized-models-declared-worlds.2 and consume .11 (P), which consumes .8 (I). The saved direct .2 edge remains intentional. No F/O requirement (now fn-237.1/fn-237.2) while this checkpoint retains old deadline actions and empty facts.
**Files:** `model/temporal/feature/activity/{model.rs,tests.rs,temporal.rs}`; `model/temporal/feature/activity/expected/**` for new activity evidence; a new owned `model/temporal/feature/activity/compatibility/**` reference/harness input tree outside consumer discovery; `model/temporal/realization/tests/realization.rs`; `tests/testpilot_model_slice_test.go` only for necessary existing-control adaptation; `model/temporal/hosts/go/umpire_temporal_guest.wasm`; `model/README.md`.
**Touches:** [model/temporal/feature/activity/**, model/temporal/realization/tests/realization.rs, model/temporal/hosts/go/umpire_temporal_guest.wasm, tests/testpilot_model_slice_test.go, model/README.md, model/temporal/realization/tests/declarative.rs, model/temporal/realization/tests/ui/realization_alias.rs, model/temporal/realization/tests/ui/realization_external_alias.rs, model/temporal/realization/tests/ui/realization_source_scope.rs, model/temporal/realization/tests/ui/realization_variant_alias.rs, model/umpire/catalog/syntax/src/services.rs, model/umpire/catalog-build/src/services.rs, model/umpire/catalog-build/tests/neutral.rs]

### Required catalog prerequisite (conductor scope amendment)

The retry consumer now supplies tuple `Poll(attempt)` and struct `RespondFailed { retryable }` variants already supported by the DSL. The shared RPC classifier introduced by fn-233 only accepts unit variants, and its reverse generator reuses an expression as a pattern. The actual activity build fails with `expected actor action variant`. Extend only `catalog/syntax/src/services.rs`, `catalog-build/src/services.rs` and the nearest `catalog-build/tests/neutral.rs` controls to carry a representative constructor expression separately from its payload-wildcard reverse pattern. Coverage identity remains the actor/variant, and distinct representative payloads must not evade duplicate variant-to-RPC binding rejection. Unit variants and Kettle behavior stay unchanged. Restore the generated Temporal reverse binding; no handwritten bypass or new DSL construct. Retain the red build, syntax/generated-compilation positives, duplicate/malformed/exhaustiveness negatives and both-consumer checks. This bounded prerequisite is necessary for R3/R4 consumption within this task, not a new work lane.

The existing `model/temporal/realization/tests/declarative.rs` control also constructs the formerly unit World. Mechanically adapt its constructors to the explicit three-attempt fixture World and its callback fixtures to real attempt one; retain all declarative projection/coverage assertions.

The four existing alias/source-scope UI controls (`realization_alias.rs`, `realization_external_alias.rs`, `realization_source_scope.rs`, `realization_variant_alias.rs`) likewise need only their `main` unit-World value changed to `World::new(3)`. Keep their intended compile-fail diagnostics; do not update snapshots to absorb an unrelated constructor error.

### Approach

1. Before any activity expectation change, preserve the original report/rows/states, cover expansions/coverage and snapshot metadata as immutable independent reference under the owned compatibility tree. Validate those copies against the original manifest/commit. Keep baseline provenance in the harness. Preserve the original behavior source in Git; do not add a sibling executable Model.
2. Introduce the bounded maximum-attempt World, attempt-one start, payload Poll and retryable failure using .2/.7. Reuse the authored old direct deadline transitions at this checkpoint. Wrong attempt rejects. Retry below the cap advances attempt and schedules; cap/nonretryable failures close. Extend both terminal laws across every terminal phase/attempt, including rejecting loops.
3. Implement a test-local explicit state/action/fact projection and exact canonical projected graph comparison against the immutable old lifecycle reference. Document treatment of input payload assignments and deduplication, and compare starts, reachable states, enabled action/result/destination relationships and rejected terminal behavior. Do not compare new rendered state bytes to old bytes or weaken the reference by dropping old deadline edges. Mutation controls change a reference lifecycle edge and must fail the same comparison. Record the one-attempt/direct-deadline witness as this spec's R4 compatibility witness; fn-237.3 owns the final one-tick World and Tick projection.
4. Update default three-attempt goldens and fn218 cover alternatives with an explicit Cases(n). Verify actual source/TLC agreement through P, not native graph emission. Adapt existing realization callbacks to Poll's observed attempt and map Fail to retryable=false until .5 supplies actual retryability. Keep old live control intentions and report all necessary expectation changes explicitly.

### Investigation targets

Required:

- `model/temporal/feature/activity/model.rs` and `tests.rs` for lifecycle edges, terminal law matrix and cover alternatives.
- `.flow/artifacts/fn-227-parameterized-models-declared-worlds/original-artifacts.sha256` for exact retained evidence.
- `.flow/tmp/fn227-exact-compatibility-research.md` for the static distinctions that the approved projection must state, without reintroducing its superseded approval gate.
- `model/temporal/feature/activity/temporal.rs` and `model/temporal/realization/tests/realization.rs` for observed attempts and fail adaptation.
- `tests/testpilot_model_slice_test.go` for current Complete/Fail/HostRefusal/Cancellation controls.

### Verification

Quick from model/: `mise exec -- cargo test -p umpire-temporal-tests`; `mise exec -- cargo test --manifest-path temporal/realization/Cargo.toml`; run activity backend controls under `UMPIRE_BACKENDS=require`. Run the reference hash/canonical graph harness plus changed-edge and terminal-law mutations. Scoped Rust fmt/strict clippy and changed-Go lint as applicable.

Full from root: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`, `make umpire-guest`, `make umpire-check-model-slice`, `make umpire-check-model-gaps`. Preserve unrelated bytes, original compatibility bytes and all registered laws; update only reviewed activity evidence. Retain native/TLC counts, Cases(n), actual timings and the exact refinement relation/result.
## Acceptance

- [ ] One/three-attempt retry controls, cap/nonretryable failures and stale Poll rejection hold; all terminal phase/attempt laws retain their positive and broken-law controls.
- [ ] The unchanged old evidence is preserved and verified; the stated checkpoint projection passes exact retained-graph comparison, and changing an old lifecycle edge fails it. No required deadline edge is omitted.
- [ ] Actual independently lifted activity TLC agrees on states/verdicts and replays payload/outcome counterexamples, with every law retained.
- [ ] Default activity goldens/coverage state Cases(n); existing realization/live controls pass with only the necessary normal refresh and fatal-only interim Fail mapping.
- [ ] Unrelated golden/expansion bytes stay exact, full gates and timing/count evidence are retained, and the final one-tick compatibility witness is explicitly handed to fn-237.3.
- [ ] Shared RPC classification/reverse generation handles unit, tuple and struct action variants; payload-wildcard matching covers all payload values without weakening duplicate-variant or exhaustive-match controls, and both existing consumers pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
