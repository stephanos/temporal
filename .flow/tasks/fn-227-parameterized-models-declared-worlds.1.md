---
satisfies: [R1, R3]
---
# fn-227-parameterized-models-declared-worlds.1 Declared World, typed world adapters and computed starts with owned native suite binding

## Description
Implements the native portion of R1 and the typed binding handoff consumed by R3. Source lifting and the shared native/TLC fixture move to the new dependent World-source task. This checkpoint must leave a usable native World, including the suite APIs; a derive that works only with direct checker calls is incomplete.

**Size:** M. Four cohesive acceptance groups adapt the existing State derive, Machine callbacks, typed registration and retained suite machinery. Expected mechanical bound propagation makes the file count wider than the core four macro/trait files; no new checker or search subsystem is planned.
**Files:** `model/umpire/dsl/macros/src/{lib,machine,state,world}.rs`, `model/umpire/dsl/src/{lib,machine,action,path_build,path_input}.rs`, `model/umpire/suite/src/{space,run,covered,lib}.rs`; new focused DSL/macro/suite tests and README authoring examples.
**Touches:** [model/umpire/dsl/**, model/umpire/suite/**, model/README.md, model/temporal/hosts/go/umpire_temporal_guest.wasm]

### Approach

1. Keep the existing absent-World expansion branch literally unchanged. Add the World derive by adapting State's existing value-trait/bound/rendering machinery, without synthesizing starts. Generate `World::new` in field order with the struct's visibility. Preserve generic Model/World/action bounds and constructor accessibility. Use the existing Machine-as-World identity.
2. Classify an optional trailing immutable reference to the declared World type before interpreting the remaining old positional signature. Names cannot classify arguments. Preserve every old before/outcome postcondition form and all ordinary/staying/rejecting dispatch branches. Apply World access to handlers, invariant, postcondition, assumption, progress and per-method `#[scope]`. The latter is a compatibility spelling for the same assumption registration/order; grouped `#[scopes]` remains refused. No claim removal or weaker readonly/reject admission.
3. Let `#[starts] impl Model` suppress only State's generated static starts method and its irrelevant Default bounds. Preserve field starts otherwise, reject both declarations even for an empty range, and pass the actual World into authored computed starts. Retain the checker's existing complete-state deduplication and order.
4. Separate state-to-World type association from implicit unit-World construction with one additive compatibility bridge for existing manual/generated Registered implementations. Declared states must retain typed actor/path inference while requiring an explicitly supplied World. Do not invent Default values or require Copy. Extend Space/Run binding over the existing retained ownership machinery so verify/refuted/find/cover/sample and retained Run rows all use one selected World, including zero starts. Existing state-only APIs remain source compatible. Retain owned World parameters and complete starts as the sole handoff to the source task; expose existing field names/types rather than a second mirror catalog or serialization subsystem.
5. Establish native RED/GREEN controls first, then compare every pre-existing golden and expansion artifact against the pinned task baseline. Native generics are required here; action domain enumeration stays in .2 and independent source type admission stays in the dependent source task. Document only delivered native spelling and the pending TLC checkpoint.

### Investigation targets

**Required** (read before coding):
- `model/umpire/dsl/macros/src/machine.rs:313` - signature admission, registration and all stepping adapters.
- `model/umpire/dsl/macros/src/state.rs:5` - bounded derives and the generated starts collision.
- `model/umpire/dsl/src/machine.rs:104` - current association/construction contract; follow its action/path consumers.
- `model/umpire/suite/src/space.rs:15` - implicit construction; follow Run/covered retained ownership consumers.
- `model/umpire/suite/tests/retained.rs` - manual registration and non-Clone outcome compatibility.
**Optional**:
- `model/umpire/dsl/macros/tests/fixtures.rs` - expansion and compile-refusal registration.
- `model/umpire/checker/src/explore.rs:275` - existing shared-start deduplication, to reuse unchanged.

### Verification

Quick from `model/`: `mise exec -- cargo test -p umpire-dsl-macros -p umpire-dsl -p umpire-suite`; then scoped strict clippy and formatting. Add new expansion/trybuild artifacts only. Compare old golden/expansion paths, including metadata, with the task baseline using Git; no global model-update.

Full from repository root: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`. Record actual timings and soft-target misses. If actual DSL linkage changes the embedded asset, use the existing guest builder, retain hash/size, and run `make umpire-check-model-slice` plus `make umpire-check-model-gaps` on those exact bytes. No freshness bypass, ABI/protocol/panic change, new dependency/version or activity Model/golden rewrite.
## Acceptance
- [ ] World value derives and field-order constructor compile with authored visibility and generic bounds across Model/World/actions; non-Copy/non-Default parameters work. Old unit/manual registrations and author-name rendering are unchanged.
- [ ] Typed trailing World dispatch passes the selected value through every adapter and ordinary/staying/rejecting branch. Controls distinguish before-only/World-only and combined before/outcome/World postconditions, rename arguments, cover assumption/scope/progress, and pin located refusals for misplaced/duplicate/mutable World parameters.
- [ ] Computed starts preserve whole-state correlation, stable first occurrence, duplicate deduplication and empty starts without irrelevant Default bounds. Trybuild pins duplicate World, mixed starts (including empty static ranges) and wrong-position World; old field-start expansions remain byte-identical.
- [ ] Explicit native suite binding retains one owned World through verify/refuted/find/cover/sample and retained Runs, including empty starts and after the source variable leaves scope. Two values of one World type change starts/guards and a claim verdict; mismatched World/state fails typing. Quick/full gates and old-artifact baseline comparison pass, timings and any regenerated-guest live proof are retained. Source/TLC completion remains assigned to the dependent source task.
## Done summary
Native model authors can declare a typed World, use computed correlated starts and trailing World callbacks, and retain one owned World through suite queries and Run rows. The additive Associated bridge preserves manual/unit registration, while Space<S, W = ()> preserves legacy unconstrained configuration, inference and auto-traits.

Tier: session (jev-unavailable(no_key)); project implementer pin retained

Baseline was fresh green for the literal Quick command and all three full gates. Final Quick, formatting, strict scoped clippy, make model-check, UMPIRE_BACKENDS=require make model-check and make model-time all exited 0. Required-backend wall time was 182.436s. Final timing was 93.288s for the gate (30s soft target missed) and 5.035s for the inner loop (10s met). All three normal full-gate receipts were checked HONORED at 155a7b29; no full gate was skipped.

All 571 original artifacts and 101 original UI diagnostics match their pinned hashes and Git metadata. The guest remains 3,329,990 bytes with SHA-256 10724102d03e1659f28b07c3a0c848d2cac5876668f3be9261d0a3b65023cf06. The gaps golden remains 879,197 bytes with SHA-256 a3974ea8da0f3c46234770068fe0f163d5e17529b275cea086c2292a124284a0. No regenerated-guest live gate or cross-OS identity proof is claimed.

Full commands, raw-log paths/hashes, RED/GREEN controls and audit reports are retained under .flow/artifacts/fn-227-parameterized-models-declared-worlds/task-1-{verification,audits,review-resolution}.md. Permanent formal receipts and verbatim reports are task-1-review-round1-* and task-1-review-round2-*. The canonical continuity receipt is /tmp/impl-review-receipt-8f37faba39e2-fn-227-parameterized-models-declared-worlds.1.json.

stage: impl-review - ran [2026-10-21T15:18:48.593876Z..2026-10-21T15:48:14.374964Z] (codex:gpt-6.1-sol:high; NEEDS_WORK then SHIP; two rounds, same-family review)
stage: quality-audits - ran (correctness and standards; frozen-range reports and rechecks preserved)
stage: memory - ran (bug/build-errors/preserve-generic-declaration-context-in-2026-10-21)
stage: plan-sync - skipped(config: planSync.enabled=false)
stage: qa - skipped(config: pipeline.qa=off)

Base is a76c3581620c39d8ccb760bc4af34d4c24b1d618; executable source ends at f0adb5afdb659614cf418a7f420106a85718ba06. Interleaved bbf0835fbf3f536a954adae8dca3ef19795627d4 and 155a7b295990f457cd85ec9e77ecfd18a1cc96eb are root-owned parent-contract and foreign fn232 evidence/documentation commits, excluded from native ownership. The evidence range retains this provenance; the later closure receipt is committed separately.

Only this native checkpoint is complete. Source lifting, private-field projection and the shared native/TLC fixture remain task .7; parent R1/R3 and the parent plan stay open. Authored field privacy is preserved with no reflection/getter catalog. Both task worktrees were integrated and removed normally, foreign fn233 files were preserved, and all worker commands are terminal. Root's explicit incremental-cache cleanup sublease is recorded in .flow/tmp/fn2271-incremental-cleanup.md; final free space was 34GiB. Source/index/cache leases return to root at handback.
## Evidence
- Commits: c3d85d9cf9387b3aaa7fdf44b5c399c91d4c6530, b5b91b4f82d46ea3e4c40b11e0a7b77bc5d1e865, 9147fbdf6b81869bc8141ac120458509ac5f30b3, 8d429e97409b1537ecb347c359129bffafa63f78, f0adb5afdb659614cf418a7f420106a85718ba06, bbf0835fbf3f536a954adae8dca3ef19795627d4, 155a7b295990f457cd85ec9e77ecfd18a1cc96eb, bc6afeba8e37262d8908a92019ced32ac005a34b
- Tests: cd model && mise exec -- cargo test -p umpire-dsl-macros -p umpire-dsl -p umpire-suite, cd model && cargo clippy -p umpire-dsl-macros -p umpire-dsl -p umpire-suite --all-targets --all-features -- -D warnings, cd model && cargo fmt --all --check, make model-check, UMPIRE_BACKENDS=require make model-check, make model-time, sha256sum -c .flow/artifacts/fn-227-parameterized-models-declared-worlds/original-artifacts.sha256, sha256sum -c .flow/tmp/fn2271-original-ui.sha256, git diff --check
- PRs: