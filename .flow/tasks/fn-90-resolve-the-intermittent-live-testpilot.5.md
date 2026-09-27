---
satisfies: [R5, R7]
---
# fn-90-resolve-the-intermittent-live-testpilot.5 Resolve or close the Nexus pair ordering failure

## Description
Resolve (2), the pair evidence-ordering failure, or close it as not reproduced (R5; R7 only if the
cause is external). Conditional on fn-90.3's receipt. Either way, confirm R5's test-side clause:
every live assertion over the pair Case correlates by key.

**Size:** S if not reproduced; M if a Model or Producer fix is needed
**Files:** always: none, or `tests/testpilot_nexus_pair_case_test.go` if the audit finds a positional assertion. On a Model fix: `model/Temporal/Feature/Nexus/Pair/Model.lean`, `model/Temporal/Feature/NexusTests.lean` (where the pair fixture is produced), the Nexus realization, the regenerated `tests/testcore/testpilot/testdata/nexusPairTests-bothComplete-case.json` (and any other fixture the generator rewrites), `tests/testpilot_nexus_control_case_test.go` (pair cross-reference control)
**Touches:** [tests/testpilot_nexus_pair_case_test.go, tests/testpilot_nexus_control_case_test.go, tests/testcore/testpilot/quarantine.go, tests/testcore/testpilot/quarantine_test.go, model/Temporal/Feature/Nexus/Pair/**, model/Temporal/Feature/NexusTests.lean, model/Temporal/Case/Realization/Nexus.lean, tests/testcore/testpilot/testdata/**, common/testing/testpilot/testdata/case-runtime-conformance/**]

### Approach
- Audit first (always): every live assertion that reads pair-Case evidence (`requireCorrelatedNexusPairEvidence` at `tests/testpilot_nexus_pair_case_test.go:78` and any other reader, `grep -n NexusPair tests/*.go`) indexes by rule and scheduled event id, never by global position. Fix any positional read found.
- Not reproduced in fn-90.3: close with that evidence (retired identity, successor, count) plus the audit result. No quarantine.
- Reproduced with an ordering signature: read the signature. If a rule is unresolved because the Scenario or Contract fixes one completion order, correct the pair Model (or the Producer that derives it) so every order the server may record is admitted, regenerate with `make umpire-gen-case-runtime-conformance`, and check with `make umpire-check-case-runtime-conformance`. Then add a live control that forges a completion referencing the other instance's scheduled event and expects VIOLATED, following `TestTestpilotNexusControlForgedCompletionIsViolated` at `tests/testpilot_nexus_control_case_test.go:30`.
- Before any Lean edit: `flowctl show fn-88-veil-concrete-checker-as-the-umpire`. If fn-88 is open, do not land the Model change uncoordinated; block this task (`flowctl block`) naming fn-88 and the change, so that session re-pins its Pair counts (spec Edge Cases, fn-88 overlap). The same applies to fn-89, which rewrites this fixture's Contract after fn-90.
- Cause outside the repository (server or SDK race): open an upstream issue and implement the R7 quarantine for exactly that signature in `tests/testcore/testpilot/quarantine.go` (entry table plus the retry decision, comparing signatures with fn-90.2's field-by-field compare) and `quarantine_test.go` (offline, proving the R7 error cases); the live test calls it.
- Rerun `^TestTestpilotNexusPairCase$` for 200 process-mode iterations after any change.

### Investigation targets
**Required:**
- fn-90.3 receipt (signatures for (2))
- `tests/testpilot_nexus_pair_case_test.go`
- `model/Temporal/Feature/Nexus/Pair/Model.lean`, `model/Temporal/Feature/Nexus/Pair/Tests.lean` (read only; fn-88 edits Tests.lean)
- `Makefile:617-630` (fixture generation and check)

**Optional:**
- `tests/testpilot_nexus_control_case_test.go:30-115`

### Key context
- Do not edit `tools/umpire/cmd/umpire-gen-case-runtime-conformance/**` or `model/Temporal/API/**` (other sessions); running the generator is fine.
- QLF-05: never widen a window or drop an obligation to make the rule satisfy.
## Acceptance
- [ ] Audit result recorded; no positional pair-evidence assertion remains.
- [ ] Either "not reproduced on the successor" with fn-90.3's evidence, or a fix with regenerated fixtures, a passing pair cross-reference VIOLATED control, and 200 zero-failure iterations of the pair test.
- [ ] Any quarantine matches the spec's contract and its offline unit test passes; otherwise the receipt says "no quarantine needed".
- [ ] `make lint-code-fast` clean; `make umpire-check-case-runtime-conformance` clean if fixtures changed.
## Done summary
Closed failure (2), the evidence-ordering mismatch, as not reproduced on the successor. The static audit found no positional read of pair-Case evidence, so there is no source change and no quarantine is needed.

Evidence from fn-90.3 (commits 71559d5f45, 376943907d, receipt ed55fb0646): the retired identity TestTestpilotTypedNexusOperationsCase became TestTestpilotNexusPairCase (fixture nexusPairTests-bothComplete). It ran 0/200 in process mode at b9bb1a58ad, 95% CI 0.00%-1.83% (Clopper-Pearson). That is 400 Runs, all RUN_DISPOSITION_COMPLETED with VERDICT_STATUS_SATISFIED. Only the default Nexus switch value ran. A 5-iteration confirmation at ed55fb0646 also passed: `make umpire-repeat-run SELECT='^TestTestpilotNexusPairCase$' COUNT=5 MODE=process` gave 0/5.

Audit (R5 test-side clause). Every live read of pair-Case evidence correlates by rule id and scheduled event id. None depends on the order in which the two completions reach history:
- The live test picks each capture rule by id, not by where its evidence landed. `verdict.GetRules()[index]` at tests/testpilot_nexus_pair_case_test.go:54 follows the Contract's declaration order. The evaluator builds that list with one slot per prepared rule, correlated clauses after (common/testing/testpilot/internal/verification/evaluator.go:86-95), so the order does not change at runtime. Line 56 then asserts the rule id.
- Each rule's supporting sequences hold only that rule's own transitions (evaluator.go:189-192). Rule relation-N reaches `satisfied` only after capturing its own scheduled event, which it selects by operation name `complete-N` (fixture lines 753-875 and 887-1009). So `events[0]` scheduled and `events[1]` completed at tests/testpilot_nexus_pair_case_test.go:93-96 follow the rule's state order, not the order of history. Lines 102-103 check that the completion references that scheduled event's id and request id.
- A completion that references the other instance's scheduled event matches no transition, so the rule stays pending. That makes both completion orders admissible (the "crossed-completion-is-inconclusive" Known Gap in model/Temporal/Feature/Nexus/Pair/Model.lean:150-153).
- The correlated clauses key each operation by the scheduled event id: `evidence.scheduled` by `event_id`, started and completed by `scheduled_event_id` (fixture lines 713, 729, 745; operationField at 1028). Their clock is CORRELATED_CLOCK_OPERATION_TRANSITIONS with bound 2 (fixture lines 1314 and 1356). The monitor keeps state per operation key (common/testing/testpilot/internal/verification/correlated.go:437-440).
- The Scenario `twoAsync` fixes one action order (Model.lean:126-132), but the Contract does not encode it. The fixture carries it only as a provenance definition (fixture line 16), so the Model does not constrain the completion order the server records.
- `grep -n NexusPair tests/*.go` finds no other live reader. tests/testpilot_nexus_control_case_test.go does not read the pair Case. The offline tests/testcore/testpilot/nexus_pair_artifact_test.go:32-40 indexes the fixture's static Contract, not a Run.

No Model, Producer, fixture or realization changed, so the fn-88 coordination clause does not apply and no VIOLATED control is required. No quarantine needed.

Baseline: green, with CC=/usr/bin/clang. The shell's PATH resolves `clang` to the Lean 4.33.1 toolchain's clang, which has no libc headers, so cgo builds fail with `'stddef.h' file not found` until CC is set. That is a local environment issue, not a repository one.

GATE_SKIPPED:unittest:docs-only - task commits touch only the task .md receipt

stage: impl-review - ran [claude]

## Evidence
- Commits:
- Tests:
- PRs:
