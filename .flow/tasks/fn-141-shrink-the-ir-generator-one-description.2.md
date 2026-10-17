---
satisfies: [R2, R16]
---
# fn-141-shrink-the-ir-generator-one-description.2 Retire unused authoring spellings

## Description
Part D: recount the DSL's alternative spellings, retain every spelling used by the current mini/shared/kit/sample sources or a kept sugar's implementation, and retire unused author-facing alternatives with their lifter arms and obsolete fixtures. MILESTONES.md's pull-forward, mini-only and faster-tasks policies supersede the original paths and full-gate cadence. The owner authorizes the conductor to follow its recommendations for routine choices; report the count-based list to the conductor before removal, without a new user-approval gate.

**Size:** M
**Touches:** [model/framework/dsl/**, model/framework/realize/**, model/framework/export/**, model/framework/lint/**, model/framework/check/**, model/temporal/check/**, model/temporal/realize/Realize.scala, model/temporal/migrate/README.md, model/README.md, model/framework/SEMANTICS.md]

### Approach
- Re-read the current framework and exporter and recount first. Inventory every multi-spelling construct, separating live mini/shared/kit/sample use, internal core/sugar dependencies, generic tests and exporter fixtures. Real-feature-only use does not constrain retirement and real-feature sources are not edited. Immutable historical sources and payloads remain untouched.
- Re-check the historical candidates individually: hand-bound Bindings, hand-written Typed* realization constructors, Instruction.readUntil versus scoped readUntil, receiver-less capabilities(limits)(...), chained overriding, and explicit Query.total(n). A class used as a kept sugar's return type or core representation is not automatically an unused author-facing spelling; retain the core mechanism and type safety.
- Reserve spellings explicitly needed by remaining approved source specs. Consult fn-141.3-.13 and fn-164/fn-165 and the current milestone before deciding a spelling is unused. A used core form or future requirement stays; record why. Do not preserve obsolete public syntax merely because an exporter fixture uses it.
- Report the full count-based removal/retention list to the conductor before deletion. The conductor's recommendation authority already comes from the user. Delete each retired author-facing spelling, its exporter recognition, obsolete positive/reject fixtures and obsolete guide text together. A surviving DSL test using retired surface syntax is converted to its canonical spelling without reducing its assertion coverage; do not substitute a renamed compatibility alias for retirement.
- Add meaningful compile-negative controls proving retired public spellings are unavailable, alongside positive controls for retained/canonical forms. Retain surviving fixture expected IR and refusal contracts; expected artifacts solely for explicitly retired fixtures may be removed with them. Keep changes minimal and split affected construct tests from Fixtures.test.scala beside their construct as required by the milestone.
- Write a concise migration recipe in model/temporal/migrate/README.md for the real-feature sweep. Record before/after source counts and the exact uses/removal decisions in the task receipt; no bespoke proof scripts, immutable captures or hash manifests.
- Baseline is fn-141.1's green .flow/tmp/checks/20261013-023527/record.md (432 tests). Iterate only affected suites, then run make model-check-changed BASE=<task-base> once at the end, including mini step-table equivalence. Strict model-mini generated-IR comparison, Go, regeneration, broad lint and independent review remain held. Existing generated artifacts are not edited.

## Acceptance
- R2: a use-count and dependency table covers every multi-spelling construct; removal/retention decisions follow the current mini/shared scope, retained core/sugar and upcoming approved requirements, under the owner's recommendation delegation.
- Retired public spellings no longer compile, their lifter arms and obsolete fixtures/docs are gone, and canonical spellings still compile. Internal representations required by kept sugar stay usable without preserving an obsolete author-facing alternative.
- Surviving fixture expected IR/refusal behavior and mini/shared semantics are preserved. The task's model-check-changed record passes with mini step-table equivalence. Generated/real-feature sources are unchanged.
- R16: before/after source counts, retired fixture inventory, migration recipe, focused checks and deferred gates are recorded in the done summary.

## Done summary
Restricted direct author construction of nine typed realization records and removed their nine-name lifter mapping. Canonical factories, typed results, fields, case-class data and type checks remain; four constructor-only refusal cases now use factories at the same coordinates.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(config: REVIEW_MODE=none; owner defers independent review to the source-batch boundary)
baseline: green via preceding task record `.flow/tmp/checks/20261013-023527/record.md` (432 tests), reused under the owner's faster-tasks policy. This overrides the historical parent Quick commands and their full-gate cadence.

Similar code search: reused the existing private constructor pattern of `TypedProto`/`TypedEvidence`, the typed factories, `FixtureBuild` compile probes and the complete existing typed-invalid coordinate assertion.

### Use-count and dependency decisions (R2)

Counts are source occurrences at task admission, excluding comment-only lines. Production covers mini, shared foundations/actors/capabilities, the realization kit and the three framework samples; test counts include source strings used to compile probes. Internal creation counts exclude class declarations. No real-feature use constrained a decision.

| Spelling | Production author uses | Kept internal creation | Generic test occurrences | Export/lint fixture occurrences | Decision |
| --- | --- | --- | --- | --- | --- |
| `Bindings` | 0 | 0; `Rules` extends `RuleBook` directly | 23 | 204 | Reserved core oracle for approved fn-141.3/.4; retirement remains open at constructed `RuleBook` export (.10/.14) |
| `Instruction.TypedRpc(...)` | 0 | 2 (scoped `rpc`, `Instruction.rpc`) | 0 | 1 | Retire public creation; keep private case class and public result type |
| `Instruction.TypedPoll(...)` | 0 | 2 (scoped `readUntil`, `Instruction.readUntil`) | 0 | 1 | Retire public creation; keep private data |
| `Recorded.TypedRead(...)` | 0 | 1 (`Recorded.read`) | 0 | 1 | Retire public creation; keep private data |
| `Recorded.TypedSingle(...)` | 0 | 1 (`Recorded.single`) | 0 | 1 | Retire public creation; keep private data |
| `Recorded.TypedRunEvent(...)` | 0 | 1 (`Recorded.runEvent`) | 0 | 0 | Retire public creation; keep private data |
| `TypedAssignment(...)` | 0 | 1 (`Assignment.typed`) | 0 | 0 | Retire public creation; keep private data |
| `TypedResponseRead(...)` | 0 | 1 (`ResponseRead.typed`) | 0 | 0 | Retire public creation; keep private data |
| `TypedEvidenceField(...)` | 0 | 1 (`EvidenceField.typed`) | 0 | 0 | Retire public creation; keep private data |
| `WorkflowHistory.TypedHistory(...)` | 0 | 1 (`WorkflowHistory.event`, in the kit) | 0 | 0 | Retire public creation; keep private data; conductor expanded Touches to the shared-kit file |
| `Instruction.readUntil(...)` | 1 kit use (`Modules.scala:67`) | Builds typed poll data | 0 | 6 | Retain: `DescribedStatus.await` depends on it |
| Receiverless `capabilities(limits)(...)` | 0 | 0 | 0 | 0 | Already absent; compile-negative control pins absence |
| Chained `.overriding(...)` | 0 | 0 | 0 | 0 | Already absent; compile-negative control pins absence; statement `overriding` stays |
| Explicit `Query.total` | 6 mini uses | Carries asserted total; witness/core Query data need it | 0 | 82 literal total occurrences | Retain: five infix calls and one `.total(225)` |

`TypedProto`, `TypedProtoField`, `TypedProtoValue`, `TypedProtoEntry`, `TypedEvidence` and `TypedOperand` already have private constructors and remain the retained factories' typed representations. None was removed merely for its name.

The broader multi-spelling sweep retained the following families. Counts below are production source occurrences; overloads for arity, supplied names and domains remain where the retained family or an approved source task needs them.

| Construct family and current production counts | Kept core/sugar dependency or approved requirement |
| --- | --- |
| `Step(...)` 12; `enter(...)` 35; `stay(...)` 2; `reject(...)` 1 | Used core and effects; fn-141.3 requires sugar/core equivalence |
| `effect { ... }` 0 alongside function-returning effects | Existing function sugar, preserved for generic definition expansion and author-time subset checks (.3/.4); no new effect form added |
| Claim patterns (`once`/`never`/`stays`) 4; `holds`/`holdsAcross` builders 25 | Used sugar and core; .4 and fn-164 require these builders |
| `sticky`/`stickyAcross` 0; `monitor` 1 | .4 explicitly requires both monitor spelling oracles; no retirement before its proof |
| `Rules` 14; facet aggregation `Facets` 6 | Used rule and facet layers; underlying `RuleBook`, `Bound.Function`, `StepBinding`, `~>` retained for derivations and .10 |
| Named `:=` assignments 3; `.input(token)` 4; `.input[A](name)` 3 | Live input tokens/classes and retained named-input sugar; fn-165 owns their record replacement and deletion |
| Bare `witness` 11; receiver-qualified witness 0 | Keep witness builder overloads and their core Property/Scenario/Query triple for .12; generic owner-based fixtures cover shared claims |
| Explicit action name `action("name", actor)` 0 | .5/.9 require explicit-name fallback when no val/object supplies a name; inferred action/actor names and explicit constructors remain |
| Scoped `rpc` 2; core `Instruction.rpc` 0 (18 fixture calls) | Canonical typed factories and scope implementation retained for complete runtime data in .7/.8 |
| Scoped `readUntil` 1 kit use; core `Instruction.readUntil` 1 kit use | The kit depends on both levels; .7/.8 complete and export their data |
| Scoped `proto[...]` 0; `Proto[...]` 4 kit calls | Retained literal scopes and typed core values used by duration/payload/failure helpers; .7/.8 require complete factory values |
| Object machine/composition/realization headers and constructed core values | Used public headers assemble retained typed core values; .7-.13 export those values, so their constructors are core mechanisms, not retired author aliases |
| `synced`/`own`, `through`, rule `on`/`from`/phase/guard variants, `records`/membership/implication variants | Core/sugar pairs named by .3/.4/.10/.11; composition `records` needs its member selector and rules retain their typed arity/domain checks |

The count-based recommendations were sent to the conductor before deletion. The conductor reserved `Bindings` explicitly for .3/.4 and directed its later removal; this checkpoint does not discharge the final unused-spelling obligation. The compiler probe showed Scala 3.9 makes generated `apply` and `copy` as private as a case class's constructor, so changing constructor visibility retains the case-class representation while retiring public creation.

### Fixtures, source counts and migration (R16)

Converted `typedInvalid/Invalid.scala`'s four direct-constructor alternatives (`directWrongRpc`, `directWrongPoll`, `directWrongRead`, `directWrongSingle`) to `Instruction.rpc`, `Instruction.readUntil`, `Recorded.read` and `Recorded.single`. The mismatch checks remain at 107:7, 118:7, 128:3 and 132:3; all 24 original compiler refusal coordinates remain unchanged. Moved that assertion from `Fixtures.test.scala` to `TypedDeclarations.test.scala`, preserving its complete expected list. No expected IR/refusal artifact was exclusively owned by the retired variants, so none was removed or rewritten; surviving golden fixtures stay intact.

Main-source sizes count tracked `.scala` files excluding `.test.scala`, `testdata` and build output, from pre-edit `50b6cd6cac` to the checkpoint:

| Area | Before | After | Delta |
| --- | --- | --- | --- |
| Framework exporter (current IR-generator home) | 12,280 | 12,269 | -11 |
| Framework DSL | 3,792 | 3,792 | 0 |
| Framework realization core | 1,072 | 1,072 | 0 |
| Framework checks | 3,868 | 3,868 | 0 |
| Framework lint | 3,003 | 3,003 | 0 |
| Temporal checks | 2,124 | 2,126 | +2 registry entries |
| Temporal realization `Realize.scala` | 263 | 263 | 0 |

Net main source: -9 lines. Exporter tests add 107 lines for the constructor controls and 61 for the moved refusal assertion, removing 33 from `Fixtures.test.scala` (net +135 test lines); the existing 170-line typed-invalid fixture keeps its line count. The short migration recipe is `model/temporal/migrate/README.md#typed-realization-construction-fn-1412`, with all nine factory replacements and instructions for scopes, typed annotations and former copies. No new Scalafix rule, generated payload, real-feature source, Go/tool/Makefile change, bespoke capture or manifest shipped.

### Verification

- Pre-change red: `RetiredSpellingsTest`, 11 tests, nine intended failures because all nine direct constructors compiled; canonical factories and the already-absent forms passed (`.flow/tmp/fn-141.2-red-all.log`).
- Focused green: `mise exec -- scala-cli run --server=false --test model/framework/export model/temporal/export --main-class org.junit.runner.JUnitCore -- framework.export.RetiredSpellingsTest framework.export.TypedDeclarationsTest`, 12 tests passed. Each retired type rejects universal/companion creation, `new` and generated `copy`; canonical typed factories compile and all 24 refusal positions match (`.flow/tmp/fn-141.2-focused.log`).
- Scoped model jar packaging passed; only the two new test files were formatted; `git diff --check` passed.
- First final scoped check: `make model-check-changed BASE=50b6cd6cac` exited 2; `.flow/tmp/checks/20261013-030555/record.md` records 139 model tests passed, 288/289 lifter tests passed, and mini step-table equivalence passed. The sole failure was the already-recorded fn-169.1 native-probe classpath/build race: `framework.lint.PropertyGroupsFixtures` could not load `framework.dsl.PropertyGroupsNative`. Its retained flat fixture jar contains that class, and a direct launch with the retained classpath passed with `PROPERTY_GROUPS_NATIVE_OK predicateRows=30 monitorRows=8 properties=6 progress=4`. No assertion, fixture oracle or gate was changed to hide the failure.
- Standalone Property-groups suite: `mise exec -- scala-cli run --server=false --test model/framework/export model/temporal/export --main-class org.junit.runner.JUnitCore -- framework.lint.PropertyGroupsFixtures`, all 13 tests passed (`.flow/tmp/fn-141.2-property-groups.log`).
- Final scoped rerun: `make model-check-changed BASE=50b6cd6cac` exited 0; `.flow/tmp/checks/20261013-031649/record.md` passed in 558.1 s. All 597 tests passed (139 model, 289 lifter, 154 check, 15 migration), across 99 selected suites; 17 behind-feature suites were skipped by name. All three framework samples passed, and all 16 mini/shared step tables are equivalent to `.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2` under the existing map. The constructor and surviving fixture/refusal controls passed in this run too. The first failed record remains recorded separately; the conductor authorized this rerun after a fresh failure, and the green receipt was issued only after the complete rerun exited 0 (`c635a6ef-model-check-changed.json`).

Held: strict comparison against checked-in mini generated IR, real-feature migration/lifting, broad Model/lifter lint, full Scala gate, independent review, Go consumers, regeneration and live execution. The scoped record reports generated-IR drift as held and the behind-feature suites by name; none is credited as passing. No review verdict is claimed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c635a6ef5a7eca817a8670f693a9c5b87e555f74
- Tests: make model/build/model-scala.jar FEATURES=activity_mini, mise exec -- scala-cli run --server=false --test model/framework/export model/temporal/export --main-class org.junit.runner.JUnitCore -- framework.export.RetiredSpellingsTest framework.export.TypedDeclarationsTest, mise exec -- scala-cli run --server=false --test model/framework/export model/temporal/export --main-class org.junit.runner.JUnitCore -- framework.lint.PropertyGroupsFixtures, make model-check-changed BASE=50b6cd6cac, git diff --check
- PRs: