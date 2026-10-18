---
satisfies: [R7]
---
# fn-155-name-the-standalone-activitys-repeated.6 Regenerate, prove the mapping and close fn-155

## Description
**Recut 2026-10-18 (one-language series):** Close on the recorded evidence, or re-prove under fn-203's single regeneration; no Go tables and no gen-cases.

Closes the spec: one regeneration, the projection proof, Go test and fixture updates, a docs check, gates and review (spec R7, Decision Context batch nature). It hands the mapping to fn-140.4/.6 and fn-129.3.

**Size:** M
**Files:** `model/ir/activity-standalone*.json`, `model/cases/**`, `model/irgen/testdata/lifts/expected/hints*.json`, affected Go tests (including existing Testpilot identity/binding/replay test seams if needed), `model/README.md` if a cited example changed, `.flow/tmp/fn-155/mapping.md`; conductor owns `MILESTONES.md` updates
**Touches:** [model/ir/**, model/cases/**, model/irgen/testdata/lifts/expected/**, tools/umpire/**/*_test.go, common/testing/testpilot/assessment_external_test.go, common/testing/testpilot/recordedrun/recordedrun_test.go, common/testing/testpilot/evaluation/admission_test.go, common/testing/testpilot/replay/subject_test.go, common/testing/testpilot/internal/execution/wait_expiry_test.go, model/README.md]

### Approach
- **Regenerate.** Run `make umpire-gen-model`, then `project.py` against the task 1 baseline. Every non-position difference must appear in `mapping.md`, either in the identity mapping or with a reason in the structural-review section. Re-dump the step tables and compare them with the baseline.
- **Fixtures.** Refresh the irgen hints fixtures through the gate's `--update` path (`model/check/Gate.scala:495-500`).
- **Go tests.** This task alone updates Go tests that read renamed effects, rule shapes or state-expression shapes: `lower/withholding_test.go:44` (reads `retryCompletes` as a `construct`) and `:47-50`, `export/quint_test.go:258` and `:1188-1191`, `export/open_test.go:154`, `lint/holes_test.go:73,163`, `interp/decisions_test.go:74,91`.
- **Gates.** Run the model gate, `make lint-model`, `make umpire-check-cases` and the Go tooling suite (`-tags test_dep -p 2 -timeout 30m`, under the shared flock). Also run the glossary gate on the new names.
- **Docs.** Check `model/README.md` (Recorded, `toProduct`, `deadlines`, and the reset-override example near the Retries section). Edit it only where a cited example changed.
- **Handoff.** Give `mapping.md` to fn-140.4/.6 and fn-129.3 as their re-anchor. Return milestone-row evidence to the conductor, who alone edits MILESTONES.md.

- **Cumulative proof.** After integrating .2/.3/.4/.5, rerun the sealed original-to-current inventory proof against .1's original baseline and exact joined source/IR/producer/domain inputs. Use .5's fixed eleven Case locations and declaration/full-action identities plus its one separately fixed manifest Query/refusal Position selector. Rederive current coordinates from actual joined Scala and lifted ServerSteps/Script. The .5 observed line numbers are not substitute input pins. Recheck every original file, including Cases previously equal, and reject missing/extra files. Restore only those twelve enumerated leaves after explicit identity substitutions, then compare all 31 raw original files. An unexplained Case or manifest byte delta fails regardless of an IR structural-review entry.
- **Fresh manifest refusal.** Regenerate the current manifest, preserving the immutable original archive. For the exact original `timeouts/startToCloseTimeout` Query and `unsupported[0]` attempts refusal fixed by .5, require original/current unchanged-producer default Check and Find receipts tied to exact source, lifted Script and domain pins. Keep standing, Construct, ID, Owner, Why, path and every other manifest byte identical. Only its source-script coordinate is mapped; derive the manifest digest normally. Run the separate manifest mutation controls and repeat them against joined inputs. This exception creates no Case and permits no Case content-identity substitution.
- **Record the amendment.** Preserve the original strict RED receipt and its 12-equal/8-different diagnostic observation. Publish the approved source-provenance map, exact final input hashes, complete raw-byte comparison, mutation receipts and fresh identity/diagnostic values alongside `mapping.md`. Do not rewrite .1 history or change an inherited Batch 5 expectation, fatal result, reason, status or coverage classification to obtain credit.
- **Current artifacts and historical records.** Regenerate managed model Cases and refresh existing functional/canary execution pins, prepared bindings and receipt references only where the actual changed Case is selected under existing generation contracts. Use `lower.SelectCases` and existing identity/binding code. Retain original Case+Run companion fixtures and recorded bindings unchanged. Verify current-Case/old-Run and current-Case/old-factory rejection. Existing consumers requiring fresh recording/replay keep that requirement; hand off exact current artifacts and required fresh record updates under existing authorization instead of relabeling historical evidence.
- **Timeout detail.** Record its exact production-derived changes and repeat the scoped no-semantic-Detail-consumer proof over the final affected IR/Cases. If a consumer exists or the inventory is incomplete, the narrow amendment does not pass. Contract diagnostics are not interchangeable merely because step tables match.
- **Downstream handoff.** Hand fn-140.4/.6, fn-129.3 and the conductor the final approved mapping and provenance/content-identity pins. fn-156 consumes the actual integrated committed fn-155 closure as its baseline, with its own strict positions-inclusive byte comparison unchanged. A .5 scratch candidate or historical source line is not its baseline.
- **Fail closed outside the twelve enumerated leaves.** The isolated .5 exceptions cover eleven Case waitHint integer fields and one distinct manifest refusal Position, with their fixed original selectors. They do not authorize another task's coordinate change. Halt on every nonallowlisted joined Case or manifest byte difference and return exact independent evidence to the conductor for an explicit separately reviewed contract decision before credit.
- **Identity-control limits.** The .5 constructed INCONCLUSIVE record controls establish production-derived identities, preparation/binding and crossed rejection only. They do not establish matching replay.Admit, genuine violated replay, live execution or semantic equivalence. Preserve all existing true live/replay obligations and actual original Case+Run pairs.

### Investigation targets
**Required:**
- `MILESTONES.md` — Verification instructions and Batches
- `.flow/tmp/fn-155/mapping.md`

### Acceptance
- [ ] Step tables equal the baseline; projection reports only positions, mapped identities and reasoned structural-review entries
- [ ] Model gate, model lint, `umpire-check-cases` and the Go tooling suite pass, or inherited batch 5 failures are shown unchanged from the baseline
- [ ] The final joined inventory matches all 31 frozen original files and exactly eleven Case waitHint line leaves plus one separate manifest refusal Position; restoring only those twelve enumerated leaves recovers raw original bytes after the explicit identity map. Every nonallowlisted Case or manifest byte delta halts closure pending independent evidence and a separately reviewed contract decision.
- [ ] Fresh original/current unchanged-producer default Check and Find reproduce the exact single manifest refusal with unchanged standing, Construct, ID, Owner, Why and source path, at independently derived source/lifted-Script coordinates. Manifest controls reject wrong/stale/other-site coordinates, missing/duplicate/mislabeled selectors and every other changed field; its digest is freshly derived.
- [ ] Current CaseIdentity/CaseFingerprint/prepared bindings and timeout diagnostics follow production derivation; current pins/receipt bindings are refreshed under existing selection contracts, historical Case+Run pairs stay exact, and crossed bindings reject.
- [ ] No affected current IR/Case declaration consumes changed timeout Detail semantically. Identity-only INCONCLUSIVE controls grant no successful matching replay, live or conformance credit; existing actual live/replay obligations remain required.
- [ ] Independent review of the diff and mapping is recorded
- [ ] MILESTONES.md is updated and the mapping is linked for the downstream specs
## Acceptance
- [ ] TBD

## Done summary
fn-155.6 lane `w-fn1556` (2026-10-11): the complete R7 equivalence proof now holds; the full gates are deferred to Batch 5b.

The joined candidate is `551bf89c68`. Its Model tree equals the integrated fn-155.5 state, and between it and the sealed fn-151 baseline `999fb4fdc4` only the Activity Model sources differ. Lifter, lowering, interpreter and protos are the original ones, so neither the positions nor the Duration encoding needs normalizing. It was regenerated in scratch with the gate's own build steps (`--server=false`, 70 s, 1.3 GB peak). The production `umpire-gen-cases` ran under `GOMEMLIMIT` (101 s, 5.0 GB peak); without that limit, a global OOM killed it at 7.7 GB.

- IR: the 4 Nexus IR files are byte-identical and the 4 lint acceptance sets are unchanged. Under Task2's 4-entry effect-merge map, the projection leaves 45 raw differences, all in 7 function bodies. The structural review accounts for each one: Product `reset`/`unpause` are equal per phase across 9 phases x 3 files, and `resetSettles` is equal across 13 phases. All 25 mutation controls are rejected.
- Step tables: 46 tables and 13,891,948 records are equal, covering 36 machine tables, 8 composition tables and both Pins tables. The Pins test bytes are unchanged. The comparison applies one new normalization: an empty `interp.Value` Fields/Items slice counts the same as a nil one. Rows produced by the Recorded setter differ only that way, and `Value.Equal`/`Key` read only `len()`. Three controls show the normalization still rejects real changes.
- Cases: all 31 files are present. 22 are byte-equal. The other 8 Cases and the manifest differ only in the 11 waitHint lines and the one manifest refusal Position (`Realization.scala:310` -> `:278`). Both coordinates of every leaf are rederived from the lifted ServerStep or Script and from the Scala text. Restoring those leaves recovers every original byte. All 21 mutation controls are rejected.
- Unchanged producer: the same binary run on the original IR reproduces all 31 original files byte for byte.
- No-Detail-consumer scan: 0 consumers. The 20 Activity Cases are byte-identical to Task5's fresh Cases, so Task5's production-derived identities, crossed-binding rejections and timeout Detail values carry over as they are.
- The mapping and the downstream re-anchor are in `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/w-fn1556/.flow/tmp/fn1556/proof/mapping.md`.

Gates are deferred to Batch 5b on the root's instruction. The partial Go suite (`logs/gosuite.jsonl`) shows no failure from a renamed effect or a changed shape. `export` had 207 passes before it timed out, `check` timed out, and a global OOM killed `conformance`. `lint` has two pins to refresh at the regeneration: api_test expects Realization.scala:408 where the joined lift gives :375, and the coverage summary pin predates fn-151. lower, testpilot and canary were not reached. The model gate and lint-model were not run.

### Earlier deferral (retained)
Blocked:
# Deferred fn-155 final validation

Deferred under MILESTONES.md's one-hour stuck-validation rule. Fn-155 remains open, and its full R7 and task .6 acceptance remain unchanged. Fn-156's committed-closure prerequisite remains unsatisfied.

Command `make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks` failed in attempts 1 through 3 after a total 1,480.0276258 executed seconds. Attempt 4 has a retained terminal class-loading failure log and a 900-second fixture stage, but its wrapper exit/wall receipt is missing. The observed executed-time lower bound is 2,380.0276258 seconds; the unresolved validation wall span from 2026-10-10 02:27:21 UTC to 06:13:53 UTC is 3 hours, 46 minutes and 32 seconds, including pauses and unobserved time.

Evidence lives at `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/agent/fn155-task6/.flow/tmp/fn-155/task6/`. Preserve generation1 through generation4 logs/receipts, `generation4-observation.json`, `qualifiednames-diagnostic.json`, `scratch-generation2.json`, `scoped-partitions.json` and the worker's mapping/handover. The unchanged focused QualifiedNames suite passed four tests in 103.0023378 seconds. The full scratch lower.GenerateCases attempt was killed after 59.4813334 seconds and wrote no Cases. Seventeen scoped declaration partitions matched their joined full IR in 227.547 seconds; this establishes neither complete native table equality nor whole Case/manifest freshness.

Defer unchanged canonical retries, the dependent model/managed Case gates and full Go validation. All 46 complete table comparisons, all 31 original raw files, the whole fresh manifest, the exact twelve allowed source-coordinate leaves, production identities/bindings/diagnostics, historical companion retention, independent controls and required full gates remain follow-up obligations. No passing gate, live execution, matching replay or fn-155 closure credit is granted by this deferral.

Revisit after a separately scoped tooling/resource diagnosis verifies a remedy on the unchanged full domains, default memory policy and normal command overlap, or runner provisioning supplies those conditions. Then run normal generation once and establish the complete joined proofs and gates. Keep all RED and inconclusive receipts. Independently verifiable work remains available, so the exception permitting continued retries beyond one hour does not apply.

Blocked:
# Deferred fn-155 final validation

Deferred under MILESTONES.md's one-hour stuck-validation rule. Fn-155 remains open, and its full R7 and task .6 acceptance remain unchanged. Fn-156's committed-closure prerequisite remains unsatisfied.

Command `make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks` failed in attempts 1 through 3 after a total 1,480.0276258 executed seconds. Attempt 4 has a retained terminal class-loading failure log and a 900-second fixture stage, but its wrapper exit/wall receipt is missing. The observed executed-time lower bound is 2,380.0276258 seconds; the unresolved validation wall span from 2026-10-10 02:27:21 UTC to 06:13:53 UTC is 3 hours, 46 minutes and 32 seconds, including pauses and unobserved time.

Evidence lives at `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/agent/fn155-task6/.flow/tmp/fn-155/task6/`. Preserve generation1 through generation4 logs/receipts, `generation4-observation.json`, `qualifiednames-diagnostic.json`, `scratch-generation2.json`, `scoped-partitions.json` and the worker's mapping/handover. The unchanged focused QualifiedNames suite passed four tests in 103.0023378 seconds. The full scratch lower.GenerateCases attempt was killed after 59.4813334 seconds and wrote no Cases. Seventeen scoped declaration partitions matched their joined full IR in 227.547 seconds; this establishes neither complete native table equality nor whole Case/manifest freshness.

Defer unchanged canonical retries, the dependent model/managed Case gates and full Go validation. All 46 complete table comparisons, all 31 original raw files, the whole fresh manifest, the exact twelve allowed source-coordinate leaves, production identities/bindings/diagnostics, historical companion retention, independent controls and required full gates remain follow-up obligations. No passing gate, live execution, matching replay or fn-155 closure credit is granted by this deferral.

Revisit after a separately scoped tooling/resource diagnosis verifies a remedy on the unchanged full domains, default memory policy and normal command overlap, or runner provisioning supplies those conditions. Then run normal generation once and establish the complete joined proofs and gates. Keep all RED and inconclusive receipts. Independently verifiable work remains available, so the exception permitting continued retries beyond one hour does not apply.

The later read-only diagnosis confirmed both Go kills as host-wide OOM. Kernel records attribute PID 14872 (`scratch-generat`) at 2026-10-10 06:09:46 UTC with 5,806,848 KiB anonymous RSS and PID 21666 (`lower.test`) at 06:17:38 UTC with 6,487,808 KiB anonymous RSS to `CONSTRAINT_NONE,global_oom`. Root retained the excerpts in `kernel-oom-attribution-20261010.log` alongside the original task6 receipts, without rewriting them. The effective container and parent cgroups have no memory maximum; the host reports 16,593,792 KiB RAM and zero swap. Increasing a container limit would not address this measured boundary. These victim RSS values are lower bounds, not measured successful peak capacity or allocation-owner attribution.

Partial checkpoint `2250d1f9ae4ca7c5b64a5641901511978d25819e` remains recoverable on `agent/fn155-task6`. It carries updated hint fixtures, the README's existing reset reason and a Construct/Copy test setup adapter. The task-unique `resumed-handover-summary.md` and `resumed-handover-evidence.json` record bounded setup-only greens and all unmet acceptance. This checkpoint has not been integrated or implementation-reviewed because producer/full validation remains RED or inconclusive. It supplies no closure credit.
## Evidence
- Commits: 2f4242abb9 (withholding fixture reads a Copy-based expected state; README cites states.pendingResetReason)
- Tests: /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/w-fn1556/.flow/tmp/fn1556/proof/{projection,structural-review,table-comparison,twelve-leaf,nil-empty-controls,detail-consumer-scan}.json; logs/{lift551,native,pins-dump,gen-cases-2,gen-cases-original-2,table-compare-2,gosuite}.json[l]
- PRs:
