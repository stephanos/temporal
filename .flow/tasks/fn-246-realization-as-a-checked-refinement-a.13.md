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
4. Own a shared neutral parameter contract provisionally at `model/temporal/shared/protocol/activity/parameters.rs`, neutral observable-view types and protocol-full-state-to-view conversion, plus the protocol-local World adapter. .3 owns the feature-local World adapter, candidate-state-to-view compatibility and total full-state map in `feature/activity/realization.rs`; no shared module imports the feature. Validate exact parameters against both independent World schemas and account every full-state field, retaining field/domain/type provenance. Reject omissions/misnaming, wrong kinds/bounds, erased budgets, clamping/defaults and lost None/present-zero distinctions. Keep local Worlds separate; no generic imported/shared-World admission. Reanchor fn-237.3 constructors/accessors after committed fn-247. Snapshot decoding only constructs an observable target, never the protocol's latent full state. Preserve independent semantic lifecycle/checkpoint evidence in Git through semantic checking; retain the historical original-artifact hash manifest only as provenance; report an exact disagreement rather than fit either Model.

**Dispatch exclusion:** Serialize with .2 on realization/src/lib.rs; the parent overlap guard applies even when the task DAG otherwise permits parallel work. Preserve existing dependencies.

### Investigation targets

**Required:**

- `model/temporal/feature/activity/model.rs` and final fn-237.3 World/cancellation contract artifacts for comparison inputs and admitted domains only.
- `model/temporal/foundation/api.rs` and pinned standalone API/cancellation protocol evidence for independent request/snapshot types.
- `model/umpire/dev/testing/src/runner.rs` and `backend.rs` for native/TLC fixture ownership.
- `model/temporal/testing/fixtures.rs` and `build.rs` for current fixture discovery.
- `model/temporal/feature/activity/compatibility/checkpoint.rs:82-93` (semantic lifecycle comparison) for independent evidence; original-artifact hash authority is retired; the protected historical manifest remains provenance.
- `model/umpire/backends/tlc` existing World/state-set agreement tests for comparable-scope controls.

### Verification

From model/: `UMPIRE_BACKENDS=require mise exec -- cargo test -p umpire-temporal-tests --test fixtures` requires backend execution through the current fixture harness. Run `mise exec -- cargo test -p umpire-temporal-tests --test snapshot_ownership` for the new shared protocol snapshots. Run focused native/TLC selected-World controls, fmt and strict clippy. Record actual backend tool/version/World/bound/counts; no TLC refinement claim.

## Acceptance
- [ ] Protocol states/actions/outcomes and invariants have independent primary contract/protocol provenance for every selected World and cover final prerequisite dimensions without copying feature enablement.
- [ ] Full protocol state retains independently authored latent budget/timer laws, separate public view and exact every-field accounting. Countdown/reset/spanning/priority/expiry-fact/terminal-clear and erased-budget mutants fail even when public snapshot classes agree; snapshots never manufacture a full protocol state.
- [ ] Native graphs/goldens and required TLC own-invariant/state-set agreement execute for comparable finite scopes; wrong World, incomplete, unsupported and mismatching answers fail. The snapshot-ownership gate includes the new shared protocol snapshots and passes as a separate integration-test target.
- [ ] Terminal outcome, retained retry history, cancellation/heartbeat/acknowledgement and rejected RPC outcomes remain distinct; no projected feature edge defines protocol behavior.
- [ ] Shared parameter contract and protocol-local adapter account for every independently declared World field with exact name/kind/domain/type provenance. Omission, misnaming, wrong bounds/kind, clamping/default substitution and None/present-zero mutants fail; no imported-World front-end admission is added.
- [ ] Original lifecycle evidence still supports the independent semantic refinement proof. The task .15 corrected feature baseline gains only the approved pending-cancellation worker-failure/local-timeout Failed/TimedOut correction, with renewed independent native/TLC evidence and primary contract provenance. Scheduled cancellation and explicit worker acknowledgement remain Canceled; task .15 retry behavior and every unrelated Model policy remain unchanged. Representation/goldens may regenerate. Focused backend/fixture evidence includes actual counts, bounds and tool identity; no online or TLC-refinement proof is claimed here.


## Done summary
Authored an independent full standalone-activity protocol Model with latent budgets/timers, RPC outcomes, separate terminal/retained failure, neutral snapshot view and exact local-World/parameter schema provenance. Applied only the owner-approved pending worker-failure/local-expiry Failed/TimedOut feature cells; scheduled cancellation, acknowledgement, atomic compatibility and task15 retry semantics remain pinned. No imported-World frontend admission or feature-derived protocol edges.

Renewed required evidence: protocol22 tests across12 complete Worlds (bound4096;432 summed states/2506edges/192 own-law rows),20 source-mutant disagreements, feature policy15 tests across33 Worlds plus exact pending/retry controls, default1 and required complement7. Exact fixture partition covers62active=45required+17mode-independent native, with2inherited ignores unchanged. Shared exact-source agreement unit supplies missing-state/verdict/cut/missing-claim refusal; protocol tests independently refuse wrong World/incomplete/cut/unsupported and changed typed expiry facts. No fabricated protocol-specific TLC state-set mutation is claimed.

Final correction uses public production parameter validators rather than a duplicate test copy; native62, required protocol22, strict focused Clippy/fmt and changed-input guest freshness pass. Protocol World/mutant reports remain byte-identical; feature15+7 evidence reuses82 exact unchanged feature/consumer inputs. Snapshot ownership2 and named adapter1 pass. Current driver reproduces the3,988,325-byte guest/canary hash8b9ea3c0b29ab0b0f727cf083027c9ad9a6886634d8ce8be25a84115e315be50. Protected original lifecycle records remain historical provenance; active checkpoint uses semantic1start/6state/11edge comparison with changed-edge rejection. Pinned API source/archive/origin/go.sum proof retained.

stage: impl-review - ran (model: claude-fable-5-1 at high); initial correctness/contracts/integration SHIP onac7679cb09, then retained parameter-reuse/timing/atomic-doc findings fixed and single contracts follow-up SHIP on8eeb9a85af. Final reviewer-requested README evidence-link correction verified locally; no third review for link-only change.
stage: completion-review - skipped(policy: unfinished spec tasks).
Gates: focused required fixtures/ownership/lint/format/guest evidence; broad current-input/combined gates remain fn-254.8. No online binding, native refinement, TLC-refinement, observable-view commutation or full live/gaps proof claimed here; those remain .3/.4, fn237.4 and the named live tasks. Retained .9 source remains unintegrated; code-generation ownership was coordinated by exact guest/runtime declaration inspection without runtime/proto/Go edits.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: aaba137190, 5d8c856521, ac7679cb09, 2ac14b2163, 8eeb9a85af, 8e81e2ba7d
- Tests: Required protocol22 current-source tests pass:12 finite Worlds,16laws each,20 source mutants; scopes/mutant reports unchanged after production-validator reuse, Required feature policy15 plus default1 and seven-test require-sensitive complement pass; exact62active/2ignored fixture coverage partition retained, Native fixtures62pass; snapshot ownership2pass; named adapter1pass; atomic1/6/11 semantic checkpoint and changed-edge rejection pass, Strict focused umpire-temporal/umpire-temporal-tests all-target Clippy and formatting pass; unsupported timing text removed, Current-driver guest reproduction and changed-input --check pass with unchanged guest/canary8b9ea3 hash, Shared exact-source agreement complete-state/verdict/cut/missing-claim negatives reused from recorded fn253.5 required13-test support run, Final source envelope247primary+101driver/tool sources verified; source refresh retains identical executed fixture binary; final link-only documentation check passes
- PRs:
## Approved cancellation correction and parameter boundary (2026-10-26)

The explicit user reply is retained at `.flow/artifacts/fn-246-realization-as-a-checked-refinement-a/task13/planning/approved-api-correction.json`. Amend the independent contract from primary pinned API evidence, correct the feature cancellation cells and author the new protocol independently, then renew native/TLC proofs. Preserve historical evidence rather than rewriting the old chosen-policy artifact. New shared protocol snapshots belong to the snapshot-ownership gate.

Raw Start requests and effective modeled budgets are separate types/contracts. New raw admission must implement and test the pinned explicit defaults at its named boundary; effective parameters remain exact, with None and present-zero distinct and no hidden fallback/clamping. Existing all-absent or otherwise non-API-realizable checker Worlds remain explicitly labelled abstract scopes, never claimed accepted raw requests. Preserve their exploration rather than silently excluding them. Do not change the feature Model's timeout-absence policy without a separate concrete decision if one becomes necessary.

Conditional generated guest and canary modelHash paths are declared for actual changed-input reproduction/publication only; compare exact bytes first. Coordinate code-generation ownership with task9 before publication. No Task9 runtime/proto/Go edits belong here.
