---
satisfies: [R3, R7]
---
# fn-195-modelsuite-a-models-test-file-holds-its.9 DSL completion: reusable exploration spaces and native question adapters

## Description
Authors declare a reusable space in `explorations`, ask a horizon-bounded find with positioned `expect`, replay seeded samples judged by the Model's claims, and run a derived negative control through `refuted(machine, claim)`. Translate these declarations into the shipped checker operations. Keep the immutable suite execution policy and exact receiver established by S.

This checkpoint adds public declaration capture and finite adaptation. `checker/Spaces.scala` already owns the schedule selector and bounded find; `checker/Sampling.scala` already owns seeded walks and sampled safety/monitor/progress checks. `checker/Execution.scala` already passes an immutable execution budget. No exploration engine, second policy parser or new execution backend belongs here.

**Size:** M, provided P has delivered the deferred body/expect carriers and S has delivered registration, exact naming and the suite execution bridge. If those inputs are absent, retain the dependency rather than recreating them here.

### Dependencies and consumed contracts

- P, authored capture/replay, integrated and closed. Consume its deferred action/alternative carriers, positioned Boolean expectations, exact instance types and body admission.
- S, suite lifecycle/discovery/policy, integrated and closed. Consume its four-section receiver, immutable exact-suite execution budget, native result-to-MUnit boundary and declaration identity ingredients.
- The existing native checker checkpoint remains admitted. Do not add whole-parent dependencies on open fn-194/fn-196 or future fn-198.

Conformance C consumes the reusable sample descriptor and selected seeded paths delivered here. `.3` owns deletion of `assertRefuted`, `NativeControls`, old query/exploration authoring and final corpus lint. This task exposes the replacement and keeps that retirement dependency explicit.

### Files and Touches

**Files:** `model/framework/dsl/Spaces.scala`, `model/framework/dsl/ModelSuite.scala`, `model/framework/dsl/Sections.scala`, `model/framework/dsl/ModelSuiteMacros.scala`, `model/framework/dsl/NativeRuntime.scala`, `model/framework/checker/Native.scala`, `model/framework/checker/SuiteExecution.scala`.

**Touches:** [model/framework/dsl/Spaces.scala, model/framework/dsl/ModelSuite.scala, model/framework/dsl/Sections.scala, model/framework/dsl/ModelSuiteMacros.scala, model/framework/dsl/NativeRuntime.scala, model/framework/checker/Native.scala, model/framework/checker/SuiteExecution.scala, model/framework/dsl/Spaces.test.scala, model/framework/checker/SuiteExecution.test.scala, model/framework/dsl/testdata/modelsuite/ExplorationClient.scala.txt, model/framework/dsl/testdata/modelsuite/ExplorationRefusals.scala.txt, model/framework/dsl/ClaimStatements.scala, model/framework/dsl/Syntax.scala, model/framework/checker/Checker.scala, model/framework/checker/Execution.scala]

Paths are relative to the repository root. Reuse P/S files where they already exist.

- `model/framework/dsl/Spaces.scala` (new if P/S did not create its declaration shell). Own immutable exact-instance space, positioned selectors and find/sample requests.
- `model/framework/dsl/ModelSuite.scala`, `Sections.scala`, `ModelSuiteMacros.scala`. Add exploration-context operations and capture per-input `any`, whole-action selection, positioned target and deferred claim references.
- `model/framework/dsl/NativeRuntime.scala` and `model/framework/checker/Native.scala`. Extend only the suite operations needed to pass these declarations to the checker.
- `model/framework/checker/SuiteExecution.scala`. Translate to existing `Space`, `Expectation` and execution methods and retain their ordinary result evidence.
- `model/framework/dsl/Spaces.test.scala`, `model/framework/checker/SuiteExecution.test.scala`, and focused public source/library fixtures under `model/framework/dsl/testdata/modelsuite/` (new fixtures as needed).
- `model/framework/dsl/ClaimStatements.scala` or `Syntax.scala` only if P's typed claim-reference carrier does not already support negative controls. Retain normal claim-context refusals and avoid a second public query API.
- A narrowly selected `model/framework/checker/Checker.scala` or `Execution.scala` change is allowed only if necessary to consume the already captured private claim reference without the public `verify` builder. Prefer the existing private held/refuted route; `.3` still owns complete old-word retirement.

The DSL must not import checker implementation classes or return checker-owned `Space`, `Expectation` or `Result` from its public signatures.

### Investigation targets

- Required `model/framework/checker/Spaces.scala`. Reuse positioned selectors, target context and horizon/result semantics.
- Required `model/framework/checker/Sampling.scala`. Preserve seeded paths, sampled claim checks and failure-index evidence.
- Required `model/framework/checker/Execution.scala`. Preserve explicit budget passing and identify the private replacement for public verify capture.
- Required `model/framework/dsl/NativeRuntime.scala`. Extend the existing declaration-to-runtime seam.
- Required `model/framework/dsl/ClaimStatements.scala`. Reuse deferred typed native claim references without eager Unit evaluation.
- Optional `model/framework/dsl/Syntax.scala`. Inspect only if P's capture seam cannot already resolve negative-control claim ownership.

### Implementation boundary

Capture `explore(steps = n): ...` as a reusable declaration scoped to its exact Model. A whole action selects its admitted input classes. An input-level `any` ranges only that input's existing finite domain while retaining the other named fields and their types. Preserve selector position and deterministic native class ordering. Reuse P's single-evaluation handling of ordinary named input expressions.

An exploration test can request `space.find(steps = horizon): expect: ...`, `space.sample(count, seed)` or `refuted(derivedMachine, claim)`. Find targets use P's positioned assertions and step context. A group of target assertions must retain every condition; neither taking only the last Boolean nor flattening away its location is valid. Use the existing native target semantics and report found/notFound/unknown without treating absence as success.

Sampling preserves seed, requested count, walk horizon, retained paths and failing sample index. Judge walks with the existing Model claims, including native monitor/progress behavior already supplied by Sampling. Preserve finite-experiment evidence and cuts. Reusing a space from conformance for sampling is an input C can consume, while direct `any` remains confined to exploration declarations.

Capture negative-control claims by owner/section/member or the existing typed native claim reference. Collection must not execute a Unit-valued claim body. Return the ordinary checker refutation result to the suite adapter; MUnit expects a refutation while the retained result continues to say `refuted`. A verified or unknown ordinary result must not masquerade as a successful negative control.


### Verification

At implementation time, first run focused existing checker space/sampling/negative-control controls and new X suite selectors through the current native README roots. Record exact commands, exits and nonzero executed counts. Include one real public ModelSuite selected through S's actual Registry route so direct checker controls cannot substitute for integration.

Run both a fresh framework-main plus authored `*.test.scala` route and an outside-package client against a separately compiled framework/macro jar. Preserve `dsl/project.scala:8` strict flags and fresh Scala CLI workspaces. The final warning receipt at `.flow/tmp/fn1952-plan/expect-probe.md` establishes public reflection-based positioning for those boundaries; it adds no compiler dependency. Carry P's unrelated-warning and assertion-retention controls when X changes its target capture.

Prove DSL/realize declarations compile without checker implementation classes and do not evaluate callbacks. Add checker only for native execution. Assert exact intended diagnostics for negative clients as well as exit status. Preserve state restoration, result budget and source positions in executed controls.

Use the already admitted native verifier stages and focused X selectors; keep real repository Make, Go, Temporal, export/lifting/IR/schema/backend/generated/regeneration routes held. Existing disposable dry-run Make boundary fixtures remain admitted under their original scope. No full Make-green or Case-production claim follows from this task.

### Requirement handoff

X implements R3's new author surface and local R7 refusals. `.3` retains old-control retirement and corpus admission after C.

## Acceptance
- [ ] Reusable spaces support whole-action and per-input `any` with exact instance ownership, bounded positions and named-field constraints. Two independent instances cannot exchange spaces, starts or selected classes.
- [ ] Find honors the named horizon and every positioned Boolean target. It retains witness evidence for found, completes the requested horizon before notFound, and preserves unknown/counterexample precedence on a budget cut.
- [ ] Seeded samples repeat their selected paths for identical declarations/seed, preserve sample metadata and fail the MUnit test on a claim violation. A finite sample never earns an exhaustive claim.
- [ ] Refuted captures the named derived claim without eager evaluation and retains the ordinary native result and concrete counterexample. A non-refuted neighbor fails the negative-control test.
- [ ] All operations receive S's same resolved suite budget. Horizon, count and seed remain independent of it. Repeated use of one space does not mutate it or leak a prior question's state/context.
- [ ] Local admitted/refused controls cover `any` in examples/conformance, find/sample in examples, a missing target, horizon outside `1..space.steps`, nonpositive space size, count zero/below one and foreign owner values. Locate each public refusal at the authored operation. `.3` later checks the complete source corpus with the final suite pass.
- [ ] A conformance consumer can retain the exact sample declaration and deterministic seeded paths without executing Case generation or reading a server Run.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
