---
satisfies: [R7]
---
# fn-155-name-the-standalone-activitys-repeated.1 Capture the baseline, build the equivalence projection and probe helper forms

## Description
This task sets up the proof every later task relies on (spec R7, Early proof point). It captures the post-fn-151 baseline, builds the normalizing projection, and writes the identity-mapping skeleton. It also probes the lifter for each helper form sections A–F plan to use. It does not refactor any model.

**Size:** M
**Files:** `.flow/tmp/fn-155/` (baseline IR, step-table dump, `project.py`, `mapping.md`, probe notes); scratch probe edits reverted before handoff
**Touches:** [.flow/tmp/fn-155/**]

### Approach
- **Re-anchor.** Start only after fn-151-split-standalone-activity-into-smaller closes. Re-anchor every section of the spec to fn-151's files (`model/temporal/features/activity/standalone/system/**`, `product/Product.scala`, `Standalone.scala`) and record the new file:line anchors in `mapping.md`. The spec's pre-split lines are stale.
- **Baseline capture.** Start from fn-151's closed, committed production IR and its passing production-generation, complete native and seven-Model Case receipts. Verify the exact source/artifact pins before copying `activity-standalone*.json` into `.flow/tmp/fn-155/before/`; record the closed commit and receipt hashes. Do not repeat fn-151's unchanged `make umpire-check-model` no-update invocation merely to capture the same baseline. Its generator OOM and scratch ENOSPC are RED, explicitly deferred to fn-157, and supply no passing gate credit. Obtain any genuinely missing lifted input through the smallest already-established producer, with full domains and identity pins unchanged. Changed proof inputs require affected verification; production regeneration and final gate disposition belong to task .6.
- **Step-table dump.** Dump the step table the Pins test checks (`StandaloneActivityPins.test.scala:1101-1104`, helpers `:104-124`) for Product and System. Also dump the interpreter's tables built from the lifted IR for every other activity machine (the Dispatch designs, HeldDispatch, the compositions). Later tasks diff both.
- **Projection.** Write `project.py`, following `.flow/tmp/fn-126/` and `.flow/tmp/fn-132.6/prove.py`. It applies the mapping (many-to-one effect merges allowed) and erases positions. It normalizes per spec §Edge Cases: inline lifted `states` and effect definitions at their call sites, fold a `copy` over the initial `construct`, and drop unreferenced definitions. It reports the remaining differences instead of only asserting equality, and `mapping.md` gains a structural-review section for those.
- **Lifter probes.** Probe the open forms against irgen on a scratch edit, then revert:
  - `enter(…, states.landingStatus(s), …)` with a computed status fact;
  - a `states` def called inside `property.holdsAcross`;
  - `.because(reasons.x)` referencing a value in another object or package;
  - `ActivityRecord.restrict(...).rebind(...).refining(ActivityProduct)(...)` (the record declaration now lives in `system/Dispatch.scala`) for HeldDispatch (`restrict` does not inherit a refinement, `model/framework/Machine.scala:410-412`, `:456`), and whether its monitors and evidence carry over;
  - a capability class shared by `OverQueue` and `OverMatching` through a common bound on the composed record (`c.own(_.activity, …)` needs it, `model/framework/Compose.scala:93`), and how the lift pinned at `model/irgen/test/Fixtures.test.scala:2279-2284` changes;
  - `respondFailedByID` reusing the kind's `respondFailed` examples;
  - one scratch named-guard rewrite and one scratch effect merge, run through `project.py` to show they reduce to mapped identities.
- **Record outcomes.** Record each probe's outcome in `mapping.md` as `accepted`, `refused → fallback`, or `refused → deferred`.

### Investigation targets
**Required:**
- `model/irgen/Expressions.scala:600-670` — call lifting and `.because` limits
- `model/irgen/Declarations.scala:790-870` — guard and effect forms in rules
- `model/irgen/Syntax.scala:180-300` — `effect {}` lifting and status ordering
- `model/temporal/features/activity/standalone/StandaloneActivityPins.test.scala:1050-1104` — binding order and step-table pins
- `.flow/tasks/fn-151-split-standalone-activity-into-smaller.3.md` — how fn-151 states its identity mapping

**Optional:**
- `.flow/tmp/fn-132.6/prove.py`, `.flow/tmp/fn-126/` — earlier projection scripts

### Acceptance
- [ ] Baseline IR and step-table dump stored under `.flow/tmp/fn-155/before/`, taken from the closed fn-151 tree
- [ ] `project.py` reports equality on the baseline against itself, flags a deliberate one-fact edit, and reduces the scratch named-guard rewrite and effect merge to mapped identities, or the residue is recorded and section A's scope re-evaluated
- [ ] `mapping.md` lists the re-anchored locations and an outcome for every probe
- [ ] No model source changes remain

### Candidate proof evidence (IN_PROGRESS)

The task remains IN_PROGRESS. This lifecycle receipt presents the ignored executable proof substrate for review; it records no review verdict or completion. The empty implementation checkpoint is not acceptance evidence.

The complete content seal is `.flow/tmp/fn-155/task1/proof-manifest.json`, SHA256 `983c66b072cfd926ae6358bdb332a163a53cab87062664542da68fe19cdf7ef6`. Its files map pins the exact scripts, scratch source/artifacts, all dumps and receipts listed below. The closed source baseline is `999fb4fdc4eaa559539ab443c2dc3e99349dc182`; 1134 production input pins remain unchanged.

The whole inventory comparison reads all 46 tables and 13,891,948 ordered records, including every native state/action pair and both unchanged Pins expected-rule lists. Complete Model projections pass self and both actual named-guard/effect-merge variants. A deliberate one-fact Model mutation and one logical native Because mutation are rejected. The two final native helper receipts use the harness without explicit GC/resource controls; superseded observations have no gate credit. The canonical RED baseline and task .6 gate ownership remain as stated above.

The review acceptance surface consists of these pinned executable implementations and observations. It must establish their correctness and task coverage through the artifacts themselves; the receipt prose and empty source diff supply no SHIP evidence.

- `.flow/tmp/fn-155/project.py`
- `.flow/tmp/fn-155/compare_cases.py`
- `.flow/tmp/fn-155/mapping.md`
- `.flow/tmp/fn-155/identity-map.json`
- `.flow/tmp/fn-155/table-identity-map.json`
- `.flow/tmp/fn-155/before/baseline.json`
- `.flow/tmp/fn-155/before/identity-inventory.json`
- `.flow/tmp/fn-155/before/step-tables/completeness.json`
- `.flow/tmp/fn-155/task1/capture_baseline.py`
- `.flow/tmp/fn-155/task1/test_project.py`
- `.flow/tmp/fn-155/task1/controls.py`
- `.flow/tmp/fn-155/task1/verify.py`
- `.flow/tmp/fn-155/task1/seal.py`
- `.flow/tmp/fn-155/task1/tables/dump.go`
- `.flow/tmp/fn-155/task1/tables/PinsDump.scala`
- `.flow/tmp/fn-155/task1/tables/run.py`
- `.flow/tmp/fn-155/task1/tables/compare.py`
- `.flow/tmp/fn-155/task1/probes/run.py`
- `.flow/tmp/fn-155/task1/probes/native.go`
- `.flow/tmp/fn-155/task1/dispatch-probes/lift.py`
- `.flow/tmp/fn-155/task1/dispatch-probes/table.go`
- `.flow/tmp/fn-155/task1/dispatch-probes/run.py`
- `.flow/tmp/fn-155/task1/final-verification.json`
- `.flow/tmp/fn-155/task1/projection-self.json`
- `.flow/tmp/fn-155/task1/projection-combined.json`
- `.flow/tmp/fn-155/task1/projection-named-status.json`
- `.flow/tmp/fn-155/task1/projection-one-fact.json`
- `.flow/tmp/fn-155/task1/controls-receipt.json`
- `.flow/tmp/fn-155/task1/cases-self.json`
- `.flow/tmp/fn-155/task1/tables/self-equality.json`
- `.flow/tmp/fn-155/task1/tables/negative-control/negative-control.json`
- `.flow/tmp/fn-155/task1/probes/probe-outcomes.json`
- `.flow/tmp/fn-155/task1/dispatch-probes/results.json`
- `.flow/tmp/fn-155/task1/dispatch-probes/notes.md`
## Acceptance
- [ ] TBD

## Done summary
# fn155 task1 handover

Captured the closed fn151 baseline and built the complete Model projection, native/Pins table proof, negative controls and A–F lifter mapping. All task1 acceptance criteria hold; no production model, artifact or test source changed.

Tier: session (jev-unavailable(no_key)). The host exposes no executed implementer model metadata. Three independent proof lanes supplied baseline tables, helper probes and dispatch probes; the worker reconciled their exact inputs, failures and receipts.

stage: impl-review - ran [2026-10-10T00:05:56.705385Z..2026-10-10T00:17:55.092486Z] - SHIP, codex:gpt-6.1-sol:high. This is the explicitly requested same-family project reviewer, not an independent model-family review. All three draws returned SHIP with zero findings and no unaddressed requirements.

### Proof seal and provenance

All relative pointers below resolve from `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn-155-name-the-standalone-activitys-repeated`.

- `.flow/tmp/fn-155/task1/proof-manifest.json` SHA256 `983c66b072cfd926ae6358bdb332a163a53cab87062664542da68fe19cdf7ef6` seals 1271 files, 470927734 bytes. This reviewed seal remains unchanged. Final lifecycle/review/lint receipts added afterward are separate from the sealed proof.
- `.flow/tmp/fn-155/before/baseline.json` pins 1134 production inputs to closed fn151 commit `999fb4fdc4eaa559539ab443c2dc3e99349dc182`; 44 byte-sealed local proof/gate receipts remain in `before/receipts/`.
- Six complete activity IR/lint files remain in `before/ir/`. All 31 baseline Case JSON files, including the seven-Model manifest, remain in `before/cases/`.
- `before/identity-inventory.json` covers every declaration in all three complete activity Models, including all 154 Queries.
- `.flow/tmp/fn-155/mapping.md` supplies current post-fn151 source anchors, every probe outcome, structural-review rules and exact reusable commands. `identity-map.json` and `table-identity-map.json` are separate empty mapping skeletons.

Later isolated lanes must verify the reviewed seal and selected production input pins before consuming these absolute pointers. Do not copy a partial proof or treat the empty implementation checkpoint as acceptance evidence.

### Complete proof observations

`task1/final-verification.json` indexes the proof receipts. `project.py` traverses the whole Model, preserving Query coverage, bounds, expectations, refinements, monitors, evidence, realization/controller semantics and all ordered behavior lists. The six-file self projection and both actual whole-primary-Model scratch variants have no unexplained residue. `task1/controls-receipt.json` records the actual one-fact mutation returning exit 1 with two dependent difference paths. Six focused projection tests passed, including capture/precondition retention, identity mapping, computed landing, semantic negatives and Case bytes/fact order.

`before/step-tables/completeness.json` contains the full inventory: 36 native machine occurrences and eight composition occurrences, 13108098 state/action pairs, 2349892 enabled pairs, 10758206 disabled pairs and 2350141 ordered result alternatives. Composition tables retain the interpreter's complete reachable domain under unchanged default scope, not an invented Cartesian domain. The two source-compiled Product/System Pins tables preserve both original expected-rule lists and run the original assertions.

`task1/tables/self-equality.json` and its receipt cover all 46 native/Pins tables and 13891948 ordered records, validating both complete sides before comparison. The actual logical native Because mutation is rejected at `$.native_steps[0].Fields[3].Text`. Strict positions, explicit position allowance, accompanying Because mutation, injective qualified native identity mapping and mapped-filename collision controls have separate receipts under `task1/tables/focused-controls/`. The first incorrect negative-wrapper assertion is retained without passing credit; the corrected attempt is authoritative.

`task1/cases-self.json` records byte equality for all 31 retained Case files. These observations are task1 foundation proof, not fresh canonical generator or Go-suite passes.

### Probe decisions for downstream tasks

The actual named guard, computed landing phase/status effect merge, `states` call inside `holdsAcross`, cross-object/package reason values and kind-owned shared example text values compile and lift. Both complete Model projections pass; the corrected native harness compares full Product/System domains, ordered facts, reasons, public identities, reachability and required metadata without resource controls.

The active helper harness is `task1/probes/native.go`, SHA256 `4b2a1a0b4bf847fcb8772ee901673bc530e81d561165a1ec8017ceac05954527`. Only `native-combined-no-forced-gc.json` and `native-named-status-no-forced-gc.json` supply native helper credit. Earlier explicit-GC observations and harnesses remain preserved and invalidated; no production model or baseline resource knob changed.

R5 must use the measured fallbacks. Restrict/rebind/refining fails the actual lifter because restriction removes the refinement. An unrefined diagnostic has equal transition rows/monitors/evidence but loses refinement and native Product StateFields, so HeldDispatch stays handwritten. The common composition capability bound compiles but the lifter refuses its class-owned type; both existing capability classes stay. The fixture capability root is included. Exact commands, source patches, logs, failures and assertions are under `task1/dispatch-probes/`.

The example probe proves shared kind-owned text values in `Activity.scala`, not reuse of an action's `examples` collection. Root added that narrowly scoped file to task .3 after reviewing this evidence. No production refactor is included here.

### Review and final verification

The actual review receipt is `task1/review/receipt.json`; raw executable artifact inspections, per-draw metadata and verdicts are in `task1/review/`. `task1/review-receipt-pins.sha256` seals those copies. RID `924fcbedeede4daeb517386525880579` reviewed base `3eca0a3b200ed617cd95acea000ae1c6b4ec6a4d` and head `6fd35fbd3fe5ccc909eaa047a8e12edbe8c890e4`. All draws verified the 1271-file seal, 1134 production pins, complete projections and negatives. The correctness draw independently completed all 46 tables/13891948 records and the actual native mutant; `review/exhaustive-check.log` preserves its result. SHIP therefore certifies executable proof, not only tracked receipt prose.

Fresh final checks verified all 1271 sealed hashes and all 1134 production hashes, with no production range diff. `task1/final-lint.json` records `make lint-model` exit 0 in 42.34 seconds under the actual exclusive `/tmp/umpire-heavy-gates.lock`. Its actual cwd is the unchanged byte-pinned fn151 checkout, not this worker checkout. The baseline lint receipt records a separate pre-edit pass.

baseline: red (`make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` and ordinary Case/fixture/Go gates remain inherited RED). No green handoff is asserted. Per the authoritative task1 policy, unchanged expensive failed generators were not repeated; fn157 owns native OOM/scratch ENOSPC, fn154 owns Quint and Batch5 owns inherited completion/fatal/pause semantics. Production regeneration and final canonical disposition remain task .6. `canonical_gate_credit` remains false.

The final tracked-range classifier reports `TIER_B: docs-only (3 files)` for lifecycle metadata only. It does not classify ignored executable proof or convert inherited RED into green:

GATE_SKIPPED:model:docs-only - cumulative tracked lifecycle diff classified tier-B (no executable paths touched); canonical baseline remains RED under task1 policy.

GATE_SKIPPED:cases:docs-only - cumulative tracked lifecycle diff classified tier-B (no executable paths touched); canonical baseline remains RED under task1 policy.

All worker-owned commands and delegated probe commands exited. Completion is recorded only through Flow's actual done command and verified terminal status; the candidate description remains an explicitly historical IN_PROGRESS receipt.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 781bf6965054d09b07f350168b1da8d5f54b88aa, 6fd35fbd3fe5ccc909eaa047a8e12edbe8c890e4
- Tests: python3 .flow/tmp/fn-155/task1/test_project.py, python3 .flow/tmp/fn-155/task1/controls.py, python3 .flow/tmp/fn-155/task1/verify.py, python3 .flow/tmp/fn-155/task1/tables/compare.py self .flow/tmp/fn-155/before/step-tables --receipt .flow/tmp/fn-155/task1/tables/self-equality.json, python3 .flow/tmp/fn-155/task1/tables/compare.py negative-control .flow/tmp/fn-155/before/step-tables .flow/tmp/fn-155/task1/tables/negative-control --receipt .flow/tmp/fn-155/task1/tables/negative-equality.json, python3 .flow/tmp/fn-155/task1/tables/focused_controls.py .flow/tmp/fn-155/before/step-tables .flow/tmp/fn-155/task1/tables/focused-controls, make lint-model, GATE_SKIPPED:model:docs-only - cumulative tracked lifecycle diff classified tier-B (no executable paths touched); canonical baseline remains RED under task1 policy., GATE_SKIPPED:cases:docs-only - cumulative tracked lifecycle diff classified tier-B (no executable paths touched); canonical baseline remains RED under task1 policy., BASELINE_RED: unchanged canonical Model/Case/Go gates; exact receipts retained; no canonical green credit; final disposition task .6/fn157.
- PRs: