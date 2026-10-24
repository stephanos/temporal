---
satisfies: [R1, R2, R7]
---
# fn-195-modelsuite-a-models-test-file-holds-its.8 DSL completion: ModelSuite lifecycle, discovery and suite execution policy

## Description
Expose `framework.dsl.ModelSuite(instance)` as the four-section MUnit facade over P's proved declaration/request boundary. Register authored path tests in `examples` and `conformance`, ordinary MUnit tests in `unit`, and the `explorations` container whose actual search forms arrive in X. Keep top-level shared helper definitions, exact instance binding, stable instance diagnostics, and one common admission path for every String and TestOptions registration.

Adopt the completed fn-195.1 post-construction registration seam. Initializing subclass sections from the superclass constructor is unsafe. Register at first `munitTests` enumeration, after eager concrete helper initialization and before the inherited MUnit test-list snapshot. Unrelated lazy helpers need not have initialized; preserve their ordinary lazy behavior. Preserve the prototype's section order and lifecycle controls while supplying production source locations and real Registry discovery. This task does not implement a new test runner.

Run the bound instance's claims automatically through the existing checker and preserve their native report, vacuity warnings, budget, completion and evidence. Ordinary claims must not disappear merely because a suite has only authored examples or only unit tests. A refuted claim remains a visible failure. Unknown/cut results remain explicitly inconclusive. Choose a bounded, tested and documented MUnit failure/skip disposition when implementing this checkpoint; a silently green skip or verified label does not satisfy it.

**Size:** M, bounded to facade registration/discovery and one suite execution adapter. P owns all body rewriting and strict Boolean/context mechanics. X owns exploration grammar and wrappers. C owns typed conformance metadata, realization validation, capability paths and future identity ingredients. If these deferred mechanisms are required to make registration itself correct, report the concrete dependency rather than importing their implementation into S.

**Files:** `model/framework/dsl/ModelSuite.scala`, `Sections.scala` and focused registration/lifecycle `.test.scala` controls; `model/framework/dsl/project.scala` for the existing MUnit version's ordinary dependency; the finite-request extension of `model/framework/dsl/NativeRuntime.scala` and its sole implementation `model/framework/checker/Native.scala`; one `model/framework/checker/SuiteExecution.scala` adapter with focused tests; `model/framework/check/Registry.scala`, new `RegistryDiscovery.test.scala` and existing `BudgetGate.test.scala` for generic discovery/budget controls. Reuse `model/framework/checker/Execution.scala`, `BudgetPolicy.scala` and `model/framework/check/Tools.scala` unless a demonstrated integration gap requires a narrow edit. The existing disposable FrameworkLoop Make dry-run fixture remains admitted; actual repository Make execution stays held.

**Touches:** [model/framework/dsl/ModelSuite.scala, model/framework/dsl/ModelSuite.test.scala, model/framework/dsl/Sections.scala, model/framework/dsl/Sections.test.scala, model/framework/dsl/project.scala, model/framework/dsl/NativeRuntime.scala, model/framework/checker/Native.scala, model/framework/checker/ClaimEvaluation.scala, model/framework/checker/SuiteExecution.scala, model/framework/checker/SuiteExecution.test.scala, model/framework/check/Registry.scala, model/framework/check/RegistryDiscovery.test.scala, model/framework/check/BudgetGate.test.scala, model/framework/REFERENCE.md]

**Dependencies:** Consume the committed existing `.2` P checkpoint and fn-195.1's recorded lifecycle findings. This new task depends on `.2` using Flow's same-spec dependency mechanism. X consumes S's section and immutable execution boundary; C follows the declared P/S/X carrier dependencies. Existing .3 must wait until the complete replacement surface, including C, is committed. Preserve all existing task IDs and the `.4 -> .6 -> .5` sequence.

### Investigation targets

- `model/design/modelsuite-prototype/ModelSuite.scala` for the measured post-construction registration/admission mechanism.
- `model/framework/checker/Native.scala` for the sole NativeRuntime implementation forwarding suite requests to the one adapter.
- `model/framework/dsl/NativeRuntime.scala` for DSL-owned finite request dispatch without a reverse checker import.
- `model/framework/checker/Execution.scala` for explicit immutable caller-budget passing across native questions.
- `model/framework/checker/BudgetPolicy.scala` for parsing, normalization, exact suite names and unknown-override validation.
- `model/framework/check/Registry.scala` for actual source suite discovery and its current direct-FunSuite limitation.
- `model/framework/check/Tools.scala` for existing gate policy validation and normalized environment forwarding.

### Import and execution boundary

Keep the accepted public facade in `framework.dsl`; do not relocate it to `framework.checker` or require a surprising new author import. Promote the existing pinned MUnit dependency from test-only to ordinary compile dependency for this facade. The DSL/realize module may hold declaration references and finite execution requests, but imports no checker implementation, export/lifter, gate or system-specific module.

Use the existing lazy `NativeRuntime` boundary and one checker-owned `SuiteExecution` adapter. The adapter alone resolves the existing `BudgetPolicy`/`Execution` and delegates to `verifyAll` and replay. All future X/C operations must have an explicit route through that same immutable execution selection; this task prepares the request boundary without implementing their grammar. Do not expose checker `Result` classes or search machinery as a reverse DSL dependency. Retain native evidence through the DSL-owned execution/report boundary and MUnit diagnostics.

The framework gate already validates and normalizes caller policy in `Tools.withCheckerBudgets`. Keep that contract and repair the actual generic Registry discovery needed to supply its suite names. The checker adapter resolves the exact suite's immutable execution from the supplied normalized policy. Model construction and DSL-only class loading do not parse environment policy, execute claims or load the checker. Runtime execution may require the checker and must fail explicitly if it is absent.


### Confirmed discovery prerequisite

The clean committed implementation checkpoint `ff0dcaf55d460f887bf5fb56d7e31b4ecd92d7f1` passes all 20 admitted native stages. It remains incomplete. The retained `.flow/tmp/fn1958/metadata-probe/` proves two valid public no-argument suites extending stable-term superclass paths run four JUnit tests while actual Registry finds zero. This is a compiler-metadata dependency, not an authoring restriction or a reason to keep adding source parser cases.

Wait for `.12`, the `.13` in-process metadata producer and the explicitly blocked `.14` complete gate integration. The `.12` source checkpoint `41454f742a` proves bounded classfile/typed joins and source-less helper acquisition, while preserving a partial-ancestor RED. It establishes neither complete held-export identities nor actual staged gate callers. Malformed/empty/duplicate/nonpositive/overflowing policy must still refuse before child commands. The proposed exact unknown-name check follows admitted metadata compilation and precedes any suite construction, setup, claims or test execution. The conductor must amend the stronger fn-196.2 ordering explicitly after complete real-caller proof; no current source or historical receipt is reclassified by this planning change. Held/unselected suite overrides must remain valid without compiling owner-held inputs. A generic-only inventory is insufficient. `.14` must receive a scope-admitted complete metadata route and honest bounded decomposition before implementation.

Before resuming this task, integrate the demonstrated production prerequisites and rerun actual Registry and ordinary public source/library execution against the retained stable-term suites, plus their eligibility/provenance neighbors. Keep `.3` responsible for final complete typed section admission and the no-test corpus check. Keep `.10` responsible for its independent exact-instance realization/wait/capability proof. No task completion is earned by this partial source checkpoint.

### Narrow native prerequisite: constant claim ownership

Automatic claims exposed a compiler-erased outer-receiver case: a constant claim section may have a zero-argument constructor even though its exact declaring Machine provides the section getter. Preserve the existing exact declared getter/return-type and Machine ownership checks, while admitting the compiler's zero-argument section constructor only for a section lexically declared by that exact Machine, alongside the existing exact outer-argument constructor. A Machine getter aliasing a stateless foreign facet section must still refuse. Use the corresponding public zero-argument constructor when reloading that already-admitted section without a MODULE$ field; preserve the existing exact-outer constructor path and owner-required refusal otherwise. Add a dedicated multiline constant-claim regression through the existing suite adapter, retaining verified and refuted native evidence and the exact model owner. Retain the predicate-only RED that exposed the reload prerequisite and the stateless foreign-alias RED that exposed the missing lexical-owner guard. Keep foreign-instance/facet refusals and the existing ambiguous same-line initializer refusal unchanged. This adds only `ClaimEvaluation.scala` to the implementation surface; no NativeClaimSections change or new public grammar is admitted. Preserve the original public-runner failure in the task evidence.

### Public reference coverage

Maintain the existing ReferenceSurface gate as this public API lands. Add minimum accurate user-facing coverage of ModelSuite registration, section forms, execution policy and any new public signature carriers in `model/framework/REFERENCE.md`. Keep that reference free of Scala-file references and distinguish native replay registration from the still-pending exploration/conformance/Case checkpoints. This is API coverage required by the existing gate, not the comprehensive .5 rewrite; do not weaken or bypass the gate.

### Verification

Use the then-current README Native checks recipe and unchanged strict flags. Focus first on real MUnit registration/lifecycle, generic Registry discovery, the checker suite adapter and existing budget gate suites. Execute at least one public ModelSuite through ordinary source-test discovery and one against freshly packaged declaration/checker libraries in a fresh workspace. Verify nonzero test counts and exact selected suite names; passing only the focused adapter's direct checker tests does not establish automatic policy resolution.

Keep source/lifecycle negative controls for placement, invalid members, loading-state re-entry, early section access, sticky failure and both registration overloads. Distinguish the established loading-state test-transform control from unproved ready-state suite-transform recursion. Verify eager concrete helpers are initialized and unrelated lazy helpers are not gratuitously forced. Use synthetic native instances with known verified, refuted and cut/unknown claims so the selected MUnit disposition and retained evidence are observable. Re-run P's positioned expectation control through the real ModelSuite receiver. Confirm public declaration compile/load without checker initialization and the explicit checker-required execution failure.

Record commands, exits, selected suite names, counts, policy values and native evidence. Retain the full final-lint obligation in .3; when this task changes a source lint, its focused admitted/refused controls and nonempty native corpus check are required here.

### Coverage and held work

S completes R1's public registration/placement, integrates R2 replay into MUnit and supplies local R7 lifecycle/placement/slug controls. Automatic claims follow the parent's five-kind table and existing fn-196 `verifyAll`/caller-budget contract. Existing .3 owns final corpus lint.

X retains R3's exploration API, and C retains R4's typed metadata, realization/capability binding, eligibility and native identities. Conformance authored bodies here earn native replay evidence only. Existing retirement/migration/class-only/docs/final-polish ordering remains.

Parent holds remain unchanged. Keep mixed parents OPEN, review deferred and all Phase 3 proof obligations visible.

## Acceptance
- [ ] ModelSuite registers `examples`, `explorations`, `conformance` and `unit` after complete subclass construction in that order. Ordinary `unit` bodies retain MUnit semantics; P's authored bodies execute against the exact bound instance. Absent, empty and unit-only sections work. The suite top level allows sections and shared helpers. Misplaced tests and invalid non-test section members refuse with a member/test location; the future admitted exploration space val remains a supported member kind without accepting arbitrary section data.
- [ ] Registration is admitted once per instance and preserves the fn-195.1 lifecycle behavior. Repeated/concurrent enumeration returns the complete registered list; re-entry while loading fails explicitly; partial initializer failure remains sticky. Cross-section early registration and an eager helper touching an unregistered section refuse. Tests and transforms use the actual suite receiver after eager helpers initialize. String and TestOptions registrations share placement and slug checks, including mixed-overload and cross-section collisions.
- [ ] Ordinary MUnit discovery/description, test transforms, repeated suite transforms, filtering, ignore/only and setup ordering remain intact. Registration completes before the inherited test snapshot and before setup; filtering can execute an exact unit receiver without running filtered bodies. Loading-state registration re-entry, including the prototype's test-transform recursion, remains visible through the ordinary runner. Preserve MUnit's ready-state suite-transform behavior without claiming the prototype proved ready-state transform-recursion handling. Do not claim forced scheduler contention from the prototype's two-submission control; add a bounded overlap control only if needed for the production synchronization claim.
- [ ] Actual Registry discovery recognizes supported ModelSuite subclasses and reports their exact fully qualified suite names while preserving existing direct FunSuite handling and testdata/build exclusions. Abstract/helper/non-suite classes are not accidentally registered. A real selected ModelSuite run executes a nonzero test count without a prototype-only explicit selector. DSL/realize-only declaration compile/load remains valid with ordinary MUnit and without the checker; execution without the checker fails through the existing explicit runtime boundary.
- [ ] The existing caller policy resolves once per actual suite execution and supplies one immutable `Execution` for automatic claims and P replay, with an extension seam for X/C. Real suite controls demonstrate default and exact-suite override effects on reported native budgets. Existing malformed/empty policy, nonpositive/out-of-Int values, duplicate keys and unknown-suite overrides refuse at the correct gate/suite boundary. Automatic claims retain ordinary native results, counterexample paths and vacuity warnings for the bound instance; refuted results fail, and unknown/cut results retain a visible documented inconclusive MUnit disposition. A horizon, sample count or Model field cannot replace the caller budget, and a discovered counterexample survives an unrelated cut.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: