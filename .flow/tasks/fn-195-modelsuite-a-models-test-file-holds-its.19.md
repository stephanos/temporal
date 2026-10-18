---
satisfies: [R3, R7]
---
# fn-195-modelsuite-a-models-test-file-holds-its.19 Implement immutable native exploration spaces and selectors

## Description
Implement R3's reusable native space declaration and its local R7 refusals. Consume committed .2 and the lifecycle/native runtime checkpoint `ff0dcaf55d460f887bf5fb56d7e31b4ecd92d7f1`; full .8 is still incomplete. This leaf supplies declarations to the question adapter leaf. Original .9 retains actual Registry integration.

**Size:** M, immutable selector capture adapting the existing typed action and run-position machinery, with four acceptance groups.
**Files:** new `model/framework/dsl/Spaces.scala`, `model/framework/dsl/ModelSuite.scala`, new `model/framework/dsl/Spaces.test.scala`, `model/framework/REFERENCE.md`; conditional `model/framework/dsl/InputSource.scala` only for typed per-input expansion.
**Touches:** [model/framework/dsl/Spaces.scala, model/framework/dsl/ModelSuite.scala, model/framework/dsl/Spaces.test.scala, model/framework/dsl/InputSource.scala, model/framework/REFERENCE.md, .flow/tmp/fn195-native-spaces/**]

### Approach
- Add the suite's `explore(steps = n)` declaration factory and DSL-owned immutable exact-instance space, using the existing `ExplorationSpace` marker admitted by `Sections.scala:146`. Keep question registration for the next leaf. `ModelSuiteMacros` lives inside `ModelSuite.scala:165`; no standalone file exists.
- Reuse `RunBody.scala:5` positions and `RunBodyMacros.roots` for exact owner checks. Its target helper rejects input-bearing whole actions at lines 104-109; give exploration capture an explicit separate admission path without weakening authored-run rules. Reuse `InputSource` finite domain metadata for per-input `any`, retaining named constraints, typed field identity and deterministic class ordering. Do not enumerate a replacement domain.
- Retain selectors and ordinary named expressions without executing actions, guards, effects or claims during capture. Evaluate each supplied constraint once at the deferred space acquisition boundary, in authored order, retain that immutable value across reuse and locate failures at its authored expression. Existing position-select callbacks do not restore reached state; do not claim per-reached-state reevaluation or add it accidentally. Test acquisition timing and unchanged surrounding Model state explicitly.
- Bound declared selector positions by space.steps and preserve existing search semantics for unspecified positions. Retain test-level `from`/single-start resolution for the later question adapter; do not invent an explore-from API. Foreign starts/classes/space receivers refuse, and a question missing `from` on a multi-start Model retains the parent's refusal.
- Define DSL-owned positioned find-target and seeded sample-request carriers/signatures here for the next adapter to consume. Retain every target condition and sample horizon/count/seed without installing execution early or importing checker types.
- Add minimum accurate user-facing REFERENCE coverage for the shipped declaration only, without Scala-file references or claiming installed questions/conformance. Run the existing ReferenceSurface gate. Final .5 still owns complete guidance.

### Investigation targets
**Required:**
- `model/framework/dsl/RunBody.scala:76` - capture validation, stable roots, positions and deferred targets.
- `model/framework/dsl/InputSource.scala` - existing finite typed input domains.
- `model/framework/dsl/ModelSuite.scala:165` - actual inline macro receiver and section context.
- `model/framework/dsl/Sections.scala:146` - immutable exploration-space member admission.
- `model/framework/checker/Spaces.scala:7` - exact starts and finite position selector contract.
**Optional:**
- `model/framework/check/lint/ReferenceSurface.test.scala` - existing public reference checks.

### Verification and handoff
Use focused native selectors and fresh framework-main plus authored test compilation, then a separately packaged framework/macro jar and outside-package client under unchanged `dsl/project.scala` strict flags. Allocate a fresh Scala CLI workspace for each complete recipe; same-path jar replacement is insufficient. Preserve commands, terminal exits, nonzero counts, original sources and artifact hashes under this task's scratch root. Prove declaration compile/load without checker initialization, zero capture callbacks and located intended refusals. The next leaf must consume actual retained declarations, not hand-built selector substitutes.

No Registry/gate caller, policy parsing, realization selection, capability production or Case eligibility changes belong here. Keep .8/.14/.11 and fn-196.2's ordering contract open. Temporal reads, actual repository Make, Go, export/lifting/IR/schema/backend/generated/regeneration and server/recordedRun/Case operations remain held. Export compile-only approval remains ungranted. Formal review stays phase-batch deferred.

## Acceptance
- [ ] A public reusable `explore` declaration captures whole actions with inputs and independently varying per-input `any`, preserving remaining named typed constraints, bounded selector positions, unspecified-position search, domain ordering/correlations and positive step bound. An admitted fixed-input neighbor pins unchanged ordinary authored-run behavior. Find-target/sample-request carriers preserve every condition and requested bound for the next adapter.
- [ ] Fresh exact Model instances retain separate spaces, starts and classes; aliases on the same exact instance remain valid. Foreign owners/starts/classes and nonpositive steps refuse at their authored locations. Repeated acquisition/use cannot mutate an earlier descriptor or leak Model state.
- [ ] Constraint expressions evaluate once at deferred space acquisition and remain immutable across questions; declaration capture/load executes no effect/guard/claim callback. Direct `any` outside exploration capture, including examples and conformance, refuses at the authored expression with an admitted exploration neighbor.
- [ ] Focused native, fresh source and separate-library controls execute nonzero counts under unchanged strict flags; declaration-only loading works without checker execution. REFERENCE and ReferenceSurface agree on the shipped subset, and the receipt assigns question execution to the next leaf and actual Registry integration to original .9.


## Done summary
# fn-195.19 native space checkpoint

ModelSuite authors can retain exact-instance `explore` spaces with ordered whole-action, fixed-class and typed per-input wildcard selectors. Deferred acquisition retains its first success or failure, evaluates named constraints once in authored order, and supplies immutable native selectors plus positioned find/sample declaration carriers.

Current source checkpoint: `254116aba7044bee7be792b791bb8979ffa0f763`. The original audited checkpoint remains `650a5044a4499fa6a5eb734865727f47ca59d4b6`. Baseline: green, 4 focused CheckerTest controls before edits at `41564bbc20a22a3fa2b3bf9823adeb25442f287d`.

Tier: session (jev-unavailable(no_key)); role pinned to gpt-6.1-sol medium by AGENTS

stage: impl-review - skipped(config: REVIEW_MODE=none; policy: owner phase-batch formal review deferral)
stage: plan-sync - skipped(config: planSync=false)

### Acceptance evidence

- Fresh `review-source-v2` and `review-client-v2` each execute 7 SpacesTest controls and 1 outside-package public declaration control. Their exact commands, copied sources, status 0, logs and tool versions live under `.flow/tmp/fn195-native-spaces/`. The native client compiles only authored client files against the packaged jar and frozen IR dependency. Its public control verifies the Explore class's actual jar origin. The original `final-source` and `final-client` checkpoints each executed the prior 6 native controls and 1 public control.
- The 6 native controls cover whole input actions, independently varying named inputs, scalar input order, whole-record correlations/order, fixed classes, search through an unspecified second position, positive/bounded selector positions, defaulted/required fields, exact roots/actions/classes/starts/receivers, same-instance aliases, multiple-start `from`, nested state restoration, zero declaration/acquisition callbacks, once-only ordered constraints, all retained target conditions and sample horizon/count/seed, and immutable repeated success/failure acquisition. The ordinary fixed-input authored-run neighbor remains verified. The compiletime refusal table has 8 entries.
- `declarations-source-final` and `declarations-client-final` each execute 1 outside-package public loading/registration control with checker classes absent. The carrier-only helper definitions live outside the exploration section. Construction and registration execute zero guard/effect/claim/constraint/target callbacks. `declarations-library-final` packages the DSL/realization declaration library without checker sources.
- `ingredient-controls-final` executes 21 existing RunBodyTest and ModelSuiteTest controls. `reference-final` executes all 4 original ReferenceSurface checks unchanged. REFERENCE documents the installed declaration subset without Scala-file references.
- Source and library each independently produce 5 outer refusal diagnostics and 2 nested typed-field refusal diagnostics. Each compiler exits 1 and each dedicated validator exits 0. Versioned `*-outer-refusals-v2` and `*-field-refusals-v2` preserve the authored locations. Source and library each additionally refuse 2 strict-warning controls under unchanged flags. Count these 14 direct compile refusals and 4 warning controls separately from executed ingredient tests.
- Canonical Quick runs exactly `mise exec -- scala-cli test --server=false model/framework/checker --require-tests --test-only checkercontrol.CheckerTest`, exits 0 and executes 4 tests. Its log is `quick-final/output.log`. The receipt is `.flow/tmp/green-receipts/650a5044-unittest.json`. Repeated formatting with `model/.scalafmt.conf` leaves identical bytes in the before/after manifests. `git diff --check` passes.

The canonical Quick observation belongs to checkpoint 650a. A subsequent concurrent workspace change removed `model/build/ir-scalapb.jar`. The worker preserved that removal and restored no repository dependency. The final source/client and `review-quick-v2` recipes use the original checksum-verified immutable dependency copy only within fresh scratch roots. `review-quick-v2` executes the same 4 CheckerTest controls and exits 0 under the unchanged strict flags. It is a scratch Quick observation with a different command, rather than a new canonical receipt. Repeated final test formatting also leaves identical bytes.

### Reproduction and failed preparation

`red-v1` records missing explore/any APIs with downstream assertion typing errors. `impl-v3` compiles and executes the original wildcard test, then fails because any reaches runtime. `missing-field-red-v2` executes 4 passing controls and the intended missing-required-field failure. `repeat-failure-red` executes the intended evaluations=2 versus expected=1 failure before first-failure retention. Earlier constructor/annotation/visibility compilation failures and the over-specific diagnostic assertion remain raw failed attempts, rather than green evidence.

The first aggregate `source-refusals-final` invocation emits only 5 of the intended 7 diagnostics because outer macro errors prevent the nested InputSource expansions. Its validator and enclosing `verify.sh` correctly exit 1. The separate v2 groups establish all 7 intended diagnostics per route. `finish-verify.sh` completes with exit 0. No failed log or source snapshot was overwritten.

### Immutable bindings and history

`final-source-v2.sha256` binds the current five task-owned source/test/reference paths. `final-artifacts-v2.sha256` binds all current versioned recipes, original copied sources, logs, commands, statuses and artifacts. The original `final-source.sha256` and `final-artifacts.sha256` remain unchanged for the 650a checkpoint. The original artifact manifest still verifies; its source manifest records the earlier test bytes. Every current native main source matches the native library's retained source copy byte for byte. The final formatted authored tests match both fresh review-v2 source/client snapshots.

- Native library SHA256: `49522e56986755fbb74a20e30899c76ade5f3a7a9eebe9c8ace12cbd37992ae1`.
- Declaration library SHA256: `0f3494cf5d4107f73946c09571d63cc5713d059135dfee0088b46936d59cbb32`.
- Frozen read-only IR dependency SHA256: `b9e0538afffad2b33b9fc8fc9eb2a3f4e63b10928fa38506b585747535e85c92`.

The concurrent user checkpoint `bc31408eff79383a780516deea49402c15f7c91c` committed the task's five paths alongside 77 user-owned design-file renames. This worker issued no commit command for that checkpoint and preserved it unchanged. The worker's conventional checkpoint `650a5044a4` adds the final unspecified-position native-search control. The evidence range starts at the original pre-edit baseline and therefore includes the user checkpoint; task ownership and audit scope remain limited to the five declared paths.

Checkpoint `254116aba7` adds the receiver-context regression after the original quality audit. Concurrent untracked fn-199 and later specs belong to the user. The conductor explicitly authorized task-scoped `git commit --only` instead of catch-all staging to preserve those files and the user's index. The worker checked the index first and committed only Spaces.test.scala. Closure staging follows the same exception while unrelated user files remain.

### Quality finding evaluation

The two original read-only reports remain verbatim in `quality-reports.md`. Standards reports zero findings. Correctness raised one source-walk hypothesis that find might acquire its receiver twice. The initial public packaged probe passes with one acquisition. The tracked regression then tests both an effectful same-owner receiver and an alternating-owner receiver, asserts one acquisition, and evaluates the actual retained target assertion to verify its captured RunContext carries the first exact Model. Both source and separate-library routes pass. Scala stabilizes the inline receiver prefix. The hypothesis is refuted and no speculative production patch was retained. Formal implementation review remains deferred.

The correctness reviewer retracts the complete finding after inspecting the executed controls. The standards follow-up reports zero findings on the added regression. All four reports and the conductor's bounded acceptance remain in `quality-reports.md`. The conductor accepts .19 with zero outstanding findings; this supplies no formal phase-batch verdict or broader integration credit.

### Boundaries and process ownership

Question registration/execution belongs to .20, refutation/final refusals to .21 and actual Registry-selected integration to original .9. .8/.14/.11 remain incomplete. Native private acquisition/engine tests and public declaration loading are separate proof families. No public acquisition escape hatch was added.

The cumulative classifier returns exit 1, FULL, for the range including the user's design renames. The focused Quick and task-native gates above ran. The broad Make/changed-scope integration gate remains owner-held, with no skipped/full-gate claim fabricated. No Make, export/plugin/reify, lifting/IR/schema generation, backend, Go, Temporal, server, Testpilot, recordedRun or Case operation ran. The export compile-only approval remains ungranted.

The later `classify-v2.log` also returns exit 1, FULL, naming the concurrently edited user-owned MILESTONES.md. These cumulative classifier observations include unrelated user work. Neither observation authorizes a held broad gate or counts as a skipped test.

Every started compiler/JVM command was joined. The final serialized wrapper handle `19693` terminated with exit 0; earlier wrapper `52399` terminated with exit 1 at its refusal validator. `processes.md` records all tool session exits. No task compiler lane remains. Historical Bloop PID 93084 was left untouched.

Supplemental receiver probe handle `51767` exits 0. The first follow-up source preparation exits 1 before starting a compiler because the repository dependency disappeared; its partial directory and preparation-failure note are retained. The replacement serialized review wrapper handle `33064` exits 0 after source/client/Quick verification using the frozen scratch dependency. Every supplemental command is joined.
## Evidence
- Commits: bc31408eff79383a780516deea49402c15f7c91c, 650a5044a4499fa6a5eb734865727f47ca59d4b6, 254116aba7044bee7be792b791bb8979ffa0f763
- Tests: env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-spaces/finish-verify.sh, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-spaces/review-verify-v2.sh, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-spaces/run.sh declarations-source-final declarations-source, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-spaces/run.sh declarations-library-final declarations-library, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-spaces/run.sh declarations-client-final declarations-client /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-spaces/declarations-library-final/declarations.jar, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-spaces/run.sh ingredient-controls-final controls, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-spaces/run.sh reference-final reference, mise exec -- scala-cli test --server=false model/framework/checker --require-tests --test-only checkercontrol.CheckerTest, sha256sum -c .flow/tmp/fn19519/final-source-v2.sha256, sha256sum -c .flow/tmp/fn19519/final-artifacts-v2.sha256, git diff --check
- PRs: