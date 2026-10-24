---
satisfies: [R5]
---
# fn-246-realization-as-a-checked-refinement-a.13 Independent standalone-activity protocol Model and native/TLC invariants

## Description
Author R5's independent standalone-activity protocol Model and native/TLC fixture evidence. This branch of work consumes the admitted final fn-237.3 World contract, while the online binding stays with .3 and the generic refinement consumer with .4.

**Size:** M
**Files:** new shared protocol/activity Model, its authored tests and expected outputs, fixture discovery/exports only where current conventions require registration.
**Touches:** [model/temporal/shared/protocol/activity/**, model/temporal/testing/fixtures.rs, model/temporal/testing/build.rs, model/temporal/realization/src/lib.rs]

### Approach

All source coordinates are provisional planning anchors. Reanchor final fn-237.3 fields, emitted graph, accessors, golden hashes and fn-247 xtask/build owners after their committed closures before implementation. The current fn-237.3 source checkpoint and fn-247 plan do not satisfy those gates.

1. Reanchor final World domains, bounded Tick/timeout/failure/cancel/heartbeat dimensions and pinned standalone API/cancellation contract before authoring. Document the independent contract/protocol evidence behind each state class, RPC outcome and invariant. Existing feature transitions are a comparison target, never the source of protocol edges.
2. Define a plain full protocol machine with independently authored latent timers/Remaining budgets, RPC/gRPC outcomes and a separate public snapshot view. Finite attempt/time/budget domains come from its local selected World via the shared exact parameter contract. Author complete starts, arming/reset/spanning/decrement/priority, expiry cause/fact and terminal-clearing semantics; do not erase hidden state to fit a visible snapshot table or feature graph. Keep terminal outcome and retained failure separate and pending cancellation/heartbeat/ACK distinct, without inventing worker evidence production owned by fn-237.4. Explicit protocol timer actions remain where required by its independent semantics.
3. Explore natively and retain states/edges/starts/outcomes/invariants per selected admitted World. Add required TLC own-invariant and state-set agreement through existing fixture/backend paths, using explicit comparable World/bound reports. Increase selected Worlds where required by the pinned contract; incomplete/unsupported backend results cannot pass.
4. Own a shared neutral parameter contract provisionally at `model/temporal/shared/protocol/activity/parameters.rs`, neutral observable-view types and protocol-full-state-to-view conversion, plus the protocol-local World adapter. .3 owns the feature-local World adapter, candidate-state-to-view compatibility and total full-state map in `feature/activity/realization.rs`; no shared module imports the feature. Validate exact parameters against both independent World schemas and account every full-state field, retaining field/domain/type provenance. Reject omissions/misnaming, wrong kinds/bounds, erased budgets, clamping/defaults and lost None/present-zero distinctions. Keep local Worlds separate; no generic imported/shared-World admission. Reanchor fn-237.3 constructors/accessors after committed fn-247. Snapshot decoding only constructs an observable target, never the protocol's latent full state. Preserve independent compatibility/checkpoint/hash artifacts; report an exact disagreement rather than fit either Model.

### Investigation targets

**Required:**

- `model/temporal/feature/activity/model.rs` and final fn-237.3 World/cancellation contract artifacts for comparison inputs and admitted domains only.
- `model/temporal/foundation/api.rs` and pinned standalone API/cancellation protocol evidence for independent request/snapshot types.
- `model/umpire/dev/testing/src/runner.rs` and `backend.rs` for native/TLC fixture ownership.
- `model/temporal/testing/fixtures.rs` and `build.rs` for current fixture discovery.
- `model/temporal/feature/activity/compatibility/checkpoint.rs:82-93` and retained original manifest for independent evidence.
- `model/umpire/backends/tlc` existing World/state-set agreement tests for comparable-scope controls.

### Verification

From model/: `mise exec -- cargo test -p umpire-temporal-tests --test fixtures`; require backend execution with `UMPIRE_BACKENDS=require` through the current fixture harness. Run focused native/TLC selected-World controls, fmt and strict clippy. Record actual backend tool/version/World/bound/counts; no TLC refinement claim.

## Acceptance
- [ ] Protocol states/actions/outcomes and invariants have independent primary contract/protocol provenance for every selected World and cover final prerequisite dimensions without copying feature enablement.
- [ ] Full protocol state retains independently authored latent budget/timer laws, separate public view and exact every-field accounting. Countdown/reset/spanning/priority/expiry-fact/terminal-clear and erased-budget mutants fail even when public snapshot classes agree; snapshots never manufacture a full protocol state.
- [ ] Native graphs/goldens and required TLC own-invariant/state-set agreement execute for comparable finite scopes; wrong World, incomplete, unsupported and mismatching answers fail.
- [ ] Terminal outcome, retained retry history, cancellation/heartbeat/acknowledgement and rejected RPC outcomes remain distinct; no projected feature edge defines protocol behavior.
- [ ] Shared parameter contract and protocol-local adapter account for every independently declared World field with exact name/kind/domain/type provenance. Omission, misnaming, wrong bounds/kind, clamping/default substitution and None/present-zero mutants fail; no imported-World front-end admission is added.
- [ ] Original checkpoint/manifest and feature semantics remain byte/hash exact. Focused backend/fixture evidence includes actual counts, bounds and tool identity; no online or TLC-refinement proof is claimed here.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
