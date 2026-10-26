---
satisfies: [R5]
---
# fn-246-realization-as-a-checked-refinement-a.13 Independent standalone-activity protocol Model and native/TLC invariants

## Description
Author R5's independent standalone-activity protocol Model and native/TLC fixture evidence. This branch of work consumes the admitted final fn-237.3 World contract, while the online binding stays with .3 and the generic refinement consumer with .4.

**Size:** M
**Files:** new shared protocol/activity Model, its authored tests and expected outputs, fixture discovery/exports only where current conventions require registration.
**Touches:** [model/temporal/shared/protocol/activity/**, model/temporal/testing/fixtures.rs, model/temporal/testing/build.rs, model/temporal/realization/src/lib.rs, model/temporal/testing/snapshot_ownership.rs, model/temporal/feature/activity/**, model/temporal/hosts/go/umpire_temporal_guest.wasm, tools/canary/policy/production-canary.json]

### Approach

Consume task .15's committed, independently proved corrected timeout-retry baseline. The 2026-10-26 explicit user approval additionally corrects pending-cancellation worker-failure/local-timeout outcomes to the pinned API Failed/TimedOut results. Scheduled cancellation and explicit worker acknowledgement remain Canceled. Other Model policies and the original atomic lifecycle checkpoint remain unchanged; retain task15 retry behavior.

All source coordinates are provisional planning anchors. Reanchor final fn-237.3 fields, emitted graph, accessors and fn-247 xtask/build owners after their committed closures before implementation. The current fn-237.3 source checkpoint and fn-247 plan do not satisfy those gates.

1. Reanchor final World domains, bounded Tick/timeout/failure/cancel/heartbeat dimensions and pinned standalone API/cancellation contract before authoring. Document the independent contract/protocol evidence behind each state class, RPC outcome and invariant. Existing feature transitions are a comparison target, never the source of protocol edges.
2. Define a plain full protocol machine with independently authored latent timers/Remaining budgets, RPC/gRPC outcomes and a separate public snapshot view. Finite attempt/time/budget domains come from its local selected World via the shared exact parameter contract. Author complete starts, arming/reset/spanning/decrement/priority, expiry cause/fact and terminal-clearing semantics; do not erase hidden state to fit a visible snapshot table or feature graph. Keep terminal outcome and retained failure separate and pending cancellation/heartbeat/ACK distinct, without inventing worker evidence production owned by fn-237.4. Explicit protocol timer actions remain where required by its independent semantics.
3. Explore natively and retain states/edges/starts/outcomes/invariants per selected admitted World. Add required TLC own-invariant and state-set agreement through existing fixture/backend paths, using explicit comparable World/bound reports. Increase selected Worlds where required by the pinned contract; incomplete/unsupported backend results cannot pass.
4. Own a shared neutral parameter contract provisionally at `model/temporal/shared/protocol/activity/parameters.rs`, neutral observable-view types and protocol-full-state-to-view conversion, plus the protocol-local World adapter. .3 owns the feature-local World adapter, candidate-state-to-view compatibility and total full-state map in `feature/activity/realization.rs`; no shared module imports the feature. Validate exact parameters against both independent World schemas and account every full-state field, retaining field/domain/type provenance. Reject omissions/misnaming, wrong kinds/bounds, erased budgets, clamping/defaults and lost None/present-zero distinctions. Keep local Worlds separate; no generic imported/shared-World admission. Reanchor fn-237.3 constructors/accessors after committed fn-247. Snapshot decoding only constructs an observable target, never the protocol's latent full state. Preserve independent semantic lifecycle/checkpoint evidence in Git, without the removed original-artifact hash manifest; report an exact disagreement rather than fit either Model.

**Dispatch exclusion:** Serialize with .2 on realization/src/lib.rs; the parent overlap guard applies even when the task DAG otherwise permits parallel work. Preserve existing dependencies.

### Investigation targets

**Required:**

- `model/temporal/feature/activity/model.rs` and final fn-237.3 World/cancellation contract artifacts for comparison inputs and admitted domains only.
- `model/temporal/foundation/api.rs` and pinned standalone API/cancellation protocol evidence for independent request/snapshot types.
- `model/umpire/dev/testing/src/runner.rs` and `backend.rs` for native/TLC fixture ownership.
- `model/temporal/testing/fixtures.rs` and `build.rs` for current fixture discovery.
- `model/temporal/feature/activity/compatibility/checkpoint.rs:82-93` (semantic lifecycle comparison) for independent evidence; the original-artifact hash manifest is removed.
- `model/umpire/backends/tlc` existing World/state-set agreement tests for comparable-scope controls.

### Verification

From model/: `mise exec -- cargo test -p umpire-temporal-tests --test fixtures`; require backend execution with `UMPIRE_BACKENDS=require` through the current fixture harness. Run focused native/TLC selected-World controls, fmt and strict clippy. Record actual backend tool/version/World/bound/counts; no TLC refinement claim.

## Acceptance
- [ ] Protocol states/actions/outcomes and invariants have independent primary contract/protocol provenance for every selected World and cover final prerequisite dimensions without copying feature enablement.
- [ ] Full protocol state retains independently authored latent budget/timer laws, separate public view and exact every-field accounting. Countdown/reset/spanning/priority/expiry-fact/terminal-clear and erased-budget mutants fail even when public snapshot classes agree; snapshots never manufacture a full protocol state.
- [ ] Native graphs/goldens and required TLC own-invariant/state-set agreement execute for comparable finite scopes; wrong World, incomplete, unsupported and mismatching answers fail.
- [ ] Terminal outcome, retained retry history, cancellation/heartbeat/acknowledgement and rejected RPC outcomes remain distinct; no projected feature edge defines protocol behavior.
- [ ] Shared parameter contract and protocol-local adapter account for every independently declared World field with exact name/kind/domain/type provenance. Omission, misnaming, wrong bounds/kind, clamping/default substitution and None/present-zero mutants fail; no imported-World front-end admission is added.
- [ ] Original lifecycle evidence still supports the independent semantic refinement proof and the task .15 corrected feature baseline gains no further Model-policy changes; representation/goldens may regenerate. Focused backend/fixture evidence includes actual counts, bounds and tool identity; no online or TLC-refinement proof is claimed here.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

## Approved cancellation correction and parameter boundary (2026-10-26)

The explicit user reply is retained at `.flow/artifacts/fn-246-realization-as-a-checked-refinement-a/task13/planning/approved-api-correction.json`. Amend the independent contract from primary pinned API evidence, correct the feature cancellation cells and author the new protocol independently, then renew native/TLC proofs. Preserve historical evidence rather than rewriting the old chosen-policy artifact. New shared protocol snapshots belong to the snapshot-ownership gate.

Raw Start requests and effective modeled budgets are separate types/contracts. New raw admission must implement and test the pinned explicit defaults at its named boundary; effective parameters remain exact, with None and present-zero distinct and no hidden fallback/clamping. Existing all-absent or otherwise non-API-realizable checker Worlds remain explicitly labelled abstract scopes, never claimed accepted raw requests. Preserve their exploration rather than silently excluding them. Do not change the feature Model's timeout-absence policy without a separate concrete decision if one becomes necessary.

Conditional generated guest and canary modelHash paths are declared for actual changed-input reproduction/publication only; compare exact bytes first. Coordinate code-generation ownership with task9 before publication. No Task9 runtime/proto/Go edits belong here.
