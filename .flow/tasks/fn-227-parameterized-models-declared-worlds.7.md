---
satisfies: [R1, R3]
---
# fn-227-parameterized-models-declared-worlds.7 After native World: independent typed source binding, whole-state Init and parameterized_world TLC proof

## Description

Completes R1's shared fixture/TLC evidence and R3's World/computed-start source lowering after native World authoring. Retains dependency on `fn-227-parameterized-models-declared-worlds.1`; its committed native implementation is `f0adb5afdb659614cf418a7f420106a85718ba06` and verified closure is `186d362f72bbc27040ee60443db7c26d4b01dbd8` in umpire. Inputs, Phase/Phased/reject source admission and activity changes remain in their existing tasks. Preserve the owner's recorded activity-contract amendments without resolving or expanding them here.

**Size:** M. Adapt the existing source Type/Expr admission, IR, fixture generation and five CFG emitters through one typed binding route. Existing expression/claim lowering and replay remain the implementation; a second evaluator or binding subsystem is outside this task.

**Files:** `model/umpire/lift/src/{lib,tree,shape,emit,build,names}.rs`, `model/umpire/lift/src/body/{mod,claim,expr,effects,progress}.rs` as mechanical environment/signature consumers; focused lift tests; `model/umpire/testing/src/{discovery,backend}.rs` and `model/umpire/testing/tests/discovery.rs`; `model/umpire/tests/{build.rs,fixtures.rs,fixtures/parameterized_world/**}`; authoring README. Reserve the embedded Temporal guest only for a necessary builder-generated refresh.

**Touches:** [model/umpire/lift/**, model/umpire/testing/src/discovery.rs, model/umpire/testing/src/backend.rs, model/umpire/testing/tests/discovery.rs, model/umpire/tests/**, model/README.md, model/temporal/hosts/go/umpire_temporal_guest.wasm]

### Native handoff

`Associated` identifies a state's World without constructing it; existing `Registered` implementations receive that association through a blanket implementation. Declared states implement `Associated`, and the generated World implements `Machine`. Its `starts(&self)` enumerates complete authored starts. No native reflection, catalog or serialization interface is needed for source binding.

The committed suite uses `Space<S, W = ()>`. `Space::<S>::new(world)` enumerates starts; `Space::with_world(world, starts)` retains already selected starts. Both return a space holding `Rc<S::World>`. Sealed `SpaceBinding<S>` supplies retained starts, budgets and the same World to suite operations. Where the new fixture exercises the suite after enumerating starts for binding, use explicit-start construction rather than calling `Space::new` and enumerating again. Preserve these APIs and the existing state-only shorthand; this task does not reopen the native DSL or suite design.

### Approach

1. Preserve source-only unit APIs, eager fixture generation and exact existing artifacts. Add an explicit bound route using the existing source Type/value representation for ordered World fields and ordered complete initial states. Validate names, counts, types and values against the independently admitted source schema. Expose only the minimal literal-value construction needed by generated glue; convert into the current typed tree rather than introducing another value interpreter. Missing/extra/duplicate/wrong-type World entries, malformed complete states, absent binding and unsupported field types must produce located refusals.
2. Keep original `model.rs` as the independent source-admission input. For selected bound fixtures only, generate an augmented compile-only copy in `OUT_DIR`, preserving the original authored declarations and adding a small projection function inside the inline machine module. Direct typed field access there can read private World and state fields without changing their visibility. Generate scalar conversion or exhaustive enum projection from the admitted schema; rustc must check the concrete field types. Existing `syn`, `quote` and `prettyplease` dependencies suffice. Compile that copy under the original fixture module identity once, rather than compiling original and augmented Models into different types. Admit/lift the original file, never the generated helpers.
3. Extend discovery with a narrow optional per-input generated-path override for the selected fixture's `model.rs` and, where needed, its `tests.rs`. Keep default discovery output, sorted naming, required-input checks, registration and rerun tracking unchanged. `tests.rs` currently compiles as a sibling module, not inside the machine: preserve its original lexical scope when generating selection glue so imported names and private constructor helpers resolve. The selected World expression comes from that fixture's authored hook and is compiled as Rust; neither a source constructor string evaluated elsewhere nor a separate hand-maintained value table establishes the binding. Generated source stays disposable in `OUT_DIR`, outside authored fixture inputs and goldens.
4. For each bound execution, construct one owned selected World, enumerate its authored starts once, and retain both. The generated inner-module projection maps the World and each complete start into the source binding without running handlers or claims. Complete starts may be enumerated natively as R1 requires; native reached states, edges and verdicts must never define emitted Next or claim expressions. Finish binding/emission at runtime when those values exist. Native checking and all TLC counterexample replay in that execution use the same retained World and original start set. Adapt the backend entry point to consume retained starts/exploration, preserving `check_machine` compatibility for unit fixtures and avoiding its current implicit re-exploration/re-enumeration on the bound route. Repeated runner determinism calls are separate executions, not a reason to introduce a global World cache.
5. Introduce distinct lookup identities for World constants, source constants, dynamic state and before-state. Apply the native typed trailing-World classification before interpreting the remaining old positional arguments; argument names do not select semantics. Cover the admitted handler, invariant, postcondition, assumption/`#[scope]` and progress spellings without broadening unrelated source refusal boundaries. Reuse located Diagnostic and collision-safe name generation. An unsupported generic source form stays explicitly refused; native generic support does not authorize a backend substitution. No arbitrary Rust evaluator, reflection registry, new third-party dependency or native DSL trait is introduced.
6. Emit World CONSTANT declarations and the same selected assignments in main, invariant, postcondition, negative and progress CFGs. Emit computed Init as alternatives of complete states, preserving field correlation, duplicate semantics and empty starts. Keep the old field-product path unchanged for unit fixtures. Preserve independent transition/claim translation, disabledness, before/outcome identities and replay. Verify that replay begins in the retained bound starts, without re-calling the authored start function. New action-input semantics remain in .2; no activity source-admission shortcut.
7. Add `parameterized_world` using two selected values of one two-parameter World, private field projection, correlated starts, a world-bounded claim and a deliberately false claim with a replayable path. Reuse discovery, fixture consumers and render_rows if graph size requires it. Execute real binding-sensitive TLC controls for all five CFG families, including negative and progress; emitted CONSTANT text alone is insufficient. Add native report/rows/states/paths and source/TLC artifacts only for new fixtures/controls. Constructor/start counters in focused controls must establish one selection/enumeration per bound execution, and private helper/import controls must establish correct tests.rs scope. Existing unit generation and golden bytes remain unchanged.

### Investigation targets

**Required** (read before coding):

- `model/umpire/lift/src/lib.rs:179` and `src/tree.rs` - original-source admission, typed tree and minimal bound literal handoff.
- `model/umpire/lift/src/shape.rs:91` and `src/body/claim.rs` - World/start admission and World/before parameter lookup identities.
- `model/umpire/lift/src/emit.rs:51` - field-product Init and all five CFG construction paths.
- `model/umpire/lift/src/build.rs:7` - eager unit artifacts, generated typed fixture code and the current sibling replay wrapper.
- `model/umpire/testing/src/discovery.rs:64` and `model/umpire/tests/build.rs:9` - sibling model/tests inclusion, original module identity, generated-path overrides and build rerun tracking.
- `model/umpire/testing/src/backend.rs:59` and `model/umpire/tests/fixtures.rs:10` - current repeated native exploration/starts lookup, consumer selection, required coverage and replay.
- `model/umpire/dsl/src/machine.rs:94` and `model/umpire/suite/src/space.rs:54,131` - committed Associated/Registered contract and explicit-start SpaceBinding ownership; follow native typed World signature handling in `dsl/macros/src/machine/declared.rs` without editing it.

**Optional**:

- `model/umpire/tests/fixtures/sample_kettle/tests.rs` - native rendering, negative and suite consumer patterns; do not copy its repeated World-expression evaluation into the bound route.
- `model/umpire/lift/tests/emission.rs` and `model/umpire/testing/tests/discovery.rs` - empty Init, stable artifacts and compiler-executed discovery controls.

### Verification

Quick from `model/`: `mise exec -- cargo test -p umpire-lift -p umpire-testing -p umpire-tests`, repeated with `UMPIRE_BACKENDS=require` for actual backend controls; scoped strict clippy and formatting. Include compiler-checked private-field projection and tests.rs lexical scope, same-World/one-start-enumeration controls, malformed typed binding refusals, colliding field names and renamed parameters, correlated/duplicate/empty starts, each CFG's actual TLC execution and same-World replay. Compare every pre-existing golden/expansion path against the pinned task baseline, including metadata and unchanged unit generated artifacts. No global model-update or replacement of source-derived transitions with native exploration.

Full from repository root: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`. Preserve timing output, byte comparison, source/type refusal controls, actual TLC logs and same-World replay evidence. If linked guest bytes change, use `make umpire-guest`, retain hash/size and run `make umpire-check-model-slice` plus `make umpire-check-model-gaps` using those exact embedded bytes; the latter includes the private-output logger control. Preserve the activity Model and existing goldens, ABI, panic policy, protocol, dependencies and current owner amendments. Source/TLC completion belongs to this task; the measured R7 DESIGN/readiness closure remains .6.

## Acceptance

- [ ] A compiler-checked per-fixture route projects private fields inside the augmented compiled machine module while independent admission reads untouched authored source. Selection glue preserves tests.rs imports/private helper scope. One owned selected World and one complete-start enumeration feed binding, native checking and replay per bound execution. Missing/extra/duplicate/wrong-type World values, malformed complete states, absent binding and unsupported field types have located refusal controls; old source-only unit APIs, generation and artifacts are unchanged.
- [ ] World/source/state/before identities remain distinct under colliding field names and renamed arguments, including every supported postcondition form and the admitted assumption/scope/progress surface. Source transitions and claims remain independently translated; no native reached graph supplies their meaning, and no native reflection/serialization interface or unrelated source admission is added.
- [ ] Whole-state Init admits correlated pairs without cross-products, preserves duplicate/empty semantics and uses the exact selected World binding. Controls distinguish two values of the same World and deliberately false claim verdicts. Bound replay checks the retained original starts without re-enumeration.
- [ ] Main, invariant, postcondition, negative and progress CFGs carry identical selected assignments and pass real binding-sensitive TLC controls with same-World counterexample replay. Existing source refusal boundaries, fixture determinism, module identity and required-backend coverage checks remain intact.
- [ ] The new two-parameter `parameterized_world` fixture has native report/rows/states/paths plus source/TLC artifacts and actual agreement. All old goldens/expansions are byte-identical; Quick/full gates pass with timing output retained and any necessary rebuilt-guest live slice/gap/privacy proof. Together with native predecessor this discharges the World portion of R1/R3; input and activity portions remain open under their existing tasks and owner amendments.

## Done summary
Blocked:
BLOCKED: EXTERNAL_BLOCKED
Task: fn-227-parameterized-models-declared-worlds.7
Summary: external_source_concurrency prevents attributable final verification; the owner requested wrapping the in-flight checkpoint and stopping.
Impact: The World source/TLC checkpoint is preserved, but final full verification, implementation review and task closure remain open. No further task is admitted.
Suggested resolution: Coordinate a stable guest recipe/compiler-helper/embedded-asset ownership boundary, re-anchor this checkpoint and any intervening foreign commits, run fresh scoped required-backend and all three full gates, obtain the required independent implementation-review verdict, then complete the task normally. Never reuse the mixed-input gate as a green receipt.

The unowned, uncommitted guest recipe, compiler helper/tests and embedded asset differ from the pinned pre-edit baseline. They were detected while the final full gate was active; no isolated stable owned input manifest supports attributing that gate to Task 7. This is an evidence/attribution gap, not a claimed test failure or a claim about exactly when an individual stage read a changed file.

Default and required-backend scoped Quick commands passed, including actual binding-sensitive TLC executions of all five configuration families and same-World false-claim replay. Scoped strict clippy and formatting passed. Both fresh static quality audits returned zero findings and are retained verbatim. The task has no formal implementation-review verdict and is not complete.
## Evidence
- Commits:
- Tests:
- PRs:
