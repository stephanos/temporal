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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
