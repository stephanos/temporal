---
satisfies: [R4, R5, R7]
---
# fn-195-modelsuite-a-models-test-file-holds-its.18 Add realize expectation conveniences and shared conformance validation

Touches: [model/framework/realize/Expectations.scala, model/framework/realize/Expectations.test.scala, model/framework/REFERENCE.md, .flow/tmp/fn195-expectations/**]

## Description
Add the accepted .10 convenience tuples and one shared validator for later conformance admission. Reuse the existing RunExpectation and MonitorExpectation hierarchy while preserving raw apply/copy compatibility. This leaf is independent of waited coverage, inventory and capability feasibility; .11 retains actual conformance-entry integration.

**Size:** M, three expected files and four acceptance groups.
**Files:** new `model/framework/realize/Expectations.scala`, new `Expectations.test.scala`, and the expectation/native-conformance sections of `model/framework/REFERENCE.md`. Existing .5 owns final README/global guidance synchronization.

### Approach
- Expose satisfied, conformant and the accepted property-only inconclusive(reason, because) conveniences from framework.realize. Preserve the existing nine-field order and independent conformance/property/Contract/disposition/cleanup/reason/conformanceReason/monitor/because meanings at `Realize.scala:736`.
- Use the frozen .10 tuples. Path satisfied is conformant/satisfied/satisfied/completed/succeeded with absent reasons, empty monitors and empty because. Sample conformant changes the singular Property to inconclusive with neverEvaluated and exactly `A sampled path selects no individual Property`; its Contract/disposition/cleanup remain satisfied/completed/succeeded.
- Restrict the property-only inconclusive shorthand to explanationsDisagree, neverEvaluated and unreadable with supplied nonblank because. Other reasons require an explicit full RunExpectation. Preserve independently mixed fields instead of deriving outcomes from one another.
- Provide private[framework] `validateConformanceExpectation(value: RunExpectation): Unit`, reporting invalid metadata with IllegalArgumentException detail. C later wraps that detail in its located diagnostic. Convenience outputs and later retained raw/copied values use this same rule set without checker-owned types. Preserve the existing raw constructors and restricted copy, including the deliberately accepted copied-empty raw value at `Expectation.test.scala:97`; reject such values only at conformance validation.
- Cover all nine independent Property/Contract pairs. Shared reason is empty exactly when both assessments are satisfied; because is empty in that pair and trim-nonempty in each other pair. ConformanceReason is empty exactly when conformance is conformant. Each monitor's reason is empty exactly when its own outcome is satisfied, with empty because for satisfied and trim-nonempty because otherwise. Map no semantic outcome from another field. Preserve monitor names/references unchanged. C still resolves monitor identity, duplicates and foreign-owner references when binding an actual declaration.
- Document the convenience tuples and shared admission boundary in REFERENCE without Scala-file links or a claim that ModelSuite conformance has shipped. Existing .5 consumes this interface for final guidance; .11 owns revalidation at its real entry point and .3 retains old vocabulary retirement.

### Investigation targets
**Required:**
- `.flow/artifacts/fn-195-modelsuite-a-models-test-file-holds-its/conformance-admission.md` - accepted complete tuple table and shorthand domain.
- `model/framework/realize/Realize.scala:736` - RunExpectation raw constructor, copy and independent fields.
- `model/framework/realize/Realize.scala:833` - MonitorExpectation construction/copy and field ownership.
- `model/framework/dsl/Expectation.test.scala:72` - existing copy-bypass controls and copied-empty compatibility neighbor.
- `.flow/tmp/fn1952-conformance-proof/CapabilityProbe.scala:76` - executed scratch tuple/validator controls.
- `model/framework/REFERENCE.md:1165` - task-owned expectation forms and refusals.
**Optional:**
- `model/framework/REFERENCE.md:1461` - native ModelSuite guidance and current integration limit.

### Verification and handoff
Run focused convenience/validator controls, the unchanged raw expectation compatibility neighbors and fresh outside-package source/separately packaged library clients under existing strict flags, with fresh complete-recipe workspaces. Retain exact selectors/nonzero counts, private-constructor/copy compile diagnostics, artifact hashes, commands and terminal exits under task-owned `.flow/tmp/fn195-expectations/`. Confirm declaration-only loading without checker initialization. The helpers express authored metadata and establish no observed Run/server verdict.

Keep .11 blocked on actual integrated inventory/capability prerequisites and its other leaves plus .9; this task does not register suites or make declarations eligible. Preserve .8/.14 and fn-196.2 holds, .3 retirement ownership and phase-batch review deferral. Temporal reads, actual repository Make, Go, export/lifting/IR/schema/backend/generated/regeneration, server/recorded-Run and Case operations remain held; export compile-only permission remains pending and ungranted.

## Acceptance
- [ ] Public framework.realize conveniences produce every accepted nine-field path/sample default, including sample Property inconclusive/neverEvaluated and its exact fixed because text, independent satisfied Contract/completed disposition/succeeded cleanup, absent conformanceReason and empty monitors. No default invents a selected singular Property or an observed result.
- [ ] Property-only inconclusive admits explanationsDisagree, neverEvaluated and unreadable with nonblank because; unsupported shorthand reasons refuse and remain expressible through the independent full RunExpectation. Mixed conformance/Property/Contract/disposition/cleanup/reason/monitor controls retain every supplied field.
- [ ] One reusable conformance validator checks full/raw/copied values and monitor values. It refuses omitted, empty, whitespace-only and copied-empty text for non-satisfied Property/Contract assessments, requires empty text for fully satisfied assessments, and checks all nine Property/Contract pairs with shared reason empty exactly when both are satisfied. ConformanceReason is empty exactly when conformant; each monitor has reason empty exactly when satisfied and its own matching text rule. Existing raw apply/copy behavior and private-construction/copy refusals remain unchanged; C retains actual monitor identity/owner/duplicate admission.
- [ ] Focused and fresh outside-package source/library controls execute nonzero counts under strict flags and load declarations without the checker. REFERENCE documents the forms and validation boundary without Scala-file links or shipped C claims. .11 explicitly consumes this validator at actual retained-value admission; .3 retains retirement and .5 retains final guide synchronization.

## Done summary
Added `framework.realize.satisfied`, `conformant` and the three-reason Property-only `inconclusive` shorthand, plus the shared `private[framework] validateConformanceExpectation` boundary. The six focused tests cover all nine independent Property/Contract pairs, conformance reasons, monitor metadata, raw/copied empty text and mixed fields; existing raw constructors and restricted copy remain unchanged.

The native source and separately packaged-library routes each pass 13 tests: ExpectationsTest 6, unchanged ExpectationTest 3, outside ExpectationsClientTest 3 and checker-absent DeclarationsLoadingTest 1. Both compile-only refusal routes exit 1 with exactly nine intended diagnostics: two private constructors, two outcome-changing copies, two nonempty explanation copies, two satisfied raw explanation arguments and the private validator's inaccessible boundary. ReferenceSurfaceTest passes 4 tests, scoped formatting check and `git diff --check` pass, and the required checkercontrol.CheckerTest passes 4 tests before and after implementation. All owned command handles are joined.

baseline: green (fresh actual Quick 4 tests and unchanged raw neighbor 3 tests; no handoff or receipt reused). The first raw-neighbor attempt used an incomplete source selection and exited 1; it is inconclusive, not a product failure or a passing baseline. The corrected command passed before tracked edits. The executable first RED failed one intended MUnit assertion at 9d03fb2412 before implementation; the next RED source attempt refused the missing helper names. Source-v1 then exposed a discarded exception value in the scratch loader under strict flags; the fixture was corrected without changing flags. These earlier logs remain retained and receive no green aggregate credit.

Final source/library commands, logs, terminal statuses, source/dependency manifests, refusal diagnostics and the verifier live under `.flow/tmp/fn195-expectations/`. Source-v3, library-v1 and client-v1 bind the final formatted product sources; source-refusals-v1 and library-refusals-v1 bind the same sources. `verify.sh` exits 0 and checks counts, source equality, both raw files unchanged and the immutable dependency. The packaged library SHA-256 is `e2c6a13a63b6a7f5fbc3cc160cc93678992d49aa5fde2ab23ac1d8abc660f5a1`; frozen `model/build/ir-scalapb.jar` remains `b9e0538afffad2b33b9fc8fc9eb2a3f4e63b10928fa38506b585747535e85c92`.

Gate classification reports FULL, exit 1, due unmatched REFERENCE. This is faithfully retained in `gate-classify.log`; it supplies no broad model-gate pass. The owner authorized native focused recipes and the parent Quick only. Actual Make, Temporal reads/builds, Go, export/reify/lifting/IR/schema/backend/generated/regeneration, server/Case operations and publishing remain held. No observed verdict or R4 integration completion is claimed. REFERENCE contains no Scala-file references or links.

Fn-195.11 must call this validator at actual full/retained/copied expectation admission and separately resolve monitor identities, exact ownership and duplicates. It remains blocked on its existing prerequisites. Fn-195.3 retains vocabulary retirement and fn-195.5 final guide synchronization. No feature-map route changed. Tracker sync: n/a (bridge inactive).

Tier: session (jev-unavailable(no_key))

Two fresh conductor-owned quality audits report zero findings on correctness and standards. Their verbatim reports and conductor decision are retained in `.flow/tmp/fn19518/quality-reports.md`. These audits do not replace the owner's deferred formal phase-batch implementation review. The conductor authorized normal task closure after verifying the committed checkpoint and evidence bindings.

stage: impl-review - skipped(config: REVIEW_MODE=none; owner phase-batch formal-review deferral)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9d03fb24125b668f9a8a20abf80743b461fcb2ce, 84baf63e9ba87908a3c81638cecf209bd2488917
- Tests: mise exec -- scala-cli test --server=false model/framework/checker --require-tests --test-only checkercontrol.CheckerTest, timeout 600 mise exec -- scala-cli test --server=false --workspace .flow/tmp/fn195-expectations/baseline/raw-corrected-workspace model/framework/checker --require-tests --test-only framework.dsl.ExpectationTest, timeout 600 mise exec -- scala-cli test --server=false --workspace .flow/tmp/fn195-expectations/red-api-workspace model/framework/checker --require-tests --test-only framework.realize.ExpectationsTest, env -u BASH_ENV bash .flow/tmp/fn195-expectations/run.sh source-v3 source, env -u BASH_ENV bash .flow/tmp/fn195-expectations/run.sh library-v1 library, env -u BASH_ENV bash .flow/tmp/fn195-expectations/run.sh client-v1 client .flow/tmp/fn195-expectations/library-v1/expectations-library.jar, env -u BASH_ENV bash .flow/tmp/fn195-expectations/run.sh source-refusals-v1 source-refusals, env -u BASH_ENV bash .flow/tmp/fn195-expectations/run.sh library-refusals-v1 library-refusals .flow/tmp/fn195-expectations/library-v1/expectations-library.jar, timeout 600 mise exec -- scala-cli test --server=false --scala 3.9.0 --jvm 27 --dependency org.scalameta::munit:1.2.0 --workspace .flow/tmp/fn195-expectations/reference-workspace model/framework/check/lint/Literals.scala model/framework/check/lint/ReferenceSurface.test.scala --require-tests --test-only framework.check.lint.ReferenceSurfaceTest, mise exec -- scala-cli fmt --check --scalafmt-conf model/.scalafmt.conf model/framework/realize/Expectations.scala model/framework/realize/Expectations.test.scala, env -u BASH_ENV bash .flow/tmp/fn195-expectations/verify.sh, git diff --check
- PRs: