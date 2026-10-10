---
satisfies: [R7]
---
# fn-155-name-the-standalone-activitys-repeated.6 Regenerate, prove the mapping and close fn-155

## Description
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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
