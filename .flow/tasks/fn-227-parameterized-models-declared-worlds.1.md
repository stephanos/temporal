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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
