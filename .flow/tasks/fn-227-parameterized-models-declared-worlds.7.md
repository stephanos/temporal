---
satisfies: [R1, R3]
---
# fn-227-parameterized-models-declared-worlds.7 After native World: independent typed source binding, whole-state Init and parameterized_world TLC proof

## Description
Completes R1's shared fixture/TLC evidence and R3's World/computed-start source lowering after native World authoring. Consumes the predecessor's owned World/type association and complete computed starts. Inputs, Phase/Phased/reject source admission and activity changes remain in their existing tasks.

**Size:** M. Adapt the existing source Type/Expr admission, IR and five CFG emitters through one typed binding route. Existing expression/claim lowering and replay remain the implementation; a second evaluator or binding subsystem is outside this task.
**Files:** `model/umpire/lift/src/{lib,tree,shape,emit,build,names}.rs`, `model/umpire/lift/src/body/{mod,claim,expr,effects,progress}.rs` as mechanical environment/signature consumers; focused lift tests; `model/umpire/tests/{build.rs,fixtures.rs,fixtures/parameterized_world/**}` and authoring README.
**Touches:** [model/umpire/lift/**, model/umpire/testing/src/backend.rs, model/umpire/tests/**, model/README.md, model/temporal/hosts/go/umpire_temporal_guest.wasm]

### Approach

1. Preserve source-only unit APIs and their exact artifacts. Add an explicit bound route using the current source Type/value representation for ordered World fields plus ordered complete initial states. Validate names, counts and types against the source declarations. The fixture generator must produce a compiler-checked projection from one selected native World and its authored computed starts into this binding; use that same owned World for native checking and backend replay. Constructor strings or a separate hand-maintained value table are insufficient.
2. Keep source/build admission independent of executing Rust transitions. Build generation discovers/adopts the authored World schema and generates typed binding code for selected fixtures; the new bound route may finish value binding/emission at runtime when the concrete fixture World exists. Existing unit fixtures keep their old eager generation path. Complete starts may be enumerated from the authored starts method, as R1 requires; native reached states/edges must never define Next or claim meaning. No arbitrary Rust evaluator, reflection registry or new third-party dependency.
3. Introduce distinct lookup identities for World constants, source constants, state and before-state. Apply native typed trailing-World classification before old argument arity. Reuse located Diagnostic and collision-safe name generation. Unsupported World field types and absent/missing/extra/duplicate/wrong-type binding data must refuse with source location, including malformed complete states. An unsupported generic source form stays explicitly refused; native generics do not authorize a silent backend substitution.
4. Emit World CONSTANT declarations and the same selected assignments in main, invariant, postcondition, negative and progress CFGs. Emit computed Init as alternatives of complete states, preserving correlation/dedup/empty semantics. Keep all independent transition/claim translation, disabledness, before/outcome identities and replay controls intact. New action-input semantics stay in .2; no activity admission shortcut.
5. Add `parameterized_world` using two selected values of one two-parameter World, a world-bounded claim, correlated starts and a deliberately false claim with a replayable path. Reuse discovery/fixture consumers and render_rows if graph size requires it. Execute real binding-sensitive TLC controls for every CFG family, including negative/progress, rather than only matching emitted CONSTANT text. Add native report/rows/states/paths and TLC/source artifacts only for new fixtures/controls.

### Investigation targets

**Required** (read before coding):
- `model/umpire/lift/src/lib.rs:179` - source admission API and current IR handoff.
- `model/umpire/lift/src/shape.rs:91` - starts and source type admission; follow body/claim World/before lookup.
- `model/umpire/lift/src/emit.rs:51` - Cartesian Init and all five CFG construction paths.
- `model/umpire/lift/src/build.rs:7` - eager fixture artifacts and generated native constructor/replay binding.
- `model/umpire/tests/fixtures.rs:10` - registered source/TLC consumers and required-backend coverage.
**Optional**:
- `model/umpire/tests/fixtures/sample_kettle/tests.rs` - fixture/native rendering pattern.
- `model/umpire/lift/tests/emission.rs` - empty Init and stable source artifact controls.

### Verification

Quick from `model/`: `mise exec -- cargo test -p umpire-lift -p umpire-tests`, repeated with `UMPIRE_BACKENDS=require` for actual backend controls; scoped strict clippy and formatting. Compare every pre-existing golden/expansion path against the pinned task baseline, including metadata. No global model-update or source-derived-transition replacement with native exploration.

Full from repository root: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`. Preserve timing output, byte comparison, source/type refusal controls, actual TLC logs and same-World replay evidence. If linked guest bytes change, rebuild with the existing builder, pin hash/size and run the unchanged embedded slice/gap/privacy targets. Preserve current activity Model, goldens, ABI, panic policy and protocol.

## Acceptance
- [ ] A compiler-checked per-fixture route binds one selected native World and its complete authored starts to the source schema, emission and replay. Missing/extra/duplicate/wrong-type World values, malformed complete states, absent binding and unsupported field types have located refusal controls; old source-only unit APIs/artifacts are unchanged.
- [ ] World/source/state/before identities remain distinct under colliding field names and renamed arguments, including every supported postcondition form. Source transitions and claims remain independently translated; no native reached graph supplies their meaning.
- [ ] Whole-state Init admits correlated pairs without cross-products, preserves duplicate/empty semantics and uses the exact selected World binding. Controls distinguish two values of the same World and deliberately false claim verdicts.
- [ ] Main, invariant, postcondition, negative and progress CFGs carry identical selected assignments and pass real binding-sensitive TLC controls with same-World counterexample replay. Existing source refusal boundaries and required-backend coverage checks remain intact.
- [ ] The new two-parameter `parameterized_world` fixture has native report/rows/states/paths plus source/TLC artifacts and actual agreement. All old goldens/expansions are byte-identical; Quick/full gates pass with timing output retained and any necessary rebuilt-guest live slice/gap/privacy proof. Together with native predecessor this discharges the World portion of R1/R3; input and activity portions remain open.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
